package service

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

// deepSeekBalanceProber is intentionally narrower than the concrete probe
// service.  Keeping the scheduler dependent on this small contract makes the
// pause/recovery policy deterministic in tests and prevents a future provider
// from accidentally entering the DeepSeek-only runner.
type deepSeekBalanceProber interface {
	QueryBalance(context.Context, int64) (*DeepSeekBalanceResult, error)
}

const (
	deepSeekBalanceCheckDefaultInterval = 10 * time.Minute
	deepSeekBalanceCheckMinInterval     = time.Minute
	deepSeekBalanceCheckMaxInterval     = 24 * time.Hour
	deepSeekBalanceCheckConcurrency     = 4
	deepSeekBalanceLowReasonPrefix      = "cn_balance_low"
)

// DeepSeekBalanceCheckService periodically probes active DeepSeek API-key
// (pay-as-you-go) accounts.  A successful probe below the configured floor
// temporarily removes the account from scheduling; a later healthy probe only
// clears a temporary block written by this service.
//
// The service is deliberately DeepSeek-only.  Coding-plan accounts, OAuth
// accounts, disabled accounts, and accounts manually marked unschedulable are
// never changed by this policy.
type DeepSeekBalanceCheckService struct {
	accountRepo    AccountRepository
	balanceService deepSeekBalanceProber
	cfg            *config.Config
	interval       time.Duration

	parentCtx    context.Context
	parentCancel context.CancelFunc
	mu           sync.Mutex
	started      bool
	stopped      bool
	wg           sync.WaitGroup
	cycleMu      sync.Mutex
	now          func() time.Time
}

// NewDeepSeekBalanceCheckService constructs the periodic checker.  The
// interval is normalized here so callers that build a Config literal (for
// example tests) retain a safe default without going through config.Load.
func NewDeepSeekBalanceCheckService(
	accountRepo AccountRepository,
	balanceService deepSeekBalanceProber,
	cfg *config.Config,
	interval time.Duration,
) *DeepSeekBalanceCheckService {
	if interval <= 0 {
		interval = deepSeekBalanceCheckDefaultInterval
	}
	if interval < deepSeekBalanceCheckMinInterval {
		interval = deepSeekBalanceCheckMinInterval
	}
	if interval > deepSeekBalanceCheckMaxInterval {
		interval = deepSeekBalanceCheckMaxInterval
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &DeepSeekBalanceCheckService{
		accountRepo:    accountRepo,
		balanceService: balanceService,
		cfg:            cfg,
		interval:       interval,
		parentCtx:      ctx,
		parentCancel:   cancel,
		now:            time.Now,
	}
}

// ProvideDeepSeekBalanceCheckService is the Wire constructor.  Lifecycle
// ownership stays with the application supervisor; constructing the service
// never starts a goroutine.
func ProvideDeepSeekBalanceCheckService(
	accountRepo AccountRepository,
	balanceService *DeepSeekBalanceService,
	cfg *config.Config,
) *DeepSeekBalanceCheckService {
	interval := deepSeekBalanceCheckDefaultInterval
	if cfg != nil && cfg.Gateway.DeepSeekBalance.IntervalMinutes > 0 {
		interval = time.Duration(cfg.Gateway.DeepSeekBalance.IntervalMinutes) * time.Minute
	}
	return NewDeepSeekBalanceCheckService(accountRepo, balanceService, cfg, interval)
}

// Start launches the runner only when explicitly enabled.  The first probe is
// delayed by one interval to avoid a startup burst; RunOnce is available for
// an explicit/manual cycle.
func (s *DeepSeekBalanceCheckService) Start() {
	if s == nil || s.accountRepo == nil || s.balanceService == nil || !s.enabled() {
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
					log.Printf("[DeepSeekBalance] cycle failed: %v", err)
				}
			case <-s.parentCtx.Done():
				return
			}
		}
	}()
}

// Stop is idempotent and waits for an in-flight cycle to finish.
func (s *DeepSeekBalanceCheckService) Stop() {
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

// RunOnce executes one bounded DeepSeek-only probe cycle.  Probe failures are
// intentionally non-fatal: transient network/auth errors must not mutate the
// account's existing scheduling state.
func (s *DeepSeekBalanceCheckService) RunOnce(ctx context.Context) error {
	if s == nil || s.accountRepo == nil || s.balanceService == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	s.cycleMu.Lock()
	defer s.cycleMu.Unlock()

	accounts, err := s.accountRepo.ListByPlatform(ctx, PlatformDeepseek)
	if err != nil {
		return fmt.Errorf("list DeepSeek accounts: %w", err)
	}

	targets := make([]Account, 0, len(accounts))
	seen := make(map[int64]struct{}, len(accounts))
	for i := range accounts {
		account := accounts[i]
		if _, exists := seen[account.ID]; exists {
			continue
		}
		seen[account.ID] = struct{}{}
		if !isDeepSeekPayGBalanceTarget(&account) {
			continue
		}
		targets = append(targets, account)
	}

	// Keep a slow/failed account from serially delaying every other account, but
	// cap outbound work so one cycle cannot overwhelm the upstream.
	sem := make(chan struct{}, deepSeekBalanceCheckConcurrency)
	var wg sync.WaitGroup
	for i := range targets {
		account := targets[i]
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()
			s.checkOne(ctx, &account, s.threshold())
		}()
	}
	wg.Wait()
	return nil
}

// isDeepSeekPayGBalanceTarget excludes all account classes for which the
// public /user/balance endpoint is not a valid source of truth.  An omitted
// account_mode is the legacy DeepSeek payg representation, so it remains
// eligible; an explicit coding mode is never probed.
func isDeepSeekPayGBalanceTarget(account *Account) bool {
	if account == nil || !account.IsActive() || !account.IsDeepseek() {
		return false
	}
	// Schedulable is the operator-controlled/manual switch.  Do not spend an
	// upstream request on an account the operator has explicitly disabled, and
	// in particular do not let a healthy probe clear its temporary metadata.
	// Temporary/rate-limit blocks remain eligible so this service can recover a
	// block that it owns once the balance is healthy.
	if !account.Schedulable {
		return false
	}
	if account.Type != AccountTypeAPIKey || account.GetAccountMode() == AccountModeCoding {
		return false
	}
	return strings.TrimSpace(account.GetCNAPIKey()) != ""
}

func (s *DeepSeekBalanceCheckService) checkOne(ctx context.Context, account *Account, threshold float64) {
	if account == nil {
		return
	}
	if ctx != nil {
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
	result, err := s.balanceService.QueryBalance(ctx, account.ID)
	if err != nil || result == nil || !result.Success {
		return
	}
	if ctx != nil {
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
	low := !result.Available || allDeepSeekBalancesBelowThreshold(result, threshold)
	if low {
		// IsSchedulable includes manual disables and other active temporary
		// blocks.  Never overwrite another subsystem's reason.
		if !account.IsSchedulable() {
			return
		}
		reason := deepSeekBalanceLowReason(result, threshold)
		until := s.currentTime().Add(s.cooldown())
		if err := s.accountRepo.SetTempUnschedulable(ctx, account.ID, until, reason); err != nil {
			log.Printf("[DeepSeekBalance] pause account %d failed: %v", account.ID, err)
			return
		}
		_ = s.accountRepo.UpdateExtra(ctx, account.ID, map[string]any{deepSeekBalanceLowKey: true})
		return
	}

	// Recovery is intentionally reason-scoped.  A healthy balance must not
	// clear an administrator or another subsystem's temporary quarantine.
	if account.TempUnschedulableUntil != nil &&
		strings.HasPrefix(strings.TrimSpace(account.TempUnschedulableReason), deepSeekBalanceLowReasonPrefix) {
		if err := s.accountRepo.ClearTempUnschedulable(ctx, account.ID); err != nil {
			log.Printf("[DeepSeekBalance] recover account %d failed: %v", account.ID, err)
			return
		}
		_ = s.accountRepo.UpdateExtra(ctx, account.ID, map[string]any{deepSeekBalanceLowKey: false})
	}
}

func deepSeekBalanceLowReason(result *DeepSeekBalanceResult, threshold float64) string {
	if result == nil {
		return deepSeekBalanceLowReasonPrefix + ": balance below threshold"
	}
	return fmt.Sprintf("%s: balance %.6g %s below threshold %.6g",
		deepSeekBalanceLowReasonPrefix, result.Balance, result.Currency, threshold)
}

func allDeepSeekBalancesBelowThreshold(result *DeepSeekBalanceResult, threshold float64) bool {
	if result == nil {
		return true
	}
	if len(result.Balances) == 0 {
		return result.Balance < threshold
	}
	for _, entry := range result.Balances {
		if entry.Balance >= threshold {
			return false
		}
	}
	return true
}

func (s *DeepSeekBalanceCheckService) enabled() bool {
	return s != nil && s.cfg != nil && s.cfg.Gateway.DeepSeekBalance.Enabled
}

func (s *DeepSeekBalanceCheckService) threshold() float64 {
	threshold := 0.5
	if s != nil && s.cfg != nil {
		threshold = s.cfg.Gateway.DeepSeekBalance.Threshold
	}
	if threshold < 0 || math.IsNaN(threshold) || math.IsInf(threshold, 0) { // config.Validate catches this on load.
		return 0.5
	}
	return threshold
}

func (s *DeepSeekBalanceCheckService) cooldown() time.Duration {
	interval := deepSeekBalanceCheckDefaultInterval
	if s != nil && s.interval > 0 {
		interval = s.interval
	}
	if interval > (24*time.Hour)/2 {
		return 24 * time.Hour
	}
	return 2 * interval
}

func (s *DeepSeekBalanceCheckService) currentTime() time.Time {
	if s != nil && s.now != nil {
		return s.now()
	}
	return time.Now()
}
