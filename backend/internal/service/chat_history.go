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

	"github.com/Wei-Shaw/sub2api/internal/config"
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

const (
	ChatMessageActivitySourceOpenAIResponses = "openai_responses"
	ChatMessageActivityTypeReasoningSummary  = "reasoning_summary"

	ChatMessageActivityStatusInProgress   = "in_progress"
	ChatMessageActivityStatusCompleted    = "completed"
	ChatMessageActivityStatusIncomplete   = "incomplete"
	ChatMessageActivityStatusFailed       = "failed"
	ChatMessageActivityStatusInterrupted  = "interrupted"
	ChatMessageActivityStatusStopped      = "stopped"
	ChatMessageActivityStatusDisconnected = "disconnected"

	maxChatMessageActivitiesPerSnapshot = 256
	maxChatMessageActivityTextBytes     = 2 << 20
	maxChatMessageActivityMetadataBytes = 16 << 10
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
	ID                    string                `json:"id"`
	Position              int64                 `json:"position"`
	Role                  string                `json:"role"`
	Content               string                `json:"content"`
	Status                string                `json:"status"`
	RequestedModel        string                `json:"requested_model,omitempty"`
	FinishReason          *string               `json:"finish_reason,omitempty"`
	ErrorCode             *string               `json:"error_code,omitempty"`
	ErrorMessage          *string               `json:"error_message,omitempty"`
	AttemptID             *string               `json:"attempt_id,omitempty"`
	ReceiptID             *string               `json:"receipt_id,omitempty"`
	ExcludedFromContext   bool                  `json:"excluded_from_context,omitempty"`
	SupersededByMessageID *string               `json:"superseded_by_message_id,omitempty"`
	CheckpointSeq         int64                 `json:"checkpoint_seq,omitempty"`
	CreatedAt             time.Time             `json:"created_at"`
	UpdatedAt             time.Time             `json:"updated_at"`
	TerminalAt            *time.Time            `json:"terminal_at,omitempty"`
	Attachments           []ChatAttachment      `json:"attachments,omitempty"`
	Activities            []ChatMessageActivity `json:"activities,omitempty"`
}

// ChatMessageActivity is a durable, snapshot-upserted reasoning-summary part
// associated with one assistant message. SequenceStart/SequenceEnd refer to
// upstream Responses event ordering, while SortOrder is the stable UI order.
type ChatMessageActivity struct {
	ID              int64           `json:"id,omitempty"`
	ResponseID      string          `json:"response_id,omitempty"`
	Source          string          `json:"source"`
	ActivityType    string          `json:"activity_type"`
	ItemID          string          `json:"item_id"`
	OutputIndex     int             `json:"output_index"`
	SummaryIndex    int             `json:"summary_index"`
	SortOrder       int64           `json:"sort_order"`
	Status          string          `json:"status"`
	Text            string          `json:"text"`
	SequenceStart   int64           `json:"sequence_start"`
	SequenceEnd     int64           `json:"sequence_end"`
	ReasoningMode   string          `json:"reasoning_mode,omitempty"`
	ReasoningEffort string          `json:"reasoning_effort,omitempty"`
	StartedAt       time.Time       `json:"started_at"`
	CompletedAt     *time.Time      `json:"completed_at,omitempty"`
	Metadata        json.RawMessage `json:"metadata,omitempty"`
	CreatedAt       time.Time       `json:"created_at,omitempty"`
	UpdatedAt       time.Time       `json:"updated_at,omitempty"`
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

type StopChatCompletionResult struct {
	AttemptID      string     `json:"attempt_id"`
	Accepted       bool       `json:"accepted"`
	AttemptStatus  string     `json:"attempt_status"`
	DeliveryStatus string     `json:"delivery_status,omitempty"`
	StoppedAt      *time.Time `json:"stopped_at,omitempty"`
}

type ChatCompletionHistoryUserMessage struct {
	ID            string
	Content       string
	AttachmentIDs []string
	Attachments   []ChatAttachment
}

type PrepareChatCompletionInput struct {
	ConversationID        string
	Model                 string
	ReasoningMode         string
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
	Role        string
	Content     string
	Attachments []ChatAttachment
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
	Activities         []ChatMessageActivity
}

type CheckpointChatCompletionInput struct {
	AttemptID          string
	AssistantMessageID string
	Content            string
	CheckpointSeq      int64
	Activities         []ChatMessageActivity
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

type chatHistoryStopRepository interface {
	StopCompletion(ctx context.Context, userID int64, attemptID string) (*StopChatCompletionResult, error)
}

type ChatHistoryService struct {
	repo             ChatHistoryRepository
	attachments      ChatAttachmentRepository
	attachmentConfig config.ChatAttachmentConfig
}

func NewChatHistoryService(repo ChatHistoryRepository) *ChatHistoryService {
	return &ChatHistoryService{repo: repo}
}

func ProvideChatHistoryService(repo ChatHistoryRepository, attachments ChatAttachmentRepository, cfg *config.Config) *ChatHistoryService {
	service := NewChatHistoryService(repo)
	service.attachments = attachments
	if cfg != nil {
		service.attachmentConfig = cfg.ChatAttachments
	}
	return service
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

func (s *ChatHistoryService) StopCompletion(
	ctx context.Context,
	userID int64,
	attemptID string,
) (*StopChatCompletionResult, error) {
	if s == nil || s.repo == nil || userID <= 0 {
		return nil, ErrChatHistoryUnavailable
	}
	attemptID = strings.TrimSpace(attemptID)
	if !validChatAttemptID(attemptID) {
		return nil, ErrChatAttemptIDInvalid
	}
	repo, ok := s.repo.(chatHistoryStopRepository)
	if !ok || repo == nil {
		return nil, ErrChatHistoryUnavailable
	}
	return repo.StopCompletion(ctx, userID, attemptID)
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
	normalized.ReasoningMode = strings.ToLower(strings.TrimSpace(normalized.ReasoningMode))
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
		message.AttachmentIDs = append([]string(nil), message.AttachmentIDs...)
		if !validChatHistoryPublicID(message.ID) ||
			(strings.TrimSpace(message.Content) == "" && len(message.AttachmentIDs) == 0) ||
			len(message.Content) > maxChatCompletionContentBytes {
			return nil, ErrChatHistoryInvalid
		}
		if len(message.AttachmentIDs) > 0 {
			if s.attachments == nil {
				return nil, ErrChatAttachmentUnavailable
			}
			for i := range message.AttachmentIDs {
				message.AttachmentIDs[i] = strings.TrimSpace(message.AttachmentIDs[i])
				if !validChatHistoryPublicID(message.AttachmentIDs[i]) {
					return nil, ErrChatAttachmentInvalid
				}
			}
			attachments, resolveErr := s.attachments.ResolveForCompletion(ctx, userID, message.AttachmentIDs)
			if resolveErr != nil {
				return nil, resolveErr
			}
			if validateErr := ValidateAttachmentSelection(attachments, s.attachmentConfig); validateErr != nil {
				return nil, validateErr
			}
			message.Attachments = attachments
		}
		normalized.UserMessage = &message
	} else if !validChatHistoryPublicID(normalized.RetryOfMessageID) {
		return nil, ErrChatHistoryInvalid
	}

	type hashAttachment struct {
		ID     string `json:"id"`
		Digest string `json:"digest"`
	}
	type hashUserMessage struct {
		ID      string `json:"id"`
		Content string `json:"content"`
	}
	var hashAttachments []hashAttachment
	var hashUser *hashUserMessage
	if normalized.UserMessage != nil {
		hashUser = &hashUserMessage{ID: normalized.UserMessage.ID, Content: normalized.UserMessage.Content}
		for i := range normalized.UserMessage.Attachments {
			hashAttachments = append(hashAttachments, hashAttachment{
				ID:     normalized.UserMessage.Attachments[i].ID,
				Digest: normalized.UserMessage.Attachments[i].Digest,
			})
		}
	}
	hashPayload := struct {
		ConversationID        string           `json:"conversation_id"`
		Model                 string           `json:"model"`
		ReasoningMode         string           `json:"reasoning_mode,omitempty"`
		ReasoningEffort       string           `json:"reasoning_effort,omitempty"`
		ExpectedHeadMessageID *string          `json:"expected_head_message_id"`
		UserMessage           *hashUserMessage `json:"user_message,omitempty"`
		RetryOfMessageID      string           `json:"retry_of_message_id,omitempty"`
		AssistantMessageID    string           `json:"assistant_message_id"`
		Attachments           []hashAttachment `json:"attachments,omitempty"`
	}{
		ConversationID:        normalized.ConversationID,
		Model:                 normalized.Model,
		ReasoningMode:         normalized.ReasoningMode,
		ReasoningEffort:       normalized.ReasoningEffort,
		ExpectedHeadMessageID: normalized.ExpectedHeadMessageID,
		UserMessage:           hashUser,
		RetryOfMessageID:      normalized.RetryOfMessageID,
		AssistantMessageID:    normalized.AssistantMessageID,
		Attachments:           hashAttachments,
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
	activities, activitiesErr := normalizeChatMessageActivities(normalized.Activities)
	if activitiesErr != nil {
		return activitiesErr
	}
	normalized.Activities = activities
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
	activities, activitiesErr := normalizeChatMessageActivities(normalized.Activities)
	if activitiesErr != nil {
		return activitiesErr
	}
	normalized.Activities = activities
	if !validChatAttemptID(normalized.AttemptID) ||
		!validChatHistoryPublicID(normalized.AssistantMessageID) ||
		normalized.CheckpointSeq <= 0 {
		return ErrChatHistoryInvalid
	}
	return s.repo.CheckpointCompletion(ctx, userID, &normalized)
}

func normalizeChatMessageActivities(input []ChatMessageActivity) ([]ChatMessageActivity, error) {
	if len(input) == 0 {
		return nil, nil
	}
	if len(input) > maxChatMessageActivitiesPerSnapshot {
		return nil, ErrChatHistoryInvalid
	}
	normalized := make([]ChatMessageActivity, len(input))
	seen := make(map[string]struct{}, len(input))
	totalTextBytes := 0
	for i := range input {
		activity := input[i]
		activity.ResponseID = strings.TrimSpace(activity.ResponseID)
		activity.Source = strings.TrimSpace(activity.Source)
		activity.ActivityType = strings.TrimSpace(activity.ActivityType)
		activity.ItemID = strings.TrimSpace(activity.ItemID)
		activity.Status = strings.TrimSpace(activity.Status)
		activity.ReasoningMode = strings.TrimSpace(activity.ReasoningMode)
		activity.ReasoningEffort = strings.TrimSpace(activity.ReasoningEffort)
		if activity.Source != ChatMessageActivitySourceOpenAIResponses ||
			activity.ActivityType != ChatMessageActivityTypeReasoningSummary ||
			activity.ItemID == "" ||
			len(activity.ResponseID) > 128 ||
			len(activity.ItemID) > 128 ||
			activity.OutputIndex < 0 ||
			activity.SummaryIndex < 0 ||
			activity.SortOrder <= 0 ||
			activity.SequenceStart < 0 ||
			activity.SequenceEnd < activity.SequenceStart ||
			len(activity.ReasoningMode) > 32 ||
			len(activity.ReasoningEffort) > 32 ||
			!validChatMessageActivityStatus(activity.Status) {
			return nil, ErrChatHistoryInvalid
		}
		totalTextBytes += len(activity.Text)
		if totalTextBytes > maxChatMessageActivityTextBytes {
			return nil, ErrChatHistoryInvalid
		}
		if activity.StartedAt.IsZero() {
			activity.StartedAt = time.Now().UTC()
		} else {
			activity.StartedAt = activity.StartedAt.UTC()
		}
		if activity.CompletedAt != nil {
			completedAt := activity.CompletedAt.UTC()
			if completedAt.Before(activity.StartedAt) {
				return nil, ErrChatHistoryInvalid
			}
			activity.CompletedAt = &completedAt
		}
		if len(activity.Metadata) == 0 || string(activity.Metadata) == "null" {
			activity.Metadata = json.RawMessage(`{}`)
		} else if len(activity.Metadata) > maxChatMessageActivityMetadataBytes ||
			!json.Valid(activity.Metadata) ||
			strings.TrimSpace(string(activity.Metadata))[0] != '{' {
			return nil, ErrChatHistoryInvalid
		} else {
			activity.Metadata = append(json.RawMessage(nil), activity.Metadata...)
		}
		key := strings.Join([]string{
			activity.ResponseID,
			activity.ItemID,
			strconv.Itoa(activity.OutputIndex),
			strconv.Itoa(activity.SummaryIndex),
		}, "\x00")
		if _, exists := seen[key]; exists {
			return nil, ErrChatHistoryInvalid
		}
		seen[key] = struct{}{}
		normalized[i] = activity
	}
	return normalized, nil
}

func validChatMessageActivityStatus(status string) bool {
	switch status {
	case ChatMessageActivityStatusInProgress,
		ChatMessageActivityStatusCompleted,
		ChatMessageActivityStatusIncomplete,
		ChatMessageActivityStatusFailed,
		ChatMessageActivityStatusInterrupted,
		ChatMessageActivityStatusStopped,
		ChatMessageActivityStatusDisconnected:
		return true
	default:
		return false
	}
}

// TerminalizeChatMessageActivities returns a detached authoritative terminal
// snapshot without changing summary text or stable coordinates.
func TerminalizeChatMessageActivities(
	input []ChatMessageActivity,
	status string,
	lastEvent string,
	completedAt time.Time,
) []ChatMessageActivity {
	if len(input) == 0 || status == ChatMessageActivityStatusInProgress || !validChatMessageActivityStatus(status) {
		return append([]ChatMessageActivity(nil), input...)
	}
	if completedAt.IsZero() {
		completedAt = time.Now().UTC()
	} else {
		completedAt = completedAt.UTC()
	}
	result := make([]ChatMessageActivity, len(input))
	for i := range input {
		activity := input[i]
		activity.Status = status
		if activity.CompletedAt == nil {
			value := completedAt
			activity.CompletedAt = &value
		}
		metadata := make(map[string]any)
		if len(activity.Metadata) > 0 {
			_ = json.Unmarshal(activity.Metadata, &metadata)
		}
		if lastEvent != "" {
			metadata["last_event"] = lastEvent
		}
		encoded, err := json.Marshal(metadata)
		if err != nil {
			encoded = []byte(`{}`)
		}
		activity.Metadata = encoded
		result[i] = activity
	}
	return result
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
	case "stopped":
		return ChatMessageDeliveryStopped
	case "streaming", "interrupted", "error":
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
