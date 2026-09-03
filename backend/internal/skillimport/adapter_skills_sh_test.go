package skillimport

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestSkillsSHDiscoveryPaginatesStableRanking(t *testing.T) {
	fetcher := &scriptedFetcher{handler: func(target string, _ FetchOptions) (FetchResult, error) {
		var body string
		switch {
		case strings.HasSuffix(target, "/0"):
			body = `{"skills":[{"source":"one/repo","skillId":"a","name":"A","installs":10},{"source":"one/repo","skillId":"b","name":"B","installs":9}],"page":0,"hasMore":true}`
		case strings.HasSuffix(target, "/1"):
			body = `{"skills":[{"source":"two/repo","skillId":"c","name":"C","installs":8},{"source":"two/repo","skillId":"d","name":"D","installs":7}],"page":1,"hasMore":true}`
		default:
			return FetchResult{}, fmt.Errorf("unexpected URL %s", target)
		}
		return fetchResult(target, "application/json", []byte(body)), nil
	}}
	adapter := NewSkillsSHAdapter(fetcher)
	selection := json.RawMessage(`{"start_rank":2,"limit":3,"page_size":100}`)
	first, err := adapter.Discover(context.Background(), DiscoverRequest{Config: json.RawMessage(`{"view":"all-time"}`), Selection: selection})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != 1 || *first.Items[0].Rank != 2 || first.Items[0].StableKey() != "skills_sh\x00one/repo\x00b" || first.NextCursor == "" {
		t.Fatalf("unexpected first page: %#v", first)
	}
	second, err := adapter.Discover(context.Background(), DiscoverRequest{
		Config: json.RawMessage(`{"view":"all-time"}`), Selection: selection, Cursor: first.NextCursor,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Items) != 2 || *second.Items[0].Rank != 3 || *second.Items[1].Rank != 4 || second.NextCursor != "" {
		t.Fatalf("unexpected second page: %#v", second)
	}
}

func TestSkillsSHAcquisitionUsesGitHubBeforeScarceSnapshot(t *testing.T) {
	commit := strings.Repeat("a", 40)
	skillA := []byte("---\nname: demo-a\ndescription: A demo\n---\n\n# Demo A\n")
	skillB := []byte("---\nname: demo-b\ndescription: B demo\n---\n\n# Demo B\n")
	fetcher := &scriptedFetcher{handler: func(target string, _ FetchOptions) (FetchResult, error) {
		switch {
		case strings.Contains(target, "/commits/HEAD"):
			return fetchResult(target, "application/json", []byte(`{"sha":"`+commit+`"}`)), nil
		case strings.Contains(target, "/git/trees/"):
			body := fmt.Sprintf(`{"sha":%q,"truncated":false,"tree":[{"path":"skills/demo-a/SKILL.md","mode":"100644","type":"blob","sha":%q,"size":%d},{"path":"skills/demo-b/SKILL.md","mode":"100644","type":"blob","sha":%q,"size":%d}]}`,
				commit, gitBlobObjectID(skillA), len(skillA), gitBlobObjectID(skillB), len(skillB))
			return fetchResult(target, "application/json", []byte(body)), nil
		case strings.Contains(target, "skills/demo-a/SKILL.md"):
			return fetchResult(target, "text/plain", skillA), nil
		case strings.Contains(target, "skills/demo-b/SKILL.md"):
			return fetchResult(target, "text/plain", skillB), nil
		case strings.Contains(target, "/api/download/"):
			return FetchResult{}, fmt.Errorf("scarce skills.sh snapshot endpoint must not be called")
		default:
			return FetchResult{}, fmt.Errorf("unexpected URL %s", target)
		}
	}}
	github := NewGitHubAdapter(fetcher, "token-not-logged")
	adapter := NewSkillsSHAdapter(fetcher, github)
	for _, skillID := range []string{"demo-a", "demo-b"} {
		originURL := "https://skills.sh/owner/repo/" + skillID
		bundle, err := adapter.Acquire(context.Background(), AcquireRequest{
			Config: json.RawMessage(`{"view":"all-time"}`),
			Skill:  DiscoveredSkill{SuggestedName: skillID, SuggestedSlug: skillID, CanonicalURL: originURL, Opaque: safeOpaque(skillsSHOpaque{Source: "owner/repo", SkillID: skillID})},
		})
		if err != nil {
			t.Fatalf("Acquire(%s): %v", skillID, err)
		}
		if bundle.Revision != commit || len(bundle.Files) != 1 || bundle.Files[0].Path != "SKILL.md" {
			t.Fatalf("Acquire(%s) bundle = %#v", skillID, bundle)
		}
		if bundle.CanonicalURL != originURL {
			t.Fatalf("Acquire(%s) canonical URL = %q, want skills.sh origin %q", skillID, bundle.CanonicalURL, originURL)
		}
	}
	if got := fetcher.callCount("/commits/HEAD"); got != 1 {
		t.Fatalf("commit API calls = %d, want 1", got)
	}
	if got := fetcher.callCount("/git/trees/"); got != 1 {
		t.Fatalf("tree API calls = %d, want 1", got)
	}
	if got := fetcher.callCount("/api/download/"); got != 0 {
		t.Fatalf("skills.sh snapshot calls = %d, want 0", got)
	}
}

func TestSkillsSHDoesNotFallbackAfterAuthoritativeSafetyFailure(t *testing.T) {
	commit := strings.Repeat("a", 40)
	skillMD := []byte("---\nname: demo\ndescription: Demo\n---\n")
	fetcher := &scriptedFetcher{handler: func(target string, _ FetchOptions) (FetchResult, error) {
		switch {
		case strings.Contains(target, "/commits/HEAD"):
			return fetchResult(target, "application/json", []byte(`{"sha":"`+commit+`"}`)), nil
		case strings.Contains(target, "/git/trees/"):
			body := fmt.Sprintf(`{"sha":%q,"truncated":false,"tree":[{"path":"skills/demo/SKILL.md","mode":"100644","type":"blob","sha":%q,"size":%d},{"path":"skills/demo/link","mode":"120000","type":"blob","sha":%q,"size":3}]}`, commit, gitBlobObjectID(skillMD), len(skillMD), strings.Repeat("b", 40))
			return fetchResult(target, "application/json", []byte(body)), nil
		case strings.Contains(target, "/api/download/"):
			return FetchResult{}, fmt.Errorf("unsafe GitHub source must not fall back to snapshot")
		default:
			return FetchResult{}, fmt.Errorf("unexpected URL %s", target)
		}
	}}
	adapter := NewSkillsSHAdapter(fetcher, NewGitHubAdapter(fetcher, ""))
	_, err := adapter.Acquire(context.Background(), AcquireRequest{
		Config: json.RawMessage(`{"acquisition_order":["github","skills_sh_snapshot"]}`),
		Skill:  DiscoveredSkill{Opaque: safeOpaque(skillsSHOpaque{Source: "owner/repo", SkillID: "demo"})},
	})
	if ErrorKindOf(err) != ErrorUnsafe {
		t.Fatalf("Acquire() error = %v, want unsafe", err)
	}
	if fetcher.callCount("/api/download/") != 0 {
		t.Fatal("snapshot fallback was attempted after a safety failure")
	}
}

func TestSkillsSHSnapshotChannelHasHourlyQuota(t *testing.T) {
	body := []byte(`{"files":[{"path":"SKILL.md","contents":"---\nname: demo\ndescription: Demo\n---\n\n# Demo\n"}],"hash":"provider-snapshot"}`)
	fetcher := &scriptedFetcher{handler: func(target string, _ FetchOptions) (FetchResult, error) {
		return fetchResult(target, "application/json", body), nil
	}}
	adapter := NewSkillsSHAdapter(fetcher)
	config := json.RawMessage(`{"acquisition_order":["skills_sh_snapshot"],"download_quota_per_hour":1}`)
	for index, skillID := range []string{"one", "two"} {
		bundle, err := adapter.Acquire(context.Background(), AcquireRequest{
			Config: config, Skill: DiscoveredSkill{Opaque: safeOpaque(skillsSHOpaque{Source: "owner/repo", SkillID: skillID})},
		})
		if index == 0 && err != nil {
			t.Fatalf("first snapshot: %v", err)
		}
		if index == 0 && (!bundle.NeedsReview || bundle.IntegrityVerified || len(bundle.ReviewReasons) != 1 || bundle.Revision != "provider-snapshot") {
			t.Fatalf("snapshot trust classification = %#v", bundle)
		}
		if index == 1 && ErrorKindOf(err) != ErrorRateLimited {
			t.Fatalf("second snapshot kind = %q, want rate_limited (err=%v)", ErrorKindOf(err), err)
		}
	}
	if got := fetcher.callCount("/api/download/"); got != 1 {
		t.Fatalf("snapshot HTTP calls = %d, want 1", got)
	}
}

func TestSkillsSHUsesConfiguredBaseURLForNonGitHubSource(t *testing.T) {
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
	adapter := NewSkillsSHAdapter(fetcher)
	origin := "https://skills.sh/sentry/dev/demo"
	bundle, err := adapter.Acquire(context.Background(), AcquireRequest{
		Config: json.RawMessage(`{
			"allowed_source_hosts":["cli.sentry.dev"],
			"source_base_urls":{"sentry/dev":"https://cli.sentry.dev"}
		}`),
		Skill: DiscoveredSkill{CanonicalURL: origin, ExternalID: "demo", Opaque: safeOpaque(skillsSHOpaque{Source: "sentry/dev", SkillID: "demo"})},
	})
	if err != nil || len(bundle.Files) != 1 {
		t.Fatalf("Acquire() files=%d err=%v", len(bundle.Files), err)
	}
	if bundle.CanonicalURL != origin {
		t.Fatalf("canonical URL = %q, want %q", bundle.CanonicalURL, origin)
	}
	if fetcher.callCount("cli.sentry.dev") == 0 || fetcher.callCount("https://sentry/dev") != 0 {
		t.Fatalf("unexpected calls: %#v", fetcher.calls)
	}
}

func TestSkillsSHUsesConfiguredBaseURLForDomainSourceKey(t *testing.T) {
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
	adapter := NewSkillsSHAdapter(fetcher)
	bundle, err := adapter.Acquire(context.Background(), AcquireRequest{
		Config: json.RawMessage(`{
			"allowed_source_hosts":["open.feishu.cn"],
			"source_base_urls":{"open.feishu.cn":"https://open.feishu.cn"}
		}`),
		Skill: DiscoveredSkill{
			CanonicalURL: "https://skills.sh/open.feishu.cn/demo", ExternalID: "demo",
			Opaque: safeOpaque(skillsSHOpaque{Source: "open.feishu.cn", SkillID: "demo"}),
		},
	})
	if err != nil || len(bundle.Files) != 1 {
		t.Fatalf("Acquire() files=%d err=%v", len(bundle.Files), err)
	}
	if fetcher.callCount("open.feishu.cn") == 0 || fetcher.callCount("api.github.com") != 0 || fetcher.callCount("/api/download/") != 0 {
		t.Fatalf("mapped domain source used the wrong acquisition path: %#v", fetcher.calls)
	}
}

func TestSkillsSHRejectsUnmappedNonGitHubSource(t *testing.T) {
	adapter := NewSkillsSHAdapter(&scriptedFetcher{})
	_, err := adapter.Acquire(context.Background(), AcquireRequest{
		Config: json.RawMessage(`{"allowed_source_hosts":["cli.sentry.dev"]}`),
		Skill:  DiscoveredSkill{Opaque: safeOpaque(skillsSHOpaque{Source: "sentry/dev/catalog", SkillID: "demo"})},
	})
	if ErrorKindOf(err) != ErrorInvalidConfig || !strings.Contains(err.Error(), "source_base_urls") {
		t.Fatalf("Acquire() error = %v", err)
	}
}
