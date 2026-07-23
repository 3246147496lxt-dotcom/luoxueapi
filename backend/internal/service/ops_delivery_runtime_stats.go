package service

import "github.com/Wei-Shaw/sub2api/internal/pkg/opsruntime"

type OpsDeliveryRuntimeStats = opsruntime.DeliveryRuntimeStats

// SnapshotDeliveryRuntimeStats is a dependency-free ops facade over the
// process-local migration and outbox delivery acceptance signals.
func SnapshotDeliveryRuntimeStats() OpsDeliveryRuntimeStats {
	return opsruntime.SnapshotDeliveryRuntimeStats()
}

func (s *OpsService) GetDeliveryRuntimeStats() OpsDeliveryRuntimeStats {
	return SnapshotDeliveryRuntimeStats()
}
