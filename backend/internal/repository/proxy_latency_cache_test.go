package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestProxyLatencyCacheHasBoundedTTLAndCanBeCleared(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache := NewProxyLatencyCache(rdb)
	ctx := context.Background()

	require.NoError(t, cache.SetProxyLatency(ctx, 42, &service.ProxyLatencyInfo{Success: true, UpdatedAt: time.Now()}))
	ttl := mr.TTL(proxyLatencyKey(42))
	require.Greater(t, ttl, time.Duration(0))
	require.LessOrEqual(t, ttl, proxyLatencySnapshotRetention)

	loaded, err := cache.GetProxyLatencies(ctx, []int64{42})
	require.NoError(t, err)
	require.NotNil(t, loaded[42])

	require.NoError(t, cache.DeleteProxyLatency(ctx, 42))
	loaded, err = cache.GetProxyLatencies(ctx, []int64{42})
	require.NoError(t, err)
	require.Nil(t, loaded[42])
}
