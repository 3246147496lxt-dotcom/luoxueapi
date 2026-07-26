package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestGetBatchAPIKeyUsageStatsExcludesWebChatPrincipals(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	queryPattern := `(?s)FILTER \(WHERE ul\.created_at >= \$2::timestamptz AND ul\.created_at < \$3::timestamptz\).*FILTER \(WHERE ul\.created_at >= \$4::timestamptz\).*FROM api_keys ak.*LEFT JOIN usage_logs ul.*ul\.api_key_id = ak\.id.*LEAST\(\$2::timestamptz, \$4::timestamptz\).*WHERE ak\.id = ANY\(\$1\).*ak\.purpose = 'user'.*GROUP BY ak\.id`
	mock.ExpectQuery(queryPattern).
		WithArgs(sqlmock.AnyArg(), start, end, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"api_key_id", "total_cost", "today_cost"}).
			AddRow(int64(7), float64(0), float64(0)))

	stats, err := repo.GetBatchAPIKeyUsageStats(context.Background(), []int64{7, 2}, start, end)

	require.NoError(t, err)
	require.Contains(t, stats, int64(7), "visible user keys retain a zero-valued row")
	require.NotContains(t, stats, int64(2), "internal web-chat principals must not be serialized")
	require.NoError(t, mock.ExpectationsWereMet())
}
