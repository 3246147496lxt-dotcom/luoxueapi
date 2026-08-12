package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSkillImportRepositoryClaimsStaleItemWithDatabaseLease(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	now := time.Date(2026, time.August, 12, 1, 0, 0, 0, time.UTC)
	leaseUntil := now.Add(5 * time.Minute)

	mock.ExpectQuery(`(?s)FOR UPDATE OF i SKIP LOCKED LIMIT 1.*UPDATE skill_import_run_items i SET status='processing'.*lease_owner=\$3.*lease_expires_at=\$5`).
		WithArgs(int64(17), "worker-a", "item-claim-a", now, leaseUntil).
		WillReturnRows(skillImportItemRows().AddRow(
			int64(91), int64(17), int64(2), "skills.sh", "owner/repo/demo",
			1, "demo", "processing", "demo", "https://skills.sh/demo", "abc",
			sha64("1"), sha64("2"), "", []byte(`{"slug":"demo"}`), []byte(`{}`),
			[]byte(`{"valid":true,"errors":[],"warnings":[]}`), []byte(`{}`),
			false, []byte(`[]`), []byte(`[]`), nil, nil, 2, nil, "", "",
			"item-claim-a", leaseUntil, now, now, nil, now, now,
		))

	repo := &skillImportRepository{db: db}
	item, err := repo.ClaimNextRunItem(context.Background(), 17, "worker-a", "item-claim-a", now, leaseUntil)
	require.NoError(t, err)
	require.EqualValues(t, 91, item.ID)
	require.Equal(t, service.SkillImportItemStatusProcessing, item.Status)
	require.Equal(t, 2, item.AttemptCount)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSkillImportRepositoryGetsNextItemRetryWithUnpaginatedAggregate(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	want := time.Date(2026, time.August, 12, 1, 4, 0, 0, time.UTC)

	mock.ExpectQuery(`(?s)SELECT MIN\(CASE.*WHEN status='queued' THEN next_attempt_at.*WHEN status='processing' THEN lease_expires_at.*FROM skill_import_run_items.*WHERE run_id=\$1 AND status IN \('queued','processing'\)`).
		WithArgs(int64(17)).
		WillReturnRows(sqlmock.NewRows([]string{"next_retry_at"}).AddRow(want))

	repo := &skillImportRepository{db: db}
	got, err := repo.GetNextRunItemRetryAt(context.Background(), 17)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, want, *got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSkillImportRepositoryItemMutationsRequireCurrentParentLease(t *testing.T) {
	t.Run("heartbeat", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		t.Cleanup(func() { _ = db.Close() })
		leaseUntil := time.Now().UTC().Add(time.Minute)
		mock.ExpectExec(`(?s)UPDATE skill_import_run_items i SET lease_expires_at=\$4.*FROM skill_import_runs r.*r.lease_owner=\$3.*r.lease_expires_at > NOW\(\)`).
			WithArgs(int64(91), "item-claim-a", "worker-a", leaseUntil).
			WillReturnResult(sqlmock.NewResult(0, 0))

		repo := &skillImportRepository{db: db}
		err = repo.HeartbeatRunItem(context.Background(), 91, "item-claim-a", "worker-a", leaseUntil)
		require.ErrorIs(t, err, service.ErrSkillImportLeaseLost)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("complete", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		t.Cleanup(func() { _ = db.Close() })
		mock.ExpectExec(`(?s)UPDATE skill_import_run_items i SET status=\$3.*FROM skill_import_runs r.*r.lease_owner=\$20.*r.lease_expires_at > NOW\(\)`).
			WillReturnResult(sqlmock.NewResult(0, 0))

		repo := &skillImportRepository{db: db}
		err = repo.CompleteRunItem(context.Background(), 91, "item-claim-a", "worker-a", service.SkillImportRunItemPatch{
			Status: service.SkillImportItemStatusFailed,
		})
		require.ErrorIs(t, err, service.ErrSkillImportLeaseLost)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSkillImportRepositoryResetFailedItemsAlsoRequeuesParentRun(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT status, lease_owner, lease_expires_at.*WHERE id=\$1 FOR UPDATE`).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"status", "lease_owner", "lease_expires_at"}).AddRow(service.SkillImportRunStatusPartialSucceeded, nil, nil))
	mock.ExpectExec(`(?s)UPDATE skill_import_run_items SET status='queued'.*status IN \('failed','blocked'\)`).
		WithArgs(int64(7), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(`(?s)UPDATE skill_import_runs SET status='queued'.*finished_at=NULL`).
		WithArgs(int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`(?s)WITH item_counts AS.*UPDATE skill_import_runs r SET.*SELECT \* FROM updated`).
		WithArgs(int64(7)).
		WillReturnRows(importCountRows().AddRow(500, 500, 498, 490, 8, 2, 0, 0, 0, 490))
	mock.ExpectCommit()

	repo := &skillImportRepository{db: db}
	affected, err := repo.ResetFailedItems(context.Background(), 7, []int64{44, 45})
	require.NoError(t, err)
	require.EqualValues(t, 2, affected)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSkillImportRepositoryResetFailedItemsRecoversStaleProcessing(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT status, lease_owner, lease_expires_at.*WHERE id=\$1 FOR UPDATE`).
		WithArgs(int64(8)).
		WillReturnRows(sqlmock.NewRows([]string{"status", "lease_owner", "lease_expires_at"}).
			AddRow(service.SkillImportRunStatusFailed, nil, nil))
	mock.ExpectExec(`(?s)UPDATE skill_import_run_items SET status='queued'.*status='failed'.*status='processing'.*lease_expires_at <= \$2`).
		WithArgs(int64(8), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)UPDATE skill_import_runs SET status='queued'.*finished_at=NULL`).
		WithArgs(int64(8)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`(?s)WITH item_counts AS.*UPDATE skill_import_runs r SET.*SELECT \* FROM updated`).
		WithArgs(int64(8)).
		WillReturnRows(importCountRows().AddRow(500, 500, 499, 490, 9, 0, 0, 0, 0, 0))
	mock.ExpectCommit()

	repo := &skillImportRepository{db: db}
	affected, err := repo.ResetFailedItems(context.Background(), 8, nil)
	require.NoError(t, err)
	require.EqualValues(t, 1, affected)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSkillImportRepositoryResetFailedRunAllowsParentOnlyRecovery(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT status, lease_owner, lease_expires_at.*WHERE id=\$1 FOR UPDATE`).
		WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"status", "lease_owner", "lease_expires_at"}).
			AddRow(service.SkillImportRunStatusFailed, nil, nil))
	mock.ExpectExec(`(?s)UPDATE skill_import_run_items SET status='queued'.*status='failed'`).
		WithArgs(int64(9), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`(?s)UPDATE skill_import_runs SET status='queued'.*finished_at=NULL`).
		WithArgs(int64(9)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`(?s)WITH item_counts AS.*UPDATE skill_import_runs r SET.*SELECT \* FROM updated`).
		WithArgs(int64(9)).
		WillReturnRows(importCountRows().AddRow(500, 500, 500, 490, 0, 10, 0, 0, 0, 0))
	mock.ExpectCommit()

	repo := &skillImportRepository{db: db}
	affected, err := repo.ResetFailedItems(context.Background(), 9, nil)
	require.NoError(t, err)
	require.Zero(t, affected)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSkillImportRepositoryCancellationImmediatelyFinishesIdleRunStates(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT status FROM skill_import_runs WHERE id=\$1 FOR UPDATE`).
		WithArgs(int64(27)).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(service.SkillImportRunStatusReady))
	mock.ExpectExec(`(?s)UPDATE skill_import_runs SET.*status='cancelled'.*finished_at=NOW\(\)`).
		WithArgs(int64(27), int64(5), service.SkillImportRunStatusReady).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)UPDATE skill_import_run_items SET status='cancelled'.*status NOT IN \('published','unchanged','skipped','cancelled'\)`).
		WithArgs(int64(27)).WillReturnResult(sqlmock.NewResult(0, 4))
	mock.ExpectExec(`(?s)UPDATE skill_import_run_items SET.*staged_package_data=NULL.*staged_artifact=staged_artifact - 'skill_md' - 'file_manifest'`).
		WithArgs(int64(27)).WillReturnResult(sqlmock.NewResult(0, 4))
	mock.ExpectCommit()

	repo := &skillImportRepository{db: db}
	changed, err := repo.RequestRunCancellation(context.Background(), 27, 5)
	require.NoError(t, err)
	require.True(t, changed)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSkillImportRepositoryEligibleCohortIncludesMetadataOnlyItems(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectQuery(`(?s)SELECT id FROM skill_import_run_items.*status IN \('ready','unchanged'\).*validation_report`).
		WithArgs(int64(41)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(8)).AddRow(int64(9)))

	repo := &skillImportRepository{db: db}
	ids, err := repo.ListEligibleItemIDs(context.Background(), 41)
	require.NoError(t, err)
	require.Equal(t, []int64{8, 9}, ids)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSkillImportRepositoryEventRetentionOnlyDeletesTerminalRunHistory(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	before := time.Date(2026, time.February, 13, 0, 0, 0, 0, time.UTC)

	mock.ExpectExec(`(?s)DELETE FROM skill_import_events e.*USING skill_import_runs r.*e.created_at < \$1.*partial_succeeded.*cancelled`).
		WithArgs(before).
		WillReturnResult(sqlmock.NewResult(0, 24))

	repo := &skillImportRepository{db: db}
	removed, err := repo.DeleteTerminalRunEventsBefore(context.Background(), before)
	require.NoError(t, err)
	require.EqualValues(t, 24, removed)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSkillImportRepositoryStageReturnsRenormalizeBeforeAnyWrite(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT r.source_id, r.status, r.cancel_requested_at, src.catalog_priority.*FOR UPDATE OF r`).
		WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "status", "cancel_requested_at", "catalog_priority", "lease_owner", "lease_expires_at",
		}).AddRow(int64(2), service.SkillImportRunStatusPreparing, nil, 100, "worker-a", time.Now().UTC().Add(time.Hour)))
	mock.ExpectQuery(`(?s)SELECT i.source_id, i.namespace, i.external_id, i.status.*FOR UPDATE OF i`).
		WithArgs(int64(33), int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "namespace", "external_id", "status", "lease_owner",
		}).AddRow(int64(2), "skills.sh", "owner/repo/demo",
			service.SkillImportItemStatusProcessing, "item-claim-a"))
	mock.ExpectExec(`SELECT pg_advisory_xact_lock\(hashtextextended\(\$1,0\)\)`).
		WithArgs("demo").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`(?s)SELECT o.skill_id, o.market_slug, s.slug, s.status.*FROM skill_origins.*FOR UPDATE OF o, s`).
		WithArgs(int64(2), "skills.sh", "owner/repo/demo").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`(?s)SELECT EXISTS\(SELECT 1 FROM skills WHERE slug=\$1\).*skill_import_run_items`).
		WithArgs("demo", int64(33)).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`(?s)SELECT EXISTS\(SELECT 1 FROM skills WHERE slug=\$1\).*skill_import_run_items`).
		WithArgs(sqlmock.AnyArg(), int64(33)).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectRollback()

	repo := &skillImportRepository{db: db}
	result, err := repo.StagePreparedItem(context.Background(), service.SkillImportStagePreparedInput{
		RunID: 9, RunItemID: 33, RunWorkerID: "worker-a", ItemLeaseOwner: "item-claim-a",
		StableKey:    service.SkillImportStableKey{SourceID: 2, Namespace: "skills.sh", ExternalID: "owner/repo/demo"},
		DesiredSkill: service.SkillImportDesiredSkill{Slug: "demo", OriginURL: "https://skills.sh/demo"},
		OriginURL:    "https://skills.sh/demo",
		Artifact: service.SkillImportPreparedArtifact{
			ManifestName: "demo", PackageData: []byte("zip"), PackageSHA256: sha64("2"),
			ValidationReport: service.SkillValidationReport{Valid: true},
		},
	})
	require.NoError(t, err)
	require.Equal(t, service.SkillImportStageActionRenormalize, result.Action)
	require.NotEqual(t, "demo", result.MarketSlug)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSkillImportRepositoryStageRejectsStaleItemClaimToken(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT r.source_id, r.status, r.cancel_requested_at, src.catalog_priority.*FOR UPDATE OF r`).
		WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "status", "cancel_requested_at", "catalog_priority", "lease_owner", "lease_expires_at",
		}).AddRow(int64(2), service.SkillImportRunStatusPreparing, nil, 100, "worker-a", time.Now().UTC().Add(time.Hour)))
	mock.ExpectQuery(`(?s)SELECT i.source_id, i.namespace, i.external_id, i.status.*FOR UPDATE OF i`).
		WithArgs(int64(33), int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "namespace", "external_id", "status", "lease_owner",
		}).AddRow(int64(2), "skills.sh", "owner/repo/demo",
			service.SkillImportItemStatusProcessing, "new-item-claim"))
	mock.ExpectRollback()

	repo := &skillImportRepository{db: db}
	_, err = repo.StagePreparedItem(context.Background(), service.SkillImportStagePreparedInput{
		RunID: 9, RunItemID: 33, RunWorkerID: "worker-a", ItemLeaseOwner: "stale-item-claim",
		StableKey:    service.SkillImportStableKey{SourceID: 2, Namespace: "skills.sh", ExternalID: "owner/repo/demo"},
		DesiredSkill: service.SkillImportDesiredSkill{Slug: "demo", OriginURL: "https://skills.sh/demo"},
		Artifact: service.SkillImportPreparedArtifact{
			ManifestName: "demo", PackageData: []byte("zip"), PackageSHA256: sha64("2"),
			ValidationReport: service.SkillValidationReport{Valid: true},
		},
	})
	require.ErrorIs(t, err, service.ErrSkillImportLeaseLost)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSkillImportRepositoryStagePersistsCreateWithoutMarketplaceRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT r.source_id, r.status, r.cancel_requested_at, src.catalog_priority.*FOR UPDATE OF r`).
		WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "status", "cancel_requested_at", "catalog_priority", "lease_owner", "lease_expires_at",
		}).AddRow(int64(2), service.SkillImportRunStatusPreparing, nil, 100, "worker-a", time.Now().UTC().Add(time.Hour)))
	mock.ExpectQuery(`(?s)SELECT i.source_id, i.namespace, i.external_id, i.status.*FOR UPDATE OF i`).
		WithArgs(int64(33), int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "namespace", "external_id", "status", "lease_owner",
		}).AddRow(int64(2), "skills.sh", "owner/repo/demo",
			service.SkillImportItemStatusProcessing, "item-claim-a"))
	mock.ExpectExec(`SELECT pg_advisory_xact_lock\(hashtextextended\(\$1,0\)\)`).
		WithArgs("demo").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`(?s)SELECT o.skill_id, o.market_slug, s.slug, s.status.*FROM skill_origins.*FOR UPDATE OF o, s`).
		WithArgs(int64(2), "skills.sh", "owner/repo/demo").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`(?s)SELECT EXISTS\(SELECT 1 FROM skills WHERE slug=\$1\).*skill_import_run_items`).
		WithArgs("demo", int64(33)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	// Any INSERT into skills, versions, origins, or version_origins here would
	// be an unexpected sqlmock call and fail this test.
	mock.ExpectExec(`(?s)UPDATE skill_import_run_items SET.*stage_action=\$16.*staged_artifact=\$17::jsonb.*staged_package_data=\$18`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`(?s)WITH item_counts AS.*i.stage_action='create'.*i.stage_action='new_version'.*UPDATE skill_import_runs`).
		WithArgs(int64(9)).
		WillReturnRows(importCountRows().AddRow(1, 1, 1, 1, 0, 0, 0, 0, 0, 0))
	mock.ExpectCommit()

	repo := &skillImportRepository{db: db}
	result, err := repo.StagePreparedItem(context.Background(), service.SkillImportStagePreparedInput{
		RunID: 9, RunItemID: 33, RunWorkerID: "worker-a", ItemLeaseOwner: "item-claim-a",
		StableKey: service.SkillImportStableKey{SourceID: 2, Namespace: "skills.sh", ExternalID: "owner/repo/demo"},
		DesiredSkill: service.SkillImportDesiredSkill{
			Slug: "demo", DisplayName: "Demo", OriginURL: "https://skills.sh/demo",
		},
		OriginURL: "https://skills.sh/demo",
		Artifact: service.SkillImportPreparedArtifact{
			ManifestName: "demo", ManifestDescription: "Demo", SkillMD: "# Demo",
			PackageData: []byte("zip"), PackageSHA256: sha64("2"), ByteSize: 3,
			UnpackedSize: 6, FileCount: 1,
			FileManifest:     []service.SkillArchiveFile{{Path: "SKILL.md", ByteSize: 6, SHA256: sha64("3")}},
			ValidationReport: service.SkillValidationReport{Valid: true},
		},
	})
	require.NoError(t, err)
	require.Equal(t, service.SkillImportStageActionCreate, result.Action)
	require.Equal(t, "demo", result.MarketSlug)
	require.Zero(t, result.SkillID)
	require.Zero(t, result.VersionID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSkillImportRepositoryStageRejectsYankedMatchingPackage(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT r.source_id, r.status, r.cancel_requested_at, src.catalog_priority.*FOR UPDATE OF r`).
		WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{
		"source_id", "status", "cancel_requested_at", "catalog_priority", "lease_owner", "lease_expires_at",
	}).AddRow(int64(2), service.SkillImportRunStatusPreparing, nil, 100, "worker-a", time.Now().UTC().Add(time.Hour)))
	mock.ExpectQuery(`(?s)SELECT i.source_id, i.namespace, i.external_id, i.status.*FOR UPDATE OF i`).
		WithArgs(int64(33), int64(9)).WillReturnRows(sqlmock.NewRows([]string{
		"source_id", "namespace", "external_id", "status", "lease_owner",
	}).AddRow(int64(2), "skills.sh", "owner/repo/demo", service.SkillImportItemStatusProcessing, "item-claim-a"))
	mock.ExpectExec(`SELECT pg_advisory_xact_lock`).WithArgs("demo").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`(?s)SELECT o.skill_id, o.market_slug, s.slug, s.status.*FOR UPDATE OF o, s`).
		WithArgs(int64(2), "skills.sh", "owner/repo/demo").
		WillReturnRows(sqlmock.NewRows([]string{"skill_id", "market_slug", "slug", "status"}).
			AddRow(int64(101), "demo", "demo", service.SkillStatusPublished))
	mock.ExpectQuery(`(?s)SELECT id, version, yanked_at FROM skill_versions.*sha256=\$2`).
		WithArgs(int64(101), sha64("2")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "version", "yanked_at"}).
			AddRow(int64(201), "1.0.0", time.Now().UTC()))
	mock.ExpectRollback()

	repo := &skillImportRepository{db: db}
	_, err = repo.StagePreparedItem(context.Background(), service.SkillImportStagePreparedInput{
		RunID: 9, RunItemID: 33, RunWorkerID: "worker-a", ItemLeaseOwner: "item-claim-a",
		StableKey:    service.SkillImportStableKey{SourceID: 2, Namespace: "skills.sh", ExternalID: "owner/repo/demo"},
		DesiredSkill: service.SkillImportDesiredSkill{Slug: "demo", OriginURL: "https://skills.sh/demo"},
		Artifact: service.SkillImportPreparedArtifact{
			ManifestName: "demo", PackageData: []byte("zip"), PackageSHA256: sha64("2"),
			ValidationReport: service.SkillValidationReport{Valid: true},
		},
	})
	require.ErrorIs(t, err, service.ErrSkillImportVersionYanked)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSkillImportRepositoryPublishCreatesStagedMarketplaceRowsAtomically(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	expectStagedCreatePublishPrelude(mock, 9, 51, "demo")
	mock.ExpectQuery(`(?s)SELECT EXISTS\(.*FROM skill_origins.*OR EXISTS\(SELECT 1 FROM skills WHERE slug=\$4\)`).
		WithArgs(int64(2), "skills.sh", "owner/repo/demo", "demo").
		WillReturnRows(sqlmock.NewRows([]string{"occupied"}).AddRow(false))
	mock.ExpectQuery(`(?s)INSERT INTO skills \(.*status.*\).*RETURNING id`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(101)))
	mock.ExpectQuery(`(?s)INSERT INTO skill_versions \(.*package_data.*\).*RETURNING id`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(201)))
	mock.ExpectQuery(`(?s)INSERT INTO skill_origins \(.*ON CONFLICT.*RETURNING id`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(301)))
	mock.ExpectExec(`(?s)INSERT INTO skill_version_origins \(.*ON CONFLICT.*DO NOTHING`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)UPDATE skill_versions SET released_at=COALESCE`).
		WithArgs(int64(201), int64(101), nil).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)UPDATE skills SET status='published'.*current_version_id=\$2`).
		WithArgs(int64(101), int64(201), nil).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)UPDATE skill_import_run_items SET status=\$2, skill_id=\$3, version_id=\$4,.*staged_package_data=NULL.*staged_artifact=staged_artifact - 'skill_md' - 'file_manifest'`).
		WithArgs(int64(51), service.SkillImportItemStatusPublished, int64(101), int64(201)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`(?s)WITH item_counts AS.*i.stage_action='create'.*UPDATE skill_import_runs`).
		WithArgs(int64(9)).
		WillReturnRows(importCountRows().AddRow(1, 1, 1, 1, 0, 0, 0, 0, 0, 1))
	mock.ExpectExec(`(?s)UPDATE skill_import_runs SET status=\$2.*finished_at=NOW\(\)`).
		WithArgs(int64(9), service.SkillImportRunStatusSucceeded, true).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)INSERT INTO skill_import_events .*publication_completed`).
		WithArgs(int64(9), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	repo := &skillImportRepository{db: db}
	result, err := repo.PublishEligibleItems(context.Background(), 9, []int64{51}, "worker-a", nil)
	require.NoError(t, err)
	require.Equal(t, service.SkillImportRunStatusSucceeded, result.Status)
	require.Equal(t, []service.SkillImportPublishItemResult{{
		RunItemID: 51, SkillID: 101, VersionID: 201,
	}}, result.Items)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSkillImportRepositoryPublishRollsBackCreatedSkillWhenVersionFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	expectStagedCreatePublishPrelude(mock, 9, 51, "demo")
	mock.ExpectQuery(`(?s)SELECT EXISTS\(.*FROM skill_origins.*OR EXISTS\(SELECT 1 FROM skills WHERE slug=\$4\)`).
		WithArgs(int64(2), "skills.sh", "owner/repo/demo", "demo").
		WillReturnRows(sqlmock.NewRows([]string{"occupied"}).AddRow(false))
	mock.ExpectQuery(`(?s)INSERT INTO skills \(.*status.*\).*RETURNING id`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(101)))
	mock.ExpectQuery(`(?s)INSERT INTO skill_versions \(.*package_data.*\).*RETURNING id`).
		WillReturnError(errors.New("injected immutable version failure"))
	mock.ExpectRollback()

	repo := &skillImportRepository{db: db}
	_, err = repo.PublishEligibleItems(context.Background(), 9, []int64{51}, "worker-a", nil)
	require.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSkillImportRepositoryPublishCreatesNewVersionAtomically(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	expectStagedNewVersionPublishPrelude(mock, 10, 61, "demo")
	mock.ExpectQuery(`(?s)SELECT o.skill_id, s.slug, s.status.*FOR UPDATE OF o, s`).
		WithArgs(int64(2), "skills.sh", "owner/repo/demo").
		WillReturnRows(sqlmock.NewRows([]string{"skill_id", "slug", "status"}).
			AddRow(int64(101), "demo", service.SkillStatusPublished))
	mock.ExpectQuery(`(?s)SELECT id, version FROM skill_versions.*sha256=\$2.*yanked_at IS NULL`).
		WithArgs(int64(101), sha64("2")).WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`SELECT version FROM skill_versions WHERE skill_id=\$1 ORDER BY id`).
		WithArgs(int64(101)).
		WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow("1.0.0"))
	mock.ExpectQuery(`(?s)INSERT INTO skill_versions \(.*package_data.*\).*RETURNING id`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(202)))
	mock.ExpectExec(`(?s)UPDATE skills SET display_name=\$2.*WHERE id=\$1 AND slug=\$18`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`(?s)INSERT INTO skill_origins \(.*ON CONFLICT.*RETURNING id`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(301)))
	mock.ExpectExec(`(?s)INSERT INTO skill_version_origins \(.*ON CONFLICT.*DO NOTHING`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)UPDATE skill_versions SET released_at=COALESCE`).
		WithArgs(int64(202), int64(101), nil).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)UPDATE skills SET status='published'.*current_version_id=\$2`).
		WithArgs(int64(101), int64(202), nil).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)UPDATE skill_import_run_items SET status=\$2, skill_id=\$3, version_id=\$4,.*staged_package_data=NULL`).
		WithArgs(int64(61), service.SkillImportItemStatusPublished, int64(101), int64(202)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`(?s)WITH item_counts AS.*i.stage_action='new_version'.*UPDATE skill_import_runs`).
		WithArgs(int64(10)).
		WillReturnRows(importCountRows().AddRow(1, 1, 1, 0, 1, 0, 0, 0, 0, 1))
	mock.ExpectExec(`(?s)UPDATE skill_import_runs SET status=\$2.*finished_at=NOW\(\)`).
		WithArgs(int64(10), service.SkillImportRunStatusSucceeded, true).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)INSERT INTO skill_import_events .*publication_completed`).
		WithArgs(int64(10), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	repo := &skillImportRepository{db: db}
	result, err := repo.PublishEligibleItems(context.Background(), 10, []int64{61}, "worker-a", nil)
	require.NoError(t, err)
	require.EqualValues(t, 202, result.Items[0].VersionID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func expectStagedCreatePublishPrelude(mock sqlmock.Sqlmock, runID, itemID int64, slug string) {
	desired := []byte(`{"slug":"` + slug + `","display_name":"Demo","summary":"Summary","description":"Description","category":"tools","tags":[],"icon":"","example_prompts":[],"risk_notes":"","origin_url":"https://skills.sh/demo","source_url":"","source_repository":"","featured":false,"sort_order":1,"catalog_source_priority":100}`)
	artifact := []byte(`{"changelog":"","manifest_name":"` + slug + `","manifest_description":"Demo","skill_md":"# Demo","package_sha256":"` + sha64("2") + `","byte_size":3,"unpacked_size":6,"file_count":1,"file_manifest":[{"path":"SKILL.md","byte_size":6,"sha256":"` + sha64("3") + `"}],"validation_report":{"valid":true,"errors":[],"warnings":[]},"transformed":false}`)
	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT status, request_config, cancel_requested_at, lease_owner, lease_expires_at.*WHERE id=\$1 FOR UPDATE`).
		WithArgs(runID).
		WillReturnRows(sqlmock.NewRows([]string{"status", "request_config", "cancel_requested_at", "lease_owner", "lease_expires_at"}).
			AddRow(service.SkillImportRunStatusPreparing, []byte(`{}`), nil, "worker-a", time.Now().UTC().Add(time.Hour)))
	mock.ExpectQuery(`(?s)SELECT COUNT\(\*\) FROM skill_import_run_items.*status IN \('queued','processing'\)`).
		WithArgs(runID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`(?s)SELECT id, source_id, namespace, external_id, rank, market_slug, status,.*staged_package_data, provenance.*FOR UPDATE`).
		WithArgs(runID, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "source_id", "namespace", "external_id", "rank", "market_slug",
			"status", "stage_action", "desired_skill", "valid", "skill_id", "version_id",
			"origin_url", "source_revision", "source_content_sha256", "package_sha256",
			"staged_artifact", "staged_package_data", "provenance",
		}).AddRow(itemID, int64(2), "skills.sh", "owner/repo/demo", 1, slug,
			service.SkillImportItemStatusReady, service.SkillImportStageActionCreate,
			desired, true, nil, nil, "https://skills.sh/demo", "rev", sha64("1"),
			sha64("2"), artifact, []byte("zip"), []byte(`{"source":"skills.sh"}`)))
	mock.ExpectQuery(`(?s)SELECT COUNT\(\*\) FROM skill_import_run_items.*NOT \(id=ANY\(\$2\)\)`).
		WithArgs(runID, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectExec(`SELECT pg_advisory_xact_lock\(hashtextextended\(\$1,0\)\)`).
		WithArgs(slug).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE skill_import_runs SET status='publishing'`).
		WithArgs(runID).WillReturnResult(sqlmock.NewResult(0, 1))
}

func expectStagedNewVersionPublishPrelude(mock sqlmock.Sqlmock, runID, itemID int64, slug string) {
	desired := []byte(`{"slug":"` + slug + `","display_name":"Demo","summary":"Summary","description":"Description","category":"tools","tags":[],"icon":"","example_prompts":[],"risk_notes":"","origin_url":"https://skills.sh/demo","source_url":"","source_repository":"","featured":false,"sort_order":1,"catalog_source_priority":100}`)
	artifact := []byte(`{"changelog":"","manifest_name":"` + slug + `","manifest_description":"Demo","skill_md":"# Demo","package_sha256":"` + sha64("2") + `","byte_size":3,"unpacked_size":6,"file_count":1,"file_manifest":[{"path":"SKILL.md","byte_size":6,"sha256":"` + sha64("3") + `"}],"validation_report":{"valid":true,"errors":[],"warnings":[]},"transformed":false}`)
	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT status, request_config, cancel_requested_at, lease_owner, lease_expires_at.*WHERE id=\$1 FOR UPDATE`).
		WithArgs(runID).WillReturnRows(sqlmock.NewRows([]string{"status", "request_config", "cancel_requested_at", "lease_owner", "lease_expires_at"}).
		AddRow(service.SkillImportRunStatusPreparing, []byte(`{}`), nil, "worker-a", time.Now().UTC().Add(time.Hour)))
	mock.ExpectQuery(`(?s)SELECT COUNT\(\*\) FROM skill_import_run_items.*status IN \('queued','processing'\)`).
		WithArgs(runID).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`(?s)SELECT id, source_id, namespace, external_id, rank, market_slug, status,.*staged_package_data, provenance.*FOR UPDATE`).
		WithArgs(runID, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "source_id", "namespace", "external_id", "rank", "market_slug",
			"status", "stage_action", "desired_skill", "valid", "skill_id", "version_id",
			"origin_url", "source_revision", "source_content_sha256", "package_sha256",
			"staged_artifact", "staged_package_data", "provenance",
		}).AddRow(itemID, int64(2), "skills.sh", "owner/repo/demo", 1, slug,
			service.SkillImportItemStatusReady, service.SkillImportStageActionNewVersion,
			desired, true, nil, nil, "https://skills.sh/demo", "rev", sha64("1"),
			sha64("2"), artifact, []byte("zip"), []byte(`{"source":"skills.sh"}`)))
	mock.ExpectQuery(`(?s)SELECT COUNT\(\*\) FROM skill_import_run_items.*NOT \(id=ANY\(\$2\)\)`).
		WithArgs(runID, sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectExec(`SELECT pg_advisory_xact_lock\(hashtextextended\(\$1,0\)\)`).
		WithArgs(slug).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE skill_import_runs SET status='publishing'`).
		WithArgs(runID).WillReturnResult(sqlmock.NewResult(0, 1))
}

func TestSkillImportRepositoryCompleteDiscoveryIsOneTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	now := time.Date(2026, time.August, 12, 2, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT source_id, status, COALESCE\(lease_owner,''\).*FOR UPDATE`).
		WithArgs(int64(12)).
		WillReturnRows(sqlmock.NewRows([]string{"source_id", "status", "lease_owner"}).
			AddRow(int64(2), service.SkillImportRunStatusDiscovering, "worker-a"))
	mock.ExpectQuery(`(?s)INSERT INTO skill_import_run_items.*RETURNING id, created_at, updated_at`).
		WithArgs(
			int64(12), int64(2), "inline-manifest", "demo", 1, "demo",
			service.SkillImportItemStatusQueued, "demo", "", "", "", "",
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), false,
			sqlmock.AnyArg(), sqlmock.AnyArg(), nil, nil, nil,
		).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(int64(80), now, now))
	mock.ExpectExec(`(?s)UPDATE skill_import_runs SET snapshot=\$3::jsonb.*status='preparing'.*discovered_count=\$5`).
		WithArgs(int64(12), "worker-a", `{"count":1}`, sha64("a"), 1).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	repo := &skillImportRepository{db: db}
	err = repo.CompleteDiscovery(context.Background(), 12, "worker-a", json.RawMessage(`{"count":1}`), sha64("a"), []service.SkillImportRunItem{{
		StableKey: service.SkillImportStableKey{Namespace: "inline-manifest", ExternalID: "demo"},
		Rank:      intPointer(1), MarketSlug: "demo", UpstreamName: "demo", OriginURL: "",
	}})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSkillImportRepositoryCancellationImmediatelyFinalizesUnownedState(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT status FROM skill_import_runs WHERE id=\$1 FOR UPDATE`).
		WithArgs(int64(24)).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(service.SkillImportRunStatusAwaitingReview))
	mock.ExpectExec(`(?s)UPDATE skill_import_runs SET.*status='cancelled'`).
		WithArgs(int64(24), int64(6), service.SkillImportRunStatusAwaitingReview).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)UPDATE skill_import_run_items SET status='cancelled'.*status NOT IN \('published','unchanged','skipped','cancelled'\)`).
		WithArgs(int64(24)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)UPDATE skill_import_run_items SET.*staged_package_data=NULL.*staged_artifact=staged_artifact - 'skill_md' - 'file_manifest'`).
		WithArgs(int64(24)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	repo := &skillImportRepository{db: db}
	changed, err := repo.RequestRunCancellation(context.Background(), 24, 6)
	require.NoError(t, err)
	require.True(t, changed)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSkillImportRepositoryBootstrapRequiresCurrentRunLease(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT source_id, trigger_type, status, cancel_requested_at,.*lease_owner, lease_expires_at.*WHERE id=\$1 FOR UPDATE`).
		WithArgs(int64(24)).
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "trigger_type", "status", "cancel_requested_at", "lease_owner", "lease_expires_at",
		}).AddRow(int64(2), service.SkillImportTriggerBootstrap,
			service.SkillImportRunStatusDiscovering, nil, "new-worker", time.Now().UTC().Add(time.Hour)))
	mock.ExpectRollback()

	repo := &skillImportRepository{db: db}
	_, err = repo.BootstrapOrigin(context.Background(), service.SkillImportBootstrapOriginInput{
		RunID: 24, RunWorkerID: "stale-worker",
		StableKey: service.SkillImportStableKey{SourceID: 2, Namespace: "skills.sh", ExternalID: "owner/repo/demo"},
	})
	require.ErrorIs(t, err, service.ErrSkillImportInvalidState)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSkillImportRepositoryCancellationImmediatelyFencesProcessingRun(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT status FROM skill_import_runs WHERE id=\$1 FOR UPDATE`).
		WithArgs(int64(25)).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(service.SkillImportRunStatusPreparing))
	mock.ExpectExec(`(?s)UPDATE skill_import_runs SET.*status='cancelled'`).
		WithArgs(int64(25), int64(6), service.SkillImportRunStatusPreparing).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)UPDATE skill_import_run_items SET status='cancelled'`).
		WithArgs(int64(25)).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(`(?s)UPDATE skill_import_run_items SET.*staged_package_data=NULL`).
		WithArgs(int64(25)).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	repo := &skillImportRepository{db: db}
	changed, err := repo.RequestRunCancellation(context.Background(), 25, 6)
	require.NoError(t, err)
	require.True(t, changed)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSkillImportRepositoryWorkerCancellationScrubsStagedBytesAtomically(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectExec(`(?s)UPDATE skill_import_runs SET status='cancelled'.*finished_at=NOW\(\).*WHERE id=\$1 AND lease_owner=\$2`).
		WithArgs(int64(26), "worker-a", "CANCELLED", "Cancelled by administrator").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)UPDATE skill_import_run_items SET status='cancelled'.*status NOT IN \('published','unchanged','skipped','cancelled'\)`).
		WithArgs(int64(26)).WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectExec(`(?s)UPDATE skill_import_run_items SET.*staged_package_data=NULL.*staged_artifact=staged_artifact - 'skill_md' - 'file_manifest'`).
		WithArgs(int64(26)).WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectCommit()

	repo := &skillImportRepository{db: db}
	err = repo.UpdateRunStatus(context.Background(), 26, "worker-a",
		service.SkillImportRunStatusCancelled, "CANCELLED", "Cancelled by administrator", nil)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSkillImportRepositoryRequestedCancellationOverridesRetryStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT cancel_requested_at FROM skill_import_runs.*WHERE id=\$1 AND lease_owner=\$2 FOR UPDATE`).
		WithArgs(int64(28), "worker-a").
		WillReturnRows(sqlmock.NewRows([]string{"cancel_requested_at"}).AddRow(time.Now().UTC()))
	mock.ExpectExec(`(?s)UPDATE skill_import_runs SET status='cancelled'.*WHERE id=\$1 AND lease_owner=\$2`).
		WithArgs(int64(28), "worker-a", "CANCELLED", "Cancelled by administrator").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)UPDATE skill_import_run_items SET status='cancelled'`).
		WithArgs(int64(28)).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(`(?s)UPDATE skill_import_run_items SET.*staged_package_data=NULL`).
		WithArgs(int64(28)).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	next := time.Now().UTC().Add(time.Minute)
	repo := &skillImportRepository{db: db}
	err = repo.UpdateRunStatus(context.Background(), 28, "worker-a",
		service.SkillImportRunStatusWaitingRetry, "TEMPORARY", "temporary failure", &next)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSkillImportRepositoryCancellationImmediatelyFinalizesExpiredLease(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT status FROM skill_import_runs WHERE id=\$1 FOR UPDATE`).
		WithArgs(int64(29)).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(service.SkillImportRunStatusPreparing))
	mock.ExpectExec(`(?s)UPDATE skill_import_runs SET.*status='cancelled'`).
		WithArgs(int64(29), int64(6), service.SkillImportRunStatusPreparing).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)UPDATE skill_import_run_items SET status='cancelled'`).
		WithArgs(int64(29)).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(`(?s)UPDATE skill_import_run_items SET.*staged_package_data=NULL`).
		WithArgs(int64(29)).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	repo := &skillImportRepository{db: db}
	changed, err := repo.RequestRunCancellation(context.Background(), 29, 6)
	require.NoError(t, err)
	require.True(t, changed)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSkillImportRepositoryPublishRejectsRequestedCancellation(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT status, request_config, cancel_requested_at, lease_owner, lease_expires_at.*WHERE id=\$1 FOR UPDATE`).
		WithArgs(int64(30)).
		WillReturnRows(sqlmock.NewRows([]string{"status", "request_config", "cancel_requested_at", "lease_owner", "lease_expires_at"}).
			AddRow(service.SkillImportRunStatusPreparing, []byte(`{}`), time.Now().UTC(), "worker-a", time.Now().UTC().Add(time.Hour)))
	mock.ExpectRollback()

	repo := &skillImportRepository{db: db}
	_, err = repo.PublishEligibleItems(context.Background(), 30, []int64{1}, "worker-a", nil)
	require.ErrorIs(t, err, service.ErrSkillImportInvalidState)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSkillImportRepositoryEligibleItemsIncludeUnchangedMetadataRefreshes(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectQuery(`(?s)SELECT id FROM skill_import_run_items.*status IN \('ready','unchanged'\).*validation_report`).
		WithArgs(int64(18)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(3)).AddRow(int64(4)))

	repo := &skillImportRepository{db: db}
	ids, err := repo.ListEligibleItemIDs(context.Background(), 18)
	require.NoError(t, err)
	require.Equal(t, []int64{3, 4}, ids)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSkillImportRepositoryListRunsRedactsInlineUploadInDatabase(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM skill_import_runs r`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))
	mock.ExpectQuery(`(?s)SELECT.*request_config #- '\{adapter_config,inline_data_base64\}'.*FROM skill_import_runs r.*LIMIT \$1 OFFSET \$2`).
		WithArgs(20, 0).
		WillReturnRows(sqlmock.NewRows([]string{"unused"}))

	repo := &skillImportRepository{db: db}
	runs, total, err := repo.ListRuns(context.Background(), service.SkillImportListFilter{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Empty(t, runs)
	require.Zero(t, total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSkillImportInlineUploadIsRetainedUntilRetryableRunFullySucceeds(t *testing.T) {
	require.False(t, purgeInlineSkillImportUpload(service.SkillImportRunStatusFailed))
	require.False(t, purgeInlineSkillImportUpload(service.SkillImportRunStatusPartialSucceeded))
	require.True(t, purgeInlineSkillImportUpload(service.SkillImportRunStatusSucceeded))
}

func skillImportItemRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "run_id", "source_id", "namespace", "external_id", "rank",
		"market_slug", "status", "upstream_name", "origin_url", "source_revision",
		"source_content_sha256", "package_sha256", "stage_action", "desired_skill", "source_payload",
		"validation_report", "provenance", "license_unverified", "excluded_files",
		"warnings", "skill_id", "version_id", "attempt_count", "next_attempt_at",
		"error_code", "error_message", "lease_owner", "lease_expires_at", "heartbeat_at",
		"started_at", "completed_at", "created_at", "updated_at",
	})
}

func importCountRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"requested_count", "discovered_count", "prepared_count", "created_count",
		"updated_count", "unchanged_count", "skipped_count", "blocked_count",
		"failed_count", "published_count",
	})
}

func sha64(character string) string { return strings.Repeat(character, 64) }

func intPointer(value int) *int { return &value }
