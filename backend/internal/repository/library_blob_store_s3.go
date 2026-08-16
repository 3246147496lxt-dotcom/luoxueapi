package repository

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type libraryS3API interface {
	PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	DeleteObject(context.Context, *s3.DeleteObjectInput, ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	ListObjectsV2(context.Context, *s3.ListObjectsV2Input, ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
}

type s3LibraryBlobStore struct {
	client       libraryS3API
	bucket       string
	orphanMu     sync.Mutex
	orphanCursor libraryS3OrphanCursor
}

// libraryS3OrphanCursor identifies a position inside an S3 listing page. The
// continuation token points to the start of that page and afterKey records the
// last object processed in it. Keeping both prevents a delete/scan limit hit in
// the middle of a page from skipping the page's remaining objects next round.
type libraryS3OrphanCursor struct {
	continuationToken string
	afterKey          string
}

var _ service.LibraryBlobStore = (*s3LibraryBlobStore)(nil)

func newS3LibraryBlobStore(ctx context.Context, cfg resolvedLibraryS3Config) (service.LibraryBlobStore, error) {
	client, err := newS3Client(ctx, s3ClientParams{
		Endpoint:        cfg.Endpoint,
		Region:          cfg.Region,
		AccessKeyID:     cfg.AccessKeyID,
		SecretAccessKey: cfg.SecretAccessKey,
		ForcePathStyle:  cfg.ForcePathStyle,
	})
	if err != nil {
		return nil, err
	}
	return &s3LibraryBlobStore{client: client, bucket: cfg.Bucket}, nil
}

func (s *s3LibraryBlobStore) Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error {
	if err := s.validateOperation(ctx, key, body, size); err != nil {
		return err
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	exact := &s3ExactSizeReader{ctx: ctx, source: body, remaining: size}
	// Intentionally omit ACL/public URL/presigning fields. S3's default object
	// ACL is private, and ACL headers break bucket-owner-enforced deployments.
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        &s.bucket,
		Key:           &key,
		Body:          exact,
		ContentLength: &size,
		ContentType:   &contentType,
	})
	if err != nil {
		return fmt.Errorf("put private library S3 object %q: %w", key, err)
	}
	if exact.remaining != 0 {
		s.cleanupFailedPut(ctx, key)
		return fmt.Errorf("put private library S3 object %q: expected %d bytes, received %d", key, size, size-exact.remaining)
	}
	hasExtra, err := readerHasExtraByte(ctx, body)
	if err != nil {
		s.cleanupFailedPut(ctx, key)
		return fmt.Errorf("verify private library S3 object %q size: %w", key, err)
	}
	if hasExtra {
		s.cleanupFailedPut(ctx, key)
		return fmt.Errorf("put private library S3 object %q: body exceeds declared size %d", key, size)
	}
	return nil
}

func (s *s3LibraryBlobStore) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := s.validateKeyAndContext(ctx, key); err != nil {
		return nil, err
	}
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: &s.bucket, Key: &key})
	if err != nil {
		return nil, fmt.Errorf("get private library S3 object %q: %w", key, err)
	}
	if result == nil || result.Body == nil {
		return nil, fmt.Errorf("get private library S3 object %q: empty response body", key)
	}
	return &contextAwareReadCloser{ctx: ctx, ReadCloser: result.Body}, nil
}

func (s *s3LibraryBlobStore) Delete(ctx context.Context, key string) error {
	if err := s.validateKeyAndContext(ctx, key); err != nil {
		return err
	}
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &s.bucket, Key: &key})
	if err != nil {
		return fmt.Errorf("delete private library S3 object %q: %w", key, err)
	}
	return nil
}

func (s *s3LibraryBlobStore) CleanupOrphans(ctx context.Context, referenced map[string]struct{}, olderThan time.Time, limit int) (int, error) {
	if s == nil || s.client == nil || s.bucket == "" || ctx == nil {
		return 0, errors.New("library S3 blob store is not initialized")
	}
	if limit <= 0 {
		return 0, nil
	}
	// The cursor is shared across janitor rounds; serialize callers so each run
	// advances from the exact point committed by the previous one.
	s.orphanMu.Lock()
	defer s.orphanMu.Unlock()
	prefix := "library/"
	deleted := 0
	scanned := 0
	pages := 0
	scanBudget := libraryOrphanScanBudget(limit)
	pageBudget := (scanBudget+libraryS3ListPageSize-1)/libraryS3ListPageSize + 1
	cursor := s.orphanCursor
	continuationToken := strings.TrimSpace(cursor.continuationToken)
	afterKey := strings.TrimSpace(cursor.afterKey)
	seenTokens := make(map[string]struct{}, pageBudget)
	if continuationToken != "" {
		seenTokens[continuationToken] = struct{}{}
	}
	for deleted < limit && scanned < scanBudget && pages < pageBudget {
		if err := ctx.Err(); err != nil {
			return deleted, err
		}
		pageSize := scanBudget - scanned
		if pageSize > libraryS3ListPageSize {
			pageSize = libraryS3ListPageSize
		}
		result, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket: &s.bucket, Prefix: &prefix,
			ContinuationToken: libraryOptionalS3String(continuationToken),
			MaxKeys:           aws.Int32(int32(pageSize)),
		})
		pages++
		if err != nil {
			// Stored continuation tokens are opaque and may expire or be rejected
			// by an S3-compatible service. Reset to a safe full-pass start so the
			// next cleanup retries without becoming permanently wedged.
			if continuationToken != "" {
				s.orphanCursor = libraryS3OrphanCursor{}
			}
			return deleted, fmt.Errorf("list private library S3 objects: %w", err)
		}
		if result == nil {
			s.orphanCursor = libraryS3OrphanCursor{}
			return deleted, nil
		}
		resumeAfterKey := afterKey
		for _, object := range result.Contents {
			if object.Key == nil {
				continue
			}
			key := *object.Key
			// A resumed page may no longer contain the exact last key because a
			// previous pass deleted it. S3 listings are lexical, so <= is the
			// stable resume predicate in both cases.
			if resumeAfterKey != "" && key <= resumeAfterKey {
				continue
			}
			if deleted >= limit || scanned >= scanBudget {
				s.orphanCursor = libraryS3OrphanCursor{continuationToken: continuationToken, afterKey: afterKey}
				return deleted, nil
			}
			scanned++
			afterKey = key
			if object.LastModified == nil || !object.LastModified.Before(olderThan) {
				continue
			}
			if !strings.HasPrefix(key, prefix) || validateLibraryBlobKey(key) != nil {
				continue
			}
			if _, ok := referenced[key]; ok {
				continue
			}
			if _, err = s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &s.bucket, Key: &key}); err != nil {
				return deleted, fmt.Errorf("delete orphan private library S3 object %q: %w", key, err)
			}
			deleted++
			if deleted >= limit || scanned >= scanBudget {
				s.orphanCursor = libraryS3OrphanCursor{continuationToken: continuationToken, afterKey: afterKey}
				return deleted, nil
			}
		}
		if result.IsTruncated == nil || !*result.IsTruncated {
			// End of the lexical pass: wrap so newly-created keys that sort before
			// the previous cursor are included on the next run.
			s.orphanCursor = libraryS3OrphanCursor{}
			return deleted, nil
		}
		nextToken := strings.TrimSpace(aws.ToString(result.NextContinuationToken))
		if nextToken == "" || nextToken == continuationToken {
			s.orphanCursor = libraryS3OrphanCursor{}
			return deleted, errors.New("list private library S3 objects returned a non-progressing continuation token")
		}
		if _, ok := seenTokens[nextToken]; ok {
			s.orphanCursor = libraryS3OrphanCursor{}
			return deleted, errors.New("list private library S3 objects returned a repeated continuation token")
		}
		seenTokens[nextToken] = struct{}{}
		continuationToken = nextToken
		afterKey = ""
	}
	// A page budget can stop a pathological sequence of empty/truncated pages
	// before any object consumes the scan budget. Persist the next page token so
	// later rounds still make forward progress instead of restarting at page one.
	s.orphanCursor = libraryS3OrphanCursor{continuationToken: continuationToken, afterKey: afterKey}
	return deleted, nil
}

func libraryOptionalS3String(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return aws.String(value)
}

func (s *s3LibraryBlobStore) validateOperation(ctx context.Context, key string, body io.Reader, size int64) error {
	if body == nil {
		return errors.New("library blob body must not be nil")
	}
	if size < 0 {
		return errors.New("library blob size must not be negative")
	}
	return s.validateKeyAndContext(ctx, key)
}

func (s *s3LibraryBlobStore) validateKeyAndContext(ctx context.Context, key string) error {
	if s == nil || s.client == nil || s.bucket == "" {
		return errors.New("library S3 blob store is not initialized")
	}
	if ctx == nil {
		return errors.New("library blob context must not be nil")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return validateLibraryBlobKey(key)
}

func (s *s3LibraryBlobStore) cleanupFailedPut(ctx context.Context, key string) {
	if ctx == nil || ctx.Err() != nil {
		ctx = context.Background()
	}
	_, _ = s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &s.bucket, Key: &key})
}

type s3ExactSizeReader struct {
	ctx       context.Context
	source    io.Reader
	remaining int64
}

func (r *s3ExactSizeReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	if r.remaining == 0 {
		return 0, io.EOF
	}
	if int64(len(p)) > r.remaining {
		p = p[:r.remaining]
	}
	n, err := r.source.Read(p)
	r.remaining -= int64(n)
	if r.remaining == 0 {
		if n > 0 {
			return n, nil
		}
		return 0, io.EOF
	}
	if errors.Is(err, io.EOF) {
		return n, io.ErrUnexpectedEOF
	}
	return n, err
}
