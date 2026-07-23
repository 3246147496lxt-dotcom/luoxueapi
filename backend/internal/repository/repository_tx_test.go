package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
)

func TestWithRepositoryTxCommitsOwnedTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	t.Cleanup(func() { _ = client.Close() })

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE accounts").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	committedHere, err := withRepositoryTx(context.Background(), client, "test mutation", func(ctx context.Context, txClient *dbent.Client) error {
		_, err := txClient.ExecContext(ctx, "UPDATE accounts SET updated_at = NOW() WHERE id = 1")
		return err
	})

	require.NoError(t, err)
	require.True(t, committedHere)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestWithRepositoryTxRollsBackCallbackFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	t.Cleanup(func() { _ = client.Close() })

	wantErr := errors.New("outbox failed")
	mock.ExpectBegin()
	mock.ExpectRollback()

	committedHere, err := withRepositoryTx(context.Background(), client, "test mutation", func(context.Context, *dbent.Client) error {
		return wantErr
	})

	require.ErrorIs(t, err, wantErr)
	require.False(t, committedHere)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestWithRepositoryTxReusesCallerTransactionWithoutNestedBegin(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	t.Cleanup(func() { _ = client.Close() })

	mock.ExpectBegin()
	tx, err := client.Tx(context.Background())
	require.NoError(t, err)
	txCtx := dbent.NewTxContext(context.Background(), tx)
	mock.ExpectExec("UPDATE groups").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectRollback()

	committedHere, err := withRepositoryTx(txCtx, client, "nested mutation", func(ctx context.Context, txClient *dbent.Client) error {
		_, err := txClient.ExecContext(ctx, "UPDATE groups SET updated_at = NOW() WHERE id = 1")
		return err
	})

	require.NoError(t, err)
	require.False(t, committedHere)
	require.NoError(t, tx.Rollback())
	require.NoError(t, mock.ExpectationsWereMet())
}
