package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSkillMarketRepositoryClaimsStarsWithLeaseAndSkipLocked(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	claimedUntil := time.Date(2026, time.August, 5, 4, 15, 0, 0, time.UTC)
	mock.ExpectQuery(`(?s)repository_stars_refresh_after <= NOW\(\).*FOR UPDATE SKIP LOCKED.*LIMIT \$1.*UPDATE skills AS skill.*repository_stars_refresh_after = \$2.*RETURNING skill.id, skill.source_url, skill.source_repository`).
		WithArgs(20, claimedUntil).
		WillReturnRows(sqlmock.NewRows([]string{"id", "source_url", "source_repository"}).
			AddRow(int64(10), "https://github.com/anthropics/skills", "anthropics/skills").
			AddRow(int64(11), "https://github.com/openai/codex", "openai/codex"))

	repo := &skillMarketRepository{db: db}
	targets, err := repo.ClaimRepositoryStarsRefresh(context.Background(), 200, claimedUntil)
	require.NoError(t, err)
	require.Equal(t, []service.SkillRepositoryStarsTarget{
		{ID: 10, SourceURL: "https://github.com/anthropics/skills", SourceRepository: "anthropics/skills"},
		{ID: 11, SourceURL: "https://github.com/openai/codex", SourceRepository: "openai/codex"},
	}, targets)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSkillMarketRepositoryCompletesAndDefersStarsWithoutClearingSnapshot(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	fetchedAt := time.Date(2026, time.August, 5, 5, 0, 0, 0, time.UTC)
	refreshAfter := fetchedAt.Add(24 * time.Hour)
	sourceURL := "https://github.com/anthropics/skills"
	mock.ExpectExec(`(?s)UPDATE skills.*repository_stars = \$3.*repository_stars_fetched_at = \$4.*repository_stars_refresh_after = \$5.*WHERE id = \$1 AND source_url = \$2`).
		WithArgs(int64(7), sourceURL, int64(0), fetchedAt, refreshAfter).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)UPDATE skills.*SET repository_stars_refresh_after = \$3.*WHERE id = \$1 AND source_url = \$2`).
		WithArgs(int64(7), sourceURL, refreshAfter.Add(time.Hour)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	repo := &skillMarketRepository{db: db}
	require.NoError(t, repo.CompleteRepositoryStarsRefresh(
		context.Background(), 7, sourceURL, 0, fetchedAt, refreshAfter,
	))
	require.NoError(t, repo.DeferRepositoryStarsRefresh(
		context.Background(), 7, sourceURL, refreshAfter.Add(time.Hour),
	))
	require.ErrorIs(t, repo.CompleteRepositoryStarsRefresh(
		context.Background(), 7, sourceURL, -1, fetchedAt, refreshAfter,
	), service.ErrSkillInvalid)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSkillMarketRepositoryInsertVersionUsesExactlyFourteenArguments(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	actorID := int64(9)
	createdAt := time.Date(2026, time.August, 5, 6, 0, 0, 0, time.UTC)
	version := &service.SkillVersion{
		SkillID: 21, Version: "1.0.0", Changelog: "Initial release",
		ManifestName: "demo-skill", ManifestDescription: "A useful skill",
		SkillMD: "# Instructions", PackageData: []byte("zip"), SHA256: "sha",
		ByteSize: 3, UnpackedSize: 12, FileCount: 1,
		FileManifest:     []service.SkillArchiveFile{{Path: "demo-skill/SKILL.md", ByteSize: 12, SHA256: "file-sha"}},
		ValidationReport: service.SkillValidationReport{Valid: true, Errors: []service.SkillValidationIssue{}, Warnings: []service.SkillValidationIssue{}},
		CreatedBy:        &actorID,
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT slug, status FROM skills WHERE id=\$1 FOR UPDATE`).
		WithArgs(version.SkillID).
		WillReturnRows(sqlmock.NewRows([]string{"slug", "status"}).AddRow("demo-skill", service.SkillStatusDraft))
	mock.ExpectQuery(`(?s)INSERT INTO skill_versions.*VALUES \(\$1,\$2,\$3,\$4,\$5,\$6,\$7,\$8,\$9,\$10,\$11,\$12::jsonb,\$13::jsonb,\$14\).*RETURNING id, created_at`).
		WithArgs(
			version.SkillID, version.Version, version.Changelog, version.ManifestName,
			version.ManifestDescription, version.SkillMD, version.PackageData, version.SHA256,
			version.ByteSize, version.UnpackedSize, version.FileCount,
			sqlmock.AnyArg(), sqlmock.AnyArg(), actorID,
		).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(int64(88), createdAt))
	mock.ExpectCommit()

	repo := &skillMarketRepository{db: db}
	require.NoError(t, repo.InsertVersion(context.Background(), version))
	require.EqualValues(t, 88, version.ID)
	require.Equal(t, createdAt, version.CreatedAt)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBuildSkillMarketWhereSearchesLocalizedAndUpstreamCopy(t *testing.T) {
	where, args := buildSkillMarketWhere(service.SkillListFilter{
		Search: "浏览器",
	}, true, true, []any{"zh-CN"})

	require.Contains(t, where, "s.display_name ILIKE $2")
	require.Contains(t, where, "catalog_copy.display_name ILIKE $2")
	require.Contains(t, where, "catalog_copy.summary ILIKE $2")
	require.Contains(t, where, "catalog_copy.description ILIKE $2")
	require.Equal(t, []any{"zh-CN", "%浏览器%"}, args)
}

func TestBuildSkillMarketWhereKeepsAdminSearchOnUpstreamCopy(t *testing.T) {
	where, args := buildSkillMarketWhere(service.SkillListFilter{
		Search: "browser",
	}, false, false, nil)

	require.Contains(t, where, "s.display_name ILIKE $1")
	require.NotContains(t, where, "catalog_copy")
	require.Equal(t, []any{"%browser%"}, args)
}

func TestSkillMarketRepositoryPublicListJoinsRequestedCatalogLocale(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	sentinel := errors.New("stop after inspecting localized count query")
	mock.ExpectQuery(`(?s)SELECT COUNT\(\*\).*LEFT JOIN skill_catalog_localizations catalog_copy.*catalog_copy\.locale=\$1.*catalog_copy\.display_name ILIKE \$2`).
		WithArgs("zh-CN", "%浏览器%").
		WillReturnError(sentinel)

	repo := &skillMarketRepository{db: db}
	_, _, err = repo.ListPublished(context.Background(), service.SkillListFilter{
		Locale: "zh-CN", Search: "浏览器", Page: 1, PageSize: 12,
	})
	require.ErrorIs(t, err, sentinel)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSkillMarketRepositoryPublicDetailJoinsRequestedCatalogLocale(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	sentinel := errors.New("stop after inspecting localized detail query")
	mock.ExpectQuery(`(?s)LEFT JOIN skill_catalog_localizations catalog_copy.*catalog_copy\.locale=\$2.*WHERE s\.slug=\$1`).
		WithArgs("find-skills", "zh-CN").
		WillReturnError(sentinel)

	repo := &skillMarketRepository{db: db}
	_, err = repo.GetPublishedBySlugLocalized(context.Background(), "find-skills", "zh-CN")
	require.ErrorIs(t, err, sentinel)
	require.NoError(t, mock.ExpectationsWereMet())
}
