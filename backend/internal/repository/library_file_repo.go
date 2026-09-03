package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type libraryFileRepository struct {
	db               *sql.DB
	uploadsPerMinute int
	dailyUploadBytes int64
}

type libraryFileQueryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// NewLibraryFileRepository builds the durable metadata repository. Storage
// entitlements are resolved by the service and passed to CreatePending;
// the repository deliberately has no dependency on application configuration.
func NewLibraryFileRepository(db *sql.DB, cfg *config.Config) service.LibraryFileRepository {
	uploadsPerMinute := 10
	dailyUploadBytes := int64(100 << 20)
	if cfg != nil {
		uploadsPerMinute = cfg.ChatAttachments.UploadsPerMinute
		dailyUploadBytes = cfg.ChatAttachments.DailyUploadBytes
	}
	if uploadsPerMinute <= 0 {
		uploadsPerMinute = 10
	}
	if dailyUploadBytes <= 0 {
		dailyUploadBytes = 100 << 20
	}
	return &libraryFileRepository{
		db:               db,
		uploadsPerMinute: uploadsPerMinute,
		dailyUploadBytes: dailyUploadBytes,
	}
}

const libraryFileColumns = `
	f.id,
	f.public_id,
	f.user_id,
	f.original_name,
	f.storage_kind,
	f.category,
	f.file_type,
	f.source,
	COALESCE(f.source_key, ''),
	f.mime_type,
	f.extension,
	f.byte_size,
	f.stored_size,
	f.status,
	f.page_count,
	f.width,
	f.height,
	COALESCE(f.storage_key, ''),
	f.sha256,
	f.created_at,
	f.updated_at,
	f.last_used_at,
	f.deleted_at,
	f.cleaned_at`

func scanLibraryFile(scan chatHistoryScanner) (*service.LibraryFile, error) {
	var (
		file       service.LibraryFile
		pageCount  sql.NullInt64
		width      sql.NullInt64
		height     sql.NullInt64
		lastUsedAt sql.NullTime
		deletedAt  sql.NullTime
		cleanedAt  sql.NullTime
	)
	if err := scan(
		&file.InternalID, &file.ID, &file.UserID, &file.Name, &file.Format,
		&file.Category, &file.Type, &file.Source, &file.SourceKey, &file.MIMEType, &file.Extension,
		&file.Size, &file.StoredSize, &file.Status, &pageCount, &width, &height,
		&file.StorageKey, &file.Digest, &file.CreatedAt,
		&file.UpdatedAt, &lastUsedAt, &deletedAt, &cleanedAt,
	); err != nil {
		return nil, err
	}
	if pageCount.Valid {
		file.PageCount = int(pageCount.Int64)
	}
	if width.Valid {
		file.Width = int(width.Int64)
	}
	if height.Valid {
		file.Height = int(height.Int64)
	}
	if lastUsedAt.Valid {
		value := lastUsedAt.Time
		file.LastUsedAt = &value
	}
	if deletedAt.Valid {
		value := deletedAt.Time
		file.DeletedAt = &value
	}
	if cleanedAt.Valid {
		value := cleanedAt.Time
		file.CleanedAt = &value
	}
	return &file, nil
}

func (r *libraryFileRepository) ResolveStorageLimit(
	ctx context.Context,
	userID, defaultBytes int64,
	groupOverrides map[int64]int64,
) (int64, error) {
	if r == nil || r.db == nil || userID <= 0 {
		return 0, service.ErrChatAttachmentUnavailable
	}
	if len(groupOverrides) == 0 {
		return defaultBytes, nil
	}
	groupIDs := make([]int64, 0, len(groupOverrides))
	for groupID := range groupOverrides {
		if groupID > 0 {
			groupIDs = append(groupIDs, groupID)
		}
	}
	sort.Slice(groupIDs, func(i, j int) bool { return groupIDs[i] < groupIDs[j] })
	if len(groupIDs) == 0 {
		return defaultBytes, nil
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT group_id
		FROM user_subscriptions
		WHERE user_id=$1 AND status='active' AND expires_at > NOW()
		  AND deleted_at IS NULL AND group_id = ANY($2)
	`, userID, pq.Array(groupIDs))
	if err != nil {
		return 0, err
	}
	defer func() { _ = rows.Close() }()
	limit := defaultBytes
	matched := false
	for rows.Next() {
		var groupID int64
		if err = rows.Scan(&groupID); err != nil {
			return 0, err
		}
		candidate, ok := groupOverrides[groupID]
		if !ok {
			continue
		}
		// Zero is an explicit unlimited entitlement and therefore dominates all
		// finite limits. Negative values are ignored as invalid service input.
		if candidate == 0 {
			return 0, nil
		}
		if candidate > 0 && (!matched || candidate > limit) {
			limit = candidate
		}
		matched = true
	}
	if err = rows.Err(); err != nil {
		return 0, err
	}
	if !matched {
		return defaultBytes, nil
	}
	return limit, nil
}

func (r *libraryFileRepository) CheckUploadQuota(ctx context.Context, userID, rawSize int64) error {
	if r == nil || r.db == nil || userID <= 0 || rawSize <= 0 {
		return service.ErrChatAttachmentRateLimit
	}
	var uploadsLastMinute int
	var bytesLastDay int64
	if err := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '1 minute'),
			COALESCE(SUM(byte_size) FILTER (WHERE created_at >= NOW() - INTERVAL '24 hours'), 0)
		FROM library_files
		WHERE user_id=$1 AND source='uploaded'
	`, userID).Scan(&uploadsLastMinute, &bytesLastDay); err != nil {
		return err
	}
	if uploadsLastMinute >= r.uploadsPerMinute || rawSize > r.dailyUploadBytes || bytesLastDay > r.dailyUploadBytes-rawSize {
		return service.ErrChatAttachmentRateLimit
	}
	return nil
}

func (r *libraryFileRepository) CreatePending(ctx context.Context, input *service.CreateLibraryFileInput, limitBytes int64) (*service.LibraryFile, error) {
	if r == nil || r.db == nil || input == nil || input.UserID <= 0 {
		return nil, service.ErrChatAttachmentUnavailable
	}
	f := input.File
	if f.Size <= 0 || f.StoredSize <= 0 {
		return nil, service.ErrLibraryInvalidRequest
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var lockResult any
	if err = tx.QueryRowContext(ctx,
		`SELECT pg_advisory_xact_lock(hashtext('library_files'), hashint8($1::bigint))`, input.UserID,
	).Scan(&lockResult); err != nil {
		return nil, err
	}
	if input.SourceKey != "" {
		row := tx.QueryRowContext(ctx, `SELECT `+libraryFileColumns+`
			FROM library_files f
			WHERE f.source_key=$1
			FOR UPDATE`, input.SourceKey)
		existing, lookupErr := scanLibraryFile(row.Scan)
		if lookupErr == nil {
			if existing.UserID != input.UserID {
				return nil, service.ErrLibraryFileAccessDenied
			}
			if err = tx.Commit(); err != nil {
				return nil, err
			}
			return existing, nil
		}
		if !errors.Is(lookupErr, sql.ErrNoRows) {
			return nil, lookupErr
		}
	}

	var uploadsLastMinute int
	var bytesLastDay int64
	var usedBytes int64
	if err = tx.QueryRowContext(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE source='uploaded' AND created_at >= NOW() - INTERVAL '1 minute'),
			COALESCE(SUM(byte_size) FILTER (WHERE source='uploaded' AND created_at >= NOW() - INTERVAL '24 hours'), 0),
			COALESCE(SUM(stored_size) FILTER (WHERE status IN ('pending','ready')), 0)
		FROM library_files
		WHERE user_id=$1
	`, input.UserID).Scan(&uploadsLastMinute, &bytesLastDay, &usedBytes); err != nil {
		return nil, err
	}
	if !input.BypassUploadQuota && (uploadsLastMinute >= r.uploadsPerMinute || f.Size > r.dailyUploadBytes || bytesLastDay > r.dailyUploadBytes-f.Size) {
		return nil, service.ErrChatAttachmentRateLimit
	}
	if limitBytes > 0 && (f.StoredSize > limitBytes || usedBytes > limitBytes-f.StoredSize) {
		return nil, service.ErrLibraryStorageLimit.WithMetadata(map[string]string{
			"used_bytes":     fmt.Sprintf("%d", usedBytes),
			"limit_bytes":    fmt.Sprintf("%d", limitBytes),
			"incoming_bytes": fmt.Sprintf("%d", f.StoredSize),
		})
	}

	row := tx.QueryRowContext(ctx, `
		INSERT INTO library_files AS f (
			public_id, user_id, original_name, storage_kind, category, file_type,
			source, source_key, mime_type, extension, byte_size, stored_size, sha256,
			storage_key, page_count, width, height, status
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,NULLIF($8,''),$9,$10,$11,$12,$13,NULLIF($14,''),
			$15,$16,$17,$18
		)
		RETURNING `+libraryFileColumns,
		f.ID, input.UserID, f.Name, f.Format, f.Category, f.Type, f.Source,
		input.SourceKey, f.MIMEType, f.Extension, f.Size, f.StoredSize, f.Digest, f.StorageKey,
		nullablePositive(f.PageCount), nullablePositive(f.Width),
		nullablePositive(f.Height), f.Status,
	)
	created, err := scanLibraryFile(row.Scan)
	if err != nil {
		return nil, err
	}

	aliasKind := service.ChatAttachmentFormatDOCX
	if f.Category == "image" {
		aliasKind = service.ChatAttachmentKindImage
	} else if f.Type == "pdf" {
		aliasKind = service.ChatAttachmentFormatPDF
	}
	aliasExpiresAt := input.AliasExpiresAt
	if aliasExpiresAt.IsZero() {
		// expires_at remains NOT NULL for the pre-existing chat schema. Library
		// aliases are excluded from every expiry predicate by library_file_id.
		aliasExpiresAt = time.Now().UTC().AddDate(100, 0, 0)
	}
	if _, err = tx.ExecContext(ctx, `
		INSERT INTO chat_attachments (
			public_id, user_id, conversation_id, library_file_id, original_name,
			kind, mime_type, byte_size, stored_size, sha256, storage_key,
			extracted_text, page_count, width, height, status, expires_at
		) VALUES ($1,$2,NULL,$3,$4,$5,$6,$7,$8,$9,NULL,NULL,$10,$11,$12,$13,$14)
	`, f.ID, input.UserID, created.InternalID, f.Name, aliasKind, f.MIMEType,
		f.Size, f.StoredSize, f.Digest,
		nullablePositive(f.PageCount), nullablePositive(f.Width), nullablePositive(f.Height),
		f.Status, aliasExpiresAt,
	); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return created, nil
}

func (r *libraryFileRepository) Activate(ctx context.Context, userID int64, publicID string) (*service.LibraryFile, error) {
	if r == nil || r.db == nil || userID <= 0 {
		return nil, service.ErrLibraryFileNotFound
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	row := tx.QueryRowContext(ctx, `
		UPDATE library_files AS f
		SET status='ready', updated_at=NOW()
		WHERE f.user_id=$1 AND f.public_id=$2 AND f.status='pending'
		RETURNING `+libraryFileColumns, userID, publicID)
	file, err := scanLibraryFile(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrLibraryFileNotFound
	}
	if err != nil {
		return nil, err
	}
	result, err := tx.ExecContext(ctx, `
		UPDATE chat_attachments
		SET status='ready', updated_at=NOW()
		WHERE user_id=$1 AND public_id=$2 AND library_file_id=$3 AND status='pending'
	`, userID, publicID, file.InternalID)
	if err != nil {
		return nil, err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if updated != 1 {
		return nil, service.ErrChatAttachmentUnavailable
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return file, nil
}

func (r *libraryFileRepository) Abort(ctx context.Context, userID int64, publicID string) (*service.LibraryFile, error) {
	if r == nil || r.db == nil || userID <= 0 {
		return nil, service.ErrLibraryFileNotFound
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	var lockResult any
	if err = tx.QueryRowContext(ctx,
		`SELECT pg_advisory_xact_lock(hashtext('library_files'), hashint8($1::bigint))`, userID,
	).Scan(&lockResult); err != nil {
		return nil, err
	}
	row := tx.QueryRowContext(ctx, `
		UPDATE library_files AS f
		SET status='deleted', deleted_at=NOW(), updated_at=NOW(), cleanup_claimed_at=NULL,
		    source_key=CASE WHEN source='generated' THEN NULL ELSE source_key END
		WHERE f.user_id=$1 AND f.public_id=$2 AND f.status='pending'
		RETURNING `+libraryFileColumns, userID, publicID)
	file, err := scanLibraryFile(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, r.libraryLookupError(ctx, tx, userID, publicID)
	}
	if err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `
		UPDATE chat_attachments
		SET status='deleted', extracted_text=NULL, updated_at=NOW()
		WHERE user_id=$1 AND public_id=$2 AND library_file_id=$3 AND status='pending'
	`, userID, publicID, file.InternalID); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return file, nil
}

func (r *libraryFileRepository) GetOwned(ctx context.Context, userID int64, publicID string) (*service.LibraryFile, error) {
	if r == nil || r.db == nil || userID <= 0 {
		return nil, service.ErrLibraryFileNotFound
	}
	row := r.db.QueryRowContext(ctx, `SELECT `+libraryFileColumns+`
		FROM library_files f
		WHERE f.user_id=$1 AND f.public_id=$2 AND f.status='ready'`, userID, publicID)
	file, err := scanLibraryFile(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, r.libraryLookupError(ctx, r.db, userID, publicID)
	}
	return file, err
}

func (r *libraryFileRepository) List(ctx context.Context, userID int64, query service.LibraryFileQuery) (*service.LibraryFilePage, error) {
	if r == nil || r.db == nil || userID <= 0 {
		return nil, service.ErrChatAttachmentUnavailable
	}
	where := []string{"f.user_id=$1", "f.status='ready'"}
	args := []any{userID}
	addArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if query.Q != "" {
		where = append(where, `f.original_name ILIKE '%' || `+addArg(escapeLibraryLike(query.Q))+` || '%' ESCAPE '\'`)
	}
	if query.Category != "" && query.Category != "all" {
		where = append(where, "f.category="+addArg(query.Category))
	}
	if query.Source != "" && query.Source != "all" {
		where = append(where, "f.source="+addArg(query.Source))
	}
	if query.Type != "" && query.Type != "all" {
		where = append(where, "f.file_type="+addArg(query.Type))
	}
	whereSQL := strings.Join(where, " AND ")
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM library_files f WHERE "+whereSQL, args...).Scan(&total); err != nil {
		return nil, err
	}
	orderBy := libraryOrderBy(query.Sort)
	page := query.Page
	if page <= 0 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	listArgs := append(append([]any(nil), args...), pageSize, (page-1)*pageSize)
	rows, err := r.db.QueryContext(ctx, `SELECT `+libraryFileColumns+`
		FROM library_files f
		WHERE `+whereSQL+`
		ORDER BY `+orderBy+`
		LIMIT $`+fmt.Sprintf("%d", len(args)+1)+` OFFSET $`+fmt.Sprintf("%d", len(args)+2), listArgs...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	files := make([]service.LibraryFile, 0, pageSize)
	for rows.Next() {
		file, scanErr := scanLibraryFile(rows.Scan)
		if scanErr != nil {
			return nil, scanErr
		}
		files = append(files, *file)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return &service.LibraryFilePage{Files: files, Total: total, Page: page, PageSize: pageSize}, nil
}

func libraryOrderBy(sortValue string) string {
	switch sortValue {
	case "updated_asc":
		return "f.updated_at ASC, f.id ASC"
	case "name_asc":
		return "LOWER(f.original_name) ASC, f.id ASC"
	case "name_desc":
		return "LOWER(f.original_name) DESC, f.id DESC"
	case "size_asc":
		return "f.byte_size ASC, f.id ASC"
	case "size_desc":
		return "f.byte_size DESC, f.id DESC"
	default:
		return "f.updated_at DESC, f.id DESC"
	}
}

func escapeLibraryLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	return strings.ReplaceAll(value, `_`, `\_`)
}

func (r *libraryFileRepository) ResolveOwned(ctx context.Context, userID int64, publicIDs []string) ([]service.LibraryFile, error) {
	if r == nil || r.db == nil || userID <= 0 || len(publicIDs) == 0 {
		return nil, nil
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+libraryFileColumns+`
		FROM unnest($2::text[]) WITH ORDINALITY requested(public_id, position)
		JOIN library_files f ON f.public_id=requested.public_id
		 AND f.user_id=$1 AND f.status='ready'
		ORDER BY requested.position
	`, userID, pq.Array(publicIDs))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	files := make([]service.LibraryFile, 0, len(publicIDs))
	for rows.Next() {
		file, scanErr := scanLibraryFile(rows.Scan)
		if scanErr != nil {
			return nil, scanErr
		}
		files = append(files, *file)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if len(files) != len(publicIDs) {
		return nil, r.libraryResolveError(ctx, userID, publicIDs)
	}
	return files, nil
}

func (r *libraryFileRepository) libraryLookupError(
	ctx context.Context,
	queryer libraryFileQueryer,
	userID int64,
	publicID string,
) error {
	var ownerID int64
	err := queryer.QueryRowContext(ctx, `
		SELECT user_id FROM library_files WHERE public_id=$1 AND status='ready'
	`, publicID).Scan(&ownerID)
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrLibraryFileNotFound
	}
	if err != nil {
		return err
	}
	if ownerID != userID {
		return service.ErrLibraryFileAccessDenied
	}
	return service.ErrLibraryFileNotFound
}

func (r *libraryFileRepository) libraryResolveError(ctx context.Context, userID int64, publicIDs []string) error {
	rows, err := r.db.QueryContext(ctx, `
		SELECT public_id, user_id
		FROM library_files
		WHERE public_id = ANY($1) AND status='ready'
	`, pq.Array(publicIDs))
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var publicID string
		var ownerID int64
		if err = rows.Scan(&publicID, &ownerID); err != nil {
			return err
		}
		if ownerID != userID {
			return service.ErrLibraryFileAccessDenied
		}
	}
	if err = rows.Err(); err != nil {
		return err
	}
	return service.ErrLibraryFileNotFound
}

func (r *libraryFileRepository) MarkDeleted(ctx context.Context, userID int64, publicID string) (*service.LibraryFile, error) {
	if r == nil || r.db == nil || userID <= 0 {
		return nil, service.ErrLibraryFileNotFound
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	var lockResult any
	if err = tx.QueryRowContext(ctx,
		`SELECT pg_advisory_xact_lock(hashtext('library_files'), hashint8($1::bigint))`, userID,
	).Scan(&lockResult); err != nil {
		return nil, err
	}
	row := tx.QueryRowContext(ctx, `
		UPDATE library_files AS f
		SET status='deleted', deleted_at=NOW(), updated_at=NOW(), cleanup_claimed_at=NULL
		WHERE f.user_id=$1 AND f.public_id=$2 AND f.status IN ('pending','ready')
		RETURNING `+libraryFileColumns, userID, publicID)
	file, err := scanLibraryFile(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, r.libraryLookupError(ctx, tx, userID, publicID)
	}
	if err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `
		UPDATE chat_attachments
		SET status='deleted', extracted_text=NULL, updated_at=NOW()
		WHERE user_id=$1 AND public_id=$2 AND library_file_id=$3
	`, userID, publicID, file.InternalID); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return file, nil
}

func (r *libraryFileRepository) Usage(ctx context.Context, userID int64) (int64, error) {
	if r == nil || r.db == nil || userID <= 0 {
		return 0, service.ErrChatAttachmentUnavailable
	}
	var used int64
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(stored_size), 0)
		FROM library_files
		WHERE user_id=$1 AND status='ready'
	`, userID).Scan(&used)
	return used, err
}

func (r *libraryFileRepository) ClaimCleanup(
	ctx context.Context,
	deletedBefore, pendingBefore time.Time,
	limit int,
) ([]service.LibraryFile, error) {
	if r == nil || r.db == nil {
		return nil, service.ErrChatAttachmentUnavailable
	}
	if limit <= 0 {
		return nil, nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	rows, err := tx.QueryContext(ctx, `
		WITH candidates AS (
			SELECT id, status, created_at, deleted_at
			FROM library_files
			WHERE storage_key IS NOT NULL
			  AND (
				(status='deleted' AND deleted_at <= $1) OR
				(status='pending' AND created_at <= $2)
			  )
			  AND (cleanup_claimed_at IS NULL OR cleanup_claimed_at < NOW() - INTERVAL '15 minutes')
			ORDER BY CASE WHEN status='pending' THEN created_at ELSE deleted_at END, id
			LIMIT $3
			FOR UPDATE SKIP LOCKED
		)
		UPDATE library_files AS f
		SET status='deleted', deleted_at=COALESCE(f.deleted_at, NOW()),
			cleanup_claimed_at=NOW(), updated_at=NOW(),
			source_key=CASE WHEN candidates.status='pending' THEN NULL ELSE f.source_key END
		FROM candidates
		WHERE f.id=candidates.id
		RETURNING `+libraryFileColumns,
		deletedBefore, pendingBefore, limit)
	if err != nil {
		return nil, err
	}
	files := make([]service.LibraryFile, 0, limit)
	for rows.Next() {
		file, scanErr := scanLibraryFile(rows.Scan)
		if scanErr != nil {
			_ = rows.Close()
			return nil, scanErr
		}
		files = append(files, *file)
	}
	if err = rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err = rows.Close(); err != nil {
		return nil, err
	}
	claimedIDs := make([]int64, 0, len(files))
	for i := range files {
		claimedIDs = append(claimedIDs, files[i].InternalID)
	}
	if _, err = tx.ExecContext(ctx, `
		UPDATE chat_attachments AS a
		SET status='deleted', extracted_text=NULL, updated_at=NOW()
		WHERE a.library_file_id=ANY($1)
		  AND a.status='pending'
	`, pq.Array(claimedIDs)); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return files, nil
}

func (r *libraryFileRepository) FinalizeCleanup(ctx context.Context, userID int64, publicID string) error {
	if r == nil || r.db == nil || userID <= 0 {
		return service.ErrLibraryFileNotFound
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(ctx, `
		UPDATE library_files
		SET storage_key=NULL, cleaned_at=NOW(),
		    cleanup_claimed_at=NULL, updated_at=NOW()
		WHERE user_id=$1 AND public_id=$2 AND status='deleted'
	`, userID, publicID)
	if err != nil {
		return err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if updated == 0 {
		return service.ErrLibraryFileNotFound
	}
	if _, err = tx.ExecContext(ctx, `
		UPDATE chat_attachments
		SET storage_key=NULL, extracted_text=NULL, cleaned_at=NOW(), updated_at=NOW()
		WHERE user_id=$1 AND public_id=$2 AND library_file_id IS NOT NULL AND status='deleted'
	`, userID, publicID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *libraryFileRepository) ListStorageKeys(ctx context.Context) (map[string]struct{}, error) {
	if r == nil || r.db == nil {
		return nil, service.ErrChatAttachmentUnavailable
	}
	rows, err := r.db.QueryContext(ctx, `SELECT storage_key FROM library_files WHERE storage_key IS NOT NULL`)
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
