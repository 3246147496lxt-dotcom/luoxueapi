package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

const chatPrincipalCreateAttempts = 3

type ChatPrincipalRepository interface {
	GetOrCreateWebChatPrincipal(ctx context.Context, userID, groupID int64, generatedKey string) (*APIKey, error)
}

type ChatPrincipalResolver struct {
	repo ChatPrincipalRepository
	cfg  *config.Config
}

func NewChatPrincipalResolver(repo ChatPrincipalRepository, cfg *config.Config) *ChatPrincipalResolver {
	return &ChatPrincipalResolver{repo: repo, cfg: cfg}
}

func (r *ChatPrincipalResolver) Resolve(
	ctx context.Context,
	userID int64,
	group *Group,
	subscription *UserSubscription,
) (*APIKey, error) {
	if r == nil || r.repo == nil {
		return nil, fmt.Errorf("chat principal repository is unavailable")
	}
	if userID <= 0 || group == nil || group.ID <= 0 || !group.IsActive() ||
		group.Platform != PlatformOpenAI || !chatSubscriptionMatchesGroup(subscription, userID, group) {
		return nil, fmt.Errorf("invalid web chat principal scope")
	}

	for attempt := 0; attempt < chatPrincipalCreateAttempts; attempt++ {
		key, err := r.generateKey()
		if err != nil {
			return nil, err
		}
		principal, err := r.repo.GetOrCreateWebChatPrincipal(ctx, userID, group.ID, key)
		if err == nil {
			if err := validateResolvedChatPrincipal(principal, userID, group.ID, subscription); err != nil {
				return nil, err
			}
			return principal, nil
		}
		if !errors.Is(err, ErrAPIKeyExists) {
			return nil, fmt.Errorf("resolve web chat principal: %w", err)
		}
	}
	return nil, fmt.Errorf("resolve web chat principal: %w", ErrAPIKeyExists)
}

func (r *ChatPrincipalResolver) generateKey() (string, error) {
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("generate web chat key: %w", err)
	}
	prefix := "sk-"
	if r != nil && r.cfg != nil && strings.TrimSpace(r.cfg.Default.APIKeyPrefix) != "" {
		prefix = strings.TrimSpace(r.cfg.Default.APIKeyPrefix)
	}
	return prefix + hex.EncodeToString(random), nil
}

func validateResolvedChatPrincipal(
	principal *APIKey,
	userID, groupID int64,
	subscription *UserSubscription,
) error {
	if principal == nil || principal.ID <= 0 || principal.UserID != userID ||
		principal.GroupID == nil || *principal.GroupID != groupID ||
		principal.Purpose != APIKeyPurposeWebChat || !principal.IsActive() ||
		principal.User == nil || principal.User.ID != userID ||
		principal.Group == nil || principal.Group.ID != groupID ||
		!principal.User.IsActive() ||
		(!principal.Group.IsSubscriptionType() && !principal.User.CanBindGroup(principal.Group.ID, principal.Group.IsExclusive)) ||
		!IsGroupContextValid(principal.Group) || !principal.Group.IsActive() ||
		principal.Group.Platform != PlatformOpenAI ||
		!chatSubscriptionMatchesGroup(subscription, userID, principal.Group) {
		return fmt.Errorf("resolved web chat principal is incomplete")
	}
	return nil
}

func chatSubscriptionMatchesGroup(subscription *UserSubscription, userID int64, group *Group) bool {
	if group == nil {
		return false
	}
	if !group.IsSubscriptionType() {
		return subscription == nil
	}
	return subscription != nil &&
		subscription.UserID == userID &&
		subscription.GroupID == group.ID &&
		subscription.IsActive()
}
