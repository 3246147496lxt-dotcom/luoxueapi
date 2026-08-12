package skillimport

import (
	"bytes"
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

func TestNormalizerMarksExclusionsNeedsReviewInsteadOfSilentRemoval(t *testing.T) {
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
	if result.Blocked || !result.NeedsReview || len(result.ExcludedFiles) != 2 || len(result.PackageData) == 0 {
		t.Fatalf("unexpected exclusion result: %#v", result)
	}
	joined := strings.Join(result.ReviewReasons, " ")
	if !strings.Contains(joined, "excluded") {
		t.Fatalf("review reasons = %q, want explicit exclusion reason", joined)
	}
	reviewWarnings := 0
	for _, warning := range result.Warnings {
		if warning.Code == "REVIEW_REQUIRED" {
			reviewWarnings++
			if warning.Message == "" {
				t.Fatal("REVIEW_REQUIRED warning must expose the review reason")
			}
		}
	}
	if reviewWarnings != len(result.ReviewReasons) {
		t.Fatalf("review warnings = %d, reasons = %d", reviewWarnings, len(result.ReviewReasons))
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
