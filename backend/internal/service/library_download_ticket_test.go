//go:build unit

package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/url"
	"path"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type fakeLibraryDownloadTicketStore struct {
	mu               sync.Mutex
	records          map[string]fakeLibraryDownloadTicketRecord
	lastHandle       string
	lastSecretDigest string
	lastTTL          time.Duration
	putErr           error
	consumeErr       error
	allowIssue       bool
	allowIssueErr    error
}

type fakeLibraryDownloadTicketRecord struct {
	secretDigest string
	payload      []byte
}

func newFakeLibraryDownloadTicketStore() *fakeLibraryDownloadTicketStore {
	return &fakeLibraryDownloadTicketStore{
		records: make(map[string]fakeLibraryDownloadTicketRecord), allowIssue: true,
	}
}

func (s *fakeLibraryDownloadTicketStore) Put(
	_ context.Context,
	handle string,
	secretDigest string,
	payload []byte,
	ttl time.Duration,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.putErr != nil {
		return s.putErr
	}
	s.lastHandle = handle
	s.lastSecretDigest = secretDigest
	s.lastTTL = ttl
	s.records[handle] = fakeLibraryDownloadTicketRecord{
		secretDigest: secretDigest, payload: append([]byte(nil), payload...),
	}
	return nil
}

func (s *fakeLibraryDownloadTicketStore) Consume(
	_ context.Context,
	handle string,
	secretDigest string,
) ([]byte, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.consumeErr != nil {
		return nil, false, s.consumeErr
	}
	record, found := s.records[handle]
	if !found || record.secretDigest != secretDigest {
		return nil, false, nil
	}
	delete(s.records, handle)
	return append([]byte(nil), record.payload...), true, nil
}

func (s *fakeLibraryDownloadTicketStore) AllowIssue(
	_ context.Context,
	_ int64,
	_ int64,
	_ time.Duration,
) (bool, error) {
	if s.allowIssueErr != nil {
		return false, s.allowIssueErr
	}
	return s.allowIssue, nil
}

func newLibraryDownloadTicketTestService(t *testing.T) (
	*LibraryDownloadTicketService,
	*libraryRepoFake,
	*fakeLibraryDownloadTicketStore,
	*time.Time,
) {
	t.Helper()
	now := time.Date(2026, 8, 14, 8, 0, 0, 0, time.UTC)
	fileID := "file_0123456789abcdef0123456789abcdef"
	repo := &libraryRepoFake{files: map[string]LibraryFile{
		fileID: {
			ID: fileID, UserID: 7, Name: "设计稿.pdf", MIMEType: "application/pdf",
			Size: 12, StoredSize: 12, Status: LibraryFileStatusReady,
		},
	}}
	library := NewLibraryService(repo, &libraryBlobStoreFake{}, &config.Config{Library: config.LibraryConfig{
		BatchDownloadLimit: 100, BatchDownloadMaxBytes: 100 << 20,
	}})
	store := newFakeLibraryDownloadTicketStore()
	svc := NewLibraryDownloadTicketService(library, store)
	svc.now = func() time.Time { return now }
	svc.random = bytes.NewReader(bytes.Repeat(
		[]byte{0x42}, libraryDownloadTicketHandleBytes+libraryDownloadTicketSecretBytes,
	))
	return svc, repo, store, &now
}

func libraryDownloadIDFromPath(t *testing.T, downloadPath string) string {
	t.Helper()
	parsed, err := url.Parse(downloadPath)
	require.NoError(t, err)
	require.Empty(t, parsed.RawQuery)
	require.Equal(t, "/api/v1/library/download", path.Dir(parsed.Path))
	return path.Base(parsed.Path)
}

func TestLibraryDownloadTicketIsHashedSingleUseAndReauthorizes(t *testing.T) {
	svc, repo, store, _ := newLibraryDownloadTicketTestService(t)
	fileID := "file_0123456789abcdef0123456789abcdef"

	ticket, err := svc.Issue(context.Background(), 7, []string{fileID})
	require.NoError(t, err)
	require.Equal(t, "设计稿.pdf", ticket.Filename)
	require.Equal(t, 1, ticket.FileCount)
	require.EqualValues(t, 12, ticket.TotalBytes)
	require.Equal(t, int(LibraryDownloadTicketTTL/time.Second), ticket.ExpiresIn)

	handle := libraryDownloadIDFromPath(t, ticket.DownloadPath)
	require.Len(t, handle, 22)
	require.True(t, ValidLibraryDownloadTicketID(handle))
	require.Len(t, ticket.Secret, 43)
	require.Equal(t, handle, ticket.TicketID)
	require.Equal(t, handle, store.lastHandle)
	require.Len(t, store.lastSecretDigest, 64)
	require.NotContains(t, store.lastSecretDigest, ticket.Secret)
	require.Equal(t, LibraryDownloadTicketTTL, store.lastTTL)
	payload := store.records[store.lastHandle].payload
	require.NotContains(t, string(payload), ticket.Secret)
	var persisted libraryDownloadTicketRecord
	require.NoError(t, json.Unmarshal(payload, &persisted))
	require.Equal(t, []string{fileID}, persisted.FileIDs)
	serialized, err := json.Marshal(ticket)
	require.NoError(t, err)
	require.NotContains(t, string(serialized), ticket.Secret)
	require.NotContains(t, string(serialized), "TicketID")

	grant, err := svc.Consume(context.Background(), handle, ticket.Secret)
	require.NoError(t, err)
	require.Equal(t, int64(7), grant.UserID)
	require.Equal(t, []string{fileID}, []string{grant.Files[0].ID})

	_, err = svc.Consume(context.Background(), handle, ticket.Secret)
	require.ErrorIs(t, err, ErrLibraryDownloadTicketInvalid)

	svc.random = bytes.NewReader(bytes.Repeat(
		[]byte{0x24}, libraryDownloadTicketHandleBytes+libraryDownloadTicketSecretBytes,
	))
	second, err := svc.Issue(context.Background(), 7, []string{fileID})
	require.NoError(t, err)
	repo.files[fileID] = LibraryFile{ID: fileID, Status: LibraryFileStatusDeleted}
	_, err = svc.Consume(context.Background(), libraryDownloadIDFromPath(t, second.DownloadPath), second.Secret)
	require.ErrorIs(t, err, ErrLibraryDownloadTicketInvalid)
}

func TestLibraryDownloadTicketWrongSecretCannotBurnTicket(t *testing.T) {
	svc, _, store, _ := newLibraryDownloadTicketTestService(t)
	fileID := "file_0123456789abcdef0123456789abcdef"
	ticket, err := svc.Issue(context.Background(), 7, []string{fileID})
	require.NoError(t, err)
	handle := libraryDownloadIDFromPath(t, ticket.DownloadPath)
	wrongSecret := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0x99}, libraryDownloadTicketSecretBytes))

	_, err = svc.Consume(context.Background(), handle, wrongSecret)
	require.ErrorIs(t, err, ErrLibraryDownloadTicketInvalid)
	require.Contains(t, store.records, handle, "wrong cookie secret must not delete the ticket")

	grant, err := svc.Consume(context.Background(), handle, ticket.Secret)
	require.NoError(t, err)
	require.Equal(t, fileID, grant.Files[0].ID)
}

func TestLibraryDownloadTicketIssueRateLimitFailsClosed(t *testing.T) {
	svc, _, store, _ := newLibraryDownloadTicketTestService(t)
	fileID := "file_0123456789abcdef0123456789abcdef"

	store.allowIssue = false
	_, err := svc.Issue(context.Background(), 7, []string{fileID})
	require.ErrorIs(t, err, ErrLibraryDownloadTicketRateLimited)

	store.allowIssueErr = errors.New("redis unavailable")
	_, err = svc.Issue(context.Background(), 7, []string{fileID})
	require.ErrorIs(t, err, ErrLibraryDownloadTicketUnavailable)
}

func TestValidLibraryDownloadTicketIDIsStrict(t *testing.T) {
	valid := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0x42}, libraryDownloadTicketHandleBytes))
	require.True(t, ValidLibraryDownloadTicketID(valid))
	require.False(t, ValidLibraryDownloadTicketID(" "+valid))
	require.False(t, ValidLibraryDownloadTicketID(valid+"="))
	require.False(t, ValidLibraryDownloadTicketID(valid+"extra"))
}

func TestLibraryDownloadTicketRejectsExpiredMalformedAndUnavailableState(t *testing.T) {
	svc, _, store, now := newLibraryDownloadTicketTestService(t)
	fileID := "file_0123456789abcdef0123456789abcdef"
	ticket, err := svc.Issue(context.Background(), 7, []string{fileID})
	require.NoError(t, err)
	handle := libraryDownloadIDFromPath(t, ticket.DownloadPath)
	*now = (*now).Add(LibraryDownloadTicketTTL + time.Second)
	_, err = svc.Consume(context.Background(), handle, ticket.Secret)
	require.ErrorIs(t, err, ErrLibraryDownloadTicketInvalid)

	_, err = svc.Consume(context.Background(), "not-a-ticket", ticket.Secret)
	require.ErrorIs(t, err, ErrLibraryDownloadTicketInvalid)

	store.putErr = errors.New("redis unavailable")
	svc.random = bytes.NewReader(bytes.Repeat(
		[]byte{0x24}, libraryDownloadTicketHandleBytes+libraryDownloadTicketSecretBytes,
	))
	_, err = svc.Issue(context.Background(), 7, []string{fileID})
	require.ErrorIs(t, err, ErrLibraryDownloadTicketUnavailable)
	require.NotContains(t, strings.ToLower(ErrLibraryDownloadTicketUnavailable.Message), "redis unavailable")
}
