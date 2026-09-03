package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/modules/skillimport/application"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	core "github.com/Wei-Shaw/sub2api/internal/skillimport"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newSkillImportAdminHandlers() *handler.Handlers {
	importService := application.NewService(
		nil,
		application.NewAdapterRegistry(core.NewManifestAdapter(nil)),
		&config.Config{SkillImport: config.SkillImportConfig{Enabled: true}},
	)
	return &handler.Handlers{Admin: &handler.AdminHandlers{
		SkillImport: adminhandler.NewSkillImportHandler(importService),
	}}
}

func TestSkillImportAdminRoutesAreRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")
	passThrough := func(c *gin.Context) { c.Next() }
	RegisterAdminRoutes(
		v1,
		newSkillImportAdminHandlers(),
		middleware.AdminAuthMiddleware(passThrough),
		middleware.AuditLogMiddleware(passThrough),
		middleware.StepUpAuthMiddleware(passThrough),
		nil,
	)

	registered := make(map[string]struct{})
	for _, route := range router.Routes() {
		registered[route.Method+" "+route.Path] = struct{}{}
	}
	for _, expected := range []string{
		"GET /api/v1/admin/skill-import/adapters",
		"GET /api/v1/admin/skill-import/sources",
		"POST /api/v1/admin/skill-import/sources",
		"GET /api/v1/admin/skill-import/sources/:id",
		"PUT /api/v1/admin/skill-import/sources/:id",
		"GET /api/v1/admin/skill-import/schedules",
		"POST /api/v1/admin/skill-import/schedules",
		"GET /api/v1/admin/skill-import/schedules/:id",
		"PUT /api/v1/admin/skill-import/schedules/:id",
		"POST /api/v1/admin/skill-import/schedules/:id/run",
		"GET /api/v1/admin/skill-import/runs",
		"POST /api/v1/admin/skill-import/runs",
		"POST /api/v1/admin/skill-import/runs/upload",
		"GET /api/v1/admin/skill-import/runs/:id",
		"GET /api/v1/admin/skill-import/runs/:id/items",
		"GET /api/v1/admin/skill-import/runs/:id/events",
		"POST /api/v1/admin/skill-import/runs/:id/cancel",
		"POST /api/v1/admin/skill-import/runs/:id/retry-failed",
		"POST /api/v1/admin/skill-import/runs/:id/publish",
	} {
		_, ok := registered[expected]
		require.True(t, ok, "missing route %s", expected)
	}
}

func TestSkillImportAdminRoutesRunBehindAuthAndAudit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("authentication aborts before audit and handler", func(t *testing.T) {
		authCalls, auditCalls := 0, 0
		router := gin.New()
		v1 := router.Group("/api/v1")
		RegisterAdminRoutes(
			v1,
			newSkillImportAdminHandlers(),
			middleware.AdminAuthMiddleware(func(c *gin.Context) {
				authCalls++
				c.AbortWithStatus(http.StatusUnauthorized)
			}),
			middleware.AuditLogMiddleware(func(c *gin.Context) {
				auditCalls++
				c.Next()
			}),
			middleware.StepUpAuthMiddleware(func(c *gin.Context) { c.Next() }),
			nil,
		)

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/skill-import/adapters", nil)
		router.ServeHTTP(recorder, request)

		require.Equal(t, http.StatusUnauthorized, recorder.Code)
		require.Equal(t, 1, authCalls)
		require.Zero(t, auditCalls)
	})

	t.Run("authenticated request is audited", func(t *testing.T) {
		authCalls, auditCalls := 0, 0
		router := gin.New()
		v1 := router.Group("/api/v1")
		RegisterAdminRoutes(
			v1,
			newSkillImportAdminHandlers(),
			middleware.AdminAuthMiddleware(func(c *gin.Context) {
				authCalls++
				c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 42})
				c.Next()
			}),
			middleware.AuditLogMiddleware(func(c *gin.Context) {
				auditCalls++
				c.Next()
			}),
			middleware.StepUpAuthMiddleware(func(c *gin.Context) { c.Next() }),
			nil,
		)

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/skill-import/adapters", nil)
		router.ServeHTTP(recorder, request)

		require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
		require.Equal(t, 1, authCalls)
		require.Equal(t, 1, auditCalls)
		require.Contains(t, recorder.Body.String(), `"type":"manifest"`)
	})
}
