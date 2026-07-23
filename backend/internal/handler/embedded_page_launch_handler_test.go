package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestEmbeddedPageExchangeRejectsBrowserAndClearsCORSHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	w.Header().Set("Access-Control-Allow-Origin", "*")
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/embedded-pages/exchange", strings.NewReader(`{"client_id":"pay","code":"secret"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("Origin", "https://pay.example.com")

	(&EmbeddedPageLaunchHandler{}).Exchange(c)

	require.Equal(t, http.StatusForbidden, w.Code)
	require.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	require.Equal(t, "no-referrer", w.Header().Get("Referrer-Policy"))
	require.Contains(t, w.Body.String(), "EMBEDDED_PAGE_SERVER_EXCHANGE_REQUIRED")
}

func TestEmbeddedPageExchangeMalformedBodyUsesUnifiedInvalidCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/embedded-pages/exchange", strings.NewReader(`{"client_id":`))
	c.Request.Header.Set("Content-Type", "application/json")

	(&EmbeddedPageLaunchHandler{}).Exchange(c)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	require.Contains(t, w.Body.String(), "INVALID_EMBED_LAUNCH_CODE")
}

func TestEmbeddedPageSourceOriginIgnoresSpoofedForwardedHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, router := gin.CreateTestContext(w)
	require.NoError(t, router.SetTrustedProxies(nil))
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/user/custom-pages/pay/launch", nil)
	c.Request.RemoteAddr = "203.0.113.10:43100"
	c.Request.Host = "api.example.com"
	c.Request.Header.Set("X-Forwarded-For", "198.51.100.20")
	c.Request.Header.Set("X-Forwarded-Proto", "https")
	c.Request.Header.Set("X-Forwarded-Host", "attacker.example")

	require.Equal(t, "http://api.example.com", embeddedPageSourceOrigin(c))
}

func TestEmbeddedPageSourceOriginSupportsDefaultLoopbackCaddy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, router := gin.CreateTestContext(w)
	require.NoError(t, router.SetTrustedProxies(nil))
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/user/custom-pages/pay/launch", nil)
	c.Request.RemoteAddr = "127.0.0.1:43100"
	c.Request.Host = "api.example.com"
	c.Request.Header.Set("X-Forwarded-For", "198.51.100.20")
	c.Request.Header.Set("X-Forwarded-Proto", "https")
	c.Request.Header.Set("X-Forwarded-Host", "attacker.example")

	require.Equal(t, "https://api.example.com", embeddedPageSourceOrigin(c))
}

func TestEmbeddedPageSourceOriginUsesGinTrustedProxyChain(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, router := gin.CreateTestContext(w)
	require.NoError(t, router.SetTrustedProxies([]string{"10.0.0.0/8"}))
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/user/custom-pages/pay/launch", nil)
	c.Request.RemoteAddr = "10.0.0.8:43100"
	c.Request.Host = "api.example.com"
	c.Request.Header.Set("X-Forwarded-For", "198.51.100.20")
	c.Request.Header.Set("X-Forwarded-Proto", "https")

	require.Equal(t, "https://api.example.com", embeddedPageSourceOrigin(c))
}
