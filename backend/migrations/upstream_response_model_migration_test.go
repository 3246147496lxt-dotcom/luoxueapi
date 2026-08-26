package migrations

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpstreamResponseModelColumnMigrationIsAdditiveAndIdempotent(t *testing.T) {
	content, err := FS.ReadFile("234_add_usage_log_upstream_response_model.sql")
	require.NoError(t, err)
	sql := strings.Join(strings.Fields(string(content)), " ")

	require.Contains(t, sql, "ALTER TABLE usage_logs")
	require.Equal(t, 2, strings.Count(sql, "ADD COLUMN IF NOT EXISTS"))
	require.Contains(t, sql, "upstream_response_model VARCHAR(200)")
	require.Contains(t, sql, "upstream_model_mismatch BOOLEAN")
	require.False(t, regexp.MustCompile(`(?i)\b(DROP|DELETE|TRUNCATE|UPDATE|INSERT|RENAME)\b`).MatchString(sql))
}

func TestUpstreamResponseModelMismatchIndexMigrationIsConcurrentAndPartial(t *testing.T) {
	content, err := FS.ReadFile("235_add_usage_log_upstream_model_mismatch_index_notx.sql")
	require.NoError(t, err)
	sql := strings.Join(strings.Fields(string(content)), " ")

	require.Contains(t, sql, "CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_logs_upstream_model_mismatch_created_at")
	require.Contains(t, sql, "ON usage_logs (created_at DESC, id DESC)")
	require.Contains(t, sql, "WHERE upstream_model_mismatch IS TRUE")
	require.NotContains(t, strings.ToUpper(sql), "BEGIN")
	require.NotContains(t, strings.ToUpper(sql), "COMMIT")
}
