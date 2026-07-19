package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestDocumentationPublicGuardFailsClosed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name  string
		guard gin.HandlerFunc
	}{
		{name: "missing setting service", guard: documentationPublicGuard(nil)},
		{name: "backend mode enabled", guard: documentationBackendModeGuard(func(context.Context) bool { return true })},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			router := gin.New()
			router.GET("/documentation", tt.guard, func(c *gin.Context) {
				called = true
				c.Status(http.StatusNoContent)
			})

			request := httptest.NewRequest(http.MethodGet, "/documentation", nil)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			require.Equal(t, http.StatusNotFound, response.Code)
			require.Equal(t, "no-store", response.Header().Get("Cache-Control"))
			require.False(t, called)
		})
	}
}

func TestDocumentationBackendModeGuardAllowsPublicMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/documentation", documentationBackendModeGuard(func(context.Context) bool { return false }), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/documentation", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusNoContent, response.Code)
}

func TestRegisterDocumentationRoutesFailsClosedBeforeRateLimiterWhenSettingsMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")
	RegisterDocumentationRoutes(v1, &handler.Handlers{Documentation: &handler.DocumentationHandler{}}, nil, nil)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/public/documentation", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusNotFound, response.Code)
	require.Equal(t, "no-store", response.Header().Get("Cache-Control"))
}

type documentationRoutesSettingRepository struct {
	service.SettingRepository
}

func (documentationRoutesSettingRepository) GetValue(_ context.Context, key string) (string, error) {
	if key == service.SettingKeyBackendModeEnabled {
		return "false", nil
	}
	return "", service.ErrSettingNotFound
}

type documentationRoutesRepository struct {
	service.DocumentationRepository
	assetID string
}

func (documentationRoutesRepository) EnsureSeed(context.Context, json.RawMessage) error {
	return nil
}

func (documentationRoutesRepository) GetPublished(context.Context) (*service.DocumentationSnapshot, error) {
	return &service.DocumentationSnapshot{
		Content:   []byte(`{"schema_version":1,"tutorials":[]}`),
		Version:   1,
		UpdatedAt: time.Unix(1_700_000_000, 0).UTC(),
	}, nil
}

func (r documentationRoutesRepository) GetAsset(_ context.Context, id string) (*service.DocumentationAsset, error) {
	if id != r.assetID {
		return nil, service.ErrDocumentationNotFound
	}
	return &service.DocumentationAsset{
		ID:          id,
		ContentType: "image/png",
		ByteSize:    1,
		Width:       1,
		Height:      1,
		Data:        []byte{0},
	}, nil
}

func TestRegisterDocumentationRoutesAppliesIndependentAnonymousRateLimits(t *testing.T) {
	gin.SetMode(gin.TestMode)
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })

	assetID := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	documentationService := service.NewDocumentationService(documentationRoutesRepository{assetID: assetID})
	documentationHandler := handler.NewDocumentationHandler(documentationService)
	settingService := service.NewSettingService(documentationRoutesSettingRepository{}, nil)

	router := gin.New()
	v1 := router.Group("/api/v1")
	RegisterDocumentationRoutes(v1, &handler.Handlers{Documentation: documentationHandler}, redisClient, settingService)

	assertRateLimited := func(path string, limit int, remoteAddr string) {
		t.Helper()
		for requestNumber := 1; requestNumber <= limit+1; requestNumber++ {
			request := httptest.NewRequest(http.MethodGet, path, nil)
			request.RemoteAddr = remoteAddr
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if requestNumber <= limit {
				require.Equal(t, http.StatusOK, response.Code, "request %d path=%s", requestNumber, path)
				continue
			}
			require.Equal(t, http.StatusTooManyRequests, response.Code, "request %d path=%s", requestNumber, path)
		}
	}

	assertRateLimited("/api/v1/public/documentation", publicDocumentationRateLimit, "203.0.113.10:12345")
	assertRateLimited("/api/v1/public/documentation/assets/"+assetID, publicDocumentationAssetRateLimit, "203.0.113.10:12345")
}
