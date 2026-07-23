package application

import (
	"context"
	"errors"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/modules/auth/domain"
	"github.com/Wei-Shaw/sub2api/internal/modules/auth/ports"
)

var ErrUseCaseUnavailable = errors.New("auth use case is unavailable")

type Dependencies struct {
	UnitOfWork ports.AuthUnitOfWork
}

// Facade is the stable module entry point while the legacy auth service remains
// the adapter behind these ports for one compatibility release.
type Facade struct {
	unitOfWork ports.AuthUnitOfWork
}

func NewFacade(deps Dependencies) *Facade {
	return &Facade{unitOfWork: deps.UnitOfWork}
}

func (f *Facade) withinAuthUnit(ctx context.Context, operation func(context.Context, ports.AuthOperations) error) error {
	if f == nil || f.unitOfWork == nil {
		return ErrUseCaseUnavailable
	}
	return f.unitOfWork.WithinAuthUnit(ctx, operation)
}

func (f *Facade) VerifyLoginChallenge(ctx context.Context, command domain.LoginChallengeCommand) error {
	return f.withinAuthUnit(ctx, func(unitCtx context.Context, operations ports.AuthOperations) error {
		return operations.VerifyLoginChallenge(unitCtx, command)
	})
}

func (f *Facade) Login(ctx context.Context, command domain.LoginCommand) (domain.LoginResult, error) {
	command.Email = strings.TrimSpace(strings.ToLower(command.Email))
	var result domain.LoginResult
	err := f.withinAuthUnit(ctx, func(unitCtx context.Context, operations ports.AuthOperations) error {
		var operationErr error
		result, operationErr = operations.Login(unitCtx, command)
		return operationErr
	})
	return result, err
}

func (f *Facade) Refresh(ctx context.Context, refreshToken string) (domain.TokenPair, error) {
	var result domain.TokenPair
	err := f.withinAuthUnit(ctx, func(unitCtx context.Context, operations ports.AuthOperations) error {
		var operationErr error
		result, operationErr = operations.Refresh(unitCtx, refreshToken)
		return operationErr
	})
	return result, err
}

func (f *Facade) Revoke(ctx context.Context, refreshToken string) error {
	return f.withinAuthUnit(ctx, func(unitCtx context.Context, operations ports.AuthOperations) error {
		return operations.Revoke(unitCtx, refreshToken)
	})
}

func (f *Facade) RevokeAll(ctx context.Context, userID int64) error {
	return f.withinAuthUnit(ctx, func(unitCtx context.Context, operations ports.AuthOperations) error {
		return operations.RevokeAll(unitCtx, userID)
	})
}

func (f *Facade) VerifySignupChallenge(ctx context.Context, command domain.SignupChallengeCommand) error {
	return f.withinAuthUnit(ctx, func(unitCtx context.Context, operations ports.AuthOperations) error {
		return operations.VerifySignupChallenge(unitCtx, command)
	})
}

func (f *Facade) Signup(ctx context.Context, command domain.SignupCommand) (domain.LoginResult, error) {
	command.Email = strings.TrimSpace(strings.ToLower(command.Email))
	command.PromoCode = strings.TrimSpace(command.PromoCode)
	command.InvitationCode = strings.TrimSpace(command.InvitationCode)
	command.AffiliateCode = strings.TrimSpace(command.AffiliateCode)
	var result domain.LoginResult
	err := f.withinAuthUnit(ctx, func(unitCtx context.Context, operations ports.AuthOperations) error {
		var operationErr error
		result, operationErr = operations.Signup(unitCtx, command)
		return operationErr
	})
	return result, err
}

func (f *Facade) SendVerification(ctx context.Context, command domain.VerificationCommand) (domain.VerificationResult, error) {
	var result domain.VerificationResult
	err := f.withinAuthUnit(ctx, func(unitCtx context.Context, operations ports.AuthOperations) error {
		var operationErr error
		result, operationErr = operations.SendVerification(unitCtx, command)
		return operationErr
	})
	return result, err
}

func (f *Facade) CreatePendingIdentity(ctx context.Context, command domain.CreatePendingIdentityCommand) (domain.PendingIdentitySession, error) {
	command.Intent = strings.TrimSpace(command.Intent)
	command.Identity.ProviderType = strings.TrimSpace(command.Identity.ProviderType)
	command.Identity.ProviderKey = strings.TrimSpace(command.Identity.ProviderKey)
	command.Identity.ProviderSubject = strings.TrimSpace(command.Identity.ProviderSubject)
	command.ResolvedEmail = strings.TrimSpace(strings.ToLower(command.ResolvedEmail))
	command.RedirectTo = strings.TrimSpace(command.RedirectTo)
	command.BrowserSessionKey = strings.TrimSpace(command.BrowserSessionKey)
	command.UpstreamClaims = cloneMap(command.UpstreamClaims)
	command.FlowState = cloneMap(command.FlowState)
	var result domain.PendingIdentitySession
	err := f.withinAuthUnit(ctx, func(unitCtx context.Context, operations ports.AuthOperations) error {
		var operationErr error
		result, operationErr = operations.CreatePendingIdentity(unitCtx, command)
		return operationErr
	})
	return result, err
}

func (f *Facade) GetPendingIdentity(ctx context.Context, sessionToken, browserSessionKey string) (domain.PendingIdentitySession, error) {
	var result domain.PendingIdentitySession
	err := f.withinAuthUnit(ctx, func(unitCtx context.Context, operations ports.AuthOperations) error {
		var operationErr error
		result, operationErr = operations.GetPendingIdentity(unitCtx, sessionToken, browserSessionKey)
		return operationErr
	})
	return result, err
}

func (f *Facade) ConsumePendingIdentity(ctx context.Context, sessionToken, browserSessionKey string) (domain.PendingIdentitySession, error) {
	var result domain.PendingIdentitySession
	err := f.withinAuthUnit(ctx, func(unitCtx context.Context, operations ports.AuthOperations) error {
		var operationErr error
		result, operationErr = operations.ConsumePendingIdentity(unitCtx, sessionToken, browserSessionKey)
		return operationErr
	})
	return result, err
}

func cloneMap(values map[string]any) map[string]any {
	if values == nil {
		return nil
	}
	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}
