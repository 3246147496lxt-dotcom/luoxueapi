package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type chatHistoryHandlerRepositoryStub struct {
	createCalls    int
	listCalls      int
	syncAfter      int64
	deleteRevision int64
	stopUserID     int64
	stopAttemptID  string
}

func (s *chatHistoryHandlerRepositoryStub) CreateConversation(
	context.Context,
	int64,
	*service.CreateChatHistoryConversationInput,
) (*service.ChatHistoryConversation, error) {
	s.createCalls++
	return &service.ChatHistoryConversation{ID: "conversation-12345678"}, nil
}

func (s *chatHistoryHandlerRepositoryStub) ListConversations(
	context.Context,
	int64,
	string,
	int,
	string,
) (*service.ChatHistoryConversationList, error) {
	s.listCalls++
	return &service.ChatHistoryConversationList{}, nil
}

func (*chatHistoryHandlerRepositoryStub) GetConversation(context.Context, int64, string) (*service.ChatHistoryConversation, error) {
	return &service.ChatHistoryConversation{}, nil
}

func (*chatHistoryHandlerRepositoryStub) ListMessages(context.Context, int64, string, *int64, int) (*service.ChatHistoryMessagePage, error) {
	return &service.ChatHistoryMessagePage{}, nil
}

func (*chatHistoryHandlerRepositoryStub) UpdateConversation(context.Context, int64, *service.UpdateChatHistoryConversationInput) (*service.ChatHistoryConversation, error) {
	return &service.ChatHistoryConversation{}, nil
}

func (s *chatHistoryHandlerRepositoryStub) DeleteConversation(
	_ context.Context,
	_ int64,
	_ string,
	revision int64,
) error {
	s.deleteRevision = revision
	return nil
}

func (s *chatHistoryHandlerRepositoryStub) Sync(
	_ context.Context,
	_, afterVersion int64,
	_ int,
) (*service.ChatHistorySyncPage, error) {
	s.syncAfter = afterVersion
	return &service.ChatHistorySyncPage{NextCursor: "17"}, nil
}

func (*chatHistoryHandlerRepositoryStub) GetAttempt(context.Context, int64, string) (*service.ChatHistoryAttempt, error) {
	return &service.ChatHistoryAttempt{}, nil
}

func (*chatHistoryHandlerRepositoryStub) PrepareCompletion(
	context.Context,
	int64,
	*service.PrepareChatCompletionInput,
) (*service.PreparedChatCompletion, error) {
	return nil, errors.New("unexpected prepare completion")
}

func (*chatHistoryHandlerRepositoryStub) FinalizeCompletion(
	context.Context,
	int64,
	*service.FinalizeChatCompletionInput,
) error {
	return errors.New("unexpected finalize completion")
}

func (*chatHistoryHandlerRepositoryStub) CheckpointCompletion(
	context.Context,
	int64,
	*service.CheckpointChatCompletionInput,
) error {
	return errors.New("unexpected checkpoint completion")
}

func (s *chatHistoryHandlerRepositoryStub) StopCompletion(
	_ context.Context,
	userID int64,
	attemptID string,
) (*service.StopChatCompletionResult, error) {
	s.stopUserID = userID
	s.stopAttemptID = attemptID
	return &service.StopChatCompletionResult{
		AttemptID:      attemptID,
		Accepted:       true,
		AttemptStatus:  service.ChatAttemptStatusInterrupted,
		DeliveryStatus: service.ChatMessageDeliveryStopped,
	}, nil
}

func chatHistoryTestContext(method, target, body string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		c.Request.Header.Set("Content-Type", "application/json")
	}
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})
	return c, recorder
}

func TestChatHistoryCreateRejectsUntrustedBillingFields(t *testing.T) {
	repo := &chatHistoryHandlerRepositoryStub{}
	handler := &ChatHandler{history: service.NewChatHistoryService(repo)}
	c, recorder := chatHistoryTestContext(
		http.MethodPost,
		"/api/v1/chat/conversations",
		`{"id":"conversation-12345678","title":"Title","model":"gpt-5.5","charged_amount":99}`,
	)

	handler.CreateConversation(c)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Zero(t, repo.createCalls)
}

func TestChatHistoryDeleteReadsRevisionFromJSONBody(t *testing.T) {
	repo := &chatHistoryHandlerRepositoryStub{}
	handler := &ChatHandler{history: service.NewChatHistoryService(repo)}
	c, recorder := chatHistoryTestContext(
		http.MethodDelete,
		"/api/v1/chat/conversations/conversation-12345678",
		`{"revision":7}`,
	)
	c.Params = gin.Params{{Key: "conversation_id", Value: "conversation-12345678"}}

	handler.DeleteConversation(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, int64(7), repo.deleteRevision)
}

func TestChatHistorySyncAcceptsAfterVersionFallback(t *testing.T) {
	repo := &chatHistoryHandlerRepositoryStub{}
	handler := &ChatHandler{history: service.NewChatHistoryService(repo)}
	c, recorder := chatHistoryTestContext(
		http.MethodGet,
		"/api/v1/chat/sync?after_version=17",
		"",
	)

	handler.SyncConversations(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, int64(17), repo.syncAfter)
}

func TestChatHistoryListRejectsSearchInURL(t *testing.T) {
	repo := &chatHistoryHandlerRepositoryStub{}
	handler := &ChatHandler{history: service.NewChatHistoryService(repo)}
	c, recorder := chatHistoryTestContext(
		http.MethodGet,
		"/api/v1/chat/conversations?q=secret",
		"",
	)

	handler.ListConversations(c)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Zero(t, repo.listCalls)
}

func TestChatHistoryStopAttemptUsesAuthenticatedUserAndAttemptID(t *testing.T) {
	repo := &chatHistoryHandlerRepositoryStub{}
	handler := &ChatHandler{history: service.NewChatHistoryService(repo)}
	c, recorder := chatHistoryTestContext(
		http.MethodPost,
		"/api/v1/chat/attempts/attempt-12345678/stop",
		"",
	)
	c.Params = gin.Params{{Key: "attempt_id", Value: "attempt-12345678"}}

	handler.StopAttempt(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, int64(42), repo.stopUserID)
	require.Equal(t, "attempt-12345678", repo.stopAttemptID)
	require.Contains(t, recorder.Body.String(), `"accepted":true`)
	require.Contains(t, recorder.Body.String(), `"delivery_status":"stopped"`)
}
