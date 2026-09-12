package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	accountHealthLLMMaxRounds     = 3
	accountHealthLLMTimeout       = 60 * time.Second
	accountHealthLLMMaxReplyRunes = 1200
	accountHealthToolScan         = "scan_account_pool"
	accountHealthToolTest         = "test_account_pool"
	accountHealthLLMPickLimit     = 50
	accountHealthLLMDefaultModel  = "gpt-4o-mini"
)

var errAccountHealthLLMNoAccount = errors.New("account health llm unavailable")

// AccountHealthLLM is the constrained completion surface used by the pool assistant.
// Production uses a healthy OpenAI-compatible API-key account from the pool.
type AccountHealthLLM interface {
	Complete(ctx context.Context, req AccountHealthLLMRequest) (*AccountHealthLLMMessage, error)
}

type AccountHealthLLMToolSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type AccountHealthLLMToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type AccountHealthLLMMessage struct {
	Role       string                     `json:"role"`
	Content    string                     `json:"content,omitempty"`
	ToolCallID string                     `json:"tool_call_id,omitempty"`
	Name       string                     `json:"name,omitempty"`
	ToolCalls  []AccountHealthLLMToolCall `json:"tool_calls,omitempty"`
}

type AccountHealthLLMRequest struct {
	Messages []AccountHealthLLMMessage
	Tools    []AccountHealthLLMToolSpec
}

type poolAccountHealthLLM struct {
	admin  AccountHealthAdmin
	client *http.Client
}

func newPoolAccountHealthLLM(admin AccountHealthAdmin, client *http.Client) *poolAccountHealthLLM {
	if client == nil {
		client = &http.Client{Timeout: accountHealthLLMTimeout}
	}
	return &poolAccountHealthLLM{admin: admin, client: client}
}

func accountHealthLLMTools() []AccountHealthLLMToolSpec {
	params := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"platform": map[string]any{
				"type":        "string",
				"description": "Optional platform filter such as openai or claude.",
			},
			"account_ids": map[string]any{
				"type":        "array",
				"description": "Optional account IDs to inspect. Never treat this as a delete list.",
				"items":       map[string]any{"type": "integer"},
			},
		},
	}
	return []AccountHealthLLMToolSpec{
		{
			Name:        accountHealthToolScan,
			Description: "Scan the account pool for expired or authentication-failed accounts. Returns a review list. Does not delete anything.",
			Parameters:  params,
		},
		{
			Name:        accountHealthToolTest,
			Description: "Live-test accounts in the pool, including active accounts whose credentials may have just become unusable, and return a review list marking only expired or authentication-failed ones for cleanup. Does not delete anything.",
			Parameters:  params,
		},
	}
}

func (s *AccountHealthService) chatWithLLM(ctx context.Context, req AccountHealthAssistantRequest, chinese bool) (*AccountHealthAssistantResult, error) {
	if s == nil || s.llm == nil {
		return &AccountHealthAssistantResult{
			Intent: AccountHealthIntentHelp,
			Reply:  accountHealthNeedWorkingKeyReply(chinese),
		}, nil
	}

	messages := buildAccountHealthLLMMessages(req, chinese)
	tools := accountHealthLLMTools()
	var (
		lastScan   *AccountHealthScanResult
		lastIntent = AccountHealthIntentHelp
		traces     []AccountHealthToolTrace
		lastText   string
	)

	for round := 0; round < accountHealthLLMMaxRounds; round++ {
		if err := ctx.Err(); err != nil {
			return accountHealthLLMPartialResult(lastIntent, lastScan, traces, chinese, accountHealthLLMFailedReply(chinese)), nil
		}
		out, err := s.llm.Complete(ctx, AccountHealthLLMRequest{Messages: messages, Tools: tools})
		if err != nil {
			reply := accountHealthLLMFailedReply(chinese)
			if errors.Is(err, errAccountHealthLLMNoAccount) {
				reply = accountHealthNeedWorkingKeyReply(chinese)
			}
			return accountHealthLLMPartialResult(lastIntent, lastScan, traces, chinese, reply), nil
		}
		if out == nil {
			break
		}
		lastText = strings.TrimSpace(out.Content)
		if len(out.ToolCalls) == 0 {
			break
		}

		assistant := *out
		assistant.Role = "assistant"
		messages = append(messages, assistant)
		for _, call := range out.ToolCalls {
			scan, intent, trace := s.executeAccountHealthTool(ctx, call, req.Platform)
			traces = append(traces, trace)
			if intent != "" {
				lastIntent = intent
			}
			if scan != nil {
				lastScan = scan
			}
			messages = append(messages, AccountHealthLLMMessage{
				Role:       "tool",
				ToolCallID: strings.TrimSpace(call.ID),
				Name:       strings.TrimSpace(call.Name),
				Content:    marshalAccountHealthToolResult(scan, trace),
			})
		}
	}

	reply := sanitizeAccountHealthLLMText(lastText)
	if reply == "" {
		if lastScan != nil {
			reply = accountHealthScanReply(lastScan, lastIntent == AccountHealthIntentTest, chinese)
		} else {
			reply = accountHealthHelpReply(chinese)
		}
	}

	return &AccountHealthAssistantResult{
		Intent:     lastIntent,
		Reply:      reply,
		Scan:       lastScan,
		Proposal:   proposalFromOptionalScan(lastScan),
		ToolTraces: traces,
	}, nil
}

func accountHealthLLMPartialResult(intent string, scan *AccountHealthScanResult, traces []AccountHealthToolTrace, chinese bool, reply string) *AccountHealthAssistantResult {
	if scan != nil {
		reply = accountHealthScanReply(scan, intent == AccountHealthIntentTest, chinese)
	}
	return &AccountHealthAssistantResult{
		Intent:     intent,
		Reply:      reply,
		Scan:       scan,
		Proposal:   proposalFromOptionalScan(scan),
		ToolTraces: traces,
	}
}

func proposalFromOptionalScan(scan *AccountHealthScanResult) *AccountHealthProposal {
	if scan == nil {
		return nil
	}
	return proposalFromScan(scan)
}

func (s *AccountHealthService) executeAccountHealthTool(ctx context.Context, call AccountHealthLLMToolCall, fallbackPlatform string) (*AccountHealthScanResult, string, AccountHealthToolTrace) {
	name := strings.TrimSpace(call.Name)
	trace := AccountHealthToolTrace{Name: name, Status: "error", Summary: "tool failed"}
	switch name {
	case accountHealthToolScan, accountHealthToolTest:
	default:
		trace.Name = name
		trace.Summary = "unsupported tool"
		return nil, "", trace
	}

	platform, ids := parseAccountHealthToolArgs(call.Arguments)
	if platform == "" {
		platform = strings.TrimSpace(fallbackPlatform)
	}
	scan, err := s.Scan(ctx, AccountHealthScanRequest{
		Platform:   platform,
		AccountIDs: ids,
		Test:       name == accountHealthToolTest,
	})
	if err != nil {
		return nil, "", trace
	}
	intent := AccountHealthIntentScan
	summary := "scanned account pool"
	if name == accountHealthToolTest {
		intent = AccountHealthIntentTest
		summary = "tested account pool"
	}
	if scan != nil {
		trace = AccountHealthToolTrace{
			Name:    name,
			Status:  "ok",
			Summary: summary,
		}
	}
	return scan, intent, trace
}

func parseAccountHealthToolArgs(raw string) (string, []int64) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" {
		return "", nil
	}
	var args struct {
		Platform   string  `json:"platform"`
		AccountIDs []int64 `json:"account_ids"`
	}
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		return "", nil
	}
	return strings.TrimSpace(args.Platform), uniquePositiveIDs(args.AccountIDs, MaxAccountHealthScanLimit)
}

func marshalAccountHealthToolResult(scan *AccountHealthScanResult, trace AccountHealthToolTrace) string {
	payload := map[string]any{
		"ok":     trace.Status == "ok",
		"tool":   trace.Name,
		"status": trace.Status,
	}
	if scan != nil {
		items := make([]map[string]any, 0, len(scan.Items))
		for _, item := range scan.Items {
			items = append(items, map[string]any{
				"account_id": item.AccountID,
				"name":       item.Name,
				"platform":   item.Platform,
				"reason":     item.Reason,
				"summary":    item.Summary,
				"cleanup":    item.Cleanup,
				"tested":     item.Tested,
			})
		}
		payload["total"] = scan.Total
		payload["tested"] = scan.Tested
		payload["cleanup_count"] = scan.CleanupCount
		payload["truncated"] = scan.Truncated
		payload["items"] = items
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return `{"ok":false}`
	}
	return string(body)
}

func buildAccountHealthLLMMessages(req AccountHealthAssistantRequest, chinese bool) []AccountHealthLLMMessage {
	system := accountHealthLLMSystemPrompt(chinese)
	if n := len(pendingProposalIDs(req.PendingProposal)); n > 0 {
		if chinese {
			system += " 当前待确认列表里已有筛选结果，管理员需要在界面勾选并确认后才会删除。你不能删除。"
		} else {
			system += " A review list already exists. The admin must select accounts in the UI and confirm before any delete. You cannot delete."
		}
	}
	out := []AccountHealthLLMMessage{{Role: "system", Content: system}}
	for _, msg := range req.Messages {
		role := strings.ToLower(strings.TrimSpace(msg.Role))
		content := sanitizeAccountHealthLLMText(msg.Content)
		if content == "" {
			continue
		}
		if role != "user" && role != "assistant" {
			continue
		}
		out = append(out, AccountHealthLLMMessage{Role: role, Content: content})
	}
	if len(out) == 1 {
		fallback := strings.TrimSpace(req.Intent)
		if fallback == "" {
			fallback = accountHealthHelpReply(chinese)
		}
		out = append(out, AccountHealthLLMMessage{Role: "user", Content: fallback})
	}
	return out
}

func accountHealthLLMSystemPrompt(chinese bool) string {
	if chinese {
		return "你是账号池运维助手。只能调用扫描或测试工具，找出过期或鉴权失败的账号并交给管理员在界面确认。管理员可以把 API Key 交给本服务添加，凭证不会发给你；你不要向聊天索要密钥，也不要复述密钥。添加和删除都要管理员在右侧确认。禁止删除、禁止复述原始报错。限流、额度用尽、过载、暂时不可调度的账号不要当作删除对象。用管理员的语言简短回复。"
	}
	return "You are the account-pool ops assistant. You may only call scan or test tools to find expired or authentication-failed accounts and hand them to the admin for UI confirmation. Admins can hand API keys to this service to add accounts; those credentials are not sent to you. Never ask for secrets, never repeat them, and never delete. Add and delete both require confirmation in the review panel. Never repeat raw error strings. Rate limits, quota exhaustion, overload, and temporary unschedulability are not delete reasons. Reply briefly in the admin's language."
}

func sanitizeAccountHealthLLMText(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	lower := strings.ToLower(trimmed)
	blocked := []string{
		"sk-", "rk-", "bearer ", "authorization:",
		"api_key:", `"api_key"`, "access_token", "refresh_token", "id_token",
		"client_secret", "session_token",
	}
	for _, needle := range blocked {
		if strings.Contains(lower, needle) {
			return ""
		}
	}
	if utf8.RuneCountInString(trimmed) > accountHealthLLMMaxReplyRunes {
		runes := []rune(trimmed)
		trimmed = strings.TrimSpace(string(runes[:accountHealthLLMMaxReplyRunes]))
	}
	var b strings.Builder
	b.Grow(len(trimmed))
	for _, r := range trimmed {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			continue
		}
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

func (l *poolAccountHealthLLM) Complete(ctx context.Context, req AccountHealthLLMRequest) (*AccountHealthLLMMessage, error) {
	if l == nil || l.admin == nil {
		return nil, errAccountHealthLLMNoAccount
	}
	account, endpoint, apiKey, modelID, err := l.pickLLMAccount(ctx)
	if err != nil {
		return nil, err
	}
	_ = account
	payload, err := buildAccountHealthLLMPayload(req, modelID)
	if err != nil {
		return nil, errAccountHealthLLMNoAccount
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, errAccountHealthLLMNoAccount
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	client := l.client
	if client == nil {
		client = &http.Client{Timeout: accountHealthLLMTimeout}
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, errAccountHealthLLMNoAccount
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, errAccountHealthLLMNoAccount
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errAccountHealthLLMNoAccount
	}
	msg, err := parseAccountHealthLLMResponse(body)
	if err != nil {
		return nil, errAccountHealthLLMNoAccount
	}
	return msg, nil
}

func (l *poolAccountHealthLLM) pickLLMAccount(ctx context.Context) (*Account, string, string, string, error) {
	accounts, _, err := l.admin.ListAccounts(ctx, 1, accountHealthLLMPickLimit, "", AccountTypeAPIKey, StatusActive, "", 0, "", "priority", "desc")
	if err != nil {
		return nil, "", "", "", errAccountHealthLLMNoAccount
	}
	var chosen *Account
	for i := range accounts {
		acc := &accounts[i]
		if !isHealthyAccountHealthLLMAccount(acc) {
			continue
		}
		if acc.Platform == PlatformOpenAI {
			chosen = acc
			break
		}
		if chosen == nil {
			chosen = acc
		}
	}
	if chosen == nil {
		return nil, "", "", "", errAccountHealthLLMNoAccount
	}
	endpoint, apiKey := accountHealthLLMEndpoint(chosen)
	if endpoint == "" || apiKey == "" {
		return nil, "", "", "", errAccountHealthLLMNoAccount
	}
	return chosen, endpoint, apiKey, selectAccountHealthLLMModel(chosen), nil
}

func selectAccountHealthLLMModel(account *Account) string {
	if account == nil {
		return accountHealthLLMDefaultModel
	}
	mapping := account.GetModelMapping()
	preferred := []string{
		"gpt-4o-mini", "gpt-4.1-mini", "gpt-4o", "gpt-4.1", "gpt-4-turbo", "gpt-4",
	}
	pick := func(raw string) string {
		raw = strings.TrimSpace(raw)
		if raw == "" || strings.Contains(raw, "*") || !isAccountHealthChatModel(raw) {
			return ""
		}
		return raw
	}
	for _, want := range preferred {
		for key, upstream := range mapping {
			if strings.EqualFold(strings.TrimSpace(key), want) {
				if model := pick(upstream); model != "" {
					return model
				}
			}
			if strings.EqualFold(strings.TrimSpace(upstream), want) {
				if model := pick(upstream); model != "" {
					return model
				}
			}
		}
	}
	candidates := make([]string, 0, len(mapping))
	seen := make(map[string]struct{}, len(mapping))
	for _, upstream := range mapping {
		model := pick(upstream)
		if model == "" {
			continue
		}
		if _, ok := seen[model]; ok {
			continue
		}
		seen[model] = struct{}{}
		candidates = append(candidates, model)
	}
	if len(candidates) == 0 {
		return accountHealthLLMDefaultModel
	}
	sort.Strings(candidates)
	return candidates[0]
}

func isAccountHealthChatModel(id string) bool {
	lower := strings.ToLower(strings.TrimSpace(id))
	if lower == "" {
		return false
	}
	skip := []string{"whisper", "tts-", "dall-e", "embedding", "moderation", "realtime", "image"}
	for _, needle := range skip {
		if strings.Contains(lower, needle) {
			return false
		}
	}
	return true
}

func isHealthyAccountHealthLLMAccount(account *Account) bool {
	if account == nil || account.Type != AccountTypeAPIKey || !account.IsOpenAICompatible() {
		return false
	}
	if account.Status == StatusError || account.Status == StatusDisabled {
		return false
	}
	if !account.IsSchedulable() {
		return false
	}
	if ClassifyAccountHealth(account, nil, false).Cleanup {
		return false
	}
	endpoint, apiKey := accountHealthLLMEndpoint(account)
	return endpoint != "" && apiKey != ""
}

func accountHealthLLMEndpoint(account *Account) (string, string) {
	if account == nil {
		return "", ""
	}
	apiKey := strings.TrimSpace(account.GetOpenAIProtocolAPIKey())
	if apiKey == "" {
		return "", ""
	}
	base := strings.TrimSpace(account.GetOpenAIFormatBaseURL())
	if base == "" && account.IsGrok() {
		base = strings.TrimSpace(account.GetGrokBaseURL())
	}
	if base == "" {
		return "", ""
	}
	return buildOpenAIChatCompletionsURLForPlatform(account.Platform, base), apiKey
}

func buildAccountHealthLLMPayload(req AccountHealthLLMRequest, modelID string) ([]byte, error) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		modelID = accountHealthLLMDefaultModel
	}
	type functionCall struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	}
	type toolCall struct {
		ID       string       `json:"id,omitempty"`
		Type     string       `json:"type"`
		Function functionCall `json:"function"`
	}
	type message struct {
		Role       string     `json:"role"`
		Content    any        `json:"content"`
		ToolCallID string     `json:"tool_call_id,omitempty"`
		Name       string     `json:"name,omitempty"`
		ToolCalls  []toolCall `json:"tool_calls,omitempty"`
	}
	type tool struct {
		Type     string `json:"type"`
		Function struct {
			Name        string         `json:"name"`
			Description string         `json:"description"`
			Parameters  map[string]any `json:"parameters"`
		} `json:"function"`
	}

	messages := make([]message, 0, len(req.Messages))
	for _, msg := range req.Messages {
		item := message{
			Role:       msg.Role,
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID,
			Name:       msg.Name,
		}
		if len(msg.ToolCalls) > 0 {
			item.Content = nil
			item.ToolCalls = make([]toolCall, 0, len(msg.ToolCalls))
			for _, call := range msg.ToolCalls {
				args := strings.TrimSpace(call.Arguments)
				if args == "" {
					args = "{}"
				}
				item.ToolCalls = append(item.ToolCalls, toolCall{
					ID:   call.ID,
					Type: "function",
					Function: functionCall{
						Name:      call.Name,
						Arguments: args,
					},
				})
			}
		}
		messages = append(messages, item)
	}

	tools := make([]tool, 0, len(req.Tools))
	for _, spec := range req.Tools {
		item := tool{Type: "function"}
		item.Function.Name = spec.Name
		item.Function.Description = spec.Description
		item.Function.Parameters = spec.Parameters
		tools = append(tools, item)
	}

	return json.Marshal(map[string]any{
		"model":       modelID,
		"messages":    messages,
		"tools":       tools,
		"tool_choice": "auto",
		"temperature": 0,
	})
}

func parseAccountHealthLLMResponse(body []byte) (*AccountHealthLLMMessage, error) {
	var parsed struct {
		Choices []struct {
			Message struct {
				Role      string `json:"role"`
				Content   any    `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Choices) == 0 {
		return nil, errAccountHealthLLMNoAccount
	}
	raw := parsed.Choices[0].Message
	msg := &AccountHealthLLMMessage{
		Role:    strings.TrimSpace(raw.Role),
		Content: stringifyAccountHealthLLMContent(raw.Content),
	}
	if msg.Role == "" {
		msg.Role = "assistant"
	}
	for _, call := range raw.ToolCalls {
		name := strings.TrimSpace(call.Function.Name)
		if name == "" {
			continue
		}
		msg.ToolCalls = append(msg.ToolCalls, AccountHealthLLMToolCall{
			ID:        strings.TrimSpace(call.ID),
			Name:      name,
			Arguments: strings.TrimSpace(call.Function.Arguments),
		})
	}
	if strings.TrimSpace(msg.Content) == "" && len(msg.ToolCalls) == 0 {
		return nil, errAccountHealthLLMNoAccount
	}
	return msg, nil
}

func stringifyAccountHealthLLMContent(content any) string {
	switch value := content.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(value)
	default:
		return ""
	}
}
