//go:build unit

package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func bufferedResponseFailedSSE(includeTopLevelUsage bool) string {
	usage := `"usage":{"input_tokens":17,"output_tokens":5,"input_tokens_details":{"cached_tokens":3}}`
	if includeTopLevelUsage {
		return strings.Join([]string{
			`event: response.failed`,
			`data: {"type":"response.failed","response":{"id":"resp-buffered-failed","status":"failed","error":{"type":"invalid_request_error","message":"request rejected"}},` + usage + `}`,
			"",
		}, "\n")
	}
	return strings.Join([]string{
		`event: response.failed`,
		`data: {"type":"response.failed","response":{"id":"resp-buffered-failed","status":"failed","usage":{"input_tokens":17,"output_tokens":5,"input_tokens_details":{"cached_tokens":3}},"error":{"type":"invalid_request_error","message":"request rejected"}}}`,
		"",
	}, "\n")
}

func newBufferedPartialUsageContext(path string) (*gin.Context, *httptest.ResponseRecorder) {
	setGinTestMode()
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, path, bytes.NewReader([]byte(`{}`)))
	c.Request.Header.Set("Content-Type", "application/json")
	return c, recorder
}

func TestHandleSSEToJSON_ResponseFailedPreservesObservedUsage(t *testing.T) {
	c, recorder := newBufferedPartialUsageContext("/v1/responses")
	svc := &OpenAIGatewayService{}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader("")),
	}
	body := []byte(bufferedResponseFailedSSE(false))

	result, err := svc.handleSSEToJSON(resp, c, body, "gpt-5.5", "gpt-5.5")
	require.Error(t, err)
	require.NotNil(t, result)
	require.Equal(t, 17, result.OpenAIUsage.InputTokens)
	require.Equal(t, 5, result.OpenAIUsage.OutputTokens)
	require.Equal(t, 3, result.OpenAIUsage.CacheReadInputTokens)
	require.Equal(t, "resp-buffered-failed", result.responseID)
	require.Equal(t, http.StatusBadGateway, recorder.Code)
}

func TestHandlePassthroughSSEToJSON_ResponseFailedPreservesObservedUsage(t *testing.T) {
	c, recorder := newBufferedPartialUsageContext("/v1/responses")
	svc := &OpenAIGatewayService{}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader("")),
	}
	body := []byte(bufferedResponseFailedSSE(false))

	result, err := svc.handlePassthroughSSEToJSON(resp, c, body, "gpt-5.5", "gpt-5.5")
	require.Error(t, err)
	require.NotNil(t, result)
	require.Equal(t, 17, result.OpenAIUsage.InputTokens)
	require.Equal(t, 5, result.OpenAIUsage.OutputTokens)
	require.Equal(t, 3, result.OpenAIUsage.CacheReadInputTokens)
	require.Equal(t, "resp-buffered-failed", result.responseID)
	require.Equal(t, http.StatusBadGateway, recorder.Code)
}

func TestReadOpenAICompatBufferedTerminal_UsesTopLevelTerminalUsage(t *testing.T) {
	setGinTestMode()
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	_ = c
	svc := &OpenAIGatewayService{}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(bufferedResponseFailedSSE(true))),
	}

	finalResponse, usage, _, err := svc.readOpenAICompatBufferedTerminal(resp, "test buffered", "rid")
	require.NoError(t, err)
	require.NotNil(t, finalResponse)
	require.Equal(t, "failed", finalResponse.Status)
	require.Equal(t, 17, usage.InputTokens)
	require.Equal(t, 5, usage.OutputTokens)
	require.Equal(t, 3, usage.CacheReadInputTokens)
}

func TestOpenAIGatewayService_ForwardNonStreamingResponseFailedPreservesUsage(t *testing.T) {
	setGinTestMode()
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}, "X-Request-Id": []string{"rid-buffered-forward"}},
		Body:       io.NopCloser(strings.NewReader(bufferedResponseFailedSSE(false))),
	}}
	svc := &OpenAIGatewayService{cfg: rawChatCompletionsTestConfig(), httpUpstream: upstream}
	account := rawChatCompletionsTestAccount()
	account.Extra = map[string]any{"use_responses_api": true}
	c, _ := newBufferedPartialUsageContext("/v1/responses")
	body := []byte(`{"model":"gpt-5.5","stream":false,"input":"hello"}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))

	result, err := svc.Forward(context.Background(), c, account, body)
	require.Error(t, err)
	require.NotNil(t, result)
	require.Equal(t, 17, result.Usage.InputTokens)
	require.Equal(t, 5, result.Usage.OutputTokens)
	require.Equal(t, "rid-buffered-forward", result.RequestID)
}

func TestOpenAIGatewayService_ForwardPassthroughNonStreamingResponseFailedPreservesUsage(t *testing.T) {
	setGinTestMode()
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}, "X-Request-Id": []string{"rid-buffered-passthrough"}},
		Body:       io.NopCloser(strings.NewReader(bufferedResponseFailedSSE(false))),
	}}
	svc := &OpenAIGatewayService{cfg: rawChatCompletionsTestConfig(), httpUpstream: upstream}
	account := rawChatCompletionsTestAccount()
	account.Extra = map[string]any{"openai_passthrough": true}
	c, _ := newBufferedPartialUsageContext("/v1/responses")
	body := []byte(`{"model":"gpt-5.5","stream":false,"instructions":"Be helpful.","input":"hello"}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))

	result, err := svc.Forward(context.Background(), c, account, body)
	require.Error(t, err)
	require.NotNil(t, result)
	require.Equal(t, 17, result.Usage.InputTokens)
	require.Equal(t, 5, result.Usage.OutputTokens)
	require.Equal(t, "rid-buffered-passthrough", result.RequestID)
}
