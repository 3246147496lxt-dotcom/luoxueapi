package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestClassifyProxyHealthFreshnessAndStates(t *testing.T) {
	now := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	checked := now.Add(-5 * time.Minute).Unix()
	tests := []struct {
		name     string
		snapshot *ProxyLatencyInfo
		want     ProxyHealthStatus
		reason   string
	}{
		{name: "missing", want: ProxyHealthUnknown, reason: "not_checked"},
		{name: "stale base", snapshot: &ProxyLatencyInfo{Success: true, UpdatedAt: now.Add(-11 * time.Minute), QualityCheckedAt: &checked, QualityStatus: "healthy"}, want: ProxyHealthUnknown, reason: "connectivity_stale"},
		{name: "base failed", snapshot: &ProxyLatencyInfo{Success: false, UpdatedAt: now.Add(-time.Minute), QualityCheckedAt: &checked}, want: ProxyHealthFailed, reason: "connectivity_failed"},
		{name: "quality stale", snapshot: &ProxyLatencyInfo{Success: true, UpdatedAt: now.Add(-time.Minute), QualityStatus: "healthy"}, want: ProxyHealthUnknown, reason: "quality_stale"},
		{name: "healthy", snapshot: &ProxyLatencyInfo{Success: true, UpdatedAt: now.Add(-time.Minute), QualityCheckedAt: &checked, QualityStatus: "healthy"}, want: ProxyHealthHealthy, reason: "all_supported_targets_passed"},
		{name: "warning", snapshot: &ProxyLatencyInfo{Success: true, UpdatedAt: now.Add(-time.Minute), QualityCheckedAt: &checked, QualityStatus: "warn"}, want: ProxyHealthDegraded, reason: "supported_target_warning_or_failure"},
		{name: "target failure with base pass", snapshot: &ProxyLatencyInfo{Success: true, UpdatedAt: now.Add(-time.Minute), QualityCheckedAt: &checked, QualityStatus: "failed"}, want: ProxyHealthDegraded, reason: "supported_target_warning_or_failure"},
		{name: "challenge", snapshot: &ProxyLatencyInfo{Success: true, UpdatedAt: now.Add(-time.Minute), QualityCheckedAt: &checked, QualityStatus: "challenge"}, want: ProxyHealthSuspectedRestricted, reason: "challenge_detected"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, reason, _, _ := classifyProxyHealth(test.snapshot, now, 10*time.Minute, time.Hour)
			require.Equal(t, test.want, got)
			require.Equal(t, test.reason, reason)
		})
	}
}

func TestProxyLifecycleUsesPerProxyWarningDays(t *testing.T) {
	now := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	expires := now.Add(48 * time.Hour)
	require.Equal(t, ProxyLifecycleActive, proxyLifecycle(Proxy{Status: StatusActive, ExpiresAt: &expires, ExpiryWarnDays: 1}, now))
	require.Equal(t, ProxyLifecycleExpiringSoon, proxyLifecycle(Proxy{Status: StatusActive, ExpiresAt: &expires, ExpiryWarnDays: 3}, now))
	expired := now.Add(-time.Second)
	require.Equal(t, ProxyLifecycleExpired, proxyLifecycle(Proxy{Status: StatusActive, ExpiresAt: &expired, ExpiryWarnDays: 3}, now))
	require.Equal(t, ProxyLifecycleInactive, proxyLifecycle(Proxy{Status: "inactive"}, now))
}

func TestProxyHealthInactiveAndExpiredCannotUseHealthySnapshot(t *testing.T) {
	now := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	checked := now.Unix()
	snapshot := &ProxyLatencyInfo{Success: true, UpdatedAt: now, QualityCheckedAt: &checked, QualityStatus: "healthy"}
	svc := &ProxyHealthService{cfg: defaultProxyHealthRuntimeConfig()}

	inactive := svc.buildItem(ProxyWithAccountImpact{Proxy: Proxy{ID: 1, Status: "inactive"}}, snapshot, now)
	require.Equal(t, ProxyLifecycleInactive, inactive.Lifecycle)
	require.Equal(t, ProxyHealthUnknown, inactive.Health)
	require.Equal(t, "proxy_inactive", inactive.HealthReason)

	expires := now.Add(-time.Second)
	expired := svc.buildItem(ProxyWithAccountImpact{Proxy: Proxy{ID: 2, Status: StatusActive, ExpiresAt: &expires}}, snapshot, now)
	require.Equal(t, ProxyLifecycleExpired, expired.Lifecycle)
	require.Equal(t, ProxyHealthUnknown, expired.Health)
	require.Equal(t, "proxy_expired", expired.HealthReason)
}

func TestBuildProxyPlatformImpactsMarksCoverage(t *testing.T) {
	items := buildProxyPlatformImpacts(map[string]int64{
		"openai": 3, "anthropic": 2, "antigravity": 1,
	}, &ProxyLatencyInfo{QualityItems: []ProxyQualityCheckItem{
		{Target: "openai", Status: "pass"},
		{Target: "anthropic", Status: "challenge"},
	}})
	require.Len(t, items, 3)
	byPlatform := make(map[string]ProxyHealthPlatformImpact)
	for _, item := range items {
		byPlatform[item.Platform] = item
	}
	require.Equal(t, "pass", byPlatform["openai"].Status)
	require.Equal(t, "challenge", byPlatform["anthropic"].Status)
	require.Equal(t, "uncovered", byPlatform["antigravity"].Coverage)
	require.Equal(t, "uncovered", byPlatform["antigravity"].Status)
}

func TestProxyHealthItemJSONNeverContainsProxyCredentialsOrURL(t *testing.T) {
	now := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	qualityChecked := now.Unix()
	svc := &ProxyHealthService{cfg: defaultProxyHealthRuntimeConfig()}
	item := svc.buildItem(ProxyWithAccountImpact{
		Proxy: Proxy{ID: 7, Name: "edge-a", Protocol: "socks5", Host: "private.example", Port: 1080, Username: "operator", Password: "secret-value", Status: StatusActive},
	}, &ProxyLatencyInfo{Success: true, UpdatedAt: now, QualityCheckedAt: &qualityChecked, QualityStatus: "healthy", IPAddress: "203.0.113.4"}, now)
	payload, err := json.Marshal(item)
	require.NoError(t, err)
	jsonText := string(payload)
	for _, forbidden := range []string{"username", "password", "proxy_url", "private.example", "1080", "operator", "socks5://", "secret-value"} {
		require.False(t, strings.Contains(jsonText, forbidden), jsonText)
	}
}

type concurrencyProbeStub struct {
	mu        sync.Mutex
	current   int
	max       int
	qualityID []int64
}

func (p *concurrencyProbeStub) TestProxy(_ context.Context, id int64) (*ProxyTestResult, error) {
	return &ProxyTestResult{Success: true}, nil
}

func (p *concurrencyProbeStub) CheckProxyQuality(_ context.Context, id int64) (*ProxyQualityCheckResult, error) {
	p.mu.Lock()
	p.current++
	if p.current > p.max {
		p.max = p.current
	}
	p.qualityID = append(p.qualityID, id)
	p.mu.Unlock()
	time.Sleep(5 * time.Millisecond)
	p.mu.Lock()
	p.current--
	p.mu.Unlock()
	return &ProxyQualityCheckResult{ProxyID: id}, nil
}

func TestProxyHealthProbeIDsHonorsConcurrencyLimit(t *testing.T) {
	prober := &concurrencyProbeStub{}
	svc := &ProxyHealthService{
		prober:   prober,
		cfg:      ProxyHealthRuntimeConfig{MaxConcurrency: 3},
		inflight: make(map[int64]struct{}),
	}
	svc.probeIDs(context.Background(), []int64{1, 2, 3, 4, 5, 6, 7}, true, 0)
	require.LessOrEqual(t, prober.max, 3)
	require.Len(t, prober.qualityID, 7)
}

type healthProxyRepoStub struct {
	ProxyRepository
	mu      sync.Mutex
	active  []Proxy
	impacts []ProxyWithAccountImpact
	proxy   *Proxy
}

func (r *healthProxyRepoStub) ListActive(context.Context) ([]Proxy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Proxy(nil), r.active...), nil
}

func (r *healthProxyRepoStub) ListAllWithAccountImpact(context.Context) ([]ProxyWithAccountImpact, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]ProxyWithAccountImpact(nil), r.impacts...), nil
}

func (r *healthProxyRepoStub) GetByID(context.Context, int64) (*Proxy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.proxy == nil {
		return nil, ErrProxyNotFound
	}
	copy := *r.proxy
	return &copy, nil
}

func (r *healthProxyRepoStub) Update(_ context.Context, proxy *Proxy) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copy := *proxy
	r.proxy = &copy
	return nil
}

type healthCacheStub struct {
	ProxyLatencyCache
	mu        sync.Mutex
	snapshots map[int64]*ProxyLatencyInfo
	err       error
	deleted   []int64
}

func (c *healthCacheStub) GetProxyLatencies(context.Context, []int64) (map[int64]*ProxyLatencyInfo, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return nil, c.err
	}
	out := make(map[int64]*ProxyLatencyInfo, len(c.snapshots))
	for id, snapshot := range c.snapshots {
		out[id] = snapshot
	}
	return out, nil
}

func (c *healthCacheStub) DeleteProxyLatency(_ context.Context, id int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.deleted = append(c.deleted, id)
	delete(c.snapshots, id)
	return c.err
}

type blockingHealthProber struct {
	mu      sync.Mutex
	calls   []int64
	started chan int64
	release chan struct{}
}

func (p *blockingHealthProber) TestProxy(ctx context.Context, id int64) (*ProxyTestResult, error) {
	return p.wait(ctx, id)
}

func (p *blockingHealthProber) CheckProxyQuality(ctx context.Context, id int64) (*ProxyQualityCheckResult, error) {
	_, err := p.wait(ctx, id)
	return &ProxyQualityCheckResult{ProxyID: id}, err
}

func (p *blockingHealthProber) wait(ctx context.Context, id int64) (*ProxyTestResult, error) {
	p.mu.Lock()
	p.calls = append(p.calls, id)
	p.mu.Unlock()
	if p.started != nil {
		select {
		case p.started <- id:
		default:
		}
	}
	if p.release == nil {
		return &ProxyTestResult{Success: true}, nil
	}
	select {
	case <-p.release:
		return &ProxyTestResult{Success: true}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (p *blockingHealthProber) callCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.calls)
}

func testProxyHealthConfig() ProxyHealthRuntimeConfig {
	return ProxyHealthRuntimeConfig{
		Enabled: true, BaseInterval: time.Minute, QualityInterval: time.Minute,
		BaseFreshness: 2 * time.Minute, QualityFreshness: 2 * time.Minute,
		MaxConcurrency: 3,
	}
}

func TestProxyHealthDisabledDoesNotStartCompleteProbeScheduler(t *testing.T) {
	repo := &healthProxyRepoStub{active: []Proxy{{ID: 1, Status: StatusActive}}}
	prober := &blockingHealthProber{}
	svc := &ProxyHealthService{
		proxyRepo: repo,
		prober:    prober,
		cfg:       defaultProxyHealthRuntimeConfig(),
		inflight:  make(map[int64]struct{}),
	}

	svc.Start()
	svc.Stop()

	require.Nil(t, svc.cancel)
	require.Zero(t, prober.callCount())
}

func TestProxyHealthProbeCancellationReleasesInflight(t *testing.T) {
	prober := &blockingHealthProber{started: make(chan int64, 1), release: make(chan struct{})}
	svc := &ProxyHealthService{prober: prober, cfg: testProxyHealthConfig(), inflight: make(map[int64]struct{})}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	start := time.Now()
	svc.probeIDs(ctx, []int64{1}, true, 0)
	require.Less(t, time.Since(start), time.Second)
	require.ErrorIs(t, ctx.Err(), context.DeadlineExceeded)
	svc.inflightMu.Lock()
	require.Empty(t, svc.inflight)
	svc.inflightMu.Unlock()
}

func TestProxyHealthSameInstancePreventsDuplicateProbe(t *testing.T) {
	prober := &blockingHealthProber{started: make(chan int64, 1), release: make(chan struct{})}
	svc := &ProxyHealthService{prober: prober, cfg: testProxyHealthConfig(), inflight: make(map[int64]struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		svc.probeIDs(ctx, []int64{7}, true, 0)
	}()
	select {
	case <-prober.started:
	case <-time.After(time.Second):
		t.Fatal("first probe did not start")
	}
	svc.probeIDs(context.Background(), []int64{7}, true, 0)
	require.Equal(t, 1, prober.callCount())
	cancel()
	<-done
}

func TestProxyHealthLeaderLockPreventsCrossInstanceProbe(t *testing.T) {
	now := time.Now()
	repo := &healthProxyRepoStub{active: []Proxy{{ID: 1, Status: StatusActive}}}
	cache := &healthCacheStub{snapshots: map[int64]*ProxyLatencyInfo{}}
	lock := &fakeLeaderLockCache{}
	firstProber := &blockingHealthProber{started: make(chan int64, 1), release: make(chan struct{})}
	secondProber := &blockingHealthProber{}
	first := &ProxyHealthService{proxyRepo: repo, cache: cache, prober: firstProber, lockCache: lock, cfg: testProxyHealthConfig(), now: func() time.Time { return now }, instanceID: "first", inflight: make(map[int64]struct{})}
	second := &ProxyHealthService{proxyRepo: repo, cache: cache, prober: secondProber, lockCache: lock, cfg: testProxyHealthConfig(), now: func() time.Time { return now }, instanceID: "second", inflight: make(map[int64]struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		first.runScheduledCycle(ctx)
	}()
	select {
	case <-firstProber.started:
	case <-time.After(time.Second):
		t.Fatal("leader probe did not start")
	}
	second.runScheduledCycle(context.Background())
	require.Zero(t, secondProber.callCount())
	cancel()
	<-done
}

func TestProxyHealthSnapshotFailureReturnsPartialData(t *testing.T) {
	now := time.Now()
	repo := &healthProxyRepoStub{impacts: []ProxyWithAccountImpact{{
		Proxy:  Proxy{ID: 1, Name: "edge", Status: StatusActive},
		Impact: ProxyAccountImpact{AccountCount: 2, ActiveAccountCount: 1},
	}}}
	svc := &ProxyHealthService{proxyRepo: repo, cache: &healthCacheStub{err: errors.New("redis down")}, cfg: testProxyHealthConfig(), now: func() time.Time { return now }}
	result, err := svc.List(context.Background(), ProxyHealthFilter{})
	require.NoError(t, err)
	require.Equal(t, ProxyHealthDataPartial, result.DataStatus)
	require.Len(t, result.Items, 1)
	require.Equal(t, ProxyHealthUnknown, result.Items[0].Health)
}

func TestProxyHealthSnapshotFailureSkipsBackgroundProbeBurst(t *testing.T) {
	now := time.Now()
	repo := &healthProxyRepoStub{active: []Proxy{{ID: 1, Status: StatusActive}}}
	prober := &blockingHealthProber{}
	svc := &ProxyHealthService{
		proxyRepo: repo, cache: &healthCacheStub{err: errors.New("redis down")}, prober: prober,
		cfg: testProxyHealthConfig(), now: func() time.Time { return now }, instanceID: "single", inflight: make(map[int64]struct{}),
	}
	svc.runScheduledCycle(context.Background())
	require.Zero(t, prober.callCount())
}

func TestProxyHealthSchedulerSkipsInactiveAndExpired(t *testing.T) {
	now := time.Now()
	expired := now.Add(-time.Minute)
	repo := &healthProxyRepoStub{active: []Proxy{
		{ID: 1, Status: StatusActive},
		{ID: 2, Status: StatusActive, ExpiresAt: &expired},
		{ID: 3, Status: "inactive"},
	}}
	prober := &blockingHealthProber{}
	svc := &ProxyHealthService{proxyRepo: repo, cache: &healthCacheStub{snapshots: map[int64]*ProxyLatencyInfo{}}, prober: prober, cfg: testProxyHealthConfig(), now: func() time.Time { return now }, instanceID: "single", inflight: make(map[int64]struct{})}
	svc.runScheduledCycle(context.Background())
	require.Equal(t, 1, prober.callCount())
	require.Equal(t, []int64{1}, prober.calls)
}

func TestProxyConfigUpdateInvalidatesHealthSnapshot(t *testing.T) {
	repo := &healthProxyRepoStub{proxy: &Proxy{ID: 9, Name: "edge", Protocol: "http", Host: "old.example", Port: 8080, Status: StatusActive, FallbackMode: FallbackModeNone}}
	cache := &healthCacheStub{snapshots: map[int64]*ProxyLatencyInfo{9: {Success: true, UpdatedAt: time.Now()}}}
	admin := &adminServiceImpl{proxyRepo: repo, proxyLatencyCache: cache}
	_, err := admin.UpdateProxy(context.Background(), 9, &UpdateProxyInput{Host: "new.example", FallbackMode: FallbackModeNone})
	require.NoError(t, err)
	require.Equal(t, []int64{9}, cache.deleted)
}

func TestProxyProbeQueueSpreadsFastJobs(t *testing.T) {
	prober := &blockingHealthProber{}
	cfg := testProxyHealthConfig()
	cfg.MaxConcurrency = 1
	svc := &ProxyHealthService{prober: prober, cfg: cfg, inflight: make(map[int64]struct{})}
	start := time.Now()
	svc.probeIDs(context.Background(), []int64{1, 2, 3, 4}, true, 10*time.Millisecond)
	require.GreaterOrEqual(t, time.Since(start), 25*time.Millisecond)
	require.Equal(t, 4, prober.callCount())
}
