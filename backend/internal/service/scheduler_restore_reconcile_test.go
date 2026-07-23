package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type schedulerRestoreTrace struct {
	mu     sync.Mutex
	events []string
}

func (t *schedulerRestoreTrace) add(event string) {
	t.mu.Lock()
	t.events = append(t.events, event)
	t.mu.Unlock()
}

func (t *schedulerRestoreTrace) snapshot() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]string(nil), t.events...)
}

type schedulerRestoreCache struct {
	SchedulerCache
	trace       *schedulerRestoreTrace
	resetErr    error
	resetBusy   int
	resetHook   func()
	listErr     error
	snapshotCnt int
	lockBusy    int
	lockCalls   int
}

func (c *schedulerRestoreCache) ResetForAuthoritativeRebuild(context.Context) error {
	c.trace.add("reset")
	if c.resetBusy > 0 {
		c.resetBusy--
		return ErrSchedulerCacheResetInProgress
	}
	if c.resetHook != nil {
		c.resetHook()
	}
	return c.resetErr
}

func (c *schedulerRestoreCache) ListBuckets(context.Context) ([]SchedulerBucket, error) {
	c.trace.add("rebuild")
	return nil, c.listErr
}

func (c *schedulerRestoreCache) CaptureBucketWriteToken(_ context.Context, bucket SchedulerBucket) (SchedulerBucketWriteToken, error) {
	return SchedulerBucketWriteToken{Bucket: bucket, Generation: "restore", Epoch: 1}, nil
}

func (c *schedulerRestoreCache) TryLockBucket(context.Context, SchedulerBucket, time.Duration) (bool, error) {
	c.lockCalls++
	if c.lockBusy > 0 {
		c.lockBusy--
		return false, nil
	}
	return true, nil
}

func (c *schedulerRestoreCache) UnlockBucket(context.Context, SchedulerBucket) error { return nil }

func (c *schedulerRestoreCache) SetSnapshot(context.Context, SchedulerBucket, SchedulerBucketWriteToken, []Account) error {
	c.snapshotCnt++
	return nil
}

type schedulerRestoreAccountRepo struct{ AccountRepository }

func (schedulerRestoreAccountRepo) ListSchedulableByPlatform(context.Context, string) ([]Account, error) {
	return nil, nil
}

func (schedulerRestoreAccountRepo) ListSchedulableByPlatforms(context.Context, []string) ([]Account, error) {
	return nil, nil
}

type schedulerRecoveryCache struct {
	*schedulerRestoreCache
	recoverCalls int
	recovered    bool
	recoverErr   error
}

func (c *schedulerRecoveryCache) RecoverInterruptedAuthoritativeReset(context.Context) (bool, error) {
	c.recoverCalls++
	if c.recoverErr != nil {
		return false, c.recoverErr
	}
	recovered := c.recovered
	c.recovered = false
	return recovered, nil
}

type schedulerRestoreLease struct {
	trace *schedulerRestoreTrace
}

func (l *schedulerRestoreLease) ListAfterAndReleaseDedup(context.Context, int64, int) ([]SchedulerOutboxEvent, error) {
	return nil, nil
}

func (l *schedulerRestoreLease) BindContext(ctx context.Context) (context.Context, error) {
	return ctx, nil
}

func (l *schedulerRestoreLease) Release() error {
	l.trace.add("release")
	return nil
}

type schedulerRestoreOutboxRepo struct {
	SchedulerOutboxRepository
	trace *schedulerRestoreTrace
	lease *schedulerRestoreLease
}

func (r *schedulerRestoreOutboxRepo) TryAcquireConsumerLease(context.Context) (SchedulerOutboxConsumerLease, bool, error) {
	r.trace.add("acquire")
	return r.lease, true, nil
}

func TestSchedulerRestoreReconcileFencesThenRebuilds(t *testing.T) {
	trace := &schedulerRestoreTrace{}
	cache := &schedulerRestoreCache{trace: trace}
	lease := &schedulerRestoreLease{trace: trace}
	outbox := &schedulerRestoreOutboxRepo{trace: trace, lease: lease}
	svc := NewSchedulerSnapshotService(
		cache,
		outbox,
		schedulerRestoreAccountRepo{},
		nil,
		&config.Config{RunMode: config.RunModeSimple},
	)

	require.NoError(t, svc.ReconcileAfterDatabaseRestore(context.Background()))
	require.Equal(t, []string{"acquire", "reset", "release", "rebuild"}, trace.snapshot())
	require.Equal(t, len(schedulerCanonicalBuckets(0)), cache.snapshotCnt)
	status := svc.InitialSnapshotStatus()
	require.True(t, status.Done)
	require.NoError(t, status.Err)
}

func TestSchedulerRestoreReconcileInvalidatesBeforeCapabilityCheck(t *testing.T) {
	type cacheWithoutReset struct{ SchedulerCache }
	svc := &SchedulerSnapshotService{cache: cacheWithoutReset{}}
	svc.recordAuthoritativeRebuild(1, nil)
	require.True(t, svc.InitialSnapshotStatus().Done)

	err := svc.ReconcileAfterDatabaseRestore(context.Background())
	require.ErrorContains(t, err, "does not support authoritative reset")
	require.False(t, svc.InitialSnapshotStatus().Done)
}

func TestSchedulerRestoreReconcileRetriesStrictBucketLeaseBusy(t *testing.T) {
	trace := &schedulerRestoreTrace{}
	cache := &schedulerRestoreCache{trace: trace, lockBusy: 1}
	lease := &schedulerRestoreLease{trace: trace}
	outbox := &schedulerRestoreOutboxRepo{trace: trace, lease: lease}
	svc := NewSchedulerSnapshotService(cache, outbox, schedulerRestoreAccountRepo{}, nil, &config.Config{RunMode: config.RunModeSimple})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	require.NoError(t, svc.ReconcileAfterDatabaseRestore(ctx))
	require.Greater(t, cache.lockCalls, len(schedulerCanonicalBuckets(0)))
	status := svc.InitialSnapshotStatus()
	require.True(t, status.Done)
	require.NoError(t, status.Err)
}

func TestSchedulerRestoreReconcileRetriesInterruptedResetOwner(t *testing.T) {
	trace := &schedulerRestoreTrace{}
	cache := &schedulerRestoreCache{trace: trace, resetBusy: 1}
	lease := &schedulerRestoreLease{trace: trace}
	outbox := &schedulerRestoreOutboxRepo{trace: trace, lease: lease}
	svc := NewSchedulerSnapshotService(cache, outbox, schedulerRestoreAccountRepo{}, nil, &config.Config{RunMode: config.RunModeSimple})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	require.NoError(t, svc.ReconcileAfterDatabaseRestore(ctx))
	require.Equal(t, []string{"acquire", "reset", "reset", "release", "rebuild"}, trace.snapshot())
}

func TestSchedulerFullRebuildRecoversInterruptedResetBeforePublishingReady(t *testing.T) {
	trace := &schedulerRestoreTrace{}
	base := &schedulerRestoreCache{trace: trace, lockBusy: 1}
	cache := &schedulerRecoveryCache{schedulerRestoreCache: base, recovered: true}
	svc := NewSchedulerSnapshotService(cache, nil, schedulerRestoreAccountRepo{}, nil, &config.Config{RunMode: config.RunModeSimple})
	svc.recordAuthoritativeRebuild(1, nil)
	beforeEpoch := svc.authoritativeSnapshotEpoch()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	require.NoError(t, svc.triggerFullRebuildWithContext(ctx, "startup", false))
	require.Equal(t, 1, cache.recoverCalls)
	require.Greater(t, base.lockCalls, len(schedulerCanonicalBuckets(0)))
	require.Greater(t, svc.authoritativeSnapshotEpoch(), beforeEpoch)
	status := svc.InitialSnapshotStatus()
	require.True(t, status.Done)
	require.NoError(t, status.Err)
}

func TestSchedulerRestoreReconcileReleasesLeaseWhenResetFails(t *testing.T) {
	trace := &schedulerRestoreTrace{}
	resetErr := errors.New("redis reset failed")
	cache := &schedulerRestoreCache{trace: trace, resetErr: resetErr}
	lease := &schedulerRestoreLease{trace: trace}
	outbox := &schedulerRestoreOutboxRepo{trace: trace, lease: lease}
	svc := NewSchedulerSnapshotService(cache, outbox, schedulerRestoreAccountRepo{}, nil, &config.Config{RunMode: config.RunModeSimple})

	require.ErrorIs(t, svc.ReconcileAfterDatabaseRestore(context.Background()), resetErr)
	require.Equal(t, []string{"acquire", "reset", "release"}, trace.snapshot())
	require.Zero(t, cache.snapshotCnt)
	status := svc.InitialSnapshotStatus()
	require.True(t, status.Done)
	require.ErrorIs(t, status.Err, resetErr)
}

func TestSchedulerRestoreReconcileFailedRebuildInvalidatesEarlierReadiness(t *testing.T) {
	trace := &schedulerRestoreTrace{}
	rebuildErr := errors.New("restored snapshot rebuild failed")
	cache := &schedulerRestoreCache{trace: trace, listErr: rebuildErr}
	lease := &schedulerRestoreLease{trace: trace}
	outbox := &schedulerRestoreOutboxRepo{trace: trace, lease: lease}
	svc := NewSchedulerSnapshotService(cache, outbox, schedulerRestoreAccountRepo{}, nil, &config.Config{RunMode: config.RunModeSimple})
	svc.recordAuthoritativeRebuild(1, nil)
	status := svc.InitialSnapshotStatus()
	require.True(t, status.Done)
	require.NoError(t, status.Err)

	require.ErrorIs(t, svc.ReconcileAfterDatabaseRestore(context.Background()), rebuildErr)
	require.Equal(t, []string{"acquire", "reset", "release", "rebuild"}, trace.snapshot())
	status = svc.InitialSnapshotStatus()
	require.True(t, status.Done)
	require.ErrorIs(t, status.Err, rebuildErr)
}

func TestSchedulerRestoreReconcileResetFencesRacingPreResetSuccess(t *testing.T) {
	trace := &schedulerRestoreTrace{}
	rebuildErr := errors.New("restore rebuild failed after reset")
	cache := &schedulerRestoreCache{trace: trace, listErr: rebuildErr}
	lease := &schedulerRestoreLease{trace: trace}
	outbox := &schedulerRestoreOutboxRepo{trace: trace, lease: lease}
	svc := NewSchedulerSnapshotService(cache, outbox, schedulerRestoreAccountRepo{}, nil, &config.Config{RunMode: config.RunModeSimple})
	cache.resetHook = func() {
		// Simulate an older periodic rebuild publishing success after restore's
		// first invalidation but immediately before reset removes that projection.
		require.NoError(t, svc.coalesceFullRebuild(func() error { return nil }))
	}

	require.ErrorIs(t, svc.ReconcileAfterDatabaseRestore(context.Background()), rebuildErr)
	require.Equal(t, []string{"acquire", "reset", "release", "rebuild"}, trace.snapshot())
	status := svc.InitialSnapshotStatus()
	require.True(t, status.Done)
	require.ErrorIs(t, status.Err, rebuildErr)
}

func TestSchedulerSnapshotInvalidationRejectsDelayedPreResetReadinessResult(t *testing.T) {
	svc := &SchedulerSnapshotService{}
	preResetEpoch := svc.authoritativeSnapshotEpoch()
	svc.invalidateAuthoritativeSnapshot()

	svc.recordAuthoritativeRebuildAtEpoch(1, preResetEpoch, nil)
	status := svc.InitialSnapshotStatus()
	require.False(t, status.Done)
	require.NoError(t, status.Err)
}
