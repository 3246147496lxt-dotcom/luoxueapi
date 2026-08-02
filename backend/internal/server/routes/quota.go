package routes

import (
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	basemiddleware "github.com/Wei-Shaw/sub2api/internal/middleware"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const quotaControlRequestBodyLimit int64 = 64 << 10

// RegisterQuotaRoutes exposes the quota viewer's isolated device authorization,
// installer-download, and read-only quota surfaces. A quota access token is
// deliberately never accepted by the website write routes.
func RegisterQuotaRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	jwtAuth middleware.JWTAuthMiddleware,
	quotaAuth middleware.QuotaAuthMiddleware,
	auditLog middleware.AuditLogMiddleware,
	settingService *service.SettingService,
	cfg *config.Config,
	redisClient *redis.Client,
) {
	rateLimiter := basemiddleware.NewRateLimiter(redisClient)
	failClose := basemiddleware.RateLimitOptions{
		FailureMode:          basemiddleware.RateLimitFailClose,
		BackendFailureStatus: http.StatusServiceUnavailable,
		BackendFailureReason: "QUOTA_SERVICE_UNAVAILABLE",
	}
	bodyLimit := middleware.RequestBodyLimit(quotaControlRequestBodyLimit)
	installerStore := repository.NewQuotaViewerInstallerTicketStore(redisClient)
	installerService := service.NewQuotaViewerInstallerDownloadService(installerStore)
	installerHandler := handler.NewQuotaViewerInstallerHandler(installerService)

	root := v1.Group("/quota")
	root.GET(
		"/releases/:platform/latest/download",
		quotaViewerInstallerDownloadHeaders(),
		rateLimiter.LimitWithOptions("quota-installer-download", 60, time.Minute, failClose),
		installerHandler.Download,
	)
	root.POST(
		"/pairings",
		bodyLimit,
		rateLimiter.LimitWithOptions("quota-pairing-create", 10, time.Minute, failClose),
		h.QuotaAuth.CreatePairing,
	)
	root.POST(
		"/pairings/token",
		bodyLimit,
		rateLimiter.LimitWithOptions("quota-pairing-exchange", 30, time.Minute, failClose),
		h.QuotaAuth.ExchangePairing,
	)
	root.POST(
		"/sessions/refresh",
		bodyLimit,
		rateLimiter.LimitWithOptions("quota-session-refresh", 30, time.Minute, failClose),
		h.QuotaAuth.RefreshSession,
	)

	web := root.Group("/authorizations")
	web.Use(gin.HandlerFunc(jwtAuth))
	web.Use(middleware.BackendModeUserGuard(settingService))
	web.Use(gin.HandlerFunc(auditLog))
	web.GET("/:user_code", h.QuotaAuth.PairingPreview)
	web.POST("/:user_code/approve", bodyLimit, h.QuotaAuth.ApprovePairing)

	webDevices := root.Group("/devices")
	webDevices.Use(gin.HandlerFunc(jwtAuth))
	webDevices.Use(middleware.BackendModeUserGuard(settingService))
	webDevices.Use(gin.HandlerFunc(auditLog))
	webDevices.GET("", h.QuotaAuth.ListDevices)
	webDevices.DELETE("/:device_id", h.QuotaAuth.RevokeDevice)

	webDownloads := root.Group("/releases")
	webDownloads.Use(quotaViewerInstallerDownloadHeaders())
	webDownloads.Use(gin.HandlerFunc(jwtAuth))
	webDownloads.Use(middleware.BackendModeUserGuard(settingService))
	webDownloads.Use(quotaViewerInstallerDownloadAuditAction())
	webDownloads.Use(gin.HandlerFunc(auditLog))
	webDownloads.POST(
		"/:platform/latest/download",
		rateLimiter.LimitWithOptions("quota-installer-ticket-issue", 12, time.Minute, failClose),
		installerHandler.IssueDownload,
	)

	readOnly := root.Group("")
	readOnly.Use(gin.HandlerFunc(quotaAuth))
	readOnly.Use(middleware.RequireQuotaRead())
	readOnly.Use(quotaOverviewModeGuard(cfg))
	readOnly.GET(
		"/overview",
		rateLimiter.LimitWithOptions("quota-overview-read", 12, time.Minute, failClose),
		h.QuotaOverview.GetOverview,
	)
}

func quotaViewerInstallerDownloadAuditAction() gin.HandlerFunc {
	return func(c *gin.Context) {
		middleware.SetAuditAction(c, handler.QuotaViewerInstallerDownloadAuditAction)
		c.Next()
	}
}

func quotaViewerInstallerDownloadHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		c.Header("Pragma", "no-cache")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("Vary", "Authorization")
		c.Next()
	}
}

func quotaOverviewModeGuard(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg == nil || cfg.RunMode != config.RunModeSimple {
			c.Next()
			return
		}

		c.Header("Cache-Control", "private, no-store")
		c.Header("Vary", "Authorization")
		response.ErrorFrom(c, service.ErrQuotaOverviewUnavailable)
		c.Abort()
	}
}
