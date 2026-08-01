package service

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/shopspring/decimal"
)

type quotaOverviewBillingCache interface {
	GetUserBalance(ctx context.Context, userID int64) (float64, error)
	GetSubscriptionCache(ctx context.Context, userID, groupID int64) (*SubscriptionCacheData, error)
}

type billingQuotaOverviewConsistencyChecker struct {
	cache quotaOverviewBillingCache
}

func NewBillingQuotaOverviewConsistencyChecker(cache quotaOverviewBillingCache) QuotaOverviewConsistencyChecker {
	return &billingQuotaOverviewConsistencyChecker{cache: cache}
}

// CheckQuotaOverviewConsistency compares only cache entries that exist.
// A normal miss is consistent because request admission also falls back to the
// authoritative DB. Redis errors, malformed entries, or differing values are
// not safe to present as current.
func (c *billingQuotaOverviewConsistencyChecker) CheckQuotaOverviewConsistency(
	ctx context.Context,
	snapshot *QuotaOverviewSnapshot,
) (QuotaOverviewConsistency, error) {
	if c == nil || c.cache == nil {
		return QuotaOverviewConsistency{
			Consistent:   false,
			WarningCodes: []string{"quota_consistency_unverified"},
		}, nil
	}

	cachedBalance, err := c.cache.GetUserBalance(ctx, snapshot.Account.ID)
	switch {
	case errors.Is(err, ErrBillingCacheMiss):
		// A cache miss is not an alternate source of truth.
	case err != nil:
		return inconsistentQuotaOverview("quota_balance_cache_unavailable"), nil
	case math.IsNaN(cachedBalance), math.IsInf(cachedBalance, 0):
		return inconsistentQuotaOverview("quota_balance_cache_mismatch"), nil
	case decimal.NewFromFloat(cachedBalance).StringFixed(10) != snapshot.Account.Balance.StringFixed(10):
		return inconsistentQuotaOverview("quota_balance_cache_mismatch"), nil
	}

	subscriptionByGroup := make(map[int64]QuotaOverviewSubscriptionSnapshot, len(snapshot.Subscriptions))
	groupsToCheck := make(map[int64]struct{}, len(snapshot.Subscriptions))
	for i := range snapshot.Subscriptions {
		sub := snapshot.Subscriptions[i]
		subscriptionByGroup[sub.GroupID] = sub
		groupsToCheck[sub.GroupID] = struct{}{}
	}
	for i := range snapshot.Keys {
		key := snapshot.Keys[i]
		if key.GroupID == nil || key.Group == nil || key.Group.SubscriptionType != SubscriptionTypeSubscription {
			continue
		}
		groupsToCheck[*key.GroupID] = struct{}{}
	}
	for groupID := range groupsToCheck {
		cached, cacheErr := c.cache.GetSubscriptionCache(ctx, snapshot.Account.ID, groupID)
		if errors.Is(cacheErr, ErrBillingCacheMiss) {
			continue
		}
		if cacheErr != nil {
			return inconsistentQuotaOverview("quota_subscription_cache_unavailable"), nil
		}
		sub, exists := subscriptionByGroup[groupID]
		if !exists || !quotaOverviewSubscriptionIsCurrent(sub, snapshot.AsOf) {
			// Revoked, expired, suspended and missing entitlements must not
			// leave any Redis admission state behind.
			if cached != nil {
				return inconsistentQuotaOverview("quota_subscription_cache_mismatch"), nil
			}
			continue
		}
		if cached == nil {
			// Normal eviction/miss: admission will use the same DB row.
			continue
		}
		if sub.WeeklyWindowProjectedFrom != nil || sub.MonthlyWindowProjectedFrom != nil {
			// A projected read is allowed only when Redis is absent. Any cache
			// entry would be a second source of admission state that cannot be
			// proven identical to the still-unmaintained DB row.
			return inconsistentQuotaOverview("quota_subscription_cache_mismatch"), nil
		}
		if !quotaSubscriptionCacheMatchesSnapshot(cached, sub, snapshot.AsOf) {
			return inconsistentQuotaOverview("quota_subscription_cache_mismatch"), nil
		}
	}
	return QuotaOverviewConsistency{Consistent: true}, nil
}

func quotaOverviewSubscriptionIsCurrent(sub QuotaOverviewSubscriptionSnapshot, asOf time.Time) bool {
	return !sub.Revoked &&
		sub.Status == SubscriptionStatusActive &&
		!asOf.Before(sub.StartsAt) &&
		asOf.Before(sub.ExpiresAt)
}

func quotaSubscriptionCacheMatchesSnapshot(
	cached *SubscriptionCacheData,
	sub QuotaOverviewSubscriptionSnapshot,
	asOf time.Time,
) bool {
	if cached == nil ||
		cached.WeeklyWindowStart == nil ||
		sub.WeeklyWindowStart == nil ||
		cached.MonthlyWindowStart == nil ||
		sub.MonthlyWindowStart == nil {
		return false
	}
	weeklyPeriodStart, weeklyPeriodEnd, ok := AnchoredWeeklyWindow(sub.StartsAt, asOf)
	if !ok {
		return false
	}
	monthlyPeriodStart, monthlyPeriodEnd, ok := AnchoredMonthlyWindow(sub.StartsAt, asOf)
	if !ok {
		return false
	}
	if !sub.WeeklyWindowStart.Equal(weeklyPeriodStart) ||
		!sub.MonthlyWindowStart.Equal(monthlyPeriodStart) ||
		cached.SubscriptionID != sub.ID ||
		!cached.StartsAt.Equal(sub.StartsAt) ||
		!cached.ExpiresAt.Equal(sub.ExpiresAt) ||
		!cached.WeeklyWindowStart.Equal(weeklyPeriodStart) ||
		!cached.WeeklyWindowEnd.Equal(weeklyPeriodEnd) ||
		!cached.MonthlyWindowStart.Equal(monthlyPeriodStart) ||
		!cached.MonthlyWindowEnd.Equal(monthlyPeriodEnd) ||
		cached.Status != sub.Status ||
		sub.UpdatedAt.IsZero() ||
		cached.Version != sub.UpdatedAt.UnixMicro() ||
		math.IsNaN(cached.WeeklyUsage) ||
		math.IsInf(cached.WeeklyUsage, 0) ||
		decimal.NewFromFloat(cached.WeeklyUsage).StringFixed(10) != sub.WeeklyUsed.StringFixed(10) ||
		math.IsNaN(cached.MonthlyUsage) ||
		math.IsInf(cached.MonthlyUsage, 0) ||
		decimal.NewFromFloat(cached.MonthlyUsage).StringFixed(10) != sub.MonthlyUsed.StringFixed(10) {
		return false
	}
	return true
}

func inconsistentQuotaOverview(warning string) QuotaOverviewConsistency {
	return QuotaOverviewConsistency{
		Consistent:   false,
		WarningCodes: []string{warning},
	}
}

func validQuotaOverviewSnapshot(snapshot *QuotaOverviewSnapshot, expectedUserID int64) bool {
	if snapshot == nil ||
		snapshot.AsOf.IsZero() ||
		snapshot.Account.ID != expectedUserID ||
		snapshot.Account.Status != StatusActive ||
		snapshot.Account.Balance.IsNegative() ||
		snapshot.Account.FrozenBalance.IsNegative() ||
		snapshot.TodaySpend.IsNegative() ||
		snapshot.MonthSpend.IsNegative() {
		return false
	}
	for i := range snapshot.Keys {
		key := snapshot.Keys[i]
		if key.ID <= 0 || key.Quota.IsNegative() || key.QuotaUsed.IsNegative() {
			return false
		}
		if key.GroupID == nil {
			if key.Group != nil {
				return false
			}
			continue
		}
		if key.Group == nil ||
			key.Group.ID <= 0 ||
			*key.GroupID != key.Group.ID ||
			(key.Group.SubscriptionType != SubscriptionTypeStandard &&
				key.Group.SubscriptionType != SubscriptionTypeSubscription) {
			// A grouped key whose source row cannot be identified must not be
			// guessed as wallet-backed. Degrade the entire availability
			// snapshot to unknown instead.
			return false
		}
	}

	groups := make(map[int64]struct{}, len(snapshot.Subscriptions))
	for i := range snapshot.Subscriptions {
		sub := snapshot.Subscriptions[i]
		if sub.ID <= 0 ||
			sub.GroupID <= 0 ||
			sub.StartsAt.IsZero() ||
			sub.ExpiresAt.IsZero() ||
			!sub.ExpiresAt.After(sub.StartsAt) ||
			sub.WeeklyUsed.IsNegative() ||
			sub.MonthlyUsed.IsNegative() {
			return false
		}
		if _, duplicate := groups[sub.GroupID]; duplicate {
			return false
		}
		groups[sub.GroupID] = struct{}{}
		if sub.WeeklyLimit != nil && sub.WeeklyLimit.IsNegative() {
			return false
		}
		if sub.MonthlyLimit != nil && sub.MonthlyLimit.IsNegative() {
			return false
		}
		if sub.Revoked || sub.Status != SubscriptionStatusActive || !snapshot.AsOf.Before(sub.ExpiresAt) {
			continue
		}
		weeklyPeriodStart, _, weeklyOK := AnchoredWeeklyWindow(sub.StartsAt, snapshot.AsOf)
		monthlyPeriodStart, _, monthlyOK := AnchoredMonthlyWindow(sub.StartsAt, snapshot.AsOf)
		_, hasMonthlyLimit := quotaOverviewEffectiveMonthlyLimit(sub)
		if !weeklyOK ||
			!monthlyOK ||
			sub.WeeklyWindowStart == nil ||
			!sub.WeeklyWindowStart.Equal(weeklyPeriodStart) ||
			sub.MonthlyWindowStart == nil ||
			!sub.MonthlyWindowStart.Equal(monthlyPeriodStart) ||
			sub.WeeklyLimit == nil ||
			sub.WeeklyLimit.IsNegative() ||
			!hasMonthlyLimit ||
			sub.UpdatedAt.IsZero() {
			return false
		}
		if sub.WeeklyWindowProjectedFrom != nil {
			_, periodEnd, ok := AnchoredWeeklyWindow(sub.StartsAt, snapshot.AsOf)
			if !ok ||
				!quotaOverviewIsStrictPastAnchoredWindow(
					sub.StartsAt,
					*sub.WeeklyWindowProjectedFrom,
					weeklyPeriodStart,
				) ||
				!sub.WeeklyUsed.IsZero() ||
				!quotaOverviewPeriodUsageProvesEmpty(
					sub.PeriodUsage,
					weeklyPeriodStart,
					periodEnd,
					snapshot.AsOf,
					sub.ExpiresAt,
				) {
				return false
			}
		}
		if sub.MonthlyWindowProjectedFrom != nil &&
			(!quotaOverviewIsStrictPastMonthlyAnchoredWindow(
				sub.StartsAt,
				*sub.MonthlyWindowProjectedFrom,
				monthlyPeriodStart,
			) ||
				!sub.MonthlyUsed.IsZero()) {
			return false
		}
	}
	return true
}
