package service

import (
	"context"
	"errors"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type cnQuotaCheckRepoStub struct {
	AccountRepository
	accounts []Account
	err      error
}

func (r *cnQuotaCheckRepoStub) ListByPlatform(context.Context, string) ([]Account, error) {
	if r.err != nil {
		return nil, r.err
	}
	return append([]Account(nil), r.accounts...), nil
}

type cnQuotaCheckRecordingProber struct {
	mu     sync.Mutex
	probed []int64
}

func (p *cnQuotaCheckRecordingProber) QueryUsage(_ context.Context, id int64) (*CNProviderQuotaProbeResult, error) {
	p.mu.Lock()
	p.probed = append(p.probed, id)
	p.mu.Unlock()
	return &CNProviderQuotaProbeResult{Success: true}, nil
}

func (p *cnQuotaCheckRecordingProber) ids() []int64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	result := append([]int64(nil), p.probed...)
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func TestCNProviderBalanceCheckRunOnceProbesActiveZhipuCodingAccounts(t *testing.T) {
	repo := &cnQuotaCheckRepoStub{accounts: []Account{
		{ID: 1, Platform: PlatformZhipu, Type: AccountTypeAPIKey, Status: StatusActive, Credentials: map[string]any{"account_mode": AccountModeCoding}},
		// A threshold-paused coding account still needs a fresh snapshot.
		{ID: 2, Platform: PlatformZhipu, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: false, Credentials: map[string]any{"account_mode": AccountModeCoding}},
		// Duplicate rows should not trigger duplicate upstream calls.
		{ID: 2, Platform: PlatformZhipu, Type: AccountTypeAPIKey, Status: StatusActive, Credentials: map[string]any{"account_mode": AccountModeCoding}},
		{ID: 3, Platform: PlatformZhipu, Type: AccountTypeAPIKey, Status: StatusActive, Credentials: map[string]any{"account_mode": AccountModePayG}},
		{ID: 4, Platform: PlatformZhipu, Type: AccountTypeAPIKey, Status: StatusDisabled, Credentials: map[string]any{"account_mode": AccountModeCoding}},
		{ID: 5, Platform: PlatformDeepseek, Type: AccountTypeAPIKey, Status: StatusActive, Credentials: map[string]any{"account_mode": AccountModeCoding}},
	}}
	prober := &cnQuotaCheckRecordingProber{}
	svc := NewCNProviderBalanceCheckService(repo, prober, &config.Config{}, time.Minute)

	require.NoError(t, svc.RunOnce(context.Background()))
	require.Equal(t, []int64{1, 2}, prober.ids())
}

func TestCNProviderBalanceCheckRunOnceReturnsListError(t *testing.T) {
	repo := &cnQuotaCheckRepoStub{err: errors.New("database unavailable")}
	prober := &cnQuotaCheckRecordingProber{}
	svc := NewCNProviderBalanceCheckService(repo, prober, &config.Config{}, time.Minute)

	err := svc.RunOnce(context.Background())
	require.ErrorContains(t, err, "list zhipu accounts")
	require.Empty(t, prober.ids())
}

func TestCNProviderBalanceCheckStartDisabledAndStopAreSafe(t *testing.T) {
	repo := &cnQuotaCheckRepoStub{}
	prober := &cnQuotaCheckRecordingProber{}
	svc := NewCNProviderBalanceCheckService(repo, prober, &config.Config{}, time.Minute)

	require.NotPanics(t, func() {
		svc.Start()
		svc.Stop()
		svc.Stop()
	})
	require.Empty(t, prober.ids())
}
