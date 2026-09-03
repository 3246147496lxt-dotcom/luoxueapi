package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func testTranscriptionConfig() config.TranscriptionConfig {
	return config.TranscriptionConfig{
		MaxUploadBytes:        1024,
		MaxConcurrentGlobal:   1,
		MaxConcurrentPerUser:  1,
		QueueTimeoutMS:        10,
		UserRequestsPerMinute: 10,
		IPRequestsPerMinute:   10,
		UserDailyAudioSeconds: 120,
		AcceptedMIMETypes:     []string{"audio/webm", "audio/wav"},
	}
}

func transcriptionMultipartRequest(t *testing.T, contentType string, audio []byte, extraField bool) *gin.Context {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="file"; filename="recording.webm"`)
	header.Set("Content-Type", contentType)
	part, err := w.CreatePart(header)
	require.NoError(t, err)
	_, err = part.Write(audio)
	require.NoError(t, err)
	if extraField {
		require.NoError(t, w.WriteField("model", "client-override"))
	}
	require.NoError(t, w.Close())

	req := httptest.NewRequest("POST", "/api/v1/chat/transcriptions", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", w.FormDataContentType())
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = req
	return c
}

func TestParseTranscriptionUploadValidatesMagicAndCleansPrivateTempFile(t *testing.T) {
	cfg := testTranscriptionConfig()
	audio := append([]byte{0x1a, 0x45, 0xdf, 0xa3}, bytes.Repeat([]byte{0}, 20)...)
	c := transcriptionMultipartRequest(t, "audio/webm;codecs=opus", audio, false)

	upload, err := parseTranscriptionUpload(c, cfg)
	require.NoError(t, err)
	require.Equal(t, "audio/webm", upload.contentType)
	require.Equal(t, "webm", upload.extension)
	info, err := os.Stat(upload.path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	path := upload.path
	upload.Cleanup()
	_, err = os.Stat(path)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestParseTranscriptionUploadRejectsExtraFieldsAndMismatchedContainer(t *testing.T) {
	cfg := testTranscriptionConfig()
	webm := append([]byte{0x1a, 0x45, 0xdf, 0xa3}, bytes.Repeat([]byte{0}, 20)...)

	_, err := parseTranscriptionUpload(transcriptionMultipartRequest(t, "audio/webm", webm, true), cfg)
	var requestErr *transcriptionRequestError
	require.ErrorAs(t, err, &requestErr)
	require.Equal(t, "INVALID_TRANSCRIPTION_FIELDS", requestErr.reason)

	_, err = parseTranscriptionUpload(transcriptionMultipartRequest(t, "audio/wav", webm, false), cfg)
	require.ErrorAs(t, err, &requestErr)
	require.Equal(t, "INVALID_AUDIO_CONTAINER", requestErr.reason)
}

func TestParseTranscriptionUploadRejectsActualOversize(t *testing.T) {
	cfg := testTranscriptionConfig()
	cfg.MaxUploadBytes = 8
	webm := append([]byte{0x1a, 0x45, 0xdf, 0xa3}, bytes.Repeat([]byte{0}, 20)...)

	_, err := parseTranscriptionUpload(transcriptionMultipartRequest(t, "audio/webm", webm, false), cfg)
	var requestErr *transcriptionRequestError
	require.ErrorAs(t, err, &requestErr)
	require.Equal(t, "AUDIO_TOO_LARGE", requestErr.reason)
}

func TestFFprobeAudioProberCountsDecodedSamplesInsteadOfContainerTimeline(t *testing.T) {
	ffprobePath, err := exec.LookPath("ffprobe")
	if err != nil {
		t.Skip("ffprobe is unavailable")
	}
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg is unavailable for the regression fixture")
	}
	dir := t.TempDir()
	normalPath := filepath.Join(dir, "normal.webm")
	warpedPath := filepath.Join(dir, "fast-pts.webm")
	generate := exec.Command(
		ffmpegPath,
		"-v", "error",
		"-f", "lavfi",
		"-i", "anullsrc=r=48000:cl=mono",
		"-t", "2",
		"-c:a", "libopus",
		normalPath,
	)
	require.NoError(t, generate.Run())
	warp := exec.Command(
		ffmpegPath,
		"-v", "error",
		"-i", normalPath,
		"-af", "asetpts=PTS/100",
		"-c:a", "libopus",
		warpedPath,
	)
	require.NoError(t, warp.Run())

	metadata := exec.Command(
		ffprobePath,
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		warpedPath,
	)
	metadataOutput, err := metadata.Output()
	require.NoError(t, err)
	metadataDuration, err := strconv.ParseFloat(strings.TrimSpace(string(metadataOutput)), 64)
	require.NoError(t, err)
	require.Less(t, metadataDuration, 0.1, "fixture must reproduce the timestamp-compression bypass")

	prober := ffprobeAudioProber{
		path:               ffprobePath,
		timeout:            5 * time.Second,
		maxDurationSeconds: 10,
	}
	decodedDuration, err := prober.ProbeDuration(context.Background(), warpedPath)
	require.NoError(t, err)
	require.InDelta(t, 2, decodedDuration, 0.05)

	prober.maxDurationSeconds = 1
	limitedDuration, err := prober.ProbeDuration(context.Background(), warpedPath)
	require.NoError(t, err)
	require.Greater(t, limitedDuration, 1.0, "over-limit decoding must stop early but still reject")
}

func TestFFprobeAudioProberRemovesOnlyBoundedAACEncoderPadding(t *testing.T) {
	ffprobePath, err := exec.LookPath("ffprobe")
	if err != nil {
		t.Skip("ffprobe is unavailable")
	}
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg is unavailable for the regression fixture")
	}
	for _, tc := range []struct {
		name       string
		sampleRate int
		duration   int
		fragmented bool
	}{
		{name: "browser quality 48kHz", sampleRate: 48000, duration: 2},
		{name: "telephony quality 8kHz", sampleRate: 8000, duration: 3},
		{name: "Safari fragmented MP4", sampleRate: 48000, duration: 2, fragmented: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "exact-limit.mp4")
			args := []string{
				"-v", "error",
				"-f", "lavfi",
				"-i", fmt.Sprintf("anullsrc=r=%d:cl=mono", tc.sampleRate),
				"-t", strconv.Itoa(tc.duration),
				"-c:a", "aac",
			}
			if tc.fragmented {
				args = append(args, "-movflags", "frag_keyframe+empty_moov")
			}
			args = append(args, path)
			generate := exec.Command(ffmpegPath, args...)
			require.NoError(t, generate.Run())

			prober := ffprobeAudioProber{
				path:               ffprobePath,
				timeout:            5 * time.Second,
				maxDurationSeconds: tc.duration,
			}
			duration, err := prober.ProbeDuration(context.Background(), path)
			require.NoError(t, err)
			require.InDelta(t, tc.duration, duration, 0.001, "codec padding must not reject or overcharge an exact-limit recording")
		})
	}
}

func TestTranscriptionRuntimeEnforcesInflightConcurrencyAndRate(t *testing.T) {
	cfg := testTranscriptionConfig()
	runtime := newTranscriptionRuntime(cfg)
	fixedNow := time.Date(2026, 8, 7, 1, 2, 3, 0, time.UTC)
	runtime.now = func() time.Time { return fixedNow }

	release, admissionErr := runtime.Acquire(context.Background(), 7, "192.0.2.1", "first")
	require.Empty(t, admissionErr)
	_, admissionErr = runtime.Acquire(context.Background(), 7, "192.0.2.1", "first")
	require.Equal(t, transcriptionAdmissionDuplicate, admissionErr)
	_, admissionErr = runtime.Acquire(context.Background(), 7, "192.0.2.1", "second")
	require.Equal(t, transcriptionAdmissionBusy, admissionErr)
	release()

	release, admissionErr = runtime.Acquire(context.Background(), 7, "192.0.2.1", "third")
	require.Empty(t, admissionErr)
	release()
	require.Empty(t, runtime.users)
	runtime.cfg.UserRequestsPerMinute = 3
	_, admissionErr = runtime.Acquire(context.Background(), 7, "192.0.2.1", "fourth")
	require.Equal(t, transcriptionAdmissionRate, admissionErr)
}

func TestTranscriptionRuntimeReclaimsIdleUserAndExpiredRateState(t *testing.T) {
	cfg := testTranscriptionConfig()
	runtime := newTranscriptionRuntime(cfg)
	now := time.Date(2026, 8, 7, 1, 10, 0, 0, time.UTC)
	runtime.now = func() time.Time { return now }
	runtime.lastRateCleanup = now.Add(-6 * time.Minute)
	runtime.userRPM[99] = transcriptionRateWindow{startedAt: now.Add(-3 * time.Minute), count: 1}
	runtime.ipRPM["stale-ip"] = transcriptionRateWindow{startedAt: now.Add(-3 * time.Minute), count: 1}

	release, admissionErr := runtime.Acquire(context.Background(), 7, "192.0.2.7", "active")
	require.Empty(t, admissionErr)
	release()

	require.Empty(t, runtime.users)
	require.NotContains(t, runtime.userRPM, int64(99))
	require.NotContains(t, runtime.ipRPM, "stale-ip")
}

func TestTranscriptionUploadReadDeadlineNeverOutlivesWorkflow(t *testing.T) {
	now := time.Now()
	workflowDeadline := now.Add(5 * time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), workflowDeadline)
	defer cancel()

	deadline, ok := transcriptionUploadReadDeadline(ctx, now, 30*time.Second)
	require.True(t, ok)
	require.Equal(t, workflowDeadline, deadline)

	deadline, ok = transcriptionUploadReadDeadline(context.Background(), now, 30*time.Second)
	require.True(t, ok)
	require.Equal(t, now.Add(30*time.Second), deadline)

	canceled, cancelNow := context.WithCancel(context.Background())
	cancelNow()
	_, ok = transcriptionUploadReadDeadline(canceled, now, 30*time.Second)
	require.False(t, ok)
}

func TestTranscriptionDailyQuotaExceededHasDedicatedMachineReadableResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	now := time.Date(2026, 8, 8, 23, 59, 59, 500_000_000, time.UTC)

	transcriptionDailyQuotaExceeded(c, now, 7200)

	require.Equal(t, http.StatusTooManyRequests, recorder.Code)
	require.Equal(t, "1", recorder.Header().Get("Retry-After"), "Retry-After must round up and never invite an early retry")
	var got response.Response
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &got))
	require.Equal(t, transcriptionDailyQuotaExceededReason, got.Reason)
	require.Equal(t, "7200", got.Metadata["limit_seconds"])
	require.Equal(t, recorder.Header().Get("Retry-After"), got.Metadata["reset_in_seconds"])
	require.Equal(t, "2026-08-09T00:00:00Z", got.Metadata["reset_at"])
}
