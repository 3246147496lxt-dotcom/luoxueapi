package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
)

var (
	ErrChatInsufficientBalance     = infraerrors.Forbidden("INSUFFICIENT_BALANCE", "Insufficient account balance")
	ErrChatModelNotAvailable       = infraerrors.BadRequest("CHAT_MODEL_NOT_AVAILABLE", "The selected model is not available")
	ErrChatReasoningNotAllowed     = infraerrors.BadRequest("CHAT_REASONING_EFFORT_NOT_AVAILABLE", "The selected reasoning effort is not available for this model")
	ErrChatReasoningModeInvalid    = infraerrors.BadRequest("CHAT_REASONING_MODE_INVALID", "The selected reasoning mode is invalid")
	ErrChatProReasoningUnavailable = infraerrors.BadRequest("PRO_REASONING_UNAVAILABLE", "Pro reasoning is not available for this model or account")
	ErrChatCatalogUnavailable      = infraerrors.ServiceUnavailable("CHAT_CATALOG_UNAVAILABLE", "Chat model catalog is temporarily unavailable")
	ErrChatTranscriptionOff        = infraerrors.NotFound("TRANSCRIPTION_DISABLED", "Voice transcription is not enabled")
	ErrChatTranscriptionAbsent     = infraerrors.ServiceUnavailable("TRANSCRIPTION_UNAVAILABLE", "Voice transcription is temporarily unavailable")
)

const preferredChatModelID = "gpt-5.6-sol"

type ChatModelPricing struct {
	BillingMode     string  `json:"billing_mode"`
	Unit            string  `json:"unit"`
	InputPrice      float64 `json:"input_price"`
	OutputPrice     float64 `json:"output_price"`
	PerRequestPrice float64 `json:"per_request_price,omitempty"`
	RateMultiplier  float64 `json:"rate_multiplier"`
}

type ChatModel struct {
	ID                        string           `json:"id"`
	DisplayName               string           `json:"display_name"`
	Recommended               bool             `json:"recommended"`
	InputPrice                float64          `json:"input_price"`
	OutputPrice               float64          `json:"output_price"`
	Pricing                   ChatModelPricing `json:"pricing"`
	SupportsVision            bool             `json:"supports_vision"`
	SupportsReasoningSlider   bool             `json:"supports_reasoning_slider"`
	SupportsResponses         bool             `json:"supports_responses"`
	SupportsReasoningSummary  bool             `json:"supports_reasoning_summary"`
	SupportsReasoningProMode  bool             `json:"supports_reasoning_pro_mode"`
	SupportedReasoningEfforts []string         `json:"supported_reasoning_efforts"`
}

type ChatModelsResult struct {
	Models        []ChatModel                 `json:"models"`
	Balance       float64                     `json:"balance"`
	Transcription ChatTranscriptionCapability `json:"transcription"`
}

// ChatPrincipal is the trusted routing and billing identity selected for one
// Web Chat request. Subscription must be present exactly when APIKey belongs
// to a subscription group.
type ChatPrincipal struct {
	APIKey       *APIKey
	Subscription *UserSubscription
}

type ChatCapabilitiesResult struct {
	Transcription ChatTranscriptionProductCapability `json:"transcription"`
}

type ChatTranscriptionProductCapability struct {
	Enabled            bool     `json:"enabled"`
	MaxUploadBytes     int64    `json:"max_upload_bytes,omitempty"`
	MaxDurationSeconds int      `json:"max_duration_seconds,omitempty"`
	AcceptedMIMETypes  []string `json:"accepted_mime_types,omitempty"`
}

type ChatTranscriptionCapability struct {
	Enabled            bool     `json:"enabled"`
	BillingMode        string   `json:"billing_mode,omitempty"`
	MaxUploadBytes     int64    `json:"max_upload_bytes,omitempty"`
	MaxDurationSeconds int      `json:"max_duration_seconds,omitempty"`
	AcceptedMIMETypes  []string `json:"accepted_mime_types,omitempty"`
}

type chatUserReader interface {
	GetByID(ctx context.Context, id int64) (*User, error)
}

type chatGroupAccess interface {
	GetAvailableGroups(ctx context.Context, userID int64) ([]Group, error)
}

type chatModelCatalog interface {
	GetAvailableModelsStrict(ctx context.Context, groupID *int64, platform string) ([]string, error)
	ResolveChannelMappingStrict(ctx context.Context, groupID int64, model string) (ChannelMappingResult, error)
	IsModelRestrictedStrict(ctx context.Context, groupID int64, model string) (bool, error)
	ResolveUserGroupRateMultiplier(ctx context.Context, userID, groupID int64, groupDefaultMultiplier float64) float64
}

type chatModelSchedulability interface {
	HasSchedulableChatCompletionsAccount(ctx context.Context, groupID int64, model string) (bool, error)
	HasSchedulableWebChatReasoningAccount(ctx context.Context, groupID int64, model string, options WebChatReasoningOptions) (bool, error)
}

type chatTranscriptionSchedulability interface {
	HasSchedulableTranscriptionAccount(ctx context.Context, groupID int64, model string) (bool, error)
}

type chatTranscriptionConfigProvider interface {
	EffectiveTranscriptionConfig(ctx context.Context) (config.TranscriptionConfig, error)
}

type chatPricingResolver interface {
	Resolve(ctx context.Context, input PricingInput) *ResolvedPricing
	GetIntervalPricing(resolved *ResolvedPricing, totalContextTokens int) *ModelPricing
}

type chatPrincipalProvider interface {
	Resolve(ctx context.Context, userID int64, group *Group, subscription *UserSubscription) (*APIKey, error)
}

type chatBillingEligibility interface {
	PeekWebChatEligibility(ctx context.Context, userID int64, platform string) (float64, error)
	PeekWebChatSubscriptionEligibility(ctx context.Context, userID int64, group *Group, subscription *UserSubscription) error
}

type chatSubscriptionProvider interface {
	GetActiveSubscription(ctx context.Context, userID, groupID int64) (*UserSubscription, error)
	ValidateAndCheckLimits(sub *UserSubscription, group *Group) (needsMaintenance bool, err error)
	EnsureWindowMaintenance(ctx context.Context, sub *UserSubscription) (*UserSubscription, error)
}

type ChatService struct {
	users      chatUserReader
	groups     chatGroupAccess
	catalog    chatModelCatalog
	scheduler  chatModelSchedulability
	pricing    chatPricingResolver
	billing    chatBillingEligibility
	subs       chatSubscriptionProvider
	principals chatPrincipalProvider
	transcribe config.TranscriptionConfig
}

func NewChatService(
	users UserRepository,
	groups *APIKeyService,
	catalog *GatewayService,
	scheduler *OpenAIGatewayService,
	pricing *ModelPricingResolver,
	billing *BillingCacheService,
	subscriptions *SubscriptionService,
	principals *ChatPrincipalResolver,
) *ChatService {
	service := &ChatService{
		users:      users,
		groups:     groups,
		catalog:    catalog,
		scheduler:  scheduler,
		pricing:    pricing,
		billing:    billing,
		subs:       subscriptions,
		principals: principals,
	}
	if scheduler != nil && scheduler.cfg != nil && scheduler.cfg.RunMode != config.RunModeSimple {
		service.transcribe = scheduler.cfg.Transcription
	}
	return service
}

func (s *ChatService) ListModels(ctx context.Context, userID int64) (*ChatModelsResult, error) {
	_, balance, choices, err := s.authorizedModelChoices(ctx, userID)
	if err != nil {
		return nil, err
	}
	models := make([]ChatModel, 0, len(choices))
	for i := range choices {
		model := choices[i].model
		if isHiddenWebChatCatalogModelID(model.ID) {
			continue
		}
		model.Recommended = strings.EqualFold(model.ID, preferredChatModelID)
		models = append(models, model)
	}
	capability := s.transcriptionCapability(ctx, userID)
	return &ChatModelsResult{Models: models, Balance: balance, Transcription: capability}, nil
}

// Capabilities returns product-level Chat capabilities without performing any
// per-user admission or runtime account checks. Actual transcription requests
// still resolve the user principal and scheduler state at submission time.
func (s *ChatService) Capabilities(ctx context.Context) (*ChatCapabilitiesResult, error) {
	cfg, err := s.effectiveTranscriptionConfig(ctx)
	if err != nil {
		cfg = config.TranscriptionConfig{}
		if s != nil {
			cfg = cloneTranscriptionConfig(s.transcribe)
		}
		cfg.Enabled = false
	}
	return &ChatCapabilitiesResult{
		Transcription: ChatTranscriptionProductCapability{
			Enabled:            cfg.Enabled,
			MaxUploadBytes:     cfg.MaxUploadBytes,
			MaxDurationSeconds: cfg.MaxDurationSeconds,
			AcceptedMIMETypes:  append([]string(nil), cfg.AcceptedMIMETypes...),
		},
	}, nil
}

// isHiddenWebChatCatalogModelID removes product-hidden choices only from the
// Web Chat picker. Authorization keeps accepting these requested IDs so saved
// conversations and direct historical requests remain compatible.
func isHiddenWebChatCatalogModelID(model string) bool {
	normalized := canonicalizeOpenAIModelAliasSpelling(model)
	switch normalized {
	case "gpt-5.6", "gpt-5.3-codex-spark":
		return true
	}

	const family = "gpt-5.4"
	if normalized == family {
		return true
	}
	if !strings.HasPrefix(normalized, family) || len(normalized) == len(family) {
		return false
	}
	next := normalized[len(family)]
	return (next < 'a' || next > 'z') && (next < '0' || next > '9')
}

func (s *ChatService) ResolveTranscriptionPrincipal(ctx context.Context, userID int64) (*APIKey, error) {
	group, err := s.resolveTranscriptionGroup(ctx, userID)
	if err != nil {
		return nil, err
	}
	if s.principals == nil {
		return nil, fmt.Errorf("chat principal resolver is unavailable")
	}
	return s.principals.Resolve(ctx, userID, group, nil)
}

func (s *ChatService) transcriptionCapability(ctx context.Context, userID int64) ChatTranscriptionCapability {
	cfg, err := s.effectiveTranscriptionConfig(ctx)
	if err != nil {
		cfg = s.transcribe
		cfg.Enabled = false
	}
	capability := ChatTranscriptionCapability{
		Enabled:            false,
		BillingMode:        "subsidized",
		MaxUploadBytes:     cfg.MaxUploadBytes,
		MaxDurationSeconds: cfg.MaxDurationSeconds,
		AcceptedMIMETypes:  append([]string(nil), cfg.AcceptedMIMETypes...),
	}
	if !cfg.Enabled {
		return capability
	}
	if _, err := s.resolveTranscriptionGroupWithConfig(ctx, userID, cfg); err == nil {
		capability.Enabled = true
	}
	return capability
}

func (s *ChatService) resolveTranscriptionGroup(ctx context.Context, userID int64) (*Group, error) {
	if s == nil {
		return nil, ErrChatTranscriptionOff
	}
	cfg, err := s.effectiveTranscriptionConfig(ctx)
	if err != nil {
		return nil, chatCatalogUnavailable(fmt.Errorf("load transcription settings: %w", err))
	}
	return s.resolveTranscriptionGroupWithConfig(ctx, userID, cfg)
}

func (s *ChatService) effectiveTranscriptionConfig(ctx context.Context) (config.TranscriptionConfig, error) {
	if s != nil {
		if provider, ok := s.scheduler.(chatTranscriptionConfigProvider); ok && provider != nil {
			return provider.EffectiveTranscriptionConfig(ctx)
		}
		return cloneTranscriptionConfig(s.transcribe), nil
	}
	return config.TranscriptionConfig{}, nil
}

func (s *ChatService) resolveTranscriptionGroupWithConfig(ctx context.Context, userID int64, cfg config.TranscriptionConfig) (*Group, error) {
	if s == nil || !cfg.Enabled {
		return nil, ErrChatTranscriptionOff
	}
	if s.users == nil || s.groups == nil || s.billing == nil || s.principals == nil {
		return nil, ErrChatTranscriptionAbsent
	}
	scheduler, ok := s.scheduler.(chatTranscriptionSchedulability)
	if !ok || scheduler == nil {
		return nil, ErrChatTranscriptionAbsent
	}
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, err
		}
		return nil, chatCatalogUnavailable(fmt.Errorf("load transcription user: %w", err))
	}
	if user == nil || !user.IsActive() {
		return nil, ErrUserNotFound
	}
	if _, err := s.billing.PeekWebChatEligibility(ctx, userID, PlatformOpenAI); err != nil {
		return nil, normalizeChatBillingError(err)
	}
	availableGroups, err := s.groups.GetAvailableGroups(ctx, userID)
	if err != nil {
		return nil, chatCatalogUnavailable(fmt.Errorf("load transcription groups: %w", err))
	}
	byID := make(map[int64]*Group, len(availableGroups))
	for i := range availableGroups {
		group := &availableGroups[i]
		byID[group.ID] = group
	}
	model := strings.TrimSpace(cfg.Model)
	for _, groupID := range cfg.GroupIDs {
		group := byID[groupID]
		if group == nil || !group.IsActive() || group.Platform != PlatformOpenAI || group.IsSubscriptionType() {
			continue
		}
		schedulable, schedErr := scheduler.HasSchedulableTranscriptionAccount(ctx, group.ID, model)
		if schedErr != nil {
			return nil, chatCatalogUnavailable(schedErr)
		}
		if schedulable {
			return group, nil
		}
	}
	return nil, ErrChatTranscriptionAbsent
}

func (s *ChatService) ResolvePrincipal(ctx context.Context, userID int64, model string) (*ChatPrincipal, error) {
	_, _, choices, err := s.authorizedModelChoices(ctx, userID)
	if err != nil {
		return nil, err
	}
	model = strings.TrimSpace(model)
	for i := range choices {
		if choices[i].model.ID != model {
			continue
		}
		if s.principals == nil {
			return nil, fmt.Errorf("chat principal resolver is unavailable")
		}
		principal, err := s.principals.Resolve(ctx, userID, choices[i].group, choices[i].subscription)
		if err != nil {
			return nil, err
		}
		return &ChatPrincipal{APIKey: principal, Subscription: choices[i].subscription}, nil
	}
	return nil, ErrChatModelNotAvailable
}

// ResolveWebChatPrincipal binds billing to the same capability-qualified group
// used for reasoning admission. This prevents a Pro-capable wallet group from
// being validated and then replaced by a higher-priority raw-CC group.
func (s *ChatService) ResolveWebChatPrincipal(
	ctx context.Context,
	userID int64,
	model string,
	options WebChatReasoningOptions,
) (*ChatPrincipal, error) {
	_, _, choices, err := s.authorizedModelChoicesForReasoning(ctx, userID, &options)
	if err != nil {
		return nil, err
	}
	model = strings.TrimSpace(model)
	for i := range choices {
		if choices[i].model.ID != model {
			continue
		}
		if s.principals == nil {
			return nil, fmt.Errorf("chat principal resolver is unavailable")
		}
		principal, resolveErr := s.principals.Resolve(ctx, userID, choices[i].group, choices[i].subscription)
		if resolveErr != nil {
			return nil, resolveErr
		}
		return &ChatPrincipal{APIKey: principal, Subscription: choices[i].subscription}, nil
	}
	if mode, valid := NormalizeWebChatReasoningMode(options.Mode); valid && mode == WebChatReasoningModePro {
		return nil, ErrChatProReasoningUnavailable
	}
	return nil, ErrChatModelNotAvailable
}

// NormalizeReasoningEffort validates Web Chat's reasoning contract against the
// authorized model's effective channel mapping. Mapping lookup stays strict so
// an unavailable or ambiguous alias never silently bypasses model constraints.
func (s *ChatService) NormalizeReasoningEffort(ctx context.Context, userID int64, model, effort string) (string, error) {
	effort = strings.ToLower(strings.TrimSpace(effort))
	if effort == "" {
		effort = "low"
	}
	switch effort {
	case "low", "medium", "high", "xhigh", "max":
	default:
		return "", ErrChatReasoningNotAllowed
	}

	_, _, choices, err := s.authorizedModelChoices(ctx, userID)
	if err != nil {
		return "", err
	}
	model = strings.TrimSpace(model)
	for i := range choices {
		if choices[i].model.ID != model {
			continue
		}
		mapping, mappingErr := s.catalog.ResolveChannelMappingStrict(ctx, choices[i].group.ID, model)
		if mappingErr != nil {
			return "", chatCatalogUnavailable(fmt.Errorf("resolve chat reasoning model mapping: %w", mappingErr))
		}
		effectiveModel := strings.TrimSpace(mapping.MappedModel)
		if effectiveModel == "" {
			return "", ErrChatModelNotAvailable
		}
		if !webChatReasoningEffortAllowedForMapping(mapping, effort) {
			return "", ErrChatReasoningNotAllowed
		}
		return effort, nil
	}
	return "", ErrChatModelNotAvailable
}

// NormalizeWebChatReasoning validates mode and effort as separate dimensions,
// then proves that the authorized model has at least one currently schedulable
// native Responses account. Pro is never rewritten to standard.
func (s *ChatService) NormalizeWebChatReasoning(
	ctx context.Context,
	userID int64,
	model string,
	options WebChatReasoningOptions,
) (WebChatReasoningOptions, error) {
	mode, valid := NormalizeWebChatReasoningMode(options.Mode)
	if !valid {
		return WebChatReasoningOptions{}, ErrChatReasoningModeInvalid
	}
	effort := strings.ToLower(strings.TrimSpace(options.Effort))
	if effort == "" {
		effort = "low"
	}
	switch effort {
	case "low", "medium", "high", "xhigh", "max":
	default:
		return WebChatReasoningOptions{}, ErrChatReasoningNotAllowed
	}
	options = WebChatReasoningOptions{Mode: mode, Effort: effort}

	_, _, choices, err := s.authorizedModelChoicesForReasoning(ctx, userID, &options)
	if err != nil {
		return WebChatReasoningOptions{}, err
	}
	model = strings.TrimSpace(model)
	for i := range choices {
		if choices[i].model.ID != model {
			continue
		}
		mapping, mappingErr := s.catalog.ResolveChannelMappingStrict(ctx, choices[i].group.ID, model)
		if mappingErr != nil {
			return WebChatReasoningOptions{}, chatCatalogUnavailable(fmt.Errorf("resolve chat reasoning mode mapping: %w", mappingErr))
		}
		capability, known := openai.DefaultModelByID(mapping.MappedModel)
		if !known || !capability.SupportsResponses || !capability.SupportsReasoningSummary {
			if mode == WebChatReasoningModePro {
				return WebChatReasoningOptions{}, ErrChatProReasoningUnavailable
			}
			return WebChatReasoningOptions{}, ErrChatModelNotAvailable
		}
		if !capability.SupportsReasoningEffort(effort) {
			return WebChatReasoningOptions{}, ErrChatReasoningNotAllowed
		}
		if !webChatReasoningEffortAllowedForMapping(mapping, effort) {
			return WebChatReasoningOptions{}, ErrChatReasoningNotAllowed
		}
		if mode == WebChatReasoningModePro && !capability.SupportsReasoningProMode {
			return WebChatReasoningOptions{}, ErrChatProReasoningUnavailable
		}

		schedulable, scheduleErr := s.scheduler.HasSchedulableWebChatReasoningAccount(
			ctx,
			choices[i].group.ID,
			model,
			options,
		)
		if scheduleErr != nil {
			return WebChatReasoningOptions{}, chatCatalogUnavailable(scheduleErr)
		}
		if !schedulable {
			if mode == WebChatReasoningModePro {
				return WebChatReasoningOptions{}, ErrChatProReasoningUnavailable
			}
			return WebChatReasoningOptions{}, ErrChatModelNotAvailable
		}
		return options, nil
	}
	if mode == WebChatReasoningModePro {
		return WebChatReasoningOptions{}, ErrChatProReasoningUnavailable
	}
	return WebChatReasoningOptions{}, ErrChatModelNotAvailable
}

// webChatReasoningEffortAllowedForMapping is shared by request admission and
// the authenticated model catalog. Keeping this rule in one place prevents
// the UI from advertising a mode/effort combination that the POST path will
// immediately reject.
func webChatReasoningEffortAllowedForMapping(mapping ChannelMappingResult, effort string) bool {
	if !strings.EqualFold(strings.TrimSpace(effort), "max") {
		return true
	}
	// Upstream-billed channels may apply an account-level credentials mapping
	// after the channel mapping. The catalog cannot prove every eligible account
	// stays off Sol, so max fails closed for the same reason as request admission.
	return mapping.BillingModelSource != BillingModelSourceUpstream &&
		normalizeKnownOpenAICodexModel(mapping.MappedModel) != "gpt-5.6-sol"
}

func isWebChatCatalogReasoningEffort(effort string) bool {
	switch strings.ToLower(strings.TrimSpace(effort)) {
	case "low", "medium", "high", "xhigh":
		return true
	default:
		return false
	}
}

func (s *ChatService) SupportsVision(ctx context.Context, userID int64, model string) (bool, error) {
	_, _, choices, err := s.authorizedModelChoices(ctx, userID)
	if err != nil {
		return false, err
	}
	model = strings.TrimSpace(model)
	for i := range choices {
		if choices[i].model.ID == model {
			return choices[i].model.SupportsVision, nil
		}
	}
	return false, ErrChatModelNotAvailable
}

type chatModelChoice struct {
	model        ChatModel
	group        *Group
	subscription *UserSubscription
}

func (s *ChatService) authorizedModelChoices(ctx context.Context, userID int64) (*User, float64, []chatModelChoice, error) {
	return s.authorizedModelChoicesForReasoning(ctx, userID, nil)
}

func (s *ChatService) authorizedModelChoicesForReasoning(
	ctx context.Context,
	userID int64,
	reasoning *WebChatReasoningOptions,
) (*User, float64, []chatModelChoice, error) {
	if s == nil || s.users == nil || s.groups == nil || s.catalog == nil || s.scheduler == nil || s.pricing == nil {
		return nil, 0, nil, fmt.Errorf("chat model service is unavailable")
	}
	if s.billing == nil {
		return nil, 0, nil, ErrBillingServiceUnavailable
	}
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, 0, nil, err
		}
		return nil, 0, nil, chatCatalogUnavailable(fmt.Errorf("load chat user: %w", err))
	}
	if user == nil || !user.IsActive() {
		return nil, 0, nil, ErrUserNotFound
	}
	balance, walletErr := s.billing.PeekWebChatEligibility(ctx, userID, PlatformOpenAI)
	var walletTechnicalErr error
	if walletErr != nil && !isChatBillingSourceUnavailable(walletErr) {
		walletTechnicalErr = walletErr
	}
	groups, err := s.groups.GetAvailableGroups(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, 0, nil, err
		}
		return nil, 0, nil, chatCatalogUnavailable(fmt.Errorf("load chat groups: %w", err))
	}
	sort.SliceStable(groups, func(i, j int) bool {
		leftSubscription := groups[i].IsSubscriptionType()
		rightSubscription := groups[j].IsSubscriptionType()
		if leftSubscription != rightSubscription {
			return leftSubscription
		}
		if groups[i].SortOrder == groups[j].SortOrder {
			return groups[i].ID < groups[j].ID
		}
		return groups[i].SortOrder < groups[j].SortOrder
	})

	choicesByModel := make(map[string]chatModelChoice)
	hadBillableCandidate := false
	hadUnavailableSource := false
	hadEligibleSubscription := false
	var subscriptionTechnicalErr error
	for i := range groups {
		group := &groups[i]
		if !group.IsActive() || group.Platform != PlatformOpenAI {
			continue
		}
		candidates, err := s.candidateModelsForGroup(ctx, group)
		if err != nil {
			return nil, 0, nil, chatCatalogUnavailable(err)
		}
		models := make([]ChatModel, 0, len(candidates))
		for _, model := range candidates {
			if !isTextGPTChatModel(model) {
				continue
			}
			chatModel, ok, err := s.buildChatModel(ctx, userID, group, model)
			if err != nil {
				return nil, 0, nil, chatCatalogUnavailable(err)
			}
			if !ok {
				continue
			}
			if reasoning != nil && !chatModelSupportsReasoningOptions(chatModel, *reasoning) {
				continue
			}
			models = append(models, chatModel)
		}
		if len(models) == 0 {
			continue
		}
		hadBillableCandidate = true

		var subscription *UserSubscription
		if group.IsSubscriptionType() {
			subscription, err = s.resolveEligibleChatSubscription(ctx, userID, group)
			if err != nil {
				if isChatBillingSourceUnavailable(err) {
					hadUnavailableSource = true
					continue
				}
				if subscriptionTechnicalErr == nil {
					subscriptionTechnicalErr = err
				}
				continue
			}
			hadEligibleSubscription = true
		} else if walletErr != nil {
			hadUnavailableSource = true
			continue
		}

		for _, model := range models {
			key := strings.ToLower(model.ID)
			if _, exists := choicesByModel[key]; exists {
				continue
			}
			choicesByModel[key] = chatModelChoice{
				model:        model,
				group:        group,
				subscription: subscription,
			}
		}
	}
	if !hadEligibleSubscription && subscriptionTechnicalErr != nil && hadBillableCandidate {
		return nil, balance, nil, normalizeChatBillingError(subscriptionTechnicalErr)
	}
	if len(choicesByModel) == 0 && walletTechnicalErr != nil && hadBillableCandidate {
		return nil, balance, nil, normalizeChatBillingError(walletTechnicalErr)
	}
	if len(choicesByModel) == 0 && hadBillableCandidate && hadUnavailableSource {
		return nil, balance, nil, ErrChatInsufficientBalance
	}

	choices := make([]chatModelChoice, 0, len(choicesByModel))
	for _, choice := range choicesByModel {
		choices = append(choices, choice)
	}
	sort.Slice(choices, func(i, j int) bool {
		leftPreferred := strings.EqualFold(choices[i].model.ID, preferredChatModelID)
		rightPreferred := strings.EqualFold(choices[j].model.ID, preferredChatModelID)
		if leftPreferred != rightPreferred {
			return leftPreferred
		}
		return choices[i].model.ID < choices[j].model.ID
	})
	return user, balance, choices, nil
}

func chatModelSupportsReasoningOptions(model ChatModel, options WebChatReasoningOptions) bool {
	mode, valid := NormalizeWebChatReasoningMode(options.Mode)
	if !valid || !model.SupportsResponses || !model.SupportsReasoningSummary {
		return false
	}
	effortSupported := false
	for _, effort := range model.SupportedReasoningEfforts {
		if strings.EqualFold(strings.TrimSpace(effort), strings.TrimSpace(options.Effort)) {
			effortSupported = true
			break
		}
	}
	if !effortSupported {
		return false
	}
	return mode != WebChatReasoningModePro || model.SupportsReasoningProMode
}

func (s *ChatService) resolveEligibleChatSubscription(
	ctx context.Context,
	userID int64,
	group *Group,
) (*UserSubscription, error) {
	if s == nil || s.subs == nil || s.billing == nil || group == nil || !group.IsSubscriptionType() {
		return nil, ErrBillingServiceUnavailable
	}
	subscription, err := s.subs.GetActiveSubscription(ctx, userID, group.ID)
	if err != nil {
		return nil, err
	}
	if subscription == nil || subscription.UserID != userID || subscription.GroupID != group.ID {
		return nil, ErrSubscriptionInvalid
	}
	needsMaintenance, err := s.subs.ValidateAndCheckLimits(subscription, group)
	if err != nil {
		return nil, err
	}
	if needsMaintenance {
		subscription, err = s.subs.EnsureWindowMaintenance(ctx, subscription)
		if err != nil {
			return nil, ErrBillingServiceUnavailable.WithCause(err)
		}
		needsMaintenance, err = s.subs.ValidateAndCheckLimits(subscription, group)
		if err != nil {
			return nil, err
		}
		if needsMaintenance {
			return nil, ErrBillingServiceUnavailable.WithCause(errors.New("subscription windows remain stale after maintenance"))
		}
	}
	if err := s.billing.PeekWebChatSubscriptionEligibility(ctx, userID, group, subscription); err != nil {
		return nil, err
	}
	return subscription, nil
}

func isChatBillingSourceUnavailable(err error) bool {
	return errors.Is(err, ErrInsufficientBalance) ||
		errors.Is(err, ErrChatInsufficientBalance) ||
		errors.Is(err, ErrSubscriptionNotFound) ||
		errors.Is(err, ErrSubscriptionExpired) ||
		errors.Is(err, ErrSubscriptionSuspended) ||
		errors.Is(err, ErrSubscriptionInvalid) ||
		errors.Is(err, ErrWeeklyLimitExceeded) ||
		errors.Is(err, ErrMonthlyLimitExceeded) ||
		errors.Is(err, ErrUserPlatformDailyQuotaExhausted) ||
		errors.Is(err, ErrUserPlatformWeeklyQuotaExhausted) ||
		errors.Is(err, ErrUserPlatformMonthlyQuotaExhausted)
}

func (s *ChatService) candidateModelsForGroup(ctx context.Context, group *Group) ([]string, error) {
	available, err := s.catalog.GetAvailableModelsStrict(ctx, &group.ID, PlatformOpenAI)
	if err != nil {
		return nil, err
	}
	if group.CustomModelsListEnabled() {
		// The explicit group list is the candidate source. The per-model
		// scheduler check below still enforces account mappings and runtime state.
		return normalizedChatModelList(group.ModelsListConfig.Models), nil
	}
	return normalizedChatModelList(available), nil
}

func (s *ChatService) buildChatModel(ctx context.Context, userID int64, group *Group, requestedModel string) (ChatModel, bool, error) {
	if !isTextGPTChatModel(requestedModel) {
		return ChatModel{}, false, nil
	}
	mapping, err := s.catalog.ResolveChannelMappingStrict(ctx, group.ID, requestedModel)
	if err != nil {
		return ChatModel{}, false, err
	}
	if !isTextGPTChatModel(mapping.MappedModel) {
		return ChatModel{}, false, nil
	}
	schedulable, err := s.scheduler.HasSchedulableChatCompletionsAccount(ctx, group.ID, requestedModel)
	if err != nil {
		return ChatModel{}, false, err
	}
	if !schedulable {
		return ChatModel{}, false, nil
	}
	billingModel := requestedModel
	switch mapping.BillingModelSource {
	case BillingModelSourceRequested:
		billingModel = requestedModel
	case BillingModelSourceChannelMapped, "":
		billingModel = mapping.MappedModel
	case BillingModelSourceUpstream:
		// Account-specific upstream mapping cannot be selected on a GET without
		// consuming scheduler state. The channel mapping is the narrowest safe
		// pricing candidate available at catalog time.
		billingModel = mapping.MappedModel
	}
	if strings.TrimSpace(billingModel) == "" {
		return ChatModel{}, false, nil
	}
	restricted, err := s.catalog.IsModelRestrictedStrict(ctx, group.ID, billingModel)
	if err != nil {
		return ChatModel{}, false, err
	}
	if restricted {
		return ChatModel{}, false, nil
	}

	resolved := s.pricing.Resolve(ctx, PricingInput{Model: billingModel, GroupID: &group.ID})
	if resolved == nil {
		return ChatModel{}, false, nil
	}
	multiplier := s.catalog.ResolveUserGroupRateMultiplier(ctx, userID, group.ID, group.RateMultiplier)
	if multiplier < 0 {
		return ChatModel{}, false, nil
	}

	price := ChatModelPricing{BillingMode: string(resolved.Mode), RateMultiplier: multiplier}
	switch resolved.Mode {
	case "", BillingModeToken:
		tokenPricing := s.pricing.GetIntervalPricing(resolved, 0)
		if tokenPricing == nil {
			return ChatModel{}, false, nil
		}
		price.BillingMode = string(BillingModeToken)
		price.Unit = "usd_per_million_tokens"
		price.InputPrice = tokenPricing.InputPricePerToken * 1_000_000 * multiplier
		price.OutputPrice = tokenPricing.OutputPricePerToken * 1_000_000 * multiplier
	case BillingModePerRequest:
		if resolved.Source == "" {
			return ChatModel{}, false, nil
		}
		price.Unit = "usd_per_request"
		price.PerRequestPrice = resolved.DefaultPerRequestPrice * multiplier
	default:
		return ChatModel{}, false, nil
	}

	supportsVision := false
	if capabilities, ok := s.pricing.(interface{ SupportsVision(string) bool }); ok {
		supportsVision = capabilities.SupportsVision(billingModel)
	}
	modelCapability, knownCapability := openai.DefaultModelByID(mapping.MappedModel)
	supportedReasoningEfforts := make([]string, 0)
	supportsResponses := false
	supportsReasoningSummary := false
	supportsReasoningProMode := false
	if knownCapability && modelCapability.SupportsResponses && modelCapability.SupportsReasoningSummary {
		seenEfforts := make(map[string]struct{}, len(modelCapability.SupportedReasoningEfforts))
		for _, advertisedEffort := range modelCapability.SupportedReasoningEfforts {
			effort := strings.ToLower(strings.TrimSpace(advertisedEffort))
			if !isWebChatCatalogReasoningEffort(effort) ||
				!webChatReasoningEffortAllowedForMapping(mapping, effort) {
				continue
			}
			if _, duplicate := seenEfforts[effort]; duplicate {
				continue
			}
			seenEfforts[effort] = struct{}{}
			standardSchedulable, scheduleErr := s.scheduler.HasSchedulableWebChatReasoningAccount(
				ctx,
				group.ID,
				requestedModel,
				WebChatReasoningOptions{Mode: WebChatReasoningModeStandard, Effort: effort},
			)
			if scheduleErr != nil {
				return ChatModel{}, false, scheduleErr
			}
			if standardSchedulable {
				supportedReasoningEfforts = append(supportedReasoningEfforts, effort)
			}
		}
		supportsResponses = len(supportedReasoningEfforts) > 0
		supportsReasoningSummary = supportsResponses
		if modelCapability.SupportsReasoningProMode && supportsReasoningSummary {
			// The existing catalog schema exposes one effort list shared by both
			// modes. Advertise Pro only when every visible effort remains
			// schedulable in Pro, so the picker cannot form an invalid pair.
			supportsReasoningProMode = true
			for _, effort := range supportedReasoningEfforts {
				proSchedulable, scheduleErr := s.scheduler.HasSchedulableWebChatReasoningAccount(
					ctx,
					group.ID,
					requestedModel,
					WebChatReasoningOptions{Mode: WebChatReasoningModePro, Effort: effort},
				)
				if scheduleErr != nil {
					return ChatModel{}, false, scheduleErr
				}
				if !proSchedulable {
					supportsReasoningProMode = false
					break
				}
			}
		}
	}
	return ChatModel{
		ID:                        requestedModel,
		DisplayName:               chatModelDisplayName(requestedModel),
		InputPrice:                price.InputPrice,
		OutputPrice:               price.OutputPrice,
		Pricing:                   price,
		SupportsVision:            supportsVision,
		SupportsReasoningSlider:   len(supportedReasoningEfforts) > 0,
		SupportsResponses:         supportsResponses,
		SupportsReasoningSummary:  supportsReasoningSummary,
		SupportsReasoningProMode:  supportsReasoningProMode,
		SupportedReasoningEfforts: supportedReasoningEfforts,
	}, true, nil
}

// HasSchedulableChatCompletionsAccount is a read-only catalog check. The real
// request still goes through the scheduler and repeats all runtime checks.
func (s *OpenAIGatewayService) HasSchedulableChatCompletionsAccount(ctx context.Context, groupID int64, model string) (bool, error) {
	return s.hasSchedulableWebChatAccount(ctx, groupID, model, nil)
}

// HasSchedulableWebChatReasoningAccount is a fail-closed read-only admission
// check for reasoning summaries. Runtime selection must repeat the same
// AccountSupportsWebChatReasoning predicate before forwarding.
func (s *OpenAIGatewayService) HasSchedulableWebChatReasoningAccount(
	ctx context.Context,
	groupID int64,
	model string,
	options WebChatReasoningOptions,
) (bool, error) {
	return s.hasSchedulableWebChatAccount(ctx, groupID, model, &options)
}

func (s *OpenAIGatewayService) hasSchedulableWebChatAccount(
	ctx context.Context,
	groupID int64,
	model string,
	reasoning *WebChatReasoningOptions,
) (bool, error) {
	if s == nil || (s.schedulerSnapshot == nil && s.accountRepo == nil) {
		return false, errors.New("OpenAI scheduler is unavailable")
	}
	if s.channelService == nil {
		return false, errors.New("channel service is unavailable")
	}
	model = strings.TrimSpace(model)
	if groupID <= 0 || !s.isWebChatTextGPTModel(model) {
		return false, nil
	}

	ctx = s.withOpenAIQuotaAutoPauseContext(ctx)
	accounts, err := s.listSchedulableAccounts(ctx, &groupID, PlatformOpenAI)
	if err != nil {
		return false, err
	}

	mapping, err := s.channelService.ResolveChannelMappingStrict(ctx, groupID, model)
	if err != nil {
		return false, err
	}
	forwardedModel := strings.TrimSpace(mapping.MappedModel)
	if !s.isWebChatTextGPTModel(forwardedModel) {
		return false, nil
	}
	if billingModel := billingModelForRestriction(mapping.BillingModelSource, model, mapping.MappedModel); billingModel != "" {
		restricted, restrictErr := s.channelService.IsModelRestrictedStrict(ctx, groupID, billingModel)
		if restrictErr != nil {
			return false, restrictErr
		}
		if restricted {
			return false, nil
		}
	}

	parentCache := make(map[int64]*Account)
	parentLoaded := make(map[int64]struct{})
	var parentLookupErr error
	hasSchedulableAccount := false
	requiredCapability := OpenAIEndpointCapabilityChatCompletions
	if reasoning != nil {
		requiredCapability = OpenAIEndpointCapabilityResponses
	}
	for i := range accounts {
		account := &accounts[i]
		if !isOpenAICompatibleAccountEligibleForRequest(
			ctx,
			account,
			PlatformOpenAI,
			model,
			false,
			requiredCapability,
		) || s.isOpenAIAccountRequestRuntimeBlocked(account, model) {
			continue
		}

		if account.IsShadow() {
			parentID := *account.ParentAccountID
			parent, loaded := parentCache[parentID]
			if _, ok := parentLoaded[parentID]; !ok {
				parent, err = s.lookupChatShadowParent(ctx, parentID)
				if err != nil {
					parentLookupErr = err
					continue
				}
				parentCache[parentID] = parent
				parentLoaded[parentID] = struct{}{}
			}
			if !loaded {
				parent = parentCache[parentID]
			}
			if !parentHealthyForShadow(account, func(int64) *Account { return parent }) {
				continue
			}
		}

		if mapping.BillingModelSource == BillingModelSourceUpstream {
			runtimeRestrictionModel := resolveOpenAIAccountUpstreamModelForRequest(account, model, false)
			restricted, restrictErr := s.channelService.IsModelRestrictedStrict(ctx, groupID, runtimeRestrictionModel)
			if restrictErr != nil {
				return false, restrictErr
			}
			if restricted {
				continue
			}
		}

		accountMappedModel := resolveOpenAIAccountUpstreamModelForRequest(account, forwardedModel, false)
		upstreamModel := normalizeOpenAIModelForUpstream(account, accountMappedModel)
		if !s.isWebChatTextGPTModel(accountMappedModel) || !s.isWebChatTextGPTModel(upstreamModel) {
			if reasoning != nil {
				continue
			}
			// The runtime scheduler does not know about the Web Chat capability
			// boundary. If any currently selectable account can turn this model
			// into a dedicated non-text model, reject the whole alias.
			return false, nil
		}
		if reasoning != nil && !s.AccountSupportsWebChatReasoningForModel(account, forwardedModel, *reasoning) {
			continue
		}
		hasSchedulableAccount = true
	}
	if parentLookupErr != nil {
		return false, parentLookupErr
	}
	return hasSchedulableAccount, nil
}

// HasSchedulableTranscriptionAccount performs a read-only admission check for
// the Web Chat transcription group selector. The real request repeats the
// scheduler decision and acquires account concurrency immediately before the
// upload is sent.
func (s *OpenAIGatewayService) HasSchedulableTranscriptionAccount(ctx context.Context, groupID int64, model string) (bool, error) {
	if s == nil || (s.schedulerSnapshot == nil && s.accountRepo == nil) {
		return false, errors.New("OpenAI scheduler is unavailable")
	}
	if s.channelService == nil {
		return false, errors.New("channel service is unavailable")
	}
	model = strings.TrimSpace(model)
	if groupID <= 0 || model == "" {
		return false, nil
	}

	ctx = s.withOpenAIQuotaAutoPauseContext(ctx)
	mapping, err := s.channelService.ResolveChannelMappingStrict(ctx, groupID, model)
	if err != nil {
		return false, err
	}
	forwardedModel := strings.TrimSpace(mapping.MappedModel)
	if forwardedModel == "" {
		return false, nil
	}
	if billingModel := billingModelForRestriction(mapping.BillingModelSource, model, forwardedModel); billingModel != "" {
		restricted, restrictErr := s.channelService.IsModelRestrictedStrict(ctx, groupID, billingModel)
		if restrictErr != nil {
			return false, restrictErr
		}
		if restricted {
			return false, nil
		}
	}
	accounts, err := s.listSchedulableAccounts(ctx, &groupID, PlatformOpenAI)
	if err != nil {
		return false, err
	}
	for i := range accounts {
		account := &accounts[i]
		if account.Type != AccountTypeAPIKey || s.isOpenAIAccountRequestRuntimeBlocked(account, model) {
			continue
		}
		if !isOpenAICompatibleAccountEligibleForRequest(
			ctx,
			account,
			PlatformOpenAI,
			model,
			false,
			OpenAIEndpointCapabilityAudioTranscriptions,
		) {
			continue
		}
		if mapping.BillingModelSource == BillingModelSourceUpstream {
			upstreamModel := resolveOpenAIForwardModel(account, model, forwardedModel)
			upstreamModel = normalizeOpenAIModelForUpstream(account, upstreamModel)
			restricted, restrictErr := s.channelService.IsModelRestrictedStrict(ctx, groupID, upstreamModel)
			if restrictErr != nil {
				return false, restrictErr
			}
			if restricted {
				continue
			}
		}
		return true, nil
	}
	return false, nil
}

func (s *OpenAIGatewayService) lookupChatShadowParent(ctx context.Context, parentID int64) (*Account, error) {
	var snapshotErr error
	if s.schedulerSnapshot != nil {
		parent, err := s.schedulerSnapshot.GetAccount(ctx, parentID)
		if err == nil && parent != nil {
			return parent, nil
		}
		snapshotErr = err
	}
	if s.accountRepo != nil {
		parent, err := s.accountRepo.GetByID(ctx, parentID)
		if errors.Is(err, ErrAccountNotFound) {
			return nil, nil
		}
		if err != nil {
			return nil, fmt.Errorf("load shadow parent account: %w", err)
		}
		return parent, nil
	}
	if snapshotErr != nil {
		return nil, snapshotErr
	}
	return nil, nil
}

func chatCatalogUnavailable(err error) error {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, ErrChatCatalogUnavailable) {
		return err
	}
	return ErrChatCatalogUnavailable.WithCause(err)
}

func normalizeChatBillingError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrInsufficientBalance) || errors.Is(err, ErrChatInsufficientBalance) {
		return ErrChatInsufficientBalance.WithCause(err)
	}
	if infraerrors.IsServiceUnavailable(err) || infraerrors.IsTooManyRequests(err) || infraerrors.IsForbidden(err) {
		return err
	}
	return ErrBillingServiceUnavailable.WithCause(err)
}

func normalizedChatModelList(models []string) []string {
	seen := make(map[string]struct{}, len(models))
	out := make([]string, 0, len(models))
	for _, model := range models {
		model = strings.TrimSpace(model)
		key := strings.ToLower(model)
		if model == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, model)
	}
	return out
}

var webChatDedicatedCapabilityTokens = map[string]struct{}{
	"audio":           {},
	"embedding":       {},
	"embeddings":      {},
	"image":           {},
	"images":          {},
	"imagegen":        {},
	"imagegeneration": {},
	"moderation":      {},
	"moderations":     {},
	"realtime":        {},
	"search":          {},
	"speech":          {},
	"transcribe":      {},
	"transcription":   {},
	"transcriptions":  {},
	"tts":             {},
	"video":           {},
	"videos":          {},
	"websearch":       {},
}

// isTextGPTChatModel is the centralized name-level fallback for Web Chat.
// Exact model metadata is not guaranteed for private aliases, so each routing
// stage must also satisfy this conservative family policy.
func isTextGPTChatModel(model string) bool {
	model = strings.ToLower(lastOpenAIModelSegment(model))
	if isRetiredGPT52ChatModel(model) || !strings.HasPrefix(model, "gpt-") || strings.Contains(model, "*") || IsGPTImageGenerationModel(model) {
		return false
	}
	for _, token := range strings.FieldsFunc(model, func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9')
	}) {
		if _, dedicated := webChatDedicatedCapabilityTokens[token]; dedicated {
			return false
		}
	}
	return true
}

func isRetiredGPT52ChatModel(model string) bool {
	const family = "gpt-5.2"
	if model == family {
		return true
	}
	if !strings.HasPrefix(model, family) || len(model) == len(family) {
		return false
	}
	next := model[len(family)]
	return (next < 'a' || next > 'z') && (next < '0' || next > '9')
}

// isWebChatTextGPTModel augments the name policy with exact LiteLLM metadata
// when it is available. Metadata mode is reliable for dedicated image/audio/
// completion models, while private GPT aliases deliberately fall back to the
// centralized name policy above.
func (s *OpenAIGatewayService) isWebChatTextGPTModel(model string) bool {
	if !isTextGPTChatModel(model) {
		return false
	}
	if s == nil || s.billingService == nil || s.billingService.pricingService == nil {
		return true
	}
	_, metadata := s.billingService.pricingService.GetExactModelPricing(model)
	if metadata == nil {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(metadata.Mode)) {
	case "chat", "responses":
		return !metadata.SupportsAudioInput && !metadata.SupportsAudioOutput
	default:
		return false
	}
}

func chatModelDisplayName(model string) string {
	model = strings.TrimSpace(model)
	for _, candidate := range openai.DefaultModels {
		if strings.EqualFold(candidate.ID, model) && strings.TrimSpace(candidate.DisplayName) != "" {
			return candidate.DisplayName
		}
	}
	return model
}
