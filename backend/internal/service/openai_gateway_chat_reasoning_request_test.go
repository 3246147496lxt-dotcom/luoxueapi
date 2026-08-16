package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai_compat"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestForwardAsChatCompletionsReasoningModeUsesTrustedWebChatContextOnly(t *testing.T) {
	setGinTestMode()

	tests := []struct {
		name          string
		body          string
		trusted       *WebChatReasoningOptions
		wantReasoning bool
		wantMode      string
		wantEffort    string
		wantSummary   string
	}{
		{
			name: "public mode only is ignored",
			body: `{"model":"gpt-5.6-sol","messages":[{"role":"user","content":"hello"}],"reasoning_mode":"pro","stream":false}`,
		},
		{
			name:          "public forged Pro cannot alter effort-only behavior",
			body:          `{"model":"gpt-5.6-sol","messages":[{"role":"user","content":"hello"}],"reasoning_mode":"pro","reasoning_effort":"high","stream":false}`,
			wantReasoning: true,
			wantEffort:    "high",
			wantSummary:   "auto",
		},
		{
			name: "trusted Web Chat context overrides untrusted body fields",
			body: `{"model":"gpt-5.6-sol","messages":[{"role":"user","content":"hello"}],"reasoning_mode":"standard","reasoning_effort":"high","stream":false}`,
			trusted: &WebChatReasoningOptions{
				Mode: WebChatReasoningModePro, Effort: "low",
			},
			wantReasoning: true,
			wantMode:      "pro",
			wantEffort:    "",
			wantSummary:   "auto",
		},
		{
			name: "trusted standard mode preserves xhigh effort",
			body: `{"model":"gpt-5.6-sol","messages":[{"role":"user","content":"hello"}],"stream":false}`,
			trusted: &WebChatReasoningOptions{
				Mode: WebChatReasoningModeStandard, Effort: "xhigh",
			},
			wantReasoning: true,
			wantMode:      "standard",
			wantEffort:    "xhigh",
			wantSummary:   "auto",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			body := []byte(tt.body)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
			if tt.trusted != nil {
				SetWebChatReasoningOptions(c, *tt.trusted)
			}

			upstream := &httpUpstreamRecorder{resp: &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
				Body: io.NopCloser(strings.NewReader(
					`data: {"type":"response.completed","response":{"id":"resp_reasoning","object":"response","model":"gpt-5.6-sol","status":"completed","output":[{"type":"message","id":"msg_1","role":"assistant","status":"completed","content":[{"type":"output_text","text":"ok"}]}],"usage":{"input_tokens":5,"output_tokens":2,"total_tokens":7}}}` + "\n\n",
				)),
			}}
			cfg := &config.Config{}
			cfg.Security.URLAllowlist.Enabled = false
			svc := &OpenAIGatewayService{cfg: cfg, httpUpstream: upstream}
			account := &Account{
				ID:       701,
				Platform: PlatformOpenAI,
				Type:     AccountTypeAPIKey,
				Credentials: map[string]any{
					"api_key": "sk-test", "base_url": "https://upstream.example/v1",
				},
				Extra: map[string]any{openai_compat.ExtraKeyResponsesSupported: true},
			}

			result, err := svc.ForwardAsChatCompletions(context.Background(), c, account, body, "", "")
			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotNil(t, upstream.lastReq)
			require.Equal(t, "/v1/responses", upstream.lastReq.URL.Path)

			reasoning := gjson.GetBytes(upstream.lastBody, "reasoning")
			require.Equal(t, tt.wantReasoning, reasoning.Exists())
			if !tt.wantReasoning {
				return
			}
			mode := gjson.GetBytes(upstream.lastBody, "reasoning.mode")
			if tt.wantMode == "" {
				require.False(t, mode.Exists(), "public body must not supply reasoning.mode")
			} else {
				require.Equal(t, tt.wantMode, mode.String())
			}
			require.Equal(t, tt.wantEffort, gjson.GetBytes(upstream.lastBody, "reasoning.effort").String())
			require.Equal(t, tt.wantSummary, gjson.GetBytes(upstream.lastBody, "reasoning.summary").String())
		})
	}
}
