//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
)

func TestWithGatewayTokenRequestPricingFreezesInstant(t *testing.T) {
	ctx, pricingAt := WithGatewayTokenRequestPricing(nil)
	require.False(t, pricingAt.IsZero())

	fromContext, ok := gatewayTokenRequestPricingAtFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, pricingAt, fromContext)
	require.Equal(t, pricingAt, GatewayTokenRequestPricingAtFromContext(ctx))
	require.True(t, GatewayTokenRequestPricingAtFromContext(context.Background()).IsZero())
}

func TestWithGatewayTokenRequestPricingCapturesOnlyHydratedBillingGroup(t *testing.T) {
	valid := newProfitControlTestGroup(41, PlatformGemini)
	ctx, _ := WithGatewayTokenRequestPricing(profitControlTestContext(valid))
	require.Same(t, valid, gatewayTokenRequestBillingGroupFromContext(ctx))

	invalid := newProfitControlTestGroup(42, PlatformGemini)
	invalid.Hydrated = false
	ctx, _ = WithGatewayTokenRequestPricing(profitControlTestContext(invalid))
	require.Nil(t, gatewayTokenRequestBillingGroupFromContext(ctx))
}

func TestWithGatewayTokenRequestBillingGroupPreservesFrozenInstant(t *testing.T) {
	original := newProfitControlTestGroup(51, PlatformAntigravity)
	original.RateMultiplier = 4
	fallback := newProfitControlTestGroup(52, PlatformAnthropic)
	fallback.RateMultiplier = 1

	ctx, pricingAt := WithGatewayTokenRequestPricing(profitControlTestContext(original))
	ctx = WithGatewayTokenRequestBillingGroup(ctx, fallback)
	// Scheduling resolves the fallback group into its local group context
	// before installing the gate.
	ctx = context.WithValue(ctx, ctxkey.Group, fallback)

	require.Equal(t, pricingAt, GatewayTokenRequestPricingAtFromContext(ctx))
	require.Same(t, fallback, gatewayTokenRequestBillingGroupFromContext(ctx))

	svc := &GatewayService{}
	fallbackID := fallback.ID
	ctx = svc.withGatewayProfitControlGate(ctx, &fallbackID)
	gate, ok := profitControlGateFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, fallback.ID, gate.groupID)
	require.InDelta(t, fallback.RateMultiplier*(1-fallback.ProfitMinMargin-fallback.ProfitSafetyBuffer), gate.threshold, 1e-12,
		"fallback admission must use the same group multiplier that will bill usage")
}

func TestWithGatewayTokenRequestBillingGroupRejectsUntrustedReplacement(t *testing.T) {
	original := newProfitControlTestGroup(61, PlatformAnthropic)
	ctx, pricingAt := WithGatewayTokenRequestPricing(profitControlTestContext(original))
	untrusted := newProfitControlTestGroup(62, PlatformAnthropic)
	untrusted.Hydrated = false

	updated := WithGatewayTokenRequestBillingGroup(ctx, untrusted)

	require.Equal(t, pricingAt, GatewayTokenRequestPricingAtFromContext(updated))
	require.Same(t, original, gatewayTokenRequestBillingGroupFromContext(updated))
}
