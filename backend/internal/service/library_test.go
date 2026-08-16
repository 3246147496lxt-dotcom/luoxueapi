package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type libraryRepoFake struct {
	files             map[string]LibraryFile
	limit             int64
	createErr         error
	aborted           []string
	created           int
	activated         int
	usage             int64
	lastCreated       *CreateLibraryFileInput
	activateErr       error
	activateCommitted bool
}

func (f *libraryRepoFake) ResolveStorageLimit(context.Context, int64, int64, map[int64]int64) (int64, error) {
	return f.limit, nil
}
func (f *libraryRepoFake) CheckUploadQuota(context.Context, int64, int64) error { return nil }
func (f *libraryRepoFake) CreatePending(_ context.Context, input *CreateLibraryFileInput, _ int64) (*LibraryFile, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	f.created++
	copyInput := *input
	f.lastCreated = &copyInput
	file := input.File
	file.InternalID = int64(f.created)
	if f.files == nil {
		f.files = make(map[string]LibraryFile)
	}
	f.files[file.ID] = file
	return &file, nil
}
func (f *libraryRepoFake) Activate(_ context.Context, _ int64, id string) (*LibraryFile, error) {
	file, ok := f.files[id]
	if !ok {
		return nil, ErrLibraryFileNotFound
	}
	f.activated++
	file.Status = LibraryFileStatusReady
	file.CreatedAt = time.Now().UTC()
	file.UpdatedAt = file.CreatedAt
	f.files[id] = file
	if f.activateErr != nil {
		if !f.activateCommitted {
			file.Status = LibraryFileStatusPending
			f.files[id] = file
		}
		return nil, f.activateErr
	}
	return &file, nil
}
func (f *libraryRepoFake) Abort(_ context.Context, _ int64, id string) (*LibraryFile, error) {
	f.aborted = append(f.aborted, id)
	file := f.files[id]
	file.Status = LibraryFileStatusDeleted
	f.files[id] = file
	return &file, nil
}
func (f *libraryRepoFake) GetOwned(_ context.Context, _ int64, id string) (*LibraryFile, error) {
	file, ok := f.files[id]
	if !ok || file.Status != LibraryFileStatusReady {
		return nil, ErrLibraryFileNotFound
	}
	return &file, nil
}
func (f *libraryRepoFake) List(context.Context, int64, LibraryFileQuery) (*LibraryFilePage, error) {
	return &LibraryFilePage{}, nil
}
func (f *libraryRepoFake) ResolveOwned(_ context.Context, _ int64, ids []string) ([]LibraryFile, error) {
	result := make([]LibraryFile, 0, len(ids))
	for _, id := range ids {
		file, ok := f.files[id]
		if !ok || file.Status != LibraryFileStatusReady {
			return nil, ErrLibraryFileNotFound
		}
		result = append(result, file)
	}
	return result, nil
}
func (f *libraryRepoFake) MarkDeleted(_ context.Context, _ int64, id string) (*LibraryFile, error) {
	file, ok := f.files[id]
	if !ok {
		return nil, ErrLibraryFileNotFound
	}
	file.Status = LibraryFileStatusDeleted
	f.files[id] = file
	return &file, nil
}
func (f *libraryRepoFake) Usage(context.Context, int64) (int64, error) { return f.usage, nil }
func (f *libraryRepoFake) ClaimCleanup(context.Context, time.Time, time.Time, int) ([]LibraryFile, error) {
	return nil, nil
}
func (f *libraryRepoFake) FinalizeCleanup(context.Context, int64, string) error { return nil }
func (f *libraryRepoFake) ListStorageKeys(context.Context) (map[string]struct{}, error) {
	return map[string]struct{}{}, nil
}

type failingLibraryBlobStore struct{ putErr error }

func (f *failingLibraryBlobStore) Put(context.Context, string, io.Reader, int64, string) error {
	return f.putErr
}
func (*failingLibraryBlobStore) Open(context.Context, string) (io.ReadCloser, error) {
	return nil, errors.New("missing")
}
func (*failingLibraryBlobStore) Delete(context.Context, string) error { return nil }

type closingLibraryBlobStore struct {
	libraryBlobStoreFake
	closed int
}

func (s *closingLibraryBlobStore) Close() error {
	s.closed++
	return nil
}

func TestLibraryUploadCreatesOneDurableRecordAndChatAliasMetadata(t *testing.T) {
	repo := &libraryRepoFake{}
	store := &libraryBlobStoreFake{}
	service := NewLibraryService(repo, store, &config.Config{Library: config.LibraryConfig{MaxFileBytes: 20 << 20}})
	data := []byte("hello library")

	file, err := service.Upload(context.Background(), 7, LibraryUpload{
		Filename: "notes.txt", DeclaredMIME: "text/plain", Data: data,
	}, "uploaded")
	require.NoError(t, err)
	require.Equal(t, LibraryFileStatusReady, file.Status)
	require.Equal(t, "document", file.Type)
	require.Equal(t, "uploaded", file.Source)
	require.Equal(t, 1, repo.created)
	require.Equal(t, 1, repo.activated)
	require.NotNil(t, repo.lastCreated)
	require.Empty(t, repo.lastCreated.File.ExtractedText, "library persistence must not extract document content")
	require.Contains(t, repo.lastCreated.File.StorageKey, "library/7/")
	require.Equal(t, data, store.files[repo.lastCreated.File.StorageKey])
}

func TestLibraryUploadStorageFailureAbortsPendingRecord(t *testing.T) {
	repo := &libraryRepoFake{}
	service := NewLibraryService(repo, &failingLibraryBlobStore{putErr: errors.New("storage down")}, nil)

	_, err := service.Upload(context.Background(), 7, LibraryUpload{
		Filename: "notes.txt", DeclaredMIME: "text/plain", Data: []byte("hello"),
	}, "uploaded")
	require.ErrorIs(t, err, ErrLibraryUploadFailed)
	require.Len(t, repo.aborted, 1)
	require.Zero(t, repo.activated)
}

func TestLibrarySaveGeneratedRejectsSourceKeyBeyondSchemaBound(t *testing.T) {
	service := NewLibraryService(&libraryRepoFake{}, &libraryBlobStoreFake{}, nil)

	_, err := service.SaveGeneratedIdempotent(context.Background(), 7, LibraryUpload{}, strings.Repeat("x", maxLibrarySourceKeyBytes+1))
	require.ErrorIs(t, err, ErrLibraryInvalidRequest)
}

func TestLibraryUploadReconcilesAmbiguousActivationCommit(t *testing.T) {
	repo := &libraryRepoFake{activateErr: errors.New("commit acknowledgement lost"), activateCommitted: true}
	store := &libraryBlobStoreFake{}
	service := NewLibraryService(repo, store, nil)

	file, err := service.Upload(context.Background(), 7, LibraryUpload{
		Filename: "notes.txt", DeclaredMIME: "text/plain", Data: []byte("durable"),
	}, "uploaded")
	require.NoError(t, err)
	require.Equal(t, LibraryFileStatusReady, file.Status)
	require.Empty(t, repo.aborted)
	require.Contains(t, store.files, file.StorageKey)
}

func TestLibraryVerifiedReaderRejectsSameSizeCorruption(t *testing.T) {
	want := []byte("good")
	digest := sha256.Sum256(want)
	reader := newVerifiedLibraryReader(io.NopCloser(bytes.NewReader([]byte("evil"))), &LibraryFile{
		StoredSize: int64(len(want)), Digest: hex.EncodeToString(digest[:]),
	})
	_, err := io.ReadAll(reader)
	require.Error(t, err)
	require.NoError(t, reader.Close())
}

type dataAndEOFReader struct {
	data []byte
	done bool
}

func (r *dataAndEOFReader) Read(p []byte) (int, error) {
	if r.done {
		return 0, io.EOF
	}
	r.done = true
	return copy(p, r.data), io.EOF
}

func (*dataAndEOFReader) Close() error { return nil }

func TestLibraryVerifiedReaderChecksDigestWhenLastReadReturnsDataAndEOF(t *testing.T) {
	want := []byte("good")
	digest := sha256.Sum256(want)
	reader := newVerifiedLibraryReader(&dataAndEOFReader{data: []byte("evil")}, &LibraryFile{
		StoredSize: int64(len(want)), Digest: hex.EncodeToString(digest[:]),
	})
	_, err := io.ReadAll(reader)
	require.Error(t, err)
}

func TestLibraryStorageReportsFiniteAndUnlimitedEntitlements(t *testing.T) {
	repo := &libraryRepoFake{usage: 120, limit: 100}
	service := NewLibraryService(repo, &libraryBlobStoreFake{}, nil)
	storage, err := service.Storage(context.Background(), 7)
	require.NoError(t, err)
	require.True(t, storage.OverLimit)
	require.Zero(t, storage.RemainingBytes)

	repo.limit = 0
	storage, err = service.Storage(context.Background(), 7)
	require.NoError(t, err)
	require.False(t, storage.OverLimit)
	require.Zero(t, storage.RemainingBytes)
}

func TestLibraryResolveBatchDownloadRejectsOversizedSelection(t *testing.T) {
	repo := &libraryRepoFake{files: map[string]LibraryFile{
		"file_one": {ID: "file_one", Status: LibraryFileStatusReady, StoredSize: 60},
		"file_two": {ID: "file_two", Status: LibraryFileStatusReady, StoredSize: 50},
	}}
	service := NewLibraryService(repo, &libraryBlobStoreFake{}, &config.Config{Library: config.LibraryConfig{
		BatchDownloadLimit: 10, BatchDownloadMaxBytes: 100,
	}})

	_, err := service.ResolveBatchDownload(context.Background(), 7, []string{"file_one", "file_two"})
	require.ErrorIs(t, err, ErrLibraryBatchTooLarge)
}

func TestLibraryBatchMaxBytesDefensivelyCapsUnvalidatedConfig(t *testing.T) {
	service := NewLibraryService(&libraryRepoFake{}, &libraryBlobStoreFake{}, &config.Config{Library: config.LibraryConfig{
		BatchDownloadMaxBytes: 10 << 30,
	}})
	require.EqualValues(t, 100<<20, service.BatchMaxBytes())
}

func TestLibraryImagePreviewIsSanitizedAndThumbnailIsBounded(t *testing.T) {
	originalImage := image.NewNRGBA(image.Rect(0, 0, 1024, 256))
	originalImage.Set(0, 0, color.NRGBA{R: 0xcc, G: 0x44, B: 0x22, A: 0xff})
	var original bytes.Buffer
	require.NoError(t, png.Encode(&original, originalImage))
	digest := sha256.Sum256(original.Bytes())
	file := LibraryFile{
		ID: "file_image", Name: "wide.png", MIMEType: "image/png", Category: "image", Type: "image",
		Status: LibraryFileStatusReady, Format: "png", Size: int64(original.Len()), StoredSize: int64(original.Len()),
		StorageKey: "library/7/wide", Digest: hex.EncodeToString(digest[:]),
	}
	repo := &libraryRepoFake{files: map[string]LibraryFile{file.ID: file}}
	store := &libraryBlobStoreFake{files: map[string][]byte{file.StorageKey: original.Bytes()}}
	service := NewLibraryService(repo, store, nil)

	preview, err := service.Open(context.Background(), 7, file.ID, "preview")
	require.NoError(t, err)
	previewData, err := io.ReadAll(preview.Body)
	require.NoError(t, err)
	require.NoError(t, preview.Body.Close())
	require.Equal(t, "image/png", preview.File.MIMEType)
	require.NotEmpty(t, previewData)
	require.EqualValues(t, len(previewData), preview.File.Size)

	thumbnail, err := service.Open(context.Background(), 7, file.ID, "thumbnail")
	require.NoError(t, err)
	thumbnailData, err := io.ReadAll(thumbnail.Body)
	require.NoError(t, err)
	require.NoError(t, thumbnail.Body.Close())
	require.Equal(t, "image/jpeg", thumbnail.File.MIMEType)
	decoded, _, err := image.Decode(bytes.NewReader(thumbnailData))
	require.NoError(t, err)
	require.LessOrEqual(t, decoded.Bounds().Dx(), libraryThumbnailMaxDimension)
	require.LessOrEqual(t, decoded.Bounds().Dy(), libraryThumbnailMaxDimension)

	download, err := service.Open(context.Background(), 7, file.ID, "download")
	require.NoError(t, err)
	downloadData, err := io.ReadAll(download.Body)
	require.NoError(t, err)
	require.NoError(t, download.Body.Close())
	require.Equal(t, original.Bytes(), downloadData, "downloads must preserve the original object")
}

type blockingLibraryBlobStore struct {
	libraryBlobStoreFake
	opened        chan struct{}
	release       chan struct{}
	once          sync.Once
	ignoreContext bool
}

type doneNotifyingContext struct {
	context.Context
	called chan struct{}
	once   sync.Once
}

func (c *doneNotifyingContext) Done() <-chan struct{} {
	c.once.Do(func() { close(c.called) })
	return c.Context.Done()
}

func (s *blockingLibraryBlobStore) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	s.once.Do(func() { close(s.opened) })
	if s.ignoreContext {
		<-s.release
	} else {
		select {
		case <-s.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return s.libraryBlobStoreFake.Open(ctx, key)
}

func TestLibraryImageRenditionFollowerSurvivesLeaderCancellation(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	var original bytes.Buffer
	require.NoError(t, png.Encode(&original, img))
	digest := sha256.Sum256(original.Bytes())
	file := LibraryFile{
		ID: "file_image_cancel", Name: "image.png", MIMEType: "image/png", Category: "image", Type: "image",
		Status: LibraryFileStatusReady, Size: int64(original.Len()), StoredSize: int64(original.Len()),
		StorageKey: "library/7/cancel", Digest: hex.EncodeToString(digest[:]),
	}
	service := NewLibraryService(&libraryRepoFake{files: map[string]LibraryFile{file.ID: file}}, &blockingLibraryBlobStore{
		libraryBlobStoreFake: libraryBlobStoreFake{files: map[string][]byte{file.StorageKey: original.Bytes()}},
		opened:               make(chan struct{}), release: make(chan struct{}), ignoreContext: true,
	}, nil)
	store := service.store.(*blockingLibraryBlobStore)
	leaderCtx, cancelLeader := context.WithCancel(context.Background())
	leaderResult := make(chan error, 1)
	go func() {
		_, err := service.Open(leaderCtx, 7, file.ID, "thumbnail")
		leaderResult <- err
	}()
	<-store.opened
	followerResult := make(chan error, 1)
	followerRegistered := make(chan struct{})
	go func() {
		opened, err := service.Open(&doneNotifyingContext{
			Context: context.Background(),
			called:  followerRegistered,
		}, 7, file.ID, "thumbnail")
		if err == nil {
			_ = opened.Body.Close()
		}
		followerResult <- err
	}()
	<-followerRegistered
	cancelLeader()
	require.Error(t, <-leaderResult)
	close(store.release)
	require.NoError(t, <-followerResult)
	require.Equal(t, []string{file.StorageKey}, store.opens)
}

func TestLibraryImageRenditionSingleflightCoversPrivateObjectOpen(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	var original bytes.Buffer
	require.NoError(t, png.Encode(&original, img))
	digest := sha256.Sum256(original.Bytes())
	file := LibraryFile{
		ID: "file_image_flight", Name: "image.png", MIMEType: "image/png", Category: "image", Type: "image",
		Status: LibraryFileStatusReady, Size: int64(original.Len()), StoredSize: int64(original.Len()),
		StorageKey: "library/7/flight", Digest: hex.EncodeToString(digest[:]),
	}
	repo := &libraryRepoFake{files: map[string]LibraryFile{file.ID: file}}
	store := &blockingLibraryBlobStore{
		libraryBlobStoreFake: libraryBlobStoreFake{files: map[string][]byte{file.StorageKey: original.Bytes()}},
		opened:               make(chan struct{}), release: make(chan struct{}),
	}
	service := NewLibraryService(repo, store, nil)
	results := make(chan error, 2)
	open := func(ctx context.Context) {
		opened, err := service.Open(ctx, 7, file.ID, "thumbnail")
		if err == nil {
			_, err = io.ReadAll(opened.Body)
			_ = opened.Body.Close()
		}
		results <- err
	}
	go open(context.Background())
	<-store.opened
	followerRegistered := make(chan struct{})
	go open(&doneNotifyingContext{Context: context.Background(), called: followerRegistered})
	<-followerRegistered
	close(store.release)
	require.NoError(t, <-results)
	require.NoError(t, <-results)
	require.Equal(t, []string{file.StorageKey}, store.opens, "duplicate renditions must share the object read")
}

func TestLibraryStopClosesBlobStoreOnce(t *testing.T) {
	store := &closingLibraryBlobStore{}
	service := NewLibraryService(&libraryRepoFake{}, store, nil)
	service.Start()
	service.Stop()
	service.Stop()
	require.Equal(t, 1, store.closed)
}
