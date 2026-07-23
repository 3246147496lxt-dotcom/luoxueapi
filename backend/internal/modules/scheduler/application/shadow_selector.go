package application

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/modules/scheduler/domain"
	"github.com/Wei-Shaw/sub2api/internal/modules/scheduler/ports"
)

// ShadowComparingSelector always returns the primary result. When enabled, it
// also runs the shadow selector and compares normalized candidate ID sets.
type ShadowComparingSelector struct {
	primary  ports.CandidateSelector
	shadow   ports.CandidateSelector
	observer ports.CandidateMismatchObserver
	enabled  bool
	// asyncSlots bounds production shadow reads. The authoritative selector is
	// never delayed when every slot is occupied; that comparison is skipped.
	asyncSlots chan struct{}
	workerMu   sync.Mutex
	workerCtx  context.Context
	cancel     context.CancelFunc
	workerWG   sync.WaitGroup
	started    bool
	stopping   bool
	stopOnce   sync.Once
	stopped    chan struct{}

	comparisons atomic.Uint64
	mismatches  atomic.Uint64
	errors      atomic.Uint64
	statusMu    sync.RWMutex
	last        domain.RuntimeStatus
}

const (
	// Repository-backed shadow selection hydrates complete account records and
	// their group/proxy associations today. Keep its process-local concurrency
	// deliberately small until the port is replaced by a narrow ID-only query.
	defaultShadowComparisonConcurrency = 2
	defaultShadowComparisonTimeout     = 2 * time.Second
)

func NewShadowComparingSelector(primary, shadow ports.CandidateSelector, observer ports.CandidateMismatchObserver, enabled bool) *ShadowComparingSelector {
	return &ShadowComparingSelector{
		primary:    primary,
		shadow:     shadow,
		observer:   observer,
		enabled:    enabled,
		asyncSlots: make(chan struct{}, defaultShadowComparisonConcurrency),
		stopped:    make(chan struct{}),
	}
}

// Start enables asynchronous shadow observations. It is intentionally
// separate from construction so the process Supervisor owns the worker
// lifetime. Calling Start more than once before Stop is harmless.
func (s *ShadowComparingSelector) Start(ctx context.Context) error {
	if s == nil || !s.enabled || s.shadow == nil {
		return nil
	}
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}

	s.workerMu.Lock()
	defer s.workerMu.Unlock()
	if s.stopping {
		return context.Canceled
	}
	if s.started {
		return nil
	}
	s.workerCtx, s.cancel = context.WithCancel(context.Background())
	s.started = true
	return nil
}

// Stop cancels all in-flight comparisons and waits for them to finish. It is
// safe to call repeatedly; a caller with a shorter deadline may time out while
// a later call still observes the same eventual drain completion.
func (s *ShadowComparingSelector) Stop(ctx context.Context) error {
	if s == nil {
		return nil
	}
	s.stopOnce.Do(func() {
		s.workerMu.Lock()
		s.stopping = true
		if s.cancel != nil {
			s.cancel()
		}
		s.workerMu.Unlock()

		go func() {
			s.workerWG.Wait()
			close(s.stopped)
		}()
	})
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-s.stopped:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *ShadowComparingSelector) SelectCandidates(ctx context.Context, query domain.CandidateQuery) (domain.CandidateSet, error) {
	if s == nil || s.primary == nil {
		return domain.CandidateSet{}, ErrPortUnavailable
	}
	primary, err := s.primary.SelectCandidates(ctx, query)
	if err != nil || !s.enabled || s.shadow == nil {
		return primary, err
	}

	s.compare(ctx, query, primary)
	return primary, nil
}

// ObservePrimaryAsync compares an already-selected authoritative candidate set
// in a detached, bounded goroutine. It is the production gateway path: callers
// return the legacy result immediately, and shadow saturation or failure cannot
// affect routing. The boolean reports whether a comparison was scheduled.
func (s *ShadowComparingSelector) ObservePrimaryAsync(_ context.Context, query domain.CandidateQuery, primary domain.CandidateSet) bool {
	if s == nil || !s.enabled || s.shadow == nil {
		return false
	}
	s.workerMu.Lock()
	if !s.started || s.stopping || s.workerCtx == nil {
		s.workerMu.Unlock()
		return false
	}
	select {
	case s.asyncSlots <- struct{}{}:
	default:
		s.workerMu.Unlock()
		return false
	}
	workerCtx := s.workerCtx
	s.workerWG.Add(1)
	s.workerMu.Unlock()

	query = cloneCandidateQuery(query)
	primary.AccountIDs = append([]int64(nil), primary.AccountIDs...)
	go func() {
		defer s.workerWG.Done()
		defer func() { <-s.asyncSlots }()
		shadowCtx, cancel := context.WithTimeout(workerCtx, defaultShadowComparisonTimeout)
		defer cancel()
		s.compare(shadowCtx, query, primary)
	}()
	return true
}

func (s *ShadowComparingSelector) compare(ctx context.Context, query domain.CandidateQuery, primary domain.CandidateSet) {
	s.comparisons.Add(1)
	shadow, shadowErr := s.shadow.SelectCandidates(ctx, query)
	if shadowErr != nil {
		// Supervisor shutdown cancels the worker context deliberately. Do not
		// surface that expected drain signal as a comparison failure or warning.
		if errors.Is(ctx.Err(), context.Canceled) {
			return
		}
		s.errors.Add(1)
		mismatch := domain.CandidateMismatch{
			Query:             query,
			PrimaryAccountIDs: append([]int64(nil), primary.AccountIDs...),
			PrimaryMixed:      primary.MixedScheduling,
			ShadowError:       shadowErr.Error(),
		}
		s.record(mismatch, false, shadowErr.Error())
		s.observe(ctx, mismatch)
		return
	}

	mismatch, different := domain.CompareCandidateSets(query, primary, shadow)
	if !different {
		s.record(mismatch, false, "")
		return
	}
	s.mismatches.Add(1)
	s.record(mismatch, true, "")
	s.observe(ctx, mismatch)
}

func cloneCandidateQuery(query domain.CandidateQuery) domain.CandidateQuery {
	if query.GroupID == nil {
		return query
	}
	groupID := *query.GroupID
	query.GroupID = &groupID
	return query
}

func (s *ShadowComparingSelector) observe(ctx context.Context, mismatch domain.CandidateMismatch) {
	if s.observer != nil {
		s.observer.ObserveCandidateMismatch(ctx, mismatch)
	}
}

func (s *ShadowComparingSelector) record(_ domain.CandidateMismatch, mismatch bool, comparisonErr string) {
	s.statusMu.Lock()
	defer s.statusMu.Unlock()
	if mismatch {
		s.last.LastShadowMismatchAt = time.Now().UTC()
	}
	s.last.LastShadowComparisonError = comparisonErr
}

func (s *ShadowComparingSelector) ShadowComparisonStatus() domain.RuntimeStatus {
	if s == nil {
		return domain.RuntimeStatus{}
	}
	s.statusMu.RLock()
	status := s.last
	s.statusMu.RUnlock()
	status.ShadowComparisons = s.comparisons.Load()
	status.ShadowMismatches = s.mismatches.Load()
	status.ShadowComparisonErrors = s.errors.Load()
	return status
}
