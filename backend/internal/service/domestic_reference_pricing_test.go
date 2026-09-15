package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestDomesticReferencePricesPublishWithoutInventedCacheFees(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "..", "deploy", "pricing", "model_pricing_overrides.json"))
	require.NoError(t, err)
	refs, err := (&PricingService{}).parsePricingOverrides(body)
	require.NoError(t, err)
	require.Len(t, refs, 20)
	for name, exact := range refs {
		t.Run(name, func(t *testing.T) {
			require.True(t, exact.InputCostPerTokenSet)
			require.True(t, exact.OutputCostPerTokenSet)
			require.True(t, exact.CacheReadInputTokenCostSet)
			var write *float64
			if exact.CacheCreationInputTokenCostSet {
				write = &exact.CacheCreationInputTokenCost
			}
			svc := comparisonCatalogService(&Group{ID: 29, Name: "国模", RateMultiplier: 0.45}, ChannelModelPricing{
				InputPrice: &exact.InputCostPerToken, OutputPrice: &exact.OutputCostPerToken,
				CacheReadPrice: &exact.CacheReadInputTokenCost, CacheWritePrice: write,
			}, exact)
			response, _, err := svc.PublicSnapshot(context.Background(), "zh-CN")
			require.NoError(t, err)
			require.Len(t, response.Items, 1)
			item := response.Items[0]
			require.InDelta(t, exact.InputCostPerToken*0.45, *item.Pricing.InputPrice, 1e-15)
			require.InDelta(t, exact.OutputCostPerToken, *item.OfficialPricing.OutputPrice, 1e-15)
			if write == nil {
				require.Nil(t, item.Pricing.CacheWritePrice)
				require.Nil(t, item.OfficialPricing.CacheWritePrice)
			} else {
				require.InDelta(t, *write*0.45, *item.Pricing.CacheWritePrice, 1e-15)
			}
			if name == "minimax-m3" {
				require.Len(t, item.Pricing.Intervals, 1)
				require.Equal(t, 524288, item.Pricing.Intervals[0].MinTokens)
				require.InDelta(t, 0.12e-6, *item.OfficialPricing.Intervals[0].CacheReadPrice, 1e-15)
			}
		})
	}
	require.InDelta(t, 0.15e-6, refs["glm-5.3-flash"].InputCostPerToken, 1e-15)
	require.InDelta(t, 0.50e-6, refs["glm-5.3-flash"].OutputCostPerToken, 1e-15)
	require.True(t, refs["mimo-v2.5"].CacheCreationInputTokenCostSet)
	require.Zero(t, refs["mimo-v2.5"].CacheCreationInputTokenCost)
	for _, unknown := range []string{"k3", "hy3-preview", "deepseek-v4.1-flash/Global"} {
		require.Nil(t, refs[unknown], "an upstream-only alias is not an official quote")
	}

	// The same rule feeds billing and catalog projection. The inclusive 512K
	// limit must not be rounded to 512000 or charged twice.
	pricing := modelPricingFromLiteLLM(refs["minimax-m3"])
	billing := NewBillingService(&config.Config{}, nil)
	for _, total := range []int{524288, 524289} {
		tokens := UsageTokens{InputTokens: total - 10, CacheReadTokens: 10, OutputTokens: 100}
		bd := billing.computeTokenBreakdown(pricing, tokens, 0.45, "", true)
		multiplier := 1.0
		if total > 524288 {
			multiplier = 2
		}
		expected := (float64(tokens.InputTokens)*.30e-6 + 10*.06e-6 + 100*1.20e-6) * multiplier * .45
		require.InDelta(t, expected, bd.ActualCost, 1e-12)
	}
}
