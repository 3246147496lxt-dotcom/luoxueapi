//go:build integration

package repository

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func newBillingReceiptWebChatFixture(t *testing.T, balance float64) (int64, int64, int64) {
	t.Helper()
	ctx := context.Background()
	client := testEntClient(t)
	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("billing-receipt-%s@example.com", uuid.NewString()),
		PasswordHash: "hash",
		Balance:      balance,
	})
	group := mustCreateGroup(t, client, &service.Group{
		Name:     "billing-receipt-" + uuid.NewString(),
		Platform: service.PlatformOpenAI,
	})
	account := mustCreateAccount(t, client, &service.Account{
		Name: "billing-receipt-" + uuid.NewString(),
		Type: service.AccountTypeAPIKey,
	})
	apiKeyID, inserted, err := insertWebChatPrincipal(
		ctx,
		integrationDB,
		user.ID,
		group.ID,
		"sk-web-chat-receipt-"+uuid.NewString(),
	)
	require.NoError(t, err)
	require.True(t, inserted)
	return user.ID, apiKeyID, account.ID
}

func insertBillingReceiptWebChatAttempt(
	t *testing.T,
	userID int64,
	clientRequestID string,
	status string,
) string {
	t.Helper()
	attemptID := "attempt-" + uuid.NewString()
	_, err := integrationDB.ExecContext(context.Background(), `
		INSERT INTO chat_request_attempts (
			user_id,
			attempt_id,
			client_request_id,
			request_hash,
			status
		)
		VALUES ($1, $2, $3, $4, $5)
	`, userID, attemptID, clientRequestID, strings.Repeat("a", 64), status)
	require.NoError(t, err)
	return attemptID
}

func TestUsageBillingRepositoryApplyWritesChargeReceiptSnapshot(t *testing.T) {
	ctx := context.Background()
	userID, apiKeyID, accountID := newBillingReceiptWebChatFixture(t, 50)
	clientRequestID := uuid.NewString()
	requestID := "client:" + clientRequestID
	insertBillingReceiptWebChatAttempt(
		t,
		userID,
		clientRequestID,
		service.ChatAttemptStatusProcessing,
	)

	billingRepo := NewUsageBillingRepository(testEntClient(t), integrationDB)
	result, err := billingRepo.Apply(ctx, &service.UsageBillingCommand{
		RequestID:           requestID,
		Source:              service.BillingReceiptSourceWebChat,
		UserID:              userID,
		APIKeyID:            apiKeyID,
		AccountID:           accountID,
		AccountType:         service.AccountTypeAPIKey,
		Model:               "gpt-5.5",
		RequestedModel:      "gpt-5.5",
		BillingType:         service.BillingTypeBalance,
		InputTokens:         120,
		OutputTokens:        30,
		CacheCreationTokens: 5,
		CacheReadTokens:     9,
		GrossCost:           1,
		BalanceCost:         1.25,
	})
	require.NoError(t, err)
	require.True(t, result.Applied)
	require.NotNil(t, result.BalanceBefore)
	require.NotNil(t, result.NewBalance)
	require.InDelta(t, 50, *result.BalanceBefore, 0.000001)
	require.InDelta(t, 48.75, *result.NewBalance, 0.000001)

	receiptRepo := NewBillingReceiptRepository(integrationDB)
	receipt, err := receiptRepo.GetUserWebChatReceipt(
		ctx,
		userID,
		requestID,
	)
	require.NoError(t, err)
	require.Equal(t, service.BillingReceiptStatusCharged, receipt.Status)
	require.Equal(t, service.BillingReceiptSourceWebChat, receipt.Source)
	require.Equal(t, "gpt-5.5", receipt.RequestedModel)
	require.Equal(t, 120, receipt.InputTokens)
	require.Equal(t, 30, receipt.OutputTokens)
	require.InDelta(t, 1.25, receipt.ChargedAmount, 0.000001)
	require.InDelta(t, 50, *receipt.BalanceBefore, 0.000001)
	require.InDelta(t, 48.75, *receipt.BalanceAfter, 0.000001)
	require.Nil(t, receipt.UsageLogID)

	var usageLogID int64
	err = integrationDB.QueryRowContext(ctx, `
		INSERT INTO usage_logs (
			user_id,
			api_key_id,
			account_id,
			request_id,
			model,
			requested_model,
			input_tokens,
			output_tokens,
			cache_creation_tokens,
			cache_read_tokens,
			total_cost,
			actual_cost,
			billing_type
		)
		VALUES ($1, $2, $3, $4, $5, $5, 120, 30, 5, 9, 1, 1.25, 0)
		RETURNING id
	`, userID, apiKeyID, accountID, requestID, "gpt-5.5").Scan(&usageLogID)
	require.NoError(t, err)

	receipt, err = receiptRepo.GetUserWebChatReceipt(ctx, userID, requestID)
	require.NoError(t, err)
	require.NotNil(t, receipt.UsageLogID)
	require.Equal(t, usageLogID, *receipt.UsageLogID)
}

func TestUsageBillingRepositoryRejectsLateWebChatBillingAfterTerminalAttempt(t *testing.T) {
	ctx := context.Background()
	userID, apiKeyID, accountID := newBillingReceiptWebChatFixture(t, 50)
	clientRequestID := uuid.NewString()
	requestID := "client:" + clientRequestID
	insertBillingReceiptWebChatAttempt(
		t,
		userID,
		clientRequestID,
		service.ChatAttemptStatusInterrupted,
	)

	billingRepo := NewUsageBillingRepository(testEntClient(t), integrationDB)
	result, err := billingRepo.Apply(ctx, &service.UsageBillingCommand{
		RequestID:      requestID,
		Source:         service.BillingReceiptSourceWebChat,
		UserID:         userID,
		APIKeyID:       apiKeyID,
		AccountID:      accountID,
		AccountType:    service.AccountTypeAPIKey,
		Model:          "gpt-5.5",
		RequestedModel: "gpt-5.5",
		BalanceCost:    1.25,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.Applied)
	require.True(t, result.SettlementClosed)

	var balance float64
	require.NoError(t, integrationDB.QueryRowContext(
		ctx,
		"SELECT balance FROM users WHERE id = $1",
		userID,
	).Scan(&balance))
	require.InDelta(t, 50, balance, 0.000001)

	var dedupCount, receiptCount int
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM usage_billing_dedup
		WHERE request_id = $1 AND api_key_id = $2
	`, requestID, apiKeyID).Scan(&dedupCount))
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM billing_usage_entries
		WHERE request_id = $1 AND api_key_id = $2
	`, requestID, apiKeyID).Scan(&receiptCount))
	require.Zero(t, dedupCount)
	require.Zero(t, receiptCount)
}

func TestBillingReceiptReadProjectionMovesFromPendingToSimpleModeUsage(t *testing.T) {
	ctx := context.Background()
	userID, apiKeyID, accountID := newBillingReceiptWebChatFixture(t, 0)
	clientRequestID := uuid.NewString()
	requestID := "client:" + clientRequestID

	_, err := integrationDB.ExecContext(ctx, `
		INSERT INTO chat_request_attempts (
			user_id,
			attempt_id,
			client_request_id,
			request_hash,
			status
		)
		VALUES ($1, $2, $3, $4, 'processing')
	`, userID, "attempt-"+uuid.NewString(), clientRequestID, strings.Repeat("a", 64))
	require.NoError(t, err)

	receiptRepo := NewBillingReceiptRepository(integrationDB)
	pending, err := receiptRepo.GetUserWebChatReceipt(ctx, userID, clientRequestID)
	require.NoError(t, err)
	require.Equal(t, service.BillingReceiptStatusPending, pending.Status)

	_, err = integrationDB.ExecContext(ctx, `
		INSERT INTO usage_logs (
			user_id,
			api_key_id,
			account_id,
			request_id,
			model,
			requested_model,
			input_tokens,
			output_tokens,
			cache_creation_tokens,
			cache_read_tokens,
			total_cost,
			actual_cost,
			billing_type
		)
		VALUES ($1, $2, $3, $4, 'gpt-5.5', 'gpt-5.5', 11, 7, 2, 3, 0.25, 0.25, 0)
	`, userID, apiKeyID, accountID, requestID)
	require.NoError(t, err)

	notCharged, err := receiptRepo.GetUserWebChatReceipt(ctx, userID, clientRequestID)
	require.NoError(t, err)
	require.Equal(t, service.BillingReceiptStatusNotCharged, notCharged.Status)
	require.Equal(t, "gpt-5.5", notCharged.RequestedModel)
	require.Equal(t, 11, notCharged.InputTokens)
	require.InDelta(t, 0.25, notCharged.GrossAmount, 0.000001)
	require.Zero(t, notCharged.ChargedAmount)

	list, err := receiptRepo.ListBillingReceipts(ctx, &service.BillingReceiptFilter{
		RequestID: clientRequestID,
		Page:      1,
		PageSize:  10,
	})
	require.NoError(t, err)
	require.Equal(t, 1, list.Total)
	require.Len(t, list.Receipts, 1)
	require.Equal(t, service.BillingReceiptStatusNotCharged, list.Receipts[0].Status)
	require.Equal(t, requestID, list.Receipts[0].RequestID)
}

func TestBillingReceiptReadProjectionReturnsNotChargedTerminalAttempt(t *testing.T) {
	for _, status := range []string{
		service.ChatAttemptStatusCompleted,
		service.ChatAttemptStatusInterrupted,
	} {
		t.Run(status, func(t *testing.T) {
			ctx := context.Background()
			userID, _, _ := newBillingReceiptWebChatFixture(t, 0)
			clientRequestID := uuid.NewString()
			requestID := "client:" + clientRequestID
			insertBillingReceiptWebChatAttempt(t, userID, clientRequestID, status)

			receiptRepo := NewBillingReceiptRepository(integrationDB)
			receipt, err := receiptRepo.GetUserWebChatReceipt(ctx, userID, clientRequestID)
			require.NoError(t, err)
			require.Equal(t, service.BillingReceiptStatusNotCharged, receipt.Status)
			require.Equal(t, requestID, receipt.RequestID)

			list, err := receiptRepo.ListBillingReceipts(ctx, &service.BillingReceiptFilter{
				RequestID: clientRequestID,
				Status:    service.BillingReceiptStatusNotCharged,
				Page:      1,
				PageSize:  10,
			})
			require.NoError(t, err)
			require.Equal(t, 1, list.Total)
			require.Len(t, list.Receipts, 1)
			require.Equal(t, service.BillingReceiptStatusNotCharged, list.Receipts[0].Status)
		})
	}
}

func TestBillingReceiptReadProjectionReturnsFailedAttempt(t *testing.T) {
	ctx := context.Background()
	userID, _, _ := newBillingReceiptWebChatFixture(t, 0)
	clientRequestID := uuid.NewString()
	requestID := "client:" + clientRequestID

	_, err := integrationDB.ExecContext(ctx, `
		INSERT INTO chat_request_attempts (
			user_id,
			attempt_id,
			client_request_id,
			request_hash,
			status,
			http_status,
			failure_code,
			failure_reason
		)
		VALUES ($1, $2, $3, $4, 'failed', 502, 'UPSTREAM_ERROR', 'upstream unavailable')
	`, userID, "attempt-"+uuid.NewString(), clientRequestID, strings.Repeat("b", 64))
	require.NoError(t, err)

	receiptRepo := NewBillingReceiptRepository(integrationDB)
	failed, err := receiptRepo.GetUserWebChatReceipt(ctx, userID, clientRequestID)
	require.NoError(t, err)
	require.Equal(t, service.BillingReceiptStatusFailed, failed.Status)
	require.NotNil(t, failed.FailureCode)
	require.NotNil(t, failed.FailureReason)
	require.Equal(t, "UPSTREAM_ERROR", *failed.FailureCode)
	require.Equal(t, "upstream unavailable", *failed.FailureReason)

	list, err := receiptRepo.ListBillingReceipts(ctx, &service.BillingReceiptFilter{
		RequestID: requestID,
		Status:    service.BillingReceiptStatusFailed,
		Page:      1,
		PageSize:  10,
	})
	require.NoError(t, err)
	require.Equal(t, 1, list.Total)
	require.Len(t, list.Receipts, 1)
	require.Equal(t, service.BillingReceiptStatusFailed, list.Receipts[0].Status)
	require.Equal(t, "UPSTREAM_ERROR", *list.Receipts[0].FailureCode)
}
