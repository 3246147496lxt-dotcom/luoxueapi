package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"unicode"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	maxManagedTranscriptionGroups = 64

	// The managed transcription allowance has one cross-node migration default
	// and a bounded operator contract. One day is the natural ceiling for a daily
	// quota and keeps an accidental extra zero from silently removing protection.
	ManagedTranscriptionDailyAudioSecondsDefault = 20 * 60
	ManagedTranscriptionDailyAudioSecondsMin     = 1
	ManagedTranscriptionDailyAudioSecondsMax     = 24 * 60 * 60
)

// WebChatTranscriptionSettings are the operator-managed parts of Web Chat
// speech-to-text. Process-level safety limits remain in config.yaml; the
// per-user daily allowance is an operational policy and is managed here.
type WebChatTranscriptionSettings struct {
	Enabled               bool    `json:"enabled"`
	Model                 string  `json:"model"`
	GroupIDs              []int64 `json:"group_ids"`
	UserDailyAudioSeconds int     `json:"user_daily_audio_seconds"`
}

type effectiveTranscriptionConfigContextKey struct{}

// WithEffectiveTranscriptionConfig pins one resolved configuration to a
// request. This prevents an administrator update halfway through an upload
// from mixing admission, group selection and forwarding settings.
func WithEffectiveTranscriptionConfig(ctx context.Context, cfg config.TranscriptionConfig) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, effectiveTranscriptionConfigContextKey{}, cloneTranscriptionConfig(cfg))
}

// EffectiveTranscriptionConfigFromContext returns the request-pinned config.
func EffectiveTranscriptionConfigFromContext(ctx context.Context) (config.TranscriptionConfig, bool) {
	if ctx == nil {
		return config.TranscriptionConfig{}, false
	}
	cfg, ok := ctx.Value(effectiveTranscriptionConfigContextKey{}).(config.TranscriptionConfig)
	if !ok {
		return config.TranscriptionConfig{}, false
	}
	return cloneTranscriptionConfig(cfg), true
}

// GetWebChatTranscriptionSettings returns the database-managed control plane.
// When an upgraded installation has not saved it yet, config.yaml is used as a
// one-time compatibility seed and managed=false is returned.
func (s *SettingService) GetWebChatTranscriptionSettings(ctx context.Context) (*WebChatTranscriptionSettings, bool, error) {
	fallback := s.defaultWebChatTranscriptionSettings()
	if s == nil || s.settingRepo == nil {
		return fallback, false, nil
	}

	raw, err := s.settingRepo.GetValue(ctx, SettingKeyWebChatTranscriptionSettings)
	if errors.Is(err, ErrSettingNotFound) {
		return fallback, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("get web chat transcription settings: %w", err)
	}

	// Decode through a pointer-bearing storage shape so a legacy object with no
	// quota can be distinguished from an explicitly stored zero. The former is
	// migrated in memory to one cross-node constant; the latter is corruption and
	// must fail closed instead of silently changing operator intent.
	var stored struct {
		Enabled               bool    `json:"enabled"`
		Model                 string  `json:"model"`
		GroupIDs              []int64 `json:"group_ids"`
		UserDailyAudioSeconds *int    `json:"user_daily_audio_seconds"`
	}
	if err := json.Unmarshal([]byte(raw), &stored); err != nil {
		return nil, true, fmt.Errorf("decode web chat transcription settings: %w", err)
	}
	settings := WebChatTranscriptionSettings{
		Enabled:  stored.Enabled,
		Model:    stored.Model,
		GroupIDs: append([]int64(nil), stored.GroupIDs...),
	}
	if stored.UserDailyAudioSeconds == nil {
		settings.UserDailyAudioSeconds = ManagedTranscriptionDailyAudioSecondsDefault
	} else {
		settings.UserDailyAudioSeconds = *stored.UserDailyAudioSeconds
	}
	if err := normalizeWebChatTranscriptionSettings(&settings); err != nil {
		return nil, true, fmt.Errorf("invalid stored web chat transcription settings: %w", err)
	}
	return &settings, true, nil
}

// SetWebChatTranscriptionSettings persists the mutable STT control plane. All
// instances observe the new value on their next request because reads are made
// from the shared settings repository rather than process memory.
func (s *SettingService) SetWebChatTranscriptionSettings(ctx context.Context, settings *WebChatTranscriptionSettings) error {
	if s == nil || s.settingRepo == nil {
		return infraerrors.ServiceUnavailable("TRANSCRIPTION_SETTINGS_UNAVAILABLE", "transcription settings are unavailable")
	}
	normalized, err := s.ValidateWebChatTranscriptionSettings(ctx, settings)
	if err != nil {
		return err
	}

	data, err := json.Marshal(normalized)
	if err != nil {
		return fmt.Errorf("encode web chat transcription settings: %w", err)
	}
	if err := s.settingRepo.Set(ctx, SettingKeyWebChatTranscriptionSettings, string(data)); err != nil {
		return fmt.Errorf("save web chat transcription settings: %w", err)
	}
	s.notifyUpdated()
	return nil
}

// ValidateWebChatTranscriptionSettings normalizes a proposed control-plane
// update and validates every referenced group without persisting it.
func (s *SettingService) ValidateWebChatTranscriptionSettings(ctx context.Context, settings *WebChatTranscriptionSettings) (*WebChatTranscriptionSettings, error) {
	if settings == nil {
		return nil, infraerrors.BadRequest("INVALID_TRANSCRIPTION_SETTINGS", "transcription settings are required")
	}
	normalized := *settings
	normalized.GroupIDs = append([]int64(nil), settings.GroupIDs...)
	if err := normalizeWebChatTranscriptionSettings(&normalized); err != nil {
		return nil, infraerrors.BadRequest("INVALID_TRANSCRIPTION_SETTINGS", err.Error())
	}
	if normalized.Enabled {
		if err := s.validateWebChatTranscriptionRuntime(); err != nil {
			return nil, err
		}
		if err := s.validateWebChatTranscriptionGroups(ctx, normalized.GroupIDs); err != nil {
			return nil, err
		}
	}
	return &normalized, nil
}

func (s *SettingService) validateWebChatTranscriptionRuntime() error {
	if s == nil || s.cfg == nil {
		return infraerrors.ServiceUnavailable("TRANSCRIPTION_SETTINGS_UNAVAILABLE", "transcription runtime configuration is unavailable")
	}
	if s.cfg.RunMode == config.RunModeSimple {
		return infraerrors.BadRequest("TRANSCRIPTION_UNSUPPORTED_RUN_MODE", "voice transcription is not supported in simple mode")
	}

	requestBodyLimit := s.cfg.Transcription.RequestBodyLimit()
	if s.cfg.Gateway.MaxBodySize < requestBodyLimit {
		return infraerrors.BadRequest(
			"TRANSCRIPTION_BODY_LIMIT_TOO_SMALL",
			fmt.Sprintf("gateway request body limit must be at least %d bytes before voice transcription can be enabled", requestBodyLimit),
		)
	}
	globalBodyLimit := s.cfg.Server.MaxRequestBodySize
	if globalBodyLimit <= 0 {
		globalBodyLimit = s.cfg.Gateway.MaxBodySize
	}
	if globalBodyLimit < requestBodyLimit {
		return infraerrors.BadRequest(
			"TRANSCRIPTION_BODY_LIMIT_TOO_SMALL",
			fmt.Sprintf("server request body limit must be at least %d bytes before voice transcription can be enabled", requestBodyLimit),
		)
	}

	ffprobePath := strings.TrimSpace(s.cfg.Transcription.FFprobePath)
	if ffprobePath == "" {
		return infraerrors.BadRequest("TRANSCRIPTION_FFPROBE_UNAVAILABLE", "ffprobe must be configured before voice transcription can be enabled")
	}
	if _, err := exec.LookPath(ffprobePath); err != nil {
		return infraerrors.BadRequest("TRANSCRIPTION_FFPROBE_UNAVAILABLE", "ffprobe is unavailable on this server")
	}
	return nil
}

// EffectiveTranscriptionConfig overlays the database-managed fields onto the
// deployment-owned limits. A context-pinned value wins for request consistency.
func (s *SettingService) EffectiveTranscriptionConfig(ctx context.Context) (config.TranscriptionConfig, error) {
	if pinned, ok := EffectiveTranscriptionConfigFromContext(ctx); ok {
		return pinned, nil
	}

	base := config.TranscriptionConfig{
		Provider: "openai_compatible",
		Model:    "gpt-4o-mini-transcribe",
	}
	if s != nil && s.cfg != nil {
		base = cloneTranscriptionConfig(s.cfg.Transcription)
	}
	settings, _, err := s.GetWebChatTranscriptionSettings(ctx)
	if err != nil {
		return config.TranscriptionConfig{}, err
	}
	base.Enabled = settings.Enabled
	base.Model = settings.Model
	base.GroupIDs = append([]int64(nil), settings.GroupIDs...)
	base.UserDailyAudioSeconds = settings.UserDailyAudioSeconds
	if s != nil && s.cfg != nil && s.cfg.RunMode == config.RunModeSimple {
		base.Enabled = false
	}
	return base, nil
}

func (s *SettingService) defaultWebChatTranscriptionSettings() *WebChatTranscriptionSettings {
	settings := &WebChatTranscriptionSettings{
		Model:                 "gpt-4o-mini-transcribe",
		UserDailyAudioSeconds: ManagedTranscriptionDailyAudioSecondsDefault,
	}
	if s != nil && s.cfg != nil {
		settings.Enabled = s.cfg.Transcription.Enabled && s.cfg.RunMode != config.RunModeSimple
		settings.Model = strings.TrimSpace(s.cfg.Transcription.Model)
		settings.GroupIDs = append([]int64(nil), s.cfg.Transcription.GroupIDs...)
		settings.UserDailyAudioSeconds = s.cfg.Transcription.UserDailyAudioSeconds
	}
	if settings.Model == "" {
		settings.Model = "gpt-4o-mini-transcribe"
	}
	settings.UserDailyAudioSeconds = managedTranscriptionDailyAudioSeed(settings.UserDailyAudioSeconds)
	return settings
}

func normalizeWebChatTranscriptionSettings(settings *WebChatTranscriptionSettings) error {
	if settings.UserDailyAudioSeconds < ManagedTranscriptionDailyAudioSecondsMin ||
		settings.UserDailyAudioSeconds > ManagedTranscriptionDailyAudioSecondsMax {
		return fmt.Errorf(
			"transcription user_daily_audio_seconds must be between %d and %d",
			ManagedTranscriptionDailyAudioSecondsMin,
			ManagedTranscriptionDailyAudioSecondsMax,
		)
	}
	settings.Model = strings.TrimSpace(settings.Model)
	if settings.Model == "" {
		return errors.New("transcription model must not be empty")
	}
	if len(settings.Model) > 255 {
		return errors.New("transcription model is invalid")
	}
	for _, value := range settings.Model {
		if unicode.IsControl(value) {
			return errors.New("transcription model is invalid")
		}
	}
	if len(settings.GroupIDs) > maxManagedTranscriptionGroups {
		return fmt.Errorf("transcription group_ids must contain at most %d groups", maxManagedTranscriptionGroups)
	}

	seen := make(map[int64]struct{}, len(settings.GroupIDs))
	groups := make([]int64, 0, len(settings.GroupIDs))
	for _, groupID := range settings.GroupIDs {
		if groupID <= 0 {
			return errors.New("transcription group_ids must contain only positive IDs")
		}
		if _, exists := seen[groupID]; exists {
			return fmt.Errorf("transcription group_ids contains duplicate ID %d", groupID)
		}
		seen[groupID] = struct{}{}
		groups = append(groups, groupID)
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i] < groups[j] })
	settings.GroupIDs = groups
	if settings.Enabled && len(settings.GroupIDs) == 0 {
		return errors.New("at least one transcription resource group is required when enabled")
	}
	return nil
}

func managedTranscriptionDailyAudioSeed(value int) int {
	switch {
	case value < ManagedTranscriptionDailyAudioSecondsMin:
		return ManagedTranscriptionDailyAudioSecondsDefault
	case value > ManagedTranscriptionDailyAudioSecondsMax:
		return ManagedTranscriptionDailyAudioSecondsMax
	default:
		return value
	}
}

func (s *SettingService) validateWebChatTranscriptionGroups(ctx context.Context, groupIDs []int64) error {
	if len(groupIDs) == 0 {
		return nil
	}
	if s == nil || s.defaultSubGroupReader == nil {
		return infraerrors.ServiceUnavailable("TRANSCRIPTION_SETTINGS_UNAVAILABLE", "transcription resource group validation is unavailable")
	}
	for _, groupID := range groupIDs {
		group, err := s.defaultSubGroupReader.GetByID(ctx, groupID)
		if err != nil {
			if errors.Is(err, ErrGroupNotFound) {
				return infraerrors.BadRequest("INVALID_TRANSCRIPTION_GROUP", fmt.Sprintf("transcription resource group %d does not exist", groupID))
			}
			return fmt.Errorf("get transcription resource group %d: %w", groupID, err)
		}
		if group == nil || !group.IsActive() || group.Platform != PlatformOpenAI || group.SubscriptionType != SubscriptionTypeStandard {
			return infraerrors.BadRequest("INVALID_TRANSCRIPTION_GROUP", fmt.Sprintf("transcription resource group %d must be an active standard OpenAI group", groupID))
		}
	}
	return nil
}

func cloneTranscriptionConfig(cfg config.TranscriptionConfig) config.TranscriptionConfig {
	cfg.GroupIDs = append([]int64(nil), cfg.GroupIDs...)
	cfg.AcceptedMIMETypes = append([]string(nil), cfg.AcceptedMIMETypes...)
	return cfg
}
