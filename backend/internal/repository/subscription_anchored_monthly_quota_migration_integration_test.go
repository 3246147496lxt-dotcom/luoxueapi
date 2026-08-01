//go:build integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	dbmigrations "github.com/Wei-Shaw/sub2api/migrations"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

type migration195Fixture struct {
	userID         int64
	otherUserID    int64
	groupID        int64
	apiKeyID       int64
	otherAPIKeyID  int64
	accountID      int64
	subscriptionID int64
	startsAt       time.Time
	weeklyStart    time.Time
	monthlyStart   time.Time
	occurredAt     time.Time
}

type migration195BlockerCounts struct {
	receipts         int
	subscriptionLogs int
	activePlans      int
	times            int
}

func migration195RequestID(label string) string {
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")
	maxLabelLength := 64 - len(suffix) - 1
	if len(label) > maxLabelLength {
		label = label[:maxLabelLength]
	}
	return label + "-" + suffix
}

func newMigration195Fixture(t *testing.T, tx *sql.Tx, age time.Duration) migration195Fixture {
	t.Helper()
	ctx := context.Background()
	suffix := uuid.NewString()

	var databaseNow time.Time
	require.NoError(t, tx.QueryRowContext(ctx, "SELECT CURRENT_TIMESTAMP").Scan(&databaseNow))
	startsAt := databaseNow.Add(-age)
	weeklyStart := startsAt.Add(
		(databaseNow.Sub(startsAt) / (7 * 24 * time.Hour)) * (7 * 24 * time.Hour),
	)
	monthlyStart := startsAt.Add(
		(databaseNow.Sub(startsAt) / (30 * 24 * time.Hour)) * (30 * 24 * time.Hour),
	)

	var fixture migration195Fixture
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO users (email, password_hash, role, status, balance, concurrency)
VALUES ($1, 'migration-195-test', 'user', 'active', 0, 5)
RETURNING id
`, "migration-195-"+suffix+"@example.com").Scan(&fixture.userID))
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO groups (
    name,
    platform,
    subscription_type,
    status,
    weekly_limit_usd,
    monthly_limit_usd
)
VALUES ($1, 'openai', 'subscription', 'active', 100, 400)
RETURNING id
`, "migration-195-"+suffix).Scan(&fixture.groupID))
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO accounts (name, platform, type, status)
VALUES ($1, 'openai', 'oauth', 'active')
RETURNING id
`, "migration-195-"+suffix).Scan(&fixture.accountID))
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO api_keys (user_id, key, name, group_id, status)
VALUES ($1, $2, 'migration-195', $3, 'active')
RETURNING id
`, fixture.userID, "sk-migration-195-"+suffix, fixture.groupID).Scan(&fixture.apiKeyID))
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO user_subscriptions (
    user_id,
    group_id,
    starts_at,
    expires_at,
    status,
    weekly_window_start,
    monthly_window_start,
    weekly_usage_usd,
    monthly_usage_usd
)
VALUES ($1, $2, $3, $4, 'active', $5, $6, 17, 999)
RETURNING id
`,
		fixture.userID,
		fixture.groupID,
		startsAt,
		databaseNow.Add(90*24*time.Hour),
		weeklyStart,
		monthlyStart,
	).Scan(&fixture.subscriptionID))

	fixture.startsAt = startsAt
	fixture.weeklyStart = weeklyStart
	fixture.monthlyStart = monthlyStart
	fixture.occurredAt = databaseNow.Add(-time.Hour)
	return fixture
}

func insertMigration195UsageLog(
	t *testing.T,
	tx *sql.Tx,
	fixture migration195Fixture,
	requestID string,
	billingType int,
	actualCost float64,
) int64 {
	t.Helper()

	var usageLogID int64
	require.NoError(t, tx.QueryRowContext(context.Background(), `
INSERT INTO usage_logs (
    user_id,
    api_key_id,
    account_id,
    request_id,
    model,
    group_id,
    subscription_id,
    total_cost,
    actual_cost,
    billing_type,
    created_at
)
VALUES ($1, $2, $3, $4, 'migration-195-model', $5, $6, $7, $7, $8, $9)
RETURNING id
`,
		fixture.userID,
		fixture.apiKeyID,
		fixture.accountID,
		requestID,
		fixture.groupID,
		fixture.subscriptionID,
		actualCost,
		billingType,
		fixture.occurredAt,
	).Scan(&usageLogID))
	return usageLogID
}

func insertMigration195LegacyReceipt(
	t *testing.T,
	tx *sql.Tx,
	fixture migration195Fixture,
	usageLogID int64,
	requestID string,
	amount float64,
) {
	t.Helper()

	_, err := tx.ExecContext(context.Background(), `
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
    gross_amount,
    charged_amount,
    status,
    overdraft,
    created_at
)
VALUES (
    $1, $2, $3, $4, NULL, 1, TRUE, -($5::DECIMAL(20, 10)), $6, $7, 'api', $8,
    'migration-195-model', 'migration-195-model', $5, 0, 'subscription', FALSE, $9
)
`,
		usageLogID,
		fixture.userID,
		fixture.apiKeyID,
		fixture.subscriptionID,
		amount,
		requestID,
		strings.Repeat("a", 64),
		fixture.accountID,
		fixture.occurredAt,
	)
	require.NoError(t, err)
}

func applyMigration195(t *testing.T, tx *sql.Tx) error {
	t.Helper()
	migrationSQL, err := dbmigrations.FS.ReadFile("195_subscription_anchored_monthly_quota.sql")
	require.NoError(t, err)
	_, err = tx.ExecContext(context.Background(), string(migrationSQL))
	return err
}

func requireMigration195Blockers(
	t *testing.T,
	err error,
	want migration195BlockerCounts,
) {
	t.Helper()
	require.Error(t, err)

	var postgresErr *pq.Error
	require.ErrorAs(t, err, &postgresErr)
	require.Equal(t, "P0001", string(postgresErr.Code))
	require.Equal(t, fmt.Sprintf(
		"cannot rebuild anchored subscription quota: %d invalid current-window receipt(s), %d invalid current-window subscription usage log(s), %d invalid active plan(s), %d invalid subscription time range(s)",
		want.receipts,
		want.subscriptionLogs,
		want.activePlans,
		want.times,
	), postgresErr.Message)
}

func TestMigration195RebuildsMonthlyUsageWithoutBalanceLogLeakageOrReceiptDoubleCount(t *testing.T) {
	tx := testTx(t)
	ctx := context.Background()
	fixture := newMigration195Fixture(t, tx, 40*24*time.Hour)

	insertMigration195UsageLog(t, tx, fixture, migration195RequestID("legacy-sub"), 1, 3)
	insertMigration195UsageLog(t, tx, fixture, migration195RequestID("balance"), 0, 90)
	receiptRequestID := migration195RequestID("receipt")
	receiptLogID := insertMigration195UsageLog(t, tx, fixture, receiptRequestID, 1, 4)
	insertMigration195LegacyReceipt(t, tx, fixture, receiptLogID, receiptRequestID, 4)
	_, err := tx.ExecContext(ctx, `
UPDATE user_subscriptions
SET weekly_usage_usd = 7
WHERE id = $1
`, fixture.subscriptionID)
	require.NoError(t, err)

	migrationSQL, err := dbmigrations.FS.ReadFile("195_subscription_anchored_monthly_quota.sql")
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, string(migrationSQL))
	require.NoError(t, err)

	var weeklyUsage, monthlyUsage float64
	var weeklyWindowStart, monthlyWindowStart time.Time
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT
    weekly_usage_usd,
    monthly_usage_usd,
    weekly_window_start,
    monthly_window_start
FROM user_subscriptions
WHERE id = $1
`, fixture.subscriptionID).Scan(
		&weeklyUsage,
		&monthlyUsage,
		&weeklyWindowStart,
		&monthlyWindowStart,
	))

	require.Equal(t, 7.0, weeklyUsage,
		"an already-authoritative current weekly counter must be preserved")
	require.Equal(t, 7.0, monthlyUsage,
		"monthly usage must count each subscription charge once and ignore balance-billed logs")
	require.True(t, weeklyWindowStart.Equal(fixture.weeklyStart))
	require.True(t, monthlyWindowStart.Equal(fixture.monthlyStart))
}

func TestMigration195FailsClosedWhenAuthoritativeWeeklyCounterDiffersFromLedger(t *testing.T) {
	tx := testTx(t)
	newMigration195Fixture(t, tx, 10*24*time.Hour)

	err := applyMigration195(t, tx)
	require.ErrorContains(t, err, "authoritative weekly counter")
}

func TestMigration195UsesHalfOpenWeeklyAndMonthlyBoundaries(t *testing.T) {
	tx := testTx(t)
	ctx := context.Background()
	fixture := newMigration195Fixture(t, tx, 40*24*time.Hour)

	testCases := []struct {
		label      string
		occurredAt time.Time
		amount     float64
	}{
		{"monthly-start", fixture.monthlyStart, 1},
		{"weekly-start", fixture.weeklyStart, 2},
		{"weekly-end", fixture.weeklyStart.Add(7 * 24 * time.Hour), 4},
		{"monthly-end", fixture.monthlyStart.Add(30 * 24 * time.Hour), 8},
	}
	for _, testCase := range testCases {
		usageLogID := insertMigration195UsageLog(
			t,
			tx,
			fixture,
			migration195RequestID(testCase.label),
			1,
			testCase.amount,
		)
		_, err := tx.ExecContext(ctx, `
UPDATE usage_logs
SET created_at = $1
WHERE id = $2
`, testCase.occurredAt, usageLogID)
		require.NoError(t, err)
	}

	_, err := tx.ExecContext(ctx, `
UPDATE user_subscriptions
SET weekly_window_start = starts_at,
    weekly_usage_usd = 999
WHERE id = $1
`, fixture.subscriptionID)
	require.NoError(t, err)
	require.NoError(t, applyMigration195(t, tx))

	var weeklyUsage, monthlyUsage float64
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT weekly_usage_usd, monthly_usage_usd
FROM user_subscriptions
WHERE id = $1
`, fixture.subscriptionID).Scan(&weeklyUsage, &monthlyUsage))
	require.Equal(t, 2.0, weeklyUsage)
	require.Equal(t, 7.0, monthlyUsage)
}

func TestMigration195FailsClosedWhenSubscriptionReceiptMatchesOnlyBalanceBilledLog(t *testing.T) {
	tx := testTx(t)
	fixture := newMigration195Fixture(t, tx, 10*24*time.Hour)
	requestID := migration195RequestID("mismatched")
	balanceLogID := insertMigration195UsageLog(t, tx, fixture, requestID, 0, 9)
	insertMigration195LegacyReceipt(t, tx, fixture, balanceLogID, requestID, 9)
	_, err := tx.ExecContext(context.Background(), `
UPDATE usage_logs
SET subscription_id = NULL
WHERE id = $1
`, balanceLogID)
	require.NoError(t, err)
	_, err = tx.ExecContext(context.Background(), `
UPDATE billing_usage_entries
SET subscription_amount = 9
WHERE request_id = $1 AND api_key_id = $2
`, requestID, fixture.apiKeyID)
	require.NoError(t, err)

	migrationSQL, err := dbmigrations.FS.ReadFile("195_subscription_anchored_monthly_quota.sql")
	require.NoError(t, err)
	_, err = tx.ExecContext(context.Background(), string(migrationSQL))
	requireMigration195Blockers(t, err, migration195BlockerCounts{receipts: 1})
}

func TestMigration195FailsClosedWhenSubscriptionLogMatchesBalanceReceiptWithNoSubscription(t *testing.T) {
	tx := testTx(t)
	fixture := newMigration195Fixture(t, tx, 10*24*time.Hour)
	requestID := migration195RequestID("reverse-balance-receipt")
	usageLogID := insertMigration195UsageLog(t, tx, fixture, requestID, 1, 9)
	insertMigration195LegacyReceipt(t, tx, fixture, usageLogID, requestID, 9)

	_, err := tx.ExecContext(context.Background(), `
UPDATE billing_usage_entries
SET subscription_id = NULL,
    billing_type = 0,
    status = 'charged',
    charged_amount = 9
WHERE request_id = $1 AND api_key_id = $2
`, requestID, fixture.apiKeyID)
	require.NoError(t, err)

	err = applyMigration195(t, tx)
	requireMigration195Blockers(t, err, migration195BlockerCounts{subscriptionLogs: 1})
}

func TestMigration195FailsClosedWhenCurrentReceiptMatchesOldTermLogOutsideGuard(t *testing.T) {
	tx := testTx(t)
	fixture := newMigration195Fixture(t, tx, 10*24*time.Hour)
	requestID := migration195RequestID("old-term-outside-guard")
	usageLogID := insertMigration195UsageLog(t, tx, fixture, requestID, 1, 4)
	insertMigration195LegacyReceipt(t, tx, fixture, usageLogID, requestID, 4)

	_, err := tx.ExecContext(context.Background(), `
UPDATE usage_logs
SET created_at = $1
WHERE id = $2
`, fixture.startsAt.Add(-time.Hour), usageLogID)
	require.NoError(t, err)

	err = applyMigration195(t, tx)
	requireMigration195Blockers(t, err, migration195BlockerCounts{receipts: 1})
}

func TestMigration195FailsClosedOnNegativeSubscriptionAmount(t *testing.T) {
	tx := testTx(t)
	fixture := newMigration195Fixture(t, tx, 10*24*time.Hour)
	requestID := migration195RequestID("negative-receipt")
	usageLogID := insertMigration195UsageLog(t, tx, fixture, requestID, 1, 4)
	insertMigration195LegacyReceipt(t, tx, fixture, usageLogID, requestID, 4)

	_, err := tx.ExecContext(context.Background(), `
ALTER TABLE billing_usage_entries
DROP CONSTRAINT billing_usage_entries_subscription_amount_check
`)
	require.NoError(t, err)
	_, err = tx.ExecContext(context.Background(), `
UPDATE billing_usage_entries
SET subscription_amount = -4
WHERE request_id = $1 AND api_key_id = $2
`, requestID, fixture.apiKeyID)
	require.NoError(t, err)

	err = applyMigration195(t, tx)
	require.ErrorContains(t, err, "billing_usage_entries_subscription_amount_check")
}

func TestMigration195FailsClosedOnNaNSubscriptionAmount(t *testing.T) {
	tx := testTx(t)
	fixture := newMigration195Fixture(t, tx, 10*24*time.Hour)
	requestID := migration195RequestID("nan-receipt")
	usageLogID := insertMigration195UsageLog(t, tx, fixture, requestID, 1, 4)
	insertMigration195LegacyReceipt(t, tx, fixture, usageLogID, requestID, 4)

	_, err := tx.ExecContext(context.Background(), `
ALTER TABLE billing_usage_entries
DROP CONSTRAINT billing_usage_entries_subscription_amount_check
`)
	require.NoError(t, err)
	_, err = tx.ExecContext(context.Background(), `
UPDATE billing_usage_entries
SET subscription_amount = 'NaN'::numeric
WHERE usage_log_id = $1
`, usageLogID)
	require.NoError(t, err)

	err = applyMigration195(t, tx)
	require.ErrorContains(t, err, "billing_usage_entries_subscription_amount_check")
}

func TestMigration195FailsClosedOnNegativeReceiptFallbackActualCost(t *testing.T) {
	tx := testTx(t)
	fixture := newMigration195Fixture(t, tx, 10*24*time.Hour)
	requestID := migration195RequestID("negative-fallback")
	usageLogID := insertMigration195UsageLog(t, tx, fixture, requestID, 1, 4)
	insertMigration195LegacyReceipt(t, tx, fixture, usageLogID, requestID, 4)

	_, err := tx.ExecContext(context.Background(), `
UPDATE usage_logs
SET actual_cost = -4
WHERE id = $1
`, usageLogID)
	require.NoError(t, err)

	err = applyMigration195(t, tx)
	requireMigration195Blockers(t, err, migration195BlockerCounts{
		receipts:         1,
		subscriptionLogs: 1,
	})
}

func TestMigration195FailsClosedOnNaNReceiptFallbackActualCost(t *testing.T) {
	tx := testTx(t)
	fixture := newMigration195Fixture(t, tx, 10*24*time.Hour)
	requestID := migration195RequestID("nan-fallback")
	usageLogID := insertMigration195UsageLog(t, tx, fixture, requestID, 1, 4)
	insertMigration195LegacyReceipt(t, tx, fixture, usageLogID, requestID, 4)

	_, err := tx.ExecContext(context.Background(), `
UPDATE usage_logs
SET actual_cost = 'NaN'::numeric
WHERE id = $1
`, usageLogID)
	require.NoError(t, err)

	err = applyMigration195(t, tx)
	requireMigration195Blockers(t, err, migration195BlockerCounts{
		receipts:         1,
		subscriptionLogs: 1,
	})
}

func TestMigration195FailsClosedOnNegativeUnmatchedSubscriptionLog(t *testing.T) {
	tx := testTx(t)
	fixture := newMigration195Fixture(t, tx, 10*24*time.Hour)
	usageLogID := insertMigration195UsageLog(
		t,
		tx,
		fixture,
		migration195RequestID("negative-legacy"),
		1,
		4,
	)

	_, err := tx.ExecContext(context.Background(), `
UPDATE usage_logs
SET actual_cost = -4
WHERE id = $1
`, usageLogID)
	require.NoError(t, err)

	err = applyMigration195(t, tx)
	requireMigration195Blockers(t, err, migration195BlockerCounts{subscriptionLogs: 1})
}

func TestMigration195FailsClosedOnSubscriptionReceiptBillingTypeMismatch(t *testing.T) {
	tx := testTx(t)
	fixture := newMigration195Fixture(t, tx, 10*24*time.Hour)
	requestID := migration195RequestID("receipt-type")
	usageLogID := insertMigration195UsageLog(t, tx, fixture, requestID, 1, 4)
	insertMigration195LegacyReceipt(t, tx, fixture, usageLogID, requestID, 4)

	_, err := tx.ExecContext(context.Background(), `
UPDATE billing_usage_entries
SET billing_type = 0
WHERE request_id = $1 AND api_key_id = $2
`, requestID, fixture.apiKeyID)
	require.NoError(t, err)

	err = applyMigration195(t, tx)
	requireMigration195Blockers(t, err, migration195BlockerCounts{
		receipts:         1,
		subscriptionLogs: 1,
	})
}

func TestMigration195FailsClosedWhenReceiptMatchesMultipleUsageLogs(t *testing.T) {
	tx := testTx(t)
	fixture := newMigration195Fixture(t, tx, 10*24*time.Hour)
	usageLogID := insertMigration195UsageLog(
		t,
		tx,
		fixture,
		migration195RequestID("candidate-by-id"),
		1,
		4,
	)
	requestMatchedLogID := migration195RequestID("candidate-by-request")
	insertMigration195UsageLog(t, tx, fixture, requestMatchedLogID, 1, 4)
	insertMigration195LegacyReceipt(t, tx, fixture, usageLogID, requestMatchedLogID, 4)

	err := applyMigration195(t, tx)
	requireMigration195Blockers(t, err, migration195BlockerCounts{
		receipts:         1,
		subscriptionLogs: 1,
	})
}

func TestMigration195FailsClosedWhenDirectReceiptRequestIDConflicts(t *testing.T) {
	tx := testTx(t)
	fixture := newMigration195Fixture(t, tx, 10*24*time.Hour)
	usageLogID := insertMigration195UsageLog(
		t,
		tx,
		fixture,
		migration195RequestID("direct-log-request"),
		1,
		4,
	)
	insertMigration195LegacyReceipt(
		t,
		tx,
		fixture,
		usageLogID,
		migration195RequestID("direct-receipt-request"),
		4,
	)

	err := applyMigration195(t, tx)
	requireMigration195Blockers(t, err, migration195BlockerCounts{
		receipts:         1,
		subscriptionLogs: 1,
	})
}

func TestMigration195FailsClosedOnNaNUsageInsideMonthlyButOutsideWeeklyWindow(t *testing.T) {
	tx := testTx(t)
	fixture := newMigration195Fixture(t, tx, 10*24*time.Hour)
	usageLogID := insertMigration195UsageLog(
		t,
		tx,
		fixture,
		migration195RequestID("nan-monthly-only"),
		1,
		4,
	)

	_, err := tx.ExecContext(context.Background(), `
UPDATE usage_logs
SET actual_cost = 'NaN'::numeric,
    created_at = $1
WHERE id = $2
`, fixture.weeklyStart.Add(-time.Hour), usageLogID)
	require.NoError(t, err)

	err = applyMigration195(t, tx)
	requireMigration195Blockers(t, err, migration195BlockerCounts{subscriptionLogs: 1})
}

func TestMigration195FailsClosedOnNaNExistingSubscriptionCounter(t *testing.T) {
	tx := testTx(t)
	fixture := newMigration195Fixture(t, tx, 10*24*time.Hour)

	_, err := tx.ExecContext(context.Background(), `
ALTER TABLE user_subscriptions
DROP CONSTRAINT user_subscriptions_usage_amounts_check
`)
	require.NoError(t, err)
	_, err = tx.ExecContext(context.Background(), `
UPDATE user_subscriptions
SET monthly_usage_usd = 'NaN'::numeric
WHERE id = $1
`, fixture.subscriptionID)
	require.NoError(t, err)

	err = applyMigration195(t, tx)
	require.ErrorContains(t, err, "user_subscriptions_usage_amounts_check")
}

func TestMigration195FailsClosedOnInvalidSubscriptionTimeRange(t *testing.T) {
	tx := testTx(t)
	fixture := newMigration195Fixture(t, tx, 10*24*time.Hour)

	_, err := tx.ExecContext(context.Background(), `
UPDATE user_subscriptions
SET expires_at = starts_at
WHERE id = $1
`, fixture.subscriptionID)
	require.NoError(t, err)

	err = applyMigration195(t, tx)
	requireMigration195Blockers(t, err, migration195BlockerCounts{times: 1})
}

func TestMigration195FailsClosedOnNaNActivePlanLimit(t *testing.T) {
	tx := testTx(t)
	fixture := newMigration195Fixture(t, tx, 10*24*time.Hour)

	_, err := tx.ExecContext(context.Background(), `
ALTER TABLE groups
DROP CONSTRAINT groups_subscription_quota_limits_check
`)
	require.NoError(t, err)
	_, err = tx.ExecContext(context.Background(), `
UPDATE groups
SET weekly_limit_usd = 'NaN'::numeric
WHERE id = $1
`, fixture.groupID)
	require.NoError(t, err)

	err = applyMigration195(t, tx)
	require.ErrorContains(t, err, "groups_subscription_quota_limits_check")
}

func TestMigration195FailsClosedWhenActiveSubscriptionHasNoWeeklyLimit(t *testing.T) {
	tx := testTx(t)
	fixture := newMigration195Fixture(t, tx, 10*24*time.Hour)

	_, err := tx.ExecContext(context.Background(), `
UPDATE groups
SET weekly_limit_usd = NULL
WHERE id = $1
`, fixture.groupID)
	require.NoError(t, err)

	err = applyMigration195(t, tx)
	requireMigration195Blockers(t, err, migration195BlockerCounts{activePlans: 1})
}

func TestMigration195KeepsNegativeLimitCompatibilityForStandardGroups(t *testing.T) {
	tx := testTx(t)
	var groupID int64
	require.NoError(t, tx.QueryRowContext(context.Background(), `
INSERT INTO groups (
    name,
    platform,
    subscription_type,
    status,
    weekly_limit_usd,
    monthly_limit_usd
)
VALUES ($1, 'openai', 'standard', 'active', -1, -1)
RETURNING id
`, "migration-195-standard-"+uuid.NewString()).Scan(&groupID))

	require.NoError(t, applyMigration195(t, tx))
	var weeklyLimit, monthlyLimit float64
	require.NoError(t, tx.QueryRowContext(context.Background(), `
SELECT weekly_limit_usd, monthly_limit_usd
FROM groups
WHERE id = $1
`, groupID).Scan(&weeklyLimit, &monthlyLimit))
	require.Equal(t, -1.0, weeklyLimit)
	require.Equal(t, -1.0, monthlyLimit)
}

func TestMigration195FailsClosedOnCrossOwnerDirectReceiptLinks(t *testing.T) {
	testCases := []struct {
		name          string
		updateReceipt string
	}{
		{
			name:          "cross user",
			updateReceipt: "SET user_id = $1, subscription_amount = 4",
		},
		{
			name:          "cross API key",
			updateReceipt: "SET api_key_id = $1, subscription_amount = 4",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			tx := testTx(t)
			fixture := newMigration195Fixture(t, tx, 10*24*time.Hour)
			other := newMigration195Fixture(t, tx, 10*24*time.Hour)
			requestID := migration195RequestID("cross-owner-direct")
			usageLogID := insertMigration195UsageLog(t, tx, fixture, requestID, 1, 4)
			insertMigration195LegacyReceipt(t, tx, fixture, usageLogID, requestID, 4)

			foreignIdentityID := other.userID
			if testCase.name == "cross API key" {
				foreignIdentityID = other.apiKeyID
			}
			_, err := tx.ExecContext(
				context.Background(),
				"UPDATE billing_usage_entries "+testCase.updateReceipt+" WHERE usage_log_id = $2",
				foreignIdentityID,
				usageLogID,
			)
			require.NoError(t, err)

			err = applyMigration195(t, tx)
			requireMigration195Blockers(t, err, migration195BlockerCounts{
				receipts:         1,
				subscriptionLogs: 1,
			})
		})
	}
}
