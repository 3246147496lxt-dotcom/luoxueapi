package routes

import (
	"context"
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/lifecycle"
	"github.com/gin-gonic/gin"
)

// RegisterCommonRoutes 注册通用路由（健康检查、状态等）
func RegisterCommonRoutes(r *gin.Engine) {
	RegisterCommonRoutesWithReadiness(r, nil)
}

// RegisterCommonRoutesWithReadiness preserves /health byte-for-byte while
// exposing process liveness separately from dependency-aware readiness.
func RegisterCommonRoutesWithReadiness(r *gin.Engine, readinessProbe lifecycle.ReadinessProbe) {
	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/livez", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/readyz", func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		if readinessProbe == nil {
			c.JSON(http.StatusServiceUnavailable, lifecycle.ReadinessResult{
				Status: lifecycle.ReadinessNotReady,
				Checks: map[string]lifecycle.CheckResult{
					"probe": {Status: lifecycle.ReadinessNotReady, Detail: "unavailable"},
				},
			})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		result := readinessProbe.Probe(ctx)
		status := http.StatusOK
		if !result.Ready() {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, result)
	})

	// Claude Code 遥测日志（忽略，直接返回200）
	r.POST("/api/event_logging/batch", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Setup status endpoint (always returns needs_setup: false in normal mode)
	// This is used by the frontend to detect when the service has restarted after setup
	r.GET("/setup/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": gin.H{
				"needs_setup": false,
				"step":        "completed",
			},
		})
	})
}
