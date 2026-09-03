//go:build integration

package repository

import (
	"context"
	"testing"

	dbmigrations "github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

func TestSkillCatalogChineseLocalizationMigrations(t *testing.T) {
	ctx := context.Background()
	db := openIsolatedMigrationIntegrationDB(t, "sub2api_skill_catalog_zh")
	files := []string{
		"237_skill_catalog_localizations.sql",
		"238_skill_catalog_zh_001_167.sql",
		"239_skill_catalog_zh_168_334.sql",
		"240_skill_catalog_zh_335_500.sql",
		"241_skill_catalog_zh_batch_1.sql",
		"242_skill_catalog_zh_batch_2.sql",
		"243_skill_catalog_zh_batch_3.sql",
		"244_skill_catalog_zh_batch_4.sql",
		"245_skill_catalog_zh_batch_5.sql",
		"246_skill_catalog_zh_batch_6.sql",
	}
	for _, name := range files {
		migration, err := dbmigrations.FS.ReadFile(name)
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, string(migration))
		require.NoErrorf(t, err, "apply %s", name)
	}

	var total, invalid int
	require.NoError(t, db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM skill_catalog_localizations WHERE locale='zh-CN'`).Scan(&total))
	require.Equal(t, 967, total)

	require.NoError(t, db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM skill_catalog_localizations
WHERE locale <> 'zh-CN'
   OR display_name <> btrim(display_name)
   OR summary <> btrim(summary)
   OR description <> btrim(description)
   OR char_length(display_name) > 120
   OR char_length(summary) > 280
   OR display_name !~ '[一-龥]'
   OR summary !~ '[一-龥]'
   OR description !~ '[一-龥]'`).Scan(&invalid))
	require.Zero(t, invalid)

	var rankedSnapshotTotal int
	require.NoError(t, db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM skill_catalog_localizations
WHERE locale='zh-CN'
  AND description ~ 'skills[.]sh 历史总榜第 [#]?[0-9]+ 名快照'`).Scan(&rankedSnapshotTotal))
	require.Equal(t, 500, rankedSnapshotTotal)

	require.NoError(t, db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM skill_catalog_localizations
WHERE description ~ 'skills[.]sh 历史总榜第 [#]?[0-9]+ 名快照'
  AND (description !~ '原始来源：'
   OR description !~ '原始 slug：')
   OR description ~* 'skills[.]sh all-time rank|Original source|Original slug|Packaging excluded|No license file|Copyright remains'`).Scan(&invalid))
	require.Zero(t, invalid)

	// The seed is intentionally idempotent because disaster recovery may replay
	// migration SQL against a restored catalog before checksums are reconciled.
	for _, name := range files[1:] {
		migration, err := dbmigrations.FS.ReadFile(name)
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, string(migration))
		require.NoErrorf(t, err, "reapply %s", name)
	}
	require.NoError(t, db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM skill_catalog_localizations WHERE locale='zh-CN'`).Scan(&total))
	require.Equal(t, 967, total)
}
