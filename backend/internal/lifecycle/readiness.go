package lifecycle

import "context"

const (
	ReadinessReady    = "ready"
	ReadinessDegraded = "degraded"
	ReadinessNotReady = "not_ready"
)

type CheckResult struct {
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

type ReadinessResult struct {
	Status string                 `json:"status"`
	Checks map[string]CheckResult `json:"checks,omitempty"`
}

func (r ReadinessResult) Ready() bool {
	return r.Status == ReadinessReady || r.Status == ReadinessDegraded
}

// ReadinessProbe reports whether this process should receive new traffic.
type ReadinessProbe interface {
	Probe(context.Context) ReadinessResult
}
