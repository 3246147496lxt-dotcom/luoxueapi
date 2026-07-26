package handler

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// newTestGinContext builds a bare gin.Context backed by an httptest recorder.
func newTestGinContext() *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c
}

// TestRecordCyberPolicyIfMarked_NoMark verifies that when no cyber mark is set,
// the function returns immediately and does NOT set the recorded flag.
func TestRecordCyberPolicyIfMarked_NoMark(t *testing.T) {
	c := newTestGinContext()
	h := &OpenAIGatewayHandler{}

	h.recordCyberPolicyIfMarked(c, nil, nil, nil, "gpt-5", true, "", service.ChannelUsageFields{}, "")

	// Flag must NOT be set when there was no mark.
	require.False(t, c.GetBool(cyberPolicyRecordedKey),
		"cyberPolicyRecordedKey must remain false when no cyber mark is present")
}

// TestRecordCyberPolicyIfMarked_WithMark verifies that:
//  1. When a cyber mark is present, the recorded flag is set (guard activated).
//  2. A second call is a no-op (idempotent guard).
//  3. Nil services do not panic.
func TestRecordCyberPolicyIfMarked_WithMark(t *testing.T) {
	c := newTestGinContext()
	service.MarkOpsCyberPolicy(c, service.CyberPolicyMark{
		Message:        "flagged",
		Body:           `{"error":{"code":"cyber_policy"}}`,
		UpstreamStatus: 400,
	})

	h := &OpenAIGatewayHandler{} // nil services — must not panic

	// First call: should set the flag.
	require.NotPanics(t, func() {
		h.recordCyberPolicyIfMarked(c, nil, nil, nil, "gpt-5", true, "", service.ChannelUsageFields{}, "")
	})
	require.True(t, c.GetBool(cyberPolicyRecordedKey),
		"cyberPolicyRecordedKey must be true after first call with a mark")

	// Second call: flag already set — must be a no-op (idempotent).
	require.NotPanics(t, func() {
		h.recordCyberPolicyIfMarked(c, nil, nil, nil, "gpt-5", false, "", service.ChannelUsageFields{}, "")
	})
	// Flag should still be true (not toggled or cleared).
	require.True(t, c.GetBool(cyberPolicyRecordedKey),
		"cyberPolicyRecordedKey must remain true after second call (guard)")
}

// TestRecordCyberPolicyIfMarked_ForwardSuccessSkipsUsageLog verifies the semantic:
// when forwardErrored=false the function still sets the guard flag (mark present),
// but the cyber usage row is NOT requested (only RecordCyberPolicyEvent fires).
// Since services are nil here we only verify the guard flag and no panic.
func TestRecordCyberPolicyIfMarked_ForwardSuccessSkipsUsageLog(t *testing.T) {
	c := newTestGinContext()
	service.MarkOpsCyberPolicy(c, service.CyberPolicyMark{
		Message:        "flagged",
		UpstreamStatus: 200,
	})

	h := &OpenAIGatewayHandler{}

	require.NotPanics(t, func() {
		h.recordCyberPolicyIfMarked(c, nil, nil, nil, "gpt-5", false /* forwardErrored=false */, "", service.ChannelUsageFields{}, "")
	})
	require.True(t, c.GetBool(cyberPolicyRecordedKey))
}

// TestClearCyberPolicyTurnState verifies F1 at the handler level: after a turn
// is finalized, both the mark and the recorded guard are reset so the next WS
// turn detects/records independently.
func TestClearCyberPolicyTurnState(t *testing.T) {
	c := newTestGinContext()
	h := &OpenAIGatewayHandler{}

	service.MarkOpsCyberPolicy(c, service.CyberPolicyMark{Message: "turn1", UpstreamStatus: 200})
	h.recordCyberPolicyIfMarked(c, nil, nil, nil, "gpt-5", false, "", service.ChannelUsageFields{}, "")
	require.True(t, c.GetBool(cyberPolicyRecordedKey))

	clearCyberPolicyTurnState(c)
	require.Nil(t, service.GetOpsCyberPolicy(c))
	require.False(t, c.GetBool(cyberPolicyRecordedKey))

	// turn2: a fresh cyber hit must be recordable again.
	service.MarkOpsCyberPolicy(c, service.CyberPolicyMark{Message: "turn2", UpstreamStatus: 200})
	h.recordCyberPolicyIfMarked(c, nil, nil, nil, "gpt-5", false, "", service.ChannelUsageFields{}, "")
	require.True(t, c.GetBool(cyberPolicyRecordedKey))
	require.Equal(t, "turn2", service.GetOpsCyberPolicy(c).Message)
}

func TestReportOpenAIAccountScheduleFailureSkipsCyber(t *testing.T) {
	cfg := &config.Config{}
	settingRepo := &contentModerationHandlerSettingRepo{values: map[string]string{}}
	settingService := service.NewSettingService(settingRepo, cfg)
	setAdvancedScheduler := func(enabled bool) {
		require.NoError(t, settingService.UpdateSettings(context.Background(), &service.SystemSettings{
			OpenAIAdvancedSchedulerEnabled: enabled,
		}))
	}
	setAdvancedScheduler(true)
	t.Cleanup(func() {
		setAdvancedScheduler(false)
	})

	rateLimitService := service.NewRateLimitService(nil, nil, cfg, nil, nil)
	rateLimitService.SetSettingService(settingService)
	gatewayService := service.NewOpenAIGatewayService(
		nil, nil, nil, nil, nil, nil, nil, cfg, nil, nil, nil,
		rateLimitService, nil, nil, nil, nil, nil, nil, nil, nil,
		settingService, nil,
	)
	h := &OpenAIGatewayHandler{gatewayService: gatewayService}
	c := newTestGinContext()
	service.MarkOpsCyberPolicy(c, service.CyberPolicyMark{
		Message:        "blocked",
		UpstreamStatus: 403,
	})

	h.reportOpenAIAccountScheduleFailure(c, 42, "gpt-5.5")
	require.Zero(t, gatewayService.SnapshotOpenAIAccountSchedulerMetrics().RuntimeStatsAccountCount)

	service.ClearOpsCyberPolicy(c)
	h.reportOpenAIAccountScheduleFailure(c, 42, "gpt-5.5")
	require.Equal(t, 1, gatewayService.SnapshotOpenAIAccountSchedulerMetrics().RuntimeStatsAccountCount)
}

type openAIHTTPCyberHandlerUpstream struct {
	service.HTTPUpstream
	mu    sync.Mutex
	calls int
}

func (u *openAIHTTPCyberHandlerUpstream) Do(*http.Request, string, int64, int) (*http.Response, error) {
	u.mu.Lock()
	u.calls++
	u.mu.Unlock()
	return &http.Response{
		StatusCode: http.StatusForbidden,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body: io.NopCloser(strings.NewReader(
			`{"error":{"code":"cyber_policy","message":"blocked"}}`,
		)),
	}, nil
}

func (u *openAIHTTPCyberHandlerUpstream) callCount() int {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.calls
}

type openAIHTTPCyberUsageRepo struct {
	service.UsageLogRepository
	mu   sync.Mutex
	logs []service.UsageLog
}

func (r *openAIHTTPCyberUsageRepo) Create(_ context.Context, log *service.UsageLog) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.logs = append(r.logs, *log)
	return true, nil
}

func (r *openAIHTTPCyberUsageRepo) snapshot() []service.UsageLog {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]service.UsageLog(nil), r.logs...)
}

func TestOpenAIHTTPHandlersCyberPolicyDoesNotPenalizeScheduler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{RunMode: config.RunModeSimple}
	cfg.Default.RateMultiplier = 1
	cfg.Security.URLAllowlist.Enabled = false
	settingRepo := &contentModerationHandlerSettingRepo{values: map[string]string{}}
	settingService := service.NewSettingService(settingRepo, cfg)
	setAdvancedScheduler := func(enabled bool) {
		require.NoError(t, settingService.UpdateSettings(context.Background(), &service.SystemSettings{
			OpenAIAdvancedSchedulerEnabled: enabled,
		}))
	}
	setAdvancedScheduler(true)
	t.Cleanup(func() {
		setAdvancedScheduler(false)
	})

	tests := []struct {
		name   string
		path   string
		body   string
		invoke func(*OpenAIGatewayHandler, *gin.Context)
	}{
		{
			name: "chat_completions",
			path: "/v1/chat/completions",
			body: `{"model":"gpt-5.5","messages":[{"role":"user","content":"hello"}],"stream":false}`,
			invoke: func(h *OpenAIGatewayHandler, c *gin.Context) {
				h.ChatCompletions(c)
			},
		},
		{
			name: "responses",
			path: "/v1/responses",
			body: `{"model":"gpt-5.5","input":"hello","stream":false}`,
			invoke: func(h *OpenAIGatewayHandler, c *gin.Context) {
				h.Responses(c)
			},
		},
		{
			name: "messages",
			path: "/v1/messages",
			body: `{"model":"gpt-5.5","max_tokens":64,"messages":[{"role":"user","content":"hello"}],"stream":false}`,
			invoke: func(h *OpenAIGatewayHandler, c *gin.Context) {
				h.Messages(c)
			},
		},
	}

	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			groupID := int64(7100 + index)
			accountRepo := &openAIWSUsageHandlerAccountRepoStub{account: service.Account{
				ID:          int64(7200 + index),
				Name:        "cyber-handler-account",
				Platform:    service.PlatformOpenAI,
				Type:        service.AccountTypeOAuth,
				Status:      service.StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Credentials: map[string]any{"access_token": "test-token"},
			}}
			usageRepo := &openAIHTTPCyberUsageRepo{}
			upstream := &openAIHTTPCyberHandlerUpstream{}
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
				billingCacheService, upstream, &service.DeferredService{}, nil, nil, nil,
				nil, nil, settingService, nil,
			)
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

			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, test.path, strings.NewReader(test.body))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Set(string(middleware.ContextKeyAPIKey), &service.APIKey{
				ID:      int64(7300 + index),
				GroupID: &groupID,
				User:    &service.User{ID: int64(7400 + index), Status: service.StatusActive},
				Group: &service.Group{
					ID:                    groupID,
					Platform:              service.PlatformOpenAI,
					Status:                service.StatusActive,
					RateMultiplier:        1,
					AllowMessagesDispatch: true,
				},
			})
			c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{
				UserID:      int64(7400 + index),
				Concurrency: 1,
			})

			test.invoke(h, c)

			require.Equal(t, http.StatusForbidden, recorder.Code, recorder.Body.String())
			require.Equal(t, 1, upstream.callCount(), "Cyber must not trigger retry or failover")
			require.NotNil(t, service.GetOpsCyberPolicy(c))
			require.Zero(t, gatewayService.SnapshotOpenAIAccountSchedulerMetrics().RuntimeStatsAccountCount)
			logs := usageRepo.snapshot()
			require.Len(t, logs, 1, "Cyber usage must be recorded exactly once")
			require.Equal(t, service.RequestTypeCyberBlocked, logs[0].RequestType)
		})
	}
}

// TestBuildCyberSessionBlockedOpsEntry verifies the locally-rejected request is
// auditable: 403 / phase=request / type=cyber_policy_session_blocked — distinct
// from upstream cyber_policy hits, and it must NOT touch moderation/violation.
func TestBuildCyberSessionBlockedOpsEntry(t *testing.T) {
	entry := buildCyberSessionBlockedOpsEntry(cyberPolicyOpsErrorMeta{
		RequestID: "req-9", Model: "gpt-5", RequestPath: "/openai/v1/responses",
	})
	require.Equal(t, 403, entry.StatusCode)
	require.Equal(t, "cyber_policy_session_blocked", entry.ErrorType)
	require.Equal(t, "request", entry.ErrorPhase)
	require.True(t, entry.IsBusinessLimited)
	require.Equal(t, "gateway_local", entry.ErrorSource)
	require.Equal(t, "platform", entry.ErrorOwner)
	require.Empty(t, entry.ErrorBody, "no session block key → ErrorBody must be empty")

	entryWithKey := buildCyberSessionBlockedOpsEntry(cyberPolicyOpsErrorMeta{
		RequestID: "req-9", Model: "gpt-5", RequestPath: "/openai/v1/responses",
		SessionBlockKey: "abc123",
	})
	require.Equal(t, "session_block_key=abc123", entryWithKey.ErrorBody)
}

// TestRejectIfCyberSessionBlocked_FailOpen verifies fail-open paths: nil handler
// services, no explicit session signal, and (implicitly) disabled switch all
// pass the request through.
func TestRejectIfCyberSessionBlocked_FailOpen(t *testing.T) {
	c := newTestGinContext()
	c.Request = httptest.NewRequest("POST", "/openai/v1/responses", strings.NewReader(`{}`))

	h := &OpenAIGatewayHandler{}
	require.False(t, h.rejectIfCyberSessionBlocked(c, nil, []byte(`{}`), "gpt-5", cyberBlockFormatResponses), "nil apiKey → pass")

	h2 := &OpenAIGatewayHandler{gatewayService: nil}
	key := &service.APIKey{ID: 1}
	require.False(t, h2.rejectIfCyberSessionBlocked(c, key, []byte(`{}`), "gpt-5", cyberBlockFormatResponses), "nil gateway service → pass")
}

// TestRecordCyberPolicyIfMarked_BlockKeyPlumbed verifies the 6th param is
// accepted and a non-empty key with nil gateway service does not panic
// (write-side guards live in the service layer).
func TestRecordCyberPolicyIfMarked_BlockKeyPlumbed(t *testing.T) {
	c := newTestGinContext()
	service.MarkOpsCyberPolicy(c, service.CyberPolicyMark{Message: "x", UpstreamStatus: 400})
	h := &OpenAIGatewayHandler{}
	require.NotPanics(t, func() {
		h.recordCyberPolicyIfMarked(c, nil, nil, nil, "gpt-5", true, "deadbeef", service.ChannelUsageFields{}, "")
	})
}

// TestBuildCyberPolicyOpsErrorEntry_StatusCode verifies F6: the ops error log
// records the status the codex client actually received (400 non-stream / 200 stream),
// not a hardcoded 403.
func TestBuildCyberPolicyOpsErrorEntry_StatusCode(t *testing.T) {
	for _, tc := range []struct {
		name           string
		upstreamStatus int
	}{
		{"non_stream_400", 400},
		{"stream_200", 200},
		{"zero_value", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mark := &service.CyberPolicyMark{
				Code:           "cyber_policy",
				Message:        "blocked",
				UpstreamStatus: tc.upstreamStatus,
			}
			entry := buildCyberPolicyOpsErrorEntry(cyberPolicyOpsErrorMeta{
				RequestID: "req-1", Model: "gpt-5", RequestPath: "/openai/v1/responses",
			}, mark)
			require.Equal(t, tc.upstreamStatus, entry.StatusCode)
			require.Equal(t, "cyber_policy", entry.ErrorType)
			require.Equal(t, "request", entry.ErrorPhase)
		})
	}
}
