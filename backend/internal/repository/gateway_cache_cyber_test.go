package repository

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type cyberRedisCommandHook struct {
	mu            sync.Mutex
	mgetKeyCounts []int
	setBatchSizes []int
}

func (h *cyberRedisCommandHook) DialHook(next redis.DialHook) redis.DialHook { return next }

func (h *cyberRedisCommandHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if cmd.Name() == "mget" {
			h.mu.Lock()
			h.mgetKeyCounts = append(h.mgetKeyCounts, len(cmd.Args())-1)
			h.mu.Unlock()
		}
		return next(ctx, cmd)
	}
}

func (h *cyberRedisCommandHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		setCount := 0
		for _, cmd := range cmds {
			if cmd.Name() == "eval" {
				setCount++
			}
		}
		if setCount > 0 {
			h.mu.Lock()
			h.setBatchSizes = append(h.setBatchSizes, setCount)
			h.mu.Unlock()
		}
		return next(ctx, cmds)
	}
}

func TestGatewayCacheCyberBlockWritesScopeAndExactKeysTogether(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	store, ok := NewGatewayCache(client).(service.CyberSessionTranscriptBlockStore)
	require.True(t, ok)

	ctx := context.Background()
	require.NoError(t, store.SetCyberSessionBlockedKeys(ctx, "scope-1", []string{"block-1", "block-2"}, time.Minute))
	active, err := store.IsCyberSessionScopeActive(ctx, "scope-1")
	require.NoError(t, err)
	require.True(t, active)
	matched, err := store.FindCyberSessionBlocked(ctx, []string{"missing", "block-1", "block-2"})
	require.NoError(t, err)
	require.Equal(t, "block-1", matched)
	require.Greater(t, server.TTL(cyberSessionScopePrefix+"scope-1"), time.Duration(0))
	require.Equal(t, server.TTL(cyberSessionBlockPrefix+"block-1"), server.TTL(cyberSessionBlockPrefix+"block-2"))
}

func TestGatewayCacheCyberBlockCommandsAreBoundedAndLookupShortCircuits(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	hook := &cyberRedisCommandHook{}
	client.AddHook(hook)
	store, ok := NewGatewayCache(client).(service.CyberSessionTranscriptBlockStore)
	require.True(t, ok)

	keys := make([]string, cyberSessionRedisCommandMaxKeys*2+44)
	for i := range keys {
		keys[i] = "block-" + strconv.Itoa(i)
	}
	ctx := context.Background()
	require.NoError(t, store.SetCyberSessionBlockedKeys(ctx, "large-scope", keys, time.Minute))
	require.Equal(t, []int{cyberSessionRedisCommandMaxKeys, cyberSessionRedisCommandMaxKeys, 44}, hook.setBatchSizes)

	lookup := make([]string, len(keys))
	for i := range lookup {
		lookup[i] = "missing-" + strconv.Itoa(i)
	}
	lookup[cyberSessionRedisCommandMaxKeys+3] = keys[cyberSessionRedisCommandMaxKeys+3]
	matched, err := store.FindCyberSessionBlocked(ctx, lookup)
	require.NoError(t, err)
	require.Equal(t, keys[cyberSessionRedisCommandMaxKeys+3], matched)
	require.Equal(t, []int{cyberSessionRedisCommandMaxKeys, cyberSessionRedisCommandMaxKeys}, hook.mgetKeyCounts)
}

func TestGatewayCacheCyberMarkersExpireWithoutShorteningExistingTTL(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	store := NewGatewayCache(client).(service.CyberSessionTranscriptBlockStore)
	ctx := context.Background()
	require.NoError(t, store.SetCyberSessionBlockedKeys(ctx, "scope", []string{"old"}, 5*time.Minute))
	server.FastForward(time.Minute)
	require.NoError(t, store.SetCyberSessionBlockedKeys(ctx, "scope", []string{"new", "old"}, time.Minute))
	require.Equal(t, 4*time.Minute, server.TTL(cyberSessionBlockPrefix+"old"))
	require.Equal(t, 4*time.Minute, server.TTL(cyberSessionScopePrefix+"scope"))
	server.FastForward(2 * time.Minute)
	active, err := store.IsCyberSessionScopeActive(ctx, "scope")
	require.NoError(t, err)
	require.True(t, active)
	key, err := store.FindCyberSessionBlocked(ctx, []string{"new", "old"})
	require.NoError(t, err)
	require.Equal(t, "old", key)
	server.FastForward(3 * time.Minute)
	active, err = store.IsCyberSessionScopeActive(ctx, "scope")
	require.NoError(t, err)
	require.False(t, active)
	key, err = store.FindCyberSessionBlocked(ctx, []string{"new", "old"})
	require.NoError(t, err)
	require.Empty(t, key)
}

func TestGatewayCacheCyberScopeCoversExistingLongerExactBlock(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	store := NewGatewayCache(client).(service.CyberSessionTranscriptBlockStore)
	ctx := context.Background()
	require.NoError(t, store.SetCyberSessionBlockedKeys(ctx, "first-scope", []string{"shared-key"}, 5*time.Minute))
	server.FastForward(time.Minute)
	require.NoError(t, store.SetCyberSessionBlockedKeys(ctx, "second-scope", []string{"shared-key"}, time.Minute))
	require.GreaterOrEqual(t, server.TTL(cyberSessionScopePrefix+"second-scope"), server.TTL(cyberSessionBlockPrefix+"shared-key"))
}

func TestGatewayCacheCyberEmptyAndInvalidInputsDoNotActivateScope(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	store := NewGatewayCache(client).(service.CyberSessionTranscriptBlockStore)
	ctx := context.Background()
	require.NoError(t, store.SetCyberSessionBlockedKeys(ctx, "scope", []string{"", ""}, time.Minute))
	require.NoError(t, store.SetCyberSessionBlockedKeys(ctx, "scope", nil, time.Minute))
	for _, ttl := range []time.Duration{0, -time.Second, time.Nanosecond} {
		require.Error(t, store.SetCyberSessionBlockedKeys(ctx, "scope", []string{"key"}, ttl))
	}
	require.Empty(t, server.Keys())
	key, err := store.FindCyberSessionBlocked(ctx, []string{"", ""})
	require.NoError(t, err)
	require.Empty(t, key)
	active, err := store.IsCyberSessionScopeActive(ctx, "scope")
	require.NoError(t, err)
	require.False(t, active)
}

func TestGatewayCacheCyberFailedWriteDoesNotActivateScope(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr(), MaxRetries: -1})
	t.Cleanup(func() { _ = client.Close() })
	store := NewGatewayCache(client).(service.CyberSessionTranscriptBlockStore)
	server.SetError("ERR storage unavailable")
	require.Error(t, store.SetCyberSessionBlockedKeys(context.Background(), "scope", []string{"key"}, time.Minute))
	server.SetError("")
	require.False(t, server.Exists(cyberSessionScopePrefix+"scope"))
}

func TestGatewayCacheCyberLegacyInterfaceRemainsCompatible(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	cache := NewGatewayCache(client)
	legacy := cache.(service.CyberSessionBlockStore)
	transcript := cache.(service.CyberSessionTranscriptBlockStore)
	ctx := context.Background()
	require.NoError(t, legacy.SetCyberSessionBlocked(ctx, "legacy", time.Minute))
	key, err := transcript.FindCyberSessionBlocked(ctx, []string{"missing", "", "legacy"})
	require.NoError(t, err)
	require.Equal(t, "legacy", key)
	require.NoError(t, transcript.SetCyberSessionBlockedKeys(ctx, "scope", []string{"new"}, time.Minute))
	blocked, err := legacy.IsCyberSessionBlocked(ctx, "new")
	require.NoError(t, err)
	require.True(t, blocked)
}
