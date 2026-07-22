package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestNewDashboardActiveAPIKeyWeekWindow(t *testing.T) {
	shanghai, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	newYork, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	tests := []struct {
		name          string
		loc           *time.Location
		now           time.Time
		currentStart  time.Time
		previousStart time.Time
		previousEnd   time.Time
	}{
		{
			name:          "midweek in configured timezone",
			loc:           shanghai,
			now:           time.Date(2026, 7, 21, 15, 4, 5, 600, shanghai),
			currentStart:  time.Date(2026, 7, 20, 0, 0, 0, 0, shanghai),
			previousStart: time.Date(2026, 7, 13, 0, 0, 0, 0, shanghai),
			previousEnd:   time.Date(2026, 7, 14, 15, 4, 5, 600, shanghai),
		},
		{
			name:          "monday boundary creates two empty same-period windows",
			loc:           shanghai,
			now:           time.Date(2026, 7, 20, 0, 0, 0, 0, shanghai),
			currentStart:  time.Date(2026, 7, 20, 0, 0, 0, 0, shanghai),
			previousStart: time.Date(2026, 7, 13, 0, 0, 0, 0, shanghai),
			previousEnd:   time.Date(2026, 7, 13, 0, 0, 0, 0, shanghai),
		},
		{
			name:          "sunday end remains in the monday-based natural week",
			loc:           shanghai,
			now:           time.Date(2026, 7, 26, 23, 59, 59, 0, shanghai),
			currentStart:  time.Date(2026, 7, 20, 0, 0, 0, 0, shanghai),
			previousStart: time.Date(2026, 7, 13, 0, 0, 0, 0, shanghai),
			previousEnd:   time.Date(2026, 7, 19, 23, 59, 59, 0, shanghai),
		},
		{
			name:          "calendar subtraction preserves wall clock across DST",
			loc:           newYork,
			now:           time.Date(2026, 3, 9, 12, 30, 0, 0, newYork),
			currentStart:  time.Date(2026, 3, 9, 0, 0, 0, 0, newYork),
			previousStart: time.Date(2026, 3, 2, 0, 0, 0, 0, newYork),
			previousEnd:   time.Date(2026, 3, 2, 12, 30, 0, 0, newYork),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newDashboardActiveAPIKeyWeekWindow(tt.now, tt.loc)
			require.Equal(t, tt.currentStart, got.currentStart)
			require.Equal(t, tt.now, got.currentEnd)
			require.Equal(t, tt.previousStart, got.previousStart)
			require.Equal(t, tt.previousEnd, got.previousEnd)
		})
	}
}

func TestFillDashboardActiveAPIKeyWeekStats_UsesOneBoundedDistinctQuery(t *testing.T) {
	require.Contains(t, dashboardActiveAPIKeyWeekStatsQuery, "COUNT(DISTINCT ul.api_key_id)")
	require.Contains(t, dashboardActiveAPIKeyWeekStatsQuery, "ul.actual_cost > 0")
	require.NotContains(t, dashboardActiveAPIKeyWeekStatsQuery, "JOIN api_keys")
	require.NotContains(t, dashboardActiveAPIKeyWeekStatsQuery, "ak.status")
	require.NotContains(t, dashboardActiveAPIKeyWeekStatsQuery, "ak.deleted_at")
	require.Contains(t, dashboardActiveAPIKeyWeekStatsQuery, "OR (ul.created_at >= $3 AND ul.created_at < $4)")

	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}
	loc := timezone.Location()
	now := time.Date(2026, 7, 21, 15, 4, 5, 0, loc)
	windows := newDashboardActiveAPIKeyWeekWindow(now, loc)

	mock.ExpectQuery(regexp.QuoteMeta(dashboardActiveAPIKeyWeekStatsQuery)).
		WithArgs(
			windows.currentStart,
			windows.currentEnd,
			windows.previousStart,
			windows.previousEnd,
		).
		WillReturnRows(sqlmock.NewRows([]string{
			"current_week_active_api_keys",
			"previous_week_same_period_active_api_keys",
		}).AddRow(int64(17), int64(11)))

	stats := &DashboardStats{}
	require.NoError(t, repo.fillDashboardActiveAPIKeyWeekStats(context.Background(), stats, now))
	require.Equal(t, int64(17), stats.CurrentWeekActiveAPIKeys)
	require.Equal(t, int64(11), stats.PreviousWeekSamePeriodActiveAPIKeys)
	require.Equal(t, windows.currentStart.Format(time.RFC3339Nano), stats.CurrentWeekStartAt)
	require.Equal(t, windows.currentEnd.Format(time.RFC3339Nano), stats.CurrentWeekEndAt)
	require.Equal(t, windows.previousStart.Format(time.RFC3339Nano), stats.PreviousWeekSamePeriodStartAt)
	require.Equal(t, windows.previousEnd.Format(time.RFC3339Nano), stats.PreviousWeekSamePeriodEndAt)
	require.Equal(t, timezone.Name(), stats.StatsTimezone)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFillDashboardEntityStats_HealthyAccountsExcludeTransientlyUnavailable(t *testing.T) {
	require.Contains(t, dashboardAccountStatsQuery, "status = $1")
	require.Contains(t, dashboardAccountStatsQuery, "schedulable = true")
	require.Contains(t, dashboardAccountStatsQuery, "rate_limit_reset_at IS NULL OR rate_limit_reset_at <= $3")
	require.Contains(t, dashboardAccountStatsQuery, "overload_until IS NULL OR overload_until <= $3")
	require.Contains(t, dashboardAccountStatsQuery, "temp_unschedulable_until IS NULL OR temp_unschedulable_until <= $3")
	require.Contains(t, dashboardAccountStatsQuery, "auto_pause_on_expired = false OR expires_at IS NULL OR expires_at > $3")
	require.NotContains(t, dashboardAccountStatsQuery, "extra ->", "quota windows in extra JSON must not be approximated")

	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}
	now := time.Date(2026, 7, 21, 15, 4, 5, 0, time.UTC)

	windows := newDashboardDayComparisonWindow(now, timezone.Location())
	mock.ExpectQuery(regexp.QuoteMeta(dashboardUserStatsQuery)).
		WithArgs(windows.currentStart, windows.currentEnd, windows.previousStart, windows.previousEnd).
		WillReturnRows(sqlmock.NewRows([]string{
			"total_users",
			"current_day_new_users",
			"previous_day_same_period_new_users",
		}).AddRow(int64(0), int64(0), int64(0)))
	mock.ExpectQuery("(?s)SELECT.*total_api_keys.*FROM api_keys").
		WithArgs(service.StatusActive).
		WillReturnRows(sqlmock.NewRows([]string{"total_api_keys", "active_api_keys"}).AddRow(int64(0), int64(0)))
	mock.ExpectQuery(regexp.QuoteMeta(dashboardAccountStatsQuery)).
		WithArgs(service.StatusActive, service.StatusError, now, now).
		WillReturnRows(sqlmock.NewRows([]string{
			"total_accounts",
			"normal_accounts",
			"healthy_accounts",
			"error_accounts",
			"ratelimit_accounts",
			"overload_accounts",
		}).AddRow(int64(6), int64(4), int64(1), int64(1), int64(1), int64(1)))

	stats := &DashboardStats{}
	require.NoError(t, repo.fillDashboardEntityStats(context.Background(), stats, now))
	require.Equal(t, int64(4), stats.NormalAccounts, "legacy active+schedulable count must remain unchanged")
	require.Equal(t, int64(1), stats.HealthyAccounts, "rate-limited, overloaded and temporarily cooled-down accounts must not be healthy")
	require.NoError(t, mock.ExpectationsWereMet())
}

func expectDashboardEntityAndActiveKeyWeekQueries(mock sqlmock.Sqlmock, current, previous int64) {
	mock.ExpectQuery(regexp.QuoteMeta(dashboardUserStatsQuery)).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"total_users",
			"current_day_new_users",
			"previous_day_same_period_new_users",
		}).AddRow(int64(0), int64(0), int64(0)))
	mock.ExpectQuery("(?s)SELECT.*total_api_keys.*FROM api_keys").
		WithArgs(service.StatusActive).
		WillReturnRows(sqlmock.NewRows([]string{"total_api_keys", "active_api_keys"}).AddRow(int64(0), int64(0)))
	mock.ExpectQuery("(?s)SELECT.*total_accounts.*FROM accounts").
		WithArgs(service.StatusActive, service.StatusError, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"total_accounts",
			"normal_accounts",
			"healthy_accounts",
			"error_accounts",
			"ratelimit_accounts",
			"overload_accounts",
		}).AddRow(int64(0), int64(0), int64(0), int64(0), int64(0), int64(0)))
	mock.ExpectQuery(regexp.QuoteMeta(dashboardActiveAPIKeyWeekStatsQuery)).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"current_week_active_api_keys",
			"previous_week_same_period_active_api_keys",
		}).AddRow(current, previous))
}

func expectDashboardPerformanceQuery(mock sqlmock.Sqlmock) {
	mock.ExpectQuery("(?s)SELECT.*request_count.*token_count.*FROM usage_logs").
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"request_count", "token_count"}).AddRow(int64(0), int64(0)))
}

func expectDashboardDayRequestComparison(mock sqlmock.Sqlmock, current, previous int64) {
	mock.ExpectQuery(regexp.QuoteMeta(dashboardDayRequestComparisonQuery)).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"current_day_requests",
			"previous_day_same_period_requests",
		}).AddRow(current, previous))
}

func TestGetDashboardStats_PopulatesActiveAPIKeyWeekFields(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}
	expectDashboardEntityAndActiveKeyWeekQueries(mock, 23, 19)
	mock.ExpectQuery("(?s)SELECT.*SUM\\(total_requests\\).*FROM usage_dashboard_daily").
		WillReturnRows(sqlmock.NewRows([]string{
			"total_requests",
			"total_input_tokens",
			"total_output_tokens",
			"total_cache_creation_tokens",
			"total_cache_read_tokens",
			"total_cost",
			"total_actual_cost",
			"total_account_cost",
			"total_duration_ms",
		}).AddRow(int64(0), int64(0), int64(0), int64(0), int64(0), 0, 0, 0, int64(0)))
	mock.ExpectQuery("(?s)SELECT.*today_requests.*FROM usage_dashboard_daily").
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"today_requests",
			"today_input_tokens",
			"today_output_tokens",
			"today_cache_creation_tokens",
			"today_cache_read_tokens",
			"today_cost",
			"today_actual_cost",
			"today_account_cost",
			"active_users",
		}).AddRow(int64(0), int64(0), int64(0), int64(0), int64(0), 0, 0, 0, int64(0)))
	mock.ExpectQuery("(?s)SELECT active_users.*FROM usage_dashboard_hourly").
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"active_users"}).AddRow(int64(0)))
	expectDashboardDayRequestComparison(mock, 31, 27)
	expectDashboardPerformanceQuery(mock)

	stats, err := repo.GetDashboardStats(context.Background())
	require.NoError(t, err)
	require.Equal(t, int64(23), stats.CurrentWeekActiveAPIKeys)
	require.Equal(t, int64(19), stats.PreviousWeekSamePeriodActiveAPIKeys)
	require.NotEmpty(t, stats.CurrentWeekStartAt)
	require.NotEmpty(t, stats.PreviousWeekSamePeriodEndAt)
	require.Equal(t, int64(31), stats.TodayRequests)
	require.Equal(t, int64(27), stats.PreviousDaySamePeriodRequests)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetDashboardStatsWithRange_PopulatesActiveAPIKeyWeekFields(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}
	expectDashboardEntityAndActiveKeyWeekQueries(mock, 29, 13)
	mock.ExpectQuery("(?s)WITH scoped AS.*total_input_tokens.*total_account_cost.*FROM scoped").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"total_requests",
			"total_input_tokens",
			"total_output_tokens",
			"total_cache_creation_tokens",
			"total_cache_read_tokens",
			"total_cost",
			"total_actual_cost",
			"total_account_cost",
			"total_duration_ms",
			"today_requests",
			"today_input_tokens",
			"today_output_tokens",
			"today_cache_creation_tokens",
			"today_cache_read_tokens",
			"today_cost",
			"today_actual_cost",
			"today_account_cost",
		}).AddRow(
			int64(0), int64(0), int64(0), int64(0), int64(0), 0, 0, 0, int64(0),
			int64(0), int64(0), int64(0), int64(0), int64(0), 0, 0, 0,
		))
	mock.ExpectQuery("(?s)WITH scoped AS.*SELECT user_id, created_at.*hourly_active_users").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"active_users", "hourly_active_users"}).AddRow(int64(0), int64(0)))
	expectDashboardDayRequestComparison(mock, 37, 21)
	expectDashboardPerformanceQuery(mock)

	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	stats, err := repo.GetDashboardStatsWithRange(context.Background(), start, end)
	require.NoError(t, err)
	require.Equal(t, int64(29), stats.CurrentWeekActiveAPIKeys)
	require.Equal(t, int64(13), stats.PreviousWeekSamePeriodActiveAPIKeys)
	require.NotEmpty(t, stats.CurrentWeekStartAt)
	require.NotEmpty(t, stats.PreviousWeekSamePeriodEndAt)
	require.Equal(t, int64(37), stats.TodayRequests)
	require.Equal(t, int64(21), stats.PreviousDaySamePeriodRequests)
	require.NoError(t, mock.ExpectationsWereMet())
}
