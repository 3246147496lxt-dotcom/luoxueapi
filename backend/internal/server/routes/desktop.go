package routes

import (
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	basemiddleware "github.com/Wei-Shaw/sub2api/internal/middleware"
	"github.com/Wei-Shaw/sub2api/internal/modules/desktop"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const desktopControlRequestBodyLimit int64 = 64 << 10

func RegisterDesktopRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	jwtAuth middleware.JWTAuthMiddleware,
	desktopAuth middleware.DesktopAuthMiddleware,
	auditLog middleware.AuditLogMiddleware,
	settingService *service.SettingService,
	redisClient *redis.Client,
) {
	public := v1.Group("/public/desktop")
	public.GET("/releases/macos/latest", h.Desktop.PublicLatestRelease)
	public.GET("/releases/macos/latest/download", h.Desktop.PublicLatestReleaseDownload)

	rateLimiter := basemiddleware.NewRateLimiter(redisClient)
	failClose := basemiddleware.RateLimitOptions{
		FailureMode:          basemiddleware.RateLimitFailClose,
		BackendFailureStatus: http.StatusServiceUnavailable,
		BackendFailureReason: "DESKTOP_SERVICE_UNAVAILABLE",
	}
	bodyLimit := middleware.RequestBodyLimit(desktopControlRequestBodyLimit)

	root := v1.Group("/desktop")
	root.POST("/pairings", bodyLimit, rateLimiter.LimitWithOptions("desktop-pairing-create", 10, time.Minute, failClose), h.Desktop.CreatePairing)
	root.POST("/pairings/token", bodyLimit, rateLimiter.LimitWithOptions("desktop-pairing-exchange", 30, time.Minute, failClose), h.Desktop.ExchangePairing)
	root.POST("/sessions/refresh", bodyLimit, rateLimiter.LimitWithOptions("desktop-session-refresh", 30, time.Minute, failClose), h.Desktop.RefreshSession)

	web := root.Group("/authorizations")
	web.Use(gin.HandlerFunc(jwtAuth))
	web.Use(middleware.BackendModeUserGuard(settingService))
	web.Use(gin.HandlerFunc(auditLog))
	web.GET("/:user_code", h.Desktop.PairingPreview)
	web.POST("/:user_code/approve", h.Desktop.ApprovePairing)

	webDevices := root.Group("/devices")
	webDevices.Use(gin.HandlerFunc(jwtAuth))
	webDevices.Use(middleware.BackendModeUserGuard(settingService))
	webDevices.Use(gin.HandlerFunc(auditLog))
	webDevices.GET("", h.Desktop.ListDevices)
	webDevices.PATCH("/:device_id", h.Desktop.RenameDevice)
	webDevices.DELETE("/:device_id", h.Desktop.RevokeDevice)

	device := root.Group("")
	device.Use(gin.HandlerFunc(desktopAuth))
	device.GET("/me", middleware.RequireDesktopScope(desktop.ScopeProfileRead), h.Desktop.Me)
	device.POST("/me/heartbeat", middleware.RequireDesktopScope(desktop.ScopeProfileRead), h.Desktop.Heartbeat)
	device.POST("/me/activate", middleware.RequireDesktopScope(desktop.ScopeProfileRead), h.Desktop.Activate)
	device.DELETE("/me", middleware.RequireDesktopScope(desktop.ScopeProfileRead), h.Desktop.RevokeCurrentDevice)
	device.GET("/routes", middleware.RequireDesktopScope(desktop.ScopeRoutesRead), h.Desktop.Routes)
	device.GET("/usage/today", middleware.RequireDesktopScope(desktop.ScopeProfileRead), h.Desktop.TodayUsage)
	device.GET("/releases/:target/:arch/:current_version", middleware.RequireDesktopScope(desktop.ScopeProfileRead), h.Desktop.Release)
	device.POST("/diagnostics", middleware.RequireDesktopScope(desktop.ScopeDiagnosticsWrite), h.Desktop.UploadDiagnostic)
	device.PUT("/managed-keys/:group_id", middleware.RequireDesktopScope(desktop.ScopeManagedKeyWrite), h.Desktop.EnsureManagedKey)
}
