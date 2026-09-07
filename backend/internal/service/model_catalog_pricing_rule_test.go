package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// The channel editor stores the official USD quote × 70 (the points/CNY and
// USD/CNY conversion baseline).  The catalog projection then applies the
// selected group's runtime multiplier and keeps the result in CREDIT units;
// the public client is responsible for the final CREDIT → CNY / 10 display
// conversion.  Keep these layers explicit so a future change cannot silently
// divide the API value or apply the channel multiplier a second time.
func TestCatalogPricingRuleKeepsEffectiveCreditValueUntilClientConversion(t *testing.T) {
	rawInput := 0.00035 // $5/MTok × 70, stored per token
	rawOutput := 0.0021 // $30/MTok × 70
	rawCacheWrite := 0.0004375
	rawCacheRead := 0.000035
	intervalInput := 0.0007

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
	require.Equal(t, catalogPricingCurrencyCredit, public.Currency)
	require.InDelta(t, 0.00035*0.056, *public.InputPrice, 1e-15)
	require.InDelta(t, 0.0021*0.056, *public.OutputPrice, 1e-15)
	require.InDelta(t, 0.0004375*0.056, *public.CacheWritePrice, 1e-15)
	require.InDelta(t, 0.000035*0.056, *public.CacheReadPrice, 1e-15)
	require.Len(t, public.Intervals, 1)
	require.InDelta(t, 0.0007*0.056, *public.Intervals[0].InputPrice, 1e-15)

	// JSON serialization must carry the effective CREDIT value unchanged.  A
	// browser can then render ¥ by multiplying per-token values by 1e6 and
	// dividing by 10 exactly once.
	wire, err := json.Marshal(public)
	require.NoError(t, err)
	var decoded struct {
		Currency   string  `json:"currency"`
		InputPrice float64 `json:"input_price"`
	}
	require.NoError(t, json.Unmarshal(wire, &decoded))
	require.Equal(t, catalogPricingCurrencyCredit, decoded.Currency)
	require.InDelta(t, 0.00035*0.056, decoded.InputPrice, 1e-15)
}
