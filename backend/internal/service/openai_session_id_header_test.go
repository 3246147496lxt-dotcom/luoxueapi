//go:build unit

package service

import (
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
