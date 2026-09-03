//go:build unit

package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOpenAISessionIDHeaderHasPriority(t *testing.T) {
	gin.SetMode(gin.TestMode)
	req := httptest.NewRequest("POST", "/v1/responses", nil)
	req.Header.Set("session-id", "codex-session")
	req.Header.Set("session_id", "legacy-session")
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req
	service := &OpenAIGatewayService{}
	require.Equal(t, "codex-session", service.ExtractSessionID(c, nil))
}

func TestOpenAIWSSessionIDHeaderHasPriority(t *testing.T) {
	req := httptest.NewRequest("POST", "/v1/responses", nil)
	req.Header.Set("session-id", "codex-session")
	req.Header.Set("session_id", "legacy-session")
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req
	resolved := resolveOpenAIWSSessionHeaders(c, "")
	require.Equal(t, "codex-session", resolved.SessionID)
	require.Equal(t, "header_session-id", resolved.SessionSource)
}

func TestOpenAICompactSessionIDHeaderHasPriority(t *testing.T) {
	req := httptest.NewRequest("POST", "/v1/responses/compact", nil)
	req.Header.Set("session-id", "codex-session")
	req.Header.Set("session_id", "legacy-session")
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req
	require.Equal(t, "codex-session", resolveOpenAICompactSessionID(c))
}

func TestOpenAIBuildUpstreamRequestIsolatesHyphenatedSessionID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"gpt-5.4","input":"hello"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	req.Header.Set("session-id", "codex-session")
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req

	service := &OpenAIGatewayService{}
	upstreamReq, err := service.buildUpstreamRequest(
		context.Background(), c,
		&Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth},
		body, "oauth-token", false, "", false,
	)
	require.NoError(t, err)
	require.Empty(t, upstreamReq.Header.Get("session-id"), "raw client spelling must not escape to OAuth upstream")
	require.Equal(t, isolateOpenAISessionID(0, "codex-session"), upstreamReq.Header.Get("session_id"))
}

func TestOpenAIBuildUpstreamRequestCompactSessionSeedNotOverriddenByPromptCacheKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	req := httptest.NewRequest(http.MethodPost, "/v1/responses/compact", nil)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req
	c.Set(openAICompactSessionSeedKey, "compact-seed")

	service := &OpenAIGatewayService{}
	upstreamReq, err := service.buildUpstreamRequest(
		context.Background(), c,
		&Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth},
		[]byte(`{"model":"gpt-5.4","input":"hello"}`), "oauth-token", false, "prompt-cache-fallback", false,
	)
	require.NoError(t, err)
	require.Equal(t, isolateOpenAISessionID(0, "compact-seed"), upstreamReq.Header.Get("session_id"))
	require.NotEqual(t, isolateOpenAISessionID(0, "prompt-cache-fallback"), upstreamReq.Header.Get("session_id"))
}
