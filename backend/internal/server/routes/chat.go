package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// RegisterChatRoutes deliberately omits AuditLogMiddleware so conversation
// bodies are never persisted as administrative audit request payloads.
func RegisterChatRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	jwtAuth middleware.JWTAuthMiddleware,
	opsService *service.OpsService,
	settingService *service.SettingService,
	cfg *config.Config,
) {
	bodyLimit := middleware.RequestBodyLimit(cfg.Gateway.MaxBodySize)
	clientRequestID := middleware.ClientRequestID()
	opsErrorLogger := handler.OpsErrorLoggerMiddleware(opsService)
	endpointNorm := handler.InboundEndpointMiddleware()

	chat := v1.Group("/chat")
	chat.Use(bodyLimit)
	chat.Use(clientRequestID)
	chat.Use(middleware.WebChatIngress())
	chat.Use(opsErrorLogger)
	chat.Use(endpointNorm)
	chat.Use(gin.HandlerFunc(jwtAuth))
	chat.Use(middleware.BackendModeUserGuard(settingService))
	chat.GET("/models", h.Chat.Models)
	chat.GET("/capabilities", h.Chat.Capabilities)
	chat.POST("/attachments", middleware.RequestBodyLimit(cfg.ChatAttachments.RequestBodyLimit()), h.Chat.UploadAttachment)
	chat.DELETE("/attachments/:id", h.Chat.DeleteAttachment)
	chat.GET("/attachments/:id/content", h.Chat.AttachmentContent)
	chat.GET("/receipts/:receipt_id", h.Chat.Receipt)
	chat.GET("/attempts/:attempt_id", h.Chat.Attempt)
	chat.POST("/attempts/:attempt_id/stop", h.Chat.StopAttempt)
	chat.GET("/sync", h.Chat.SyncConversations)
	chat.POST("/conversations/search", h.Chat.SearchConversations)
	chat.POST("/conversations", h.Chat.CreateConversation)
	chat.GET("/conversations", h.Chat.ListConversations)
	chat.GET("/conversations/:conversation_id", h.Chat.GetConversation)
	chat.GET("/conversations/:conversation_id/messages", h.Chat.ListConversationMessages)
	chat.PATCH("/conversations/:conversation_id", h.Chat.UpdateConversation)
	chat.PATCH("/conversations/:conversation_id/project", h.Chat.MoveConversationToProject)
	chat.DELETE("/conversations/:conversation_id", h.Chat.DeleteConversation)
	chat.POST("/transcriptions", middleware.RequestBodyLimit(cfg.Transcription.RequestBodyLimit()), h.Chat.Transcriptions)
	chat.POST("/completions", h.Chat.Completions)
}
