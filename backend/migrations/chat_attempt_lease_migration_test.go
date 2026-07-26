package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChatAttemptLeaseMigrationIsNarrowAndIndexesProcessingHistoryAttempts(t *testing.T) {
	migration, err := FS.ReadFile("189_web_chat_attempt_lease.sql")
	require.NoError(t, err)
	indexMigration, err := FS.ReadFile("189a_web_chat_attempt_lease_index_notx.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(migration)), " ")
	indexSQL := strings.Join(strings.Fields(string(indexMigration)), " ")
	combinedLower := strings.ToLower(sql + " " + indexSQL)

	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS lease_expires_at TIMESTAMPTZ")
	require.Contains(t, indexSQL, "CREATE INDEX CONCURRENTLY IF NOT EXISTS")
	require.Contains(t, indexSQL, "WHERE status = 'processing'")
	require.Contains(t, indexSQL, "conversation_public_id IS NOT NULL")
	require.Contains(t, indexSQL, "assistant_message_public_id IS NOT NULL")
	require.NotContains(t, combinedLower, "refund")
	require.NotContains(t, combinedLower, "usage_logs")
	require.NotContains(t, combinedLower, "billing_usage_entries")
}
