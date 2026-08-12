package skillimport

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const ManifestAdapterType = "manifest"

const MaxInlineManifestBytes = 5 * 1024 * 1024

var maxInlineManifestEncodedBytes = base64.StdEncoding.EncodedLen(MaxInlineManifestBytes)

type ManifestAdapter struct {
	fetcher HTTPFetcher
}

type manifestConfig struct {
	URL              string   `json:"url,omitempty"`
	BaseURL          string   `json:"base_url,omitempty"`
	InlineDataBase64 string   `json:"inline_data_base64,omitempty"`
	InlineDataBytes  int      `json:"inline_data_byte_size,omitempty"`
	InlineDataSHA256 string   `json:"inline_data_sha256,omitempty"`
	InlineEncodedLen int      `json:"inline_data_encoded_bytes,omitempty"`
	UploadOnly       bool     `json:"upload_only,omitempty"`
	Format           string   `json:"format,omitempty"`
	AllowedHosts     []string `json:"allowed_hosts,omitempty"`
	DefaultNamespace string   `json:"default_namespace,omitempty"`
	SkillID          string   `json:"skill_id,omitempty"`
	Name             string   `json:"name,omitempty"`
	Description      string   `json:"description,omitempty"`
	Digest           string   `json:"digest,omitempty"`
	LicenseURL       string   `json:"license_url,omitempty"`
	Revision         string   `json:"revision,omitempty"`
}

type manifestDocument struct {
	SchemaVersion int             `json:"schema_version,omitempty"`
	Skills        []manifestEntry `json:"skills"`
}

type manifestEntry struct {
	Namespace    string                `json:"namespace,omitempty"`
	ExternalID   string                `json:"external_id,omitempty"`
	ID           string                `json:"id,omitempty"`
	SkillID      string                `json:"skill_id,omitempty"`
	Name         string                `json:"name,omitempty"`
	Slug         string                `json:"slug,omitempty"`
	Description  string                `json:"description,omitempty"`
	Type         string                `json:"type,omitempty"`
	URL          string                `json:"url,omitempty"`
	Digest       string                `json:"digest,omitempty"`
	Files        []remoteFileReference `json:"files,omitempty"`
	LicenseURL   string                `json:"license_url,omitempty"`
	Revision     string                `json:"revision,omitempty"`
	CanonicalURL string                `json:"canonical_url,omitempty"`
	Rank         int                   `json:"rank,omitempty"`
}

type manifestCursor struct {
	Offset int `json:"offset"`
}

func NewManifestAdapter(fetcher HTTPFetcher) *ManifestAdapter {
	return &ManifestAdapter{fetcher: fetcher}
}

func (a *ManifestAdapter) Type() string    { return ManifestAdapterType }
func (a *ManifestAdapter) Version() string { return "1.0.0" }

func (a *ManifestAdapter) ValidateConfig(raw json.RawMessage) error {
	_, err := parseManifestConfig(raw)
	return err
}

func parseManifestConfig(raw json.RawMessage) (manifestConfig, error) {
	config := manifestConfig{Format: "auto"}
	if err := decodeConfig(raw, &config); err != nil {
		return manifestConfig{}, err
	}
	config.URL = strings.TrimSpace(config.URL)
	config.BaseURL = strings.TrimSpace(config.BaseURL)
	config.InlineDataBase64 = strings.TrimSpace(config.InlineDataBase64)
	config.Format = strings.ToLower(strings.TrimSpace(config.Format))
	config.DefaultNamespace = strings.TrimSpace(config.DefaultNamespace)
	config.SkillID = strings.TrimSpace(config.SkillID)
	config.Name = strings.TrimSpace(config.Name)
	config.Description = strings.TrimSpace(config.Description)
	config.Revision = strings.TrimSpace(config.Revision)
	switch config.Format {
	case "auto", "json", "csv", "zip":
	default:
		return manifestConfig{}, errors.New("format must be auto, json, csv, or zip")
	}
	if config.URL != "" && config.InlineDataBase64 != "" {
		return manifestConfig{}, errors.New("exactly one of url or inline_data_base64 is required")
	}
	if config.URL == "" && config.InlineDataBase64 == "" && !config.UploadOnly {
		return manifestConfig{}, errors.New("exactly one of url or inline_data_base64 is required")
	}
	if config.URL != "" {
		if _, err := explicitHostsForBase(config.AllowedHosts, config.URL); err != nil {
			return manifestConfig{}, fmt.Errorf("url: %w", err)
		}
	} else if config.InlineDataBase64 != "" {
		if config.Format == "auto" {
			return manifestConfig{}, errors.New("format must be explicit for inline_data_base64")
		}
		decoded, err := decodeInlineManifest(config.InlineDataBase64)
		if err != nil {
			return manifestConfig{}, err
		}
		switch config.Format {
		case "json":
			if _, err := parseJSONManifest(decoded); err != nil {
				return manifestConfig{}, fmt.Errorf("inline JSON manifest: %w", err)
			}
		case "csv":
			if _, err := parseCSVManifest(decoded); err != nil {
				return manifestConfig{}, fmt.Errorf("inline CSV manifest: %w", err)
			}
		case "zip":
			files, err := ReadZIPArtifact(decoded)
			if err != nil {
				return manifestConfig{}, fmt.Errorf("inline ZIP manifest: %w", err)
			}
			if config.SkillID == "" || config.Name == "" || config.Description == "" {
				skillID, description, inferErr := inferInlineZIPIdentity(files)
				if inferErr != nil {
					return manifestConfig{}, inferErr
				}
				if config.SkillID == "" {
					config.SkillID = skillID
				}
				if config.Name == "" {
					config.Name = skillID
				}
				if config.Description == "" {
					config.Description = description
				}
			}
		}
	}
	if config.BaseURL != "" {
		if _, err := explicitHostsForBase(config.AllowedHosts, config.BaseURL); err != nil {
			return manifestConfig{}, fmt.Errorf("base_url: %w", err)
		}
	}
	if _, err := expectedDigest(config.Digest); err != nil {
		return manifestConfig{}, err
	}
	if manifestFormat(config) == "zip" && strings.TrimSpace(config.SkillID) == "" {
		return manifestConfig{}, errors.New("skill_id is required for a direct ZIP URL")
	}
	return config, nil
}

func (a *ManifestAdapter) Discover(ctx context.Context, request DiscoverRequest) (DiscoveryPage, error) {
	if a == nil || a.fetcher == nil {
		return DiscoveryPage{}, NewAdapterError(ManifestAdapterType, "discover", ErrorInvalidConfig, errors.New("HTTP fetcher is required"))
	}
	config, err := parseManifestConfig(request.Config)
	if err != nil {
		return DiscoveryPage{}, NewAdapterError(ManifestAdapterType, "discover", ErrorInvalidConfig, err)
	}
	if config.URL == "" && config.InlineDataBase64 == "" {
		return DiscoveryPage{}, NewAdapterError(ManifestAdapterType, "discover", ErrorInvalidConfig, errors.New("upload_only sources require a run-scoped uploaded artifact"))
	}
	selection, err := parseSelection(request.Selection)
	if err != nil {
		return DiscoveryPage{}, NewAdapterError(ManifestAdapterType, "discover", ErrorInvalidConfig, err)
	}
	entries := []manifestEntry(nil)
	evidence := []Evidence(nil)
	format := manifestFormat(config)
	if format == "zip" {
		artifactType := "archive"
		if config.InlineDataBase64 != "" {
			artifactType = "inline_zip"
		}
		entries = []manifestEntry{{
			Namespace: config.DefaultNamespace, ExternalID: config.SkillID, Name: config.Name,
			Slug: config.SkillID, Description: config.Description, Type: artifactType, URL: config.URL,
			Digest: config.Digest, LicenseURL: config.LicenseURL, Revision: config.Revision,
		}}
	} else {
		var manifestData []byte
		if config.InlineDataBase64 != "" {
			manifestData, err = decodeInlineManifest(config.InlineDataBase64)
		} else {
			result, fetchErr := a.fetchManifest(ctx, config)
			if fetchErr != nil {
				return DiscoveryPage{}, fetchErr
			}
			evidence = append(evidence, result.Evidence)
			manifestData = result.Body
		}
		if err != nil {
			return DiscoveryPage{}, NewAdapterError(ManifestAdapterType, "decode manifest", ErrorInvalidConfig, err)
		}
		if format == "json" {
			entries, err = parseJSONManifest(manifestData)
		} else {
			entries, err = parseCSVManifest(manifestData)
		}
		if err != nil {
			return DiscoveryPage{}, NewAdapterError(ManifestAdapterType, "decode manifest", ErrorInvalidSource, err)
		}
	}
	if len(entries) > maxDiscoveryLimit {
		return DiscoveryPage{}, NewAdapterError(ManifestAdapterType, "decode manifest", ErrorBlocked, fmt.Errorf("manifest exceeds %d entries", maxDiscoveryLimit))
	}
	cursor := manifestCursor{}
	if err := decodeCursor(request.Cursor, &cursor); err != nil || cursor.Offset < 0 {
		if err == nil {
			err = errors.New("invalid manifest cursor state")
		}
		return DiscoveryPage{}, NewAdapterError(ManifestAdapterType, "discover", ErrorInvalidSource, err)
	}
	startIndex := selection.StartRank - 1 + cursor.Offset
	if startIndex >= len(entries) || cursor.Offset >= selection.Limit {
		return DiscoveryPage{Items: []DiscoveredSkill{}, Evidence: evidence}, nil
	}
	count := selection.PageSize
	if remaining := selection.Limit - cursor.Offset; count > remaining {
		count = remaining
	}
	if available := len(entries) - startIndex; count > available {
		count = available
	}
	manifestBase := manifestBaseURL(config)
	manifestURL, _ := url.Parse(manifestBase)
	defaultNamespace := config.DefaultNamespace
	if defaultNamespace == "" {
		defaultNamespace = strings.ToLower(manifestURL.Hostname())
		if defaultNamespace == "" {
			defaultNamespace = "inline-manifest"
		}
	}
	items := make([]DiscoveredSkill, 0, count)
	for offset := 0; offset < count; offset++ {
		entry := entries[startIndex+offset]
		if err := normalizeManifestEntry(&entry, defaultNamespace, manifestBase); err != nil {
			return DiscoveryPage{}, NewAdapterError(ManifestAdapterType, "decode manifest", ErrorIntegrity, fmt.Errorf("entry %d: %w", startIndex+offset+1, err))
		}
		rank := entry.Rank
		if rank <= 0 {
			rank = startIndex + offset + 1
		}
		items = append(items, DiscoveredSkill{
			AdapterType: ManifestAdapterType, Namespace: entry.Namespace, ExternalID: entry.ExternalID,
			SuggestedName: entry.Name, SuggestedSlug: entry.Slug, Description: entry.Description,
			CanonicalURL: entry.CanonicalURL, Revision: entry.Revision, Rank: intPointer(rank), Opaque: safeOpaque(entry),
		})
	}
	cursor.Offset += len(items)
	next := ""
	if cursor.Offset < selection.Limit && selection.StartRank-1+cursor.Offset < len(entries) {
		next = encodeCursor(cursor)
	}
	return DiscoveryPage{Items: items, NextCursor: next, Evidence: evidence}, nil
}

func (a *ManifestAdapter) Acquire(ctx context.Context, request AcquireRequest) (SourceBundle, error) {
	if a == nil || a.fetcher == nil {
		return SourceBundle{}, NewAdapterError(ManifestAdapterType, "acquire", ErrorInvalidConfig, errors.New("HTTP fetcher is required"))
	}
	config, err := parseManifestConfig(request.Config)
	if err != nil {
		return SourceBundle{}, NewAdapterError(ManifestAdapterType, "acquire", ErrorInvalidConfig, err)
	}
	if config.URL == "" && config.InlineDataBase64 == "" {
		return SourceBundle{}, NewAdapterError(ManifestAdapterType, "acquire", ErrorInvalidConfig, errors.New("upload_only sources require a run-scoped uploaded artifact"))
	}
	var entry manifestEntry
	if err := json.Unmarshal(request.Skill.Opaque, &entry); err != nil {
		return SourceBundle{}, NewAdapterError(ManifestAdapterType, "acquire", ErrorInvalidSource, errors.New("skill discovery payload is invalid"))
	}
	manifestBase := manifestBaseURL(config)
	manifestURL, _ := url.Parse(manifestBase)
	defaultNamespace := config.DefaultNamespace
	if defaultNamespace == "" {
		defaultNamespace = strings.ToLower(manifestURL.Hostname())
		if defaultNamespace == "" {
			defaultNamespace = "inline-manifest"
		}
	}
	if err := normalizeManifestEntry(&entry, defaultNamespace, manifestBase); err != nil {
		return SourceBundle{}, NewAdapterError(ManifestAdapterType, "acquire", ErrorInvalidSource, err)
	}
	if config.InlineDataBase64 != "" && manifestFormat(config) == "zip" {
		raw, decodeErr := decodeInlineManifest(config.InlineDataBase64)
		if decodeErr != nil {
			return SourceBundle{}, NewAdapterError(ManifestAdapterType, "acquire", ErrorInvalidConfig, decodeErr)
		}
		if strings.TrimSpace(config.Digest) != "" {
			if digestErr := verifyDigest(config.Digest, raw); digestErr != nil {
				return SourceBundle{}, NewAdapterError(ManifestAdapterType, "verify artifact", ErrorIntegrity, digestErr)
			}
		}
		files, readErr := ReadZIPArtifact(raw)
		if readErr != nil {
			return SourceBundle{}, withAdapter(readErr, ManifestAdapterType, "read inline ZIP")
		}
		return SourceBundle{
			Files: files, Revision: config.Revision, UpstreamContentHash: canonicalFilesHash(files),
			CanonicalURL: request.Skill.CanonicalURL, LicenseEvidence: licenseEvidenceFromFiles(files),
			IntegrityVerified: strings.TrimSpace(config.Digest) != "",
		}, nil
	}
	if manifestBase == "" {
		manifestBase = entry.URL
	}
	return acquireRemoteArtifact(ctx, a.fetcher, ManifestAdapterType, manifestBase, config.AllowedHosts, remoteArtifact{
		Type: entry.Type, URL: entry.URL, Digest: entry.Digest, Files: entry.Files,
		LicenseURL: entry.LicenseURL, Revision: entry.Revision, CanonicalURL: entry.CanonicalURL,
	})
}

func (a *ManifestAdapter) fetchManifest(ctx context.Context, config manifestConfig) (FetchResult, error) {
	hosts, _ := explicitHostsForBase(config.AllowedHosts, config.URL)
	return a.fetcher.Get(ctx, config.URL, FetchOptions{
		Adapter: ManifestAdapterType, Operation: "fetch manifest", AllowedHosts: hosts, MaxBytes: 8 * 1024 * 1024,
	})
}

func manifestFormat(config manifestConfig) string {
	if config.Format != "auto" {
		return config.Format
	}
	parsed, _ := url.Parse(config.URL)
	lower := strings.ToLower(parsed.Path)
	switch {
	case strings.HasSuffix(lower, ".csv"):
		return "csv"
	case strings.HasSuffix(lower, ".zip"):
		return "zip"
	default:
		return "json"
	}
}

func manifestBaseURL(config manifestConfig) string {
	if strings.TrimSpace(config.URL) != "" {
		return strings.TrimSpace(config.URL)
	}
	return strings.TrimSpace(config.BaseURL)
}

func decodeInlineManifest(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, errors.New("inline_data_base64 is empty")
	}
	if len(value) > maxInlineManifestEncodedBytes {
		return nil, fmt.Errorf("inline_data_base64 exceeds %d encoded bytes", maxInlineManifestEncodedBytes)
	}
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(value)
	}
	if err != nil {
		return nil, errors.New("inline_data_base64 is not valid base64")
	}
	if len(decoded) == 0 || len(decoded) > MaxInlineManifestBytes {
		return nil, fmt.Errorf("inline_data_base64 must decode to between 1 and %d bytes", MaxInlineManifestBytes)
	}
	return decoded, nil
}

func inferInlineZIPIdentity(files []SourceFile) (string, string, error) {
	var raw []byte
	for _, file := range files {
		if file.Path == "SKILL.md" {
			raw = file.Data
			break
		}
	}
	if len(raw) == 0 {
		return "", "", errors.New("inline ZIP must contain a root SKILL.md")
	}
	text := strings.ReplaceAll(string(raw), "\r\n", "\n")
	if !strings.HasPrefix(text, "---\n") {
		return "", "", errors.New("inline ZIP SKILL.md must contain frontmatter or provide skill_id")
	}
	end := strings.Index(text[4:], "\n---\n")
	if end < 0 {
		return "", "", errors.New("inline ZIP SKILL.md frontmatter is not terminated")
	}
	var manifest struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}
	if err := yaml.Unmarshal([]byte(text[4:4+end]), &manifest); err != nil {
		return "", "", errors.New("inline ZIP SKILL.md frontmatter is invalid")
	}
	manifest.Name = strings.TrimSpace(manifest.Name)
	manifest.Description = strings.TrimSpace(manifest.Description)
	if manifest.Name == "" {
		return "", "", errors.New("inline ZIP SKILL.md frontmatter name is required")
	}
	return manifest.Name, manifest.Description, nil
}

func parseJSONManifest(raw []byte) ([]manifestEntry, error) {
	var document manifestDocument
	// json.Unmarshal is intentionally used instead of a single Decoder.Decode:
	// Decode alone accepts a valid first value followed by arbitrary bytes.
	if err := json.Unmarshal(raw, &document); err == nil && document.Skills != nil {
		return document.Skills, nil
	}
	var entries []manifestEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, errors.New("JSON manifest must be an object with skills or an array")
	}
	return entries, nil
}

func parseCSVManifest(raw []byte) ([]manifestEntry, error) {
	reader := csv.NewReader(bytes.NewReader(raw))
	reader.ReuseRecord = false
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read CSV header: %w", err)
	}
	columns := make(map[string]int, len(header))
	for index, value := range header {
		name := strings.ToLower(strings.TrimSpace(value))
		if name == "" {
			return nil, errors.New("CSV header contains an empty column name")
		}
		if _, duplicate := columns[name]; duplicate {
			return nil, fmt.Errorf("CSV header contains duplicate column %q", name)
		}
		columns[name] = index
	}
	entries := make([]manifestEntry, 0)
	for rowNumber := 2; ; rowNumber++ {
		row, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("read CSV row %d: %w", rowNumber, readErr)
		}
		value := func(name string) string {
			index, exists := columns[name]
			if !exists || index >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[index])
		}
		rank := 0
		if rawRank := value("rank"); rawRank != "" {
			parsed, parseErr := strconv.Atoi(rawRank)
			if parseErr != nil || parsed <= 0 {
				return nil, fmt.Errorf("CSV row %d rank must be a positive integer", rowNumber)
			}
			rank = parsed
		}
		entries = append(entries, manifestEntry{
			Namespace: value("namespace"), ExternalID: value("external_id"), ID: value("id"),
			SkillID: value("skill_id"), Name: value("name"), Slug: value("slug"),
			Description: value("description"), Type: value("type"), URL: value("url"),
			Digest: value("digest"), LicenseURL: value("license_url"), Revision: value("revision"),
			CanonicalURL: value("canonical_url"), Rank: rank,
		})
	}
	return entries, nil
}

func normalizeManifestEntry(entry *manifestEntry, defaultNamespace, manifestURL string) error {
	if entry == nil {
		return errors.New("entry is nil")
	}
	entry.Namespace = strings.TrimSpace(entry.Namespace)
	if entry.Namespace == "" {
		entry.Namespace = defaultNamespace
	}
	entry.ExternalID = firstNonEmpty(entry.ExternalID, entry.ID, entry.SkillID, entry.Slug)
	entry.Slug = firstNonEmpty(entry.Slug, entry.SkillID, entry.ExternalID)
	entry.Name = firstNonEmpty(entry.Name, entry.Slug, entry.ExternalID)
	entry.Description = strings.TrimSpace(entry.Description)
	entry.Type = strings.ToLower(strings.TrimSpace(entry.Type))
	entry.URL = strings.TrimSpace(entry.URL)
	entry.Revision = strings.TrimSpace(entry.Revision)
	if entry.Rank < 0 {
		return errors.New("rank must be positive when provided")
	}
	if entry.Namespace == "" || entry.ExternalID == "" || entry.Name == "" || entry.Slug == "" {
		return errors.New("namespace, stable id, name, and slug are required")
	}
	if len(entry.Namespace) > 160 || len(entry.ExternalID) > 512 || len(entry.Name) > 20000 || len(entry.Slug) > 20000 || len(entry.Description) > 20000 || len(entry.Revision) > 240 {
		return errors.New("manifest identity or display metadata exceeds the supported limit")
	}
	if entry.URL == "" && len(entry.Files) == 0 && entry.Type != "inline_zip" {
		return errors.New("url or files is required")
	}
	if _, err := expectedDigest(entry.Digest); err != nil {
		return err
	}
	if entry.CanonicalURL == "" {
		entry.CanonicalURL = entry.URL
	}
	if entry.CanonicalURL != "" {
		resolved, err := resolveReference(manifestURL, entry.CanonicalURL)
		if err != nil {
			return err
		}
		entry.CanonicalURL = resolved
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
