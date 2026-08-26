//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUsageBillingSideEffectCostsPreferCommittedQuantizedDeltas(t *testing.T) {
	const rawCost = 0.000078125
	const committedBalanceCost = 0.00007813
	const committedAccountCost = 0.00009766

	p := &postUsageBillingParams{
		Cost: &CostBreakdown{
			ActualCost: rawCost,
			TotalCost:  rawCost,
		},
		AccountRateMultiplier: 1.25,
	}
	result := &UsageBillingApplyResult{
		BalanceChargedCost:         committedBalanceCost,
		APIKeyRateLimitChargedCost: committedBalanceCost,
		AccountQuotaChargedCost:    committedAccountCost,
	}

	require.Equal(t, committedBalanceCost, usageBillingBalanceChargedCost(p, result))
	require.Equal(t, committedBalanceCost, usageBillingRateLimitChargedCost(p, result))
	require.Equal(t, committedAccountCost, usageBillingAccountQuotaChargedCost(p, result))
}

func TestUsageBillingSideEffectCostsFallbackToSameQuantizer(t *testing.T) {
	const rawCost = 0.000078125
	p := &postUsageBillingParams{
		Cost: &CostBreakdown{
			ActualCost: rawCost,
			TotalCost:  rawCost,
		},
		AccountRateMultiplier: 1.25,
	}

	require.Equal(t, QuantizeUsageBillingAmount(rawCost), usageBillingBalanceChargedCost(p, nil))
	require.Equal(t, QuantizeUsageBillingAmount(rawCost), usageBillingRateLimitChargedCost(p, nil))
	require.Equal(t, QuantizeUsageBillingAmount(rawCost*1.25), usageBillingAccountQuotaChargedCost(p, nil))
}

func TestResolveOldBalancePrefersDatabaseBeforeBalance(t *testing.T) {
	const rawCost = 0.000078125
	p := &postUsageBillingParams{
		Cost: &CostBreakdown{ActualCost: rawCost},
		User: &User{Balance: 99},
	}
	before := 10.0
	after := 9.99992187
	result := &UsageBillingApplyResult{
		BalanceBefore:      &before,
		NewBalance:         &after,
		BalanceChargedCost: QuantizeUsageBillingAmount(rawCost),
	}

	require.Equal(t, before, resolveOldBalance(p, result))
}
