package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestLibraryDownloadTicketRouteIsAnonymousButIssuanceRemainsAuthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtCalls := 0
	jwt := middleware.JWTAuthMiddleware(func(c *gin.Context) {
		jwtCalls++
		c.AbortWithStatusJSON(http.StatusTeapot, gin.H{"message": "jwt invoked"})
	})
	router := gin.New()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { require.NoError(t, rdb.Close()) })
	RegisterLibraryRoutes(
		router.Group("/api/v1"),
		&handler.Handlers{Library: handler.NewLibraryHandler(nil, nil)},
		jwt,
		nil,
		&config.Config{},
		rdb,
	)

	downloadRequest := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/library/download/AAAAAAAAAAAAAAAAAAAAAA",
		nil,
	)
	downloadRecorder := httptest.NewRecorder()
	router.ServeHTTP(downloadRecorder, downloadRequest)

	require.Equal(t, http.StatusServiceUnavailable, downloadRecorder.Code)
	require.Contains(t, downloadRecorder.Body.String(), "LIBRARY_DOWNLOAD_TICKET_UNAVAILABLE")
	require.Equal(t, "private, no-store", downloadRecorder.Header().Get("Cache-Control"))
	require.Equal(t, "no-referrer", downloadRecorder.Header().Get("Referrer-Policy"))
	require.Equal(t, "SAMEORIGIN", downloadRecorder.Header().Get("X-Frame-Options"))
	require.Zero(t, jwtCalls, "the one-time capability GET must not require a bearer token")

	issueRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/library/files/download-ticket",
		strings.NewReader(`{"file_ids":["file_alpha"]}`),
	)
	issueRequest.Header.Set("Content-Type", "application/json")
	issueRecorder := httptest.NewRecorder()
	router.ServeHTTP(issueRecorder, issueRequest)

	require.Equal(t, http.StatusTeapot, issueRecorder.Code)
	require.Equal(t, 1, jwtCalls, "ticket issuance must remain behind JWT authentication")
}

func TestLibraryDownloadTicketPublicRateLimiterFailsClosedWithoutRedis(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterLibraryRoutes(
		router.Group("/api/v1"),
		&handler.Handlers{Library: handler.NewLibraryHandler(nil, nil)},
		middleware.JWTAuthMiddleware(func(c *gin.Context) { c.Next() }),
		nil,
		&config.Config{},
		nil,
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/library/download/AAAAAAAAAAAAAAAAAAAAAA",
		nil,
	)
	recorder := httptest.NewRecorder()
	require.NotPanics(t, func() { router.ServeHTTP(recorder, request) })
	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	require.Contains(t, recorder.Body.String(), "LIBRARY_DOWNLOAD_TICKET_UNAVAILABLE")
	require.Equal(t, "private, no-store", recorder.Header().Get("Cache-Control"))
	require.Equal(t, "SAMEORIGIN", recorder.Header().Get("X-Frame-Options"))
}
