package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWebChatAttachmentsMigrationKeepsBinaryPayloadsOutOfPostgres(t *testing.T) {
	content, err := FS.ReadFile("198_web_chat_attachments.sql")
	require.NoError(t, err)
	sql := strings.Join(strings.Fields(string(content)), " ")
	lowerSQL := strings.ToLower(sql)

	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS chat_attachments")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS chat_message_attachments")
	require.Contains(t, sql, "storage_key TEXT")
	require.Contains(t, sql, "stored_size BIGINT NOT NULL DEFAULT 0")
	require.Contains(t, sql, "extracted_text TEXT")
	require.Contains(t, sql, "expires_at TIMESTAMPTZ NOT NULL")
	require.Contains(t, sql, "status IN ('pending', 'ready', 'expired', 'deleted')")
	require.Contains(t, sql, "position BETWEEN 1 AND 4")
	require.Contains(t, sql, "REFERENCES chat_conversations(id, user_id) ON DELETE CASCADE")
	require.Contains(t, sql, "attachment_id BIGINT NOT NULL REFERENCES chat_attachments(id) ON DELETE CASCADE")
	require.NotContains(t, sql, "REFERENCES chat_conversations(id, user_id) ON DELETE SET NULL")
	require.NotContains(t, sql, "attachment_id BIGINT NOT NULL REFERENCES chat_attachments(id) ON DELETE RESTRICT")
	require.NotContains(t, lowerSQL, "bytea")
	require.NotContains(t, lowerSQL, "base64")
}
