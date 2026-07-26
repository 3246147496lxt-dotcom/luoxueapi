package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestGetAPIKeyUsageTrendExcludesWebChatPrincipals(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	queryPattern := `(?s)WITH top_keys AS .*INNER JOIN api_keys k ON u\.api_key_id = k\.id AND k\.purpose = 'user'.*FROM usage_logs u.*INNER JOIN api_keys k ON u\.api_key_id = k\.id AND k\.purpose = 'user'`
	mock.ExpectQuery(queryPattern).
		WithArgs(start, end, 10, start, end).
		WillReturnRows(sqlmock.NewRows([]string{"date", "api_key_id", "key_name", "requests", "tokens"}).
			AddRow("2026-07-01", int64(7), "User Key", int64(2), int64(30)))

	rows, err := repo.GetAPIKeyUsageTrend(context.Background(), start, end, "day", 10)

	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, int64(7), rows[0].APIKeyID)
	require.Equal(t, "User Key", rows[0].KeyName)
	require.NoError(t, mock.ExpectationsWereMet())
}
