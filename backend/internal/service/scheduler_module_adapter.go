package service

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"log/slog"
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	schedulerapp "github.com/Wei-Shaw/sub2api/internal/modules/scheduler/application"
	schedulerdomain "github.com/Wei-Shaw/sub2api/internal/modules/scheduler/domain"
	schedulerports "github.com/Wei-Shaw/sub2api/internal/modules/scheduler/ports"
)

type legacySchedulerModuleAdapter struct {
	snapshot *SchedulerSnapshotService
}

var (
	_ schedulerports.CandidateSelector     = (*legacySchedulerModuleAdapter)(nil)
	_ schedulerports.AccountSnapshotReader = (*legacySchedulerModuleAdapter)(nil)
	_ schedulerports.Rebuilder             = (*legacySchedulerModuleAdapter)(nil)
	_ schedulerports.OutboxConsumer        = (*legacySchedulerModuleAdapter)(nil)
	_ schedulerports.RuntimeStatusReader   = (*legacySchedulerModuleAdapter)(nil)
)

func (a *legacySchedulerModuleAdapter) SelectCandidates(ctx context.Context, query schedulerdomain.CandidateQuery) (schedulerdomain.CandidateSet, error) {
	if a == nil || a.snapshot == nil {
		return schedulerdomain.CandidateSet{}, schedulerapp.ErrPortUnavailable
	}
	accounts, mixed, err := a.snapshot.ListSchedulableAccounts(ctx, query.GroupID, query.Platform, query.ForcePlatform)
	if err != nil {
		return schedulerdomain.CandidateSet{}, err
	}
	ids := make([]int64, 0, len(accounts))
	for i := range accounts {
		if accounts[i].ID > 0 {
			ids = append(ids, accounts[i].ID)
		}
	}
	return schedulerdomain.CandidateSet{AccountIDs: ids, MixedScheduling: mixed}, nil
}

func (a *legacySchedulerModuleAdapter) ReadAccountSnapshot(ctx context.Context, accountID int64) (schedulerdomain.AccountSnapshot, bool, error) {
	if a == nil || a.snapshot == nil {
		return schedulerdomain.AccountSnapshot{}, false, schedulerapp.ErrPortUnavailable
	}
	account, err := a.snapshot.GetAccount(ctx, accountID)
	if err != nil || account == nil {
		return schedulerdomain.AccountSnapshot{}, false, err
	}
	return schedulerdomain.AccountSnapshot{
		ID:          account.ID,
		Platform:    account.Platform,
		Status:      account.Status,
		Schedulable: account.Schedulable,
		UpdatedAt:   account.UpdatedAt,
	}, true, nil
}

func (a *legacySchedulerModuleAdapter) Rebuild(ctx context.Context, reason string) error {
	if a == nil || a.snapshot == nil {
		return schedulerapp.ErrPortUnavailable
	}
	if ctx == nil {
		ctx = context.Background()
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "module-facade"
	}
	return a.snapshot.triggerFullRebuildWithContext(ctx, reason, false)
}

func (a *legacySchedulerModuleAdapter) ConsumeOutbox(ctx context.Context, _ int) (schedulerdomain.OutboxConsumeResult, error) {
	if a == nil || a.snapshot == nil {
		return schedulerdomain.OutboxConsumeResult{}, schedulerapp.ErrPortUnavailable
	}
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return schedulerdomain.OutboxConsumeResult{}, err
		}
	}
	// Legacy polling owns its timeout and fixed batch limit. This adapter is a
	// trigger boundary until the consumer loop itself moves into the module.
	a.snapshot.pollOutbox()
	return schedulerdomain.OutboxConsumeResult{Attempted: true, Consumed: -1}, nil
}

func (a *legacySchedulerModuleAdapter) RuntimeStatus(context.Context) schedulerdomain.RuntimeStatus {
	if a == nil || a.snapshot == nil {
		return schedulerdomain.RuntimeStatus{}
	}
	initial := a.snapshot.InitialSnapshotStatus()
	status := schedulerdomain.RuntimeStatus{
		Started:             true,
		InitialSnapshotDone: initial.Done,
	}
	if initial.Err != nil {
		status.Degraded = true
		status.LastError = initial.Err.Error()
	}
	a.snapshot.lagMu.Lock()
	if a.snapshot.outboxRebuildLatched || a.snapshot.outboxRebuildRunning || a.snapshot.outboxRebuildFailures > 0 {
		status.Degraded = true
	}
	a.snapshot.lagMu.Unlock()
	a.snapshot.fullRebuildStateMu.Lock()
	if err := a.snapshot.fullRebuildLastErr; err != nil {
		status.Degraded = true
		if status.LastError == "" {
			status.LastError = err.Error()
		}
	}
	a.snapshot.fullRebuildStateMu.Unlock()
	return status
}

// schedulerCandidateRepository is the database read boundary used by the
// module-side candidate selector. It intentionally excludes scheduler cache
// and mutation capabilities so the shadow path remains an independent read.
type schedulerCandidateRepository interface {
	ListSchedulableByPlatform(ctx context.Context, platform string) ([]Account, error)
	ListSchedulableByGroupIDAndPlatform(ctx context.Context, groupID int64, platform string) ([]Account, error)
	ListSchedulableByPlatforms(ctx context.Context, platforms []string) ([]Account, error)
	ListSchedulableByGroupIDAndPlatforms(ctx context.Context, groupID int64, platforms []string) ([]Account, error)
	ListSchedulableUngroupedByPlatform(ctx context.Context, platform string) ([]Account, error)
	ListSchedulableUngroupedByPlatforms(ctx context.Context, platforms []string) ([]Account, error)
}

// schedulerRepositoryCandidateSelector reconstructs the legacy candidate-set
// semantics directly from PostgreSQL. It is only used as a shadow selector;
// its result is never returned to gateway request paths.
type schedulerRepositoryCandidateSelector struct {
	repo          schedulerCandidateRepository
	runModeSimple bool
}

var _ schedulerports.CandidateSelector = (*schedulerRepositoryCandidateSelector)(nil)

func newSchedulerRepositoryCandidateSelector(repo schedulerCandidateRepository, cfg *config.Config) *schedulerRepositoryCandidateSelector {
	return &schedulerRepositoryCandidateSelector{
		repo:          repo,
		runModeSimple: cfg != nil && cfg.RunMode == config.RunModeSimple,
	}
}

func (s *schedulerRepositoryCandidateSelector) SelectCandidates(ctx context.Context, query schedulerdomain.CandidateQuery) (schedulerdomain.CandidateSet, error) {
	if s == nil || s.repo == nil {
		return schedulerdomain.CandidateSet{}, schedulerapp.ErrPortUnavailable
	}

	useMixed := (query.Platform == PlatformAnthropic || query.Platform == PlatformGemini) && !query.ForcePlatform
	groupID := int64(0)
	if !s.runModeSimple && query.GroupID != nil && *query.GroupID > 0 {
		groupID = *query.GroupID
	}

	var (
		accounts []Account
		err      error
	)
	if useMixed {
		platforms := []string{query.Platform, PlatformAntigravity}
		switch {
		case groupID > 0:
			accounts, err = s.repo.ListSchedulableByGroupIDAndPlatforms(ctx, groupID, platforms)
		case s.runModeSimple:
			accounts, err = s.repo.ListSchedulableByPlatforms(ctx, platforms)
		default:
			accounts, err = s.repo.ListSchedulableUngroupedByPlatforms(ctx, platforms)
		}
	} else {
		switch {
		case groupID > 0:
			accounts, err = s.repo.ListSchedulableByGroupIDAndPlatform(ctx, groupID, query.Platform)
		case s.runModeSimple:
			accounts, err = s.repo.ListSchedulableByPlatform(ctx, query.Platform)
		default:
			accounts, err = s.repo.ListSchedulableUngroupedByPlatform(ctx, query.Platform)
		}
	}
	if err != nil {
		return schedulerdomain.CandidateSet{MixedScheduling: useMixed}, err
	}

	ids := make([]int64, 0, len(accounts))
	for i := range accounts {
		account := &accounts[i]
		if useMixed && account.Platform == PlatformAntigravity && !account.IsMixedSchedulingEnabled() {
			continue
		}
		if account.ID > 0 {
			ids = append(ids, account.ID)
		}
	}
	return schedulerdomain.CandidateSet{AccountIDs: ids, MixedScheduling: useMixed}, nil
}

type schedulerShadowMismatchLogger struct{}

func (schedulerShadowMismatchLogger) ObserveCandidateMismatch(_ context.Context, mismatch schedulerdomain.CandidateMismatch) {
	primarySample, primaryCount, primaryHash := schedulerShadowIDLogSummary(mismatch.PrimaryAccountIDs)
	shadowSample, shadowCount, shadowHash := schedulerShadowIDLogSummary(mismatch.ShadowAccountIDs)
	missingSample, missingCount, missingHash := schedulerShadowIDLogSummary(mismatch.MissingFromShadow)
	unexpectedSample, unexpectedCount, unexpectedHash := schedulerShadowIDLogSummary(mismatch.UnexpectedInShadow)
	attrs := []any{
		"platform", mismatch.Query.Platform,
		"primary_id_sample", primarySample,
		"primary_id_count", primaryCount,
		"primary_id_hash", primaryHash,
		"shadow_id_sample", shadowSample,
		"shadow_id_count", shadowCount,
		"shadow_id_hash", shadowHash,
		"primary_mixed", mismatch.PrimaryMixed,
		"shadow_mixed", mismatch.ShadowMixed,
		"missing_from_shadow_sample", missingSample,
		"missing_from_shadow_count", missingCount,
		"missing_from_shadow_hash", missingHash,
		"unexpected_in_shadow_sample", unexpectedSample,
		"unexpected_in_shadow_count", unexpectedCount,
		"unexpected_in_shadow_hash", unexpectedHash,
	}
	if mismatch.Query.GroupID != nil {
		attrs = append(attrs, "group_id", *mismatch.Query.GroupID)
	}
	if mismatch.ShadowError != "" {
		attrs = append(attrs, "shadow_error", mismatch.ShadowError)
	}
	slog.Warn("scheduler candidate shadow mismatch", attrs...)
}

const schedulerShadowLogIDLimit = 20

// schedulerShadowIDLogSummary keeps mismatch logs bounded even when a group
// contains thousands of accounts. The digest is calculated from the sorted,
// de-duplicated positive ID set so operators can correlate repeated mismatches
// without logging the complete candidate population.
func schedulerShadowIDLogSummary(ids []int64) (sample []int64, count int, digest string) {
	normalized := append([]int64(nil), ids...)
	sort.Slice(normalized, func(i, j int) bool { return normalized[i] < normalized[j] })
	write := 0
	for _, id := range normalized {
		if id <= 0 || (write > 0 && normalized[write-1] == id) {
			continue
		}
		normalized[write] = id
		write++
	}
	normalized = normalized[:write]
	count = len(normalized)
	limit := count
	if limit > schedulerShadowLogIDLimit {
		limit = schedulerShadowLogIDLimit
	}
	sample = append([]int64(nil), normalized[:limit]...)

	hash := sha256.New()
	var encoded [8]byte
	for _, id := range normalized {
		binary.BigEndian.PutUint64(encoded[:], uint64(id))
		_, _ = hash.Write(encoded[:])
	}
	sum := hash.Sum(nil)
	digest = hex.EncodeToString(sum[:8])
	return sample, count, digest
}

// NewSchedulerModuleFacade exposes the new application boundary while keeping
// SchedulerSnapshotService as the outer legacy adapter. Shadow comparison is
// disabled unless explicitly requested and never changes the primary result.
func NewSchedulerModuleFacade(
	snapshot *SchedulerSnapshotService,
	shadow schedulerports.CandidateSelector,
	shadowComparisonEnabled bool,
	observer schedulerports.CandidateMismatchObserver,
) *schedulerapp.Facade {
	adapter := &legacySchedulerModuleAdapter{snapshot: snapshot}
	var selector schedulerports.CandidateSelector = adapter
	if shadowComparisonEnabled && shadow != nil {
		if observer == nil {
			observer = schedulerShadowMismatchLogger{}
		}
		selector = schedulerapp.NewShadowComparingSelector(adapter, shadow, observer, true)
	}
	return schedulerapp.NewFacade(schedulerapp.Dependencies{
		Selector:       selector,
		Snapshots:      adapter,
		Rebuilder:      adapter,
		OutboxConsumer: adapter,
		Runtime:        adapter,
	})
}

// ProvideSchedulerModuleFacade wires the legacy cache selector as authoritative
// and a direct PostgreSQL read as the module shadow. Gateway integration invokes
// the comparison asynchronously, so this provider cannot alter request routing.
func ProvideSchedulerModuleFacade(
	snapshot *SchedulerSnapshotService,
	accountRepo AccountRepository,
	cfg *config.Config,
) *schedulerapp.Facade {
	shadow := newSchedulerRepositoryCandidateSelector(accountRepo, cfg)
	enabled := true
	if cfg != nil {
		enabled = cfg.Gateway.Scheduling.ShadowComparisonEnabled
	}
	return NewSchedulerModuleFacade(snapshot, shadow, enabled, nil)
}
