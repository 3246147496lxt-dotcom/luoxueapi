package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
)

type billingReceiptRepositoryStub struct {
	filter *BillingReceiptFilter
}

func (s *billingReceiptRepositoryStub) GetUserWebChatReceipt(context.Context, int64, string) (*BillingReceipt, error) {
	return nil, ErrBillingReceiptNotFound
}

func (s *billingReceiptRepositoryStub) ListBillingReceipts(_ context.Context, filter *BillingReceiptFilter) (*BillingReceiptList, error) {
	s.filter = filter
	return &BillingReceiptList{}, nil
}

func TestBillingReceiptServiceNormalizesAttemptStatuses(t *testing.T) {
	for _, status := range []string{BillingReceiptStatusPending, BillingReceiptStatusFailed} {
		t.Run(status, func(t *testing.T) {
			repo := &billingReceiptRepositoryStub{}
			svc := NewBillingReceiptService(repo)

			_, err := svc.ListBillingReceipts(context.Background(), &BillingReceiptFilter{
				Status: " " + status + " ",
				Source: " WEB_CHAT ",
			})
			require.NoError(t, err)
			require.Equal(t, status, repo.filter.Status)
			require.Equal(t, BillingReceiptSourceWebChat, repo.filter.Source)
		})
	}
}

func TestBillingReceiptServiceRejectsInvalidFilter(t *testing.T) {
	svc := NewBillingReceiptService(&billingReceiptRepositoryStub{})
	_, err := svc.ListBillingReceipts(context.Background(), &BillingReceiptFilter{Status: "unknown"})
	require.ErrorIs(t, err, ErrBillingReceiptInvalidFilter)
}

func TestUsageBillingReceiptSourceIsTrustedAndFingerprinted(t *testing.T) {
	require.Equal(t, BillingReceiptSourceAPI, resolveBillingReceiptSource(context.Background()))
	webChatCtx := context.WithValue(context.Background(), ctxkey.WebChat, true)
	require.Equal(t, BillingReceiptSourceWebChat, resolveBillingReceiptSource(webChatCtx))

	apiCommand := &UsageBillingCommand{
		UserID:    1,
		APIKeyID:  2,
		RequestID: "client:test",
		Source:    BillingReceiptSourceAPI,
	}
	webChatCommand := *apiCommand
	webChatCommand.Source = BillingReceiptSourceWebChat
	apiCommand.Normalize()
	webChatCommand.Normalize()
	require.NotEqual(t, apiCommand.RequestFingerprint, webChatCommand.RequestFingerprint)
}
