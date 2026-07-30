package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSubscriptionAnchoredWeeklyQuotaMigrationPreservesRollbackConfiguration(t *testing.T) {
	content, err := FS.ReadFile("192_subscription_anchored_weekly_quota.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	upperSQL := strings.ToUpper(sql)

	require.NotContains(t, upperSQL, "UPDATE GROUPS")
	require.NotContains(t, sql, "groups_subscription_weekly_only_check")
	require.NotContains(t, upperSQL, "DAILY_LIMIT_USD = NULL")
	require.NotContains(t, upperSQL, "MONTHLY_LIMIT_USD = NULL")
	require.NotContains(t, upperSQL, "DAILY_USAGE_USD = 0")
	require.NotContains(t, upperSQL, "MONTHLY_USAGE_USD = 0")
}

func TestSubscriptionAnchoredWeeklyQuotaMigrationRebuildsOnlyStaleWindow(t *testing.T) {
	content, err := FS.ReadFile("192_subscription_anchored_weekly_quota.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	upperSQL := strings.ToUpper(sql)

	require.Contains(t, sql, "ul.created_at >= a.period_start")
	require.Contains(t, sql, "ul.created_at < a.period_start + INTERVAL '7 days'")
	require.Contains(t, sql, "WHEN us.weekly_window_start = a.period_start THEN us.weekly_usage_usd ELSE lu.used END")
	require.NotContains(t, sql, "GREATEST(us.weekly_usage_usd, lu.used)")
	require.NotContains(t, sql, "ALTER COLUMN weekly_window_start SET NOT NULL")
	require.Contains(t, sql, "user_subscriptions_weekly_window_anchored_check")
	require.Contains(t, sql, "weekly_window_start IS NULL OR")
	require.Contains(t, upperSQL, "NOT VALID")
	require.Contains(t, upperSQL, "VALIDATE CONSTRAINT")
	require.Contains(t, sql, "604800")
}
