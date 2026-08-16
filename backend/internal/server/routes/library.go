package routes

import (
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	basemiddleware "github.com/Wei-Shaw/sub2api/internal/middleware"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const libraryDownloadTicketConsumeIPLimit = 120

// RegisterLibraryRoutes exposes private, user-owned persistent files. File
// bodies intentionally bypass the administrative audit middleware.
func RegisterLibraryRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	jwtAuth middleware.JWTAuthMiddleware,
	settingService *service.SettingService,
	cfg *config.Config,
	redisClient *redis.Client,
) {
	libraryRoot := v1.Group("/library")
	libraryRoot.Use(middleware.ClientRequestID())
	// A download ticket is a 60-second, single-use handle plus an independent
	// HttpOnly cookie secret issued only after JWT authorization. The URL and
	// access logs contain only the non-sensitive handle. Keeping this GET outside
	// JWT lets the browser download manager stream to disk without a bearer token.
	rateLimiter := basemiddleware.NewRateLimiter(redisClient)
	libraryRoot.GET(
		"/download/:ticket_id",
		func(c *gin.Context) {
			c.Header("Cache-Control", "private, no-store")
			c.Header("Referrer-Policy", "no-referrer")
			c.Header("X-Content-Type-Options", "nosniff")
			c.Header("X-Frame-Options", "SAMEORIGIN")
			c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'self'")
			c.Next()
		},
		rateLimiter.LimitWithOptions(
			"library-download-ticket-consume",
			libraryDownloadTicketConsumeIPLimit,
			time.Minute,
			basemiddleware.RateLimitOptions{
				FailureMode:          basemiddleware.RateLimitFailClose,
				BackendFailureStatus: http.StatusServiceUnavailable,
				BackendFailureReason: service.ErrLibraryDownloadTicketUnavailable.Reason,
			},
		),
		h.Library.DownloadWithTicket,
	)

	library := libraryRoot.Group("")
	library.Use(gin.HandlerFunc(jwtAuth))
	library.Use(middleware.BackendModeUserGuard(settingService))
	library.GET("/files", h.Library.List)
	library.POST("/files/upload", middleware.RequestBodyLimit(cfg.Library.RequestBodyLimit()), h.Library.Upload)
	library.POST("/files/batch-download", h.Library.BatchDownload)
	library.POST("/files/download-ticket", h.Library.IssueDownloadTicket)
	library.GET("/files/:id", h.Library.Detail)
	library.GET("/files/:id/thumbnail", h.Library.Thumbnail)
	library.GET("/files/:id/preview", h.Library.Preview)
	library.GET("/files/:id/download", h.Library.Download)
	library.DELETE("/files/:id", h.Library.Delete)
	library.GET("/storage", h.Library.Storage)
}
