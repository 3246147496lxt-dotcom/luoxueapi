package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	internalserver "github.com/Wei-Shaw/sub2api/internal/server"
	"github.com/stretchr/testify/require"
)

type shutdownRuntimeTestServer struct {
	shutdown func(context.Context) error
	close    func() error
}

func (s shutdownRuntimeTestServer) Shutdown(ctx context.Context) error {
	return s.shutdown(ctx)
}

func (s shutdownRuntimeTestServer) Close() error {
	if s.close == nil {
		return nil
	}
	return s.close()
}

type shutdownRuntimeTestSupervisor struct {
	drainCalls atomic.Int32
	stopCalls  atomic.Int32
	stop       func(context.Context) error
}

func (s *shutdownRuntimeTestSupervisor) BeginDrain() {
	s.drainCalls.Add(1)
}

func (s *shutdownRuntimeTestSupervisor) Stop(ctx context.Context) error {
	s.stopCalls.Add(1)
	if s.stop != nil {
		return s.stop(ctx)
	}
	return nil
}

func TestShutdownApplicationRuntimeGracefullyDrainsBeforeStoppingComponents(t *testing.T) {
	drainer := internalserver.NewRequestDrainer()
	started := make(chan struct{})
	release := make(chan struct{})
	handler := drainer.Wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		close(started)
		<-release
	}))
	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/stream", nil))
		close(done)
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("request did not start")
	}

	supervisor := &shutdownRuntimeTestSupervisor{stop: func(context.Context) error {
		require.Zero(t, drainer.ActiveCount(), "components stopped before the handler returned")
		return nil
	}}
	server := shutdownRuntimeTestServer{shutdown: func(ctx context.Context) error {
		close(release)
		return drainer.Wait(ctx)
	}}
	drained, err := shutdownApplicationRuntime(server, drainer, supervisor, time.Second, time.Second, time.Second)
	require.NoError(t, err)
	require.True(t, drained)
	require.Equal(t, int32(1), supervisor.drainCalls.Load())
	require.Equal(t, int32(1), supervisor.stopCalls.Load())
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("request did not finish")
	}
}

func TestShutdownApplicationRuntimeForceCancelsThenStopsComponents(t *testing.T) {
	drainer := internalserver.NewRequestDrainer()
	started := make(chan struct{})
	requestContext := make(chan context.Context, 1)
	handler := drainer.Wrap(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		requestContext <- r.Context()
		close(started)
		<-r.Context().Done()
	}))
	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/stream", nil))
		close(done)
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("request did not start")
	}

	var closeCalls atomic.Int32
	supervisor := &shutdownRuntimeTestSupervisor{stop: func(context.Context) error {
		require.Zero(t, drainer.ActiveCount(), "components stopped before the canceled handler returned")
		return nil
	}}
	server := shutdownRuntimeTestServer{
		shutdown: func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		},
		close: func() error {
			require.ErrorIs(t, (<-requestContext).Err(), context.Canceled, "server closed before request contexts were canceled")
			closeCalls.Add(1)
			return nil
		},
	}
	drained, err := shutdownApplicationRuntime(server, drainer, supervisor, 20*time.Millisecond, time.Second, time.Second)
	require.True(t, drained)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Equal(t, int32(1), closeCalls.Load())
	require.Equal(t, int32(1), supervisor.stopCalls.Load())
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("request did not exit after forced cancellation")
	}
}

func TestShutdownApplicationRuntimeStuckHandlerPreventsComponentStop(t *testing.T) {
	drainer := internalserver.NewRequestDrainer()
	started := make(chan struct{})
	release := make(chan struct{})
	handler := drainer.Wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		close(started)
		<-release
	}))
	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/stuck", nil))
		close(done)
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("request did not start")
	}

	supervisor := &shutdownRuntimeTestSupervisor{}
	server := shutdownRuntimeTestServer{
		shutdown: func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		},
		close: func() error { return nil },
	}
	drained, err := shutdownApplicationRuntime(server, drainer, supervisor, 20*time.Millisecond, 20*time.Millisecond, time.Second)
	require.False(t, drained)
	require.Error(t, err)
	require.True(t, errors.Is(err, context.DeadlineExceeded))
	require.Equal(t, int32(1), supervisor.drainCalls.Load())
	require.Zero(t, supervisor.stopCalls.Load(), "dependencies must remain available to a stuck handler")

	close(release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("stuck request cleanup did not finish")
	}
}
