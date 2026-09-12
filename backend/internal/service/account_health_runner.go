package service

import (
	"context"
	"log"
	"sync"
	"time"
)

// AccountHealthRunner performs the conservative daily health scan in the
// background. It intentionally reuses AccountHealthService, so manual scans
// and scheduled scans share classification and quarantine behavior.
type AccountHealthRunner struct {
	svc  *AccountHealthService
	stop chan struct{}
	done chan struct{}
	once sync.Once
}

func NewAccountHealthRunner(svc *AccountHealthService) *AccountHealthRunner {
	return &AccountHealthRunner{svc: svc, stop: make(chan struct{}), done: make(chan struct{})}
}

func (r *AccountHealthRunner) Start(ctx context.Context) error {
	if r == nil || r.svc == nil {
		return nil
	}
	r.once.Do(func() {
		go func() {
			defer close(r.done)
			nextRun := time.Now().Add(24 * time.Hour)
			if cfg, err := r.svc.GetSettings(ctx); err == nil && cfg != nil && cfg.ScanIntervalMinutes > 0 {
				nextRun = time.Now().Add(time.Duration(cfg.ScanIntervalMinutes) * time.Minute)
			}
			ticker := time.NewTicker(time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-r.stop:
					return
				case <-ticker.C:
					cfg, err := r.svc.GetSettings(ctx)
					if err != nil || cfg == nil || !cfg.Enabled {
						continue
					}
					interval := 24 * time.Hour
					if cfg.ScanIntervalMinutes > 0 {
						interval = time.Duration(cfg.ScanIntervalMinutes) * time.Minute
					}
					if time.Now().Before(nextRun) {
						continue
					}
					if _, err := r.svc.Scan(ctx, AccountHealthScanRequest{Test: true}); err != nil {
						log.Printf("[AccountHealth] scheduled scan failed: %v", err)
					}
					nextRun = time.Now().Add(interval)
				}
			}
		}()
	})
	return nil
}

func (r *AccountHealthRunner) Stop(ctx context.Context) error {
	if r == nil {
		return nil
	}
	select {
	case <-r.done:
		return nil
	default:
	}
	select {
	case <-r.stop:
	default:
		close(r.stop)
	}
	select {
	case <-r.done:
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}
