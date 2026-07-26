package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	maxChatRequestBytes              = 2 << 20
	chatSettlementFailureCode        = "CHAT_SETTLEMENT_FAILED"
	chatSettlementFailureReason      = "Chat usage settlement could not be completed"
	chatSettlementReceiptReadTimeout = 5 * time.Second
)

type chatApplication interface {
	ListModels(ctx context.Context, userID int64) (*service.ChatModelsResult, error)
	ResolvePrincipal(ctx context.Context, userID int64, model string) (*service.APIKey, error)
}

type chatCompletionDelegator interface {
	ChatCompletions(c *gin.Context)
}

type chatAttemptLeaseManager interface {
	StartLeaseHeartbeat(userID int64, attemptID string) func()
}

type webChatSettlementResolution uint8

const (
	webChatSettlementUnknown webChatSettlementResolution = iota
	webChatSettlementHasEvidence
	webChatSettlementNoBilling
)

type ChatHandler struct {
	chat     chatApplication
	attempts chatAttemptLeaseManager
	receipts *service.BillingReceiptService
	history  *service.ChatHistoryService
	gateway  chatCompletionDelegator
}

func NewChatHandler(
	chat *service.ChatService,
	attempts *service.ChatAttemptService,
	receipts *service.BillingReceiptService,
	gateway *OpenAIGatewayHandler,
) *ChatHandler {
	return &ChatHandler{
		chat:     chat,
		attempts: attempts,
		receipts: receipts,
		gateway:  gateway,
	}
}

func ProvideChatHandler(
	chat *service.ChatService,
	attempts *service.ChatAttemptService,
	receipts *service.BillingReceiptService,
	history *service.ChatHistoryService,
	gateway *OpenAIGatewayHandler,
) *ChatHandler {
	handler := NewChatHandler(chat, attempts, receipts, gateway)
	handler.history = history
	return handler
}

func (h *ChatHandler) Models(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if h == nil || h.chat == nil {
		response.InternalError(c, "Chat service is unavailable")
		return
	}

	result, err := h.chat.ListModels(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *ChatHandler) Receipt(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if h == nil || h.receipts == nil {
		response.InternalError(c, "Chat receipt service is unavailable")
		return
	}

	receipt, err := h.receipts.GetUserWebChatReceipt(
		c.Request.Context(),
		subject.UserID,
		c.Param("receipt_id"),
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	payload := chatReceiptFromService(receipt)
	if payload.Status == service.BillingReceiptStatusPending {
		response.Accepted(c, payload)
		return
	}
	response.Success(c, payload)
}

type chatReceiptResponse struct {
	ReceiptID           string     `json:"receipt_id"`
	Status              string     `json:"status"`
	UsageLogID          *int64     `json:"usage_log_id,omitempty"`
	Model               string     `json:"model"`
	InputTokens         int        `json:"input_tokens"`
	OutputTokens        int        `json:"output_tokens"`
	CacheCreationTokens int        `json:"cache_creation_tokens"`
	CacheReadTokens     int        `json:"cache_read_tokens"`
	TotalTokens         int        `json:"total_tokens"`
	GrossCost           float64    `json:"gross_cost"`
	ChargedAmount       float64    `json:"charged_amount"`
	BillingType         int8       `json:"billing_type"`
	BalanceBefore       *float64   `json:"balance_before,omitempty"`
	BalanceAfter        *float64   `json:"balance_after,omitempty"`
	Overdraft           bool       `json:"overdraft"`
	FailureCode         *string    `json:"failure_code,omitempty"`
	FailureReason       *string    `json:"failure_reason,omitempty"`
	CreatedAt           *time.Time `json:"created_at,omitempty"`
}

func chatReceiptFromService(receipt *service.BillingReceipt) chatReceiptResponse {
	if receipt == nil {
		return chatReceiptResponse{}
	}
	requestID := strings.TrimSpace(receipt.RequestID)
	requestID = strings.TrimPrefix(requestID, "client:")
	model := strings.TrimSpace(receipt.Model)
	if model == "" {
		model = strings.TrimSpace(receipt.RequestedModel)
	}
	createdAt := receipt.CreatedAt
	return chatReceiptResponse{
		ReceiptID:           requestID,
		Status:              receipt.Status,
		UsageLogID:          receipt.UsageLogID,
		Model:               model,
		InputTokens:         receipt.InputTokens,
		OutputTokens:        receipt.OutputTokens,
		CacheCreationTokens: receipt.CacheCreationTokens,
		CacheReadTokens:     receipt.CacheReadTokens,
		TotalTokens: receipt.InputTokens +
			receipt.OutputTokens +
			receipt.CacheCreationTokens +
			receipt.CacheReadTokens,
		GrossCost:     receipt.GrossAmount,
		ChargedAmount: receipt.ChargedAmount,
		BillingType:   receipt.BillingType,
		BalanceBefore: receipt.BalanceBefore,
		BalanceAfter:  receipt.BalanceAfter,
		Overdraft:     receipt.Overdraft,
		FailureCode:   receipt.FailureCode,
		FailureReason: receipt.FailureReason,
		CreatedAt:     &createdAt,
	}
}

func (h *ChatHandler) Completions(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if h == nil || h.chat == nil || h.history == nil || h.gateway == nil {
		response.InternalError(c, "Chat service is unavailable")
		return
	}
	if !isJSONContentType(c.GetHeader("Content-Type")) {
		response.ErrorWithDetails(c, http.StatusUnsupportedMediaType, "Content-Type must be application/json", "UNSUPPORTED_MEDIA_TYPE", nil)
		return
	}

	req, err := decodeWebChatCompletionRequest(c)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			response.ErrorWithDetails(c, http.StatusRequestEntityTooLarge, "Chat request is too large", "REQUEST_TOO_LARGE", nil)
			return
		}
		response.ErrorWithDetails(c, http.StatusBadRequest, "Invalid chat request", "INVALID_CHAT_REQUEST", nil)
		return
	}

	clientRequestID, _ := c.Request.Context().Value(ctxkey.ClientRequestID).(string)
	attemptID := strings.TrimSpace(c.GetHeader("X-Chat-Attempt-ID"))
	prepared, err := h.history.PrepareCompletion(
		c.Request.Context(),
		subject.UserID,
		attemptID,
		clientRequestID,
		&service.PrepareChatCompletionInput{
			ConversationID:        req.ConversationID,
			Model:                 req.Model,
			ExpectedHeadMessageID: req.ExpectedHeadMessageID,
			UserMessage:           chatCompletionUserMessageFromRequest(req.UserMessage),
			RetryOfMessageID:      req.RetryOfMessageID,
			AssistantMessageID:    req.AssistantMessageID,
		},
	)
	if prepared != nil && strings.TrimSpace(prepared.ClientRequestID) != "" {
		c.Header("X-Client-Request-ID", strings.TrimSpace(prepared.ClientRequestID))
	}
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if prepared == nil || !prepared.Claimed {
		response.InternalError(c, "Failed to initialize chat request")
		return
	}
	stopHeartbeat := func() {}
	if h.attempts != nil {
		stopHeartbeat = h.attempts.StartLeaseHeartbeat(subject.UserID, attemptID)
	}
	defer stopHeartbeat()

	messages := make([]webChatMessage, len(prepared.Messages))
	for i := range prepared.Messages {
		messages[i] = webChatMessage{
			Role:    prepared.Messages[i].Role,
			Content: prepared.Messages[i].Content,
		}
	}
	body, err := json.Marshal(openAIWebChatCompletionRequest{
		Model:    req.Model,
		Messages: messages,
		Stream:   true,
	})
	if err != nil {
		response.InternalError(c, "Failed to initialize chat request")
		h.finalizeChatCompletion(
			subject.UserID,
			attemptID,
			req.AssistantMessageID,
			c.Writer.Status(),
			deliveredChatStreamSnapshot{},
		)
		return
	}

	principal, err := h.chat.ResolvePrincipal(c.Request.Context(), subject.UserID, req.Model)
	if err != nil {
		response.ErrorFrom(c, err)
		h.finalizeChatCompletion(
			subject.UserID,
			attemptID,
			req.AssistantMessageID,
			c.Writer.Status(),
			deliveredChatStreamSnapshot{},
		)
		return
	}
	if principal == nil || principal.UserID != subject.UserID || !middleware2.BindChatPrincipalContext(c, principal) {
		response.InternalError(c, "Failed to initialize chat request")
		h.finalizeChatCompletion(
			subject.UserID,
			attemptID,
			req.AssistantMessageID,
			c.Writer.Status(),
			deliveredChatStreamSnapshot{},
		)
		return
	}

	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	c.Request.ContentLength = int64(len(body))
	c.Request.Header.Set("Content-Type", "application/json")

	requestContext := c.Request.Context()
	usageBarrier := newWebChatUsageBarrier()
	c.Request = c.Request.WithContext(
		withWebChatUsageBarrier(requestContext, usageBarrier),
	)
	checkpoint := func(sequence int64, content string) error {
		checkpointContext, cancel := context.WithTimeout(
			context.WithoutCancel(requestContext),
			3*time.Second,
		)
		defer cancel()
		return h.history.CheckpointCompletion(
			checkpointContext,
			subject.UserID,
			&service.CheckpointChatCompletionInput{
				AttemptID:          attemptID,
				AssistantMessageID: req.AssistantMessageID,
				Content:            content,
				CheckpointSeq:      sequence,
			},
		)
	}
	observer := newDeliveredChatStreamObserver(checkpoint)
	observer.bindCancellation(requestContext.Done())
	originalWriter := c.Writer
	c.Writer = &deliveredChatResponseWriter{
		ResponseWriter: originalWriter,
		observer:       observer,
	}
	h.gateway.ChatCompletions(c)
	usageResult := usageBarrier.SealAndWait()
	statusCode := c.Writer.Status()
	c.Writer = originalWriter
	if requestContext.Err() != nil {
		observer.Freeze()
	}
	if _, checkpointErr := observer.ForceCheckpoint(); checkpointErr != nil {
		slog.Warn("flush final chat history checkpoint failed",
			"user_id", subject.UserID,
			"attempt_id", attemptID,
			"error", checkpointErr,
		)
	}
	streamSnapshot := observer.Snapshot()
	if usageErr := usageResult.Err(); usageErr != nil {
		resolution, receipt, receiptErr := h.resolveWebChatSettlement(
			requestContext,
			subject.UserID,
			prepared.ClientRequestID,
		)
		switch resolution {
		case webChatSettlementHasEvidence:
			slog.Warn("web chat usage producer failed but canonical settlement evidence exists",
				"user_id", subject.UserID,
				"attempt_id", attemptID,
				"receipt_id", strings.TrimSpace(prepared.ClientRequestID),
				"receipt_status", receipt.Status,
				"producer_count", usageResult.ProducerCount,
				"error", usageErr,
			)
		case webChatSettlementNoBilling:
			streamSnapshot.ErrorCode = chatSettlementFailureCode
			streamSnapshot.ErrorMessage = chatSettlementFailureReason
			slog.Error("web chat usage settlement failed without billing evidence",
				"user_id", subject.UserID,
				"attempt_id", attemptID,
				"receipt_id", strings.TrimSpace(prepared.ClientRequestID),
				"producer_count", usageResult.ProducerCount,
				"error", usageErr,
			)
		default:
			slog.Error("web chat usage settlement outcome is unknown; leaving attempt processing",
				"user_id", subject.UserID,
				"attempt_id", attemptID,
				"receipt_id", strings.TrimSpace(prepared.ClientRequestID),
				"producer_count", usageResult.ProducerCount,
				"producer_error", usageErr,
				"receipt_error", receiptErr,
			)
			return
		}
	}
	h.finalizeChatCompletion(
		subject.UserID,
		attemptID,
		req.AssistantMessageID,
		statusCode,
		streamSnapshot,
	)
}

func (h *ChatHandler) resolveWebChatSettlement(
	requestContext context.Context,
	userID int64,
	receiptID string,
) (webChatSettlementResolution, *service.BillingReceipt, error) {
	if h == nil || h.receipts == nil || userID <= 0 || strings.TrimSpace(receiptID) == "" {
		return webChatSettlementUnknown, nil, errors.New("chat receipt service is unavailable")
	}
	base := context.Background()
	if requestContext != nil {
		base = context.WithoutCancel(requestContext)
	}
	ctx, cancel := context.WithTimeout(base, chatSettlementReceiptReadTimeout)
	defer cancel()

	receipt, err := h.receipts.GetUserWebChatReceipt(ctx, userID, receiptID)
	if err != nil || receipt == nil {
		if err == nil {
			err = errors.New("chat settlement receipt is nil")
		}
		return webChatSettlementUnknown, receipt, err
	}

	hasCanonicalEvidence := receipt.ID > 0 || receipt.UsageLogID != nil
	if hasCanonicalEvidence {
		switch receipt.Status {
		case service.BillingReceiptStatusCharged,
			service.BillingReceiptStatusSubscription,
			service.BillingReceiptStatusNotCharged:
			return webChatSettlementHasEvidence, receipt, nil
		default:
			return webChatSettlementUnknown, receipt, errors.New("canonical chat settlement has an invalid status")
		}
	}

	switch receipt.Status {
	case service.BillingReceiptStatusPending,
		service.BillingReceiptStatusFailed,
		service.BillingReceiptStatusNotCharged:
		return webChatSettlementNoBilling, receipt, nil
	default:
		return webChatSettlementUnknown, receipt, errors.New("chat settlement receipt has no canonical evidence")
	}
}

func (h *ChatHandler) finalizeChatCompletion(
	userID int64,
	attemptID string,
	assistantMessageID string,
	statusCode int,
	stream deliveredChatStreamSnapshot,
) {
	if h == nil || h.history == nil {
		return
	}
	input := &service.FinalizeChatCompletionInput{
		AttemptID:          attemptID,
		AssistantMessageID: assistantMessageID,
		Content:            stream.Content,
		CheckpointSeq:      stream.CheckpointSeq + 1,
		HTTPStatus:         statusCode,
		FinishReason:       stream.FinishReason,
	}
	switch {
	case statusCode >= http.StatusBadRequest || stream.ErrorCode != "":
		input.DeliveryStatus = service.ChatMessageDeliveryError
		input.AttemptStatus = service.ChatAttemptStatusFailed
		input.ErrorCode = stream.ErrorCode
		input.ErrorMessage = stream.ErrorMessage
		if input.ErrorCode == "" {
			input.ErrorCode = "HTTP_" + strconv.Itoa(statusCode)
		}
		if input.ErrorMessage == "" {
			input.ErrorMessage = http.StatusText(statusCode)
		}
		if input.ErrorMessage == "" {
			input.ErrorMessage = "Chat request failed"
		}
	case stream.Done:
		input.DeliveryStatus = service.ChatMessageDeliveryCompleted
		input.AttemptStatus = service.ChatAttemptStatusCompleted
	default:
		input.DeliveryStatus = service.ChatMessageDeliveryInterrupted
		input.AttemptStatus = service.ChatAttemptStatusInterrupted
		if input.FinishReason == "" {
			input.FinishReason = "interrupted"
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := h.history.FinalizeCompletion(ctx, userID, input); err != nil {
		slog.Warn("finalize chat history attempt failed",
			"user_id", userID,
			"attempt_id", attemptID,
			"status", input.AttemptStatus,
			"error", err,
		)
	}
}

type webChatCompletionRequest struct {
	ConversationID        string                        `json:"conversation_id"`
	Model                 string                        `json:"model"`
	ExpectedHeadMessageID *string                       `json:"expected_head_message_id"`
	UserMessage           *webChatCompletionUserMessage `json:"user_message,omitempty"`
	RetryOfMessageID      string                        `json:"retry_of_message_id,omitempty"`
	AssistantMessageID    string                        `json:"assistant_message_id"`
}

type webChatCompletionUserMessage struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

type openAIWebChatCompletionRequest struct {
	Model    string           `json:"model"`
	Messages []webChatMessage `json:"messages"`
	Stream   bool             `json:"stream"`
}

type webChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func decodeWebChatCompletionRequest(c *gin.Context) (*webChatCompletionRequest, error) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxChatRequestBytes)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	var req webChatCompletionRequest
	if err := decoder.Decode(&req); err != nil {
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, err
	}

	return &req, nil
}

func chatCompletionUserMessageFromRequest(
	message *webChatCompletionUserMessage,
) *service.ChatCompletionHistoryUserMessage {
	if message == nil {
		return nil
	}
	return &service.ChatCompletionHistoryUserMessage{
		ID:      message.ID,
		Content: message.Content,
	}
}

func isJSONContentType(value string) bool {
	mediaType, _, err := mime.ParseMediaType(value)
	return err == nil && strings.EqualFold(mediaType, "application/json")
}
