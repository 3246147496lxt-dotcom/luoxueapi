package service

import (
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
)

const defaultWebChatUpstreamDrainTimeout = 180 * time.Second

var errWebChatUpstreamDrainTimeout = errors.New("web chat upstream drain timeout")

// webChatDrainGuard keeps a detached upstream bounded after the browser has
// gone away. Closing an HTTP response body cancels the remaining HTTP/2 stream
// (or closes the HTTP/1 response), allowing the handler to release concurrency
// only after terminal usage or this timeout.
type webChatDrainGuard struct {
	body    io.Closer
	cancel  context.CancelFunc
	timeout time.Duration

	startOnce sync.Once
	stopOnce  sync.Once
	mu        sync.Mutex
	timer     *time.Timer
	stopped   bool
	stopWatch func() bool
	started   atomic.Bool
	timedOut  atomic.Bool
}

func newWebChatDrainGuard(ctx context.Context, body io.Closer, timeout time.Duration) *webChatDrainGuard {
	if body == nil {
		return nil
	}
	return newWebChatDrainGuardWithCancel(ctx, body, timeout, nil)
}

func newWebChatUpstreamDrainContext(ctx context.Context, timeout time.Duration) (context.Context, *webChatDrainGuard) {
	base := context.Background()
	if ctx != nil {
		base = context.WithoutCancel(ctx)
	}
	if !isWebChatContext(ctx) {
		return base, nil
	}
	upstreamCtx, cancel := context.WithCancel(base)
	return upstreamCtx, newWebChatDrainGuardWithCancel(ctx, nil, timeout, cancel)
}

func newWebChatDrainGuardWithCancel(
	ctx context.Context,
	body io.Closer,
	timeout time.Duration,
	cancel context.CancelFunc,
) *webChatDrainGuard {
	if ctx == nil || (body == nil && cancel == nil) || !isWebChatContext(ctx) {
		return nil
	}
	if timeout <= 0 {
		timeout = defaultWebChatUpstreamDrainTimeout
	}
	g := &webChatDrainGuard{body: body, cancel: cancel, timeout: timeout}
	g.stopWatch = context.AfterFunc(ctx, g.Start)
	return g
}

func isWebChatContext(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	webChat, _ := ctx.Value(ctxkey.WebChat).(bool)
	return webChat
}

func webChatClientDisconnected(ctx context.Context) bool {
	return isWebChatContext(ctx) && ctx.Err() != nil
}

func (g *webChatDrainGuard) Start() {
	if g == nil {
		return
	}
	g.startOnce.Do(func() {
		g.mu.Lock()
		defer g.mu.Unlock()
		if g.stopped {
			return
		}
		g.started.Store(true)
		g.timer = time.AfterFunc(g.timeout, g.expire)
	})
}

func (g *webChatDrainGuard) expire() {
	if g == nil {
		return
	}
	g.mu.Lock()
	if g.stopped {
		g.mu.Unlock()
		return
	}
	g.stopped = true
	g.timedOut.Store(true)
	body := g.body
	cancel := g.cancel
	g.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if body != nil {
		_ = body.Close()
	}
}

func (g *webChatDrainGuard) Stop() {
	if g == nil {
		return
	}
	g.stopOnce.Do(func() {
		if g.stopWatch != nil {
			_ = g.stopWatch()
		}
		g.mu.Lock()
		g.stopped = true
		if g.timer != nil {
			_ = g.timer.Stop()
		}
		cancel := g.cancel
		g.mu.Unlock()
		if cancel != nil {
			cancel()
		}
	})
}

func (g *webChatDrainGuard) SetBody(body io.Closer) {
	if g == nil || body == nil {
		return
	}
	g.mu.Lock()
	if g.timedOut.Load() {
		g.mu.Unlock()
		_ = body.Close()
		return
	}
	if !g.stopped {
		g.body = body
	}
	g.mu.Unlock()
}

func (g *webChatDrainGuard) Started() bool {
	return g != nil && g.started.Load()
}

func (g *webChatDrainGuard) TimedOut() bool {
	return g != nil && g.timedOut.Load()
}

func webChatUpstreamDrainTimeout(cfg *config.Config) time.Duration {
	if cfg != nil && cfg.Gateway.StreamDataIntervalTimeout > 0 {
		return time.Duration(cfg.Gateway.StreamDataIntervalTimeout) * time.Second
	}
	return defaultWebChatUpstreamDrainTimeout
}
