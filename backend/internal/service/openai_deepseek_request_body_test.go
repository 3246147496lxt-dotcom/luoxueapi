package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeDeepSeekResponsesRequestBody(t *testing.T) {
	tests := []struct {
		name       string
		account    *Account
		body       string
		wantStore  bool
		wantPrev   bool
		wantChange bool
	}{
		{
			name:       "native responses forces stateless fields",
			account:    &Account{Platform: "deepseek", Credentials: map[string]any{"api_protocol": "responses"}},
			body:       `{"model":"deepseek-v4-pro","store":true,"previous_response_id":"resp_old","input":"hello"}`,
			wantStore:  false,
			wantPrev:   false,
			wantChange: true,
		},
		{
			name:       "adaptive responses forces stateless fields",
			account:    &Account{Platform: "deepseek", Credentials: map[string]any{"api_protocol": "adaptive"}},
			body:       `{"model":"deepseek-v4-pro","previous_response_id":"resp_old"}`,
			wantStore:  false,
			wantPrev:   false,
			wantChange: true,
		},
		{
			name:       "missing protocol keeps chat default untouched",
			account:    &Account{Platform: "deepseek"},
			body:       `{"model":"deepseek-v4-pro","store":true,"previous_response_id":"resp_old"}`,
			wantStore:  true,
			wantPrev:   true,
			wantChange: false,
		},
		{
			name:       "explicit chat protocol is untouched",
			account:    &Account{Platform: "deepseek", Credentials: map[string]any{"api_protocol": "chat_completions"}},
			body:       `{"model":"deepseek-v4-pro","store":true,"previous_response_id":"resp_old"}`,
			wantStore:  true,
			wantPrev:   true,
			wantChange: false,
		},
		{
			name:       "other platform is untouched",
			account:    &Account{Platform: "openai", Credentials: map[string]any{"api_protocol": "responses"}},
			body:       `{"model":"gpt-5.4","store":true,"previous_response_id":"resp_old"}`,
			wantStore:  true,
			wantPrev:   true,
			wantChange: false,
		},
		{
			name:       "malformed JSON is passed through",
			account:    &Account{Platform: "deepseek"},
			body:       `{"store":true`,
			wantChange: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := []byte(tt.body)
			got := normalizeDeepSeekResponsesRequestBody(tt.account, body)
			if !tt.wantChange {
				require.Equal(t, tt.body, string(got))
				return
			}

			var decoded map[string]any
			require.NoError(t, json.Unmarshal(got, &decoded))
			store, ok := decoded["store"].(bool)
			require.True(t, ok, "normalized body must contain a boolean store field")
			require.Equal(t, tt.wantStore, store)
			_, hasPrevious := decoded["previous_response_id"]
			require.Equal(t, tt.wantPrev, hasPrevious)
		})
	}
}

func TestNormalizeDeepSeekResponsesRequestBodyNilSafety(t *testing.T) {
	require.Nil(t, normalizeDeepSeekResponsesRequestBody(nil, nil))
	require.Equal(t, []byte{}, normalizeDeepSeekResponsesRequestBody(&Account{Platform: "deepseek"}, []byte{}))
	for _, body := range []string{"null", "[]", `"scalar"`} {
		require.Equal(t, body, string(normalizeDeepSeekResponsesRequestBody(&Account{Platform: "deepseek"}, []byte(body))))
	}
}

func TestDeepSeekResponsesNeedsChatFallback(t *testing.T) {
	tests := []struct {
		name string
		body string
		want bool
	}{
		{
			name: "ordinary function stays native",
			body: `{"model":"deepseek-v4-pro","tools":[{"type":"function","name":"lookup"}]}`,
			want: false,
		},
		{
			name: "built in apply patch stays native",
			body: `{"model":"deepseek-v4-pro","tools":[{"type":"custom","name":"apply_patch"}],"input":[{"type":"custom_tool_call","call_id":"c1","name":"apply_patch","input":"*** a"},{"type":"custom_tool_call_output","call_id":"c1","output":"ok"}]}`,
			want: false,
		},
		{
			name: "arbitrary custom tool falls back",
			body: `{"model":"deepseek-v4-pro","tools":[{"type":"custom","name":"exec"}]}`,
			want: true,
		},
		{
			name: "string custom tool shorthand falls back",
			body: `{"model":"deepseek-v4-pro","tools":["exec"]}`,
			want: true,
		},
		{
			name: "string apply patch shorthand stays native",
			body: `{"model":"deepseek-v4-pro","tools":["apply_patch"]}`,
			want: false,
		},
		{
			name: "apply patch name must match exactly",
			body: `{"model":"deepseek-v4-pro","tools":[{"type":"custom","name":"APPLY_PATCH"}]}`,
			want: true,
		},
		{
			name: "tool search in additional tools falls back",
			body: `{"model":"deepseek-v4-pro","input":[{"type":"additional_tools","tools":[{"type":"tool_search"}]}]}`,
			want: true,
		},
		{
			name: "namespace declaration falls back",
			body: `{"model":"deepseek-v4-pro","tools":[{"type":"namespace","name":"team","tools":[{"type":"function","name":"send"}]}]}`,
			want: true,
		},
		{
			name: "unknown custom output falls back",
			body: `{"model":"deepseek-v4-pro","input":[{"type":"custom_tool_call_output","call_id":"c1","output":"ok"}]}`,
			want: true,
		},
		{
			name: "malformed body is left for normal validation",
			body: `{"model":"deepseek-v4-pro"`,
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, deepSeekResponsesNeedsChatFallback([]byte(tt.body)))
		})
	}
}
