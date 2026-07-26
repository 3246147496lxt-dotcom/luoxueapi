package admin

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type AdminChatHistoryHandler struct {
	chat *service.AdminChatHistoryService
}

func NewAdminChatHistoryHandler(
	chat *service.AdminChatHistoryService,
) *AdminChatHistoryHandler {
	return &AdminChatHistoryHandler{chat: chat}
}

func (h *AdminChatHistoryHandler) ListUserConversations(c *gin.Context) {
	setAdminChatPrivateHeaders(c)
	if _, ok := requireAdminChatJWT(c); !ok {
		return
	}
	targetUserID, ok := parsePositivePathID(c, "id")
	if !ok {
		return
	}
	limit, ok := parseAdminChatLimit(c)
	if !ok {
		return
	}
	cursor, ok := decodeAdminChatCursor(c.Query("cursor"))
	if !ok {
		response.ErrorWithDetails(
			c,
			http.StatusBadRequest,
			"Invalid chat conversation cursor",
			"INVALID_CHAT_CONVERSATION_CURSOR",
			nil,
		)
		return
	}

	query := service.AdminChatConversationListQuery{
		TargetUserID: targetUserID,
		Limit:        limit,
	}
	if cursor != nil {
		query.BeforeUpdatedAt = &cursor.UpdatedAt
		query.BeforeID = &cursor.ID
	}
	page, err := h.chat.ListUserConversationRefs(c.Request.Context(), query)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if page.NextUpdatedAt != nil && page.NextID != nil {
		page.NextCursor = encodeAdminChatCursor(*page.NextUpdatedAt, *page.NextID)
	}
	response.Success(c, page)
}

func (h *AdminChatHistoryHandler) ViewConversation(c *gin.Context) {
	setAdminChatPrivateHeaders(c)
	adminID, ok := requireAdminChatJWT(c)
	if !ok {
		return
	}
	targetUserID, ok := parsePositivePathID(c, "id")
	if !ok {
		return
	}
	conversationID := strings.TrimSpace(c.Param("conversation_id"))
	if conversationID == "" || len(conversationID) > 80 {
		response.ErrorWithDetails(
			c,
			http.StatusBadRequest,
			"Invalid chat conversation id",
			"INVALID_CHAT_CONVERSATION_ID",
			nil,
		)
		return
	}
	limit, ok := parseAdminChatLimit(c)
	if !ok {
		return
	}
	beforePosition, ok := parseOptionalPositiveInt64(c, "before_position")
	if !ok {
		return
	}
	requestID, _ := c.Request.Context().Value(ctxkey.RequestID).(string)

	page, err := h.chat.ViewConversationPage(
		c.Request.Context(),
		service.AdminChatContentAccessQuery{
			AdminID:              adminID,
			TargetUserID:         targetUserID,
			ConversationPublicID: conversationID,
			BeforePosition:       beforePosition,
			Limit:                limit,
			RequestID:            requestID,
			ClientIP:             ip.GetTrustedClientIP(c),
			UserAgent:            c.Request.UserAgent(),
		},
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, page)
}

func setAdminChatPrivateHeaders(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	c.Header("Pragma", "no-cache")
	c.Header("Vary", "Authorization")
}

func requireAdminChatJWT(c *gin.Context) (int64, bool) {
	if c.GetString("auth_method") != service.AuditAuthMethodJWT {
		response.ErrorWithDetails(
			c,
			http.StatusForbidden,
			"An administrator login session is required",
			"ADMIN_CHAT_JWT_REQUIRED",
			nil,
		)
		return 0, false
	}
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.ErrorWithDetails(
			c,
			http.StatusUnauthorized,
			"Administrator not authenticated",
			"UNAUTHORIZED",
			nil,
		)
		return 0, false
	}
	role, ok := middleware.GetUserRoleFromContext(c)
	if !ok || role != "admin" {
		response.ErrorWithDetails(
			c,
			http.StatusForbidden,
			"Administrator access required",
			"ADMIN_ACCESS_REQUIRED",
			nil,
		)
		return 0, false
	}
	return subject.UserID, true
}

func parsePositivePathID(c *gin.Context, name string) (int64, bool) {
	value, err := strconv.ParseInt(strings.TrimSpace(c.Param(name)), 10, 64)
	if err != nil || value <= 0 {
		response.ErrorWithDetails(
			c,
			http.StatusBadRequest,
			"Invalid "+name,
			"INVALID_ADMIN_CHAT_PARAMETER",
			nil,
		)
		return 0, false
	}
	return value, true
}

func parseAdminChatLimit(c *gin.Context) (int, bool) {
	raw := strings.TrimSpace(c.Query("limit"))
	if raw == "" {
		return service.DefaultAdminChatPageLimit, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 || value > service.MaxAdminChatPageLimit {
		response.ErrorWithDetails(
			c,
			http.StatusBadRequest,
			"Chat page limit must be between 1 and 100",
			"INVALID_ADMIN_CHAT_PAGE_LIMIT",
			nil,
		)
		return 0, false
	}
	return value, true
}

func parseOptionalPositiveInt64(c *gin.Context, name string) (*int64, bool) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return nil, true
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		response.ErrorWithDetails(
			c,
			http.StatusBadRequest,
			"Invalid "+name,
			"INVALID_ADMIN_CHAT_PARAMETER",
			nil,
		)
		return nil, false
	}
	return &value, true
}

type adminChatCursor struct {
	UpdatedAt time.Time `json:"updated_at"`
	ID        int64     `json:"id"`
}

func encodeAdminChatCursor(updatedAt time.Time, id int64) string {
	payload, err := json.Marshal(adminChatCursor{
		UpdatedAt: updatedAt.UTC(),
		ID:        id,
	})
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(payload)
}

func decodeAdminChatCursor(value string) (*adminChatCursor, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, true
	}
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, false
	}
	var cursor adminChatCursor
	if err := json.Unmarshal(raw, &cursor); err != nil {
		return nil, false
	}
	if cursor.ID <= 0 || cursor.UpdatedAt.IsZero() {
		return nil, false
	}
	cursor.UpdatedAt = cursor.UpdatedAt.UTC()
	return &cursor, true
}
