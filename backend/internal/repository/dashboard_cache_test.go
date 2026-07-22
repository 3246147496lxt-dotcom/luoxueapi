package repository

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestNewDashboardCacheKeyPrefix(t *testing.T) {
	cache := NewDashboardCache(nil, &config.Config{
		Dashboard: config.DashboardCacheConfig{
			KeyPrefix: "prod",
		},
	})
	impl, ok := cache.(*dashboardCache)
	require.True(t, ok)
	require.Equal(t, "prod:", impl.keyPrefix)

	cache = NewDashboardCache(nil, &config.Config{
		Dashboard: config.DashboardCacheConfig{
			KeyPrefix: "staging:",
		},
	})
	impl, ok = cache.(*dashboardCache)
	require.True(t, ok)
	require.Equal(t, "staging:", impl.keyPrefix)
}

func TestDashboardCacheKeyVersionInvalidatesV3PayloadsMissingDayUserComparison(t *testing.T) {
	redisServer := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { require.NoError(t, rdb.Close()) })

	const legacyKey = "sub2api:dashboard:stats:v3"
	require.NoError(t, redisServer.Set(legacyKey, `{"stats":{"current_day_start_at":"2026-07-21T00:00:00+08:00","current_day_requests":12},"updated_at":1}`))

	cache := NewDashboardCache(rdb, nil)
	impl, ok := cache.(*dashboardCache)
	require.True(t, ok)
	require.Equal(t, "sub2api:dashboard:stats:v4", impl.buildKey())
	require.NotEqual(t, legacyKey, impl.buildKey())

	_, err := cache.GetDashboardStats(context.Background())
	require.ErrorIs(t, err, service.ErrDashboardStatsCacheMiss)
}
