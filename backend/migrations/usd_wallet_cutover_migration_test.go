package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUSDWalletCutoverMigrationIsGuardedAndConvertsOnlyWalletUnits(t *testing.T) {
	content, err := FS.ReadFile("251_usd_wallet_cutover.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	lower := strings.ToLower(sql)

	require.Contains(t, sql, "billing_unit_migration_v1")
	require.Contains(t, sql, "schema_migration_runner_state")
	require.Contains(t, sql, "origin IS DISTINCT FROM 'legacy'")
	require.Contains(t, sql, "marker IN ('usd_div10', 'skipped_fresh')")
	require.Contains(t, sql, "ROUND(balance / 10, 8)")
	require.Contains(t, sql, "ROUND(value::NUMERIC / 10, 8)::TEXT")
	require.Contains(t, sql, "ROUND(amount / 10, 2)")
	require.Contains(t, sql, "WHERE order_type = 'balance'")
	require.Contains(t, sql, "WHERE type IN ('balance', 'admin_balance')")
	require.Contains(t, sql, "WHERE billing_type = 0")
	require.Contains(t, sql, "UPDATE billing_usage_entries")
	require.Contains(t, sql, "UPDATE usage_dashboard_hourly")
	require.Contains(t, sql, "COALESCE(SUM(COALESCE(account_stats_cost, total_cost) * COALESCE(account_rate_multiplier, 1)), 0)")
	require.Contains(t, sql, "UPDATE user_platform_quotas")
	require.Contains(t, sql, "po.order_type = 'subscription'")
	require.Contains(t, sql, "ual.action <> 'accrue'")
	require.Contains(t, sql, "ROUND(input_price / 70, 12)")
	require.Contains(t, sql, "ROUND(image_output_price / 70, 8)")
	require.Contains(t, sql, "'creditedAmount'")
	require.Contains(t, sql, "'rebateAmount'")
	require.Contains(t, sql, "UPDATE api_keys")
	require.Contains(t, sql, "quota_used = ROUND(quota_used / 10, 8)")
	require.Contains(t, sql, "UPDATE batch_image_jobs")

	// Gateway payment amounts and subscription/account-stat/usage ledgers are
	// already in their own currencies and must not be rewritten by this cutover.
	require.NotContains(t, lower, "pay_amount = round(pay_amount / 10")
	require.NotContains(t, lower, "subscription_orders")
	require.NotContains(t, lower, "account_stats_cost = round(account_stats_cost / 10")
	require.NotContains(t, lower, "channel_account_stats_model_pricing")
	// Subscription affiliate rebates are already USD; only balance-recharge
	// rebates and transfer snapshots use the legacy wallet conversion.
	require.NotContains(t, lower, "update user_affiliate_ledger set amount = round(amount / 10")
	// Dashboard totals contain both wallet and subscription rows; the migration
	// adjusts only the standard-wallet component via the bucket CTEs above.
	require.NotContains(t, lower, "update usage_dashboard_hourly set total_cost =")
	require.NotContains(t, lower, "update usage_dashboard_daily set total_cost =")
}
