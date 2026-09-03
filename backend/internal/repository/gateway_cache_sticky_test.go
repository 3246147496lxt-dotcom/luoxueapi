package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestGatewayCacheStickySessionMissUsesServiceSentinel(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { require.NoError(t, rdb.Close()) })
	cache := &gatewayCache{rdb: rdb}

	accountID, err := cache.GetSessionAccountID(context.Background(), 7, "missing")

	require.Zero(t, accountID)
	require.ErrorIs(t, err, service.ErrStickySessionNotFound)
	require.False(t, errors.Is(err, redis.Nil), "repository-specific cache misses must not leak across the service boundary")
}

func TestGatewayCacheStickySessionRoundTripStillWorks(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { require.NoError(t, rdb.Close()) })
	cache := &gatewayCache{rdb: rdb}
	ctx := context.Background()

	require.NoError(t, cache.SetSessionAccountID(ctx, 7, "session", 42, time.Minute))
	accountID, err := cache.GetSessionAccountID(ctx, 7, "session")

	require.NoError(t, err)
	require.Equal(t, int64(42), accountID)
}
