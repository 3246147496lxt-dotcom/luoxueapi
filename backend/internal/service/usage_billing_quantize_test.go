package service

import (
	"math"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestUsageBillingNormalizeQuantizesMonetaryFields(t *testing.T) {
	const raw = 0.000078125
	cmd := &UsageBillingCommand{
		RequestID: "quantize", UserID: 1, AccountID: 2, APIKeyID: 3,
		BalanceCost: raw, SubscriptionCost: raw, APIKeyQuotaCost: raw,
		APIKeyRateLimitCost: raw, AccountQuotaCost: raw,
	}
	cmd.Normalize()
	want, _ := decimal.NewFromFloat(raw).Round(UsageBillingMonetaryScale).Float64()
	for _, got := range []float64{cmd.BalanceCost, cmd.SubscriptionCost, cmd.APIKeyQuotaCost, cmd.APIKeyRateLimitCost, cmd.AccountQuotaCost} {
		require.Equal(t, want, got)
	}
}

func TestQuantizeUsageBillingAmountUsesHalfAwayFromZero(t *testing.T) {
	for _, raw := range []float64{0.000078125, -0.000078125, 0.000078124, -0.000078124} {
		want, _ := decimal.NewFromFloat(raw).Round(UsageBillingMonetaryScale).Float64()
		require.Equal(t, want, QuantizeUsageBillingAmount(raw))
	}
	require.True(t, math.IsNaN(QuantizeUsageBillingAmount(math.NaN())))
}

func TestUsageBillingNormalizeFingerprintUsesRawAmount(t *testing.T) {
	cmd := &UsageBillingCommand{RequestID: "quantize-fingerprint", UserID: 1, AccountID: 2, APIKeyID: 3, Source: BillingReceiptSourceAPI, GrossCost: 0.000078125, BalanceCost: 0.000078125}
	want := buildUsageBillingFingerprint(cmd)
	cmd.Normalize()
	require.Equal(t, want, cmd.RequestFingerprint)
}
