package service

import (
	"context"
	"strings"

	entsql "entgo.io/ent/dialect/sql"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/authidentity"
	"github.com/Wei-Shaw/sub2api/ent/identityadoptiondecision"
	"github.com/Wei-Shaw/sub2api/ent/predicate"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type UpdatePendingAuthSessionProgressInput struct {
	SessionID          int64
	Intent             string
	ResolvedEmail      string
	TargetUserID       *int64
	CompletionResponse map[string]any
}

func (s *AuthPendingIdentityService) UpdateSessionProgress(ctx context.Context, input UpdatePendingAuthSessionProgressInput) (*PendingAuthSession, error) {
	if s == nil || s.entClient == nil || input.SessionID <= 0 {
		return nil, infraerrors.BadRequest("PENDING_AUTH_SESSION_INVALID", "pending auth session is invalid")
	}

	session, err := s.entClient.PendingAuthSession.Get(ctx, input.SessionID)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, ErrPendingAuthSessionNotFound
		}
		return nil, err
	}
	localFlowState := copyPendingMap(session.LocalFlowState)
	localFlowState["completion_response"] = copyPendingMap(input.CompletionResponse)

	update := s.entClient.PendingAuthSession.UpdateOneID(session.ID).
		SetIntent(strings.TrimSpace(input.Intent)).
		SetResolvedEmail(strings.TrimSpace(input.ResolvedEmail)).
		SetLocalFlowState(localFlowState)
	if input.TargetUserID != nil && *input.TargetUserID > 0 {
		update = update.SetTargetUserID(*input.TargetUserID)
	} else {
		update = update.ClearTargetUserID()
	}
	updated, err := update.Save(ctx)
	return pendingAuthSessionFromEntity(updated), err
}

func (s *AuthPendingIdentityService) GetAdoptionDecision(ctx context.Context, pendingAuthSessionID int64) (*PendingIdentityDecision, error) {
	if s == nil || s.entClient == nil || pendingAuthSessionID <= 0 {
		return nil, infraerrors.BadRequest("PENDING_AUTH_SESSION_INVALID", "pending auth session is invalid")
	}
	decision, err := s.entClient.IdentityAdoptionDecision.Query().
		Where(identityadoptiondecision.PendingAuthSessionIDEQ(pendingAuthSessionID)).
		Only(ctx)
	if dbent.IsNotFound(err) {
		return nil, nil
	}
	return pendingIdentityDecisionFromEntity(decision), err
}

func normalizedPendingAuthEmailPredicate(email string) predicate.User {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if normalized == "" {
		return dbuser.EmailEQ(email)
	}
	return predicate.User(func(selector *entsql.Selector) {
		selector.Where(entsql.P(func(builder *entsql.Builder) {
			builder.WriteString("LOWER(TRIM(").
				Ident(selector.C(dbuser.FieldEmail)).
				WriteString(")) = ").
				Arg(normalized)
		}))
	})
}

func (s *AuthPendingIdentityService) FindUserByNormalizedEmail(ctx context.Context, email string) (*AuthIdentityUser, error) {
	if s == nil || s.entClient == nil {
		return nil, infraerrors.ServiceUnavailable("PENDING_AUTH_NOT_READY", "pending auth service is not ready")
	}
	matches, err := s.entClient.User.Query().
		Where(normalizedPendingAuthEmailPredicate(email)).
		Order(dbent.Asc(dbuser.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	switch len(matches) {
	case 0:
		return nil, ErrUserNotFound
	case 1:
		return authIdentityUserFromEntity(matches[0]), nil
	default:
		return nil, infraerrors.Conflict("USER_EMAIL_CONFLICT", "normalized email matched multiple users")
	}
}

func (s *AuthPendingIdentityService) FindIdentityUser(ctx context.Context, identity PendingAuthIdentityKey) (*AuthIdentityUser, error) {
	if s == nil || s.entClient == nil {
		return nil, infraerrors.ServiceUnavailable("PENDING_AUTH_NOT_READY", "pending auth service is not ready")
	}
	record, err := s.entClient.AuthIdentity.Query().
		Where(
			authidentity.ProviderTypeEQ(strings.TrimSpace(identity.ProviderType)),
			authidentity.ProviderKeyEQ(strings.TrimSpace(identity.ProviderKey)),
			authidentity.ProviderSubjectEQ(strings.TrimSpace(identity.ProviderSubject)),
		).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, nil
		}
		return nil, infraerrors.InternalServer("AUTH_IDENTITY_LOOKUP_FAILED", "failed to inspect auth identity ownership").WithCause(err)
	}
	return s.findActiveUserByID(ctx, record.UserID)
}

func (s *AuthPendingIdentityService) EnsureRegistrationIdentityAvailable(ctx context.Context, session *PendingAuthSession) error {
	if s == nil || s.entClient == nil || session == nil {
		return infraerrors.BadRequest("PENDING_AUTH_SESSION_INVALID", "pending auth registration context is invalid")
	}
	identity, err := s.entClient.AuthIdentity.Query().
		Where(
			authidentity.ProviderTypeEQ(strings.TrimSpace(session.ProviderType)),
			authidentity.ProviderKeyEQ(strings.TrimSpace(session.ProviderKey)),
			authidentity.ProviderSubjectEQ(strings.TrimSpace(session.ProviderSubject)),
		).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil
		}
		return err
	}
	if identity == nil || identity.UserID <= 0 {
		return nil
	}
	activeOwner, err := s.findActiveUserByID(ctx, identity.UserID)
	if err != nil {
		return err
	}
	if activeOwner != nil {
		return infraerrors.Conflict("AUTH_IDENTITY_OWNERSHIP_CONFLICT", "auth identity already belongs to another user")
	}
	return nil
}

func (s *AuthPendingIdentityService) IdentityExistsForUser(ctx context.Context, session *PendingAuthSession, userID int64, compatibleProviderKeys []string) (bool, error) {
	if s == nil || s.entClient == nil || session == nil || userID <= 0 {
		return false, nil
	}
	providerType := strings.TrimSpace(session.ProviderType)
	providerKey := strings.TrimSpace(session.ProviderKey)
	providerSubject := strings.TrimSpace(session.ProviderSubject)
	if providerType == "" || providerSubject == "" {
		return false, nil
	}

	query := s.entClient.AuthIdentity.Query().Where(
		authidentity.ProviderTypeEQ(providerType),
		authidentity.ProviderSubjectEQ(providerSubject),
		authidentity.UserIDEQ(userID),
	)
	if len(compatibleProviderKeys) > 0 {
		query = query.Where(authidentity.ProviderKeyIn(compatibleProviderKeys...))
	} else if providerKey != "" {
		query = query.Where(authidentity.ProviderKeyEQ(providerKey))
	}
	count, err := query.Count(ctx)
	if err != nil {
		return false, infraerrors.InternalServer("AUTH_IDENTITY_LOOKUP_FAILED", "failed to inspect auth identity ownership").WithCause(err)
	}
	return count > 0, nil
}

func (s *AuthPendingIdentityService) findActiveUserByID(ctx context.Context, userID int64) (*AuthIdentityUser, error) {
	if userID <= 0 {
		return nil, nil
	}
	userEntity, err := s.entClient.User.Get(ctx, userID)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, nil
		}
		return nil, infraerrors.InternalServer("AUTH_IDENTITY_USER_LOOKUP_FAILED", "failed to load auth identity user").WithCause(err)
	}
	if !strings.EqualFold(strings.TrimSpace(userEntity.Status), StatusActive) {
		return nil, ErrUserNotActive
	}
	return authIdentityUserFromEntity(userEntity), nil
}
