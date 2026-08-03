package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSkillMarketMigrationIsBoundedAndVersioned(t *testing.T) {
	content, err := FS.ReadFile("196_skill_market.sql")
	require.NoError(t, err)

	sql := strings.ToUpper(strings.Join(strings.Fields(string(content)), " "))
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS SKILLS")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS SKILL_VERSIONS")
	require.Contains(t, sql, "PACKAGE_DATA BYTEA NOT NULL")
	require.Contains(t, sql, "BYTE_SIZE <= 5242880")
	require.Contains(t, sql, "UNPACKED_SIZE <= 5242880")
	require.Contains(t, sql, "FILE_COUNT <= 100")
	require.Contains(t, sql, "SKILLS_CURRENT_VERSION_SAME_SKILL_FK")
	require.Contains(t, sql, "SKILLS_PUBLISHED_METADATA_CHECK")
	require.Contains(t, sql, "JSONB_ARRAY_LENGTH(EXAMPLE_PROMPTS) > 0")
	require.Contains(t, sql, "SKILL_VERSIONS_SKILL_SHA256_KEY")
	require.Contains(t, sql, "ON DELETE RESTRICT")
	require.Contains(t, sql, "ENFORCE_SKILL_VERSION_IMMUTABLE_CONTENT")
	require.Contains(t, sql, "PREVENT_SKILL_VERSION_DELETE")
	require.Contains(t, sql, "RELEASED_AT IS NOT NULL AND YANKED_AT IS NULL")
}
