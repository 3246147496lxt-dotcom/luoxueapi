package repository

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

func TestLocalLibraryBlobStorePutOpenDelete(t *testing.T) {
	rootPath := filepath.Join(t.TempDir(), "library")
	store := newTestLocalLibraryBlobStore(t, rootPath)
	ctx := context.Background()
	key := "library/42/objects/asset.bin"
	payload := []byte("private streamed content")

	require.NoError(t, store.Put(ctx, key, bytes.NewReader(payload), int64(len(payload)), "application/octet-stream"))

	reader, err := store.Open(ctx, key)
	require.NoError(t, err)
	actual, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	require.Equal(t, payload, actual)

	assertFileMode(t, rootPath, 0o700)
	assertFileMode(t, filepath.Join(rootPath, "library"), 0o700)
	assertFileMode(t, filepath.Join(rootPath, "library", "42", "objects"), 0o700)
	assertFileMode(t, filepath.Join(rootPath, filepath.FromSlash(key)), 0o600)

	require.NoError(t, store.Delete(ctx, key))
	require.NoError(t, store.Delete(ctx, key), "deleting a missing blob must be idempotent")
	_, err = store.Open(ctx, key)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestLocalLibraryBlobStoreRejectsSizeMismatchWithoutCommitting(t *testing.T) {
	rootPath := filepath.Join(t.TempDir(), "library")
	store := newTestLocalLibraryBlobStore(t, rootPath)
	ctx := context.Background()

	tests := []struct {
		name    string
		key     string
		payload []byte
		size    int64
	}{
		{name: "short", key: "objects/short", payload: []byte("abc"), size: 4},
		{name: "long", key: "objects/long", payload: []byte("abcd"), size: 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.Put(ctx, tt.key, bytes.NewReader(tt.payload), tt.size, "text/plain")
			require.Error(t, err)
			_, statErr := os.Stat(filepath.Join(rootPath, filepath.FromSlash(tt.key)))
			require.ErrorIs(t, statErr, os.ErrNotExist)
		})
	}
}

func TestLibraryBlobStoresRejectNonCanonicalKeys(t *testing.T) {
	local := newTestLocalLibraryBlobStore(t, filepath.Join(t.TempDir(), "library"))
	s3Store := &s3LibraryBlobStore{client: newFakeLibraryS3Client(), bucket: "private"}
	invalid := []string{"", ".", "../escape", "/absolute", "a/../b", "a//b", "a/./b", `a\b`, "a/", "\x00"}

	for _, key := range invalid {
		t.Run(key, func(t *testing.T) {
			require.Error(t, local.Put(context.Background(), key, bytes.NewReader(nil), 0, ""))
			require.Error(t, s3Store.Put(context.Background(), key, bytes.NewReader(nil), 0, ""))
		})
	}
}

func TestLocalLibraryBlobStoreRejectsSymlinkedDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation commonly requires elevated privileges on Windows")
	}
	rootPath := filepath.Join(t.TempDir(), "library")
	store := newTestLocalLibraryBlobStore(t, rootPath)
	realDir := filepath.Join(rootPath, "real")
	require.NoError(t, os.Mkdir(realDir, 0o700))
	require.NoError(t, os.Symlink("real", filepath.Join(rootPath, "linked")))

	err := store.Put(context.Background(), "linked/blob", bytes.NewReader([]byte("x")), 1, "")
	require.Error(t, err)
	_, statErr := os.Stat(filepath.Join(realDir, "blob"))
	require.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestLocalLibraryBlobStoreCleanupOrphansIsAgeGuardedAndBounded(t *testing.T) {
	rootPath := filepath.Join(t.TempDir(), "library")
	store := newTestLocalLibraryBlobStore(t, rootPath)
	ctx := context.Background()
	referencedKey := "library/7/referenced"
	orphanKey := "library/7/orphan"
	secondOrphanKey := "library/7/orphan-two"
	for _, key := range []string{referencedKey, orphanKey, secondOrphanKey} {
		require.NoError(t, store.Put(ctx, key, bytes.NewReader([]byte("x")), 1, "text/plain"))
		require.NoError(t, os.Chtimes(filepath.Join(rootPath, filepath.FromSlash(key)), time.Now().Add(-48*time.Hour), time.Now().Add(-48*time.Hour)))
	}
	recentKey := "library/7/recent"
	require.NoError(t, store.Put(ctx, recentKey, bytes.NewReader([]byte("x")), 1, "text/plain"))

	deleted, err := store.CleanupOrphans(ctx, map[string]struct{}{referencedKey: {}}, time.Now().Add(-24*time.Hour), 1)
	require.NoError(t, err)
	require.Equal(t, 1, deleted)
	referenced, err := store.Open(ctx, referencedKey)
	require.NoError(t, err)
	require.NoError(t, referenced.Close())
	recent, err := store.Open(ctx, recentKey)
	require.NoError(t, err)
	require.NoError(t, recent.Close())
	remainingOld := 0
	for _, key := range []string{orphanKey, secondOrphanKey} {
		if _, statErr := os.Stat(filepath.Join(rootPath, filepath.FromSlash(key))); statErr == nil {
			remainingOld++
		}
	}
	require.Equal(t, 1, remainingOld)
}

func TestLocalLibraryBlobStoreCleanupOrphansStopsAtScanBudget(t *testing.T) {
	rootPath := filepath.Join(t.TempDir(), "library")
	store := newTestLocalLibraryBlobStore(t, rootPath)
	objectDir := filepath.Join(rootPath, "library", "7")
	require.NoError(t, os.MkdirAll(objectDir, 0o700))
	old := time.Now().Add(-48 * time.Hour)
	referenced := make(map[string]struct{}, libraryOrphanScanBudget(1))
	for i := 0; i < libraryOrphanScanBudget(1); i++ {
		name := fmt.Sprintf("a-referenced-%04d", i)
		key := "library/7/" + name
		path := filepath.Join(objectDir, name)
		require.NoError(t, os.WriteFile(path, []byte("r"), 0o600))
		require.NoError(t, os.Chtimes(path, old, old))
		referenced[key] = struct{}{}
	}
	orphanPath := filepath.Join(objectDir, "z-orphan")
	require.NoError(t, os.WriteFile(orphanPath, []byte("o"), 0o600))
	require.NoError(t, os.Chtimes(orphanPath, old, old))

	deleted, err := store.CleanupOrphans(context.Background(), referenced, time.Now().Add(-24*time.Hour), 1)
	require.NoError(t, err)
	require.Zero(t, deleted)
	_, err = os.Stat(orphanPath)
	require.NoError(t, err, "objects beyond the per-run scan budget must not be visited")

	deleted, err = store.CleanupOrphans(context.Background(), referenced, time.Now().Add(-24*time.Hour), 1)
	require.NoError(t, err)
	require.Equal(t, 1, deleted, "the rolling cursor must reach an orphan beyond the first scan budget")
	require.ErrorIs(t, func() error { _, statErr := os.Stat(orphanPath); return statErr }(), os.ErrNotExist)

	// The deletion-limit stop leaves the cursor at the old lexical end. One
	// empty continuation round wraps it, after which an earlier key is eligible.
	deleted, err = store.CleanupOrphans(context.Background(), referenced, time.Now().Add(-24*time.Hour), 1)
	require.NoError(t, err)
	require.Zero(t, deleted)
	earlyOrphanPath := filepath.Join(objectDir, "0-orphan")
	require.NoError(t, os.WriteFile(earlyOrphanPath, []byte("o"), 0o600))
	require.NoError(t, os.Chtimes(earlyOrphanPath, old, old))
	deleted, err = store.CleanupOrphans(context.Background(), referenced, time.Now().Add(-24*time.Hour), 1)
	require.NoError(t, err)
	require.Equal(t, 1, deleted, "a completed pass must wrap to keys created behind the cursor")
	require.ErrorIs(t, func() error { _, statErr := os.Stat(earlyOrphanPath); return statErr }(), os.ErrNotExist)
}

func TestResolveLibraryS3ConfigFallsBackToPrivateImageCredentials(t *testing.T) {
	resolved, err := resolveLibraryS3Config(
		config.LibraryS3Config{Bucket: "library-bucket"},
		config.ImageStorageConfig{
			Endpoint:        "https://private.example.test",
			Region:          "auto",
			Bucket:          "image-bucket",
			AccessKeyID:     "private-key",
			SecretAccessKey: "private-secret",
			ForcePathStyle:  true,
			PublicBaseURL:   "https://public.example.test",
		},
	)
	require.NoError(t, err)
	require.Equal(t, "https://private.example.test", resolved.Endpoint)
	require.Equal(t, "auto", resolved.Region)
	require.Equal(t, "library-bucket", resolved.Bucket)
	require.Equal(t, "private-key", resolved.AccessKeyID)
	require.Equal(t, "private-secret", resolved.SecretAccessKey)
	require.True(t, resolved.ForcePathStyle)
}

func TestResolveLibraryS3ConfigRequiresPrivateCredentials(t *testing.T) {
	_, err := resolveLibraryS3Config(config.LibraryS3Config{}, config.ImageStorageConfig{PublicBaseURL: "https://public.example.test"})
	require.Error(t, err)
}

func TestResolveLibraryS3ConfigNeverInheritsPotentiallyPublicImageBucket(t *testing.T) {
	_, err := resolveLibraryS3Config(config.LibraryS3Config{}, config.ImageStorageConfig{
		Bucket: "public-images", AccessKeyID: "key", SecretAccessKey: "secret",
		PublicBaseURL: "https://cdn.example.test",
	})
	require.ErrorContains(t, err, "bucket")
}

func TestResolveLibraryS3ConfigRejectsExplicitPublicImageBucketReuse(t *testing.T) {
	_, err := resolveLibraryS3Config(
		config.LibraryS3Config{Bucket: "shared-bucket"},
		config.ImageStorageConfig{
			Bucket: "shared-bucket", AccessKeyID: "key", SecretAccessKey: "secret",
			PublicBaseURL: "https://cdn.example.test",
		},
	)
	require.ErrorContains(t, err, "must differ from image_storage bucket")
}

func TestS3LibraryBlobStoreStreamsPrivatePutOpenDelete(t *testing.T) {
	client := newFakeLibraryS3Client()
	store := &s3LibraryBlobStore{client: client, bucket: "private"}
	ctx := context.Background()
	key := "library/7/object"
	payload := []byte("stream me")

	require.NoError(t, store.Put(ctx, key, bytes.NewReader(payload), int64(len(payload)), "text/plain"))
	require.NotNil(t, client.lastPut.ContentLength)
	require.EqualValues(t, len(payload), *client.lastPut.ContentLength)
	require.Equal(t, "text/plain", *client.lastPut.ContentType)
	require.Empty(t, client.lastPut.ACL, "the private store must not request a public ACL")

	reader, err := store.Open(ctx, key)
	require.NoError(t, err)
	actual, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	require.Equal(t, payload, actual)

	require.NoError(t, store.Delete(ctx, key))
	require.NoError(t, store.Delete(ctx, key))
}

func TestS3LibraryBlobStoreRejectsSizeMismatchAndCleansUploadedObject(t *testing.T) {
	t.Run("short", func(t *testing.T) {
		client := newFakeLibraryS3Client()
		store := &s3LibraryBlobStore{client: client, bucket: "private"}
		err := store.Put(context.Background(), "object", bytes.NewReader([]byte("abc")), 4, "")
		require.Error(t, err)
		require.Empty(t, client.objects)
	})

	t.Run("long", func(t *testing.T) {
		client := newFakeLibraryS3Client()
		store := &s3LibraryBlobStore{client: client, bucket: "private"}
		err := store.Put(context.Background(), "object", bytes.NewReader([]byte("abcd")), 3, "")
		require.Error(t, err)
		require.Empty(t, client.objects)
		require.Equal(t, 1, client.deleteCalls)
	})
}

func TestS3LibraryBlobStoreCleanupOrphansKeepsReferencedAndRecent(t *testing.T) {
	client := newFakeLibraryS3Client()
	store := &s3LibraryBlobStore{client: client, bucket: "private"}
	old := time.Now().Add(-48 * time.Hour)
	recent := time.Now()
	client.objects["library/7/referenced"] = []byte("r")
	client.modified["library/7/referenced"] = old
	client.objects["library/7/orphan"] = []byte("o")
	client.modified["library/7/orphan"] = old
	client.objects["library/7/recent"] = []byte("n")
	client.modified["library/7/recent"] = recent

	deleted, err := store.CleanupOrphans(context.Background(), map[string]struct{}{
		"library/7/referenced": {},
	}, time.Now().Add(-24*time.Hour), 10)
	require.NoError(t, err)
	require.Equal(t, 1, deleted)
	require.Contains(t, client.objects, "library/7/referenced")
	require.Contains(t, client.objects, "library/7/recent")
	require.NotContains(t, client.objects, "library/7/orphan")
}

func TestS3LibraryBlobStoreCleanupOrphansPaginatesWithMaxKeys(t *testing.T) {
	base := newFakeLibraryS3Client()
	old := time.Now().Add(-48 * time.Hour)
	base.objects["library/7/orphan"] = []byte("o")
	base.modified["library/7/orphan"] = old
	client := &scriptedLibraryS3Client{fakeLibraryS3Client: base}
	client.list = func(input *s3.ListObjectsV2Input, call int) (*s3.ListObjectsV2Output, error) {
		require.NotNil(t, input.MaxKeys)
		switch call {
		case 1:
			require.EqualValues(t, libraryS3ListPageSize, *input.MaxKeys)
			require.Nil(t, input.ContinuationToken)
			return &s3.ListObjectsV2Output{
				Contents:              []types.Object{{Key: aws.String("library/7/referenced"), LastModified: &old}},
				IsTruncated:           aws.Bool(true),
				NextContinuationToken: aws.String("page-2"),
			}, nil
		case 2:
			require.EqualValues(t, libraryS3ListPageSize-1, *input.MaxKeys)
			require.Equal(t, "page-2", aws.ToString(input.ContinuationToken))
			return &s3.ListObjectsV2Output{
				Contents:    []types.Object{{Key: aws.String("library/7/orphan"), LastModified: &old}},
				IsTruncated: aws.Bool(false),
			}, nil
		default:
			return nil, fmt.Errorf("unexpected list call %d", call)
		}
	}
	store := &s3LibraryBlobStore{client: client, bucket: "private"}

	deleted, err := store.CleanupOrphans(context.Background(), map[string]struct{}{
		"library/7/referenced": {},
	}, time.Now().Add(-24*time.Hour), 1)
	require.NoError(t, err)
	require.Equal(t, 1, deleted)
	require.Equal(t, 2, client.listCalls)
	require.NotContains(t, base.objects, "library/7/orphan")
}

func TestS3LibraryBlobStoreCleanupOrphansRejectsNonProgressingToken(t *testing.T) {
	client := &scriptedLibraryS3Client{fakeLibraryS3Client: newFakeLibraryS3Client()}
	client.list = func(_ *s3.ListObjectsV2Input, _ int) (*s3.ListObjectsV2Output, error) {
		return &s3.ListObjectsV2Output{
			IsTruncated:           aws.Bool(true),
			NextContinuationToken: aws.String("stuck"),
		}, nil
	}
	store := &s3LibraryBlobStore{client: client, bucket: "private"}

	deleted, err := store.CleanupOrphans(context.Background(), nil, time.Now(), 1)
	require.Zero(t, deleted)
	require.ErrorContains(t, err, "non-progressing continuation token")
	require.Equal(t, 2, client.listCalls)
}

func TestS3LibraryBlobStoreCleanupOrphansStopsAtScanBudget(t *testing.T) {
	base := newFakeLibraryS3Client()
	old := time.Now().Add(-48 * time.Hour)
	recent := time.Now()
	base.objects["library/7/z-orphan"] = []byte("o")
	base.modified["library/7/z-orphan"] = old
	recentContents := make([]types.Object, 0, libraryOrphanScanBudget(1))
	for i := 0; i < libraryOrphanScanBudget(1); i++ {
		key := fmt.Sprintf("library/7/a-recent-%04d", i)
		recentContents = append(recentContents, types.Object{Key: aws.String(key), LastModified: &recent})
	}
	client := &scriptedLibraryS3Client{fakeLibraryS3Client: base}
	client.list = func(input *s3.ListObjectsV2Input, _ int) (*s3.ListObjectsV2Output, error) {
		require.EqualValues(t, libraryS3ListPageSize, aws.ToInt32(input.MaxKeys))
		contents := append([]types.Object(nil), recentContents...)
		if _, ok := base.objects["library/7/0-orphan"]; ok {
			contents = append([]types.Object{{Key: aws.String("library/7/0-orphan"), LastModified: &old}}, contents...)
		}
		if _, ok := base.objects["library/7/z-orphan"]; ok {
			contents = append(contents, types.Object{Key: aws.String("library/7/z-orphan"), LastModified: &old})
		}
		return &s3.ListObjectsV2Output{Contents: contents}, nil
	}
	store := &s3LibraryBlobStore{client: client, bucket: "private"}

	deleted, err := store.CleanupOrphans(context.Background(), nil, time.Now().Add(-24*time.Hour), 1)
	require.NoError(t, err)
	require.Zero(t, deleted)
	require.Contains(t, base.objects, "library/7/z-orphan")
	require.Equal(t, 1, client.listCalls)

	deleted, err = store.CleanupOrphans(context.Background(), nil, time.Now().Add(-24*time.Hour), 1)
	require.NoError(t, err)
	require.Equal(t, 1, deleted, "the S3 rolling cursor must reach objects after the first scan budget")
	require.NotContains(t, base.objects, "library/7/z-orphan")

	// Finish the resumed page to wrap, then prove an earlier key is not starved.
	deleted, err = store.CleanupOrphans(context.Background(), nil, time.Now().Add(-24*time.Hour), 1)
	require.NoError(t, err)
	require.Zero(t, deleted)
	base.objects["library/7/0-orphan"] = []byte("o")
	base.modified["library/7/0-orphan"] = old
	deleted, err = store.CleanupOrphans(context.Background(), nil, time.Now().Add(-24*time.Hour), 1)
	require.NoError(t, err)
	require.Equal(t, 1, deleted, "a completed S3 pass must wrap to keys created behind the cursor")
	require.NotContains(t, base.objects, "library/7/0-orphan")
}

func TestS3LibraryBlobStoreCleanupOrphansResetsRejectedStoredToken(t *testing.T) {
	client := &scriptedLibraryS3Client{fakeLibraryS3Client: newFakeLibraryS3Client()}
	client.list = func(input *s3.ListObjectsV2Input, call int) (*s3.ListObjectsV2Output, error) {
		if call == 1 {
			require.Equal(t, "expired-token", aws.ToString(input.ContinuationToken))
			return nil, errors.New("invalid continuation token")
		}
		require.Nil(t, input.ContinuationToken, "a rejected stored token must not wedge later cleanup rounds")
		return &s3.ListObjectsV2Output{IsTruncated: aws.Bool(false)}, nil
	}
	store := &s3LibraryBlobStore{
		client: client, bucket: "private",
		orphanCursor: libraryS3OrphanCursor{continuationToken: "expired-token", afterKey: "library/7/old"},
	}

	_, err := store.CleanupOrphans(context.Background(), nil, time.Now(), 1)
	require.ErrorContains(t, err, "invalid continuation token")
	require.Empty(t, store.orphanCursor)
	deleted, err := store.CleanupOrphans(context.Background(), nil, time.Now(), 1)
	require.NoError(t, err)
	require.Zero(t, deleted)
}

func TestS3LibraryBlobStoreCleanupOrphansBoundsEmptyPages(t *testing.T) {
	client := &scriptedLibraryS3Client{fakeLibraryS3Client: newFakeLibraryS3Client()}
	client.list = func(input *s3.ListObjectsV2Input, call int) (*s3.ListObjectsV2Output, error) {
		require.NotNil(t, input.MaxKeys)
		require.EqualValues(t, libraryS3ListPageSize, *input.MaxKeys)
		return &s3.ListObjectsV2Output{
			IsTruncated:           aws.Bool(true),
			NextContinuationToken: aws.String(fmt.Sprintf("page-%d", call)),
		}, nil
	}
	store := &s3LibraryBlobStore{client: client, bucket: "private"}

	deleted, err := store.CleanupOrphans(context.Background(), nil, time.Now(), 1)
	require.NoError(t, err)
	require.Zero(t, deleted)
	require.Equal(t, 2, client.listCalls, "empty truncated pages must still be bounded")
}

func TestProvideLibraryBlobStoreSelectsLocal(t *testing.T) {
	rootPath := filepath.Join(t.TempDir(), "library")
	store, err := ProvideLibraryBlobStore(&config.Config{Library: config.LibraryConfig{
		StorageDriver: "local",
		StorageDir:    rootPath,
	}})
	require.NoError(t, err)
	local, ok := store.(*localLibraryBlobStore)
	require.True(t, ok)
	t.Cleanup(func() { require.NoError(t, local.root.Close()) })
}

func TestProvideLibraryBlobStoreUsesIndependentDefaultDirectory(t *testing.T) {
	t.Chdir(t.TempDir())
	store, err := ProvideLibraryBlobStore(&config.Config{
		Library:         config.LibraryConfig{StorageDriver: "local"},
		ChatAttachments: config.ChatAttachmentConfig{StorageDir: "./data/chat-attachments"},
	})
	require.NoError(t, err)
	local, ok := store.(*localLibraryBlobStore)
	require.True(t, ok)
	t.Cleanup(func() { require.NoError(t, local.root.Close()) })
	require.Equal(t, filepath.Join(mustAbs(t, "."), "data", "library-files"), local.rootPath)
}

func newTestLocalLibraryBlobStore(t *testing.T, rootPath string) *localLibraryBlobStore {
	t.Helper()
	store, err := NewLocalLibraryBlobStore(rootPath)
	require.NoError(t, err)
	local, ok := store.(*localLibraryBlobStore)
	require.True(t, ok)
	t.Cleanup(func() { require.NoError(t, local.root.Close()) })
	return local
}

func assertFileMode(t *testing.T, path string, expected os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, expected, info.Mode().Perm())
}

func mustAbs(t *testing.T, path string) string {
	t.Helper()
	absolute, err := filepath.Abs(path)
	require.NoError(t, err)
	return absolute
}

type fakeLibraryS3Client struct {
	objects     map[string][]byte
	modified    map[string]time.Time
	lastPut     *s3.PutObjectInput
	deleteCalls int
}

type scriptedLibraryS3Client struct {
	*fakeLibraryS3Client
	listCalls int
	list      func(*s3.ListObjectsV2Input, int) (*s3.ListObjectsV2Output, error)
}

func (c *scriptedLibraryS3Client) ListObjectsV2(_ context.Context, input *s3.ListObjectsV2Input, _ ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	c.listCalls++
	if c.list == nil {
		return c.fakeLibraryS3Client.ListObjectsV2(context.Background(), input)
	}
	return c.list(input, c.listCalls)
}

func newFakeLibraryS3Client() *fakeLibraryS3Client {
	return &fakeLibraryS3Client{objects: make(map[string][]byte), modified: make(map[string]time.Time)}
}

func (c *fakeLibraryS3Client) PutObject(_ context.Context, input *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	c.lastPut = input
	data, err := io.ReadAll(input.Body)
	if err != nil {
		return nil, err
	}
	if input.ContentLength == nil || int64(len(data)) != *input.ContentLength {
		return nil, errors.New("content length mismatch")
	}
	c.objects[*input.Key] = data
	c.modified[*input.Key] = time.Now()
	return &s3.PutObjectOutput{}, nil
}

func (c *fakeLibraryS3Client) GetObject(_ context.Context, input *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	data, ok := c.objects[*input.Key]
	if !ok {
		return nil, os.ErrNotExist
	}
	return &s3.GetObjectOutput{Body: io.NopCloser(bytes.NewReader(data))}, nil
}

func (c *fakeLibraryS3Client) DeleteObject(_ context.Context, input *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	c.deleteCalls++
	delete(c.objects, *input.Key)
	delete(c.modified, *input.Key)
	return &s3.DeleteObjectOutput{}, nil
}

func (c *fakeLibraryS3Client) ListObjectsV2(_ context.Context, input *s3.ListObjectsV2Input, _ ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	prefix := ""
	if input.Prefix != nil {
		prefix = *input.Prefix
	}
	contents := make([]types.Object, 0, len(c.objects))
	for key := range c.objects {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		modified := c.modified[key]
		contents = append(contents, types.Object{Key: aws.String(key), LastModified: &modified})
	}
	return &s3.ListObjectsV2Output{Contents: contents}, nil
}
