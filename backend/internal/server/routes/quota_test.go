//go:build unit

package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/modules/quotaauth"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

type quotaRoutesOverviewRepo struct {
	snapshot *service.QuotaOverviewSnapshot
	calls    int
}

func (r *quotaRoutesOverviewRepo) LoadQuotaOverviewSnapshot(
	context.Context,
	int64,
	string,
) (*service.QuotaOverviewSnapshot, error) {
	r.calls++
	return r.snapshot, nil
}

type quotaRoutesConsistencyChecker struct{}

func (quotaRoutesConsistencyChecker) CheckQuotaOverviewConsistency(
	context.Context,
	*service.QuotaOverviewSnapshot,
) (service.QuotaOverviewConsistency, error) {
	return service.QuotaOverviewConsistency{Consistent: true}, nil
}

func newQuotaRoutesModeTestRouter(
	t *testing.T,
	runMode string,
) (*gin.Engine, *quotaRoutesOverviewRepo) {
	t.Helper()

	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { require.NoError(t, redisClient.Close()) })

	asOf := time.Date(2026, 7, 29, 7, 30, 0, 0, time.UTC)
	repo := &quotaRoutesOverviewRepo{snapshot: &service.QuotaOverviewSnapshot{
		AsOf: asOf,
		Account: service.QuotaOverviewAccountSnapshot{
			ID:      42,
			Email:   "quota@example.com",
			Status:  service.StatusActive,
			Balance: decimal.NewFromInt(10),
		},
	}}
	// Keep the service in standard mode so this test independently proves the
	// route guard, rather than relying on the service's defense in depth.
	overviewService := service.NewQuotaOverviewService(
		repo,
		quotaRoutesConsistencyChecker{},
		&config.Config{RunMode: config.RunModeStandard},
	)

	pass := func(c *gin.Context) { c.Next() }
	quotaAuth := func(c *gin.Context) {
		c.Set("quota_auth_subject", quotaauth.Subject{
			UserID:   42,
			ClientID: quotaauth.ClientID,
			Scopes:   []string{quotaauth.ScopeRead},
		})
		c.Next()
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterQuotaRoutes(
		router.Group("/api/v1"),
		&handler.Handlers{
			QuotaAuth:     handler.NewQuotaAuthHandler(nil),
			QuotaOverview: handler.NewQuotaOverviewHandler(overviewService),
		},
		middleware.JWTAuthMiddleware(pass),
		middleware.QuotaAuthMiddleware(quotaAuth),
		middleware.AuditLogMiddleware(pass),
		nil,
		&config.Config{RunMode: runMode},
		redisClient,
	)
	return router, repo
}

func TestQuotaOverviewRouteFailsClosedInSimpleMode(t *testing.T) {
	router, repo := newQuotaRoutesModeTestRouter(t, config.RunModeSimple)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/quota/overview", nil)
	request.Header.Set("Authorization", "Bearer quota-viewer-token")

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	require.Equal(t, "private, no-store", recorder.Header().Get("Cache-Control"))
	require.Equal(t, "Authorization", recorder.Header().Get("Vary"))
	require.JSONEq(t, `{
		"code": 503,
		"message": "Unable to generate a safe quota snapshot. Please retry later.",
		"reason": "QUOTA_OVERVIEW_UNAVAILABLE"
	}`, recorder.Body.String())
	require.Zero(t, repo.calls, "simple mode must not invoke the overview read model")
}

func TestQuotaOverviewRouteRemainsAvailableInStandardMode(t *testing.T) {
	router, repo := newQuotaRoutesModeTestRouter(t, config.RunModeStandard)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/quota/overview", nil)
	request.Header.Set("Authorization", "Bearer quota-viewer-token")

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Equal(t, "private, no-store", recorder.Header().Get("Cache-Control"))
	require.Equal(t, "Authorization", recorder.Header().Get("Vary"))
	require.Equal(t, 1, repo.calls)
}

func TestQuotaViewerInstallerDownloadIssueRouteRequiresJWT(t *testing.T) {
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { require.NoError(t, redisClient.Close()) })

	jwtCalls := 0
	jwtAuth := func(c *gin.Context) {
		jwtCalls++
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "authentication required"})
	}
	pass := func(c *gin.Context) { c.Next() }

	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterQuotaRoutes(
		router.Group("/api/v1"),
		&handler.Handlers{
			QuotaAuth:     handler.NewQuotaAuthHandler(nil),
			QuotaOverview: handler.NewQuotaOverviewHandler(nil),
		},
		middleware.JWTAuthMiddleware(jwtAuth),
		middleware.QuotaAuthMiddleware(pass),
		middleware.AuditLogMiddleware(pass),
		nil,
		&config.Config{RunMode: config.RunModeStandard},
		redisClient,
	)

	var routeExists bool
	for _, route := range router.Routes() {
		if route.Method == http.MethodPost && route.Path == "/api/v1/quota/releases/:platform/latest/download" {
			routeExists = true
			break
		}
	}
	require.True(t, routeExists)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/quota/releases/macos/latest/download",
		nil,
	)
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.Equal(t, 1, jwtCalls)
	require.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))
	require.Equal(t, "no-referrer", recorder.Header().Get("Referrer-Policy"))
}

func TestQuotaViewerInstallerDownloadRoutesIssueAuditAndRedirectOnce(t *testing.T) {
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { require.NoError(t, redisClient.Close()) })

	jwtAuth := func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 42})
		c.Set(string(middleware.ContextKeyUserRole), "user")
		c.Next()
	}
	var auditAction string
	audit := func(c *gin.Context) {
		c.Next()
		auditAction = c.GetString("audit_action")
	}
	pass := func(c *gin.Context) { c.Next() }

	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterQuotaRoutes(
		router.Group("/api/v1"),
		&handler.Handlers{
			QuotaAuth:     handler.NewQuotaAuthHandler(nil),
			QuotaOverview: handler.NewQuotaOverviewHandler(nil),
		},
		middleware.JWTAuthMiddleware(jwtAuth),
		middleware.QuotaAuthMiddleware(pass),
		middleware.AuditLogMiddleware(audit),
		nil,
		&config.Config{RunMode: config.RunModeStandard},
		redisClient,
	)

	issueRecorder := httptest.NewRecorder()
	issueRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/quota/releases/windows/latest/download",
		nil,
	)
	router.ServeHTTP(issueRecorder, issueRequest)
	require.Equal(t, http.StatusOK, issueRecorder.Code, issueRecorder.Body.String())
	require.Equal(t, handler.QuotaViewerInstallerDownloadAuditAction, auditAction)

	var issueResponse struct {
		Data service.QuotaViewerInstallerDownload `json:"data"`
	}
	require.NoError(t, json.Unmarshal(issueRecorder.Body.Bytes(), &issueResponse))
	require.Equal(t, "windows", issueResponse.Data.Platform)
	require.Equal(t, "2.0.0-rc.5", issueResponse.Data.Version)

	downloadRecorder := httptest.NewRecorder()
	downloadRequest := httptest.NewRequest(http.MethodGet, issueResponse.Data.DownloadPath, nil)
	router.ServeHTTP(downloadRecorder, downloadRequest)
	require.Equal(t, http.StatusTemporaryRedirect, downloadRecorder.Code)
	require.Equal(
		t,
		"https://github.com/3246147496lxt-dotcom/luoxueapi/releases/download/v2.0.0-rc.5/Luoxue-Quota-Viewer_2.0.0-rc.5_windows-x64_NSIS-UNSIGNED.exe",
		downloadRecorder.Header().Get("Location"),
	)

	replayRecorder := httptest.NewRecorder()
	replayRequest := httptest.NewRequest(http.MethodGet, issueResponse.Data.DownloadPath, nil)
	router.ServeHTTP(replayRecorder, replayRequest)
	require.Equal(t, http.StatusUnauthorized, replayRecorder.Code)
}
