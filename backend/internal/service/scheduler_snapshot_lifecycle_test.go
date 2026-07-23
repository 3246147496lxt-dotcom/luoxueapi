//go:build unit

package service

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type schedulerLifecycleInitialCache struct {
	SchedulerCache

	started  chan struct{}
	canceled chan struct{}
	release  chan struct{}
}

func (c *schedulerLifecycleInitialCache) ListBuckets(ctx context.Context) ([]SchedulerBucket, error) {
	close(c.started)
	<-ctx.Done()
	close(c.canceled)
	<-c.release
	return nil, ctx.Err()
}

type schedulerLifecycleIntervalCache struct {
	SchedulerCache

	calls           atomic.Int32
	startupErr      error
	intervalStarted chan struct{}
	intervalCancel  chan struct{}
	release         chan struct{}
}

func (c *schedulerLifecycleIntervalCache) ListBuckets(ctx context.Context) ([]SchedulerBucket, error) {
	if c.calls.Add(1) == 1 {
		return nil, c.startupErr
	}
	close(c.intervalStarted)
	<-ctx.Done()
	close(c.intervalCancel)
	<-c.release
	return nil, ctx.Err()
}

type schedulerLifecycleOutboxCache struct {
	*outboxCleanupCache

	once     sync.Once
	started  chan struct{}
	canceled chan struct{}
	release  chan struct{}
}

func (c *schedulerLifecycleOutboxCache) UpdateLastUsed(ctx context.Context, _ map[int64]time.Time) error {
	c.once.Do(func() { close(c.started) })
	<-ctx.Done()
	close(c.canceled)
	<-c.release
	return ctx.Err()
}

type schedulerLagDeadlineRepo struct {
	SchedulerOutboxRepository

	queryExpired chan struct{}
}

func (r *schedulerLagDeadlineRepo) FirstCreatedAtAfter(ctx context.Context, _ int64) (time.Time, bool, error) {
	<-ctx.Done()
	close(r.queryExpired)
	return time.Now().Add(-time.Hour), true, nil
}

type schedulerLagRebuildCache struct {
	SchedulerCache

	rebuildStarted  chan struct{}
	rebuildCanceled chan struct{}
	release         chan struct{}
}

func (c *schedulerLagRebuildCache) ListBuckets(ctx context.Context) ([]SchedulerBucket, error) {
	close(c.rebuildStarted)
	<-ctx.Done()
	close(c.rebuildCanceled)
	<-c.release
	return nil, ctx.Err()
}

func TestSchedulerSnapshotServiceStopCancelsAndJoinsInitialRebuild(t *testing.T) {
	cache := &schedulerLifecycleInitialCache{
		started:  make(chan struct{}),
		canceled: make(chan struct{}),
		release:  make(chan struct{}),
	}
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(cache.release) }) }
	t.Cleanup(release)
	svc := NewSchedulerSnapshotService(cache, nil, nil, nil, nil)

	svc.Start()
	requireSchedulerLifecycleSignal(t, cache.started, "initial rebuild did not start")

	stopped := make(chan struct{}, 2)
	for range 2 {
		go func() {
			svc.Stop()
			stopped <- struct{}{}
		}()
	}
	requireSchedulerLifecycleSignal(t, cache.canceled, "initial rebuild did not observe Stop cancellation")
	assertSchedulerStopWaiting(t, stopped)

	release()
	requireSchedulerLifecycleSignal(t, stopped, "first Stop did not join the initial rebuild")
	requireSchedulerLifecycleSignal(t, stopped, "second Stop did not share the completed join")
	status := svc.InitialSnapshotStatus()
	require.True(t, status.Done)
	require.ErrorIs(t, status.Err, context.Canceled)
}

func TestSchedulerSnapshotServiceStopCancelsAndJoinsIntervalFullRebuild(t *testing.T) {
	startupErr := errors.New("startup rebuild failed")
	cache := &schedulerLifecycleIntervalCache{
		startupErr:      startupErr,
		intervalStarted: make(chan struct{}),
		intervalCancel:  make(chan struct{}),
		release:         make(chan struct{}),
	}
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(cache.release) }) }
	t.Cleanup(release)
	svc := NewSchedulerSnapshotService(cache, nil, nil, nil, &config.Config{
		Gateway: config.GatewayConfig{
			Scheduling: config.GatewaySchedulingConfig{FullRebuildIntervalSeconds: 1},
		},
	})

	svc.Start()
	requireSchedulerLifecycleSignal(t, cache.intervalStarted, "interval full rebuild did not start")

	stopped := make(chan struct{}, 1)
	go func() {
		svc.Stop()
		close(stopped)
	}()
	requireSchedulerLifecycleSignal(t, cache.intervalCancel, "interval full rebuild did not observe Stop cancellation")
	assertSchedulerStopWaiting(t, stopped)

	release()
	requireSchedulerLifecycleSignal(t, stopped, "Stop did not join the interval full rebuild")
	require.EqualValues(t, 2, cache.calls.Load())
}

func TestSchedulerSnapshotServiceStopCancelsOutboxWithoutAdvancingWatermark(t *testing.T) {
	cache := &schedulerLifecycleOutboxCache{
		outboxCleanupCache: &outboxCleanupCache{listBucketErr: errors.New("startup rebuild failed")},
		started:            make(chan struct{}),
		canceled:           make(chan struct{}),
		release:            make(chan struct{}),
	}
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(cache.release) }) }
	t.Cleanup(release)
	repo := &outboxCleanupRepo{events: []SchedulerOutboxEvent{{
		ID:        1,
		EventType: SchedulerOutboxEventAccountLastUsed,
		Payload: map[string]any{
			"last_used": map[string]any{"101": float64(123)},
		},
	}}}
	svc := NewSchedulerSnapshotService(cache, repo, nil, nil, nil)

	svc.Start()
	requireSchedulerLifecycleSignal(t, cache.started, "outbox event did not start")

	stopped := make(chan struct{}, 1)
	go func() {
		svc.Stop()
		close(stopped)
	}()
	requireSchedulerLifecycleSignal(t, cache.canceled, "outbox event did not observe Stop cancellation")
	assertSchedulerStopWaiting(t, stopped)

	release()
	requireSchedulerLifecycleSignal(t, stopped, "Stop did not join the outbox worker")
	require.Zero(t, cache.watermark)
	require.Empty(t, cache.setWatermarks, "a canceled event must remain durable for replay")
	require.Len(t, repo.events, 1)
	repo.consumerMu.Lock()
	require.False(t, repo.consumerLeaseHeld)
	require.Equal(t, 1, repo.consumerReleases)
	repo.consumerMu.Unlock()
}

func TestSchedulerSnapshotServiceLagQueryDeadlineDoesNotBoundRebuildAndStopCancelsIt(t *testing.T) {
	cache := &schedulerLagRebuildCache{
		rebuildStarted:  make(chan struct{}),
		rebuildCanceled: make(chan struct{}),
		release:         make(chan struct{}),
	}
	repo := &schedulerLagDeadlineRepo{queryExpired: make(chan struct{})}
	svc := NewSchedulerSnapshotService(cache, repo, nil, nil, &config.Config{
		Gateway: config.GatewayConfig{
			Scheduling: config.GatewaySchedulingConfig{
				OutboxLagRebuildSeconds:  1,
				OutboxLagRebuildFailures: 1,
			},
		},
	})

	rebuildParent, rebuildCancel := context.WithCancel(context.Background())
	svc.lifecycleMu.Lock()
	svc.lifecycleCancel = rebuildCancel
	svc.lifecycleMu.Unlock()
	queryCtx, queryCancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer queryCancel()
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(cache.release) }) }
	t.Cleanup(func() {
		rebuildCancel()
		release()
	})

	checkDone := make(chan struct{})
	svc.wg.Add(1)
	go func() {
		defer svc.wg.Done()
		defer close(checkDone)
		svc.checkOutboxLagWithRebuildParent(queryCtx, rebuildParent, 0)
	}()

	requireSchedulerLifecycleSignal(t, repo.queryExpired, "lag query context did not reach its deadline")
	requireSchedulerLifecycleSignal(t, cache.rebuildStarted, "lag rebuild inherited the expired query context")
	require.ErrorIs(t, queryCtx.Err(), context.DeadlineExceeded)

	stopped := make(chan struct{})
	go func() {
		svc.Stop()
		close(stopped)
	}()
	requireSchedulerLifecycleSignal(t, cache.rebuildCanceled, "lag rebuild did not observe Stop cancellation")
	assertSchedulerStopWaiting(t, stopped)

	release()
	requireSchedulerLifecycleSignal(t, stopped, "Stop did not join the lag rebuild")
	requireSchedulerLifecycleSignal(t, checkDone, "lag check did not exit after Stop")
}

func requireSchedulerLifecycleSignal(t *testing.T, signal <-chan struct{}, failure string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(3 * time.Second):
		t.Fatal(failure)
	}
}

func assertSchedulerStopWaiting(t *testing.T, stopped <-chan struct{}) {
	t.Helper()
	select {
	case <-stopped:
		t.Fatal("Stop returned before canceled scheduler work exited")
	default:
	}
}
