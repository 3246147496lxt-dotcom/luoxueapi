package skillimport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

const WellKnownAdapterType = "well_known"

const (
	agentSkillsWellKnownPath  = ".well-known/agent-skills"
	legacySkillsWellKnownPath = ".well-known/skills"
	wellKnownSchemaV02        = "https://schemas.agentskills.io/discovery/0.2.0/schema.json"
)

type WellKnownAdapter struct {
	fetcher HTTPFetcher
}

type wellKnownConfig struct {
	BaseURL      string   `json:"base_url,omitempty"`
	AllowedHosts []string `json:"allowed_hosts,omitempty"`
}

type wellKnownIndex struct {
	Schema string           `json:"$schema,omitempty"`
	Skills []wellKnownSkill `json:"skills"`
}

type wellKnownSkill struct {
	Name             string   `json:"name"`
	Type             string   `json:"type,omitempty"`
	Description      string   `json:"description,omitempty"`
	URL              string   `json:"url,omitempty"`
	Digest           string   `json:"digest,omitempty"`
	Files            []string `json:"files,omitempty"`
	LicenseURL       string   `json:"license_url,omitempty"`
	IndexPath        string   `json:"well_known_path,omitempty"`
	ObservedIndexURL string   `json:"_observed_index_url,omitempty"`
}

type wellKnownCursor struct {
	Offset int `json:"offset"`
}

var errWellKnownSoftNotFound = errors.New("well-known endpoint returned a not-found envelope")

func NewWellKnownAdapter(fetcher HTTPFetcher) *WellKnownAdapter {
	return &WellKnownAdapter{fetcher: fetcher}
}

func (a *WellKnownAdapter) Type() string    { return WellKnownAdapterType }
func (a *WellKnownAdapter) Version() string { return "1.0.2" }

func (a *WellKnownAdapter) ValidateConfig(raw json.RawMessage) error {
	_, err := parseWellKnownConfig(raw, "")
	return err
}

func parseWellKnownConfig(raw json.RawMessage, requestBaseURL string) (wellKnownConfig, error) {
	config := wellKnownConfig{}
	if err := decodeConfig(raw, &config); err != nil {
		return wellKnownConfig{}, err
	}
	if strings.TrimSpace(requestBaseURL) != "" {
		config.BaseURL = strings.TrimSpace(requestBaseURL)
	}
	config.BaseURL = strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	if config.BaseURL == "" {
		// BaseURL is a first-class source field, so source configuration can be
		// validated before the handler combines both values.
		return config, nil
	}
	if _, err := explicitHostsForBase(config.AllowedHosts, config.BaseURL); err != nil {
		return wellKnownConfig{}, err
	}
	return config, nil
}

func (a *WellKnownAdapter) Discover(ctx context.Context, request DiscoverRequest) (DiscoveryPage, error) {
	if a == nil || a.fetcher == nil {
		return DiscoveryPage{}, NewAdapterError(WellKnownAdapterType, "discover", ErrorInvalidConfig, errors.New("HTTP fetcher is required"))
	}
	config, err := parseWellKnownConfig(request.Config, request.BaseURL)
	if err != nil || config.BaseURL == "" {
		if err == nil {
			err = errors.New("base_url is required")
		}
		return DiscoveryPage{}, NewAdapterError(WellKnownAdapterType, "discover", ErrorInvalidConfig, err)
	}
	selection, err := parseSelection(request.Selection)
	if err != nil {
		return DiscoveryPage{}, NewAdapterError(WellKnownAdapterType, "discover", ErrorInvalidConfig, err)
	}
	cursor := wellKnownCursor{}
	if err := decodeCursor(request.Cursor, &cursor); err != nil || cursor.Offset < 0 {
		if err == nil {
			err = errors.New("invalid well-known cursor state")
		}
		return DiscoveryPage{}, NewAdapterError(WellKnownAdapterType, "discover", ErrorInvalidSource, err)
	}
	index, indexPath, indexURL, evidence, err := a.fetchIndex(ctx, config)
	if err != nil {
		return DiscoveryPage{}, err
	}
	startIndex := selection.StartRank - 1 + cursor.Offset
	if startIndex >= len(index.Skills) || cursor.Offset >= selection.Limit {
		return DiscoveryPage{Items: []DiscoveredSkill{}, Evidence: []Evidence{evidence}}, nil
	}
	count := selection.PageSize
	if remaining := selection.Limit - cursor.Offset; count > remaining {
		count = remaining
	}
	if available := len(index.Skills) - startIndex; count > available {
		count = available
	}
	base, _ := url.Parse(config.BaseURL)
	items := make([]DiscoveredSkill, 0, count)
	for indexOffset := 0; indexOffset < count; indexOffset++ {
		entry := index.Skills[startIndex+indexOffset]
		entry.IndexPath = indexPath
		entry.ObservedIndexURL = indexURL
		entry.Name = strings.TrimSpace(entry.Name)
		if !safeWellKnownSkillName(entry.Name) {
			return DiscoveryPage{}, NewAdapterError(WellKnownAdapterType, "decode index", ErrorIntegrity, fmt.Errorf("well-known skill at index %d has an unsafe name", startIndex+indexOffset))
		}
		if entry.Name == "" || (strings.TrimSpace(entry.URL) == "" && len(entry.Files) == 0) {
			return DiscoveryPage{}, NewAdapterError(WellKnownAdapterType, "decode index", ErrorIntegrity, fmt.Errorf("well-known skill at index %d is malformed", startIndex+indexOffset))
		}
		canonical, resolveErr := resolveWellKnownReference(config, indexURL, "./"+url.PathEscape(entry.Name))
		if resolveErr != nil {
			return DiscoveryPage{}, NewAdapterError(WellKnownAdapterType, "decode index", ErrorIntegrity, fmt.Errorf("well-known skill at index %d has an invalid canonical URL: %w", startIndex+indexOffset, resolveErr))
		}
		if strings.TrimSpace(entry.URL) != "" {
			canonical, resolveErr = resolveWellKnownReference(config, indexURL, entry.URL)
			if resolveErr != nil {
				return DiscoveryPage{}, NewAdapterError(WellKnownAdapterType, "decode index", ErrorIntegrity, fmt.Errorf("well-known skill at index %d has an invalid artifact URL: %w", startIndex+indexOffset, resolveErr))
			}
		}
		rank := startIndex + indexOffset + 1
		items = append(items, DiscoveredSkill{
			AdapterType: WellKnownAdapterType, Namespace: strings.ToLower(base.Hostname()), ExternalID: entry.Name,
			SuggestedName: entry.Name, SuggestedSlug: entry.Name, Description: strings.TrimSpace(entry.Description),
			CanonicalURL: canonical, Rank: intPointer(rank), Opaque: safeOpaque(entry),
		})
	}
	cursor.Offset += len(items)
	next := ""
	if cursor.Offset < selection.Limit && selection.StartRank-1+cursor.Offset < len(index.Skills) {
		next = encodeCursor(cursor)
	}
	return DiscoveryPage{Items: items, NextCursor: next, Evidence: []Evidence{evidence}}, nil
}

func safeWellKnownSkillName(value string) bool {
	if value == "" || value != strings.TrimSpace(value) {
		return false
	}
	_, directory, err := safeArchivePath(value)
	return err == nil && !directory && !strings.Contains(value, "/")
}

func (a *WellKnownAdapter) Acquire(ctx context.Context, request AcquireRequest) (SourceBundle, error) {
	if a == nil || a.fetcher == nil {
		return SourceBundle{}, NewAdapterError(WellKnownAdapterType, "acquire", ErrorInvalidConfig, errors.New("HTTP fetcher is required"))
	}
	config, err := parseWellKnownConfig(request.Config, request.BaseURL)
	if err != nil || config.BaseURL == "" {
		if err == nil {
			err = errors.New("base_url is required")
		}
		return SourceBundle{}, NewAdapterError(WellKnownAdapterType, "acquire", ErrorInvalidConfig, err)
	}
	var entry wellKnownSkill
	if err := json.Unmarshal(request.Skill.Opaque, &entry); err != nil || !safeWellKnownSkillName(entry.Name) || !validWellKnownPath(entry.IndexPath) {
		index, indexPath, indexURL, _, fetchErr := a.fetchIndex(ctx, config)
		if fetchErr != nil {
			return SourceBundle{}, fetchErr
		}
		wanted := strings.TrimSpace(request.Skill.ExternalID)
		for _, candidate := range index.Skills {
			if strings.TrimSpace(candidate.Name) == wanted && safeWellKnownSkillName(candidate.Name) {
				entry = candidate
				entry.IndexPath = indexPath
				entry.ObservedIndexURL = indexURL
				break
			}
		}
		if strings.TrimSpace(entry.Name) == "" {
			return SourceBundle{}, NewAdapterError(WellKnownAdapterType, "acquire", ErrorNotFound, errors.New("skill is not present in the well-known index"))
		}
	}
	if !validWellKnownPath(entry.IndexPath) {
		return SourceBundle{}, NewAdapterError(WellKnownAdapterType, "acquire", ErrorIntegrity, errors.New("well-known discovery path is invalid"))
	}
	indexURL, err := observedWellKnownIndexURL(config, entry)
	if err != nil {
		return SourceBundle{}, NewAdapterError(WellKnownAdapterType, "acquire", ErrorIntegrity, err)
	}
	artifact := remoteArtifact{
		Type: entry.Type, URL: entry.URL, Digest: entry.Digest, LicenseURL: entry.LicenseURL,
		CanonicalURL: request.Skill.CanonicalURL,
	}
	// Discovery v0.2 defines URL as one complete artifact. "files" was removed
	// from that model and may still appear as extension metadata. Fetching both
	// would overlay duplicate content and invalidate the artifact digest.
	if strings.TrimSpace(entry.URL) == "" {
		for _, fileName := range entry.Files {
			name, directory, pathErr := safeArchivePath(strings.TrimSpace(fileName))
			if pathErr != nil || directory {
				if pathErr == nil {
					pathErr = errors.New("well-known file path must name a file")
				}
				return SourceBundle{}, NewAdapterError(WellKnownAdapterType, "acquire", ErrorUnsafe, pathErr)
			}
			fileURL, resolveErr := resolveWellKnownReference(config, indexURL, "./"+url.PathEscape(strings.TrimSpace(entry.Name))+"/"+escapeURLPath(name))
			if resolveErr != nil {
				return SourceBundle{}, NewAdapterError(WellKnownAdapterType, "resolve file", ErrorInvalidSource, resolveErr)
			}
			artifact.Files = append(artifact.Files, remoteFileReference{Path: name, URL: fileURL})
		}
	}
	return acquireRemoteArtifact(ctx, a.fetcher, WellKnownAdapterType, indexURL, config.AllowedHosts, artifact)
}

func (a *WellKnownAdapter) fetchIndex(ctx context.Context, config wellKnownConfig) (wellKnownIndex, string, string, Evidence, error) {
	hosts, err := explicitHostsForBase(config.AllowedHosts, config.BaseURL)
	if err != nil {
		return wellKnownIndex{}, "", "", Evidence{}, NewAdapterError(WellKnownAdapterType, "read index", ErrorInvalidConfig, err)
	}
	for pathIndex, indexPath := range []string{agentSkillsWellKnownPath, legacySkillsWellKnownPath} {
		endpoint := config.BaseURL + "/" + indexPath + "/index.json"
		result, fetchErr := a.fetcher.Get(ctx, endpoint, jsonFetchOptions(WellKnownAdapterType, "read index", hosts, 4*1024*1024, nil))
		if fetchErr != nil {
			if pathIndex == 0 && ErrorKindOf(fetchErr) == ErrorNotFound {
				continue
			}
			return wellKnownIndex{}, "", "", Evidence{}, fetchErr
		}
		index, decodeErr := decodeWellKnownIndex(result.Body)
		if errors.Is(decodeErr, errWellKnownSoftNotFound) {
			if pathIndex == 0 {
				continue
			}
			return wellKnownIndex{}, "", "", Evidence{}, NewAdapterError(WellKnownAdapterType, "read index", ErrorNotFound, decodeErr)
		}
		if decodeErr != nil {
			return wellKnownIndex{}, "", "", Evidence{}, NewAdapterError(WellKnownAdapterType, "decode index", ErrorInvalidSource, decodeErr)
		}
		if len(index.Skills) > maxDiscoveryLimit {
			return wellKnownIndex{}, "", "", Evidence{}, NewAdapterError(WellKnownAdapterType, "decode index", ErrorBlocked, fmt.Errorf("well-known index exceeds %d entries", maxDiscoveryLimit))
		}
		if validateErr := validateWellKnownIndex(index); validateErr != nil {
			return wellKnownIndex{}, "", "", Evidence{}, NewAdapterError(WellKnownAdapterType, "decode index", ErrorInvalidSource, validateErr)
		}
		observedIndexURL := strings.TrimSpace(result.Evidence.URL)
		if observedIndexURL == "" {
			observedIndexURL = endpoint
		}
		if _, validateErr := validateHTTPSImportTarget(observedIndexURL, hosts); validateErr != nil {
			return wellKnownIndex{}, "", "", Evidence{}, NewAdapterError(WellKnownAdapterType, "read index", ErrorUnsafe, fmt.Errorf("observed index URL is invalid: %w", validateErr))
		}
		return index, indexPath, observedIndexURL, result.Evidence, nil
	}
	return wellKnownIndex{}, "", "", Evidence{}, NewAdapterError(WellKnownAdapterType, "read index", ErrorNotFound, errors.New("neither agent-skills nor skills well-known index exists"))
}

func observedWellKnownIndexURL(config wellKnownConfig, entry wellKnownSkill) (string, error) {
	indexURL := strings.TrimSpace(entry.ObservedIndexURL)
	if indexURL == "" {
		indexURL = config.BaseURL + "/" + entry.IndexPath + "/index.json"
	}
	hosts, err := explicitHostsForBase(config.AllowedHosts, config.BaseURL)
	if err != nil {
		return "", err
	}
	validated, err := validateHTTPSImportTarget(indexURL, hosts)
	if err != nil {
		return "", fmt.Errorf("observed well-known index URL is invalid: %w", err)
	}
	return validated, nil
}

func resolveWellKnownReference(config wellKnownConfig, indexURL, reference string) (string, error) {
	hosts, err := explicitHostsForBase(config.AllowedHosts, config.BaseURL)
	if err != nil {
		return "", err
	}
	resolved, err := resolveReference(indexURL, reference)
	if err != nil {
		return "", err
	}
	return validateHTTPSImportTarget(resolved, hosts)
}

// decodeWellKnownIndex distinguishes an explicit empty catalog from a JSON
// error envelope served with HTTP 200. Only the narrow, observed NotFound
// shape is eligible for path fallback; malformed or schema-invalid JSON must
// remain visible to operators instead of being silently masked by legacy data.
func decodeWellKnownIndex(body []byte) (wellKnownIndex, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return wellKnownIndex{}, err
	}
	if fields == nil {
		return wellKnownIndex{}, errors.New("well-known index must be a JSON object")
	}
	if rawSkills, exists := fields["skills"]; exists {
		if bytes.Equal(bytes.TrimSpace(rawSkills), []byte("null")) {
			return wellKnownIndex{}, errors.New("well-known index skills must be an array")
		}
		var skills []wellKnownSkill
		if err := json.Unmarshal(rawSkills, &skills); err != nil {
			return wellKnownIndex{}, fmt.Errorf("well-known index skills must be an array: %w", err)
		}
		if skills == nil {
			return wellKnownIndex{}, errors.New("well-known index skills must be an array")
		}
		var schema string
		if rawSchema, exists := fields["$schema"]; exists {
			if err := json.Unmarshal(rawSchema, &schema); err != nil || strings.TrimSpace(schema) == "" {
				return wellKnownIndex{}, errors.New("well-known index $schema must be a non-empty string")
			}
			schema = strings.TrimSpace(schema)
		}
		return wellKnownIndex{Schema: schema, Skills: skills}, nil
	}
	if isWellKnownNotFoundEnvelope(fields) {
		return wellKnownIndex{}, errWellKnownSoftNotFound
	}
	return wellKnownIndex{}, errors.New("well-known index is missing the skills array")
}

func validateWellKnownIndex(index wellKnownIndex) error {
	if index.Schema == "" {
		// Absence is the backward-compatibility signal for the v0.1 files model.
		for position, entry := range index.Skills {
			if strings.TrimSpace(entry.URL) == "" && len(entry.Files) == 0 {
				return fmt.Errorf("legacy well-known skill at index %d must declare files or url", position)
			}
		}
		return nil
	}
	if index.Schema != wellKnownSchemaV02 {
		return fmt.Errorf("unsupported well-known index schema %q", index.Schema)
	}
	for position, entry := range index.Skills {
		name := strings.TrimSpace(entry.Name)
		if len(name) == 0 || len(name) > 64 || !marketSlugPattern.MatchString(name) {
			return fmt.Errorf("well-known v0.2 skill at index %d has an invalid name", position)
		}
		description := strings.TrimSpace(entry.Description)
		if description == "" || len(description) > 1024 {
			return fmt.Errorf("well-known v0.2 skill at index %d has an invalid description", position)
		}
		kind := strings.TrimSpace(entry.Type)
		if kind != "skill-md" && kind != "archive" {
			return fmt.Errorf("well-known v0.2 skill at index %d has unsupported type %q", position, entry.Type)
		}
		if strings.TrimSpace(entry.URL) == "" {
			return fmt.Errorf("well-known v0.2 skill at index %d is missing url", position)
		}
		if strings.TrimSpace(entry.Digest) == "" {
			return fmt.Errorf("well-known v0.2 skill at index %d is missing digest", position)
		}
		if _, err := expectedDigest(entry.Digest); err != nil {
			return fmt.Errorf("well-known v0.2 skill at index %d has invalid digest: %w", position, err)
		}
	}
	return nil
}

func isWellKnownNotFoundEnvelope(fields map[string]json.RawMessage) bool {
	rawMetadata, exists := fields["ResponseMetadata"]
	if !exists {
		return false
	}
	var metadata struct {
		Error *struct {
			Code    string `json:"Code"`
			Message string `json:"Message"`
		} `json:"Error"`
	}
	if err := json.Unmarshal(rawMetadata, &metadata); err != nil || metadata.Error == nil {
		return false
	}
	code := strings.ToLower(strings.TrimSpace(metadata.Error.Code))
	message := strings.ToLower(strings.TrimSpace(metadata.Error.Message))
	return (code == "notfound" || code == "not_found" || code == "404") && strings.Contains(message, "not found")
}

func validWellKnownPath(value string) bool {
	return value == agentSkillsWellKnownPath || value == legacySkillsWellKnownPath
}
