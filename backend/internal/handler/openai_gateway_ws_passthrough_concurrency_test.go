package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type openAIWSPassthroughSlotSnapshot struct {
	userAcquires                []string
	userReleases                []string
	accountAcquires             []string
	accountReleases             []string
	releaseDuringBlockedAcquire bool
}

type openAIWSPassthroughConcurrencyCache struct {
	*concurrencyCacheMock

	mu                          sync.Mutex
	userAcquires                []string
	userReleases                []string
	accountAcquires             []string
	accountReleases             []string
	blockNextAccountAcquire     bool
	blockedAccountAcquireActive bool
	blockedAccountAcquireEnter  chan struct{}
	unblockAccountAcquire       chan struct{}
	releaseDuringBlockedAcquire bool
}

func newOpenAIWSPassthroughConcurrencyCache() *openAIWSPassthroughConcurrencyCache {
	return &openAIWSPassthroughConcurrencyCache{
		concurrencyCacheMock: &concurrencyCacheMock{},
	}
}

func (c *openAIWSPassthroughConcurrencyCache) AcquireUserSlot(
	_ context.Context,
	_ int64,
	_ int,
	requestID string,
) (bool, error) {
	c.mu.Lock()
	c.userAcquires = append(c.userAcquires, requestID)
	c.mu.Unlock()
	return true, nil
}

func (c *openAIWSPassthroughConcurrencyCache) ReleaseUserSlot(
	_ context.Context,
	_ int64,
	requestID string,
) error {
	c.mu.Lock()
	c.userReleases = append(c.userReleases, requestID)
	if c.blockedAccountAcquireActive {
		c.releaseDuringBlockedAcquire = true
	}
	c.mu.Unlock()
	return nil
}

func (c *openAIWSPassthroughConcurrencyCache) AcquireAccountSlot(
	_ context.Context,
	_ int64,
	_ int,
	requestID string,
) (bool, error) {
	c.mu.Lock()
	c.accountAcquires = append(c.accountAcquires, requestID)
	shouldBlock := c.blockNextAccountAcquire
	if shouldBlock {
		c.blockNextAccountAcquire = false
		c.blockedAccountAcquireActive = true
	}
	entered := c.blockedAccountAcquireEnter
	unblock := c.unblockAccountAcquire
	c.mu.Unlock()

	if shouldBlock {
		close(entered)
		<-unblock

		c.mu.Lock()
		c.blockedAccountAcquireActive = false
		c.mu.Unlock()
	}
	return true, nil
}

func (c *openAIWSPassthroughConcurrencyCache) ReleaseAccountSlot(
	_ context.Context,
	_ int64,
	requestID string,
) error {
	c.mu.Lock()
	c.accountReleases = append(c.accountReleases, requestID)
	if c.blockedAccountAcquireActive {
		c.releaseDuringBlockedAcquire = true
	}
	c.mu.Unlock()
	return nil
}

func (c *openAIWSPassthroughConcurrencyCache) blockNextAccountSlotAcquire() (<-chan struct{}, func()) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.blockNextAccountAcquire = true
	c.blockedAccountAcquireEnter = make(chan struct{})
	c.unblockAccountAcquire = make(chan struct{})
	entered := c.blockedAccountAcquireEnter
	unblock := c.unblockAccountAcquire
	var once sync.Once
	return entered, func() {
		once.Do(func() {
			close(unblock)
		})
	}
}

func (c *openAIWSPassthroughConcurrencyCache) snapshot() openAIWSPassthroughSlotSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()

	return openAIWSPassthroughSlotSnapshot{
		userAcquires:                append([]string(nil), c.userAcquires...),
		userReleases:                append([]string(nil), c.userReleases...),
		accountAcquires:             append([]string(nil), c.accountAcquires...),
		accountReleases:             append([]string(nil), c.accountReleases...),
		releaseDuringBlockedAcquire: c.releaseDuringBlockedAcquire,
	}
}

func (c *openAIWSPassthroughConcurrencyCache) releasedGenerations(generations int) bool {
	snapshot := c.snapshot()
	return len(snapshot.userReleases) == generations &&
		len(snapshot.accountReleases) == generations
}

func requireOpenAIWSPassthroughSlotBalance(
	t *testing.T,
	cache *openAIWSPassthroughConcurrencyCache,
	generations int,
) {
	t.Helper()

	snapshot := cache.snapshot()
	require.Len(t, snapshot.userAcquires, generations)
	require.Len(t, snapshot.userReleases, generations)
	require.Len(t, snapshot.accountAcquires, generations)
	require.Len(t, snapshot.accountReleases, generations)
	require.Equal(t, snapshot.userAcquires, snapshot.userReleases, "每代用户槽位必须使用同一 request ID 释放")
	require.Equal(t, snapshot.accountAcquires, snapshot.accountReleases, "每代账号槽位必须使用同一 request ID 释放")
	require.False(t, snapshot.releaseDuringBlockedAcquire, "fallback 不得与正在执行的槽位获取并发释放")

	for _, requestIDs := range [][]string{snapshot.userAcquires, snapshot.accountAcquires} {
		for _, requestID := range requestIDs {
			require.NotEmpty(t, requestID)
		}
		if generations > 1 {
			require.NotEqual(t, requestIDs[0], requestIDs[1], "不同 turn 必须使用不同的 request ID")
		}
	}
}

type openAIWSPassthroughConcurrencyHarness struct {
	clientConn     *coderws.Conn
	handlerDone    <-chan struct{}
	usageRepo      *openAIWSUsageHandlerUsageLogRepoStub
	gatewayService *service.OpenAIGatewayService
}

func newOpenAIWSPassthroughConcurrencyClient(
	t *testing.T,
	upstreamURL string,
	cache *openAIWSPassthroughConcurrencyCache,
) (*coderws.Conn, <-chan struct{}) {
	t.Helper()
	harness := newOpenAIWSPassthroughConcurrencyHarness(t, upstreamURL, cache)
	return harness.clientConn, harness.handlerDone
}

func newOpenAIWSPassthroughConcurrencyHarness(
	t *testing.T,
	upstreamURL string,
	cache *openAIWSPassthroughConcurrencyCache,
) *openAIWSPassthroughConcurrencyHarness {
	t.Helper()
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{RunMode: config.RunModeSimple}
	cfg.Default.RateMultiplier = 1
	cfg.Security.URLAllowlist.Enabled = false
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	cfg.Gateway.OpenAIWS.Enabled = true
	cfg.Gateway.OpenAIWS.APIKeyEnabled = true
	cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true
	cfg.Gateway.OpenAIWS.ModeRouterV2Enabled = true
	cfg.Gateway.OpenAIWS.DialTimeoutSeconds = 3
	cfg.Gateway.OpenAIWS.ReadTimeoutSeconds = 3
	cfg.Gateway.OpenAIWS.WriteTimeoutSeconds = 3
	cfg.Gateway.MaxAccountSwitches = 1

	groupID := int64(9501)
	account := service.Account{
		ID:          9502,
		Name:        "openai-ws-passthrough-concurrency-contract",
		Platform:    service.PlatformOpenAI,
		Type:        service.AccountTypeAPIKey,
		Status:      service.StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  "sk-test",
			"base_url": upstreamURL,
		},
		Extra: map[string]any{
			"openai_apikey_responses_websockets_v2_enabled": true,
			"openai_apikey_responses_websockets_v2_mode":    service.OpenAIWSIngressModePassthrough,
		},
	}
	accountRepo := &openAIWSUsageHandlerAccountRepoStub{account: account}
	usageRepo := &openAIWSUsageHandlerUsageLogRepoStub{created: make(chan *service.UsageLog, 8)}
	concurrencyService := service.NewConcurrencyService(cache)
	billingCacheService := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(billingCacheService.Stop)
	settingRepo := &openAIWSCyberSettingRepo{values: map[string]string{}}
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
	rateLimitService := service.NewRateLimitService(accountRepo, usageRepo, cfg, nil, nil)
	rateLimitService.SetSettingService(settingService)
	gatewayService := service.NewOpenAIGatewayService(
		accountRepo,
		usageRepo,
		nil,
		nil,
		nil,
		nil,
		nil,
		cfg,
		nil,
		concurrencyService,
		service.NewBillingService(cfg, nil),
		rateLimitService,
		billingCacheService,
		nil,
		&service.DeferredService{},
		nil,
		nil,
		nil,
		nil,
		nil,
		settingService,
		nil,
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
		ID:      9503,
		GroupID: &groupID,
		User:    &service.User{ID: 9504, Status: service.StatusActive},
		Group: &service.Group{
			ID:             groupID,
			Platform:       service.PlatformOpenAI,
			Status:         service.StatusActive,
			RateMultiplier: 1,
		},
	}
	handlerDone := make(chan struct{})
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyAPIKey), apiKey)
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{
			UserID:      apiKey.User.ID,
			Concurrency: 1,
		})
		c.Next()
	})
	router.GET("/openai/v1/responses", func(c *gin.Context) {
		defer close(handlerDone)
		h.ResponsesWebSocket(c)
	})
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
	t.Cleanup(func() {
		_ = clientConn.CloseNow()
	})
	return &openAIWSPassthroughConcurrencyHarness{
		clientConn:     clientConn,
		handlerDone:    handlerDone,
		usageRepo:      usageRepo,
		gatewayService: gatewayService,
	}
}

func writeOpenAIWSPassthroughTestMessage(t *testing.T, conn *coderws.Conn, payload string) {
	t.Helper()

	writeCtx, cancelWrite := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelWrite()
	require.NoError(t, conn.Write(writeCtx, coderws.MessageText, []byte(payload)))
}

func readOpenAIWSPassthroughTestMessage(t *testing.T, conn *coderws.Conn) []byte {
	t.Helper()

	readCtx, cancelRead := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelRead()
	msgType, payload, err := conn.Read(readCtx)
	require.NoError(t, err)
	require.Equal(t, coderws.MessageText, msgType)
	return payload
}

func readOpenAIWSPassthroughUsageLog(
	t *testing.T,
	repo *openAIWSUsageHandlerUsageLogRepoStub,
) *service.UsageLog {
	t.Helper()
	require.NotNil(t, repo)
	select {
	case usageLog := <-repo.created:
		require.NotNil(t, usageLog)
		return usageLog
	case <-time.After(3 * time.Second):
		t.Fatal("等待 passthrough usage 写入超时")
		return nil
	}
}

func requireNoOpenAIWSPassthroughUsageLog(
	t *testing.T,
	repo *openAIWSUsageHandlerUsageLogRepoStub,
) {
	t.Helper()
	require.NotNil(t, repo)
	select {
	case usageLog := <-repo.created:
		t.Fatalf("unexpected passthrough usage log: %+v", usageLog)
	default:
	}
}

func TestOpenAIResponsesWebSocket_PassthroughRotatesTurnConcurrencySlots(t *testing.T) {
	upstreamPayloads := make(chan []byte, 2)
	upstreamErrCh := make(chan error, 1)
	upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := coderws.Accept(w, r, &coderws.AcceptOptions{
			CompressionMode: coderws.CompressionContextTakeover,
		})
		if err != nil {
			upstreamErrCh <- err
			return
		}
		defer func() {
			_ = conn.CloseNow()
		}()

		terminalEvents := [][]byte{
			[]byte(`{"type":"response.completed","response":{"id":"resp_slot_turn_1","model":"gpt-5.5","usage":{"input_tokens":2,"output_tokens":1}}}`),
			[]byte(`{"type":"response.completed","response":{"id":"resp_slot_turn_2","model":"gpt-5.5","usage":{"input_tokens":3,"output_tokens":2}}}`),
		}
		for _, terminalEvent := range terminalEvents {
			readCtx, cancelRead := context.WithTimeout(r.Context(), 3*time.Second)
			msgType, payload, readErr := conn.Read(readCtx)
			cancelRead()
			if readErr != nil {
				upstreamErrCh <- readErr
				return
			}
			if msgType != coderws.MessageText && msgType != coderws.MessageBinary {
				upstreamErrCh <- coderws.CloseError{
					Code:   coderws.StatusUnsupportedData,
					Reason: "unexpected upstream message type",
				}
				return
			}
			upstreamPayloads <- append([]byte(nil), payload...)

			writeCtx, cancelWrite := context.WithTimeout(r.Context(), 3*time.Second)
			writeErr := conn.Write(writeCtx, coderws.MessageText, terminalEvent)
			cancelWrite()
			if writeErr != nil {
				upstreamErrCh <- writeErr
				return
			}
		}
		_ = conn.Close(coderws.StatusNormalClosure, "done")
		upstreamErrCh <- nil
	}))
	t.Cleanup(upstreamServer.Close)

	cache := newOpenAIWSPassthroughConcurrencyCache()
	harness := newOpenAIWSPassthroughConcurrencyHarness(t, upstreamServer.URL, cache)
	clientConn := harness.clientConn
	handlerDone := harness.handlerDone

	writeOpenAIWSPassthroughTestMessage(
		t,
		clientConn,
		`{"type":"response.create","model":"gpt-5.5","input":"first"}`,
	)
	firstTerminal := readOpenAIWSPassthroughTestMessage(t, clientConn)
	require.Equal(t, "response.completed", gjson.GetBytes(firstTerminal, "type").String())
	require.Equal(t, "resp_slot_turn_1", gjson.GetBytes(firstTerminal, "response.id").String())
	require.Eventually(t, func() bool {
		return cache.releasedGenerations(1)
	}, 3*time.Second, 10*time.Millisecond, "首轮 terminal 后应释放首代用户和账号槽位")

	writeOpenAIWSPassthroughTestMessage(
		t,
		clientConn,
		`{"type":"response.create","model":"gpt-5.5","previous_response_id":"resp_slot_turn_1","input":"second"}`,
	)
	secondTerminal := readOpenAIWSPassthroughTestMessage(t, clientConn)
	require.Equal(t, "response.completed", gjson.GetBytes(secondTerminal, "type").String())
	require.Equal(t, "resp_slot_turn_2", gjson.GetBytes(secondTerminal, "response.id").String())

	select {
	case <-handlerDone:
	case <-time.After(3 * time.Second):
		t.Fatal("两轮 passthrough 上游正常关闭后 handler 未结束")
	}
	requireOpenAIWSPassthroughSlotBalance(t, cache, 2)

	firstUpstreamPayload := <-upstreamPayloads
	secondUpstreamPayload := <-upstreamPayloads
	require.Equal(t, "response.create", gjson.GetBytes(firstUpstreamPayload, "type").String())
	require.Equal(t, "response.create", gjson.GetBytes(secondUpstreamPayload, "type").String())
	require.Equal(t, "resp_slot_turn_1", gjson.GetBytes(secondUpstreamPayload, "previous_response_id").String())

	select {
	case upstreamErr := <-upstreamErrCh:
		require.NoError(t, upstreamErr)
	case <-time.After(3 * time.Second):
		t.Fatal("等待 passthrough 测试上游结束超时")
	}
}

func TestOpenAIResponsesWebSocket_PassthroughFallbackWaitsForTurnSlotAcquire(t *testing.T) {
	closeUpstream := make(chan struct{})
	upstreamClosed := make(chan struct{})
	upstreamErrCh := make(chan error, 1)
	upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := coderws.Accept(w, r, &coderws.AcceptOptions{
			CompressionMode: coderws.CompressionContextTakeover,
		})
		if err != nil {
			upstreamErrCh <- err
			return
		}
		defer func() {
			_ = conn.CloseNow()
		}()

		readCtx, cancelRead := context.WithTimeout(r.Context(), 3*time.Second)
		_, _, readErr := conn.Read(readCtx)
		cancelRead()
		if readErr != nil {
			upstreamErrCh <- readErr
			return
		}
		writeCtx, cancelWrite := context.WithTimeout(r.Context(), 3*time.Second)
		writeErr := conn.Write(
			writeCtx,
			coderws.MessageText,
			[]byte(`{"type":"response.completed","response":{"id":"resp_blocked_slot_turn_1","model":"gpt-5.5","usage":{"input_tokens":2,"output_tokens":1}}}`),
		)
		cancelWrite()
		if writeErr != nil {
			upstreamErrCh <- writeErr
			return
		}

		<-closeUpstream
		_ = conn.Close(coderws.StatusInternalError, "forced upstream failure")
		close(upstreamClosed)
		upstreamErrCh <- nil
	}))
	var closeUpstreamOnce sync.Once
	stopUpstream := func() {
		closeUpstreamOnce.Do(func() {
			close(closeUpstream)
		})
	}
	t.Cleanup(stopUpstream)
	t.Cleanup(upstreamServer.Close)

	cache := newOpenAIWSPassthroughConcurrencyCache()
	harness := newOpenAIWSPassthroughConcurrencyHarness(t, upstreamServer.URL, cache)
	clientConn := harness.clientConn
	handlerDone := harness.handlerDone

	writeOpenAIWSPassthroughTestMessage(
		t,
		clientConn,
		`{"type":"response.create","model":"gpt-5.5","input":"first"}`,
	)
	firstTerminal := readOpenAIWSPassthroughTestMessage(t, clientConn)
	require.Equal(t, "response.completed", gjson.GetBytes(firstTerminal, "type").String())
	require.Eventually(t, func() bool {
		return cache.releasedGenerations(1)
	}, 3*time.Second, 10*time.Millisecond, "首轮 terminal 后应释放首代槽位")
	firstUsage := readOpenAIWSPassthroughUsageLog(t, harness.usageRepo)
	require.Equal(t, "resp_blocked_slot_turn_1", firstUsage.RequestID)

	accountAcquireEntered, unblockAccountAcquire := cache.blockNextAccountSlotAcquire()
	t.Cleanup(unblockAccountAcquire)
	writeOpenAIWSPassthroughTestMessage(
		t,
		clientConn,
		`{"type":"response.create","model":"gpt-5.5","previous_response_id":"resp_blocked_slot_turn_1","input":"second"}`,
	)
	select {
	case <-accountAcquireEntered:
	case <-time.After(3 * time.Second):
		t.Fatal("第二轮账号槽位获取未进入阻塞点")
	}

	stopUpstream()
	select {
	case <-upstreamClosed:
	case <-time.After(3 * time.Second):
		t.Fatal("测试上游未按预期断开")
	}

	select {
	case <-handlerDone:
		t.Fatal("第二轮 begin hook 仍阻塞时 error fallback 不得让 handler 提前返回")
	case <-time.After(500 * time.Millisecond):
	}
	blockedSnapshot := cache.snapshot()
	require.Len(t, blockedSnapshot.userAcquires, 2)
	require.Len(t, blockedSnapshot.accountAcquires, 2)
	require.Len(t, blockedSnapshot.userReleases, 1, "fallback 不得提前释放仍在 begin hook 内的第二代用户槽位")
	require.Len(t, blockedSnapshot.accountReleases, 1, "fallback 不得提前释放仍在 begin hook 内的第二代账号槽位")
	require.False(t, blockedSnapshot.releaseDuringBlockedAcquire)

	unblockAccountAcquire()
	require.Eventually(t, func() bool {
		return cache.releasedGenerations(2)
	}, 3*time.Second, 10*time.Millisecond, "解除 begin hook 阻塞后 fallback 应精确释放第二代槽位")
	requireOpenAIWSPassthroughSlotBalance(t, cache, 2)

	closeReadCtx, cancelCloseRead := context.WithTimeout(context.Background(), 3*time.Second)
	_, _, clientCloseErr := clientConn.Read(closeReadCtx)
	cancelCloseRead()
	require.Error(t, clientCloseErr)
	require.Equal(t, coderws.StatusInternalError, coderws.CloseStatus(clientCloseErr))

	select {
	case <-handlerDone:
	case <-time.After(3 * time.Second):
		t.Fatal("error fallback 释放第二代槽位后 handler 未结束")
	}

	select {
	case upstreamErr := <-upstreamErrCh:
		require.NoError(t, upstreamErr)
	case <-time.After(3 * time.Second):
		t.Fatal("等待错误交错测试上游结束超时")
	}
	requireNoOpenAIWSPassthroughUsageLog(t, harness.usageRepo)
	schedulerMetrics := harness.gatewayService.SnapshotOpenAIAccountSchedulerMetrics()
	require.Equal(t, int64(1), schedulerMetrics.ReportSuccessTotal)
	require.Equal(t, int64(1), schedulerMetrics.ReportFailureTotal)
}

func TestOpenAIResponsesWebSocket_PassthroughClientDisconnectWithoutTerminalIsNeutral(t *testing.T) {
	upstreamReceived := make(chan struct{})
	upstreamDone := make(chan error, 1)
	upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := coderws.Accept(w, r, nil)
		if err != nil {
			upstreamDone <- err
			return
		}
		defer func() {
			_ = conn.CloseNow()
		}()

		readCtx, cancelRead := context.WithTimeout(r.Context(), 3*time.Second)
		_, _, readErr := conn.Read(readCtx)
		cancelRead()
		if readErr != nil {
			upstreamDone <- readErr
			return
		}
		close(upstreamReceived)

		closeCtx, cancelClose := context.WithTimeout(r.Context(), 5*time.Second)
		_, _, _ = conn.Read(closeCtx)
		cancelClose()
		upstreamDone <- nil
	}))
	t.Cleanup(upstreamServer.Close)

	cache := newOpenAIWSPassthroughConcurrencyCache()
	harness := newOpenAIWSPassthroughConcurrencyHarness(t, upstreamServer.URL, cache)
	writeOpenAIWSPassthroughTestMessage(
		t,
		harness.clientConn,
		`{"type":"response.create","model":"gpt-5.5","input":"disconnect before terminal"}`,
	)
	select {
	case <-upstreamReceived:
	case <-time.After(3 * time.Second):
		t.Fatal("测试上游未收到首轮 response.create")
	}

	clientCloseDone := make(chan error, 1)
	go func() {
		clientCloseDone <- harness.clientConn.Close(coderws.StatusNormalClosure, "done")
	}()
	select {
	case <-harness.handlerDone:
	case <-time.After(4 * time.Second):
		t.Fatal("客户端断开后的 neutral drain 未在限定时间结束")
	}
	select {
	case <-clientCloseDone:
	case <-time.After(2 * time.Second):
		t.Fatal("客户端关闭握手未结束")
	}

	requireOpenAIWSPassthroughSlotBalance(t, cache, 1)
	requireNoOpenAIWSPassthroughUsageLog(t, harness.usageRepo)
	schedulerMetrics := harness.gatewayService.SnapshotOpenAIAccountSchedulerMetrics()
	require.Zero(t, schedulerMetrics.ReportSuccessTotal)
	require.Zero(t, schedulerMetrics.ReportFailureTotal)

	select {
	case upstreamErr := <-upstreamDone:
		require.NoError(t, upstreamErr)
	case <-time.After(3 * time.Second):
		t.Fatal("等待 neutral 测试上游结束超时")
	}
}

func TestOpenAIResponsesWebSocket_PassthroughUpstreamDisconnectWithoutTerminalFails(t *testing.T) {
	upstreamDone := make(chan error, 1)
	upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := coderws.Accept(w, r, nil)
		if err != nil {
			upstreamDone <- err
			return
		}
		defer func() {
			_ = conn.CloseNow()
		}()

		readCtx, cancelRead := context.WithTimeout(r.Context(), 3*time.Second)
		_, _, readErr := conn.Read(readCtx)
		cancelRead()
		if readErr != nil {
			upstreamDone <- readErr
			return
		}
		_ = conn.Close(coderws.StatusInternalError, "forced upstream failure")
		upstreamDone <- nil
	}))
	t.Cleanup(upstreamServer.Close)

	cache := newOpenAIWSPassthroughConcurrencyCache()
	harness := newOpenAIWSPassthroughConcurrencyHarness(t, upstreamServer.URL, cache)
	writeOpenAIWSPassthroughTestMessage(
		t,
		harness.clientConn,
		`{"type":"response.create","model":"gpt-5.5","input":"upstream disconnect"}`,
	)

	closeReadCtx, cancelCloseRead := context.WithTimeout(context.Background(), 4*time.Second)
	_, _, clientCloseErr := harness.clientConn.Read(closeReadCtx)
	cancelCloseRead()
	require.Error(t, clientCloseErr)
	require.NotErrorIs(t, clientCloseErr, context.DeadlineExceeded)
	require.Contains(
		t,
		[]coderws.StatusCode{coderws.StatusInternalError, coderws.StatusCode(-1)},
		coderws.CloseStatus(clientCloseErr),
	)
	select {
	case <-harness.handlerDone:
	case <-time.After(3 * time.Second):
		t.Fatal("上游非终态断开后 handler 未结束")
	}

	requireOpenAIWSPassthroughSlotBalance(t, cache, 1)
	requireNoOpenAIWSPassthroughUsageLog(t, harness.usageRepo)
	schedulerMetrics := harness.gatewayService.SnapshotOpenAIAccountSchedulerMetrics()
	require.Zero(t, schedulerMetrics.ReportSuccessTotal)
	require.Equal(t, int64(1), schedulerMetrics.ReportFailureTotal)

	select {
	case upstreamErr := <-upstreamDone:
		require.NoError(t, upstreamErr)
	case <-time.After(3 * time.Second):
		t.Fatal("等待上游失败测试结束超时")
	}
}

func TestOpenAIResponsesWebSocket_PassthroughClientDisconnectDrainCapturesTerminal(t *testing.T) {
	upstreamReceived := make(chan struct{})
	upstreamDone := make(chan error, 1)
	upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := coderws.Accept(w, r, nil)
		if err != nil {
			upstreamDone <- err
			return
		}
		defer func() {
			_ = conn.CloseNow()
		}()

		readCtx, cancelRead := context.WithTimeout(r.Context(), 3*time.Second)
		_, _, readErr := conn.Read(readCtx)
		cancelRead()
		if readErr != nil {
			upstreamDone <- readErr
			return
		}
		close(upstreamReceived)

		timer := time.NewTimer(100 * time.Millisecond)
		defer timer.Stop()
		select {
		case <-r.Context().Done():
			upstreamDone <- r.Context().Err()
			return
		case <-timer.C:
		}
		writeCtx, cancelWrite := context.WithTimeout(r.Context(), 3*time.Second)
		writeErr := conn.Write(
			writeCtx,
			coderws.MessageText,
			[]byte(`{"type":"response.completed","response":{"id":"resp_client_drain_terminal","model":"gpt-5.5","usage":{"input_tokens":6,"output_tokens":4}}}`),
		)
		cancelWrite()
		upstreamDone <- writeErr
	}))
	t.Cleanup(upstreamServer.Close)

	cache := newOpenAIWSPassthroughConcurrencyCache()
	harness := newOpenAIWSPassthroughConcurrencyHarness(t, upstreamServer.URL, cache)
	writeOpenAIWSPassthroughTestMessage(
		t,
		harness.clientConn,
		`{"type":"response.create","model":"gpt-5.5","input":"drain terminal"}`,
	)
	select {
	case <-upstreamReceived:
	case <-time.After(3 * time.Second):
		t.Fatal("测试上游未收到 drain turn")
	}

	clientCloseDone := make(chan error, 1)
	go func() {
		clientCloseDone <- harness.clientConn.Close(coderws.StatusNormalClosure, "done")
	}()
	select {
	case <-harness.handlerDone:
	case <-time.After(4 * time.Second):
		t.Fatal("drain 捕获 terminal 后 handler 未结束")
	}
	select {
	case <-clientCloseDone:
	case <-time.After(2 * time.Second):
		t.Fatal("drain terminal 客户端关闭握手未结束")
	}

	requireOpenAIWSPassthroughSlotBalance(t, cache, 1)
	usageLog := readOpenAIWSPassthroughUsageLog(t, harness.usageRepo)
	require.Equal(t, "resp_client_drain_terminal", usageLog.RequestID)
	require.Equal(t, 6, usageLog.InputTokens)
	require.Equal(t, 4, usageLog.OutputTokens)
	requireNoOpenAIWSPassthroughUsageLog(t, harness.usageRepo)
	schedulerMetrics := harness.gatewayService.SnapshotOpenAIAccountSchedulerMetrics()
	require.Equal(t, int64(1), schedulerMetrics.ReportSuccessTotal)
	require.Zero(t, schedulerMetrics.ReportFailureTotal)

	select {
	case upstreamErr := <-upstreamDone:
		require.NoError(t, upstreamErr)
	case <-time.After(3 * time.Second):
		t.Fatal("等待 drain terminal 测试上游结束超时")
	}
}
