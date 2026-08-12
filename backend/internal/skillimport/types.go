// Package skillimport contains the source-agnostic ingestion core for the
// Skill marketplace. It deliberately has no persistence, handler, scheduler,
// or marketplace lifecycle dependencies: callers own orchestration and may
// safely resume work from the values defined here.
package skillimport

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

const CoreVersion = "1.0.0"

// SourceAdapter discovers stable upstream identities and acquires their raw
// files. An adapter must never execute upstream code or write to the market.
type SourceAdapter interface {
	Type() string
	Version() string
	ValidateConfig(json.RawMessage) error
	Discover(context.Context, DiscoverRequest) (DiscoveryPage, error)
	Acquire(context.Context, AcquireRequest) (SourceBundle, error)
}

// NewBuiltInAdapters constructs one shared GitHub adapter so skills.sh
// acquisitions reuse repository snapshots and the configured API credential.
func NewBuiltInAdapters(fetcher HTTPFetcher, githubToken string) []SourceAdapter {
	github := NewGitHubAdapter(fetcher, githubToken)
	return []SourceAdapter{
		NewSkillsSHAdapter(fetcher, github),
		github,
		NewWellKnownAdapter(fetcher),
		NewManifestAdapter(fetcher),
	}
}

type DiscoverRequest struct {
	Config    json.RawMessage `json:"config"`
	Selection json.RawMessage `json:"selection,omitempty"`
	BaseURL   string          `json:"base_url,omitempty"`
	Cursor    string          `json:"cursor,omitempty"`
}

type AcquireRequest struct {
	Config  json.RawMessage `json:"config"`
	BaseURL string          `json:"base_url,omitempty"`
	Skill   DiscoveredSkill `json:"skill"`
}

// DiscoveredSkill is the immutable, persistence-friendly identity contract.
// AdapterType + Namespace + ExternalID is the stable key; Rank is observation
// metadata and must never participate in identity or checkpoint keys.
type DiscoveredSkill struct {
	AdapterType   string           `json:"adapter_type"`
	Namespace     string           `json:"namespace"`
	ExternalID    string           `json:"external_id"`
	SuggestedName string           `json:"suggested_name"`
	SuggestedSlug string           `json:"suggested_slug"`
	Description   string           `json:"description,omitempty"`
	CanonicalURL  string           `json:"canonical_url,omitempty"`
	Revision      string           `json:"revision,omitempty"`
	Rank          *int             `json:"rank,omitempty"`
	Metrics       map[string]int64 `json:"metrics,omitempty"`
	Opaque        json.RawMessage  `json:"opaque,omitempty"`
}

func (s DiscoveredSkill) StableKey() string {
	return s.AdapterType + "\x00" + s.Namespace + "\x00" + s.ExternalID
}

type DiscoveryPage struct {
	Items      []DiscoveredSkill `json:"items"`
	NextCursor string            `json:"next_cursor,omitempty"`
	Evidence   []Evidence        `json:"evidence,omitempty"`
}

type SourceFile struct {
	Path       string `json:"path"`
	Data       []byte `json:"data,omitempty"`
	Executable bool   `json:"executable,omitempty"`
}

// Evidence describes a bounded HTTP observation. URL is stored without
// credentials or fragments by the fetcher.
type Evidence struct {
	URL          string    `json:"url"`
	CapturedAt   time.Time `json:"captured_at"`
	ETag         string    `json:"etag,omitempty"`
	LastModified string    `json:"last_modified,omitempty"`
	SHA256       string    `json:"sha256"`
	ByteSize     int64     `json:"byte_size"`
	ContentType  string    `json:"content_type,omitempty"`
}

type LicenseEvidence struct {
	Kind   string `json:"kind"`
	Path   string `json:"path,omitempty"`
	URL    string `json:"url,omitempty"`
	SHA256 string `json:"sha256,omitempty"`
}

// SourceBundle is raw source truth. UpstreamContentHash hashes the logical
// file set, whereas a normalized archive has its own, separate SHA-256.
type SourceBundle struct {
	Files               []SourceFile      `json:"files"`
	Revision            string            `json:"revision,omitempty"`
	UpstreamContentHash string            `json:"upstream_content_hash"`
	CanonicalURL        string            `json:"canonical_url,omitempty"`
	LicenseEvidence     []LicenseEvidence `json:"license_evidence,omitempty"`
	Evidence            []Evidence        `json:"evidence,omitempty"`
	IntegrityVerified   bool              `json:"integrity_verified"`
	NeedsReview         bool              `json:"needs_review"`
	ReviewReasons       []string          `json:"review_reasons,omitempty"`
}

type ExcludedFile struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
	Bytes  int64  `json:"bytes,omitempty"`
}

type NormalizeInput struct {
	Skill      DiscoveredSkill `json:"skill"`
	Bundle     SourceBundle    `json:"bundle"`
	MarketSlug string          `json:"market_slug"`
}

// NormalizedSkill is produced even for policy blocks, allowing the caller to
// persist a reviewable outcome. PackageData is present only when Blocked=false.
type NormalizedSkill struct {
	MarketSlug          string                         `json:"market_slug"`
	DisplayName         string                         `json:"display_name"`
	Description         string                         `json:"description"`
	UpstreamContentHash string                         `json:"upstream_content_hash"`
	NormalizedSHA256    string                         `json:"normalized_sha256,omitempty"`
	PackageData         []byte                         `json:"package_data,omitempty"`
	UnpackedSize        int64                          `json:"unpacked_size,omitempty"`
	FileCount           int                            `json:"file_count,omitempty"`
	FileManifest        []service.SkillArchiveFile     `json:"file_manifest,omitempty"`
	ValidationReport    *service.SkillValidationReport `json:"validation_report,omitempty"`
	LicenseEvidence     []LicenseEvidence              `json:"license_evidence,omitempty"`
	LicenseUnverified   bool                           `json:"license_unverified"`
	ExcludedFiles       []ExcludedFile                 `json:"excluded_files,omitempty"`
	Warnings            []service.SkillValidationIssue `json:"warnings,omitempty"`
	Transformed         bool                           `json:"transformed"`
	NeedsReview         bool                           `json:"needs_review"`
	ReviewReasons       []string                       `json:"review_reasons,omitempty"`
	Blocked             bool                           `json:"blocked"`
	BlockedReasons      []string                       `json:"blocked_reasons,omitempty"`
}
