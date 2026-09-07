package service

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOfficialPriceToChannelUsesBaselineMultiplier(t *testing.T) {
	tests := []struct {
		name     string
		official float64
		want     float64
	}{
		{name: "input quote", official: 5e-6, want: 0.00035},
		{name: "output quote", official: 30e-6, want: 0.0021},
		{name: "zero quote", official: 0, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.InDelta(t, tt.want, OfficialPriceToChannel(tt.official), 1e-15)
		})
	}
	require.Equal(t, 70.0, ChannelPricingBaselineMultiplier)
}

func TestOfficialPriceToChannelLeavesInvalidValuesUnchanged(t *testing.T) {
	for _, value := range []float64{-1, math.Inf(1), math.Inf(-1)} {
		require.Equal(t, value, OfficialPriceToChannel(value))
	}

	got := OfficialPriceToChannel(math.NaN())
	require.True(t, math.IsNaN(got), "NaN must not be converted into a usable/free price")
}
