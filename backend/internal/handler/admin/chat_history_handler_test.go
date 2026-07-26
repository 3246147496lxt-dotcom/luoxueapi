package admin

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type adminChatHistoryRepositoryStub struct {
	listQuery service.AdminChatConversationListQuery
	viewQuery service.AdminChatContentAccessQuery
	list      *service.AdminChatConversationRefPage
	view      *service.AdminChatConversationPage
	listErr   error
	viewErr   error
}

func (s *adminChatHistoryRepositoryStub) ListUserConversationRefs(
	_ context.Context,
	query service.AdminChatConversationListQuery,
) (*service.AdminChatConversationRefPage, error) {
	s.listQuery = query
	return s.list, s.listErr
}

func (s *adminChatHistoryRepositoryStub) ViewConversationPageAndRecordAccess(
	_ context.Context,
	query service.AdminChatContentAccessQuery,
) (*service.AdminChatConversationPage, error) {
	s.viewQuery = query
	return s.view, s.viewErr
}

func adminChatTestRouter(
	repo service.AdminChatHistoryRepository,
	authMethod string,
) *gin.Engine {
	return adminChatTestRouterWithRole(repo, authMethod, "admin")
}

func adminChatTestRouterWithRole(
	repo service.AdminChatHistoryRepository,
	authMethod string,
	role string,
) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewAdminChatHistoryHandler(service.NewAdminChatHistoryService(repo))
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 7})
		c.Set(string(middleware.ContextKeyUserRole), role)
		c.Set("auth_method", authMethod)
		c.Next()
	})
	router.GET("/admin/users/:id/chat/conversations", h.ListUserConversations)
	router.GET(
		"/admin/users/:id/chat/conversations/:conversation_id",
		h.ViewConversation,
	)
	return router
}

func TestAdminChatListReturnsMetadataWithoutContent(t *testing.T) {
	updatedAt := time.Date(2026, 7, 25, 10, 0, 0, 0, time.UTC)
	nextID := int64(11)
	repo := &adminChatHistoryRepositoryStub{
		list: &service.AdminChatConversationRefPage{
			Items: []service.AdminChatConversationRef{{
				ID:           "conversation-1",
				Model:        "gpt-5.5",
				MessageCount: 2,
				Status:       "active",
				CreatedAt:    updatedAt.Add(-time.Hour),
				UpdatedAt:    updatedAt,
			}},
			HasMore:       true,
			NextUpdatedAt: &updatedAt,
			NextID:        &nextID,
		},
	}
	router := adminChatTestRouter(repo, service.AuditAuthMethodJWT)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/admin/users/42/chat/conversations?limit=20",
		nil,
	)

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, int64(42), repo.listQuery.TargetUserID)
	require.Equal(t, 20, repo.listQuery.Limit)
	require.Contains(t, recorder.Body.String(), `"model":"gpt-5.5"`)
	require.Contains(t, recorder.Body.String(), `"next_cursor":"`)
	require.NotContains(t, recorder.Body.String(), "title")
	require.NotContains(t, recorder.Body.String(), "content")
	require.Equal(t, "private, no-store", recorder.Header().Get("Cache-Control"))
	require.Equal(t, "Authorization", recorder.Header().Get("Vary"))
}

func TestAdminChatViewRecordsAuthenticatedAdminMetadata(t *testing.T) {
	secret := "private conversation body"
	repo := &adminChatHistoryRepositoryStub{
		view: &service.AdminChatConversationPage{
			Conversation: service.AdminChatConversationContent{
				ID:     "conversation-1",
				Title:  "Private title",
				Model:  "gpt-5.5",
				Status: "active",
			},
			Messages: []service.AdminChatMessage{{
				ID:       "message-1",
				Position: 1,
				Role:     "user",
				Content:  secret,
				Status:   "completed",
			}},
		},
	}
	router := adminChatTestRouter(repo, service.AuditAuthMethodJWT)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/admin/users/42/chat/conversations/conversation-1?before_position=9&limit=25",
		nil,
	)
	request.RemoteAddr = "203.0.113.8:4321"
	request.Header.Set("User-Agent", "admin-browser")

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), secret)
	require.Equal(t, int64(7), repo.viewQuery.AdminID)
	require.Equal(t, int64(42), repo.viewQuery.TargetUserID)
	require.Equal(t, "conversation-1", repo.viewQuery.ConversationPublicID)
	require.NotNil(t, repo.viewQuery.BeforePosition)
	require.Equal(t, int64(9), *repo.viewQuery.BeforePosition)
	require.Equal(t, 25, repo.viewQuery.Limit)
	require.Equal(t, "admin-browser", repo.viewQuery.UserAgent)
	require.Equal(t, "203.0.113.8", repo.viewQuery.ClientIP)
	require.Equal(t, "no-cache", recorder.Header().Get("Pragma"))
}

func TestAdminChatViewRejectsAdminAPIKey(t *testing.T) {
	repo := &adminChatHistoryRepositoryStub{}
	router := adminChatTestRouter(repo, service.AuditAuthMethodAdminAPIKey)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/admin/users/42/chat/conversations/conversation-1",
		nil,
	)

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.Contains(t, recorder.Body.String(), "ADMIN_CHAT_JWT_REQUIRED")
	require.Zero(t, repo.viewQuery.AdminID)
}

func TestAdminChatViewRejectsOrdinaryUserJWT(t *testing.T) {
	repo := &adminChatHistoryRepositoryStub{}
	router := adminChatTestRouterWithRole(repo, service.AuditAuthMethodJWT, "user")
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/admin/users/42/chat/conversations/conversation-1",
		nil,
	)

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.Contains(t, recorder.Body.String(), "ADMIN_ACCESS_REQUIRED")
	require.Zero(t, repo.viewQuery.AdminID)
}

func TestAdminChatViewDoesNotReturnContentWhenAuditTransactionFails(t *testing.T) {
	secret := "must never leak"
	repo := &adminChatHistoryRepositoryStub{
		view: &service.AdminChatConversationPage{
			Messages: []service.AdminChatMessage{{Content: secret}},
		},
		viewErr: service.ErrAdminChatContentAuditUnavailable.WithCause(
			errors.New("audit insert failed"),
		),
	}
	router := adminChatTestRouter(repo, service.AuditAuthMethodJWT)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/admin/users/42/chat/conversations/conversation-1",
		nil,
	)

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	require.NotContains(t, recorder.Body.String(), secret)
	require.True(t, strings.Contains(
		recorder.Body.String(),
		"ADMIN_CHAT_CONTENT_AUDIT_UNAVAILABLE",
	))
}

func TestAdminChatCursorRoundTripAndRejectsMalformedValues(t *testing.T) {
	updatedAt := time.Date(2026, 7, 25, 10, 20, 30, 123, time.FixedZone("CST", 8*3600))
	encoded := encodeAdminChatCursor(updatedAt, 88)
	decoded, ok := decodeAdminChatCursor(encoded)
	require.True(t, ok)
	require.Equal(t, int64(88), decoded.ID)
	require.True(t, updatedAt.Equal(decoded.UpdatedAt))

	decoded, ok = decodeAdminChatCursor("not-base64")
	require.False(t, ok)
	require.Nil(t, decoded)
}
