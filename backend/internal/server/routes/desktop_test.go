package routes

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/modules/desktop"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

const routesTestDevicePublicID = "c7d06b34-7c6a-4a59-ad48-648ade0525e1"

func TestRegisterDesktopRoutesExposesDeviceManagementContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")
	pass := func(c *gin.Context) { c.Next() }
	RegisterDesktopRoutes(
		v1,
		&handler.Handlers{Desktop: handler.NewDesktopHandler(nil)},
		middleware.JWTAuthMiddleware(pass),
		middleware.DesktopAuthMiddleware(pass),
		middleware.AuditLogMiddleware(pass),
		nil,
		nil,
	)

	registered := make(map[string]struct{})
	for _, route := range router.Routes() {
		registered[route.Method+" "+route.Path] = struct{}{}
	}
	for _, route := range []string{
		"GET /api/v1/public/desktop/releases/macos/latest",
		"GET /api/v1/public/desktop/releases/macos/latest/download",
		"GET /api/v1/desktop/devices",
		"PATCH /api/v1/desktop/devices/:device_id",
		"DELETE /api/v1/desktop/devices/:device_id",
		"POST /api/v1/desktop/me/heartbeat",
		"POST /api/v1/desktop/me/activate",
		"DELETE /api/v1/desktop/me",
		"GET /api/v1/desktop/usage/today",
		"GET /api/v1/desktop/releases/:target/:arch/:current_version",
		"POST /api/v1/desktop/diagnostics",
	} {
		_, ok := registered[route]
		require.Truef(t, ok, "missing route %s", route)
	}
}

func TestDesktopPairingCreateIsRateLimited(t *testing.T) {
	gin.SetMode(gin.TestMode)
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { require.NoError(t, redisClient.Close()) })

	router := gin.New()
	v1 := router.Group("/api/v1")
	pass := func(c *gin.Context) { c.Next() }
	RegisterDesktopRoutes(
		v1,
		&handler.Handlers{Desktop: handler.NewDesktopHandler(nil)},
		middleware.JWTAuthMiddleware(pass),
		middleware.DesktopAuthMiddleware(pass),
		middleware.AuditLogMiddleware(pass),
		nil,
		redisClient,
	)

	for range 10 {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/v1/desktop/pairings", bytes.NewBufferString("{}"))
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, request)
		require.Equal(t, http.StatusBadRequest, recorder.Code)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/desktop/pairings", bytes.NewBufferString("{}"))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusTooManyRequests, recorder.Code)
}

func TestDesktopPairingCreateRejectsOversizedBodyBeforeService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { require.NoError(t, redisClient.Close()) })

	router := gin.New()
	v1 := router.Group("/api/v1")
	pass := func(c *gin.Context) { c.Next() }
	RegisterDesktopRoutes(
		v1,
		&handler.Handlers{Desktop: handler.NewDesktopHandler(nil)},
		middleware.JWTAuthMiddleware(pass),
		middleware.DesktopAuthMiddleware(pass),
		middleware.AuditLogMiddleware(pass),
		nil,
		redisClient,
	)

	payload, err := json.Marshal(map[string]any{
		"code_challenge":        "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"code_challenge_method": "S256",
		"installation_id":       "installation-id",
		"device_name":           string(bytes.Repeat([]byte("x"), int(desktopControlRequestBodyLimit))),
		"platform":              "macos",
		"architecture":          "arm64",
	})
	require.NoError(t, err)
	require.Greater(t, int64(len(payload)), desktopControlRequestBodyLimit)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/desktop/pairings", bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestDesktopPairingCreateFailsClosedWhenRedisIsUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	redisClient := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	t.Cleanup(func() { require.NoError(t, redisClient.Close()) })

	router := gin.New()
	v1 := router.Group("/api/v1")
	pass := func(c *gin.Context) { c.Next() }
	RegisterDesktopRoutes(
		v1,
		&handler.Handlers{Desktop: handler.NewDesktopHandler(nil)},
		middleware.JWTAuthMiddleware(pass),
		middleware.DesktopAuthMiddleware(pass),
		middleware.AuditLogMiddleware(pass),
		nil,
		redisClient,
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/desktop/pairings", bytes.NewBufferString("{}"))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	require.JSONEq(
		t,
		`{"code":503,"message":"Service Unavailable","reason":"DESKTOP_SERVICE_UNAVAILABLE"}`,
		recorder.Body.String(),
	)
}

func TestRegisterDesktopDiagnosticAdminRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerDesktopDiagnosticRoutes(router.Group("/api/v1/admin"), &handler.Handlers{
		Admin: &handler.AdminHandlers{DesktopDiagnostic: adminhandler.NewDesktopDiagnosticHandler(nil)},
	})

	registered := make(map[string]struct{})
	for _, route := range router.Routes() {
		registered[route.Method+" "+route.Path] = struct{}{}
	}
	for _, route := range []string{
		"GET /api/v1/admin/desktop/diagnostics",
		"GET /api/v1/admin/desktop/diagnostics/:id",
		"GET /api/v1/admin/desktop/diagnostics/:id/download",
	} {
		_, ok := registered[route]
		require.Truef(t, ok, "missing route %s", route)
	}
}

func TestDesktopReleaseRouteRequiresDeviceTokenAndPublicDownloadStaysStable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")
	cfg := &config.Config{}
	cfg.JWT.Secret = "route-desktop-release-secret"
	cfg.Desktop.Releases = []config.DesktopReleaseConfig{
		{Version: "9.0.0", Channel: desktop.ReleaseChannelInternal, Target: desktop.ReleaseTargetDarwin, Arch: desktop.ReleaseArchUniversal, Enabled: true, URL: "https://downloads.example.com/internal.tar.gz", InstallerURL: "https://downloads.example.com/internal.dmg", Signature: "internal", PubDate: "2026-07-28T00:00:00Z"},
		{Version: "0.2.0", Channel: desktop.ReleaseChannelStable, Target: desktop.ReleaseTargetDarwin, Arch: desktop.ReleaseArchUniversal, Enabled: true, URL: "https://downloads.example.com/stable.tar.gz", InstallerURL: "https://downloads.example.com/stable.dmg", Signature: "stable", PubDate: "2026-07-28T00:00:00Z"},
	}
	svc := desktop.NewService(nil, &approvalRepository{}, nil, nil, cfg)
	pass := func(c *gin.Context) { c.Next() }
	RegisterDesktopRoutes(
		v1,
		&handler.Handlers{Desktop: handler.NewDesktopHandler(svc)},
		middleware.JWTAuthMiddleware(pass),
		middleware.NewDesktopAuthMiddleware(svc),
		middleware.AuditLogMiddleware(pass),
		nil,
		nil,
	)

	unauthenticated := httptest.NewRecorder()
	router.ServeHTTP(unauthenticated, httptest.NewRequest(http.MethodGet, "/api/v1/desktop/releases/darwin/universal/0.1.0", nil))
	require.Equal(t, http.StatusUnauthorized, unauthenticated.Code)

	now := time.Now()
	claims := jwt.MapClaims{
		"iss": desktop.DesktopIssuer, "aud": []string{desktop.DesktopAudience},
		"sub": routesTestDevicePublicID, "exp": now.Add(time.Minute).Unix(),
		"iat": now.Unix(), "nbf": now.Unix(), "jti": "release-route-test",
		"device_id": int64(41), "user_id": int64(7), "token_version": int64(1),
		"scope": []string{desktop.ScopeProfileRead},
	}
	key := sha256.Sum256([]byte("luoxue-desktop-v1\x00" + cfg.JWT.Secret))
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(key[:])
	require.NoError(t, err)

	authenticated := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/desktop/releases/darwin/universal/0.1.0", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(authenticated, request)
	require.Equal(t, http.StatusOK, authenticated.Code)
	require.Contains(t, authenticated.Body.String(), `"version":"0.2.0"`)
	require.NotContains(t, authenticated.Body.String(), "internal")

	publicDownload := httptest.NewRecorder()
	router.ServeHTTP(publicDownload, httptest.NewRequest(http.MethodGet, "/api/v1/public/desktop/releases/macos/latest/download", nil))
	require.Equal(t, http.StatusTemporaryRedirect, publicDownload.Code)
	require.Equal(t, "https://downloads.example.com/stable.dmg", publicDownload.Header().Get("Location"))
}

func TestDesktopPairingApprovalDoesNotRequireTOTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")
	jwt := func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 7})
		c.Next()
	}
	pass := func(c *gin.Context) { c.Next() }
	svc := desktop.NewService(
		&approvalPairingStore{pairing: desktop.Pairing{
			UserCode: "ABCD-EFGH", DeviceName: "Mac", Platform: "macos", Architecture: "arm64",
			Status: desktop.PairingStatusPending, ExpiresAt: time.Now().Add(time.Minute),
		}},
		&approvalRepository{}, nil, nil, &config.Config{},
	)
	RegisterDesktopRoutes(
		v1,
		&handler.Handlers{Desktop: handler.NewDesktopHandler(svc)},
		middleware.JWTAuthMiddleware(jwt),
		middleware.DesktopAuthMiddleware(pass),
		middleware.AuditLogMiddleware(pass),
		nil,
		nil,
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/desktop/authorizations/ABCD-EFGH/approve", nil)
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"id":"c7d06b34-7c6a-4a59-ad48-648ade0525e1"`)
}

func TestDesktopTodayUsageRouteAcceptsDesktopAccessToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")
	cfg := &config.Config{}
	cfg.JWT.Secret = "route-desktop-secret"
	svc := desktop.NewService(nil, &approvalRepository{}, nil, nil, cfg)
	pass := func(c *gin.Context) { c.Next() }
	RegisterDesktopRoutes(
		v1,
		&handler.Handlers{Desktop: handler.NewDesktopHandler(svc)},
		middleware.JWTAuthMiddleware(pass),
		middleware.NewDesktopAuthMiddleware(svc),
		middleware.AuditLogMiddleware(pass),
		nil,
		nil,
	)

	now := time.Now()
	claims := jwt.MapClaims{
		"iss": desktop.DesktopIssuer, "aud": []string{desktop.DesktopAudience},
		"sub": routesTestDevicePublicID, "exp": now.Add(time.Minute).Unix(),
		"iat": now.Unix(), "nbf": now.Unix(), "jti": "route-test",
		"device_id": int64(41), "user_id": int64(7), "token_version": int64(1),
		"scope": []string{desktop.ScopeProfileRead},
	}
	key := sha256.Sum256([]byte("luoxue-desktop-v1\x00" + cfg.JWT.Secret))
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(key[:])
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/desktop/usage/today", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"requests":4`)
	require.Contains(t, recorder.Body.String(), `"tokens":1500`)
	require.Contains(t, recorder.Body.String(), `"cost":1.25`)
	require.Contains(t, recorder.Body.String(), `"balance":9.75`)
}

type approvalPairingStore struct {
	pairing desktop.Pairing
}

func (s *approvalPairingStore) Create(context.Context, desktop.Pairing, time.Duration) error {
	return nil
}
func (s *approvalPairingStore) GetByUserCode(context.Context, string) (*desktop.Pairing, error) {
	out := s.pairing
	return &out, nil
}
func (s *approvalPairingStore) GetByDeviceCode(context.Context, string) (*desktop.Pairing, error) {
	out := s.pairing
	return &out, nil
}
func (s *approvalPairingStore) BeginApproval(context.Context, string) (*desktop.Pairing, error) {
	out := s.pairing
	out.Status = desktop.PairingStatusApproving
	return &out, nil
}
func (s *approvalPairingStore) FinishApproval(context.Context, string, int64, int64) error {
	return nil
}
func (s *approvalPairingStore) CancelApproval(context.Context, string) error { return nil }
func (s *approvalPairingStore) Consume(context.Context, string) (*desktop.Pairing, error) {
	return nil, desktop.ErrPairingState
}
func (s *approvalPairingStore) RestoreConsumed(context.Context, string) error { return nil }

type approvalRepository struct{}

func (r *approvalRepository) ReserveDevice(_ context.Context, userID int64, pairing desktop.Pairing) (*desktop.Device, error) {
	return &desktop.Device{
		ID: 41, PublicID: "c7d06b34-7c6a-4a59-ad48-648ade0525e1", UserID: userID,
		Name: pairing.DeviceName, Platform: "macos", Architecture: pairing.Architecture,
		Status: desktop.DeviceStatusPending, TokenVersion: 1,
	}, nil
}
func (r *approvalRepository) DeletePendingDevice(context.Context, int64) error { return nil }
func (r *approvalRepository) ActivateDevice(context.Context, int64, string, string, time.Time) (*desktop.Device, error) {
	return nil, desktop.ErrDeviceUnauthorized
}
func (r *approvalRepository) RotateSession(context.Context, string, string, time.Time) (*desktop.Device, error) {
	return nil, desktop.ErrRefreshToken
}
func (r *approvalRepository) GetAuthorizedDevice(context.Context, int64, int64, int64) (*desktop.Device, error) {
	return &desktop.Device{
		ID: 41, PublicID: routesTestDevicePublicID, UserID: 7, Name: "Mac",
		Platform: "macos", Architecture: "arm64", Status: desktop.DeviceStatusActive, TokenVersion: 1,
	}, nil
}
func (r *approvalRepository) ListDevices(context.Context, int64) ([]desktop.Device, error) {
	return []desktop.Device{}, nil
}
func (r *approvalRepository) RenameDevice(context.Context, int64, string, string) (*desktop.Device, error) {
	return nil, desktop.ErrDeviceNotFound
}
func (r *approvalRepository) HeartbeatDevice(context.Context, int64, int64, int64) (*desktop.Device, error) {
	return nil, desktop.ErrDeviceUnauthorized
}
func (r *approvalRepository) MarkDeviceActivated(context.Context, int64, int64, int64, time.Time) (*desktop.Device, error) {
	return nil, desktop.ErrDeviceUnauthorized
}
func (r *approvalRepository) RevokeDevice(context.Context, int64, string) (*desktop.DeviceRevocation, error) {
	return nil, desktop.ErrDeviceNotFound
}
func (r *approvalRepository) EnsureManagedKey(context.Context, int64, int64, int64, string) (*desktop.ManagedKey, error) {
	return nil, desktop.ErrDeviceUnauthorized
}
func (r *approvalRepository) GetTodayUsage(context.Context, int64, int64, time.Time, time.Time) (*desktop.TodayUsage, error) {
	return &desktop.TodayUsage{Requests: 4, Tokens: 1500, Cost: 1.25, Balance: 9.75}, nil
}
func (r *approvalRepository) GetReleaseChannel(context.Context, int64, int64, int64) (string, error) {
	return desktop.ReleaseChannelStable, nil
}
