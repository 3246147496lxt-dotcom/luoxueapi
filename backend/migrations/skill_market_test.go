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

func TestSkillMarketSourceMigrationSimplifiesMetadataAndAddsSnapshot(t *testing.T) {
	content, err := FS.ReadFile("197_skill_market_simplify_and_source.sql")
	require.NoError(t, err)

	sql := strings.ToUpper(strings.Join(strings.Fields(string(content)), " "))
	require.Contains(t, sql, "SOURCE_URL TEXT NOT NULL DEFAULT ''")
	require.Contains(t, sql, "SOURCE_REPOSITORY VARCHAR(140) NOT NULL DEFAULT ''")
	require.Contains(t, sql, "REPOSITORY_STARS BIGINT NULL")
	require.Contains(t, sql, "REPOSITORY_STARS_FETCHED_AT TIMESTAMPTZ NULL")
	require.Contains(t, sql, "REPOSITORY_STARS_REFRESH_AFTER TIMESTAMPTZ NULL")
	require.Contains(t, sql, "SET DISPLAY_NAME = 'FRONTEND-DESIGN'")
	require.Contains(t, sql, "SOURCE_URL = 'HTTPS://GITHUB.COM/ANTHROPICS/SKILLS/TREE/MAIN/SKILLS/FRONTEND-DESIGN'")
	require.Contains(t, sql, "SOURCE_REPOSITORY = 'ANTHROPICS/SKILLS'")
	require.Contains(t, sql, "RISK_NOTES = ''")

	constraintAt := strings.LastIndex(sql, "ADD CONSTRAINT SKILLS_PUBLISHED_METADATA_CHECK")
	require.NotEqual(t, -1, constraintAt)
	updateAt := strings.Index(sql, "UPDATE SKILLS")
	require.NotEqual(t, -1, updateAt)
	require.Less(t, constraintAt, updateAt, "published metadata constraint must be relaxed before curated rows are updated")
	publishConstraint := sql[constraintAt:updateAt]
	require.Contains(t, publishConstraint, "BTRIM(SUMMARY) <> ''")
	require.NotContains(t, publishConstraint, "RISK_NOTES")
	require.NotContains(t, publishConstraint, "EXAMPLE_PROMPTS")
}

func TestSkillCatalogLocalizationMigrationKeepsUpstreamCopySeparate(t *testing.T) {
	content, err := FS.ReadFile("237_skill_catalog_localizations.sql")
	require.NoError(t, err)

	sql := strings.ToUpper(strings.Join(strings.Fields(string(content)), " "))
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS SKILL_CATALOG_LOCALIZATIONS")
	require.Contains(t, sql, "PRIMARY KEY (SLUG, LOCALE)")
	require.Contains(t, sql, "DISPLAY_NAME VARCHAR(120) NOT NULL")
	require.Contains(t, sql, "SUMMARY VARCHAR(280) NOT NULL")
	require.Contains(t, sql, "DESCRIPTION TEXT NOT NULL")
	require.NotContains(t, sql, "ALTER TABLE SKILLS", "catalog translations must not be overwritten by importer refreshes")
}
