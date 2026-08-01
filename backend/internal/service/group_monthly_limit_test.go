//go:build unit

package service

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEffectiveMonthlyLimitUSDUsesExplicitValueOrWeeklyFallback(t *testing.T) {
	weekly := 25.0
	explicit := 70.0
	zero := 0.0
	negative := -1.0
	nan := math.NaN()
	positiveInfinity := math.Inf(1)
	negativeInfinity := math.Inf(-1)
	overflowingWeekly := math.MaxFloat64

	tests := []struct {
		name       string
		group      *Group
		want       float64
		configured bool
	}{
		{
			name:       "missing monthly uses four weekly allowances",
			group:      &Group{WeeklyLimitUSD: &weekly},
			want:       100,
			configured: true,
		},
		{
			name:       "explicit monthly wins",
			group:      &Group{WeeklyLimitUSD: &weekly, MonthlyLimitUSD: &explicit},
			want:       70,
			configured: true,
		},
		{
			name:       "explicit zero is exhausted and does not fall back",
			group:      &Group{WeeklyLimitUSD: &weekly, MonthlyLimitUSD: &zero},
			want:       0,
			configured: true,
		},
		{
			name:       "negative explicit value fails closed without fallback",
			group:      &Group{WeeklyLimitUSD: &weekly, MonthlyLimitUSD: &negative},
			want:       0,
			configured: true,
		},
		{
			name:       "NaN explicit value fails closed without fallback",
			group:      &Group{WeeklyLimitUSD: &weekly, MonthlyLimitUSD: &nan},
			want:       0,
			configured: true,
		},
		{
			name:       "positive infinity explicit value fails closed without fallback",
			group:      &Group{WeeklyLimitUSD: &weekly, MonthlyLimitUSD: &positiveInfinity},
			want:       0,
			configured: true,
		},
		{
			name:       "negative infinity explicit value fails closed without fallback",
			group:      &Group{WeeklyLimitUSD: &weekly, MonthlyLimitUSD: &negativeInfinity},
			want:       0,
			configured: true,
		},
		{
			name:       "NaN weekly fallback fails closed",
			group:      &Group{WeeklyLimitUSD: &nan},
			want:       0,
			configured: true,
		},
		{
			name:       "positive infinity weekly fallback fails closed",
			group:      &Group{WeeklyLimitUSD: &positiveInfinity},
			want:       0,
			configured: true,
		},
		{
			name:       "negative infinity weekly fallback fails closed",
			group:      &Group{WeeklyLimitUSD: &negativeInfinity},
			want:       0,
			configured: true,
		},
		{
			name:       "overflowing weekly fallback fails closed",
			group:      &Group{WeeklyLimitUSD: &overflowingWeekly},
			want:       0,
			configured: true,
		},
		{
			name:       "no configured source",
			group:      &Group{},
			configured: false,
		},
		{
			name:       "nil group",
			group:      nil,
			configured: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, configured := tt.group.EffectiveMonthlyLimitUSD()
			require.Equal(t, tt.configured, configured)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestEffectiveWeeklyLimitUSDFailsClosedForPresentInvalidValues(t *testing.T) {
	valid := 25.0
	nan := math.NaN()
	positiveInfinity := math.Inf(1)
	negativeInfinity := math.Inf(-1)
	negative := -1.0

	tests := []struct {
		name       string
		group      *Group
		want       float64
		configured bool
	}{
		{name: "nil group", group: nil, want: 0, configured: false},
		{name: "omitted", group: &Group{}, want: 0, configured: false},
		{name: "valid", group: &Group{WeeklyLimitUSD: &valid}, want: valid, configured: true},
		{name: "negative", group: &Group{WeeklyLimitUSD: &negative}, want: 0, configured: true},
		{name: "NaN", group: &Group{WeeklyLimitUSD: &nan}, want: 0, configured: true},
		{name: "positive infinity", group: &Group{WeeklyLimitUSD: &positiveInfinity}, want: 0, configured: true},
		{name: "negative infinity", group: &Group{WeeklyLimitUSD: &negativeInfinity}, want: 0, configured: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, configured := tt.group.EffectiveWeeklyLimitUSD()
			require.Equal(t, tt.configured, configured)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestInvalidWeeklyLimitFailsClosedAlongSubscriptionAdmissionPath(t *testing.T) {
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	startsAt := now.Add(-time.Hour)
	nan := math.NaN()
	validMonthly := 100.0
	subscription := &UserSubscription{
		StartsAt:          startsAt,
		WeeklyWindowStart: &startsAt,
		WeeklyUsageUSD:    0,
	}
	group := &Group{WeeklyLimitUSD: &nan, MonthlyLimitUSD: &validMonthly}

	require.False(t, subscription.CheckWeeklyLimitAt(group, 0, now))
}

func TestGroupLimitPredicatesRejectInvalidNumbers(t *testing.T) {
	positive := 1.0
	zero := 0.0
	negative := -1.0
	nan := math.NaN()
	positiveInfinity := math.Inf(1)
	negativeInfinity := math.Inf(-1)

	tests := []struct {
		name  string
		limit *float64
		want  bool
	}{
		{name: "missing", limit: nil, want: false},
		{name: "positive", limit: &positive, want: true},
		{name: "zero", limit: &zero, want: true},
		{name: "negative", limit: &negative, want: false},
		{name: "NaN", limit: &nan, want: false},
		{name: "positive infinity", limit: &positiveInfinity, want: false},
		{name: "negative infinity", limit: &negativeInfinity, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			group := &Group{
				DailyLimitUSD:   tt.limit,
				WeeklyLimitUSD:  tt.limit,
				MonthlyLimitUSD: tt.limit,
			}
			require.Equal(t, tt.want, group.HasDailyLimit())
			require.Equal(t, tt.want, group.HasWeeklyLimit())
			require.Equal(t, tt.want, group.HasMonthlyLimit())
		})
	}

	var group *Group
	require.False(t, group.HasDailyLimit())
	require.False(t, group.HasWeeklyLimit())
	require.False(t, group.HasMonthlyLimit())
}
