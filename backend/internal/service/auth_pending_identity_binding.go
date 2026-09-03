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
	"github.com/Wei-Shaw/sub2api/ent/pendingauthsession"
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
	txCtx, deferredInvalidations := withDeferredSubscriptionCacheInvalidations(txCtx)
	if err := s.finalizePendingOAuthAccountTx(txCtx, tx, input); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	flushDeferredSubscriptionCacheInvalidations(deferredInvalidations)
	// This method can also be called directly (without a caller-owned
	// transaction).  Keep the quota snapshot out of the uncommitted tx in that
	// case as well; the two HTTP coordinators that own an outer transaction call
	// the same hook after their commit.
	s.SnapshotPlatformQuotaDefaultsAfterCommit(ctx, input.User.ID, input.ProviderType)
	return nil
}

func (s *AuthService) finalizePendingOAuthAccountTx(ctx context.Context, tx *dbent.Tx, input FinalizePendingOAuthAccountInput) error {
	if tx == nil || input.Session == nil {
		return infraerrors.ServiceUnavailable("PENDING_AUTH_NOT_READY", "pending auth transaction is not ready")
	}
	releaseSessionLock, err := lockAuthPendingIdentityKeys(ctx, tx.Client(), pendingAuthSessionLockKey(input.Session.ID))
	if err != nil {
		return err
	}
	defer releaseSessionLock()
	// The handler obtains the session/decision before opening this transaction.
	// Reload and lock both rows so a concurrent finalizer cannot use a stale
	// target or overwrite a newer adoption choice after the initial read.
	sessionQuery := tx.Client().PendingAuthSession.Query().Where(
		pendingauthsession.IDEQ(input.Session.ID),
	)
	if pendingAuthSupportsSelectForUpdate(tx.Client()) {
		sessionQuery = sessionQuery.ForUpdate()
	}
	storedSession, err := sessionQuery.Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return ErrPendingAuthSessionNotFound
		}
		return err
	}
	pendingSession := pendingAuthSessionFromEntity(storedSession)
	if pendingSession == nil {
		return ErrPendingAuthSessionNotFound
	}
	if pendingSession.ConsumedAt != nil {
		return ErrPendingAuthSessionConsumed
	}
	now := time.Now().UTC()
	if !pendingSession.ExpiresAt.IsZero() && now.After(pendingSession.ExpiresAt) {
		return ErrPendingAuthSessionExpired
	}
	if strings.TrimSpace(pendingSession.BrowserSessionKey) != "" &&
		strings.TrimSpace(input.Session.BrowserSessionKey) != strings.TrimSpace(pendingSession.BrowserSessionKey) {
		return ErrPendingAuthBrowserMismatch
	}
	if provider := strings.TrimSpace(input.ProviderType); provider != "" &&
		!strings.EqualFold(provider, strings.TrimSpace(pendingSession.ProviderType)) {
		return infraerrors.BadRequest("PENDING_AUTH_PROVIDER_MISMATCH", "pending oauth provider does not match the session")
	}
	if pendingSession.TargetUserID != nil && *pendingSession.TargetUserID > 0 && *pendingSession.TargetUserID != input.User.ID {
		return infraerrors.Conflict("PENDING_AUTH_TARGET_USER_MISMATCH", "pending oauth session must be completed by the targeted user")
	}

	decision := input.Decision
	decisionQuery := tx.Client().IdentityAdoptionDecision.Query().Where(
		identityadoptiondecision.PendingAuthSessionIDEQ(pendingSession.ID),
	)
	if pendingAuthSupportsSelectForUpdate(tx.Client()) {
		decisionQuery = decisionQuery.ForUpdate()
	}
	storedDecision, decisionErr := decisionQuery.Only(ctx)
	if decisionErr == nil {
		decision = pendingIdentityDecisionFromEntity(storedDecision)
	} else if !dbent.IsNotFound(decisionErr) {
		return decisionErr
	}
	pending := NewAuthPendingIdentityService(s.entClient)
	if err := pending.applyBindingTx(ctx, tx, ApplyPendingIdentityBindingInput{
		Session:        pendingSession,
		Decision:       decision,
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
	if err := consumePendingIdentitySessionTx(ctx, tx, pendingSession); err != nil {
		return err
	}
	if input.BeforeCommit != nil {
		if err := input.BeforeCommit(ctx, pendingSession); err != nil {
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
		storedSession, releaseLock, err := reloadPendingIdentityBindingSession(ctx, tx, input.Session)
		if err != nil {
			releaseLock()
			return err
		}
		defer releaseLock()
		input.Session = storedSession
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
	txCtx, deferredInvalidations := withDeferredSubscriptionCacheInvalidations(txCtx)
	storedSession, releaseLock, err := reloadPendingIdentityBindingSession(txCtx, tx, input.Session)
	if err != nil {
		releaseLock()
		return err
	}
	defer releaseLock()
	input.Session = storedSession
	if err := s.applyBindingTx(txCtx, tx, input); err != nil {
		return err
	}
	if err := consumePendingIdentitySessionTx(txCtx, tx, input.Session); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	flushDeferredSubscriptionCacheInvalidations(deferredInvalidations)
	return nil
}

// reloadPendingIdentityBindingSession makes the database row authoritative
// before any identity/profile write.  The handler normally passes a snapshot
// loaded before opening its transaction; UpdateSessionProgress can otherwise
// change the target/provider tuple between that read and the binding write.
// Holding the scoped lock also serializes SQLite/unit flows, where SELECT FOR
// UPDATE is not available.
func reloadPendingIdentityBindingSession(
	ctx context.Context,
	tx *dbent.Tx,
	candidate *PendingAuthSession,
) (*PendingAuthSession, func(), error) {
	if tx == nil || candidate == nil || candidate.ID <= 0 {
		return nil, func() {}, infraerrors.BadRequest("PENDING_AUTH_SESSION_INVALID", "pending auth session is invalid")
	}
	release, err := lockAuthPendingIdentityKeys(ctx, tx.Client(), pendingAuthSessionLockKey(candidate.ID))
	if err != nil {
		return nil, func() {}, err
	}
	query := tx.Client().PendingAuthSession.Query().Where(pendingauthsession.IDEQ(candidate.ID))
	if pendingAuthSupportsSelectForUpdate(tx.Client()) {
		query = query.ForUpdate()
	}
	storedEntity, err := query.Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, release, ErrPendingAuthSessionNotFound
		}
		return nil, release, err
	}
	stored := pendingAuthSessionFromEntity(storedEntity)
	if stored == nil {
		return nil, release, ErrPendingAuthSessionNotFound
	}
	if stored.ConsumedAt != nil {
		return nil, release, ErrPendingAuthSessionConsumed
	}
	now := time.Now().UTC()
	if !stored.ExpiresAt.IsZero() && now.After(stored.ExpiresAt) {
		return nil, release, ErrPendingAuthSessionExpired
	}
	if strings.TrimSpace(candidate.BrowserSessionKey) != "" &&
		!strings.EqualFold(strings.TrimSpace(candidate.BrowserSessionKey), strings.TrimSpace(stored.BrowserSessionKey)) {
		return nil, release, ErrPendingAuthBrowserMismatch
	}
	// An ID-only input is supported for service-level callers; once any
	// snapshot fields are supplied, changes to the binding tuple are rejected
	// instead of silently applying the stale request to a new flow.
	hasSnapshot := strings.TrimSpace(candidate.Intent) != "" ||
		strings.TrimSpace(candidate.ProviderType) != "" ||
		strings.TrimSpace(candidate.ProviderKey) != "" ||
		strings.TrimSpace(candidate.ProviderSubject) != "" ||
		strings.TrimSpace(candidate.ResolvedEmail) != "" ||
		candidate.TargetUserID != nil
	if hasSnapshot {
		if strings.TrimSpace(candidate.Intent) != "" && !strings.EqualFold(strings.TrimSpace(candidate.Intent), strings.TrimSpace(stored.Intent)) {
			return nil, release, ErrPendingAuthSessionChanged
		}
		if strings.TrimSpace(candidate.ProviderType) != "" && !strings.EqualFold(strings.TrimSpace(candidate.ProviderType), strings.TrimSpace(stored.ProviderType)) {
			return nil, release, ErrPendingAuthSessionChanged
		}
		if strings.TrimSpace(candidate.ProviderKey) != "" && strings.TrimSpace(candidate.ProviderKey) != strings.TrimSpace(stored.ProviderKey) {
			return nil, release, ErrPendingAuthSessionChanged
		}
		if strings.TrimSpace(candidate.ProviderSubject) != "" && strings.TrimSpace(candidate.ProviderSubject) != strings.TrimSpace(stored.ProviderSubject) {
			return nil, release, ErrPendingAuthSessionChanged
		}
		if strings.TrimSpace(candidate.ResolvedEmail) != "" && !strings.EqualFold(strings.TrimSpace(candidate.ResolvedEmail), strings.TrimSpace(stored.ResolvedEmail)) {
			return nil, release, ErrPendingAuthSessionChanged
		}
		if pendingAuthTargetUserID(candidate) != pendingAuthTargetUserID(stored) {
			return nil, release, ErrPendingAuthSessionChanged
		}
	}
	return stored, release, nil
}

func pendingAuthTargetUserID(session *PendingAuthSession) int64 {
	if session == nil || session.TargetUserID == nil || *session.TargetUserID <= 0 {
		return 0
	}
	return *session.TargetUserID
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
	if tx == nil || session == nil || userID <= 0 {
		return nil, infraerrors.BadRequest("PENDING_AUTH_IDENTITY_INVALID", "pending auth identity is invalid")
	}
	// The provider tuple is the unique identity key.  Lock it before the
	// read/decide/create sequence so two finalizers cannot both observe a free
	// tuple and then race on ownership or metadata updates.  The lock helper
	// also keeps an in-process fallback for SQLite/unit-test clients.
	releaseLocks, err := lockAuthPendingIdentityKeys(ctx, tx.Client(), pendingIdentityBindingLockKeys(session)...)
	if err != nil {
		return nil, err
	}
	defer releaseLocks()

	if strings.EqualFold(strings.TrimSpace(session.ProviderType), "wechat") {
		return ensurePendingWeChatIdentityForUser(ctx, tx, session, userID)
	}
	client := tx.Client()
	identityQuery := client.AuthIdentity.Query().Where(
		authidentity.ProviderTypeEQ(strings.TrimSpace(session.ProviderType)),
		authidentity.ProviderKeyEQ(strings.TrimSpace(session.ProviderKey)),
		authidentity.ProviderSubjectEQ(strings.TrimSpace(session.ProviderSubject)),
	)
	if pendingAuthSupportsSelectForUpdate(client) {
		identityQuery = identityQuery.ForUpdate()
	}
	identity, err := identityQuery.Only(ctx)
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

	identityQuery := client.AuthIdentity.Query().Where(
		authidentity.ProviderTypeEQ(providerType), authidentity.ProviderKeyIn(providerKeys...), authidentity.ProviderSubjectEQ(providerSubject),
	)
	if pendingAuthSupportsSelectForUpdate(client) {
		identityQuery = identityQuery.ForUpdate()
	}
	records, err := identityQuery.All(ctx)
	if err != nil {
		return nil, err
	}
	identity, canonical, err := choosePendingWeChatIdentity(ctx, client, records, userID, providerKey)
	if err != nil {
		return nil, err
	}
	var legacy *dbent.AuthIdentity
	if channelSubject != "" && channelSubject != providerSubject {
		legacyQuery := client.AuthIdentity.Query().Where(
			authidentity.ProviderTypeEQ(providerType), authidentity.ProviderKeyIn(providerKeys...), authidentity.ProviderSubjectEQ(channelSubject),
		)
		if pendingAuthSupportsSelectForUpdate(client) {
			legacyQuery = legacyQuery.ForUpdate()
		}
		legacyRecords, err := legacyQuery.All(ctx)
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

	channelQuery := client.AuthIdentityChannel.Query().Where(
		authidentitychannel.ProviderTypeEQ(providerType), authidentitychannel.ProviderKeyIn(providerKeys...),
		authidentitychannel.ChannelEQ(channel), authidentitychannel.ChannelAppIDEQ(channelAppID),
		authidentitychannel.ChannelSubjectEQ(channelSubject),
	).WithIdentity()
	if pendingAuthSupportsSelectForUpdate(client) {
		channelQuery = channelQuery.ForUpdate()
	}
	channelRecords, err := channelQuery.All(ctx)
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
	storedQuery := tx.Client().PendingAuthSession.Query().Where(
		pendingauthsession.IDEQ(session.ID),
	)
	if pendingAuthSupportsSelectForUpdate(tx.Client()) {
		storedQuery = storedQuery.ForUpdate()
	}
	stored, err := storedQuery.Only(ctx)
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
	if stored.CompletionCodeExpiresAt != nil && now.After(*stored.CompletionCodeExpiresAt) {
		return ErrPendingAuthSessionExpired
	}
	if strings.TrimSpace(stored.BrowserSessionKey) != "" && strings.TrimSpace(stored.BrowserSessionKey) != strings.TrimSpace(session.BrowserSessionKey) {
		return ErrPendingAuthBrowserMismatch
	}
	updated, err := tx.Client().PendingAuthSession.UpdateOneID(stored.ID).Where(
		pendingauthsession.ConsumedAtIsNil(),
		pendingauthsession.ExpiresAtGTE(now),
		pendingauthsession.Or(
			pendingauthsession.CompletionCodeExpiresAtIsNil(),
			pendingauthsession.CompletionCodeExpiresAtGTE(now),
		),
	).SetConsumedAt(now).
		SetLocalFlowState(sanitizePendingAuthLocalFlowState(stored.LocalFlowState)).
		SetCompletionCodeHash("").ClearCompletionCodeExpiresAt().Save(ctx)
	if err == nil {
		_ = updated
		return nil
	}
	if !dbent.IsNotFound(err) {
		return err
	}
	// A caller may have consumed the row through another path that does not
	// share this transaction.  Re-read only to classify the race; never report
	// success after a zero-row conditional update.
	current, currentErr := tx.Client().PendingAuthSession.Get(ctx, stored.ID)
	if currentErr != nil {
		if dbent.IsNotFound(currentErr) {
			return ErrPendingAuthSessionNotFound
		}
		return currentErr
	}
	if current.ConsumedAt != nil {
		return ErrPendingAuthSessionConsumed
	}
	if !current.ExpiresAt.IsZero() && now.After(current.ExpiresAt) {
		return ErrPendingAuthSessionExpired
	}
	if current.CompletionCodeExpiresAt != nil && now.After(*current.CompletionCodeExpiresAt) {
		return ErrPendingAuthSessionExpired
	}
	if strings.TrimSpace(current.BrowserSessionKey) != "" && strings.TrimSpace(current.BrowserSessionKey) != strings.TrimSpace(session.BrowserSessionKey) {
		return ErrPendingAuthBrowserMismatch
	}
	return ErrPendingAuthSessionConsumed
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
