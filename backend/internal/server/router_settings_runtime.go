package server

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/web"
	"github.com/gin-gonic/gin"
)

const frameSrcRefreshTimeout = 5 * time.Second

type routerSettingsSource interface {
	GetFrameSrcOrigins(ctx context.Context) ([]string, error)
	RegisterOnUpdateCallback(callback func()) func()
}

// RouterSettingsRuntime owns every settings callback used by the HTTP router.
// Construction only prepares immutable handlers and empty caches; database
// reads and callback registration happen under Supervisor ownership in Start.
type RouterSettingsRuntime struct {
	settings routerSettingsSource
	frontend *web.FrontendServer
	frontErr error

	frameOrigins atomic.Pointer[[]string]

	mu          sync.Mutex
	started     bool
	unsubscribe func()
	callbacks   sync.WaitGroup
	stopDone    chan struct{}
}

func ProvideRouterSettingsRuntime(settingService *service.SettingService) *RouterSettingsRuntime {
	frontend, err := web.NewFrontendServer(settingService) //nolint:staticcheck // embed and !embed intentionally differ.
	return newRouterSettingsRuntime(settingService, frontend, err)
}

func newRouterSettingsRuntime(settings routerSettingsSource, frontend *web.FrontendServer, frontendErr error) *RouterSettingsRuntime {
	runtime := &RouterSettingsRuntime{
		settings: settings,
		frontend: frontend,
		frontErr: frontendErr,
	}
	empty := []string{}
	runtime.frameOrigins.Store(&empty)
	return runtime
}

func (r *RouterSettingsRuntime) Start(ctx context.Context) error {
	if r == nil || r.settings == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		r.mu.Lock()
		if r.started {
			r.mu.Unlock()
			return nil
		}
		previousStop := r.stopDone
		if previousStop == nil {
			r.started = true
			r.unsubscribe = r.settings.RegisterOnUpdateCallback(r.handleUpdate)
			r.mu.Unlock()
			break
		}
		r.mu.Unlock()

		select {
		case <-previousStop:
			r.mu.Lock()
			if r.stopDone == previousStop {
				r.stopDone = nil
			}
			r.mu.Unlock()
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Match the old startup behavior without making router construction perform
	// an infrastructure read. A transient failure keeps the last safe value.
	r.refreshFrameOrigins(ctx)
	if r.frontend != nil {
		r.frontend.InvalidateCache()
	}
	return nil
}

func (r *RouterSettingsRuntime) Stop(ctx context.Context) error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	if !r.started && r.stopDone == nil {
		r.mu.Unlock()
		return nil
	}
	unsubscribe := func() {}
	done := r.stopDone
	startWaiter := false
	if r.started {
		r.started = false
		if r.unsubscribe != nil {
			unsubscribe = r.unsubscribe
		}
		r.unsubscribe = nil
		done = make(chan struct{})
		r.stopDone = done
		startWaiter = true
	}
	r.mu.Unlock()
	unsubscribe()

	if startWaiter {
		go func() {
			r.callbacks.Wait()
			close(done)
		}()
	}
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *RouterSettingsRuntime) handleUpdate() {
	r.mu.Lock()
	if !r.started {
		r.mu.Unlock()
		return
	}
	r.callbacks.Add(1)
	r.mu.Unlock()
	defer r.callbacks.Done()

	if r.frontend != nil {
		r.frontend.InvalidateCache()
	}
	ctx, cancel := context.WithTimeout(context.Background(), frameSrcRefreshTimeout)
	defer cancel()
	r.refreshFrameOrigins(ctx)
}

func (r *RouterSettingsRuntime) refreshFrameOrigins(ctx context.Context) {
	if r == nil || r.settings == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, frameSrcRefreshTimeout)
		defer cancel()
	}
	origins, err := r.settings.GetFrameSrcOrigins(ctx)
	if err != nil {
		return
	}
	copyOfOrigins := append([]string(nil), origins...)
	r.frameOrigins.Store(&copyOfOrigins)
}

func (r *RouterSettingsRuntime) FrameSrcOrigins() []string {
	if r == nil {
		return nil
	}
	origins := r.frameOrigins.Load()
	if origins == nil {
		return nil
	}
	return *origins
}

func (r *RouterSettingsRuntime) FrontendMiddleware() (gin.HandlerFunc, bool, error) {
	if r == nil || r.frontend == nil {
		return nil, false, r.frontErr
	}
	return r.frontend.Middleware(), true, nil
}
