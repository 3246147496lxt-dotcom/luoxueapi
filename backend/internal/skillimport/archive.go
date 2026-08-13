package skillimport

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"golang.org/x/text/unicode/norm"
)

const (
	MaxSourceArchiveBytes   int64 = 8 * 1024 * 1024
	MaxSourceUnpackedBytes  int64 = 16 * 1024 * 1024
	MaxSourceFileBytes      int64 = 2 * 1024 * 1024
	MaxSourceArchiveEntries       = 500
)

// ReadZIPArtifact parses an upstream ZIP without writing it to disk. It never
// follows links or executes entries. Unsafe structure blocks the whole source;
// the normalizer may later omit only inert metadata and blocks any truncation
// that could remove functional source content.
func ReadZIPArtifact(raw []byte) ([]SourceFile, error) {
	if len(raw) == 0 || int64(len(raw)) > MaxSourceArchiveBytes {
		return nil, NewAdapterError("", "read zip", ErrorBlocked, fmt.Errorf("source ZIP must be between 1 byte and %d bytes", MaxSourceArchiveBytes))
	}
	reader, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return nil, NewAdapterError("", "read zip", ErrorInvalidSource, errors.New("artifact is not a readable ZIP"))
	}
	if len(reader.File) == 0 || len(reader.File) > MaxSourceArchiveEntries {
		return nil, NewAdapterError("", "read zip", ErrorBlocked, fmt.Errorf("source ZIP entry count must be between 1 and %d", MaxSourceArchiveEntries))
	}
	files := make([]SourceFile, 0, len(reader.File))
	seen := make(map[string]string, len(reader.File))
	var declaredTotal int64
	var actualTotal int64
	for _, item := range reader.File {
		name, isDirectory, pathErr := safeArchivePath(item.Name)
		if pathErr != nil {
			return nil, NewAdapterError("", "read zip", ErrorUnsafe, pathErr)
		}
		folded := strings.ToLower(norm.NFC.String(name))
		if prior, exists := seen[folded]; exists {
			return nil, NewAdapterError("", "read zip", ErrorUnsafe, fmt.Errorf("duplicate or case-conflicting ZIP paths %q and %q", prior, name))
		}
		seen[folded] = name
		if item.Flags&0x1 != 0 {
			return nil, NewAdapterError("", "read zip", ErrorUnsafe, fmt.Errorf("encrypted ZIP entry %q is not allowed", name))
		}
		mode := item.Mode()
		if mode&os.ModeSymlink != 0 || mode&os.ModeType != 0 && !mode.IsDir() {
			return nil, NewAdapterError("", "read zip", ErrorUnsafe, fmt.Errorf("link or special ZIP entry %q is not allowed", name))
		}
		if isDirectory {
			continue
		}
		if item.Method != zip.Store && item.Method != zip.Deflate {
			return nil, NewAdapterError("", "read zip", ErrorUnsafe, fmt.Errorf("unsupported ZIP method for %q", name))
		}
		if item.UncompressedSize64 > uint64(MaxSourceFileBytes) {
			return nil, NewAdapterError("", "read zip", ErrorBlocked, fmt.Errorf("source ZIP file %q exceeds %d bytes", name, MaxSourceFileBytes))
		}
		declaredTotal += int64(item.UncompressedSize64)
		if declaredTotal > MaxSourceUnpackedBytes {
			return nil, NewAdapterError("", "read zip", ErrorBlocked, fmt.Errorf("source ZIP expands beyond %d bytes", MaxSourceUnpackedBytes))
		}
		remaining := MaxSourceUnpackedBytes - actualTotal
		if remaining <= 0 {
			return nil, NewAdapterError("", "read zip", ErrorBlocked, fmt.Errorf("source ZIP expands beyond %d bytes", MaxSourceUnpackedBytes))
		}
		readLimit := min(MaxSourceFileBytes, remaining)
		stream, err := item.Open()
		if err != nil {
			return nil, NewAdapterError("", "read zip", ErrorInvalidSource, fmt.Errorf("open %q: %w", name, err))
		}
		data, readErr := io.ReadAll(io.LimitReader(stream, readLimit+1))
		closeErr := stream.Close()
		if readErr != nil || closeErr != nil {
			return nil, NewAdapterError("", "read zip", ErrorInvalidSource, fmt.Errorf("read %q safely", name))
		}
		if int64(len(data)) > MaxSourceFileBytes || int64(len(data)) > remaining {
			return nil, NewAdapterError("", "read zip", ErrorBlocked, fmt.Errorf("source ZIP expands beyond %d bytes", MaxSourceUnpackedBytes))
		}
		actualTotal += int64(len(data))
		files = append(files, SourceFile{Path: name, Data: data, Executable: mode.Perm()&0o111 != 0})
	}
	if actualTotal > 1024*1024 && actualTotal > int64(len(raw))*service.SkillArchiveMaxCompressionRatio {
		return nil, NewAdapterError("", "read zip", ErrorBlocked, errors.New("source ZIP compression ratio exceeds safety limit"))
	}
	files = stripZIPWrapper(files)
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func safeArchivePath(raw string) (string, bool, error) {
	if raw == "" || !utf8.ValidString(raw) || strings.ContainsRune(raw, 0) || strings.Contains(raw, "\\") {
		return "", false, fmt.Errorf("unsafe ZIP path %q", raw)
	}
	isDirectory := strings.HasSuffix(raw, "/")
	name := strings.TrimSuffix(raw, "/")
	if name == "" || strings.HasPrefix(name, "/") || path.IsAbs(name) || path.Clean(name) != name || norm.NFC.String(name) != name {
		return "", false, fmt.Errorf("unsafe ZIP path %q", raw)
	}
	if len(name) > service.SkillArchiveMaxPathBytes {
		return "", false, fmt.Errorf("ZIP path %q exceeds %d bytes", raw, service.SkillArchiveMaxPathBytes)
	}
	parts := strings.Split(name, "/")
	if len(parts) > service.SkillArchiveMaxDepth {
		return "", false, fmt.Errorf("ZIP path %q exceeds depth limit", raw)
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", false, fmt.Errorf("ZIP path traversal %q is not allowed", raw)
		}
	}
	return name, isDirectory, nil
}

func stripZIPWrapper(files []SourceFile) []SourceFile {
	rootSkill := false
	common := ""
	for index, file := range files {
		if file.Path == "SKILL.md" {
			rootSkill = true
		}
		parts := strings.Split(file.Path, "/")
		if len(parts) < 2 {
			common = ""
			continue
		}
		if index == 0 {
			common = parts[0]
		} else if common != parts[0] {
			common = ""
		}
	}
	if rootSkill || common == "" {
		return files
	}
	hasWrappedSkill := false
	for _, file := range files {
		if file.Path == common+"/SKILL.md" {
			hasWrappedSkill = true
			break
		}
	}
	if !hasWrappedSkill {
		return files
	}
	result := make([]SourceFile, len(files))
	for index, file := range files {
		file.Path = strings.TrimPrefix(file.Path, common+"/")
		result[index] = file
	}
	return result
}
