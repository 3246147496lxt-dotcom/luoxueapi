package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type modelCatalogRepoTestStub struct {
	ModelCatalogRepository
	models      []ModelCatalogModel
	model       *ModelCatalogModel
	updateCalls int
}

func (s *modelCatalogRepoTestStub) ListPublished(context.Context) ([]ModelCatalogModel, error) {
	return append([]ModelCatalogModel(nil), s.models...), nil
}

func (s *modelCatalogRepoTestStub) GetByID(context.Context, int64) (*ModelCatalogModel, error) {
	copy := *s.model
	return &copy, nil
}

func (s *modelCatalogRepoTestStub) Update(_ context.Context, model *ModelCatalogModel) error {
	s.updateCalls++
	copy := *model
	s.model = &copy
	return nil
}

type modelCatalogChannelRepoTestStub struct {
	ChannelRepository
	channels []Channel
}

func (s *modelCatalogChannelRepoTestStub) ListAll(context.Context) ([]Channel, error) {
	return append([]Channel(nil), s.channels...), nil
}

type modelCatalogGroupRepoTestStub struct {
	GroupRepository
	group *Group
}

func (s *modelCatalogGroupRepoTestStub) GetByIDLite(context.Context, int64) (*Group, error) {
	copy := *s.group
	return &copy, nil
}

func (s *modelCatalogGroupRepoTestStub) ListActive(context.Context) ([]Group, error) {
	copy := *s.group
	return []Group{copy}, nil
}

func TestSynthesizeCatalogPricingPreservesExplicitZero(t *testing.T) {
	pricing := synthesizeCatalogPricingFromLiteLLM(&LiteLLMModelPricing{
		InputCostPerToken:     0,
		InputCostPerTokenSet:  true,
		OutputCostPerToken:    0,
		OutputCostPerTokenSet: false,
	}, nil)
	require.NotNil(t, pricing.InputPrice)
	require.Zero(t, *pricing.InputPrice)
	require.Nil(t, pricing.OutputPrice)
}

func TestCatalogPricingLabelsEffectiveChargeAsCreditWithoutChangingRawPrice(t *testing.T) {
	rawInputPrice := 5.0
	pricing := &ChannelModelPricing{
		BillingMode: BillingModeToken,
		InputPrice:  &rawInputPrice,
	}

	public := catalogPricingFromChannel(pricing, &Group{RateMultiplier: 70})

	require.NotNil(t, public)
	require.Equal(t, "CREDIT", public.Currency)
	require.InDelta(t, 350, *public.InputPrice, 1e-12)
	require.InDelta(t, 5, *pricing.InputPrice, 1e-12, "catalog projection must not mutate the channel source price")
}

func TestCatalogGlobalImageTokenZeroMirrorsBillingTextFallback(t *testing.T) {
	pricingService := &PricingService{pricingData: map[string]*LiteLLMModelPricing{
		"image-token-zero": {
			InputCostPerToken: 2, InputCostPerTokenSet: true,
			OutputCostPerToken: 3, OutputCostPerTokenSet: true,
			InputCostPerImageToken: 0, InputCostPerImageTokenSet: true,
			OutputCostPerImageToken: 0, OutputCostPerImageTokenSet: true,
		},
	}}
	catalogService := &ModelCatalogService{pricingService: pricingService}
	resolved := catalogService.resolveCatalogChannelPricing("image-token-zero", nil)
	require.NotNil(t, resolved)
	require.InDelta(t, 2, *resolved.ImageInputPrice, 1e-12)
	require.InDelta(t, 3, *resolved.ImageOutputPrice, 1e-12)

	billingService := &BillingService{pricingService: pricingService}
	cost, err := billingService.CalculateCost("image-token-zero", UsageTokens{
		InputTokens: 1, ImageInputTokens: 1, OutputTokens: 1, ImageOutputTokens: 1,
	}, 1)
	require.NoError(t, err)
	require.InDelta(t, *resolved.ImageInputPrice, cost.ImageInputCost, 1e-12)
	require.InDelta(t, *resolved.ImageOutputPrice, cost.ImageOutputCost, 1e-12)
}

func TestCatalogBillingSourceUsesActualBillableModel(t *testing.T) {
	requestedPrice, mappedPrice := 1.0, 2.0
	channel := &Channel{
		BillingModelSource: BillingModelSourceChannelMapped,
		ModelMapping:       map[string]map[string]string{PlatformOpenAI: {"alias-*": "target-model"}},
		ModelPricing: []ChannelModelPricing{
			{Platform: PlatformOpenAI, Models: []string{"alias-model"}, InputPrice: &requestedPrice},
			{Platform: PlatformOpenAI, Models: []string{"target-model"}, InputPrice: &mappedPrice},
		},
	}
	supported := SupportedModel{Name: "alias-model", Platform: PlatformOpenAI, Pricing: &channel.ModelPricing[0]}
	model, pricing, ok := catalogBillingModelAndPricing(channel, supported)
	require.True(t, ok)
	require.Equal(t, "target-model", model)
	require.Equal(t, &mappedPrice, pricing.InputPrice)

	channel.BillingModelSource = BillingModelSourceResponse
	model, pricing, ok = catalogBillingModelAndPricing(channel, supported)
	require.True(t, ok, "response_model has a deterministic mapped upper-bound price")
	require.Equal(t, "target-model", model)
	require.Equal(t, &mappedPrice, pricing.InputPrice)

	channel.BillingModelSource = BillingModelSourceRequested
	model, pricing, ok = catalogBillingModelAndPricing(channel, supported)
	require.True(t, ok)
	require.Equal(t, "alias-model", model)
	require.Equal(t, &requestedPrice, pricing.InputPrice)

	channel.BillingModelSource = BillingModelSourceUpstream
	_, _, ok = catalogBillingModelAndPricing(channel, supported)
	require.False(t, ok)
}

func TestCatalogRejectsGrokVideoWithoutVideoPricingSchema(t *testing.T) {
	price := 0.10
	channel := &Channel{
		BillingModelSource: BillingModelSourceRequested,
		ModelPricing: []ChannelModelPricing{{
			Platform: PlatformOpenAI, Models: []string{"grok-imagine-video-1.5"},
			BillingMode: BillingModePerRequest, PerRequestPrice: &price,
		}},
	}
	supported := SupportedModel{
		Name: "grok-imagine-video-1.5", Platform: PlatformOpenAI, Pricing: &channel.ModelPricing[0],
	}
	_, _, ok := catalogBillingModelAndPricing(channel, supported)
	require.False(t, ok, "video pricing varies by resolution and duration and has no public DTO yet")

	aliasPrice := 0.20
	channel.ModelMapping = map[string]map[string]string{
		PlatformOpenAI: {"custom-video-alias": "grok-imagine-video-1.5"},
	}
	channel.ModelPricing = []ChannelModelPricing{{
		Platform: PlatformOpenAI, Models: []string{"custom-video-alias"},
		BillingMode: BillingModePerRequest, PerRequestPrice: &aliasPrice,
	}}
	supported = SupportedModel{
		Name: "custom-video-alias", Platform: PlatformOpenAI, Pricing: &channel.ModelPricing[0],
	}
	_, _, ok = catalogBillingModelAndPricing(channel, supported)
	require.False(t, ok, "requested pricing identity must still reject a deterministic Grok video mapping target")
}

func TestCatalogRestrictedMappedModelRequiresChannelPricing(t *testing.T) {
	pricingService := &PricingService{pricingData: map[string]*LiteLLMModelPricing{
		"target-model": {InputCostPerToken: 1, InputCostPerTokenSet: true},
	}}
	service := &ModelCatalogService{pricingService: pricingService}
	model := &ModelCatalogModel{Model: "public-alias", Platform: PlatformOpenAI}
	group := &Group{
		ID: 7, Platform: PlatformOpenAI, Status: StatusActive,
		SubscriptionType: SubscriptionTypeStandard, RateMultiplier: 1,
	}
	channel := Channel{
		Status: StatusActive, GroupIDs: []int64{group.ID},
		BillingModelSource: BillingModelSourceChannelMapped,
		ModelMapping: map[string]map[string]string{
			PlatformOpenAI: {"public-alias": "target-model"},
		},
	}

	channel.RestrictModels = true
	offer, validationErrors := service.resolveOfferingFromSources(model, group, []Channel{channel})
	require.Nil(t, offer)
	require.NotEmpty(t, validationErrors, "global exact pricing must not bypass a restricted channel allowlist")

	channel.RestrictModels = false
	offer, validationErrors = service.resolveOfferingFromSources(model, group, []Channel{channel})
	require.NotNil(t, offer)
	require.Empty(t, validationErrors)
	require.InDelta(t, 1, *offer.public.InputPrice, 1e-12)
}

func TestResolveCatalogPricingDoesNotTurnUnknownIntoFree(t *testing.T) {
	service := &ModelCatalogService{}
	resolved := service.resolveCatalogChannelPricing("unknown", &ChannelModelPricing{BillingMode: BillingModeToken})
	require.NotNil(t, resolved)
	require.Nil(t, resolved.InputPrice)
	require.Nil(t, resolved.OutputPrice)
	require.False(t, hasExplicitCatalogPricing(resolved, &Group{}))
	cachePrice := 0.5
	configured := &ChannelModelPricing{BillingMode: BillingModeToken, CacheWritePrice: &cachePrice}
	resolved = service.resolveCatalogChannelPricing("unknown", configured)
	public := catalogPricingFromChannel(resolved, &Group{RateMultiplier: 2})
	service.applyCatalogComplexPricing(public, "unknown", resolved, configured, &Group{RateMultiplier: 2})
	require.InDelta(t, 1, *public.CacheWrite1hPrice, 1e-12)

	intervalPrice := 3.0
	configured = &ChannelModelPricing{BillingMode: BillingModeToken, InputPrice: &cachePrice, Intervals: []PricingInterval{{MinTokens: 0, InputPrice: &intervalPrice}}}
	resolved = service.resolveCatalogChannelPricing("unknown", configured)
	require.Nil(t, resolved.InputPrice, "flat channel fields are ignored when interval pricing is active")
	require.Len(t, resolved.Intervals, 1)

	zero := 0.0
	service.pricingService = &PricingService{pricingData: map[string]*LiteLLMModelPricing{
		"known": {InputCostPerToken: 2, InputCostPerTokenSet: true},
	}}
	resolved = service.resolveCatalogChannelPricing("known", &ChannelModelPricing{BillingMode: BillingModeToken, ImageInputPrice: &zero})
	require.NotNil(t, resolved.ImageInputPrice)
	require.InDelta(t, 2, *resolved.ImageInputPrice, 1e-12)
}

func TestCatalogComplexPricingMirrorsChannelOverridesAndLongContext(t *testing.T) {
	pricingService := &PricingService{pricingData: map[string]*LiteLLMModelPricing{
		"exact-model": {
			InputCostPerToken: 1, OutputCostPerToken: 2, CacheCreationInputTokenCost: 3,
			CacheCreationInputTokenCostAbove1hr: 4, InputCostPerTokenPriority: 5,
			LongContextInputTokenThreshold: 100, LongContextInputCostMultiplier: 2,
			LongContextOutputCostMultiplier: 3,
		},
	}}
	service := &ModelCatalogService{pricingService: pricingService}
	configuredInput, configuredCache := 7.0, 8.0
	configured := &ChannelModelPricing{BillingMode: BillingModeToken, InputPrice: &configuredInput, CacheWritePrice: &configuredCache}
	resolved := service.resolveCatalogChannelPricing("exact-model", configured)
	group := &Group{RateMultiplier: 2}
	public := catalogPricingFromChannel(resolved, group)
	service.applyCatalogComplexPricing(public, "exact-model", resolved, configured, group)

	require.InDelta(t, 14, *public.InputPrice, 1e-12)
	require.InDelta(t, 14, *public.PriorityInputPrice, 1e-12)
	require.InDelta(t, 16, *public.CacheWrite1hPrice, 1e-12)
	require.Len(t, public.Intervals, 1)
	require.Equal(t, 100, public.Intervals[0].MinTokens)
	require.InDelta(t, 28, *public.Intervals[0].InputPrice, 1e-12)
	require.InDelta(t, 32, *public.Intervals[0].CacheWrite1hPrice, 1e-12)
}

func TestCatalogSuppressesTopLevelPriorityWhenChannelIntervalsApply(t *testing.T) {
	pricingService := &PricingService{pricingData: map[string]*LiteLLMModelPricing{
		"interval-priority": {
			InputCostPerToken: 1, InputCostPerTokenPriority: 10,
			OutputCostPerToken: 2, OutputCostPerTokenPriority: 20,
		},
	}}
	intervalInput, intervalOutput := 3.0, 4.0
	configured := &ChannelModelPricing{BillingMode: BillingModeToken, Intervals: []PricingInterval{{
		MinTokens: 0, InputPrice: &intervalInput, OutputPrice: &intervalOutput,
	}}}
	catalogService := &ModelCatalogService{pricingService: pricingService}
	resolved := catalogService.resolveCatalogChannelPricing("interval-priority", configured)
	group := &Group{RateMultiplier: 1}
	public := catalogPricingFromChannel(resolved, group)
	catalogService.applyCatalogComplexPricing(public, "interval-priority", resolved, configured, group)
	require.Nil(t, public.PriorityInputPrice)
	require.Nil(t, public.PriorityOutputPrice)
	require.Len(t, public.Intervals, 1)

	billingService := &BillingService{pricingService: pricingService}
	basePricing, err := billingService.GetModelPricing("interval-priority")
	require.NoError(t, err)
	runtimeResolved := &ResolvedPricing{
		Mode: BillingModeToken, BasePricing: basePricing,
		Intervals: append([]PricingInterval(nil), configured.Intervals...),
	}
	cost, err := billingService.CalculateCostUnified(CostInput{
		Model: "interval-priority", Tokens: UsageTokens{InputTokens: 1, OutputTokens: 1},
		RateMultiplier: 1, ServiceTier: "priority", Resolver: &ModelPricingResolver{}, Resolved: runtimeResolved,
	})
	require.NoError(t, err)
	require.InDelta(t, *public.Intervals[0].InputPrice+*public.Intervals[0].OutputPrice, cost.ActualCost, 1e-12)
}

func TestExactCatalogAppliesSameGPTModelPolicyAsBilling(t *testing.T) {
	tests := []struct {
		model   string
		pricing *LiteLLMModelPricing
	}{
		{
			model: "gpt-5.4",
			pricing: &LiteLLMModelPricing{
				InputCostPerToken: 2.5e-6, InputCostPerTokenPriority: 5e-6,
				OutputCostPerToken: 15e-6, OutputCostPerTokenPriority: 30e-6,
				CacheCreationInputTokenCost: 2.5e-6, CacheReadInputTokenCost: 0.25e-6,
			},
		},
		{
			model: "gpt-5.6-sol",
			pricing: &LiteLLMModelPricing{
				InputCostPerToken: 5e-6, InputCostPerTokenPriority: 10e-6,
				OutputCostPerToken: 30e-6, OutputCostPerTokenPriority: 60e-6,
				CacheReadInputTokenCost: 0.5e-6, CacheReadInputTokenCostPriority: 1e-6,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			pricingService := &PricingService{pricingData: map[string]*LiteLLMModelPricing{tt.model: tt.pricing}}
			billingService := &BillingService{pricingService: pricingService}
			actual, err := billingService.GetModelPricing(tt.model)
			require.NoError(t, err)

			catalogService := &ModelCatalogService{pricingService: pricingService}
			resolved := catalogService.resolveCatalogChannelPricing(tt.model, nil)
			require.NotNil(t, resolved)
			public := catalogPricingFromChannel(resolved, &Group{RateMultiplier: 1})
			catalogService.applyCatalogComplexPricing(public, tt.model, resolved, nil, &Group{RateMultiplier: 1})

			require.InDelta(t, actual.CacheCreationPricePerToken, *public.CacheWritePrice, 1e-15)
			effectivePriority := func(priority, regular float64) float64 {
				if priority > 0 {
					return priority
				}
				return regular
			}
			require.InDelta(t, effectivePriority(actual.InputPricePerTokenPriority, actual.InputPricePerToken), *public.PriorityInputPrice, 1e-15)
			require.InDelta(t, effectivePriority(actual.OutputPricePerTokenPriority, actual.OutputPricePerToken), *public.PriorityOutputPrice, 1e-15)
			require.InDelta(t, effectivePriority(actual.CacheCreationPricePerTokenPriority, actual.CacheCreationPricePerToken), *public.PriorityCacheWritePrice, 1e-15)
			require.InDelta(t, effectivePriority(actual.CacheReadPricePerTokenPriority, actual.CacheReadPricePerToken), *public.PriorityCacheReadPrice, 1e-15)
			require.Len(t, public.Intervals, 1)
			longContext := public.Intervals[0]
			require.Equal(t, actual.LongContextInputThreshold, longContext.MinTokens)
			require.InDelta(t, actual.InputPricePerToken*actual.LongContextInputMultiplier, *longContext.InputPrice, 1e-15)
			require.InDelta(t, actual.OutputPricePerToken*actual.LongContextOutputMultiplier, *longContext.OutputPrice, 1e-15)
			require.InDelta(t, actual.CacheCreationPricePerToken*actual.LongContextInputMultiplier, *longContext.CacheWritePrice, 1e-15)
		})
	}
}

func TestCatalogImageTiersMatchBillingAndGroupOverrides(t *testing.T) {
	baseImagePrice := 0.10
	pricingService := &PricingService{pricingData: map[string]*LiteLLMModelPricing{
		"exact-image": {
			Mode: "image_generation", OutputCostPerImage: baseImagePrice, OutputCostPerImageSet: true,
		},
	}}
	catalogService := &ModelCatalogService{pricingService: pricingService}
	billingService := &BillingService{pricingService: pricingService}
	group2K := 0.40
	group := &Group{RateMultiplier: 2, ImagePrice2K: &group2K}
	groupConfig := &ImagePriceConfig{Price2K: &group2K}
	resolved := catalogService.resolveCatalogChannelPricing("exact-image", nil)
	public := catalogPricingFromChannel(resolved, group)
	catalogService.applyCatalogImagePricing(public, "exact-image", resolved, nil, group)
	require.True(t, catalogHasCompleteImageTiers(public))

	for _, tier := range []string{ImageBillingSize1K, ImageBillingSize2K, ImageBillingSize4K} {
		expected := billingService.CalculateImageCost("exact-image", tier, 1, groupConfig, group.RateMultiplier).ActualCost
		actual := catalogImageTierPrice(public.Intervals, tier)
		require.NotNil(t, actual)
		require.InDelta(t, expected, *actual, 1e-12)
	}
	require.InDelta(t, *catalogImageTierPrice(public.Intervals, ImageBillingSize2K), *public.PerRequestPrice, 1e-12)
}

func TestCatalogImagePricingDistinguishesLiteLLMZeroAndExplicitChannelZero(t *testing.T) {
	zero := 0.0
	pricingService := &PricingService{pricingData: map[string]*LiteLLMModelPricing{
		"zero-image": {
			Mode: "image_generation", OutputCostPerImage: 0, OutputCostPerImageSet: true,
		},
	}}
	catalogService := &ModelCatalogService{pricingService: pricingService}
	billingService := &BillingService{pricingService: pricingService}
	group := &Group{RateMultiplier: 1}

	resolved := catalogService.resolveCatalogChannelPricing("zero-image", nil)
	public := catalogPricingFromChannel(resolved, group)
	catalogService.applyCatalogImagePricing(public, "zero-image", resolved, nil, group)
	for _, tier := range []string{ImageBillingSize1K, ImageBillingSize2K, ImageBillingSize4K} {
		expected := billingService.CalculateImageCost("zero-image", tier, 1, nil, 1).ActualCost
		actual := catalogImageTierPrice(public.Intervals, tier)
		require.NotNil(t, actual)
		require.Greater(t, *actual, 0.0, "LiteLLM zero falls back to the runtime image default")
		require.InDelta(t, expected, *actual, 1e-12)
	}

	configured := &ChannelModelPricing{BillingMode: BillingModeImage, PerRequestPrice: &zero}
	resolved = catalogService.resolveCatalogChannelPricing("zero-image", configured)
	public = catalogPricingFromChannel(resolved, group)
	catalogService.applyCatalogImagePricing(public, "zero-image", resolved, configured, group)
	require.True(t, catalogHasCompleteImageTiers(public))
	runtimeResolved := &ResolvedPricing{Mode: BillingModeImage}
	runtimeResolver := &ModelPricingResolver{}
	runtimeResolver.applyRequestTierOverrides(configured, runtimeResolved)
	for _, tier := range []string{ImageBillingSize1K, ImageBillingSize2K, ImageBillingSize4K} {
		actual := catalogImageTierPrice(public.Intervals, tier)
		require.NotNil(t, actual)
		cost, err := billingService.CalculateCostUnified(CostInput{
			Model: "zero-image", RequestCount: 1, SizeTier: tier, RateMultiplier: 1,
			Resolver: runtimeResolver, Resolved: runtimeResolved,
		})
		require.NoError(t, err)
		require.Zero(t, cost.ActualCost, "explicit channel default zero is truly free at runtime")
		require.InDelta(t, cost.ActualCost, *actual, 1e-12)
	}
}

func TestCatalogPerRequestExplicitZeroTierMatchesBilling(t *testing.T) {
	zero, defaultPrice := 0.0, 0.75
	configured := &ChannelModelPricing{
		BillingMode: BillingModePerRequest, PerRequestPrice: &defaultPrice,
		Intervals: []PricingInterval{{TierLabel: ImageBillingSize1K, PerRequestPrice: &zero}},
	}
	catalogService := &ModelCatalogService{}
	resolved := catalogService.resolveCatalogChannelPricing("per-request-model", configured)
	public := catalogPricingFromChannel(resolved, &Group{RateMultiplier: 1})
	publicTier := catalogImageTierPrice(public.Intervals, ImageBillingSize1K)
	require.NotNil(t, publicTier)
	require.Zero(t, *publicTier)

	runtimeResolved := &ResolvedPricing{Mode: BillingModePerRequest}
	runtimeResolver := &ModelPricingResolver{}
	runtimeResolver.applyRequestTierOverrides(configured, runtimeResolved)
	cost, err := (&BillingService{}).CalculateCostUnified(CostInput{
		Model: "per-request-model", RequestCount: 1, SizeTier: ImageBillingSize1K,
		RateMultiplier: 1, Resolver: runtimeResolver, Resolved: runtimeResolved,
	})
	require.NoError(t, err)
	require.Zero(t, cost.ActualCost, "an explicitly configured zero tier must not fall back to the non-zero default")
	require.InDelta(t, *publicTier, cost.ActualCost, 1e-12)
}

func TestCatalogGrokImageTiersMatchBillingHardcodedPrices(t *testing.T) {
	catalogService := &ModelCatalogService{}
	billingService := &BillingService{}
	group := &Group{RateMultiplier: 1}
	model := "grok-imagine-image-quality"
	resolved := catalogService.resolveCatalogChannelPricing(model, nil)
	require.NotNil(t, resolved)
	public := catalogPricingFromChannel(resolved, group)
	catalogService.applyCatalogImagePricing(public, model, resolved, nil, group)
	for _, tier := range []string{ImageBillingSize1K, ImageBillingSize2K, ImageBillingSize4K} {
		expected := billingService.CalculateImageCost(model, tier, 1, nil, 1).ActualCost
		require.InDelta(t, expected, *catalogImageTierPrice(public.Intervals, tier), 1e-12)
	}
}

func TestCatalogImagePricingRejectsNonCanonicalRuntimeTierLabels(t *testing.T) {
	price1K, price2K, price4K := 0.1, 0.2, 0.4
	configured := &ChannelModelPricing{
		BillingMode: BillingModeImage,
		Intervals: []PricingInterval{
			{TierLabel: "1k", PerRequestPrice: &price1K},
			{TierLabel: "2k", PerRequestPrice: &price2K},
			{TierLabel: "4k", PerRequestPrice: &price4K},
		},
	}
	service := &ModelCatalogService{}
	group := &Group{RateMultiplier: 1}
	resolved := service.resolveCatalogChannelPricing("image-model", configured)
	public := catalogPricingFromChannel(resolved, group)
	service.applyCatalogImagePricing(public, "image-model", resolved, configured, group)
	require.False(t, catalogPricingIsPublishable(resolved, group, public))
	require.True(t, catalogHasNonCanonicalImageTier(public))

	// Runtime tier matching is exact, so canonical request size "1K" misses
	// the lowercase configured tier and falls through to the zero default.
	billingService := &BillingService{}
	runtimeResolved := &ResolvedPricing{
		Mode:         BillingModeImage,
		RequestTiers: append([]PricingInterval(nil), configured.Intervals...),
	}
	cost, err := billingService.CalculateCostUnified(CostInput{
		Model: "image-model", RequestCount: 1, SizeTier: ImageBillingSize1K,
		RateMultiplier: 1, Resolver: &ModelPricingResolver{}, Resolved: runtimeResolved,
	})
	require.NoError(t, err)
	require.Zero(t, cost.ActualCost)
}

func TestCatalogRejectsAmbiguousPerRequestImageMultiplier(t *testing.T) {
	price := 0.25
	service := &ModelCatalogService{}
	model := &ModelCatalogModel{Model: "flux-pro-custom", Platform: PlatformOpenAI}
	group := &Group{
		ID: 7, Platform: PlatformOpenAI, Status: StatusActive, SubscriptionType: SubscriptionTypeStandard,
		RateMultiplier: 1, ImageRateIndependent: true, ImageRateMultiplier: 2,
	}
	channel := Channel{
		Status: StatusActive, GroupIDs: []int64{group.ID}, BillingModelSource: BillingModelSourceRequested,
		ModelPricing: []ChannelModelPricing{{
			Platform: PlatformOpenAI, Models: []string{"flux-pro-custom"},
			BillingMode: BillingModePerRequest, PerRequestPrice: &price,
		}},
	}
	offer, validationErrors := service.resolveOfferingFromSources(model, group, []Channel{channel})
	require.Nil(t, offer)
	require.NotEmpty(t, validationErrors)

	group.ImageRateMultiplier = group.RateMultiplier
	offer, validationErrors = service.resolveOfferingFromSources(model, group, []Channel{channel})
	require.NotNil(t, offer)
	require.Empty(t, validationErrors)

	group.ImagePrice1K = &price
	offer, validationErrors = service.resolveOfferingFromSources(model, group, []Channel{channel})
	require.Nil(t, offer, "group image tier prices bypass channel per_request pricing at runtime")
	require.NotEmpty(t, validationErrors)
}

func TestUpdatePublishedCatalogRejectsMissingRequiredChineseCopyBeforeWrite(t *testing.T) {
	groupID := int64(7)
	price := 1.0
	repo := &modelCatalogRepoTestStub{model: &ModelCatalogModel{
		ID: 1, Model: "model", Platform: PlatformOpenAI, DisplayNameZH: "名称", SummaryZH: "描述",
		PublicGroupID: &groupID, Status: ModelCatalogStatusPublished,
	}}
	groupRepo := &modelCatalogGroupRepoTestStub{group: &Group{
		ID: groupID, Platform: PlatformOpenAI, Status: StatusActive, SubscriptionType: SubscriptionTypeStandard, RateMultiplier: 1,
	}}
	channelRepo := &modelCatalogChannelRepoTestStub{channels: []Channel{{
		Status: StatusActive, GroupIDs: []int64{groupID}, ModelPricing: []ChannelModelPricing{{Platform: PlatformOpenAI, Models: []string{"model"}, InputPrice: &price}},
	}}}
	service := NewModelCatalogService(repo, channelRepo, groupRepo, nil)
	_, err := service.Update(context.Background(), 1, ModelCatalogInput{
		Model: "model", Platform: PlatformOpenAI, DisplayNameZH: "", SummaryZH: "描述", PublicGroupID: &groupID,
	})
	require.Error(t, err)
	require.Zero(t, repo.updateCalls)
}

func TestPublicModelCatalogSnapshotFiltersAndHasStableETag(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	groupID := int64(7)
	zero, output := 0.0, 0.00001
	repo := &modelCatalogRepoTestStub{models: []ModelCatalogModel{
		{ID: 1, Slug: "openai-published", Model: "published", Platform: PlatformOpenAI, DisplayNameZH: "已发布", SummaryZH: "描述", PublicGroupID: &groupID, Status: ModelCatalogStatusPublished, UpdatedAt: now},
		{ID: 2, Slug: "openai-draft", Model: "draft", Platform: PlatformOpenAI, DisplayNameZH: "草稿", SummaryZH: "不可见", PublicGroupID: &groupID, Status: ModelCatalogStatusDraft, UpdatedAt: now},
	}}
	groupRepo := &modelCatalogGroupRepoTestStub{group: &Group{
		ID: groupID, Name: "public", Platform: PlatformOpenAI, Status: StatusActive,
		SubscriptionType: SubscriptionTypeStandard, RateMultiplier: 2, UpdatedAt: now,
	}}
	channelRepo := &modelCatalogChannelRepoTestStub{channels: []Channel{{
		ID: 9, Name: "internal", Status: StatusActive, GroupIDs: []int64{groupID}, UpdatedAt: now,
		ModelPricing: []ChannelModelPricing{{Platform: PlatformOpenAI, Models: []string{"published", "draft"}, InputPrice: &zero, OutputPrice: &output}},
	}}}
	service := NewModelCatalogService(repo, channelRepo, groupRepo, nil)

	first, etag1, err := service.PublicSnapshot(context.Background(), "zh-CN")
	require.NoError(t, err)
	second, etag2, err := service.PublicSnapshot(context.Background(), "zh-CN")
	require.NoError(t, err)
	require.Equal(t, etag1, etag2)
	require.Equal(t, first, second)
	require.Len(t, first.Items, 1)
	require.Equal(t, "published", first.Items[0].Model)
	require.NotNil(t, first.Items[0].Pricing.InputPrice)
	require.Equal(t, "CREDIT", first.Items[0].Pricing.Currency)
	require.Zero(t, *first.Items[0].Pricing.InputPrice)
	require.InDelta(t, output*2, *first.Items[0].Pricing.OutputPrice, 1e-12)

	groupRepo.group.IsExclusive = true
	service.InvalidatePublicCache()
	filtered, _, err := service.PublicSnapshot(context.Background(), "zh-CN")
	require.NoError(t, err)
	require.Empty(t, filtered.Items)
}
