//go:build unit

package service

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// generationBalanceHitCacheStub returns a value captured before an
// invalidation on the first read, then exposes the post-invalidation value on
// the retry. This models a Redis hit racing a committed balance update.
type generationBalanceHitCacheStub struct {
	billingCacheWorkerStub
	mu                      sync.Mutex
	balance                 float64
	freshBalance            float64
	generation              uint64
	service                 *BillingCacheService
	invalidateOnRead        atomic.Bool
	returnMiss              atomic.Bool
	returnGenerationChanged atomic.Bool
	rejectNextSet           atomic.Bool
	readCalls               atomic.Int64
	invalidateCalls         atomic.Int64
	conditionalSetCall      atomic.Int64
	tokenReadCalls          atomic.Int64
	remoteInvalidateOnToken atomic.Bool
	onReject                func()
}

func (s *generationBalanceHitCacheStub) GetUserBalanceWithGeneration(ctx context.Context, userID int64) (float64, uint64, error) {
	s.readCalls.Add(1)
	s.mu.Lock()
	balance, generation := s.balance, s.generation
	shouldInvalidate := s.invalidateOnRead.CompareAndSwap(true, false)
	s.mu.Unlock()
	if s.returnGenerationChanged.CompareAndSwap(true, false) {
		return 0, generation, ErrBillingCacheGenerationChanged
	}
	if s.returnMiss.Load() {
		return 0, generation, ErrBillingCacheMiss
	}
	if shouldInvalidate && s.service != nil {
		if err := s.service.InvalidateUserBalance(ctx, userID); err != nil {
			return 0, generation, err
		}
	}
	return balance, generation, nil
}

func (s *generationBalanceHitCacheStub) SetUserBalanceIfGeneration(_ context.Context, _ int64, balance float64, generation uint64) error {
	s.conditionalSetCall.Add(1)
	if s.rejectNextSet.CompareAndSwap(true, false) {
		s.mu.Lock()
		s.generation++
		s.mu.Unlock()
		if s.onReject != nil {
			s.onReject()
		}
		return ErrBillingCacheGenerationChanged
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if generation != s.generation {
		return ErrBillingCacheGenerationChanged
	}
	s.balance = balance
	return nil
}

func (s *generationBalanceHitCacheStub) GetUserBalanceGeneration(_ context.Context, _ int64) (uint64, error) {
	s.tokenReadCalls.Add(1)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.remoteInvalidateOnToken.CompareAndSwap(true, false) {
		s.generation++
		s.balance = s.freshBalance
	}
	return s.generation, nil
}

func (s *generationBalanceHitCacheStub) InvalidateUserBalance(_ context.Context, _ int64) error {
	s.invalidateCalls.Add(1)
	s.mu.Lock()
	s.generation++
	s.balance = s.freshBalance
	s.mu.Unlock()
	return nil
}

type generationRateLimitHitCacheStub struct {
	billingCacheWorkerStub
	mu                      sync.Mutex
	data                    *APIKeyRateLimitCacheData
	freshData               *APIKeyRateLimitCacheData
	generation              uint64
	service                 *BillingCacheService
	invalidateOnRead        atomic.Bool
	remoteInvalidateOnToken atomic.Bool
	returnGenerationChanged atomic.Bool
	tokenReadCalls          atomic.Int64
	readCalls               atomic.Int64
	invalidateCalls         atomic.Int64
}

func (s *generationRateLimitHitCacheStub) GetAPIKeyRateLimitWithGeneration(ctx context.Context, keyID int64) (*APIKeyRateLimitCacheData, uint64, error) {
	s.readCalls.Add(1)
	s.mu.Lock()
	data, generation := s.data, s.generation
	shouldInvalidate := s.invalidateOnRead.CompareAndSwap(true, false)
	s.mu.Unlock()
	if s.returnGenerationChanged.CompareAndSwap(true, false) {
		return nil, generation, ErrBillingCacheGenerationChanged
	}
	if shouldInvalidate && s.service != nil {
		if err := s.service.InvalidateAPIKeyRateLimit(ctx, keyID); err != nil {
			return nil, generation, err
		}
	}
	return data, generation, nil
}

func (s *generationRateLimitHitCacheStub) SetAPIKeyRateLimitIfGeneration(_ context.Context, _ int64, data *APIKeyRateLimitCacheData, generation uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if generation != s.generation {
		return ErrBillingCacheGenerationChanged
	}
	s.data = data
	return nil
}

func (s *generationRateLimitHitCacheStub) GetAPIKeyRateLimitGeneration(_ context.Context, _ int64) (uint64, error) {
	s.tokenReadCalls.Add(1)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.remoteInvalidateOnToken.CompareAndSwap(true, false) {
		s.generation++
		s.data = s.freshData
	}
	return s.generation, nil
}

func (s *generationRateLimitHitCacheStub) InvalidateAPIKeyRateLimit(_ context.Context, _ int64) error {
	s.invalidateCalls.Add(1)
	s.mu.Lock()
	s.generation++
	s.data = s.freshData
	s.mu.Unlock()
	return nil
}

func TestGetUserBalanceRetriesHitRacedByInvalidation(t *testing.T) {
	cache := &generationBalanceHitCacheStub{
		balance:      10,
		freshBalance: 2,
	}
	cache.invalidateOnRead.Store(true)
	svc := &BillingCacheService{cache: cache}
	cache.service = svc

	got, err := svc.GetUserBalance(context.Background(), 8101)
	require.NoError(t, err)
	require.Equal(t, 2.0, got)
	require.EqualValues(t, 2, cache.readCalls.Load())
	require.EqualValues(t, 1, cache.invalidateCalls.Load())
	require.Zero(t, cache.conditionalSetCall.Load(), "a hit retry should not perform an unnecessary refill")
}

func TestGetUserBalanceRejectsCrossProcessGenerationChangeAfterHit(t *testing.T) {
	cache := &generationBalanceHitCacheStub{balance: 10, freshBalance: 3}
	cache.remoteInvalidateOnToken.Store(true)
	svc := &BillingCacheService{cache: cache}

	got, err := svc.GetUserBalance(context.Background(), 8104)
	require.NoError(t, err)
	require.Equal(t, 3.0, got)
	require.EqualValues(t, 2, cache.readCalls.Load())
	require.EqualValues(t, 2, cache.tokenReadCalls.Load())
}

func TestGetUserBalanceRetriesConditionalGenerationRejection(t *testing.T) {
	cache := &generationBalanceHitCacheStub{}
	cache.returnMiss.Store(true)
	cache.rejectNextSet.Store(true)
	repo := &balanceLoadUserRepoStub{balance: 1}
	cache.onReject = func() { repo.balance = 2 }
	svc := &BillingCacheService{cache: cache, userRepo: repo}

	got, err := svc.GetUserBalance(context.Background(), 8103)
	require.NoError(t, err)
	require.Equal(t, 2.0, got)
	require.EqualValues(t, 2, repo.calls.Load(), "a rejected conditional refill must be retried")
	require.EqualValues(t, 2, cache.conditionalSetCall.Load())
}

func TestGetUserBalanceRetriesGenerationSentinelFromRead(t *testing.T) {
	cache := &generationBalanceHitCacheStub{}
	cache.returnGenerationChanged.Store(true)
	cache.returnMiss.Store(true)
	repo := &balanceLoadUserRepoStub{balance: 4.25}
	svc := &BillingCacheService{cache: cache, userRepo: repo}

	got, err := svc.GetUserBalance(context.Background(), 8106)
	require.NoError(t, err)
	require.Equal(t, 4.25, got)
	require.EqualValues(t, 2, cache.readCalls.Load(), "a read-side generation sentinel must restart the cache read")
	require.EqualValues(t, 1, repo.calls.Load(), "the retry should load the DB only after obtaining a fresh generation")
}

func TestCheckAPIKeyRateLimitsRetriesHitRacedByInvalidation(t *testing.T) {
	now := time.Now()
	cache := &generationRateLimitHitCacheStub{
		data: &APIKeyRateLimitCacheData{
			Usage5h:  9,
			Usage1d:  9,
			Usage7d:  9,
			Window5h: now.Unix(),
			Window1d: now.Unix(),
			Window7d: now.Unix(),
		},
		freshData: &APIKeyRateLimitCacheData{
			Usage5h:  1,
			Usage1d:  1,
			Usage7d:  1,
			Window5h: now.Unix(),
			Window1d: now.Unix(),
			Window7d: now.Unix(),
		},
	}
	cache.invalidateOnRead.Store(true)
	svc := &BillingCacheService{cache: cache}
	cache.service = svc

	err := svc.checkAPIKeyRateLimits(context.Background(), &APIKey{
		ID:          8102,
		RateLimit5h: 10,
		RateLimit1d: 10,
		RateLimit7d: 10,
	})
	require.NoError(t, err)
	require.EqualValues(t, 2, cache.readCalls.Load())
	require.EqualValues(t, 1, cache.invalidateCalls.Load())
}

func TestCheckAPIKeyRateLimitsRejectsCrossProcessGenerationChangeAfterHit(t *testing.T) {
	now := time.Now()
	cache := &generationRateLimitHitCacheStub{
		data:      &APIKeyRateLimitCacheData{Usage5h: 9, Usage1d: 9, Usage7d: 9, Window5h: now.Unix(), Window1d: now.Unix(), Window7d: now.Unix()},
		freshData: &APIKeyRateLimitCacheData{Usage5h: 1, Usage1d: 1, Usage7d: 1, Window5h: now.Unix(), Window1d: now.Unix(), Window7d: now.Unix()},
	}
	cache.remoteInvalidateOnToken.Store(true)
	svc := &BillingCacheService{cache: cache}

	err := svc.checkAPIKeyRateLimits(context.Background(), &APIKey{ID: 8105, RateLimit5h: 10, RateLimit1d: 10, RateLimit7d: 10})
	require.NoError(t, err)
	require.EqualValues(t, 2, cache.readCalls.Load())
	require.EqualValues(t, 2, cache.tokenReadCalls.Load())
}

func TestCheckAPIKeyRateLimitsRetriesGenerationSentinelFromRead(t *testing.T) {
	now := time.Now()
	cache := &generationRateLimitHitCacheStub{}
	cache.returnGenerationChanged.Store(true)
	svc := &BillingCacheService{cache: cache}
	var loaderCalls atomic.Int64
	svc.apiKeyRateLimitLoader = rateLimitLoaderFunc(func(context.Context, int64) (*APIKeyRateLimitData, error) {
		loaderCalls.Add(1)
		return &APIKeyRateLimitData{
			Usage5h:       1,
			Usage1d:       1,
			Usage7d:       1,
			Window5hStart: &now,
			Window1dStart: &now,
			Window7dStart: &now,
		}, nil
	})

	err := svc.checkAPIKeyRateLimits(context.Background(), &APIKey{ID: 8107, RateLimit5h: 10, RateLimit1d: 10, RateLimit7d: 10})
	require.NoError(t, err)
	require.EqualValues(t, 2, cache.readCalls.Load(), "a read-side generation sentinel must restart the cache read")
	require.EqualValues(t, 1, loaderCalls.Load(), "the retry should load the DB only after obtaining a fresh generation")
}

func TestCheckAPIKeyRateLimitsNilAPIKeyIsNoop(t *testing.T) {
	svc := &BillingCacheService{}
	require.NoError(t, svc.checkAPIKeyRateLimits(context.Background(), nil))
}
