package server

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/lifecycle"
)

type fakeRouterSettingsSource struct {
	mu               sync.Mutex
	origins          []string
	callback         func()
	registerCalls    int
	unsubscribeCalls int
	readCalls        int
	blockAfterRead   int
	blockedRead      chan struct{}
	releaseRead      chan struct{}
	blockedOnce      sync.Once
}

func (f *fakeRouterSettingsSource) GetFrameSrcOrigins(context.Context) ([]string, error) {
	f.mu.Lock()
	f.readCalls++
	readCall := f.readCalls
	origins := append([]string(nil), f.origins...)
	blockAfterRead := f.blockAfterRead
	blockedRead := f.blockedRead
	releaseRead := f.releaseRead
	f.mu.Unlock()
	if blockAfterRead > 0 && readCall > blockAfterRead {
		f.blockedOnce.Do(func() {
			if blockedRead != nil {
				close(blockedRead)
			}
		})
		if releaseRead != nil {
			<-releaseRead
		}
	}
	return origins, nil
}

func (f *fakeRouterSettingsSource) RegisterOnUpdateCallback(callback func()) func() {
	f.mu.Lock()
	f.registerCalls++
	f.callback = callback
	f.mu.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			f.mu.Lock()
			f.unsubscribeCalls++
			f.callback = nil
			f.mu.Unlock()
		})
	}
}

func (f *fakeRouterSettingsSource) notify() {
	f.mu.Lock()
	callback := f.callback
	f.mu.Unlock()
	if callback != nil {
		callback()
	}
}

func TestRouterSettingsRuntimeConstructionHasNoCallbackOrDatabaseSideEffect(t *testing.T) {
	source := &fakeRouterSettingsSource{origins: []string{"https://one.example"}}
	runtime := newRouterSettingsRuntime(source, nil, nil)

	if source.registerCalls != 0 || source.readCalls != 0 {
		t.Fatalf("construction calls: register=%d read=%d", source.registerCalls, source.readCalls)
	}
	if got := runtime.FrameSrcOrigins(); len(got) != 0 {
		t.Fatalf("construction origins = %v, want empty", got)
	}
}

func TestRouterSettingsRuntimeStartRefreshesAndStopUnsubscribes(t *testing.T) {
	source := &fakeRouterSettingsSource{origins: []string{"https://one.example"}}
	runtime := newRouterSettingsRuntime(source, nil, nil)

	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	if source.registerCalls != 1 || source.readCalls != 1 {
		t.Fatalf("start calls: register=%d read=%d", source.registerCalls, source.readCalls)
	}
	if got := runtime.FrameSrcOrigins(); !reflect.DeepEqual(got, source.origins) {
		t.Fatalf("origins = %v, want %v", got, source.origins)
	}

	source.mu.Lock()
	source.origins = []string{"https://two.example"}
	source.mu.Unlock()
	source.notify()
	if got := runtime.FrameSrcOrigins(); !reflect.DeepEqual(got, []string{"https://two.example"}) {
		t.Fatalf("updated origins = %v", got)
	}

	if err := runtime.Stop(context.Background()); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if source.unsubscribeCalls != 1 {
		t.Fatalf("unsubscribe calls = %d, want 1", source.unsubscribeCalls)
	}
	readsAfterStop := source.readCalls
	source.notify()
	if source.readCalls != readsAfterStop {
		t.Fatalf("callback read after stop: before=%d after=%d", readsAfterStop, source.readCalls)
	}
}

func TestRouterSettingsRuntimeIsUnsubscribedBySupervisorRollback(t *testing.T) {
	source := &fakeRouterSettingsSource{}
	runtime := newRouterSettingsRuntime(source, nil, nil)
	supervisor := lifecycle.NewSupervisor(
		lifecycle.ComponentFuncs{ComponentName: "router-settings", StartFunc: runtime.Start, StopFunc: runtime.Stop},
		lifecycle.ComponentFuncs{ComponentName: "fail", StartFunc: func(context.Context) error { return errors.New("boom") }},
	)

	if err := supervisor.Start(context.Background()); err == nil {
		t.Fatal("expected startup failure")
	}
	if source.registerCalls != 1 || source.unsubscribeCalls != 1 {
		t.Fatalf("rollback calls: register=%d unsubscribe=%d", source.registerCalls, source.unsubscribeCalls)
	}
}

func TestRouterSettingsRuntimeStopTimeoutCanBeRetried(t *testing.T) {
	source := &fakeRouterSettingsSource{
		blockAfterRead: 1,
		blockedRead:    make(chan struct{}),
		releaseRead:    make(chan struct{}),
	}
	runtime := newRouterSettingsRuntime(source, nil, nil)
	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	notifyDone := make(chan struct{})
	go func() {
		source.notify()
		close(notifyDone)
	}()
	select {
	case <-source.blockedRead:
	case <-time.After(5 * time.Second):
		t.Fatal("settings callback did not block")
	}

	stopCtx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	err := runtime.Stop(stopCtx)
	cancel()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("first stop error = %v, want deadline exceeded", err)
	}
	retryCtx, retryCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer retryCancel()
	retryDone := make(chan error, 1)
	go func() { retryDone <- runtime.Stop(retryCtx) }()
	select {
	case err := <-retryDone:
		t.Fatalf("retry stop returned before callback finished: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	close(source.releaseRead)
	if err := <-retryDone; err != nil {
		t.Fatalf("retry stop: %v", err)
	}
	select {
	case <-notifyDone:
	case <-time.After(5 * time.Second):
		t.Fatal("settings callback did not finish")
	}
	if source.unsubscribeCalls != 1 {
		t.Fatalf("unsubscribe calls = %d, want 1", source.unsubscribeCalls)
	}
}
