package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	MaxChatHistoryImportMessages        = 200
	MaxChatHistoryImportedConversations = 50
	maxChatHistoryPublicIDLength        = 80
	maxChatHistoryTitleLength           = 200
	maxChatHistoryModelLength           = 128
	maxChatHistorySearchLength          = 100
	maxChatHistoryPageSize              = 100
	maxChatHistoryMessagePageSize       = 200
	maxChatCompletionContextMessages    = 200
	maxChatCompletionContentBytes       = 2 << 20
)

const (
	ChatMessageDeliveryPending     = "pending"
	ChatMessageDeliveryStreaming   = "streaming"
	ChatMessageDeliveryCompleted   = "completed"
	ChatMessageDeliveryPartial     = "partial"
	ChatMessageDeliveryStopped     = "stopped"
	ChatMessageDeliveryInterrupted = "interrupted"
	ChatMessageDeliveryError       = "error"
)

var (
	ErrChatHistoryInvalid = infraerrors.BadRequest(
		"CHAT_HISTORY_INVALID",
		"The chat history request is invalid",
	)
	ErrChatHistoryNotFound = infraerrors.NotFound(
		"CHAT_CONVERSATION_NOT_FOUND",
		"Chat conversation not found",
	)
	ErrChatHistoryConflict = infraerrors.Conflict(
		"CHAT_HISTORY_CONFLICT",
		"The chat history id was reused with different content",
	)
	ErrChatHistoryRevisionConflict = infraerrors.Conflict(
		"CHAT_HISTORY_REVISION_CONFLICT",
		"The chat conversation was changed on another device",
	)
	ErrChatTurnInProgress = infraerrors.Conflict(
		"CHAT_TURN_IN_PROGRESS",
		"The chat conversation already has a response in progress",
	)
	ErrChatHistoryImportLimit = infraerrors.Conflict(
		"CHAT_HISTORY_IMPORT_LIMIT",
		"The legacy chat history import limit was reached",
	)
	ErrChatHistoryUnavailable = infraerrors.ServiceUnavailable(
		"CHAT_HISTORY_UNAVAILABLE",
		"Chat history is temporarily unavailable",
	)
	ErrChatAttemptNotFound = infraerrors.NotFound(
		"CHAT_ATTEMPT_NOT_FOUND",
		"Chat attempt not found",
	)
)

type ChatHistoryConversation struct {
	ID            string     `json:"id"`
	Title         string     `json:"title"`
	Model         string     `json:"model"`
	Revision      int64      `json:"revision"`
	Version       int64      `json:"version"`
	HeadMessageID *string    `json:"head_message_id,omitempty"`
	MessageCount  int        `json:"message_count"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `json:"-"`
}

type ChatHistoryMessage struct {
	ID                    string     `json:"id"`
	Position              int64      `json:"position"`
	Role                  string     `json:"role"`
	Content               string     `json:"content"`
	Status                string     `json:"status"`
	RequestedModel        string     `json:"requested_model,omitempty"`
	FinishReason          *string    `json:"finish_reason,omitempty"`
	ErrorCode             *string    `json:"error_code,omitempty"`
	ErrorMessage          *string    `json:"error_message,omitempty"`
	AttemptID             *string    `json:"attempt_id,omitempty"`
	ReceiptID             *string    `json:"receipt_id,omitempty"`
	ExcludedFromContext   bool       `json:"excluded_from_context,omitempty"`
	SupersededByMessageID *string    `json:"superseded_by_message_id,omitempty"`
	CheckpointSeq         int64      `json:"checkpoint_seq,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
	TerminalAt            *time.Time `json:"terminal_at,omitempty"`
}

type ChatHistoryImportedMessage struct {
	ID        string
	Role      string
	Content   string
	Status    string
	CreatedAt time.Time
}

type CreateChatHistoryConversationInput struct {
	ID               string
	Title            string
	Model            string
	Imported         bool
	ImportedMessages []ChatHistoryImportedMessage
	CreateHash       string
}

type UpdateChatHistoryConversationInput struct {
	ID       string
	Revision int64
	Title    *string
	Model    *string
}

type ChatHistoryConversationList struct {
	Items      []*ChatHistoryConversation `json:"items"`
	NextCursor string                     `json:"next_cursor"`
	HasMore    bool                       `json:"has_more"`
}

type ChatHistoryMessagePage struct {
	Items              []*ChatHistoryMessage `json:"items"`
	NextBeforePosition *int64                `json:"next_before_position,omitempty"`
	HasMore            bool                  `json:"has_more"`
}

type ChatHistorySyncChange struct {
	Version        int64                    `json:"version"`
	Type           string                   `json:"type"`
	ConversationID string                   `json:"conversation_id"`
	Conversation   *ChatHistoryConversation `json:"conversation,omitempty"`
	DeletedAt      *time.Time               `json:"deleted_at,omitempty"`
}

type ChatHistorySyncPage struct {
	Changes       []*ChatHistorySyncChange `json:"changes"`
	LatestVersion int64                    `json:"latest_version"`
	NextCursor    string                   `json:"next_cursor"`
	HasMore       bool                     `json:"has_more"`
}

type ChatHistoryAttempt struct {
	AttemptID          string              `json:"attempt_id"`
	ConversationID     string              `json:"conversation_id,omitempty"`
	AssistantMessageID string              `json:"assistant_message_id,omitempty"`
	ReceiptID          string              `json:"receipt_id"`
	Status             string              `json:"status"`
	HTTPStatus         *int                `json:"http_status,omitempty"`
	FailureCode        *string             `json:"failure_code,omitempty"`
	FailureReason      *string             `json:"failure_reason,omitempty"`
	AssistantMessage   *ChatHistoryMessage `json:"assistant_message,omitempty"`
	CreatedAt          time.Time           `json:"created_at"`
	UpdatedAt          time.Time           `json:"updated_at"`
}

type ChatCompletionHistoryUserMessage struct {
	ID      string
	Content string
}

type PrepareChatCompletionInput struct {
	ConversationID        string
	Model                 string
	ReasoningEffort       string
	ExpectedHeadMessageID *string
	UserMessage           *ChatCompletionHistoryUserMessage
	RetryOfMessageID      string
	AssistantMessageID    string
	AttemptID             string
	ClientRequestID       string
	RequestHash           string
	ContextMessageLimit   int
}

type ChatCompletionContextMessage struct {
	Role    string
	Content string
}

type PreparedChatCompletion struct {
	Claimed            bool
	ClientRequestID    string
	AttemptStatus      string
	ConversationID     string
	AssistantMessageID string
	Messages           []ChatCompletionContextMessage
}

type FinalizeChatCompletionInput struct {
	AttemptID          string
	AssistantMessageID string
	Content            string
	CheckpointSeq      int64
	DeliveryStatus     string
	AttemptStatus      string
	HTTPStatus         int
	FinishReason       string
	ErrorCode          string
	ErrorMessage       string
}

type CheckpointChatCompletionInput struct {
	AttemptID          string
	AssistantMessageID string
	Content            string
	CheckpointSeq      int64
}

type ChatHistoryRepository interface {
	CreateConversation(ctx context.Context, userID int64, input *CreateChatHistoryConversationInput) (*ChatHistoryConversation, error)
	ListConversations(ctx context.Context, userID int64, cursor string, limit int, search string) (*ChatHistoryConversationList, error)
	GetConversation(ctx context.Context, userID int64, publicID string) (*ChatHistoryConversation, error)
	ListMessages(ctx context.Context, userID int64, conversationPublicID string, beforePosition *int64, limit int) (*ChatHistoryMessagePage, error)
	UpdateConversation(ctx context.Context, userID int64, input *UpdateChatHistoryConversationInput) (*ChatHistoryConversation, error)
	DeleteConversation(ctx context.Context, userID int64, publicID string, revision int64) error
	Sync(ctx context.Context, userID, afterVersion int64, limit int) (*ChatHistorySyncPage, error)
	GetAttempt(ctx context.Context, userID int64, attemptID string) (*ChatHistoryAttempt, error)
	PrepareCompletion(ctx context.Context, userID int64, input *PrepareChatCompletionInput) (*PreparedChatCompletion, error)
	CheckpointCompletion(ctx context.Context, userID int64, input *CheckpointChatCompletionInput) error
	FinalizeCompletion(ctx context.Context, userID int64, input *FinalizeChatCompletionInput) error
}

type ChatHistoryService struct {
	repo ChatHistoryRepository
}

func NewChatHistoryService(repo ChatHistoryRepository) *ChatHistoryService {
	return &ChatHistoryService{repo: repo}
}

func (s *ChatHistoryService) CreateConversation(
	ctx context.Context,
	userID int64,
	input *CreateChatHistoryConversationInput,
) (*ChatHistoryConversation, error) {
	if s == nil || s.repo == nil || userID <= 0 || input == nil {
		return nil, ErrChatHistoryUnavailable
	}
	normalized, err := normalizeCreateChatHistoryInput(input)
	if err != nil {
		return nil, err
	}
	return s.repo.CreateConversation(ctx, userID, normalized)
}

func (s *ChatHistoryService) ListConversations(
	ctx context.Context,
	userID int64,
	cursor string,
	limit int,
	search string,
) (*ChatHistoryConversationList, error) {
	if s == nil || s.repo == nil || userID <= 0 {
		return nil, ErrChatHistoryUnavailable
	}
	cursor = strings.TrimSpace(cursor)
	if limit <= 0 {
		limit = 20
	}
	if limit > maxChatHistoryPageSize {
		limit = maxChatHistoryPageSize
	}
	search = strings.TrimSpace(search)
	if utf8.RuneCountInString(search) > maxChatHistorySearchLength {
		return nil, ErrChatHistoryInvalid
	}
	return s.repo.ListConversations(ctx, userID, cursor, limit, search)
}

func (s *ChatHistoryService) GetConversation(
	ctx context.Context,
	userID int64,
	publicID string,
) (*ChatHistoryConversation, error) {
	if s == nil || s.repo == nil || userID <= 0 {
		return nil, ErrChatHistoryUnavailable
	}
	publicID = strings.TrimSpace(publicID)
	if !validChatHistoryPublicID(publicID) {
		return nil, ErrChatHistoryInvalid
	}
	return s.repo.GetConversation(ctx, userID, publicID)
}

func (s *ChatHistoryService) ListMessages(
	ctx context.Context,
	userID int64,
	conversationPublicID string,
	beforePosition *int64,
	limit int,
) (*ChatHistoryMessagePage, error) {
	if s == nil || s.repo == nil || userID <= 0 {
		return nil, ErrChatHistoryUnavailable
	}
	conversationPublicID = strings.TrimSpace(conversationPublicID)
	if !validChatHistoryPublicID(conversationPublicID) ||
		(beforePosition != nil && *beforePosition <= 0) {
		return nil, ErrChatHistoryInvalid
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > maxChatHistoryMessagePageSize {
		limit = maxChatHistoryMessagePageSize
	}
	return s.repo.ListMessages(ctx, userID, conversationPublicID, beforePosition, limit)
}

func (s *ChatHistoryService) UpdateConversation(
	ctx context.Context,
	userID int64,
	input *UpdateChatHistoryConversationInput,
) (*ChatHistoryConversation, error) {
	if s == nil || s.repo == nil || userID <= 0 || input == nil {
		return nil, ErrChatHistoryUnavailable
	}
	normalized := *input
	normalized.ID = strings.TrimSpace(normalized.ID)
	if !validChatHistoryPublicID(normalized.ID) || normalized.Revision <= 0 ||
		(normalized.Title == nil && normalized.Model == nil) {
		return nil, ErrChatHistoryInvalid
	}
	if normalized.Title != nil {
		value := strings.TrimSpace(*normalized.Title)
		if value == "" || utf8.RuneCountInString(value) > maxChatHistoryTitleLength {
			return nil, ErrChatHistoryInvalid
		}
		normalized.Title = &value
	}
	if normalized.Model != nil {
		value := strings.TrimSpace(*normalized.Model)
		if value == "" || len(value) > maxChatHistoryModelLength {
			return nil, ErrChatHistoryInvalid
		}
		normalized.Model = &value
	}
	return s.repo.UpdateConversation(ctx, userID, &normalized)
}

func (s *ChatHistoryService) DeleteConversation(
	ctx context.Context,
	userID int64,
	publicID string,
	revision int64,
) error {
	if s == nil || s.repo == nil || userID <= 0 {
		return ErrChatHistoryUnavailable
	}
	publicID = strings.TrimSpace(publicID)
	if !validChatHistoryPublicID(publicID) || revision <= 0 {
		return ErrChatHistoryInvalid
	}
	return s.repo.DeleteConversation(ctx, userID, publicID, revision)
}

func (s *ChatHistoryService) Sync(
	ctx context.Context,
	userID int64,
	cursor string,
	limit int,
) (*ChatHistorySyncPage, error) {
	if s == nil || s.repo == nil || userID <= 0 {
		return nil, ErrChatHistoryUnavailable
	}
	afterVersion, err := parseChatHistoryVersionCursor(cursor)
	if err != nil {
		return nil, ErrChatHistoryInvalid
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > maxChatHistoryPageSize {
		limit = maxChatHistoryPageSize
	}
	return s.repo.Sync(ctx, userID, afterVersion, limit)
}

func (s *ChatHistoryService) GetAttempt(
	ctx context.Context,
	userID int64,
	attemptID string,
) (*ChatHistoryAttempt, error) {
	if s == nil || s.repo == nil || userID <= 0 {
		return nil, ErrChatHistoryUnavailable
	}
	attemptID = strings.TrimSpace(attemptID)
	if !validChatAttemptID(attemptID) {
		return nil, ErrChatAttemptIDInvalid
	}
	return s.repo.GetAttempt(ctx, userID, attemptID)
}

func (s *ChatHistoryService) PrepareCompletion(
	ctx context.Context,
	userID int64,
	attemptID string,
	clientRequestID string,
	input *PrepareChatCompletionInput,
) (*PreparedChatCompletion, error) {
	if s == nil || s.repo == nil || userID <= 0 || input == nil {
		return nil, ErrChatHistoryUnavailable
	}

	normalized := *input
	normalized.ConversationID = strings.TrimSpace(normalized.ConversationID)
	normalized.Model = strings.TrimSpace(normalized.Model)
	normalized.ReasoningEffort = strings.TrimSpace(normalized.ReasoningEffort)
	normalized.AssistantMessageID = strings.TrimSpace(normalized.AssistantMessageID)
	normalized.RetryOfMessageID = strings.TrimSpace(normalized.RetryOfMessageID)
	normalized.AttemptID = strings.TrimSpace(attemptID)
	normalized.ClientRequestID = strings.TrimSpace(clientRequestID)
	normalized.ContextMessageLimit = maxChatCompletionContextMessages

	if !validChatHistoryPublicID(normalized.ConversationID) ||
		!validChatHistoryPublicID(normalized.AssistantMessageID) ||
		!validChatAttemptID(normalized.AttemptID) ||
		normalized.ClientRequestID == "" ||
		len(normalized.ClientRequestID) > 64 ||
		normalized.Model == "" ||
		len(normalized.Model) > maxChatHistoryModelLength {
		return nil, ErrChatHistoryInvalid
	}

	if normalized.ExpectedHeadMessageID != nil {
		value := strings.TrimSpace(*normalized.ExpectedHeadMessageID)
		if !validChatHistoryPublicID(value) {
			return nil, ErrChatHistoryInvalid
		}
		normalized.ExpectedHeadMessageID = &value
	}

	hasUserMessage := normalized.UserMessage != nil
	hasRetry := normalized.RetryOfMessageID != ""
	if hasUserMessage == hasRetry {
		return nil, ErrChatHistoryInvalid
	}
	if hasUserMessage {
		message := *normalized.UserMessage
		message.ID = strings.TrimSpace(message.ID)
		if !validChatHistoryPublicID(message.ID) ||
			strings.TrimSpace(message.Content) == "" ||
			len(message.Content) > maxChatCompletionContentBytes {
			return nil, ErrChatHistoryInvalid
		}
		normalized.UserMessage = &message
	} else if !validChatHistoryPublicID(normalized.RetryOfMessageID) {
		return nil, ErrChatHistoryInvalid
	}

	hashPayload := struct {
		ConversationID        string                            `json:"conversation_id"`
		Model                 string                            `json:"model"`
		ReasoningEffort       string                            `json:"reasoning_effort,omitempty"`
		ExpectedHeadMessageID *string                           `json:"expected_head_message_id"`
		UserMessage           *ChatCompletionHistoryUserMessage `json:"user_message,omitempty"`
		RetryOfMessageID      string                            `json:"retry_of_message_id,omitempty"`
		AssistantMessageID    string                            `json:"assistant_message_id"`
	}{
		ConversationID:        normalized.ConversationID,
		Model:                 normalized.Model,
		ReasoningEffort:       normalized.ReasoningEffort,
		ExpectedHeadMessageID: normalized.ExpectedHeadMessageID,
		UserMessage:           normalized.UserMessage,
		RetryOfMessageID:      normalized.RetryOfMessageID,
		AssistantMessageID:    normalized.AssistantMessageID,
	}
	payload, err := json.Marshal(hashPayload)
	if err != nil {
		return nil, ErrChatHistoryInvalid.WithCause(err)
	}
	sum := sha256.Sum256(payload)
	normalized.RequestHash = hex.EncodeToString(sum[:])

	result, err := s.repo.PrepareCompletion(ctx, userID, &normalized)
	if err != nil {
		return result, err
	}
	if result == nil || strings.TrimSpace(result.ClientRequestID) == "" {
		return nil, ErrChatHistoryUnavailable
	}
	if result.Claimed {
		return result, nil
	}

	metadata := map[string]string{
		"receipt_id":           result.ClientRequestID,
		"attempt_status":       result.AttemptStatus,
		"conversation_id":      result.ConversationID,
		"assistant_message_id": result.AssistantMessageID,
	}
	return result, ErrChatAttemptAlreadySubmitted.WithMetadata(metadata)
}

func (s *ChatHistoryService) FinalizeCompletion(
	ctx context.Context,
	userID int64,
	input *FinalizeChatCompletionInput,
) error {
	if s == nil || s.repo == nil || userID <= 0 || input == nil {
		return ErrChatHistoryUnavailable
	}
	normalized := *input
	normalized.AttemptID = strings.TrimSpace(normalized.AttemptID)
	normalized.AssistantMessageID = strings.TrimSpace(normalized.AssistantMessageID)
	normalized.FinishReason = strings.TrimSpace(normalized.FinishReason)
	normalized.ErrorCode = strings.TrimSpace(normalized.ErrorCode)
	normalized.ErrorMessage = strings.TrimSpace(normalized.ErrorMessage)
	if !validChatAttemptID(normalized.AttemptID) ||
		!validChatHistoryPublicID(normalized.AssistantMessageID) ||
		normalized.CheckpointSeq <= 0 ||
		len(normalized.FinishReason) > 64 ||
		len(normalized.ErrorCode) > 64 ||
		utf8.RuneCountInString(normalized.ErrorMessage) > 500 {
		return ErrChatHistoryInvalid
	}

	validTerminalState := false
	switch normalized.DeliveryStatus {
	case ChatMessageDeliveryCompleted:
		validTerminalState = normalized.AttemptStatus == ChatAttemptStatusCompleted
	case ChatMessageDeliveryPartial, ChatMessageDeliveryInterrupted, ChatMessageDeliveryStopped:
		validTerminalState = normalized.AttemptStatus == ChatAttemptStatusInterrupted
	case ChatMessageDeliveryError:
		validTerminalState = normalized.AttemptStatus == ChatAttemptStatusFailed
	}
	if !validTerminalState {
		return ErrChatHistoryInvalid
	}
	return s.repo.FinalizeCompletion(ctx, userID, &normalized)
}

func (s *ChatHistoryService) CheckpointCompletion(
	ctx context.Context,
	userID int64,
	input *CheckpointChatCompletionInput,
) error {
	if s == nil || s.repo == nil || userID <= 0 || input == nil {
		return ErrChatHistoryUnavailable
	}
	normalized := *input
	normalized.AttemptID = strings.TrimSpace(normalized.AttemptID)
	normalized.AssistantMessageID = strings.TrimSpace(normalized.AssistantMessageID)
	if !validChatAttemptID(normalized.AttemptID) ||
		!validChatHistoryPublicID(normalized.AssistantMessageID) ||
		normalized.CheckpointSeq <= 0 {
		return ErrChatHistoryInvalid
	}
	return s.repo.CheckpointCompletion(ctx, userID, &normalized)
}

func normalizeCreateChatHistoryInput(
	input *CreateChatHistoryConversationInput,
) (*CreateChatHistoryConversationInput, error) {
	normalized := *input
	normalized.ID = strings.TrimSpace(normalized.ID)
	normalized.Title = strings.TrimSpace(normalized.Title)
	normalized.Model = strings.TrimSpace(normalized.Model)
	if !validChatHistoryPublicID(normalized.ID) ||
		normalized.Title == "" ||
		utf8.RuneCountInString(normalized.Title) > maxChatHistoryTitleLength ||
		normalized.Model == "" ||
		len(normalized.Model) > maxChatHistoryModelLength ||
		len(normalized.ImportedMessages) > MaxChatHistoryImportMessages {
		return nil, ErrChatHistoryInvalid
	}
	if !normalized.Imported && len(normalized.ImportedMessages) > 0 {
		return nil, ErrChatHistoryInvalid
	}

	now := time.Now().UTC()
	seen := make(map[string]struct{}, len(normalized.ImportedMessages))
	for i := range normalized.ImportedMessages {
		message := &normalized.ImportedMessages[i]
		message.ID = strings.TrimSpace(message.ID)
		message.Role = strings.TrimSpace(message.Role)
		message.Status = normalizeImportedChatMessageStatus(message.Status)
		if !validChatHistoryPublicID(message.ID) ||
			(message.Role != "user" && message.Role != "assistant") ||
			message.Status == "" ||
			strings.TrimSpace(message.Content) == "" {
			return nil, ErrChatHistoryInvalid
		}
		if _, exists := seen[message.ID]; exists {
			return nil, ErrChatHistoryInvalid
		}
		seen[message.ID] = struct{}{}
		if message.CreatedAt.IsZero() ||
			message.CreatedAt.After(now.Add(5*time.Minute)) {
			return nil, ErrChatHistoryInvalid
		}
		message.CreatedAt = message.CreatedAt.UTC()
	}

	hashPayload := struct {
		ID       string                       `json:"id"`
		Title    string                       `json:"title"`
		Model    string                       `json:"model"`
		Imported bool                         `json:"imported"`
		Messages []ChatHistoryImportedMessage `json:"messages,omitempty"`
	}{
		ID:       normalized.ID,
		Title:    normalized.Title,
		Model:    normalized.Model,
		Imported: normalized.Imported,
		Messages: normalized.ImportedMessages,
	}
	payload, err := json.Marshal(hashPayload)
	if err != nil {
		return nil, ErrChatHistoryInvalid.WithCause(err)
	}
	sum := sha256.Sum256(payload)
	normalized.CreateHash = hex.EncodeToString(sum[:])
	return &normalized, nil
}

func normalizeImportedChatMessageStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "complete", "completed":
		return ChatMessageDeliveryCompleted
	case "streaming", "stopped", "interrupted", "error":
		return ChatMessageDeliveryInterrupted
	default:
		return ""
	}
}

func validChatHistoryPublicID(value string) bool {
	if len(value) < 8 || len(value) > maxChatHistoryPublicIDLength {
		return false
	}
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-', r == '_':
		default:
			return false
		}
	}
	return true
}

func parseChatHistoryVersionCursor(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	cursor, err := strconv.ParseInt(value, 10, 64)
	if err != nil || cursor < 0 {
		return 0, ErrChatHistoryInvalid
	}
	return cursor, nil
}
