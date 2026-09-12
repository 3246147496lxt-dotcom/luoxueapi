package service

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/google/uuid"
)

const SettingKeyAccountImportWatcher = "account_health.import_watcher"

type AccountImportWatcherSettings struct {
	Enabled      bool   `json:"enabled"`
	OutputDir    string `json:"output_dir,omitempty"`
	AutoDiscover bool   `json:"auto_discover"`
	LastSeen     string `json:"last_seen,omitempty"`
}

type AccountImportFile struct {
	Name       string    `json:"name"`
	Size       int64     `json:"size"`
	ModifiedAt time.Time `json:"modified_at"`
	TokenCount int       `json:"token_count"`
	New        bool      `json:"new"`
}

type AccountImportWatcherScanResult struct {
	OperationID string              `json:"operation_id"`
	Files       []AccountImportFile `json:"files"`
}

func (s *AccountHealthService) GetImportWatcherSettings(ctx context.Context) (*AccountImportWatcherSettings, error) {
	cfg := &AccountImportWatcherSettings{}
	if s == nil || s.settingsRepo == nil {
		return cfg, nil
	}
	raw, err := s.settingsRepo.GetValue(ctx, SettingKeyAccountImportWatcher)
	if err != nil || strings.TrimSpace(raw) == "" {
		return cfg, nil
	}
	if err := json.Unmarshal([]byte(raw), cfg); err != nil {
		return &AccountImportWatcherSettings{}, nil
	}
	cfg.OutputDir = strings.TrimSpace(cfg.OutputDir)
	return cfg, nil
}

func (s *AccountHealthService) UpdateImportWatcherSettings(ctx context.Context, cfg *AccountImportWatcherSettings) (*AccountImportWatcherSettings, error) {
	if cfg == nil {
		return nil, infraerrors.BadRequest("IMPORT_WATCHER_SETTINGS_REQUIRED", "settings are required")
	}
	out := *cfg
	out.OutputDir = strings.TrimSpace(out.OutputDir)
	// Preserve the discovery cursor when callers update only the directory or
	// toggle. The cursor is advanced by a successful import operation.
	if out.LastSeen == "" && s != nil && s.settingsRepo != nil {
		if current, getErr := s.GetImportWatcherSettings(ctx); getErr == nil {
			out.LastSeen = current.LastSeen
		}
	}
	if out.Enabled && out.OutputDir == "" {
		return nil, infraerrors.BadRequest("IMPORT_WATCHER_OUTPUT_DIR_REQUIRED", "output_dir is required when watcher is enabled")
	}
	if out.OutputDir != "" {
		if info, err := os.Stat(out.OutputDir); err != nil || !info.IsDir() {
			return nil, infraerrors.BadRequest("IMPORT_WATCHER_OUTPUT_DIR_INVALID", "output_dir must be an existing directory")
		}
	}
	if s != nil && s.settingsRepo != nil {
		payload, err := json.Marshal(out)
		if err != nil {
			return nil, err
		}
		if err := s.settingsRepo.Set(ctx, SettingKeyAccountImportWatcher, string(payload)); err != nil {
			return nil, err
		}
	}
	return &out, nil
}

// ScanImportWatcher lists registration exports without returning credentials.
func (s *AccountHealthService) ScanImportWatcher(ctx context.Context) (*AccountImportWatcherScanResult, error) {
	cfg, err := s.GetImportWatcherSettings(ctx)
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled || cfg.OutputDir == "" {
		return &AccountImportWatcherScanResult{OperationID: uuid.NewString(), Files: []AccountImportFile{}}, nil
	}
	entries, err := os.ReadDir(cfg.OutputDir)
	if err != nil {
		return nil, infraerrors.BadRequest("IMPORT_WATCHER_READ_FAILED", "unable to read registration output directory")
	}
	lastSeen, _ := time.Parse(time.RFC3339Nano, cfg.LastSeen)
	files := make([]AccountImportFile, 0)
	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || !strings.HasPrefix(entry.Name(), "accounts_") || !strings.HasSuffix(entry.Name(), ".txt") {
			continue
		}
		info, err := entry.Info()
		if err != nil || info.Size() > 10<<20 {
			continue
		}
		content, err := os.ReadFile(filepath.Join(cfg.OutputDir, entry.Name()))
		if err != nil {
			continue
		}
		count := len(parseAccountHealthHandoff(string(content), PlatformGrok))
		files = append(files, AccountImportFile{Name: entry.Name(), Size: info.Size(), ModifiedAt: info.ModTime(), TokenCount: count, New: info.ModTime().After(lastSeen)})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].ModifiedAt.After(files[j].ModifiedAt) })
	return &AccountImportWatcherScanResult{OperationID: uuid.NewString(), Files: files}, nil
}

func (s *AccountHealthService) ImportWatcherFile(ctx context.Context, file string, indexes []int, proxyID *int64, groupIDs []int64, confirm, operationID string) (*AccountHealthAddResult, error) {
	cfg, err := s.GetImportWatcherSettings(ctx)
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled || cfg.OutputDir == "" {
		return nil, infraerrors.BadRequest("IMPORT_WATCHER_DISABLED", "registration watcher is disabled")
	}
	name := filepath.Base(strings.TrimSpace(file))
	if name == "." || name != file || !strings.HasPrefix(name, "accounts_") || !strings.HasSuffix(name, ".txt") {
		return nil, infraerrors.BadRequest("IMPORT_WATCHER_FILE_INVALID", "file must be an accounts_*.txt export")
	}
	baseDir, err := filepath.EvalSymlinks(cfg.OutputDir)
	if err != nil {
		return nil, infraerrors.BadRequest("IMPORT_WATCHER_FILE_READ_FAILED", "unable to read registration export")
	}
	path := filepath.Join(baseDir, name)
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, infraerrors.BadRequest("IMPORT_WATCHER_FILE_READ_FAILED", "unable to read registration export")
	}
	rel, err := filepath.Rel(baseDir, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, infraerrors.BadRequest("IMPORT_WATCHER_FILE_INVALID", "file must stay inside output_dir")
	}
	content, err := os.ReadFile(resolved)
	if err != nil {
		return nil, infraerrors.BadRequest("IMPORT_WATCHER_FILE_READ_FAILED", "unable to read registration export")
	}
	// An empty index list means import every parsed credential in this export.
	// The watcher intentionally exposes only file metadata to the UI, so the
	// server resolves the indexes after reading the file.
	if len(indexes) == 0 {
		parsed := parseAccountHealthHandoff(string(content), PlatformGrok)
		indexes = make([]int, 0, len(parsed))
		for i := range parsed {
			indexes = append(indexes, i)
		}
	}
	result, err := s.Add(ctx, AccountHealthAddRequest{Handoff: string(content), Indexes: indexes, Confirm: confirm, Platform: PlatformGrok, ProxyID: proxyID, GroupIDs: groupIDs, OperationID: operationID})
	if err != nil {
		return nil, err
	}
	if info, statErr := os.Stat(resolved); statErr == nil {
		// Never move the cursor backwards when a batch is imported newest-first.
		seenAt := info.ModTime().UTC()
		lastSeen, _ := time.Parse(time.RFC3339Nano, cfg.LastSeen)
		if lastSeen.IsZero() || seenAt.After(lastSeen) {
			cfg.LastSeen = seenAt.Format(time.RFC3339Nano)
			_, _ = s.UpdateImportWatcherSettings(ctx, cfg)
		}
	}
	return result, nil
}
