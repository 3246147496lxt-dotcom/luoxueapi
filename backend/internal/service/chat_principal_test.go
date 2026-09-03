package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type chatPrincipalRepositoryStub struct {
	principal *APIKey
	err       error
	calls     int
	userID    int64
	groupID   int64
	key       string
}

func (s *chatPrincipalRepositoryStub) GetOrCreateWebChatPrincipal(
	_ context.Context,
	userID, groupID int64,
	key string,
) (*APIKey, error) {
	s.calls++
	s.userID = userID
	s.groupID = groupID
	s.key = key
	return s.principal, s.err
}

func newResolvedChatPrincipalFixture(subscriptionType string) (*User, *Group, *APIKey) {
	user := &User{ID: 7, Status: StatusActive}
	group := &Group{
		ID:               20,
		Platform:         PlatformOpenAI,
		Status:           StatusActive,
		SubscriptionType: subscriptionType,
		Hydrated:         true,
	}
	principal := &APIKey{
		ID:      99,
		UserID:  user.ID,
		GroupID: &group.ID,
		Status:  StatusActive,
		Purpose: APIKeyPurposeWebChat,
		User:    user,
		Group:   group,
	}
	return user, group, principal
}

func TestChatPrincipalResolverAcceptsMatchingSubscriptionPrincipal(t *testing.T) {
	user, group, principal := newResolvedChatPrincipalFixture(SubscriptionTypeSubscription)
	subscription := &UserSubscription{
		ID:        700,
		UserID:    user.ID,
		GroupID:   group.ID,
		Status:    SubscriptionStatusActive,
		StartsAt:  time.Now().Add(-time.Hour),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	repo := &chatPrincipalRepositoryStub{principal: principal}
	resolver := NewChatPrincipalResolver(repo, nil)

	got, err := resolver.Resolve(context.Background(), user.ID, group, subscription)
	require.NoError(t, err)
	require.Same(t, principal, got)
	require.Equal(t, 1, repo.calls)
	require.Equal(t, user.ID, repo.userID)
	require.Equal(t, group.ID, repo.groupID)
	require.True(t, strings.HasPrefix(repo.key, "sk-"))
}

func TestChatPrincipalResolverRejectsMissingOrMismatchedSubscription(t *testing.T) {
	user, group, principal := newResolvedChatPrincipalFixture(SubscriptionTypeSubscription)
	valid := &UserSubscription{
		ID:        700,
		UserID:    user.ID,
		GroupID:   group.ID,
		Status:    SubscriptionStatusActive,
		StartsAt:  time.Now().Add(-time.Hour),
		ExpiresAt: time.Now().Add(time.Hour),
	}

	for _, tt := range []struct {
		name         string
		subscription *UserSubscription
	}{
		{name: "missing"},
		{name: "wrong user", subscription: func() *UserSubscription {
			copy := *valid
			copy.UserID = 99
			return &copy
		}()},
		{name: "wrong group", subscription: func() *UserSubscription {
			copy := *valid
			copy.GroupID = 99
			return &copy
		}()},
		{name: "expired", subscription: func() *UserSubscription {
			copy := *valid
			copy.ExpiresAt = time.Now().Add(-time.Minute)
			return &copy
		}()},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repo := &chatPrincipalRepositoryStub{principal: principal}
			resolver := NewChatPrincipalResolver(repo, nil)

			got, err := resolver.Resolve(context.Background(), user.ID, group, tt.subscription)
			require.Error(t, err)
			require.Nil(t, got)
			require.Zero(t, repo.calls, "untrusted subscription must be rejected before principal creation")
		})
	}
}

func TestChatPrincipalResolverPreservesWalletPrincipalPath(t *testing.T) {
	user, group, principal := newResolvedChatPrincipalFixture(SubscriptionTypeStandard)
	repo := &chatPrincipalRepositoryStub{principal: principal}
	resolver := NewChatPrincipalResolver(repo, nil)

	got, err := resolver.Resolve(context.Background(), user.ID, group, nil)
	require.NoError(t, err)
	require.Same(t, principal, got)
	require.Equal(t, 1, repo.calls)
}
