package migrations

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSkillImportMigrationPinsWorkflowStatusesAndSafeDefaultSchedule(t *testing.T) {
	data, err := FS.ReadFile("199_skill_import.sql")
	require.NoError(t, err)
	sqlText := string(data)

	for _, status := range []string{
		"'queued'", "'discovering'", "'preparing'", "'waiting_retry'",
		"'ready'", "'awaiting_review'", "'publishing'", "'succeeded'",
		"'partial_succeeded'", "'failed'", "'cancelled'",
	} {
		require.Contains(t, sqlText, status)
	}
	for _, status := range []string{
		"'processing'", "'unchanged'", "'blocked'", "'published'", "'skipped'",
	} {
		require.Contains(t, sqlText, status)
	}
	require.Contains(t, sqlText, "'Daily Top 500'")
	require.Contains(t, sqlText, "FALSE,\n    '0 3 * * *',\n    'Asia/Shanghai'")
	require.Contains(t, sqlText, `'{"start_rank":1,"limit":500}'::jsonb`)
	require.Contains(t, sqlText, `'{"bootstrap_existing":true}'::jsonb`)
	require.Contains(t, sqlText, "'bootstrap',\n    'dry_run',\n    'queued'")
	require.Contains(t, sqlText, `"acquisition_order":["github","skills_sh_snapshot"]`)
	require.Contains(t, sqlText, `"download_quota_per_hour":60`)
	require.Contains(t, sqlText, `"allowed_source_hosts":["open.feishu.cn","uizze.com","agent.qq.com","cli.sentry.dev"]`)
	require.Contains(t, sqlText, `"source_base_urls":{"open.feishu.cn":"https://open.feishu.cn","uizze.com":"https://uizze.com","agent.qq.com":"https://agent.qq.com","sentry/dev":"https://cli.sentry.dev"}`)
	require.Contains(t, sqlText, `"source_base_urls":{`)
	require.Contains(t, sqlText, `"open.feishu.cn":"https://open.feishu.cn"`)
	require.Contains(t, sqlText, `"sentry/dev":"https://cli.sentry.dev"`)
	require.Contains(t, sqlText, "'Manual Manifest Upload'")
	require.Contains(t, sqlText, `'{"upload_only":true}'::jsonb`)
	bootstrapSeed := regexp.MustCompile(`(?s)INSERT INTO skill_import_runs \(\s*source_id,\s*trigger_type,\s*mode,\s*status,\s*request_config,\s*snapshot,\s*idempotency_key_hash,\s*requested_count\s*\)\s*SELECT\s*id,\s*'bootstrap',\s*'dry_run',\s*'queued',\s*'\{"bootstrap_existing":true\}'::jsonb,\s*'\{\}'::jsonb,\s*'[a-f0-9]{64}',\s*0\s*FROM skill_import_sources`)
	require.Regexp(t, bootstrapSeed, sqlText)
	require.Contains(t, sqlText, `"auto_publish_gate":{"require_all_valid":false,"allow_license_unverified":true}`)
	require.Contains(t, sqlText, "'Manual Manifest Upload'")
	require.Contains(t, sqlText, "'manifest',\n    'manual-upload',\n    ''")
	require.Contains(t, sqlText, `'{"upload_only":true}'::jsonb`)
	require.Contains(t, sqlText, "base_url = ''")
	require.NotContains(t, sqlText, "'partial'")
	require.NotContains(t, sqlText, "'canceled'")
	require.NotContains(t, sqlText, "skill_import_run_items_run_rank_key")
}

func TestSkillImportMigrationAllowsMissingOriginURLButKeepsHTTPSConstraint(t *testing.T) {
	data, err := FS.ReadFile("199_skill_import.sql")
	require.NoError(t, err)
	sqlText := string(data)

	for _, constraint := range []string{
		"skill_import_run_items_origin_url_check",
		"skill_origins_origin_url_check",
	} {
		start := strings.Index(sqlText, "CONSTRAINT "+constraint+" CHECK")
		require.NotEqual(t, -1, start, "missing constraint %s", constraint)
		fragment := sqlText[start:]
		end := strings.Index(fragment, "),\n")
		require.NotEqual(t, -1, end, "unterminated constraint %s", constraint)
		fragment = fragment[:end]
		require.Contains(t, fragment, "origin_url = ''")
		require.Contains(t, fragment, "char_length(origin_url) <= 2048")
		require.Contains(t, fragment, "origin_url ~ '^https://[^[:space:]]+$'")
	}
}

func TestSkillImportMigrationStagesArtifactsWithoutConsumingMarketplaceRows(t *testing.T) {
	data, err := FS.ReadFile("199_skill_import.sql")
	require.NoError(t, err)
	sqlText := string(data)

	for _, fragment := range []string{
		"stage_action VARCHAR(24) NOT NULL DEFAULT ''",
		"staged_artifact JSONB NOT NULL DEFAULT '{}'::jsonb",
		"staged_package_data BYTEA NULL",
		"stage_action IN ('', 'create', 'new_version', 'unchanged')",
		"jsonb_typeof(staged_artifact) = 'object'",
		"octet_length(staged_artifact::text) <= 6291456",
		"octet_length(staged_package_data) BETWEEN 1 AND 5242880",
		"stage_action IN ('create', 'new_version')\n            AND staged_artifact <> '{}'::jsonb",
		"staged_package_data IS NOT NULL\n                OR status IN ('published', 'cancelled')",
		"idx_skill_import_run_items_staged_slug_reservation",
		"WHERE status IN ('ready', 'unchanged')",
		"AND stage_action IN ('create', 'new_version')",
	} {
		require.Contains(t, sqlText, fragment)
	}

	reservationIndex := regexp.MustCompile(`(?is)CREATE\s+INDEX\s+IF\s+NOT\s+EXISTS\s+idx_skill_import_run_items_staged_slug_reservation\s+ON\s+skill_import_run_items\s*\(\s*market_slug\s*,\s*run_id\s*\)`)
	require.Regexp(t, reservationIndex, sqlText)

	// Rank is upstream ordering metadata, not an identity. Duplicate or missing
	// ranks within the same run must remain valid.
	runRankUnique := regexp.MustCompile(`(?is)(?:UNIQUE\s*\(\s*run_id\s*,\s*rank\s*\)|UNIQUE\s*\(\s*rank\s*,\s*run_id\s*\)|CREATE\s+UNIQUE\s+INDEX[^;]*\(\s*(?:run_id\s*,\s*rank|rank\s*,\s*run_id)(?:\s|,|\)))`)
	require.NotRegexp(t, runRankUnique, sqlText)
}

func TestSkillImportManualRollbackIsOutsideEmbeddedMigrationSet(t *testing.T) {
	entries, err := FS.ReadDir(".")
	require.NoError(t, err)
	for _, entry := range entries {
		require.False(t, strings.Contains(entry.Name(), ".down."))
	}
	down, err := os.ReadFile("rollback/199_skill_import.down.sql")
	require.NoError(t, err)
	require.Contains(t, string(down), "DROP TABLE IF EXISTS skill_import_runs")
	require.NotContains(t, string(down), "DROP TRIGGER IF EXISTS", "partial rollback must not reference relations that may not exist")
}
