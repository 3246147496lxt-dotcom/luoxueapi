package service

import (
	"context"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/authidentity"
	"github.com/Wei-Shaw/sub2api/ent/authidentitychannel"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type WeChatIdentityOwnershipInput struct {
	UserID          int64
	ProviderSubject string
	ProviderKeys    []string
	Channel         string
	ChannelAppID    string
	ChannelSubject  string
}

type WeChatIdentityLookupInput struct {
	Identity     PendingAuthIdentityKey
	ProviderKeys []string
	OpenID       string
	Channel      string
	ChannelAppID string
}

func (s *AuthPendingIdentityService) FindUserByID(ctx context.Context, userID int64) (*AuthIdentityUser, error) {
	if s == nil || s.entClient == nil {
		return nil, infraerrors.ServiceUnavailable("PENDING_AUTH_NOT_READY", "pending auth service is not ready")
	}
	user, err := s.entClient.User.Get(ctx, userID)
	if dbent.IsNotFound(err) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return authIdentityUserFromEntity(user), nil
}

func (s *AuthPendingIdentityService) EnsureWeChatIdentityOwnership(ctx context.Context, input WeChatIdentityOwnershipInput) error {
	if s == nil || s.entClient == nil {
		return infraerrors.ServiceUnavailable("PENDING_AUTH_NOT_READY", "pending auth service is not ready")
	}
	identities, err := s.entClient.AuthIdentity.Query().Where(
		authidentity.ProviderTypeEQ("wechat"),
		authidentity.ProviderKeyIn(input.ProviderKeys...),
		authidentity.ProviderSubjectEQ(strings.TrimSpace(input.ProviderSubject)),
	).All(ctx)
	if err != nil {
		return infraerrors.InternalServer("WECHAT_BIND_LOOKUP_FAILED", "failed to inspect wechat identity ownership").WithCause(err)
	}
	for _, identity := range identities {
		if identity == nil || identity.UserID == input.UserID {
			continue
		}
		owner, lookupErr := findActivePendingIdentityUser(ctx, s.entClient, identity.UserID)
		if lookupErr != nil {
			return lookupErr
		}
		if owner != nil {
			return infraerrors.Conflict("AUTH_IDENTITY_OWNERSHIP_CONFLICT", "auth identity already belongs to another user")
		}
	}

	if strings.TrimSpace(input.ChannelSubject) == "" || strings.TrimSpace(input.ChannelAppID) == "" {
		return nil
	}
	channels, err := s.entClient.AuthIdentityChannel.Query().Where(
		authidentitychannel.ProviderTypeEQ("wechat"),
		authidentitychannel.ProviderKeyIn(input.ProviderKeys...),
		authidentitychannel.ChannelEQ(strings.TrimSpace(input.Channel)),
		authidentitychannel.ChannelAppIDEQ(strings.TrimSpace(input.ChannelAppID)),
		authidentitychannel.ChannelSubjectEQ(strings.TrimSpace(input.ChannelSubject)),
	).WithIdentity().All(ctx)
	if err != nil {
		return infraerrors.InternalServer("WECHAT_BIND_CHANNEL_LOOKUP_FAILED", "failed to inspect wechat identity channel ownership").WithCause(err)
	}
	for _, channel := range channels {
		if channel == nil || channel.Edges.Identity == nil || channel.Edges.Identity.UserID == input.UserID {
			continue
		}
		owner, lookupErr := findActivePendingIdentityUser(ctx, s.entClient, channel.Edges.Identity.UserID)
		if lookupErr != nil {
			return lookupErr
		}
		if owner != nil {
			return infraerrors.Conflict("AUTH_IDENTITY_CHANNEL_OWNERSHIP_CONFLICT", "auth identity channel already belongs to another user")
		}
	}
	return nil
}

func (s *AuthPendingIdentityService) FindWeChatIdentityUser(ctx context.Context, input WeChatIdentityLookupInput) (*AuthIdentityUser, error) {
	if s == nil || s.entClient == nil {
		return nil, infraerrors.ServiceUnavailable("PENDING_AUTH_NOT_READY", "pending auth service is not ready")
	}
	providerType := strings.TrimSpace(input.Identity.ProviderType)
	providerSubject := strings.TrimSpace(input.Identity.ProviderSubject)
	if providerSubject != "" {
		records, err := s.entClient.AuthIdentity.Query().Where(
			authidentity.ProviderTypeEQ(providerType),
			authidentity.ProviderKeyIn(input.ProviderKeys...),
			authidentity.ProviderSubjectEQ(providerSubject),
		).WithUser().All(ctx)
		if err != nil {
			return nil, infraerrors.InternalServer("AUTH_IDENTITY_LOOKUP_FAILED", "failed to inspect auth identity ownership").WithCause(err)
		}
		user, err := singlePendingWeChatIdentityUser(records)
		if err != nil || user != nil {
			return s.activeIdentityUserResult(ctx, user, err)
		}
	}

	openID := strings.TrimSpace(input.OpenID)
	channel := strings.TrimSpace(input.Channel)
	channelAppID := strings.TrimSpace(input.ChannelAppID)
	if openID != "" && channel != "" && channelAppID != "" {
		records, err := s.entClient.AuthIdentityChannel.Query().Where(
			authidentitychannel.ProviderTypeEQ(providerType),
			authidentitychannel.ProviderKeyIn(input.ProviderKeys...),
			authidentitychannel.ChannelEQ(channel),
			authidentitychannel.ChannelAppIDEQ(channelAppID),
			authidentitychannel.ChannelSubjectEQ(openID),
		).WithIdentity(func(query *dbent.AuthIdentityQuery) { query.WithUser() }).All(ctx)
		if err != nil {
			return nil, infraerrors.InternalServer("AUTH_IDENTITY_CHANNEL_LOOKUP_FAILED", "failed to inspect auth identity channel ownership").WithCause(err)
		}
		user, err := singlePendingWeChatChannelUser(records)
		if err != nil || user != nil {
			return s.activeIdentityUserResult(ctx, user, err)
		}
	}
	if openID == "" {
		return nil, nil
	}
	records, err := s.entClient.AuthIdentity.Query().Where(
		authidentity.ProviderTypeEQ(providerType),
		authidentity.ProviderKeyIn(input.ProviderKeys...),
		authidentity.ProviderSubjectEQ(openID),
	).WithUser().All(ctx)
	if err != nil {
		return nil, infraerrors.InternalServer("AUTH_IDENTITY_LOOKUP_FAILED", "failed to inspect auth identity ownership").WithCause(err)
	}
	user, err := singlePendingWeChatIdentityUser(records)
	return s.activeIdentityUserResult(ctx, user, err)
}

func (s *AuthPendingIdentityService) activeIdentityUserResult(ctx context.Context, user *dbent.User, err error) (*AuthIdentityUser, error) {
	if err != nil || user == nil {
		return nil, err
	}
	active, err := findActivePendingIdentityUser(ctx, s.entClient, user.ID)
	return authIdentityUserFromEntity(active), err
}

func singlePendingWeChatIdentityUser(records []*dbent.AuthIdentity) (*dbent.User, error) {
	var resolved *dbent.User
	for _, record := range records {
		if record == nil || record.Edges.User == nil {
			continue
		}
		if resolved == nil {
			resolved = record.Edges.User
			continue
		}
		if resolved.ID != record.Edges.User.ID {
			return nil, infraerrors.Conflict("AUTH_IDENTITY_OWNERSHIP_CONFLICT", "auth identity already belongs to another user")
		}
	}
	return resolved, nil
}

func singlePendingWeChatChannelUser(records []*dbent.AuthIdentityChannel) (*dbent.User, error) {
	var resolved *dbent.User
	for _, record := range records {
		if record == nil || record.Edges.Identity == nil || record.Edges.Identity.Edges.User == nil {
			continue
		}
		user := record.Edges.Identity.Edges.User
		if resolved == nil {
			resolved = user
			continue
		}
		if resolved.ID != user.ID {
			return nil, infraerrors.Conflict("AUTH_IDENTITY_CHANNEL_OWNERSHIP_CONFLICT", "auth identity channel already belongs to another user")
		}
	}
	return resolved, nil
}
