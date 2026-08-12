package skillimport

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"
)

var (
	sha256DigestPattern = regexp.MustCompile(`^sha256:([a-f0-9]{64})$`)
	githubOwnerPattern  = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,37}[A-Za-z0-9])?$`)
	githubRepoPattern   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,99}$`)
)

func decodeConfig(raw json.RawMessage, target any) error {
	if len(bytes.TrimSpace(raw)) == 0 {
		raw = json.RawMessage(`{}`)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("configuration must contain one JSON object")
	}
	return nil
}

func encodeCursor(value any) string {
	raw, _ := json.Marshal(value)
	return base64.RawURLEncoding.EncodeToString(raw)
}

func decodeCursor(raw string, target any) error {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return errors.New("invalid cursor encoding")
	}
	if err := json.Unmarshal(decoded, target); err != nil {
		return errors.New("invalid cursor payload")
	}
	return nil
}

func canonicalFilesHash(files []SourceFile) string {
	ordered := append([]SourceFile(nil), files...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Path < ordered[j].Path })
	hasher := sha256.New()
	for _, file := range ordered {
		_, _ = io.WriteString(hasher, file.Path)
		_, _ = hasher.Write([]byte{0})
		if file.Executable {
			_, _ = hasher.Write([]byte{1})
		} else {
			_, _ = hasher.Write([]byte{0})
		}
		_, _ = hasher.Write(file.Data)
		_, _ = hasher.Write([]byte{0})
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

func licenseEvidenceFromFiles(files []SourceFile) []LicenseEvidence {
	result := make([]LicenseEvidence, 0)
	seen := make(map[string]struct{})
	for _, file := range files {
		base := strings.ToLower(path.Base(file.Path))
		stem := strings.TrimSuffix(base, path.Ext(base))
		if stem != "license" && stem != "licence" && stem != "copying" && stem != "notice" {
			continue
		}
		if _, exists := seen[file.Path]; exists {
			continue
		}
		seen[file.Path] = struct{}{}
		result = append(result, LicenseEvidence{Kind: "bundled_file", Path: file.Path, SHA256: digestBytes(file.Data)})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result
}

func mergeLicenseEvidence(groups ...[]LicenseEvidence) []LicenseEvidence {
	seen := make(map[string]struct{})
	result := make([]LicenseEvidence, 0)
	for _, group := range groups {
		for _, evidence := range group {
			key := evidence.Kind + "\x00" + evidence.Path + "\x00" + evidence.URL + "\x00" + evidence.SHA256
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			result = append(result, evidence)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		left := result[i].Kind + "\x00" + result[i].Path + "\x00" + result[i].URL
		right := result[j].Kind + "\x00" + result[j].Path + "\x00" + result[j].URL
		return left < right
	})
	return result
}

func resolveReference(baseURL, reference string) (string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	ref, err := url.Parse(strings.TrimSpace(reference))
	if err != nil {
		return "", err
	}
	if ref.User != nil || ref.Fragment != "" {
		return "", errors.New("artifact URL userinfo and fragments are not allowed")
	}
	return base.ResolveReference(ref).String(), nil
}

func hostsForURLs(configured []string, values ...string) ([]string, error) {
	set := make(map[string]struct{})
	for _, configuredHost := range configured {
		host := strings.ToLower(strings.TrimSpace(configuredHost))
		if host == "" || strings.ContainsAny(host, "/?#@") {
			return nil, fmt.Errorf("invalid allowed host %q", configuredHost)
		}
		set[host] = struct{}{}
	}
	for _, value := range values {
		parsed, err := url.Parse(value)
		if err != nil || parsed.Hostname() == "" {
			return nil, fmt.Errorf("invalid URL %q", value)
		}
		set[strings.ToLower(parsed.Hostname())] = struct{}{}
	}
	result := make([]string, 0, len(set))
	for host := range set {
		result = append(result, host)
	}
	sort.Strings(result)
	return result, nil
}

func expectedDigest(raw string) (string, error) {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return "", nil
	}
	match := sha256DigestPattern.FindStringSubmatch(raw)
	if len(match) != 2 {
		return "", errors.New("digest must use sha256:<64 lowercase hex> format")
	}
	return match[1], nil
}

func verifyDigest(raw string, data []byte) error {
	wanted, err := expectedDigest(raw)
	if err != nil {
		return err
	}
	if wanted == "" {
		return nil
	}
	if actual := digestBytes(data); actual != wanted {
		return fmt.Errorf("artifact digest mismatch: expected %s, got %s", wanted, actual)
	}
	return nil
}

func parseGitHubRepository(raw string) (string, string, error) {
	value := strings.TrimSpace(raw)
	value = strings.TrimPrefix(value, "https://github.com/")
	value = strings.TrimSuffix(value, ".git")
	value = strings.Trim(value, "/")
	parts := strings.Split(value, "/")
	if len(parts) != 2 || !githubOwnerPattern.MatchString(parts[0]) || !githubRepoPattern.MatchString(parts[1]) {
		return "", "", errors.New("repository must be a public GitHub owner/repository")
	}
	return parts[0], parts[1], nil
}

func safeOpaque(value any) json.RawMessage {
	raw, _ := json.Marshal(value)
	return raw
}

func intPointer(value int) *int { return &value }
