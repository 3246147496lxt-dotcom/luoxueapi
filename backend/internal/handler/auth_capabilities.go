package handler

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// LoginUseCases is the authentication/session surface consumed by the HTTP
// layer. It deliberately excludes registration and persistence access.
type LoginUseCases interface {
	VerifyTurnstile(ctx context.Context, token, remoteIP string) error
	Login(ctx context.Context, email, password string) (string, *service.User, error)
	ValidatePasswordCredentials(ctx context.Context, email, password string) (*service.User, error)
	GenerateToken(ctx context.Context, user *service.User) (string, error)
	GenerateTokenPair(ctx context.Context, user *service.User, familyID string) (*service.TokenPair, error)
	RefreshTokenPair(ctx context.Context, refreshToken string) (*service.TokenPairWithUser, error)
	RevokeRefreshToken(ctx context.Context, refreshToken string) error
	RevokeAllUserTokens(ctx context.Context, userID int64) error
	RecordSuccessfulLogin(ctx context.Context, userID int64)
	RequestPasswordResetAsync(ctx context.Context, email, frontendBaseURL string, locale ...string) error
	ResetPassword(ctx context.Context, email, token, newPassword string) error
}

// SignupUseCases is the registration surface consumed by the HTTP layer.
type SignupUseCases interface {
	VerifyTurnstileForRegister(ctx context.Context, token, remoteIP, verifyCode string) error
	RegisterWithVerification(ctx context.Context, email, password, verifyCode, promoCode, invitationCode, affiliateCode string) (string, *service.User, error)
	SendVerifyCodeAsync(ctx context.Context, email string, locale ...string) (*service.SendVerifyCodeResult, error)
	SendPendingOAuthVerifyCode(ctx context.Context, email string, locale ...string) (*service.SendVerifyCodeResult, error)
	IsEmailVerifyEnabled(ctx context.Context) bool
}

// PendingIdentityUseCases is the browser-bound pending identity capability.
// It exposes transport-neutral service models; query construction and ORM
// ownership stay behind the service boundary.
type PendingIdentityUseCases interface {
	CreatePendingSession(ctx context.Context, input service.CreatePendingAuthSessionInput) (*service.PendingAuthSession, error)
	IssueCompletionCode(ctx context.Context, input service.IssuePendingAuthCompletionCodeInput) (*service.IssuePendingAuthCompletionCodeResult, error)
	ConsumeCompletionCode(ctx context.Context, rawCode, browserSessionKey string) (*service.PendingAuthSession, error)
	ConsumeBrowserSession(ctx context.Context, sessionToken, browserSessionKey string) (*service.PendingAuthSession, error)
	GetBrowserSession(ctx context.Context, sessionToken, browserSessionKey string) (*service.PendingAuthSession, error)
	UpsertAdoptionDecision(ctx context.Context, input service.PendingIdentityAdoptionDecisionInput) (*service.PendingIdentityDecision, error)
	GetAdoptionDecision(ctx context.Context, pendingAuthSessionID int64) (*service.PendingIdentityDecision, error)
	UpdateSessionProgress(ctx context.Context, input service.UpdatePendingAuthSessionProgressInput) (*service.PendingAuthSession, error)
	FindUserByNormalizedEmail(ctx context.Context, email string) (*service.AuthIdentityUser, error)
	FindIdentityUser(ctx context.Context, identity service.PendingAuthIdentityKey) (*service.AuthIdentityUser, error)
	EnsureRegistrationIdentityAvailable(ctx context.Context, session *service.PendingAuthSession) error
	IdentityExistsForUser(ctx context.Context, session *service.PendingAuthSession, userID int64, compatibleProviderKeys []string) (bool, error)
	ApplyBinding(ctx context.Context, input service.ApplyPendingIdentityBindingInput) error
	ApplyBindingAndConsume(ctx context.Context, input service.ApplyPendingIdentityBindingInput) error
	FindUserByID(ctx context.Context, userID int64) (*service.AuthIdentityUser, error)
	EnsureWeChatIdentityOwnership(ctx context.Context, input service.WeChatIdentityOwnershipInput) error
	FindWeChatIdentityUser(ctx context.Context, input service.WeChatIdentityLookupInput) (*service.AuthIdentityUser, error)
}

var (
	_ LoginUseCases           = (*service.AuthService)(nil)
	_ SignupUseCases          = (*service.AuthService)(nil)
	_ PendingIdentityUseCases = (*service.AuthPendingIdentityService)(nil)
)
