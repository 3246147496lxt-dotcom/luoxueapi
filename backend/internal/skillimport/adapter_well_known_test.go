package skillimport

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestWellKnownAdapterDiscoversAndAcquiresDeclaredFiles(t *testing.T) {
	index := `{"skills":[{"name":"demo","description":"Preserved upstream description","files":["SKILL.md","LICENSE"]}]}`
	fetcher := &scriptedFetcher{handler: func(target string, _ FetchOptions) (FetchResult, error) {
		switch {
		case strings.HasSuffix(target, "/.well-known/agent-skills/index.json"):
			return FetchResult{}, NewAdapterError(WellKnownAdapterType, "read index", ErrorNotFound, fmt.Errorf("not found"))
		case strings.HasSuffix(target, "/.well-known/skills/index.json"):
			return fetchResult(target, "application/json", []byte(index)), nil
		case strings.HasSuffix(target, "/demo/SKILL.md"):
			return fetchResult(target, "text/markdown", []byte("---\nname: demo\ndescription: Demo\n---\n\n# Demo\n")), nil
		case strings.HasSuffix(target, "/demo/LICENSE"):
			return fetchResult(target, "text/plain", []byte("MIT\n")), nil
		default:
			return FetchResult{}, fmt.Errorf("unexpected URL %s", target)
		}
	}}
	adapter := NewWellKnownAdapter(fetcher)
	page, err := adapter.Discover(context.Background(), DiscoverRequest{
		Config: json.RawMessage(`{}`), BaseURL: "https://skills.example.test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].StableKey() != "well_known\x00skills.example.test\x00demo" || page.Items[0].Description != "Preserved upstream description" {
		t.Fatalf("well-known discovery = %#v", page)
	}
	bundle, err := adapter.Acquire(context.Background(), AcquireRequest{
		Config: json.RawMessage(`{}`), BaseURL: "https://skills.example.test", Skill: page.Items[0],
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Files) != 2 || len(bundle.LicenseEvidence) != 1 || bundle.UpstreamContentHash == "" {
		t.Fatalf("well-known bundle = %#v", bundle)
	}
	if fetcher.callCount("/.well-known/agent-skills/index.json") != 1 || fetcher.callCount("/.well-known/skills/index.json") != 1 {
		t.Fatalf("well-known fallback calls = %#v", fetcher.calls)
	}
}

func TestWellKnownAdapterPrefersAgentSkillsPathAndReusesItForFiles(t *testing.T) {
	index := `{"skills":[{"name":"demo","files":["SKILL.md"]}]}`
	fetcher := &scriptedFetcher{handler: func(target string, _ FetchOptions) (FetchResult, error) {
		switch {
		case strings.HasSuffix(target, "/.well-known/agent-skills/index.json"):
			return fetchResult(target, "application/json", []byte(index)), nil
		case strings.HasSuffix(target, "/.well-known/agent-skills/demo/SKILL.md"):
			return fetchResult(target, "text/markdown", []byte("---\nname: demo\ndescription: Demo\n---\n")), nil
		default:
			return FetchResult{}, fmt.Errorf("unexpected URL %s", target)
		}
	}}
	adapter := NewWellKnownAdapter(fetcher)
	page, err := adapter.Discover(context.Background(), DiscoverRequest{BaseURL: "https://skills.example.test"})
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("Discover() items=%d err=%v", len(page.Items), err)
	}
	if !strings.Contains(page.Items[0].CanonicalURL, "/.well-known/agent-skills/demo") {
		t.Fatalf("canonical URL = %q", page.Items[0].CanonicalURL)
	}
	bundle, err := adapter.Acquire(context.Background(), AcquireRequest{BaseURL: "https://skills.example.test", Skill: page.Items[0]})
	if err != nil || len(bundle.Files) != 1 {
		t.Fatalf("Acquire() files=%d err=%v", len(bundle.Files), err)
	}
	if fetcher.callCount("/.well-known/skills/") != 0 {
		t.Fatal("legacy well-known path must not be used when agent-skills exists")
	}
}

func TestWellKnownAdapterRejectsSkillNamePathTraversal(t *testing.T) {
	index := `{"skills":[{"name":"../outside","files":["SKILL.md"]}]}`
	fetcher := &scriptedFetcher{handler: func(target string, _ FetchOptions) (FetchResult, error) {
		return fetchResult(target, "application/json", []byte(index)), nil
	}}
	adapter := NewWellKnownAdapter(fetcher)
	_, err := adapter.Discover(context.Background(), DiscoverRequest{BaseURL: "https://skills.example.test"})
	if ErrorKindOf(err) != ErrorIntegrity || !strings.Contains(err.Error(), "unsafe name") {
		t.Fatalf("traversal name error = %v", err)
	}
	if fetcher.callCount("outside") != 0 {
		t.Fatal("unsafe skill name was used to construct a file URL")
	}
}
