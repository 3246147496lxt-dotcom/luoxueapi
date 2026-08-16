package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai_compat"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type webChatReasoningFailoverAccountRepo struct {
	service.AccountRepository
	accounts  []service.Account
	listCalls int
}

func (r *webChatReasoningFailoverAccountRepo) ListSchedulableByGroupIDAndPlatform(_ context.Context, _ int64, platform string) ([]service.Account, error) {
	r.listCalls++
	return r.accountsForPlatform(platform), nil
}

func (r *webChatReasoningFailoverAccountRepo) ListSchedulableByPlatform(_ context.Context, platform string) ([]service.Account, error) {
	r.listCalls++
	return r.accountsForPlatform(platform), nil
}

func (r *webChatReasoningFailoverAccountRepo) ListSchedulableUngroupedByPlatform(_ context.Context, platform string) ([]service.Account, error) {
	r.listCalls++
	return r.accountsForPlatform(platform), nil
}

func (r *webChatReasoningFailoverAccountRepo) accountsForPlatform(platform string) []service.Account {
	accounts := make([]service.Account, 0, len(r.accounts))
	for i := range r.accounts {
		if r.accounts[i].Platform == platform {
			accounts = append(accounts, r.accounts[i])
		}
	}
	return accounts
}

type webChatReasoningFailoverUpstream struct {
	service.HTTPUpstream
	mu            sync.Mutex
	accountIDs    []int64
	paths         []string
	bodies        [][]byte
	failureCounts map[int64]int
}

func (u *webChatReasoningFailoverUpstream) Do(req *http.Request, _ string, accountID int64, _ int) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	u.mu.Lock()
	u.accountIDs = append(u.accountIDs, accountID)
	u.paths = append(u.paths, req.URL.Path)
	u.bodies = append(u.bodies, append([]byte(nil), body...))
	shouldFail := u.failureCounts[accountID] > 0
	if shouldFail {
		u.failureCounts[accountID]--
	}
	u.mu.Unlock()
	if shouldFail {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Status:     "500 Internal Server Error",
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewBufferString(`{"error":{"type":"server_error","message":"retry another account"}}`)),
		}, nil
	}
	if strings.HasSuffix(req.URL.Path, "/chat/completions") {
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body: io.NopCloser(bytes.NewBufferString(
				`{"id":"chatcmpl_public","object":"chat.completion","model":"gpt-5.6-sol","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":2,"total_tokens":7}}`,
			)),
		}, nil
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body: io.NopCloser(bytes.NewBufferString(
			`data: {"type":"response.completed","response":{"id":"resp_pro","object":"response","model":"gpt-5.6-sol","status":"completed","output":[{"type":"message","id":"msg_1","role":"assistant","status":"completed","content":[{"type":"output_text","text":"ok"}]}],"usage":{"input_tokens":5,"output_tokens":2,"total_tokens":7}}}` + "\n\n",
		)),
	}, nil
}

func (u *webChatReasoningFailoverUpstream) calls() ([]int64, []string, [][]byte) {
	u.mu.Lock()
	defer u.mu.Unlock()
	bodies := make([][]byte, len(u.bodies))
	for i := range u.bodies {
		bodies[i] = append([]byte(nil), u.bodies[i]...)
	}
	return append([]int64(nil), u.accountIDs...), append([]string(nil), u.paths...), bodies
}

func newWebChatReasoningGatewayTestHandler(
	t *testing.T,
	groupID int64,
	accounts []service.Account,
	upstream service.HTTPUpstream,
) (*OpenAIGatewayHandler, *service.APIKey, *webChatReasoningFailoverAccountRepo) {
	t.Helper()
	cfg := &config.Config{RunMode: config.RunModeSimple}
	cfg.Security.URLAllowlist.Enabled = false
	billingCache := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(billingCache.Stop)
	repo := &webChatReasoningFailoverAccountRepo{accounts: accounts}
	gateway := service.NewOpenAIGatewayService(
		repo,
		nil, nil, nil, nil, nil, nil, cfg, nil, nil,
		service.NewBillingService(cfg, nil), nil, billingCache, upstream,
		&service.DeferredService{}, nil, nil, nil, nil, nil, nil, nil,
	)
	cache := &concurrencyCacheMock{
		acquireUserSlotFn:    func(context.Context, int64, int, string) (bool, error) { return true, nil },
		acquireAccountSlotFn: func(context.Context, int64, int, string) (bool, error) { return true, nil },
	}
	handler := &OpenAIGatewayHandler{
		gatewayService:      gateway,
		billingCacheService: billingCache,
		apiKeyService:       &service.APIKeyService{},
		concurrencyHelper:   NewConcurrencyHelper(service.NewConcurrencyService(cache), SSEPingFormatNone, time.Second),
		maxAccountSwitches:  3,
		cfg:                 cfg,
	}
	apiKey := &service.APIKey{
		ID: groupID + 1, GroupID: &groupID,
		User:  &service.User{ID: groupID + 2, Status: service.StatusActive},
		Group: &service.Group{ID: groupID, Platform: service.PlatformOpenAI, Status: service.StatusActive},
	}
	return handler, apiKey, repo
}

func runWebChatReasoningGatewayRequest(
	t *testing.T,
	handler *OpenAIGatewayHandler,
	apiKey *service.APIKey,
	body string,
	options service.WebChatReasoningOptions,
) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/openai/v1/chat/completions", bytes.NewBufferString(body))
	c.Set(string(middleware2.ContextKeyAPIKey), apiKey)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: apiKey.User.ID, Concurrency: 1})
	service.SetWebChatReasoningOptions(c, options)
	handler.ChatCompletions(c)
	return recorder
}

func runPublicChatGatewayRequest(
	t *testing.T,
	handler *OpenAIGatewayHandler,
	apiKey *service.APIKey,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/openai/v1/chat/completions", bytes.NewBufferString(body))
	c.Set(string(middleware2.ContextKeyAPIKey), apiKey)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: apiKey.User.ID, Concurrency: 1})
	handler.ChatCompletions(c)
	return recorder
}

func TestOpenAIChatCompletionsWebChatProSkipsRawCCAccountAndForwardsOnlyResponses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	groupID := int64(4110)
	accounts := []service.Account{
		{
			ID: 1, Name: "raw-cc", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey,
			Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 0,
			Credentials: map[string]any{"api_key": "sk-raw", "base_url": "https://raw.example/v1"},
			Extra:       map[string]any{openai_compat.ExtraKeyResponsesSupported: false},
		},
		{
			ID: 2, Name: "responses-pro", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey,
			Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1,
			Credentials: map[string]any{"api_key": "sk-responses", "base_url": "https://responses.example/v1"},
			Extra:       map[string]any{openai_compat.ExtraKeyResponsesSupported: true},
		},
	}
	upstream := &webChatReasoningFailoverUpstream{}
	h, apiKey, repo := newWebChatReasoningGatewayTestHandler(t, groupID, accounts, upstream)
	recorder := runWebChatReasoningGatewayRequest(t, h, apiKey,
		`{"model":"gpt-5.6-sol","messages":[{"role":"user","content":"hello"}],"reasoning_mode":"standard","reasoning_effort":"high","stream":false}`,
		service.WebChatReasoningOptions{
			Mode: service.WebChatReasoningModePro, Effort: "low",
		})

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.GreaterOrEqual(t, repo.listCalls, 2, "raw CC account should be excluded and the selector retried")
	accountIDs, paths, bodies := upstream.calls()
	require.Equal(t, []int64{2}, accountIDs, "raw CC account must never receive the request")
	require.Equal(t, []string{"/v1/responses"}, paths)
	require.Len(t, bodies, 1)
	var upstreamBody map[string]any
	require.NoError(t, json.Unmarshal(bodies[0], &upstreamBody))
	require.Equal(t, map[string]any{
		"mode": "pro", "summary": "auto",
	}, upstreamBody["reasoning"], "the selected Responses account must receive the trusted Pro contract without a standard effort")
}

func TestOpenAIChatCompletionsWebChatStandardForwardsTrustedReasoningToUpstream(t *testing.T) {
	gin.SetMode(gin.TestMode)
	groupID := int64(4210)
	accounts := []service.Account{{
		ID: 11, Name: "responses-standard", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey,
		Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 0,
		Credentials: map[string]any{"api_key": "sk-standard", "base_url": "https://standard.example/v1"},
		Extra:       map[string]any{openai_compat.ExtraKeyResponsesSupported: true},
	}}
	upstream := &webChatReasoningFailoverUpstream{}
	h, apiKey, _ := newWebChatReasoningGatewayTestHandler(t, groupID, accounts, upstream)
	recorder := runWebChatReasoningGatewayRequest(t, h, apiKey,
		`{"model":"gpt-5.6-sol","messages":[{"role":"user","content":"hello"}],"reasoning_mode":"pro","reasoning_effort":"high","stream":false}`,
		service.WebChatReasoningOptions{
			Mode: service.WebChatReasoningModeStandard, Effort: "medium",
		})

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	accountIDs, paths, bodies := upstream.calls()
	require.Equal(t, []int64{11}, accountIDs)
	require.Equal(t, []string{"/v1/responses"}, paths)
	require.Len(t, bodies, 1)
	var upstreamBody map[string]any
	require.NoError(t, json.Unmarshal(bodies[0], &upstreamBody))
	require.Equal(t, map[string]any{
		"mode": "standard", "effort": "medium", "summary": "auto",
	}, upstreamBody["reasoning"], "standard Web Chat must use the trusted context, not public body fields")
}

func TestOpenAIChatCompletionsWebChatProFailoverPreservesReasoningBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	groupID := int64(4310)
	accounts := []service.Account{
		{
			ID: 21, Name: "responses-pro-primary", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey,
			Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 0,
			Credentials: map[string]any{"api_key": "sk-primary", "base_url": "https://primary.example/v1"},
			Extra:       map[string]any{openai_compat.ExtraKeyResponsesSupported: true},
		},
		{
			ID: 22, Name: "responses-pro-secondary", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey,
			Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1,
			Credentials: map[string]any{"api_key": "sk-secondary", "base_url": "https://secondary.example/v1"},
			Extra:       map[string]any{openai_compat.ExtraKeyResponsesSupported: true},
		},
	}
	upstream := &webChatReasoningFailoverUpstream{failureCounts: map[int64]int{21: 1}}
	h, apiKey, _ := newWebChatReasoningGatewayTestHandler(t, groupID, accounts, upstream)
	recorder := runWebChatReasoningGatewayRequest(t, h, apiKey,
		`{"model":"gpt-5.6-sol","messages":[{"role":"user","content":"hello"}],"reasoning_effort":"medium","stream":false}`,
		service.WebChatReasoningOptions{
			Mode: service.WebChatReasoningModePro, Effort: "high",
		})

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	accountIDs, paths, bodies := upstream.calls()
	require.Equal(t, []int64{21, 22}, accountIDs, "the second Pro-capable account must receive the failover request")
	require.Equal(t, []string{"/v1/responses", "/v1/responses"}, paths)
	require.Len(t, bodies, 2)
	for i := range bodies {
		var upstreamBody map[string]any
		require.NoError(t, json.Unmarshal(bodies[i], &upstreamBody))
		require.Equal(t, map[string]any{
			"mode": "pro", "summary": "auto",
		}, upstreamBody["reasoning"], "failover attempt %d must preserve the trusted Pro contract without a standard effort", i+1)
	}
}

func TestOpenAIChatCompletionsPublicReasoningModeIsStrippedBeforeRawUpstream(t *testing.T) {
	gin.SetMode(gin.TestMode)
	groupID := int64(4410)
	accounts := []service.Account{{
		ID: 31, Name: "raw-public", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey,
		Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 0,
		Credentials: map[string]any{"api_key": "sk-raw-public", "base_url": "https://raw-public.example/v1"},
		Extra:       map[string]any{openai_compat.ExtraKeyResponsesSupported: false},
	}}
	upstream := &webChatReasoningFailoverUpstream{}
	h, apiKey, _ := newWebChatReasoningGatewayTestHandler(t, groupID, accounts, upstream)
	recorder := runPublicChatGatewayRequest(t, h, apiKey,
		`{"model":"gpt-5.6-sol","messages":[{"role":"user","content":"hello"}],"reasoning_mode":"pro","reasoning_effort":"high","stream":false}`,
	)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	accountIDs, paths, bodies := upstream.calls()
	require.Equal(t, []int64{31}, accountIDs)
	require.Equal(t, []string{"/v1/chat/completions"}, paths)
	require.Len(t, bodies, 1)
	var upstreamBody map[string]any
	require.NoError(t, json.Unmarshal(bodies[0], &upstreamBody))
	_, hasMode := upstreamBody["reasoning_mode"]
	require.False(t, hasMode, "public reasoning_mode must be stripped before raw forwarding")
	require.Equal(t, "high", upstreamBody["reasoning_effort"], "existing public effort behavior must remain unchanged")
}
