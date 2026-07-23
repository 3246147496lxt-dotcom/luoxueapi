package service

import (
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
)

func TestNewPaymentOrderViewCopiesResponseFieldsAndResolvesCurrency(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.July, 22, 12, 34, 56, 0, time.UTC)
	userNotes := "reviewed"
	providerID := "stripe-primary"
	order := &dbent.PaymentOrder{
		ID:                 17,
		UserID:             23,
		UserEmail:          "user@example.com",
		UserName:           "user",
		UserNotes:          &userNotes,
		Amount:             100,
		PayAmount:          103,
		FeeRate:            3,
		OutTradeNo:         "sub2_202607220001",
		PaymentType:        "stripe",
		OrderType:          "subscription",
		ProviderInstanceID: &providerID,
		Status:             OrderStatusCompleted,
		ExpiresAt:          now.Add(time.Hour),
		CreatedAt:          now,
		UpdatedAt:          now.Add(time.Minute),
		ProviderSnapshot: map[string]any{
			"schema_version": 2,
			"currency":       "USD",
		},
	}

	got := newPaymentOrderView(order)
	if got == nil {
		t.Fatal("expected payment order view")
	}
	if got.ID != order.ID || got.UserID != order.UserID || got.OutTradeNo != order.OutTradeNo {
		t.Fatalf("identity fields changed during projection: %#v", got)
	}
	if got.Currency != "USD" {
		t.Fatalf("expected snapshot currency USD, got %q", got.Currency)
	}
	if got.ProviderInstanceID == nil || *got.ProviderInstanceID != providerID {
		t.Fatalf("provider instance changed during projection: %#v", got.ProviderInstanceID)
	}
	if got.UpdatedAt != order.UpdatedAt {
		t.Fatalf("updated_at changed during projection: got %v want %v", got.UpdatedAt, order.UpdatedAt)
	}
}

func TestNewPaymentOrderViewPreservesNil(t *testing.T) {
	t.Parallel()

	if got := newPaymentOrderView(nil); got != nil {
		t.Fatalf("expected nil projection, got %#v", got)
	}
}
