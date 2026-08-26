//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
)

type profitControlSnapshotStub struct {
	account *Account
	group   *Group
	err     error
}

func (s *profitControlSnapshotStub) GetAccount(context.Context, int64) (*Account, error) {
	return s.account, s.err
}

func (s *profitControlSnapshotStub) GetGroupByID(context.Context, int64) (*Group, error) {
	return s.group, s.err
}

func (s *profitControlSnapshotStub) UpdateAccountInCache(context.Context, *Account) error {
	return nil
}

func TestGatewayProfitControlRequiresTokenPricingMarker(t *testing.T) {
	group := newProfitControlTestGroup(21, PlatformAnthropic)
	groupID := group.ID
	svc := &GatewayService{}
	ctx := profitControlTestContext(group)

	withoutMarker := svc.withGatewayProfitControlGate(ctx, &groupID)
	require.False(t, gatewayProfitControlGateActive(withoutMarker))

	marked, pricingAt := WithGatewayTokenRequestPricing(ctx)
	withGate := svc.withGatewayProfitControlGate(marked, &groupID)
	gate, ok := profitControlGateFromContext(withGate)
	require.True(t, ok)
	require.Equal(t, pricingAt, gate.pricingAt)
	require.Equal(t, PlatformAnthropic, gate.platform)
}

func TestGatewayProfitControlSupportsFiveTokenPlatforms(t *testing.T) {
	platforms := []string{PlatformOpenAI, PlatformAnthropic, PlatformGemini, PlatformGrok, PlatformAntigravity}
	for i, platform := range platforms {
		t.Run(platform, func(t *testing.T) {
			group := newProfitControlTestGroup(int64(100+i), platform)
			groupID := group.ID
			ctx, _ := WithGatewayTokenRequestPricing(profitControlTestContext(group))
			ctx = (&GatewayService{}).withGatewayProfitControlGate(ctx, &groupID)
			gate, ok := profitControlGateFromContext(ctx)
			require.True(t, ok)
			require.Equal(t, platform, gate.platform)
		})
	}
}

func TestGatewayProfitControlKeepsIngressBillingGroup(t *testing.T) {
	billingGroup := newProfitControlTestGroup(31, PlatformAnthropic)
	billingGroup.RateMultiplier = 2
	dispatchGroup := newProfitControlTestGroup(32, PlatformGemini)
	dispatchGroup.RateMultiplier = 10
	dispatchGroup.ProfitMinMargin = 0.20
	dispatchGroup.ProfitSafetyBuffer = 0.05

	ctx, _ := WithGatewayTokenRequestPricing(profitControlTestContext(billingGroup))
	ctx = context.WithValue(ctx, ctxkey.Group, dispatchGroup)
	groupID := dispatchGroup.ID
	ctx = (&GatewayService{}).withGatewayProfitControlGate(ctx, &groupID)

	gate, ok := profitControlGateFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, dispatchGroup.ID, gate.groupID)
	// Configuration comes from the dispatch group, while D comes from the
	// authenticated billing group frozen before routing.
	require.InDelta(t, 2*(1-0.20-0.05), gate.threshold, 1e-12)
	require.Same(t, billingGroup, gatewayTokenRequestBillingGroupFromContext(ctx))
}

func TestProfitControlVetoLatestUsesOnlyNonStaleRefresh(t *testing.T) {
	now := time.Now()
	selected := &Account{ID: 1, UpdatedAt: now, RateMultiplier: profitControlRate(0.4)}
	gateCtx := context.WithValue(context.Background(), openAIProfitControlGateCtxKey{}, &openAIProfitControlGate{
		groupID: 1, platform: PlatformAnthropic, threshold: 0.5,
	})

	t.Run("newer snapshot can veto", func(t *testing.T) {
		refreshed := &Account{ID: 1, UpdatedAt: now.Add(time.Second), RateMultiplier: profitControlRate(0.8)}
		latest, vetoed, reason := profitControlVetoLatest(gateCtx, selected, &profitControlSnapshotStub{account: refreshed})
		require.Same(t, refreshed, latest)
		require.True(t, vetoed)
		require.Equal(t, openAIProfitFilterReasonThreshold, reason)
	})

	t.Run("older snapshot cannot replace a fresh selection", func(t *testing.T) {
		stale := &Account{ID: 1, UpdatedAt: now.Add(-time.Second), RateMultiplier: profitControlRate(0.8)}
		latest, vetoed, reason := profitControlVetoLatest(gateCtx, selected, &profitControlSnapshotStub{account: stale})
		require.Same(t, selected, latest)
		require.False(t, vetoed)
		require.Empty(t, reason)
	})

	t.Run("refresh failure uses selected snapshot", func(t *testing.T) {
		latest, vetoed, reason := profitControlVetoLatest(gateCtx, selected, &profitControlSnapshotStub{err: errors.New("cache unavailable")})
		require.Same(t, selected, latest)
		require.False(t, vetoed)
		require.Empty(t, reason)
	})
}
