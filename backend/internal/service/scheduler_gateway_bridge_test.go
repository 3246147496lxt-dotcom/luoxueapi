package service

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	schedulerdomain "github.com/Wei-Shaw/sub2api/internal/modules/scheduler/domain"
	"github.com/stretchr/testify/require"
)

type schedulerBridgeCacheStub struct {
	SchedulerCache
	accounts []Account
}

func (s *schedulerBridgeCacheStub) GetSnapshot(context.Context, SchedulerBucket) ([]*Account, bool, error) {
	copies := append([]Account(nil), s.accounts...)
	accounts := make([]*Account, len(copies))
	for i := range copies {
		accounts[i] = &copies[i]
	}
	return accounts, true, nil
}

type schedulerBridgeShadowSelectorStub struct {
	result schedulerdomain.CandidateSet
	calls  atomic.Int32
}

func (s *schedulerBridgeShadowSelectorStub) SelectCandidates(context.Context, schedulerdomain.CandidateQuery) (schedulerdomain.CandidateSet, error) {
	s.calls.Add(1)
	return s.result, nil
}

type schedulerBridgeMismatchObserverStub struct {
	observed chan schedulerdomain.CandidateMismatch
}

func (s *schedulerBridgeMismatchObserverStub) ObserveCandidateMismatch(_ context.Context, mismatch schedulerdomain.CandidateMismatch) {
	s.observed <- mismatch
}

func TestSchedulerGatewayBridgeKeepsLegacyAuthoritativeAndThrottlesShadow(t *testing.T) {
	cache := &schedulerBridgeCacheStub{accounts: []Account{
		{ID: 7, Platform: PlatformOpenAI},
		{ID: 3, Platform: PlatformOpenAI},
	}}
	legacy := NewSchedulerSnapshotService(cache, nil, nil, nil, &config.Config{})
	shadow := &schedulerBridgeShadowSelectorStub{result: schedulerdomain.CandidateSet{AccountIDs: []int64{3}}}
	observer := &schedulerBridgeMismatchObserverStub{observed: make(chan schedulerdomain.CandidateMismatch, 1)}
	module := NewSchedulerModuleFacade(legacy, shadow, true, observer)
	require.NoError(t, module.StartShadow(context.Background()))
	t.Cleanup(func() { require.NoError(t, module.StopShadow(context.Background())) })
	bridge := newSchedulerGatewayBridge(legacy, module)

	accounts, mixed, err := bridge.ListSchedulableAccounts(context.Background(), nil, PlatformOpenAI, false)
	require.NoError(t, err)
	require.False(t, mixed)
	require.Equal(t, []int64{7, 3}, []int64{accounts[0].ID, accounts[1].ID})

	select {
	case mismatch := <-observer.observed:
		require.Equal(t, []int64{3, 7}, mismatch.PrimaryAccountIDs)
		require.Equal(t, []int64{3}, mismatch.ShadowAccountIDs)
	case <-time.After(time.Second):
		t.Fatal("production bridge did not schedule the shadow comparison")
	}

	_, _, err = bridge.ListSchedulableAccounts(context.Background(), nil, PlatformOpenAI, false)
	require.NoError(t, err)
	require.Never(t, func() bool { return shadow.calls.Load() > 1 }, 100*time.Millisecond, 10*time.Millisecond)
}

func TestSchedulerGatewayBridgeShadowCanBeDisabledWithoutChangingLegacy(t *testing.T) {
	cache := &schedulerBridgeCacheStub{accounts: []Account{{ID: 9, Platform: PlatformOpenAI}}}
	legacy := NewSchedulerSnapshotService(cache, nil, nil, nil, &config.Config{})
	shadow := &schedulerBridgeShadowSelectorStub{result: schedulerdomain.CandidateSet{AccountIDs: []int64{99}}}
	module := NewSchedulerModuleFacade(legacy, shadow, false, nil)
	require.NoError(t, module.StartShadow(context.Background()))
	t.Cleanup(func() { require.NoError(t, module.StopShadow(context.Background())) })
	bridge := newSchedulerGatewayBridge(legacy, module)

	accounts, mixed, err := bridge.ListSchedulableAccounts(context.Background(), nil, PlatformOpenAI, false)
	require.NoError(t, err)
	require.False(t, mixed)
	require.Equal(t, []int64{9}, []int64{accounts[0].ID})
	require.Zero(t, shadow.calls.Load())
}

type schedulerCandidateRepositoryStub struct {
	call      string
	groupID   int64
	platforms []string
	accounts  []Account
}

func (s *schedulerCandidateRepositoryStub) result(call string, groupID int64, platforms []string) ([]Account, error) {
	s.call = call
	s.groupID = groupID
	s.platforms = append([]string(nil), platforms...)
	return append([]Account(nil), s.accounts...), nil
}

func (s *schedulerCandidateRepositoryStub) ListSchedulableByPlatform(_ context.Context, platform string) ([]Account, error) {
	return s.result("platform", 0, []string{platform})
}

func (s *schedulerCandidateRepositoryStub) ListSchedulableByGroupIDAndPlatform(_ context.Context, groupID int64, platform string) ([]Account, error) {
	return s.result("group-platform", groupID, []string{platform})
}

func (s *schedulerCandidateRepositoryStub) ListSchedulableByPlatforms(_ context.Context, platforms []string) ([]Account, error) {
	return s.result("platforms", 0, platforms)
}

func (s *schedulerCandidateRepositoryStub) ListSchedulableByGroupIDAndPlatforms(_ context.Context, groupID int64, platforms []string) ([]Account, error) {
	return s.result("group-platforms", groupID, platforms)
}

func (s *schedulerCandidateRepositoryStub) ListSchedulableUngroupedByPlatform(_ context.Context, platform string) ([]Account, error) {
	return s.result("ungrouped-platform", 0, []string{platform})
}

func (s *schedulerCandidateRepositoryStub) ListSchedulableUngroupedByPlatforms(_ context.Context, platforms []string) ([]Account, error) {
	return s.result("ungrouped-platforms", 0, platforms)
}

func TestSchedulerRepositoryCandidateSelectorMatchesMixedGroupSemantics(t *testing.T) {
	repo := &schedulerCandidateRepositoryStub{accounts: []Account{
		{ID: 1, Platform: PlatformAnthropic},
		{ID: 2, Platform: PlatformAntigravity, Extra: map[string]any{"mixed_scheduling": true}},
		{ID: 3, Platform: PlatformAntigravity, Extra: map[string]any{"mixed_scheduling": false}},
	}}
	selector := newSchedulerRepositoryCandidateSelector(repo, &config.Config{})
	groupID := int64(42)

	result, err := selector.SelectCandidates(context.Background(), schedulerdomain.CandidateQuery{
		GroupID:  &groupID,
		Platform: PlatformAnthropic,
	})

	require.NoError(t, err)
	require.Equal(t, "group-platforms", repo.call)
	require.EqualValues(t, 42, repo.groupID)
	require.Equal(t, []string{PlatformAnthropic, PlatformAntigravity}, repo.platforms)
	require.Equal(t, []int64{1, 2}, result.AccountIDs)
	require.True(t, result.MixedScheduling)
}

func TestSchedulerRepositoryCandidateSelectorIgnoresGroupInSimpleMode(t *testing.T) {
	repo := &schedulerCandidateRepositoryStub{accounts: []Account{{ID: 5, Platform: PlatformOpenAI}}}
	cfg := &config.Config{RunMode: config.RunModeSimple}
	selector := newSchedulerRepositoryCandidateSelector(repo, cfg)
	groupID := int64(42)

	result, err := selector.SelectCandidates(context.Background(), schedulerdomain.CandidateQuery{
		GroupID:  &groupID,
		Platform: PlatformOpenAI,
	})

	require.NoError(t, err)
	require.Equal(t, "platform", repo.call)
	require.Zero(t, repo.groupID)
	require.Equal(t, []int64{5}, result.AccountIDs)
	require.False(t, result.MixedScheduling)
}

func TestSchedulerShadowIDLogSummaryIsBoundedAndStable(t *testing.T) {
	ids := make([]int64, 0, schedulerShadowLogIDLimit+5)
	for id := int64(schedulerShadowLogIDLimit + 5); id > 0; id-- {
		ids = append(ids, id)
	}
	ids = append(ids, 1, 0, -1)

	sample, count, digest := schedulerShadowIDLogSummary(ids)
	require.Len(t, sample, schedulerShadowLogIDLimit)
	require.Equal(t, schedulerShadowLogIDLimit+5, count)
	require.Equal(t, int64(1), sample[0])
	require.Equal(t, int64(schedulerShadowLogIDLimit), sample[len(sample)-1])
	require.Len(t, digest, 16)

	reordered := append([]int64(nil), ids...)
	for left, right := 0, len(reordered)-1; left < right; left, right = left+1, right-1 {
		reordered[left], reordered[right] = reordered[right], reordered[left]
	}
	_, reorderedCount, reorderedDigest := schedulerShadowIDLogSummary(reordered)
	require.Equal(t, count, reorderedCount)
	require.Equal(t, digest, reorderedDigest)
}
