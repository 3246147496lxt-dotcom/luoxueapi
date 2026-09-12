package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type importWatcherRepo struct{ value string }

func (r *importWatcherRepo) GetValue(_ context.Context, key string) (string, error) {
	if key == SettingKeyAccountImportWatcher {
		return r.value, nil
	}
	return "", nil
}
func (r *importWatcherRepo) Set(_ context.Context, key, value string) error {
	if key == SettingKeyAccountImportWatcher {
		r.value = value
	}
	return nil
}
func (r *importWatcherRepo) Get(context.Context, string) (*Setting, error) { return nil, nil }
func (r *importWatcherRepo) GetMultiple(context.Context, []string) (map[string]string, error) {
	return nil, nil
}
func (r *importWatcherRepo) SetMultiple(context.Context, map[string]string) error { return nil }
func (r *importWatcherRepo) GetAll(context.Context) (map[string]string, error)    { return nil, nil }
func (r *importWatcherRepo) Delete(context.Context, string) error                 { return nil }

func TestScanImportWatcherListsOnlyRegistrationExports(t *testing.T) {
	dir := t.TempDir()
	token := "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJzZXNzaW9uX2lkIjoiYWJjIn0.signature"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "accounts_demo.txt"), []byte(token+"\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "other.txt"), []byte(token), 0600))
	repo := &importWatcherRepo{}
	svc := NewAccountHealthService(nil, nil)
	svc.SetSettingsRepository(repo)
	_, err := svc.UpdateImportWatcherSettings(context.Background(), &AccountImportWatcherSettings{Enabled: true, OutputDir: dir, AutoDiscover: true})
	require.NoError(t, err)
	result, err := svc.ScanImportWatcher(context.Background())
	require.NoError(t, err)
	require.Len(t, result.Files, 1)
	require.Equal(t, "accounts_demo.txt", result.Files[0].Name)
	require.Equal(t, 1, result.Files[0].TokenCount)
	require.NotEmpty(t, result.OperationID)
}

func TestImportWatcherSettingsRejectMissingDirectory(t *testing.T) {
	svc := NewAccountHealthService(nil, nil)
	_, err := svc.UpdateImportWatcherSettings(context.Background(), &AccountImportWatcherSettings{Enabled: true, OutputDir: filepath.Join(t.TempDir(), "missing")})
	require.Error(t, err)
}

func TestScanImportWatcherMarksFilesNewAfterLastSeen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "accounts_demo.txt")
	require.NoError(t, os.WriteFile(path, []byte("token"), 0600))
	mod := time.Now().Add(-time.Hour).UTC()
	require.NoError(t, os.Chtimes(path, mod, mod))
	repo := &importWatcherRepo{}
	svc := NewAccountHealthService(nil, nil)
	svc.SetSettingsRepository(repo)
	_, err := svc.UpdateImportWatcherSettings(context.Background(), &AccountImportWatcherSettings{Enabled: true, OutputDir: dir, LastSeen: time.Now().UTC().Format(time.RFC3339Nano)})
	require.NoError(t, err)
	result, err := svc.ScanImportWatcher(context.Background())
	require.NoError(t, err)
	require.False(t, result.Files[0].New)
}
