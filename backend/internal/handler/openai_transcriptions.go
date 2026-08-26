package handler

import (
	"context"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type chatTranscriptionResponse struct {
	Text       string                           `json:"text"`
	Language   string                           `json:"language,omitempty"`
	DurationMS int64                            `json:"duration_ms"`
	RequestID  string                           `json:"request_id,omitempty"`
	Billing    chatTranscriptionBillingResponse `json:"billing"`
}

type chatTranscriptionBillingResponse struct {
	Mode          string  `json:"mode"`
	ChargedAmount float64 `json:"charged_amount"`
}

const (
	transcriptionAdmissionContextKey      = "web_chat_transcription_admitted"
	transcriptionDailyQuotaExceededReason = "TRANSCRIPTION_DAILY_QUOTA_EXCEEDED"
)

// AdmitTranscription validates and rate-limits the JWT-authenticated ingress
// before ChatService performs user/group/principal lookups. The returned
// release must remain held through principal resolution and upstream work.
func (h *OpenAIGatewayHandler) AdmitTranscription(c *gin.Context) (func(), bool) {
	c.Header("Cache-Control", "private, no-store")
	if h == nil || h.cfg == nil || h.cfg.RunMode == config.RunModeSimple {
		transcriptionError(c, http.StatusNotFound, "TRANSCRIPTION_DISABLED", "Voice transcription is not enabled")
		return nil, false
	}
	if c.Request == nil {
		transcriptionError(c, http.StatusServiceUnavailable, "TRANSCRIPTION_UNAVAILABLE", "Voice transcription is temporarily unavailable")
		return nil, false
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		transcriptionError(c, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return nil, false
	}
	idempotencyKey, ok := transcriptionIdempotencyKey(c)
	if !ok {
		transcriptionError(c, http.StatusBadRequest, "INVALID_IDEMPOTENCY_KEY", "A canonical UUID Idempotency-Key is required")
		return nil, false
	}
	if h.gatewayService == nil {
		transcriptionError(c, http.StatusServiceUnavailable, "LIMITER_UNAVAILABLE", "Voice transcription admission control is temporarily unavailable")
		return nil, false
	}
	transcriptionCfg, err := h.gatewayService.EffectiveTranscriptionConfig(c.Request.Context())
	if err != nil {
		transcriptionError(c, http.StatusServiceUnavailable, "TRANSCRIPTION_UNAVAILABLE", "Voice transcription is temporarily unavailable")
		return nil, false
	}
	if !transcriptionCfg.Enabled {
		transcriptionError(c, http.StatusNotFound, "TRANSCRIPTION_DISABLED", "Voice transcription is not enabled")
		return nil, false
	}
	workflowTimeout := service.TranscriptionWorkflowTimeout(transcriptionCfg)
	if workflowTimeout <= 0 {
		transcriptionError(c, http.StatusServiceUnavailable, "TRANSCRIPTION_UNAVAILABLE", "Voice transcription is temporarily unavailable")
		return nil, false
	}
	pinnedCtx := service.WithEffectiveTranscriptionConfig(c.Request.Context(), transcriptionCfg)
	workflowCtx, cancelWorkflow := context.WithTimeout(pinnedCtx, workflowTimeout)
	c.Request = c.Request.WithContext(workflowCtx)
	workflowHandedOff := false
	defer func() {
		if !workflowHandedOff {
			cancelWorkflow()
		}
	}()
	clientIP := ip.GetTrustedClientIP(c)
	distributedRelease, distributedDecision, distributedErr := h.gatewayService.AcquireTranscriptionAdmission(
		c.Request.Context(),
		subject.UserID,
		clientIP,
		idempotencyKey,
		transcriptionCfg,
	)
	if distributedErr != nil {
		transcriptionError(c, http.StatusServiceUnavailable, "LIMITER_UNAVAILABLE", "Voice transcription admission control is temporarily unavailable")
		return nil, false
	}
	switch distributedDecision {
	case service.TranscriptionAdmissionAcquired:
		if distributedRelease == nil {
			transcriptionError(c, http.StatusServiceUnavailable, "LIMITER_UNAVAILABLE", "Voice transcription admission control is temporarily unavailable")
			return nil, false
		}
	case service.TranscriptionAdmissionDuplicate:
		transcriptionError(c, http.StatusConflict, "TRANSCRIPTION_IN_PROGRESS", "This transcription has already been submitted")
		return nil, false
	case service.TranscriptionAdmissionRate:
		c.Header("Retry-After", "60")
		transcriptionError(c, http.StatusTooManyRequests, "TRANSCRIPTION_RATE_LIMITED", "Voice transcription rate limit reached")
		return nil, false
	case service.TranscriptionAdmissionBusy:
		c.Header("Retry-After", transcriptionBusyRetryAfter(transcriptionCfg.QueueTimeoutMS))
		transcriptionError(c, http.StatusTooManyRequests, "TRANSCRIPTION_BUSY", "Voice transcription is busy; try again shortly")
		return nil, false
	default:
		transcriptionError(c, http.StatusServiceUnavailable, "LIMITER_UNAVAILABLE", "Voice transcription admission control is temporarily unavailable")
		return nil, false
	}

	runtime := h.transcriptionRuntime
	if runtime == nil {
		distributedRelease()
		transcriptionError(c, http.StatusServiceUnavailable, "TRANSCRIPTION_UNAVAILABLE", "Voice transcription is temporarily unavailable")
		return nil, false
	}
	localRelease, admissionErr := runtime.Acquire(c.Request.Context(), subject.UserID, clientIP, idempotencyKey)
	if admissionErr != "" {
		distributedRelease()
	}
	switch admissionErr {
	case "":
		if localRelease == nil {
			distributedRelease()
			transcriptionError(c, http.StatusServiceUnavailable, "TRANSCRIPTION_UNAVAILABLE", "Voice transcription is temporarily unavailable")
			return nil, false
		}
	case transcriptionAdmissionDuplicate:
		transcriptionError(c, http.StatusConflict, "TRANSCRIPTION_IN_PROGRESS", "This transcription is already in progress")
		return nil, false
	case transcriptionAdmissionRate:
		c.Header("Retry-After", "60")
		transcriptionError(c, http.StatusTooManyRequests, "TRANSCRIPTION_RATE_LIMITED", "Voice transcription rate limit reached")
		return nil, false
	case transcriptionAdmissionBusy:
		c.Header("Retry-After", transcriptionBusyRetryAfter(transcriptionCfg.QueueTimeoutMS))
		transcriptionError(c, http.StatusTooManyRequests, "TRANSCRIPTION_BUSY", "Voice transcription is busy; try again shortly")
		return nil, false
	case transcriptionAdmissionCanceled:
		return nil, false
	default:
		transcriptionError(c, http.StatusServiceUnavailable, "TRANSCRIPTION_UNAVAILABLE", "Voice transcription is temporarily unavailable")
		return nil, false
	}

	c.Set(transcriptionAdmissionContextKey, true)
	var once sync.Once
	workflowHandedOff = true
	return func() {
		once.Do(func() {
			cancelWorkflow()
			localRelease()
			distributedRelease()
		})
	}, true
}

func transcriptionIdempotencyKey(c *gin.Context) (string, bool) {
	if c == nil {
		return "", false
	}
	raw := c.GetHeader("Idempotency-Key")
	trimmed := strings.TrimSpace(raw)
	parsed, err := uuid.Parse(trimmed)
	return trimmed, err == nil && raw == trimmed && parsed.String() == trimmed
}

// Transcriptions handles the authenticated Web Chat cloud STT endpoint. It is
// deliberately separate from the public API-key gateway and never records a
// billing receipt in the platform-subsidized first release.
func (h *OpenAIGatewayHandler) Transcriptions(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	if h == nil || h.cfg == nil || h.cfg.RunMode == config.RunModeSimple {
		transcriptionError(c, http.StatusNotFound, "TRANSCRIPTION_DISABLED", "Voice transcription is not enabled")
		return
	}
	if c.Request == nil || c.Request.Context().Err() != nil {
		if c.Request != nil && errors.Is(c.Request.Context().Err(), context.DeadlineExceeded) {
			transcriptionError(c, http.StatusGatewayTimeout, "TRANSCRIPTION_TIMEOUT", "Voice transcription timed out")
		}
		return
	}
	admitted, _ := c.Get(transcriptionAdmissionContextKey)
	if admitted != true {
		transcriptionError(c, http.StatusServiceUnavailable, "LIMITER_UNAVAILABLE", "Voice transcription admission control is temporarily unavailable")
		return
	}
	if h.gatewayService == nil || h.billingCacheService == nil || h.concurrencyHelper == nil {
		transcriptionError(c, http.StatusServiceUnavailable, "TRANSCRIPTION_UNAVAILABLE", "Voice transcription is temporarily unavailable")
		return
	}
	transcriptionCfg, err := h.gatewayService.EffectiveTranscriptionConfig(c.Request.Context())
	if err != nil {
		transcriptionError(c, http.StatusServiceUnavailable, "TRANSCRIPTION_UNAVAILABLE", "Voice transcription is temporarily unavailable")
		return
	}
	if !transcriptionCfg.Enabled {
		transcriptionError(c, http.StatusNotFound, "TRANSCRIPTION_DISABLED", "Voice transcription is not enabled")
		return
	}
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok || apiKey == nil {
		transcriptionError(c, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 || apiKey.UserID != subject.UserID {
		transcriptionError(c, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	requestTimeout := time.Duration(transcriptionCfg.RequestTimeoutSeconds) * time.Second
	runtime := h.transcriptionRuntime
	if runtime == nil {
		transcriptionError(c, http.StatusServiceUnavailable, "TRANSCRIPTION_UNAVAILABLE", "Voice transcription is temporarily unavailable")
		return
	}

	readDeadline, ok := transcriptionUploadReadDeadline(
		c.Request.Context(),
		time.Now(),
		time.Duration(transcriptionCfg.UploadTimeoutSeconds)*time.Second,
	)
	if !ok {
		if errors.Is(c.Request.Context().Err(), context.DeadlineExceeded) {
			transcriptionError(c, http.StatusGatewayTimeout, "TRANSCRIPTION_TIMEOUT", "Voice transcription timed out")
		}
		return
	}
	clearReadDeadline, err := setTranscriptionReadDeadline(c, readDeadline)
	if err != nil {
		transcriptionError(c, http.StatusServiceUnavailable, "TRANSCRIPTION_UNAVAILABLE", "Voice transcription is temporarily unavailable")
		return
	}
	defer clearReadDeadline()
	upload, err := parseTranscriptionUpload(c, transcriptionCfg)
	clearReadDeadline()
	if err != nil {
		var requestErr *transcriptionRequestError
		if errors.As(err, &requestErr) {
			transcriptionError(c, requestErr.status, requestErr.reason, requestErr.message)
			return
		}
		transcriptionError(c, http.StatusBadRequest, "INVALID_AUDIO", "Invalid audio upload")
		return
	}
	allowConnectionReuseAfterTranscriptionBodyRead(c)
	defer upload.Cleanup()

	durationSeconds, err := runtime.prober.ProbeDuration(c.Request.Context(), upload.path)
	if err != nil {
		if errors.Is(err, errTranscriptionProbeUnavailable) {
			transcriptionError(c, http.StatusServiceUnavailable, "TRANSCRIPTION_UNAVAILABLE", "Voice transcription is temporarily unavailable")
			return
		}
		transcriptionError(c, http.StatusUnprocessableEntity, "INVALID_AUDIO", "Audio file could not be validated")
		return
	}
	if durationSeconds > float64(transcriptionCfg.MaxDurationSeconds) {
		transcriptionError(c, http.StatusUnprocessableEntity, "AUDIO_TOO_LONG", "Audio recording is too long")
		return
	}
	if _, err := h.billingCacheService.PeekWebChatEligibility(
		c.Request.Context(),
		subject.UserID,
		service.PlatformOpenAI,
	); err != nil {
		status, code, message, retryAfter := billingErrorDetails(err)
		if retryAfter > 0 {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
		}
		transcriptionError(c, status, code, message)
		return
	}

	model := strings.TrimSpace(transcriptionCfg.Model)
	setOpsRequestContext(c, model, false)
	setOpsEndpointContext(c, EndpointAudioTranscriptions, int16(service.RequestTypeSync))
	if apiKey.GroupID == nil || *apiKey.GroupID <= 0 {
		transcriptionError(c, http.StatusServiceUnavailable, "TRANSCRIPTION_UNAVAILABLE", "Voice transcription is temporarily unavailable")
		return
	}
	channelMapping, restricted, mappingErr := h.gatewayService.ResolveTranscriptionChannelMapping(
		c.Request.Context(),
		*apiKey.GroupID,
		model,
	)
	if mappingErr != nil || restricted {
		transcriptionError(c, http.StatusServiceUnavailable, "TRANSCRIPTION_UNAVAILABLE", "Voice transcription is temporarily unavailable")
		return
	}
	c.Request = c.Request.WithContext(service.WithOpenAIProfitControlSuppressed(c.Request.Context()))
	selection, _, err := h.gatewayService.SelectAccountWithSchedulerForCapability(
		c.Request.Context(),
		apiKey.GroupID,
		"",
		"",
		model,
		nil,
		service.OpenAIUpstreamTransportHTTPSSE,
		service.OpenAIEndpointCapabilityAudioTranscriptions,
		false,
		false,
		false,
	)
	if err != nil || selection == nil || selection.Account == nil {
		transcriptionError(c, http.StatusServiceUnavailable, "TRANSCRIPTION_UNAVAILABLE", "Voice transcription is temporarily unavailable")
		return
	}
	account := selection.Account
	if account.Type != service.AccountTypeAPIKey || !account.SupportsOpenAIEndpointCapability(service.OpenAIEndpointCapabilityAudioTranscriptions) {
		if selection.ReleaseFunc != nil {
			selection.ReleaseFunc()
		}
		transcriptionError(c, http.StatusServiceUnavailable, "TRANSCRIPTION_UNAVAILABLE", "Voice transcription is temporarily unavailable")
		return
	}
	accountRestricted, restrictionErr := h.gatewayService.IsTranscriptionAccountModelRestricted(
		c.Request.Context(),
		*apiKey.GroupID,
		account,
		model,
		channelMapping,
	)
	if restrictionErr != nil || accountRestricted {
		if selection.ReleaseFunc != nil {
			selection.ReleaseFunc()
		}
		transcriptionError(c, http.StatusServiceUnavailable, "TRANSCRIPTION_UNAVAILABLE", "Voice transcription is temporarily unavailable")
		return
	}
	setOpsSelectedAccount(c, account.ID, account.Platform)

	accountRelease, acquired := h.acquireTranscriptionAccountSlot(c, selection)
	if !acquired {
		c.Header("Retry-After", transcriptionBusyRetryAfter(transcriptionCfg.QueueTimeoutMS))
		transcriptionError(c, http.StatusTooManyRequests, "TRANSCRIPTION_BUSY", "Voice transcription is busy; try again shortly")
		return
	}
	if accountRelease != nil {
		defer accountRelease()
	}
	if err := c.Request.Context().Err(); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			transcriptionError(c, http.StatusGatewayTimeout, "TRANSCRIPTION_TIMEOUT", "Voice transcription timed out")
		}
		return
	}

	// Reserve daily audio only after eligibility, mapping, scheduling and the
	// account slot all succeed. Local/provider-busy failures therefore do not
	// irreversibly consume a user's daily allowance without an upstream attempt.
	distributedDailyAllowed, distributedDailyErr := h.gatewayService.ReserveTranscriptionDailyAudio(
		c.Request.Context(),
		subject.UserID,
		durationSeconds,
		transcriptionCfg.UserDailyAudioSeconds,
	)
	if distributedDailyErr != nil {
		transcriptionError(c, http.StatusServiceUnavailable, "LIMITER_UNAVAILABLE", "Voice transcription admission control is temporarily unavailable")
		return
	}
	if !distributedDailyAllowed {
		transcriptionDailyQuotaExceeded(c, runtime.now(), transcriptionCfg.UserDailyAudioSeconds)
		return
	}

	upstreamCtx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()
	result, err := h.gatewayService.ForwardTranscription(upstreamCtx, account, service.TranscriptionForwardInput{
		FilePath:                 upload.path,
		FileExtension:            upload.extension,
		ContentType:              upload.contentType,
		Model:                    model,
		ChannelMappedModel:       channelMapping.MappedModel,
		UpstreamResponseMaxBytes: transcriptionCfg.UpstreamResponseMaxBytes,
	})
	if err != nil {
		var forwardErr *service.TranscriptionForwardError
		if errors.As(err, &forwardErr) {
			if forwardErr.Kind == service.TranscriptionForwardUpstreamRejected {
				h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, forwardErr.UpstreamModel, false, nil)
			}
			switch {
			case forwardErr.Timeout:
				transcriptionError(c, http.StatusGatewayTimeout, "TRANSCRIPTION_TIMEOUT", "Voice transcription timed out")
			case forwardErr.Kind == service.TranscriptionForwardEmpty:
				transcriptionError(c, http.StatusUnprocessableEntity, "NO_SPEECH_DETECTED", "No speech was detected")
			default:
				transcriptionError(c, http.StatusBadGateway, "TRANSCRIPTION_UPSTREAM_FAILED", "Voice transcription provider failed")
			}
			return
		}
		transcriptionError(c, http.StatusBadGateway, "TRANSCRIPTION_UPSTREAM_FAILED", "Voice transcription provider failed")
		return
	}
	h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, result.UpstreamModel, true, nil)
	response.Success(c, chatTranscriptionResponse{
		Text:       result.Text,
		Language:   result.Language,
		DurationMS: int64(math.Round(durationSeconds * 1000)),
		RequestID:  result.RequestID,
		Billing: chatTranscriptionBillingResponse{
			Mode:          "subsidized",
			ChargedAmount: 0,
		},
	})
}

func (h *OpenAIGatewayHandler) acquireTranscriptionAccountSlot(c *gin.Context, selection *service.AccountSelectionResult) (func(), bool) {
	if selection == nil || selection.Account == nil {
		return nil, false
	}
	if selection.Acquired {
		return wrapGatewayRelease(c.Request.Context(), selection.ReleaseFunc), true
	}
	if selection.WaitPlan == nil || h.concurrencyHelper == nil {
		return nil, false
	}
	release, acquired, err := h.concurrencyHelper.TryAcquireAccountSlot(
		c.Request.Context(),
		selection.Account.ID,
		selection.WaitPlan.MaxConcurrency,
	)
	if err != nil || !acquired {
		return nil, false
	}
	return wrapGatewayRelease(c.Request.Context(), release), true
}

func transcriptionError(c *gin.Context, status int, reason, message string) {
	response.ErrorWithDetails(c, status, message, reason, nil)
}

func transcriptionDailyQuotaExceeded(c *gin.Context, now time.Time, limitSeconds int) {
	retryAfter, resetAt := transcriptionDailyQuotaReset(now)
	c.Header("Retry-After", retryAfter)
	response.ErrorWithDetails(
		c,
		http.StatusTooManyRequests,
		"Daily voice transcription limit reached",
		transcriptionDailyQuotaExceededReason,
		map[string]string{
			"limit_seconds":    strconv.Itoa(limitSeconds),
			"reset_in_seconds": retryAfter,
			"reset_at":         resetAt.Format(time.RFC3339),
		},
	)
}

func transcriptionDailyQuotaReset(now time.Time) (string, time.Time) {
	now = now.UTC()
	next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
	seconds := int(math.Ceil(next.Sub(now).Seconds()))
	if seconds <= 0 {
		seconds = 1
	}
	return strconv.Itoa(seconds), next
}

func transcriptionBusyRetryAfter(queueTimeoutMS int) string {
	seconds := (queueTimeoutMS + 999) / 1000
	if seconds <= 0 {
		seconds = 1
	}
	if seconds > 5 {
		seconds = 5
	}
	return strconv.Itoa(seconds)
}
