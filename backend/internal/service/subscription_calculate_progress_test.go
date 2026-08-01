package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestSubscriptionService() *SubscriptionService {
	return &SubscriptionService{}
}

func ptrFloat64(v float64) *float64  { return &v }
func ptrTime(t time.Time) *time.Time { return &t }
func progressValue(t *testing.T, value *float64) float64 {
	t.Helper()
	require.NotNil(t, value)
	return *value
}

func TestCalculateProgress_BasicFields(t *testing.T) {
	svc := newTestSubscriptionService()
	now := time.Now()
	sub := &UserSubscription{
		ID:        100,
		StartsAt:  now,
		ExpiresAt: now.Add(30 * 24 * time.Hour),
	}
	group := &Group{Name: "Premium"}

	progress := svc.calculateProgress(sub, group)

	assert.Equal(t, int64(100), progress.ID)
	assert.Equal(t, "Premium", progress.GroupName)
	assert.Equal(t, sub.ExpiresAt, progress.ExpiresAt)
	assert.True(t, progress.ExpiresInDays == 29 || progress.ExpiresInDays == 30)
	assert.Nil(t, progress.Daily)
	assert.Nil(t, progress.Weekly)
	assert.Nil(t, progress.Monthly)
}

func TestCalculateProgress_ExposesAnchoredWeeklyAndConfiguredMonthlyQuota(t *testing.T) {
	svc := newTestSubscriptionService()
	now := time.Now()
	anchor := now.Add(-8 * 24 * time.Hour)
	currentStart, currentEnd, ok := AnchoredWeeklyWindow(anchor, now)
	require.True(t, ok)

	sub := &UserSubscription{
		ID:                 1,
		StartsAt:           anchor,
		ExpiresAt:          now.Add(20 * 24 * time.Hour),
		DailyWindowStart:   ptrTime(now.Add(-time.Hour)),
		WeeklyWindowStart:  ptrTime(currentStart),
		MonthlyWindowStart: ptrTime(anchor),
		DailyUsageUSD:      99,
		WeeklyUsageUSD:     25,
		MonthlyUsageUSD:    0.25,
	}
	group := &Group{
		Name:            "Pro",
		DailyLimitUSD:   ptrFloat64(1),
		WeeklyLimitUSD:  ptrFloat64(50),
		MonthlyLimitUSD: ptrFloat64(1),
	}

	progress := svc.calculateProgress(sub, group)

	assert.Nil(t, progress.Daily)
	require.NotNil(t, progress.Weekly)
	assert.Equal(t, usageWindowProgressActive, progress.Weekly.State)
	assert.Equal(t, 50.0, progress.Weekly.LimitUSD)
	assert.Equal(t, 25.0, progressValue(t, progress.Weekly.UsedUSD))
	assert.Equal(t, 25.0, progressValue(t, progress.Weekly.RemainingUSD))
	assert.Equal(t, 50.0, progressValue(t, progress.Weekly.Percentage))
	assert.True(t, progress.Weekly.WindowStart.Equal(currentStart))
	assert.True(t, progress.Weekly.ResetsAt.Equal(currentEnd))
	assert.GreaterOrEqual(t, progress.Weekly.ResetsInSeconds, int64(0))
	require.NotNil(t, progress.Monthly)
	assert.Equal(t, usageWindowProgressActive, progress.Monthly.State)
	assert.Equal(t, 1.0, progress.Monthly.LimitUSD)
	assert.Equal(t, 0.25, progressValue(t, progress.Monthly.UsedUSD))
	assert.Equal(t, 0.75, progressValue(t, progress.Monthly.RemainingUSD))
	assert.Equal(t, 25.0, progressValue(t, progress.Monthly.Percentage))
}

func TestCalculateProgress_DerivesMonthlyQuotaFromFourWeeklyAllowances(t *testing.T) {
	svc := newTestSubscriptionService()
	now := time.Now()
	weeklyLimit := 50.0
	sub := &UserSubscription{
		ID:                 1,
		StartsAt:           now,
		ExpiresAt:          now.Add(60 * 24 * time.Hour),
		WeeklyWindowStart:  ptrTime(now),
		MonthlyWindowStart: ptrTime(now),
		WeeklyUsageUSD:     5,
		MonthlyUsageUSD:    80,
	}
	group := &Group{Name: "Pro", WeeklyLimitUSD: &weeklyLimit}

	progress := svc.calculateProgress(sub, group)

	require.NotNil(t, progress.Monthly)
	assert.Equal(t, 200.0, progress.Monthly.LimitUSD)
	assert.Equal(t, 80.0, progressValue(t, progress.Monthly.UsedUSD))
	assert.Equal(t, 120.0, progressValue(t, progress.Monthly.RemainingUSD))
	assert.Equal(t, 40.0, progressValue(t, progress.Monthly.Percentage))
}

func TestCalculateProgress_ExplicitZeroMonthlyQuotaIsExhausted(t *testing.T) {
	svc := newTestSubscriptionService()
	now := time.Now()
	weeklyLimit := 50.0
	monthlyLimit := 0.0
	sub := &UserSubscription{
		ID:                 1,
		StartsAt:           now,
		ExpiresAt:          now.Add(60 * 24 * time.Hour),
		WeeklyWindowStart:  ptrTime(now),
		MonthlyWindowStart: ptrTime(now),
	}
	group := &Group{
		Name:            "Pro",
		WeeklyLimitUSD:  &weeklyLimit,
		MonthlyLimitUSD: &monthlyLimit,
	}

	progress := svc.calculateProgress(sub, group)

	require.NotNil(t, progress.Monthly)
	assert.Equal(t, usageWindowProgressExhausted, progress.Monthly.State)
	assert.Equal(t, 0.0, progress.Monthly.LimitUSD)
	assert.Equal(t, 0.0, progressValue(t, progress.Monthly.RemainingUSD))
	assert.Equal(t, 100.0, progressValue(t, progress.Monthly.Percentage))
}

func TestCalculateProgress_StaleWeeklyCounterIsNotCarriedIntoCurrentPeriod(t *testing.T) {
	svc := newTestSubscriptionService()
	now := time.Now()
	anchor := now.Add(-15 * 24 * time.Hour)
	oldStart := anchor.Add(7 * 24 * time.Hour)
	currentStart, currentEnd, ok := AnchoredWeeklyWindow(anchor, now)
	require.True(t, ok)
	require.False(t, oldStart.Equal(currentStart))

	sub := &UserSubscription{
		ID:                1,
		StartsAt:          anchor,
		ExpiresAt:         now.Add(20 * 24 * time.Hour),
		WeeklyWindowStart: &oldStart,
		WeeklyUsageUSD:    999,
	}
	group := &Group{Name: "Pro", WeeklyLimitUSD: ptrFloat64(50)}

	progress := svc.calculateProgress(sub, group)

	require.NotNil(t, progress.Weekly)
	assert.Equal(t, usageWindowProgressUnknown, progress.Weekly.State)
	assert.Nil(t, progress.Weekly.UsedUSD)
	assert.Nil(t, progress.Weekly.RemainingUSD)
	assert.Nil(t, progress.Weekly.Percentage)
	assert.True(t, progress.Weekly.WindowStart.Equal(currentStart))
	assert.True(t, progress.Weekly.ResetsAt.Equal(currentEnd))
}

func TestCalculateProgress_WeeklyOverLimitIsClamped(t *testing.T) {
	svc := newTestSubscriptionService()
	now := time.Now()
	anchor := now.Add(-24 * time.Hour)
	currentStart, _, ok := AnchoredWeeklyWindow(anchor, now)
	require.True(t, ok)
	sub := &UserSubscription{
		ID:                1,
		StartsAt:          anchor,
		ExpiresAt:         now.Add(10 * 24 * time.Hour),
		WeeklyWindowStart: &currentStart,
		WeeklyUsageUSD:    15,
	}
	group := &Group{Name: "Pro", WeeklyLimitUSD: ptrFloat64(10)}

	progress := svc.calculateProgress(sub, group)

	require.NotNil(t, progress.Weekly)
	assert.Equal(t, usageWindowProgressExhausted, progress.Weekly.State)
	assert.Equal(t, 100.0, progressValue(t, progress.Weekly.Percentage))
	assert.Equal(t, 0.0, progressValue(t, progress.Weekly.RemainingUSD))
}

func TestCalculateProgress_ExpiredSubscription(t *testing.T) {
	svc := newTestSubscriptionService()
	sub := &UserSubscription{
		ID:        1,
		StartsAt:  time.Now().Add(-48 * time.Hour),
		ExpiresAt: time.Now().Add(-24 * time.Hour),
	}

	progress := svc.calculateProgress(sub, &Group{Name: "Expired"})

	assert.Equal(t, 0, progress.ExpiresInDays)
}
