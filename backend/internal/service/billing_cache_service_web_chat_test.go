package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type webChatPeekCache struct {
	billingCacheWorkerStub
	balance               float64
	balanceErr            error
	balanceGetCalls       int
	quotaEntry            *UserPlatformQuotaCacheEntry
	quotaFound            bool
	quotaErr              error
	quotaGetCalls         int
	quotaIncrementCalls   int
	apiKeyRateUpdateCalls int
}

func (s *webChatPeekCache) GetUserBalance(context.Context, int64) (float64, error) {
	s.balanceGetCalls++
	return s.balance, s.balanceErr
}

func (s *webChatPeekCache) GetUserPlatformQuotaCache(context.Context, int64, string) (*UserPlatformQuotaCacheEntry, bool, error) {
	s.quotaGetCalls++
	return s.quotaEntry, s.quotaFound, s.quotaErr
}

func (s *webChatPeekCache) IncrUserPlatformQuotaUsageCache(context.Context, int64, string, float64, time.Duration, bool) error {
	s.quotaIncrementCalls++
	return nil
}

func (s *webChatPeekCache) UpdateAPIKeyRateLimitUsage(context.Context, int64, float64) error {
	s.apiKeyRateUpdateCalls++
	return nil
}

type webChatPeekRPM struct {
	UserRPMCache
	userCalls      int
	userGroupCalls int
}

func (s *webChatPeekRPM) IncrementUserRPM(context.Context, int64) (int, error) {
	s.userCalls++
	return 1, nil
}

func (s *webChatPeekRPM) IncrementUserGroupRPM(context.Context, int64, int64) (int, error) {
	s.userGroupCalls++
	return 1, nil
}

type webChatPeekAPIKeyRepo struct {
	APIKeyRepository
	rateLimitLoads int
}

func (s *webChatPeekAPIKeyRepo) GetRateLimitData(context.Context, int64) (*APIKeyRateLimitData, error) {
	s.rateLimitLoads++
	return nil, nil
}

type webChatPeekQuotaRepo struct {
	UserPlatformQuotaRepository
	record *UserPlatformQuotaRecord
	err    error
	calls  int
}

type webChatPeekUserRepo struct {
	UserRepository
	err   error
	calls int
}

func (s *webChatPeekUserRepo) GetByID(context.Context, int64) (*User, error) {
	s.calls++
	return nil, s.err
}

func (s *webChatPeekQuotaRepo) GetByUserPlatform(context.Context, int64, string) (*UserPlatformQuotaRecord, error) {
	s.calls++
	return s.record, s.err
}

func newWebChatPeekService(cache BillingCache, apiKeys APIKeyRepository, rpm UserRPMCache, cfg *config.Config, quotas UserPlatformQuotaRepository) *BillingCacheService {
	return NewBillingCacheService(cache, nil, nil, apiKeys, rpm, nil, cfg, quotas)
}

func TestPeekWebChatEligibilityReturnsBalanceWithoutConsumingLimits(t *testing.T) {
	limit := 10.0
	now := time.Now()
	cache := &webChatPeekCache{
		balance:    6.25,
		quotaFound: true,
		quotaEntry: &UserPlatformQuotaCacheEntry{
			SchemaVersion:      UserPlatformQuotaCacheSchemaV1,
			DailyLimitUSD:      &limit,
			DailyUsageUSD:      2,
			DailyWindowStart:   &now,
			WeeklyWindowStart:  &now,
			MonthlyWindowStart: &now,
		},
	}
	rpm := &webChatPeekRPM{}
	apiKeys := &webChatPeekAPIKeyRepo{}
	cfg := &config.Config{}
	cfg.Billing.MinimumBalanceReserve = 0.01
	svc := newWebChatPeekService(cache, apiKeys, rpm, cfg, nil)

	balance, err := svc.PeekWebChatEligibility(context.Background(), 42, PlatformOpenAI)
	require.NoError(t, err)
	require.Equal(t, 6.25, balance)
	require.Equal(t, 1, cache.quotaGetCalls)
	require.Zero(t, cache.quotaIncrementCalls)
	require.Zero(t, cache.apiKeyRateUpdateCalls)
	require.Zero(t, rpm.userCalls)
	require.Zero(t, rpm.userGroupCalls)
	require.Zero(t, apiKeys.rateLimitLoads)
}

func TestPeekWebChatEligibilityEnforcesReserveAndPlatformQuota(t *testing.T) {
	t.Run("minimum balance reserve", func(t *testing.T) {
		cache := &webChatPeekCache{balance: 0.005}
		cfg := &config.Config{}
		cfg.Billing.MinimumBalanceReserve = 0.01
		svc := newWebChatPeekService(cache, nil, nil, cfg, nil)

		balance, err := svc.PeekWebChatEligibility(context.Background(), 42, PlatformOpenAI)
		require.Equal(t, 0.005, balance)
		require.ErrorIs(t, err, ErrInsufficientBalance)
		require.Zero(t, cache.quotaGetCalls, "quota lookup should not run after balance rejection")
	})

	t.Run("OpenAI daily quota", func(t *testing.T) {
		limit := 5.0
		now := time.Now()
		cache := &webChatPeekCache{
			balance:    10,
			quotaFound: true,
			quotaEntry: &UserPlatformQuotaCacheEntry{
				SchemaVersion:     UserPlatformQuotaCacheSchemaV1,
				DailyLimitUSD:     &limit,
				DailyUsageUSD:     limit,
				DailyWindowStart:  &now,
				WeeklyWindowStart: &now,
			},
		}
		cfg := &config.Config{}
		svc := newWebChatPeekService(cache, nil, nil, cfg, nil)

		_, err := svc.PeekWebChatEligibility(context.Background(), 42, PlatformOpenAI)
		require.ErrorIs(t, err, ErrUserPlatformDailyQuotaExhausted)
		require.True(t, infraerrors.IsTooManyRequests(err))
	})
}

func TestPeekWebChatEligibilityFailsClosedWhenQuotaCannotBeConfirmed(t *testing.T) {
	cache := &webChatPeekCache{balance: 10, quotaErr: errors.New("redis unavailable")}
	quotaRepo := &webChatPeekQuotaRepo{err: errors.New("postgres unavailable")}
	cfg := &config.Config{}
	svc := newWebChatPeekService(cache, nil, nil, cfg, quotaRepo)

	_, err := svc.PeekWebChatEligibility(context.Background(), 42, PlatformOpenAI)
	require.True(t, infraerrors.IsServiceUnavailable(err))
	require.Equal(t, 1, quotaRepo.calls)
}

func assertWebChatBalanceDependencyFailure(t *testing.T, runMode string) {
	t.Helper()

	dependencyErr := errors.New("balance cache and database unavailable")
	cache := &webChatPeekCache{balanceErr: errors.New("redis unavailable")}
	users := &webChatPeekUserRepo{err: dependencyErr}
	rpm := &webChatPeekRPM{}
	apiKeys := &webChatPeekAPIKeyRepo{}
	quotas := &webChatPeekQuotaRepo{err: errors.New("quota repository must not be read")}
	cfg := &config.Config{RunMode: runMode}
	svc := NewBillingCacheService(cache, users, nil, apiKeys, rpm, nil, cfg, quotas)

	_, err := svc.PeekWebChatEligibility(context.Background(), 42, PlatformOpenAI)
	require.ErrorIs(t, err, dependencyErr)
	require.True(t, infraerrors.IsServiceUnavailable(err))
	require.Equal(t, 1, cache.balanceGetCalls)
	require.Equal(t, 1, users.calls)
	require.Zero(t, cache.quotaGetCalls)
	require.Zero(t, cache.quotaIncrementCalls)
	require.Zero(t, cache.apiKeyRateUpdateCalls)
	require.Zero(t, quotas.calls)
	require.Zero(t, rpm.userCalls)
	require.Zero(t, rpm.userGroupCalls)
	require.Zero(t, apiKeys.rateLimitLoads)
}

func TestPeekWebChatEligibilityStandardModeBalanceDependencyFailureReturns503WithoutSideEffects(t *testing.T) {
	assertWebChatBalanceDependencyFailure(t, config.RunModeStandard)
}

func TestPeekWebChatEligibilitySimpleModeBalanceDependencyFailureReturns503WithoutSideEffects(t *testing.T) {
	assertWebChatBalanceDependencyFailure(t, config.RunModeSimple)
}

func TestPeekWebChatEligibilitySimpleModeRequiresPositiveBalanceOnly(t *testing.T) {
	tests := []struct {
		name        string
		balance     float64
		wantAllowed bool
	}{
		{name: "negative balance", balance: -3},
		{name: "zero balance", balance: 0},
		{name: "positive balance below standard reserve", balance: 0.005, wantAllowed: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := &webChatPeekCache{balance: tt.balance, quotaErr: errors.New("must not be read")}
			cfg := &config.Config{RunMode: config.RunModeSimple}
			cfg.Billing.MinimumBalanceReserve = 0.01
			svc := newWebChatPeekService(cache, nil, nil, cfg, nil)

			balance, err := svc.PeekWebChatEligibility(context.Background(), 42, PlatformOpenAI)
			require.Equal(t, tt.balance, balance)
			if tt.wantAllowed {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, ErrInsufficientBalance)
			}
			require.Zero(t, cache.quotaGetCalls)
		})
	}
}

func TestCheckBillingEligibilitySimpleModeRechecksWebChatBalanceOnly(t *testing.T) {
	t.Run("web chat rejects exhausted balance", func(t *testing.T) {
		cache := &webChatPeekCache{balance: 0, quotaErr: errors.New("must not be read")}
		cfg := &config.Config{RunMode: config.RunModeSimple}
		svc := newWebChatPeekService(cache, nil, nil, cfg, nil)
		ctx := context.WithValue(context.Background(), ctxkey.WebChat, true)

		err := svc.CheckBillingEligibility(ctx, &User{ID: 42}, nil, nil, nil, PlatformOpenAI)
		require.ErrorIs(t, err, ErrInsufficientBalance)
		require.Zero(t, cache.quotaGetCalls)
	})

	t.Run("web chat allows positive balance below standard reserve", func(t *testing.T) {
		cache := &webChatPeekCache{balance: 0.005, quotaErr: errors.New("must not be read")}
		cfg := &config.Config{RunMode: config.RunModeSimple}
		cfg.Billing.MinimumBalanceReserve = 0.01
		svc := newWebChatPeekService(cache, nil, nil, cfg, nil)
		ctx := context.WithValue(context.Background(), ctxkey.WebChat, true)

		err := svc.CheckBillingEligibility(ctx, &User{ID: 42}, nil, nil, nil, PlatformOpenAI)
		require.NoError(t, err)
		require.Zero(t, cache.quotaGetCalls)
	})

	t.Run("standard API preserves simple mode bypass", func(t *testing.T) {
		cache := &webChatPeekCache{balance: -3, quotaErr: errors.New("must not be read")}
		cfg := &config.Config{RunMode: config.RunModeSimple}
		svc := newWebChatPeekService(cache, nil, nil, cfg, nil)

		err := svc.CheckBillingEligibility(context.Background(), &User{ID: 42}, nil, nil, nil, PlatformOpenAI)
		require.NoError(t, err)
		require.Zero(t, cache.quotaGetCalls)
	})
}

func TestCheckBillingEligibilitySimpleModeRechecksWebChatSubscription(t *testing.T) {
	newFixture := func(t *testing.T) (*BillingCacheService, *User, *APIKey, *Group, *UserSubscription, *subscriptionMonthlyCacheStub) {
		t.Helper()
		subscription, cacheData, _ := billingCacheMonthlyFixture(t)
		cache := &subscriptionMonthlyCacheStub{data: cacheData}
		cfg := &config.Config{RunMode: config.RunModeSimple}
		svc := NewBillingCacheService(cache, nil, nil, nil, nil, nil, cfg, nil)
		group := &Group{
			ID:               subscription.GroupID,
			Status:           StatusActive,
			Platform:         PlatformOpenAI,
			SubscriptionType: SubscriptionTypeSubscription,
			Hydrated:         true,
		}
		user := &User{ID: subscription.UserID, Status: StatusActive, Balance: 0}
		apiKey := &APIKey{
			ID:      99,
			UserID:  user.ID,
			GroupID: &group.ID,
			Status:  StatusActive,
			Purpose: APIKeyPurposeWebChat,
			User:    user,
			Group:   group,
		}
		return svc, user, apiKey, group, subscription, cache
	}

	t.Run("valid subscription does not require wallet balance", func(t *testing.T) {
		svc, user, apiKey, group, subscription, cache := newFixture(t)
		ctx := context.WithValue(context.Background(), ctxkey.WebChat, true)

		err := svc.CheckBillingEligibility(ctx, user, apiKey, group, subscription, PlatformOpenAI)
		require.NoError(t, err)
		require.Equal(t, 1, cache.getCalls)
	})

	t.Run("missing subscription context fails closed", func(t *testing.T) {
		svc, user, apiKey, group, _, cache := newFixture(t)
		ctx := context.WithValue(context.Background(), ctxkey.WebChat, true)

		err := svc.CheckBillingEligibility(ctx, user, apiKey, group, nil, PlatformOpenAI)
		require.Error(t, err)
		require.True(t, infraerrors.IsServiceUnavailable(err))
		require.Zero(t, cache.getCalls)
	})

	t.Run("exhausted subscription is rejected by final gateway check", func(t *testing.T) {
		svc, user, apiKey, group, subscription, cache := newFixture(t)
		weeklyLimit := 1.0
		group.WeeklyLimitUSD = &weeklyLimit
		cache.data.WeeklyUsage = weeklyLimit
		ctx := context.WithValue(context.Background(), ctxkey.WebChat, true)

		err := svc.CheckBillingEligibility(ctx, user, apiKey, group, subscription, PlatformOpenAI)
		require.ErrorIs(t, err, ErrWeeklyLimitExceeded)
		require.Equal(t, 1, cache.getCalls)
	})
}

func TestPeekWebChatEligibilityBusinessRejectionsCloseCircuitBreaker(t *testing.T) {
	tests := []struct {
		name  string
		cache *webChatPeekCache
		cfg   *config.Config
		want  error
	}{
		{
			name:  "insufficient balance",
			cache: &webChatPeekCache{balance: 0.005},
			cfg: func() *config.Config {
				cfg := &config.Config{}
				cfg.Billing.MinimumBalanceReserve = 0.01
				return cfg
			}(),
			want: ErrInsufficientBalance,
		},
		{
			name: "platform quota",
			cache: func() *webChatPeekCache {
				limit := 5.0
				now := time.Now()
				return &webChatPeekCache{
					balance:    10,
					quotaFound: true,
					quotaEntry: &UserPlatformQuotaCacheEntry{
						SchemaVersion:     UserPlatformQuotaCacheSchemaV1,
						DailyLimitUSD:     &limit,
						DailyUsageUSD:     limit,
						DailyWindowStart:  &now,
						WeeklyWindowStart: &now,
					},
				}
			}(),
			cfg:  &config.Config{},
			want: ErrUserPlatformDailyQuotaExhausted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.cfg.Billing.CircuitBreaker = config.CircuitBreakerConfig{
				Enabled:             true,
				FailureThreshold:    1,
				ResetTimeoutSeconds: 1,
				HalfOpenRequests:    1,
			}
			svc := newWebChatPeekService(tt.cache, nil, nil, tt.cfg, nil)
			require.NotNil(t, svc.circuitBreaker)
			svc.circuitBreaker.state = billingCircuitHalfOpen
			svc.circuitBreaker.halfOpenRemaining = 1

			_, err := svc.PeekWebChatEligibility(context.Background(), 42, PlatformOpenAI)
			require.ErrorIs(t, err, tt.want)
			require.Equal(t, billingCircuitClosed, svc.circuitBreaker.state)
			require.Zero(t, svc.circuitBreaker.failures)
			require.Zero(t, svc.circuitBreaker.halfOpenRemaining)
		})
	}
}
