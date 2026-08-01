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
	require.NoError(t, ApplyMigrations(ctx, db))

	applied := runMigration195Preflight(t, db, expectedChecksum)
	require.True(t, applied.SchemaOK)
	require.True(t, applied.ReadOnlyVerified)
	require.True(t, applied.AlreadyApplied)
	require.Empty(t, applied.Blockers)
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

func deleteMigration195PreflightFixture(
	t *testing.T,
	db *sql.DB,
	fixture migration195Fixture,
) {
	t.Helper()
	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	defer tx.Rollback()

	statements := []struct {
		query string
		id    int64
	}{
		{"DELETE FROM billing_usage_entries WHERE user_id = $1", fixture.userID},
		{"DELETE FROM usage_logs WHERE user_id = $1", fixture.userID},
		{"DELETE FROM user_subscriptions WHERE id = $1", fixture.subscriptionID},
		{"DELETE FROM api_keys WHERE id = $1", fixture.apiKeyID},
		{"DELETE FROM users WHERE id = $1", fixture.userID},
		{"DELETE FROM accounts WHERE id = $1", fixture.accountID},
		{"DELETE FROM groups WHERE id = $1", fixture.groupID},
	}
	for _, statement := range statements {
		_, execErr := tx.ExecContext(context.Background(), statement.query, statement.id)
		require.NoError(t, execErr, fmt.Sprintf("cleanup query failed: %s", statement.query))
	}
	require.NoError(t, tx.Commit())
}
