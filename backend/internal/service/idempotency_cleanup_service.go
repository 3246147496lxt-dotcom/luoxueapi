package service

import (
	"context"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// IdempotencyCleanupService 定期清理已过期的幂等记录，避免表无限增长。
type IdempotencyCleanupService struct {
	repo     IdempotencyRepository
	interval time.Duration
	batch    int

	startOnce sync.Once
	stopOnce  sync.Once

	lifecycleMu sync.Mutex
	runCtx      context.Context
	runCancel   context.CancelFunc
	runWG       sync.WaitGroup
	stopped     bool
}

func NewIdempotencyCleanupService(repo IdempotencyRepository, cfg *config.Config) *IdempotencyCleanupService {
	interval := 60 * time.Second
	batch := 500
	if cfg != nil {
		if cfg.Idempotency.CleanupIntervalSeconds > 0 {
			interval = time.Duration(cfg.Idempotency.CleanupIntervalSeconds) * time.Second
		}
		if cfg.Idempotency.CleanupBatchSize > 0 {
			batch = cfg.Idempotency.CleanupBatchSize
		}
	}
	return &IdempotencyCleanupService{
		repo:     repo,
		interval: interval,
		batch:    batch,
	}
}

func (s *IdempotencyCleanupService) Start() {
	if s == nil || s.repo == nil {
		return
	}
	s.startOnce.Do(func() {
		s.lifecycleMu.Lock()
		if s.stopped {
			s.lifecycleMu.Unlock()
			return
		}
		if s.runCtx == nil {
			s.runCtx, s.runCancel = context.WithCancel(context.Background())
		}
		runCtx := s.runCtx
		s.runWG.Add(1)
		s.lifecycleMu.Unlock()

		logger.LegacyPrintf("service.idempotency_cleanup", "[IdempotencyCleanup] started interval=%s batch=%d", s.interval, s.batch)
		go func() {
			defer s.runWG.Done()
			s.runLoop(runCtx)
		}()
	})
}

func (s *IdempotencyCleanupService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		s.lifecycleMu.Lock()
		s.stopped = true
		cancel := s.runCancel
		s.lifecycleMu.Unlock()
		if cancel != nil {
			cancel()
		}
		s.runWG.Wait()
		logger.LegacyPrintf("service.idempotency_cleanup", "[IdempotencyCleanup] stopped")
	})
}

func (s *IdempotencyCleanupService) runLoop(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// 启动后先清理一轮，防止重启后积压。
	s.cleanupOnceWithContext(ctx)
	if ctx.Err() != nil {
		return
	}

	for {
		select {
		case <-ticker.C:
			s.cleanupOnceWithContext(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (s *IdempotencyCleanupService) cleanupOnce() {
	s.cleanupOnceWithContext(s.operationContext())
}

func (s *IdempotencyCleanupService) cleanupOnceWithContext(parent context.Context) {
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()

	deleted, err := s.repo.DeleteExpired(ctx, time.Now(), s.batch)
	if err != nil {
		logger.LegacyPrintf("service.idempotency_cleanup", "[IdempotencyCleanup] cleanup failed err=%v", err)
		return
	}
	if deleted > 0 {
		logger.LegacyPrintf("service.idempotency_cleanup", "[IdempotencyCleanup] cleaned expired records count=%d", deleted)
	}
}

func (s *IdempotencyCleanupService) operationContext() context.Context {
	if s == nil {
		return context.Background()
	}
	s.lifecycleMu.Lock()
	ctx := s.runCtx
	s.lifecycleMu.Unlock()
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
