package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWebChatHistoryMigrationKeepsContentAndBillingLifecyclesSeparate(t *testing.T) {
	content, err := FS.ReadFile("187_web_chat_history.sql")
	require.NoError(t, err)
	sql := strings.Join(strings.Fields(string(content)), " ")
	lowerSQL := strings.ToLower(sql)

	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS chat_history_sync_states")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS chat_conversations")
	require.Contains(t, sql, "public_id VARCHAR(80) NOT NULL")
	require.Contains(t, sql, "revision BIGINT NOT NULL DEFAULT 1")
	require.Contains(t, sql, "version BIGINT NOT NULL DEFAULT 0")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS chat_messages")
	require.Contains(t, sql, "position BIGINT NOT NULL")
	require.Contains(t, sql, "delivery_status VARCHAR(20) NOT NULL")
	require.Contains(t, sql, "checkpoint_seq BIGINT NOT NULL DEFAULT 0")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS chat_history_changes")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS assistant_message_id BIGINT")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS conversation_public_id VARCHAR(80)")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS assistant_message_public_id VARCHAR(80)")
	require.Contains(t, sql, "REFERENCES chat_messages(id) ON DELETE SET NULL")

	require.NotContains(t, lowerSQL, "refund")
	require.NotContains(t, sql, "REFERENCES billing_usage_entries")
	require.NotContains(t, sql, "REFERENCES usage_logs")
}
