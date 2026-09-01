package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func RegisterProjectRoutes(v1 *gin.RouterGroup, h *handler.Handlers, jwtAuth middleware.JWTAuthMiddleware, settingService *service.SettingService) {
	projects := v1.Group("/projects")
	projects.Use(gin.HandlerFunc(jwtAuth), middleware.BackendModeUserGuard(settingService))
	projects.GET("", h.Chat.ListProjects)
	projects.POST("", h.Chat.CreateProject)
	projects.GET("/:project_id", h.Chat.GetProject)
	projects.PATCH("/:project_id", h.Chat.UpdateProject)
	projects.DELETE("/:project_id", h.Chat.DeleteProject)
	projects.POST("/:project_id/files", h.Chat.AddProjectFile)
	projects.DELETE("/:project_id/files/:file_id", h.Chat.RemoveProjectFile)
}
