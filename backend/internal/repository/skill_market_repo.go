package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type skillMarketRepository struct{ db *sql.DB }

func NewSkillMarketRepository(db *sql.DB) service.SkillMarketRepository {
	return &skillMarketRepository{db: db}
}

type skillMarketScanner interface{ Scan(...any) error }

const skillMarketSelectColumns = `
s.id, s.slug, s.display_name, s.summary, s.description, s.category, s.tags,
s.icon, s.example_prompts, s.risk_notes, s.origin_url, s.source_url, s.source_repository,
s.repository_stars, s.repository_stars_fetched_at, s.repository_stars_refresh_after,
s.status, s.featured, s.sort_order, s.catalog_source_priority, s.catalog_source_rank,
s.current_version_id, s.published_at, s.archived_at, s.created_by, s.updated_by,
s.created_at, s.updated_at,
COALESCE((SELECT SUM(v.download_count) FROM skill_versions v WHERE v.skill_id = s.id), 0)`

const skillMarketLocalizedSelectColumns = `
s.id, s.slug,
COALESCE(NULLIF(catalog_copy.display_name, ''), s.display_name),
COALESCE(NULLIF(catalog_copy.summary, ''), s.summary),
COALESCE(NULLIF(catalog_copy.description, ''), s.description),
s.category, s.tags,
s.icon, s.example_prompts, s.risk_notes, s.origin_url, s.source_url, s.source_repository,
s.repository_stars, s.repository_stars_fetched_at, s.repository_stars_refresh_after,
s.status, s.featured, s.sort_order, s.catalog_source_priority, s.catalog_source_rank,
s.current_version_id, s.published_at, s.archived_at, s.created_by, s.updated_by,
s.created_at, s.updated_at,
COALESCE((SELECT SUM(v.download_count) FROM skill_versions v WHERE v.skill_id = s.id), 0)`

func scanSkillMarket(scanner skillMarketScanner) (*service.Skill, error) {
	var (
		skill                   service.Skill
		tagsJSON, promptsJSON   []byte
		repositoryStars         sql.NullInt64
		catalogSourceRank       sql.NullInt64
		starsFetchedAt          sql.NullTime
		starsRefreshAfter       sql.NullTime
		currentVersionID        sql.NullInt64
		publishedAt, archivedAt sql.NullTime
		createdBy, updatedBy    sql.NullInt64
	)
	if err := scanner.Scan(
		&skill.ID, &skill.Slug, &skill.DisplayName, &skill.Summary, &skill.Description,
		&skill.Category, &tagsJSON, &skill.Icon, &promptsJSON, &skill.RiskNotes,
		&skill.OriginURL, &skill.SourceURL, &skill.SourceRepository, &repositoryStars, &starsFetchedAt,
		&starsRefreshAfter,
		&skill.Status, &skill.Featured, &skill.SortOrder, &skill.CatalogSourcePriority,
		&catalogSourceRank, &currentVersionID,
		&publishedAt, &archivedAt, &createdBy, &updatedBy, &skill.CreatedAt,
		&skill.UpdatedAt, &skill.DownloadCount,
	); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(tagsJSON, &skill.Tags); err != nil {
		return nil, fmt.Errorf("decode skill tags: %w", err)
	}
	if err := json.Unmarshal(promptsJSON, &skill.ExamplePrompts); err != nil {
		return nil, fmt.Errorf("decode skill example prompts: %w", err)
	}
	if skill.Tags == nil {
		skill.Tags = []string{}
	}
	if skill.ExamplePrompts == nil {
		skill.ExamplePrompts = []string{}
	}
	if repositoryStars.Valid {
		value := repositoryStars.Int64
		skill.RepositoryStars = &value
	}
	if catalogSourceRank.Valid {
		value := int(catalogSourceRank.Int64)
		skill.CatalogSourceRank = &value
	}
	if starsFetchedAt.Valid {
		value := starsFetchedAt.Time
		skill.RepositoryStarsFetchedAt = &value
	}
	if starsRefreshAfter.Valid {
		value := starsRefreshAfter.Time
		skill.RepositoryStarsRefreshAfter = &value
	}
	if currentVersionID.Valid {
		value := currentVersionID.Int64
		skill.CurrentVersionID = &value
	}
	if publishedAt.Valid {
		value := publishedAt.Time
		skill.PublishedAt = &value
	}
	if archivedAt.Valid {
		value := archivedAt.Time
		skill.ArchivedAt = &value
	}
	if createdBy.Valid {
		value := createdBy.Int64
		skill.CreatedBy = &value
	}
	if updatedBy.Valid {
		value := updatedBy.Int64
		skill.UpdatedBy = &value
	}
	return &skill, nil
}

func encodeSkillStringList(values []string) (string, error) {
	if values == nil {
		values = []string{}
	}
	encoded, err := json.Marshal(values)
	return string(encoded), err
}

func (r *skillMarketRepository) Create(ctx context.Context, skill *service.Skill) error {
	tagsJSON, err := encodeSkillStringList(skill.Tags)
	if err != nil {
		return fmt.Errorf("encode skill tags: %w", err)
	}
	promptsJSON, err := encodeSkillStringList(skill.ExamplePrompts)
	if err != nil {
		return fmt.Errorf("encode skill prompts: %w", err)
	}
	err = r.db.QueryRowContext(ctx, `
INSERT INTO skills (
  slug, display_name, summary, description, category, tags, icon,
  example_prompts, risk_notes, origin_url, source_url, source_repository, repository_stars_refresh_after,
  status, featured, sort_order, catalog_source_priority, catalog_source_rank, created_by, updated_by
) VALUES ($1,$2,$3,$4,$5,$6::jsonb,$7,$8::jsonb,$9,$10,$11,$12,
  CASE WHEN $11='' THEN NULL ELSE NOW() END,'draft',$13,$14,$15,$16,$17,$17)
RETURNING id`,
		skill.Slug, skill.DisplayName, skill.Summary, skill.Description, skill.Category,
		tagsJSON, skill.Icon, promptsJSON, skill.RiskNotes,
		skill.OriginURL, skill.SourceURL, skill.SourceRepository, skill.Featured, skill.SortOrder,
		skill.CatalogSourcePriority, skill.CatalogSourceRank, skill.CreatedBy,
	).Scan(&skill.ID)
	if err != nil {
		if isUniqueViolation(err) {
			return service.ErrSkillExists
		}
		if isSkillMarketInputViolation(err) {
			return service.ErrSkillInvalid
		}
		return fmt.Errorf("create skill: %w", err)
	}
	created, err := r.GetByID(ctx, skill.ID)
	if err != nil {
		return err
	}
	*skill = *created
	return nil
}

func (r *skillMarketRepository) Update(ctx context.Context, skill *service.Skill, previousSlug string) error {
	tagsJSON, err := encodeSkillStringList(skill.Tags)
	if err != nil {
		return fmt.Errorf("encode skill tags: %w", err)
	}
	promptsJSON, err := encodeSkillStringList(skill.ExamplePrompts)
	if err != nil {
		return fmt.Errorf("encode skill prompts: %w", err)
	}
	result, err := r.db.ExecContext(ctx, `
UPDATE skills SET
  slug=$2, display_name=$3, summary=$4, description=$5, category=$6,
  tags=$7::jsonb, icon=$8, example_prompts=$9::jsonb, risk_notes=$10,
  origin_url=$11, source_url=$12, source_repository=$13,
  repository_stars=CASE WHEN skills.source_url=$12 THEN repository_stars ELSE NULL END,
  repository_stars_fetched_at=CASE WHEN skills.source_url=$12 THEN repository_stars_fetched_at ELSE NULL END,
  repository_stars_refresh_after=CASE
    WHEN skills.source_url=$12 THEN repository_stars_refresh_after
    WHEN $12='' THEN NULL
    ELSE NOW()
  END,
  featured=$14, sort_order=$15, catalog_source_priority=$16,
  catalog_source_rank=$17, updated_by=$18, updated_at=NOW()
WHERE id=$1 AND slug=$19
  AND ($2=slug OR NOT EXISTS (SELECT 1 FROM skill_versions WHERE skill_id=skills.id))
  AND (
    status <> 'published'
    OR (
      BTRIM($4) <> '' AND BTRIM($5) <> '' AND BTRIM($6) <> ''
    )
  )`,
		skill.ID, skill.Slug, skill.DisplayName, skill.Summary, skill.Description,
		skill.Category, tagsJSON, skill.Icon, promptsJSON, skill.RiskNotes,
		skill.OriginURL, skill.SourceURL, skill.SourceRepository,
		skill.Featured, skill.SortOrder, skill.CatalogSourcePriority,
		skill.CatalogSourceRank, skill.UpdatedBy, previousSlug,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return service.ErrSkillExists
		}
		if isSkillMarketInputViolation(err) {
			return service.ErrSkillInvalid
		}
		return fmt.Errorf("update skill: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect skill update: %w", err)
	}
	if affected == 0 {
		var currentSlug, status string
		var hasVersions bool
		lookupErr := r.db.QueryRowContext(ctx, `
SELECT slug, status, EXISTS(SELECT 1 FROM skill_versions WHERE skill_id=skills.id)
FROM skills WHERE id=$1`, skill.ID).Scan(&currentSlug, &status, &hasVersions)
		if errors.Is(lookupErr, sql.ErrNoRows) {
			return service.ErrSkillNotFound
		}
		if lookupErr != nil {
			return fmt.Errorf("inspect rejected skill update: %w", lookupErr)
		}
		if currentSlug != previousSlug || (skill.Slug != currentSlug && hasVersions) {
			return service.ErrSkillSlugLocked
		}
		if status == service.SkillStatusPublished {
			return service.ErrSkillInvalid
		}
		return service.ErrSkillInvalid
	}
	updated, err := r.GetByID(ctx, skill.ID)
	if err != nil {
		return err
	}
	*skill = *updated
	return nil
}

func (r *skillMarketRepository) GetByID(ctx context.Context, id int64) (*service.Skill, error) {
	skill, err := scanSkillMarket(r.db.QueryRowContext(ctx, `SELECT `+skillMarketSelectColumns+` FROM skills s WHERE s.id=$1`, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrSkillNotFound
		}
		return nil, fmt.Errorf("get skill: %w", err)
	}
	return skill, nil
}

func (r *skillMarketRepository) GetPublishedBySlug(ctx context.Context, slug string) (*service.Skill, error) {
	skill, err := scanSkillMarket(r.db.QueryRowContext(ctx, `
SELECT `+skillMarketSelectColumns+`
FROM skills s
JOIN skill_versions current_version ON current_version.id=s.current_version_id
WHERE s.slug=$1 AND s.status='published'
  AND current_version.released_at IS NOT NULL AND current_version.yanked_at IS NULL`, slug))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrSkillNotFound
		}
		return nil, fmt.Errorf("get published skill: %w", err)
	}
	return skill, nil
}

func (r *skillMarketRepository) GetPublishedBySlugLocalized(ctx context.Context, slug, locale string) (*service.Skill, error) {
	if strings.TrimSpace(locale) == "" {
		locale = "zh-CN"
	}
	skill, err := scanSkillMarket(r.db.QueryRowContext(ctx, `
SELECT `+skillMarketLocalizedSelectColumns+`
FROM skills s
LEFT JOIN skill_catalog_localizations catalog_copy
  ON catalog_copy.slug=s.slug AND catalog_copy.locale=$2
JOIN skill_versions current_version ON current_version.id=s.current_version_id
WHERE s.slug=$1 AND s.status='published'
  AND current_version.released_at IS NOT NULL AND current_version.yanked_at IS NULL`, slug, locale))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrSkillNotFound
		}
		return nil, fmt.Errorf("get localized published skill: %w", err)
	}
	return skill, nil
}

func buildSkillMarketWhere(filter service.SkillListFilter, public, localized bool, args []any) (string, []any) {
	conditions := make([]string, 0, 5)
	add := func(condition string, value any) {
		args = append(args, value)
		conditions = append(conditions, fmt.Sprintf(condition, len(args)))
	}
	if public {
		conditions = append(conditions, "s.status='published'", "s.current_version_id IS NOT NULL")
		conditions = append(conditions, `EXISTS (
  SELECT 1 FROM skill_versions public_version
  WHERE public_version.id=s.current_version_id
    AND public_version.released_at IS NOT NULL AND public_version.yanked_at IS NULL
)`)
	} else if filter.Status != "" {
		add("s.status=$%d", filter.Status)
	}
	if filter.Search != "" {
		args = append(args, "%"+filter.Search+"%")
		n := len(args)
		searchCondition := fmt.Sprintf(
			"(s.slug ILIKE $%d OR s.display_name ILIKE $%d OR s.summary ILIKE $%d OR s.description ILIKE $%d OR s.tags::text ILIKE $%d",
			n, n, n, n, n,
		)
		if localized {
			searchCondition += fmt.Sprintf(
				" OR catalog_copy.display_name ILIKE $%d OR catalog_copy.summary ILIKE $%d OR catalog_copy.description ILIKE $%d",
				n, n, n,
			)
		}
		conditions = append(conditions, searchCondition+")")
	}
	if filter.Category != "" {
		add("s.category=$%d", filter.Category)
	}
	if filter.Featured != nil {
		add("s.featured=$%d", *filter.Featured)
	}
	if len(conditions) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

func (r *skillMarketRepository) list(ctx context.Context, filter service.SkillListFilter, public bool) ([]service.Skill, int64, error) {
	selectColumns := skillMarketSelectColumns
	fromClause := " FROM skills s"
	args := make([]any, 0, 7)
	localized := false
	if public {
		locale := strings.TrimSpace(filter.Locale)
		if locale == "" {
			locale = "zh-CN"
		}
		args = append(args, locale)
		localized = true
		selectColumns = skillMarketLocalizedSelectColumns
		fromClause += ` LEFT JOIN skill_catalog_localizations catalog_copy
  ON catalog_copy.slug=s.slug AND catalog_copy.locale=$1`
	}
	where, args := buildSkillMarketWhere(filter, public, localized, args)
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*)"+fromClause+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count skills: %w", err)
	}
	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)
	query := `SELECT ` + selectColumns + fromClause + where +
		fmt.Sprintf(" ORDER BY s.featured DESC, s.catalog_source_priority ASC, s.catalog_source_rank ASC NULLS LAST, s.sort_order ASC, s.id ASC LIMIT $%d OFFSET $%d", len(args)-1, len(args))
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list skills: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]service.Skill, 0)
	for rows.Next() {
		skill, err := scanSkillMarket(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan skill: %w", err)
		}
		items = append(items, *skill)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate skills: %w", err)
	}
	return items, total, nil
}

func (r *skillMarketRepository) ListAdmin(ctx context.Context, filter service.SkillListFilter) ([]service.Skill, int64, error) {
	return r.list(ctx, filter, false)
}

func (r *skillMarketRepository) ListPublished(ctx context.Context, filter service.SkillListFilter) ([]service.Skill, int64, error) {
	return r.list(ctx, filter, true)
}

func (r *skillMarketRepository) ListPublishedCategories(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT DISTINCT s.category
FROM skills s
JOIN skill_versions v ON v.id=s.current_version_id
WHERE s.status='published' AND s.category<>''
  AND v.released_at IS NOT NULL AND v.yanked_at IS NULL
ORDER BY s.category`)
	if err != nil {
		return nil, fmt.Errorf("list skill categories: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]string, 0)
	for rows.Next() {
		var category string
		if err := rows.Scan(&category); err != nil {
			return nil, fmt.Errorf("scan skill category: %w", err)
		}
		items = append(items, category)
	}
	return items, rows.Err()
}

const skillVersionSelectColumns = `
v.id, v.skill_id, v.version, v.changelog, v.manifest_name, v.manifest_description,
v.skill_md, v.sha256, v.byte_size, v.unpacked_size, v.file_count, v.file_manifest,
v.validation_report, v.download_count, v.released_at, v.yanked_at, v.created_by, v.created_at`

func scanSkillVersion(scanner skillMarketScanner) (*service.SkillVersion, error) {
	var (
		version                      service.SkillVersion
		manifestJSON, validationJSON []byte
		releasedAt, yankedAt         sql.NullTime
		createdBy                    sql.NullInt64
	)
	if err := scanner.Scan(
		&version.ID, &version.SkillID, &version.Version, &version.Changelog,
		&version.ManifestName, &version.ManifestDescription, &version.SkillMD,
		&version.SHA256, &version.ByteSize, &version.UnpackedSize, &version.FileCount,
		&manifestJSON, &validationJSON, &version.DownloadCount, &releasedAt,
		&yankedAt, &createdBy, &version.CreatedAt,
	); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(manifestJSON, &version.FileManifest); err != nil {
		return nil, fmt.Errorf("decode skill file manifest: %w", err)
	}
	if err := json.Unmarshal(validationJSON, &version.ValidationReport); err != nil {
		return nil, fmt.Errorf("decode skill validation report: %w", err)
	}
	if version.FileManifest == nil {
		version.FileManifest = []service.SkillArchiveFile{}
	}
	if version.ValidationReport.Errors == nil {
		version.ValidationReport.Errors = []service.SkillValidationIssue{}
	}
	if version.ValidationReport.Warnings == nil {
		version.ValidationReport.Warnings = []service.SkillValidationIssue{}
	}
	if releasedAt.Valid {
		value := releasedAt.Time
		version.ReleasedAt = &value
	}
	if yankedAt.Valid {
		value := yankedAt.Time
		version.YankedAt = &value
	}
	if createdBy.Valid {
		value := createdBy.Int64
		version.CreatedBy = &value
	}
	return &version, nil
}

func (r *skillMarketRepository) InsertVersion(ctx context.Context, version *service.SkillVersion) error {
	manifestJSON, err := json.Marshal(version.FileManifest)
	if err != nil {
		return fmt.Errorf("encode skill file manifest: %w", err)
	}
	validationJSON, err := json.Marshal(version.ValidationReport)
	if err != nil {
		return fmt.Errorf("encode skill validation report: %w", err)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin skill version insert: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var slug, status string
	if err := tx.QueryRowContext(ctx, `SELECT slug, status FROM skills WHERE id=$1 FOR UPDATE`, version.SkillID).Scan(&slug, &status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return service.ErrSkillNotFound
		}
		return fmt.Errorf("lock skill for version insert: %w", err)
	}
	if status == service.SkillStatusArchived {
		return service.ErrSkillArchived
	}
	if slug != version.ManifestName {
		return service.ErrSkillSlugLocked
	}
	err = tx.QueryRowContext(ctx, `
INSERT INTO skill_versions (
  skill_id, version, changelog, manifest_name, manifest_description, skill_md,
  package_data, sha256, byte_size, unpacked_size, file_count, file_manifest,
  validation_report, created_by
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12::jsonb,$13::jsonb,$14)
RETURNING id, created_at`,
		version.SkillID, version.Version, version.Changelog, version.ManifestName,
		version.ManifestDescription, version.SkillMD, version.PackageData, version.SHA256,
		version.ByteSize, version.UnpackedSize, version.FileCount, string(manifestJSON),
		string(validationJSON), version.CreatedBy,
	).Scan(&version.ID, &version.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return service.ErrSkillVersionExists
		}
		if isSkillMarketInputViolation(err) {
			return service.ErrSkillArchiveInvalid
		}
		return fmt.Errorf("insert skill version: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit skill version insert: %w", err)
	}
	return nil
}

func (r *skillMarketRepository) GetVersionByID(ctx context.Context, skillID, versionID int64) (*service.SkillVersion, error) {
	version, err := scanSkillVersion(r.db.QueryRowContext(ctx, `SELECT `+skillVersionSelectColumns+` FROM skill_versions v WHERE v.skill_id=$1 AND v.id=$2`, skillID, versionID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrSkillVersionNotFound
		}
		return nil, fmt.Errorf("get skill version: %w", err)
	}
	return version, nil
}

func (r *skillMarketRepository) GetVersionSummaryByID(ctx context.Context, skillID, versionID int64) (*service.SkillVersion, error) {
	version := &service.SkillVersion{}
	var releasedAt, yankedAt sql.NullTime
	err := r.db.QueryRowContext(ctx, `
SELECT id, skill_id, version, sha256, byte_size, file_count, download_count,
       released_at, yanked_at, created_at
FROM skill_versions WHERE skill_id=$1 AND id=$2`, skillID, versionID).Scan(
		&version.ID, &version.SkillID, &version.Version, &version.SHA256,
		&version.ByteSize, &version.FileCount, &version.DownloadCount,
		&releasedAt, &yankedAt, &version.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrSkillVersionNotFound
		}
		return nil, fmt.Errorf("get skill version summary: %w", err)
	}
	if releasedAt.Valid {
		value := releasedAt.Time
		version.ReleasedAt = &value
	}
	if yankedAt.Valid {
		value := yankedAt.Time
		version.YankedAt = &value
	}
	return version, nil
}

func (r *skillMarketRepository) ListVersions(ctx context.Context, skillID int64, releasedOnly bool) ([]service.SkillVersion, error) {
	where := "v.skill_id=$1"
	if releasedOnly {
		where += " AND v.released_at IS NOT NULL AND v.yanked_at IS NULL"
	}
	rows, err := r.db.QueryContext(ctx, `SELECT `+skillVersionSelectColumns+` FROM skill_versions v WHERE `+where+` ORDER BY v.created_at DESC, v.id DESC`, skillID)
	if err != nil {
		return nil, fmt.Errorf("list skill versions: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]service.SkillVersion, 0)
	for rows.Next() {
		version, err := scanSkillVersion(rows)
		if err != nil {
			return nil, fmt.Errorf("scan skill version: %w", err)
		}
		items = append(items, *version)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate skill versions: %w", err)
	}
	return items, nil
}

func (r *skillMarketRepository) Publish(ctx context.Context, skillID, versionID int64, actorID *int64) error {
	return r.activate(ctx, skillID, versionID, actorID, false)
}

func (r *skillMarketRepository) Activate(ctx context.Context, skillID, versionID int64, actorID *int64) error {
	return r.activate(ctx, skillID, versionID, actorID, true)
}

func (r *skillMarketRepository) activate(ctx context.Context, skillID, versionID int64, actorID *int64, requirePublished bool) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin skill activation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var status string
	var metadataPublishable bool
	if err := tx.QueryRowContext(ctx, `
SELECT status,
       BTRIM(summary) <> '' AND BTRIM(description) <> '' AND BTRIM(category) <> ''
FROM skills WHERE id=$1 FOR UPDATE`, skillID).Scan(&status, &metadataPublishable); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return service.ErrSkillNotFound
		}
		return fmt.Errorf("lock skill for activation: %w", err)
	}
	if status == service.SkillStatusArchived {
		return service.ErrSkillArchived
	}
	if requirePublished && status != service.SkillStatusPublished {
		return service.ErrSkillInvalid
	}
	if !metadataPublishable {
		return service.ErrSkillInvalid
	}
	var yankedAt sql.NullTime
	var valid bool
	if err := tx.QueryRowContext(ctx, `
SELECT yanked_at, COALESCE((validation_report->>'valid')::boolean, false)
FROM skill_versions WHERE id=$1 AND skill_id=$2 FOR UPDATE`, versionID, skillID).Scan(&yankedAt, &valid); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return service.ErrSkillVersionNotFound
		}
		return fmt.Errorf("lock skill version for activation: %w", err)
	}
	if yankedAt.Valid {
		return service.ErrSkillVersionYanked
	}
	if !valid {
		return service.ErrSkillArchiveInvalid
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE skill_versions SET
  released_at=COALESCE(released_at, NOW()),
  released_by=CASE WHEN released_at IS NULL THEN $3 ELSE released_by END
WHERE id=$1 AND skill_id=$2`, versionID, skillID, actorID); err != nil {
		return fmt.Errorf("release skill version: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE skills SET status='published', current_version_id=$2,
  published_at=COALESCE(published_at, NOW()), archived_at=NULL,
  updated_by=$3, updated_at=NOW()
WHERE id=$1`, skillID, versionID, actorID); err != nil {
		if isSkillMarketInputViolation(err) {
			return service.ErrSkillInvalid
		}
		return fmt.Errorf("publish skill version: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit skill activation: %w", err)
	}
	return nil
}

func (r *skillMarketRepository) Yank(ctx context.Context, skillID, versionID int64, actorID *int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin skill yank: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var currentVersionID sql.NullInt64
	if err := tx.QueryRowContext(ctx, `SELECT current_version_id FROM skills WHERE id=$1 FOR UPDATE`, skillID).Scan(&currentVersionID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return service.ErrSkillNotFound
		}
		return fmt.Errorf("lock skill for yank: %w", err)
	}
	if currentVersionID.Valid && currentVersionID.Int64 == versionID {
		return service.ErrSkillCurrentVersion
	}
	result, err := tx.ExecContext(ctx, `
UPDATE skill_versions SET yanked_at=COALESCE(yanked_at,NOW()),
  yanked_by=CASE WHEN yanked_at IS NULL THEN $3 ELSE yanked_by END
WHERE id=$1 AND skill_id=$2`, versionID, skillID, actorID)
	if err != nil {
		return fmt.Errorf("yank skill version: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect skill version yank: %w", err)
	}
	if affected == 0 {
		return service.ErrSkillVersionNotFound
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit skill yank: %w", err)
	}
	return nil
}

func (r *skillMarketRepository) Archive(ctx context.Context, skillID int64, actorID *int64) error {
	result, err := r.db.ExecContext(ctx, `
UPDATE skills SET status='archived', archived_at=COALESCE(archived_at,NOW()),
  updated_by=$2, updated_at=NOW()
WHERE id=$1`, skillID, actorID)
	if err != nil {
		return fmt.Errorf("archive skill: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect skill archive: %w", err)
	}
	if affected == 0 {
		return service.ErrSkillNotFound
	}
	return nil
}

func (r *skillMarketRepository) GetVersionArtifact(ctx context.Context, slug, version string) (*service.SkillArtifact, error) {
	artifact := &service.SkillArtifact{}
	err := r.db.QueryRowContext(ctx, `
SELECT s.id, v.id, s.slug, v.version, v.sha256, v.package_data
FROM skills s
JOIN skill_versions v ON v.skill_id=s.id
WHERE s.slug=$1 AND v.version=$2 AND s.status='published'
  AND v.released_at IS NOT NULL AND v.yanked_at IS NULL`, slug, version).Scan(
		&artifact.SkillID, &artifact.VersionID, &artifact.Slug,
		&artifact.Version, &artifact.SHA256, &artifact.Data,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrSkillDownloadNotFound
		}
		return nil, fmt.Errorf("get skill artifact: %w", err)
	}
	return artifact, nil
}

func (r *skillMarketRepository) RecordDownload(ctx context.Context, skillVersionID int64) error {
	result, err := r.db.ExecContext(ctx, `UPDATE skill_versions SET download_count=download_count+1 WHERE id=$1`, skillVersionID)
	if err != nil {
		return fmt.Errorf("record skill download: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect skill download count: %w", err)
	}
	if affected == 0 {
		return service.ErrSkillVersionNotFound
	}
	return nil
}

func (r *skillMarketRepository) ClaimRepositoryStarsRefresh(
	ctx context.Context,
	limit int,
	claimedUntil time.Time,
) ([]service.SkillRepositoryStarsTarget, error) {
	if limit < 1 {
		return []service.SkillRepositoryStarsTarget{}, nil
	}
	if limit > 20 {
		limit = 20
	}
	rows, err := r.db.QueryContext(ctx, `
WITH due AS (
  SELECT id
  FROM skills
  WHERE source_url <> ''
    AND source_repository <> ''
    AND (
      repository_stars_refresh_after IS NULL
      OR repository_stars_refresh_after <= NOW()
    )
  ORDER BY repository_stars_refresh_after ASC NULLS FIRST, id ASC
  FOR UPDATE SKIP LOCKED
  LIMIT $1
)
UPDATE skills AS skill
SET repository_stars_refresh_after = $2
FROM due
WHERE skill.id = due.id
RETURNING skill.id, skill.source_url, skill.source_repository`, limit, claimedUntil)
	if err != nil {
		return nil, fmt.Errorf("claim skill repository stars refresh: %w", err)
	}
	defer func() { _ = rows.Close() }()

	targets := make([]service.SkillRepositoryStarsTarget, 0, limit)
	for rows.Next() {
		var target service.SkillRepositoryStarsTarget
		if err := rows.Scan(&target.ID, &target.SourceURL, &target.SourceRepository); err != nil {
			return nil, fmt.Errorf("scan skill repository stars refresh: %w", err)
		}
		targets = append(targets, target)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate skill repository stars refresh: %w", err)
	}
	return targets, nil
}

func (r *skillMarketRepository) CompleteRepositoryStarsRefresh(
	ctx context.Context,
	skillID int64,
	sourceURL string,
	stars int64,
	fetchedAt time.Time,
	refreshAfter time.Time,
) error {
	if stars < 0 {
		return service.ErrSkillInvalid
	}
	if _, err := r.db.ExecContext(ctx, `
UPDATE skills
SET repository_stars = $3,
    repository_stars_fetched_at = $4,
    repository_stars_refresh_after = $5
WHERE id = $1 AND source_url = $2`,
		skillID, sourceURL, stars, fetchedAt, refreshAfter,
	); err != nil {
		return fmt.Errorf("complete skill repository stars refresh: %w", err)
	}
	return nil
}

func (r *skillMarketRepository) DeferRepositoryStarsRefresh(
	ctx context.Context,
	skillID int64,
	sourceURL string,
	refreshAfter time.Time,
) error {
	if _, err := r.db.ExecContext(ctx, `
UPDATE skills
SET repository_stars_refresh_after = $3
WHERE id = $1 AND source_url = $2`, skillID, sourceURL, refreshAfter); err != nil {
		return fmt.Errorf("defer skill repository stars refresh: %w", err)
	}
	return nil
}

func isSkillMarketInputViolation(err error) bool {
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
