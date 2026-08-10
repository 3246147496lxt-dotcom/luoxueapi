package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type createChatConversationRequest struct {
	ID               string                        `json:"id"`
	Title            string                        `json:"title"`
	Model            string                        `json:"model"`
	ImportedMessages *[]importedChatMessageRequest `json:"imported_messages,omitempty"`
}

type importedChatMessageRequest struct {
	ID        string          `json:"id"`
	Role      string          `json:"role"`
	Content   string          `json:"content"`
	Status    string          `json:"status"`
	CreatedAt json.RawMessage `json:"created_at,omitempty"`
}

type updateChatConversationRequest struct {
	Revision int64   `json:"revision"`
	Title    *string `json:"title,omitempty"`
	Model    *string `json:"model,omitempty"`
}

type deleteChatConversationRequest struct {
	Revision int64 `json:"revision"`
}

type searchChatConversationsRequest struct {
	Query  string `json:"query"`
	Cursor string `json:"cursor,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

func (h *ChatHandler) CreateConversation(c *gin.Context) {
	userID, ok := h.chatHistoryUser(c)
	if !ok {
		return
	}
	var request createChatConversationRequest
	if err := decodeStrictChatHistoryJSON(c, &request); err != nil {
		writeChatHistoryDecodeError(c, err)
		return
	}

	imported := request.ImportedMessages != nil
	messages := make([]service.ChatHistoryImportedMessage, 0)
	if imported {
		messages = make([]service.ChatHistoryImportedMessage, 0, len(*request.ImportedMessages))
		for i := range *request.ImportedMessages {
			message := (*request.ImportedMessages)[i]
			createdAt, err := parseImportedChatMessageTime(message.CreatedAt)
			if err != nil {
				response.ErrorFrom(c, service.ErrChatHistoryInvalid)
				return
			}
			messages = append(messages, service.ChatHistoryImportedMessage{
				ID:        message.ID,
				Role:      message.Role,
				Content:   message.Content,
				Status:    message.Status,
				CreatedAt: createdAt,
			})
		}
	}

	conversation, err := h.history.CreateConversation(
		c.Request.Context(),
		userID,
		&service.CreateChatHistoryConversationInput{
			ID:               request.ID,
			Title:            request.Title,
			Model:            request.Model,
			Imported:         imported,
			ImportedMessages: messages,
		},
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, conversation)
}

func (h *ChatHandler) ListConversations(c *gin.Context) {
	userID, ok := h.chatHistoryUser(c)
	if !ok {
		return
	}
	if strings.TrimSpace(c.Query("q")) != "" {
		response.ErrorFrom(c, service.ErrChatHistoryInvalid)
		return
	}
	limit, err := parsePositiveChatHistoryInt(c.Query("limit"), 20)
	if err != nil {
		response.ErrorFrom(c, service.ErrChatHistoryInvalid)
		return
	}
	result, err := h.history.ListConversations(
		c.Request.Context(),
		userID,
		c.Query("cursor"),
		limit,
		"",
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *ChatHandler) SearchConversations(c *gin.Context) {
	userID, ok := h.chatHistoryUser(c)
	if !ok {
		return
	}
	var request searchChatConversationsRequest
	if err := decodeStrictChatHistoryJSON(c, &request); err != nil {
		writeChatHistoryDecodeError(c, err)
		return
	}
	result, err := h.history.ListConversations(
		c.Request.Context(),
		userID,
		request.Cursor,
		request.Limit,
		request.Query,
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *ChatHandler) GetConversation(c *gin.Context) {
	userID, ok := h.chatHistoryUser(c)
	if !ok {
		return
	}
	conversation, err := h.history.GetConversation(
		c.Request.Context(),
		userID,
		c.Param("conversation_id"),
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, conversation)
}

func (h *ChatHandler) ListConversationMessages(c *gin.Context) {
	userID, ok := h.chatHistoryUser(c)
	if !ok {
		return
	}
	var beforePosition *int64
	if raw := strings.TrimSpace(c.Query("before_position")); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value <= 0 {
			response.ErrorFrom(c, service.ErrChatHistoryInvalid)
			return
		}
		beforePosition = &value
	}
	limit, err := parsePositiveChatHistoryInt(c.Query("limit"), 50)
	if err != nil {
		response.ErrorFrom(c, service.ErrChatHistoryInvalid)
		return
	}
	page, err := h.history.ListMessages(
		c.Request.Context(),
		userID,
		c.Param("conversation_id"),
		beforePosition,
		limit,
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, page)
}

func (h *ChatHandler) UpdateConversation(c *gin.Context) {
	userID, ok := h.chatHistoryUser(c)
	if !ok {
		return
	}
	var request updateChatConversationRequest
	if err := decodeStrictChatHistoryJSON(c, &request); err != nil {
		writeChatHistoryDecodeError(c, err)
		return
	}
	conversation, err := h.history.UpdateConversation(
		c.Request.Context(),
		userID,
		&service.UpdateChatHistoryConversationInput{
			ID:       c.Param("conversation_id"),
			Revision: request.Revision,
			Title:    request.Title,
			Model:    request.Model,
		},
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, conversation)
}

func (h *ChatHandler) DeleteConversation(c *gin.Context) {
	userID, ok := h.chatHistoryUser(c)
	if !ok {
		return
	}
	var revision int64
	if c.Request.ContentLength != 0 {
		var request deleteChatConversationRequest
		if err := decodeStrictChatHistoryJSON(c, &request); err != nil {
			writeChatHistoryDecodeError(c, err)
			return
		}
		revision = request.Revision
	} else if raw := strings.TrimSpace(c.Query("revision")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			response.ErrorFrom(c, service.ErrChatHistoryInvalid)
			return
		}
		revision = parsed
	}
	if revision <= 0 {
		response.ErrorFrom(c, service.ErrChatHistoryInvalid)
		return
	}
	if err := h.history.DeleteConversation(
		c.Request.Context(),
		userID,
		c.Param("conversation_id"),
		revision,
	); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if h.attachments != nil {
		if err := h.attachments.CleanupConversation(c.Request.Context(), userID, c.Param("conversation_id")); err != nil {
			slog.Warn("chat attachment conversation cleanup deferred to janitor", "user_id", userID, "conversation_id", c.Param("conversation_id"), "error", err)
		}
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *ChatHandler) SyncConversations(c *gin.Context) {
	userID, ok := h.chatHistoryUser(c)
	if !ok {
		return
	}
	limit, err := parsePositiveChatHistoryInt(c.Query("limit"), 100)
	if err != nil {
		response.ErrorFrom(c, service.ErrChatHistoryInvalid)
		return
	}
	cursor := strings.TrimSpace(c.Query("cursor"))
	if cursor == "" {
		cursor = strings.TrimSpace(c.Query("after_version"))
	}
	page, err := h.history.Sync(
		c.Request.Context(),
		userID,
		cursor,
		limit,
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, page)
}

func (h *ChatHandler) Attempt(c *gin.Context) {
	userID, ok := h.chatHistoryUser(c)
	if !ok {
		return
	}
	attempt, err := h.history.GetAttempt(
		c.Request.Context(),
		userID,
		c.Param("attempt_id"),
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, attempt)
}

func (h *ChatHandler) chatHistoryUser(c *gin.Context) (int64, bool) {
	c.Header("Cache-Control", "private, no-store")
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "User not authenticated")
		return 0, false
	}
	if h == nil || h.history == nil {
		response.InternalError(c, "Chat history service is unavailable")
		return 0, false
	}
	return subject.UserID, true
}

func decodeStrictChatHistoryJSON(c *gin.Context, destination any) error {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxChatRequestBytes)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return err
	}
	return nil
}

func writeChatHistoryDecodeError(c *gin.Context, err error) {
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		response.ErrorWithDetails(
			c,
			http.StatusRequestEntityTooLarge,
			"Chat history request is too large",
			"REQUEST_TOO_LARGE",
			nil,
		)
		return
	}
	response.ErrorFrom(c, service.ErrChatHistoryInvalid)
}

func parsePositiveChatHistoryInt(value string, fallback int) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, service.ErrChatHistoryInvalid
	}
	return parsed, nil
}

func parseImportedChatMessageTime(raw json.RawMessage) (time.Time, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return time.Time{}, nil
	}
	if raw[0] == '"' {
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return time.Time{}, err
		}
		return time.Parse(time.RFC3339Nano, value)
	}
	var milliseconds int64
	if err := json.Unmarshal(raw, &milliseconds); err != nil {
		return time.Time{}, err
	}
	if milliseconds < 0 {
		return time.Time{}, service.ErrChatHistoryInvalid
	}
	return time.UnixMilli(milliseconds).UTC(), nil
}
