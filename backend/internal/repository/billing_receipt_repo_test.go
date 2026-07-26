package repository

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

var billingReceiptTestColumns = []string{
	"id",
	"usage_log_id",
	"request_id",
	"source",
	"user_id",
	"user_email",
	"api_key_id",
	"account_id",
	"subscription_id",
	"billing_type",
	"model",
	"requested_model",
	"input_tokens",
	"output_tokens",
	"cache_creation_tokens",
	"cache_read_tokens",
	"gross_amount",
	"charged_amount",
	"balance_before",
	"balance_after",
	"status",
	"overdraft",
	"failure_code",
	"failure_reason",
	"created_at",
}

func expectMissingBillingLedger(
	mock sqlmock.Sqlmock,
	requestID string,
	userID int64,
) {
	mock.ExpectQuery(regexp.QuoteMeta("FROM billing_usage_entries e")).
		WithArgs(
			requestID,
			userID,
			service.BillingReceiptSourceWebChat,
			service.APIKeyPurposeWebChat,
		).
		WillReturnError(sql.ErrNoRows)
}

func TestCanonicalizeWebChatReceiptRequestID(t *testing.T) {
	id := uuid.New()
	for _, value := range []string{id.String(), " client:" + id.String() + " "} {
		got, err := canonicalizeWebChatReceiptRequestID(value)
		require.NoError(t, err)
		require.Equal(t, "client:"+id.String(), got)
	}

	_, err := canonicalizeWebChatReceiptRequestID("not-a-uuid")
	require.ErrorIs(t, err, service.ErrBillingReceiptInvalidRequestID)
}

func TestBillingReceiptRepositoryFallsBackToPendingAttempt(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	id := uuid.NewString()
	requestID := "client:" + id
	userID := int64(42)
	expectMissingBillingLedger(mock, requestID, userID)
	mock.ExpectQuery(regexp.QuoteMeta("FROM usage_logs ul")).
		WithArgs(
			requestID,
			userID,
			service.BillingReceiptSourceWebChat,
			service.BillingReceiptStatusNotCharged,
			service.APIKeyPurposeWebChat,
		).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(regexp.QuoteMeta("FROM chat_request_attempts a")).
		WithArgs(
			userID,
			id,
			requestID,
			service.BillingReceiptSourceWebChat,
			service.BillingReceiptStatusFailed,
			service.BillingReceiptStatusNotCharged,
			service.BillingReceiptStatusPending,
		).
		WillReturnRows(sqlmock.NewRows(billingReceiptTestColumns).AddRow(
			0, nil, requestID, service.BillingReceiptSourceWebChat,
			userID, "user@example.com", 0, nil, nil, 0,
			"", "", 0, 0, 0, 0, 0, 0, nil, nil,
			service.BillingReceiptStatusPending, false, nil, nil, time.Now().UTC(),
		))

	repo := NewBillingReceiptRepository(db)
	receipt, err := repo.GetUserWebChatReceipt(context.Background(), userID, id)
	require.NoError(t, err)
	require.Equal(t, service.BillingReceiptStatusPending, receipt.Status)
	require.Equal(t, requestID, receipt.RequestID)
	require.Nil(t, receipt.UsageLogID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBillingReceiptRepositoryFallsBackToSimpleModeUsage(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	id := uuid.NewString()
	requestID := "client:" + id
	userID := int64(43)
	createdAt := time.Now().UTC()
	expectMissingBillingLedger(mock, requestID, userID)
	mock.ExpectQuery(regexp.QuoteMeta("FROM usage_logs ul")).
		WithArgs(
			requestID,
			userID,
			service.BillingReceiptSourceWebChat,
			service.BillingReceiptStatusNotCharged,
			service.APIKeyPurposeWebChat,
		).
		WillReturnRows(sqlmock.NewRows(billingReceiptTestColumns).AddRow(
			0, 99, requestID, service.BillingReceiptSourceWebChat,
			userID, "user@example.com", 7, 8, nil, service.BillingTypeBalance,
			"gpt-5.5", "gpt-5.5", 10, 20, 3, 4, 0.125, 0, nil, nil,
			service.BillingReceiptStatusNotCharged, false, nil, nil, createdAt,
		))

	repo := NewBillingReceiptRepository(db)
	receipt, err := repo.GetUserWebChatReceipt(context.Background(), userID, id)
	require.NoError(t, err)
	require.Equal(t, service.BillingReceiptStatusNotCharged, receipt.Status)
	require.Equal(t, int64(99), *receipt.UsageLogID)
	require.InDelta(t, 0.125, receipt.GrossAmount, 0.000001)
	require.Zero(t, receipt.ChargedAmount)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBillingReceiptAdminProjectionIncludesAttemptsAndSimpleUsage(t *testing.T) {
	require.Contains(t, billingReceiptAdminProjection, "FROM chat_request_attempts a")
	require.Contains(t, billingReceiptAdminProjection, "LEFT JOIN LATERAL")
	require.Contains(t, billingReceiptAdminProjection, "FROM usage_logs log")
	require.Contains(t, billingReceiptAdminProjection, "WHEN ul.id IS NOT NULL THEN 'not_charged'")
	require.Contains(t, billingReceiptAdminProjection, "WHEN a.status = 'failed' THEN 'failed'")
	require.Contains(t, billingReceiptAdminProjection, "WHEN a.status IN ('completed', 'interrupted') THEN 'not_charged'")
}
