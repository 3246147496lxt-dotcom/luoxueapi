//go:build integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type expiredChatAttemptFixture struct {
	userID                   int64
	attemptID                string
	clientRequestID          string
	conversationID           int64
	conversationPublicID     string
	assistantMessageID       int64
	assistantMessagePublicID string
}

func newExpiredChatAttemptFixture(t *testing.T) expiredChatAttemptFixture {
	t.Helper()
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	user := mustCreateUser(t, integrationEntClient, &service.User{
		Email: "chat-attempt-recovery-" + suffix + "@example.com",
	})
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(
			context.Background(),
			"DELETE FROM users WHERE id = $1",
			user.ID,
		)
	})

	fixture := expiredChatAttemptFixture{
		userID:                   user.ID,
		attemptID:                "attempt-" + suffix,
		clientRequestID:          "client-" + suffix,
		conversationPublicID:     "conversation-" + suffix,
		assistantMessagePublicID: "assistant-" + suffix,
	}
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		INSERT INTO chat_conversations (
			public_id,
			user_id,
			title,
			model,
			create_hash
		)
		VALUES ($1, $2, 'Recovery test', 'gpt-5.5', $3)
		RETURNING id
	`, fixture.conversationPublicID, fixture.userID, fmt.Sprintf("%064d", 1)).
		Scan(&fixture.conversationID))
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		INSERT INTO chat_messages (
			public_id,
			user_id,
			conversation_id,
			position,
			role,
			content,
			delivery_status,
			requested_model,
			checkpoint_seq
		)
		VALUES ($1, $2, $3, 1, 'assistant', 'kept prefix', 'streaming', 'gpt-5.5', 7)
		RETURNING id
	`,
		fixture.assistantMessagePublicID,
		fixture.userID,
		fixture.conversationID,
	).Scan(&fixture.assistantMessageID))
	_, err := integrationDB.ExecContext(ctx, `
		UPDATE chat_conversations
		SET head_message_id = $2,
		    message_count = 1
		WHERE id = $1
	`, fixture.conversationID, fixture.assistantMessageID)
	require.NoError(t, err)
	_, err = integrationDB.ExecContext(ctx, `
		INSERT INTO chat_request_attempts (
			user_id,
			attempt_id,
			client_request_id,
			request_hash,
			status,
			assistant_message_id,
			conversation_public_id,
			assistant_message_public_id,
			lease_expires_at
		)
		VALUES ($1, $2, $3, $4, 'processing', $5, $6, $7, $8)
	`,
		fixture.userID,
		fixture.attemptID,
		fixture.clientRequestID,
		fmt.Sprintf("%064d", 2),
		fixture.assistantMessageID,
		fixture.conversationPublicID,
		fixture.assistantMessagePublicID,
		time.Now().UTC().Add(-time.Minute),
	)
	require.NoError(t, err)
	return fixture
}

func TestChatAttemptRecoveryTwoReconcilersTransitionOnce(t *testing.T) {
	fixture := newExpiredChatAttemptFixture(t)
	expiredAt := time.Now().UTC()
	repositories := []service.ChatAttemptRepository{
		NewChatAttemptRepository(nil, integrationDB),
		NewChatAttemptRepository(nil, integrationDB),
	}

	start := make(chan struct{})
	type recoveryResult struct {
		recovered int
		err       error
	}
	results := make(chan recoveryResult, len(repositories))
	var wg sync.WaitGroup
	for _, repo := range repositories {
		wg.Add(1)
		go func(repo service.ChatAttemptRepository) {
			defer wg.Done()
			<-start
			recovered, err := repo.RecoverExpiredLeases(
				context.Background(),
				expiredAt,
				100,
			)
			results <- recoveryResult{recovered: recovered, err: err}
		}(repo)
	}
	close(start)
	wg.Wait()
	close(results)

	totalRecovered := 0
	for result := range results {
		require.NoError(t, result.err)
		totalRecovered += result.recovered
	}
	require.Equal(t, 1, totalRecovered)

	var (
		attemptStatus string
		lease         sql.NullTime
		messageStatus string
		content       string
		checkpointSeq int64
		changeCount   int
	)
	require.NoError(t, integrationDB.QueryRow(`
		SELECT status, lease_expires_at
		FROM chat_request_attempts
		WHERE user_id = $1 AND attempt_id = $2
	`, fixture.userID, fixture.attemptID).Scan(&attemptStatus, &lease))
	require.NoError(t, integrationDB.QueryRow(`
		SELECT delivery_status, content, checkpoint_seq
		FROM chat_messages
		WHERE id = $1
	`, fixture.assistantMessageID).Scan(&messageStatus, &content, &checkpointSeq))
	require.NoError(t, integrationDB.QueryRow(`
		SELECT COUNT(*)
		FROM chat_history_changes
		WHERE user_id = $1
		  AND conversation_public_id = $2
	`, fixture.userID, fixture.conversationPublicID).Scan(&changeCount))

	require.Equal(t, service.ChatAttemptStatusInterrupted, attemptStatus)
	require.False(t, lease.Valid)
	require.Equal(t, service.ChatMessageDeliveryInterrupted, messageStatus)
	require.Equal(t, "kept prefix", content)
	require.Equal(t, int64(7), checkpointSeq)
	require.Equal(t, 1, changeCount)
}

func TestChatAttemptHeartbeatAndRecoveryLinearizeOnAttempt(t *testing.T) {
	fixture := newExpiredChatAttemptFixture(t)
	repo := NewChatAttemptRepository(nil, integrationDB)
	expiredAt := time.Now().UTC()
	renewedUntil := expiredAt.Add(service.ChatAttemptLeaseDuration)

	start := make(chan struct{})
	var (
		renewed      bool
		renewErr     error
		recovered    int
		recoveryErr  error
		operationsWg sync.WaitGroup
	)
	operationsWg.Add(2)
	go func() {
		defer operationsWg.Done()
		<-start
		renewed, renewErr = repo.RenewLease(
			context.Background(),
			fixture.userID,
			fixture.attemptID,
			renewedUntil,
		)
	}()
	go func() {
		defer operationsWg.Done()
		<-start
		recovered, recoveryErr = repo.RecoverExpiredLeases(
			context.Background(),
			expiredAt,
			100,
		)
	}()
	close(start)
	operationsWg.Wait()

	require.NoError(t, renewErr)
	require.NoError(t, recoveryErr)
	var (
		status       string
		lease        sql.NullTime
		messageState string
		changeCount  int
	)
	require.NoError(t, integrationDB.QueryRow(`
		SELECT status, lease_expires_at
		FROM chat_request_attempts
		WHERE user_id = $1 AND attempt_id = $2
	`, fixture.userID, fixture.attemptID).Scan(&status, &lease))
	require.NoError(t, integrationDB.QueryRow(`
		SELECT delivery_status
		FROM chat_messages
		WHERE id = $1
	`, fixture.assistantMessageID).Scan(&messageState))
	require.NoError(t, integrationDB.QueryRow(`
		SELECT COUNT(*)
		FROM chat_history_changes
		WHERE user_id = $1
		  AND conversation_public_id = $2
	`, fixture.userID, fixture.conversationPublicID).Scan(&changeCount))

	if renewed {
		require.Zero(t, recovered)
		require.Equal(t, service.ChatAttemptStatusProcessing, status)
		require.True(t, lease.Valid)
		require.WithinDuration(t, renewedUntil, lease.Time, time.Microsecond)
		require.Equal(t, service.ChatMessageDeliveryStreaming, messageState)
		require.Zero(t, changeCount)
		return
	}
	require.Equal(t, 1, recovered)
	require.Equal(t, service.ChatAttemptStatusInterrupted, status)
	require.False(t, lease.Valid)
	require.Equal(t, service.ChatMessageDeliveryInterrupted, messageState)
	require.Equal(t, 1, changeCount)
}

func TestChatAttemptRecoveryFencesLateCheckpointAndFinalize(t *testing.T) {
	fixture := newExpiredChatAttemptFixture(t)
	attemptRepo := NewChatAttemptRepository(nil, integrationDB)
	historyRepo := NewChatHistoryRepository(integrationDB)

	recovered, err := attemptRepo.RecoverExpiredLeases(
		context.Background(),
		time.Now().UTC(),
		100,
	)
	require.NoError(t, err)
	require.Equal(t, 1, recovered)
	require.NoError(t, historyRepo.CheckpointCompletion(
		context.Background(),
		fixture.userID,
		&service.CheckpointChatCompletionInput{
			AttemptID:          fixture.attemptID,
			AssistantMessageID: fixture.assistantMessagePublicID,
			Content:            "kept prefix plus late checkpoint",
			CheckpointSeq:      8,
		},
	))
	require.NoError(t, historyRepo.FinalizeCompletion(
		context.Background(),
		fixture.userID,
		&service.FinalizeChatCompletionInput{
			AttemptID:          fixture.attemptID,
			AssistantMessageID: fixture.assistantMessagePublicID,
			Content:            "kept prefix plus late finalize",
			CheckpointSeq:      9,
			DeliveryStatus:     service.ChatMessageDeliveryCompleted,
			AttemptStatus:      service.ChatAttemptStatusCompleted,
		},
	))

	var (
		attemptStatus string
		messageStatus string
		content       string
		checkpointSeq int64
		changeCount   int
	)
	require.NoError(t, integrationDB.QueryRow(`
		SELECT status
		FROM chat_request_attempts
		WHERE user_id = $1 AND attempt_id = $2
	`, fixture.userID, fixture.attemptID).Scan(&attemptStatus))
	require.NoError(t, integrationDB.QueryRow(`
		SELECT delivery_status, content, checkpoint_seq
		FROM chat_messages
		WHERE id = $1
	`, fixture.assistantMessageID).Scan(&messageStatus, &content, &checkpointSeq))
	require.NoError(t, integrationDB.QueryRow(`
		SELECT COUNT(*)
		FROM chat_history_changes
		WHERE user_id = $1
		  AND conversation_public_id = $2
	`, fixture.userID, fixture.conversationPublicID).Scan(&changeCount))

	require.Equal(t, service.ChatAttemptStatusInterrupted, attemptStatus)
	require.Equal(t, service.ChatMessageDeliveryInterrupted, messageStatus)
	require.Equal(t, "kept prefix", content)
	require.Equal(t, int64(7), checkpointSeq)
	require.Equal(t, 1, changeCount)
}

func TestChatAttemptRecoveryAfterDeletePreservesStablePublicIDs(t *testing.T) {
	fixture := newExpiredChatAttemptFixture(t)
	attemptRepo := NewChatAttemptRepository(nil, integrationDB)
	historyRepo := NewChatHistoryRepository(integrationDB)

	require.NoError(t, historyRepo.DeleteConversation(
		context.Background(),
		fixture.userID,
		fixture.conversationPublicID,
		1,
	))
	recovered, err := attemptRepo.RecoverExpiredLeases(
		context.Background(),
		time.Now().UTC(),
		100,
	)
	require.NoError(t, err)
	require.Equal(t, 1, recovered)

	var (
		status                   string
		assistantInternalID      sql.NullInt64
		conversationPublicID     string
		assistantMessagePublicID string
		changeCount              int
	)
	require.NoError(t, integrationDB.QueryRow(`
		SELECT
			status,
			assistant_message_id,
			conversation_public_id,
			assistant_message_public_id
		FROM chat_request_attempts
		WHERE user_id = $1 AND attempt_id = $2
	`, fixture.userID, fixture.attemptID).Scan(
		&status,
		&assistantInternalID,
		&conversationPublicID,
		&assistantMessagePublicID,
	))
	require.NoError(t, integrationDB.QueryRow(`
		SELECT COUNT(*)
		FROM chat_history_changes
		WHERE user_id = $1
		  AND conversation_public_id = $2
	`, fixture.userID, fixture.conversationPublicID).Scan(&changeCount))

	require.Equal(t, service.ChatAttemptStatusInterrupted, status)
	require.False(t, assistantInternalID.Valid)
	require.Equal(t, fixture.conversationPublicID, conversationPublicID)
	require.Equal(t, fixture.assistantMessagePublicID, assistantMessagePublicID)
	require.Equal(t, 1, changeCount)
}
