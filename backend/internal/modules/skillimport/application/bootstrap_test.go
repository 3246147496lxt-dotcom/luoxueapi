package application

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"

	domain "github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestExtractSkillsSHBootstrapEvidenceUsesEmbeddedStableIdentity(t *testing.T) {
	packageData := bootstrapTestZIP(t, "shared-skill", `---
name: shared-skill
description: A reusable test skill.
metadata:
  original_source: example/skills
  skills_sh_id: original:skill
  snapshot_hash: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
---
Use these instructions for the test.
`)
	validated, err := domain.ValidateSkillArchive(packageData, "shared-skill")
	require.NoError(t, err)

	evidence, err := ExtractSkillsSHBootstrapEvidence(9, domain.SkillImportBootstrapCandidate{
		SkillID: 12, Slug: "shared-skill", VersionID: 15,
		SHA256: validated.SHA256, PackageData: packageData,
	})
	require.NoError(t, err)
	require.Equal(t, domain.SkillImportStableKey{SourceID: 9, Namespace: "example/skills", ExternalID: "original:skill"}, evidence.Key)
	require.Equal(t, "https://skills.sh/example/skills/original:skill", evidence.OriginURL)
	require.Equal(t, strings.Repeat("a", 64), evidence.SourceRevision)
	require.Empty(t, evidence.SourceContentSHA256, "opaque snapshot hashes and Git commits are not canonical logical-file hashes")
	require.Equal(t, validated.SHA256, evidence.PackageSHA256)
}

func TestExtractSkillsSHBootstrapEvidenceAcceptsExplicitCanonicalContentHash(t *testing.T) {
	contentHash := strings.Repeat("b", 64)
	packageData := bootstrapTestZIP(t, "shared-skill", `---
name: shared-skill
description: A reusable test skill.
metadata:
  skills_sh_import:
    source: example/skills
    skill_id: original-skill
    revision: git-revision
    canonical_content_hash: `+contentHash+`
---
Use these instructions for the test.
`)
	evidence, err := ExtractSkillsSHBootstrapEvidence(9, domain.SkillImportBootstrapCandidate{
		SkillID: 12, Slug: "shared-skill", VersionID: 15, PackageData: packageData,
	})
	require.NoError(t, err)
	require.Equal(t, "git-revision", evidence.SourceRevision)
	require.Equal(t, contentHash, evidence.SourceContentSHA256)
}

func TestExtractSkillsSHBootstrapEvidenceRejectsAmbiguousPackage(t *testing.T) {
	packageData := bootstrapTestZIP(t, "plain-skill", `---
name: plain-skill
description: A reusable test skill.
---
Use these instructions for the test.
`)
	_, err := ExtractSkillsSHBootstrapEvidence(9, domain.SkillImportBootstrapCandidate{
		SkillID: 12, Slug: "plain-skill", VersionID: 15, PackageData: packageData,
	})
	require.ErrorContains(t, err, "no unambiguous")
}

func TestExtractSkillsSHBootstrapEvidenceRejectsInvalidStableIdentity(t *testing.T) {
	packageData := bootstrapTestZIP(t, "shared-skill", `---
name: shared-skill
description: A reusable test skill.
metadata:
  original_source: bad source
  skills_sh_id: demo
---
Use these instructions for the test.
`)
	_, err := ExtractSkillsSHBootstrapEvidence(9, domain.SkillImportBootstrapCandidate{
		SkillID: 12, Slug: "shared-skill", VersionID: 15, PackageData: packageData,
	})
	require.ErrorContains(t, err, "invalid or oversized")
}

func bootstrapTestZIP(t *testing.T, slug, skillMD string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	entry, err := writer.Create(slug + "/SKILL.md")
	require.NoError(t, err)
	_, err = entry.Write([]byte(skillMD))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	return buffer.Bytes()
}
