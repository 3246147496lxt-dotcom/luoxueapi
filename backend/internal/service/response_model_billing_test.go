//go:build unit

package service

import (
	"context"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

const (
	anthropicResponseBillingCheapModel  = "claude-sonnet-4"
	anthropicResponseBillingPriceyModel = "claude-opus-4.5"
	openAIResponseBillingCheapModel     = "gpt-5.4-nano"
	openAIResponseBillingPriceyModel    = "gpt-5.5"
)

func orderedResponseBillingModels(
	t *testing.T,
	billing *BillingService,
	tokens UsageTokens,
	a, b string,
) (cheaper, pricier string, cheaperCost, pricierCost *CostBreakdown) {
	t.Helper()
	costA, err := billing.CalculateCost(a, tokens, 1.1)
	require.NoError(t, err)
	costB, err := billing.CalculateCost(b, tokens, 1.1)
	require.NoError(t, err)
	require.NotEqual(t, costA.TotalCost, costB.TotalCost, "fixture prices must differ")
	require.True(t, billing.HasIdentifiedTokenPricing(a), "fixture %q must be deterministic", a)
	require.True(t, billing.HasIdentifiedTokenPricing(b), "fixture %q must be deterministic", b)
	if costA.TotalCost < costB.TotalCost {
		return a, b, costA, costB
	}
	return b, a, costB, costA
}

func TestResponseModelBillingDeclaration(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		source      string
		model       string
		conflict    bool
		mediaBilled bool
		want        string
	}{
		{name: "explicit_clean_opt_in", source: BillingModelSourceResponse, model: " gpt-5.4-nano ", want: "gpt-5.4-nano"},
		{name: "other_source", source: BillingModelSourceChannelMapped, model: "gpt-5.4-nano"},
		{name: "conflict", source: BillingModelSourceResponse, model: "gpt-5.4-nano", conflict: true},
		{name: "per_unit_media", source: BillingModelSourceResponse, model: "gpt-5.4-nano", mediaBilled: true},
		{name: "blank", source: BillingModelSourceResponse, model: "  "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, responseModelBillingDeclaration(tt.source, tt.model, tt.conflict, tt.mediaBilled))
		})
	}
}

func TestResponseModelBillingAdoptable(t *testing.T) {
	t.Parallel()
	cost := func(total float64) *CostBreakdown { return &CostBreakdown{TotalCost: total, ActualCost: total} }
	tests := []struct {
		name                  string
		baseline              *CostBreakdown
		response              *CostBreakdown
		baselineChannelPriced bool
		responseChannelPriced bool
		want                  bool
	}{
		{name: "cheaper", baseline: cost(1), response: cost(0.5), want: true},
		{name: "equal", baseline: cost(1), response: cost(1), want: true},
		{name: "epsilon", baseline: cost(1), response: cost(1 + 1e-13), want: true},
		{name: "pricier", baseline: cost(1), response: cost(1.001)},
		{name: "actual_cost_pricier", baseline: cost(1), response: &CostBreakdown{TotalCost: 0.5, ActualCost: 1.001}},
		{name: "positive_to_zero", baseline: cost(1), response: cost(0)},
		{name: "positive_to_negative", baseline: cost(1), response: cost(-1)},
		{name: "non_finite", baseline: cost(1), response: &CostBreakdown{TotalCost: math.NaN(), ActualCost: math.NaN()}},
		{name: "zero_stays_zero", baseline: cost(0), response: cost(0), want: true},
		{name: "channel_to_global", baseline: cost(1), response: cost(0.5), baselineChannelPriced: true},
		{name: "channel_to_channel", baseline: cost(1), response: cost(0.5), baselineChannelPriced: true, responseChannelPriced: true, want: true},
		{name: "global_to_channel", baseline: cost(1), response: cost(0.5), responseChannelPriced: true, want: true},
		{name: "nil_baseline", response: cost(0.5)},
		{name: "nil_response", baseline: cost(1)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, responseModelBillingAdoptable(
				tt.baseline, tt.response, tt.baselineChannelPriced, tt.responseChannelPriced,
			))
		})
	}
}

func TestBillingServiceHasIdentifiedTokenPricingRejectsFamilyGuesses(t *testing.T) {
	t.Parallel()
	billing := newGatewayRecordUsageServiceForTest(
		&openAIRecordUsageLogRepoStub{}, &openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{},
	).billingService

	require.True(t, billing.HasIdentifiedTokenPricing("claude-sonnet-4"))
	require.True(t, billing.HasIdentifiedTokenPricing("  CLAUDE-SONNET-4  "))
	require.True(t, billing.HasIdentifiedTokenPricing("gpt-5.4-nano"))

	const forged = "totally-made-up-haiku-v9"
	_, err := billing.GetModelPricing(forged)
	require.NoError(t, err, "wide lookup should demonstrate the unsafe family fallback")
	require.False(t, billing.HasIdentifiedTokenPricing(forged))
	require.False(t, billing.HasIdentifiedTokenPricing("zz-unpriced-response-model"))
	require.False(t, billing.HasIdentifiedTokenPricing(""))
}

func TestPricingServiceGetIdentifiedModelPricingUsesOnlyDeterministicMatches(t *testing.T) {
	t.Parallel()
	base := &LiteLLMModelPricing{InputCostPerToken: 1e-6, OutputCostPerToken: 2e-6}
	dated := &LiteLLMModelPricing{InputCostPerToken: 3e-6, OutputCostPerToken: 4e-6}
	imageOnly := &LiteLLMModelPricing{OutputCostPerImage: 0.04, TokenPricingAbsent: true}
	pricing := &PricingService{pricingData: map[string]*LiteLLMModelPricing{
		"claude-opus-4.5":          base,
		"claude-opus-4.5-20251101": dated,
		"image-only":               imageOnly,
	}}

	require.Same(t, base, pricing.GetIdentifiedModelPricing("claude-opus-4.5"))
	require.Same(t, base, pricing.GetIdentifiedModelPricing("claude-opus-4-5"), "hyphenated base spelling remains deterministic")
	require.Same(t, dated, pricing.GetIdentifiedModelPricing("claude-opus-4-5-20251101"), "known spelling variants remain deterministic")
	require.Same(t, base, pricing.GetIdentifiedModelPricing("claude-opus-4.5-20261212"), "an unknown date must prefer the explicit base price")
	require.Same(t, base, pricing.GetIdentifiedModelPricing("claude-opus-4-5-20261212"), "a hyphenated unknown date must prefer the explicit base price")
	require.Nil(t, pricing.GetIdentifiedModelPricing("totally-made-up-haiku-v9"))

	ambiguous := &PricingService{pricingData: map[string]*LiteLLMModelPricing{
		"claude-opus-4.5-20251101": dated,
		"claude-opus-4.5-20260101": base,
	}}
	require.Nil(t, ambiguous.GetIdentifiedModelPricing("claude-opus-4.5-20261212"), "multiple dated prices without a base are ambiguous")

	billing := NewBillingService(&config.Config{}, pricing)
	require.True(t, billing.HasIdentifiedTokenPricing("claude-opus-4.5"))
	require.False(t, billing.HasIdentifiedTokenPricing("image-only"), "an image-only catalog row is not token pricing")
}

func TestGatewayRecordUsageResponseModelSafeDowngrade(t *testing.T) {
	usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
	userRepo := &openAIRecordUsageUserRepoStub{}
	svc := newGatewayRecordUsageServiceForTest(usageRepo, userRepo, &openAIRecordUsageSubRepoStub{})
	tokens := UsageTokens{InputTokens: 100, OutputTokens: 50}
	cheaper, pricier, cheaperCost, _ := orderedResponseBillingModels(
		t, svc.billingService, tokens, anthropicResponseBillingCheapModel, anthropicResponseBillingPriceyModel,
	)

	err := svc.RecordUsage(context.Background(), &RecordUsageInput{
		Result: &ForwardResult{
			RequestID:             "gateway-response-model-downgrade",
			Usage:                 ClaudeUsage{InputTokens: 100, OutputTokens: 50},
			Model:                 pricier,
			UpstreamResponseModel: cheaper,
			Duration:              time.Second,
		},
		APIKey:  &APIKey{ID: 501, Quota: 100},
		User:    &User{ID: 601},
		Account: &Account{ID: 701, Platform: PlatformAnthropic},
		ChannelUsageFields: ChannelUsageFields{
			ChannelID:          9,
			OriginalModel:      pricier,
			ChannelMappedModel: pricier,
			BillingModelSource: BillingModelSourceResponse,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, usageRepo.lastLog)
	require.InDelta(t, cheaperCost.ActualCost, usageRepo.lastLog.ActualCost, 1e-12)
	require.InDelta(t, cheaperCost.ActualCost, userRepo.lastAmount, 1e-12)
	require.Positive(t, usageRepo.lastLog.ActualCost)
	require.Equal(t, pricier, usageRepo.lastLog.Model)
	require.NotNil(t, usageRepo.lastLog.UpstreamResponseModel)
	require.Equal(t, cheaper, *usageRepo.lastLog.UpstreamResponseModel)
	require.NotNil(t, usageRepo.lastLog.UpstreamModelMismatch)
	require.True(t, *usageRepo.lastLog.UpstreamModelMismatch)
}

func TestGatewayRecordUsageResponseModelUnpricedAliasUsesConcreteBaseline(t *testing.T) {
	usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
	userRepo := &openAIRecordUsageUserRepoStub{}
	svc := newGatewayRecordUsageServiceForTest(usageRepo, userRepo, &openAIRecordUsageSubRepoStub{})
	tokens := UsageTokens{InputTokens: 100, OutputTokens: 50}
	cheaper, pricier, cheaperCost, _ := orderedResponseBillingModels(
		t, svc.billingService, tokens, anthropicResponseBillingCheapModel, anthropicResponseBillingPriceyModel,
	)
	const publicAlias = "public-alias-without-pricing"

	err := svc.RecordUsage(context.Background(), &RecordUsageInput{
		Result: &ForwardResult{
			RequestID:             "gateway-response-model-concrete-baseline",
			Usage:                 ClaudeUsage{InputTokens: 100, OutputTokens: 50},
			Model:                 publicAlias,
			UpstreamModel:         pricier,
			UpstreamResponseModel: cheaper,
			Duration:              time.Second,
		},
		APIKey:  &APIKey{ID: 502, Quota: 100},
		User:    &User{ID: 602},
		Account: &Account{ID: 702, Platform: PlatformAnthropic},
		ChannelUsageFields: ChannelUsageFields{
			ChannelID:          10,
			OriginalModel:      publicAlias,
			ChannelMappedModel: publicAlias,
			BillingModelSource: BillingModelSourceResponse,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, usageRepo.lastLog)
	require.InDelta(t, cheaperCost.ActualCost, usageRepo.lastLog.ActualCost, 1e-12)
	require.InDelta(t, cheaperCost.ActualCost, userRepo.lastAmount, 1e-12)
	require.Positive(t, usageRepo.lastLog.ActualCost, "an unpriced public alias must not create a free baseline")
}

func TestGatewayRecordUsageResponseModelFallbacks(t *testing.T) {
	tests := []struct {
		name     string
		response string
		conflict bool
		source   string
	}{
		{name: "pricier", response: anthropicResponseBillingPriceyModel, source: BillingModelSourceResponse},
		{name: "conflict", response: anthropicResponseBillingCheapModel, conflict: true, source: BillingModelSourceResponse},
		{name: "unpriced", response: "zz-unpriced-response-model", source: BillingModelSourceResponse},
		{name: "not_opted_in", response: anthropicResponseBillingCheapModel, source: BillingModelSourceChannelMapped},
		{name: "family_guess", response: "totally-made-up-haiku-v9", source: BillingModelSourceResponse},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
			userRepo := &openAIRecordUsageUserRepoStub{}
			svc := newGatewayRecordUsageServiceForTest(usageRepo, userRepo, &openAIRecordUsageSubRepoStub{})
			tokens := UsageTokens{InputTokens: 100, OutputTokens: 50}
			cheaper, pricier, _, pricierCost := orderedResponseBillingModels(
				t, svc.billingService, tokens, anthropicResponseBillingCheapModel, anthropicResponseBillingPriceyModel,
			)
			baseline := pricier
			response := tt.response
			if tt.name == "pricier" {
				baseline = cheaper
				response = pricier
				pricierCost, _ = svc.billingService.CalculateCost(cheaper, tokens, 1.1)
			}

			err := svc.RecordUsage(context.Background(), &RecordUsageInput{
				Result: &ForwardResult{
					RequestID:                     "gateway-response-model-fallback-" + tt.name,
					Usage:                         ClaudeUsage{InputTokens: 100, OutputTokens: 50},
					Model:                         baseline,
					UpstreamResponseModel:         response,
					UpstreamResponseModelConflict: tt.conflict,
					Duration:                      time.Second,
				},
				APIKey:  &APIKey{ID: 501, Quota: 100},
				User:    &User{ID: 601},
				Account: &Account{ID: 701, Platform: PlatformAnthropic},
				ChannelUsageFields: ChannelUsageFields{
					ChannelID:          9,
					OriginalModel:      baseline,
					ChannelMappedModel: baseline,
					BillingModelSource: tt.source,
				},
			})

			require.NoError(t, err)
			require.NotNil(t, usageRepo.lastLog)
			require.InDelta(t, pricierCost.ActualCost, usageRepo.lastLog.ActualCost, 1e-12)
			require.InDelta(t, pricierCost.ActualCost, userRepo.lastAmount, 1e-12)
		})
	}
}

func TestOpenAIRecordUsageResponseModelSafeDowngrade(t *testing.T) {
	usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
	userRepo := &openAIRecordUsageUserRepoStub{}
	svc := newOpenAIRecordUsageServiceForTest(usageRepo, userRepo, &openAIRecordUsageSubRepoStub{}, nil)
	tokens := UsageTokens{InputTokens: 20, OutputTokens: 10}
	cheaper, pricier, cheaperCost, _ := orderedResponseBillingModels(
		t, svc.billingService, tokens, openAIResponseBillingCheapModel, openAIResponseBillingPriceyModel,
	)

	err := svc.RecordUsage(context.Background(), &OpenAIRecordUsageInput{
		Result: &OpenAIForwardResult{
			RequestID:             "openai-response-model-downgrade",
			Model:                 pricier,
			UpstreamModel:         pricier,
			UpstreamResponseModel: cheaper,
			Usage:                 OpenAIUsage{InputTokens: 20, OutputTokens: 10},
			Duration:              time.Second,
		},
		APIKey:  &APIKey{ID: 10},
		User:    &User{ID: 20},
		Account: &Account{ID: 30, Platform: PlatformOpenAI},
		ChannelUsageFields: ChannelUsageFields{
			ChannelID:          9,
			OriginalModel:      pricier,
			ChannelMappedModel: pricier,
			BillingModelSource: BillingModelSourceResponse,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, usageRepo.lastLog)
	require.InDelta(t, cheaperCost.ActualCost, usageRepo.lastLog.ActualCost, 1e-12)
	require.InDelta(t, cheaperCost.ActualCost, userRepo.lastAmount, 1e-12)
	require.Positive(t, usageRepo.lastLog.ActualCost)
	require.Equal(t, pricier, usageRepo.lastLog.Model)
	require.NotNil(t, usageRepo.lastLog.UpstreamResponseModel)
	require.Equal(t, cheaper, *usageRepo.lastLog.UpstreamResponseModel)
	require.NotNil(t, usageRepo.lastLog.UpstreamModelMismatch)
	require.True(t, *usageRepo.lastLog.UpstreamModelMismatch)
}

func TestOpenAIRecordUsageResponseModelCannotBypassFallbackChannelPricing(t *testing.T) {
	usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
	userRepo := &openAIRecordUsageUserRepoStub{}
	svc := newOpenAIRecordUsageServiceForTest(usageRepo, userRepo, &openAIRecordUsageSubRepoStub{}, nil)
	groupID := int64(91)
	svc.resolver = newOpenAITokenImageChannelPricingResolverForTest(t, groupID, openAIResponseBillingPriceyModel)
	apiKey := &APIKey{
		ID:      11,
		GroupID: &groupID,
		Group: &Group{
			ID:             groupID,
			Hydrated:       true,
			Platform:       PlatformOpenAI,
			RateMultiplier: 1.1,
		},
	}
	tokens := UsageTokens{InputTokens: 20, OutputTokens: 10}
	expected, err := svc.calculateOpenAIRecordUsageTokenCost(
		context.Background(), apiKey, openAIResponseBillingPriceyModel, 1.1, tokens, "", false,
	)
	require.NoError(t, err)
	require.Positive(t, expected.ActualCost)
	const publicAlias = "unpriced-openai-public-alias"

	err = svc.RecordUsage(context.Background(), &OpenAIRecordUsageInput{
		Result: &OpenAIForwardResult{
			RequestID:             "openai-response-model-channel-source-guard",
			Model:                 publicAlias,
			BillingModel:          publicAlias,
			UpstreamModel:         openAIResponseBillingPriceyModel,
			UpstreamResponseModel: openAIResponseBillingCheapModel,
			Usage:                 OpenAIUsage{InputTokens: 20, OutputTokens: 10},
			Duration:              time.Second,
		},
		APIKey:  apiKey,
		User:    &User{ID: 21},
		Account: &Account{ID: 31, Platform: PlatformOpenAI},
		ChannelUsageFields: ChannelUsageFields{
			ChannelID:          10,
			OriginalModel:      publicAlias,
			BillingModelSource: BillingModelSourceResponse,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, usageRepo.lastLog)
	require.InDelta(t, expected.ActualCost, usageRepo.lastLog.ActualCost, 1e-12)
	require.InDelta(t, expected.ActualCost, userRepo.lastAmount, 1e-12)
}

func newMutableResponseBillingResolver(billing *BillingService) *ModelPricingResolver {
	channelService := &ChannelService{}
	storeResponseBillingChannelPrice(channelService, 0, "", 0, 0)
	return NewModelPricingResolver(channelService, billing)
}

func storeResponseBillingChannelPrice(channelService *ChannelService, groupID int64, model string, inputPrice, outputPrice float64) {
	cache := newEmptyChannelCache()
	if groupID > 0 {
		cache.channelByGroupID[groupID] = &Channel{ID: groupID, Status: StatusActive}
		cache.groupPlatform[groupID] = ""
	}
	if strings.TrimSpace(model) != "" {
		cache.pricingByGroupModel[channelModelKey{groupID: groupID, model: strings.ToLower(strings.TrimSpace(model))}] = &ChannelModelPricing{
			BillingMode: BillingModeToken,
			InputPrice:  &inputPrice,
			OutputPrice: &outputPrice,
		}
	}
	cache.loadedAt = time.Now()
	channelService.cache.Store(cache)
}

func TestOpenAIResponseBillingBaselineChannelThenRemovedStillBlocksGlobal(t *testing.T) {
	svc := newOpenAIRecordUsageServiceForTest(
		&openAIRecordUsageLogRepoStub{}, &openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{}, nil,
	)
	groupID := int64(92)
	svc.resolver = newMutableResponseBillingResolver(svc.billingService)
	storeResponseBillingChannelPrice(svc.resolver.channelService, groupID, openAIResponseBillingPriceyModel, 9e-6, 45e-6)
	apiKey := &APIKey{GroupID: &groupID, Group: &Group{ID: groupID}}
	tokens := UsageTokens{InputTokens: 20, OutputTokens: 10}

	baseline := svc.calculateOpenAIRecordUsageCostResolved(
		context.Background(), &OpenAIForwardResult{}, apiKey,
		[]string{openAIResponseBillingPriceyModel}, 1, 1, 1, 1, tokens, "", false,
	)
	require.NoError(t, baseline.Err)
	require.True(t, baseline.channelPriced())

	storeResponseBillingChannelPrice(svc.resolver.channelService, groupID, "", 0, 0)
	response := svc.calculateOpenAIRecordUsageCostResolved(
		context.Background(), &OpenAIForwardResult{}, apiKey,
		[]string{openAIResponseBillingCheapModel}, 1, 1, 1, 1, tokens, "", false,
	)
	require.NoError(t, response.Err)
	require.False(t, response.channelPriced())
	require.True(t, response.Identified)
	require.Less(t, response.Cost.ActualCost, baseline.Cost.ActualCost, "fixture must exercise the cross-source guard, not the higher-cost guard")
	require.False(t, responseModelBillingAdoptable(
		baseline.Cost, response.Cost, baseline.channelPriced(), response.channelPriced(),
	))
}

func TestGatewayResponseBillingCapturedChannelResponseSurvivesRemoval(t *testing.T) {
	svc := newGatewayRecordUsageServiceForTest(
		&openAIRecordUsageLogRepoStub{}, &openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{},
	)
	groupID := int64(93)
	svc.resolver = newMutableResponseBillingResolver(svc.billingService)
	storeResponseBillingChannelPrice(svc.resolver.channelService, groupID, anthropicResponseBillingCheapModel, 8e-6, 40e-6)
	apiKey := &APIKey{GroupID: &groupID, Group: &Group{ID: groupID}}
	result := &ForwardResult{Usage: ClaudeUsage{InputTokens: 100, OutputTokens: 50}}
	opts := &recordUsageOpts{}

	captured := svc.calculateResponseBillingTokenCost(
		context.Background(), result, apiKey, anthropicResponseBillingCheapModel, 1, opts,
	)
	require.NoError(t, captured.Err)
	require.True(t, captured.channelPriced())
	require.True(t, captured.Identified)
	capturedCost := captured.Cost.ActualCost

	storeResponseBillingChannelPrice(svc.resolver.channelService, groupID, "", 0, 0)
	afterRemoval := svc.calculateResponseBillingTokenCost(
		context.Background(), result, apiKey, anthropicResponseBillingCheapModel, 1, opts,
	)
	require.NoError(t, afterRemoval.Err)
	require.False(t, afterRemoval.channelPriced())
	require.NotEqual(t, afterRemoval.Cost.ActualCost, capturedCost)
	require.Equal(t, capturedCost, captured.Cost.ActualCost, "captured channel cost must remain immutable after cache replacement")
}

func TestOpenAIResponseBillingCapturedGlobalIgnoresNewChannel(t *testing.T) {
	svc := newOpenAIRecordUsageServiceForTest(
		&openAIRecordUsageLogRepoStub{}, &openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{}, nil,
	)
	groupID := int64(94)
	svc.resolver = newMutableResponseBillingResolver(svc.billingService)
	storeResponseBillingChannelPrice(svc.resolver.channelService, groupID, "", 0, 0)
	apiKey := &APIKey{GroupID: &groupID, Group: &Group{ID: groupID}}
	tokens := UsageTokens{InputTokens: 20, OutputTokens: 10}

	captured := svc.calculateOpenAIRecordUsageCostResolved(
		context.Background(), &OpenAIForwardResult{}, apiKey,
		[]string{openAIResponseBillingCheapModel}, 1, 1, 1, 1, tokens, "", false,
	)
	require.NoError(t, captured.Err)
	require.False(t, captured.channelPriced())
	require.True(t, captured.Identified)

	storeResponseBillingChannelPrice(svc.resolver.channelService, groupID, openAIResponseBillingCheapModel, 10e-6, 50e-6)
	newChannelBaseline := svc.calculateOpenAIRecordUsageCostResolved(
		context.Background(), &OpenAIForwardResult{}, apiKey,
		[]string{openAIResponseBillingCheapModel}, 1, 1, 1, 1, tokens, "", false,
	)
	require.NoError(t, newChannelBaseline.Err)
	require.True(t, newChannelBaseline.channelPriced())
	require.Less(t, captured.Cost.ActualCost, newChannelBaseline.Cost.ActualCost)
	require.False(t, responseModelBillingAdoptable(
		newChannelBaseline.Cost, captured.Cost, newChannelBaseline.channelPriced(), captured.channelPriced(),
	))
}

func TestOpenAIRecordUsageResponseModelRejectsPricierAndConflict(t *testing.T) {
	tests := []struct {
		name     string
		conflict bool
	}{
		{name: "pricier"},
		{name: "conflict", conflict: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
			userRepo := &openAIRecordUsageUserRepoStub{}
			svc := newOpenAIRecordUsageServiceForTest(usageRepo, userRepo, &openAIRecordUsageSubRepoStub{}, nil)
			tokens := UsageTokens{InputTokens: 20, OutputTokens: 10}
			cheaper, pricier, cheaperCost, pricierCost := orderedResponseBillingModels(
				t, svc.billingService, tokens, openAIResponseBillingCheapModel, openAIResponseBillingPriceyModel,
			)
			baseline, response, want := cheaper, pricier, cheaperCost
			if tt.conflict {
				baseline, response, want = pricier, cheaper, pricierCost
			}

			err := svc.RecordUsage(context.Background(), &OpenAIRecordUsageInput{
				Result: &OpenAIForwardResult{
					RequestID:                     "openai-response-model-reject-" + tt.name,
					Model:                         baseline,
					UpstreamModel:                 baseline,
					UpstreamResponseModel:         response,
					UpstreamResponseModelConflict: tt.conflict,
					Usage:                         OpenAIUsage{InputTokens: 20, OutputTokens: 10},
					Duration:                      time.Second,
				},
				APIKey:  &APIKey{ID: 10},
				User:    &User{ID: 20},
				Account: &Account{ID: 30, Platform: PlatformOpenAI},
				ChannelUsageFields: ChannelUsageFields{
					ChannelID:          9,
					OriginalModel:      baseline,
					ChannelMappedModel: baseline,
					BillingModelSource: BillingModelSourceResponse,
				},
			})

			require.NoError(t, err)
			require.NotNil(t, usageRepo.lastLog)
			require.InDelta(t, want.ActualCost, usageRepo.lastLog.ActualCost, 1e-12)
			require.InDelta(t, want.ActualCost, userRepo.lastAmount, 1e-12)
		})
	}
}

func TestToUsageFieldsResponseModelSourcePassesThrough(t *testing.T) {
	fields := (ChannelMappingResult{
		MappedModel:        "claude-sonnet-4",
		ChannelID:          4,
		BillingModelSource: BillingModelSourceResponse,
	}).ToUsageFields("claude-sonnet-4", "claude-sonnet-4")
	require.Equal(t, int64(4), fields.ChannelID)
	require.Equal(t, BillingModelSourceResponse, fields.BillingModelSource)
}

type responseModelChatPricingSpy struct {
	model string
}

func (s *responseModelChatPricingSpy) Resolve(_ context.Context, input PricingInput) *ResolvedPricing {
	s.model = input.Model
	return &ResolvedPricing{
		Mode:   BillingModeToken,
		Source: PricingSourceFallback,
		BasePricing: &ModelPricing{
			InputPricePerToken:  1e-6,
			OutputPricePerToken: 2e-6,
		},
	}
}

func (s *responseModelChatPricingSpy) GetIntervalPricing(resolved *ResolvedPricing, _ int) *ModelPricing {
	return resolved.BasePricing
}

func TestChatResponseModelSourcePreviewsMappedBaseline(t *testing.T) {
	svc, _, catalog, _, _, _ := newChatServiceBehaviorTest()
	catalog.models = []string{"gpt-5.5"}
	catalog.mapping = ChannelMappingResult{
		MappedModel:        "gpt-5.4",
		Mapped:             true,
		BillingModelSource: BillingModelSourceResponse,
	}
	pricing := &responseModelChatPricingSpy{}
	svc.pricing = pricing

	result, err := svc.ListModels(context.Background(), 7)
	require.NoError(t, err)
	require.Len(t, result.Models, 1)
	require.Equal(t, "gpt-5.4", pricing.model)
}
