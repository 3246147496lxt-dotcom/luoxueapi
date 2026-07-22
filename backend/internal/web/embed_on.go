//go:build embed

package web

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	htmlpkg "html"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

const (
	// NonceHTMLPlaceholder is the placeholder for nonce in HTML script tags
	NonceHTMLPlaceholder      = "__CSP_NONCE_VALUE__"
	publicSiteOrigin          = "https://luoxueapi.cc"
	tutorialDocsBasePath      = "/tutorial-docs"
	tutorialDocsIndexFilePath = "tutorial-docs/index.html"
)

//go:embed all:dist
var frontendFS embed.FS

// PublicSettingsProvider is an interface to fetch public settings
type PublicSettingsProvider interface {
	GetPublicSettingsForInjection(ctx context.Context) (any, error)
}

// FrontendServer serves the embedded frontend with settings injection
type FrontendServer struct {
	distFS      fs.FS
	fileServer  http.Handler
	baseHTML    []byte
	cache       *HTMLCache
	settings    PublicSettingsProvider
	overrideDir string // local file override directory
}

// NewFrontendServer creates a new frontend server with settings injection
func NewFrontendServer(settingsProvider PublicSettingsProvider) (*FrontendServer, error) {
	distFS, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		return nil, err
	}

	// Read base HTML once
	file, err := distFS.Open("index.html")
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	baseHTML, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	cache := NewHTMLCache()
	cache.SetBaseHTML(baseHTML)

	return &FrontendServer{
		distFS:      distFS,
		fileServer:  http.FileServer(http.FS(distFS)),
		baseHTML:    baseHTML,
		cache:       cache,
		settings:    settingsProvider,
		overrideDir: filepath.Join("data", "public"),
	}, nil
}

// InvalidateCache invalidates the HTML cache (call when settings change)
func (s *FrontendServer) InvalidateCache() {
	if s != nil && s.cache != nil {
		s.cache.Invalidate()
	}
}

// Middleware returns the Gin middleware handler
func (s *FrontendServer) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Skip API routes
		if shouldBypassEmbeddedFrontend(path) {
			c.Next()
			return
		}
		if serveTutorialDocsRequest(c, s.distFS, s.fileServer, s.overrideDir) {
			return
		}

		cleanPath := strings.TrimPrefix(path, "/")
		if cleanPath == "" {
			cleanPath = "index.html"
		}

		// For index.html or SPA routes, serve with injected settings
		if cleanPath == "index.html" || !s.fileExists(cleanPath) {
			s.serveIndexHTML(c)
			return
		}

		// Try local override first
		if s.tryServeOverride(c, cleanPath) {
			return
		}

		// Serve static files normally (hashed assets get long-lived cache headers)
		applyStaticAssetCacheHeaders(c.Writer.Header(), cleanPath)
		s.fileServer.ServeHTTP(c.Writer, c.Request)
		c.Abort()
	}
}

func (s *FrontendServer) fileExists(path string) bool {
	return embeddedPathExists(s.distFS, path)
}

func embeddedPathExists(fsys fs.FS, path string) bool {
	file, err := fsys.Open(path)
	if err != nil {
		return false
	}
	_ = file.Close()
	return true
}

func serveTutorialDocsRequest(c *gin.Context, distFS fs.FS, fileServer http.Handler, overrideDir string) bool {
	requestPath := c.Request.URL.Path
	if requestPath != tutorialDocsBasePath && !strings.HasPrefix(requestPath, tutorialDocsBasePath+"/") {
		return false
	}

	if requestPath == tutorialDocsBasePath {
		location := tutorialDocsBasePath + "/"
		if query := c.Request.URL.RawQuery; query != "" {
			location += "?" + query
		}
		c.Redirect(http.StatusMovedPermanently, location)
		c.Abort()
		return true
	}

	cleanPath := strings.TrimPrefix(requestPath, "/")
	if cleanPath == "tutorial-docs/" {
		cleanPath = tutorialDocsIndexFilePath
	}
	if !embeddedPathExists(distFS, cleanPath) {
		// Extensionless paths are documentation SPA routes. Missing assets must
		// remain 404s instead of returning HTML with the wrong MIME type.
		if filepath.Ext(strings.TrimSuffix(cleanPath, "/")) != "" ||
			!embeddedPathExists(distFS, tutorialDocsIndexFilePath) {
			c.Status(http.StatusNotFound)
			c.Abort()
			return true
		}
		cleanPath = tutorialDocsIndexFilePath
	}

	if tryServeOverrideFile(c, overrideDir, cleanPath) {
		return true
	}
	if cleanPath == tutorialDocsIndexFilePath {
		c.Header("Cache-Control", "no-cache")
		serveEmbeddedHTMLFile(c, distFS, tutorialDocsIndexFilePath)
		return true
	}

	applyStaticAssetCacheHeaders(c.Writer.Header(), cleanPath)

	// http.FileServer resolves from Request.URL.Path. Rewrite only for this
	// request so docs deep links use the docs entry point, not the main SPA.
	originalPath := c.Request.URL.Path
	c.Request.URL.Path = "/" + cleanPath
	fileServer.ServeHTTP(c.Writer, c.Request)
	c.Request.URL.Path = originalPath
	c.Abort()
	return true
}

// tryServeOverride checks if a local override file exists and serves it.
// Files in overrideDir take precedence over embedded files.
func (s *FrontendServer) tryServeOverride(c *gin.Context, cleanPath string) bool {
	if s.overrideDir == "" {
		return false
	}
	filePath := filepath.Join(s.overrideDir, filepath.Clean("/"+cleanPath))
	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() {
		return false
	}
	c.File(filePath)
	c.Abort()
	return true
}

func (s *FrontendServer) serveIndexHTML(c *gin.Context) {
	// Get nonce from context (generated by SecurityHeaders middleware)
	nonce := middleware.GetNonceFromContext(c)
	cacheKey := htmlRouteCacheKey(c.Request.URL.Path)
	applyRouteIndexingHeaders(c.Writer.Header(), cacheKey)

	// Check cache first
	cached := s.cache.GetForKey(cacheKey)
	if cached != nil {
		// Check If-None-Match for 304 response
		if match := c.GetHeader("If-None-Match"); match == cached.ETag {
			c.Status(http.StatusNotModified)
			c.Abort()
			return
		}

		// Replace nonce placeholder with actual nonce before serving
		content := replaceNoncePlaceholder(cached.Content, nonce)

		c.Header("ETag", cached.ETag)
		c.Header("Cache-Control", "no-cache") // Must revalidate
		c.Data(http.StatusOK, "text/html; charset=utf-8", content)
		c.Abort()
		return
	}

	// Cache miss - fetch settings and render
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	settings, err := s.settings.GetPublicSettingsForInjection(ctx)
	if err != nil {
		// Keep route-level SEO directives even when public settings are
		// temporarily unavailable. In particular, private SPA routes must not
		// become indexable just because the settings lookup failed.
		fallback := injectRouteMetadata(s.baseHTML, nil, c.Request.URL.Path)
		c.Data(http.StatusOK, "text/html; charset=utf-8", fallback)
		c.Abort()
		return
	}

	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		fallback := injectRouteMetadata(s.baseHTML, nil, c.Request.URL.Path)
		c.Data(http.StatusOK, "text/html; charset=utf-8", fallback)
		c.Abort()
		return
	}

	rendered := s.injectSettingsForPath(settingsJSON, c.Request.URL.Path)
	s.cache.SetForKey(cacheKey, rendered, settingsJSON)

	// Replace nonce placeholder with actual nonce before serving
	content := replaceNoncePlaceholder(rendered, nonce)

	cached = s.cache.GetForKey(cacheKey)
	if cached != nil {
		c.Header("ETag", cached.ETag)
	}
	c.Header("Cache-Control", "no-cache")
	c.Data(http.StatusOK, "text/html; charset=utf-8", content)
	c.Abort()
}

func (s *FrontendServer) injectSettings(settingsJSON []byte) []byte {
	return s.injectSettingsForPath(settingsJSON, "")
}

func (s *FrontendServer) injectSettingsForPath(settingsJSON []byte, requestPath string) []byte {
	// Create the script tag to inject with nonce placeholder
	// The placeholder will be replaced with actual nonce at request time
	script := []byte(`<script nonce="` + NonceHTMLPlaceholder + `">window.__APP_CONFIG__=` + string(settingsJSON) + `;</script>`)

	// Inject before </head>
	headClose := []byte("</head>")
	result := bytes.Replace(s.baseHTML, headClose, append(script, headClose...), 1)

	// Replace <title> with custom site name so the browser tab shows it immediately
	result = injectSiteTitle(result, settingsJSON)
	return injectRouteMetadata(result, settingsJSON, requestPath)
}

const (
	homeHTMLCacheKey         = "home"
	modelCatalogHTMLCacheKey = "models.html"
	noIndexHTMLCacheKey      = "noindex"
)

func htmlRouteCacheKey(requestPath string) string {
	path := strings.TrimSpace(requestPath)
	if len(path) > 1 {
		path = strings.TrimRight(path, "/")
	}

	switch path {
	case "":
		// Preserve the neutral cache entry used by injectSettings and the
		// legacy HTMLCache API. Real HTTP requests always provide a path.
		return ""
	case "/", "/home", "/index.html":
		return homeHTMLCacheKey
	case "/models.html":
		return modelCatalogHTMLCacheKey
	default:
		// All other SPA routes are application, authentication, callback, or
		// unknown routes. They should remain usable in a browser but must not
		// appear in search results.
		return noIndexHTMLCacheKey
	}
}

func applyRouteIndexingHeaders(header http.Header, cacheKey string) {
	if header == nil || cacheKey != noIndexHTMLCacheKey {
		return
	}
	header.Set("X-Robots-Tag", "noindex, nofollow")
}

func injectRouteMetadata(html, settingsJSON []byte, requestPath string) []byte {
	switch htmlRouteCacheKey(requestPath) {
	case homeHTMLCacheKey:
		backendMode, _ := publicSEOFlags(settingsJSON)
		if backendMode {
			return injectNoIndexMetadata(html)
		}
		return injectHomeMetadata(html, settingsJSON)
	case modelCatalogHTMLCacheKey:
		backendMode, publicModelCatalogEnabled := publicSEOFlags(settingsJSON)
		if backendMode || !publicModelCatalogEnabled {
			return injectNoIndexMetadata(html)
		}
		return injectModelCatalogMetadata(html, settingsJSON)
	case noIndexHTMLCacheKey:
		return injectNoIndexMetadata(html)
	default:
		return html
	}
}

func publicSEOFlags(settingsJSON []byte) (backendMode, publicModelCatalogEnabled bool) {
	var cfg struct {
		BackendModeEnabled        bool `json:"backend_mode_enabled"`
		PublicModelCatalogEnabled bool `json:"public_model_catalog_enabled"`
	}
	if err := json.Unmarshal(settingsJSON, &cfg); err != nil {
		return false, false
	}
	return cfg.BackendModeEnabled, cfg.PublicModelCatalogEnabled
}

func publicSiteName(settingsJSON []byte) string {
	var cfg struct {
		SiteName string `json:"site_name"`
	}
	_ = json.Unmarshal(settingsJSON, &cfg)
	siteName := strings.TrimSpace(cfg.SiteName)
	if siteName == "" {
		siteName = "落雪API"
	}
	return siteName
}

// injectHomeMetadata gives crawlers a useful initial document before Vue
// mounts. Canonical and Open Graph URLs use the fixed public origin instead of
// trusting a request Host header.
func injectHomeMetadata(html, settingsJSON []byte) []byte {
	siteName := publicSiteName(settingsJSON)
	title := siteName + " · GPT API 接入与密钥管理"
	description := siteName + " 是面向开发者的 AI API 网关，提供兼容 OpenAI 的 GPT API 接入、API 密钥管理、调用记录、Token 与费用查询。"
	return injectIndexablePageMetadata(html, siteName, title, description, publicSiteOrigin+"/home")
}

// injectModelCatalogMetadata gives the anonymous catalog a useful initial
// document head before Vue mounts.
func injectModelCatalogMetadata(html, settingsJSON []byte) []byte {
	siteName := publicSiteName(settingsJSON)

	title := "模型广场 · " + siteName
	description := "查看 " + siteName + " 已公开的模型、能力、上下文和标准价格。"
	return injectIndexablePageMetadata(html, siteName, title, description, publicSiteOrigin+"/models.html")
}

func injectIndexablePageMetadata(html []byte, siteName, title, description, canonicalURL string) []byte {
	result := replaceHTMLTitle(html, title)

	metadata := []byte(
		"\n    " + `<meta name="description" content="` + htmlpkg.EscapeString(description) + `" />` +
			"\n    " + `<meta name="robots" content="index, follow, max-image-preview:large, max-snippet:-1, max-video-preview:-1" />` +
			"\n    " + `<link rel="canonical" href="` + htmlpkg.EscapeString(canonicalURL) + `" />` +
			"\n    " + `<meta property="og:type" content="website" />` +
			"\n    " + `<meta property="og:site_name" content="` + htmlpkg.EscapeString(siteName) + `" />` +
			"\n    " + `<meta property="og:title" content="` + htmlpkg.EscapeString(title) + `" />` +
			"\n    " + `<meta property="og:description" content="` + htmlpkg.EscapeString(description) + `" />` +
			"\n    " + `<meta property="og:url" content="` + htmlpkg.EscapeString(canonicalURL) + `" />` +
			"\n    " + `<meta name="twitter:card" content="summary" />` +
			"\n    " + `<meta name="twitter:title" content="` + htmlpkg.EscapeString(title) + `" />` +
			"\n    " + `<meta name="twitter:description" content="` + htmlpkg.EscapeString(description) + `" />` +
			"\n  ",
	)
	result = bytes.Replace(result, []byte("</head>"), append(metadata, []byte("</head>")...), 1)

	// Non-JavaScript crawlers and users still receive a concise, truthful
	// description with links to the public surfaces. Vue replaces this fallback
	// when the application mounts.
	fallback := []byte(
		`<noscript><main aria-label="` + htmlpkg.EscapeString(title) + `">` +
			`<h1>` + htmlpkg.EscapeString(title) + `</h1>` +
			`<p>` + htmlpkg.EscapeString(description) + `</p>` +
			`<nav aria-label="公开页面">` +
			`<a href="/home">首页</a> ` +
			`<a href="/tutorial-docs/">使用教程</a>` +
			`</nav></main></noscript>`,
	)
	return bytes.Replace(result, []byte(`<div id="app"></div>`), []byte(`<div id="app">`+string(fallback)+`</div>`), 1)
}

func injectNoIndexMetadata(html []byte) []byte {
	metadata := []byte("\n    " + `<meta name="robots" content="noindex, nofollow" />` + "\n  ")
	return bytes.Replace(html, []byte("</head>"), append(metadata, []byte("</head>")...), 1)
}

func replaceHTMLTitle(html []byte, title string) []byte {
	titleStart := bytes.Index(html, []byte("<title>"))
	titleEnd := bytes.Index(html, []byte("</title>"))
	if titleStart == -1 || titleEnd == -1 || titleEnd <= titleStart {
		return html
	}

	newTitle := []byte("<title>" + htmlpkg.EscapeString(title) + "</title>")
	var buf bytes.Buffer
	buf.Write(html[:titleStart])
	buf.Write(newTitle)
	buf.Write(html[titleEnd+len("</title>"):])
	return buf.Bytes()
}

// injectSiteTitle replaces the static <title> in HTML with the configured site name.
// This ensures the browser tab shows the correct title before JS executes.
func injectSiteTitle(html, settingsJSON []byte) []byte {
	var cfg struct {
		SiteName string `json:"site_name"`
	}
	if err := json.Unmarshal(settingsJSON, &cfg); err != nil || cfg.SiteName == "" {
		return html
	}

	// Find and replace the existing <title>...</title>
	titleStart := bytes.Index(html, []byte("<title>"))
	titleEnd := bytes.Index(html, []byte("</title>"))
	if titleStart == -1 || titleEnd == -1 || titleEnd <= titleStart {
		return html
	}

	newTitle := []byte("<title>" + htmlpkg.EscapeString(cfg.SiteName) + " - AI API Gateway</title>")
	var buf bytes.Buffer
	buf.Write(html[:titleStart])
	buf.Write(newTitle)
	buf.Write(html[titleEnd+len("</title>"):])
	return buf.Bytes()
}

// replaceNoncePlaceholder replaces the nonce placeholder with actual nonce value
func replaceNoncePlaceholder(html []byte, nonce string) []byte {
	return bytes.ReplaceAll(html, []byte(NonceHTMLPlaceholder), []byte(nonce))
}

// ServeEmbeddedFrontend returns a middleware for serving embedded frontend
// This is the legacy function for backward compatibility when no settings provider is available
func ServeEmbeddedFrontend() gin.HandlerFunc {
	distFS, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		panic("failed to get dist subdirectory: " + err.Error())
	}
	fileServer := http.FileServer(http.FS(distFS))
	overrideDir := filepath.Join("data", "public")

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		if shouldBypassEmbeddedFrontend(path) {
			c.Next()
			return
		}
		if serveTutorialDocsRequest(c, distFS, fileServer, overrideDir) {
			return
		}

		cleanPath := strings.TrimPrefix(path, "/")
		if cleanPath == "" {
			cleanPath = "index.html"
		}

		if file, err := distFS.Open(cleanPath); err == nil {
			_ = file.Close()
			// Try local override first
			if tryServeOverrideFile(c, overrideDir, cleanPath) {
				return
			}
			applyStaticAssetCacheHeaders(c.Writer.Header(), cleanPath)
			fileServer.ServeHTTP(c.Writer, c.Request)
			c.Abort()
			return
		}

		serveIndexHTML(c, distFS)
	}
}

// tryServeOverrideFile is a standalone version of tryServeOverride for legacy usage.
func tryServeOverrideFile(c *gin.Context, overrideDir, cleanPath string) bool {
	if overrideDir == "" {
		return false
	}
	filePath := filepath.Join(overrideDir, filepath.Clean("/"+cleanPath))
	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() {
		return false
	}
	c.File(filePath)
	c.Abort()
	return true
}

func shouldBypassEmbeddedFrontend(path string) bool {
	trimmed := strings.TrimSpace(path)
	return strings.HasPrefix(trimmed, "/api/") ||
		strings.HasPrefix(trimmed, "/v1/") ||
		strings.HasPrefix(trimmed, "/v1beta/") ||
		strings.HasPrefix(trimmed, "/backend-api/") ||
		strings.HasPrefix(trimmed, "/antigravity/") ||
		strings.HasPrefix(trimmed, "/setup/") ||
		trimmed == "/health" ||
		trimmed == "/models" ||
		trimmed == "/responses" ||
		strings.HasPrefix(trimmed, "/responses/") ||
		trimmed == "/alpha/search" ||
		strings.HasPrefix(trimmed, "/images/") ||
		strings.HasPrefix(trimmed, "/videos/")
}

func serveIndexHTML(c *gin.Context, fsys fs.FS) {
	serveEmbeddedHTMLFile(c, fsys, "index.html")
}

func serveEmbeddedHTMLFile(c *gin.Context, fsys fs.FS, path string) {
	file, err := fsys.Open(path)
	if err != nil {
		c.String(http.StatusNotFound, "Frontend not found")
		c.Abort()
		return
	}
	defer func() { _ = file.Close() }()

	content, err := io.ReadAll(file)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to read index.html")
		c.Abort()
		return
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", content)
	c.Abort()
}

func HasEmbeddedFrontend() bool {
	_, err := frontendFS.ReadFile("dist/index.html")
	return err == nil
}
