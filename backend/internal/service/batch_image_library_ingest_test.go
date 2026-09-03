//go:build unit

package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type generatedLibraryRepoFake struct {
	libraryRepoFake
	bySourceKey map[string]string
	quotaCalls  int
	quotaErr    error
}

func (f *generatedLibraryRepoFake) CheckUploadQuota(context.Context, int64, int64) error {
	f.quotaCalls++
	return f.quotaErr
}

func (f *generatedLibraryRepoFake) CreatePending(ctx context.Context, input *CreateLibraryFileInput, limit int64) (*LibraryFile, error) {
	if input.SourceKey != "" {
		if id := f.bySourceKey[input.SourceKey]; id != "" {
			file := f.files[id]
			return &file, nil
		}
	}
	created, err := f.libraryRepoFake.CreatePending(ctx, input, limit)
	if err != nil {
		return nil, err
	}
	if input.SourceKey != "" {
		if f.bySourceKey == nil {
			f.bySourceKey = make(map[string]string)
		}
		created.SourceKey = input.SourceKey
		f.files[created.ID] = *created
		f.bySourceKey[input.SourceKey] = created.ID
	}
	return created, nil
}

func (f *generatedLibraryRepoFake) Abort(ctx context.Context, userID int64, id string) (*LibraryFile, error) {
	file, err := f.libraryRepoFake.Abort(ctx, userID, id)
	if err == nil && file != nil && file.SourceKey != "" {
		delete(f.bySourceKey, file.SourceKey)
		file.SourceKey = ""
		f.files[id] = *file
	}
	return file, err
}

type batchImageLibraryStateFake struct {
	state       BatchImageLibraryIngestState
	retries     int
	completed   int
	beginErr    error
	completeErr error
}

func (f *batchImageLibraryStateFake) BeginBatchImageLibraryIngest(_ context.Context, batchID string, maxAttempts int) (*BatchImageLibraryIngestState, bool, error) {
	if f.beginErr != nil {
		return nil, false, f.beginErr
	}
	if f.state.JobID == "" {
		f.state.JobID = batchID
		f.state.Status = BatchImageLibraryIngestPending
	}
	if f.state.Status == BatchImageLibraryIngestCompleted || f.state.Status == BatchImageLibraryIngestCompletedWithErrors {
		return &f.state, false, nil
	}
	if f.state.Attempts >= maxAttempts {
		f.state.Status = BatchImageLibraryIngestCompletedWithErrors
		return &f.state, false, nil
	}
	f.state.Attempts++
	copyState := f.state
	return &copyState, true, nil
}

func TestBatchImageLibraryIngestRequeuesWhenDurableStateCannotBegin(t *testing.T) {
	state := &batchImageLibraryStateFake{beginErr: errors.New("database unavailable")}
	ingestor := &BatchImageLibraryIngestor{StateRepo: state, Library: NewLibraryService(&generatedLibraryRepoFake{}, &libraryBlobStoreFake{}, nil), RetryDelay: 25 * time.Millisecond}
	result := ingestor.Ingest(context.Background(), &BatchImageJob{BatchID: "imgbatch_state_down", Status: BatchImageJobStatusCompleted})
	require.False(t, result.Done)
	require.Equal(t, 25*time.Millisecond, result.RetryAfter)
}

func (f *batchImageLibraryStateFake) RetryBatchImageLibraryIngest(_ context.Context, _ string, attempt BatchImageLibraryIngestAttempt) error {
	f.retries++
	f.state.Status = BatchImageLibraryIngestRetrying
	f.apply(attempt)
	return nil
}

func (f *batchImageLibraryStateFake) CompleteBatchImageLibraryIngest(_ context.Context, _ string, status string, attempt BatchImageLibraryIngestAttempt) error {
	f.completed++
	if f.completeErr != nil {
		return f.completeErr
	}
	f.state.Status = status
	f.apply(attempt)
	return nil
}

func TestBatchImageLibraryIngestRequeuesFinalAttemptWhenTerminalStateWriteFails(t *testing.T) {
	imageData := encodeBatchImageLibraryPNG(t)
	provider := &fakeProcessorProvider{result: batchImageLibraryResultLine("retry", imageData)}
	accountID := int64(3)
	job := &BatchImageJob{BatchID: "imgbatch_final_state_write", UserID: 2, AccountID: &accountID, Provider: "fake", Status: BatchImageJobStatusCompleted}
	batchRepo := newFakeBatchImageRepository()
	batchRepo.jobs[job.BatchID] = job
	batchRepo.items[job.BatchID] = []CreateBatchImageItemParams{{JobID: job.BatchID, CustomID: "retry", Status: BatchImageItemStatusSuccess, ImageCount: 1}}
	state := &batchImageLibraryStateFake{state: BatchImageLibraryIngestState{JobID: job.BatchID, Status: BatchImageLibraryIngestRetrying, Attempts: 2}, completeErr: errors.New("state write failed")}
	ingestor := &BatchImageLibraryIngestor{
		Repo: batchRepo, StateRepo: state, ProviderRegistry: NewBatchImageProviderRegistry(provider),
		AccountResolver: &fakeBatchImageAccountResolver{account: &Account{}},
		Library:         NewLibraryService(&generatedLibraryRepoFake{}, &failingLibraryBlobStore{putErr: errors.New("storage offline")}, nil),
		MaxAttempts:     3, RetryDelay: 25 * time.Millisecond,
	}
	result := ingestor.Ingest(context.Background(), job)
	require.False(t, result.Done)
	require.Equal(t, 25*time.Millisecond, result.RetryAfter)
	require.Equal(t, 3, state.state.Attempts)

	// The next durable Begin observes attempts>=max and terminalizes without
	// rescanning provider output (the fake represents that branch as !started).
	state.completeErr = nil
	provider.openResultCalled = false
	require.True(t, ingestor.Ingest(context.Background(), job).Done)
	require.False(t, provider.openResultCalled)
}

func (f *batchImageLibraryStateFake) apply(attempt BatchImageLibraryIngestAttempt) {
	f.state.ImportedCount = attempt.ImportedCount
	f.state.SuppressedCount = attempt.SuppressedCount
	f.state.FailedCount = attempt.FailedCount
	f.state.LastErrorCode = attempt.ErrorCode
	f.state.LastErrorMessage = attempt.ErrorMessage
}

func TestBatchImageLibraryIngestSuccessMultiImageAndPipelineCompleted(t *testing.T) {
	imageData := encodeBatchImageLibraryPNG(t)
	encoded := base64.StdEncoding.EncodeToString(imageData)
	provider := &fakeProcessorProvider{result: `{"key":"img/alpha","response":{"candidates":[{"content":{"parts":[` +
		`{"inlineData":{"mimeType":"image/png","data":"` + encoded + `"}},` +
		`{"inlineData":{"mimeType":"image/png","data":"` + encoded + `"}}]}}]}}` + "\n" +
		batchImageLibraryResultLine("provider-extra", imageData)}
	accountID := int64(17)
	job := &BatchImageJob{BatchID: "imgbatch_library_success", UserID: 9, AccountID: &accountID, Provider: "fake", Status: BatchImageJobStatusCompleted}
	batchRepo := newFakeBatchImageRepository()
	batchRepo.jobs[job.BatchID] = job
	batchRepo.items[job.BatchID] = []CreateBatchImageItemParams{{
		JobID: job.BatchID, CustomID: "img/alpha", Status: BatchImageItemStatusSuccess, ImageCount: 2,
	}}
	libraryRepo := &generatedLibraryRepoFake{quotaErr: ErrChatAttachmentRateLimit}
	library := NewLibraryService(libraryRepo, &libraryBlobStoreFake{}, &config.Config{
		Library:         config.LibraryConfig{MaxFileBytes: 20 << 20},
		ChatAttachments: config.ChatAttachmentConfig{MaxImageBytes: 10 << 20, MaxConcurrentGlobal: 1, MaxConcurrentPerUser: 1},
	})
	state := &batchImageLibraryStateFake{}
	ingestor := &BatchImageLibraryIngestor{
		Repo: batchRepo, StateRepo: state, ProviderRegistry: NewBatchImageProviderRegistry(provider),
		AccountResolver: &fakeBatchImageAccountResolver{account: &Account{}}, Library: library,
	}
	processor := &BatchImagePipelineProcessor{
		ProviderProcessor: &BatchImageProviderProcessor{Repo: batchRepo}, LibraryIngestor: ingestor,
	}

	result, err := processor.Process(context.Background(), job.BatchID)
	require.NoError(t, err)
	require.True(t, result.Terminal)
	require.Equal(t, BatchImageLibraryIngestCompleted, state.state.Status)
	require.Equal(t, 2, state.state.ImportedCount)
	require.Equal(t, 2, libraryRepo.created)
	require.Zero(t, libraryRepo.quotaCalls, "trusted generated ingestion must bypass interactive upload quotas")
	require.Contains(t, libraryRepo.bySourceKey, BatchImageLibrarySourceKey(job.BatchID, "img/alpha", 0))
	require.Contains(t, libraryRepo.bySourceKey, BatchImageLibrarySourceKey(job.BatchID, "img/alpha", 1))
	require.Equal(t, "batch_image/imgbatch_library_success/bcf5a9c4c14f00850b0ec1361503654fd1afeeaad8064be821e6bad690b410ae/1", BatchImageLibrarySourceKey(job.BatchID, "img/alpha", 1))

	// Completed ingest state avoids reopening/scanning the provider output.
	provider.openResultCalled = false
	result, err = processor.Process(context.Background(), job.BatchID)
	require.NoError(t, err)
	require.True(t, result.Terminal)
	require.False(t, provider.openResultCalled)
	require.Equal(t, 2, libraryRepo.created)
}

func TestBatchImageLibraryIngestScannerAcceptsLineNearConfiguredFileLimit(t *testing.T) {
	// Thirteen MiB expands past the former fixed 16 MiB Scanner ceiling. Keep
	// this row provider-extra so the test isolates framing/parsing from image
	// validation while still exercising the real ingestion scan.
	const maxUploadBytes = int64(13 << 20)
	encodedBytes := base64.StdEncoding.EncodedLen(int(maxUploadBytes - 1))
	provider := &fakeProcessorProvider{result: `{"key":"provider-extra","response":{"candidates":[{"content":{"parts":[{"inlineData":{"mimeType":"image/png","data":"` +
		strings.Repeat("A", encodedBytes) + `"}}]}}]}}` + "\n"}
	accountID := int64(17)
	job := &BatchImageJob{BatchID: "imgbatch_library_near_limit", UserID: 9, AccountID: &accountID, Provider: "fake", Status: BatchImageJobStatusCompleted}
	batchRepo := newFakeBatchImageRepository()
	library := NewLibraryService(&generatedLibraryRepoFake{}, &libraryBlobStoreFake{}, &config.Config{
		Library: config.LibraryConfig{MaxFileBytes: maxUploadBytes},
	})
	ingestor := &BatchImageLibraryIngestor{
		Repo: batchRepo, ProviderRegistry: NewBatchImageProviderRegistry(provider),
		AccountResolver: &fakeBatchImageAccountResolver{account: &Account{}}, Library: library,
	}

	attempt := ingestor.ingestAttempt(context.Background(), job)
	require.Empty(t, attempt.ErrorCode)
	require.Zero(t, attempt.FailedCount)
	require.Greater(t, len(provider.result), 16<<20)
	require.Less(t, len(provider.result), batchImageLibraryResultLineLimit(maxUploadBytes))
}

func TestBatchImageLibraryResultLineLimitIsDerivedAndAbsolutelyBounded(t *testing.T) {
	const maxUploadBytes = int64(13 << 20)
	require.Equal(t,
		base64.StdEncoding.EncodedLen(int(maxUploadBytes))+batchImageLibraryResultJSONOverheadBytes,
		batchImageLibraryResultLineLimit(maxUploadBytes),
	)
	require.Greater(t, batchImageLibraryResultLineLimit(maxUploadBytes), 16<<20)
	require.Equal(t, batchImageLibraryResultAbsoluteMaxLineBytes, batchImageLibraryResultLineLimit(1<<62))
}

func TestLibrarySaveGeneratedIdempotentSoftDeleteSuppressesResurrection(t *testing.T) {
	repo := &generatedLibraryRepoFake{quotaErr: ErrChatAttachmentRateLimit}
	library := NewLibraryService(repo, &libraryBlobStoreFake{}, nil)
	input := LibraryUpload{Filename: "result.png", DeclaredMIME: "image/png", Data: encodeBatchImageLibraryPNG(t)}
	key := BatchImageLibrarySourceKey("imgbatch_delete", "cover", 0)

	first, err := library.SaveGeneratedIdempotent(context.Background(), 5, input, key)
	require.NoError(t, err)
	require.NotNil(t, first.File)
	_, err = repo.MarkDeleted(context.Background(), 5, first.File.ID)
	require.NoError(t, err)

	replayed, err := library.SaveGeneratedIdempotent(context.Background(), 5, input, key)
	require.NoError(t, err)
	require.True(t, replayed.Suppressed)
	require.Nil(t, replayed.File)
	require.Equal(t, 1, repo.created)
	require.Zero(t, repo.quotaCalls)
}

func TestBatchImageLibraryIngestRetriesTransientFailureBoundedly(t *testing.T) {
	imageData := encodeBatchImageLibraryPNG(t)
	provider := &fakeProcessorProvider{result: batchImageLibraryResultLine("retry", imageData)}
	accountID := int64(3)
	job := &BatchImageJob{BatchID: "imgbatch_retry", UserID: 2, AccountID: &accountID, Provider: "fake", Status: BatchImageJobStatusCompleted}
	batchRepo := newFakeBatchImageRepository()
	batchRepo.jobs[job.BatchID] = job
	batchRepo.items[job.BatchID] = []CreateBatchImageItemParams{{
		JobID: job.BatchID, CustomID: "retry", Status: BatchImageItemStatusSuccess, ImageCount: 1,
	}}
	state := &batchImageLibraryStateFake{}
	ingestor := &BatchImageLibraryIngestor{
		Repo: batchRepo, StateRepo: state, ProviderRegistry: NewBatchImageProviderRegistry(provider),
		AccountResolver: &fakeBatchImageAccountResolver{account: &Account{}},
		Library:         NewLibraryService(&generatedLibraryRepoFake{}, &failingLibraryBlobStore{putErr: errors.New("storage offline")}, nil),
		MaxAttempts:     3, RetryDelay: 25 * time.Millisecond,
	}

	first := ingestor.Ingest(context.Background(), job)
	second := ingestor.Ingest(context.Background(), job)
	third := ingestor.Ingest(context.Background(), job)
	require.False(t, first.Done)
	require.False(t, second.Done)
	require.Equal(t, 25*time.Millisecond, first.RetryAfter)
	require.True(t, third.Done)
	require.Equal(t, 2, state.retries)
	require.Equal(t, 1, state.completed)
	require.Equal(t, BatchImageLibraryIngestCompletedWithErrors, state.state.Status)
	require.Equal(t, 3, state.state.Attempts)
	require.Equal(t, "LIBRARY_GENERATED_SAVE_RETRYABLE", state.state.LastErrorCode)

	// Terminal state means no fourth provider scan and no infinite requeue.
	provider.openResultCalled = false
	require.True(t, ingestor.Ingest(context.Background(), job).Done)
	require.False(t, provider.openResultCalled)
}

func TestBatchImageLibraryIngestDeterministicQuotaFailureIsTerminal(t *testing.T) {
	provider := &fakeProcessorProvider{result: batchImageLibraryResultLine("quota", encodeBatchImageLibraryPNG(t))}
	accountID := int64(3)
	job := &BatchImageJob{BatchID: "imgbatch_quota", UserID: 2, AccountID: &accountID, Provider: "fake", Status: BatchImageJobStatusCompleted}
	batchRepo := newFakeBatchImageRepository()
	batchRepo.items[job.BatchID] = []CreateBatchImageItemParams{{
		JobID: job.BatchID, CustomID: "quota", Status: BatchImageItemStatusSuccess, ImageCount: 1,
	}}
	state := &batchImageLibraryStateFake{}
	libraryRepo := &generatedLibraryRepoFake{}
	libraryRepo.createErr = ErrLibraryStorageLimit
	ingestor := &BatchImageLibraryIngestor{
		Repo: batchRepo, StateRepo: state, ProviderRegistry: NewBatchImageProviderRegistry(provider),
		AccountResolver: &fakeBatchImageAccountResolver{account: &Account{}},
		Library:         NewLibraryService(libraryRepo, &libraryBlobStoreFake{}, nil), MaxAttempts: 3,
	}

	result := ingestor.Ingest(context.Background(), job)
	require.True(t, result.Done)
	require.Zero(t, result.RetryAfter)
	require.Zero(t, state.retries)
	require.Equal(t, 1, state.completed)
	require.Equal(t, 1, state.state.Attempts)
	require.Equal(t, BatchImageLibraryIngestCompletedWithErrors, state.state.Status)
	require.Equal(t, "LIBRARY_STORAGE_LIMIT_EXCEEDED", state.state.LastErrorCode)
}

func batchImageLibraryResultLine(customID string, data []byte) string {
	return `{"key":"` + customID + `","response":{"candidates":[{"content":{"parts":[{"inlineData":{"mimeType":"image/png","data":"` +
		base64.StdEncoding.EncodeToString(data) + `"}}]}}]}}` + "\n"
}

func encodeBatchImageLibraryPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.SetNRGBA(0, 0, color.NRGBA{R: 0x33, G: 0x66, B: 0x99, A: 0xff})
	var output bytes.Buffer
	require.NoError(t, png.Encode(&output, img))
	return output.Bytes()
}
