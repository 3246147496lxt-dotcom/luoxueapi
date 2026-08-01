package server

import (
	"log"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/lifecycle"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/server/routes"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/web"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// SetupRouter 配置路由器中间件和路由
func SetupRouter(
	r *gin.Engine,
	handlers *handler.Handlers,
	jwtAuth middleware2.JWTAuthMiddleware,
	adminAuth middleware2.AdminAuthMiddleware,
	apiKeyAuth middleware2.APIKeyAuthMiddleware,
	desktopAuth middleware2.DesktopAuthMiddleware,
	quotaAuth middleware2.QuotaAuthMiddleware,
	auditLog middleware2.AuditLogMiddleware,
	stepUpAuth middleware2.StepUpAuthMiddleware,
	apiKeyService *service.APIKeyService,
	subscriptionService *service.SubscriptionService,
	opsService *service.OpsService,
	settingService *service.SettingService,
	cfg *config.Config,
	redisClient *redis.Client,
	readinessProbe lifecycle.ReadinessProbe,
	routerSettingsRuntime *RouterSettingsRuntime,
) *gin.Engine {
	// 应用中间件
	r.Use(middleware2.RequestLogger())
	// 将可信客户端 IP + UA 注入 request context，供 token 签发路径写入会话绑定
	r.Use(middleware2.SessionBindingContext())
	r.Use(middleware2.Logger())
	r.Use(middleware2.CORS(cfg.CORS))
	r.Use(middleware2.SecurityHeaders(cfg.Security.CSP, func() []string {
		if routerSettingsRuntime != nil {
			return routerSettingsRuntime.FrameSrcOrigins()
		}
		return nil
	}))
	r.Use(middleware2.ServerTiming(cfg.Server.EnableServerTiming))

	// Serve embedded frontend with settings injection if available
	if web.HasEmbeddedFrontend() {
		var frontendMiddleware gin.HandlerFunc
		var err error
		ok := false
		if routerSettingsRuntime != nil {
			frontendMiddleware, ok, err = routerSettingsRuntime.FrontendMiddleware()
		}
		if !ok {
			log.Printf("Warning: Failed to create frontend server with settings injection: %v, using legacy mode", err)
			r.Use(web.ServeEmbeddedFrontend())
		} else {
			r.Use(frontendMiddleware)
		}
	}

	// 注册路由
	registerRoutes(r, handlers, jwtAuth, adminAuth, apiKeyAuth, desktopAuth, quotaAuth, auditLog, stepUpAuth, apiKeyService, subscriptionService, opsService, settingService, cfg, redisClient, readinessProbe)

	return r
}

// registerRoutes 注册所有 HTTP 路由
func registerRoutes(
	r *gin.Engine,
	h *handler.Handlers,
	jwtAuth middleware2.JWTAuthMiddleware,
	adminAuth middleware2.AdminAuthMiddleware,
	apiKeyAuth middleware2.APIKeyAuthMiddleware,
	desktopAuth middleware2.DesktopAuthMiddleware,
	quotaAuth middleware2.QuotaAuthMiddleware,
	auditLog middleware2.AuditLogMiddleware,
	stepUpAuth middleware2.StepUpAuthMiddleware,
	apiKeyService *service.APIKeyService,
	subscriptionService *service.SubscriptionService,
	opsService *service.OpsService,
	settingService *service.SettingService,
	cfg *config.Config,
	redisClient *redis.Client,
	readinessProbe lifecycle.ReadinessProbe,
) {
	// 通用路由（健康检查、状态等）
	routes.RegisterCommonRoutesWithReadiness(r, readinessProbe)

	// API v1
	v1 := r.Group("/api/v1")

	// 注册各模块路由
	routes.RegisterAuthRoutes(v1, h, jwtAuth, auditLog, redisClient, settingService)
	routes.RegisterUserRoutes(v1, h, jwtAuth, auditLog, settingService)
	routes.RegisterDesktopRoutes(v1, h, jwtAuth, desktopAuth, auditLog, settingService, redisClient)
	routes.RegisterQuotaRoutes(v1, h, jwtAuth, quotaAuth, auditLog, settingService, cfg, redisClient)
	routes.RegisterChatRoutes(v1, h, jwtAuth, opsService, settingService, cfg)
	routes.RegisterAdminRoutes(v1, h, adminAuth, auditLog, stepUpAuth, settingService)
	routes.RegisterModelCatalogRoutes(v1, h, redisClient)
	routes.RegisterDocumentationRoutes(v1, h, redisClient, settingService)
	routes.RegisterEmbeddedPageRoutes(v1, jwtAuth, auditLog, settingService, redisClient)
	routes.RegisterGatewayRoutes(r, h, apiKeyAuth, apiKeyService, subscriptionService, opsService, settingService, cfg)
	routes.RegisterPaymentRoutes(v1, h.Payment, h.PaymentWebhook, h.Admin.Payment, jwtAuth, adminAuth, auditLog, settingService)

	handler.RegisterPageRoutes(v1, cfg.Pricing.DataDir, gin.HandlerFunc(jwtAuth), gin.HandlerFunc(adminAuth), settingService)
}
