package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type chatAttemptRepository struct {
	db *sql.DB
}

func NewChatAttemptRepository(_ *dbent.Client, sqlDB *sql.DB) service.ChatAttemptRepository {
	return &chatAttemptRepository{db: sqlDB}
}

func (r *chatAttemptRepository) Claim(
	ctx context.Context,
	claim service.ChatAttemptClaim,
) (*service.ChatAttemptClaimResult, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("chat attempt repository db is nil")
	}

	var insertedID int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO chat_request_attempts (
			user_id,
			attempt_id,
			client_request_id,
			request_hash
		)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, attempt_id) DO NOTHING
		RETURNING id
	`,
		claim.UserID,
		strings.TrimSpace(claim.AttemptID),
		strings.TrimSpace(claim.ClientRequestID),
		strings.TrimSpace(claim.RequestHash),
	).Scan(&insertedID)
	if err == nil {
		return &service.ChatAttemptClaimResult{
			Claimed:         true,
			ClientRequestID: strings.TrimSpace(claim.ClientRequestID),
		}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	var existingRequestID, existingHash string
	if err := r.db.QueryRowContext(ctx, `
		SELECT client_request_id, request_hash
		FROM chat_request_attempts
		WHERE user_id = $1 AND attempt_id = $2
	`,
		claim.UserID,
		strings.TrimSpace(claim.AttemptID),
	).Scan(&existingRequestID, &existingHash); err != nil {
		return nil, err
	}
	if strings.TrimSpace(existingHash) != strings.TrimSpace(claim.RequestHash) {
		return nil, service.ErrChatAttemptConflict
	}
	return &service.ChatAttemptClaimResult{
		Claimed:         false,
		ClientRequestID: strings.TrimSpace(existingRequestID),
	}, nil
}

func (r *chatAttemptRepository) MarkOutcome(
	ctx context.Context,
	userID int64,
	attemptID string,
	status string,
	httpStatus int,
	failureCode string,
	failureReason string,
) error {
	if r == nil || r.db == nil {
		return errors.New("chat attempt repository db is nil")
	}
	result, err := r.db.ExecContext(ctx, `
		UPDATE chat_request_attempts
		SET
			status = $3,
			http_status = NULLIF($4, 0),
			failure_code = NULLIF($5, ''),
			failure_reason = NULLIF($6, ''),
			lease_expires_at = CASE WHEN $3 = 'processing' THEN lease_expires_at ELSE NULL END,
			updated_at = NOW()
		WHERE user_id = $1 AND attempt_id = $2
	`,
		userID,
		strings.TrimSpace(attemptID),
		strings.TrimSpace(status),
		httpStatus,
		strings.TrimSpace(failureCode),
		strings.TrimSpace(failureReason),
	)
	if err != nil {
		return err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if updated != 1 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *chatAttemptRepository) RenewLease(
	ctx context.Context,
	userID int64,
	attemptID string,
	leaseExpiresAt time.Time,
) (bool, error) {
	if r == nil || r.db == nil {
		return false, errors.New("chat attempt repository db is nil")
	}
	result, err := r.db.ExecContext(ctx, `
		UPDATE chat_request_attempts
		SET lease_expires_at = $3,
		    updated_at = NOW()
		WHERE user_id = $1
		  AND attempt_id = $2
		  AND status = 'processing'
	`, userID, strings.TrimSpace(attemptID), leaseExpiresAt.UTC())
	if err != nil {
		return false, err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return updated == 1, nil
}

type expiredChatAttemptCandidate struct {
	id     int64
	userID int64
}

func (r *chatAttemptRepository) RecoverExpiredLeases(
	ctx context.Context,
	expiredAt time.Time,
	limit int,
) (int, error) {
	if r == nil || r.db == nil {
		return 0, errors.New("chat attempt repository db is nil")
	}
	if limit <= 0 {
		return 0, nil
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id
		FROM chat_request_attempts
		WHERE status = 'processing'
		  AND conversation_public_id IS NOT NULL
		  AND assistant_message_public_id IS NOT NULL
		  AND (lease_expires_at IS NULL OR lease_expires_at <= $1)
		ORDER BY lease_expires_at ASC NULLS FIRST, id ASC
		LIMIT $2
	`, expiredAt.UTC(), limit)
	if err != nil {
		return 0, err
	}
	candidates := make([]expiredChatAttemptCandidate, 0, limit)
	for rows.Next() {
		var candidate expiredChatAttemptCandidate
		if err = rows.Scan(&candidate.id, &candidate.userID); err != nil {
			_ = rows.Close()
			return 0, err
		}
		candidates = append(candidates, candidate)
	}
	if err = rows.Err(); err != nil {
		_ = rows.Close()
		return 0, err
	}
	if err = rows.Close(); err != nil {
		return 0, err
	}

	recovered := 0
	var recoveryErr error
	for _, candidate := range candidates {
		changed, candidateErr := r.recoverExpiredLease(
			ctx,
			candidate,
			expiredAt.UTC(),
		)
		if candidateErr != nil {
			recoveryErr = errors.Join(recoveryErr, candidateErr)
			continue
		}
		if changed {
			recovered++
		}
	}
	return recovered, recoveryErr
}

func (r *chatAttemptRepository) recoverExpiredLease(
	ctx context.Context,
	candidate expiredChatAttemptCandidate,
	expiredAt time.Time,
) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()

	// Recovery follows the same global order as Finalize:
	// per-user sync state, attempt, then conversation/message.
	if _, err = lockChatHistorySyncState(ctx, tx, candidate.userID); err != nil {
		return false, err
	}

	var (
		status              string
		assistantInternalID sql.NullInt64
		leaseExpiresAt      sql.NullTime
	)
	err = tx.QueryRowContext(ctx, `
		SELECT status, assistant_message_id, lease_expires_at
		FROM chat_request_attempts
		WHERE id = $1
		  AND user_id = $2
		FOR UPDATE
	`, candidate.id, candidate.userID).Scan(
		&status,
		&assistantInternalID,
		&leaseExpiresAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if status != service.ChatAttemptStatusProcessing ||
		(leaseExpiresAt.Valid && leaseExpiresAt.Time.After(expiredAt)) {
		if err = tx.Commit(); err != nil {
			return false, err
		}
		return false, nil
	}

	now := time.Now().UTC()
	if !assistantInternalID.Valid {
		if err = interruptExpiredChatAttempt(
			ctx,
			tx,
			candidate.id,
			candidate.userID,
			now,
		); err != nil {
			return false, err
		}
		if err = tx.Commit(); err != nil {
			return false, err
		}
		return true, nil
	}

	var (
		conversationID       int64
		conversationPublicID string
	)
	err = tx.QueryRowContext(ctx, `
		SELECT m.conversation_id, c.public_id
		FROM chat_messages m
		JOIN chat_conversations c
		  ON c.id = m.conversation_id
		 AND c.user_id = m.user_id
		WHERE m.id = $1
		  AND m.user_id = $2
		  AND c.deleted_at IS NULL
		  AND m.delivery_status IN ('pending', 'streaming')
		FOR UPDATE OF c, m
	`, assistantInternalID.Int64, candidate.userID).Scan(
		&conversationID,
		&conversationPublicID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		if err = interruptExpiredChatAttempt(
			ctx,
			tx,
			candidate.id,
			candidate.userID,
			now,
		); err != nil {
			return false, err
		}
		if err = tx.Commit(); err != nil {
			return false, err
		}
		return true, nil
	}
	if err != nil {
		return false, err
	}

	result, err := tx.ExecContext(ctx, `
		UPDATE chat_messages
		SET delivery_status = 'interrupted',
		    finish_reason = COALESCE(NULLIF(finish_reason, ''), 'interrupted'),
		    updated_at = $3,
		    terminal_at = $3
		WHERE id = $1
		  AND user_id = $2
		  AND delivery_status IN ('pending', 'streaming')
	`, assistantInternalID.Int64, candidate.userID, now)
	if err != nil {
		return false, err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if updated != 1 {
		return false, fmt.Errorf("recover chat attempt %d: assistant message was not interrupted", candidate.id)
	}

	version, err := bumpChatHistorySyncVersion(ctx, tx, candidate.userID)
	if err != nil {
		return false, err
	}
	result, err = tx.ExecContext(ctx, `
		UPDATE chat_conversations
		SET revision = revision + 1,
		    version = $3,
		    updated_at = $4
		WHERE user_id = $1
		  AND id = $2
		  AND deleted_at IS NULL
	`, candidate.userID, conversationID, version, now)
	if err != nil {
		return false, err
	}
	updated, err = result.RowsAffected()
	if err != nil {
		return false, err
	}
	if updated != 1 {
		return false, fmt.Errorf("recover chat attempt %d: conversation was not updated", candidate.id)
	}
	if err = insertChatHistoryChange(
		ctx,
		tx,
		candidate.userID,
		version,
		"upsert",
		conversationID,
		conversationPublicID,
		nil,
	); err != nil {
		return false, err
	}
	if err = interruptExpiredChatAttempt(
		ctx,
		tx,
		candidate.id,
		candidate.userID,
		now,
	); err != nil {
		return false, err
	}
	if err = tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func interruptExpiredChatAttempt(
	ctx context.Context,
	tx *sql.Tx,
	attemptID int64,
	userID int64,
	now time.Time,
) error {
	result, err := tx.ExecContext(ctx, `
		UPDATE chat_request_attempts
		SET status = 'interrupted',
		    lease_expires_at = NULL,
		    updated_at = $3,
		    terminal_at = $3
		WHERE id = $1
		  AND user_id = $2
		  AND status = 'processing'
	`, attemptID, userID, now)
	if err != nil {
		return err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if updated != 1 {
		return fmt.Errorf("recover chat attempt %d: attempt was not interrupted", attemptID)
	}
	return nil
}
