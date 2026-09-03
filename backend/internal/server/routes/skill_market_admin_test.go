package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newSkillMarketRouteTestRouter() *gin.Engine {
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
	registerSkillMarketRoutes(admin, handlers)
	return router
}

func TestSkillMarketplaceConfigUpdateReachesHandlerWithoutStepUp(t *testing.T) {
	router := newSkillMarketRouteTestRouter()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/skills/config", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSkillMarketplaceLifecycleMutationsReachHandlersWithoutStepUp(t *testing.T) {
	tests := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPut, "/api/v1/admin/skills/0", `{}`},
		{http.MethodPost, "/api/v1/admin/skills/1/publish", `{}`},
		{http.MethodPost, "/api/v1/admin/skills/1/versions/0/activate", ""},
		{http.MethodPost, "/api/v1/admin/skills/1/versions/0/yank", ""},
		{http.MethodPost, "/api/v1/admin/skills/0/archive", ""},
	}

	for _, test := range tests {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			router := newSkillMarketRouteTestRouter()
			req := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			if test.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}
