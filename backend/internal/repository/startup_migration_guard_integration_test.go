//go:build integration

package repository

import (
	"context"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/require"
)

func TestApplyMigrationsBlocksMaintenanceMigrationOnPersistedFreshOrigin(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db := openIsolatedMigrationIntegrationDB(t, "sub2api_startup_guard_fresh_origin")
	oldReleaseFS := fstest.MapFS{
		"194a_startup_guard_probe.sql": &fstest.MapFile{Data: []byte(`
			CREATE TABLE startup_guard_probe (id BIGINT PRIMARY KEY);
		`)},
	}
	require.NoError(t, applyMigrationsFS(ctx, db, oldReleaseFS))
	requireMigrationOrigin(t, ctx, db, schemaMigrationOriginFresh)

	err := ApplyMigrations(ctx, db)
	require.ErrorContains(t, err, subscriptionAnchoredMonthlyQuotaMigration)
	require.ErrorContains(t, err, "--migrate-only")

	var applied bool
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM schema_migrations
			WHERE filename = $1
		)
	`, subscriptionAnchoredMonthlyQuotaMigration).Scan(&applied))
	require.False(t, applied)
}

func TestApplyMigrationsBlocksExistingSchemaWithoutMigrationMetadata(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db := openIsolatedMigrationIntegrationDB(t, "sub2api_startup_guard_missing_metadata")
	_, err := db.ExecContext(ctx, `CREATE TABLE legacy_business_data (id BIGINT PRIMARY KEY)`)
	require.NoError(t, err)

	err = ApplyMigrations(ctx, db)
	require.ErrorContains(t, err, subscriptionAnchoredMonthlyQuotaMigration)
	require.ErrorContains(t, err, "existing public tables")
	require.ErrorContains(t, err, "--migrate-only")

	var hasSchemaMigrations bool
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = 'public'
			  AND table_name = 'schema_migrations'
		)
	`).Scan(&hasSchemaMigrations))
	require.False(t, hasSchemaMigrations)
}
