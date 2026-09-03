package service

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

const (
	BatchImageLibraryIngestPending             = "pending"
	BatchImageLibraryIngestRetrying            = "retrying"
	BatchImageLibraryIngestCompleted           = "completed"
	BatchImageLibraryIngestCompletedWithErrors = "completed_with_errors"

	defaultBatchImageLibraryIngestMaxAttempts = 3
	defaultBatchImageLibraryIngestRetryDelay  = time.Minute
	batchImageLibraryStateWriteTimeout        = 15 * time.Second
	batchImageLibraryEventWriteTimeout        = 5 * time.Second
	batchImageLibraryResultJSONOverheadBytes  = 1 << 20
	// Config validation caps Library.MaxFileBytes at 100 MiB. Its padded
	// base64 representation is about 134 MiB, so 144 MiB admits every valid
	// configured image plus JSON metadata while still bounding Scanner and
	// json.Unmarshal memory if a service is constructed with invalid config.
	batchImageLibraryResultAbsoluteMaxLineBytes = 144 << 20
)

type BatchImageLibraryIngestState struct {
	JobID            string
	Status           string
	Attempts         int
	ImportedCount    int
	SuppressedCount  int
	FailedCount      int
	LastErrorCode    string
	LastErrorMessage string
}

type BatchImageLibraryIngestAttempt struct {
	ImportedCount   int
	SuppressedCount int
	FailedCount     int
	ErrorCode       string
	ErrorMessage    string
}

// BatchImageLibraryIngestStateRepository is deliberately separate from the
// core batch repository interface: library ingestion is an optional,
// post-settlement concern and cannot participate in billing correctness.
type BatchImageLibraryIngestStateRepository interface {
	BeginBatchImageLibraryIngest(ctx context.Context, batchID string, maxAttempts int) (*BatchImageLibraryIngestState, bool, error)
	RetryBatchImageLibraryIngest(ctx context.Context, batchID string, attempt BatchImageLibraryIngestAttempt) error
	CompleteBatchImageLibraryIngest(ctx context.Context, batchID, status string, attempt BatchImageLibraryIngestAttempt) error
}

type BatchImageLibraryIngestResult struct {
	RetryAfter time.Duration
	Done       bool
}

type BatchImageLibraryIngestor struct {
	Repo             BatchImageRepository
	StateRepo        BatchImageLibraryIngestStateRepository
	ProviderRegistry *BatchImageProviderRegistry
	AccountResolver  BatchImageAccountResolver
	Library          *LibraryService
	MaxAttempts      int
	RetryDelay       time.Duration
}

// Ingest is best-effort by design. Once billing has completed, every failure
// is converted into either a bounded retry or a terminal observable state; it
// never returns an error that can undo settlement or poison the worker queue.
func (s *BatchImageLibraryIngestor) Ingest(ctx context.Context, job *BatchImageJob) BatchImageLibraryIngestResult {
	if s == nil || job == nil || job.Status != BatchImageJobStatusCompleted || s.Library == nil || s.StateRepo == nil {
		return BatchImageLibraryIngestResult{Done: true}
	}
	maxAttempts := s.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = defaultBatchImageLibraryIngestMaxAttempts
	}
	state, started, err := s.StateRepo.BeginBatchImageLibraryIngest(ctx, job.BatchID, maxAttempts)
	if err != nil {
		s.logFailure(job.BatchID, "state_begin_failed", err)
		// No durable state may exist yet, so ACK would permanently lose a
		// settled result. Keep the queue item recoverable; once the database is
		// reachable, Begin persists the bounded attempt count before any provider
		// output is scanned.
		return s.retryOrFinish(0, maxAttempts)
	}
	if state == nil || !started {
		return BatchImageLibraryIngestResult{Done: true}
	}
	s.event(ctx, job.BatchID, "library_ingest_started", map[string]any{"attempt": state.Attempts})

	attempt := s.ingestAttempt(ctx, job)
	if attempt.ErrorCode != "" && isRetryableBatchImageLibraryErrorCode(attempt.ErrorCode) && state.Attempts < maxAttempts {
		writeCtx, cancelWrite := batchImageLibraryDurableContext(ctx, batchImageLibraryStateWriteTimeout)
		err := s.StateRepo.RetryBatchImageLibraryIngest(writeCtx, job.BatchID, attempt)
		cancelWrite()
		if err != nil {
			s.logFailure(job.BatchID, "state_retry_failed", err)
			return s.retryAfterStateWriteFailure()
		}
		s.event(ctx, job.BatchID, "library_ingest_retry_scheduled", map[string]any{
			"attempt": state.Attempts, "error_code": attempt.ErrorCode,
		})
		return s.retryOrFinish(state.Attempts, maxAttempts)
	}

	status := BatchImageLibraryIngestCompleted
	if attempt.FailedCount > 0 || attempt.ErrorCode != "" {
		status = BatchImageLibraryIngestCompletedWithErrors
	}
	writeCtx, cancelWrite := batchImageLibraryDurableContext(ctx, batchImageLibraryStateWriteTimeout)
	err = s.StateRepo.CompleteBatchImageLibraryIngest(writeCtx, job.BatchID, status, attempt)
	cancelWrite()
	if err != nil {
		s.logFailure(job.BatchID, "state_complete_failed", err)
		// Even on the final content attempt, requeue once so the next Begin can
		// durably terminalize attempts>=max under its row lock. ACKing here would
		// strand a pending/retrying state with no recovery scanner.
		return s.retryAfterStateWriteFailure()
	}
	s.event(ctx, job.BatchID, "library_ingest_completed", map[string]any{
		"status": status, "attempt": state.Attempts, "imported_count": attempt.ImportedCount,
		"suppressed_count": attempt.SuppressedCount, "failed_count": attempt.FailedCount,
		"error_code": attempt.ErrorCode,
	})
	return BatchImageLibraryIngestResult{Done: true}
}

func (s *BatchImageLibraryIngestor) retryOrFinish(attempts, maxAttempts int) BatchImageLibraryIngestResult {
	if attempts >= maxAttempts {
		return BatchImageLibraryIngestResult{Done: true}
	}
	delay := s.RetryDelay
	if delay <= 0 {
		delay = defaultBatchImageLibraryIngestRetryDelay
	}
	return BatchImageLibraryIngestResult{RetryAfter: delay}
}

func (s *BatchImageLibraryIngestor) retryAfterStateWriteFailure() BatchImageLibraryIngestResult {
	delay := s.RetryDelay
	if delay <= 0 {
		delay = defaultBatchImageLibraryIngestRetryDelay
	}
	return BatchImageLibraryIngestResult{RetryAfter: delay}
}

func (s *BatchImageLibraryIngestor) ingestAttempt(ctx context.Context, job *BatchImageJob) BatchImageLibraryIngestAttempt {
	attempt := BatchImageLibraryIngestAttempt{}
	indexed, err := s.listIndexedSuccesses(ctx, job.BatchID)
	if err != nil {
		attempt.FailedCount = 1
		attempt.ErrorCode = "BATCH_IMAGE_LIBRARY_STATE_UNAVAILABLE"
		attempt.ErrorMessage = "indexed batch results are temporarily unavailable"
		return attempt
	}
	provider, account, err := s.providerAndAccount(ctx, job)
	if err != nil {
		attempt.FailedCount = 1
		attempt.ErrorCode = "BATCH_IMAGE_LIBRARY_PROVIDER_UNAVAILABLE"
		attempt.ErrorMessage = "batch result provider is temporarily unavailable"
		return attempt
	}
	r, _, err := provider.OpenResult(ctx, job, account)
	if err != nil {
		attempt.FailedCount = 1
		attempt.ErrorCode = "BATCH_IMAGE_LIBRARY_OUTPUT_UNAVAILABLE"
		attempt.ErrorMessage = "batch result output is temporarily unavailable"
		return attempt
	}
	defer func() { _ = r.Close() }()

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), batchImageLibraryResultLineLimit(s.Library.MaxUploadBytes()))
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			attempt.FailedCount++
			attempt.ErrorCode = "BATCH_IMAGE_LIBRARY_CONTEXT_CANCELLED"
			attempt.ErrorMessage = "batch library ingestion was interrupted"
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts, parseErr := ExtractBatchImagePartsFromResultLine([]byte(line))
		if parseErr != nil {
			attempt.FailedCount++
			setFirstBatchImageLibraryError(&attempt, "BATCH_IMAGE_LIBRARY_RESULT_INVALID", "batch result output could not be parsed")
			continue
		}
		indexedImageCount, expected := indexed[parts.CustomID]
		if !expected {
			// The indexer has already reconciled the provider output against the
			// submitted manifest. Never import unknown/provider-extra rows.
			continue
		}
		for imageIndex, image := range parts.Images {
			if imageIndex >= indexedImageCount {
				continue
			}
			data, decodeErr := decodeBatchImageLibraryData(image.Base64Data, s.Library.MaxUploadBytes())
			if decodeErr != nil {
				attempt.FailedCount++
				setFirstBatchImageLibraryError(&attempt, "BATCH_IMAGE_LIBRARY_IMAGE_INVALID", "generated image data is invalid")
				continue
			}
			extension := strings.TrimSpace(image.Extension)
			if extension == "" {
				extension = batchImageFileExtension(image.MimeType)
			}
			filename := batchImageGeneratedFilename(parts.CustomID, imageIndex, extension)
			sourceKey := BatchImageLibrarySourceKey(job.BatchID, parts.CustomID, imageIndex)
			saved, saveErr := s.Library.SaveGeneratedIdempotent(ctx, job.UserID, LibraryUpload{
				Filename: filename, DeclaredMIME: image.MimeType, Data: data,
			}, sourceKey)
			if saveErr != nil {
				attempt.FailedCount++
				code, message, retryable := classifyBatchImageLibrarySaveError(saveErr)
				setFirstBatchImageLibraryError(&attempt, code, message)
				if retryable && !isRetryableBatchImageLibraryErrorCode(attempt.ErrorCode) {
					attempt.ErrorCode, attempt.ErrorMessage = code, message
				}
				continue
			}
			if saved != nil && saved.Suppressed {
				attempt.SuppressedCount++
			} else {
				attempt.ImportedCount++
			}
		}
	}
	if scanErr := scanner.Err(); scanErr != nil {
		attempt.FailedCount++
		attempt.ErrorCode = "BATCH_IMAGE_LIBRARY_OUTPUT_UNAVAILABLE"
		attempt.ErrorMessage = "batch result output could not be read"
	}
	return attempt
}

func batchImageLibraryResultLineLimit(maxUploadBytes int64) int {
	if maxUploadBytes <= 0 {
		maxUploadBytes = 20 << 20
	}
	// Clamp before converting to int or calling EncodedLen. This keeps the
	// helper safe even when LibraryService was built directly and bypassed
	// normal config validation.
	maxEncodedBytes := int64(batchImageLibraryResultAbsoluteMaxLineBytes - batchImageLibraryResultJSONOverheadBytes)
	maxDecodedBytes := (maxEncodedBytes / 4) * 3
	if maxUploadBytes > maxDecodedBytes {
		maxUploadBytes = maxDecodedBytes
	}
	encodedBytes := base64.StdEncoding.EncodedLen(int(maxUploadBytes))
	lineBytes := encodedBytes + batchImageLibraryResultJSONOverheadBytes
	if lineBytes > batchImageLibraryResultAbsoluteMaxLineBytes {
		return batchImageLibraryResultAbsoluteMaxLineBytes
	}
	return lineBytes
}

func (s *BatchImageLibraryIngestor) listIndexedSuccesses(ctx context.Context, batchID string) (map[string]int, error) {
	if s == nil || s.Repo == nil {
		return nil, errors.New("batch image repository is unavailable")
	}
	const pageSize = 500
	result := make(map[string]int)
	for offset := 0; ; {
		items, err := s.Repo.ListBatchImageItems(ctx, batchID, BatchImageItemFilter{
			Status: BatchImageItemStatusSuccess, Limit: pageSize, Offset: offset,
		})
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if item != nil && item.ImageCount > 0 {
				result[item.CustomID] = item.ImageCount
			}
		}
		if len(items) < pageSize {
			return result, nil
		}
		offset += len(items)
	}
}

func (s *BatchImageLibraryIngestor) providerAndAccount(ctx context.Context, job *BatchImageJob) (BatchImageProvider, *Account, error) {
	if s.ProviderRegistry == nil || s.AccountResolver == nil || job == nil || job.AccountID == nil || *job.AccountID <= 0 {
		return nil, nil, errors.New("batch image library provider is not configured")
	}
	provider, ok := s.ProviderRegistry.Get(job.Provider)
	if !ok || provider == nil {
		return nil, nil, ErrBatchImageUnsupportedProvider
	}
	account, err := s.AccountResolver.ResolveBatchImageAccount(ctx, *job.AccountID)
	if err != nil || account == nil || !provider.SupportsAccount(account) {
		return nil, nil, errors.New("batch image library account is unavailable")
	}
	return provider, account, nil
}

func BatchImageLibrarySourceKey(batchID, customID string, imageIndex int) string {
	// Hash the provider-controlled custom id so the persisted key stays inside
	// the schema bound even if a future provider accepts very long identifiers.
	digest := sha256.Sum256([]byte(strings.TrimSpace(customID)))
	return "batch_image/" + strings.TrimSpace(batchID) + "/" + hex.EncodeToString(digest[:]) + "/" + strconv.Itoa(imageIndex)
}

func batchImageGeneratedFilename(customID string, imageIndex int, extension string) string {
	base := sanitizeBatchImageFilenameBase(customID)
	if imageIndex > 0 {
		base = fmt.Sprintf("%s_%d", base, imageIndex+1)
	}
	return BatchImageSafeDownloadFilename(base, extension)
}

func decodeBatchImageLibraryData(encoded string, maxBytes int64) ([]byte, error) {
	if maxBytes <= 0 {
		maxBytes = 20 << 20
	}
	decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(strings.TrimSpace(encoded)))
	data, err := io.ReadAll(io.LimitReader(decoder, maxBytes+1))
	if err != nil || len(data) == 0 || int64(len(data)) > maxBytes {
		return nil, errors.New("invalid or oversized generated image")
	}
	return data, nil
}

func classifyBatchImageLibrarySaveError(err error) (string, string, bool) {
	switch {
	case errors.Is(err, ErrLibraryStorageLimit):
		return "LIBRARY_STORAGE_LIMIT_EXCEEDED", "library storage limit exceeded", false
	case errors.Is(err, ErrLibraryFileTooLarge):
		return "LIBRARY_FILE_TOO_LARGE", "generated image exceeds the library file limit", false
	case errors.Is(err, ErrLibraryFileUnsupported):
		return "LIBRARY_FILE_TYPE_NOT_SUPPORTED", "generated image type is not supported", false
	case errors.Is(err, ErrLibraryInvalidRequest):
		return "LIBRARY_GENERATED_FILE_INVALID", "generated image did not pass library validation", false
	case errors.Is(err, ErrLibraryUnavailable), errors.Is(err, ErrLibraryUploadFailed):
		return "LIBRARY_GENERATED_SAVE_RETRYABLE", "generated image could not be saved temporarily", true
	default:
		if infraerrors.Code(err) >= http.StatusInternalServerError {
			return "LIBRARY_GENERATED_SAVE_RETRYABLE", "generated image could not be saved temporarily", true
		}
		return "LIBRARY_GENERATED_SAVE_FAILED", "generated image could not be saved", false
	}
}

func isRetryableBatchImageLibraryErrorCode(code string) bool {
	switch code {
	case "BATCH_IMAGE_LIBRARY_PROVIDER_UNAVAILABLE", "BATCH_IMAGE_LIBRARY_OUTPUT_UNAVAILABLE",
		"BATCH_IMAGE_LIBRARY_STATE_UNAVAILABLE", "BATCH_IMAGE_LIBRARY_CONTEXT_CANCELLED", "LIBRARY_GENERATED_SAVE_RETRYABLE":
		return true
	default:
		return false
	}
}

func setFirstBatchImageLibraryError(attempt *BatchImageLibraryIngestAttempt, code, message string) {
	if attempt != nil && attempt.ErrorCode == "" {
		attempt.ErrorCode, attempt.ErrorMessage = code, message
	}
}

func (s *BatchImageLibraryIngestor) event(ctx context.Context, batchID, eventType string, payload any) {
	if s == nil || s.Repo == nil {
		return
	}
	eventCtx, cancel := batchImageLibraryDurableContext(ctx, batchImageLibraryEventWriteTimeout)
	defer cancel()
	if err := s.Repo.AppendBatchImageEvent(eventCtx, batchID, eventType, payload); err != nil {
		s.logFailure(batchID, "event_write_failed", err)
	}
}

func batchImageLibraryDurableContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(context.WithoutCancel(parent), timeout)
}

func (s *BatchImageLibraryIngestor) logFailure(batchID, operation string, err error) {
	logger.L().Warn("batch_image.library_ingest_failed",
		zap.String("batch_id", batchID), zap.String("operation", operation), zap.Error(err))
}
