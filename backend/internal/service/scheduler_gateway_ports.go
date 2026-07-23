package service

import (
	"context"
	"sync"
	"time"

	schedulerapp "github.com/Wei-Shaw/sub2api/internal/modules/scheduler/application"
	schedulerdomain "github.com/Wei-Shaw/sub2api/internal/modules/scheduler/domain"
)

// CandidateSelector is the only candidate-list capability required by gateway
// request paths. Scheduler runtime, rebuild and outbox lifecycle are excluded.
type CandidateSelector interface {
	ListSchedulableAccounts(ctx context.Context, groupID *int64, platform string, hasForcePlatform bool) ([]Account, bool, error)
}

// AccountSnapshotReader is the gateway-facing snapshot boundary. The cache
// update hook remains here temporarily because Antigravity applies an immediate
// model-limit patch after persistence; it does not expose scheduler lifecycle.
type AccountSnapshotReader interface {
	GetAccount(ctx context.Context, accountID int64) (*Account, error)
	GetGroupByID(ctx context.Context, groupID int64) (*Group, error)
	UpdateAccountInCache(ctx context.Context, account *Account) error
}

type GatewayScheduler interface {
	CandidateSelector
	AccountSnapshotReader
}

const (
	// The current shadow selector performs full repository hydration. Sample at
	// a low rate until it is replaced with an ID-only query so comparison cannot
	// contend materially with request-path database work.
	schedulerShadowComparisonInterval = 2 * time.Minute
	schedulerShadowGlobalInterval     = 2 * time.Second
	schedulerShadowQueryLimit         = 1024
)

type schedulerShadowQueryKey struct {
	groupID       int64
	platform      string
	forcePlatform bool
}

// schedulerGatewayBridge keeps SchedulerSnapshotService authoritative while
// sending a throttled copy of successful selections to the module facade. The
// facade owns the bounded asynchronous comparison; no module result can flow
// back into the gateway response.
type schedulerGatewayBridge struct {
	legacy *SchedulerSnapshotService
	module *schedulerapp.Facade

	shadowMu            sync.Mutex
	shadowLast          map[schedulerShadowQueryKey]time.Time
	shadowNextAllowedAt time.Time
}

var _ GatewayScheduler = (*schedulerGatewayBridge)(nil)

func newSchedulerGatewayBridge(legacy *SchedulerSnapshotService, module *schedulerapp.Facade) *schedulerGatewayBridge {
	return &schedulerGatewayBridge{
		legacy:     legacy,
		module:     module,
		shadowLast: make(map[schedulerShadowQueryKey]time.Time),
	}
}

func (b *schedulerGatewayBridge) ListSchedulableAccounts(
	ctx context.Context,
	groupID *int64,
	platform string,
	hasForcePlatform bool,
) ([]Account, bool, error) {
	if b == nil || b.legacy == nil {
		return nil, false, ErrSchedulerCacheNotReady
	}
	accounts, mixed, err := b.legacy.ListSchedulableAccounts(ctx, groupID, platform, hasForcePlatform)
	if err != nil || b.module == nil {
		return accounts, mixed, err
	}

	key, reservedAt, ok := b.reserveShadowComparison(groupID, platform, hasForcePlatform)
	if !ok {
		return accounts, mixed, nil
	}
	query := schedulerdomain.CandidateQuery{
		Platform:      platform,
		ForcePlatform: hasForcePlatform,
	}
	if groupID != nil {
		value := *groupID
		query.GroupID = &value
	}
	ids := make([]int64, 0, len(accounts))
	for i := range accounts {
		if accounts[i].ID > 0 {
			ids = append(ids, accounts[i].ID)
		}
	}
	scheduled := b.module.ObservePrimaryCandidates(ctx, query, schedulerdomain.CandidateSet{
		AccountIDs:      ids,
		MixedScheduling: mixed,
	})
	if !scheduled {
		b.releaseShadowReservation(key, reservedAt)
	}
	return accounts, mixed, nil
}

func (b *schedulerGatewayBridge) GetAccount(ctx context.Context, accountID int64) (*Account, error) {
	if b == nil || b.legacy == nil {
		return nil, ErrSchedulerCacheNotReady
	}
	return b.legacy.GetAccount(ctx, accountID)
}

func (b *schedulerGatewayBridge) GetGroupByID(ctx context.Context, groupID int64) (*Group, error) {
	if b == nil || b.legacy == nil {
		return nil, ErrSchedulerCacheNotReady
	}
	return b.legacy.GetGroupByID(ctx, groupID)
}

func (b *schedulerGatewayBridge) UpdateAccountInCache(ctx context.Context, account *Account) error {
	if b == nil || b.legacy == nil {
		return ErrSchedulerCacheNotReady
	}
	return b.legacy.UpdateAccountInCache(ctx, account)
}

func (b *schedulerGatewayBridge) reserveShadowComparison(groupID *int64, platform string, forcePlatform bool) (schedulerShadowQueryKey, time.Time, bool) {
	key := schedulerShadowQueryKey{platform: platform, forcePlatform: forcePlatform}
	if groupID != nil && *groupID > 0 && (b.legacy == nil || !b.legacy.isRunModeSimple()) {
		key.groupID = *groupID
	}
	now := time.Now()
	b.shadowMu.Lock()
	defer b.shadowMu.Unlock()
	if b.shadowLast == nil {
		b.shadowLast = make(map[schedulerShadowQueryKey]time.Time)
	}
	if now.Before(b.shadowNextAllowedAt) {
		return key, time.Time{}, false
	}
	if last, exists := b.shadowLast[key]; exists && now.Sub(last) < schedulerShadowComparisonInterval {
		return key, time.Time{}, false
	}
	if len(b.shadowLast) >= schedulerShadowQueryLimit {
		cutoff := now.Add(-2 * schedulerShadowComparisonInterval)
		for existingKey, last := range b.shadowLast {
			if last.Before(cutoff) {
				delete(b.shadowLast, existingKey)
			}
		}
	}
	if len(b.shadowLast) >= schedulerShadowQueryLimit {
		var (
			oldestKey schedulerShadowQueryKey
			oldest    time.Time
		)
		for existingKey, last := range b.shadowLast {
			if oldest.IsZero() || last.Before(oldest) {
				oldestKey = existingKey
				oldest = last
			}
		}
		delete(b.shadowLast, oldestKey)
	}
	b.shadowLast[key] = now
	b.shadowNextAllowedAt = now.Add(schedulerShadowGlobalInterval)
	return key, now, true
}

func (b *schedulerGatewayBridge) releaseShadowReservation(key schedulerShadowQueryKey, reservedAt time.Time) {
	b.shadowMu.Lock()
	defer b.shadowMu.Unlock()
	if current, ok := b.shadowLast[key]; ok && current.Equal(reservedAt) {
		delete(b.shadowLast, key)
	}
}

// ProvideGatewayScheduler is the production gateway boundary. Legacy remains
// the only synchronous selector and snapshot source.
func ProvideGatewayScheduler(snapshot *SchedulerSnapshotService, module *schedulerapp.Facade) GatewayScheduler {
	if snapshot == nil {
		return nil
	}
	return newSchedulerGatewayBridge(snapshot, module)
}

func ProvideAccountSnapshotReader(scheduler GatewayScheduler) AccountSnapshotReader {
	if scheduler == nil {
		return nil
	}
	return scheduler
}

var _ GatewayScheduler = (*SchedulerSnapshotService)(nil)
