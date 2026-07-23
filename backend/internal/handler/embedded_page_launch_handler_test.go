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
