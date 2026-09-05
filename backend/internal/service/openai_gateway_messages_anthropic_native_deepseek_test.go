package service

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func deepSeekNativeAnthropicTestConfig() *config.Config {
	return &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{
		Enabled:           false,
		AllowInsecureHTTP: true,
	}}}
}

func deepSeekNativeAnthropicTestAccount() *Account {
	return &Account{
		ID:          902,
		Name:        "deepseek-native-anthropic",
		Platform:    PlatformDeepseek,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":      "sk-deepseek-test",
			"api_protocol": APIProtocolAnthropic,
			"base_url":     "http://deepseek.example/anthropic",
		},
	}
}

func newNativeAnthropicTestContext(method, path string, body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(method, path, bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("anthropic-version", "2023-06-01")
	return c, recorder
}

func TestForwardAsAnthropicDeepSeekNativeAnthropicBuffered(t *testing.T) {
	body := []byte(`{"model":"deepseek-v4-pro","max_tokens":32,"stream":false,"messages":[{"role":"user","content":"hello"}]}`)
	c, recorder := newNativeAnthropicTestContext(http.MethodPost, "/v1/messages", body)
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}, "X-Request-Id": []string{"native-rid"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"msg_1","type":"message","role":"assistant","model":"deepseek-v4-pro","content":[{"type":"text","text":"pong"}],"stop_reason":"end_turn","usage":{"input_tokens":7,"output_tokens":2}}`)),
	}}
	svc := &OpenAIGatewayService{cfg: deepSeekNativeAnthropicTestConfig(), httpUpstream: upstream}

	result, err := svc.ForwardAsAnthropic(context.Background(), c, deepSeekNativeAnthropicTestAccount(), body, "", "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "http://deepseek.example/anthropic/v1/messages", upstream.lastReq.URL.String())
	require.Equal(t, "sk-deepseek-test", upstream.lastReq.Header.Get("x-api-key"))
	require.Equal(t, "deepseek-v4-pro", gjson.GetBytes(upstream.lastBody, "model").String())
	require.Equal(t, "pong", gjson.Get(recorder.Body.String(), "content.0.text").String())
	require.Equal(t, 7, result.Usage.InputTokens)
	require.Equal(t, 2, result.Usage.OutputTokens)
	require.Equal(t, "native-rid", result.RequestID)
	require.False(t, result.Stream)
}

func TestForwardAsAnthropicDeepSeekNativeAnthropicStreaming(t *testing.T) {
	body := []byte(`{"model":"deepseek-v4-pro","max_tokens":32,"stream":true,"messages":[{"role":"user","content":"hello"}]}`)
	c, recorder := newNativeAnthropicTestContext(http.MethodPost, "/v1/messages", body)
	sse := "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_1\",\"type\":\"message\",\"role\":\"assistant\",\"model\":\"deepseek-v4-pro\",\"content\":[],\"usage\":{\"input_tokens\":7}}}\n\n" +
		"event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n" +
		"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"pong\"}}\n\n" +
		"event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":2}}\n\n" +
		"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(sse)),
	}}
	svc := &OpenAIGatewayService{cfg: deepSeekNativeAnthropicTestConfig(), httpUpstream: upstream}

	result, err := svc.ForwardAsAnthropic(context.Background(), c, deepSeekNativeAnthropicTestAccount(), body, "", "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Stream)
	require.Equal(t, 7, result.Usage.InputTokens)
	require.Equal(t, 2, result.Usage.OutputTokens)
	require.Contains(t, recorder.Body.String(), "event: message_stop")
	require.Equal(t, "http://deepseek.example/anthropic/v1/messages", upstream.lastReq.URL.String())
}

func TestForwardAsAnthropicDeepSeekNativeAnthropicRecordsOutputEffort(t *testing.T) {
	body := []byte(`{"model":"deepseek-v4-pro","max_tokens":32,"stream":false,"output_config":{"effort":"low"},"messages":[{"role":"user","content":"hello"}]}`)
	c, _ := newNativeAnthropicTestContext(http.MethodPost, "/v1/messages", body)
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"msg_1","type":"message","role":"assistant","model":"deepseek-v4-pro","content":[{"type":"text","text":"pong"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`)),
	}}
	svc := &OpenAIGatewayService{cfg: deepSeekNativeAnthropicTestConfig(), httpUpstream: upstream}
	result, err := svc.ForwardAsAnthropic(context.Background(), c, deepSeekNativeAnthropicTestAccount(), body, "", "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.ReasoningEffort)
	require.Equal(t, "low", *result.ReasoningEffort)
}

func TestForwardAsAnthropicDeepSeekNativeAnthropicAggregatesCacheUsage(t *testing.T) {
	body := []byte(`{"model":"deepseek-v4-pro","max_tokens":32,"stream":false,"messages":[{"role":"user","content":"hello"}]}`)
	c, _ := newNativeAnthropicTestContext(http.MethodPost, "/v1/messages", body)
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"msg_cache","type":"message","role":"assistant","model":"deepseek-v4-pro","content":[{"type":"text","text":"pong"}],"stop_reason":"end_turn","usage":{"input_tokens":7,"cache_creation_input_tokens":3,"cache_read_input_tokens":5,"output_tokens":2}}`)),
	}}
	svc := &OpenAIGatewayService{cfg: deepSeekNativeAnthropicTestConfig(), httpUpstream: upstream}

	result, err := svc.ForwardAsAnthropic(context.Background(), c, deepSeekNativeAnthropicTestAccount(), body, "", "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 15, result.Usage.InputTokens, "canonical input usage includes cache buckets")
	require.Equal(t, 3, result.Usage.CacheCreationInputTokens)
	require.Equal(t, 5, result.Usage.CacheReadInputTokens)
}

func TestDeepSeekAnthropicProtocolBaseURLNormalizesLegacyRoot(t *testing.T) {
	account := &Account{
		Platform: PlatformDeepseek,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_protocol": APIProtocolAnthropic,
			"base_url":     DefaultDeepseekBaseURL,
		},
	}
	require.Equal(t, DefaultDeepseekAnthropicBaseURL, account.GetAnthropicProtocolBaseURL())

	account.Credentials["base_url"] = DefaultDeepseekBaseURL + "/"
	require.Equal(t, DefaultDeepseekAnthropicBaseURL, account.GetAnthropicProtocolBaseURL())

	account.Credentials["base_url"] = "https://relay.example/deepseek"
	require.Equal(t, "https://relay.example/deepseek", account.GetAnthropicProtocolBaseURL())

	account.Credentials["base_url"] = "https://relay.example/deepseek?redirect=/"
	require.Equal(t, "https://relay.example/deepseek?redirect=/", account.GetAnthropicProtocolBaseURL())
}

// deepSeekNativeAnthropicCrossSSE is the smallest complete Anthropic stream:
// message_start, one text block, terminal message_delta, and message_stop.
// Both cross-protocol adapters consume the same upstream stream while exposing
// their respective OpenAI response shape to the caller.
const deepSeekNativeAnthropicCrossSSE = "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_cross\",\"type\":\"message\",\"role\":\"assistant\",\"model\":\"deepseek-v4-pro\",\"content\":[],\"usage\":{\"input_tokens\":7}}}\n\n" +
	"event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n" +
	"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"pong\"}}\n\n" +
	"event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n" +
	"event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":2}}\n\n" +
	"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"

func deepSeekNativeAnthropicCrossRecorder() *httpUpstreamRecorder {
	return &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}, "X-Request-Id": []string{"cross-rid"}},
		Body:       io.NopCloser(strings.NewReader(deepSeekNativeAnthropicCrossSSE)),
	}}
}

func TestForwardAsChatCompletionsDeepSeekAnthropicCrossBuffered(t *testing.T) {
	body := []byte(`{"model":"deepseek-v4-pro","messages":[{"role":"user","content":"hello"}],"stream":false}`)
	c, recorder := newNativeAnthropicTestContext(http.MethodPost, "/v1/chat/completions", body)
	upstream := deepSeekNativeAnthropicCrossRecorder()
	svc := &OpenAIGatewayService{cfg: deepSeekNativeAnthropicTestConfig(), httpUpstream: upstream}

	result, err := svc.ForwardAsChatCompletions(context.Background(), c, deepSeekNativeAnthropicTestAccount(), body, "", "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.Stream)
	require.Equal(t, "http://deepseek.example/anthropic/v1/messages", upstream.lastReq.URL.String())
	require.Equal(t, "true", gjson.GetBytes(upstream.lastBody, "stream").String())
	require.Equal(t, "pong", gjson.Get(recorder.Body.String(), "choices.0.message.content").String())
	require.Equal(t, "chat.completion", gjson.Get(recorder.Body.String(), "object").String())
	require.Equal(t, 7, result.Usage.InputTokens)
	require.Equal(t, 2, result.Usage.OutputTokens)
}

func TestForwardAsChatCompletionsDeepSeekAnthropicCrossStreaming(t *testing.T) {
	body := []byte(`{"model":"deepseek-v4-pro","messages":[{"role":"user","content":"hello"}],"stream":true}`)
	c, recorder := newNativeAnthropicTestContext(http.MethodPost, "/v1/chat/completions", body)
	upstream := deepSeekNativeAnthropicCrossRecorder()
	svc := &OpenAIGatewayService{cfg: deepSeekNativeAnthropicTestConfig(), httpUpstream: upstream}

	result, err := svc.ForwardAsChatCompletions(context.Background(), c, deepSeekNativeAnthropicTestAccount(), body, "", "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Stream)
	require.Contains(t, recorder.Body.String(), "data: [DONE]")
	require.Contains(t, recorder.Body.String(), "pong")
	require.Equal(t, "http://deepseek.example/anthropic/v1/messages", upstream.lastReq.URL.String())
}

func TestForwardDeepSeekAnthropicCrossBuffered(t *testing.T) {
	body := []byte(`{"model":"deepseek-v4-pro","input":"hello","stream":false}`)
	c, recorder := newNativeAnthropicTestContext(http.MethodPost, "/v1/responses", body)
	upstream := deepSeekNativeAnthropicCrossRecorder()
	svc := &OpenAIGatewayService{cfg: deepSeekNativeAnthropicTestConfig(), httpUpstream: upstream}

	result, err := svc.Forward(context.Background(), c, deepSeekNativeAnthropicTestAccount(), body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.Stream)
	require.Equal(t, "response", gjson.Get(recorder.Body.String(), "object").String())
	require.Equal(t, "completed", gjson.Get(recorder.Body.String(), "status").String())
	require.Equal(t, "pong", gjson.Get(recorder.Body.String(), "output.0.content.0.text").String())
	require.Equal(t, "http://deepseek.example/anthropic/v1/messages", upstream.lastReq.URL.String())
}

func TestForwardDeepSeekAnthropicCrossStreaming(t *testing.T) {
	body := []byte(`{"model":"deepseek-v4-pro","input":"hello","stream":true}`)
	c, recorder := newNativeAnthropicTestContext(http.MethodPost, "/v1/responses", body)
	upstream := deepSeekNativeAnthropicCrossRecorder()
	svc := &OpenAIGatewayService{cfg: deepSeekNativeAnthropicTestConfig(), httpUpstream: upstream}

	result, err := svc.Forward(context.Background(), c, deepSeekNativeAnthropicTestAccount(), body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Stream)
	require.Contains(t, recorder.Body.String(), "response.created")
	require.Contains(t, recorder.Body.String(), "response.completed")
	require.Equal(t, "http://deepseek.example/anthropic/v1/messages", upstream.lastReq.URL.String())
}

func TestDeepSeekAnthropicCrossRejectsUnsupportedTools(t *testing.T) {
	tests := []struct {
		name string
		path string
		body []byte
		call func(*OpenAIGatewayService, *gin.Context, *Account, []byte) (*OpenAIForwardResult, error)
	}{
		{
			name: "chat server tool",
			path: "/v1/chat/completions",
			body: []byte(`{"model":"deepseek-v4-pro","messages":[{"role":"user","content":"hello"}],"tools":[{"type":"web_search"}]}`),
			call: func(s *OpenAIGatewayService, c *gin.Context, a *Account, b []byte) (*OpenAIForwardResult, error) {
				return s.ForwardAsChatCompletions(context.Background(), c, a, b, "", "")
			},
		},
		{
			name: "responses custom tool",
			path: "/v1/responses",
			body: []byte(`{"model":"deepseek-v4-pro","input":"hello","tools":[{"type":"custom","name":"paint"}]}`),
			call: func(s *OpenAIGatewayService, c *gin.Context, a *Account, b []byte) (*OpenAIForwardResult, error) {
				return s.Forward(context.Background(), c, a, b)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, recorder := newNativeAnthropicTestContext(http.MethodPost, tc.path, tc.body)
			upstream := deepSeekNativeAnthropicCrossRecorder()
			svc := &OpenAIGatewayService{cfg: deepSeekNativeAnthropicTestConfig(), httpUpstream: upstream}
			_, err := tc.call(svc, c, deepSeekNativeAnthropicTestAccount(), tc.body)
			require.Error(t, err)
			require.Equal(t, http.StatusBadRequest, recorder.Code)
			require.Contains(t, recorder.Body.String(), "supports only function tools")
			require.Empty(t, upstream.requests, "invalid tools must not reach Chat/Responses or Anthropic upstream")
		})
	}
}

func TestAnthropicNativeLinePumpBoundsIdleRead(t *testing.T) {
	reader, writer := io.Pipe()
	scanner := bufio.NewScanner(reader)
	pump := newAnthropicNativeLinePump(scanner, 20*time.Millisecond)
	defer func() {
		_ = reader.Close()
		pump.stop()
	}()

	_, err := pump.next()
	require.ErrorIs(t, err, errAnthropicNativeStreamIdle)
	_ = writer.Close()
}

func TestDeepSeekAnthropicCrossAcceptsEventOnlyType(t *testing.T) {
	body := []byte(`{"model":"deepseek-v4-pro","messages":[{"role":"user","content":"hello"}],"stream":false}`)
	c, recorder := newNativeAnthropicTestContext(http.MethodPost, "/v1/chat/completions", body)
	eventOnlySSE := "event: message_start\ndata: {\"message\":{\"id\":\"msg_event_only\",\"type\":\"message\",\"role\":\"assistant\",\"model\":\"deepseek-v4-pro\",\"content\":[],\"usage\":{\"input_tokens\":1}}}\n\n" +
		"event: content_block_start\ndata: {\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n" +
		"event: content_block_delta\ndata: {\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"ok\"}}\n\n" +
		"event: message_delta\ndata: {\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":1}}\n\n" +
		"event: message_stop\ndata: {}\n\n"
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(eventOnlySSE)),
	}}
	svc := &OpenAIGatewayService{cfg: deepSeekNativeAnthropicTestConfig(), httpUpstream: upstream}

	result, err := svc.ForwardAsChatCompletions(context.Background(), c, deepSeekNativeAnthropicTestAccount(), body, "", "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "ok", gjson.Get(recorder.Body.String(), "choices.0.message.content").String())
}
