package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakeAccountHealthAdmin struct {
	accounts  []Account
	created   []CreateAccountInput
	deleted   []int64
	deleteErr map[int64]error
	createErr error
	nextID    int64
}

func (f *fakeAccountHealthAdmin) ListAccounts(_ context.Context, page, pageSize int, platform, _accountType, status, _search string, _groupID int64, _privacyMode, _sortBy, _sortOrder string) ([]Account, int64, error) {
	now := time.Now()
	matched := make([]Account, 0, len(f.accounts))
	for _, acc := range f.accounts {
		if platform != "" && acc.Platform != platform {
			continue
		}
		switch status {
		case StatusError:
			if acc.Status != StatusError {
				continue
			}
		case AccountListStatusExpired:
			if !acc.AutoPauseOnExpired || acc.ExpiresAt == nil || now.Before(*acc.ExpiresAt) {
				continue
			}
		default:
			if status != "" && acc.Status != status {
				continue
			}
		}
		matched = append(matched, acc)
	}
	total := int64(len(matched))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = len(matched)
	}
	start := (page - 1) * pageSize
	if start >= len(matched) {
		return []Account{}, total, nil
	}
	end := start + pageSize
	if end > len(matched) {
		end = len(matched)
	}
	return matched[start:end], total, nil
}

func (f *fakeAccountHealthAdmin) GetAccountsByIDs(_ context.Context, ids []int64) ([]*Account, error) {
	byID := make(map[int64]Account, len(f.accounts))
	for _, acc := range f.accounts {
		byID[acc.ID] = acc
	}
	out := make([]*Account, 0, len(ids))
	for _, id := range ids {
		acc, ok := byID[id]
		if !ok {
			continue
		}
		copy := acc
		out = append(out, &copy)
	}
	return out, nil
}

func (f *fakeAccountHealthAdmin) DeleteAccount(_ context.Context, id int64) error {
	if err := f.deleteErr[id]; err != nil {
		return err
	}
	f.deleted = append(f.deleted, id)
	return nil
}

func (f *fakeAccountHealthAdmin) CreateAccount(_ context.Context, input *CreateAccountInput) (*Account, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	if input == nil {
		return nil, errAccountHealthLLMNoAccount
	}
	f.nextID++
	if f.nextID <= 0 {
		f.nextID = 1
	}
	creds := map[string]any{}
	for k, v := range input.Credentials {
		creds[k] = v
	}
	copied := *input
	copied.Credentials = creds
	f.created = append(f.created, copied)
	acc := Account{
		ID:          f.nextID,
		Name:        input.Name,
		Platform:    input.Platform,
		Type:        input.Type,
		Credentials: creds,
		Status:      StatusActive,
		Schedulable: true,
	}
	f.accounts = append(f.accounts, acc)
	return &acc, nil
}

type fakeAccountHealthTester struct {
	results map[int64]*ScheduledTestResult
	calls   []int64
}

type fakeAccountHealthGrokSSOImporter struct{}

func (fakeAccountHealthGrokSSOImporter) ConvertFromSSO(context.Context, string, *int64) (*GrokTokenInfo, error) {
	return &GrokTokenInfo{AccessToken: "access", RefreshToken: "refresh", Email: "grok@example.com"}, nil
}

func (fakeAccountHealthGrokSSOImporter) BuildAccountCredentials(*GrokTokenInfo) map[string]any {
	return map[string]any{"access_token": "access", "refresh_token": "refresh"}
}

func (f *fakeAccountHealthTester) RunTestBackground(_ context.Context, accountID int64, _ string) (*ScheduledTestResult, error) {
	f.calls = append(f.calls, accountID)
	if result, ok := f.results[accountID]; ok {
		return result, nil
	}
	return &ScheduledTestResult{Status: "success"}, nil
}

func pastExpiry() *time.Time {
	t := time.Now().Add(-2 * time.Hour)
	return &t
}

func futureTime() *time.Time {
	t := time.Now().Add(2 * time.Hour)
	return &t
}

func TestClassifyAccountHealthReasons(t *testing.T) {
	t.Parallel()

	authFailed := &Account{ID: 1, Name: "auth", Status: StatusError, ErrorMessage: "invalid_grant: token revoked"}
	expired := &Account{ID: 2, Name: "expired", Status: StatusActive, AutoPauseOnExpired: true, ExpiresAt: pastExpiry()}
	rateLimited := &Account{ID: 3, Name: "rl", Status: StatusActive, RateLimitResetAt: futureTime()}
	overloaded := &Account{ID: 4, Name: "ol", Status: StatusActive, OverloadUntil: futureTime()}
	temp := &Account{ID: 5, Name: "temp", Status: StatusActive, TempUnschedulableUntil: futureTime()}
	unknown := &Account{ID: 6, Name: "unknown", Status: StatusError, ErrorMessage: "upstream exploded"}
	healthy := &Account{ID: 7, Name: "ok", Status: StatusActive}
	inactive := &Account{ID: 8, Name: "off", Status: StatusDisabled, ErrorMessage: "invalid_api_key"}

	require.Equal(t, AccountHealthReasonAuthFailed, ClassifyAccountHealth(authFailed, nil, false).Reason)
	require.True(t, ClassifyAccountHealth(authFailed, nil, false).Cleanup)
	require.Equal(t, AccountHealthReasonExpired, ClassifyAccountHealth(expired, nil, false).Reason)
	require.True(t, ClassifyAccountHealth(expired, nil, false).Cleanup)
	require.Equal(t, AccountHealthReasonRateLimited, ClassifyAccountHealth(rateLimited, nil, false).Reason)
	require.False(t, ClassifyAccountHealth(rateLimited, nil, false).Cleanup)
	require.Equal(t, AccountHealthReasonOverloaded, ClassifyAccountHealth(overloaded, nil, false).Reason)
	require.Equal(t, AccountHealthReasonTempUnschedulable, ClassifyAccountHealth(temp, nil, false).Reason)
	require.Equal(t, AccountHealthReasonErrorUnknown, ClassifyAccountHealth(unknown, nil, false).Reason)
	require.False(t, ClassifyAccountHealth(unknown, nil, false).Cleanup)
	require.Equal(t, AccountHealthReasonHealthy, ClassifyAccountHealth(healthy, nil, false).Reason)
	require.Equal(t, AccountHealthReasonInactive, ClassifyAccountHealth(inactive, nil, false).Reason)
	require.False(t, ClassifyAccountHealth(inactive, nil, false).Cleanup)

	testedAuth := ClassifyAccountHealth(healthy, &ScheduledTestResult{Status: "failed", ErrorMessage: "401 unauthorized invalid_api_key"}, true)
	require.Equal(t, AccountHealthReasonAuthFailed, testedAuth.Reason)
	require.True(t, testedAuth.Cleanup)
	testedOK := ClassifyAccountHealth(authFailed, &ScheduledTestResult{Status: "success"}, true)
	require.Equal(t, AccountHealthReasonHealthy, testedOK.Reason)
	require.False(t, testedOK.Cleanup)
}

func TestAccountHealthScanDoesNotLeakSecrets(t *testing.T) {
	t.Parallel()
	secret := "sk-secret-leaked-token-123456"
	admin := &fakeAccountHealthAdmin{accounts: []Account{{
		ID:           11,
		Name:         "leaky",
		Platform:     PlatformOpenAI,
		Type:         AccountTypeOAuth,
		Status:       StatusError,
		ErrorMessage: "invalid_api_key " + secret,
		Credentials:  map[string]any{"access_token": secret, "refresh_token": secret},
	}}}
	svc := NewAccountHealthService(admin, nil)
	result, err := svc.Scan(context.Background(), AccountHealthScanRequest{})
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	require.Equal(t, AccountHealthReasonAuthFailed, result.Items[0].Reason)
	body, err := json.Marshal(result)
	require.NoError(t, err)
	require.NotContains(t, string(body), secret)
	require.NotContains(t, string(body), "invalid_api_key")
}

func TestAccountHealthScanWithoutTest(t *testing.T) {
	t.Parallel()
	admin := &fakeAccountHealthAdmin{accounts: []Account{
		{ID: 1, Name: "broken", Status: StatusError, ErrorMessage: "unauthorized"},
		{ID: 2, Name: "expired", Status: StatusActive, AutoPauseOnExpired: true, ExpiresAt: pastExpiry()},
		{ID: 3, Name: "healthy", Status: StatusActive},
		{ID: 4, Name: "limited", Status: StatusActive, RateLimitResetAt: futureTime()},
	}}
	tester := &fakeAccountHealthTester{}
	svc := NewAccountHealthService(admin, tester)
	result, err := svc.Scan(context.Background(), AccountHealthScanRequest{})
	require.NoError(t, err)
	require.Equal(t, 0, result.Tested)
	require.Empty(t, tester.calls)
	require.Equal(t, 2, result.CleanupCount)
	ids := make([]int64, 0, len(result.Items))
	for _, item := range result.Items {
		ids = append(ids, item.AccountID)
	}
	require.Equal(t, []int64{1, 2}, ids)
	require.True(t, result.Items[0].Cleanup)
	require.True(t, result.Items[1].Cleanup)
}

func TestAccountHealthScanWithTestChecksActiveAccounts(t *testing.T) {
	t.Parallel()
	admin := &fakeAccountHealthAdmin{accounts: []Account{
		{ID: 1, Name: "active-broken", Status: StatusActive, Schedulable: true},
		{ID: 2, Name: "active-ok", Status: StatusActive, Schedulable: true},
	}}
	tester := &fakeAccountHealthTester{results: map[int64]*ScheduledTestResult{
		1: {Status: "failed", ErrorMessage: "401 unauthorized"},
		2: {Status: "success"},
	}}
	svc := NewAccountHealthService(admin, tester)
	result, err := svc.Scan(context.Background(), AccountHealthScanRequest{Test: true})
	require.NoError(t, err)
	require.Equal(t, 2, result.Total)
	require.Equal(t, 2, result.Tested)
	require.Equal(t, 1, result.CleanupCount)
	require.Equal(t, AccountHealthReasonAuthFailed, result.Items[0].Reason)
	require.True(t, result.Items[0].Cleanup)
	require.Equal(t, AccountHealthReasonHealthy, result.Items[1].Reason)
}

func TestAccountHealthCleanupRequiresConfirm(t *testing.T) {
	t.Parallel()
	admin := &fakeAccountHealthAdmin{accounts: []Account{{ID: 9, Name: "x", Status: StatusError}}}
	svc := NewAccountHealthService(admin, nil)

	_, err := svc.Cleanup(context.Background(), AccountHealthCleanupRequest{AccountIDs: []int64{9}})
	require.Error(t, err)
	require.Empty(t, admin.deleted)

	_, err = svc.Cleanup(context.Background(), AccountHealthCleanupRequest{AccountIDs: []int64{9}, Confirm: "please"})
	require.Error(t, err)
	require.Empty(t, admin.deleted)

	result, err := svc.Cleanup(context.Background(), AccountHealthCleanupRequest{AccountIDs: []int64{9}, Confirm: AccountHealthConfirmDelete})
	require.NoError(t, err)
	require.Equal(t, []int64{9}, result.Deleted)
	require.Equal(t, []int64{9}, admin.deleted)
}

type scriptedAccountHealthLLM struct {
	calls []AccountHealthLLMRequest
	seq   []scriptedAccountHealthTurn
}

type scriptedAccountHealthTurn struct {
	msg *AccountHealthLLMMessage
	err error
}

func (s *scriptedAccountHealthLLM) Complete(_ context.Context, req AccountHealthLLMRequest) (*AccountHealthLLMMessage, error) {
	s.calls = append(s.calls, req)
	if len(s.seq) == 0 {
		return &AccountHealthLLMMessage{Role: "assistant", Content: "ok"}, nil
	}
	turn := s.seq[0]
	s.seq = s.seq[1:]
	return turn.msg, turn.err
}

func TestAccountHealthChatDoesNotDeleteWithoutPendingIDs(t *testing.T) {
	t.Parallel()
	admin := &fakeAccountHealthAdmin{accounts: []Account{{ID: 42, Name: "target", Status: StatusError}}}
	svc := NewAccountHealthService(admin, nil)

	result, err := svc.Chat(context.Background(), AccountHealthAssistantRequest{
		Messages:        []AccountHealthChatMessage{{Role: "user", Content: "确认删除"}},
		PendingProposal: &AccountHealthProposal{AccountIDs: []int64{42}},
		Confirm:         AccountHealthConfirmDelete,
	})
	require.NoError(t, err)
	require.Equal(t, AccountHealthIntentHelp, result.Intent)
	require.Nil(t, result.Cleanup)
	require.Empty(t, admin.deleted)
	require.Contains(t, result.Reply, "OpenAI")
}

func TestAccountHealthChatScanShortcutWorksWithoutLLM(t *testing.T) {
	t.Parallel()
	admin := &fakeAccountHealthAdmin{accounts: []Account{{ID: 7, Name: "broken", Status: StatusError, ErrorMessage: "unauthorized"}}}
	svc := NewAccountHealthService(admin, nil)
	result, err := svc.Chat(context.Background(), AccountHealthAssistantRequest{
		Intent:   AccountHealthIntentScan,
		Messages: []AccountHealthChatMessage{{Role: "user", Content: "扫描异常账号"}},
	})
	require.NoError(t, err)
	require.Equal(t, AccountHealthIntentScan, result.Intent)
	require.NotNil(t, result.Scan)
	require.Equal(t, 1, result.Scan.CleanupCount)
	require.NotNil(t, result.Proposal)
}

func TestAccountHealthChatScanUsesLLMTools(t *testing.T) {
	t.Parallel()
	admin := &fakeAccountHealthAdmin{accounts: []Account{
		{ID: 1, Name: "broken", Status: StatusError, ErrorMessage: "unauthorized"},
		{ID: 3, Name: "healthy", Status: StatusActive},
	}}
	svc := NewAccountHealthService(admin, nil)
	svc.llm = &scriptedAccountHealthLLM{seq: []scriptedAccountHealthTurn{
		{msg: &AccountHealthLLMMessage{
			Role: "assistant",
			ToolCalls: []AccountHealthLLMToolCall{{
				ID:        "call_scan",
				Name:      accountHealthToolScan,
				Arguments: "{}",
			}},
		}},
		{msg: &AccountHealthLLMMessage{Role: "assistant", Content: "已筛出过期账号，请在右侧确认。"}},
	}}

	result, err := svc.Chat(context.Background(), AccountHealthAssistantRequest{
		Messages: []AccountHealthChatMessage{{Role: "user", Content: "扫描过期账号"}},
	})
	require.NoError(t, err)
	require.Equal(t, AccountHealthIntentScan, result.Intent)
	require.Nil(t, result.Cleanup)
	require.Empty(t, admin.deleted)
	require.NotNil(t, result.Scan)
	require.Equal(t, 1, result.Scan.CleanupCount)
	require.NotNil(t, result.Proposal)
	require.Equal(t, []int64{1}, result.Proposal.AccountIDs)
	require.Equal(t, "已筛出过期账号，请在右侧确认。", result.Reply)
	require.Len(t, result.ToolTraces, 1)
	require.Equal(t, accountHealthToolScan, result.ToolTraces[0].Name)
	require.Equal(t, "ok", result.ToolTraces[0].Status)
}

func TestAccountHealthChatDeleteTextDoesNotDelete(t *testing.T) {
	t.Parallel()
	admin := &fakeAccountHealthAdmin{accounts: []Account{{ID: 42, Name: "target", Status: StatusError}}}
	svc := NewAccountHealthService(admin, nil)
	svc.llm = &scriptedAccountHealthLLM{seq: []scriptedAccountHealthTurn{
		{msg: &AccountHealthLLMMessage{Role: "assistant", Content: "请先扫描再确认"}},
	}}

	result, err := svc.Chat(context.Background(), AccountHealthAssistantRequest{
		Messages:   []AccountHealthChatMessage{{Role: "user", Content: "删除 42"}},
		AccountIDs: []int64{42},
		Confirm:    AccountHealthConfirmDelete,
	})
	require.NoError(t, err)
	require.Nil(t, result.Cleanup)
	require.Empty(t, admin.deleted)
}

func TestAccountHealthChatIgnoresDeleteNamedTool(t *testing.T) {
	t.Parallel()
	admin := &fakeAccountHealthAdmin{accounts: []Account{{ID: 42, Name: "target", Status: StatusError}}}
	svc := NewAccountHealthService(admin, nil)
	svc.llm = &scriptedAccountHealthLLM{seq: []scriptedAccountHealthTurn{
		{msg: &AccountHealthLLMMessage{
			Role: "assistant",
			ToolCalls: []AccountHealthLLMToolCall{{
				ID:        "call_delete",
				Name:      "delete_accounts",
				Arguments: `{"account_ids":[42]}`,
			}},
		}},
		{msg: &AccountHealthLLMMessage{Role: "assistant", Content: "ignored"}},
	}}

	result, err := svc.Chat(context.Background(), AccountHealthAssistantRequest{
		Messages: []AccountHealthChatMessage{{Role: "user", Content: "删除这些账号"}},
	})
	require.NoError(t, err)
	require.Nil(t, result.Cleanup)
	require.Empty(t, admin.deleted)
	require.Nil(t, result.Scan)
	require.Len(t, result.ToolTraces, 1)
	require.Equal(t, "delete_accounts", result.ToolTraces[0].Name)
	require.Equal(t, "error", result.ToolTraces[0].Status)
}

func TestParseAccountHealthIntent(t *testing.T) {
	t.Parallel()
	require.Equal(t, AccountHealthIntentScan, parseAccountHealthIntent("", "扫描异常账号"))
	require.Equal(t, AccountHealthIntentTest, parseAccountHealthIntent("", "测试并筛选"))
	require.Equal(t, AccountHealthIntentCleanup, parseAccountHealthIntent("", "确认删除"))
	require.Equal(t, AccountHealthIntentHelp, parseAccountHealthIntent("", "hello"))
	require.Equal(t, AccountHealthIntentScan, parseAccountHealthIntent("scan", "删除 1"))
}

func TestNormalizeAccountHealthSignalKeepsAuthPhrases(t *testing.T) {
	t.Parallel()
	require.Equal(t, AccountHealthReasonAuthFailed, classifyAccountHealthSignals("Authentication failed (401): code=token_expired"))
	require.Equal(t, AccountHealthReasonAuthFailed, classifyAccountHealthSignals("invalid_grant"))
	require.Equal(t, AccountHealthReasonRateLimited, classifyAccountHealthSignals("HTTP 429 too many requests"))
}

func TestUniquePositiveIDsPreservesOrderAndDropsJunk(t *testing.T) {
	t.Parallel()
	require.Equal(t, []int64{3, 1}, uniquePositiveIDs([]int64{3, 0, 3, -2, 1, 1}, 10))
	require.Equal(t, []int64{8, 9}, uniquePositiveIDs([]int64{8, 9, 10}, 2))
}

func TestAccountHealthChatScanFallsBackWhenLLMLeaksSecret(t *testing.T) {
	t.Parallel()
	admin := &fakeAccountHealthAdmin{accounts: []Account{
		{ID: 1, Name: "broken", Status: StatusError, ErrorMessage: "unauthorized"},
	}}
	svc := NewAccountHealthService(admin, nil)
	svc.llm = &scriptedAccountHealthLLM{seq: []scriptedAccountHealthTurn{
		{msg: &AccountHealthLLMMessage{
			Role: "assistant",
			ToolCalls: []AccountHealthLLMToolCall{{
				ID:        "call_scan",
				Name:      accountHealthToolScan,
				Arguments: "{}",
			}},
		}},
		{msg: &AccountHealthLLMMessage{Role: "assistant", Content: "found sk-secret-token-123456, delete it"}},
	}}

	result, err := svc.Chat(context.Background(), AccountHealthAssistantRequest{
		Messages: []AccountHealthChatMessage{{Role: "user", Content: "扫描过期账号"}},
	})
	require.NoError(t, err)
	require.Equal(t, AccountHealthIntentScan, result.Intent)
	require.NotNil(t, result.Scan)
	require.Contains(t, result.Reply, "扫描")
	require.NotContains(t, result.Reply, "sk-")
	require.NotContains(t, result.Reply, "secret-token")
}

func TestSanitizeAccountHealthLLMText(t *testing.T) {
	t.Parallel()
	require.Equal(t, "已筛出过期账号，请在右侧确认。", sanitizeAccountHealthLLMText("已筛出过期账号，请在右侧确认。"))
	require.Equal(t, "token expired, invalid_api_key was classified", sanitizeAccountHealthLLMText("token expired, invalid_api_key was classified"))
	require.Empty(t, sanitizeAccountHealthLLMText("here is sk-abc123"))
	require.Empty(t, sanitizeAccountHealthLLMText("Authorization: Bearer xyz"))
	require.Empty(t, sanitizeAccountHealthLLMText("access_token=abc"))
}

func TestSelectAccountHealthLLMModel(t *testing.T) {
	t.Parallel()
	require.Equal(t, accountHealthLLMDefaultModel, selectAccountHealthLLMModel(nil))
	require.Equal(t, accountHealthLLMDefaultModel, selectAccountHealthLLMModel(&Account{}))

	mappedMini := &Account{Credentials: map[string]any{
		"model_mapping": map[string]any{"gpt-4o-mini": "pool-mini"},
	}}
	require.Equal(t, "pool-mini", selectAccountHealthLLMModel(mappedMini))

	mappedOther := &Account{Credentials: map[string]any{
		"model_mapping": map[string]any{"custom": "vendor-chat"},
	}}
	require.Equal(t, "vendor-chat", selectAccountHealthLLMModel(mappedOther))

	skipImage := &Account{Credentials: map[string]any{
		"model_mapping": map[string]any{"image": "dall-e-3", "chat": "vendor-chat"},
	}}
	require.Equal(t, "vendor-chat", selectAccountHealthLLMModel(skipImage))
}

func TestParseAccountHealthHandoffJSONAndLines(t *testing.T) {
	t.Parallel()
	jsonItems := parseAccountHealthHandoff(`[
		{"name":"alpha","platform":"openai","api_key":"sk-secret-token-aaaa","base_url":"https://api.openai.com"},
		{"api_key":"sk-secret-token-bbbb"}
	]`, "")
	require.Len(t, jsonItems, 2)
	require.Equal(t, 0, jsonItems[0].Item.Index)
	require.Equal(t, "alpha", jsonItems[0].Item.Name)
	require.Equal(t, PlatformOpenAI, jsonItems[0].Item.Platform)
	require.Equal(t, AccountTypeAPIKey, jsonItems[0].Item.Type)
	require.Equal(t, "https://api.openai.com", jsonItems[0].Item.BaseURL)
	require.Equal(t, "sk-••••aaaa", jsonItems[0].Item.KeyHint)
	require.Equal(t, "imported-openai-1", jsonItems[1].Item.Name)

	lineItems := parseAccountHealthHandoff("sk-secret-token-cccc\ngrok rk-secret-token-dddd https://api.x.ai\nbot|openai|sk-secret-token-eeee|https://relay.example", "")
	require.Len(t, lineItems, 3)
	require.Equal(t, PlatformGrok, lineItems[1].Item.Platform)
	require.Equal(t, "https://api.x.ai", lineItems[1].Item.BaseURL)
	require.Equal(t, "bot", lineItems[2].Item.Name)
	require.Equal(t, "https://relay.example", lineItems[2].Item.BaseURL)
}

func TestAccountHealthIngestDoesNotReturnRawKey(t *testing.T) {
	t.Parallel()
	secret := "sk-secret-token-zzzz"
	proposal := ingestAccountHealthHandoff(secret+" https://api.openai.com", "")
	require.Len(t, proposal.Items, 1)
	raw, err := json.Marshal(proposal)
	require.NoError(t, err)
	require.NotContains(t, string(raw), secret)
	require.Contains(t, string(raw), "••••")
	require.Equal(t, "sk-••••zzzz", proposal.Items[0].KeyHint)
}

func TestParseAccountHealthHandoffRecognizesGrokSSO(t *testing.T) {
	t.Parallel()
	token := "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.signature"
	items := parseAccountHealthHandoff(token+"\n", "")
	require.Len(t, items, 1)
	require.Equal(t, PlatformGrok, items[0].Item.Platform)
	require.Equal(t, AccountTypeOAuth, items[0].Item.Type)
	require.Equal(t, "grok_sso", items[0].Item.AuthMethod)
	require.NotContains(t, items[0].Item.KeyHint, token)
	raw, err := json.Marshal(ingestAccountHealthHandoff(token, ""))
	require.NoError(t, err)
	require.NotContains(t, string(raw), token)
}

func TestParseAccountHealthHandoffRecognizesRegistrationFullExport(t *testing.T) {
	t.Parallel()
	token := "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.signature"
	items := parseAccountHealthHandoff("person@example.com----secret-password----"+token+"\n", PlatformGrok)
	require.Len(t, items, 1)
	require.Equal(t, PlatformGrok, items[0].Item.Platform)
	require.Equal(t, "person@example.com", items[0].Item.Name)
	require.Equal(t, "grok_sso", items[0].Item.AuthMethod)
	require.NotContains(t, items[0].Item.KeyHint, "secret-password")
}

func TestAccountHealthAddConvertsGrokSSOAndAppliesRouting(t *testing.T) {
	t.Parallel()
	admin := &fakeAccountHealthAdmin{}
	svc := NewAccountHealthService(admin, nil)
	svc.SetGrokSSOImporter(fakeAccountHealthGrokSSOImporter{})
	proxyID := int64(7)
	token := "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.signature"
	result, err := svc.Add(context.Background(), AccountHealthAddRequest{
		Handoff: token, Indexes: []int{0}, Confirm: AccountHealthConfirmAdd,
		ProxyID: &proxyID, GroupIDs: []int64{11, 12},
	})
	require.NoError(t, err)
	require.Len(t, result.Created, 1)
	require.Len(t, admin.created, 1)
	require.Equal(t, AccountTypeOAuth, admin.created[0].Type)
	require.Equal(t, PlatformGrok, admin.created[0].Platform)
	require.Equal(t, &proxyID, admin.created[0].ProxyID)
	require.Equal(t, []int64{11, 12}, admin.created[0].GroupIDs)
	require.Equal(t, "refresh", admin.created[0].Credentials["refresh_token"])
}

func TestAccountHealthChatHandoffDoesNotCallLLMOrCreate(t *testing.T) {
	t.Parallel()
	admin := &fakeAccountHealthAdmin{}
	llm := &scriptedAccountHealthLLM{}
	svc := NewAccountHealthService(admin, nil)
	svc.llm = llm

	secret := "sk-secret-token-hhhh"
	result, err := svc.Chat(context.Background(), AccountHealthAssistantRequest{
		Messages: []AccountHealthChatMessage{{Role: "user", Content: "已提交待添加的账号凭证"}},
		Handoff:  secret,
	})
	require.NoError(t, err)
	require.Equal(t, AccountHealthIntentAdd, result.Intent)
	require.NotNil(t, result.AddProposal)
	require.Len(t, result.AddProposal.Items, 1)
	require.Empty(t, llm.calls)
	require.Empty(t, admin.created)
	raw, err := json.Marshal(result)
	require.NoError(t, err)
	require.NotContains(t, string(raw), secret)

	result, err = svc.Chat(context.Background(), AccountHealthAssistantRequest{
		Messages: []AccountHealthChatMessage{{Role: "user", Content: secret}},
	})
	require.NoError(t, err)
	require.Equal(t, AccountHealthIntentAdd, result.Intent)
	require.Empty(t, llm.calls)
	require.Empty(t, admin.created)
}

func TestAccountHealthAddRequiresConfirmThenCreates(t *testing.T) {
	t.Parallel()
	admin := &fakeAccountHealthAdmin{}
	svc := NewAccountHealthService(admin, nil)
	secret := "sk-secret-token-wwww"

	_, err := svc.Add(context.Background(), AccountHealthAddRequest{
		Handoff: secret,
		Indexes: []int{0},
	})
	require.Error(t, err)
	require.Empty(t, admin.created)

	result, err := svc.Add(context.Background(), AccountHealthAddRequest{
		Handoff: secret,
		Indexes: []int{0},
		Confirm: AccountHealthConfirmAdd,
	})
	require.NoError(t, err)
	require.Len(t, result.Created, 1)
	require.Empty(t, result.Failed)
	require.Len(t, admin.created, 1)
	require.Equal(t, secret, admin.created[0].Credentials["api_key"])
	require.Equal(t, AccountTypeAPIKey, admin.created[0].Type)
	raw, err := json.Marshal(result)
	require.NoError(t, err)
	require.NotContains(t, string(raw), secret)
	require.Equal(t, "sk-••••wwww", result.Created[0].KeyHint)
}

func TestAccountHealthChatStillCannotDelete(t *testing.T) {
	t.Parallel()
	admin := &fakeAccountHealthAdmin{accounts: []Account{{ID: 42, Name: "target", Status: StatusError}}}
	svc := NewAccountHealthService(admin, nil)
	svc.llm = &scriptedAccountHealthLLM{seq: []scriptedAccountHealthTurn{
		{msg: &AccountHealthLLMMessage{Role: "assistant", Content: "ok"}},
	}}
	result, err := svc.Chat(context.Background(), AccountHealthAssistantRequest{
		Messages:   []AccountHealthChatMessage{{Role: "user", Content: "确认删除 42"}},
		AccountIDs: []int64{42},
		Confirm:    AccountHealthConfirmDelete,
	})
	require.NoError(t, err)
	require.Nil(t, result.Cleanup)
	require.Empty(t, admin.deleted)
}

func TestAccountHealthChatExplicitTestBypassesLLMAndChecksActivePool(t *testing.T) {
	t.Parallel()
	admin := &fakeAccountHealthAdmin{accounts: []Account{
		{ID: 1, Name: "active", Status: StatusActive},
		{ID: 2, Name: "broken", Status: StatusActive},
	}}
	tester := &fakeAccountHealthTester{results: map[int64]*ScheduledTestResult{
		1: {Status: "success"},
		2: {Status: "failed", ErrorMessage: "401 unauthorized"},
	}}
	svc := NewAccountHealthService(admin, tester)
	svc.llm = &scriptedAccountHealthLLM{seq: []scriptedAccountHealthTurn{{msg: &AccountHealthLLMMessage{Role: "assistant", Content: "should not be called"}}}}
	result, err := svc.Chat(context.Background(), AccountHealthAssistantRequest{
		Intent:   AccountHealthIntentTest,
		Messages: []AccountHealthChatMessage{{Role: "user", Content: "扫描全部账号并实时检查"}},
	})
	require.NoError(t, err)
	require.Equal(t, AccountHealthIntentTest, result.Intent)
	require.NotNil(t, result.Scan)
	require.Equal(t, 2, result.Scan.Total)
	require.Equal(t, 1, result.Scan.CleanupCount)
}
