package service

import (
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/stretchr/testify/require"
)

func dashboardCurrencyTestOrder(currency string, payAmount float64, paidAt time.Time) *dbent.PaymentOrder {
	snapshot := map[string]any{}
	if currency != "" {
		snapshot["currency"] = currency
	}
	return &dbent.PaymentOrder{
		PayAmount:        payAmount,
		PaidAt:           &paidAt,
		ProviderSnapshot: snapshot,
	}
}

func TestDashboardCurrencyHelpersKeepAggregatesSingleCurrency(t *testing.T) {
	now := time.Now()
	orders := []*dbent.PaymentOrder{
		dashboardCurrencyTestOrder("", 10, now),
		dashboardCurrencyTestOrder("CNY", 20, now),
		dashboardCurrencyTestOrder("usd", 30, now),
		dashboardCurrencyTestOrder("invalid", 40, now),
	}

	require.Equal(t, []string{"CNY", "HKD", "USD"}, dashboardCurrencies(orders, "HKD"))
	require.Len(t, filterPaymentOrdersByCurrency(orders, "USD"), 1)

	cnyOrders := filterPaymentOrdersByCurrency(orders, "CNY")
	require.Len(t, cnyOrders, 3)
	stats := &DashboardStats{}
	computeBasicStats(stats, cnyOrders, now.Add(-time.Hour))
	require.Equal(t, 70.0, stats.TodayAmount)
	require.Equal(t, 70.0, stats.TotalAmount)
	require.Equal(t, 3, stats.TotalCount)
}
