//go:build unit

package service

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
)

func newProfitControlTestGroup(id int64, platform string) *Group {
	return &Group{
		ID:                   id,
		Platform:             platform,
		Status:               StatusActive,
		Hydrated:             true,
		RateMultiplier:       2,
		SubscriptionType:     SubscriptionTypeStandard,
		ProfitControlEnabled: true,
		ProfitMinMargin:      0.25,
		ProfitSafetyBuffer:   0.10,
	}
}

func profitControlTestContext(group *Group) context.Context {
	return context.WithValue(context.Background(), ctxkey.Group, group)
}

func profitControlRate(rate float64) *float64 {
	return &rate
}

func TestOpenAIProfitControlPricingAndGate(t *testing.T) {
	group := newProfitControlTestGroup(11, PlatformOpenAI)
	groupID := group.ID
	svc := &OpenAIGatewayService{}

	ctx, pricingAt := svc.WithOpenAIRequestPricingContext(profitControlTestContext(group), &groupID)
	require.False(t, pricingAt.IsZero())
	require.Equal(t, pricingAt, OpenAIPricingAtFromContext(ctx))

	gate, ok := profitControlGateFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, group.ID, gate.groupID)
	require.Equal(t, PlatformOpenAI, gate.platform)
	require.Equal(t, pricingAt, gate.pricingAt)
	require.InDelta(t, 2*(1-0.25-0.10), gate.threshold, 1e-12)
}

func TestOpenAIProfitControlDefaultOffAndScope(t *testing.T) {
	svc := &OpenAIGatewayService{}

	t.Run("disabled by default", func(t *testing.T) {
		group := newProfitControlTestGroup(1, PlatformOpenAI)
		group.ProfitControlEnabled = false
		id := group.ID
		ctx, _ := svc.WithOpenAIRequestPricingContext(profitControlTestContext(group), &id)
		require.False(t, gatewayProfitControlGateActive(ctx))
	})

	t.Run("OpenAI service also supports Grok", func(t *testing.T) {
		group := newProfitControlTestGroup(2, PlatformGrok)
		id := group.ID
		ctx, _ := svc.WithOpenAIRequestPricingContext(profitControlTestContext(group), &id)
		gate, ok := profitControlGateFromContext(ctx)
		require.True(t, ok)
		require.Equal(t, PlatformGrok, gate.platform)
	})

	t.Run("other platforms use the shared gateway", func(t *testing.T) {
		group := newProfitControlTestGroup(3, PlatformAnthropic)
		id := group.ID
		ctx, _ := svc.WithOpenAIRequestPricingContext(profitControlTestContext(group), &id)
		require.False(t, gatewayProfitControlGateActive(ctx))
	})

	t.Run("suppression freezes pricing without installing a gate", func(t *testing.T) {
		group := newProfitControlTestGroup(4, PlatformOpenAI)
		id := group.ID
		ctx := WithOpenAIProfitControlSuppressed(profitControlTestContext(group))
		ctx, pricingAt := svc.WithOpenAIRequestPricingContext(ctx, &id)
		require.False(t, pricingAt.IsZero())
		require.Equal(t, pricingAt, OpenAIPricingAtFromContext(ctx))
		require.False(t, gatewayProfitControlGateActive(ctx))
	})

	t.Run("a foreign gate is cleared when dispatch group has no gate", func(t *testing.T) {
		group := newProfitControlTestGroup(5, PlatformOpenAI)
		id := group.ID
		ctx, _ := svc.WithOpenAIRequestPricingContext(profitControlTestContext(group), &id)
		require.True(t, gatewayProfitControlGateActive(ctx))

		foreignID := int64(999)
		ctx = svc.withOpenAIProfitControlGate(ctx, &foreignID)
		require.False(t, gatewayProfitControlGateActive(ctx))
	})
}

func TestOpenAIProfitControlVetoBoundaries(t *testing.T) {
	ctxWithThreshold := func(threshold float64) context.Context {
		return context.WithValue(context.Background(), openAIProfitControlGateCtxKey{}, &openAIProfitControlGate{
			groupID:   1,
			platform:  PlatformOpenAI,
			threshold: threshold,
			pricingAt: time.Now(),
		})
	}

	t.Run("no gate admits", func(t *testing.T) {
		vetoed, reason := openAIProfitControlVetoReason(context.Background(), &Account{})
		require.False(t, vetoed)
		require.Empty(t, reason)
	})

	t.Run("equal and float-noise rates admit", func(t *testing.T) {
		for _, rate := range []float64{0.7, 0.7 + 1e-12} {
			vetoed, reason := openAIProfitControlVetoReason(ctxWithThreshold(0.7), &Account{RateMultiplier: profitControlRate(rate)})
			require.False(t, vetoed)
			require.Empty(t, reason)
		}
	})

	t.Run("over-threshold rate is rejected", func(t *testing.T) {
		vetoed, reason := openAIProfitControlVetoReason(ctxWithThreshold(0.7), &Account{RateMultiplier: profitControlRate(0.8)})
		require.True(t, vetoed)
		require.Equal(t, openAIProfitFilterReasonThreshold, reason)
	})

	t.Run("zero threshold admits only a free upstream", func(t *testing.T) {
		vetoed, _ := openAIProfitControlVetoReason(ctxWithThreshold(0), &Account{RateMultiplier: profitControlRate(0)})
		require.False(t, vetoed)
		vetoed, reason := openAIProfitControlVetoReason(ctxWithThreshold(0), &Account{RateMultiplier: profitControlRate(0.01)})
		require.True(t, vetoed)
		require.Equal(t, openAIProfitFilterReasonThreshold, reason)
	})

	t.Run("missing negative and non-finite rates fail closed", func(t *testing.T) {
		accounts := []*Account{
			{},
			{RateMultiplier: profitControlRate(-1)},
			{RateMultiplier: profitControlRate(math.NaN())},
			{RateMultiplier: profitControlRate(math.Inf(1))},
		}
		for _, account := range accounts {
			vetoed, reason := openAIProfitControlVetoReason(ctxWithThreshold(0.7), account)
			require.True(t, vetoed)
			require.Equal(t, openAIProfitFilterReasonInvalidAccountRate, reason)
		}
	})
}

func TestProfitControlThresholdSanitization(t *testing.T) {
	require.Zero(t, clampProfitControlThreshold(-1))
	require.Zero(t, clampProfitControlThreshold(math.NaN()))
	require.Zero(t, clampProfitControlThreshold(math.Inf(1)))
	require.Equal(t, 0.25, clampProfitControlThreshold(0.25))
}

func TestSelectionCarriesResolvedProfitGate(t *testing.T) {
	var nilSelection *AccountSelectionResult
	require.False(t, nilSelection.ProfitGateActive())

	gate := &openAIProfitControlGate{groupID: 7, platform: PlatformOpenAI, threshold: 0.5}
	selectionCtx := context.WithValue(context.Background(), openAIProfitControlGateCtxKey{}, gate)
	selection := attachSelectionProfitGate(selectionCtx, &AccountSelectionResult{Account: &Account{ID: 1}})
	require.Same(t, gate, selection.profitGate)
	require.True(t, selection.ProfitGateActive())

	replayed := ContextWithSelectionProfitGate(context.Background(), selection)
	replayedGate, ok := profitControlGateFromContext(replayed)
	require.True(t, ok)
	require.Same(t, gate, replayedGate)
}
