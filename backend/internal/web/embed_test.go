//go:build embed

package web

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

const adminEntryMarker = `<meta name="app-entry" content="admin" />`

func TestInjectSiteTitle(t *testing.T) {
	t.Run("replaces_title_with_site_name", func(t *testing.T) {
		html := []byte(`<html><head><title>Sub2API - AI API Gateway</title></head><body></body></html>`)
		settingsJSON := []byte(`{"site_name":"MyCustomSite"}`)

		result := injectSiteTitle(html, settingsJSON)

		assert.Contains(t, string(result), "<title>MyCustomSite - AI API Gateway</title>")
		assert.NotContains(t, string(result), "Sub2API")
	})

	t.Run("returns_unchanged_when_site_name_empty", func(t *testing.T) {
		html := []byte(`<html><head><title>Sub2API - AI API Gateway</title></head><body></body></html>`)
		settingsJSON := []byte(`{"site_name":""}`)

		result := injectSiteTitle(html, settingsJSON)

		assert.Equal(t, string(html), string(result))
	})

	t.Run("returns_unchanged_when_site_name_missing", func(t *testing.T) {
		html := []byte(`<html><head><title>Sub2API - AI API Gateway</title></head><body></body></html>`)
		settingsJSON := []byte(`{"other_field":"value"}`)

		result := injectSiteTitle(html, settingsJSON)

		assert.Equal(t, string(html), string(result))
	})

	t.Run("returns_unchanged_when_invalid_json", func(t *testing.T) {
		html := []byte(`<html><head><title>Sub2API - AI API Gateway</title></head><body></body></html>`)
		settingsJSON := []byte(`{invalid json}`)

		result := injectSiteTitle(html, settingsJSON)

		assert.Equal(t, string(html), string(result))
	})

	t.Run("returns_unchanged_when_no_title_tag", func(t *testing.T) {
		html := []byte(`<html><head></head><body></body></html>`)
		settingsJSON := []byte(`{"site_name":"MyCustomSite"}`)

		result := injectSiteTitle(html, settingsJSON)

		assert.Equal(t, string(html), string(result))
	})

	t.Run("returns_unchanged_when_title_has_attributes", func(t *testing.T) {
		// The function looks for "<title>" literally, so attributes are not supported
		// This is acceptable since index.html uses plain <title> without attributes
		html := []byte(`<html><head><title lang="en">Sub2API</title></head><body></body></html>`)
		settingsJSON := []byte(`{"site_name":"NewSite"}`)

		result := injectSiteTitle(html, settingsJSON)

		// Should return unchanged since <title> with attributes is not matched
		assert.Equal(t, string(html), string(result))
	})

	t.Run("escapes_html_in_site_name", func(t *testing.T) {
		html := []byte(`<html><head><title>Sub2API - AI API Gateway</title></head><body></body></html>`)
		settingsJSON := []byte(`{"site_name":"</title><script>alert(1)</script><title>"}`)

		result := injectSiteTitle(html, settingsJSON)

		assert.NotContains(t, string(result), "<script>")
		assert.Contains(t, string(result), "&lt;/title&gt;&lt;script&gt;alert(1)&lt;/script&gt;&lt;title&gt;")
	})

	t.Run("escapes_ampersand_in_site_name", func(t *testing.T) {
		html := []byte(`<html><head><title>Sub2API</title></head><body></body></html>`)
		settingsJSON := []byte(`{"site_name":"A&B"}`)

		result := injectSiteTitle(html, settingsJSON)

		assert.Contains(t, string(result), "<title>A&amp;B - AI API Gateway</title>")
	})

	t.Run("preserves_rest_of_html", func(t *testing.T) {
		html := []byte(`<html><head><meta charset="UTF-8"><title>Sub2API</title><script src="app.js"></script></head><body><div id="app"></div></body></html>`)
		settingsJSON := []byte(`{"site_name":"TestSite"}`)

		result := injectSiteTitle(html, settingsJSON)

		assert.Contains(t, string(result), `<meta charset="UTF-8">`)
		assert.Contains(t, string(result), `<script src="app.js"></script>`)
		assert.Contains(t, string(result), `<div id="app"></div>`)
		assert.Contains(t, string(result), "<title>TestSite - AI API Gateway</title>")
	})
}

func TestReplaceNoncePlaceholder(t *testing.T) {
	t.Run("replaces_single_placeholder", func(t *testing.T) {
		html := []byte(`<script nonce="__CSP_NONCE_VALUE__">console.log('test');</script>`)
		nonce := "abc123xyz"

		result := replaceNoncePlaceholder(html, nonce)

		expected := `<script nonce="abc123xyz">console.log('test');</script>`
		assert.Equal(t, expected, string(result))
	})

	t.Run("replaces_multiple_placeholders", func(t *testing.T) {
		html := []byte(`<script nonce="__CSP_NONCE_VALUE__">a</script><script nonce="__CSP_NONCE_VALUE__">b</script>`)
		nonce := "nonce123"

		result := replaceNoncePlaceholder(html, nonce)

		assert.Equal(t, 2, strings.Count(string(result), `nonce="nonce123"`))
		assert.NotContains(t, string(result), NonceHTMLPlaceholder)
	})

	t.Run("handles_empty_nonce", func(t *testing.T) {
		html := []byte(`<script nonce="__CSP_NONCE_VALUE__">test</script>`)
		nonce := ""

		result := replaceNoncePlaceholder(html, nonce)

		assert.Equal(t, `<script nonce="">test</script>`, string(result))
	})

	t.Run("no_placeholder_returns_unchanged", func(t *testing.T) {
		html := []byte(`<script>console.log('test');</script>`)
		nonce := "abc123"

		result := replaceNoncePlaceholder(html, nonce)

		assert.Equal(t, string(html), string(result))
	})

	t.Run("handles_empty_html", func(t *testing.T) {
		html := []byte(``)
		nonce := "abc123"

		result := replaceNoncePlaceholder(html, nonce)

		assert.Empty(t, result)
	})
}

func TestNonceHTMLPlaceholder(t *testing.T) {
	t.Run("constant_value", func(t *testing.T) {
		assert.Equal(t, "__CSP_NONCE_VALUE__", NonceHTMLPlaceholder)
	})
}

// mockSettingsProvider implements PublicSettingsProvider for testing
type mockSettingsProvider struct {
	settings any
	err      error
	called   int
}

func (m *mockSettingsProvider) GetPublicSettingsForInjection(ctx context.Context) (any, error) {
	m.called++
	return m.settings, m.err
}

type blockingSettingsProvider struct {
	mu           sync.Mutex
	first        any
	current      any
	calls        int
	firstStarted chan struct{}
	releaseFirst chan struct{}
}

func (p *blockingSettingsProvider) GetPublicSettingsForInjection(ctx context.Context) (any, error) {
	p.mu.Lock()
	p.calls++
	call := p.calls
	settings := p.current
	if call == 1 {
		settings = p.first
	}
	p.mu.Unlock()

	if call == 1 {
		close(p.firstStarted)
		select {
		case <-p.releaseFirst:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return settings, nil
}

func (p *blockingSettingsProvider) setCurrent(settings any) {
	p.mu.Lock()
	p.current = settings
	p.mu.Unlock()
}

func (p *blockingSettingsProvider) callCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.calls
}

func TestFrontendServer_InjectSettings(t *testing.T) {
	t.Run("injects_settings_with_nonce_placeholder", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]string{"key": "value"},
		}

		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		settingsJSON := []byte(`{"test":"data"}`)
		result := server.injectSettings(settingsJSON)

		// Should contain the script with nonce placeholder
		assert.Contains(t, string(result), `<script nonce="__CSP_NONCE_VALUE__">`)
		assert.Contains(t, string(result), `window.__APP_CONFIG__={"test":"data"};`)
		assert.Contains(t, string(result), `</script></head>`)
	})

	t.Run("injects_before_head_close", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]string{"key": "value"},
		}

		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		settingsJSON := []byte(`{}`)
		result := server.injectSettings(settingsJSON)

		// Script should be injected before </head>
		headCloseIndex := bytes.Index(result, []byte("</head>"))
		scriptIndex := bytes.Index(result, []byte(`<script nonce="`))

		assert.True(t, scriptIndex < headCloseIndex, "script should be before </head>")
	})

	t.Run("handles_complex_settings", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]any{
				"nested": map[string]any{
					"array": []int{1, 2, 3},
				},
			},
		}

		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		settingsJSON := []byte(`{"nested":{"array":[1,2,3]},"special":"<>&"}`)
		result := server.injectSettings(settingsJSON)

		assert.Contains(t, string(result), `window.__APP_CONFIG__={"nested":{"array":[1,2,3]},"special":"<>&"};`)
	})

	t.Run("selects_the_admin_html_entry_for_admin_routes", func(t *testing.T) {
		provider := &mockSettingsProvider{settings: map[string]string{"site_name": "落雪API"}}
		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		adminResult := server.injectSettingsForPath([]byte(`{"site_name":"落雪API"}`), "/admin/dashboard")
		userResult := server.injectSettingsForPath([]byte(`{"site_name":"落雪API"}`), "/dashboard")

		assert.Contains(t, string(adminResult), adminEntryMarker)
		assert.Contains(t, string(adminResult), `window.__APP_CONFIG__={"site_name":"落雪API"};`)
		assert.Contains(t, string(adminResult), `<meta name="robots" content="noindex, nofollow" />`)
		assert.Equal(t, 1, strings.Count(string(adminResult), `name="robots"`))
		assert.NotContains(t, string(userResult), adminEntryMarker)
	})

	t.Run("injects_route_specific_model_catalog_metadata", func(t *testing.T) {
		provider := &mockSettingsProvider{settings: map[string]any{
			"site_name":                    "落雪API",
			"public_model_catalog_enabled": true,
		}}
		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		result := server.injectSettingsForPath([]byte(`{"site_name":"落雪API","public_model_catalog_enabled":true}`), "/models.html")

		assert.Contains(t, string(result), "<title>模型广场 · 落雪API</title>")
		assert.Contains(t, string(result), `<meta name="description"`)
		assert.Contains(t, string(result), `<meta name="robots" content="index, follow`)
		assert.Contains(t, string(result), `<link rel="canonical" href="https://luoxueapi.cc/models.html" />`)
		assert.Contains(t, string(result), `<noscript><main`)
	})

	t.Run("injects_route_specific_quota_viewer_metadata", func(t *testing.T) {
		provider := &mockSettingsProvider{settings: map[string]any{"site_name": "落雪API"}}
		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		result := server.injectSettingsForPath([]byte(`{"site_name":"落雪API"}`), "/quota-viewer")
		body := string(result)

		assert.Contains(t, body, "<title>桌面积分查看器 · 落雪API</title>")
		assert.Contains(t, body, `content="在 macOS 和 Windows 桌面查看周剩余积分`)
		assert.Contains(t, body, `<meta name="robots" content="index, follow`)
		assert.Contains(t, body, `<link rel="canonical" href="https://luoxueapi.cc/quota-viewer" />`)
		assert.Contains(t, body, `<meta property="og:url" content="https://luoxueapi.cc/quota-viewer" />`)
		assert.Contains(t, body, `<noscript><main`)
	})

	t.Run("marks_quota_viewer_noindex_in_backend_mode", func(t *testing.T) {
		provider := &mockSettingsProvider{settings: map[string]any{
			"site_name":            "落雪API",
			"backend_mode_enabled": true,
		}}
		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		result := server.injectSettingsForPath(
			[]byte(`{"site_name":"落雪API","backend_mode_enabled":true}`),
			"/quota-viewer",
		)
		body := string(result)

		assert.Contains(t, body, `<meta name="robots" content="noindex, nofollow" />`)
		assert.NotContains(t, body, `rel="canonical"`)
	})

	t.Run("marks_disabled_model_catalog_noindex", func(t *testing.T) {
		provider := &mockSettingsProvider{settings: map[string]any{
			"site_name":                    "落雪API",
			"public_model_catalog_enabled": false,
		}}
		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		result := server.injectSettingsForPath([]byte(`{"site_name":"落雪API","public_model_catalog_enabled":false}`), "/models.html")
		body := string(result)

		assert.Contains(t, body, `<meta name="robots" content="noindex, nofollow" />`)
		assert.NotContains(t, body, `rel="canonical"`)
		assert.NotContains(t, body, `<noscript><main`)
	})

	t.Run("injects_route_specific_home_metadata", func(t *testing.T) {
		provider := &mockSettingsProvider{settings: map[string]string{"site_name": "落雪API"}}
		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		result := server.injectSettingsForPath([]byte(`{"site_name":"落雪API"}`), "/home")
		body := string(result)

		assert.Contains(t, body, "<title>落雪API · GPT API 接入与密钥管理</title>")
		assert.Contains(t, body, `content="落雪API 是面向开发者的 AI API 网关`)
		assert.Contains(t, body, `<meta name="robots" content="index, follow`)
		assert.Contains(t, body, `<link rel="canonical" href="https://luoxueapi.cc/home" />`)
		assert.Contains(t, body, `<meta property="og:url" content="https://luoxueapi.cc/home" />`)
		assert.Contains(t, body, `<noscript><main`)
		assert.NotContains(t, body, `content="noindex, nofollow"`)
	})

	t.Run("marks_home_noindex_in_backend_mode", func(t *testing.T) {
		provider := &mockSettingsProvider{settings: map[string]any{
			"site_name":            "落雪API",
			"backend_mode_enabled": true,
		}}
		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		result := server.injectSettingsForPath([]byte(`{"site_name":"落雪API","backend_mode_enabled":true}`), "/home")
		body := string(result)

		assert.Contains(t, body, `<meta name="robots" content="noindex, nofollow" />`)
		assert.NotContains(t, body, `rel="canonical"`)
	})

	t.Run("marks_application_routes_noindex", func(t *testing.T) {
		provider := &mockSettingsProvider{settings: map[string]string{"site_name": "落雪API"}}
		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		for _, path := range []string{"/login", "/register", "/dashboard", "/admin/dashboard", "/unknown"} {
			result := server.injectSettingsForPath([]byte(`{"site_name":"落雪API"}`), path)
			body := string(result)

			assert.Contains(t, body, `<meta name="robots" content="noindex, nofollow" />`, "path=%s", path)
			assert.NotContains(t, body, `rel="canonical"`, "path=%s", path)
		}
	})
}

func TestHTMLRouteCacheKey(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{path: "", want: ""},
		{path: "/", want: homeHTMLCacheKey},
		{path: "/home", want: homeHTMLCacheKey},
		{path: "/home/", want: homeHTMLCacheKey},
		{path: "/index.html", want: homeHTMLCacheKey},
		{path: "/models.html", want: modelCatalogHTMLCacheKey},
		{path: "/models.html/", want: modelCatalogHTMLCacheKey},
		{path: "/quota-viewer", want: quotaViewerHTMLCacheKey},
		{path: "/quota-viewer/", want: quotaViewerHTMLCacheKey},
		{path: "/dashboard", want: noIndexHTMLCacheKey},
		{path: "/admin/dashboard", want: noIndexHTMLCacheKey},
		{path: "/not-found", want: noIndexHTMLCacheKey},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Equal(t, tt.want, htmlRouteCacheKey(tt.path))
		})
	}
}

func TestFrontendIndexFilePath(t *testing.T) {
	tests := []struct {
		requestPath string
		want        string
	}{
		{requestPath: "/", want: userFrontendIndexFilePath},
		{requestPath: "/dashboard", want: userFrontendIndexFilePath},
		{requestPath: "/administrator", want: userFrontendIndexFilePath},
		{requestPath: "/admin", want: adminFrontendIndexFilePath},
		{requestPath: "/admin/", want: adminFrontendIndexFilePath},
		{requestPath: "/admin/dashboard", want: adminFrontendIndexFilePath},
		{requestPath: "/admin/index.html", want: adminFrontendIndexFilePath},
	}

	for _, tt := range tests {
		t.Run(tt.requestPath, func(t *testing.T) {
			assert.Equal(t, tt.want, frontendIndexFilePath(tt.requestPath))
		})
	}
}

func TestApplyRouteIndexingHeaders(t *testing.T) {
	indexable := make(http.Header)
	applyRouteIndexingHeaders(indexable, homeHTMLCacheKey)
	assert.Empty(t, indexable.Get("X-Robots-Tag"))

	private := make(http.Header)
	applyRouteIndexingHeaders(private, noIndexHTMLCacheKey)
	assert.Equal(t, "noindex, nofollow", private.Get("X-Robots-Tag"))
}

func TestInjectNoIndexMetadata(t *testing.T) {
	t.Run("adds_noindex_when_missing", func(t *testing.T) {
		html := []byte(`<html><head><title>Private</title></head><body></body></html>`)

		result := injectNoIndexMetadata(html)

		assert.Contains(t, string(result), `<meta name="robots" content="noindex, nofollow" />`)
	})

	t.Run("keeps_an_existing_noindex_meta_without_duplication", func(t *testing.T) {
		html := []byte(`<html><head><META content='nofollow, noindex' NAME='robots'></head><body></body></html>`)

		result := injectNoIndexMetadata(html)

		assert.Equal(t, html, result)
		assert.Len(t, metaTagPattern.FindAll(result, -1), 1)
	})

	t.Run("does_not_treat_an_indexable_robots_meta_as_noindex", func(t *testing.T) {
		html := []byte(`<html><head><meta name="robots" content="index, follow" /></head><body></body></html>`)

		result := injectNoIndexMetadata(html)

		assert.Contains(t, string(result), `<meta name="robots" content="noindex, nofollow" />`)
		assert.Len(t, metaTagPattern.FindAll(result, -1), 2)
	})
}

func TestFrontendServer_ServeIndexHTML(t *testing.T) {
	t.Run("serves_html_with_nonce", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]string{"test": "value"},
		}

		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		// Create a gin context with nonce
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

		// Set nonce in context (simulating SecurityHeaders middleware)
		testNonce := "test-nonce-12345"
		c.Set(middleware.CSPNonceKey, testNonce)

		server.serveIndexHTML(c)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "text/html")

		body := w.Body.String()
		// Nonce placeholder should be replaced
		assert.NotContains(t, body, NonceHTMLPlaceholder)
		assert.Contains(t, body, `nonce="`+testNonce+`"`)
	})

	t.Run("caches_html_content", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]string{"test": "value"},
		}

		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		// First request
		w1 := httptest.NewRecorder()
		c1, _ := gin.CreateTestContext(w1)
		c1.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c1.Set(middleware.CSPNonceKey, "nonce1")

		server.serveIndexHTML(c1)
		assert.Equal(t, 1, provider.called)

		// Second request - should use cache
		w2 := httptest.NewRecorder()
		c2, _ := gin.CreateTestContext(w2)
		c2.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c2.Set(middleware.CSPNonceKey, "nonce2")

		server.serveIndexHTML(c2)
		// Settings provider should not be called again
		assert.Equal(t, 1, provider.called)

		// But nonce should be different
		assert.Contains(t, w2.Body.String(), `nonce="nonce2"`)
	})

	t.Run("sets_etag_header", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]string{"test": "value"},
		}

		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Set(middleware.CSPNonceKey, "nonce123")

		server.serveIndexHTML(c)

		etag := w.Header().Get("ETag")
		assert.NotEmpty(t, etag)
		assert.True(t, strings.HasPrefix(etag, `"`))
		assert.True(t, strings.HasSuffix(etag, `"`))
	})

	t.Run("returns_fresh_nonce_body_for_matching_etag", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]string{"test": "value"},
		}

		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		// Model SecurityHeaders issuing a fresh nonce for every response.
		nonceSequence := 0
		router := gin.New()
		router.Use(func(c *gin.Context) {
			nonceSequence++
			c.Set(middleware.CSPNonceKey, fmt.Sprintf("test-nonce-%d", nonceSequence))
			c.Next()
		})
		router.Use(server.Middleware())

		// First request to populate cache and get ETag
		w1 := httptest.NewRecorder()
		req1 := httptest.NewRequest(http.MethodGet, "/", nil)
		router.ServeHTTP(w1, req1)
		etag := w1.Header().Get("ETag")
		require.NotEmpty(t, etag)

		// Second request with If-None-Match
		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest(http.MethodGet, "/", nil)
		req2.Header.Set("If-None-Match", etag)
		router.ServeHTTP(w2, req2)

		assert.Equal(t, http.StatusOK, w2.Code)
		assert.Equal(t, etag, w2.Header().Get("ETag"))
		assert.Equal(t, "no-cache", w2.Header().Get("Cache-Control"))
		assert.Contains(t, w2.Body.String(), `nonce="test-nonce-2"`)
		assert.NotContains(t, w2.Body.String(), `nonce="test-nonce-1"`)
	})

	t.Run("sets_cache_control_header", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]string{"test": "value"},
		}

		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Set(middleware.CSPNonceKey, "nonce123")

		server.serveIndexHTML(c)

		assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
	})

	t.Run("fallback_on_settings_error", func(t *testing.T) {
		provider := &mockSettingsProvider{
			err: context.DeadlineExceeded,
		}

		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		// Invalidate cache to force settings fetch
		server.InvalidateCache()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Set(middleware.CSPNonceKey, "nonce123")

		server.serveIndexHTML(c)

		// Should still return 200 with route-level metadata.
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
		assert.Contains(t, w.Body.String(), `<link rel="canonical" href="https://luoxueapi.cc/home" />`)
	})

	t.Run("isolates_indexable_and_noindex_route_caches", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]any{
				"site_name":                    "落雪API",
				"public_model_catalog_enabled": true,
			},
		}

		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		request := func(path string) *httptest.ResponseRecorder {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, path, nil)
			c.Set(middleware.CSPNonceKey, "nonce")
			server.serveIndexHTML(c)
			return w
		}

		home := request("/home")
		private := request("/dashboard")
		models := request("/models.html")
		quotaViewer := request("/quota-viewer")
		homeAgain := request("/")

		assert.Contains(t, home.Body.String(), `<link rel="canonical" href="https://luoxueapi.cc/home" />`)
		assert.Empty(t, home.Header().Get("X-Robots-Tag"))
		assert.Contains(t, private.Body.String(), `content="noindex, nofollow"`)
		assert.Equal(t, "noindex, nofollow", private.Header().Get("X-Robots-Tag"))
		assert.Contains(t, models.Body.String(), `<link rel="canonical" href="https://luoxueapi.cc/models.html" />`)
		assert.Contains(t, quotaViewer.Body.String(), `<link rel="canonical" href="https://luoxueapi.cc/quota-viewer" />`)
		assert.Empty(t, quotaViewer.Header().Get("X-Robots-Tag"))
		assert.Contains(t, homeAgain.Body.String(), `<link rel="canonical" href="https://luoxueapi.cc/home" />`)
		assert.Equal(t, 4, provider.called)
	})

	t.Run("isolates_user_and_admin_entry_caches", func(t *testing.T) {
		provider := &mockSettingsProvider{settings: map[string]string{"site_name": "落雪API"}}
		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		request := func(path, nonce string) *httptest.ResponseRecorder {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, path, nil)
			c.Set(middleware.CSPNonceKey, nonce)
			server.serveIndexHTML(c)
			return w
		}

		user := request("/dashboard", "user-nonce")
		admin := request("/admin/dashboard", "admin-nonce")
		adminAgain := request("/admin/users", "admin-second-nonce")

		assert.NotContains(t, user.Body.String(), adminEntryMarker)
		assert.Contains(t, admin.Body.String(), adminEntryMarker)
		assert.Contains(t, admin.Body.String(), `nonce="admin-nonce"`)
		assert.Contains(t, adminAgain.Body.String(), `nonce="admin-second-nonce"`)
		assert.NotEqual(t, user.Header().Get("ETag"), admin.Header().Get("ETag"))
		assert.Equal(t, 2, provider.called)
	})
}

func TestFrontendServer_InvalidateCache(t *testing.T) {
	t.Run("invalidates_cache", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]string{"test": "value"},
		}

		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		// First request to populate cache
		w1 := httptest.NewRecorder()
		c1, _ := gin.CreateTestContext(w1)
		c1.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c1.Set(middleware.CSPNonceKey, "nonce1")

		server.serveIndexHTML(c1)
		assert.Equal(t, 1, provider.called)

		// Invalidate cache
		server.InvalidateCache()

		// Update settings
		provider.settings = map[string]string{"test": "new_value"}

		// Second request should fetch new settings
		w2 := httptest.NewRecorder()
		c2, _ := gin.CreateTestContext(w2)
		c2.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c2.Set(middleware.CSPNonceKey, "nonce2")

		server.serveIndexHTML(c2)
		assert.Equal(t, 2, provider.called)
	})

	t.Run("old_etag_is_replaced_after_invalidation", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]bool{"skill_marketplace_enabled": true},
		}
		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		request := func(etag string) *httptest.ResponseRecorder {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			if etag != "" {
				c.Request.Header.Set("If-None-Match", etag)
			}
			c.Set(middleware.CSPNonceKey, "nonce")
			server.serveIndexHTML(c)
			return w
		}

		first := request("")
		oldETag := first.Header().Get("ETag")
		require.NotEmpty(t, oldETag)
		require.Contains(t, first.Body.String(), `"skill_marketplace_enabled":true`)

		provider.settings = map[string]bool{"skill_marketplace_enabled": false}
		server.InvalidateCache()
		refreshed := request(oldETag)

		assert.Equal(t, http.StatusOK, refreshed.Code)
		assert.NotEqual(t, oldETag, refreshed.Header().Get("ETag"))
		assert.Contains(t, refreshed.Body.String(), `"skill_marketplace_enabled":false`)
	})

	t.Run("invalidation_rejects_an_inflight_stale_render", func(t *testing.T) {
		provider := &blockingSettingsProvider{
			first:        map[string]bool{"skill_marketplace_enabled": true},
			current:      map[string]bool{"skill_marketplace_enabled": true},
			firstStarted: make(chan struct{}),
			releaseFirst: make(chan struct{}),
		}
		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		firstDone := make(chan *httptest.ResponseRecorder, 1)
		go func() {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			c.Set(middleware.CSPNonceKey, "nonce-old")
			server.serveIndexHTML(c)
			firstDone <- w
		}()

		select {
		case <-provider.firstStarted:
		case <-time.After(time.Second):
			t.Fatal("first settings read did not start")
		}
		provider.setCurrent(map[string]bool{"skill_marketplace_enabled": false})
		server.InvalidateCache()
		close(provider.releaseFirst)

		var first *httptest.ResponseRecorder
		select {
		case first = <-firstDone:
		case <-time.After(time.Second):
			t.Fatal("first settings read did not finish")
		}
		require.Contains(t, first.Body.String(), `"skill_marketplace_enabled":true`)
		assert.Empty(t, first.Header().Get("ETag"))
		assert.Nil(t, server.cache.GetForKey(homeHTMLCacheKey))

		second := httptest.NewRecorder()
		secondContext, _ := gin.CreateTestContext(second)
		secondContext.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		secondContext.Set(middleware.CSPNonceKey, "nonce-new")
		server.serveIndexHTML(secondContext)

		assert.Contains(t, second.Body.String(), `"skill_marketplace_enabled":false`)
		assert.NotEmpty(t, second.Header().Get("ETag"))
		assert.Equal(t, 2, provider.callCount())
	})

	t.Run("handles_nil_server", func(t *testing.T) {
		var server *FrontendServer
		// Should not panic
		assert.NotPanics(t, func() {
			server.InvalidateCache()
		})
	})

	t.Run("handles_nil_cache", func(t *testing.T) {
		server := &FrontendServer{}
		// Should not panic
		assert.NotPanics(t, func() {
			server.InvalidateCache()
		})
	})

	t.Run("invalidates_user_and_admin_caches", func(t *testing.T) {
		provider := &mockSettingsProvider{settings: map[string]string{"test": "value"}}
		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		server.cache.SetForKey(noIndexHTMLCacheKey, []byte("user"), []byte(`{}`))
		server.adminCache.SetForKey(noIndexHTMLCacheKey, []byte("admin"), []byte(`{}`))
		server.InvalidateCache()

		assert.Nil(t, server.cache.GetForKey(noIndexHTMLCacheKey))
		assert.Nil(t, server.adminCache.GetForKey(noIndexHTMLCacheKey))
	})
}

func TestOverrideFilesNeverReceiveImmutableCacheHeaders(t *testing.T) {
	t.Parallel()

	overrideDir := t.TempDir()
	cleanPath := "assets/index-AbCd1234.js"
	filePath := filepath.Join(overrideDir, cleanPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(filePath), 0o755))
	require.NoError(t, os.WriteFile(filePath, []byte("override"), 0o644))

	t.Run("frontend_server_override", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/"+cleanPath, nil)

		server := &FrontendServer{overrideDir: overrideDir}
		assert.True(t, server.tryServeOverride(c, cleanPath))
		assert.Empty(t, w.Header().Get("Cache-Control"))
	})

	t.Run("legacy_override", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/"+cleanPath, nil)

		assert.True(t, tryServeOverrideFile(c, overrideDir, cleanPath))
		assert.Empty(t, w.Header().Get("Cache-Control"))
	})
}

func TestFrontendServer_Middleware(t *testing.T) {
	t.Run("skips_api_routes", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]string{"test": "value"},
		}

		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		apiPaths := []string{
			"/api/v1/users",
			"/models",
			"/v1/models",
			"/v1beta/chat",
			"/backend-api/codex/responses",
			"/backend-api/codex/responses/compact",
			"/antigravity/test",
			"/setup/init",
			"/health",
			"/livez",
			"/readyz",
			"/responses",
			"/responses/compact",
		}

		for _, path := range apiPaths {
			t.Run(path, func(t *testing.T) {
				router := gin.New()
				router.Use(server.Middleware())
				nextCalled := false
				router.GET(path, func(c *gin.Context) {
					nextCalled = true
					c.String(http.StatusOK, "ok")
				})

				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, path, nil)
				router.ServeHTTP(w, req)

				assert.True(t, nextCalled, "next handler should be called for API route")
			})
		}
	})

	t.Run("skips_responses_compact_post_routes", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]string{"test": "value"},
		}

		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		router := gin.New()
		router.Use(server.Middleware())
		nextCalled := false
		router.POST("/responses/compact", func(c *gin.Context) {
			nextCalled = true
			c.String(http.StatusOK, `{"ok":true}`)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/responses/compact", strings.NewReader(`{"model":"gpt-5"}`))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.True(t, nextCalled, "next handler should be called for compact API route")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, `{"ok":true}`, w.Body.String())
	})

	t.Run("skips_alpha_search_post_route", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]string{"test": "value"},
		}

		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		router := gin.New()
		router.Use(server.Middleware())
		nextCalled := false
		router.POST("/alpha/search", func(c *gin.Context) {
			nextCalled = true
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/alpha/search", strings.NewReader(`{"model":"gpt-5.6-sol"}`))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.True(t, nextCalled, "next handler should be called for alpha search API route")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, `{"ok":true}`, w.Body.String())
	})

	t.Run("serves_index_for_spa_routes", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]string{"test": "value"},
		}

		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set(middleware.CSPNonceKey, "test-nonce")
			c.Next()
		})
		router.Use(server.Middleware())

		spaPaths := []string{
			"/",
			"/dashboard",
			"/users/123",
			"/settings/profile",
		}

		for _, path := range spaPaths {
			t.Run(path, func(t *testing.T) {
				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, path, nil)
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)
				assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
			})
		}
	})

	t.Run("serves_admin_entry_for_admin_roots_and_deep_links", func(t *testing.T) {
		provider := &mockSettingsProvider{settings: map[string]string{"site_name": "落雪API"}}
		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set(middleware.CSPNonceKey, "admin-route-nonce")
			c.Next()
		})
		router.Use(server.Middleware())

		for _, requestPath := range []string{
			"/admin",
			"/admin/",
			"/admin/index.html",
			"/admin/dashboard",
			"/admin/users/123/chat-history",
		} {
			t.Run(requestPath, func(t *testing.T) {
				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, requestPath, nil)
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)
				assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
				assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
				assert.Equal(t, "noindex, nofollow", w.Header().Get("X-Robots-Tag"))
				assert.Contains(t, w.Body.String(), adminEntryMarker)
				assert.Contains(t, w.Body.String(), `nonce="admin-route-nonce"`)
			})
		}

		userWriter := httptest.NewRecorder()
		router.ServeHTTP(userWriter, httptest.NewRequest(http.MethodGet, "/dashboard", nil))
		assert.NotContains(t, userWriter.Body.String(), adminEntryMarker)
	})

	t.Run("serves_documentation_site_and_its_deep_links", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]string{"site_name": "Configured main site"},
		}

		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		router := gin.New()
		router.Use(server.Middleware())

		redirectWriter := httptest.NewRecorder()
		redirectRequest := httptest.NewRequest(http.MethodGet, "/tutorial-docs?source=nav", nil)
		router.ServeHTTP(redirectWriter, redirectRequest)
		assert.Equal(t, http.StatusMovedPermanently, redirectWriter.Code)
		assert.Equal(t, "/tutorial-docs/?source=nav", redirectWriter.Header().Get("Location"))

		for _, requestPath := range []string{"/tutorial-docs/", "/tutorial-docs/orders"} {
			t.Run(requestPath, func(t *testing.T) {
				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, requestPath, nil)
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)
				assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
				assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
				assert.Contains(t, w.Body.String(), `href="https://luoxueapi.cc/tutorial-docs/"`)
				assert.Contains(t, w.Body.String(), "/tutorial-docs/assets/")
				assert.NotContains(t, w.Body.String(), "Configured main site")
			})
		}

		missingAssetWriter := httptest.NewRecorder()
		missingAssetRequest := httptest.NewRequest(http.MethodGet, "/tutorial-docs/assets/missing.js", nil)
		router.ServeHTTP(missingAssetWriter, missingAssetRequest)
		assert.Equal(t, http.StatusNotFound, missingAssetWriter.Code)
		assert.NotContains(t, missingAssetWriter.Body.String(), "<!doctype html>")
	})

	t.Run("serves_documentation_fingerprinted_assets_with_immutable_cache", func(t *testing.T) {
		provider := &mockSettingsProvider{settings: map[string]string{"test": "value"}}
		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		entries, err := fs.ReadDir(server.distFS, "tutorial-docs/assets")
		require.NoError(t, err)
		fingerprintedPath := ""
		for _, entry := range entries {
			candidate := "tutorial-docs/assets/" + entry.Name()
			if !entry.IsDir() && isFingerprintedEmbeddedAssetPath(candidate) {
				fingerprintedPath = candidate
				break
			}
		}
		require.NotEmpty(t, fingerprintedPath)

		router := gin.New()
		router.Use(server.Middleware())
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/"+fingerprintedPath, nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, staticAssetsCacheControl, w.Header().Get("Cache-Control"))
	})

	t.Run("serves_static_files", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]string{"test": "value"},
		}

		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		router := gin.New()
		router.Use(server.Middleware())

		// Request for existing static file
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/logo.png", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "image/png")
		assert.Empty(t, w.Header().Get("Cache-Control"))

		entries, err := fs.ReadDir(server.distFS, "assets")
		require.NoError(t, err)
		fingerprintedPath := ""
		for _, entry := range entries {
			candidate := "assets/" + entry.Name()
			if !entry.IsDir() && isFingerprintedEmbeddedAssetPath(candidate) {
				fingerprintedPath = candidate
				break
			}
		}
		require.NotEmpty(t, fingerprintedPath)

		assetWriter := httptest.NewRecorder()
		assetRequest := httptest.NewRequest(http.MethodGet, "/"+fingerprintedPath, nil)
		router.ServeHTTP(assetWriter, assetRequest)

		assert.Equal(t, http.StatusOK, assetWriter.Code)
		assert.Equal(t, staticAssetsCacheControl, assetWriter.Header().Get("Cache-Control"))
	})

	t.Run("serves_search_engine_files_instead_of_spa_html", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]string{"site_name": "落雪API"},
		}

		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		router := gin.New()
		router.Use(server.Middleware())

		robotsWriter := httptest.NewRecorder()
		robotsRequest := httptest.NewRequest(http.MethodGet, "/robots.txt", nil)
		router.ServeHTTP(robotsWriter, robotsRequest)

		assert.Equal(t, http.StatusOK, robotsWriter.Code)
		assert.Contains(t, robotsWriter.Header().Get("Content-Type"), "text/plain")
		assert.Contains(t, robotsWriter.Body.String(), "User-agent: *")
		assert.Contains(t, robotsWriter.Body.String(), "Sitemap: https://luoxueapi.cc/sitemap.xml")
		assert.NotContains(t, robotsWriter.Body.String(), "<!doctype html>")

		sitemapWriter := httptest.NewRecorder()
		sitemapRequest := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
		router.ServeHTTP(sitemapWriter, sitemapRequest)

		assert.Equal(t, http.StatusOK, sitemapWriter.Code)
		assert.Contains(t, sitemapWriter.Header().Get("Content-Type"), "xml")
		assert.Contains(t, sitemapWriter.Body.String(), `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
		assert.Contains(t, sitemapWriter.Body.String(), `<loc>https://luoxueapi.cc/home</loc>`)
		assert.Contains(t, sitemapWriter.Body.String(), `<loc>https://luoxueapi.cc/quota-viewer</loc>`)
		assert.NotContains(t, sitemapWriter.Body.String(), `<loc>https://luoxueapi.cc/models.html</loc>`)
		assert.NotContains(t, sitemapWriter.Body.String(), "<!doctype html>")
	})
}

func TestEmbeddedFrontendBypassesBareVideoAPIRoutes(t *testing.T) {
	for _, path := range []string{
		"/videos/generations",
		"/videos/edits",
		"/videos/extensions",
		"/videos/request-123",
	} {
		require.True(t, shouldBypassEmbeddedFrontend(path), "path=%s", path)
	}
}

func TestEmbeddedFrontendServesEveryBuiltAssetExactly(t *testing.T) {
	provider := &mockSettingsProvider{settings: map[string]string{"site_name": "Asset gate"}}
	server, err := NewFrontendServer(provider)
	require.NoError(t, err)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.CSPNonceKey, "recursive-asset-gate")
		c.Next()
	})
	router.Use(server.Middleware())

	entrypoints := map[string]bool{
		"index.html":               true,
		"admin/index.html":         true,
		"tutorial-docs/index.html": true,
	}
	seenEntrypoints := make(map[string]bool, len(entrypoints))
	assetCount := 0

	err = fs.WalkDir(server.distFS, ".", func(path string, entry fs.DirEntry, walkErr error) error {
		require.NoError(t, walkErr)
		if entry.IsDir() {
			return nil
		}
		require.False(t, strings.HasPrefix(filepath.Base(path), "."),
			"hidden build metadata must not enter the embedded bundle: %s", path)

		body, readErr := fs.ReadFile(server.distFS, path)
		require.NoError(t, readErr, path)
		require.NotEmpty(t, body, path)

		writer := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/"+path, nil)
		router.ServeHTTP(writer, request)
		require.Equal(t, http.StatusOK, writer.Code, path)
		require.NotEmpty(t, writer.Body.Bytes(), path)

		contentType, _, parseErr := mime.ParseMediaType(writer.Header().Get("Content-Type"))
		require.NoError(t, parseErr, path)
		if entrypoints[path] {
			require.Equal(t, "text/html", contentType, path)
			seenEntrypoints[path] = true
			return nil
		}

		expectedType := mime.TypeByExtension(filepath.Ext(path))
		require.NotEmpty(t, expectedType, "no MIME mapping for %s", path)
		expectedType, _, parseErr = mime.ParseMediaType(expectedType)
		require.NoError(t, parseErr, path)
		require.NotEqual(t, "text/html", contentType, path)
		require.Equal(t, expectedType, contentType, path)
		require.Equal(t, sha256.Sum256(body), sha256.Sum256(writer.Body.Bytes()), path)
		assetCount++
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, entrypoints, seenEntrypoints)
	require.Positive(t, assetCount)
}

func TestNewFrontendServer(t *testing.T) {
	t.Run("creates_server_successfully", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]string{"test": "value"},
		}

		server, err := NewFrontendServer(provider)

		require.NoError(t, err)
		assert.NotNil(t, server)
		assert.NotNil(t, server.distFS)
		assert.NotNil(t, server.fileServer)
		assert.NotNil(t, server.baseHTML)
		assert.NotNil(t, server.adminBaseHTML)
		assert.NotNil(t, server.cache)
		assert.NotNil(t, server.adminCache)
		assert.Equal(t, provider, server.settings)
	})

	t.Run("reads_base_html", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]string{"test": "value"},
		}

		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		assert.NotEmpty(t, server.baseHTML)
		assert.Contains(t, string(server.baseHTML), "<!doctype html>")
		assert.NotEmpty(t, server.adminBaseHTML)
		assert.Contains(t, string(server.adminBaseHTML), adminEntryMarker)
	})
}

func TestHasEmbeddedFrontend(t *testing.T) {
	t.Run("returns_true_when_frontend_embedded", func(t *testing.T) {
		result := HasEmbeddedFrontend()
		assert.True(t, result)
	})
}

// Tests for legacy ServeEmbeddedFrontend function
func TestServeEmbeddedFrontend(t *testing.T) {
	t.Run("serves_static_files", func(t *testing.T) {
		middleware := ServeEmbeddedFrontend()

		router := gin.New()
		router.Use(middleware)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/logo.png", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "image/png")
	})

	t.Run("serves_index_html_for_root", func(t *testing.T) {
		middleware := ServeEmbeddedFrontend()

		router := gin.New()
		router.Use(middleware)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
		assert.Contains(t, w.Body.String(), "<!doctype html>")
	})

	t.Run("serves_index_html_for_spa_routes", func(t *testing.T) {
		middleware := ServeEmbeddedFrontend()

		router := gin.New()
		router.Use(middleware)

		spaPaths := []string{"/dashboard", "/users/123", "/settings"}

		for _, path := range spaPaths {
			t.Run(path, func(t *testing.T) {
				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, path, nil)
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)
				assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
			})
		}
	})

	t.Run("serves_admin_index_for_admin_roots_and_deep_links", func(t *testing.T) {
		middleware := ServeEmbeddedFrontend()
		router := gin.New()
		router.Use(middleware)

		for _, requestPath := range []string{
			"/admin",
			"/admin/",
			"/admin/index.html",
			"/admin/dashboard",
			"/admin/users/123/chat-history",
		} {
			t.Run(requestPath, func(t *testing.T) {
				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, requestPath, nil)
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)
				assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
				assert.Contains(t, w.Body.String(), adminEntryMarker)
			})
		}
	})

	t.Run("serves_documentation_index_for_deep_links", func(t *testing.T) {
		middleware := ServeEmbeddedFrontend()
		router := gin.New()
		router.Use(middleware)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/tutorial-docs/orders", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
		assert.Contains(t, w.Body.String(), `href="https://luoxueapi.cc/tutorial-docs/"`)
		assert.Contains(t, w.Body.String(), "/tutorial-docs/assets/")
	})

	t.Run("skips_api_routes", func(t *testing.T) {
		middleware := ServeEmbeddedFrontend()

		apiPaths := []string{
			"/api/users",
			"/models",
			"/v1/models",
			"/v1beta/chat",
			"/backend-api/codex/responses",
			"/backend-api/codex/responses/compact",
			"/antigravity/test",
			"/setup/init",
			"/health",
			"/livez",
			"/readyz",
			"/responses",
			"/responses/compact",
		}

		for _, path := range apiPaths {
			t.Run(path, func(t *testing.T) {
				nextCalled := false
				router := gin.New()
				router.Use(middleware)
				router.GET(path, func(c *gin.Context) {
					nextCalled = true
					c.String(http.StatusOK, "ok")
				})

				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, path, nil)
				router.ServeHTTP(w, req)

				assert.True(t, nextCalled, "next handler should be called for API route")
			})
		}
	})
}

// Tests for HTMLCache
func TestHTMLCache(t *testing.T) {
	t.Run("new_cache_returns_nil", func(t *testing.T) {
		cache := NewHTMLCache()
		assert.Nil(t, cache.Get())
	})

	t.Run("set_and_get", func(t *testing.T) {
		cache := NewHTMLCache()
		cache.SetBaseHTML([]byte("<html></html>"))

		html := []byte("<html><body>test</body></html>")
		settings := []byte(`{"key":"value"}`)
		cache.Set(html, settings)

		result := cache.Get()
		require.NotNil(t, result)
		assert.Equal(t, html, result.Content)
		assert.NotEmpty(t, result.ETag)
	})

	t.Run("invalidate_clears_cache", func(t *testing.T) {
		cache := NewHTMLCache()
		cache.SetBaseHTML([]byte("<html></html>"))

		html := []byte("<html><body>test</body></html>")
		settings := []byte(`{"key":"value"}`)
		cache.Set(html, settings)

		require.NotNil(t, cache.Get())

		cache.Invalidate()

		assert.Nil(t, cache.Get())
	})

	t.Run("invalidate_rejects_write_from_an_older_generation", func(t *testing.T) {
		cache := NewHTMLCache()
		cache.SetBaseHTML([]byte("<html></html>"))
		generation := cache.Generation()

		cache.Invalidate()
		stored := cache.SetForKeyIfGeneration(
			homeHTMLCacheKey,
			[]byte("<html>stale</html>"),
			[]byte(`{"skill_marketplace_enabled":true}`),
			generation,
		)

		assert.False(t, stored)
		assert.Nil(t, cache.GetForKey(homeHTMLCacheKey))
	})

	t.Run("etag_changes_with_settings", func(t *testing.T) {
		cache := NewHTMLCache()
		cache.SetBaseHTML([]byte("<html></html>"))

		html := []byte("<html><body>test</body></html>")

		cache.Set(html, []byte(`{"v":1}`))
		etag1 := cache.Get().ETag

		cache.Invalidate()
		cache.Set(html, []byte(`{"v":2}`))
		etag2 := cache.Get().ETag

		assert.NotEqual(t, etag1, etag2)
	})

	t.Run("etag_format", func(t *testing.T) {
		cache := NewHTMLCache()
		cache.SetBaseHTML([]byte("<html></html>"))

		cache.Set([]byte("<html></html>"), []byte(`{}`))
		result := cache.Get()

		// ETag should be quoted
		assert.True(t, strings.HasPrefix(result.ETag, `"`))
		assert.True(t, strings.HasSuffix(result.ETag, `"`))
		// Should contain dash separator
		assert.Contains(t, result.ETag[1:len(result.ETag)-1], "-")
	})

	t.Run("keeps_route_specific_documents_isolated", func(t *testing.T) {
		cache := NewHTMLCache()
		cache.SetBaseHTML([]byte("<html></html>"))
		settings := []byte(`{"site_name":"落雪API"}`)

		cache.SetForKey(homeHTMLCacheKey, []byte("home"), settings)
		cache.SetForKey(modelCatalogHTMLCacheKey, []byte("models"), settings)
		cache.SetForKey(noIndexHTMLCacheKey, []byte("private"), settings)

		assert.Equal(t, []byte("home"), cache.GetForKey(homeHTMLCacheKey).Content)
		assert.Equal(t, []byte("models"), cache.GetForKey(modelCatalogHTMLCacheKey).Content)
		assert.Equal(t, []byte("private"), cache.GetForKey(noIndexHTMLCacheKey).Content)
		assert.NotEqual(t, cache.GetForKey(homeHTMLCacheKey).ETag, cache.GetForKey(modelCatalogHTMLCacheKey).ETag)
		assert.NotEqual(t, cache.GetForKey(homeHTMLCacheKey).ETag, cache.GetForKey(noIndexHTMLCacheKey).ETag)
	})

	t.Run("expires_injected_settings_after_bounded_ttl", func(t *testing.T) {
		cache := NewHTMLCache()
		cache.SetBaseHTML([]byte("<html></html>"))
		now := time.Date(2026, time.August, 5, 0, 0, 0, 0, time.UTC)
		cache.now = func() time.Time { return now }
		cache.Set([]byte("<html>enabled</html>"), []byte(`{"skill_marketplace_enabled":true}`))

		require.NotNil(t, cache.Get())
		now = now.Add(injectedHTMLCacheTTL)
		assert.Nil(t, cache.Get())
	})
}

// Benchmark tests
func BenchmarkReplaceNoncePlaceholder(b *testing.B) {
	html := []byte(`<!DOCTYPE html><html><head><script nonce="__CSP_NONCE_VALUE__">window.__APP_CONFIG__={"test":"data"};</script></head><body></body></html>`)
	nonce := "abcdefghijklmnop123456=="

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		replaceNoncePlaceholder(html, nonce)
	}
}

func BenchmarkFrontendServerServeIndexHTML(b *testing.B) {
	provider := &mockSettingsProvider{
		settings: map[string]string{"test": "value"},
	}

	server, _ := NewFrontendServer(provider)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Set(middleware.CSPNonceKey, "test-nonce")

		server.serveIndexHTML(c)
	}
}
