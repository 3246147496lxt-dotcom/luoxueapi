package service

import (
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
)

// PendingAuthSession is the transport-neutral pending identity aggregate.
// ORM entities are converted at the service boundary and never reach handlers.
type PendingAuthSession struct {
	ID                       int64
	CreatedAt                time.Time
	UpdatedAt                time.Time
	SessionToken             string
	Intent                   string
	ProviderType             string
	ProviderKey              string
	ProviderSubject          string
	TargetUserID             *int64
	RedirectTo               string
	ResolvedEmail            string
	RegistrationPasswordHash string
	UpstreamIdentityClaims   map[string]any
	LocalFlowState           map[string]any
	BrowserSessionKey        string
	CompletionCodeHash       string
	CompletionCodeExpiresAt  *time.Time
	EmailVerifiedAt          *time.Time
	PasswordVerifiedAt       *time.Time
	TotpVerifiedAt           *time.Time
	ExpiresAt                time.Time
	ConsumedAt               *time.Time
}

type PendingIdentityDecision struct {
	ID                   int64
	PendingAuthSessionID int64
	IdentityID           *int64
	AdoptDisplayName     bool
	AdoptAvatar          bool
	DecidedAt            time.Time
}

type AuthIdentityUser struct {
	ID           int64
	Email        string
	Username     string
	Status       string
	SignupSource string
}

func pendingAuthSessionFromEntity(entity *dbent.PendingAuthSession) *PendingAuthSession {
	if entity == nil {
		return nil
	}
	return &PendingAuthSession{
		ID:                       entity.ID,
		CreatedAt:                entity.CreatedAt,
		UpdatedAt:                entity.UpdatedAt,
		SessionToken:             entity.SessionToken,
		Intent:                   entity.Intent,
		ProviderType:             entity.ProviderType,
		ProviderKey:              entity.ProviderKey,
		ProviderSubject:          entity.ProviderSubject,
		TargetUserID:             cloneInt64Pointer(entity.TargetUserID),
		RedirectTo:               entity.RedirectTo,
		ResolvedEmail:            entity.ResolvedEmail,
		RegistrationPasswordHash: entity.RegistrationPasswordHash,
		UpstreamIdentityClaims:   copyPendingMap(entity.UpstreamIdentityClaims),
		LocalFlowState:           copyPendingMap(entity.LocalFlowState),
		BrowserSessionKey:        entity.BrowserSessionKey,
		CompletionCodeHash:       entity.CompletionCodeHash,
		CompletionCodeExpiresAt:  cloneTimePointer(entity.CompletionCodeExpiresAt),
		EmailVerifiedAt:          cloneTimePointer(entity.EmailVerifiedAt),
		PasswordVerifiedAt:       cloneTimePointer(entity.PasswordVerifiedAt),
		TotpVerifiedAt:           cloneTimePointer(entity.TotpVerifiedAt),
		ExpiresAt:                entity.ExpiresAt,
		ConsumedAt:               cloneTimePointer(entity.ConsumedAt),
	}
}

func pendingIdentityDecisionFromEntity(entity *dbent.IdentityAdoptionDecision) *PendingIdentityDecision {
	if entity == nil {
		return nil
	}
	return &PendingIdentityDecision{
		ID:                   entity.ID,
		PendingAuthSessionID: entity.PendingAuthSessionID,
		IdentityID:           cloneInt64Pointer(entity.IdentityID),
		AdoptDisplayName:     entity.AdoptDisplayName,
		AdoptAvatar:          entity.AdoptAvatar,
		DecidedAt:            entity.DecidedAt,
	}
}

func authIdentityUserFromEntity(entity *dbent.User) *AuthIdentityUser {
	if entity == nil {
		return nil
	}
	return &AuthIdentityUser{
		ID:           entity.ID,
		Email:        entity.Email,
		Username:     entity.Username,
		Status:       entity.Status,
		SignupSource: entity.SignupSource,
	}
}

func cloneInt64Pointer(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
