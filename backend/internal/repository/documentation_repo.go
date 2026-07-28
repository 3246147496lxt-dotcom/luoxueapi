package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type documentationRepository struct {
	db *sql.DB
}

func NewDocumentationRepository(db *sql.DB) service.DocumentationRepository {
	return &documentationRepository{db: db}
}

func (r *documentationRepository) EnsureSeed(ctx context.Context, content json.RawMessage) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin documentation seed: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var insertedID int
	err = tx.QueryRowContext(ctx, `
INSERT INTO documentation_documents (
  id, draft_content, draft_version, draft_updated_at,
  published_content, published_version, published_at
) VALUES (1, $1::jsonb, 1, NOW(), $1::jsonb, 1, NOW())
ON CONFLICT (id) DO UPDATE SET
  draft_content = EXCLUDED.draft_content,
  draft_updated_at = NOW(),
  published_content = EXCLUDED.published_content,
  published_at = NOW()
WHERE documentation_documents.draft_version = 1
  AND documentation_documents.published_version = 1
  AND documentation_documents.draft_updated_by IS NULL
  AND documentation_documents.published_by IS NULL
RETURNING id`, string(content)).Scan(&insertedID)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("seed documentation state: %w", err)
	}
	if err == nil {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO documentation_revisions (version, content, published_at)
VALUES (1, $1::jsonb, NOW())
ON CONFLICT (version) DO UPDATE SET
  content = EXCLUDED.content,
  published_at = EXCLUDED.published_at
WHERE documentation_revisions.published_by IS NULL`, string(content)); err != nil {
			return fmt.Errorf("seed documentation revision: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit documentation seed: %w", err)
	}
	return nil
}

type documentationRowScanner interface {
	Scan(dest ...any) error
}

const documentationStateColumns = `
draft_content, draft_version, draft_updated_at, draft_updated_by,
published_content, published_version, published_at, published_by`

func scanDocumentationState(scanner documentationRowScanner) (*service.DocumentationAdminState, error) {
	var (
		draftContent     []byte
		draftVersion     int64
		draftUpdatedAt   sql.NullTime
		draftUpdatedBy   sql.NullInt64
		publishedContent []byte
		publishedVersion int64
		publishedAt      sql.NullTime
		publishedBy      sql.NullInt64
	)
	if err := scanner.Scan(
		&draftContent, &draftVersion, &draftUpdatedAt, &draftUpdatedBy,
		&publishedContent, &publishedVersion, &publishedAt, &publishedBy,
	); err != nil {
		return nil, err
	}
	state := &service.DocumentationAdminState{}
	if len(draftContent) > 0 && draftUpdatedAt.Valid {
		state.Draft = &service.DocumentationSnapshot{
			Content:   json.RawMessage(append([]byte(nil), draftContent...)),
			Version:   draftVersion,
			UpdatedAt: draftUpdatedAt.Time,
			UpdatedBy: nullableInt64Pointer(draftUpdatedBy),
		}
	}
	if len(publishedContent) > 0 && publishedAt.Valid && publishedVersion > 0 {
		state.Published = &service.DocumentationSnapshot{
			Content:   json.RawMessage(append([]byte(nil), publishedContent...)),
			Version:   publishedVersion,
			UpdatedAt: publishedAt.Time,
			UpdatedBy: nullableInt64Pointer(publishedBy),
		}
	}
	return state, nil
}

func nullableInt64Pointer(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	copy := value.Int64
	return &copy
}

func (r *documentationRepository) GetState(ctx context.Context) (*service.DocumentationAdminState, error) {
	state, err := scanDocumentationState(r.db.QueryRowContext(ctx,
		`SELECT `+documentationStateColumns+` FROM documentation_documents WHERE id = 1`))
	if err == sql.ErrNoRows {
		return &service.DocumentationAdminState{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get documentation state: %w", err)
	}
	return state, nil
}

func (r *documentationRepository) SaveDraft(ctx context.Context, content json.RawMessage, actorID *int64) (*service.DocumentationAdminState, error) {
	state, err := scanDocumentationState(r.db.QueryRowContext(ctx, `
INSERT INTO documentation_documents (
  id, draft_content, draft_version, draft_updated_at, draft_updated_by
) VALUES (1, $1::jsonb, 1, NOW(), $2)
ON CONFLICT (id) DO UPDATE SET
  draft_content = EXCLUDED.draft_content,
  draft_version = documentation_documents.draft_version + 1,
  draft_updated_at = NOW(),
  draft_updated_by = EXCLUDED.draft_updated_by
RETURNING `+documentationStateColumns, string(content), actorID))
	if err != nil {
		return nil, fmt.Errorf("save documentation draft: %w", err)
	}
	return state, nil
}

func (r *documentationRepository) Publish(ctx context.Context, actorID *int64) (*service.DocumentationAdminState, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin documentation publish: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var content []byte
	var publishedVersion int64
	if err := tx.QueryRowContext(ctx, `
SELECT draft_content, published_version
FROM documentation_documents
WHERE id = 1
FOR UPDATE`).Scan(&content, &publishedVersion); err != nil {
		if err == sql.ErrNoRows {
			return nil, service.ErrDocumentationNoDraft
		}
		return nil, fmt.Errorf("lock documentation draft: %w", err)
	}
	if len(content) == 0 {
		return nil, service.ErrDocumentationNoDraft
	}
	// Validate the row while holding the singleton lock. This closes the race
	// where another admin saves an incomplete draft between the service's
	// preflight validation and this publish transaction.
	if _, err := service.ValidateDocumentationContent(content); err != nil {
		return nil, err
	}
	version := publishedVersion + 1
	if _, err := tx.ExecContext(ctx, `
INSERT INTO documentation_revisions (version, content, published_at, published_by)
VALUES ($1, $2::jsonb, NOW(), $3)`, version, string(content), actorID); err != nil {
		return nil, fmt.Errorf("create documentation revision: %w", err)
	}
	state, err := scanDocumentationState(tx.QueryRowContext(ctx, `
UPDATE documentation_documents SET
  published_content = $1::jsonb,
  published_version = $2,
  published_at = NOW(),
  published_by = $3
WHERE id = 1
RETURNING `+documentationStateColumns, string(content), version, actorID))
	if err != nil {
		return nil, fmt.Errorf("publish documentation: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit documentation publish: %w", err)
	}
	return state, nil
}

func (r *documentationRepository) ListRevisions(ctx context.Context) ([]service.DocumentationRevision, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, version, published_at, published_by
FROM documentation_revisions
ORDER BY version DESC`)
	if err != nil {
		return nil, fmt.Errorf("list documentation revisions: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]service.DocumentationRevision, 0)
	for rows.Next() {
		var item service.DocumentationRevision
		var publishedBy sql.NullInt64
		if err := rows.Scan(&item.ID, &item.Version, &item.PublishedAt, &publishedBy); err != nil {
			return nil, fmt.Errorf("scan documentation revision: %w", err)
		}
		item.PublishedBy = nullableInt64Pointer(publishedBy)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate documentation revisions: %w", err)
	}
	return items, nil
}

func (r *documentationRepository) RestoreRevision(ctx context.Context, revisionID int64, actorID *int64) (*service.DocumentationAdminState, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin documentation restore: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var content []byte
	if err := tx.QueryRowContext(ctx,
		`SELECT content FROM documentation_revisions WHERE id = $1`, revisionID).Scan(&content); err != nil {
		if err == sql.ErrNoRows {
			return nil, service.ErrDocumentationNotFound
		}
		return nil, fmt.Errorf("get documentation revision: %w", err)
	}
	state, err := scanDocumentationState(tx.QueryRowContext(ctx, `
UPDATE documentation_documents SET
  draft_content = $1::jsonb,
  draft_version = draft_version + 1,
  draft_updated_at = NOW(),
  draft_updated_by = $2
WHERE id = 1
RETURNING `+documentationStateColumns, string(content), actorID))
	if err != nil {
		return nil, fmt.Errorf("restore documentation revision: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit documentation restore: %w", err)
	}
	return state, nil
}

func (r *documentationRepository) GetPublished(ctx context.Context) (*service.DocumentationSnapshot, error) {
	var (
		content     []byte
		version     int64
		publishedAt sql.NullTime
		publishedBy sql.NullInt64
	)
	err := r.db.QueryRowContext(ctx, `
SELECT published_content, published_version, published_at, published_by
FROM documentation_documents
WHERE id = 1 AND published_content IS NOT NULL AND published_version > 0`).Scan(
		&content, &version, &publishedAt, &publishedBy,
	)
	if err == sql.ErrNoRows {
		return nil, service.ErrDocumentationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get published documentation: %w", err)
	}
	if len(content) == 0 || !publishedAt.Valid {
		return nil, service.ErrDocumentationNotFound
	}
	return &service.DocumentationSnapshot{
		Content:   json.RawMessage(append([]byte(nil), content...)),
		Version:   version,
		UpdatedAt: publishedAt.Time,
		UpdatedBy: nullableInt64Pointer(publishedBy),
	}, nil
}

func (r *documentationRepository) SaveAsset(ctx context.Context, asset *service.DocumentationAsset) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO documentation_assets (
  id, content_type, byte_size, width, height, data, created_at, created_by
) VALUES ($1, $2, $3, $4, $5, $6, NOW(), $7)
ON CONFLICT (id) DO NOTHING`,
		asset.ID, asset.ContentType, asset.ByteSize, asset.Width, asset.Height, asset.Data, asset.CreatedBy,
	)
	if err != nil {
		return fmt.Errorf("save documentation asset: %w", err)
	}
	return nil
}

func (r *documentationRepository) GetAsset(ctx context.Context, id string) (*service.DocumentationAsset, error) {
	asset := &service.DocumentationAsset{ID: id}
	var createdBy sql.NullInt64
	if err := r.db.QueryRowContext(ctx, `
SELECT content_type, byte_size, width, height, data, created_at, created_by
FROM documentation_assets
WHERE id = $1`, id).Scan(
		&asset.ContentType, &asset.ByteSize, &asset.Width, &asset.Height,
		&asset.Data, &asset.CreatedAt, &createdBy,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, service.ErrDocumentationNotFound
		}
		return nil, fmt.Errorf("get documentation asset: %w", err)
	}
	asset.CreatedBy = nullableInt64Pointer(createdBy)
	return asset, nil
}
