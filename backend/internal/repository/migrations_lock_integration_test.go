//go:build integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

const migrationIntegrationBlockLockID int64 = 694208311321144028

// TestApplyMigrationsFS_SerializesInstancesOnOneBackendSession exercises the
// real PostgreSQL session semantics that sqlmock cannot model. The first
// instance holds the migration lock while blocked inside its first migration;
// a second instance must be cancellable while waiting. After release, a fresh
// instance must observe the committed migration and both migration files must
// have recorded the same backend PID.
func TestApplyMigrationsFS_SerializesInstancesOnOneBackendSession(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	databaseName := fmt.Sprintf("sub2api_migration_lock_%d", time.Now().UnixNano())
	_, err := integrationDB.ExecContext(ctx, "CREATE DATABASE "+pq.QuoteIdentifier(databaseName))
	require.NoError(t, err)
	t.Cleanup(func() {
		dropCtx, dropCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer dropCancel()
		_, _ = integrationDB.ExecContext(
			dropCtx,
			"DROP DATABASE IF EXISTS "+pq.QuoteIdentifier(databaseName)+" WITH (FORCE)",
		)
	})

	dsn := migrationIntegrationDatabaseDSN(t, integrationDSN, databaseName)
	firstDB := openMigrationIntegrationDB(t, dsn)
	secondDB := openMigrationIntegrationDB(t, dsn)

	blocker, err := firstDB.Conn(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = blocker.Close() })
	_, err = blocker.ExecContext(ctx, "SELECT pg_advisory_lock($1)", migrationIntegrationBlockLockID)
	require.NoError(t, err)
	blockReleased := false
	t.Cleanup(func() {
		if blockReleased {
			return
		}
		unlockCtx, unlockCancel := context.WithTimeout(context.Background(), time.Second)
		defer unlockCancel()
		_, _ = blocker.ExecContext(unlockCtx, "SELECT pg_advisory_unlock($1)", migrationIntegrationBlockLockID)
	})

	migrationFS := fstest.MapFS{
		"001_create_probe.sql": &fstest.MapFile{Data: []byte(fmt.Sprintf(`
			SELECT pg_advisory_lock(%d);
			SELECT pg_advisory_unlock(%d);
			CREATE TABLE migration_session_probe (
				stage TEXT PRIMARY KEY,
				backend_pid INTEGER NOT NULL
			);
			INSERT INTO migration_session_probe(stage, backend_pid)
			VALUES ('first', pg_backend_pid());
		`, migrationIntegrationBlockLockID, migrationIntegrationBlockLockID))},
		"002_record_probe.sql": &fstest.MapFile{Data: []byte(`
			INSERT INTO migration_session_probe(stage, backend_pid)
			VALUES ('second', pg_backend_pid());
		`)},
	}

	firstResult := make(chan error, 1)
	go func() {
		firstResult <- applyMigrationsFS(ctx, firstDB, migrationFS)
	}()

	require.Eventually(t, func() bool {
		var tableName sql.NullString
		queryCtx, queryCancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		defer queryCancel()
		if err := secondDB.QueryRowContext(queryCtx, "SELECT to_regclass('public.schema_migrations')::text").Scan(&tableName); err != nil {
			return false
		}
		return tableName.Valid
	}, 5*time.Second, 25*time.Millisecond, "first instance never reached the locked migration section")

	waitCtx, waitCancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	waitErr := applyMigrationsFS(waitCtx, secondDB, migrationFS)
	waitCancel()
	require.Error(t, waitErr)
	require.ErrorIs(t, waitErr, context.DeadlineExceeded)
	require.Contains(t, waitErr.Error(), "acquire migrations lock")

	_, err = blocker.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", migrationIntegrationBlockLockID)
	require.NoError(t, err)
	blockReleased = true

	select {
	case err := <-firstResult:
		require.NoError(t, err)
	case <-ctx.Done():
		t.Fatal("first migration instance did not finish after releasing the blocker")
	}

	require.NoError(t, applyMigrationsFS(ctx, secondDB, migrationFS))

	rows, err := secondDB.QueryContext(ctx, "SELECT backend_pid FROM migration_session_probe ORDER BY stage")
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	var backendPIDs []int
	for rows.Next() {
		var backendPID int
		require.NoError(t, rows.Scan(&backendPID))
		backendPIDs = append(backendPIDs, backendPID)
	}
	require.NoError(t, rows.Err())
	require.Len(t, backendPIDs, 2)
	require.Equal(t, backendPIDs[0], backendPIDs[1], "all migration SQL must use one fixed PostgreSQL backend session")
}

func migrationIntegrationDatabaseDSN(t *testing.T, rawDSN, databaseName string) string {
	t.Helper()
	parsed, err := url.Parse(rawDSN)
	require.NoError(t, err)
	require.True(t, parsed.IsAbs())
	parsed.Path = "/" + databaseName
	return parsed.String()
}

func openMigrationIntegrationDB(t *testing.T, dsn string) *sql.DB {
	t.Helper()
	db, err := sql.Open("postgres", strings.TrimSpace(dsn))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	require.NoError(t, db.PingContext(pingCtx))
	return db
}
