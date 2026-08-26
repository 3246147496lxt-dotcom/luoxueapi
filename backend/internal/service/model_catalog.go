package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
)

const (
	ModelCatalogStatusDraft     = domain.ModelCatalogStatusDraft
	ModelCatalogStatusPublished = domain.ModelCatalogStatusPublished
	ModelCatalogStatusArchived  = domain.ModelCatalogStatusArchived

	modelCatalogSnapshotTTL      = 30 * time.Second
	catalogPricingCurrencyCredit = "CREDIT"
)

var (
	ErrModelCatalogNotFound = infraerrors.NotFound("MODEL_CATALOG_NOT_FOUND", "model catalog entry not found")
	ErrModelCatalogExists   = infraerrors.Conflict("MODEL_CATALOG_EXISTS", "model catalog entry already exists")
	ErrModelCatalogInvalid  = infraerrors.BadRequest("MODEL_CATALOG_INVALID", "model catalog entry is invalid")
)

// ModelCatalogModel is curated display metadata. Public pricing is never stored
// on this entity; it is resolved from PublicGroupID and the group's active
// channel every time a snapshot is rebuilt.
type ModelCatalogModel struct {
	ID              int64      `json:"id"`
	Slug            string     `json:"slug"`
	Model           string     `json:"model"`
	Platform        string     `json:"platform"`
	MetadataModelID string     `json:"metadata_model_id"`
	DisplayNameZH   string     `json:"display_name_zh"`
	DisplayNameEN   string     `json:"display_name_en"`
	SummaryZH       string     `json:"summary_zh"`
	SummaryEN       string     `json:"summary_en"`
	Provider        string     `json:"provider"`
	LogoKey         string     `json:"logo_key"`
	Category        string     `json:"category"`
	Tags            []string   `json:"tags"`
	Capabilities    []string   `json:"capabilities"`
	ContextWindow   *int64     `json:"context_window"`
	MaxOutputTokens *int64     `json:"max_output_tokens"`
	PublicGroupID   *int64     `json:"public_group_id"`
	Status          string     `json:"status"`
	Featured        bool       `json:"featured"`
	SortOrder       int        `json:"sort_order"`
	PublishedAt     *time.Time `json:"published_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type ModelCatalogRepository interface {
	Create(ctx context.Context, model *ModelCatalogModel) error
	GetByID(ctx context.Context, id int64) (*ModelCatalogModel, error)
	Update(ctx context.Context, model *ModelCatalogModel) error
	ListAll(ctx context.Context) ([]ModelCatalogModel, error)
	ListPublished(ctx context.Context) ([]ModelCatalogModel, error)
}

type ModelCatalogInput struct {
	Slug            string   `json:"slug"`
	Model           string   `json:"model"`
	Platform        string   `json:"platform"`
	MetadataModelID string   `json:"metadata_model_id"`
	DisplayNameZH   string   `json:"display_name_zh"`
	DisplayNameEN   string   `json:"display_name_en"`
	SummaryZH       string   `json:"summary_zh"`
	SummaryEN       string   `json:"summary_en"`
	Provider        string   `json:"provider"`
	LogoKey         string   `json:"logo_key"`
	Category        string   `json:"category"`
	Tags            []string `json:"tags"`
	Capabilities    []string `json:"capabilities"`
	ContextWindow   *int64   `json:"context_window"`
	MaxOutputTokens *int64   `json:"max_output_tokens"`
	PublicGroupID   *int64   `json:"public_group_id"`
	Featured        bool     `json:"featured"`
	SortOrder       int      `json:"sort_order"`
}

type ModelCatalogValidation struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors"`
}

type CatalogPeakRate struct {
	Enabled    bool    `json:"enabled"`
	Start      string  `json:"start"`
	End        string  `json:"end"`
	Multiplier float64 `json:"multiplier"`
}

type CatalogPricingInterval struct {
	MinTokens         int      `json:"min_tokens"`
	MaxTokens         *int     `json:"max_tokens"`
	TierLabel         string   `json:"tier_label,omitempty"`
	InputPrice        *float64 `json:"input_price"`
	OutputPrice       *float64 `json:"output_price"`
	CacheWritePrice   *float64 `json:"cache_write_price"`
	CacheReadPrice    *float64 `json:"cache_read_price"`
	CacheWrite1hPrice *float64 `json:"cache_write_1h_price,omitempty"`
	PerRequestPrice   *float64 `json:"per_request_price"`
}

type CatalogPricing struct {
	Label                   string                   `json:"label"`
	BillingMode             string                   `json:"billing_mode"`
	Currency                string                   `json:"currency"`
	Unit                    string                   `json:"unit"`
	InputPrice              *float64                 `json:"input_price"`
	OutputPrice             *float64                 `json:"output_price"`
	CacheWritePrice         *float64                 `json:"cache_write_price"`
	CacheReadPrice          *float64                 `json:"cache_read_price"`
	CacheWrite1hPrice       *float64                 `json:"cache_write_1h_price,omitempty"`
	PriorityInputPrice      *float64                 `json:"priority_input_price,omitempty"`
	PriorityOutputPrice     *float64                 `json:"priority_output_price,omitempty"`
	PriorityCacheWritePrice *float64                 `json:"priority_cache_write_price,omitempty"`
	PriorityCacheReadPrice  *float64                 `json:"priority_cache_read_price,omitempty"`
	ImageInputPrice         *float64                 `json:"image_input_price"`
	ImageOutputPrice        *float64                 `json:"image_output_price"`
	PerRequestPrice         *float64                 `json:"per_request_price"`
	Intervals               []CatalogPricingInterval `json:"intervals"`
	PeakRate                CatalogPeakRate          `json:"peak_rate"`
}

type ModelCatalogPublicGroup struct {
	ID                 int64   `json:"id"`
	Name               string  `json:"name"`
	Platform           string  `json:"platform"`
	RateMultiplier     float64 `json:"rate_multiplier"`
	PeakRateEnabled    bool    `json:"peak_rate_enabled"`
	PeakStart          string  `json:"peak_start"`
	PeakEnd            string  `json:"peak_end"`
	PeakRateMultiplier float64 `json:"peak_rate_multiplier"`
}

type ModelCatalogAdminItem struct {
	ModelCatalogModel
	PublicGroup *ModelCatalogPublicGroup `json:"public_group"`
	// PublicGroupName is retained as a flat convenience field for admin clients.
	PublicGroupName  string                 `json:"public_group_name"`
	Pricing          *CatalogPricing        `json:"pricing"`
	ResolvedMetadata *CatalogMetadata       `json:"resolved_metadata"`
	Validation       ModelCatalogValidation `json:"validation"`
}

type ModelCatalogGroupOption struct {
	ID                 int64           `json:"id"`
	Name               string          `json:"name"`
	Platform           string          `json:"platform"`
	ChannelID          int64           `json:"channel_id"`
	ChannelName        string          `json:"channel_name"`
	RateMultiplier     float64         `json:"rate_multiplier"`
	PeakRateEnabled    bool            `json:"peak_rate_enabled"`
	PeakStart          string          `json:"peak_start"`
	PeakEnd            string          `json:"peak_end"`
	PeakRateMultiplier float64         `json:"peak_rate_multiplier"`
	Pricing            *CatalogPricing `json:"pricing"`
}

type ModelCatalogCandidate struct {
	Model           string                    `json:"model"`
	Platform        string                    `json:"platform"`
	Provider        string                    `json:"provider"`
	LogoKey         string                    `json:"logo_key"`
	Category        string                    `json:"category"`
	ContextWindow   *int64                    `json:"context_window"`
	MaxOutputTokens *int64                    `json:"max_output_tokens"`
	Capabilities    []string                  `json:"capabilities"`
	GroupOptions    []ModelCatalogGroupOption `json:"group_options"`
}

// CatalogMetadata is the exact (never fuzzy-family-matched) LiteLLM metadata
// used as an optional display fallback.
type CatalogMetadata struct {
	ModelID         string   `json:"model_id"`
	Provider        string   `json:"provider"`
	Mode            string   `json:"mode"`
	ContextWindow   *int64   `json:"context_window"`
	MaxOutputTokens *int64   `json:"max_output_tokens"`
	Capabilities    []string `json:"capabilities"`
}

type PublicModelCatalogItem struct {
	Slug            string          `json:"slug"`
	Model           string          `json:"model"`
	DisplayName     string          `json:"display_name"`
	Summary         string          `json:"summary"`
	Provider        string          `json:"provider"`
	LogoKey         string          `json:"logo_key"`
	Category        string          `json:"category"`
	Tags            []string        `json:"tags"`
	Capabilities    []string        `json:"capabilities"`
	ContextWindow   *int64          `json:"context_window"`
	MaxOutputTokens *int64          `json:"max_output_tokens"`
	Featured        bool            `json:"featured"`
	Pricing         *CatalogPricing `json:"pricing"`
}

type PublicModelCatalogResponse struct {
	Items            []PublicModelCatalogItem `json:"items"`
	ServerTimezone   string                   `json:"server_timezone"`
	PricingUpdatedAt string                   `json:"pricing_updated_at"`
}

type modelCatalogResolvedOffer struct {
	group     *Group
	channel   *Channel
	pricing   *ChannelModelPricing
	public    *CatalogPricing
	updatedAt time.Time
}

type modelCatalogSnapshot struct {
	data      PublicModelCatalogResponse
	etag      string
	expiresAt time.Time
}

type ModelCatalogService struct {
	repo           ModelCatalogRepository
	channelRepo    ChannelRepository
	groupRepo      GroupRepository
	pricingService *PricingService

	cacheMu         sync.RWMutex
	cache           map[string]modelCatalogSnapshot
	cacheGeneration uint64
	buildMu         sync.Mutex
}

func NewModelCatalogService(
	repo ModelCatalogRepository,
	channelRepo ChannelRepository,
	groupRepo GroupRepository,
	pricingService *PricingService,
) *ModelCatalogService {
	return &ModelCatalogService{
		repo: repo, channelRepo: channelRepo, groupRepo: groupRepo, pricingService: pricingService,
		cache: make(map[string]modelCatalogSnapshot, 2),
	}
}

var slugUnsafePattern = regexp.MustCompile(`[^a-z0-9]+`)

func modelCatalogSlug(platform, model string) string {
	raw := strings.ToLower(strings.TrimSpace(platform + "-" + model))
	raw = strings.Trim(slugUnsafePattern.ReplaceAllString(raw, "-"), "-")
	if len(raw) > 150 {
		raw = strings.Trim(raw[:150], "-")
	}
	return raw
}

func normalizeCatalogInput(input ModelCatalogInput) (ModelCatalogInput, error) {
	input.Model = strings.TrimSpace(input.Model)
	input.Platform = strings.ToLower(strings.TrimSpace(input.Platform))
	if input.Model == "" || input.Platform == "" {
		return input, ErrModelCatalogInvalid
	}
	input.Slug = strings.Trim(slugUnsafePattern.ReplaceAllString(strings.ToLower(strings.TrimSpace(input.Slug)), "-"), "-")
	if input.Slug == "" {
		input.Slug = modelCatalogSlug(input.Platform, input.Model)
	}
	if input.Slug == "" || utf8.RuneCountInString(input.Slug) > 160 || utf8.RuneCountInString(input.Model) > 255 ||
		utf8.RuneCountInString(input.Platform) > 64 || utf8.RuneCountInString(input.MetadataModelID) > 255 ||
		utf8.RuneCountInString(input.DisplayNameZH) > 160 || utf8.RuneCountInString(input.DisplayNameEN) > 160 ||
		utf8.RuneCountInString(input.Provider) > 80 || utf8.RuneCountInString(input.LogoKey) > 80 ||
		utf8.RuneCountInString(input.Category) > 80 || int64(input.SortOrder) < -2147483648 || int64(input.SortOrder) > 2147483647 {
		return input, ErrModelCatalogInvalid
	}
	input.MetadataModelID = strings.TrimSpace(input.MetadataModelID)
	input.DisplayNameZH = strings.TrimSpace(input.DisplayNameZH)
	input.DisplayNameEN = strings.TrimSpace(input.DisplayNameEN)
	input.SummaryZH = strings.TrimSpace(input.SummaryZH)
	input.SummaryEN = strings.TrimSpace(input.SummaryEN)
	input.Provider = strings.ToLower(strings.TrimSpace(input.Provider))
	input.LogoKey = strings.ToLower(strings.TrimSpace(input.LogoKey))
	input.Category = strings.ToLower(strings.TrimSpace(input.Category))
	input.Tags = normalizedStringSet(input.Tags)
	input.Capabilities = normalizedStringSet(input.Capabilities)
	if input.ContextWindow != nil && *input.ContextWindow <= 0 {
		input.ContextWindow = nil
	}
	if input.MaxOutputTokens != nil && *input.MaxOutputTokens <= 0 {
		input.MaxOutputTokens = nil
	}
	if input.PublicGroupID != nil && *input.PublicGroupID <= 0 {
		input.PublicGroupID = nil
	}
	return input, nil
}

func normalizedStringSet(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func catalogModelFromInput(input ModelCatalogInput) *ModelCatalogModel {
	return &ModelCatalogModel{
		Slug: input.Slug, Model: input.Model, Platform: input.Platform,
		MetadataModelID: input.MetadataModelID, DisplayNameZH: input.DisplayNameZH,
		DisplayNameEN: input.DisplayNameEN, SummaryZH: input.SummaryZH, SummaryEN: input.SummaryEN,
		Provider: input.Provider, LogoKey: input.LogoKey, Category: input.Category,
		Tags: input.Tags, Capabilities: input.Capabilities, ContextWindow: input.ContextWindow,
		MaxOutputTokens: input.MaxOutputTokens, PublicGroupID: input.PublicGroupID,
		Status: ModelCatalogStatusDraft, Featured: input.Featured, SortOrder: input.SortOrder,
	}
}

func applyCatalogInput(dst *ModelCatalogModel, input ModelCatalogInput) {
	dst.Slug, dst.Model, dst.Platform = input.Slug, input.Model, input.Platform
	dst.MetadataModelID = input.MetadataModelID
	dst.DisplayNameZH, dst.DisplayNameEN = input.DisplayNameZH, input.DisplayNameEN
	dst.SummaryZH, dst.SummaryEN = input.SummaryZH, input.SummaryEN
	dst.Provider, dst.LogoKey, dst.Category = input.Provider, input.LogoKey, input.Category
	dst.Tags, dst.Capabilities = input.Tags, input.Capabilities
	dst.ContextWindow, dst.MaxOutputTokens = input.ContextWindow, input.MaxOutputTokens
	dst.PublicGroupID, dst.Featured, dst.SortOrder = input.PublicGroupID, input.Featured, input.SortOrder
}

func (s *ModelCatalogService) Create(ctx context.Context, input ModelCatalogInput) (*ModelCatalogAdminItem, error) {
	normalized, err := normalizeCatalogInput(input)
	if err != nil {
		return nil, err
	}
	if err := s.validateCatalogGroupReference(ctx, normalized.PublicGroupID); err != nil {
		return nil, err
	}
	model := catalogModelFromInput(normalized)
	view, err := s.adminView(ctx, model)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, model); err != nil {
		return nil, err
	}
	s.InvalidatePublicCache()
	view.ModelCatalogModel = *model
	return view, nil
}

func (s *ModelCatalogService) Update(ctx context.Context, id int64, input ModelCatalogInput) (*ModelCatalogAdminItem, error) {
	normalized, err := normalizeCatalogInput(input)
	if err != nil {
		return nil, err
	}
	model, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.validateCatalogGroupReference(ctx, normalized.PublicGroupID); err != nil {
		return nil, err
	}
	applyCatalogInput(model, normalized)
	if model.Status == ModelCatalogStatusPublished {
		if err := s.validatePublishable(ctx, model); err != nil {
			return nil, err
		}
	}
	view, err := s.adminView(ctx, model)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, model); err != nil {
		return nil, err
	}
	s.InvalidatePublicCache()
	view.ModelCatalogModel = *model
	return view, nil
}

func (s *ModelCatalogService) validateCatalogGroupReference(ctx context.Context, groupID *int64) error {
	if groupID == nil {
		return nil
	}
	group, err := s.groupRepo.GetByIDLite(ctx, *groupID)
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			return infraerrors.BadRequest("MODEL_CATALOG_GROUP_INVALID", "public_group_id does not exist")
		}
		return fmt.Errorf("validate public model catalog group: %w", err)
	}
	if group == nil {
		return infraerrors.BadRequest("MODEL_CATALOG_GROUP_INVALID", "public_group_id does not exist")
	}
	return nil
}

func (s *ModelCatalogService) Get(ctx context.Context, id int64) (*ModelCatalogAdminItem, error) {
	model, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.adminView(ctx, model)
}

func (s *ModelCatalogService) List(ctx context.Context) ([]ModelCatalogAdminItem, error) {
	models, err := s.repo.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ModelCatalogAdminItem, 0, len(models))
	for i := range models {
		view, err := s.adminView(ctx, &models[i])
		if err != nil {
			return nil, err
		}
		out = append(out, *view)
	}
	return out, nil
}

func (s *ModelCatalogService) Publish(ctx context.Context, id int64) (*ModelCatalogAdminItem, error) {
	model, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.validatePublishable(ctx, model); err != nil {
		return nil, err
	}
	view, err := s.adminView(ctx, model)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	model.Status = ModelCatalogStatusPublished
	model.PublishedAt = &now
	if err := s.repo.Update(ctx, model); err != nil {
		return nil, err
	}
	s.InvalidatePublicCache()
	view.ModelCatalogModel = *model
	return view, nil
}

func (s *ModelCatalogService) validatePublishable(ctx context.Context, model *ModelCatalogModel) error {
	if model == nil || strings.TrimSpace(model.DisplayNameZH) == "" || strings.TrimSpace(model.SummaryZH) == "" {
		return infraerrors.BadRequest("MODEL_CATALOG_COPY_REQUIRED", "display_name_zh and summary_zh are required before publishing")
	}
	_, validation, err := s.resolveOffering(ctx, model)
	if err != nil {
		return err
	}
	if !validation.Valid {
		return infraerrors.BadRequest("MODEL_CATALOG_NOT_PUBLISHABLE", strings.Join(validation.Errors, "; "))
	}
	return nil
}

func (s *ModelCatalogService) Unpublish(ctx context.Context, id int64) (*ModelCatalogAdminItem, error) {
	model, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	view, err := s.adminView(ctx, model)
	if err != nil {
		return nil, err
	}
	model.Status = ModelCatalogStatusDraft
	model.PublishedAt = nil
	if err := s.repo.Update(ctx, model); err != nil {
		return nil, err
	}
	s.InvalidatePublicCache()
	view.ModelCatalogModel = *model
	return view, nil
}

func (s *ModelCatalogService) InvalidatePublicCache() {
	s.cacheMu.Lock()
	s.cacheGeneration++
	s.cache = make(map[string]modelCatalogSnapshot, 2)
	s.cacheMu.Unlock()
}

func (s *ModelCatalogService) adminView(ctx context.Context, model *ModelCatalogModel) (*ModelCatalogAdminItem, error) {
	metadata := s.metadataFor(model)
	offer, validation, err := s.resolveOffering(ctx, model)
	if err != nil {
		return nil, err
	}
	view := &ModelCatalogAdminItem{
		ModelCatalogModel: *model,
		ResolvedMetadata:  metadata,
		Validation:        validation,
	}
	if offer != nil && offer.group != nil {
		view.PublicGroup = publicGroupView(offer.group)
		view.PublicGroupName = offer.group.Name
		view.Pricing = offer.public
	}
	return view, nil
}

func publicGroupView(group *Group) *ModelCatalogPublicGroup {
	if group == nil {
		return nil
	}
	return &ModelCatalogPublicGroup{
		ID: group.ID, Name: group.Name, Platform: group.Platform, RateMultiplier: group.RateMultiplier,
		PeakRateEnabled: group.PeakRateEnabled, PeakStart: group.PeakStart, PeakEnd: group.PeakEnd,
		PeakRateMultiplier: group.PeakRateMultiplier,
	}
}

func (s *ModelCatalogService) resolveOffering(ctx context.Context, model *ModelCatalogModel) (*modelCatalogResolvedOffer, ModelCatalogValidation, error) {
	errorsList := make([]string, 0, 4)
	if model == nil || model.PublicGroupID == nil || *model.PublicGroupID <= 0 {
		return nil, ModelCatalogValidation{Valid: false, Errors: []string{"public_group_id is required"}}, nil
	}
	group, err := s.groupRepo.GetByIDLite(ctx, *model.PublicGroupID)
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			return nil, ModelCatalogValidation{Valid: false, Errors: []string{"public group does not exist"}}, nil
		}
		return nil, ModelCatalogValidation{}, fmt.Errorf("get public model catalog group: %w", err)
	}
	if group == nil {
		return nil, ModelCatalogValidation{Valid: false, Errors: []string{"public group does not exist"}}, nil
	}
	if group.Status != StatusActive {
		errorsList = append(errorsList, "public group is not active")
	}
	if group.SubscriptionType != SubscriptionTypeStandard {
		errorsList = append(errorsList, "public group must use standard billing")
	}
	if group.IsExclusive {
		errorsList = append(errorsList, "exclusive groups cannot be published")
	}
	if !strings.EqualFold(group.Platform, model.Platform) {
		errorsList = append(errorsList, "public group platform does not match model platform")
	}
	channels, err := s.channelRepo.ListAll(ctx)
	if err != nil {
		return nil, ModelCatalogValidation{}, fmt.Errorf("list public model catalog channels: %w", err)
	}
	selected, channelErrors := s.resolveOfferingFromSources(model, group, channels)
	errorsList = append(errorsList, channelErrors...)
	return selected, ModelCatalogValidation{Valid: len(errorsList) == 0 && selected != nil, Errors: errorsList}, nil
}

func (s *ModelCatalogService) resolveOfferingFromSources(model *ModelCatalogModel, group *Group, channels []Channel) (*modelCatalogResolvedOffer, []string) {
	errorsList := make([]string, 0, 2)
	var selected *modelCatalogResolvedOffer
	for i := range channels {
		channel := &channels[i]
		if channel.Status != StatusActive || !containsInt64(channel.GroupIDs, group.ID) {
			continue
		}
		for _, supported := range channel.SupportedModels() {
			if !strings.EqualFold(supported.Platform, model.Platform) || !strings.EqualFold(supported.Name, model.Model) {
				continue
			}
			billingModel, configuredPricing, supportedForCatalog := catalogBillingModelAndPricing(channel, supported)
			if !supportedForCatalog {
				errorsList = append(errorsList, "channel billing source cannot provide an exact public price")
				break
			}
			resolvedPricing := s.resolveCatalogChannelPricing(billingModel, configuredPricing)
			publicPricing := catalogPricingFromChannel(resolvedPricing, group)
			s.applyCatalogComplexPricing(publicPricing, billingModel, resolvedPricing, configuredPricing, group)
			s.applyCatalogImagePricing(publicPricing, billingModel, resolvedPricing, configuredPricing, group)
			if catalogPerRequestImageRateAmbiguous(resolvedPricing, group) ||
				!catalogPricingIsPublishable(resolvedPricing, group, publicPricing) {
				errorsList = append(errorsList, "model has no explicit channel price")
				break
			}
			updatedAt := channel.UpdatedAt
			if group.UpdatedAt.After(updatedAt) {
				updatedAt = group.UpdatedAt
			}
			if resolvedPricing != nil && resolvedPricing.UpdatedAt.After(updatedAt) {
				updatedAt = resolvedPricing.UpdatedAt
			}
			if s.pricingService != nil {
				if _, exact := s.pricingService.GetExactModelPricing(billingModel); exact != nil {
					if pricingUpdated := s.pricingService.LastUpdated(); pricingUpdated.After(updatedAt) {
						updatedAt = pricingUpdated
					}
				}
			}
			selected = &modelCatalogResolvedOffer{
				group: group, channel: channel, pricing: resolvedPricing,
				public: publicPricing, updatedAt: updatedAt,
			}
			break
		}
		if selected != nil {
			break
		}
	}
	if selected == nil && len(errorsList) == 0 {
		errorsList = append(errorsList, "active channel does not support this model for the selected group")
	}
	if selected != nil {
		return selected, nil
	}
	return selected, errorsList
}

func (s *ModelCatalogService) applyCatalogComplexPricing(public *CatalogPricing, billingModel string, resolved, configured *ChannelModelPricing, group *Group) {
	if public == nil || resolved == nil || group == nil {
		return
	}
	mode := resolved.BillingMode
	if mode == "" {
		mode = BillingModeToken
	}
	if mode != BillingModeToken {
		return
	}
	validConfiguredIntervals := configured != nil && len(filterValidIntervals(configured.Intervals)) > 0
	if configured != nil && !validConfiguredIntervals && configured.CacheWritePrice != nil {
		public.CacheWrite1hPrice = public.CacheWritePrice
	}
	if s.pricingService == nil {
		return
	}
	_, exact := s.pricingService.GetExactModelPricing(billingModel)
	if exact == nil {
		return
	}
	multiplier := group.RateMultiplier
	if resolved.BillingMode == BillingModeImage && group.ImageRateIndependent {
		multiplier = group.ImageRateMultiplier
	}
	if multiplier < 0 {
		multiplier = 0
	}
	if public.CacheWrite1hPrice == nil && exact.CacheCreationInputTokenCostAbove1hr > 0 && exact.CacheCreationInputTokenCostAbove1hr > exact.CacheCreationInputTokenCost {
		public.CacheWrite1hPrice = multipliedPrice(&exact.CacheCreationInputTokenCostAbove1hr, multiplier)
	}
	priorityConfigured := exact.InputCostPerTokenPriority > 0 || exact.OutputCostPerTokenPriority > 0 ||
		exact.CacheCreationInputTokenCostPriority > 0 || exact.CacheReadInputTokenCostPriority > 0
	// Runtime interval pricing assigns each interval's regular prices to its
	// priority fields. The public DTO currently has no per-interval priority
	// shape, so a top-level LiteLLM priority quote would be false for requests
	// that hit a channel interval. Suppress it until that contract exists.
	if len(resolved.Intervals) == 0 && (exact.SupportsServiceTier || priorityConfigured) {
		priority := func(value float64, base *float64) *float64 {
			if value > 0 {
				return multipliedPrice(&value, multiplier)
			}
			if priorityConfigured {
				return base
			}
			if base == nil {
				return nil
			}
			doubled := *base * 2
			return &doubled
		}
		public.PriorityInputPrice = priority(exact.InputCostPerTokenPriority, public.InputPrice)
		public.PriorityOutputPrice = priority(exact.OutputCostPerTokenPriority, public.OutputPrice)
		public.PriorityCacheWritePrice = priority(exact.CacheCreationInputTokenCostPriority, public.CacheWritePrice)
		public.PriorityCacheReadPrice = priority(exact.CacheReadInputTokenCostPriority, public.CacheReadPrice)
		if configured != nil && !validConfiguredIntervals {
			if configured.InputPrice != nil {
				public.PriorityInputPrice = public.InputPrice
			}
			if configured.OutputPrice != nil {
				public.PriorityOutputPrice = public.OutputPrice
			}
			if configured.CacheWritePrice != nil {
				public.PriorityCacheWritePrice = public.CacheWritePrice
			}
			if configured.CacheReadPrice != nil {
				public.PriorityCacheReadPrice = public.CacheReadPrice
			}
		}
	}
	if len(resolved.Intervals) != 0 || exact.LongContextInputTokenThreshold <= 0 ||
		(exact.LongContextInputCostMultiplier <= 1 && exact.LongContextOutputCostMultiplier <= 1) {
		return
	}
	inputMultiplier := exact.LongContextInputCostMultiplier
	if inputMultiplier <= 0 {
		inputMultiplier = 1
	}
	outputMultiplier := exact.LongContextOutputCostMultiplier
	if outputMultiplier <= 0 {
		outputMultiplier = 1
	}
	public.Intervals = append(public.Intervals, CatalogPricingInterval{
		MinTokens: exact.LongContextInputTokenThreshold, TierLabel: "long_context",
		InputPrice:        multipliedPrice(public.InputPrice, inputMultiplier),
		OutputPrice:       multipliedPrice(public.OutputPrice, outputMultiplier),
		CacheWritePrice:   multipliedPrice(public.CacheWritePrice, inputMultiplier),
		CacheReadPrice:    multipliedPrice(public.CacheReadPrice, inputMultiplier),
		CacheWrite1hPrice: multipliedPrice(public.CacheWrite1hPrice, inputMultiplier),
	})
}

func (s *ModelCatalogService) applyCatalogImagePricing(public *CatalogPricing, billingModel string, resolved, configured *ChannelModelPricing, group *Group) {
	if public == nil || resolved == nil || group == nil {
		return
	}
	mode := resolved.BillingMode
	if mode == "" {
		mode = BillingModeToken
	}
	if mode != BillingModeImage {
		return
	}
	multiplier := group.RateMultiplier
	if group.ImageRateIndependent {
		multiplier = group.ImageRateMultiplier
	}
	if multiplier < 0 {
		multiplier = 0
	}
	groupConfig := &ImagePriceConfig{Price1K: group.ImagePrice1K, Price2K: group.ImagePrice2K, Price4K: group.ImagePrice4K}
	exactLookup := func(model string) *LiteLLMModelPricing {
		if s.pricingService == nil {
			return nil
		}
		_, pricing := s.pricingService.GetExactModelPricing(model)
		return pricing
	}
	for _, tier := range []string{ImageBillingSize1K, ImageBillingSize2K, ImageBillingSize4K} {
		var rawPrice *float64
		if configured == nil {
			price := resolveImageUnitPrice(billingModel, tier, groupConfig, exactLookup)
			rawPrice = &price
		} else {
			rawPrice = catalogConfiguredImageTierPrice(configured, groupConfig, tier)
		}
		if rawPrice == nil {
			continue
		}
		public.Intervals = appendConfiguredImagePrice(public.Intervals, tier, rawPrice, multiplier)
	}
	if price := catalogImageTierPrice(public.Intervals, ImageBillingSize2K); price != nil {
		// 2K is the runtime default for omitted/invalid image sizes.
		public.PerRequestPrice = price
	}
}

func catalogConfiguredImageTierPrice(configured *ChannelModelPricing, groupConfig *ImagePriceConfig, tier string) *float64 {
	if groupConfig != nil {
		switch tier {
		case ImageBillingSize1K:
			if groupConfig.Price1K != nil {
				return groupConfig.Price1K
			}
		case ImageBillingSize2K:
			if groupConfig.Price2K != nil {
				return groupConfig.Price2K
			}
		case ImageBillingSize4K:
			if groupConfig.Price4K != nil {
				return groupConfig.Price4K
			}
		}
	}
	if configured == nil {
		return nil
	}
	for i := range configured.Intervals {
		interval := &configured.Intervals[i]
		// Runtime tier lookup is exact and preserves an explicit zero as free.
		if interval.TierLabel == tier && interval.PerRequestPrice != nil {
			return interval.PerRequestPrice
		}
	}
	return configured.PerRequestPrice
}

func catalogImageTierPrice(intervals []CatalogPricingInterval, tier string) *float64 {
	for i := range intervals {
		if intervals[i].TierLabel == tier && intervals[i].PerRequestPrice != nil {
			return intervals[i].PerRequestPrice
		}
	}
	return nil
}

func catalogHasCompleteImageTiers(pricing *CatalogPricing) bool {
	if pricing == nil {
		return false
	}
	return catalogImageTierPrice(pricing.Intervals, ImageBillingSize1K) != nil &&
		catalogImageTierPrice(pricing.Intervals, ImageBillingSize2K) != nil &&
		catalogImageTierPrice(pricing.Intervals, ImageBillingSize4K) != nil
}

func catalogHasNonCanonicalImageTier(pricing *CatalogPricing) bool {
	if pricing == nil {
		return false
	}
	canonical := []string{ImageBillingSize1K, ImageBillingSize2K, ImageBillingSize4K}
	for _, interval := range pricing.Intervals {
		for _, tier := range canonical {
			if strings.EqualFold(interval.TierLabel, tier) && interval.TierLabel != tier {
				return true
			}
		}
	}
	return false
}

func catalogPricingIsPublishable(resolved *ChannelModelPricing, group *Group, public *CatalogPricing) bool {
	if resolved == nil {
		return false
	}
	mode := resolved.BillingMode
	if mode == "" {
		mode = BillingModeToken
	}
	if mode == BillingModeImage {
		// Runtime request-tier lookup is case-sensitive. Reject visually similar
		// lowercase/mixed-case labels rather than publishing a tier the gateway
		// will miss and potentially charge at a different fallback price.
		return !catalogHasNonCanonicalImageTier(public) && catalogHasCompleteImageTiers(public)
	}
	return hasExplicitCatalogPricing(resolved, group)
}

func catalogPerRequestImageRateAmbiguous(resolved *ChannelModelPricing, group *Group) bool {
	if resolved == nil || group == nil || resolved.BillingMode != BillingModePerRequest {
		return false
	}
	// A per_request channel can be used by the image endpoint regardless of
	// whether its custom model name or LiteLLM metadata identifies it as an
	// image model. Without endpoint-specific catalog pricing, any such price
	// ambiguity must fail closed rather than relying on model-name inference.
	// Group image tier overrides bypass channel per_request pricing for the
	// matching size, which the current public per_request DTO cannot classify.
	if group.ImagePrice1K != nil || group.ImagePrice2K != nil || group.ImagePrice4K != nil {
		return true
	}
	return group.ImageRateIndependent && group.ImageRateMultiplier != group.RateMultiplier
}

func catalogBillingModelAndPricing(channel *Channel, supported SupportedModel) (string, *ChannelModelPricing, bool) {
	if channel == nil {
		return "", nil, false
	}
	source := strings.TrimSpace(channel.BillingModelSource)
	if source == "" {
		source = BillingModelSourceChannelMapped
	}
	var billingModel string
	var configured *ChannelModelPricing
	switch source {
	case BillingModelSourceUpstream:
		// The upstream response can select/rename the billable model at runtime;
		// a static anonymous catalog cannot make an exact price promise.
		return "", nil, false
	case BillingModelSourceRequested:
		billingModel = supported.Name
		configured = catalogRequestedModelPricing(channel, supported.Platform, billingModel)
		if mapping := channel.ModelMapping[supported.Platform]; len(mapping) > 0 {
			target, matched, exact := catalogExactMappingTarget(mapping, supported.Name)
			if matched && exact && isGrokVideoBillingModel(target) {
				// requested remains the pricing identity, but the deterministic
				// channel target still drives runtime video detection and billing.
				return "", nil, false
			}
		}
	case BillingModelSourceChannelMapped, BillingModelSourceResponse:
		billingModel = supported.Name
		mappedTarget := false
		if mapping := channel.ModelMapping[supported.Platform]; len(mapping) > 0 {
			target, matched, exact := catalogExactMappingTarget(mapping, supported.Name)
			if matched && !exact {
				return "", nil, false
			}
			if matched && target != "" {
				billingModel = target
				configured = catalogRequestedModelPricing(channel, supported.Platform, billingModel)
				mappedTarget = true
			}
		}
		if !mappedTarget && configured == nil {
			configured = supported.Pricing
		}
	default:
		return "", nil, false
	}
	if strings.TrimSpace(billingModel) == "" {
		return "", nil, false
	}
	if isGrokVideoBillingModel(billingModel) {
		// Video billing is resolution- and duration-based and uses independent
		// group video prices/multipliers. CatalogPricing has no video schema yet.
		return "", nil, false
	}
	if channel.RestrictModels {
		// Gateway restriction checks the actual billable model, not the public
		// alias. A global LiteLLM exact match must not bypass that channel-local
		// allowlist. Re-resolve here to preserve exact/wildcard platform parity.
		configured = catalogRequestedModelPricing(channel, supported.Platform, billingModel)
		if configured == nil {
			return "", nil, false
		}
	}
	return billingModel, configured, true
}

func catalogExactMappingTarget(mapping map[string]string, model string) (target string, matched bool, exact bool) {
	for sourceModel, destination := range mapping {
		if strings.Contains(sourceModel, "*") || !strings.EqualFold(strings.TrimSpace(sourceModel), strings.TrimSpace(model)) {
			continue
		}
		destination = strings.TrimSpace(destination)
		if destination == "" || strings.Contains(destination, "*") {
			return "", true, false
		}
		return destination, true, true
	}
	modelLower := strings.ToLower(strings.TrimSpace(model))
	matchedTargets := make(map[string]string)
	for sourceModel, destination := range mapping {
		prefix, wildcard := splitWildcardSuffix(strings.ToLower(strings.TrimSpace(sourceModel)))
		if !wildcard || !strings.HasPrefix(modelLower, prefix) {
			continue
		}
		destination = strings.TrimSpace(destination)
		if destination == "" || strings.Contains(destination, "*") {
			return "", true, false
		}
		matchedTargets[strings.ToLower(destination)] = destination
	}
	if len(matchedTargets) != 1 {
		return "", len(matchedTargets) > 0, false
	}
	for _, destination := range matchedTargets {
		return destination, true, true
	}
	return "", false, true
}

func catalogRequestedModelPricing(channel *Channel, platform, model string) *ChannelModelPricing {
	if channel == nil {
		return nil
	}
	for i := range channel.ModelPricing {
		pricing := &channel.ModelPricing[i]
		if !strings.EqualFold(pricing.Platform, platform) {
			continue
		}
		for _, configuredModel := range pricing.Models {
			if strings.EqualFold(strings.TrimSpace(configuredModel), strings.TrimSpace(model)) {
				clone := pricing.Clone()
				return &clone
			}
		}
	}
	modelLower := strings.ToLower(strings.TrimSpace(model))
	for i := range channel.ModelPricing {
		pricing := &channel.ModelPricing[i]
		if !strings.EqualFold(pricing.Platform, platform) {
			continue
		}
		for _, configuredModel := range pricing.Models {
			prefix, wildcard := splitWildcardSuffix(strings.ToLower(strings.TrimSpace(configuredModel)))
			if wildcard && strings.HasPrefix(modelLower, prefix) {
				clone := pricing.Clone()
				return &clone
			}
		}
	}
	return nil
}

func hasExplicitCatalogPricing(pricing *ChannelModelPricing, group *Group) bool {
	if pricing == nil {
		return false
	}
	mode := pricing.BillingMode
	if mode == "" {
		mode = BillingModeToken
	}
	if mode == BillingModeToken {
		if pricing.InputPrice != nil || pricing.OutputPrice != nil || pricing.CacheWritePrice != nil ||
			pricing.CacheReadPrice != nil || pricing.ImageInputPrice != nil {
			return true
		}
	} else if pricing.PerRequestPrice != nil {
		return true
	}
	for _, interval := range pricing.Intervals {
		if mode == BillingModeToken && (interval.InputPrice != nil || interval.OutputPrice != nil ||
			interval.CacheWritePrice != nil || interval.CacheReadPrice != nil) {
			return true
		}
		if mode != BillingModeToken && interval.PerRequestPrice != nil {
			return true
		}
	}
	if mode == BillingModeImage && group != nil {
		return group.ImagePrice1K != nil || group.ImagePrice2K != nil || group.ImagePrice4K != nil
	}
	return false
}

// resolveCatalogChannelPricing mirrors the billing precedence without using the
// fuzzy family fallbacks that are acceptable on the billing hot path but unsafe
// for a public price promise: exact LiteLLM data provides the base and explicit
// channel fields override it. An exact miss plus no channel price fails closed.
func (s *ModelCatalogService) resolveCatalogChannelPricing(model string, configured *ChannelModelPricing) *ChannelModelPricing {
	var resolved *ChannelModelPricing
	if s.pricingService != nil {
		_, exact := s.pricingService.GetExactModelPricing(model)
		if exact != nil {
			resolved = synthesizeCatalogPricingFromLiteLLM(exact, configured)
		}
	}
	hadExactBase := resolved != nil
	if configured == nil {
		if resolved == nil {
			if _, grokImage := getDefaultGrokImagineImagePrice(model, ImageBillingSize2K); grokImage {
				return &ChannelModelPricing{BillingMode: BillingModeImage}
			}
		}
		return resolved
	}
	if resolved == nil {
		clone := configured.Clone()
		resolved = &clone
	}
	resolved.ID = configured.ID
	resolved.ChannelID = configured.ChannelID
	resolved.Platform = configured.Platform
	resolved.Models = append([]string(nil), configured.Models...)
	resolved.CreatedAt = configured.CreatedAt
	resolved.UpdatedAt = configured.UpdatedAt
	if configured.BillingMode != "" {
		resolved.BillingMode = configured.BillingMode
	}
	validIntervals := filterValidIntervals(configured.Intervals)
	mode := resolved.BillingMode
	if mode == "" {
		mode = BillingModeToken
		resolved.BillingMode = mode
	}
	if mode == BillingModeToken && len(validIntervals) == 0 {
		if configured.InputPrice != nil {
			resolved.InputPrice = configured.InputPrice
		}
		if configured.OutputPrice != nil {
			resolved.OutputPrice = configured.OutputPrice
		}
		if configured.CacheWritePrice != nil {
			resolved.CacheWritePrice = configured.CacheWritePrice
		}
		if configured.CacheReadPrice != nil {
			resolved.CacheReadPrice = configured.CacheReadPrice
		}
	} else if mode == BillingModeToken && !hadExactBase {
		// Actual resolver ignores channel flat token/cache fields whenever valid
		// intervals exist. Without an exact public base, keep the out-of-tier
		// price unknown rather than advertising those ignored flat fields.
		resolved.InputPrice = nil
		resolved.OutputPrice = nil
		resolved.CacheWritePrice = nil
		resolved.CacheReadPrice = nil
	}
	if mode != BillingModeToken {
		resolved.InputPrice = nil
		resolved.OutputPrice = nil
		resolved.CacheWritePrice = nil
		resolved.CacheReadPrice = nil
		resolved.ImageInputPrice = nil
		resolved.ImageOutputPrice = nil
		// Channel image/per-request pricing does not inherit LiteLLM's default
		// per-image price. The resolver uses only explicit channel defaults/tiers.
		resolved.PerRequestPrice = configured.PerRequestPrice
	}
	if mode == BillingModeToken {
		if configured.ImageInputPrice != nil && *configured.ImageInputPrice > 0 {
			resolved.ImageInputPrice = configured.ImageInputPrice
		} else {
			// Billing falls back to the effective text input rate.
			resolved.ImageInputPrice = resolved.InputPrice
		}
		if configured.ImageOutputPrice != nil {
			resolved.ImageOutputPrice = configured.ImageOutputPrice
		} else {
			zero := 0.0
			resolved.ImageOutputPrice = &zero
		}
	}
	if mode == BillingModeToken && configured.PerRequestPrice != nil {
		resolved.PerRequestPrice = configured.PerRequestPrice
	}
	if len(validIntervals) > 0 {
		resolved.Intervals = append([]PricingInterval(nil), validIntervals...)
	}
	return resolved
}

// synthesizeCatalogPricingFromLiteLLM is deliberately separate from the
// existing available-channel display helper: the public catalog must preserve
// explicit zeroes, while older UI behavior treats zero-valued LiteLLM fields
// as absent. A non-zero value also counts as present for built-in fallback
// entries created in Go rather than parsed from JSON.
func synthesizeCatalogPricingFromLiteLLM(lp *LiteLLMModelPricing, existing *ChannelModelPricing) *ChannelModelPricing {
	if lp == nil {
		return nil
	}
	mode := BillingModeToken
	if existing != nil {
		mode = existing.BillingMode
		if mode == "" {
			mode = BillingModeToken
		}
	} else if lp.Mode == "image_generation" {
		mode = BillingModeImage
	}
	present := func(value float64, explicitlySet bool) *float64 {
		if !explicitlySet && value == 0 {
			return nil
		}
		copy := value
		return &copy
	}
	result := &ChannelModelPricing{
		BillingMode:      mode,
		InputPrice:       present(lp.InputCostPerToken, lp.InputCostPerTokenSet),
		OutputPrice:      present(lp.OutputCostPerToken, lp.OutputCostPerTokenSet),
		CacheWritePrice:  present(lp.CacheCreationInputTokenCost, lp.CacheCreationInputTokenCostSet),
		CacheReadPrice:   present(lp.CacheReadInputTokenCost, lp.CacheReadInputTokenCostSet),
		ImageInputPrice:  present(lp.InputCostPerImageToken, lp.InputCostPerImageTokenSet),
		ImageOutputPrice: present(lp.OutputCostPerImageToken, lp.OutputCostPerImageTokenSet),
		PerRequestPrice:  present(lp.OutputCostPerImage, lp.OutputCostPerImageSet),
	}
	if mode == BillingModeToken {
		// BillingService treats a zero/missing global LiteLLM image-token price
		// as absent: image input falls back to text input, and image output falls
		// back to text output. Mirror that effective price in the public DTO.
		if result.ImageInputPrice == nil || *result.ImageInputPrice == 0 {
			result.ImageInputPrice = result.InputPrice
		}
		if result.ImageOutputPrice == nil || *result.ImageOutputPrice == 0 {
			result.ImageOutputPrice = result.OutputPrice
		}
	}
	return result
}

func multipliedPrice(value *float64, multiplier float64) *float64 {
	if value == nil {
		return nil
	}
	if multiplier < 0 {
		multiplier = 0
	}
	result := *value * multiplier
	return &result
}

func catalogPricingFromChannel(pricing *ChannelModelPricing, group *Group) *CatalogPricing {
	if pricing == nil || group == nil {
		return nil
	}
	mode := pricing.BillingMode
	if mode == "" {
		mode = BillingModeToken
	}
	multiplier := group.RateMultiplier
	if mode == BillingModeImage && group.ImageRateIndependent {
		multiplier = group.ImageRateMultiplier
	}
	if multiplier < 0 {
		multiplier = 0
	}
	unit := "per_token"
	if mode == BillingModePerRequest || mode == BillingModeImage {
		unit = "per_request"
	}
	result := &CatalogPricing{
		Label: "公开标准价", BillingMode: string(mode), Currency: catalogPricingCurrencyCredit, Unit: unit,
		InputPrice:       multipliedPrice(pricing.InputPrice, multiplier),
		OutputPrice:      multipliedPrice(pricing.OutputPrice, multiplier),
		CacheWritePrice:  multipliedPrice(pricing.CacheWritePrice, multiplier),
		CacheReadPrice:   multipliedPrice(pricing.CacheReadPrice, multiplier),
		ImageInputPrice:  multipliedPrice(pricing.ImageInputPrice, multiplier),
		ImageOutputPrice: multipliedPrice(pricing.ImageOutputPrice, multiplier),
		PerRequestPrice:  multipliedPrice(pricing.PerRequestPrice, multiplier),
		Intervals:        make([]CatalogPricingInterval, 0, len(pricing.Intervals)+3),
		// Peak billing is only valid for subscription groups. Public catalog groups
		// are standard, so fail closed instead of advertising an inapplicable rule.
		PeakRate: CatalogPeakRate{Enabled: false, Start: group.PeakStart, End: group.PeakEnd, Multiplier: group.PeakRateMultiplier},
	}
	for _, interval := range pricing.Intervals {
		if interval.InputPrice == nil && interval.OutputPrice == nil && interval.CacheWritePrice == nil &&
			interval.CacheReadPrice == nil && interval.PerRequestPrice == nil {
			continue
		}
		result.Intervals = append(result.Intervals, CatalogPricingInterval{
			MinTokens: interval.MinTokens, MaxTokens: interval.MaxTokens, TierLabel: interval.TierLabel,
			InputPrice:        multipliedPrice(interval.InputPrice, multiplier),
			OutputPrice:       multipliedPrice(interval.OutputPrice, multiplier),
			CacheWritePrice:   multipliedPrice(interval.CacheWritePrice, multiplier),
			CacheReadPrice:    multipliedPrice(interval.CacheReadPrice, multiplier),
			CacheWrite1hPrice: multipliedPrice(interval.CacheWritePrice, multiplier),
			PerRequestPrice:   multipliedPrice(interval.PerRequestPrice, multiplier),
		})
	}
	return result
}

func appendConfiguredImagePrice(intervals []CatalogPricingInterval, label string, value *float64, multiplier float64) []CatalogPricingInterval {
	if value == nil {
		return intervals
	}
	for i := range intervals {
		if intervals[i].TierLabel == label {
			intervals[i].PerRequestPrice = multipliedPrice(value, multiplier)
			return intervals
		}
	}
	return append(intervals, CatalogPricingInterval{TierLabel: label, PerRequestPrice: multipliedPrice(value, multiplier)})
}

func (s *ModelCatalogService) ListCandidates(ctx context.Context) ([]ModelCatalogCandidate, error) {
	groups, err := s.groupRepo.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("list catalog groups: %w", err)
	}
	publicGroups := make(map[int64]*Group)
	for i := range groups {
		group := &groups[i]
		if group.Status == StatusActive && group.SubscriptionType == SubscriptionTypeStandard && !group.IsExclusive {
			publicGroups[group.ID] = group
		}
	}
	channels, err := s.channelRepo.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("list catalog channels: %w", err)
	}
	byKey := make(map[string]*ModelCatalogCandidate)
	seenGroup := make(map[string]map[int64]struct{})
	for i := range channels {
		channel := &channels[i]
		if channel.Status != StatusActive {
			continue
		}
		for _, groupID := range channel.GroupIDs {
			group := publicGroups[groupID]
			if group == nil {
				continue
			}
			for _, supported := range channel.SupportedModels() {
				billingModel, configuredPricing, supportedForCatalog := catalogBillingModelAndPricing(channel, supported)
				if !supportedForCatalog {
					continue
				}
				resolvedPricing := s.resolveCatalogChannelPricing(billingModel, configuredPricing)
				publicPricing := catalogPricingFromChannel(resolvedPricing, group)
				s.applyCatalogComplexPricing(publicPricing, billingModel, resolvedPricing, configuredPricing, group)
				s.applyCatalogImagePricing(publicPricing, billingModel, resolvedPricing, configuredPricing, group)
				if !strings.EqualFold(supported.Platform, group.Platform) ||
					catalogPerRequestImageRateAmbiguous(resolvedPricing, group) ||
					!catalogPricingIsPublishable(resolvedPricing, group, publicPricing) {
					continue
				}
				key := strings.ToLower(supported.Platform) + "\x00" + strings.ToLower(supported.Name)
				candidate := byKey[key]
				if candidate == nil {
					metadata := s.metadataForName(supported.Name)
					provider := inferCatalogProvider(supported.Name, supported.Platform, metadata)
					candidate = &ModelCatalogCandidate{
						Model: supported.Name, Platform: supported.Platform, Provider: provider,
						LogoKey: provider, Category: inferCatalogCategory(supported.Name, metadata),
						Capabilities: []string{}, GroupOptions: []ModelCatalogGroupOption{},
					}
					if metadata != nil {
						candidate.ContextWindow, candidate.MaxOutputTokens = metadata.ContextWindow, metadata.MaxOutputTokens
						candidate.Capabilities = append([]string(nil), metadata.Capabilities...)
					}
					byKey[key] = candidate
					seenGroup[key] = make(map[int64]struct{})
				}
				if _, duplicate := seenGroup[key][group.ID]; duplicate {
					continue
				}
				seenGroup[key][group.ID] = struct{}{}
				candidate.GroupOptions = append(candidate.GroupOptions, ModelCatalogGroupOption{
					ID: group.ID, Name: group.Name, Platform: group.Platform,
					ChannelID: channel.ID, ChannelName: channel.Name, RateMultiplier: group.RateMultiplier,
					PeakRateEnabled: false, PeakStart: group.PeakStart, PeakEnd: group.PeakEnd,
					PeakRateMultiplier: group.PeakRateMultiplier,
					Pricing:            publicPricing,
				})
			}
		}
	}
	out := make([]ModelCatalogCandidate, 0, len(byKey))
	for _, candidate := range byKey {
		sort.SliceStable(candidate.GroupOptions, func(i, j int) bool {
			if candidate.GroupOptions[i].Name != candidate.GroupOptions[j].Name {
				return candidate.GroupOptions[i].Name < candidate.GroupOptions[j].Name
			}
			return candidate.GroupOptions[i].ID < candidate.GroupOptions[j].ID
		})
		out = append(out, *candidate)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Platform != out[j].Platform {
			return out[i].Platform < out[j].Platform
		}
		return strings.ToLower(out[i].Model) < strings.ToLower(out[j].Model)
	})
	return out, nil
}

func (s *ModelCatalogService) metadataFor(model *ModelCatalogModel) *CatalogMetadata {
	if model == nil {
		return nil
	}
	name := strings.TrimSpace(model.MetadataModelID)
	if name == "" {
		name = model.Model
	}
	return s.metadataForName(name)
}

func (s *ModelCatalogService) metadataForName(name string) *CatalogMetadata {
	if s.pricingService == nil {
		return nil
	}
	modelID, pricing := s.pricingService.GetExactModelPricing(name)
	if pricing == nil {
		return nil
	}
	return &CatalogMetadata{
		ModelID: modelID, Provider: strings.ToLower(pricing.LiteLLMProvider), Mode: pricing.Mode,
		ContextWindow:   positiveInt64Ptr(pricing.MaxInputTokens),
		MaxOutputTokens: positiveInt64Ptr(pricing.MaxOutputTokens),
		Capabilities:    pricing.CapabilityNames(),
	}
}

func positiveInt64Ptr(value int64) *int64 {
	if value <= 0 {
		return nil
	}
	result := value
	return &result
}

func inferCatalogProvider(model, platform string, metadata *CatalogMetadata) string {
	if metadata != nil && strings.TrimSpace(metadata.Provider) != "" {
		return strings.ToLower(strings.TrimSpace(metadata.Provider))
	}
	lower := strings.ToLower(model)
	switch {
	case strings.Contains(lower, "claude"):
		return "anthropic"
	case strings.Contains(lower, "gemini") || strings.Contains(lower, "imagen"):
		return "google"
	case strings.HasPrefix(lower, "gpt-") || strings.Contains(lower, "codex") || strings.HasPrefix(lower, "o1") || strings.HasPrefix(lower, "o3"):
		return "openai"
	case strings.Contains(lower, "deepseek"):
		return "deepseek"
	case strings.Contains(lower, "grok"):
		return "xai"
	case strings.Contains(lower, "kimi") || strings.Contains(lower, "moonshot"):
		return "moonshot"
	case strings.Contains(lower, "glm"):
		return "zhipu"
	default:
		return strings.ToLower(strings.TrimSpace(platform))
	}
}

func inferCatalogCategory(model string, metadata *CatalogMetadata) string {
	mode := ""
	capabilities := []string(nil)
	if metadata != nil {
		mode = strings.ToLower(metadata.Mode)
		capabilities = metadata.Capabilities
	}
	lower := strings.ToLower(model)
	switch {
	case strings.Contains(mode, "embedding") || strings.Contains(lower, "embed"):
		return "embedding"
	case strings.Contains(mode, "image") || strings.Contains(lower, "image") || strings.Contains(lower, "imagen"):
		return "image"
	case strings.Contains(mode, "audio") || strings.Contains(lower, "audio") || strings.Contains(lower, "whisper"):
		return "audio"
	case strings.Contains(lower, "codex") || strings.Contains(lower, "coder") || strings.Contains(lower, "code"):
		return "code"
	case containsString(capabilities, "reasoning"):
		return "reasoning"
	default:
		return "chat"
	}
}

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func (s *ModelCatalogService) PublicSnapshot(ctx context.Context, locale string) (PublicModelCatalogResponse, string, error) {
	locale = normalizeCatalogLocale(locale)
	for {
		now := time.Now()
		s.cacheMu.RLock()
		snapshot, ok := s.cache[locale]
		s.cacheMu.RUnlock()
		if ok && now.Before(snapshot.expiresAt) {
			return snapshot.data, snapshot.etag, nil
		}

		s.buildMu.Lock()
		now = time.Now()
		s.cacheMu.RLock()
		snapshot, ok = s.cache[locale]
		generation := s.cacheGeneration
		s.cacheMu.RUnlock()
		if ok && now.Before(snapshot.expiresAt) {
			s.buildMu.Unlock()
			return snapshot.data, snapshot.etag, nil
		}

		result, etag, err := s.buildPublicSnapshot(ctx, locale)
		if err != nil {
			s.buildMu.Unlock()
			return PublicModelCatalogResponse{}, "", err
		}
		s.cacheMu.Lock()
		if s.cacheGeneration == generation {
			s.cache[locale] = modelCatalogSnapshot{data: result, etag: etag, expiresAt: time.Now().Add(modelCatalogSnapshotTTL)}
			s.cacheMu.Unlock()
			s.buildMu.Unlock()
			return result, etag, nil
		}
		s.cacheMu.Unlock()
		s.buildMu.Unlock()
		if err := ctx.Err(); err != nil {
			return PublicModelCatalogResponse{}, "", err
		}
		// An admin mutation invalidated the cache while this snapshot was being
		// built. Retry from current sources instead of repopulating stale data.
	}
}

func (s *ModelCatalogService) buildPublicSnapshot(ctx context.Context, locale string) (PublicModelCatalogResponse, string, error) {
	models, err := s.repo.ListPublished(ctx)
	if err != nil {
		return PublicModelCatalogResponse{}, "", err
	}
	groups, err := s.groupRepo.ListActive(ctx)
	if err != nil {
		return PublicModelCatalogResponse{}, "", fmt.Errorf("list public model catalog groups: %w", err)
	}
	channels, err := s.channelRepo.ListAll(ctx)
	if err != nil {
		return PublicModelCatalogResponse{}, "", fmt.Errorf("list public model catalog channels: %w", err)
	}
	groupsByID := make(map[int64]*Group, len(groups))
	for i := range groups {
		groupsByID[groups[i].ID] = &groups[i]
	}
	result := PublicModelCatalogResponse{
		Items: make([]PublicModelCatalogItem, 0, len(models)), ServerTimezone: timezone.Name(),
	}
	var pricingUpdatedAt time.Time
	for i := range models {
		model := &models[i]
		// Keep this check even though the repository query is filtered. It makes
		// anonymous serialization fail closed if a future repository or cache
		// implementation accidentally returns a draft/archived row.
		if model.Status != ModelCatalogStatusPublished || strings.TrimSpace(model.DisplayNameZH) == "" || strings.TrimSpace(model.SummaryZH) == "" {
			continue
		}
		if model.PublicGroupID == nil {
			continue
		}
		group := groupsByID[*model.PublicGroupID]
		if group == nil || group.Status != StatusActive || group.SubscriptionType != SubscriptionTypeStandard || group.IsExclusive || !strings.EqualFold(group.Platform, model.Platform) {
			continue
		}
		offer, validationErrors := s.resolveOfferingFromSources(model, group, channels)
		if len(validationErrors) != 0 || offer == nil || offer.public == nil {
			continue
		}
		metadata := s.metadataFor(model)
		provider := strings.ToLower(strings.TrimSpace(model.Provider))
		if provider == "" {
			provider = inferCatalogProvider(model.Model, model.Platform, metadata)
		}
		logoKey := strings.ToLower(strings.TrimSpace(model.LogoKey))
		if logoKey == "" {
			logoKey = provider
		}
		category := strings.ToLower(strings.TrimSpace(model.Category))
		if category == "" {
			category = inferCatalogCategory(model.Model, metadata)
		}
		capabilities := append([]string(nil), model.Capabilities...)
		contextWindow, maxOutputTokens := model.ContextWindow, model.MaxOutputTokens
		if metadata != nil {
			if len(capabilities) == 0 {
				capabilities = append([]string(nil), metadata.Capabilities...)
			}
			if contextWindow == nil {
				contextWindow = metadata.ContextWindow
			}
			if maxOutputTokens == nil {
				maxOutputTokens = metadata.MaxOutputTokens
			}
		}
		displayName, summary := localizedCatalogCopy(model, locale)
		result.Items = append(result.Items, PublicModelCatalogItem{
			Slug: model.Slug, Model: model.Model, DisplayName: displayName, Summary: summary,
			Provider: provider, LogoKey: logoKey, Category: category,
			Tags: append([]string(nil), model.Tags...), Capabilities: capabilities,
			ContextWindow: contextWindow, MaxOutputTokens: maxOutputTokens,
			Featured: model.Featured, Pricing: offer.public,
		})
		updatedAt := model.UpdatedAt
		if offer.updatedAt.After(updatedAt) {
			updatedAt = offer.updatedAt
		}
		if updatedAt.After(pricingUpdatedAt) {
			pricingUpdatedAt = updatedAt
		}
	}
	if !pricingUpdatedAt.IsZero() {
		result.PricingUpdatedAt = pricingUpdatedAt.UTC().Format(time.RFC3339)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return PublicModelCatalogResponse{}, "", fmt.Errorf("marshal model catalog snapshot: %w", err)
	}
	digest := sha256.Sum256(encoded)
	etag := `"` + hex.EncodeToString(digest[:]) + `"`
	return result, etag, nil
}

func normalizeCatalogLocale(locale string) string {
	locale = strings.ToLower(strings.TrimSpace(locale))
	if strings.HasPrefix(locale, "en") {
		return "en"
	}
	return "zh"
}

func localizedCatalogCopy(model *ModelCatalogModel, locale string) (string, string) {
	if normalizeCatalogLocale(locale) == "en" {
		name := strings.TrimSpace(model.DisplayNameEN)
		summary := strings.TrimSpace(model.SummaryEN)
		if name == "" {
			name = strings.TrimSpace(model.DisplayNameZH)
		}
		if summary == "" {
			summary = strings.TrimSpace(model.SummaryZH)
		}
		return name, summary
	}
	name := strings.TrimSpace(model.DisplayNameZH)
	summary := strings.TrimSpace(model.SummaryZH)
	if name == "" {
		name = strings.TrimSpace(model.DisplayNameEN)
	}
	if summary == "" {
		summary = strings.TrimSpace(model.SummaryEN)
	}
	return name, summary
}
