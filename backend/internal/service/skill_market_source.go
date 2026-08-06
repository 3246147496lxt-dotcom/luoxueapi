package service

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

var (
	githubOwnerPattern      = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,37}[A-Za-z0-9])?$`)
	githubRepositoryPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,99}$`)
	githubPathPartPattern   = regexp.MustCompile(`^[^[:space:]?#]+$`)
)

// normalizeSkillSourceURL accepts only canonical HTTPS github.com repository,
// tree and blob URLs. The repository identity is derived server-side so callers
// cannot pair a trusted-looking label with a different destination.
func normalizeSkillSourceURL(raw string) (normalizedURL, repository string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", nil
	}
	if len(raw) > 2048 {
		return "", "", errors.New("unsupported source URL")
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Host != "github.com" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.Fragment != "" || parsed.RawPath != "" {
		return "", "", errors.New("unsupported source URL")
	}

	pathValue := strings.TrimSuffix(parsed.Path, "/")
	parts := strings.Split(strings.TrimPrefix(pathValue, "/"), "/")
	if len(parts) < 2 || !githubOwnerPattern.MatchString(parts[0]) || !githubRepositoryPattern.MatchString(parts[1]) {
		return "", "", errors.New("unsupported source URL")
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || !githubPathPartPattern.MatchString(part) {
			return "", "", errors.New("unsupported source URL")
		}
	}

	switch {
	case len(parts) == 2:
	case len(parts) >= 4 && parts[2] == "tree":
	case len(parts) >= 5 && parts[2] == "blob":
	default:
		return "", "", errors.New("unsupported source URL")
	}

	repository = parts[0] + "/" + parts[1]
	return "https://github.com/" + strings.Join(parts, "/"), repository, nil
}
