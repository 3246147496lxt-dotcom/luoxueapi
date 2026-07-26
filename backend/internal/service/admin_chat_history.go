package service

import (
	"context"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	DefaultAdminChatPageLimit = 30
	MaxAdminChatPageLimit     = 100
)

var (
	ErrAdminChatHistoryInvalid = infraerrors.BadRequest(
		"ADMIN_CHAT_HISTORY_INVALID",
		"invalid administrator chat history request",
	)
	ErrAdminChatConversationNotFound = infraerrors.NotFound(
		"ADMIN_CHAT_CONVERSATION_NOT_FOUND",
		"chat conversation not found",
	)
	ErrAdminChatHistoryUnavailable = infraerrors.ServiceUnavailable(
		"ADMIN_CHAT_HISTORY_UNAVAILABLE",
		"chat history is temporarily unavailable",
	)
	ErrAdminChatContentAuditUnavailable = infraerrors.ServiceUnavailable(
		"ADMIN_CHAT_CONTENT_AUDIT_UNAVAILABLE",
		"chat content access could not be audited",
	)
)

// AdminChatConversationRef is the deliberately narrow metadata projection used
// by administrators before they choose to access any conversation content.
// Title and message previews are intentionally absent.
type AdminChatConversationRef struct {
	ID           string    `json:"id"`
	Model        string    `json:"model"`
	MessageCount int       `json:"message_count"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type AdminChatConversationRefPage struct {
	Items         []AdminChatConversationRef `json:"items"`
	NextCursor    string                     `json:"next_cursor,omitempty"`
	HasMore       bool                       `json:"has_more"`
	NextUpdatedAt *time.Time                 `json:"-"`
	NextID        *int64                     `json:"-"`
}

type AdminChatConversationListQuery struct {
	TargetUserID    int64
	Limit           int
	BeforeUpdatedAt *time.Time
	BeforeID        *int64
}

type AdminChatMessage struct {
	ID             string     `json:"id"`
	Position       int64      `json:"position"`
	Role           string     `json:"role"`
	Content        string     `json:"content"`
	Status         string     `json:"status"`
	RequestedModel string     `json:"requested_model,omitempty"`
	FinishReason   string     `json:"finish_reason,omitempty"`
	ErrorCode      string     `json:"error_code,omitempty"`
	ErrorMessage   string     `json:"error_message,omitempty"`
	AttemptID      string     `json:"attempt_id,omitempty"`
	ReceiptID      string     `json:"receipt_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

type AdminChatConversationContent struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Model     string    `json:"model"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AdminChatConversationPage struct {
	Conversation       AdminChatConversationContent `json:"conversation"`
	Messages           []AdminChatMessage           `json:"messages"`
	NextBeforePosition *int64                       `json:"next_before_position,omitempty"`
	HasMore            bool                         `json:"has_more"`
}

// AdminChatContentAccessQuery contains only immutable identifiers and request
// metadata. Message content must never be copied into the access audit row.
type AdminChatContentAccessQuery struct {
	AdminID              int64
	TargetUserID         int64
	ConversationPublicID string
	BeforePosition       *int64
	Limit                int
	RequestID            string
	ClientIP             string
	UserAgent            string
}

// AdminChatHistoryRepository intentionally exposes no mutation or export
// methods. The content read and audit insert are one database transaction.
type AdminChatHistoryRepository interface {
	ListUserConversationRefs(
		ctx context.Context,
		query AdminChatConversationListQuery,
	) (*AdminChatConversationRefPage, error)
	ViewConversationPageAndRecordAccess(
		ctx context.Context,
		query AdminChatContentAccessQuery,
	) (*AdminChatConversationPage, error)
}

type AdminChatHistoryService struct {
	repo AdminChatHistoryRepository
}

func NewAdminChatHistoryService(repo AdminChatHistoryRepository) *AdminChatHistoryService {
	return &AdminChatHistoryService{repo: repo}
}

func (s *AdminChatHistoryService) ListUserConversationRefs(
	ctx context.Context,
	query AdminChatConversationListQuery,
) (*AdminChatConversationRefPage, error) {
	if s == nil || s.repo == nil {
		return nil, ErrAdminChatContentAuditUnavailable
	}
	if query.TargetUserID <= 0 ||
		(query.BeforeUpdatedAt == nil) != (query.BeforeID == nil) ||
		(query.BeforeID != nil && *query.BeforeID <= 0) {
		return nil, ErrAdminChatHistoryInvalid
	}
	query.Limit = normalizeAdminChatPageLimit(query.Limit)
	return s.repo.ListUserConversationRefs(ctx, query)
}

func (s *AdminChatHistoryService) ViewConversationPage(
	ctx context.Context,
	query AdminChatContentAccessQuery,
) (*AdminChatConversationPage, error) {
	if s == nil || s.repo == nil {
		return nil, ErrAdminChatContentAuditUnavailable
	}
	query.ConversationPublicID = strings.TrimSpace(query.ConversationPublicID)
	query.RequestID = truncateAdminChatAuditValue(query.RequestID, 64)
	query.ClientIP = truncateAdminChatAuditValue(query.ClientIP, 64)
	query.UserAgent = truncateAdminChatAuditValue(query.UserAgent, 512)
	query.Limit = normalizeAdminChatPageLimit(query.Limit)
	if query.AdminID <= 0 ||
		query.TargetUserID <= 0 ||
		query.ConversationPublicID == "" ||
		len(query.ConversationPublicID) > 80 ||
		(query.BeforePosition != nil && *query.BeforePosition <= 0) {
		return nil, ErrAdminChatHistoryInvalid
	}
	return s.repo.ViewConversationPageAndRecordAccess(ctx, query)
}

func normalizeAdminChatPageLimit(limit int) int {
	if limit <= 0 {
		return DefaultAdminChatPageLimit
	}
	if limit > MaxAdminChatPageLimit {
		return MaxAdminChatPageLimit
	}
	return limit
}

func truncateAdminChatAuditValue(value string, max int) string {
	value = strings.TrimSpace(value)
	if max > 0 && len(value) > max {
		return value[:max]
	}
	return value
}
