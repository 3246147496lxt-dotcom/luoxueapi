package service

import (
	"context"
	"strings"
	"sync"
	"time"
	"unicode"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	AccountHealthReasonAuthFailed        = "auth_failed"
	AccountHealthReasonExpired           = "expired"
	AccountHealthReasonRateLimited       = "rate_limited"
	AccountHealthReasonQuotaExhausted    = "quota_exhausted"
	AccountHealthReasonOverloaded        = "overloaded"
	AccountHealthReasonTempUnschedulable = "temp_unschedulable"
	AccountHealthReasonErrorUnknown      = "error_unknown"
	AccountHealthReasonHealthy           = "healthy"
	AccountHealthReasonInactive          = "inactive"

	AccountHealthIntentScan    = "scan"
	AccountHealthIntentTest    = "test"
	AccountHealthIntentCleanup = "cleanup"
	AccountHealthIntentAdd     = "add"
	AccountHealthIntentHelp    = "help"
	AccountHealthConfirmDelete = "DELETE"
	AccountHealthConfirmAdd    = "ADD"

	DefaultAccountHealthScanLimit = 100
	MaxAccountHealthScanLimit     = 200
	DefaultAccountHealthTestLimit = 20
	MaxAccountHealthTestLimit     = 50
	MaxAccountHealthCleanupIDs    = 50
	MaxAccountHealthAddItems      = 50
	accountHealthTestConcurrency  = 4
)

// AccountHealthAdmin is the account persistence surface used by the pool copilot.
type AccountHealthAdmin interface {
	ListAccounts(ctx context.Context, page, pageSize int, platform, accountType, status, search string, groupID int64, privacyMode string, sortBy, sortOrder string) ([]Account, int64, error)
	GetAccountsByIDs(ctx context.Context, ids []int64) ([]*Account, error)
	CreateAccount(ctx context.Context, input *CreateAccountInput) (*Account, error)
	DeleteAccount(ctx context.Context, id int64) error
}

// AccountHealthTester runs a non-SSE connectivity probe for one account.
type AccountHealthTester interface {
	RunTestBackground(ctx context.Context, accountID int64, modelID string) (*ScheduledTestResult, error)
}

type AccountHealthService struct {
	admin        AccountHealthAdmin
	tester       AccountHealthTester
	llm          AccountHealthLLM
	grokSSO      AccountHealthGrokSSOImporter
	settingsRepo SettingRepository
	operationMu  sync.Mutex
	operations   map[string]any
}

// SetSettingsRepository enables DB-backed lifecycle policy persistence.
func (s *AccountHealthService) SetSettingsRepository(repo SettingRepository) {
	if s != nil {
		s.settingsRepo = repo
	}
}

// SetGrokSSOImporter enables secure Grok SSO conversion for assistant imports.
func (s *AccountHealthService) SetGrokSSOImporter(importer AccountHealthGrokSSOImporter) {
	if s != nil {
		s.grokSSO = importer
	}
}

func NewAccountHealthService(admin AccountHealthAdmin, tester AccountHealthTester) *AccountHealthService {
	return &AccountHealthService{admin: admin, tester: tester, operations: make(map[string]any)}
}

func (s *AccountHealthService) cachedOperation(id string) any {
	if s == nil || strings.TrimSpace(id) == "" {
		return nil
	}
	s.operationMu.Lock()
	defer s.operationMu.Unlock()
	return s.operations[id]
}
func (s *AccountHealthService) rememberOperation(id string, result any) {
	if s == nil || strings.TrimSpace(id) == "" {
		return
	}
	s.operationMu.Lock()
	defer s.operationMu.Unlock()
	if s.operations == nil {
		s.operations = make(map[string]any)
	}
	s.operations[id] = result
}

func NewAccountHealthServiceFrom(admin AccountHealthAdmin, testSvc *AccountTestService) *AccountHealthService {
	var tester AccountHealthTester
	if testSvc != nil {
		tester = testSvc
	}
	svc := NewAccountHealthService(admin, tester)
	if admin != nil {
		svc.llm = newPoolAccountHealthLLM(admin, nil)
	}
	return svc
}

type AccountHealthScanRequest struct {
	Platform   string  `json:"platform"`
	AccountIDs []int64 `json:"account_ids"`
	Test       bool    `json:"test"`
	Limit      int     `json:"limit"`
}

type AccountHealthItem struct {
	AccountID  int64  `json:"account_id"`
	Name       string `json:"name"`
	Platform   string `json:"platform"`
	Type       string `json:"type"`
	Status     string `json:"status"`
	Reason     string `json:"reason"`
	Summary    string `json:"summary"`
	Cleanup    bool   `json:"cleanup"`
	Tested     bool   `json:"tested"`
	TestStatus string `json:"test_status,omitempty"`
}

type AccountHealthScanResult struct {
	Items        []AccountHealthItem `json:"items"`
	Total        int                 `json:"total"`
	Tested       int                 `json:"tested"`
	CleanupCount int                 `json:"cleanup_count"`
	Truncated    bool                `json:"truncated"`
}

type AccountHealthCleanupRequest struct {
	AccountIDs  []int64 `json:"account_ids"`
	Confirm     string  `json:"confirm"`
	OperationID string  `json:"operation_id,omitempty"`
}

type AccountHealthCleanupFailure struct {
	AccountID int64  `json:"account_id"`
	Message   string `json:"message"`
}

type AccountHealthCleanupResult struct {
	Deleted     []int64                       `json:"deleted"`
	Failed      []AccountHealthCleanupFailure `json:"failed"`
	OperationID string                        `json:"operation_id,omitempty"`
}

type AccountHealthChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AccountHealthProposal struct {
	AccountIDs []int64             `json:"account_ids"`
	Items      []AccountHealthItem `json:"items,omitempty"`
}

type AccountHealthAssistantRequest struct {
	Messages        []AccountHealthChatMessage `json:"messages"`
	Intent          string                     `json:"intent"`
	PendingProposal *AccountHealthProposal     `json:"pending_proposal"`
	AccountIDs      []int64                    `json:"account_ids"`
	Confirm         string                     `json:"confirm"`
	Platform        string                     `json:"platform"`
	Handoff         string                     `json:"handoff"`
}

type AccountHealthToolTrace struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Summary string `json:"summary"`
}

type AccountHealthAssistantResult struct {
	Intent      string                      `json:"intent"`
	Reply       string                      `json:"reply"`
	Scan        *AccountHealthScanResult    `json:"scan,omitempty"`
	Cleanup     *AccountHealthCleanupResult `json:"cleanup,omitempty"`
	Proposal    *AccountHealthProposal      `json:"proposal,omitempty"`
	AddProposal *AccountHealthAddProposal   `json:"add_proposal,omitempty"`
	Add         *AccountHealthAddResult     `json:"add,omitempty"`
	ToolTraces  []AccountHealthToolTrace    `json:"tool_traces,omitempty"`
}

func IsAccountHealthCleanupReason(reason string) bool {
	switch reason {
	case AccountHealthReasonAuthFailed, AccountHealthReasonExpired:
		return true
	default:
		return false
	}
}

func ClassifyAccountHealth(account *Account, testResult *ScheduledTestResult, tested bool) AccountHealthItem {
	item := AccountHealthItem{
		Reason:  AccountHealthReasonHealthy,
		Summary: accountHealthCannedSummary(AccountHealthReasonHealthy),
	}
	if account == nil {
		return item
	}
	item.AccountID = account.ID
	item.Name = strings.TrimSpace(account.Name)
	item.Platform = account.Platform
	item.Type = account.Type
	item.Status = account.Status
	item.Tested = tested
	if tested && testResult != nil {
		item.TestStatus = strings.TrimSpace(testResult.Status)
	}

	reason := classifyAccountHealthReason(account, testResult, tested)
	item.Reason = reason
	item.Summary = accountHealthCannedSummary(reason)
	item.Cleanup = IsAccountHealthCleanupReason(reason)
	return item
}

func classifyAccountHealthReason(account *Account, testResult *ScheduledTestResult, tested bool) string {
	if account == nil {
		return AccountHealthReasonHealthy
	}
	if account.Status == StatusDisabled {
		return AccountHealthReasonInactive
	}

	if tested {
		if testResult != nil && strings.EqualFold(strings.TrimSpace(testResult.Status), "success") {
			return AccountHealthReasonHealthy
		}
		if testResult != nil {
			if reason := classifyAccountHealthSignals(testResult.ErrorMessage); reason != "" {
				return reason
			}
		}
		if reason := classifyAccountHealthSignals(account.ErrorMessage); reason != "" {
			return reason
		}
		if accountIsExpiredForHealth(account) {
			return AccountHealthReasonExpired
		}
		return AccountHealthReasonErrorUnknown
	}

	if reason := classifyAccountHealthSignals(account.ErrorMessage); reason != "" {
		return reason
	}
	if accountIsExpiredForHealth(account) {
		return AccountHealthReasonExpired
	}
	if account.IsAPIKeyOrBedrock() && account.IsQuotaExceeded() {
		return AccountHealthReasonQuotaExhausted
	}
	if account.IsRateLimited() {
		return AccountHealthReasonRateLimited
	}
	if account.IsOverloaded() {
		return AccountHealthReasonOverloaded
	}
	if account.TempUnschedulableUntil != nil && time.Now().Before(*account.TempUnschedulableUntil) {
		return AccountHealthReasonTempUnschedulable
	}
	if account.Status == StatusError {
		return AccountHealthReasonErrorUnknown
	}
	return AccountHealthReasonHealthy
}

func accountIsExpiredForHealth(account *Account) bool {
	if account == nil || !account.AutoPauseOnExpired || account.ExpiresAt == nil {
		return false
	}
	return !time.Now().Before(*account.ExpiresAt)
}

func classifyAccountHealthSignals(raw string) string {
	normalized := normalizeAccountHealthSignal(raw)
	if normalized == "" {
		return ""
	}

	authSignals := []string{
		"invalid_grant",
		"token_expired",
		"expired_token",
		"invalid_token",
		"invalid_api_key",
		"invalid_apikey",
		"invalid_authentication",
		"authentication_error",
		"unauthenticated",
		"unauthorized",
		"account_deactivated",
		"account_is_deactivated",
		"deactivated_account",
		"deactivated_workspace",
		"account_suspended",
		"account_disabled",
		"workspace_deactivated",
		"workspace_suspended",
		"revoked",
		"authentication_failed",
		"access_forbidden",
	}
	for _, signal := range authSignals {
		if strings.Contains(normalized, signal) {
			return AccountHealthReasonAuthFailed
		}
	}
	if strings.Contains(normalized, "401") {
		return AccountHealthReasonAuthFailed
	}

	quotaSignals := []string{
		"quota_exhausted",
		"insufficient_quota",
		"quota_exceeded",
		"credit_exhausted",
		"credits_exhausted",
	}
	for _, signal := range quotaSignals {
		if strings.Contains(normalized, signal) {
			return AccountHealthReasonQuotaExhausted
		}
	}

	rateSignals := []string{
		"rate_limit",
		"ratelimit",
		"too_many_requests",
		"429",
	}
	for _, signal := range rateSignals {
		if strings.Contains(normalized, signal) {
			return AccountHealthReasonRateLimited
		}
	}

	overloadSignals := []string{
		"overloaded",
		"overload",
		"529",
		"capacity",
	}
	for _, signal := range overloadSignals {
		if strings.Contains(normalized, signal) {
			return AccountHealthReasonOverloaded
		}
	}
	return ""
}

func normalizeAccountHealthSignal(raw string) string {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if trimmed == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(trimmed))
	prevUnderscore := false
	for _, r := range trimmed {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			prevUnderscore = false
		default:
			if !prevUnderscore {
				b.WriteByte('_')
				prevUnderscore = true
			}
		}
	}
	return strings.Trim(b.String(), "_")
}

func accountHealthCannedSummary(reason string) string {
	switch reason {
	case AccountHealthReasonAuthFailed:
		return "Authentication failed or token is unusable"
	case AccountHealthReasonExpired:
		return "Account expiry time has passed"
	case AccountHealthReasonRateLimited:
		return "Temporarily rate limited"
	case AccountHealthReasonQuotaExhausted:
		return "Quota is exhausted"
	case AccountHealthReasonOverloaded:
		return "Upstream is overloaded"
	case AccountHealthReasonTempUnschedulable:
		return "Temporarily unschedulable"
	case AccountHealthReasonErrorUnknown:
		return "Account is in error state"
	case AccountHealthReasonInactive:
		return "Account is disabled"
	default:
		return "Account looks healthy"
	}
}

func (s *AccountHealthService) Scan(ctx context.Context, req AccountHealthScanRequest) (*AccountHealthScanResult, error) {
	if s == nil || s.admin == nil {
		return nil, infraerrors.InternalServer("ACCOUNT_HEALTH_UNAVAILABLE", "account health service unavailable")
	}

	limit := req.Limit
	if limit <= 0 {
		limit = DefaultAccountHealthScanLimit
	}
	if limit > MaxAccountHealthScanLimit {
		limit = MaxAccountHealthScanLimit
	}

	platform := strings.TrimSpace(req.Platform)
	ids := uniquePositiveIDs(req.AccountIDs, MaxAccountHealthScanLimit)

	var (
		accounts  []Account
		truncated bool
	)
	if len(ids) > 0 {
		fetched, err := s.admin.GetAccountsByIDs(ctx, ids)
		if err != nil {
			return nil, err
		}
		byID := make(map[int64]*Account, len(fetched))
		for _, acc := range fetched {
			if acc == nil {
				continue
			}
			byID[acc.ID] = acc
		}
		for _, id := range ids {
			acc, ok := byID[id]
			if !ok {
				continue
			}
			if platform != "" && acc.Platform != platform {
				continue
			}
			accounts = append(accounts, *acc)
			if len(accounts) >= limit {
				truncated = len(ids) > limit
				break
			}
		}
		if !truncated && len(ids) > len(accounts) && len(accounts) >= limit {
			truncated = true
		}
	} else if req.Test {
		// A live test is explicitly a pool-wide health check. Restricting the
		// query to status=error/expired would make it impossible to discover an
		// active account whose token has just become unusable but has not yet
		// been marked errored by a request.
		allAccounts, total, err := s.admin.ListAccounts(ctx, 1, limit, platform, "", "", "", 0, "", "id", "asc")
		if err != nil {
			return nil, err
		}
		accounts = allAccounts
		truncated = total > int64(len(allAccounts))
	} else {
		errorAccounts, errorTotal, err := s.admin.ListAccounts(ctx, 1, limit, platform, "", StatusError, "", 0, "", "id", "asc")
		if err != nil {
			return nil, err
		}
		expiredAccounts, expiredTotal, err := s.admin.ListAccounts(ctx, 1, limit, platform, "", AccountListStatusExpired, "", 0, "", "id", "asc")
		if err != nil {
			return nil, err
		}
		merged := mergeAccountHealthCandidates(errorAccounts, expiredAccounts, limit)
		accounts = merged
		truncated = errorTotal > int64(len(errorAccounts)) || expiredTotal > int64(len(expiredAccounts)) || int64(len(errorAccounts)+len(expiredAccounts)) > int64(len(merged))
	}

	items := make([]AccountHealthItem, len(accounts))
	for i := range accounts {
		items[i] = ClassifyAccountHealth(&accounts[i], nil, false)
	}

	testedCount := 0
	if req.Test && s.tester != nil && len(accounts) > 0 {
		testLimit := DefaultAccountHealthTestLimit
		if testLimit > len(accounts) {
			testLimit = len(accounts)
		}
		if testLimit > MaxAccountHealthTestLimit {
			testLimit = MaxAccountHealthTestLimit
		}
		results := s.runHealthTests(ctx, accounts[:testLimit])
		for i := 0; i < testLimit; i++ {
			result := results[accounts[i].ID]
			items[i] = ClassifyAccountHealth(&accounts[i], result, true)
			testedCount++
		}
	}

	cleanupCount := 0
	settings, _ := s.GetSettings(ctx)
	for _, item := range items {
		if item.Cleanup {
			cleanupCount++
			// Quarantine is a safe reversible state change. Keep this optional
			// so focused test doubles and legacy integrations remain compatible.
			if settings != nil && settings.AutoQuarantine {
				if updater, ok := s.admin.(interface {
					UpdateAccount(context.Context, int64, *UpdateAccountInput) (*Account, error)
				}); ok {
					_, _ = updater.UpdateAccount(ctx, item.AccountID, &UpdateAccountInput{Status: StatusDisabled})
				}
			}
		}
	}

	return &AccountHealthScanResult{
		Items:        items,
		Total:        len(items),
		Tested:       testedCount,
		CleanupCount: cleanupCount,
		Truncated:    truncated,
	}, nil
}

func mergeAccountHealthCandidates(errorAccounts, expiredAccounts []Account, limit int) []Account {
	seen := make(map[int64]struct{}, len(errorAccounts)+len(expiredAccounts))
	out := make([]Account, 0, min(limit, len(errorAccounts)+len(expiredAccounts)))
	appendUnique := func(list []Account) {
		for _, acc := range list {
			if _, ok := seen[acc.ID]; ok {
				continue
			}
			seen[acc.ID] = struct{}{}
			out = append(out, acc)
			if len(out) >= limit {
				return
			}
		}
	}
	appendUnique(errorAccounts)
	if len(out) < limit {
		appendUnique(expiredAccounts)
	}
	return out
}

func (s *AccountHealthService) runHealthTests(ctx context.Context, accounts []Account) map[int64]*ScheduledTestResult {
	results := make(map[int64]*ScheduledTestResult, len(accounts))
	if s.tester == nil || len(accounts) == 0 {
		return results
	}

	sem := make(chan struct{}, accountHealthTestConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	for i := range accounts {
		accountID := accounts[i].ID
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()
			result, err := s.tester.RunTestBackground(ctx, id, "")
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				results[id] = &ScheduledTestResult{
					Status:       "failed",
					ErrorMessage: "test failed",
				}
				return
			}
			if result == nil {
				results[id] = &ScheduledTestResult{Status: "failed", ErrorMessage: "test failed"}
				return
			}
			// Drop raw upstream text so later JSON never echoes credentials.
			results[id] = &ScheduledTestResult{
				Status:       result.Status,
				ErrorMessage: result.ErrorMessage,
				LatencyMs:    result.LatencyMs,
			}
		}(accountID)
	}
	wg.Wait()
	return results
}

func (s *AccountHealthService) Cleanup(ctx context.Context, req AccountHealthCleanupRequest) (*AccountHealthCleanupResult, error) {
	if s == nil || s.admin == nil {
		return nil, infraerrors.InternalServer("ACCOUNT_HEALTH_UNAVAILABLE", "account health service unavailable")
	}
	if strings.TrimSpace(req.Confirm) != AccountHealthConfirmDelete {
		return nil, infraerrors.BadRequest("ACCOUNT_HEALTH_CONFIRM_REQUIRED", "confirm must be DELETE")
	}
	if cached, ok := s.cachedOperation(req.OperationID).(*AccountHealthCleanupResult); ok && cached != nil {
		return cached, nil
	}
	ids := uniquePositiveIDs(req.AccountIDs, MaxAccountHealthCleanupIDs+1)
	if len(ids) == 0 {
		return nil, infraerrors.BadRequest("ACCOUNT_HEALTH_IDS_REQUIRED", "account_ids is required")
	}
	if len(ids) > MaxAccountHealthCleanupIDs {
		return nil, infraerrors.BadRequest("ACCOUNT_HEALTH_IDS_LIMIT", "account_ids exceeds the maximum of 50")
	}

	result := &AccountHealthCleanupResult{
		Deleted: make([]int64, 0, len(ids)),
		Failed:  make([]AccountHealthCleanupFailure, 0),
	}
	result.OperationID = strings.TrimSpace(req.OperationID)
	for _, id := range ids {
		if err := s.admin.DeleteAccount(ctx, id); err != nil {
			result.Failed = append(result.Failed, AccountHealthCleanupFailure{
				AccountID: id,
				Message:   "delete failed",
			})
			continue
		}
		result.Deleted = append(result.Deleted, id)
	}
	s.rememberOperation(req.OperationID, result)
	return result, nil
}

func (s *AccountHealthService) Chat(ctx context.Context, req AccountHealthAssistantRequest) (*AccountHealthAssistantResult, error) {
	if s == nil || s.admin == nil {
		return nil, infraerrors.InternalServer("ACCOUNT_HEALTH_UNAVAILABLE", "account health service unavailable")
	}

	last := lastAccountHealthUserMessage(req.Messages)
	chinese := containsCJK(last) || containsCJK(req.Intent) || containsCJK(req.Handoff)

	handoff := strings.TrimSpace(req.Handoff)
	if handoff == "" && looksLikeAccountHealthHandoff(last) {
		handoff = last
	}
	if handoff != "" {
		proposal := ingestAccountHealthHandoff(handoff, req.Platform)
		return &AccountHealthAssistantResult{
			Intent:      AccountHealthIntentAdd,
			Reply:       accountHealthAddPreviewReply(len(proposal.Items), chinese),
			AddProposal: proposal,
		}, nil
	}
	// The UI's primary "scan" action requests a live check explicitly. Honor
	// that intent before the LLM so it cannot downgrade the operation to a
	// status-only lookup based on the wording of the chat message.
	if normalizeAccountHealthIntent(req.Intent) == AccountHealthIntentTest {
		scan, err := s.Scan(ctx, AccountHealthScanRequest{
			Platform:   req.Platform,
			AccountIDs: req.AccountIDs,
			Test:       true,
		})
		if err != nil {
			return nil, err
		}
		return &AccountHealthAssistantResult{
			Intent:   AccountHealthIntentTest,
			Reply:    accountHealthScanReply(scan, true, chinese),
			Scan:     scan,
			Proposal: proposalFromScan(scan),
		}, nil
	}

	if s.llm == nil {
		// Scan/test shortcuts remain useful even when no model account is
		// configured. Natural-language cleanup/help still requires an LLM, but
		// deterministic health checks can run directly against the admin API.
		intent := parseAccountHealthIntent(req.Intent, last)
		if intent == AccountHealthIntentScan || intent == AccountHealthIntentTest {
			scan, err := s.Scan(ctx, AccountHealthScanRequest{
				Platform:   req.Platform,
				AccountIDs: req.AccountIDs,
				Test:       intent == AccountHealthIntentTest,
			})
			if err != nil {
				return nil, err
			}
			return &AccountHealthAssistantResult{
				Intent:   intent,
				Reply:    accountHealthScanReply(scan, intent == AccountHealthIntentTest, chinese),
				Scan:     scan,
				Proposal: proposalFromScan(scan),
			}, nil
		}
		return &AccountHealthAssistantResult{
			Intent: AccountHealthIntentHelp,
			Reply:  accountHealthNeedWorkingKeyReply(chinese),
		}, nil
	}
	return s.chatWithLLM(ctx, req, chinese)
}

func lastAccountHealthUserMessage(messages []AccountHealthChatMessage) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if strings.EqualFold(strings.TrimSpace(messages[i].Role), "user") {
			return strings.TrimSpace(messages[i].Content)
		}
	}
	if len(messages) == 0 {
		return ""
	}
	return strings.TrimSpace(messages[len(messages)-1].Content)
}

func parseAccountHealthIntent(explicit, message string) string {
	if intent := normalizeAccountHealthIntent(explicit); intent != "" {
		return intent
	}
	n := strings.ToLower(strings.TrimSpace(message))
	if n == "" {
		return AccountHealthIntentHelp
	}
	switch {
	case strings.Contains(n, "确认删除") || strings.Contains(n, "确认清理") || strings.Contains(n, "confirm delete"):
		return AccountHealthIntentCleanup
	case strings.Contains(n, "测试") || strings.Contains(n, "筛选") || strings.Contains(n, "test"):
		return AccountHealthIntentTest
	case strings.Contains(n, "扫描") || strings.Contains(n, "检查") || strings.Contains(n, "scan"):
		return AccountHealthIntentScan
	case strings.Contains(n, "删除") || strings.Contains(n, "清理") || strings.Contains(n, "cleanup") || strings.Contains(n, "delete"):
		return AccountHealthIntentCleanup
	case strings.Contains(n, "帮助") || strings.Contains(n, "怎么用") || strings.Contains(n, "help"):
		return AccountHealthIntentHelp
	default:
		return AccountHealthIntentHelp
	}
}

func normalizeAccountHealthIntent(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case AccountHealthIntentScan:
		return AccountHealthIntentScan
	case AccountHealthIntentTest:
		return AccountHealthIntentTest
	case AccountHealthIntentCleanup, "delete":
		return AccountHealthIntentCleanup
	case AccountHealthIntentHelp:
		return AccountHealthIntentHelp
	default:
		return ""
	}
}

func isAccountHealthDeleteConfirm(confirm, message string) bool {
	if strings.TrimSpace(confirm) == AccountHealthConfirmDelete {
		return true
	}
	n := strings.ToLower(strings.TrimSpace(message))
	if n == "" {
		return false
	}
	if n == "确认删除" || n == "确认清理" || n == "confirm delete" || n == "confirm cleanup" {
		return true
	}
	return false
}

func pendingProposalIDs(proposal *AccountHealthProposal) []int64 {
	if proposal == nil {
		return nil
	}
	ids := uniquePositiveIDs(proposal.AccountIDs, MaxAccountHealthCleanupIDs)
	if len(ids) > 0 {
		return ids
	}
	collected := make([]int64, 0, len(proposal.Items))
	for _, item := range proposal.Items {
		collected = append(collected, item.AccountID)
	}
	return uniquePositiveIDs(collected, MaxAccountHealthCleanupIDs)
}

func proposalFromScan(scan *AccountHealthScanResult) *AccountHealthProposal {
	if scan == nil {
		return &AccountHealthProposal{AccountIDs: []int64{}, Items: []AccountHealthItem{}}
	}
	items := make([]AccountHealthItem, 0, scan.CleanupCount)
	ids := make([]int64, 0, scan.CleanupCount)
	for _, item := range scan.Items {
		if !item.Cleanup {
			continue
		}
		items = append(items, item)
		ids = append(ids, item.AccountID)
	}
	return &AccountHealthProposal{AccountIDs: ids, Items: items}
}

func cloneHealthItems(proposal *AccountHealthProposal) []AccountHealthItem {
	if proposal == nil || len(proposal.Items) == 0 {
		return nil
	}
	out := make([]AccountHealthItem, len(proposal.Items))
	copy(out, proposal.Items)
	return out
}

func uniquePositiveIDs(ids []int64, max int) []int64 {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
		if max > 0 && len(out) >= max {
			break
		}
	}
	return out
}

func uniqueNonNegativeInts(values []int, max int) []int {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[int]struct{}, len(values))
	out := make([]int, 0, len(values))
	for _, value := range values {
		if value < 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
		if max > 0 && len(out) >= max {
			break
		}
	}
	return out
}

func containsCJK(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

func accountHealthHelpReply(chinese bool) string {
	if chinese {
		return "我可以扫描账号池里过期或鉴权失败的账号，筛出来交给你在右侧确认后再删除；也可以把 API Key 交给本服务添加，凭证不会发给模型。添加和删除都要在右侧勾选确认，我不会自动改账号池。"
	}
	return "I can scan the account pool for expired or unusable credentials and put them on the right for you to confirm before delete. You can also hand API keys to this service to add accounts; credentials are not sent to the model. Add and delete both require confirmation in the review panel."
}

func accountHealthNeedWorkingKeyReply(chinese bool) string {
	if chinese {
		return "需要至少一个可用的 OpenAI 兼容 API Key 账号，助手才能工作。请先在账号池里放一个可调度的 API Key 账号。"
	}
	return "The assistant needs at least one working OpenAI-compatible API Key account in the pool before it can run."
}

func accountHealthLLMFailedReply(chinese bool) string {
	if chinese {
		return "助手暂时无法完成这次请求。请稍后重试，或先用快捷指令扫描、测试。"
	}
	return "The assistant could not complete this request. Please retry, or use the scan/test shortcuts."
}

func accountHealthScanReply(scan *AccountHealthScanResult, tested, chinese bool) string {
	if scan == nil || scan.CleanupCount == 0 {
		if chinese {
			if tested {
				return "测试完成，没有发现需要清理的过期或鉴权失败账号。"
			}
			return "扫描完成，没有发现需要清理的过期或鉴权失败账号。"
		}
		if tested {
			return "Testing finished. No expired or unusable accounts need cleanup."
		}
		return "Scan finished. No expired or unusable accounts need cleanup."
	}
	if chinese {
		if tested {
			return "已测试并筛选出待确认账号。请勾选后确认删除，我不会自动删除。"
		}
		return "已扫描出可能过期或不可用的账号。请勾选后确认删除，我不会自动删除。"
	}
	if tested {
		return "Finished testing and filtered accounts that look unusable. Select them and confirm delete; nothing is deleted automatically."
	}
	return "Found accounts that may be expired or unusable. Select them and confirm delete; nothing is deleted automatically."
}

func accountHealthNeedScanBeforeDeleteReply(chinese bool) string {
	if chinese {
		return "还没有待确认的账号列表。请先扫描或测试，再从筛选结果里确认删除。我不会从聊天文本里解析要删除的账号 ID。"
	}
	return "There is no pending review list yet. Scan or test first, then confirm delete from that filtered list. I will not parse account IDs from chat text."
}

func accountHealthNeedConfirmReply(count int, chinese bool) string {
	if chinese {
		return "这些账号只会在你明确确认后删除。请在弹窗中确认，或回复「确认删除」。"
	}
	if count == 1 {
		return "This account will be deleted only after you explicitly confirm. Confirm in the dialog, or reply “confirm delete”."
	}
	return "These accounts will be deleted only after you explicitly confirm. Confirm in the dialog, or reply “confirm delete”."
}

func accountHealthDeletedReply(deleted, failed int, chinese bool) string {
	if chinese {
		if failed > 0 {
			return "已按你的确认删除部分账号，其余删除失败，请稍后重试。"
		}
		if deleted == 0 {
			return "没有账号被删除。"
		}
		return "已按你的确认删除所选账号。"
	}
	if failed > 0 {
		return "Some selected accounts were deleted; others failed. Please retry the remaining ones."
	}
	if deleted == 0 {
		return "No accounts were deleted."
	}
	return "Deleted the selected accounts after your confirmation."
}
