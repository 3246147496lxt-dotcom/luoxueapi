package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// Exercise real handler/service integration with both the legacy exact-key
// store and the additive transcript store used by the production Redis cache.
type openAICyberContinuationCache struct {
	mu      sync.Mutex
	blocked map[string]bool
	scopes  map[string]bool
}

func (*openAICyberContinuationCache) GetSessionAccountID(context.Context, int64, string) (int64, error) {
	return 0, nil
}
func (*openAICyberContinuationCache) SetSessionAccountID(context.Context, int64, string, int64, time.Duration) error {
	return nil
}
func (*openAICyberContinuationCache) RefreshSessionTTL(context.Context, int64, string, time.Duration) error {
	return nil
}
func (*openAICyberContinuationCache) DeleteSessionAccountID(context.Context, int64, string) error {
	return nil
}
func (s *openAICyberContinuationCache) SetCyberSessionBlocked(ctx context.Context, key string, ttl time.Duration) error {
	return s.SetCyberSessionBlockedKeys(ctx, "", []string{key}, ttl)
}
func (s *openAICyberContinuationCache) IsCyberSessionBlocked(ctx context.Context, key string) (bool, error) {
	found, err := s.FindCyberSessionBlocked(ctx, []string{key})
	return found != "", err
}
func (s *openAICyberContinuationCache) SetCyberSessionBlockedKeys(_ context.Context, scopeKey string, keys []string, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.blocked == nil {
		s.blocked = map[string]bool{}
		s.scopes = map[string]bool{}
	}
	for _, key := range keys {
		s.blocked[key] = true
	}
	if scopeKey != "" {
		s.scopes[scopeKey] = true
	}
	return nil
}
func (s *openAICyberContinuationCache) IsCyberSessionScopeActive(_ context.Context, scopeKey string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.scopes[scopeKey], nil
}
func (s *openAICyberContinuationCache) FindCyberSessionBlocked(_ context.Context, keys []string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, key := range keys {
		if s.blocked[key] {
			return key, nil
		}
	}
	return "", nil
}

func newOpenAICyberContinuationHandler(t *testing.T, wsBaseURL string, failover bool) (*OpenAIGatewayHandler, *service.APIKey, *openAIHTTPCyberHandlerUpstream, *openAIHTTPCyberUsageRepo) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{RunMode: config.RunModeSimple}
	cfg.Default.RateMultiplier = 1
	cfg.Security.URLAllowlist.Enabled = false
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	cfg.Gateway.MaxAccountSwitches = 3
	cfg.Gateway.OpenAIWS.Enabled = true
	cfg.Gateway.OpenAIWS.APIKeyEnabled = true
	cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true
	cfg.Gateway.OpenAIWS.IngressModeDefault = service.OpenAIWSIngressModeCtxPool
	cfg.Gateway.OpenAIWS.DialTimeoutSeconds = 3
	cfg.Gateway.OpenAIWS.ReadTimeoutSeconds = 3
	cfg.Gateway.OpenAIWS.WriteTimeoutSeconds = 3

	settingService := service.NewSettingService(&openAIWSCyberSettingRepo{values: map[string]string{
		service.SettingKeyCyberSessionBlockEnabled:    "true",
		service.SettingKeyCyberSessionBlockTTLSeconds: "60",
	}}, cfg)
	account := service.Account{
		ID:          9201,
		Name:        "cyber-continuation-account",
		Platform:    service.PlatformOpenAI,
		Type:        service.AccountTypeOAuth,
		Status:      service.StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{"access_token": "test-token"},
	}
	if wsBaseURL != "" {
		account.Type = service.AccountTypeAPIKey
		account.Credentials = map[string]any{"api_key": "sk-test", "base_url": wsBaseURL}
		account.Extra = map[string]any{"openai_apikey_responses_websockets_v2_enabled": true}
	}
	var accountRepo service.AccountRepository = &openAIWSUsageHandlerAccountRepoStub{account: account}
	if failover {
		cfg.Gateway.OpenAIWS.ModeRouterV2Enabled = true
		account.Priority = 1
		account.Extra = map[string]any{
			"openai_apikey_responses_websockets_v2_enabled": true,
			"openai_apikey_responses_websockets_v2_mode":    service.OpenAIWSIngressModePassthrough,
		}
		second := account
		second.ID++
		second.Priority = 2
		second.Extra = map[string]any{
			"openai_apikey_responses_websockets_v2_enabled": true,
			"openai_apikey_responses_websockets_v2_mode":    service.OpenAIWSIngressModeCtxPool,
		}
		accountRepo = &openAIWSFailoverHandlerAccountRepoStub{accounts: []service.Account{account, second}}
	}
	usageRepo := &openAIHTTPCyberUsageRepo{}
	upstream := &openAIHTTPCyberHandlerUpstream{}
	rateLimitService := service.NewRateLimitService(accountRepo, usageRepo, cfg, nil, nil)
	rateLimitService.SetSettingService(settingService)
	concurrencyService := service.NewConcurrencyService(&concurrencyCacheMock{
		acquireUserSlotFn: func(context.Context, int64, int, string) (bool, error) {
			return true, nil
		},
		acquireAccountSlotFn: func(context.Context, int64, int, string) (bool, error) {
			return true, nil
		},
	})
	billingCacheService := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(billingCacheService.Stop)
	gatewayService := service.NewOpenAIGatewayService(
		accountRepo, usageRepo, nil, nil, nil, nil, &openAICyberContinuationCache{}, cfg, nil,
		concurrencyService, service.NewBillingService(cfg, nil), rateLimitService,
		billingCacheService, upstream, &service.DeferredService{}, nil, nil, nil,
		nil, nil, settingService, nil,
	)
	t.Cleanup(gatewayService.CloseOpenAIWSPool)
	h := NewOpenAIGatewayHandler(
		gatewayService, concurrencyService, billingCacheService,
		service.NewAPIKeyService(nil, nil, nil, nil, nil, nil, cfg), nil, nil, nil, nil, cfg,
	)
	groupID := int64(9101)
	apiKey := &service.APIKey{
		ID:      9301,
		GroupID: &groupID,
		User:    &service.User{ID: 9401, Status: service.StatusActive},
		Group: &service.Group{
			ID:                    groupID,
			Platform:              service.PlatformOpenAI,
			Status:                service.StatusActive,
			RateMultiplier:        1,
			AllowMessagesDispatch: true,
		},
	}
	return h, apiKey, upstream, usageRepo
}

func setOpenAICyberContinuationAuth(c *gin.Context, apiKey *service.APIKey) {
	c.Set(string(middleware.ContextKeyAPIKey), apiKey)
	c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: apiKey.User.ID, Concurrency: 1})
}

func TestOpenAIHTTPHandlersCyberTranscriptContinuation(t *testing.T) {
	const original = `[{"role":"user","content":"shared opening"},{"role":"assistant","content":"unique conversation context 481"},{"role":"user","content":"flagged request"}]`
	const rewritten = `[{"role":"user","content":"shared opening"},{"role":"assistant","content":"unique conversation context 481"},{"role":"user","content":"try a rewritten request"}]`
	const independent = `[{"role":"user","content":"shared opening"},{"role":"assistant","content":"a different conversation context 782"},{"role":"user","content":"a new question"}]`
	for _, test := range []struct {
		name  string
		path  string
		field string
		call  func(*OpenAIGatewayHandler, *gin.Context)
	}{
		{"responses", "/v1/responses", "input", (*OpenAIGatewayHandler).Responses},
		{"chat_completions", "/v1/chat/completions", "messages", (*OpenAIGatewayHandler).ChatCompletions},
		{"messages", "/v1/messages", "messages", (*OpenAIGatewayHandler).Messages},
	} {
		t.Run(test.name, func(t *testing.T) {
			h, apiKey, upstream, usageRepo := newOpenAICyberContinuationHandler(t, "", false)
			request := func(key *service.APIKey, transcript string) (*httptest.ResponseRecorder, *gin.Context) {
				recorder := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(recorder)
				body := `{"model":"gpt-5.5","max_tokens":64,"stream":false,"` + test.field + `":` + transcript + `}`
				c.Request = httptest.NewRequest(http.MethodPost, test.path, strings.NewReader(body))
				c.Request.Header.Set("Content-Type", "application/json")
				c.Request.Header.Set("User-Agent", "cyber-continuation-test/1.0")
				setOpenAICyberContinuationAuth(c, key)
				test.call(h, c)
				return recorder, c
			}

			first, firstContext := request(apiKey, original)
			require.Equal(t, http.StatusForbidden, first.Code, first.Body.String())
			require.NotNil(t, service.GetOpsCyberPolicy(firstContext))
			require.Equal(t, 1, upstream.callCount(), "Cyber must not retry or fail over")
			require.Len(t, usageRepo.snapshot(), 1)

			// Intentionally no Eventually or sleep: the very next HTTP request must
			// observe the block, without waiting for background risk/email work.
			followup, followupContext := request(apiKey, rewritten)
			require.Equal(t, http.StatusForbidden, followup.Code, followup.Body.String())
			require.Contains(t, followup.Body.String(), service.OpenAICyberSessionBlockedClientMessage)
			require.Nil(t, service.GetOpsCyberPolicy(followupContext), "local rejection must not look like a new upstream Cyber hit")
			require.Equal(t, 1, upstream.callCount(), "rewritten continuation must be blocked before forwarding")
			require.Len(t, usageRepo.snapshot(), 1, "local rejections must not create another upstream usage row")

			newConversation, newConversationContext := request(apiKey, independent)
			require.Equal(t, http.StatusForbidden, newConversation.Code, newConversation.Body.String())
			require.NotNil(t, service.GetOpsCyberPolicy(newConversationContext), "independent conversation must still reach upstream")
			require.Equal(t, 2, upstream.callCount())

			otherKey := *apiKey
			otherKey.ID++
			otherResponse, otherContext := request(&otherKey, rewritten)
			require.Equal(t, http.StatusForbidden, otherResponse.Code, otherResponse.Body.String())
			require.NotNil(t, service.GetOpsCyberPolicy(otherContext), "transcript hashes must remain API-key isolated")
			require.Equal(t, 3, upstream.callCount())
			logs := usageRepo.snapshot()
			require.Len(t, logs, 3)
			for _, log := range logs {
				require.Equal(t, service.RequestTypeCyberBlocked, log.RequestType)
			}
		})
	}
}

func TestOpenAIResponsesWebSocketCyberReconnect(t *testing.T) {
	for _, test := range []struct {
		name     string
		warmup   string
		failover bool
		first    string
		followup string
	}{
		{
			name:     "first_frame_prompt_cache_key",
			first:    `{"type":"response.create","model":"gpt-5.5","prompt_cache_key":"cyber-reconnect-session","input":"flagged request"}`,
			followup: `{"type":"response.create","model":"gpt-5.5","prompt_cache_key":"cyber-reconnect-session","input":"rewritten request"}`,
		},
		{
			name:     "transcript_without_explicit_session",
			first:    `{"type":"response.create","model":"gpt-5.5","input":[{"role":"user","content":"hello"},{"role":"assistant","content":"unique websocket context 381"},{"role":"user","content":"flagged request"}]}`,
			followup: `{"type":"response.create","model":"gpt-5.5","input":[{"role":"user","content":"hello"},{"role":"assistant","content":"unique websocket context 381"},{"role":"user","content":"rewritten request"}]}`,
		},
		{
			name:     "second_turn_transcript_after_successful_first_turn",
			warmup:   `{"type":"response.create","model":"gpt-5.5","input":[{"role":"user","content":"hello"}]}`,
			first:    `{"type":"response.create","model":"gpt-5.5","input":[{"role":"user","content":"hello"},{"role":"assistant","content":"unique websocket context 482"},{"role":"user","content":"flagged second turn"}]}`,
			followup: `{"type":"response.create","model":"gpt-5.5","input":[{"role":"user","content":"hello"},{"role":"assistant","content":"unique websocket context 482"},{"role":"user","content":"rewritten second turn"}]}`,
		},
		{
			name:     "cyber_after_passthrough_to_ctx_pool_failover",
			failover: true,
			first:    `{"type":"response.create","model":"gpt-5.5","input":[{"role":"user","content":"hello"},{"role":"assistant","content":"unique failover context 583"},{"role":"user","content":"flagged request"}]}`,
			followup: `{"type":"response.create","model":"gpt-5.5","input":[{"role":"user","content":"hello"},{"role":"assistant","content":"unique failover context 583"},{"role":"user","content":"rewritten request"}]}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var upstreamCalls atomic.Int32
			releaseUpstream := make(chan struct{})
			upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				conn, err := coderws.Accept(w, r, &coderws.AcceptOptions{CompressionMode: coderws.CompressionContextTakeover})
				if err != nil {
					return
				}
				connectionNumber := upstreamCalls.Add(1)
				defer func() { _ = conn.CloseNow() }()
				ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
				defer cancel()
				if _, _, err := conn.Read(ctx); err != nil {
					return
				}
				if test.failover && connectionNumber == 1 {
					if err := conn.Write(ctx, coderws.MessageText, []byte(`{"type":"error","error":{"code":"rate_limit_exceeded","type":"usage_limit_reached","message":"The usage limit has been reached"}}`)); err != nil {
						return
					}
					// Read the gateway close frame so the first account can finish
					// its WebSocket closing handshake before failover starts.
					_, _, _ = conn.Read(ctx)
					return
				}
				if test.warmup != "" {
					if err := conn.Write(ctx, coderws.MessageText, []byte(`{"type":"response.completed","response":{"id":"resp_cyber_warmup","status":"completed","model":"gpt-5.5","output":[],"usage":{"input_tokens":1,"output_tokens":1}}}`)); err != nil {
						return
					}
					if _, _, err := conn.Read(ctx); err != nil {
						return
					}
				}
				if err := conn.Write(ctx, coderws.MessageText, []byte(`{"type":"response.failed","response":{"id":"resp_cyber_reconnect","status":"failed","model":"gpt-5.5","error":{"code":"cyber_policy","message":"blocked"},"usage":{"input_tokens":7,"output_tokens":3}}}`)); err != nil {
					return
				}
				<-releaseUpstream
			}))
			t.Cleanup(func() {
				close(releaseUpstream)
				upstreamServer.CloseClientConnections()
				upstreamServer.Close()
			})
			h, apiKey, _, usageRepo := newOpenAICyberContinuationHandler(t, upstreamServer.URL, test.failover)
			router := gin.New()
			router.Use(func(c *gin.Context) {
				setOpenAICyberContinuationAuth(c, apiKey)
				c.Next()
			})
			handlerDone := make(chan struct{}, 2)
			router.GET("/openai/v1/responses", func(c *gin.Context) {
				defer func() { handlerDone <- struct{}{} }()
				h.ResponsesWebSocket(c)
			})
			handlerServer := httptest.NewServer(router)
			t.Cleanup(handlerServer.Close)
			dialAndWrite := func(body string) *coderws.Conn {
				t.Helper()
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				conn, _, err := coderws.Dial(ctx, "ws"+strings.TrimPrefix(handlerServer.URL, "http")+"/openai/v1/responses", &coderws.DialOptions{
					CompressionMode: coderws.CompressionContextTakeover,
					HTTPHeader:      http.Header{"User-Agent": []string{"cyber-reconnect-test/1.0"}},
				})
				require.NoError(t, err)
				t.Cleanup(func() { _ = conn.CloseNow() })
				require.NoError(t, conn.Write(ctx, coderws.MessageText, []byte(body)))
				return conn
			}
			waitHandler := func() {
				t.Helper()
				select {
				case <-handlerDone:
				case <-time.After(3 * time.Second):
					t.Fatal("WebSocket handler did not finish")
				}
			}

			initialPayload := test.first
			if test.warmup != "" {
				initialPayload = test.warmup
			}
			first := dialAndWrite(initialPayload)
			if test.warmup != "" {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				_, event, err := first.Read(ctx)
				require.NoError(t, err)
				require.Contains(t, string(event), `"type":"response.completed"`)
				err = first.Write(ctx, coderws.MessageText, []byte(test.first))
				cancel()
				require.NoError(t, err)
			}
			readCtx, cancelRead := context.WithTimeout(context.Background(), 3*time.Second)
			_, event, err := first.Read(readCtx)
			cancelRead()
			require.NoError(t, err)
			require.Contains(t, string(event), `"type":"response.failed"`)
			require.NoError(t, first.CloseNow())
			waitHandler()
			wantUpstreamCalls := int32(1)
			if test.failover {
				wantUpstreamCalls++
			}
			require.Equal(t, wantUpstreamCalls, upstreamCalls.Load(), "Cyber itself must not trigger another account failover")

			second := dialAndWrite(test.followup)
			readCtx, cancelRead = context.WithTimeout(context.Background(), 3*time.Second)
			_, event, err = second.Read(readCtx)
			cancelRead()
			require.NoError(t, err)
			require.Contains(t, string(event), "session_blocked_by_cyber_policy")
			readCtx, cancelRead = context.WithTimeout(context.Background(), 3*time.Second)
			_, _, err = second.Read(readCtx)
			cancelRead()
			require.Equal(t, coderws.StatusPolicyViolation, coderws.CloseStatus(err))
			waitHandler()
			require.Equal(t, wantUpstreamCalls, upstreamCalls.Load(), "blocked reconnect must not dial upstream")
			logs := usageRepo.snapshot()
			wantUsageRows := 1
			if test.warmup != "" {
				wantUsageRows++
			}
			require.Len(t, logs, wantUsageRows, "reconnect must not duplicate the original Cyber usage")
			cyberUsage := logs[len(logs)-1]
			require.Equal(t, service.RequestTypeCyberBlocked, cyberUsage.EffectiveRequestType())
			require.Equal(t, 7, cyberUsage.InputTokens)
			require.Equal(t, 3, cyberUsage.OutputTokens)
			if test.failover {
				require.Equal(t, int64(9202), cyberUsage.AccountID, "Cyber must retain the second account's identity after failover")
			}
		})
	}
}
