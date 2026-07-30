package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQuotaViewerDeviceAuthMigrationKeepsAuthorizationReadOnlyAndHashed(t *testing.T) {
	content, err := FS.ReadFile("193_quota_viewer_device_auth.sql")
	require.NoError(t, err)
	sql := strings.Join(strings.Fields(string(content)), " ")

	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS quota_viewer_devices")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS quota_viewer_device_sessions")
	require.Contains(t, sql, "CHECK (client_id = 'luoxue-quota-viewer')")
	require.Contains(t, sql, "CHECK (scope = 'quota:read')")
	require.Contains(t, sql, "CHECK (platform IN ('macos', 'windows'))")
	require.Contains(t, sql, "refresh_token_hash VARCHAR(64) NOT NULL UNIQUE")
	require.Contains(t, sql, "CHECK (length(refresh_token_hash) = 64)")
	require.Contains(t, sql, "REFERENCES quota_viewer_devices(id) ON DELETE CASCADE")
	require.Contains(t, sql, "idx_quota_viewer_device_sessions_one_active_device")
	require.Contains(t, sql, "idx_quota_viewer_device_sessions_one_active_family")
	require.NotContains(t, sql, "refresh_token VARCHAR")
	require.NotContains(t, sql, "api_keys")
	require.NotContains(t, sql, "managed_key")
}

func TestQuotaViewerRefreshRecoveryMigrationIsAdditiveBoundedAndHashOnly(t *testing.T) {
	content, err := FS.ReadFile("194_quota_viewer_refresh_recovery.sql")
	require.NoError(t, err)
	sql := strings.Join(strings.Fields(string(content)), " ")

	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS rotation_id UUID")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS replacement_token_hash VARCHAR(64)")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS recovery_expires_at TIMESTAMPTZ")
	require.Contains(t, sql, "rotation_id IS NULL AND replacement_token_hash IS NULL AND recovery_expires_at IS NULL")
	require.Contains(t, sql, "rotation_id IS NOT NULL AND replacement_token_hash IS NOT NULL AND recovery_expires_at IS NOT NULL")
	require.Contains(t, sql, "consumed_at IS NOT NULL")
	require.Contains(t, sql, "recovery_expires_at > consumed_at")
	require.Contains(t, sql, "status IN ('consumed', 'revoked')")
	require.Contains(t, sql, "replacement_token_hash ~ '^[0-9a-f]{64}$'")
	require.Contains(t, sql, "ON quota_viewer_device_sessions(device_id, rotation_id)")
	require.Contains(t, sql, "WHERE rotation_id IS NOT NULL")
	require.Contains(t, sql, "ON quota_viewer_device_sessions(recovery_expires_at)")
	require.Contains(t, sql, "WHERE recovery_expires_at IS NOT NULL")
	require.NotContains(t, sql, "candidate_refresh_token")
	require.NotContains(t, sql, "replacement_token VARCHAR")
}
