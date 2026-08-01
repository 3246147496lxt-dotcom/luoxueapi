package repository

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigration195PreflightSQLFailsClosedOnPreparedTransactions(t *testing.T) {
	content, err := os.ReadFile(filepath.Join(
		"..",
		"..",
		"scripts",
		"sql",
		"preflight_subscription_anchored_monthly_quota.sql",
	))
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "FROM pg_prepared_xacts")
	require.Contains(t, sql, "WHERE database = current_database()")
	require.Contains(t, sql, "'database_prepared_transactions'")
	require.Contains(t, sql, "WHEN p.preflight_mode = 'maintenance' THEN 'BLOCK'")
}
