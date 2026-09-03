package routes

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	basemiddleware "github.com/Wei-Shaw/sub2api/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func RegisterSkillMarketRoutes(v1 *gin.RouterGroup, h *handler.Handlers, redisClient *redis.Client) {
	limiter := basemiddleware.NewRateLimiter(redisClient)
	skills := v1.Group("/catalog/skills")
	skills.Use(h.SkillMarket.RequireEnabled)
	{
		skills.GET("", limiter.Limit("public-skill-market-list", 120, time.Minute), h.SkillMarket.List)
		skills.GET("/:slug", limiter.Limit("public-skill-market-detail", 120, time.Minute), h.SkillMarket.Get)
		skills.GET("/:slug/versions", limiter.Limit("public-skill-market-versions", 120, time.Minute), h.SkillMarket.Versions)
		skills.GET("/:slug/versions/:version/download", limiter.Limit("public-skill-market-download", 60, time.Minute), h.SkillMarket.DownloadVersion)
	}
}
