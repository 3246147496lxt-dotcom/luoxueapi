package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// SettingKeyAccountHealthSettings stores the account-pool copilot policy.
const SettingKeyAccountHealthSettings = "account_health.settings"

type AccountHealthRoutingRule struct {
	ProxyID  *int64  `json:"proxy_id,omitempty"`
	GroupIDs []int64 `json:"group_ids,omitempty"`
}

type AccountHealthNotificationSettings struct {
	InApp      bool   `json:"in_app"`
	Email      bool   `json:"email"`
	Webhook    bool   `json:"webhook"`
	WebhookURL string `json:"webhook_url,omitempty"`
}

// AccountHealthRegistrationSettings configures discovery of SSO exports
// produced by the local registration tool. LastSeen is an RFC3339 timestamp
// and is only advanced after the caller confirms an import.
type AccountHealthRegistrationSettings struct {
	Enabled   bool   `json:"enabled"`
	OutputDir string `json:"output_dir,omitempty"`
	LastSeen  string `json:"last_seen,omitempty"`
}

// AccountHealthSettings is deliberately compact so the normal workflow needs
// no configuration. Routing keys are platform names (for example "grok").
type AccountHealthSettings struct {
	Enabled                   bool                                `json:"enabled"`
	ScanIntervalMinutes       int                                 `json:"scan_interval_minutes"`
	AutoQuarantine            bool                                `json:"auto_quarantine"`
	RequireDeleteConfirmation bool                                `json:"require_delete_confirmation"`
	RoutingRules              map[string]AccountHealthRoutingRule `json:"routing_rules,omitempty"`
	Notifications             AccountHealthNotificationSettings   `json:"notifications"`
	Registration              AccountHealthRegistrationSettings   `json:"registration"`
}

func DefaultAccountHealthSettings() *AccountHealthSettings {
	return &AccountHealthSettings{
		Enabled: true, ScanIntervalMinutes: 1440, AutoQuarantine: true,
		RequireDeleteConfirmation: true,
		RoutingRules:              map[string]AccountHealthRoutingRule{},
		Notifications:             AccountHealthNotificationSettings{InApp: true},
		Registration:              AccountHealthRegistrationSettings{},
	}
}

func normalizeAccountHealthSettings(in *AccountHealthSettings) (*AccountHealthSettings, error) {
	if in == nil {
		return nil, fmt.Errorf("settings are required")
	}
	out := *in
	if out.ScanIntervalMinutes <= 0 {
		out.ScanIntervalMinutes = 1440
	}
	if out.ScanIntervalMinutes < 5 || out.ScanIntervalMinutes > 10080 {
		return nil, fmt.Errorf("scan_interval_minutes must be between 5 and 10080")
	}
	if out.RoutingRules == nil {
		out.RoutingRules = map[string]AccountHealthRoutingRule{}
	}
	for platform, rule := range out.RoutingRules {
		key := strings.ToLower(strings.TrimSpace(platform))
		if key == "" {
			delete(out.RoutingRules, platform)
			continue
		}
		if key != platform {
			delete(out.RoutingRules, platform)
			out.RoutingRules[key] = rule
		}
	}
	if !out.RequireDeleteConfirmation { // destructive deletion is never policy-only
		out.RequireDeleteConfirmation = true
	}
	return &out, nil
}

func (s *AccountHealthService) GetSettings(ctx context.Context) (*AccountHealthSettings, error) {
	defaults := DefaultAccountHealthSettings()
	if s == nil || s.settingsRepo == nil {
		return defaults, nil
	}
	raw, err := s.settingsRepo.GetValue(ctx, SettingKeyAccountHealthSettings)
	if err != nil || strings.TrimSpace(raw) == "" {
		return defaults, nil
	}
	var cfg AccountHealthSettings
	if json.Unmarshal([]byte(raw), &cfg) != nil {
		return defaults, nil
	}
	if cfg.ScanIntervalMinutes == 0 {
		cfg.ScanIntervalMinutes = defaults.ScanIntervalMinutes
	}
	if cfg.RoutingRules == nil {
		cfg.RoutingRules = defaults.RoutingRules
	}
	if !cfg.Notifications.InApp && !cfg.Notifications.Email && !cfg.Notifications.Webhook {
		cfg.Notifications.InApp = defaults.Notifications.InApp
	}
	return normalizeAccountHealthSettings(&cfg)
}

func (s *AccountHealthService) UpdateSettings(ctx context.Context, cfg *AccountHealthSettings) (*AccountHealthSettings, error) {
	normalized, err := normalizeAccountHealthSettings(cfg)
	if err != nil {
		return nil, err
	}
	if s == nil || s.settingsRepo == nil {
		return normalized, nil
	}
	payload, err := json.Marshal(normalized)
	if err != nil {
		return nil, err
	}
	if err := s.settingsRepo.Set(ctx, SettingKeyAccountHealthSettings, string(payload)); err != nil {
		return nil, err
	}
	return normalized, nil
}
