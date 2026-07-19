package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type modelCatalogRepository struct {
	db *sql.DB
}

func NewModelCatalogRepository(db *sql.DB) service.ModelCatalogRepository {
	return &modelCatalogRepository{db: db}
}

const modelCatalogSelectColumns = `
id, slug, model, platform, metadata_model_id, display_name_zh, display_name_en,
summary_zh, summary_en, provider, logo_key, category, tags, capabilities,
context_window, max_output_tokens, public_group_id, status, featured, sort_order,
published_at, created_at, updated_at`

type modelCatalogRowScanner interface {
	Scan(dest ...any) error
}

func scanModelCatalogModel(scanner modelCatalogRowScanner) (*service.ModelCatalogModel, error) {
	var (
		model          service.ModelCatalogModel
		tagsJSON       []byte
		capabilityJSON []byte
		contextWindow  sql.NullInt64
		maxOutput      sql.NullInt64
		publicGroupID  sql.NullInt64
		publishedAt    sql.NullTime
	)
	if err := scanner.Scan(
		&model.ID, &model.Slug, &model.Model, &model.Platform, &model.MetadataModelID,
		&model.DisplayNameZH, &model.DisplayNameEN, &model.SummaryZH, &model.SummaryEN,
		&model.Provider, &model.LogoKey, &model.Category, &tagsJSON, &capabilityJSON,
		&contextWindow, &maxOutput, &publicGroupID, &model.Status, &model.Featured,
		&model.SortOrder, &publishedAt, &model.CreatedAt, &model.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if len(tagsJSON) > 0 {
		if err := json.Unmarshal(tagsJSON, &model.Tags); err != nil {
			return nil, fmt.Errorf("decode model catalog tags: %w", err)
		}
	}
	if model.Tags == nil {
		model.Tags = []string{}
	}
	if len(capabilityJSON) > 0 {
		if err := json.Unmarshal(capabilityJSON, &model.Capabilities); err != nil {
			return nil, fmt.Errorf("decode model catalog capabilities: %w", err)
		}
	}
	if model.Capabilities == nil {
		model.Capabilities = []string{}
	}
	if contextWindow.Valid {
		value := contextWindow.Int64
		model.ContextWindow = &value
	}
	if maxOutput.Valid {
		value := maxOutput.Int64
		model.MaxOutputTokens = &value
	}
	if publicGroupID.Valid {
		value := publicGroupID.Int64
		model.PublicGroupID = &value
	}
	if publishedAt.Valid {
		value := publishedAt.Time
		model.PublishedAt = &value
	}
	return &model, nil
}

func marshalCatalogStrings(values []string) ([]byte, error) {
	if values == nil {
		values = []string{}
	}
	return json.Marshal(values)
}

func (r *modelCatalogRepository) Create(ctx context.Context, model *service.ModelCatalogModel) error {
	tagsJSON, err := marshalCatalogStrings(model.Tags)
	if err != nil {
		return fmt.Errorf("encode model catalog tags: %w", err)
	}
	capabilitiesJSON, err := marshalCatalogStrings(model.Capabilities)
	if err != nil {
		return fmt.Errorf("encode model catalog capabilities: %w", err)
	}
	row := r.db.QueryRowContext(ctx, `
INSERT INTO model_catalog_models (
  slug, model, platform, metadata_model_id, display_name_zh, display_name_en,
  summary_zh, summary_en, provider, logo_key, category, tags, capabilities,
  context_window, max_output_tokens, public_group_id, status, featured, sort_order,
  published_at
) VALUES (
  $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12::jsonb,$13::jsonb,$14,$15,$16,$17,$18,$19,$20
)
RETURNING `+modelCatalogSelectColumns,
		model.Slug, model.Model, model.Platform, model.MetadataModelID,
		model.DisplayNameZH, model.DisplayNameEN, model.SummaryZH, model.SummaryEN,
		model.Provider, model.LogoKey, model.Category, string(tagsJSON), string(capabilitiesJSON),
		model.ContextWindow, model.MaxOutputTokens, model.PublicGroupID, model.Status,
		model.Featured, model.SortOrder, model.PublishedAt,
	)
	created, err := scanModelCatalogModel(row)
	if err != nil {
		if isUniqueViolation(err) {
			return service.ErrModelCatalogExists
		}
		if isModelCatalogInputViolation(err) {
			return service.ErrModelCatalogInvalid
		}
		return fmt.Errorf("insert model catalog entry: %w", err)
	}
	*model = *created
	return nil
}

func (r *modelCatalogRepository) GetByID(ctx context.Context, id int64) (*service.ModelCatalogModel, error) {
	model, err := scanModelCatalogModel(r.db.QueryRowContext(ctx,
		`SELECT `+modelCatalogSelectColumns+` FROM model_catalog_models WHERE id = $1`, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, service.ErrModelCatalogNotFound
		}
		return nil, fmt.Errorf("get model catalog entry: %w", err)
	}
	return model, nil
}

func (r *modelCatalogRepository) Update(ctx context.Context, model *service.ModelCatalogModel) error {
	tagsJSON, err := marshalCatalogStrings(model.Tags)
	if err != nil {
		return fmt.Errorf("encode model catalog tags: %w", err)
	}
	capabilitiesJSON, err := marshalCatalogStrings(model.Capabilities)
	if err != nil {
		return fmt.Errorf("encode model catalog capabilities: %w", err)
	}
	row := r.db.QueryRowContext(ctx, `
UPDATE model_catalog_models SET
  slug=$2, model=$3, platform=$4, metadata_model_id=$5,
  display_name_zh=$6, display_name_en=$7, summary_zh=$8, summary_en=$9,
  provider=$10, logo_key=$11, category=$12, tags=$13::jsonb, capabilities=$14::jsonb,
  context_window=$15, max_output_tokens=$16, public_group_id=$17, status=$18,
  featured=$19, sort_order=$20, published_at=$21, updated_at=NOW()
WHERE id=$1
RETURNING `+modelCatalogSelectColumns,
		model.ID, model.Slug, model.Model, model.Platform, model.MetadataModelID,
		model.DisplayNameZH, model.DisplayNameEN, model.SummaryZH, model.SummaryEN,
		model.Provider, model.LogoKey, model.Category, string(tagsJSON), string(capabilitiesJSON),
		model.ContextWindow, model.MaxOutputTokens, model.PublicGroupID, model.Status,
		model.Featured, model.SortOrder, model.PublishedAt,
	)
	updated, err := scanModelCatalogModel(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return service.ErrModelCatalogNotFound
		}
		if isUniqueViolation(err) {
			return service.ErrModelCatalogExists
		}
		if isModelCatalogInputViolation(err) {
			return service.ErrModelCatalogInvalid
		}
		return fmt.Errorf("update model catalog entry: %w", err)
	}
	*model = *updated
	return nil
}

func isModelCatalogInputViolation(err error) bool {
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) || pqErr == nil {
		return false
	}
	switch pqErr.Code {
	case "22001", "22003", "23503", "23514":
		return true
	default:
		return false
	}
}

func (r *modelCatalogRepository) ListAll(ctx context.Context) ([]service.ModelCatalogModel, error) {
	return r.list(ctx, `
SELECT `+modelCatalogSelectColumns+`
FROM model_catalog_models
ORDER BY featured DESC, sort_order ASC, id ASC`)
}

func (r *modelCatalogRepository) ListPublished(ctx context.Context) ([]service.ModelCatalogModel, error) {
	return r.list(ctx, `
SELECT `+modelCatalogSelectColumns+`
FROM model_catalog_models
WHERE status = 'published'
ORDER BY featured DESC, sort_order ASC, id ASC`)
}

func (r *modelCatalogRepository) list(ctx context.Context, query string) ([]service.ModelCatalogModel, error) {
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list model catalog entries: %w", err)
	}
	defer rows.Close()
	out := make([]service.ModelCatalogModel, 0)
	for rows.Next() {
		model, err := scanModelCatalogModel(rows)
		if err != nil {
			return nil, fmt.Errorf("scan model catalog entry: %w", err)
		}
		out = append(out, *model)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate model catalog entries: %w", err)
	}
	return out, nil
}
