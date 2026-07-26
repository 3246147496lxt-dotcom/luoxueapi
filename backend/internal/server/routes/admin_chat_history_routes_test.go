package routes

import (
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAdminChatHistoryRoutesAreRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")
	handlers := &handler.Handlers{
		Admin: &handler.AdminHandlers{
			ChatHistory: adminhandler.NewAdminChatHistoryHandler(
				service.NewAdminChatHistoryService(nil),
			),
		},
	}
	passThrough := func(c *gin.Context) { c.Next() }
	require.NotPanics(t, func() {
		RegisterAdminRoutes(
			v1,
			handlers,
			middleware.AdminAuthMiddleware(passThrough),
			middleware.AuditLogMiddleware(passThrough),
			middleware.StepUpAuthMiddleware(passThrough),
			nil,
		)
	})

	registered := make(map[string]struct{})
	for _, route := range router.Routes() {
		registered[route.Method+" "+route.Path] = struct{}{}
	}
	for _, expected := range []string{
		http.MethodGet + " /api/v1/admin/users/:id/chat/conversations",
		http.MethodGet + " /api/v1/admin/users/:id/chat/conversations/:conversation_id",
	} {
		_, ok := registered[expected]
		require.True(t, ok, "missing route %s", expected)
	}
}
