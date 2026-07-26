package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type adminBillingReceiptRepositoryStub struct {
	filter *service.BillingReceiptFilter
	list   *service.BillingReceiptList
	err    error
}

func (s *adminBillingReceiptRepositoryStub) GetUserWebChatReceipt(
	context.Context,
	int64,
	string,
) (*service.BillingReceipt, error) {
	return nil, service.ErrBillingReceiptNotFound
}

func (s *adminBillingReceiptRepositoryStub) ListBillingReceipts(
	_ context.Context,
	filter *service.BillingReceiptFilter,
) (*service.BillingReceiptList, error) {
	s.filter = filter
	return s.list, s.err
}

func TestAdminListBillingReceiptsPassesReadOnlyFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &adminBillingReceiptRepositoryStub{
		list: &service.BillingReceiptList{
			Receipts: []*service.BillingReceipt{{
				ID:             1,
				RequestID:      "client:11111111-1111-4111-8111-111111111111",
				Source:         service.BillingReceiptSourceWebChat,
				UserID:         42,
				RequestedModel: "gpt-5.5",
				ChargedAmount:  0.01,
				Status:         service.BillingReceiptStatusCharged,
				CreatedAt:      time.Now(),
			}},
			Total:    1,
			Page:     2,
			PageSize: 25,
		},
	}
	handler := NewUsageHandler(
		nil,
		nil,
		nil,
		nil,
		service.NewBillingReceiptService(repo),
	)
	router := gin.New()
	router.GET("/admin/billing/receipts", handler.ListBillingReceipts)

	req := httptest.NewRequest(
		http.MethodGet,
		"/admin/billing/receipts?page=2&page_size=25&user_id=42&model=gpt-5.5&receipt_id=11111111-1111-4111-8111-111111111111&status=charged&source=web_chat&start_date=2026-07-01&end_date=2026-07-25",
		nil,
	)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.NotNil(t, repo.filter)
	require.Equal(t, 2, repo.filter.Page)
	require.Equal(t, 25, repo.filter.PageSize)
	require.NotNil(t, repo.filter.UserID)
	require.Equal(t, int64(42), *repo.filter.UserID)
	require.Equal(t, "gpt-5.5", repo.filter.Model)
	require.Equal(t, "11111111-1111-4111-8111-111111111111", repo.filter.RequestID)
	require.Equal(t, "charged", repo.filter.Status)
	require.Equal(t, "web_chat", repo.filter.Source)
	require.NotNil(t, repo.filter.StartTime)
	require.NotNil(t, repo.filter.EndTime)
	require.Contains(t, recorder.Body.String(), `"charged_amount":0.01`)
}

func TestAdminListBillingReceiptsRejectsInvalidUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewUsageHandler(
		nil,
		nil,
		nil,
		nil,
		service.NewBillingReceiptService(&adminBillingReceiptRepositoryStub{}),
	)
	router := gin.New()
	router.GET("/admin/billing/receipts", handler.ListBillingReceipts)

	req := httptest.NewRequest(http.MethodGet, "/admin/billing/receipts?user_id=bad", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}
