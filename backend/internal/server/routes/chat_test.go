package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newChatRoutesRequestIDTestRouter(capturedClientRequestIDs *[]string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(servermiddleware.RequestLogger())
	router.Use(func(c *gin.Context) {
		c.Next()
		if clientRequestID, _ := c.Request.Context().Value(ctxkey.ClientRequestID).(string); clientRequestID != "" {
			*capturedClientRequestIDs = append(*capturedClientRequestIDs, clientRequestID)
		}
	})
	v1 := router.Group("/api/v1")
	RegisterChatRoutes(
		v1,
		&handler.Handlers{Chat: handler.NewChatHandler(nil, nil, nil, nil)},
		servermiddleware.JWTAuthMiddleware(func(c *gin.Context) {
			c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: 42})
			c.Next()
		}),
		nil,
		nil,
		&config.Config{Gateway: config.GatewayConfig{MaxBodySize: 4 << 20}},
	)
	return router
}

func TestChatRoutesGenerateUniqueClientRequestIDForRepeatedXRequestID(t *testing.T) {
	capturedClientRequestIDs := make([]string, 0, 2)
	router := newChatRoutesRequestIDTestRouter(&capturedClientRequestIDs)
	clientRequestIDs := make([]string, 0, 2)

	for range 2 {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/chat/models", nil)
		req.Header.Set("X-Request-ID", "attacker-reused-request-id")
		recorder := httptest.NewRecorder()

		router.ServeHTTP(recorder, req)

		require.Equal(t, "attacker-reused-request-id", recorder.Header().Get("X-Request-ID"))
		clientRequestID := recorder.Header().Get("X-Client-Request-ID")
		require.NotEmpty(t, clientRequestID)
		clientRequestIDs = append(clientRequestIDs, clientRequestID)
	}

	require.Equal(t, clientRequestIDs, capturedClientRequestIDs)
	require.NotEqual(t, clientRequestIDs[0], clientRequestIDs[1])
}

func TestChatHistoryRoutesAreRegistered(t *testing.T) {
	router := newChatRoutesRequestIDTestRouter(&[]string{})
	registered := make(map[string]struct{})
	for _, route := range router.Routes() {
		registered[route.Method+" "+route.Path] = struct{}{}
	}

	for _, expected := range []string{
		http.MethodGet + " /api/v1/chat/capabilities",
		http.MethodPost + " /api/v1/chat/conversations",
		http.MethodGet + " /api/v1/chat/conversations",
		http.MethodPost + " /api/v1/chat/conversations/search",
		http.MethodGet + " /api/v1/chat/conversations/:conversation_id",
		http.MethodPatch + " /api/v1/chat/conversations/:conversation_id",
		http.MethodDelete + " /api/v1/chat/conversations/:conversation_id",
		http.MethodGet + " /api/v1/chat/conversations/:conversation_id/messages",
		http.MethodGet + " /api/v1/chat/sync",
		http.MethodGet + " /api/v1/chat/attempts/:attempt_id",
		http.MethodPost + " /api/v1/chat/transcriptions",
		http.MethodPost + " /api/v1/chat/attachments",
		http.MethodDelete + " /api/v1/chat/attachments/:id",
		http.MethodGet + " /api/v1/chat/attachments/:id/content",
	} {
		_, ok := registered[expected]
		require.True(t, ok, "missing route %s", expected)
	}
}

func TestChatCapabilitiesRouteIsJWTProtected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authCalls := 0
	router := gin.New()
	v1 := router.Group("/api/v1")
	RegisterChatRoutes(
		v1,
		&handler.Handlers{Chat: handler.NewChatHandler(nil, nil, nil, nil)},
		servermiddleware.JWTAuthMiddleware(func(c *gin.Context) {
			authCalls++
			c.AbortWithStatus(http.StatusUnauthorized)
		}),
		nil,
		nil,
		&config.Config{Gateway: config.GatewayConfig{MaxBodySize: 4 << 20}},
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/chat/capabilities", nil)
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.Equal(t, 1, authCalls)
}
