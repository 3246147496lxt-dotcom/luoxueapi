package main

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestBuildValidatedPackageFiltersAndNormalizes(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "SKILL.md", []byte("---\nname: react:components\ndescription: Build reusable React components.\n---\n\n# Instructions\n\nUse semantic HTML.\n"), 0o644)
	writeTestFile(t, root, "LICENSE", []byte("Example license text.\n"), 0o644)
	writeTestFile(t, root, "scripts/check.sh", []byte("#!/bin/sh\necho ok\n"), 0o755)
	writeTestFile(t, root, "blob.bin", []byte("binary-looking extension"), 0o644)
	writeTestFile(t, root, "nested.zip", []byte("not a zip"), 0o644)
	writeTestFile(t, root, ".DS_Store", []byte("metadata"), 0o644)
	writeTestFile(t, root, "secret.txt", []byte("-----BEGIN PRIVATE KEY-----\nsecret"), 0o644)

	built, err := buildValidatedPackage(root, provenance{
		Source: "example/skills", OriginalSlug: "react:components", OriginalName: "React Components",
		MarketSlug: "react-components", SnapshotSHA256: strings.Repeat("a", 64), DownloadHash: "download-hash", Rank: 445,
	})
	if err != nil {
		t.Fatal(err)
	}
	if built.Validated.ManifestName != "react-components" {
		t.Fatalf("manifest name = %q", built.Validated.ManifestName)
	}
	if built.Description != "Build reusable React components." {
		t.Fatalf("description = %q", built.Description)
	}
	if len(built.Excluded) != 4 {
		t.Fatalf("excluded = %#v, want 4 files", built.Excluded)
	}
	revalidated, err := service.ValidateSkillArchive(built.Validated.PackageData, "react-components")
	if err != nil {
		t.Fatalf("normalized package did not revalidate: %v", err)
	}
	if revalidated.SHA256 != built.Validated.SHA256 {
		t.Fatalf("normalized package is not deterministic")
	}

	reader, err := zip.NewReader(bytes.NewReader(built.Validated.PackageData), int64(len(built.Validated.PackageData)))
	if err != nil {
		t.Fatal(err)
	}
	var skillMD string
	foundLicense := false
	for _, file := range reader.File {
		if strings.HasSuffix(file.Name, ".bin") || strings.HasSuffix(file.Name, ".zip") || strings.HasSuffix(file.Name, ".DS_Store") || strings.HasSuffix(file.Name, "secret.txt") {
			t.Fatalf("excluded file leaked into package: %s", file.Name)
		}
		if file.Name == "react-components/SKILL.md" {
			stream, openErr := file.Open()
			if openErr != nil {
				t.Fatal(openErr)
			}
			raw, readErr := io.ReadAll(stream)
			_ = stream.Close()
			if readErr != nil {
				t.Fatal(readErr)
			}
			skillMD = string(raw)
		}
		if file.Name == "react-components/LICENSE" {
			foundLicense = true
		}
	}
	if !foundLicense {
		t.Fatal("LICENSE was not preserved")
	}
	if !strings.Contains(skillMD, "name: react-components") || !strings.Contains(skillMD, "skills_sh_import:") || !strings.Contains(skillMD, "skill_id: react:components") {
		t.Fatalf("rewritten provenance missing from SKILL.md:\n%s", skillMD)
	}
}

func TestBuildValidatedPackageRebuildsInvalidFrontmatter(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "SKILL.md", []byte("---\nname: [invalid\ndescription: broken\n---\n\n# Keep this body\n"), 0o644)
	built, err := buildValidatedPackage(root, provenance{
		Source: "example/skills", OriginalSlug: "broken", OriginalName: "Broken",
		MarketSlug: "broken", SnapshotSHA256: strings.Repeat("b", 64), Rank: 9,
	})
	if err != nil {
		t.Fatal(err)
	}
	if built.Validated.ManifestName != "broken" || built.Validated.ManifestDescription == "" {
		t.Fatalf("frontmatter was not rebuilt: name=%q description=%q", built.Validated.ManifestName, built.Validated.ManifestDescription)
	}
	if !strings.Contains(built.Validated.SkillMD, "# Keep this body") {
		t.Fatalf("instruction body was not preserved: %s", built.Validated.SkillMD)
	}
}

func writeTestFile(t *testing.T, root, relative string, data []byte, mode os.FileMode) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, mode); err != nil {
		t.Fatal(err)
	}
}
