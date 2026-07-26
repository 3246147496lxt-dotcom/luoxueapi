package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
)

type billingReceiptRepository struct {
	db *sql.DB
}

func NewBillingReceiptRepository(db *sql.DB) service.BillingReceiptRepository {
	return &billingReceiptRepository{db: db}
}

const billingReceiptSelectColumns = `
	e.id,
	COALESCE(
		e.usage_log_id,
		(
			SELECT ul.id
			FROM usage_logs ul
			WHERE ul.request_id = e.request_id
			  AND ul.api_key_id = e.api_key_id
			LIMIT 1
		)
	) AS usage_log_id,
	COALESCE(e.request_id, ''),
	e.source,
	e.user_id,
	u.email,
	e.api_key_id,
	e.account_id,
	e.subscription_id,
	e.billing_type,
	e.model,
	CASE WHEN e.requested_model = '' THEN e.model ELSE e.requested_model END,
	e.input_tokens,
	e.output_tokens,
	e.cache_creation_tokens,
	e.cache_read_tokens,
	e.gross_amount::double precision,
	e.charged_amount::double precision,
	e.balance_before::double precision,
	e.balance_after::double precision,
	e.status,
	e.overdraft,
	NULL::text AS failure_code,
	NULL::text AS failure_reason,
	e.created_at`

const billingReceiptAdminProjection = `
	WITH receipt_rows AS (
		SELECT
			e.id,
			COALESCE(
				e.usage_log_id,
				(
					SELECT ul.id
					FROM usage_logs ul
					WHERE ul.request_id = e.request_id
					  AND ul.api_key_id = e.api_key_id
					LIMIT 1
				)
			) AS usage_log_id,
			e.request_id,
			e.source,
			e.user_id,
			u.email AS user_email,
			e.api_key_id,
			e.account_id,
			e.subscription_id,
			e.billing_type,
			e.model,
			CASE WHEN e.requested_model = '' THEN e.model ELSE e.requested_model END AS requested_model,
			e.input_tokens,
			e.output_tokens,
			e.cache_creation_tokens,
			e.cache_read_tokens,
			e.gross_amount::double precision AS gross_amount,
			e.charged_amount::double precision AS charged_amount,
			e.balance_before::double precision AS balance_before,
			e.balance_after::double precision AS balance_after,
			e.status,
			e.overdraft,
			NULL::text AS failure_code,
			NULL::text AS failure_reason,
			e.created_at
		FROM billing_usage_entries e
		JOIN users u ON u.id = e.user_id
		WHERE e.request_id IS NOT NULL
	),
	attempt_rows AS (
		SELECT
			0::bigint AS id,
			ul.id AS usage_log_id,
			'client:' || a.client_request_id AS request_id,
			'web_chat'::varchar AS source,
			a.user_id,
			u.email AS user_email,
			COALESCE(ul.api_key_id, 0)::bigint AS api_key_id,
			ul.account_id,
			ul.subscription_id,
			COALESCE(ul.billing_type, 0)::smallint AS billing_type,
			COALESCE(ul.model, '')::varchar AS model,
			COALESCE(NULLIF(ul.requested_model, ''), ul.model, '')::varchar AS requested_model,
			COALESCE(ul.input_tokens, 0)::integer AS input_tokens,
			COALESCE(ul.output_tokens, 0)::integer AS output_tokens,
			COALESCE(ul.cache_creation_tokens, 0)::integer AS cache_creation_tokens,
			COALESCE(ul.cache_read_tokens, 0)::integer AS cache_read_tokens,
			COALESCE(ul.actual_cost, 0)::double precision AS gross_amount,
			0::double precision AS charged_amount,
			NULL::double precision AS balance_before,
			NULL::double precision AS balance_after,
			CASE
				WHEN ul.id IS NOT NULL THEN 'not_charged'
				WHEN a.status = 'failed' THEN 'failed'
				WHEN a.status IN ('completed', 'interrupted') THEN 'not_charged'
				ELSE 'pending'
			END::varchar AS status,
			FALSE AS overdraft,
			CASE WHEN ul.id IS NULL THEN a.failure_code END::text AS failure_code,
			CASE WHEN ul.id IS NULL THEN a.failure_reason END::text AS failure_reason,
			a.created_at
		FROM chat_request_attempts a
		JOIN users u ON u.id = a.user_id
		LEFT JOIN LATERAL (
			SELECT
				log.id,
				log.api_key_id,
				log.account_id,
				log.subscription_id,
				log.billing_type,
				log.model,
				log.requested_model,
				log.input_tokens,
				log.output_tokens,
				log.cache_creation_tokens,
				log.cache_read_tokens,
				log.actual_cost
			FROM usage_logs log
			JOIN api_keys ak
			  ON ak.id = log.api_key_id
			 AND ak.user_id = a.user_id
			 AND ak.purpose = 'web_chat'
			 AND ak.deleted_at IS NOT NULL
			WHERE log.user_id = a.user_id
			  AND log.request_id = 'client:' || a.client_request_id
			LIMIT 1
		) ul ON TRUE
		WHERE NOT EXISTS (
			SELECT 1
			FROM billing_usage_entries e
			WHERE e.request_id = 'client:' || a.client_request_id
			  AND e.user_id = a.user_id
		)
	),
	all_billing_rows AS (
		SELECT * FROM receipt_rows
		UNION ALL
		SELECT * FROM attempt_rows
	)`

const billingReceiptAdminSelectColumns = `
	r.id,
	r.usage_log_id,
	r.request_id,
	r.source,
	r.user_id,
	r.user_email,
	r.api_key_id,
	r.account_id,
	r.subscription_id,
	r.billing_type,
	r.model,
	r.requested_model,
	r.input_tokens,
	r.output_tokens,
	r.cache_creation_tokens,
	r.cache_read_tokens,
	r.gross_amount,
	r.charged_amount,
	r.balance_before,
	r.balance_after,
	r.status,
	r.overdraft,
	r.failure_code,
	r.failure_reason,
	r.created_at`

type billingReceiptScanner func(dest ...any) error

func scanBillingReceipt(scan billingReceiptScanner) (*service.BillingReceipt, error) {
	var (
		receipt       service.BillingReceipt
		usageLogID    sql.NullInt64
		accountID     sql.NullInt64
		subscription  sql.NullInt64
		balanceBefore sql.NullFloat64
		balanceAfter  sql.NullFloat64
		failureCode   sql.NullString
		failureReason sql.NullString
	)
	if err := scan(
		&receipt.ID,
		&usageLogID,
		&receipt.RequestID,
		&receipt.Source,
		&receipt.UserID,
		&receipt.UserEmail,
		&receipt.APIKeyID,
		&accountID,
		&subscription,
		&receipt.BillingType,
		&receipt.Model,
		&receipt.RequestedModel,
		&receipt.InputTokens,
		&receipt.OutputTokens,
		&receipt.CacheCreationTokens,
		&receipt.CacheReadTokens,
		&receipt.GrossAmount,
		&receipt.ChargedAmount,
		&balanceBefore,
		&balanceAfter,
		&receipt.Status,
		&receipt.Overdraft,
		&failureCode,
		&failureReason,
		&receipt.CreatedAt,
	); err != nil {
		return nil, err
	}
	if usageLogID.Valid {
		value := usageLogID.Int64
		receipt.UsageLogID = &value
	}
	if accountID.Valid {
		value := accountID.Int64
		receipt.AccountID = &value
	}
	if subscription.Valid {
		value := subscription.Int64
		receipt.SubscriptionID = &value
	}
	if balanceBefore.Valid {
		value := balanceBefore.Float64
		receipt.BalanceBefore = &value
	}
	if balanceAfter.Valid {
		value := balanceAfter.Float64
		receipt.BalanceAfter = &value
	}
	if failureCode.Valid {
		value := failureCode.String
		receipt.FailureCode = &value
	}
	if failureReason.Valid {
		value := failureReason.String
		receipt.FailureReason = &value
	}
	return &receipt, nil
}

func (r *billingReceiptRepository) GetUserWebChatReceipt(
	ctx context.Context,
	userID int64,
	requestID string,
) (*service.BillingReceipt, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("billing receipt repository db is nil")
	}

	canonicalRequestID, err := canonicalizeWebChatReceiptRequestID(requestID)
	if err != nil {
		return nil, err
	}
	row := r.db.QueryRowContext(ctx, `
		SELECT `+billingReceiptSelectColumns+`
		FROM billing_usage_entries e
		JOIN users u ON u.id = e.user_id
		JOIN api_keys ak ON ak.id = e.api_key_id
		WHERE e.request_id = $1
		  AND e.user_id = $2
		  AND e.source = $3
		  AND ak.user_id = $2
		  AND ak.purpose = $4
		  AND ak.deleted_at IS NOT NULL
		LIMIT 1
	`, canonicalRequestID, userID, service.BillingReceiptSourceWebChat, service.APIKeyPurposeWebChat)

	receipt, err := scanBillingReceipt(row.Scan)
	if err == nil {
		return receipt, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	receipt, err = r.getUserWebChatUsageReceipt(ctx, userID, canonicalRequestID)
	if err == nil {
		return receipt, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	receipt, err = r.getUserWebChatAttemptReceipt(ctx, userID, canonicalRequestID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrBillingReceiptNotFound
	}
	return receipt, err
}

func (r *billingReceiptRepository) getUserWebChatUsageReceipt(
	ctx context.Context,
	userID int64,
	canonicalRequestID string,
) (*service.BillingReceipt, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT
			0::bigint,
			ul.id,
			ul.request_id,
			$3::varchar,
			ul.user_id,
			u.email,
			ul.api_key_id,
			ul.account_id,
			ul.subscription_id,
			ul.billing_type,
			ul.model,
			COALESCE(NULLIF(ul.requested_model, ''), ul.model),
			ul.input_tokens,
			ul.output_tokens,
			ul.cache_creation_tokens,
			ul.cache_read_tokens,
			ul.actual_cost::double precision,
			0::double precision,
			NULL::double precision,
			NULL::double precision,
			$4::varchar,
			FALSE,
			NULL::text,
			NULL::text,
			ul.created_at
		FROM usage_logs ul
		JOIN users u ON u.id = ul.user_id
		JOIN api_keys ak ON ak.id = ul.api_key_id
		WHERE ul.request_id = $1
		  AND ul.user_id = $2
		  AND ak.user_id = $2
		  AND ak.purpose = $5
		  AND ak.deleted_at IS NOT NULL
		LIMIT 1
	`,
		canonicalRequestID,
		userID,
		service.BillingReceiptSourceWebChat,
		service.BillingReceiptStatusNotCharged,
		service.APIKeyPurposeWebChat,
	)
	return scanBillingReceipt(row.Scan)
}

func (r *billingReceiptRepository) getUserWebChatAttemptReceipt(
	ctx context.Context,
	userID int64,
	canonicalRequestID string,
) (*service.BillingReceipt, error) {
	clientRequestID := strings.TrimPrefix(canonicalRequestID, "client:")
	row := r.db.QueryRowContext(ctx, `
		SELECT
			0::bigint,
			NULL::bigint,
			$3::varchar,
			$4::varchar,
			a.user_id,
			u.email,
			0::bigint,
			NULL::bigint,
			NULL::bigint,
			0::smallint,
			''::varchar,
			''::varchar,
			0::integer,
			0::integer,
			0::integer,
			0::integer,
			0::double precision,
			0::double precision,
			NULL::double precision,
			NULL::double precision,
			CASE
				WHEN a.status = 'failed' THEN $5::varchar
				WHEN a.status IN ('completed', 'interrupted') THEN $6::varchar
				ELSE $7::varchar
			END,
			FALSE,
			a.failure_code::text,
			a.failure_reason::text,
			a.created_at
		FROM chat_request_attempts a
		JOIN users u ON u.id = a.user_id
		WHERE a.user_id = $1
		  AND a.client_request_id = $2
		LIMIT 1
	`,
		userID,
		clientRequestID,
		canonicalRequestID,
		service.BillingReceiptSourceWebChat,
		service.BillingReceiptStatusFailed,
		service.BillingReceiptStatusNotCharged,
		service.BillingReceiptStatusPending,
	)
	return scanBillingReceipt(row.Scan)
}

func canonicalizeWebChatReceiptRequestID(requestID string) (string, error) {
	value := strings.TrimSpace(requestID)
	if strings.HasPrefix(value, "client:") {
		value = strings.TrimSpace(strings.TrimPrefix(value, "client:"))
	}
	parsed, err := uuid.Parse(value)
	if err != nil {
		return "", service.ErrBillingReceiptInvalidRequestID
	}
	return "client:" + parsed.String(), nil
}

func (r *billingReceiptRepository) ListBillingReceipts(
	ctx context.Context,
	filter *service.BillingReceiptFilter,
) (*service.BillingReceiptList, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("billing receipt repository db is nil")
	}
	if filter == nil {
		filter = &service.BillingReceiptFilter{}
	}

	page := filter.Page
	if page <= 0 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}

	where, args := buildBillingReceiptWhere(filter)
	var total int
	if err := r.db.QueryRowContext(
		ctx,
		billingReceiptAdminProjection+"\nSELECT COUNT(*) FROM all_billing_rows r "+where,
		args...,
	).Scan(&total); err != nil {
		return nil, err
	}

	queryArgs := append(append([]any(nil), args...), pageSize, (page-1)*pageSize)
	query := billingReceiptAdminProjection + `
		SELECT ` + billingReceiptAdminSelectColumns + `
		FROM all_billing_rows r
		` + where + `
		ORDER BY r.created_at DESC, r.id DESC
		LIMIT $` + strconv.Itoa(len(args)+1) + ` OFFSET $` + strconv.Itoa(len(args)+2)
	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	receipts := make([]*service.BillingReceipt, 0, pageSize)
	for rows.Next() {
		receipt, err := scanBillingReceipt(rows.Scan)
		if err != nil {
			return nil, err
		}
		receipts = append(receipts, receipt)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &service.BillingReceiptList{
		Receipts: receipts,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func buildBillingReceiptWhere(filter *service.BillingReceiptFilter) (string, []any) {
	clauses := []string{"TRUE"}
	args := make([]any, 0, 7)
	add := func(clause string, value any) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf(clause, len(args)))
	}

	if filter.StartTime != nil {
		add("r.created_at >= $%d", filter.StartTime.UTC())
	}
	if filter.EndTime != nil {
		add("r.created_at <= $%d", filter.EndTime.UTC())
	}
	if filter.UserID != nil {
		add("r.user_id = $%d", *filter.UserID)
	}
	if model := strings.TrimSpace(filter.Model); model != "" {
		add("(LOWER(r.model) = LOWER($%[1]d) OR LOWER(r.requested_model) = LOWER($%[1]d))", model)
	}
	if requestID := normalizeAdminBillingReceiptRequestID(filter.RequestID); requestID != "" {
		add("r.request_id = $%d", requestID)
	}
	if status := strings.ToLower(strings.TrimSpace(filter.Status)); status != "" {
		add("r.status = $%d", status)
	}
	if source := strings.ToLower(strings.TrimSpace(filter.Source)); source != "" {
		add("r.source = $%d", source)
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func normalizeAdminBillingReceiptRequestID(requestID string) string {
	value := strings.TrimSpace(requestID)
	if value == "" || strings.Contains(value, ":") {
		return value
	}
	if parsed, err := uuid.Parse(value); err == nil {
		return "client:" + parsed.String()
	}
	return value
}
