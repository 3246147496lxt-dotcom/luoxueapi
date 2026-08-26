//go:build unit

package service

import (
	"context"
	"errors"
	"math"
	"sync/atomic"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type balanceEligibilityCacheStub struct {
	billingCacheWorkerStub

	balance                  float64
	cacheMissAfterInvalidate bool
	invalidated              atomic.Bool
	deductCalls              atomic.Int64
	invalidateCalls          atomic.Int64
	lastDeductAmount         atomic.Uint64
}

type failedBalanceInvalidationCacheStub struct {
	billingCacheWorkerStub
	balance         float64
	getCalls        atomic.Int64
	setCalls        atomic.Int64
	invalidateCalls atomic.Int64
}

func (s *failedBalanceInvalidationCacheStub) GetUserBalance(context.Context, int64) (float64, error) {
	s.getCalls.Add(1)
	return s.balance, nil
}

func (s *failedBalanceInvalidationCacheStub) SetUserBalance(_ context.Context, _ int64, balance float64) error {
	s.setCalls.Add(1)
	s.balance = balance
	return nil
}

func (s *failedBalanceInvalidationCacheStub) InvalidateUserBalance(context.Context, int64) error {
	s.invalidateCalls.Add(1)
	return errors.New("redis unavailable")
}

func (s *balanceEligibilityCacheStub) GetUserBalance(context.Context, int64) (float64, error) {
	if s.cacheMissAfterInvalidate && s.invalidated.Load() {
		return 0, errors.New("cache miss")
	}
	return s.balance, nil
}

func (s *balanceEligibilityCacheStub) DeductUserBalance(_ context.Context, _ int64, amount float64) error {
	s.lastDeductAmount.Store(math.Float64bits(amount))
	s.deductCalls.Add(1)
	return nil
}

func (s *balanceEligibilityCacheStub) InvalidateUserBalance(context.Context, int64) error {
	s.invalidateCalls.Add(1)
	s.invalidated.Store(true)
	return nil
}

func TestCheckBillingEligibility_RejectsBalanceBelowMinimumReserve(t *testing.T) {
	cache := &balanceEligibilityCacheStub{balance: 0.005}
	cfg := &config.Config{}
	cfg.Billing.MinimumBalanceReserve = 0.01
	svc := newStartedBillingCacheServiceForTest(cache, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(svc.Stop)

	err := svc.CheckBillingEligibility(context.Background(), &User{ID: 1}, nil, nil, nil, "")
	require.ErrorIs(t, err, ErrInsufficientBalance)
}

func TestCheckBillingEligibility_AllowsBalanceAtMinimumReserve(t *testing.T) {
	cache := &balanceEligibilityCacheStub{balance: 0.01}
	cfg := &config.Config{}
	cfg.Billing.MinimumBalanceReserve = 0.01
	svc := newStartedBillingCacheServiceForTest(cache, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(svc.Stop)

	err := svc.CheckBillingEligibility(context.Background(), &User{ID: 1}, nil, nil, nil, "")
	require.NoError(t, err)
}

func TestGetUserBalanceBypassesStaleCacheAfterInvalidationFailure(t *testing.T) {
	cache := &failedBalanceInvalidationCacheStub{balance: 10}
	userRepo := &balanceLoadUserRepoStub{balance: 2}
	svc := &BillingCacheService{cache: cache, userRepo: userRepo}

	require.Error(t, svc.InvalidateUserBalance(context.Background(), 42))
	balance, err := svc.GetUserBalance(context.Background(), 42)
	require.NoError(t, err)
	require.Equal(t, 2.0, balance)
	require.Zero(t, cache.getCalls.Load(), "a dirty legacy cache must not be consulted")
	require.EqualValues(t, 1, cache.setCalls.Load(), "the authoritative DB value should repair the cache")
	require.EqualValues(t, 1, userRepo.calls.Load())
	require.Zero(t, svc.balanceCacheDirtyGeneration(42), "successful repair should clear only the observed dirty generation")

	// Once repaired, the next read can safely use the cache without another DB
	// lookup even though the original DEL failed.
	balance, err = svc.GetUserBalance(context.Background(), 42)
	require.NoError(t, err)
	require.Equal(t, 2.0, balance)
	require.EqualValues(t, 1, userRepo.calls.Load())
}

func TestSyncBalanceCacheAfterDeduction_InvalidatesExhaustedBalance(t *testing.T) {
	cache := &balanceEligibilityCacheStub{
		balance:                  0.50,
		cacheMissAfterInvalidate: true,
	}
	userRepo := &balanceLoadUserRepoStub{balance: -0.25}
	cfg := &config.Config{}
	cfg.Billing.MinimumBalanceReserve = 0.01
	svc := newStartedBillingCacheServiceForTest(cache, userRepo, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(svc.Stop)

	newBalance := -0.25
	syncBalanceCacheAfterDeduction(context.Background(), &postUsageBillingParams{
		Cost: &CostBreakdown{ActualCost: 0.75},
		User: &User{ID: 1},
	}, &billingDeps{billingCacheService: svc}, &UsageBillingApplyResult{
		NewBalance:         &newBalance,
		BalanceOverdrafted: true,
	})

	require.Equal(t, int64(1), cache.invalidateCalls.Load())
	require.Equal(t, int64(0), cache.deductCalls.Load())

	err := svc.CheckBillingEligibility(context.Background(), &User{ID: 1}, nil, nil, nil, "")
	require.ErrorIs(t, err, ErrInsufficientBalance)
	require.Equal(t, int64(1), userRepo.calls.Load())
}

func TestSyncBalanceCacheAfterDeduction_InvalidatesWhenBalanceFallsBelowReserve(t *testing.T) {
	cache := &balanceEligibilityCacheStub{balance: 0.50}
	cfg := &config.Config{}
	cfg.Billing.MinimumBalanceReserve = 0.01
	svc := newStartedBillingCacheServiceForTest(cache, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(svc.Stop)

	newBalance := 0.005
	syncBalanceCacheAfterDeduction(context.Background(), &postUsageBillingParams{
		Cost: &CostBreakdown{ActualCost: 0.495},
		User: &User{ID: 1},
	}, &billingDeps{billingCacheService: svc}, &UsageBillingApplyResult{NewBalance: &newBalance})

	require.Equal(t, int64(1), cache.invalidateCalls.Load())
	require.Equal(t, int64(0), cache.deductCalls.Load())
}

func TestSyncBalanceCacheAfterDeduction_InvalidatesCommittedEligibleBalance(t *testing.T) {
	cache := &balanceEligibilityCacheStub{balance: 1}
	cfg := &config.Config{}
	cfg.Billing.MinimumBalanceReserve = 0.01
	svc := newStartedBillingCacheServiceForTest(cache, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(svc.Stop)

	newBalance := 0.75
	syncBalanceCacheAfterDeduction(context.Background(), &postUsageBillingParams{
		Cost: &CostBreakdown{ActualCost: 0.25},
		User: &User{ID: 1},
	}, &billingDeps{billingCacheService: svc}, &UsageBillingApplyResult{NewBalance: &newBalance})

	require.Equal(t, int64(1), cache.invalidateCalls.Load())
	require.Equal(t, int64(0), cache.deductCalls.Load())
}

func TestSyncBalanceCacheAfterDeduction_UsesCommittedQuantizedCost(t *testing.T) {
	cache := &balanceEligibilityCacheStub{balance: 1}
	cfg := &config.Config{}
	cfg.Billing.MinimumBalanceReserve = 0.01
	svc := newStartedBillingCacheServiceForTest(cache, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(svc.Stop)

	const rawCost = 0.000078125
	const committedCost = 0.00007813
	newBalance := 1.0
	syncBalanceCacheAfterDeduction(context.Background(), &postUsageBillingParams{
		Cost: &CostBreakdown{ActualCost: rawCost},
		User: &User{ID: 1},
	}, &billingDeps{billingCacheService: svc}, &UsageBillingApplyResult{
		NewBalance:         &newBalance,
		BalanceChargedCost: committedCost,
	})

	require.Equal(t, int64(1), cache.invalidateCalls.Load())
	require.Equal(t, int64(0), cache.deductCalls.Load())
}
