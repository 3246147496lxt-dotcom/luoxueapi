package service

import (
	"context"
	"time"
)

type SchedulerOutboxEvent struct {
	ID        int64
	EventType string
	AccountID *int64
	GroupID   *int64
	Payload   map[string]any
	CreatedAt time.Time
}

// SchedulerOutboxRepository 提供调度 outbox 的读取接口。
type SchedulerOutboxRepository interface {
	// TryAcquireConsumerLease serializes the complete outbox consume cycle across
	// application instances. The session-scoped lease must remain held until the
	// selected batch is committed, applied, and its Redis watermark is advanced.
	TryAcquireConsumerLease(ctx context.Context) (SchedulerOutboxConsumerLease, bool, error)
	ListAfterAndReleaseDedup(ctx context.Context, afterID int64, limit int) ([]SchedulerOutboxEvent, error)
	// FirstCreatedAtAfter 返回指定水位之后第一条待消费事件的创建时间，不领取事件或修改去重键。
	FirstCreatedAtAfter(ctx context.Context, afterID int64) (time.Time, bool, error)
	MaxID(ctx context.Context) (int64, error)
	DeleteConsumedUpTo(ctx context.Context, watermark int64, limit int) (int64, error)
	TryAcquireCleanupLock(ctx context.Context) (SchedulerOutboxCleanupLease, bool, error)
}

// SchedulerOutboxConsumerLease holds the PostgreSQL session advisory lock used
// to serialize outbox consumption across application instances.
type SchedulerOutboxConsumerLease interface {
	// ListAfterAndReleaseDedup executes on the same fixed PostgreSQL session
	// that owns the advisory lock, avoiding pool self-deadlock at max_open_conns=1.
	ListAfterAndReleaseDedup(ctx context.Context, afterID int64, limit int) ([]SchedulerOutboxEvent, error)
	// BindContext routes repository reads performed while applying an event to
	// the same fixed PostgreSQL session. The returned context is valid only
	// until Release; implementations must fail closed afterwards.
	BindContext(ctx context.Context) (context.Context, error)
	Release() error
}

// SchedulerOutboxCleanupLease holds the PostgreSQL advisory lock used by
// scheduler outbox cleanup.
type SchedulerOutboxCleanupLease interface {
	// DeleteConsumedUpTo executes on the lease-owning session for the same
	// single-connection safety as the consumer path.
	DeleteConsumedUpTo(ctx context.Context, watermark int64, limit int) (int64, error)
	Release() error
}
