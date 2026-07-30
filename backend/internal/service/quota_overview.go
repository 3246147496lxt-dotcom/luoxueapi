package service

import (
	"context"
	"errors"
	"math"
	"sort"
	"strconv"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

const (
	QuotaOverviewSchemaVersion = 1

	QuotaFreshnessFresh   = "fresh"
	QuotaFreshnessStale   = "stale"
	QuotaFreshnessUnknown = "unknown"

	QuotaStateAllAvailable = "all_resources_available"
	QuotaStatePartial      = "partially_restricted"
	QuotaStateAllBlocked   = "all_resources_restricted"
	QuotaStateNoKeys       = "no_enabled_keys"
	QuotaStateUnknown      = "unknown"

	QuotaGroupUsable          = "usable"
	QuotaGroupPartiallyUsable = "partially_usable"
	QuotaGroupBlocked         = "blocked"
	QuotaGroupNoKeys          = "no_enabled_keys"
	QuotaGroupUnknown         = "unknown"

	QuotaWindowActive    = "active"
	QuotaWindowExhausted = "exhausted"
	QuotaWindowUnknown   = "unknown"

	QuotaPeriodUsageAvailable = "available"
	QuotaPeriodUsageUnknown   = "unknown"

	QuotaPeriodUsageBucketAnchored24h = "anchored_24h"

	QuotaPeriodUsagePointComplete = "complete"
	QuotaPeriodUsagePointPartial  = "partial"
	QuotaPeriodUsagePointFuture   = "future"

	QuotaWalletAvailable = "available"
	QuotaWalletExhausted = "exhausted"
	QuotaWalletUnknown   = "unknown"

	QuotaBillingBalance      = "balance"
	QuotaBillingSubscription = "subscription"
)

var ErrQuotaOverviewUnavailable = infraerrors.ServiceUnavailable(
	"QUOTA_OVERVIEW_UNAVAILABLE",
	"Unable to generate a safe quota snapshot. Please retry later.",
)

type QuotaOverviewRepository interface {
	LoadQuotaOverviewSnapshot(ctx context.Context, userID int64, displayTimezone string) (*QuotaOverviewSnapshot, error)
}

// QuotaOverviewConsistencyChecker is the adapter point between the read model
// and the billing/cache implementation. A checker must compare the snapshot's
// subscription identity, anchored period and version with the state used by
// request admission. Returning Consistent=false makes every availability state
// unknown while retaining the DB amounts for diagnosis.
type QuotaOverviewConsistencyChecker interface {
	CheckQuotaOverviewConsistency(ctx context.Context, snapshot *QuotaOverviewSnapshot) (QuotaOverviewConsistency, error)
}

type QuotaOverviewConsistency struct {
	Consistent   bool
	WarningCodes []string
}

type QuotaOverviewSnapshot struct {
	AsOf                        time.Time
	Account                     QuotaOverviewAccountSnapshot
	TodaySpend                  decimal.Decimal
	MonthSpend                  decimal.Decimal
	Keys                        []QuotaOverviewKeySnapshot
	Subscriptions               []QuotaOverviewSubscriptionSnapshot
	PeriodUsageAggregationError bool
}

type QuotaOverviewAccountSnapshot struct {
	ID            int64
	Email         string
	Status        string
	Balance       decimal.Decimal
	FrozenBalance decimal.Decimal
}

type QuotaOverviewGroupSnapshot struct {
	ID               int64
	Name             string
	Status           string
	SubscriptionType string
	Deleted          bool
}

// QuotaOverviewKeySnapshot is deliberately a minimal, already-redacted
// projection. It must never grow a RawKey/Key field.
type QuotaOverviewKeySnapshot struct {
	ID        int64
	Name      string
	MaskedKey string
	Status    string
	GroupID   *int64
	Group     *QuotaOverviewGroupSnapshot
	Quota     decimal.Decimal
	QuotaUsed decimal.Decimal
	ExpiresAt *time.Time
}

type QuotaOverviewSubscriptionSnapshot struct {
	ID                        int64
	GroupID                   int64
	GroupName                 string
	GroupStatus               string
	GroupDeleted              bool
	Status                    string
	Revoked                   bool
	StartsAt                  time.Time
	ExpiresAt                 time.Time
	WeeklyWindowStart         *time.Time
	WeeklyWindowProjectedFrom *time.Time
	WeeklyLimit               *decimal.Decimal
	WeeklyUsed                decimal.Decimal
	UpdatedAt                 time.Time
	PeriodUsage               *QuotaOverviewPeriodUsageSnapshot
}

type QuotaOverviewPeriodUsageSnapshot struct {
	PeriodStart   time.Time
	PeriodEnd     time.Time
	ObservedUntil time.Time
	Buckets       [7]QuotaOverviewPeriodUsageBucketSnapshot
}

type QuotaOverviewPeriodUsageBucketSnapshot struct {
	Requests        int64
	CacheHitTokens  int64
	CacheMissTokens int64
	OutputTokens    int64
}

type QuotaOverview struct {
	SchemaVersion   int
	RequestID       string
	GeneratedAt     time.Time
	AsOf            time.Time
	FreshUntil      time.Time
	DisplayTimezone string
	Freshness       string
	Coverage        QuotaOverviewCoverage
	Account         QuotaOverviewAccount
	Wallet          QuotaOverviewWallet
	BillingGroups   []QuotaOverviewBillingGroup
	Subscriptions   []QuotaOverviewSubscription
	Actions         QuotaOverviewActions
	Warnings        []string
}

type QuotaOverviewCoverage struct {
	Included []string
	Excluded []string
}

type QuotaOverviewAccount struct {
	DisplayLabel      string
	DataScope         string
	QuotaState        string
	CanMakeRequest    *bool
	UsableGroupCount  int
	BlockedGroupCount int
	UnknownGroupCount int
	PrimaryIssue      *QuotaOverviewIssue
}

type QuotaOverviewIssue struct {
	ScopeType         string
	ScopeID           string
	ReasonCode        string
	RecommendedAction string
	RecoversAt        *time.Time
}

type QuotaOverviewWallet struct {
	Unit                  string
	State                 string
	Available             string
	Reserved              string
	TodaySpend            string
	MonthSpend            string
	BalanceBilledKeyCount int
}

type QuotaOverviewBillingGroup struct {
	ID                string
	DisplayName       string
	BillingMode       string
	State             string
	ReasonCode        *string
	RecommendedAction string
	FallbackPolicy    string
	ResourceRef       QuotaOverviewResourceRef
	Keys              []QuotaOverviewKey
}

type QuotaOverviewResourceRef struct {
	Kind string
	ID   *string
}

type QuotaOverviewKey struct {
	ID        string
	Name      string
	MaskedKey string
	State     string
}

type QuotaOverviewSubscription struct {
	ID           string
	GroupID      string
	Name         string
	Status       string
	StartsAt     time.Time
	ExpiresAt    time.Time
	WeeklyWindow QuotaOverviewWeeklyWindow
	PeriodUsage  QuotaOverviewPeriodUsage
	NextEvent    *QuotaOverviewNextEvent
}

type QuotaOverviewPeriodUsage struct {
	State         string
	ObservedUntil *time.Time
	BucketKind    string
	TotalRequests *int64
	TotalTokens   *int64
	Points        []QuotaOverviewPeriodUsagePoint
}

type QuotaOverviewPeriodUsagePoint struct {
	Index           int
	StartAt         time.Time
	EndAt           time.Time
	State           string
	Requests        *int64
	CacheHitTokens  *int64
	CacheMissTokens *int64
	OutputTokens    *int64
	TotalTokens     *int64
}

type QuotaOverviewWeeklyWindow struct {
	Kind        string
	State       string
	AnchorAt    time.Time
	PeriodStart *time.Time
	PeriodEnd   *time.Time
	ResetsAt    *time.Time
	Limit       *string
	Used        *string
	Remaining   *string
	UsedPercent *float64
}

type QuotaOverviewNextEvent struct {
	Kind string
	At   time.Time
}

type QuotaOverviewActions struct {
	RechargeURL            string
	ManageKeysURL          string
	ManageSubscriptionsURL string
}

type cachedQuotaOverview struct {
	overview QuotaOverview
	storedAt time.Time
}

type quotaOverviewCacheKey struct {
	userID          int64
	displayTimezone string
}

type QuotaOverviewService struct {
	repo               QuotaOverviewRepository
	consistencyChecker QuotaOverviewConsistencyChecker
	now                func() time.Time
	freshFor           time.Duration
	staleRetention     time.Duration
	cacheMu            sync.Mutex
	lastGood           map[quotaOverviewCacheKey]cachedQuotaOverview
	actions            QuotaOverviewActions
}

func NewQuotaOverviewService(repo QuotaOverviewRepository, checker QuotaOverviewConsistencyChecker) *QuotaOverviewService {
	return &QuotaOverviewService{
		repo:               repo,
		consistencyChecker: checker,
		now:                time.Now,
		freshFor:           5 * time.Minute,
		staleRetention:     24 * time.Hour,
		lastGood:           make(map[quotaOverviewCacheKey]cachedQuotaOverview),
	}
}

func (s *QuotaOverviewService) SetActions(actions QuotaOverviewActions) {
	if s != nil {
		s.actions = actions
	}
}

func (s *QuotaOverviewService) GetOverview(ctx context.Context, userID int64, displayTimezone string) (*QuotaOverview, error) {
	if s == nil || s.repo == nil {
		return nil, ErrQuotaOverviewUnavailable
	}
	if displayTimezone == "" {
		displayTimezone = "UTC"
	}
	snapshot, err := s.repo.LoadQuotaOverviewSnapshot(ctx, userID, displayTimezone)
	if err != nil {
		if stale := s.loadStale(userID, displayTimezone); stale != nil {
			stale.GeneratedAt = s.now().UTC()
			degradeQuotaOverview(stale, QuotaFreshnessStale, "data_stale")
			return stale, nil
		}
		return nil, ErrQuotaOverviewUnavailable.WithCause(err)
	}
	if snapshot == nil || snapshot.AsOf.IsZero() {
		return nil, ErrQuotaOverviewUnavailable
	}

	normalizeQuotaOverviewElapsedEmptyWindows(snapshot)
	overview := buildQuotaOverview(snapshot, s.now().UTC(), displayTimezone, s.freshFor, s.actions)
	consistency := QuotaOverviewConsistency{Consistent: false, WarningCodes: []string{"quota_consistency_unverified"}}
	if !validQuotaOverviewSnapshot(snapshot, userID) {
		consistency = QuotaOverviewConsistency{Consistent: false, WarningCodes: []string{"invalid_quota_snapshot"}}
	} else if s.consistencyChecker != nil {
		checked, checkErr := s.consistencyChecker.CheckQuotaOverviewConsistency(ctx, snapshot)
		if checkErr != nil {
			consistency = QuotaOverviewConsistency{Consistent: false, WarningCodes: []string{"quota_consistency_check_failed"}}
		} else {
			consistency = checked
		}
	}
	if !consistency.Consistent {
		overview.Warnings = appendUniqueStrings(overview.Warnings, consistency.WarningCodes...)
		degradeQuotaOverview(&overview, QuotaFreshnessUnknown, "data_unavailable")
		return &overview, nil
	}

	overview.Warnings = appendUniqueStrings(overview.Warnings, consistency.WarningCodes...)
	s.storeFresh(userID, displayTimezone, overview)
	return &overview, nil
}

func (s *QuotaOverviewService) storeFresh(userID int64, displayTimezone string, overview QuotaOverview) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	now := s.now()
	for key, cached := range s.lastGood {
		if now.Sub(cached.storedAt) > s.staleRetention {
			delete(s.lastGood, key)
		}
	}
	// Keep this process-local safety cache bounded. It contains only the
	// redacted overview, never credentials.
	if len(s.lastGood) >= 2048 {
		var oldestKey quotaOverviewCacheKey
		var oldestAt time.Time
		for key, cached := range s.lastGood {
			if oldestAt.IsZero() || cached.storedAt.Before(oldestAt) {
				oldestKey, oldestAt = key, cached.storedAt
			}
		}
		delete(s.lastGood, oldestKey)
	}
	key := quotaOverviewCacheKey{userID: userID, displayTimezone: displayTimezone}
	s.lastGood[key] = cachedQuotaOverview{overview: cloneQuotaOverview(overview), storedAt: now}
}

func (s *QuotaOverviewService) loadStale(userID int64, displayTimezone string) *QuotaOverview {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	key := quotaOverviewCacheKey{userID: userID, displayTimezone: displayTimezone}
	cached, ok := s.lastGood[key]
	if !ok {
		return nil
	}
	if s.now().Sub(cached.storedAt) > s.staleRetention {
		delete(s.lastGood, key)
		return nil
	}
	clone := cloneQuotaOverview(cached.overview)
	return &clone
}

type quotaResourceState struct {
	state      string
	reason     string
	action     string
	recoversAt *time.Time
	resourceID *string
}

func buildQuotaOverview(snapshot *QuotaOverviewSnapshot, generatedAt time.Time, displayTimezone string, freshFor time.Duration, actions QuotaOverviewActions) QuotaOverview {
	asOf := snapshot.AsOf.UTC()
	if displayTimezone == "" {
		displayTimezone = "UTC"
	}
	out := QuotaOverview{
		SchemaVersion:   QuotaOverviewSchemaVersion,
		GeneratedAt:     generatedAt.UTC(),
		AsOf:            asOf,
		FreshUntil:      quotaOverviewFreshUntil(snapshot, asOf, freshFor, displayTimezone),
		DisplayTimezone: displayTimezone,
		Freshness:       QuotaFreshnessFresh,
		Coverage: QuotaOverviewCoverage{
			Included: []string{
				"wallet",
				"account_spend_today",
				"account_spend_month_to_date",
				"api_key_billing_groups",
				"subscription_7d",
				"subscription_period_usage",
			},
			Excluded: []string{"routing", "rpm", "concurrency", "upstream_quota", "codex_quota"},
		},
		Account: QuotaOverviewAccount{
			// Only the masked label crosses the service/DTO boundary.
			DisplayLabel:   MaskEmail(snapshot.Account.Email),
			DataScope:      "all_enabled_api_keys",
			CanMakeRequest: nil,
		},
		Wallet: QuotaOverviewWallet{
			Unit:       "snow_credit",
			State:      QuotaWalletAvailable,
			Available:  fixedQuotaAmount(snapshot.Account.Balance),
			Reserved:   fixedQuotaAmount(snapshot.Account.FrozenBalance),
			TodaySpend: fixedQuotaAmount(snapshot.TodaySpend),
			MonthSpend: fixedQuotaAmount(snapshot.MonthSpend),
		},
		Actions:  actions,
		Warnings: []string{},
	}
	if snapshot.PeriodUsageAggregationError {
		out.Warnings = appendUniqueStrings(out.Warnings, "subscription_period_usage_unavailable")
	}
	if !snapshot.Account.Balance.GreaterThan(decimal.Zero) {
		out.Wallet.State = QuotaWalletExhausted
	}
	if snapshot.Account.FrozenBalance.IsNegative() {
		out.Wallet.State = QuotaWalletUnknown
		out.Warnings = appendUniqueStrings(out.Warnings, "invalid_wallet_snapshot")
	}

	subscriptionByGroup := make(map[int64]QuotaOverviewSubscriptionSnapshot, len(snapshot.Subscriptions))
	subscriptionStateByGroup := make(map[int64]quotaResourceState, len(snapshot.Subscriptions))
	for i := range snapshot.Subscriptions {
		sub := snapshot.Subscriptions[i]
		subscriptionByGroup[sub.GroupID] = sub
		view, state := buildSubscriptionOverview(sub, asOf)
		if view.WeeklyWindow.State != QuotaWindowUnknown &&
			view.PeriodUsage.State != QuotaPeriodUsageAvailable {
			out.Warnings = appendUniqueStrings(
				out.Warnings,
				"subscription_period_usage_unavailable",
			)
		}
		out.Subscriptions = append(out.Subscriptions, view)
		subscriptionStateByGroup[sub.GroupID] = state
	}

	type groupAccumulator struct {
		group    QuotaOverviewBillingGroup
		resource quotaResourceState
	}
	groups := make(map[string]*groupAccumulator)
	for i := range snapshot.Keys {
		key := snapshot.Keys[i]
		groupID, displayName, mode := quotaBillingGroupIdentity(key)
		acc, ok := groups[groupID]
		if !ok {
			resource := quotaResourceState{state: QuotaGroupUsable, action: "none"}
			resourceRef := QuotaOverviewResourceRef{Kind: "wallet"}
			if mode == QuotaBillingSubscription {
				resourceRef.Kind = "subscription"
				if key.GroupID != nil {
					if sub, exists := subscriptionByGroup[*key.GroupID]; exists {
						id := strconv.FormatInt(sub.ID, 10)
						resourceRef.ID = &id
					}
					if state, exists := subscriptionStateByGroup[*key.GroupID]; exists {
						resource = state
					} else {
						resource = quotaResourceState{
							state:  QuotaGroupBlocked,
							reason: "no_active_subscription",
							action: "renew_subscription",
						}
					}
				}
			} else if out.Wallet.State == QuotaWalletExhausted {
				resource = quotaResourceState{state: QuotaGroupBlocked, reason: "wallet_empty", action: "recharge"}
			} else if out.Wallet.State == QuotaWalletUnknown {
				resource = quotaResourceState{state: QuotaGroupUnknown, reason: "data_unavailable", action: "retry"}
			}
			if key.GroupID != nil && (key.Group == nil || key.Group.Deleted || key.Group.Status != StatusActive) {
				resource = quotaResourceState{state: QuotaGroupBlocked, reason: "billing_group_unavailable", action: "manage_api_keys"}
			}
			acc = &groupAccumulator{
				group: QuotaOverviewBillingGroup{
					ID:                groupID,
					DisplayName:       displayName,
					BillingMode:       mode,
					State:             QuotaGroupNoKeys,
					RecommendedAction: "none",
					FallbackPolicy:    "none",
					ResourceRef:       resourceRef,
					Keys:              []QuotaOverviewKey{},
				},
				resource: resource,
			}
			groups[groupID] = acc
		}

		keyState, _, _ := quotaKeyOwnState(key, asOf)
		if keyState == QuotaGroupUsable && acc.resource.state != QuotaGroupUsable {
			keyState = acc.resource.state
		}
		acc.group.Keys = append(acc.group.Keys, QuotaOverviewKey{
			ID:        strconv.FormatInt(key.ID, 10),
			Name:      key.Name,
			MaskedKey: key.MaskedKey,
			State:     keyState,
		})
		if mode == QuotaBillingBalance {
			out.Wallet.BalanceBilledKeyCount++
		}
	}

	// Active or historical subscriptions may exist without an enabled key.
	for i := range snapshot.Subscriptions {
		sub := snapshot.Subscriptions[i]
		groupID := strconv.FormatInt(sub.GroupID, 10)
		if _, exists := groups[groupID]; exists {
			continue
		}
		subID := strconv.FormatInt(sub.ID, 10)
		groups[groupID] = &groupAccumulator{
			group: QuotaOverviewBillingGroup{
				ID:                groupID,
				DisplayName:       sub.GroupName,
				BillingMode:       QuotaBillingSubscription,
				State:             QuotaGroupNoKeys,
				RecommendedAction: "none",
				FallbackPolicy:    "none",
				ResourceRef:       QuotaOverviewResourceRef{Kind: "subscription", ID: &subID},
				Keys:              []QuotaOverviewKey{},
			},
			resource: subscriptionStateByGroup[sub.GroupID],
		}
	}

	groupIDs := make([]string, 0, len(groups))
	for id := range groups {
		groupIDs = append(groupIDs, id)
	}
	sort.Strings(groupIDs)
	hasEnabledKeys := false
	hasUsable := false
	hasRestricted := false
	hasUnknown := false
	for _, id := range groupIDs {
		acc := groups[id]
		if len(acc.group.Keys) == 0 {
			out.BillingGroups = append(out.BillingGroups, acc.group)
			continue
		}
		hasEnabledKeys = true
		usable, blocked, unknown := 0, 0, 0
		var ownReason string
		for i := range acc.group.Keys {
			switch acc.group.Keys[i].State {
			case QuotaGroupUsable:
				usable++
			case QuotaGroupUnknown:
				unknown++
			default:
				blocked++
				if ownReason == "" {
					keySnapshot := findQuotaKeySnapshot(snapshot.Keys, acc.group.Keys[i].ID)
					_, ownReason, _ = quotaKeyOwnState(keySnapshot, asOf)
				}
			}
		}
		switch {
		case unknown > 0 && usable == 0:
			acc.group.State = QuotaGroupUnknown
		case usable > 0 && (blocked > 0 || unknown > 0):
			acc.group.State = QuotaGroupPartiallyUsable
		case usable > 0:
			acc.group.State = QuotaGroupUsable
		default:
			acc.group.State = QuotaGroupBlocked
		}
		reason, action := acc.resource.reason, acc.resource.action
		if reason == "" && ownReason != "" {
			reason = ownReason
			action = "manage_api_keys"
		}
		if acc.group.State != QuotaGroupUsable && acc.group.State != QuotaGroupNoKeys {
			acc.group.ReasonCode = stringPointer(reason)
			acc.group.RecommendedAction = action
		}
		switch acc.group.State {
		case QuotaGroupUsable:
			out.Account.UsableGroupCount++
			hasUsable = true
		case QuotaGroupPartiallyUsable:
			out.Account.UsableGroupCount++
			hasUsable, hasRestricted = true, true
		case QuotaGroupUnknown:
			out.Account.UnknownGroupCount++
			hasUnknown = true
		default:
			out.Account.BlockedGroupCount++
			hasRestricted = true
		}
		if out.Account.PrimaryIssue == nil && acc.group.State != QuotaGroupUsable {
			out.Account.PrimaryIssue = &QuotaOverviewIssue{
				ScopeType:         "billing_group",
				ScopeID:           acc.group.ID,
				ReasonCode:        reason,
				RecommendedAction: action,
				RecoversAt:        acc.resource.recoversAt,
			}
		}
		out.BillingGroups = append(out.BillingGroups, acc.group)
	}

	switch {
	case !hasEnabledKeys:
		out.Account.QuotaState = QuotaStateNoKeys
	case hasUsable && (hasRestricted || hasUnknown):
		out.Account.QuotaState = QuotaStatePartial
	case hasUsable:
		out.Account.QuotaState = QuotaStateAllAvailable
	case hasUnknown:
		out.Account.QuotaState = QuotaStateUnknown
	default:
		out.Account.QuotaState = QuotaStateAllBlocked
	}
	return out
}

func buildSubscriptionOverview(sub QuotaOverviewSubscriptionSnapshot, asOf time.Time) (QuotaOverviewSubscription, quotaResourceState) {
	id := strconv.FormatInt(sub.ID, 10)
	groupID := strconv.FormatInt(sub.GroupID, 10)
	out := QuotaOverviewSubscription{
		ID:        id,
		GroupID:   groupID,
		Name:      sub.GroupName,
		Status:    normalizedQuotaSubscriptionStatus(sub, asOf),
		StartsAt:  sub.StartsAt.UTC(),
		ExpiresAt: sub.ExpiresAt.UTC(),
		WeeklyWindow: QuotaOverviewWeeklyWindow{
			Kind:     "7d_from_subscription_start",
			State:    QuotaWindowUnknown,
			AnchorAt: sub.StartsAt.UTC(),
		},
		PeriodUsage: unknownQuotaOverviewPeriodUsage(),
	}
	state := quotaResourceState{state: QuotaGroupUnknown, reason: "data_unavailable", action: "retry", resourceID: &id}
	if sub.StartsAt.IsZero() || sub.ExpiresAt.IsZero() || !sub.ExpiresAt.After(sub.StartsAt) {
		out.Status = "unknown"
		return out, state
	}
	switch out.Status {
	case SubscriptionStatusExpired:
		state = quotaResourceState{state: QuotaGroupBlocked, reason: "subscription_expired", action: "renew_subscription", resourceID: &id}
		return out, state
	case SubscriptionStatusSuspended:
		state = quotaResourceState{state: QuotaGroupBlocked, reason: "subscription_suspended", action: "contact_support", resourceID: &id}
		return out, state
	case SubscriptionStatusRevoked:
		state = quotaResourceState{state: QuotaGroupBlocked, reason: "subscription_revoked", action: "renew_subscription", resourceID: &id}
		return out, state
	case SubscriptionStatusActive:
	default:
		out.Status = "unknown"
		return out, state
	}
	if asOf.Before(sub.StartsAt) || sub.WeeklyLimit == nil || !sub.WeeklyLimit.GreaterThan(decimal.Zero) {
		return out, state
	}

	periodStart, periodEnd, ok := AnchoredWeeklyWindow(sub.StartsAt, asOf)
	if !ok {
		return out, state
	}
	out.WeeklyWindow.PeriodStart = timePointer(periodStart)
	out.WeeklyWindow.PeriodEnd = timePointer(periodEnd)
	out.WeeklyWindow.ResetsAt = timePointer(periodEnd)
	if sub.WeeklyWindowStart == nil || !sub.WeeklyWindowStart.Equal(periodStart) {
		return out, state
	}
	limit := *sub.WeeklyLimit
	used := sub.WeeklyUsed
	limitString := fixedQuotaAmount(limit)
	usedString := fixedQuotaAmount(used)
	out.WeeklyWindow.Limit = &limitString
	out.WeeklyWindow.Used = &usedString
	if used.IsNegative() {
		return out, state
	}
	remaining := limit.Sub(used)
	if remaining.IsNegative() {
		remaining = decimal.Zero
	}
	remainingString := fixedQuotaAmount(remaining)
	percentDecimal := used.Mul(decimal.NewFromInt(100)).Div(limit)
	percent, _ := percentDecimal.Float64()
	if percent > 100 {
		percent = 100
	}
	if percent < 0 {
		percent = 0
	}
	out.WeeklyWindow.Remaining = &remainingString
	out.WeeklyWindow.UsedPercent = &percent
	out.WeeklyWindow.State = QuotaWindowActive
	out.PeriodUsage = buildQuotaOverviewPeriodUsage(
		sub.PeriodUsage,
		periodStart,
		periodEnd,
		asOf,
		sub.ExpiresAt,
	)
	state = quotaResourceState{state: QuotaGroupUsable, action: "none", resourceID: &id}
	if !used.LessThan(limit) {
		out.WeeklyWindow.State = QuotaWindowExhausted
		state = quotaResourceState{
			state: QuotaGroupBlocked, reason: "subscription_weekly_exhausted",
			action: "manage_api_keys", resourceID: &id,
		}
		// A reset cannot restore availability after the entitlement has
		// already expired. Avoid advertising a false recovery time.
		if periodEnd.Before(sub.ExpiresAt) {
			reset := periodEnd
			state.recoversAt = &reset
		}
	}
	if sub.ExpiresAt.Before(periodEnd) {
		out.NextEvent = &QuotaOverviewNextEvent{Kind: "expiry", At: sub.ExpiresAt.UTC()}
	} else {
		out.NextEvent = &QuotaOverviewNextEvent{Kind: "reset", At: periodEnd}
	}
	return out, state
}

// normalizeQuotaOverviewElapsedEmptyWindows mirrors request admission's
// EffectiveWeeklyUsageAt rule for one narrow maintenance gap: the persisted
// marker is an exact, older anchored period and the current period has no
// successful usage at all. In that case admission treats the stale counter as
// zero, so the read model may project the current period without writing DB.
//
// The original marker is retained as evidence for validation and cache
// consistency checks. A missing, future or off-anchor marker, unavailable
// period usage, or any current-period activity remains unknown.
func normalizeQuotaOverviewElapsedEmptyWindows(snapshot *QuotaOverviewSnapshot) {
	if snapshot == nil {
		return
	}
	asOf := snapshot.AsOf.UTC()
	for i := range snapshot.Subscriptions {
		sub := &snapshot.Subscriptions[i]
		if sub.Revoked ||
			sub.Status != SubscriptionStatusActive ||
			asOf.Before(sub.StartsAt) ||
			!asOf.Before(sub.ExpiresAt) ||
			sub.WeeklyWindowStart == nil {
			continue
		}
		periodStart, periodEnd, ok := AnchoredWeeklyWindow(sub.StartsAt, asOf)
		if !ok ||
			!quotaOverviewIsStrictPastAnchoredWindow(
				sub.StartsAt,
				*sub.WeeklyWindowStart,
				periodStart,
			) ||
			!quotaOverviewPeriodUsageProvesEmpty(
				sub.PeriodUsage,
				periodStart,
				periodEnd,
				asOf,
				sub.ExpiresAt,
			) {
			continue
		}

		projectedFrom := sub.WeeklyWindowStart.UTC()
		currentStart := periodStart.UTC()
		sub.WeeklyWindowProjectedFrom = &projectedFrom
		sub.WeeklyWindowStart = &currentStart
		sub.WeeklyUsed = decimal.Zero
	}
}

func quotaOverviewIsStrictPastAnchoredWindow(
	anchor time.Time,
	candidate time.Time,
	currentStart time.Time,
) bool {
	if !candidate.Before(currentStart) {
		return false
	}
	candidateStart, _, ok := AnchoredWeeklyWindow(anchor, candidate)
	return ok && candidate.Equal(candidateStart)
}

func quotaOverviewPeriodUsageProvesEmpty(
	usage *QuotaOverviewPeriodUsageSnapshot,
	periodStart time.Time,
	periodEnd time.Time,
	asOf time.Time,
	expiresAt time.Time,
) bool {
	expectedObservedUntil := minQuotaOverviewUsageTime(asOf, periodEnd, expiresAt)
	if usage == nil ||
		!usage.PeriodStart.Equal(periodStart) ||
		!usage.PeriodEnd.Equal(periodEnd) ||
		!usage.ObservedUntil.Equal(expectedObservedUntil) ||
		usage.ObservedUntil.Before(periodStart) {
		return false
	}
	for i := range usage.Buckets {
		bucket := usage.Buckets[i]
		if bucket.Requests != 0 ||
			bucket.CacheHitTokens != 0 ||
			bucket.CacheMissTokens != 0 ||
			bucket.OutputTokens != 0 {
			return false
		}
	}
	return true
}

func unknownQuotaOverviewPeriodUsage() QuotaOverviewPeriodUsage {
	return QuotaOverviewPeriodUsage{
		State:      QuotaPeriodUsageUnknown,
		BucketKind: QuotaPeriodUsageBucketAnchored24h,
		Points:     nil,
	}
}

func buildQuotaOverviewPeriodUsage(
	usage *QuotaOverviewPeriodUsageSnapshot,
	periodStart time.Time,
	periodEnd time.Time,
	asOf time.Time,
	expiresAt time.Time,
) QuotaOverviewPeriodUsage {
	out := unknownQuotaOverviewPeriodUsage()
	expectedObservedUntil := minQuotaOverviewUsageTime(asOf, periodEnd, expiresAt)
	if usage == nil ||
		!usage.PeriodStart.Equal(periodStart) ||
		!usage.PeriodEnd.Equal(periodEnd) ||
		!usage.ObservedUntil.Equal(expectedObservedUntil) ||
		usage.ObservedUntil.Before(periodStart) {
		return out
	}

	totalRequests := int64(0)
	totalTokens := int64(0)
	points := make([]QuotaOverviewPeriodUsagePoint, 0, len(usage.Buckets))
	for i := range usage.Buckets {
		startAt := periodStart.Add(time.Duration(i) * 24 * time.Hour)
		endAt := startAt.Add(24 * time.Hour)
		point := QuotaOverviewPeriodUsagePoint{
			Index:   i + 1,
			StartAt: startAt,
			EndAt:   endAt,
			State:   QuotaPeriodUsagePointFuture,
		}
		if startAt.Before(usage.ObservedUntil) {
			bucket := usage.Buckets[i]
			if bucket.Requests < 0 ||
				bucket.CacheHitTokens < 0 ||
				bucket.CacheMissTokens < 0 ||
				bucket.OutputTokens < 0 {
				return out
			}
			point.State = QuotaPeriodUsagePointPartial
			if !endAt.After(usage.ObservedUntil) {
				point.State = QuotaPeriodUsagePointComplete
			}
			pointTotal, ok := addQuotaOverviewNonNegativeInt64(
				bucket.CacheHitTokens,
				bucket.CacheMissTokens,
				bucket.OutputTokens,
			)
			if !ok {
				return out
			}
			nextTotalRequests, ok := addQuotaOverviewNonNegativeInt64(
				totalRequests,
				bucket.Requests,
			)
			if !ok {
				return out
			}
			nextTotalTokens, ok := addQuotaOverviewNonNegativeInt64(totalTokens, pointTotal)
			if !ok {
				return out
			}
			point.Requests = int64Pointer(bucket.Requests)
			point.CacheHitTokens = int64Pointer(bucket.CacheHitTokens)
			point.CacheMissTokens = int64Pointer(bucket.CacheMissTokens)
			point.OutputTokens = int64Pointer(bucket.OutputTokens)
			point.TotalTokens = int64Pointer(pointTotal)
			totalRequests = nextTotalRequests
			totalTokens = nextTotalTokens
		}
		points = append(points, point)
	}

	observedUntil := usage.ObservedUntil.UTC()
	out.State = QuotaPeriodUsageAvailable
	out.ObservedUntil = &observedUntil
	out.TotalRequests = int64Pointer(totalRequests)
	out.TotalTokens = int64Pointer(totalTokens)
	out.Points = points
	return out
}

func addQuotaOverviewNonNegativeInt64(values ...int64) (int64, bool) {
	total := int64(0)
	for _, value := range values {
		if value < 0 || total > math.MaxInt64-value {
			return 0, false
		}
		total += value
	}
	return total, true
}

func minQuotaOverviewUsageTime(values ...time.Time) time.Time {
	if len(values) == 0 {
		return time.Time{}
	}
	minimum := values[0]
	for i := 1; i < len(values); i++ {
		if values[i].Before(minimum) {
			minimum = values[i]
		}
	}
	return minimum
}

func normalizedQuotaSubscriptionStatus(sub QuotaOverviewSubscriptionSnapshot, asOf time.Time) string {
	if sub.Revoked {
		return SubscriptionStatusRevoked
	}
	if sub.Status == SubscriptionStatusActive && !asOf.Before(sub.ExpiresAt) {
		return SubscriptionStatusExpired
	}
	switch sub.Status {
	case SubscriptionStatusActive, SubscriptionStatusExpired, SubscriptionStatusSuspended, SubscriptionStatusRevoked:
		return sub.Status
	default:
		return "unknown"
	}
}

func quotaBillingGroupIdentity(key QuotaOverviewKeySnapshot) (id, name, mode string) {
	if key.GroupID == nil {
		return "balance_default", "标准计费", QuotaBillingBalance
	}
	id = strconv.FormatInt(*key.GroupID, 10)
	if key.Group == nil {
		return id, "未知分组", QuotaBillingBalance
	}
	name = key.Group.Name
	if key.Group.SubscriptionType == SubscriptionTypeSubscription {
		mode = QuotaBillingSubscription
	} else {
		mode = QuotaBillingBalance
	}
	return
}

func quotaKeyOwnState(key QuotaOverviewKeySnapshot, asOf time.Time) (state, reason, action string) {
	if key.Quota.IsNegative() || key.QuotaUsed.IsNegative() {
		return QuotaGroupUnknown, "data_unavailable", "retry"
	}
	switch key.Status {
	case StatusAPIKeyQuotaExhausted:
		return QuotaGroupBlocked, "api_key_limit_exhausted", "manage_api_keys"
	case StatusAPIKeyExpired:
		return QuotaGroupBlocked, "api_key_expired", "manage_api_keys"
	case StatusAPIKeyActive:
	default:
		return QuotaGroupUnknown, "data_unavailable", "retry"
	}
	if key.ExpiresAt != nil && !asOf.Before(*key.ExpiresAt) {
		return QuotaGroupBlocked, "api_key_expired", "manage_api_keys"
	}
	if key.Quota.GreaterThan(decimal.Zero) && !key.QuotaUsed.LessThan(key.Quota) {
		return QuotaGroupBlocked, "api_key_limit_exhausted", "manage_api_keys"
	}
	return QuotaGroupUsable, "", "none"
}

func findQuotaKeySnapshot(keys []QuotaOverviewKeySnapshot, id string) QuotaOverviewKeySnapshot {
	for i := range keys {
		if strconv.FormatInt(keys[i].ID, 10) == id {
			return keys[i]
		}
	}
	return QuotaOverviewKeySnapshot{}
}

func fixedQuotaAmount(value decimal.Decimal) string {
	return value.StringFixed(10)
}

func quotaOverviewFreshUntil(
	snapshot *QuotaOverviewSnapshot,
	asOf time.Time,
	freshFor time.Duration,
	displayTimezone string,
) time.Time {
	freshUntil := asOf.Add(freshFor)
	clamp := func(boundary time.Time) {
		boundary = boundary.UTC()
		if boundary.After(asOf) && boundary.Before(freshUntil) {
			freshUntil = boundary
		}
	}
	if nextMidnight, ok := quotaOverviewNextLocalMidnight(asOf, displayTimezone); ok {
		clamp(nextMidnight)
	}
	for i := range snapshot.Keys {
		if snapshot.Keys[i].ExpiresAt != nil {
			clamp(*snapshot.Keys[i].ExpiresAt)
		}
	}
	for i := range snapshot.Subscriptions {
		sub := snapshot.Subscriptions[i]
		if sub.Revoked {
			continue
		}
		clamp(sub.StartsAt)
		clamp(sub.ExpiresAt)
		if _, periodEnd, ok := AnchoredWeeklyWindow(sub.StartsAt, asOf); ok {
			clamp(periodEnd)
		}
	}
	return freshUntil
}

func quotaOverviewNextLocalMidnight(asOf time.Time, displayTimezone string) (time.Time, bool) {
	if displayTimezone == "" {
		displayTimezone = "UTC"
	}
	location, err := time.LoadLocation(displayTimezone)
	if err != nil {
		return time.Time{}, false
	}
	local := asOf.In(location)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
	return start.AddDate(0, 0, 1).UTC(), true
}

func degradeQuotaOverview(overview *QuotaOverview, freshness, warning string) {
	if overview == nil {
		return
	}
	overview.Freshness = freshness
	overview.Warnings = appendUniqueStrings(overview.Warnings, warning)
	overview.Account.QuotaState = QuotaStateUnknown
	overview.Account.UsableGroupCount = 0
	overview.Account.BlockedGroupCount = 0
	overview.Account.UnknownGroupCount = 0
	overview.Account.PrimaryIssue = &QuotaOverviewIssue{
		ScopeType:         "account",
		ScopeID:           "",
		ReasonCode:        warning,
		RecommendedAction: "retry",
	}
	overview.Wallet.State = QuotaWalletUnknown
	for i := range overview.BillingGroups {
		group := &overview.BillingGroups[i]
		if len(group.Keys) == 0 {
			continue
		}
		group.State = QuotaGroupUnknown
		reason := warning
		group.ReasonCode = &reason
		group.RecommendedAction = "retry"
		overview.Account.UnknownGroupCount++
		for j := range group.Keys {
			group.Keys[j].State = QuotaGroupUnknown
		}
	}
	for i := range overview.Subscriptions {
		overview.Subscriptions[i].WeeklyWindow.State = QuotaWindowUnknown
		overview.Subscriptions[i].WeeklyWindow.UsedPercent = nil
		overview.Subscriptions[i].PeriodUsage = unknownQuotaOverviewPeriodUsage()
	}
}

func cloneQuotaOverview(in QuotaOverview) QuotaOverview {
	out := in
	out.Coverage.Included = append([]string(nil), in.Coverage.Included...)
	out.Coverage.Excluded = append([]string(nil), in.Coverage.Excluded...)
	out.Warnings = append([]string(nil), in.Warnings...)
	if in.Account.PrimaryIssue != nil {
		issue := *in.Account.PrimaryIssue
		out.Account.PrimaryIssue = &issue
	}
	out.BillingGroups = make([]QuotaOverviewBillingGroup, len(in.BillingGroups))
	for i := range in.BillingGroups {
		out.BillingGroups[i] = in.BillingGroups[i]
		out.BillingGroups[i].Keys = append([]QuotaOverviewKey(nil), in.BillingGroups[i].Keys...)
	}
	out.Subscriptions = make([]QuotaOverviewSubscription, len(in.Subscriptions))
	copy(out.Subscriptions, in.Subscriptions)
	for i := range out.Subscriptions {
		if in.Subscriptions[i].NextEvent != nil {
			event := *in.Subscriptions[i].NextEvent
			out.Subscriptions[i].NextEvent = &event
		}
		if in.Subscriptions[i].PeriodUsage.ObservedUntil != nil {
			observedUntil := *in.Subscriptions[i].PeriodUsage.ObservedUntil
			out.Subscriptions[i].PeriodUsage.ObservedUntil = &observedUntil
		}
		if in.Subscriptions[i].PeriodUsage.TotalRequests != nil {
			totalRequests := *in.Subscriptions[i].PeriodUsage.TotalRequests
			out.Subscriptions[i].PeriodUsage.TotalRequests = &totalRequests
		}
		if in.Subscriptions[i].PeriodUsage.TotalTokens != nil {
			totalTokens := *in.Subscriptions[i].PeriodUsage.TotalTokens
			out.Subscriptions[i].PeriodUsage.TotalTokens = &totalTokens
		}
		out.Subscriptions[i].PeriodUsage.Points = cloneQuotaOverviewPeriodUsagePoints(
			in.Subscriptions[i].PeriodUsage.Points,
		)
	}
	return out
}

func cloneQuotaOverviewPeriodUsagePoints(
	in []QuotaOverviewPeriodUsagePoint,
) []QuotaOverviewPeriodUsagePoint {
	if in == nil {
		return nil
	}
	out := make([]QuotaOverviewPeriodUsagePoint, len(in))
	for i := range in {
		out[i] = in[i]
		out[i].Requests = cloneQuotaOverviewInt64Pointer(in[i].Requests)
		out[i].CacheHitTokens = cloneQuotaOverviewInt64Pointer(in[i].CacheHitTokens)
		out[i].CacheMissTokens = cloneQuotaOverviewInt64Pointer(in[i].CacheMissTokens)
		out[i].OutputTokens = cloneQuotaOverviewInt64Pointer(in[i].OutputTokens)
		out[i].TotalTokens = cloneQuotaOverviewInt64Pointer(in[i].TotalTokens)
	}
	return out
}

func cloneQuotaOverviewInt64Pointer(value *int64) *int64 {
	if value == nil {
		return nil
	}
	return int64Pointer(*value)
}

func appendUniqueStrings(values []string, additions ...string) []string {
	seen := make(map[string]struct{}, len(values)+len(additions))
	for _, value := range values {
		if value != "" {
			seen[value] = struct{}{}
		}
	}
	for _, value := range additions {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		values = append(values, value)
		seen[value] = struct{}{}
	}
	return values
}

func stringPointer(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func timePointer(value time.Time) *time.Time {
	value = value.UTC()
	return &value
}

func int64Pointer(value int64) *int64 {
	return &value
}

func IsQuotaOverviewUnavailable(err error) bool {
	return errors.Is(err, ErrQuotaOverviewUnavailable)
}
