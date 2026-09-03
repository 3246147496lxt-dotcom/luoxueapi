package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type localChatAttachmentBlobStore struct {
	root string
}

func NewChatAttachmentBlobStore(cfg *config.Config) (service.ChatAttachmentBlobStore, error) {
	root := "./data/chat-attachments"
	if cfg != nil && strings.TrimSpace(cfg.ChatAttachments.StorageDir) != "" {
		root = cfg.ChatAttachments.StorageDir
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err = os.MkdirAll(absolute, 0o700); err != nil {
		return nil, err
	}
	if err = os.Chmod(absolute, 0o700); err != nil {
		return nil, err
	}
	return &localChatAttachmentBlobStore{root: absolute}, nil
}

func (s *localChatAttachmentBlobStore) path(key string) (string, error) {
	if s == nil || s.root == "" || key == "" || filepath.IsAbs(key) {
		return "", errors.New("invalid chat attachment storage key")
	}
	clean := filepath.Clean(filepath.FromSlash(key))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("invalid chat attachment storage key")
	}
	path := filepath.Join(s.root, clean)
	rel, err := filepath.Rel(s.root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("chat attachment storage key escapes root")
	}
	return path, nil
}

func (s *localChatAttachmentBlobStore) pendingPath(key string) (string, error) {
	path, err := s.path(key)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(s.root, path)
	if err != nil {
		return "", err
	}
	return filepath.Join(s.root, ".pending", rel), nil
}

func writePrivateFileAtomically(ctx context.Context, path string, data []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".chat-attachment-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if err = tmp.Chmod(0o600); err == nil {
		_, err = tmp.Write(data)
	}
	if err == nil {
		err = tmp.Sync()
	}
	closeErr := tmp.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func (s *localChatAttachmentBlobStore) PutPending(ctx context.Context, key string, data []byte) error {
	path, err := s.pendingPath(key)
	if err != nil {
		return err
	}
	return writePrivateFileAtomically(ctx, path, data)
}

func (s *localChatAttachmentBlobStore) PromotePending(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	pending, err := s.pendingPath(key)
	if err != nil {
		return err
	}
	final, err := s.path(key)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(final), 0o700); err != nil {
		return err
	}
	return os.Rename(pending, final)
}

func (s *localChatAttachmentBlobStore) DeletePending(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := s.pendingPath(key)
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (s *localChatAttachmentBlobStore) CleanupPending(ctx context.Context, olderThan time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	root := filepath.Join(s.root, ".pending")
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			if os.IsNotExist(walkErr) {
				return nil
			}
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.ModTime().After(olderThan) {
			return nil
		}
		return os.Remove(path)
	})
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (s *localChatAttachmentBlobStore) CleanupOrphans(ctx context.Context, referenced map[string]struct{}, olderThan time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return filepath.WalkDir(s.root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			if path == filepath.Join(s.root, ".pending") {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.ModTime().After(olderThan) {
			return nil
		}
		rel, err := filepath.Rel(s.root, path)
		if err != nil {
			return err
		}
		key := filepath.ToSlash(rel)
		if _, ok := referenced[key]; ok {
			return nil
		}
		return os.Remove(path)
	})
}

func (s *localChatAttachmentBlobStore) Get(ctx context.Context, key string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	path, err := s.path(key)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("chat attachment blob is unavailable: %w", err)
	}
	return data, err
}

func (s *localChatAttachmentBlobStore) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := s.path(key)
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
