package transcriptiontemp

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	directoryName = "sub2api-transcription"
	staleAfter    = time.Hour
	sweepInterval = 10 * time.Minute
)

var (
	defaultPreparer = &workspacePreparer{}
	maintenanceOnce sync.Once
)

type workspacePreparer struct {
	mu          sync.Mutex
	preparedDir string
}

// Prepare creates the process-shared private transcription workspace and
// removes files left by a crashed prior process after the maximum supported
// request lifetime has safely elapsed.
func Prepare() (string, error) {
	return defaultPreparer.Prepare(os.TempDir(), time.Now())
}

// StartMaintenance performs an immediate crash-leftover sweep and keeps
// retrying in the background. It is intentionally safe to call even when STT
// is disabled so a restart with the feature off still cleans prior audio.
func StartMaintenance() error {
	maintenanceOnce.Do(func() { go runJanitor() })
	_, err := Prepare()
	return err
}

func (p *workspacePreparer) Prepare(baseDir string, now time.Time) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.preparedDir != "" {
		if err := secureWorkspace(p.preparedDir); err == nil {
			return p.preparedDir, nil
		}
		// A host tmp cleaner may remove the directory, or an unsafe replacement
		// may appear after startup. Drop the cache and run the full create/owner
		// validation path again instead of remaining permanently unavailable.
		p.preparedDir = ""
	}
	dir, err := prepare(baseDir, now)
	if err != nil {
		// Do not cache failures: Create and the maintenance loop must be able to
		// recover after an operator repairs permissions without a process restart.
		return "", err
	}
	p.preparedDir = dir
	return dir, nil
}

// Create creates a private temporary file in the transcription workspace.
// Callers still own closing and removing the returned file.
func Create(pattern string) (*os.File, error) {
	dir, err := Prepare()
	if err != nil {
		return nil, err
	}
	if pattern == "" || strings.ContainsAny(pattern, `/\\`) {
		return nil, errors.New("invalid transcription temporary file pattern")
	}
	file, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return nil, fmt.Errorf("create transcription temporary file: %w", err)
	}
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		_ = os.Remove(file.Name())
		return nil, fmt.Errorf("secure transcription temporary file: %w", err)
	}
	return file, nil
}

func prepare(baseDir string, now time.Time) (string, error) {
	baseDir = strings.TrimSpace(baseDir)
	if baseDir == "" {
		return "", errors.New("operating system temporary directory is unavailable")
	}
	dir := filepath.Join(baseDir, directoryName)
	if err := os.Mkdir(dir, 0o700); err != nil && !errors.Is(err, fs.ErrExist) {
		return "", fmt.Errorf("create transcription temporary directory: %w", err)
	}
	if err := secureWorkspace(dir); err != nil {
		return "", err
	}
	if err := removeStaleFiles(dir, now.Add(-staleAfter)); err != nil {
		return "", err
	}
	return dir, nil
}

func secureWorkspace(dir string) error {
	info, err := os.Lstat(dir)
	if err != nil {
		return fmt.Errorf("inspect transcription temporary directory: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("transcription temporary path is not a private directory")
	}
	if err := verifyWorkspaceOwnership(info); err != nil {
		return err
	}
	if info.Mode().Perm() != 0o700 {
		if err := os.Chmod(dir, 0o700); err != nil {
			return fmt.Errorf("secure transcription temporary directory: %w", err)
		}
	}
	return nil
}

func removeStaleFiles(dir string, cutoff time.Time) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("list transcription temporary directory: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || !ownedFileName(entry.Name()) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return fmt.Errorf("inspect stale transcription file: %w", err)
		}
		if !info.Mode().IsRegular() || !info.ModTime().Before(cutoff) {
			continue
		}
		if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("remove stale transcription file: %w", err)
		}
	}
	return nil
}

func runJanitor() {
	ticker := time.NewTicker(sweepInterval)
	defer ticker.Stop()
	for now := range ticker.C {
		dir, err := Prepare()
		if err != nil {
			continue
		}
		// A failed sweep is retried at the next interval. Request-time cleanup
		// remains deterministic, while janitor errors never expose file names or
		// transcript data through logs.
		_ = removeStaleFiles(dir, now.Add(-staleAfter))
	}
}

func ownedFileName(name string) bool {
	return strings.HasPrefix(name, "audio-") || strings.HasPrefix(name, "upstream-")
}
