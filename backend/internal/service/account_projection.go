package service

import "context"

// AccountProjectionWriter is the optional low-latency write side of the
// scheduler's account projection. The scheduler outbox remains the durable
// correctness source; implementations of this port are only a best-effort
// visibility optimization after the owning database transaction commits.
type AccountProjectionWriter interface {
	SetAccount(ctx context.Context, account *Account) error
	// DeleteAccount establishes a durable deletion fence in addition to removing
	// cached payloads. Account IDs are sequence-generated and must never be reused.
	DeleteAccount(ctx context.Context, accountID int64) error
}
