package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
)

var (
	ErrChatInsufficientBalance = infraerrors.Forbidden("INSUFFICIENT_BALANCE", "Insufficient account balance")
	ErrChatModelNotAvailable   = infraerrors.BadRequest("CHAT_MODEL_NOT_AVAILABLE", "The selected model is not available")
	ErrChatCatalogUnavailable  = infraerrors.ServiceUnavailable("CHAT_CATALOG_UNAVAILABLE", "Chat model catalog is temporarily unavailable")
)

const preferredChatModelID = "gpt-5.5"

type ChatModelPricing struct {
	BillingMode     string  `json:"billing_mode"`
	Unit            string  `json:"unit"`
	InputPrice      float64 `json:"input_price"`
	OutputPrice     float64 `json:"output_price"`
	PerRequestPrice float64 `json:"per_request_price,omitempty"`
	RateMultiplier  float64 `json:"rate_multiplier"`
}

type ChatModel struct {
	ID          string           `json:"id"`
	DisplayName string           `json:"display_name"`
	Recommended bool             `json:"recommended"`
	InputPrice  float64          `json:"input_price"`
	OutputPrice float64          `json:"output_price"`
	Pricing     ChatModelPricing `json:"pricing"`
}

type ChatModelsResult struct {
	Models  []ChatModel `json:"models"`
	Balance float64     `json:"balance"`
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
}

type chatPricingResolver interface {
	Resolve(ctx context.Context, input PricingInput) *ResolvedPricing
	GetIntervalPricing(resolved *ResolvedPricing, totalContextTokens int) *ModelPricing
}

type chatPrincipalProvider interface {
	Resolve(ctx context.Context, userID int64, group *Group) (*APIKey, error)
}

type chatBillingEligibility interface {
	PeekWebChatEligibility(ctx context.Context, userID int64, platform string) (float64, error)
}

type ChatService struct {
	users      chatUserReader
	groups     chatGroupAccess
	catalog    chatModelCatalog
	scheduler  chatModelSchedulability
	pricing    chatPricingResolver
	billing    chatBillingEligibility
	principals chatPrincipalProvider
}

func NewChatService(
	users UserRepository,
	groups *APIKeyService,
	catalog *GatewayService,
	scheduler *OpenAIGatewayService,
	pricing *ModelPricingResolver,
	billing *BillingCacheService,
	principals *ChatPrincipalResolver,
) *ChatService {
	return &ChatService{
		users:      users,
		groups:     groups,
		catalog:    catalog,
		scheduler:  scheduler,
		pricing:    pricing,
		billing:    billing,
		principals: principals,
	}
}

func (s *ChatService) ListModels(ctx context.Context, userID int64) (*ChatModelsResult, error) {
	_, balance, choices, err := s.authorizedModelChoices(ctx, userID)
	if err != nil {
		return nil, err
	}
	models := make([]ChatModel, len(choices))
	for i := range choices {
		models[i] = choices[i].model
		models[i].Recommended = strings.EqualFold(models[i].ID, preferredChatModelID)
	}
	return &ChatModelsResult{Models: models, Balance: balance}, nil
}

func (s *ChatService) ResolvePrincipal(ctx context.Context, userID int64, model string) (*APIKey, error) {
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
		return s.principals.Resolve(ctx, userID, choices[i].group)
	}
	return nil, ErrChatModelNotAvailable
}

type chatModelChoice struct {
	model ChatModel
	group *Group
}

func (s *ChatService) authorizedModelChoices(ctx context.Context, userID int64) (*User, float64, []chatModelChoice, error) {
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
	balance, err := s.billing.PeekWebChatEligibility(ctx, userID, PlatformOpenAI)
	if err != nil {
		return nil, 0, nil, normalizeChatBillingError(err)
	}
	groups, err := s.groups.GetAvailableGroups(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, 0, nil, err
		}
		return nil, 0, nil, chatCatalogUnavailable(fmt.Errorf("load chat groups: %w", err))
	}
	sort.SliceStable(groups, func(i, j int) bool {
		if groups[i].SortOrder == groups[j].SortOrder {
			return groups[i].ID < groups[j].ID
		}
		return groups[i].SortOrder < groups[j].SortOrder
	})

	choicesByModel := make(map[string]chatModelChoice)
	for i := range groups {
		group := &groups[i]
		if !group.IsActive() || group.Platform != PlatformOpenAI || group.IsSubscriptionType() {
			continue
		}
		candidates, err := s.candidateModelsForGroup(ctx, group)
		if err != nil {
			return nil, 0, nil, chatCatalogUnavailable(err)
		}
		for _, model := range candidates {
			if !isTextGPTChatModel(model) {
				continue
			}
			key := strings.ToLower(model)
			if _, exists := choicesByModel[key]; exists {
				continue
			}
			chatModel, ok, err := s.buildChatModel(ctx, userID, group, model)
			if err != nil {
				return nil, 0, nil, chatCatalogUnavailable(err)
			}
			if !ok {
				continue
			}
			choicesByModel[key] = chatModelChoice{model: chatModel, group: group}
		}
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

	return ChatModel{
		ID:          requestedModel,
		DisplayName: chatModelDisplayName(requestedModel),
		InputPrice:  price.InputPrice,
		OutputPrice: price.OutputPrice,
		Pricing:     price,
	}, true, nil
}

// HasSchedulableChatCompletionsAccount is a read-only catalog check. The real
// request still goes through the scheduler and repeats all runtime checks.
func (s *OpenAIGatewayService) HasSchedulableChatCompletionsAccount(ctx context.Context, groupID int64, model string) (bool, error) {
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
	for i := range accounts {
		account := &accounts[i]
		if !isOpenAICompatibleAccountEligibleForRequest(
			ctx,
			account,
			PlatformOpenAI,
			model,
			false,
			OpenAIEndpointCapabilityChatCompletions,
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
			// The runtime scheduler does not know about the Web Chat capability
			// boundary. If any currently selectable account can turn this model
			// into a dedicated non-text model, reject the whole alias.
			return false, nil
		}
		hasSchedulableAccount = true
	}
	if parentLookupErr != nil {
		return false, parentLookupErr
	}
	return hasSchedulableAccount, nil
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
