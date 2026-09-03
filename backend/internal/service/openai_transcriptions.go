package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/transcriptiontemp"
)

type TranscriptionForwardInput struct {
	FilePath                 string
	FileExtension            string
	ContentType              string
	Model                    string
	ChannelMappedModel       string
	UpstreamResponseMaxBytes int64
}

type TranscriptionForwardResult struct {
	Text          string
	Language      string
	RequestID     string
	UpstreamModel string
	Duration      time.Duration
}

type TranscriptionForwardErrorKind string

const (
	TranscriptionForwardRequestFailed    TranscriptionForwardErrorKind = "request_failed"
	TranscriptionForwardUpstreamRejected TranscriptionForwardErrorKind = "upstream_rejected"
	TranscriptionForwardResponseTooLarge TranscriptionForwardErrorKind = "response_too_large"
	TranscriptionForwardInvalidResponse  TranscriptionForwardErrorKind = "invalid_response"
	TranscriptionForwardEmpty            TranscriptionForwardErrorKind = "empty_transcription"
)

type TranscriptionForwardError struct {
	Kind          TranscriptionForwardErrorKind
	StatusCode    int
	Timeout       bool
	UpstreamModel string
}

func (e *TranscriptionForwardError) Error() string {
	if e == nil {
		return "transcription forwarding failed"
	}
	return "transcription forwarding failed: " + string(e.Kind)
}

// ForwardTranscription sends one validated local audio file to one selected
// OpenAI-compatible API-key account. It never selects another account or
// retries after the request body may have been transmitted.
func (s *OpenAIGatewayService) ForwardTranscription(
	ctx context.Context,
	account *Account,
	input TranscriptionForwardInput,
) (*TranscriptionForwardResult, error) {
	startedAt := time.Now()
	if s == nil || s.httpUpstream == nil || account == nil || account.Type != AccountTypeAPIKey {
		return nil, &TranscriptionForwardError{Kind: TranscriptionForwardRequestFailed}
	}
	apiKey := account.GetOpenAIApiKey()
	if strings.TrimSpace(apiKey) == "" {
		return nil, &TranscriptionForwardError{Kind: TranscriptionForwardRequestFailed}
	}
	baseURL := strings.TrimSpace(account.GetOpenAIBaseURL())
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	validatedURL, err := s.validateUpstreamBaseURL(baseURL)
	if err != nil {
		return nil, &TranscriptionForwardError{Kind: TranscriptionForwardRequestFailed}
	}
	targetURL := buildOpenAIEndpointURL(validatedURL, "/v1/audio/transcriptions")

	upstreamModel := resolveOpenAIForwardModel(account, input.Model, input.ChannelMappedModel)
	upstreamModel = normalizeOpenAIModelForUpstream(account, upstreamModel)
	if strings.TrimSpace(upstreamModel) == "" {
		return nil, &TranscriptionForwardError{Kind: TranscriptionForwardRequestFailed}
	}

	body, contentType, cleanup, err := buildTranscriptionMultipart(input, upstreamModel)
	if err != nil {
		return nil, &TranscriptionForwardError{Kind: TranscriptionForwardRequestFailed}
	}
	defer cleanup()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, body)
	if err != nil {
		return nil, &TranscriptionForwardError{Kind: TranscriptionForwardRequestFailed}
	}
	req = req.WithContext(WithHTTPUpstreamProfile(req.Context(), HTTPUpstreamProfileOpenAI))
	req.ContentLength = bodySize(body)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	if customUA := account.GetOpenAIUserAgent(); customUA != "" {
		req.Header.Set("User-Agent", customUA)
	}
	account.ApplyHeaderOverrides(req.Header)

	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		return nil, &TranscriptionForwardError{
			Kind:    TranscriptionForwardRequestFailed,
			Timeout: errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded),
		}
	}
	defer func() { _ = resp.Body.Close() }()

	maxResponseBytes := input.UpstreamResponseMaxBytes
	if maxResponseBytes <= 0 {
		maxResponseBytes = 1 << 20
	}
	responseBody, tooLarge, readErr := readTranscriptionResponse(resp.Body, maxResponseBytes)
	if readErr != nil {
		return nil, &TranscriptionForwardError{Kind: TranscriptionForwardInvalidResponse}
	}
	if tooLarge {
		return nil, &TranscriptionForwardError{Kind: TranscriptionForwardResponseTooLarge, StatusCode: resp.StatusCode}
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		// STT provider bodies may contain transcript fragments or echoed request
		// metadata. Account health only receives status/headers so no raw provider
		// body can reach account error state or logs.
		s.handleOpenAIAccountUpstreamError(ctx, account, resp.StatusCode, resp.Header, nil, upstreamModel)
		return nil, &TranscriptionForwardError{
			Kind:          TranscriptionForwardUpstreamRejected,
			StatusCode:    resp.StatusCode,
			UpstreamModel: upstreamModel,
		}
	}

	var payload struct {
		Text     string `json:"text"`
		Language string `json:"language"`
	}
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return nil, &TranscriptionForwardError{Kind: TranscriptionForwardInvalidResponse, StatusCode: resp.StatusCode}
	}
	payload.Text = strings.TrimSpace(payload.Text)
	if payload.Text == "" {
		return nil, &TranscriptionForwardError{Kind: TranscriptionForwardEmpty, StatusCode: resp.StatusCode}
	}
	return &TranscriptionForwardResult{
		Text:          payload.Text,
		Language:      strings.TrimSpace(payload.Language),
		RequestID:     transcriptionUpstreamRequestID(resp.Header),
		UpstreamModel: upstreamModel,
		Duration:      time.Since(startedAt),
	}, nil
}

func transcriptionUpstreamRequestID(header http.Header) string {
	return firstNonEmptyString(
		header.Get("x-request-id"),
		header.Get("request-id"),
		header.Get("x-siliconcloud-trace-id"),
	)
}

// ResolveTranscriptionChannelMapping performs the channel lookup and the
// requested/channel-mapped restriction check without hiding repository or
// cache failures. Upstream-based restrictions are checked after account
// selection by IsTranscriptionAccountModelRestricted.
func (s *OpenAIGatewayService) ResolveTranscriptionChannelMapping(
	ctx context.Context,
	groupID int64,
	requestedModel string,
) (ChannelMappingResult, bool, error) {
	if s == nil || s.channelService == nil || groupID <= 0 {
		return ChannelMappingResult{}, false, errors.New("channel service is unavailable")
	}
	mapping, err := s.channelService.ResolveChannelMappingStrict(ctx, groupID, requestedModel)
	if err != nil {
		return ChannelMappingResult{}, false, err
	}
	billingModel := billingModelForRestriction(mapping.BillingModelSource, requestedModel, mapping.MappedModel)
	if strings.TrimSpace(billingModel) == "" {
		return mapping, false, nil
	}
	restricted, err := s.channelService.IsModelRestrictedStrict(ctx, groupID, billingModel)
	if err != nil {
		return ChannelMappingResult{}, false, err
	}
	return mapping, restricted, nil
}

func (s *OpenAIGatewayService) IsTranscriptionAccountModelRestricted(
	ctx context.Context,
	groupID int64,
	account *Account,
	requestedModel string,
	mapping ChannelMappingResult,
) (bool, error) {
	if mapping.BillingModelSource != BillingModelSourceUpstream {
		return false, nil
	}
	if s == nil || s.channelService == nil || groupID <= 0 || account == nil {
		return false, errors.New("channel service is unavailable")
	}
	upstreamModel := resolveOpenAIForwardModel(account, requestedModel, mapping.MappedModel)
	upstreamModel = normalizeOpenAIModelForUpstream(account, upstreamModel)
	if strings.TrimSpace(upstreamModel) == "" {
		return true, nil
	}
	return s.channelService.IsModelRestrictedStrict(ctx, groupID, upstreamModel)
}

func buildTranscriptionMultipart(input TranscriptionForwardInput, model string) (*os.File, string, func(), error) {
	source, err := os.Open(input.FilePath)
	if err != nil {
		return nil, "", func() {}, err
	}
	defer func() { _ = source.Close() }()

	body, err := transcriptiontemp.Create("upstream-*")
	if err != nil {
		return nil, "", func() {}, err
	}
	cleanup := func() {
		_ = body.Close()
		_ = os.Remove(body.Name())
	}
	w := multipart.NewWriter(body)
	extension := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(input.FileExtension)), ".")
	if extension == "" {
		extension = "audio"
	}
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="recording.%s"`, extension))
	header.Set("Content-Type", strings.TrimSpace(input.ContentType))
	part, err := w.CreatePart(header)
	if err == nil {
		_, err = io.Copy(part, source)
	}
	if err == nil {
		err = w.WriteField("model", model)
	}
	if err == nil {
		err = w.WriteField("response_format", "json")
	}
	contentType := w.FormDataContentType()
	if closeErr := w.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		_, err = body.Seek(0, io.SeekStart)
	}
	if err != nil {
		cleanup()
		return nil, "", func() {}, err
	}
	return body, contentType, cleanup, nil
}

func bodySize(file *os.File) int64 {
	if file == nil {
		return -1
	}
	info, err := file.Stat()
	if err != nil {
		return -1
	}
	return info.Size()
}

func readTranscriptionResponse(r io.Reader, maxBytes int64) ([]byte, bool, error) {
	limited := io.LimitReader(r, maxBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, false, err
	}
	if int64(len(body)) > maxBytes {
		return nil, true, nil
	}
	return body, false, nil
}
