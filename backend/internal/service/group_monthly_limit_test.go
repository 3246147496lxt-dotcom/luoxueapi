//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEffectiveMonthlyLimitUSDUsesExplicitValueOrWeeklyFallback(t *testing.T) {
	weekly := 25.0
	explicit := 70.0
	zero := 0.0
	negative := -1.0

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
