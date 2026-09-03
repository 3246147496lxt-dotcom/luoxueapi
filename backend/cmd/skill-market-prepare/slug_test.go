package main

import (
	"regexp"
	"strings"
	"testing"
)

func TestAllocateMarketSlugsHandlesCollisionsIllegalCharactersAndLength(t *testing.T) {
	records := []rankingRecord{
		{Source: "alpha/skills", SkillID: "shared", Name: "Shared", Installs: 10},
		{Source: "beta/skills", SkillID: "shared", Name: "Shared", Installs: 9},
		{Source: "beta/skills", SkillID: "shared", Name: "Shared", Installs: 8},
		{Source: "gamma/skills", SkillID: "React:Components", Name: "React:Components", Installs: 7},
		{Source: "delta/skills", SkillID: "!!!", Name: "Symbols", Installs: 6},
		{Source: "epsilon/repository", SkillID: strings.Repeat("very-long-", 12), Name: "Long", Installs: 5},
	}
	entries := allocateMarketSlugs(records)
	want := []string{"shared", "shared-beta-skills", "shared-beta-skills-2", "react-components", "skill"}
	for index, expected := range want {
		if entries[index].MarketSlug != expected {
			t.Fatalf("slug %d = %q, want %q", index, entries[index].MarketSlug, expected)
		}
	}
	if entries[0].Name != "Shared" {
		t.Fatalf("first display name changed: %q", entries[0].Name)
	}
	if entries[1].Name != "Shared · beta/skills" {
		t.Fatalf("colliding display name = %q", entries[1].Name)
	}
	valid := regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	seen := map[string]bool{}
	for _, entry := range entries {
		if len(entry.MarketSlug) > 64 || !valid.MatchString(entry.MarketSlug) {
			t.Fatalf("invalid market slug %q", entry.MarketSlug)
		}
		if seen[entry.MarketSlug] {
			t.Fatalf("duplicate market slug %q", entry.MarketSlug)
		}
		seen[entry.MarketSlug] = true
	}
}
