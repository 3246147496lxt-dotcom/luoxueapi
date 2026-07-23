package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/modules/scheduler/domain"
	"github.com/stretchr/testify/require"
)

type selectorStub struct {
	result domain.CandidateSet
	err    error
	calls  int
}

func (s *selectorStub) SelectCandidates(context.Context, domain.CandidateQuery) (domain.CandidateSet, error) {
	s.calls++
	return s.result, s.err
}

type mismatchObserverStub struct {
	mismatches []domain.CandidateMismatch
}

func (s *mismatchObserverStub) ObserveCandidateMismatch(_ context.Context, mismatch domain.CandidateMismatch) {
	s.mismatches = append(s.mismatches, mismatch)
}

func TestShadowSelectorDisabledHasNoShadowBehavior(t *testing.T) {
	primary := &selectorStub{result: domain.CandidateSet{AccountIDs: []int64{1, 2}}}
	shadow := &selectorStub{result: domain.CandidateSet{AccountIDs: []int64{3}}}
	selector := NewShadowComparingSelector(primary, shadow, nil, false)

	result, err := selector.SelectCandidates(context.Background(), domain.CandidateQuery{Platform: "openai"})

	require.NoError(t, err)
	require.Equal(t, []int64{1, 2}, result.AccountIDs)
	require.Zero(t, shadow.calls)
	require.Zero(t, selector.ShadowComparisonStatus().ShadowComparisons)
}

func TestShadowSelectorReportsMismatchWithoutChangingPrimaryResult(t *testing.T) {
	primary := &selectorStub{result: domain.CandidateSet{AccountIDs: []int64{8, 3}}}
	shadow := &selectorStub{result: domain.CandidateSet{AccountIDs: []int64{3, 5}}}
	observer := &mismatchObserverStub{}
	selector := NewShadowComparingSelector(primary, shadow, observer, true)

	result, err := selector.SelectCandidates(context.Background(), domain.CandidateQuery{Platform: "openai"})

	require.NoError(t, err)
	require.Equal(t, []int64{8, 3}, result.AccountIDs)
	require.Len(t, observer.mismatches, 1)
	require.Equal(t, []int64{8}, observer.mismatches[0].MissingFromShadow)
	require.Equal(t, []int64{5}, observer.mismatches[0].UnexpectedInShadow)
	status := selector.ShadowComparisonStatus()
	require.EqualValues(t, 1, status.ShadowComparisons)
	require.EqualValues(t, 1, status.ShadowMismatches)
	require.False(t, status.LastShadowMismatchAt.IsZero())
}

func TestShadowSelectorObservesShadowErrorAndReturnsPrimary(t *testing.T) {
	primary := &selectorStub{result: domain.CandidateSet{AccountIDs: []int64{4}}}
	shadow := &selectorStub{err: errors.New("shadow unavailable")}
	observer := &mismatchObserverStub{}
	selector := NewShadowComparingSelector(primary, shadow, observer, true)

	result, err := selector.SelectCandidates(context.Background(), domain.CandidateQuery{})

	require.NoError(t, err)
	require.Equal(t, []int64{4}, result.AccountIDs)
	require.EqualValues(t, 1, selector.ShadowComparisonStatus().ShadowComparisonErrors)
	require.Equal(t, "shadow unavailable", observer.mismatches[0].ShadowError)
}

type asyncSelectorStub struct {
	started chan struct{}
	release chan struct{}
	result  domain.CandidateSet
}

func (s *asyncSelectorStub) SelectCandidates(ctx context.Context, _ domain.CandidateQuery) (domain.CandidateSet, error) {
	select {
	case s.started <- struct{}{}:
	default:
	}
	select {
	case <-s.release:
		return s.result, nil
	case <-ctx.Done():
		return domain.CandidateSet{}, ctx.Err()
	}
}

type asyncMismatchObserverStub struct {
	observed chan domain.CandidateMismatch
}

func (s *asyncMismatchObserverStub) ObserveCandidateMismatch(_ context.Context, mismatch domain.CandidateMismatch) {
	s.observed <- mismatch
}

func TestShadowSelectorObservesSuppliedPrimaryAsynchronously(t *testing.T) {
	primary := &selectorStub{result: domain.CandidateSet{AccountIDs: []int64{99}}}
	shadow := &asyncSelectorStub{
		started: make(chan struct{}, 1),
		release: make(chan struct{}),
		result:  domain.CandidateSet{AccountIDs: []int64{2}},
	}
	observer := &asyncMismatchObserverStub{observed: make(chan domain.CandidateMismatch, 1)}
	selector := NewShadowComparingSelector(primary, shadow, observer, true)
	require.NoError(t, selector.Start(context.Background()))
	t.Cleanup(func() { require.NoError(t, selector.Stop(context.Background())) })
	groupID := int64(7)

	scheduled := selector.ObservePrimaryAsync(
		context.Background(),
		domain.CandidateQuery{GroupID: &groupID, Platform: "openai"},
		domain.CandidateSet{AccountIDs: []int64{1}},
	)
	require.True(t, scheduled)
	require.Zero(t, primary.calls, "the supplied authoritative result must not be selected again")

	select {
	case <-shadow.started:
	case <-time.After(time.Second):
		t.Fatal("shadow selector did not start")
	}
	close(shadow.release)

	select {
	case mismatch := <-observer.observed:
		require.Equal(t, []int64{1}, mismatch.PrimaryAccountIDs)
		require.Equal(t, []int64{2}, mismatch.ShadowAccountIDs)
		require.NotNil(t, mismatch.Query.GroupID)
		require.EqualValues(t, 7, *mismatch.Query.GroupID)
	case <-time.After(time.Second):
		t.Fatal("shadow mismatch was not observed")
	}
	require.EqualValues(t, 1, selector.ShadowComparisonStatus().ShadowComparisons)
}

func TestShadowSelectorStopCancelsWaitsAndIsIdempotent(t *testing.T) {
	shadow := &asyncSelectorStub{
		started: make(chan struct{}, 1),
		release: make(chan struct{}),
	}
	observer := &asyncMismatchObserverStub{observed: make(chan domain.CandidateMismatch, 1)}
	selector := NewShadowComparingSelector(&selectorStub{}, shadow, observer, true)
	require.NoError(t, selector.Start(context.Background()))
	require.True(t, selector.ObservePrimaryAsync(
		context.Background(),
		domain.CandidateQuery{Platform: "openai"},
		domain.CandidateSet{AccountIDs: []int64{1}},
	))

	select {
	case <-shadow.started:
	case <-time.After(time.Second):
		t.Fatal("shadow selector did not start")
	}

	stopCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, selector.Stop(stopCtx))
	require.NoError(t, selector.Stop(context.Background()))
	require.False(t, selector.ObservePrimaryAsync(
		context.Background(),
		domain.CandidateQuery{Platform: "openai"},
		domain.CandidateSet{AccountIDs: []int64{2}},
	))
}
