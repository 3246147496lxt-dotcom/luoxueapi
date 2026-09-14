package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// The channel editor stores the official USD quote × 1. The catalog projection
// applies the selected group's runtime multiplier and keeps the result in USD
// wallet units for direct display and charging.
func TestCatalogPricingRuleKeepsEffectiveUSDValue(t *testing.T) {
	rawInput := 0.000005 // $5/MTok × 1, stored per token
	rawOutput := 0.00003 // $30/MTok × 1
	rawCacheWrite := 0.00000625
	rawCacheRead := 0.00000025
	intervalInput := 0.00001

	pricing := &ChannelModelPricing{
		BillingMode:     BillingModeToken,
		InputPrice:      &rawInput,
		OutputPrice:     &rawOutput,
		CacheWritePrice: &rawCacheWrite,
		CacheReadPrice:  &rawCacheRead,
		Intervals: []PricingInterval{{
			MinTokens:  272_000,
			InputPrice: &intervalInput,
		}},
	}
	public := catalogPricingFromChannel(pricing, &Group{RateMultiplier: 0.056})

	require.NotNil(t, public)
	require.Equal(t, catalogPricingCurrencyUSD, public.Currency)
	require.InDelta(t, 0.000005*0.056, *public.InputPrice, 1e-15)
	require.InDelta(t, 0.00003*0.056, *public.OutputPrice, 1e-15)
	require.InDelta(t, 0.00000625*0.056, *public.CacheWritePrice, 1e-15)
	require.InDelta(t, 0.00000025*0.056, *public.CacheReadPrice, 1e-15)
	require.Len(t, public.Intervals, 1)
	require.InDelta(t, 0.00001*0.056, *public.Intervals[0].InputPrice, 1e-15)

	// JSON serialization must carry the effective USD value unchanged so the
	// client can display the same direct USD amount without a second conversion.
	wire, err := json.Marshal(public)
	require.NoError(t, err)
	var decoded struct {
		Currency   string  `json:"currency"`
		InputPrice float64 `json:"input_price"`
	}
	require.NoError(t, json.Unmarshal(wire, &decoded))
	require.Equal(t, catalogPricingCurrencyUSD, decoded.Currency)
	require.InDelta(t, 0.000005*0.056, decoded.InputPrice, 1e-15)
}
