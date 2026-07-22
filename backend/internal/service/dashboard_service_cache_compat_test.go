package service

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/stretchr/testify/require"
)

func TestDashboardService_OldCachePayloadDefaultsActiveAPIKeyWeekFields(t *testing.T) {
	oldPayload := fmt.Sprintf(
		`{"stats":{"total_users":7,"stats_updated_at":"2026-07-21T07:00:00Z"},"updated_at":%d}`,
		time.Now().Unix(),
	)
	cache := &dashboardCacheStub{
		get: func(context.Context) (string, error) {
			return oldPayload, nil
		},
	}
	repo := &usageRepoStub{stats: &usagestats.DashboardStats{TotalUsers: 99}}
	svc := NewDashboardService(repo, nil, cache, &config.Config{
		Dashboard: config.DashboardCacheConfig{Enabled: true},
	})

	stats, err := svc.GetDashboardStats(context.Background())
	require.NoError(t, err)
	require.Equal(t, int64(7), stats.TotalUsers)
	require.Zero(t, stats.CurrentWeekActiveAPIKeys)
	require.Zero(t, stats.PreviousWeekSamePeriodActiveAPIKeys)
	require.Empty(t, stats.CurrentWeekStartAt)
	require.Empty(t, stats.CurrentWeekEndAt)
	require.Empty(t, stats.PreviousWeekSamePeriodStartAt)
	require.Empty(t, stats.PreviousWeekSamePeriodEndAt)
	require.Empty(t, stats.StatsTimezone)
	require.Zero(t, atomic.LoadInt32(&repo.calls))
}
