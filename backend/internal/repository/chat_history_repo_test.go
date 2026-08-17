package repository

import (
	"context"
	"database/sql/driver"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type chatActivityPayloadMatcher struct {
	status string
	text   string
}

func (m chatActivityPayloadMatcher) Match(value driver.Value) bool {
	var payload string
	switch typed := value.(type) {
	case []byte:
		payload = string(typed)
	case string:
		payload = typed
	default:
		return false
	}
	return strings.Contains(payload, `"status":"`+m.status+`"`) &&
		strings.Contains(payload, `"text":"`+m.text+`"`)
}

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
		"internal_id",
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
			AddRow(int64(105), "message-00000005", 5, "assistant", "five", "completed", "gpt-5.5", nil, nil, nil, false, nil, 4, nil, nil, now, now, now).
			AddRow(int64(104), "message-00000004", 4, "user", "four", "completed", "", nil, nil, nil, false, nil, 0, nil, nil, now, now, now).
			AddRow(int64(103), "message-00000003", 3, "assistant", "three", "completed", "gpt-5.5", nil, nil, nil, false, nil, 2, nil, nil, now, now, now))
	mock.ExpectQuery(regexp.QuoteMeta("FROM chat_message_attachments ma")).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"message_id", "public_id", "original_name", "kind", "mime_type", "byte_size", "stored_size", "status", "expires_at",
			"page_count", "width", "height", "storage_key", "sha256", "extracted_text",
		}))
	mock.ExpectQuery(regexp.QuoteMeta("FROM chat_message_activities a")).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"message_id", "id", "response_id", "source", "activity_type", "item_id",
			"output_index", "summary_index", "sort_order", "status", "text",
			"sequence_start", "sequence_end", "reasoning_mode", "reasoning_effort",
			"started_at", "completed_at", "metadata", "created_at", "updated_at",
		}).AddRow(
			int64(105), int64(501), "resp_1", "openai_responses", "reasoning_summary", "rs_1",
			0, 0, int64(1), "completed", "summary", int64(1), int64(4), "pro", "medium",
			now, now, []byte(`{"last_event":"response.completed"}`), now, now,
		))

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
	require.Empty(t, page.Items[0].Activities)
	require.Len(t, page.Items[1].Activities, 1)
	require.Equal(t, "summary", page.Items[1].Activities[0].Text)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestChatMessageActivitiesBatchUpsertIsIdempotentAndLoadsStableOrder(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	now := time.Now().UTC()
	completedAt := now.Add(time.Second)
	activities := []service.ChatMessageActivity{{
		ResponseID:      "resp_1",
		Source:          service.ChatMessageActivitySourceOpenAIResponses,
		ActivityType:    service.ChatMessageActivityTypeReasoningSummary,
		ItemID:          "rs_1",
		OutputIndex:     0,
		SummaryIndex:    0,
		SortOrder:       1,
		Status:          service.ChatMessageActivityStatusCompleted,
		Text:            "final summary",
		SequenceStart:   2,
		SequenceEnd:     9,
		ReasoningMode:   "pro",
		ReasoningEffort: "medium",
		StartedAt:       now,
		CompletedAt:     &completedAt,
		Metadata:        []byte(`{"last_event":"response.completed"}`),
	}}

	require.Contains(t, upsertChatMessageActivitiesSQL, "jsonb_to_recordset")
	require.Contains(t, upsertChatMessageActivitiesSQL, "ON CONFLICT")
	require.Contains(t, upsertChatMessageActivitiesSQL, "EXCLUDED.sequence_end")
	for range 2 {
		mock.ExpectExec("INSERT INTO chat_message_activities").
			WithArgs(int64(77), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))
		require.NoError(t, upsertChatMessageActivities(context.Background(), db, 77, activities))
	}

	mock.ExpectQuery("FROM chat_message_activities a").
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"message_id", "id", "response_id", "source", "activity_type", "item_id",
			"output_index", "summary_index", "sort_order", "status", "text",
			"sequence_start", "sequence_end", "reasoning_mode", "reasoning_effort",
			"started_at", "completed_at", "metadata", "created_at", "updated_at",
		}).AddRow(
			int64(77), int64(700), "resp_1", "openai_responses", "reasoning_summary", "rs_1",
			0, 0, int64(1), "completed", "final summary", int64(2), int64(9), "pro", "medium",
			now, completedAt, []byte(`{"last_event":"response.completed"}`), now, completedAt,
		))
	loaded, err := loadChatMessageActivitiesForMessages(context.Background(), db, []int64{77})
	require.NoError(t, err)
	require.Len(t, loaded[77], 1)
	require.Equal(t, "final summary", loaded[77][0].Text)
	require.Equal(t, int64(9), loaded[77][0].SequenceEnd)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestChatHistoryCheckpointPersistsActivitySnapshotInOneBatch(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	now := time.Now().UTC()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT message.id").
		WithArgs(int64(42), "attempt-12345678", "message-assistant-12345678").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(77)))
	mock.ExpectExec("UPDATE chat_messages AS message").
		WithArgs(
			int64(42),
			"attempt-12345678",
			"message-assistant-12345678",
			"answer",
			int64(8),
		).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO chat_message_activities").
		WithArgs(int64(77), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	repo := NewChatHistoryRepository(db)
	err = repo.CheckpointCompletion(context.Background(), 42, &service.CheckpointChatCompletionInput{
		AttemptID:          "attempt-12345678",
		AssistantMessageID: "message-assistant-12345678",
		Content:            "answer",
		CheckpointSeq:      8,
		Activities: []service.ChatMessageActivity{
			{
				ResponseID:    "resp_1",
				Source:        service.ChatMessageActivitySourceOpenAIResponses,
				ActivityType:  service.ChatMessageActivityTypeReasoningSummary,
				ItemID:        "rs_1",
				OutputIndex:   0,
				SummaryIndex:  0,
				SortOrder:     1,
				Status:        service.ChatMessageActivityStatusInProgress,
				Text:          "one",
				SequenceStart: 1,
				SequenceEnd:   3,
				StartedAt:     now,
			},
			{
				ResponseID:    "resp_1",
				Source:        service.ChatMessageActivitySourceOpenAIResponses,
				ActivityType:  service.ChatMessageActivityTypeReasoningSummary,
				ItemID:        "rs_2",
				OutputIndex:   1,
				SummaryIndex:  0,
				SortOrder:     2,
				Status:        service.ChatMessageActivityStatusInProgress,
				Text:          "two",
				SequenceStart: 4,
				SequenceEnd:   6,
				StartedAt:     now,
			},
		},
	})

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestChatHistoryFinalizeReplayUpsertsTerminalActivitySnapshot(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	now := time.Now().UTC()
	completedAt := now.Add(time.Second)
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
			"status", "assistant_message_id", "assistant_message_public_id", "stop_requested_at",
		}).AddRow(
			service.ChatAttemptStatusCompleted,
			int64(77),
			"message-assistant-12345678",
			nil,
		))
	mock.ExpectExec("INSERT INTO chat_message_activities").
		WithArgs(int64(77), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	repo := NewChatHistoryRepository(db)
	err = repo.FinalizeCompletion(context.Background(), 42, &service.FinalizeChatCompletionInput{
		AttemptID:          "attempt-12345678",
		AssistantMessageID: "message-assistant-12345678",
		Content:            "answer",
		CheckpointSeq:      9,
		DeliveryStatus:     service.ChatMessageDeliveryCompleted,
		AttemptStatus:      service.ChatAttemptStatusCompleted,
		HTTPStatus:         http.StatusOK,
		Activities: []service.ChatMessageActivity{{
			ResponseID:      "resp_1",
			Source:          service.ChatMessageActivitySourceOpenAIResponses,
			ActivityType:    service.ChatMessageActivityTypeReasoningSummary,
			ItemID:          "rs_1",
			OutputIndex:     0,
			SummaryIndex:    0,
			SortOrder:       1,
			Status:          service.ChatMessageActivityStatusCompleted,
			Text:            "authoritative summary",
			SequenceStart:   2,
			SequenceEnd:     9,
			ReasoningMode:   "pro",
			ReasoningEffort: "medium",
			StartedAt:       now,
			CompletedAt:     &completedAt,
		}},
	})

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestChatHistoryStopCompletionPersistsAuthenticatedIntentAndPartialActivity(t *testing.T) {
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
	mock.ExpectQuery("SELECT status, assistant_message_id, stop_requested_at").
		WithArgs(int64(42), "attempt-12345678").
		WillReturnRows(sqlmock.NewRows([]string{
			"status", "assistant_message_id", "stop_requested_at",
		}).AddRow(service.ChatAttemptStatusProcessing, int64(77), nil))
	mock.ExpectExec("UPDATE chat_request_attempts").
		WithArgs(int64(42), "attempt-12345678", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT m.conversation_id, c.public_id").
		WithArgs(int64(77), int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{
			"conversation_id", "conversation_public_id",
		}).AddRow(int64(9), "conversation-12345678"))
	mock.ExpectExec("UPDATE chat_messages").
		WithArgs(int64(77), int64(42), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE chat_message_activities").
		WithArgs(int64(77), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("UPDATE chat_history_sync_states").
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(10))
	mock.ExpectExec("UPDATE chat_conversations").
		WithArgs(int64(42), int64(9), int64(10), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO chat_history_changes").
		WithArgs(int64(42), int64(10), "upsert", int64(9), "conversation-12345678", nil).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	repo, ok := NewChatHistoryRepository(db).(*chatHistoryRepository)
	require.True(t, ok)
	result, err := repo.StopCompletion(context.Background(), 42, "attempt-12345678")

	require.NoError(t, err)
	require.True(t, result.Accepted)
	require.Equal(t, service.ChatAttemptStatusInterrupted, result.AttemptStatus)
	require.Equal(t, service.ChatMessageDeliveryStopped, result.DeliveryStatus)
	require.NotNil(t, result.StoppedAt)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestChatHistoryStopCompletionDoesNotOverrideCompletedAttempt(t *testing.T) {
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
	mock.ExpectQuery("SELECT status, assistant_message_id, stop_requested_at").
		WithArgs(int64(42), "attempt-12345678").
		WillReturnRows(sqlmock.NewRows([]string{
			"status", "assistant_message_id", "stop_requested_at",
		}).AddRow(service.ChatAttemptStatusCompleted, int64(77), nil))
	mock.ExpectCommit()

	repo, ok := NewChatHistoryRepository(db).(*chatHistoryRepository)
	require.True(t, ok)
	result, err := repo.StopCompletion(context.Background(), 42, "attempt-12345678")

	require.NoError(t, err)
	require.False(t, result.Accepted)
	require.Equal(t, service.ChatAttemptStatusCompleted, result.AttemptStatus)
	require.Equal(t, service.ChatMessageDeliveryCompleted, result.DeliveryStatus)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestChatHistoryStopCompletionCorrectsPriorDisconnectedFinalize(t *testing.T) {
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
	mock.ExpectQuery("SELECT status, assistant_message_id, stop_requested_at").
		WithArgs(int64(42), "attempt-12345678").
		WillReturnRows(sqlmock.NewRows([]string{
			"status", "assistant_message_id", "stop_requested_at",
		}).AddRow(service.ChatAttemptStatusInterrupted, int64(77), nil))
	mock.ExpectExec("UPDATE chat_request_attempts").
		WithArgs(int64(42), "attempt-12345678", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT m.conversation_id, c.public_id").
		WithArgs(int64(77), int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{
			"conversation_id", "conversation_public_id",
		}).AddRow(int64(9), "conversation-12345678"))
	mock.ExpectExec("UPDATE chat_messages").
		WithArgs(int64(77), int64(42), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE chat_message_activities").
		WithArgs(int64(77), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("UPDATE chat_history_sync_states").
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(10))
	mock.ExpectExec("UPDATE chat_conversations").
		WithArgs(int64(42), int64(9), int64(10), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO chat_history_changes").
		WithArgs(int64(42), int64(10), "upsert", int64(9), "conversation-12345678", nil).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	repo, ok := NewChatHistoryRepository(db).(*chatHistoryRepository)
	require.True(t, ok)
	result, err := repo.StopCompletion(context.Background(), 42, "attempt-12345678")

	require.NoError(t, err)
	require.True(t, result.Accepted)
	require.Equal(t, service.ChatMessageDeliveryStopped, result.DeliveryStatus)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestChatHistoryLateFinalizeHonorsStoredStopIntent(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	now := time.Now().UTC()
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
			"status", "assistant_message_id", "assistant_message_public_id", "stop_requested_at",
		}).AddRow(
			service.ChatAttemptStatusInterrupted,
			int64(77),
			"message-assistant-12345678",
			now,
		))
	mock.ExpectQuery("SELECT m.conversation_id, c.public_id, m.public_id, c.deleted_at").
		WithArgs(int64(77), int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{
			"conversation_id", "conversation_public_id", "message_public_id", "deleted_at",
		}).AddRow(int64(9), "conversation-12345678", "message-assistant-12345678", nil))
	mock.ExpectExec("UPDATE chat_messages").
		WithArgs(int64(77), int64(42), "late answer", int64(9), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO chat_message_activities").
		WithArgs(int64(77), chatActivityPayloadMatcher{
			status: service.ChatMessageActivityStatusStopped,
			text:   "kept partial summary",
		}).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("UPDATE chat_history_sync_states").
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(10))
	mock.ExpectExec("UPDATE chat_conversations").
		WithArgs(int64(42), int64(9), int64(10), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO chat_history_changes").
		WithArgs(int64(42), int64(10), "upsert", int64(9), "conversation-12345678", nil).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	repo := NewChatHistoryRepository(db)
	err = repo.FinalizeCompletion(context.Background(), 42, &service.FinalizeChatCompletionInput{
		AttemptID:          "attempt-12345678",
		AssistantMessageID: "message-assistant-12345678",
		Content:            "late answer",
		CheckpointSeq:      9,
		DeliveryStatus:     service.ChatMessageDeliveryCompleted,
		AttemptStatus:      service.ChatAttemptStatusCompleted,
		FinishReason:       "stop",
		HTTPStatus:         http.StatusOK,
		Activities: []service.ChatMessageActivity{{
			ResponseID:    "resp_1",
			Source:        service.ChatMessageActivitySourceOpenAIResponses,
			ActivityType:  service.ChatMessageActivityTypeReasoningSummary,
			ItemID:        "rs_1",
			OutputIndex:   0,
			SummaryIndex:  0,
			SortOrder:     1,
			Status:        service.ChatMessageActivityStatusDisconnected,
			Text:          "kept partial summary",
			SequenceStart: 1,
			SequenceEnd:   8,
			StartedAt:     now.Add(-time.Second),
		}},
	})

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
	require.Contains(t, upsertChatMessageActivitiesSQL, "chat_message_activities.status <> 'stopped'")
}

func TestChatHistoryGetAttemptLoadsAssistantActivities(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	now := time.Now().UTC()
	mock.ExpectQuery("FROM chat_request_attempts a").
		WithArgs(int64(42), "attempt-12345678").
		WillReturnRows(sqlmock.NewRows([]string{
			"attempt_id", "conversation_public_id", "assistant_message_public_id", "client_request_id",
			"status", "http_status", "failure_code", "failure_reason", "attempt_created_at", "attempt_updated_at",
			"message_internal_id", "message_public_id", "position", "role", "content", "delivery_status",
			"requested_model", "finish_reason", "error_code", "error_message", "excluded_from_context",
			"superseded_by_message_id", "checkpoint_seq", "linked_attempt_id", "receipt_id",
			"message_created_at", "message_updated_at", "terminal_at",
		}).AddRow(
			"attempt-12345678", "conversation-12345678", "message-assistant-12345678", "receipt-12345678",
			service.ChatAttemptStatusCompleted, 200, nil, nil, now, now,
			int64(77), "message-assistant-12345678", int64(2), "assistant", "answer", "completed",
			"gpt-5.5", "stop", nil, nil, false, nil, int64(9), "attempt-12345678", "receipt-12345678",
			now, now, now,
		))
	mock.ExpectQuery("FROM chat_message_activities a").
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"message_id", "id", "response_id", "source", "activity_type", "item_id",
			"output_index", "summary_index", "sort_order", "status", "text",
			"sequence_start", "sequence_end", "reasoning_mode", "reasoning_effort",
			"started_at", "completed_at", "metadata", "created_at", "updated_at",
		}).AddRow(
			int64(77), int64(700), "resp_1", "openai_responses", "reasoning_summary", "rs_1",
			0, 0, int64(1), "completed", "summary", int64(1), int64(8), "pro", "medium",
			now, now, []byte(`{}`), now, now,
		))

	repo := NewChatHistoryRepository(db)
	attempt, err := repo.GetAttempt(context.Background(), 42, "attempt-12345678")
	require.NoError(t, err)
	require.NotNil(t, attempt.AssistantMessage)
	require.Len(t, attempt.AssistantMessage.Activities, 1)
	require.Equal(t, "summary", attempt.AssistantMessage.Activities[0].Text)
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
	mock.ExpectExec(regexp.QuoteMeta("UPDATE chat_attachments")).
		WithArgs(userID, conversationID, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 0))
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
			"stop_requested_at",
		}).AddRow(
			service.ChatAttemptStatusProcessing,
			nil,
			"message-assistant-12345678",
			nil,
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
