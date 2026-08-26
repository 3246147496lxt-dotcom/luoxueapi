package service

import (
	"context"
	"strings"
)

// Registration-only inbox identity normalization. Stored email addresses are
// intentionally unchanged; this is used solely to prevent signup-grant abuse
// through provider aliases.
var gmailFamilyDomains = map[string]struct{}{
	"gmail.com": {}, "googlemail.com": {},
}

// NormalizeEmailForAliasDedup returns the canonical inbox identity. It strips
// a non-empty plus suffix for all domains, removes Gmail dots, folds the Gmail
// family to gmail.com, and ignores a trailing FQDN root dot.
func NormalizeEmailForAliasDedup(email string) string {
	local, domain, ok := splitEmailForAliasDedup(email)
	if !ok {
		return strings.ToLower(strings.TrimSpace(email))
	}
	local = stripEmailPlusSuffix(local)
	if isGmailFamilyDomain(domain) {
		local = stripEmailLocalDots(local)
		domain = "gmail.com"
	}
	return local + "@" + domain
}

// EmailAliasProbe describes a dot-stripped lookup probe used by the repository.
type EmailAliasProbe struct{ Local, Domain string }

func EmailAliasDedupProbes(email string) []EmailAliasProbe {
	local, domain, ok := splitEmailForAliasDedup(email)
	if !ok {
		return nil
	}
	probeLocal := strings.ReplaceAll(stripEmailPlusSuffix(local), ".", "")
	if probeLocal == "" {
		return nil
	}
	domains := []string{domain}
	if isGmailFamilyDomain(domain) {
		domains = []string{"gmail.com", "googlemail.com"}
	}
	probes := make([]EmailAliasProbe, 0, len(domains))
	for _, candidate := range domains {
		probes = append(probes, EmailAliasProbe{Local: probeLocal, Domain: strings.ReplaceAll(candidate, ".", "")})
	}
	return probes
}

func splitEmailForAliasDedup(email string) (string, string, bool) {
	local, domain, ok := splitEmailForPolicy(email)
	if !ok {
		return "", "", false
	}
	domain = strings.TrimRight(domain, ".")
	if domain == "" {
		return "", "", false
	}
	return local, domain, true
}

func stripEmailPlusSuffix(local string) string {
	if idx := strings.IndexByte(local, '+'); idx > 0 {
		return local[:idx]
	}
	return local
}

func stripEmailLocalDots(local string) string {
	if stripped := strings.ReplaceAll(local, ".", ""); stripped != "" {
		return stripped
	}
	return local
}

func isGmailFamilyDomain(domain string) bool {
	_, ok := gmailFamilyDomains[domain]
	return ok
}

// existsByEmailOrAlias is used by local registration and verification-code
// flows. Exact lookup remains the fast path; alias lookup is fail-closed.
func (s *AuthService) existsByEmailOrAlias(ctx context.Context, email string) (bool, error) {
	exists, err := s.userRepo.ExistsByEmail(ctx, email)
	if err != nil || exists {
		return exists, err
	}
	if repo, ok := s.userRepo.(interface {
		ExistsByEmailAlias(context.Context, string) (bool, error)
	}); ok {
		return repo.ExistsByEmailAlias(ctx, email)
	}
	return false, nil
}

// createUserWithEmailAliasGuard keeps legacy test doubles and third-party
// repository implementations source-compatible while using the stronger
// atomic guard whenever the concrete repository provides it.
func (s *AuthService) createUserWithEmailAliasGuard(ctx context.Context, user *User) error {
	if repo, ok := s.userRepo.(interface {
		CreateWithEmailAliasGuard(context.Context, *User) error
	}); ok {
		return repo.CreateWithEmailAliasGuard(ctx, user)
	}
	return s.userRepo.Create(ctx, user)
}
