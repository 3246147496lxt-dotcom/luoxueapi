package skillimport

import (
	"archive/tar"
	"context"
	"encoding/json"
	"errors"
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

func TestWellKnownAdapterV02ResolvesTarGZFromIndexAndTreatsURLAsSingleArtifact(t *testing.T) {
	artifact := makeSourceTarGZ(t, []tarFixtureEntry{
		{Header: tar.Header{Name: "SKILL.md"}, Data: []byte("---\nname: build-tam\ndescription: Build a TAM\n---\n")},
		{Header: tar.Header{Name: "skill-metadata.json"}, Data: []byte(`{"install_surface":"v1"}`)},
	})
	index := fmt.Sprintf(`{
  "$schema":"https://schemas.agentskills.io/discovery/0.2.0/schema.json",
  "skills":[{
    "name":"build-tam",
    "description":"Provider-Led Account And Contact Sourcing",
    "type":"archive",
    "url":"./archives/build-tam.tar.gz",
    "digest":"sha256:%s",
    "files":["SKILL.md","skill-metadata.json"]
  }]
}`, digestBytes(artifact))
	wantedArtifactURL := "https://code.deepline.com/.well-known/skills/archives/build-tam.tar.gz"
	fetcher := &scriptedFetcher{handler: func(target string, _ FetchOptions) (FetchResult, error) {
		switch target {
		case "https://code.deepline.com/.well-known/agent-skills/index.json":
			return FetchResult{}, NewAdapterError(WellKnownAdapterType, "read index", ErrorNotFound, errors.New("not found"))
		case "https://code.deepline.com/.well-known/skills/index.json":
			return fetchResult(target, "application/json", []byte(index)), nil
		case wantedArtifactURL:
			return fetchResult(target, "application/gzip", artifact), nil
		default:
			return FetchResult{}, fmt.Errorf("unexpected URL %s", target)
		}
	}}
	adapter := NewWellKnownAdapter(fetcher)
	page, err := adapter.Discover(context.Background(), DiscoverRequest{BaseURL: "https://code.deepline.com"})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].CanonicalURL != wantedArtifactURL {
		t.Fatalf("Deepline discovery = %#v", page)
	}
	bundle, err := adapter.Acquire(context.Background(), AcquireRequest{BaseURL: "https://code.deepline.com", Skill: page.Items[0]})
	if err != nil {
		t.Fatal(err)
	}
	if !bundle.IntegrityVerified || len(bundle.Files) != 2 || bundle.Files[0].Path != "SKILL.md" || bundle.Files[1].Path != "skill-metadata.json" {
		t.Fatalf("Deepline bundle = %#v", bundle)
	}
	if fetcher.callCount(wantedArtifactURL) != 1 || fetcher.callCount("/build-tam/SKILL.md") != 0 || fetcher.callCount("/build-tam/skill-metadata.json") != 0 {
		t.Fatalf("URL artifact must not be overlaid with files: calls=%#v", fetcher.calls)
	}
}

func TestWellKnownAdapterResolvesRelativeURLAgainstObservedIndexURL(t *testing.T) {
	artifact := makeSourceTarGZ(t, []tarFixtureEntry{{
		Header: tar.Header{Name: "SKILL.md"}, Data: []byte("---\nname: redirected\ndescription: Redirected\n---\n"),
	}})
	index := fmt.Sprintf(`{"skills":[{"name":"redirected","type":"archive","url":"./archives/redirected.tgz","digest":"sha256:%s"}]}`, digestBytes(artifact))
	observedIndexURL := "https://assets.example.test/catalog/v2/index.json"
	wantedArtifactURL := "https://assets.example.test/catalog/v2/archives/redirected.tgz"
	fetcher := &scriptedFetcher{handler: func(target string, _ FetchOptions) (FetchResult, error) {
		switch {
		case strings.HasSuffix(target, "/.well-known/agent-skills/index.json"):
			return FetchResult{}, NewAdapterError(WellKnownAdapterType, "read index", ErrorNotFound, errors.New("not found"))
		case strings.HasSuffix(target, "/.well-known/skills/index.json"):
			result := fetchResult(target, "application/json", []byte(index))
			result.Evidence.URL = observedIndexURL
			return result, nil
		case target == wantedArtifactURL:
			return fetchResult(target, "application/gzip", artifact), nil
		default:
			return FetchResult{}, fmt.Errorf("unexpected URL %s", target)
		}
	}}
	config := json.RawMessage(`{"allowed_hosts":["assets.example.test"]}`)
	adapter := NewWellKnownAdapter(fetcher)
	page, err := adapter.Discover(context.Background(), DiscoverRequest{Config: config, BaseURL: "https://skills.example.test"})
	if err != nil || len(page.Items) != 1 || page.Items[0].CanonicalURL != wantedArtifactURL {
		t.Fatalf("redirected discovery = %#v err=%v", page, err)
	}
	if _, err := adapter.Acquire(context.Background(), AcquireRequest{Config: config, BaseURL: "https://skills.example.test", Skill: page.Items[0]}); err != nil {
		t.Fatal(err)
	}
	if fetcher.callCount(wantedArtifactURL) != 1 {
		t.Fatalf("relative artifact did not use observed index URL: calls=%#v", fetcher.calls)
	}
}

func TestWellKnownAdapterRejectsArtifactDigestMismatch(t *testing.T) {
	artifact := makeSourceTarGZ(t, []tarFixtureEntry{{
		Header: tar.Header{Name: "SKILL.md"}, Data: []byte("---\nname: mismatch\ndescription: Mismatch\n---\n"),
	}})
	index := `{"skills":[{"name":"mismatch","type":"archive","url":"./archives/mismatch.tar.gz","digest":"sha256:0000000000000000000000000000000000000000000000000000000000000000"}]}`
	fetcher := &scriptedFetcher{handler: func(target string, _ FetchOptions) (FetchResult, error) {
		switch {
		case strings.HasSuffix(target, "/.well-known/agent-skills/index.json"):
			return FetchResult{}, NewAdapterError(WellKnownAdapterType, "read index", ErrorNotFound, errors.New("not found"))
		case strings.HasSuffix(target, "/.well-known/skills/index.json"):
			return fetchResult(target, "application/json", []byte(index)), nil
		case strings.HasSuffix(target, "/archives/mismatch.tar.gz"):
			return fetchResult(target, "application/gzip", artifact), nil
		default:
			return FetchResult{}, fmt.Errorf("unexpected URL %s", target)
		}
	}}
	adapter := NewWellKnownAdapter(fetcher)
	page, err := adapter.Discover(context.Background(), DiscoverRequest{BaseURL: "https://skills.example.test"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = adapter.Acquire(context.Background(), AcquireRequest{BaseURL: "https://skills.example.test", Skill: page.Items[0]})
	if ErrorKindOf(err) != ErrorIntegrity || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("digest mismatch error = %v", err)
	}
}

func TestWellKnownAdapterRejectsUnknownArtifactType(t *testing.T) {
	index := `{"skills":[{"name":"unknown","type":"rar","url":"./archives/unknown.rar"}]}`
	fetcher := &scriptedFetcher{handler: func(target string, _ FetchOptions) (FetchResult, error) {
		switch {
		case strings.HasSuffix(target, "/.well-known/agent-skills/index.json"):
			return FetchResult{}, NewAdapterError(WellKnownAdapterType, "read index", ErrorNotFound, errors.New("not found"))
		case strings.HasSuffix(target, "/.well-known/skills/index.json"):
			return fetchResult(target, "application/json", []byte(index)), nil
		case strings.HasSuffix(target, "/archives/unknown.rar"):
			return fetchResult(target, "application/octet-stream", []byte("rar")), nil
		default:
			return FetchResult{}, fmt.Errorf("unexpected URL %s", target)
		}
	}}
	adapter := NewWellKnownAdapter(fetcher)
	page, err := adapter.Discover(context.Background(), DiscoverRequest{BaseURL: "https://skills.example.test"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = adapter.Acquire(context.Background(), AcquireRequest{BaseURL: "https://skills.example.test", Skill: page.Items[0]})
	if ErrorKindOf(err) != ErrorInvalidSource || !strings.Contains(err.Error(), `unsupported artifact type "rar"`) {
		t.Fatalf("unknown type error = %v", err)
	}
	if fetcher.callCount("/archives/unknown.rar") != 0 {
		t.Fatalf("unknown artifact type must fail before download: calls=%#v", fetcher.calls)
	}
}

func TestWellKnownAdapterV02AcquiresSkillMDAndVerifiesDigest(t *testing.T) {
	skillMD := []byte("---\nname: simple\ndescription: Simple v0.2 skill\n---\n")
	index := fmt.Sprintf(`{"$schema":%q,"skills":[{"name":"simple","description":"Simple v0.2 skill","type":"skill-md","url":"./simple/SKILL.md","digest":"sha256:%s"}]}`,
		wellKnownSchemaV02, digestBytes(skillMD))
	fetcher := &scriptedFetcher{handler: func(target string, _ FetchOptions) (FetchResult, error) {
		switch {
		case strings.HasSuffix(target, "/.well-known/agent-skills/index.json"):
			return fetchResult(target, "application/json", []byte(index)), nil
		case strings.HasSuffix(target, "/.well-known/agent-skills/simple/SKILL.md"):
			return fetchResult(target, "text/markdown", skillMD), nil
		default:
			return FetchResult{}, fmt.Errorf("unexpected URL %s", target)
		}
	}}
	adapter := NewWellKnownAdapter(fetcher)
	page, err := adapter.Discover(context.Background(), DiscoverRequest{BaseURL: "https://skills.example.test"})
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := adapter.Acquire(context.Background(), AcquireRequest{BaseURL: "https://skills.example.test", Skill: page.Items[0]})
	if err != nil || !bundle.IntegrityVerified || len(bundle.Files) != 1 || bundle.Files[0].Path != "SKILL.md" {
		t.Fatalf("skill-md bundle = %#v err=%v", bundle, err)
	}
}

func TestWellKnownAdapterV02IgnoresExtensionFilesWhenURLExists(t *testing.T) {
	skillMD := []byte("---\nname: extension-files\ndescription: Extension files\n---\n")
	index := fmt.Sprintf(`{"$schema":%q,"skills":[{"name":"extension-files","description":"Extension files","type":"skill-md","url":"./extension-files/SKILL.md","digest":"sha256:%s","files":["SHOULD-NOT-FETCH.md"]}]}`,
		wellKnownSchemaV02, digestBytes(skillMD))
	fetcher := &scriptedFetcher{handler: func(target string, _ FetchOptions) (FetchResult, error) {
		switch {
		case strings.HasSuffix(target, "/.well-known/agent-skills/index.json"):
			return fetchResult(target, "application/json", []byte(index)), nil
		case strings.HasSuffix(target, "/extension-files/SKILL.md"):
			return fetchResult(target, "text/markdown", skillMD), nil
		default:
			return FetchResult{}, fmt.Errorf("unexpected URL %s", target)
		}
	}}
	adapter := NewWellKnownAdapter(fetcher)
	page, err := adapter.Discover(context.Background(), DiscoverRequest{BaseURL: "https://skills.example.test"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.Acquire(context.Background(), AcquireRequest{BaseURL: "https://skills.example.test", Skill: page.Items[0]}); err != nil {
		t.Fatal(err)
	}
	if fetcher.callCount("SHOULD-NOT-FETCH.md") != 0 {
		t.Fatalf("v0.2 extension files were fetched: calls=%#v", fetcher.calls)
	}
}

func TestWellKnownAdapterEnforcesSchemaVersionAndV02EntryContract(t *testing.T) {
	tests := map[string]string{
		"unknown schema": `{"$schema":"https://schemas.agentskills.io/discovery/9.9.9/schema.json","skills":[]}`,
		"missing type":   fmt.Sprintf(`{"$schema":%q,"skills":[{"name":"demo","description":"Demo","url":"demo.md","digest":"sha256:%s"}]}`, wellKnownSchemaV02, strings.Repeat("0", 64)),
		"unknown type":   fmt.Sprintf(`{"$schema":%q,"skills":[{"name":"demo","description":"Demo","type":"rar","url":"demo.rar","digest":"sha256:%s"}]}`, wellKnownSchemaV02, strings.Repeat("0", 64)),
		"missing url":    fmt.Sprintf(`{"$schema":%q,"skills":[{"name":"demo","description":"Demo","type":"skill-md","digest":"sha256:%s"}]}`, wellKnownSchemaV02, strings.Repeat("0", 64)),
		"missing digest": fmt.Sprintf(`{"$schema":%q,"skills":[{"name":"demo","description":"Demo","type":"skill-md","url":"demo.md"}]}`, wellKnownSchemaV02),
		"invalid digest": fmt.Sprintf(`{"$schema":%q,"skills":[{"name":"demo","description":"Demo","type":"skill-md","url":"demo.md","digest":"sha256:bad"}]}`, wellKnownSchemaV02),
		"missing description": fmt.Sprintf(`{"$schema":%q,"skills":[{"name":"demo","type":"skill-md","url":"demo.md","digest":"sha256:%s"}]}`,
			wellKnownSchemaV02, strings.Repeat("0", 64)),
		"invalid name": fmt.Sprintf(`{"$schema":%q,"skills":[{"name":"Bad_Name","description":"Demo","type":"skill-md","url":"demo.md","digest":"sha256:%s"}]}`,
			wellKnownSchemaV02, strings.Repeat("0", 64)),
	}
	for name, index := range tests {
		t.Run(name, func(t *testing.T) {
			fetcher := &scriptedFetcher{handler: func(target string, _ FetchOptions) (FetchResult, error) {
				return fetchResult(target, "application/json", []byte(index)), nil
			}}
			_, err := NewWellKnownAdapter(fetcher).Discover(context.Background(), DiscoverRequest{BaseURL: "https://skills.example.test"})
			if ErrorKindOf(err) != ErrorInvalidSource {
				t.Fatalf("schema-contract error = %v", err)
			}
			if len(fetcher.calls) != 1 {
				t.Fatalf("invalid index must fail before artifact/fallback fetch: calls=%#v", fetcher.calls)
			}
		})
	}
}

func TestWellKnownAdapterFallsBackFromVolcesSoftNotFoundEnvelope(t *testing.T) {
	// skills.volces.com has returned this JSON error envelope with HTTP 200 for
	// an unsupported well-known candidate. It is not an empty skill catalog.
	softNotFound := `{"ResponseMetadata":{"Action":"","Service":"skillhub","RequestId":"fixture","Error":{"Code":"NotFound","Message":"The requested API is not found."}},"Result":null}`
	legacyIndex := `{"skills":[{"name":"demo","description":"Legacy catalog entry","files":["SKILL.md"]}]}`
	fetcher := &scriptedFetcher{handler: func(target string, _ FetchOptions) (FetchResult, error) {
		switch {
		case strings.HasSuffix(target, "/.well-known/agent-skills/index.json"):
			return fetchResult(target, "application/json", []byte(softNotFound)), nil
		case strings.HasSuffix(target, "/.well-known/skills/index.json"):
			return fetchResult(target, "application/json", []byte(legacyIndex)), nil
		default:
			return FetchResult{}, fmt.Errorf("unexpected URL %s", target)
		}
	}}

	page, err := NewWellKnownAdapter(fetcher).Discover(context.Background(), DiscoverRequest{
		BaseURL: "https://skills.volces.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].ExternalID != "demo" || !strings.Contains(page.Items[0].CanonicalURL, "/.well-known/skills/demo") {
		t.Fatalf("fallback discovery = %#v", page)
	}
	if fetcher.callCount("/.well-known/agent-skills/index.json") != 1 || fetcher.callCount("/.well-known/skills/index.json") != 1 {
		t.Fatalf("fallback calls = %#v", fetcher.calls)
	}
}

func TestWellKnownAdapterTreatsExplicitEmptySkillsAsValidCatalog(t *testing.T) {
	fetcher := &scriptedFetcher{handler: func(target string, _ FetchOptions) (FetchResult, error) {
		if strings.HasSuffix(target, "/.well-known/agent-skills/index.json") {
			return fetchResult(target, "application/json", []byte(`{"skills":[]}`)), nil
		}
		return FetchResult{}, fmt.Errorf("legacy fallback must not be requested: %s", target)
	}}

	page, err := NewWellKnownAdapter(fetcher).Discover(context.Background(), DiscoverRequest{
		BaseURL: "https://skills.example.test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 0 || fetcher.callCount("/.well-known/skills/index.json") != 0 {
		t.Fatalf("empty catalog result=%#v calls=%#v", page, fetcher.calls)
	}
}

func TestWellKnownAdapterDoesNotMaskMalformedOrSchemaInvalidJSON(t *testing.T) {
	tests := map[string]string{
		"malformed":      `{"skills":[`,
		"missing skills": `{"message":"Not Found"}`,
		"null skills":    `{"skills":null}`,
		"object skills":  `{"skills":{}}`,
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			fetcher := &scriptedFetcher{handler: func(target string, _ FetchOptions) (FetchResult, error) {
				return fetchResult(target, "application/json", []byte(body)), nil
			}}
			_, err := NewWellKnownAdapter(fetcher).Discover(context.Background(), DiscoverRequest{
				BaseURL: "https://skills.example.test",
			})
			if ErrorKindOf(err) != ErrorInvalidSource {
				t.Fatalf("error = %v, want invalid_source", err)
			}
			if fetcher.callCount("/.well-known/skills/index.json") != 0 {
				t.Fatalf("invalid candidate was silently hidden by fallback: calls=%#v", fetcher.calls)
			}
		})
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
