//go:build unit

package service

import (
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func profitStickyPathAccount(id int64, rate float64) Account {
	return Account{
		ID:             id,
		Name:           "profit-sticky",
		Platform:       PlatformOpenAI,
		Type:           AccountTypeAPIKey,
		Status:         StatusActive,
		Schedulable:    true,
		Concurrency:    2,
		RateMultiplier: &rate,
	}
}

func profitStickyPathGroup(id int64, enabled bool) *Group {
	return &Group{
		ID:                   id,
		Platform:             PlatformOpenAI,
		Status:               StatusActive,
		Hydrated:             true,
		RateMultiplier:       1,
		SubscriptionType:     SubscriptionTypeStandard,
		ProfitControlEnabled: enabled,
		ProfitMinMargin:      0.5,
	}
}

func newProfitStickyPathService(accounts []Account, cache *schedulerTestGatewayCache, concurrencyCache ConcurrencyCache, loadBatch bool) *OpenAIGatewayService {
	cfg := &config.Config{}
	cfg.Gateway.Scheduling.LoadBatchEnabled = loadBatch
	var concurrency *ConcurrencyService
	if concurrencyCache != nil {
		concurrency = NewConcurrencyService(concurrencyCache)
	}
	return &OpenAIGatewayService{
		accountRepo:        stubOpenAIAccountRepo{accounts: accounts},
		cache:              cache,
		cfg:                cfg,
		concurrencyService: concurrency,
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("false"),
	}
}

func TestOpenAIProfitControlDefersAllSchedulerStickyWrites(t *testing.T) {
	cheap := profitStickyPathAccount(45, 0.3)
	expensive := profitStickyPathAccount(46, 0.8)
	accounts := []Account{cheap, expensive}
	groupID := int64(9)
	const sessionHash = "profit-sticky"

	tests := []struct {
		name             string
		loadBatch        bool
		concurrencyCache ConcurrencyCache
	}{
		{name: "legacy engine", loadBatch: false},
		{name: "load-aware success", loadBatch: true, concurrencyCache: stubConcurrencyCache{}},
		{name: "load-aware batch error fallback", loadBatch: true, concurrencyCache: stubConcurrencyCache{loadBatchErr: errors.New("redis unavailable")}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cache := &schedulerTestGatewayCache{sessionBindings: map[string]int64{}}
			svc := newProfitStickyPathService(accounts, cache, tc.concurrencyCache, tc.loadBatch)
			group := profitStickyPathGroup(groupID, true)

			selection, _, err := svc.SelectAccountWithScheduler(
				profitControlTestContext(group), &groupID, "", sessionHash, "gpt-test", nil, OpenAIUpstreamTransportAny, false,
			)

			require.NoError(t, err)
			require.NotNil(t, selection)
			require.Equal(t, cheap.ID, selection.Account.ID)
			require.True(t, selection.ProfitGateActive())
			require.Empty(t, cache.sessionBindings, "a gated scheduler must wait for terminal admission before binding sticky")
			if selection.ReleaseFunc != nil {
				selection.ReleaseFunc()
			}
		})
	}
}

func TestOpenAIProfitControlDisabledKeepsLegacyEagerSticky(t *testing.T) {
	cheap := profitStickyPathAccount(55, 0.3)
	cache := &schedulerTestGatewayCache{sessionBindings: map[string]int64{}}
	svc := newProfitStickyPathService([]Account{cheap}, cache, nil, false)
	groupID := int64(10)
	group := profitStickyPathGroup(groupID, false)

	selection, _, err := svc.SelectAccountWithScheduler(
		profitControlTestContext(group), &groupID, "", "ungated-sticky", "gpt-test", nil, OpenAIUpstreamTransportAny, false,
	)

	require.NoError(t, err)
	require.NotNil(t, selection)
	require.False(t, selection.ProfitGateActive())
	require.Equal(t, cheap.ID, cache.sessionBindings["openai:ungated-sticky"])
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}
