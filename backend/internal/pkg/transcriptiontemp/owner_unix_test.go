//go:build linux || darwin

package transcriptiontemp

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestVerifyWorkspaceOwnershipRejectsAnotherUser(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, directoryName)
	require.NoError(t, os.Mkdir(dir, 0o755))
	info, err := os.Lstat(dir)
	require.NoError(t, err)
	stat, ok := info.Sys().(*syscall.Stat_t)
	require.True(t, ok)

	err = verifyWorkspaceOwnershipForUID(info, stat.Uid+1)
	require.ErrorContains(t, err, "not owned by the service user")

	// Ownership is checked before chmod so a privileged process never mutates
	// a foreign directory and then treats it as its private workspace.
	info, err = os.Stat(dir)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o755), info.Mode().Perm())
}

func TestWorkspacePreparerRejectsUnsafeCachedPathReplacement(t *testing.T) {
	base := t.TempDir()
	preparer := &workspacePreparer{}
	dir, err := preparer.Prepare(base, time.Now())
	require.NoError(t, err)
	require.NoError(t, os.Remove(dir))
	require.NoError(t, os.Symlink(t.TempDir(), dir))

	_, err = preparer.Prepare(base, time.Now())
	require.ErrorContains(t, err, "not a private directory")
}
