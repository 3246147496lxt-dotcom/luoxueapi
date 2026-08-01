//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

type quotaOverviewRepoStub struct {
	snapshots []*QuotaOverviewSnapshot
	errs      []error
	timezones []string
	calls     int
}

func (s *quotaOverviewRepoStub) LoadQuotaOverviewSnapshot(_ context.Context, _ int64, displayTimezone string) (*QuotaOverviewSnapshot, error) {
	index := s.calls
	s.calls++
	s.timezones = append(s.timezones, displayTimezone)
	if index < len(s.errs) && s.errs[index] != nil {
		return nil, s.errs[index]
	}
	if index >= len(s.snapshots) {
		return nil, errors.New("snapshot not configured")
	}
	return s.snapshots[index], nil
}

type quotaOverviewCheckerStub struct {
	result QuotaOverviewConsistency
	err    error
}

func (s quotaOverviewCheckerStub) CheckQuotaOverviewConsistency(context.Context, *QuotaOverviewSnapshot) (QuotaOverviewConsistency, error) {
	return s.result, s.err
}

type quotaOverviewCacheStub struct {
	balance    float64
	balanceErr error
	subs       map[int64]*SubscriptionCacheData
	subErrs    map[int64]error
}

func (s *quotaOverviewCacheStub) GetUserBalance(context.Context, int64) (float64, error) {
	return s.balance, s.balanceErr
}

func (s *quotaOverviewCacheStub) GetSubscriptionCache(_ context.Context, _, groupID int64) (*SubscriptionCacheData, error) {
	if err := s.subErrs[groupID]; err != nil {
		return nil, err
	}
	if sub, ok := s.subs[groupID]; ok {
		return sub, nil
	}
	return nil, redis.Nil
}

func quotaOverviewTestSnapshot(asOf time.Time) *QuotaOverviewSnapshot {
	anchor := time.Date(2026, 7, 25, 9, 30, 0, 0, time.UTC)
	windowStart, windowEnd, _ := AnchoredWeeklyWindow(anchor, asOf)
	monthlyWindowStart, _, _ := AnchoredMonthlyWindow(anchor, asOf)
	limit := decimal.RequireFromString("200")
	balanceGroupID := int64(10)
	memberGroupID := int64(20)
	periodUsage := &QuotaOverviewPeriodUsageSnapshot{
		PeriodStart:   windowStart,
		PeriodEnd:     windowEnd,
		ObservedUntil: asOf,
	}
	periodUsage.Buckets[0] = QuotaOverviewPeriodUsageBucketSnapshot{
		Requests: 58, CacheHitTokens: 92000, CacheMissTokens: 41000, OutputTokens: 49000,
	}
	periodUsage.Buckets[1] = QuotaOverviewPeriodUsageBucketSnapshot{
		Requests: 66, CacheHitTokens: 105000, CacheMissTokens: 52000, OutputTokens: 57000,
	}
	periodUsage.Buckets[2] = QuotaOverviewPeriodUsageBucketSnapshot{
		Requests: 73, CacheHitTokens: 118000, CacheMissTokens: 57000, OutputTokens: 71000,
	}
	periodUsage.Buckets[3] = QuotaOverviewPeriodUsageBucketSnapshot{
		Requests: 82, CacheHitTokens: 128000, CacheMissTokens: 61000, OutputTokens: 70000,
	}
	return &QuotaOverviewSnapshot{
		AsOf: asOf,
		Account: QuotaOverviewAccountSnapshot{
			ID:            42,
			Email:         "pu@example.com",
			Status:        StatusActive,
			Balance:       decimal.RequireFromString("128.64"),
			FrozenBalance: decimal.Zero,
		},
		TodaySpend: decimal.RequireFromString("1.16"),
		MonthSpend: decimal.RequireFromString("10.74"),
		Keys: []QuotaOverviewKeySnapshot{
			{
				ID: 101, Name: "工作电脑", MaskedKey: "sk-••••7A9C", Status: StatusAPIKeyActive,
				GroupID: &balanceGroupID,
				Group: &QuotaOverviewGroupSnapshot{
					ID: balanceGroupID, Name: "标准计费", Status: StatusActive,
					SubscriptionType: SubscriptionTypeStandard,
				},
			},
			{
				ID: 202, Name: "Codex", MaskedKey: "sk-••••42FD", Status: StatusAPIKeyActive,
				GroupID: &memberGroupID,
				Group: &QuotaOverviewGroupSnapshot{
					ID: memberGroupID, Name: "Pro 会员", Status: StatusActive,
					SubscriptionType: SubscriptionTypeSubscription,
				},
			},
		},
		Subscriptions: []QuotaOverviewSubscriptionSnapshot{
			{
				ID: 8, GroupID: memberGroupID, GroupName: "Pro 会员", GroupStatus: StatusActive,
				Status: SubscriptionStatusActive, StartsAt: anchor,
				ExpiresAt:          anchor.Add(31 * 24 * time.Hour),
				WeeklyWindowStart:  &windowStart,
				WeeklyLimit:        &limit,
				WeeklyUsed:         decimal.RequireFromString("200"),
				MonthlyWindowStart: &monthlyWindowStart,
				MonthlyUsed:        decimal.RequireFromString("100"),
				UpdatedAt:          asOf.Add(-time.Second),
				PeriodUsage:        periodUsage,
			},
		},
	}
}

func newConsistentQuotaOverviewService(snapshot *QuotaOverviewSnapshot) *QuotaOverviewService {
	service := NewQuotaOverviewService(
		&quotaOverviewRepoStub{snapshots: []*QuotaOverviewSnapshot{snapshot}},
		quotaOverviewCheckerStub{result: QuotaOverviewConsistency{Consistent: true}},
	)
	service.now = func() time.Time { return snapshot.AsOf.Add(time.Second) }
	return service
}

func TestQuotaOverviewMemberExhaustionNeverFallsBackToBalance(t *testing.T) {
	asOf := time.Date(2026, 7, 29, 7, 30, 0, 0, time.UTC)
	overview, err := newConsistentQuotaOverviewService(quotaOverviewTestSnapshot(asOf)).
		GetOverview(context.Background(), 42, "Asia/Shanghai")
	require.NoError(t, err)

	require.Equal(t, QuotaFreshnessFresh, overview.Freshness)
	require.Equal(t, "p***@example.com", overview.Account.DisplayLabel)
	require.Nil(t, overview.Account.CanMakeRequest)
	require.Equal(t, QuotaStatePartial, overview.Account.QuotaState)
	require.Equal(t, QuotaWalletAvailable, overview.Wallet.State)
	require.Equal(t, "128.6400000000", overview.Wallet.Available)
	require.Equal(t, "0.0000000000", overview.Wallet.Reserved)
	require.Equal(t, "1.1600000000", overview.Wallet.TodaySpend)
	require.Equal(t, "10.7400000000", overview.Wallet.MonthSpend)
	require.Contains(t, overview.Coverage.Included, "account_spend_today")
	require.Contains(t, overview.Coverage.Included, "account_spend_month_to_date")
	require.Len(t, overview.BillingGroups, 2)

	var balanceGroup, memberGroup *QuotaOverviewBillingGroup
	for i := range overview.BillingGroups {
		switch overview.BillingGroups[i].BillingMode {
		case QuotaBillingBalance:
			balanceGroup = &overview.BillingGroups[i]
		case QuotaBillingSubscription:
			memberGroup = &overview.BillingGroups[i]
		}
	}
	require.NotNil(t, balanceGroup)
	require.Equal(t, QuotaGroupUsable, balanceGroup.State)
	require.NotNil(t, memberGroup)
	require.Equal(t, QuotaGroupBlocked, memberGroup.State)
	require.Equal(t, "none", memberGroup.FallbackPolicy)
	require.Equal(t, "subscription_weekly_exhausted", *memberGroup.ReasonCode)
	require.Equal(t, "manage_api_keys", memberGroup.RecommendedAction)
	require.NotEqual(t, "recharge", memberGroup.RecommendedAction)
}

func TestQuotaOverviewEmptyBalanceDoesNotBlockUsableMembership(t *testing.T) {
	asOf := time.Date(2026, 7, 29, 7, 30, 0, 0, time.UTC)
	snapshot := quotaOverviewTestSnapshot(asOf)
	snapshot.Account.Balance = decimal.Zero
	snapshot.Subscriptions[0].WeeklyUsed = decimal.RequireFromString("12.5")

	overview, err := newConsistentQuotaOverviewService(snapshot).
		GetOverview(context.Background(), 42, "UTC")
	require.NoError(t, err)
	require.Equal(t, QuotaWalletExhausted, overview.Wallet.State)
	require.Equal(t, QuotaStatePartial, overview.Account.QuotaState)

	states := make(map[string]string)
	for i := range overview.BillingGroups {
		states[overview.BillingGroups[i].BillingMode] = overview.BillingGroups[i].State
	}
	require.Equal(t, QuotaGroupBlocked, states[QuotaBillingBalance])
	require.Equal(t, QuotaGroupUsable, states[QuotaBillingSubscription])
}

func TestQuotaOverviewMonthlyWindowUsesConfiguredLimitOrWeeklyFallback(t *testing.T) {
	asOf := time.Date(2026, 7, 29, 7, 30, 0, 0, time.UTC)
	configuredMonthly := decimal.RequireFromString("500")
	tests := []struct {
		name           string
		configured     *decimal.Decimal
		wantLimit      string
		wantRemaining  string
		wantPercentage float64
	}{
		{
			name:           "configured monthly limit",
			configured:     &configuredMonthly,
			wantLimit:      "500.0000000000",
			wantRemaining:  "375.0000000000",
			wantPercentage: 25,
		},
		{
			name:           "nil monthly limit falls back to weekly times four",
			wantLimit:      "800.0000000000",
			wantRemaining:  "675.0000000000",
			wantPercentage: 15.625,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			snapshot := quotaOverviewTestSnapshot(asOf)
			snapshot.Subscriptions[0].WeeklyUsed = decimal.RequireFromString("12.5")
			snapshot.Subscriptions[0].MonthlyLimit = test.configured
			snapshot.Subscriptions[0].MonthlyUsed = decimal.RequireFromString("125")

			overview, err := newConsistentQuotaOverviewService(snapshot).
				GetOverview(context.Background(), 42, "UTC")
			require.NoError(t, err)
			require.Contains(t, overview.Coverage.Included, "subscription_30d")
			require.Len(t, overview.Subscriptions, 1)

			window := overview.Subscriptions[0].MonthlyWindow
			require.Equal(t, "30d_from_subscription_start", window.Kind)
			require.Equal(t, QuotaWindowActive, window.State)
			require.Equal(t, snapshot.Subscriptions[0].StartsAt, window.AnchorAt)
			require.Equal(t, snapshot.Subscriptions[0].StartsAt, *window.PeriodStart)
			require.Equal(t, snapshot.Subscriptions[0].StartsAt.Add(SubscriptionMonthlyWindowDuration), *window.PeriodEnd)
			require.Equal(t, *window.PeriodEnd, *window.ResetsAt)
			require.Equal(t, test.wantLimit, *window.Limit)
			require.Equal(t, "125.0000000000", *window.Used)
			require.Equal(t, test.wantRemaining, *window.Remaining)
			require.Equal(t, test.wantPercentage, *window.UsedPercent)
		})
	}
}

func TestQuotaOverviewExplicitZeroMonthlyLimitBlocksWithoutFallback(t *testing.T) {
	asOf := time.Date(2026, 7, 29, 7, 30, 0, 0, time.UTC)
	snapshot := quotaOverviewTestSnapshot(asOf)
	zero := decimal.Zero
	snapshot.Subscriptions[0].WeeklyUsed = decimal.RequireFromString("12.5")
	snapshot.Subscriptions[0].MonthlyLimit = &zero
	snapshot.Subscriptions[0].MonthlyUsed = decimal.Zero

	overview, err := newConsistentQuotaOverviewService(snapshot).
		GetOverview(context.Background(), 42, "UTC")
	require.NoError(t, err)

	window := overview.Subscriptions[0].MonthlyWindow
	require.Equal(t, QuotaWindowExhausted, window.State)
	require.Equal(t, "0.0000000000", *window.Limit)
	require.Equal(t, "0.0000000000", *window.Used)
	require.Equal(t, "0.0000000000", *window.Remaining)
	require.Equal(t, float64(100), *window.UsedPercent)

	var memberGroup *QuotaOverviewBillingGroup
	for i := range overview.BillingGroups {
		if overview.BillingGroups[i].BillingMode == QuotaBillingSubscription {
			memberGroup = &overview.BillingGroups[i]
			break
		}
	}
	require.NotNil(t, memberGroup)
	require.Equal(t, QuotaGroupBlocked, memberGroup.State)
	require.Equal(t, "subscription_monthly_exhausted", *memberGroup.ReasonCode)
	require.Equal(t, "none", memberGroup.FallbackPolicy)
	require.NotNil(t, overview.Account.PrimaryIssue)
	require.Equal(t, *window.PeriodEnd, *overview.Account.PrimaryIssue.RecoversAt)
}

func TestQuotaOverviewExplicitZeroWeeklyLimitIsExhaustedNotUnknown(t *testing.T) {
	asOf := time.Date(2026, 7, 29, 7, 30, 0, 0, time.UTC)
	snapshot := quotaOverviewTestSnapshot(asOf)
	zero := decimal.Zero
	monthlyLimit := decimal.RequireFromString("500")
	snapshot.Subscriptions[0].WeeklyLimit = &zero
	snapshot.Subscriptions[0].WeeklyUsed = decimal.Zero
	snapshot.Subscriptions[0].MonthlyLimit = &monthlyLimit
	snapshot.Subscriptions[0].MonthlyUsed = decimal.Zero

	overview, err := newConsistentQuotaOverviewService(snapshot).
		GetOverview(context.Background(), 42, "UTC")
	require.NoError(t, err)
	require.Equal(t, QuotaFreshnessFresh, overview.Freshness)

	window := overview.Subscriptions[0].WeeklyWindow
	require.Equal(t, QuotaWindowExhausted, window.State)
	require.Equal(t, "0.0000000000", *window.Limit)
	require.Equal(t, "0.0000000000", *window.Remaining)
	require.Equal(t, float64(100), *window.UsedPercent)
	require.NotNil(t, overview.Account.PrimaryIssue)
	require.Equal(t, "subscription_weekly_exhausted", overview.Account.PrimaryIssue.ReasonCode)
}

func TestQuotaOverviewCombinedExhaustionRecoversOnlyAfterBothWindowsReset(t *testing.T) {
	asOf := time.Date(2026, 7, 29, 7, 30, 0, 0, time.UTC)
	snapshot := quotaOverviewTestSnapshot(asOf)
	snapshot.Subscriptions[0].WeeklyUsed = decimal.RequireFromString("200")
	snapshot.Subscriptions[0].MonthlyUsed = decimal.RequireFromString("800")

	overview, err := newConsistentQuotaOverviewService(snapshot).
		GetOverview(context.Background(), 42, "UTC")
	require.NoError(t, err)

	subscription := overview.Subscriptions[0]
	require.Equal(t, QuotaWindowExhausted, subscription.WeeklyWindow.State)
	require.Equal(t, QuotaWindowExhausted, subscription.MonthlyWindow.State)
	require.NotNil(t, overview.Account.PrimaryIssue)
	require.Equal(t, "subscription_monthly_exhausted", overview.Account.PrimaryIssue.ReasonCode)
	require.Equal(t, *subscription.MonthlyWindow.PeriodEnd, *overview.Account.PrimaryIssue.RecoversAt)
	require.True(t, overview.Account.PrimaryIssue.RecoversAt.After(*subscription.WeeklyWindow.PeriodEnd))
}

func TestQuotaOverviewOrdersNewestCurrentMembershipFirstAndKeepsHistory(t *testing.T) {
	asOf := time.Date(2026, 7, 29, 7, 30, 0, 0, time.UTC)
	snapshot := quotaOverviewTestSnapshot(asOf)
	limit := decimal.RequireFromString("200")

	currentSubscription := func(
		id, groupID int64,
		name string,
		startsAt, expiresAt time.Time,
	) QuotaOverviewSubscriptionSnapshot {
		windowStart, _, ok := AnchoredWeeklyWindow(startsAt, asOf)
		require.True(t, ok)
		monthlyWindowStart, _, ok := AnchoredMonthlyWindow(startsAt, asOf)
		require.True(t, ok)
		return QuotaOverviewSubscriptionSnapshot{
			ID:                 id,
			GroupID:            groupID,
			GroupName:          name,
			GroupStatus:        StatusActive,
			Status:             SubscriptionStatusActive,
			StartsAt:           startsAt,
			ExpiresAt:          expiresAt,
			WeeklyWindowStart:  &windowStart,
			WeeklyLimit:        &limit,
			WeeklyUsed:         decimal.Zero,
			MonthlyWindowStart: &monthlyWindowStart,
			MonthlyUsed:        decimal.Zero,
			UpdatedAt:          asOf.Add(-time.Second),
		}
	}

	oldHistorical := QuotaOverviewSubscriptionSnapshot{
		ID:          100,
		GroupID:     10,
		GroupName:   "很久以前的会员",
		GroupStatus: StatusActive,
		Status:      SubscriptionStatusExpired,
		StartsAt:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		ExpiresAt:   time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   asOf.Add(-time.Hour),
	}
	olderCurrent := currentSubscription(
		200,
		20,
		"较早的当前会员",
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
	)
	newestCurrent := currentSubscription(
		300,
		30,
		"最新会员",
		time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC),
	)
	newerRevokedHistory := QuotaOverviewSubscriptionSnapshot{
		ID:          400,
		GroupID:     40,
		GroupName:   "最近撤销的会员",
		GroupStatus: StatusActive,
		Status:      SubscriptionStatusActive,
		Revoked:     true,
		StartsAt:    time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC),
		ExpiresAt:   time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   asOf,
	}

	// Mirrors the old repository order by ascending group_id.
	snapshot.Subscriptions = []QuotaOverviewSubscriptionSnapshot{
		oldHistorical,
		olderCurrent,
		newestCurrent,
		newerRevokedHistory,
	}

	overview, err := newConsistentQuotaOverviewService(snapshot).
		GetOverview(context.Background(), 42, "UTC")
	require.NoError(t, err)
	require.Equal(t, []string{"300", "200", "400", "100"}, []string{
		overview.Subscriptions[0].ID,
		overview.Subscriptions[1].ID,
		overview.Subscriptions[2].ID,
		overview.Subscriptions[3].ID,
	})
	require.Equal(t, "最新会员", overview.Subscriptions[0].Name)
	require.Equal(t, SubscriptionStatusActive, overview.Subscriptions[0].Status)
	require.Equal(t, SubscriptionStatusRevoked, overview.Subscriptions[2].Status)
	require.Equal(t, SubscriptionStatusExpired, overview.Subscriptions[3].Status)
}

func TestOrderQuotaOverviewSubscriptionsUsesTermTieBreakersWithoutMutatingInput(t *testing.T) {
	asOf := time.Date(2026, 7, 29, 7, 30, 0, 0, time.UTC)
	startsAt := asOf.Add(-24 * time.Hour)
	expiresAt := asOf.Add(24 * time.Hour)
	input := []QuotaOverviewSubscriptionSnapshot{
		{ID: 1, Status: SubscriptionStatusActive, StartsAt: startsAt, ExpiresAt: expiresAt},
		{ID: 3, Status: SubscriptionStatusActive, StartsAt: startsAt, ExpiresAt: expiresAt.Add(time.Hour)},
		{ID: 2, Status: SubscriptionStatusActive, StartsAt: startsAt, ExpiresAt: expiresAt.Add(time.Hour)},
	}

	ordered := orderQuotaOverviewSubscriptions(input, asOf)

	require.Equal(t, []int64{3, 2, 1}, []int64{ordered[0].ID, ordered[1].ID, ordered[2].ID})
	require.Equal(t, []int64{1, 3, 2}, []int64{input[0].ID, input[1].ID, input[2].ID})
}

func TestQuotaOverviewNoEnabledKeysAndKeyOwnLimit(t *testing.T) {
	asOf := time.Date(2026, 7, 29, 7, 30, 0, 0, time.UTC)
	noKeys := quotaOverviewTestSnapshot(asOf)
	noKeys.Keys = nil
	overview, err := newConsistentQuotaOverviewService(noKeys).
		GetOverview(context.Background(), 42, "UTC")
	require.NoError(t, err)
	require.Equal(t, QuotaStateNoKeys, overview.Account.QuotaState)
	require.Len(t, overview.BillingGroups, 1)
	require.Equal(t, QuotaGroupNoKeys, overview.BillingGroups[0].State)

	exhaustedKey := quotaOverviewTestSnapshot(asOf)
	exhaustedKey.Keys = exhaustedKey.Keys[:1]
	exhaustedKey.Keys[0].Quota = decimal.RequireFromString("10")
	exhaustedKey.Keys[0].QuotaUsed = decimal.RequireFromString("10")
	exhaustedKey.Subscriptions = nil
	overview, err = newConsistentQuotaOverviewService(exhaustedKey).
		GetOverview(context.Background(), 42, "UTC")
	require.NoError(t, err)
	require.Equal(t, QuotaStateAllBlocked, overview.Account.QuotaState)
	require.Equal(t, QuotaGroupBlocked, overview.BillingGroups[0].State)
	require.Equal(t, "api_key_limit_exhausted", *overview.BillingGroups[0].ReasonCode)
}

func TestQuotaOverviewAnchoredBoundaryAndFreshUntil(t *testing.T) {
	anchor := time.Date(2026, 7, 25, 9, 30, 0, 0, time.UTC)
	boundary := anchor.Add(SubscriptionWeeklyWindowDuration)

	before := quotaOverviewTestSnapshot(boundary.Add(-time.Millisecond))
	overview, err := newConsistentQuotaOverviewService(before).
		GetOverview(context.Background(), 42, "UTC")
	require.NoError(t, err)
	require.Equal(t, boundary, overview.FreshUntil)
	require.Equal(t, anchor, *overview.Subscriptions[0].WeeklyWindow.PeriodStart)
	require.Equal(t, boundary, *overview.Subscriptions[0].WeeklyWindow.PeriodEnd)

	atBoundary := quotaOverviewTestSnapshot(boundary)
	atBoundary.Subscriptions[0].WeeklyUsed = decimal.Zero
	overview, err = newConsistentQuotaOverviewService(atBoundary).
		GetOverview(context.Background(), 42, "UTC")
	require.NoError(t, err)
	require.Equal(t, boundary, *overview.Subscriptions[0].WeeklyWindow.PeriodStart)
	require.Equal(t, boundary.Add(SubscriptionWeeklyWindowDuration), *overview.Subscriptions[0].WeeklyWindow.PeriodEnd)
	require.Equal(t, "0.0000000000", *overview.Subscriptions[0].WeeklyWindow.Used)
	require.Equal(t, QuotaWindowActive, overview.Subscriptions[0].WeeklyWindow.State)
	require.Equal(t, QuotaPeriodUsageAvailable, overview.Subscriptions[0].PeriodUsage.State)
	require.Equal(t, int64(0), *overview.Subscriptions[0].PeriodUsage.TotalRequests)
	require.Equal(t, int64(0), *overview.Subscriptions[0].PeriodUsage.TotalTokens)
	for i := range overview.Subscriptions[0].PeriodUsage.Points {
		require.Equal(
			t,
			QuotaPeriodUsagePointFuture,
			overview.Subscriptions[0].PeriodUsage.Points[i].State,
		)
		require.Nil(t, overview.Subscriptions[0].PeriodUsage.Points[i].Requests)
		require.Nil(t, overview.Subscriptions[0].PeriodUsage.Points[i].TotalTokens)
	}

	keyExpirySnapshot := quotaOverviewTestSnapshot(boundary.Add(-2 * time.Minute))
	keyExpiry := keyExpirySnapshot.AsOf.Add(30 * time.Second)
	keyExpirySnapshot.Keys[0].ExpiresAt = &keyExpiry
	overview, err = newConsistentQuotaOverviewService(keyExpirySnapshot).
		GetOverview(context.Background(), 42, "UTC")
	require.NoError(t, err)
	require.Equal(t, keyExpiry, overview.FreshUntil)
}

func TestQuotaOverviewProjectsEmptyNewAnchoredPeriodLikeAdmission(t *testing.T) {
	anchor := time.Date(2026, 7, 25, 9, 30, 0, 0, time.UTC)
	asOf := anchor.Add(SubscriptionWeeklyWindowDuration)
	snapshot := quotaOverviewTestSnapshot(asOf)
	snapshot.Subscriptions[0].WeeklyWindowStart = timePointer(anchor)
	snapshot.Subscriptions[0].WeeklyUsed = decimal.RequireFromString("200")
	for i := range snapshot.Subscriptions[0].PeriodUsage.Buckets {
		snapshot.Subscriptions[0].PeriodUsage.Buckets[i] = QuotaOverviewPeriodUsageBucketSnapshot{}
	}

	service := NewQuotaOverviewService(
		&quotaOverviewRepoStub{snapshots: []*QuotaOverviewSnapshot{snapshot}},
		NewBillingQuotaOverviewConsistencyChecker(&quotaOverviewCacheStub{
			balance: 128.64,
		}),
	)
	service.now = func() time.Time { return asOf }
	overview, err := service.GetOverview(context.Background(), 42, "UTC")
	require.NoError(t, err)

	require.Equal(t, QuotaFreshnessFresh, overview.Freshness)
	require.Equal(t, QuotaWindowActive, overview.Subscriptions[0].WeeklyWindow.State)
	require.Equal(t, asOf, *overview.Subscriptions[0].WeeklyWindow.PeriodStart)
	require.Equal(t, "0.0000000000", *overview.Subscriptions[0].WeeklyWindow.Used)
	require.Equal(t, "200.0000000000", *overview.Subscriptions[0].WeeklyWindow.Remaining)
	require.Equal(t, float64(0), *overview.Subscriptions[0].WeeklyWindow.UsedPercent)
	require.Equal(t, QuotaPeriodUsageAvailable, overview.Subscriptions[0].PeriodUsage.State)
	require.Equal(t, int64(0), *overview.Subscriptions[0].PeriodUsage.TotalRequests)
	require.Equal(t, int64(0), *overview.Subscriptions[0].PeriodUsage.TotalTokens)
	require.NotNil(t, snapshot.Subscriptions[0].WeeklyWindowProjectedFrom)
	require.Equal(t, anchor, *snapshot.Subscriptions[0].WeeklyWindowProjectedFrom)
}

func TestQuotaOverviewDoesNotProjectElapsedMonthlyWindowWithoutAuthoritativeProof(t *testing.T) {
	anchor := time.Date(2026, 7, 25, 9, 30, 0, 0, time.UTC)
	asOf := anchor.Add(SubscriptionMonthlyWindowDuration)
	snapshot := quotaOverviewTestSnapshot(asOf)
	snapshot.Subscriptions[0].MonthlyWindowStart = timePointer(anchor)
	snapshot.Subscriptions[0].MonthlyUsed = decimal.RequireFromString("800")

	service := NewQuotaOverviewService(
		&quotaOverviewRepoStub{snapshots: []*QuotaOverviewSnapshot{snapshot}},
		NewBillingQuotaOverviewConsistencyChecker(&quotaOverviewCacheStub{
			balance: 128.64,
		}),
	)
	service.now = func() time.Time { return asOf }
	overview, err := service.GetOverview(context.Background(), 42, "UTC")
	require.NoError(t, err)

	window := overview.Subscriptions[0].MonthlyWindow
	require.Equal(t, QuotaFreshnessUnknown, overview.Freshness)
	require.Equal(t, QuotaWindowUnknown, window.State)
	require.Equal(t, asOf, *window.PeriodStart)
	require.Nil(t, window.Limit)
	require.Nil(t, window.Used)
	require.Nil(t, window.Remaining)
	require.Nil(t, window.UsedPercent)
	require.Equal(t, anchor, *snapshot.Subscriptions[0].MonthlyWindowStart)
	require.Equal(t, "800", snapshot.Subscriptions[0].MonthlyUsed.String())
	require.Nil(t, snapshot.Subscriptions[0].MonthlyWindowProjectedFrom)
}

func TestQuotaOverviewInvalidMonthlyWindowMarkerIsUnknown(t *testing.T) {
	anchor := time.Date(2026, 7, 25, 9, 30, 0, 0, time.UTC)
	asOf := anchor.Add(SubscriptionMonthlyWindowDuration + time.Minute)
	snapshot := quotaOverviewTestSnapshot(asOf)
	offAnchor := anchor.Add(time.Second)
	snapshot.Subscriptions[0].MonthlyWindowStart = &offAnchor

	overview, err := newConsistentQuotaOverviewService(snapshot).
		GetOverview(context.Background(), 42, "UTC")
	require.NoError(t, err)
	require.Equal(t, QuotaFreshnessUnknown, overview.Freshness)
	require.Equal(t, QuotaWindowUnknown, overview.Subscriptions[0].MonthlyWindow.State)
	require.Nil(t, snapshot.Subscriptions[0].MonthlyWindowProjectedFrom)
}

func TestQuotaOverviewDoesNotProjectStaleWindowWithCurrentUsageOrInvalidMarker(t *testing.T) {
	anchor := time.Date(2026, 7, 25, 9, 30, 0, 0, time.UTC)
	asOf := anchor.Add(SubscriptionWeeklyWindowDuration + time.Minute)

	tests := []struct {
		name   string
		mutate func(*QuotaOverviewSubscriptionSnapshot)
	}{
		{
			name: "current period already has successful usage",
			mutate: func(sub *QuotaOverviewSubscriptionSnapshot) {
				sub.PeriodUsage.Buckets[0].Requests = 1
				sub.PeriodUsage.Buckets[0].OutputTokens = 10
			},
		},
		{
			name: "persisted marker is off anchor",
			mutate: func(sub *QuotaOverviewSubscriptionSnapshot) {
				offAnchor := anchor.Add(time.Second)
				sub.WeeklyWindowStart = &offAnchor
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			snapshot := quotaOverviewTestSnapshot(asOf)
			snapshot.Subscriptions[0].WeeklyWindowStart = timePointer(anchor)
			snapshot.Subscriptions[0].WeeklyUsed = decimal.RequireFromString("200")
			for i := range snapshot.Subscriptions[0].PeriodUsage.Buckets {
				snapshot.Subscriptions[0].PeriodUsage.Buckets[i] = QuotaOverviewPeriodUsageBucketSnapshot{}
			}
			test.mutate(&snapshot.Subscriptions[0])

			overview, err := newConsistentQuotaOverviewService(snapshot).
				GetOverview(context.Background(), 42, "UTC")
			require.NoError(t, err)
			require.Equal(t, QuotaFreshnessUnknown, overview.Freshness)
			require.Equal(t, QuotaWindowUnknown, overview.Subscriptions[0].WeeklyWindow.State)
			require.Nil(t, snapshot.Subscriptions[0].WeeklyWindowProjectedFrom)
		})
	}
}

func TestQuotaOverviewProjectedWindowRequiresSubscriptionCacheMiss(t *testing.T) {
	anchor := time.Date(2026, 7, 25, 9, 30, 0, 0, time.UTC)
	asOf := anchor.Add(SubscriptionWeeklyWindowDuration)
	snapshot := quotaOverviewTestSnapshot(asOf)
	snapshot.Subscriptions[0].WeeklyWindowStart = timePointer(anchor)
	snapshot.Subscriptions[0].WeeklyUsed = decimal.RequireFromString("200")
	for i := range snapshot.Subscriptions[0].PeriodUsage.Buckets {
		snapshot.Subscriptions[0].PeriodUsage.Buckets[i] = QuotaOverviewPeriodUsageBucketSnapshot{}
	}
	normalizeQuotaOverviewElapsedEmptyWindows(snapshot)

	sub := snapshot.Subscriptions[0]
	_, periodEnd, ok := AnchoredWeeklyWindow(sub.StartsAt, asOf)
	require.True(t, ok)
	_, monthlyPeriodEnd, ok := AnchoredMonthlyWindow(sub.StartsAt, asOf)
	require.True(t, ok)
	checker := NewBillingQuotaOverviewConsistencyChecker(&quotaOverviewCacheStub{
		balance: 128.64,
		subs: map[int64]*SubscriptionCacheData{
			sub.GroupID: {
				SubscriptionID:     sub.ID,
				Status:             sub.Status,
				StartsAt:           sub.StartsAt,
				ExpiresAt:          sub.ExpiresAt,
				WeeklyWindowStart:  sub.WeeklyWindowStart,
				WeeklyWindowEnd:    periodEnd,
				MonthlyWindowStart: sub.MonthlyWindowStart,
				MonthlyWindowEnd:   monthlyPeriodEnd,
				WeeklyUsage:        0,
				MonthlyUsage:       100,
				Version:            sub.UpdatedAt.UnixMicro(),
			},
		},
	})
	result, err := checker.CheckQuotaOverviewConsistency(context.Background(), snapshot)
	require.NoError(t, err)
	require.False(t, result.Consistent)
	require.Contains(t, result.WarningCodes, "quota_subscription_cache_mismatch")
}

func TestQuotaOverviewPeriodUsageUsesAnchoredBucketsAndNullFutureValues(t *testing.T) {
	anchor := time.Date(2026, 7, 25, 9, 30, 0, 0, time.UTC)
	asOf := anchor.Add(3*24*time.Hour + 22*time.Hour)
	snapshot := quotaOverviewTestSnapshot(asOf)

	overview, err := newConsistentQuotaOverviewService(snapshot).
		GetOverview(context.Background(), 42, "Asia/Shanghai")
	require.NoError(t, err)
	require.Contains(t, overview.Coverage.Included, "subscription_period_usage")
	require.Len(t, overview.Subscriptions, 1)

	usage := overview.Subscriptions[0].PeriodUsage
	require.Equal(t, QuotaPeriodUsageAvailable, usage.State)
	require.Equal(t, QuotaPeriodUsageBucketAnchored24h, usage.BucketKind)
	require.Equal(t, asOf, *usage.ObservedUntil)
	require.Equal(t, int64(279), *usage.TotalRequests)
	require.Equal(t, int64(901000), *usage.TotalTokens)
	require.Len(t, usage.Points, 7)
	require.Equal(t, []string{
		QuotaPeriodUsagePointComplete,
		QuotaPeriodUsagePointComplete,
		QuotaPeriodUsagePointComplete,
		QuotaPeriodUsagePointPartial,
		QuotaPeriodUsagePointFuture,
		QuotaPeriodUsagePointFuture,
		QuotaPeriodUsagePointFuture,
	}, []string{
		usage.Points[0].State,
		usage.Points[1].State,
		usage.Points[2].State,
		usage.Points[3].State,
		usage.Points[4].State,
		usage.Points[5].State,
		usage.Points[6].State,
	})
	require.Equal(t, 1, usage.Points[0].Index)
	require.Equal(t, anchor, usage.Points[0].StartAt)
	require.Equal(t, anchor.Add(24*time.Hour), usage.Points[0].EndAt)
	require.Equal(t, int64(182000), *usage.Points[0].TotalTokens)
	require.Equal(t, int64(259000), *usage.Points[3].TotalTokens)
	require.Nil(t, usage.Points[4].Requests)
	require.Nil(t, usage.Points[4].CacheHitTokens)
	require.Nil(t, usage.Points[4].CacheMissTokens)
	require.Nil(t, usage.Points[4].OutputTokens)
	require.Nil(t, usage.Points[4].TotalTokens)
}

func TestQuotaOverviewPeriodUsageAtPeriodStartIsAvailableZeroWithAllFuturePoints(t *testing.T) {
	anchor := time.Date(2026, 7, 25, 9, 30, 0, 0, time.UTC)
	snapshot := quotaOverviewTestSnapshot(anchor)
	for i := range snapshot.Subscriptions[0].PeriodUsage.Buckets {
		snapshot.Subscriptions[0].PeriodUsage.Buckets[i] = QuotaOverviewPeriodUsageBucketSnapshot{}
	}

	overview, err := newConsistentQuotaOverviewService(snapshot).
		GetOverview(context.Background(), 42, "UTC")
	require.NoError(t, err)
	usage := overview.Subscriptions[0].PeriodUsage
	require.Equal(t, QuotaPeriodUsageAvailable, usage.State)
	require.Equal(t, int64(0), *usage.TotalRequests)
	require.Equal(t, int64(0), *usage.TotalTokens)
	require.Len(t, usage.Points, 7)
	for i := range usage.Points {
		require.Equal(t, i+1, usage.Points[i].Index)
		require.Equal(t, QuotaPeriodUsagePointFuture, usage.Points[i].State)
		require.Nil(t, usage.Points[i].Requests)
		require.Nil(t, usage.Points[i].TotalTokens)
	}
}

func TestQuotaOverviewPeriodUsageSuccessfulEmptyElapsedBucketsAreRealZero(t *testing.T) {
	anchor := time.Date(2026, 7, 25, 9, 30, 0, 0, time.UTC)
	asOf := anchor.Add(36 * time.Hour)
	snapshot := quotaOverviewTestSnapshot(asOf)
	for i := range snapshot.Subscriptions[0].PeriodUsage.Buckets {
		snapshot.Subscriptions[0].PeriodUsage.Buckets[i] = QuotaOverviewPeriodUsageBucketSnapshot{}
	}

	overview, err := newConsistentQuotaOverviewService(snapshot).
		GetOverview(context.Background(), 42, "UTC")
	require.NoError(t, err)
	usage := overview.Subscriptions[0].PeriodUsage
	require.Equal(t, QuotaPeriodUsageAvailable, usage.State)
	require.Equal(t, int64(0), *usage.TotalRequests)
	require.Equal(t, int64(0), *usage.TotalTokens)
	require.Equal(t, QuotaPeriodUsagePointComplete, usage.Points[0].State)
	require.Equal(t, int64(0), *usage.Points[0].Requests)
	require.Equal(t, int64(0), *usage.Points[0].CacheHitTokens)
	require.Equal(t, int64(0), *usage.Points[0].CacheMissTokens)
	require.Equal(t, int64(0), *usage.Points[0].OutputTokens)
	require.Equal(t, int64(0), *usage.Points[0].TotalTokens)
	require.Equal(t, QuotaPeriodUsagePointPartial, usage.Points[1].State)
	require.Equal(t, int64(0), *usage.Points[1].Requests)
	require.Equal(t, int64(0), *usage.Points[1].TotalTokens)
	require.Equal(t, QuotaPeriodUsagePointFuture, usage.Points[2].State)
	require.Nil(t, usage.Points[2].Requests)
	require.Nil(t, usage.Points[2].TotalTokens)
}

func TestQuotaOverviewPeriodUsageKeepsContinuous24HourBucketsAcrossCalendarsAndDST(t *testing.T) {
	tests := []struct {
		name   string
		anchor time.Time
	}{
		{
			name:   "crosses_month_and_year",
			anchor: time.Date(2026, 12, 29, 23, 45, 0, 0, time.UTC),
		},
		{
			name:   "crosses_new_york_dst",
			anchor: time.Date(2026, 3, 7, 17, 30, 0, 0, time.UTC),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			periodEnd := test.anchor.Add(SubscriptionWeeklyWindowDuration)
			usage := &QuotaOverviewPeriodUsageSnapshot{
				PeriodStart:   test.anchor,
				PeriodEnd:     periodEnd,
				ObservedUntil: test.anchor.Add(49 * time.Hour),
			}
			got := buildQuotaOverviewPeriodUsage(
				usage,
				test.anchor,
				periodEnd,
				usage.ObservedUntil,
				periodEnd.Add(24*time.Hour),
			)
			require.Equal(t, QuotaPeriodUsageAvailable, got.State)
			require.Len(t, got.Points, 7)
			for i := 1; i < len(got.Points); i++ {
				require.Equal(t, 24*time.Hour, got.Points[i].StartAt.Sub(got.Points[i-1].StartAt))
			}
			require.Equal(t, QuotaPeriodUsagePointComplete, got.Points[0].State)
			require.Equal(t, QuotaPeriodUsagePointComplete, got.Points[1].State)
			require.Equal(t, QuotaPeriodUsagePointPartial, got.Points[2].State)
			require.Equal(t, QuotaPeriodUsagePointFuture, got.Points[3].State)
		})
	}
}

func TestQuotaOverviewPeriodUsageFailureDoesNotDegradeQuotaSnapshot(t *testing.T) {
	asOf := time.Date(2026, 7, 29, 7, 30, 0, 0, time.UTC)
	snapshot := quotaOverviewTestSnapshot(asOf)
	snapshot.PeriodUsageAggregationError = true
	snapshot.Subscriptions[0].PeriodUsage = nil
	snapshot.Subscriptions[0].WeeklyUsed = decimal.RequireFromString("12.5")

	overview, err := newConsistentQuotaOverviewService(snapshot).
		GetOverview(context.Background(), 42, "UTC")
	require.NoError(t, err)
	require.Equal(t, QuotaFreshnessFresh, overview.Freshness)
	require.Equal(t, QuotaWalletAvailable, overview.Wallet.State)
	require.Equal(t, QuotaWindowActive, overview.Subscriptions[0].WeeklyWindow.State)
	require.Equal(t, "12.5000000000", *overview.Subscriptions[0].WeeklyWindow.Used)
	require.Equal(t, QuotaPeriodUsageUnknown, overview.Subscriptions[0].PeriodUsage.State)
	require.Nil(t, overview.Subscriptions[0].PeriodUsage.ObservedUntil)
	require.Nil(t, overview.Subscriptions[0].PeriodUsage.TotalRequests)
	require.Nil(t, overview.Subscriptions[0].PeriodUsage.TotalTokens)
	require.Nil(t, overview.Subscriptions[0].PeriodUsage.Points)
	require.Contains(t, overview.Warnings, "subscription_period_usage_unavailable")
}

func TestQuotaOverviewPeriodUsageMismatchedBoundsAreUnknownWithoutDegradingQuota(t *testing.T) {
	asOf := time.Date(2026, 7, 29, 7, 30, 0, 0, time.UTC)
	snapshot := quotaOverviewTestSnapshot(asOf)
	snapshot.Subscriptions[0].WeeklyUsed = decimal.RequireFromString("12.5")
	snapshot.Subscriptions[0].PeriodUsage.ObservedUntil = asOf.Add(time.Second)

	overview, err := newConsistentQuotaOverviewService(snapshot).
		GetOverview(context.Background(), 42, "UTC")
	require.NoError(t, err)
	require.Equal(t, QuotaFreshnessFresh, overview.Freshness)
	require.Equal(t, QuotaWalletAvailable, overview.Wallet.State)
	require.Equal(t, QuotaWindowActive, overview.Subscriptions[0].WeeklyWindow.State)
	require.Equal(t, QuotaPeriodUsageUnknown, overview.Subscriptions[0].PeriodUsage.State)
	require.Nil(t, overview.Subscriptions[0].PeriodUsage.TotalRequests)
	require.Nil(t, overview.Subscriptions[0].PeriodUsage.TotalTokens)
	require.Nil(t, overview.Subscriptions[0].PeriodUsage.Points)
	require.Contains(t, overview.Warnings, "subscription_period_usage_unavailable")
}

func TestCloneQuotaOverviewDeepCopiesPeriodUsagePointers(t *testing.T) {
	asOf := time.Date(2026, 7, 29, 7, 30, 0, 0, time.UTC)
	overview, err := newConsistentQuotaOverviewService(quotaOverviewTestSnapshot(asOf)).
		GetOverview(context.Background(), 42, "UTC")
	require.NoError(t, err)
	require.Equal(t, QuotaPeriodUsageAvailable, overview.Subscriptions[0].PeriodUsage.State)
	canMakeRequest := false
	overview.Account.CanMakeRequest = &canMakeRequest

	clone := cloneQuotaOverview(*overview)
	*clone.Subscriptions[0].PeriodUsage.TotalRequests = 999
	*clone.Subscriptions[0].PeriodUsage.TotalTokens = 999
	*clone.Subscriptions[0].PeriodUsage.Points[0].Requests = 999
	*clone.Subscriptions[0].PeriodUsage.Points[0].TotalTokens = 999
	*clone.Subscriptions[0].PeriodUsage.ObservedUntil = asOf.Add(time.Hour)
	*clone.Subscriptions[0].MonthlyWindow.Used = "999.0000000000"
	*clone.Subscriptions[0].MonthlyWindow.UsedPercent = 99
	*clone.Subscriptions[0].MonthlyWindow.PeriodEnd = asOf
	*clone.Account.CanMakeRequest = true
	require.NotNil(t, clone.Account.PrimaryIssue)
	require.NotNil(t, clone.Account.PrimaryIssue.RecoversAt)
	originalRecovery := *overview.Account.PrimaryIssue.RecoversAt
	*clone.Account.PrimaryIssue.RecoversAt = asOf.Add(48 * time.Hour)
	memberGroupIndex := -1
	for i := range clone.BillingGroups {
		if clone.BillingGroups[i].ResourceRef.Kind == "subscription" {
			memberGroupIndex = i
			break
		}
	}
	require.NotEqual(t, -1, memberGroupIndex)
	require.NotNil(t, clone.BillingGroups[memberGroupIndex].ReasonCode)
	require.NotNil(t, clone.BillingGroups[memberGroupIndex].ResourceRef.ID)
	originalReason := *overview.BillingGroups[memberGroupIndex].ReasonCode
	originalResourceID := *overview.BillingGroups[memberGroupIndex].ResourceRef.ID
	*clone.BillingGroups[memberGroupIndex].ReasonCode = "mutated"
	*clone.BillingGroups[memberGroupIndex].ResourceRef.ID = "mutated"

	require.Equal(t, int64(279), *overview.Subscriptions[0].PeriodUsage.TotalRequests)
	require.Equal(t, int64(901000), *overview.Subscriptions[0].PeriodUsage.TotalTokens)
	require.Equal(t, int64(58), *overview.Subscriptions[0].PeriodUsage.Points[0].Requests)
	require.Equal(t, int64(182000), *overview.Subscriptions[0].PeriodUsage.Points[0].TotalTokens)
	require.Equal(t, asOf, *overview.Subscriptions[0].PeriodUsage.ObservedUntil)
	require.Equal(t, "100.0000000000", *overview.Subscriptions[0].MonthlyWindow.Used)
	require.Equal(t, float64(12.5), *overview.Subscriptions[0].MonthlyWindow.UsedPercent)
	require.NotEqual(t, asOf, *overview.Subscriptions[0].MonthlyWindow.PeriodEnd)
	require.False(t, *overview.Account.CanMakeRequest)
	require.Equal(t, originalRecovery, *overview.Account.PrimaryIssue.RecoversAt)
	require.Equal(t, originalReason, *overview.BillingGroups[memberGroupIndex].ReasonCode)
	require.Equal(t, originalResourceID, *overview.BillingGroups[memberGroupIndex].ResourceRef.ID)
}

func TestQuotaOverviewFreshUntilStopsAtRequestedTimezoneMidnight(t *testing.T) {
	asOf := time.Date(2026, 7, 29, 15, 59, 30, 0, time.UTC)
	overview, err := newConsistentQuotaOverviewService(quotaOverviewTestSnapshot(asOf)).
		GetOverview(context.Background(), 42, "Asia/Shanghai")
	require.NoError(t, err)
	require.Equal(t, time.Date(2026, 7, 29, 16, 0, 0, 0, time.UTC), overview.FreshUntil)
}

func TestQuotaOverviewFreshUntilStopsAtMonthlyReset(t *testing.T) {
	anchor := time.Date(2026, 7, 25, 9, 30, 0, 0, time.UTC)
	asOf := anchor.Add(SubscriptionMonthlyWindowDuration - 2*time.Minute)
	snapshot := quotaOverviewTestSnapshot(asOf)

	overview, err := newConsistentQuotaOverviewService(snapshot).
		GetOverview(context.Background(), 42, "UTC")
	require.NoError(t, err)
	require.Equal(t, anchor.Add(SubscriptionMonthlyWindowDuration), overview.FreshUntil)
}

func TestQuotaOverviewExpiryPrecedesResetAndUsedAboveLimitIsPreserved(t *testing.T) {
	asOf := time.Date(2026, 7, 29, 7, 30, 0, 0, time.UTC)
	snapshot := quotaOverviewTestSnapshot(asOf)
	expiry := asOf.Add(2 * time.Minute)
	snapshot.Subscriptions[0].ExpiresAt = expiry
	snapshot.Subscriptions[0].WeeklyUsed = decimal.RequireFromString("234.5678901234")

	overview, err := newConsistentQuotaOverviewService(snapshot).
		GetOverview(context.Background(), 42, "UTC")
	require.NoError(t, err)
	require.Equal(t, expiry, overview.FreshUntil)
	require.Equal(t, "expiry", overview.Subscriptions[0].NextEvent.Kind)
	require.Equal(t, expiry, overview.Subscriptions[0].NextEvent.At)
	require.Equal(t, "234.5678901234", *overview.Subscriptions[0].WeeklyWindow.Used)
	require.Equal(t, "0.0000000000", *overview.Subscriptions[0].WeeklyWindow.Remaining)
	require.Equal(t, float64(100), *overview.Subscriptions[0].WeeklyWindow.UsedPercent)
	require.NotNil(t, overview.Account.PrimaryIssue)
	require.Nil(t, overview.Account.PrimaryIssue.RecoversAt)
}

func TestQuotaOverviewInvalidNegativeUsageAndConsistencyMismatchAreUnknown(t *testing.T) {
	asOf := time.Date(2026, 7, 29, 7, 30, 0, 0, time.UTC)
	negative := quotaOverviewTestSnapshot(asOf)
	negative.Subscriptions[0].WeeklyUsed = decimal.RequireFromString("-0.1")
	overview, err := newConsistentQuotaOverviewService(negative).
		GetOverview(context.Background(), 42, "UTC")
	require.NoError(t, err)
	require.Equal(t, QuotaFreshnessUnknown, overview.Freshness)
	require.Equal(t, QuotaStateUnknown, overview.Account.QuotaState)
	require.Contains(t, overview.Warnings, "invalid_quota_snapshot")

	negativeBalance := quotaOverviewTestSnapshot(asOf)
	negativeBalance.Account.Balance = decimal.RequireFromString("-0.0000000001")
	overview, err = newConsistentQuotaOverviewService(negativeBalance).
		GetOverview(context.Background(), 42, "UTC")
	require.NoError(t, err)
	require.Equal(t, QuotaFreshnessUnknown, overview.Freshness)
	require.Equal(t, QuotaStateUnknown, overview.Account.QuotaState)

	negativeSpend := quotaOverviewTestSnapshot(asOf)
	negativeSpend.TodaySpend = decimal.RequireFromString("-0.1")
	overview, err = newConsistentQuotaOverviewService(negativeSpend).
		GetOverview(context.Background(), 42, "UTC")
	require.NoError(t, err)
	require.Equal(t, QuotaFreshnessUnknown, overview.Freshness)
	require.Contains(t, overview.Warnings, "invalid_quota_snapshot")

	orphanedGroup := quotaOverviewTestSnapshot(asOf)
	orphanedGroup.Keys[0].Group = nil
	overview, err = newConsistentQuotaOverviewService(orphanedGroup).
		GetOverview(context.Background(), 42, "UTC")
	require.NoError(t, err)
	require.Equal(t, QuotaFreshnessUnknown, overview.Freshness)
	require.Contains(t, overview.Warnings, "invalid_quota_snapshot")

	mismatch := quotaOverviewTestSnapshot(asOf)
	service := NewQuotaOverviewService(
		&quotaOverviewRepoStub{snapshots: []*QuotaOverviewSnapshot{mismatch}},
		quotaOverviewCheckerStub{result: inconsistentQuotaOverview("quota_subscription_cache_mismatch")},
	)
	overview, err = service.GetOverview(context.Background(), 42, "UTC")
	require.NoError(t, err)
	require.Equal(t, QuotaFreshnessUnknown, overview.Freshness)
	require.Equal(t, "200.0000000000", *overview.Subscriptions[0].WeeklyWindow.Used)
	require.Nil(t, overview.Subscriptions[0].WeeklyWindow.UsedPercent)
	require.Equal(t, "100.0000000000", *overview.Subscriptions[0].MonthlyWindow.Used)
	require.Equal(t, QuotaWindowUnknown, overview.Subscriptions[0].MonthlyWindow.State)
	require.Nil(t, overview.Subscriptions[0].MonthlyWindow.UsedPercent)
	require.Equal(t, QuotaPeriodUsageUnknown, overview.Subscriptions[0].PeriodUsage.State)
	require.Nil(t, overview.Subscriptions[0].PeriodUsage.TotalRequests)
	require.Nil(t, overview.Subscriptions[0].PeriodUsage.Points)
	require.Equal(t, QuotaGroupUnknown, overview.BillingGroups[1].State)
}

func TestQuotaOverviewRepositoryFailureUsesStaleWithoutInventingZero(t *testing.T) {
	asOf := time.Date(2026, 7, 29, 7, 30, 0, 0, time.UTC)
	snapshot := quotaOverviewTestSnapshot(asOf)
	repo := &quotaOverviewRepoStub{
		snapshots: []*QuotaOverviewSnapshot{snapshot},
		errs:      []error{nil, errors.New("database unavailable")},
	}
	service := NewQuotaOverviewService(
		repo,
		quotaOverviewCheckerStub{result: QuotaOverviewConsistency{Consistent: true}},
	)
	now := asOf.Add(time.Second)
	service.now = func() time.Time { return now }
	_, err := service.GetOverview(context.Background(), 42, "UTC")
	require.NoError(t, err)

	now = now.Add(time.Minute)
	stale, err := service.GetOverview(context.Background(), 42, "UTC")
	require.NoError(t, err)
	require.Equal(t, QuotaFreshnessStale, stale.Freshness)
	require.Equal(t, "128.6400000000", stale.Wallet.Available)
	require.Equal(t, "1.1600000000", stale.Wallet.TodaySpend)
	require.Equal(t, "10.7400000000", stale.Wallet.MonthSpend)
	require.Equal(t, "200.0000000000", *stale.Subscriptions[0].WeeklyWindow.Used)
	require.Nil(t, stale.Subscriptions[0].WeeklyWindow.UsedPercent)
	require.Equal(t, "100.0000000000", *stale.Subscriptions[0].MonthlyWindow.Used)
	require.Equal(t, QuotaWindowUnknown, stale.Subscriptions[0].MonthlyWindow.State)
	require.Nil(t, stale.Subscriptions[0].MonthlyWindow.UsedPercent)
	require.Equal(t, QuotaPeriodUsageUnknown, stale.Subscriptions[0].PeriodUsage.State)
	require.Nil(t, stale.Subscriptions[0].PeriodUsage.TotalTokens)
	require.Nil(t, stale.Subscriptions[0].PeriodUsage.Points)

	firstFailure := NewQuotaOverviewService(
		&quotaOverviewRepoStub{errs: []error{errors.New("database unavailable")}},
		quotaOverviewCheckerStub{result: QuotaOverviewConsistency{Consistent: true}},
	)
	_, err = firstFailure.GetOverview(context.Background(), 42, "UTC")
	require.ErrorIs(t, err, ErrQuotaOverviewUnavailable)
}

func TestQuotaOverviewStaleCacheIsIsolatedByDisplayTimezone(t *testing.T) {
	asOf := time.Date(2026, 7, 29, 7, 30, 0, 0, time.UTC)
	repo := &quotaOverviewRepoStub{
		snapshots: []*QuotaOverviewSnapshot{quotaOverviewTestSnapshot(asOf)},
		errs:      []error{nil, errors.New("database unavailable")},
	}
	service := NewQuotaOverviewService(
		repo,
		quotaOverviewCheckerStub{result: QuotaOverviewConsistency{Consistent: true}},
	)
	service.now = func() time.Time { return asOf.Add(time.Second) }

	_, err := service.GetOverview(context.Background(), 42, "Asia/Shanghai")
	require.NoError(t, err)
	_, err = service.GetOverview(context.Background(), 42, "UTC")
	require.ErrorIs(t, err, ErrQuotaOverviewUnavailable)
	require.Equal(t, []string{"Asia/Shanghai", "UTC"}, repo.timezones)
}

func TestBillingQuotaOverviewConsistencyCheckerCacheMissAndStrictMatches(t *testing.T) {
	asOf := time.Date(2026, 7, 29, 7, 30, 0, 0, time.UTC)
	snapshot := quotaOverviewTestSnapshot(asOf)

	missChecker := NewBillingQuotaOverviewConsistencyChecker(&quotaOverviewCacheStub{
		balanceErr: redis.Nil,
	})
	result, err := missChecker.CheckQuotaOverviewConsistency(context.Background(), snapshot)
	require.NoError(t, err)
	require.True(t, result.Consistent)

	sub := snapshot.Subscriptions[0]
	_, periodEnd, ok := AnchoredWeeklyWindow(sub.StartsAt, asOf)
	require.True(t, ok)
	_, monthlyPeriodEnd, ok := AnchoredMonthlyWindow(sub.StartsAt, asOf)
	require.True(t, ok)
	cacheData := &SubscriptionCacheData{
		SubscriptionID:     sub.ID,
		Status:             sub.Status,
		StartsAt:           sub.StartsAt,
		ExpiresAt:          sub.ExpiresAt,
		WeeklyWindowStart:  sub.WeeklyWindowStart,
		WeeklyWindowEnd:    periodEnd,
		MonthlyWindowStart: sub.MonthlyWindowStart,
		MonthlyWindowEnd:   monthlyPeriodEnd,
		WeeklyUsage:        200,
		MonthlyUsage:       100,
		Version:            sub.UpdatedAt.UnixMicro(),
	}
	cache := &quotaOverviewCacheStub{
		// The insignificant binary/11th-decimal drift must not turn a
		// NUMERIC(20,10) value into a false mismatch.
		balance: 128.64000000004,
		subs:    map[int64]*SubscriptionCacheData{sub.GroupID: cacheData},
	}
	checker := NewBillingQuotaOverviewConsistencyChecker(cache)
	result, err = checker.CheckQuotaOverviewConsistency(context.Background(), snapshot)
	require.NoError(t, err)
	require.True(t, result.Consistent)

	tests := map[string]func(*SubscriptionCacheData){
		"identity": func(value *SubscriptionCacheData) { value.SubscriptionID++ },
		"anchor":   func(value *SubscriptionCacheData) { value.StartsAt = value.StartsAt.Add(time.Second) },
		"window":   func(value *SubscriptionCacheData) { value.WeeklyWindowEnd = value.WeeklyWindowEnd.Add(time.Second) },
		"monthly_window": func(value *SubscriptionCacheData) {
			value.MonthlyWindowEnd = value.MonthlyWindowEnd.Add(time.Second)
		},
		"version": func(value *SubscriptionCacheData) { value.Version++ },
		"usage":   func(value *SubscriptionCacheData) { value.WeeklyUsage++ },
		"monthly_usage": func(value *SubscriptionCacheData) {
			value.MonthlyUsage++
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			value := *cacheData
			mutate(&value)
			cache.subs[sub.GroupID] = &value
			got, checkErr := checker.CheckQuotaOverviewConsistency(context.Background(), snapshot)
			require.NoError(t, checkErr)
			require.False(t, got.Consistent)
			require.Contains(t, got.WarningCodes, "quota_subscription_cache_mismatch")
		})
	}

	t.Run("no-key active subscription is still compared", func(t *testing.T) {
		noKeySnapshot := quotaOverviewTestSnapshot(asOf)
		noKeySnapshot.Keys = nil
		value := *cacheData
		value.WeeklyUsage = 199
		noKeyCache := &quotaOverviewCacheStub{
			balance: 128.64,
			subs:    map[int64]*SubscriptionCacheData{sub.GroupID: &value},
		}
		got, checkErr := NewBillingQuotaOverviewConsistencyChecker(noKeyCache).
			CheckQuotaOverviewConsistency(context.Background(), noKeySnapshot)
		require.NoError(t, checkErr)
		require.False(t, got.Consistent)
	})

	t.Run("expired subscription requires cache miss", func(t *testing.T) {
		expiredSnapshot := quotaOverviewTestSnapshot(asOf)
		expiredSnapshot.Subscriptions[0].Status = SubscriptionStatusExpired
		expiredSnapshot.Subscriptions[0].ExpiresAt = asOf.Add(-time.Second)
		value := *cacheData
		expiredCache := &quotaOverviewCacheStub{
			balance: 128.64,
			subs:    map[int64]*SubscriptionCacheData{sub.GroupID: &value},
		}
		got, checkErr := NewBillingQuotaOverviewConsistencyChecker(expiredCache).
			CheckQuotaOverviewConsistency(context.Background(), expiredSnapshot)
		require.NoError(t, checkErr)
		require.False(t, got.Consistent)
		require.Contains(t, got.WarningCodes, "quota_subscription_cache_mismatch")
	})

	t.Run("member key without subscription requires cache miss", func(t *testing.T) {
		missingSnapshot := quotaOverviewTestSnapshot(asOf)
		missingSnapshot.Subscriptions = nil
		value := *cacheData
		missingCache := &quotaOverviewCacheStub{
			balance: 128.64,
			subs:    map[int64]*SubscriptionCacheData{sub.GroupID: &value},
		}
		got, checkErr := NewBillingQuotaOverviewConsistencyChecker(missingCache).
			CheckQuotaOverviewConsistency(context.Background(), missingSnapshot)
		require.NoError(t, checkErr)
		require.False(t, got.Consistent)
		require.Contains(t, got.WarningCodes, "quota_subscription_cache_mismatch")
	})
}
