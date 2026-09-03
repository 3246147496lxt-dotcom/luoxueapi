package main

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"golang.org/x/text/unicode/norm"
	"gopkg.in/yaml.v3"
)

const filteredPayloadBudget = service.SkillArchiveMaxUnpackedBytes - 64*1024

var excludedBinaryExtensions = map[string]string{
	".a": "binary extension", ".bin": "binary extension", ".class": "binary extension",
	".dll": "binary extension", ".dylib": "binary extension", ".exe": "binary extension",
	".jar": "binary extension", ".o": "binary extension", ".so": "binary extension", ".wasm": "binary extension",
	".7z": "nested archive", ".bz2": "nested archive", ".gz": "nested archive", ".rar": "nested archive",
	".tar": "nested archive", ".tgz": "nested archive", ".xz": "nested archive", ".zip": "nested archive",
	".pem": "key container", ".key": "key container", ".p12": "key container", ".pfx": "key container",
	".jks": "key container", ".ppk": "key container",
}

type packageFile struct {
	Path       string
	Data       []byte
	Executable bool
}

type packageBuildResult struct {
	Validated   *service.ValidatedSkillArchive
	Description string
	Excluded    []excludedFile
}

type provenance struct {
	Source         string
	OriginalSlug   string
	OriginalName   string
	MarketSlug     string
	SnapshotSHA256 string
	DownloadHash   string
	Rank           int
}

func buildValidatedPackage(root string, origin provenance) (*packageBuildResult, error) {
	candidates, excluded, err := collectPackageCandidates(root)
	if err != nil {
		return nil, err
	}
	skillIndex := -1
	for index := range candidates {
		if candidates[index].Path == "SKILL.md" {
			skillIndex = index
			break
		}
	}
	if skillIndex < 0 {
		return nil, errors.New("source package has no root SKILL.md")
	}
	rewritten, description, err := rewriteSkillManifest(candidates[skillIndex].Data, origin)
	if err != nil {
		return nil, fmt.Errorf("rewrite SKILL.md: %w", err)
	}
	candidates[skillIndex].Data = rewritten
	if int64(len(rewritten)) > service.SkillArchiveMaxSkillMDBytes {
		return nil, fmt.Errorf("rewritten SKILL.md exceeds %d bytes", service.SkillArchiveMaxSkillMDBytes)
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Path == "SKILL.md" {
			return true
		}
		if candidates[j].Path == "SKILL.md" {
			return false
		}
		return candidates[i].Path < candidates[j].Path
	})
	selected := make([]packageFile, 0, min(len(candidates), service.SkillArchiveMaxFiles))
	var total int64
	for _, candidate := range candidates {
		size := int64(len(candidate.Data))
		switch {
		case len(selected) >= service.SkillArchiveMaxFiles:
			excluded = append(excluded, excludedFile{Path: candidate.Path, Reason: "package file-count limit (100)", Bytes: size})
		case total+size > filteredPayloadBudget:
			excluded = append(excluded, excludedFile{Path: candidate.Path, Reason: "package unpacked-size budget", Bytes: size})
		default:
			selected = append(selected, candidate)
			total += size
		}
	}
	if len(selected) == 0 || selected[0].Path != "SKILL.md" {
		return nil, errors.New("SKILL.md cannot fit inside marketplace package limits")
	}

	var validated *service.ValidatedSkillArchive
	for {
		rawZIP, buildErr := buildInputZIP(origin.MarketSlug, selected)
		if buildErr != nil {
			return nil, buildErr
		}
		validated, err = service.ValidateSkillArchive(rawZIP, origin.MarketSlug)
		if err == nil {
			break
		}
		removeIndex, reason := validationRecoveryFile(err, origin.MarketSlug, selected)
		if removeIndex < 1 || removeIndex >= len(selected) {
			return nil, fmt.Errorf("ValidateSkillArchive: %w", err)
		}
		removed := selected[removeIndex]
		excluded = append(excluded, excludedFile{Path: removed.Path, Reason: reason, Bytes: int64(len(removed.Data))})
		selected = append(selected[:removeIndex], selected[removeIndex+1:]...)
	}
	sort.Slice(excluded, func(i, j int) bool { return excluded[i].Path < excluded[j].Path })
	return &packageBuildResult{Validated: validated, Description: description, Excluded: excluded}, nil
}

func collectPackageCandidates(root string) ([]packageFile, []excludedFile, error) {
	var candidates []packageFile
	var excluded []excludedFile
	err := filepath.WalkDir(root, func(filePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if filePath == root {
			return nil
		}
		relative, err := filepath.Rel(root, filePath)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if entry.IsDir() {
			if excludedDirectory(relative) {
				excluded = append(excluded, excludedFile{Path: relative + "/", Reason: "repository or operating-system metadata directory"})
				return fs.SkipDir
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			excluded = append(excluded, excludedFile{Path: relative, Reason: "symlink or special file", Bytes: info.Size()})
			return nil
		}
		if reason := excludedPathReason(relative); reason != "" {
			excluded = append(excluded, excludedFile{Path: relative, Reason: reason, Bytes: info.Size()})
			return nil
		}
		if info.Size() > service.SkillArchiveMaxFileBytes {
			excluded = append(excluded, excludedFile{Path: relative, Reason: "file exceeds 1 MiB", Bytes: info.Size()})
			return nil
		}
		if len(relative)+65 > service.SkillArchiveMaxPathBytes || strings.Count(relative, "/")+2 > service.SkillArchiveMaxDepth {
			excluded = append(excluded, excludedFile{Path: relative, Reason: "archive path length or depth limit", Bytes: info.Size()})
			return nil
		}
		file, err := os.Open(filePath)
		if err != nil {
			return err
		}
		data, readErr := io.ReadAll(io.LimitReader(file, service.SkillArchiveMaxFileBytes+1))
		closeErr := file.Close()
		if readErr != nil || closeErr != nil {
			return fmt.Errorf("read %s: %v %v", relative, readErr, closeErr)
		}
		if int64(len(data)) > service.SkillArchiveMaxFileBytes {
			excluded = append(excluded, excludedFile{Path: relative, Reason: "file exceeds 1 MiB", Bytes: int64(len(data))})
			return nil
		}
		if !utf8.Valid(data) || bytes.IndexByte(data, 0) >= 0 {
			excluded = append(excluded, excludedFile{Path: relative, Reason: "binary or non-UTF-8 content", Bytes: int64(len(data))})
			return nil
		}
		if containsPrivateKeyMaterial(data) {
			excluded = append(excluded, excludedFile{Path: relative, Reason: "private-key material", Bytes: int64(len(data))})
			return nil
		}
		candidates = append(candidates, packageFile{Path: relative, Data: data, Executable: info.Mode().Perm()&0o111 != 0})
		return nil
	})
	if err != nil {
		return nil, excluded, err
	}
	return candidates, excluded, nil
}

func excludedDirectory(relative string) bool {
	for _, component := range strings.Split(strings.ToLower(relative), "/") {
		if component == ".git" || component == "__macosx" || component == ".svn" || component == ".hg" {
			return true
		}
	}
	return false
}

func excludedPathReason(relative string) string {
	if strings.Contains(relative, "\\") || norm.NFC.String(relative) != relative {
		return "non-canonical or non-NFC archive path"
	}
	base := strings.ToLower(filepath.Base(relative))
	if base == ".skills-sh-download.json" || base == "skills-lock.json" {
		return "acquisition metadata"
	}
	if base == ".ds_store" || strings.HasPrefix(base, "._") {
		return "operating-system metadata"
	}
	if base == ".env" || strings.HasPrefix(base, ".env.") {
		return "environment file"
	}
	if base == "id_rsa" || base == "id_dsa" || base == "id_ecdsa" || base == "id_ed25519" {
		return "private-key filename"
	}
	if reason := excludedBinaryExtensions[strings.ToLower(filepath.Ext(base))]; reason != "" {
		return reason
	}
	for _, component := range strings.Split(relative, "/") {
		if component == "" || component == "." || component == ".." {
			return "unsafe path"
		}
	}
	return ""
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

func rewriteSkillManifest(raw []byte, origin provenance) ([]byte, string, error) {
	text := strings.ReplaceAll(string(raw), "\r\n", "\n")
	var frontmatter string
	body := text
	if strings.HasPrefix(text, "---\n") {
		if end := strings.Index(text[4:], "\n---\n"); end >= 0 {
			frontmatter = text[4 : 4+end]
			body = strings.TrimSpace(text[4+end+len("\n---\n"):])
		}
	}
	if strings.TrimSpace(body) == "" {
		body = fmt.Sprintf("# %s\n\nThis package is a skills.sh snapshot imported from `%s`.", origin.OriginalName, origin.Source)
	}

	root := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	if frontmatter != "" {
		var document yaml.Node
		if err := yaml.Unmarshal([]byte(frontmatter), &document); err == nil && len(document.Content) > 0 && document.Content[0].Kind == yaml.MappingNode {
			root = document.Content[0]
		}
	}
	setYAMLScalar(root, "name", origin.MarketSlug)
	description := yamlScalar(root, "description")
	if strings.TrimSpace(description) == "" {
		description = fmt.Sprintf("Imported skill %s from %s.", origin.OriginalSlug, origin.Source)
	}
	description = truncateRunes(strings.TrimSpace(description), 1024)
	setYAMLScalar(root, "description", description)
	setYAMLMapping(root, "skills_sh_import", map[string]string{
		"source":          origin.Source,
		"skill_id":        origin.OriginalSlug,
		"original_name":   origin.OriginalName,
		"market_slug":     origin.MarketSlug,
		"snapshot_sha256": origin.SnapshotSHA256,
		"download_hash":   origin.DownloadHash,
		"rank":            fmt.Sprint(origin.Rank),
	})
	frontmatterBytes, err := yaml.Marshal(root)
	if err != nil {
		return nil, "", err
	}
	rewritten := append([]byte("---\n"), frontmatterBytes...)
	rewritten = append(rewritten, []byte("---\n\n"+strings.TrimSpace(body)+"\n")...)
	return rewritten, description, nil
}

func validationRecoveryFile(err error, slug string, files []packageFile) (int, string) {
	reason := "removed during archive validation recovery"
	var validationErr *service.SkillArchiveValidationError
	if errors.As(err, &validationErr) && len(validationErr.Report.Errors) > 0 {
		issue := validationErr.Report.Errors[0]
		reason = fmt.Sprintf("archive validation recovery: %s: %s", issue.Code, issue.Message)
		relative := strings.TrimPrefix(issue.Path, slug+"/")
		if relative != "" && relative != "SKILL.md" {
			for index := range files {
				if files[index].Path == relative {
					return index, reason
				}
			}
		}
	}
	if len(files) > 1 {
		return len(files) - 1, reason
	}
	return -1, reason
}

func yamlScalar(mapping *yaml.Node, key string) string {
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		if mapping.Content[index].Value == key {
			return mapping.Content[index+1].Value
		}
	}
	return ""
}

func setYAMLScalar(mapping *yaml.Node, key, value string) {
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

func setYAMLMapping(mapping *yaml.Node, key string, values map[string]string) {
	node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	keys := make([]string, 0, len(values))
	for itemKey := range values {
		if values[itemKey] != "" {
			keys = append(keys, itemKey)
		}
	}
	sort.Strings(keys)
	for _, itemKey := range keys {
		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: itemKey},
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: values[itemKey]},
		)
	}
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		if mapping.Content[index].Value == key {
			mapping.Content[index+1] = node
			return
		}
	}
	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, node,
	)
}

func buildInputZIP(slug string, files []packageFile) ([]byte, error) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	fixedTime := time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)
	for _, file := range files {
		header := &zip.FileHeader{Name: slug + "/" + file.Path, Method: zip.Deflate, Modified: fixedTime}
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

func truncateRunes(value string, maximum int) string {
	if utf8.RuneCountInString(value) <= maximum {
		return value
	}
	runes := []rune(value)
	return strings.TrimSpace(string(runes[:maximum]))
}
