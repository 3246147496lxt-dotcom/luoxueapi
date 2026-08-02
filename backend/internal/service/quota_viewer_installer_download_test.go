//go:build unit

package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type fakeQuotaViewerInstallerTicketRecord struct {
	payload []byte
	ttl     time.Duration
}

type fakeQuotaViewerInstallerTicketStore struct {
	mu         sync.Mutex
	records    map[string]fakeQuotaViewerInstallerTicketRecord
	putErr     error
	consumeErr error
	lastDigest string
	putCalls   int
}

func newFakeQuotaViewerInstallerTicketStore() *fakeQuotaViewerInstallerTicketStore {
	return &fakeQuotaViewerInstallerTicketStore{records: make(map[string]fakeQuotaViewerInstallerTicketRecord)}
}

func (s *fakeQuotaViewerInstallerTicketStore) Put(
	_ context.Context,
	digest string,
	payload []byte,
	ttl time.Duration,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.putCalls++
	if s.putErr != nil {
		return s.putErr
	}
	s.lastDigest = digest
	s.records[digest] = fakeQuotaViewerInstallerTicketRecord{
		payload: append([]byte(nil), payload...),
		ttl:     ttl,
	}
	return nil
}

func (s *fakeQuotaViewerInstallerTicketStore) Consume(
	_ context.Context,
	digest string,
) ([]byte, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.consumeErr != nil {
		return nil, false, s.consumeErr
	}
	record, found := s.records[digest]
	if !found {
		return nil, false, nil
	}
	delete(s.records, digest)
	return append([]byte(nil), record.payload...), true, nil
}

func (s *fakeQuotaViewerInstallerTicketStore) observation() (string, []byte, time.Duration, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.records[s.lastDigest]
	return s.lastDigest, append([]byte(nil), record.payload...), record.ttl, ok
}

func newQuotaViewerInstallerDownloadTestService(
	t *testing.T,
) (*QuotaViewerInstallerDownloadService, *fakeQuotaViewerInstallerTicketStore, *time.Time) {
	t.Helper()
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	store := newFakeQuotaViewerInstallerTicketStore()
	svc := NewQuotaViewerInstallerDownloadService(store)
	svc.now = func() time.Time { return now }
	svc.random = bytes.NewReader(bytes.Repeat([]byte{0x42}, 32))
	return svc, store, &now
}

func quotaViewerInstallerCodeFromPath(t *testing.T, downloadPath string) string {
	t.Helper()
	parsed, err := url.Parse(downloadPath)
	require.NoError(t, err)
	return parsed.Query().Get("code")
}

func TestQuotaViewerInstallerIssueStoresOnlyDigestWithSixtySecondTTL(t *testing.T) {
	svc, store, _ := newQuotaViewerInstallerDownloadTestService(t)

	download, err := svc.Issue(context.Background(), 42, "macos")
	require.NoError(t, err)
	require.Equal(t, "/api/v1/quota/releases/macos/latest/download", strings.Split(download.DownloadPath, "?")[0])
	require.Equal(t, 60, download.ExpiresIn)
	require.Equal(t, "macos", download.Platform)
	require.Equal(t, "2.0.0-rc.6", download.Version)
	require.Equal(t, "Luoxue-Quota-Viewer_2.0.0-rc.6_macOS-universal-UNNOTARIZED.dmg", download.Filename)
	require.Equal(t, int64(10655545), download.Size)
	require.Equal(t, "63c7e3890ef45e882a99e609c73f3fa5e8452cc708bc4e1ed6f0278bdc519f92", download.SHA256)

	code := quotaViewerInstallerCodeFromPath(t, download.DownloadPath)
	decoded, err := base64.RawURLEncoding.DecodeString(code)
	require.NoError(t, err)
	require.Len(t, decoded, 32)

	digest, payload, ttl, found := store.observation()
	require.True(t, found)
	require.Equal(t, quotaViewerInstallerDownloadDigest(code), digest)
	require.Len(t, digest, 64)
	require.NotContains(t, digest, code)
	require.NotContains(t, string(payload), code)
	require.Equal(t, QuotaViewerInstallerDownloadTTL, ttl)

	var ticket quotaViewerInstallerTicket
	require.NoError(t, json.Unmarshal(payload, &ticket))
	require.Equal(t, int64(42), ticket.UserID)
	require.Equal(t, "macos", ticket.Platform)
	require.Equal(t, download.SHA256, ticket.AssetSHA256)
}

func TestQuotaViewerInstallerPlatformAllowlist(t *testing.T) {
	svc, store, _ := newQuotaViewerInstallerDownloadTestService(t)

	windows, err := svc.Issue(context.Background(), 42, "windows")
	require.NoError(t, err)
	require.Equal(t, "2.0.0-rc.5", windows.Version)
	require.Equal(t, "Luoxue-Quota-Viewer_2.0.0-rc.5_windows-x64_NSIS-UNSIGNED.exe", windows.Filename)
	require.Equal(t, int64(5110179), windows.Size)
	require.Equal(t, "2f1a0b37f3720d02564d306bc5393e71edfbc2ea5d788b89f87feb9a2b4a27d1", windows.SHA256)

	for _, platform := range []string{"", "darwin", "MacOS", "linux", "../macos", " macos"} {
		_, err := svc.Issue(context.Background(), 42, platform)
		require.Equal(t, "UNSUPPORTED_QUOTA_VIEWER_INSTALLER_PLATFORM", infraerrors.Reason(err), platform)
	}
	require.Equal(t, 1, store.putCalls)
}

func TestQuotaViewerInstallerConsumeSucceedsOnceAndRejectsReplay(t *testing.T) {
	svc, _, _ := newQuotaViewerInstallerDownloadTestService(t)
	download, err := svc.Issue(context.Background(), 7, "macos")
	require.NoError(t, err)
	code := quotaViewerInstallerCodeFromPath(t, download.DownloadPath)

	asset, err := svc.Consume(context.Background(), "macos", code)
	require.NoError(t, err)
	require.Equal(t, quotaViewerInstallerAssets["macos"], *asset)

	_, err = svc.Consume(context.Background(), "macos", code)
	require.Equal(t, "INVALID_QUOTA_VIEWER_INSTALLER_DOWNLOAD_CODE", infraerrors.Reason(err))
}

func TestQuotaViewerInstallerExpiredAndPlatformMismatchedTicketsAreConsumed(t *testing.T) {
	t.Run("expired", func(t *testing.T) {
		svc, _, now := newQuotaViewerInstallerDownloadTestService(t)
		download, err := svc.Issue(context.Background(), 7, "macos")
		require.NoError(t, err)
		code := quotaViewerInstallerCodeFromPath(t, download.DownloadPath)
		*now = now.Add(QuotaViewerInstallerDownloadTTL + time.Second)

		_, err = svc.Consume(context.Background(), "macos", code)
		require.Equal(t, "INVALID_QUOTA_VIEWER_INSTALLER_DOWNLOAD_CODE", infraerrors.Reason(err))
		_, err = svc.Consume(context.Background(), "macos", code)
		require.Equal(t, "INVALID_QUOTA_VIEWER_INSTALLER_DOWNLOAD_CODE", infraerrors.Reason(err))
	})

	t.Run("platform mismatch", func(t *testing.T) {
		svc, _, _ := newQuotaViewerInstallerDownloadTestService(t)
		download, err := svc.Issue(context.Background(), 7, "macos")
		require.NoError(t, err)
		code := quotaViewerInstallerCodeFromPath(t, download.DownloadPath)

		_, err = svc.Consume(context.Background(), "windows", code)
		require.Equal(t, "INVALID_QUOTA_VIEWER_INSTALLER_DOWNLOAD_CODE", infraerrors.Reason(err))
		_, err = svc.Consume(context.Background(), "macos", code)
		require.Equal(t, "INVALID_QUOTA_VIEWER_INSTALLER_DOWNLOAD_CODE", infraerrors.Reason(err))
	})
}

func TestQuotaViewerInstallerRedisFailuresFailClosed(t *testing.T) {
	svc, store, _ := newQuotaViewerInstallerDownloadTestService(t)
	store.putErr = errors.New("redis unavailable")

	_, err := svc.Issue(context.Background(), 7, "macos")
	require.Equal(t, 503, infraerrors.Code(err))
	require.Equal(t, "QUOTA_VIEWER_INSTALLER_DOWNLOAD_UNAVAILABLE", infraerrors.Reason(err))

	store.putErr = nil
	store.consumeErr = errors.New("redis unavailable")
	validCode := base64.RawURLEncoding.EncodeToString(make([]byte, 32))
	_, err = svc.Consume(context.Background(), "macos", validCode)
	require.Equal(t, 503, infraerrors.Code(err))
	require.Equal(t, "QUOTA_VIEWER_INSTALLER_DOWNLOAD_UNAVAILABLE", infraerrors.Reason(err))
}

func TestQuotaViewerInstallerMissingStoreOrAdmissionFailsClosed(t *testing.T) {
	svc := NewQuotaViewerInstallerDownloadService(nil)
	_, err := svc.Issue(context.Background(), 7, "macos")
	require.Equal(t, 503, infraerrors.Code(err))

	store := newFakeQuotaViewerInstallerTicketStore()
	svc = NewQuotaViewerInstallerDownloadService(store)
	svc.downloadAdmission = nil
	validCode := base64.RawURLEncoding.EncodeToString(make([]byte, 32))
	_, err = svc.Consume(context.Background(), "macos", validCode)
	require.Equal(t, 503, infraerrors.Code(err))
}

func TestQuotaViewerInstallerAssetURLAllowlist(t *testing.T) {
	asset := quotaViewerInstallerAssets["macos"]
	require.NoError(t, validateQuotaViewerInstallerAsset(asset))

	for name, rawURL := range map[string]string{
		"http":             "http://github.com/3246147496lxt-dotcom/luoxueapi/releases/download/tag/file.dmg",
		"lookalike host":   "https://github.com.evil.example/3246147496lxt-dotcom/luoxueapi/releases/download/tag/file.dmg",
		"wrong repository": "https://github.com/other/luoxueapi/releases/download/tag/file.dmg",
		"dot segment":      "https://github.com/3246147496lxt-dotcom/luoxueapi/releases/download/../" + asset.Filename,
		"query":            asset.URL + "?download=1",
		"userinfo":         "https://user@github.com/3246147496lxt-dotcom/luoxueapi/releases/download/tag/file.dmg",
	} {
		t.Run(name, func(t *testing.T) {
			candidate := asset
			candidate.URL = rawURL
			require.Error(t, validateQuotaViewerInstallerAsset(candidate))
		})
	}
}
