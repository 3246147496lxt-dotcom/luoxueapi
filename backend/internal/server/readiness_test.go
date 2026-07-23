package server

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/lifecycle"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

type schedulerReadinessStub struct {
	status service.SchedulerInitialSnapshotStatus
}

func (s schedulerReadinessStub) InitialSnapshotStatus() service.SchedulerInitialSnapshotStatus {
	return s.status
}

func newReadinessTestDependencies(t *testing.T) (*dependencyReadinessProbe, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("new sql mock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })
	supervisor := lifecycle.NewSupervisor()
	if err := supervisor.Start(context.Background()); err != nil {
		t.Fatalf("start supervisor: %v", err)
	}
	t.Cleanup(func() { _ = supervisor.Stop(context.Background()) })
	return &dependencyReadinessProbe{
		db:         db,
		redis:      redisClient,
		supervisor: supervisor,
	}, mock
}

func TestDependencyReadinessProbeReady(t *testing.T) {
	probe, mock := newReadinessTestDependencies(t)
	mock.ExpectPing()
	probe.scheduler = schedulerReadinessStub{status: service.SchedulerInitialSnapshotStatus{Done: true}}

	result := probe.Probe(context.Background())
	if result.Status != lifecycle.ReadinessReady {
		t.Fatalf("status = %q, want ready", result.Status)
	}
	if check := result.Checks["components"]; check.Status != lifecycle.ReadinessReady || check.Detail == "" {
		t.Fatalf("components check = %#v", check)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDependencyReadinessProbeRejectsSupervisorThatHasNotStarted(t *testing.T) {
	probe, _ := newReadinessTestDependencies(t)
	probe.supervisor = lifecycle.NewSupervisor(lifecycle.ComponentFuncs{ComponentName: "worker"})

	result := probe.Probe(context.Background())
	if result.Status != lifecycle.ReadinessNotReady {
		t.Fatalf("status = %q, want not_ready", result.Status)
	}
	check := result.Checks["components"]
	if check.Status != lifecycle.ReadinessNotReady || check.Detail == "" {
		t.Fatalf("components check = %#v", check)
	}
}

func TestDependencyReadinessProbeAllowsSchedulerDegradedOnlyWithFallback(t *testing.T) {
	for _, test := range []struct {
		name       string
		fallback   bool
		wantStatus string
	}{
		{name: "fallback enabled", fallback: true, wantStatus: lifecycle.ReadinessDegraded},
		{name: "fallback disabled", fallback: false, wantStatus: lifecycle.ReadinessNotReady},
	} {
		t.Run(test.name, func(t *testing.T) {
			probe, mock := newReadinessTestDependencies(t)
			mock.ExpectPing()
			probe.dbFallback = test.fallback
			probe.scheduler = schedulerReadinessStub{status: service.SchedulerInitialSnapshotStatus{
				Done: true,
				Err:  errors.New("rebuild failed"),
			}}

			result := probe.Probe(context.Background())
			if result.Status != test.wantStatus {
				t.Fatalf("status = %q, want %q", result.Status, test.wantStatus)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestDependencyReadinessProbeRejectsUnavailableDatabase(t *testing.T) {
	probe, mock := newReadinessTestDependencies(t)
	mock.ExpectPing().WillReturnError(errors.New("database unavailable"))
	probe.scheduler = schedulerReadinessStub{status: service.SchedulerInitialSnapshotStatus{Done: true}}

	result := probe.Probe(context.Background())
	if result.Status != lifecycle.ReadinessNotReady {
		t.Fatalf("status = %q, want not_ready", result.Status)
	}
	if check := result.Checks["database"]; check.Status != lifecycle.ReadinessNotReady {
		t.Fatalf("database check = %#v", check)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDependencyReadinessProbeRejectsUnavailableRedis(t *testing.T) {
	probe, mock := newReadinessTestDependencies(t)
	mock.ExpectPing()
	probe.scheduler = schedulerReadinessStub{status: service.SchedulerInitialSnapshotStatus{Done: true}}
	if err := probe.redis.Close(); err != nil {
		t.Fatalf("close Redis client: %v", err)
	}

	result := probe.Probe(context.Background())
	if result.Status != lifecycle.ReadinessNotReady {
		t.Fatalf("status = %q, want not_ready", result.Status)
	}
	if check := result.Checks["redis"]; check.Status != lifecycle.ReadinessNotReady {
		t.Fatalf("redis check = %#v", check)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDependencyReadinessProbeRejectsDrainingBeforeDependencyChecks(t *testing.T) {
	probe, mock := newReadinessTestDependencies(t)
	probe.supervisor.BeginDrain()

	result := probe.Probe(context.Background())
	if result.Status != lifecycle.ReadinessNotReady {
		t.Fatalf("status = %q, want not_ready", result.Status)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
