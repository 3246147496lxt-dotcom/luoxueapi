package server

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/Wei-Shaw/sub2api/internal/lifecycle"
)

// RequestDrainer prevents new business requests from entering after shutdown
// starts and tracks every active handler independently of its HTTP connection.
// Tracking at the handler boundary also covers concurrent HTTP/2 streams.
type RequestDrainer struct {
	mu        sync.Mutex
	draining  bool
	nextID    uint64
	active    map[uint64]context.CancelFunc
	drainedCh chan struct{}
	drainOnce sync.Once
}

func NewRequestDrainer() *RequestDrainer {
	return &RequestDrainer{
		active:    make(map[uint64]context.CancelFunc),
		drainedCh: make(chan struct{}),
	}
}

func (d *RequestDrainer) Wrap(next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if d == nil {
			next.ServeHTTP(w, r)
			return
		}

		ctx, cancel, requestID, accepted := d.begin(r.Context())
		if !accepted {
			w.Header().Set("Connection", "close")
			w.Header().Set("Retry-After", "1")
			if r.URL != nil && r.URL.Path == "/readyz" {
				writeDrainingReadiness(w)
				return
			}
			http.Error(w, "server is draining", http.StatusServiceUnavailable)
			return
		}
		defer d.finish(requestID)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func writeDrainingReadiness(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusServiceUnavailable)
	_ = json.NewEncoder(w).Encode(lifecycle.ReadinessResult{
		Status: lifecycle.ReadinessNotReady,
		Checks: map[string]lifecycle.CheckResult{
			"draining": {
				Status: lifecycle.ReadinessNotReady,
				Detail: "ingress_disabled",
			},
		},
	})
}

func (d *RequestDrainer) begin(parent context.Context) (context.Context, context.CancelFunc, uint64, bool) {
	if parent == nil {
		parent = context.Background()
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.draining {
		return nil, nil, 0, false
	}
	d.nextID++
	requestID := d.nextID
	ctx, cancel := context.WithCancel(parent)
	d.active[requestID] = cancel
	return ctx, cancel, requestID, true
}

func (d *RequestDrainer) finish(requestID uint64) {
	d.mu.Lock()
	delete(d.active, requestID)
	shouldClose := d.draining && len(d.active) == 0
	d.mu.Unlock()
	if shouldClose {
		d.closeDrained()
	}
}

// BeginDrain atomically closes admission before callers start waiting.
func (d *RequestDrainer) BeginDrain() {
	if d == nil {
		return
	}
	d.mu.Lock()
	d.draining = true
	shouldClose := len(d.active) == 0
	d.mu.Unlock()
	if shouldClose {
		d.closeDrained()
	}
}

// CancelActive is used only after the graceful window expires. It
// gives streaming and hijacked handlers a process-local cancellation signal
// before infrastructure teardown is considered.
func (d *RequestDrainer) CancelActive() {
	if d == nil {
		return
	}
	d.mu.Lock()
	cancels := make([]context.CancelFunc, 0, len(d.active))
	for _, cancel := range d.active {
		cancels = append(cancels, cancel)
	}
	d.mu.Unlock()
	for _, cancel := range cancels {
		cancel()
	}
}

func (d *RequestDrainer) Wait(ctx context.Context) error {
	if d == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-d.drainedCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (d *RequestDrainer) ActiveCount() int {
	if d == nil {
		return 0
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.active)
}

func (d *RequestDrainer) closeDrained() {
	d.drainOnce.Do(func() { close(d.drainedCh) })
}
