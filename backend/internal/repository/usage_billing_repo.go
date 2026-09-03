package repository

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type usageBillingRepository struct {
	db *sql.DB
}

func NewUsageBillingRepository(_ *dbent.Client, sqlDB *sql.DB) service.UsageBillingRepository {
	return &usageBillingRepository{db: sqlDB}
}

func (r *usageBillingRepository) Apply(ctx context.Context, cmd *service.UsageBillingCommand) (_ *service.UsageBillingApplyResult, err error) {
	if cmd == nil {
		return &service.UsageBillingApplyResult{}, nil
	}
	if r == nil || r.db == nil {
		return nil, errors.New("usage billing repository db is nil")
	}

	cmd.Normalize()
	if err := cmd.ValidateMonetaryFields(); err != nil {
		return nil, err
	}
	if cmd.RequestID == "" {
		return nil, service.ErrUsageBillingRequestIDRequired
	}
	if cmd.SubscriptionCost > 0 {
		if cmd.SubscriptionID == nil {
			return nil, service.ErrSubscriptionBillingContextRequired
		}
		if cmd.SubscriptionStartsAt == nil || cmd.SubscriptionStartsAt.IsZero() {
			return nil, service.ErrUsageBillingSubscriptionTermRequired
		}
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	settlementClosed, err := r.lockWebChatAttemptForBilling(ctx, tx, cmd)
	if err != nil {
		return nil, err
	}
	if settlementClosed {
		return &service.UsageBillingApplyResult{SettlementClosed: true}, nil
	}

	applied, err := r.claimUsageBillingKey(ctx, tx, cmd)
	if err != nil {
		return nil, err
	}
	if !applied {
		return &service.UsageBillingApplyResult{Applied: false}, nil
	}

	result := &service.UsageBillingApplyResult{Applied: true, EffectsKnown: true}
	if err := r.applyUsageBillingEffects(ctx, tx, cmd, result); err != nil {
		return nil, err
	}
	if err := insertUsageBillingReceipt(ctx, tx, cmd, result); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	tx = nil
	return result, nil
}

func (r *usageBillingRepository) lockWebChatAttemptForBilling(
	ctx context.Context,
	tx *sql.Tx,
	cmd *service.UsageBillingCommand,
) (bool, error) {
	if cmd == nil || cmd.Source != service.BillingReceiptSourceWebChat {
		return false, nil
	}

	const clientRequestPrefix = "client:"
	if !strings.HasPrefix(cmd.RequestID, clientRequestPrefix) {
		return false, service.ErrUsageBillingAttemptNotFound
	}
	clientRequestID := strings.TrimSpace(strings.TrimPrefix(cmd.RequestID, clientRequestPrefix))
	if clientRequestID == "" || cmd.UserID <= 0 {
		return false, service.ErrUsageBillingAttemptNotFound
	}

	var lockedUserID int64
	err := tx.QueryRowContext(ctx, `
		SELECT id
		FROM users
		WHERE id = $1
		  AND deleted_at IS NULL
		FOR UPDATE
	`, cmd.UserID).Scan(&lockedUserID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, service.ErrUserNotFound
	}
	if err != nil {
		return false, err
	}

	var status string
	err = tx.QueryRowContext(ctx, `
		SELECT status
		FROM chat_request_attempts
		WHERE client_request_id = $1
		  AND user_id = $2
		FOR UPDATE
	`, clientRequestID, cmd.UserID).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return false, service.ErrUsageBillingAttemptNotFound
	}
	if err != nil {
		return false, err
	}
	if status == service.ChatAttemptStatusProcessing {
		return false, nil
	}

	exists, err := usageBillingClaimExists(
		ctx,
		tx,
		cmd.RequestID,
		cmd.APIKeyID,
		cmd.RequestFingerprint,
	)
	if err != nil {
		return false, err
	}
	return !exists, nil
}

func usageBillingClaimExists(
	ctx context.Context,
	tx *sql.Tx,
	requestID string,
	apiKeyID int64,
	requestFingerprint string,
) (bool, error) {
	for _, table := range []string{"usage_billing_dedup", "usage_billing_dedup_archive"} {
		var existingFingerprint string
		err := tx.QueryRowContext(ctx, `
			SELECT request_fingerprint
			FROM `+table+`
			WHERE request_id = $1 AND api_key_id = $2
		`, requestID, apiKeyID).Scan(&existingFingerprint)
		if err == nil {
			if strings.TrimSpace(existingFingerprint) != strings.TrimSpace(requestFingerprint) {
				return false, service.ErrUsageBillingRequestConflict
			}
			return true, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return false, err
		}
	}
	return false, nil
}

func (r *usageBillingRepository) claimUsageBillingKey(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand) (bool, error) {
	return r.claimUsageBillingRequest(ctx, tx, cmd.RequestID, cmd.APIKeyID, cmd.RequestFingerprint)
}

func (r *usageBillingRepository) claimUsageBillingRequest(ctx context.Context, tx *sql.Tx, requestID string, apiKeyID int64, requestFingerprint string) (bool, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `
		INSERT INTO usage_billing_dedup (request_id, api_key_id, request_fingerprint)
		VALUES ($1, $2, $3)
		ON CONFLICT (request_id, api_key_id) DO NOTHING
		RETURNING id
	`, requestID, apiKeyID, requestFingerprint).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		var existingFingerprint string
		if err := tx.QueryRowContext(ctx, `
			SELECT request_fingerprint
			FROM usage_billing_dedup
			WHERE request_id = $1 AND api_key_id = $2
		`, requestID, apiKeyID).Scan(&existingFingerprint); err != nil {
			return false, err
		}
		if strings.TrimSpace(existingFingerprint) != strings.TrimSpace(requestFingerprint) {
			return false, service.ErrUsageBillingRequestConflict
		}
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var archivedFingerprint string
	err = tx.QueryRowContext(ctx, `
		SELECT request_fingerprint
		FROM usage_billing_dedup_archive
		WHERE request_id = $1 AND api_key_id = $2
	`, requestID, apiKeyID).Scan(&archivedFingerprint)
	if err == nil {
		if strings.TrimSpace(archivedFingerprint) != strings.TrimSpace(requestFingerprint) {
			return false, service.ErrUsageBillingRequestConflict
		}
		return false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	return true, nil
}

func (r *usageBillingRepository) ReserveBatchImageBalance(ctx context.Context, cmd *service.BatchImageBalanceHoldCommand) (*service.BatchImageBalanceHoldResult, error) {
	return r.applyBatchImageBalanceHold(ctx, cmd, reserveUsageBillingBatchImageBalance, false)
}

func (r *usageBillingRepository) CaptureBatchImageBalance(ctx context.Context, cmd *service.BatchImageBalanceHoldCommand) (*service.BatchImageBalanceHoldResult, error) {
	return r.applyBatchImageBalanceHold(ctx, cmd, captureUsageBillingBatchImageBalance, true)
}

func (r *usageBillingRepository) ReleaseBatchImageBalance(ctx context.Context, cmd *service.BatchImageBalanceHoldCommand) (*service.BatchImageBalanceHoldResult, error) {
	return r.applyBatchImageBalanceHold(ctx, cmd, releaseUsageBillingBatchImageBalance, true)
}

func (r *usageBillingRepository) applyBatchImageBalanceHold(
	ctx context.Context,
	cmd *service.BatchImageBalanceHoldCommand,
	apply func(context.Context, *sql.Tx, *service.BatchImageBalanceHoldCommand) (*service.BatchImageBalanceHoldResult, error),
	requireReservation bool,
) (_ *service.BatchImageBalanceHoldResult, err error) {
	if cmd == nil {
		return &service.BatchImageBalanceHoldResult{}, nil
	}
	if r == nil || r.db == nil {
		return nil, errors.New("usage billing repository db is nil")
	}
	cmd.Normalize()
	if err := cmd.ValidateMonetaryFields(); err != nil {
		return nil, err
	}
	if cmd.RequestID == "" {
		return nil, service.ErrUsageBillingRequestIDRequired
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	applied, err := r.claimUsageBillingRequest(ctx, tx, cmd.RequestID, cmd.APIKeyID, cmd.RequestFingerprint)
	if err != nil {
		return nil, err
	}
	if !applied {
		return &service.BatchImageBalanceHoldResult{Applied: false}, nil
	}
	// A capture/release is allowed to move funds only when this batch's
	// reservation is present and still describes the same principal and hold
	// amount.  The operation dedup claim is intentionally made first: an
	// already-applied replay returns without requiring the batch row to remain
	// queryable (for example, after archival), while a new operation fails closed
	// before touching frozen_balance.
	if requireReservation {
		if err := validateBatchImageHoldReservation(ctx, tx, cmd); err != nil {
			return nil, err
		}
	}

	result, err := apply(ctx, tx, cmd)
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = &service.BatchImageBalanceHoldResult{}
	}
	result.Applied = true

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	tx = nil
	return result, nil
}

func (r *usageBillingRepository) applyUsageBillingEffects(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand, result *service.UsageBillingApplyResult) error {
	if cmd.SubscriptionCost > 0 && cmd.SubscriptionID != nil {
		if err := incrementUsageBillingSubscription(
			ctx,
			tx,
			*cmd.SubscriptionID,
			cmd.UserID,
			cmd.APIKeyID,
			*cmd.SubscriptionStartsAt,
			cmd.SubscriptionCost,
		); err != nil {
			return err
		}
		result.SubscriptionChargedCost = cmd.SubscriptionCost
		result.SubscriptionCharged = true
	}

	if cmd.BalanceCost > 0 {
		balanceBefore, newBalance, sufficient, err := deductUsageBillingBalance(ctx, tx, cmd.UserID, cmd.BalanceCost)
		if err != nil {
			return err
		}
		result.BalanceBefore = &balanceBefore
		result.NewBalance = &newBalance
		result.BalanceOverdrafted = !sufficient
		result.BalanceChargedCost = cmd.BalanceCost
		result.BalanceCharged = true
	}

	if cmd.APIKeyQuotaCost > 0 {
		exhausted, err := incrementUsageBillingAPIKeyQuota(ctx, tx, cmd.APIKeyID, cmd.APIKeyQuotaCost)
		if err != nil {
			return err
		}
		result.APIKeyQuotaExhausted = exhausted
		result.APIKeyQuotaChargedCost = cmd.APIKeyQuotaCost
		result.APIKeyQuotaCharged = true
	}

	if cmd.APIKeyRateLimitCost > 0 {
		if err := incrementUsageBillingAPIKeyRateLimit(ctx, tx, cmd.APIKeyID, cmd.APIKeyRateLimitCost); err != nil {
			return err
		}
		result.APIKeyRateLimitChargedCost = cmd.APIKeyRateLimitCost
		result.APIKeyRateLimitCharged = true
	}

	if cmd.AccountQuotaCost > 0 && (strings.EqualFold(cmd.AccountType, service.AccountTypeAPIKey) || strings.EqualFold(cmd.AccountType, service.AccountTypeBedrock)) {
		quotaState, err := incrementUsageBillingAccountQuota(ctx, tx, cmd.AccountID, cmd.AccountQuotaCost)
		if err != nil {
			return err
		}
		result.QuotaState = quotaState
		result.AccountQuotaChargedCost = cmd.AccountQuotaCost
		result.AccountQuotaCharged = true
	}

	return nil
}

func incrementUsageBillingSubscription(
	ctx context.Context,
	tx *sql.Tx,
	subscriptionID, userID, apiKeyID int64,
	subscriptionStartsAt time.Time,
	costUSD float64,
) error {
	const updateSQL = `
		WITH anchored AS (
			SELECT
				us.id,
				us.starts_at
					+ FLOOR(
						GREATEST(
							0,
							EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - us.starts_at))
						) / 604800
					) * 604800 * INTERVAL '1 second' AS weekly_period_start,
				us.starts_at
					+ FLOOR(
						GREATEST(
							0,
							EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - us.starts_at))
						) / 2592000
					) * 2592000 * INTERVAL '1 second' AS monthly_period_start
			FROM user_subscriptions us
			JOIN groups g
				ON g.id = us.group_id
				AND g.deleted_at IS NULL
				AND g.subscription_type = 'subscription'
			JOIN api_keys ak
				ON ak.id = $4
				AND ak.deleted_at IS NULL
				AND ak.user_id = us.user_id
				AND ak.group_id = us.group_id
			WHERE us.id = $2
				AND us.user_id = $3
				AND us.deleted_at IS NULL
		)
		UPDATE user_subscriptions us
		SET
			weekly_usage_usd = CASE
				WHEN us.weekly_window_start = anchored.weekly_period_start
					THEN us.weekly_usage_usd + $1
				ELSE $1
			END,
			weekly_window_start = anchored.weekly_period_start,
			monthly_usage_usd = CASE
				WHEN us.monthly_window_start = anchored.monthly_period_start
					THEN us.monthly_usage_usd + $1
				ELSE $1
			END,
			monthly_window_start = anchored.monthly_period_start,
			updated_at = NOW()
		FROM anchored
		WHERE us.id = anchored.id
			AND us.starts_at = $5
	`
	res, err := tx.ExecContext(
		ctx,
		updateSQL,
		costUSD,
		subscriptionID,
		userID,
		apiKeyID,
		subscriptionStartsAt,
	)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected > 0 {
		return nil
	}

	var currentStartsAt time.Time
	err = tx.QueryRowContext(ctx, `
		SELECT starts_at
		FROM user_subscriptions
		WHERE id = $1
		  AND user_id = $2
		  AND deleted_at IS NULL
	`, subscriptionID, userID).Scan(&currentStartsAt)
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrSubscriptionNotFound
	}
	if err != nil {
		return err
	}
	if !currentStartsAt.Equal(subscriptionStartsAt) {
		return service.ErrUsageBillingSubscriptionTermMismatch
	}
	return service.ErrSubscriptionNotFound
}

func deductUsageBillingBalance(ctx context.Context, tx *sql.Tx, userID int64, amount float64) (float64, float64, bool, error) {
	var balanceBefore, newBalance float64
	err := tx.QueryRowContext(ctx, `
		UPDATE users
		SET balance = balance - $1,
			updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL AND balance >= $1
		RETURNING balance + $1, balance
	`, amount, userID).Scan(&balanceBefore, &newBalance)
	if err == nil {
		return balanceBefore, newBalance, true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, 0, false, err
	}

	err = tx.QueryRowContext(ctx, `
		UPDATE users
		SET balance = balance - $1,
			updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
		RETURNING balance + $1, balance
	`, amount, userID).Scan(&balanceBefore, &newBalance)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, false, service.ErrUserNotFound
	}
	if err != nil {
		return 0, 0, false, err
	}
	return balanceBefore, newBalance, false, nil
}

func insertUsageBillingReceipt(
	ctx context.Context,
	tx *sql.Tx,
	cmd *service.UsageBillingCommand,
	result *service.UsageBillingApplyResult,
) error {
	if cmd == nil {
		return nil
	}

	var receiptID int64
	err := tx.QueryRowContext(ctx, `
		INSERT INTO billing_usage_entries (
			usage_log_id,
			user_id,
			api_key_id,
			subscription_id,
			subscription_amount,
			billing_type,
			applied,
			delta_usd,
			request_id,
			request_fingerprint,
			source,
			account_id,
			model,
			requested_model,
			input_tokens,
			output_tokens,
			cache_creation_tokens,
			cache_read_tokens,
			gross_amount,
			charged_amount,
			balance_before,
			balance_after,
			status,
			overdraft,
			created_at
		)
		SELECT
			(
				SELECT ul.id
				FROM usage_logs ul
				WHERE ul.request_id = $1 AND ul.api_key_id = $2
				LIMIT 1
			),
			ak.user_id,
			ak.id,
			$4::bigint,
			CASE
				WHEN $4::bigint IS NULL THEN NULL
				ELSE $21::numeric
			END,
			$5::smallint,
			TRUE,
			-($6::numeric),
			$1::varchar,
			$7::varchar,
			$8::varchar,
			NULLIF($9::bigint, 0),
			$10::varchar,
			$11::varchar,
			$12::integer,
			$13::integer,
			$14::integer,
			$15::integer,
			$16::numeric,
			$6::numeric,
			$17::numeric,
			$18::numeric,
			$19::varchar,
			$20::boolean,
			NOW()
		FROM api_keys ak
		WHERE ak.id = $2
		  AND ($3::bigint = 0 OR ak.user_id = $3::bigint)
		RETURNING id
	`,
		cmd.RequestID,
		cmd.APIKeyID,
		cmd.UserID,
		cmd.SubscriptionID,
		cmd.BillingType,
		cmd.BalanceCost,
		cmd.RequestFingerprint,
		cmd.Source,
		cmd.AccountID,
		cmd.Model,
		cmd.RequestedModel,
		cmd.InputTokens,
		cmd.OutputTokens,
		cmd.CacheCreationTokens,
		cmd.CacheReadTokens,
		cmd.GrossCost,
		result.BalanceBefore,
		result.NewBalance,
		cmd.ReceiptStatus(),
		result.BalanceOverdrafted,
		cmd.SubscriptionCost,
	).Scan(&receiptID)
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrUsageBillingPrincipalMismatch
	}
	return err
}

func reserveUsageBillingBatchImageBalance(ctx context.Context, tx *sql.Tx, cmd *service.BatchImageBalanceHoldCommand) (*service.BatchImageBalanceHoldResult, error) {
	if cmd.HoldAmount <= 0 {
		return &service.BatchImageBalanceHoldResult{}, nil
	}
	var balance, frozen float64
	err := tx.QueryRowContext(ctx, `
		UPDATE users
		SET balance = balance - $1,
			frozen_balance = COALESCE(frozen_balance, 0) + $1,
			updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL AND balance >= $1
		RETURNING balance, frozen_balance
	`, cmd.HoldAmount, cmd.UserID).Scan(&balance, &frozen)
	if err == nil {
		return &service.BatchImageBalanceHoldResult{NewBalance: &balance, FrozenBalance: &frozen}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if exists, existsErr := userExistsForBilling(ctx, tx, cmd.UserID); existsErr != nil {
		return nil, existsErr
	} else if !exists {
		return nil, service.ErrUserNotFound
	}
	return nil, service.ErrBatchImageInsufficientBalance
}

func captureUsageBillingBatchImageBalance(ctx context.Context, tx *sql.Tx, cmd *service.BatchImageBalanceHoldCommand) (*service.BatchImageBalanceHoldResult, error) {
	if cmd.HoldAmount <= 0 && cmd.ActualAmount <= 0 {
		return &service.BatchImageBalanceHoldResult{}, nil
	}
	if cmd.ActualAmount-cmd.HoldAmount > 0.00000001 {
		return nil, service.ErrBatchImageSettlementCostExceedsHold
	}
	var balance, frozen float64
	err := tx.QueryRowContext(ctx, `
		UPDATE users
		SET balance = balance
				+ CASE WHEN $1 > $2 THEN $1 - $2 ELSE 0 END
				- CASE WHEN $2 > $1 THEN $2 - $1 ELSE 0 END,
			frozen_balance = COALESCE(frozen_balance, 0) - $1,
			updated_at = NOW()
		WHERE id = $3 AND deleted_at IS NULL AND COALESCE(frozen_balance, 0) >= $1
		RETURNING balance, frozen_balance
	`, cmd.HoldAmount, cmd.ActualAmount, cmd.UserID).Scan(&balance, &frozen)
	if err == nil {
		return &service.BatchImageBalanceHoldResult{NewBalance: &balance, FrozenBalance: &frozen}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if exists, existsErr := userExistsForBilling(ctx, tx, cmd.UserID); existsErr != nil {
		return nil, existsErr
	} else if !exists {
		return nil, service.ErrUserNotFound
	}
	return nil, errors.New("batch image frozen balance is insufficient")
}

func releaseUsageBillingBatchImageBalance(ctx context.Context, tx *sql.Tx, cmd *service.BatchImageBalanceHoldCommand) (*service.BatchImageBalanceHoldResult, error) {
	if cmd.HoldAmount <= 0 {
		return &service.BatchImageBalanceHoldResult{}, nil
	}
	// 释放前校验该 job 确实预留过 hold（hold request id 已被 claim），
	// 防止从未成功冻结的 job 触发"幻影释放"，从其他用户的冻结资金池中凭空生成余额。
	held, heldErr := batchImageHoldClaimExists(ctx, tx, service.BatchImageHoldRequestID(cmd.BatchID), cmd.APIKeyID)
	if heldErr != nil {
		return nil, heldErr
	}
	if !held {
		logger.LegacyPrintf("repository.usage_billing", "[BatchImage] release skipped, hold was never reserved: batch=%s", cmd.BatchID)
		return &service.BatchImageBalanceHoldResult{}, nil
	}
	var balance, frozen float64
	err := tx.QueryRowContext(ctx, `
		UPDATE users
		SET balance = balance + $1,
			frozen_balance = COALESCE(frozen_balance, 0) - $1,
			updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL AND COALESCE(frozen_balance, 0) >= $1
		RETURNING balance, frozen_balance
	`, cmd.HoldAmount, cmd.UserID).Scan(&balance, &frozen)
	if err == nil {
		return &service.BatchImageBalanceHoldResult{NewBalance: &balance, FrozenBalance: &frozen}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if exists, existsErr := userExistsForBilling(ctx, tx, cmd.UserID); existsErr != nil {
		return nil, existsErr
	} else if !exists {
		return nil, service.ErrUserNotFound
	}
	return nil, errors.New("batch image frozen balance is insufficient")
}

// validateBatchImageHoldReservation verifies the immutable reservation
// context before a capture/release update can touch users.frozen_balance.
//
// The dedup row is keyed by (batch-image hold request id, API key id), while
// the batch row is the authoritative source for the user/API-key pairing and
// the amount that was frozen.  Checking both inside the same transaction
// prevents a caller with a stale or tampered job snapshot from consuming a
// different user's frozen funds (or only releasing part of its own hold).
func validateBatchImageHoldReservation(ctx context.Context, tx *sql.Tx, cmd *service.BatchImageBalanceHoldCommand) error {
	if tx == nil || cmd == nil || strings.TrimSpace(cmd.BatchID) == "" || cmd.UserID <= 0 || cmd.APIKeyID <= 0 || cmd.HoldAmount < 0 || cmd.ActualAmount < 0 {
		return service.ErrUsageBillingHoldReservationInvalid
	}

	var (
		jobUserID    int64
		jobAPIKeyID  sql.NullInt64
		jobHold      float64
		jobRequestID sql.NullString
	)
	err := tx.QueryRowContext(ctx, `
		SELECT user_id, api_key_id, COALESCE(hold_amount, estimated_cost, 0), request_hash
		FROM batch_image_jobs
		WHERE batch_id = $1
		FOR UPDATE
	`, cmd.BatchID).Scan(&jobUserID, &jobAPIKeyID, &jobHold, &jobRequestID)
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrUsageBillingHoldReservationInvalid
	}
	if err != nil {
		return err
	}
	jobHoldAmount := service.QuantizeUsageBillingAmount(jobHold)
	cmdHoldAmount := service.QuantizeUsageBillingAmount(cmd.HoldAmount)
	if !jobAPIKeyID.Valid || jobUserID != cmd.UserID || jobAPIKeyID.Int64 != cmd.APIKeyID ||
		math.IsNaN(jobHold) || math.IsInf(jobHold, 0) ||
		jobHoldAmount != cmdHoldAmount {
		return service.ErrUsageBillingHoldReservationInvalid
	}
	// A zero-priced batch never creates a reserve dedup claim because the
	// reserve operation is intentionally a no-op.  Permit its matching
	// zero-cost capture after validating the immutable job snapshot, but reject
	// any attempt to smuggle a positive settlement through a zero hold.
	zeroValueReservation := jobHoldAmount == 0 && cmdHoldAmount == 0
	if zeroValueReservation && service.QuantizeUsageBillingAmount(cmd.ActualAmount) != 0 {
		return service.ErrUsageBillingHoldReservationInvalid
	}
	if !zeroValueReservation && (jobHoldAmount <= 0 || cmdHoldAmount <= 0) {
		return service.ErrUsageBillingHoldReservationInvalid
	}

	// Keep the explicit API-key ownership check separate from the batch-row
	// check.  It protects against a corrupted batch row that pairs a valid key
	// with another user's id, and intentionally does not filter deleted_at so a
	// soft-deleted key can still release funds held before deletion.
	var ownerID int64
	err = tx.QueryRowContext(ctx, `
		SELECT user_id
		FROM api_keys
		WHERE id = $1
	`, cmd.APIKeyID).Scan(&ownerID)
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrUsageBillingHoldReservationInvalid
	}
	if err != nil {
		return err
	}
	if ownerID != cmd.UserID {
		return service.ErrUsageBillingHoldReservationInvalid
	}
	if zeroValueReservation {
		return nil
	}

	// request_hash is selected above to make the lock cover the complete
	// reservation snapshot.  The operation fingerprint may intentionally differ
	// (capture uses a manifest hash), so it is not compared here.
	_ = jobRequestID
	held, err := batchImageHoldClaimExists(ctx, tx, service.BatchImageHoldRequestID(cmd.BatchID), cmd.APIKeyID)
	if err != nil {
		return err
	}
	if !held {
		return service.ErrUsageBillingHoldReservationInvalid
	}
	return nil
}

// batchImageHoldClaimExists 检查 hold request id 是否已在 dedup（或归档）表中被 claim，
// 即该 batch 的冻结操作确实成功提交过。
func batchImageHoldClaimExists(ctx context.Context, tx *sql.Tx, holdRequestID string, apiKeyID int64) (bool, error) {
	var exists int
	err := tx.QueryRowContext(ctx, `
		SELECT 1
		FROM usage_billing_dedup
		WHERE request_id = $1 AND api_key_id = $2
		FOR UPDATE
	`, holdRequestID, apiKeyID).Scan(&exists)
	if err == nil {
		return true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	err = tx.QueryRowContext(ctx, `
		SELECT 1
		FROM usage_billing_dedup_archive
		WHERE request_id = $1 AND api_key_id = $2
		FOR UPDATE
	`, holdRequestID, apiKeyID).Scan(&exists)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return false, err
}

func userExistsForBilling(ctx context.Context, tx *sql.Tx, userID int64) (bool, error) {
	var exists int
	err := tx.QueryRowContext(ctx, `
		SELECT 1
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`, userID).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func incrementUsageBillingAPIKeyQuota(ctx context.Context, tx *sql.Tx, apiKeyID int64, amount float64) (bool, error) {
	var exhausted bool
	err := tx.QueryRowContext(ctx, `
		UPDATE api_keys
		SET quota_used = quota_used + $1,
			status = CASE
				WHEN quota > 0
					AND status = $3
					AND quota_used < quota
					AND quota_used + $1 >= quota
				THEN $4
				ELSE status
			END,
			updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
		RETURNING quota > 0 AND quota_used >= quota AND quota_used - $1 < quota
	`, amount, apiKeyID, service.StatusAPIKeyActive, service.StatusAPIKeyQuotaExhausted).Scan(&exhausted)
	if errors.Is(err, sql.ErrNoRows) {
		return false, service.ErrAPIKeyNotFound
	}
	if err != nil {
		return false, err
	}
	return exhausted, nil
}

func incrementUsageBillingAPIKeyRateLimit(ctx context.Context, tx *sql.Tx, apiKeyID int64, cost float64) error {
	res, err := tx.ExecContext(ctx, `
		UPDATE api_keys SET
			usage_5h = CASE WHEN window_5h_start IS NOT NULL AND window_5h_start + INTERVAL '5 hours' <= NOW() THEN $1 ELSE usage_5h + $1 END,
			usage_1d = CASE WHEN window_1d_start IS NOT NULL AND window_1d_start + INTERVAL '24 hours' <= NOW() THEN $1 ELSE usage_1d + $1 END,
			usage_7d = CASE WHEN window_7d_start IS NOT NULL AND window_7d_start + INTERVAL '7 days' <= NOW() THEN $1 ELSE usage_7d + $1 END,
			window_5h_start = CASE WHEN window_5h_start IS NULL OR window_5h_start + INTERVAL '5 hours' <= NOW() THEN NOW() ELSE window_5h_start END,
			window_1d_start = CASE WHEN window_1d_start IS NULL OR window_1d_start + INTERVAL '24 hours' <= NOW() THEN date_trunc('day', NOW()) ELSE window_1d_start END,
			window_7d_start = CASE WHEN window_7d_start IS NULL OR window_7d_start + INTERVAL '7 days' <= NOW() THEN date_trunc('day', NOW()) ELSE window_7d_start END,
			updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
	`, cost, apiKeyID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrAPIKeyNotFound
	}
	return nil
}

func incrementUsageBillingAccountQuota(ctx context.Context, tx *sql.Tx, accountID int64, amount float64) (*service.AccountQuotaState, error) {
	rows, err := tx.QueryContext(ctx,
		`UPDATE accounts SET extra = (
			COALESCE(extra, '{}'::jsonb)
			|| jsonb_build_object('quota_used', COALESCE((extra->>'quota_used')::numeric, 0) + $1)
			|| CASE WHEN COALESCE((extra->>'quota_daily_limit')::numeric, 0) > 0 THEN
				jsonb_build_object(
					'quota_daily_used',
					CASE WHEN `+dailyExpiredExpr+`
					THEN $1
					ELSE COALESCE((extra->>'quota_daily_used')::numeric, 0) + $1 END,
					'quota_daily_start',
					CASE WHEN `+dailyExpiredExpr+`
					THEN `+nowUTC+`
					ELSE COALESCE(extra->>'quota_daily_start', `+nowUTC+`) END
				)
				|| CASE WHEN `+dailyExpiredExpr+` AND `+nextDailyResetAtExpr+` IS NOT NULL
				   THEN jsonb_build_object('quota_daily_reset_at', `+nextDailyResetAtExpr+`)
				   ELSE '{}'::jsonb END
			ELSE '{}'::jsonb END
			|| CASE WHEN COALESCE((extra->>'quota_weekly_limit')::numeric, 0) > 0 THEN
				jsonb_build_object(
					'quota_weekly_used',
					CASE WHEN `+weeklyExpiredExpr+`
					THEN $1
					ELSE COALESCE((extra->>'quota_weekly_used')::numeric, 0) + $1 END,
					'quota_weekly_start',
					CASE WHEN `+weeklyExpiredExpr+`
					THEN `+nowUTC+`
					ELSE COALESCE(extra->>'quota_weekly_start', `+nowUTC+`) END
				)
				|| CASE WHEN `+weeklyExpiredExpr+` AND `+nextWeeklyResetAtExpr+` IS NOT NULL
				   THEN jsonb_build_object('quota_weekly_reset_at', `+nextWeeklyResetAtExpr+`)
				   ELSE '{}'::jsonb END
			ELSE '{}'::jsonb END
		), updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
		RETURNING
			COALESCE((extra->>'quota_used')::numeric, 0),
			COALESCE((extra->>'quota_limit')::numeric, 0),
			COALESCE((extra->>'quota_daily_used')::numeric, 0),
			COALESCE((extra->>'quota_daily_limit')::numeric, 0),
			COALESCE((extra->>'quota_weekly_used')::numeric, 0),
			COALESCE((extra->>'quota_weekly_limit')::numeric, 0)`,
		amount, accountID)
	if err != nil {
		return nil, err
	}

	var state service.AccountQuotaState
	if rows.Next() {
		if err := rows.Scan(
			&state.TotalUsed, &state.TotalLimit,
			&state.DailyUsed, &state.DailyLimit,
			&state.WeeklyUsed, &state.WeeklyLimit,
		); err != nil {
			_ = rows.Close()
			return nil, err
		}
	} else {
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return nil, err
		}
		_ = rows.Close()
		return nil, service.ErrAccountNotFound
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	// 必须在执行下一条 SQL 前显式关闭 rows：pq 驱动在同一连接上
	// 不允许前一条查询的结果集未耗尽时启动新查询，否则会返回
	// "unexpected Parse response" 错误。
	if err := rows.Close(); err != nil {
		return nil, err
	}
	// 任意维度额度在本次递增中从"未超"跨越到"已超"时，必须刷新调度快照，
	// 否则 Redis 中缓存的 Account 仍显示旧的 used 值，后续请求会继续选中本账号，
	// 最终观察到 daily_used / weekly_used 大幅超过配置的 limit。
	// 对于日/周额度，即使本次触发了周期重置（pre=0、post=amount），
	// 判定式 (post-amount) < limit 同样成立，逻辑与总额度保持一致。
	crossedTotal := state.TotalLimit > 0 && state.TotalUsed >= state.TotalLimit && (state.TotalUsed-amount) < state.TotalLimit
	crossedDaily := state.DailyLimit > 0 && state.DailyUsed >= state.DailyLimit && (state.DailyUsed-amount) < state.DailyLimit
	crossedWeekly := state.WeeklyLimit > 0 && state.WeeklyUsed >= state.WeeklyLimit && (state.WeeklyUsed-amount) < state.WeeklyLimit
	if crossedTotal || crossedDaily || crossedWeekly {
		if err := enqueueSchedulerOutbox(ctx, tx, service.SchedulerOutboxEventAccountChanged, &accountID, nil, nil); err != nil {
			logger.LegacyPrintf("repository.usage_billing", "[SchedulerOutbox] enqueue quota exceeded failed: account=%d err=%v", accountID, err)
			return nil, err
		}
	}
	return &state, nil
}
