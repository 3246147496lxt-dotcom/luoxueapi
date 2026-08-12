package skillimport

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestManifestAdapterJSONPaginationAndZIPAcquisition(t *testing.T) {
	artifact := makeSourceZIP(map[string][]byte{
		"demo/SKILL.md": []byte("---\nname: demo\ndescription: Demo\n---\n\n# Demo\n"),
		"demo/LICENSE":  []byte("MIT\n"),
	})
	manifest := fmt.Sprintf(`{"schema_version":1,"skills":[{"namespace":"catalog","id":"demo","name":"Demo","slug":"demo","description":"Preserved","type":"zip","url":"artifacts/demo.zip","digest":"sha256:%s"},{"namespace":"catalog","id":"other","name":"Other","slug":"other","type":"skill_md","url":"other.md"}]}`, digestBytes(artifact))
	fetcher := &scriptedFetcher{handler: func(target string, _ FetchOptions) (FetchResult, error) {
		switch {
		case strings.HasSuffix(target, "/skills.json"):
			return fetchResult(target, "application/json", []byte(manifest)), nil
		case strings.HasSuffix(target, "/artifacts/demo.zip"):
			return fetchResult(target, "application/zip", artifact), nil
		default:
			return FetchResult{}, fmt.Errorf("unexpected URL %s", target)
		}
	}}
	adapter := NewManifestAdapter(fetcher)
	config := json.RawMessage(`{"url":"https://catalog.example.test/skills.json","format":"json"}`)
	selection := json.RawMessage(`{"start_rank":1,"limit":2,"page_size":1}`)
	first, err := adapter.Discover(context.Background(), DiscoverRequest{Config: config, Selection: selection})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != 1 || first.Items[0].StableKey() != "manifest\x00catalog\x00demo" || first.NextCursor == "" {
		t.Fatalf("first manifest page = %#v", first)
	}
	second, err := adapter.Discover(context.Background(), DiscoverRequest{Config: config, Selection: selection, Cursor: first.NextCursor})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Items) != 1 || second.Items[0].ExternalID != "other" || second.NextCursor != "" {
		t.Fatalf("second manifest page = %#v", second)
	}
	bundle, err := adapter.Acquire(context.Background(), AcquireRequest{Config: config, Skill: first.Items[0]})
	if err != nil {
		t.Fatal(err)
	}
	if !bundle.IntegrityVerified || len(bundle.Files) != 2 || bundle.Files[0].Path != "LICENSE" || bundle.Files[1].Path != "SKILL.md" {
		t.Fatalf("manifest bundle = %#v", bundle)
	}
}

func TestManifestAdapterParsesQuotedCSV(t *testing.T) {
	raw := []byte("namespace,id,name,slug,description,type,url,rank\nteam,demo,Demo,demo,\"Use commas, safely\",skill_md,skills/demo.md,7\n")
	entries, err := parseCSVManifest(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].ExternalID != "" || entries[0].ID != "demo" || entries[0].Description != "Use commas, safely" || entries[0].Rank != 7 {
		t.Fatalf("CSV entries = %#v", entries)
	}
}

func TestManifestAdapterRejectsMalformedCSVRankAndDuplicateHeaders(t *testing.T) {
	for _, raw := range [][]byte{
		[]byte("id,name,rank\ndemo,Demo,not-a-number\n"),
		[]byte("id,ID,name\ndemo,other,Demo\n"),
	} {
		if _, err := parseCSVManifest(raw); err == nil {
			t.Fatalf("parseCSVManifest(%q) accepted malformed data", raw)
		}
	}
}

func TestManifestAdapterRejectsTrailingJSON(t *testing.T) {
	for _, raw := range [][]byte{
		[]byte(`{"skills":[]} trailing`),
		[]byte(`[{"id":"demo"}] {"second":true}`),
	} {
		if _, err := parseJSONManifest(raw); err == nil {
			t.Fatalf("parseJSONManifest(%q) accepted trailing data", raw)
		}
	}
}

func TestManifestAdapterDirectZIPRequiresStableID(t *testing.T) {
	adapter := NewManifestAdapter(&scriptedFetcher{})
	if err := adapter.ValidateConfig(json.RawMessage(`{"url":"https://catalog.example.test/demo.zip"}`)); err == nil {
		t.Fatal("direct ZIP without skill_id must be rejected")
	}
}

func TestManifestAdapterAllowsUploadOnlySourceDefinition(t *testing.T) {
	adapter := NewManifestAdapter(&scriptedFetcher{})
	config := json.RawMessage(`{"upload_only":true,"allowed_hosts":["assets.example.test"]}`)
	if err := adapter.ValidateConfig(config); err != nil {
		t.Fatalf("upload-only source definition must be valid: %v", err)
	}
	_, err := adapter.Discover(context.Background(), DiscoverRequest{Config: config})
	if ErrorKindOf(err) != ErrorInvalidConfig || !strings.Contains(err.Error(), "run-scoped uploaded artifact") {
		t.Fatalf("unbound upload-only discovery error = %v", err)
	}
}

func TestManifestAdapterInlineZIPStaysInMemoryAndUsesArchiveSafety(t *testing.T) {
	artifact := makeSourceZIP(map[string][]byte{
		"wrapper/SKILL.md": []byte("---\nname: inline\ndescription: Inline\n---\n\n# Inline\n"),
		"wrapper/LICENSE":  []byte("MIT\n"),
	})
	encoded := base64.StdEncoding.EncodeToString(artifact)
	config, _ := json.Marshal(map[string]any{
		"format": "zip", "inline_data_base64": encoded,
		"digest": "sha256:" + digestBytes(artifact),
	})
	adapter := NewManifestAdapter(&scriptedFetcher{handler: func(target string, _ FetchOptions) (FetchResult, error) {
		return FetchResult{}, fmt.Errorf("inline ZIP must not fetch %s", target)
	}})
	page, err := adapter.Discover(context.Background(), DiscoverRequest{Config: config})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].StableKey() != "manifest\x00inline-manifest\x00inline" {
		t.Fatalf("inline discovery = %#v", page)
	}
	if page.Items[0].CanonicalURL != "" {
		t.Fatalf("inline ZIP without a declared upstream URL got origin %q", page.Items[0].CanonicalURL)
	}
	bundle, err := adapter.Acquire(context.Background(), AcquireRequest{Config: config, Skill: page.Items[0]})
	if err != nil {
		t.Fatal(err)
	}
	if !bundle.IntegrityVerified || len(bundle.Files) != 2 || bundle.UpstreamContentHash == "" {
		t.Fatalf("inline bundle = %#v", bundle)
	}
	if bundle.CanonicalURL != "" {
		t.Fatalf("inline ZIP bundle without a declared upstream URL got origin %q", bundle.CanonicalURL)
	}

	unsafeZIP := makeSourceZIP(map[string][]byte{"../SKILL.md": []byte("bad")})
	unsafeConfig, _ := json.Marshal(map[string]any{
		"format": "zip", "skill_id": "unsafe", "inline_data_base64": base64.StdEncoding.EncodeToString(unsafeZIP),
	})
	if err := adapter.ValidateConfig(unsafeConfig); err == nil {
		t.Fatal("unsafe inline ZIP must be rejected during configuration validation")
	}
}

func TestManifestAdapterInlineDataHasStrictEncodedLimit(t *testing.T) {
	raw := make([]byte, MaxInlineManifestBytes)
	encoded := base64.StdEncoding.EncodeToString(raw)
	if len(encoded) != maxInlineManifestEncodedBytes {
		t.Fatalf("encoded size = %d, want %d", len(encoded), maxInlineManifestEncodedBytes)
	}
	decoded, err := decodeInlineManifest(encoded)
	if err != nil || len(decoded) != MaxInlineManifestBytes {
		t.Fatalf("5 MiB raw inline data must decode: len=%d err=%v", len(decoded), err)
	}
	oversized := encoded + "AAAA"
	if _, err := decodeInlineManifest(oversized); err == nil || !strings.Contains(err.Error(), "encoded bytes") {
		t.Fatalf("oversized encoded inline error = %v", err)
	}
	// For this boundary, RawStdEncoding can encode Max+1 bytes in the same
	// number of characters as padded StdEncoding encodes Max bytes. The decoded
	// limit must therefore be enforced independently of the encoded limit.
	rawBoundary := base64.RawStdEncoding.EncodeToString(make([]byte, MaxInlineManifestBytes+1))
	if len(rawBoundary) > maxInlineManifestEncodedBytes {
		t.Fatalf("test precondition: raw boundary encoded size = %d", len(rawBoundary))
	}
	if _, err := decodeInlineManifest(rawBoundary); err == nil || !strings.Contains(err.Error(), "decode to between") {
		t.Fatalf("oversized decoded inline error = %v", err)
	}
}
