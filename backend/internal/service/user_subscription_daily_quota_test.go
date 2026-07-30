//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type anchoredResetTrackingUserSubRepo struct {
	userSubRepoNoop

	resetWeeklyCalled bool
	resetStart        time.Time
}

func (r *anchoredResetTrackingUserSubRepo) ResetWeeklyUsage(_ context.Context, _ int64, _ *time.Time, start time.Time) error {
	r.resetWeeklyCalled = true
	r.resetStart = start
	return nil
}

type continuousRenewalUserSubRepo struct {
	*subscriptionUserSubRepoStub
}

func (r *continuousRenewalUserSubRepo) ExtendExpiry(_ context.Context, id int64, expiresAt time.Time) error {
	sub := r.byID[id]
	if sub == nil {
		return ErrSubscriptionNotFound
	}
	sub.ExpiresAt = expiresAt
	return nil
}

func (r *continuousRenewalUserSubRepo) UpdateStatus(_ context.Context, id int64, status string) error {
	sub := r.byID[id]
	if sub == nil {
		return ErrSubscriptionNotFound
	}
	sub.Status = status
	return nil
}

func (r *continuousRenewalUserSubRepo) UpdateNotes(_ context.Context, id int64, notes string) error {
	sub := r.byID[id]
	if sub == nil {
		return ErrSubscriptionNotFound
	}
	sub.Notes = notes
	return nil
}

func TestAssignOrExtendSubscription_ExpiredTermGetsExactNewAnchor(t *testing.T) {
	groupRepo := &subscriptionGroupRepoStub{
		group: &Group{ID: 1, SubscriptionType: SubscriptionTypeSubscription},
	}
	subRepo := newSubscriptionUserSubRepoStub()
	oldStart := time.Now().Add(-30 * 24 * time.Hour)
	subRepo.seed(&UserSubscription{
		ID:                 100,
		UserID:             200,
		GroupID:            1,
		StartsAt:           oldStart,
		ExpiresAt:          oldStart.Add(24 * time.Hour),
		Status:             SubscriptionStatusExpired,
		DailyWindowStart:   ptrTime(oldStart),
		WeeklyWindowStart:  ptrTime(oldStart),
		MonthlyWindowStart: ptrTime(oldStart),
		DailyUsageUSD:      10,
		WeeklyUsageUSD:     20,
		MonthlyUsageUSD:    30,
	})
	svc := NewSubscriptionService(groupRepo, subRepo, nil, nil, nil)
	before := time.Now()

	renewed, reused, err := svc.AssignOrExtendSubscription(context.Background(), &AssignSubscriptionInput{
		UserID:       200,
		GroupID:      1,
		ValidityDays: 30,
	})

	after := time.Now()
	require.NoError(t, err)
	require.True(t, reused)
	require.False(t, renewed.StartsAt.Before(before))
	require.False(t, renewed.StartsAt.After(after))
	require.NotNil(t, renewed.WeeklyWindowStart)
	require.True(t, renewed.WeeklyWindowStart.Equal(renewed.StartsAt))
	require.Nil(t, renewed.DailyWindowStart)
	require.Nil(t, renewed.MonthlyWindowStart)
	require.Zero(t, renewed.DailyUsageUSD)
	require.Zero(t, renewed.WeeklyUsageUSD)
	require.Zero(t, renewed.MonthlyUsageUSD)
}

func TestAssignOrExtendSubscription_UninterruptedRenewalPreservesAnchorAndUsage(t *testing.T) {
	groupRepo := &subscriptionGroupRepoStub{
		group: &Group{ID: 1, SubscriptionType: SubscriptionTypeSubscription},
	}
	baseRepo := newSubscriptionUserSubRepoStub()
	anchor := time.Now().Add(-48 * time.Hour)
	oldExpiry := time.Now().Add(10 * 24 * time.Hour)
	baseRepo.seed(&UserSubscription{
		ID:                101,
		UserID:            201,
		GroupID:           1,
		StartsAt:          anchor,
		ExpiresAt:         oldExpiry,
		Status:            SubscriptionStatusActive,
		WeeklyWindowStart: &anchor,
		WeeklyUsageUSD:    4,
	})
	repo := &continuousRenewalUserSubRepo{subscriptionUserSubRepoStub: baseRepo}
	svc := NewSubscriptionService(groupRepo, repo, nil, nil, nil)

	renewed, reused, err := svc.AssignOrExtendSubscription(context.Background(), &AssignSubscriptionInput{
		UserID:       201,
		GroupID:      1,
		ValidityDays: 7,
	})

	require.NoError(t, err)
	require.True(t, reused)
	require.True(t, renewed.StartsAt.Equal(anchor))
	require.NotNil(t, renewed.WeeklyWindowStart)
	require.True(t, renewed.WeeklyWindowStart.Equal(anchor))
	require.Equal(t, 4.0, renewed.WeeklyUsageUSD)
	require.True(t, renewed.ExpiresAt.Equal(oldExpiry.AddDate(0, 0, 7)))
}

func TestCheckAndResetWindowsAdvancesOnlyAnchoredWeeklyWindow(t *testing.T) {
	now := time.Now()
	anchor := now.Add(-15 * 24 * time.Hour)
	currentStart, _, ok := AnchoredWeeklyWindow(anchor, now)
	require.True(t, ok)
	repo := &anchoredResetTrackingUserSubRepo{}
	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)
	sub := &UserSubscription{
		ID:                 1,
		UserID:             10,
		GroupID:            20,
		StartsAt:           anchor,
		ExpiresAt:          now.Add(30 * 24 * time.Hour),
		DailyWindowStart:   ptrTime(anchor),
		WeeklyWindowStart:  ptrTime(anchor),
		MonthlyWindowStart: ptrTime(anchor),
		DailyUsageUSD:      10,
		WeeklyUsageUSD:     20,
		MonthlyUsageUSD:    30,
	}

	err := svc.CheckAndResetWindows(context.Background(), sub)

	require.NoError(t, err)
	require.True(t, repo.resetWeeklyCalled)
	require.True(t, repo.resetStart.Equal(currentStart))
	require.NotNil(t, sub.WeeklyWindowStart)
	require.True(t, sub.WeeklyWindowStart.Equal(currentStart))
	require.Zero(t, sub.WeeklyUsageUSD)
	require.Equal(t, 10.0, sub.DailyUsageUSD)
	require.Equal(t, 30.0, sub.MonthlyUsageUSD)
}
