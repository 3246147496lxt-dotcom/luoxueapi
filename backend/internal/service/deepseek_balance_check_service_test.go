package service

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type deepSeekBalanceCheckRepo struct {
	AccountRepository
	mu       sync.Mutex
	accounts []Account
	pauses   []deepSeekBalancePause
	clears   []int64
	updates  []map[string]any
	listErr  error
}

type deepSeekBalancePause struct {
	id     int64
	until  time.Time
	reason string
}

func (r *deepSeekBalanceCheckRepo) ListByPlatform(context.Context, string) ([]Account, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.listErr != nil {
		return nil, r.listErr
	}
	accounts := make([]Account, len(r.accounts))
	copy(accounts, r.accounts)
	return accounts, nil
}

func (r *deepSeekBalanceCheckRepo) SetTempUnschedulable(_ context.Context, id int64, until time.Time, reason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pauses = append(r.pauses, deepSeekBalancePause{id: id, until: until, reason: reason})
	return nil
}

func (r *deepSeekBalanceCheckRepo) ClearTempUnschedulable(_ context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clears = append(r.clears, id)
	return nil
}

func (r *deepSeekBalanceCheckRepo) UpdateExtra(_ context.Context, _ int64, updates map[string]any) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.updates = append(r.updates, updates)
	return nil
}

type deepSeekBalanceCheckProber struct {
	mu      sync.Mutex
	results map[int64]*DeepSeekBalanceResult
	errors  map[int64]error
	probed  []int64
}

func (p *deepSeekBalanceCheckProber) QueryBalance(_ context.Context, id int64) (*DeepSeekBalanceResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.probed = append(p.probed, id)
	if err := p.errors[id]; err != nil {
		return nil, err
	}
	return p.results[id], nil
}

func deepSeekBalanceCheckAccount(id int64, mode string) Account {
	return Account{
		ID:          id,
		Platform:    PlatformDeepseek,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Credentials: map[string]any{"api_key": "sk-test", "account_mode": mode},
	}
}

func deepSeekBalanceCheckConfig(threshold float64) *config.Config {
	return &config.Config{Gateway: config.GatewayConfig{DeepSeekBalance: config.GatewayDeepSeekBalanceConfig{
		Enabled:   true,
		Threshold: threshold,
	}}}
}

func TestDeepSeekBalanceCheckPausesLowBalanceAndPersistsMarker(t *testing.T) {
	account := deepSeekBalanceCheckAccount(11, AccountModePayG)
	repo := &deepSeekBalanceCheckRepo{accounts: []Account{account}}
	prober := &deepSeekBalanceCheckProber{results: map[int64]*DeepSeekBalanceResult{
		account.ID: {
			Provider:  PlatformDeepseek,
			Success:   true,
			Available: true,
			Balance:   0.25,
			Currency:  "USD",
			Balances:  []DeepSeekBalanceEntry{{Currency: "USD", Balance: 0.25}, {Currency: "CNY", Balance: 0.4}},
		},
	}}
	svc := NewDeepSeekBalanceCheckService(repo, prober, deepSeekBalanceCheckConfig(1), time.Minute)
	fixedNow := time.Date(2026, 9, 4, 1, 2, 3, 0, time.UTC)
	svc.now = func() time.Time { return fixedNow }

	require.NoError(t, svc.RunOnce(context.Background()))
	require.Len(t, repo.pauses, 1)
	require.Equal(t, account.ID, repo.pauses[0].id)
	require.Equal(t, fixedNow.Add(2*time.Minute), repo.pauses[0].until)
	require.True(t, strings.HasPrefix(repo.pauses[0].reason, deepSeekBalanceLowReasonPrefix))
	require.Len(t, repo.updates, 1)
	require.Equal(t, true, repo.updates[0][deepSeekBalanceLowKey])
}

func TestDeepSeekBalanceCheckMultiCurrencyAnyHealthyBalanceAvoidsPause(t *testing.T) {
	account := deepSeekBalanceCheckAccount(12, "") // legacy payg representation
	repo := &deepSeekBalanceCheckRepo{accounts: []Account{account}}
	prober := &deepSeekBalanceCheckProber{results: map[int64]*DeepSeekBalanceResult{
		account.ID: {
			Provider:  PlatformDeepseek,
			Success:   true,
			Available: true,
			Balance:   0.2,
			Currency:  "CNY",
			Balances:  []DeepSeekBalanceEntry{{Currency: "CNY", Balance: 0.2}, {Currency: "USD", Balance: 4}},
		},
	}}
	svc := NewDeepSeekBalanceCheckService(repo, prober, deepSeekBalanceCheckConfig(1), time.Minute)
	require.NoError(t, svc.RunOnce(context.Background()))
	require.Empty(t, repo.pauses)
	require.Empty(t, repo.clears)
}

func TestDeepSeekBalanceCheckRecoversOnlyOwnTemporaryPause(t *testing.T) {
	until := time.Now().Add(time.Hour)
	owned := deepSeekBalanceCheckAccount(13, AccountModePayG)
	owned.TempUnschedulableUntil = &until
	owned.TempUnschedulableReason = deepSeekBalanceLowReasonPrefix + ": old low balance"
	foreign := deepSeekBalanceCheckAccount(14, AccountModePayG)
	foreign.TempUnschedulableUntil = &until
	foreign.TempUnschedulableReason = "admin: maintenance"
	repo := &deepSeekBalanceCheckRepo{accounts: []Account{owned, foreign}}
	prober := &deepSeekBalanceCheckProber{results: map[int64]*DeepSeekBalanceResult{
		owned.ID:   {Provider: PlatformDeepseek, Success: true, Available: true, Balance: 10, Currency: "USD"},
		foreign.ID: {Provider: PlatformDeepseek, Success: true, Available: true, Balance: 10, Currency: "USD"},
	}}
	svc := NewDeepSeekBalanceCheckService(repo, prober, deepSeekBalanceCheckConfig(1), time.Minute)

	require.NoError(t, svc.RunOnce(context.Background()))
	require.Equal(t, []int64{owned.ID}, repo.clears)
	require.Len(t, repo.updates, 1)
	require.Equal(t, false, repo.updates[0][deepSeekBalanceLowKey])
}

func TestDeepSeekBalanceCheckSkipsNonPayGAccountsAndProbeFailures(t *testing.T) {
	valid := deepSeekBalanceCheckAccount(15, AccountModePayG)
	coding := deepSeekBalanceCheckAccount(16, AccountModeCoding)
	coding.Schedulable = true
	oauth := deepSeekBalanceCheckAccount(17, AccountModePayG)
	oauth.Type = AccountTypeOAuth
	manualDisabled := deepSeekBalanceCheckAccount(18, AccountModePayG)
	manualDisabled.Schedulable = false
	other := valid
	other.ID = 19
	other.Platform = PlatformOpenAI
	repo := &deepSeekBalanceCheckRepo{accounts: []Account{valid, coding, oauth, manualDisabled, other}}
	prober := &deepSeekBalanceCheckProber{
		results: map[int64]*DeepSeekBalanceResult{valid.ID: nil},
		errors:  map[int64]error{valid.ID: errors.New("temporary timeout")},
	}
	svc := NewDeepSeekBalanceCheckService(repo, prober, deepSeekBalanceCheckConfig(1), time.Minute)

	require.NoError(t, svc.RunOnce(context.Background()))
	require.Equal(t, []int64{valid.ID}, prober.probed)
	require.Empty(t, repo.pauses)
	require.Empty(t, repo.clears)
}

func TestDeepSeekBalanceCheckStartDisabledAndStopAreSafe(t *testing.T) {
	account := deepSeekBalanceCheckAccount(19, AccountModePayG)
	repo := &deepSeekBalanceCheckRepo{accounts: []Account{account}}
	prober := &deepSeekBalanceCheckProber{results: map[int64]*DeepSeekBalanceResult{}}
	cfg := deepSeekBalanceCheckConfig(1)
	cfg.Gateway.DeepSeekBalance.Enabled = false
	svc := NewDeepSeekBalanceCheckService(repo, prober, cfg, time.Minute)
	svc.Start()
	svc.Stop()
	svc.Stop()
	require.Empty(t, prober.probed)
}

func TestDeepSeekBalanceCheckRunOnceReturnsListError(t *testing.T) {
	repo := &deepSeekBalanceCheckRepo{listErr: errors.New("db unavailable")}
	prober := &deepSeekBalanceCheckProber{}
	svc := NewDeepSeekBalanceCheckService(repo, prober, deepSeekBalanceCheckConfig(1), time.Minute)
	require.ErrorContains(t, svc.RunOnce(context.Background()), "list DeepSeek accounts")
}
