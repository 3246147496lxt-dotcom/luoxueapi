package service

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

type blockingLifecycleSettingRepository struct {
	SettingRepository
	started  chan struct{}
	canceled chan struct{}
	release  chan struct{}
	calls    atomic.Int32
}

func newBlockingLifecycleSettingRepository(expectedCalls int) *blockingLifecycleSettingRepository {
	return &blockingLifecycleSettingRepository{
		started:  make(chan struct{}, expectedCalls),
		canceled: make(chan struct{}, expectedCalls),
		release:  make(chan struct{}),
	}
}

func (r *blockingLifecycleSettingRepository) GetValue(ctx context.Context, _ string) (string, error) {
	r.calls.Add(1)
	r.started <- struct{}{}
	<-ctx.Done()
	r.canceled <- struct{}{}
	<-r.release
	return "", ctx.Err()
}

type lifecycleOpsRepository struct {
	OpsRepository
}

type blockingLifecycleIdempotencyRepository struct {
	IdempotencyRepository
	started  chan struct{}
	canceled chan struct{}
	release  chan struct{}
	calls    atomic.Int32
}

func newBlockingLifecycleIdempotencyRepository() *blockingLifecycleIdempotencyRepository {
	return &blockingLifecycleIdempotencyRepository{
		started:  make(chan struct{}, 1),
		canceled: make(chan struct{}, 1),
		release:  make(chan struct{}),
	}
}

func (r *blockingLifecycleIdempotencyRepository) DeleteExpired(ctx context.Context, _ time.Time, _ int) (int64, error) {
	r.calls.Add(1)
	r.started <- struct{}{}
	<-ctx.Done()
	r.canceled <- struct{}{}
	<-r.release
	return 0, ctx.Err()
}

func TestOpsMetricsCollectorStopCancelsAndJoinsLoop(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	settings := newBlockingLifecycleSettingRepository(1)
	collector := NewOpsMetricsCollector(
		&lifecycleOpsRepository{},
		settings,
		nil,
		nil,
		db,
		nil,
		nil,
	)
	collector.Start()
	waitLifecycleSignals(t, settings.started, 1)

	assertConcurrentStopCancelsAndJoins(t, collector.Stop, settings.canceled, settings.release, 1)
	require.Equal(t, int32(1), settings.calls.Load())
}

func TestOpsAggregationServiceStopCancelsAndJoinsBothLoops(t *testing.T) {
	settings := newBlockingLifecycleSettingRepository(2)
	aggregation := NewOpsAggregationService(&lifecycleOpsRepository{}, settings, nil, nil, nil)
	aggregation.Start()
	waitLifecycleSignals(t, settings.started, 2)

	assertConcurrentStopCancelsAndJoins(t, aggregation.Stop, settings.canceled, settings.release, 2)
	require.Equal(t, int32(2), settings.calls.Load())
}

func TestIdempotencyCleanupServiceStopCancelsAndJoinsLoop(t *testing.T) {
	repo := newBlockingLifecycleIdempotencyRepository()
	cleanup := NewIdempotencyCleanupService(repo, nil)
	cleanup.Start()
	waitLifecycleSignals(t, repo.started, 1)

	assertConcurrentStopCancelsAndJoins(t, cleanup.Stop, repo.canceled, repo.release, 1)
	require.Equal(t, int32(1), repo.calls.Load())
}

func assertConcurrentStopCancelsAndJoins(
	t *testing.T,
	stop func(),
	canceled <-chan struct{},
	release chan struct{},
	expectedCancellations int,
) {
	t.Helper()
	const callers = 8
	start := make(chan struct{})
	returned := make(chan struct{}, callers)
	for i := 0; i < callers; i++ {
		go func() {
			<-start
			stop()
			returned <- struct{}{}
		}()
	}
	close(start)
	waitLifecycleSignals(t, canceled, expectedCancellations)

	select {
	case <-returned:
		t.Fatal("Stop returned before the canceled background work exited")
	case <-time.After(50 * time.Millisecond):
	}

	close(release)
	waitLifecycleSignals(t, returned, callers)

	idempotentReturn := make(chan struct{}, 1)
	go func() {
		stop()
		idempotentReturn <- struct{}{}
	}()
	waitLifecycleSignals(t, idempotentReturn, 1)
}

func waitLifecycleSignals(t *testing.T, signals <-chan struct{}, count int) {
	t.Helper()
	for i := 0; i < count; i++ {
		select {
		case <-signals:
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for lifecycle signal %d/%d", i+1, count)
		}
	}
}
