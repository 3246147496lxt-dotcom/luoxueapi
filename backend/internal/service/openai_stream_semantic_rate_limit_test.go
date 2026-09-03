//go:build unit

package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// OpenAI's Responses endpoint can carry a terminal rate-limit error in an
// otherwise successful (HTTP 200) SSE stream.  Include an output delta first
// so these tests exercise the post-commit path, where account failover is no
// longer safe but account health still must be updated.
func semanticRateLimitAfterOutputSSE() string {
	return strings.Join([]string{
		"event: response.created",
		`data: {"type":"response.created","response":{"id":"resp-rate-limit","status":"in_progress"}}`,
		"",
		"event: response.output_text.delta",
		`data: {"type":"response.output_text.delta","response_id":"resp-rate-limit","output_index":0,"content_index":0,"delta":"partial"}`,
		"",
		"event: response.failed",
		`data: {"type":"response.failed","response":{"id":"resp-rate-limit","status":"failed","error":{"type":"invalid_request_error","code":"rate_limit_exceeded","message":"quota exhausted"},"usage":{"input_tokens":3,"output_tokens":1}}}`,
		"",
	}, "\n")
}

func newSemanticRateLimitStreamResponse() *http.Response {
	// These are a normal successful-stream quota snapshot.  They deliberately
	// must not become the cooldown for the semantic 429 event.
	return &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type":                        []string{"text/event-stream"},
			"X-Codex-Primary-Used-Percent":        []string{"37"},
			"X-Codex-Primary-Reset-After-Seconds": []string{"604800"},
			"X-Codex-Primary-Window-Minutes":      []string{"10080"},
		},
		Body: io.NopCloser(strings.NewReader(semanticRateLimitAfterOutputSSE())),
	}
}

func newSemanticRateLimitTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	setGinTestMode()
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	return c, recorder
}

func requireSemanticRateLimitRuntimeBlock(t *testing.T, svc *OpenAIGatewayService, account *Account) {
	t.Helper()
	value, ok := svc.openaiAccountRuntimeBlockUntil.Load(account.ID)
	require.True(t, ok, "semantic response.failed 429 must update account runtime health")
	blockedUntil, ok := value.(time.Time)
	require.True(t, ok)
	remaining := time.Until(blockedUntil)
	require.Greater(t, remaining, time.Duration(0))
	require.Less(t, remaining, time.Minute,
		"successful HTTP 200 quota snapshot must not be used as semantic 429 reset")
}

func TestOpenAIResponsesSemanticRateLimitAfterOutputUpdatesAccountHealth(t *testing.T) {
	c, recorder := newSemanticRateLimitTestContext()
	svc := &OpenAIGatewayService{}
	account := &Account{ID: 9101, Platform: PlatformOpenAI, Type: AccountTypeOAuth}

	_, err := svc.handleStreamingResponse(
		c.Request.Context(), newSemanticRateLimitStreamResponse(), c, account,
		time.Now(), "gpt-5", "gpt-5",
	)

	require.Error(t, err)
	var failoverErr *UpstreamFailoverError
	require.False(t, errors.As(err, &failoverErr), "output-started stream must not fail over")
	require.Contains(t, recorder.Body.String(), "partial")
	requireSemanticRateLimitRuntimeBlock(t, svc, account)
}

func TestOpenAIResponsesPassthroughSemanticRateLimitAfterOutputUpdatesAccountHealth(t *testing.T) {
	c, recorder := newSemanticRateLimitTestContext()
	svc := &OpenAIGatewayService{}
	account := &Account{ID: 9102, Platform: PlatformOpenAI, Type: AccountTypeOAuth}

	_, err := svc.handleStreamingResponsePassthrough(
		context.Background(), newSemanticRateLimitStreamResponse(), c, account,
		time.Now(), "gpt-5", "gpt-5",
	)

	require.Error(t, err)
	var failoverErr *UpstreamFailoverError
	require.False(t, errors.As(err, &failoverErr), "output-started stream must not fail over")
	require.Contains(t, recorder.Body.String(), "partial")
	requireSemanticRateLimitRuntimeBlock(t, svc, account)
}

func TestOpenAIMessagesSemanticRateLimitAfterOutputUpdatesAccountHealth(t *testing.T) {
	c, recorder := newSemanticRateLimitTestContext()
	svc := &OpenAIGatewayService{}
	account := &Account{ID: 9103, Platform: PlatformOpenAI, Type: AccountTypeOAuth}

	_, err := svc.handleAnthropicStreamingResponse(
		newSemanticRateLimitStreamResponse(), c, account,
		"gpt-5", "gpt-5", "gpt-5", time.Now(),
	)

	require.Error(t, err)
	var failoverErr *UpstreamFailoverError
	require.False(t, errors.As(err, &failoverErr), "output-started stream must not fail over")
	require.Contains(t, recorder.Body.String(), "partial")
	requireSemanticRateLimitRuntimeBlock(t, svc, account)
}

func TestOpenAIResponsesSemanticRateLimitBeforeOutputReturns429Failover(t *testing.T) {
	c, recorder := newSemanticRateLimitTestContext()
	svc := &OpenAIGatewayService{}
	account := &Account{
		ID:       9104,
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"pool_mode":                    true,
			"pool_mode_retry_count":        float64(1),
			"pool_mode_retry_status_codes": []any{float64(http.StatusTooManyRequests)},
		},
	}

	// Remove the output delta so the semantic terminal can safely enter the
	// failover loop. The HTTP 200 quota headers must not be interpreted as an
	// account-level cooldown for this retryable event.
	body := strings.Replace(
		semanticRateLimitAfterOutputSSE(),
		"event: response.output_text.delta\ndata: {\"type\":\"response.output_text.delta\",\"response_id\":\"resp-rate-limit\",\"output_index\":0,\"content_index\":0,\"delta\":\"partial\"}\n\n",
		"",
		1,
	)
	resp := newSemanticRateLimitStreamResponse()
	resp.Body = io.NopCloser(strings.NewReader(body))

	_, err := svc.handleStreamingResponse(c.Request.Context(), resp, c, account, time.Now(), "gpt-5", "gpt-5")
	require.Error(t, err)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Equal(t, http.StatusTooManyRequests, failoverErr.StatusCode)
	require.True(t, failoverErr.RetryableOnSameAccount)
	require.Equal(t, "rate_limit_error", gjson.GetBytes(failoverErr.ResponseBody, "error.type").String())
	require.Equal(t, "37", resp.Header.Get("X-Codex-Primary-Used-Percent"))
	require.NotNil(t, failoverErr.ResponseHeaders)
	require.Equal(t, "37", failoverErr.ResponseHeaders.Get("X-Codex-Primary-Used-Percent"))
	require.False(t, svc.isOpenAIAccountRuntimeBlocked(account))
	require.False(t, recorder.Code == http.StatusTooManyRequests, "the handler owns failover response writing")
}

func TestOpenAIChatCompletionsSemanticRateLimitAfterOutputDoesNotFailover(t *testing.T) {
	c, recorder := newSemanticRateLimitTestContext()
	svc := &OpenAIGatewayService{}
	account := &Account{ID: 9105, Platform: PlatformOpenAI, Type: AccountTypeOAuth}
	resp := newSemanticRateLimitStreamResponse()

	result, err := svc.handleChatStreamingResponse(resp, c, account, "gpt-5", "gpt-5", "gpt-5", time.Now(), 0, nil)
	require.Error(t, err)
	var failoverErr *UpstreamFailoverError
	require.False(t, errors.As(err, &failoverErr), "output-started Chat stream must not be replayed")
	require.NotNil(t, result)
	require.Equal(t, 3, result.Usage.InputTokens)
	require.Equal(t, 1, result.Usage.OutputTokens)
	require.Contains(t, recorder.Body.String(), "partial")
	requireSemanticRateLimitRuntimeBlock(t, svc, account)
}
