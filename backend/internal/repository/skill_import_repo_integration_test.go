//go:build integration

package repository

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	importapp "github.com/Wei-Shaw/sub2api/internal/modules/skillimport/application"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSkillImportRepositoryPersistsRunAndReclaimsExpiredLease(t *testing.T) {
	ctx := context.Background()
	repo := NewSkillImportRepository(integrationDB)
	stamp := time.Now().UTC().UnixNano()
	source := &service.SkillImportSource{
		Name: "Integration import source", Adapter: "manifest",
		Namespace: "integration-" + time.Now().UTC().Format("20060102150405") + "-" + formatImportTestStamp(stamp),
		BaseURL:   "", SourceConfig: json.RawMessage(`{"upload_only":true}`), Enabled: true,
	}
	require.NoError(t, repo.CreateSource(ctx, source))
	run := &service.SkillImportRun{
		SourceID: source.ID, TriggerType: service.SkillImportTriggerManual,
		Mode: service.SkillImportModeDryRun, Status: service.SkillImportRunStatusQueued,
		RequestConfig: json.RawMessage(`{"adapter_config":{"inline_data_base64":"emlw","inline_data_sha256":"evidence"}}`),
		Snapshot:      json.RawMessage(`{}`),
	}
	require.NoError(t, repo.CreateRun(ctx, run, ""))

	now := time.Now().UTC()
	_, err := integrationDB.ExecContext(ctx, `
UPDATE skill_import_runs SET next_attempt_at=$2 WHERE id=$1`, run.ID, now.Add(-time.Hour))
	require.NoError(t, err)
	claimed, err := repo.ClaimNextRun(ctx, "integration-worker", now, now.Add(time.Minute))
	require.NoError(t, err)
	require.NotNil(t, claimed)
	require.Equal(t, run.ID, claimed.ID)
	require.Equal(t, service.SkillImportRunStatusDiscovering, claimed.Status)

	_, err = integrationDB.ExecContext(ctx, `
UPDATE skill_import_runs SET lease_expires_at=$2, next_attempt_at=$3 WHERE id=$1`,
		run.ID, now.Add(-time.Minute), now.Add(-time.Hour))
	require.NoError(t, err)
	reclaimed, err := repo.ClaimNextRun(ctx, "integration-worker-2", now, now.Add(time.Minute))
	require.NoError(t, err)
	require.NotNil(t, reclaimed)
	require.Equal(t, run.ID, reclaimed.ID)
}

func TestSkillImportRepositoryInlineManifestAllowsEmptyOriginURL(t *testing.T) {
	ctx := context.Background()
	repo := NewSkillImportRepository(integrationDB)
	stamp := formatImportTestStamp(time.Now().UTC().UnixNano())
	marketSlug := "inline-" + stamp
	source := &service.SkillImportSource{
		Name: "Inline empty origin " + stamp, Adapter: "manifest",
		Namespace: "inline-empty-" + stamp, BaseURL: "",
		SourceConfig: json.RawMessage(`{"upload_only":true}`), Enabled: true,
	}
	require.NoError(t, repo.CreateSource(ctx, source))

	run := &service.SkillImportRun{
		SourceID: source.ID, TriggerType: service.SkillImportTriggerManual,
		Mode: service.SkillImportModeDryRun, Status: service.SkillImportRunStatusQueued,
		RequestConfig: json.RawMessage(`{}`), Snapshot: json.RawMessage(`{}`),
	}
	require.NoError(t, repo.CreateRun(ctx, run, ""))
	_, err := integrationDB.ExecContext(ctx, `
UPDATE skill_import_runs SET status='discovering', lease_owner='inline-worker',
  lease_expires_at=NOW()+INTERVAL '5 minutes'
WHERE id=$1`, run.ID)
	require.NoError(t, err)

	items := []service.SkillImportRunItem{{
		StableKey: service.SkillImportStableKey{
			Namespace: source.Namespace, ExternalID: "inline-demo",
		},
		MarketSlug: marketSlug, UpstreamName: "Inline Demo", OriginURL: "",
	}}
	require.NoError(t, repo.CompleteDiscovery(
		ctx, run.ID, "inline-worker", json.RawMessage(`{"count":1}`),
		strings.Repeat("a", 64), items,
	))
	item, err := repo.GetRunItem(ctx, items[0].ID)
	require.NoError(t, err)
	require.Empty(t, item.OriginURL)
	claimed, err := repo.ClaimNextRunItem(
		ctx, run.ID, "inline-worker", "inline-item-claim", time.Now().UTC(), time.Now().UTC().Add(time.Minute),
	)
	require.NoError(t, err)
	require.NotNil(t, claimed)
	stage, err := repo.StagePreparedItem(ctx, service.SkillImportStagePreparedInput{
		RunID: run.ID, RunItemID: claimed.ID,
		RunWorkerID: "inline-worker", ItemLeaseOwner: "inline-item-claim",
		StableKey: claimed.StableKey,
		DesiredSkill: service.SkillImportDesiredSkill{
			Slug: marketSlug, DisplayName: "Inline Demo", Summary: "Inline summary",
			Description: "Inline description", Category: "tools", Tags: []string{},
			ExamplePrompts: []string{}, OriginURL: "", SortOrder: 1,
		},
		OriginURL: "",
		Artifact: service.SkillImportPreparedArtifact{
			ManifestName: marketSlug, ManifestDescription: "Inline description",
			SkillMD: "# Inline Demo", PackageData: []byte("zip"),
			PackageSHA256: strings.Repeat("b", 64), ByteSize: 3,
			UnpackedSize: 13, FileCount: 1,
			FileManifest: []service.SkillArchiveFile{{
				Path: marketSlug + "/SKILL.md", ByteSize: 13, SHA256: strings.Repeat("c", 64),
			}},
			ValidationReport: service.SkillValidationReport{
				Valid: true, Errors: []service.SkillValidationIssue{}, Warnings: []service.SkillValidationIssue{},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, service.SkillImportStageActionCreate, stage.Action)

	var marketRows int
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
SELECT COUNT(*) FROM skills WHERE slug=$1`, marketSlug).Scan(&marketRows))
	require.Zero(t, marketRows, "staging must not create a draft Skill")
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
SELECT COUNT(*) FROM skill_origins
WHERE source_id=$1 AND namespace=$2 AND external_id=$3`, source.ID,
		source.Namespace, "inline-demo").Scan(&marketRows))
	require.Zero(t, marketRows, "staging must not create an origin")

	published, err := repo.PublishEligibleItems(ctx, run.ID, []int64{claimed.ID}, "inline-worker", nil)
	require.NoError(t, err)
	require.Len(t, published.Items, 1)
	var persistedOrigin, persistedInline string
	var stagedBytes []byte
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
SELECT s.origin_url, i.staged_package_data,
  COALESCE(r.request_config #>> '{adapter_config,inline_data_base64}','')
FROM skills s
JOIN skill_import_run_items i ON i.skill_id=s.id
JOIN skill_import_runs r ON r.id=i.run_id
WHERE i.id=$1`, claimed.ID).Scan(&persistedOrigin, &stagedBytes, &persistedInline))
	require.Empty(t, persistedOrigin)
	require.Empty(t, stagedBytes, "successful publication clears retry ZIP bytes")
	require.Empty(t, persistedInline, "successful publication clears inline upload bytes")

	_, err = integrationDB.ExecContext(ctx, `
UPDATE skill_import_run_items SET origin_url='http://insecure.example/skill'
WHERE id=$1`, claimed.ID)
	require.Error(t, err, "non-empty item origins must remain HTTPS")
	_, err = integrationDB.ExecContext(ctx, `
UPDATE skill_import_sources SET base_url='http://insecure.example'
WHERE id=$1`, source.ID)
	require.Error(t, err, "non-empty source base URLs must remain HTTPS")
}

func TestSkillImportRepositoryCancellationScrubsStagedAndInlineBytes(t *testing.T) {
	ctx := context.Background()
	repo := NewSkillImportRepository(integrationDB)
	stamp := formatImportTestStamp(time.Now().UTC().UnixNano())
	marketSlug := "cancel-" + stamp
	source := &service.SkillImportSource{
		Name: "Cancellation scrub " + stamp, Adapter: "manifest",
		Namespace: "cancel-scrub-" + stamp, BaseURL: "",
		SourceConfig: json.RawMessage(`{"upload_only":true}`), Enabled: true,
	}
	require.NoError(t, repo.CreateSource(ctx, source))
	run := &service.SkillImportRun{
		SourceID: source.ID, TriggerType: service.SkillImportTriggerManual,
		Mode: service.SkillImportModeReview, Status: service.SkillImportRunStatusReady,
		RequestConfig: json.RawMessage(`{"adapter_config":{"inline_data_base64":"emlw","inline_data_sha256":"evidence"}}`),
		Snapshot:      json.RawMessage(`{}`),
	}
	require.NoError(t, repo.CreateRun(ctx, run, ""))
	actor := mustCreateUser(t, testEntClient(t), &service.User{
		Email: "skill-import-cancel-" + stamp + "@example.com",
		Role:  service.RoleAdmin,
	})
	var itemID int64
	err := integrationDB.QueryRowContext(ctx, `
INSERT INTO skill_import_run_items (
  run_id, source_id, namespace, external_id, market_slug, status,
  stage_action, origin_url, package_sha256, staged_artifact,
  staged_package_data, desired_skill, validation_report
) VALUES (
  $1,$2,$3,'cancel-demo',$4::text,'ready','create','',$5::text,
  jsonb_build_object(
    'manifest_name',$4::text,'manifest_description','Cancellation test',
    'skill_md','# Cancellation test','package_sha256',$5::text,
    'byte_size',3,'unpacked_size',20,'file_count',1,
    'file_manifest',jsonb_build_array(jsonb_build_object('path','SKILL.md')),
    'validation_report',jsonb_build_object('valid',TRUE,'errors',jsonb_build_array(),'warnings',jsonb_build_array())
  ),
  $6,jsonb_build_object('slug',$4::text),
  jsonb_build_object('valid',TRUE,'errors',jsonb_build_array(),'warnings',jsonb_build_array())
)
RETURNING id`, run.ID, source.ID, source.Namespace, marketSlug,
		strings.Repeat("d", 64), []byte("zip")).Scan(&itemID)
	require.NoError(t, err)

	changed, err := repo.RequestRunCancellation(ctx, run.ID, actor.ID)
	require.NoError(t, err)
	require.True(t, changed)
	var runStatus, itemStatus, persistedInline string
	var stagedBytes []byte
	var hasSkillMD, hasFileManifest bool
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
SELECT r.status, i.status, i.staged_package_data,
  i.staged_artifact ? 'skill_md', i.staged_artifact ? 'file_manifest',
  COALESCE(r.request_config #>> '{adapter_config,inline_data_base64}','')
FROM skill_import_runs r
JOIN skill_import_run_items i ON i.run_id=r.id
WHERE r.id=$1 AND i.id=$2`, run.ID, itemID).Scan(
		&runStatus, &itemStatus, &stagedBytes, &hasSkillMD, &hasFileManifest, &persistedInline,
	))
	require.Equal(t, service.SkillImportRunStatusCancelled, runStatus)
	require.Equal(t, service.SkillImportItemStatusCancelled, itemStatus)
	require.Empty(t, stagedBytes)
	require.False(t, hasSkillMD)
	require.False(t, hasFileManifest)
	require.Empty(t, persistedInline)
	var marketRows int
	require.NoError(t, integrationDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM skills WHERE slug=$1`, marketSlug).Scan(&marketRows))
	require.Zero(t, marketRows, "cancellation must not consume a market slug")
}

func TestSkillImportRepositoryCompletesItemWithFencedRunAndItemLeases(t *testing.T) {
	ctx := context.Background()
	repo := NewSkillImportRepository(integrationDB)
	stamp := formatImportTestStamp(time.Now().UTC().UnixNano())
	source := &service.SkillImportSource{
		Name: "Fenced item " + stamp, Adapter: "manifest",
		Namespace: "fenced-item-" + stamp, BaseURL: "",
		SourceConfig: json.RawMessage(`{"upload_only":true}`), Enabled: true,
	}
	require.NoError(t, repo.CreateSource(ctx, source))
	run := &service.SkillImportRun{
		SourceID: source.ID, TriggerType: service.SkillImportTriggerManual,
		Mode: service.SkillImportModeDryRun, Status: service.SkillImportRunStatusQueued,
		RequestConfig: json.RawMessage(`{}`), Snapshot: json.RawMessage(`{}`),
	}
	require.NoError(t, repo.CreateRun(ctx, run, ""))
	now := time.Now().UTC()
	_, err := integrationDB.ExecContext(ctx, `
UPDATE skill_import_runs SET next_attempt_at=$2 WHERE id=$1`, run.ID, now.Add(-time.Hour))
	require.NoError(t, err)
	claimedRun, err := repo.ClaimNextRun(ctx, "fenced-run-worker", now, now.Add(time.Minute))
	require.NoError(t, err)
	require.Equal(t, run.ID, claimedRun.ID)
	item := service.SkillImportRunItem{
		StableKey:  service.SkillImportStableKey{Namespace: source.Namespace, ExternalID: "demo"},
		MarketSlug: "fenced-" + stamp, Status: service.SkillImportItemStatusQueued,
		UpstreamName: "Demo", OriginURL: "", SourcePayload: json.RawMessage(`{}`),
	}
	require.NoError(t, repo.CompleteDiscovery(
		ctx, run.ID, "fenced-run-worker", json.RawMessage(`{"count":1}`),
		strings.Repeat("d", 64), []service.SkillImportRunItem{item},
	))
	claimedItem, err := repo.ClaimNextRunItem(
		ctx, run.ID, "fenced-run-worker", "fenced-item-claim", now, now.Add(time.Minute),
	)
	require.NoError(t, err)
	require.NotNil(t, claimedItem)
	require.NoError(t, repo.HeartbeatRunItem(
		ctx, claimedItem.ID, "fenced-item-claim", "fenced-run-worker", now.Add(2*time.Minute),
	))
	require.NoError(t, repo.CompleteRunItem(
		ctx, claimedItem.ID, "fenced-item-claim", "fenced-run-worker",
		service.SkillImportRunItemPatch{
			Status: service.SkillImportItemStatusSkipped, MarketSlug: claimedItem.MarketSlug,
			DesiredSkill:     service.SkillImportDesiredSkill{Slug: claimedItem.MarketSlug},
			ValidationReport: service.SkillValidationReport{Valid: true},
		},
	))
	completed, err := repo.GetRunItem(ctx, claimedItem.ID)
	require.NoError(t, err)
	require.Equal(t, service.SkillImportItemStatusSkipped, completed.Status)
	require.Nil(t, completed.LeaseOwner)
}

func TestSkillImportRepositoryMigrationBootstrapSucceedsOnPostgreSQL18(t *testing.T) {
	ctx := context.Background()
	db := openIsolatedMigrationIntegrationDB(t, "sub2api_skill_import_bootstrap")
	require.NoError(t, ApplyMigrations(ctx, db))
	repo := NewSkillImportRepository(db)

	stamp := formatImportTestStamp(time.Now().UTC().UnixNano())
	slug := "bootstrap-" + stamp
	externalID := "existing-" + stamp
	packageData := skillImportBootstrapPackage(t, slug, `---
name: `+slug+`
description: PostgreSQL 18 bootstrap integration fixture.
metadata:
  original_source: integration/skills
  skills_sh_id: `+externalID+`
  snapshot_hash: `+strings.Repeat("a", 64)+`
---
Use this fixture to verify durable bootstrap provenance.
`)
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
  example_prompts, risk_notes, status, sort_order
) VALUES (
  $1,$1,'Bootstrap fixture','Bootstrap fixture','integration','[]'::jsonb,
  '[]'::jsonb,'','draft',7
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
	var seededTrigger, seededMode, seededStatus, seededIdempotencyHash string
	var seededRequestConfig []byte
	err = db.QueryRowContext(ctx, `
SELECT source.id, run.id, run.trigger_type, run.mode, run.status,
  run.request_config, run.idempotency_key_hash
FROM skill_import_sources source
JOIN skill_import_runs run ON run.source_id=source.id
WHERE source.adapter='skills_sh' AND source.namespace='skills.sh'
  AND run.trigger_type='bootstrap'`).Scan(
		&sourceID, &seededRunID, &seededTrigger, &seededMode, &seededStatus,
		&seededRequestConfig, &seededIdempotencyHash,
	)
	require.NoError(t, err)
	require.Equal(t, service.SkillImportTriggerBootstrap, seededTrigger)
	require.Equal(t, service.SkillImportModeDryRun, seededMode)
	require.Equal(t, service.SkillImportRunStatusQueued, seededStatus)
	require.JSONEq(t, `{"bootstrap_existing":true}`, string(seededRequestConfig))
	require.Equal(t, "b00757a9e5d127fe24a83c84f5a48a663f18895d28b42b10cf0b769891c5f241", seededIdempotencyHash)

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
		var failedItems, failedEvents int
		require.NoError(t, db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM skill_import_run_items
WHERE run_id=$1 AND status='failed'`, runID).Scan(&failedItems))
		require.NoError(t, db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM skill_import_events
WHERE run_id=$1 AND level='error'`, runID).Scan(&failedEvents))
		require.Zero(t, failedItems)
		require.Zero(t, failedEvents)
	}

	assertBootstrapSucceeded(seededRunID)
	var stagedName, desiredSlug, desiredOrigin string
	require.NoError(t, db.QueryRowContext(ctx, `
SELECT staged_artifact->>'manifest_name', desired_skill->>'slug',
  desired_skill->>'origin_url'
FROM skill_import_run_items WHERE run_id=$1 AND skill_id=$2`, seededRunID, skillID).Scan(
		&stagedName, &desiredSlug, &desiredOrigin,
	))
	require.Equal(t, slug, stagedName)
	require.Equal(t, slug, desiredSlug)
	require.Equal(t, "https://skills.sh/integration/skills/"+externalID, desiredOrigin)

	secondRun := &service.SkillImportRun{
		SourceID: sourceID, TriggerType: service.SkillImportTriggerBootstrap,
		Mode: service.SkillImportModeDryRun, Status: service.SkillImportRunStatusQueued,
		RequestConfig: json.RawMessage(`{"bootstrap_existing":true}`),
		Snapshot:      json.RawMessage(`{}`),
	}
	require.NoError(t, repo.CreateRun(ctx, secondRun, ""))
	assertBootstrapSucceeded(secondRun.ID)

	var origins, versionOrigins, secondItems int
	require.NoError(t, db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM skill_origins
WHERE source_id=$1 AND namespace='integration/skills' AND external_id=$2`,
		sourceID, externalID).Scan(&origins))
	require.NoError(t, db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM skill_version_origins version_origin
JOIN skill_origins origin ON origin.id=version_origin.origin_id
WHERE origin.source_id=$1 AND origin.namespace='integration/skills'
  AND origin.external_id=$2`, sourceID, externalID).Scan(&versionOrigins))
	require.NoError(t, db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM skill_import_run_items
WHERE run_id=$1 AND status='unchanged'`, secondRun.ID).Scan(&secondItems))
	require.Equal(t, 1, origins, "a repeated bootstrap must reuse the durable origin")
	require.Equal(t, 2, versionOrigins, "each bootstrap run keeps its own provenance evidence")
	require.Equal(t, 1, secondItems)

	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, worker.Stop(stopCtx))
	workerStopped = true
}

func skillImportBootstrapPackage(t *testing.T, slug, skillMD string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	entry, err := writer.Create(slug + "/SKILL.md")
	require.NoError(t, err)
	_, err = entry.Write([]byte(skillMD))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	return buffer.Bytes()
}

func formatImportTestStamp(value int64) string {
	const alphabet = "0123456789abcdefghijklmnopqrstuvwxyz"
	if value < 0 {
		value = -value
	}
	result := make([]byte, 0, 16)
	for value > 0 {
		result = append(result, alphabet[value%int64(len(alphabet))])
		value /= int64(len(alphabet))
	}
	if len(result) == 0 {
		return "0"
	}
	for left, right := 0, len(result)-1; left < right; left, right = left+1, right-1 {
		result[left], result[right] = result[right], result[left]
	}
	return string(result)
}
