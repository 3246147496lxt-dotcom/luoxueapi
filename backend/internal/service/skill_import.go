package service

import (
	"context"
	"encoding/json"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	SkillImportTriggerManual    = "manual"
	SkillImportTriggerScheduled = "scheduled"
	SkillImportTriggerRetry     = "retry"
	SkillImportTriggerBootstrap = "bootstrap"

	SkillImportModeReview      = "review"
	SkillImportModeAutoPublish = "auto_publish"
	SkillImportModeDryRun      = "dry_run"

	SkillImportRunStatusQueued           = "queued"
	SkillImportRunStatusDiscovering      = "discovering"
	SkillImportRunStatusPreparing        = "preparing"
	SkillImportRunStatusWaitingRetry     = "waiting_retry"
	SkillImportRunStatusReady            = "ready"
	SkillImportRunStatusAwaitingReview   = "awaiting_review"
	SkillImportRunStatusPublishing       = "publishing"
	SkillImportRunStatusSucceeded        = "succeeded"
	SkillImportRunStatusPartialSucceeded = "partial_succeeded"
	SkillImportRunStatusFailed           = "failed"
	SkillImportRunStatusCancelled        = "cancelled"

	// Backward-compatible Go names; persisted/API values use the canonical
	// contract above.
	SkillImportRunStatusPartial  = SkillImportRunStatusPartialSucceeded
	SkillImportRunStatusCanceled = SkillImportRunStatusCancelled

	SkillImportItemStatusQueued     = "queued"
	SkillImportItemStatusProcessing = "processing"
	SkillImportItemStatusReady      = "ready"
	SkillImportItemStatusUnchanged  = "unchanged"
	SkillImportItemStatusBlocked    = "blocked"
	SkillImportItemStatusFailed     = "failed"
	SkillImportItemStatusPublished  = "published"
	SkillImportItemStatusSkipped    = "skipped"
	SkillImportItemStatusCancelled  = "cancelled"

	SkillImportItemStatusPending  = SkillImportItemStatusQueued
	SkillImportItemStatusFetching = SkillImportItemStatusProcessing
	SkillImportItemStatusPrepared = SkillImportItemStatusReady
	SkillImportItemStatusCanceled = SkillImportItemStatusCancelled

	SkillImportPublishPolicyReview = "review"
	SkillImportPublishPolicyAuto   = "auto_publish"

	SkillImportMetadataPolicyCreateOnly = "create_only"
	SkillImportMetadataPolicyRefresh    = "refresh"

	SkillImportEventLevelDebug = "debug"
	SkillImportEventLevelInfo  = "info"
	SkillImportEventLevelWarn  = "warn"
	SkillImportEventLevelError = "error"
)

var (
	ErrSkillImportSourceNotFound   = infraerrors.NotFound("SKILL_IMPORT_SOURCE_NOT_FOUND", "skill import source not found")
	ErrSkillImportScheduleNotFound = infraerrors.NotFound("SKILL_IMPORT_SCHEDULE_NOT_FOUND", "skill import schedule not found")
	ErrSkillImportRunNotFound      = infraerrors.NotFound("SKILL_IMPORT_RUN_NOT_FOUND", "skill import run not found")
	ErrSkillImportItemNotFound     = infraerrors.NotFound("SKILL_IMPORT_ITEM_NOT_FOUND", "skill import item not found")
	ErrSkillImportOriginNotFound   = infraerrors.NotFound("SKILL_IMPORT_ORIGIN_NOT_FOUND", "skill import origin not found")
	ErrSkillImportConflict         = infraerrors.Conflict("SKILL_IMPORT_CONFLICT", "skill import record conflicts with an existing record")
	ErrSkillImportInvalidState     = infraerrors.Conflict("SKILL_IMPORT_INVALID_STATE", "skill import record is not in the required state")
	ErrSkillImportLeaseLost        = infraerrors.Conflict("SKILL_IMPORT_LEASE_LOST", "skill import lease is no longer owned by this worker")
	ErrSkillImportPublishInvalid   = infraerrors.Conflict("SKILL_IMPORT_PUBLISH_INVALID", "one or more eligible import items cannot be published")
	ErrSkillImportVersionYanked    = infraerrors.Conflict("SKILL_IMPORT_VERSION_YANKED", "the matching imported package version is yanked and cannot be republished")
)

// SkillImportStableKey is the durable upstream identity. Rank, slug and URL are
// deliberately excluded because they may change between catalog snapshots.
type SkillImportStableKey struct {
	SourceID   int64  `json:"source_id"`
	Namespace  string `json:"namespace"`
	ExternalID string `json:"external_id"`
}

type SkillImportSource struct {
	ID              int64           `json:"id"`
	Name            string          `json:"name"`
	Adapter         string          `json:"adapter"`
	Namespace       string          `json:"namespace"`
	BaseURL         string          `json:"base_url"`
	SourceConfig    json.RawMessage `json:"source_config"`
	CatalogPriority int             `json:"catalog_priority"`
	Enabled         bool            `json:"enabled"`
	CreatedBy       *int64          `json:"created_by,omitempty"`
	UpdatedBy       *int64          `json:"updated_by,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type SkillImportSchedule struct {
	ID             int64              `json:"id"`
	SourceID       int64              `json:"source_id"`
	Source         *SkillImportSource `json:"source,omitempty"`
	Name           string             `json:"name"`
	Enabled        bool               `json:"enabled"`
	CronExpression string             `json:"cron_expression"`
	Timezone       string             `json:"timezone"`
	Selection      json.RawMessage    `json:"selection"`
	RunConfig      json.RawMessage    `json:"run_config"`
	PublishPolicy  string             `json:"publish_policy"`
	MetadataPolicy string             `json:"metadata_policy"`
	NextRunAt      *time.Time         `json:"next_run_at"`
	LastRunAt      *time.Time         `json:"last_run_at"`
	LastRunID      *int64             `json:"last_run_id"`
	LeaseOwner     *string            `json:"-"`
	LeaseExpiresAt *time.Time         `json:"-"`
	CreatedBy      *int64             `json:"created_by,omitempty"`
	UpdatedBy      *int64             `json:"updated_by,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
}

type SkillImportRunCounts struct {
	Requested  int `json:"requested"`
	Discovered int `json:"discovered"`
	Prepared   int `json:"prepared"`
	Created    int `json:"created"`
	Updated    int `json:"updated"`
	Unchanged  int `json:"unchanged"`
	Skipped    int `json:"skipped"`
	Blocked    int `json:"blocked"`
	Failed     int `json:"failed"`
	Published  int `json:"published"`
}

type SkillImportRun struct {
	ID                int64                `json:"id"`
	SourceID          int64                `json:"source_id"`
	ScheduleID        *int64               `json:"schedule_id,omitempty"`
	ParentRunID       *int64               `json:"parent_run_id,omitempty"`
	TriggerType       string               `json:"trigger_type"`
	Mode              string               `json:"mode"`
	Status            string               `json:"status"`
	RequestConfig     json.RawMessage      `json:"request_config"`
	Snapshot          json.RawMessage      `json:"snapshot"`
	SnapshotSHA256    string               `json:"snapshot_sha256,omitempty"`
	ScheduledFor      *time.Time           `json:"scheduled_for,omitempty"`
	Counts            SkillImportRunCounts `json:"counts"`
	CancelRequestedAt *time.Time           `json:"cancel_requested_at,omitempty"`
	CancelRequestedBy *int64               `json:"cancel_requested_by,omitempty"`
	AttemptCount      int                  `json:"attempt_count"`
	NextAttemptAt     *time.Time           `json:"next_attempt_at,omitempty"`
	LastErrorCode     string               `json:"last_error_code,omitempty"`
	LastErrorMessage  string               `json:"last_error_message,omitempty"`
	LeaseOwner        *string              `json:"-"`
	LeaseExpiresAt    *time.Time           `json:"-"`
	HeartbeatAt       *time.Time           `json:"-"`
	CreatedBy         *int64               `json:"created_by,omitempty"`
	StartedAt         *time.Time           `json:"started_at,omitempty"`
	FinishedAt        *time.Time           `json:"finished_at,omitempty"`
	CreatedAt         time.Time            `json:"created_at"`
	UpdatedAt         time.Time            `json:"updated_at"`
}

type SkillImportDesiredSkill struct {
	Slug                  string   `json:"slug"`
	DisplayName           string   `json:"display_name"`
	Summary               string   `json:"summary"`
	Description           string   `json:"description"`
	Category              string   `json:"category"`
	Tags                  []string `json:"tags"`
	Icon                  string   `json:"icon"`
	ExamplePrompts        []string `json:"example_prompts"`
	RiskNotes             string   `json:"risk_notes"`
	OriginURL             string   `json:"origin_url"`
	SourceURL             string   `json:"source_url"`
	SourceRepository      string   `json:"source_repository"`
	Featured              bool     `json:"featured"`
	SortOrder             int      `json:"sort_order"`
	CatalogSourcePriority int      `json:"catalog_source_priority"`
	CatalogSourceRank     *int     `json:"catalog_source_rank,omitempty"`
}

type SkillImportRunItem struct {
	ID                  int64                   `json:"id"`
	RunID               int64                   `json:"run_id"`
	StableKey           SkillImportStableKey    `json:"stable_key"`
	Rank                *int                    `json:"rank,omitempty"`
	MarketSlug          string                  `json:"market_slug"`
	Status              string                  `json:"status"`
	UpstreamName        string                  `json:"upstream_name"`
	OriginURL           string                  `json:"origin_url"`
	SourceRevision      string                  `json:"source_revision,omitempty"`
	SourceContentSHA256 string                  `json:"source_content_sha256,omitempty"`
	PackageSHA256       string                  `json:"package_sha256,omitempty"`
	StageAction         string                  `json:"stage_action,omitempty"`
	DesiredSkill        SkillImportDesiredSkill `json:"desired_skill"`
	SourcePayload       json.RawMessage         `json:"source_payload"`
	ValidationReport    SkillValidationReport   `json:"validation_report"`
	Provenance          json.RawMessage         `json:"provenance"`
	LicenseUnverified   bool                    `json:"license_unverified"`
	ExcludedFiles       []string                `json:"excluded_files"`
	Warnings            []SkillValidationIssue  `json:"warnings"`
	SkillID             *int64                  `json:"skill_id,omitempty"`
	VersionID           *int64                  `json:"version_id,omitempty"`
	AttemptCount        int                     `json:"attempt_count"`
	NextAttemptAt       *time.Time              `json:"next_attempt_at,omitempty"`
	ErrorCode           string                  `json:"error_code,omitempty"`
	ErrorMessage        string                  `json:"error_message,omitempty"`
	LeaseOwner          *string                 `json:"-"`
	LeaseExpiresAt      *time.Time              `json:"-"`
	HeartbeatAt         *time.Time              `json:"-"`
	StartedAt           *time.Time              `json:"started_at,omitempty"`
	CompletedAt         *time.Time              `json:"completed_at,omitempty"`
	CreatedAt           time.Time               `json:"created_at"`
	UpdatedAt           time.Time               `json:"updated_at"`
}

type SkillOrigin struct {
	ID                  int64                `json:"id"`
	StableKey           SkillImportStableKey `json:"stable_key"`
	SkillID             int64                `json:"skill_id"`
	MarketSlug          string               `json:"market_slug"`
	OriginURL           string               `json:"origin_url"`
	SourceRevision      string               `json:"source_revision,omitempty"`
	SourceContentSHA256 string               `json:"source_content_sha256,omitempty"`
	FirstSeenRunID      int64                `json:"first_seen_run_id"`
	LastSeenRunID       int64                `json:"last_seen_run_id"`
	FirstSeenAt         time.Time            `json:"first_seen_at"`
	LastSeenAt          time.Time            `json:"last_seen_at"`
	LastRank            *int                 `json:"last_rank,omitempty"`
	Active              bool                 `json:"active"`
	CreatedAt           time.Time            `json:"created_at"`
	UpdatedAt           time.Time            `json:"updated_at"`
}

type SkillVersionOrigin struct {
	ID                  int64           `json:"id"`
	VersionID           int64           `json:"version_id"`
	OriginID            int64           `json:"origin_id"`
	RunItemID           int64           `json:"run_item_id"`
	SourceRevision      string          `json:"source_revision,omitempty"`
	SourceContentSHA256 string          `json:"source_content_sha256,omitempty"`
	Transformed         bool            `json:"transformed"`
	Provenance          json.RawMessage `json:"provenance"`
	CreatedAt           time.Time       `json:"created_at"`
}

type SkillImportEvent struct {
	ID        int64           `json:"id"`
	RunID     int64           `json:"run_id"`
	RunItemID *int64          `json:"run_item_id,omitempty"`
	Level     string          `json:"level"`
	EventType string          `json:"event_type"`
	Message   string          `json:"message"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

type SkillImportListFilter struct {
	Search   string `json:"search,omitempty"`
	Status   string `json:"status,omitempty"`
	SourceID *int64 `json:"source_id,omitempty"`
	RunID    *int64 `json:"run_id,omitempty"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}

type SkillImportListResult[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Pages    int   `json:"pages"`
}

type SkillImportRunProgress struct {
	RunID      int64                `json:"run_id"`
	Status     string               `json:"status"`
	Counts     SkillImportRunCounts `json:"counts"`
	TotalItems int                  `json:"total_items"`
	Percent    float64              `json:"percent"`
	UpdatedAt  time.Time            `json:"updated_at"`
}

type SkillImportRunItemPatch struct {
	Status              string
	MarketSlug          string
	SourceRevision      string
	SourceContentSHA256 string
	PackageSHA256       string
	DesiredSkill        SkillImportDesiredSkill
	ValidationReport    SkillValidationReport
	Provenance          json.RawMessage
	LicenseUnverified   bool
	ExcludedFiles       []string
	Warnings            []SkillValidationIssue
	SkillID             *int64
	VersionID           *int64
	ErrorCode           string
	ErrorMessage        string
	NextAttemptAt       *time.Time
}

type SkillImportPreparedArtifact struct {
	Changelog           string                `json:"changelog"`
	ManifestName        string                `json:"manifest_name"`
	ManifestDescription string                `json:"manifest_description"`
	SkillMD             string                `json:"skill_md"`
	PackageData         []byte                `json:"-"`
	PackageSHA256       string                `json:"package_sha256"`
	ByteSize            int64                 `json:"byte_size"`
	UnpackedSize        int64                 `json:"unpacked_size"`
	FileCount           int                   `json:"file_count"`
	FileManifest        []SkillArchiveFile    `json:"file_manifest"`
	ValidationReport    SkillValidationReport `json:"validation_report"`
	Transformed         bool                  `json:"transformed"`
}

type SkillImportStagePreparedInput struct {
	RunID               int64                       `json:"run_id"`
	RunItemID           int64                       `json:"run_item_id"`
	RunWorkerID         string                      `json:"-"`
	ItemLeaseOwner      string                      `json:"-"`
	ActorID             *int64                      `json:"-"`
	StableKey           SkillImportStableKey        `json:"stable_key"`
	Rank                *int                        `json:"rank,omitempty"`
	DesiredSkill        SkillImportDesiredSkill     `json:"desired_skill"`
	OriginURL           string                      `json:"origin_url"`
	SourceRevision      string                      `json:"source_revision,omitempty"`
	SourceContentSHA256 string                      `json:"source_content_sha256,omitempty"`
	Artifact            SkillImportPreparedArtifact `json:"artifact"`
	Provenance          json.RawMessage             `json:"provenance"`
	Transformed         bool                        `json:"transformed"`
	LicenseUnverified   bool                        `json:"license_unverified"`
	ExcludedFiles       []string                    `json:"excluded_files"`
	Warnings            []SkillValidationIssue      `json:"warnings"`
}

const (
	SkillImportStageActionCreate      = "create"
	SkillImportStageActionNewVersion  = "new_version"
	SkillImportStageActionUnchanged   = "unchanged"
	SkillImportStageActionRenormalize = "renormalize"
)

type SkillImportStagePreparedResult struct {
	Action     string `json:"action"`
	MarketSlug string `json:"market_slug"`
	SkillID    int64  `json:"skill_id"`
	VersionID  int64  `json:"version_id"`
	Version    string `json:"version"`
}

type SkillImportBootstrapCursor struct {
	AfterSkillID int64 `json:"after_skill_id"`
	Limit        int   `json:"limit"`
}

type SkillImportBootstrapCandidate struct {
	SkillID          int64  `json:"skill_id"`
	Slug             string `json:"slug"`
	SortOrder        int    `json:"sort_order"`
	OriginURL        string `json:"origin_url"`
	SourceURL        string `json:"source_url"`
	SourceRepository string `json:"source_repository"`
	VersionID        int64  `json:"version_id"`
	Version          string `json:"version"`
	SHA256           string `json:"sha256"`
	PackageData      []byte `json:"-"`
}

type SkillImportBootstrapOriginInput struct {
	RunID               int64                `json:"run_id"`
	RunWorkerID         string               `json:"-"`
	StableKey           SkillImportStableKey `json:"stable_key"`
	SkillID             int64                `json:"skill_id"`
	VersionID           int64                `json:"version_id"`
	MarketSlug          string               `json:"market_slug"`
	OriginURL           string               `json:"origin_url"`
	SourceRevision      string               `json:"source_revision,omitempty"`
	SourceContentSHA256 string               `json:"source_content_sha256,omitempty"`
	Provenance          json.RawMessage      `json:"provenance"`
}

type SkillImportBootstrapOriginResult struct {
	OriginID        int64 `json:"origin_id"`
	RunItemID       int64 `json:"run_item_id"`
	VersionOriginID int64 `json:"version_origin_id"`
}

type SkillImportPublishItemResult struct {
	RunItemID int64 `json:"run_item_id"`
	SkillID   int64 `json:"skill_id"`
	VersionID int64 `json:"version_id"`
}

type SkillImportPublishResult struct {
	RunID       int64                          `json:"run_id"`
	Status      string                         `json:"status"`
	Items       []SkillImportPublishItemResult `json:"items"`
	Counts      SkillImportRunCounts           `json:"counts"`
	PublishedAt time.Time                      `json:"published_at"`
}

// SkillImportRepository contains persistence primitives only. Discovery,
// validation, retry policy and scheduling decisions belong to the service and
// worker layers.
type SkillImportRepository interface {
	CreateSource(ctx context.Context, source *SkillImportSource) error
	UpdateSource(ctx context.Context, source *SkillImportSource) error
	GetSource(ctx context.Context, id int64) (*SkillImportSource, error)
	ListSources(ctx context.Context, filter SkillImportListFilter) ([]SkillImportSource, int64, error)

	CreateSchedule(ctx context.Context, schedule *SkillImportSchedule) error
	UpdateSchedule(ctx context.Context, schedule *SkillImportSchedule) error
	GetSchedule(ctx context.Context, id int64) (*SkillImportSchedule, error)
	ListSchedules(ctx context.Context, filter SkillImportListFilter) ([]SkillImportSchedule, int64, error)
	ClaimDueSchedule(ctx context.Context, workerID string, now, leaseUntil time.Time) (*SkillImportSchedule, error)
	CompleteScheduleClaim(ctx context.Context, scheduleID int64, workerID string, runID *int64, lastRunAt, nextRunAt time.Time) error
	ReleaseScheduleClaim(ctx context.Context, scheduleID int64, workerID string, nextRunAt time.Time) error

	CreateRun(ctx context.Context, run *SkillImportRun, idempotencyKeyHash string) error
	GetRun(ctx context.Context, id int64) (*SkillImportRun, error)
	ListRuns(ctx context.Context, filter SkillImportListFilter) ([]SkillImportRun, int64, error)
	ClaimNextRun(ctx context.Context, workerID string, now, leaseUntil time.Time) (*SkillImportRun, error)
	HeartbeatRun(ctx context.Context, runID int64, workerID string, leaseUntil time.Time) error
	UpdateRunStatus(ctx context.Context, runID int64, workerID, status, errorCode, errorMessage string, nextAttemptAt *time.Time) error
	RequestRunCancellation(ctx context.Context, runID, actorID int64) (bool, error)
	RefreshRunCounts(ctx context.Context, runID int64) (SkillImportRunCounts, error)
	CompleteDiscovery(ctx context.Context, runID int64, workerID string, snapshot json.RawMessage, snapshotSHA256 string, items []SkillImportRunItem) error

	CreateRunItems(ctx context.Context, items []SkillImportRunItem) error
	GetRunItem(ctx context.Context, id int64) (*SkillImportRunItem, error)
	ListRunItems(ctx context.Context, filter SkillImportListFilter) ([]SkillImportRunItem, int64, error)
	GetNextRunItemRetryAt(ctx context.Context, runID int64) (*time.Time, error)
	ClaimNextRunItem(ctx context.Context, runID int64, runWorkerID, itemLeaseOwner string, now, leaseUntil time.Time) (*SkillImportRunItem, error)
	HeartbeatRunItem(ctx context.Context, itemID int64, itemLeaseOwner, runWorkerID string, leaseUntil time.Time) error
	CompleteRunItem(ctx context.Context, itemID int64, itemLeaseOwner, runWorkerID string, patch SkillImportRunItemPatch) error
	ResetFailedItems(ctx context.Context, runID int64, itemIDs []int64) (int64, error)
	ListEligibleItemIDs(ctx context.Context, runID int64) ([]int64, error)
	// StagePreparedItem stores a bounded normalized artifact on the run item.
	// It must not create or mutate marketplace Skill/origin/version rows.
	StagePreparedItem(ctx context.Context, input SkillImportStagePreparedInput) (*SkillImportStagePreparedResult, error)

	AppendEvent(ctx context.Context, event *SkillImportEvent) error
	ListEvents(ctx context.Context, filter SkillImportListFilter) ([]SkillImportEvent, int64, error)
	DeleteTerminalRunEventsBefore(ctx context.Context, before time.Time) (int64, error)

	GetOriginByStableKey(ctx context.Context, key SkillImportStableKey) (*SkillOrigin, error)
	UpsertOrigin(ctx context.Context, origin *SkillOrigin) error
	CreateVersionOrigin(ctx context.Context, origin *SkillVersionOrigin) error
	ListBootstrapCandidates(ctx context.Context, cursor SkillImportBootstrapCursor) ([]SkillImportBootstrapCandidate, error)
	BootstrapOrigin(ctx context.Context, input SkillImportBootstrapOriginInput) (*SkillImportBootstrapOriginResult, error)

	// PublishEligibleItems locks exactly itemIDs and atomically materializes and
	// publishes that complete staged cohort. Blocked/failed items must never be
	// included; any failed identity/version/state check rolls back every market
	// row created for the cohort.
	PublishEligibleItems(ctx context.Context, runID int64, itemIDs []int64, workerID string, actorID *int64) (*SkillImportPublishResult, error)
}
