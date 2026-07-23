package server

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/lifecycle"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

type schedulerInitialSnapshotReader interface {
	InitialSnapshotStatus() service.SchedulerInitialSnapshotStatus
}

type dependencyReadinessProbe struct {
	db         *sql.DB
	redis      *redis.Client
	scheduler  schedulerInitialSnapshotReader
	dbFallback bool
	supervisor *lifecycle.Supervisor
}

func ProvideReadinessProbe(
	db *sql.DB,
	redisClient *redis.Client,
	scheduler *service.SchedulerSnapshotService,
	cfg *config.Config,
	supervisor *lifecycle.Supervisor,
) lifecycle.ReadinessProbe {
	dbFallback := false
	if cfg != nil {
		dbFallback = cfg.Gateway.Scheduling.DbFallbackEnabled
	}
	return &dependencyReadinessProbe{
		db:         db,
		redis:      redisClient,
		scheduler:  scheduler,
		dbFallback: dbFallback,
		supervisor: supervisor,
	}
}

func (p *dependencyReadinessProbe) Probe(ctx context.Context) lifecycle.ReadinessResult {
	result := lifecycle.ReadinessResult{
		Status: lifecycle.ReadinessReady,
		Checks: make(map[string]lifecycle.CheckResult, 5),
	}

	if p == nil || p.supervisor == nil {
		result.Status = lifecycle.ReadinessNotReady
		result.Checks["draining"] = lifecycle.CheckResult{Status: lifecycle.ReadinessNotReady, Detail: "ingress_disabled"}
		result.Checks["components"] = lifecycle.CheckResult{Status: lifecycle.ReadinessNotReady, Detail: "supervisor_unavailable"}
		return result
	}
	runtimeStats := p.supervisor.RuntimeStats()
	componentDetail := fmt.Sprintf(
		"state=%s started=%d/%d stopped=%d errors=%d",
		runtimeStats.State,
		runtimeStats.StartedCount,
		runtimeStats.ComponentCount,
		runtimeStats.StoppedCount,
		runtimeStats.ErrorCount,
	)
	componentsReady := runtimeStats.State == "running" &&
		runtimeStats.StartedCount == runtimeStats.ComponentCount &&
		runtimeStats.ErrorCount == 0
	if componentsReady {
		result.Checks["components"] = lifecycle.CheckResult{Status: lifecycle.ReadinessReady, Detail: componentDetail}
	} else {
		result.Status = lifecycle.ReadinessNotReady
		result.Checks["components"] = lifecycle.CheckResult{Status: lifecycle.ReadinessNotReady, Detail: componentDetail}
	}
	if p.supervisor.IsDraining() {
		result.Status = lifecycle.ReadinessNotReady
		result.Checks["draining"] = lifecycle.CheckResult{Status: lifecycle.ReadinessNotReady, Detail: "ingress_disabled"}
		return result
	}
	result.Checks["draining"] = lifecycle.CheckResult{Status: lifecycle.ReadinessReady}
	if !componentsReady {
		return result
	}

	if p.db == nil || p.db.PingContext(ctx) != nil {
		result.Status = lifecycle.ReadinessNotReady
		result.Checks["database"] = lifecycle.CheckResult{Status: lifecycle.ReadinessNotReady, Detail: "unavailable"}
	} else {
		result.Checks["database"] = lifecycle.CheckResult{Status: lifecycle.ReadinessReady}
	}

	if p.redis == nil || p.redis.Ping(ctx).Err() != nil {
		result.Status = lifecycle.ReadinessNotReady
		result.Checks["redis"] = lifecycle.CheckResult{Status: lifecycle.ReadinessNotReady, Detail: "unavailable"}
	} else {
		result.Checks["redis"] = lifecycle.CheckResult{Status: lifecycle.ReadinessReady}
	}

	schedulerStatus := service.SchedulerInitialSnapshotStatus{}
	if p.scheduler != nil {
		schedulerStatus = p.scheduler.InitialSnapshotStatus()
	}
	if p.scheduler != nil && schedulerStatus.Done && schedulerStatus.Err == nil {
		result.Checks["scheduler"] = lifecycle.CheckResult{Status: lifecycle.ReadinessReady}
		return result
	}

	detail := "initializing"
	if p.scheduler == nil {
		detail = "unavailable"
	} else if schedulerStatus.Err != nil {
		detail = "initial_snapshot_failed"
	}
	if p.dbFallback && result.Status != lifecycle.ReadinessNotReady {
		result.Status = lifecycle.ReadinessDegraded
		result.Checks["scheduler"] = lifecycle.CheckResult{Status: lifecycle.ReadinessDegraded, Detail: detail}
		return result
	}
	result.Status = lifecycle.ReadinessNotReady
	result.Checks["scheduler"] = lifecycle.CheckResult{Status: lifecycle.ReadinessNotReady, Detail: detail}
	return result
}
