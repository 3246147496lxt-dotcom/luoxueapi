//go:build embed

package web

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
)

// HTMLCache manages the cached index.html with injected settings
type HTMLCache struct {
	mu              sync.RWMutex
	entries         map[string]CachedHTML
	baseHTMLHash    string // Hash of the original index.html (immutable after build)
	settingsVersion uint64 // Incremented when settings change
}

// CachedHTML represents the cache state
type CachedHTML struct {
	Content []byte
	ETag    string
}

// NewHTMLCache creates a new HTML cache instance
func NewHTMLCache() *HTMLCache {
	return &HTMLCache{entries: make(map[string]CachedHTML)}
}

// SetBaseHTML initializes the cache with the base HTML template
func (c *HTMLCache) SetBaseHTML(baseHTML []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	hash := sha256.Sum256(baseHTML)
	c.baseHTMLHash = hex.EncodeToString(hash[:8]) // First 8 bytes for brevity
}

// Invalidate marks the cache as stale
func (c *HTMLCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.settingsVersion++
	c.entries = make(map[string]CachedHTML)
}

// Get returns the cached HTML or nil if cache is stale
func (c *HTMLCache) Get() *CachedHTML {
	return c.GetForKey("")
}

// GetForKey returns a route-specific cached HTML document. Keeping the
// default Get/Set pair preserves the legacy single-document API while public
// marketing routes can safely inject their own metadata and ETag.
func (c *HTMLCache) GetForKey(key string) *CachedHTML {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok {
		return nil
	}
	return &CachedHTML{
		Content: entry.Content,
		ETag:    entry.ETag,
	}
}

// Set updates the cache with new rendered HTML
func (c *HTMLCache) Set(html []byte, settingsJSON []byte) {
	c.SetForKey("", html, settingsJSON)
}

// SetForKey stores a route-specific rendered HTML document.
func (c *HTMLCache) SetForKey(key string, html []byte, settingsJSON []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.entries == nil {
		c.entries = make(map[string]CachedHTML)
	}
	c.entries[key] = CachedHTML{
		Content: html,
		ETag:    c.generateETagForKey(key, settingsJSON),
	}
}

// generateETag creates an ETag from base HTML hash + settings hash
func (c *HTMLCache) generateETag(settingsJSON []byte) string {
	return c.generateETagForKey("", settingsJSON)
}

func (c *HTMLCache) generateETagForKey(key string, settingsJSON []byte) string {
	settingsHash := sha256.Sum256(append(append([]byte(key), 0), settingsJSON...))
	return `"` + c.baseHTMLHash + "-" + hex.EncodeToString(settingsHash[:8]) + `"`
}
