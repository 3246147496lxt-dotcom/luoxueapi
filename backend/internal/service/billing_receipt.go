package service

import (
	"context"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	BillingReceiptSourceAPI     = "api"
	BillingReceiptSourceWebChat = "web_chat"

	BillingReceiptStatusCharged      = "charged"
	BillingReceiptStatusNotCharged   = "not_charged"
	BillingReceiptStatusSubscription = "subscription"
	BillingReceiptStatusPending      = "pending"
	BillingReceiptStatusFailed       = "failed"
)

var (
	ErrBillingReceiptNotFound = infraerrors.NotFound(
		"BILLING_RECEIPT_NOT_FOUND",
		"billing receipt not found",
	)
	ErrBillingReceiptInvalidRequestID = infraerrors.BadRequest(
		"BILLING_RECEIPT_INVALID_REQUEST_ID",
		"billing receipt request id must be a UUID",
	)
	ErrBillingReceiptInvalidFilter = infraerrors.BadRequest(
		"BILLING_RECEIPT_INVALID_FILTER",
		"invalid billing receipt filter",
	)
)

// BillingReceipt is the immutable server-side charge snapshot for one request.
// UsageLogID may be resolved dynamically because usage_logs can be persisted
// after the atomic charge transaction commits.
type BillingReceipt struct {
	ID                  int64     `json:"id"`
	UsageLogID          *int64    `json:"usage_log_id,omitempty"`
	RequestID           string    `json:"request_id"`
	Source              string    `json:"source"`
	UserID              int64     `json:"user_id"`
	UserEmail           string    `json:"user_email,omitempty"`
	APIKeyID            int64     `json:"api_key_id"`
	AccountID           *int64    `json:"account_id,omitempty"`
	SubscriptionID      *int64    `json:"subscription_id,omitempty"`
	BillingType         int8      `json:"billing_type"`
	Model               string    `json:"model"`
	RequestedModel      string    `json:"requested_model"`
	InputTokens         int       `json:"input_tokens"`
	OutputTokens        int       `json:"output_tokens"`
	CacheCreationTokens int       `json:"cache_creation_tokens"`
	CacheReadTokens     int       `json:"cache_read_tokens"`
	GrossAmount         float64   `json:"gross_amount"`
	ChargedAmount       float64   `json:"charged_amount"`
	BalanceBefore       *float64  `json:"balance_before,omitempty"`
	BalanceAfter        *float64  `json:"balance_after,omitempty"`
	Status              string    `json:"status"`
	Overdraft           bool      `json:"overdraft"`
	FailureCode         *string   `json:"failure_code,omitempty"`
	FailureReason       *string   `json:"failure_reason,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
}

type BillingReceiptFilter struct {
	Page     int
	PageSize int

	StartTime *time.Time
	EndTime   *time.Time
	UserID    *int64
	Model     string
	RequestID string
	Status    string
	Source    string
}

type BillingReceiptList struct {
	Receipts []*BillingReceipt `json:"receipts"`
	Total    int               `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
}

type BillingReceiptRepository interface {
	GetUserWebChatReceipt(ctx context.Context, userID int64, requestID string) (*BillingReceipt, error)
	ListBillingReceipts(ctx context.Context, filter *BillingReceiptFilter) (*BillingReceiptList, error)
}

type BillingReceiptService struct {
	repo BillingReceiptRepository
}

func NewBillingReceiptService(repo BillingReceiptRepository) *BillingReceiptService {
	return &BillingReceiptService{repo: repo}
}

func (s *BillingReceiptService) GetUserWebChatReceipt(ctx context.Context, userID int64, requestID string) (*BillingReceipt, error) {
	return s.repo.GetUserWebChatReceipt(ctx, userID, requestID)
}

func (s *BillingReceiptService) ListBillingReceipts(ctx context.Context, filter *BillingReceiptFilter) (*BillingReceiptList, error) {
	normalized, err := normalizeBillingReceiptFilter(filter)
	if err != nil {
		return nil, err
	}
	return s.repo.ListBillingReceipts(ctx, normalized)
}

func normalizeBillingReceiptFilter(filter *BillingReceiptFilter) (*BillingReceiptFilter, error) {
	normalized := &BillingReceiptFilter{}
	if filter != nil {
		*normalized = *filter
	}

	normalized.Model = strings.TrimSpace(normalized.Model)
	normalized.RequestID = strings.TrimSpace(normalized.RequestID)
	normalized.Status = strings.ToLower(strings.TrimSpace(normalized.Status))
	normalized.Source = strings.ToLower(strings.TrimSpace(normalized.Source))

	switch normalized.Status {
	case "",
		BillingReceiptStatusCharged,
		BillingReceiptStatusNotCharged,
		BillingReceiptStatusSubscription,
		BillingReceiptStatusPending,
		BillingReceiptStatusFailed:
	default:
		return nil, ErrBillingReceiptInvalidFilter
	}
	switch normalized.Source {
	case "", BillingReceiptSourceAPI, BillingReceiptSourceWebChat:
	default:
		return nil, ErrBillingReceiptInvalidFilter
	}
	if normalized.StartTime != nil && normalized.EndTime != nil && normalized.StartTime.After(*normalized.EndTime) {
		return nil, ErrBillingReceiptInvalidFilter
	}
	return normalized, nil
}
