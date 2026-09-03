// Package application orchestrates durable Skill imports without coupling
// source adapters to persistence or HTTP handlers.
package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	domain "github.com/Wei-Shaw/sub2api/internal/service"
	core "github.com/Wei-Shaw/sub2api/internal/skillimport"
	"github.com/robfig/cron/v3"
)

const (
	DefaultScheduleCron     = "0 3 * * *"
	DefaultScheduleTimezone = "Asia/Shanghai"
)

var (
	ErrInvalidInput        = infraerrors.BadRequest("SKILL_IMPORT_INVALID", "skill import input is invalid")
	ErrAdapterNotFound     = infraerrors.BadRequest("SKILL_IMPORT_ADAPTER_NOT_FOUND", "skill import adapter is not registered")
	ErrFeatureDisabled     = infraerrors.ServiceUnavailable("SKILL_IMPORT_DISABLED", "skill import is disabled")
	importCronParser       = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	sourceNamespacePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,159}$`)
)

type AdapterRegistry struct {
	adapters map[string]core.SourceAdapter
}

func NewAdapterRegistry(adapters ...core.SourceAdapter) *AdapterRegistry {
	registry := &AdapterRegistry{adapters: make(map[string]core.SourceAdapter, len(adapters))}
	for _, adapter := range adapters {
		if adapter == nil {
			continue
		}
		kind := strings.ToLower(strings.TrimSpace(adapter.Type()))
		if kind != "" {
			registry.adapters[kind] = adapter
		}
	}
	return registry
}

func (r *AdapterRegistry) Get(kind string) (core.SourceAdapter, bool) {
	if r == nil {
		return nil, false
	}
	adapter, ok := r.adapters[strings.ToLower(strings.TrimSpace(kind))]
	return adapter, ok
}

func (r *AdapterRegistry) Types() []string {
	if r == nil {
		return []string{}
	}
	types := make([]string, 0, len(r.adapters))
	for kind := range r.adapters {
		types = append(types, kind)
	}
	sort.Strings(types)
	return types
}

type AdapterDescriptor struct {
	Type    string `json:"type"`
	Version string `json:"version"`
}

func (s *Service) AdapterDescriptors() []AdapterDescriptor {
	if s == nil || s.registry == nil {
		return []AdapterDescriptor{}
	}
	types := s.registry.Types()
	result := make([]AdapterDescriptor, 0, len(types))
	for _, kind := range types {
		adapter, ok := s.registry.Get(kind)
		if ok {
			result = append(result, AdapterDescriptor{Type: kind, Version: adapter.Version()})
		}
	}
	return result
}

type Service struct {
	repo     domain.SkillImportRepository
	registry *AdapterRegistry
	cfg      config.SkillImportConfig
	now      func() time.Time
}

func NewService(repo domain.SkillImportRepository, registry *AdapterRegistry, cfg *config.Config) *Service {
	settings := config.SkillImportConfig{}
	if cfg != nil {
		settings = cfg.SkillImport
	}
	return &Service{repo: repo, registry: registry, cfg: settings, now: time.Now}
}

func (s *Service) Enabled() bool {
	return s != nil && s.repo != nil && s.registry != nil && s.cfg.Enabled
}

func (s *Service) requireEnabled() error {
	if !s.Enabled() {
		return ErrFeatureDisabled
	}
	return nil
}

type SourceInput struct {
	Name            string          `json:"name"`
	Adapter         string          `json:"adapter"`
	Namespace       string          `json:"namespace"`
	BaseURL         string          `json:"base_url"`
	SourceConfig    json.RawMessage `json:"source_config"`
	CatalogPriority int             `json:"catalog_priority"`
	Enabled         bool            `json:"enabled"`
}

func (s *Service) CreateSource(ctx context.Context, input SourceInput, actorID *int64) (*domain.SkillImportSource, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	source, err := s.normalizeSource(input)
	if err != nil {
		return nil, err
	}
	source.CreatedBy, source.UpdatedBy = actorID, actorID
	if err := s.repo.CreateSource(ctx, source); err != nil {
		return nil, err
	}
	return source, nil
}

func (s *Service) UpdateSource(ctx context.Context, id int64, input SourceInput, actorID *int64) (*domain.SkillImportSource, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	current, err := s.repo.GetSource(ctx, id)
	if err != nil {
		return nil, err
	}
	input.Adapter = current.Adapter
	input.Namespace = current.Namespace
	normalized, err := s.normalizeSource(input)
	if err != nil {
		return nil, err
	}
	normalized.ID = id
	normalized.CreatedBy = current.CreatedBy
	normalized.CreatedAt = current.CreatedAt
	normalized.UpdatedBy = actorID
	if err := s.repo.UpdateSource(ctx, normalized); err != nil {
		return nil, err
	}
	return s.repo.GetSource(ctx, id)
}

func (s *Service) GetSource(ctx context.Context, id int64) (*domain.SkillImportSource, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	source, err := s.repo.GetSource(ctx, id)
	if err != nil {
		return nil, err
	}
	return redactSourceForResponse(source), nil
}

func (s *Service) ListSources(ctx context.Context, filter domain.SkillImportListFilter) (*domain.SkillImportListResult[domain.SkillImportSource], error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	filter = normalizeListFilter(filter)
	items, total, err := s.repo.ListSources(ctx, filter)
	if err != nil {
		return nil, err
	}
	for index := range items {
		items[index] = *redactSourceForResponse(&items[index])
	}
	return listResult(items, total, filter), nil
}

func (s *Service) normalizeSource(input SourceInput) (*domain.SkillImportSource, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Adapter = strings.ToLower(strings.TrimSpace(input.Adapter))
	input.Namespace = strings.TrimSpace(input.Namespace)
	input.BaseURL = strings.TrimRight(strings.TrimSpace(input.BaseURL), "/")
	if input.Name == "" || len([]rune(input.Name)) > 120 || input.Adapter == "" || !sourceNamespacePattern.MatchString(input.Namespace) {
		return nil, ErrInvalidInput
	}
	if input.CatalogPriority < 0 || input.CatalogPriority > 1000000 {
		return nil, ErrInvalidInput.WithMetadata(map[string]string{"catalog_priority": "must be between 0 and 1000000"})
	}
	if input.BaseURL != "" {
		parsed, err := url.Parse(input.BaseURL)
		if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil || parsed.Fragment != "" || len(input.BaseURL) > 2048 {
			return nil, ErrInvalidInput.WithMetadata(map[string]string{"base_url": "must be a canonical HTTPS URL"})
		}
	}
	adapter, ok := s.registry.Get(input.Adapter)
	if !ok {
		return nil, ErrAdapterNotFound.WithMetadata(map[string]string{"adapter": input.Adapter})
	}
	configJSON, err := normalizeJSONObject(input.SourceConfig)
	if err != nil {
		return nil, ErrInvalidInput.WithMetadata(map[string]string{"source_config": err.Error()})
	}
	if hasInlineSourceConfig(configJSON) {
		return nil, ErrInvalidInput.WithMetadata(map[string]string{
			"source_config": "inline upload data is only allowed on a single run upload",
		})
	}
	if err := adapter.ValidateConfig(configJSON); err != nil {
		return nil, ErrInvalidInput.WithMetadata(map[string]string{"source_config": err.Error()})
	}
	return &domain.SkillImportSource{
		Name: input.Name, Adapter: input.Adapter, Namespace: input.Namespace,
		BaseURL: input.BaseURL, SourceConfig: configJSON,
		CatalogPriority: input.CatalogPriority, Enabled: input.Enabled,
	}, nil
}

func hasInlineSourceConfig(raw json.RawMessage) bool {
	var value map[string]json.RawMessage
	if json.Unmarshal(raw, &value) != nil {
		return false
	}
	for key := range value {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(key)), "inline_data_") {
			return true
		}
	}
	return false
}

func redactSourceForResponse(source *domain.SkillImportSource) *domain.SkillImportSource {
	if source == nil {
		return nil
	}
	redacted := *source
	redacted.SourceConfig = append(json.RawMessage(nil), source.SourceConfig...)
	var value map[string]json.RawMessage
	if json.Unmarshal(redacted.SourceConfig, &value) != nil {
		return &redacted
	}
	found := false
	for key := range value {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(key)), "inline_data_") {
			delete(value, key)
			found = true
		}
	}
	if !found {
		return &redacted
	}
	value["inline_data_redacted"] = json.RawMessage(`true`)
	if encoded, err := json.Marshal(value); err == nil {
		redacted.SourceConfig = encoded
	}
	return &redacted
}

type ScheduleInput struct {
	SourceID       int64           `json:"source_id"`
	Name           string          `json:"name"`
	Enabled        bool            `json:"enabled"`
	CronExpression string          `json:"cron_expression"`
	Timezone       string          `json:"timezone"`
	Selection      json.RawMessage `json:"selection"`
	RunConfig      json.RawMessage `json:"run_config"`
	PublishPolicy  string          `json:"publish_policy"`
	MetadataPolicy string          `json:"metadata_policy"`
}

func (s *Service) CreateSchedule(ctx context.Context, input ScheduleInput, actorID *int64) (*domain.SkillImportSchedule, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	schedule, err := s.normalizeSchedule(ctx, input)
	if err != nil {
		return nil, err
	}
	schedule.CreatedBy, schedule.UpdatedBy = actorID, actorID
	if err := s.repo.CreateSchedule(ctx, schedule); err != nil {
		return nil, err
	}
	return schedule, nil
}

func (s *Service) UpdateSchedule(ctx context.Context, id int64, input ScheduleInput, actorID *int64) (*domain.SkillImportSchedule, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	current, err := s.repo.GetSchedule(ctx, id)
	if err != nil {
		return nil, err
	}
	// A schedule's source is part of its durable identity. Changing it after a
	// run exists would make last_run_id point at a different source and would
	// also rewrite the meaning of the schedule's audit history. Create a new
	// schedule when an operator wants to target another source.
	input.SourceID = current.SourceID
	schedule, err := s.normalizeSchedule(ctx, input)
	if err != nil {
		return nil, err
	}
	schedule.ID = id
	schedule.LastRunAt, schedule.LastRunID = current.LastRunAt, current.LastRunID
	schedule.CreatedBy, schedule.CreatedAt, schedule.UpdatedBy = current.CreatedBy, current.CreatedAt, actorID
	if err := s.repo.UpdateSchedule(ctx, schedule); err != nil {
		return nil, err
	}
	return s.repo.GetSchedule(ctx, id)
}

func (s *Service) GetSchedule(ctx context.Context, id int64) (*domain.SkillImportSchedule, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	return s.repo.GetSchedule(ctx, id)
}

func (s *Service) ListSchedules(ctx context.Context, filter domain.SkillImportListFilter) (*domain.SkillImportListResult[domain.SkillImportSchedule], error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	filter = normalizeListFilter(filter)
	items, total, err := s.repo.ListSchedules(ctx, filter)
	if err != nil {
		return nil, err
	}
	return listResult(items, total, filter), nil
}

func (s *Service) normalizeSchedule(ctx context.Context, input ScheduleInput) (*domain.SkillImportSchedule, error) {
	if input.SourceID <= 0 {
		return nil, ErrInvalidInput.WithMetadata(map[string]string{"source_id": "must be positive"})
	}
	if _, err := s.repo.GetSource(ctx, input.SourceID); err != nil {
		return nil, err
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || len([]rune(input.Name)) > 120 {
		return nil, ErrInvalidInput.WithMetadata(map[string]string{"name": "is required and must be at most 120 characters"})
	}
	input.CronExpression = strings.TrimSpace(input.CronExpression)
	if input.CronExpression == "" {
		input.CronExpression = DefaultScheduleCron
	}
	input.Timezone = strings.TrimSpace(input.Timezone)
	if input.Timezone == "" {
		input.Timezone = DefaultScheduleTimezone
	}
	selection, err := normalizeJSONObject(input.Selection)
	if err != nil {
		return nil, ErrInvalidInput.WithMetadata(map[string]string{"selection": err.Error()})
	}
	runConfig, err := normalizeJSONObject(input.RunConfig)
	if err != nil {
		return nil, ErrInvalidInput.WithMetadata(map[string]string{"run_config": err.Error()})
	}
	if err := validateSkillImportRunConfig(runConfig); err != nil {
		return nil, ErrInvalidInput.WithMetadata(map[string]string{"run_config": err.Error()})
	}
	input.PublishPolicy = strings.ToLower(strings.TrimSpace(input.PublishPolicy))
	if input.PublishPolicy == "" {
		input.PublishPolicy = domain.SkillImportPublishPolicyAuto
	}
	if input.PublishPolicy != domain.SkillImportPublishPolicyAuto && input.PublishPolicy != domain.SkillImportPublishPolicyReview {
		return nil, ErrInvalidInput.WithMetadata(map[string]string{"publish_policy": "must be auto_publish or review"})
	}
	input.MetadataPolicy = strings.ToLower(strings.TrimSpace(input.MetadataPolicy))
	if input.MetadataPolicy == "" {
		input.MetadataPolicy = domain.SkillImportMetadataPolicyRefresh
	}
	if input.MetadataPolicy != domain.SkillImportMetadataPolicyRefresh && input.MetadataPolicy != domain.SkillImportMetadataPolicyCreateOnly {
		return nil, ErrInvalidInput.WithMetadata(map[string]string{"metadata_policy": "must be refresh or create_only"})
	}
	next, err := nextScheduleTime(input.CronExpression, input.Timezone, s.now().UTC())
	if err != nil {
		return nil, ErrInvalidInput.WithMetadata(map[string]string{"cron_expression": err.Error()})
	}
	if !input.Enabled {
		next = nil
	}
	return &domain.SkillImportSchedule{
		SourceID: input.SourceID, Name: input.Name, Enabled: input.Enabled,
		CronExpression: input.CronExpression, Timezone: input.Timezone,
		Selection: selection, RunConfig: runConfig, PublishPolicy: input.PublishPolicy,
		MetadataPolicy: input.MetadataPolicy, NextRunAt: next,
	}, nil
}

func nextScheduleTime(expression, timezone string, from time.Time) (*time.Time, error) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone: %w", err)
	}
	schedule, err := importCronParser.Parse(expression)
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression: %w", err)
	}
	next := schedule.Next(from.In(location)).UTC()
	return &next, nil
}

func normalizeJSONObject(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "" || strings.TrimSpace(string(raw)) == "null" {
		return json.RawMessage(`{}`), nil
	}
	var value map[string]any
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, errors.New("must be a JSON object")
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return nil, errors.New("must contain exactly one JSON object")
	}
	if value == nil {
		return nil, errors.New("must be a JSON object")
	}
	normalized, err := json.Marshal(value)
	if err != nil {
		return nil, errors.New("cannot encode JSON object")
	}
	return normalized, nil
}

// mergeJSONObjects overlays trusted per-run adapter values on the durable
// source configuration. This is used by bounded manifest uploads so source
// policy such as allowed_hosts remains effective while the uploaded bytes are
// frozen only in the run configuration.
func mergeJSONObjects(base, overlay json.RawMessage) (json.RawMessage, error) {
	baseNormalized, err := normalizeJSONObject(base)
	if err != nil {
		return nil, err
	}
	overlayNormalized, err := normalizeJSONObject(overlay)
	if err != nil {
		return nil, err
	}
	var baseValue, overlayValue map[string]any
	if err := json.Unmarshal(baseNormalized, &baseValue); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(overlayNormalized, &overlayValue); err != nil {
		return nil, err
	}
	for key, value := range overlayValue {
		baseValue[key] = value
	}
	return json.Marshal(baseValue)
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var trailing any
	err := decoder.Decode(&trailing)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("unexpected trailing JSON value")
	}
	return err
}

func normalizeListFilter(filter domain.SkillImportListFilter) domain.SkillImportListFilter {
	filter.Search = strings.TrimSpace(filter.Search)
	filter.Status = strings.TrimSpace(filter.Status)
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
	return filter
}

func listResult[T any](items []T, total int64, filter domain.SkillImportListFilter) *domain.SkillImportListResult[T] {
	pages := int((total + int64(filter.PageSize) - 1) / int64(filter.PageSize))
	if pages < 1 {
		pages = 1
	}
	if items == nil {
		items = []T{}
	}
	return &domain.SkillImportListResult[T]{
		Items: items, Total: total, Page: filter.Page, PageSize: filter.PageSize, Pages: pages,
	}
}
