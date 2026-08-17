package skillimport

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type tarFixtureEntry struct {
	Header tar.Header
	Data   []byte
}

func makeSourceTarGZ(t *testing.T, entries []tarFixtureEntry) []byte {
	t.Helper()
	var compressed bytes.Buffer
	gzipWriter := gzip.NewWriter(&compressed)
	tarWriter := tar.NewWriter(gzipWriter)
	for _, entry := range entries {
		header := entry.Header
		if header.Mode == 0 {
			header.Mode = 0o644
		}
		if header.Typeflag == 0 {
			header.Typeflag = tar.TypeReg
		}
		if header.Typeflag == tar.TypeReg {
			header.Size = int64(len(entry.Data))
		}
		if err := tarWriter.WriteHeader(&header); err != nil {
			t.Fatalf("write TAR header %q: %v", header.Name, err)
		}
		if len(entry.Data) > 0 {
			if _, err := tarWriter.Write(entry.Data); err != nil {
				t.Fatalf("write TAR body %q: %v", header.Name, err)
			}
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	return compressed.Bytes()
}

func TestReadSourceArchiveReadsTarGZFixture(t *testing.T) {
	raw := makeSourceTarGZ(t, []tarFixtureEntry{
		{Header: tar.Header{Name: "SKILL.md"}, Data: []byte("---\nname: build-tam\ndescription: Build a TAM\n---\n")},
		{Header: tar.Header{Name: "skill-metadata.json"}, Data: []byte(`{"install_surface":"v1"}`)},
	})
	files, err := ReadSourceArchive(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 || files[0].Path != "SKILL.md" || files[1].Path != "skill-metadata.json" {
		t.Fatalf("tar.gz files = %#v", files)
	}
}

func TestReadSourceArchiveKeepsZIPSupport(t *testing.T) {
	raw := makeSourceZIP(map[string][]byte{
		"wrapper/SKILL.md": []byte("---\nname: zip-fixture\ndescription: ZIP fixture\n---\n"),
		"wrapper/LICENSE":  []byte("MIT\n"),
	})
	files, err := ReadSourceArchive(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 || files[0].Path != "LICENSE" || files[1].Path != "SKILL.md" {
		t.Fatalf("ZIP files = %#v", files)
	}
}

func TestReadSourceArchiveRejectsUnknownFormat(t *testing.T) {
	_, err := ReadSourceArchive([]byte("not an archive"))
	if ErrorKindOf(err) != ErrorInvalidSource || !strings.Contains(err.Error(), "neither a ZIP") {
		t.Fatalf("unknown archive error = %v", err)
	}
}

func TestReadTarGZArtifactRejectsUnsafePathsAndEntryTypes(t *testing.T) {
	tests := map[string]tarFixtureEntry{
		"traversal": {Header: tar.Header{Name: "../SKILL.md"}, Data: []byte("bad")},
		"absolute":  {Header: tar.Header{Name: "/SKILL.md"}, Data: []byte("bad")},
		"windows":   {Header: tar.Header{Name: "C:/SKILL.md"}, Data: []byte("bad")},
		"backslash": {Header: tar.Header{Name: `dir\SKILL.md`}, Data: []byte("bad")},
		"symlink":   {Header: tar.Header{Name: "SKILL.md", Typeflag: tar.TypeSymlink, Linkname: "target"}},
		"hardlink":  {Header: tar.Header{Name: "SKILL.md", Typeflag: tar.TypeLink, Linkname: "target"}},
		"device":    {Header: tar.Header{Name: "device", Typeflag: tar.TypeChar}},
		"fifo":      {Header: tar.Header{Name: "pipe", Typeflag: tar.TypeFifo}},
	}
	for name, entry := range tests {
		t.Run(name, func(t *testing.T) {
			raw := makeSourceTarGZ(t, []tarFixtureEntry{entry})
			_, err := ReadTarGZArtifact(raw)
			if ErrorKindOf(err) != ErrorUnsafe {
				t.Fatalf("unsafe TAR error = %v", err)
			}
		})
	}
}

func TestReadTarGZArtifactRejectsDuplicateAndCaseConflictingPaths(t *testing.T) {
	for name, paths := range map[string][]string{
		"duplicate":            {"SKILL.md", "SKILL.md"},
		"case conflict":        {"SKILL.md", "skill.md"},
		"parent case conflict": {"Foo/one.md", "foo/two.md"},
		"file parent conflict": {"references", "references/one.md"},
	} {
		t.Run(name, func(t *testing.T) {
			raw := makeSourceTarGZ(t, []tarFixtureEntry{
				{Header: tar.Header{Name: paths[0]}, Data: []byte("first")},
				{Header: tar.Header{Name: paths[1]}, Data: []byte("second")},
			})
			_, err := ReadTarGZArtifact(raw)
			if ErrorKindOf(err) != ErrorUnsafe || (!strings.Contains(err.Error(), "case-conflicting") && !strings.Contains(err.Error(), "file and a directory")) {
				t.Fatalf("duplicate TAR error = %v", err)
			}
		})
	}
}

func TestReadTarGZArtifactRejectsCorruptAndConcatenatedGzipStreams(t *testing.T) {
	raw := makeSourceTarGZ(t, []tarFixtureEntry{{
		Header: tar.Header{Name: "SKILL.md"}, Data: []byte("---\nname: gzip\ndescription: Gzip\n---\n"),
	}})
	corrupt := append([]byte(nil), raw...)
	corrupt[len(corrupt)-1] ^= 0xff
	if _, err := ReadTarGZArtifact(corrupt); ErrorKindOf(err) != ErrorInvalidSource {
		t.Fatalf("corrupt gzip error = %v", err)
	}
	concatenated := append(append([]byte(nil), raw...), raw...)
	if _, err := ReadTarGZArtifact(concatenated); ErrorKindOf(err) != ErrorUnsafe || !strings.Contains(err.Error(), "multiple gzip members") {
		t.Fatalf("concatenated gzip error = %v", err)
	}
	gzipReader, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	tarBytes, err := io.ReadAll(gzipReader)
	if err != nil {
		t.Fatal(err)
	}
	if err := gzipReader.Close(); err != nil {
		t.Fatal(err)
	}
	tarBytes = append(tarBytes, []byte("hidden second payload")...)
	var repacked bytes.Buffer
	gzipWriter := gzip.NewWriter(&repacked)
	if _, err := gzipWriter.Write(tarBytes); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadTarGZArtifact(repacked.Bytes()); ErrorKindOf(err) != ErrorUnsafe || !strings.Contains(err.Error(), "non-padding data") {
		t.Fatalf("hidden TAR trailer error = %v", err)
	}
}

func TestReadTarGZArtifactHandlesPAXAndGNULongNameMetadataSafely(t *testing.T) {
	for name, format := range map[string]tar.Format{"pax": tar.FormatPAX, "gnu": tar.FormatGNU} {
		t.Run(name, func(t *testing.T) {
			header := tar.Header{Name: "SKILL.md", Format: format}
			if format == tar.FormatPAX {
				header.PAXRecords = map[string]string{"comment": "metadata"}
			} else {
				header.Name = strings.Repeat("a", 120) + "/SKILL.md"
			}
			raw := makeSourceTarGZ(t, []tarFixtureEntry{{Header: header, Data: []byte("skill")}})
			files, err := ReadTarGZArtifact(raw)
			wantPath := header.Name
			if format == tar.FormatGNU {
				wantPath = "SKILL.md" // the deterministic single wrapper is stripped
			}
			if err != nil || len(files) != 1 || files[0].Path != wantPath {
				t.Fatalf("extended-header files = %#v err=%v", files, err)
			}
		})
	}
	t.Run("effective path limit", func(t *testing.T) {
		header := tar.Header{Name: strings.Repeat("a", service.SkillArchiveMaxPathBytes) + "/SKILL.md", Format: tar.FormatPAX}
		_, err := ReadTarGZArtifact(makeSourceTarGZ(t, []tarFixtureEntry{{Header: header, Data: []byte("skill")}}))
		if ErrorKindOf(err) != ErrorUnsafe || !strings.Contains(err.Error(), "exceeds 512 bytes") {
			t.Fatalf("PAX effective path error = %v", err)
		}
	})
}

func TestReadTarGZArtifactRejectsInvalidDirectoryMetadata(t *testing.T) {
	for name, header := range map[string]tar.Header{
		"linkname": {Name: "references/", Typeflag: tar.TypeDir, Linkname: "target"},
		"no slash": {Name: "references", Typeflag: tar.TypeDir},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := ReadTarGZArtifact(makeSourceTarGZ(t, []tarFixtureEntry{{Header: header}}))
			if ErrorKindOf(err) != ErrorUnsafe || !strings.Contains(err.Error(), "directory") {
				t.Fatalf("invalid directory error = %v", err)
			}
		})
	}
}

func TestReadZIPArtifactRejectsDirectoryDataAndModeMismatch(t *testing.T) {
	t.Run("directory mode without slash", func(t *testing.T) {
		var buffer bytes.Buffer
		writer := zip.NewWriter(&buffer)
		header := &zip.FileHeader{Name: "references", Method: zip.Store}
		header.SetMode(os.ModeDir | 0o755)
		if _, err := writer.CreateHeader(header); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		_, err := ReadZIPArtifact(buffer.Bytes())
		if ErrorKindOf(err) != ErrorUnsafe || !strings.Contains(err.Error(), "must end with a slash") {
			t.Fatalf("ZIP directory-mode error = %v", err)
		}
	})
}

func TestReadTarGZArtifactEnforcesEntryFileTotalAndCompressionLimits(t *testing.T) {
	t.Run("entry count", func(t *testing.T) {
		entries := make([]tarFixtureEntry, 0, MaxSourceArchiveEntries+1)
		for index := 0; index <= MaxSourceArchiveEntries; index++ {
			entries = append(entries, tarFixtureEntry{Header: tar.Header{Name: fmt.Sprintf("%03d.txt", index)}, Data: []byte("x")})
		}
		_, err := ReadTarGZArtifact(makeSourceTarGZ(t, entries))
		if ErrorKindOf(err) != ErrorBlocked || !strings.Contains(err.Error(), "more than 500") {
			t.Fatalf("entry-count error = %v", err)
		}
	})

	t.Run("per file", func(t *testing.T) {
		raw := makeSourceTarGZ(t, []tarFixtureEntry{{
			Header: tar.Header{Name: "too-large.bin"}, Data: bytes.Repeat([]byte("x"), int(MaxSourceFileBytes)+1),
		}})
		_, err := ReadTarGZArtifact(raw)
		if ErrorKindOf(err) != ErrorBlocked || !strings.Contains(err.Error(), "exceeds 2097152 bytes") {
			t.Fatalf("per-file error = %v", err)
		}
	})

	t.Run("total unpacked", func(t *testing.T) {
		entries := make([]tarFixtureEntry, 0, 9)
		chunk := bytes.Repeat([]byte("abcdefgh"), int(MaxSourceFileBytes)/8)
		for index := 0; index < 9; index++ {
			entries = append(entries, tarFixtureEntry{Header: tar.Header{Name: fmt.Sprintf("%d.bin", index)}, Data: chunk})
		}
		_, err := ReadTarGZArtifact(makeSourceTarGZ(t, entries))
		if ErrorKindOf(err) != ErrorBlocked || !strings.Contains(err.Error(), "expands beyond") {
			t.Fatalf("unpacked-total error = %v", err)
		}
	})

	t.Run("physical prescan total", func(t *testing.T) {
		entries := make([]tarFixtureEntry, 0, 9)
		for index := 0; index < 9; index++ {
			entries = append(entries, tarFixtureEntry{Header: tar.Header{Name: fmt.Sprintf("zeros-%d.bin", index)}, Data: make([]byte, MaxSourceFileBytes)})
		}
		_, err := ReadTarGZArtifact(makeSourceTarGZ(t, entries))
		if ErrorKindOf(err) != ErrorBlocked || !strings.Contains(err.Error(), "expands beyond") {
			t.Fatalf("physical-prescan error = %v", err)
		}
	})

	t.Run("compression bomb", func(t *testing.T) {
		raw := makeSourceTarGZ(t, []tarFixtureEntry{{
			Header: tar.Header{Name: "zeros.bin"}, Data: make([]byte, MaxSourceFileBytes),
		}})
		_, err := ReadTarGZArtifact(raw)
		if ErrorKindOf(err) != ErrorBlocked || !strings.Contains(err.Error(), "compression ratio") {
			t.Fatalf("compression-ratio error = %v (compressed=%d)", err, len(raw))
		}
	})
}
