package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type chatAttachmentRepository struct {
	db               *sql.DB
	uploadsPerMinute int
	dailyUploadBytes int64
}

func NewChatAttachmentRepository(db *sql.DB, cfg *config.Config) service.ChatAttachmentRepository {
	uploadsPerMinute := 10
	dailyUploadBytes := int64(100 << 20)
	if cfg != nil {
		if cfg.ChatAttachments.UploadsPerMinute > 0 {
			uploadsPerMinute = cfg.ChatAttachments.UploadsPerMinute
		}
		if cfg.ChatAttachments.DailyUploadBytes > 0 {
			dailyUploadBytes = cfg.ChatAttachments.DailyUploadBytes
		}
	}
	return &chatAttachmentRepository{db: db, uploadsPerMinute: uploadsPerMinute, dailyUploadBytes: dailyUploadBytes}
}

const chatAttachmentColumns = `
	a.public_id,
	a.original_name,
	a.kind,
	a.mime_type,
	a.byte_size,
	a.stored_size,
	a.status,
	a.expires_at,
	a.page_count,
	a.width,
	a.height,
	COALESCE(a.storage_key, ''),
	a.sha256,
	COALESCE(a.extracted_text, '')`

func scanChatAttachment(scan chatHistoryScanner) (*service.ChatAttachment, error) {
	var (
		attachment  service.ChatAttachment
		storageKind string
		pageCount   sql.NullInt64
		width       sql.NullInt64
		height      sql.NullInt64
	)
	if err := scan(
		&attachment.ID, &attachment.Name, &storageKind, &attachment.MIMEType,
		&attachment.Size, &attachment.StoredSize, &attachment.Status, &attachment.ExpiresAt,
		&pageCount, &width, &height, &attachment.StorageKey, &attachment.Digest,
		&attachment.ExtractedText,
	); err != nil {
		return nil, err
	}
	if pageCount.Valid {
		attachment.PageCount = int(pageCount.Int64)
	}
	if width.Valid {
		attachment.Width = int(width.Int64)
	}
	if height.Valid {
		attachment.Height = int(height.Int64)
	}
	attachment.Format = storageKind
	attachment.Kind = service.ChatAttachmentKindDocument
	if storageKind == service.ChatAttachmentKindImage {
		attachment.Kind = service.ChatAttachmentKindImage
	}
	return &attachment, nil
}

func nullablePositive(value int) any {
	if value <= 0 {
		return nil
	}
	return value
}

func (r *chatAttachmentRepository) Create(ctx context.Context, input *service.CreateChatAttachmentInput) (*service.ChatAttachment, error) {
	if r == nil || r.db == nil || input == nil || input.UserID <= 0 {
		return nil, errors.New("chat attachment repository is unavailable")
	}
	a := input.Attachment
	storageKind := a.Format
	if storageKind == "" {
		storageKind = a.Kind
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	var lockResult any
	if err = tx.QueryRowContext(ctx, `SELECT pg_advisory_xact_lock(hashtext('chat_attachments'), hashint8($1::bigint))`, input.UserID).Scan(&lockResult); err != nil {
		return nil, err
	}
	var uploadsLastMinute int
	var bytesLastDay int64
	if err = tx.QueryRowContext(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '1 minute'),
			COALESCE(SUM(byte_size) FILTER (WHERE created_at >= NOW() - INTERVAL '24 hours'), 0)
		FROM chat_attachments
		WHERE user_id=$1
	`, input.UserID).Scan(&uploadsLastMinute, &bytesLastDay); err != nil {
		return nil, err
	}
	if uploadsLastMinute >= r.uploadsPerMinute || a.Size <= 0 || bytesLastDay > r.dailyUploadBytes-a.Size {
		return nil, service.ErrChatAttachmentRateLimit
	}
	row := tx.QueryRowContext(ctx, `
		INSERT INTO chat_attachments AS a (
			public_id, user_id, original_name, kind, mime_type, byte_size, stored_size,
			sha256, storage_key, extracted_text, page_count, width, height,
			status, expires_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NULLIF($9,''),NULLIF($10,''),$11,$12,$13,$14,$15)
		RETURNING `+chatAttachmentColumns,
		a.ID, input.UserID, a.Name, storageKind, a.MIMEType, a.Size, a.StoredSize, a.Digest,
		a.StorageKey, a.ExtractedText, nullablePositive(a.PageCount), nullablePositive(a.Width),
		nullablePositive(a.Height), a.Status, a.ExpiresAt,
	)
	attachment, err := scanChatAttachment(row.Scan)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return attachment, nil
}

func (r *chatAttachmentRepository) GetOwned(ctx context.Context, userID int64, publicID string) (*service.ChatAttachment, error) {
	if r == nil || r.db == nil || userID <= 0 {
		return nil, service.ErrChatAttachmentNotFound
	}
	row := r.db.QueryRowContext(ctx, `SELECT `+chatAttachmentColumns+`
		FROM chat_attachments a WHERE a.user_id=$1 AND a.public_id=$2`, userID, publicID)
	attachment, err := scanChatAttachment(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrChatAttachmentNotFound
	}
	return attachment, err
}

func (r *chatAttachmentRepository) MarkReady(ctx context.Context, userID int64, publicID string) (*service.ChatAttachment, error) {
	if r == nil || r.db == nil || userID <= 0 {
		return nil, service.ErrChatAttachmentNotFound
	}
	row := r.db.QueryRowContext(ctx, `
		UPDATE chat_attachments a
		SET status='ready', updated_at=NOW()
		WHERE a.user_id=$1 AND a.public_id=$2 AND a.status='pending'
		RETURNING `+chatAttachmentColumns, userID, publicID)
	attachment, err := scanChatAttachment(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrChatAttachmentNotFound
	}
	return attachment, err
}

func (r *chatAttachmentRepository) CheckUploadQuota(ctx context.Context, userID, rawSize int64) error {
	if r == nil || r.db == nil || userID <= 0 || rawSize <= 0 {
		return service.ErrChatAttachmentRateLimit
	}
	var uploadsLastMinute int
	var bytesLastDay int64
	if err := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '1 minute'),
			COALESCE(SUM(byte_size) FILTER (WHERE created_at >= NOW() - INTERVAL '24 hours'), 0)
		FROM chat_attachments
		WHERE user_id=$1
	`, userID).Scan(&uploadsLastMinute, &bytesLastDay); err != nil {
		return err
	}
	if uploadsLastMinute >= r.uploadsPerMinute || rawSize > r.dailyUploadBytes || bytesLastDay > r.dailyUploadBytes-rawSize {
		return service.ErrChatAttachmentRateLimit
	}
	return nil
}

func (r *chatAttachmentRepository) ResolveForCompletion(ctx context.Context, userID int64, publicIDs []string) ([]service.ChatAttachment, error) {
	if r == nil || r.db == nil || userID <= 0 || len(publicIDs) == 0 {
		return nil, nil
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+chatAttachmentColumns+`
		FROM unnest($2::text[]) WITH ORDINALITY requested(public_id, position)
		JOIN chat_attachments a
		  ON a.public_id=requested.public_id AND a.user_id=$1
		ORDER BY requested.position`, userID, pq.Array(publicIDs))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	attachments := make([]service.ChatAttachment, 0, len(publicIDs))
	for rows.Next() {
		attachment, scanErr := scanChatAttachment(rows.Scan)
		if scanErr != nil {
			return nil, scanErr
		}
		attachments = append(attachments, *attachment)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if len(attachments) != len(publicIDs) {
		return nil, service.ErrChatAttachmentNotFound
	}
	return attachments, nil
}

func (r *chatAttachmentRepository) MarkDeleted(ctx context.Context, userID int64, publicID string) (*service.ChatAttachment, error) {
	if r == nil || r.db == nil || userID <= 0 {
		return nil, service.ErrChatAttachmentNotFound
	}
	row := r.db.QueryRowContext(ctx, `
		UPDATE chat_attachments a
		SET status='deleted', extracted_text=NULL, updated_at=NOW()
		WHERE a.user_id=$1 AND a.public_id=$2 AND a.conversation_id IS NULL
		RETURNING `+chatAttachmentColumns, userID, publicID)
	attachment, err := scanChatAttachment(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrChatAttachmentNotFound
	}
	return attachment, err
}

func (r *chatAttachmentRepository) MarkConversationDeleted(ctx context.Context, userID int64, conversationPublicID string) ([]service.ChatAttachment, error) {
	if r == nil || r.db == nil || userID <= 0 {
		return nil, service.ErrChatAttachmentUnavailable
	}
	rows, err := r.db.QueryContext(ctx, `
		UPDATE chat_attachments a
		SET status='deleted', extracted_text=NULL, updated_at=NOW()
		FROM chat_conversations c
		WHERE c.user_id=$1 AND c.public_id=$2
		  AND a.user_id=c.user_id AND a.conversation_id=c.id
		RETURNING `+chatAttachmentColumns, userID, conversationPublicID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var attachments []service.ChatAttachment
	for rows.Next() {
		a, scanErr := scanChatAttachment(rows.Scan)
		if scanErr != nil {
			return nil, scanErr
		}
		attachments = append(attachments, *a)
	}
	return attachments, rows.Err()
}

func (r *chatAttachmentRepository) ClaimCleanup(ctx context.Context, now time.Time, limit int) ([]service.ChatAttachment, error) {
	if r == nil || r.db == nil {
		return nil, service.ErrChatAttachmentUnavailable
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(ctx, `
		UPDATE chat_attachments
		SET status='expired', extracted_text=NULL, updated_at=$1
		WHERE status='ready' AND expires_at <= $1`, now); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `
		UPDATE chat_attachments
		SET status='deleted', extracted_text=NULL, updated_at=$1
		WHERE status='pending' AND updated_at <= $1 - INTERVAL '1 hour'`, now); err != nil {
		return nil, err
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT `+chatAttachmentColumns+`
		FROM chat_attachments a
		WHERE a.status IN ('expired','deleted') AND a.storage_key IS NOT NULL
		ORDER BY a.updated_at, a.id
		LIMIT $1
		FOR UPDATE SKIP LOCKED`, limit)
	if err != nil {
		return nil, err
	}
	var attachments []service.ChatAttachment
	for rows.Next() {
		a, scanErr := scanChatAttachment(rows.Scan)
		if scanErr != nil {
			_ = rows.Close()
			return nil, scanErr
		}
		attachments = append(attachments, *a)
	}
	if err = rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err = rows.Close(); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return attachments, nil
}

func (r *chatAttachmentRepository) FinalizeCleanup(ctx context.Context, userID int64, publicID string) error {
	if r == nil || r.db == nil {
		return service.ErrChatAttachmentUnavailable
	}
	result, err := r.db.ExecContext(ctx, `
		UPDATE chat_attachments
		SET storage_key=NULL, extracted_text=NULL, cleaned_at=NOW(), updated_at=NOW()
		WHERE public_id=$1 AND ($2=0 OR user_id=$2) AND status IN ('expired','deleted')`, publicID, userID)
	if err != nil {
		return err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if updated == 0 && userID > 0 {
		return service.ErrChatAttachmentNotFound
	}
	return nil
}

func (r *chatAttachmentRepository) ListStorageKeys(ctx context.Context) (map[string]struct{}, error) {
	if r == nil || r.db == nil {
		return nil, service.ErrChatAttachmentUnavailable
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT storage_key FROM chat_attachments WHERE storage_key IS NOT NULL
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	keys := make(map[string]struct{})
	for rows.Next() {
		var key string
		if err = rows.Scan(&key); err != nil {
			return nil, err
		}
		if key != "" {
			keys[key] = struct{}{}
		}
	}
	return keys, rows.Err()
}
