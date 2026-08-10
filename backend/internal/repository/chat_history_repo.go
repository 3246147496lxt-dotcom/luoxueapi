package repository

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type chatHistoryRepository struct {
	db *sql.DB
}

func NewChatHistoryRepository(db *sql.DB) service.ChatHistoryRepository {
	return &chatHistoryRepository{db: db}
}

type chatHistoryScanner func(dest ...any) error

const chatHistoryConversationColumns = `
	c.id,
	c.public_id,
	c.title,
	c.model,
	c.revision,
	c.version,
	hm.public_id,
	c.message_count,
	c.created_at,
	c.updated_at,
	c.deleted_at`

func scanChatHistoryConversation(scan chatHistoryScanner) (int64, *service.ChatHistoryConversation, error) {
	var (
		internalID    int64
		conversation  service.ChatHistoryConversation
		headMessageID sql.NullString
		deletedAt     sql.NullTime
	)
	if err := scan(
		&internalID,
		&conversation.ID,
		&conversation.Title,
		&conversation.Model,
		&conversation.Revision,
		&conversation.Version,
		&headMessageID,
		&conversation.MessageCount,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
		&deletedAt,
	); err != nil {
		return 0, nil, err
	}
	if headMessageID.Valid {
		value := headMessageID.String
		conversation.HeadMessageID = &value
	}
	if deletedAt.Valid {
		value := deletedAt.Time
		conversation.DeletedAt = &value
	}
	return internalID, &conversation, nil
}

func (r *chatHistoryRepository) CreateConversation(
	ctx context.Context,
	userID int64,
	input *service.CreateChatHistoryConversationInput,
) (*service.ChatHistoryConversation, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("chat history repository db is nil")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err = lockChatHistorySyncState(ctx, tx, userID); err != nil {
		return nil, err
	}

	var (
		existingID      int64
		existingHash    string
		existingDeleted sql.NullTime
	)
	err = tx.QueryRowContext(ctx, `
		SELECT id, create_hash, deleted_at
		FROM chat_conversations
		WHERE user_id = $1 AND public_id = $2
		FOR UPDATE
	`, userID, input.ID).Scan(&existingID, &existingHash, &existingDeleted)
	if err == nil {
		if existingHash != input.CreateHash || existingDeleted.Valid {
			return nil, service.ErrChatHistoryConflict
		}
		conversation, getErr := getChatHistoryConversationWith(ctx, tx, userID, input.ID, false)
		if getErr != nil {
			return nil, getErr
		}
		if err = tx.Commit(); err != nil {
			return nil, err
		}
		return conversation, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	if input.Imported {
		var importedCount int
		if err = tx.QueryRowContext(ctx, `
			SELECT COUNT(*)
			FROM chat_conversations
			WHERE user_id = $1 AND imported_at IS NOT NULL
		`, userID).Scan(&importedCount); err != nil {
			return nil, err
		}
		if importedCount >= service.MaxChatHistoryImportedConversations {
			return nil, service.ErrChatHistoryImportLimit
		}
	}

	version, err := bumpChatHistorySyncVersion(ctx, tx, userID)
	if err != nil {
		return nil, err
	}
	var conversationID int64
	var importedAt any
	if input.Imported {
		importedAt = time.Now().UTC()
	}
	if err = tx.QueryRowContext(ctx, `
		INSERT INTO chat_conversations (
			public_id,
			user_id,
			title,
			model,
			revision,
			version,
			create_hash,
			imported_at
		)
		VALUES ($1, $2, $3, $4, 1, $5, $6, $7)
		RETURNING id
	`,
		input.ID,
		userID,
		input.Title,
		input.Model,
		version,
		input.CreateHash,
		importedAt,
	).Scan(&conversationID); err != nil {
		return nil, err
	}

	var headMessageID *int64
	for i := range input.ImportedMessages {
		message := input.ImportedMessages[i]
		position := int64(i + 1)
		var messageID int64
		if err = tx.QueryRowContext(ctx, `
			INSERT INTO chat_messages (
				public_id,
				user_id,
				conversation_id,
				position,
				role,
				content,
				delivery_status,
				created_at,
				updated_at,
				terminal_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8, $8)
			RETURNING id
		`,
			message.ID,
			userID,
			conversationID,
			position,
			message.Role,
			message.Content,
			message.Status,
			message.CreatedAt,
		).Scan(&messageID); err != nil {
			return nil, err
		}
		headMessageID = &messageID
	}
	if _, err = tx.ExecContext(ctx, `
		UPDATE chat_conversations
		SET head_message_id = $2,
		    message_count = $3
		WHERE id = $1
	`, conversationID, headMessageID, len(input.ImportedMessages)); err != nil {
		return nil, err
	}
	if err = insertChatHistoryChange(
		ctx,
		tx,
		userID,
		version,
		"upsert",
		conversationID,
		input.ID,
		nil,
	); err != nil {
		return nil, err
	}

	conversation, err := getChatHistoryConversationWith(ctx, tx, userID, input.ID, false)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return conversation, nil
}

func (r *chatHistoryRepository) ListConversations(
	ctx context.Context,
	userID int64,
	cursor string,
	limit int,
	search string,
) (*service.ChatHistoryConversationList, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("chat history repository db is nil")
	}
	search = strings.TrimSpace(search)
	where := `c.user_id = $1 AND c.deleted_at IS NULL`
	args := []any{userID}
	if search != "" {
		args = append(args, escapeChatHistoryLike(search))
		searchParam := strconv.Itoa(len(args))
		where += ` AND (
			c.title ILIKE '%' || $` + searchParam + ` || '%' ESCAPE '\'
			OR EXISTS (
				SELECT 1
				FROM chat_messages search_message
				WHERE search_message.conversation_id = c.id
				  AND search_message.content ILIKE '%' || $` + searchParam + ` || '%' ESCAPE '\'
			)
		)`
	}
	if strings.TrimSpace(cursor) != "" {
		cursorTime, cursorID, err := decodeChatHistoryConversationCursor(cursor)
		if err != nil {
			return nil, service.ErrChatHistoryInvalid
		}
		args = append(args, cursorTime, cursorID)
		timeParam := strconv.Itoa(len(args) - 1)
		idParam := strconv.Itoa(len(args))
		where += ` AND (c.updated_at, c.id) < ($` + timeParam + `, $` + idParam + `)`
	}

	queryArgs := append(append([]any(nil), args...), limit+1)
	limitParam := len(args) + 1
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+chatHistoryConversationColumns+`
		FROM chat_conversations c
		LEFT JOIN chat_messages hm ON hm.id = c.head_message_id
		WHERE `+where+`
		ORDER BY c.updated_at DESC, c.id DESC
		LIMIT $`+strconv.Itoa(limitParam),
		queryArgs...,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	type conversationWithInternalID struct {
		internalID   int64
		conversation *service.ChatHistoryConversation
	}
	scanned := make([]conversationWithInternalID, 0, limit+1)
	for rows.Next() {
		internalID, conversation, scanErr := scanChatHistoryConversation(rows.Scan)
		if scanErr != nil {
			return nil, scanErr
		}
		scanned = append(scanned, conversationWithInternalID{
			internalID:   internalID,
			conversation: conversation,
		})
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	hasMore := len(scanned) > limit
	if hasMore {
		scanned = scanned[:limit]
	}
	items := make([]*service.ChatHistoryConversation, len(scanned))
	for i := range scanned {
		items[i] = scanned[i].conversation
	}
	nextCursor := ""
	if hasMore && len(scanned) > 0 {
		last := scanned[len(scanned)-1]
		nextCursor = encodeChatHistoryConversationCursor(
			last.conversation.UpdatedAt,
			last.internalID,
		)
	}
	return &service.ChatHistoryConversationList{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func (r *chatHistoryRepository) GetConversation(
	ctx context.Context,
	userID int64,
	publicID string,
) (*service.ChatHistoryConversation, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("chat history repository db is nil")
	}
	return getChatHistoryConversationWith(ctx, r.db, userID, publicID, false)
}

func (r *chatHistoryRepository) ListMessages(
	ctx context.Context,
	userID int64,
	conversationPublicID string,
	beforePosition *int64,
	limit int,
) (*service.ChatHistoryMessagePage, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("chat history repository db is nil")
	}
	var conversationID int64
	if err := r.db.QueryRowContext(ctx, `
		SELECT id
		FROM chat_conversations
		WHERE user_id = $1
		  AND public_id = $2
		  AND deleted_at IS NULL
	`, userID, conversationPublicID).Scan(&conversationID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrChatHistoryNotFound
		}
		return nil, err
	}

	args := []any{userID, conversationID}
	beforeClause := ""
	if beforePosition != nil {
		args = append(args, *beforePosition)
		beforeClause = ` AND m.position < $3`
	}
	args = append(args, limit+1)
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			m.id,
			m.public_id,
			m.position,
			m.role,
			m.content,
			m.delivery_status,
			COALESCE(m.requested_model, ''),
			m.finish_reason,
			m.error_code,
			m.error_message,
			m.excluded_from_context,
			superseding.public_id,
			m.checkpoint_seq,
			a.attempt_id,
			a.client_request_id,
			m.created_at,
			m.updated_at,
			m.terminal_at
		FROM chat_messages m
		LEFT JOIN chat_request_attempts a ON a.assistant_message_id = m.id
		LEFT JOIN chat_messages superseding ON superseding.id = m.superseded_by_message_id
		WHERE m.user_id = $1
		  AND m.conversation_id = $2
		  `+beforeClause+`
		ORDER BY m.position DESC
		LIMIT $`+strconv.Itoa(len(args)),
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	descending := make([]*service.ChatHistoryMessage, 0, limit+1)
	messageInternalIDs := make([]int64, 0, limit+1)
	for rows.Next() {
		var internalID int64
		message, scanErr := scanChatHistoryMessage(func(dest ...any) error {
			return rows.Scan(append([]any{&internalID}, dest...)...)
		})
		if scanErr != nil {
			return nil, scanErr
		}
		descending = append(descending, message)
		messageInternalIDs = append(messageInternalIDs, internalID)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	hasMore := len(descending) > limit
	if hasMore {
		descending = descending[:limit]
		messageInternalIDs = messageInternalIDs[:limit]
	}
	attachmentsByMessage, err := loadChatAttachmentsForMessages(ctx, r.db, messageInternalIDs)
	if err != nil {
		return nil, err
	}
	for i := range descending {
		descending[i].Attachments = attachmentsByMessage[messageInternalIDs[i]]
	}
	items := make([]*service.ChatHistoryMessage, len(descending))
	for i := range descending {
		items[len(descending)-1-i] = descending[i]
	}
	var nextBefore *int64
	if hasMore && len(items) > 0 {
		value := items[0].Position
		nextBefore = &value
	}
	return &service.ChatHistoryMessagePage{
		Items:              items,
		NextBeforePosition: nextBefore,
		HasMore:            hasMore,
	}, nil
}

func (r *chatHistoryRepository) UpdateConversation(
	ctx context.Context,
	userID int64,
	input *service.UpdateChatHistoryConversationInput,
) (*service.ChatHistoryConversation, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("chat history repository db is nil")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = lockChatHistorySyncState(ctx, tx, userID); err != nil {
		return nil, err
	}

	var (
		conversationID int64
		title          string
		model          string
		revision       int64
	)
	if err = tx.QueryRowContext(ctx, `
		SELECT id, title, model, revision
		FROM chat_conversations
		WHERE user_id = $1
		  AND public_id = $2
		  AND deleted_at IS NULL
		FOR UPDATE
	`, userID, input.ID).Scan(&conversationID, &title, &model, &revision); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrChatHistoryNotFound
		}
		return nil, err
	}

	targetTitle := title
	if input.Title != nil {
		targetTitle = *input.Title
	}
	targetModel := model
	if input.Model != nil {
		targetModel = *input.Model
	}
	sameTarget := targetTitle == title && targetModel == model
	if input.Revision != revision && !sameTarget {
		return nil, service.ErrChatHistoryRevisionConflict.WithMetadata(map[string]string{
			"current_revision": strconv.FormatInt(revision, 10),
		})
	}
	if sameTarget {
		conversation, getErr := getChatHistoryConversationWith(ctx, tx, userID, input.ID, false)
		if getErr != nil {
			return nil, getErr
		}
		if err = tx.Commit(); err != nil {
			return nil, err
		}
		return conversation, nil
	}

	version, err := bumpChatHistorySyncVersion(ctx, tx, userID)
	if err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `
		UPDATE chat_conversations
		SET title = $3,
		    model = $4,
		    revision = revision + 1,
		    version = $5,
		    updated_at = NOW()
		WHERE user_id = $1 AND id = $2
	`, userID, conversationID, targetTitle, targetModel, version); err != nil {
		return nil, err
	}
	if err = insertChatHistoryChange(
		ctx,
		tx,
		userID,
		version,
		"upsert",
		conversationID,
		input.ID,
		nil,
	); err != nil {
		return nil, err
	}

	conversation, err := getChatHistoryConversationWith(ctx, tx, userID, input.ID, false)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return conversation, nil
}

func (r *chatHistoryRepository) DeleteConversation(
	ctx context.Context,
	userID int64,
	publicID string,
	revision int64,
) error {
	if r == nil || r.db == nil {
		return errors.New("chat history repository db is nil")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = lockChatHistorySyncState(ctx, tx, userID); err != nil {
		return err
	}

	var (
		conversationID  int64
		currentRevision int64
		deletedAt       sql.NullTime
	)
	if err = tx.QueryRowContext(ctx, `
		SELECT id, revision, deleted_at
		FROM chat_conversations
		WHERE user_id = $1 AND public_id = $2
		FOR UPDATE
	`, userID, publicID).Scan(&conversationID, &currentRevision, &deletedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return service.ErrChatHistoryNotFound
		}
		return err
	}
	if deletedAt.Valid {
		return tx.Commit()
	}
	if revision != currentRevision {
		return service.ErrChatHistoryRevisionConflict.WithMetadata(map[string]string{
			"current_revision": strconv.FormatInt(currentRevision, 10),
		})
	}

	version, err := bumpChatHistorySyncVersion(ctx, tx, userID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	// Mark first, then delete message associations. Blob deletion happens after
	// commit; retaining storage_key makes a failed immediate cleanup recoverable
	// by the attachment janitor.
	if _, err = tx.ExecContext(ctx, `
		UPDATE chat_attachments
		SET status='deleted', extracted_text=NULL, updated_at=$3
		WHERE user_id=$1 AND conversation_id=$2
	`, userID, conversationID, now); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `
		DELETE FROM chat_messages
		WHERE user_id = $1 AND conversation_id = $2
	`, userID, conversationID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `
		UPDATE chat_conversations
		SET title = '',
		    model = '',
		    revision = revision + 1,
		    version = $3,
		    head_message_id = NULL,
		    message_count = 0,
		    deleted_at = $4,
		    updated_at = $4
		WHERE user_id = $1 AND id = $2
	`, userID, conversationID, version, now); err != nil {
		return err
	}
	if err = insertChatHistoryChange(
		ctx,
		tx,
		userID,
		version,
		"deleted",
		conversationID,
		publicID,
		&now,
	); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *chatHistoryRepository) Sync(
	ctx context.Context,
	userID, afterVersion int64,
	limit int,
) (*service.ChatHistorySyncPage, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("chat history repository db is nil")
	}
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var latestVersion int64
	err = tx.QueryRowContext(ctx, `
		SELECT version
		FROM chat_history_sync_states
		WHERE user_id = $1
	`, userID).Scan(&latestVersion)
	if errors.Is(err, sql.ErrNoRows) {
		latestVersion = 0
	} else if err != nil {
		return nil, err
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			ch.version,
			ch.change_type,
			ch.conversation_public_id,
			ch.deleted_at,
			c.id,
			c.public_id,
			c.title,
			c.model,
			c.revision,
			c.version,
			hm.public_id,
			c.message_count,
			c.created_at,
			c.updated_at,
			c.deleted_at
		FROM chat_history_changes ch
		LEFT JOIN chat_conversations c ON c.id = ch.conversation_id
		LEFT JOIN chat_messages hm ON hm.id = c.head_message_id
		WHERE ch.user_id = $1
		  AND ch.version > $2
		ORDER BY ch.version ASC
		LIMIT $3
	`, userID, afterVersion, limit+1)
	if err != nil {
		return nil, err
	}
	changes := make([]*service.ChatHistorySyncChange, 0, limit+1)
	for rows.Next() {
		var (
			change         service.ChatHistorySyncChange
			changeDeleted  sql.NullTime
			internalID     sql.NullInt64
			conversation   service.ChatHistoryConversation
			conversationID sql.NullString
			title          sql.NullString
			model          sql.NullString
			revision       sql.NullInt64
			version        sql.NullInt64
			headID         sql.NullString
			messageCount   sql.NullInt64
			createdAt      sql.NullTime
			updatedAt      sql.NullTime
			deletedAt      sql.NullTime
		)
		if err = rows.Scan(
			&change.Version,
			&change.Type,
			&change.ConversationID,
			&changeDeleted,
			&internalID,
			&conversationID,
			&title,
			&model,
			&revision,
			&version,
			&headID,
			&messageCount,
			&createdAt,
			&updatedAt,
			&deletedAt,
		); err != nil {
			_ = rows.Close()
			return nil, err
		}
		if changeDeleted.Valid {
			value := changeDeleted.Time
			change.DeletedAt = &value
		}
		if change.Type == "upsert" && internalID.Valid && conversationID.Valid && !deletedAt.Valid {
			conversation.ID = conversationID.String
			conversation.Title = title.String
			conversation.Model = model.String
			conversation.Revision = revision.Int64
			conversation.Version = version.Int64
			conversation.MessageCount = int(messageCount.Int64)
			conversation.CreatedAt = createdAt.Time
			conversation.UpdatedAt = updatedAt.Time
			if headID.Valid {
				value := headID.String
				conversation.HeadMessageID = &value
			}
			change.Conversation = &conversation
		}
		changes = append(changes, &change)
	}
	if err = rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err = rows.Close(); err != nil {
		return nil, err
	}

	hasMore := len(changes) > limit
	if hasMore {
		changes = changes[:limit]
	}
	nextCursor := afterVersion
	if len(changes) > 0 {
		nextCursor = changes[len(changes)-1].Version
	}
	result := &service.ChatHistorySyncPage{
		Changes:       changes,
		LatestVersion: latestVersion,
		NextCursor:    strconv.FormatInt(nextCursor, 10),
		HasMore:       hasMore,
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *chatHistoryRepository) PrepareCompletion(
	ctx context.Context,
	userID int64,
	input *service.PrepareChatCompletionInput,
) (*service.PreparedChatCompletion, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("chat history repository db is nil")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	if existing, existingHash, findErr := findPreparedChatAttempt(
		ctx,
		tx,
		userID,
		input.AttemptID,
	); findErr == nil {
		if existingHash != input.RequestHash {
			return nil, service.ErrChatAttemptConflict
		}
		if err = tx.Commit(); err != nil {
			return nil, err
		}
		return existing, nil
	} else if !errors.Is(findErr, sql.ErrNoRows) {
		return nil, findErr
	}

	if _, err = lockChatHistorySyncState(ctx, tx, userID); err != nil {
		return nil, err
	}
	// The per-user sync lock serializes first claims. Re-check after acquiring
	// it so concurrent requests cannot append the same attempt twice.
	if existing, existingHash, findErr := findPreparedChatAttempt(
		ctx,
		tx,
		userID,
		input.AttemptID,
	); findErr == nil {
		if existingHash != input.RequestHash {
			return nil, service.ErrChatAttemptConflict
		}
		if err = tx.Commit(); err != nil {
			return nil, err
		}
		return existing, nil
	} else if !errors.Is(findErr, sql.ErrNoRows) {
		return nil, findErr
	}

	var (
		conversationID int64
		revision       int64
		messageCount   int
		headMessageID  sql.NullInt64
		headPublicID   sql.NullString
		headRole       sql.NullString
		headStatus     sql.NullString
	)
	err = tx.QueryRowContext(ctx, `
		SELECT
			c.id,
			c.revision,
			c.message_count,
			c.head_message_id,
			head.public_id,
			head.role,
			head.delivery_status
		FROM chat_conversations c
		LEFT JOIN chat_messages head ON head.id = c.head_message_id
		WHERE c.user_id = $1
		  AND c.public_id = $2
		  AND c.deleted_at IS NULL
		FOR UPDATE OF c
	`, userID, input.ConversationID).Scan(
		&conversationID,
		&revision,
		&messageCount,
		&headMessageID,
		&headPublicID,
		&headRole,
		&headStatus,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrChatHistoryNotFound
	}
	if err != nil {
		return nil, err
	}

	headMatches := input.ExpectedHeadMessageID == nil && !headPublicID.Valid
	if input.ExpectedHeadMessageID != nil {
		headMatches = headPublicID.Valid && headPublicID.String == *input.ExpectedHeadMessageID
	}
	if !headMatches {
		currentHead := ""
		if headPublicID.Valid {
			currentHead = headPublicID.String
		}
		return nil, service.ErrChatHistoryRevisionConflict.WithMetadata(map[string]string{
			"current_revision":        strconv.FormatInt(revision, 10),
			"current_head_message_id": currentHead,
		})
	}
	if headRole.String == "assistant" &&
		(headStatus.String == service.ChatMessageDeliveryPending ||
			headStatus.String == service.ChatMessageDeliveryStreaming) {
		return nil, service.ErrChatTurnInProgress.WithMetadata(map[string]string{
			"current_revision":        strconv.FormatInt(revision, 10),
			"current_head_message_id": headPublicID.String,
		})
	}

	now := time.Now().UTC()
	appendCount := 1
	var userMessageInternalID int64
	if input.UserMessage != nil {
		appendCount = 2
		if err = tx.QueryRowContext(ctx, `
			INSERT INTO chat_messages (
				public_id,
				user_id,
				conversation_id,
				position,
				role,
				content,
				delivery_status,
				created_at,
				updated_at,
				terminal_at
			)
			VALUES ($1, $2, $3, $4, 'user', $5, $6, $7, $7, $7)
			RETURNING id
		`,
			input.UserMessage.ID,
			userID,
			conversationID,
			messageCount+1,
			input.UserMessage.Content,
			service.ChatMessageDeliveryCompleted,
			now,
		).Scan(&userMessageInternalID); err != nil {
			return nil, err
		}
		for position := range input.UserMessage.Attachments {
			attachment := input.UserMessage.Attachments[position]
			var attachmentInternalID int64
			if err = tx.QueryRowContext(ctx, `
				UPDATE chat_attachments
				SET conversation_id=$3, updated_at=$4
				WHERE user_id=$1 AND public_id=$2
				  AND status='ready' AND expires_at > $4
				  AND (conversation_id IS NULL OR conversation_id=$3)
				  AND sha256=$5
				RETURNING id
			`, userID, attachment.ID, conversationID, now, attachment.Digest).Scan(&attachmentInternalID); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return nil, service.ErrChatAttachmentNotFound
				}
				return nil, err
			}
			if _, err = tx.ExecContext(ctx, `
				INSERT INTO chat_message_attachments (message_id, attachment_id, position)
				VALUES ($1,$2,$3)
			`, userMessageInternalID, attachmentInternalID, position+1); err != nil {
				return nil, err
			}
		}
	} else {
		if !headMessageID.Valid ||
			!headPublicID.Valid ||
			headPublicID.String != input.RetryOfMessageID {
			return nil, service.ErrChatHistoryRevisionConflict.WithMetadata(map[string]string{
				"current_revision": strconv.FormatInt(revision, 10),
			})
		}
		var (
			retryRole       string
			retryStatus     string
			retryExcluded   bool
			retrySuperseded sql.NullInt64
		)
		if err = tx.QueryRowContext(ctx, `
			SELECT
				role,
				delivery_status,
				excluded_from_context,
				superseded_by_message_id
			FROM chat_messages
			WHERE id = $1
			  AND user_id = $2
			  AND conversation_id = $3
			FOR UPDATE
		`, headMessageID.Int64, userID, conversationID).Scan(
			&retryRole,
			&retryStatus,
			&retryExcluded,
			&retrySuperseded,
		); err != nil {
			return nil, err
		}
		terminalRetryTarget := false
		switch retryStatus {
		case service.ChatMessageDeliveryCompleted,
			service.ChatMessageDeliveryPartial,
			service.ChatMessageDeliveryStopped,
			service.ChatMessageDeliveryInterrupted,
			service.ChatMessageDeliveryError:
			terminalRetryTarget = true
		}
		if retryRole != "assistant" ||
			!terminalRetryTarget ||
			retryExcluded ||
			retrySuperseded.Valid {
			return nil, service.ErrChatHistoryInvalid
		}
	}

	assistantPosition := messageCount + appendCount
	var assistantInternalID int64
	if err = tx.QueryRowContext(ctx, `
		INSERT INTO chat_messages (
			public_id,
			user_id,
			conversation_id,
			position,
			role,
			content,
			delivery_status,
			requested_model,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, 'assistant', '', $5, $6, $7, $7)
		RETURNING id
	`,
		input.AssistantMessageID,
		userID,
		conversationID,
		assistantPosition,
		service.ChatMessageDeliveryStreaming,
		input.Model,
		now,
	).Scan(&assistantInternalID); err != nil {
		return nil, err
	}

	if input.UserMessage == nil {
		if _, err = tx.ExecContext(ctx, `
			UPDATE chat_messages
			SET excluded_from_context = TRUE,
			    superseded_by_message_id = $2,
			    updated_at = $3
			WHERE id = $1
		`, headMessageID.Int64, assistantInternalID, now); err != nil {
			return nil, err
		}
	}

	if err = insertPreparedChatAttempt(
		ctx,
		tx,
		userID,
		input,
		assistantInternalID,
		now,
	); err != nil {
		return nil, err
	}

	version, err := bumpChatHistorySyncVersion(ctx, tx, userID)
	if err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `
		UPDATE chat_conversations
		SET model = $3,
		    revision = revision + 1,
		    version = $4,
		    head_message_id = $5,
		    message_count = message_count + $6,
		    updated_at = $7
		WHERE user_id = $1
		  AND id = $2
		  AND deleted_at IS NULL
	`,
		userID,
		conversationID,
		input.Model,
		version,
		assistantInternalID,
		appendCount,
		now,
	); err != nil {
		return nil, err
	}
	if err = insertChatHistoryChange(
		ctx,
		tx,
		userID,
		version,
		"upsert",
		conversationID,
		input.ConversationID,
		nil,
	); err != nil {
		return nil, err
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT context_message.id, context_message.role, context_message.content
		FROM (
			SELECT m.id, m.position, m.role, m.content
			FROM chat_messages m
			WHERE m.user_id = $1
			  AND m.conversation_id = $2
			  AND m.id <> $3
			  AND m.excluded_from_context = FALSE
			  AND (
				(m.role = 'user' AND m.delivery_status = 'completed')
				OR (
					m.role = 'assistant'
					AND m.delivery_status IN ('completed', 'partial', 'stopped', 'interrupted')
					AND m.content <> ''
				)
			  )
			ORDER BY m.position DESC
			LIMIT $4
		) context_message
		ORDER BY context_message.position ASC
	`, userID, conversationID, assistantInternalID, input.ContextMessageLimit)
	if err != nil {
		return nil, err
	}
	messages := make([]service.ChatCompletionContextMessage, 0, input.ContextMessageLimit)
	contextMessageIDs := make([]int64, 0, input.ContextMessageLimit)
	for rows.Next() {
		var message service.ChatCompletionContextMessage
		var messageID int64
		if err = rows.Scan(&messageID, &message.Role, &message.Content); err != nil {
			_ = rows.Close()
			return nil, err
		}
		messages = append(messages, message)
		contextMessageIDs = append(contextMessageIDs, messageID)
	}
	if err = rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err = rows.Close(); err != nil {
		return nil, err
	}
	attachmentsByMessage, err := loadChatAttachmentsForMessages(ctx, tx, contextMessageIDs)
	if err != nil {
		return nil, err
	}
	for i := range messages {
		messages[i].Attachments = attachmentsByMessage[contextMessageIDs[i]]
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return &service.PreparedChatCompletion{
		Claimed:            true,
		ClientRequestID:    input.ClientRequestID,
		AttemptStatus:      service.ChatAttemptStatusProcessing,
		ConversationID:     input.ConversationID,
		AssistantMessageID: input.AssistantMessageID,
		Messages:           messages,
	}, nil
}

func insertPreparedChatAttempt(
	ctx context.Context,
	exec sqlExecutor,
	userID int64,
	input *service.PrepareChatCompletionInput,
	assistantInternalID int64,
	now time.Time,
) error {
	_, err := exec.ExecContext(ctx, `
		INSERT INTO chat_request_attempts (
			user_id,
			attempt_id,
			client_request_id,
			request_hash,
			status,
			assistant_message_id,
			conversation_public_id,
			assistant_message_public_id,
			lease_expires_at,
			created_at,
			updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			NOW() + INTERVAL '5 minutes',
			$9, $9
		)
	`,
		userID,
		input.AttemptID,
		input.ClientRequestID,
		input.RequestHash,
		service.ChatAttemptStatusProcessing,
		assistantInternalID,
		input.ConversationID,
		input.AssistantMessageID,
		now,
	)
	return err
}

func findPreparedChatAttempt(
	ctx context.Context,
	tx *sql.Tx,
	userID int64,
	attemptID string,
) (*service.PreparedChatCompletion, string, error) {
	var (
		requestHash        string
		result             service.PreparedChatCompletion
		conversationID     sql.NullString
		assistantMessageID sql.NullString
	)
	err := tx.QueryRowContext(ctx, `
		SELECT
			a.request_hash,
			a.client_request_id,
			a.status,
			COALESCE(a.conversation_public_id, c.public_id),
			COALESCE(a.assistant_message_public_id, m.public_id)
		FROM chat_request_attempts a
		LEFT JOIN chat_messages m
		  ON m.id = a.assistant_message_id
		 AND m.user_id = a.user_id
		LEFT JOIN chat_conversations c
		  ON c.id = m.conversation_id
		 AND c.user_id = a.user_id
		WHERE a.user_id = $1
		  AND a.attempt_id = $2
		FOR UPDATE OF a
	`, userID, attemptID).Scan(
		&requestHash,
		&result.ClientRequestID,
		&result.AttemptStatus,
		&conversationID,
		&assistantMessageID,
	)
	if err != nil {
		return nil, "", err
	}
	if conversationID.Valid {
		result.ConversationID = conversationID.String
	}
	if assistantMessageID.Valid {
		result.AssistantMessageID = assistantMessageID.String
	}
	return &result, requestHash, nil
}

func (r *chatHistoryRepository) CheckpointCompletion(
	ctx context.Context,
	userID int64,
	input *service.CheckpointChatCompletionInput,
) error {
	if r == nil || r.db == nil {
		return errors.New("chat history repository db is nil")
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE chat_messages AS message
		SET content = $4,
		    checkpoint_seq = $5,
		    delivery_status = 'streaming',
		    updated_at = NOW()
		FROM chat_request_attempts AS attempt,
		     chat_conversations AS conversation
		WHERE attempt.user_id = $1
		  AND attempt.attempt_id = $2
		  AND attempt.status = 'processing'
		  AND attempt.assistant_message_id = message.id
		  AND attempt.assistant_message_public_id = $3
		  AND message.user_id = $1
		  AND message.public_id = $3
		  AND message.conversation_id = conversation.id
		  AND conversation.user_id = $1
		  AND conversation.deleted_at IS NULL
		  AND message.delivery_status IN ('pending', 'streaming')
		  AND message.checkpoint_seq < $5
		  AND LEFT($4, LENGTH(message.content)) = message.content
	`,
		userID,
		input.AttemptID,
		input.AssistantMessageID,
		input.Content,
		input.CheckpointSeq,
	)
	return err
}

func (r *chatHistoryRepository) FinalizeCompletion(
	ctx context.Context,
	userID int64,
	input *service.FinalizeChatCompletionInput,
) error {
	if r == nil || r.db == nil {
		return errors.New("chat history repository db is nil")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = lockChatHistorySyncState(ctx, tx, userID); err != nil {
		return err
	}

	var (
		currentStatus           string
		assistantInternal       sql.NullInt64
		stableAssistantPublicID sql.NullString
	)
	err = tx.QueryRowContext(ctx, `
		SELECT status, assistant_message_id, assistant_message_public_id
		FROM chat_request_attempts
		WHERE user_id = $1
		  AND attempt_id = $2
		FOR UPDATE
	`, userID, input.AttemptID).Scan(
		&currentStatus,
		&assistantInternal,
		&stableAssistantPublicID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrChatAttemptNotFound
	}
	if err != nil {
		return err
	}
	switch currentStatus {
	case service.ChatAttemptStatusCompleted,
		service.ChatAttemptStatusInterrupted,
		service.ChatAttemptStatusFailed:
		return tx.Commit()
	}
	if stableAssistantPublicID.Valid &&
		stableAssistantPublicID.String != input.AssistantMessageID {
		return service.ErrChatAttemptConflict
	}

	now := time.Now().UTC()
	if !assistantInternal.Valid {
		if err = updateTerminalChatAttempt(ctx, tx, userID, input, now); err != nil {
			return err
		}
		return tx.Commit()
	}

	var (
		conversationID       int64
		conversationPublicID string
		assistantPublicID    string
		currentCheckpointSeq int64
	)
	err = tx.QueryRowContext(ctx, `
		SELECT
			m.conversation_id,
			c.public_id,
			m.public_id,
			m.checkpoint_seq
		FROM chat_messages m
		JOIN chat_conversations c
		  ON c.id = m.conversation_id
		 AND c.user_id = m.user_id
		WHERE m.id = $1
		  AND m.user_id = $2
		  AND c.deleted_at IS NULL
		FOR UPDATE OF c, m
	`, assistantInternal.Int64, userID).Scan(
		&conversationID,
		&conversationPublicID,
		&assistantPublicID,
		&currentCheckpointSeq,
	)
	if errors.Is(err, sql.ErrNoRows) {
		if err = updateTerminalChatAttempt(ctx, tx, userID, input, now); err != nil {
			return err
		}
		return tx.Commit()
	}
	if err != nil {
		return err
	}
	if assistantPublicID != input.AssistantMessageID {
		return service.ErrChatAttemptConflict
	}
	if input.CheckpointSeq <= currentCheckpointSeq {
		return service.ErrChatAttemptConflict
	}

	messageResult, err := tx.ExecContext(ctx, `
		UPDATE chat_messages
		SET content = $3,
		    delivery_status = $4,
		    finish_reason = NULLIF($5, ''),
		    error_code = NULLIF($6, ''),
		    error_message = NULLIF($7, ''),
		    checkpoint_seq = $8,
		    updated_at = $9,
		    terminal_at = $9
		WHERE id = $1
		  AND user_id = $2
		  AND delivery_status IN ('pending', 'streaming')
		  AND checkpoint_seq < $8
		  AND LEFT($3, LENGTH(content)) = content
	`,
		assistantInternal.Int64,
		userID,
		input.Content,
		input.DeliveryStatus,
		input.FinishReason,
		input.ErrorCode,
		input.ErrorMessage,
		input.CheckpointSeq,
		now,
	)
	if err != nil {
		return err
	}
	messageUpdated, err := messageResult.RowsAffected()
	if err != nil {
		return err
	}
	if messageUpdated != 1 {
		return service.ErrChatAttemptConflict
	}
	version, err := bumpChatHistorySyncVersion(ctx, tx, userID)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		UPDATE chat_conversations
		SET revision = revision + 1,
		    version = $3,
		    updated_at = $4
		WHERE user_id = $1
		  AND id = $2
		  AND deleted_at IS NULL
	`, userID, conversationID, version, now)
	if err != nil {
		return err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if updated == 0 {
		if err = updateTerminalChatAttempt(ctx, tx, userID, input, now); err != nil {
			return err
		}
		return tx.Commit()
	}
	if err = insertChatHistoryChange(
		ctx,
		tx,
		userID,
		version,
		"upsert",
		conversationID,
		conversationPublicID,
		nil,
	); err != nil {
		return err
	}
	if err = updateTerminalChatAttempt(ctx, tx, userID, input, now); err != nil {
		return err
	}
	return tx.Commit()
}

func updateTerminalChatAttempt(
	ctx context.Context,
	tx *sql.Tx,
	userID int64,
	input *service.FinalizeChatCompletionInput,
	now time.Time,
) error {
	result, err := tx.ExecContext(ctx, `
		UPDATE chat_request_attempts
		SET status = $3,
		    http_status = NULLIF($4, 0),
		    failure_code = NULLIF($5, ''),
		    failure_reason = NULLIF($6, ''),
		    lease_expires_at = NULL,
		    updated_at = $7,
		    terminal_at = $7
		WHERE user_id = $1
		  AND attempt_id = $2
	`,
		userID,
		input.AttemptID,
		input.AttemptStatus,
		input.HTTPStatus,
		input.ErrorCode,
		input.ErrorMessage,
		now,
	)
	if err != nil {
		return err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if updated != 1 {
		return service.ErrChatAttemptNotFound
	}
	return nil
}

func (r *chatHistoryRepository) GetAttempt(
	ctx context.Context,
	userID int64,
	attemptID string,
) (*service.ChatHistoryAttempt, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("chat history repository db is nil")
	}
	row := r.db.QueryRowContext(ctx, `
		SELECT
			a.attempt_id,
			COALESCE(a.conversation_public_id, c.public_id),
			COALESCE(a.assistant_message_public_id, m.public_id),
			a.client_request_id,
			a.status,
			a.http_status,
			a.failure_code,
			a.failure_reason,
			a.created_at,
			a.updated_at,
			m.public_id,
			m.position,
			m.role,
			m.content,
			m.delivery_status,
			COALESCE(m.requested_model, ''),
			m.finish_reason,
			m.error_code,
			m.error_message,
			m.excluded_from_context,
			superseding.public_id,
			m.checkpoint_seq,
			a.attempt_id,
			a.client_request_id,
			m.created_at,
			m.updated_at,
			m.terminal_at
		FROM chat_request_attempts a
		LEFT JOIN chat_messages m
		  ON m.id = a.assistant_message_id
		 AND m.user_id = a.user_id
		LEFT JOIN chat_conversations c
		  ON c.id = m.conversation_id
		 AND c.user_id = a.user_id
		LEFT JOIN chat_messages superseding
		  ON superseding.id = m.superseded_by_message_id
		WHERE a.user_id = $1
		  AND a.attempt_id = $2
		LIMIT 1
	`, userID, attemptID)

	var (
		attempt            service.ChatHistoryAttempt
		conversationID     sql.NullString
		assistantMessageID sql.NullString
		httpStatus         sql.NullInt64
		failureCode        sql.NullString
		failureReason      sql.NullString
		messageID          sql.NullString
		position           sql.NullInt64
		role               sql.NullString
		content            sql.NullString
		status             sql.NullString
		requestedModel     sql.NullString
		finishReason       sql.NullString
		errorCode          sql.NullString
		errorMessage       sql.NullString
		excluded           sql.NullBool
		supersededBy       sql.NullString
		checkpointSeq      sql.NullInt64
		linkedAttempt      sql.NullString
		receiptID          sql.NullString
		messageCreated     sql.NullTime
		messageUpdated     sql.NullTime
		terminalAt         sql.NullTime
	)
	err := row.Scan(
		&attempt.AttemptID,
		&conversationID,
		&assistantMessageID,
		&attempt.ReceiptID,
		&attempt.Status,
		&httpStatus,
		&failureCode,
		&failureReason,
		&attempt.CreatedAt,
		&attempt.UpdatedAt,
		&messageID,
		&position,
		&role,
		&content,
		&status,
		&requestedModel,
		&finishReason,
		&errorCode,
		&errorMessage,
		&excluded,
		&supersededBy,
		&checkpointSeq,
		&linkedAttempt,
		&receiptID,
		&messageCreated,
		&messageUpdated,
		&terminalAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrChatAttemptNotFound
	}
	if err != nil {
		return nil, err
	}
	if conversationID.Valid {
		attempt.ConversationID = conversationID.String
	}
	if assistantMessageID.Valid {
		attempt.AssistantMessageID = assistantMessageID.String
	}
	if httpStatus.Valid {
		value := int(httpStatus.Int64)
		attempt.HTTPStatus = &value
	}
	if failureCode.Valid {
		value := failureCode.String
		attempt.FailureCode = &value
	}
	if failureReason.Valid {
		value := failureReason.String
		attempt.FailureReason = &value
	}
	if messageID.Valid {
		message := &service.ChatHistoryMessage{
			ID:                  messageID.String,
			Position:            position.Int64,
			Role:                role.String,
			Content:             content.String,
			Status:              status.String,
			RequestedModel:      requestedModel.String,
			ExcludedFromContext: excluded.Bool,
			CheckpointSeq:       checkpointSeq.Int64,
			CreatedAt:           messageCreated.Time,
			UpdatedAt:           messageUpdated.Time,
		}
		if finishReason.Valid {
			value := finishReason.String
			message.FinishReason = &value
		}
		if errorCode.Valid {
			value := errorCode.String
			message.ErrorCode = &value
		}
		if errorMessage.Valid {
			value := errorMessage.String
			message.ErrorMessage = &value
		}
		if supersededBy.Valid {
			value := supersededBy.String
			message.SupersededByMessageID = &value
		}
		if linkedAttempt.Valid {
			value := linkedAttempt.String
			message.AttemptID = &value
		}
		if receiptID.Valid {
			value := receiptID.String
			message.ReceiptID = &value
		}
		if terminalAt.Valid {
			value := terminalAt.Time
			message.TerminalAt = &value
		}
		attempt.AssistantMessage = message
	}
	return &attempt, nil
}

type chatHistoryQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func getChatHistoryConversationWith(
	ctx context.Context,
	queryer chatHistoryQueryer,
	userID int64,
	publicID string,
	includeDeleted bool,
) (*service.ChatHistoryConversation, error) {
	deletedClause := "AND c.deleted_at IS NULL"
	if includeDeleted {
		deletedClause = ""
	}
	row := queryer.QueryRowContext(ctx, `
		SELECT `+chatHistoryConversationColumns+`
		FROM chat_conversations c
		LEFT JOIN chat_messages hm ON hm.id = c.head_message_id
		WHERE c.user_id = $1
		  AND c.public_id = $2
		  `+deletedClause+`
		LIMIT 1
	`, userID, publicID)
	_, conversation, err := scanChatHistoryConversation(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrChatHistoryNotFound
	}
	return conversation, err
}

func scanChatHistoryMessage(scan chatHistoryScanner) (*service.ChatHistoryMessage, error) {
	var (
		message      service.ChatHistoryMessage
		finishReason sql.NullString
		errorCode    sql.NullString
		errorMessage sql.NullString
		supersededBy sql.NullString
		attemptID    sql.NullString
		receiptID    sql.NullString
		terminalAt   sql.NullTime
	)
	if err := scan(
		&message.ID,
		&message.Position,
		&message.Role,
		&message.Content,
		&message.Status,
		&message.RequestedModel,
		&finishReason,
		&errorCode,
		&errorMessage,
		&message.ExcludedFromContext,
		&supersededBy,
		&message.CheckpointSeq,
		&attemptID,
		&receiptID,
		&message.CreatedAt,
		&message.UpdatedAt,
		&terminalAt,
	); err != nil {
		return nil, err
	}
	if finishReason.Valid {
		value := finishReason.String
		message.FinishReason = &value
	}
	if errorCode.Valid {
		value := errorCode.String
		message.ErrorCode = &value
	}
	if errorMessage.Valid {
		value := errorMessage.String
		message.ErrorMessage = &value
	}
	if supersededBy.Valid {
		value := supersededBy.String
		message.SupersededByMessageID = &value
	}
	if attemptID.Valid {
		value := attemptID.String
		message.AttemptID = &value
	}
	if receiptID.Valid {
		value := receiptID.String
		message.ReceiptID = &value
	}
	if terminalAt.Valid {
		value := terminalAt.Time
		message.TerminalAt = &value
	}
	return &message, nil
}

func loadChatAttachmentsForMessages(
	ctx context.Context,
	queryer sqlQueryer,
	messageIDs []int64,
) (map[int64][]service.ChatAttachment, error) {
	result := make(map[int64][]service.ChatAttachment)
	if len(messageIDs) == 0 {
		return result, nil
	}
	rows, err := queryer.QueryContext(ctx, `
		SELECT ma.message_id, `+chatAttachmentColumns+`
		FROM chat_message_attachments ma
		JOIN chat_attachments a ON a.id=ma.attachment_id
		WHERE ma.message_id = ANY($1)
		ORDER BY ma.message_id, ma.position
	`, pq.Array(messageIDs))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var messageID int64
		attachment, scanErr := scanChatAttachment(func(dest ...any) error {
			return rows.Scan(append([]any{&messageID}, dest...)...)
		})
		if scanErr != nil {
			return nil, scanErr
		}
		result[messageID] = append(result[messageID], *attachment)
	}
	return result, rows.Err()
}

func lockChatHistorySyncState(ctx context.Context, tx *sql.Tx, userID int64) (int64, error) {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO chat_history_sync_states (user_id)
		VALUES ($1)
		ON CONFLICT (user_id) DO NOTHING
	`, userID); err != nil {
		return 0, err
	}
	var version int64
	err := tx.QueryRowContext(ctx, `
		SELECT version
		FROM chat_history_sync_states
		WHERE user_id = $1
		FOR UPDATE
	`, userID).Scan(&version)
	return version, err
}

func bumpChatHistorySyncVersion(ctx context.Context, tx *sql.Tx, userID int64) (int64, error) {
	var version int64
	err := tx.QueryRowContext(ctx, `
		UPDATE chat_history_sync_states
		SET version = version + 1,
		    updated_at = NOW()
		WHERE user_id = $1
		RETURNING version
	`, userID).Scan(&version)
	return version, err
}

func insertChatHistoryChange(
	ctx context.Context,
	tx *sql.Tx,
	userID, version int64,
	changeType string,
	conversationID int64,
	conversationPublicID string,
	deletedAt *time.Time,
) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO chat_history_changes (
			user_id,
			version,
			change_type,
			conversation_id,
			conversation_public_id,
			deleted_at
		)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, userID, version, changeType, conversationID, conversationPublicID, deletedAt)
	return err
}

func escapeChatHistoryLike(value string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`%`, `\%`,
		`_`, `\_`,
	)
	return replacer.Replace(value)
}

func encodeChatHistoryConversationCursor(updatedAt time.Time, internalID int64) string {
	value := strconv.FormatInt(updatedAt.UTC().UnixNano(), 10) +
		":" +
		strconv.FormatInt(internalID, 10)
	return base64.RawURLEncoding.EncodeToString([]byte(value))
}

func decodeChatHistoryConversationCursor(value string) (time.Time, int64, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, 0, err
	}
	parts := strings.Split(string(decoded), ":")
	if len(parts) != 2 {
		return time.Time{}, 0, service.ErrChatHistoryInvalid
	}
	unixNano, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Time{}, 0, err
	}
	internalID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || internalID <= 0 {
		return time.Time{}, 0, service.ErrChatHistoryInvalid
	}
	return time.Unix(0, unixNano).UTC(), internalID, nil
}
