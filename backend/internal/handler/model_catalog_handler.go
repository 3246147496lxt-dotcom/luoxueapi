package handler

import (
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// ModelCatalogHandler serves the anonymous, sanitized model marketplace.
type ModelCatalogHandler struct {
	catalogService *service.ModelCatalogService
	settingService *service.SettingService
}

const modelCatalogGuardPassedKey = "public_model_catalog_guard_passed"

func NewModelCatalogHandler(catalogService *service.ModelCatalogService, settingService *service.SettingService) *ModelCatalogHandler {
	return &ModelCatalogHandler{catalogService: catalogService, settingService: settingService}
}

// RequireEnabled runs before anonymous rate limiting so a disabled/private
// deployment always looks like a missing route, even under request floods.
func (h *ModelCatalogHandler) RequireEnabled(c *gin.Context) {
	if h.settingService == nil || h.catalogService == nil {
		c.Header("Cache-Control", "no-store")
		response.NotFound(c, "Not found")
		c.Abort()
		return
	}
	runtime := h.settingService.GetPublicModelCatalogRuntime(c.Request.Context())
	if !runtime.Enabled || runtime.BackendMode {
		c.Header("Cache-Control", "no-store")
		response.NotFound(c, "Not found")
		c.Abort()
		return
	}
	c.Set(modelCatalogGuardPassedKey, true)
	c.Next()
}

// List handles GET /api/v1/catalog/models.
func (h *ModelCatalogHandler) List(c *gin.Context) {
	if h.settingService == nil || h.catalogService == nil {
		c.Header("Cache-Control", "no-store")
		response.NotFound(c, "Not found")
		return
	}
	guardPassed, _ := c.Get(modelCatalogGuardPassedKey)
	if passed, _ := guardPassed.(bool); !passed {
		runtime := h.settingService.GetPublicModelCatalogRuntime(c.Request.Context())
		if !runtime.Enabled || runtime.BackendMode {
			// Deliberately indistinguishable from an absent route: disabled/backend
			// deployments must not reveal whether a private catalog exists.
			c.Header("Cache-Control", "no-store")
			response.NotFound(c, "Not found")
			return
		}
	}
	locale := strings.TrimSpace(c.Query("lang"))
	if locale == "" {
		locale = c.GetHeader("Accept-Language")
	}
	data, etag, err := h.catalogService.PublicSnapshot(c.Request.Context(), locale)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Cache-Control", "public, max-age=30, must-revalidate")
	c.Header("ETag", etag)
	c.Writer.Header().Add("Vary", "Accept-Language")
	if requestETagMatches(c.GetHeader("If-None-Match"), etag) {
		c.Status(http.StatusNotModified)
		return
	}
	response.Success(c, data)
}

func requestETagMatches(header, etag string) bool {
	if etag == "" {
		return false
	}
	for _, candidate := range strings.Split(header, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" || candidate == etag || strings.TrimPrefix(candidate, "W/") == etag {
			return true
		}
	}
	return false
}
