package service

import (
	"context"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
)

// PaymentOrderView is the transport-neutral projection exposed to HTTP and
// other delivery adapters. Keeping the Ent entity behind this boundary avoids
// coupling handlers to persistence details such as provider_snapshot and
// generated edge fields.
type PaymentOrderView struct {
	ID                  int64
	UserID              int64
	UserEmail           string
	UserName            string
	UserNotes           *string
	Amount              float64
	PayAmount           float64
	FeeRate             float64
	Currency            string
	RechargeCode        string
	OutTradeNo          string
	PaymentType         string
	PaymentTradeNo      string
	PayURL              *string
	QRCode              *string
	QRCodeImg           *string
	OrderType           string
	PlanID              *int64
	SubscriptionGroupID *int64
	SubscriptionDays    *int
	ProviderInstanceID  *string
	ProviderKey         *string
	Status              string
	RefundAmount        float64
	RefundReason        *string
	RefundAt            *time.Time
	ForceRefund         bool
	RefundRequestedAt   *time.Time
	RefundRequestReason *string
	RefundRequestedBy   *string
	ExpiresAt           time.Time
	PaidAt              *time.Time
	CompletedAt         *time.Time
	FailedAt            *time.Time
	FailedReason        *string
	ClientIP            string
	SrcHost             string
	SrcURL              *string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// PaymentAuditLogView preserves the existing JSON contract without exposing
// the generated Ent audit-log entity to the admin handler.
type PaymentAuditLogView struct {
	ID        int64     `json:"id,omitempty"`
	OrderID   string    `json:"order_id,omitempty"`
	Action    string    `json:"action,omitempty"`
	Detail    string    `json:"detail,omitempty"`
	Operator  string    `json:"operator,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

func newPaymentOrderView(order *dbent.PaymentOrder) *PaymentOrderView {
	if order == nil {
		return nil
	}
	return &PaymentOrderView{
		ID:                  order.ID,
		UserID:              order.UserID,
		UserEmail:           order.UserEmail,
		UserName:            order.UserName,
		UserNotes:           order.UserNotes,
		Amount:              order.Amount,
		PayAmount:           order.PayAmount,
		FeeRate:             order.FeeRate,
		Currency:            PaymentOrderCurrency(order),
		RechargeCode:        order.RechargeCode,
		OutTradeNo:          order.OutTradeNo,
		PaymentType:         order.PaymentType,
		PaymentTradeNo:      order.PaymentTradeNo,
		PayURL:              order.PayURL,
		QRCode:              order.QrCode,
		QRCodeImg:           order.QrCodeImg,
		OrderType:           order.OrderType,
		PlanID:              order.PlanID,
		SubscriptionGroupID: order.SubscriptionGroupID,
		SubscriptionDays:    order.SubscriptionDays,
		ProviderInstanceID:  order.ProviderInstanceID,
		ProviderKey:         order.ProviderKey,
		Status:              order.Status,
		RefundAmount:        order.RefundAmount,
		RefundReason:        order.RefundReason,
		RefundAt:            order.RefundAt,
		ForceRefund:         order.ForceRefund,
		RefundRequestedAt:   order.RefundRequestedAt,
		RefundRequestReason: order.RefundRequestReason,
		RefundRequestedBy:   order.RefundRequestedBy,
		ExpiresAt:           order.ExpiresAt,
		PaidAt:              order.PaidAt,
		CompletedAt:         order.CompletedAt,
		FailedAt:            order.FailedAt,
		FailedReason:        order.FailedReason,
		ClientIP:            order.ClientIP,
		SrcHost:             order.SrcHost,
		SrcURL:              order.SrcURL,
		CreatedAt:           order.CreatedAt,
		UpdatedAt:           order.UpdatedAt,
	}
}

func newPaymentOrderViews(orders []*dbent.PaymentOrder) []*PaymentOrderView {
	views := make([]*PaymentOrderView, 0, len(orders))
	for _, order := range orders {
		views = append(views, newPaymentOrderView(order))
	}
	return views
}

// GetOrderView returns a user-owned order without leaking the persistence
// entity across the service boundary.
func (s *PaymentService) GetOrderView(ctx context.Context, orderID, userID int64) (*PaymentOrderView, error) {
	order, err := s.GetOrder(ctx, orderID, userID)
	if err != nil {
		return nil, err
	}
	return newPaymentOrderView(order), nil
}

// GetOrderByIDView returns an admin order projection.
func (s *PaymentService) GetOrderByIDView(ctx context.Context, orderID int64) (*PaymentOrderView, error) {
	order, err := s.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	return newPaymentOrderView(order), nil
}

// GetUserOrderViews returns paginated user order projections.
func (s *PaymentService) GetUserOrderViews(ctx context.Context, userID int64, params OrderListParams) ([]*PaymentOrderView, int, error) {
	orders, total, err := s.GetUserOrders(ctx, userID, params)
	if err != nil {
		return nil, 0, err
	}
	return newPaymentOrderViews(orders), total, nil
}

// AdminListOrderViews returns paginated admin order projections.
func (s *PaymentService) AdminListOrderViews(ctx context.Context, userID int64, params OrderListParams) ([]*PaymentOrderView, int, error) {
	orders, total, err := s.AdminListOrders(ctx, userID, params)
	if err != nil {
		return nil, 0, err
	}
	return newPaymentOrderViews(orders), total, nil
}

// VerifyOrderByOutTradeNoView verifies and projects a user-owned order.
func (s *PaymentService) VerifyOrderByOutTradeNoView(ctx context.Context, outTradeNo string, userID int64) (*PaymentOrderView, error) {
	order, err := s.VerifyOrderByOutTradeNo(ctx, outTradeNo, userID)
	if err != nil {
		return nil, err
	}
	return newPaymentOrderView(order), nil
}

// VerifyOrderPublicView verifies and projects the legacy anonymous order view.
func (s *PaymentService) VerifyOrderPublicView(ctx context.Context, outTradeNo string) (*PaymentOrderView, error) {
	order, err := s.VerifyOrderPublic(ctx, outTradeNo)
	if err != nil {
		return nil, err
	}
	return newPaymentOrderView(order), nil
}

// GetPublicOrderByResumeTokenView resolves and projects a signed checkout
// resume token without returning the underlying Ent entity.
func (s *PaymentService) GetPublicOrderByResumeTokenView(ctx context.Context, token string) (*PaymentOrderView, error) {
	order, err := s.GetPublicOrderByResumeToken(ctx, token)
	if err != nil {
		return nil, err
	}
	return newPaymentOrderView(order), nil
}

// GetOrderAuditLogViews returns the existing audit-log JSON shape through a
// persistence-neutral DTO.
func (s *PaymentService) GetOrderAuditLogViews(ctx context.Context, orderID int64) ([]*PaymentAuditLogView, error) {
	logs, err := s.GetOrderAuditLogs(ctx, orderID)
	if err != nil {
		return nil, err
	}
	views := make([]*PaymentAuditLogView, 0, len(logs))
	for _, log := range logs {
		if log == nil {
			views = append(views, nil)
			continue
		}
		views = append(views, &PaymentAuditLogView{
			ID:        log.ID,
			OrderID:   log.OrderID,
			Action:    log.Action,
			Detail:    log.Detail,
			Operator:  log.Operator,
			CreatedAt: log.CreatedAt,
		})
	}
	return views, nil
}
