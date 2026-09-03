package chatattachment

import (
	"strings"
	"unicode/utf8"
)

type boundedText struct {
	builder   strings.Builder
	runeCount int
	limit     int
}

func newBoundedText(limit int) *boundedText {
	return &boundedText{limit: limit}
}

func (b *boundedText) append(value string) bool {
	if value == "" {
		return true
	}
	value = strings.ToValidUTF8(value, "\uFFFD")
	count := utf8.RuneCountInString(value)
	if count > b.limit-b.runeCount {
		return false
	}
	_, _ = b.builder.WriteString(value)
	b.runeCount += count
	return true
}

func (b *boundedText) string() string {
	return strings.TrimSpace(b.builder.String())
}

func (b *boundedText) empty() bool {
	return strings.TrimSpace(b.builder.String()) == ""
}
