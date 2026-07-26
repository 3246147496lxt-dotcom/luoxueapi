package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type adminChatHistoryRepository struct {
	db *sql.DB
}

func NewAdminChatHistoryRepository(db *sql.DB) service.AdminChatHistoryRepository {
	return &adminChatHistoryRepository{db: db}
}

const adminChatConversationRefsBaseQuery = `
SELECT
    c.id,
    c.public_id,
    COALESCE(c.model, ''),
    (SELECT COUNT(*) FROM chat_messages m WHERE m.conversation_id = c.id),
    'active',
    c.created_at,
    c.updated_at
FROM chat_conversations c
WHERE c.user_id = $1
  AND c.deleted_at IS NULL`

func (r *adminChatHistoryRepository) ListUserConversationRefs(
	ctx context.Context,
	query service.AdminChatConversationListQuery,
) (*service.AdminChatConversationRefPage, error) {
	if r == nil || r.db == nil {
		return nil, service.ErrAdminChatHistoryUnavailable
	}

	args := []any{query.TargetUserID}
	sqlQuery := adminChatConversationRefsBaseQuery
	if query.BeforeUpdatedAt != nil && query.BeforeID != nil {
		args = append(args, query.BeforeUpdatedAt.UTC(), *query.BeforeID)
		sqlQuery += `
  AND (c.updated_at, c.id) < ($2, $3)`
	}
	args = append(args, query.Limit+1)
	sqlQuery += fmt.Sprintf(`
ORDER BY c.updated_at DESC, c.id DESC
LIMIT $%d`, len(args))

	rows, err := r.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, service.ErrAdminChatHistoryUnavailable.WithCause(
			fmt.Errorf("list administrator chat conversation references: %w", err),
		)
	}
	defer func() { _ = rows.Close() }()

	type rowItem struct {
		internalID int64
		item       service.AdminChatConversationRef
	}
	items := make([]rowItem, 0, query.Limit+1)
	for rows.Next() {
		var item rowItem
		if err := rows.Scan(
			&item.internalID,
			&item.item.ID,
			&item.item.Model,
			&item.item.MessageCount,
			&item.item.Status,
			&item.item.CreatedAt,
			&item.item.UpdatedAt,
		); err != nil {
			return nil, service.ErrAdminChatHistoryUnavailable.WithCause(
				fmt.Errorf("scan administrator chat conversation reference: %w", err),
			)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, service.ErrAdminChatHistoryUnavailable.WithCause(
			fmt.Errorf("iterate administrator chat conversation references: %w", err),
		)
	}

	page := &service.AdminChatConversationRefPage{
		Items: make([]service.AdminChatConversationRef, 0, min(len(items), query.Limit)),
	}
	if len(items) > query.Limit {
		page.HasMore = true
		items = items[:query.Limit]
	}
	for _, item := range items {
		page.Items = append(page.Items, item.item)
	}
	if page.HasMore && len(items) > 0 {
		last := items[len(items)-1]
		updatedAt := last.item.UpdatedAt
		internalID := last.internalID
		page.NextUpdatedAt = &updatedAt
		page.NextID = &internalID
	}
	return page, nil
}

const adminChatConversationContentQuery = `
SELECT
    c.id,
    c.public_id,
    COALESCE(c.title, ''),
    COALESCE(c.model, ''),
    'active',
    c.created_at,
    c.updated_at
FROM chat_conversations c
WHERE c.user_id = $1
  AND c.public_id = $2
  AND c.deleted_at IS NULL
FOR SHARE`

const adminChatMessagesBaseQuery = `
SELECT
    m.public_id,
    m.position,
    m.role,
    m.content,
    m.delivery_status,
    COALESCE(m.requested_model, ''),
    COALESCE(m.finish_reason, ''),
    COALESCE(m.error_code, ''),
    COALESCE(m.error_message, ''),
    COALESCE(a.attempt_id, ''),
    COALESCE(a.client_request_id, ''),
    m.created_at,
    m.terminal_at
FROM chat_messages m
LEFT JOIN chat_request_attempts a
  ON a.assistant_message_id = m.id
WHERE m.conversation_id = $1`

const adminChatContentAccessInsert = `
INSERT INTO chat_admin_content_access_logs (
    admin_id,
    target_user_id,
    conversation_id,
    conversation_public_id,
    request_id,
    client_ip,
    user_agent,
    before_position,
    page_limit
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

func (r *adminChatHistoryRepository) ViewConversationPageAndRecordAccess(
	ctx context.Context,
	query service.AdminChatContentAccessQuery,
) (*service.AdminChatConversationPage, error) {
	if r == nil || r.db == nil {
		return nil, service.ErrAdminChatContentAuditUnavailable
	}
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, service.ErrAdminChatHistoryUnavailable.WithCause(
			fmt.Errorf("begin administrator chat content read: %w", err),
		)
	}
	defer func() { _ = tx.Rollback() }()

	var conversationInternalID int64
	page := &service.AdminChatConversationPage{}
	err = tx.QueryRowContext(
		ctx,
		adminChatConversationContentQuery,
		query.TargetUserID,
		query.ConversationPublicID,
	).Scan(
		&conversationInternalID,
		&page.Conversation.ID,
		&page.Conversation.Title,
		&page.Conversation.Model,
		&page.Conversation.Status,
		&page.Conversation.CreatedAt,
		&page.Conversation.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrAdminChatConversationNotFound
	}
	if err != nil {
		return nil, service.ErrAdminChatHistoryUnavailable.WithCause(
			fmt.Errorf("read administrator chat conversation: %w", err),
		)
	}

	messageQuery := adminChatMessagesBaseQuery
	messageArgs := []any{conversationInternalID}
	if query.BeforePosition != nil {
		messageArgs = append(messageArgs, *query.BeforePosition)
		messageQuery += `
  AND m.position < $2`
	}
	messageArgs = append(messageArgs, query.Limit+1)
	messageQuery += fmt.Sprintf(`
ORDER BY m.position DESC, m.id DESC
LIMIT $%d`, len(messageArgs))

	rows, err := tx.QueryContext(ctx, messageQuery, messageArgs...)
	if err != nil {
		return nil, service.ErrAdminChatHistoryUnavailable.WithCause(
			fmt.Errorf("read administrator chat messages: %w", err),
		)
	}
	messages := make([]service.AdminChatMessage, 0, query.Limit+1)
	for rows.Next() {
		var message service.AdminChatMessage
		if err := rows.Scan(
			&message.ID,
			&message.Position,
			&message.Role,
			&message.Content,
			&message.Status,
			&message.RequestedModel,
			&message.FinishReason,
			&message.ErrorCode,
			&message.ErrorMessage,
			&message.AttemptID,
			&message.ReceiptID,
			&message.CreatedAt,
			&message.CompletedAt,
		); err != nil {
			_ = rows.Close()
			return nil, service.ErrAdminChatHistoryUnavailable.WithCause(
				fmt.Errorf("scan administrator chat message: %w", err),
			)
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, service.ErrAdminChatHistoryUnavailable.WithCause(
			fmt.Errorf("iterate administrator chat messages: %w", err),
		)
	}
	if err := rows.Close(); err != nil {
		return nil, service.ErrAdminChatHistoryUnavailable.WithCause(
			fmt.Errorf("close administrator chat message rows: %w", err),
		)
	}

	if len(messages) > query.Limit {
		page.HasMore = true
		messages = messages[:query.Limit]
	}
	if page.HasMore && len(messages) > 0 {
		nextBeforePosition := messages[len(messages)-1].Position
		page.NextBeforePosition = &nextBeforePosition
	}
	for left, right := 0, len(messages)-1; left < right; left, right = left+1, right-1 {
		messages[left], messages[right] = messages[right], messages[left]
	}
	page.Messages = messages

	if _, err := tx.ExecContext(
		ctx,
		adminChatContentAccessInsert,
		query.AdminID,
		query.TargetUserID,
		conversationInternalID,
		query.ConversationPublicID,
		query.RequestID,
		query.ClientIP,
		query.UserAgent,
		query.BeforePosition,
		query.Limit,
	); err != nil {
		return nil, service.ErrAdminChatContentAuditUnavailable.WithCause(
			fmt.Errorf("record administrator chat content access: %w", err),
		)
	}
	if err := tx.Commit(); err != nil {
		return nil, service.ErrAdminChatContentAuditUnavailable.WithCause(
			fmt.Errorf("commit administrator chat content access: %w", err),
		)
	}
	return page, nil
}
