package repository

import (
	"context"
	"regexp"
	"testing"
	"testing/fstest"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

const tableExistsQueryPattern = `
	SELECT EXISTS (
		SELECT 1
		FROM information_schema.tables
		WHERE table_schema = 'public' AND table_name = $1
	)
`

const maintenanceMigrationAppliedQueryPattern = `
	SELECT EXISTS (
		SELECT 1
		FROM schema_migrations
		WHERE filename = $1
	)
`

const existingPublicTablesQueryPattern = `
	SELECT EXISTS (
		SELECT 1
		FROM pg_catalog.pg_class c
		JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public'
		  AND c.relkind IN ('r', 'p', 'v', 'm', 'f')
		  AND c.relname <> 'schema_migration_runner_state'
	)
`

func TestRejectPendingMaintenanceOnlyMigrationsAllowsFreshDatabase(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(regexp.QuoteMeta(tableExistsQueryPattern)).
		WithArgs("schema_migrations").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectQuery(regexp.QuoteMeta(existingPublicTablesQueryPattern)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	require.NoError(t, rejectPendingMaintenanceOnlyMigrations(context.Background(), db))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRejectPendingMaintenanceOnlyMigrationsBlocksExistingSchemaWithoutMetadata(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	expectSchemaMigrationsTable(t, mock, false)
	mock.ExpectQuery(regexp.QuoteMeta(existingPublicTablesQueryPattern)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	err = rejectPendingMaintenanceOnlyMigrations(context.Background(), db)
	require.ErrorContains(t, err, subscriptionAnchoredMonthlyQuotaMigration)
	require.ErrorContains(t, err, "existing public tables")
	require.ErrorContains(t, err, "--migrate-only")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRejectPendingMaintenanceOnlyMigrationsBlocksPersistedFreshDatabase(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	expectSchemaMigrationsTable(t, mock, true)
	mock.ExpectQuery(regexp.QuoteMeta(maintenanceMigrationAppliedQueryPattern)).
		WithArgs(subscriptionAnchoredMonthlyQuotaMigration).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	err = rejectPendingMaintenanceOnlyMigrations(context.Background(), db)
	require.ErrorContains(t, err, subscriptionAnchoredMonthlyQuotaMigration)
	require.ErrorContains(t, err, "--migrate-only")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRejectPendingMaintenanceOnlyMigrationsBlocksLegacyDatabase(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	expectSchemaMigrationsTable(t, mock, true)
	mock.ExpectQuery(regexp.QuoteMeta(maintenanceMigrationAppliedQueryPattern)).
		WithArgs(subscriptionAnchoredMonthlyQuotaMigration).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	err = rejectPendingMaintenanceOnlyMigrations(context.Background(), db)
	require.ErrorContains(t, err, subscriptionAnchoredMonthlyQuotaMigration)
	require.ErrorContains(t, err, "--migrate-only")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRejectPendingMaintenanceOnlyMigrationsAllowsAppliedLegacyDatabase(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	expectSchemaMigrationsTable(t, mock, true)
	mock.ExpectQuery(regexp.QuoteMeta(maintenanceMigrationAppliedQueryPattern)).
		WithArgs(subscriptionAnchoredMonthlyQuotaMigration).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	require.NoError(t, rejectPendingMaintenanceOnlyMigrations(context.Background(), db))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRejectPendingMaintenanceOnlyMigrationsFailsClosedWhenAppliedStateIsUnavailable(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	expectSchemaMigrationsTable(t, mock, true)
	mock.ExpectQuery(regexp.QuoteMeta(maintenanceMigrationAppliedQueryPattern)).
		WithArgs(subscriptionAnchoredMonthlyQuotaMigration).
		WillReturnError(context.DeadlineExceeded)

	err = rejectPendingMaintenanceOnlyMigrations(context.Background(), db)
	require.ErrorContains(t, err, "check maintenance-only migration")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOrdinaryStartupChecksMaintenanceGateOnLockedRunnerSessionBeforeSchemaWrites(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("SELECT pg_try_advisory_lock\\(\\$1\\)").
		WithArgs(migrationsAdvisoryLockID).
		WillReturnRows(sqlmock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(true))
	expectSchemaMigrationsTable(t, mock, true)
	mock.ExpectQuery(regexp.QuoteMeta(maintenanceMigrationAppliedQueryPattern)).
		WithArgs(subscriptionAnchoredMonthlyQuotaMigration).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	expectMigrationsUnlock(mock, true)

	err = applyMigrationsFSWithPolicy(
		context.Background(),
		db,
		fstest.MapFS{},
		migrationRunnerPolicy{},
	)
	require.ErrorContains(t, err, subscriptionAnchoredMonthlyQuotaMigration)
	require.ErrorContains(t, err, "--migrate-only")
	require.NoError(t, mock.ExpectationsWereMet(), "the gate must run after the lock and before runner metadata or DDL")
}

func TestOrdinaryStartupAllowsEmptyDatabaseThenBootstrapsOnSameLockedSession(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("SELECT pg_try_advisory_lock\\(\\$1\\)").
		WithArgs(migrationsAdvisoryLockID).
		WillReturnRows(sqlmock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(true))
	expectSchemaMigrationsTable(t, mock, false)
	mock.ExpectQuery(regexp.QuoteMeta(existingPublicTablesQueryPattern)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	expectSchemaMigrationsTable(t, mock, false)
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS schema_migration_runner_state").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO schema_migration_runner_state").
		WithArgs(schemaMigrationOriginStateKey, schemaMigrationOriginFresh).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT state_value FROM schema_migration_runner_state").
		WithArgs(schemaMigrationOriginStateKey).
		WillReturnRows(sqlmock.NewRows([]string{"state_value"}).AddRow(schemaMigrationOriginFresh))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS schema_migrations").
		WillReturnResult(sqlmock.NewResult(0, 0))
	expectMigrationsUnlock(mock, true)

	err = applyMigrationsFSWithPolicy(
		context.Background(),
		db,
		fstest.MapFS{},
		migrationRunnerPolicy{},
	)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMaintenancePolicyCannotApplyMaintenanceMigrationWithoutIdentity(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	err = applyMigrationsFSWithPolicy(
		context.Background(),
		db,
		fstest.MapFS{
			subscriptionAnchoredMonthlyQuotaMigration: &fstest.MapFile{Data: []byte("SELECT 1;")},
		},
		migrationRunnerPolicy{allowMaintenanceOnly: true},
	)
	require.ErrorContains(t, err, "require an expected database identity")
	require.NoError(t, mock.ExpectationsWereMet(), "identity policy must fail before issuing SQL")
}

func expectSchemaMigrationsTable(t *testing.T, mock sqlmock.Sqlmock, exists bool) {
	t.Helper()
	mock.ExpectQuery(regexp.QuoteMeta(tableExistsQueryPattern)).
		WithArgs("schema_migrations").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(exists))
}
