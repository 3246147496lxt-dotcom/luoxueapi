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
	chatReceiptIDHeader              = "X-Chat-Receipt-ID"
	chatSettlementFailureReason      = "Chat usage settlement could not be completed"
	chatSettlementReceiptReadTimeout = 5 * time.Second
	transcriptionForcedCloseKey      = "web_chat_transcription_forced_connection_close"
)

type chatApplication interface {
	ListModels(ctx context.Context, userID int64) (*service.ChatModelsResult, error)
	ResolvePrincipal(ctx context.Context, userID int64, model string) (*service.ChatPrincipal, error)
}

type chatCapabilitiesApplication interface {
	Capabilities(ctx context.Context) (*service.ChatCapabilitiesResult, error)
}

type chatReasoningApplication interface {
	NormalizeWebChatReasoning(ctx context.Context, userID int64, model string, options service.WebChatReasoningOptions) (service.WebChatReasoningOptions, error)
}

type chatReasoningPrincipalApplication interface {
	ResolveWebChatPrincipal(ctx context.Context, userID int64, model string, options service.WebChatReasoningOptions) (*service.ChatPrincipal, error)
}

type chatVisionApplication interface {
	SupportsVision(ctx context.Context, userID int64, model string) (bool, error)
}

type chatCompletionDelegator interface {
	ChatCompletions(c *gin.Context)
}

type chatTranscriptionApplication interface {
	ResolveTranscriptionPrincipal(ctx context.Context, userID int64) (*service.APIKey, error)
}

type chatTranscriptionDelegator interface {
	AdmitTranscription(c *gin.Context) (release func(), admitted bool)
	Transcriptions(c *gin.Context)
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
	chat        chatApplication
	attempts    chatAttemptLeaseManager
	receipts    *service.BillingReceiptService
	history     *service.ChatHistoryService
	attachments *service.ChatAttachmentService
	library     *service.LibraryService
	gateway     chatCompletionDelegator
	projects    *service.ProjectService
}

func (h *ChatHandler) SetProjectService(projects *service.ProjectService) { h.projects = projects }

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
	attachments *service.ChatAttachmentService,
	library *service.LibraryService,
	projects *service.ProjectService,
	gateway *OpenAIGatewayHandler,
) *ChatHandler {
	handler := NewChatHandler(chat, attempts, receipts, gateway)
	handler.history = history
	handler.attachments = attachments
	handler.library = library
	handler.projects = projects
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

func (h *ChatHandler) Capabilities(c *gin.Context) {
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

	application, ok := h.chat.(chatCapabilitiesApplication)
	if !ok || application == nil {
		response.InternalError(c, "Chat capabilities service is unavailable")
		return
	}
	result, err := application.Capabilities(c.Request.Context())
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

func (h *ChatHandler) Transcriptions(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	forceCloseUntilTranscriptionBodyRead(c)
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if h == nil {
		response.ErrorWithDetails(c, http.StatusServiceUnavailable, "Voice transcription is temporarily unavailable", "TRANSCRIPTION_UNAVAILABLE", nil)
		return
	}
	application, appOK := h.chat.(chatTranscriptionApplication)
	delegator, gatewayOK := h.gateway.(chatTranscriptionDelegator)
	if !appOK || application == nil || !gatewayOK || delegator == nil {
		response.ErrorWithDetails(c, http.StatusServiceUnavailable, "Voice transcription is temporarily unavailable", "TRANSCRIPTION_UNAVAILABLE", nil)
		return
	}
	releaseAdmission, admitted := delegator.AdmitTranscription(c)
	if !admitted {
		return
	}
	if releaseAdmission == nil {
		response.ErrorWithDetails(c, http.StatusServiceUnavailable, "Voice transcription is temporarily unavailable", "TRANSCRIPTION_UNAVAILABLE", nil)
		return
	}
	defer releaseAdmission()
	principal, err := application.ResolveTranscriptionPrincipal(c.Request.Context(), subject.UserID)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(c.Request.Context().Err(), context.DeadlineExceeded) {
			response.ErrorWithDetails(c, http.StatusGatewayTimeout, "Voice transcription timed out", "TRANSCRIPTION_TIMEOUT", nil)
			return
		}
		response.ErrorFrom(c, err)
		return
	}
	if err := c.Request.Context().Err(); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			response.ErrorWithDetails(c, http.StatusGatewayTimeout, "Voice transcription timed out", "TRANSCRIPTION_TIMEOUT", nil)
		}
		return
	}
	if principal == nil || principal.UserID != subject.UserID || !middleware2.BindChatPrincipalContext(c, principal) {
		response.ErrorWithDetails(c, http.StatusServiceUnavailable, "Voice transcription is temporarily unavailable", "TRANSCRIPTION_UNAVAILABLE", nil)
		return
	}
	delegator.Transcriptions(c)
}

// A rejected HTTP/1 transcription request may still have a large, slow body.
// Prevent net/http from draining that unread body after the handler returns.
// Once multipart parsing reaches EOF, the gateway explicitly restores reuse.
func forceCloseUntilTranscriptionBodyRead(c *gin.Context) {
	if c == nil || c.Request == nil {
		return
	}
	if !c.Request.Close {
		c.Set(transcriptionForcedCloseKey, true)
	}
	c.Request.Close = true
	c.Header("Connection", "close")
}

func allowConnectionReuseAfterTranscriptionBodyRead(c *gin.Context) {
	if c == nil || c.Request == nil {
		return
	}
	forced, ok := c.Get(transcriptionForcedCloseKey)
	if !ok || forced != true {
		return
	}
	c.Request.Close = false
	c.Writer.Header().Del("Connection")
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
	libraryIDs, err := normalizeWebChatCompletionAttachments(req)
	if err != nil {
		response.ErrorWithDetails(c, http.StatusBadRequest, "Invalid chat request", "INVALID_CHAT_REQUEST", nil)
		return
	}
	if len(libraryIDs) > 0 {
		if h.library == nil {
			response.ErrorFrom(c, service.ErrLibraryUnavailable)
			return
		}
		if _, err = h.library.Resolve(c.Request.Context(), subject.UserID, libraryIDs); err != nil {
			response.ErrorFrom(c, err)
			return
		}
	}
	reasoning, ok := h.chat.(chatReasoningApplication)
	if !ok || reasoning == nil {
		response.InternalError(c, "Chat reasoning service is unavailable")
		return
	}
	normalizedReasoning, err := reasoning.NormalizeWebChatReasoning(
		c.Request.Context(),
		subject.UserID,
		req.Model,
		service.WebChatReasoningOptions{Mode: req.ReasoningMode, Effort: req.ReasoningEffort},
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	req.ReasoningMode = normalizedReasoning.Mode
	req.ReasoningEffort = normalizedReasoning.Effort
	visionSupported := false
	visionChecked := false
	if req.UserMessage != nil && len(req.UserMessage.AttachmentIDs) > 0 {
		if h.attachments == nil {
			response.ErrorFrom(c, service.ErrChatAttachmentUnavailable)
			return
		}
		_, requiresVision, resolveErr := h.attachments.ResolveSelection(c.Request.Context(), subject.UserID, req.UserMessage.AttachmentIDs)
		if resolveErr != nil {
			response.ErrorFrom(c, resolveErr)
			return
		}
		if requiresVision {
			visionApp, ok := h.chat.(chatVisionApplication)
			if !ok {
				response.ErrorFrom(c, service.ErrChatAttachmentVisionUnsupported)
				return
			}
			visionSupported, err = visionApp.SupportsVision(c.Request.Context(), subject.UserID, req.Model)
			visionChecked = true
			if err != nil {
				response.ErrorFrom(c, err)
				return
			}
			if !visionSupported {
				response.ErrorFrom(c, service.ErrChatAttachmentVisionUnsupported)
				return
			}
		}
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
			ReasoningMode:         req.ReasoningMode,
			ReasoningEffort:       req.ReasoningEffort,
			ExpectedHeadMessageID: req.ExpectedHeadMessageID,
			UserMessage:           chatCompletionUserMessageFromRequest(req.UserMessage),
			RetryOfMessageID:      req.RetryOfMessageID,
			AssistantMessageID:    req.AssistantMessageID,
		},
	)
	if prepared != nil && strings.TrimSpace(prepared.ClientRequestID) != "" {
		receiptID := strings.TrimSpace(prepared.ClientRequestID)
		c.Header("X-Client-Request-ID", receiptID)
		// X-Client-Request-ID is a request-correlation header written for every
		// response by middleware. Keep a distinct header for the stronger
		// contract that this chat attempt was persisted (or is a replay of one).
		if err == nil || errors.Is(err, service.ErrChatAttemptAlreadySubmitted) {
			c.Header(chatReceiptIDHeader, receiptID)
		}
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

	messages := make([]webChatMessage, 0, len(prepared.Messages))
	requiresVision := false
	if h.attachments != nil {
		modelMessages, modelRequiresVision, buildErr := h.attachments.BuildModelContext(c.Request.Context(), prepared.Messages)
		if buildErr != nil {
			response.ErrorFrom(c, buildErr)
			h.finalizeChatCompletion(subject.UserID, attemptID, req.AssistantMessageID, c.Writer.Status(), deliveredChatStreamSnapshot{})
			return
		}
		requiresVision = modelRequiresVision
		for i := range modelMessages {
			messages = append(messages, webChatMessage{Role: modelMessages[i].Role, Content: modelMessages[i].Content})
		}
	} else {
		for i := range prepared.Messages {
			content, _ := json.Marshal(prepared.Messages[i].Content)
			messages = append(messages, webChatMessage{Role: prepared.Messages[i].Role, Content: content})
		}
	}
	if requiresVision && !visionChecked {
		visionApp, ok := h.chat.(chatVisionApplication)
		if !ok {
			response.ErrorFrom(c, service.ErrChatAttachmentVisionUnsupported)
			h.finalizeChatCompletion(subject.UserID, attemptID, req.AssistantMessageID, c.Writer.Status(), deliveredChatStreamSnapshot{})
			return
		}
		visionSupported, err = visionApp.SupportsVision(c.Request.Context(), subject.UserID, req.Model)
		if err != nil {
			response.ErrorFrom(c, err)
			h.finalizeChatCompletion(subject.UserID, attemptID, req.AssistantMessageID, c.Writer.Status(), deliveredChatStreamSnapshot{})
			return
		}
	}
	if requiresVision && !visionSupported {
		response.ErrorFrom(c, service.ErrChatAttachmentVisionUnsupported)
		h.finalizeChatCompletion(subject.UserID, attemptID, req.AssistantMessageID, c.Writer.Status(), deliveredChatStreamSnapshot{})
		return
	}
	body, err := json.Marshal(openAIWebChatCompletionRequest{
		Model:    req.Model,
		Messages: messages,
		Reasoning: &webChatReasoningRequest{
			Mode: req.ReasoningMode,
			// Pro is a mode and must not carry an effort field to the
			// upstream request. The normalized low effort remains an
			// internal capability/scheduling default only.
			Effort: func() string {
				if req.ReasoningMode == service.WebChatReasoningModePro {
					return ""
				}
				return req.ReasoningEffort
			}(),
			Summary: "auto",
		},
		Stream: true,
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

	reasoningPrincipal, ok := h.chat.(chatReasoningPrincipalApplication)
	if !ok || reasoningPrincipal == nil {
		response.InternalError(c, "Chat reasoning principal service is unavailable")
		h.finalizeChatCompletion(
			subject.UserID,
			attemptID,
			req.AssistantMessageID,
			c.Writer.Status(),
			deliveredChatStreamSnapshot{},
		)
		return
	}
	resolvedPrincipal, err := reasoningPrincipal.ResolveWebChatPrincipal(
		c.Request.Context(),
		subject.UserID,
		req.Model,
		service.WebChatReasoningOptions{Mode: req.ReasoningMode, Effort: req.ReasoningEffort},
	)
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
	if resolvedPrincipal == nil || resolvedPrincipal.APIKey == nil ||
		resolvedPrincipal.APIKey.UserID != subject.UserID ||
		!middleware2.BindChatPrincipalBillingContext(c, resolvedPrincipal.APIKey, resolvedPrincipal.Subscription) {
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
	observer := newDeliveredChatStreamObserver()
	observer.setReasoningMetadata(req.ReasoningMode, req.ReasoningEffort)
	observer.bindSnapshotCheckpoint(func(checkpoint deliveredChatStreamCheckpoint) error {
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
				Content:            checkpoint.Content,
				CheckpointSeq:      checkpoint.Sequence,
				Activities:         checkpoint.Activities,
			},
		)
	})
	observer.bindCancellation(requestContext.Done())
	originalWriter := c.Writer
	c.Writer = &deliveredChatResponseWriter{
		ResponseWriter: originalWriter,
		observer:       observer,
	}
	service.SetWebChatReasoningOptions(c, service.WebChatReasoningOptions{
		Mode:   req.ReasoningMode,
		Effort: req.ReasoningEffort,
	})
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
			// Keep the settlement failure on the attempt/receipt axis while the
			// independently observed stream still determines message delivery.
			if deliveredChatStreamCompletedSuccessfully(streamSnapshot) &&
				streamSnapshot.ErrorCode == "" {
				streamSnapshot.ErrorCode = service.ChatAttemptFailureCodeSettlement
				streamSnapshot.ErrorMessage = chatSettlementFailureReason
			}
			slog.Error("web chat usage settlement failed without billing evidence",
				"user_id", subject.UserID,
				"attempt_id", attemptID,
				"receipt_id", strings.TrimSpace(prepared.ClientRequestID),
				"delivery_done", streamSnapshot.Done,
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
		Activities:         stream.Activities,
	}
	switch {
	case statusCode < http.StatusBadRequest &&
		deliveredChatStreamCompletedSuccessfully(stream) &&
		stream.ErrorCode == service.ChatAttemptFailureCodeSettlement:
		// The answer was delivered successfully. Keep the message completed,
		// while the failed attempt retains settlement evidence for its receipt.
		input.DeliveryStatus = service.ChatMessageDeliveryCompleted
		input.AttemptStatus = service.ChatAttemptStatusFailed
		input.ErrorCode = stream.ErrorCode
		input.ErrorMessage = stream.ErrorMessage
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
	case stream.ResponsesTerminalSeen &&
		stream.ResponsesTerminalStatus == service.ChatMessageActivityStatusFailed:
		input.DeliveryStatus = service.ChatMessageDeliveryError
		input.AttemptStatus = service.ChatAttemptStatusFailed
		input.ErrorCode = "OPENAI_RESPONSE_FAILED"
		input.ErrorMessage = "OpenAI response failed"
	case stream.ResponsesTerminalSeen &&
		stream.ResponsesTerminalStatus == service.ChatMessageActivityStatusIncomplete:
		input.DeliveryStatus = service.ChatMessageDeliveryPartial
		input.AttemptStatus = service.ChatAttemptStatusInterrupted
		if input.FinishReason == "" {
			input.FinishReason = "incomplete"
		}
	case stream.Done:
		input.DeliveryStatus = service.ChatMessageDeliveryCompleted
		input.AttemptStatus = service.ChatAttemptStatusCompleted
	default:
		input.DeliveryStatus = service.ChatMessageDeliveryInterrupted
		input.AttemptStatus = service.ChatAttemptStatusInterrupted
		if !stream.ResponsesTerminalSeen {
			input.Activities = service.TerminalizeChatMessageActivities(
				input.Activities,
				service.ChatMessageActivityStatusDisconnected,
				"client_disconnected",
				time.Now(),
			)
		}
		if input.FinishReason == "" {
			input.FinishReason = "disconnected"
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

func deliveredChatStreamCompletedSuccessfully(stream deliveredChatStreamSnapshot) bool {
	if !stream.Done {
		return false
	}
	return !stream.ResponsesTerminalSeen ||
		stream.ResponsesTerminalStatus == service.ChatMessageActivityStatusCompleted
}

type webChatCompletionRequest struct {
	ConversationID        string                                 `json:"conversation_id"`
	Model                 string                                 `json:"model"`
	ReasoningMode         string                                 `json:"reasoning_mode,omitempty"`
	ReasoningEffort       string                                 `json:"reasoning_effort,omitempty"`
	Reasoning             *webChatReasoningRequest               `json:"reasoning,omitempty"`
	ExpectedHeadMessageID *string                                `json:"expected_head_message_id"`
	UserMessage           *webChatCompletionUserMessage          `json:"user_message,omitempty"`
	Attachments           []webChatCompletionAttachmentReference `json:"attachments,omitempty"`
	RetryOfMessageID      string                                 `json:"retry_of_message_id,omitempty"`
	AssistantMessageID    string                                 `json:"assistant_message_id"`
}

// webChatReasoningRequest is the canonical Responses-shaped selector sent by
// the Chat UI. The legacy flat fields above remain accepted for old clients,
// but are normalized into this same trusted contract before forwarding.
type webChatReasoningRequest struct {
	Mode    string `json:"mode,omitempty"`
	Effort  string `json:"effort,omitempty"`
	Summary string `json:"summary,omitempty"`
}

type webChatCompletionUserMessage struct {
	ID            string                                 `json:"id"`
	Content       string                                 `json:"content"`
	AttachmentIDs []string                               `json:"attachment_ids,omitempty"`
	Attachments   []webChatCompletionAttachmentReference `json:"attachments,omitempty"`
}

type webChatCompletionAttachmentReference struct {
	Source      string `json:"source"`
	FileID      string `json:"file_id,omitempty"`
	FileIDCamel string `json:"fileId,omitempty"`
}

func normalizeWebChatCompletionAttachments(request *webChatCompletionRequest) ([]string, error) {
	if request == nil {
		return nil, errors.New("chat request is nil")
	}
	references := append([]webChatCompletionAttachmentReference(nil), request.Attachments...)
	if request.UserMessage != nil {
		references = append(references, request.UserMessage.Attachments...)
	}
	if len(references) > 0 && request.UserMessage == nil {
		return nil, errors.New("attachments require a user message")
	}
	libraryIDs, err := normalizeLibraryAttachmentReferences(references)
	if err != nil {
		return nil, err
	}
	if request.UserMessage == nil {
		return libraryIDs, nil
	}
	allIDs := make([]string, 0, len(request.UserMessage.AttachmentIDs)+len(libraryIDs))
	allIDs = append(allIDs, request.UserMessage.AttachmentIDs...)
	allIDs = append(allIDs, libraryIDs...)
	seen := make(map[string]struct{}, len(allIDs))
	request.UserMessage.AttachmentIDs = request.UserMessage.AttachmentIDs[:0]
	for _, rawID := range allIDs {
		id := strings.TrimSpace(rawID)
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		request.UserMessage.AttachmentIDs = append(request.UserMessage.AttachmentIDs, id)
	}
	return libraryIDs, nil
}

func normalizeLibraryAttachmentReferences(references []webChatCompletionAttachmentReference) ([]string, error) {
	ids := make([]string, 0, len(references))
	seen := make(map[string]struct{}, len(references))
	for i := range references {
		if strings.ToLower(strings.TrimSpace(references[i].Source)) != "library" {
			return nil, errors.New("invalid attachment source")
		}
		snakeID := strings.TrimSpace(references[i].FileID)
		camelID := strings.TrimSpace(references[i].FileIDCamel)
		if snakeID != "" && camelID != "" && snakeID != camelID {
			return nil, errors.New("conflicting library file ids")
		}
		id := snakeID
		if id == "" {
			id = camelID
		}
		if !validWebChatAttachmentReferenceID(id) {
			return nil, errors.New("invalid library file id")
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids, nil
}

func validWebChatAttachmentReferenceID(value string) bool {
	if len(value) < 8 || len(value) > 80 {
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

type openAIWebChatCompletionRequest struct {
	Model     string                   `json:"model"`
	Messages  []webChatMessage         `json:"messages"`
	Reasoning *webChatReasoningRequest `json:"reasoning,omitempty"`
	Stream    bool                     `json:"stream"`
}

type webChatMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
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

	if req.Reasoning != nil {
		// The nested shape is authoritative when present. Ignore a client
		// supplied summary value and add the product's required `auto` summary
		// later in the trusted Responses conversion.
		req.ReasoningMode = req.Reasoning.Mode
		req.ReasoningEffort = req.Reasoning.Effort
	}
	req.ReasoningMode = strings.ToLower(strings.TrimSpace(req.ReasoningMode))
	if req.ReasoningMode == "" {
		req.ReasoningMode = service.WebChatReasoningModeStandard
	}
	switch req.ReasoningMode {
	case service.WebChatReasoningModeStandard, service.WebChatReasoningModePro:
	default:
		return nil, errors.New("invalid reasoning mode")
	}

	req.ReasoningEffort = strings.ToLower(strings.TrimSpace(req.ReasoningEffort))
	if req.ReasoningEffort == "" {
		req.ReasoningEffort = "low"
	}
	switch req.ReasoningEffort {
	case "low", "medium", "high", "xhigh", "max":
	default:
		return nil, errors.New("invalid reasoning effort")
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
		ID:            message.ID,
		Content:       message.Content,
		AttachmentIDs: append([]string(nil), message.AttachmentIDs...),
	}
}

func isJSONContentType(value string) bool {
	mediaType, _, err := mime.ParseMediaType(value)
	return err == nil && strings.EqualFold(mediaType, "application/json")
}
