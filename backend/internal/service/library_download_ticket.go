package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	LibraryDownloadTicketTTL = 60 * time.Second

	libraryDownloadPath                  = "/api/v1/library/download/%s"
	libraryDownloadTicketIssueLimit      = 30
	libraryDownloadTicketIssueWindow     = time.Minute
	libraryDownloadTicketHandleBytes     = 16
	libraryDownloadTicketSecretBytes     = 32
	libraryDownloadTicketMaxPayloadBytes = 32 << 10
)

var (
	ErrLibraryDownloadTicketInvalid = infraerrors.Unauthorized(
		"LIBRARY_DOWNLOAD_TICKET_INVALID",
		"The library download link is invalid or expired",
	)
	ErrLibraryDownloadTicketUnavailable = infraerrors.ServiceUnavailable(
		"LIBRARY_DOWNLOAD_TICKET_UNAVAILABLE",
		"The library download could not be prepared",
	)
	ErrLibraryDownloadTicketRateLimited = infraerrors.TooManyRequests(
		"LIBRARY_DOWNLOAD_TICKET_RATE_LIMITED",
		"Too many library download requests",
	)
)

// LibraryDownloadTicketStore persists a non-sensitive random handle as its
// lookup key and only the SHA-256 digest of the cookie secret in its payload.
// Consume must atomically remove and return the payload. AllowIssue must be an
// atomic, cross-instance Redis admission check.
type LibraryDownloadTicketStore interface {
	Put(ctx context.Context, handle, secretDigest string, payload []byte, ttl time.Duration) error
	Consume(ctx context.Context, handle, secretDigest string) (payload []byte, found bool, err error)
	AllowIssue(ctx context.Context, userID int64, limit int64, window time.Duration) (allowed bool, err error)
}

type libraryDownloadTicketRecord struct {
	UserID    int64     `json:"user_id"`
	FileIDs   []string  `json:"file_ids"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// LibraryDownloadTicket is safe metadata returned to the authenticated UI.
// DownloadPath contains only a non-sensitive handle. TicketID and Secret are
// handler-only material and can never be serialized into the JSON response.
type LibraryDownloadTicket struct {
	DownloadPath string `json:"download_path"`
	Filename     string `json:"filename"`
	FileCount    int    `json:"file_count"`
	TotalBytes   int64  `json:"total_bytes"`
	ExpiresIn    int    `json:"expires_in"`
	TicketID     string `json:"-"`
	Secret       string `json:"-"`
}

type LibraryDownloadGrant struct {
	UserID int64
	Files  []LibraryFile
}

// LibraryDownloadTicketService bridges authenticated JSON requests to a
// browser-native attachment GET without exposing the user's bearer token in a
// URL. The private file set is authorized both when issued and when consumed.
type LibraryDownloadTicketService struct {
	library *LibraryService
	store   LibraryDownloadTicketStore
	now     func() time.Time
	random  io.Reader
}

func NewLibraryDownloadTicketService(
	library *LibraryService,
	store LibraryDownloadTicketStore,
) *LibraryDownloadTicketService {
	return &LibraryDownloadTicketService{
		library: library,
		store:   store,
		now:     time.Now,
		random:  rand.Reader,
	}
}

func (s *LibraryDownloadTicketService) Issue(
	ctx context.Context,
	userID int64,
	fileIDs []string,
) (*LibraryDownloadTicket, error) {
	if s == nil || s.library == nil || s.store == nil || userID <= 0 {
		return nil, ErrLibraryDownloadTicketUnavailable
	}
	allowed, err := s.store.AllowIssue(
		ctx,
		userID,
		libraryDownloadTicketIssueLimit,
		libraryDownloadTicketIssueWindow,
	)
	if err != nil {
		return nil, ErrLibraryDownloadTicketUnavailable.WithCause(fmt.Errorf("admit library download ticket: %w", err))
	}
	if !allowed {
		return nil, ErrLibraryDownloadTicketRateLimited
	}
	files, err := s.library.ResolveBatchDownload(ctx, userID, fileIDs)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(files))
	var totalBytes int64
	for i := range files {
		ids = append(ids, files[i].ID)
		if files[i].Size > 0 {
			totalBytes += files[i].Size
		}
	}
	rawHandle := make([]byte, libraryDownloadTicketHandleBytes)
	if _, err = io.ReadFull(s.random, rawHandle); err != nil {
		return nil, ErrLibraryDownloadTicketUnavailable.WithCause(fmt.Errorf("generate library download handle: %w", err))
	}
	rawSecret := make([]byte, libraryDownloadTicketSecretBytes)
	if _, err = io.ReadFull(s.random, rawSecret); err != nil {
		return nil, ErrLibraryDownloadTicketUnavailable.WithCause(fmt.Errorf("generate library download secret: %w", err))
	}
	handle := base64.RawURLEncoding.EncodeToString(rawHandle)
	secret := base64.RawURLEncoding.EncodeToString(rawSecret)
	now := s.now().UTC()
	payload, err := json.Marshal(libraryDownloadTicketRecord{
		UserID: userID, FileIDs: ids, IssuedAt: now, ExpiresAt: now.Add(LibraryDownloadTicketTTL),
	})
	if err != nil || len(payload) > libraryDownloadTicketMaxPayloadBytes {
		return nil, ErrLibraryDownloadTicketUnavailable.WithCause(fmt.Errorf("encode library download ticket"))
	}
	if err = s.store.Put(ctx, handle, libraryDownloadTicketDigest(secret), payload, LibraryDownloadTicketTTL); err != nil {
		return nil, ErrLibraryDownloadTicketUnavailable.WithCause(fmt.Errorf("store library download ticket: %w", err))
	}

	filename := "library-files.zip"
	if len(files) == 1 {
		filename = files[0].Name
	}
	return &LibraryDownloadTicket{
		DownloadPath: fmt.Sprintf(libraryDownloadPath, url.PathEscape(handle)),
		Filename:     filename,
		FileCount:    len(files),
		TotalBytes:   totalBytes,
		ExpiresIn:    int(LibraryDownloadTicketTTL / time.Second),
		TicketID:     handle,
		Secret:       secret,
	}, nil
}

func (s *LibraryDownloadTicketService) Consume(
	ctx context.Context,
	handle string,
	secret string,
) (*LibraryDownloadGrant, error) {
	if !validLibraryDownloadCredential(handle, libraryDownloadTicketHandleBytes) ||
		!validLibraryDownloadCredential(secret, libraryDownloadTicketSecretBytes) {
		return nil, ErrLibraryDownloadTicketInvalid
	}
	if s == nil || s.library == nil || s.store == nil {
		return nil, ErrLibraryDownloadTicketUnavailable
	}
	payload, found, err := s.store.Consume(ctx, handle, libraryDownloadTicketDigest(secret))
	if err != nil {
		return nil, ErrLibraryDownloadTicketUnavailable.WithCause(fmt.Errorf("consume library download ticket: %w", err))
	}
	if !found || len(payload) == 0 || len(payload) > libraryDownloadTicketMaxPayloadBytes {
		return nil, ErrLibraryDownloadTicketInvalid
	}
	var ticket libraryDownloadTicketRecord
	if err = json.Unmarshal(payload, &ticket); err != nil {
		return nil, ErrLibraryDownloadTicketInvalid
	}
	now := s.now().UTC()
	if ticket.UserID <= 0 || len(ticket.FileIDs) == 0 ||
		ticket.IssuedAt.After(now.Add(time.Second)) || !ticket.ExpiresAt.After(now) ||
		ticket.ExpiresAt.After(ticket.IssuedAt.Add(LibraryDownloadTicketTTL+time.Second)) {
		return nil, ErrLibraryDownloadTicketInvalid
	}
	files, err := s.library.ResolveBatchDownload(ctx, ticket.UserID, ticket.FileIDs)
	if err != nil {
		if errors.Is(err, ErrLibraryFileNotFound) || errors.Is(err, ErrLibraryFileAccessDenied) ||
			errors.Is(err, ErrLibraryInvalidRequest) || errors.Is(err, ErrLibraryBatchLimit) ||
			errors.Is(err, ErrLibraryBatchTooLarge) {
			return nil, ErrLibraryDownloadTicketInvalid
		}
		return nil, err
	}
	return &LibraryDownloadGrant{UserID: ticket.UserID, Files: files}, nil
}

func validLibraryDownloadCredential(value string, expectedBytes int) bool {
	decoded, err := base64.RawURLEncoding.Strict().DecodeString(value)
	return err == nil && len(decoded) == expectedBytes
}

// ValidLibraryDownloadTicketID lets the HTTP boundary reject malformed path
// parameters before deriving a cookie name/path or emitting Set-Cookie.
func ValidLibraryDownloadTicketID(value string) bool {
	return validLibraryDownloadCredential(value, libraryDownloadTicketHandleBytes)
}

func libraryDownloadTicketDigest(secret string) string {
	digest := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(digest[:])
}
