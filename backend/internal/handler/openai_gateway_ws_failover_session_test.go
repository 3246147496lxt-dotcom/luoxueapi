package handler

import (
	"context"
	"errors"
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

func readOpenAIWSFailoverSessionUpstreamHit(
	t *testing.T,
	hits <-chan []byte,
	label string,
) []byte {
	t.Helper()
	select {
	case payload := <-hits:
		return payload
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for %s upstream request", label)
		return nil
	}
}

func requireNoExtraOpenAIWSFailoverSessionUpstreamHit(
	t *testing.T,
	hits <-chan []byte,
	label string,
) {
	t.Helper()
	select {
	case payload := <-hits:
		t.Fatalf("%s upstream received an extra request: %s", label, payload)
	case <-time.After(100 * time.Millisecond):
	}
}

func collectOpenAIWSFailoverSessionRemainingUpstreamFrames(
	conn *coderws.Conn,
	hits chan<- []byte,
	stop <-chan struct{},
) error {
	readCtx, cancelRead := context.WithCancel(context.Background())
	readDone := make(chan struct{})
	go func() {
		select {
		case <-stop:
			cancelRead()
		case <-readDone:
		}
	}()
	defer func() {
		close(readDone)
		cancelRead()
	}()

	for {
		msgType, payload, err := conn.Read(readCtx)
		if err != nil {
			return nil
		}
		if msgType != coderws.MessageText {
			return errors.New("upstream received an extra non-text websocket frame")
		}
		select {
		case hits <- append([]byte(nil), payload...):
		default:
			return errors.New("upstream extra-frame buffer is full")
		}
	}
}

type openAIWSFailoverSessionSlotEvent struct {
	slotType  string
	action    string
	ownerID   int64
	requestID string
}

type openAIWSFailoverSessionConcurrencyCache struct {
	*openAIWSPassthroughConcurrencyCache

	mu       sync.Mutex
	timeline []openAIWSFailoverSessionSlotEvent
}

func newOpenAIWSFailoverSessionConcurrencyCache() *openAIWSFailoverSessionConcurrencyCache {
	return &openAIWSFailoverSessionConcurrencyCache{
		openAIWSPassthroughConcurrencyCache: newOpenAIWSPassthroughConcurrencyCache(),
	}
}

func (c *openAIWSFailoverSessionConcurrencyCache) recordSlotEvent(
	slotType string,
	action string,
	ownerID int64,
	requestID string,
) {
	c.mu.Lock()
	c.timeline = append(c.timeline, openAIWSFailoverSessionSlotEvent{
		slotType:  slotType,
		action:    action,
		ownerID:   ownerID,
		requestID: requestID,
	})
	c.mu.Unlock()
}

func (c *openAIWSFailoverSessionConcurrencyCache) AcquireUserSlot(
	ctx context.Context,
	userID int64,
	maxConcurrency int,
	requestID string,
) (bool, error) {
	c.recordSlotEvent("user", "acquire", userID, requestID)
	return c.openAIWSPassthroughConcurrencyCache.AcquireUserSlot(ctx, userID, maxConcurrency, requestID)
}

func (c *openAIWSFailoverSessionConcurrencyCache) ReleaseUserSlot(
	ctx context.Context,
	userID int64,
	requestID string,
) error {
	c.recordSlotEvent("user", "release", userID, requestID)
	return c.openAIWSPassthroughConcurrencyCache.ReleaseUserSlot(ctx, userID, requestID)
}

func (c *openAIWSFailoverSessionConcurrencyCache) AcquireAccountSlot(
	ctx context.Context,
	accountID int64,
	maxConcurrency int,
	requestID string,
) (bool, error) {
	c.recordSlotEvent("account", "acquire", accountID, requestID)
	return c.openAIWSPassthroughConcurrencyCache.AcquireAccountSlot(ctx, accountID, maxConcurrency, requestID)
}

func (c *openAIWSFailoverSessionConcurrencyCache) ReleaseAccountSlot(
	ctx context.Context,
	accountID int64,
	requestID string,
) error {
	c.recordSlotEvent("account", "release", accountID, requestID)
	return c.openAIWSPassthroughConcurrencyCache.ReleaseAccountSlot(ctx, accountID, requestID)
}

func (c *openAIWSFailoverSessionConcurrencyCache) slotTimeline() []openAIWSFailoverSessionSlotEvent {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]openAIWSFailoverSessionSlotEvent(nil), c.timeline...)
}

func TestOpenAIResponsesWebSocket_FailoverSessionSurvivesPassthroughToCtxPool(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stopUpstreamReads := make(chan struct{})
	var stopUpstreamReadsOnce sync.Once
	stopUpstreamProbes := func() {
		stopUpstreamReadsOnce.Do(func() {
			close(stopUpstreamReads)
		})
	}
	t.Cleanup(stopUpstreamProbes)

	firstHits := make(chan []byte, 8)
	firstDone := make(chan error, 1)
	firstUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := coderws.Accept(w, r, nil)
		if err != nil {
			firstDone <- err
			return
		}
		defer func() {
			_ = conn.CloseNow()
		}()

		readCtx, cancelRead := context.WithTimeout(r.Context(), 3*time.Second)
		msgType, payload, readErr := conn.Read(readCtx)
		cancelRead()
		if readErr != nil {
			firstDone <- readErr
			return
		}
		if msgType != coderws.MessageText {
			firstDone <- errors.New("first upstream received a non-text websocket frame")
			return
		}
		firstHits <- append([]byte(nil), payload...)

		writeCtx, cancelWrite := context.WithTimeout(r.Context(), 3*time.Second)
		writeErr := conn.Write(
			writeCtx,
			coderws.MessageText,
			[]byte(`{"type":"error","error":{"code":"rate_limit_exceeded","type":"usage_limit_reached","message":"The usage limit has been reached"}}`),
		)
		cancelWrite()
		if writeErr != nil {
			firstDone <- writeErr
			return
		}
		firstDone <- collectOpenAIWSFailoverSessionRemainingUpstreamFrames(conn, firstHits, stopUpstreamReads)
	}))
	t.Cleanup(firstUpstream.Close)

	secondHits := make(chan []byte, 8)
	secondDone := make(chan error, 1)
	secondUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := coderws.Accept(w, r, nil)
		if err != nil {
			secondDone <- err
			return
		}
		defer func() {
			_ = conn.CloseNow()
		}()

		readCtx, cancelRead := context.WithTimeout(r.Context(), 3*time.Second)
		msgType, payload, readErr := conn.Read(readCtx)
		cancelRead()
		if readErr != nil {
			secondDone <- readErr
			return
		}
		if msgType != coderws.MessageText {
			secondDone <- errors.New("second upstream received a non-text websocket frame")
			return
		}
		secondHits <- append([]byte(nil), payload...)

		writeCtx, cancelWrite := context.WithTimeout(r.Context(), 3*time.Second)
		writeErr := conn.Write(
			writeCtx,
			coderws.MessageText,
			[]byte(`{"type":"response.completed","response":{"id":"resp_failover_session_ok","model":"gpt-5.5","usage":{"input_tokens":3,"output_tokens":2}}}`),
		)
		cancelWrite()
		if writeErr != nil {
			secondDone <- writeErr
			return
		}
		secondDone <- collectOpenAIWSFailoverSessionRemainingUpstreamFrames(conn, secondHits, stopUpstreamReads)
	}))
	t.Cleanup(secondUpstream.Close)

	groupID := int64(4961)
	accounts := []service.Account{
		{
			ID:          4962,
			Name:        "passthrough-rate-limited",
			Platform:    service.PlatformOpenAI,
			Type:        service.AccountTypeAPIKey,
			Status:      service.StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    1,
			Credentials: map[string]any{
				"api_key":  "sk-first",
				"base_url": firstUpstream.URL,
			},
			Extra: map[string]any{
				"openai_apikey_responses_websockets_v2_enabled": true,
				"openai_apikey_responses_websockets_v2_mode":    service.OpenAIWSIngressModePassthrough,
			},
		},
		{
			ID:          4963,
			Name:        "ctx-pool-healthy",
			Platform:    service.PlatformOpenAI,
			Type:        service.AccountTypeAPIKey,
			Status:      service.StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    2,
			Credentials: map[string]any{
				"api_key":  "sk-second",
				"base_url": secondUpstream.URL,
			},
			Extra: map[string]any{
				"openai_apikey_responses_websockets_v2_enabled": true,
				"openai_apikey_responses_websockets_v2_mode":    service.OpenAIWSIngressModeCtxPool,
			},
		},
	}

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
	cfg.Gateway.MaxAccountSwitches = 2

	accountRepo := &openAIWSFailoverHandlerAccountRepoStub{accounts: accounts}
	usageRepo := &openAIWSUsageHandlerUsageLogRepoStub{
		created: make(chan *service.UsageLog, 4),
	}
	rateLimitService := service.NewRateLimitService(accountRepo, usageRepo, cfg, nil, nil)
	billingCacheService := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(billingCacheService.Stop)
	concurrencyCache := newOpenAIWSFailoverSessionConcurrencyCache()
	concurrencyService := service.NewConcurrencyService(concurrencyCache)
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
		nil,
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
		ID:      4964,
		GroupID: &groupID,
		User:    &service.User{ID: 4965, Status: service.StatusActive},
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
		nil,
	)
	cancelDial()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = clientConn.CloseNow()
	})

	firstClientMessage := []byte(
		`{"type":"response.create","model":"gpt-5.5","input":"survive failover","stream":false}`,
	)
	writeCtx, cancelWrite := context.WithTimeout(context.Background(), 3*time.Second)
	err = clientConn.Write(writeCtx, coderws.MessageText, firstClientMessage)
	cancelWrite()
	require.NoError(t, err)

	readCtx, cancelRead := context.WithTimeout(context.Background(), 5*time.Second)
	msgType, terminal, readErr := clientConn.Read(readCtx)
	cancelRead()
	require.NoError(t, readErr, "failover must preserve the original client websocket")
	require.Equal(t, coderws.MessageText, msgType)
	require.Equal(t, "response.completed", gjson.GetBytes(terminal, "type").String())
	require.Equal(t, "resp_failover_session_ok", gjson.GetBytes(terminal, "response.id").String())

	firstForwarded := readOpenAIWSFailoverSessionUpstreamHit(t, firstHits, "first")
	secondForwarded := readOpenAIWSFailoverSessionUpstreamHit(t, secondHits, "second")
	require.Equal(t, firstClientMessage, firstForwarded)
	require.JSONEq(t, string(firstClientMessage), string(secondForwarded))

	_ = clientConn.Close(coderws.StatusNormalClosure, "done")
	select {
	case <-handlerDone:
	case <-time.After(3 * time.Second):
		t.Fatal("handler did not finish after the failover session client closed")
	}

	requireOpenAIWSPassthroughSlotBalance(
		t,
		concurrencyCache.openAIWSPassthroughConcurrencyCache,
		2,
	)
	slots := concurrencyCache.openAIWSPassthroughConcurrencyCache.snapshot()
	require.Equal(t, []openAIWSFailoverSessionSlotEvent{
		{slotType: "user", action: "acquire", ownerID: 4965, requestID: slots.userAcquires[0]},
		{slotType: "account", action: "acquire", ownerID: 4962, requestID: slots.accountAcquires[0]},
		{slotType: "account", action: "release", ownerID: 4962, requestID: slots.accountReleases[0]},
		{slotType: "user", action: "release", ownerID: 4965, requestID: slots.userReleases[0]},
		{slotType: "user", action: "acquire", ownerID: 4965, requestID: slots.userAcquires[1]},
		{slotType: "account", action: "acquire", ownerID: 4963, requestID: slots.accountAcquires[1]},
		{slotType: "account", action: "release", ownerID: 4963, requestID: slots.accountReleases[1]},
		{slotType: "user", action: "release", ownerID: 4965, requestID: slots.userReleases[1]},
	}, concurrencyCache.slotTimeline())

	usageLog := readOpenAIWSPassthroughUsageLog(t, usageRepo)
	require.Equal(t, int64(4963), usageLog.AccountID)
	require.Equal(t, "resp_failover_session_ok", usageLog.RequestID)
	requireNoOpenAIWSPassthroughUsageLog(t, usageRepo)
	require.Equal(t, []int64{4962}, accountRepo.rateLimitedIDs)

	stopUpstreamProbes()
	select {
	case upstreamErr := <-firstDone:
		require.NoError(t, upstreamErr)
	case <-time.After(3 * time.Second):
		t.Fatal("first upstream did not finish")
	}
	select {
	case upstreamErr := <-secondDone:
		require.NoError(t, upstreamErr)
	case <-time.After(3 * time.Second):
		t.Fatal("second upstream did not finish")
	}
	requireNoExtraOpenAIWSFailoverSessionUpstreamHit(t, firstHits, "first")
	requireNoExtraOpenAIWSFailoverSessionUpstreamHit(t, secondHits, "second")
}
