package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/modules/desktop"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type createPairingStore struct {
	pairing desktop.Pairing
}

type todayUsageRepository struct {
	desktop.Repository
	usage desktop.TodayUsage
}

type diagnosticHandlerRepository struct {
	desktop.Repository
	created desktop.DiagnosticRecord
}

func (r *diagnosticHandlerRepository) CreateDiagnostic(_ context.Context, record desktop.DiagnosticRecord) (*desktop.DiagnosticCreated, error) {
	r.created = record
	return &desktop.DiagnosticCreated{ID: record.ID, CreatedAt: record.CreatedAt, ExpiresAt: record.ExpiresAt}, nil
}

func (r *diagnosticHandlerRepository) ListDiagnostics(context.Context, int, int) (*desktop.DiagnosticPage, error) {
	return &desktop.DiagnosticPage{}, nil
}

func (r *diagnosticHandlerRepository) GetDiagnostic(context.Context, string) (*desktop.DiagnosticRecord, error) {
	return nil, desktop.ErrDiagnosticNotFound
}

type releaseHandlerRepository struct {
	desktop.Repository
	channel string
}

func (r *releaseHandlerRepository) GetReleaseChannel(context.Context, int64, int64, int64) (string, error) {
	return r.channel, nil
}

func (r *todayUsageRepository) GetTodayUsage(context.Context, int64, int64, time.Time, time.Time) (*desktop.TodayUsage, error) {
	out := r.usage
	return &out, nil
}

func (s *createPairingStore) Create(_ context.Context, pairing desktop.Pairing, _ time.Duration) error {
	s.pairing = pairing
	return nil
}
func (s *createPairingStore) GetByUserCode(context.Context, string) (*desktop.Pairing, error) {
	return nil, desktop.ErrPairingState
}
func (s *createPairingStore) GetByDeviceCode(context.Context, string) (*desktop.Pairing, error) {
	return nil, desktop.ErrPairingState
}
func (s *createPairingStore) BeginApproval(context.Context, string) (*desktop.Pairing, error) {
	return nil, desktop.ErrPairingState
}
func (s *createPairingStore) FinishApproval(context.Context, string, int64, int64) error {
	return desktop.ErrPairingState
}
func (s *createPairingStore) CancelApproval(context.Context, string) error { return nil }
func (s *createPairingStore) Consume(context.Context, string) (*desktop.Pairing, error) {
	return nil, desktop.ErrPairingState
}
func (s *createPairingStore) RestoreConsumed(context.Context, string) error { return nil }

func TestDesktopCreatePairingReturnsSafeCompleteVerificationURI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &createPairingStore{}
	svc := desktop.NewService(store, nil, nil, nil, &config.Config{})
	h := NewDesktopHandler(svc)

	verifier := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	digest := sha256.Sum256([]byte(verifier))
	body, err := json.Marshal(map[string]any{
		"code_challenge":        base64.RawURLEncoding.EncodeToString(digest[:]),
		"code_challenge_method": "S256",
		"installation_id":       "installation-1",
		"device_name":           "Work Mac",
		"platform":              "macos",
		"architecture":          "arm64",
		"os_version":            "15.0",
		"app_version":           "1.0.0",
	})
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/desktop/pairings", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	h.CreatePairing(c)

	require.Equal(t, http.StatusCreated, recorder.Code)
	var envelope struct {
		Code int `json:"code"`
		Data struct {
			DeviceCode              string `json:"device_code"`
			UserCode                string `json:"user_code"`
			VerificationURI         string `json:"verification_uri"`
			VerificationURIComplete string `json:"verification_uri_complete"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	require.Zero(t, envelope.Code)
	require.Equal(t, "/desktop/authorize", envelope.Data.VerificationURI)

	complete, err := url.Parse(envelope.Data.VerificationURIComplete)
	require.NoError(t, err)
	require.Equal(t, envelope.Data.VerificationURI, complete.Path)
	require.Equal(t, envelope.Data.UserCode, complete.Query().Get("user_code"))
	require.Empty(t, complete.Query().Get("device_code"))
	require.NotContains(t, envelope.Data.VerificationURIComplete, envelope.Data.DeviceCode)
	require.NotEmpty(t, store.pairing.DeviceCodeHash)
}

func TestDesktopTodayUsageReturnsStableEnvelopeDTO(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &todayUsageRepository{usage: desktop.TodayUsage{
		Requests: 12, Tokens: 3456, Cost: 1.25, Balance: 9.5,
	}}
	svc := desktop.NewService(nil, repo, nil, nil, &config.Config{})
	h := NewDesktopHandler(svc)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("desktop_subject", desktop.DesktopSubject{
		DeviceID: 41, UserID: 7, Scopes: []string{desktop.ScopeProfileRead},
	})
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/desktop/usage/today", nil)

	h.TodayUsage(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	var envelope struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Requests int64   `json:"requests"`
			Tokens   int64   `json:"tokens"`
			Cost     float64 `json:"cost"`
			Balance  float64 `json:"balance"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	require.Zero(t, envelope.Code)
	require.Equal(t, "success", envelope.Message)
	require.Equal(t, int64(12), envelope.Data.Requests)
	require.Equal(t, int64(3456), envelope.Data.Tokens)
	require.Equal(t, 1.25, envelope.Data.Cost)
	require.Equal(t, 9.5, envelope.Data.Balance)
}

func TestDesktopDiagnosticUploadAcceptsFixedDTOAndRejectsUnknownOrOversizedFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Now().UTC().Add(-time.Minute).Truncate(time.Second)
	valid := map[string]any{
		"app_version": "0.1.0", "platform": "macos", "architecture": "arm64", "os_version": "15.0",
		"gateway": map[string]any{"status": "running", "port": 11430, "takeover_enabled": true},
		"codex": map[string]any{
			"config_status": "managed",
			"installations": []map[string]any{
				{"kind": "cli", "installed": true, "version": "0.1.0"},
				{"kind": "desktop", "installed": true, "version": "0.1.0"},
			},
		},
		"route": map[string]any{"group_id": 12, "model": "gpt-5.x", "available_route_count": 2},
		"requests": map[string]any{
			"sample_count": 3, "success_count": 2, "error_count": 1,
			"average_duration_ms": 1234, "average_first_token_ms": 456,
			"recent": []map[string]any{{
				"occurred_at": now.Format(time.RFC3339), "model": "gpt-5.x", "status_code": 200,
				"duration_ms": 1234, "first_token_ms": 456, "request_id": "req_safe_1",
			}},
		},
	}

	call := func(payload any) *httptest.ResponseRecorder {
		repo := &diagnosticHandlerRepository{}
		cfg := &config.Config{}
		cfg.JWT.Secret = "diagnostic-handler-secret"
		h := NewDesktopHandler(desktop.NewService(nil, repo, nil, nil, cfg))
		body, err := json.Marshal(payload)
		require.NoError(t, err)
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Set("desktop_subject", desktop.DesktopSubject{
			DeviceID: 41, UserID: 7, TokenVersion: 1, Scopes: []string{desktop.ScopeDiagnosticsWrite},
		})
		c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/desktop/diagnostics", bytes.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		h.UploadDiagnostic(c)
		return recorder
	}

	success := call(valid)
	require.Equal(t, http.StatusCreated, success.Code)
	require.Contains(t, success.Body.String(), `"expires_at"`)
	require.NotContains(t, success.Body.String(), `encrypted_payload`)

	stopped := make(map[string]any, len(valid))
	for key, value := range valid {
		stopped[key] = value
	}
	stopped["gateway"] = map[string]any{"status": "stopped", "port": nil, "takeover_enabled": false}
	stoppedResponse := call(stopped)
	require.Equal(t, http.StatusCreated, stoppedResponse.Code, "Rust Option<u16> serializes a stopped gateway port as null")

	forbidden := make(map[string]any, len(valid)+1)
	for key, value := range valid {
		forbidden[key] = value
	}
	forbidden["prompt"] = "must never be accepted"
	rejected := call(forbidden)
	require.Equal(t, http.StatusBadRequest, rejected.Code)
	require.Contains(t, rejected.Body.String(), `DESKTOP_DIAGNOSTICS_INVALID`)

	forbiddenHeaders := make(map[string]any, len(valid))
	for key, value := range valid {
		forbiddenHeaders[key] = value
	}
	requestSummary := make(map[string]any)
	for key, value := range valid["requests"].(map[string]any) {
		requestSummary[key] = value
	}
	requestSummary["headers"] = map[string]string{"Authorization": "redacted"}
	forbiddenHeaders["requests"] = requestSummary
	rejected = call(forbiddenHeaders)
	require.Equal(t, http.StatusBadRequest, rejected.Code)
	require.Contains(t, rejected.Body.String(), `DESKTOP_DIAGNOSTICS_INVALID`)

	repo := &diagnosticHandlerRepository{}
	cfg := &config.Config{}
	cfg.JWT.Secret = "diagnostic-handler-secret"
	h := NewDesktopHandler(desktop.NewService(nil, repo, nil, nil, cfg))
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("desktop_subject", desktop.DesktopSubject{
		DeviceID: 41, UserID: 7, TokenVersion: 1, Scopes: []string{desktop.ScopeDiagnosticsWrite},
	})
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/desktop/diagnostics", strings.NewReader(
		`{"padding":"`+strings.Repeat("x", desktop.DiagnosticRequestMaxBytes)+`"}`,
	))
	h.UploadDiagnostic(c)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Empty(t, repo.created.ID)
}

func TestDesktopReleaseHandlersFollowTauriAndPublicStableContracts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{}
	cfg.JWT.Secret = "release-handler-secret"
	cfg.Desktop.Releases = []config.DesktopReleaseConfig{
		{Version: "9.0.0", Channel: desktop.ReleaseChannelInternal, Target: desktop.ReleaseTargetDarwin, Arch: desktop.ReleaseArchUniversal, Enabled: true, URL: "https://downloads.example.com/internal.tar.gz", InstallerURL: "https://downloads.example.com/internal.dmg", Signature: "internal-sig", PubDate: "2026-07-28T00:00:00Z"},
		{Version: "0.2.0", Channel: desktop.ReleaseChannelStable, Target: desktop.ReleaseTargetDarwin, Arch: desktop.ReleaseArchUniversal, Enabled: true, URL: "https://downloads.example.com/stable.tar.gz", InstallerURL: "https://downloads.example.com/stable.dmg", SHA256: strings.Repeat("a", 64), Signature: "stable-sig", Notes: "Stable", PubDate: "2026-07-28T00:00:00Z"},
	}
	h := NewDesktopHandler(desktop.NewService(nil, &releaseHandlerRepository{channel: desktop.ReleaseChannelStable}, nil, nil, cfg))

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Params = gin.Params{{Key: "target", Value: "darwin"}, {Key: "arch", Value: "universal"}, {Key: "current_version", Value: "0.1.0"}}
	c.Set("desktop_subject", desktop.DesktopSubject{DeviceID: 41, UserID: 7, TokenVersion: 1, Scopes: []string{desktop.ScopeProfileRead}})
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/desktop/releases/darwin/universal/0.1.0", nil)
	h.Release(c)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, `{"version":"0.2.0","notes":"Stable","pub_date":"2026-07-28T00:00:00Z","url":"https://downloads.example.com/stable.tar.gz","signature":"stable-sig"}`, recorder.Body.String())

	publicRecorder := httptest.NewRecorder()
	publicContext, _ := gin.CreateTestContext(publicRecorder)
	publicContext.Request = httptest.NewRequest(http.MethodGet, "/api/v1/public/desktop/releases/macos/latest", nil)
	h.PublicLatestRelease(publicContext)
	require.Equal(t, http.StatusOK, publicRecorder.Code)
	require.Contains(t, publicRecorder.Body.String(), `"installer_url":"https://downloads.example.com/stable.dmg"`)
	require.NotContains(t, publicRecorder.Body.String(), "internal")
	require.NotContains(t, publicRecorder.Body.String(), "stable.tar.gz")

	downloadRecorder := httptest.NewRecorder()
	downloadContext, _ := gin.CreateTestContext(downloadRecorder)
	downloadContext.Request = httptest.NewRequest(http.MethodGet, "/api/v1/public/desktop/releases/macos/latest/download", nil)
	h.PublicLatestReleaseDownload(downloadContext)
	require.Equal(t, http.StatusTemporaryRedirect, downloadRecorder.Code)
	require.Equal(t, "https://downloads.example.com/stable.dmg", downloadRecorder.Header().Get("Location"))
}

func TestDesktopReleaseHandlersReturnNoUpdateInvalidParametersAndMissingDownload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{}
	cfg.JWT.Secret = "release-handler-secret"
	cfg.Desktop.Releases = []config.DesktopReleaseConfig{{
		Version: "9.0.0", Channel: desktop.ReleaseChannelInternal, Target: desktop.ReleaseTargetDarwin,
		Arch: desktop.ReleaseArchUniversal, Enabled: true, InstallerURL: "https://downloads.example.com/internal.dmg",
		URL: "https://downloads.example.com/internal.tar.gz", Signature: "sig", PubDate: "2026-07-28T00:00:00Z",
	}}
	h := NewDesktopHandler(desktop.NewService(nil, &releaseHandlerRepository{channel: desktop.ReleaseChannelStable}, nil, nil, cfg))
	subject := desktop.DesktopSubject{DeviceID: 41, UserID: 7, TokenVersion: 1, Scopes: []string{desktop.ScopeProfileRead}}

	noUpdate := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(noUpdate)
	c.Params = gin.Params{{Key: "target", Value: "darwin"}, {Key: "arch", Value: "universal"}, {Key: "current_version", Value: "0.1.0"}}
	c.Set("desktop_subject", subject)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/desktop/releases/darwin/universal/0.1.0", nil)
	h.Release(c)
	require.Equal(t, http.StatusNoContent, noUpdate.Code)

	invalid := httptest.NewRecorder()
	c, _ = gin.CreateTestContext(invalid)
	c.Params = gin.Params{{Key: "target", Value: "windows"}, {Key: "arch", Value: "universal"}, {Key: "current_version", Value: "not-semver"}}
	c.Set("desktop_subject", subject)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/desktop/releases/windows/universal/not-semver", nil)
	h.Release(c)
	require.Equal(t, http.StatusBadRequest, invalid.Code)

	missing := httptest.NewRecorder()
	c, _ = gin.CreateTestContext(missing)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/public/desktop/releases/macos/latest/download", nil)
	h.PublicLatestReleaseDownload(c)
	require.Equal(t, http.StatusNotFound, missing.Code)
	require.Empty(t, missing.Header().Get("Location"))
}
