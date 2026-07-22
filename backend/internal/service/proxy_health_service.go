package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var (
	ErrProxyNotOperational  = infraerrors.Conflict("PROXY_NOT_OPERATIONAL", "proxy is not active or has expired")
	ErrProxyProbeInProgress = infraerrors.Conflict("PROXY_PROBE_IN_PROGRESS", "proxy health probe is already in progress")
)

var errProxyHealthCacheUnavailable = errors.New("proxy health snapshot cache unavailable")

type ProxyHealthStatus string

const (
	ProxyHealthHealthy             ProxyHealthStatus = "healthy"
	ProxyHealthDegraded            ProxyHealthStatus = "degraded"
	ProxyHealthSuspectedRestricted ProxyHealthStatus = "suspected_restricted"
	ProxyHealthFailed              ProxyHealthStatus = "failed"
	ProxyHealthUnknown             ProxyHealthStatus = "unknown"
)

const (
	ProxyLifecycleActive       = "active"
	ProxyLifecycleInactive     = "inactive"
	ProxyLifecycleExpired      = "expired"
	ProxyLifecycleExpiringSoon = "expiring_soon"

	ProxyHealthDataComplete = "complete"
	ProxyHealthDataPartial  = "partial"
)

type ProxyHealthFilter struct {
	Health    string
	Lifecycle string
	Protocol  string
}

type ProxyHealthSummary struct {
	Total               int64 `json:"total"`
	Operational         int64 `json:"operational"`
	Inactive            int64 `json:"inactive"`
	Expired             int64 `json:"expired"`
	ExpiringSoon        int64 `json:"expiring_soon"`
	Healthy             int64 `json:"healthy"`
	Degraded            int64 `json:"degraded"`
	SuspectedRestricted int64 `json:"suspected_restricted"`
	Failed              int64 `json:"failed"`
	Unknown             int64 `json:"unknown"`
	AffectedAccounts    int64 `json:"affected_accounts"`
}

type ProxyHealthPlatformImpact struct {
	Platform     string `json:"platform"`
	AccountCount int64  `json:"account_count"`
	Coverage     string `json:"coverage"` // supported/uncovered
	Status       string `json:"status"`   // pass/warn/fail/challenge/unknown/uncovered
}

type ProxyHealthTarget struct {
	Target     string `json:"target"`
	Status     string `json:"status"`
	HTTPStatus int    `json:"http_status,omitempty"`
	LatencyMs  int64  `json:"latency_ms,omitempty"`
}

// ProxyHealthItem is deliberately a safe projection. It must not grow fields
// containing proxy Host, Port, Username, Password, URL, raw probe errors or
// request headers. Those values belong only to the proxy management boundary.
type ProxyHealthItem struct {
	ID                    int64                       `json:"id"`
	Name                  string                      `json:"name"`
	Protocol              string                      `json:"protocol"`
	Lifecycle             string                      `json:"lifecycle"`
	Health                ProxyHealthStatus           `json:"health"`
	HealthReason          string                      `json:"health_reason"`
	AccountCount          int64                       `json:"account_count"`
	ActiveAccountCount    int64                       `json:"active_account_count"`
	Platforms             []ProxyHealthPlatformImpact `json:"platforms"`
	Targets               []ProxyHealthTarget         `json:"targets,omitempty"`
	LatencyMs             *int64                      `json:"latency_ms,omitempty"`
	ExitIP                string                      `json:"exit_ip,omitempty"`
	Country               string                      `json:"country,omitempty"`
	CountryCode           string                      `json:"country_code,omitempty"`
	Region                string                      `json:"region,omitempty"`
	City                  string                      `json:"city,omitempty"`
	ExpiresAt             *time.Time                  `json:"expires_at,omitempty"`
	ExpiryWarnDays        int                         `json:"expiry_warn_days"`
	ConnectivityCheckedAt *time.Time                  `json:"connectivity_checked_at,omitempty"`
	QualityCheckedAt      *time.Time                  `json:"quality_checked_at,omitempty"`
	ConnectivityStale     bool                        `json:"connectivity_stale"`
	QualityStale          bool                        `json:"quality_stale"`
	QualityScore          *int                        `json:"quality_score,omitempty"`
	QualityGrade          string                      `json:"quality_grade,omitempty"`
}

type ProxyHealthListResponse struct {
	Summary     ProxyHealthSummary `json:"summary"`
	Items       []ProxyHealthItem  `json:"items"`
	GeneratedAt time.Time          `json:"generated_at"`
	DataStatus  string             `json:"data_status"`
}

type proxyHealthImpactRepository interface {
	ListAllWithAccountImpact(ctx context.Context) ([]ProxyWithAccountImpact, error)
}

type proxyHealthProber interface {
	TestProxy(ctx context.Context, id int64) (*ProxyTestResult, error)
	CheckProxyQuality(ctx context.Context, id int64) (*ProxyQualityCheckResult, error)
}

type ProxyHealthRuntimeConfig struct {
	Enabled          bool
	BaseInterval     time.Duration
	QualityInterval  time.Duration
	BaseFreshness    time.Duration
	QualityFreshness time.Duration
	MaxConcurrency   int
}

func defaultProxyHealthRuntimeConfig() ProxyHealthRuntimeConfig {
	return ProxyHealthRuntimeConfig{
		Enabled:          false,
		BaseInterval:     5 * time.Minute,
		QualityInterval:  30 * time.Minute,
		BaseFreshness:    10 * time.Minute,
		QualityFreshness: 60 * time.Minute,
		MaxConcurrency:   3,
	}
}

func proxyHealthRuntimeConfigFromConfig(cfg *config.Config) ProxyHealthRuntimeConfig {
	out := defaultProxyHealthRuntimeConfig()
	if cfg == nil {
		return out
	}
	in := cfg.Ops.ProxyHealth
	out.Enabled = in.Enabled
	if in.BaseInterval > 0 {
		out.BaseInterval = in.BaseInterval
	}
	if in.QualityInterval > 0 {
		out.QualityInterval = in.QualityInterval
	}
	if in.BaseFreshness > 0 {
		out.BaseFreshness = in.BaseFreshness
	}
	if in.QualityFreshness > 0 {
		out.QualityFreshness = in.QualityFreshness
	}
	if in.MaxConcurrency > 0 {
		out.MaxConcurrency = in.MaxConcurrency
	}
	return out
}

type ProxyHealthService struct {
	proxyRepo  ProxyRepository
	cache      ProxyLatencyCache
	prober     proxyHealthProber
	opsService *OpsService
	lockCache  LeaderLockCache
	db         *sql.DB
	cfg        ProxyHealthRuntimeConfig
	now        func() time.Time
	instanceID string

	startOnce sync.Once
	stopOnce  sync.Once
	cancel    context.CancelFunc
	wg        sync.WaitGroup

	inflightMu sync.Mutex
	inflight   map[int64]struct{}
}

func NewProxyHealthService(
	adminService AdminService,
	proxyRepo ProxyRepository,
	cache ProxyLatencyCache,
	opsService *OpsService,
	lockCache LeaderLockCache,
	db *sql.DB,
	cfg *config.Config,
) *ProxyHealthService {
	return &ProxyHealthService{
		proxyRepo:  proxyRepo,
		cache:      cache,
		prober:     adminService,
		opsService: opsService,
		lockCache:  lockCache,
		db:         db,
		cfg:        proxyHealthRuntimeConfigFromConfig(cfg),
		now:        time.Now,
		instanceID: fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano()),
		inflight:   make(map[int64]struct{}),
	}
}

func (s *ProxyHealthService) Start() {
	if s == nil || s.proxyRepo == nil || s.prober == nil || !s.cfg.Enabled {
		return
	}
	s.startOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		s.cancel = cancel
		s.wg.Add(1)
		go s.run(ctx)
	})
}

func (s *ProxyHealthService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		if s.cancel != nil {
			s.cancel()
		}
	})
	s.wg.Wait()
}

func (s *ProxyHealthService) run(ctx context.Context) {
	defer s.wg.Done()
	baseTicker := time.NewTicker(s.cfg.BaseInterval)
	defer baseTicker.Stop()

	// A single leader-elected cycle owns both base and quality work. Keeping one
	// distributed lock prevents different replicas from winning separate base and
	// quality locks and probing the same proxy concurrently.
	s.runScheduledCycle(ctx)
	for {
		select {
		case <-baseTicker.C:
			s.runScheduledCycle(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (s *ProxyHealthService) monitoringEnabled(ctx context.Context) bool {
	return s.opsService == nil || s.opsService.IsMonitoringEnabled(ctx)
}

func (s *ProxyHealthService) runScheduledCycle(parent context.Context) {
	if !s.monitoringEnabled(parent) {
		return
	}
	cycleTimeout := s.cfg.QualityInterval * 4 / 5
	if cycleTimeout < 30*time.Second {
		cycleTimeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(parent, cycleTimeout)
	defer cancel()

	release, ok := tryAcquireSingletonLeaderLock(ctx, s.lockCache, s.db, "ops:proxy-health:probe", s.instanceID, cycleTimeout+time.Minute)
	if !ok {
		return
	}
	defer release()

	proxies, err := s.proxyRepo.ListActive(ctx)
	if err != nil {
		log.Printf("[ProxyHealth] list active proxies failed: %v", err)
		return
	}
	ids := make([]int64, 0, len(proxies))
	for i := range proxies {
		if proxies[i].IsActive() && !proxies[i].IsExpired(s.now()) {
			ids = append(ids, proxies[i].ID)
		}
	}
	snapshots, err := s.loadSnapshots(ctx, ids)
	if err != nil {
		// Without a shared snapshot we cannot tell which probes are due and cannot
		// persist the result. Degrade the read API to data_status=partial and avoid
		// hammering providers every base interval until Redis recovers.
		log.Printf("[ProxyHealth] load snapshots failed; skip probe cycle: %v", err)
		return
	}

	qualityDue := make([]int64, 0, len(ids))
	baseDue := make([]int64, 0, len(ids))
	now := s.now()
	for _, id := range ids {
		snapshot := snapshots[id]
		if qualityProbeDue(snapshot, now, s.cfg.QualityInterval) {
			qualityDue = append(qualityDue, id)
			continue
		}
		if snapshot == nil || snapshot.UpdatedAt.IsZero() || now.Sub(snapshot.UpdatedAt) >= s.cfg.BaseInterval {
			baseDue = append(baseDue, id)
		}
	}
	// Quality probes also refresh base connectivity, so each proxy appears in at
	// most one queue per cycle. Process the more informative queue first.
	qualityWindow := minProxyHealthDuration(s.cfg.QualityInterval/3, 10*time.Minute)
	baseWindow := minProxyHealthDuration(s.cfg.BaseInterval/2, 2*time.Minute)
	s.probeIDs(ctx, qualityDue, true, proxyProbeDispatchSpacing(len(qualityDue), s.cfg.MaxConcurrency, qualityWindow))
	s.probeIDs(ctx, baseDue, false, proxyProbeDispatchSpacing(len(baseDue), s.cfg.MaxConcurrency, baseWindow))
}

func qualityProbeDue(snapshot *ProxyLatencyInfo, now time.Time, interval time.Duration) bool {
	if snapshot == nil || snapshot.QualityCheckedAt == nil || *snapshot.QualityCheckedAt <= 0 {
		return true
	}
	return now.Sub(time.Unix(*snapshot.QualityCheckedAt, 0)) >= interval
}

func (s *ProxyHealthService) probeIDs(ctx context.Context, ids []int64, quality bool, dispatchSpacing time.Duration) {
	if len(ids) == 0 {
		return
	}
	workers := s.cfg.MaxConcurrency
	if workers < 1 {
		workers = 1
	}
	if workers > len(ids) {
		workers = len(ids)
	}
	jobs := make(chan int64)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for id := range jobs {
				if ctx.Err() != nil {
					return
				}
				if !s.claimProbe(id) {
					continue
				}
				if quality {
					_, _ = s.prober.CheckProxyQuality(ctx, id)
				} else {
					_, _ = s.prober.TestProxy(ctx, id)
				}
				s.releaseProbe(id)
			}
		}()
	}
	for i, id := range ids {
		if i >= workers && dispatchSpacing > 0 {
			timer := time.NewTimer(dispatchSpacing)
			select {
			case <-timer.C:
			case <-ctx.Done():
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				close(jobs)
				wg.Wait()
				return
			}
		}
		select {
		case jobs <- id:
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return
		}
	}
	close(jobs)
	wg.Wait()
}

func proxyProbeDispatchSpacing(total, workers int, window time.Duration) time.Duration {
	if workers < 1 {
		workers = 1
	}
	remaining := total - workers
	if remaining <= 0 || window <= 0 {
		return 0
	}
	spacing := window / time.Duration(remaining)
	if spacing > 10*time.Second {
		return 10 * time.Second
	}
	if spacing < 100*time.Millisecond {
		return 100 * time.Millisecond
	}
	return spacing
}

func minProxyHealthDuration(left, right time.Duration) time.Duration {
	if left < right {
		return left
	}
	return right
}

func (s *ProxyHealthService) claimProbe(id int64) bool {
	s.inflightMu.Lock()
	defer s.inflightMu.Unlock()
	if _, exists := s.inflight[id]; exists {
		return false
	}
	s.inflight[id] = struct{}{}
	return true
}

func (s *ProxyHealthService) releaseProbe(id int64) {
	s.inflightMu.Lock()
	delete(s.inflight, id)
	s.inflightMu.Unlock()
}

func (s *ProxyHealthService) ProbeNow(ctx context.Context, id int64) (*ProxyHealthItem, error) {
	if err := s.requireAvailable(ctx); err != nil {
		return nil, err
	}
	proxy, err := s.proxyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !proxy.IsActive() || proxy.IsExpired(s.now()) {
		return nil, ErrProxyNotOperational
	}
	if !s.claimProbe(id) {
		return nil, ErrProxyProbeInProgress
	}
	defer s.releaseProbe(id)
	if _, err := s.prober.CheckProxyQuality(ctx, id); err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *ProxyHealthService) List(ctx context.Context, filter ProxyHealthFilter) (*ProxyHealthListResponse, error) {
	if err := s.requireAvailable(ctx); err != nil {
		return nil, err
	}
	if err := validateProxyHealthFilter(filter); err != nil {
		return nil, err
	}
	proxies, err := s.listProxyImpacts(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(proxies))
	for i := range proxies {
		ids = append(ids, proxies[i].ID)
	}
	snapshots, cacheErr := s.loadSnapshots(ctx, ids)
	dataStatus := ProxyHealthDataComplete
	if cacheErr != nil {
		dataStatus = ProxyHealthDataPartial
		snapshots = map[int64]*ProxyLatencyInfo{}
	}

	now := s.now()
	response := &ProxyHealthListResponse{
		Items:       make([]ProxyHealthItem, 0, len(proxies)),
		GeneratedAt: now,
		DataStatus:  dataStatus,
	}
	for i := range proxies {
		item := s.buildItem(proxies[i], snapshots[proxies[i].ID], now)
		accumulateProxyHealthSummary(&response.Summary, item)
		if proxyHealthMatchesFilter(item, filter) {
			response.Items = append(response.Items, item)
		}
	}
	sort.SliceStable(response.Items, func(i, j int) bool {
		left, right := response.Items[i], response.Items[j]
		if proxyHealthSeverity(left) != proxyHealthSeverity(right) {
			return proxyHealthSeverity(left) > proxyHealthSeverity(right)
		}
		if left.ActiveAccountCount != right.ActiveAccountCount {
			return left.ActiveAccountCount > right.ActiveAccountCount
		}
		return left.ID > right.ID
	})
	return response, nil
}

func validateProxyHealthFilter(filter ProxyHealthFilter) error {
	if value := strings.ToLower(strings.TrimSpace(filter.Health)); value != "" {
		switch ProxyHealthStatus(value) {
		case ProxyHealthHealthy, ProxyHealthDegraded, ProxyHealthSuspectedRestricted, ProxyHealthFailed, ProxyHealthUnknown:
		default:
			return infraerrors.BadRequest("PROXY_HEALTH_FILTER_INVALID", "invalid proxy health filter")
		}
	}
	if value := strings.ToLower(strings.TrimSpace(filter.Lifecycle)); value != "" {
		switch value {
		case ProxyLifecycleActive, ProxyLifecycleInactive, ProxyLifecycleExpired, ProxyLifecycleExpiringSoon:
		default:
			return infraerrors.BadRequest("PROXY_LIFECYCLE_FILTER_INVALID", "invalid proxy lifecycle filter")
		}
	}
	return nil
}

func (s *ProxyHealthService) Get(ctx context.Context, id int64) (*ProxyHealthItem, error) {
	response, err := s.List(ctx, ProxyHealthFilter{})
	if err != nil {
		return nil, err
	}
	for i := range response.Items {
		if response.Items[i].ID == id {
			item := response.Items[i]
			return &item, nil
		}
	}
	return nil, ErrProxyNotFound
}

func (s *ProxyHealthService) requireAvailable(ctx context.Context) error {
	if s == nil || s.proxyRepo == nil {
		return fmt.Errorf("proxy health service unavailable")
	}
	if s.opsService != nil {
		return s.opsService.RequireMonitoringEnabled(ctx)
	}
	return nil
}

func (s *ProxyHealthService) loadSnapshots(ctx context.Context, ids []int64) (map[int64]*ProxyLatencyInfo, error) {
	if len(ids) == 0 {
		return map[int64]*ProxyLatencyInfo{}, nil
	}
	if s.cache == nil {
		return map[int64]*ProxyLatencyInfo{}, errProxyHealthCacheUnavailable
	}
	return s.cache.GetProxyLatencies(ctx, ids)
}

func (s *ProxyHealthService) listProxyImpacts(ctx context.Context) ([]ProxyWithAccountImpact, error) {
	if repo, ok := s.proxyRepo.(proxyHealthImpactRepository); ok {
		return repo.ListAllWithAccountImpact(ctx)
	}
	// Compatibility fallback for alternate repository implementations. Production
	// uses the batched path above; this keeps narrow test repositories usable.
	proxies, err := s.proxyRepo.ListAllForFallback(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ProxyWithAccountImpact, 0, len(proxies))
	for i := range proxies {
		accounts, listErr := s.proxyRepo.ListAccountSummariesByProxyID(ctx, proxies[i].ID)
		if listErr != nil {
			return nil, listErr
		}
		impact := ProxyAccountImpact{PlatformCounts: make(map[string]int64)}
		for _, account := range accounts {
			impact.AccountCount++
			// Summary fallback has no status field. Conservatively treat linked
			// accounts as active impact rather than under-reporting an outage.
			impact.ActiveAccountCount++
			impact.PlatformCounts[account.Platform]++
		}
		result = append(result, ProxyWithAccountImpact{Proxy: proxies[i], Impact: impact})
	}
	return result, nil
}

func (s *ProxyHealthService) buildItem(proxy ProxyWithAccountImpact, snapshot *ProxyLatencyInfo, now time.Time) ProxyHealthItem {
	lifecycle := proxyLifecycle(proxy.Proxy, now)
	health, reason, baseStale, qualityStale := classifyProxyHealth(snapshot, now, s.cfg.BaseFreshness, s.cfg.QualityFreshness)
	// Lifecycle remains a separate dimension, but the public definition of
	// healthy explicitly requires an active, non-expired proxy. Never let an old
	// successful snapshot make a disabled/expired resource look usable.
	switch lifecycle {
	case ProxyLifecycleInactive:
		health = ProxyHealthUnknown
		reason = "proxy_inactive"
	case ProxyLifecycleExpired:
		health = ProxyHealthUnknown
		reason = "proxy_expired"
	}
	item := ProxyHealthItem{
		ID:                 proxy.ID,
		Name:               proxy.Name,
		Protocol:           proxy.Protocol,
		Lifecycle:          lifecycle,
		Health:             health,
		HealthReason:       reason,
		AccountCount:       proxy.Impact.AccountCount,
		ActiveAccountCount: proxy.Impact.ActiveAccountCount,
		ExpiresAt:          proxy.ExpiresAt,
		ExpiryWarnDays:     proxy.ExpiryWarnDays,
		ConnectivityStale:  baseStale,
		QualityStale:       qualityStale,
		Platforms:          buildProxyPlatformImpacts(proxy.Impact.PlatformCounts, snapshot),
	}
	if snapshot == nil {
		return item
	}
	item.LatencyMs = snapshot.LatencyMs
	item.ExitIP = snapshot.IPAddress
	item.Country = snapshot.Country
	item.CountryCode = snapshot.CountryCode
	item.Region = snapshot.Region
	item.City = snapshot.City
	item.QualityScore = snapshot.QualityScore
	item.QualityGrade = snapshot.QualityGrade
	if !snapshot.UpdatedAt.IsZero() {
		checked := snapshot.UpdatedAt
		item.ConnectivityCheckedAt = &checked
	}
	if snapshot.QualityCheckedAt != nil && *snapshot.QualityCheckedAt > 0 {
		checked := time.Unix(*snapshot.QualityCheckedAt, 0)
		item.QualityCheckedAt = &checked
	}
	for _, target := range snapshot.QualityItems {
		if target.Target == "base_connectivity" || target.Target == "http_client" {
			continue
		}
		item.Targets = append(item.Targets, ProxyHealthTarget{
			Target: target.Target, Status: target.Status,
			HTTPStatus: target.HTTPStatus, LatencyMs: target.LatencyMs,
		})
	}
	return item
}

func proxyLifecycle(proxy Proxy, now time.Time) string {
	if proxy.IsExpired(now) || proxy.Status == StatusExpired {
		return ProxyLifecycleExpired
	}
	if !proxy.IsActive() {
		return ProxyLifecycleInactive
	}
	if proxy.ExpiresAt != nil && proxy.ExpiryWarnDays > 0 && !proxy.ExpiresAt.After(now.Add(time.Duration(proxy.ExpiryWarnDays)*24*time.Hour)) {
		return ProxyLifecycleExpiringSoon
	}
	return ProxyLifecycleActive
}

func classifyProxyHealth(snapshot *ProxyLatencyInfo, now time.Time, baseFreshness, qualityFreshness time.Duration) (ProxyHealthStatus, string, bool, bool) {
	if snapshot == nil || snapshot.UpdatedAt.IsZero() {
		return ProxyHealthUnknown, "not_checked", true, true
	}
	baseStale := now.Sub(snapshot.UpdatedAt) > baseFreshness
	qualityStale := snapshot.QualityCheckedAt == nil || *snapshot.QualityCheckedAt <= 0 || now.Sub(time.Unix(*snapshot.QualityCheckedAt, 0)) > qualityFreshness
	if baseStale {
		return ProxyHealthUnknown, "connectivity_stale", true, qualityStale
	}
	if !snapshot.Success {
		return ProxyHealthFailed, "connectivity_failed", false, qualityStale
	}
	if qualityStale {
		return ProxyHealthUnknown, "quality_stale", false, true
	}
	switch snapshot.QualityStatus {
	case "healthy":
		return ProxyHealthHealthy, "all_supported_targets_passed", false, false
	case "challenge":
		return ProxyHealthSuspectedRestricted, "challenge_detected", false, false
	case "warn", "failed":
		return ProxyHealthDegraded, "supported_target_warning_or_failure", false, false
	default:
		return ProxyHealthUnknown, "quality_unknown", false, false
	}
}

func buildProxyPlatformImpacts(counts map[string]int64, snapshot *ProxyLatencyInfo) []ProxyHealthPlatformImpact {
	statuses := make(map[string]string)
	if snapshot != nil {
		for _, item := range snapshot.QualityItems {
			statuses[item.Target] = item.Status
		}
	}
	platforms := make([]ProxyHealthPlatformImpact, 0, len(counts))
	for platform, count := range counts {
		target, covered := proxyQualityTargetForPlatform(platform)
		impact := ProxyHealthPlatformImpact{Platform: platform, AccountCount: count}
		if !covered {
			impact.Coverage = "uncovered"
			impact.Status = "uncovered"
		} else {
			impact.Coverage = "supported"
			impact.Status = statuses[target]
			if impact.Status == "" {
				impact.Status = "unknown"
			}
		}
		platforms = append(platforms, impact)
	}
	sort.Slice(platforms, func(i, j int) bool {
		if platforms[i].AccountCount != platforms[j].AccountCount {
			return platforms[i].AccountCount > platforms[j].AccountCount
		}
		return platforms[i].Platform < platforms[j].Platform
	})
	return platforms
}

func proxyQualityTargetForPlatform(platform string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "openai":
		return "openai", true
	case "anthropic":
		return "anthropic", true
	case "gemini":
		return "gemini", true
	default:
		return "", false
	}
}

func accumulateProxyHealthSummary(summary *ProxyHealthSummary, item ProxyHealthItem) {
	summary.Total++
	switch item.Lifecycle {
	case ProxyLifecycleInactive:
		summary.Inactive++
	case ProxyLifecycleExpired:
		summary.Expired++
	default:
		summary.Operational++
		if item.Lifecycle == ProxyLifecycleExpiringSoon {
			summary.ExpiringSoon++
		}
		switch item.Health {
		case ProxyHealthHealthy:
			summary.Healthy++
		case ProxyHealthDegraded:
			summary.Degraded++
		case ProxyHealthSuspectedRestricted:
			summary.SuspectedRestricted++
		case ProxyHealthFailed:
			summary.Failed++
		default:
			summary.Unknown++
		}
	}
	if item.Lifecycle == ProxyLifecycleInactive || item.Lifecycle == ProxyLifecycleExpired || item.Health != ProxyHealthHealthy {
		summary.AffectedAccounts += item.ActiveAccountCount
	}
}

func proxyHealthMatchesFilter(item ProxyHealthItem, filter ProxyHealthFilter) bool {
	if value := strings.ToLower(strings.TrimSpace(filter.Health)); value != "" && value != string(item.Health) {
		return false
	}
	if value := strings.ToLower(strings.TrimSpace(filter.Lifecycle)); value != "" && value != item.Lifecycle {
		return false
	}
	if value := strings.ToLower(strings.TrimSpace(filter.Protocol)); value != "" && value != strings.ToLower(item.Protocol) {
		return false
	}
	return true
}

func proxyHealthSeverity(item ProxyHealthItem) int {
	if item.Lifecycle == ProxyLifecycleExpired {
		return 6
	}
	switch item.Health {
	case ProxyHealthFailed:
		return 5
	case ProxyHealthSuspectedRestricted:
		return 4
	case ProxyHealthDegraded:
		return 3
	case ProxyHealthUnknown:
		return 2
	case ProxyHealthHealthy:
		if item.Lifecycle == ProxyLifecycleExpiringSoon {
			return 1
		}
	}
	return 0
}
