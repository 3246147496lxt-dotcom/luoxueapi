//go:build unit

package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestUsageBillingRepositoryApply_RequiresSubscriptionTermIdentity(t *testing.T) {
	ctx := context.Background()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	subscriptionID := int64(42)
	result, err := (&usageBillingRepository{db: db}).Apply(ctx, &service.UsageBillingCommand{
		RequestID:        "request-without-term",
		APIKeyID:         7,
		UserID:           9,
		SubscriptionID:   &subscriptionID,
		SubscriptionCost: 2.5,
	})

	require.Nil(t, result)
	require.ErrorIs(t, err, service.ErrUsageBillingSubscriptionTermRequired)
	require.NoError(t, mock.ExpectationsWereMet(), "validation must fail before opening or mutating a transaction")
}

func TestIncrementUsageBillingSubscription_RejectsChangedTerm(t *testing.T) {
	ctx := context.Background()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	priorStartsAt := time.Date(2026, time.July, 1, 2, 3, 4, 0, time.UTC)
	currentStartsAt := priorStartsAt.Add(24 * time.Hour)
	mock.ExpectBegin()
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	mock.ExpectExec(`(?s)UPDATE user_subscriptions us.*WHERE us\.id = anchored\.id\s+AND us\.starts_at = \$5`).
		WithArgs(2.5, int64(42), int64(9), int64(7), priorStartsAt).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`(?s)SELECT starts_at\s+FROM user_subscriptions\s+WHERE id = \$1\s+AND user_id = \$2\s+AND deleted_at IS NULL`).
		WithArgs(int64(42), int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"starts_at"}).AddRow(currentStartsAt))
	mock.ExpectRollback()

	err = incrementUsageBillingSubscription(ctx, tx, 42, 9, 7, priorStartsAt, 2.5)
	require.ErrorIs(t, err, service.ErrUsageBillingSubscriptionTermMismatch)
	require.NoError(t, tx.Rollback())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertUsageBillingReceiptPersistsExactSubscriptionAmount(t *testing.T) {
	ctx := context.Background()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	subscriptionID := int64(42)
	cmd := &service.UsageBillingCommand{
		RequestID:          "request-subscription-receipt",
		RequestFingerprint: "fingerprint",
		Source:             service.BillingReceiptSourceAPI,
		UserID:             9,
		APIKeyID:           7,
		SubscriptionID:     &subscriptionID,
		BillingType:        service.BillingTypeSubscription,
		Model:              "claude",
		RequestedModel:     "claude",
		GrossCost:          1,
		SubscriptionCost:   2.5,
	}

	mock.ExpectBegin()
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	mock.ExpectQuery(`(?s)INSERT INTO billing_usage_entries.*subscription_amount.*ELSE \$21::numeric`).
		WithArgs(
			cmd.RequestID,
			cmd.APIKeyID,
			cmd.UserID,
			subscriptionID,
			cmd.BillingType,
			0.0,
			cmd.RequestFingerprint,
			cmd.Source,
			int64(0),
			cmd.Model,
			cmd.RequestedModel,
			0,
			0,
			0,
			0,
			cmd.GrossCost,
			nil,
			nil,
			service.BillingReceiptStatusSubscription,
			false,
			cmd.SubscriptionCost,
		).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))
	mock.ExpectRollback()

	require.NoError(t, insertUsageBillingReceipt(
		ctx,
		tx,
		cmd,
		&service.UsageBillingApplyResult{},
	))
	require.NoError(t, tx.Rollback())
	require.NoError(t, mock.ExpectationsWereMet())
}

const (
	conditionalBalanceDeductSQL = `(?s)UPDATE users\s+SET balance = balance - \$1,\s+updated_at = NOW\(\)\s+WHERE id = \$2 AND deleted_at IS NULL AND balance >= \$1\s+RETURNING balance \+ \$1, balance`
	overdraftBalanceDeductSQL   = `(?s)UPDATE users\s+SET balance = balance - \$1,\s+updated_at = NOW\(\)\s+WHERE id = \$2 AND deleted_at IS NULL\s+RETURNING balance \+ \$1, balance`
	lockWebChatBillingUserSQL   = `(?s)SELECT id\s+FROM users\s+WHERE id = \$1\s+AND deleted_at IS NULL\s+FOR UPDATE`
	lockWebChatAttemptSQL       = `(?s)SELECT status\s+FROM chat_request_attempts\s+WHERE client_request_id = \$1\s+AND user_id = \$2\s+FOR UPDATE`
	findUsageBillingClaimSQL    = `(?s)SELECT request_fingerprint\s+FROM usage_billing_dedup\s+WHERE request_id = \$1 AND api_key_id = \$2`
	findArchivedBillingClaimSQL = `(?s)SELECT request_fingerprint\s+FROM usage_billing_dedup_archive\s+WHERE request_id = \$1 AND api_key_id = \$2`
	reserveBatchImageHoldSQL    = `(?s)UPDATE users\s+SET balance = balance - \$1,\s+frozen_balance = COALESCE\(frozen_balance, 0\) \+ \$1,\s+updated_at = NOW\(\)\s+WHERE id = \$2 AND deleted_at IS NULL AND balance >= \$1\s+RETURNING balance, frozen_balance`
	captureBatchImageHoldSQL    = `(?s)UPDATE users\s+SET balance = balance\s+\+ CASE WHEN \$1 > \$2 THEN \$1 - \$2 ELSE 0 END\s+- CASE WHEN \$2 > \$1 THEN \$2 - \$1 ELSE 0 END,\s+frozen_balance = COALESCE\(frozen_balance, 0\) - \$1,\s+updated_at = NOW\(\)\s+WHERE id = \$3 AND deleted_at IS NULL AND COALESCE\(frozen_balance, 0\) >= \$1\s+RETURNING balance, frozen_balance`
	releaseBatchImageHoldSQL    = `(?s)UPDATE users\s+SET balance = balance \+ \$1,\s+frozen_balance = COALESCE\(frozen_balance, 0\) - \$1,\s+updated_at = NOW\(\)\s+WHERE id = \$2 AND deleted_at IS NULL AND COALESCE\(frozen_balance, 0\) >= \$1\s+RETURNING balance, frozen_balance`
	userExistsForBillingSQL     = `(?s)SELECT 1\s+FROM users\s+WHERE id = \$1 AND deleted_at IS NULL`
)

func TestUsageBillingRepositoryApply_RejectsLateWebChatBillingAfterTerminalAttempt(t *testing.T) {
	ctx := context.Background()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	requestID := "client:9c466607-6e38-42dd-9012-2a3f01b26bed"
	mock.ExpectBegin()
	mock.ExpectQuery(lockWebChatBillingUserSQL).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	mock.ExpectQuery(lockWebChatAttemptSQL).
		WithArgs("9c466607-6e38-42dd-9012-2a3f01b26bed", int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(service.ChatAttemptStatusInterrupted))
	mock.ExpectQuery(findUsageBillingClaimSQL).
		WithArgs(requestID, int64(7)).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(findArchivedBillingClaimSQL).
		WithArgs(requestID, int64(7)).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	result, err := (&usageBillingRepository{db: db}).Apply(ctx, &service.UsageBillingCommand{
		RequestID:          requestID,
		RequestFingerprint: "fingerprint",
		Source:             service.BillingReceiptSourceWebChat,
		UserID:             42,
		APIKeyID:           7,
		BalanceCost:        2.5,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.Applied)
	require.True(t, result.SettlementClosed)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageBillingRepositoryApply_DistinguishesTerminalIdempotentReplay(t *testing.T) {
	ctx := context.Background()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	requestID := "client:9c466607-6e38-42dd-9012-2a3f01b26bed"
	mock.ExpectBegin()
	mock.ExpectQuery(lockWebChatBillingUserSQL).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	mock.ExpectQuery(lockWebChatAttemptSQL).
		WithArgs("9c466607-6e38-42dd-9012-2a3f01b26bed", int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(service.ChatAttemptStatusCompleted))
	mock.ExpectQuery(findUsageBillingClaimSQL).
		WithArgs(requestID, int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"request_fingerprint"}).AddRow("fingerprint"))
	mock.ExpectQuery(`INSERT INTO usage_billing_dedup`).
		WithArgs(requestID, int64(7), "fingerprint").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`SELECT request_fingerprint\s+FROM usage_billing_dedup`).
		WithArgs(requestID, int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"request_fingerprint"}).AddRow("fingerprint"))
	mock.ExpectRollback()

	result, err := (&usageBillingRepository{db: db}).Apply(ctx, &service.UsageBillingCommand{
		RequestID:          requestID,
		RequestFingerprint: "fingerprint",
		Source:             service.BillingReceiptSourceWebChat,
		UserID:             42,
		APIKeyID:           7,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.Applied)
	require.False(t, result.SettlementClosed)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDeductUsageBillingBalance_UsesSufficientBalanceGuard(t *testing.T) {
	ctx := context.Background()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectBegin()
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	mock.ExpectQuery(conditionalBalanceDeductSQL).
		WithArgs(2.5, int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"balance_before", "balance"}).AddRow(10.0, 7.5))
	mock.ExpectCommit()

	balanceBefore, newBalance, sufficient, err := deductUsageBillingBalance(ctx, tx, 42, 2.5)
	require.NoError(t, err)
	require.True(t, sufficient)
	require.InDelta(t, 10.0, balanceBefore, 0.000001)
	require.InDelta(t, 7.5, newBalance, 0.000001)
	require.NoError(t, tx.Commit())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDeductUsageBillingBalance_RecordsOverdraftWhenGuardMisses(t *testing.T) {
	ctx := context.Background()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectBegin()
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	mock.ExpectQuery(conditionalBalanceDeductSQL).
		WithArgs(10.0, int64(42)).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(overdraftBalanceDeductSQL).
		WithArgs(10.0, int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"balance_before", "balance"}).AddRow(5.0, -5.0))
	mock.ExpectCommit()

	balanceBefore, newBalance, sufficient, err := deductUsageBillingBalance(ctx, tx, 42, 10)
	require.NoError(t, err)
	require.False(t, sufficient)
	require.InDelta(t, 5.0, balanceBefore, 0.000001)
	require.InDelta(t, -5.0, newBalance, 0.000001)
	require.NoError(t, tx.Commit())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestApplyUsageBillingEffects_FlagsBalanceOverdraft(t *testing.T) {
	ctx := context.Background()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectBegin()
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	mock.ExpectQuery(conditionalBalanceDeductSQL).
		WithArgs(10.0, int64(42)).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(overdraftBalanceDeductSQL).
		WithArgs(10.0, int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"balance_before", "balance"}).AddRow(5.0, -5.0))
	mock.ExpectCommit()

	result := &service.UsageBillingApplyResult{Applied: true}
	err = (&usageBillingRepository{}).applyUsageBillingEffects(ctx, tx, &service.UsageBillingCommand{
		UserID:      42,
		BalanceCost: 10,
	}, result)
	require.NoError(t, err)
	require.NotNil(t, result.BalanceBefore)
	require.InDelta(t, 5.0, *result.BalanceBefore, 0.000001)
	require.NotNil(t, result.NewBalance)
	require.InDelta(t, -5.0, *result.NewBalance, 0.000001)
	require.True(t, result.BalanceOverdrafted)
	require.NoError(t, tx.Commit())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDeductUsageBillingBalance_ReturnsUserNotFoundWhenNoUserUpdated(t *testing.T) {
	ctx := context.Background()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectBegin()
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	mock.ExpectQuery(conditionalBalanceDeductSQL).
		WithArgs(10.0, int64(42)).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(overdraftBalanceDeductSQL).
		WithArgs(10.0, int64(42)).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	_, _, _, err = deductUsageBillingBalance(ctx, tx, 42, 10)
	require.ErrorIs(t, err, service.ErrUserNotFound)
	require.NoError(t, tx.Rollback())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReserveUsageBillingBatchImageBalance_MovesAvailableToFrozen(t *testing.T) {
	ctx := context.Background()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectBegin()
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	mock.ExpectQuery(reserveBatchImageHoldSQL).
		WithArgs(2.5, int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"balance", "frozen_balance"}).AddRow(7.5, 2.5))
	mock.ExpectCommit()

	result, err := reserveUsageBillingBatchImageBalance(ctx, tx, &service.BatchImageBalanceHoldCommand{UserID: 42, HoldAmount: 2.5})
	require.NoError(t, err)
	require.NotNil(t, result.NewBalance)
	require.NotNil(t, result.FrozenBalance)
	require.InDelta(t, 7.5, *result.NewBalance, 0.000001)
	require.InDelta(t, 2.5, *result.FrozenBalance, 0.000001)
	require.NoError(t, tx.Commit())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReserveUsageBillingBatchImageBalance_InsufficientBalance(t *testing.T) {
	ctx := context.Background()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectBegin()
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	mock.ExpectQuery(reserveBatchImageHoldSQL).
		WithArgs(10.0, int64(42)).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(userExistsForBillingSQL).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"?column?"}).AddRow(1))
	mock.ExpectRollback()

	_, err = reserveUsageBillingBatchImageBalance(ctx, tx, &service.BatchImageBalanceHoldCommand{UserID: 42, HoldAmount: 10})
	require.ErrorIs(t, err, service.ErrBatchImageInsufficientBalance)
	require.NoError(t, tx.Rollback())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCaptureUsageBillingBatchImageBalance_ReleasesRemainder(t *testing.T) {
	ctx := context.Background()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectBegin()
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	mock.ExpectQuery(captureBatchImageHoldSQL).
		WithArgs(1.0, 0.25, int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"balance", "frozen_balance"}).AddRow(9.75, 0.0))
	mock.ExpectCommit()

	result, err := captureUsageBillingBatchImageBalance(ctx, tx, &service.BatchImageBalanceHoldCommand{UserID: 42, HoldAmount: 1, ActualAmount: 0.25})
	require.NoError(t, err)
	require.InDelta(t, 9.75, *result.NewBalance, 0.000001)
	require.InDelta(t, 0.0, *result.FrozenBalance, 0.000001)
	require.NoError(t, tx.Commit())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCaptureUsageBillingBatchImageBalance_RejectsActualCostOverHold(t *testing.T) {
	ctx := context.Background()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectBegin()
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	mock.ExpectRollback()

	_, err = captureUsageBillingBatchImageBalance(ctx, tx, &service.BatchImageBalanceHoldCommand{UserID: 42, HoldAmount: 0.5, ActualAmount: 1})
	require.ErrorIs(t, err, service.ErrBatchImageSettlementCostExceedsHold)
	require.NoError(t, tx.Rollback())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReleaseUsageBillingBatchImageBalance_ReturnsFrozenToAvailable(t *testing.T) {
	ctx := context.Background()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectBegin()
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	mock.ExpectQuery(`SELECT 1\s+FROM usage_billing_dedup\s+WHERE request_id = \$1 AND api_key_id = \$2`).
		WithArgs(service.BatchImageHoldRequestID("imgbatch_release"), int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"?column?"}).AddRow(1))
	mock.ExpectQuery(releaseBatchImageHoldSQL).
		WithArgs(1.0, int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"balance", "frozen_balance"}).AddRow(10.0, 0.0))
	mock.ExpectCommit()

	result, err := releaseUsageBillingBatchImageBalance(ctx, tx, &service.BatchImageBalanceHoldCommand{UserID: 42, APIKeyID: 7, BatchID: "imgbatch_release", HoldAmount: 1})
	require.NoError(t, err)
	require.InDelta(t, 10.0, *result.NewBalance, 0.000001)
	require.InDelta(t, 0.0, *result.FrozenBalance, 0.000001)
	require.NoError(t, tx.Commit())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReleaseUsageBillingBatchImageBalance_SkipsWhenHoldNeverReserved(t *testing.T) {
	ctx := context.Background()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectBegin()
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	// dedup 与归档表均无 hold claim：说明该 job 从未成功冻结，
	// 释放必须跳过，不得从他人冻结资金池中凭空生成余额。
	mock.ExpectQuery(`SELECT 1\s+FROM usage_billing_dedup\s+WHERE request_id = \$1 AND api_key_id = \$2`).
		WithArgs(service.BatchImageHoldRequestID("imgbatch_phantom"), int64(7)).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`SELECT 1\s+FROM usage_billing_dedup_archive\s+WHERE request_id = \$1 AND api_key_id = \$2`).
		WithArgs(service.BatchImageHoldRequestID("imgbatch_phantom"), int64(7)).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectCommit()

	result, err := releaseUsageBillingBatchImageBalance(ctx, tx, &service.BatchImageBalanceHoldCommand{UserID: 42, APIKeyID: 7, BatchID: "imgbatch_phantom", HoldAmount: 1})
	require.NoError(t, err)
	require.Nil(t, result.NewBalance)
	require.Nil(t, result.FrozenBalance)
	require.NoError(t, tx.Commit())
	require.NoError(t, mock.ExpectationsWereMet())
}
