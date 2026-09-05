package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/tidwall/gjson"
	"golang.org/x/sync/singleflight"
)

const (
	cnQuotaUpstreamTimeout = 15 * time.Second
	cnQuotaMaxBodyBytes    = 256 * 1024

	cnExtraSuffix5hUsed       = "5h_used_percent"
	cnExtraSuffix5hReset      = "5h_reset_at"
	cnExtraSuffixWeeklyUsed   = "weekly_used_percent"
	cnExtraSuffixWeeklyReset  = "weekly_reset_at"
	cnExtraSuffixUsageUpdated = "usage_updated_at"
)

func cnExtraKey(provider, suffix string) string { return provider + "_" + suffix }

// CNQuotaTier is one rolling usage window returned by a Coding Plan probe.
type CNQuotaTier struct {
	Window      string  `json:"window"`
	UsedPercent float64 `json:"used_percent"`
	ResetAt     string  `json:"reset_at,omitempty"`
}

// CNProviderQuotaProbeResult is the sanitized admin/UI response.
type CNProviderQuotaProbeResult struct {
	Provider        string        `json:"provider"`
	Source          string        `json:"source"`
	Success         bool          `json:"success"`
	CredentialValid bool          `json:"credential_valid"`
	Tiers           []CNQuotaTier `json:"tiers,omitempty"`
	PlanLevel       string        `json:"plan_level,omitempty"`
	StatusCode      int           `json:"status_code,omitempty"`
	FetchedAt       int64         `json:"fetched_at"`
	Persisted       bool          `json:"persisted"`
	Error           string        `json:"error,omitempty"`
}

// CNProviderQuotaService queries Zhipu GLM Coding Plan rolling windows.
// The service is intentionally provider-oriented so additional Coding Plan
// endpoints can be added without changing the admin/API contract.
type CNProviderQuotaService struct {
	accountRepo  AccountRepository
	proxyRepo    ProxyRepository
	httpUpstream HTTPUpstream
	cfg          *config.Config
	flight       singleflight.Group
}

func NewCNProviderQuotaService(accountRepo AccountRepository, proxyRepo ProxyRepository, httpUpstream HTTPUpstream, cfg *config.Config) *CNProviderQuotaService {
	return &CNProviderQuotaService{accountRepo: accountRepo, proxyRepo: proxyRepo, httpUpstream: httpUpstream, cfg: cfg}
}

func ProvideCNProviderQuotaService(accountRepo AccountRepository, proxyRepo ProxyRepository, httpUpstream HTTPUpstream, cfg *config.Config) *CNProviderQuotaService {
	return NewCNProviderQuotaService(accountRepo, proxyRepo, httpUpstream, cfg)
}

// QueryUsage probes one Coding Plan account. Concurrent requests for the same
// account are coalesced to avoid duplicate upstream calls.
func (s *CNProviderQuotaService) QueryUsage(ctx context.Context, accountID int64) (*CNProviderQuotaProbeResult, error) {
	if s == nil || s.accountRepo == nil || s.httpUpstream == nil {
		return nil, infraerrors.New(http.StatusInternalServerError, "CN_QUOTA_NOT_CONFIGURED", "cn provider quota service is not configured")
	}
	// context.Context is conventionally non-nil, but this service is also used
	// by optional admin/worker integrations. Treat a nil context as background
	// instead of panicking while selecting on ctx.Done below.
	if ctx == nil {
		ctx = context.Background()
	}
	key := "cn_quota:" + strconv.FormatInt(accountID, 10)
	resultCh := s.flight.DoChan(key, func() (any, error) {
		probeCtx, cancel := context.WithTimeout(context.Background(), cnQuotaUpstreamTimeout+5*time.Second)
		defer cancel()
		return s.queryUsage(probeCtx, accountID)
	})
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case flightResult := <-resultCh:
		if flightResult.Err != nil {
			return nil, flightResult.Err
		}
		result, ok := flightResult.Val.(*CNProviderQuotaProbeResult)
		if !ok || result == nil {
			return nil, infraerrors.New(http.StatusInternalServerError, "CN_QUOTA_PROBE_RESULT_INVALID", "invalid cn provider quota probe result")
		}
		clone := *result
		clone.Tiers = append([]CNQuotaTier(nil), result.Tiers...)
		return &clone, nil
	}
}

func (s *CNProviderQuotaService) queryUsage(ctx context.Context, accountID int64) (*CNProviderQuotaProbeResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	account, err := s.loadCodingPlanAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return s.queryUsageForAccount(ctx, account)
}

// QueryUsageForAccount probes an already-loaded Coding Plan account.  It is
// used by scheduler/monitor workers that already have a fresh account
// snapshot, avoiding a second repository read while retaining the same
// singleflight protection as QueryUsage.
func (s *CNProviderQuotaService) QueryUsageForAccount(ctx context.Context, account *Account) (*CNProviderQuotaProbeResult, error) {
	if s == nil || s.accountRepo == nil || s.httpUpstream == nil {
		return nil, infraerrors.New(http.StatusInternalServerError, "CN_QUOTA_NOT_CONFIGURED", "cn provider quota service is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateCodingPlanAccount(account); err != nil {
		return nil, err
	}
	key := "cn_quota:" + strconv.FormatInt(account.ID, 10)
	resultCh := s.flight.DoChan(key, func() (any, error) {
		probeCtx, cancel := context.WithTimeout(context.Background(), cnQuotaUpstreamTimeout+5*time.Second)
		defer cancel()
		return s.queryUsageForAccount(probeCtx, account)
	})
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case flightResult := <-resultCh:
		if flightResult.Err != nil {
			return nil, flightResult.Err
		}
		result, ok := flightResult.Val.(*CNProviderQuotaProbeResult)
		if !ok || result == nil {
			return nil, infraerrors.New(http.StatusInternalServerError, "CN_QUOTA_PROBE_RESULT_INVALID", "invalid cn provider quota probe result")
		}
		clone := *result
		clone.Tiers = append([]CNQuotaTier(nil), result.Tiers...)
		return &clone, nil
	}
}

func validateCodingPlanAccount(account *Account) error {
	if account == nil {
		return infraerrors.New(http.StatusNotFound, "CN_QUOTA_ACCOUNT_NOT_FOUND", "account not found")
	}
	if !account.IsCNProvider() {
		return infraerrors.New(http.StatusBadRequest, "CN_QUOTA_INVALID_PLATFORM", "account is not a CN provider account")
	}
	if !account.IsCodingPlan() {
		return infraerrors.New(http.StatusBadRequest, "CN_QUOTA_NOT_CODING_PLAN", "account is not a coding plan account")
	}
	if account.Type != AccountTypeAPIKey {
		return infraerrors.New(http.StatusBadRequest, "CN_QUOTA_INVALID_ACCOUNT_TYPE", "coding plan account must use an api key")
	}
	if strings.TrimSpace(account.GetCNAPIKey()) == "" {
		return infraerrors.New(http.StatusBadRequest, "CN_QUOTA_NO_APIKEY", "account api_key is empty")
	}
	return nil
}

func (s *CNProviderQuotaService) queryUsageForAccount(ctx context.Context, account *Account) (*CNProviderQuotaProbeResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	provider := account.GetCodingPlanProvider()
	if provider != PlatformZhipu {
		return nil, infraerrors.New(http.StatusBadRequest, "CN_QUOTA_NOT_CODING_PLAN", "account is not a zhipu coding plan account")
	}
	apiKey := strings.TrimSpace(account.GetCNAPIKey())
	if apiKey == "" {
		return nil, infraerrors.New(http.StatusBadRequest, "CN_QUOTA_NO_APIKEY", "account api_key is empty")
	}

	targetURL := zhipuQuotaURL(account.GetOpenAIBaseURL())
	org := strings.TrimSpace(account.GetCredential("zhipu_organization"))
	if org != "" {
		targetURL += "?type=2"
	}
	validatedURL, err := cnValidateProbeURL(s.cfg, targetURL)
	if err != nil {
		return nil, infraerrors.New(http.StatusForbidden, "CN_QUOTA_URL_REJECTED", err.Error())
	}
	proxyURL := s.resolveProxyURL(ctx, account)
	callCtx, cancel := context.WithTimeout(ctx, cnQuotaUpstreamTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(callCtx, http.MethodGet, validatedURL, nil)
	if err != nil {
		return nil, infraerrors.Newf(http.StatusInternalServerError, "CN_QUOTA_REQUEST_BUILD_FAILED", "build request: %v", err)
	}
	// Quota probes carry a live API key. Use the OpenAI-compatible transport
	// policy and never follow redirects, which could otherwise forward the key
	// to an attacker-controlled Location target.
	reqCtx := WithHTTPUpstreamProfile(req.Context(), HTTPUpstreamProfileOpenAI)
	req = req.WithContext(WithHTTPUpstreamRedirectsDisabled(reqCtx))
	req.Header.Set("Authorization", apiKey) // Zhipu quota API expects the raw key.
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "en-US,en")
	if org != "" {
		req.Header.Set("bigmodel-organization", org)
		if project := strings.TrimSpace(account.GetCredential("zhipu_project")); project != "" {
			req.Header.Set("bigmodel-project", project)
		}
	}
	account.ApplyHeaderOverrides(req.Header)
	resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, maxInt(account.Concurrency, 1))
	if err != nil {
		return nil, infraerrors.Newf(http.StatusBadGateway, "CN_QUOTA_REQUEST_FAILED", "upstream request failed: %v", err)
	}
	if resp == nil || resp.Body == nil {
		return nil, infraerrors.New(http.StatusBadGateway, "CN_QUOTA_EMPTY_RESPONSE", "upstream returned an empty response")
	}
	defer func() { _ = resp.Body.Close() }()
	bodyBytes, readErr := io.ReadAll(io.LimitReader(resp.Body, cnQuotaMaxBodyBytes+1))
	if readErr != nil {
		return nil, infraerrors.Newf(http.StatusBadGateway, "CN_QUOTA_RESPONSE_READ_FAILED", "read upstream response: %v", readErr)
	}
	if len(bodyBytes) > cnQuotaMaxBodyBytes {
		return nil, infraerrors.New(http.StatusBadGateway, "CN_QUOTA_RESPONSE_TOO_LARGE", "upstream quota response is too large")
	}

	now := time.Now().UTC()
	result := &CNProviderQuotaProbeResult{Provider: provider, Source: "coding_plan", FetchedAt: now.Unix(), StatusCode: resp.StatusCode}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		result.Error = fmt.Sprintf("Authentication failed (HTTP %d)", resp.StatusCode)
		return result, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		result.Error = fmt.Sprintf("API error (HTTP %d): %s", resp.StatusCode, truncate(strings.TrimSpace(string(bodyBytes)), 240))
		return result, nil
	}
	if success := gjson.GetBytes(bodyBytes, "success"); success.Exists() && !success.Bool() {
		msg := strings.TrimSpace(gjson.GetBytes(bodyBytes, "msg").String())
		if msg == "" {
			msg = "unknown zhipu quota error"
		}
		result.Error = "API error: " + msg
		return result, nil
	}
	result.Tiers = parseZhipuTokenTiers(gjson.GetBytes(bodyBytes, "data"))
	result.PlanLevel = strings.TrimSpace(gjson.GetBytes(bodyBytes, "data.level").String())
	result.Success = true
	result.CredentialValid = true
	if err := s.accountRepo.UpdateExtra(ctx, account.ID, cnQuotaExtraUpdates(provider, result.Tiers, now)); err != nil {
		slog.Warn("cn_quota_persist_failed", "account_id", account.ID, "error", err)
	} else {
		result.Persisted = true
	}
	return result, nil
}

func (s *CNProviderQuotaService) loadCodingPlanAccount(ctx context.Context, accountID int64) (*Account, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, infraerrors.Newf(http.StatusNotFound, "CN_QUOTA_ACCOUNT_NOT_FOUND", "account not found: %v", err)
	}
	if account == nil {
		return nil, infraerrors.New(http.StatusNotFound, "CN_QUOTA_ACCOUNT_NOT_FOUND", "account not found")
	}
	if !account.IsCNProvider() {
		return nil, infraerrors.New(http.StatusBadRequest, "CN_QUOTA_INVALID_PLATFORM", "account is not a CN provider account")
	}
	if !account.IsCodingPlan() {
		return nil, infraerrors.New(http.StatusBadRequest, "CN_QUOTA_NOT_CODING_PLAN", "account is not a coding plan account")
	}
	if account.Type != AccountTypeAPIKey {
		return nil, infraerrors.New(http.StatusBadRequest, "CN_QUOTA_INVALID_ACCOUNT_TYPE", "coding plan account must use an api key")
	}
	return account, nil
}

func (s *CNProviderQuotaService) resolveProxyURL(ctx context.Context, account *Account) string {
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

func zhipuQuotaURL(baseURL string) string {
	return zhipuQuotaHost(baseURL) + "/api/monitor/usage/quota/limit"
}

func zhipuQuotaHost(baseURL string) string {
	lower := strings.ToLower(strings.TrimSpace(baseURL))
	if strings.Contains(lower, "api.z.ai") {
		return "https://api.z.ai"
	}
	return "https://open.bigmodel.cn"
}

type cnZhipuWindow int

const (
	cnZhipuWindowUnknown cnZhipuWindow = iota
	cnZhipuWindow5h
	cnZhipuWindowWeekly
)

func classifyZhipuWindowUnit(unit int64) cnZhipuWindow {
	switch unit {
	case 3:
		return cnZhipuWindow5h
	case 6:
		return cnZhipuWindowWeekly
	default:
		return cnZhipuWindowUnknown
	}
}

// parseZhipuTokenTiers parses data.limits from the GLM quota response. Explicit
// unit values (3=5h, 6=weekly) take precedence over reset-time heuristics;
// CREDIT_LIMIT is used only when no TOKENS_LIMIT entry exists.
func parseZhipuTokenTiers(data gjson.Result) []CNQuotaTier {
	type entry struct {
		resetMs    int64
		hasReset   bool
		percentage float64
		resetISO   string
	}
	var fiveHour, weekly entry
	var fiveHourSet, weeklySet bool
	var unclassified, creditFallback []entry
	hasTokensLimit := false

	classify := func(item gjson.Result, e entry) {
		switch classifyZhipuWindowUnit(item.Get("unit").Int()) {
		case cnZhipuWindow5h:
			if !fiveHourSet {
				fiveHour, fiveHourSet = e, true
			} else {
				unclassified = append(unclassified, e)
			}
		case cnZhipuWindowWeekly:
			if !weeklySet {
				weekly, weeklySet = e, true
			} else {
				unclassified = append(unclassified, e)
			}
		default:
			unclassified = append(unclassified, e)
		}
	}

	data.Get("limits").ForEach(func(_, item gjson.Result) bool {
		kind := strings.ToUpper(strings.TrimSpace(item.Get("type").String()))
		if kind != "TOKENS_LIMIT" && kind != "CREDIT_LIMIT" {
			return true
		}
		percentage := 0.0
		if value, ok := cnParseF64(item.Get("percentage").Value()); ok {
			percentage = value
		}
		var resetMs int64
		var resetISO string
		next := item.Get("nextResetTime")
		if next.Exists() {
			switch next.Type {
			case gjson.Number:
				resetMs = next.Int()
				if resetMs > 0 {
					resetISO = cnMillisToRFC3339(resetMs)
				}
			case gjson.String:
				resetISO = cnNormalizeResetTime(next.String())
			}
		}
		// Keep the sort key in a single unit (milliseconds since epoch). The
		// upstream occasionally serializes nextResetTime as an ISO/epoch string;
		// leaving resetMs at zero for those entries makes fallback ordering depend
		// on response order instead of the actual reset time.
		if resetISO != "" {
			if parsed, parseErr := time.Parse(time.RFC3339, resetISO); parseErr == nil {
				resetMs = parsed.UnixMilli()
			}
		}
		e := entry{resetMs: resetMs, hasReset: resetISO != "", percentage: percentage, resetISO: resetISO}
		if kind == "TOKENS_LIMIT" {
			hasTokensLimit = true
			classify(item, e)
		} else {
			creditFallback = append(creditFallback, e)
		}
		return true
	})
	if !hasTokensLimit {
		unclassified = append(unclassified, creditFallback...)
	}
	sort.SliceStable(unclassified, func(i, j int) bool {
		if unclassified[i].hasReset != unclassified[j].hasReset {
			return !unclassified[i].hasReset
		}
		return unclassified[i].resetMs < unclassified[j].resetMs
	})
	for _, e := range unclassified {
		if !fiveHourSet {
			fiveHour, fiveHourSet = e, true
		} else if !weeklySet {
			weekly, weeklySet = e, true
		}
	}
	result := make([]CNQuotaTier, 0, 2)
	if fiveHourSet {
		result = append(result, CNQuotaTier{Window: "5h", UsedPercent: fiveHour.percentage, ResetAt: fiveHour.resetISO})
	}
	if weeklySet {
		result = append(result, CNQuotaTier{Window: "weekly", UsedPercent: weekly.percentage, ResetAt: weekly.resetISO})
	}
	return result
}

func cnQuotaExtraUpdates(provider string, tiers []CNQuotaTier, now time.Time) map[string]any {
	updates := map[string]any{cnExtraKey(provider, cnExtraSuffixUsageUpdated): now.Format(time.RFC3339)}
	for _, tier := range tiers {
		switch tier.Window {
		case "5h":
			updates[cnExtraKey(provider, cnExtraSuffix5hUsed)] = tier.UsedPercent
			if tier.ResetAt != "" {
				updates[cnExtraKey(provider, cnExtraSuffix5hReset)] = tier.ResetAt
			}
		case "weekly":
			updates[cnExtraKey(provider, cnExtraSuffixWeeklyUsed)] = tier.UsedPercent
			if tier.ResetAt != "" {
				updates[cnExtraKey(provider, cnExtraSuffixWeeklyReset)] = tier.ResetAt
			}
		}
	}
	return updates
}

func cnParseF64(raw any) (float64, bool) {
	switch value := raw.(type) {
	case float64:
		return value, true
	case float32:
		return float64(value), true
	case int:
		return float64(value), true
	case int64:
		return float64(value), true
	case json.Number:
		parsed, err := value.Float64()
		return parsed, err == nil
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func cnNormalizeResetTime(raw any) string {
	switch value := raw.(type) {
	case string:
		value = strings.TrimSpace(value)
		if value == "" {
			return ""
		}
		for _, layout := range []string{
			time.RFC3339Nano,
			time.RFC3339,
			"2006-01-02 15:04:05Z07:00",
			"2006-01-02 15:04:05",
			time.RFC1123Z,
			time.RFC1123,
		} {
			if parsed, err := time.Parse(layout, value); err == nil {
				return parsed.UTC().Format(time.RFC3339)
			}
		}
		// Some GLM deployments serialize epoch milliseconds as a JSON string
		// (for example, "1760000000000") instead of a JSON number. Accept both
		// integer and decimal representations so the UI does not lose reset data.
		if epoch, err := strconv.ParseFloat(value, 64); err == nil && epoch > 0 {
			return cnMillisToRFC3339(int64(epoch))
		}
		return ""
	case float64:
		return cnMillisToRFC3339(int64(value))
	case int:
		return cnMillisToRFC3339(int64(value))
	case int64:
		return cnMillisToRFC3339(value)
	case json.Number:
		parsed, err := value.Int64()
		if err == nil {
			return cnMillisToRFC3339(parsed)
		}
	}
	return ""
}

func cnMillisToRFC3339(value int64) string {
	if value <= 0 {
		return ""
	}
	if value < 1_000_000_000_000 {
		value *= 1000
	}
	return time.UnixMilli(value).UTC().Format(time.RFC3339)
}
