package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"
	"golang.org/x/sync/singleflight"
)

const (
	deepSeekBalancePath         = "/user/balance"
	deepSeekBalanceTimeout      = 15 * time.Second
	deepSeekBalanceMaxBody      = 256 * 1024
	deepSeekBalanceExtraKey     = "deepseek_balance"
	deepSeekBalanceCurrencyKey  = "deepseek_balance_currency"
	deepSeekBalanceAvailableKey = "deepseek_balance_available"
	deepSeekBalanceUpdatedAtKey = "deepseek_balance_updated_at"
	deepSeekBalanceEntriesKey   = "deepseek_balances"
	deepSeekBalanceLowKey       = "deepseek_balance_low"
)

// DeepSeekBalanceEntry is one currency returned by DeepSeek's /user/balance
// endpoint.  DeepSeek normally returns both CNY and USD entries.
type DeepSeekBalanceEntry struct {
	Currency string  `json:"currency"`
	Balance  float64 `json:"balance"`
}

// DeepSeekBalanceResult is the sanitized result exposed to the admin API.
// A failed upstream response is represented by Success=false and Error while
// the Go error is reserved for local validation/transport failures.
type DeepSeekBalanceResult struct {
	Provider   string                 `json:"provider"`
	Success    bool                   `json:"success"`
	Balance    float64                `json:"balance"`
	Currency   string                 `json:"currency,omitempty"`
	Balances   []DeepSeekBalanceEntry `json:"balances,omitempty"`
	Available  bool                   `json:"available"`
	StatusCode int                    `json:"status_code,omitempty"`
	FetchedAt  int64                  `json:"fetched_at"`
	Persisted  bool                   `json:"persisted"`
	Error      string                 `json:"error,omitempty"`
}

// DeepSeekBalanceService queries the pay-as-you-go DeepSeek balance endpoint.
// The probe is shared by the manual admin endpoint and the optional periodic
// checker; scheduling/automatic account pausing remains in the separate policy
// service and is still opt-in.
type DeepSeekBalanceService struct {
	accountRepo  AccountRepository
	proxyRepo    ProxyRepository
	httpUpstream HTTPUpstream
	cfg          *config.Config
	flight       singleflight.Group
}

// NewDeepSeekBalanceService constructs the balance probe service.
func NewDeepSeekBalanceService(
	accountRepo AccountRepository,
	proxyRepo ProxyRepository,
	httpUpstream HTTPUpstream,
	cfg *config.Config,
) *DeepSeekBalanceService {
	return &DeepSeekBalanceService{
		accountRepo:  accountRepo,
		proxyRepo:    proxyRepo,
		httpUpstream: httpUpstream,
		cfg:          cfg,
	}
}

// ProvideDeepSeekBalanceService is kept as a small Wire provider so the probe
// can be enabled without coupling account construction to HTTP details.
func ProvideDeepSeekBalanceService(
	accountRepo AccountRepository,
	proxyRepo ProxyRepository,
	httpUpstream HTTPUpstream,
	cfg *config.Config,
) *DeepSeekBalanceService {
	return NewDeepSeekBalanceService(accountRepo, proxyRepo, httpUpstream, cfg)
}

// QueryBalance loads an API-key DeepSeek account, probes /user/balance, and
// persists a sanitized snapshot in account.extra on valid success.
func (s *DeepSeekBalanceService) QueryBalance(ctx context.Context, accountID int64) (*DeepSeekBalanceResult, error) {
	if s == nil || s.accountRepo == nil || s.httpUpstream == nil {
		return nil, infraerrors.New(http.StatusInternalServerError, "DEEPSEEK_BALANCE_NOT_CONFIGURED", "DeepSeek balance service is not configured")
	}
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil || account == nil {
		if err != nil {
			return nil, infraerrors.Newf(http.StatusNotFound, "DEEPSEEK_BALANCE_ACCOUNT_NOT_FOUND", "account not found: %v", err)
		}
		return nil, infraerrors.New(http.StatusNotFound, "DEEPSEEK_BALANCE_ACCOUNT_NOT_FOUND", "account not found")
	}
	return s.QueryBalanceForAccount(ctx, account)
}

// QueryBalanceForAccount probes an already-loaded account.  Calls for the same
// account are coalesced so an admin refresh cannot fan out duplicate requests.
func (s *DeepSeekBalanceService) QueryBalanceForAccount(ctx context.Context, account *Account) (*DeepSeekBalanceResult, error) {
	if s == nil || s.accountRepo == nil || s.httpUpstream == nil {
		return nil, infraerrors.New(http.StatusInternalServerError, "DEEPSEEK_BALANCE_NOT_CONFIGURED", "DeepSeek balance service is not configured")
	}
	if err := validateDeepSeekBalanceAccount(account); err != nil {
		return nil, err
	}
	key := "deepseek_balance:" + strconv.FormatInt(account.ID, 10)
	resultCh := s.flight.DoChan(key, func() (any, error) {
		probeCtx, cancel := context.WithTimeout(context.Background(), deepSeekBalanceTimeout+5*time.Second)
		defer cancel()
		return s.queryBalanceForAccount(probeCtx, account)
	})
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case flightResult := <-resultCh:
		if flightResult.Err != nil {
			return nil, flightResult.Err
		}
		result, ok := flightResult.Val.(*DeepSeekBalanceResult)
		if !ok || result == nil {
			return nil, infraerrors.New(http.StatusInternalServerError, "DEEPSEEK_BALANCE_RESULT_INVALID", "invalid DeepSeek balance probe result")
		}
		clone := *result
		clone.Balances = append([]DeepSeekBalanceEntry(nil), result.Balances...)
		return &clone, nil
	}
}

func validateDeepSeekBalanceAccount(account *Account) error {
	if account == nil {
		return infraerrors.New(http.StatusNotFound, "DEEPSEEK_BALANCE_ACCOUNT_NOT_FOUND", "account not found")
	}
	if !account.IsDeepseek() {
		return infraerrors.New(http.StatusBadRequest, "DEEPSEEK_BALANCE_INVALID_PLATFORM", "account is not a DeepSeek account")
	}
	if account.Type != AccountTypeAPIKey {
		return infraerrors.New(http.StatusBadRequest, "DEEPSEEK_BALANCE_INVALID_TYPE", "DeepSeek balance requires an API-key account")
	}
	if account.GetAccountMode() == AccountModeCoding {
		return infraerrors.New(http.StatusBadRequest, "DEEPSEEK_BALANCE_CODING_PLAN", "DeepSeek coding-plan account has no balance endpoint")
	}
	if strings.TrimSpace(account.GetCNAPIKey()) == "" {
		return infraerrors.New(http.StatusBadRequest, "DEEPSEEK_BALANCE_NO_APIKEY", "DeepSeek api_key is empty")
	}
	return nil
}

func (s *DeepSeekBalanceService) queryBalanceForAccount(ctx context.Context, account *Account) (*DeepSeekBalanceResult, error) {
	baseURL := strings.TrimSpace(account.GetOpenAIFormatBaseURL())
	if baseURL == "" {
		baseURL = DefaultDeepseekBaseURL
	}
	targetURL, err := buildDeepSeekBalanceURL(baseURL, s.cfg)
	if err != nil {
		return nil, infraerrors.Newf(http.StatusForbidden, "DEEPSEEK_BALANCE_URL_REJECTED", "%v", err)
	}
	proxyURL := s.resolveProxyURL(ctx, account)
	callCtx, cancel := context.WithTimeout(ctx, deepSeekBalanceTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(callCtx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, infraerrors.Newf(http.StatusInternalServerError, "DEEPSEEK_BALANCE_REQUEST_BUILD_FAILED", "build request: %v", err)
	}
	req = req.WithContext(WithHTTPUpstreamProfile(req.Context(), HTTPUpstreamProfileOpenAI))
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(account.GetCNAPIKey()))
	req.Header.Set("Accept", "application/json")
	account.ApplyHeaderOverrides(req.Header)
	resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, maxInt(account.Concurrency, 1))
	if err != nil {
		return nil, infraerrors.Newf(http.StatusBadGateway, "DEEPSEEK_BALANCE_REQUEST_FAILED", "upstream request failed: %v", err)
	}
	if resp == nil || resp.Body == nil {
		return nil, infraerrors.New(http.StatusBadGateway, "DEEPSEEK_BALANCE_EMPTY_RESPONSE", "upstream returned an empty response")
	}
	defer func() { _ = resp.Body.Close() }()
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, deepSeekBalanceMaxBody+1))
	if readErr != nil {
		return nil, infraerrors.Newf(http.StatusBadGateway, "DEEPSEEK_BALANCE_RESPONSE_READ_FAILED", "read upstream response: %v", readErr)
	}
	if len(body) > deepSeekBalanceMaxBody {
		return nil, infraerrors.New(http.StatusBadGateway, "DEEPSEEK_BALANCE_RESPONSE_TOO_LARGE", "upstream balance response is too large")
	}

	now := time.Now().UTC()
	result := &DeepSeekBalanceResult{
		Provider:   PlatformDeepseek,
		StatusCode: resp.StatusCode,
		Available:  true,
		FetchedAt:  now.Unix(),
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		result.Error = fmt.Sprintf("DeepSeek balance API returned HTTP %d", resp.StatusCode)
		return result, nil
	}
	entries, available, parseErr := parseDeepSeekBalanceResponse(body)
	if parseErr != nil {
		result.Error = parseErr.Error()
		return result, nil
	}
	result.Balances = entries
	result.Balance = entries[0].Balance
	result.Currency = entries[0].Currency
	result.Available = available
	result.Success = true

	updates := map[string]any{
		deepSeekBalanceExtraKey:     result.Balance,
		deepSeekBalanceCurrencyKey:  result.Currency,
		deepSeekBalanceAvailableKey: result.Available,
		deepSeekBalanceUpdatedAtKey: now.Format(time.RFC3339),
		deepSeekBalanceEntriesKey:   deepSeekBalanceEntriesForExtra(entries),
		// A successful probe supersedes a stale reactive low-balance marker.
		deepSeekBalanceLowKey: false,
	}
	if err := s.accountRepo.UpdateExtra(ctx, account.ID, updates); err == nil {
		result.Persisted = true
	}
	return result, nil
}

func (s *DeepSeekBalanceService) resolveProxyURL(ctx context.Context, account *Account) string {
	if account == nil || account.ProxyID == nil {
		return ""
	}
	if account.Proxy != nil {
		return account.Proxy.URL()
	}
	if s != nil && s.proxyRepo != nil {
		if proxy, err := s.proxyRepo.GetByID(ctx, *account.ProxyID); err == nil && proxy != nil {
			account.Proxy = proxy
			return proxy.URL()
		}
	}
	return ""
}

func buildDeepSeekBalanceURL(base string, cfg *config.Config) (string, error) {
	base = strings.TrimSpace(base)
	if base == "" {
		base = DefaultDeepseekBaseURL
	}
	validated, err := validateDeepSeekProbeURL(cfg, base)
	if err != nil {
		return "", err
	}
	// Keep this helper safe for direct callers as well as account-backed probes:
	// normalize only the official Anthropic facade path, while preserving custom
	// relay prefixes.
	validated = normalizeDeepseekOpenAIFormatBaseURL(validated)
	parsed, err := url.Parse(validated)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid DeepSeek base URL")
	}
	path := strings.TrimRight(parsed.Path, "/")
	lowerPath := strings.ToLower(path)
	if !strings.HasSuffix(lowerPath, deepSeekBalancePath) {
		path += deepSeekBalancePath
	}
	parsed.Path = path
	parsed.RawPath = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func validateDeepSeekProbeURL(cfg *config.Config, raw string) (string, error) {
	if cfg == nil || !cfg.Security.URLAllowlist.Enabled {
		allowHTTP := cfg != nil && cfg.Security.URLAllowlist.AllowInsecureHTTP
		validated, err := urlvalidator.ValidateURLFormat(raw, allowHTTP)
		if err != nil {
			return "", fmt.Errorf("base URL rejected by URL security policy: %w", err)
		}
		return validated, nil
	}
	validated, err := urlvalidator.ValidateHTTPSURL(raw, urlvalidator.ValidationOptions{
		AllowedHosts:     cfg.Security.URLAllowlist.UpstreamHosts,
		RequireAllowlist: true,
		AllowPrivate:     cfg.Security.URLAllowlist.AllowPrivateHosts,
	})
	if err != nil {
		return "", fmt.Errorf("base URL rejected by URL security policy: %w", err)
	}
	return validated, nil
}

type deepSeekBalanceResponse struct {
	IsAvailable  *bool                 `json:"is_available"`
	BalanceInfos []deepSeekBalanceInfo `json:"balance_infos"`
}

type deepSeekBalanceInfo struct {
	Currency     string          `json:"currency"`
	TotalBalance json.RawMessage `json:"total_balance"`
}

func parseDeepSeekBalanceResponse(body []byte) ([]DeepSeekBalanceEntry, bool, error) {
	var payload deepSeekBalanceResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, true, fmt.Errorf("invalid balance response: %v", err)
	}
	if payload.BalanceInfos == nil {
		return nil, true, fmt.Errorf("invalid balance response: missing balance_infos")
	}
	entries := make([]DeepSeekBalanceEntry, 0, len(payload.BalanceInfos))
	for _, info := range payload.BalanceInfos {
		balance, ok := parseDeepSeekBalanceNumber(info.TotalBalance)
		if !ok {
			continue
		}
		currency := strings.ToUpper(strings.TrimSpace(info.Currency))
		if currency == "" {
			currency = "CNY"
		}
		entries = append(entries, DeepSeekBalanceEntry{Currency: currency, Balance: balance})
	}
	if len(entries) == 0 {
		return nil, true, fmt.Errorf("invalid balance response: no valid balance entries")
	}
	available := true
	if payload.IsAvailable != nil {
		available = *payload.IsAvailable
	}
	return entries, available, nil
}

func parseDeepSeekBalanceNumber(raw json.RawMessage) (float64, bool) {
	value := strings.TrimSpace(string(raw))
	if value == "" || value == "null" {
		return 0, false
	}
	if strings.HasPrefix(value, "\"") {
		var text string
		if err := json.Unmarshal(raw, &text); err != nil {
			return 0, false
		}
		value = strings.TrimSpace(text)
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
		return 0, false
	}
	return parsed, true
}

func deepSeekBalanceEntriesForExtra(entries []DeepSeekBalanceEntry) []map[string]any {
	result := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		result = append(result, map[string]any{"currency": entry.Currency, "balance": entry.Balance})
	}
	return result
}
