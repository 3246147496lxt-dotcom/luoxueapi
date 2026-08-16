package skillimport

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
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

// ReadSourceArchive detects one of the archive formats supported by the
// importer. Detection is based on the artifact bytes rather than a server's
// Content-Type, which is frequently generic or incorrect. Unknown formats are
// rejected instead of being interpreted as a different artifact type.
func ReadSourceArchive(raw []byte) ([]SourceFile, error) {
	switch {
	case hasZIPMagic(raw):
		return ReadZIPArtifact(raw)
	case hasGZIPMagic(raw):
		return ReadTarGZArtifact(raw)
	default:
		return nil, NewAdapterError("", "read archive", ErrorInvalidSource, errors.New("artifact is neither a ZIP nor a gzip-compressed TAR archive"))
	}
}

func hasZIPMagic(raw []byte) bool {
	return len(raw) >= 4 && raw[0] == 'P' && raw[1] == 'K' &&
		((raw[2] == 3 && raw[3] == 4) || (raw[2] == 5 && raw[3] == 6) || (raw[2] == 7 && raw[3] == 8))
}

func hasGZIPMagic(raw []byte) bool {
	return len(raw) >= 3 && raw[0] == 0x1f && raw[1] == 0x8b && raw[2] == 8
}

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
	seen := newArchivePathTracker(len(reader.File))
	var declaredTotal int64
	var actualTotal int64
	for _, item := range reader.File {
		name, isDirectory, pathErr := safeArchivePath(item.Name)
		if pathErr != nil {
			return nil, NewAdapterError("", "read zip", ErrorUnsafe, pathErr)
		}
		if trackErr := seen.Add(name, isDirectory); trackErr != nil {
			return nil, NewAdapterError("", "read zip", ErrorUnsafe, trackErr)
		}
		if item.Flags&0x1 != 0 {
			return nil, NewAdapterError("", "read zip", ErrorUnsafe, fmt.Errorf("encrypted ZIP entry %q is not allowed", name))
		}
		mode := item.Mode()
		if mode&os.ModeSymlink != 0 || mode&os.ModeType != 0 && !mode.IsDir() {
			return nil, NewAdapterError("", "read zip", ErrorUnsafe, fmt.Errorf("link or special ZIP entry %q is not allowed", name))
		}
		if item.Method != zip.Store && item.Method != zip.Deflate {
			return nil, NewAdapterError("", "read zip", ErrorUnsafe, fmt.Errorf("unsupported ZIP method for %q", name))
		}
		if isDirectory {
			if item.UncompressedSize64 != 0 {
				return nil, NewAdapterError("", "read zip", ErrorUnsafe, fmt.Errorf("ZIP directory %q contains hidden data", name))
			}
			continue
		}
		if mode.IsDir() {
			return nil, NewAdapterError("", "read zip", ErrorUnsafe, fmt.Errorf("ZIP directory %q must end with a slash", name))
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
	files = stripArchiveWrapper(files)
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

// ReadTarGZArtifact parses a gzip-compressed TAR entirely in memory. It
// accepts regular files and directories only, never follows links, and bounds
// both logical file content and the decoded TAR stream. The latter prevents
// compressed metadata/padding from bypassing the file-size accounting.
func ReadTarGZArtifact(raw []byte) ([]SourceFile, error) {
	if len(raw) == 0 || int64(len(raw)) > MaxSourceArchiveBytes {
		return nil, NewAdapterError("", "read tar.gz", ErrorBlocked, fmt.Errorf("source tar.gz must be between 1 byte and %d bytes", MaxSourceArchiveBytes))
	}
	if !hasGZIPMagic(raw) {
		return nil, NewAdapterError("", "read tar.gz", ErrorInvalidSource, errors.New("artifact is not a gzip-compressed TAR archive"))
	}
	maxDecodedStreamBytes := MaxSourceUnpackedBytes + int64(MaxSourceArchiveEntries)*4096 + 4096
	compressed := bytes.NewReader(raw)
	gzipReader, err := gzip.NewReader(compressed)
	if err != nil {
		return nil, NewAdapterError("", "read tar.gz", ErrorInvalidSource, errors.New("artifact has an invalid gzip stream"))
	}
	// A single artifact must contain exactly one gzip member. Otherwise a
	// second member could be hidden after an apparently valid first archive.
	gzipReader.Multistream(false)
	defer func() { _ = gzipReader.Close() }()
	if err := validatePhysicalTarHeaders(raw, maxDecodedStreamBytes); err != nil {
		return nil, err
	}

	// TAR adds one 512-byte header and up to 511 padding bytes per file. Leave
	// bounded room for those records and common PAX metadata, while preserving
	// the same 16 MiB logical content ceiling used by ZIP sources.
	decoded := &io.LimitedReader{R: gzipReader, N: maxDecodedStreamBytes + 1}
	tarReader := tar.NewReader(decoded)
	files := make([]SourceFile, 0, 16)
	seen := newArchivePathTracker(16)
	var declaredTotal int64
	var actualTotal int64
	entryCount := 0
	for {
		header, nextErr := tarReader.Next()
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			if maxDecodedStreamBytes+1-decoded.N > maxDecodedStreamBytes {
				return nil, NewAdapterError("", "read tar.gz", ErrorBlocked, fmt.Errorf("source tar.gz expands beyond %d bytes", MaxSourceUnpackedBytes))
			}
			return nil, NewAdapterError("", "read tar.gz", ErrorInvalidSource, fmt.Errorf("read TAR header: %w", nextErr))
		}
		entryCount++
		if entryCount > MaxSourceArchiveEntries {
			return nil, NewAdapterError("", "read tar.gz", ErrorBlocked, fmt.Errorf("source tar.gz has more than %d entries", MaxSourceArchiveEntries))
		}
		name, isDirectory, pathErr := safeArchivePath(header.Name)
		if pathErr != nil {
			return nil, NewAdapterError("", "read tar.gz", ErrorUnsafe, pathErr)
		}
		if trackErr := seen.Add(name, isDirectory); trackErr != nil {
			return nil, NewAdapterError("", "read tar.gz", ErrorUnsafe, trackErr)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if !isDirectory || header.Size != 0 || strings.TrimSpace(header.Linkname) != "" {
				return nil, NewAdapterError("", "read tar.gz", ErrorUnsafe, fmt.Errorf("invalid TAR directory entry %q", name))
			}
			continue
		case tar.TypeReg, tar.TypeRegA:
			if isDirectory || strings.TrimSpace(header.Linkname) != "" {
				return nil, NewAdapterError("", "read tar.gz", ErrorUnsafe, fmt.Errorf("invalid regular TAR entry %q", name))
			}
		case tar.TypeSymlink, tar.TypeLink, tar.TypeChar, tar.TypeBlock, tar.TypeFifo:
			return nil, NewAdapterError("", "read tar.gz", ErrorUnsafe, fmt.Errorf("link or special TAR entry %q is not allowed", name))
		default:
			return nil, NewAdapterError("", "read tar.gz", ErrorUnsafe, fmt.Errorf("unsupported TAR entry type %d for %q", header.Typeflag, name))
		}
		if header.Size < 0 || header.Size > MaxSourceFileBytes {
			return nil, NewAdapterError("", "read tar.gz", ErrorBlocked, fmt.Errorf("source tar.gz file %q exceeds %d bytes", name, MaxSourceFileBytes))
		}
		declaredTotal += header.Size
		if declaredTotal > MaxSourceUnpackedBytes {
			return nil, NewAdapterError("", "read tar.gz", ErrorBlocked, fmt.Errorf("source tar.gz expands beyond %d bytes", MaxSourceUnpackedBytes))
		}
		remaining := MaxSourceUnpackedBytes - actualTotal
		if remaining <= 0 && header.Size > 0 {
			return nil, NewAdapterError("", "read tar.gz", ErrorBlocked, fmt.Errorf("source tar.gz expands beyond %d bytes", MaxSourceUnpackedBytes))
		}
		readLimit := min(MaxSourceFileBytes, remaining)
		data, readErr := io.ReadAll(io.LimitReader(tarReader, readLimit+1))
		if readErr != nil {
			return nil, NewAdapterError("", "read tar.gz", ErrorInvalidSource, fmt.Errorf("read %q safely: %w", name, readErr))
		}
		if int64(len(data)) != header.Size {
			if int64(len(data)) > readLimit {
				return nil, NewAdapterError("", "read tar.gz", ErrorBlocked, fmt.Errorf("source tar.gz expands beyond %d bytes", MaxSourceUnpackedBytes))
			}
			return nil, NewAdapterError("", "read tar.gz", ErrorInvalidSource, fmt.Errorf("TAR file %q has a truncated body", name))
		}
		actualTotal += int64(len(data))
		files = append(files, SourceFile{Path: name, Data: data, Executable: header.FileInfo().Mode().Perm()&0o111 != 0})
	}
	if entryCount == 0 || len(files) == 0 {
		return nil, NewAdapterError("", "read tar.gz", ErrorInvalidSource, errors.New("source tar.gz contains no files"))
	}
	// Drain through the gzip checksum and include bytes after TAR's logical EOF
	// in the decoded-stream bound. A valid archive has no second gzip member or
	// non-gzip trailer.
	trailer := zeroCheckingWriter{}
	if _, err := io.Copy(&trailer, decoded); err != nil {
		return nil, NewAdapterError("", "read tar.gz", ErrorInvalidSource, fmt.Errorf("verify gzip stream: %w", err))
	}
	decodedBytes := maxDecodedStreamBytes + 1 - decoded.N
	if decodedBytes > maxDecodedStreamBytes {
		return nil, NewAdapterError("", "read tar.gz", ErrorBlocked, fmt.Errorf("source tar.gz expands beyond %d bytes", MaxSourceUnpackedBytes))
	}
	if err := gzipReader.Close(); err != nil {
		return nil, NewAdapterError("", "read tar.gz", ErrorInvalidSource, errors.New("artifact has an invalid gzip checksum"))
	}
	if compressed.Len() != 0 {
		return nil, NewAdapterError("", "read tar.gz", ErrorUnsafe, errors.New("source tar.gz contains trailing data or multiple gzip members"))
	}
	if trailer.nonZero {
		return nil, NewAdapterError("", "read tar.gz", ErrorUnsafe, errors.New("source tar.gz contains non-padding data after the TAR end marker"))
	}
	if actualTotal > 1024*1024 && actualTotal > int64(len(raw))*service.SkillArchiveMaxCompressionRatio {
		return nil, NewAdapterError("", "read tar.gz", ErrorBlocked, errors.New("source tar.gz compression ratio exceeds safety limit"))
	}
	files = stripArchiveWrapper(files)
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

// validatePhysicalTarHeaders counts raw TAR headers before archive/tar can
// consume PAX and GNU extension records internally. It also bounds metadata
// body sizes and rejects special records not needed for portable skill paths.
func validatePhysicalTarHeaders(raw []byte, maxDecodedStreamBytes int64) error {
	gzipReader, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return NewAdapterError("", "read tar.gz", ErrorInvalidSource, errors.New("artifact has an invalid gzip stream"))
	}
	gzipReader.Multistream(false)
	defer func() { _ = gzipReader.Close() }()
	decoded := &io.LimitedReader{R: gzipReader, N: maxDecodedStreamBytes + 1}
	var block [512]byte
	physicalEntries := 0
	var logicalBodyBytes int64
	for {
		if _, err := io.ReadFull(decoded, block[:]); err != nil {
			if decoded.N == 0 {
				return NewAdapterError("", "read tar.gz", ErrorBlocked, fmt.Errorf("source tar.gz expands beyond %d bytes", MaxSourceUnpackedBytes))
			}
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return NewAdapterError("", "read tar.gz", ErrorInvalidSource, errors.New("TAR stream is missing its end marker"))
			}
			return NewAdapterError("", "read tar.gz", ErrorInvalidSource, fmt.Errorf("read physical TAR header: %w", err))
		}
		if isZeroTarBlock(block[:]) {
			if _, err := io.ReadFull(decoded, block[:]); err != nil || !isZeroTarBlock(block[:]) {
				return NewAdapterError("", "read tar.gz", ErrorInvalidSource, errors.New("TAR stream has an invalid end marker"))
			}
			return nil
		}
		physicalEntries++
		if physicalEntries > MaxSourceArchiveEntries {
			return NewAdapterError("", "read tar.gz", ErrorBlocked, fmt.Errorf("source tar.gz has more than %d physical entries", MaxSourceArchiveEntries))
		}
		typeFlag := block[156]
		size, parseErr := parseTarOctal(block[124:136])
		if parseErr != nil || size < 0 {
			return NewAdapterError("", "read tar.gz", ErrorInvalidSource, errors.New("TAR entry has an invalid size"))
		}
		metadata := typeFlag == tar.TypeXHeader || typeFlag == tar.TypeXGlobalHeader || typeFlag == tar.TypeGNULongName || typeFlag == tar.TypeGNULongLink
		if !metadata && size > MaxSourceFileBytes {
			return NewAdapterError("", "read tar.gz", ErrorBlocked, fmt.Errorf("source tar.gz physical entry exceeds %d bytes", MaxSourceFileBytes))
		}
		if metadata && size > service.SkillArchiveMaxPathBytes*4 {
			return NewAdapterError("", "read tar.gz", ErrorBlocked, errors.New("TAR path metadata exceeds the safety limit"))
		}
		if !metadata {
			logicalBodyBytes += size
			if logicalBodyBytes > MaxSourceUnpackedBytes {
				return NewAdapterError("", "read tar.gz", ErrorBlocked, fmt.Errorf("source tar.gz expands beyond %d bytes", MaxSourceUnpackedBytes))
			}
		}
		padding := (512 - size%512) % 512
		if size+padding > decoded.N {
			return NewAdapterError("", "read tar.gz", ErrorBlocked, fmt.Errorf("source tar.gz expands beyond %d bytes", MaxSourceUnpackedBytes))
		}
		if _, err := io.CopyN(io.Discard, decoded, size+padding); err != nil {
			return NewAdapterError("", "read tar.gz", ErrorInvalidSource, errors.New("TAR entry body is truncated"))
		}
	}
}

func isZeroTarBlock(block []byte) bool {
	for _, value := range block {
		if value != 0 {
			return false
		}
	}
	return true
}

func parseTarOctal(field []byte) (int64, error) {
	if len(field) > 0 && field[0]&0x80 != 0 {
		return 0, errors.New("base-256 TAR sizes are not supported")
	}
	value := int64(0)
	seenDigit := false
	for _, character := range field {
		switch {
		case character == 0 || character == ' ':
			continue
		case character >= '0' && character <= '7':
			seenDigit = true
			if value > (MaxSourceUnpackedBytes-int64(character-'0'))/8 {
				return 0, errors.New("TAR size overflows the source limit")
			}
			value = value*8 + int64(character-'0')
		default:
			return 0, errors.New("TAR size is not octal")
		}
	}
	if !seenDigit {
		return 0, nil
	}
	return value, nil
}

func safeArchivePath(raw string) (string, bool, error) {
	if raw == "" || !utf8.ValidString(raw) || strings.ContainsRune(raw, 0) || strings.Contains(raw, "\\") {
		return "", false, fmt.Errorf("unsafe archive path %q", raw)
	}
	isDirectory := strings.HasSuffix(raw, "/")
	name := strings.TrimSuffix(raw, "/")
	if name == "" || strings.HasPrefix(name, "/") || path.IsAbs(name) || isWindowsArchivePath(name) || path.Clean(name) != name || norm.NFC.String(name) != name {
		return "", false, fmt.Errorf("unsafe archive path %q", raw)
	}
	if len(name) > service.SkillArchiveMaxPathBytes {
		return "", false, fmt.Errorf("archive path %q exceeds %d bytes", raw, service.SkillArchiveMaxPathBytes)
	}
	parts := strings.Split(name, "/")
	if len(parts) > service.SkillArchiveMaxDepth {
		return "", false, fmt.Errorf("archive path %q exceeds depth limit", raw)
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", false, fmt.Errorf("archive path traversal %q is not allowed", raw)
		}
	}
	return name, isDirectory, nil
}

func isWindowsArchivePath(name string) bool {
	return len(name) >= 2 && ((name[0] >= 'a' && name[0] <= 'z') || (name[0] >= 'A' && name[0] <= 'Z')) && name[1] == ':'
}

type archivePathNode struct {
	name      string
	directory bool
	explicit  bool
}

type zeroCheckingWriter struct {
	nonZero bool
}

func (w *zeroCheckingWriter) Write(data []byte) (int, error) {
	for _, value := range data {
		if value != 0 {
			w.nonZero = true
			break
		}
	}
	return len(data), nil
}

type archivePathTracker struct {
	nodes map[string]archivePathNode
}

func newArchivePathTracker(capacity int) *archivePathTracker {
	return &archivePathTracker{nodes: make(map[string]archivePathNode, capacity*2)}
}

// Add records explicit entries as well as their implicit parent directories.
// This catches collisions such as Foo/a + foo/b and file + file/child, which
// are unsafe even though the complete entry names are not identical.
func (t *archivePathTracker) Add(name string, directory bool) error {
	parts := strings.Split(name, "/")
	for index := range parts {
		prefix := strings.Join(parts[:index+1], "/")
		folded := strings.ToLower(norm.NFC.String(prefix))
		last := index == len(parts)-1
		wantedDirectory := !last || directory
		existing, exists := t.nodes[folded]
		if exists {
			if existing.name != prefix {
				return fmt.Errorf("duplicate or case-conflicting archive paths %q and %q", existing.name, prefix)
			}
			if existing.directory != wantedDirectory {
				return fmt.Errorf("archive path %q is both a file and a directory", prefix)
			}
			if last && existing.explicit {
				return fmt.Errorf("duplicate or case-conflicting archive paths %q and %q", existing.name, prefix)
			}
			if last {
				existing.explicit = true
				t.nodes[folded] = existing
			}
			continue
		}
		t.nodes[folded] = archivePathNode{name: prefix, directory: wantedDirectory, explicit: last}
	}
	return nil
}

func stripArchiveWrapper(files []SourceFile) []SourceFile {
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
