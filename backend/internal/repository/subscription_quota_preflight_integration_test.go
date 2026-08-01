//go:build integration

package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	dbmigrations "github.com/Wei-Shaw/sub2api/migrations"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

type migration195PreflightPayload struct {
	Contract         string            `json:"contract"`
	MigrationSHA256  string            `json:"migration_sha256"`
	Status           string            `json:"status"`
	AlreadyApplied   bool              `json:"already_applied"`
	SchemaOK         bool              `json:"schema_ok"`
	ReadOnlyVerified bool              `json:"read_only_verified"`
	Counts           map[string]string `json:"counts"`
	Blockers         []string          `json:"blockers"`
}

type migration195PreflightReceiptScenario int

const (
	migration195PreflightReverseBalanceReceipt migration195PreflightReceiptScenario = iota
	migration195PreflightForwardBalanceLog
	migration195PreflightOldTermLog
	migration195PreflightNegativeLogReceiptOutsideWindow
	migration195PreflightCrossUserDirectLink
	migration195PreflightCrossAPIKeyDirectLink
	migration195PreflightDirectRequestIDConflict
)

func TestMigration195PreflightCoversPreMigrationBlockerAndAppliedChecksum(t *testing.T) {
	ctx := context.Background()
	db := newMigration195PreMigrationDatabase(t)
	expectedChecksum := migration195ExpectedChecksum(t)

	clean := runMigration195Preflight(t, db, expectedChecksum)
	require.Equal(t, "migration-195-preflight/v1", clean.Contract)
	require.Equal(t, expectedChecksum, clean.MigrationSHA256)
	require.True(t, clean.SchemaOK)
	require.True(t, clean.ReadOnlyVerified)
	require.False(t, clean.AlreadyApplied)
	require.Equal(t, "0", clean.Counts["database_prepared_transactions"])
	require.Empty(t, clean.Blockers)

	limitedObserverRole := newMigration195PreflightReadRole(t, db)
	limitedObserver := runMigration195PreflightAsRole(
		t,
		db,
		expectedChecksum,
		limitedObserverRole,
	)
	require.Contains(t, limitedObserver.Blockers, "database_writer_visibility")

	fixture := insertUnresolvedMigration195PreflightReceipt(t, db)
	blocked := runMigration195Preflight(t, db, expectedChecksum)
	require.Equal(t, "blocked", blocked.Status)
	require.Equal(t, "1", blocked.Counts["unresolved_receipts"])
	require.Contains(t, blocked.Blockers, "receipt_reconstructability")

	deleteMigration195PreflightFixture(t, db, fixture)
	err := ApplyMigrations(ctx, db)
	require.ErrorContains(t, err, "maintenance-only migration")

	expectedIdentity, err := queryDatabaseIdentity(ctx, db)
	require.NoError(t, err)
	require.NoError(t, applyMigrationsFSWithExpectedDatabaseIdentity(
		ctx,
		db,
		dbmigrations.FS,
		&expectedIdentity,
	))

	applied := runMigration195Preflight(t, db, expectedChecksum)
	require.True(t, applied.SchemaOK)
	require.True(t, applied.ReadOnlyVerified)
	require.True(t, applied.AlreadyApplied)
	require.Empty(t, applied.Blockers)
}

func TestMigration195PreflightBlocksReverseReceiptIdentityAndDefiniteOldTerm(t *testing.T) {
	db := newMigration195PreMigrationDatabase(t)
	expectedChecksum := migration195ExpectedChecksum(t)

	t.Run("membership log linked to balance receipt without subscription", func(t *testing.T) {
		fixture := insertMigration195PreflightLinkedReceipt(
			t,
			db,
			migration195PreflightReverseBalanceReceipt,
		)
		t.Cleanup(func() { deleteMigration195PreflightFixture(t, db, fixture) })

		payload := runMigration195Preflight(t, db, expectedChecksum)
		require.Equal(t, "blocked", payload.Status)
		require.Equal(t, "1", payload.Counts["usage_log_receipt_identity_conflicts"])
		require.Contains(t, payload.Blockers, "receipt_billing_identity")
	})

	t.Run("subscription receipt linked to balance log without subscription", func(t *testing.T) {
		fixture := insertMigration195PreflightLinkedReceipt(
			t,
			db,
			migration195PreflightForwardBalanceLog,
		)
		t.Cleanup(func() { deleteMigration195PreflightFixture(t, db, fixture) })

		payload := runMigration195Preflight(t, db, expectedChecksum)
		require.Equal(t, "blocked", payload.Status)
		require.Equal(t, "1", payload.Counts["ambiguous_or_mixed_receipt_matches"])
		require.Contains(t, payload.Blockers, "receipt_billing_identity")
	})

	t.Run("current receipt linked to old term outside guard", func(t *testing.T) {
		fixture := insertMigration195PreflightLinkedReceipt(
			t,
			db,
			migration195PreflightOldTermLog,
		)
		t.Cleanup(func() { deleteMigration195PreflightFixture(t, db, fixture) })

		payload := runMigration195Preflight(t, db, expectedChecksum)
		require.Equal(t, "blocked", payload.Status)
		require.Equal(t, "1", payload.Counts["definite_old_term_receipts"])
		require.Contains(t, payload.Blockers, "definite_old_term_events")
	})

	t.Run("negative membership log linked to receipt outside current windows", func(t *testing.T) {
		fixture := insertMigration195PreflightLinkedReceipt(
			t,
			db,
			migration195PreflightNegativeLogReceiptOutsideWindow,
		)
		t.Cleanup(func() { deleteMigration195PreflightFixture(t, db, fixture) })

		payload := runMigration195Preflight(t, db, expectedChecksum)
		require.Equal(t, "blocked", payload.Status)
		require.Equal(t, "1", payload.Counts["negative_usage_amounts"])
		require.Contains(t, payload.Blockers, "nonnegative_usage_amounts")
	})

	t.Run("direct receipt and usage log request IDs conflict", func(t *testing.T) {
		fixture := insertMigration195PreflightLinkedReceipt(
			t,
			db,
			migration195PreflightDirectRequestIDConflict,
		)
		t.Cleanup(func() { deleteMigration195PreflightFixture(t, db, fixture) })

		payload := runMigration195Preflight(t, db, expectedChecksum)
		require.Equal(t, "blocked", payload.Status)
		require.Equal(t, "1", payload.Counts["ambiguous_or_mixed_receipt_matches"])
		require.Contains(t, payload.Blockers, "receipt_billing_identity")
	})

	for _, testCase := range []struct {
		name     string
		scenario migration195PreflightReceiptScenario
	}{
		{"direct link crosses receipt user", migration195PreflightCrossUserDirectLink},
		{"direct link crosses receipt API key", migration195PreflightCrossAPIKeyDirectLink},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := insertMigration195PreflightLinkedReceipt(t, db, testCase.scenario)
			t.Cleanup(func() { deleteMigration195PreflightFixture(t, db, fixture) })

			payload := runMigration195Preflight(t, db, expectedChecksum)
			require.Equal(t, "blocked", payload.Status)
			require.Equal(t, "1", payload.Counts["receipt_identity_conflicts"])
			require.Contains(t, payload.Blockers, "receipt_billing_identity")
		})
	}
}

func TestMigration195PreflightAndMigrationBlockNaNUsageInsideMonthlyOnly(t *testing.T) {
	db := newMigration195PreMigrationDatabase(t)
	fixture := insertMigration195PreflightNaNMonthlyOnlyLog(t, db)
	t.Cleanup(func() { deleteMigration195PreflightFixture(t, db, fixture) })

	payload := runMigration195Preflight(t, db, migration195ExpectedChecksum(t))
	require.Equal(t, "blocked", payload.Status)
	require.Equal(t, "1", payload.Counts["negative_usage_amounts"])
	require.Contains(t, payload.Blockers, "nonnegative_usage_amounts")

	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	err = applyMigration195(t, tx)
	require.ErrorContains(t, err, "invalid current-window subscription usage log")
	require.NoError(t, tx.Rollback())
}

func TestMigration195PreflightAndMigrationMirrorCoreSafetyBlockers(t *testing.T) {
	db := newMigration195PreMigrationDatabase(t)
	expectedChecksum := migration195ExpectedChecksum(t)

	t.Run("authoritative weekly ledger mismatch", func(t *testing.T) {
		tx, err := db.BeginTx(context.Background(), nil)
		require.NoError(t, err)
		fixture := newMigration195Fixture(t, tx, 10*24*time.Hour)
		require.NoError(t, tx.Commit())

		payload := runMigration195Preflight(t, db, expectedChecksum)
		require.Equal(t, "1", payload.Counts["authoritative_weekly_ledger_mismatches"])
		require.Contains(t, payload.Blockers, "authoritative_weekly_ledger_consistency")

		migrationTx, err := db.BeginTx(context.Background(), nil)
		require.NoError(t, err)
		err = applyMigration195(t, migrationTx)
		require.ErrorContains(t, err, "authoritative weekly counter")
		require.NoError(t, migrationTx.Rollback())
		deleteMigration195PreflightFixture(t, db, fixture)
	})

	t.Run("invalid subscription time range", func(t *testing.T) {
		tx, err := db.BeginTx(context.Background(), nil)
		require.NoError(t, err)
		fixture := newMigration195Fixture(t, tx, 10*24*time.Hour)
		_, err = tx.ExecContext(context.Background(), `
UPDATE user_subscriptions
SET expires_at = starts_at,
    weekly_usage_usd = 0
WHERE id = $1
`, fixture.subscriptionID)
		require.NoError(t, err)
		require.NoError(t, tx.Commit())

		payload := runMigration195Preflight(t, db, expectedChecksum)
		require.Equal(t, "1", payload.Counts["invalid_subscription_times"])
		require.Contains(t, payload.Blockers, "subscription_time_invariants")

		migrationTx, err := db.BeginTx(context.Background(), nil)
		require.NoError(t, err)
		err = applyMigration195(t, migrationTx)
		require.ErrorContains(t, err, "invalid subscription time range")
		require.NoError(t, migrationTx.Rollback())
		deleteMigration195PreflightFixture(t, db, fixture)
	})
}

func TestMigration195PreflightBlocksNaNCountersAndPlanLimits(t *testing.T) {
	db := newMigration195PreMigrationDatabase(t)
	expectedChecksum := migration195ExpectedChecksum(t)

	t.Run("persisted subscription counter", func(t *testing.T) {
		tx, err := db.BeginTx(context.Background(), nil)
		require.NoError(t, err)
		fixture := newMigration195Fixture(t, tx, 10*24*time.Hour)
		_, err = tx.ExecContext(context.Background(), `
UPDATE user_subscriptions
SET monthly_usage_usd = 'NaN'::numeric
WHERE id = $1
`, fixture.subscriptionID)
		require.NoError(t, err)
		require.NoError(t, tx.Commit())
		t.Cleanup(func() { deleteMigration195PreflightFixture(t, db, fixture) })

		payload := runMigration195Preflight(t, db, expectedChecksum)
		require.Equal(t, "1", payload.Counts["invalid_subscription_counters"])
		require.Contains(t, payload.Blockers, "subscription_counter_values")
	})

	t.Run("subscription plan limit", func(t *testing.T) {
		tx, err := db.BeginTx(context.Background(), nil)
		require.NoError(t, err)
		fixture := newMigration195Fixture(t, tx, 10*24*time.Hour)
		_, err = tx.ExecContext(context.Background(), `
UPDATE groups
SET monthly_limit_usd = 'NaN'::numeric
WHERE id = $1
`, fixture.groupID)
		require.NoError(t, err)
		require.NoError(t, tx.Commit())
		t.Cleanup(func() { deleteMigration195PreflightFixture(t, db, fixture) })

		payload := runMigration195Preflight(t, db, expectedChecksum)
		require.Equal(t, "1", payload.Counts["invalid_subscription_plan_limit_values"])
		require.Contains(t, payload.Blockers, "subscription_plan_limit_values")
	})
}

func TestMigration195PreflightBlocksPreparedTransactionsWhenEnabled(t *testing.T) {
	db := newMigration195PreMigrationDatabase(t)

	var maxPreparedTransactions int
	require.NoError(t, db.QueryRowContext(
		context.Background(),
		"SELECT current_setting('max_prepared_transactions')::integer",
	).Scan(&maxPreparedTransactions))
	if maxPreparedTransactions == 0 {
		t.Skip("PostgreSQL max_prepared_transactions=0; static SQL coverage enforces this gate")
	}

	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	fixture := newMigration195Fixture(t, tx, 10*24*time.Hour)
	_, err = tx.ExecContext(context.Background(), `
UPDATE user_subscriptions
SET weekly_usage_usd = 0
WHERE id = $1
`, fixture.subscriptionID)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	gid := "migration195_preflight_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	prepared := false
	defer func() {
		if prepared {
			_, rollbackErr := db.ExecContext(
				context.Background(),
				"ROLLBACK PREPARED "+pq.QuoteLiteral(gid),
			)
			require.NoError(t, rollbackErr)
		}
		deleteMigration195PreflightFixture(t, db, fixture)
	}()

	conn, err := db.Conn(context.Background())
	require.NoError(t, err)
	defer conn.Close()
	_, err = conn.ExecContext(context.Background(), "BEGIN")
	require.NoError(t, err)
	_, err = conn.ExecContext(context.Background(), `
UPDATE user_subscriptions
SET updated_at = NOW()
WHERE id = $1
`, fixture.subscriptionID)
	require.NoError(t, err)
	_, err = conn.ExecContext(
		context.Background(),
		"PREPARE TRANSACTION "+pq.QuoteLiteral(gid),
	)
	require.NoError(t, err)
	prepared = true

	payload := runMigration195Preflight(t, db, migration195ExpectedChecksum(t))
	require.Equal(t, "1", payload.Counts["database_prepared_transactions"])
	require.Contains(t, payload.Blockers, "database_prepared_transactions")
}

func newMigration195PreMigrationDatabase(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()
	databaseName := "migration195_preflight_" + strings.ReplaceAll(uuid.NewString(), "-", "")

	_, err := integrationDB.ExecContext(
		ctx,
		"CREATE DATABASE "+pq.QuoteIdentifier(databaseName),
	)
	require.NoError(t, err)

	parsedDSN, err := url.Parse(integrationDSN)
	require.NoError(t, err)
	parsedDSN.Path = "/" + databaseName
	db, err := openSQLWithRetry(ctx, parsedDSN.String(), 30*time.Second)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = db.Close()
		_, dropErr := integrationDB.ExecContext(
			context.Background(),
			"DROP DATABASE "+pq.QuoteIdentifier(databaseName)+" WITH (FORCE)",
		)
		require.NoError(t, dropErr)
	})

	require.NoError(t, applyMigrationsFS(ctx, db, migration195PreMigrationFS(t)))
	return db
}

func migration195PreMigrationFS(t *testing.T) fs.FS {
	t.Helper()
	entries, err := fs.ReadDir(dbmigrations.FS, ".")
	require.NoError(t, err)

	files := fstest.MapFS{}
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "195_subscription_anchored_monthly_quota.sql" {
			continue
		}
		content, readErr := fs.ReadFile(dbmigrations.FS, entry.Name())
		require.NoError(t, readErr)
		files[entry.Name()] = &fstest.MapFile{Data: content, Mode: 0o444}
	}
	return files
}

func migration195ExpectedChecksum(t *testing.T) string {
	t.Helper()
	content, err := dbmigrations.FS.ReadFile("195_subscription_anchored_monthly_quota.sql")
	require.NoError(t, err)
	digest := sha256.Sum256([]byte(strings.TrimSpace(string(content))))
	return hex.EncodeToString(digest[:])
}

func runMigration195Preflight(
	t *testing.T,
	db *sql.DB,
	expectedChecksum string,
) migration195PreflightPayload {
	t.Helper()
	return runMigration195PreflightAsRole(t, db, expectedChecksum, "")
}

func runMigration195PreflightAsRole(
	t *testing.T,
	db *sql.DB,
	expectedChecksum string,
	role string,
) migration195PreflightPayload {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(
		"..",
		"..",
		"scripts",
		"sql",
		"preflight_subscription_anchored_monthly_quota.sql",
	))
	require.NoError(t, err)

	query := string(content)
	replacements := map[string]string{
		":'preflight_mode'":                    "'maintenance'",
		":'legacy_writers_stopped_at'":         "''",
		":'term_guard_hours'":                  "'2'",
		":'acknowledge_ambiguous_term_events'": "'false'",
		":'expected_migration_checksum'":       pq.QuoteLiteral(expectedChecksum),
	}
	for placeholder, value := range replacements {
		query = strings.ReplaceAll(query, placeholder, value)
	}

	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	})
	require.NoError(t, err)
	defer tx.Rollback()
	if role != "" {
		_, err = tx.ExecContext(
			context.Background(),
			"SET LOCAL ROLE "+pq.QuoteIdentifier(role),
		)
		require.NoError(t, err)
	}

	var rawPayload string
	require.NoError(t, tx.QueryRowContext(context.Background(), query).Scan(&rawPayload))
	require.NoError(t, tx.Commit())

	var payload migration195PreflightPayload
	require.NoError(t, json.Unmarshal([]byte(rawPayload), &payload))
	return payload
}

func newMigration195PreflightReadRole(t *testing.T, db *sql.DB) string {
	t.Helper()
	role := "migration195_reader_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err := db.ExecContext(
		context.Background(),
		"CREATE ROLE "+pq.QuoteIdentifier(role)+" NOLOGIN",
	)
	require.NoError(t, err)
	_, err = db.ExecContext(
		context.Background(),
		"GRANT USAGE ON SCHEMA public TO "+pq.QuoteIdentifier(role),
	)
	require.NoError(t, err)
	_, err = db.ExecContext(
		context.Background(),
		"GRANT SELECT ON groups, user_subscriptions, billing_usage_entries, usage_logs, schema_migrations TO "+pq.QuoteIdentifier(role),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, revokeErr := db.ExecContext(
			context.Background(),
			"REVOKE ALL PRIVILEGES ON groups, user_subscriptions, billing_usage_entries, usage_logs, schema_migrations FROM "+pq.QuoteIdentifier(role),
		)
		require.NoError(t, revokeErr)
		_, revokeErr = db.ExecContext(
			context.Background(),
			"REVOKE USAGE ON SCHEMA public FROM "+pq.QuoteIdentifier(role),
		)
		require.NoError(t, revokeErr)
		_, dropErr := db.ExecContext(
			context.Background(),
			"DROP ROLE "+pq.QuoteIdentifier(role),
		)
		require.NoError(t, dropErr)
	})
	return role
}

func insertUnresolvedMigration195PreflightReceipt(
	t *testing.T,
	db *sql.DB,
) migration195Fixture {
	t.Helper()
	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	defer tx.Rollback()

	fixture := newMigration195Fixture(t, tx, 10*24*time.Hour)
	requestID := migration195RequestID("preflight-unresolved")
	_, err = tx.ExecContext(context.Background(), `
INSERT INTO billing_usage_entries (
    usage_log_id,
    user_id,
    api_key_id,
    subscription_id,
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
    NULL, $1, $2, $3, 1, TRUE, -4, $4, $5, 'api', $6,
    'migration-195-model', 'migration-195-model', 4, 0,
    'subscription', FALSE, $7
)
`,
		fixture.userID,
		fixture.apiKeyID,
		fixture.subscriptionID,
		requestID,
		strings.Repeat("b", 64),
		fixture.accountID,
		fixture.occurredAt,
	)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())
	return fixture
}

func insertMigration195PreflightLinkedReceipt(
	t *testing.T,
	db *sql.DB,
	scenario migration195PreflightReceiptScenario,
) migration195Fixture {
	t.Helper()
	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	defer tx.Rollback()

	fixture := newMigration195Fixture(t, tx, 10*24*time.Hour)
	if scenario == migration195PreflightCrossUserDirectLink ||
		scenario == migration195PreflightCrossAPIKeyDirectLink {
		suffix := uuid.NewString()
		require.NoError(t, tx.QueryRowContext(context.Background(), `
INSERT INTO users (email, password_hash, role, status, balance, concurrency)
VALUES ($1, 'migration-195-test', 'user', 'active', 0, 5)
RETURNING id
`, "migration-195-other-"+suffix+"@example.com").Scan(&fixture.otherUserID))
		require.NoError(t, tx.QueryRowContext(context.Background(), `
INSERT INTO api_keys (user_id, key, name, group_id, status)
VALUES ($1, $2, 'migration-195-other', $3, 'active')
RETURNING id
`,
			fixture.otherUserID,
			"sk-migration-195-other-"+suffix,
			fixture.groupID,
		).Scan(&fixture.otherAPIKeyID))
	}
	requestID := migration195RequestID("preflight-linked")
	receiptRequestID := requestID
	if scenario == migration195PreflightDirectRequestIDConflict {
		receiptRequestID = migration195RequestID("preflight-receipt-conflict")
	}
	logBillingType := 1
	if scenario == migration195PreflightForwardBalanceLog {
		logBillingType = 0
	}
	usageLogID := insertMigration195UsageLog(t, tx, fixture, requestID, logBillingType, 4)
	if scenario == migration195PreflightForwardBalanceLog {
		_, err = tx.ExecContext(context.Background(), `
UPDATE usage_logs
SET subscription_id = NULL
WHERE id = $1
`, usageLogID)
		require.NoError(t, err)
	}
	if scenario == migration195PreflightOldTermLog {
		_, err = tx.ExecContext(context.Background(), `
UPDATE usage_logs
SET created_at = $1
WHERE id = $2
`, fixture.startsAt.Add(-time.Hour), usageLogID)
		require.NoError(t, err)
	}
	if scenario == migration195PreflightNegativeLogReceiptOutsideWindow {
		_, err = tx.ExecContext(context.Background(), `
UPDATE usage_logs
SET actual_cost = -4
WHERE id = $1
`, usageLogID)
		require.NoError(t, err)
	}

	var subscriptionID any = fixture.subscriptionID
	receiptUserID := fixture.userID
	receiptAPIKeyID := fixture.apiKeyID
	billingType := 1
	status := "subscription"
	chargedAmount := 0
	if scenario == migration195PreflightReverseBalanceReceipt {
		subscriptionID = nil
		billingType = 0
		status = "charged"
		chargedAmount = 4
	}
	if scenario == migration195PreflightCrossUserDirectLink {
		receiptUserID = fixture.otherUserID
	}
	if scenario == migration195PreflightCrossAPIKeyDirectLink {
		receiptAPIKeyID = fixture.otherAPIKeyID
	}
	_, err = tx.ExecContext(context.Background(), `
INSERT INTO billing_usage_entries (
    usage_log_id,
    user_id,
    api_key_id,
    subscription_id,
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
    $1, $2, $3, $4, $5, TRUE, -4, $6, $7, 'api', $8,
    'migration-195-model', 'migration-195-model', 4, $9, $10, FALSE, $11
)
`,
		usageLogID,
		receiptUserID,
		receiptAPIKeyID,
		subscriptionID,
		billingType,
		receiptRequestID,
		strings.Repeat("c", 64),
		fixture.accountID,
		chargedAmount,
		status,
		fixture.occurredAt,
	)
	require.NoError(t, err)
	if scenario == migration195PreflightNegativeLogReceiptOutsideWindow {
		_, err = tx.ExecContext(context.Background(), `
UPDATE billing_usage_entries
SET created_at = $1
WHERE usage_log_id = $2
`, fixture.startsAt.Add(-time.Hour), usageLogID)
		require.NoError(t, err)
	}
	require.NoError(t, tx.Commit())
	return fixture
}

func insertMigration195PreflightNaNMonthlyOnlyLog(
	t *testing.T,
	db *sql.DB,
) migration195Fixture {
	t.Helper()
	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	defer tx.Rollback()

	fixture := newMigration195Fixture(t, tx, 10*24*time.Hour)
	usageLogID := insertMigration195UsageLog(
		t,
		tx,
		fixture,
		migration195RequestID("preflight-nan-monthly-only"),
		1,
		4,
	)
	_, err = tx.ExecContext(context.Background(), `
UPDATE usage_logs
SET actual_cost = 'NaN'::numeric,
    created_at = $1
WHERE id = $2
`, fixture.weeklyStart.Add(-time.Hour), usageLogID)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())
	return fixture
}

func deleteMigration195PreflightFixture(
	t *testing.T,
	db *sql.DB,
	fixture migration195Fixture,
) {
	t.Helper()
	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	defer tx.Rollback()

	_, err = tx.ExecContext(context.Background(), `
DELETE FROM billing_usage_entries
WHERE subscription_id = $1
   OR user_id = $2
   OR ($3 <> 0 AND user_id = $3)
   OR usage_log_id IN (
       SELECT id
       FROM usage_logs
       WHERE subscription_id = $1
          OR user_id = $2
          OR ($3 <> 0 AND user_id = $3)
   )
`, fixture.subscriptionID, fixture.userID, fixture.otherUserID)
	require.NoError(t, err)
	_, err = tx.ExecContext(context.Background(), `
DELETE FROM usage_logs
WHERE subscription_id = $1
   OR user_id = $2
   OR ($3 <> 0 AND user_id = $3)
`, fixture.subscriptionID, fixture.userID, fixture.otherUserID)
	require.NoError(t, err)

	statements := []struct {
		query string
		id    int64
	}{
		{"DELETE FROM user_subscriptions WHERE id = $1", fixture.subscriptionID},
		{"DELETE FROM api_keys WHERE id = $1", fixture.apiKeyID},
		{"DELETE FROM users WHERE id = $1", fixture.userID},
		{"DELETE FROM accounts WHERE id = $1", fixture.accountID},
	}
	for _, statement := range statements {
		_, execErr := tx.ExecContext(context.Background(), statement.query, statement.id)
		require.NoError(t, execErr, fmt.Sprintf("cleanup query failed: %s", statement.query))
	}
	if fixture.otherAPIKeyID != 0 {
		_, err = tx.ExecContext(
			context.Background(),
			"DELETE FROM api_keys WHERE id = $1",
			fixture.otherAPIKeyID,
		)
		require.NoError(t, err)
	}
	if fixture.otherUserID != 0 {
		_, err = tx.ExecContext(
			context.Background(),
			"DELETE FROM users WHERE id = $1",
			fixture.otherUserID,
		)
		require.NoError(t, err)
	}
	_, err = tx.ExecContext(
		context.Background(),
		"DELETE FROM groups WHERE id = $1",
		fixture.groupID,
	)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())
}
