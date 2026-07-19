package routes

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	basemiddleware "github.com/Wei-Shaw/sub2api/internal/middleware"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const (
	publicDocumentationRateLimit      = 120
	publicDocumentationAssetRateLimit = 120
)

func RegisterDocumentationRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	redisClient *redis.Client,
	settingService *service.SettingService,
) {
	limiter := basemiddleware.NewRateLimiter(redisClient)
	documentation := v1.Group("/public/documentation")
	documentation.Use(documentationPublicGuard(settingService))
	{
		documentation.GET("", limiter.Limit("public-documentation", publicDocumentationRateLimit, time.Minute), h.Documentation.Get)
		documentation.GET("/assets/:id", limiter.Limit("public-documentation-assets", publicDocumentationAssetRateLimit, time.Minute), h.Documentation.GetAsset)
	}
}

func documentationPublicGuard(settingService *service.SettingService) gin.HandlerFunc {
	if settingService == nil {
		return documentationBackendModeGuard(nil)
	}
	return documentationBackendModeGuard(settingService.IsBackendModeEnabled)
}

func documentationBackendModeGuard(isBackendModeEnabled func(context.Context) bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if isBackendModeEnabled == nil || isBackendModeEnabled(c.Request.Context()) {
			c.Header("Cache-Control", "no-store")
			response.NotFound(c, "Not found")
			c.Abort()
			return
		}
		c.Next()
	}
}
