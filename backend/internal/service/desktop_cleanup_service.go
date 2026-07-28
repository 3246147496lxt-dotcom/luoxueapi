package service

import (
	"context"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/modules/desktop"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

const (
	defaultDesktopCleanupInterval  = time.Hour
	defaultDesktopCleanupBatchSize = 500
	defaultDesktopSessionRetention = 7 * 24 * time.Hour
	desktopCleanupOperationTimeout = 30 * time.Second
)

type DesktopCleanupService struct {
	repo             desktop.CleanupRepository
	interval         time.Duration
	batch            int
	sessionRetention time.Duration

	startOnce   sync.Once
	stopOnce    sync.Once
	lifecycleMu sync.Mutex
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	stopped     bool
}

func NewDesktopCleanupService(repo desktop.CleanupRepository) *DesktopCleanupService {
	return &DesktopCleanupService{
		repo:             repo,
		interval:         defaultDesktopCleanupInterval,
		batch:            defaultDesktopCleanupBatchSize,
		sessionRetention: defaultDesktopSessionRetention,
	}
}

func (s *DesktopCleanupService) Start() {
	if s == nil || s.repo == nil {
		return
	}
	s.startOnce.Do(func() {
		s.lifecycleMu.Lock()
		if s.stopped {
			s.lifecycleMu.Unlock()
			return
		}
		ctx, cancel := context.WithCancel(context.Background())
		s.cancel = cancel
		s.wg.Add(1)
		s.lifecycleMu.Unlock()
		go func() {
			defer s.wg.Done()
			s.run(ctx)
		}()
	})
}

func (s *DesktopCleanupService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		s.lifecycleMu.Lock()
		s.stopped = true
		cancel := s.cancel
		s.lifecycleMu.Unlock()
		if cancel != nil {
			cancel()
		}
		s.wg.Wait()
	})
}

func (s *DesktopCleanupService) RunOnce(ctx context.Context, now time.Time) (*desktop.CleanupResult, error) {
	if s == nil || s.repo == nil {
		return &desktop.CleanupResult{}, nil
	}
	return s.repo.CleanupExpired(ctx, now, now.Add(-s.sessionRetention), s.batch)
}

func (s *DesktopCleanupService) run(ctx context.Context) {
	runOnce := func() {
		operationCtx, cancel := context.WithTimeout(ctx, desktopCleanupOperationTimeout)
		defer cancel()
		result, err := s.RunOnce(operationCtx, time.Now())
		if err != nil {
			logger.LegacyPrintf("service.desktop_cleanup", "[DesktopCleanup] cleanup failed err=%v", err)
			return
		}
		if result.PendingDevicesDeleted > 0 || result.SessionsDeleted > 0 || result.DiagnosticsDeleted > 0 {
			logger.LegacyPrintf(
				"service.desktop_cleanup",
				"[DesktopCleanup] cleaned pending_devices=%d sessions=%d diagnostics=%d",
				result.PendingDevicesDeleted,
				result.SessionsDeleted,
				result.DiagnosticsDeleted,
			)
		}
	}

	runOnce()
	if ctx.Err() != nil {
		return
	}
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			runOnce()
		case <-ctx.Done():
			return
		}
	}
}
