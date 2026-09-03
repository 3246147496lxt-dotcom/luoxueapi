package service

import (
	"bytes"
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestForwardTranscriptionRebuildsControlledMultipartAndUsesSelectedAccount(t *testing.T) {
	audioPath := filepath.Join(t.TempDir(), "private-user-filename.webm")
	audio := []byte{0x1a, 0x45, 0xdf, 0xa3, 0, 1, 2, 3}
	require.NoError(t, os.WriteFile(audioPath, audio, 0o600))

	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
			"X-Request-Id": []string{"transcription-request-id"},
		},
		Body: io.NopCloser(strings.NewReader(`{"text":"hello world","language":"en"}`)),
	}}
	svc := &OpenAIGatewayService{cfg: &config.Config{}, httpUpstream: upstream}
	account := &Account{
		ID:          42,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Concurrency: 3,
		Proxy:       &Proxy{Protocol: "http", Host: "127.0.0.1", Port: 8080},
		Credentials: map[string]any{
			"api_key":                 "sk-transcription-test",
			"base_url":                "https://provider.example/v1",
			"header_override_enabled": true,
			"model_mapping": map[string]any{
				"voice-alias": "provider-stt-model",
			},
			"header_overrides": map[string]any{"X-Provider-Tenant": "tenant-a"},
		},
	}

	result, err := svc.ForwardTranscription(context.Background(), account, TranscriptionForwardInput{
		FilePath:                 audioPath,
		FileExtension:            "webm",
		ContentType:              "audio/webm",
		Model:                    "voice-alias",
		UpstreamResponseMaxBytes: 1024,
	})
	require.NoError(t, err)
	require.Equal(t, "hello world", result.Text)
	require.Equal(t, "en", result.Language)
	require.Equal(t, "transcription-request-id", result.RequestID)
	require.Equal(t, "provider-stt-model", result.UpstreamModel)
	require.Len(t, upstream.requests, 1)
	require.Equal(t, "https://provider.example/v1/audio/transcriptions", upstream.lastReq.URL.String())
	require.Equal(t, "Bearer sk-transcription-test", upstream.lastReq.Header.Get("Authorization"))
	require.Equal(t, "tenant-a", getHeaderRaw(upstream.lastReq.Header, "x-provider-tenant"))
	require.Equal(t, "http://127.0.0.1:8080", upstream.lastProxyURL)

	mediaType, params, err := mime.ParseMediaType(upstream.lastReq.Header.Get("Content-Type"))
	require.NoError(t, err)
	require.Equal(t, "multipart/form-data", mediaType)
	form, err := multipart.NewReader(bytes.NewReader(upstream.lastBody), params["boundary"]).ReadForm(1 << 20)
	require.NoError(t, err)
	defer func() { _ = form.RemoveAll() }()
	require.Equal(t, []string{"provider-stt-model"}, form.Value["model"])
	require.Equal(t, []string{"json"}, form.Value["response_format"])
	require.Len(t, form.Value, 2)
	require.Len(t, form.File, 1)
	require.Len(t, form.File["file"], 1)
	require.Equal(t, "recording.webm", form.File["file"][0].Filename)
	file, err := form.File["file"][0].Open()
	require.NoError(t, err)
	forwardedAudio, err := io.ReadAll(file)
	require.NoError(t, err)
	require.NoError(t, file.Close())
	require.Equal(t, audio, forwardedAudio)
}

func TestForwardTranscriptionDoesNotRetryAmbiguousTransportFailure(t *testing.T) {
	audioPath := filepath.Join(t.TempDir(), "audio.webm")
	require.NoError(t, os.WriteFile(audioPath, []byte{0x1a, 0x45, 0xdf, 0xa3}, 0o600))
	upstream := &httpUpstreamRecorder{err: context.DeadlineExceeded}
	svc := &OpenAIGatewayService{cfg: &config.Config{}, httpUpstream: upstream}
	account := &Account{
		ID:       7,
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":  "sk-test",
			"base_url": "https://provider.example",
		},
	}

	_, err := svc.ForwardTranscription(context.Background(), account, TranscriptionForwardInput{
		FilePath: audioPath, FileExtension: "webm", ContentType: "audio/webm", Model: "stt",
	})
	require.Error(t, err)
	require.Len(t, upstream.requests, 1)
	var forwardErr *TranscriptionForwardError
	require.ErrorAs(t, err, &forwardErr)
	require.True(t, forwardErr.Timeout)
}

func TestTranscriptionUpstreamRequestIDSupportsSiliconFlowTraceHeader(t *testing.T) {
	header := http.Header{"X-Siliconcloud-Trace-Id": []string{"siliconflow-trace-id"}}
	require.Equal(t, "siliconflow-trace-id", transcriptionUpstreamRequestID(header))

	header.Set("Request-Id", "generic-request-id")
	require.Equal(t, "generic-request-id", transcriptionUpstreamRequestID(header))

	header.Set("X-Request-Id", "openai-request-id")
	require.Equal(t, "openai-request-id", transcriptionUpstreamRequestID(header))
}
