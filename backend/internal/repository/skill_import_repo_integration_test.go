//go:build integration

package repository

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

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
