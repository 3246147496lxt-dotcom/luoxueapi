package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai_compat"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type compatCyberAccountStateRepo struct {
	AccountRepository

	setErrorCalls           int
	setRateLimitedCalls     int
	setModelRateLimitCalls  int
	setOverloadedCalls      int
	setTempUnschedulable    int
	updateSessionWindowCall int
	updateExtraCalls        int
}

func (r *compatCyberAccountStateRepo) SetError(context.Context, int64, string) error {
	r.setErrorCalls++
	return nil
}

func (r *compatCyberAccountStateRepo) SetRateLimited(context.Context, int64, time.Time) error {
	r.setRateLimitedCalls++
	return nil
}

func (r *compatCyberAccountStateRepo) SetModelRateLimit(context.Context, int64, string, time.Time, ...string) error {
	r.setModelRateLimitCalls++
	return nil
}

func (r *compatCyberAccountStateRepo) SetOverloaded(context.Context, int64, time.Time) error {
	r.setOverloadedCalls++
	return nil
}

func (r *compatCyberAccountStateRepo) SetTempUnschedulable(context.Context, int64, time.Time, string) error {
	r.setTempUnschedulable++
	return nil
}

func (r *compatCyberAccountStateRepo) UpdateSessionWindow(context.Context, int64, *time.Time, *time.Time, string) error {
	r.updateSessionWindowCall++
	return nil
}

func (r *compatCyberAccountStateRepo) UpdateExtra(context.Context, int64, map[string]any) error {
	r.updateExtraCalls++
	return nil
}

func (r *compatCyberAccountStateRepo) mutationCount() int {
	return r.setErrorCalls +
		r.setRateLimitedCalls +
		r.setModelRateLimitCalls +
		r.setOverloadedCalls +
		r.setTempUnschedulable +
		r.updateSessionWindowCall +
		r.updateExtraCalls
}

func compatCyberHTTPConfig() *config.Config {
	return &config.Config{
		Security: config.SecurityConfig{
			URLAllowlist: config.URLAllowlistConfig{
				Enabled:           false,
				AllowInsecureHTTP: true,
			},
		},
	}
}

func compatCyberHTTPAPIKeyAccount(useResponses bool) *Account {
	account := &Account{
		ID:          7301,
		Name:        "compat-cyber-apikey",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  "sk-test",
			"base_url": "https://upstream.example",
		},
	}
	if !useResponses {
		account.Extra = map[string]any{
			openai_compat.ExtraKeyResponsesSupported: false,
		}
	}
	return account
}

func compatCyberHTTPResponse(statusCode int) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
			"x-request-id": []string{fmt.Sprintf("rid-cyber-%d", statusCode)},
		},
		Body: io.NopCloser(strings.NewReader(
			fmt.Sprintf(`{"error":{"code":"cyber_policy","message":"blocked at %d"}}`, statusCode),
		)),
	}
}

type compatCyberHTTPPath struct {
	name               string
	endpoint           string
	body               []byte
	useResponses       bool
	wantUpstreamSuffix string
	forward            func(context.Context, *OpenAIGatewayService, *gin.Context, *Account, []byte) (*OpenAIForwardResult, error)
}

func compatCyberHTTPPaths() []compatCyberHTTPPath {
	chatBody := []byte(`{"model":"gpt-5.5","messages":[{"role":"user","content":"hi"}],"stream":false}`)
	messagesBody := []byte(`{"model":"gpt-5.5","max_tokens":64,"messages":[{"role":"user","content":"hi"}],"stream":false}`)
	return []compatCyberHTTPPath{
		{
			name:               "chat_responses_bridge",
			endpoint:           "/v1/chat/completions",
			body:               chatBody,
			useResponses:       true,
			wantUpstreamSuffix: "/v1/responses",
			forward: func(ctx context.Context, svc *OpenAIGatewayService, c *gin.Context, account *Account, body []byte) (*OpenAIForwardResult, error) {
				return svc.ForwardAsChatCompletions(ctx, c, account, body, "", "gpt-5.5")
			},
		},
		{
			name:               "raw_chat",
			endpoint:           "/v1/chat/completions",
			body:               chatBody,
			useResponses:       false,
			wantUpstreamSuffix: "/v1/chat/completions",
			forward: func(ctx context.Context, svc *OpenAIGatewayService, c *gin.Context, account *Account, body []byte) (*OpenAIForwardResult, error) {
				return svc.forwardAsRawChatCompletions(ctx, c, account, body, "gpt-5.5")
			},
		},
		{
			name:               "messages_responses_bridge",
			endpoint:           "/v1/messages",
			body:               messagesBody,
			useResponses:       true,
			wantUpstreamSuffix: "/v1/responses",
			forward: func(ctx context.Context, svc *OpenAIGatewayService, c *gin.Context, account *Account, body []byte) (*OpenAIForwardResult, error) {
				return svc.ForwardAsAnthropic(ctx, c, account, body, "", "gpt-5.5")
			},
		},
		{
			name:               "messages_raw_fallback",
			endpoint:           "/v1/messages",
			body:               messagesBody,
			useResponses:       false,
			wantUpstreamSuffix: "/v1/chat/completions",
			forward: func(ctx context.Context, svc *OpenAIGatewayService, c *gin.Context, account *Account, body []byte) (*OpenAIForwardResult, error) {
				return svc.ForwardAsAnthropic(ctx, c, account, body, "", "gpt-5.5")
			},
		},
	}
}

func TestCompatHTTPNon2xxCyberPolicyNeverFailsOverOrCoolsAccount(t *testing.T) {
	setGinTestMode()

	for _, statusCode := range []int{
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
	} {
		statusCode := statusCode
		t.Run(fmt.Sprintf("status_%d", statusCode), func(t *testing.T) {
			for _, path := range compatCyberHTTPPaths() {
				path := path
				t.Run(path.name, func(t *testing.T) {
					rec := httptest.NewRecorder()
					c, _ := gin.CreateTestContext(rec)
					c.Request = httptest.NewRequest(http.MethodPost, path.endpoint, bytes.NewReader(path.body))
					c.Request.Header.Set("Content-Type", "application/json")

					cfg := compatCyberHTTPConfig()
					repo := &compatCyberAccountStateRepo{}
					upstream := &httpUpstreamRecorder{resp: compatCyberHTTPResponse(statusCode)}
					svc := &OpenAIGatewayService{
						accountRepo:  repo,
						cfg:          cfg,
						httpUpstream: upstream,
					}
					svc.rateLimitService = NewRateLimitService(repo, nil, cfg, nil, nil)
					svc.rateLimitService.SetAccountRuntimeBlocker(svc)
					account := compatCyberHTTPAPIKeyAccount(path.useResponses)

					result, err := path.forward(context.Background(), svc, c, account, path.body)

					require.Error(t, err)
					require.Nil(t, result)
					var failoverErr *UpstreamFailoverError
					require.False(t, errors.As(err, &failoverErr), "cyber_policy must remain a terminal request error")
					require.Len(t, upstream.requests, 1, "cyber_policy must not replay or fail over")
					require.True(t, strings.HasSuffix(upstream.requests[0].URL.Path, path.wantUpstreamSuffix))
					require.Equal(t, statusCode, rec.Code)
					require.NotNil(t, GetOpsCyberPolicy(c))
					_, hasFailoverEvents := c.Get(OpsUpstreamErrorsKey)
					require.False(t, hasFailoverEvents)
					require.Zero(t, repo.mutationCount(), "cyber_policy must not persist account cooldown or error state")
					_, runtimeBlocked := svc.openaiAccountRuntimeBlockUntil.Load(account.ID)
					require.False(t, runtimeBlocked, "cyber_policy must not install account runtime cooldown")
					require.Nil(t, svc.openaiModelTransient, "cyber_policy must not mutate model transient state")
				})
			}
		})
	}
}

func TestChatResponsesBridgeHTTP404CyberPolicySkipsRawFallback(t *testing.T) {
	setGinTestMode()

	body := []byte(`{"model":"gpt-5.5","messages":[{"role":"user","content":"hi"}],"stream":false}`)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	upstream := &httpUpstreamRecorder{responses: []*http.Response{
		compatCyberHTTPResponse(http.StatusNotFound),
		{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body: io.NopCloser(strings.NewReader(
				`{"id":"chatcmpl_fallback","choices":[{"message":{"role":"assistant","content":"must not run"}}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`,
			)),
		},
	}}
	cfg := compatCyberHTTPConfig()
	svc := &OpenAIGatewayService{cfg: cfg, httpUpstream: upstream}
	account := compatCyberHTTPAPIKeyAccount(true)

	result, err := svc.ForwardAsChatCompletions(context.Background(), c, account, body, "", "gpt-5.5")

	require.Error(t, err)
	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.False(t, errors.As(err, &failoverErr))
	require.Len(t, upstream.requests, 1, "404 cyber_policy must win over unsupported /responses fallback")
	require.True(t, strings.HasSuffix(upstream.requests[0].URL.Path, "/v1/responses"))
	require.NotNil(t, GetOpsCyberPolicy(c))
}

func TestCompatHTTPNonCyber500StillFailsOver(t *testing.T) {
	setGinTestMode()

	for _, path := range compatCyberHTTPPaths()[:2] {
		path := path
		t.Run(path.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodPost, path.endpoint, bytes.NewReader(path.body))
			c.Request.Header.Set("Content-Type", "application/json")

			upstream := &httpUpstreamRecorder{resp: &http.Response{
				StatusCode: http.StatusInternalServerError,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"error":{"code":"server_error","message":"temporary outage"}}`)),
			}}
			cfg := compatCyberHTTPConfig()
			svc := &OpenAIGatewayService{cfg: cfg, httpUpstream: upstream}
			account := compatCyberHTTPAPIKeyAccount(path.useResponses)

			result, err := path.forward(context.Background(), svc, c, account, path.body)

			require.Error(t, err)
			require.Nil(t, result)
			var failoverErr *UpstreamFailoverError
			require.True(t, errors.As(err, &failoverErr), "ordinary 500 must keep existing failover behavior")
			require.Equal(t, http.StatusInternalServerError, failoverErr.StatusCode)
			require.Len(t, upstream.requests, 1)
			require.Nil(t, GetOpsCyberPolicy(c))
		})
	}
}

func TestShouldFailoverOpenAIUpstreamResponseRejectsCyberPolicyDefensively(t *testing.T) {
	svc := &OpenAIGatewayService{}
	for _, statusCode := range []int{
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
	} {
		body := []byte(`{"error":{"code":"cyber_policy","message":"blocked"}}`)
		require.False(t, svc.shouldFailoverOpenAIUpstreamResponse(statusCode, "blocked", body))
	}
	require.True(t, svc.shouldFailoverOpenAIUpstreamResponse(
		http.StatusInternalServerError,
		"temporary outage",
		[]byte(`{"error":{"code":"server_error","message":"temporary outage"}}`),
	))
}
