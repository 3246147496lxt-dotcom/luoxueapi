package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"golang.org/x/time/rate"
)

const (
	QuotaViewerInstallerDownloadTTL = 60 * time.Second

	quotaViewerInstallerDownloadPath = "/api/v1/quota/releases/%s/latest/download?code=%s"
	quotaViewerReleasePathPrefix     = "/3246147496lxt-dotcom/luoxueapi/releases/download/"

	quotaViewerDownloadRequestsPerSecond = 100
	quotaViewerDownloadBurst             = 200
)

var (
	ErrUnsupportedQuotaViewerInstallerPlatform = infraerrors.BadRequest(
		"UNSUPPORTED_QUOTA_VIEWER_INSTALLER_PLATFORM",
		"unsupported quota viewer installer platform",
	)
	ErrInvalidQuotaViewerInstallerDownloadCode = infraerrors.Unauthorized(
		"INVALID_QUOTA_VIEWER_INSTALLER_DOWNLOAD_CODE",
		"invalid or expired quota viewer installer download code",
	)
	ErrQuotaViewerInstallerAuthenticationRequired = infraerrors.Unauthorized(
		"QUOTA_VIEWER_INSTALLER_AUTHENTICATION_REQUIRED",
		"authentication is required to download the quota viewer installer",
	)
	ErrQuotaViewerInstallerDownloadUnavailable = infraerrors.ServiceUnavailable(
		"QUOTA_VIEWER_INSTALLER_DOWNLOAD_UNAVAILABLE",
		"quota viewer installer download is temporarily unavailable",
	)
	ErrQuotaViewerInstallerDownloadRateLimited = infraerrors.TooManyRequests(
		"QUOTA_VIEWER_INSTALLER_DOWNLOAD_RATE_LIMITED",
		"too many quota viewer installer download requests",
	)
)

// QuotaViewerInstallerAsset is the immutable release manifest exposed to the
// website. URL is deliberately omitted from JSON: callers receive a short-lived
// same-origin path instead of the backing GitHub asset URL.
type QuotaViewerInstallerAsset struct {
	Platform      string `json:"platform"`
	Version       string `json:"version"`
	Filename      string `json:"filename"`
	SHA256        string `json:"sha256"`
	Size          int64  `json:"size"`
	Architecture  string `json:"architecture"`
	SigningStatus string `json:"signing_status"`
	URL           string `json:"-"`
}

// QuotaViewerInstallerDownload is returned after an authenticated user asks
// for an installer. DownloadPath contains a 60-second, single-use credential.
type QuotaViewerInstallerDownload struct {
	DownloadPath  string `json:"download_path"`
	ExpiresIn     int    `json:"expires_in"`
	Platform      string `json:"platform"`
	Version       string `json:"version"`
	Filename      string `json:"filename"`
	SHA256        string `json:"sha256"`
	Size          int64  `json:"size"`
	Architecture  string `json:"architecture"`
	SigningStatus string `json:"signing_status"`
}

var quotaViewerInstallerAssets = map[string]QuotaViewerInstallerAsset{
	"macos": {
		Platform:      "macos",
		Version:       "2.0.0-rc.6",
		Filename:      "Luoxue-Quota-Viewer_2.0.0-rc.6_macOS-universal-UNNOTARIZED.dmg",
		SHA256:        "63c7e3890ef45e882a99e609c73f3fa5e8452cc708bc4e1ed6f0278bdc519f92",
		Size:          10655545,
		Architecture:  "universal",
		SigningStatus: "unsigned-unnotarized",
		URL:           "https://github.com/3246147496lxt-dotcom/luoxueapi/releases/download/quota-viewer-v2.0.0-rc.6/Luoxue-Quota-Viewer_2.0.0-rc.6_macOS-universal-UNNOTARIZED.dmg",
	},
	"windows": {
		Platform:      "windows",
		Version:       "2.0.0-rc.5",
		Filename:      "Luoxue-Quota-Viewer_2.0.0-rc.5_windows-x64_NSIS-UNSIGNED.exe",
		SHA256:        "2f1a0b37f3720d02564d306bc5393e71edfbc2ea5d788b89f87feb9a2b4a27d1",
		Size:          5110179,
		Architecture:  "x64",
		SigningStatus: "unsigned",
		URL:           "https://github.com/3246147496lxt-dotcom/luoxueapi/releases/download/v2.0.0-rc.5/Luoxue-Quota-Viewer_2.0.0-rc.5_windows-x64_NSIS-UNSIGNED.exe",
	},
}

// QuotaViewerInstallerTicketStore persists only hashed download credentials.
// Consume must atomically remove and return the ticket.
type QuotaViewerInstallerTicketStore interface {
	Put(ctx context.Context, digest string, payload []byte, ttl time.Duration) error
	Consume(ctx context.Context, digest string) (payload []byte, found bool, err error)
}

type quotaViewerInstallerDownloadAdmission interface {
	Allow() bool
}

type quotaViewerInstallerTicket struct {
	UserID      int64     `json:"user_id"`
	Platform    string    `json:"platform"`
	AssetSHA256 string    `json:"asset_sha256"`
	IssuedAt    time.Time `json:"issued_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// QuotaViewerInstallerDownloadService issues and consumes one-time installer
// download credentials. Redis never receives the raw credential.
type QuotaViewerInstallerDownloadService struct {
	store             QuotaViewerInstallerTicketStore
	downloadAdmission quotaViewerInstallerDownloadAdmission
	now               func() time.Time
	random            io.Reader
}

func NewQuotaViewerInstallerDownloadService(store QuotaViewerInstallerTicketStore) *QuotaViewerInstallerDownloadService {
	return &QuotaViewerInstallerDownloadService{
		store: store,
		downloadAdmission: rate.NewLimiter(
			rate.Limit(quotaViewerDownloadRequestsPerSecond),
			quotaViewerDownloadBurst,
		),
		now:    time.Now,
		random: rand.Reader,
	}
}

// Issue creates a 60-second download credential for any authenticated user.
func (s *QuotaViewerInstallerDownloadService) Issue(
	ctx context.Context,
	userID int64,
	platform string,
) (*QuotaViewerInstallerDownload, error) {
	asset, err := quotaViewerInstallerAssetForPlatform(platform)
	if err != nil {
		return nil, err
	}
	if userID <= 0 {
		return nil, ErrQuotaViewerInstallerAuthenticationRequired
	}
	if s == nil || s.store == nil {
		return nil, ErrQuotaViewerInstallerDownloadUnavailable
	}
	if err := validateQuotaViewerInstallerAsset(asset); err != nil {
		return nil, ErrQuotaViewerInstallerDownloadUnavailable.WithCause(fmt.Errorf("validate quota viewer installer asset: %w", err))
	}

	rawCode := make([]byte, 32)
	if _, err := io.ReadFull(s.random, rawCode); err != nil {
		return nil, infraerrors.InternalServer(
			"QUOTA_VIEWER_INSTALLER_DOWNLOAD_GENERATION_FAILED",
			"failed to generate quota viewer installer download code",
		).WithCause(err)
	}
	code := base64.RawURLEncoding.EncodeToString(rawCode)

	now := s.now().UTC()
	ticket := quotaViewerInstallerTicket{
		UserID:      userID,
		Platform:    asset.Platform,
		AssetSHA256: asset.SHA256,
		IssuedAt:    now,
		ExpiresAt:   now.Add(QuotaViewerInstallerDownloadTTL),
	}
	payload, err := json.Marshal(ticket)
	if err != nil {
		return nil, infraerrors.InternalServer(
			"QUOTA_VIEWER_INSTALLER_DOWNLOAD_GENERATION_FAILED",
			"failed to generate quota viewer installer download ticket",
		).WithCause(err)
	}
	if err := s.store.Put(ctx, quotaViewerInstallerDownloadDigest(code), payload, QuotaViewerInstallerDownloadTTL); err != nil {
		return nil, ErrQuotaViewerInstallerDownloadUnavailable.WithCause(fmt.Errorf("store quota viewer installer download ticket: %w", err))
	}

	return &QuotaViewerInstallerDownload{
		DownloadPath: fmt.Sprintf(
			quotaViewerInstallerDownloadPath,
			url.PathEscape(asset.Platform),
			url.QueryEscape(code),
		),
		ExpiresIn:     int(QuotaViewerInstallerDownloadTTL / time.Second),
		Platform:      asset.Platform,
		Version:       asset.Version,
		Filename:      asset.Filename,
		SHA256:        asset.SHA256,
		Size:          asset.Size,
		Architecture:  asset.Architecture,
		SigningStatus: asset.SigningStatus,
	}, nil
}

// Consume atomically spends one credential and returns the whitelisted backing
// asset. Invalid, expired, replayed, or platform-mismatched codes are
// intentionally indistinguishable.
func (s *QuotaViewerInstallerDownloadService) Consume(
	ctx context.Context,
	platform string,
	code string,
) (*QuotaViewerInstallerAsset, error) {
	asset, err := quotaViewerInstallerAssetForPlatform(platform)
	if err != nil {
		return nil, err
	}
	code = strings.TrimSpace(code)
	decoded, err := base64.RawURLEncoding.DecodeString(code)
	if err != nil || len(decoded) != 32 {
		return nil, ErrInvalidQuotaViewerInstallerDownloadCode
	}
	if s == nil || s.store == nil || s.downloadAdmission == nil {
		return nil, ErrQuotaViewerInstallerDownloadUnavailable
	}
	if !s.downloadAdmission.Allow() {
		return nil, ErrQuotaViewerInstallerDownloadRateLimited
	}

	payload, found, err := s.store.Consume(ctx, quotaViewerInstallerDownloadDigest(code))
	if err != nil {
		return nil, ErrQuotaViewerInstallerDownloadUnavailable.WithCause(fmt.Errorf("consume quota viewer installer download ticket: %w", err))
	}
	if !found {
		return nil, ErrInvalidQuotaViewerInstallerDownloadCode
	}

	var ticket quotaViewerInstallerTicket
	if err := json.Unmarshal(payload, &ticket); err != nil {
		return nil, ErrInvalidQuotaViewerInstallerDownloadCode
	}
	now := s.now().UTC()
	if ticket.UserID <= 0 || ticket.Platform != asset.Platform || ticket.AssetSHA256 != asset.SHA256 ||
		ticket.IssuedAt.After(now.Add(time.Second)) || !ticket.ExpiresAt.After(now) {
		return nil, ErrInvalidQuotaViewerInstallerDownloadCode
	}
	if err := validateQuotaViewerInstallerAsset(asset); err != nil {
		return nil, ErrQuotaViewerInstallerDownloadUnavailable.WithCause(fmt.Errorf("validate quota viewer installer asset: %w", err))
	}

	return &asset, nil
}

func quotaViewerInstallerAssetForPlatform(platform string) (QuotaViewerInstallerAsset, error) {
	asset, ok := quotaViewerInstallerAssets[strings.TrimSpace(platform)]
	if !ok || platform != strings.TrimSpace(platform) {
		return QuotaViewerInstallerAsset{}, ErrUnsupportedQuotaViewerInstallerPlatform
	}
	return asset, nil
}

func validateQuotaViewerInstallerAsset(asset QuotaViewerInstallerAsset) error {
	if asset.Platform != "macos" && asset.Platform != "windows" {
		return fmt.Errorf("unsupported platform %q", asset.Platform)
	}
	if asset.Version == "" || asset.Filename == "" || asset.Size <= 0 || asset.Architecture == "" || asset.SigningStatus == "" {
		return fmt.Errorf("incomplete asset metadata")
	}
	checksum, err := hex.DecodeString(asset.SHA256)
	if err != nil || len(checksum) != sha256.Size {
		return fmt.Errorf("invalid SHA-256 checksum")
	}

	target, err := url.Parse(asset.URL)
	if err != nil || !target.IsAbs() || target.User != nil || target.RawPath != "" || target.RawQuery != "" || target.Fragment != "" {
		return fmt.Errorf("invalid asset URL")
	}
	if !strings.EqualFold(target.Scheme, "https") || !strings.EqualFold(target.Hostname(), "github.com") || target.Port() != "" {
		return fmt.Errorf("asset URL is outside the HTTPS github.com allowlist")
	}
	if !strings.HasPrefix(target.Path, quotaViewerReleasePathPrefix) {
		return fmt.Errorf("asset URL is outside the luoxueapi release path")
	}
	remaining := strings.TrimPrefix(target.Path, quotaViewerReleasePathPrefix)
	segments := strings.Split(remaining, "/")
	if len(segments) != 2 || segments[0] == "" || segments[1] == "" ||
		segments[0] == "." || segments[0] == ".." || segments[1] == "." || segments[1] == ".." ||
		path.Base(target.Path) != asset.Filename {
		return fmt.Errorf("asset URL does not identify the declared release file")
	}
	return nil
}

func quotaViewerInstallerDownloadDigest(code string) string {
	digest := sha256.Sum256([]byte(code))
	return hex.EncodeToString(digest[:])
}
