package skillimport

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"golang.org/x/text/unicode/norm"
	"gopkg.in/yaml.v3"
)

var marketSlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

var normalizerForbiddenExtensions = map[string]string{
	".7z": "nested archive", ".a": "binary object", ".bin": "binary file", ".bz2": "nested archive",
	".class": "compiled class", ".dll": "binary library", ".dylib": "binary library", ".exe": "executable binary",
	".gz": "nested archive", ".jar": "nested executable archive", ".o": "binary object", ".p12": "key container",
	".pem": "key container", ".pfx": "key container", ".ppk": "key container", ".rar": "nested archive",
	".so": "binary library", ".tar": "nested archive", ".tgz": "nested archive", ".wasm": "compiled binary",
	".xz": "nested archive", ".zip": "nested archive",
}

type Normalizer struct{}

func NewNormalizer() *Normalizer { return &Normalizer{} }

func (n *Normalizer) Normalize(input NormalizeInput) (NormalizedSkill, error) {
	result := NormalizedSkill{
		MarketSlug:          strings.TrimSpace(input.MarketSlug),
		DisplayName:         strings.TrimSpace(input.Skill.SuggestedName),
		UpstreamContentHash: strings.TrimSpace(input.Bundle.UpstreamContentHash),
		LicenseEvidence:     mergeLicenseEvidence(input.Bundle.LicenseEvidence, licenseEvidenceFromFiles(input.Bundle.Files)),
		NeedsReview:         input.Bundle.NeedsReview,
		ReviewReasons:       append([]string(nil), input.Bundle.ReviewReasons...),
	}
	if !input.Bundle.IntegrityVerified {
		result.NeedsReview = true
		result.ReviewReasons = append(result.ReviewReasons, "upstream content did not include an independently verified digest")
	}
	result.LicenseUnverified = len(result.LicenseEvidence) == 0
	if result.DisplayName == "" {
		result.DisplayName = strings.TrimSpace(input.Skill.SuggestedSlug)
	}
	if !marketSlugPattern.MatchString(result.MarketSlug) || len(result.MarketSlug) > 64 {
		return result, NewAdapterError(input.Skill.AdapterType, "normalize", ErrorInvalidConfig, errors.New("market slug must be lowercase kebab-case and at most 64 bytes"))
	}
	if len(input.Bundle.Files) == 0 {
		result.Blocked = true
		result.BlockedReasons = []string{"source bundle contains no files"}
		return finalizeNormalized(result), nil
	}
	actualUpstreamHash := canonicalFilesHash(input.Bundle.Files)
	if result.UpstreamContentHash == "" {
		result.UpstreamContentHash = actualUpstreamHash
	} else if !strings.EqualFold(result.UpstreamContentHash, actualUpstreamHash) {
		result.Blocked = true
		result.BlockedReasons = []string{"source bundle content hash does not match its logical file set"}
		return finalizeNormalized(result), nil
	}

	files, blocked := normalizeSourceSet(input.Bundle.Files)
	if len(blocked) > 0 {
		result.Blocked = true
		result.BlockedReasons = append(result.BlockedReasons, blocked...)
		return finalizeNormalized(result), nil
	}
	var skillIndex = -1
	for index := range files {
		if files[index].Path == "SKILL.md" {
			skillIndex = index
			break
		}
	}
	if skillIndex < 0 {
		result.Blocked = true
		result.BlockedReasons = []string{"SKILL.md is required at the source bundle root"}
		return finalizeNormalized(result), nil
	}
	rewritten, description, changed, rewriteReview, err := rewriteSkillMD(files[skillIndex].Data, result.MarketSlug, input.Skill.Description)
	if err != nil {
		result.Blocked = true
		result.BlockedReasons = []string{err.Error()}
		return finalizeNormalized(result), nil
	}
	files[skillIndex].Data = rewritten
	result.Description = description
	result.Transformed = changed
	if len(rewriteReview) > 0 {
		result.NeedsReview = true
		result.ReviewReasons = append(result.ReviewReasons, rewriteReview...)
	}

	included, excluded, selectionBlocked := selectMarketFiles(files)
	result.ExcludedFiles = excluded
	if len(selectionBlocked) > 0 {
		result.Blocked = true
		result.BlockedReasons = append(result.BlockedReasons, selectionBlocked...)
		return finalizeNormalized(result), nil
	}
	if len(excluded) > 0 {
		result.Transformed = true
		result.Warnings = append(result.Warnings, service.SkillValidationIssue{
			Code: "METADATA_EXCLUDED", Message: fmt.Sprintf("%d inert metadata file(s) were safely excluded from the marketplace package", len(excluded)),
		})
	}
	archive, err := buildNormalizerZIP(result.MarketSlug, included)
	if err != nil {
		return result, NewAdapterError(input.Skill.AdapterType, "normalize", ErrorTemporary, err)
	}
	validated, err := service.ValidateSkillArchive(archive, result.MarketSlug)
	if err != nil {
		result.Blocked = true
		var validationErr *service.SkillArchiveValidationError
		if errors.As(err, &validationErr) {
			report := validationErr.Report
			result.ValidationReport = &report
			for _, issue := range report.Errors {
				result.BlockedReasons = append(result.BlockedReasons, issue.Code+": "+issue.Message)
			}
		} else {
			result.BlockedReasons = append(result.BlockedReasons, err.Error())
		}
		return finalizeNormalized(result), nil
	}
	report := validated.ValidationReport
	result.PackageData = validated.PackageData
	result.NormalizedSHA256 = validated.SHA256
	result.UnpackedSize = validated.UnpackedSize
	result.FileCount = len(validated.FileManifest)
	result.FileManifest = validated.FileManifest
	result.ValidationReport = &report
	result.Description = validated.ManifestDescription
	result.Warnings = append(result.Warnings, report.Warnings...)
	result.LicenseUnverified = len(result.LicenseEvidence) == 0
	if result.LicenseUnverified {
		result.Warnings = append(result.Warnings, service.SkillValidationIssue{
			Code: "LICENSE_UNVERIFIED", Message: "no upstream LICENSE, LICENCE, COPYING, or NOTICE evidence was present",
		})
	}
	if !input.Bundle.IntegrityVerified {
		result.Warnings = append(result.Warnings, service.SkillValidationIssue{
			Code: "UPSTREAM_INTEGRITY_UNVERIFIED", Message: "the source did not provide an independently verified content digest",
		})
	}
	return finalizeNormalized(result), nil
}

func NormalizeSkill(input NormalizeInput) (NormalizedSkill, error) {
	return NewNormalizer().Normalize(input)
}

func normalizeSourceSet(source []SourceFile) ([]SourceFile, []string) {
	files := append([]SourceFile(nil), source...)
	seen := make(map[string]string, len(files))
	blocked := make([]string, 0)
	for index := range files {
		name, directory, err := safeArchivePath(files[index].Path)
		if err != nil || directory || norm.NFC.String(name) != name {
			blocked = append(blocked, fmt.Sprintf("unsafe or non-canonical source path %q", files[index].Path))
			continue
		}
		files[index].Path = name
		folded := strings.ToLower(name)
		if prior, exists := seen[folded]; exists {
			blocked = append(blocked, fmt.Sprintf("duplicate or case-conflicting source paths %q and %q", prior, name))
			continue
		}
		seen[folded] = name
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, dedupeSortedStrings(blocked)
}

func selectMarketFiles(files []SourceFile) ([]SourceFile, []ExcludedFile, []string) {
	prioritized := append([]SourceFile(nil), files...)
	sort.SliceStable(prioritized, func(i, j int) bool {
		priority := func(file SourceFile) int {
			if file.Path == "SKILL.md" {
				return 0
			}
			if isLicenseBase(file.Path) {
				return 1
			}
			return 2
		}
		left, right := priority(prioritized[i]), priority(prioritized[j])
		if left == right {
			return prioritized[i].Path < prioritized[j].Path
		}
		return left < right
	})
	included := make([]SourceFile, 0, min(len(prioritized), service.SkillArchiveMaxFiles))
	excluded := make([]ExcludedFile, 0)
	blocked := make([]string, 0)
	var total int64
	for _, file := range prioritized {
		reason := exclusionReason(file)
		if reason == "" && len(file.Path)+65 > service.SkillArchiveMaxPathBytes {
			reason = "archive path exceeds marketplace length limit"
		}
		if reason == "" && strings.Count(file.Path, "/")+2 > service.SkillArchiveMaxDepth {
			reason = "archive path exceeds marketplace depth limit"
		}
		if reason == "" && len(included) >= service.SkillArchiveMaxFiles {
			reason = "marketplace file count limit"
		}
		if reason == "" && total+int64(len(file.Data)) > service.SkillArchiveMaxUnpackedBytes {
			reason = "marketplace unpacked size limit"
		}
		if reason != "" {
			if file.Path == "SKILL.md" {
				blocked = append(blocked, "SKILL.md cannot be safely packaged: "+reason)
				continue
			}
			excluded = append(excluded, ExcludedFile{Path: file.Path, Reason: reason, Bytes: int64(len(file.Data))})
			switch reason {
			case "marketplace file count limit":
				blocked = append(blocked, "source bundle exceeds the marketplace file count limit; functional files would be omitted")
			case "marketplace unpacked size limit":
				blocked = append(blocked, "source bundle exceeds the marketplace unpacked size limit; functional files would be omitted")
			default:
				if !isSafelyIgnorableMetadata(file.Path) {
					blocked = append(blocked, fmt.Sprintf("functional source file %q cannot be safely omitted: %s", file.Path, reason))
				}
			}
			continue
		}
		included = append(included, file)
		total += int64(len(file.Data))
	}
	sort.Slice(included, func(i, j int) bool { return included[i].Path < included[j].Path })
	sort.Slice(excluded, func(i, j int) bool { return excluded[i].Path < excluded[j].Path })
	return included, excluded, blocked
}

func exclusionReason(file SourceFile) string {
	base := strings.ToLower(path.Base(file.Path))
	for _, component := range strings.Split(strings.ToLower(file.Path), "/") {
		if component == ".git" || component == ".svn" || component == ".hg" || component == "__macosx" {
			return "version-control or operating-system metadata"
		}
	}
	if base == ".ds_store" || strings.HasPrefix(base, "._") {
		return "operating-system metadata"
	}
	if base == ".coverage" || strings.HasPrefix(base, ".coverage.") {
		return "test-coverage metadata"
	}
	if base == ".env" || strings.HasPrefix(base, ".env.") {
		return "environment file"
	}
	if base == "id_rsa" || base == "id_dsa" || base == "id_ecdsa" || base == "id_ed25519" {
		return "private-key filename"
	}
	if reason := normalizerForbiddenExtensions[strings.ToLower(path.Ext(base))]; reason != "" {
		return reason
	}
	if int64(len(file.Data)) > service.SkillArchiveMaxFileBytes {
		return "file exceeds 1 MiB marketplace limit"
	}
	if !utf8.Valid(file.Data) || bytes.IndexByte(file.Data, 0) >= 0 {
		return "binary or non-UTF-8 content"
	}
	if containsPrivateKeyMaterial(file.Data) {
		return "private-key material"
	}
	return ""
}

func isSafelyIgnorableMetadata(filePath string) bool {
	lowerPath := strings.ToLower(filePath)
	for _, component := range strings.Split(lowerPath, "/") {
		if component == ".git" || component == ".svn" || component == ".hg" || component == "__macosx" {
			return true
		}
	}
	base := path.Base(lowerPath)
	return base == ".ds_store" || strings.HasPrefix(base, "._") || base == ".coverage" || strings.HasPrefix(base, ".coverage.")
}

func containsPrivateKeyMaterial(data []byte) bool {
	upper := bytes.ToUpper(data)
	for _, marker := range []string{
		"-----BEGIN PRIVATE KEY-----", "-----BEGIN ENCRYPTED PRIVATE KEY-----",
		"-----BEGIN RSA PRIVATE KEY-----", "-----BEGIN EC PRIVATE KEY-----",
		"-----BEGIN DSA PRIVATE KEY-----", "-----BEGIN OPENSSH PRIVATE KEY-----",
		"-----BEGIN PGP PRIVATE KEY BLOCK-----", "PUTTY-USER-KEY-FILE-",
	} {
		if bytes.Contains(upper, []byte(marker)) {
			return true
		}
	}
	return false
}

func rewriteSkillMD(raw []byte, slug, fallbackDescription string) ([]byte, string, bool, []string, error) {
	if !utf8.Valid(raw) || bytes.IndexByte(raw, 0) >= 0 {
		return nil, "", false, nil, errors.New("SKILL.md is not valid UTF-8 text")
	}
	text := strings.ReplaceAll(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\r", "\n")
	body := strings.TrimSpace(text)
	frontmatter := ""
	missingFrontmatter := true
	if strings.HasPrefix(text, "---\n") {
		if end := strings.Index(text[4:], "\n---\n"); end >= 0 {
			frontmatter = text[4 : 4+end]
			body = strings.TrimSpace(text[4+end+len("\n---\n"):])
			missingFrontmatter = false
		}
	}
	if body == "" {
		return nil, "", false, nil, errors.New("SKILL.md contains no instruction body")
	}
	root := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	if frontmatter != "" {
		var document yaml.Node
		if err := yaml.Unmarshal([]byte(frontmatter), &document); err != nil || len(document.Content) == 0 || document.Content[0].Kind != yaml.MappingNode {
			return nil, "", false, nil, errors.New("SKILL.md frontmatter is not a YAML mapping")
		}
		root = document.Content[0]
	}
	originalName := yamlMappingScalar(root, "name")
	description := strings.TrimSpace(yamlMappingScalar(root, "description"))
	reviewReasons := make([]string, 0, 3)
	if description == "" {
		description = strings.TrimSpace(fallbackDescription)
		if description == "" {
			description = "Imported skill " + slug + "."
		}
		reviewReasons = append(reviewReasons, "SKILL.md was missing a usable description and one was supplied deterministically")
	}
	if utf8.RuneCountInString(description) > 1024 {
		description = truncateUTF8Runes(description, 1024)
		reviewReasons = append(reviewReasons, "SKILL.md description exceeded the marketplace limit and was truncated")
	}
	setYAMLMappingScalar(root, "name", slug)
	setYAMLMappingScalar(root, "description", description)
	frontmatterBytes, err := yaml.Marshal(root)
	if err != nil {
		return nil, "", false, nil, err
	}
	output := append([]byte("---\n"), frontmatterBytes...)
	output = append(output, []byte("---\n\n"+body+"\n")...)
	changed := !bytes.Equal(output, raw)
	if missingFrontmatter {
		reviewReasons = append(reviewReasons, "SKILL.md had no valid frontmatter and was wrapped deterministically")
	} else if strings.TrimSpace(originalName) != slug {
		// Name rewriting is expected when marketplace slug collision handling
		// assigns a suffix, so it is auditable transformation but not a review
		// blocker by itself.
		changed = true
	}
	return output, description, changed, dedupeSortedStrings(reviewReasons), nil
}

func yamlMappingScalar(mapping *yaml.Node, key string) string {
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		if mapping.Content[index].Value == key && mapping.Content[index+1].Kind == yaml.ScalarNode {
			return mapping.Content[index+1].Value
		}
	}
	return ""
}

func setYAMLMappingScalar(mapping *yaml.Node, key, value string) {
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		if mapping.Content[index].Value == key {
			mapping.Content[index+1] = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
			return
		}
	}
	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value},
	)
}

func buildNormalizerZIP(slug string, files []SourceFile) ([]byte, error) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	fixedTime := time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)
	for _, file := range files {
		header := &zip.FileHeader{Name: slug + "/" + file.Path, Method: zip.Deflate}
		header.Modified = fixedTime
		if file.Executable {
			header.SetMode(0o755)
		} else {
			header.SetMode(0o644)
		}
		entry, err := writer.CreateHeader(header)
		if err != nil {
			_ = writer.Close()
			return nil, err
		}
		if _, err := entry.Write(file.Data); err != nil {
			_ = writer.Close()
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func finalizeNormalized(result NormalizedSkill) NormalizedSkill {
	result.BlockedReasons = dedupeSortedStrings(result.BlockedReasons)
	result.ReviewReasons = dedupeSortedStrings(result.ReviewReasons)
	for _, reason := range result.ReviewReasons {
		result.Warnings = append(result.Warnings, service.SkillValidationIssue{
			Code: "REVIEW_REQUIRED", Message: reason,
		})
	}
	sort.SliceStable(result.Warnings, func(i, j int) bool {
		if result.Warnings[i].Path == result.Warnings[j].Path {
			if result.Warnings[i].Code == result.Warnings[j].Code {
				return result.Warnings[i].Message < result.Warnings[j].Message
			}
			return result.Warnings[i].Code < result.Warnings[j].Code
		}
		return result.Warnings[i].Path < result.Warnings[j].Path
	})
	if len(result.Warnings) > 1 {
		deduped := result.Warnings[:1]
		for _, warning := range result.Warnings[1:] {
			prior := deduped[len(deduped)-1]
			if warning.Code == prior.Code && warning.Message == prior.Message && warning.Path == prior.Path {
				continue
			}
			deduped = append(deduped, warning)
		}
		result.Warnings = deduped
	}
	return result
}

func truncateUTF8Runes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}
