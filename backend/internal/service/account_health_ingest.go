package service

import (
	"context"
	"encoding/json"
	"strings"
	"unicode"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type AccountHealthAddItem struct {
	Index      int     `json:"index"`
	Name       string  `json:"name"`
	Platform   string  `json:"platform"`
	Type       string  `json:"type"`
	BaseURL    string  `json:"base_url,omitempty"`
	KeyHint    string  `json:"key_hint"`
	Summary    string  `json:"summary"`
	AuthMethod string  `json:"auth_method,omitempty"`
	ProxyID    *int64  `json:"proxy_id,omitempty"`
	GroupIDs   []int64 `json:"group_ids,omitempty"`
}

type AccountHealthAddProposal struct {
	Items []AccountHealthAddItem `json:"items"`
}

type AccountHealthAddRequest struct {
	Handoff     string  `json:"handoff"`
	Indexes     []int   `json:"indexes"`
	Confirm     string  `json:"confirm"`
	Platform    string  `json:"platform"`
	ProxyID     *int64  `json:"proxy_id,omitempty"`
	GroupIDs    []int64 `json:"group_ids,omitempty"`
	OperationID string  `json:"operation_id,omitempty"`
}

type AccountHealthAddFailure struct {
	Index   int    `json:"index"`
	Message string `json:"message"`
}

type AccountHealthAddResult struct {
	Created     []AccountHealthAddItem    `json:"created"`
	Failed      []AccountHealthAddFailure `json:"failed"`
	OperationID string                    `json:"operation_id,omitempty"`
}

type accountHealthIngested struct {
	Item   AccountHealthAddItem
	APIKey string
}

// AccountHealthGrokSSOImporter converts Grok Web SSO tokens into OAuth credentials.
type AccountHealthGrokSSOImporter interface {
	ConvertFromSSO(ctx context.Context, ssoToken string, proxyID *int64) (*GrokTokenInfo, error)
	BuildAccountCredentials(tokenInfo *GrokTokenInfo) map[string]any
}

var accountHealthAddPlatforms = map[string]string{
	"openai":    PlatformOpenAI,
	"grok":      PlatformGrok,
	"kimi":      PlatformKimi,
	"zhipu":     PlatformZhipu,
	"deepseek":  PlatformDeepseek,
	"anthropic": PlatformAnthropic,
	"claude":    PlatformAnthropic,
	"gemini":    PlatformGemini,
}

func looksLikeAccountHealthHandoff(raw string) bool {
	n := strings.ToLower(strings.TrimSpace(raw))
	if n == "" {
		return false
	}
	if looksLikeAccountHealthSSOToken(n) {
		return true
	}
	needles := []string{
		"sk-",
		"rk-",
		"bearer ",
		`"api_key"`,
		"api_key:",
		"access_token",
		"refresh_token",
		"id_token",
		"client_secret",
		"session_token",
		"authorization:",
	}
	for _, needle := range needles {
		if strings.Contains(n, needle) {
			return true
		}
	}
	return false
}

// looksLikeAccountHealthSSOToken recognizes JWT-shaped SSO exports without
// decoding or logging the token. It is used only to keep credentials out of
// the LLM context; SSO import itself is handled by the Grok importer.
func looksLikeAccountHealthSSOToken(raw string) bool {
	for _, line := range strings.Split(raw, "\n") {
		parts := strings.Split(strings.TrimSpace(line), ".")
		if len(parts) == 3 && strings.HasPrefix(strings.ToLower(parts[0]), "eyj") {
			return true
		}
	}
	return false
}

func ingestAccountHealthHandoff(raw, defaultPlatform string) *AccountHealthAddProposal {
	items := parseAccountHealthHandoff(raw, defaultPlatform)
	out := &AccountHealthAddProposal{Items: make([]AccountHealthAddItem, 0, len(items))}
	for _, item := range items {
		out.Items = append(out.Items, item.Item)
	}
	return out
}

func parseAccountHealthHandoff(raw, defaultPlatform string) []accountHealthIngested {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var decoded any
	if json.Unmarshal([]byte(raw), &decoded) == nil {
		return finalizeAccountHealthIngested(collectAccountHealthJSON(decoded, defaultPlatform), defaultPlatform)
	}
	return finalizeAccountHealthIngested(collectAccountHealthLines(raw, defaultPlatform), defaultPlatform)
}

func collectAccountHealthJSON(value any, defaultPlatform string) []accountHealthIngested {
	switch typed := value.(type) {
	case []any:
		out := make([]accountHealthIngested, 0, len(typed))
		for _, item := range typed {
			out = append(out, collectAccountHealthJSON(item, defaultPlatform)...)
		}
		return out
	case map[string]any:
		if item, ok := accountHealthItemFromJSONObject(typed, defaultPlatform); ok {
			return []accountHealthIngested{item}
		}
		for _, key := range []string{"accounts", "items", "keys", "data"} {
			if nested, exists := typed[key]; exists {
				return collectAccountHealthJSON(nested, defaultPlatform)
			}
		}
		return nil
	case string:
		return collectAccountHealthLines(typed, defaultPlatform)
	default:
		return nil
	}
}

func accountHealthItemFromJSONObject(obj map[string]any, defaultPlatform string) (accountHealthIngested, bool) {
	apiKey := firstJSONString(obj, "api_key", "access_token", "key", "token")
	if apiKey == "" {
		if creds, ok := obj["credentials"].(map[string]any); ok {
			apiKey = firstJSONString(creds, "api_key", "access_token", "key", "token")
		}
	}
	rawCredential := strings.TrimSpace(apiKey)
	platform := normalizeAccountHealthAddPlatform(firstJSONString(obj, "platform"), defaultPlatform)
	if looksLikeAccountHealthSSOToken(rawCredential) {
		name := sanitizeAccountHealthAddName(firstJSONString(obj, "name", "account_name"))
		item := accountHealthSSOItem(rawCredential, name, defaultPlatform)
		return item, true
	}
	apiKey = normalizeAccountHealthAPIKey(rawCredential)
	if apiKey == "" {
		return accountHealthIngested{}, false
	}
	baseURL := firstJSONString(obj, "base_url", "baseUrl", "url")
	if baseURL == "" {
		if creds, ok := obj["credentials"].(map[string]any); ok {
			baseURL = firstJSONString(creds, "base_url", "baseUrl", "url")
		}
	}
	name := sanitizeAccountHealthAddName(firstJSONString(obj, "name", "account_name"))
	return accountHealthIngested{
		APIKey: apiKey,
		Item: AccountHealthAddItem{
			Name:     name,
			Platform: platform,
			Type:     AccountTypeAPIKey,
			BaseURL:  strings.TrimSpace(baseURL),
			KeyHint:  accountHealthKeyHint(apiKey),
			Summary:  accountHealthAddSummary(platform),
		},
	}, true
}

func collectAccountHealthLines(raw, defaultPlatform string) []accountHealthIngested {
	lines := strings.Split(raw, "\n")
	out := make([]accountHealthIngested, 0, len(lines))
	for _, line := range lines {
		item, ok := accountHealthItemFromLine(line, defaultPlatform)
		if !ok {
			continue
		}
		out = append(out, item)
	}
	return out
}

func accountHealthItemFromLine(line, defaultPlatform string) (accountHealthIngested, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return accountHealthIngested{}, false
	}
	// Registration exports may include email----password----SSO token records.
	// Keep only the token for conversion and never expose the password.
	if strings.Contains(line, "----") {
		for i, part := range strings.Split(line, "----") {
			part = strings.TrimSpace(part)
			if looksLikeAccountHealthSSOToken(part) {
				name := ""
				if i > 0 {
					name = strings.TrimSpace(strings.Split(line, "----")[0])
				}
				return accountHealthSSOItem(part, name, defaultPlatform), true
			}
		}
	}
	if strings.Contains(line, "|") {
		parts := splitAccountHealthPipes(line)
		for i, part := range parts {
			if looksLikeAccountHealthSSOToken(part) {
				name := ""
				if i > 0 && !looksLikeAccountHealthSSOToken(parts[0]) {
					name = parts[0]
				}
				return accountHealthSSOItem(part, name, defaultPlatform), true
			}
		}
		if len(parts) >= 3 {
			name := ""
			platform := ""
			apiKey := ""
			baseURL := ""
			switch {
			case looksLikeAccountHealthAPIKey(parts[0]) && len(parts) >= 2:
				apiKey = parts[0]
				baseURL = parts[1]
			case looksLikeAccountHealthAPIKey(parts[1]):
				name = parts[0]
				apiKey = parts[1]
				if len(parts) >= 3 {
					baseURL = parts[2]
				}
			default:
				name = parts[0]
				platform = parts[1]
				apiKey = parts[2]
				if len(parts) >= 4 {
					baseURL = parts[3]
				}
			}
			apiKey = normalizeAccountHealthAPIKey(apiKey)
			if apiKey == "" {
				return accountHealthIngested{}, false
			}
			resolved := normalizeAccountHealthAddPlatform(platform, defaultPlatform)
			return accountHealthIngested{
				APIKey: apiKey,
				Item: AccountHealthAddItem{
					Name:     sanitizeAccountHealthAddName(name),
					Platform: resolved,
					Type:     AccountTypeAPIKey,
					BaseURL:  strings.TrimSpace(baseURL),
					KeyHint:  accountHealthKeyHint(apiKey),
					Summary:  accountHealthAddSummary(resolved),
				},
			}, true
		}
	}

	fields := strings.Fields(line)
	if len(fields) == 0 {
		return accountHealthIngested{}, false
	}
	if len(fields) == 1 && looksLikeAccountHealthSSOToken(fields[0]) {
		return accountHealthSSOItem(fields[0], "", defaultPlatform), true
	}
	for i, field := range fields {
		if looksLikeAccountHealthSSOToken(field) {
			name := ""
			if i > 0 {
				name = fields[0]
			}
			return accountHealthSSOItem(field, name, defaultPlatform), true
		}
	}
	var (
		name     string
		platform string
		apiKey   string
		baseURL  string
	)
	for _, field := range fields {
		switch {
		case strings.HasPrefix(strings.ToLower(field), "http://") || strings.HasPrefix(strings.ToLower(field), "https://"):
			if baseURL == "" {
				baseURL = field
			}
		case looksLikeAccountHealthAPIKey(field) || strings.HasPrefix(strings.ToLower(field), "bearer"):
			if apiKey == "" {
				apiKey = field
			}
		case accountHealthAddPlatforms[strings.ToLower(field)] != "":
			if platform == "" {
				platform = field
			}
		default:
			if name == "" && apiKey == "" {
				name = field
			}
		}
	}
	if apiKey == "" && looksLikeAccountHealthAPIKey(line) {
		apiKey = line
	}
	apiKey = normalizeAccountHealthAPIKey(apiKey)
	if apiKey == "" {
		return accountHealthIngested{}, false
	}
	resolved := normalizeAccountHealthAddPlatform(platform, defaultPlatform)
	return accountHealthIngested{
		APIKey: apiKey,
		Item: AccountHealthAddItem{
			Name:     sanitizeAccountHealthAddName(name),
			Platform: resolved,
			Type:     AccountTypeAPIKey,
			BaseURL:  strings.TrimSpace(baseURL),
			KeyHint:  accountHealthKeyHint(apiKey),
			Summary:  accountHealthAddSummary(resolved),
		},
	}, true
}

func finalizeAccountHealthIngested(items []accountHealthIngested, defaultPlatform string) []accountHealthIngested {
	_ = defaultPlatform
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	out := make([]accountHealthIngested, 0, len(items))
	generated := map[string]int{}
	for _, item := range items {
		rawCredential := strings.TrimSpace(item.APIKey)
		isSSO := looksLikeAccountHealthSSOToken(rawCredential) || item.Item.AuthMethod == "grok_sso"
		apiKey := rawCredential
		if !isSSO {
			apiKey = normalizeAccountHealthAPIKey(rawCredential)
		}
		if apiKey == "" {
			continue
		}
		platform := normalizeAccountHealthAddPlatform(item.Item.Platform, "")
		dupKey := platform + "\x00" + apiKey
		if _, ok := seen[dupKey]; ok {
			continue
		}
		seen[dupKey] = struct{}{}
		name := sanitizeAccountHealthAddName(item.Item.Name)
		if name == "" {
			generated[platform]++
			name = "imported-" + platform + "-" + itoaAccountHealth(generated[platform])
		}
		item.APIKey = apiKey
		item.Item.Name = name
		item.Item.Platform = platform
		if isSSO {
			item.Item.Type = AccountTypeOAuth
			item.Item.Platform = PlatformGrok
			item.Item.AuthMethod = "grok_sso"
			item.Item.KeyHint = "sso••••" + accountHealthTokenSuffix(apiKey)
			item.Item.Summary = "Grok SSO · grok"
		} else {
			item.Item.Type = AccountTypeAPIKey
			item.Item.KeyHint = accountHealthKeyHint(apiKey)
			item.Item.Summary = accountHealthAddSummary(platform)
		}
		item.Item.Index = len(out)
		out = append(out, item)
		if len(out) >= MaxAccountHealthAddItems {
			break
		}
	}
	return out
}

func (s *AccountHealthService) Add(ctx context.Context, req AccountHealthAddRequest) (*AccountHealthAddResult, error) {
	if s == nil || s.admin == nil {
		return nil, infraerrors.InternalServer("ACCOUNT_HEALTH_UNAVAILABLE", "account health service unavailable")
	}
	if strings.TrimSpace(req.Confirm) != AccountHealthConfirmAdd {
		return nil, infraerrors.BadRequest("ACCOUNT_HEALTH_CONFIRM_REQUIRED", "confirm must be ADD")
	}
	if cached, ok := s.cachedOperation(req.OperationID).(*AccountHealthAddResult); ok && cached != nil {
		return cached, nil
	}
	handoff := strings.TrimSpace(req.Handoff)
	if handoff == "" {
		return nil, infraerrors.BadRequest("ACCOUNT_HEALTH_HANDOFF_REQUIRED", "handoff is required")
	}
	indexes := uniqueNonNegativeInts(req.Indexes, MaxAccountHealthAddItems+1)
	if len(indexes) == 0 {
		return nil, infraerrors.BadRequest("ACCOUNT_HEALTH_INDEXES_REQUIRED", "indexes is required")
	}
	if len(indexes) > MaxAccountHealthAddItems {
		return nil, infraerrors.BadRequest("ACCOUNT_HEALTH_INDEXES_LIMIT", "indexes exceeds the maximum of 50")
	}

	parsed := parseAccountHealthHandoff(handoff, req.Platform)
	byIndex := make(map[int]accountHealthIngested, len(parsed))
	for _, item := range parsed {
		byIndex[item.Item.Index] = item
	}

	result := &AccountHealthAddResult{
		Created: make([]AccountHealthAddItem, 0, len(indexes)),
		Failed:  make([]AccountHealthAddFailure, 0),
	}
	result.OperationID = strings.TrimSpace(req.OperationID)
	for _, index := range indexes {
		item, ok := byIndex[index]
		if !ok {
			result.Failed = append(result.Failed, AccountHealthAddFailure{Index: index, Message: "create failed"})
			continue
		}
		accountType := AccountTypeAPIKey
		credentials := map[string]any{"api_key": item.APIKey}
		if item.Item.AuthMethod == "grok_sso" || looksLikeAccountHealthSSOToken(item.APIKey) {
			if s.grokSSO == nil {
				result.Failed = append(result.Failed, AccountHealthAddFailure{Index: index, Message: "Grok SSO importer unavailable"})
				continue
			}
			tokenInfo, err := s.grokSSO.ConvertFromSSO(ctx, item.APIKey, req.ProxyID)
			if err != nil || tokenInfo == nil {
				result.Failed = append(result.Failed, AccountHealthAddFailure{Index: index, Message: "Grok SSO conversion failed"})
				continue
			}
			credentials = s.grokSSO.BuildAccountCredentials(tokenInfo)
			accountType = AccountTypeOAuth
			if strings.TrimSpace(item.Item.Name) == "" && strings.TrimSpace(tokenInfo.Email) != "" {
				item.Item.Name = tokenInfo.Email
			}
		}
		if strings.TrimSpace(item.Item.BaseURL) != "" {
			credentials["base_url"] = strings.TrimSpace(item.Item.BaseURL)
		}
		created, err := s.admin.CreateAccount(ctx, &CreateAccountInput{
			Name:        item.Item.Name,
			Platform:    item.Item.Platform,
			Type:        accountType,
			Credentials: credentials,
			ProxyID:     req.ProxyID,
			GroupIDs:    append([]int64(nil), req.GroupIDs...),
		})
		if err != nil || created == nil {
			result.Failed = append(result.Failed, AccountHealthAddFailure{Index: index, Message: "create failed"})
			continue
		}
		public := item.Item
		if name := strings.TrimSpace(created.Name); name != "" {
			public.Name = name
		}
		result.Created = append(result.Created, public)
	}
	s.rememberOperation(req.OperationID, result)
	return result, nil
}

func accountHealthSSOItem(token, name, defaultPlatform string) accountHealthIngested {
	return accountHealthIngested{APIKey: strings.TrimSpace(token), Item: AccountHealthAddItem{
		Name: sanitizeAccountHealthAddName(name), Platform: PlatformGrok, Type: AccountTypeOAuth,
		AuthMethod: "grok_sso", KeyHint: "sso••••" + accountHealthTokenSuffix(token),
		Summary: "Grok SSO · grok",
	}}
}

func accountHealthTokenSuffix(token string) string {
	r := []rune(strings.TrimSpace(token))
	if len(r) > 4 {
		r = r[len(r)-4:]
	}
	return string(r)
}

func normalizeAccountHealthAddPlatform(raw, fallback string) string {
	n := strings.ToLower(strings.TrimSpace(raw))
	if mapped, ok := accountHealthAddPlatforms[n]; ok {
		return mapped
	}
	if strings.TrimSpace(fallback) != "" && !strings.EqualFold(strings.TrimSpace(fallback), raw) {
		return normalizeAccountHealthAddPlatform(fallback, "")
	}
	if mapped, ok := accountHealthAddPlatforms[strings.ToLower(strings.TrimSpace(fallback))]; ok {
		return mapped
	}
	return PlatformOpenAI
}

func looksLikeAccountHealthAPIKey(raw string) bool {
	n := strings.TrimSpace(raw)
	if n == "" {
		return false
	}
	lower := strings.ToLower(n)
	if strings.HasPrefix(lower, "bearer ") {
		return utf8.RuneCountInString(strings.TrimSpace(n[7:])) >= 8
	}
	if strings.HasPrefix(lower, "sk-") || strings.HasPrefix(lower, "rk-") {
		return utf8.RuneCountInString(n) >= 8
	}
	return false
}

func normalizeAccountHealthAPIKey(raw string) string {
	n := strings.TrimSpace(raw)
	if n == "" {
		return ""
	}
	lower := strings.ToLower(n)
	if strings.HasPrefix(lower, "bearer ") {
		n = strings.TrimSpace(n[7:])
		lower = strings.ToLower(n)
	}
	if i := strings.Index(lower, "api_key:"); i >= 0 {
		n = strings.TrimSpace(n[i+len("api_key:"):])
		n = strings.Trim(n, `"'`)
		lower = strings.ToLower(n)
	}
	n = strings.Trim(n, `"'`)
	if looksLikeAccountHealthAPIKey(n) {
		if strings.HasPrefix(strings.ToLower(n), "bearer ") {
			return strings.TrimSpace(n[7:])
		}
		return n
	}
	return ""
}

func accountHealthKeyHint(apiKey string) string {
	key := strings.TrimSpace(apiKey)
	if strings.HasPrefix(strings.ToLower(key), "bearer ") {
		key = strings.TrimSpace(key[7:])
	}
	prefix := "key"
	lower := strings.ToLower(key)
	switch {
	case strings.HasPrefix(lower, "sk-"):
		prefix = "sk-"
	case strings.HasPrefix(lower, "rk-"):
		prefix = "rk-"
	}
	suffix := key
	if utf8.RuneCountInString(suffix) > 4 {
		runes := []rune(suffix)
		suffix = string(runes[len(runes)-4:])
	}
	return prefix + "••••" + suffix
}

func accountHealthAddSummary(platform string) string {
	platform = strings.TrimSpace(platform)
	if platform == "" {
		platform = PlatformOpenAI
	}
	return "API Key · " + platform
}

func accountHealthAddPreviewReply(count int, chinese bool) string {
	if count <= 0 {
		if chinese {
			return "没有解析到可添加的 API Key。可以粘贴 Key、JSON，或每行一个 Key。凭证只留在本服务，不会发给模型。"
		}
		return "No API keys were found. Paste a key, JSON, or one key per line. Credentials stay on this service and are not sent to the model."
	}
	if chinese {
		return "已解析出待添加账号。凭证只留在本服务，不会发给模型。请在右侧勾选后确认添加。"
	}
	return "Parsed accounts to add. Credentials stay on this service and are not sent to the model. Select them on the right and confirm to add."
}

func firstJSONString(obj map[string]any, keys ...string) string {
	if obj == nil {
		return ""
	}
	for _, key := range keys {
		if value, ok := obj[key]; ok {
			if text, ok := value.(string); ok {
				if trimmed := strings.TrimSpace(text); trimmed != "" {
					return trimmed
				}
			}
		}
	}
	return ""
}

func splitAccountHealthPipes(line string) []string {
	raw := strings.Split(line, "|")
	out := make([]string, 0, len(raw))
	for _, part := range raw {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

func sanitizeAccountHealthAddName(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(trimmed))
	for _, r := range trimmed {
		if r == '\n' || r == '\r' || unicode.IsControl(r) {
			continue
		}
		b.WriteRune(r)
	}
	name := strings.TrimSpace(b.String())
	if utf8.RuneCountInString(name) > 80 {
		runes := []rune(name)
		name = strings.TrimSpace(string(runes[:80]))
	}
	return name
}

func itoaAccountHealth(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
