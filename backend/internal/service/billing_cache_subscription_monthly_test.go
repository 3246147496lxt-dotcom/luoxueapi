package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type subscriptionMonthlyCacheStub struct {
	BillingCache

	data          *SubscriptionCacheData
	setData       *SubscriptionCacheData
	setErr        error
	getCalls      int
	invalidations int
	invalidateErr error
	publishedKeys []string
	publishErr    error
}

func (s *subscriptionMonthlyCacheStub) GetSubscriptionCache(
	context.Context,
	int64,
	int64,
) (*SubscriptionCacheData, error) {
	s.getCalls++
	return s.data, nil
}

func (s *subscriptionMonthlyCacheStub) SetSubscriptionCache(
	_ context.Context,
	_, _ int64,
	data *SubscriptionCacheData,
) error {
	if s.setErr != nil {
		return s.setErr
	}
	s.setData = data
	s.data = data
	return nil
}

func (s *subscriptionMonthlyCacheStub) InvalidateSubscriptionCache(
	context.Context,
	int64,
	int64,
) error {
	s.invalidations++
	return s.invalidateErr
}

func (s *subscriptionMonthlyCacheStub) PublishSubscriptionCacheInvalidation(
	_ context.Context,
	cacheKey string,
) error {
	s.publishedKeys = append(s.publishedKeys, cacheKey)
	return s.publishErr
}

func (s *subscriptionMonthlyCacheStub) SubscribeSubscriptionCacheInvalidation(
	context.Context,
	func(cacheKey string),
) error {
	return nil
}

type subscriptionMonthlyRepoStub struct {
	userSubRepoNoop
	sub   *UserSubscription
	calls int
}

func (s *subscriptionMonthlyRepoStub) GetActiveByUserIDAndGroupID(
	context.Context,
	int64,
	int64,
) (*UserSubscription, error) {
	s.calls++
	return s.sub, nil
}

func billingCacheMonthlyFixture(
	t *testing.T,
) (*UserSubscription, *SubscriptionCacheData, time.Time) {
	t.Helper()

	now := time.Now()
	startsAt := now.Add(-8 * 24 * time.Hour)
	weeklyStart, weeklyEnd, weeklyOK := AnchoredWeeklyWindow(startsAt, now)
	require.True(t, weeklyOK)
	monthlyStart, monthlyEnd, monthlyOK := AnchoredMonthlyWindow(startsAt, now)
	require.True(t, monthlyOK)
	updatedAt := now.Add(-time.Minute)

	subscription := &UserSubscription{
		ID:                 501,
		UserID:             42,
		GroupID:            10,
		Status:             SubscriptionStatusActive,
		StartsAt:           startsAt,
		ExpiresAt:          now.Add(60 * 24 * time.Hour),
		WeeklyWindowStart:  &weeklyStart,
		MonthlyWindowStart: &monthlyStart,
		UpdatedAt:          updatedAt,
	}
	cacheData := &SubscriptionCacheData{
		SubscriptionID:     subscription.ID,
		Status:             subscription.Status,
		StartsAt:           subscription.StartsAt,
		ExpiresAt:          subscription.ExpiresAt,
		WeeklyWindowStart:  &weeklyStart,
		WeeklyWindowEnd:    weeklyEnd,
		MonthlyWindowStart: &monthlyStart,
		MonthlyWindowEnd:   monthlyEnd,
		Version:            updatedAt.UnixMicro(),
	}
	return subscription, cacheData, now
}

func billingCacheFloat64Ptr(value float64) *float64 {
	return &value
}

func TestBillingCacheSubscriptionEligibilityEnforcesEffectiveMonthlyLimit(t *testing.T) {
	tests := []struct {
		name          string
		weeklyLimit   *float64
		monthlyLimit  *float64
		weeklyUsage   float64
		monthlyUsage  float64
		expectedError error
	}{
		{
			name:          "configured monthly limit",
			weeklyLimit:   billingCacheFloat64Ptr(10),
			monthlyLimit:  billingCacheFloat64Ptr(25),
			weeklyUsage:   1,
			monthlyUsage:  25,
			expectedError: ErrMonthlyLimitExceeded,
		},
		{
			name:          "four weekly allowances fallback",
			weeklyLimit:   billingCacheFloat64Ptr(10),
			weeklyUsage:   1,
			monthlyUsage:  40,
			expectedError: ErrMonthlyLimitExceeded,
		},
		{
			name:          "explicit zero monthly allowance",
			weeklyLimit:   billingCacheFloat64Ptr(10),
			monthlyLimit:  billingCacheFloat64Ptr(0),
			weeklyUsage:   0,
			monthlyUsage:  0,
			expectedError: ErrMonthlyLimitExceeded,
		},
		{
			name:         "usage below both limits",
			weeklyLimit:  billingCacheFloat64Ptr(10),
			weeklyUsage:  9,
			monthlyUsage: 39.99,
		},
		{
			name:          "weekly exhaustion wins",
			weeklyLimit:   billingCacheFloat64Ptr(10),
			monthlyLimit:  billingCacheFloat64Ptr(25),
			weeklyUsage:   10,
			monthlyUsage:  25,
			expectedError: ErrWeeklyLimitExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subscription, cacheData, _ := billingCacheMonthlyFixture(t)
			cacheData.WeeklyUsage = tt.weeklyUsage
			cacheData.MonthlyUsage = tt.monthlyUsage
			cache := &subscriptionMonthlyCacheStub{data: cacheData}
			svc := &BillingCacheService{cache: cache}
			group := &Group{
				ID:               subscription.GroupID,
				SubscriptionType: SubscriptionTypeSubscription,
				WeeklyLimitUSD:   tt.weeklyLimit,
				MonthlyLimitUSD:  tt.monthlyLimit,
			}

			err := svc.checkSubscriptionEligibility(
				context.Background(),
				subscription.UserID,
				group,
				subscription,
			)

			if tt.expectedError == nil {
				require.NoError(t, err)
			} else {
				require.True(t, errors.Is(err, tt.expectedError), "unexpected error: %v", err)
			}
		})
	}
}

func TestGetSubscriptionStatusReloadsCacheWithAuthoritativeMonthlyWindow(t *testing.T) {
	subscription, staleCacheData, _ := billingCacheMonthlyFixture(t)
	staleStart := staleCacheData.MonthlyWindowStart.Add(-SubscriptionMonthlyWindowDuration)
	staleCacheData.MonthlyWindowStart = &staleStart
	staleCacheData.MonthlyWindowEnd = staleStart.Add(SubscriptionMonthlyWindowDuration)

	subscription.WeeklyUsageUSD = 2
	subscription.MonthlyUsageUSD = 8
	cache := &subscriptionMonthlyCacheStub{data: staleCacheData}
	svc := &BillingCacheService{
		cache:   cache,
		subRepo: &subscriptionMonthlyRepoStub{sub: subscription},
	}

	got, err := svc.GetSubscriptionStatus(
		context.Background(),
		subscription.UserID,
		subscription.GroupID,
	)

	require.NoError(t, err)
	require.Equal(t, 1, cache.invalidations)
	require.Equal(t, subscription.MonthlyUsageUSD, got.MonthlyUsage)
	require.NotNil(t, got.MonthlyWindowStart)
	require.True(t, got.MonthlyWindowStart.Equal(*subscription.MonthlyWindowStart))
	require.NotNil(t, cache.setData)
	require.True(t, cache.setData.MonthlyWindowEnd.Equal(got.MonthlyWindowEnd))
}

func TestGetSubscriptionFromDBRejectsUnmaintainedMonthlyWindow(t *testing.T) {
	subscription, _, _ := billingCacheMonthlyFixture(t)
	staleStart := subscription.MonthlyWindowStart.Add(-SubscriptionMonthlyWindowDuration)
	subscription.MonthlyWindowStart = &staleStart
	svc := &BillingCacheService{
		subRepo: &subscriptionMonthlyRepoStub{sub: subscription},
	}

	_, err := svc.getSubscriptionFromDB(
		context.Background(),
		subscription.UserID,
		subscription.GroupID,
	)

	require.ErrorContains(t, err, "anchored monthly window is not maintained")
}

func TestSubscriptionCacheWindowIsNotCurrentAtExpiryBoundary(t *testing.T) {
	_, cacheData, now := billingCacheMonthlyFixture(t)
	cacheData.ExpiresAt = now
	data := (&BillingCacheService{}).convertFromPortsData(cacheData)

	require.True(t, data.ExpiresAt.Equal(now))
	require.False(t, subscriptionCacheWindowIsCurrent(data, now))
}

func TestSubscriptionCacheInvalidationFailureFencesStaleRedisUntilDBRefresh(t *testing.T) {
	subscription, staleCacheData, _ := billingCacheMonthlyFixture(t)
	staleCacheData.WeeklyUsage = 1
	staleCacheData.MonthlyUsage = 1
	subscription.WeeklyUsageUSD = 9
	subscription.MonthlyUsageUSD = 29

	invalidateErr := errors.New("redis delete timed out")
	cache := &subscriptionMonthlyCacheStub{
		data:          staleCacheData,
		invalidateErr: invalidateErr,
	}
	repo := &subscriptionMonthlyRepoStub{sub: subscription}
	svc := &BillingCacheService{cache: cache, subRepo: repo}

	require.ErrorIs(
		t,
		svc.InvalidateSubscription(
			context.Background(),
			subscription.UserID,
			subscription.GroupID,
		),
		invalidateErr,
	)

	got, err := svc.GetSubscriptionStatus(
		context.Background(),
		subscription.UserID,
		subscription.GroupID,
	)
	require.NoError(t, err)
	require.Equal(t, subscription.WeeklyUsageUSD, got.WeeklyUsage)
	require.Equal(t, subscription.MonthlyUsageUSD, got.MonthlyUsage)
	require.Equal(t, 0, cache.getCalls, "dirty cache must be bypassed")
	require.Equal(t, 1, repo.calls)

	// The successful authoritative refresh clears only the observed fence; the
	// next read may safely use the newly written Redis snapshot.
	got, err = svc.GetSubscriptionStatus(
		context.Background(),
		subscription.UserID,
		subscription.GroupID,
	)
	require.NoError(t, err)
	require.Equal(t, subscription.MonthlyUsageUSD, got.MonthlyUsage)
	require.Equal(t, 1, cache.getCalls)
	require.Equal(t, 1, repo.calls)
}

func TestSubscriptionCacheSetFailureKeepsDBBypassFence(t *testing.T) {
	subscription, staleCacheData, _ := billingCacheMonthlyFixture(t)
	subscription.WeeklyUsageUSD = 9
	subscription.MonthlyUsageUSD = 29
	cache := &subscriptionMonthlyCacheStub{
		data:          staleCacheData,
		invalidateErr: errors.New("redis delete failed"),
		setErr:        errors.New("redis set failed"),
	}
	repo := &subscriptionMonthlyRepoStub{sub: subscription}
	svc := &BillingCacheService{cache: cache, subRepo: repo}

	require.Error(t, svc.InvalidateSubscription(
		context.Background(),
		subscription.UserID,
		subscription.GroupID,
	))

	for range 2 {
		got, err := svc.GetSubscriptionStatus(
			context.Background(),
			subscription.UserID,
			subscription.GroupID,
		)
		require.NoError(t, err)
		require.Equal(t, subscription.MonthlyUsageUSD, got.MonthlyUsage)
	}
	require.Equal(t, 0, cache.getCalls, "failed refresh must keep bypassing stale Redis")
	require.Equal(t, 2, repo.calls)
}

func TestUsageBillingInvalidationPublishesAcrossInstancesEvenWhenDeleteFails(t *testing.T) {
	cache := &subscriptionMonthlyCacheStub{
		invalidateErr: errors.New("redis delete failed"),
	}
	svc := &BillingCacheService{cache: cache}

	invalidateSubscriptionBillingCaches(
		context.Background(),
		svc,
		42,
		10,
		"test",
	)

	require.Equal(t, 1, cache.invalidations)
	require.Equal(t, []string{subCacheKey(42, 10)}, cache.publishedKeys)
}
