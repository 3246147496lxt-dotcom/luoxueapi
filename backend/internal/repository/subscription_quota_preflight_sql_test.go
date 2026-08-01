package repository

import (
	"os"
	"path/filepath"
	"strings"
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

func TestMigration195PreflightSQLChecksBothReceiptDirectionsAndAllDefiniteOldTerms(t *testing.T) {
	content, err := os.ReadFile(filepath.Join(
		"..",
		"..",
		"scripts",
		"sql",
		"preflight_subscription_anchored_monthly_quota.sql",
	))
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "window_logs AS MATERIALIZED")
	require.Contains(t, sql, "all_receipt_count <> 1")
	require.Contains(t, sql, "valid_receipt_count <> 1")
	require.Contains(t, sql, "usage_log_receipt_identity_conflicts")
	require.Contains(t, sql, "receipt and usage-log identities must agree bidirectionally")
	require.Contains(t, sql, "ul.subscription_id IS DISTINCT FROM rr.subscription_id")
	require.Contains(t, sql, "ul.user_id IS DISTINCT FROM rr.subscription_user_id")
	require.Contains(t, sql, "ul.api_key_id IS DISTINCT FROM rr.api_key_id")
	require.Contains(t, sql, "ul.id IS NOT DISTINCT FROM rr.usage_log_id")
	require.Contains(t, sql, "ul.request_id IS DISTINCT FROM rr.request_id")
	require.Contains(t, sql, "rr.request_id IS NULL OR ul.request_id IS NULL OR ul.request_id = rr.request_id")
	require.Contains(t, sql, "receipt_identity_conflicts")
	require.Contains(t, sql, "receipt_api_key_user_id IS NOT DISTINCT FROM rr.receipt_user_id")
	require.Contains(t, sql, "log_key.user_id IS NOT DISTINCT FROM ul.user_id")
	require.Contains(t, sql, "FROM billing_usage_entries bue WHERE bue.usage_log_id = ul.id OR")
	require.Contains(t, sql, "actual_cost = 'NaN'::numeric")
	require.Contains(t, sql, "subscription_amount = 'NaN'::numeric")
	require.Contains(t, sql, "weekly_usage_usd = 'NaN'::numeric")
	require.Contains(t, sql, "monthly_usage_usd = 'NaN'::numeric")
	require.Contains(t, sql, "g.weekly_limit_usd = 'NaN'::numeric")
	require.Contains(t, sql, "g.monthly_limit_usd = 'NaN'::numeric")
	require.Contains(t, sql, "subscription_counter_values")
	require.Contains(t, sql, "groups_subscription_quota_limits_check")
	require.Contains(t, sql, "subscription_plan_limit_values")
	require.Contains(t, sql, "actual_cost = 'Infinity'::numeric")

	definiteStart := strings.Index(sql, "definite_term_risk AS")
	ambiguousStart := strings.Index(sql, "ambiguous_term_risk AS")
	require.NotEqual(t, -1, definiteStart)
	require.Greater(t, ambiguousStart, definiteStart)
	definiteSQL := sql[definiteStart:ambiguousStart]
	require.Contains(t, definiteSQL, "rc.matched_created_at < rc.starts_at")
	require.NotContains(t, definiteSQL, "term_guard_hours")
	require.NotContains(t, definiteSQL, "legacy_writers_stopped_at")
}
