package main

import "testing"

func TestEscapeURLSegmentEncodesColonAndSlash(t *testing.T) {
	if got := escapeURLSegment("react:components"); got != "react%3Acomponents" {
		t.Fatalf("colon segment = %q", got)
	}
	if got := escapeURLSegment("nested/name"); got != "nested%2Fname" {
		t.Fatalf("slash segment = %q", got)
	}
}
