package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

// cnQuotaProber is the small contract needed by the periodic runner.  Keeping
// the runner independent from the concrete HTTP service makes it possible to
// exercise scheduling behaviour without making network requests.
type cnQuotaProber interface {
	QueryUsage(context.Context, int64) (*CNProviderQuotaProbeResult, error)
}

const (
	cnProviderQuotaCheckDefaultInterval = 10 * time.Minute
	cnProviderQuotaProbeConcurrency     = 4
)

// CNProviderBalanceCheckService periodically refreshes Zhipu GLM Coding Plan
// rolling-window quota snapshots.  The historical upstream name is retained
// for compatibility with deployments that already refer to this component as
// the CN balance checker; this Zhipu-focused implementation deliberately does
// not probe pay-as-you-go balances or DeepSeek accounts.
//
// A probe only persists the snapshot.  Scheduling decisions remain in
// EvaluateAccountSchedulingThreshold/RateLimitService so the same threshold
// semantics are used for request-time and background paths.  In particular,
// accounts already paused by a threshold are still probed: their fresh window
// is needed to decide whether the pause should continue.
type CNProviderBalanceCheckService struct {
	accountRepo  AccountRepository
	quotaService cnQuotaProber
	cfg          *config.Config
	interval     time.Duration

	parentCtx    context.Context
	parentCancel context.CancelFunc
	mu           sync.Mutex
	started      bool
	stopped      bool
	wg           sync.WaitGroup
	cycleMu      sync.Mutex
}

// NewCNProviderBalanceCheckService constructs the periodic Zhipu quota
// checker.  The variadic tail intentionally accepts both the current compact
// form
//
//	NewCNProviderBalanceCheckService(repo, quota, cfg, interval)
//
// and the five-argument shape used by older CN-provider builds, where a
// pay-as-you-go balance service occupied the second slot:
//
//	NewCNProviderBalanceCheckService(repo, nil, quota, cfg, interval)
//
// Ignoring unknown arguments keeps upgrades source-compatible while this
// branch only supports Zhipu quota probing.
func NewCNProviderBalanceCheckService(accountRepo AccountRepository, args ...any) *CNProviderBalanceCheckService {
	var (
		quotaService cnQuotaProber
		cfg          *config.Config
		interval     time.Duration
	)
	for _, arg := range args {
		switch value := arg.(type) {
		case cnQuotaProber:
			if value != nil {
				quotaService = value
			}
		case *config.Config:
			cfg = value
		case time.Duration:
			interval = value
		}
	}
	if interval <= 0 {
		interval = cnProviderQuotaCheckDefaultInterval
	}
	parentCtx, parentCancel := context.WithCancel(context.Background())
	return &CNProviderBalanceCheckService{
		accountRepo:  accountRepo,
		quotaService: quotaService,
		cfg:          cfg,
		interval:     interval,
		parentCtx:    parentCtx,
		parentCancel: parentCancel,
	}
}

// ProvideCNProviderBalanceCheckService is the Wire provider.  It uses the
// compact constructor form and is intentionally separate from the existing
// DeepSeek balance checker.
func ProvideCNProviderBalanceCheckService(
	accountRepo AccountRepository,
	quotaService *CNProviderQuotaService,
	cfg *config.Config,
) *CNProviderBalanceCheckService {
	interval := cnProviderQuotaCheckDefaultInterval
	if cfg != nil && cfg.Gateway.CNProviders.BalanceCheckIntervalMinutes > 0 {
		interval = time.Duration(cfg.Gateway.CNProviders.BalanceCheckIntervalMinutes) * time.Minute
	}
	return NewCNProviderBalanceCheckService(accountRepo, quotaService, cfg, interval)
}

// Start launches the ticker only when explicitly enabled in configuration.
// The first cycle is delayed by one interval to avoid a startup request burst;
// RunOnce is available for an explicit/manual cycle.
func (s *CNProviderBalanceCheckService) Start() {
	if s == nil || s.accountRepo == nil || s.quotaService == nil || !s.enabled() {
		return
	}
	s.mu.Lock()
	if s.started || s.stopped {
		s.mu.Unlock()
		return
	}
	s.started = true
	s.wg.Add(1)
	s.mu.Unlock()

	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := s.RunOnce(s.parentCtx); err != nil {
					log.Printf("[CNProviderQuota] cycle failed: %v", err)
				}
			case <-s.parentCtx.Done():
				return
			}
		}
	}()
}

// Stop is idempotent and waits for an in-flight probe cycle to finish.
func (s *CNProviderBalanceCheckService) Stop() {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return
	}
	s.stopped = true
	if s.parentCancel != nil {
		s.parentCancel()
	}
	s.mu.Unlock()
	s.wg.Wait()
}

// RunOnce refreshes all active Zhipu Coding Plan accounts.  Probe failures are
// isolated to the account and do not mutate its prior snapshot.
func (s *CNProviderBalanceCheckService) RunOnce(ctx context.Context) error {
	if s == nil || s.accountRepo == nil || s.quotaService == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	s.cycleMu.Lock()
	defer s.cycleMu.Unlock()

	accounts, err := s.accountRepo.ListByPlatform(ctx, PlatformZhipu)
	if err != nil {
		return fmt.Errorf("list %s accounts: %w", PlatformZhipu, err)
	}

	targets := make([]int64, 0, len(accounts))
	seen := make(map[int64]struct{}, len(accounts))
	for i := range accounts {
		account := &accounts[i]
		if !account.IsActive() || !account.IsZhipu() || !account.IsCodingPlan() {
			continue
		}
		if _, exists := seen[account.ID]; exists {
			continue
		}
		seen[account.ID] = struct{}{}
		targets = append(targets, account.ID)
	}
	if len(targets) == 0 {
		return nil
	}

	sem := make(chan struct{}, cnProviderQuotaProbeConcurrency)
	var wg sync.WaitGroup
	for _, accountID := range targets {
		if err := ctx.Err(); err != nil {
			return err
		}
		accountID := accountID
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()
			s.probeQuota(ctx, accountID)
		}()
	}
	wg.Wait()
	return ctx.Err()
}

// runOnce is retained for compatibility with the original CN checker tests
// and with small internal callers that do not need an error return.
func (s *CNProviderBalanceCheckService) runOnce() {
	if err := s.RunOnce(context.Background()); err != nil {
		log.Printf("[CNProviderQuota] run failed: %v", err)
	}
}

func (s *CNProviderBalanceCheckService) probeQuota(ctx context.Context, accountID int64) {
	if s == nil || s.quotaService == nil {
		return
	}
	result, err := s.quotaService.QueryUsage(ctx, accountID)
	if err != nil {
		log.Printf("[CNProviderQuota] account %d probe failed: %v", accountID, err)
		return
	}
	if result != nil && !result.Success && result.Error != "" {
		log.Printf("[CNProviderQuota] account %d probe error: %s", accountID, result.Error)
	}
}

func (s *CNProviderBalanceCheckService) enabled() bool {
	return s != nil && s.cfg != nil && s.cfg.Gateway.CNProviders.BalanceCheckEnabled && s.interval > 0
}

// ZhipuQuotaCheckService is a semantic alias for callers that prefer the
// provider-specific name introduced by the GLM integration.
type ZhipuQuotaCheckService = CNProviderBalanceCheckService

func NewZhipuQuotaCheckService(accountRepo AccountRepository, quotaService *CNProviderQuotaService, cfg *config.Config, interval time.Duration) *ZhipuQuotaCheckService {
	return NewCNProviderBalanceCheckService(accountRepo, quotaService, cfg, interval)
}

func ProvideZhipuQuotaCheckService(accountRepo AccountRepository, quotaService *CNProviderQuotaService, cfg *config.Config) *ZhipuQuotaCheckService {
	return ProvideCNProviderBalanceCheckService(accountRepo, quotaService, cfg)
}
