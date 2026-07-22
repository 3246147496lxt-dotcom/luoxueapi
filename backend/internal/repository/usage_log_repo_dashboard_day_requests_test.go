package repository

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/stretchr/testify/require"
)

func TestNewDashboardDayComparisonWindow(t *testing.T) {
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
			name:          "ordinary wall clock window",
			loc:           shanghai,
			now:           time.Date(2026, 7, 21, 15, 4, 5, 600, shanghai),
			currentStart:  time.Date(2026, 7, 21, 0, 0, 0, 0, shanghai),
			previousStart: time.Date(2026, 7, 20, 0, 0, 0, 0, shanghai),
			previousEnd:   time.Date(2026, 7, 20, 15, 4, 5, 600, shanghai),
		},
		{
			name:          "midnight produces two empty half-open windows",
			loc:           shanghai,
			now:           time.Date(2026, 7, 21, 0, 0, 0, 0, shanghai),
			currentStart:  time.Date(2026, 7, 21, 0, 0, 0, 0, shanghai),
			previousStart: time.Date(2026, 7, 20, 0, 0, 0, 0, shanghai),
			previousEnd:   time.Date(2026, 7, 20, 0, 0, 0, 0, shanghai),
		},
		{
			name:          "calendar subtraction crosses the year boundary",
			loc:           shanghai,
			now:           time.Date(2027, 1, 1, 15, 4, 5, 0, shanghai),
			currentStart:  time.Date(2027, 1, 1, 0, 0, 0, 0, shanghai),
			previousStart: time.Date(2026, 12, 31, 0, 0, 0, 0, shanghai),
			previousEnd:   time.Date(2026, 12, 31, 15, 4, 5, 0, shanghai),
		},
		{
			name:          "spring DST keeps the same prior-day wall clock",
			loc:           newYork,
			now:           time.Date(2026, 3, 8, 12, 30, 0, 0, newYork),
			currentStart:  time.Date(2026, 3, 8, 0, 0, 0, 0, newYork),
			previousStart: time.Date(2026, 3, 7, 0, 0, 0, 0, newYork),
			previousEnd:   time.Date(2026, 3, 7, 12, 30, 0, 0, newYork),
		},
		{
			name:          "fall DST keeps the same prior-day wall clock",
			loc:           newYork,
			now:           time.Date(2026, 11, 1, 12, 30, 0, 0, newYork),
			currentStart:  time.Date(2026, 11, 1, 0, 0, 0, 0, newYork),
			previousStart: time.Date(2026, 10, 31, 0, 0, 0, 0, newYork),
			previousEnd:   time.Date(2026, 10, 31, 12, 30, 0, 0, newYork),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newDashboardDayComparisonWindow(tt.now, tt.loc)
			require.Equal(t, tt.currentStart, got.currentStart)
			require.Equal(t, tt.now, got.currentEnd)
			require.Equal(t, tt.previousStart, got.previousStart)
			require.Equal(t, tt.previousEnd, got.previousEnd)
		})
	}
}

func TestFillDashboardDayRequestComparison_UsesOneBoundedCountQuery(t *testing.T) {
	require.Equal(t, 2, strings.Count(dashboardDayRequestComparisonQuery, "COUNT(*) FILTER"))
	require.Contains(t, dashboardDayRequestComparisonQuery, "OR (created_at >= $3 AND created_at < $4)")
	require.NotContains(t, dashboardDayRequestComparisonQuery, "actual_cost", "today_requests counts every request row")

	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}
	loc := timezone.Location()
	now := time.Date(2026, 7, 21, 15, 4, 5, 0, loc)
	windows := newDashboardDayComparisonWindow(now, loc)

	mock.ExpectQuery(regexp.QuoteMeta(dashboardDayRequestComparisonQuery)).
		WithArgs(windows.currentStart, windows.currentEnd, windows.previousStart, windows.previousEnd).
		WillReturnRows(sqlmock.NewRows([]string{
			"current_day_requests",
			"previous_day_same_period_requests",
		}).AddRow(int64(182460), int64(174102)))

	stats := &DashboardStats{TodayRequests: 999}
	require.NoError(t, repo.fillDashboardDayRequestComparison(context.Background(), stats, now))
	require.Equal(t, int64(182460), stats.CurrentDayRequests)
	require.Equal(t, stats.CurrentDayRequests, stats.TodayRequests, "main value must share the real-time comparison numerator")
	require.Equal(t, int64(174102), stats.PreviousDaySamePeriodRequests)
	require.Equal(t, windows.currentStart.Format(time.RFC3339Nano), stats.CurrentDayStartAt)
	require.Equal(t, windows.currentEnd.Format(time.RFC3339Nano), stats.CurrentDayEndAt)
	require.Equal(t, windows.previousStart.Format(time.RFC3339Nano), stats.PreviousDaySamePeriodStartAt)
	require.Equal(t, windows.previousEnd.Format(time.RFC3339Nano), stats.PreviousDaySamePeriodEndAt)
	require.Equal(t, timezone.Name(), stats.StatsTimezone)
	require.NoError(t, mock.ExpectationsWereMet())
}
