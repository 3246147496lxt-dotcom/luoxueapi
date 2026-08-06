package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"golang.org/x/mod/semver"
)

const (
	SkillStatusDraft     = "draft"
	SkillStatusPublished = "published"
	SkillStatusArchived  = "archived"

	SkillVersionStatusAvailable = "available"
	SkillVersionStatusActive    = "active"
	SkillVersionStatusYanked    = "yanked"
)

var (
	ErrSkillNotFound         = infraerrors.NotFound("SKILL_NOT_FOUND", "skill not found")
	ErrSkillExists           = infraerrors.Conflict("SKILL_EXISTS", "skill slug already exists")
	ErrSkillInvalid          = infraerrors.BadRequest("SKILL_INVALID", "skill metadata is invalid")
	ErrSkillArchived         = infraerrors.Conflict("SKILL_ARCHIVED", "archived skill cannot be published")
	ErrSkillSlugLocked       = infraerrors.Conflict("SKILL_SLUG_LOCKED", "skill slug is locked after a version is uploaded")
	ErrSkillVersionNotFound  = infraerrors.NotFound("SKILL_VERSION_NOT_FOUND", "skill version not found")
	ErrSkillVersionExists    = infraerrors.Conflict("SKILL_VERSION_EXISTS", "skill version already exists")
	ErrSkillVersionInvalid   = infraerrors.BadRequest("SKILL_VERSION_INVALID", "skill version must be valid SemVer")
	ErrSkillVersionYanked    = infraerrors.Conflict("SKILL_VERSION_YANKED", "yanked skill version cannot be activated")
	ErrSkillCurrentVersion   = infraerrors.Conflict("SKILL_CURRENT_VERSION", "activate another version before yanking the current version")
	ErrSkillArchiveInvalid   = infraerrors.New(http.StatusUnprocessableEntity, "SKILL_ARCHIVE_INVALID", "skill archive failed validation")
	ErrSkillDownloadNotFound = infraerrors.NotFound("SKILL_DOWNLOAD_NOT_FOUND", "skill package not found")
)

type SkillValidationIssue struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Path    string `json:"path,omitempty"`
}

type SkillValidationReport struct {
	Valid    bool                   `json:"valid"`
	Errors   []SkillValidationIssue `json:"errors"`
	Warnings []SkillValidationIssue `json:"warnings"`
}

type SkillArchiveValidationError struct {
	Report SkillValidationReport
	cause  error
}

func (e *SkillArchiveValidationError) Error() string { return e.cause.Error() }
func (e *SkillArchiveValidationError) Unwrap() error { return e.cause }

type SkillArchiveFile struct {
	Path     string `json:"path"`
	ByteSize int64  `json:"byte_size"`
	SHA256   string `json:"sha256"`
}

type SkillVersion struct {
	ID                  int64                 `json:"id"`
	SkillID             int64                 `json:"skill_id"`
	Version             string                `json:"version"`
	Status              string                `json:"status"`
	Changelog           string                `json:"changelog"`
	ManifestName        string                `json:"manifest_name"`
	ManifestDescription string                `json:"manifest_description"`
	SkillMD             string                `json:"skill_md"`
	SHA256              string                `json:"sha256"`
	ByteSize            int64                 `json:"byte_size"`
	UnpackedSize        int64                 `json:"unpacked_size"`
	FileCount           int                   `json:"file_count"`
	FileManifest        []SkillArchiveFile    `json:"file_manifest"`
	ValidationReport    SkillValidationReport `json:"validation_report"`
	DownloadCount       int64                 `json:"download_count"`
	ReleasedAt          *time.Time            `json:"released_at"`
	YankedAt            *time.Time            `json:"yanked_at"`
	CreatedAt           time.Time             `json:"created_at"`
	CreatedBy           *int64                `json:"-"`
	PackageData         []byte                `json:"-"`
}

type Skill struct {
	ID                          int64          `json:"id"`
	Slug                        string         `json:"slug"`
	DisplayName                 string         `json:"display_name"`
	Summary                     string         `json:"summary"`
	Description                 string         `json:"description"`
	Category                    string         `json:"category"`
	Tags                        []string       `json:"tags"`
	Icon                        string         `json:"icon"`
	ExamplePrompts              []string       `json:"example_prompts"`
	RiskNotes                   string         `json:"risk_notes"`
	SourceURL                   string         `json:"source_url"`
	SourceRepository            string         `json:"source_repository"`
	RepositoryStars             *int64         `json:"repository_stars"`
	Status                      string         `json:"status"`
	Featured                    bool           `json:"featured"`
	SortOrder                   int            `json:"sort_order"`
	CurrentVersionID            *int64         `json:"current_version_id"`
	CurrentVersion              *SkillVersion  `json:"current_version"`
	LatestVersion               *SkillVersion  `json:"latest_version"`
	Versions                    []SkillVersion `json:"versions,omitempty"`
	DownloadCount               int64          `json:"download_count"`
	PublishedAt                 *time.Time     `json:"published_at"`
	ArchivedAt                  *time.Time     `json:"archived_at"`
	CreatedAt                   time.Time      `json:"created_at"`
	UpdatedAt                   time.Time      `json:"updated_at"`
	CreatedBy                   *int64         `json:"-"`
	UpdatedBy                   *int64         `json:"-"`
	RepositoryStarsFetchedAt    *time.Time     `json:"-"`
	RepositoryStarsRefreshAfter *time.Time     `json:"-"`
}

type SkillInput struct {
	Slug           string   `json:"slug"`
	DisplayName    string   `json:"display_name"`
	Summary        string   `json:"summary"`
	Description    string   `json:"description"`
	Category       string   `json:"category"`
	Tags           []string `json:"tags"`
	Icon           string   `json:"icon"`
	ExamplePrompts []string `json:"example_prompts"`
	RiskNotes      string   `json:"risk_notes"`
	SourceURL      *string  `json:"source_url"`
	Featured       bool     `json:"featured"`
	SortOrder      int      `json:"sort_order"`
}

type SkillListFilter struct {
	Search   string
	Category string
	Status   string
	Featured *bool
	Page     int
	PageSize int
}

type SkillListResult struct {
	Items      []Skill  `json:"items"`
	Total      int64    `json:"total"`
	Page       int      `json:"page"`
	PageSize   int      `json:"page_size"`
	Pages      int      `json:"pages"`
	Categories []string `json:"categories,omitempty"`
}

type PublicSkillVersion struct {
	Version             string                `json:"version"`
	Changelog           string                `json:"changelog"`
	ManifestName        string                `json:"manifest_name"`
	ManifestDescription string                `json:"manifest_description"`
	SkillMD             string                `json:"skill_md"`
	SHA256              string                `json:"sha256"`
	ByteSize            int64                 `json:"byte_size"`
	UnpackedSize        int64                 `json:"unpacked_size"`
	FileCount           int                   `json:"file_count"`
	FileManifest        []SkillArchiveFile    `json:"file_manifest"`
	ValidationReport    SkillValidationReport `json:"validation_report"`
	DownloadCount       int64                 `json:"download_count"`
	ReleasedAt          *time.Time            `json:"released_at"`
}

type PublicSkillVersionSummary struct {
	Version       string     `json:"version"`
	SHA256        string     `json:"sha256"`
	ByteSize      int64      `json:"byte_size"`
	FileCount     int        `json:"file_count"`
	DownloadCount int64      `json:"download_count"`
	ReleasedAt    *time.Time `json:"released_at"`
}

type PublicSkill struct {
	Slug             string                     `json:"slug"`
	DisplayName      string                     `json:"display_name"`
	Summary          string                     `json:"summary"`
	Description      string                     `json:"description"`
	Category         string                     `json:"category"`
	Tags             []string                   `json:"tags"`
	Icon             string                     `json:"icon"`
	ExamplePrompts   []string                   `json:"example_prompts"`
	RiskNotes        string                     `json:"risk_notes"`
	SourceURL        string                     `json:"source_url"`
	SourceRepository string                     `json:"source_repository"`
	RepositoryStars  *int64                     `json:"repository_stars"`
	Featured         bool                       `json:"featured"`
	CurrentVersion   *PublicSkillVersionSummary `json:"current_version"`
	Versions         []PublicSkillVersion       `json:"versions,omitempty"`
	DownloadCount    int64                      `json:"download_count"`
	PublishedAt      *time.Time                 `json:"published_at"`
	UpdatedAt        time.Time                  `json:"updated_at"`
}

type PublicSkillListResult struct {
	Items      []PublicSkill `json:"items"`
	Total      int64         `json:"total"`
	Page       int           `json:"page"`
	PageSize   int           `json:"page_size"`
	Pages      int           `json:"pages"`
	Categories []string      `json:"categories"`
}

type SkillArtifact struct {
	SkillID   int64
	VersionID int64
	Slug      string
	Version   string
	SHA256    string
	Data      []byte
}

type SkillRepositoryStarsTarget struct {
	ID               int64
	SourceURL        string
	SourceRepository string
}

type SkillMarketRepository interface {
	Create(ctx context.Context, skill *Skill) error
	Update(ctx context.Context, skill *Skill, previousSlug string) error
	GetByID(ctx context.Context, id int64) (*Skill, error)
	GetPublishedBySlug(ctx context.Context, slug string) (*Skill, error)
	ListAdmin(ctx context.Context, filter SkillListFilter) ([]Skill, int64, error)
	ListPublished(ctx context.Context, filter SkillListFilter) ([]Skill, int64, error)
	ListPublishedCategories(ctx context.Context) ([]string, error)
	InsertVersion(ctx context.Context, version *SkillVersion) error
	GetVersionByID(ctx context.Context, skillID, versionID int64) (*SkillVersion, error)
	GetVersionSummaryByID(ctx context.Context, skillID, versionID int64) (*SkillVersion, error)
	ListVersions(ctx context.Context, skillID int64, releasedOnly bool) ([]SkillVersion, error)
	Publish(ctx context.Context, skillID, versionID int64, actorID *int64) error
	Activate(ctx context.Context, skillID, versionID int64, actorID *int64) error
	Yank(ctx context.Context, skillID, versionID int64, actorID *int64) error
	Archive(ctx context.Context, skillID int64, actorID *int64) error
	GetVersionArtifact(ctx context.Context, slug, version string) (*SkillArtifact, error)
	RecordDownload(ctx context.Context, skillVersionID int64) error
	ClaimRepositoryStarsRefresh(ctx context.Context, limit int, claimedUntil time.Time) ([]SkillRepositoryStarsTarget, error)
	CompleteRepositoryStarsRefresh(ctx context.Context, skillID int64, sourceURL string, stars int64, fetchedAt, refreshAfter time.Time) error
	DeferRepositoryStarsRefresh(ctx context.Context, skillID int64, sourceURL string, refreshAfter time.Time) error
}

type SkillMarketplaceConfig struct {
	Enabled bool `json:"enabled"`
}

type SkillMarketService struct {
	repo           SkillMarketRepository
	settingRepo    SettingRepository
	settingService *SettingService
	gateMu         sync.Mutex
	gateEnabled    bool
	gateExpires    time.Time
	stars          skillRepositoryStarsRuntime
}

func NewSkillMarketService(repo SkillMarketRepository, settingService *SettingService) *SkillMarketService {
	var settingRepo SettingRepository
	if settingService != nil {
		settingRepo = settingService.settingRepo
	}
	return &SkillMarketService{
		repo:           repo,
		settingRepo:    settingRepo,
		settingService: settingService,
		stars:          newSkillRepositoryStarsRuntime(),
	}
}

func (s *SkillMarketService) GetConfig(ctx context.Context) (*SkillMarketplaceConfig, error) {
	if s.settingRepo == nil {
		return &SkillMarketplaceConfig{Enabled: false}, nil
	}
	value, err := s.settingRepo.GetValue(ctx, SettingKeySkillMarketplaceEnabled)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return &SkillMarketplaceConfig{Enabled: false}, nil
		}
		return nil, fmt.Errorf("get skill marketplace config: %w", err)
	}
	return &SkillMarketplaceConfig{Enabled: strings.TrimSpace(value) == "true"}, nil
}

func (s *SkillMarketService) UpdateConfig(ctx context.Context, enabled bool) (*SkillMarketplaceConfig, error) {
	// Expire the hot-path gate before the synchronous settings notification so
	// requests cannot keep consuming the old five-second snapshot while the
	// current instance invalidates its derived runtime caches.
	s.gateMu.Lock()
	s.gateExpires = time.Time{}
	s.gateMu.Unlock()

	if s.settingService == nil {
		return nil, infraerrors.InternalServer("SKILL_CONFIG_UNAVAILABLE", "skill marketplace config storage is unavailable")
	}
	if err := s.settingService.UpdateSkillMarketplaceEnabled(ctx, enabled); err != nil {
		return nil, err
	}
	return &SkillMarketplaceConfig{Enabled: enabled}, nil
}

func (s *SkillMarketService) IsPublicEnabled(ctx context.Context) bool {
	if s.settingRepo == nil {
		return false
	}
	s.gateMu.Lock()
	defer s.gateMu.Unlock()
	if time.Now().Before(s.gateExpires) {
		return s.gateEnabled
	}
	values, err := s.settingRepo.GetMultiple(ctx, []string{
		SettingKeySkillMarketplaceEnabled,
		SettingKeyBackendModeEnabled,
	})
	enabled := false
	if err != nil {
		s.gateEnabled = false
		s.gateExpires = time.Now().Add(5 * time.Second)
		return false
	}
	enabled = strings.TrimSpace(values[SettingKeySkillMarketplaceEnabled]) == "true" &&
		strings.TrimSpace(values[SettingKeyBackendModeEnabled]) != "true"
	s.gateEnabled = enabled
	s.gateExpires = time.Now().Add(5 * time.Second)
	return enabled
}

var skillSlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func normalizeSkillInput(input SkillInput) (SkillInput, error) {
	input.Slug = strings.ToLower(strings.TrimSpace(input.Slug))
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.Summary = strings.TrimSpace(input.Summary)
	input.Description = strings.TrimSpace(input.Description)
	input.Category = strings.ToLower(strings.TrimSpace(input.Category))
	input.Icon = strings.TrimSpace(input.Icon)
	input.RiskNotes = strings.TrimSpace(input.RiskNotes)
	if input.SourceURL != nil {
		sourceURL := strings.TrimSpace(*input.SourceURL)
		normalizedSourceURL, _, err := normalizeSkillSourceURL(sourceURL)
		if err != nil {
			return input, ErrSkillInvalid.WithMetadata(map[string]string{"source_url": "must be a supported github.com repository URL"})
		}
		input.SourceURL = &normalizedSourceURL
	}
	if !skillSlugPattern.MatchString(input.Slug) || len(input.Slug) > 64 || input.DisplayName == "" ||
		utf8.RuneCountInString(input.DisplayName) > 120 || utf8.RuneCountInString(input.Summary) > 280 ||
		utf8.RuneCountInString(input.Description) > 100_000 || utf8.RuneCountInString(input.Category) > 80 ||
		utf8.RuneCountInString(input.Icon) > 160 || utf8.RuneCountInString(input.RiskNotes) > 10_000 {
		return input, ErrSkillInvalid
	}
	input.Tags = normalizeSkillStrings(input.Tags, 20, 40, true)
	input.ExamplePrompts = normalizeSkillStrings(input.ExamplePrompts, 10, 500, false)
	if input.Tags == nil || input.ExamplePrompts == nil {
		return input, ErrSkillInvalid
	}
	return input, nil
}

func normalizeSkillStrings(values []string, maxItems, maxRunes int, lower bool) []string {
	if len(values) > maxItems {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if lower {
			value = strings.ToLower(value)
		}
		if value == "" || utf8.RuneCountInString(value) > maxRunes {
			return nil
		}
		key := strings.ToLower(value)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	if lower {
		sort.Strings(out)
	}
	return out
}

func skillFromInput(input SkillInput, actorID *int64) *Skill {
	sourceURL := ""
	if input.SourceURL != nil {
		sourceURL = *input.SourceURL
	}
	_, sourceRepository, _ := normalizeSkillSourceURL(sourceURL)
	return &Skill{
		Slug: input.Slug, DisplayName: input.DisplayName, Summary: input.Summary,
		Description: input.Description, Category: input.Category, Tags: input.Tags,
		Icon: input.Icon, ExamplePrompts: input.ExamplePrompts, RiskNotes: input.RiskNotes,
		SourceURL: sourceURL, SourceRepository: sourceRepository,
		Status: SkillStatusDraft, Featured: input.Featured, SortOrder: input.SortOrder,
		CreatedBy: actorID, UpdatedBy: actorID,
	}
}

func applySkillInput(skill *Skill, input SkillInput, actorID *int64) {
	skill.Slug, skill.DisplayName = input.Slug, input.DisplayName
	skill.Summary, skill.Description, skill.Category = input.Summary, input.Description, input.Category
	skill.Tags, skill.Icon, skill.ExamplePrompts = input.Tags, input.Icon, input.ExamplePrompts
	skill.RiskNotes, skill.Featured, skill.SortOrder = input.RiskNotes, input.Featured, input.SortOrder
	if input.SourceURL != nil {
		sourceURL := *input.SourceURL
		_, sourceRepository, _ := normalizeSkillSourceURL(sourceURL)
		sourceChanged := skill.SourceURL != sourceURL
		skill.SourceURL, skill.SourceRepository = sourceURL, sourceRepository
		if sourceChanged {
			skill.RepositoryStars = nil
			skill.RepositoryStarsFetchedAt = nil
			if sourceURL == "" {
				skill.RepositoryStarsRefreshAfter = nil
			} else {
				now := time.Now().UTC()
				skill.RepositoryStarsRefreshAfter = &now
			}
		}
	}
	skill.UpdatedBy = actorID
}

func (s *SkillMarketService) Create(ctx context.Context, input SkillInput, actorID *int64) (*Skill, error) {
	normalized, err := normalizeSkillInput(input)
	if err != nil {
		return nil, err
	}
	skill := skillFromInput(normalized, actorID)
	if err := s.repo.Create(ctx, skill); err != nil {
		return nil, err
	}
	skill.Versions = []SkillVersion{}
	return skill, nil
}

func (s *SkillMarketService) Update(ctx context.Context, id int64, input SkillInput, actorID *int64) (*Skill, error) {
	normalized, err := normalizeSkillInput(input)
	if err != nil {
		return nil, err
	}
	skill, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if skill.Slug != normalized.Slug {
		versions, listErr := s.repo.ListVersions(ctx, id, false)
		if listErr != nil {
			return nil, listErr
		}
		if len(versions) > 0 || skill.Status != SkillStatusDraft {
			return nil, ErrSkillSlugLocked
		}
	}
	previousSlug := skill.Slug
	applySkillInput(skill, normalized, actorID)
	if skill.Status == SkillStatusPublished {
		if skill.CurrentVersionID == nil {
			return nil, ErrSkillInvalid.WithMetadata(map[string]string{"current_version": "required"})
		}
		current, currentErr := s.repo.GetVersionByID(ctx, skill.ID, *skill.CurrentVersionID)
		if currentErr != nil {
			return nil, currentErr
		}
		if err := validateSkillPublishable(skill, current); err != nil {
			return nil, err
		}
	}
	if err := s.repo.Update(ctx, skill, previousSlug); err != nil {
		return nil, err
	}
	return s.GetAdmin(ctx, id)
}

func normalizeSkillListFilter(filter SkillListFilter, public bool) (SkillListFilter, error) {
	filter.Search = strings.TrimSpace(filter.Search)
	filter.Category = strings.ToLower(strings.TrimSpace(filter.Category))
	filter.Status = strings.ToLower(strings.TrimSpace(filter.Status))
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
	if public {
		filter.Status = SkillStatusPublished
	} else if filter.Status != "" && filter.Status != SkillStatusDraft && filter.Status != SkillStatusPublished && filter.Status != SkillStatusArchived {
		return filter, ErrSkillInvalid
	}
	return filter, nil
}

func (s *SkillMarketService) ListAdmin(ctx context.Context, filter SkillListFilter) (*SkillListResult, error) {
	filter, err := normalizeSkillListFilter(filter, false)
	if err != nil {
		return nil, err
	}
	items, total, err := s.repo.ListAdmin(ctx, filter)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if err := s.hydrateAdminVersionSummary(ctx, &items[i]); err != nil {
			return nil, err
		}
	}
	return &SkillListResult{Items: items, Total: total, Page: filter.Page, PageSize: filter.PageSize, Pages: pageCount(total, filter.PageSize)}, nil
}

func (s *SkillMarketService) GetAdmin(ctx context.Context, id int64) (*Skill, error) {
	skill, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	versions, err := s.repo.ListVersions(ctx, skill.ID, false)
	if err != nil {
		return nil, err
	}
	for i := range versions {
		setSkillVersionStatus(&versions[i], skill.CurrentVersionID)
	}
	skill.Versions = versions
	setCurrentVersionFromList(skill)
	setLatestVersionFromList(skill)
	return skill, nil
}

func normalizeSkillVersion(version string) (string, error) {
	version = strings.TrimSpace(version)
	version = strings.TrimPrefix(version, "v")
	if version == "" || len(version) > 64 || !semver.IsValid("v"+version) {
		return "", ErrSkillVersionInvalid
	}
	return version, nil
}

func (s *SkillMarketService) UploadVersion(ctx context.Context, skillID int64, version, changelog string, archive []byte, actorID *int64) (*SkillVersion, error) {
	skill, err := s.repo.GetByID(ctx, skillID)
	if err != nil {
		return nil, err
	}
	if skill.Status == SkillStatusArchived {
		return nil, ErrSkillArchived
	}
	version, err = normalizeSkillVersion(version)
	if err != nil {
		return nil, err
	}
	if utf8.RuneCountInString(changelog) > 10_000 {
		return nil, ErrSkillVersionInvalid
	}
	validated, err := ValidateSkillArchive(archive, skill.Slug)
	if err != nil {
		return nil, err
	}
	item := &SkillVersion{
		SkillID: skillID, Version: version, Status: SkillVersionStatusAvailable,
		Changelog: strings.TrimSpace(changelog), ManifestName: validated.ManifestName,
		ManifestDescription: validated.ManifestDescription, SkillMD: validated.SkillMD,
		SHA256: validated.SHA256, ByteSize: int64(len(validated.PackageData)),
		UnpackedSize: validated.UnpackedSize, FileCount: len(validated.FileManifest),
		FileManifest: validated.FileManifest, ValidationReport: validated.ValidationReport,
		PackageData: validated.PackageData, CreatedBy: actorID,
	}
	if err := s.repo.InsertVersion(ctx, item); err != nil {
		return nil, err
	}
	item.PackageData = nil
	return item, nil
}

func (s *SkillMarketService) Publish(ctx context.Context, skillID, versionID int64, actorID *int64) (*Skill, error) {
	skill, err := s.repo.GetByID(ctx, skillID)
	if err != nil {
		return nil, err
	}
	if skill.Status == SkillStatusArchived {
		return nil, ErrSkillArchived
	}
	version, err := s.repo.GetVersionByID(ctx, skillID, versionID)
	if err != nil {
		return nil, err
	}
	if version.YankedAt != nil {
		return nil, ErrSkillVersionYanked
	}
	if err := validateSkillPublishable(skill, version); err != nil {
		return nil, err
	}
	if err := s.repo.Publish(ctx, skillID, versionID, actorID); err != nil {
		return nil, err
	}
	return s.GetAdmin(ctx, skillID)
}

func (s *SkillMarketService) Activate(ctx context.Context, skillID, versionID int64, actorID *int64) (*Skill, error) {
	skill, err := s.repo.GetByID(ctx, skillID)
	if err != nil {
		return nil, err
	}
	if skill.Status == SkillStatusArchived {
		return nil, ErrSkillArchived
	}
	if skill.Status != SkillStatusPublished {
		return nil, ErrSkillInvalid.WithMetadata(map[string]string{"status": "publish the skill before activating another version"})
	}
	version, err := s.repo.GetVersionByID(ctx, skillID, versionID)
	if err != nil {
		return nil, err
	}
	if version.YankedAt != nil {
		return nil, ErrSkillVersionYanked
	}
	if err := validateSkillPublishable(skill, version); err != nil {
		return nil, err
	}
	if err := s.repo.Activate(ctx, skillID, versionID, actorID); err != nil {
		return nil, err
	}
	return s.GetAdmin(ctx, skillID)
}

func (s *SkillMarketService) Yank(ctx context.Context, skillID, versionID int64, actorID *int64) (*Skill, error) {
	skill, err := s.repo.GetByID(ctx, skillID)
	if err != nil {
		return nil, err
	}
	if skill.CurrentVersionID != nil && *skill.CurrentVersionID == versionID {
		return nil, ErrSkillCurrentVersion
	}
	if _, err := s.repo.GetVersionByID(ctx, skillID, versionID); err != nil {
		return nil, err
	}
	if err := s.repo.Yank(ctx, skillID, versionID, actorID); err != nil {
		return nil, err
	}
	return s.GetAdmin(ctx, skillID)
}

func (s *SkillMarketService) Archive(ctx context.Context, skillID int64, actorID *int64) (*Skill, error) {
	if _, err := s.repo.GetByID(ctx, skillID); err != nil {
		return nil, err
	}
	if err := s.repo.Archive(ctx, skillID, actorID); err != nil {
		return nil, err
	}
	return s.GetAdmin(ctx, skillID)
}

func (s *SkillMarketService) ListPublic(ctx context.Context, filter SkillListFilter) (*PublicSkillListResult, error) {
	filter, err := normalizeSkillListFilter(filter, true)
	if err != nil {
		return nil, err
	}
	items, total, err := s.repo.ListPublished(ctx, filter)
	if err != nil {
		return nil, err
	}
	out := make([]PublicSkill, 0, len(items))
	for i := range items {
		if err := s.hydratePublicCurrent(ctx, &items[i]); err != nil {
			return nil, err
		}
		out = append(out, publicSkillFromModel(&items[i], false))
	}
	categories, err := s.repo.ListPublishedCategories(ctx)
	if err != nil {
		return nil, err
	}
	return &PublicSkillListResult{Items: out, Total: total, Page: filter.Page, PageSize: filter.PageSize, Pages: pageCount(total, filter.PageSize), Categories: categories}, nil
}

func (s *SkillMarketService) GetPublic(ctx context.Context, slug string) (*PublicSkill, error) {
	slug = strings.ToLower(strings.TrimSpace(slug))
	skill, err := s.repo.GetPublishedBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	versions, err := s.repo.ListVersions(ctx, skill.ID, true)
	if err != nil {
		return nil, err
	}
	for i := range versions {
		setSkillVersionStatus(&versions[i], skill.CurrentVersionID)
	}
	skill.Versions = versions
	setCurrentVersionFromList(skill)
	result := publicSkillFromModel(skill, true)
	return &result, nil
}

func (s *SkillMarketService) ListPublicVersions(ctx context.Context, slug string) ([]PublicSkillVersion, error) {
	skill, err := s.repo.GetPublishedBySlug(ctx, strings.ToLower(strings.TrimSpace(slug)))
	if err != nil {
		return nil, err
	}
	versions, err := s.repo.ListVersions(ctx, skill.ID, true)
	if err != nil {
		return nil, err
	}
	out := make([]PublicSkillVersion, 0, len(versions))
	for i := range versions {
		out = append(out, publicSkillVersionFromModel(&versions[i]))
	}
	return out, nil
}

func (s *SkillMarketService) DownloadVersion(ctx context.Context, slug, version string) (*SkillArtifact, error) {
	version, err := normalizeSkillVersion(version)
	if err != nil {
		return nil, ErrSkillDownloadNotFound
	}
	artifact, err := s.repo.GetVersionArtifact(ctx, strings.ToLower(strings.TrimSpace(slug)), version)
	if err != nil {
		return nil, err
	}
	return artifact, nil
}

func (s *SkillMarketService) RecordDownloadBestEffort(ctx context.Context, artifact *SkillArtifact) {
	if artifact == nil {
		return
	}
	if err := s.repo.RecordDownload(ctx, artifact.VersionID); err != nil {
		// Analytics must never make an otherwise valid download unavailable.
		slog.Warn("record skill download failed", "skill_id", artifact.SkillID, "version_id", artifact.VersionID, "err", err)
	}
}

func (s *SkillMarketService) hydratePublicCurrent(ctx context.Context, skill *Skill) error {
	if skill.CurrentVersionID == nil {
		skill.CurrentVersion = nil
		return nil
	}
	version, err := s.repo.GetVersionSummaryByID(ctx, skill.ID, *skill.CurrentVersionID)
	if err != nil {
		return fmt.Errorf("load public skill current version summary: %w", err)
	}
	setSkillVersionStatus(version, skill.CurrentVersionID)
	skill.CurrentVersion = version
	return nil
}

func (s *SkillMarketService) hydrateAdminVersionSummary(ctx context.Context, skill *Skill) error {
	versions, err := s.repo.ListVersions(ctx, skill.ID, false)
	if err != nil {
		return err
	}
	for i := range versions {
		setSkillVersionStatus(&versions[i], skill.CurrentVersionID)
	}
	if skill.CurrentVersionID != nil {
		for i := range versions {
			if versions[i].ID == *skill.CurrentVersionID {
				copy := versions[i]
				skill.CurrentVersion = &copy
				break
			}
		}
	}
	for i := range versions {
		if versions[i].YankedAt == nil {
			copy := versions[i]
			skill.LatestVersion = &copy
			break
		}
	}
	return nil
}

func setCurrentVersionFromList(skill *Skill) {
	skill.CurrentVersion = nil
	if skill.CurrentVersionID == nil {
		return
	}
	for i := range skill.Versions {
		if skill.Versions[i].ID == *skill.CurrentVersionID {
			copy := skill.Versions[i]
			skill.CurrentVersion = &copy
			return
		}
	}
}

func setLatestVersionFromList(skill *Skill) {
	skill.LatestVersion = nil
	for i := range skill.Versions {
		if skill.Versions[i].YankedAt == nil {
			copy := skill.Versions[i]
			skill.LatestVersion = &copy
			return
		}
	}
}

func setSkillVersionStatus(version *SkillVersion, currentVersionID *int64) {
	switch {
	case version.YankedAt != nil:
		version.Status = SkillVersionStatusYanked
	case currentVersionID != nil && version.ID == *currentVersionID:
		version.Status = SkillVersionStatusActive
	default:
		version.Status = SkillVersionStatusAvailable
	}
}

func publicSkillFromModel(skill *Skill, includeVersions bool) PublicSkill {
	result := PublicSkill{
		Slug: skill.Slug, DisplayName: skill.DisplayName, Summary: skill.Summary,
		Description: skill.Description, Category: skill.Category, Tags: skill.Tags,
		Icon: skill.Icon, ExamplePrompts: skill.ExamplePrompts, RiskNotes: skill.RiskNotes,
		SourceURL: skill.SourceURL, SourceRepository: skill.SourceRepository,
		RepositoryStars: skill.RepositoryStars,
		Featured:        skill.Featured, DownloadCount: skill.DownloadCount,
		PublishedAt: skill.PublishedAt, UpdatedAt: skill.UpdatedAt,
	}
	if skill.CurrentVersion != nil {
		current := PublicSkillVersionSummary{
			Version: skill.CurrentVersion.Version, SHA256: skill.CurrentVersion.SHA256,
			ByteSize: skill.CurrentVersion.ByteSize, FileCount: skill.CurrentVersion.FileCount,
			DownloadCount: skill.CurrentVersion.DownloadCount, ReleasedAt: skill.CurrentVersion.ReleasedAt,
		}
		result.CurrentVersion = &current
	}
	if includeVersions {
		result.Versions = make([]PublicSkillVersion, 0, len(skill.Versions))
		for i := range skill.Versions {
			result.Versions = append(result.Versions, publicSkillVersionFromModel(&skill.Versions[i]))
		}
	}
	return result
}

func publicSkillVersionFromModel(version *SkillVersion) PublicSkillVersion {
	return PublicSkillVersion{
		Version: version.Version, Changelog: version.Changelog,
		ManifestName: version.ManifestName, ManifestDescription: version.ManifestDescription,
		SkillMD: version.SkillMD, SHA256: version.SHA256, ByteSize: version.ByteSize,
		UnpackedSize: version.UnpackedSize, FileCount: version.FileCount,
		FileManifest: version.FileManifest, ValidationReport: version.ValidationReport,
		DownloadCount: version.DownloadCount, ReleasedAt: version.ReleasedAt,
	}
}

func pageCount(total int64, pageSize int) int {
	if pageSize <= 0 || total <= 0 {
		return 1
	}
	return int((total + int64(pageSize) - 1) / int64(pageSize))
}

func validateSkillPublishable(skill *Skill, version *SkillVersion) error {
	missing := make([]string, 0, 3)
	if strings.TrimSpace(skill.Summary) == "" {
		missing = append(missing, "summary")
	}
	if strings.TrimSpace(skill.Description) == "" {
		missing = append(missing, "description")
	}
	if strings.TrimSpace(skill.Category) == "" {
		missing = append(missing, "category")
	}
	if len(missing) > 0 {
		return ErrSkillInvalid.WithMetadata(map[string]string{"missing_fields": strings.Join(missing, ",")})
	}
	if version == nil || !version.ValidationReport.Valid {
		return ErrSkillArchiveInvalid.WithMetadata(map[string]string{"validation_code": "VERSION_NOT_VALID"})
	}
	return nil
}

func skillArchiveError(report SkillValidationReport) error {
	metadata := map[string]string{}
	if len(report.Errors) > 0 {
		metadata["validation_code"] = report.Errors[0].Code
		if report.Errors[0].Path != "" {
			metadata["path"] = report.Errors[0].Path
		}
	}
	if encoded, err := json.Marshal(report); err == nil {
		metadata["validation_report"] = string(encoded)
	}
	return &SkillArchiveValidationError{Report: report, cause: ErrSkillArchiveInvalid.WithMetadata(metadata)}
}
