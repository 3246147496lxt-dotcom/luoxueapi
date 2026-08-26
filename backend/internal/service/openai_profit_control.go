package service

// Profit control is a request-scoped account-admission gate. It does not
// change scheduler ordering: it only removes accounts whose upstream rate U
// exceeds the frozen downstream threshold
//
//	D(pricingAt) * (1 - minMargin - safetyBuffer).
//
// The feature is disabled by default at the group level. Configuration-load
// failures deliberately fail open (and are logged) so a cache or repository
// outage does not turn into a gateway outage. Invalid account rates fail
// closed once a gate is active.

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
)

const (
	profitControlRateEpsilon = 1e-9

	openAIProfitFilterReasonThreshold          = "profit_threshold"
	openAIProfitFilterReasonInvalidAccountRate = "profit_invalid_account_rate"

	profitControlActivityLogInterval = 5 * time.Minute
)

// ErrStickySessionNotFound abstracts a cache miss from the backing cache
// implementation. GatewayCache implementations should map their native miss
// error (for example redis.Nil) to this sentinel.
var ErrStickySessionNotFound = errors.New("sticky session not found")

type openAIProfitControlGateCtxKey struct{}
type openAIProfitControlSuppressCtxKey struct{}
type openAIPricingAtCtxKey struct{}

func clampProfitControlThreshold(threshold float64) float64 {
	if math.IsNaN(threshold) || math.IsInf(threshold, 0) || threshold < 0 {
		return 0
	}
	return threshold
}

func profitControlOverThreshold(upstream, threshold float64) bool {
	return upstream-threshold > profitControlRateEpsilon*math.Max(1, math.Abs(threshold))
}

// openAIProfitControlGate contains only immutable request-scoped values, so it
// can safely be reused across retries and failover attempts.
type openAIProfitControlGate struct {
	groupID   int64
	platform  string
	threshold float64
	pricingAt time.Time
}

// WithOpenAIRequestPricingContext freezes the pricing instant and installs an
// OpenAI/Grok profit gate when the selected group explicitly enables it.
func (s *OpenAIGatewayService) WithOpenAIRequestPricingContext(ctx context.Context, groupID *int64) (context.Context, time.Time) {
	if ctx == nil {
		ctx = context.Background()
	}
	pricingAt := timezone.Now()
	ctx = context.WithValue(ctx, openAIPricingAtCtxKey{}, pricingAt)
	return s.withOpenAIProfitControlGate(ctx, groupID), pricingAt
}

// WithOpenAIProfitControlSuppressed marks a non-token-billed request (for
// example a media or count-only endpoint) as outside the profit-control scope.
func WithOpenAIProfitControlSuppressed(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, openAIProfitControlSuppressCtxKey{}, struct{}{})
}

// WithOpenAITurnPricingContext refreshes pricing and profit configuration for
// each turn of a long-lived connection. The already-selected dispatch group is
// retained when it differs from the ingress billing group.
func (s *OpenAIGatewayService) WithOpenAITurnPricingContext(ctx context.Context, groupID *int64) (context.Context, time.Time) {
	if ctx == nil {
		ctx = context.Background()
	}
	pricingAt := timezone.Now()
	ctx = context.WithValue(ctx, openAIPricingAtCtxKey{}, pricingAt)
	if _, suppressed := ctx.Value(openAIProfitControlSuppressCtxKey{}).(struct{}); suppressed {
		return ctx, pricingAt
	}
	if existing, ok := profitControlGateFromContext(ctx); ok {
		id := existing.groupID
		groupID = &id
	}
	gate := s.resolveOpenAIProfitControlGate(ctx, groupID)
	if gate == nil {
		if existing, ok := profitControlGateFromContext(ctx); ok && existing != nil {
			ctx = context.WithValue(ctx, openAIProfitControlGateCtxKey{}, (*openAIProfitControlGate)(nil))
		}
		return ctx, pricingAt
	}
	openAIProfitControlObserverInstance.recordInstall(gate.groupID, gate.platform, gate.threshold)
	return context.WithValue(ctx, openAIProfitControlGateCtxKey{}, gate), pricingAt
}

func openAIPricingAtFromContext(ctx context.Context) (time.Time, bool) {
	if ctx == nil {
		return time.Time{}, false
	}
	pricingAt, ok := ctx.Value(openAIPricingAtCtxKey{}).(time.Time)
	return pricingAt, ok && !pricingAt.IsZero()
}

func OpenAIPricingAtFromContext(ctx context.Context) time.Time {
	pricingAt, _ := openAIPricingAtFromContext(ctx)
	return pricingAt
}

func profitControlGateFromContext(ctx context.Context) (*openAIProfitControlGate, bool) {
	if ctx == nil {
		return nil, false
	}
	gate, ok := ctx.Value(openAIProfitControlGateCtxKey{}).(*openAIProfitControlGate)
	return gate, ok && gate != nil
}

func (s *OpenAIGatewayService) withOpenAIProfitControlGate(ctx context.Context, groupID *int64) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, suppressed := ctx.Value(openAIProfitControlSuppressCtxKey{}).(struct{}); suppressed {
		return ctx
	}
	if groupID != nil {
		if existing, ok := profitControlGateFromContext(ctx); ok && existing.groupID == *groupID {
			return ctx
		}
	}

	gate := s.resolveOpenAIProfitControlGate(ctx, groupID)
	if gate == nil {
		// Do not leak a parent/member gate into a different dispatch group.
		if existing, ok := profitControlGateFromContext(ctx); ok && groupID != nil && existing.groupID != *groupID {
			return context.WithValue(ctx, openAIProfitControlGateCtxKey{}, (*openAIProfitControlGate)(nil))
		}
		return ctx
	}
	openAIProfitControlObserverInstance.recordInstall(gate.groupID, gate.platform, gate.threshold)
	return context.WithValue(ctx, openAIProfitControlGateCtxKey{}, gate)
}

func (s *OpenAIGatewayService) resolveOpenAIProfitControlGate(ctx context.Context, groupID *int64) *openAIProfitControlGate {
	if s == nil || groupID == nil || *groupID <= 0 {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	var group *Group
	if ctxGroup, ok := ctx.Value(ctxkey.Group).(*Group); ok && IsGroupContextValid(ctxGroup) && ctxGroup.ID == *groupID {
		group = ctxGroup
	} else if s.schedulerSnapshot != nil {
		loaded, err := s.schedulerSnapshot.GetGroupByID(ctx, *groupID)
		if err != nil {
			slog.Warn("profit_control_group_load_failed", "group_id", *groupID, "error", err)
			return nil
		}
		group = loaded
	}
	if group == nil || !group.ProfitControlEnabled ||
		(group.Platform != PlatformOpenAI && group.Platform != PlatformGrok) {
		return nil
	}

	pricingAt, ok := openAIPricingAtFromContext(ctx)
	if !ok {
		pricingAt = timezone.Now()
	}

	// The ingress group is the billing owner even if dispatch later routes to a
	// different member/fallback group.
	billingGroup := group
	if ctxGroup, ok := ctx.Value(ctxkey.Group).(*Group); ok && IsGroupContextValid(ctxGroup) {
		billingGroup = ctxGroup
	}
	downstream := billingGroup.RateMultiplier
	if userID, _ := ctx.Value(ctxkey.UserID).(int64); userID > 0 {
		downstream = s.ResolveUserGroupRateMultiplier(ctx, userID, billingGroup.ID, billingGroup.RateMultiplier)
	}
	downstream *= billingGroup.PeakMultiplierAt(pricingAt)

	threshold := clampProfitControlThreshold(downstream * (1 - group.ProfitMinMargin - group.ProfitSafetyBuffer))
	return &openAIProfitControlGate{
		groupID:   group.ID,
		platform:  group.Platform,
		threshold: threshold,
		pricingAt: pricingAt,
	}
}

// attachSelectionProfitGate carries the gate out of a scheduler-local context.
func attachSelectionProfitGate(ctx context.Context, selection *AccountSelectionResult) *AccountSelectionResult {
	if selection == nil {
		return nil
	}
	if gate, ok := profitControlGateFromContext(ctx); ok {
		selection.profitGate = gate
	}
	return selection
}

// ProfitGateActive reports whether selection was produced under a profit gate.
// Handlers use it to decide whether terminal admission and deferred sticky
// binding are required.
func (s *AccountSelectionResult) ProfitGateActive() bool {
	return s != nil && s.profitGate != nil
}

// ContextWithSelectionProfitGate restores the exact gate used during account
// selection before the handler performs the terminal post-slot check.
func ContextWithSelectionProfitGate(ctx context.Context, selection *AccountSelectionResult) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if selection == nil || selection.profitGate == nil {
		return ctx
	}
	if existing, ok := profitControlGateFromContext(ctx); ok && existing == selection.profitGate {
		return ctx
	}
	return context.WithValue(ctx, openAIProfitControlGateCtxKey{}, selection.profitGate)
}

func openAIProfitControlVetoReason(ctx context.Context, account *Account) (bool, string) {
	gate, ok := profitControlGateFromContext(ctx)
	if !ok || account == nil {
		return false, ""
	}
	if account.RateMultiplier == nil || math.IsNaN(*account.RateMultiplier) ||
		math.IsInf(*account.RateMultiplier, 0) || *account.RateMultiplier < 0 {
		openAIProfitControlObserverInstance.recordVeto(gate.groupID, gate.platform, gate.threshold, openAIProfitFilterReasonInvalidAccountRate)
		return true, openAIProfitFilterReasonInvalidAccountRate
	}
	if profitControlOverThreshold(*account.RateMultiplier, gate.threshold) {
		openAIProfitControlObserverInstance.recordVeto(gate.groupID, gate.platform, gate.threshold, openAIProfitFilterReasonThreshold)
		return true, openAIProfitFilterReasonThreshold
	}
	return false, ""
}

func OpenAIProfitControlVeto(ctx context.Context, account *Account) (bool, string) {
	return openAIProfitControlVetoReason(ctx, account)
}

func (s *OpenAIGatewayService) ProfitControlVetoLatest(ctx context.Context, selected *Account) (*Account, bool, string) {
	if s == nil {
		return selected, false, ""
	}
	return profitControlVetoLatest(ctx, selected, s.schedulerSnapshot)
}

// bindOpenAIStickySessionDuringSelection retains eager binding for requests
// without a gate. Gated requests bind only after terminal admission.
func (s *OpenAIGatewayService) bindOpenAIStickySessionDuringSelection(ctx context.Context, groupID *int64, sessionHash string, accountID int64) error {
	if gatewayProfitControlGateActive(ctx) {
		return nil
	}
	return s.BindStickySession(ctx, groupID, sessionHash, accountID)
}

// BindStickySessionAfterProfitAdmission avoids replacing a different existing
// binding when a gated request had to spill over to another account.
func (s *OpenAIGatewayService) BindStickySessionAfterProfitAdmission(ctx context.Context, groupID *int64, sessionHash string, accountID int64) error {
	if s == nil || sessionHash == "" || accountID <= 0 {
		return nil
	}
	if !gatewayProfitControlGateActive(ctx) {
		return s.BindStickySession(ctx, groupID, sessionHash, accountID)
	}
	existingAccountID, err := s.getStickySessionAccountID(ctx, groupID, sessionHash)
	if err != nil && !errors.Is(err, ErrStickySessionNotFound) {
		slog.Warn("profit_control_sticky_binding_read_failed", "group_id", derefGroupID(groupID), "account_id", accountID, "error", err)
		return nil
	}
	if existingAccountID > 0 && existingAccountID != accountID {
		return nil
	}
	return s.BindStickySession(ctx, groupID, sessionHash, accountID)
}

type openAIProfitControlGroupStats struct {
	installs         atomic.Int64
	vetoThreshold    atomic.Int64
	vetoInvalidRate  atomic.Int64
	refreshFailures  atomic.Int64
	lastLogUnixMilli atomic.Int64
}

type openAIProfitControlObserver struct {
	groups sync.Map
}

var openAIProfitControlObserverInstance = &openAIProfitControlObserver{}

func profitControlObserverKey(groupID int64, platform string) string {
	return fmt.Sprintf("%s:%d", platform, groupID)
}

func (o *openAIProfitControlObserver) stats(groupID int64, platform string) *openAIProfitControlGroupStats {
	key := profitControlObserverKey(groupID, platform)
	if value, ok := o.groups.Load(key); ok {
		if stats, ok := value.(*openAIProfitControlGroupStats); ok {
			return stats
		}
	}
	value, _ := o.groups.LoadOrStore(key, &openAIProfitControlGroupStats{})
	if stats, ok := value.(*openAIProfitControlGroupStats); ok {
		return stats
	}
	return &openAIProfitControlGroupStats{}
}

func (o *openAIProfitControlObserver) recordInstall(groupID int64, platform string, threshold float64) {
	stats := o.stats(groupID, platform)
	stats.installs.Add(1)
	o.maybeLog(groupID, platform, threshold, stats)
}

func (o *openAIProfitControlObserver) recordVeto(groupID int64, platform string, threshold float64, reason string) {
	stats := o.stats(groupID, platform)
	switch reason {
	case openAIProfitFilterReasonThreshold:
		stats.vetoThreshold.Add(1)
	case openAIProfitFilterReasonInvalidAccountRate:
		stats.vetoInvalidRate.Add(1)
	}
	o.maybeLog(groupID, platform, threshold, stats)
}

func (o *openAIProfitControlObserver) recordRefreshFailure(groupID int64, platform string, threshold float64) {
	stats := o.stats(groupID, platform)
	stats.refreshFailures.Add(1)
	o.maybeLog(groupID, platform, threshold, stats)
}

func (o *openAIProfitControlObserver) maybeLog(groupID int64, platform string, threshold float64, stats *openAIProfitControlGroupStats) {
	now := time.Now().UnixMilli()
	last := stats.lastLogUnixMilli.Load()
	if last != 0 && now-last < profitControlActivityLogInterval.Milliseconds() {
		return
	}
	if !stats.lastLogUnixMilli.CompareAndSwap(last, now) {
		return
	}
	slog.Info("profit_control_activity",
		"group_id", groupID,
		"platform", platform,
		"threshold", threshold,
		"installs_total", stats.installs.Load(),
		"veto_threshold_total", stats.vetoThreshold.Load(),
		"veto_invalid_account_rate_total", stats.vetoInvalidRate.Load(),
		"refresh_failure_total", stats.refreshFailures.Load(),
	)
}
