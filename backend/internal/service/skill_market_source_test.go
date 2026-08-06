package service

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNormalizeSkillSourceURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantURL  string
		wantRepo string
		wantErr  bool
	}{
		{name: "empty", input: ""},
		{name: "repository", input: " https://github.com/anthropics/skills/ ", wantURL: "https://github.com/anthropics/skills", wantRepo: "anthropics/skills"},
		{name: "tree", input: "https://github.com/anthropics/skills/tree/main/skills/frontend-design", wantURL: "https://github.com/anthropics/skills/tree/main/skills/frontend-design", wantRepo: "anthropics/skills"},
		{name: "blob", input: "https://github.com/anthropics/skills/blob/main/LICENSE", wantURL: "https://github.com/anthropics/skills/blob/main/LICENSE", wantRepo: "anthropics/skills"},
		{name: "http", input: "http://github.com/anthropics/skills", wantErr: true},
		{name: "lookalike", input: "https://github.com.evil.test/anthropics/skills", wantErr: true},
		{name: "port", input: "https://github.com:443/anthropics/skills", wantErr: true},
		{name: "credentials", input: "https://user@github.com/anthropics/skills", wantErr: true},
		{name: "query", input: "https://github.com/anthropics/skills?tab=readme", wantErr: true},
		{name: "unsupported path", input: "https://github.com/anthropics/skills/issues", wantErr: true},
		{name: "encoded traversal", input: "https://github.com/anthropics/skills/tree/main/%2e%2e/secret", wantErr: true},
		{name: "blob missing file", input: "https://github.com/anthropics/skills/blob/main", wantErr: true},
		{name: "oversized", input: "https://github.com/anthropics/skills/tree/main/" + strings.Repeat("a", 2048), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL, gotRepo, err := normalizeSkillSourceURL(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantURL, gotURL)
			require.Equal(t, tt.wantRepo, gotRepo)
		})
	}
}

func TestNormalizeSkillInputDerivesSourceAndAllowsEmptyRiskAndExamples(t *testing.T) {
	sourceURL := "https://github.com/Example/Repo/tree/main/skill"
	input, err := normalizeSkillInput(SkillInput{
		Slug:        "demo-skill",
		DisplayName: "Demo",
		Summary:     "Summary",
		Description: "Description",
		Category:    "Developer Tools",
		SourceURL:   &sourceURL,
	})
	require.NoError(t, err)
	require.NotNil(t, input.SourceURL)
	require.Equal(t, "https://github.com/Example/Repo/tree/main/skill", *input.SourceURL)
	require.Empty(t, input.RiskNotes)
	require.Empty(t, input.ExamplePrompts)

	skill := skillFromInput(input, nil)
	require.Equal(t, "Example/Repo", skill.SourceRepository)
	require.Nil(t, skill.RepositoryStars)
}

func TestApplySkillInputPreservesOmittedSourceAndClearsExplicitEmptySource(t *testing.T) {
	fetchedAt := time.Now().UTC().Add(-time.Hour)
	refreshAfter := fetchedAt.Add(24 * time.Hour)
	stars := int64(42)
	skill := &Skill{
		SourceURL:                   "https://github.com/Example/Repo",
		SourceRepository:            "Example/Repo",
		RepositoryStars:             &stars,
		RepositoryStarsFetchedAt:    &fetchedAt,
		RepositoryStarsRefreshAfter: &refreshAfter,
	}

	applySkillInput(skill, SkillInput{}, nil)
	require.Equal(t, "https://github.com/Example/Repo", skill.SourceURL)
	require.Equal(t, "Example/Repo", skill.SourceRepository)
	require.Equal(t, int64(42), *skill.RepositoryStars)

	emptySource := ""
	applySkillInput(skill, SkillInput{SourceURL: &emptySource}, nil)
	require.Empty(t, skill.SourceURL)
	require.Empty(t, skill.SourceRepository)
	require.Nil(t, skill.RepositoryStars)
	require.Nil(t, skill.RepositoryStarsFetchedAt)
	require.Nil(t, skill.RepositoryStarsRefreshAfter)
}
func TestValidateSkillPublishableDoesNotRequireRiskNotesOrExamplePrompts(t *testing.T) {
	skill := &Skill{
		DisplayName: "demo-skill",
		Summary:     "Summary",
		Description: "Description",
		Category:    "developer-tools",
	}
	version := &SkillVersion{ValidationReport: SkillValidationReport{Valid: true}}
	require.NoError(t, validateSkillPublishable(skill, version))
}
