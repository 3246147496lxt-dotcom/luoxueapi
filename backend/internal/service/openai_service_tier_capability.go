package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"golang.org/x/sync/singleflight"
)

// OpenAIServiceTierSupportState is deliberately tri-state. Unknown is treated
// as Standard for the current request and may be populated asynchronously from
// the account's Codex model manifest.
type OpenAIServiceTierSupportState uint8

const (
	OpenAIServiceTierSupportUnknown OpenAIServiceTierSupportState = iota
	OpenAIServiceTierSupportUnsupported
	OpenAIServiceTierSupportSupported
)

const (
	openAIServiceTierCapabilityTTL      = 10 * time.Minute
	openAIServiceTierCapabilityNegative = 2 * time.Minute
	// Unknown is kept briefly after a failed/background probe. It remains
	// fail-closed for the current request, while allowing later traffic to
	// retry discovery without hammering an unavailable manifest endpoint.
	openAIServiceTierCapabilityUnknown = 15 * time.Second
	openAIServiceTierProbeTimeout      = 15 * time.Second
	openAIServiceTierModel             = "gpt-5.6-sol"
)

type openAIServiceTierCapabilityEntry struct {
	state     OpenAIServiceTierSupportState
	expiresAt time.Time
}

type openAIServiceTierCapabilityCache struct {
	mu      sync.RWMutex
	entries map[string]openAIServiceTierCapabilityEntry
	probes  singleflight.Group
}

func (c *openAIServiceTierCapabilityCache) get(key string, now time.Time) (OpenAIServiceTierSupportState, bool) {
	if c == nil || key == "" {
		return OpenAIServiceTierSupportUnknown, false
	}
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok || !now.Before(entry.expiresAt) {
		if ok {
			c.mu.Lock()
			// Do not remove a newer value installed by a concurrent manifest
			// observer between the read and this expiry cleanup.
			if current, stillPresent := c.entries[key]; stillPresent &&
				current.state == entry.state && current.expiresAt.Equal(entry.expiresAt) {
				delete(c.entries, key)
			}
			c.mu.Unlock()
		}
		return OpenAIServiceTierSupportUnknown, false
	}
	return entry.state, true
}

// setUnknownIfNotKnown records a failed discovery only while the entry is
// absent/expired or already unknown. A late probe failure must not overwrite a
// supported/unsupported observation produced by a concurrent manifest fetch.
func (c *openAIServiceTierCapabilityCache) setUnknownIfNotKnown(key string, now time.Time) {
	if c == nil || key == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if current, ok := c.entries[key]; ok && now.Before(current.expiresAt) &&
		current.state != OpenAIServiceTierSupportUnknown {
		return
	}
	if c.entries == nil {
		c.entries = make(map[string]openAIServiceTierCapabilityEntry)
	}
	c.entries[key] = openAIServiceTierCapabilityEntry{
		state:     OpenAIServiceTierSupportUnknown,
		expiresAt: now.Add(openAIServiceTierCapabilityUnknown),
	}
}

func (c *openAIServiceTierCapabilityCache) set(key string, state OpenAIServiceTierSupportState, now time.Time) {
	if c == nil || key == "" {
		return
	}
	ttl := openAIServiceTierCapabilityTTL
	switch state {
	case OpenAIServiceTierSupportUnsupported:
		ttl = openAIServiceTierCapabilityNegative
	case OpenAIServiceTierSupportUnknown:
		ttl = openAIServiceTierCapabilityUnknown
	}
	c.mu.Lock()
	if c.entries == nil {
		c.entries = make(map[string]openAIServiceTierCapabilityEntry)
	}
	c.entries[key] = openAIServiceTierCapabilityEntry{state: state, expiresAt: now.Add(ttl)}
	c.mu.Unlock()
}

func openAIServiceTierCapabilityKey(account *Account, upstreamModel string) string {
	if account == nil {
		return ""
	}
	isOAuth := account.IsOpenAIOAuth()
	base := strings.TrimSpace(account.GetOpenAIBaseURL())
	if isOAuth {
		base = chatgptCodexModelsURL
	}
	if base == "" {
		base = "https://api.openai.com"
	}
	if parsed, err := url.Parse(base); err == nil {
		// OAuth manifests always use the process-wide ChatGPT endpoint; query
		// parameters (including client_version) are request metadata, not a
		// distinct upstream capability. Custom API-key bases may use a static
		// tenant query parameter, so retain it to avoid sharing entitlements
		// between otherwise different upstream addresses.
		if isOAuth {
			parsed.RawQuery = ""
		}
		parsed.Fragment = ""
		parsed.Path = strings.TrimRight(parsed.Path, "/")
		parsed.Scheme = strings.ToLower(parsed.Scheme)
		parsed.Host = strings.ToLower(parsed.Host)
		base = parsed.String()
	}
	h := sha256.Sum256([]byte(base))
	modelKey := normalizeKnownOpenAICodexModel(upstreamModel)
	if modelKey == "" {
		modelKey = strings.ToLower(strings.TrimSpace(upstreamModel))
	}
	return accountCredentialIDForTier(account) + ":" + hex.EncodeToString(h[:]) + ":" + modelKey
}

// openAIServiceTierSelectedAccountKey identifies the account selected by the
// scheduler while retaining its selected-account upstream address. The
// credential ID component intentionally follows accountCredentialIDForTier,
// so a shadow account and its parent share an identity dimension but can still
// be aliased when their effective upstream address differs.
func openAIServiceTierSelectedAccountKey(account *Account, upstreamModel string) string {
	return openAIServiceTierCapabilityKey(account, upstreamModel)
}

func accountCredentialIDForTier(account *Account) string {
	if account == nil {
		return "0"
	}
	if account.ParentAccountID != nil && *account.ParentAccountID > 0 {
		return stringID(*account.ParentAccountID)
	}
	return stringID(account.ID)
}

func stringID(id int64) string {
	return strconv.FormatInt(id, 10)
}

// pricingSupportsOpenAIPriority is intentionally exact to the target model.
// The project fallback pricing is also a valid local price declaration.
func (s *OpenAIGatewayService) pricingSupportsOpenAIPriority(model string) bool {
	if s == nil || s.billingService == nil || normalizeKnownOpenAICodexModel(model) != openAIServiceTierModel {
		return false
	}
	pricing, err := s.billingService.GetModelPricing(openAIServiceTierModel)
	if err != nil || pricing == nil {
		return false
	}
	return usePriorityServiceTierPricing(OpenAIFastTierPriority, pricing)
}

func (s *OpenAIGatewayService) serviceTierSupport(ctx context.Context, account *Account, model string) OpenAIServiceTierSupportState {
	if s == nil || account == nil || !account.IsOpenAI() || normalizeKnownOpenAICodexModel(model) != openAIServiceTierModel || !s.pricingSupportsOpenAIPriority(model) {
		return OpenAIServiceTierSupportUnsupported
	}
	key := openAIServiceTierCapabilityKey(account, openAIServiceTierModel)
	state, cached := s.openAIServiceTierCapabilities.get(key, time.Now())
	if !cached {
		selectedKey := openAIServiceTierSelectedAccountKey(account, openAIServiceTierModel)
		if selectedKey != key {
			state, cached = s.openAIServiceTierCapabilities.get(selectedKey, time.Now())
		}
	}
	if !cached {
		s.startServiceTierProbe(ctx, account)
	}
	return state
}

func (s *OpenAIGatewayService) startServiceTierProbe(ctx context.Context, account *Account) {
	if s == nil || account == nil {
		return
	}
	key := openAIServiceTierCapabilityKey(account, openAIServiceTierModel)
	if key == "" {
		return
	}
	_ = s.openAIServiceTierCapabilities.probes.DoChan(key, func() (any, error) {
		probeCtx, cancel := context.WithTimeout(context.Background(), openAIServiceTierProbeTimeout)
		defer cancel()
		manifest, err := s.FetchCodexModelsManifest(probeCtx, account, openAICodexProbeVersion, "")
		if err != nil || manifest == nil {
			// Cache the unknown result briefly so a broken manifest endpoint does
			// not launch one network probe per inference request. The next request
			// after the short unknown TTL retries discovery.
			s.openAIServiceTierCapabilities.setUnknownIfNotKnown(key, time.Now())
			return nil, err
		}
		// A 304 without a local body (possible for an unusual OAuth/custom
		// upstream) carries no capability data. Keep the result fail-closed and
		// rate-limit retries instead of leaving the key uncached.
		if _, cached := s.openAIServiceTierCapabilities.get(key, time.Now()); !cached {
			s.openAIServiceTierCapabilities.setUnknownIfNotKnown(key, time.Now())
		}
		return nil, nil
	})
}

// observeCodexModelsManifest updates capability metadata after a successful
// manifest fetch. It never changes the verbatim body returned to clients.
func (s *OpenAIGatewayService) observeCodexModelsManifest(request codexModelsManifestRequest, body []byte) {
	if s == nil || len(body) == 0 {
		return
	}
	// Keep this helper fail-closed when called by future fetch paths: only a
	// validated Codex envelope may write capability state. In particular, a
	// normal OpenAI `{object:"list",data:[...]}` response must never become a
	// negative capability observation.
	if err := validateCodexModelsManifestEnvelope(body); err != nil {
		return
	}
	var envelope struct {
		Models []json.RawMessage `json:"models"`
	}
	if json.Unmarshal(body, &envelope) != nil {
		return
	}
	key := tierCapabilityKeyFromManifestRequest(request, openAIServiceTierModel)
	if key == "" {
		return
	}
	selectedKey := ""
	if request.account != nil {
		selectedKey = openAIServiceTierSelectedAccountKey(request.account, openAIServiceTierModel)
	}
	matched := false
	anySupported := false
	for _, raw := range envelope.Models {
		var model struct {
			Slug                 string   `json:"slug"`
			Model                string   `json:"model"`
			ModelSlug            string   `json:"model_slug"`
			ID                   string   `json:"id"`
			Name                 string   `json:"name"`
			AdditionalSpeedTiers []string `json:"additional_speed_tiers"`
			ServiceTiers         []struct {
				ID string `json:"id"`
			} `json:"service_tiers"`
		}
		if json.Unmarshal(raw, &model) != nil {
			continue
		}
		candidates := []string{model.Slug, model.ModelSlug, model.Model, model.ID, model.Name}
		canonical := ""
		for _, candidate := range candidates {
			if normalizeKnownOpenAICodexModel(candidate) == openAIServiceTierModel {
				canonical = openAIServiceTierModel
				break
			}
		}
		if canonical == "" {
			continue
		}
		matched = true
		supported := false
		for _, tier := range model.ServiceTiers {
			if strings.EqualFold(strings.TrimSpace(tier.ID), OpenAIFastTierPriority) {
				supported = true
				break
			}
		}
		if !supported {
			for _, tier := range model.AdditionalSpeedTiers {
				if strings.EqualFold(strings.TrimSpace(tier), "fast") {
					supported = true
					break
				}
			}
		}
		if supported {
			anySupported = true
		}
	}
	state := OpenAIServiceTierSupportUnsupported
	if matched && anySupported {
		state = OpenAIServiceTierSupportSupported
	}
	// A valid manifest with no target model is a verified negative result, not
	// an unknown one; avoid launching a probe on every request in that case.
	s.openAIServiceTierCapabilities.set(key, state, time.Now())
	if selectedKey != "" && selectedKey != key {
		s.openAIServiceTierCapabilities.set(selectedKey, state, time.Now())
	}
}

func tierCapabilityKeyFromManifestRequest(request codexModelsManifestRequest, model string) string {
	account := request.credentialAccount
	if account == nil {
		account = &Account{ID: request.credentialAccountID, Platform: PlatformOpenAI}
	}
	return openAIServiceTierCapabilityKey(account, model)
}

func apiKeyServiceTierPreference(ctx context.Context, c *gin.Context) string {
	if ctx != nil {
		if webChat, _ := ctx.Value(ctxkey.WebChat).(bool); webChat {
			return ServiceTierPreferenceStandard
		}
		if value, ok := ctx.Value(ctxkey.OpenAIServiceTierPreference).(string); ok {
			if normalized, valid := NormalizeServiceTierPreference(value); valid {
				return normalized
			}
		}
	}
	// Do not fall back to Gin keys or request metadata here. The preference is
	// trusted only when the API-key authentication middleware writes it to the
	// typed request context above; this prevents a handler/client from
	// smuggling a default tier through a mutable Gin value.
	_ = c
	return ServiceTierPreferenceStandard
}

// injectDefaultOpenAIServiceTier adds Priority only when the server-side key
// preference and verified upstream capability both allow it. It deliberately
// checks field presence with gjson, so explicit null/empty values are preserved.
func (s *OpenAIGatewayService) injectDefaultOpenAIServiceTier(ctx context.Context, c *gin.Context, account *Account, model string, body []byte) ([]byte, bool, error) {
	if len(body) == 0 || apiKeyServiceTierPreference(ctx, c) != ServiceTierPreferencePriority || account == nil || !account.IsOpenAI() || normalizeKnownOpenAICodexModel(model) != openAIServiceTierModel || gjson.GetBytes(body, "service_tier").Exists() {
		return body, false, nil
	}
	if s.serviceTierSupport(ctx, account, model) != OpenAIServiceTierSupportSupported {
		return body, false, nil
	}
	updated, err := sjson.SetBytes(body, "service_tier", OpenAIFastTierPriority)
	if err != nil {
		return body, false, err
	}
	return updated, true, nil
}
