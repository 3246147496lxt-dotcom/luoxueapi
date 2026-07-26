package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type chatApplicationStub struct {
	resolveCalls int
	resolveErr   error
	principal    *service.APIKey
}

func (s *chatApplicationStub) ListModels(context.Context, int64) (*service.ChatModelsResult, error) {
	return nil, nil
}

func (s *chatApplicationStub) ResolvePrincipal(context.Context, int64, string) (*service.APIKey, error) {
	s.resolveCalls++
	return s.principal, s.resolveErr
}

type chatCompletionDelegatorStub struct {
	calls       int
	requestBody string
	response    string
	hook        func(*gin.Context)
}

type chatAttemptLeaseManagerStub struct {
	started   bool
	stopped   bool
	userID    int64
	attemptID string
}

func (s *chatAttemptLeaseManagerStub) StartLeaseHeartbeat(
	userID int64,
	attemptID string,
) func() {
	s.started = true
	s.userID = userID
	s.attemptID = attemptID
	return func() {
		s.stopped = true
	}
}

type billingReceiptRepositoryStub struct {
	receipt *service.BillingReceipt
	err     error
}

func (s *billingReceiptRepositoryStub) GetUserWebChatReceipt(
	context.Context,
	int64,
	string,
) (*service.BillingReceipt, error) {
	return s.receipt, s.err
}

func (s *billingReceiptRepositoryStub) ListBillingReceipts(
	context.Context,
	*service.BillingReceiptFilter,
) (*service.BillingReceiptList, error) {
	return nil, errors.New("unexpected list call")
}

func (s *chatCompletionDelegatorStub) ChatCompletions(c *gin.Context) {
	s.calls++
	if c != nil && c.Request != nil && c.Request.Body != nil {
		body, _ := io.ReadAll(c.Request.Body)
		s.requestBody = string(body)
	}
	if s.response != "" {
		_, _ = c.Writer.WriteString(s.response)
	}
	if s.hook != nil {
		s.hook(c)
	}
}

type chatAttemptClaimerStub struct {
	calls         int
	outcomeCalls  int
	outcomeStatus string
	result        *service.ChatAttemptClaimResult
	err           error
}

func (s *chatAttemptClaimerStub) Claim(
	context.Context,
	int64,
	string,
	string,
	[]byte,
) (*service.ChatAttemptClaimResult, error) {
	s.calls++
	if s.result != nil || s.err != nil {
		return s.result, s.err
	}
	return &service.ChatAttemptClaimResult{
		Claimed:         true,
		ClientRequestID: "server-request-id",
	}, nil
}

func (s *chatAttemptClaimerStub) MarkOutcome(
	_ context.Context,
	_ int64,
	_ string,
	status string,
	_ int,
	_ string,
	_ string,
) error {
	s.outcomeCalls++
	s.outcomeStatus = status
	return nil
}

type trackedChatRequestBody struct {
	read bool
}

func (b *trackedChatRequestBody) Read([]byte) (int, error) {
	b.read = true
	return 0, io.EOF
}

func (*trackedChatRequestBody) Close() error { return nil }

type chatCompletionHistoryRepositoryStub struct {
	prepareInput    *service.PrepareChatCompletionInput
	prepareResult   *service.PreparedChatCompletion
	prepareErr      error
	checkpointInput *service.CheckpointChatCompletionInput
	finalizeInput   *service.FinalizeChatCompletionInput
	finalizeErr     error
	finalizeHook    func()
}

func (*chatCompletionHistoryRepositoryStub) CreateConversation(
	context.Context,
	int64,
	*service.CreateChatHistoryConversationInput,
) (*service.ChatHistoryConversation, error) {
	return nil, errors.New("unexpected create conversation")
}

func (*chatCompletionHistoryRepositoryStub) ListConversations(
	context.Context,
	int64,
	string,
	int,
	string,
) (*service.ChatHistoryConversationList, error) {
	return nil, errors.New("unexpected list conversations")
}

func (*chatCompletionHistoryRepositoryStub) GetConversation(
	context.Context,
	int64,
	string,
) (*service.ChatHistoryConversation, error) {
	return nil, errors.New("unexpected get conversation")
}

func (*chatCompletionHistoryRepositoryStub) ListMessages(
	context.Context,
	int64,
	string,
	*int64,
	int,
) (*service.ChatHistoryMessagePage, error) {
	return nil, errors.New("unexpected list messages")
}

func (*chatCompletionHistoryRepositoryStub) UpdateConversation(
	context.Context,
	int64,
	*service.UpdateChatHistoryConversationInput,
) (*service.ChatHistoryConversation, error) {
	return nil, errors.New("unexpected update conversation")
}

func (*chatCompletionHistoryRepositoryStub) DeleteConversation(
	context.Context,
	int64,
	string,
	int64,
) error {
	return errors.New("unexpected delete conversation")
}

func (*chatCompletionHistoryRepositoryStub) Sync(
	context.Context,
	int64,
	int64,
	int,
) (*service.ChatHistorySyncPage, error) {
	return nil, errors.New("unexpected sync")
}

func (*chatCompletionHistoryRepositoryStub) GetAttempt(
	context.Context,
	int64,
	string,
) (*service.ChatHistoryAttempt, error) {
	return nil, errors.New("unexpected get attempt")
}

func (s *chatCompletionHistoryRepositoryStub) PrepareCompletion(
	_ context.Context,
	_ int64,
	input *service.PrepareChatCompletionInput,
) (*service.PreparedChatCompletion, error) {
	s.prepareInput = input
	if s.prepareResult != nil || s.prepareErr != nil {
		return s.prepareResult, s.prepareErr
	}
	return &service.PreparedChatCompletion{
		Claimed:            true,
		ClientRequestID:    input.ClientRequestID,
		AttemptStatus:      service.ChatAttemptStatusProcessing,
		ConversationID:     input.ConversationID,
		AssistantMessageID: input.AssistantMessageID,
		Messages: []service.ChatCompletionContextMessage{{
			Role:    "user",
			Content: "server-owned context",
		}},
	}, nil
}

func (s *chatCompletionHistoryRepositoryStub) FinalizeCompletion(
	_ context.Context,
	_ int64,
	input *service.FinalizeChatCompletionInput,
) error {
	s.finalizeInput = input
	if s.finalizeHook != nil {
		s.finalizeHook()
	}
	return s.finalizeErr
}

func (s *chatCompletionHistoryRepositoryStub) CheckpointCompletion(
	_ context.Context,
	_ int64,
	input *service.CheckpointChatCompletionInput,
) error {
	s.checkpointInput = input
	return nil
}

func chatHistoryServiceForCompletion(
	repo *chatCompletionHistoryRepositoryStub,
) *service.ChatHistoryService {
	return service.NewChatHistoryService(repo)
}

func TestChatCompletionsRejectsUnsupportedContentTypeBeforeBodyOrService(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for _, tt := range []struct {
		name        string
		contentType string
	}{
		{name: "missing"},
		{name: "different media type", contentType: "text/plain"},
		{name: "malformed parameter", contentType: "application/json; charset"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			body := &trackedChatRequestBody{}
			request := httptest.NewRequest(http.MethodPost, "/api/v1/chat/completions", nil)
			request.Body = body
			if tt.contentType != "" {
				request.Header.Set("Content-Type", tt.contentType)
			}

			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = request
			c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})
			application := &chatApplicationStub{}
			gateway := &chatCompletionDelegatorStub{}
			history := &chatCompletionHistoryRepositoryStub{}

			(&ChatHandler{
				chat:    application,
				history: chatHistoryServiceForCompletion(history),
				gateway: gateway,
			}).Completions(c)

			require.Equal(t, http.StatusUnsupportedMediaType, recorder.Code)
			var got response.Response
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &got))
			require.Equal(t, "UNSUPPORTED_MEDIA_TYPE", got.Reason)
			require.False(t, body.read)
			require.Zero(t, application.resolveCalls)
			require.Zero(t, gateway.calls)
		})
	}
}

func TestChatReceiptReturnsAcceptedPendingPayloadWithBareReceiptID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	createdAt := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)
	receiptService := service.NewBillingReceiptService(&billingReceiptRepositoryStub{
		receipt: &service.BillingReceipt{
			RequestID: "client:11111111-1111-4111-8111-111111111111",
			Source:    service.BillingReceiptSourceWebChat,
			UserID:    42,
			Status:    service.BillingReceiptStatusPending,
			CreatedAt: createdAt,
		},
	})
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/chat/receipts/11111111-1111-4111-8111-111111111111", nil)
	c.Params = gin.Params{{
		Key:   "receipt_id",
		Value: "11111111-1111-4111-8111-111111111111",
	}}
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})

	(&ChatHandler{receipts: receiptService}).Receipt(c)

	require.Equal(t, http.StatusAccepted, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"receipt_id":"11111111-1111-4111-8111-111111111111"`)
	require.Contains(t, recorder.Body.String(), `"status":"pending"`)
	require.NotContains(t, recorder.Body.String(), `"user_id"`)
	require.NotContains(t, recorder.Body.String(), `"api_key_id"`)
}

func TestChatReceiptUsesActualModelBeforeRequestedModel(t *testing.T) {
	payload := chatReceiptFromService(&service.BillingReceipt{
		Model:          "gpt-5.5-actual",
		RequestedModel: "gpt-5.5",
	})

	require.Equal(t, "gpt-5.5-actual", payload.Model)
}

func TestChatCompletionsAcceptsJSONMediaTypeVariants(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for _, contentType := range []string{
		"application/json; charset=utf-8",
		"Application/JSON",
	} {
		t.Run(contentType, func(t *testing.T) {
			request := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/chat/completions",
				strings.NewReader(`{
					"conversation_id":"conversation-12345678",
					"model":"gpt-5",
					"expected_head_message_id":null,
					"user_message":{"id":"message-user-12345678","content":"hello"},
					"assistant_message_id":"message-assistant-12345678"
				}`),
			)
			request.Header.Set("Content-Type", contentType)
			request.Header.Set("X-Chat-Attempt-ID", "attempt-12345678")
			request = request.WithContext(context.WithValue(
				request.Context(),
				ctxkey.ClientRequestID,
				"server-request-id",
			))
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = request
			c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})
			application := &chatApplicationStub{resolveErr: service.ErrChatModelNotAvailable}
			gateway := &chatCompletionDelegatorStub{}
			history := &chatCompletionHistoryRepositoryStub{}

			(&ChatHandler{
				chat:    application,
				history: chatHistoryServiceForCompletion(history),
				gateway: gateway,
			}).Completions(c)

			require.Equal(t, http.StatusBadRequest, recorder.Code)
			require.Equal(t, 1, application.resolveCalls)
			require.Zero(t, gateway.calls)
		})
	}
}

func TestChatCompletionsRejectsBodyOverTwoMiB(t *testing.T) {
	gin.SetMode(gin.TestMode)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/chat/completions",
		strings.NewReader(`{"model":"gpt-5","messages":[{"role":"user","content":"`+strings.Repeat("a", maxChatRequestBytes)+`"}]}`),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = request
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})
	application := &chatApplicationStub{}
	gateway := &chatCompletionDelegatorStub{}
	history := &chatCompletionHistoryRepositoryStub{}

	(&ChatHandler{
		chat:    application,
		history: chatHistoryServiceForCompletion(history),
		gateway: gateway,
	}).Completions(c)

	require.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)
	var got response.Response
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &got))
	require.Equal(t, "REQUEST_TOO_LARGE", got.Reason)
	require.Zero(t, application.resolveCalls)
	require.Zero(t, gateway.calls)
}

func TestChatCompletionsClaimsAttemptBeforeForwarding(t *testing.T) {
	gin.SetMode(gin.TestMode)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/chat/completions",
		strings.NewReader(`{
			"conversation_id":"conversation-12345678",
			"model":"gpt-5.5",
			"expected_head_message_id":"message-head-12345678",
			"user_message":{"id":"message-user-12345678","content":"client-only user message"},
			"assistant_message_id":"message-assistant-12345678"
		}`),
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Chat-Attempt-ID", "attempt-12345678")
	request = request.WithContext(context.WithValue(request.Context(), ctxkey.WebChatIngress, true))
	request = request.WithContext(context.WithValue(request.Context(), ctxkey.ClientRequestID, "server-request-id"))

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = request
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})

	history := &chatCompletionHistoryRepositoryStub{}
	lease := &chatAttemptLeaseManagerStub{}
	history.finalizeHook = func() {
		require.True(t, lease.started)
		require.False(t, lease.stopped)
	}
	gateway := &chatCompletionDelegatorStub{
		response: "data: {\"choices\":[{\"delta\":{\"content\":\"hello\"},\"finish_reason\":\"stop\"}]}\n\n" +
			"data: [DONE]\n\n",
	}
	(&ChatHandler{
		chat:     &chatApplicationStub{principal: validChatHandlerPrincipal()},
		attempts: lease,
		history:  chatHistoryServiceForCompletion(history),
		gateway:  gateway,
	}).Completions(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.NotNil(t, history.prepareInput)
	require.NotNil(t, history.finalizeInput)
	require.Equal(t, service.ChatAttemptStatusCompleted, history.finalizeInput.AttemptStatus)
	require.Equal(t, "hello", history.finalizeInput.Content)
	require.Equal(t, 1, gateway.calls)
	require.True(t, lease.stopped)
	require.Equal(t, int64(42), lease.userID)
	require.Equal(t, "attempt-12345678", lease.attemptID)
	require.Equal(t, "server-request-id", recorder.Header().Get("X-Client-Request-ID"))
	require.Contains(t, gateway.requestBody, `"content":"server-owned context"`)
	require.NotContains(t, gateway.requestBody, "client-only user message")
}

func TestChatCompletionsReplayReturnsOriginalReceiptWithoutForwarding(t *testing.T) {
	gin.SetMode(gin.TestMode)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/chat/completions",
		strings.NewReader(`{
			"conversation_id":"conversation-12345678",
			"model":"gpt-5.5",
			"expected_head_message_id":"message-head-12345678",
			"user_message":{"id":"message-user-12345678","content":"hello"},
			"assistant_message_id":"message-assistant-12345678"
		}`),
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Chat-Attempt-ID", "attempt-12345678")
	request = request.WithContext(context.WithValue(request.Context(), ctxkey.WebChatIngress, true))
	request = request.WithContext(context.WithValue(request.Context(), ctxkey.ClientRequestID, "new-server-request-id"))

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = request
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})

	history := &chatCompletionHistoryRepositoryStub{
		prepareResult: &service.PreparedChatCompletion{
			Claimed:            false,
			ClientRequestID:    "original-server-request-id",
			AttemptStatus:      service.ChatAttemptStatusCompleted,
			ConversationID:     "conversation-12345678",
			AssistantMessageID: "message-assistant-12345678",
		},
	}
	gateway := &chatCompletionDelegatorStub{}
	(&ChatHandler{
		chat:    &chatApplicationStub{principal: validChatHandlerPrincipal()},
		history: chatHistoryServiceForCompletion(history),
		gateway: gateway,
	}).Completions(c)

	require.Equal(t, http.StatusConflict, recorder.Code)
	require.Equal(t, "original-server-request-id", recorder.Header().Get("X-Client-Request-ID"))
	require.Zero(t, gateway.calls)
	require.Nil(t, history.finalizeInput)
}

func TestChatCompletionsWaitsForUsageProducerBeforeFinalize(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c, recorder := newChatCompletionContractContext(
		"11111111-1111-4111-8111-111111111111",
	)
	producerStarted := make(chan struct{})
	releaseProducer := make(chan struct{})
	finalized := make(chan struct{})
	handlerDone := make(chan struct{})

	history := &chatCompletionHistoryRepositoryStub{
		finalizeHook: func() {
			close(finalized)
		},
	}
	gateway := &chatCompletionDelegatorStub{
		response: "data: {\"choices\":[{\"delta\":{\"content\":\"hello\"},\"finish_reason\":\"stop\"}]}\n\n" +
			"data: [DONE]\n\n",
		hook: func(c *gin.Context) {
			barrier := webChatUsageBarrierFromContext(c.Request.Context())
			require.NotNil(t, barrier)
			producer, ok := barrier.RegisterResult(func(context.Context) error {
				close(producerStarted)
				<-releaseProducer
				return nil
			})
			require.True(t, ok)
			go producer(context.Background())
		},
	}
	handler := &ChatHandler{
		chat:     &chatApplicationStub{principal: validChatHandlerPrincipal()},
		attempts: &chatAttemptLeaseManagerStub{},
		history:  chatHistoryServiceForCompletion(history),
		gateway:  gateway,
	}

	go func() {
		defer close(handlerDone)
		handler.Completions(c)
	}()

	select {
	case <-producerStarted:
	case <-time.After(time.Second):
		t.Fatal("usage producer did not start")
	}
	select {
	case <-finalized:
		t.Fatal("chat attempt finalized before usage producer completed")
	case <-time.After(20 * time.Millisecond):
	}
	close(releaseProducer)
	select {
	case <-handlerDone:
	case <-time.After(time.Second):
		t.Fatal("chat handler did not finish after usage producer completed")
	}

	require.Equal(t, http.StatusOK, recorder.Code)
	require.NotNil(t, history.finalizeInput)
	require.Equal(t, service.ChatAttemptStatusCompleted, history.finalizeInput.AttemptStatus)
}

func TestChatCompletionsUsageProducerFailureSettlementContract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	producerErr := errors.New("usage database unavailable")
	for _, tt := range []struct {
		name               string
		receipt            *service.BillingReceipt
		receiptErr         error
		wantFinalize       bool
		wantAttemptStatus  string
		wantFailureCode    string
		wantDeliveryStatus string
	}{
		{
			name: "canonical charge evidence keeps delivered outcome",
			receipt: &service.BillingReceipt{
				ID:     91,
				Status: service.BillingReceiptStatusCharged,
			},
			wantFinalize:       true,
			wantAttemptStatus:  service.ChatAttemptStatusCompleted,
			wantDeliveryStatus: service.ChatMessageDeliveryCompleted,
		},
		{
			name: "confirmed no billing finalizes settlement failure",
			receipt: &service.BillingReceipt{
				Status: service.BillingReceiptStatusPending,
			},
			wantFinalize:       true,
			wantAttemptStatus:  service.ChatAttemptStatusFailed,
			wantFailureCode:    chatSettlementFailureCode,
			wantDeliveryStatus: service.ChatMessageDeliveryError,
		},
		{
			name:         "unqueryable receipt leaves attempt processing",
			receiptErr:   errors.New("receipt database unavailable"),
			wantFinalize: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := newChatCompletionContractContext(
				"22222222-2222-4222-8222-222222222222",
			)
			history := &chatCompletionHistoryRepositoryStub{}
			lease := &chatAttemptLeaseManagerStub{}
			gateway := &chatCompletionDelegatorStub{
				response: "data: {\"choices\":[{\"delta\":{\"content\":\"hello\"},\"finish_reason\":\"stop\"}]}\n\n" +
					"data: [DONE]\n\n",
				hook: func(c *gin.Context) {
					barrier := webChatUsageBarrierFromContext(c.Request.Context())
					require.NotNil(t, barrier)
					producer, ok := barrier.RegisterResult(func(context.Context) error {
						return producerErr
					})
					require.True(t, ok)
					producer(context.Background())
				},
			}
			handler := &ChatHandler{
				chat:     &chatApplicationStub{principal: validChatHandlerPrincipal()},
				attempts: lease,
				receipts: service.NewBillingReceiptService(&billingReceiptRepositoryStub{
					receipt: tt.receipt,
					err:     tt.receiptErr,
				}),
				history: chatHistoryServiceForCompletion(history),
				gateway: gateway,
			}

			handler.Completions(c)

			require.True(t, lease.stopped)
			if !tt.wantFinalize {
				require.Nil(t, history.finalizeInput)
				return
			}
			require.NotNil(t, history.finalizeInput)
			require.Equal(t, tt.wantAttemptStatus, history.finalizeInput.AttemptStatus)
			require.Equal(t, tt.wantFailureCode, history.finalizeInput.ErrorCode)
			require.Equal(t, tt.wantDeliveryStatus, history.finalizeInput.DeliveryStatus)
		})
	}
}

func newChatCompletionContractContext(clientRequestID string) (*gin.Context, *httptest.ResponseRecorder) {
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/chat/completions",
		strings.NewReader(`{
			"conversation_id":"conversation-12345678",
			"model":"gpt-5.5",
			"expected_head_message_id":null,
			"user_message":{"id":"message-user-12345678","content":"hello"},
			"assistant_message_id":"message-assistant-12345678"
		}`),
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Chat-Attempt-ID", "attempt-12345678")
	request = request.WithContext(context.WithValue(request.Context(), ctxkey.WebChatIngress, true))
	request = request.WithContext(context.WithValue(request.Context(), ctxkey.WebChat, true))
	request = request.WithContext(context.WithValue(request.Context(), ctxkey.ClientRequestID, clientRequestID))

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = request
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})
	return c, recorder
}

func validChatHandlerPrincipal() *service.APIKey {
	group := &service.Group{
		ID:               9,
		Status:           service.StatusActive,
		Platform:         service.PlatformOpenAI,
		SubscriptionType: service.SubscriptionTypeStandard,
		Hydrated:         true,
	}
	user := &service.User{ID: 42, Status: service.StatusActive}
	return &service.APIKey{
		ID:      101,
		UserID:  user.ID,
		GroupID: &group.ID,
		Status:  service.StatusActive,
		Purpose: service.APIKeyPurposeWebChat,
		User:    user,
		Group:   group,
	}
}
