package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type localLibraryBlobStore struct {
	root         *os.Root
	rootPath     string
	orphanMu     sync.Mutex
	orphanCursor string
}

var _ service.LibraryBlobStore = (*localLibraryBlobStore)(nil)

// Close releases the root directory descriptor held for path confinement.
// LibraryService invokes it after its cleanup worker has stopped.
func (s *localLibraryBlobStore) Close() error {
	if s == nil || s.root == nil {
		return nil
	}
	return s.root.Close()
}

// NewLocalLibraryBlobStore creates a private, root-confined local blob store.
func NewLocalLibraryBlobStore(storageDir string) (service.LibraryBlobStore, error) {
	storageDir = strings.TrimSpace(storageDir)
	if storageDir == "" {
		return nil, errors.New("library local storage directory must not be empty")
	}
	absolute, err := filepath.Abs(storageDir)
	if err != nil {
		return nil, fmt.Errorf("resolve library storage directory: %w", err)
	}
	if err = os.MkdirAll(absolute, 0o700); err != nil {
		return nil, fmt.Errorf("create library storage directory: %w", err)
	}
	info, err := os.Lstat(absolute)
	if err != nil {
		return nil, fmt.Errorf("inspect library storage directory: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, errors.New("library local storage root must be a real directory")
	}
	if err = os.Chmod(absolute, 0o700); err != nil {
		return nil, fmt.Errorf("secure library storage directory: %w", err)
	}
	root, err := os.OpenRoot(absolute)
	if err != nil {
		return nil, fmt.Errorf("open library storage root: %w", err)
	}
	return &localLibraryBlobStore{root: root, rootPath: absolute}, nil
}

func (s *localLibraryBlobStore) Put(ctx context.Context, key string, body io.Reader, size int64, _ string) error {
	if err := s.validateOperation(ctx, key, body, size); err != nil {
		return err
	}
	parent := path.Dir(key)
	if err := s.ensurePrivateDirectories(parent); err != nil {
		return err
	}

	tempKey, temp, err := s.createPrivateTemp(parent)
	if err != nil {
		return err
	}
	keepTemp := true
	defer func() {
		_ = temp.Close()
		if keepTemp {
			_ = s.root.Remove(tempKey)
		}
	}()

	reader := &contextAwareReader{ctx: ctx, r: body}
	written, copyErr := io.CopyN(temp, reader, size)
	if copyErr != nil {
		return fmt.Errorf("write library blob %q: expected %d bytes, received %d: %w", key, size, written, copyErr)
	}
	hasExtra, err := readerHasExtraByte(ctx, body)
	if err != nil {
		return fmt.Errorf("verify library blob %q size: %w", key, err)
	}
	if hasExtra {
		return fmt.Errorf("write library blob %q: body exceeds declared size %d", key, size)
	}
	if err = temp.Sync(); err != nil {
		return fmt.Errorf("sync library blob %q: %w", key, err)
	}
	if err = temp.Close(); err != nil {
		return fmt.Errorf("close library blob %q: %w", key, err)
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if err = s.root.Rename(tempKey, key); err != nil {
		return fmt.Errorf("commit library blob %q: %w", key, err)
	}
	keepTemp = false
	if err = s.syncDirectory(parent); err != nil {
		return fmt.Errorf("sync library blob directory: %w", err)
	}
	return nil
}

func (s *localLibraryBlobStore) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := s.validateKeyAndContext(ctx, key); err != nil {
		return nil, err
	}
	before, err := s.root.Lstat(key)
	if err != nil {
		return nil, fmt.Errorf("inspect library blob %q: %w", key, err)
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return nil, fmt.Errorf("library blob %q is not a regular file", key)
	}
	file, err := s.root.Open(key)
	if err != nil {
		return nil, fmt.Errorf("open library blob %q: %w", key, err)
	}
	after, err := file.Stat()
	if err != nil || !after.Mode().IsRegular() || !os.SameFile(before, after) {
		_ = file.Close()
		if err != nil {
			return nil, fmt.Errorf("inspect opened library blob %q: %w", key, err)
		}
		return nil, fmt.Errorf("library blob %q changed while opening", key)
	}
	return &contextAwareReadCloser{ctx: ctx, ReadCloser: file}, nil
}

func (s *localLibraryBlobStore) Delete(ctx context.Context, key string) error {
	if err := s.validateKeyAndContext(ctx, key); err != nil {
		return err
	}
	info, err := s.root.Lstat(key)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("inspect library blob %q: %w", key, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("library blob %q is not a regular file", key)
	}
	if err = s.root.Remove(key); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete library blob %q: %w", key, err)
	}
	if err == nil {
		if syncErr := s.syncDirectory(path.Dir(key)); syncErr != nil {
			return fmt.Errorf("sync library blob directory: %w", syncErr)
		}
	}
	return nil
}

func (s *localLibraryBlobStore) CleanupOrphans(ctx context.Context, referenced map[string]struct{}, olderThan time.Time, limit int) (int, error) {
	if s == nil || s.root == nil || ctx == nil {
		return 0, errors.New("library local blob store is not initialized")
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if limit <= 0 {
		return 0, nil
	}
	// WalkDir starts at the root on every invocation. Preserve a lexical cursor
	// across cleanup rounds so a large referenced prefix cannot permanently
	// starve objects that sort after the per-run scan budget. The lock also
	// prevents concurrent janitor calls from racing the shared cursor.
	s.orphanMu.Lock()
	defer s.orphanMu.Unlock()
	startAfter := s.orphanCursor
	deleted := 0
	scanned := 0
	scanBudget := libraryOrphanScanBudget(limit)
	lastScanned := startAfter
	stoppedEarly := false
	err := fs.WalkDir(s.root.FS(), ".", func(key string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if key == "." {
			return nil
		}
		if startAfter != "" && key <= startAfter {
			// WalkDir is lexical. A directory that cannot contain the cursor can
			// be skipped wholesale, keeping resume work proportional to depth
			// instead of rescanning the whole already-visited prefix.
			if entry.IsDir() && !strings.HasPrefix(startAfter, key+"/") {
				return fs.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if deleted >= limit || scanned >= scanBudget {
			stoppedEarly = true
			return fs.SkipAll
		}
		scanned++
		lastScanned = key
		if entry.Type()&os.ModeSymlink != 0 {
			return stopLibraryLocalOrphanWalkIfBounded(&stoppedEarly, deleted, limit, scanned, scanBudget)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || !info.ModTime().Before(olderThan) {
			return nil
		}
		if strings.HasPrefix(path.Base(key), ".library-blob-") {
			if err = s.root.Remove(key); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
			deleted++
			return stopLibraryLocalOrphanWalkIfBounded(&stoppedEarly, deleted, limit, scanned, scanBudget)
		}
		if validateLibraryBlobKey(key) != nil || !strings.HasPrefix(key, "library/") {
			return stopLibraryLocalOrphanWalkIfBounded(&stoppedEarly, deleted, limit, scanned, scanBudget)
		}
		if _, ok := referenced[key]; ok {
			return stopLibraryLocalOrphanWalkIfBounded(&stoppedEarly, deleted, limit, scanned, scanBudget)
		}
		if err = s.root.Remove(key); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		deleted++
		return stopLibraryLocalOrphanWalkIfBounded(&stoppedEarly, deleted, limit, scanned, scanBudget)
	})
	if err != nil {
		return deleted, err
	}
	if stoppedEarly {
		s.orphanCursor = lastScanned
	} else {
		// Reaching the lexical end completes a pass. Reset so files created
		// behind the cursor are included in the next cleanup round.
		s.orphanCursor = ""
	}
	return deleted, err
}

func stopLibraryLocalOrphanWalkIfBounded(stoppedEarly *bool, deleted, deleteLimit, scanned, scanBudget int) error {
	if deleted < deleteLimit && scanned < scanBudget {
		return nil
	}
	*stoppedEarly = true
	return fs.SkipAll
}

func (s *localLibraryBlobStore) validateOperation(ctx context.Context, key string, body io.Reader, size int64) error {
	if body == nil {
		return errors.New("library blob body must not be nil")
	}
	if size < 0 {
		return errors.New("library blob size must not be negative")
	}
	return s.validateKeyAndContext(ctx, key)
}

func (s *localLibraryBlobStore) validateKeyAndContext(ctx context.Context, key string) error {
	if s == nil || s.root == nil {
		return errors.New("library local blob store is not initialized")
	}
	if ctx == nil {
		return errors.New("library blob context must not be nil")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return validateLibraryBlobKey(key)
}

func (s *localLibraryBlobStore) ensurePrivateDirectories(parent string) error {
	if parent == "." {
		return nil
	}
	if err := s.root.MkdirAll(parent, 0o700); err != nil {
		return fmt.Errorf("create library blob directory: %w", err)
	}
	current := ""
	for _, segment := range strings.Split(parent, "/") {
		if current == "" {
			current = segment
		} else {
			current += "/" + segment
		}
		info, err := s.root.Lstat(current)
		if err != nil {
			return fmt.Errorf("inspect library blob directory: %w", err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("library blob directory %q is not a real directory", current)
		}
		if err = s.root.Chmod(current, 0o700); err != nil {
			return fmt.Errorf("secure library blob directory: %w", err)
		}
	}
	return nil
}

func (s *localLibraryBlobStore) createPrivateTemp(parent string) (string, *os.File, error) {
	for attempt := 0; attempt < 10; attempt++ {
		var random [12]byte
		if _, err := rand.Read(random[:]); err != nil {
			return "", nil, fmt.Errorf("generate library blob temporary name: %w", err)
		}
		name := ".library-blob-" + hex.EncodeToString(random[:])
		if parent != "." {
			name = parent + "/" + name
		}
		file, err := s.root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
		if err != nil {
			if errors.Is(err, os.ErrExist) {
				continue
			}
			return "", nil, fmt.Errorf("create library blob temporary file: %w", err)
		}
		if err = file.Chmod(0o600); err != nil {
			_ = file.Close()
			_ = s.root.Remove(name)
			return "", nil, fmt.Errorf("secure library blob temporary file: %w", err)
		}
		return name, file, nil
	}
	return "", nil, errors.New("could not allocate library blob temporary file")
}

func (s *localLibraryBlobStore) syncDirectory(directory string) error {
	dir, err := s.root.Open(directory)
	if err != nil {
		return err
	}
	syncErr := dir.Sync()
	closeErr := dir.Close()
	if syncErr != nil {
		return syncErr
	}
	return closeErr
}
