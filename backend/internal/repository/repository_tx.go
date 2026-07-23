package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/opsruntime"
)

// withRepositoryTx executes fn in the caller-owned Ent transaction when one is
// present, otherwise it creates and commits a repository-owned transaction.
// The returned boolean is true only after a transaction opened here has
// committed. Callers use it to keep best-effort cache propagation behind the
// durable commit boundary.
func withRepositoryTx(
	ctx context.Context,
	rootClient *dbent.Client,
	operation string,
	fn func(context.Context, *dbent.Client) error,
) (bool, error) {
	if fn == nil {
		return false, errors.New("repository transaction callback is nil")
	}
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return false, fn(ctx, tx.Client())
	}
	if rootClient == nil {
		return false, errors.New("repository transaction client is not configured")
	}

	tx, err := rootClient.Tx(ctx)
	if errors.Is(err, dbent.ErrTxStarted) {
		// Compatibility for repositories constructed with tx.Client() without a
		// matching dbent.NewTxContext. The caller owns that transaction, so this
		// helper must neither begin nor commit another one.
		return false, fn(ctx, rootClient)
	}
	if err != nil {
		return false, fmt.Errorf("begin %s transaction: %w", operation, err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	txCtx := dbent.NewTxContext(ctx, tx)
	if err := fn(txCtx, tx.Client()); err != nil {
		opsruntime.ObserveProducerTransactionRollback()
		return false, err
	}
	if err := tx.Commit(); err != nil {
		opsruntime.ObserveProducerTransactionRollback()
		return false, fmt.Errorf("commit %s transaction: %w", operation, err)
	}
	committed = true
	return true, nil
}

// sqlExecutorFromContext keeps raw SQL mutations in the same Ent transaction
// as their caller instead of silently escaping through the repository's base
// *sql.DB.
func sqlExecutorFromContext(ctx context.Context, fallback sqlExecutor) sqlExecutor {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return tx.Client()
	}
	return fallback
}

func rawStatementCommittedHere(ctx context.Context, exec sqlExecutor) bool {
	if dbent.TxFromContext(ctx) != nil {
		return false
	}
	_, ok := exec.(*sql.DB)
	return ok
}
