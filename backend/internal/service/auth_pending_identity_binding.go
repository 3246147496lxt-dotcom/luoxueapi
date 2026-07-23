package service

import (
	"context"
	"errors"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/authidentity"
	"github.com/Wei-Shaw/sub2api/ent/authidentitychannel"
	"github.com/Wei-Shaw/sub2api/ent/identityadoptiondecision"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	pendingWeChatProviderKey       = "wechat-main"
	pendingWeChatLegacyProviderKey = "wechat"
)

type PendingIdentityDefaultApplier interface {
	ApplyProviderDefaultSettingsOnFirstBind(ctx context.Context, userID int64, providerType string) error
}

type PendingIdentityAvatarWriter interface {
	SetAvatar(ctx context.Context, userID int64, raw string) (*UserAvatar, error)
}

type ApplyPendingIdentityBindingInput struct {
	Session                *PendingAuthSession
	Decision               *PendingIdentityDecision
	OverrideUserID         *int64
	ForceBind              bool
	ApplyFirstBindDefaults bool
	DefaultApplier         PendingIdentityDefaultApplier
	AvatarWriter           PendingIdentityAvatarWriter
}

type FinalizePendingOAuthAccountInput struct {
	Session        *PendingAuthSession
	Decision       *PendingIdentityDecision
	User           *User
	InvitationCode string
	ProviderType   string
	AffiliateCode  string
	AvatarWriter   PendingIdentityAvatarWriter
	BeforeCommit   func(context.Context, *PendingAuthSession) error
}

// FinalizePendingOAuthAccount commits identity binding, signup grants and
// pending-session consumption as one database transaction.
func (s *AuthService) FinalizePendingOAuthAccount(ctx context.Context, input FinalizePendingOAuthAccountInput) error {
	if s == nil || s.entClient == nil || input.Session == nil || input.User == nil || input.User.ID <= 0 {
		return infraerrors.ServiceUnavailable("PENDING_AUTH_NOT_READY", "pending auth service is not ready")
	}
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return s.finalizePendingOAuthAccountTx(ctx, tx, input)
	}
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)
	if err := s.finalizePendingOAuthAccountTx(txCtx, tx, input); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *AuthService) finalizePendingOAuthAccountTx(ctx context.Context, tx *dbent.Tx, input FinalizePendingOAuthAccountInput) error {
	if tx == nil {
		return infraerrors.ServiceUnavailable("PENDING_AUTH_NOT_READY", "pending auth transaction is not ready")
	}
	pending := NewAuthPendingIdentityService(s.entClient)
	if err := pending.applyBindingTx(ctx, tx, ApplyPendingIdentityBindingInput{
		Session:        input.Session,
		Decision:       input.Decision,
		OverrideUserID: &input.User.ID,
		ForceBind:      true,
		DefaultApplier: s,
		AvatarWriter:   input.AvatarWriter,
	}); err != nil {
		return err
	}
	if err := s.FinalizeOAuthEmailAccount(
		ctx,
		input.User,
		strings.TrimSpace(input.InvitationCode),
		strings.TrimSpace(input.ProviderType),
		strings.TrimSpace(input.AffiliateCode),
	); err != nil {
		return err
	}
	if err := consumePendingIdentitySessionTx(ctx, tx, input.Session); err != nil {
		return err
	}
	if input.BeforeCommit != nil {
		if err := input.BeforeCommit(ctx, input.Session); err != nil {
			return err
		}
	}
	return nil
}

func (s *AuthPendingIdentityService) ApplyBinding(ctx context.Context, input ApplyPendingIdentityBindingInput) error {
	if s == nil || s.entClient == nil || input.Session == nil {
		return nil
	}
	if !input.ForceBind && !shouldBindPendingIdentity(input.Session, input.Decision) {
		return nil
	}
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return s.applyBindingTx(ctx, tx, input)
	}
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)
	if err := s.applyBindingTx(txCtx, tx, input); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *AuthPendingIdentityService) ApplyBindingAndConsume(ctx context.Context, input ApplyPendingIdentityBindingInput) error {
	if s == nil || s.entClient == nil || input.Session == nil {
		return infraerrors.BadRequest("PENDING_AUTH_SESSION_INVALID", "pending auth registration context is invalid")
	}
	if tx := dbent.TxFromContext(ctx); tx != nil {
		if err := s.applyBindingTx(ctx, tx, input); err != nil {
			return err
		}
		return consumePendingIdentitySessionTx(ctx, tx, input.Session)
	}
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)
	if err := s.applyBindingTx(txCtx, tx, input); err != nil {
		return err
	}
	if err := consumePendingIdentitySessionTx(txCtx, tx, input.Session); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *AuthPendingIdentityService) applyBindingTx(ctx context.Context, tx *dbent.Tx, input ApplyPendingIdentityBindingInput) error {
	if tx == nil || input.Session == nil {
		return nil
	}
	if !input.ForceBind && !shouldBindPendingIdentity(input.Session, input.Decision) {
		return nil
	}

	targetUserID := int64(0)
	if input.OverrideUserID != nil && *input.OverrideUserID > 0 {
		targetUserID = *input.OverrideUserID
	} else {
		resolved, err := resolvePendingIdentityTargetUserID(ctx, tx.Client(), input.Session)
		if err != nil {
			return err
		}
		targetUserID = resolved
	}

	adoptedDisplayName := ""
	if input.Decision != nil && input.Decision.AdoptDisplayName {
		adoptedDisplayName = normalizePendingIdentityDisplayName(pendingIdentityStringValue(input.Session.UpstreamIdentityClaims, "suggested_display_name"))
	}
	adoptedAvatarURL := ""
	if input.Decision != nil && input.Decision.AdoptAvatar {
		adoptedAvatarURL = pendingIdentityStringValue(input.Session.UpstreamIdentityClaims, "suggested_avatar_url")
	}
	shouldAdoptAvatar := false
	if input.Decision != nil && input.Decision.AdoptAvatar && adoptedAvatarURL != "" {
		if err := ValidateUserAvatar(adoptedAvatarURL); err == nil {
			shouldAdoptAvatar = true
		} else if !errors.Is(err, ErrAvatarInvalid) && !errors.Is(err, ErrAvatarTooLarge) && !errors.Is(err, ErrAvatarNotImage) {
			return err
		}
	}

	if input.Decision != nil && input.Decision.AdoptDisplayName && adoptedDisplayName != "" {
		if err := tx.Client().User.UpdateOneID(targetUserID).SetUsername(adoptedDisplayName).Exec(ctx); err != nil {
			return err
		}
	}

	identity, err := ensurePendingIdentityForUser(ctx, tx, input.Session, targetUserID)
	if err != nil {
		return err
	}
	metadata := copyPendingMap(identity.Metadata)
	for key, value := range input.Session.UpstreamIdentityClaims {
		metadata[key] = value
	}
	if input.Decision != nil && input.Decision.AdoptDisplayName && adoptedDisplayName != "" {
		metadata["display_name"] = adoptedDisplayName
	}
	if shouldAdoptAvatar {
		metadata["avatar_url"] = adoptedAvatarURL
	}
	updateIdentity := tx.Client().AuthIdentity.UpdateOneID(identity.ID).SetMetadata(metadata)
	if issuer := pendingIdentityIssuer(input.Session); issuer != nil {
		updateIdentity = updateIdentity.SetIssuer(strings.TrimSpace(*issuer))
	}
	if _, err := updateIdentity.Save(ctx); err != nil {
		return err
	}

	if input.Decision != nil && (input.Decision.IdentityID == nil || *input.Decision.IdentityID != identity.ID) {
		if _, err := tx.Client().IdentityAdoptionDecision.Update().
			Where(identityadoptiondecision.IdentityIDEQ(identity.ID), identityadoptiondecision.IDNEQ(input.Decision.ID)).
			ClearIdentityID().Save(ctx); err != nil {
			return err
		}
		if _, err := tx.Client().IdentityAdoptionDecision.UpdateOneID(input.Decision.ID).SetIdentityID(identity.ID).Save(ctx); err != nil {
			return err
		}
	}

	if input.ApplyFirstBindDefaults && input.DefaultApplier != nil {
		if err := input.DefaultApplier.ApplyProviderDefaultSettingsOnFirstBind(ctx, targetUserID, input.Session.ProviderType); err != nil {
			return err
		}
	}
	if shouldAdoptAvatar && input.AvatarWriter != nil {
		if _, err := input.AvatarWriter.SetAvatar(ctx, targetUserID, adoptedAvatarURL); err != nil {
			return err
		}
	}
	return nil
}

func resolvePendingIdentityTargetUserID(ctx context.Context, client *dbent.Client, session *PendingAuthSession) (int64, error) {
	if session == nil {
		return 0, infraerrors.BadRequest("PENDING_AUTH_SESSION_INVALID", "pending auth session is invalid")
	}
	if session.TargetUserID != nil && *session.TargetUserID > 0 {
		return *session.TargetUserID, nil
	}
	if strings.TrimSpace(session.ResolvedEmail) == "" {
		return 0, infraerrors.BadRequest("PENDING_AUTH_TARGET_USER_MISSING", "pending auth target user is missing")
	}
	matches, err := client.User.Query().Where(normalizedPendingAuthEmailPredicate(session.ResolvedEmail)).All(ctx)
	if err != nil {
		return 0, err
	}
	if len(matches) != 1 {
		return 0, infraerrors.InternalServer("PENDING_AUTH_TARGET_USER_NOT_FOUND", "pending auth target user was not found")
	}
	return matches[0].ID, nil
}

func ensurePendingIdentityForUser(ctx context.Context, tx *dbent.Tx, session *PendingAuthSession, userID int64) (*dbent.AuthIdentity, error) {
	if strings.EqualFold(strings.TrimSpace(session.ProviderType), "wechat") {
		return ensurePendingWeChatIdentityForUser(ctx, tx, session, userID)
	}
	client := tx.Client()
	identity, err := client.AuthIdentity.Query().Where(
		authidentity.ProviderTypeEQ(strings.TrimSpace(session.ProviderType)),
		authidentity.ProviderKeyEQ(strings.TrimSpace(session.ProviderKey)),
		authidentity.ProviderSubjectEQ(strings.TrimSpace(session.ProviderSubject)),
	).Only(ctx)
	if err != nil && !dbent.IsNotFound(err) {
		return nil, err
	}
	if identity != nil {
		if identity.UserID != userID {
			activeOwner, err := findActivePendingIdentityUser(ctx, client, identity.UserID)
			if err != nil {
				return nil, err
			}
			if activeOwner != nil {
				return nil, infraerrors.Conflict("AUTH_IDENTITY_OWNERSHIP_CONFLICT", "auth identity already belongs to another user")
			}
			return client.AuthIdentity.UpdateOneID(identity.ID).SetUserID(userID).Save(ctx)
		}
		return identity, nil
	}
	create := client.AuthIdentity.Create().
		SetUserID(userID).
		SetProviderType(strings.TrimSpace(session.ProviderType)).
		SetProviderKey(strings.TrimSpace(session.ProviderKey)).
		SetProviderSubject(strings.TrimSpace(session.ProviderSubject)).
		SetMetadata(copyPendingMap(session.UpstreamIdentityClaims))
	if issuer := pendingIdentityIssuer(session); issuer != nil {
		create = create.SetIssuer(strings.TrimSpace(*issuer))
	}
	return create.Save(ctx)
}

func ensurePendingWeChatIdentityForUser(ctx context.Context, tx *dbent.Tx, session *PendingAuthSession, userID int64) (*dbent.AuthIdentity, error) {
	client := tx.Client()
	providerType := strings.TrimSpace(session.ProviderType)
	providerKey := strings.TrimSpace(session.ProviderKey)
	providerSubject := strings.TrimSpace(session.ProviderSubject)
	providerKeys := pendingWeChatCompatibleProviderKeys(providerKey)
	channel := pendingIdentityStringValue(session.UpstreamIdentityClaims, "channel")
	channelAppID := pendingIdentityStringValue(session.UpstreamIdentityClaims, "channel_app_id")
	channelSubject := pendingIdentityStringValue(session.UpstreamIdentityClaims, "channel_subject")
	metadata := copyPendingMap(session.UpstreamIdentityClaims)

	records, err := client.AuthIdentity.Query().Where(
		authidentity.ProviderTypeEQ(providerType), authidentity.ProviderKeyIn(providerKeys...), authidentity.ProviderSubjectEQ(providerSubject),
	).All(ctx)
	if err != nil {
		return nil, err
	}
	identity, canonical, err := choosePendingWeChatIdentity(ctx, client, records, userID, providerKey)
	if err != nil {
		return nil, err
	}
	var legacy *dbent.AuthIdentity
	if channelSubject != "" && channelSubject != providerSubject {
		legacyRecords, err := client.AuthIdentity.Query().Where(
			authidentity.ProviderTypeEQ(providerType), authidentity.ProviderKeyIn(providerKeys...), authidentity.ProviderSubjectEQ(channelSubject),
		).All(ctx)
		if err != nil {
			return nil, err
		}
		legacy, _, err = choosePendingWeChatIdentity(ctx, client, legacyRecords, userID, providerKey)
		if err != nil {
			return nil, err
		}
	}

	switch {
	case identity != nil:
		update := client.AuthIdentity.UpdateOneID(identity.ID).SetMetadata(mergePendingIdentityMetadata(identity.Metadata, metadata))
		if identity.UserID != userID {
			update = update.SetUserID(userID)
		}
		if !strings.EqualFold(strings.TrimSpace(identity.ProviderKey), providerKey) && !canonical {
			update = update.SetProviderKey(providerKey)
		}
		if issuer := pendingIdentityIssuer(session); issuer != nil {
			update = update.SetIssuer(strings.TrimSpace(*issuer))
		}
		identity, err = update.Save(ctx)
	case legacy != nil:
		update := client.AuthIdentity.UpdateOneID(legacy.ID).
			SetProviderKey(providerKey).SetProviderSubject(providerSubject).
			SetMetadata(mergePendingIdentityMetadata(legacy.Metadata, metadata))
		if issuer := pendingIdentityIssuer(session); issuer != nil {
			update = update.SetIssuer(strings.TrimSpace(*issuer))
		}
		identity, err = update.Save(ctx)
	default:
		create := client.AuthIdentity.Create().SetUserID(userID).SetProviderType(providerType).
			SetProviderKey(providerKey).SetProviderSubject(providerSubject).SetMetadata(metadata)
		if issuer := pendingIdentityIssuer(session); issuer != nil {
			create = create.SetIssuer(strings.TrimSpace(*issuer))
		}
		identity, err = create.Save(ctx)
	}
	if err != nil || channel == "" || channelAppID == "" || channelSubject == "" {
		return identity, err
	}

	channelRecords, err := client.AuthIdentityChannel.Query().Where(
		authidentitychannel.ProviderTypeEQ(providerType), authidentitychannel.ProviderKeyIn(providerKeys...),
		authidentitychannel.ChannelEQ(channel), authidentitychannel.ChannelAppIDEQ(channelAppID),
		authidentitychannel.ChannelSubjectEQ(channelSubject),
	).WithIdentity().All(ctx)
	if err != nil {
		return nil, err
	}
	channelRecord, canonicalChannel, err := choosePendingWeChatChannel(ctx, client, channelRecords, userID, providerKey)
	if err != nil {
		return nil, err
	}
	channelMetadata := metadata
	if channelRecord != nil {
		channelMetadata = mergePendingIdentityMetadata(channelRecord.Metadata, metadata)
	}
	if channelRecord == nil {
		_, err = client.AuthIdentityChannel.Create().SetIdentityID(identity.ID).SetProviderType(providerType).
			SetProviderKey(providerKey).SetChannel(channel).SetChannelAppID(channelAppID).
			SetChannelSubject(channelSubject).SetMetadata(channelMetadata).Save(ctx)
		return identity, err
	}
	updateChannel := client.AuthIdentityChannel.UpdateOneID(channelRecord.ID).SetIdentityID(identity.ID).SetMetadata(channelMetadata)
	if !strings.EqualFold(strings.TrimSpace(channelRecord.ProviderKey), providerKey) && !canonicalChannel {
		updateChannel = updateChannel.SetProviderKey(providerKey)
	}
	_, err = updateChannel.Save(ctx)
	return identity, err
}

func choosePendingWeChatIdentity(ctx context.Context, client *dbent.Client, records []*dbent.AuthIdentity, userID int64, preferredKey string) (*dbent.AuthIdentity, bool, error) {
	var preferred, fallback *dbent.AuthIdentity
	canonical := false
	for _, record := range records {
		if record == nil {
			continue
		}
		if record.UserID != userID {
			owner, err := findActivePendingIdentityUser(ctx, client, record.UserID)
			if err != nil {
				return nil, false, err
			}
			if owner != nil {
				return nil, false, infraerrors.Conflict("AUTH_IDENTITY_OWNERSHIP_CONFLICT", "auth identity already belongs to another user")
			}
		}
		if strings.EqualFold(strings.TrimSpace(record.ProviderKey), preferredKey) {
			canonical = true
			if preferred == nil {
				preferred = record
			}
		} else if fallback == nil {
			fallback = record
		}
	}
	if preferred != nil {
		return preferred, canonical, nil
	}
	return fallback, canonical, nil
}

func choosePendingWeChatChannel(ctx context.Context, client *dbent.Client, records []*dbent.AuthIdentityChannel, userID int64, preferredKey string) (*dbent.AuthIdentityChannel, bool, error) {
	var preferred, fallback *dbent.AuthIdentityChannel
	canonical := false
	for _, record := range records {
		if record == nil {
			continue
		}
		if record.Edges.Identity != nil && record.Edges.Identity.UserID != userID {
			owner, err := findActivePendingIdentityUser(ctx, client, record.Edges.Identity.UserID)
			if err != nil {
				return nil, false, err
			}
			if owner != nil {
				return nil, false, infraerrors.Conflict("AUTH_IDENTITY_CHANNEL_OWNERSHIP_CONFLICT", "auth identity channel already belongs to another user")
			}
		}
		if strings.EqualFold(strings.TrimSpace(record.ProviderKey), preferredKey) {
			canonical = true
			if preferred == nil {
				preferred = record
			}
		} else if fallback == nil {
			fallback = record
		}
	}
	if preferred != nil {
		return preferred, canonical, nil
	}
	return fallback, canonical, nil
}

func findActivePendingIdentityUser(ctx context.Context, client *dbent.Client, userID int64) (*dbent.User, error) {
	if client == nil || userID <= 0 {
		return nil, nil
	}
	user, err := client.User.Get(ctx, userID)
	if dbent.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, infraerrors.InternalServer("AUTH_IDENTITY_USER_LOOKUP_FAILED", "failed to load auth identity user").WithCause(err)
	}
	if !strings.EqualFold(strings.TrimSpace(user.Status), StatusActive) {
		return nil, ErrUserNotActive
	}
	return user, nil
}

func consumePendingIdentitySessionTx(ctx context.Context, tx *dbent.Tx, session *PendingAuthSession) error {
	if tx == nil || session == nil {
		return ErrPendingAuthSessionNotFound
	}
	stored, err := tx.Client().PendingAuthSession.Get(ctx, session.ID)
	if dbent.IsNotFound(err) {
		return ErrPendingAuthSessionNotFound
	}
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if stored.ConsumedAt != nil {
		return ErrPendingAuthSessionConsumed
	}
	if !stored.ExpiresAt.IsZero() && now.After(stored.ExpiresAt) {
		return ErrPendingAuthSessionExpired
	}
	if strings.TrimSpace(stored.BrowserSessionKey) != "" && strings.TrimSpace(stored.BrowserSessionKey) != strings.TrimSpace(session.BrowserSessionKey) {
		return ErrPendingAuthBrowserMismatch
	}
	_, err = tx.Client().PendingAuthSession.UpdateOneID(stored.ID).SetConsumedAt(now).
		SetCompletionCodeHash("").ClearCompletionCodeExpiresAt().Save(ctx)
	return err
}

func shouldBindPendingIdentity(session *PendingAuthSession, decision *PendingIdentityDecision) bool {
	if session == nil || decision == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(session.Intent)) {
	case "bind_current_user", "login", "adopt_existing_user_by_email":
		return true
	default:
		return decision.AdoptDisplayName || decision.AdoptAvatar
	}
}

func pendingIdentityIssuer(session *PendingAuthSession) *string {
	if session == nil {
		return nil
	}
	issuer := pendingIdentityStringValue(session.UpstreamIdentityClaims, "issuer")
	if strings.TrimSpace(session.ProviderType) == "oidc" && strings.TrimSpace(session.ProviderKey) != "" {
		issuer = strings.TrimSpace(session.ProviderKey)
	}
	if issuer == "" {
		return nil
	}
	return &issuer
}

func pendingIdentityStringValue(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return strings.TrimSpace(value)
}

func normalizePendingIdentityDisplayName(value string) string {
	value = strings.TrimSpace(value)
	if runes := []rune(value); len(runes) > 100 {
		return string(runes[:100])
	}
	return value
}

func mergePendingIdentityMetadata(base, overlay map[string]any) map[string]any {
	merged := copyPendingMap(base)
	for key, value := range overlay {
		merged[key] = value
	}
	return merged
}

func pendingWeChatCompatibleProviderKeys(providerKey string) []string {
	preferred := strings.TrimSpace(providerKey)
	if preferred == "" {
		preferred = pendingWeChatProviderKey
	}
	keys := []string{preferred}
	if !strings.EqualFold(preferred, pendingWeChatLegacyProviderKey) {
		keys = append(keys, pendingWeChatLegacyProviderKey)
	}
	return keys
}
