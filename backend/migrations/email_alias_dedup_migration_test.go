package migrations

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEmailAliasDedupMigrationUsesBoundedConcurrentIndex(t *testing.T) {
	content, err := FS.ReadFile("231_add_users_email_alias_dedup_index_notx.sql")
	require.NoError(t, err)
	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_email_dot_stripped")
	require.Contains(t, sql, "REPLACE(LOWER(TRIM(email)), '.', '')")
	require.Contains(t, sql, "WHERE deleted_at IS NULL")
	require.NotContains(t, strings.ToLower(sql), "delete from users")
}

func TestEmailAliasDedupMigrationIndexesNormalizedProviderAddresses(t *testing.T) {
	content, err := FS.ReadFile("232_add_users_email_normalized_index_notx.sql")
	require.NoError(t, err)
	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_email_normalized")
	require.Contains(t, sql, "RTRIM(LOWER(TRIM(email)), '.')")
	require.Contains(t, sql, "WHERE deleted_at IS NULL")
	require.NotContains(t, strings.ToLower(sql), "delete from users")
}

func TestEmailAliasDedupAuditCoversProviderAliases(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "tools", "audit_email_alias_conflicts.sql"))
	require.NoError(t, err)
	sql := strings.ToLower(strings.Join(strings.Fields(string(content)), " "))
	require.Contains(t, sql, "split_part(local_part, '+', 1)")
	require.Contains(t, sql, "rtrim(split_part(lower(trim(email)), '@', 2), '.')")
	require.Contains(t, sql, "'gmail.com', 'googlemail.com'")
	require.Contains(t, sql, "|| '@gmail.com'")
}
