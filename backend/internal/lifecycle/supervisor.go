package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type ComponentStatus struct {
	Name    string
	Started bool
	Stopped bool
	Error   string
}

// SupervisorRuntimeStats is a point-in-time process lifecycle summary. Stop
// durations measure component drain/flush time after HTTP ingress has begun
// shutting down; repeated idempotent Stop calls do not increment StopCount.
type SupervisorRuntimeStats struct {
	State             string
	Draining          bool
	ComponentCount    int
	StartedCount      int
	StoppedCount      int
	ErrorCount        int
	LastStopDuration  time.Duration
	TotalStopDuration time.Duration
	StopCount         uint64
}

// Supervisor starts components in dependency order and stops only the
// successfully started prefix in reverse order.
type Supervisor struct {
	components []Component

	mu      sync.Mutex
	started []Component
	status  map[string]ComponentStatus
	state   supervisorState

	draining atomic.Bool
	stopMu   sync.Mutex
	stopErr  error

	lastStopDuration  time.Duration
	totalStopDuration time.Duration
	stopCount         uint64
}

type supervisorState uint8

const (
	supervisorNew supervisorState = iota
	supervisorStarting
	supervisorRunning
	supervisorStopping
	supervisorStopped
)

const startupRollbackTimeout = 10 * time.Second

func NewSupervisor(components ...Component) *Supervisor {
	filtered := make([]Component, 0, len(components))
	status := make(map[string]ComponentStatus, len(components))
	for _, component := range components {
		if component == nil {
			continue
		}
		filtered = append(filtered, component)
		status[component.Name()] = ComponentStatus{Name: component.Name()}
	}
	return &Supervisor{components: filtered, status: status}
}

func (s *Supervisor) Start(ctx context.Context) error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	if s.state == supervisorRunning {
		s.mu.Unlock()
		return nil
	}
	if s.state != supervisorNew {
		state := s.state
		s.mu.Unlock()
		return fmt.Errorf("lifecycle supervisor cannot start from state %d", state)
	}
	s.state = supervisorStarting
	s.mu.Unlock()

	for _, component := range s.components {
		if err := component.Start(ctx); err != nil {
			s.setComponentError(component.Name(), err)
			// A startup failure is often caused by ctx cancellation. Reusing that
			// canceled context would skip context-aware Stop implementations and
			// leave an already-started prefix running while the process unwinds.
			rollbackCtx, cancelRollback := context.WithTimeout(context.Background(), startupRollbackTimeout)
			rollbackErr := s.rollback(rollbackCtx)
			cancelRollback()
			s.mu.Lock()
			if len(s.started) == 0 {
				s.state = supervisorStopped
			} else {
				s.state = supervisorStopping
			}
			s.mu.Unlock()
			return errors.Join(fmt.Errorf("start component %s: %w", component.Name(), err), rollbackErr)
		}
		s.mu.Lock()
		s.started = append(s.started, component)
		componentStatus := s.status[component.Name()]
		componentStatus.Started = true
		s.status[component.Name()] = componentStatus
		s.mu.Unlock()
	}

	s.mu.Lock()
	s.state = supervisorRunning
	s.mu.Unlock()
	return nil
}

func (s *Supervisor) BeginDrain() {
	if s != nil {
		s.draining.Store(true)
	}
}

func (s *Supervisor) IsDraining() bool {
	return s != nil && s.draining.Load()
}

func (s *Supervisor) Stop(ctx context.Context) error {
	if s == nil {
		return nil
	}
	s.BeginDrain()
	s.stopMu.Lock()
	defer s.stopMu.Unlock()

	s.mu.Lock()
	if len(s.started) == 0 {
		s.state = supervisorStopped
		s.stopErr = nil
		s.mu.Unlock()
		return nil
	}
	s.state = supervisorStopping
	s.mu.Unlock()

	startedAt := time.Now()
	s.stopErr = s.rollback(ctx)
	s.mu.Lock()
	duration := time.Since(startedAt)
	s.lastStopDuration = duration
	s.totalStopDuration += duration
	s.stopCount++
	if len(s.started) == 0 {
		s.state = supervisorStopped
	} else {
		s.state = supervisorStopping
	}
	s.mu.Unlock()
	return s.stopErr
}

func (s *Supervisor) rollback(ctx context.Context) error {
	s.mu.Lock()
	started := append([]Component(nil), s.started...)
	s.mu.Unlock()

	var errs []error
	for i := len(started) - 1; i >= 0; i-- {
		component := started[i]
		if err := component.Stop(ctx); err != nil {
			errs = append(errs, fmt.Errorf("stop component %s: %w", component.Name(), err))
			s.setComponentError(component.Name(), err)
			continue
		}
		s.markComponentStopped(component)
	}
	return errors.Join(errs...)
}

func (s *Supervisor) markComponentStopped(component Component) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, started := range s.started {
		if started.Name() == component.Name() {
			s.started = append(s.started[:i], s.started[i+1:]...)
			break
		}
	}
	componentStatus := s.status[component.Name()]
	componentStatus.Stopped = true
	componentStatus.Error = ""
	s.status[component.Name()] = componentStatus
}

// HasPendingComponents reports whether shutdown still has a component that
// may access process infrastructure. Callers must not close DB/Redis while it
// returns true.
func (s *Supervisor) HasPendingComponents() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.started) > 0
}

func (s *Supervisor) setComponentError(name string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	componentStatus := s.status[name]
	componentStatus.Name = name
	if err != nil {
		componentStatus.Error = err.Error()
	}
	s.status[name] = componentStatus
}

func (s *Supervisor) Status() []ComponentStatus {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]ComponentStatus, 0, len(s.components))
	for _, component := range s.components {
		result = append(result, s.status[component.Name()])
	}
	return result
}

// RuntimeStats returns a consistent lifecycle/readiness snapshot.
func (s *Supervisor) RuntimeStats() SupervisorRuntimeStats {
	if s == nil {
		return SupervisorRuntimeStats{State: "unavailable"}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	stats := SupervisorRuntimeStats{
		State:             s.state.String(),
		Draining:          s.draining.Load(),
		ComponentCount:    len(s.components),
		LastStopDuration:  s.lastStopDuration,
		TotalStopDuration: s.totalStopDuration,
		StopCount:         s.stopCount,
	}
	for _, component := range s.components {
		status := s.status[component.Name()]
		if status.Started {
			stats.StartedCount++
		}
		if status.Stopped {
			stats.StoppedCount++
		}
		if status.Error != "" {
			stats.ErrorCount++
		}
	}
	return stats
}

func (s supervisorState) String() string {
	switch s {
	case supervisorNew:
		return "new"
	case supervisorStarting:
		return "starting"
	case supervisorRunning:
		return "running"
	case supervisorStopping:
		return "stopping"
	case supervisorStopped:
		return "stopped"
	default:
		return "unknown"
	}
}
