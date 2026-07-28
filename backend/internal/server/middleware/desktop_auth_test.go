package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/modules/desktop"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestDesktopAuthErrorsUseUnifiedEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.HandlerFunc(NewDesktopAuthMiddleware(nil)))
	router.GET("/desktop/me", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/desktop/me", nil)
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	var envelope response.Response
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	require.Equal(t, http.StatusUnauthorized, envelope.Code)
	require.Equal(t, "DESKTOP_AUTH_REQUIRED", envelope.Reason)
	require.Equal(t, "Desktop bearer token is required", envelope.Message)
}

func TestDesktopScopeErrorsUseUnifiedEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(desktopSubjectContextKey, desktop.DesktopSubject{Scopes: []string{desktop.ScopeProfileRead}})
		c.Next()
	})
	router.GET("/desktop/routes", RequireDesktopScope(desktop.ScopeRoutesRead), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/desktop/routes", nil)
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	var envelope response.Response
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	require.Equal(t, http.StatusForbidden, envelope.Code)
	require.Equal(t, "DESKTOP_SCOPE_FORBIDDEN", envelope.Reason)
}
