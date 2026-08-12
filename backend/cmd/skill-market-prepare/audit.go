package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type auditResponse struct {
	ID     string        `json:"id"`
	Source string        `json:"source"`
	Slug   string        `json:"slug"`
	Audits []auditRecord `json:"audits"`
}

func loadOrFetchAudit(ctx context.Context, client *http.Client, outputDir, baseURL string, enabled bool, entry snapshotEntry) ([]auditRecord, error) {
	cachePath := filepath.Join(outputDir, "cache", "audits", itemCacheName(entry)+".json")
	if raw, err := os.ReadFile(cachePath); err == nil {
		var cached auditResponse
		if json.Unmarshal(raw, &cached) == nil {
			return cached.Audits, nil
		}
	}
	if !enabled {
		return nil, nil
	}
	endpoint := strings.TrimRight(baseURL, "/")
	for _, component := range append(strings.Split(entry.Source, "/"), entry.OriginalSlug) {
		endpoint += "/" + escapeURLSegment(component)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "sub2api-skill-market-prepare/1")
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	body, readErr := io.ReadAll(io.LimitReader(response.Body, 2*1024*1024))
	closeErr := response.Body.Close()
	if readErr != nil || closeErr != nil {
		return nil, fmt.Errorf("read audit response: %v %v", readErr, closeErr)
	}
	if response.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("audit endpoint returned HTTP %d", response.StatusCode)
	}
	var decoded auditResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		return nil, err
	}
	if err := writeFileAtomic(cachePath, append(body, '\n'), 0o644); err != nil {
		return nil, err
	}
	return decoded.Audits, nil
}

func escapeURLSegment(value string) string {
	escaped := url.PathEscape(value)
	// url.PathEscape intentionally leaves ':' as a valid pchar. The APIs use
	// opaque path segments, so encode it as well to avoid a skill id being
	// interpreted by an intermediary.
	escaped = strings.ReplaceAll(escaped, ":", "%3A")
	return escaped
}

func auditRiskNotes(audits []auditRecord) []string {
	notes := make([]string, 0, len(audits))
	for _, audit := range audits {
		parts := []string{"skills.sh audit", strings.TrimSpace(audit.Provider)}
		if audit.Status != "" {
			parts = append(parts, "status="+audit.Status)
		}
		if audit.RiskLevel != "" {
			parts = append(parts, "risk="+audit.RiskLevel)
		}
		if audit.Summary != "" {
			parts = append(parts, "summary="+audit.Summary)
		}
		notes = append(notes, strings.Join(parts, " | "))
	}
	return notes
}
