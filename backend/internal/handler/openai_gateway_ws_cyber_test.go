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

func TestOpenAIResponsesWebSocketCyberSchedulerContract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name                 string
		upstreamEvent        string
		sendFollowup         bool
		wantRuntimeAccounts  int
		wantUsageRequestType service.RequestType
	}{
		{
			name:                 "cyber turn and policy close do not penalize scheduler",
			upstreamEvent:        `{"type":"response.failed","response":{"id":"resp_ws_cyber","status":"failed","model":"gpt-5.5","error":{"code":"cyber_policy","message":"blocked"},"usage":{"input_tokens":2,"output_tokens":1}}}`,
			sendFollowup:         true,
			wantRuntimeAccounts:  0,
			wantUsageRequestType: service.RequestTypeCyberBlocked,
		},
		{
			name:                 "ordinary failed turn still penalizes scheduler",
			upstreamEvent:        `{"type":"response.failed","response":{"id":"resp_ws_failed","status":"failed","model":"gpt-5.5","error":{"code":"server_error","message":"temporary failure"},"usage":{"input_tokens":2,"output_tokens":1}}}`,
			wantRuntimeAccounts:  1,
			wantUsageRequestType: service.RequestTypeWSV2,
		},
	}

	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var upstreamCalls atomic.Int32
			releaseUpstream := make(chan struct{})
			upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				conn, err := coderws.Accept(w, r, &coderws.AcceptOptions{
					CompressionMode: coderws.CompressionContextTakeover,
				})
				if err != nil {
					return
				}
				upstreamCalls.Add(1)
				defer func() { _ = conn.CloseNow() }()

				readCtx, cancelRead := context.WithTimeout(r.Context(), 3*time.Second)
				_, _, err = conn.Read(readCtx)
				cancelRead()
				if err != nil {
					return
				}

				writeCtx, cancelWrite := context.WithTimeout(r.Context(), 3*time.Second)
				err = conn.Write(writeCtx, coderws.MessageText, []byte(test.upstreamEvent))
				cancelWrite()
				if err != nil {
					return
				}
				<-releaseUpstream
			}))
			t.Cleanup(func() {
				close(releaseUpstream)
				upstreamServer.CloseClientConnections()
				upstreamServer.Close()
			})

			cfg := &config.Config{RunMode: config.RunModeSimple}
			cfg.Default.RateMultiplier = 1
			cfg.Security.URLAllowlist.Enabled = false
			cfg.Security.URLAllowlist.AllowInsecureHTTP = true
			cfg.Gateway.OpenAIWS.Enabled = true
			cfg.Gateway.OpenAIWS.APIKeyEnabled = true
			cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true
			cfg.Gateway.OpenAIWS.IngressModeDefault = service.OpenAIWSIngressModeCtxPool
			cfg.Gateway.OpenAIWS.DialTimeoutSeconds = 3
			cfg.Gateway.OpenAIWS.ReadTimeoutSeconds = 3
			cfg.Gateway.OpenAIWS.WriteTimeoutSeconds = 3
			cfg.Gateway.MaxAccountSwitches = 3

			settingRepo := &openAIWSCyberSettingRepo{values: map[string]string{}}
			settingService := service.NewSettingService(settingRepo, cfg)
			require.NoError(t, settingService.UpdateSettings(context.Background(), &service.SystemSettings{
				OpenAIAdvancedSchedulerEnabled: true,
			}))
			t.Cleanup(func() {
				require.NoError(t, settingService.UpdateSettings(context.Background(), &service.SystemSettings{
					OpenAIAdvancedSchedulerEnabled: false,
				}))
			})

			groupID := int64(8100 + index)
			account := service.Account{
				ID:          int64(8200 + index),
				Name:        "openai-ws-cyber-scheduler-contract",
				Platform:    service.PlatformOpenAI,
				Type:        service.AccountTypeAPIKey,
				Status:      service.StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Credentials: map[string]any{
					"api_key":  "sk-test",
					"base_url": upstreamServer.URL,
				},
				Extra: map[string]any{
					"openai_apikey_responses_websockets_v2_enabled": true,
				},
			}
			accountRepo := &openAIWSUsageHandlerAccountRepoStub{account: account}
			usageRepo := &openAIHTTPCyberUsageRepo{}
			rateLimitService := service.NewRateLimitService(accountRepo, usageRepo, cfg, nil, nil)
			rateLimitService.SetSettingService(settingService)
			concurrencyCache := &concurrencyCacheMock{
				acquireUserSlotFn: func(context.Context, int64, int, string) (bool, error) {
					return true, nil
				},
				acquireAccountSlotFn: func(context.Context, int64, int, string) (bool, error) {
					return true, nil
				},
			}
			concurrencyService := service.NewConcurrencyService(concurrencyCache)
			billingCacheService := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg, nil)
			t.Cleanup(billingCacheService.Stop)
			gatewayService := service.NewOpenAIGatewayService(
				accountRepo, usageRepo, nil, nil, nil, nil, nil, cfg, nil,
				concurrencyService, service.NewBillingService(cfg, nil), rateLimitService,
				billingCacheService, nil, &service.DeferredService{}, nil, nil, nil,
				nil, nil, settingService, nil,
			)
			t.Cleanup(gatewayService.CloseOpenAIWSPool)
			h := NewOpenAIGatewayHandler(
				gatewayService,
				concurrencyService,
				billingCacheService,
				service.NewAPIKeyService(nil, nil, nil, nil, nil, nil, cfg),
				nil,
				nil,
				nil,
				nil,
				cfg,
			)

			apiKey := &service.APIKey{
				ID:      int64(8300 + index),
				GroupID: &groupID,
				User:    &service.User{ID: int64(8400 + index), Status: service.StatusActive},
				Group: &service.Group{
					ID:             groupID,
					Platform:       service.PlatformOpenAI,
					Status:         service.StatusActive,
					RateMultiplier: 1,
				},
			}
			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Set(string(middleware.ContextKeyAPIKey), apiKey)
				c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{
					UserID:      apiKey.User.ID,
					Concurrency: 1,
				})
				c.Next()
			})
			router.GET("/openai/v1/responses", h.ResponsesWebSocket)
			handlerServer := httptest.NewServer(router)
			t.Cleanup(handlerServer.Close)

			dialCtx, cancelDial := context.WithTimeout(context.Background(), 3*time.Second)
			clientConn, _, err := coderws.Dial(
				dialCtx,
				"ws"+strings.TrimPrefix(handlerServer.URL, "http")+"/openai/v1/responses",
				&coderws.DialOptions{CompressionMode: coderws.CompressionContextTakeover},
			)
			cancelDial()
			require.NoError(t, err)
			t.Cleanup(func() { _ = clientConn.CloseNow() })

			writeCtx, cancelWrite := context.WithTimeout(context.Background(), 3*time.Second)
			err = clientConn.Write(writeCtx, coderws.MessageText, []byte(
				`{"type":"response.create","model":"gpt-5.5","input":"hello"}`,
			))
			cancelWrite()
			require.NoError(t, err)

			readCtx, cancelRead := context.WithTimeout(context.Background(), 3*time.Second)
			_, event, err := clientConn.Read(readCtx)
			cancelRead()
			require.NoError(t, err)
			require.Contains(t, string(event), `"type":"response.failed"`)

			if test.sendFollowup {
				writeCtx, cancelWrite = context.WithTimeout(context.Background(), 3*time.Second)
				err = clientConn.Write(writeCtx, coderws.MessageText, []byte(
					`{"type":"response.create","model":"gpt-5.5","input":"again"}`,
				))
				cancelWrite()
				require.NoError(t, err)

				readCtx, cancelRead = context.WithTimeout(context.Background(), 3*time.Second)
				_, _, err = clientConn.Read(readCtx)
				cancelRead()
				require.Error(t, err)
				require.Equal(t, coderws.StatusPolicyViolation, coderws.CloseStatus(err))
			} else {
				_ = clientConn.Close(coderws.StatusNormalClosure, "done")
			}

			require.Equal(t, int32(1), upstreamCalls.Load(), "follow-up Cyber turn must not reach upstream")
			require.Equal(
				t,
				test.wantRuntimeAccounts,
				gatewayService.SnapshotOpenAIAccountSchedulerMetrics().RuntimeStatsAccountCount,
			)
			logs := usageRepo.snapshot()
			require.Len(t, logs, 1, "each terminal WS turn must persist exactly one usage row")
			require.Equal(t, test.wantUsageRequestType, logs[0].EffectiveRequestType())
			require.Equal(t, 2, logs[0].InputTokens)
			require.Equal(t, 1, logs[0].OutputTokens)
		})
	}
}

type openAIWSCyberSettingRepo struct {
	mu     sync.RWMutex
	values map[string]string
}

func (r *openAIWSCyberSettingRepo) Get(_ context.Context, key string) (*service.Setting, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if value, ok := r.values[key]; ok {
		return &service.Setting{Key: key, Value: value}, nil
	}
	return nil, service.ErrSettingNotFound
}

func (r *openAIWSCyberSettingRepo) GetValue(_ context.Context, key string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if value, ok := r.values[key]; ok {
		return value, nil
	}
	return "", service.ErrSettingNotFound
}

func (r *openAIWSCyberSettingRepo) Set(_ context.Context, key, value string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.values == nil {
		r.values = map[string]string{}
	}
	r.values[key] = value
	return nil
}

func (r *openAIWSCyberSettingRepo) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := r.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (r *openAIWSCyberSettingRepo) SetMultiple(_ context.Context, settings map[string]string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.values == nil {
		r.values = map[string]string{}
	}
	for key, value := range settings {
		r.values[key] = value
	}
	return nil
}

func (r *openAIWSCyberSettingRepo) GetAll(_ context.Context) (map[string]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]string, len(r.values))
	for key, value := range r.values {
		out[key] = value
	}
	return out, nil
}

func (r *openAIWSCyberSettingRepo) Delete(_ context.Context, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.values, key)
	return nil
}
