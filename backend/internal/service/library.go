package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"log/slog"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/chatattachment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"golang.org/x/image/draw"
	"golang.org/x/sync/singleflight"
)

const (
	LibraryFileStatusPending = "pending"
	LibraryFileStatusReady   = "ready"
	LibraryFileStatusDeleted = "deleted"

	libraryThumbnailMaxDimension = 512
	maxLibrarySourceKeyBytes     = 512
	libraryImageRenderTimeout    = 2 * time.Minute
	hardLibraryBatchMaxBytes     = int64(100 << 20)
	defaultLibraryPageSize       = 20
	maxLibraryPageSize           = 100
	maxLibrarySearchRunes        = 100
)

var (
	ErrLibraryFileNotFound       = infraerrors.NotFound("LIBRARY_FILE_NOT_FOUND", "Library file not found")
	ErrLibraryFileAccessDenied   = infraerrors.Forbidden("LIBRARY_FILE_ACCESS_DENIED", "You do not have access to this library file")
	ErrLibraryInvalidRequest     = infraerrors.BadRequest("LIBRARY_INVALID_REQUEST", "The library request is invalid")
	ErrLibraryFileUnsupported    = infraerrors.New(415, "LIBRARY_FILE_TYPE_NOT_SUPPORTED", "This file type is not supported")
	ErrLibraryPreviewUnsupported = infraerrors.New(415, "LIBRARY_PREVIEW_NOT_SUPPORTED", "This file type cannot be previewed")
	ErrLibraryStorageLimit       = infraerrors.New(507, "LIBRARY_STORAGE_LIMIT_EXCEEDED", "Library storage limit exceeded")
	ErrLibraryBatchLimit         = infraerrors.BadRequest("LIBRARY_BATCH_LIMIT_EXCEEDED", "Too many library files were selected")
	ErrLibraryBatchTooLarge      = infraerrors.New(413, "LIBRARY_BATCH_TOO_LARGE", "The selected library files exceed the batch download size limit")
	ErrLibraryFileTooLarge       = infraerrors.New(413, "LIBRARY_FILE_TOO_LARGE", "The library file exceeds the allowed size")
	ErrLibraryUploadFailed       = infraerrors.New(503, "LIBRARY_UPLOAD_FAILED", "The library file could not be saved")
	ErrLibraryDownloadFailed     = infraerrors.New(503, "LIBRARY_DOWNLOAD_FAILED", "The library file could not be downloaded")
	ErrLibraryDeleteFailed       = infraerrors.New(503, "LIBRARY_DELETE_FAILED", "The library file could not be deleted")
	ErrLibraryUnavailable        = infraerrors.ServiceUnavailable("LIBRARY_UNAVAILABLE", "The file library is temporarily unavailable")
)

// LibraryFile is the public file metadata plus repository-only fields. Fields
// tagged json:"-" must never be exposed by a handler.
type LibraryFile struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	MIMEType    string    `json:"mime_type"`
	Extension   string    `json:"extension"`
	Size        int64     `json:"size"`
	Category    string    `json:"category"`
	Type        string    `json:"type"`
	Source      string    `json:"source"`
	Status      string    `json:"status"`
	PageCount   int       `json:"page_count,omitempty"`
	Width       int       `json:"width,omitempty"`
	Height      int       `json:"height,omitempty"`
	Previewable bool      `json:"previewable"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	InternalID    int64      `json:"-"`
	UserID        int64      `json:"-"`
	Format        string     `json:"-"`
	StoredSize    int64      `json:"-"`
	StorageKey    string     `json:"-"`
	SourceKey     string     `json:"-"`
	Digest        string     `json:"-"`
	ExtractedText string     `json:"-"` // legacy compatibility; library persistence never writes content
	LastUsedAt    *time.Time `json:"-"`
	DeletedAt     *time.Time `json:"-"`
	CleanedAt     *time.Time `json:"-"`
}

type LibraryFileQuery struct {
	Q        string
	Category string
	Source   string
	Type     string
	Sort     string
	Page     int
	PageSize int
}

type LibraryFilePage struct {
	Files    []LibraryFile `json:"files"`
	Total    int64         `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
}

type LibraryStorage struct {
	UsedBytes      int64 `json:"used_bytes"`
	LimitBytes     int64 `json:"limit_bytes"`
	RemainingBytes int64 `json:"remaining_bytes"`
	OverLimit      bool  `json:"over_limit"`
}

type CreateLibraryFileInput struct {
	UserID         int64
	File           LibraryFile
	AliasExpiresAt time.Time
	// SourceKey is a stable producer identity for generated files. BypassUploadQuota
	// is restricted to trusted internal producers; storage entitlement is still
	// reserved atomically by CreatePending.
	SourceKey         string
	BypassUploadQuota bool
}

type LibraryFileRepository interface {
	ResolveStorageLimit(ctx context.Context, userID, defaultBytes int64, groupOverrides map[int64]int64) (int64, error)
	CheckUploadQuota(ctx context.Context, userID, rawSize int64) error
	CreatePending(ctx context.Context, input *CreateLibraryFileInput, limitBytes int64) (*LibraryFile, error)
	Activate(ctx context.Context, userID int64, publicID string) (*LibraryFile, error)
	Abort(ctx context.Context, userID int64, publicID string) (*LibraryFile, error)
	GetOwned(ctx context.Context, userID int64, publicID string) (*LibraryFile, error)
	List(ctx context.Context, userID int64, query LibraryFileQuery) (*LibraryFilePage, error)
	ResolveOwned(ctx context.Context, userID int64, publicIDs []string) ([]LibraryFile, error)
	MarkDeleted(ctx context.Context, userID int64, publicID string) (*LibraryFile, error)
	Usage(ctx context.Context, userID int64) (int64, error)
	ClaimCleanup(ctx context.Context, deletedBefore, pendingBefore time.Time, limit int) ([]LibraryFile, error)
	FinalizeCleanup(ctx context.Context, userID int64, publicID string) error
	ListStorageKeys(ctx context.Context) (map[string]struct{}, error)
}

type LibraryUpload struct {
	Filename     string
	DeclaredMIME string
	Data         []byte
}

type GeneratedLibrarySaveResult struct {
	File          *LibraryFile
	AlreadyExists bool
	Suppressed    bool
}

type LibraryOpenFile struct {
	File *LibraryFile
	Body io.ReadCloser
}

type libraryImageRendition struct {
	data     []byte
	mimeType string
}

type LibraryService struct {
	repo  LibraryFileRepository
	store LibraryBlobStore
	cfg   config.LibraryConfig

	groupOverrides map[int64]int64
	configErr      error

	mu           sync.Mutex
	cancel       context.CancelFunc
	done         chan struct{}
	storeClose   sync.Once
	activeGlobal int
	activeByUser map[int64]int
	chatLimits   config.ChatAttachmentConfig
	renderGate   chan struct{}
	renderFlight singleflight.Group
}

func NewLibraryService(repo LibraryFileRepository, store LibraryBlobStore, cfg *config.Config) *LibraryService {
	libraryCfg := config.LibraryConfig{
		StorageDriver: "local", StorageDir: "./data/library-files", MaxFileBytes: 20 << 20,
		BatchDownloadLimit: 100, BatchDownloadMaxBytes: 100 << 20,
		DeletedRetentionDays: 30, PendingUploadStaleMinutes: 60, CleanupIntervalMinutes: 60, CleanupBatchSize: 100,
	}
	chatLimits := config.ChatAttachmentConfig{
		MaxImageBytes: 10 << 20, MaxDocumentBytes: 20 << 20,
		MaxConcurrentGlobal: 4, MaxConcurrentPerUser: 2,
	}
	if cfg != nil {
		libraryCfg = cfg.Library
		chatLimits = cfg.ChatAttachments
	}
	overrides, configErr := parseLibraryGroupOverrides(libraryCfg.SubscriptionGroupStorageBytes)
	return &LibraryService{
		repo: repo, store: store, cfg: libraryCfg, groupOverrides: overrides, configErr: configErr,
		activeByUser: make(map[int64]int), chatLimits: chatLimits,
		renderGate: make(chan struct{}, max(1, min(2, chatLimits.MaxConcurrentGlobal))),
	}
}

func parseLibraryGroupOverrides(values map[string]int64) (map[int64]int64, error) {
	result := make(map[int64]int64, len(values))
	for raw, limit := range values {
		id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
		if err != nil || id <= 0 || limit < 0 {
			return nil, fmt.Errorf("invalid library subscription group override %q", raw)
		}
		result[id] = limit
	}
	return result, nil
}

func (s *LibraryService) MaxUploadBytes() int64 {
	if s == nil || s.cfg.MaxFileBytes <= 0 {
		return 20 << 20
	}
	return s.cfg.MaxFileBytes
}

func (s *LibraryService) BatchLimit() int {
	if s == nil || s.cfg.BatchDownloadLimit <= 0 {
		return 100
	}
	return s.cfg.BatchDownloadLimit
}

func (s *LibraryService) BatchMaxBytes() int64 {
	if s == nil || s.cfg.BatchDownloadMaxBytes <= 0 {
		return hardLibraryBatchMaxBytes
	}
	return min(s.cfg.BatchDownloadMaxBytes, hardLibraryBatchMaxBytes)
}

func (s *LibraryService) AdmitUpload(userID int64) (func(), error) {
	if s == nil || userID <= 0 {
		return nil, ErrLibraryUnavailable
	}
	globalLimit := s.chatLimits.MaxConcurrentGlobal
	if globalLimit <= 0 {
		globalLimit = 4
	}
	perUserLimit := s.chatLimits.MaxConcurrentPerUser
	if perUserLimit <= 0 {
		perUserLimit = 2
	}
	s.mu.Lock()
	if s.activeGlobal >= globalLimit || s.activeByUser[userID] >= perUserLimit {
		s.mu.Unlock()
		return nil, ErrChatAttachmentRateLimit
	}
	s.activeGlobal++
	s.activeByUser[userID]++
	s.mu.Unlock()
	return func() {
		s.mu.Lock()
		s.activeGlobal--
		s.activeByUser[userID]--
		if s.activeByUser[userID] == 0 {
			delete(s.activeByUser, userID)
		}
		s.mu.Unlock()
	}, nil
}

func (s *LibraryService) Upload(ctx context.Context, userID int64, input LibraryUpload, source string) (*LibraryFile, error) {
	release, err := s.AdmitUpload(userID)
	if err != nil {
		return nil, err
	}
	defer release()
	return s.UploadAdmitted(ctx, userID, input, source)
}

// SaveGenerated is the extension point for real generated-file pipelines. It
// shares the exact same validation, quota and persistence path as user uploads.
func (s *LibraryService) SaveGenerated(ctx context.Context, userID int64, input LibraryUpload) (*LibraryFile, error) {
	return s.Upload(ctx, userID, input, "generated")
}

// SaveGeneratedIdempotent is for trusted durable producer pipelines. It skips
// interactive upload frequency/concurrency limits, while retaining content,
// file-size and atomically reserved storage-entitlement checks. A source key on
// a soft-deleted row is a permanent suppression marker and is never revived.
func (s *LibraryService) SaveGeneratedIdempotent(ctx context.Context, userID int64, input LibraryUpload, sourceKey string) (*GeneratedLibrarySaveResult, error) {
	sourceKey = strings.TrimSpace(sourceKey)
	if sourceKey == "" || len(sourceKey) > maxLibrarySourceKeyBytes {
		return nil, ErrLibraryInvalidRequest
	}
	return s.uploadAdmitted(ctx, userID, input, "generated", sourceKey, true)
}

func (s *LibraryService) UploadAdmitted(ctx context.Context, userID int64, input LibraryUpload, source string) (*LibraryFile, error) {
	result, err := s.uploadAdmitted(ctx, userID, input, source, "", false)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrLibraryUploadFailed
	}
	return result.File, nil
}

func (s *LibraryService) uploadAdmitted(ctx context.Context, userID int64, input LibraryUpload, source, sourceKey string, bypassUploadQuota bool) (*GeneratedLibrarySaveResult, error) {
	if s == nil || s.repo == nil || s.store == nil || userID <= 0 || s.configErr != nil {
		return nil, ErrLibraryUnavailable.WithCause(s.configErr)
	}
	name, err := safeLibraryFilename(input.Filename)
	if err != nil || len(input.Data) == 0 {
		return nil, ErrLibraryInvalidRequest
	}
	if int64(len(input.Data)) > s.MaxUploadBytes() {
		return nil, ErrLibraryFileTooLarge
	}
	source = strings.ToLower(strings.TrimSpace(source))
	if source == "" {
		source = "uploaded"
	}
	if source != "uploaded" && source != "generated" {
		return nil, ErrLibraryInvalidRequest
	}
	if !bypassUploadQuota {
		if err = s.repo.CheckUploadQuota(ctx, userID, int64(len(input.Data))); err != nil {
			if errors.Is(err, ErrChatAttachmentRateLimit) {
				return nil, err
			}
			return nil, ErrLibraryUploadFailed.WithCause(err)
		}
	}
	parsed, err := chatattachment.ProcessLibrary(chatattachment.Input{
		Filename: name, DeclaredMIME: input.DeclaredMIME, Data: input.Data,
	})
	if err != nil {
		switch chatattachment.CodeOf(err) {
		case chatattachment.CodeUnsupportedType, chatattachment.CodeTypeMismatch:
			return nil, ErrLibraryFileUnsupported.WithMetadata(map[string]string{"attachment_error": string(chatattachment.CodeOf(err))}).WithCause(err)
		default:
			return nil, ErrLibraryInvalidRequest.WithMetadata(map[string]string{"attachment_error": string(chatattachment.CodeOf(err))}).WithCause(err)
		}
	}
	if parsed.Category == chatattachment.LibraryCategoryImage && int64(len(input.Data)) > effectiveLibraryImageLimit(s.chatLimits, s.MaxUploadBytes()) {
		return nil, ErrLibraryFileTooLarge
	}
	limit, err := s.repo.ResolveStorageLimit(ctx, userID, s.cfg.DefaultStorageBytes, s.groupOverrides)
	if err != nil {
		return nil, ErrLibraryUnavailable.WithCause(err)
	}
	publicID, err := newLibraryFilePublicID()
	if err != nil {
		return nil, ErrLibraryUploadFailed.WithCause(err)
	}
	key, err := newLibraryStorageKey(userID)
	if err != nil {
		return nil, ErrLibraryUploadFailed.WithCause(err)
	}
	digest := sha256.Sum256(input.Data)
	file := LibraryFile{
		ID: publicID, Name: name, MIMEType: parsed.MIMEType, Extension: strings.TrimPrefix(parsed.Extension, "."),
		Size: int64(len(input.Data)), Category: string(parsed.Category), Type: string(parsed.Type), Source: source,
		Status: LibraryFileStatusPending, PageCount: parsed.PageCount, Width: parsed.Width, Height: parsed.Height,
		Previewable: parsed.Type == chatattachment.LibraryTypeImage || parsed.Type == chatattachment.LibraryTypePDF,
		Format:      string(parsed.Format), StoredSize: int64(len(input.Data)), StorageKey: key,
		Digest: hex.EncodeToString(digest[:]),
	}
	reserved, err := s.repo.CreatePending(ctx, &CreateLibraryFileInput{
		UserID: userID, File: file, SourceKey: sourceKey, BypassUploadQuota: bypassUploadQuota,
	}, limit)
	if err != nil {
		if errors.Is(err, ErrChatAttachmentRateLimit) || errors.Is(err, ErrLibraryStorageLimit) || errors.Is(err, ErrLibraryInvalidRequest) {
			return nil, err
		}
		return nil, ErrLibraryUploadFailed.WithCause(err)
	}
	if reserved == nil {
		return nil, ErrLibraryUploadFailed
	}
	// A ready row is an idempotent replay. A deleted row intentionally retains
	// its source key as a user suppression marker and must not be resurrected.
	if sourceKey != "" {
		switch reserved.Status {
		case LibraryFileStatusReady:
			if reserved.Digest != file.Digest || reserved.StoredSize != file.StoredSize || reserved.MIMEType != file.MIMEType {
				return nil, ErrLibraryInvalidRequest.WithMetadata(map[string]string{"conflict": "generated_source_key"})
			}
			decorateLibraryFile(reserved)
			return &GeneratedLibrarySaveResult{File: reserved, AlreadyExists: true}, nil
		case LibraryFileStatusDeleted:
			return &GeneratedLibrarySaveResult{Suppressed: true}, nil
		case LibraryFileStatusPending:
			if reserved.Digest != file.Digest || reserved.StoredSize != file.StoredSize || reserved.MIMEType != file.MIMEType {
				return nil, ErrLibraryInvalidRequest.WithMetadata(map[string]string{"conflict": "generated_source_key"})
			}
			// Resume a crash-left pending reservation using its original identity/key.
			publicID = reserved.ID
			key = reserved.StorageKey
		default:
			return nil, ErrLibraryUploadFailed
		}
	}
	if err = s.store.Put(ctx, key, bytes.NewReader(input.Data), int64(len(input.Data)), parsed.MIMEType); err != nil {
		_, _ = s.repo.Abort(context.WithoutCancel(ctx), userID, publicID)
		_ = s.store.Delete(context.WithoutCancel(ctx), key)
		return nil, ErrLibraryUploadFailed.WithCause(err)
	}
	commitCtx := context.WithoutCancel(ctx)
	activated, err := s.repo.Activate(commitCtx, userID, publicID)
	if err != nil {
		// A database COMMIT can succeed server-side while its acknowledgement is
		// lost. Reconcile before compensating so we never delete the object behind
		// an already-ready metadata row.
		if committed, getErr := s.repo.GetOwned(commitCtx, userID, publicID); getErr == nil &&
			committed != nil && committed.StorageKey == key && committed.StoredSize == file.StoredSize &&
			subtle.ConstantTimeCompare([]byte(committed.Digest), []byte(file.Digest)) == 1 {
			decorateLibraryFile(committed)
			return &GeneratedLibrarySaveResult{File: committed}, nil
		}
		if _, abortErr := s.repo.Abort(commitCtx, userID, publicID); abortErr == nil {
			_ = s.store.Delete(commitCtx, key)
		}
		return nil, ErrLibraryUploadFailed.WithCause(err)
	}
	decorateLibraryFile(activated)
	return &GeneratedLibrarySaveResult{File: activated}, nil
}

func effectiveLibraryImageLimit(cfg config.ChatAttachmentConfig, maxFile int64) int64 {
	limit := cfg.MaxImageBytes
	if limit <= 0 {
		limit = 10 << 20
	}
	if maxFile > 0 && maxFile < limit {
		return maxFile
	}
	return limit
}

func (s *LibraryService) List(ctx context.Context, userID int64, query LibraryFileQuery) (*LibraryFilePage, error) {
	if s == nil || s.repo == nil || userID <= 0 {
		return nil, ErrLibraryUnavailable
	}
	normalized, err := normalizeLibraryFileQuery(query)
	if err != nil {
		return nil, err
	}
	page, err := s.repo.List(ctx, userID, normalized)
	if err != nil {
		return nil, ErrLibraryUnavailable.WithCause(err)
	}
	for i := range page.Files {
		decorateLibraryFile(&page.Files[i])
	}
	return page, nil
}

func (s *LibraryService) Get(ctx context.Context, userID int64, publicID string) (*LibraryFile, error) {
	if s == nil || s.repo == nil || userID <= 0 || !validLibraryPublicID(publicID) {
		return nil, ErrLibraryFileNotFound
	}
	file, err := s.repo.GetOwned(ctx, userID, strings.TrimSpace(publicID))
	if err != nil {
		if errors.Is(err, ErrLibraryFileNotFound) || errors.Is(err, ErrLibraryFileAccessDenied) {
			return nil, err
		}
		return nil, ErrLibraryUnavailable.WithCause(err)
	}
	decorateLibraryFile(file)
	return file, nil
}

func (s *LibraryService) Resolve(ctx context.Context, userID int64, publicIDs []string) ([]LibraryFile, error) {
	if s == nil || s.repo == nil || userID <= 0 {
		return nil, ErrLibraryUnavailable
	}
	ids, err := normalizeLibraryIDs(publicIDs, s.BatchLimit())
	if err != nil {
		return nil, err
	}
	files, err := s.repo.ResolveOwned(ctx, userID, ids)
	if err != nil {
		if errors.Is(err, ErrLibraryFileNotFound) || errors.Is(err, ErrLibraryFileAccessDenied) {
			return nil, err
		}
		return nil, ErrLibraryUnavailable.WithCause(err)
	}
	for i := range files {
		decorateLibraryFile(&files[i])
	}
	return files, nil
}

// ResolveBatchDownload performs all authorization and resource-limit checks
// before the handler sends streaming response headers.
func (s *LibraryService) ResolveBatchDownload(ctx context.Context, userID int64, publicIDs []string) ([]LibraryFile, error) {
	files, err := s.Resolve(ctx, userID, publicIDs)
	if err != nil {
		return nil, err
	}
	limit := s.BatchMaxBytes()
	var total int64
	for i := range files {
		if files[i].StoredSize <= 0 || files[i].StoredSize > limit || total > limit-files[i].StoredSize {
			return nil, ErrLibraryBatchTooLarge.WithMetadata(map[string]string{
				"limit_bytes": fmt.Sprintf("%d", limit),
			})
		}
		total += files[i].StoredSize
	}
	return files, nil
}

func (s *LibraryService) Open(ctx context.Context, userID int64, publicID, mode string) (*LibraryOpenFile, error) {
	file, err := s.Get(ctx, userID, publicID)
	if err != nil {
		return nil, err
	}
	switch mode {
	case "thumbnail":
		if file.Category != "image" {
			return nil, ErrLibraryPreviewUnsupported
		}
	case "preview":
		if file.Type != "image" && file.Type != "pdf" {
			return nil, ErrLibraryPreviewUnsupported
		}
	case "download":
	default:
		return nil, ErrLibraryInvalidRequest
	}
	if mode != "download" && file.Category == "image" {
		return s.openSafeImageRendition(ctx, file, mode)
	}
	body, err := s.store.Open(ctx, file.StorageKey)
	if err != nil {
		return nil, ErrLibraryDownloadFailed.WithCause(err)
	}
	verified := newVerifiedLibraryReader(body, file)
	return &LibraryOpenFile{File: file, Body: verified}, nil
}

// openSafeImageRendition revalidates the private original and returns a
// metadata-free browser rendition. Thumbnail responses are additionally
// downscaled so a library grid cannot download a page of full-resolution
// originals. Downloads continue to use the exact original object.
func (s *LibraryService) openSafeImageRendition(ctx context.Context, file *LibraryFile, mode string) (*LibraryOpenFile, error) {
	rendition, err := s.renderSafeImage(ctx, file, mode)
	if err != nil {
		return nil, err
	}
	responseFile := *file
	responseFile.MIMEType = rendition.mimeType
	responseFile.Size = int64(len(rendition.data))
	responseFile.StoredSize = responseFile.Size
	return &LibraryOpenFile{File: &responseFile, Body: io.NopCloser(bytes.NewReader(rendition.data))}, nil
}

// renderSafeImage owns the complete private-object read, integrity check and
// decode/re-encode operation. Keeping all three inside the same singleflight
// and semaphore boundary prevents duplicate thumbnail requests from each
// holding an S3 body/file descriptor and a full-resolution image in memory.
// Chat message images and model image parts call this same method, so those
// paths cannot bypass the library-wide rendering resource limit.
func (s *LibraryService) renderSafeImage(ctx context.Context, file *LibraryFile, mode string) (libraryImageRendition, error) {
	if s == nil || s.store == nil || file == nil || file.StorageKey == "" || file.StoredSize <= 0 ||
		file.Digest == "" || (mode != "preview" && mode != "thumbnail") {
		return libraryImageRendition{}, ErrLibraryDownloadFailed
	}
	// A joined request must not inherit cancellation from whichever waiter won
	// the singleflight race. Each caller still waits with its own ctx below;
	// the shared operation has an independent hard deadline.
	resultCh := s.renderFlight.DoChan(file.StorageKey+":"+file.Digest+":"+mode, func() (any, error) {
		flightCtx, cancelFlight := context.WithTimeout(context.WithoutCancel(ctx), libraryImageRenderTimeout)
		defer cancelFlight()
		select {
		case s.renderGate <- struct{}{}:
			defer func() { <-s.renderGate }()
		case <-flightCtx.Done():
			return nil, flightCtx.Err()
		}
		body, openErr := s.store.Open(flightCtx, file.StorageKey)
		if openErr != nil {
			return nil, openErr
		}
		verified := newVerifiedLibraryReader(body, file)
		data, readErr := io.ReadAll(verified)
		closeErr := verified.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		processed, processErr := chatattachment.ProcessLibrary(chatattachment.Input{
			Filename: file.Name, DeclaredMIME: file.MIMEType, Data: data,
		})
		if processErr != nil || processed.Kind != chatattachment.KindImage || len(processed.SanitizedData) == 0 {
			if processErr == nil {
				processErr = errors.New("library image did not produce a safe rendition")
			}
			return nil, processErr
		}
		rendered := processed.SanitizedData
		mimeType := processed.SanitizedMIMEType
		if mode == "thumbnail" {
			rendered, processErr = makeLibraryThumbnail(rendered)
			mimeType = "image/jpeg"
		}
		if processErr != nil {
			return nil, processErr
		}
		return libraryImageRendition{data: rendered, mimeType: mimeType}, nil
	})
	select {
	case <-ctx.Done():
		return libraryImageRendition{}, ErrLibraryDownloadFailed.WithCause(ctx.Err())
	case result := <-resultCh:
		if result.Err != nil {
			return libraryImageRendition{}, ErrLibraryDownloadFailed.WithCause(result.Err)
		}
		rendition, ok := result.Val.(libraryImageRendition)
		if !ok || len(rendition.data) == 0 || rendition.mimeType == "" {
			return libraryImageRendition{}, ErrLibraryDownloadFailed.WithCause(errors.New("library image rendition is empty"))
		}
		return rendition, nil
	}
}

func makeLibraryThumbnail(data []byte) ([]byte, error) {
	source, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	bounds := source.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 {
		return nil, errors.New("library thumbnail has invalid dimensions")
	}
	if width <= libraryThumbnailMaxDimension && height <= libraryThumbnailMaxDimension {
		var output bytes.Buffer
		err = jpeg.Encode(&output, source, &jpeg.Options{Quality: 82})
		return output.Bytes(), err
	}
	newWidth, newHeight := libraryThumbnailMaxDimension, libraryThumbnailMaxDimension
	if width >= height {
		newHeight = max(1, int(int64(height)*libraryThumbnailMaxDimension/int64(width)))
	} else {
		newWidth = max(1, int(int64(width)*libraryThumbnailMaxDimension/int64(height)))
	}
	thumbnail := image.NewNRGBA(image.Rect(0, 0, newWidth, newHeight))
	draw.ApproxBiLinear.Scale(thumbnail, thumbnail.Bounds(), source, bounds, draw.Over, nil)
	var output bytes.Buffer
	if err = jpeg.Encode(&output, thumbnail, &jpeg.Options{Quality: 82}); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func (s *LibraryService) Delete(ctx context.Context, userID int64, publicID string) error {
	if s == nil || s.repo == nil || userID <= 0 || !validLibraryPublicID(publicID) {
		return ErrLibraryFileNotFound
	}
	if _, err := s.repo.MarkDeleted(ctx, userID, strings.TrimSpace(publicID)); err != nil {
		if errors.Is(err, ErrLibraryFileNotFound) || errors.Is(err, ErrLibraryFileAccessDenied) {
			return err
		}
		return ErrLibraryDeleteFailed.WithCause(err)
	}
	return nil
}

func (s *LibraryService) Storage(ctx context.Context, userID int64) (*LibraryStorage, error) {
	if s == nil || s.repo == nil || userID <= 0 || s.configErr != nil {
		return nil, ErrLibraryUnavailable.WithCause(s.configErr)
	}
	used, err := s.repo.Usage(ctx, userID)
	if err != nil {
		return nil, ErrLibraryUnavailable.WithCause(err)
	}
	limit, err := s.repo.ResolveStorageLimit(ctx, userID, s.cfg.DefaultStorageBytes, s.groupOverrides)
	if err != nil {
		return nil, ErrLibraryUnavailable.WithCause(err)
	}
	remaining := int64(0)
	overLimit := false
	if limit > 0 {
		overLimit = used > limit
		if used < limit {
			remaining = limit - used
		}
	}
	return &LibraryStorage{UsedBytes: used, LimitBytes: limit, RemainingBytes: remaining, OverLimit: overLimit}, nil
}

func (s *LibraryService) Start() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.done = make(chan struct{})
	go s.janitor(ctx, s.done)
}

func (s *LibraryService) Stop() {
	if s == nil {
		return
	}
	s.mu.Lock()
	cancel, done := s.cancel, s.done
	s.cancel, s.done = nil, nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
	s.storeClose.Do(func() {
		if closer, ok := s.store.(io.Closer); ok {
			if err := closer.Close(); err != nil {
				slog.Warn("close library blob store failed", "error", err)
			}
		}
	})
}

func (s *LibraryService) janitor(ctx context.Context, done chan<- struct{}) {
	defer close(done)
	interval := time.Duration(s.cfg.CleanupIntervalMinutes) * time.Minute
	if interval <= 0 {
		interval = time.Hour
	}
	if err := s.RunCleanup(ctx); err != nil {
		slog.Warn("initial library cleanup failed", "error", err)
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.RunCleanup(ctx); err != nil {
				slog.Warn("library cleanup failed", "error", err)
			}
		}
	}
}

func (s *LibraryService) RunCleanup(ctx context.Context) error {
	if s == nil || s.repo == nil || s.store == nil {
		return ErrLibraryUnavailable
	}
	limit := s.cfg.CleanupBatchSize
	if limit <= 0 {
		limit = 100
	}
	retention := s.cfg.DeletedRetentionDays
	if retention < 0 {
		retention = 30
	}
	deletedBefore := time.Now().UTC().Add(-time.Duration(retention) * 24 * time.Hour)
	pendingStaleMinutes := s.cfg.PendingUploadStaleMinutes
	if pendingStaleMinutes <= 0 {
		pendingStaleMinutes = 60
	}
	pendingBefore := time.Now().UTC().Add(-time.Duration(pendingStaleMinutes) * time.Minute)
	for {
		files, err := s.repo.ClaimCleanup(ctx, deletedBefore, pendingBefore, limit)
		if err != nil {
			return ErrLibraryUnavailable.WithCause(err)
		}
		for i := range files {
			if files[i].StorageKey != "" {
				if err = s.store.Delete(ctx, files[i].StorageKey); err != nil {
					return ErrLibraryDeleteFailed.WithCause(err)
				}
			}
			if err = s.repo.FinalizeCleanup(ctx, files[i].UserID, files[i].ID); err != nil {
				return ErrLibraryDeleteFailed.WithCause(err)
			}
		}
		if len(files) < limit {
			break
		}
	}
	cleaner, ok := s.store.(LibraryBlobOrphanCleaner)
	if !ok {
		return nil
	}
	// Fail closed: never sweep when the authoritative reference snapshot cannot
	// be read. The 24h minimum grace also protects snapshot-vs-new-upload races.
	referenced, err := s.repo.ListStorageKeys(ctx)
	if err != nil {
		return ErrLibraryUnavailable.WithCause(err)
	}
	grace := 24 * time.Hour
	if configured := 2 * time.Duration(s.cfg.CleanupIntervalMinutes) * time.Minute; configured > grace {
		grace = configured
	}
	if pending := time.Duration(s.cfg.PendingUploadStaleMinutes) * time.Minute; pending > grace {
		grace = pending
	}
	if _, err = cleaner.CleanupOrphans(ctx, referenced, time.Now().UTC().Add(-grace), limit); err != nil {
		return ErrLibraryDeleteFailed.WithCause(err)
	}
	return nil
}

func normalizeLibraryFileQuery(query LibraryFileQuery) (LibraryFileQuery, error) {
	query.Q = strings.TrimSpace(query.Q)
	if utf8.RuneCountInString(query.Q) > maxLibrarySearchRunes {
		return LibraryFileQuery{}, ErrLibraryInvalidRequest
	}
	query.Category = strings.ToLower(strings.TrimSpace(query.Category))
	if query.Category == "" {
		query.Category = "all"
	}
	query.Source = strings.ToLower(strings.TrimSpace(query.Source))
	if query.Source == "" {
		query.Source = "all"
	}
	query.Type = strings.ToLower(strings.TrimSpace(query.Type))
	if query.Type == "" {
		query.Type = "all"
	}
	query.Sort = strings.ToLower(strings.TrimSpace(query.Sort))
	if query.Sort == "" {
		query.Sort = "updated_desc"
	}
	if !stringIn(query.Category, "all", "image", "file") ||
		!stringIn(query.Source, "all", "uploaded", "generated") ||
		!stringIn(query.Type, "all", "image", "pdf", "document", "spreadsheet", "presentation", "other") ||
		!stringIn(query.Sort, "updated_desc", "updated_asc", "name_asc", "name_desc", "size_asc", "size_desc") {
		return LibraryFileQuery{}, ErrLibraryInvalidRequest
	}
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = defaultLibraryPageSize
	}
	if query.PageSize > maxLibraryPageSize {
		return LibraryFileQuery{}, ErrLibraryInvalidRequest
	}
	return query, nil
}

func normalizeLibraryIDs(values []string, limit int) ([]string, error) {
	if len(values) == 0 || len(values) > limit {
		return nil, ErrLibraryBatchLimit
	}
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if !validLibraryPublicID(value) {
			return nil, ErrLibraryInvalidRequest
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	if len(result) == 0 {
		return nil, ErrLibraryBatchLimit
	}
	return result, nil
}

func stringIn(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func safeLibraryFilename(value string) (string, error) {
	value = strings.TrimSpace(filepath.Base(strings.ReplaceAll(value, "\\", "/")))
	if value == "" || value == "." || utf8.RuneCountInString(value) > 255 {
		return "", errors.New("invalid library filename")
	}
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || r == '/' || r == '\\' || r == ':' {
			return '_'
		}
		return r
	}, value)
	if strings.TrimSpace(value) == "" {
		return "", errors.New("invalid library filename")
	}
	return value, nil
}

func uniqueLibraryZipName(name string, used map[string]int) string {
	key := strings.ToLower(name)
	if used[key] == 0 {
		used[key] = 1
		return name
	}
	extension := filepath.Ext(name)
	base := strings.TrimSuffix(name, extension)
	for suffix := used[key] + 1; ; suffix++ {
		candidate := fmt.Sprintf("%s (%d)%s", base, suffix, extension)
		candidateKey := strings.ToLower(candidate)
		if used[candidateKey] != 0 {
			continue
		}
		used[key] = suffix
		used[candidateKey] = 1
		return candidate
	}
}

func newLibraryFilePublicID() (string, error) {
	var random [16]byte
	if _, err := io.ReadFull(rand.Reader, random[:]); err != nil {
		return "", err
	}
	return "file_" + hex.EncodeToString(random[:]), nil
}

func newLibraryStorageKey(userID int64) (string, error) {
	var random [16]byte
	if _, err := io.ReadFull(rand.Reader, random[:]); err != nil {
		return "", err
	}
	return fmt.Sprintf("library/%d/%s", userID, hex.EncodeToString(random[:])), nil
}

func validLibraryPublicID(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) < 8 || len(value) > 80 {
		return false
	}
	for _, r := range value {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

func decorateLibraryFile(file *LibraryFile) {
	if file == nil {
		return
	}
	file.Previewable = file.Type == "image" || file.Type == "pdf"
	file.Extension = strings.TrimPrefix(file.Extension, ".")
}

type verifiedLibraryReader struct {
	body     io.ReadCloser
	left     int64
	expected []byte
	hash     hashWriter
	failed   error
	ended    bool
}

type hashWriter interface {
	Write([]byte) (int, error)
	Sum([]byte) []byte
}

func newVerifiedLibraryReader(body io.ReadCloser, file *LibraryFile) io.ReadCloser {
	expected, _ := hex.DecodeString(strings.TrimSpace(file.Digest))
	return &verifiedLibraryReader{body: body, left: file.StoredSize, expected: expected, hash: sha256.New()}
}

func (r *verifiedLibraryReader) Read(p []byte) (int, error) {
	if r.failed != nil {
		return 0, r.failed
	}
	if r.ended {
		return 0, io.EOF
	}
	if r.left == 0 {
		return r.finish()
	}
	if int64(len(p)) > r.left {
		p = p[:r.left]
	}
	n, err := r.body.Read(p)
	if n > 0 {
		r.left -= int64(n)
		_, _ = r.hash.Write(p[:n])
	}
	if errors.Is(err, io.EOF) {
		if r.left != 0 {
			r.failed = io.ErrUnexpectedEOF
			return n, r.failed
		}
		// Readers may legally return both data and EOF. Validate immediately so
		// callers such as io.Copy cannot treat the final bytes as verified before
		// the SHA-256 and exact-length checks run.
		_, finishErr := r.finish()
		return n, finishErr
	}
	return n, err
}

func (r *verifiedLibraryReader) finish() (int, error) {
	var probe [1]byte
	for attempts := 0; attempts < 100; attempts++ {
		n, err := r.body.Read(probe[:])
		if n != 0 || (err != nil && !errors.Is(err, io.EOF)) {
			r.failed = errors.New("library file integrity check failed")
			return 0, r.failed
		}
		if errors.Is(err, io.EOF) {
			if verifyErr := r.verifyEOF(); verifyErr != nil {
				return 0, verifyErr
			}
			return 0, io.EOF
		}
	}
	r.failed = io.ErrNoProgress
	return 0, r.failed
}

func (r *verifiedLibraryReader) verifyEOF() error {
	if len(r.expected) != sha256.Size || subtle.ConstantTimeCompare(r.hash.Sum(nil), r.expected) != 1 {
		r.failed = errors.New("library file integrity check failed")
		return r.failed
	}
	r.ended = true
	return nil
}

func (r *verifiedLibraryReader) Close() error { return r.body.Close() }

// SortedLibraryIDs is used by tests and diagnostics without exposing physical keys.
func SortedLibraryIDs(files []LibraryFile) []string {
	ids := make([]string, 0, len(files))
	for i := range files {
		ids = append(ids, files[i].ID)
	}
	sort.Strings(ids)
	return ids
}
