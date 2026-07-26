package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBillingUsageChargeReceiptsMigration(t *testing.T) {
	content, err := FS.ReadFile("185_billing_usage_charge_receipts.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "ALTER COLUMN usage_log_id DROP NOT NULL")
	require.Contains(t, sql, "REFERENCES usage_logs(id) ON DELETE SET NULL")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS request_id VARCHAR(255)")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS request_fingerprint VARCHAR(64)")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS source VARCHAR(20)")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS model VARCHAR(255)")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS requested_model VARCHAR(255)")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS balance_before DECIMAL(20, 10)")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS balance_after DECIMAL(20, 10)")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS charged_amount DECIMAL(20, 10)")
	require.Contains(t, sql, "billing_usage_entries_request_api_key_unique")
	require.Contains(t, sql, "WHERE request_id IS NOT NULL")
	require.Contains(t, sql, "status IN ('charged', 'not_charged', 'subscription')")

	require.NotContains(t, strings.ToLower(sql), "operator")
}
