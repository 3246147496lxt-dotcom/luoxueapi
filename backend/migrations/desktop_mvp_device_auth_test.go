package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDesktopMVPDeviceAuthMigrationKeepsManagedKeysDeviceScoped(t *testing.T) {
	content, err := FS.ReadFile("190_desktop_mvp_device_auth.sql")
	require.NoError(t, err)
	sql := strings.Join(strings.Fields(string(content)), " ")

	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS desktop_devices")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS desktop_device_sessions")
	require.Contains(t, sql, "CHECK (purpose IN ('user', 'web_chat', 'desktop'))")
	require.Contains(t, sql, "managed_device_id BIGINT REFERENCES desktop_devices(id) ON DELETE RESTRICT")
	require.Contains(t, sql, "purpose = 'desktop' AND managed_device_id IS NOT NULL AND group_id IS NOT NULL")
	require.Contains(t, sql, "purpose <> 'desktop' AND managed_device_id IS NULL")
	require.Contains(t, sql, "CREATE UNIQUE INDEX IF NOT EXISTS idx_api_keys_desktop_device_group")
	require.Contains(t, sql, "WHERE purpose = 'desktop' AND deleted_at IS NULL")
	require.NotContains(t, sql, "managed_device_id BIGINT REFERENCES desktop_devices(id) ON DELETE SET NULL")
}

func TestDesktopReleaseAndDiagnosticMigrationKeepsChannelsAndCiphertextScoped(t *testing.T) {
	content, err := FS.ReadFile("191_desktop_releases_diagnostics.sql")
	require.NoError(t, err)
	sql := strings.Join(strings.Fields(string(content)), " ")

	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS release_channel VARCHAR(20) NOT NULL DEFAULT 'stable'")
	require.Contains(t, sql, "CHECK (release_channel IN ('internal', 'stable'))")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS desktop_diagnostics")
	require.Contains(t, sql, "device_id BIGINT NOT NULL REFERENCES desktop_devices(id) ON DELETE CASCADE")
	require.Contains(t, sql, "user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE")
	require.Contains(t, sql, "encrypted_payload TEXT NOT NULL")
	require.Contains(t, sql, "expires_at TIMESTAMPTZ NOT NULL")
	require.Contains(t, sql, "idx_desktop_diagnostics_expires_at")
	require.NotContains(t, strings.ToLower(sql), "diagnostics_encryption_key")
	require.NotContains(t, strings.ToLower(sql), "prompt")
}
