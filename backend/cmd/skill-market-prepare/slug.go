package main

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

var repeatedDashPattern = regexp.MustCompile(`-+`)

func slugify(input string) string {
	decomposed := norm.NFKD.String(strings.ToLower(strings.TrimSpace(input)))
	var output strings.Builder
	output.Grow(len(decomposed))
	lastDash := false
	for _, r := range decomposed {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			_, _ = output.WriteRune(r)
			lastDash = false
		case unicode.Is(unicode.Mn, r):
			continue
		default:
			if output.Len() > 0 && !lastDash {
				_ = output.WriteByte('-')
				lastDash = true
			}
		}
	}
	result := strings.Trim(repeatedDashPattern.ReplaceAllString(output.String(), "-"), "-")
	if result == "" {
		result = "skill"
	}
	return truncateSlug(result, 64)
}

func allocateMarketSlugs(records []rankingRecord) []snapshotEntry {
	used := make(map[string]struct{}, len(records))
	seenNames := make(map[string]struct{}, len(records))
	entries := make([]snapshotEntry, 0, len(records))
	for index, record := range records {
		base := slugify(record.SkillID)
		candidate := base
		_, slugCollision := used[candidate]
		if slugCollision {
			suffix := slugify(strings.ReplaceAll(record.Source, "/", "-"))
			candidate = joinSlugParts(base, suffix)
			for sequence := 2; ; sequence++ {
				if _, exists := used[candidate]; !exists {
					break
				}
				candidate = joinSlugParts(base, suffix, base36(sequence))
			}
		}
		if _, exists := used[candidate]; exists {
			digest := sha256.Sum256([]byte(record.Source + "\x00" + record.SkillID))
			candidate = joinSlugParts(base, hex.EncodeToString(digest[:4]))
		}
		used[candidate] = struct{}{}
		displayName := truncateRunes(strings.TrimSpace(record.Name), 120)
		nameKey := strings.ToLower(displayName)
		_, nameCollision := seenNames[nameKey]
		if nameCollision || slugCollision {
			displayName = truncateRunes(displayName+" · "+record.Source, 120)
		}
		seenNames[nameKey] = struct{}{}
		entries = append(entries, snapshotEntry{
			Rank: index + 1, Source: record.Source, OriginalSlug: record.SkillID,
			MarketSlug: candidate, OriginalName: record.Name, Name: displayName, Installs: record.Installs,
		})
	}
	return entries
}

func joinSlugParts(parts ...string) string {
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.Trim(slugify(part), "-")
		if part != "" {
			clean = append(clean, part)
		}
	}
	if len(clean) == 0 {
		return "skill"
	}
	suffix := ""
	if len(clean) > 1 {
		suffix = "-" + strings.Join(clean[1:], "-")
	}
	maxBase := 64 - len(suffix)
	if maxBase < 1 {
		digest := sha256.Sum256([]byte(strings.Join(clean, "\x00")))
		return "skill-" + hex.EncodeToString(digest[:8])
	}
	base := truncateSlug(clean[0], maxBase)
	return strings.Trim(base+suffix, "-")
}

func truncateSlug(value string, maximum int) string {
	value = strings.Trim(value, "-")
	if len(value) <= maximum {
		return value
	}
	value = strings.Trim(value[:maximum], "-")
	if value == "" {
		return "skill"
	}
	return value
}

func base36(value int) string {
	const alphabet = "0123456789abcdefghijklmnopqrstuvwxyz"
	if value <= 0 {
		return "0"
	}
	var buffer [16]byte
	position := len(buffer)
	for value > 0 {
		position--
		buffer[position] = alphabet[value%36]
		value /= 36
	}
	return string(buffer[position:])
}
