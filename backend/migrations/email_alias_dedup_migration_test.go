package migrations

import (
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
