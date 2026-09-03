package application

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"

	domain "github.com/Wei-Shaw/sub2api/internal/service"
	"gopkg.in/yaml.v3"
)

type BootstrapEvidence struct {
	Key                 domain.SkillImportStableKey `json:"stable_key"`
	OriginURL           string                      `json:"origin_url"`
	SourceRevision      string                      `json:"source_revision"`
	SourceContentSHA256 string                      `json:"source_content_sha256"`
	PackageSHA256       string                      `json:"package_sha256"`
	Provenance          json.RawMessage             `json:"provenance"`
}

// ExtractSkillsSHBootstrapEvidence only accepts provenance embedded in a
// package that still passes the current archive validator. It never infers an
// identity from a mutable rank or from a display name.
func ExtractSkillsSHBootstrapEvidence(sourceID int64, candidate domain.SkillImportBootstrapCandidate) (*BootstrapEvidence, error) {
	if sourceID <= 0 || candidate.SkillID <= 0 || candidate.VersionID <= 0 || candidate.Slug == "" {
		return nil, errors.New("bootstrap candidate identity is incomplete")
	}
	validated, err := domain.ValidateSkillArchive(candidate.PackageData, candidate.Slug)
	if err != nil {
		return nil, fmt.Errorf("validate existing package: %w", err)
	}
	if candidate.SHA256 != "" && !strings.EqualFold(candidate.SHA256, validated.SHA256) {
		return nil, errors.New("existing package SHA-256 does not match its normalized bytes")
	}
	skillMD, err := readBootstrapSkillMD(candidate.PackageData, candidate.Slug)
	if err != nil {
		return nil, err
	}
	frontmatter, err := bootstrapFrontmatter(skillMD)
	if err != nil {
		return nil, err
	}
	source, externalID, revision, contentHash := bootstrapMetadataIdentity(frontmatter.Metadata)
	if source == "" || externalID == "" {
		if candidate.Slug == "frontend-design" && candidate.SourceRepository == "anthropics/skills" {
			source, externalID = "anthropics/skills", "frontend-design"
		} else {
			return nil, errors.New("existing package has no unambiguous skills.sh identity evidence")
		}
	}
	if !sourceNamespacePattern.MatchString(source) || externalID != strings.TrimSpace(externalID) || externalID == "" || len(externalID) > 512 || len(revision) > 240 {
		return nil, errors.New("existing package has invalid or oversized skills.sh identity evidence")
	}
	for _, current := range externalID {
		if current < 0x20 || current == 0x7f {
			return nil, errors.New("existing package identity contains control characters")
		}
	}
	originURL := "https://skills.sh/" + escapeBootstrapPath(source) + "/" + url.PathEscape(externalID)
	provenance, _ := json.Marshal(map[string]any{
		"bootstrap": true, "source": source, "external_id": externalID,
		"snapshot_hash": revision, "skill_id": candidate.SkillID,
		"version_id": candidate.VersionID, "package_sha256": validated.SHA256,
	})
	return &BootstrapEvidence{
		Key:       domain.SkillImportStableKey{SourceID: sourceID, Namespace: source, ExternalID: externalID},
		OriginURL: originURL, SourceRevision: revision, SourceContentSHA256: contentHash,
		PackageSHA256: validated.SHA256, Provenance: provenance,
	}, nil
}

type bootstrapManifest struct {
	Metadata map[string]any `yaml:"metadata"`
}

func bootstrapFrontmatter(raw []byte) (bootstrapManifest, error) {
	text := strings.ReplaceAll(string(raw), "\r\n", "\n")
	if !strings.HasPrefix(text, "---\n") {
		return bootstrapManifest{}, errors.New("existing SKILL.md has no YAML frontmatter")
	}
	end := strings.Index(text[4:], "\n---\n")
	if end < 0 {
		return bootstrapManifest{}, errors.New("existing SKILL.md frontmatter is unterminated")
	}
	var manifest bootstrapManifest
	if err := yaml.Unmarshal([]byte(text[4:4+end]), &manifest); err != nil {
		return bootstrapManifest{}, fmt.Errorf("decode existing SKILL.md metadata: %w", err)
	}
	return manifest, nil
}

func bootstrapMetadataIdentity(metadata map[string]any) (source, externalID, revision, contentHash string) {
	if metadata == nil {
		return "", "", "", ""
	}
	source = bootstrapString(metadata["original_source"])
	externalID = bootstrapString(metadata["skills_sh_id"])
	revision = bootstrapString(metadata["snapshot_hash"])
	contentHash = bootstrapCanonicalContentHash(metadata["canonical_content_hash"])
	if nested, ok := bootstrapStringMap(metadata["skills_sh_import"]); ok {
		if source == "" {
			source = firstBootstrapString(nested, "source", "namespace", "original_source")
		}
		if externalID == "" {
			externalID = firstBootstrapString(nested, "skill_id", "external_id", "slug")
		}
		if revision == "" {
			revision = firstBootstrapString(nested, "snapshot_hash", "revision", "content_hash")
		}
		if contentHash == "" {
			contentHash = bootstrapCanonicalContentHash(nested["canonical_content_hash"])
		}
	}
	return strings.TrimSpace(source), strings.TrimSpace(externalID), strings.ToLower(strings.TrimSpace(revision)), contentHash
}

func bootstrapCanonicalContentHash(value any) string {
	digest := strings.ToLower(bootstrapString(value))
	if len(digest) != 64 {
		return ""
	}
	for _, current := range digest {
		if current < '0' || current > '9' {
			if current < 'a' || current > 'f' {
				return ""
			}
		}
	}
	return digest
}

func bootstrapString(value any) string {
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	return ""
}

func bootstrapStringMap(value any) (map[string]any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		return typed, true
	case map[any]any:
		result := make(map[string]any, len(typed))
		for key, item := range typed {
			text, ok := key.(string)
			if !ok {
				return nil, false
			}
			result[text] = item
		}
		return result, true
	default:
		return nil, false
	}
}

func firstBootstrapString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := bootstrapString(values[key]); value != "" {
			return value
		}
	}
	return ""
}

func readBootstrapSkillMD(raw []byte, slug string) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return nil, errors.New("existing package is not a ZIP")
	}
	wanted := slug + "/SKILL.md"
	for _, file := range reader.File {
		if file.Name != wanted {
			continue
		}
		stream, err := file.Open()
		if err != nil {
			return nil, err
		}
		data, readErr := io.ReadAll(io.LimitReader(stream, domain.SkillArchiveMaxSkillMDBytes+1))
		closeErr := stream.Close()
		if readErr != nil || closeErr != nil || int64(len(data)) > domain.SkillArchiveMaxSkillMDBytes {
			return nil, errors.New("cannot safely read existing SKILL.md")
		}
		return data, nil
	}
	return nil, errors.New("existing package has no root SKILL.md")
}

func escapeBootstrapPath(value string) string {
	parts := strings.Split(strings.Trim(value, "/"), "/")
	for index := range parts {
		parts[index] = url.PathEscape(parts[index])
	}
	return strings.Join(parts, "/")
}
