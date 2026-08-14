package skillimport

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestReadZIPArtifactRejectsTraversalAndBomb(t *testing.T) {
	traversal := makeSourceZIP(map[string][]byte{
		"../SKILL.md": []byte("bad"),
	})
	_, err := ReadZIPArtifact(traversal)
	if ErrorKindOf(err) != ErrorUnsafe {
		t.Fatalf("zip traversal kind = %q, want unsafe (err=%v)", ErrorKindOf(err), err)
	}

	bomb := makeSourceZIP(map[string][]byte{
		"SKILL.md": bytes.Repeat([]byte("a"), 2*1024*1024),
	})
	_, err = ReadZIPArtifact(bomb)
	if ErrorKindOf(err) != ErrorBlocked {
		t.Fatalf("zip bomb kind = %q, want blocked (err=%v)", ErrorKindOf(err), err)
	}
}

func TestReadZIPArtifactStripsSingleWrapperDeterministically(t *testing.T) {
	raw := makeSourceZIP(map[string][]byte{
		"repo-snapshot/SKILL.md":  []byte("skill"),
		"repo-snapshot/README.md": []byte("readme"),
	})
	files, err := ReadZIPArtifact(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 || files[0].Path != "README.md" || files[1].Path != "SKILL.md" {
		t.Fatalf("wrapper output = %#v", files)
	}
}

func TestSafeArchivePathEnforcesExistingPathByteLimit(t *testing.T) {
	withinLimit := strings.Repeat("a", service.SkillArchiveMaxPathBytes)
	if name, directory, err := safeArchivePath(withinLimit); err != nil || directory || name != withinLimit {
		t.Fatalf("path at limit: name=%q directory=%v err=%v", name, directory, err)
	}
	overLimit := withinLimit + "b"
	if _, _, err := safeArchivePath(overLimit); err == nil || !strings.Contains(err.Error(), "exceeds 512 bytes") {
		t.Fatalf("path over limit error = %v", err)
	}
}

func TestNormalizerIsDeterministicAndKeepsSeparateHashes(t *testing.T) {
	skill := SourceFile{Path: "SKILL.md", Data: []byte("---\nname: upstream-name\ndescription: Original upstream description\n---\n\n# Instructions\n\nDo the work.\n")}
	license := SourceFile{Path: "LICENSE", Data: []byte("Permission granted.\n")}
	readme := SourceFile{Path: "references/readme.md", Data: []byte("Reference\n")}
	firstFiles := []SourceFile{readme, skill, license}
	secondFiles := []SourceFile{license, readme, skill}
	firstBundle := SourceBundle{Files: firstFiles, UpstreamContentHash: canonicalFilesHash(firstFiles), IntegrityVerified: true}
	secondBundle := SourceBundle{Files: secondFiles, UpstreamContentHash: canonicalFilesHash(secondFiles), IntegrityVerified: true}
	inputSkill := DiscoveredSkill{AdapterType: "fixture", SuggestedName: "Original name", SuggestedSlug: "upstream-name"}

	first, err := NormalizeSkill(NormalizeInput{Skill: inputSkill, Bundle: firstBundle, MarketSlug: "market-name"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := NormalizeSkill(NormalizeInput{Skill: inputSkill, Bundle: secondBundle, MarketSlug: "market-name"})
	if err != nil {
		t.Fatal(err)
	}
	if first.Blocked || second.Blocked || first.NeedsReview || second.NeedsReview {
		t.Fatalf("unexpected policy result: first=%#v second=%#v", first, second)
	}
	if first.NormalizedSHA256 == "" || first.NormalizedSHA256 != second.NormalizedSHA256 || !bytes.Equal(first.PackageData, second.PackageData) {
		t.Fatalf("normalization is not deterministic: %q vs %q", first.NormalizedSHA256, second.NormalizedSHA256)
	}
	if first.UpstreamContentHash == "" || first.UpstreamContentHash != second.UpstreamContentHash {
		t.Fatalf("upstream hashes differ: %q vs %q", first.UpstreamContentHash, second.UpstreamContentHash)
	}
	if first.Description != "Original upstream description" || first.LicenseUnverified || len(first.LicenseEvidence) != 1 {
		t.Fatalf("metadata was not preserved: %#v", first)
	}
	if !first.Transformed {
		t.Fatal("market slug rewrite must be recorded as transformed")
	}
}

func TestNormalizerBlocksFunctionalFilesInsteadOfPublishingTruncatedPackage(t *testing.T) {
	files := []SourceFile{
		{Path: "SKILL.md", Data: []byte("---\nname: demo\ndescription: Demo skill\n---\n\n# Demo\n")},
		{Path: ".env", Data: []byte("SECRET=value\n")},
		{Path: "image.png", Data: []byte{0, 1, 2, 3}},
	}
	bundle := SourceBundle{Files: files, UpstreamContentHash: canonicalFilesHash(files), IntegrityVerified: true}
	result, err := NormalizeSkill(NormalizeInput{
		Skill: DiscoveredSkill{AdapterType: "fixture", SuggestedName: "Demo"}, Bundle: bundle, MarketSlug: "demo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Blocked || len(result.ExcludedFiles) != 2 || len(result.PackageData) != 0 {
		t.Fatalf("unexpected exclusion result: %#v", result)
	}
	joined := strings.Join(result.BlockedReasons, " ")
	for _, wanted := range []string{".env", "image.png", "cannot be safely omitted"} {
		if !strings.Contains(joined, wanted) {
			t.Fatalf("blocked reasons = %q, want %q", joined, wanted)
		}
	}
	for _, warning := range result.Warnings {
		if warning.Code == "FILES_EXCLUDED" || warning.Code == "METADATA_EXCLUDED" {
			t.Fatalf("blocked functional exclusions must not be represented as publishable warnings: %#v", result.Warnings)
		}
	}
}

func TestNormalizerSafelyIgnoresUnreferencedInertMetadata(t *testing.T) {
	files := []SourceFile{
		{Path: "SKILL.md", Data: []byte("---\nname: demo\ndescription: Demo skill\n---\n\n# Demo\n")},
		{Path: ".DS_Store", Data: []byte{0, 1, 2, 3}},
		{Path: ".coverage", Data: []byte("metadata mentions .DS_Store but excluded metadata is never scanned\n")},
		{Path: ".git/config", Data: []byte("[core]\nrepositoryformatversion = 0\n")},
	}
	bundle := SourceBundle{Files: files, UpstreamContentHash: canonicalFilesHash(files), IntegrityVerified: true}
	result, err := NormalizeSkill(NormalizeInput{
		Skill: DiscoveredSkill{AdapterType: "fixture", SuggestedName: "Demo"}, Bundle: bundle, MarketSlug: "demo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Blocked || result.NeedsReview || len(result.ExcludedFiles) != 3 || len(result.PackageData) == 0 || !result.Transformed {
		t.Fatalf("unexpected metadata exclusion result: %#v", result)
	}
	foundMetadataWarning := false
	for _, warning := range result.Warnings {
		if warning.Code == "METADATA_EXCLUDED" {
			foundMetadataWarning = true
		}
	}
	if !foundMetadataWarning {
		t.Fatalf("warnings = %#v, want METADATA_EXCLUDED", result.Warnings)
	}
}

func TestNormalizerSafelyIgnoresReferencedInertMetadata(t *testing.T) {
	files := []SourceFile{
		{Path: "SKILL.md", Data: []byte("---\nname: demo\ndescription: Demo skill\n---\n\nDelete .DS_Store and .git/config if present.\n")},
		{Path: ".DS_Store", Data: []byte{0, 1, 2, 3}},
		{Path: ".git/config", Data: []byte("[core]\nrepositoryformatversion = 0\n")},
	}
	result, err := NormalizeSkill(NormalizeInput{
		Skill:      DiscoveredSkill{AdapterType: "fixture", SuggestedName: "Demo"},
		Bundle:     SourceBundle{Files: files, UpstreamContentHash: canonicalFilesHash(files), IntegrityVerified: true},
		MarketSlug: "demo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Blocked || result.NeedsReview || len(result.ExcludedFiles) != 2 || len(result.PackageData) == 0 {
		t.Fatalf("referenced metadata exclusion result: %#v", result)
	}
}

func TestNormalizerBlocksMarketplaceFileCountTruncation(t *testing.T) {
	files := []SourceFile{{Path: "SKILL.md", Data: []byte("---\nname: demo\ndescription: Demo skill\n---\n\n# Demo\n")}}
	for index := 0; index < service.SkillArchiveMaxFiles; index++ {
		files = append(files, SourceFile{Path: fmt.Sprintf("references/%03d.md", index), Data: []byte("reference\n")})
	}
	result, err := NormalizeSkill(NormalizeInput{
		Skill:      DiscoveredSkill{AdapterType: "fixture", SuggestedName: "Demo"},
		Bundle:     SourceBundle{Files: files, UpstreamContentHash: canonicalFilesHash(files), IntegrityVerified: true},
		MarketSlug: "demo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Blocked || len(result.PackageData) != 0 || !strings.Contains(strings.Join(result.BlockedReasons, "\n"), "file count limit") {
		t.Fatalf("file-count result = %#v", result)
	}
}

func TestNormalizerBlocksMarketplaceUnpackedSizeTruncation(t *testing.T) {
	files := []SourceFile{{Path: "SKILL.md", Data: []byte("---\nname: demo\ndescription: Demo skill\n---\n\n# Demo\n")}}
	for index := 0; index < 6; index++ {
		files = append(files, SourceFile{Path: fmt.Sprintf("references/%d.txt", index), Data: bytes.Repeat([]byte("x"), 900*1024)})
	}
	result, err := NormalizeSkill(NormalizeInput{
		Skill:      DiscoveredSkill{AdapterType: "fixture", SuggestedName: "Demo"},
		Bundle:     SourceBundle{Files: files, UpstreamContentHash: canonicalFilesHash(files), IntegrityVerified: true},
		MarketSlug: "demo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Blocked || len(result.PackageData) != 0 || !strings.Contains(strings.Join(result.BlockedReasons, "\n"), "unpacked size limit") {
		t.Fatalf("unpacked-size result = %#v", result)
	}
}

func TestNormalizerBlocksHallmarkStyleReferencedExcludedReference(t *testing.T) {
	files := []SourceFile{
		{Path: "SKILL.md", Data: []byte("---\nname: hallmark\ndescription: Hallmark fixture\n---\n\nRead [the checklist](references/checklist.md) before continuing.\n")},
		{Path: "references/checklist.md", Data: bytes.Repeat([]byte("x"), int(service.SkillArchiveMaxFileBytes)+1)},
	}
	result, err := NormalizeSkill(NormalizeInput{
		Skill:      DiscoveredSkill{AdapterType: "fixture", SuggestedName: "Hallmark"},
		Bundle:     SourceBundle{Files: files, UpstreamContentHash: canonicalFilesHash(files), IntegrityVerified: true},
		MarketSlug: "hallmark",
	})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(result.BlockedReasons, "\n")
	if !result.Blocked || len(result.PackageData) != 0 || !strings.Contains(joined, `functional source file "references/checklist.md" cannot be safely omitted`) {
		t.Fatalf("referenced-reference result = %#v", result)
	}
}

func TestNormalizerBlocksHyperframesStyleFontReferencedFromText(t *testing.T) {
	files := []SourceFile{
		{Path: "SKILL.md", Data: []byte("---\nname: hyperframes\ndescription: Hyperframes fixture\n---\n\n# Hyperframes\n")},
		{Path: "assets/styles/site.css", Data: []byte("@font-face { src: url('../fonts/Inter.woff2'); }\n")},
		{Path: "assets/fonts/Inter.woff2", Data: []byte{0, 1, 2, 3}},
	}
	result, err := NormalizeSkill(NormalizeInput{
		Skill:      DiscoveredSkill{AdapterType: "fixture", SuggestedName: "Hyperframes"},
		Bundle:     SourceBundle{Files: files, UpstreamContentHash: canonicalFilesHash(files), IntegrityVerified: true},
		MarketSlug: "hyperframes",
	})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(result.BlockedReasons, "\n")
	if !result.Blocked || len(result.PackageData) != 0 || !strings.Contains(joined, `functional source file "assets/fonts/Inter.woff2" cannot be safely omitted`) {
		t.Fatalf("referenced-font result = %#v", result)
	}
}

func TestFinalizeNormalizedDeduplicatesReviewWarnings(t *testing.T) {
	result := finalizeNormalized(NormalizedSkill{
		ReviewReasons: []string{"frontmatter needs review", "frontmatter needs review"},
		Warnings: []service.SkillValidationIssue{
			{Code: "REVIEW_REQUIRED", Message: "frontmatter needs review"},
		},
	})
	if len(result.ReviewReasons) != 1 || len(result.Warnings) != 1 {
		t.Fatalf("deduplicated review result = %#v", result)
	}
}

func TestNormalizerPreservesEveryFrontmatterReviewReason(t *testing.T) {
	files := []SourceFile{{Path: "SKILL.md", Data: []byte("# Instructions\n\nDo the work.\n")}}
	result, err := NormalizeSkill(NormalizeInput{
		Skill:      DiscoveredSkill{AdapterType: "fixture", SuggestedName: "Demo"},
		Bundle:     SourceBundle{Files: files, UpstreamContentHash: canonicalFilesHash(files), IntegrityVerified: true},
		MarketSlug: "demo",
	})
	if err != nil || result.Blocked || !result.NeedsReview {
		t.Fatalf("normalization result=%#v err=%v", result, err)
	}
	joined := strings.Join(result.ReviewReasons, "\n")
	for _, wanted := range []string{"missing a usable description", "no valid frontmatter"} {
		if !strings.Contains(joined, wanted) {
			t.Fatalf("review reasons %q do not contain %q", joined, wanted)
		}
	}
	if len(result.ReviewReasons) != 2 {
		t.Fatalf("review reasons = %#v, want two independent reasons", result.ReviewReasons)
	}
}

func TestNormalizerBlocksTamperedBundleHash(t *testing.T) {
	files := []SourceFile{{Path: "SKILL.md", Data: []byte("---\nname: demo\ndescription: Demo\n---\n\n# Demo\n")}}
	result, err := NormalizeSkill(NormalizeInput{
		Skill:  DiscoveredSkill{AdapterType: "fixture"},
		Bundle: SourceBundle{Files: files, UpstreamContentHash: strings.Repeat("0", 64)}, MarketSlug: "demo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Blocked || len(result.PackageData) != 0 {
		t.Fatalf("tampered bundle result = %#v", result)
	}
}

func TestNormalizerRequiresReviewWithoutVerifiedUpstreamDigest(t *testing.T) {
	files := []SourceFile{
		{Path: "SKILL.md", Data: []byte("---\nname: demo\ndescription: Demo skill\n---\n\n# Demo\n")},
		{Path: "LICENSE", Data: []byte("Permission granted.\n")},
	}
	result, err := NormalizeSkill(NormalizeInput{
		Skill: DiscoveredSkill{AdapterType: "fixture", SuggestedName: "Demo"},
		Bundle: SourceBundle{
			Files: files, UpstreamContentHash: canonicalFilesHash(files), IntegrityVerified: false,
		},
		MarketSlug: "demo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Blocked || !result.NeedsReview || len(result.PackageData) == 0 {
		t.Fatalf("unverified upstream result = %#v", result)
	}
	if !strings.Contains(strings.Join(result.ReviewReasons, "\n"), "independently verified digest") {
		t.Fatalf("review reasons = %#v", result.ReviewReasons)
	}
	var integrityWarning, reviewWarning bool
	for _, warning := range result.Warnings {
		switch warning.Code {
		case "UPSTREAM_INTEGRITY_UNVERIFIED":
			integrityWarning = true
		case "REVIEW_REQUIRED":
			reviewWarning = true
		}
	}
	if !integrityWarning || !reviewWarning {
		t.Fatalf("warnings = %#v", result.Warnings)
	}
}
