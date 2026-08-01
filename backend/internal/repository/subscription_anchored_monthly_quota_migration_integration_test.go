//go:build integration

package repository

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	dbmigrations "github.com/Wei-Shaw/sub2api/migrations"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type migration195Fixture struct {
	userID         int64
	groupID        int64
	apiKeyID       int64
	accountID      int64
	subscriptionID int64
	startsAt       time.Time
	weeklyStart    time.Time
	monthlyStart   time.Time
	occurredAt     time.Time
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
INSERT INTO groups (name, platform, subscription_type, status)
VALUES ($1, 'openai', 'subscription', 'active')
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

func TestMigration195RebuildsMonthlyUsageWithoutBalanceLogLeakageOrReceiptDoubleCount(t *testing.T) {
	tx := testTx(t)
	ctx := context.Background()
	fixture := newMigration195Fixture(t, tx, 40*24*time.Hour)

	insertMigration195UsageLog(t, tx, fixture, migration195RequestID("legacy-sub"), 1, 3)
	insertMigration195UsageLog(t, tx, fixture, migration195RequestID("balance"), 0, 90)
	receiptRequestID := migration195RequestID("receipt")
	receiptLogID := insertMigration195UsageLog(t, tx, fixture, receiptRequestID, 1, 4)
	insertMigration195LegacyReceipt(t, tx, fixture, receiptLogID, receiptRequestID, 4)

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

	require.Equal(t, 17.0, weeklyUsage,
		"an already-authoritative current weekly counter must be preserved")
	require.Equal(t, 7.0, monthlyUsage,
		"monthly usage must count each subscription charge once and ignore balance-billed logs")
	require.True(t, weeklyWindowStart.Equal(fixture.weeklyStart))
	require.True(t, monthlyWindowStart.Equal(fixture.monthlyStart))
}

func TestMigration195FailsClosedWhenSubscriptionReceiptMatchesOnlyBalanceBilledLog(t *testing.T) {
	tx := testTx(t)
	fixture := newMigration195Fixture(t, tx, 10*24*time.Hour)
	requestID := migration195RequestID("mismatched")
	balanceLogID := insertMigration195UsageLog(t, tx, fixture, requestID, 0, 9)
	insertMigration195LegacyReceipt(t, tx, fixture, balanceLogID, requestID, 9)

	migrationSQL, err := dbmigrations.FS.ReadFile("195_subscription_anchored_monthly_quota.sql")
	require.NoError(t, err)
	_, err = tx.ExecContext(context.Background(), string(migrationSQL))
	require.ErrorContains(t, err, "cannot rebuild anchored subscription quota")
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
	require.ErrorContains(t, err, "cannot rebuild anchored subscription quota")
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
	require.ErrorContains(t, err, "invalid unmatched subscription usage log")
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
	require.ErrorContains(t, err, "cannot rebuild anchored subscription quota")
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
	require.ErrorContains(t, err, "cannot rebuild anchored subscription quota")
}
