package ports

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/modules/scheduler/domain"
)

type CandidateSelector interface {
	SelectCandidates(ctx context.Context, query domain.CandidateQuery) (domain.CandidateSet, error)
}

type AccountSnapshotReader interface {
	ReadAccountSnapshot(ctx context.Context, accountID int64) (domain.AccountSnapshot, bool, error)
}

type Rebuilder interface {
	Rebuild(ctx context.Context, reason string) error
}

type OutboxConsumer interface {
	ConsumeOutbox(ctx context.Context, limit int) (domain.OutboxConsumeResult, error)
}

type RuntimeStatusReader interface {
	RuntimeStatus(ctx context.Context) domain.RuntimeStatus
}

type CandidateMismatchObserver interface {
	ObserveCandidateMismatch(ctx context.Context, mismatch domain.CandidateMismatch)
}
