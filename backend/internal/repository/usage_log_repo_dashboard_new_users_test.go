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

func TestDashboardUserStatsQuery_PreservesLegacyUserSemanticsWithBoundedWindows(t *testing.T) {
	require.Contains(t, dashboardUserStatsQuery, "COUNT(*) as total_users")
	require.Equal(t, 2, strings.Count(dashboardUserStatsQuery, "COUNT(*) FILTER"))
	require.Contains(t, dashboardUserStatsQuery, "created_at >= $1 AND created_at < $2")
	require.Contains(t, dashboardUserStatsQuery, "created_at >= $3 AND created_at < $4")
	require.Contains(t, dashboardUserStatsQuery, "WHERE deleted_at IS NULL", "soft-deleted users must remain excluded")
	require.NotContains(t, dashboardUserStatsQuery, "created_at <=", "comparison windows must remain half-open")
}

func TestFillDashboardUserStats_UsesOneQueryAndKeepsMainValueInSync(t *testing.T) {
	tests := []struct {
		name     string
		total    int64
		current  int64
		previous int64
	}{
		{name: "ordinary comparison", total: 8642, current: 126, previous: 116},
		{name: "both periods have zero new users", total: 8642, current: 0, previous: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newSQLMock(t)
			repo := &usageLogRepository{sql: db}
			loc := timezone.Location()
			now := time.Date(2026, 7, 21, 15, 4, 5, 0, loc)
			windows := newDashboardDayComparisonWindow(now, loc)

			mock.ExpectQuery(regexp.QuoteMeta(dashboardUserStatsQuery)).
				WithArgs(windows.currentStart, windows.currentEnd, windows.previousStart, windows.previousEnd).
				WillReturnRows(sqlmock.NewRows([]string{
					"total_users",
					"current_day_new_users",
					"previous_day_same_period_new_users",
				}).AddRow(tt.total, tt.current, tt.previous))

			stats := &DashboardStats{TodayNewUsers: 999}
			require.NoError(t, repo.fillDashboardUserStats(context.Background(), stats, now))
			require.Equal(t, tt.total, stats.TotalUsers)
			require.Equal(t, tt.current, stats.CurrentDayNewUsers)
			require.Equal(t, stats.CurrentDayNewUsers, stats.TodayNewUsers)
			require.Equal(t, tt.previous, stats.PreviousDaySamePeriodNewUsers)
			require.NoError(t, mock.ExpectationsWereMet(), "the user aggregates must use exactly one SQL query")
		})
	}
}
