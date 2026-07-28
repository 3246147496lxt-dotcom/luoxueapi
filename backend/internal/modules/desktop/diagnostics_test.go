package desktop

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type diagnosticMemoryRepository struct {
	Repository
	records          map[string]DiagnosticRecord
	expectedDeviceID int64
	expectedUserID   int64
	expectedVersion  int64
}

func (r *diagnosticMemoryRepository) CreateDiagnostic(_ context.Context, record DiagnosticRecord) (*DiagnosticCreated, error) {
	if record.DeviceInternalID != r.expectedDeviceID || record.UserID != r.expectedUserID ||
		record.DeviceTokenVersion != r.expectedVersion {
		return nil, ErrDeviceUnauthorized
	}
	if r.records == nil {
		r.records = make(map[string]DiagnosticRecord)
	}
	r.records[record.ID] = record
	return &DiagnosticCreated{ID: record.ID, CreatedAt: record.CreatedAt, ExpiresAt: record.ExpiresAt}, nil
}

func (r *diagnosticMemoryRepository) ListDiagnostics(context.Context, int, int) (*DiagnosticPage, error) {
	items := make([]DiagnosticMetadata, 0, len(r.records))
	for _, record := range r.records {
		items = append(items, record.DiagnosticMetadata)
	}
	return &DiagnosticPage{Items: items, Total: int64(len(items))}, nil
}

func (r *diagnosticMemoryRepository) GetDiagnostic(_ context.Context, publicID string) (*DiagnosticRecord, error) {
	record, ok := r.records[publicID]
	if !ok {
		return nil, ErrDiagnosticNotFound
	}
	return &record, nil
}

func validDiagnosticUpload(now time.Time) DiagnosticUpload {
	return DiagnosticUpload{
		AppVersion: "0.1.0", Platform: "macos", Architecture: "arm64", OSVersion: "15.0",
		Gateway: DiagnosticGateway{Status: "running", Port: 11430, TakeoverEnabled: true},
		Codex: DiagnosticCodex{
			ConfigStatus: "managed",
			Installations: []DiagnosticInstallation{
				{Kind: "cli", Installed: true, Version: "0.1.0"},
				{Kind: "desktop", Installed: true, Version: "0.1.0"},
			},
		},
		Route: DiagnosticRoute{GroupID: 12, Model: "gpt-5.x", AvailableRouteCount: 2},
		Requests: DiagnosticRequests{
			SampleCount: 3, SuccessCount: 2, ErrorCount: 1,
			AverageDurationMS: 1234, AverageFirstTokenMS: 456,
			Recent: []DiagnosticRecentRequest{{
				OccurredAt: now.Add(-time.Minute), Model: "gpt-5.x", StatusCode: 200,
				DurationMS: 1234, FirstTokenMS: 456, RequestID: "req_safe_1",
			}},
		},
	}
}

func newDiagnosticTestService(repo *diagnosticMemoryRepository, now time.Time, secret string) *Service {
	cfg := &config.Config{}
	cfg.JWT.Secret = secret
	svc := NewService(nil, repo, nil, nil, cfg)
	svc.now = func() time.Time { return now }
	return svc
}

func TestDiagnosticUploadEncryptsAndAdminReadDecrypts(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	repo := &diagnosticMemoryRepository{expectedDeviceID: 41, expectedUserID: 7, expectedVersion: 3}
	svc := newDiagnosticTestService(repo, now, "diagnostics-test-secret")
	subject := DesktopSubject{DeviceID: 41, UserID: 7, TokenVersion: 3, Scopes: []string{ScopeDiagnosticsWrite}}

	created, err := svc.UploadDiagnostic(context.Background(), subject, validDiagnosticUpload(now))
	require.NoError(t, err)
	require.Equal(t, now, created.CreatedAt)
	require.Equal(t, now.Add(7*24*time.Hour), created.ExpiresAt)

	record := repo.records[created.ID]
	require.NotEmpty(t, record.EncryptedPayload)
	require.NotContains(t, record.EncryptedPayload, "gpt-5.x")
	require.NotContains(t, record.EncryptedPayload, "req_safe_1")

	detail, err := svc.GetDiagnostic(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, "gpt-5.x", detail.Diagnostic.Route.Model)
	require.Equal(t, "req_safe_1", detail.Diagnostic.Requests.Recent[0].RequestID)
}

func TestDiagnosticCiphertextRejectsTamperingAndWrongKey(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	repo := &diagnosticMemoryRepository{expectedDeviceID: 41, expectedUserID: 7, expectedVersion: 1}
	svc := newDiagnosticTestService(repo, now, "first-secret")
	created, err := svc.UploadDiagnostic(context.Background(), DesktopSubject{
		DeviceID: 41, UserID: 7, TokenVersion: 1, Scopes: []string{ScopeDiagnosticsWrite},
	}, validDiagnosticUpload(now))
	require.NoError(t, err)

	wrongKeyService := newDiagnosticTestService(repo, now, "different-secret")
	_, err = wrongKeyService.GetDiagnostic(context.Background(), created.ID)
	require.ErrorIs(t, err, ErrDiagnosticsUnavailable)

	record := repo.records[created.ID]
	ciphertext, err := base64.StdEncoding.DecodeString(record.EncryptedPayload)
	require.NoError(t, err)
	ciphertext[len(ciphertext)-1] ^= 0xff
	record.EncryptedPayload = base64.StdEncoding.EncodeToString(ciphertext)
	repo.records[created.ID] = record
	_, err = svc.GetDiagnostic(context.Background(), created.ID)
	require.ErrorIs(t, err, ErrDiagnosticsUnavailable)
}

func TestDiagnosticUploadRejectsSensitiveOversizedAndUnauthorizedPayloads(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	repo := &diagnosticMemoryRepository{expectedDeviceID: 41, expectedUserID: 7, expectedVersion: 1}
	svc := newDiagnosticTestService(repo, now, "diagnostics-test-secret")
	allowed := DesktopSubject{DeviceID: 41, UserID: 7, TokenVersion: 1, Scopes: []string{ScopeDiagnosticsWrite}}

	sensitive := validDiagnosticUpload(now)
	sensitive.Route.Model = "prompt"
	_, err := svc.UploadDiagnostic(context.Background(), allowed, sensitive)
	require.ErrorIs(t, err, ErrDiagnosticsSensitive)

	tooManyRecent := validDiagnosticUpload(now)
	tooManyRecent.Requests.Recent = make([]DiagnosticRecentRequest, maxDiagnosticRecent+1)
	_, err = svc.UploadDiagnostic(context.Background(), allowed, tooManyRecent)
	require.ErrorIs(t, err, ErrDiagnosticsInvalid)

	_, err = svc.UploadDiagnostic(context.Background(), DesktopSubject{
		DeviceID: 41, UserID: 7, TokenVersion: 1, Scopes: []string{ScopeProfileRead},
	}, validDiagnosticUpload(now))
	require.ErrorIs(t, err, ErrDesktopScope)

	_, err = svc.UploadDiagnostic(context.Background(), DesktopSubject{
		DeviceID: 42, UserID: 7, TokenVersion: 1, Scopes: []string{ScopeDiagnosticsWrite},
	}, validDiagnosticUpload(now))
	require.ErrorIs(t, err, ErrDeviceUnauthorized)
}

func TestDiagnosticEncryptionKeySupportsExplicitServerConfiguration(t *testing.T) {
	cfg := &config.Config{}
	cfg.Desktop.DiagnosticsEncryptionKey = strings.Repeat("ab", 32)
	key, err := resolveDiagnosticKey(cfg)
	require.NoError(t, err)
	require.Len(t, key, 32)
	require.Equal(t, byte(0xab), key[0])

	cfg.Desktop.DiagnosticsEncryptionKey = "not-a-64-character-hex-key"
	_, err = resolveDiagnosticKey(cfg)
	require.Error(t, err)
}

func TestDiagnosticGatewayPortMatchesRustOptionalPortContract(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		status  string
		port    int
		wantErr bool
	}{
		{name: "running valid", status: "running", port: 11430},
		{name: "running missing", status: "running", port: 0, wantErr: true},
		{name: "stopped missing", status: "stopped", port: 0},
		{name: "error missing", status: "error", port: 0},
		{name: "stopped negative", status: "stopped", port: -1, wantErr: true},
		{name: "error too large", status: "error", port: 65536, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upload := validDiagnosticUpload(now)
			upload.Gateway.Status = tt.status
			upload.Gateway.Port = tt.port
			err := validateDiagnostic(now, upload)
			if tt.wantErr {
				require.ErrorIs(t, err, ErrDiagnosticsInvalid)
				return
			}
			require.NoError(t, err)
		})
	}
}
