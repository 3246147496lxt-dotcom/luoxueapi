package repository

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAdminChatHistoryRepositoryListsMetadataWithKeysetCursor(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	now := time.Date(2026, 7, 25, 10, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows([]string{
		"id", "public_id", "model", "message_count", "status", "created_at", "updated_at",
	}).
		AddRow(30, "conversation-30", "gpt-5.5", 4, "active", now.Add(-time.Hour), now).
		AddRow(20, "conversation-20", "gpt-5.5", 2, "active", now.Add(-2*time.Hour), now.Add(-time.Minute)).
		AddRow(10, "conversation-10", "gpt-5.5", 1, "active", now.Add(-3*time.Hour), now.Add(-2*time.Minute))
	mock.ExpectQuery(regexp.QuoteMeta("FROM chat_conversations c")).
		WithArgs(int64(42), 3).
		WillReturnRows(rows)

	repo := NewAdminChatHistoryRepository(db)
	page, err := repo.ListUserConversationRefs(
		context.Background(),
		service.AdminChatConversationListQuery{TargetUserID: 42, Limit: 2},
	)

	require.NoError(t, err)
	require.Len(t, page.Items, 2)
	require.True(t, page.HasMore)
	require.NotNil(t, page.NextID)
	require.Equal(t, int64(20), *page.NextID)
	require.NotNil(t, page.NextUpdatedAt)
	require.Equal(t, "conversation-30", page.Items[0].ID)
	require.Equal(t, 4, page.Items[0].MessageCount)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAdminChatHistoryRepositoryReadsContentAndAuditInOneTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	now := time.Date(2026, 7, 25, 10, 0, 0, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("FROM chat_conversations c")).
		WithArgs(int64(42), "conversation-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "public_id", "title", "model", "status", "created_at", "updated_at",
		}).AddRow(
			99, "conversation-1", "Private title", "gpt-5.5", "active", now.Add(-time.Hour), now,
		))

	messageRows := sqlmock.NewRows([]string{
		"public_id", "position", "role", "content", "delivery_status",
		"requested_model", "finish_reason", "error_code", "error_message",
		"attempt_id", "client_request_id", "created_at", "terminal_at",
	}).
		AddRow("message-5", 5, "assistant", "newest private body", "completed",
			"gpt-5.5", "stop", "", "", "attempt-5", "receipt-5", now, now).
		AddRow("message-4", 4, "user", "older private body", "completed",
			"", "", "", "", "", "", now.Add(-time.Minute), now.Add(-time.Minute)).
		AddRow("message-3", 3, "assistant", "pagination sentinel", "completed",
			"gpt-5.5", "stop", "", "", "attempt-3", "receipt-3", now.Add(-2*time.Minute), now.Add(-2*time.Minute))
	mock.ExpectQuery(regexp.QuoteMeta("FROM chat_messages m")).
		WithArgs(int64(99), 3).
		WillReturnRows(messageRows)
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO chat_admin_content_access_logs")).
		WithArgs(
			int64(7),
			int64(42),
			int64(99),
			"conversation-1",
			"request-1",
			"203.0.113.8",
			"admin-browser",
			nil,
			2,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	repo := NewAdminChatHistoryRepository(db)
	page, err := repo.ViewConversationPageAndRecordAccess(
		context.Background(),
		service.AdminChatContentAccessQuery{
			AdminID:              7,
			TargetUserID:         42,
			ConversationPublicID: "conversation-1",
			Limit:                2,
			RequestID:            "request-1",
			ClientIP:             "203.0.113.8",
			UserAgent:            "admin-browser",
		},
	)

	require.NoError(t, err)
	require.Equal(t, "Private title", page.Conversation.Title)
	require.True(t, page.HasMore)
	require.NotNil(t, page.NextBeforePosition)
	require.Equal(t, int64(4), *page.NextBeforePosition)
	require.Len(t, page.Messages, 2)
	require.Equal(t, int64(4), page.Messages[0].Position)
	require.Equal(t, int64(5), page.Messages[1].Position)
	require.Equal(t, "receipt-5", page.Messages[1].ReceiptID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAdminChatHistoryRepositoryReturnsNoContentWhenAuditInsertFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	now := time.Now().UTC()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("FROM chat_conversations c")).
		WithArgs(int64(42), "conversation-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "public_id", "title", "model", "status", "created_at", "updated_at",
		}).AddRow(99, "conversation-1", "Private title", "gpt-5.5", "active", now, now))
	mock.ExpectQuery(regexp.QuoteMeta("FROM chat_messages m")).
		WithArgs(int64(99), 2).
		WillReturnRows(sqlmock.NewRows([]string{
			"public_id", "position", "role", "content", "delivery_status",
			"requested_model", "finish_reason", "error_code", "error_message",
			"attempt_id", "client_request_id", "created_at", "terminal_at",
		}).AddRow(
			"message-1", 1, "user", "must not escape", "completed",
			"", "", "", "", "", "", now, nil,
		))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO chat_admin_content_access_logs")).
		WithArgs(int64(7), int64(42), int64(99), "conversation-1", "", "", "", nil, 1).
		WillReturnError(errors.New("audit storage unavailable"))
	mock.ExpectRollback()

	repo := NewAdminChatHistoryRepository(db)
	page, err := repo.ViewConversationPageAndRecordAccess(
		context.Background(),
		service.AdminChatContentAccessQuery{
			AdminID:              7,
			TargetUserID:         42,
			ConversationPublicID: "conversation-1",
			Limit:                1,
		},
	)

	require.Nil(t, page)
	require.ErrorIs(t, err, service.ErrAdminChatContentAuditUnavailable)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAdminChatHistoryRepositoryDoesNotAuditMissingConversation(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("FROM chat_conversations c")).
		WithArgs(int64(42), "missing").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	repo := NewAdminChatHistoryRepository(db)
	page, err := repo.ViewConversationPageAndRecordAccess(
		context.Background(),
		service.AdminChatContentAccessQuery{
			AdminID:              7,
			TargetUserID:         42,
			ConversationPublicID: "missing",
			Limit:                30,
		},
	)

	require.Nil(t, page)
	require.ErrorIs(t, err, service.ErrAdminChatConversationNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}
