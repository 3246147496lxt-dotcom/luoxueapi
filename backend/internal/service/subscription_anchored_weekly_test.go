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
	require.NotNil(t, sub.MonthlyWindowStart)
	require.True(t, sub.MonthlyWindowStart.Equal(sub.StartsAt))
}

func TestAssignSubscriptionToDifferentGroupCreatesNewExactAnchor(t *testing.T) {
	oldAnchor := time.Date(2026, time.July, 1, 10, 11, 12, 123456789, time.UTC)
	groupRepo := &subscriptionGroupRepoStub{
		group: &Group{SubscriptionType: SubscriptionTypeSubscription},
	}
	subRepo := newSubscriptionUserSubRepoStub()
	subRepo.seed(&UserSubscription{
		ID:                 1,
		UserID:             100,
		GroupID:            1,
		Status:             SubscriptionStatusActive,
		StartsAt:           oldAnchor,
		ExpiresAt:          oldAnchor.Add(90 * 24 * time.Hour),
		WeeklyWindowStart:  ptrTime(oldAnchor),
		MonthlyWindowStart: ptrTime(oldAnchor),
	})
	svc := NewSubscriptionService(groupRepo, subRepo, nil, nil, nil)

	before := time.Now()
	sub, err := svc.AssignSubscription(context.Background(), &AssignSubscriptionInput{
		UserID:       100,
		GroupID:      2,
		ValidityDays: 30,
	})
	after := time.Now()

	require.NoError(t, err)
	require.Equal(t, int64(2), sub.GroupID)
	require.False(t, sub.StartsAt.Equal(oldAnchor))
	require.False(t, sub.StartsAt.Before(before))
	require.False(t, sub.StartsAt.After(after))
	require.NotNil(t, sub.WeeklyWindowStart)
	require.True(t, sub.WeeklyWindowStart.Equal(sub.StartsAt))
	require.NotNil(t, sub.MonthlyWindowStart)
	require.True(t, sub.MonthlyWindowStart.Equal(sub.StartsAt))
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
	require.NotNil(t, renewed.MonthlyWindowStart)
	require.True(t, renewed.MonthlyWindowStart.Equal(reopenedAt))
	require.Zero(t, renewed.DailyUsageUSD)
	require.Zero(t, renewed.WeeklyUsageUSD)
	require.Zero(t, renewed.MonthlyUsageUSD)
}

func TestValidateAndCheckLimitsEnforcesWeeklyAndConfiguredMonthlyLimits(t *testing.T) {
	now := time.Now()
	weeklyLimit := 10.0
	monthlyLimit := 100.0
	sub := &UserSubscription{
		Status:             SubscriptionStatusActive,
		StartsAt:           now,
		ExpiresAt:          now.Add(30 * 24 * time.Hour),
		WeeklyWindowStart:  &now,
		DailyUsageUSD:      100,
		WeeklyUsageUSD:     9,
		MonthlyUsageUSD:    99,
		DailyWindowStart:   &now,
		MonthlyWindowStart: &now,
	}
	group := &Group{
		SubscriptionType: SubscriptionTypeSubscription,
		DailyLimitUSD:    &monthlyLimit,
		WeeklyLimitUSD:   &weeklyLimit,
		MonthlyLimitUSD:  &monthlyLimit,
	}
	svc := &SubscriptionService{}

	needsMaintenance, err := svc.ValidateAndCheckLimits(sub, group)
	require.NoError(t, err)
	require.False(t, needsMaintenance)

	sub.WeeklyUsageUSD = weeklyLimit
	_, err = svc.ValidateAndCheckLimits(sub, group)
	require.True(t, errors.Is(err, ErrWeeklyLimitExceeded))

	sub.WeeklyUsageUSD = 0
	sub.MonthlyUsageUSD = monthlyLimit
	_, err = svc.ValidateAndCheckLimits(sub, group)
	require.True(t, errors.Is(err, ErrMonthlyLimitExceeded))
}

func TestValidateAndCheckLimitsDerivesMonthlyLimitFromFourWeeklyAllowances(t *testing.T) {
	now := time.Now()
	weeklyLimit := 10.0
	sub := &UserSubscription{
		Status:             SubscriptionStatusActive,
		StartsAt:           now,
		ExpiresAt:          now.Add(60 * 24 * time.Hour),
		WeeklyWindowStart:  &now,
		MonthlyWindowStart: &now,
		MonthlyUsageUSD:    40,
	}
	group := &Group{
		SubscriptionType: SubscriptionTypeSubscription,
		WeeklyLimitUSD:   &weeklyLimit,
	}

	_, err := (&SubscriptionService{}).ValidateAndCheckLimits(sub, group)
	require.True(t, errors.Is(err, ErrMonthlyLimitExceeded))
}

func TestValidateAndCheckLimitsExplicitZeroMonthlyLimitDoesNotFallback(t *testing.T) {
	now := time.Now()
	weeklyLimit := 10.0
	monthlyLimit := 0.0
	sub := &UserSubscription{
		Status:             SubscriptionStatusActive,
		StartsAt:           now,
		ExpiresAt:          now.Add(60 * 24 * time.Hour),
		WeeklyWindowStart:  &now,
		MonthlyWindowStart: &now,
	}
	group := &Group{
		SubscriptionType: SubscriptionTypeSubscription,
		WeeklyLimitUSD:   &weeklyLimit,
		MonthlyLimitUSD:  &monthlyLimit,
	}

	_, err := (&SubscriptionService{}).ValidateAndCheckLimits(sub, group)
	require.True(t, errors.Is(err, ErrMonthlyLimitExceeded))
}

func TestAnchoredMonthlyWindowUsesExactHalfOpenThirtyDayBoundaries(t *testing.T) {
	anchor := time.Date(2026, time.July, 29, 13, 47, 12, 345678901, time.FixedZone("UTC+8", 8*60*60))
	firstEnd := anchor.Add(SubscriptionMonthlyWindowDuration)

	start, end, ok := AnchoredMonthlyWindow(anchor, anchor)
	require.True(t, ok)
	require.True(t, start.Equal(anchor))
	require.True(t, end.Equal(firstEnd))

	start, end, ok = AnchoredMonthlyWindow(anchor, firstEnd.Add(-time.Nanosecond))
	require.True(t, ok)
	require.True(t, start.Equal(anchor))
	require.True(t, end.Equal(firstEnd))

	start, end, ok = AnchoredMonthlyWindow(anchor, firstEnd)
	require.True(t, ok)
	require.True(t, start.Equal(firstEnd))
	require.True(t, end.Equal(firstEnd.Add(SubscriptionMonthlyWindowDuration)))

	start, end, ok = AnchoredMonthlyWindow(
		anchor,
		anchor.Add(25*SubscriptionMonthlyWindowDuration+3*time.Hour),
	)
	require.True(t, ok)
	require.True(t, start.Equal(anchor.Add(25*SubscriptionMonthlyWindowDuration)))
	require.True(t, end.Equal(anchor.Add(26*SubscriptionMonthlyWindowDuration)))

	_, _, ok = AnchoredMonthlyWindow(anchor, anchor.Add(-time.Nanosecond))
	require.False(t, ok)
}

func TestMonthlyWindowStillBlocksBetweenDayTwentyEightAndThirty(t *testing.T) {
	anchor := time.Now().Add(-29 * 24 * time.Hour)
	now := time.Now()
	weeklyStart, _, weeklyOK := AnchoredWeeklyWindow(anchor, now)
	monthlyStart, _, monthlyOK := AnchoredMonthlyWindow(anchor, now)
	require.True(t, weeklyOK)
	require.True(t, monthlyOK)
	require.True(t, weeklyStart.Equal(anchor.Add(28*24*time.Hour)))
	require.True(t, monthlyStart.Equal(anchor))

	weeklyLimit := 10.0
	sub := &UserSubscription{
		Status:             SubscriptionStatusActive,
		StartsAt:           anchor,
		ExpiresAt:          now.Add(31 * 24 * time.Hour),
		WeeklyWindowStart:  &weeklyStart,
		MonthlyWindowStart: &monthlyStart,
		WeeklyUsageUSD:     1,
		MonthlyUsageUSD:    40,
	}
	group := &Group{WeeklyLimitUSD: &weeklyLimit}

	_, err := (&SubscriptionService{}).ValidateAndCheckLimits(sub, group)
	require.True(t, errors.Is(err, ErrMonthlyLimitExceeded))
}

func TestSubscriptionExpiryUsesHalfOpenEntitlementBoundary(t *testing.T) {
	expiresAt := time.Date(2026, time.August, 1, 12, 0, 0, 0, time.UTC)
	sub := &UserSubscription{ExpiresAt: expiresAt}

	require.False(t, sub.IsExpiredAt(expiresAt.Add(-time.Nanosecond)))
	require.True(t, sub.IsExpiredAt(expiresAt))
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

func TestSubscriptionCacheMatchBindsConcreteEntitlementTerm(t *testing.T) {
	now := time.Now()
	anchor := now.Add(-8 * 24 * time.Hour)
	windowStart, windowEnd, ok := AnchoredWeeklyWindow(anchor, now)
	require.True(t, ok)
	monthlyWindowStart, monthlyWindowEnd, ok := AnchoredMonthlyWindow(anchor, now)
	require.True(t, ok)
	updatedAt := now.Add(-time.Minute)
	sub := &UserSubscription{
		ID:                 42,
		Status:             SubscriptionStatusActive,
		StartsAt:           anchor,
		ExpiresAt:          now.Add(30 * 24 * time.Hour),
		WeeklyWindowStart:  &windowStart,
		MonthlyWindowStart: &monthlyWindowStart,
		UpdatedAt:          updatedAt,
	}
	data := &subscriptionCacheData{
		SubscriptionID:     sub.ID,
		Status:             sub.Status,
		StartsAt:           sub.StartsAt,
		ExpiresAt:          sub.ExpiresAt,
		WeeklyWindowStart:  &windowStart,
		WeeklyWindowEnd:    windowEnd,
		MonthlyWindowStart: &monthlyWindowStart,
		MonthlyWindowEnd:   monthlyWindowEnd,
		Version:            updatedAt.UnixMicro(),
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
	require.False(t, subscriptionCacheMatchesEntitlement(&staleVersion, sub, now),
		"an older Redis usage snapshot must be reloaded from the authoritative DB")

	newerVersion := *data
	newerVersion.Version++
	require.True(t, subscriptionCacheMatchesEntitlement(&newerVersion, sub, now),
		"a newer Redis usage snapshot remains compatible with a stale L1 entitlement")

	staleMonthlyWindow := *data
	staleMonthlyWindow.MonthlyWindowEnd = staleMonthlyWindow.MonthlyWindowEnd.Add(time.Second)
	require.False(t, subscriptionCacheMatchesEntitlement(&staleMonthlyWindow, sub, now))
}
