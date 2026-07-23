package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/opsruntime"
	"github.com/stretchr/testify/require"
)

func TestOpsDeliveryRuntimeStatsFacade(t *testing.T) {
	want := opsruntime.SnapshotDeliveryRuntimeStats()
	require.Equal(t, want, SnapshotDeliveryRuntimeStats())
	require.Equal(t, want, (*OpsService)(nil).GetDeliveryRuntimeStats())
}

func TestOpsDashboardOverviewIncludesDeliveryRuntimeStats(t *testing.T) {
	now := time.Now().UTC()
	service := &OpsService{opsRepo: &opsRepoMock{}}

	overview, err := service.GetDashboardOverview(context.Background(), &OpsDashboardFilter{
		StartTime: now.Add(-time.Minute),
		EndTime:   now,
	})

	require.NoError(t, err)
	require.Equal(t, SnapshotDeliveryRuntimeStats(), overview.DeliveryRuntime)
}
