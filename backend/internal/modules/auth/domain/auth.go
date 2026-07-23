package domain

import "time"

// Principal is the authentication module's transport-neutral user view.
type Principal struct {
	ID           int64
	Email        string
	Role         string
	Status       string
	TOTPEnabled  bool
	TokenVersion int64
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
	UserRole     string
}

type LoginChallengeCommand struct {
	TurnstileToken string
	RemoteIP       string
}

type LoginCommand struct {
	Email          string
	Password       string
	TurnstileToken string
	RemoteIP       string
}

type LoginResult struct {
	Principal Principal
	Tokens    TokenPair
}

type SignupCommand struct {
	Email          string
	Password       string
	VerifyCode     string
	PromoCode      string
	InvitationCode string
	AffiliateCode  string
	TurnstileToken string
	RemoteIP       string
}

type SignupChallengeCommand struct {
	TurnstileToken string
	RemoteIP       string
	VerifyCode     string
}

type VerificationCommand struct {
	Email  string
	Locale string
}

type VerificationResult struct {
	Countdown int
}

type PendingIdentityKey struct {
	ProviderType    string
	ProviderKey     string
	ProviderSubject string
}

type CreatePendingIdentityCommand struct {
	Intent            string
	Identity          PendingIdentityKey
	TargetUserID      *int64
	ResolvedEmail     string
	RedirectTo        string
	BrowserSessionKey string
	UpstreamClaims    map[string]any
	FlowState         map[string]any
	ExpiresAt         time.Time
}

type PendingIdentitySession struct {
	ID                int64
	SessionToken      string
	Intent            string
	Identity          PendingIdentityKey
	TargetUserID      *int64
	ResolvedEmail     string
	RedirectTo        string
	BrowserSessionKey string
	UpstreamClaims    map[string]any
	FlowState         map[string]any
	ExpiresAt         time.Time
	ConsumedAt        *time.Time
}
