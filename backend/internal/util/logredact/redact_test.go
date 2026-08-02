package logredact

import (
	"strings"
	"testing"
)

func TestRedactText_JSONLike(t *testing.T) {
	in := `{"access_token":"ya29.a0AfH6SMDUMMY","refresh_token":"1//0gDUMMY","other":"ok"}`
	out := RedactText(in)
	if out == in {
		t.Fatalf("expected redaction, got unchanged")
	}
	if want := `"access_token":"***"`; !strings.Contains(out, want) {
		t.Fatalf("expected %q in %q", want, out)
	}
	if want := `"refresh_token":"***"`; !strings.Contains(out, want) {
		t.Fatalf("expected %q in %q", want, out)
	}
}

func TestRedactText_QueryLike(t *testing.T) {
	in := "access_token=ya29.a0AfH6SMDUMMY refresh_token=1//0gDUMMY"
	out := RedactText(in)
	if strings.Contains(out, "ya29") || strings.Contains(out, "1//0") {
		t.Fatalf("expected tokens redacted, got %q", out)
	}
}

func TestRedactText_EmbeddedPageLaunchCode(t *testing.T) {
	in := "https://pay.example.com/start?s2a_client_id=payment&s2a_launch_code=raw-one-time-code"
	out := RedactText(in)
	if strings.Contains(out, "raw-one-time-code") {
		t.Fatalf("expected embedded launch code redacted, got %q", out)
	}
	if !strings.Contains(out, "s2a_launch_code=***") {
		t.Fatalf("expected launch-code key redacted, got %q", out)
	}
}

func TestRedactText_EmbeddedPageLaunchURLInsideJSON(t *testing.T) {
	in := `{"launch_url":"https://pay.example.com/start?s2a_client_id=payment&s2a_launch_code=raw-one-time-code"}`
	out := RedactText(in)
	if strings.Contains(out, "raw-one-time-code") || strings.Contains(out, "pay.example.com") {
		t.Fatalf("expected full embedded launch URL redacted, got %q", out)
	}
	if !strings.Contains(out, `"launch_url":"***"`) {
		t.Fatalf("expected launch_url key redacted in %q", out)
	}
}

func TestRedactText_QuotaViewerDownloadPathInsideJSON(t *testing.T) {
	in := `{"download_path":"/api/v1/quota/releases/macos/latest/download?code=raw-one-time-code"}`
	out := RedactText(in)
	if strings.Contains(out, "raw-one-time-code") || strings.Contains(out, "/api/v1/quota/") {
		t.Fatalf("expected full quota viewer download path redacted, got %q", out)
	}
	if !strings.Contains(out, `"download_path":"***"`) {
		t.Fatalf("expected download-path key redacted in %q", out)
	}
}

func TestRedactText_GOCSPX(t *testing.T) {
	in := "client_secret=GOCSPX-your-client-secret"
	out := RedactText(in)
	if strings.Contains(out, "your-client-secret") {
		t.Fatalf("expected secret redacted, got %q", out)
	}
	if !strings.Contains(out, "client_secret=***") {
		t.Fatalf("expected key redacted, got %q", out)
	}
}

func TestRedactText_ExtraKeyCacheUsesNormalizedSortedKey(t *testing.T) {
	clearExtraTextPatternCache()

	out1 := RedactText("custom_secret=abc", "Custom_Secret", " custom_secret ")
	out2 := RedactText("custom_secret=xyz", "custom_secret")
	if !strings.Contains(out1, "custom_secret=***") {
		t.Fatalf("expected custom key redacted in first call, got %q", out1)
	}
	if !strings.Contains(out2, "custom_secret=***") {
		t.Fatalf("expected custom key redacted in second call, got %q", out2)
	}

	if got := countExtraTextPatternCacheEntries(); got != 1 {
		t.Fatalf("expected 1 cached pattern set, got %d", got)
	}
}

func TestRedactText_DefaultPathDoesNotUseExtraCache(t *testing.T) {
	clearExtraTextPatternCache()

	out := RedactText("access_token=abc")
	if !strings.Contains(out, "access_token=***") {
		t.Fatalf("expected default key redacted, got %q", out)
	}
	if got := countExtraTextPatternCacheEntries(); got != 0 {
		t.Fatalf("expected extra cache to remain empty, got %d", got)
	}
}

func clearExtraTextPatternCache() {
	extraTextPatternCache.Range(func(key, value any) bool {
		extraTextPatternCache.Delete(key)
		return true
	})
}

func countExtraTextPatternCacheEntries() int {
	count := 0
	extraTextPatternCache.Range(func(key, value any) bool {
		count++
		return true
	})
	return count
}
