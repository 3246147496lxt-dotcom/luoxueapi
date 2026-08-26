//go:build unit

package handler

import (
	"context"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type profitCountingConcurrencyCache struct {
	fakeConcurrencyCache
	accountReleases atomic.Int64
}

func (c *profitCountingConcurrencyCache) ReleaseAccountSlot(context.Context, int64, string) error {
	c.accountReleases.Add(1)
	return nil
}

type profitStickyCountingCache struct {
	accountID int64
	setCalls  atomic.Int64
}

func (c *profitStickyCountingCache) GetSessionAccountID(context.Context, int64, string) (int64, error) {
	return c.accountID, nil
}

func (c *profitStickyCountingCache) SetSessionAccountID(_ context.Context, _ int64, _ string, accountID int64, _ time.Duration) error {
	c.accountID = accountID
	c.setCalls.Add(1)
	return nil
}

func (*profitStickyCountingCache) RefreshSessionTTL(context.Context, int64, string, time.Duration) error {
	return nil
}

func (*profitStickyCountingCache) DeleteSessionAccountID(context.Context, int64, string) error {
	return nil
}

func profitSlotTestAccount(id int64, rate float64) *service.Account {
	return &service.Account{
		ID:             id,
		Platform:       service.PlatformOpenAI,
		Type:           service.AccountTypeAPIKey,
		Status:         service.StatusActive,
		Schedulable:    true,
		Concurrency:    2,
		RateMultiplier: &rate,
	}
}

func profitSlotTestContext(t *testing.T, gateway *service.OpenAIGatewayService, groupID int64, suppress bool) context.Context {
	t.Helper()
	group := &service.Group{
		ID:                   groupID,
		Platform:             service.PlatformOpenAI,
		Status:               service.StatusActive,
		Hydrated:             true,
		RateMultiplier:       1,
		SubscriptionType:     service.SubscriptionTypeStandard,
		ProfitControlEnabled: true,
		ProfitMinMargin:      0.5,
	}
	ctx := context.WithValue(context.Background(), ctxkey.Group, group)
	if suppress {
		ctx = service.WithOpenAIProfitControlSuppressed(ctx)
	}
	ctx, pricingAt := gateway.WithOpenAIRequestPricingContext(ctx, &groupID)
	require.False(t, pricingAt.IsZero())
	return ctx
}

func TestAcquireResponsesAccountSlotProfitRecheck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gateway := &service.OpenAIGatewayService{}
	groupID := int64(50)

	newHandler := func(cache *profitCountingConcurrencyCache) *OpenAIGatewayHandler {
		return &OpenAIGatewayHandler{
			gatewayService:    gateway,
			concurrencyHelper: NewConcurrencyHelper(service.NewConcurrencyService(cache), SSEPingFormatClaude, 0),
		}
	}
	newSelection := func(account *service.Account) *service.AccountSelectionResult {
		return &service.AccountSelectionResult{
			Account:  account,
			Acquired: false,
			WaitPlan: &service.AccountWaitPlan{AccountID: account.ID, MaxConcurrency: 2, Timeout: time.Second, MaxWaiting: 2},
		}
	}

	t.Run("veto releases slot and requests reschedule without writing response", func(t *testing.T) {
		cache := &profitCountingConcurrencyCache{}
		h := newHandler(cache)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/v1/responses", nil).WithContext(profitSlotTestContext(t, gateway, groupID, false))
		streamStarted := false

		release, result := h.acquireResponsesAccountSlot(c, &groupID, "", newSelection(profitSlotTestAccount(1, 0.8)), false, &streamStarted, zap.NewNop())

		require.Equal(t, openAISlotAcquireProfitVetoed, result)
		require.Nil(t, release)
		require.Zero(t, w.Body.Len())
		require.Equal(t, int64(1), cache.accountReleases.Load())
	})

	t.Run("qualifying account acquires normally", func(t *testing.T) {
		cache := &profitCountingConcurrencyCache{}
		h := newHandler(cache)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/v1/responses", nil).WithContext(profitSlotTestContext(t, gateway, groupID, false))
		streamStarted := false

		release, result := h.acquireResponsesAccountSlot(c, &groupID, "", newSelection(profitSlotTestAccount(2, 0.3)), false, &streamStarted, zap.NewNop())

		require.Equal(t, openAISlotAcquireOK, result)
		require.NotNil(t, release)
		release()
	})

	t.Run("explicit suppression keeps non-token endpoint behavior", func(t *testing.T) {
		cache := &profitCountingConcurrencyCache{}
		h := newHandler(cache)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/v1/responses", nil).WithContext(profitSlotTestContext(t, gateway, groupID, true))
		streamStarted := false

		release, result := h.acquireResponsesAccountSlot(c, &groupID, "", newSelection(profitSlotTestAccount(3, 0.8)), false, &streamStarted, zap.NewNop())

		require.Equal(t, openAISlotAcquireOK, result)
		require.NotNil(t, release)
		release()
	})
}

func TestAcquiredUngatedSelectionDoesNotOverrideSchedulerStickyPolicy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cache := &profitStickyCountingCache{accountID: 11}
	gateway := service.NewOpenAIGatewayService(
		nil, nil, nil, nil, nil, nil, cache, nil, nil, nil, nil,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
	)
	h := &OpenAIGatewayHandler{gatewayService: gateway}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/responses", nil)
	streamStarted := false
	selection := &service.AccountSelectionResult{
		Account:     profitSlotTestAccount(12, 0.3),
		Acquired:    true,
		ReleaseFunc: func() {},
	}
	groupID := int64(50)

	release, result := h.acquireResponsesAccountSlot(c, &groupID, "preserve", selection, false, &streamStarted, zap.NewNop())

	require.Equal(t, openAISlotAcquireOK, result)
	require.NotNil(t, release)
	release()
	require.Zero(t, cache.setCalls.Load(), "the handler must not replay an ungated acquired selection's sticky write")
	require.Equal(t, int64(11), cache.accountID)
}
