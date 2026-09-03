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

// Keep the alias-specific capabilities out of the broad UserRepository
// interface so existing adapters and test doubles remain source-compatible.
// Every security-sensitive call site type-asserts these ports and fails closed
// when they are unavailable; there is no fallback to the unguarded methods.
type emailAliasLookupRepository interface {
	ExistsByEmailAlias(ctx context.Context, email string) (bool, error)
}

type emailAliasCreateRepository interface {
	CreateWithEmailAliasGuard(ctx context.Context, user *User) error
}

type emailIdentityAliasGuardRepository interface {
	UpdateEmailWithAliasGuard(ctx context.Context, userID int64, email, passwordHash string) error
}

// existsByEmailOrAlias is used by local registration and verification-code
// flows. Exact lookup remains the fast path; the repository port requires the
// alias probe so implementations cannot silently fail open.
func (s *AuthService) existsByEmailOrAlias(ctx context.Context, email string) (bool, error) {
	if s == nil || s.userRepo == nil {
		return false, ErrServiceUnavailable
	}
	exists, err := s.userRepo.ExistsByEmail(ctx, email)
	if err != nil || exists {
		return exists, err
	}
	repo, ok := s.userRepo.(emailAliasLookupRepository)
	if !ok {
		return false, ErrServiceUnavailable
	}
	return repo.ExistsByEmailAlias(ctx, email)
}

// createUserWithEmailAliasGuard routes every registration through the
// repository's atomic alias reservation. The interface requirement prevents a
// lightweight implementation from accidentally bypassing the guard.
func (s *AuthService) createUserWithEmailAliasGuard(ctx context.Context, user *User) error {
	if s == nil || s.userRepo == nil {
		return ErrServiceUnavailable
	}
	repo, ok := s.userRepo.(emailAliasCreateRepository)
	if !ok {
		return ErrServiceUnavailable
	}
	return repo.CreateWithEmailAliasGuard(ctx, user)
}
