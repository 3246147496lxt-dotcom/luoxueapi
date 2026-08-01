package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/shopspring/decimal"
)

const quotaOverviewAccountQuery = `
	SELECT
		CURRENT_TIMESTAMP,
		u.id,
		u.email,
		u.status,
		u.balance::text,
		u.frozen_balance::text
	FROM users u
	WHERE u.id = $1
	  AND u.deleted_at IS NULL
	  AND u.status = 'active'`

const quotaOverviewSpendQuery = `
	SELECT
		COALESCE(SUM(actual_cost) FILTER (WHERE created_at >= $2), 0)::text,
		COALESCE(SUM(actual_cost), 0)::text
	FROM usage_logs
	WHERE user_id = $1
	  AND created_at >= $3
	  AND created_at < $4`

const quotaOverviewPeriodUsageQuery = `
	SELECT
		FLOOR(EXTRACT(EPOCH FROM (ul.created_at - $3::timestamptz)) / 86400)::integer AS bucket_index,
		COUNT(*)::bigint AS requests,
		COALESCE(SUM(ul.cache_read_tokens), 0)::bigint AS cache_hit_tokens,
		(
			COALESCE(SUM(ul.input_tokens), 0)::bigint
			+ COALESCE(SUM(ul.cache_creation_tokens), 0)::bigint
		) AS cache_miss_tokens,
		COALESCE(SUM(ul.output_tokens), 0)::bigint AS output_tokens
	FROM usage_logs ul
	WHERE ul.user_id = $1
	  AND ul.subscription_id = $2
	  AND ul.created_at >= $3
	  AND ul.created_at < $4
	  AND ` + usageLogSuccessFilterUL + `
	GROUP BY 1
	ORDER BY 1`

const (
	quotaOverviewPeriodUsageSavepoint         = "quota_overview_period_usage"
	quotaOverviewPeriodUsageSavepointCreate   = "SAVEPOINT " + quotaOverviewPeriodUsageSavepoint
	quotaOverviewPeriodUsageSavepointRollback = "ROLLBACK TO SAVEPOINT " + quotaOverviewPeriodUsageSavepoint
	quotaOverviewPeriodUsageSavepointRelease  = "RELEASE SAVEPOINT " + quotaOverviewPeriodUsageSavepoint
)

// This is intentionally a narrow projection. In particular, the SQL driver
// never receives the complete credential: masking is performed by PostgreSQL.
const quotaOverviewKeysQuery = `
	SELECT
		k.id,
		k.name,
		(
			CASE WHEN k.key LIKE 'sk-%' THEN 'sk-' ELSE '' END
			|| '••••'
			|| CASE WHEN char_length(k.key) >= 4 THEN right(k.key, 4) ELSE '' END
		) AS masked_key,
		k.status,
		k.group_id,
		k.quota::text,
		k.quota_used::text,
		k.expires_at,
		g.id,
		g.name,
		g.status,
		g.subscription_type,
		(g.deleted_at IS NOT NULL) AS group_deleted
	FROM api_keys k
	LEFT JOIN groups g ON g.id = k.group_id
	WHERE k.user_id = $1
	  AND k.deleted_at IS NULL
	  AND k.purpose = 'user'
	  AND k.status <> 'disabled'
	ORDER BY k.id`

// Keep exactly one continuous-term record per subscription group. A live row
// wins; otherwise the latest soft-deleted row is returned as "revoked".
const quotaOverviewSubscriptionsQuery = `
	WITH ranked_subscriptions AS (
		SELECT
			us.*,
			ROW_NUMBER() OVER (
				PARTITION BY us.group_id
				ORDER BY
					CASE WHEN us.deleted_at IS NULL THEN 0 ELSE 1 END,
					us.updated_at DESC,
					us.id DESC
			) AS row_number
		FROM user_subscriptions us
		WHERE us.user_id = $1
	)
	SELECT
		us.id,
		us.group_id,
		g.name,
		g.status,
		(g.deleted_at IS NOT NULL) AS group_deleted,
		us.status,
		(us.deleted_at IS NOT NULL) AS revoked,
		us.starts_at,
		us.expires_at,
		us.weekly_window_start,
		g.weekly_limit_usd::text,
		us.weekly_usage_usd::text,
		us.monthly_window_start,
		g.monthly_limit_usd::text,
		us.monthly_usage_usd::text,
		us.updated_at
	FROM ranked_subscriptions us
	JOIN groups g ON g.id = us.group_id
	WHERE us.row_number = 1
	  AND g.subscription_type = 'subscription'
	ORDER BY us.group_id`

type quotaOverviewRepository struct {
	db *sql.DB
}

func NewQuotaOverviewRepository(db *sql.DB) service.QuotaOverviewRepository {
	return &quotaOverviewRepository{db: db}
}

func (r *quotaOverviewRepository) LoadQuotaOverviewSnapshot(
	ctx context.Context,
	userID int64,
	displayTimezone string,
) (*service.QuotaOverviewSnapshot, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("quota overview repository db is nil")
	}
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("begin quota overview snapshot: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	snapshot := &service.QuotaOverviewSnapshot{}
	var balanceText, frozenBalanceText string
	err = tx.QueryRowContext(ctx, quotaOverviewAccountQuery, userID).Scan(
		&snapshot.AsOf,
		&snapshot.Account.ID,
		&snapshot.Account.Email,
		&snapshot.Account.Status,
		&balanceText,
		&frozenBalanceText,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query quota overview account: %w", err)
	}
	snapshot.AsOf = snapshot.AsOf.UTC()
	if snapshot.Account.Balance, err = parseQuotaOverviewDecimal(balanceText); err != nil {
		return nil, fmt.Errorf("parse quota overview balance: %w", err)
	}
	if snapshot.Account.FrozenBalance, err = parseQuotaOverviewDecimal(frozenBalanceText); err != nil {
		return nil, fmt.Errorf("parse quota overview reserved balance: %w", err)
	}

	todayStart, monthStart, err := quotaOverviewSpendStarts(snapshot.AsOf, displayTimezone)
	if err != nil {
		return nil, fmt.Errorf("resolve quota overview spend windows: %w", err)
	}
	var todaySpendText, monthSpendText string
	err = tx.QueryRowContext(
		ctx,
		quotaOverviewSpendQuery,
		userID,
		todayStart,
		monthStart,
		snapshot.AsOf,
	).Scan(&todaySpendText, &monthSpendText)
	if err != nil {
		return nil, fmt.Errorf("query quota overview spend: %w", err)
	}
	if snapshot.TodaySpend, err = parseQuotaOverviewDecimal(todaySpendText); err != nil {
		return nil, fmt.Errorf("parse quota overview today spend: %w", err)
	}
	if snapshot.MonthSpend, err = parseQuotaOverviewDecimal(monthSpendText); err != nil {
		return nil, fmt.Errorf("parse quota overview month spend: %w", err)
	}

	if snapshot.Keys, err = loadQuotaOverviewKeys(ctx, tx, userID); err != nil {
		return nil, err
	}
	if snapshot.Subscriptions, err = loadQuotaOverviewSubscriptions(ctx, tx, userID); err != nil {
		return nil, err
	}
	for i := range snapshot.Subscriptions {
		subscription := &snapshot.Subscriptions[i]
		if !canLoadQuotaOverviewPeriodUsage(*subscription, snapshot.AsOf) {
			continue
		}
		periodUsage, unavailable, err := tryLoadQuotaOverviewPeriodUsage(
			ctx,
			tx,
			userID,
			*subscription,
			snapshot.AsOf,
		)
		if err != nil {
			return nil, err
		}
		if unavailable {
			snapshot.PeriodUsageAggregationError = true
			continue
		}
		subscription.PeriodUsage = periodUsage
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit quota overview snapshot: %w", err)
	}
	return snapshot, nil
}

func canLoadQuotaOverviewPeriodUsage(
	sub service.QuotaOverviewSubscriptionSnapshot,
	asOf time.Time,
) bool {
	if sub.Revoked ||
		sub.Status != service.SubscriptionStatusActive ||
		sub.StartsAt.IsZero() ||
		sub.ExpiresAt.IsZero() ||
		!sub.ExpiresAt.After(sub.StartsAt) ||
		asOf.Before(sub.StartsAt) ||
		!asOf.Before(sub.ExpiresAt) ||
		sub.WeeklyLimit == nil ||
		!sub.WeeklyLimit.GreaterThan(decimal.Zero) {
		return false
	}
	periodStart, _, ok := service.AnchoredWeeklyWindow(sub.StartsAt, asOf)
	if !ok || sub.WeeklyWindowStart == nil {
		return false
	}
	if sub.WeeklyWindowStart.Equal(periodStart) {
		return true
	}
	// Request admission treats a counter from an exact older anchored window
	// as zero until synchronous maintenance advances it. Load current-period
	// usage here as the proof needed by the service to make the same narrow
	// read-only projection. Missing, future and off-anchor markers stay
	// unavailable.
	if !sub.WeeklyWindowStart.Before(periodStart) {
		return false
	}
	storedStart, _, ok := service.AnchoredWeeklyWindow(
		sub.StartsAt,
		*sub.WeeklyWindowStart,
	)
	return ok && sub.WeeklyWindowStart.Equal(storedStart)
}

func tryLoadQuotaOverviewPeriodUsage(
	ctx context.Context,
	tx *sql.Tx,
	userID int64,
	sub service.QuotaOverviewSubscriptionSnapshot,
	asOf time.Time,
) (*service.QuotaOverviewPeriodUsageSnapshot, bool, error) {
	if _, err := tx.ExecContext(ctx, quotaOverviewPeriodUsageSavepointCreate); err != nil {
		return nil, false, fmt.Errorf("create quota overview period usage savepoint: %w", err)
	}

	usage, queryErr := loadQuotaOverviewPeriodUsage(ctx, tx, userID, sub, asOf)
	if queryErr != nil {
		if _, err := tx.ExecContext(ctx, quotaOverviewPeriodUsageSavepointRollback); err != nil {
			return nil, false, fmt.Errorf(
				"rollback quota overview period usage savepoint after %v: %w",
				queryErr,
				err,
			)
		}
		if _, err := tx.ExecContext(ctx, quotaOverviewPeriodUsageSavepointRelease); err != nil {
			return nil, false, fmt.Errorf(
				"release failed quota overview period usage savepoint after %v: %w",
				queryErr,
				err,
			)
		}
		if ctx.Err() != nil {
			return nil, false, fmt.Errorf("query quota overview period usage: %w", queryErr)
		}
		return nil, true, nil
	}
	if _, err := tx.ExecContext(ctx, quotaOverviewPeriodUsageSavepointRelease); err != nil {
		return nil, false, fmt.Errorf("release quota overview period usage savepoint: %w", err)
	}
	return usage, false, nil
}

func loadQuotaOverviewPeriodUsage(
	ctx context.Context,
	tx *sql.Tx,
	userID int64,
	sub service.QuotaOverviewSubscriptionSnapshot,
	asOf time.Time,
) (*service.QuotaOverviewPeriodUsageSnapshot, error) {
	periodStart, periodEnd, ok := service.AnchoredWeeklyWindow(sub.StartsAt, asOf)
	if !ok {
		return nil, errors.New("subscription period unavailable")
	}
	observedUntil := minQuotaOverviewTime(asOf, periodEnd, sub.ExpiresAt)
	usage := &service.QuotaOverviewPeriodUsageSnapshot{
		PeriodStart:   periodStart.UTC(),
		PeriodEnd:     periodEnd.UTC(),
		ObservedUntil: observedUntil.UTC(),
	}
	rows, err := tx.QueryContext(
		ctx,
		quotaOverviewPeriodUsageQuery,
		userID,
		sub.ID,
		periodStart,
		observedUntil,
	)
	if err != nil {
		return nil, fmt.Errorf("query quota overview subscription %d period usage: %w", sub.ID, err)
	}
	defer rows.Close()

	seen := [7]bool{}
	for rows.Next() {
		var (
			bucketIndex int
			bucket      service.QuotaOverviewPeriodUsageBucketSnapshot
		)
		if err := rows.Scan(
			&bucketIndex,
			&bucket.Requests,
			&bucket.CacheHitTokens,
			&bucket.CacheMissTokens,
			&bucket.OutputTokens,
		); err != nil {
			return nil, fmt.Errorf(
				"scan quota overview subscription %d period usage: %w",
				sub.ID,
				err,
			)
		}
		if bucketIndex < 0 ||
			bucketIndex >= len(usage.Buckets) ||
			seen[bucketIndex] ||
			bucket.Requests < 0 ||
			bucket.CacheHitTokens < 0 ||
			bucket.CacheMissTokens < 0 ||
			bucket.OutputTokens < 0 {
			return nil, fmt.Errorf(
				"invalid quota overview subscription %d period usage bucket %d",
				sub.ID,
				bucketIndex,
			)
		}
		seen[bucketIndex] = true
		usage.Buckets[bucketIndex] = bucket
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate quota overview subscription %d period usage: %w",
			sub.ID,
			err,
		)
	}
	return usage, nil
}

func minQuotaOverviewTime(values ...time.Time) time.Time {
	if len(values) == 0 {
		return time.Time{}
	}
	minimum := values[0]
	for i := 1; i < len(values); i++ {
		if values[i].Before(minimum) {
			minimum = values[i]
		}
	}
	return minimum
}

func quotaOverviewSpendStarts(asOf time.Time, displayTimezone string) (time.Time, time.Time, error) {
	if strings.TrimSpace(displayTimezone) == "" {
		displayTimezone = "UTC"
	}
	location, err := time.LoadLocation(displayTimezone)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	local := asOf.In(location)
	todayStart := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
	monthStart := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, location)
	return todayStart.UTC(), monthStart.UTC(), nil
}

func loadQuotaOverviewKeys(ctx context.Context, tx *sql.Tx, userID int64) ([]service.QuotaOverviewKeySnapshot, error) {
	rows, err := tx.QueryContext(ctx, quotaOverviewKeysQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("query quota overview keys: %w", err)
	}
	defer rows.Close()

	keys := make([]service.QuotaOverviewKeySnapshot, 0)
	for rows.Next() {
		var (
			key               service.QuotaOverviewKeySnapshot
			groupID           sql.NullInt64
			joinedGroupID     sql.NullInt64
			groupName         sql.NullString
			groupStatus       sql.NullString
			groupSubscription sql.NullString
			groupDeleted      bool
			quotaText         string
			quotaUsedText     string
			expiresAt         sql.NullTime
		)
		if err := rows.Scan(
			&key.ID,
			&key.Name,
			&key.MaskedKey,
			&key.Status,
			&groupID,
			&quotaText,
			&quotaUsedText,
			&expiresAt,
			&joinedGroupID,
			&groupName,
			&groupStatus,
			&groupSubscription,
			&groupDeleted,
		); err != nil {
			return nil, fmt.Errorf("scan quota overview key: %w", err)
		}
		if key.Quota, err = parseQuotaOverviewDecimal(quotaText); err != nil {
			return nil, fmt.Errorf("parse quota overview key %d limit: %w", key.ID, err)
		}
		if key.QuotaUsed, err = parseQuotaOverviewDecimal(quotaUsedText); err != nil {
			return nil, fmt.Errorf("parse quota overview key %d usage: %w", key.ID, err)
		}
		if groupID.Valid {
			id := groupID.Int64
			key.GroupID = &id
		}
		if joinedGroupID.Valid {
			key.Group = &service.QuotaOverviewGroupSnapshot{
				ID:               joinedGroupID.Int64,
				Name:             groupName.String,
				Status:           groupStatus.String,
				SubscriptionType: groupSubscription.String,
				Deleted:          groupDeleted,
			}
		}
		if expiresAt.Valid {
			value := expiresAt.Time.UTC()
			key.ExpiresAt = &value
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate quota overview keys: %w", err)
	}
	return keys, nil
}

func loadQuotaOverviewSubscriptions(ctx context.Context, tx *sql.Tx, userID int64) ([]service.QuotaOverviewSubscriptionSnapshot, error) {
	rows, err := tx.QueryContext(ctx, quotaOverviewSubscriptionsQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("query quota overview subscriptions: %w", err)
	}
	defer rows.Close()

	subscriptions := make([]service.QuotaOverviewSubscriptionSnapshot, 0)
	for rows.Next() {
		var (
			sub                service.QuotaOverviewSubscriptionSnapshot
			weeklyWindowStart  sql.NullTime
			weeklyLimitText    sql.NullString
			weeklyUsedText     string
			monthlyWindowStart sql.NullTime
			monthlyLimitText   sql.NullString
			monthlyUsedText    string
		)
		if err := rows.Scan(
			&sub.ID,
			&sub.GroupID,
			&sub.GroupName,
			&sub.GroupStatus,
			&sub.GroupDeleted,
			&sub.Status,
			&sub.Revoked,
			&sub.StartsAt,
			&sub.ExpiresAt,
			&weeklyWindowStart,
			&weeklyLimitText,
			&weeklyUsedText,
			&monthlyWindowStart,
			&monthlyLimitText,
			&monthlyUsedText,
			&sub.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan quota overview subscription: %w", err)
		}
		sub.StartsAt = sub.StartsAt.UTC()
		sub.ExpiresAt = sub.ExpiresAt.UTC()
		sub.UpdatedAt = sub.UpdatedAt.UTC()
		if weeklyWindowStart.Valid {
			value := weeklyWindowStart.Time.UTC()
			sub.WeeklyWindowStart = &value
		}
		if weeklyLimitText.Valid {
			value, err := parseQuotaOverviewDecimal(weeklyLimitText.String)
			if err != nil {
				return nil, fmt.Errorf("parse quota overview subscription %d limit: %w", sub.ID, err)
			}
			sub.WeeklyLimit = &value
		}
		if sub.WeeklyUsed, err = parseQuotaOverviewDecimal(weeklyUsedText); err != nil {
			return nil, fmt.Errorf("parse quota overview subscription %d usage: %w", sub.ID, err)
		}
		if monthlyWindowStart.Valid {
			value := monthlyWindowStart.Time.UTC()
			sub.MonthlyWindowStart = &value
		}
		if monthlyLimitText.Valid {
			value, err := parseQuotaOverviewDecimal(monthlyLimitText.String)
			if err != nil {
				return nil, fmt.Errorf("parse quota overview subscription %d monthly limit: %w", sub.ID, err)
			}
			sub.MonthlyLimit = &value
		}
		if sub.MonthlyUsed, err = parseQuotaOverviewDecimal(monthlyUsedText); err != nil {
			return nil, fmt.Errorf("parse quota overview subscription %d monthly usage: %w", sub.ID, err)
		}
		subscriptions = append(subscriptions, sub)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate quota overview subscriptions: %w", err)
	}
	return subscriptions, nil
}

func parseQuotaOverviewDecimal(value string) (decimal.Decimal, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return decimal.Zero, errors.New("empty decimal")
	}
	parsed, err := decimal.NewFromString(value)
	if err != nil {
		return decimal.Zero, err
	}
	return parsed, nil
}
