package service

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
	"gopkg.in/yaml.v3"
)

const (
	SkillArchiveMaxBytes            int64 = 5 * 1024 * 1024
	SkillArchiveMaxUnpackedBytes    int64 = 5 * 1024 * 1024
	SkillArchiveMaxFileBytes        int64 = 1 * 1024 * 1024
	SkillArchiveMaxFiles                  = 100
	SkillArchiveMaxEntries                = 200
	SkillArchiveMaxDepth                  = 16
	SkillArchiveMaxPathBytes              = 512
	SkillArchiveMaxSkillMDBytes     int64 = 256 * 1024
	SkillArchiveMaxCompressionRatio       = 200
)

type ValidatedSkillArchive struct {
	PackageData         []byte
	SHA256              string
	UnpackedSize        int64
	FileManifest        []SkillArchiveFile
	ValidationReport    SkillValidationReport
	ManifestName        string
	ManifestDescription string
	SkillMD             string
}

type validatedArchiveContent struct {
	name       string
	data       []byte
	executable bool
}

var windowsAbsolutePathPattern = regexp.MustCompile(`^[A-Za-z]:`)

var forbiddenBinaryExtensions = map[string]struct{}{
	".a": {}, ".bin": {}, ".class": {}, ".dll": {}, ".dylib": {}, ".exe": {},
	".jar": {}, ".o": {}, ".so": {}, ".wasm": {},
}

var forbiddenNestedArchiveExtensions = map[string]struct{}{
	".7z": {}, ".bz2": {}, ".gz": {}, ".rar": {}, ".tar": {}, ".tgz": {}, ".xz": {}, ".zip": {},
}

var scriptExtensions = map[string]struct{}{
	".bash": {}, ".js": {}, ".mjs": {}, ".php": {}, ".ps1": {}, ".py": {},
	".rb": {}, ".sh": {}, ".ts": {}, ".zsh": {},
}

func ValidateSkillArchive(raw []byte, expectedName string) (*ValidatedSkillArchive, error) {
	if len(raw) == 0 || int64(len(raw)) > SkillArchiveMaxBytes {
		return nil, invalidSkillArchive("ARCHIVE_SIZE", "ZIP must be between 1 byte and 5 MiB", "")
	}
	reader, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return nil, invalidSkillArchive("INVALID_ZIP", "file is not a readable ZIP archive", "")
	}
	if len(reader.File) == 0 || len(reader.File) > SkillArchiveMaxEntries {
		return nil, invalidSkillArchive("FILE_COUNT", "ZIP contains too many entries", "")
	}

	seenExplicit := make(map[string]struct{}, len(reader.File))
	seenFolded := make(map[string]string, len(reader.File))
	canonicalCase := make(map[string]string, len(reader.File)*2)
	nodeKinds := make(map[string]bool, len(reader.File)*2) // true is directory
	contents := make([]validatedArchiveContent, 0, len(reader.File))
	manifest := make([]SkillArchiveFile, 0, len(reader.File))
	warnings := make([]SkillValidationIssue, 0)
	var unpackedSize int64
	var skillMD []byte

	for _, file := range reader.File {
		name, isDir, issue := validateSkillArchivePath(file.Name)
		if issue != nil {
			return nil, invalidSkillArchive(issue.Code, issue.Message, issue.Path)
		}
		parts := strings.Split(name, "/")
		if parts[0] != expectedName || (!isDir && len(parts) < 2) {
			return nil, invalidSkillArchive("TOP_LEVEL_DIRECTORY", "ZIP must have exactly one top-level directory matching the skill slug", name)
		}
		foldedName := strings.ToLower(norm.NFC.String(name))
		if _, exists := seenExplicit[name]; exists {
			return nil, invalidSkillArchive("DUPLICATE_PATH", "ZIP contains a duplicate path", name)
		}
		seenExplicit[name] = struct{}{}
		if existing, exists := seenFolded[foldedName]; exists && existing != name {
			return nil, invalidSkillArchive("PATH_CASE_CONFLICT", "ZIP paths conflict after Unicode and case normalization", name)
		}
		seenFolded[foldedName] = name

		components := parts
		for i := range components {
			prefix := strings.Join(components[:i+1], "/")
			foldedPrefix := strings.ToLower(norm.NFC.String(prefix))
			if existing, exists := canonicalCase[foldedPrefix]; exists && existing != prefix {
				return nil, invalidSkillArchive("PATH_CASE_CONFLICT", "ZIP paths conflict when compared case-insensitively", name)
			}
			canonicalCase[foldedPrefix] = prefix
			if i < len(components)-1 {
				if existingIsDir, exists := nodeKinds[foldedPrefix]; exists && !existingIsDir {
					return nil, invalidSkillArchive("PATH_TYPE_CONFLICT", "a file is also used as a parent directory", name)
				}
				nodeKinds[foldedPrefix] = true
			}
		}
		if existingIsDir, exists := nodeKinds[foldedName]; exists && existingIsDir != isDir {
			return nil, invalidSkillArchive("PATH_TYPE_CONFLICT", "ZIP path is both a file and a directory", name)
		}
		nodeKinds[foldedName] = isDir

		if file.Flags&0x1 != 0 {
			return nil, invalidSkillArchive("ENCRYPTED_ENTRY", "encrypted ZIP entries are not allowed", name)
		}
		mode := file.Mode()
		if mode&os.ModeSymlink != 0 {
			return nil, invalidSkillArchive("SYMLINK", "symbolic links are not allowed", name)
		}
		if !isDir && !mode.IsRegular() {
			return nil, invalidSkillArchive("SPECIAL_FILE", "only regular files are allowed", name)
		}
		if isDir {
			continue
		}
		if file.Method != zip.Store && file.Method != zip.Deflate {
			return nil, invalidSkillArchive("ZIP_METHOD", "unsupported ZIP compression method", name)
		}
		if file.UncompressedSize64 > uint64(SkillArchiveMaxFileBytes) {
			return nil, invalidSkillArchive("FILE_SIZE", "an archive file exceeds 1 MiB", name)
		}
		if issue := forbiddenSkillArchiveFile(name); issue != nil {
			return nil, invalidSkillArchive(issue.Code, issue.Message, issue.Path)
		}

		entry, openErr := file.Open()
		if openErr != nil {
			return nil, invalidSkillArchive("ENTRY_READ", "cannot read ZIP entry", name)
		}
		data, readErr := io.ReadAll(io.LimitReader(entry, SkillArchiveMaxFileBytes+1))
		var crcProbe [1]byte
		probeN, probeErr := entry.Read(crcProbe[:])
		closeErr := entry.Close()
		if readErr != nil || closeErr != nil || int64(len(data)) > SkillArchiveMaxFileBytes || probeN != 0 || probeErr != io.EOF {
			return nil, invalidSkillArchive("ENTRY_READ", "cannot safely read ZIP entry", name)
		}
		if !utf8.Valid(data) || bytes.IndexByte(data, 0) >= 0 {
			return nil, invalidSkillArchive("BINARY_FILE", "binary files are not allowed in MVP skill packages", name)
		}
		if containsPrivateKey(data) {
			return nil, invalidSkillArchive("PRIVATE_KEY", "private key material is not allowed", name)
		}

		unpackedSize += int64(len(data))
		if unpackedSize > SkillArchiveMaxUnpackedBytes {
			return nil, invalidSkillArchive("UNPACKED_SIZE", "ZIP expands beyond the 5 MiB limit", name)
		}
		digest := sha256.Sum256(data)
		manifest = append(manifest, SkillArchiveFile{Path: name, ByteSize: int64(len(data)), SHA256: hex.EncodeToString(digest[:])})
		contents = append(contents, validatedArchiveContent{name: name, data: data, executable: mode.Perm()&0o111 != 0})
		warnings = append(warnings, skillArchiveWarnings(name, data, mode)...)
		if name == expectedName+"/SKILL.md" {
			if int64(len(data)) > SkillArchiveMaxSkillMDBytes {
				return nil, invalidSkillArchive("SKILL_MD_SIZE", "SKILL.md exceeds 256 KiB", name)
			}
			skillMD = data
		}
	}

	if len(contents) == 0 || len(contents) > SkillArchiveMaxFiles {
		return nil, invalidSkillArchive("FILE_COUNT", "ZIP must contain between 1 and 100 regular files", "")
	}
	if len(skillMD) == 0 {
		return nil, invalidSkillArchive("SKILL_MD_MISSING", "SKILL.md is required directly inside the skill top-level directory", expectedName+"/SKILL.md")
	}
	if unpackedSize > 1024*1024 && unpackedSize > int64(len(raw))*SkillArchiveMaxCompressionRatio {
		return nil, invalidSkillArchive("COMPRESSION_RATIO", "ZIP compression ratio exceeds the safety limit", "")
	}

	manifestName, manifestDescription, skillBody, err := parseSkillManifest(skillMD)
	if err != nil {
		return nil, err
	}
	if manifestName != expectedName {
		return nil, invalidSkillArchive("MANIFEST_NAME_MISMATCH", "SKILL.md name must match the marketplace slug", "SKILL.md")
	}

	sort.Slice(contents, func(i, j int) bool { return contents[i].name < contents[j].name })
	sort.Slice(manifest, func(i, j int) bool { return manifest[i].Path < manifest[j].Path })
	sort.SliceStable(warnings, func(i, j int) bool {
		if warnings[i].Path == warnings[j].Path {
			return warnings[i].Code < warnings[j].Code
		}
		return warnings[i].Path < warnings[j].Path
	})
	packageData, err := buildDeterministicSkillZIP(contents)
	if err != nil {
		return nil, invalidSkillArchive("ZIP_REBUILD", "cannot normalize ZIP archive", "")
	}
	if int64(len(packageData)) > SkillArchiveMaxBytes {
		return nil, invalidSkillArchive("ARCHIVE_SIZE", "normalized ZIP exceeds the 5 MiB storage limit", "")
	}
	packageDigest := sha256.Sum256(packageData)
	report := SkillValidationReport{Valid: true, Errors: []SkillValidationIssue{}, Warnings: warnings}
	return &ValidatedSkillArchive{
		PackageData: packageData, SHA256: hex.EncodeToString(packageDigest[:]),
		UnpackedSize: unpackedSize, FileManifest: manifest, ValidationReport: report,
		ManifestName: manifestName, ManifestDescription: manifestDescription, SkillMD: skillBody,
	}, nil
}

func validateSkillArchivePath(raw string) (string, bool, *SkillValidationIssue) {
	if raw == "" || !utf8.ValidString(raw) || strings.ContainsRune(raw, 0) || strings.Contains(raw, "\\") {
		return "", false, &SkillValidationIssue{Code: "UNSAFE_PATH", Message: "ZIP path is empty or malformed", Path: raw}
	}
	isDir := strings.HasSuffix(raw, "/")
	name := strings.TrimSuffix(raw, "/")
	if norm.NFC.String(name) != name {
		return "", false, &SkillValidationIssue{Code: "UNICODE_NORMALIZATION", Message: "ZIP paths must use Unicode NFC normalization", Path: raw}
	}
	if name == "" || strings.HasPrefix(name, "/") || path.IsAbs(name) || windowsAbsolutePathPattern.MatchString(name) || path.Clean(name) != name {
		return "", false, &SkillValidationIssue{Code: "UNSAFE_PATH", Message: "absolute, relative, and non-canonical paths are not allowed", Path: raw}
	}
	if len(name) > SkillArchiveMaxPathBytes {
		return "", false, &SkillValidationIssue{Code: "PATH_LENGTH", Message: "ZIP path exceeds 512 bytes", Path: raw}
	}
	parts := strings.Split(name, "/")
	if len(parts) > SkillArchiveMaxDepth {
		return "", false, &SkillValidationIssue{Code: "PATH_DEPTH", Message: "ZIP path exceeds 16 components", Path: raw}
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", false, &SkillValidationIssue{Code: "PATH_TRAVERSAL", Message: "path traversal is not allowed", Path: raw}
		}
		lower := strings.ToLower(part)
		if lower == ".git" {
			return "", false, &SkillValidationIssue{Code: "VCS_METADATA", Message: ".git content is not allowed", Path: raw}
		}
		if lower == "__macosx" {
			return "", false, &SkillValidationIssue{Code: "OS_METADATA", Message: "__MACOSX metadata is not allowed", Path: raw}
		}
		if strings.HasPrefix(part, "._") {
			return "", false, &SkillValidationIssue{Code: "APPLEDOUBLE", Message: "AppleDouble metadata is not allowed", Path: raw}
		}
	}
	return name, isDir, nil
}

func forbiddenSkillArchiveFile(name string) *SkillValidationIssue {
	base := path.Base(name)
	lowerBase := strings.ToLower(base)
	ext := strings.ToLower(path.Ext(lowerBase))
	switch {
	case lowerBase == ".ds_store":
		return &SkillValidationIssue{Code: "OS_METADATA", Message: ".DS_Store is not allowed", Path: name}
	case lowerBase == ".env" || strings.HasPrefix(lowerBase, ".env."):
		return &SkillValidationIssue{Code: "ENV_FILE", Message: "environment files are not allowed", Path: name}
	case lowerBase == "id_rsa" || lowerBase == "id_dsa" || lowerBase == "id_ecdsa" || lowerBase == "id_ed25519":
		return &SkillValidationIssue{Code: "PRIVATE_KEY", Message: "private key files are not allowed", Path: name}
	}
	if _, found := forbiddenNestedArchiveExtensions[ext]; found {
		return &SkillValidationIssue{Code: "NESTED_ARCHIVE", Message: "nested archives are not allowed", Path: name}
	}
	if _, found := forbiddenBinaryExtensions[ext]; found {
		return &SkillValidationIssue{Code: "BINARY_FILE", Message: "binary files are not allowed", Path: name}
	}
	if ext == ".pem" || ext == ".key" || ext == ".p12" || ext == ".pfx" || ext == ".jks" || ext == ".ppk" {
		return &SkillValidationIssue{Code: "PRIVATE_KEY", Message: "key container files are not allowed", Path: name}
	}
	return nil
}

func containsPrivateKey(data []byte) bool {
	upper := bytes.ToUpper(data)
	return bytes.Contains(upper, []byte("-----BEGIN PRIVATE KEY-----")) ||
		bytes.Contains(upper, []byte("-----BEGIN ENCRYPTED PRIVATE KEY-----")) ||
		bytes.Contains(upper, []byte("-----BEGIN RSA PRIVATE KEY-----")) ||
		bytes.Contains(upper, []byte("-----BEGIN EC PRIVATE KEY-----")) ||
		bytes.Contains(upper, []byte("-----BEGIN DSA PRIVATE KEY-----")) ||
		bytes.Contains(upper, []byte("-----BEGIN OPENSSH PRIVATE KEY-----")) ||
		bytes.Contains(upper, []byte("-----BEGIN PGP PRIVATE KEY BLOCK-----")) ||
		bytes.Contains(upper, []byte("PUTTY-USER-KEY-FILE-"))
}

func skillArchiveWarnings(name string, data []byte, mode os.FileMode) []SkillValidationIssue {
	warnings := make([]SkillValidationIssue, 0, 3)
	ext := strings.ToLower(path.Ext(name))
	if _, found := scriptExtensions[ext]; found || mode.Perm()&0o111 != 0 {
		warnings = append(warnings, SkillValidationIssue{Code: "EXECUTABLE_CONTENT", Message: "package contains executable or script content; installation never executes it automatically", Path: name})
	}
	lower := bytes.ToLower(data)
	if bytes.Contains(lower, []byte("http://")) || bytes.Contains(lower, []byte("https://")) ||
		bytes.Contains(lower, []byte("curl ")) || bytes.Contains(lower, []byte("wget ")) {
		warnings = append(warnings, SkillValidationIssue{Code: "NETWORK_REFERENCE", Message: "file references network access or remote content", Path: name})
	}
	if bytes.Contains(data, []byte("$HOME")) || bytes.Contains(data, []byte("${HOME}")) ||
		bytes.Contains(lower, []byte("process.env")) || bytes.Contains(lower, []byte("os.getenv")) ||
		bytes.Contains(lower, []byte(".ssh/")) || bytes.Contains(data, []byte("CODEX_HOME")) {
		warnings = append(warnings, SkillValidationIssue{Code: "ENVIRONMENT_ACCESS", Message: "file references environment variables or user-local paths", Path: name})
	}
	return warnings
}

func parseSkillManifest(skillMD []byte) (string, string, string, error) {
	text := strings.ReplaceAll(string(skillMD), "\r\n", "\n")
	if !strings.HasPrefix(text, "---\n") {
		return "", "", "", invalidSkillArchive("FRONTMATTER_MISSING", "SKILL.md must begin with YAML frontmatter", "SKILL.md")
	}
	end := strings.Index(text[4:], "\n---\n")
	if end < 0 {
		return "", "", "", invalidSkillArchive("FRONTMATTER_INVALID", "SKILL.md frontmatter is not terminated", "SKILL.md")
	}
	frontmatter := text[4 : 4+end]
	var manifest struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}
	if err := yaml.Unmarshal([]byte(frontmatter), &manifest); err != nil {
		return "", "", "", invalidSkillArchive("FRONTMATTER_INVALID", "SKILL.md frontmatter is not valid YAML", "SKILL.md")
	}
	manifest.Name = strings.TrimSpace(manifest.Name)
	manifest.Description = strings.TrimSpace(manifest.Description)
	if !skillSlugPattern.MatchString(manifest.Name) || len(manifest.Name) > 64 {
		return "", "", "", invalidSkillArchive("MANIFEST_NAME", "frontmatter name must be lowercase kebab-case and at most 64 bytes", "SKILL.md")
	}
	if manifest.Description == "" || utf8.RuneCountInString(manifest.Description) > 1024 {
		return "", "", "", invalidSkillArchive("MANIFEST_DESCRIPTION", "frontmatter description is required and limited to 1024 characters", "SKILL.md")
	}
	bodyStart := 4 + end + len("\n---\n")
	body := strings.TrimSpace(text[bodyStart:])
	if body == "" {
		return "", "", "", invalidSkillArchive("SKILL_BODY_MISSING", "SKILL.md must contain instructions after frontmatter", "SKILL.md")
	}
	return manifest.Name, manifest.Description, body, nil
}

func buildDeterministicSkillZIP(contents []validatedArchiveContent) ([]byte, error) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	fixedTime := time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)
	for _, content := range contents {
		header := &zip.FileHeader{Name: content.name, Method: zip.Deflate}
		header.SetModTime(fixedTime)
		if content.executable {
			header.SetMode(0o755)
		} else {
			header.SetMode(0o644)
		}
		entry, err := writer.CreateHeader(header)
		if err != nil {
			_ = writer.Close()
			return nil, err
		}
		if _, err := entry.Write(content.data); err != nil {
			_ = writer.Close()
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func invalidSkillArchive(code, message, filePath string) error {
	report := SkillValidationReport{
		Valid:    false,
		Errors:   []SkillValidationIssue{{Code: code, Message: message, Path: filePath}},
		Warnings: []SkillValidationIssue{},
	}
	return skillArchiveError(report)
}

func archiveDebugString(result *ValidatedSkillArchive) string {
	if result == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%s (%d files, %d bytes)", result.ManifestName, len(result.FileManifest), result.UnpackedSize)
}
