package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RegisterEmbeddedPageRoutes wires the authenticated launch endpoint and the
// deliberately unauthenticated, server-to-server exchange endpoint.
func RegisterEmbeddedPageRoutes(
	v1 *gin.RouterGroup,
	jwtAuth servermiddleware.JWTAuthMiddleware,
	auditLog servermiddleware.AuditLogMiddleware,
	settingService *service.SettingService,
	redisClient *redis.Client,
) {
	launchStore := repository.NewEmbeddedPageLaunchStore(redisClient)
	launchService := service.NewEmbeddedPageLaunchService(settingService, launchStore)
	h := handler.NewEmbeddedPageLaunchHandler(launchService)

	customPages := v1.Group("/user/custom-pages")
	customPages.Use(gin.HandlerFunc(jwtAuth))
	customPages.Use(servermiddleware.BackendModeUserGuard(settingService))
	customPages.Use(gin.HandlerFunc(auditLog))
	customPages.POST("/:id/launch", h.Launch)

	embeddedPages := v1.Group("/embedded-pages")
	embeddedPages.POST("/exchange", h.Exchange)
}
