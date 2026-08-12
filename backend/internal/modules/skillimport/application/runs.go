package application

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	domain "github.com/Wei-Shaw/sub2api/internal/service"
	core "github.com/Wei-Shaw/sub2api/internal/skillimport"
)

const validatorRulesetVersion = "skill-archive-v1"

type RunInput struct {
	SourceID       int64           `json:"source_id"`
	Mode           string          `json:"mode"`
	Selection      json.RawMessage `json:"selection"`
	RunConfig      json.RawMessage `json:"run_config"`
	PublishPolicy  string          `json:"publish_policy"`
	MetadataPolicy string          `json:"metadata_policy"`
	// AdapterConfigOverride is populated only by trusted application entry
	// points such as the bounded manifest upload handler; it is never decoded
	// from the public JSON run-creation body.
	AdapterConfigOverride json.RawMessage `json:"-"`
}

// FrozenRunConfig is copied into every run. A deployed adapter or normalizer
// upgrade therefore cannot silently resume an old run under different rules.
type FrozenRunConfig struct {
	Selection         json.RawMessage `json:"selection"`
	AdapterConfig     json.RawMessage `json:"adapter_config"`
	RunConfig         json.RawMessage `json:"run_config"`
	PublishPolicy     string          `json:"publish_policy"`
	MetadataPolicy    string          `json:"metadata_policy"`
	AdapterType       string          `json:"adapter_type"`
	AdapterVersion    string          `json:"adapter_version"`
	BaseURL           string          `json:"base_url"`
	SourceNamespace   string          `json:"source_namespace"`
	CatalogPriority   int             `json:"catalog_priority"`
	NormalizerVersion string          `json:"normalizer_version"`
	ValidatorVersion  string          `json:"validator_version"`
}

func (s *Service) CreateRun(ctx context.Context, input RunInput, actorID *int64, idempotencyKey string) (*domain.SkillImportRun, error) {
	return s.createRun(ctx, input, nil, domain.SkillImportTriggerManual, nil, actorID, idempotencyKey)
}

func (s *Service) RunSchedule(ctx context.Context, scheduleID int64, actorID *int64, idempotencyKey string) (*domain.SkillImportRun, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	schedule, err := s.repo.GetSchedule(ctx, scheduleID)
	if err != nil {
		return nil, err
	}
	input := RunInput{
		SourceID: schedule.SourceID, Selection: schedule.Selection,
		RunConfig: schedule.RunConfig, PublishPolicy: schedule.PublishPolicy,
		MetadataPolicy: schedule.MetadataPolicy,
	}
	if schedule.PublishPolicy == domain.SkillImportPublishPolicyReview {
		input.Mode = domain.SkillImportModeReview
	} else {
		input.Mode = domain.SkillImportModeAutoPublish
	}
	return s.createRun(ctx, input, &scheduleID, domain.SkillImportTriggerManual, nil, actorID, idempotencyKey)
}

func (s *Service) CreateScheduledRun(ctx context.Context, schedule *domain.SkillImportSchedule, scheduledFor time.Time) (*domain.SkillImportRun, error) {
	if schedule == nil || schedule.ID <= 0 {
		return nil, ErrInvalidInput.WithMetadata(map[string]string{"schedule": "is required"})
	}
	mode := domain.SkillImportModeAutoPublish
	if schedule.PublishPolicy == domain.SkillImportPublishPolicyReview {
		mode = domain.SkillImportModeReview
	}
	input := RunInput{
		SourceID: schedule.SourceID, Mode: mode, Selection: schedule.Selection,
		RunConfig: schedule.RunConfig, PublishPolicy: schedule.PublishPolicy,
		MetadataPolicy: schedule.MetadataPolicy,
	}
	idempotencyKey := fmt.Sprintf("schedule:%d:%s", schedule.ID, scheduledFor.UTC().Format(time.RFC3339))
	return s.createRun(ctx, input, &schedule.ID, domain.SkillImportTriggerScheduled, &scheduledFor, nil, idempotencyKey)
}

func (s *Service) createRun(
	ctx context.Context,
	input RunInput,
	scheduleID *int64,
	trigger string,
	scheduledFor *time.Time,
	actorID *int64,
	idempotencyKey string,
) (*domain.SkillImportRun, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	if input.SourceID <= 0 {
		return nil, ErrInvalidInput.WithMetadata(map[string]string{"source_id": "must be positive"})
	}
	source, err := s.repo.GetSource(ctx, input.SourceID)
	if err != nil {
		return nil, err
	}
	if !source.Enabled {
		return nil, infraerrors.Conflict("SKILL_IMPORT_SOURCE_DISABLED", "skill import source is disabled")
	}
	adapter, ok := s.registry.Get(source.Adapter)
	if !ok {
		return nil, ErrAdapterNotFound.WithMetadata(map[string]string{"adapter": source.Adapter})
	}
	adapterConfig := source.SourceConfig
	if len(input.AdapterConfigOverride) > 0 {
		adapterConfig, err = mergeJSONObjects(source.SourceConfig, input.AdapterConfigOverride)
		if err != nil {
			return nil, ErrInvalidInput.WithMetadata(map[string]string{"adapter_config": err.Error()})
		}
	}
	adapterConfig, err = annotateInlineAdapterConfig(adapterConfig)
	if err != nil {
		return nil, ErrInvalidInput.WithMetadata(map[string]string{"adapter_config": err.Error()})
	}
	if err := adapter.ValidateConfig(adapterConfig); err != nil {
		return nil, ErrInvalidInput.WithMetadata(map[string]string{"adapter_config": err.Error()})
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
	input.Mode = strings.ToLower(strings.TrimSpace(input.Mode))
	if input.Mode == "" {
		if input.PublishPolicy == domain.SkillImportPublishPolicyReview {
			input.Mode = domain.SkillImportModeReview
		} else {
			input.Mode = domain.SkillImportModeAutoPublish
		}
	}
	if input.Mode != domain.SkillImportModeAutoPublish && input.Mode != domain.SkillImportModeReview && input.Mode != domain.SkillImportModeDryRun {
		return nil, ErrInvalidInput.WithMetadata(map[string]string{"mode": "must be auto_publish, review or dry_run"})
	}
	if input.Mode != domain.SkillImportModeDryRun && input.Mode != input.PublishPolicy {
		return nil, ErrInvalidInput.WithMetadata(map[string]string{
			"mode": "must match publish_policy unless mode is dry_run",
		})
	}
	frozen := FrozenRunConfig{
		Selection: selection, AdapterConfig: adapterConfig, RunConfig: runConfig,
		PublishPolicy: input.PublishPolicy, MetadataPolicy: input.MetadataPolicy,
		AdapterType: source.Adapter, AdapterVersion: adapter.Version(),
		BaseURL: source.BaseURL, SourceNamespace: source.Namespace,
		CatalogPriority:   source.CatalogPriority,
		NormalizerVersion: core.CoreVersion, ValidatorVersion: validatorRulesetVersion,
	}
	frozenJSON, err := json.Marshal(frozen)
	if err != nil {
		return nil, err
	}
	run := &domain.SkillImportRun{
		SourceID: input.SourceID, ScheduleID: scheduleID, TriggerType: trigger,
		Mode: input.Mode, Status: domain.SkillImportRunStatusQueued,
		RequestConfig: frozenJSON, Snapshot: json.RawMessage(`{}`),
		ScheduledFor: scheduledFor, Counts: domain.SkillImportRunCounts{Requested: requestedCount(selection)},
		CreatedBy: actorID,
	}
	if err := s.repo.CreateRun(ctx, run, hashIdempotencyKey(idempotencyKey)); err != nil {
		return nil, err
	}
	return redactRunForResponse(run), nil
}

type skillImportAutoPublishGateConfig struct {
	RequireAllValid        *bool `json:"require_all_valid"`
	AllowLicenseUnverified *bool `json:"allow_license_unverified"`
	MaxBlockedItems        *int  `json:"max_blocked_items"`
	MaxFailedItems         *int  `json:"max_failed_items"`
}

type skillImportRunPolicyConfig struct {
	SafeGate        *bool                             `json:"safe_gate"`
	Concurrency     *int                              `json:"concurrency"`
	AutoPublishGate *skillImportAutoPublishGateConfig `json:"auto_publish_gate"`
}

func parseSkillImportRunConfig(raw json.RawMessage) (skillImportRunPolicyConfig, error) {
	var value skillImportRunPolicyConfig
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return value, err
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return value, err
	}
	return value, nil
}

func validateSkillImportRunConfig(raw json.RawMessage) error {
	value, err := parseSkillImportRunConfig(raw)
	if err != nil {
		return fmt.Errorf("has invalid policy fields: %w", err)
	}
	if value.Concurrency != nil && (*value.Concurrency < 1 || *value.Concurrency > 16) {
		return fmt.Errorf("concurrency must be between 1 and 16 when provided")
	}
	if value.AutoPublishGate != nil {
		if value.AutoPublishGate.MaxBlockedItems != nil && *value.AutoPublishGate.MaxBlockedItems < 0 {
			return fmt.Errorf("auto_publish_gate.max_blocked_items cannot be negative")
		}
		if value.AutoPublishGate.MaxFailedItems != nil && *value.AutoPublishGate.MaxFailedItems < 0 {
			return fmt.Errorf("auto_publish_gate.max_failed_items cannot be negative")
		}
	}
	return nil
}

func (s *Service) GetRun(ctx context.Context, id int64) (*domain.SkillImportRun, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	run, err := s.repo.GetRun(ctx, id)
	if err != nil {
		return nil, err
	}
	return redactRunForResponse(run), nil
}

func (s *Service) ListRuns(ctx context.Context, filter domain.SkillImportListFilter) (*domain.SkillImportListResult[domain.SkillImportRun], error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	filter = normalizeListFilter(filter)
	items, total, err := s.repo.ListRuns(ctx, filter)
	if err != nil {
		return nil, err
	}
	for index := range items {
		items[index] = *redactRunForResponse(&items[index])
	}
	return listResult(items, total, filter), nil
}

func (s *Service) ListRunItems(ctx context.Context, runID int64, filter domain.SkillImportListFilter) (*domain.SkillImportListResult[domain.SkillImportRunItem], error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	filter = normalizeListFilter(filter)
	filter.RunID = &runID
	items, total, err := s.repo.ListRunItems(ctx, filter)
	if err != nil {
		return nil, err
	}
	return listResult(items, total, filter), nil
}

func (s *Service) ListRunEvents(ctx context.Context, runID int64, filter domain.SkillImportListFilter) (*domain.SkillImportListResult[domain.SkillImportEvent], error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	filter = normalizeListFilter(filter)
	filter.RunID = &runID
	items, total, err := s.repo.ListEvents(ctx, filter)
	if err != nil {
		return nil, err
	}
	return listResult(items, total, filter), nil
}

func (s *Service) CancelRun(ctx context.Context, runID, actorID int64) (*domain.SkillImportRun, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	if runID <= 0 || actorID <= 0 {
		return nil, ErrInvalidInput
	}
	if _, err := s.repo.RequestRunCancellation(ctx, runID, actorID); err != nil {
		return nil, err
	}
	run, err := s.repo.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	return redactRunForResponse(run), nil
}

func (s *Service) RetryFailedItems(ctx context.Context, runID int64, itemIDs []int64) (*domain.SkillImportRun, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	for _, id := range itemIDs {
		if id <= 0 {
			return nil, ErrInvalidInput.WithMetadata(map[string]string{"item_ids": "must contain only positive IDs"})
		}
	}
	if _, err := s.repo.ResetFailedItems(ctx, runID, itemIDs); err != nil {
		return nil, err
	}
	run, err := s.repo.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	return redactRunForResponse(run), nil
}

func redactRunForResponse(run *domain.SkillImportRun) *domain.SkillImportRun {
	if run == nil {
		return nil
	}
	redacted := *run
	redacted.RequestConfig = redactInlineAdapterConfig(run.RequestConfig)
	redacted.Snapshot = append(json.RawMessage(nil), run.Snapshot...)
	return &redacted
}

func redactInlineAdapterConfig(raw json.RawMessage) json.RawMessage {
	cloned := append(json.RawMessage(nil), raw...)
	var frozen map[string]json.RawMessage
	if json.Unmarshal(cloned, &frozen) != nil {
		return cloned
	}
	adapterRaw, exists := frozen["adapter_config"]
	if !exists {
		return cloned
	}
	var adapter map[string]any
	if json.Unmarshal(adapterRaw, &adapter) != nil {
		return cloned
	}
	encoded := ""
	payloadKey := ""
	for key, rawValue := range adapter {
		if strings.EqualFold(key, "inline_data_base64") {
			encoded, _ = rawValue.(string)
			payloadKey = key
			break
		}
	}
	if payloadKey == "" {
		return cloned
	}
	delete(adapter, payloadKey)
	adapter["inline_data_redacted"] = true
	if _, recorded := adapter["inline_data_encoded_bytes"]; !recorded {
		adapter["inline_data_encoded_bytes"] = len(encoded)
	}
	if _, recorded := adapter["inline_data_sha256"]; !recorded {
		if decoded, err := base64.StdEncoding.DecodeString(encoded); err == nil {
			digest := sha256.Sum256(decoded)
			adapter["inline_data_byte_size"] = len(decoded)
			adapter["inline_data_sha256"] = hex.EncodeToString(digest[:])
		} else {
			digest := sha256.Sum256([]byte(encoded))
			adapter["inline_data_encoded_sha256"] = hex.EncodeToString(digest[:])
		}
	}
	adapterJSON, err := json.Marshal(adapter)
	if err != nil {
		return cloned
	}
	frozen["adapter_config"] = adapterJSON
	result, err := json.Marshal(frozen)
	if err != nil {
		return cloned
	}
	return result
}

func annotateInlineAdapterConfig(raw json.RawMessage) (json.RawMessage, error) {
	var adapter map[string]any
	if err := json.Unmarshal(raw, &adapter); err != nil {
		return nil, err
	}
	encoded, exists := adapter["inline_data_base64"].(string)
	if !exists {
		return append(json.RawMessage(nil), raw...), nil
	}
	if len(encoded) > base64.StdEncoding.EncodedLen(core.MaxInlineManifestBytes) {
		return nil, errors.New("inline_data_base64 exceeds the encoded resource limit")
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(encoded)
	}
	if err != nil {
		return nil, errors.New("inline_data_base64 is not valid base64")
	}
	if len(decoded) == 0 || len(decoded) > core.MaxInlineManifestBytes {
		return nil, errors.New("inline_data_base64 exceeds the decoded resource limit")
	}
	digest := sha256.Sum256(decoded)
	digestHex := hex.EncodeToString(digest[:])
	adapter["inline_data_encoded_bytes"] = len(encoded)
	adapter["inline_data_byte_size"] = len(decoded)
	adapter["inline_data_sha256"] = digestHex
	if configured, _ := adapter["digest"].(string); strings.TrimSpace(configured) == "" {
		adapter["digest"] = "sha256:" + digestHex
	}
	return json.Marshal(adapter)
}

func (s *Service) PublishRun(ctx context.Context, runID int64, itemIDs []int64, actorID *int64) (*domain.SkillImportPublishResult, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	if len(itemIDs) == 0 {
		var err error
		itemIDs, err = s.repo.ListEligibleItemIDs(ctx, runID)
		if err != nil {
			return nil, err
		}
	} else {
		seen := make(map[int64]struct{}, len(itemIDs))
		for _, id := range itemIDs {
			if id <= 0 {
				return nil, ErrInvalidInput.WithMetadata(map[string]string{"item_ids": "must contain only positive IDs"})
			}
			if _, duplicate := seen[id]; duplicate {
				return nil, ErrInvalidInput.WithMetadata(map[string]string{"item_ids": "must not contain duplicate IDs"})
			}
			seen[id] = struct{}{}
		}
	}
	if len(itemIDs) == 0 {
		return nil, domain.ErrSkillImportPublishInvalid.WithMetadata(map[string]string{"items": "no eligible prepared items"})
	}
	return s.repo.PublishEligibleItems(ctx, runID, itemIDs, "", actorID)
}

func requestedCount(selection json.RawMessage) int {
	var value struct {
		Limit int `json:"limit"`
	}
	if json.Unmarshal(selection, &value) == nil && value.Limit > 0 {
		return value.Limit
	}
	return 0
}

func hashIdempotencyKey(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}
