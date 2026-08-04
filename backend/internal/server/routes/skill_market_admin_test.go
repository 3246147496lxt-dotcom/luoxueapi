package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newSkillMarketRouteTestRouter(stepUpHits *int) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	admin := router.Group("/api/v1/admin")
	handlers := &handler.Handlers{
		Admin: &handler.AdminHandlers{
			SkillMarket: adminhandler.NewSkillMarketHandler(
				service.NewSkillMarketService(nil, nil),
			),
		},
	}
	stepUp := middleware.StepUpAuthMiddleware(func(c *gin.Context) {
		(*stepUpHits)++
		c.AbortWithStatus(http.StatusTeapot)
	})
	registerSkillMarketRoutes(admin, handlers, stepUp)
	return router
}

func TestSkillMarketplaceConfigUpdateDoesNotRequireStepUp(t *testing.T) {
	stepUpHits := 0
	router := newSkillMarketRouteTestRouter(&stepUpHits)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/skills/config", strings.NewReader(`{"enabled":true}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Zero(t, stepUpHits)
	require.NotEqual(t, http.StatusTeapot, rec.Code)
}

func TestSkillMarketplaceReleaseLifecycleStillRequiresStepUp(t *testing.T) {
	tests := []struct {
		method string
		path   string
	}{
		{http.MethodPut, "/api/v1/admin/skills/1"},
		{http.MethodPost, "/api/v1/admin/skills/1/publish"},
		{http.MethodPost, "/api/v1/admin/skills/1/versions/2/activate"},
		{http.MethodPost, "/api/v1/admin/skills/1/versions/2/yank"},
		{http.MethodPost, "/api/v1/admin/skills/1/archive"},
	}

	for _, test := range tests {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			stepUpHits := 0
			router := newSkillMarketRouteTestRouter(&stepUpHits)
			req := httptest.NewRequest(test.method, test.path, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusTeapot, rec.Code)
			require.Equal(t, 1, stepUpHits)
		})
	}
}
