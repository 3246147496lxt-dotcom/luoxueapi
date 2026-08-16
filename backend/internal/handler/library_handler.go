package handler

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	maxLibraryBatchRequestBytes         = 1 << 20
	maxLibraryBatchArchiveOverheadBytes = 2 << 20
	libraryDownloadCookieNamePrefix     = "sub2api_library_download_"
	libraryDownloadTicketPathPrefix     = "/api/v1/library/download/"
)

var errLibraryDownloadStageTooLarge = errors.New("staged library download exceeds its size limit")

type LibraryHandler struct {
	library            *service.LibraryService
	downloadTickets    *service.LibraryDownloadTicketService
	downloadGate       *libraryDownloadGate
	createDownloadTemp func() (*os.File, error)
}

func NewLibraryHandler(
	library *service.LibraryService,
	downloadTickets *service.LibraryDownloadTicketService,
) *LibraryHandler {
	return &LibraryHandler{
		library:         library,
		downloadTickets: downloadTickets,
		downloadGate:    newDefaultLibraryDownloadGate(),
		createDownloadTemp: func() (*os.File, error) {
			return os.CreateTemp("", "sub2api-library-download-*")
		},
	}
}

func (h *LibraryHandler) List(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	userID, ok := h.user(c)
	if !ok {
		return
	}
	page, err := positiveLibraryQueryInt(c.Query("page"), 1)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	pageSizeValue := c.Query("pageSize")
	if pageSizeValue == "" {
		pageSizeValue = c.Query("page_size")
	}
	pageSize, err := positiveLibraryQueryInt(pageSizeValue, 20)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	result, err := h.library.List(c.Request.Context(), userID, service.LibraryFileQuery{
		Q: c.Query("q"), Category: c.Query("category"), Source: c.Query("source"),
		Type: c.Query("type"), Sort: c.Query("sort"), Page: page, PageSize: pageSize,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *LibraryHandler) Upload(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	forceCloseUntilTranscriptionBodyRead(c)
	userID, ok := h.user(c)
	if !ok {
		return
	}
	release, err := h.library.AdmitUpload(userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	defer release()
	filename, declaredMIME, data, err := readSingleChatAttachment(c, h.library.MaxUploadBytes())
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) || errors.Is(err, service.ErrChatAttachmentTooLarge) {
			response.ErrorFrom(c, service.ErrLibraryFileTooLarge)
			return
		}
		response.ErrorFrom(c, service.ErrLibraryInvalidRequest.WithCause(err))
		return
	}
	file, err := h.library.UploadAdmitted(c.Request.Context(), userID, service.LibraryUpload{
		Filename: filename, DeclaredMIME: declaredMIME, Data: data,
	}, "uploaded")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, gin.H{"file": file})
}

func (h *LibraryHandler) Detail(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	userID, ok := h.user(c)
	if !ok {
		return
	}
	file, err := h.library.Get(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, file)
}

func (h *LibraryHandler) Thumbnail(c *gin.Context) { h.writeFile(c, "thumbnail", "inline") }
func (h *LibraryHandler) Preview(c *gin.Context)   { h.writeFile(c, "preview", "inline") }
func (h *LibraryHandler) Download(c *gin.Context)  { h.writeFile(c, "download", "attachment") }

func (h *LibraryHandler) writeFile(c *gin.Context, mode, disposition string) {
	c.Header("Cache-Control", "private, no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	userID, ok := h.user(c)
	if !ok {
		return
	}
	opened, err := h.library.Open(c.Request.Context(), userID, c.Param("id"), mode)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	defer func() { _ = opened.Body.Close() }()
	if mode == "preview" {
		c.Header("Content-Security-Policy", "sandbox; default-src 'none'")
	}
	c.Header("Content-Length", strconv.FormatInt(opened.File.Size, 10))
	c.Header("Content-Disposition", safeLibraryContentDisposition(disposition, opened.File.Name))
	c.Status(http.StatusOK)
	c.Header("Content-Type", opened.File.MIMEType)
	if _, err = io.Copy(c.Writer, opened.Body); err != nil {
		_ = c.Error(service.ErrLibraryDownloadFailed.WithCause(err))
	}
}

func (h *LibraryHandler) Delete(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	userID, ok := h.user(c)
	if !ok {
		return
	}
	if err := h.library.Delete(c.Request.Context(), userID, c.Param("id")); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

type libraryBatchDownloadRequest struct {
	FileIDs      []string `json:"file_ids,omitempty"`
	FileIDsCamel []string `json:"fileIds,omitempty"`
}

// IssueDownloadTicket turns a JWT-authenticated selection into a short-lived,
// single-use browser download path. The path contains only a non-sensitive
// handle; its independent 256-bit secret is scoped to that handle in an
// HttpOnly cookie and is never serialized into JSON.
func (h *LibraryHandler) IssueDownloadTicket(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	userID, ok := h.user(c)
	if !ok {
		return
	}
	if h.downloadTickets == nil {
		response.ErrorFrom(c, service.ErrLibraryDownloadTicketUnavailable)
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxLibraryBatchRequestBytes)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	var request libraryBatchDownloadRequest
	if err := decoder.Decode(&request); err != nil {
		response.ErrorFrom(c, service.ErrLibraryInvalidRequest)
		return
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF ||
		(len(request.FileIDs) > 0 && len(request.FileIDsCamel) > 0) {
		response.ErrorFrom(c, service.ErrLibraryInvalidRequest)
		return
	}
	ids := request.FileIDs
	if len(ids) == 0 {
		ids = request.FileIDsCamel
	}
	ticket, err := h.downloadTickets.Issue(c.Request.Context(), userID, ids)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if ticket == nil || !service.ValidLibraryDownloadTicketID(ticket.TicketID) || ticket.Secret == "" {
		response.ErrorFrom(c, service.ErrLibraryDownloadTicketUnavailable)
		return
	}
	setLibraryDownloadCookie(c, ticket.TicketID, ticket.Secret)
	response.Created(c, ticket)
}

// DownloadWithTicket is intentionally outside JWT middleware so a normal
// browser navigation can stream directly to its download manager. Consume is
// atomic and reauthorizes the private file set before opening it. The public
// route applies an independent Redis/IP fail-close limiter before this handler.
func (h *LibraryHandler) DownloadWithTicket(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	c.Header("Referrer-Policy", "no-referrer")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("X-Frame-Options", "SAMEORIGIN")
	c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'self'")
	if h == nil || h.downloadTickets == nil {
		response.ErrorFrom(c, service.ErrLibraryDownloadTicketUnavailable)
		return
	}
	ticketID := c.Param("ticket_id")
	if !service.ValidLibraryDownloadTicketID(ticketID) {
		response.ErrorFrom(c, service.ErrLibraryDownloadTicketInvalid)
		return
	}
	cookieName := libraryDownloadCookieName(ticketID)
	secret := ""
	if cookie, err := c.Request.Cookie(cookieName); err == nil {
		secret = cookie.Value
	}
	clearLibraryDownloadCookie(c, ticketID)
	grant, err := h.downloadTickets.Consume(c.Request.Context(), ticketID, secret)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	h.writeResolvedDownload(c, grant.UserID, grant.Files)
}

func setLibraryDownloadCookie(c *gin.Context, ticketID, secret string) {
	secure := libraryDownloadRequestIsHTTPS(c.Request)
	sameSite := http.SameSiteLaxMode
	if secure {
		sameSite = http.SameSiteNoneMode
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     libraryDownloadCookieName(ticketID),
		Value:    secret,
		Path:     libraryDownloadCookiePath(ticketID),
		Expires:  time.Now().Add(service.LibraryDownloadTicketTTL),
		MaxAge:   int(service.LibraryDownloadTicketTTL / time.Second),
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
	})
}

func clearLibraryDownloadCookie(c *gin.Context, ticketID string) {
	secure := libraryDownloadRequestIsHTTPS(c.Request)
	sameSite := http.SameSiteLaxMode
	if secure {
		sameSite = http.SameSiteNoneMode
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     libraryDownloadCookieName(ticketID),
		Path:     libraryDownloadCookiePath(ticketID),
		Expires:  time.Unix(1, 0).UTC(),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
	})
}

func libraryDownloadCookieName(ticketID string) string {
	digest := sha256.Sum256([]byte(ticketID))
	return libraryDownloadCookieNamePrefix + hex.EncodeToString(digest[:])
}

func libraryDownloadCookiePath(ticketID string) string {
	return libraryDownloadTicketPathPrefix + url.PathEscape(ticketID)
}

func libraryDownloadRequestIsHTTPS(request *http.Request) bool {
	if request == nil {
		return false
	}
	if request.TLS != nil || strings.EqualFold(strings.TrimSpace(request.URL.Scheme), "https") {
		return true
	}
	forwarded := strings.Split(request.Header.Get("X-Forwarded-Proto"), ",")
	return len(forwarded) > 0 && strings.EqualFold(strings.TrimSpace(forwarded[0]), "https")
}

func (h *LibraryHandler) BatchDownload(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	userID, ok := h.user(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxLibraryBatchRequestBytes)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	var request libraryBatchDownloadRequest
	if err := decoder.Decode(&request); err != nil {
		response.ErrorFrom(c, service.ErrLibraryInvalidRequest)
		return
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF || (len(request.FileIDs) > 0 && len(request.FileIDsCamel) > 0) {
		response.ErrorFrom(c, service.ErrLibraryInvalidRequest)
		return
	}
	ids := request.FileIDs
	if len(ids) == 0 {
		ids = request.FileIDsCamel
	}
	files, err := h.library.ResolveBatchDownload(c.Request.Context(), userID, ids)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	h.writeResolvedDownload(c, userID, files)
}

// writeResolvedDownload stages and verifies the entire response before it
// commits HTTP 200. It is intentionally reusable by authenticated POST batch
// downloads and short-lived anonymous download tickets. A client disconnect
// can still interrupt the final file-to-socket copy, but storage failures,
// digest mismatches and malformed ZIP output can no longer produce a partial
// success response.
func (h *LibraryHandler) writeResolvedDownload(c *gin.Context, userID int64, files []service.LibraryFile) {
	c.Header("Cache-Control", "private, no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	reservedBytes, err := libraryDownloadReservationBytes(files)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	lease, err := h.downloadGate.acquire(c.Request.Context(), userID, reservedBytes)
	if err != nil {
		if errors.Is(err, errLibraryDownloadBusy) {
			c.Header("Retry-After", strconv.Itoa(int(defaultLibraryDownloadAcquireTimeout/time.Second)))
		}
		response.ErrorFrom(c, err)
		return
	}
	defer lease.release()
	operationCtx := lease.context()

	staged, err := h.stageResolvedDownload(operationCtx, userID, files)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	defer func() {
		if cleanupErr := staged.closeAndRemove(); cleanupErr != nil {
			_ = c.Error(service.ErrLibraryDownloadFailed.WithCause(cleanupErr))
		}
	}()
	if contextErr := operationCtx.Err(); contextErr != nil {
		response.ErrorFrom(c, service.ErrLibraryDownloadFailed.WithCause(contextErr))
		return
	}
	responseController := http.NewResponseController(c.Writer)
	writeDeadlineSet := false
	if deadline, ok := operationCtx.Deadline(); ok {
		if deadlineErr := responseController.SetWriteDeadline(deadline); deadlineErr == nil {
			writeDeadlineSet = true
		} else if !errors.Is(deadlineErr, http.ErrNotSupported) {
			response.ErrorFrom(c, service.ErrLibraryDownloadFailed.WithCause(deadlineErr))
			return
		}
	}
	if writeDeadlineSet {
		defer func() { _ = responseController.SetWriteDeadline(time.Time{}) }()
	}
	c.Header("Content-Type", staged.contentType)
	c.Header("Content-Length", strconv.FormatInt(staged.size, 10))
	c.Header("Content-Disposition", staged.contentDisposition)
	c.Status(http.StatusOK)
	if _, err = io.Copy(c.Writer, &libraryContextReader{ctx: operationCtx, reader: staged.file}); err != nil {
		_ = c.Error(service.ErrLibraryDownloadFailed.WithCause(err))
	}
}

type stagedLibraryDownload struct {
	file               *os.File
	path               string
	size               int64
	contentType        string
	contentDisposition string
}

func (s *stagedLibraryDownload) closeAndRemove() error {
	if s == nil {
		return nil
	}
	var closeErr, removeErr error
	if s.file != nil {
		closeErr = s.file.Close()
		s.file = nil
	}
	if s.path != "" {
		removeErr = os.Remove(s.path)
		s.path = ""
	}
	return errors.Join(closeErr, removeErr)
}

func (h *LibraryHandler) stageResolvedDownload(ctx context.Context, userID int64, files []service.LibraryFile) (_ *stagedLibraryDownload, err error) {
	if h == nil || h.library == nil || userID <= 0 || len(files) == 0 {
		return nil, service.ErrLibraryInvalidRequest
	}
	createTemp := h.createDownloadTemp
	if createTemp == nil {
		createTemp = func() (*os.File, error) {
			return os.CreateTemp("", "sub2api-library-download-*")
		}
	}
	temp, err := createTemp()
	if err != nil {
		return nil, service.ErrLibraryDownloadFailed.WithCause(err)
	}
	staged := &stagedLibraryDownload{file: temp, path: temp.Name()}
	completed := false
	defer func() {
		if !completed {
			err = errors.Join(err, staged.closeAndRemove())
		}
	}()
	if err = temp.Chmod(0o600); err != nil {
		return nil, service.ErrLibraryDownloadFailed.WithCause(err)
	}
	info, statErr := temp.Stat()
	if statErr != nil || info.Mode().Perm() != 0o600 {
		if statErr == nil {
			statErr = fmt.Errorf("unexpected temporary-file permissions %04o", info.Mode().Perm())
		}
		return nil, service.ErrLibraryDownloadFailed.WithCause(statErr)
	}

	if len(files) == 1 {
		err = h.stageSingleLibraryFile(ctx, userID, files[0], staged)
	} else {
		err = h.stageLibraryZIP(ctx, userID, files, staged)
	}
	if err != nil {
		return nil, err
	}
	if _, err = temp.Seek(0, io.SeekStart); err != nil {
		return nil, service.ErrLibraryDownloadFailed.WithCause(err)
	}
	completed = true
	return staged, nil
}

func (h *LibraryHandler) stageSingleLibraryFile(ctx context.Context, userID int64, file service.LibraryFile, staged *stagedLibraryDownload) error {
	opened, err := h.library.Open(ctx, userID, file.ID, "download")
	if err != nil {
		return err
	}
	limited := &libraryMaxWriter{writer: staged.file, remaining: h.library.BatchMaxBytes()}
	_, copyErr := io.Copy(limited, &libraryContextReader{ctx: ctx, reader: opened.Body})
	closeErr := opened.Body.Close()
	if err = errors.Join(copyErr, closeErr); err != nil {
		if errors.Is(err, errLibraryDownloadStageTooLarge) {
			return service.ErrLibraryBatchTooLarge.WithCause(err)
		}
		return service.ErrLibraryDownloadFailed.WithCause(err)
	}
	position, err := staged.file.Seek(0, io.SeekCurrent)
	if err != nil {
		return service.ErrLibraryDownloadFailed.WithCause(err)
	}
	if opened.File == nil || position != opened.File.StoredSize {
		return service.ErrLibraryDownloadFailed.WithCause(errors.New("staged library file size mismatch"))
	}
	staged.size = position
	staged.contentType = opened.File.MIMEType
	staged.contentDisposition = safeLibraryContentDisposition("attachment", opened.File.Name)
	return nil
}

func (h *LibraryHandler) stageLibraryZIP(ctx context.Context, userID int64, files []service.LibraryFile, staged *stagedLibraryDownload) error {
	archiveLimit := h.library.BatchMaxBytes() + maxLibraryBatchArchiveOverheadBytes
	limited := &libraryMaxWriter{writer: staged.file, remaining: archiveLimit}
	archive := zip.NewWriter(limited)
	archiveClosed := false
	var buildErr error
	defer func() {
		if !archiveClosed {
			_ = archive.Close()
		}
	}()

	usedNames := make(map[string]int, len(files))
	expectedSizes := make([]int64, 0, len(files))
	for i := range files {
		if err := ctx.Err(); err != nil {
			return service.ErrLibraryDownloadFailed.WithCause(err)
		}
		opened, openErr := h.library.Open(ctx, userID, files[i].ID, "download")
		if openErr != nil {
			return openErr
		}
		entry, createErr := archive.Create(uniqueLibraryZipName(files[i].Name, usedNames))
		if createErr == nil {
			_, createErr = io.Copy(entry, &libraryContextReader{ctx: ctx, reader: opened.Body})
		}
		closeErr := opened.Body.Close()
		if buildErr = errors.Join(createErr, closeErr); buildErr != nil {
			if errors.Is(buildErr, errLibraryDownloadStageTooLarge) {
				return service.ErrLibraryBatchTooLarge.WithCause(buildErr)
			}
			return service.ErrLibraryDownloadFailed.WithCause(buildErr)
		}
		if opened.File == nil || opened.File.StoredSize <= 0 {
			return service.ErrLibraryDownloadFailed.WithCause(errors.New("library file metadata is incomplete"))
		}
		expectedSizes = append(expectedSizes, opened.File.StoredSize)
	}
	if buildErr = archive.Close(); buildErr != nil {
		archiveClosed = true
		if errors.Is(buildErr, errLibraryDownloadStageTooLarge) {
			return service.ErrLibraryBatchTooLarge.WithCause(buildErr)
		}
		return service.ErrLibraryDownloadFailed.WithCause(buildErr)
	}
	archiveClosed = true

	size, err := staged.file.Seek(0, io.SeekCurrent)
	if err != nil {
		return service.ErrLibraryDownloadFailed.WithCause(err)
	}
	if size <= 0 || size > archiveLimit {
		return service.ErrLibraryBatchTooLarge.WithCause(errLibraryDownloadStageTooLarge)
	}
	if err = validateStagedLibraryZIP(ctx, staged.file, size, expectedSizes, h.library.BatchMaxBytes()); err != nil {
		return service.ErrLibraryDownloadFailed.WithCause(err)
	}
	staged.size = size
	staged.contentType = "application/zip"
	staged.contentDisposition = safeLibraryContentDisposition("attachment", "library-files.zip")
	return nil
}

func validateStagedLibraryZIP(ctx context.Context, file *os.File, size int64, expectedSizes []int64, maxUncompressed int64) error {
	archive, err := zip.NewReader(file, size)
	if err != nil {
		return err
	}
	if len(archive.File) != len(expectedSizes) {
		return errors.New("staged library ZIP entry count mismatch")
	}
	remaining := maxUncompressed
	for i := range archive.File {
		if archive.File[i].UncompressedSize64 != uint64(expectedSizes[i]) {
			return errors.New("staged library ZIP entry size mismatch")
		}
		entry, openErr := archive.File[i].Open()
		if openErr != nil {
			return openErr
		}
		limited := &libraryMaxWriter{writer: io.Discard, remaining: remaining}
		_, readErr := io.Copy(limited, &libraryContextReader{ctx: ctx, reader: entry})
		closeErr := entry.Close()
		readErr = errors.Join(readErr, closeErr)
		if readErr != nil {
			return readErr
		}
		remaining = limited.remaining
	}
	return nil
}

type libraryContextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *libraryContextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := r.reader.Read(p)
	if err == nil {
		if contextErr := r.ctx.Err(); contextErr != nil {
			return n, contextErr
		}
	}
	return n, err
}

type libraryMaxWriter struct {
	writer    io.Writer
	remaining int64
}

func (w *libraryMaxWriter) Write(p []byte) (int, error) {
	if w.remaining <= 0 {
		return 0, errLibraryDownloadStageTooLarge
	}
	if int64(len(p)) <= w.remaining {
		n, err := w.writer.Write(p)
		w.remaining -= int64(n)
		return n, err
	}
	n, err := w.writer.Write(p[:w.remaining])
	w.remaining -= int64(n)
	if err != nil {
		return n, err
	}
	return n, errLibraryDownloadStageTooLarge
}

func (h *LibraryHandler) Storage(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	userID, ok := h.user(c)
	if !ok {
		return
	}
	storage, err := h.library.Storage(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, storage)
}

func (h *LibraryHandler) user(c *gin.Context) (int64, bool) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "User not authenticated")
		return 0, false
	}
	if h == nil || h.library == nil {
		response.ErrorFrom(c, service.ErrLibraryUnavailable)
		return 0, false
	}
	return subject.UserID, true
}

func positiveLibraryQueryInt(value string, fallback int) (int, error) {
	if strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, service.ErrLibraryInvalidRequest
	}
	return parsed, nil
}

func safeLibraryContentDisposition(disposition, filename string) string {
	if strings.ToLower(strings.TrimSpace(disposition)) != "inline" {
		disposition = "attachment"
	} else {
		disposition = "inline"
	}
	filename = strings.TrimSpace(filepath.Base(strings.ReplaceAll(filename, "\\", "/")))
	filename = strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' || r == 0 {
			return '_'
		}
		return r
	}, filename)
	if filename == "" || filename == "." {
		filename = "file"
	}
	if value := mime.FormatMediaType(disposition, map[string]string{"filename": filename}); value != "" {
		return value
	}
	return disposition
}

func uniqueLibraryZipName(name string, names map[string]int) string {
	name = strings.TrimSpace(filepath.Base(strings.ReplaceAll(name, "\\", "/")))
	if name == "" || name == "." {
		name = "file"
	}
	key := strings.ToLower(name)
	if names[key] == 0 {
		names[key] = 1
		return name
	}
	extension := filepath.Ext(name)
	base := strings.TrimSuffix(name, extension)
	for suffix := names[key] + 1; ; suffix++ {
		candidate := fmt.Sprintf("%s (%d)%s", base, suffix, extension)
		candidateKey := strings.ToLower(candidate)
		if names[candidateKey] != 0 {
			continue
		}
		names[key] = suffix
		names[candidateKey] = 1
		return candidate
	}
}
