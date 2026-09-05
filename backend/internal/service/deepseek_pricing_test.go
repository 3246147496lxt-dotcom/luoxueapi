//go:build unit

package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestDeepseekPeakMultiplierAt(t *testing.T) {
	weekday := func(hour, minute int) time.Time {
		return time.Date(2026, time.August, 24, hour, minute, 0, 0, time.UTC) // Monday
	}
	cases := []struct {
		name string
		now  time.Time
		want float64
	}{
		{"first peak window starts", weekday(1, 0), 2},
		{"first peak window ends", weekday(3, 59), 2},
		{"first peak window upper bound", weekday(4, 0), 1},
		{"second peak window starts", weekday(6, 0), 2},
		{"second peak window ends", weekday(9, 59), 2},
		{"second peak window upper bound", weekday(10, 0), 1},
		{"weekday off peak", weekday(12, 0), 1},
		{
			"Beijing Saturday is always off peak",
			time.Date(2026, time.August, 22, 2, 0, 0, 0, time.UTC),
			1,
		},
		{
			"UTC Saturday crossing to Beijing Sunday is off peak",
			time.Date(2026, time.August, 22, 16, 30, 0, 0, time.UTC),
			1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, deepseekPeakMultiplierAt(tc.now))
		})
	}
}

func TestIsDeepSeekModel(t *testing.T) {
	for _, model := range []string{
		"deepseek-v4-pro",
		"deepseek-v4-flash",
		"deepseek-v4-flash-vision-exp",
		"deepseek-v3-2-251201",
		" DEEPSEEK-foo ",
	} {
		require.True(t, isDeepSeekModel(model), model)
	}
	for _, model := range []string{"", "deepseek", "deepseekcoder", "gpt-5.4"} {
		require.False(t, isDeepSeekModel(model), model)
	}
}

func TestGetFallbackPricing_DeepSeekOfficialCards(t *testing.T) {
	service := newTestBillingService()
	cases := []struct {
		model                    string
		input, output, cacheRead float64
	}{
		{"deepseek-v4-pro", deepseekProOffPeakInputPrice, deepseekProOffPeakOutputPrice, deepseekProOffPeakCacheRead},
		{"deepseek-v4-flash", deepseekFlashOffPeakInputPrice, deepseekFlashOffPeakOutputPrice, deepseekFlashOffPeakCacheRead},
		{"deepseek-v4-flash-vision-exp", deepseekFlashOffPeakInputPrice, deepseekFlashOffPeakOutputPrice, deepseekFlashOffPeakCacheRead},
		{"deepseek-chat", deepseekFlashOffPeakInputPrice, deepseekFlashOffPeakOutputPrice, deepseekFlashOffPeakCacheRead},
		{"deepseek-unknown-model", deepseekFlashOffPeakInputPrice, deepseekFlashOffPeakOutputPrice, deepseekFlashOffPeakCacheRead},
	}
	for _, tc := range cases {
		t.Run(tc.model, func(t *testing.T) {
			pricing := service.getFallbackPricing(tc.model)
			require.NotNil(t, pricing)
			require.InDelta(t, tc.input, pricing.InputPricePerToken, 1e-15)
			require.InDelta(t, tc.output, pricing.OutputPricePerToken, 1e-15)
			require.InDelta(t, tc.cacheRead, pricing.CacheReadPricePerToken, 1e-15)
		})
	}
}

func TestGetModelPricing_DeepSeekForcesOfficialRatesOverLiteLLM(t *testing.T) {
	pricingService := &PricingService{pricingData: map[string]*LiteLLMModelPricing{
		"deepseek-v4-pro": {
			InputCostPerToken:       1e-6,
			OutputCostPerToken:      2e-6,
			CacheReadInputTokenCost: 3e-8,
		},
		"deepseek-v4-flash": {
			InputCostPerToken:       1e-6,
			OutputCostPerToken:      2e-6,
			CacheReadInputTokenCost: 3e-8,
		},
		"deepseek-v4-flash-vision-exp": {
			InputCostPerToken:       1e-6,
			OutputCostPerToken:      2e-6,
			CacheReadInputTokenCost: 3e-8,
		},
	}}
	service := NewBillingService(&config.Config{}, pricingService)
	cases := []struct {
		model                    string
		input, output, cacheRead float64
	}{
		{"deepseek-v4-pro", deepseekProOffPeakInputPrice, deepseekProOffPeakOutputPrice, deepseekProOffPeakCacheRead},
		{"deepseek-v4-flash", deepseekFlashOffPeakInputPrice, deepseekFlashOffPeakOutputPrice, deepseekFlashOffPeakCacheRead},
		{"deepseek-v4-flash-vision-exp", deepseekFlashOffPeakInputPrice, deepseekFlashOffPeakOutputPrice, deepseekFlashOffPeakCacheRead},
	}
	for _, tc := range cases {
		t.Run(tc.model, func(t *testing.T) {
			pricing, err := service.GetModelPricing(tc.model)
			require.NoError(t, err)
			require.InDelta(t, tc.input, pricing.InputPricePerToken, 1e-15)
			require.InDelta(t, tc.output, pricing.OutputPricePerToken, 1e-15)
			require.InDelta(t, tc.cacheRead, pricing.CacheReadPricePerToken, 1e-15)
		})
	}
}

func TestGetModelPricing_DeepSeekThirdPartyExplicitPriceIsPreserved(t *testing.T) {
	// This model is present in the bundled catalog as a Volcengine entry with
	// an explicit zero/free token price.  The DeepSeek official card must only
	// apply to the allow-listed first-party IDs; a broad `deepseek-*` prefix
	// would silently turn this entry into a paid Flash card.
	pricingService := &PricingService{pricingData: map[string]*LiteLLMModelPricing{
		"deepseek-v3-2-251201": {
			InputCostPerToken:  0,
			OutputCostPerToken: 0,
			LiteLLMProvider:    "volcengine",
		},
	}}
	service := NewBillingService(&config.Config{}, pricingService)

	pricing, err := service.GetModelPricing("deepseek-v3-2-251201")
	require.NoError(t, err)
	require.NotNil(t, pricing)
	require.Zero(t, pricing.InputPricePerToken)
	require.Zero(t, pricing.OutputPricePerToken)

	// A non-zero third-party card must likewise retain its own amount and must
	// not receive the DeepSeek peak multiplier merely because of its prefix.
	pricingService.pricingData["deepseek-relay-custom"] = &LiteLLMModelPricing{
		InputCostPerToken:       1e-6,
		OutputCostPerToken:      2e-6,
		CacheReadInputTokenCost: 3e-8,
		LiteLLMProvider:         "volcengine",
	}
	offPeakAt := time.Date(2026, time.August, 24, 12, 0, 0, 0, time.UTC)
	peakAt := time.Date(2026, time.August, 24, 2, 0, 0, 0, time.UTC)
	tokens := UsageTokens{InputTokens: 100, OutputTokens: 50, CacheReadTokens: 20}
	offPeak, err := service.CalculateCostUnified(CostInput{
		Ctx: context.Background(), Model: "deepseek-relay-custom", Tokens: tokens,
		RateMultiplier: 1, PricingAt: offPeakAt,
	})
	require.NoError(t, err)
	peak, err := service.CalculateCostUnified(CostInput{
		Ctx: context.Background(), Model: "deepseek-relay-custom", Tokens: tokens,
		RateMultiplier: 1, PricingAt: peakAt,
	})
	require.NoError(t, err)
	require.InDelta(t, offPeak.TotalCost, peak.TotalCost, 1e-15)
}

func TestDeepSeekPricingFileIncludesV4MetadataAndLegacyAliases(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "resources", "model-pricing", "model_prices_and_context_window.json"))
	require.NoError(t, err)

	pricingService := &PricingService{}
	pricingData, err := pricingService.parsePricingData(data)
	require.NoError(t, err)

	for _, model := range []string{"deepseek-v4-flash", "deepseek-v4-flash-vision-exp", "deepseek-v4-pro"} {
		require.Contains(t, pricingData, model)
	}
	flash := pricingData["deepseek-v4-flash"]
	vision := pricingData["deepseek-v4-flash-vision-exp"]
	pro := pricingData["deepseek-v4-pro"]
	require.InDelta(t, deepseekFlashOffPeakInputPrice, flash.InputCostPerToken, 1e-15)
	require.InDelta(t, deepseekFlashOffPeakOutputPrice, flash.OutputCostPerToken, 1e-15)
	require.InDelta(t, deepseekFlashOffPeakCacheRead, flash.CacheReadInputTokenCost, 1e-15)
	require.True(t, vision.SupportsVision)
	require.InDelta(t, deepseekProOffPeakInputPrice, pro.InputCostPerToken, 1e-15)
	require.InDelta(t, deepseekProOffPeakOutputPrice, pro.OutputCostPerToken, 1e-15)
	require.InDelta(t, deepseekProOffPeakCacheRead, pro.CacheReadInputTokenCost, 1e-15)

	// Keep legacy aliases parseable for existing mappings while aligning their
	// token card with the current V4 Flash fallback.
	for _, alias := range []string{"deepseek-chat", "deepseek-reasoner"} {
		legacy := pricingData[alias]
		require.NotNil(t, legacy)
		require.InDelta(t, deepseekFlashOffPeakInputPrice, legacy.InputCostPerToken, 1e-15)
		require.InDelta(t, deepseekFlashOffPeakOutputPrice, legacy.OutputCostPerToken, 1e-15)
		require.InDelta(t, deepseekFlashOffPeakCacheRead, legacy.CacheReadInputTokenCost, 1e-15)
	}
}

func TestCalculateCostUnified_DeepSeekPeakMultiplierAndChannelOverride(t *testing.T) {
	service := newTestBillingService()
	resolver := NewModelPricingResolver(nil, service)
	tokens := UsageTokens{InputTokens: 1000, OutputTokens: 500, CacheReadTokens: 1000}
	offPeakAt := time.Date(2026, time.August, 24, 12, 0, 0, 0, time.UTC)
	peakAt := time.Date(2026, time.August, 24, 2, 0, 0, 0, time.UTC)
	offPeak, err := service.CalculateCostUnified(CostInput{
		Ctx: context.Background(), Model: "deepseek-v4-flash", Tokens: tokens,
		RateMultiplier: 1, Resolver: resolver, PricingAt: offPeakAt,
	})
	require.NoError(t, err)
	peak, err := service.CalculateCostUnified(CostInput{
		Ctx: context.Background(), Model: "deepseek-v4-flash", Tokens: tokens,
		RateMultiplier: 1, Resolver: resolver, PricingAt: peakAt,
	})
	require.NoError(t, err)
	require.InDelta(t, offPeak.TotalCost*2, peak.TotalCost, 1e-12)

	// A channel card is authoritative: the same peak timestamp must not scale
	// administrator-defined input/output/cache prices.
	inputPrice, outputPrice, cacheReadPrice := 1e-6, 2e-6, 3e-8
	cache := newEmptyChannelCache()
	cache.loadedAt = time.Now()
	cache.groupPlatform[1] = PlatformDeepseek
	cache.channelByGroupID[1] = &Channel{ID: 1, Status: StatusActive}
	cache.pricingByGroupModel[channelModelKey{
		groupID: 1, platform: PlatformDeepseek, model: "deepseek-v4-flash",
	}] = &ChannelModelPricing{
		Platform:       PlatformDeepseek,
		Models:         []string{"deepseek-v4-flash"},
		BillingMode:    BillingModeToken,
		InputPrice:     &inputPrice,
		OutputPrice:    &outputPrice,
		CacheReadPrice: &cacheReadPrice,
	}
	channelService := &ChannelService{}
	channelService.cache.Store(cache)
	channelResolver := NewModelPricingResolver(channelService, service)
	groupID := int64(1)
	resolved := channelResolver.Resolve(context.Background(), PricingInput{Model: "deepseek-v4-flash", GroupID: &groupID})
	require.Equal(t, PricingSourceChannel, resolved.Source)
	channelCost, err := service.CalculateCostUnified(CostInput{
		Ctx: context.Background(), Model: "deepseek-v4-flash", GroupID: &groupID,
		Tokens: tokens, RateMultiplier: 1, Resolver: channelResolver,
		Resolved: resolved, PricingAt: peakAt,
	})
	require.NoError(t, err)
	require.InDelta(t, 1000*inputPrice+500*outputPrice+1000*cacheReadPrice, channelCost.TotalCost, 1e-12)
}
