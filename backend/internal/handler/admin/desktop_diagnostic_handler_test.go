package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/modules/desktop"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type adminDiagnosticRepository struct {
	desktop.Repository
	record desktop.DiagnosticRecord
}

func (r *adminDiagnosticRepository) CreateDiagnostic(_ context.Context, record desktop.DiagnosticRecord) (*desktop.DiagnosticCreated, error) {
	r.record = record
	return &desktop.DiagnosticCreated{ID: record.ID, CreatedAt: record.CreatedAt, ExpiresAt: record.ExpiresAt}, nil
}

func (r *adminDiagnosticRepository) ListDiagnostics(context.Context, int, int) (*desktop.DiagnosticPage, error) {
	return &desktop.DiagnosticPage{Items: []desktop.DiagnosticMetadata{r.record.DiagnosticMetadata}, Total: 1}, nil
}

func (r *adminDiagnosticRepository) GetDiagnostic(_ context.Context, publicID string) (*desktop.DiagnosticRecord, error) {
	if publicID != r.record.ID {
		return nil, desktop.ErrDiagnosticNotFound
	}
	record := r.record
	return &record, nil
}

func setupAdminDiagnosticHandler(t *testing.T) (*DesktopDiagnosticHandler, string) {
	t.Helper()
	repo := &adminDiagnosticRepository{}
	cfg := &config.Config{}
	cfg.JWT.Secret = "admin-diagnostic-test-secret"
	svc := desktop.NewService(nil, repo, nil, nil, cfg)
	created, err := svc.UploadDiagnostic(context.Background(), desktop.DesktopSubject{
		DeviceID: 41, UserID: 7, TokenVersion: 1, Scopes: []string{desktop.ScopeDiagnosticsWrite},
	}, desktop.DiagnosticUpload{
		AppVersion: "0.1.0", Platform: "macos", Architecture: "arm64", OSVersion: "15.0",
		Gateway: desktop.DiagnosticGateway{Status: "running", Port: 11430, TakeoverEnabled: true},
		Codex:   desktop.DiagnosticCodex{ConfigStatus: "managed"},
		Route:   desktop.DiagnosticRoute{GroupID: 12, Model: "gpt-5.x", AvailableRouteCount: 2},
		Requests: desktop.DiagnosticRequests{
			SampleCount: 0, SuccessCount: 0, ErrorCount: 0,
		},
	})
	require.NoError(t, err)
	return NewDesktopDiagnosticHandler(svc), created.ID
}

func adminDiagnosticContext(recorder *httptest.ResponseRecorder, method, path, role string) *gin.Context {
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(method, path, nil)
	if role != "" {
		c.Set(string(middleware.ContextKeyUserRole), role)
	}
	return c
}

func TestDesktopDiagnosticAdminRequiresAdminAndListOmitsPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _ := setupAdminDiagnosticHandler(t)

	forbidden := httptest.NewRecorder()
	h.List(adminDiagnosticContext(forbidden, http.MethodGet, "/api/v1/admin/desktop/diagnostics", "user"))
	require.Equal(t, http.StatusForbidden, forbidden.Code)

	allowed := httptest.NewRecorder()
	h.List(adminDiagnosticContext(allowed, http.MethodGet, "/api/v1/admin/desktop/diagnostics", "admin"))
	require.Equal(t, http.StatusOK, allowed.Code)
	require.Contains(t, allowed.Body.String(), `"gateway_status":"running"`)
	require.NotContains(t, allowed.Body.String(), `"diagnostic"`)
	require.NotContains(t, allowed.Body.String(), `"encrypted_payload"`)
}

func TestDesktopDiagnosticAdminDetailAndDownloadDecryptAndSetAuditActions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, diagnosticID := setupAdminDiagnosticHandler(t)

	detailRecorder := httptest.NewRecorder()
	detailContext := adminDiagnosticContext(detailRecorder, http.MethodGet, "/api/v1/admin/desktop/diagnostics/"+diagnosticID, "admin")
	detailContext.Params = gin.Params{{Key: "id", Value: diagnosticID}}
	h.Get(detailContext)
	require.Equal(t, http.StatusOK, detailRecorder.Code)
	require.Contains(t, detailRecorder.Body.String(), `"diagnostic"`)
	require.Contains(t, detailRecorder.Body.String(), `"model":"gpt-5.x"`)
	require.NotContains(t, detailRecorder.Body.String(), `"encrypted_payload"`)
	require.Equal(t, "private, no-store", detailRecorder.Header().Get("Cache-Control"))
	action, exists := detailContext.Get("audit_action")
	require.True(t, exists)
	require.Equal(t, "admin.desktop_diagnostics.view", action)

	downloadRecorder := httptest.NewRecorder()
	downloadContext := adminDiagnosticContext(downloadRecorder, http.MethodGet, "/api/v1/admin/desktop/diagnostics/"+diagnosticID+"/download", "admin")
	downloadContext.Params = gin.Params{{Key: "id", Value: diagnosticID}}
	h.Download(downloadContext)
	require.Equal(t, http.StatusOK, downloadRecorder.Code)
	require.Contains(t, downloadRecorder.Header().Get("Content-Disposition"), diagnosticID)
	require.Contains(t, downloadRecorder.Body.String(), `"diagnostic"`)
	require.NotContains(t, downloadRecorder.Body.String(), `"encrypted_payload"`)
	action, exists = downloadContext.Get("audit_action")
	require.True(t, exists)
	require.Equal(t, "admin.desktop_diagnostics.download", action)
}

func TestDesktopDiagnosticAdminRejectsInvalidOrMissingID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _ := setupAdminDiagnosticHandler(t)
	recorder := httptest.NewRecorder()
	c := adminDiagnosticContext(recorder, http.MethodGet, "/api/v1/admin/desktop/diagnostics/not-a-uuid", "admin")
	c.Params = gin.Params{{Key: "id", Value: "not-a-uuid"}}
	h.Get(c)
	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.Equal(t, "private, no-store", recorder.Header().Get("Cache-Control"), "sensitive admin reads must remain uncacheable even on errors")
}
