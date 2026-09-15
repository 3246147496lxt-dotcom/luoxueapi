//go:build unit

package admin

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestModelDefaultPricingMiniMaxM3Intervals(t *testing.T) {
	quote := modelDefaultPricingResponseFrom(&service.LiteLLMModelPricing{
		InputCostPerToken: 0.3e-6, OutputCostPerToken: 1.2e-6, CacheReadInputTokenCost: 0.06e-6,
		LongContextInputTokenThreshold: 524288,
		LongContextInputCostMultiplier: 2, LongContextOutputCostMultiplier: 2,
	})
	require.True(t, quote.Found)
	require.Len(t, quote.Intervals, 2)
	base, long := quote.Intervals[0], quote.Intervals[1]
	require.Equal(t, 0, base.MinTokens)
	require.Equal(t, 524288, *base.MaxTokens)
	require.Equal(t, 524288, long.MinTokens)
	require.Nil(t, long.MaxTokens)
	require.InDelta(t, 0.3e-6, *base.InputPrice, 1e-15)
	require.InDelta(t, 1.2e-6, *base.OutputPrice, 1e-15)
	require.InDelta(t, 0.06e-6, *base.CacheReadPrice, 1e-15)
	require.InDelta(t, 0.6e-6, *long.InputPrice, 1e-15)
	require.InDelta(t, 2.4e-6, *long.OutputPrice, 1e-15)
	require.InDelta(t, 0.12e-6, *long.CacheReadPrice, 1e-15)

	intervals := []service.PricingInterval{
		{MinTokens: base.MinTokens, MaxTokens: base.MaxTokens, InputPrice: base.InputPrice},
		{MinTokens: long.MinTokens, MaxTokens: long.MaxTokens, InputPrice: long.InputPrice},
	}
	require.Equal(t, base.InputPrice, service.FindMatchingInterval(intervals, 524288).InputPrice)
	require.Equal(t, long.InputPrice, service.FindMatchingInterval(intervals, 524289).InputPrice)

	body, err := json.Marshal(quote)
	require.NoError(t, err)
	var wire struct {
		CacheWritePrice *float64         `json:"cache_write_price"`
		Intervals       []map[string]any `json:"intervals"`
	}
	require.NoError(t, json.Unmarshal(body, &wire))
	require.Nil(t, wire.CacheWritePrice)
	for _, interval := range wire.Intervals {
		require.Contains(t, interval, "cache_write_price")
		require.Nil(t, interval["cache_write_price"], "an unknown write quote must not become a free price")
	}
}

func TestModelDefaultPricingIntervalsPreserveExplicitZeroAndInputCacheMultiplier(t *testing.T) {
	quote := modelDefaultPricingResponseFrom(&service.LiteLLMModelPricing{
		InputCostPerToken: 1e-6, OutputCostPerToken: 4e-6,
		CacheCreationInputTokenCostSet: true, CacheReadInputTokenCost: 0.2e-6,
		LongContextInputTokenThreshold: 100,
		LongContextInputCostMultiplier: 2, LongContextOutputCostMultiplier: 3,
	})
	require.Len(t, quote.Intervals, 2)
	for _, interval := range quote.Intervals {
		require.NotNil(t, interval.CacheWritePrice)
		require.Zero(t, *interval.CacheWritePrice)
	}
	require.InDelta(t, 2e-6, *quote.Intervals[1].InputPrice, 1e-15)
	require.InDelta(t, 12e-6, *quote.Intervals[1].OutputPrice, 1e-15)
	require.InDelta(t, 0.4e-6, *quote.Intervals[1].CacheReadPrice, 1e-15)
}

func TestModelDefaultPricingIntervalsRequireThresholdAndHigherPrice(t *testing.T) {
	for _, test := range []struct {
		name      string
		threshold int
		input     float64
		output    float64
	}{
		{"no threshold", 0, 2, 2},
		{"no multipliers", 524288, 0, 0},
		{"same price", 524288, 1, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			quote := modelDefaultPricingResponseFrom(&service.LiteLLMModelPricing{
				InputCostPerToken: 0.3e-6, OutputCostPerToken: 1.2e-6,
				LongContextInputTokenThreshold: test.threshold,
				LongContextInputCostMultiplier: test.input, LongContextOutputCostMultiplier: test.output,
			})
			require.Empty(t, quote.Intervals)
			body, err := json.Marshal(quote)
			require.NoError(t, err)
			require.NotContains(t, string(body), `"intervals"`)
		})
	}
}
