package middleware

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestBindChatPrincipalContextMarksWebChatRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{
		ID:               9,
		Status:           service.StatusActive,
		Platform:         service.PlatformOpenAI,
		SubscriptionType: service.SubscriptionTypeStandard,
		Hydrated:         true,
	}
	user := &service.User{ID: 42, Status: service.StatusActive, Concurrency: 3}
	principal := &service.APIKey{
		ID:      101,
		UserID:  user.ID,
		GroupID: &group.ID,
		Status:  service.StatusActive,
		Purpose: service.APIKeyPurposeWebChat,
		User:    user,
		Group:   group,
	}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("POST", "/api/v1/chat/completions", nil)
	c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ctxkey.WebChatIngress, true))

	require.True(t, BindChatPrincipalContext(c, principal))
	webChat, ok := c.Request.Context().Value(ctxkey.WebChat).(bool)
	require.True(t, ok)
	require.True(t, webChat)
	preference, ok := c.Request.Context().Value(ctxkey.OpenAIServiceTierPreference).(string)
	require.True(t, ok)
	require.Equal(t, service.ServiceTierPreferenceStandard, preference)
}

func TestBindChatPrincipalContextRejectsPrincipalOutsideWebChatIngress(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{
		ID:               9,
		Status:           service.StatusActive,
		Platform:         service.PlatformOpenAI,
		SubscriptionType: service.SubscriptionTypeStandard,
		Hydrated:         true,
	}
	user := &service.User{ID: 42, Status: service.StatusActive}
	principal := &service.APIKey{
		ID:      101,
		UserID:  user.ID,
		GroupID: &group.ID,
		Status:  service.StatusActive,
		Purpose: service.APIKeyPurposeWebChat,
		User:    user,
		Group:   group,
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)

	require.False(t, BindChatPrincipalContext(c, principal))
	_, marked := c.Request.Context().Value(ctxkey.WebChat).(bool)
	require.False(t, marked)
}

func TestBindChatPrincipalBillingContextBindsMatchingSubscription(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{
		ID:               9,
		Status:           service.StatusActive,
		Platform:         service.PlatformOpenAI,
		SubscriptionType: service.SubscriptionTypeSubscription,
		Hydrated:         true,
	}
	user := &service.User{ID: 42, Status: service.StatusActive, Concurrency: 3}
	principal := &service.APIKey{
		ID:      101,
		UserID:  user.ID,
		GroupID: &group.ID,
		Status:  service.StatusActive,
		Purpose: service.APIKeyPurposeWebChat,
		User:    user,
		Group:   group,
	}
	subscription := &service.UserSubscription{
		ID:        700,
		UserID:    user.ID,
		GroupID:   group.ID,
		Status:    service.SubscriptionStatusActive,
		StartsAt:  time.Now().Add(-time.Hour),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/api/v1/chat/completions", nil)
	c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ctxkey.WebChatIngress, true))

	require.True(t, BindChatPrincipalBillingContext(c, principal, subscription))
	bound, ok := GetSubscriptionFromContext(c)
	require.True(t, ok)
	require.Same(t, subscription, bound)
	boundGroup, ok := c.Request.Context().Value(ctxkey.Group).(*service.Group)
	require.True(t, ok)
	require.Same(t, group, boundGroup)
}

func TestBindChatPrincipalBillingContextRejectsUntrustedBillingPair(t *testing.T) {
	gin.SetMode(gin.TestMode)
	newContext := func() *gin.Context {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("POST", "/api/v1/chat/completions", nil)
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ctxkey.WebChatIngress, true))
		return c
	}
	newPrincipal := func(subscriptionType string) *service.APIKey {
		group := &service.Group{
			ID:               9,
			Status:           service.StatusActive,
			Platform:         service.PlatformOpenAI,
			SubscriptionType: subscriptionType,
			Hydrated:         true,
		}
		user := &service.User{ID: 42, Status: service.StatusActive}
		return &service.APIKey{
			ID:      101,
			UserID:  user.ID,
			GroupID: &group.ID,
			Status:  service.StatusActive,
			Purpose: service.APIKeyPurposeWebChat,
			User:    user,
			Group:   group,
		}
	}
	activeSubscription := func(userID, groupID int64) *service.UserSubscription {
		return &service.UserSubscription{
			ID:        700,
			UserID:    userID,
			GroupID:   groupID,
			Status:    service.SubscriptionStatusActive,
			StartsAt:  time.Now().Add(-time.Hour),
			ExpiresAt: time.Now().Add(time.Hour),
		}
	}

	t.Run("subscription principal without subscription", func(t *testing.T) {
		require.False(t, BindChatPrincipalContext(newContext(), newPrincipal(service.SubscriptionTypeSubscription)))
	})

	t.Run("subscription belongs to another user", func(t *testing.T) {
		principal := newPrincipal(service.SubscriptionTypeSubscription)
		subscription := activeSubscription(99, principal.Group.ID)
		require.False(t, BindChatPrincipalBillingContext(newContext(), principal, subscription))
	})

	t.Run("subscription belongs to another group", func(t *testing.T) {
		principal := newPrincipal(service.SubscriptionTypeSubscription)
		subscription := activeSubscription(principal.UserID, 99)
		require.False(t, BindChatPrincipalBillingContext(newContext(), principal, subscription))
	})

	t.Run("wallet principal carries subscription", func(t *testing.T) {
		principal := newPrincipal(service.SubscriptionTypeStandard)
		subscription := activeSubscription(principal.UserID, principal.Group.ID)
		require.False(t, BindChatPrincipalBillingContext(newContext(), principal, subscription))
	})
}
