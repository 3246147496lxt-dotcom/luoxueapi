package ports

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/modules/auth/domain"
)

type LoginUseCases interface {
	VerifyLoginChallenge(ctx context.Context, command domain.LoginChallengeCommand) error
	Login(ctx context.Context, command domain.LoginCommand) (domain.LoginResult, error)
	Refresh(ctx context.Context, refreshToken string) (domain.TokenPair, error)
	Revoke(ctx context.Context, refreshToken string) error
	RevokeAll(ctx context.Context, userID int64) error
}

type SignupUseCases interface {
	VerifySignupChallenge(ctx context.Context, command domain.SignupChallengeCommand) error
	Signup(ctx context.Context, command domain.SignupCommand) (domain.LoginResult, error)
	SendVerification(ctx context.Context, command domain.VerificationCommand) (domain.VerificationResult, error)
}

type PendingIdentityUseCases interface {
	CreatePendingIdentity(ctx context.Context, command domain.CreatePendingIdentityCommand) (domain.PendingIdentitySession, error)
	GetPendingIdentity(ctx context.Context, sessionToken, browserSessionKey string) (domain.PendingIdentitySession, error)
	ConsumePendingIdentity(ctx context.Context, sessionToken, browserSessionKey string) (domain.PendingIdentitySession, error)
}

// AuthOperations is the cohesive capability set available inside one auth
// unit of work. Keeping it separate from AuthUnitOfWork makes the transaction
// or session boundary explicit without leaking an ORM transaction.
type AuthOperations interface {
	LoginUseCases
	SignupUseCases
	PendingIdentityUseCases
}

// AuthUnitOfWork owns the consistency boundary for authentication commands.
// The compatibility adapter delegates to existing atomic service operations;
// a module-native implementation can later coordinate persistence directly.
type AuthUnitOfWork interface {
	WithinAuthUnit(ctx context.Context, operation func(context.Context, AuthOperations) error) error
}
