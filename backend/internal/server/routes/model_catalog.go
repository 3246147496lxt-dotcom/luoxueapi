package routes

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	basemiddleware "github.com/Wei-Shaw/sub2api/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func RegisterModelCatalogRoutes(v1 *gin.RouterGroup, h *handler.Handlers, redisClient *redis.Client) {
	limiter := basemiddleware.NewRateLimiter(redisClient)
	catalog := v1.Group("/catalog")
	catalog.GET("/models", h.ModelCatalog.RequireEnabled, limiter.Limit("public-model-catalog", 120, time.Minute), h.ModelCatalog.List)
}
