package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func comparisonCatalogService(group *Group, channelPricing ChannelModelPricing, exact *LiteLLMModelPricing) *ModelCatalogService {
	now := time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)
	group.Status = StatusActive
	group.SubscriptionType = SubscriptionTypeStandard
	group.Platform = PlatformOpenAI
	group.UpdatedAt = now
	channelPricing.Platform = PlatformOpenAI
	channelPricing.Models = []string{"published-model"}
	repo := &modelCatalogRepoTestStub{models: []ModelCatalogModel{{
		ID: 1, Slug: "published-model", Model: "published-model", Platform: PlatformOpenAI,
		DisplayNameZH: "公开模型", SummaryZH: "公开说明", PublicGroupID: &group.ID,
		Status: ModelCatalogStatusPublished, UpdatedAt: now,
	}}}
	channels := &modelCatalogChannelRepoTestStub{channels: []Channel{{
		ID: 99, Name: "private-channel-name", Description: "private-channel-description",
		Status: StatusActive, GroupIDs: []int64{group.ID}, UpdatedAt: now,
		ModelPricing: []ChannelModelPricing{channelPricing},
	}}}
	var pricing *PricingService
	if exact != nil {
		pricing = &PricingService{pricingData: map[string]*LiteLLMModelPricing{"published-model": exact}}
	}
	return NewModelCatalogService(repo, channels, &modelCatalogGroupRepoTestStub{group: group}, pricing)
}

func TestPublicCatalogComparisonSeparatesOfficialFromChannelAndGroupPrices(t *testing.T) {
	input, output, cacheWrite := 7e-6, 30e-6, 8e-6
	group := &Group{
		ID: 7, Name: "Public group", RateMultiplier: 0.056,
		Description: "private-group-description", ModelRouting: map[string][]int64{"private-routing": {88}},
	}
	exact := &LiteLLMModelPricing{
		InputCostPerToken: 5e-6, OutputCostPerToken: 0, OutputCostPerTokenSet: true,
		CacheCreationInputTokenCost: 6e-6, CacheCreationInputTokenCostAbove1hr: 10e-6,
		LongContextInputTokenThreshold: 272000, LongContextInputCostMultiplier: 2,
		LongContextOutputCostMultiplier: 1.5,
	}
	svc := comparisonCatalogService(group, ChannelModelPricing{
		InputPrice: &input, OutputPrice: &output, CacheWritePrice: &cacheWrite,
	}, exact)
	response, _, err := svc.PublicSnapshot(context.Background(), "zh-CN")
	require.NoError(t, err)
	require.Len(t, response.Items, 1)
	item := response.Items[0]
	require.Equal(t, &PublicCatalogGroup{ID: 7, Name: "Public group", Platform: PlatformOpenAI, RateMultiplier: 0.056}, item.PublicGroup)
	require.InDelta(t, 0.056, *item.RateMultiplier, 1e-12)
	require.InDelta(t, input*0.056, *item.Pricing.InputPrice, 1e-15)
	require.InDelta(t, output*0.056, *item.Pricing.OutputPrice, 1e-15)
	require.InDelta(t, cacheWrite*0.056, *item.Pricing.CacheWrite1hPrice, 1e-15)
	require.NotNil(t, item.OfficialPricing)
	require.Equal(t, "USD", item.OfficialPricing.Currency)
	require.InDelta(t, 5e-6, *item.OfficialPricing.InputPrice, 1e-15)
	require.Zero(t, *item.OfficialPricing.OutputPrice)
	require.Nil(t, item.OfficialPricing.CacheReadPrice)
	require.InDelta(t, 10e-6, *item.OfficialPricing.CacheWrite1hPrice, 1e-15)
	require.Len(t, item.OfficialPricing.Intervals, 1)
	require.Equal(t, 272000, item.OfficialPricing.Intervals[0].MinTokens)
	require.InDelta(t, 10e-6, *item.OfficialPricing.Intervals[0].InputPrice, 1e-15)
	require.InDelta(t, 20e-6, *item.OfficialPricing.Intervals[0].CacheWrite1hPrice, 1e-15)
	require.InDelta(t, 5e-6, exact.InputCostPerToken, 1e-15, "projection must not mutate source data")

	wire, err := json.Marshal(response)
	require.NoError(t, err)
	for _, privateValue := range []string{"private-channel-name", "private-channel-description", "private-group-description", "private-routing", "channel_id", "account_ids", "model_routing"} {
		require.NotContains(t, string(wire), privateValue)
	}
	var decoded struct {
		Items []struct {
			PublicGroup map[string]any `json:"public_group"`
		} `json:"items"`
	}
	require.NoError(t, json.Unmarshal(wire, &decoded))
	require.Len(t, decoded.Items[0].PublicGroup, 4)
}

func TestPublicCatalogComparisonZeroGroupStillHasOfficialPrice(t *testing.T) {
	input := 7e-6
	svc := comparisonCatalogService(&Group{ID: 7, Name: "Free", RateMultiplier: 0}, ChannelModelPricing{InputPrice: &input}, &LiteLLMModelPricing{InputCostPerToken: 5e-6})
	response, _, err := svc.PublicSnapshot(context.Background(), "zh-CN")
	require.NoError(t, err)
	item := response.Items[0]
	require.Zero(t, *item.RateMultiplier)
	require.Zero(t, *item.Pricing.InputPrice)
	require.InDelta(t, 5e-6, *item.OfficialPricing.InputPrice, 1e-15)
}

func TestPublicCatalogComparisonRequiresExactOfficialSource(t *testing.T) {
	input := 7e-6
	for _, exact := range []*LiteLLMModelPricing{nil, {MaxInputTokens: 1000}} {
		svc := comparisonCatalogService(&Group{ID: 7, Name: "Custom", RateMultiplier: 0.1}, ChannelModelPricing{InputPrice: &input}, exact)
		response, _, err := svc.PublicSnapshot(context.Background(), "zh-CN")
		require.NoError(t, err)
		require.Len(t, response.Items, 1)
		require.Nil(t, response.Items[0].OfficialPricing)
		require.InDelta(t, input*0.1, *response.Items[0].Pricing.InputPrice, 1e-15)
	}
	svc := &ModelCatalogService{pricingService: &PricingService{pricingData: map[string]*LiteLLMModelPricing{
		"claude-sonnet-4": {InputCostPerToken: 3e-6},
	}}}
	require.Nil(t, svc.catalogOfficialPricing("claude-sonnet-4-unpublished-version", BillingModeToken))
}

func TestPublicCatalogComparisonImageKeepsIndependentRateAndExactQuote(t *testing.T) {
	channelPrice, group2K := 0.5, 0.4
	for _, officialPrice := range []float64{0.12, 0} {
		svc := comparisonCatalogService(&Group{
			ID: 7, Name: "Images", RateMultiplier: 0.1,
			ImageRateIndependent: true, ImageRateMultiplier: 0.2, ImagePrice2K: &group2K,
		}, ChannelModelPricing{BillingMode: BillingModeImage, PerRequestPrice: &channelPrice}, &LiteLLMModelPricing{
			Mode: "image_generation", OutputCostPerImage: officialPrice, OutputCostPerImageSet: true,
		})
		response, _, err := svc.PublicSnapshot(context.Background(), "zh-CN")
		require.NoError(t, err)
		require.Len(t, response.Items, 1)
		item := response.Items[0]
		require.InDelta(t, 0.1, item.PublicGroup.RateMultiplier, 1e-12)
		require.InDelta(t, 0.2, *item.RateMultiplier, 1e-12)
		require.InDelta(t, group2K*0.2, *item.Pricing.PerRequestPrice, 1e-12)
		require.InDelta(t, channelPrice*0.2, *catalogImageTierPrice(item.Pricing.Intervals, ImageBillingSize1K), 1e-12)
		require.Equal(t, "USD", item.OfficialPricing.Currency)
		require.Equal(t, "per_request", item.OfficialPricing.Unit)
		require.InDelta(t, officialPrice, *item.OfficialPricing.PerRequestPrice, 1e-12)
		require.Empty(t, item.OfficialPricing.Intervals, "runtime size multipliers must not manufacture official tier quotes")
	}
}

func TestPublicCatalogComparisonDoesNotTurnOfficialTokenPriceIntoPerRequestPrice(t *testing.T) {
	channelPrice := 0.5
	svc := comparisonCatalogService(&Group{ID: 7, Name: "Images", RateMultiplier: 0.1}, ChannelModelPricing{
		BillingMode: BillingModeImage, PerRequestPrice: &channelPrice,
	}, &LiteLLMModelPricing{Mode: "image_generation", InputCostPerToken: 5e-6, OutputCostPerImageToken: 30e-6})
	response, _, err := svc.PublicSnapshot(context.Background(), "zh-CN")
	require.NoError(t, err)
	require.Len(t, response.Items, 1)
	require.Nil(t, response.Items[0].OfficialPricing)
}

func TestPublicCatalogComparisonKeepsOfficialAndChannelIntervalsIndependent(t *testing.T) {
	input, tierInput := 7e-6, 9e-6
	maxTokens := 100000
	svc := comparisonCatalogService(&Group{ID: 7, Name: "Public", RateMultiplier: 0.1}, ChannelModelPricing{
		InputPrice: &input,
		Intervals:  []PricingInterval{{MinTokens: 0, MaxTokens: &maxTokens, InputPrice: &tierInput}},
	}, &LiteLLMModelPricing{
		InputCostPerToken: 5e-6, OutputCostPerToken: 30e-6,
		LongContextInputTokenThreshold: 272000, LongContextInputCostMultiplier: 2,
	})
	response, _, err := svc.PublicSnapshot(context.Background(), "zh-CN")
	require.NoError(t, err)
	item := response.Items[0]
	require.Len(t, item.Pricing.Intervals, 1)
	require.Equal(t, &maxTokens, item.Pricing.Intervals[0].MaxTokens)
	require.InDelta(t, tierInput*0.1, *item.Pricing.Intervals[0].InputPrice, 1e-15)
	require.Nil(t, item.Pricing.Intervals[0].OutputPrice, "a missing interval field is not inherited from the base price")
	require.Len(t, item.OfficialPricing.Intervals, 1)
	require.Equal(t, 272000, item.OfficialPricing.Intervals[0].MinTokens)
	require.Nil(t, item.OfficialPricing.Intervals[0].MaxTokens)
	require.InDelta(t, 10e-6, *item.OfficialPricing.Intervals[0].InputPrice, 1e-15)
}

func TestPublicCatalogComparisonPreservesPublishedOfferBoundary(t *testing.T) {
	input := 5e-6
	for _, test := range []struct {
		name   string
		mutate func(*Group, *modelCatalogRepoTestStub)
	}{
		{"exclusive group", func(group *Group, _ *modelCatalogRepoTestStub) { group.IsExclusive = true }},
		{"inactive group", func(group *Group, _ *modelCatalogRepoTestStub) { group.Status = "inactive" }},
		{"subscription group", func(group *Group, _ *modelCatalogRepoTestStub) { group.SubscriptionType = SubscriptionTypeSubscription }},
		{"draft model", func(_ *Group, repo *modelCatalogRepoTestStub) { repo.models[0].Status = ModelCatalogStatusDraft }},
		{"unbound model", func(_ *Group, repo *modelCatalogRepoTestStub) { repo.models[0].PublicGroupID = nil }},
		{"other group", func(_ *Group, repo *modelCatalogRepoTestStub) { id := int64(99); repo.models[0].PublicGroupID = &id }},
	} {
		t.Run(test.name, func(t *testing.T) {
			group := &Group{ID: 7, Name: "Must not publish", RateMultiplier: 0.1}
			svc := comparisonCatalogService(group, ChannelModelPricing{InputPrice: &input}, &LiteLLMModelPricing{InputCostPerToken: input})
			test.mutate(group, svc.repo.(*modelCatalogRepoTestStub))
			response, _, err := svc.PublicSnapshot(context.Background(), "zh-CN")
			require.NoError(t, err)
			require.Empty(t, response.Items)
			wire, err := json.Marshal(response)
			require.NoError(t, err)
			require.NotContains(t, string(wire), "Must not publish")
		})
	}
}

func TestPublicCatalogOfficialPriorityDoesNotInheritRuntimeFallbacks(t *testing.T) {
	for _, priorityInput := range []float64{0, 10e-6} {
		svc := &ModelCatalogService{pricingService: &PricingService{pricingData: map[string]*LiteLLMModelPricing{
			"exact-model": {
				InputCostPerToken: 5e-6, OutputCostPerToken: 30e-6, SupportsServiceTier: true,
				InputCostPerTokenPriority: priorityInput,
			},
		}}}
		official := svc.catalogOfficialPricing("exact-model", BillingModeToken)
		require.NotNil(t, official)
		if priorityInput == 0 {
			require.Nil(t, official.PriorityInputPrice)
		} else {
			require.InDelta(t, priorityInput, *official.PriorityInputPrice, 1e-15)
		}
		require.Nil(t, official.PriorityOutputPrice)
		require.Nil(t, official.PriorityCacheWritePrice)
		require.Nil(t, official.PriorityCacheReadPrice)
	}
}
