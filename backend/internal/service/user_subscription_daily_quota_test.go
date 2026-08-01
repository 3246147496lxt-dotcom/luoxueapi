//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type anchoredResetTrackingUserSubRepo struct {
	userSubRepoNoop

	resetWeeklyCalled  bool
	resetWeeklyStart   time.Time
	resetMonthlyCalled bool
	resetMonthlyStart  time.Time
	resetMonthlyErr    error
}

func (r *anchoredResetTrackingUserSubRepo) ResetWeeklyUsage(_ context.Context, _ int64, _ *time.Time, start time.Time) error {
	r.resetWeeklyCalled = true
	r.resetWeeklyStart = start
	return nil
}

func (r *anchoredResetTrackingUserSubRepo) ResetMonthlyUsage(_ context.Context, _ int64, _ *time.Time, start time.Time) error {
	r.resetMonthlyCalled = true
	r.resetMonthlyStart = start
	return r.resetMonthlyErr
}

type continuousRenewalUserSubRepo struct {
	*subscriptionUserSubRepoStub
}

type subscriptionLookupFailureRepo struct {
	userSubRepoNoop
	err          error
	createCalled bool
}

func (r *subscriptionLookupFailureRepo) GetByUserIDAndGroupID(context.Context, int64, int64) (*UserSubscription, error) {
	return nil, r.err
}

func (r *subscriptionLookupFailureRepo) Create(context.Context, *UserSubscription) error {
	r.createCalled = true
	return nil
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
	require.NotNil(t, renewed.MonthlyWindowStart)
	require.True(t, renewed.MonthlyWindowStart.Equal(renewed.StartsAt))
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
		ID:                 101,
		UserID:             201,
		GroupID:            1,
		StartsAt:           anchor,
		ExpiresAt:          oldExpiry,
		Status:             SubscriptionStatusActive,
		WeeklyWindowStart:  &anchor,
		MonthlyWindowStart: &anchor,
		WeeklyUsageUSD:     4,
		MonthlyUsageUSD:    9,
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
	require.NotNil(t, renewed.MonthlyWindowStart)
	require.True(t, renewed.MonthlyWindowStart.Equal(anchor))
	require.Equal(t, 9.0, renewed.MonthlyUsageUSD)
	require.True(t, renewed.ExpiresAt.Equal(oldExpiry.AddDate(0, 0, 7)))
}

func TestAssignOrExtendSubscription_DoesNotCreateAfterLookupFailure(t *testing.T) {
	dbErr := errors.New("subscription lookup unavailable")
	groupRepo := &subscriptionGroupRepoStub{
		group: &Group{ID: 1, SubscriptionType: SubscriptionTypeSubscription},
	}
	subRepo := &subscriptionLookupFailureRepo{err: dbErr}
	svc := NewSubscriptionService(groupRepo, subRepo, nil, nil, nil)

	_, reused, err := svc.AssignOrExtendSubscription(context.Background(), &AssignSubscriptionInput{
		UserID:       202,
		GroupID:      1,
		ValidityDays: 7,
	})

	require.ErrorIs(t, err, dbErr)
	require.False(t, reused)
	require.False(t, subRepo.createCalled, "a failed lookup must not be mistaken for a missing subscription")
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
	require.True(t, repo.resetWeeklyStart.Equal(currentStart))
	require.NotNil(t, sub.WeeklyWindowStart)
	require.True(t, sub.WeeklyWindowStart.Equal(currentStart))
	require.Zero(t, sub.WeeklyUsageUSD)
	require.Equal(t, 10.0, sub.DailyUsageUSD)
	require.Equal(t, 30.0, sub.MonthlyUsageUSD)
}

func TestCheckAndActivateWindowResetsStaleWeeklyUsageWhenMonthlyMarkerIsMissing(t *testing.T) {
	now := time.Now()
	anchor := now.Add(-15 * 24 * time.Hour)
	weeklyStart, _, weeklyOK := AnchoredWeeklyWindow(anchor, now)
	monthlyStart, _, monthlyOK := AnchoredMonthlyWindow(anchor, now)
	require.True(t, weeklyOK)
	require.True(t, monthlyOK)
	repo := &anchoredResetTrackingUserSubRepo{}
	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)
	sub := &UserSubscription{
		ID:                 2,
		UserID:             10,
		GroupID:            20,
		StartsAt:           anchor,
		ExpiresAt:          now.Add(30 * 24 * time.Hour),
		WeeklyWindowStart:  ptrTime(anchor),
		MonthlyWindowStart: nil,
		WeeklyUsageUSD:     999,
		MonthlyUsageUSD:    999,
	}

	err := svc.CheckAndActivateWindow(context.Background(), sub)

	require.NoError(t, err)
	require.True(t, repo.resetWeeklyCalled)
	require.True(t, repo.resetMonthlyCalled)
	require.True(t, repo.resetWeeklyStart.Equal(weeklyStart))
	require.True(t, repo.resetMonthlyStart.Equal(monthlyStart))
	require.Zero(t, sub.WeeklyUsageUSD)
	require.Zero(t, sub.MonthlyUsageUSD)
}

func TestCheckAndResetWindowsDoesNotMutateSnapshotWhenMonthlyResetFails(t *testing.T) {
	now := time.Now()
	anchor := now.Add(-45 * 24 * time.Hour)
	weeklyMarker := anchor
	monthlyMarker := anchor
	resetErr := errors.New("monthly reset failed")
	repo := &anchoredResetTrackingUserSubRepo{resetMonthlyErr: resetErr}
	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)
	sub := &UserSubscription{
		ID:                 3,
		UserID:             10,
		GroupID:            20,
		StartsAt:           anchor,
		ExpiresAt:          now.Add(30 * 24 * time.Hour),
		WeeklyWindowStart:  &weeklyMarker,
		MonthlyWindowStart: &monthlyMarker,
		WeeklyUsageUSD:     12,
		MonthlyUsageUSD:    34,
	}

	err := svc.CheckAndResetWindows(context.Background(), sub)

	require.ErrorIs(t, err, resetErr)
	require.True(t, repo.resetWeeklyCalled)
	require.True(t, repo.resetMonthlyCalled)
	require.True(t, sub.WeeklyWindowStart.Equal(weeklyMarker))
	require.True(t, sub.MonthlyWindowStart.Equal(monthlyMarker))
	require.Equal(t, 12.0, sub.WeeklyUsageUSD)
	require.Equal(t, 34.0, sub.MonthlyUsageUSD)
}
