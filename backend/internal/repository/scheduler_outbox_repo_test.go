package repository

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestSchedulerOutboxRepositoryListAfterUsesCommitFenceAndClampsBatch(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &schedulerOutboxRepository{db: db}
	createdAt := time.Now().UTC().Truncate(time.Microsecond)
	const expectedSQL = `
		WITH selected AS MATERIALIZED (
			SELECT id, event_type, account_id, group_id, payload, created_at
			FROM scheduler_outbox
			WHERE id > $1
			ORDER BY id ASC
			LIMIT $2
			FOR UPDATE
		), released AS (
			UPDATE scheduler_outbox AS o
			SET dedup_key = NULL
			FROM selected AS s
			WHERE o.id = s.id
				AND o.dedup_key IS NOT NULL
			RETURNING o.id
		)
		SELECT s.id, s.event_type, s.account_id, s.group_id, s.payload, s.created_at
		FROM selected AS s
		CROSS JOIN (SELECT COUNT(*) FROM released) AS release_barrier
		ORDER BY s.id ASC
	`

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("SET LOCAL lock_timeout = '1s'")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta("LOCK TABLE scheduler_outbox IN SHARE ROW EXCLUSIVE MODE")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta(expectedSQL)).
		WithArgs(int64(41), schedulerOutboxMaxPollSize).
		WillReturnRows(sqlmock.NewRows([]string{"id", "event_type", "account_id", "group_id", "payload", "created_at"}).
			AddRow(int64(42), "account_changed", int64(7), nil, []byte(`{"group_ids":[3]}`), createdAt))
	mock.ExpectCommit()

	events, err := repo.ListAfterAndReleaseDedup(context.Background(), 41, 999)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.EqualValues(t, 42, events[0].ID)
	require.EqualValues(t, 7, *events[0].AccountID)
	require.Equal(t, []any{float64(3)}, events[0].Payload["group_ids"])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSchedulerOutboxRepositoryListAfterRollsBackWhenFenceTimesOut(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &schedulerOutboxRepository{db: db}
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("SET LOCAL lock_timeout = '1s'")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta("LOCK TABLE scheduler_outbox IN SHARE ROW EXCLUSIVE MODE")).
		WillReturnError(errors.New("lock timeout"))
	mock.ExpectRollback()

	events, err := repo.ListAfterAndReleaseDedup(context.Background(), 0, 100)
	require.Error(t, err)
	require.Contains(t, err.Error(), "commit fence")
	require.Nil(t, events)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSchedulerOutboxRepositoryFirstCreatedAtAfter(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &schedulerOutboxRepository{db: db}
	createdAt := time.Now().UTC().Truncate(time.Microsecond)
	const expectedSQL = `
		SELECT created_at
		FROM scheduler_outbox
		WHERE id > $1
		ORDER BY id ASC
		LIMIT 1
	`
	mock.ExpectQuery(regexp.QuoteMeta(expectedSQL)).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"created_at"}).AddRow(createdAt))

	got, ok, err := repo.FirstCreatedAtAfter(context.Background(), 42)

	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, createdAt, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSchedulerOutboxRepositoryFirstCreatedAtAfterReturnsNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &schedulerOutboxRepository{db: db}
	const expectedSQL = `
		SELECT created_at
		FROM scheduler_outbox
		WHERE id > $1
		ORDER BY id ASC
		LIMIT 1
	`
	mock.ExpectQuery(regexp.QuoteMeta(expectedSQL)).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"created_at"}))

	got, ok, err := repo.FirstCreatedAtAfter(context.Background(), 42)

	require.NoError(t, err)
	require.False(t, ok)
	require.True(t, got.IsZero())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSchedulerOutboxRepositoryDeleteConsumedUpToUsesBoundedCTE(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &schedulerOutboxRepository{db: db}
	const expectedSQL = `
		WITH doomed AS (
			SELECT id
			FROM scheduler_outbox
			WHERE id <= $1
				AND created_at < NOW() - INTERVAL '10 seconds'
			ORDER BY id ASC
			LIMIT $2
		)
		DELETE FROM scheduler_outbox o
		USING doomed d
		WHERE o.id = d.id
	`
	mock.ExpectExec(regexp.QuoteMeta(expectedSQL)).
		WithArgs(int64(42), 5000).
		WillReturnResult(sqlmock.NewResult(0, 17))

	deleted, err := repo.DeleteConsumedUpTo(context.Background(), 42, 5000)

	require.NoError(t, err)
	require.EqualValues(t, 17, deleted)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSchedulerOutboxRepositoryDeleteConsumedUpToSkipsNonPositiveWatermark(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &schedulerOutboxRepository{db: db}

	deleted, err := repo.DeleteConsumedUpTo(context.Background(), 0, 5000)

	require.NoError(t, err)
	require.EqualValues(t, 0, deleted)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSchedulerOutboxRepositoryConsumerLeaseAcquireRelease(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &schedulerOutboxRepository{db: db}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT pg_try_advisory_lock(hashtext($1))")).
		WithArgs(schedulerOutboxConsumerLockName).
		WillReturnRows(sqlmock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(true))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT pg_advisory_unlock(hashtext($1))")).
		WithArgs(schedulerOutboxConsumerLockName).
		WillReturnRows(sqlmock.NewRows([]string{"pg_advisory_unlock"}).AddRow(true))
	lease, acquired, err := repo.TryAcquireConsumerLease(context.Background())
	require.NoError(t, err)
	require.True(t, acquired)
	require.NotNil(t, lease)
	require.NoError(t, lease.Release())
	require.NoError(t, lease.Release(), "release must be idempotent")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSchedulerOutboxRepositoryConsumerLeaseBoundContextExpiresOnRelease(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &schedulerOutboxRepository{db: db}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT pg_try_advisory_lock(hashtext($1))")).
		WithArgs(schedulerOutboxConsumerLockName).
		WillReturnRows(sqlmock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(true))
	lease, acquired, err := repo.TryAcquireConsumerLease(context.Background())
	require.NoError(t, err)
	require.True(t, acquired)

	boundCtx, err := lease.BindContext(context.Background())
	require.NoError(t, err)
	client, ok := schedulerOutboxReadClientFromContext(boundCtx)
	require.True(t, ok)
	require.Same(t, client, clientFromContext(boundCtx, nil))
	mock.ExpectQuery(`SELECT COUNT\(.*\) FROM .*accounts.*`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT pg_advisory_unlock(hashtext($1))")).
		WithArgs(schedulerOutboxConsumerLockName).
		WillReturnRows(sqlmock.NewRows([]string{"pg_advisory_unlock"}).AddRow(true))
	count, err := client.Account.Query().Count(boundCtx)
	require.NoError(t, err)
	require.Zero(t, count)

	require.NoError(t, lease.Release())
	_, err = client.Account.Query().All(boundCtx)
	require.ErrorIs(t, err, errSchedulerOutboxConsumerLeaseReleased)
	_, err = lease.BindContext(context.Background())
	require.ErrorIs(t, err, errSchedulerOutboxConsumerLeaseReleased)
	require.NoError(t, lease.Release(), "release must remain idempotent")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSchedulerOutboxRepositoryConsumerLeaseRowsBlockReleaseUntilClosed(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &schedulerOutboxRepository{db: db}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT pg_try_advisory_lock(hashtext($1))")).
		WithArgs(schedulerOutboxConsumerLockName).
		WillReturnRows(sqlmock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(true))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT 1")).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT pg_advisory_unlock(hashtext($1))")).
		WithArgs(schedulerOutboxConsumerLockName).
		WillReturnRows(sqlmock.NewRows([]string{"pg_advisory_unlock"}).AddRow(true))

	lease, acquired, err := repo.TryAcquireConsumerLease(context.Background())
	require.NoError(t, err)
	require.True(t, acquired)
	rawLease, ok := lease.(*schedulerOutboxConsumerLease)
	require.True(t, ok)
	driver := &schedulerOutboxReadDriver{lease: rawLease}
	var rows entsql.Rows
	require.NoError(t, driver.Query(context.Background(), "SELECT 1", []any{}, &rows))

	releaseDone := make(chan error, 1)
	go func() { releaseDone <- lease.Release() }()
	select {
	case err := <-releaseDone:
		t.Fatalf("lease released while bound rows were open: %v", err)
	case <-time.After(20 * time.Millisecond):
	}

	require.NoError(t, rows.Close())
	select {
	case err := <-releaseDone:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("lease release did not resume after rows closed")
	}
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSchedulerOutboxRepositoryConsumerLeaseUnavailable(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &schedulerOutboxRepository{db: db}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT pg_try_advisory_lock(hashtext($1))")).
		WithArgs(schedulerOutboxConsumerLockName).
		WillReturnRows(sqlmock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(false))

	lease, acquired, err := repo.TryAcquireConsumerLease(context.Background())
	require.NoError(t, err)
	require.False(t, acquired)
	require.Nil(t, lease)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSchedulerOutboxRepositoryConsumerLeaseAcquireError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &schedulerOutboxRepository{db: db}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT pg_try_advisory_lock(hashtext($1))")).
		WithArgs(schedulerOutboxConsumerLockName).
		WillReturnError(errors.New("acquire failed"))

	lease, acquired, err := repo.TryAcquireConsumerLease(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "acquire scheduler outbox consumer lease")
	require.False(t, acquired)
	require.Nil(t, lease)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSchedulerOutboxRepositoryConsumerLeaseRejectsUncertainUnlock(t *testing.T) {
	tests := []struct {
		name      string
		configure func(sqlmock.Sqlmock)
		contains  string
	}{
		{
			name: "unlock false",
			configure: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT pg_advisory_unlock(hashtext($1))")).
					WithArgs(schedulerOutboxConsumerLockName).
					WillReturnRows(sqlmock.NewRows([]string{"pg_advisory_unlock"}).AddRow(false))
			},
			contains: "did not hold",
		},
		{
			name: "unlock error",
			configure: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT pg_advisory_unlock(hashtext($1))")).
					WithArgs(schedulerOutboxConsumerLockName).
					WillReturnError(errors.New("unlock failed"))
			},
			contains: "unlock failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer func() { _ = db.Close() }()

			mock.ExpectQuery(regexp.QuoteMeta("SELECT pg_try_advisory_lock(hashtext($1))")).
				WithArgs(schedulerOutboxConsumerLockName).
				WillReturnRows(sqlmock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(true))
			tt.configure(mock)
			repo := &schedulerOutboxRepository{db: db}
			lease, acquired, err := repo.TryAcquireConsumerLease(context.Background())
			require.NoError(t, err)
			require.True(t, acquired)

			err = lease.Release()
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.contains)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestSchedulerOutboxRepositoryTryAcquireCleanupLock(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &schedulerOutboxRepository{db: db}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT pg_try_advisory_lock(hashtext($1))")).
		WithArgs(schedulerOutboxCleanupLockName).
		WillReturnRows(sqlmock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(true))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT pg_advisory_unlock(hashtext($1))")).
		WithArgs(schedulerOutboxCleanupLockName).
		WillReturnRows(sqlmock.NewRows([]string{"pg_advisory_unlock"}).AddRow(true))

	lease, acquired, err := repo.TryAcquireCleanupLock(context.Background())
	require.NoError(t, err)
	require.True(t, acquired)
	require.NotNil(t, lease)

	require.NoError(t, lease.Release())
	require.NoError(t, lease.Release(), "release must be idempotent")

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSchedulerOutboxRepositoryTryAcquireCleanupLockUnavailable(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &schedulerOutboxRepository{db: db}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT pg_try_advisory_lock(hashtext($1))")).
		WithArgs(schedulerOutboxCleanupLockName).
		WillReturnRows(sqlmock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(false))

	lease, acquired, err := repo.TryAcquireCleanupLock(context.Background())
	require.NoError(t, err)
	require.False(t, acquired)
	require.Nil(t, lease)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSchedulerOutboxRepositoryCleanupLeaseAcquireError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &schedulerOutboxRepository{db: db}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT pg_try_advisory_lock(hashtext($1))")).
		WithArgs(schedulerOutboxCleanupLockName).
		WillReturnError(errors.New("acquire failed"))

	lease, acquired, err := repo.TryAcquireCleanupLock(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "acquire scheduler outbox cleanup lease")
	require.False(t, acquired)
	require.Nil(t, lease)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSchedulerOutboxRepositoryCleanupLeaseRejectsUncertainUnlock(t *testing.T) {
	tests := []struct {
		name      string
		configure func(sqlmock.Sqlmock)
		contains  string
	}{
		{
			name: "unlock false",
			configure: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT pg_advisory_unlock(hashtext($1))")).
					WithArgs(schedulerOutboxCleanupLockName).
					WillReturnRows(sqlmock.NewRows([]string{"pg_advisory_unlock"}).AddRow(false))
			},
			contains: "did not hold",
		},
		{
			name: "unlock error",
			configure: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT pg_advisory_unlock(hashtext($1))")).
					WithArgs(schedulerOutboxCleanupLockName).
					WillReturnError(errors.New("unlock failed"))
			},
			contains: "unlock failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer func() { _ = db.Close() }()

			mock.ExpectQuery(regexp.QuoteMeta("SELECT pg_try_advisory_lock(hashtext($1))")).
				WithArgs(schedulerOutboxCleanupLockName).
				WillReturnRows(sqlmock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(true))
			tt.configure(mock)
			repo := &schedulerOutboxRepository{db: db}
			lease, acquired, err := repo.TryAcquireCleanupLock(context.Background())
			require.NoError(t, err)
			require.True(t, acquired)

			err = lease.Release()
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.contains)
			require.NoError(t, lease.Release(), "failed release must still be idempotent")
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// buildSchedulerGroupPayload 在 groupIDs 为空时必须返回 untyped nil（any），
// 否则 enqueueSchedulerOutbox 的 "payload != nil" 接口判空会被 typed-nil 欺骗，
// 把 payload marshal 成 "null" 写入 dedup_key 哈希，破坏与其他 nil-payload
// 调用的去重一致性。本测试用 ungrouped 账号场景验证两条路径的 dedup_key 一致。
func TestEnqueueSchedulerOutbox_UngroupedAccountDedupesWithLiteralNilPayload(t *testing.T) {
	accountID := int64(42)

	// Path A: 显式 nil payload（如 SetError、SetStatus 等调用模式）
	keyLiteralNil := schedulerOutboxDedupKey("account_changed", &accountID, nil, nil)

	// Path B: buildSchedulerGroupPayload(account.GroupIDs) 当账号没有任何分组
	emptyGroupsPayload := buildSchedulerGroupPayload(nil)
	require.Nil(t, emptyGroupsPayload,
		"buildSchedulerGroupPayload(empty) must return untyped-nil any to avoid typed-nil marshal")

	// 模拟 enqueueSchedulerOutbox 内部的判空逻辑
	var payloadJSON []byte
	if emptyGroupsPayload != nil {
		t.Fatalf("typed-nil regression: buildSchedulerGroupPayload(empty) interface should be nil")
	}
	keyEmptyGroups := schedulerOutboxDedupKey("account_changed", &accountID, nil, payloadJSON)

	require.Equal(t, keyLiteralNil, keyEmptyGroups,
		"ungrouped-account account_changed must share dedup_key with other nil-payload variants")
}
