package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type webChatDrainTestCloser struct {
	once   sync.Once
	calls  atomic.Int32
	closed chan struct{}
}

type webChatBlockingHeaderUpstream struct {
	startedOnce  sync.Once
	canceledOnce sync.Once
	started      chan struct{}
	canceled     chan struct{}
	calls        atomic.Int32
}

type webChatCanceledHTTPErrorUpstream struct {
	startedOnce sync.Once
	started     chan struct{}
	release     chan struct{}
	calls       atomic.Int32
	statusCode  int
	body        string
}

func (u *webChatCanceledHTTPErrorUpstream) Do(_ *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	u.calls.Add(1)
	u.startedOnce.Do(func() { close(u.started) })
	<-u.release
	statusCode := u.statusCode
	if statusCode == 0 {
		statusCode = http.StatusNotFound
	}
	responseBody := io.ReadCloser(http.NoBody)
	if u.body != "" {
		responseBody = io.NopCloser(strings.NewReader(u.body))
	}
	return &http.Response{
		StatusCode: statusCode,
		Header:     make(http.Header),
		Body:       responseBody,
	}, nil
}

func (u *webChatCanceledHTTPErrorUpstream) DoWithTLS(
	req *http.Request,
	proxyURL string,
	accountID int64,
	accountConcurrency int,
	_ *tlsfingerprint.Profile,
) (*http.Response, error) {
	return u.Do(req, proxyURL, accountID, accountConcurrency)
}

func (u *webChatBlockingHeaderUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	u.calls.Add(1)
	u.startedOnce.Do(func() { close(u.started) })
	<-req.Context().Done()
	u.canceledOnce.Do(func() { close(u.canceled) })
	return nil, req.Context().Err()
}

func (u *webChatBlockingHeaderUpstream) DoWithTLS(
	req *http.Request,
	proxyURL string,
	accountID int64,
	accountConcurrency int,
	_ *tlsfingerprint.Profile,
) (*http.Response, error) {
	return u.Do(req, proxyURL, accountID, accountConcurrency)
}

func (c *webChatDrainTestCloser) Close() error {
	c.calls.Add(1)
	c.once.Do(func() { close(c.closed) })
	return nil
}

func TestWebChatDrainGuardCancelsDetachedUpstreamAfterBoundedTimeout(t *testing.T) {
	parent := context.WithValue(context.Background(), ctxkey.WebChat, true)
	ctx, cancel := context.WithCancel(parent)
	closer := &webChatDrainTestCloser{closed: make(chan struct{})}
	guard := newWebChatDrainGuard(ctx, closer, 20*time.Millisecond)
	require.NotNil(t, guard)
	t.Cleanup(guard.Stop)

	cancel()
	require.Eventually(t, guard.Started, time.Second, time.Millisecond)
	select {
	case <-closer.closed:
	case <-time.After(time.Second):
		t.Fatal("detached Web Chat upstream was not canceled after the drain timeout")
	}

	require.True(t, guard.TimedOut())
	guard.Stop()
	guard.Stop()
	require.EqualValues(t, 1, closer.calls.Load())
}

func TestWebChatDrainGuardIgnoresOrdinaryGatewayRequests(t *testing.T) {
	closer := &webChatDrainTestCloser{closed: make(chan struct{})}
	require.Nil(t, newWebChatDrainGuard(context.Background(), closer, time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())
	upstreamCtx, guard := newWebChatUpstreamDrainContext(ctx, time.Millisecond)
	require.Nil(t, guard)
	cancel()
	require.NoError(t, upstreamCtx.Err(), "ordinary Gateway upstreams must retain detached cancellation semantics")
}

func TestWebChatDrainGuardClosesBodyAttachedAfterTimeout(t *testing.T) {
	parent := context.WithValue(context.Background(), ctxkey.WebChat, true)
	ctx, cancel := context.WithCancel(parent)
	upstreamCtx, guard := newWebChatUpstreamDrainContext(ctx, 20*time.Millisecond)
	require.NotNil(t, guard)
	t.Cleanup(guard.Stop)

	cancel()
	require.Eventually(t, guard.TimedOut, time.Second, time.Millisecond)
	require.ErrorIs(t, upstreamCtx.Err(), context.Canceled)

	closer := &webChatDrainTestCloser{closed: make(chan struct{})}
	guard.SetBody(closer)
	select {
	case <-closer.closed:
	case <-time.After(time.Second):
		t.Fatal("a response body arriving at the timeout boundary was not closed")
	}
	require.EqualValues(t, 1, closer.calls.Load())
}

func TestForwardAsChatCompletionsWebChatDisconnectBeforeHeadersIsBounded(t *testing.T) {
	setGinTestMode()
	upstream := &webChatBlockingHeaderUpstream{started: make(chan struct{}), canceled: make(chan struct{})}
	svc := &OpenAIGatewayService{
		cfg: &config.Config{Gateway: config.GatewayConfig{
			StreamDataIntervalTimeout: 1,
			MaxLineSize:               defaultMaxLineSize,
		}},
		httpUpstream: upstream,
	}
	parent := context.WithValue(context.Background(), ctxkey.WebChat, true)
	reqCtx, cancel := context.WithCancel(parent)
	defer cancel()
	body := []byte(`{"model":"gpt-5.4","messages":[{"role":"user","content":"hello"}],"stream":true}`)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body)).WithContext(reqCtx)
	account := &Account{
		ID: 1, Name: "web-chat-oauth", Platform: PlatformOpenAI, Type: AccountTypeOAuth, Concurrency: 1,
		Credentials: map[string]any{"access_token": "test-token", "chatgpt_account_id": "test-account"},
	}

	type forwardResult struct {
		result *OpenAIForwardResult
		err    error
	}
	done := make(chan forwardResult, 1)
	go func() {
		result, err := svc.ForwardAsChatCompletions(reqCtx, c, account, body, "", "")
		done <- forwardResult{result: result, err: err}
	}()
	select {
	case <-upstream.started:
	case <-time.After(time.Second):
		t.Fatal("upstream request did not start")
	}

	canceledAt := time.Now()
	cancel()
	var got forwardResult
	select {
	case got = <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("response-header wait exceeded the Web Chat drain deadline")
	}

	require.ErrorIs(t, got.err, errWebChatUpstreamDrainTimeout)
	require.NotNil(t, got.result)
	require.True(t, got.result.ClientDisconnect)
	require.EqualValues(t, 1, upstream.calls.Load())
	_, opsRecorded := c.Get(OpsUpstreamErrorsKey)
	require.False(t, opsRecorded, "a client-driven drain deadline must not be attributed as an upstream fault")
	require.GreaterOrEqual(t, time.Since(canceledAt), 900*time.Millisecond, "browser cancellation must not release the detached upstream immediately")
	select {
	case <-upstream.canceled:
	default:
		t.Fatal("drain timeout did not cancel the response-header request context")
	}
}

func TestForwardAsRawChatCompletionsWebChatDisconnectBeforeHeadersIsBounded(t *testing.T) {
	setGinTestMode()
	upstream := &webChatBlockingHeaderUpstream{started: make(chan struct{}), canceled: make(chan struct{})}
	svc := &OpenAIGatewayService{
		cfg: &config.Config{
			Gateway:  config.GatewayConfig{StreamDataIntervalTimeout: 1, MaxLineSize: defaultMaxLineSize},
			Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false, AllowInsecureHTTP: true}},
		},
		httpUpstream: upstream,
	}
	parent := context.WithValue(context.Background(), ctxkey.WebChat, true)
	reqCtx, cancel := context.WithCancel(parent)
	defer cancel()
	body := []byte(`{"model":"gpt-5.4","messages":[{"role":"user","content":"hello"}],"stream":true}`)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body)).WithContext(reqCtx)
	account := &Account{
		ID: 2, Name: "web-chat-apikey", Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Concurrency: 1,
		Credentials: map[string]any{"api_key": "sk-test", "base_url": "http://upstream.example"},
	}

	type forwardResult struct {
		result *OpenAIForwardResult
		err    error
	}
	done := make(chan forwardResult, 1)
	go func() {
		result, err := svc.forwardAsRawChatCompletions(reqCtx, c, account, body, "")
		done <- forwardResult{result: result, err: err}
	}()
	select {
	case <-upstream.started:
	case <-time.After(time.Second):
		t.Fatal("upstream request did not start")
	}

	cancel()
	var got forwardResult
	select {
	case got = <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("raw response-header wait exceeded the Web Chat drain deadline")
	}

	require.ErrorIs(t, got.err, errWebChatUpstreamDrainTimeout)
	require.NotNil(t, got.result)
	require.True(t, got.result.ClientDisconnect)
	require.EqualValues(t, 1, upstream.calls.Load())
	_, opsRecorded := c.Get(OpsUpstreamErrorsKey)
	require.False(t, opsRecorded, "a raw client-driven drain deadline must not be attributed as an upstream fault")
	select {
	case <-upstream.canceled:
	default:
		t.Fatal("raw drain timeout did not cancel the response-header request context")
	}
}

func TestForwardAsChatCompletionsWebChatCancelBeforeHTTPErrorDoesNotFallback(t *testing.T) {
	setGinTestMode()
	upstream := &webChatCanceledHTTPErrorUpstream{started: make(chan struct{}), release: make(chan struct{})}
	svc := &OpenAIGatewayService{
		cfg: &config.Config{
			Gateway:  config.GatewayConfig{StreamDataIntervalTimeout: 1, MaxLineSize: defaultMaxLineSize},
			Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false, AllowInsecureHTTP: true}},
		},
		httpUpstream: upstream,
	}
	parent := context.WithValue(context.Background(), ctxkey.WebChat, true)
	reqCtx, cancel := context.WithCancel(parent)
	defer cancel()
	body := []byte(`{"model":"gpt-5.4","messages":[{"role":"user","content":"hello"}],"stream":true}`)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body)).WithContext(reqCtx)
	account := &Account{
		ID: 3, Name: "web-chat-apikey-unknown-responses", Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Concurrency: 1,
		Credentials: map[string]any{"api_key": "sk-test", "base_url": "http://upstream.example"},
	}

	type forwardResult struct {
		result *OpenAIForwardResult
		err    error
	}
	done := make(chan forwardResult, 1)
	go func() {
		result, err := svc.ForwardAsChatCompletions(reqCtx, c, account, body, "", "")
		done <- forwardResult{result: result, err: err}
	}()
	select {
	case <-upstream.started:
	case <-time.After(time.Second):
		t.Fatal("upstream request did not start")
	}

	cancel()
	close(upstream.release)
	select {
	case got := <-done:
		require.ErrorIs(t, got.err, context.Canceled)
		require.NotNil(t, got.result)
		require.True(t, got.result.ClientDisconnect)
	case <-time.After(time.Second):
		t.Fatal("canceled Web Chat request did not stop at the first HTTP error")
	}
	require.EqualValues(t, 1, upstream.calls.Load(), "a canceled Web Chat request must not replay through the raw fallback")
	require.Nil(t, GetOpsCyberPolicy(c), "ordinary canceled upstream errors must not create a Cyber mark")
	_, opsRecorded := c.Get(OpsUpstreamErrorsKey)
	require.False(t, opsRecorded, "ordinary canceled upstream errors keep the client-disconnect attribution")
}

func TestForwardAsChatCompletionsWebChatCancelBeforeHTTPCyberStillMarksPolicy(t *testing.T) {
	setGinTestMode()
	upstream := &webChatCanceledHTTPErrorUpstream{
		started:    make(chan struct{}),
		release:    make(chan struct{}),
		statusCode: http.StatusTooManyRequests,
		body:       `{"error":{"code":"cyber_policy","message":"blocked after browser disconnect"}}`,
	}
	cfg := compatCyberHTTPConfig()
	repo := &compatCyberAccountStateRepo{}
	svc := &OpenAIGatewayService{
		accountRepo:  repo,
		cfg:          cfg,
		httpUpstream: upstream,
	}
	svc.rateLimitService = NewRateLimitService(repo, nil, cfg, nil, nil)
	svc.rateLimitService.SetAccountRuntimeBlocker(svc)

	parent := context.WithValue(context.Background(), ctxkey.WebChat, true)
	reqCtx, cancel := context.WithCancel(parent)
	defer cancel()
	body := []byte(`{"model":"gpt-5.5","messages":[{"role":"user","content":"hello"}],"stream":false}`)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body)).WithContext(reqCtx)
	account := compatCyberHTTPAPIKeyAccount(true)

	type forwardResult struct {
		result *OpenAIForwardResult
		err    error
	}
	done := make(chan forwardResult, 1)
	go func() {
		result, err := svc.ForwardAsChatCompletions(reqCtx, c, account, body, "", "gpt-5.5")
		done <- forwardResult{result: result, err: err}
	}()
	select {
	case <-upstream.started:
	case <-time.After(time.Second):
		t.Fatal("upstream request did not start")
	}

	cancel()
	close(upstream.release)

	var got forwardResult
	select {
	case got = <-done:
	case <-time.After(time.Second):
		t.Fatal("canceled Web Chat Cyber response was not processed")
	}

	require.Error(t, got.err)
	require.Nil(t, got.result, "Cyber must use the dedicated producer even after the browser disconnects")
	var failoverErr *UpstreamFailoverError
	require.False(t, errors.As(got.err, &failoverErr))
	mark := GetOpsCyberPolicy(c)
	require.NotNil(t, mark, "arrived Cyber evidence must survive browser cancellation")
	require.Equal(t, "cyber_policy", mark.Code)
	require.Equal(t, http.StatusTooManyRequests, mark.UpstreamStatus)
	require.EqualValues(t, 1, upstream.calls.Load())
	require.Zero(t, repo.mutationCount(), "Cyber must not persist account cooldown state")
	_, runtimeBlocked := svc.openaiAccountRuntimeBlockUntil.Load(account.ID)
	require.False(t, runtimeBlocked)
	require.Nil(t, svc.openaiModelTransient)
	_, hasFailoverEvents := c.Get(OpsUpstreamErrorsKey)
	require.False(t, hasFailoverEvents)
}
