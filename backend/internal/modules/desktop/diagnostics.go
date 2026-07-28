package desktop

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/google/uuid"
)

const (
	DiagnosticRequestMaxBytes = 64 * 1024
	diagnosticRetention       = 7 * 24 * time.Hour
	maxDiagnosticRecent       = 20
)

var (
	diagnosticModelPattern     = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,100}$`)
	diagnosticRequestIDPattern = regexp.MustCompile(`^req_[A-Za-z0-9_-]{1,64}$`)
	diagnosticSensitivePattern = regexp.MustCompile(`(?i)(authorization|bearer[[:space:]]|api[ _-]?key|x-api-key|cookie|set-cookie|prompt|response[ _-]?body|request[ _-]?body|token|sk-[A-Za-z0-9])`)
)

type DiagnosticUpload struct {
	AppVersion   string             `json:"app_version"`
	Platform     string             `json:"platform"`
	Architecture string             `json:"architecture"`
	OSVersion    string             `json:"os_version"`
	Gateway      DiagnosticGateway  `json:"gateway"`
	Codex        DiagnosticCodex    `json:"codex"`
	Route        DiagnosticRoute    `json:"route"`
	Requests     DiagnosticRequests `json:"requests"`
}

type DiagnosticGateway struct {
	Status          string `json:"status"`
	Port            int    `json:"port"`
	TakeoverEnabled bool   `json:"takeover_enabled"`
}

type DiagnosticCodex struct {
	ConfigStatus  string                   `json:"config_status"`
	Installations []DiagnosticInstallation `json:"installations"`
}

type DiagnosticInstallation struct {
	Kind      string `json:"kind"`
	Installed bool   `json:"installed"`
	Version   string `json:"version"`
}

type DiagnosticRoute struct {
	GroupID             int64  `json:"group_id"`
	Model               string `json:"model"`
	AvailableRouteCount int    `json:"available_route_count"`
}

type DiagnosticRequests struct {
	SampleCount         int                       `json:"sample_count"`
	SuccessCount        int                       `json:"success_count"`
	ErrorCount          int                       `json:"error_count"`
	AverageDurationMS   int64                     `json:"average_duration_ms"`
	AverageFirstTokenMS int64                     `json:"average_first_token_ms"`
	Recent              []DiagnosticRecentRequest `json:"recent"`
}

type DiagnosticRecentRequest struct {
	OccurredAt   time.Time `json:"occurred_at"`
	Model        string    `json:"model"`
	StatusCode   int       `json:"status_code"`
	DurationMS   int64     `json:"duration_ms"`
	FirstTokenMS int64     `json:"first_token_ms"`
	RequestID    string    `json:"request_id"`
}

type DiagnosticCreated struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type DiagnosticMetadata struct {
	ID                 string    `json:"id"`
	DeviceID           string    `json:"device_id"`
	UserID             int64     `json:"user_id"`
	AppVersion         string    `json:"app_version"`
	Platform           string    `json:"platform"`
	Architecture       string    `json:"architecture"`
	OSVersion          string    `json:"os_version"`
	GatewayStatus      string    `json:"gateway_status"`
	CodexConfigStatus  string    `json:"codex_config_status"`
	RequestSampleCount int       `json:"request_sample_count"`
	RequestErrorCount  int       `json:"request_error_count"`
	CreatedAt          time.Time `json:"created_at"`
	ExpiresAt          time.Time `json:"expires_at"`
}

type DiagnosticRecord struct {
	DiagnosticMetadata
	DeviceInternalID   int64  `json:"-"`
	DeviceTokenVersion int64  `json:"-"`
	EncryptedPayload   string `json:"-"`
}

type DiagnosticDetail struct {
	DiagnosticMetadata
	Diagnostic DiagnosticUpload `json:"diagnostic"`
}

type DiagnosticPage struct {
	Items []DiagnosticMetadata
	Total int64
}

type DiagnosticsRepository interface {
	CreateDiagnostic(ctx context.Context, record DiagnosticRecord) (*DiagnosticCreated, error)
	ListDiagnostics(ctx context.Context, page, pageSize int) (*DiagnosticPage, error)
	GetDiagnostic(ctx context.Context, publicID string) (*DiagnosticRecord, error)
}

func (s *Service) UploadDiagnostic(ctx context.Context, subject DesktopSubject, upload DiagnosticUpload) (*DiagnosticCreated, error) {
	if !subject.HasScope(ScopeDiagnosticsWrite) {
		return nil, ErrDesktopScope
	}
	if s.diagnostics == nil || s.diagnosticKeyErr != nil || len(s.diagnosticKey) != 32 {
		return nil, ErrDiagnosticsUnavailable
	}
	now := s.now().UTC()
	if err := validateDiagnostic(now, upload); err != nil {
		return nil, err
	}
	payload, err := json.Marshal(upload)
	if err != nil {
		return nil, ErrDiagnosticsInvalid.WithCause(err)
	}
	publicID := uuid.NewString()
	encrypted, err := encryptDiagnostic(s.diagnosticKey, publicID, payload)
	if err != nil {
		return nil, ErrDiagnosticsUnavailable.WithCause(err)
	}
	return s.diagnostics.CreateDiagnostic(ctx, DiagnosticRecord{
		DiagnosticMetadata: DiagnosticMetadata{
			ID: publicID, UserID: subject.UserID,
			AppVersion: upload.AppVersion, Platform: upload.Platform,
			Architecture: upload.Architecture, OSVersion: upload.OSVersion,
			GatewayStatus: upload.Gateway.Status, CodexConfigStatus: upload.Codex.ConfigStatus,
			RequestSampleCount: upload.Requests.SampleCount, RequestErrorCount: upload.Requests.ErrorCount,
			CreatedAt: now, ExpiresAt: now.Add(diagnosticRetention),
		},
		DeviceInternalID: subject.DeviceID, DeviceTokenVersion: subject.TokenVersion,
		EncryptedPayload: encrypted,
	})
}

func (s *Service) ListDiagnostics(ctx context.Context, page, pageSize int) (*DiagnosticPage, error) {
	if s.diagnostics == nil {
		return nil, ErrDiagnosticsUnavailable
	}
	return s.diagnostics.ListDiagnostics(ctx, page, pageSize)
}

func (s *Service) GetDiagnostic(ctx context.Context, publicID string) (*DiagnosticDetail, error) {
	if _, err := uuid.Parse(publicID); err != nil || s.diagnostics == nil {
		return nil, ErrDiagnosticNotFound
	}
	record, err := s.diagnostics.GetDiagnostic(ctx, publicID)
	if err != nil {
		return nil, err
	}
	plaintext, err := decryptDiagnostic(s.diagnosticKey, record.ID, record.EncryptedPayload)
	if err != nil {
		return nil, ErrDiagnosticsUnavailable.WithCause(err)
	}
	var payload DiagnosticUpload
	if err := json.Unmarshal(plaintext, &payload); err != nil {
		return nil, ErrDiagnosticsUnavailable.WithCause(err)
	}
	return &DiagnosticDetail{DiagnosticMetadata: record.DiagnosticMetadata, Diagnostic: payload}, nil
}

func resolveDiagnosticKey(cfg *config.Config) ([]byte, error) {
	if cfg == nil {
		return nil, errors.New("desktop diagnostics config is missing")
	}
	if configured := strings.TrimSpace(cfg.Desktop.DiagnosticsEncryptionKey); configured != "" {
		key, err := hex.DecodeString(configured)
		if err != nil || len(key) != 32 {
			return nil, errors.New("desktop diagnostics encryption key must be 64 hex characters")
		}
		return key, nil
	}
	if strings.TrimSpace(cfg.JWT.Secret) == "" {
		return nil, errors.New("desktop diagnostics encryption key is unavailable")
	}
	derived := sha256.Sum256([]byte("desktop-diagnostics-v1\x00" + cfg.JWT.Secret))
	return derived[:], nil
}

func validateDiagnostic(now time.Time, upload DiagnosticUpload) error {
	if !validStrictSemver(upload.AppVersion) || upload.Platform != "macos" ||
		(upload.Architecture != "arm64" && upload.Architecture != "x86_64") ||
		strings.TrimSpace(upload.OSVersion) == "" || len(upload.OSVersion) > 50 ||
		!validDiagnosticGateway(upload.Gateway.Status, upload.Gateway.Port) ||
		!oneOf(upload.Codex.ConfigStatus, "managed", "unmanaged", "missing", "error") ||
		len(upload.Codex.Installations) > 2 || upload.Route.GroupID < 0 ||
		upload.Route.AvailableRouteCount < 0 || upload.Route.AvailableRouteCount > 1000 ||
		upload.Requests.SampleCount < 0 || upload.Requests.SampleCount > 10000 ||
		upload.Requests.SuccessCount < 0 || upload.Requests.ErrorCount < 0 ||
		upload.Requests.SuccessCount+upload.Requests.ErrorCount != upload.Requests.SampleCount ||
		upload.Requests.AverageDurationMS < 0 || upload.Requests.AverageFirstTokenMS < 0 ||
		upload.Requests.AverageFirstTokenMS > upload.Requests.AverageDurationMS ||
		len(upload.Requests.Recent) > maxDiagnosticRecent {
		return ErrDiagnosticsInvalid
	}
	if !validDiagnosticString(upload.OSVersion) ||
		(upload.Route.Model != "" && (!diagnosticModelPattern.MatchString(upload.Route.Model) || !validDiagnosticString(upload.Route.Model))) {
		return ErrDiagnosticsSensitive
	}
	seenKinds := map[string]bool{}
	for _, installation := range upload.Codex.Installations {
		if !oneOf(installation.Kind, "cli", "desktop") || seenKinds[installation.Kind] ||
			(installation.Installed && !validStrictSemver(installation.Version)) ||
			(!installation.Installed && installation.Version != "") || !validDiagnosticString(installation.Version) {
			return ErrDiagnosticsInvalid
		}
		seenKinds[installation.Kind] = true
	}
	for _, recent := range upload.Requests.Recent {
		if recent.OccurredAt.IsZero() || recent.OccurredAt.After(now.Add(5*time.Minute)) ||
			!diagnosticModelPattern.MatchString(recent.Model) || !validDiagnosticString(recent.Model) ||
			recent.StatusCode < 100 || recent.StatusCode > 599 ||
			recent.DurationMS < 0 || recent.FirstTokenMS < 0 || recent.FirstTokenMS > recent.DurationMS ||
			!diagnosticRequestIDPattern.MatchString(recent.RequestID) || !validDiagnosticString(recent.RequestID) {
			return ErrDiagnosticsInvalid
		}
	}
	return nil
}

func validDiagnosticGateway(status string, port int) bool {
	if port < 0 || port > 65535 {
		return false
	}
	switch status {
	case "running":
		return port > 0
	case "stopped", "error":
		return true
	default:
		return false
	}
}

func validDiagnosticString(value string) bool {
	return !diagnosticSensitivePattern.MatchString(value)
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func encryptDiagnostic(key []byte, aad string, plaintext []byte) (string, error) {
	gcm, err := diagnosticGCM(key)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, plaintext, []byte(aad))
	return base64.StdEncoding.EncodeToString(sealed), nil
}

func decryptDiagnostic(key []byte, aad, encoded string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	gcm, err := diagnosticGCM(key)
	if err != nil {
		return nil, err
	}
	if len(data) < gcm.NonceSize() {
		return nil, fmt.Errorf("diagnostic ciphertext is truncated")
	}
	return gcm.Open(nil, data[:gcm.NonceSize()], data[gcm.NonceSize():], []byte(aad))
}

func diagnosticGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
