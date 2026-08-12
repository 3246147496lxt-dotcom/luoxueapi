package application

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	domain "github.com/Wei-Shaw/sub2api/internal/service"
	core "github.com/Wei-Shaw/sub2api/internal/skillimport"
	"github.com/stretchr/testify/require"
)

type workerRetryRepositoryStub struct {
	domain.SkillImportRepository
	nextRetryAt *time.Time
	err         error
}

func (r *workerRetryRepositoryStub) GetNextRunItemRetryAt(_ context.Context, _ int64) (*time.Time, error) {
	return r.nextRetryAt, r.err
}

func TestItemConcurrencyUsesRunValueWithinProcessLimit(t *testing.T) {
	require.Equal(t, 3, itemConcurrency(json.RawMessage(`{"concurrency":3}`), 8))
	require.Equal(t, 4, itemConcurrency(json.RawMessage(`{"concurrency":12}`), 4))
	require.Equal(t, 2, itemConcurrency(json.RawMessage(`{}`), 2))
}

func TestItemLeaseOwnerIsUniquePerClaimAndScopedToRunWorker(t *testing.T) {
	const runWorker = "skill-import-worker-a"
	seen := make(map[string]struct{}, 128)
	for index := 0; index < 128; index++ {
		claim := newItemLeaseOwner(runWorker)
		require.NotEqual(t, runWorker, claim)
		require.Contains(t, claim, runWorker+":item:")
		_, duplicate := seen[claim]
		require.False(t, duplicate)
		seen[claim] = struct{}{}
	}
}

func TestAutoPublishGateDefaultsToEligibleCohort(t *testing.T) {
	gate := autoPublishGate(json.RawMessage(`{}`))
	require.True(t, gate.AllowLicenseUnverified)
	require.False(t, gate.RequiresReview(domain.SkillImportRunCounts{Blocked: 8, Failed: 2}))
}

func TestAutoPublishGateCanHoldCohortForReview(t *testing.T) {
	gate := autoPublishGate(json.RawMessage(`{
		"auto_publish_gate": {
			"require_all_valid": false,
			"allow_license_unverified": false,
			"max_blocked_items": 2,
			"max_failed_items": 1
		}
	}`))
	require.False(t, gate.AllowLicenseUnverified)
	require.False(t, gate.RequiresReview(domain.SkillImportRunCounts{Blocked: 2, Failed: 1}))
	require.True(t, gate.RequiresReview(domain.SkillImportRunCounts{Blocked: 3, Failed: 1}))
}

func TestValidateRunConfigRejectsNegativeGateLimit(t *testing.T) {
	err := validateSkillImportRunConfig(json.RawMessage(`{"auto_publish_gate":{"max_failed_items":-1}}`))
	require.ErrorContains(t, err, "cannot be negative")
}

func TestValidateRunConfigRejectsMistypedOrUnknownSafetyPolicy(t *testing.T) {
	for _, raw := range []string{
		`{"auto_publish_gate":{"require_all_valid":"true"}}`,
		`{"auto_publish_gate":{"allow_license_unverified":"false"}}`,
		`{"auto_publish_gate":{"unexpected":true}}`,
		`{"unexpected":true}`,
		`{"concurrency":0}`,
	} {
		require.Error(t, validateSkillImportRunConfig(json.RawMessage(raw)), raw)
	}
}

func TestMalformedRunConfigPolicyHelpersFailClosed(t *testing.T) {
	raw := json.RawMessage(`{"safe_gate":"false","auto_publish_gate":{"allow_license_unverified":"true"}}`)
	require.True(t, safeGateEnabled(raw))
	gate := autoPublishGate(raw)
	require.True(t, gate.RequireAllValid)
	require.False(t, gate.AllowLicenseUnverified)
}

func TestNeedsReviewPolicyByRunMode(t *testing.T) {
	config := json.RawMessage(`{"safe_gate":true}`)
	require.True(t, shouldBlockNeedsReview(domain.SkillImportModeAutoPublish, config, true))
	require.False(t, shouldBlockNeedsReview(domain.SkillImportModeReview, config, true), "review runs must stage reviewable packages")
	require.False(t, shouldBlockNeedsReview(domain.SkillImportModeDryRun, config, true), "dry runs must retain review evidence and skip staging")
	require.False(t, shouldBlockNeedsReview(domain.SkillImportModeAutoPublish, config, false))
	require.False(t, shouldBlockNeedsReview(domain.SkillImportModeAutoPublish, json.RawMessage(`{"safe_gate":false}`), true))
}

func TestRunDeadlineIsFreshForEveryClaim(t *testing.T) {
	old := time.Now().UTC().Add(-72 * time.Hour)
	runtime := &WorkerRuntime{cfg: config.SkillImportConfig{MaxRunDurationHours: 1}}
	ctx, cancel := runtime.withRunDeadline(context.Background(), &domain.SkillImportRun{StartedAt: &old})
	defer cancel()
	deadline, ok := ctx.Deadline()
	require.True(t, ok)
	require.Greater(t, time.Until(deadline), 59*time.Minute)
	require.NoError(t, ctx.Err(), "a resumed run must not inherit an already-expired first-attempt deadline")
}

func TestBootstrapBatchHasBoundedPackageMemory(t *testing.T) {
	require.LessOrEqual(t,
		int64(bootstrapCandidateBatchSize)*domain.SkillArchiveMaxBytes,
		int64(50*1024*1024),
	)
}

func TestRunDeadlineBecomesLeaseAwareTemporaryFailure(t *testing.T) {
	err := runProcessingError(context.DeadlineExceeded, 5*time.Minute)
	require.Equal(t, core.ErrorTemporary, core.ErrorKindOf(err))
	retryAfter, ok := core.RetryAfterOf(err)
	require.True(t, ok)
	require.Equal(t, 5*time.Minute, retryAfter)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestNextItemRetryUsesUnpaginatedRepositoryAggregate(t *testing.T) {
	lease := time.Now().UTC().Add(4 * time.Minute)
	repo := &workerRetryRepositoryStub{nextRetryAt: &lease}
	runtime := &WorkerRuntime{service: &Service{repo: repo}, cfg: config.SkillImportConfig{PollIntervalSeconds: 2}}
	require.Equal(t, lease, runtime.nextItemRetryAt(context.Background(), 7))
}

func TestNextItemRetrySafelyFallsBackWhenAggregateFails(t *testing.T) {
	repo := &workerRetryRepositoryStub{err: context.DeadlineExceeded}
	runtime := &WorkerRuntime{service: &Service{repo: repo}, cfg: config.SkillImportConfig{PollIntervalSeconds: 3}}
	before := time.Now().UTC().Add(3 * time.Second)
	next := runtime.nextItemRetryAt(context.Background(), 7)
	after := time.Now().UTC().Add(3 * time.Second)
	require.False(t, next.Before(before))
	require.False(t, next.After(after))
}

func TestDesiredSkillOnlyAttributesActualGitHubSources(t *testing.T) {
	normalized := core.NormalizedSkill{MarketSlug: "demo", DisplayName: "Demo", Description: "Demo skill"}
	mapped := desiredSkillFromNormalized(
		core.DiscoveredSkill{Namespace: "sentry/dev", ExternalID: "demo"},
		core.SourceBundle{}, normalized,
		FrozenRunConfig{
			AdapterType:   core.SkillsSHAdapterType,
			AdapterConfig: json.RawMessage(`{"source_base_urls":{"sentry/dev":"https://cli.sentry.dev"}}`),
		},
	)
	require.Empty(t, mapped.SourceURL)
	require.Empty(t, mapped.SourceRepository)

	github := desiredSkillFromNormalized(
		core.DiscoveredSkill{Namespace: "anthropics/skills", ExternalID: "frontend-design"},
		core.SourceBundle{}, normalized,
		FrozenRunConfig{AdapterType: core.SkillsSHAdapterType, AdapterConfig: json.RawMessage(`{}`)},
	)
	require.Equal(t, "https://github.com/anthropics/skills", github.SourceURL)
	require.Equal(t, "anthropics/skills", github.SourceRepository)

	manifest := desiredSkillFromNormalized(
		core.DiscoveredSkill{Namespace: "looks/like-github", ExternalID: "demo"},
		core.SourceBundle{}, normalized,
		FrozenRunConfig{AdapterType: core.ManifestAdapterType},
	)
	require.Empty(t, manifest.SourceURL)
}

func TestDiscoveredIdentityGateMatchesPersistenceBoundaries(t *testing.T) {
	require.NoError(t, validateDiscoveredSkill(core.DiscoveredSkill{
		AdapterType: core.ManifestAdapterType, Namespace: "inline-manifest", ExternalID: "demo",
		CanonicalURL: "",
	}, core.ManifestAdapterType), "inline uploads may intentionally have no public origin")
	require.Error(t, validateDiscoveredSkill(core.DiscoveredSkill{
		AdapterType: core.ManifestAdapterType, Namespace: "bad namespace", ExternalID: "demo",
	}, core.ManifestAdapterType))
	require.Error(t, validateDiscoveredSkill(core.DiscoveredSkill{
		AdapterType: core.ManifestAdapterType, Namespace: "catalog", ExternalID: strings.Repeat("x", 513),
	}, core.ManifestAdapterType))
	require.Error(t, validateDiscoveredSkill(core.DiscoveredSkill{
		AdapterType: core.ManifestAdapterType, Namespace: "catalog", ExternalID: "demo",
		CanonicalURL: "http://catalog.example.test/demo",
	}, core.ManifestAdapterType))
}
