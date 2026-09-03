package service

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai_compat"
	"github.com/gin-gonic/gin"
)

const (
	WebChatReasoningModeStandard = "standard"
	WebChatReasoningModePro      = "pro"

	webChatReasoningOptionsContextKey = "sub2api.web_chat.reasoning_options"
)

// WebChatReasoningOptions is the trusted, server-validated reasoning contract
// passed from ChatHandler to the OpenAI scheduler/forwarding layer. Mode and
// effort are independent; every Web Chat request requires a Responses API
// reasoning summary.
type WebChatReasoningOptions struct {
	Mode   string
	Effort string
}

// NormalizeWebChatReasoningMode normalizes the product mode without consulting
// request headers. An omitted mode preserves the Web Chat default of standard.
func NormalizeWebChatReasoningMode(mode string) (string, bool) {
	switch normalized := strings.ToLower(strings.TrimSpace(mode)); normalized {
	case "", WebChatReasoningModeStandard:
		return WebChatReasoningModeStandard, true
	case WebChatReasoningModePro:
		return WebChatReasoningModePro, true
	default:
		return "", false
	}
}

// SetWebChatReasoningOptions stores trusted application state directly on the
// Gin context. It intentionally does not read or mirror any client header.
func SetWebChatReasoningOptions(c *gin.Context, options WebChatReasoningOptions) {
	if c == nil {
		return
	}
	c.Set(webChatReasoningOptionsContextKey, options)
}

// GetWebChatReasoningOptions returns only options written by
// SetWebChatReasoningOptions; HTTP headers cannot populate this value.
func GetWebChatReasoningOptions(c *gin.Context) (WebChatReasoningOptions, bool) {
	if c == nil {
		return WebChatReasoningOptions{}, false
	}
	value, exists := c.Get(webChatReasoningOptionsContextKey)
	if !exists {
		return WebChatReasoningOptions{}, false
	}
	options, ok := value.(WebChatReasoningOptions)
	return options, ok
}

// AccountSupportsWebChatReasoning is the shared fail-closed capability gate
// for catalog admission and runtime account selection. API-key accounts must
// have an affirmative Responses probe (or an explicit force_responses mode):
// unknown probes and force_chat_completions are rejected because they cannot
// guarantee a native reasoning-summary stream.
func AccountSupportsWebChatReasoning(account *Account, upstreamModel string, options WebChatReasoningOptions) bool {
	mode, valid := NormalizeWebChatReasoningMode(options.Mode)
	if !valid || account == nil || !account.IsOpenAI() ||
		!account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityResponses) {
		return false
	}

	switch account.Type {
	case AccountTypeAPIKey:
		if openai_compat.ResolveResponsesSupport(account.Extra) != openai_compat.ResponsesSupportYes {
			return false
		}
	case AccountTypeOAuth, AccountTypeSetupToken:
		// OpenAI OAuth/setup-token accounts are native Responses accounts.
	default:
		return false
	}

	model, known := openai.DefaultModelByID(upstreamModel)
	if !known || !model.SupportsResponses || !model.SupportsReasoningSummary ||
		!model.SupportsReasoningEffort(options.Effort) {
		return false
	}
	if mode != WebChatReasoningModePro {
		return true
	}
	// Mode and effort remain independent: the model capability table validates
	// the effort above, while this branch validates only Pro-mode support.
	return model.SupportsReasoningProMode
}

// AccountSupportsWebChatReasoningForModel applies the account-level model
// mapping and upstream normalization used by the actual forwarding path before
// evaluating the shared capability predicate.
func (s *OpenAIGatewayService) AccountSupportsWebChatReasoningForModel(
	account *Account,
	forwardedModel string,
	options WebChatReasoningOptions,
) bool {
	if s == nil || account == nil {
		return false
	}
	accountMappedModel := resolveOpenAIAccountUpstreamModelForRequest(account, forwardedModel, false)
	upstreamModel := normalizeOpenAIModelForUpstream(account, accountMappedModel)
	return AccountSupportsWebChatReasoning(account, upstreamModel, options)
}
