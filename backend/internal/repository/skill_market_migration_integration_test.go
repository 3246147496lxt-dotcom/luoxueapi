//go:build integration

package repository

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	dbmigrations "github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

func TestSkillMarketMigration196To197OnPostgreSQL(t *testing.T) {
	ctx := context.Background()
	db := openIsolatedMigrationIntegrationDB(t, "sub2api_skill_market_197")

	migration196, err := dbmigrations.FS.ReadFile("196_skill_market.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(migration196))
	require.NoError(t, err)

	var skillID int64
	err = db.QueryRowContext(ctx, `
INSERT INTO skills (
  slug, display_name, summary, description, category, tags,
  example_prompts, risk_notes, status
) VALUES (
  'frontend-design', '前端设计', '中文摘要', '中文概述', 'design-ui', '["ui/ux"]'::jsonb,
  '["设计一个页面"]'::jsonb, '无', 'draft'
)
RETURNING id`).Scan(&skillID)
	require.NoError(t, err)

	var versionID int64
	err = db.QueryRowContext(ctx, `
INSERT INTO skill_versions (
  skill_id, version, changelog, manifest_name, manifest_description, skill_md,
  package_data, sha256, byte_size, unpacked_size, file_count, file_manifest,
  validation_report, released_at
) VALUES (
  $1, '1.0.0', '', 'frontend-design', 'Design guidance', '# Frontend Design',
  $2, $3, 3, 24, 1,
  '[{"path":"frontend-design/SKILL.md","byte_size":24,"sha256":"file"}]'::jsonb,
  '{"valid":true,"errors":[],"warnings":[]}'::jsonb, NOW()
)
RETURNING id`, skillID, []byte("zip"), strings.Repeat("a", 64)).Scan(&versionID)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `
UPDATE skills
SET status='published', current_version_id=$2, published_at=NOW()
WHERE id=$1`, skillID, versionID)
	require.NoError(t, err)

	migration197, err := dbmigrations.FS.ReadFile("197_skill_market_simplify_and_source.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(migration197))
	require.NoError(t, err)

	var displayName, sourceURL, repository, riskNotes string
	var stars sql.NullInt64
	err = db.QueryRowContext(ctx, `
SELECT display_name, source_url, source_repository, repository_stars, risk_notes
FROM skills WHERE id=$1`, skillID).Scan(
		&displayName, &sourceURL, &repository, &stars, &riskNotes,
	)
	require.NoError(t, err)
	require.Equal(t, "frontend-design", displayName)
	require.Equal(t, "https://github.com/anthropics/skills/tree/main/skills/frontend-design", sourceURL)
	require.Equal(t, "anthropics/skills", repository)
	require.False(t, stars.Valid)
	require.Empty(t, riskNotes)

	_, err = db.ExecContext(ctx, `
UPDATE skills SET risk_notes='', example_prompts='[]'::jsonb WHERE id=$1`, skillID)
	require.NoError(t, err, "published Skill must no longer require risk notes or prompts")

	_, err = db.ExecContext(ctx, `UPDATE skills SET summary='' WHERE id=$1`, skillID)
	require.Error(t, err, "summary remains required for a published Skill")
}
