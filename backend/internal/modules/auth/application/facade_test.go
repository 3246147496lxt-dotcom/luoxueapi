package application

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/modules/auth/domain"
	"github.com/Wei-Shaw/sub2api/internal/modules/auth/ports"
	"github.com/stretchr/testify/require"
)

type loginStub struct {
	command     domain.LoginCommand
	withinCalls int
}

func (s *loginStub) WithinAuthUnit(ctx context.Context, operation func(context.Context, ports.AuthOperations) error) error {
	s.withinCalls++
	return operation(ctx, s)
}

func (*loginStub) VerifyLoginChallenge(context.Context, domain.LoginChallengeCommand) error {
	return nil
}

func (s *loginStub) Login(_ context.Context, command domain.LoginCommand) (domain.LoginResult, error) {
	s.command = command
	return domain.LoginResult{Principal: domain.Principal{ID: 1, Email: command.Email}}, nil
}

func (*loginStub) Refresh(context.Context, string) (domain.TokenPair, error) {
	return domain.TokenPair{}, nil
}

func (*loginStub) Revoke(context.Context, string) error {
	return nil
}

func (*loginStub) RevokeAll(context.Context, int64) error {
	return nil
}

func (*loginStub) VerifySignupChallenge(context.Context, domain.SignupChallengeCommand) error {
	return nil
}

func (*loginStub) Signup(context.Context, domain.SignupCommand) (domain.LoginResult, error) {
	return domain.LoginResult{}, nil
}

func (*loginStub) SendVerification(context.Context, domain.VerificationCommand) (domain.VerificationResult, error) {
	return domain.VerificationResult{}, nil
}

func (*loginStub) CreatePendingIdentity(context.Context, domain.CreatePendingIdentityCommand) (domain.PendingIdentitySession, error) {
	return domain.PendingIdentitySession{}, nil
}

func (*loginStub) GetPendingIdentity(context.Context, string, string) (domain.PendingIdentitySession, error) {
	return domain.PendingIdentitySession{}, nil
}

func (*loginStub) ConsumePendingIdentity(context.Context, string, string) (domain.PendingIdentitySession, error) {
	return domain.PendingIdentitySession{}, nil
}

func TestFacadeNormalizesLoginEmail(t *testing.T) {
	login := &loginStub{}
	facade := NewFacade(Dependencies{UnitOfWork: login})

	result, err := facade.Login(context.Background(), domain.LoginCommand{Email: "  USER@Example.COM "})

	require.NoError(t, err)
	require.Equal(t, "user@example.com", login.command.Email)
	require.Equal(t, "user@example.com", result.Principal.Email)
	require.Equal(t, 1, login.withinCalls)
}

func TestFacadeFailsClosedWhenPortIsMissing(t *testing.T) {
	_, err := NewFacade(Dependencies{}).Refresh(context.Background(), "token")
	require.ErrorIs(t, err, ErrUseCaseUnavailable)
}

func TestFacadeRoutesTokenSignupAndPendingCommandsThroughUnitOfWork(t *testing.T) {
	unitOfWork := &loginStub{}
	facade := NewFacade(Dependencies{UnitOfWork: unitOfWork})
	ctx := context.Background()

	require.NoError(t, facade.VerifySignupChallenge(ctx, domain.SignupChallengeCommand{}))
	_, err := facade.SendVerification(ctx, domain.VerificationCommand{Email: "user@example.com"})
	require.NoError(t, err)
	_, err = facade.Refresh(ctx, "refresh-token")
	require.NoError(t, err)
	_, err = facade.ConsumePendingIdentity(ctx, "session-token", "browser-key")
	require.NoError(t, err)
	require.Equal(t, 4, unitOfWork.withinCalls)
}
