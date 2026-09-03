package handler

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestUniqueLibraryZipNameNeverProducesDuplicateEntries(t *testing.T) {
	used := map[string]int{}
	require.Equal(t, "a.txt", uniqueLibraryZipName("a.txt", used))
	require.Equal(t, "a (2).txt", uniqueLibraryZipName("a.txt", used))
	require.Equal(t, "a (2) (2).txt", uniqueLibraryZipName("a (2).txt", used))
	require.Equal(t, "a (3).txt", uniqueLibraryZipName("a.txt", used))
}

func TestLibraryBatchDownloadStagesVerifiedZIPWithUniqueUnicodeNames(t *testing.T) {
	first := []byte("first report")
	second := []byte("second report")
	files := map[string]service.LibraryFile{
		"file_alpha": newHandlerLibraryFile("file_alpha", "报告.txt", "library/7/alpha", first),
		"file_beta":  newHandlerLibraryFile("file_beta", "报告.txt", "library/7/beta", second),
	}
	store := &handlerLibraryStore{objects: map[string][]byte{
		"library/7/alpha": first,
		"library/7/beta":  second,
	}}
	handler, stagedPaths := newHandlerLibraryDownloadFixture(t, files, store)

	recorder := performLibraryBatchDownload(t, handler, `{"file_ids":["file_alpha","file_beta"]}`, nil)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "application/zip", recorder.Header().Get("Content-Type"))
	require.Equal(t, recorder.Body.Len(), mustPositiveInt(t, recorder.Header().Get("Content-Length")))
	archive, err := zip.NewReader(bytes.NewReader(recorder.Body.Bytes()), int64(recorder.Body.Len()))
	require.NoError(t, err)
	require.Len(t, archive.File, 2)
	require.Equal(t, "报告.txt", archive.File[0].Name)
	require.Equal(t, "报告 (2).txt", archive.File[1].Name)
	require.Equal(t, first, readZIPEntry(t, archive.File[0]))
	require.Equal(t, second, readZIPEntry(t, archive.File[1]))
	require.Equal(t, 2, store.closed)
	requireRemovedDownloadTemps(t, *stagedPaths)
}

func TestLibraryBatchDownloadMissingObjectDoesNotCommitPartialSuccess(t *testing.T) {
	first := []byte("available")
	missing := []byte("missing")
	files := map[string]service.LibraryFile{
		"file_alpha": newHandlerLibraryFile("file_alpha", "available.txt", "library/7/alpha", first),
		"file_beta":  newHandlerLibraryFile("file_beta", "missing.txt", "library/7/missing", missing),
	}
	store := &handlerLibraryStore{
		objects:    map[string][]byte{"library/7/alpha": first},
		openErrors: map[string]error{"library/7/missing": errors.New("object not found")},
	}
	handler, stagedPaths := newHandlerLibraryDownloadFixture(t, files, store)

	recorder := performLibraryBatchDownload(t, handler, `{"file_ids":["file_alpha","file_beta"]}`, nil)

	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	require.Contains(t, recorder.Body.String(), "LIBRARY_DOWNLOAD_FAILED")
	require.NotEqual(t, "application/zip", recorder.Header().Get("Content-Type"))
	require.Equal(t, 1, store.closed, "the successfully opened object must be closed before the later failure")
	requireRemovedDownloadTemps(t, *stagedPaths)
}

func TestLibrarySingleDownloadDigestFailureDoesNotCommit200(t *testing.T) {
	expected := []byte("good")
	files := map[string]service.LibraryFile{
		"file_alpha": newHandlerLibraryFile("file_alpha", "notes.txt", "library/7/alpha", expected),
	}
	store := &handlerLibraryStore{objects: map[string][]byte{"library/7/alpha": []byte("evil")}}
	handler, stagedPaths := newHandlerLibraryDownloadFixture(t, files, store)

	recorder := performLibraryBatchDownload(t, handler, `{"file_ids":["file_alpha"]}`, nil)

	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	require.Contains(t, recorder.Body.String(), "LIBRARY_DOWNLOAD_FAILED")
	require.NotContains(t, recorder.Body.String(), "evil")
	require.Equal(t, 1, store.closed)
	requireRemovedDownloadTemps(t, *stagedPaths)
}

func TestLibraryDownloadStagingUses0600AndExactCleanup(t *testing.T) {
	data := []byte("private")
	files := map[string]service.LibraryFile{
		"file_alpha": newHandlerLibraryFile("file_alpha", "private.txt", "library/7/alpha", data),
	}
	store := &handlerLibraryStore{objects: map[string][]byte{"library/7/alpha": data}}
	handler, stagedPaths := newHandlerLibraryDownloadFixture(t, files, store)

	staged, err := handler.stageResolvedDownload(context.Background(), 7, []service.LibraryFile{files["file_alpha"]})
	require.NoError(t, err)
	require.Len(t, *stagedPaths, 1)
	info, err := os.Stat(staged.path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	require.NoError(t, staged.closeAndRemove())
	requireRemovedDownloadTemps(t, *stagedPaths)
}

func TestLibraryBatchDownloadCancellationCleansStagingFile(t *testing.T) {
	data := []byte("private")
	files := map[string]service.LibraryFile{
		"file_alpha": newHandlerLibraryFile("file_alpha", "private.txt", "library/7/alpha", data),
	}
	store := &handlerLibraryStore{objects: map[string][]byte{"library/7/alpha": data}}
	handler, stagedPaths := newHandlerLibraryDownloadFixture(t, files, store)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	recorder := performLibraryBatchDownload(t, handler, `{"file_ids":["file_alpha"]}`, cancelled)

	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	require.Contains(t, recorder.Body.String(), "LIBRARY_DOWNLOAD_FAILED")
	requireRemovedDownloadTemps(t, *stagedPaths)
}

func TestLibraryIssueDownloadTicketRejectsNonStrictJSON(t *testing.T) {
	data := []byte("private")
	files := map[string]service.LibraryFile{
		"file_alpha": newHandlerLibraryFile("file_alpha", "private.txt", "library/7/alpha", data),
	}
	handler, _ := newHandlerLibraryDownloadFixture(t, files, &handlerLibraryStore{
		objects: map[string][]byte{"library/7/alpha": data},
	})
	ticketStore := newHandlerLibraryTicketStore()
	handler.downloadTickets = service.NewLibraryDownloadTicketService(handler.library, ticketStore)

	tests := map[string]string{
		"unknown field":      `{"file_ids":["file_alpha"],"persist":true}`,
		"trailing JSON":      `{"file_ids":["file_alpha"]}{"file_ids":["file_alpha"]}`,
		"both naming styles": `{"file_ids":["file_alpha"],"fileIds":["file_alpha"]}`,
		"malformed":          `{"file_ids":["file_alpha"]`,
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			recorder := performLibraryDownloadTicketIssue(t, handler, body)

			require.Equal(t, http.StatusBadRequest, recorder.Code)
			assertLibraryErrorEnvelope(t, recorder, "LIBRARY_INVALID_REQUEST")
		})
	}
	require.Zero(t, ticketStore.putCalls, "invalid JSON must not mint a download credential")
}

func TestLibraryIssueDownloadTicketWithoutServiceFailsClosed(t *testing.T) {
	data := []byte("private")
	files := map[string]service.LibraryFile{
		"file_alpha": newHandlerLibraryFile("file_alpha", "private.txt", "library/7/alpha", data),
	}
	handler, _ := newHandlerLibraryDownloadFixture(t, files, &handlerLibraryStore{
		objects: map[string][]byte{"library/7/alpha": data},
	})

	recorder := performLibraryDownloadTicketIssue(t, handler, `{"file_ids":["file_alpha"]}`)

	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	assertLibraryErrorEnvelope(t, recorder, "LIBRARY_DOWNLOAD_TICKET_UNAVAILABLE")
}

func TestLibraryDownloadTicketStagesBeforeSuccessAndRejectsReplay(t *testing.T) {
	data := []byte("private download body")
	files := map[string]service.LibraryFile{
		"file_alpha": newHandlerLibraryFile("file_alpha", "private notes.txt", "library/7/alpha", data),
	}
	store := &handlerLibraryStore{objects: map[string][]byte{"library/7/alpha": data}}
	handler, stagedPaths := newHandlerLibraryDownloadFixture(t, files, store)
	ticketStore := newHandlerLibraryTicketStore()
	handler.downloadTickets = service.NewLibraryDownloadTicketService(handler.library, ticketStore)

	issued := performLibraryDownloadTicketIssue(t, handler, `{"file_ids":["file_alpha"]}`)
	require.Equal(t, http.StatusCreated, issued.Code)
	credential := downloadTicketCredentialFromResponse(t, issued)
	require.NotContains(t, credential.ticketID, "Bearer")
	require.False(t, credential.cookie.Secure)
	require.Equal(t, http.SameSiteLaxMode, credential.cookie.SameSite)

	downloaded := performLibraryDownloadWithTicket(t, handler, credential, true)
	require.Equal(t, http.StatusOK, downloaded.Code)
	require.Equal(t, data, downloaded.Body.Bytes())
	require.Equal(t, "text/plain; charset=utf-8", downloaded.Header().Get("Content-Type"))
	require.Equal(t, fmt.Sprintf("%d", len(data)), downloaded.Header().Get("Content-Length"))
	require.Contains(t, downloaded.Header().Get("Content-Disposition"), "attachment")
	require.Contains(t, downloaded.Header().Get("Content-Disposition"), "private notes.txt")
	require.Equal(t, "private, no-store", downloaded.Header().Get("Cache-Control"))
	require.Equal(t, "no-referrer", downloaded.Header().Get("Referrer-Policy"))
	require.Equal(t, "SAMEORIGIN", downloaded.Header().Get("X-Frame-Options"))
	require.Equal(t, "default-src 'none'; frame-ancestors 'self'", downloaded.Header().Get("Content-Security-Policy"))
	cleared := downloaded.Result().Cookies()
	require.Len(t, cleared, 1)
	require.Equal(t, credential.cookie.Name, cleared[0].Name)
	require.Equal(t, credential.cookie.Path, cleared[0].Path)
	require.Equal(t, -1, cleared[0].MaxAge)
	require.Equal(t, 1, store.closed)
	require.Equal(t, 1, ticketStore.consumeCalls)
	requireRemovedDownloadTemps(t, *stagedPaths)

	replay := performLibraryDownloadWithTicket(t, handler, credential, true)
	require.Equal(t, http.StatusUnauthorized, replay.Code)
	assertLibraryErrorEnvelope(t, replay, "LIBRARY_DOWNLOAD_TICKET_INVALID")
	require.NotContains(t, replay.Body.String(), credential.ticketID)
	require.NotContains(t, replay.Body.String(), credential.secret)
	require.Equal(t, 2, ticketStore.consumeCalls)
	require.Equal(t, 1, store.closed, "a rejected replay must not reopen private storage")
	require.Len(t, *stagedPaths, 1, "a rejected replay must not create a staging file")
}

func TestLibraryDownloadTicketErrorResponseDoesNotLeakCause(t *testing.T) {
	data := []byte("private")
	files := map[string]service.LibraryFile{
		"file_alpha": newHandlerLibraryFile("file_alpha", "private.txt", "library/7/alpha", data),
	}
	handler, stagedPaths := newHandlerLibraryDownloadFixture(t, files, &handlerLibraryStore{
		objects: map[string][]byte{"library/7/alpha": data},
	})
	ticketStore := newHandlerLibraryTicketStore()
	ticketStore.consumeErr = errors.New("storage backend offline: internal-ticket-cause")
	handler.downloadTickets = service.NewLibraryDownloadTicketService(handler.library, ticketStore)
	credential := libraryDownloadTicketCredential{
		ticketID: base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0x5a}, 16)),
		secret:   base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0x6a}, 32)),
	}
	credential.cookie = &http.Cookie{
		Name: libraryDownloadCookieName(credential.ticketID), Value: credential.secret,
		Path: libraryDownloadCookiePath(credential.ticketID),
	}

	recorder := performLibraryDownloadWithTicket(t, handler, credential, true)

	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	assertLibraryErrorEnvelope(t, recorder, "LIBRARY_DOWNLOAD_TICKET_UNAVAILABLE")
	require.NotContains(t, recorder.Body.String(), "internal-ticket-cause")
	require.NotContains(t, recorder.Body.String(), credential.ticketID)
	require.NotContains(t, recorder.Body.String(), credential.secret)
	require.Empty(t, *stagedPaths)
}

func TestLibraryDownloadTicketHTTPSCookieIsSecureSameSiteNone(t *testing.T) {
	data := []byte("private")
	files := map[string]service.LibraryFile{
		"file_alpha": newHandlerLibraryFile("file_alpha", "private.txt", "library/7/alpha", data),
	}
	handler, _ := newHandlerLibraryDownloadFixture(t, files, &handlerLibraryStore{
		objects: map[string][]byte{"library/7/alpha": data},
	})
	handler.downloadTickets = service.NewLibraryDownloadTicketService(handler.library, newHandlerLibraryTicketStore())

	issued := performLibraryDownloadTicketIssueWithProto(
		t,
		handler,
		`{"file_ids":["file_alpha"]}`,
		"https",
	)
	require.Equal(t, http.StatusCreated, issued.Code)
	credential := downloadTicketCredentialFromResponse(t, issued)
	require.True(t, credential.cookie.Secure)
	require.Equal(t, http.SameSiteNoneMode, credential.cookie.SameSite)
}

func TestLibraryDownloadTicketRejectsMalformedPathBeforeCookieReflection(t *testing.T) {
	data := []byte("private")
	files := map[string]service.LibraryFile{
		"file_alpha": newHandlerLibraryFile("file_alpha", "private.txt", "library/7/alpha", data),
	}
	handler, _ := newHandlerLibraryDownloadFixture(t, files, &handlerLibraryStore{
		objects: map[string][]byte{"library/7/alpha": data},
	})
	ticketStore := newHandlerLibraryTicketStore()
	handler.downloadTickets = service.NewLibraryDownloadTicketService(handler.library, ticketStore)
	malicious := strings.Repeat("a", 4096) + "; Path=/"

	recorder := performLibraryDownloadWithTicket(t, handler, libraryDownloadTicketCredential{ticketID: malicious}, false)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	assertLibraryErrorEnvelope(t, recorder, "LIBRARY_DOWNLOAD_TICKET_INVALID")
	require.Empty(t, recorder.Header().Values("Set-Cookie"))
	require.Zero(t, ticketStore.consumeCalls)
}

type handlerLibraryRepo struct {
	service.LibraryFileRepository
	files map[string]service.LibraryFile
}

func (r *handlerLibraryRepo) GetOwned(_ context.Context, _ int64, id string) (*service.LibraryFile, error) {
	file, ok := r.files[id]
	if !ok || file.Status != service.LibraryFileStatusReady {
		return nil, service.ErrLibraryFileNotFound
	}
	return &file, nil
}

func (r *handlerLibraryRepo) ResolveOwned(_ context.Context, _ int64, ids []string) ([]service.LibraryFile, error) {
	result := make([]service.LibraryFile, 0, len(ids))
	for _, id := range ids {
		file, ok := r.files[id]
		if !ok || file.Status != service.LibraryFileStatusReady {
			return nil, service.ErrLibraryFileNotFound
		}
		result = append(result, file)
	}
	return result, nil
}

type handlerLibraryStore struct {
	objects    map[string][]byte
	openErrors map[string]error
	closed     int
}

func (s *handlerLibraryStore) Put(context.Context, string, io.Reader, int64, string) error {
	return errors.New("not implemented")
}

func (s *handlerLibraryStore) Open(_ context.Context, key string) (io.ReadCloser, error) {
	if err := s.openErrors[key]; err != nil {
		return nil, err
	}
	data, ok := s.objects[key]
	if !ok {
		return nil, errors.New("object not found")
	}
	return &handlerTrackingReadCloser{
		Reader: bytes.NewReader(data),
		onClose: func() {
			s.closed++
		},
	}, nil
}

func (*handlerLibraryStore) Delete(context.Context, string) error { return nil }

type handlerLibraryTicketStore struct {
	mu           sync.Mutex
	payloads     map[string]handlerLibraryTicketRecord
	putErr       error
	consumeErr   error
	putCalls     int
	consumeCalls int
}

type handlerLibraryTicketRecord struct {
	secretDigest string
	payload      []byte
}

func newHandlerLibraryTicketStore() *handlerLibraryTicketStore {
	return &handlerLibraryTicketStore{payloads: make(map[string]handlerLibraryTicketRecord)}
}

func (s *handlerLibraryTicketStore) Put(
	_ context.Context,
	handle string,
	secretDigest string,
	payload []byte,
	_ time.Duration,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.putCalls++
	if s.putErr != nil {
		return s.putErr
	}
	s.payloads[handle] = handlerLibraryTicketRecord{secretDigest: secretDigest, payload: bytes.Clone(payload)}
	return nil
}

func (s *handlerLibraryTicketStore) Consume(
	_ context.Context,
	handle string,
	secretDigest string,
) ([]byte, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.consumeCalls++
	if s.consumeErr != nil {
		return nil, false, s.consumeErr
	}
	record, found := s.payloads[handle]
	if found && record.secretDigest == secretDigest {
		delete(s.payloads, handle)
		return bytes.Clone(record.payload), true, nil
	}
	return nil, false, nil
}

func (s *handlerLibraryTicketStore) AllowIssue(
	context.Context,
	int64,
	int64,
	time.Duration,
) (bool, error) {
	return true, nil
}

type handlerTrackingReadCloser struct {
	*bytes.Reader
	onClose func()
	closed  bool
}

func (r *handlerTrackingReadCloser) Close() error {
	if !r.closed {
		r.closed = true
		r.onClose()
	}
	return nil
}

func newHandlerLibraryFile(id, name, key string, data []byte) service.LibraryFile {
	digest := sha256.Sum256(data)
	return service.LibraryFile{
		ID: id, Name: name, MIMEType: "text/plain; charset=utf-8", Extension: "txt",
		Size: int64(len(data)), StoredSize: int64(len(data)), Category: "file", Type: "document",
		Source: "uploaded", Status: service.LibraryFileStatusReady, StorageKey: key,
		Digest: hex.EncodeToString(digest[:]),
	}
}

func newHandlerLibraryDownloadFixture(t *testing.T, files map[string]service.LibraryFile, store *handlerLibraryStore) (*LibraryHandler, *[]string) {
	t.Helper()
	library := service.NewLibraryService(&handlerLibraryRepo{files: files}, store, &config.Config{
		Library: config.LibraryConfig{BatchDownloadLimit: 100, BatchDownloadMaxBytes: 100 << 20},
	})
	handler := NewLibraryHandler(library, nil)
	directory := t.TempDir()
	paths := make([]string, 0, 1)
	handler.createDownloadTemp = func() (*os.File, error) {
		file, err := os.CreateTemp(directory, "staged-*")
		if err == nil {
			paths = append(paths, file.Name())
		}
		return file, err
	}
	return handler, &paths
}

func performLibraryBatchDownload(t *testing.T, handler *LibraryHandler, body string, requestContext context.Context) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	if requestContext == nil {
		requestContext = context.Background()
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/library/files/batch-download", strings.NewReader(body)).WithContext(requestContext)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = request
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 7})
	handler.BatchDownload(c)
	return recorder
}

func performLibraryDownloadTicketIssue(t *testing.T, handler *LibraryHandler, body string) *httptest.ResponseRecorder {
	return performLibraryDownloadTicketIssueWithProto(t, handler, body, "")
}

func performLibraryDownloadTicketIssueWithProto(
	t *testing.T,
	handler *LibraryHandler,
	body string,
	forwardedProto string,
) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/library/files/download-ticket", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if forwardedProto != "" {
		request.Header.Set("X-Forwarded-Proto", forwardedProto)
	}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = request
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 7})
	handler.IssueDownloadTicket(c)
	return recorder
}

type libraryDownloadTicketCredential struct {
	ticketID string
	secret   string
	cookie   *http.Cookie
}

func performLibraryDownloadWithTicket(
	t *testing.T,
	handler *LibraryHandler,
	credential libraryDownloadTicketCredential,
	withCookie bool,
) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	downloadPath := libraryDownloadCookiePath(credential.ticketID)
	request := httptest.NewRequest(http.MethodGet, downloadPath, nil)
	if withCookie && credential.cookie != nil {
		request.AddCookie(credential.cookie)
	}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = request
	c.Params = gin.Params{{Key: "ticket_id", Value: credential.ticketID}}
	handler.DownloadWithTicket(c)
	return recorder
}

func downloadTicketCredentialFromResponse(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
) libraryDownloadTicketCredential {
	t.Helper()
	var envelope struct {
		Data service.LibraryDownloadTicket `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	require.NotEmpty(t, envelope.Data.DownloadPath)
	downloadURL, err := url.Parse(envelope.Data.DownloadPath)
	require.NoError(t, err)
	require.Equal(t, "/api/v1/library/download", path.Dir(downloadURL.Path))
	require.Empty(t, downloadURL.User)
	require.Empty(t, downloadURL.Host)
	require.Empty(t, downloadURL.RawQuery)
	require.Empty(t, downloadURL.Fragment)
	ticketID := path.Base(downloadURL.Path)
	require.True(t, service.ValidLibraryDownloadTicketID(ticketID))
	cookies := recorder.Result().Cookies()
	require.Len(t, cookies, 1)
	cookie := cookies[0]
	require.Equal(t, libraryDownloadCookieName(ticketID), cookie.Name)
	require.Equal(t, downloadURL.Path, cookie.Path)
	require.True(t, cookie.HttpOnly)
	require.Equal(t, int(service.LibraryDownloadTicketTTL/time.Second), cookie.MaxAge)
	require.NotEmpty(t, cookie.Value)
	require.NotContains(t, recorder.Body.String(), cookie.Value)
	require.NotContains(t, envelope.Data.DownloadPath, cookie.Value)
	return libraryDownloadTicketCredential{ticketID: ticketID, secret: cookie.Value, cookie: cookie}
}

func assertLibraryErrorEnvelope(t *testing.T, recorder *httptest.ResponseRecorder, reason string) {
	t.Helper()
	var envelope struct {
		Code     int               `json:"code"`
		Message  string            `json:"message"`
		Reason   string            `json:"reason"`
		Metadata map[string]string `json:"metadata"`
		Data     json.RawMessage   `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	require.Equal(t, recorder.Code, envelope.Code)
	require.Equal(t, reason, envelope.Reason)
	require.NotEmpty(t, envelope.Message)
	require.Empty(t, envelope.Metadata)
	require.Empty(t, envelope.Data)
	require.NotContains(t, recorder.Body.String(), "cause")
}

func readZIPEntry(t *testing.T, entry *zip.File) []byte {
	t.Helper()
	reader, err := entry.Open()
	require.NoError(t, err)
	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	return data
}

func requireRemovedDownloadTemps(t *testing.T, paths []string) {
	t.Helper()
	require.NotEmpty(t, paths)
	for _, path := range paths {
		_, err := os.Stat(path)
		require.ErrorIs(t, err, os.ErrNotExist, "temporary file must be removed: %s", filepath.Base(path))
	}
}

func mustPositiveInt(t *testing.T, value string) int {
	t.Helper()
	var parsed int
	_, err := fmt.Sscanf(value, "%d", &parsed)
	require.NoError(t, err)
	require.Positive(t, parsed)
	return parsed
}
