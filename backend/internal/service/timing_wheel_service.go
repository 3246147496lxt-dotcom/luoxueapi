package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/zeromicro/go-zero/core/collection"
)

var newTimingWheel = collection.NewTimingWheel

// TimingWheelService wraps go-zero's TimingWheel for task scheduling
type TimingWheelService struct {
	mu       sync.RWMutex
	tw       *collection.TimingWheel
	started  bool
	stopped  bool
	startErr error
	stopOnce sync.Once
}

// NewTimingWheelService creates a new TimingWheelService instance
func NewTimingWheelService() (*TimingWheelService, error) {
	return &TimingWheelService{}, nil
}

// StartWithError allocates the underlying go-zero timing wheel at process
// startup rather than during dependency construction.
func (s *TimingWheelService) StartWithError() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return s.startErr
	}
	if s.stopped {
		return fmt.Errorf("timing wheel already stopped")
	}
	s.started = true
	// 1 second tick, 3600 slots = supports up to 1 hour delay
	// execute function: runs func() type tasks
	tw, err := newTimingWheel(1*time.Second, 3600, func(key, value any) {
		if fn, ok := value.(func()); ok {
			fn()
		}
	})
	if err != nil {
		s.startErr = fmt.Errorf("创建 timing wheel 失败: %w", err)
		return s.startErr
	}
	s.tw = tw
	logger.LegacyPrintf("service.timing_wheel", "%s", "[TimingWheel] Started")
	return nil
}

// Start starts the timing wheel
func (s *TimingWheelService) Start() {
	if err := s.StartWithError(); err != nil {
		logger.LegacyPrintf("service.timing_wheel", "[TimingWheel] start failed: %v", err)
	}
}

// Stop stops the timing wheel
func (s *TimingWheelService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		s.mu.Lock()
		s.stopped = true
		tw := s.tw
		s.tw = nil
		s.mu.Unlock()
		if tw != nil {
			tw.Stop()
		}
		logger.LegacyPrintf("service.timing_wheel", "%s", "[TimingWheel] Stopped")
	})
}

// Schedule schedules a one-time task
func (s *TimingWheelService) Schedule(name string, delay time.Duration, fn func()) {
	s.mu.RLock()
	tw := s.tw
	s.mu.RUnlock()
	if tw == nil {
		logger.LegacyPrintf("service.timing_wheel", "[TimingWheel] SetTimer failed for %q: service not started", name)
		return
	}
	if err := tw.SetTimer(name, fn, delay); err != nil {
		logger.LegacyPrintf("service.timing_wheel", "[TimingWheel] SetTimer failed for %q: %v", name, err)
	}
}

// ScheduleRecurring schedules a recurring task
func (s *TimingWheelService) ScheduleRecurring(name string, interval time.Duration, fn func()) {
	var schedule func()
	schedule = func() {
		fn()
		s.mu.RLock()
		tw := s.tw
		s.mu.RUnlock()
		if tw == nil {
			return
		}
		if err := tw.SetTimer(name, schedule, interval); err != nil {
			logger.LegacyPrintf("service.timing_wheel", "[TimingWheel] recurring SetTimer failed for %q: %v", name, err)
		}
	}
	s.mu.RLock()
	tw := s.tw
	s.mu.RUnlock()
	if tw == nil {
		logger.LegacyPrintf("service.timing_wheel", "[TimingWheel] initial SetTimer failed for %q: service not started", name)
		return
	}
	if err := tw.SetTimer(name, schedule, interval); err != nil {
		logger.LegacyPrintf("service.timing_wheel", "[TimingWheel] initial SetTimer failed for %q: %v", name, err)
	}
}

// Cancel cancels a scheduled task
func (s *TimingWheelService) Cancel(name string) {
	if s == nil {
		return
	}
	s.mu.RLock()
	tw := s.tw
	s.mu.RUnlock()
	if tw != nil {
		_ = tw.RemoveTimer(name)
	}
}
