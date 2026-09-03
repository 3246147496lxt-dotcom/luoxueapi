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
)

// A standard /responses stream can report usage in response.failed after some
// output has already reached the client.  Forward must return that partial
// result together with the non-failover error so the handler can bill it.
func TestOpenAIGatewayService_ForwardPreservesStandardResponsesPartialUsage(t *testing.T) {
	setGinTestMode()
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"text/event-stream"},
			"X-Request-Id": []string{"rid-responses-partial"},
		},
		Body: io.NopCloser(strings.NewReader(strings.Join([]string{
			"event: response.created",
			`data: {"type":"response.created","response":{"id":"resp-partial"}}`,
			"",
			"event: response.output_text.delta",
			`data: {"type":"response.output_text.delta","response_id":"resp-partial","delta":"partial"}`,
			"",
			"event: response.failed",
			`data: {"type":"response.failed","response":{"id":"resp-partial","status":"failed","usage":{"input_tokens":17,"output_tokens":5,"input_tokens_details":{"cached_tokens":3}},"error":{"code":"server_error","message":"upstream failed after output"}}}`,
			"",
		}, "\n"))),
	}}
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	svc := &OpenAIGatewayService{cfg: cfg, httpUpstream: upstream}
	account := &Account{
		ID: 9101, Name: "responses-partial", Platform: PlatformOpenAI,
		Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{"access_token": "oauth-token", "chatgpt_account_id": "chatgpt-account"},
	}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(`{"model":"gpt-5.5","stream":true,"input":"hello"}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	result, err := svc.Forward(context.Background(), c, account, body)
	require.Error(t, err)
	require.Contains(t, err.Error(), "upstream response failed")
	require.NotNil(t, result)
	require.Equal(t, "rid-responses-partial", result.RequestID)
	require.Equal(t, "resp-partial", result.ResponseID)
	require.Equal(t, 17, result.Usage.InputTokens)
	require.Equal(t, 5, result.Usage.OutputTokens)
	require.Equal(t, 3, result.Usage.CacheReadInputTokens)
	require.True(t, result.HasObservedUsage())
	require.Equal(t, "gpt-5.5", result.UpstreamModel)
	require.Equal(t, "text/event-stream", result.ResponseHeaders.Get("Content-Type"))
	require.Contains(t, recorder.Body.String(), "response.failed")
}

// A pre-output failover must keep the historical nil-result invariant: a
// later account may complete the replay and only that final attempt is billed.
func TestOpenAIGatewayService_ForwardDropsPartialResultForFailover(t *testing.T) {
	setGinTestMode()
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body: io.NopCloser(strings.NewReader(strings.Join([]string{
			"event: response.created",
			`data: {"type":"response.created","response":{"id":"resp-failover"}}`,
			"",
			"event: response.failed",
			`data: {"type":"response.failed","response":{"id":"resp-failover","status":"failed","error":{"code":"server_is_overloaded","message":"retry later"}}}`,
			"",
		}, "\n"))),
	}}
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	svc := &OpenAIGatewayService{cfg: cfg, httpUpstream: upstream}
	account := &Account{
		ID: 9102, Name: "responses-failover", Platform: PlatformOpenAI,
		Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{"access_token": "oauth-token", "chatgpt_account_id": "chatgpt-account"},
	}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(`{"model":"gpt-5.5","stream":true,"input":"hello"}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))

	result, err := svc.Forward(context.Background(), c, account, body)
	require.Error(t, err)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Nil(t, result)
}

// Automatic passthrough uses a different streaming reader from managed
// Responses. It must preserve the same partial-usage contract when the
// provider fails after client-visible output.
func TestOpenAIGatewayService_ForwardPreservesPassthroughResponsesPartialUsage(t *testing.T) {
	setGinTestMode()
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"text/event-stream"},
			"X-Request-Id": []string{"rid-passthrough-partial"},
		},
		Body: io.NopCloser(strings.NewReader(strings.Join([]string{
			"event: response.output_text.delta",
			`data: {"type":"response.output_text.delta","response_id":"resp-passthrough-partial","delta":"partial"}`,
			"",
			"event: response.failed",
			`data: {"type":"response.failed","response":{"id":"resp-passthrough-partial","status":"failed","usage":{"input_tokens":23,"output_tokens":7,"input_tokens_details":{"cached_tokens":4}},"error":{"code":"server_error","message":"upstream failed after output"}}}`,
			"",
		}, "\n"))),
	}}
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	svc := &OpenAIGatewayService{cfg: cfg, httpUpstream: upstream}
	account := &Account{
		ID: 9103, Name: "responses-passthrough-partial", Platform: PlatformOpenAI,
		Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{"access_token": "oauth-token", "chatgpt_account_id": "chatgpt-account"},
		Extra:       map[string]any{"openai_passthrough": true},
	}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(`{"model":"gpt-5.5","stream":true,"instructions":"Be helpful.","input":"hello"}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	result, err := svc.Forward(context.Background(), c, account, body)
	require.Error(t, err)
	require.Contains(t, err.Error(), "upstream response failed")
	require.NotNil(t, result)
	require.Equal(t, "rid-passthrough-partial", result.RequestID)
	require.Equal(t, "resp-passthrough-partial", result.ResponseID)
	require.Equal(t, 23, result.Usage.InputTokens)
	require.Equal(t, 7, result.Usage.OutputTokens)
	require.Equal(t, 4, result.Usage.CacheReadInputTokens)
	require.True(t, result.HasObservedUsage())
	require.Equal(t, "text/event-stream", result.ResponseHeaders.Get("Content-Type"))
	require.Contains(t, recorder.Body.String(), "response.failed")
}

func TestOpenAIGatewayService_ForwardDropsPassthroughPartialResultForFailover(t *testing.T) {
	setGinTestMode()
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body: io.NopCloser(strings.NewReader(strings.Join([]string{
			"event: response.created",
			`data: {"type":"response.created","response":{"id":"resp-passthrough-failover"}}`,
			"",
			"event: response.failed",
			`data: {"type":"response.failed","response":{"id":"resp-passthrough-failover","status":"failed","error":{"code":"server_is_overloaded","message":"retry later"}}}`,
			"",
		}, "\n"))),
	}}
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	svc := &OpenAIGatewayService{cfg: cfg, httpUpstream: upstream}
	account := &Account{
		ID: 9104, Name: "responses-passthrough-failover", Platform: PlatformOpenAI,
		Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{"access_token": "oauth-token", "chatgpt_account_id": "chatgpt-account"},
		Extra:       map[string]any{"openai_passthrough": true},
	}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(`{"model":"gpt-5.5","stream":true,"instructions":"Be helpful.","input":"hello"}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))

	result, err := svc.Forward(context.Background(), c, account, body)
	require.Error(t, err)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Nil(t, result)
}

func TestParseSSEUsageBytesAcceptsCompactFailedEvent(t *testing.T) {
	// This valid terminal event is intentionally shorter than the old 72-byte
	// heuristic. Compact providers must not lose observed failure usage.
	payload := []byte(`{"type":"response.failed","usage":{"input_tokens":1}}`)
	require.Less(t, len(payload), 72)

	usage := &OpenAIUsage{}
	(&OpenAIGatewayService{}).parseSSEUsageBytes(payload, usage)

	require.Equal(t, 1, usage.InputTokens)
}
