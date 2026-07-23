//go:build integration

package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
)

func TestSchedulerSnapshotOutboxReplay(t *testing.T) {
	ctx := context.Background()
	rdb := testRedis(t)
	client := testEntClient(t)

	_, _ = integrationDB.ExecContext(ctx, "TRUNCATE scheduler_outbox")

	accountRepo := newAccountRepositoryWithSQL(client, integrationDB, nil)
	outboxRepo := NewSchedulerOutboxRepository(integrationDB)
	cache := NewSchedulerCache(rdb)

	cfg := &config.Config{
		RunMode: config.RunModeStandard,
		Gateway: config.GatewayConfig{
			Scheduling: config.GatewaySchedulingConfig{
				OutboxPollIntervalSeconds:  1,
				FullRebuildIntervalSeconds: 0,
				DbFallbackEnabled:          true,
			},
		},
	}

	account := &service.Account{
		Name:        "outbox-replay-" + time.Now().Format("150405.000000"),
		Platform:    service.PlatformOpenAI,
		Type:        service.AccountTypeAPIKey,
		Status:      service.StatusActive,
		Schedulable: true,
		Concurrency: 3,
		Priority:    1,
		Credentials: map[string]any{},
		Extra:       map[string]any{},
	}
	require.NoError(t, accountRepo.Create(ctx, account))
	require.NoError(t, cache.SetAccount(ctx, account))

	svc := service.NewSchedulerSnapshotService(cache, outboxRepo, accountRepo, nil, cfg)
	svc.Start()
	t.Cleanup(svc.Stop)

	require.NoError(t, accountRepo.UpdateLastUsed(ctx, account.ID))
	updated, err := accountRepo.GetByID(ctx, account.ID)
	require.NoError(t, err)
	require.NotNil(t, updated.LastUsedAt)
	expectedUnix := updated.LastUsedAt.Unix()

	require.Eventually(t, func() bool {
		cached, err := cache.GetAccount(ctx, account.ID)
		if err != nil || cached == nil || cached.LastUsedAt == nil {
			return false
		}
		return cached.LastUsedAt.Unix() == expectedUnix
	}, 5*time.Second, 100*time.Millisecond)
}

func TestSchedulerOutboxConsumerLeaseSerializesRepositories(t *testing.T) {
	ctx := context.Background()
	firstRepo := NewSchedulerOutboxRepository(integrationDB)
	secondRepo := NewSchedulerOutboxRepository(integrationDB)

	firstLease, acquired, err := firstRepo.TryAcquireConsumerLease(ctx)
	require.NoError(t, err)
	require.True(t, acquired)
	require.NotNil(t, firstLease)

	secondLease, acquired, err := secondRepo.TryAcquireConsumerLease(ctx)
	require.NoError(t, err)
	require.False(t, acquired)
	require.Nil(t, secondLease)

	require.NoError(t, firstLease.Release())
	secondLease, acquired, err = secondRepo.TryAcquireConsumerLease(ctx)
	require.NoError(t, err)
	require.True(t, acquired)
	require.NotNil(t, secondLease)
	require.NoError(t, secondLease.Release())
}

func TestSchedulerOutboxCleanupLeaseSerializesRepositories(t *testing.T) {
	ctx := context.Background()
	firstRepo := NewSchedulerOutboxRepository(integrationDB)
	secondRepo := NewSchedulerOutboxRepository(integrationDB)

	firstLease, acquired, err := firstRepo.TryAcquireCleanupLock(ctx)
	require.NoError(t, err)
	require.True(t, acquired)
	require.NotNil(t, firstLease)

	secondLease, acquired, err := secondRepo.TryAcquireCleanupLock(ctx)
	require.NoError(t, err)
	require.False(t, acquired)
	require.Nil(t, secondLease)

	require.NoError(t, firstLease.Release())
	secondLease, acquired, err = secondRepo.TryAcquireCleanupLock(ctx)
	require.NoError(t, err)
	require.True(t, acquired)
	require.NotNil(t, secondLease)
	require.NoError(t, secondLease.Release())
}

func TestSchedulerOutboxConsumerLeaseUsesOwningConnectionWithSingleConnectionPool(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := integrationDB.ExecContext(ctx, "TRUNCATE scheduler_outbox RESTART IDENTITY")
	require.NoError(t, err)
	_, err = integrationDB.ExecContext(ctx, "INSERT INTO scheduler_outbox (event_type) VALUES ('single_connection_consumer')")
	require.NoError(t, err)

	db, err := sql.Open("postgres", integrationDSN)
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	t.Cleanup(func() { _ = db.Close() })
	repo := NewSchedulerOutboxRepository(db)
	lease, acquired, err := repo.TryAcquireConsumerLease(ctx)
	require.NoError(t, err)
	require.True(t, acquired)
	require.NotNil(t, lease)

	events, err := lease.ListAfterAndReleaseDedup(ctx, 0, 200)
	require.NoError(t, err, "protected read must reuse the advisory-lock session instead of waiting on the pool")
	require.Len(t, events, 1)
	require.Equal(t, "single_connection_consumer", events[0].EventType)
	require.NoError(t, lease.Release())
}

func TestSchedulerSnapshotPollUsesConsumerConnectionForCompleteBatch(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := integrationDB.ExecContext(ctx, "TRUNCATE scheduler_outbox RESTART IDENTITY")
	require.NoError(t, err)

	db, err := sql.Open("postgres", integrationDSN)
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	driver := entsql.OpenDB(dialect.Postgres, db)
	client := dbent.NewClient(dbent.Driver(driver))

	var groupID int64
	err = db.QueryRowContext(ctx, `
		INSERT INTO groups (name, description, platform, status, sort_order)
		VALUES ($1, '', $2, $3, 0)
		RETURNING id
	`, "single-connection-outbox-group-"+time.Now().Format("150405.000000"), service.PlatformOpenAI, service.StatusActive).Scan(&groupID)
	require.NoError(t, err)

	var accountID int64
	err = db.QueryRowContext(ctx, `
		INSERT INTO accounts (
			name, platform, type, credentials, extra, status, schedulable,
			concurrency, priority
		)
		VALUES ($1, $2, $3, '{}'::jsonb, '{}'::jsonb, $4, TRUE, 1, 1)
		RETURNING id
	`, "single-connection-outbox-account-"+time.Now().Format("150405.000000"), service.PlatformOpenAI, service.AccountTypeAPIKey, service.StatusActive).Scan(&accountID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `
		INSERT INTO account_groups (account_id, group_id, priority)
		VALUES ($1, $2, 1)
	`, accountID, groupID)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(context.Background(), "DELETE FROM scheduler_outbox WHERE account_id = $1 OR group_id = $2", accountID, groupID)
		_, _ = integrationDB.ExecContext(context.Background(), "DELETE FROM account_groups WHERE account_id = $1 OR group_id = $2", accountID, groupID)
		_, _ = integrationDB.ExecContext(context.Background(), "DELETE FROM accounts WHERE id = $1", accountID)
		_, _ = integrationDB.ExecContext(context.Background(), "DELETE FROM groups WHERE id = $1", groupID)
	})

	rdb := testRedis(t)
	cache := NewSchedulerCache(rdb)
	outboxRepo := NewSchedulerOutboxRepository(db)
	accountRepo := newAccountRepositoryWithSQL(client, db, nil)
	groupRepo := newGroupRepositoryWithSQL(client, db)
	cfg := &config.Config{
		RunMode: config.RunModeStandard,
		Gateway: config.GatewayConfig{
			Scheduling: config.GatewaySchedulingConfig{
				OutboxPollIntervalSeconds:  1,
				FullRebuildIntervalSeconds: 0,
				DbFallbackEnabled:          true,
			},
		},
	}
	svc := service.NewSchedulerSnapshotService(cache, outboxRepo, accountRepo, groupRepo, cfg)
	svc.Start()
	t.Cleanup(func() {
		_ = db.Close()
		svc.Stop()
	})

	require.Eventually(t, func() bool {
		status := svc.InitialSnapshotStatus()
		return status.Done && status.Err == nil
	}, 15*time.Second, 100*time.Millisecond, "initial rebuild must finish before injecting the batch")

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()
	_, err = tx.ExecContext(ctx, "UPDATE accounts SET concurrency = 9, updated_at = NOW() WHERE id = $1", accountID)
	require.NoError(t, err)
	var accountEventID int64
	err = tx.QueryRowContext(ctx, `
		INSERT INTO scheduler_outbox (event_type, account_id)
		VALUES ($1, $2)
		RETURNING id
	`, service.SchedulerOutboxEventAccountChanged, accountID).Scan(&accountEventID)
	require.NoError(t, err)
	var groupEventID int64
	err = tx.QueryRowContext(ctx, `
		INSERT INTO scheduler_outbox (event_type, group_id)
		VALUES ($1, $2)
		RETURNING id
	`, service.SchedulerOutboxEventGroupChanged, groupID).Scan(&groupEventID)
	require.NoError(t, err)
	var rebuildEventID int64
	err = tx.QueryRowContext(ctx, `
		INSERT INTO scheduler_outbox (event_type)
		VALUES ($1)
		RETURNING id
	`, service.SchedulerOutboxEventFullRebuild).Scan(&rebuildEventID)
	require.NoError(t, err)
	require.Less(t, accountEventID, groupEventID)
	require.Less(t, groupEventID, rebuildEventID)
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(
			context.Background(),
			"DELETE FROM scheduler_outbox WHERE id = $1 OR id = $2 OR id = $3",
			accountEventID,
			groupEventID,
			rebuildEventID,
		)
	})
	require.NoError(t, tx.Commit())

	require.Eventually(t, func() bool {
		watermark, err := cache.GetOutboxWatermark(context.Background())
		return err == nil && watermark >= rebuildEventID
	}, 20*time.Second, 100*time.Millisecond, "account/group/full rebuild batch must advance its watermark with one DB connection")

	cached, err := cache.GetAccount(context.Background(), accountID)
	require.NoError(t, err)
	require.NotNil(t, cached)
	require.Equal(t, 9, cached.Concurrency)
	svc.Stop()
}

func TestSchedulerOutboxCleanupLeaseUsesOwningConnectionWithSingleConnectionPool(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := integrationDB.ExecContext(ctx, "TRUNCATE scheduler_outbox RESTART IDENTITY")
	require.NoError(t, err)
	var eventID int64
	err = integrationDB.QueryRowContext(ctx, `
		INSERT INTO scheduler_outbox (event_type, created_at)
		VALUES ('single_connection_cleanup', NOW() - INTERVAL '1 minute')
		RETURNING id
	`).Scan(&eventID)
	require.NoError(t, err)

	db, err := sql.Open("postgres", integrationDSN)
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	t.Cleanup(func() { _ = db.Close() })
	repo := NewSchedulerOutboxRepository(db)
	lease, acquired, err := repo.TryAcquireCleanupLock(ctx)
	require.NoError(t, err)
	require.True(t, acquired)
	require.NotNil(t, lease)

	deleted, err := lease.DeleteConsumedUpTo(ctx, eventID, 100)
	require.NoError(t, err, "protected delete must reuse the advisory-lock session instead of waiting on the pool")
	require.EqualValues(t, 1, deleted)
	require.NoError(t, lease.Release())
}
