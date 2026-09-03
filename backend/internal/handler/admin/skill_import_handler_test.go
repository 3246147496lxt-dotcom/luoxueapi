package admin

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/modules/skillimport/application"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	domain "github.com/Wei-Shaw/sub2api/internal/service"
	core "github.com/Wei-Shaw/sub2api/internal/skillimport"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type skillImportHandlerRepositoryStub struct {
	domain.SkillImportRepository
	source              *domain.SkillImportSource
	createdRuns         []*domain.SkillImportRun
	runs                map[int64]*domain.SkillImportRun
	keyHashes           []string
	cancelCalls         int
	cancelActorID       int64
	retryRunIDs         []int64
	retryItemIDs        [][]int64
	nextRunID           int64
	requestCancellation bool
}

func (r *skillImportHandlerRepositoryStub) GetSource(_ context.Context, id int64) (*domain.SkillImportSource, error) {
	if r.source == nil || r.source.ID != id {
		return nil, domain.ErrSkillImportSourceNotFound
	}
	result := *r.source
	result.SourceConfig = append(json.RawMessage(nil), r.source.SourceConfig...)
	return &result, nil
}

func (r *skillImportHandlerRepositoryStub) CreateRun(_ context.Context, run *domain.SkillImportRun, keyHash string) error {
	r.nextRunID++
	run.ID = r.nextRunID
	created := cloneSkillImportHandlerRun(run)
	r.createdRuns = append(r.createdRuns, created)
	r.keyHashes = append(r.keyHashes, keyHash)
	if r.runs == nil {
		r.runs = make(map[int64]*domain.SkillImportRun)
	}
	r.runs[run.ID] = cloneSkillImportHandlerRun(run)
	return nil
}

func (r *skillImportHandlerRepositoryStub) GetRun(_ context.Context, id int64) (*domain.SkillImportRun, error) {
	run := r.runs[id]
	if run == nil {
		return nil, domain.ErrSkillImportRunNotFound
	}
	return cloneSkillImportHandlerRun(run), nil
}

func (r *skillImportHandlerRepositoryStub) RequestRunCancellation(_ context.Context, runID, actorID int64) (bool, error) {
	r.cancelCalls++
	r.cancelActorID = actorID
	run := r.runs[runID]
	if run == nil {
		return false, domain.ErrSkillImportRunNotFound
	}
	r.requestCancellation = true
	now := time.Now().UTC()
	run.CancelRequestedAt = &now
	run.CancelRequestedBy = &actorID
	return true, nil
}

func (r *skillImportHandlerRepositoryStub) ResetFailedItems(_ context.Context, runID int64, itemIDs []int64) (int64, error) {
	if r.runs[runID] == nil {
		return 0, domain.ErrSkillImportRunNotFound
	}
	r.retryRunIDs = append(r.retryRunIDs, runID)
	r.retryItemIDs = append(r.retryItemIDs, append([]int64(nil), itemIDs...))
	return int64(len(itemIDs)), nil
}

func cloneSkillImportHandlerRun(run *domain.SkillImportRun) *domain.SkillImportRun {
	if run == nil {
		return nil
	}
	result := *run
	result.RequestConfig = append(json.RawMessage(nil), run.RequestConfig...)
	result.Snapshot = append(json.RawMessage(nil), run.Snapshot...)
	return &result
}

func newSkillImportHandlerTestService(repo *skillImportHandlerRepositoryStub) *application.Service {
	return application.NewService(
		repo,
		application.NewAdapterRegistry(core.NewManifestAdapter(nil)),
		&config.Config{SkillImport: config.SkillImportConfig{Enabled: true}},
	)
}

func newSkillImportHandlerTestRouter(handler *SkillImportHandler, actorID int64) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if actorID > 0 {
		router.Use(func(c *gin.Context) {
			c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: actorID})
			c.Next()
		})
	}
	router.POST("/api/v1/admin/skill-import/runs", handler.CreateRun)
	router.POST("/api/v1/admin/skill-import/runs/upload", handler.UploadRun)
	router.GET("/api/v1/admin/skill-import/runs/:id", handler.GetRun)
	router.POST("/api/v1/admin/skill-import/runs/:id/cancel", handler.CancelRun)
	router.POST("/api/v1/admin/skill-import/runs/:id/retry-failed", handler.RetryFailed)
	return router
}

func newManifestUploadRequest(
	t *testing.T,
	filename string,
	data []byte,
	fields map[string]string,
) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	file, err := writer.CreateFormFile("file", filename)
	require.NoError(t, err)
	_, err = file.Write(data)
	require.NoError(t, err)
	for key, value := range fields {
		require.NoError(t, writer.WriteField(key, value))
	}
	require.NoError(t, writer.Close())
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/skill-import/runs/upload", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}

func setSkillImportHandlerIdempotencyCoordinator(t *testing.T, coordinator *domain.IdempotencyCoordinator) {
	t.Helper()
	previous := domain.DefaultIdempotencyCoordinator()
	domain.SetDefaultIdempotencyCoordinator(coordinator)
	t.Cleanup(func() { domain.SetDefaultIdempotencyCoordinator(previous) })
}

func TestSkillImportUploadAcceptsExactFiveMiBAndRedactsArtifact(t *testing.T) {
	setSkillImportHandlerIdempotencyCoordinator(t, nil)
	repo := &skillImportHandlerRepositoryStub{
		source: &domain.SkillImportSource{
			ID: 11, Adapter: core.ManifestAdapterType, Namespace: "uploaded.catalog",
			SourceConfig: json.RawMessage(`{"upload_only":true}`), Enabled: true,
		},
	}
	router := newSkillImportHandlerTestRouter(NewSkillImportHandler(newSkillImportHandlerTestService(repo)), 77)
	prefix := []byte("id,name,description\ndemo,Demo,")
	data := append(prefix, bytes.Repeat([]byte("a"), int(domain.SkillArchiveMaxBytes)-len(prefix)-1)...)
	data = append(data, '\n')
	require.Len(t, data, int(domain.SkillArchiveMaxBytes))
	request := newManifestUploadRequest(t, "catalog.CSV", data, map[string]string{
		"source_id": "11", "mode": domain.SkillImportModeReview,
		"selection": `{"limit":1}`, "run_config": `{"safe_gate":true}`,
		"publish_policy":  domain.SkillImportPublishPolicyReview,
		"metadata_policy": domain.SkillImportMetadataPolicyRefresh,
	})
	request.Header.Set("X-Idempotency-Key", "upload-key")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Len(t, repo.createdRuns, 1)
	require.NotNil(t, repo.createdRuns[0].CreatedBy)
	require.Equal(t, int64(77), *repo.createdRuns[0].CreatedBy)
	digest := sha256.Sum256([]byte("upload-key"))
	require.Equal(t, hex.EncodeToString(digest[:]), repo.keyHashes[0])

	var stored application.FrozenRunConfig
	require.NoError(t, json.Unmarshal(repo.createdRuns[0].RequestConfig, &stored))
	var storedAdapter map[string]any
	require.NoError(t, json.Unmarshal(stored.AdapterConfig, &storedAdapter))
	encoded, ok := storedAdapter["inline_data_base64"].(string)
	require.True(t, ok)
	require.Len(t, encoded, base64.StdEncoding.EncodedLen(len(data)))
	require.Equal(t, float64(len(data)), storedAdapter["inline_data_byte_size"])
	require.Equal(t, domain.SkillImportPublishPolicyReview, stored.PublishPolicy)
	require.Equal(t, domain.SkillImportMetadataPolicyRefresh, stored.MetadataPolicy)

	var envelope struct {
		Data domain.SkillImportRun `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	var responseConfig application.FrozenRunConfig
	require.NoError(t, json.Unmarshal(envelope.Data.RequestConfig, &responseConfig))
	var responseAdapter map[string]any
	require.NoError(t, json.Unmarshal(responseConfig.AdapterConfig, &responseAdapter))
	require.NotContains(t, responseAdapter, "inline_data_base64")
	require.Equal(t, true, responseAdapter["inline_data_redacted"])
	require.Equal(t, float64(len(data)), responseAdapter["inline_data_byte_size"])
}

func TestSkillImportUploadRejectsUnsafeFilesAndNonManifestSources(t *testing.T) {
	tests := []struct {
		name       string
		adapter    string
		filename   string
		data       []byte
		wantStatus int
	}{
		{
			name: "one byte over five MiB", adapter: core.ManifestAdapterType, filename: "catalog.csv",
			data: bytes.Repeat([]byte("x"), int(domain.SkillArchiveMaxBytes)+1), wantStatus: http.StatusBadRequest,
		},
		{
			name: "unsupported extension", adapter: core.ManifestAdapterType, filename: "catalog.json.exe",
			data: []byte(`[]`), wantStatus: http.StatusBadRequest,
		},
		{
			name: "non manifest source", adapter: "github", filename: "catalog.json",
			data: []byte(`[]`), wantStatus: http.StatusBadRequest,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setSkillImportHandlerIdempotencyCoordinator(t, nil)
			repo := &skillImportHandlerRepositoryStub{source: &domain.SkillImportSource{
				ID: 12, Adapter: test.adapter, Namespace: "test", SourceConfig: json.RawMessage(`{"upload_only":true}`), Enabled: true,
			}}
			router := newSkillImportHandlerTestRouter(NewSkillImportHandler(newSkillImportHandlerTestService(repo)), 77)
			request := newManifestUploadRequest(t, test.filename, test.data, map[string]string{"source_id": "12"})
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			require.Equal(t, test.wantStatus, recorder.Code, recorder.Body.String())
			require.Empty(t, repo.createdRuns)
		})
	}
}

func TestSkillImportJSONDecodingAndIDsAreStrict(t *testing.T) {
	handler := NewSkillImportHandler(nil)
	router := newSkillImportHandlerTestRouter(handler, 0)
	for _, body := range []string{
		`{"source_id":1,"unknown":true}`,
		`{"source_id":1} {"second":true}`,
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/skill-import/runs", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(recorder, request)

		require.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}

	for _, id := range []string{"0", "-1", "not-a-number", strconv.FormatUint(uint64(^uint64(0)), 10)} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/skill-import/runs/"+id, nil)

		router.ServeHTTP(recorder, request)

		require.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/skill-import/runs/1/cancel", nil)
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusUnauthorized, recorder.Code, recorder.Body.String())
}

func TestSkillImportUploadRequestConfigUsesStrictJSONObject(t *testing.T) {
	for _, raw := range []string{
		`[]`,
		`{} {"second":true}`,
		`{"unknown":true}`,
		`{"selection":{}} trailing`,
	} {
		var target skillImportUploadRequestConfig
		require.Error(t, decodeSkillImportUploadRequestConfig(raw, &target), raw)
	}
	var valid skillImportUploadRequestConfig
	require.NoError(t, decodeSkillImportUploadRequestConfig(
		`{"selection":{"limit":2},"run_config":{"safe_gate":true},"publish_policy":"review","metadata_policy":"refresh"}`,
		&valid,
	))
	require.JSONEq(t, `{"limit":2}`, string(valid.Selection))
}

func TestSkillImportCancelRequiresActorAndReplaysIdempotently(t *testing.T) {
	cfg := domain.DefaultIdempotencyConfig()
	cfg.ObserveOnly = false
	setSkillImportHandlerIdempotencyCoordinator(t, domain.NewIdempotencyCoordinator(newMemoryIdempotencyRepoStub(), cfg))
	repo := &skillImportHandlerRepositoryStub{runs: map[int64]*domain.SkillImportRun{
		9: {ID: 9, SourceID: 1, Status: domain.SkillImportRunStatusPreparing, RequestConfig: json.RawMessage(`{}`)},
	}}
	router := newSkillImportHandlerTestRouter(NewSkillImportHandler(newSkillImportHandlerTestService(repo)), 77)

	for attempt := 0; attempt < 2; attempt++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/skill-import/runs/9/cancel", nil)
		request.Header.Set("Idempotency-Key", "cancel-run-9")
		router.ServeHTTP(recorder, request)
		require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
		if attempt == 1 {
			require.Equal(t, "true", recorder.Header().Get("X-Idempotency-Replayed"))
		}
	}
	require.Equal(t, 1, repo.cancelCalls)
	require.Equal(t, int64(77), repo.cancelActorID)
	require.True(t, repo.requestCancellation)
}

func TestSkillImportRetryIdempotencyFingerprintIncludesRunID(t *testing.T) {
	cfg := domain.DefaultIdempotencyConfig()
	cfg.ObserveOnly = false
	setSkillImportHandlerIdempotencyCoordinator(t, domain.NewIdempotencyCoordinator(newMemoryIdempotencyRepoStub(), cfg))
	repo := &skillImportHandlerRepositoryStub{runs: map[int64]*domain.SkillImportRun{
		1: {ID: 1, Status: domain.SkillImportRunStatusFailed, RequestConfig: json.RawMessage(`{}`)},
		2: {ID: 2, Status: domain.SkillImportRunStatusFailed, RequestConfig: json.RawMessage(`{}`)},
	}}
	router := newSkillImportHandlerTestRouter(NewSkillImportHandler(newSkillImportHandlerTestService(repo)), 77)

	first := httptest.NewRecorder()
	firstRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/skill-import/runs/1/retry-failed", strings.NewReader(`{"item_ids":[4]}`))
	firstRequest.Header.Set("Content-Type", "application/json")
	firstRequest.Header.Set("Idempotency-Key", "retry-key")
	router.ServeHTTP(first, firstRequest)
	require.Equal(t, http.StatusOK, first.Code, first.Body.String())

	second := httptest.NewRecorder()
	secondRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/skill-import/runs/2/retry-failed", strings.NewReader(`{"item_ids":[4]}`))
	secondRequest.Header.Set("Content-Type", "application/json")
	secondRequest.Header.Set("Idempotency-Key", "retry-key")
	router.ServeHTTP(second, secondRequest)
	require.Equal(t, http.StatusConflict, second.Code, second.Body.String())
	require.Equal(t, []int64{1}, repo.retryRunIDs)
	require.Equal(t, [][]int64{{4}}, repo.retryItemIDs)
}

func TestSkillImportUploadIdempotencyFingerprintIncludesPolicies(t *testing.T) {
	cfg := domain.DefaultIdempotencyConfig()
	cfg.ObserveOnly = false
	setSkillImportHandlerIdempotencyCoordinator(t, domain.NewIdempotencyCoordinator(newMemoryIdempotencyRepoStub(), cfg))
	repo := &skillImportHandlerRepositoryStub{source: &domain.SkillImportSource{
		ID: 13, Adapter: core.ManifestAdapterType, Namespace: "test",
		SourceConfig: json.RawMessage(`{"upload_only":true}`), Enabled: true,
	}}
	router := newSkillImportHandlerTestRouter(NewSkillImportHandler(newSkillImportHandlerTestService(repo)), 77)

	call := func(policy string) *httptest.ResponseRecorder {
		request := newManifestUploadRequest(t, "catalog.json", []byte(`[]`), map[string]string{
			"source_id": "13", "mode": domain.SkillImportModeDryRun,
			"publish_policy": policy, "metadata_policy": domain.SkillImportMetadataPolicyRefresh,
		})
		request.Header.Set("Idempotency-Key", "same-upload-key")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		return recorder
	}

	require.Equal(t, http.StatusOK, call(domain.SkillImportPublishPolicyAuto).Code)
	conflict := call(domain.SkillImportPublishPolicyReview)
	require.Equal(t, http.StatusConflict, conflict.Code, conflict.Body.String())
	require.Len(t, repo.createdRuns, 1)
}
