package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAPIKeyPurposeWebChatIndexesUseNonTransactionalMigration(t *testing.T) {
	baseContent, err := FS.ReadFile("183_api_key_purpose_web_chat.sql")
	require.NoError(t, err)
	baseSQL := strings.Join(strings.Fields(string(baseContent)), " ")
	require.Contains(t, baseSQL, "ADD COLUMN IF NOT EXISTS purpose")
	require.NotContains(t, baseSQL, "CREATE INDEX")
	require.NotContains(t, baseSQL, "CONCURRENTLY")

	indexContent, err := FS.ReadFile("183a_api_key_purpose_web_chat_indexes_notx.sql")
	require.NoError(t, err)
	indexSQL := strings.Join(strings.Fields(string(indexContent)), " ")
	require.Contains(t, indexSQL, "CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_api_keys_purpose")
	require.Contains(t, indexSQL, "CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_api_keys_web_chat_active_user_group")
	require.Contains(t, indexSQL, "WHERE purpose = 'web_chat' AND status = 'active' AND deleted_at IS NULL")
	require.NotContains(t, indexSQL, "ALTER TABLE")
}

func TestWebChatPrincipalRollbackCompatibilityMigration(t *testing.T) {
	baseContent, err := FS.ReadFile("184_web_chat_principal_rollback_compat.sql")
	require.NoError(t, err)
	baseSQL := strings.Join(strings.Fields(string(baseContent)), " ")
	require.Contains(t, baseSQL, "WHERE purpose = 'web_chat' AND status = 'active'")
	require.Contains(t, baseSQL, "ROW_NUMBER() OVER")
	require.Contains(t, baseSQL, "SET deleted_at = CURRENT_TIMESTAMP")
	require.Contains(t, baseSQL, "WHERE purpose = 'web_chat' AND deleted_at IS NULL")
	require.Contains(t, baseSQL, "CHECK (purpose <> 'web_chat' OR deleted_at IS NOT NULL)")
	require.NotContains(t, baseSQL, "CREATE INDEX")
	require.NotContains(t, baseSQL, "CONCURRENTLY")

	indexContent, err := FS.ReadFile("184a_web_chat_principal_rollback_compat_index_notx.sql")
	require.NoError(t, err)
	indexSQL := strings.Join(strings.Fields(string(indexContent)), " ")
	require.Contains(t, indexSQL, "DROP INDEX CONCURRENTLY IF EXISTS idx_api_keys_web_chat_principal_user_group")
	require.Contains(t, indexSQL, "CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_api_keys_web_chat_principal_user_group")
	require.Contains(t, indexSQL, "WHERE purpose = 'web_chat' AND status = 'active'")
	require.NotContains(t, indexSQL, "status = 'active' AND deleted_at IS NULL")
	require.Contains(t, indexSQL, "DROP INDEX CONCURRENTLY IF EXISTS idx_api_keys_web_chat_active_user_group")
	require.Less(t,
		strings.Index(indexSQL, "DROP INDEX CONCURRENTLY IF EXISTS idx_api_keys_web_chat_principal_user_group"),
		strings.Index(indexSQL, "CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_api_keys_web_chat_principal_user_group"),
	)
	require.Less(t,
		strings.Index(indexSQL, "CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_api_keys_web_chat_principal_user_group"),
		strings.Index(indexSQL, "DROP INDEX CONCURRENTLY IF EXISTS idx_api_keys_web_chat_active_user_group"),
	)
	require.NotContains(t, indexSQL, "ALTER TABLE")
}

func TestWebChatAttemptClaimMigrationUsesUserScopedPermanentUniqueness(t *testing.T) {
	content, err := FS.ReadFile("186_chat_request_attempts.sql")
	require.NoError(t, err)
	sql := strings.Join(strings.Fields(string(content)), " ")

	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS chat_request_attempts")
	require.Contains(t, sql, "attempt_id VARCHAR(64) NOT NULL")
	require.Contains(t, sql, "client_request_id VARCHAR(64) NOT NULL")
	require.Contains(t, sql, "request_hash VARCHAR(64) NOT NULL")
	require.Contains(t, sql, "status VARCHAR(20) NOT NULL DEFAULT 'accepted'")
	require.Contains(t, sql, "failure_code VARCHAR(64)")
	require.Contains(t, sql, "failure_reason VARCHAR(500)")
	require.Contains(t, sql, "ON chat_request_attempts (user_id, attempt_id)")
	require.Contains(t, sql, "ON chat_request_attempts (client_request_id)")
	require.NotContains(t, sql, "ON DELETE SET NULL")
}
