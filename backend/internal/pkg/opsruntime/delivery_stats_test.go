package opsruntime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDeliveryRuntimeStatsSnapshot(t *testing.T) {
	resetDeliveryRuntimeStatsForTest()

	ObserveMigrationLock(25*time.Millisecond, true, false, true)
	ObserveMigrationLock(40*time.Millisecond, true, true, false)
	ObserveOutboxFence(7*time.Millisecond, true, false)
	ObserveOutboxFence(11*time.Millisecond, false, true)
	ObserveOutboxBatch(3)
	ObserveOutboxBatch(0)
	ObserveOutboxBacklog(17)
	ObserveProducerTransactionRollback()

	stats := SnapshotDeliveryRuntimeStats()
	require.EqualValues(t, 2, stats.MigrationLockAttemptCount)
	require.EqualValues(t, 1, stats.MigrationLockAcquiredCount)
	require.EqualValues(t, 2, stats.MigrationLockWaitCount)
	require.EqualValues(t, 65*time.Millisecond, time.Duration(stats.MigrationLockWaitNanoseconds))
	require.EqualValues(t, 1, stats.MigrationLockTimeoutCount)
	require.EqualValues(t, 2, stats.OutboxFenceAttemptCount)
	require.EqualValues(t, 1, stats.OutboxFenceAcquiredCount)
	require.EqualValues(t, 18*time.Millisecond, time.Duration(stats.OutboxFenceWaitNanoseconds))
	require.EqualValues(t, 1, stats.OutboxFenceTimeoutCount)
	require.EqualValues(t, 2, stats.OutboxBatchCount)
	require.EqualValues(t, 3, stats.OutboxBatchRowsTotal)
	require.Zero(t, stats.OutboxLastBatchRows)
	require.EqualValues(t, 17, stats.OutboxBacklogRows)
	require.EqualValues(t, 1, stats.OutboxBacklogSampleCount)
	require.EqualValues(t, 1, stats.ProducerTransactionRollbackCount)
}

func TestDeliveryRuntimeStatsClampsNegativeGaugesAndDurations(t *testing.T) {
	resetDeliveryRuntimeStatsForTest()

	ObserveMigrationLock(-time.Second, true, false, false)
	ObserveOutboxFence(-time.Second, false, false)
	ObserveOutboxBatch(-5)
	ObserveOutboxBacklog(-9)

	stats := SnapshotDeliveryRuntimeStats()
	require.Zero(t, stats.MigrationLockWaitNanoseconds)
	require.Zero(t, stats.OutboxFenceWaitNanoseconds)
	require.Zero(t, stats.OutboxLastBatchRows)
	require.Zero(t, stats.OutboxBacklogRows)
}

func resetDeliveryRuntimeStatsForTest() {
	deliveryRuntime.migrationLockAttemptCount.Store(0)
	deliveryRuntime.migrationLockAcquiredCount.Store(0)
	deliveryRuntime.migrationLockWaitCount.Store(0)
	deliveryRuntime.migrationLockWaitNanoseconds.Store(0)
	deliveryRuntime.migrationLockTimeoutCount.Store(0)
	deliveryRuntime.outboxFenceAttemptCount.Store(0)
	deliveryRuntime.outboxFenceAcquiredCount.Store(0)
	deliveryRuntime.outboxFenceWaitNanoseconds.Store(0)
	deliveryRuntime.outboxFenceTimeoutCount.Store(0)
	deliveryRuntime.outboxBatchCount.Store(0)
	deliveryRuntime.outboxBatchRowsTotal.Store(0)
	deliveryRuntime.outboxLastBatchRows.Store(0)
	deliveryRuntime.outboxBacklogRows.Store(0)
	deliveryRuntime.outboxBacklogSampleCount.Store(0)
	deliveryRuntime.producerTransactionRollbackCount.Store(0)
}
