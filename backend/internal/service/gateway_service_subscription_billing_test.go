//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestBuildUsageBillingCommand_SubscriptionAppliesRateMultiplier locks in the fix
// that subscription-mode billing honours the group (and any user-specific) rate
// multiplier — i.e. cmd.SubscriptionCost tracks ActualCost (= TotalCost *
// RateMultiplier), not raw TotalCost.
func TestBuildUsageBillingCommand_SubscriptionAppliesRateMultiplier(t *testing.T) {
	t.Parallel()

	groupID := int64(7)
	subID := int64(42)
	subscriptionStartsAt := time.Date(2026, time.July, 31, 9, 8, 7, 654321000, time.FixedZone("UTC+8", 8*60*60))

	tests := []struct {
		name           string
		totalCost      float64
		actualCost     float64
		isSubscription bool
		wantSub        float64
		wantBalance    float64
	}{
		{
			name:           "subscription with 2x multiplier consumes 2x quota",
			totalCost:      1.0,
			actualCost:     2.0,
			isSubscription: true,
			wantSub:        2.0,
			wantBalance:    0,
		},
		{
			name:           "subscription with 0.5x multiplier consumes 0.5x quota",
			totalCost:      1.0,
			actualCost:     0.5,
			isSubscription: true,
			wantSub:        0.5,
			wantBalance:    0,
		},
		{
			name:           "free subscription (multiplier 0) consumes no quota",
			totalCost:      1.0,
			actualCost:     0,
			isSubscription: true,
			wantSub:        0,
			wantBalance:    0,
		},
		{
			name:           "balance billing keeps using ActualCost (regression)",
			totalCost:      1.0,
			actualCost:     2.0,
			isSubscription: false,
			wantSub:        0,
			wantBalance:    2.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := &postUsageBillingParams{
				Cost:               &CostBreakdown{TotalCost: tt.totalCost, ActualCost: tt.actualCost},
				User:               &User{ID: 1},
				APIKey:             &APIKey{ID: 2, GroupID: &groupID},
				Account:            &Account{ID: 3},
				Subscription:       &UserSubscription{ID: subID, StartsAt: subscriptionStartsAt},
				IsSubscriptionBill: tt.isSubscription,
			}

			cmd := buildUsageBillingCommand("req-1", nil, p)
			if cmd == nil {
				t.Fatal("buildUsageBillingCommand returned nil")
			}
			if cmd.SubscriptionCost != tt.wantSub {
				t.Errorf("SubscriptionCost = %v, want %v", cmd.SubscriptionCost, tt.wantSub)
			}
			if cmd.BalanceCost != tt.wantBalance {
				t.Errorf("BalanceCost = %v, want %v", cmd.BalanceCost, tt.wantBalance)
			}
			if tt.isSubscription {
				if cmd.SubscriptionStartsAt == nil || !cmd.SubscriptionStartsAt.Equal(subscriptionStartsAt) {
					t.Errorf("SubscriptionStartsAt = %v, want %v", cmd.SubscriptionStartsAt, subscriptionStartsAt)
				}
			} else if cmd.SubscriptionStartsAt != nil {
				t.Errorf("SubscriptionStartsAt = %v, want nil for balance billing", cmd.SubscriptionStartsAt)
			}
		})
	}
}

func TestUsageBillingFingerprintIncludesSubscriptionTerm(t *testing.T) {
	t.Parallel()

	subscriptionID := int64(42)
	firstStartsAt := time.Date(2026, time.July, 1, 3, 4, 5, 6000, time.UTC)
	secondStartsAt := firstStartsAt.Add(24 * time.Hour)
	first := &UsageBillingCommand{
		RequestID:            "req-1",
		APIKeyID:             2,
		UserID:               1,
		SubscriptionID:       &subscriptionID,
		SubscriptionStartsAt: &firstStartsAt,
		SubscriptionCost:     1,
	}
	second := *first
	second.SubscriptionStartsAt = &secondStartsAt

	first.Normalize()
	second.Normalize()

	if first.RequestFingerprint == second.RequestFingerprint {
		t.Fatal("subscription terms with the same row ID must have different billing fingerprints")
	}
}

type termAwareUsageRepoStub struct {
	userSubRepoNoop
	called   bool
	id       int64
	startsAt time.Time
	costUSD  float64
	err      error
}

func (s *termAwareUsageRepoStub) IncrementUsageForTerm(
	_ context.Context,
	id int64,
	startsAt time.Time,
	costUSD float64,
) error {
	s.called = true
	s.id = id
	s.startsAt = startsAt
	s.costUSD = costUSD
	return s.err
}

func TestLegacySubscriptionBillingRequiresTermAwareRepository(t *testing.T) {
	t.Parallel()

	startsAt := time.Date(2026, time.July, 31, 9, 8, 7, 0, time.UTC)
	subscription := &UserSubscription{ID: 42, StartsAt: startsAt}

	err := incrementSubscriptionUsageForTerm(
		t.Context(),
		userSubRepoNoop{},
		subscription,
		2.5,
	)
	if !errors.Is(err, ErrUsageBillingSubscriptionTermUnsupported) {
		t.Fatalf("incrementSubscriptionUsageForTerm error = %v, want %v", err, ErrUsageBillingSubscriptionTermUnsupported)
	}

	repo := &termAwareUsageRepoStub{}
	err = incrementSubscriptionUsageForTerm(t.Context(), repo, subscription, 2.5)
	if err != nil {
		t.Fatalf("incrementSubscriptionUsageForTerm returned error: %v", err)
	}
	if !repo.called || repo.id != subscription.ID || !repo.startsAt.Equal(startsAt) || repo.costUSD != 2.5 {
		t.Fatalf("term-aware increment = called:%v id:%d starts_at:%v cost:%v", repo.called, repo.id, repo.startsAt, repo.costUSD)
	}
}

func TestLegacySubscriptionBillingDoesNotFallBackToIDOnlyIncrement(t *testing.T) {
	t.Parallel()

	// userSubRepoNoop.IncrementUsage panics. The legacy path must detect that
	// the repository lacks IncrementUsageForTerm and skip the unsafe ID-only
	// method instead.
	result, err := applyUsageBilling(t.Context(), "req-legacy-term", nil, &postUsageBillingParams{
		Cost:               &CostBreakdown{TotalCost: 1, ActualCost: 1},
		User:               &User{ID: 1},
		APIKey:             &APIKey{ID: 2},
		Account:            &Account{ID: 3},
		Subscription:       &UserSubscription{ID: 42, StartsAt: time.Now().UTC()},
		IsSubscriptionBill: true,
	}, &billingDeps{userSubRepo: userSubRepoNoop{}}, nil)
	if result != nil {
		t.Fatalf("applyUsageBilling result = %+v, want nil", result)
	}
	if !errors.Is(err, ErrUsageBillingSubscriptionTermUnsupported) {
		t.Fatalf("applyUsageBilling error = %v, want %v", err, ErrUsageBillingSubscriptionTermUnsupported)
	}
}

func TestBuildUsageBillingCommand_MissingSubscriptionNeverFallsBackToBalance(t *testing.T) {
	t.Parallel()

	groupID := int64(7)
	p := &postUsageBillingParams{
		Cost:               &CostBreakdown{TotalCost: 1, ActualCost: 2},
		User:               &User{ID: 1},
		APIKey:             &APIKey{ID: 2, GroupID: &groupID},
		Account:            &Account{ID: 3},
		IsSubscriptionBill: true,
	}

	cmd := buildUsageBillingCommand("req-1", nil, p)
	if cmd == nil {
		t.Fatal("buildUsageBillingCommand returned nil")
	}
	if cmd.BalanceCost != 0 {
		t.Fatalf("BalanceCost = %v, want 0", cmd.BalanceCost)
	}
	if cmd.SubscriptionCost != 0 {
		t.Fatalf("SubscriptionCost = %v, want 0", cmd.SubscriptionCost)
	}
}

func TestApplyUsageBilling_MissingSubscriptionFailsClosed(t *testing.T) {
	t.Parallel()

	p := &postUsageBillingParams{
		Cost:               &CostBreakdown{TotalCost: 1, ActualCost: 1},
		IsSubscriptionBill: true,
	}

	_, err := applyUsageBilling(t.Context(), "req-1", nil, p, &billingDeps{}, nil)
	if !errors.Is(err, ErrSubscriptionBillingContextRequired) {
		t.Fatalf("applyUsageBilling error = %v, want %v", err, ErrSubscriptionBillingContextRequired)
	}
}

func TestResolveUsageSubscriptionBillingRequiresTrustedMatchingGroup(t *testing.T) {
	t.Parallel()

	groupID := int64(7)
	sub := &UserSubscription{ID: 9, UserID: 1, GroupID: groupID}

	tests := []struct {
		name   string
		apiKey *APIKey
		sub    *UserSubscription
		want   bool
	}{
		{name: "missing API key", apiKey: nil, sub: nil},
		{name: "group id without group", apiKey: &APIKey{UserID: 1, GroupID: &groupID}, sub: nil},
		{name: "untrusted group", apiKey: &APIKey{UserID: 1, GroupID: &groupID, Group: &Group{ID: groupID, SubscriptionType: SubscriptionTypeSubscription}}, sub: sub},
		{name: "standard group with subscription", apiKey: &APIKey{UserID: 1, GroupID: &groupID, Group: &Group{ID: groupID, Hydrated: true}}, sub: sub},
		{name: "subscription group missing subscription", apiKey: &APIKey{UserID: 1, GroupID: &groupID, Group: &Group{ID: groupID, Hydrated: true, SubscriptionType: SubscriptionTypeSubscription}}, sub: nil},
		{
			name: "trusted matching subscription",
			apiKey: &APIKey{
				UserID:  1,
				GroupID: &groupID,
				Group:   &Group{ID: groupID, Hydrated: true, SubscriptionType: SubscriptionTypeSubscription},
			},
			sub:  sub,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveUsageSubscriptionBilling(tt.apiKey, tt.sub)
			if tt.want {
				if err != nil {
					t.Fatalf("resolveUsageSubscriptionBilling error = %v", err)
				}
				if !got {
					t.Fatal("resolveUsageSubscriptionBilling returned balance mode")
				}
				return
			}
			if err == nil {
				t.Fatalf("resolveUsageSubscriptionBilling = %v, want error", got)
			}
		})
	}
}
