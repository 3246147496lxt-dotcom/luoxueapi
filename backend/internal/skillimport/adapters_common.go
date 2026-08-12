package skillimport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

const (
	defaultDiscoveryLimit  = 500
	maxDiscoveryLimit      = 5000
	defaultAdapterPageSize = 100
)

type discoverySelection struct {
	StartRank int `json:"start_rank"`
	Limit     int `json:"limit"`
	PageSize  int `json:"page_size,omitempty"`
}

func parseSelection(raw json.RawMessage) (discoverySelection, error) {
	selection := discoverySelection{StartRank: 1, Limit: defaultDiscoveryLimit, PageSize: defaultAdapterPageSize}
	if len(raw) > 0 {
		if err := decodeConfig(raw, &selection); err != nil {
			return discoverySelection{}, err
		}
	}
	if selection.StartRank <= 0 {
		return discoverySelection{}, errors.New("start_rank must be positive")
	}
	if selection.Limit <= 0 || selection.Limit > maxDiscoveryLimit {
		return discoverySelection{}, fmt.Errorf("limit must be between 1 and %d", maxDiscoveryLimit)
	}
	if selection.PageSize <= 0 || selection.PageSize > 500 {
		return discoverySelection{}, errors.New("page_size must be between 1 and 500")
	}
	return selection, nil
}

type remoteFileReference struct {
	Path   string `json:"path"`
	URL    string `json:"url"`
	Digest string `json:"digest,omitempty"`
}

type remoteArtifact struct {
	Type         string                `json:"type,omitempty"`
	URL          string                `json:"url,omitempty"`
	Digest       string                `json:"digest,omitempty"`
	Files        []remoteFileReference `json:"files,omitempty"`
	LicenseURL   string                `json:"license_url,omitempty"`
	Revision     string                `json:"revision,omitempty"`
	CanonicalURL string                `json:"canonical_url,omitempty"`
}

func acquireRemoteArtifact(
	ctx context.Context,
	fetcher HTTPFetcher,
	adapter string,
	baseURL string,
	allowedHosts []string,
	artifact remoteArtifact,
) (SourceBundle, error) {
	if fetcher == nil {
		return SourceBundle{}, NewAdapterError(adapter, "acquire", ErrorInvalidConfig, errors.New("HTTP fetcher is required"))
	}
	allowlist, err := explicitHostsForBase(allowedHosts, baseURL)
	if err != nil {
		return SourceBundle{}, NewAdapterError(adapter, "acquire", ErrorInvalidConfig, err)
	}
	files := make([]SourceFile, 0, len(artifact.Files)+1)
	evidence := make([]Evidence, 0, len(artifact.Files)+2)
	integrityChecks := 0

	if strings.TrimSpace(artifact.URL) != "" {
		resolved, resolveErr := resolveReference(baseURL, artifact.URL)
		if resolveErr != nil {
			return SourceBundle{}, NewAdapterError(adapter, "resolve artifact", ErrorInvalidSource, resolveErr)
		}
		result, fetchErr := fetcher.Get(ctx, resolved, FetchOptions{
			Adapter: adapter, Operation: "fetch artifact", AllowedHosts: allowlist,
			MaxBytes: MaxSourceArchiveBytes,
		})
		if fetchErr != nil {
			return SourceBundle{}, fetchErr
		}
		evidence = append(evidence, result.Evidence)
		if strings.TrimSpace(artifact.Digest) != "" {
			if digestErr := verifyDigest(artifact.Digest, result.Body); digestErr != nil {
				return SourceBundle{}, NewAdapterError(adapter, "verify artifact", ErrorIntegrity, digestErr)
			}
			integrityChecks++
		}
		kind := strings.ToLower(strings.TrimSpace(artifact.Type))
		if kind == "" || kind == "auto" {
			if strings.HasSuffix(strings.ToLower(resolved), ".zip") || allowedContentType(result.Evidence.ContentType, []string{"application/zip", "application/x-zip-compressed"}) {
				kind = "archive"
			} else {
				kind = "skill_md"
			}
		}
		switch kind {
		case "archive", "zip":
			files, err = ReadZIPArtifact(result.Body)
			if err != nil {
				return SourceBundle{}, withAdapter(err, adapter, "read artifact")
			}
		case "skill", "skill_md", "markdown":
			files = append(files, SourceFile{Path: "SKILL.md", Data: result.Body})
		default:
			return SourceBundle{}, NewAdapterError(adapter, "acquire", ErrorInvalidSource, fmt.Errorf("unsupported artifact type %q", artifact.Type))
		}
	}
	// Validate every declared path before any per-file network request. This
	// prevents a malformed late entry from causing a large amount of needless
	// outbound work and keeps rejection deterministic.
	declaredPaths := make(map[string]string, len(files)+len(artifact.Files))
	for _, file := range files {
		declaredPaths[strings.ToLower(file.Path)] = file.Path
	}
	for _, reference := range artifact.Files {
		name, isDir, pathErr := safeArchivePath(strings.TrimSpace(reference.Path))
		if pathErr != nil || isDir {
			if pathErr == nil {
				pathErr = errors.New("remote file path must name a file")
			}
			return SourceBundle{}, NewAdapterError(adapter, "acquire files", ErrorUnsafe, pathErr)
		}
		if prior, duplicate := declaredPaths[strings.ToLower(name)]; duplicate {
			return SourceBundle{}, NewAdapterError(adapter, "acquire files", ErrorUnsafe, fmt.Errorf("duplicate or case-conflicting source paths %q and %q", prior, name))
		}
		if _, resolveErr := resolveReference(baseURL, reference.URL); resolveErr != nil {
			return SourceBundle{}, NewAdapterError(adapter, "resolve file", ErrorInvalidSource, resolveErr)
		}
		declaredPaths[strings.ToLower(name)] = name
	}
	if len(files)+len(artifact.Files) > MaxSourceArchiveEntries {
		return SourceBundle{}, NewAdapterError(adapter, "acquire", ErrorBlocked, fmt.Errorf("source file count exceeds %d", MaxSourceArchiveEntries))
	}
	if len(files) > 0 {
		if validateErr := validateSourceFiles(files); validateErr != nil {
			return SourceBundle{}, NewAdapterError(adapter, "acquire", ErrorUnsafe, validateErr)
		}
	}
	var totalBytes int64
	seenPaths := make(map[string]string, len(files)+len(artifact.Files))
	for _, file := range files {
		totalBytes += int64(len(file.Data))
		seenPaths[strings.ToLower(file.Path)] = file.Path
	}
	if totalBytes > MaxSourceUnpackedBytes {
		return SourceBundle{}, NewAdapterError(adapter, "acquire", ErrorBlocked, fmt.Errorf("source files exceed %d unpacked bytes", MaxSourceUnpackedBytes))
	}

	for _, reference := range artifact.Files {
		name, isDir, pathErr := safeArchivePath(strings.TrimSpace(reference.Path))
		if pathErr != nil || isDir {
			if pathErr == nil {
				pathErr = errors.New("remote file path must name a file")
			}
			return SourceBundle{}, NewAdapterError(adapter, "acquire files", ErrorUnsafe, pathErr)
		}
		if prior, duplicate := seenPaths[strings.ToLower(name)]; duplicate {
			return SourceBundle{}, NewAdapterError(adapter, "acquire files", ErrorUnsafe, fmt.Errorf("duplicate or case-conflicting source paths %q and %q", prior, name))
		}
		resolved, resolveErr := resolveReference(baseURL, reference.URL)
		if resolveErr != nil {
			return SourceBundle{}, NewAdapterError(adapter, "resolve file", ErrorInvalidSource, resolveErr)
		}
		remaining := MaxSourceUnpackedBytes - totalBytes
		if remaining <= 0 {
			return SourceBundle{}, NewAdapterError(adapter, "acquire files", ErrorBlocked, fmt.Errorf("source files exceed %d unpacked bytes", MaxSourceUnpackedBytes))
		}
		result, fetchErr := fetcher.Get(ctx, resolved, FetchOptions{
			Adapter: adapter, Operation: "fetch file", AllowedHosts: allowlist,
			MaxBytes: min(MaxSourceFileBytes, remaining),
		})
		if fetchErr != nil {
			return SourceBundle{}, fetchErr
		}
		if strings.TrimSpace(reference.Digest) != "" {
			if digestErr := verifyDigest(reference.Digest, result.Body); digestErr != nil {
				return SourceBundle{}, NewAdapterError(adapter, "verify file", ErrorIntegrity, digestErr)
			}
			integrityChecks++
		}
		totalBytes += int64(len(result.Body))
		files = append(files, SourceFile{Path: name, Data: result.Body})
		seenPaths[strings.ToLower(name)] = name
		evidence = append(evidence, result.Evidence)
	}

	if len(files) == 0 {
		return SourceBundle{}, NewAdapterError(adapter, "acquire", ErrorInvalidSource, errors.New("artifact contains no files"))
	}
	if strings.TrimSpace(artifact.LicenseURL) != "" {
		if len(files) >= MaxSourceArchiveEntries {
			return SourceBundle{}, NewAdapterError(adapter, "fetch license", ErrorBlocked, fmt.Errorf("source file count exceeds %d", MaxSourceArchiveEntries))
		}
		licenseURL, resolveErr := resolveReference(baseURL, artifact.LicenseURL)
		if resolveErr != nil {
			return SourceBundle{}, NewAdapterError(adapter, "resolve license", ErrorInvalidSource, resolveErr)
		}
		remaining := MaxSourceUnpackedBytes - totalBytes
		if remaining <= 0 {
			return SourceBundle{}, NewAdapterError(adapter, "fetch license", ErrorBlocked, fmt.Errorf("source files exceed %d unpacked bytes", MaxSourceUnpackedBytes))
		}
		result, fetchErr := fetcher.Get(ctx, licenseURL, FetchOptions{
			Adapter: adapter, Operation: "fetch license", AllowedHosts: allowlist,
			MaxBytes: min(MaxSourceFileBytes, remaining), ContentTypes: []string{"text/plain", "text/markdown", "application/octet-stream"},
		})
		if fetchErr != nil {
			return SourceBundle{}, fetchErr
		}
		licensePath := "LICENSE"
		for _, file := range files {
			if strings.EqualFold(file.Path, licensePath) {
				licensePath = "UPSTREAM-LICENSE"
				break
			}
		}
		files = append(files, SourceFile{Path: licensePath, Data: result.Body})
		evidence = append(evidence, result.Evidence)
	}
	if err := validateSourceFiles(files); err != nil {
		return SourceBundle{}, NewAdapterError(adapter, "acquire", ErrorUnsafe, err)
	}

	contentHash := canonicalFilesHash(files)
	if strings.TrimSpace(artifact.Digest) != "" && strings.TrimSpace(artifact.URL) == "" {
		wanted, digestErr := expectedDigest(artifact.Digest)
		if digestErr != nil || wanted != contentHash {
			if digestErr == nil {
				digestErr = fmt.Errorf("file-set digest mismatch: expected %s, got %s", wanted, contentHash)
			}
			return SourceBundle{}, NewAdapterError(adapter, "verify file set", ErrorIntegrity, digestErr)
		}
		integrityChecks++
	}
	canonical := strings.TrimSpace(artifact.CanonicalURL)
	if canonical == "" {
		canonical = strings.TrimSpace(artifact.URL)
		if canonical != "" {
			canonical, _ = resolveReference(baseURL, canonical)
		}
	}
	return SourceBundle{
		Files: files, Revision: strings.TrimSpace(artifact.Revision), UpstreamContentHash: contentHash,
		CanonicalURL: canonical, Evidence: evidence,
		LicenseEvidence: licenseEvidenceFromFiles(files), IntegrityVerified: integrityChecks > 0,
	}, nil
}

func explicitHostsForBase(configured []string, baseURL string) ([]string, error) {
	base, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || base.Scheme != "https" || base.Hostname() == "" || base.User != nil || base.Fragment != "" {
		return nil, errors.New("base URL must be an HTTPS URL without credentials or fragments")
	}
	return hostsForURLs(configured, base.String())
}

func validateSourceFiles(files []SourceFile) error {
	if len(files) == 0 || len(files) > MaxSourceArchiveEntries {
		return fmt.Errorf("source file count must be between 1 and %d", MaxSourceArchiveEntries)
	}
	seen := make(map[string]string, len(files))
	var total int64
	for _, file := range files {
		name, directory, err := safeArchivePath(file.Path)
		if err != nil || directory {
			if err == nil {
				err = errors.New("source path must name a file")
			}
			return err
		}
		folded := strings.ToLower(name)
		if prior, exists := seen[folded]; exists {
			return fmt.Errorf("duplicate or case-conflicting source paths %q and %q", prior, name)
		}
		seen[folded] = name
		if int64(len(file.Data)) > MaxSourceFileBytes {
			return fmt.Errorf("source file %q exceeds %d bytes", name, MaxSourceFileBytes)
		}
		total += int64(len(file.Data))
		if total > MaxSourceUnpackedBytes {
			return fmt.Errorf("source files exceed %d unpacked bytes", MaxSourceUnpackedBytes)
		}
	}
	return nil
}

func withAdapter(err error, adapter, operation string) error {
	var typed *AdapterError
	if errors.As(err, &typed) {
		clone := *typed
		if clone.Adapter == "" {
			clone.Adapter = adapter
		}
		if operation != "" {
			clone.Operation = operation
		}
		return &clone
	}
	return NewAdapterError(adapter, operation, ErrorInvalidSource, err)
}

func dedupeSortedStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func jsonFetchOptions(adapter, operation string, hosts []string, maxBytes int64, headers http.Header) FetchOptions {
	return FetchOptions{
		Adapter: adapter, Operation: operation, AllowedHosts: hosts, MaxBytes: maxBytes,
		ContentTypes: []string{"application/json", "application/manifest+json", "text/json", "text/plain"},
		Headers:      headers,
	}
}
