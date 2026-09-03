package admin

import (
	"context"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type webChatTranscriptionLimitsPayload struct {
	MaxUploadBytes        int64 `json:"max_upload_bytes"`
	MaxDurationSeconds    int   `json:"max_duration_seconds"`
	MaxConcurrentGlobal   int   `json:"max_concurrent_global"`
	MaxConcurrentPerUser  int   `json:"max_concurrent_per_user"`
	UserRequestsPerMinute int   `json:"user_requests_per_minute"`
	UserDailyAudioSeconds int   `json:"user_daily_audio_seconds"`
	RequestTimeoutSeconds int   `json:"request_timeout_seconds"`
}

type webChatTranscriptionSettingsPayload struct {
	Enabled                bool                                      `json:"enabled"`
	Provider               string                                    `json:"provider"`
	Model                  string                                    `json:"model"`
	GroupIDs               []int64                                   `json:"group_ids"`
	UserDailyAudioSeconds  int                                       `json:"user_daily_audio_seconds"`
	DailyAudioSecondsMin   int                                       `json:"daily_audio_seconds_min"`
	DailyAudioSecondsMax   int                                       `json:"daily_audio_seconds_max"`
	Managed                bool                                      `json:"managed"`
	RuntimeReady           bool                                      `json:"runtime_ready"`
	CatalogAvailable       bool                                      `json:"catalog_available"`
	SelectedModelAvailable bool                                      `json:"selected_model_available"`
	ModelOptions           []service.WebChatTranscriptionModelOption `json:"model_options"`
	Limits                 webChatTranscriptionLimitsPayload         `json:"limits"`
}

type updateWebChatTranscriptionSettingsRequest struct {
	Enabled               bool    `json:"enabled"`
	Model                 string  `json:"model"`
	GroupIDs              []int64 `json:"group_ids"`
	UserDailyAudioSeconds *int    `json:"user_daily_audio_seconds"`
}

// GetWebChatTranscriptionSettings returns the database-managed STT control
// plane plus a non-secret model catalog derived from schedulable accounts.
// GET /api/v1/admin/settings/transcription
func (h *SettingHandler) GetWebChatTranscriptionSettings(c *gin.Context) {
	payload, err := h.webChatTranscriptionSettingsPayload(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, payload)
}

// UpdateWebChatTranscriptionSettings validates the selected groups/model and
// persists them in the shared settings table. No process restart is required.
// PUT /api/v1/admin/settings/transcription
func (h *SettingHandler) UpdateWebChatTranscriptionSettings(c *gin.Context) {
	var req updateWebChatTranscriptionSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if h == nil || h.settingService == nil {
		response.ErrorWithDetails(c, http.StatusServiceUnavailable, "Transcription settings are unavailable", "TRANSCRIPTION_SETTINGS_UNAVAILABLE", nil)
		return
	}

	// This endpoint replaces the complete mutable transcription settings object.
	// Requiring the allowance avoids a read-merge-write cycle on the shared JSON
	// blob, so a concurrent quota update can never be silently restored by a
	// request that omitted the field. Concurrent complete writes have ordinary,
	// explicit last-write-wins semantics.
	if req.UserDailyAudioSeconds == nil {
		response.ErrorWithDetails(
			c,
			http.StatusBadRequest,
			"user_daily_audio_seconds is required",
			"INVALID_TRANSCRIPTION_SETTINGS",
			map[string]string{"field": "user_daily_audio_seconds"},
		)
		return
	}

	validated, err := h.settingService.ValidateWebChatTranscriptionSettings(c.Request.Context(), &service.WebChatTranscriptionSettings{
		Enabled:               req.Enabled,
		Model:                 req.Model,
		GroupIDs:              req.GroupIDs,
		UserDailyAudioSeconds: *req.UserDailyAudioSeconds,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if validated.Enabled {
		if h.openAIGatewayService == nil {
			response.ErrorWithDetails(c, http.StatusServiceUnavailable, "Transcription scheduler is unavailable", "TRANSCRIPTION_SETTINGS_UNAVAILABLE", nil)
			return
		}
		available, availabilityErr := h.openAIGatewayService.HasAvailableWebChatTranscriptionRoute(
			c.Request.Context(),
			validated.GroupIDs,
			validated.Model,
		)
		if availabilityErr != nil {
			response.ErrorFrom(c, availabilityErr)
			return
		}
		if !available {
			response.ErrorWithDetails(
				c,
				http.StatusBadRequest,
				"No active audio-transcription API-key account in the selected groups supports this model",
				"TRANSCRIPTION_MODEL_UNAVAILABLE",
				nil,
			)
			return
		}
	}

	if err := h.settingService.SetWebChatTranscriptionSettings(c.Request.Context(), validated); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	payload, err := h.webChatTranscriptionSettingsPayload(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, payload)
}

func (h *SettingHandler) webChatTranscriptionSettingsPayload(ctx context.Context) (*webChatTranscriptionSettingsPayload, error) {
	settings, managed, err := h.settingService.GetWebChatTranscriptionSettings(ctx)
	if err != nil {
		return nil, err
	}
	effective, err := h.settingService.EffectiveTranscriptionConfig(ctx)
	if err != nil {
		return nil, err
	}
	runtimeReady := !settings.Enabled
	if settings.Enabled && h.openAIGatewayService != nil {
		runtimeCfg, runtimeErr := h.openAIGatewayService.EffectiveTranscriptionConfig(ctx)
		runtimeReady = runtimeErr == nil && runtimeCfg.Enabled
	}

	options := make([]service.WebChatTranscriptionModelOption, 0)
	catalogAvailable := h.openAIGatewayService != nil
	if catalogAvailable {
		options, err = h.openAIGatewayService.ListWebChatTranscriptionModelOptions(ctx, settings.GroupIDs, settings.Model)
		if err != nil {
			catalogAvailable = false
			options = make([]service.WebChatTranscriptionModelOption, 0)
		}
	}
	selectedAvailable := false
	for _, option := range options {
		if runtimeReady && strings.TrimSpace(option.ID) == settings.Model && option.AvailabilityChecked && option.AvailableGroupCount > 0 {
			selectedAvailable = true
			break
		}
	}

	return &webChatTranscriptionSettingsPayload{
		Enabled:  settings.Enabled,
		Provider: effective.Provider,
		Model:    settings.Model,
		// Keep the JSON contract stable for fresh/disabled installations. A nil
		// slice would be encoded as null and forces every client to special-case
		// an otherwise ordinary empty selection.
		GroupIDs:               append([]int64{}, settings.GroupIDs...),
		UserDailyAudioSeconds:  settings.UserDailyAudioSeconds,
		DailyAudioSecondsMin:   service.ManagedTranscriptionDailyAudioSecondsMin,
		DailyAudioSecondsMax:   service.ManagedTranscriptionDailyAudioSecondsMax,
		Managed:                managed,
		RuntimeReady:           runtimeReady,
		CatalogAvailable:       catalogAvailable,
		SelectedModelAvailable: selectedAvailable,
		ModelOptions:           options,
		Limits: webChatTranscriptionLimitsPayload{
			MaxUploadBytes:        effective.MaxUploadBytes,
			MaxDurationSeconds:    effective.MaxDurationSeconds,
			MaxConcurrentGlobal:   effective.MaxConcurrentGlobal,
			MaxConcurrentPerUser:  effective.MaxConcurrentPerUser,
			UserRequestsPerMinute: effective.UserRequestsPerMinute,
			UserDailyAudioSeconds: effective.UserDailyAudioSeconds,
			RequestTimeoutSeconds: effective.RequestTimeoutSeconds,
		},
	}, nil
}
