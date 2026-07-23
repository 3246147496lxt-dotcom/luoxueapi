package service

import (
	"context"

	authapplication "github.com/Wei-Shaw/sub2api/internal/modules/auth/application"
	authdomain "github.com/Wei-Shaw/sub2api/internal/modules/auth/domain"
	authports "github.com/Wei-Shaw/sub2api/internal/modules/auth/ports"
)

// AuthModuleAdapter keeps the legacy implementation behind the new module
// ports for one compatibility release. It can be removed after AuthUnitOfWork
// owns login, signup and pending identity orchestration directly.
type AuthModuleAdapter struct {
	auth    *AuthService
	pending *AuthPendingIdentityService
}

func NewAuthModuleAdapter(auth *AuthService) *AuthModuleAdapter {
	var pending *AuthPendingIdentityService
	if auth != nil {
		pending = auth.PendingIdentityUseCases()
	}
	return &AuthModuleAdapter{auth: auth, pending: pending}
}

func NewAuthModuleFacade(auth *AuthService) *authapplication.Facade {
	adapter := NewAuthModuleAdapter(auth)
	return authapplication.NewFacade(authapplication.Dependencies{
		UnitOfWork: adapter,
	})
}

// WithinAuthUnit is the compatibility implementation of the module unit of
// work. Legacy auth operations already own their DB/Redis atomicity, so this
// adapter must not start an unsafe cross-resource transaction around them.
func (a *AuthModuleAdapter) WithinAuthUnit(ctx context.Context, operation func(context.Context, authports.AuthOperations) error) error {
	if a == nil || a.auth == nil {
		return ErrServiceUnavailable
	}
	if operation == nil {
		return nil
	}
	return operation(ctx, a)
}

func (a *AuthModuleAdapter) VerifyLoginChallenge(ctx context.Context, command authdomain.LoginChallengeCommand) error {
	if a == nil || a.auth == nil {
		return ErrServiceUnavailable
	}
	return a.auth.VerifyTurnstile(ctx, command.TurnstileToken, command.RemoteIP)
}

func (a *AuthModuleAdapter) Login(ctx context.Context, command authdomain.LoginCommand) (authdomain.LoginResult, error) {
	if a == nil || a.auth == nil {
		return authdomain.LoginResult{}, ErrServiceUnavailable
	}
	if err := a.VerifyLoginChallenge(ctx, authdomain.LoginChallengeCommand{
		TurnstileToken: command.TurnstileToken,
		RemoteIP:       command.RemoteIP,
	}); err != nil {
		return authdomain.LoginResult{}, err
	}
	_, user, err := a.auth.Login(ctx, command.Email, command.Password)
	if err != nil {
		return authdomain.LoginResult{}, err
	}
	tokens, err := a.auth.GenerateTokenPair(ctx, user, "")
	if err != nil {
		return authdomain.LoginResult{}, err
	}
	return authLoginResult(user, tokens), nil
}

func (a *AuthModuleAdapter) Refresh(ctx context.Context, refreshToken string) (authdomain.TokenPair, error) {
	if a == nil || a.auth == nil {
		return authdomain.TokenPair{}, ErrServiceUnavailable
	}
	tokens, err := a.auth.RefreshTokenPair(ctx, refreshToken)
	if err != nil {
		return authdomain.TokenPair{}, err
	}
	result := authTokenPair(tokens.TokenPair)
	result.UserRole = tokens.UserRole
	return result, nil
}

func (a *AuthModuleAdapter) Revoke(ctx context.Context, refreshToken string) error {
	if a == nil || a.auth == nil {
		return ErrServiceUnavailable
	}
	return a.auth.RevokeRefreshToken(ctx, refreshToken)
}

func (a *AuthModuleAdapter) RevokeAll(ctx context.Context, userID int64) error {
	if a == nil || a.auth == nil {
		return ErrServiceUnavailable
	}
	return a.auth.RevokeAllUserTokens(ctx, userID)
}

func (a *AuthModuleAdapter) VerifySignupChallenge(ctx context.Context, command authdomain.SignupChallengeCommand) error {
	if a == nil || a.auth == nil {
		return ErrServiceUnavailable
	}
	return a.auth.VerifyTurnstileForRegister(ctx, command.TurnstileToken, command.RemoteIP, command.VerifyCode)
}

func (a *AuthModuleAdapter) Signup(ctx context.Context, command authdomain.SignupCommand) (authdomain.LoginResult, error) {
	if a == nil || a.auth == nil {
		return authdomain.LoginResult{}, ErrServiceUnavailable
	}
	if err := a.VerifySignupChallenge(ctx, authdomain.SignupChallengeCommand{
		TurnstileToken: command.TurnstileToken,
		RemoteIP:       command.RemoteIP,
		VerifyCode:     command.VerifyCode,
	}); err != nil {
		return authdomain.LoginResult{}, err
	}
	_, user, err := a.auth.RegisterWithVerification(
		ctx,
		command.Email,
		command.Password,
		command.VerifyCode,
		command.PromoCode,
		command.InvitationCode,
		command.AffiliateCode,
	)
	if err != nil {
		return authdomain.LoginResult{}, err
	}
	tokens, err := a.auth.GenerateTokenPair(ctx, user, "")
	if err != nil {
		return authdomain.LoginResult{}, err
	}
	return authLoginResult(user, tokens), nil
}

func (a *AuthModuleAdapter) SendVerification(ctx context.Context, command authdomain.VerificationCommand) (authdomain.VerificationResult, error) {
	if a == nil || a.auth == nil {
		return authdomain.VerificationResult{}, ErrServiceUnavailable
	}
	result, err := a.auth.SendVerifyCodeAsync(ctx, command.Email, command.Locale)
	if err != nil {
		return authdomain.VerificationResult{}, err
	}
	return authdomain.VerificationResult{Countdown: result.Countdown}, nil
}

func (a *AuthModuleAdapter) CreatePendingIdentity(ctx context.Context, command authdomain.CreatePendingIdentityCommand) (authdomain.PendingIdentitySession, error) {
	if a == nil || a.pending == nil {
		return authdomain.PendingIdentitySession{}, ErrServiceUnavailable
	}
	session, err := a.pending.CreatePendingSession(ctx, CreatePendingAuthSessionInput{
		Intent: command.Intent,
		Identity: PendingAuthIdentityKey{
			ProviderType:    command.Identity.ProviderType,
			ProviderKey:     command.Identity.ProviderKey,
			ProviderSubject: command.Identity.ProviderSubject,
		},
		TargetUserID:           command.TargetUserID,
		ResolvedEmail:          command.ResolvedEmail,
		RedirectTo:             command.RedirectTo,
		BrowserSessionKey:      command.BrowserSessionKey,
		UpstreamIdentityClaims: cloneAuthModuleMap(command.UpstreamClaims),
		LocalFlowState:         cloneAuthModuleMap(command.FlowState),
		ExpiresAt:              command.ExpiresAt,
	})
	return authModulePendingSession(session), err
}

func (a *AuthModuleAdapter) GetPendingIdentity(ctx context.Context, sessionToken, browserSessionKey string) (authdomain.PendingIdentitySession, error) {
	if a == nil || a.pending == nil {
		return authdomain.PendingIdentitySession{}, ErrServiceUnavailable
	}
	session, err := a.pending.GetBrowserSession(ctx, sessionToken, browserSessionKey)
	return authModulePendingSession(session), err
}

func (a *AuthModuleAdapter) ConsumePendingIdentity(ctx context.Context, sessionToken, browserSessionKey string) (authdomain.PendingIdentitySession, error) {
	if a == nil || a.pending == nil {
		return authdomain.PendingIdentitySession{}, ErrServiceUnavailable
	}
	session, err := a.pending.ConsumeBrowserSession(ctx, sessionToken, browserSessionKey)
	return authModulePendingSession(session), err
}

func authLoginResult(user *User, tokens *TokenPair) authdomain.LoginResult {
	if user == nil {
		return authdomain.LoginResult{}
	}
	tokenPair := authTokenPairValue(tokens)
	tokenPair.UserRole = user.Role
	return authdomain.LoginResult{
		Principal: authdomain.Principal{
			ID:           user.ID,
			Email:        user.Email,
			Role:         user.Role,
			Status:       user.Status,
			TOTPEnabled:  user.TotpEnabled,
			TokenVersion: user.TokenVersion,
		},
		Tokens: tokenPair,
	}
}

func authTokenPair(tokens TokenPair) authdomain.TokenPair {
	return authdomain.TokenPair{AccessToken: tokens.AccessToken, RefreshToken: tokens.RefreshToken, ExpiresIn: tokens.ExpiresIn}
}

func authTokenPairValue(tokens *TokenPair) authdomain.TokenPair {
	if tokens == nil {
		return authdomain.TokenPair{}
	}
	return authTokenPair(*tokens)
}

func authModulePendingSession(session *PendingAuthSession) authdomain.PendingIdentitySession {
	if session == nil {
		return authdomain.PendingIdentitySession{}
	}
	return authdomain.PendingIdentitySession{
		ID:                session.ID,
		SessionToken:      session.SessionToken,
		Intent:            session.Intent,
		Identity:          authdomain.PendingIdentityKey{ProviderType: session.ProviderType, ProviderKey: session.ProviderKey, ProviderSubject: session.ProviderSubject},
		TargetUserID:      cloneInt64Pointer(session.TargetUserID),
		ResolvedEmail:     session.ResolvedEmail,
		RedirectTo:        session.RedirectTo,
		BrowserSessionKey: session.BrowserSessionKey,
		UpstreamClaims:    cloneAuthModuleMap(session.UpstreamIdentityClaims),
		FlowState:         cloneAuthModuleMap(session.LocalFlowState),
		ExpiresAt:         session.ExpiresAt,
		ConsumedAt:        cloneTimePointer(session.ConsumedAt),
	}
}

func cloneAuthModuleMap(values map[string]any) map[string]any {
	if values == nil {
		return nil
	}
	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

var (
	_ authports.AuthOperations          = (*AuthModuleAdapter)(nil)
	_ authports.AuthUnitOfWork          = (*AuthModuleAdapter)(nil)
	_ authports.LoginUseCases           = (*AuthModuleAdapter)(nil)
	_ authports.SignupUseCases          = (*AuthModuleAdapter)(nil)
	_ authports.PendingIdentityUseCases = (*AuthModuleAdapter)(nil)
)
