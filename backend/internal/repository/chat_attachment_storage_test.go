package repository

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestLocalChatAttachmentStorePromotesPendingAndSweepsOrphans(t *testing.T) {
	root := t.TempDir()
	store, err := NewChatAttachmentBlobStore(&config.Config{ChatAttachments: config.ChatAttachmentConfig{StorageDir: root}})
	require.NoError(t, err)
	ctx := context.Background()
	require.NoError(t, store.PutPending(ctx, "7/att_12345678.png", []byte("image")))
	require.NoError(t, store.PromotePending(ctx, "7/att_12345678.png"))
	data, err := store.Get(ctx, "7/att_12345678.png")
	require.NoError(t, err)
	require.Equal(t, []byte("image"), data)

	path := filepath.Join(root, "7", "att_12345678.png")
	old := time.Now().Add(-2 * time.Hour)
	require.NoError(t, os.Chtimes(path, old, old))
	require.NoError(t, store.CleanupOrphans(ctx, map[string]struct{}{}, time.Now().Add(-time.Hour)))
	_, err = os.Stat(path)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestLocalChatAttachmentStoreSweepsPendingWithoutDatabaseRow(t *testing.T) {
	root := t.TempDir()
	store, err := NewChatAttachmentBlobStore(&config.Config{ChatAttachments: config.ChatAttachmentConfig{StorageDir: root}})
	require.NoError(t, err)
	ctx := context.Background()
	require.NoError(t, store.PutPending(ctx, "7/att_pending.png", []byte("image")))
	pendingPath := filepath.Join(root, ".pending", "7", "att_pending.png")
	old := time.Now().Add(-2 * time.Hour)
	require.NoError(t, os.Chtimes(pendingPath, old, old))
	require.NoError(t, store.CleanupPending(ctx, time.Now().Add(-time.Hour)))
	_, err = os.Stat(pendingPath)
	require.ErrorIs(t, err, os.ErrNotExist)
}
