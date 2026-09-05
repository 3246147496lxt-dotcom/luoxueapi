package admin

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type deepSeekBalanceHandlerRepo struct {
	service.AccountRepository
	account *service.Account
}

func (r *deepSeekBalanceHandlerRepo) GetByID(context.Context, int64) (*service.Account, error) {
	return r.account, nil
}

func (r *deepSeekBalanceHandlerRepo) UpdateExtra(context.Context, int64, map[string]any) error {
	return nil
}

type deepSeekBalanceHandlerUpstream struct{}

func (deepSeekBalanceHandlerUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body: io.NopCloser(strings.NewReader(
			`{"is_available":true,"balance_infos":[{"currency":"USD","total_balance":"4.20"}]}`,
		)),
	}, nil
}

func (u deepSeekBalanceHandlerUpstream) DoWithTLS(req *http.Request, proxyURL string, accountID int64, concurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, proxyURL, accountID, concurrency)
}

func newDeepSeekBalanceHandler() (*AccountHandler, *service.DeepSeekBalanceService) {
	account := &service.Account{
		ID:       1,
		Platform: service.PlatformDeepseek,
		Type:     service.AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key": "sk-handler-test",
		},
	}
	repo := &deepSeekBalanceHandlerRepo{account: account}
	probe := service.NewDeepSeekBalanceService(repo, nil, deepSeekBalanceHandlerUpstream{}, &config.Config{})
	h := NewAccountHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	h.SetDeepSeekBalanceService(probe)
	return h, probe
}

func TestAccountHandlerQueryDeepSeekBalance(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _ := newDeepSeekBalanceHandler()
	router := gin.New()
	router.GET("/admin/accounts/:id/balance", h.QueryDeepSeekBalance)

	req := httptest.NewRequest(http.MethodGet, "/admin/accounts/1/balance", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"success":true`)
	require.Contains(t, rec.Body.String(), `"balance":4.2`)
}

func TestAccountHandlerQueryDeepSeekBalanceRejectsInvalidID(t *testing.T) {
	h, _ := newDeepSeekBalanceHandler()
	router := gin.New()
	router.GET("/admin/accounts/:id/balance", h.QueryDeepSeekBalance)

	req := httptest.NewRequest(http.MethodGet, "/admin/accounts/nope/balance", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAccountHandlerQueryDeepSeekBalanceRequiresService(t *testing.T) {
	h := NewAccountHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router := gin.New()
	router.GET("/admin/accounts/:id/balance", h.QueryDeepSeekBalance)

	req := httptest.NewRequest(http.MethodGet, "/admin/accounts/1/balance", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}
