package main

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestApplicationStopWithinContextTimeoutSharesStopAcrossRetries(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var calls atomic.Int32
	stop := applicationStopWithinContext(func() {
		calls.Add(1)
		close(started)
		<-release
	})

	expiredCtx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	require.ErrorIs(t, stop(expiredCtx), context.DeadlineExceeded)
	waitRuntimeSupervisorSignal(t, started)
	require.Equal(t, int32(1), calls.Load())

	retryDone := make(chan error, 1)
	go func() {
		retryDone <- stop(context.Background())
	}()
	select {
	case err := <-retryDone:
		t.Fatalf("retry returned before the original Stop completed: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	close(release)
	require.NoError(t, waitRuntimeSupervisorResult(t, retryDone))

	completedCtx, completedCancel := context.WithCancel(context.Background())
	completedCancel()
	require.NoError(t, stop(completedCtx))
	require.Equal(t, int32(1), calls.Load())
}

func TestApplicationStopWithinContextConcurrentCallersShareStop(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var calls atomic.Int32
	stop := applicationStopWithinContext(func() {
		calls.Add(1)
		close(started)
		<-release
	})

	const callers = 16
	start := make(chan struct{})
	results := make(chan error, callers)
	var ready sync.WaitGroup
	ready.Add(callers)
	for i := 0; i < callers; i++ {
		go func() {
			ready.Done()
			<-start
			results <- stop(context.Background())
		}()
	}
	ready.Wait()
	close(start)
	waitRuntimeSupervisorSignal(t, started)

	select {
	case err := <-results:
		t.Fatalf("caller returned before the shared Stop completed: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	require.Equal(t, int32(1), calls.Load())

	close(release)
	for i := 0; i < callers; i++ {
		require.NoError(t, waitRuntimeSupervisorResult(t, results))
	}
	require.NoError(t, stop(nil))
	require.Equal(t, int32(1), calls.Load())
}

func TestApplicationStopWithinContextNilStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.NoError(t, applicationStopWithinContext(nil)(ctx))
}

func waitRuntimeSupervisorSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Stop to start")
	}
}

func waitRuntimeSupervisorResult(t *testing.T, result <-chan error) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Stop result")
		return nil
	}
}
