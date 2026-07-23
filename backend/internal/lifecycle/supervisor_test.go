package lifecycle

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestSupervisorRollsBackStartedComponentsInReverseOrder(t *testing.T) {
	var calls []string
	component := func(name string, startErr error) Component {
		return ComponentFuncs{
			ComponentName: name,
			StartFunc: func(context.Context) error {
				calls = append(calls, "start:"+name)
				return startErr
			},
			StopFunc: func(context.Context) error {
				calls = append(calls, "stop:"+name)
				return nil
			},
		}
	}

	supervisor := NewSupervisor(
		component("one", nil),
		component("two", nil),
		component("three", errors.New("boom")),
	)
	err := supervisor.Start(context.Background())
	if err == nil {
		t.Fatal("expected startup failure")
	}
	want := []string{"start:one", "start:two", "start:three", "stop:two", "stop:one"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %#v, want %#v", calls, want)
	}
}

func TestSupervisorRollbackDoesNotReuseCanceledStartupContext(t *testing.T) {
	startupCtx, cancelStartup := context.WithCancel(context.Background())
	stopSawLiveContext := false
	supervisor := NewSupervisor(
		ComponentFuncs{
			ComponentName: "started",
			StartFunc:     func(context.Context) error { return nil },
			StopFunc: func(ctx context.Context) error {
				stopSawLiveContext = ctx.Err() == nil
				return nil
			},
		},
		ComponentFuncs{
			ComponentName: "failing",
			StartFunc: func(context.Context) error {
				cancelStartup()
				return context.Canceled
			},
		},
	)

	if err := supervisor.Start(startupCtx); !errors.Is(err, context.Canceled) {
		t.Fatalf("start error = %v, want context.Canceled", err)
	}
	if !stopSawLiveContext {
		t.Fatal("startup rollback reused the canceled startup context")
	}
}

func TestSupervisorStopIsIdempotentAndReverseOrdered(t *testing.T) {
	var calls []string
	component := func(name string) Component {
		return ComponentFuncs{
			ComponentName: name,
			StartFunc:     func(context.Context) error { return nil },
			StopFunc: func(context.Context) error {
				calls = append(calls, name)
				return nil
			},
		}
	}

	supervisor := NewSupervisor(component("producer"), component("consumer"), component("flusher"))
	if err := supervisor.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := supervisor.Stop(context.Background()); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if err := supervisor.Stop(context.Background()); err != nil {
		t.Fatalf("second stop: %v", err)
	}
	want := []string{"flusher", "consumer", "producer"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %#v, want %#v", calls, want)
	}
	if !supervisor.IsDraining() {
		t.Fatal("supervisor must remain draining after stop")
	}
}

func TestSupervisorContinuesStoppingAfterError(t *testing.T) {
	var calls []string
	supervisor := NewSupervisor(
		ComponentFuncs{ComponentName: "one", StopFunc: func(context.Context) error {
			calls = append(calls, "one")
			return nil
		}},
		ComponentFuncs{ComponentName: "two", StopFunc: func(context.Context) error {
			calls = append(calls, "two")
			return errors.New("stop failed")
		}},
	)
	if err := supervisor.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := supervisor.Stop(context.Background()); err == nil {
		t.Fatal("expected stop error")
	}
	want := []string{"two", "one"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %#v, want %#v", calls, want)
	}
}

func TestSupervisorRuntimeStatsTrackComponentStateAndDrainDuration(t *testing.T) {
	supervisor := NewSupervisor(ComponentFuncs{
		ComponentName: "worker",
		StopFunc: func(context.Context) error {
			time.Sleep(5 * time.Millisecond)
			return nil
		},
	})
	if got := supervisor.RuntimeStats(); got.State != "new" || got.ComponentCount != 1 {
		t.Fatalf("new stats = %#v", got)
	}
	if err := supervisor.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	if got := supervisor.RuntimeStats(); got.State != "running" || got.StartedCount != 1 || got.ErrorCount != 0 {
		t.Fatalf("running stats = %#v", got)
	}
	if err := supervisor.Stop(context.Background()); err != nil {
		t.Fatalf("stop: %v", err)
	}
	first := supervisor.RuntimeStats()
	if first.State != "stopped" || !first.Draining || first.StoppedCount != 1 || first.StopCount != 1 {
		t.Fatalf("stopped stats = %#v", first)
	}
	if first.LastStopDuration < 5*time.Millisecond || first.TotalStopDuration != first.LastStopDuration {
		t.Fatalf("stop durations = %#v", first)
	}
	if err := supervisor.Stop(context.Background()); err != nil {
		t.Fatalf("second stop: %v", err)
	}
	second := supervisor.RuntimeStats()
	if second.StopCount != 1 || second.TotalStopDuration != first.TotalStopDuration {
		t.Fatalf("idempotent stop changed stats: first=%#v second=%#v", first, second)
	}
}

func TestSupervisorTimedOutStopRetainsComponentForRetry(t *testing.T) {
	stopCalls := 0
	supervisor := NewSupervisor(ComponentFuncs{
		ComponentName: "worker",
		StopFunc: func(ctx context.Context) error {
			stopCalls++
			if stopCalls == 1 {
				<-ctx.Done()
				return ctx.Err()
			}
			return nil
		},
	})
	if err := supervisor.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	stopCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := supervisor.Stop(stopCtx); !errors.Is(err, context.Canceled) {
		t.Fatalf("first stop error = %v, want context.Canceled", err)
	}
	if !supervisor.HasPendingComponents() {
		t.Fatal("timed-out component was forgotten")
	}
	status := supervisor.Status()
	if len(status) != 1 || status[0].Stopped || status[0].Error == "" {
		t.Fatalf("status after timeout = %#v", status)
	}
	if got := supervisor.RuntimeStats(); got.State != "stopping" || got.StopCount != 1 {
		t.Fatalf("runtime after timeout = %#v", got)
	}

	if err := supervisor.Stop(context.Background()); err != nil {
		t.Fatalf("retry stop: %v", err)
	}
	if supervisor.HasPendingComponents() {
		t.Fatal("component remained pending after successful retry")
	}
	if stopCalls != 2 {
		t.Fatalf("stop calls = %d, want 2", stopCalls)
	}
	if got := supervisor.RuntimeStats(); got.State != "stopped" || got.StopCount != 2 || got.ErrorCount != 0 {
		t.Fatalf("runtime after retry = %#v", got)
	}
}

func TestSupervisorStartupRollbackFailureCanBeRetried(t *testing.T) {
	stopCalls := 0
	supervisor := NewSupervisor(
		ComponentFuncs{
			ComponentName: "started",
			StopFunc: func(context.Context) error {
				stopCalls++
				if stopCalls == 1 {
					return errors.New("temporary stop failure")
				}
				return nil
			},
		},
		ComponentFuncs{
			ComponentName: "failing",
			StartFunc:     func(context.Context) error { return errors.New("start failure") },
		},
	)
	if err := supervisor.Start(context.Background()); err == nil {
		t.Fatal("expected startup failure")
	}
	if !supervisor.HasPendingComponents() {
		t.Fatal("failed rollback component was forgotten")
	}
	if err := supervisor.Stop(context.Background()); err != nil {
		t.Fatalf("retry rollback: %v", err)
	}
	if stopCalls != 2 || supervisor.HasPendingComponents() {
		t.Fatalf("retry state: calls=%d pending=%v", stopCalls, supervisor.HasPendingComponents())
	}
}
