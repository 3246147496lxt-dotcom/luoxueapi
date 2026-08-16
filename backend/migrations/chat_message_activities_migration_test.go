package migrations

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChatMessageActivitiesMigrationDefinesAssistantOwnedSnapshotStorage(t *testing.T) {
	migration, err := FS.ReadFile("202_chat_message_activities.sql")
	require.NoError(t, err)
	rollback, err := os.ReadFile("rollback/202_chat_message_activities.down.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(migration)), " ")
	lowerSQL := strings.ToLower(sql)

	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS chat_message_activities")
	require.Contains(t, sql, "message_id BIGINT NOT NULL REFERENCES chat_messages(id) ON DELETE CASCADE")
	require.Contains(t, sql, "response_id VARCHAR(128) NOT NULL DEFAULT ''")
	require.Contains(t, sql, "metadata JSONB NOT NULL DEFAULT '{}'::jsonb")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS stop_requested_at TIMESTAMPTZ")
	require.Contains(t, sql, "'stopped'")
	require.Contains(t, sql, "'disconnected'")
	require.Contains(t, sql, "CONSTRAINT chat_message_activities_summary_part_unique UNIQUE")
	require.Contains(t, sql, "message_id, source, response_id, item_id, output_index, summary_index")
	require.Contains(t, sql, "CREATE TRIGGER trg_chat_message_activities_assistant")
	require.Contains(t, sql, "role = 'assistant'")
	require.Contains(t, sql, "CREATE INDEX IF NOT EXISTS idx_chat_message_activities_message_order")
	require.Contains(t, sql, "CREATE INDEX IF NOT EXISTS idx_chat_message_activities_response")
	require.NotContains(t, lowerSQL, "usage_logs")
	require.NotContains(t, lowerSQL, "billing_usage")

	downSQL := strings.Join(strings.Fields(string(rollback)), " ")
	require.Contains(t, downSQL, "DROP TABLE IF EXISTS chat_message_activities")
	require.Contains(t, downSQL, "DROP FUNCTION IF EXISTS enforce_chat_message_activity_assistant()")
	require.Contains(t, downSQL, "DROP COLUMN IF EXISTS stop_requested_at")
}
