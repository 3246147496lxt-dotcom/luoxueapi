package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type healthAdminStub struct {
	*stubAdminService
	deleted []int64
}

func (s *healthAdminStub) ListAccounts(ctx context.Context, page, pageSize int, platform, accountType, status, search string, groupID int64, privacyMode, sortBy, sortOrder string) ([]service.Account, int64, error) {
	now := time.Now()
	matched := make([]service.Account, 0, len(s.accounts))
	for _, acc := range s.accounts {
		if platform != "" && acc.Platform != platform {
			continue
		}
		switch status {
		case service.StatusError:
			if acc.Status != service.StatusError {
				continue
			}
		case service.AccountListStatusExpired:
			if !acc.AutoPauseOnExpired || acc.ExpiresAt == nil || now.Before(*acc.ExpiresAt) {
				continue
			}
		default:
			if status != "" && acc.Status != status {
				continue
			}
		}
		matched = append(matched, acc)
	}
	total := int64(len(matched))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > len(matched) {
		if pageSize < 1 {
			pageSize = len(matched)
		}
	}
	start := (page - 1) * pageSize
	if start >= len(matched) {
		return []service.Account{}, total, nil
	}
	end := start + pageSize
	if end > len(matched) {
		end = len(matched)
	}
	return matched[start:end], total, nil
}

func (s *healthAdminStub) GetAccountsByIDs(_ context.Context, ids []int64) ([]*service.Account, error) {
	byID := make(map[int64]service.Account, len(s.accounts))
	for _, acc := range s.accounts {
		byID[acc.ID] = acc
	}
	out := make([]*service.Account, 0, len(ids))
	for _, id := range ids {
		acc, ok := byID[id]
		if !ok {
			continue
		}
		copy := acc
		out = append(out, &copy)
	}
	return out, nil
}

func (s *healthAdminStub) DeleteAccount(_ context.Context, id int64) error {
	s.deleted = append(s.deleted, id)
	return nil
}

func setupAccountHealthRouter(admin *healthAdminStub) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewAccountHandler(admin, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router.POST("/api/v1/admin/accounts/health/scan", handler.ScanAccountHealth)
	router.POST("/api/v1/admin/accounts/health/cleanup", handler.CleanupAccountHealth)
	router.POST("/api/v1/admin/accounts/health/add", handler.AddAccountHealth)
	router.POST("/api/v1/admin/accounts/health/assistant", handler.AccountHealthAssistant)
	return router
}

func newHealthAdmin(accounts []service.Account) *healthAdminStub {
	stub := newStubAdminService()
	stub.accounts = accounts
	return &healthAdminStub{stubAdminService: stub}
}

func TestAccountHealthHandlerScanWithoutTest(t *testing.T) {
	expiredAt := time.Now().Add(-time.Hour)
	admin := newHealthAdmin([]service.Account{
		{ID: 1, Name: "broken", Status: service.StatusError, ErrorMessage: "token_expired"},
		{ID: 2, Name: "expired", Status: service.StatusActive, AutoPauseOnExpired: true, ExpiresAt: &expiredAt},
		{ID: 3, Name: "ok", Status: service.StatusActive},
	})
	router := setupAccountHealthRouter(admin)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/health/scan", bytes.NewBufferString(`{"test":false}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var payload struct {
		Data service.AccountHealthScanResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Equal(t, 2, payload.Data.CleanupCount)
	require.Equal(t, 0, payload.Data.Tested)
	require.NotContains(t, rec.Body.String(), "token_expired")
}

func TestAccountHealthHandlerCleanupRequiresConfirm(t *testing.T) {
	admin := newHealthAdmin([]service.Account{{ID: 8, Name: "x", Status: service.StatusError}})
	router := setupAccountHealthRouter(admin)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/health/cleanup", bytes.NewBufferString(`{"account_ids":[8]}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Empty(t, admin.deleted)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/health/cleanup", bytes.NewBufferString(`{"account_ids":[8],"confirm":"DELETE"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, []int64{8}, admin.deleted)
}

func TestAccountHealthHandlerChatCannotDeleteWithoutPendingIDs(t *testing.T) {
	admin := newHealthAdmin([]service.Account{{ID: 15, Name: "target", Status: service.StatusError}})
	router := setupAccountHealthRouter(admin)

	body := `{"messages":[{"role":"user","content":"确认删除 15"}],"confirm":"DELETE"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/health/assistant", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Empty(t, admin.deleted)

	var payload struct {
		Data service.AccountHealthAssistantResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Nil(t, payload.Data.Cleanup)
}

func TestAccountHealthHandlerChatDeletesOnlyPendingIDs(t *testing.T) {
	admin := newHealthAdmin([]service.Account{{ID: 15, Name: "target", Status: service.StatusError}})
	router := setupAccountHealthRouter(admin)

	body := `{"messages":[{"role":"user","content":"确认删除 99"}],"pending_proposal":{"account_ids":[15]},"confirm":"DELETE"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/health/assistant", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Empty(t, admin.deleted)

	var payload struct {
		Data service.AccountHealthAssistantResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Nil(t, payload.Data.Cleanup)
}

func TestAccountHealthHandlerAddRequiresConfirm(t *testing.T) {
	admin := newHealthAdmin(nil)
	router := setupAccountHealthRouter(admin)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/health/add", bytes.NewBufferString(`{"handoff":"sk-secret-token-aaaa","indexes":[0]}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Empty(t, admin.createdAccounts)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/health/add", bytes.NewBufferString(`{"handoff":"sk-secret-token-aaaa","indexes":[0],"confirm":"ADD"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, admin.createdAccounts, 1)
	require.NotContains(t, rec.Body.String(), "sk-secret-token-aaaa")
	require.Contains(t, rec.Body.String(), "••••")
}

func TestAccountHealthHandlerChatHandoffDoesNotCreate(t *testing.T) {
	admin := newHealthAdmin(nil)
	router := setupAccountHealthRouter(admin)

	body := `{"handoff":"sk-secret-token-bbbb"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/health/assistant", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Empty(t, admin.createdAccounts)

	var payload struct {
		Data service.AccountHealthAssistantResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Equal(t, service.AccountHealthIntentAdd, payload.Data.Intent)
	require.NotNil(t, payload.Data.AddProposal)
	require.Len(t, payload.Data.AddProposal.Items, 1)
	require.NotContains(t, rec.Body.String(), "sk-secret-token-bbbb")
}
