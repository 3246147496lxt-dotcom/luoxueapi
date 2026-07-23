package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type embeddedPageRouteSettingsRepo struct {
	values map[string]string
}

func (r *embeddedPageRouteSettingsRepo) Get(context.Context, string) (*service.Setting, error) {
	return nil, service.ErrSettingNotFound
}
func (r *embeddedPageRouteSettingsRepo) GetValue(_ context.Context, key string) (string, error) {
	return r.values[key], nil
}
func (r *embeddedPageRouteSettingsRepo) Set(context.Context, string, string) error { return nil }
func (r *embeddedPageRouteSettingsRepo) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	values := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := r.values[key]; ok {
			values[key] = value
		}
	}
	return values, nil
}
func (r *embeddedPageRouteSettingsRepo) SetMultiple(context.Context, map[string]string) error {
	return nil
}
func (r *embeddedPageRouteSettingsRepo) GetAll(context.Context) (map[string]string, error) {
	return r.values, nil
}
func (r *embeddedPageRouteSettingsRepo) Delete(context.Context, string) error { return nil }

func newEmbeddedPageRouteTestRouter(t *testing.T) (*gin.Engine, *miniredis.Miniredis) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	settings := service.NewSettingService(&embeddedPageRouteSettingsRepo{values: map[string]string{
		service.SettingKeyCustomMenuItems: `[{"id":"payment","url":"https://pay.example.com/start","visibility":"user","auth_mode":"exchange_code"}]`,
	}}, &config.Config{})

	router := gin.New()
	v1 := router.Group("/api/v1")
	jwtAuth := servermiddleware.JWTAuthMiddleware(func(c *gin.Context) {
		c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: 42})
		c.Set(string(servermiddleware.ContextKeyUserRole), "user")
		c.Next()
	})
	audit := servermiddleware.AuditLogMiddleware(func(c *gin.Context) { c.Next() })
	RegisterEmbeddedPageRoutes(v1, jwtAuth, audit, settings, rdb)
	return router, mr
}

func TestRegisterEmbeddedPageRoutesLaunchAndExchange(t *testing.T) {
	router, _ := newEmbeddedPageRouteTestRouter(t)

	launchReq := httptest.NewRequest(http.MethodPost, "/api/v1/user/custom-pages/payment/launch", bytes.NewBufferString(`{"theme":"dark","lang":"zh-CN"}`))
	launchReq.Header.Set("Content-Type", "application/json")
	launchReq.Host = "api.example.com"
	launchRec := httptest.NewRecorder()
	router.ServeHTTP(launchRec, launchReq)
	require.Equal(t, http.StatusOK, launchRec.Code, launchRec.Body.String())
	require.Equal(t, "no-store", launchRec.Header().Get("Cache-Control"))

	var launchEnvelope response.Response
	require.NoError(t, json.Unmarshal(launchRec.Body.Bytes(), &launchEnvelope))
	launchData, ok := launchEnvelope.Data.(map[string]any)
	require.True(t, ok)
	launchURLValue, ok := launchData["launch_url"].(string)
	require.True(t, ok)
	launchURL, err := url.Parse(launchURLValue)
	require.NoError(t, err)
	// Identity is intentionally absent from the URL.
	require.Empty(t, launchURL.Query().Get("user_id"))
	require.Empty(t, launchURL.Query().Get("token"))
	code := launchURL.Query().Get("s2a_launch_code")
	require.NotEmpty(t, code)

	exchangeBody, err := json.Marshal(map[string]string{"client_id": "payment", "code": code})
	require.NoError(t, err)
	exchangeReq := httptest.NewRequest(http.MethodPost, "/api/v1/embedded-pages/exchange", bytes.NewReader(exchangeBody))
	exchangeReq.Header.Set("Content-Type", "application/json")
	exchangeRec := httptest.NewRecorder()
	router.ServeHTTP(exchangeRec, exchangeReq)
	require.Equal(t, http.StatusOK, exchangeRec.Code, exchangeRec.Body.String())
	require.Equal(t, "no-store", exchangeRec.Header().Get("Cache-Control"))

	var exchangeEnvelope response.Response
	require.NoError(t, json.Unmarshal(exchangeRec.Body.Bytes(), &exchangeEnvelope))
	exchangeData, ok := exchangeEnvelope.Data.(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(42), exchangeData["user_id"])
	require.Equal(t, "payment", exchangeData["menu_item_id"])

	replayReq := httptest.NewRequest(http.MethodPost, "/api/v1/embedded-pages/exchange", bytes.NewReader(exchangeBody))
	replayReq.Header.Set("Content-Type", "application/json")
	replayRec := httptest.NewRecorder()
	router.ServeHTTP(replayRec, replayReq)
	require.Equal(t, http.StatusUnauthorized, replayRec.Code)
	require.Contains(t, replayRec.Body.String(), "INVALID_EMBED_LAUNCH_CODE")
}

func TestEmbeddedPageExchangeRateLimitBackendFailureReturns503(t *testing.T) {
	router, mr := newEmbeddedPageRouteTestRouter(t)
	mr.Close()

	exchangeReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/embedded-pages/exchange",
		bytes.NewBufferString(`{"client_id":"payment","code":"invalid"}`),
	)
	exchangeReq.Header.Set("Content-Type", "application/json")
	exchangeRec := httptest.NewRecorder()
	router.ServeHTTP(exchangeRec, exchangeReq)

	require.Equal(t, http.StatusServiceUnavailable, exchangeRec.Code, exchangeRec.Body.String())
	var envelope response.Response
	require.NoError(t, json.Unmarshal(exchangeRec.Body.Bytes(), &envelope))
	require.Equal(t, http.StatusServiceUnavailable, envelope.Code)
	require.Equal(t, "EMBEDDED_PAGE_LAUNCH_UNAVAILABLE", envelope.Reason)
}

func TestEmbeddedPageExchangeActualLimitExceededRemains429(t *testing.T) {
	router, _ := newEmbeddedPageRouteTestRouter(t)
	validShapedCode := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	body, err := json.Marshal(map[string]string{"client_id": "payment", "code": validShapedCode})
	require.NoError(t, err)

	for requestNumber := 1; requestNumber <= 31; requestNumber++ {
		exchangeReq := httptest.NewRequest(http.MethodPost, "/api/v1/embedded-pages/exchange", bytes.NewReader(body))
		exchangeReq.Header.Set("Content-Type", "application/json")
		exchangeReq.RemoteAddr = "192.0.2.10:1234"
		exchangeRec := httptest.NewRecorder()
		router.ServeHTTP(exchangeRec, exchangeReq)

		if requestNumber <= 30 {
			require.Equal(t, http.StatusUnauthorized, exchangeRec.Code, "request %d: %s", requestNumber, exchangeRec.Body.String())
			continue
		}
		require.Equal(t, http.StatusTooManyRequests, exchangeRec.Code, exchangeRec.Body.String())
		require.Contains(t, exchangeRec.Body.String(), "rate limit exceeded")
		require.NotContains(t, exchangeRec.Body.String(), "EMBEDDED_PAGE_LAUNCH_UNAVAILABLE")
	}
}
