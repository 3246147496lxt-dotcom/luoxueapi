package handler

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type slowCyberSessionCache struct {
	openAICyberContinuationCache
	started  chan struct{}
	release  chan struct{}
	finished chan struct{}
}

func (s *slowCyberSessionCache) SetCyberSessionBlockedKeys(ctx context.Context, scope string, keys []string, ttl time.Duration) error {
	close(s.started)
	// Simulate a Redis socket read that does not honor context cancellation.
	<-s.release
	defer close(s.finished)
	return s.openAICyberContinuationCache.SetCyberSessionBlockedKeys(ctx, scope, keys, ttl)
}

func TestPersistCyberSessionBlockPlanBoundsWaitForUnresponsiveCache(t *testing.T) {
	cache := &slowCyberSessionCache{
		started: make(chan struct{}), release: make(chan struct{}), finished: make(chan struct{}),
	}
	defer func() {
		close(cache.release)
		select {
		case <-cache.finished:
		case <-time.After(2 * time.Second):
			t.Error("cache writer did not exit after release")
		}
	}()
	cfg := &config.Config{}
	settings := service.NewSettingService(&openAIWSCyberSettingRepo{values: map[string]string{
		service.SettingKeyCyberSessionBlockEnabled: "true",
	}}, cfg)
	enabled, _ := settings.GetCyberSessionBlockRuntime(context.Background())
	require.True(t, enabled)
	gateway := service.NewOpenAIGatewayService(
		nil, nil, nil, nil, nil, nil, cache, cfg, nil, nil, nil,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, settings, nil,
	)
	returned := make(chan struct{})
	go func() {
		defer close(returned)
		persistCyberSessionBlockPlan(gateway, cyberSessionBlockWritePlan{scopeKey: "scope", keys: []string{"key"}}, 50*time.Millisecond)
	}()
	select {
	case <-cache.started:
	case <-time.After(2 * time.Second):
		t.Fatal("block write was not attempted")
	}
	select {
	case <-returned:
	case <-time.After(time.Second):
		t.Fatal("handler wait exceeded its deadline while the cache ignored cancellation")
	}
}

func TestPersistCyberSessionBlockPlanWaitsForSuccessfulWrite(t *testing.T) {
	cache := &openAICyberContinuationCache{}
	cfg := &config.Config{}
	settings := service.NewSettingService(&openAIWSCyberSettingRepo{values: map[string]string{
		service.SettingKeyCyberSessionBlockEnabled: "true",
	}}, cfg)
	gateway := service.NewOpenAIGatewayService(
		nil, nil, nil, nil, nil, nil, cache, cfg, nil, nil, nil,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, settings, nil,
	)
	persistCyberSessionBlockPlan(gateway, cyberSessionBlockWritePlan{scopeKey: "scope", keys: []string{"key"}}, time.Second)
	key, err := cache.FindCyberSessionBlocked(context.Background(), []string{"key"})
	require.NoError(t, err)
	require.Equal(t, "key", key, "successful writes must be visible when the wait returns")
	active, err := cache.IsCyberSessionScopeActive(context.Background(), "scope")
	require.NoError(t, err)
	require.True(t, active)
}
