package application

import (
	"context"
	"errors"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/modules/scheduler/domain"
	"github.com/Wei-Shaw/sub2api/internal/modules/scheduler/ports"
)

var ErrPortUnavailable = errors.New("scheduler application port is unavailable")

type Dependencies struct {
	Selector       ports.CandidateSelector
	Snapshots      ports.AccountSnapshotReader
	Rebuilder      ports.Rebuilder
	OutboxConsumer ports.OutboxConsumer
	Runtime        ports.RuntimeStatusReader
}

// Facade is the application boundary used while legacy scheduler mechanics are
// incrementally moved behind module ports.
type Facade struct {
	selector       ports.CandidateSelector
	snapshots      ports.AccountSnapshotReader
	rebuilder      ports.Rebuilder
	outboxConsumer ports.OutboxConsumer
	runtime        ports.RuntimeStatusReader
}

func NewFacade(deps Dependencies) *Facade {
	return &Facade{
		selector:       deps.Selector,
		snapshots:      deps.Snapshots,
		rebuilder:      deps.Rebuilder,
		outboxConsumer: deps.OutboxConsumer,
		runtime:        deps.Runtime,
	}
}

func (f *Facade) SelectCandidates(ctx context.Context, query domain.CandidateQuery) (domain.CandidateSet, error) {
	if f == nil || f.selector == nil {
		return domain.CandidateSet{}, ErrPortUnavailable
	}
	query.Platform = strings.TrimSpace(query.Platform)
	return f.selector.SelectCandidates(ctx, query)
}

// ObservePrimaryCandidates schedules a shadow comparison for a candidate set
// already selected by the authoritative gateway path. It deliberately exposes
// no result capable of changing routing; false only means the selector is not a
// shadow comparator or its bounded comparison queue is currently full.
func (f *Facade) ObservePrimaryCandidates(ctx context.Context, query domain.CandidateQuery, primary domain.CandidateSet) bool {
	if f == nil || f.selector == nil {
		return false
	}
	observer, ok := f.selector.(interface {
		ObservePrimaryAsync(context.Context, domain.CandidateQuery, domain.CandidateSet) bool
	})
	if !ok {
		return false
	}
	return observer.ObservePrimaryAsync(ctx, query, primary)
}

// StartShadow starts the optional asynchronous shadow comparator. Facades
// without a shadow selector remain valid and require no lifecycle work.
func (f *Facade) StartShadow(ctx context.Context) error {
	if f == nil || f.selector == nil {
		return nil
	}
	runtime, ok := f.selector.(interface {
		Start(context.Context) error
	})
	if !ok {
		return nil
	}
	return runtime.Start(ctx)
}

// StopShadow cancels and drains the optional asynchronous shadow comparator.
// The underlying worker guarantees idempotency.
func (f *Facade) StopShadow(ctx context.Context) error {
	if f == nil || f.selector == nil {
		return nil
	}
	runtime, ok := f.selector.(interface {
		Stop(context.Context) error
	})
	if !ok {
		return nil
	}
	return runtime.Stop(ctx)
}

func (f *Facade) ReadAccountSnapshot(ctx context.Context, accountID int64) (domain.AccountSnapshot, bool, error) {
	if f == nil || f.snapshots == nil {
		return domain.AccountSnapshot{}, false, ErrPortUnavailable
	}
	if accountID <= 0 {
		return domain.AccountSnapshot{}, false, nil
	}
	return f.snapshots.ReadAccountSnapshot(ctx, accountID)
}

func (f *Facade) Rebuild(ctx context.Context, reason string) error {
	if f == nil || f.rebuilder == nil {
		return ErrPortUnavailable
	}
	return f.rebuilder.Rebuild(ctx, strings.TrimSpace(reason))
}

func (f *Facade) ConsumeOutbox(ctx context.Context, limit int) (domain.OutboxConsumeResult, error) {
	if f == nil || f.outboxConsumer == nil {
		return domain.OutboxConsumeResult{}, ErrPortUnavailable
	}
	if limit < 0 {
		limit = 0
	}
	return f.outboxConsumer.ConsumeOutbox(ctx, limit)
}

func (f *Facade) RuntimeStatus(ctx context.Context) domain.RuntimeStatus {
	if f == nil || f.runtime == nil {
		return domain.RuntimeStatus{}
	}
	status := f.runtime.RuntimeStatus(ctx)
	if stats, ok := f.selector.(interface{ ShadowComparisonStatus() domain.RuntimeStatus }); ok {
		shadow := stats.ShadowComparisonStatus()
		status.ShadowComparisons = shadow.ShadowComparisons
		status.ShadowMismatches = shadow.ShadowMismatches
		status.ShadowComparisonErrors = shadow.ShadowComparisonErrors
		status.LastShadowMismatchAt = shadow.LastShadowMismatchAt
		status.LastShadowComparisonError = shadow.LastShadowComparisonError
	}
	return status
}
