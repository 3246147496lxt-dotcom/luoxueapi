package middleware

import (
	"context"
	"net/http/httptest"
	"testing"

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
