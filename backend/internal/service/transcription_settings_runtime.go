package service

import (
	"context"
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

const maxWebChatTranscriptionModelOptions = 256

// WebChatTranscriptionModelOption is a non-secret admin catalog entry derived
// from currently schedulable audio-capable API-key accounts.
type WebChatTranscriptionModelOption struct {
	ID                  string `json:"id"`
	AvailableGroupCount int    `json:"available_group_count"`
	AvailabilityChecked bool   `json:"availability_checked"`
}

// EffectiveTranscriptionConfig resolves the shared database control plane and
// overlays it onto deployment-owned safety limits.
func (s *OpenAIGatewayService) EffectiveTranscriptionConfig(ctx context.Context) (config.TranscriptionConfig, error) {
	if pinned, ok := EffectiveTranscriptionConfigFromContext(ctx); ok {
		return pinned, nil
	}
	if s != nil && s.settingService != nil {
		cfg, err := s.settingService.EffectiveTranscriptionConfig(ctx)
		if err != nil {
			return config.TranscriptionConfig{}, err
		}
		// Settings are shared by every instance, while ffprobe and outer HTTP
		// limits are node-local. Re-check readiness here so a heterogeneous or
		// partially repaired node never advertises or accepts work it cannot run.
		if cfg.Enabled {
			if err := s.settingService.validateWebChatTranscriptionRuntime(); err != nil {
				return config.TranscriptionConfig{}, err
			}
		}
		return cfg, nil
	}
	if s != nil && s.cfg != nil {
		cfg := cloneTranscriptionConfig(s.cfg.Transcription)
		if s.cfg.RunMode == config.RunModeSimple {
			cfg.Enabled = false
		}
		return cfg, nil
	}
	return config.TranscriptionConfig{}, nil
}

// ListWebChatTranscriptionModelOptions returns models advertised by active,
// schedulable API-key accounts that explicitly opt into audio transcription.
// The current model is always included so an unavailable selection remains
// visible and recoverable in the admin UI.
func (s *OpenAIGatewayService) ListWebChatTranscriptionModelOptions(
	ctx context.Context,
	groupIDs []int64,
	currentModel string,
) ([]WebChatTranscriptionModelOption, error) {
	candidates := make(map[string]struct{})
	currentModel = strings.TrimSpace(currentModel)
	if currentModel != "" {
		candidates[currentModel] = struct{}{}
	}

	uniqueGroups := make([]int64, 0, len(groupIDs))
	seenGroups := make(map[int64]struct{}, len(groupIDs))
	for _, groupID := range groupIDs {
		if groupID <= 0 {
			continue
		}
		if _, exists := seenGroups[groupID]; exists {
			continue
		}
		seenGroups[groupID] = struct{}{}
		uniqueGroups = append(uniqueGroups, groupID)

		accounts, err := s.listSchedulableAccounts(ctx, &groupID, PlatformOpenAI)
		if err != nil {
			return nil, err
		}
		for i := range accounts {
			account := &accounts[i]
			if account.Type != AccountTypeAPIKey || !account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityAudioTranscriptions) {
				continue
			}
			for model := range account.GetModelMapping() {
				model = strings.TrimSpace(model)
				if model == "" || strings.Contains(model, "*") {
					continue
				}
				candidates[model] = struct{}{}
			}
		}
	}

	models := make([]string, 0, len(candidates))
	for model := range candidates {
		models = append(models, model)
	}
	sort.Strings(models)
	if len(models) > maxWebChatTranscriptionModelOptions {
		models = append([]string(nil), models[:maxWebChatTranscriptionModelOptions]...)
		currentIncluded := false
		for _, model := range models {
			if model == currentModel {
				currentIncluded = true
				break
			}
		}
		if currentModel != "" && !currentIncluded {
			models[len(models)-1] = currentModel
			sort.Strings(models)
		}
	}

	currentAvailableGroups := 0
	if currentModel != "" {
		for _, groupID := range uniqueGroups {
			available, err := s.HasSchedulableTranscriptionAccount(ctx, groupID, currentModel)
			if err != nil {
				return nil, err
			}
			if available {
				currentAvailableGroups++
			}
		}
	}

	options := make([]WebChatTranscriptionModelOption, 0, len(models))
	for _, model := range models {
		// Only the saved model gets an exact scheduler check. Other entries are
		// bounded discovery suggestions that validate on save, avoiding
		// an unbounded models × groups fan-out on the admin read path.
		availableGroups := 0
		availabilityChecked := false
		if model == currentModel {
			availableGroups = currentAvailableGroups
			availabilityChecked = true
		}
		options = append(options, WebChatTranscriptionModelOption{
			ID:                  model,
			AvailableGroupCount: availableGroups,
			AvailabilityChecked: availabilityChecked,
		})
	}
	return options, nil
}

// HasAvailableWebChatTranscriptionRoute verifies that at least one configured
// group can currently serve the selected model.
func (s *OpenAIGatewayService) HasAvailableWebChatTranscriptionRoute(ctx context.Context, groupIDs []int64, model string) (bool, error) {
	for _, groupID := range groupIDs {
		available, err := s.HasSchedulableTranscriptionAccount(ctx, groupID, model)
		if err != nil {
			return false, err
		}
		if available {
			return true, nil
		}
	}
	return false, nil
}
