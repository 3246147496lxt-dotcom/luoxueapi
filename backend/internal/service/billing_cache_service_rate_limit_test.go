//go:build unit

package service

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type versionedRateLimitCacheStub struct {
	billingCacheWorkerStub
	generation      atomic.Uint64
	setCalls        atomic.Int64
	acceptedWrites  atomic.Int64
	invalidateCalls atomic.Int64
	rejectNextSet   atomic.Bool
	onReject        func()
}

func (s *versionedRateLimitCacheStub) GetAPIKeyRateLimitWithGeneration(context.Context, int64) (*APIKeyRateLimitCacheData, uint64, error) {
	return nil, s.generation.Load(), ErrBillingCacheMiss
}

func (s *versionedRateLimitCacheStub) SetAPIKeyRateLimitIfGeneration(_ context.Context, _ int64, _ *APIKeyRateLimitCacheData, generation uint64) error {
	s.setCalls.Add(1)
	if s.rejectNextSet.CompareAndSwap(true, false) {
		s.generation.Add(1)
		if s.onReject != nil {
			s.onReject()
		}
		return ErrBillingCacheGenerationChanged
	}
	if s.generation.Load() != generation {
		return ErrBillingCacheGenerationChanged
	}
	s.acceptedWrites.Add(1)
	return nil
}

func (s *versionedRateLimitCacheStub) InvalidateAPIKeyRateLimit(context.Context, int64) error {
	s.generation.Add(1)
	s.invalidateCalls.Add(1)
	return nil
}

type rateLimitLoaderFunc func(context.Context, int64) (*APIKeyRateLimitData, error)

func (f rateLimitLoaderFunc) GetRateLimitData(ctx context.Context, keyID int64) (*APIKeyRateLimitData, error) {
	return f(ctx, keyID)
}

type rateLimitQuotaUpdaterStub struct {
	rateLimitCalls atomic.Int64
}

func (s *rateLimitQuotaUpdaterStub) UpdateQuotaUsed(context.Context, int64, float64) error {
	return nil
}

func (s *rateLimitQuotaUpdaterStub) UpdateRateLimitUsage(context.Context, int64, float64) error {
	s.rateLimitCalls.Add(1)
	return nil
}

type rateLimitInvalidationCacheStub struct {
	billingCacheWorkerStub
	invalidateCalls atomic.Int64
	updateCalls     atomic.Int64
}

type failedRateLimitInvalidationCacheStub struct {
	billingCacheWorkerStub
	stale           *APIKeyRateLimitCacheData
	getCalls        atomic.Int64
	setCalls        atomic.Int64
	invalidateCalls atomic.Int64
}

func (s *failedRateLimitInvalidationCacheStub) GetAPIKeyRateLimit(context.Context, int64) (*APIKeyRateLimitCacheData, error) {
	s.getCalls.Add(1)
	return s.stale, nil
}

func (s *failedRateLimitInvalidationCacheStub) SetAPIKeyRateLimit(_ context.Context, _ int64, data *APIKeyRateLimitCacheData) error {
	s.setCalls.Add(1)
	s.stale = data
	return nil
}

func (s *failedRateLimitInvalidationCacheStub) InvalidateAPIKeyRateLimit(context.Context, int64) error {
	s.invalidateCalls.Add(1)
	return errors.New("redis unavailable")
}

func (s *rateLimitInvalidationCacheStub) InvalidateAPIKeyRateLimit(context.Context, int64) error {
	s.invalidateCalls.Add(1)
	return nil
}

func (s *rateLimitInvalidationCacheStub) UpdateAPIKeyRateLimitUsage(context.Context, int64, float64) error {
	s.updateCalls.Add(1)
	return nil
}

func TestFinalizePostUsageBillingInvalidatesAPIKeyRateLimitSynchronously(t *testing.T) {
	cache := &rateLimitInvalidationCacheStub{}
	cacheService := &BillingCacheService{cache: cache}
	apiKey := &APIKey{ID: 7101, RateLimit5h: 10}

	finalizePostUsageBilling(
		context.Background(),
		&postUsageBillingParams{
			Cost:   &CostBreakdown{ActualCost: 1.25},
			APIKey: apiKey,
		},
		&billingDeps{billingCacheService: cacheService},
		&UsageBillingApplyResult{
			EffectsKnown:               true,
			APIKeyRateLimitCharged:     true,
			APIKeyRateLimitChargedCost: 1.25,
		},
	)

	// The post-commit path must wait for eviction before returning. A relative
	// async update would be both racy and invisible to this assertion.
	require.EqualValues(t, 1, cache.invalidateCalls.Load())
	require.Zero(t, cache.updateCalls.Load())
}

func TestQueueUpdateAPIKeyRateLimitUsageIsSafeCompatibilityShim(t *testing.T) {
	cache := &rateLimitInvalidationCacheStub{}
	cacheService := &BillingCacheService{cache: cache}

	cacheService.QueueUpdateAPIKeyRateLimitUsage(7102, 2.5)

	// Keep the old method available to alternate callers, but make it an
	// eviction rather than a relative increment so it cannot double-count a
	// committed DB charge.
	require.EqualValues(t, 1, cache.invalidateCalls.Load())
	require.Zero(t, cache.updateCalls.Load())
}

func TestCheckAPIKeyRateLimitsDoesNotRefillAfterLocalInvalidation(t *testing.T) {
	cache := &versionedRateLimitCacheStub{}
	svc := &BillingCacheService{cache: cache}
	var invalidated atomic.Bool
	var loaderCalls atomic.Int64
	now := time.Now()
	svc.apiKeyRateLimitLoader = rateLimitLoaderFunc(func(context.Context, int64) (*APIKeyRateLimitData, error) {
		call := loaderCalls.Add(1)
		// Simulate a committed billing transaction arriving while the first DB
		// miss is being loaded. The stale snapshot must be rejected, then a
		// second load at the new generation may safely refill the cache.
		if invalidated.CompareAndSwap(false, true) {
			if err := svc.InvalidateAPIKeyRateLimit(context.Background(), 7103); err != nil {
				return nil, err
			}
		}
		return &APIKeyRateLimitData{
			Usage5h:       float64(call),
			Usage1d:       float64(call),
			Usage7d:       float64(call),
			Window5hStart: &now,
			Window1dStart: &now,
			Window7dStart: &now,
		}, nil
	})

	err := svc.checkAPIKeyRateLimits(context.Background(), &APIKey{ID: 7103, RateLimit5h: 10})
	require.NoError(t, err)
	require.EqualValues(t, 1, cache.invalidateCalls.Load())
	require.EqualValues(t, 1, cache.setCalls.Load(), "only the post-invalidation snapshot should be written")
	require.EqualValues(t, 1, cache.acceptedWrites.Load())
	require.EqualValues(t, 2, loaderCalls.Load())
}

func TestCheckAPIKeyRateLimitsRetriesConditionalGenerationRejection(t *testing.T) {
	cache := &versionedRateLimitCacheStub{}
	cache.rejectNextSet.Store(true)
	var loaderCalls atomic.Int64
	now := time.Now()
	svc := &BillingCacheService{cache: cache}
	svc.apiKeyRateLimitLoader = rateLimitLoaderFunc(func(context.Context, int64) (*APIKeyRateLimitData, error) {
		call := loaderCalls.Add(1)
		return &APIKeyRateLimitData{
			Usage5h:       float64(call),
			Usage1d:       float64(call),
			Usage7d:       float64(call),
			Window5hStart: &now,
			Window1dStart: &now,
			Window7dStart: &now,
		}, nil
	})

	err := svc.checkAPIKeyRateLimits(context.Background(), &APIKey{ID: 7106, RateLimit5h: 10})
	require.NoError(t, err)
	require.EqualValues(t, 2, loaderCalls.Load(), "a rejected conditional refill must be retried")
	require.EqualValues(t, 2, cache.setCalls.Load())
	require.EqualValues(t, 1, cache.acceptedWrites.Load())
}

func TestCheckAPIKeyRateLimitsBypassesStaleCacheAfterInvalidationFailure(t *testing.T) {
	now := time.Now()
	cache := &failedRateLimitInvalidationCacheStub{stale: &APIKeyRateLimitCacheData{
		Usage5h: 0, Window5h: now.Unix(),
	}}
	var loaderCalls atomic.Int64
	svc := &BillingCacheService{
		cache: cache,
		apiKeyRateLimitLoader: rateLimitLoaderFunc(func(context.Context, int64) (*APIKeyRateLimitData, error) {
			loaderCalls.Add(1)
			return &APIKeyRateLimitData{
				Usage5h:       6,
				Window5hStart: &now,
				Window1dStart: &now,
				Window7dStart: &now,
			}, nil
		}),
	}

	require.Error(t, svc.InvalidateAPIKeyRateLimit(context.Background(), 7105))
	err := svc.checkAPIKeyRateLimits(context.Background(), &APIKey{ID: 7105, RateLimit5h: 5})
	require.ErrorIs(t, err, ErrAPIKeyRateLimit5hExceeded)
	require.Zero(t, cache.getCalls.Load(), "a dirty legacy cache must not be consulted")
	require.EqualValues(t, 1, cache.setCalls.Load(), "the authoritative DB snapshot should repair the cache")
	require.EqualValues(t, 1, loaderCalls.Load())
	require.Zero(t, svc.apiKeyRateLimitCacheDirtyGeneration(7105))
}

func TestLegacyPostUsageBillingInvalidatesRateLimitAfterDBUpdate(t *testing.T) {
	cache := &rateLimitInvalidationCacheStub{}
	cacheService := &BillingCacheService{cache: cache}
	updater := &rateLimitQuotaUpdaterStub{}

	err := postUsageBilling(context.Background(), &postUsageBillingParams{
		Cost:          &CostBreakdown{ActualCost: 1},
		User:          &User{ID: 1},
		APIKey:        &APIKey{ID: 7104, RateLimit5h: 10},
		APIKeyService: updater,
	}, &billingDeps{
		userRepo:            &mockUserRepo{},
		billingCacheService: cacheService,
	})

	require.NoError(t, err)
	require.EqualValues(t, 1, updater.rateLimitCalls.Load())
	require.EqualValues(t, 1, cache.invalidateCalls.Load())
	require.Zero(t, cache.updateCalls.Load())
}
