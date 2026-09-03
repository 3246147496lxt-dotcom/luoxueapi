package transcriptiontemp

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPrepareUsesPrivateDirectoryAndRemovesOnlyOwnedStaleFiles(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, directoryName)
	require.NoError(t, os.Mkdir(dir, 0o755))

	staleAudio := filepath.Join(dir, "audio-stale")
	staleUpstream := filepath.Join(dir, "upstream-stale")
	freshAudio := filepath.Join(dir, "audio-fresh")
	unrelated := filepath.Join(dir, "keep-me")
	for _, path := range []string{staleAudio, staleUpstream, freshAudio, unrelated} {
		require.NoError(t, os.WriteFile(path, []byte("test"), 0o600))
	}
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	staleTime := now.Add(-staleAfter - time.Minute)
	require.NoError(t, os.Chtimes(staleAudio, staleTime, staleTime))
	require.NoError(t, os.Chtimes(staleUpstream, staleTime, staleTime))
	require.NoError(t, os.Chtimes(unrelated, staleTime, staleTime))
	require.NoError(t, os.Chtimes(freshAudio, now, now))

	prepared, err := prepare(base, now)
	require.NoError(t, err)
	require.Equal(t, dir, prepared)
	info, err := os.Stat(dir)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o700), info.Mode().Perm())
	require.NoFileExists(t, staleAudio)
	require.NoFileExists(t, staleUpstream)
	require.FileExists(t, freshAudio)
	require.FileExists(t, unrelated)

	// The janitor uses the same sweep primitive, so a crash file that was fresh
	// at restart is removed once it crosses the retention boundary.
	require.NoError(t, removeStaleFiles(dir, now.Add(time.Minute)))
	require.NoFileExists(t, freshAudio)
	require.FileExists(t, unrelated)
}

func TestPrepareRejectsSymlinkWorkspace(t *testing.T) {
	base := t.TempDir()
	target := t.TempDir()
	require.NoError(t, os.Symlink(target, filepath.Join(base, directoryName)))

	_, err := prepare(base, time.Now())
	require.ErrorContains(t, err, "not a private directory")
}

func TestWorkspacePreparerRetriesAfterTransientInitializationFailure(t *testing.T) {
	base := t.TempDir()
	blockedPath := filepath.Join(base, directoryName)
	require.NoError(t, os.WriteFile(blockedPath, []byte("blocked"), 0o600))
	preparer := &workspacePreparer{}

	_, err := preparer.Prepare(base, time.Now())
	require.Error(t, err)
	require.NoError(t, os.Remove(blockedPath))

	dir, err := preparer.Prepare(base, time.Now())
	require.NoError(t, err)
	require.Equal(t, blockedPath, dir)
}

func TestWorkspacePreparerRevalidatesAndRecreatesCachedDirectory(t *testing.T) {
	base := t.TempDir()
	preparer := &workspacePreparer{}
	dir, err := preparer.Prepare(base, time.Now())
	require.NoError(t, err)
	require.NoError(t, os.Remove(dir))

	recreated, err := preparer.Prepare(base, time.Now())
	require.NoError(t, err)
	require.Equal(t, dir, recreated)
	info, err := os.Stat(recreated)
	require.NoError(t, err)
	require.True(t, info.IsDir())
	require.Equal(t, os.FileMode(0o700), info.Mode().Perm())
}
