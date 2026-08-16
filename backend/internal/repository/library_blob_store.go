package repository

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

const defaultLibraryStorageDir = "./data/library-files"

const (
	libraryOrphanMinScanBudget = 1_000
	libraryOrphanMaxScanBudget = 100_000
	libraryOrphanScanFactor    = 100
	libraryS3ListPageSize      = 1_000
)

// ProvideLibraryBlobStore selects the configured private library object store.
func ProvideLibraryBlobStore(cfg *config.Config) (service.LibraryBlobStore, error) {
	if cfg == nil {
		return nil, errors.New("library blob store requires config")
	}

	switch strings.ToLower(strings.TrimSpace(cfg.Library.StorageDriver)) {
	case "", "local":
		storageDir := strings.TrimSpace(cfg.Library.StorageDir)
		if storageDir == "" {
			storageDir = defaultLibraryStorageDir
		}
		return NewLocalLibraryBlobStore(storageDir)
	case "s3":
		resolved, err := resolveLibraryS3Config(cfg.Library.S3, cfg.ImageStorage)
		if err != nil {
			return nil, err
		}
		return newS3LibraryBlobStore(context.Background(), resolved)
	default:
		return nil, fmt.Errorf("unsupported library storage driver %q", cfg.Library.StorageDriver)
	}
}

type resolvedLibraryS3Config struct {
	Endpoint        string
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	ForcePathStyle  bool
}

func resolveLibraryS3Config(library config.LibraryS3Config, fallback config.ImageStorageConfig) (resolvedLibraryS3Config, error) {
	resolved := resolvedLibraryS3Config{
		Endpoint: firstNonBlank(library.Endpoint, fallback.Endpoint),
		Region:   firstNonBlank(library.Region, fallback.Region),
		// The bucket is never inherited from image_storage: that bucket may be
		// intentionally public through PublicBaseURL or bucket policy. Credentials
		// and endpoint may be shared, but private library data needs an explicit
		// bucket boundary.
		Bucket:          strings.TrimSpace(library.Bucket),
		AccessKeyID:     firstNonBlank(library.AccessKeyID, fallback.AccessKeyID),
		SecretAccessKey: firstNonBlank(library.SecretAccessKey, fallback.SecretAccessKey),
		ForcePathStyle:  library.ForcePathStyle || fallback.ForcePathStyle,
	}

	var missing []string
	if resolved.Bucket == "" {
		missing = append(missing, "bucket")
	}
	if resolved.AccessKeyID == "" {
		missing = append(missing, "access_key_id")
	}
	if resolved.SecretAccessKey == "" {
		missing = append(missing, "secret_access_key")
	}
	if len(missing) != 0 {
		return resolvedLibraryS3Config{}, fmt.Errorf("library S3 configuration is missing %s", strings.Join(missing, ", "))
	}
	if strings.TrimSpace(fallback.PublicBaseURL) != "" && resolved.Bucket == strings.TrimSpace(fallback.Bucket) {
		return resolvedLibraryS3Config{}, errors.New("library S3 bucket must differ from image_storage bucket when image_storage.public_base_url is configured")
	}
	return resolved, nil
}

func libraryOrphanScanBudget(deleteLimit int) int {
	if deleteLimit <= 0 {
		return 0
	}
	if deleteLimit >= libraryOrphanMaxScanBudget/libraryOrphanScanFactor {
		return libraryOrphanMaxScanBudget
	}
	budget := deleteLimit * libraryOrphanScanFactor
	if budget < libraryOrphanMinScanBudget {
		return libraryOrphanMinScanBudget
	}
	return budget
}

func firstNonBlank(primary, fallback string) string {
	if value := strings.TrimSpace(primary); value != "" {
		return value
	}
	return strings.TrimSpace(fallback)
}

// validateLibraryBlobKey accepts only canonical, relative, slash-separated
// object keys. The same validation is used for local paths and S3 keys so a
// storage-driver change cannot reinterpret an existing key.
func validateLibraryBlobKey(key string) error {
	if !fs.ValidPath(key) || key == "." || path.Clean(key) != key || strings.ContainsRune(key, '\x00') || strings.Contains(key, `\`) {
		return fmt.Errorf("invalid library blob key %q", key)
	}
	return nil
}

type contextAwareReader struct {
	ctx context.Context
	r   io.Reader
}

func (r *contextAwareReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.r.Read(p)
}

type contextAwareReadCloser struct {
	ctx context.Context
	io.ReadCloser
}

func (r *contextAwareReadCloser) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.ReadCloser.Read(p)
}

func readerHasExtraByte(ctx context.Context, body io.Reader) (bool, error) {
	reader := &contextAwareReader{ctx: ctx, r: body}
	var probe [1]byte
	for {
		n, err := reader.Read(probe[:])
		if n > 0 {
			return true, nil
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return false, nil
			}
			return false, err
		}
	}
}
