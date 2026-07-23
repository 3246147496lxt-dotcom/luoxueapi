package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/lifecycle"
	"github.com/stretchr/testify/require"
)

func TestRequestDrainerStopsAdmissionAndWaitsForActiveHandler(t *testing.T) {
	drainer := NewRequestDrainer()
	started := make(chan struct{})
	release := make(chan struct{})
	handler := drainer.Wrap(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
		w.WriteHeader(http.StatusNoContent)
	}))

	firstDone := make(chan struct{})
	go func() {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/stream", nil))
		close(firstDone)
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("active request did not start")
	}
	require.Equal(t, 1, drainer.ActiveCount())

	drainer.BeginDrain()
	rejected := httptest.NewRecorder()
	handler.ServeHTTP(rejected, httptest.NewRequest(http.MethodGet, "/new", nil))
	require.Equal(t, http.StatusServiceUnavailable, rejected.Code)
	require.Equal(t, "close", rejected.Header().Get("Connection"))

	waitCtx, cancelWait := context.WithTimeout(context.Background(), 20*time.Millisecond)
	require.ErrorIs(t, drainer.Wait(waitCtx), context.DeadlineExceeded)
	cancelWait()

	close(release)
	select {
	case <-firstDone:
	case <-time.After(time.Second):
		t.Fatal("active request did not finish")
	}
	require.NoError(t, drainer.Wait(context.Background()))
	require.Zero(t, drainer.ActiveCount())
}

func TestRequestDrainerCancelActiveCancelsEveryRequestContext(t *testing.T) {
	drainer := NewRequestDrainer()
	started := make(chan struct{}, 2)
	done := make(chan struct{}, 2)
	handler := drainer.Wrap(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		started <- struct{}{}
		<-r.Context().Done()
		done <- struct{}{}
	}))

	for range 2 {
		go handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/stream", nil))
	}
	for range 2 {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("active request did not start")
		}
	}

	drainer.BeginDrain()
	drainer.CancelActive()
	for range 2 {
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("active request did not observe cancellation")
		}
	}
	waitCtx, cancelWait := context.WithTimeout(context.Background(), time.Second)
	defer cancelWait()
	require.NoError(t, drainer.Wait(waitCtx))
}

func TestRequestDrainerReturnsStructuredReadinessWhileDraining(t *testing.T) {
	drainer := NewRequestDrainer()
	handlerCalled := false
	handler := drainer.Wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		handlerCalled = true
	}))
	drainer.BeginDrain()

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	require.Equal(t, http.StatusServiceUnavailable, response.Code)
	require.Equal(t, "application/json; charset=utf-8", response.Header().Get("Content-Type"))
	require.Equal(t, "no-store", response.Header().Get("Cache-Control"))
	require.Equal(t, "close", response.Header().Get("Connection"))
	require.Equal(t, "1", response.Header().Get("Retry-After"))
	require.False(t, handlerCalled)

	var result lifecycle.ReadinessResult
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
	require.Equal(t, lifecycle.ReadinessNotReady, result.Status)
	require.Equal(t, lifecycle.CheckResult{
		Status: lifecycle.ReadinessNotReady,
		Detail: "ingress_disabled",
	}, result.Checks["draining"])
}
