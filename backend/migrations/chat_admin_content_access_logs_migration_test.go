package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChatAdminContentAccessLogsMigrationIsAppendOnlyAndIndependent(t *testing.T) {
	content, err := FS.ReadFile("188_chat_admin_content_access_logs.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS chat_admin_content_access_logs")
	require.Contains(t, sql, "admin_id BIGINT NOT NULL")
	require.Contains(t, sql, "target_user_id BIGINT NOT NULL")
	require.Contains(t, sql, "conversation_id BIGINT NOT NULL")
	require.Contains(t, sql, "conversation_public_id VARCHAR(80) NOT NULL")
	require.Contains(t, sql, "page_limit BETWEEN 1 AND 100")
	require.Contains(t, sql, "idx_chat_admin_content_access_admin_viewed")
	require.Contains(t, sql, "idx_chat_admin_content_access_target_viewed")
	require.Contains(t, sql, "idx_chat_admin_content_access_conversation_viewed")

	// Audit identities are immutable snapshots. Foreign keys could make a later
	// administrator, user, or conversation cleanup erase the access record.
	require.NotContains(t, strings.ToUpper(sql), " REFERENCES ")
}
