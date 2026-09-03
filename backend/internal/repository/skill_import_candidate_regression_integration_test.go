//go:build integration

package repository

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	importapp "github.com/Wei-Shaw/sub2api/internal/modules/skillimport/application"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestCandidateSkillImportBootstrapSnapshotContractOnPostgreSQL18(t *testing.T) {
	ctx := context.Background()
	db := openIsolatedMigrationIntegrationDB(t, "sub2api_candidate_bootstrap")
	require.NoError(t, ApplyMigrations(ctx, db))

	var serverVersion int
	require.NoError(t, db.QueryRowContext(ctx, `
SELECT current_setting('server_version_num')::INTEGER`).Scan(&serverVersion))
	require.GreaterOrEqual(t, serverVersion, 180000)
	require.Less(t, serverVersion, 190000)

	repo := NewSkillImportRepository(db)
	stamp := formatImportTestStamp(time.Now().UTC().UnixNano())
	slug := "candidate-bootstrap-" + stamp
	externalID := "existing-" + stamp
	packageData := candidateBootstrapPackage(t, slug, externalID)
	validated, err := service.ValidateSkillArchive(packageData, slug)
	require.NoError(t, err)
	fileManifest, err := json.Marshal(validated.FileManifest)
	require.NoError(t, err)
	validationReport, err := json.Marshal(validated.ValidationReport)
	require.NoError(t, err)

	var skillID int64
	err = db.QueryRowContext(ctx, `
INSERT INTO skills (
  slug, display_name, summary, description, category, tags,
  example_prompts, risk_notes, source_url, source_repository, status, sort_order
) VALUES (
  $1,$1,'Candidate bootstrap fixture','Candidate bootstrap fixture',
  'integration','[]'::jsonb,'[]'::jsonb,'',
  'https://github.com/integration/skills','integration/skills','draft',7
)
RETURNING id`, slug).Scan(&skillID)
	require.NoError(t, err)

	var versionID int64
	err = db.QueryRowContext(ctx, `
INSERT INTO skill_versions (
  skill_id, version, changelog, manifest_name, manifest_description, skill_md,
  package_data, sha256, byte_size, unpacked_size, file_count, file_manifest,
  validation_report, released_at
) VALUES (
  $1,'1.0.0','',$2,$3,$4,$5,$6,$7,$8,$9,$10::jsonb,$11::jsonb,NOW()
)
RETURNING id`, skillID, validated.ManifestName, validated.ManifestDescription,
		validated.SkillMD, validated.PackageData, validated.SHA256, len(validated.PackageData),
		validated.UnpackedSize, len(validated.FileManifest), fileManifest, validationReport,
	).Scan(&versionID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `
UPDATE skills SET status='published', current_version_id=$2, published_at=NOW()
WHERE id=$1`, skillID, versionID)
	require.NoError(t, err)

	var sourceID, seededRunID int64
	err = db.QueryRowContext(ctx, `
SELECT source.id, bootstrap_run.id
FROM skill_import_sources AS source
JOIN skill_import_runs AS bootstrap_run ON bootstrap_run.source_id=source.id
WHERE source.adapter='skills_sh'
  AND source.namespace='skills.sh'
  AND bootstrap_run.trigger_type='bootstrap'`).Scan(&sourceID, &seededRunID)
	require.NoError(t, err)

	cfg := &config.Config{SkillImport: config.SkillImportConfig{
		Enabled: true, WorkerEnabled: true, PollIntervalSeconds: 1,
		LeaseTTLSeconds: 30, MaxAttempts: 1, MaxRunDurationHours: 1,
	}}
	worker := importapp.NewWorkerRuntime(
		importapp.NewService(repo, importapp.NewAdapterRegistry(), cfg), nil, cfg,
	)
	require.NoError(t, worker.Start(ctx))
	workerStopped := false
	t.Cleanup(func() {
		if workerStopped {
			return
		}
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = worker.Stop(stopCtx)
	})

	assertBootstrapSucceeded := func(runID int64) {
		t.Helper()
		var run *service.SkillImportRun
		require.Eventually(t, func() bool {
			current, getErr := repo.GetRun(ctx, runID)
			if getErr != nil {
				return false
			}
			run = current
			return current.Status == service.SkillImportRunStatusSucceeded ||
				current.Status == service.SkillImportRunStatusFailed ||
				current.Status == service.SkillImportRunStatusPartialSucceeded
		}, 20*time.Second, 100*time.Millisecond)
		require.Equal(t, service.SkillImportRunStatusSucceeded, run.Status,
			"bootstrap error: %s", run.LastErrorMessage)
		require.Nil(t, run.LeaseOwner)
		require.Nil(t, run.LeaseExpiresAt)
		require.Nil(t, run.NextAttemptAt)
	}

	assertBootstrapSucceeded(seededRunID)

	cancelledRun := &service.SkillImportRun{
		SourceID: sourceID, TriggerType: service.SkillImportTriggerManual,
		Mode: service.SkillImportModeReview, Status: service.SkillImportRunStatusCancelled,
		RequestConfig: json.RawMessage(`{"history":"cancelled"}`),
		Snapshot:      json.RawMessage(`{"snapshot":"cancelled-history"}`),
	}
	require.NoError(t, repo.CreateRun(ctx, cancelledRun, ""))
	_, err = db.ExecContext(ctx, `
UPDATE skill_import_runs SET finished_at=NOW() WHERE id=$1`, cancelledRun.ID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `
INSERT INTO skill_import_run_items (
  run_id, source_id, namespace, external_id, market_slug, status,
  upstream_name, origin_url, completed_at
) VALUES ($1,$2,'integration/history',$3,$4,'cancelled','Cancelled history','',NOW())`,
		cancelledRun.ID, sourceID, "cancelled-"+stamp, "cancelled-"+stamp)
	require.NoError(t, err)
	require.NoError(t, repo.AppendEvent(ctx, &service.SkillImportEvent{
		RunID: cancelledRun.ID, Level: service.SkillImportEventLevelInfo,
		EventType: "history_cancelled", Message: "Preserved cancelled history",
		Payload: json.RawMessage(`{"fixture":true}`),
	}))

	reviewRun := &service.SkillImportRun{
		SourceID: sourceID, TriggerType: service.SkillImportTriggerManual,
		Mode: service.SkillImportModeReview, Status: service.SkillImportRunStatusAwaitingReview,
		RequestConfig:  json.RawMessage(`{"history":"awaiting_review"}`),
		Snapshot:       json.RawMessage(`{"snapshot":"review-history"}`),
		SnapshotSHA256: strings.Repeat("c", 64),
	}
	require.NoError(t, repo.CreateRun(ctx, reviewRun, ""))

	var reviewItemID int64
	err = db.QueryRowContext(ctx, `
INSERT INTO skill_import_run_items (
  run_id, source_id, namespace, external_id, rank, market_slug, status,
  stage_action, upstream_name, origin_url, source_revision,
  source_content_sha256, package_sha256, staged_artifact,
  staged_package_data, desired_skill, source_payload, validation_report,
  provenance, license_unverified, excluded_files, warnings, skill_id,
  version_id, attempt_count, next_attempt_at, lease_owner, lease_expires_at,
  heartbeat_at, error_code, error_message, started_at, completed_at
)
SELECT
  $2, source_id, namespace, external_id, rank, market_slug, status,
  stage_action, upstream_name, origin_url, source_revision,
  source_content_sha256, package_sha256, staged_artifact,
  staged_package_data, desired_skill, source_payload, validation_report,
  provenance, license_unverified, excluded_files, warnings, skill_id,
  version_id, 0, NULL, NULL, NULL, NULL, '', '', started_at, completed_at
FROM skill_import_run_items
WHERE run_id=$1
RETURNING id`, seededRunID, reviewRun.ID).Scan(&reviewItemID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `
INSERT INTO skill_version_origins (
  version_id, origin_id, run_item_id, source_revision,
  source_content_sha256, transformed, provenance
)
SELECT version_origin.version_id, version_origin.origin_id, $2,
  version_origin.source_revision, version_origin.source_content_sha256,
  version_origin.transformed, version_origin.provenance
FROM skill_version_origins AS version_origin
JOIN skill_import_run_items AS bootstrap_item
  ON bootstrap_item.id=version_origin.run_item_id
WHERE bootstrap_item.run_id=$1`, seededRunID, reviewItemID)
	require.NoError(t, err)
	require.NoError(t, repo.AppendEvent(ctx, &service.SkillImportEvent{
		RunID: reviewRun.ID, RunItemID: &reviewItemID,
		Level: service.SkillImportEventLevelInfo, EventType: "review_preserved",
		Message: "Preserved awaiting-review history",
		Payload: json.RawMessage(`{"fixture":true}`),
	}))

	blockedRun := &service.SkillImportRun{
		SourceID: sourceID, TriggerType: service.SkillImportTriggerBootstrap,
		Mode: service.SkillImportModeDryRun, Status: service.SkillImportRunStatusQueued,
		RequestConfig: json.RawMessage(`{"bootstrap_existing":true}`),
		Snapshot:      json.RawMessage(`{}`),
	}
	err = repo.CreateRun(ctx, blockedRun, "")
	require.ErrorIs(t, err, service.ErrSkillImportConflict,
		"the awaiting-review run must occupy the active-source unique slot")

	claimed, err := repo.ClaimNextRun(ctx, "history-claim-probe", time.Now().UTC(), time.Now().UTC().Add(time.Minute))
	require.NoError(t, err)
	require.Nil(t, claimed, "terminal/review history must not be claimed by the worker")

	var reviewBefore string
	require.NoError(t, db.QueryRowContext(ctx, `
SELECT to_jsonb(review_run)::TEXT
FROM skill_import_runs AS review_run
WHERE review_run.id=$1`, reviewRun.ID).Scan(&reviewBefore))

	validationConn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer func() { _ = validationConn.Close() }()
	setCandidateSessionValue(t, validationConn, "candidate.review_run_id", reviewRun.ID)
	suspendSQL := readCandidateSQL(t, "suspend_skill_import_review.sql")
	_, err = validationConn.ExecContext(ctx, suspendSQL)
	require.NoError(t, err, "execute the exact candidate review-suspension SQL")

	var suspendedStatus string
	require.NoError(t, db.QueryRowContext(ctx, `
SELECT status FROM skill_import_runs WHERE id=$1`, reviewRun.ID).Scan(&suspendedStatus))
	require.Equal(t, service.SkillImportRunStatusCancelled, suspendedStatus)

	createBootstrapRun := func() int64 {
		t.Helper()
		run := &service.SkillImportRun{
			SourceID: sourceID, TriggerType: service.SkillImportTriggerBootstrap,
			Mode: service.SkillImportModeDryRun, Status: service.SkillImportRunStatusQueued,
			RequestConfig: json.RawMessage(`{"bootstrap_existing":true}`),
			Snapshot:      json.RawMessage(`{}`),
		}
		require.NoError(t, repo.CreateRun(ctx, run, ""))
		assertBootstrapSucceeded(run.ID)
		return run.ID
	}
	bootstrapRunAID := createBootstrapRun()
	bootstrapRunBID := createBootstrapRun()

	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	require.NoError(t, worker.Stop(stopCtx))
	cancel()
	workerStopped = true

	setCandidateSessionValue(t, validationConn, "candidate.bootstrap_run_a_id", bootstrapRunAID)
	setCandidateSessionValue(t, validationConn, "candidate.bootstrap_run_b_id", bootstrapRunBID)
	restoreSQL := readCandidateSQL(t, "restore_skill_import_review.sql")
	_, err = validationConn.ExecContext(ctx, restoreSQL)
	require.NoError(t, err, "execute the exact fixed candidate review-restore SQL")

	var reviewAfter string
	require.NoError(t, db.QueryRowContext(ctx, `
SELECT to_jsonb(review_run)::TEXT
FROM skill_import_runs AS review_run
WHERE review_run.id=$1`, reviewRun.ID).Scan(&reviewAfter))
	require.JSONEq(t, reviewBefore, reviewAfter)

	var validationSchemaMissing bool
	require.NoError(t, db.QueryRowContext(ctx, `
SELECT to_regnamespace('candidate_validation') IS NULL`).Scan(&validationSchemaMissing))
	require.True(t, validationSchemaMissing, "candidate SQL must clean its snapshot schema")

	var runs, items, events, origins, versionOrigins, orphanedVersionOrigins int
	require.NoError(t, db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM skill_import_runs WHERE source_id=$1`, sourceID).Scan(&runs))
	require.NoError(t, db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM skill_import_run_items WHERE source_id=$1`, sourceID).Scan(&items))
	require.NoError(t, db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM skill_import_events AS event
JOIN skill_import_runs AS run ON run.id=event.run_id
WHERE run.source_id=$1`, sourceID).Scan(&events))
	require.NoError(t, db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM skill_origins WHERE source_id=$1`, sourceID).Scan(&origins))
	require.NoError(t, db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM skill_version_origins AS version_origin
JOIN skill_origins AS origin ON origin.id=version_origin.origin_id
WHERE origin.source_id=$1`, sourceID).Scan(&versionOrigins))
	require.NoError(t, db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM skill_version_origins AS version_origin
LEFT JOIN skill_import_run_items AS run_item
  ON run_item.id=version_origin.run_item_id
LEFT JOIN skill_origins AS origin ON origin.id=version_origin.origin_id
LEFT JOIN skill_versions AS skill_version ON skill_version.id=version_origin.version_id
WHERE run_item.id IS NULL OR origin.id IS NULL OR skill_version.id IS NULL
  OR run_item.version_id<>version_origin.version_id
  OR run_item.skill_id<>origin.skill_id`).Scan(&orphanedVersionOrigins))
	require.Equal(t, 5, runs)
	require.Equal(t, 5, items)
	require.Equal(t, 5, events)
	require.Equal(t, 1, origins, "repeated bootstrap must reuse one durable origin")
	require.Equal(t, 4, versionOrigins,
		"seeded, preserved-review and two repeated bootstraps keep separate provenance")
	require.Zero(t, orphanedVersionOrigins)

	for _, runID := range []int64{seededRunID, bootstrapRunAID, bootstrapRunBID} {
		var runItems, completedEvents int
		require.NoError(t, db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM skill_import_run_items WHERE run_id=$1`, runID).Scan(&runItems))
		require.NoError(t, db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM skill_import_events
WHERE run_id=$1 AND event_type='bootstrap_completed'`, runID).Scan(&completedEvents))
		require.Equal(t, 1, runItems)
		require.Equal(t, 1, completedEvents)
	}

	for _, historicalRunID := range []int64{reviewRun.ID, cancelledRun.ID} {
		var status string
		var attemptCount int
		var leaseOwner sql.NullString
		require.NoError(t, db.QueryRowContext(ctx, `
SELECT status, attempt_count, lease_owner
FROM skill_import_runs WHERE id=$1`, historicalRunID).Scan(
			&status, &attemptCount, &leaseOwner,
		))
		require.Zero(t, attemptCount)
		require.False(t, leaseOwner.Valid)
		if historicalRunID == reviewRun.ID {
			require.Equal(t, service.SkillImportRunStatusAwaitingReview, status)
		} else {
			require.Equal(t, service.SkillImportRunStatusCancelled, status)
		}
	}
	claimed, err = repo.ClaimNextRun(ctx, "post-restore-claim-probe", time.Now().UTC(), time.Now().UTC().Add(time.Minute))
	require.NoError(t, err)
	require.Nil(t, claimed, "restored history must not create worker-claimable work")

	fingerprintSQL := readCandidateSQL(t, "skill_content_fingerprint.sql")
	contentFingerprint := func() string {
		t.Helper()
		var fingerprint string
		require.NoError(t, db.QueryRowContext(ctx, fingerprintSQL).Scan(&fingerprint))
		require.Len(t, fingerprint, 64)
		return fingerprint
	}

	beforeStars := contentFingerprint()
	marketRepo := NewSkillMarketRepository(db)
	now := time.Now().UTC().Truncate(time.Microsecond)
	targets, err := marketRepo.ClaimRepositoryStarsRefresh(ctx, 1, now.Add(5*time.Minute))
	require.NoError(t, err)
	require.Len(t, targets, 1)
	require.Equal(t, skillID, targets[0].ID)
	require.Equal(t, beforeStars, contentFingerprint(),
		"claiming stars work may change only the excluded refresh projection")
	require.NoError(t, marketRepo.CompleteRepositoryStarsRefresh(
		ctx, skillID, "https://github.com/integration/skills", 4242,
		now, now.Add(6*time.Hour),
	))
	require.Equal(t, beforeStars, contentFingerprint(),
		"stars completion may change only the three explicit runtime projection fields")

	_, err = db.ExecContext(ctx, `UPDATE skills SET summary=$2 WHERE id=$1`, skillID, "authored business change")
	require.NoError(t, err)
	require.NotEqual(t, beforeStars, contentFingerprint(),
		"changing any authored business field must fail the content fingerprint")
}

func candidateBootstrapPackage(t *testing.T, slug, externalID string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	entry, err := writer.Create(slug + "/SKILL.md")
	require.NoError(t, err)
	_, err = fmt.Fprintf(entry, `---
name: %s
description: Candidate PostgreSQL 18 bootstrap regression fixture.
metadata:
  original_source: integration/skills
  skills_sh_id: %s
  snapshot_hash: %s
---
Exercise durable candidate bootstrap validation.
`, slug, externalID, strings.Repeat("a", 64))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	return buffer.Bytes()
}

func readCandidateSQL(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("..", "..", "..", "deploy", "candidate", "sql", name)
	content, err := os.ReadFile(path)
	require.NoError(t, err, "read candidate SQL %s", name)
	require.NotEmpty(t, strings.TrimSpace(string(content)))
	return string(content)
}

func setCandidateSessionValue(t *testing.T, conn *sql.Conn, name string, value int64) {
	t.Helper()
	_, err := conn.ExecContext(context.Background(), `SELECT set_config($1,$2,FALSE)`,
		name, strconv.FormatInt(value, 10))
	require.NoError(t, err)
}
