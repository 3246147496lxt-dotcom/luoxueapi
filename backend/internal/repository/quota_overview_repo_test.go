//go:build unit

package repository

import (
	"context"
	"errors"
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestQuotaOverviewRepositoryUsesUserScopedMinimalExactProjection(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	asOf := time.Date(2026, 7, 29, 7, 30, 0, 123456000, time.UTC)
	anchor := time.Date(2026, 7, 25, 9, 30, 0, 0, time.UTC)
	expiry := anchor.Add(31 * 24 * time.Hour)
	updatedAt := asOf.Add(-time.Second)
	userID := int64(42)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(quotaOverviewAccountQuery)).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{
			"as_of", "id", "email", "status", "balance", "frozen_balance",
		}).AddRow(
			asOf, userID, "pu@example.com", service.StatusActive,
			"1234567890.12345678", "0.00000001",
		))
	mock.ExpectQuery(regexp.QuoteMeta(quotaOverviewSpendQuery)).
		WithArgs(
			userID,
			time.Date(2026, 7, 28, 16, 0, 0, 0, time.UTC),
			time.Date(2026, 6, 30, 16, 0, 0, 0, time.UTC),
			asOf,
		).
		WillReturnRows(sqlmock.NewRows([]string{
			"today_spend", "month_spend",
		}).AddRow("1.1600000000", "10.7400000000"))
	mock.ExpectQuery(regexp.QuoteMeta(quotaOverviewKeysQuery)).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "masked_key", "status", "group_id", "quota", "quota_used",
			"expires_at", "joined_group_id", "group_name", "group_status",
			"subscription_type", "group_deleted",
		}).AddRow(
			int64(9007199254740993), "Codex", "sk-••••42FD", service.StatusAPIKeyActive,
			int64(20), "200.00000000", "12.34567890", nil,
			int64(20), "Pro 会员", service.StatusActive,
			service.SubscriptionTypeSubscription, false,
		))
	mock.ExpectQuery(regexp.QuoteMeta(quotaOverviewSubscriptionsQuery)).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "group_id", "group_name", "group_status", "group_deleted", "status",
			"revoked", "starts_at", "expires_at", "weekly_window_start",
			"weekly_limit", "weekly_used", "updated_at",
		}).AddRow(
			int64(8), int64(20), "Pro 会员", service.StatusActive, false,
			service.SubscriptionStatusActive, false, anchor, expiry, anchor,
			"200.00000000", "12.3456789012", updatedAt,
		))
	mock.ExpectExec(regexp.QuoteMeta(quotaOverviewPeriodUsageSavepointCreate)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta(quotaOverviewPeriodUsageQuery)).
		WithArgs(userID, int64(8), anchor, asOf).
		WillReturnRows(sqlmock.NewRows([]string{
			"bucket_index", "requests", "cache_hit_tokens", "cache_miss_tokens", "output_tokens",
		}).
			AddRow(0, int64(58), int64(92000), int64(41000), int64(49000)).
			AddRow(3, int64(82), int64(128000), int64(61000), int64(70000)))
	mock.ExpectExec(regexp.QuoteMeta(quotaOverviewPeriodUsageSavepointRelease)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	repo := NewQuotaOverviewRepository(db)
	snapshot, err := repo.LoadQuotaOverviewSnapshot(context.Background(), userID, "Asia/Shanghai")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
	require.Equal(t, asOf, snapshot.AsOf)
	require.Equal(t, "1234567890.12345678", snapshot.Account.Balance.String())
	require.Equal(t, "0.00000001", snapshot.Account.FrozenBalance.String())
	require.Equal(t, "1.16", snapshot.TodaySpend.String())
	require.Equal(t, "10.74", snapshot.MonthSpend.String())
	require.Len(t, snapshot.Keys, 1)
	require.Equal(t, int64(9007199254740993), snapshot.Keys[0].ID)
	require.Equal(t, "sk-••••42FD", snapshot.Keys[0].MaskedKey)
	require.Equal(t, "12.3456789", snapshot.Keys[0].QuotaUsed.String())
	require.Len(t, snapshot.Subscriptions, 1)
	require.Equal(t, "12.3456789012", snapshot.Subscriptions[0].WeeklyUsed.String())
	require.False(t, snapshot.PeriodUsageAggregationError)
	require.NotNil(t, snapshot.Subscriptions[0].PeriodUsage)
	require.Equal(t, anchor, snapshot.Subscriptions[0].PeriodUsage.PeriodStart)
	require.Equal(t, asOf, snapshot.Subscriptions[0].PeriodUsage.ObservedUntil)
	require.Equal(t, int64(58), snapshot.Subscriptions[0].PeriodUsage.Buckets[0].Requests)
	require.Equal(t, int64(41000), snapshot.Subscriptions[0].PeriodUsage.Buckets[0].CacheMissTokens)
	require.Equal(t, int64(82), snapshot.Subscriptions[0].PeriodUsage.Buckets[3].Requests)
	require.Zero(t, snapshot.Subscriptions[0].PeriodUsage.Buckets[1].Requests)

	keyType := reflect.TypeOf(service.QuotaOverviewKeySnapshot{})
	_, hasKey := keyType.FieldByName("Key")
	_, hasRawKey := keyType.FieldByName("RawKey")
	require.False(t, hasKey)
	require.False(t, hasRawKey)
	require.NotContains(t, quotaOverviewKeysQuery, "SELECT k.key")
	require.NotContains(t, quotaOverviewKeysQuery, "last_used_ip")
	require.NotContains(t, quotaOverviewKeysQuery, "ip_whitelist")
	require.Contains(t, quotaOverviewPeriodUsageQuery, "ul.user_id = $1")
	require.Contains(t, quotaOverviewPeriodUsageQuery, "ul.subscription_id = $2")
	require.Contains(t, quotaOverviewPeriodUsageQuery, "ul.created_at >= $3")
	require.Contains(t, quotaOverviewPeriodUsageQuery, "ul.created_at < $4")
	require.Contains(t, quotaOverviewPeriodUsageQuery, usageLogSuccessFilterUL)
	require.NotContains(t, quotaOverviewPeriodUsageQuery, "image_input_tokens")
	require.NotContains(t, quotaOverviewPeriodUsageQuery, "image_output_tokens")
}

func TestQuotaOverviewRepositoryPeriodUsageFailureDegradesOnlyOptionalUsage(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	asOf := time.Date(2026, 7, 29, 7, 30, 0, 0, time.UTC)
	anchor := time.Date(2026, 7, 25, 9, 30, 0, 0, time.UTC)
	expiry := anchor.Add(31 * 24 * time.Hour)
	userID := int64(42)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(quotaOverviewAccountQuery)).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{
			"as_of", "id", "email", "status", "balance", "frozen_balance",
		}).AddRow(
			asOf, userID, "pu@example.com", service.StatusActive, "128.64", "0",
		))
	mock.ExpectQuery(regexp.QuoteMeta(quotaOverviewSpendQuery)).
		WithArgs(
			userID,
			time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
			asOf,
		).
		WillReturnRows(sqlmock.NewRows([]string{"today_spend", "month_spend"}).
			AddRow("1.16", "10.74"))
	mock.ExpectQuery(regexp.QuoteMeta(quotaOverviewKeysQuery)).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "masked_key", "status", "group_id", "quota", "quota_used",
			"expires_at", "joined_group_id", "group_name", "group_status",
			"subscription_type", "group_deleted",
		}))
	mock.ExpectQuery(regexp.QuoteMeta(quotaOverviewSubscriptionsQuery)).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "group_id", "group_name", "group_status", "group_deleted", "status",
			"revoked", "starts_at", "expires_at", "weekly_window_start",
			"weekly_limit", "weekly_used", "updated_at",
		}).AddRow(
			int64(8), int64(20), "Pro 会员", service.StatusActive, false,
			service.SubscriptionStatusActive, false, anchor, expiry, anchor,
			"200", "12.34", asOf.Add(-time.Second),
		))
	mock.ExpectExec(regexp.QuoteMeta(quotaOverviewPeriodUsageSavepointCreate)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta(quotaOverviewPeriodUsageQuery)).
		WithArgs(userID, int64(8), anchor, asOf).
		WillReturnError(errors.New("period usage unavailable"))
	mock.ExpectExec(regexp.QuoteMeta(quotaOverviewPeriodUsageSavepointRollback)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(quotaOverviewPeriodUsageSavepointRelease)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	repo := NewQuotaOverviewRepository(db)
	snapshot, err := repo.LoadQuotaOverviewSnapshot(context.Background(), userID, "UTC")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
	require.Equal(t, "128.64", snapshot.Account.Balance.String())
	require.Equal(t, "1.16", snapshot.TodaySpend.String())
	require.Equal(t, "10.74", snapshot.MonthSpend.String())
	require.True(t, snapshot.PeriodUsageAggregationError)
	require.Len(t, snapshot.Subscriptions, 1)
	require.Nil(t, snapshot.Subscriptions[0].PeriodUsage)
}

func TestQuotaOverviewSpendStartsUseRequestedCalendarTimezone(t *testing.T) {
	asOf := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	todayStart, monthStart, err := quotaOverviewSpendStarts(asOf, "America/New_York")
	require.NoError(t, err)
	require.Equal(t, time.Date(2026, 3, 8, 5, 0, 0, 0, time.UTC), todayStart)
	require.Equal(t, time.Date(2026, 3, 1, 5, 0, 0, 0, time.UTC), monthStart)

	_, _, err = quotaOverviewSpendStarts(asOf, "Not/AZone")
	require.Error(t, err)
}

func TestCanLoadQuotaOverviewPeriodUsageAllowsOnlyCurrentOrExactPastAnchoredWindow(t *testing.T) {
	anchor := time.Date(2026, 7, 25, 9, 30, 0, 0, time.UTC)
	asOf := anchor.Add(service.SubscriptionWeeklyWindowDuration + 3*24*time.Hour)
	limit := decimal.RequireFromString("200")
	currentStart, _, ok := service.AnchoredWeeklyWindow(anchor, asOf)
	require.True(t, ok)
	base := service.QuotaOverviewSubscriptionSnapshot{
		ID:                8,
		Status:            service.SubscriptionStatusActive,
		StartsAt:          anchor,
		ExpiresAt:         anchor.Add(31 * 24 * time.Hour),
		WeeklyWindowStart: &currentStart,
		WeeklyLimit:       &limit,
	}
	tests := []struct {
		name   string
		mutate func(*service.QuotaOverviewSubscriptionSnapshot)
		want   bool
	}{
		{name: "current authoritative window", want: true},
		{
			name: "exact past anchored window is loaded for empty-period proof",
			mutate: func(sub *service.QuotaOverviewSubscriptionSnapshot) {
				sub.WeeklyWindowStart = &anchor
			},
			want: true,
		},
		{
			name: "off-anchor past window",
			mutate: func(sub *service.QuotaOverviewSubscriptionSnapshot) {
				mismatch := anchor.Add(time.Second)
				sub.WeeklyWindowStart = &mismatch
			},
		},
		{
			name: "future window",
			mutate: func(sub *service.QuotaOverviewSubscriptionSnapshot) {
				future := currentStart.Add(service.SubscriptionWeeklyWindowDuration)
				sub.WeeklyWindowStart = &future
			},
		},
		{
			name: "revoked",
			mutate: func(sub *service.QuotaOverviewSubscriptionSnapshot) {
				sub.Revoked = true
			},
		},
		{
			name: "expired",
			mutate: func(sub *service.QuotaOverviewSubscriptionSnapshot) {
				sub.ExpiresAt = asOf
			},
		},
		{
			name: "suspended",
			mutate: func(sub *service.QuotaOverviewSubscriptionSnapshot) {
				sub.Status = service.SubscriptionStatusSuspended
			},
		},
		{
			name: "missing weekly limit",
			mutate: func(sub *service.QuotaOverviewSubscriptionSnapshot) {
				sub.WeeklyLimit = nil
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sub := base
			if test.mutate != nil {
				test.mutate(&sub)
			}
			require.Equal(t, test.want, canLoadQuotaOverviewPeriodUsage(sub, asOf))
		})
	}
}
