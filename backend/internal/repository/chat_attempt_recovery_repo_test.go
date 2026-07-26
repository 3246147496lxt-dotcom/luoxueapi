package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestChatAttemptRecoveryInterruptsLinkedMessageAndWritesOneSyncChange(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	expiredAt := time.Date(2026, 7, 25, 14, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id")).
		WithArgs(expiredAt, 100).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id"}).
			AddRow(int64(17), int64(42)))
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO chat_history_sync_states")).
		WithArgs(int64(42)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT version")).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(int64(7)))
	mock.ExpectQuery("SELECT status, assistant_message_id, lease_expires_at").
		WithArgs(int64(17), int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{
			"status",
			"assistant_message_id",
			"lease_expires_at",
		}).AddRow(
			service.ChatAttemptStatusProcessing,
			int64(23),
			expiredAt.Add(-time.Second),
		))
	mock.ExpectQuery("SELECT m.conversation_id, c.public_id").
		WithArgs(int64(23), int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{
			"conversation_id",
			"public_id",
		}).AddRow(int64(9), "conversation-12345678"))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE chat_messages")).
		WithArgs(int64(23), int64(42), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta("UPDATE chat_history_sync_states")).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(int64(8)))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE chat_conversations")).
		WithArgs(int64(42), int64(9), int64(8), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO chat_history_changes")).
		WithArgs(
			int64(42),
			int64(8),
			"upsert",
			int64(9),
			"conversation-12345678",
			nil,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE chat_request_attempts")).
		WithArgs(int64(17), int64(42), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	repo := NewChatAttemptRepository(nil, db)
	recovered, err := repo.RecoverExpiredLeases(
		context.Background(),
		expiredAt,
		100,
	)

	require.NoError(t, err)
	require.Equal(t, 1, recovered)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestChatAttemptRecoveryAfterConversationDeleteOnlyTerminatesAttempt(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	expiredAt := time.Date(2026, 7, 25, 14, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id")).
		WithArgs(expiredAt, 100).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id"}).
			AddRow(int64(17), int64(42)))
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO chat_history_sync_states")).
		WithArgs(int64(42)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT version")).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(int64(7)))
	mock.ExpectQuery("SELECT status, assistant_message_id, lease_expires_at").
		WithArgs(int64(17), int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{
			"status",
			"assistant_message_id",
			"lease_expires_at",
		}).AddRow(
			service.ChatAttemptStatusProcessing,
			nil,
			expiredAt.Add(-time.Second),
		))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE chat_request_attempts")).
		WithArgs(int64(17), int64(42), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	repo := NewChatAttemptRepository(nil, db)
	recovered, err := repo.RecoverExpiredLeases(
		context.Background(),
		expiredAt,
		100,
	)

	require.NoError(t, err)
	require.Equal(t, 1, recovered)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestChatAttemptRecoveryRechecksLeaseAfterLock(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	expiredAt := time.Date(2026, 7, 25, 14, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id")).
		WithArgs(expiredAt, 100).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id"}).
			AddRow(int64(17), int64(42)))
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO chat_history_sync_states")).
		WithArgs(int64(42)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT version")).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(int64(7)))
	mock.ExpectQuery("SELECT status, assistant_message_id, lease_expires_at").
		WithArgs(int64(17), int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{
			"status",
			"assistant_message_id",
			"lease_expires_at",
		}).AddRow(
			service.ChatAttemptStatusProcessing,
			int64(23),
			expiredAt.Add(service.ChatAttemptLeaseDuration),
		))
	mock.ExpectCommit()

	repo := NewChatAttemptRepository(nil, db)
	recovered, err := repo.RecoverExpiredLeases(
		context.Background(),
		expiredAt,
		100,
	)

	require.NoError(t, err)
	require.Zero(t, recovered)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestChatAttemptRenewLeaseIsFencedByProcessingStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	leaseExpiresAt := time.Date(2026, 7, 25, 14, 5, 0, 0, time.UTC)
	mock.ExpectExec(regexp.QuoteMeta("UPDATE chat_request_attempts")).
		WithArgs(int64(42), "attempt-12345678", leaseExpiresAt).
		WillReturnResult(sqlmock.NewResult(0, 0))

	repo := NewChatAttemptRepository(nil, db)
	renewed, err := repo.RenewLease(
		context.Background(),
		42,
		"attempt-12345678",
		leaseExpiresAt,
	)

	require.NoError(t, err)
	require.False(t, renewed)
	require.NoError(t, mock.ExpectationsWereMet())
}
