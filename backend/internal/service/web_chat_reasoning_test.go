package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai_compat"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestWebChatReasoningOptionsContextCannotBeSpoofedByHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/chat/completions", nil)
	c.Request.Header.Set("X-Web-Chat-Reasoning-Mode", WebChatReasoningModePro)

	_, ok := GetWebChatReasoningOptions(c)
	require.False(t, ok)

	want := WebChatReasoningOptions{Mode: WebChatReasoningModeStandard, Effort: "high"}
	SetWebChatReasoningOptions(c, want)
	got, ok := GetWebChatReasoningOptions(c)
	require.True(t, ok)
	require.Equal(t, want, got)
}

func TestAccountSupportsWebChatReasoningFailsClosedForAPIKeyResponsesState(t *testing.T) {
	options := WebChatReasoningOptions{Mode: WebChatReasoningModeStandard, Effort: "medium"}
	base := Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey}

	tests := []struct {
		name  string
		extra map[string]any
		want  bool
	}{
		{name: "unknown is rejected"},
		{name: "probe yes", extra: map[string]any{openai_compat.ExtraKeyResponsesSupported: true}, want: true},
		{name: "probe no", extra: map[string]any{openai_compat.ExtraKeyResponsesSupported: false}},
		{name: "force responses", extra: map[string]any{openai_compat.ExtraKeyResponsesMode: string(openai_compat.ResponsesSupportModeForceResponses)}, want: true},
		{name: "force chat completions", extra: map[string]any{openai_compat.ExtraKeyResponsesMode: string(openai_compat.ResponsesSupportModeForceChatCompletions), openai_compat.ExtraKeyResponsesSupported: true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := base
			account.Extra = tt.extra
			require.Equal(t, tt.want, AccountSupportsWebChatReasoning(&account, "gpt-5.6-sol", options))
		})
	}
}

func TestAccountSupportsWebChatReasoningKeepsModeAndEffortIndependent(t *testing.T) {
	account := &Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth}

	require.True(t, AccountSupportsWebChatReasoning(account, "gpt-5.6-sol", WebChatReasoningOptions{
		Mode: WebChatReasoningModeStandard, Effort: "low",
	}))
	require.True(t, AccountSupportsWebChatReasoning(account, "gpt-5.6-sol", WebChatReasoningOptions{
		Mode: WebChatReasoningModePro, Effort: "medium",
	}))
	require.True(t, AccountSupportsWebChatReasoning(account, "gpt-5.6-sol", WebChatReasoningOptions{
		Mode: WebChatReasoningModePro, Effort: "low",
	}), "Pro mode and effort are independent")
	require.True(t, AccountSupportsWebChatReasoning(account, "gpt-5.6-terra", WebChatReasoningOptions{
		Mode: WebChatReasoningModePro, Effort: "medium",
	}), "Pro mode works with every GPT-5.6 family model")
	require.False(t, AccountSupportsWebChatReasoning(account, "gpt-5.5", WebChatReasoningOptions{
		Mode: WebChatReasoningModePro, Effort: "medium",
	}), "GPT-5.5 must not inherit GPT-5.6 Pro capability")
	require.False(t, AccountSupportsWebChatReasoning(account, "gpt-5.6-sol-unverified", WebChatReasoningOptions{
		Mode: WebChatReasoningModePro, Effort: "medium",
	}), "unknown model variants must not inherit Pro capability")
}

func TestAccountSupportsWebChatReasoningForModelAppliesAccountMapping(t *testing.T) {
	svc := &OpenAIGatewayService{}
	account := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"model_mapping": map[string]any{"private-pro": "gpt-5.6-luna"},
		},
		Extra: map[string]any{openai_compat.ExtraKeyResponsesSupported: true},
	}

	require.True(t, svc.AccountSupportsWebChatReasoningForModel(
		account,
		"private-pro",
		WebChatReasoningOptions{Mode: WebChatReasoningModePro, Effort: "max"},
	))
}
