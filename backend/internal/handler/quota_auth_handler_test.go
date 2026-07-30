package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/modules/quotaauth"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type quotaRefreshHandlerRepository struct {
	rotation quotaauth.RefreshRotation
}

func (r *quotaRefreshHandlerRepository) ReserveDevice(context.Context, int64, quotaauth.Pairing) (*quotaauth.Device, error) {
	return nil, quotaauth.ErrServiceUnavailable
}

func (r *quotaRefreshHandlerRepository) DeletePendingDevice(context.Context, int64) error {
	return quotaauth.ErrServiceUnavailable
}

func (r *quotaRefreshHandlerRepository) ActivateDevice(context.Context, int64, string, string, time.Time) (*quotaauth.Device, error) {
	return nil, quotaauth.ErrServiceUnavailable
}

func (r *quotaRefreshHandlerRepository) RotateSession(
	_ context.Context,
	rotation quotaauth.RefreshRotation,
) (*quotaauth.RefreshRotationOutcome, error) {
	r.rotation = rotation
	return &quotaauth.RefreshRotationOutcome{
		Device: &quotaauth.Device{
			ID:           41,
			PublicID:     "41d69f86-6f59-4d70-8f06-18952934a81e",
			UserID:       7,
			ClientID:     quotaauth.ClientID,
			Scope:        quotaauth.ScopeRead,
			Name:         "Work laptop",
			Platform:     "macos",
			Architecture: "arm64",
			Status:       quotaauth.DeviceStatusActive,
			TokenVersion: 1,
		},
		Result: quotaauth.RefreshRotationCommitted,
	}, nil
}

func (r *quotaRefreshHandlerRepository) GetAuthorizedDevice(context.Context, int64, int64) (*quotaauth.Device, error) {
	return nil, quotaauth.ErrServiceUnavailable
}

func (r *quotaRefreshHandlerRepository) ListDevices(context.Context, int64) ([]quotaauth.Device, error) {
	return nil, quotaauth.ErrServiceUnavailable
}

func (r *quotaRefreshHandlerRepository) RevokeDevice(context.Context, int64, string) (*quotaauth.Device, error) {
	return nil, quotaauth.ErrServiceUnavailable
}

func TestQuotaAuthHandlerRejectsUnknownJSONAndMarksSecretsNoStore(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewQuotaAuthHandler(nil)
	router := gin.New()
	router.POST("/api/v1/quota/pairings", h.CreatePairing)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/quota/pairings",
		strings.NewReader(`{"client_id":"luoxue-quota-viewer","scope":"quota:read","unexpected":true}`),
	)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Equal(t, "private, no-store", recorder.Header().Get("Cache-Control"))
	require.Equal(t, "no-cache", recorder.Header().Get("Pragma"))
	require.NotContains(t, recorder.Body.String(), "unexpected")
}

func TestQuotaBearerCannotUseWebsiteDeviceManagementHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewQuotaAuthHandler(nil)
	router := gin.New()
	router.GET("/api/v1/quota/devices", h.ListDevices)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/quota/devices", nil)
	req.Header.Set("Authorization", "Bearer quota-reader")
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.Equal(t, "private, no-store", recorder.Header().Get("Cache-Control"))
}

func TestQuotaRefreshHandlerRequiresRotationIDAndCandidateTogether(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewQuotaAuthHandler(nil)
	router := gin.New()
	router.POST("/api/v1/quota/session/refresh", h.RefreshSession)

	requests := []string{
		`{"client_id":"luoxue-quota-viewer","refresh_token":"qvrt_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","rotation_id":"af69f4b4-5824-4f48-9149-3e55f1b5ca2d"}`,
		`{"client_id":"luoxue-quota-viewer","refresh_token":"qvrt_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","candidate_refresh_token":"qvrt_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}`,
	}
	for _, body := range requests {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/quota/session/refresh",
			strings.NewReader(body),
		)
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, req)

		require.Equal(t, http.StatusBadRequest, recorder.Code)
		require.Equal(t, "private, no-store", recorder.Header().Get("Cache-Control"))
		require.NotContains(t, recorder.Body.String(), "qvrt_")
	}
}

func TestQuotaRefreshHandlerReturnsCandidateV1CommitMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := new(quotaRefreshHandlerRepository)
	service := quotaauth.NewService(nil, repo, &config.Config{
		JWT: config.JWTConfig{Secret: strings.Repeat("q", 40)},
	})
	h := NewQuotaAuthHandler(service)
	router := gin.New()
	router.POST("/api/v1/quota/session/refresh", h.RefreshSession)

	candidate := "qvrt_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	rotationID := "af69f4b4-5824-4f48-9149-3e55f1b5ca2d"
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/quota/session/refresh",
		strings.NewReader(`{
			"client_id":"luoxue-quota-viewer",
			"refresh_token":"qvrt_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"rotation_id":"`+rotationID+`",
			"candidate_refresh_token":"`+candidate+`"
		}`),
	)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "private, no-store", recorder.Header().Get("Cache-Control"))
	require.Contains(t, recorder.Body.String(), `"refresh_token":"`+candidate+`"`)
	require.Contains(t, recorder.Body.String(), `"refresh_protocol":"candidate-v1"`)
	require.Contains(t, recorder.Body.String(), `"rotation_id":"`+rotationID+`"`)
	require.Contains(t, recorder.Body.String(), `"rotation_result":"committed"`)
	require.Equal(t, rotationID, repo.rotation.RotationID)
	require.Len(t, repo.rotation.ReplacementHash, 64)
	require.NotContains(t, repo.rotation.ReplacementHash, "qvrt_")
}
