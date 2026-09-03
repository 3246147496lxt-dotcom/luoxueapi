package skillimport

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestGitHubAdapterPackagesRootSkillWithoutWholeRepository(t *testing.T) {
	commit := strings.Repeat("a", 40)
	skillMD := []byte("---\nname: root-skill\ndescription: Root skill\n---\n\n# Root\n")
	license := []byte("MIT\n")
	script := []byte("#!/usr/bin/env python3\nprint('ok')\n")
	unrelated := []byte("must not be packaged\n")
	fetcher := &scriptedFetcher{handler: func(target string, _ FetchOptions) (FetchResult, error) {
		switch {
		case strings.Contains(target, "/commits/HEAD"):
			return fetchResult(target, "application/json", []byte(`{"sha":"`+commit+`"}`)), nil
		case strings.Contains(target, "/git/trees/"):
			body := fmt.Sprintf(`{"sha":%q,"truncated":false,"tree":[{"path":"SKILL.md","mode":"100644","type":"blob","sha":%q,"size":%d},{"path":"LICENSE","mode":"100644","type":"blob","sha":%q,"size":%d},{"path":"scripts/run.py","mode":"100755","type":"blob","sha":%q,"size":%d},{"path":"src/unrelated.txt","mode":"100644","type":"blob","sha":%q,"size":%d}]}`,
				commit, gitBlobObjectID(skillMD), len(skillMD), gitBlobObjectID(license), len(license), gitBlobObjectID(script), len(script), gitBlobObjectID(unrelated), len(unrelated))
			return fetchResult(target, "application/json", []byte(body)), nil
		case strings.HasSuffix(target, "/SKILL.md"):
			return fetchResult(target, "text/plain", skillMD), nil
		case strings.HasSuffix(target, "/LICENSE"):
			return fetchResult(target, "text/plain", license), nil
		case strings.HasSuffix(target, "/scripts/run.py"):
			return fetchResult(target, "text/plain", script), nil
		default:
			return FetchResult{}, fmt.Errorf("unexpected URL %s", target)
		}
	}}
	adapter := NewGitHubAdapter(fetcher, "")
	config := json.RawMessage(`{"repository":"owner/repo","skill_id":"root-skill"}`)
	page, err := adapter.Discover(context.Background(), DiscoverRequest{Config: config})
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("Discover() items=%d err=%v", len(page.Items), err)
	}
	bundle, err := adapter.Acquire(context.Background(), AcquireRequest{Config: config, Skill: page.Items[0]})
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Files) != 3 || bundle.Files[0].Path != "LICENSE" || bundle.Files[1].Path != "SKILL.md" || bundle.Files[2].Path != "scripts/run.py" || !bundle.Files[2].Executable {
		t.Fatalf("root bundle files = %#v", bundle.Files)
	}
	if fetcher.callCount("unrelated.txt") != 0 {
		t.Fatal("root Skill acquisition fetched unrelated repository content")
	}
}

func TestGitHubAdapterRejectsRootCompanionSymlink(t *testing.T) {
	commit := strings.Repeat("b", 40)
	skillMD := []byte("---\nname: root-skill\ndescription: Root skill\n---\n")
	fetcher := &scriptedFetcher{handler: func(target string, _ FetchOptions) (FetchResult, error) {
		switch {
		case strings.Contains(target, "/commits/HEAD"):
			return fetchResult(target, "application/json", []byte(`{"sha":"`+commit+`"}`)), nil
		case strings.Contains(target, "/git/trees/"):
			body := fmt.Sprintf(`{"sha":%q,"truncated":false,"tree":[{"path":"SKILL.md","mode":"100644","type":"blob","sha":%q,"size":%d},{"path":"assets/escape","mode":"120000","type":"blob","sha":%q,"size":4}]}`,
				commit, gitBlobObjectID(skillMD), len(skillMD), strings.Repeat("c", 40))
			return fetchResult(target, "application/json", []byte(body)), nil
		default:
			return FetchResult{}, fmt.Errorf("unexpected URL %s", target)
		}
	}}
	adapter := NewGitHubAdapter(fetcher, "")
	config := json.RawMessage(`{"repository":"owner/repo","skill_id":"root-skill"}`)
	page, err := adapter.Discover(context.Background(), DiscoverRequest{Config: config})
	if err != nil {
		t.Fatal(err)
	}
	_, err = adapter.Acquire(context.Background(), AcquireRequest{Config: config, Skill: page.Items[0]})
	if ErrorKindOf(err) != ErrorUnsafe {
		t.Fatalf("Acquire() error = %v, want unsafe", err)
	}
}

func TestGitHubAdapterDoesNotTreatRootLicenseSymlinkAsEvidence(t *testing.T) {
	tests := []struct {
		name      string
		config    json.RawMessage
		skillPath string
		relative  string
	}{
		{
			name:      "root skill",
			config:    json.RawMessage(`{"repository":"owner/repo","skill_id":"root-skill"}`),
			skillPath: "SKILL.md",
			relative:  "SKILL.md",
		},
		{
			name:      "directory skill",
			config:    json.RawMessage(`{"repository":"owner/repo","path":"skills/demo","skill_id":"demo"}`),
			skillPath: "skills/demo/SKILL.md",
			relative:  "SKILL.md",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			commit := strings.Repeat("d", 40)
			skillMD := []byte("---\nname: demo\ndescription: Demo skill\n---\n")
			fetcher := &scriptedFetcher{handler: func(target string, _ FetchOptions) (FetchResult, error) {
				switch {
				case strings.Contains(target, "/commits/HEAD"):
					return fetchResult(target, "application/json", []byte(`{"sha":"`+commit+`"}`)), nil
				case strings.Contains(target, "/git/trees/"):
					body := fmt.Sprintf(`{"sha":%q,"truncated":false,"tree":[{"path":%q,"mode":"100644","type":"blob","sha":%q,"size":%d},{"path":"LICENSE","mode":"120000","type":"blob","sha":%q,"size":14}]}`,
						commit, test.skillPath, gitBlobObjectID(skillMD), len(skillMD), strings.Repeat("e", 40))
					return fetchResult(target, "application/json", []byte(body)), nil
				case strings.HasSuffix(target, "/"+test.skillPath):
					return fetchResult(target, "text/plain", skillMD), nil
				default:
					return FetchResult{}, fmt.Errorf("unexpected URL %s", target)
				}
			}}
			adapter := NewGitHubAdapter(fetcher, "")
			page, err := adapter.Discover(context.Background(), DiscoverRequest{Config: test.config})
			if err != nil {
				t.Fatal(err)
			}
			bundle, err := adapter.Acquire(context.Background(), AcquireRequest{Config: test.config, Skill: page.Items[0]})
			if err != nil {
				t.Fatal(err)
			}
			if len(bundle.Files) != 1 || bundle.Files[0].Path != test.relative {
				t.Fatalf("bundle files = %#v", bundle.Files)
			}
			if len(bundle.LicenseEvidence) != 0 || fetcher.callCount("/LICENSE") != 0 {
				t.Fatalf("license evidence = %#v, LICENSE fetches = %d", bundle.LicenseEvidence, fetcher.callCount("/LICENSE"))
			}
		})
	}
}
