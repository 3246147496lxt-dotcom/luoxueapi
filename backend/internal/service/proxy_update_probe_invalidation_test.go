//go:build unit

package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type updatingProxyRepoStub struct {
	*proxyRepoStub
	proxy       *Proxy
	updateCalls int
}

func (s *updatingProxyRepoStub) Create(_ context.Context, proxy *Proxy) error {
	copy := *proxy
	if copy.ID == 0 {
		copy.ID = 9
		proxy.ID = copy.ID
	}
	s.proxy = &copy
	return nil
}

func (s *updatingProxyRepoStub) GetByID(context.Context, int64) (*Proxy, error) {
	copy := *s.proxy
	return &copy, nil
}

func (s *updatingProxyRepoStub) Update(_ context.Context, proxy *Proxy) error {
	s.updateCalls++
	copy := *proxy
	s.proxy = &copy
	return nil
}

type failingProxyExitInfoProber struct {
	mu    sync.Mutex
	calls []string
}

func (p *failingProxyExitInfoProber) ProbeProxy(_ context.Context, proxyURL string) (*ProxyExitInfo, int64, error) {
	p.mu.Lock()
	p.calls = append(p.calls, proxyURL)
	p.mu.Unlock()
	return nil, 0, errors.New("probe unavailable")
}

func (p *failingProxyExitInfoProber) callCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.calls)
}

type autoProbeLatencyCache struct {
	mu        sync.Mutex
	snapshots map[int64]*ProxyLatencyInfo
	deleted   []int64
	writes    chan *ProxyLatencyInfo
}

func newAutoProbeLatencyCache() *autoProbeLatencyCache {
	return &autoProbeLatencyCache{
		snapshots: make(map[int64]*ProxyLatencyInfo),
		writes:    make(chan *ProxyLatencyInfo, 4),
	}
}

func (c *autoProbeLatencyCache) GetProxyLatencies(_ context.Context, ids []int64) (map[int64]*ProxyLatencyInfo, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make(map[int64]*ProxyLatencyInfo, len(ids))
	for _, id := range ids {
		if snapshot := c.snapshots[id]; snapshot != nil {
			copy := *snapshot
			result[id] = &copy
		}
	}
	return result, nil
}

func (c *autoProbeLatencyCache) SetProxyLatency(_ context.Context, id int64, info *ProxyLatencyInfo) error {
	c.mu.Lock()
	copy := *info
	c.snapshots[id] = &copy
	c.mu.Unlock()
	c.writes <- &copy
	return nil
}

func (c *autoProbeLatencyCache) DeleteProxyLatency(_ context.Context, id int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.snapshots, id)
	c.deleted = append(c.deleted, id)
	return nil
}

func waitForAutoProbeSnapshot(t *testing.T, cache *autoProbeLatencyCache) *ProxyLatencyInfo {
	t.Helper()
	select {
	case snapshot := <-cache.writes:
		return snapshot
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for proxy connectivity snapshot")
		return nil
	}
}

func requireLightweightProbeSnapshot(t *testing.T, snapshot *ProxyLatencyInfo) {
	t.Helper()
	require.NotNil(t, snapshot)
	require.False(t, snapshot.Success)
	require.Equal(t, "probe unavailable", snapshot.Message)
	require.Nil(t, snapshot.QualityCheckedAt, "automatic create/update must not run a complete quality probe")
	require.Empty(t, snapshot.QualityItems, "automatic create/update must not persist multi-target quality results")
	require.Empty(t, snapshot.QualityStatus)
}

func TestAdminCreateProxyUsesOnlyLightweightConnectivityProbe(t *testing.T) {
	repo := &updatingProxyRepoStub{proxyRepoStub: &proxyRepoStub{}}
	prober := &failingProxyExitInfoProber{}
	cache := newAutoProbeLatencyCache()
	svc := &adminServiceImpl{proxyRepo: repo, proxyProber: prober, proxyLatencyCache: cache}

	created, err := svc.CreateProxy(context.Background(), &CreateProxyInput{
		Name: "edge", Protocol: "http", Host: "proxy.example", Port: 8080,
	})

	require.NoError(t, err)
	require.Equal(t, int64(9), created.ID)
	requireLightweightProbeSnapshot(t, waitForAutoProbeSnapshot(t, cache))
	require.Equal(t, 1, prober.callCount())
}

func TestAdminUpdateProxyUsesOnlyLightweightConnectivityProbe(t *testing.T) {
	repo := &updatingProxyRepoStub{
		proxyRepoStub: &proxyRepoStub{},
		proxy: &Proxy{
			ID: 9, Protocol: "http", Host: "old.example", Port: 8080,
			Status: StatusActive, FallbackMode: FallbackModeNone,
		},
	}
	prober := &failingProxyExitInfoProber{}
	cache := newAutoProbeLatencyCache()
	qualityChecked := time.Now().Unix()
	cache.snapshots[9] = &ProxyLatencyInfo{
		Success: true, QualityStatus: "healthy", QualityCheckedAt: &qualityChecked,
		QualityItems: []ProxyQualityCheckItem{{Target: "openai", Status: "pass"}},
	}
	svc := &adminServiceImpl{proxyRepo: repo, proxyProber: prober, proxyLatencyCache: cache}

	updated, err := svc.UpdateProxy(context.Background(), 9, &UpdateProxyInput{
		Host: "new.example", FallbackMode: FallbackModeNone,
	})

	require.NoError(t, err)
	require.Equal(t, "new.example", updated.Host)
	requireLightweightProbeSnapshot(t, waitForAutoProbeSnapshot(t, cache))
	require.Equal(t, []int64{9}, cache.deleted)
	require.Equal(t, 1, prober.callCount())
}

func TestBothProxyUpdateServicesUseRepositoryUpdateBoundary(t *testing.T) {
	t.Run("ProxyService", func(t *testing.T) {
		repo := &updatingProxyRepoStub{
			proxyRepoStub: &proxyRepoStub{},
			proxy:         &Proxy{ID: 9, Protocol: "http", Host: "old.example", Port: 8080, Status: StatusActive},
		}
		svc := NewProxyService(repo)
		host := "new.example"

		_, err := svc.Update(context.Background(), 9, UpdateProxyRequest{Host: &host})

		require.NoError(t, err)
		require.Equal(t, 1, repo.updateCalls)
		require.Equal(t, host, repo.proxy.Host)
	})

	t.Run("adminService", func(t *testing.T) {
		repo := &updatingProxyRepoStub{
			proxyRepoStub: &proxyRepoStub{},
			proxy: &Proxy{
				ID:             9,
				Protocol:       "http",
				Host:           "old.example",
				Port:           8080,
				Status:         StatusActive,
				FallbackMode:   FallbackModeNone,
				ExpiryWarnDays: 7,
			},
		}
		svc := &adminServiceImpl{proxyRepo: repo}

		_, err := svc.UpdateProxy(context.Background(), 9, &UpdateProxyInput{
			Host:           "new.example",
			FallbackMode:   FallbackModeNone,
			ExpiryWarnDays: 7,
		})

		require.NoError(t, err)
		require.Equal(t, 1, repo.updateCalls)
		require.Equal(t, "new.example", repo.proxy.Host)
	})
}
