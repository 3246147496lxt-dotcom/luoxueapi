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
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func deepSeekResponsesClientToolsTestAccount(protocol string) *Account {
	return &Account{
		ID:          7801,
		Name:        "deepseek-client-tools",
		Platform:    PlatformDeepseek,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":      "sk-deepseek-test",
			"api_protocol": protocol,
			"base_url":     "http://upstream.example",
		},
		Schedulable: true,
	}
}

func deepSeekResponsesClientToolsTestConfig() *config.Config {
	return &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{
		Enabled:           false,
		AllowInsecureHTTP: true,
	}}}
}

func deepSeekResponsesClientToolsContext(body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	setGinTestMode()
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	return c, recorder
}

func TestDeepSeekResponsesArbitraryCustomToolFallsBackToChatAndRestoresCall(t *testing.T) {
	body := []byte(`{"model":"deepseek-v4-pro","input":"run pwd","stream":false,"tools":[{"type":"custom","name":"exec","description":"Run a command"}]}`)
	c, recorder := deepSeekResponsesClientToolsContext(body)
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"chatcmpl_custom","object":"chat.completion","model":"deepseek-v4-pro","choices":[{"index":0,"message":{"role":"assistant","tool_calls":[{"id":"call_exec","type":"function","function":{"name":"exec","arguments":"{\"input\":\"pwd\"}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":3,"completion_tokens":2,"total_tokens":5}}`)),
	}}
	svc := &OpenAIGatewayService{cfg: deepSeekResponsesClientToolsTestConfig(), httpUpstream: upstream}

	result, err := svc.Forward(context.Background(), c, deepSeekResponsesClientToolsTestAccount(APIProtocolResponses), body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "http://upstream.example/chat/completions", upstream.lastReq.URL.String())
	require.Equal(t, "function", gjson.GetBytes(upstream.lastBody, "tools.0.type").String())
	require.Equal(t, "exec", gjson.GetBytes(upstream.lastBody, "tools.0.function.name").String())
	require.Equal(t, "custom_tool_call", gjson.Get(recorder.Body.String(), "output.0.type").String())
	require.Equal(t, "exec", gjson.Get(recorder.Body.String(), "output.0.name").String())
	require.Equal(t, "pwd", gjson.Get(recorder.Body.String(), "output.0.input").String())
}

func TestDeepSeekResponsesToolSearchFallsBackToChatAndRestoresClientCall(t *testing.T) {
	body := []byte(`{"model":"deepseek-v4-pro","input":"find tools","stream":false,"tools":[{"type":"tool_search"}]}`)
	c, recorder := deepSeekResponsesClientToolsContext(body)
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"chatcmpl_search","object":"chat.completion","model":"deepseek-v4-pro","choices":[{"index":0,"message":{"role":"assistant","tool_calls":[{"id":"call_search","type":"function","function":{"name":"tool_search","arguments":"{\"query\":\"github\"}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":3,"completion_tokens":2,"total_tokens":4}}`)),
	}}
	svc := &OpenAIGatewayService{cfg: deepSeekResponsesClientToolsTestConfig(), httpUpstream: upstream}

	result, err := svc.Forward(context.Background(), c, deepSeekResponsesClientToolsTestAccount(APIProtocolResponses), body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "http://upstream.example/chat/completions", upstream.lastReq.URL.String())
	require.Equal(t, "function", gjson.GetBytes(upstream.lastBody, "tools.0.type").String())
	require.Equal(t, "tool_search", gjson.GetBytes(upstream.lastBody, "tools.0.function.name").String())
	require.Equal(t, "tool_search_call", gjson.Get(recorder.Body.String(), "output.0.type").String())
	require.Equal(t, "client", gjson.Get(recorder.Body.String(), "output.0.execution").String())
	require.Equal(t, "github", gjson.Get(recorder.Body.String(), "output.0.arguments.query").String())
}

func TestDeepSeekResponsesApplyPatchStaysOnNativeResponsesEndpoint(t *testing.T) {
	body := []byte(`{"model":"deepseek-v4-pro","input":"apply patch","stream":false,"tools":[{"type":"custom","name":"apply_patch"}]}`)
	c, recorder := deepSeekResponsesClientToolsContext(body)
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"resp_patch","object":"response","model":"deepseek-v4-pro","status":"completed","output":[{"type":"custom_tool_call","id":"ct_1","call_id":"call_patch","name":"apply_patch","input":"*** Begin Patch"}],"usage":{"input_tokens":3,"output_tokens":2}}`)),
	}}
	svc := &OpenAIGatewayService{cfg: deepSeekResponsesClientToolsTestConfig(), httpUpstream: upstream}

	result, err := svc.Forward(context.Background(), c, deepSeekResponsesClientToolsTestAccount(APIProtocolResponses), body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "http://upstream.example/responses", upstream.lastReq.URL.String())
	require.Equal(t, "custom", gjson.GetBytes(upstream.lastBody, "tools.0.type").String())
	require.Equal(t, "apply_patch", gjson.GetBytes(upstream.lastBody, "tools.0.name").String())
	require.False(t, gjson.GetBytes(upstream.lastBody, "previous_response_id").Exists())
	require.False(t, gjson.GetBytes(upstream.lastBody, "store").Bool())
	require.Equal(t, "custom_tool_call", gjson.Get(recorder.Body.String(), "output.0.type").String())
	require.Equal(t, "*** Begin Patch", gjson.Get(recorder.Body.String(), "output.0.input").String())
}
