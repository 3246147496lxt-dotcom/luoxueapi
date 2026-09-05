package service

// Outbound URL validation shared by Chinese-provider quota probes. Probe
// endpoints are derived from account credentials, so they must use the same
// allowlist/SSRF policy as normal upstream forwarding.

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"
)

func cnValidateProbeURL(cfg *config.Config, raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("probe url is required")
	}
	if cfg != nil && cfg.Security.URLAllowlist.Enabled {
		normalized, err := urlvalidator.ValidateHTTPSURL(trimmed, urlvalidator.ValidationOptions{
			AllowedHosts:     cfg.Security.URLAllowlist.UpstreamHosts,
			RequireAllowlist: true,
			AllowPrivate:     cfg.Security.URLAllowlist.AllowPrivateHosts,
		})
		if err != nil {
			return "", fmt.Errorf("probe target rejected by URL security policy: %w", err)
		}
		return normalized, nil
	}
	allowHTTP := cfg != nil && cfg.Security.URLAllowlist.AllowInsecureHTTP
	normalized, err := urlvalidator.ValidateURLFormat(trimmed, allowHTTP)
	if err != nil {
		return "", fmt.Errorf("probe target rejected by URL security policy: %w", err)
	}
	return normalized, nil
}
