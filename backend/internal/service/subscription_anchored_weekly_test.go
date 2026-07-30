package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAnchoredWeeklyWindowUsesExactHalfOpenBoundaries(t *testing.T) {
	anchor := time.Date(2026, time.July, 29, 13, 47, 12, 345678901, time.FixedZone("UTC+8", 8*60*60))
	firstEnd := anchor.Add(SubscriptionWeeklyWindowDuration)

	start, end, ok := AnchoredWeeklyWindow(anchor, anchor)
	require.True(t, ok)
	require.True(t, start.Equal(anchor))
	require.True(t, end.Equal(firstEnd))

	start, end, ok = AnchoredWeeklyWindow(anchor, firstEnd.Add(-time.Nanosecond))
	require.True(t, ok)
	require.True(t, start.Equal(anchor))
	require.True(t, end.Equal(firstEnd))

	start, end, ok = AnchoredWeeklyWindow(anchor, firstEnd)
	require.True(t, ok)
	require.True(t, start.Equal(firstEnd))
	require.True(t, end.Equal(firstEnd.Add(SubscriptionWeeklyWindowDuration)))

	start, end, ok = AnchoredWeeklyWindow(anchor, anchor.Add(25*SubscriptionWeeklyWindowDuration+3*time.Hour))
	require.True(t, ok)
	require.True(t, start.Equal(anchor.Add(25*SubscriptionWeeklyWindowDuration)))
	require.True(t, end.Equal(anchor.Add(26*SubscriptionWeeklyWindowDuration)))

	_, _, ok = AnchoredWeeklyWindow(anchor, anchor.Add(-time.Nanosecond))
	require.False(t, ok)
}

func TestAssignSubscriptionInitializesExactWeeklyAnchor(t *testing.T) {
	groupRepo := &subscriptionGroupRepoStub{
		group: &Group{ID: 1, SubscriptionType: SubscriptionTypeSubscription},
	}
	subRepo := newSubscriptionUserSubRepoStub()
	svc := NewSubscriptionService(groupRepo, subRepo, nil, nil, nil)

	sub, err := svc.AssignSubscription(context.Background(), &AssignSubscriptionInput{
		UserID:       100,
		GroupID:      1,
		ValidityDays: 30,
	})

	require.NoError(t, err)
	require.NotNil(t, sub.WeeklyWindowStart)
	require.True(t, sub.WeeklyWindowStart.Equal(sub.StartsAt))
	require.Nil(t, sub.DailyWindowStart)
	require.Nil(t, sub.MonthlyWindowStart)
}

func TestRenewedSubscriptionTermUsesExactReopenInstant(t *testing.T) {
	oldAnchor := time.Date(2026, time.July, 1, 10, 0, 0, 0, time.UTC)
	reopenedAt := time.Date(2026, time.July, 29, 13, 47, 12, 987654321, time.UTC)
	existing := &UserSubscription{
		ID:                 1,
		StartsAt:           oldAnchor,
		ExpiresAt:          oldAnchor.Add(24 * time.Hour),
		Status:             SubscriptionStatusExpired,
		DailyWindowStart:   ptrTime(oldAnchor),
		WeeklyWindowStart:  ptrTime(oldAnchor),
		MonthlyWindowStart: ptrTime(oldAnchor),
		DailyUsageUSD:      10,
		WeeklyUsageUSD:     20,
		MonthlyUsageUSD:    30,
	}

	renewed := renewedSubscriptionTerm(existing, "reopened", reopenedAt, reopenedAt.Add(30*24*time.Hour))

	require.True(t, renewed.StartsAt.Equal(reopenedAt))
	require.NotNil(t, renewed.WeeklyWindowStart)
	require.True(t, renewed.WeeklyWindowStart.Equal(reopenedAt))
	require.Nil(t, renewed.DailyWindowStart)
	require.Nil(t, renewed.MonthlyWindowStart)
	require.Zero(t, renewed.DailyUsageUSD)
	require.Zero(t, renewed.WeeklyUsageUSD)
	require.Zero(t, renewed.MonthlyUsageUSD)
}

func TestValidateAndCheckLimitsIgnoresLegacyDailyAndMonthlyLimits(t *testing.T) {
	now := time.Now()
	weeklyLimit := 10.0
	legacyLimit := 1.0
	sub := &UserSubscription{
		Status:             SubscriptionStatusActive,
		StartsAt:           now,
		ExpiresAt:          now.Add(30 * 24 * time.Hour),
		WeeklyWindowStart:  &now,
		DailyUsageUSD:      100,
		WeeklyUsageUSD:     9,
		MonthlyUsageUSD:    100,
		DailyWindowStart:   &now,
		MonthlyWindowStart: &now,
	}
	group := &Group{
		SubscriptionType: SubscriptionTypeSubscription,
		DailyLimitUSD:    &legacyLimit,
		WeeklyLimitUSD:   &weeklyLimit,
		MonthlyLimitUSD:  &legacyLimit,
	}
	svc := &SubscriptionService{}

	needsMaintenance, err := svc.ValidateAndCheckLimits(sub, group)
	require.NoError(t, err)
	require.False(t, needsMaintenance)

	sub.WeeklyUsageUSD = weeklyLimit
	_, err = svc.ValidateAndCheckLimits(sub, group)
	require.True(t, errors.Is(err, ErrWeeklyLimitExceeded))
}

func TestStaleWeeklyCounterCannotBlockNewAnchoredPeriod(t *testing.T) {
	now := time.Now()
	anchor := now.Add(-8 * 24 * time.Hour)
	limit := 10.0
	sub := &UserSubscription{
		Status:            SubscriptionStatusActive,
		StartsAt:          anchor,
		ExpiresAt:         now.Add(30 * 24 * time.Hour),
		WeeklyWindowStart: &anchor,
		WeeklyUsageUSD:    100,
	}
	group := &Group{WeeklyLimitUSD: &limit}

	require.True(t, sub.NeedsWeeklyResetAt(now))
	require.Equal(t, 0.0, sub.EffectiveWeeklyUsageAt(now))
	require.True(t, sub.CheckWeeklyLimit(group, 0))

	needsMaintenance, err := (&SubscriptionService{}).ValidateAndCheckLimits(sub, group)
	require.NoError(t, err)
	require.True(t, needsMaintenance)
}

func TestSubscriptionCacheMatchBindsConcreteEntitlementVersion(t *testing.T) {
	now := time.Now()
	anchor := now.Add(-8 * 24 * time.Hour)
	windowStart, windowEnd, ok := AnchoredWeeklyWindow(anchor, now)
	require.True(t, ok)
	updatedAt := now.Add(-time.Minute)
	sub := &UserSubscription{
		ID:                42,
		Status:            SubscriptionStatusActive,
		StartsAt:          anchor,
		ExpiresAt:         now.Add(30 * 24 * time.Hour),
		WeeklyWindowStart: &windowStart,
		UpdatedAt:         updatedAt,
	}
	data := &subscriptionCacheData{
		SubscriptionID:    sub.ID,
		Status:            sub.Status,
		StartsAt:          sub.StartsAt,
		ExpiresAt:         sub.ExpiresAt,
		WeeklyWindowStart: &windowStart,
		WeeklyWindowEnd:   windowEnd,
		Version:           updatedAt.UnixMicro(),
	}

	require.True(t, subscriptionCacheMatchesEntitlement(data, sub, now))

	staleID := *data
	staleID.SubscriptionID++
	require.False(t, subscriptionCacheMatchesEntitlement(&staleID, sub, now))

	staleAnchor := *data
	staleAnchor.StartsAt = staleAnchor.StartsAt.Add(time.Second)
	require.False(t, subscriptionCacheMatchesEntitlement(&staleAnchor, sub, now))

	staleVersion := *data
	staleVersion.Version--
	require.False(t, subscriptionCacheMatchesEntitlement(&staleVersion, sub, now))
}
