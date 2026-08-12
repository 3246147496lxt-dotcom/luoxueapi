package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
)

var nextFlightScriptPattern = regexp.MustCompile(`(?s)<script[^>]*>\s*self\.__next_f\.push\((.*?)\)\s*</script>`)

func parseSkillsHTML(raw []byte, minimum int) ([]rankingRecord, error) {
	var payload strings.Builder
	for _, match := range nextFlightScriptPattern.FindAllSubmatch(raw, -1) {
		var flight []json.RawMessage
		if err := json.Unmarshal(bytes.TrimSpace(match[1]), &flight); err != nil || len(flight) < 2 {
			continue
		}
		var chunk string
		if err := json.Unmarshal(flight[1], &chunk); err == nil {
			payload.WriteString(chunk)
			payload.WriteByte('\n')
		}
	}

	candidates := []string{payload.String(), html.UnescapeString(string(raw))}
	var best []rankingRecord
	var parseErrors []error
	for _, candidate := range candidates {
		searchFrom := 0
		for searchFrom < len(candidate) {
			markerAt := strings.Index(candidate[searchFrom:], `"initialSkills"`)
			if markerAt < 0 {
				break
			}
			markerAt += searchFrom
			arrayAt := strings.IndexByte(candidate[markerAt:], '[')
			if arrayAt < 0 {
				break
			}
			arrayAt += markerAt
			array, end, err := balancedJSONArray(candidate, arrayAt)
			if err != nil {
				parseErrors = append(parseErrors, err)
				searchFrom = markerAt + len(`"initialSkills"`)
				continue
			}
			var records []rankingRecord
			if err := json.Unmarshal([]byte(array), &records); err != nil {
				parseErrors = append(parseErrors, fmt.Errorf("decode initialSkills: %w", err))
				searchFrom = end
				continue
			}
			if err := validateRankingRecords(records); err != nil {
				parseErrors = append(parseErrors, err)
				searchFrom = end
				continue
			}
			if len(records) > len(best) {
				best = records
			}
			searchFrom = end
		}
	}
	if len(best) < minimum {
		if len(parseErrors) > 0 {
			return nil, fmt.Errorf("skills.sh hydration contains %d valid records; need at least %d: %w", len(best), minimum, errors.Join(parseErrors...))
		}
		return nil, fmt.Errorf("skills.sh hydration contains %d valid records; need at least %d", len(best), minimum)
	}
	return best, nil
}

func balancedJSONArray(text string, start int) (string, int, error) {
	if start < 0 || start >= len(text) || text[start] != '[' {
		return "", start, errors.New("JSON array does not start with '['")
	}
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(text); i++ {
		ch := text[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}
		switch ch {
		case '"':
			inString = true
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return text[start : i+1], i + 1, nil
			}
			if depth < 0 {
				return "", i, errors.New("unbalanced JSON array")
			}
		}
	}
	return "", len(text), errors.New("unterminated JSON array")
}

func validateRankingRecords(records []rankingRecord) error {
	for i, record := range records {
		if strings.TrimSpace(record.Source) == "" || strings.TrimSpace(record.SkillID) == "" || strings.TrimSpace(record.Name) == "" {
			return fmt.Errorf("ranking record %d is missing source, skillId, or name", i+1)
		}
		if record.Installs < 0 {
			return fmt.Errorf("ranking record %d has negative installs", i+1)
		}
	}
	return nil
}

type rankingAPIResponse struct {
	Skills  []rankingRecord `json:"skills"`
	Total   int             `json:"total"`
	Page    int             `json:"page"`
	HasMore bool            `json:"hasMore"`
}

func fetchRankingAPI(ctx context.Context, client *http.Client, baseURL string, minimum int) ([]rankingRecord, string, error) {
	var records []rankingRecord
	hasher := sha256.New()
	for page := 0; len(records) < minimum; page++ {
		url := strings.TrimRight(baseURL, "/") + "/" + fmt.Sprint(page)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, "", err
		}
		req.Header.Set("User-Agent", "sub2api-skill-market-prepare/1")
		response, err := client.Do(req)
		if err != nil {
			return nil, "", fmt.Errorf("fetch ranking page %d: %w", page, err)
		}
		body, readErr := io.ReadAll(io.LimitReader(response.Body, 16*1024*1024))
		closeErr := response.Body.Close()
		if readErr != nil || closeErr != nil {
			return nil, "", fmt.Errorf("read ranking page %d: %v %v", page, readErr, closeErr)
		}
		if response.StatusCode != http.StatusOK {
			return nil, "", fmt.Errorf("ranking page %d returned HTTP %d", page, response.StatusCode)
		}
		_, _ = hasher.Write(body)
		var decoded rankingAPIResponse
		if err := json.Unmarshal(body, &decoded); err != nil {
			return nil, "", fmt.Errorf("decode ranking page %d: %w", page, err)
		}
		if decoded.Page != page || len(decoded.Skills) == 0 {
			return nil, "", fmt.Errorf("ranking page %d returned an unexpected page or no skills", page)
		}
		records = append(records, decoded.Skills...)
		if !decoded.HasMore && len(records) < minimum {
			return nil, "", fmt.Errorf("ranking API ended at %d records; need at least %d", len(records), minimum)
		}
	}
	if err := validateRankingRecords(records); err != nil {
		return nil, "", err
	}
	return records, hex.EncodeToString(hasher.Sum(nil)), nil
}

func bytesSHA256(raw []byte) string {
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}
