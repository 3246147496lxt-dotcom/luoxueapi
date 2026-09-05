//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func deepSeekGatewayForwardResultForTest() *ForwardResult {
	return &ForwardResult{
		Model: "deepseek-v4-flash",
		Usage: ClaudeUsage{
			InputTokens:          1000,
			OutputTokens:         500,
			CacheReadInputTokens: 1000,
		},
	}
}

func TestGatewayServiceDeepSeekPeakPricingResolverAndFallbackPaths(t *testing.T) {
	offPeakAt := time.Date(2026, time.August, 24, 12, 0, 0, 0, time.UTC)
	peakAt := time.Date(2026, time.August, 24, 2, 0, 0, 0, time.UTC)
	const offPeakTotal = 1000*deepseekFlashOffPeakInputPrice +
		500*deepseekFlashOffPeakOutputPrice + 1000*deepseekFlashOffPeakCacheRead

	cases := []struct {
		name         string
		withResolver bool
	}{
		{name: "resolver", withResolver: true},
		{name: "no resolver fallback", withResolver: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			billing := newTestBillingService()
			var resolver *ModelPricingResolver
			var apiKey *APIKey
			if tc.withResolver {
				// Resolver.Resolve probes channel overrides whenever a group ID is
				// supplied; install an empty cache so this fallback case exercises
				// the real resolver path without a nil ChannelService panic.
				channelService := &ChannelService{}
				channelCache := newEmptyChannelCache()
				channelCache.loadedAt = time.Now()
				channelService.cache.Store(channelCache)
				resolver = NewModelPricingResolver(channelService, billing)
				groupID := int64(1)
				apiKey = &APIKey{
					GroupID: &groupID,
					Group:   &Group{ID: groupID, Platform: PlatformDeepseek},
				}
			}
			svc := &GatewayService{billingService: billing, resolver: resolver}

			offPeak := svc.calculateTokenCostResolved(
				context.Background(), deepSeekGatewayForwardResultForTest(), apiKey,
				"deepseek-v4-flash", 1, &recordUsageOpts{PricingAt: offPeakAt},
			)
			require.NoError(t, offPeak.Err)
			require.InDelta(t, offPeakTotal, offPeak.Cost.TotalCost, 1e-12)

			peak := svc.calculateTokenCostResolved(
				context.Background(), deepSeekGatewayForwardResultForTest(), apiKey,
				"deepseek-v4-flash", 1, &recordUsageOpts{PricingAt: peakAt},
			)
			require.NoError(t, peak.Err)
			require.InDelta(t, offPeakTotal*2, peak.Cost.TotalCost, 1e-12)
		})
	}
}

func TestGatewayServiceDeepSeekPeakPricingLeavesChannelPriceUnscaled(t *testing.T) {
	groupID := int64(7)
	inputPrice, outputPrice, cacheReadPrice := 1e-6, 2e-6, 3e-8
	cache := newEmptyChannelCache()
	cache.loadedAt = time.Now()
	cache.groupPlatform[groupID] = PlatformDeepseek
	cache.channelByGroupID[groupID] = &Channel{ID: groupID, Status: StatusActive}
	cache.pricingByGroupModel[channelModelKey{
		groupID:  groupID,
		platform: PlatformDeepseek,
		model:    "deepseek-v4-flash",
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
	billing := newTestBillingService()
	svc := &GatewayService{
		billingService: billing,
		resolver:       NewModelPricingResolver(channelService, billing),
	}
	apiKey := &APIKey{GroupID: &groupID, Group: &Group{ID: groupID, Platform: PlatformDeepseek}}

	result := svc.calculateTokenCostResolved(
		context.Background(), deepSeekGatewayForwardResultForTest(), apiKey,
		"deepseek-v4-flash", 1, &recordUsageOpts{
			PricingAt: time.Date(2026, time.August, 24, 2, 0, 0, 0, time.UTC),
		},
	)
	require.NoError(t, result.Err)
	require.Equal(t, PricingSourceChannel, result.PricingSource)
	require.InDelta(t, 1000*inputPrice+500*outputPrice+1000*cacheReadPrice, result.Cost.TotalCost, 1e-12)
}

func TestGatewayServiceDeepSeekPeakPricingLegacyTokenPath(t *testing.T) {
	billing := newTestBillingService()
	svc := &GatewayService{billingService: billing}
	offPeakAt := time.Date(2026, time.August, 24, 12, 0, 0, 0, time.UTC)
	peakAt := time.Date(2026, time.August, 24, 2, 0, 0, 0, time.UTC)

	offPeak := svc.calculateTokenCost(
		context.Background(), deepSeekGatewayForwardResultForTest(), nil,
		"deepseek-v4-flash", 1, &recordUsageOpts{PricingAt: offPeakAt},
	)
	peak := svc.calculateTokenCost(
		context.Background(), deepSeekGatewayForwardResultForTest(), nil,
		"deepseek-v4-flash", 1, &recordUsageOpts{PricingAt: peakAt},
	)
	require.InDelta(t, offPeak.TotalCost*2, peak.TotalCost, 1e-12)
}
