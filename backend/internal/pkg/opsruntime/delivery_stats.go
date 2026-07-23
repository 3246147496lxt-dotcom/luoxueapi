package opsruntime

import (
	"sync/atomic"
	"time"
)

// DeliveryRuntimeStats contains process-local acceptance signals for the
// migration and scheduler-outbox delivery boundaries. It intentionally carries
// only counters, durations and gauges; no event payload or account identity is
// retained.
type DeliveryRuntimeStats struct {
	MigrationLockAttemptCount        uint64 `json:"migration_lock_attempt_count"`
	MigrationLockAcquiredCount       uint64 `json:"migration_lock_acquired_count"`
	MigrationLockWaitCount           uint64 `json:"migration_lock_wait_count"`
	MigrationLockWaitNanoseconds     uint64 `json:"migration_lock_wait_nanoseconds"`
	MigrationLockTimeoutCount        uint64 `json:"migration_lock_timeout_count"`
	OutboxFenceAttemptCount          uint64 `json:"outbox_fence_attempt_count"`
	OutboxFenceAcquiredCount         uint64 `json:"outbox_fence_acquired_count"`
	OutboxFenceWaitNanoseconds       uint64 `json:"outbox_fence_wait_nanoseconds"`
	OutboxFenceTimeoutCount          uint64 `json:"outbox_fence_timeout_count"`
	OutboxBatchCount                 uint64 `json:"outbox_batch_count"`
	OutboxBatchRowsTotal             uint64 `json:"outbox_batch_rows_total"`
	OutboxLastBatchRows              int64  `json:"outbox_last_batch_rows"`
	OutboxBacklogRows                int64  `json:"outbox_backlog_rows"`
	OutboxBacklogSampleCount         uint64 `json:"outbox_backlog_sample_count"`
	ProducerTransactionRollbackCount uint64 `json:"producer_transaction_rollback_count"`
}

type deliveryRuntimeCounters struct {
	migrationLockAttemptCount        atomic.Uint64
	migrationLockAcquiredCount       atomic.Uint64
	migrationLockWaitCount           atomic.Uint64
	migrationLockWaitNanoseconds     atomic.Uint64
	migrationLockTimeoutCount        atomic.Uint64
	outboxFenceAttemptCount          atomic.Uint64
	outboxFenceAcquiredCount         atomic.Uint64
	outboxFenceWaitNanoseconds       atomic.Uint64
	outboxFenceTimeoutCount          atomic.Uint64
	outboxBatchCount                 atomic.Uint64
	outboxBatchRowsTotal             atomic.Uint64
	outboxLastBatchRows              atomic.Int64
	outboxBacklogRows                atomic.Int64
	outboxBacklogSampleCount         atomic.Uint64
	producerTransactionRollbackCount atomic.Uint64
}

var deliveryRuntime deliveryRuntimeCounters

func ObserveMigrationLock(wait time.Duration, waited, timedOut, acquired bool) {
	deliveryRuntime.migrationLockAttemptCount.Add(1)
	if acquired {
		deliveryRuntime.migrationLockAcquiredCount.Add(1)
	}
	if waited {
		deliveryRuntime.migrationLockWaitCount.Add(1)
		deliveryRuntime.migrationLockWaitNanoseconds.Add(durationNanoseconds(wait))
	}
	if timedOut {
		deliveryRuntime.migrationLockTimeoutCount.Add(1)
	}
}

func ObserveOutboxFence(wait time.Duration, acquired, timedOut bool) {
	deliveryRuntime.outboxFenceAttemptCount.Add(1)
	deliveryRuntime.outboxFenceWaitNanoseconds.Add(durationNanoseconds(wait))
	if acquired {
		deliveryRuntime.outboxFenceAcquiredCount.Add(1)
	}
	if timedOut {
		deliveryRuntime.outboxFenceTimeoutCount.Add(1)
	}
}

func ObserveOutboxBatch(rows int) {
	if rows < 0 {
		rows = 0
	}
	deliveryRuntime.outboxBatchCount.Add(1)
	deliveryRuntime.outboxBatchRowsTotal.Add(uint64(rows))
	deliveryRuntime.outboxLastBatchRows.Store(int64(rows))
}

func ObserveOutboxBacklog(rows int64) {
	if rows < 0 {
		rows = 0
	}
	deliveryRuntime.outboxBacklogRows.Store(rows)
	deliveryRuntime.outboxBacklogSampleCount.Add(1)
}

func ObserveProducerTransactionRollback() {
	deliveryRuntime.producerTransactionRollbackCount.Add(1)
}

func SnapshotDeliveryRuntimeStats() DeliveryRuntimeStats {
	return DeliveryRuntimeStats{
		MigrationLockAttemptCount:        deliveryRuntime.migrationLockAttemptCount.Load(),
		MigrationLockAcquiredCount:       deliveryRuntime.migrationLockAcquiredCount.Load(),
		MigrationLockWaitCount:           deliveryRuntime.migrationLockWaitCount.Load(),
		MigrationLockWaitNanoseconds:     deliveryRuntime.migrationLockWaitNanoseconds.Load(),
		MigrationLockTimeoutCount:        deliveryRuntime.migrationLockTimeoutCount.Load(),
		OutboxFenceAttemptCount:          deliveryRuntime.outboxFenceAttemptCount.Load(),
		OutboxFenceAcquiredCount:         deliveryRuntime.outboxFenceAcquiredCount.Load(),
		OutboxFenceWaitNanoseconds:       deliveryRuntime.outboxFenceWaitNanoseconds.Load(),
		OutboxFenceTimeoutCount:          deliveryRuntime.outboxFenceTimeoutCount.Load(),
		OutboxBatchCount:                 deliveryRuntime.outboxBatchCount.Load(),
		OutboxBatchRowsTotal:             deliveryRuntime.outboxBatchRowsTotal.Load(),
		OutboxLastBatchRows:              deliveryRuntime.outboxLastBatchRows.Load(),
		OutboxBacklogRows:                deliveryRuntime.outboxBacklogRows.Load(),
		OutboxBacklogSampleCount:         deliveryRuntime.outboxBacklogSampleCount.Load(),
		ProducerTransactionRollbackCount: deliveryRuntime.producerTransactionRollbackCount.Load(),
	}
}

func durationNanoseconds(value time.Duration) uint64 {
	if value <= 0 {
		return 0
	}
	return uint64(value)
}
