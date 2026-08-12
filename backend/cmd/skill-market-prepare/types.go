package main

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

const (
	snapshotSchemaVersion   = 1
	checkpointSchemaVersion = 1
	manifestSchemaVersion   = 1
)

type rankingRecord struct {
	Source   string `json:"source"`
	SkillID  string `json:"skillId"`
	Name     string `json:"name"`
	Installs int64  `json:"installs"`
}

type snapshotEntry struct {
	Rank         int    `json:"rank"`
	Source       string `json:"source"`
	OriginalSlug string `json:"original_slug"`
	MarketSlug   string `json:"market_slug"`
	OriginalName string `json:"original_name"`
	Name         string `json:"name"`
	Installs     int64  `json:"installs"`
}

type snapshotFile struct {
	SchemaVersion  int             `json:"schema_version"`
	DataSource     string          `json:"data_source"`
	SourceURL      string          `json:"source_url"`
	CapturedAt     time.Time       `json:"captured_at"`
	UpstreamSHA256 string          `json:"upstream_sha256"`
	AvailableCount int             `json:"available_count"`
	Limit          int             `json:"limit"`
	Skills         []snapshotEntry `json:"skills"`
}

type sourceCheckpoint struct {
	Source        string    `json:"source"`
	SourceURL     string    `json:"source_url"`
	CachePath     string    `json:"cache_path"`
	HeadCommit    string    `json:"head_commit,omitempty"`
	LastError     string    `json:"last_error,omitempty"`
	LastAttemptAt time.Time `json:"last_attempt_at,omitempty"`
}

type skillCheckpoint struct {
	Rank          int        `json:"rank"`
	Source        string     `json:"source"`
	OriginalSlug  string     `json:"original_slug"`
	MarketSlug    string     `json:"market_slug"`
	Status        string     `json:"status"`
	Attempts      int        `json:"attempts"`
	Acquisition   string     `json:"acquisition,omitempty"`
	AcquiredPath  string     `json:"acquired_path,omitempty"`
	DownloadHash  string     `json:"download_hash,omitempty"`
	PackageSHA256 string     `json:"package_sha256,omitempty"`
	LastError     string     `json:"last_error,omitempty"`
	LastAttemptAt time.Time  `json:"last_attempt_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
}

type checkpointFile struct {
	SchemaVersion  int                          `json:"schema_version"`
	SnapshotSHA256 string                       `json:"snapshot_sha256"`
	UpdatedAt      time.Time                    `json:"updated_at"`
	Sources        map[string]*sourceCheckpoint `json:"sources"`
	Skills         map[string]*skillCheckpoint  `json:"skills"`
}

type excludedFile struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
	Bytes  int64  `json:"bytes,omitempty"`
}

type auditRecord struct {
	Provider   string   `json:"provider"`
	Slug       string   `json:"slug,omitempty"`
	Status     string   `json:"status,omitempty"`
	Summary    string   `json:"summary,omitempty"`
	AuditedAt  string   `json:"audited_at,omitempty"`
	RiskLevel  string   `json:"risk_level,omitempty"`
	Categories []string `json:"categories,omitempty"`
}

type marketMetadata struct {
	Rank              int                            `json:"rank"`
	Source            string                         `json:"source"`
	OriginalSlug      string                         `json:"original_slug"`
	MarketSlug        string                         `json:"market_slug"`
	OriginalName      string                         `json:"original_name"`
	Name              string                         `json:"name"`
	Installs          int64                          `json:"installs"`
	SourceURL         string                         `json:"source_url"`
	SourceCommit      string                         `json:"source_commit,omitempty"`
	Category          string                         `json:"category"`
	Tags              []string                       `json:"tags"`
	Summary           string                         `json:"summary"`
	Description       string                         `json:"description"`
	RiskNotes         []string                       `json:"risk_notes"`
	Audits            []auditRecord                  `json:"audits,omitempty"`
	SortOrder         int                            `json:"sort_order"`
	PackagePath       string                         `json:"package_path"`
	MetadataPath      string                         `json:"metadata_path"`
	Version           string                         `json:"version"`
	Changelog         string                         `json:"changelog"`
	ExcludedFiles     []excludedFile                 `json:"excluded_files"`
	SHA256            string                         `json:"sha256"`
	ByteSize          int64                          `json:"byte_size"`
	UnpackedSize      int64                          `json:"unpacked_size"`
	FileCount         int                            `json:"file_count"`
	FileManifest      []service.SkillArchiveFile     `json:"file_manifest"`
	ValidationReport  service.SkillValidationReport  `json:"validation_report"`
	Acquisition       string                         `json:"acquisition"`
	DownloadHash      string                         `json:"skills_sh_download_hash,omitempty"`
	SnapshotSHA256    string                         `json:"skills_sh_snapshot_sha256"`
	SnapshotHash      string                         `json:"snapshot_hash"`
	CapturedAt        time.Time                      `json:"skills_sh_captured_at"`
	LicenseUnverified bool                           `json:"license_unverified"`
	LicenseFiles      []string                       `json:"license_files"`
	Warnings          []service.SkillValidationIssue `json:"warnings"`
	RightsNotice      string                         `json:"rights_notice"`
}

type failureRecord struct {
	Rank         int    `json:"rank"`
	Source       string `json:"source"`
	OriginalSlug string `json:"original_slug"`
	MarketSlug   string `json:"market_slug"`
	Stage        string `json:"stage"`
	Error        string `json:"error"`
}

type manifestFile struct {
	SchemaVersion  int              `json:"schema_version"`
	Source         string           `json:"source"`
	View           string           `json:"view"`
	Snapshot       string           `json:"snapshot"`
	GeneratedAt    time.Time        `json:"generated_at"`
	SnapshotSHA256 string           `json:"snapshot_sha256"`
	Requested      int              `json:"requested"`
	Completed      int              `json:"completed"`
	Count          int              `json:"count"`
	Failed         int              `json:"failed"`
	Skills         []marketMetadata `json:"skills"`
	Failures       []failureRecord  `json:"failures"`
	SlugMappings   []slugMapping    `json:"slug_mappings"`
}

type slugMapping struct {
	Rank         int    `json:"rank"`
	Source       string `json:"source"`
	OriginalSlug string `json:"original_slug"`
	MarketSlug   string `json:"market_slug"`
}

type acquiredSkill struct {
	Path         string
	Acquisition  string
	DownloadHash string
}
