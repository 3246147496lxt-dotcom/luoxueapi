package skillimport

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestAcquireRemoteArtifactRejectsLateDuplicateBeforeFetchingAnyFile(t *testing.T) {
	fetcher := &scriptedFetcher{handler: func(target string, _ FetchOptions) (FetchResult, error) {
		return FetchResult{}, fmt.Errorf("must reject before fetching %s", target)
	}}
	_, err := acquireRemoteArtifact(context.Background(), fetcher, "fixture", "https://assets.example.test/manifest.json",
		[]string{"assets.example.test"}, remoteArtifact{Files: []remoteFileReference{
			{Path: "SKILL.md", URL: "one.md"},
			{Path: "skill.md", URL: "two.md"},
		}})
	if ErrorKindOf(err) != ErrorUnsafe || !strings.Contains(err.Error(), "case-conflicting") {
		t.Fatalf("duplicate reference error = %v", err)
	}
	if len(fetcher.calls) != 0 {
		t.Fatalf("malformed file set made outbound calls: %#v", fetcher.calls)
	}
}
