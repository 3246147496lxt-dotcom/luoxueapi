package application

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	domain "github.com/Wei-Shaw/sub2api/internal/service"
	core "github.com/Wei-Shaw/sub2api/internal/skillimport"
	"github.com/stretchr/testify/require"
)

type applicationRepositoryStub struct {
	domain.SkillImportRepository
	source          *domain.SkillImportSource
	schedule        *domain.SkillImportSchedule
	updatedSchedule *domain.SkillImportSchedule
	createdRun      *domain.SkillImportRun
	keyHash         string
	getRun          *domain.SkillImportRun
	listRuns        []domain.SkillImportRun
	eligibleIDs     []int64
	listEligible    bool
	publishCalled   bool
	publishResult   *domain.SkillImportPublishResult
}

func (r *applicationRepositoryStub) GetRun(context.Context, int64) (*domain.SkillImportRun, error) {
	return r.getRun, nil
}

func (r *applicationRepositoryStub) ListRuns(context.Context, domain.SkillImportListFilter) ([]domain.SkillImportRun, int64, error) {
	return r.listRuns, int64(len(r.listRuns)), nil
}

func (r *applicationRepositoryStub) GetSource(context.Context, int64) (*domain.SkillImportSource, error) {
	return r.source, nil
}

func (r *applicationRepositoryStub) CreateRun(_ context.Context, run *domain.SkillImportRun, keyHash string) error {
	run.ID = 41
	r.createdRun = run
	r.keyHash = keyHash
	return nil
}

func (r *applicationRepositoryStub) GetSchedule(context.Context, int64) (*domain.SkillImportSchedule, error) {
	return r.schedule, nil
}

func (r *applicationRepositoryStub) UpdateSchedule(_ context.Context, schedule *domain.SkillImportSchedule) error {
	r.updatedSchedule = schedule
	return nil
}

func (r *applicationRepositoryStub) ListEligibleItemIDs(context.Context, int64) ([]int64, error) {
	r.listEligible = true
	return append([]int64(nil), r.eligibleIDs...), nil
}

func (r *applicationRepositoryStub) PublishEligibleItems(context.Context, int64, []int64, string, *int64) (*domain.SkillImportPublishResult, error) {
	r.publishCalled = true
	return r.publishResult, nil
}

type applicationAdapterStub struct{}

func (applicationAdapterStub) Type() string                         { return "fixture" }
func (applicationAdapterStub) Version() string                      { return "2.4.1" }
func (applicationAdapterStub) ValidateConfig(json.RawMessage) error { return nil }
func (applicationAdapterStub) Discover(context.Context, core.DiscoverRequest) (core.DiscoveryPage, error) {
	return core.DiscoveryPage{}, nil
}
func (applicationAdapterStub) Acquire(context.Context, core.AcquireRequest) (core.SourceBundle, error) {
	return core.SourceBundle{}, nil
}

func TestCreateRunFreezesRulesAndHashesIdempotencyKey(t *testing.T) {
	repo := &applicationRepositoryStub{source: &domain.SkillImportSource{
		ID: 7, Adapter: "fixture", Namespace: "catalog", Enabled: true,
		SourceConfig: json.RawMessage(`{"endpoint":"https://example.test/catalog"}`),
	}}
	service := NewService(repo, NewAdapterRegistry(applicationAdapterStub{}), enabledTestConfig())

	run, err := service.CreateRun(context.Background(), RunInput{
		SourceID: 7, Selection: json.RawMessage(`{"limit":500,"start_rank":1}`),
		RunConfig: json.RawMessage(`{"safe_gate":true}`),
	}, nil, "repeatable-request")
	require.NoError(t, err)
	require.Equal(t, int64(41), run.ID)
	require.Equal(t, domain.SkillImportModeAutoPublish, run.Mode)
	require.Equal(t, 500, run.Counts.Requested)
	require.Len(t, repo.keyHash, 64)

	var frozen FrozenRunConfig
	require.NoError(t, json.Unmarshal(run.RequestConfig, &frozen))
	require.Equal(t, "fixture", frozen.AdapterType)
	require.Equal(t, "2.4.1", frozen.AdapterVersion)
	require.Equal(t, core.CoreVersion, frozen.NormalizerVersion)
	require.Equal(t, validatorRulesetVersion, frozen.ValidatorVersion)
}

func TestCreateRunRejectsModeThatBypassesReviewPolicy(t *testing.T) {
	repo := &applicationRepositoryStub{source: &domain.SkillImportSource{
		ID: 7, Adapter: "fixture", Namespace: "catalog", Enabled: true,
		SourceConfig: json.RawMessage(`{}`),
	}}
	service := NewService(repo, NewAdapterRegistry(applicationAdapterStub{}), enabledTestConfig())

	_, err := service.CreateRun(context.Background(), RunInput{
		SourceID: 7, Mode: domain.SkillImportModeAutoPublish,
		PublishPolicy: domain.SkillImportPublishPolicyReview,
	}, nil, "")
	require.ErrorContains(t, err, "must match publish_policy")
	require.Nil(t, repo.createdRun)
}

func TestPublishRunRejectsPreparedItemsFrozenUnderUnavailableRules(t *testing.T) {
	tests := map[string]func(*FrozenRunConfig){
		"old normalizer": func(frozen *FrozenRunConfig) { frozen.NormalizerVersion = "0.9.0" },
		"old validator":  func(frozen *FrozenRunConfig) { frozen.ValidatorVersion = "skill-archive-v0" },
		"old adapter":    func(frozen *FrozenRunConfig) { frozen.AdapterVersion = "2.4.0" },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			frozen := FrozenRunConfig{
				AdapterType: "fixture", AdapterVersion: applicationAdapterStub{}.Version(),
				NormalizerVersion: core.CoreVersion, ValidatorVersion: validatorRulesetVersion,
			}
			mutate(&frozen)
			requestConfig, err := json.Marshal(frozen)
			require.NoError(t, err)
			repo := &applicationRepositoryStub{getRun: &domain.SkillImportRun{
				ID: 41, Status: domain.SkillImportRunStatusAwaitingReview, RequestConfig: requestConfig,
			}}
			service := NewService(repo, NewAdapterRegistry(applicationAdapterStub{}), enabledTestConfig())

			_, err = service.PublishRun(context.Background(), 41, []int64{51}, nil)
			require.ErrorIs(t, err, domain.ErrSkillImportPublishInvalid)
			require.ErrorContains(t, err, "frozen importer rules")
			require.False(t, repo.listEligible)
			require.False(t, repo.publishCalled)
		})
	}
}

func TestPublishRunAllowsPreparedItemsFrozenUnderCurrentRules(t *testing.T) {
	frozen := FrozenRunConfig{
		AdapterType: "fixture", AdapterVersion: applicationAdapterStub{}.Version(),
		NormalizerVersion: core.CoreVersion, ValidatorVersion: validatorRulesetVersion,
	}
	requestConfig, err := json.Marshal(frozen)
	require.NoError(t, err)
	want := &domain.SkillImportPublishResult{RunID: 41, Status: domain.SkillImportRunStatusSucceeded}
	repo := &applicationRepositoryStub{
		getRun:        &domain.SkillImportRun{ID: 41, Status: domain.SkillImportRunStatusAwaitingReview, RequestConfig: requestConfig},
		publishResult: want,
	}
	service := NewService(repo, NewAdapterRegistry(applicationAdapterStub{}), enabledTestConfig())

	got, err := service.PublishRun(context.Background(), 41, []int64{51}, nil)
	require.NoError(t, err)
	require.Same(t, want, got)
	require.True(t, repo.publishCalled)
}

func TestRunResponsesRedactInlineManifestWithoutMutatingWorkerConfig(t *testing.T) {
	encoded := "aGVsbG8="
	repo := &applicationRepositoryStub{source: &domain.SkillImportSource{
		ID: 7, Adapter: "fixture", Namespace: "catalog", Enabled: true,
		SourceConfig: json.RawMessage(`{"upload_only":true}`),
	}}
	service := NewService(repo, NewAdapterRegistry(applicationAdapterStub{}), enabledTestConfig())

	created, err := service.CreateRun(context.Background(), RunInput{
		SourceID:              7,
		AdapterConfigOverride: json.RawMessage(`{"format":"zip","inline_data_base64":"` + encoded + `"}`),
	}, nil, "")
	require.NoError(t, err)
	require.NotContains(t, string(created.RequestConfig), encoded)
	require.Contains(t, string(created.RequestConfig), `"inline_data_redacted":true`)
	require.Contains(t, string(created.RequestConfig), `"inline_data_byte_size":5`)
	require.Contains(t, string(created.RequestConfig), `"inline_data_sha256":"2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"`)
	require.Contains(t, string(repo.createdRun.RequestConfig), encoded, "the durable worker configuration must retain the uploaded bytes")
	require.Contains(t, string(repo.createdRun.RequestConfig), `"inline_data_byte_size":5`, "frozen worker configuration records bounded artifact metadata once")
	require.Contains(t, string(repo.createdRun.RequestConfig), `"inline_data_sha256":"2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"`)
	require.Contains(t, string(repo.createdRun.RequestConfig), `"digest":"sha256:2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"`)

	repo.getRun = repo.createdRun
	repo.listRuns = []domain.SkillImportRun{*repo.createdRun}
	got, err := service.GetRun(context.Background(), repo.createdRun.ID)
	require.NoError(t, err)
	require.NotContains(t, string(got.RequestConfig), encoded)
	listed, err := service.ListRuns(context.Background(), domain.SkillImportListFilter{})
	require.NoError(t, err)
	require.Len(t, listed.Items, 1)
	require.NotContains(t, string(listed.Items[0].RequestConfig), encoded)
	require.Contains(t, string(repo.createdRun.RequestConfig), encoded, "read redaction must operate on a copy")
}

func TestNextScheduleTimeUsesConfiguredTimezone(t *testing.T) {
	from := time.Date(2026, time.August, 12, 16, 30, 0, 0, time.UTC)
	next, err := nextScheduleTime("0 3 * * *", "Asia/Shanghai", from)
	require.NoError(t, err)
	require.Equal(t, time.Date(2026, time.August, 12, 19, 0, 0, 0, time.UTC), *next)
}

func TestMergeJSONObjectsPreservesSourcePolicyAndOverlaysArtifact(t *testing.T) {
	merged, err := mergeJSONObjects(
		json.RawMessage(`{"upload_only":true,"allowed_hosts":["assets.example.test"],"format":"auto"}`),
		json.RawMessage(`{"inline_data_base64":"c2tpbGw=","format":"zip"}`),
	)
	require.NoError(t, err)
	var value map[string]any
	require.NoError(t, json.Unmarshal(merged, &value))
	require.Equal(t, true, value["upload_only"])
	require.Equal(t, "zip", value["format"])
	require.Equal(t, "c2tpbGw=", value["inline_data_base64"])
	require.Equal(t, []any{"assets.example.test"}, value["allowed_hosts"])
}

func TestNormalizeJSONObjectRejectsTrailingJSONValue(t *testing.T) {
	_, err := normalizeJSONObject(json.RawMessage(`{"safe_gate":true} {"second":true}`))
	require.ErrorContains(t, err, "exactly one")
}

func TestSourceConfigRejectsDurableInlineUploadData(t *testing.T) {
	service := NewService(&applicationRepositoryStub{}, NewAdapterRegistry(core.NewManifestAdapter(nil)), enabledTestConfig())
	_, err := service.normalizeSource(SourceInput{
		Name: "Unsafe inline source", Adapter: core.ManifestAdapterType,
		Namespace: "inline-source", Enabled: true,
		SourceConfig: json.RawMessage(`{"format":"zip","inline_data_base64":"c2tpbGw="}`),
	})
	require.ErrorContains(t, err, "only allowed on a single run upload")
}

func TestSourceResponseDefensivelyRedactsLegacyInlineData(t *testing.T) {
	original := &domain.SkillImportSource{SourceConfig: json.RawMessage(`{"format":"zip","INLINE_DATA_BASE64":"c2Vuc2l0aXZl","inline_data_sha256":"digest"}`)}
	redacted := redactSourceForResponse(original)
	require.NotContains(t, string(redacted.SourceConfig), "c2Vuc2l0aXZl")
	require.Contains(t, string(redacted.SourceConfig), `"inline_data_redacted":true`)
	require.Contains(t, string(original.SourceConfig), "c2Vuc2l0aXZl", "redaction must not mutate worker configuration")
}

func TestUpdateScheduleKeepsDurableSourceIdentity(t *testing.T) {
	repo := &applicationRepositoryStub{
		source: &domain.SkillImportSource{ID: 7, Adapter: "fixture", Namespace: "catalog", Enabled: true},
		schedule: &domain.SkillImportSchedule{
			ID: 9, SourceID: 7, Name: "Daily", Enabled: false,
			CronExpression: "0 3 * * *", Timezone: "Asia/Shanghai",
			Selection: json.RawMessage(`{"limit":500}`), RunConfig: json.RawMessage(`{}`),
			PublishPolicy:  domain.SkillImportPublishPolicyReview,
			MetadataPolicy: domain.SkillImportMetadataPolicyRefresh,
		},
	}
	service := NewService(repo, NewAdapterRegistry(applicationAdapterStub{}), enabledTestConfig())

	updated, err := service.UpdateSchedule(context.Background(), 9, ScheduleInput{
		SourceID: 999, Name: "Daily", Enabled: false,
		CronExpression: "0 4 * * *", Timezone: "Asia/Shanghai",
		Selection: json.RawMessage(`{"limit":100}`), RunConfig: json.RawMessage(`{}`),
		PublishPolicy:  domain.SkillImportPublishPolicyReview,
		MetadataPolicy: domain.SkillImportMetadataPolicyRefresh,
	}, nil)
	require.NoError(t, err)
	require.Equal(t, int64(7), repo.updatedSchedule.SourceID)
	require.Equal(t, int64(7), updated.SourceID)
}

func enabledTestConfig() *config.Config {
	return &config.Config{SkillImport: config.SkillImportConfig{Enabled: true}}
}
