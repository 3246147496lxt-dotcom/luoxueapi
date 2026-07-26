package repository

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestChatHistoryConversationCursorRoundTrip(t *testing.T) {
	t.Parallel()
	updatedAt := time.Date(2026, 7, 25, 12, 34, 56, 123456000, time.UTC)

	cursor := encodeChatHistoryConversationCursor(updatedAt, 91)
	decodedTime, decodedID, err := decodeChatHistoryConversationCursor(cursor)

	require.NoError(t, err)
	require.Equal(t, updatedAt, decodedTime)
	require.Equal(t, int64(91), decodedID)
}

func TestChatHistoryListMessagesReturnsAscendingPageAndBeforeCursor(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	userID := int64(42)
	conversationID := int64(9)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id")).
		WithArgs(userID, "conversation-12345678").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(conversationID))

	columns := []string{
		"public_id",
		"position",
		"role",
		"content",
		"delivery_status",
		"requested_model",
		"finish_reason",
		"error_code",
		"error_message",
		"excluded_from_context",
		"superseded_by_message_id",
		"checkpoint_seq",
		"attempt_id",
		"client_request_id",
		"created_at",
		"updated_at",
		"terminal_at",
	}
	now := time.Now().UTC()
	mock.ExpectQuery(regexp.QuoteMeta("FROM chat_messages m")).
		WithArgs(userID, conversationID, 3).
		WillReturnRows(sqlmock.NewRows(columns).
			AddRow("message-00000005", 5, "assistant", "five", "completed", "gpt-5.5", nil, nil, nil, false, nil, 4, nil, nil, now, now, now).
			AddRow("message-00000004", 4, "user", "four", "completed", "", nil, nil, nil, false, nil, 0, nil, nil, now, now, now).
			AddRow("message-00000003", 3, "assistant", "three", "completed", "gpt-5.5", nil, nil, nil, false, nil, 2, nil, nil, now, now, now))

	repo := NewChatHistoryRepository(db)
	page, err := repo.ListMessages(
		context.Background(),
		userID,
		"conversation-12345678",
		nil,
		2,
	)

	require.NoError(t, err)
	require.True(t, page.HasMore)
	require.Equal(t, int64(4), *page.NextBeforePosition)
	require.Equal(t, []int64{4, 5}, []int64{
		page.Items[0].Position,
		page.Items[1].Position,
	})
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestChatHistoryDeleteRemovesOnlyContentAndWritesTombstone(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	userID := int64(42)
	conversationID := int64(9)
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO chat_history_sync_states")).
		WithArgs(userID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT version")).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(4))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, revision, deleted_at")).
		WithArgs(userID, "conversation-12345678").
		WillReturnRows(sqlmock.NewRows([]string{"id", "revision", "deleted_at"}).
			AddRow(conversationID, 3, nil))
	mock.ExpectQuery(regexp.QuoteMeta("UPDATE chat_history_sync_states")).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(5))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM chat_messages")).
		WithArgs(userID, conversationID).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE chat_conversations")).
		WithArgs(userID, conversationID, int64(5), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO chat_history_changes")).
		WithArgs(
			userID,
			int64(5),
			"deleted",
			conversationID,
			"conversation-12345678",
			sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	repo := NewChatHistoryRepository(db)
	err = repo.DeleteConversation(
		context.Background(),
		userID,
		"conversation-12345678",
		3,
	)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestChatHistoryListRejectsMalformedCursorBeforeQuery(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := NewChatHistoryRepository(db)
	_, err = repo.ListConversations(
		context.Background(),
		42,
		"not-base64!",
		20,
		"",
	)

	require.ErrorIs(t, err, service.ErrChatHistoryInvalid)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestChatHistoryCheckpointUsesMonotonicGuardAndTreatsNoRowsAsNoOp(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectExec("UPDATE chat_messages AS message").
		WithArgs(
			int64(42),
			"attempt-12345678",
			"message-assistant-12345678",
			"delivered prefix",
			int64(7),
		).
		WillReturnResult(sqlmock.NewResult(0, 0))

	repo := NewChatHistoryRepository(db)
	err = repo.CheckpointCompletion(
		context.Background(),
		42,
		&service.CheckpointChatCompletionInput{
			AttemptID:          "attempt-12345678",
			AssistantMessageID: "message-assistant-12345678",
			Content:            "delivered prefix",
			CheckpointSeq:      7,
		},
	)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertPreparedChatAttemptStartsLeaseAtDatabaseWriteTime(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	now := time.Date(2026, 7, 25, 14, 0, 0, 0, time.UTC)
	mock.ExpectExec(
		`INSERT INTO chat_request_attempts .*NOW\(\) \+ INTERVAL '5 minutes'`,
	).
		WithArgs(
			int64(42),
			"attempt-12345678",
			"client-request-12345678",
			strings.Repeat("a", 64),
			service.ChatAttemptStatusProcessing,
			int64(23),
			"conversation-12345678",
			"message-assistant-12345678",
			now,
		).
		WillReturnResult(sqlmock.NewResult(17, 1))

	err = insertPreparedChatAttempt(
		context.Background(),
		db,
		42,
		&service.PrepareChatCompletionInput{
			AttemptID:          "attempt-12345678",
			ClientRequestID:    "client-request-12345678",
			RequestHash:        strings.Repeat("a", 64),
			ConversationID:     "conversation-12345678",
			AssistantMessageID: "message-assistant-12345678",
		},
		23,
		now,
	)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestChatHistoryFinalizeAfterDeleteOnlyTerminatesAttempt(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO chat_history_sync_states")).
		WithArgs(int64(42)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT version")).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(9))
	mock.ExpectQuery("SELECT status, assistant_message_id, assistant_message_public_id").
		WithArgs(int64(42), "attempt-12345678").
		WillReturnRows(sqlmock.NewRows([]string{
			"status",
			"assistant_message_id",
			"assistant_message_public_id",
		}).AddRow(
			service.ChatAttemptStatusProcessing,
			nil,
			"message-assistant-12345678",
		))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE chat_request_attempts")).
		WithArgs(
			int64(42),
			"attempt-12345678",
			service.ChatAttemptStatusInterrupted,
			http.StatusOK,
			"",
			"",
			sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	repo := NewChatHistoryRepository(db)
	err = repo.FinalizeCompletion(
		context.Background(),
		42,
		&service.FinalizeChatCompletionInput{
			AttemptID:          "attempt-12345678",
			AssistantMessageID: "message-assistant-12345678",
			Content:            "must not return",
			CheckpointSeq:      2,
			DeliveryStatus:     service.ChatMessageDeliveryInterrupted,
			AttemptStatus:      service.ChatAttemptStatusInterrupted,
			HTTPStatus:         http.StatusOK,
		},
	)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestChatHistoryPrepareRejectsStreamingHeadBeforeInsert(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	attemptRows := func() *sqlmock.Rows {
		return sqlmock.NewRows([]string{
			"request_hash",
			"client_request_id",
			"status",
			"conversation_public_id",
			"assistant_message_public_id",
		})
	}
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT\s+a\.request_hash`).
		WithArgs(int64(42), "attempt-12345678").
		WillReturnRows(attemptRows())
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO chat_history_sync_states")).
		WithArgs(int64(42)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT version")).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(4))
	mock.ExpectQuery(`SELECT\s+a\.request_hash`).
		WithArgs(int64(42), "attempt-12345678").
		WillReturnRows(attemptRows())
	mock.ExpectQuery(`SELECT\s+c\.id`).
		WithArgs(int64(42), "conversation-12345678").
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"revision",
			"message_count",
			"head_message_id",
			"head_public_id",
			"head_role",
			"head_status",
		}).AddRow(
			int64(9),
			int64(3),
			2,
			int64(13),
			"message-head-12345678",
			"assistant",
			service.ChatMessageDeliveryStreaming,
		))
	mock.ExpectRollback()

	expectedHead := "message-head-12345678"
	repo := NewChatHistoryRepository(db)
	_, err = repo.PrepareCompletion(
		context.Background(),
		42,
		&service.PrepareChatCompletionInput{
			ConversationID:        "conversation-12345678",
			Model:                 "gpt-5.5",
			ExpectedHeadMessageID: &expectedHead,
			UserMessage: &service.ChatCompletionHistoryUserMessage{
				ID:      "message-user-12345678",
				Content: "hello",
			},
			AssistantMessageID:  "message-assistant-12345678",
			AttemptID:           "attempt-12345678",
			ClientRequestID:     "client-request-12345678",
			RequestHash:         strings.Repeat("a", 64),
			ContextMessageLimit: 200,
		},
	)

	require.ErrorIs(t, err, service.ErrChatTurnInProgress)
	require.NoError(t, mock.ExpectationsWereMet())
}
