package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func classifySkill(entry snapshotEntry, description string) (string, []string) {
	haystack := strings.ToLower(entry.Source + " " + entry.OriginalSlug + " " + entry.Name + " " + description)
	category := "开发工具"
	cases := []struct {
		Category string
		Terms    []string
	}{
		{"安全与研究", []string{"security", "secure", "audit", "osint", "vulnerability", "pentest"}},
		{"营销与增长", []string{"marketing", "seo", "social", "content", "campaign", "sales", "cro"}},
		{"设计与前端", []string{"design", "frontend", "css", "tailwind", "ui", "ux", "animation"}},
		{"测试与质量", []string{"test", "playwright", "qa", "debug", "review", "lint"}},
		{"文档与办公", []string{"document", "docs", "pdf", "spreadsheet", "slide", "presentation", "notion", "obsidian"}},
		{"数据与数据库", []string{"database", "postgres", "sql", "data", "analytics", "supabase", "firebase"}},
		{"AI 与自动化", []string{"agent", "prompt", "model", "llm", "automation", "workflow", "mcp"}},
		{"研发协作", []string{"github", "git", "commit", "release", "deploy", "ci", "repository"}},
	}
	for _, candidate := range cases {
		for _, term := range candidate.Terms {
			if strings.Contains(haystack, term) {
				category = candidate.Category
				break
			}
		}
		if category == candidate.Category {
			break
		}
	}

	tagSet := map[string]struct{}{"skills.sh": {}}
	owner := strings.Split(entry.Source, "/")[0]
	for _, value := range []string{owner, entry.OriginalSlug} {
		for _, token := range strings.FieldsFunc(strings.ToLower(value), func(r rune) bool {
			return !(unicode.IsLetter(r) || unicode.IsDigit(r))
		}) {
			if utf8.RuneCountInString(token) >= 2 && utf8.RuneCountInString(token) <= 24 {
				tagSet[token] = struct{}{}
			}
		}
	}
	tags := make([]string, 0, len(tagSet)+1)
	for tag := range tagSet {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	if len(tags) > 7 {
		tags = tags[:7]
	}
	return category, tags
}

func summaryFromDescription(description, fallback string) string {
	description = strings.Join(strings.Fields(description), " ")
	if description == "" {
		description = fallback
	}
	for _, separator := range []string{". ", "。", "! ", "? "} {
		if index := strings.Index(description, separator); index > 20 {
			description = description[:index] + string([]rune(separator)[0])
			break
		}
	}
	return truncateRunes(description, 180)
}

func localRiskNotes(report service.SkillValidationReport) []string {
	notes := make([]string, 0, len(report.Warnings))
	for _, warning := range report.Warnings {
		where := ""
		if warning.Path != "" {
			where = " at " + warning.Path
		}
		notes = append(notes, fmt.Sprintf("Local archive preflight | %s%s | %s", warning.Code, where, warning.Message))
	}
	return notes
}

func archiveLicenseFiles(files []service.SkillArchiveFile) []string {
	var licenses []string
	for _, file := range files {
		base := strings.ToLower(filepath.Base(file.Path))
		trimmed := strings.TrimSuffix(base, filepath.Ext(base))
		if trimmed == "license" || trimmed == "licence" || trimmed == "copying" {
			licenses = append(licenses, file.Path)
		}
	}
	sort.Strings(licenses)
	return licenses
}
