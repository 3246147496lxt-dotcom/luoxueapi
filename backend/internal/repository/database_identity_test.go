package repository

import (
	"context"
	"errors"
	"testing"
	"testing/fstest"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

const databaseIdentityQueryPattern = `SELECT current_database\(\), system_identifier::text, pg_is_in_recovery\(\)\s+FROM pg_control_system\(\)`

func validExpectedDatabaseIdentity() ConfiguredDatabaseIdentity {
	return ConfiguredDatabaseIdentity{
		Contract:         ConfiguredDatabaseIdentityContract,
		Database:         "sub2api",
		SystemIdentifier: "7612345678901234567",
		InRecovery:       false,
	}
}

func TestQueryDatabaseIdentity(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(databaseIdentityQueryPattern).
		WillReturnRows(sqlmock.NewRows([]string{"current_database", "system_identifier", "in_recovery"}).
			AddRow("sub2api", "7612345678901234567", false))

	identity, err := queryDatabaseIdentity(context.Background(), db)
	require.NoError(t, err)
	require.Equal(t, ConfiguredDatabaseIdentity{
		Contract:         ConfiguredDatabaseIdentityContract,
		Database:         "sub2api",
		SystemIdentifier: "7612345678901234567",
		InRecovery:       false,
	}, identity)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestQueryDatabaseIdentityFailsClosed(t *testing.T) {
	t.Run("query error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		mock.ExpectQuery(databaseIdentityQueryPattern).
			WillReturnError(errors.New("identity unavailable"))

		_, err = queryDatabaseIdentity(context.Background(), db)
		require.ErrorContains(t, err, "query database identity")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty field", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		mock.ExpectQuery(databaseIdentityQueryPattern).
			WillReturnRows(sqlmock.NewRows([]string{"current_database", "system_identifier", "in_recovery"}).
				AddRow("sub2api", " ", false))

		_, err = queryDatabaseIdentity(context.Background(), db)
		require.ErrorContains(t, err, "empty identity field")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestValidateExpectedDatabaseIdentity(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(*ConfiguredDatabaseIdentity)
		contains string
	}{
		{
			name: "wrong contract",
			mutate: func(identity *ConfiguredDatabaseIdentity) {
				identity.Contract = "other/v1"
			},
			contains: "contract",
		},
		{
			name: "blank database",
			mutate: func(identity *ConfiguredDatabaseIdentity) {
				identity.Database = ""
			},
			contains: "database name",
		},
		{
			name: "nonnumeric system identifier",
			mutate: func(identity *ConfiguredDatabaseIdentity) {
				identity.SystemIdentifier = "cluster-a"
			},
			contains: "must be decimal",
		},
		{
			name: "standby",
			mutate: func(identity *ConfiguredDatabaseIdentity) {
				identity.InRecovery = true
			},
			contains: "writable primary",
		},
	}

	require.NoError(t, ValidateExpectedDatabaseIdentity(validExpectedDatabaseIdentity()))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			identity := validExpectedDatabaseIdentity()
			tt.mutate(&identity)
			require.ErrorContains(t, ValidateExpectedDatabaseIdentity(identity), tt.contains)
		})
	}
}

func TestApplyMigrationsFSVerifiesExpectedIdentityBeforeLock(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(databaseIdentityQueryPattern).
		WillReturnRows(sqlmock.NewRows([]string{"current_database", "system_identifier", "in_recovery"}).
			AddRow("sub2api", "7612345678901234567", false))
	mock.ExpectQuery("SELECT pg_try_advisory_lock\\(\\$1\\)").
		WithArgs(migrationsAdvisoryLockID).
		WillReturnError(errors.New("lock probe"))

	err = applyMigrationsFSWithExpectedDatabaseIdentity(
		context.Background(),
		db,
		fstest.MapFS{},
		func() *ConfiguredDatabaseIdentity {
			identity := validExpectedDatabaseIdentity()
			return &identity
		}(),
	)
	require.ErrorContains(t, err, "acquire migrations lock")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestApplyMigrationsFSWithExpectedIdentityRejectsNilIdentityBeforeSQL(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	err = applyMigrationsFSWithExpectedDatabaseIdentity(
		context.Background(),
		db,
		fstest.MapFS{},
		nil,
	)
	require.ErrorContains(t, err, "require an expected database identity")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestApplyMigrationsFSRejectsIdentityMismatchBeforeLock(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(databaseIdentityQueryPattern).
		WillReturnRows(sqlmock.NewRows([]string{"current_database", "system_identifier", "in_recovery"}).
			AddRow("other", "7612345678901234567", false))

	expected := validExpectedDatabaseIdentity()
	err = applyMigrationsFSWithExpectedDatabaseIdentity(
		context.Background(),
		db,
		fstest.MapFS{},
		&expected,
	)
	require.ErrorContains(t, err, "identity mismatch")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestApplyMigrationsFSRejectsSystemIdentifierMismatchBeforeLock(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(databaseIdentityQueryPattern).
		WillReturnRows(sqlmock.NewRows([]string{"current_database", "system_identifier", "in_recovery"}).
			AddRow("sub2api", "7999999999999999999", false))

	expected := validExpectedDatabaseIdentity()
	err = applyMigrationsFSWithExpectedDatabaseIdentity(
		context.Background(),
		db,
		fstest.MapFS{},
		&expected,
	)
	require.ErrorContains(t, err, "identity mismatch")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestApplyMigrationsFSRejectsStandbyBeforeLock(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(databaseIdentityQueryPattern).
		WillReturnRows(sqlmock.NewRows([]string{"current_database", "system_identifier", "in_recovery"}).
			AddRow("sub2api", "7612345678901234567", true))

	expected := validExpectedDatabaseIdentity()
	err = applyMigrationsFSWithExpectedDatabaseIdentity(
		context.Background(),
		db,
		fstest.MapFS{},
		&expected,
	)
	require.ErrorContains(t, err, "refusing standby target")
	require.NoError(t, mock.ExpectationsWereMet())
}
