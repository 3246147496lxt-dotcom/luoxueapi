//go:build unit

package service

import (
	"errors"
	"testing"
)

// TestBuildUsageBillingCommand_SubscriptionAppliesRateMultiplier locks in the fix
// that subscription-mode billing honours the group (and any user-specific) rate
// multiplier — i.e. cmd.SubscriptionCost tracks ActualCost (= TotalCost *
// RateMultiplier), not raw TotalCost.
func TestBuildUsageBillingCommand_SubscriptionAppliesRateMultiplier(t *testing.T) {
	t.Parallel()

	groupID := int64(7)
	subID := int64(42)

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
				Subscription:       &UserSubscription{ID: subID},
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
		})
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
