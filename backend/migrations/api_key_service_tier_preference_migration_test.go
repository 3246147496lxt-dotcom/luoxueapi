package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAPIKeyServiceTierPreferenceMigrationBackfillsSafeDefault(t *testing.T) {
	content, err := FS.ReadFile("200_api_key_service_tier_preference.sql")
	require.NoError(t, err)
	sql := strings.Join(strings.Fields(string(content)), " ")

	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS service_tier_preference VARCHAR(20) NOT NULL DEFAULT 'standard'")
	require.Contains(t, sql, "SET service_tier_preference = 'standard'")
	require.Contains(t, sql, "ALTER COLUMN service_tier_preference SET NOT NULL")
	require.Contains(t, sql, "CHECK (service_tier_preference IN ('standard', 'priority'))")
	require.Contains(t, sql, "conrelid = 'api_keys'::regclass")
	require.NotContains(t, sql, "CONCURRENTLY")
}
