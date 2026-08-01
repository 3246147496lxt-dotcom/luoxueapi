package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSubscriptionAnchoredMonthlyQuotaMigrationLocksBeforeAggregation(t *testing.T) {
	content, err := FS.ReadFile("195_subscription_anchored_monthly_quota.sql")
	require.NoError(t, err)

	upperSQL := strings.ToUpper(strings.Join(strings.Fields(string(content)), " "))
	lock := "LOCK TABLE USER_SUBSCRIPTIONS IN SHARE ROW EXCLUSIVE MODE"
	require.Contains(t, upperSQL, lock)
	require.Less(t, strings.Index(upperSQL, lock), strings.Index(upperSQL, "ALTER TABLE BILLING_USAGE_ENTRIES"))
	require.Less(t, strings.Index(upperSQL, lock), strings.Index(upperSQL, "WITH ANCHORED AS"))
	require.Contains(t, upperSQL, "DRAIN OLD/DEGRADED WRITERS")
	require.Contains(t, upperSQL, "SAME TRANSACTION")
}

func TestSubscriptionAnchoredMonthlyQuotaMigrationUsesReceiptLedgerWithDeduplicatedLegacyFallback(t *testing.T) {
	content, err := FS.ReadFile("195_subscription_anchored_monthly_quota.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	upperSQL := strings.ToUpper(sql)

	require.Contains(t, sql, "weekly_window_start = a.weekly_period_start")
	require.Contains(t, sql, "WHEN us.weekly_window_start = a.weekly_period_start THEN us.weekly_usage_usd")
	require.Contains(t, sql, "ELSE au.weekly_used")
	require.Contains(t, sql, "monthly_window_start = a.monthly_period_start")
	require.Contains(t, sql, "monthly_usage_usd = au.monthly_used")
	require.Contains(t, sql, "FROM billing_usage_entries bue")
	require.Contains(t, sql, "bue.applied")
	require.Contains(t, sql, "bue.status = 'subscription'")
	require.Contains(t, sql, "bue.subscription_amount")
	require.Contains(t, sql, "matched_log.actual_cost")
	require.Contains(t, sql, "COALESCE(bue.subscription_amount, matched_log.actual_cost)")
	require.NotContains(t, sql, "bue.gross_amount")
	require.NotContains(t, sql, "bue.charged_amount")
	require.Contains(t, sql, "FROM usage_logs ul")
	require.Equal(t, 3, strings.Count(sql, "ul.billing_type = 1"),
		"receipt matching and legacy replay must accept only membership-billed usage logs")
	require.Contains(t, upperSQL, "NOT EXISTS")
	require.Contains(t, sql, "bue.usage_log_id = ul.id")
	require.Contains(t, sql, "ul.request_id = bue.request_id")
	require.Contains(t, sql, "bue.api_key_id = ul.api_key_id")
	require.Contains(t, upperSQL, "UNION ALL")
	require.Contains(t, sql, "604800 * INTERVAL '1 second'")
	require.Contains(t, sql, "2592000 * INTERVAL '1 second'")
	require.NotContains(t, upperSQL, "INTERVAL '7 DAYS'")
	require.NotContains(t, upperSQL, "INTERVAL '30 DAYS'")
	require.NotContains(t, sql, "WHEN us.monthly_window_start = a.monthly_period_start")
	require.NotContains(t, sql, "GREATEST(us.monthly_usage_usd, au.monthly_used)")
	require.NotContains(t, sql, "ALTER COLUMN monthly_window_start SET NOT NULL")
	require.Contains(t, sql, "user_subscriptions_weekly_window_anchored_check")
	require.Contains(t, sql, "user_subscriptions_monthly_window_anchored_check")
	require.Contains(t, sql, "weekly_window_start IS NULL OR")
	require.Contains(t, sql, "monthly_window_start IS NULL OR")
	require.Contains(t, upperSQL, "NOT VALID")
	require.Contains(t, upperSQL, "VALIDATE CONSTRAINT")
	require.Contains(t, sql, "604800")
	require.Contains(t, sql, "2592000")
}

func TestSubscriptionAnchoredMonthlyQuotaMigrationFailsClosedOnUnreconstructableReceipts(t *testing.T) {
	content, err := FS.ReadFile("195_subscription_anchored_monthly_quota.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	upperSQL := strings.ToUpper(sql)

	require.Contains(t, upperSQL, "ADD COLUMN IF NOT EXISTS SUBSCRIPTION_AMOUNT DECIMAL(20, 10)")
	require.NotContains(t, upperSQL, "ALTER COLUMN SUBSCRIPTION_AMOUNT SET NOT NULL")
	require.Contains(t, sql, "bue.subscription_amount IS NULL")
	require.Contains(t, sql, "matched_log.actual_cost IS NULL")
	require.Contains(t, upperSQL, "RAISE EXCEPTION")
	require.Contains(t, upperSQL, "DRAIN BILLING TRAFFIC")
	require.Contains(t, sql, "billing_usage_entries_subscription_amount_check")
}

func TestSubscriptionAnchoredMonthlyQuotaMigrationPreservesPlanConfiguration(t *testing.T) {
	content, err := FS.ReadFile("195_subscription_anchored_monthly_quota.sql")
	require.NoError(t, err)

	upperSQL := strings.ToUpper(strings.Join(strings.Fields(string(content)), " "))
	require.NotContains(t, upperSQL, "UPDATE GROUPS")
	require.NotContains(t, upperSQL, "MONTHLY_LIMIT_USD =")
	require.NotContains(t, upperSQL, "WEEKLY_LIMIT_USD =")
}
