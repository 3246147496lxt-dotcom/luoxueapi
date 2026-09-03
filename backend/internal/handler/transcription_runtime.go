package handler

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"mime"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/transcriptiontemp"
	"github.com/gin-gonic/gin"
)

type transcriptionAudioProber interface {
	ProbeDuration(context.Context, string) (float64, error)
}

type ffprobeAudioProber struct {
	path               string
	timeout            time.Duration
	maxDurationSeconds int
}

var errTranscriptionProbeUnavailable = errors.New("audio probe unavailable")

const transcriptionProbeMaxAllocationBytes = "67108864"

// Encoders can leave one or two codec frames of priming/padding around the
// semantic recording duration. Container duration is only used to remove that
// small, bounded delta after decoded-sample counting proves it is not a forged
// timeline. Larger discrepancies always retain the decoded duration.
const (
	transcriptionProbeDecodeSlackSeconds          = 0.25
	transcriptionAACMinCorrectionSeconds          = 0.05
	transcriptionAACMaxCorrectionSeconds          = 0.25
	transcriptionMinimumSupportedSampleRate int64 = 8000
)

type ffprobeAudioStreamMetadata struct {
	Streams []struct {
		CodecName  string `json:"codec_name"`
		SampleRate string `json:"sample_rate"`
		Duration   string `json:"duration"`
	} `json:"streams"`
}

func (p ffprobeAudioProber) ProbeDuration(ctx context.Context, filePath string) (float64, error) {
	if _, err := exec.LookPath(p.path); err != nil {
		return 0, errTranscriptionProbeUnavailable
	}
	probeCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()
	sampleRateCmd := exec.CommandContext(
		probeCtx,
		p.path,
		"-v", "error",
		"-protocol_whitelist", "file,pipe",
		"-max_alloc", transcriptionProbeMaxAllocationBytes,
		"-threads", "1",
		"-select_streams", "a",
		"-show_entries", "stream=codec_name,sample_rate,duration",
		"-of", "json",
		filePath,
	)
	sampleRateOutput, err := sampleRateCmd.Output()
	if err != nil {
		if errors.Is(probeCtx.Err(), context.DeadlineExceeded) {
			return 0, context.DeadlineExceeded
		}
		return 0, errors.New("audio probe failed")
	}
	var metadata ffprobeAudioStreamMetadata
	if err := json.Unmarshal(sampleRateOutput, &metadata); err != nil || len(metadata.Streams) != 1 {
		return 0, errors.New("exactly one audio stream is required")
	}
	sampleRate, err := strconv.ParseInt(strings.TrimSpace(metadata.Streams[0].SampleRate), 10, 64)
	if err != nil || sampleRate < transcriptionMinimumSupportedSampleRate || sampleRate > 768000 {
		return 0, errors.New("invalid audio stream sample rate")
	}
	containerDuration, durationErr := strconv.ParseFloat(strings.TrimSpace(metadata.Streams[0].Duration), 64)
	if durationErr != nil || containerDuration <= 0 || math.IsNaN(containerDuration) || math.IsInf(containerDuration, 0) {
		containerDuration = 0
	}

	// Container timestamps are attacker-controlled and can under-report hours
	// of encoded speech as a few seconds. Counting decoded frame samples makes
	// max-duration and daily-quota enforcement independent of PTS metadata.
	framesCmd := exec.CommandContext(
		probeCtx,
		p.path,
		"-v", "error",
		"-protocol_whitelist", "file,pipe",
		"-max_alloc", transcriptionProbeMaxAllocationBytes,
		"-threads", "1",
		"-select_streams", "a:0",
		"-show_frames",
		"-show_entries", "frame=nb_samples",
		"-of", "csv=p=0",
		filePath,
	)
	framesCmd.Stderr = io.Discard
	stdout, err := framesCmd.StdoutPipe()
	if err != nil {
		return 0, errors.New("audio probe failed")
	}
	if err := framesCmd.Start(); err != nil {
		return 0, errors.New("audio probe failed")
	}
	maximumDuration := float64(p.maxDurationSeconds)
	if maximumDuration <= 0 {
		maximumDuration = 60
	}
	maximumSamples := int64(math.Ceil(float64(sampleRate) * (maximumDuration + transcriptionProbeDecodeSlackSeconds)))
	var totalSamples int64
	var largestFrameSamples int64
	frameCount := 0
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		value := strings.TrimSpace(scanner.Text())
		if value == "" {
			continue
		}
		samples, parseErr := strconv.ParseInt(value, 10, 64)
		if parseErr != nil || samples <= 0 || totalSamples > (1<<62)-samples {
			_ = framesCmd.Process.Kill()
			_ = framesCmd.Wait()
			return 0, errors.New("invalid decoded audio frame")
		}
		totalSamples += samples
		if samples > largestFrameSamples {
			largestFrameSamples = samples
		}
		frameCount++
		if totalSamples > maximumSamples {
			_ = framesCmd.Process.Kill()
			_ = framesCmd.Wait()
			return float64(totalSamples) / float64(sampleRate), nil
		}
	}
	scanErr := scanner.Err()
	waitErr := framesCmd.Wait()
	if probeCtx.Err() != nil {
		return 0, probeCtx.Err()
	}
	if scanErr != nil || waitErr != nil || frameCount == 0 || totalSamples <= 0 {
		return 0, errors.New("audio frame probe failed")
	}
	decodedDuration := float64(totalSamples) / float64(sampleRate)
	correctionLimit := boundedContainerDurationCorrection(
		metadata.Streams[0].CodecName,
		sampleRate,
		largestFrameSamples,
	)
	semanticContainerDuration := containerDuration
	if correctionLimit > 0 && hasTopLevelMP4Fragment(filePath) {
		// Fragmented AAC/MP4 (the shape produced by Safari MediaRecorder) has
		// no edit list and commonly includes one encoder-delay frame in the
		// stream duration. Remove exactly one observed frame, then apply the same
		// decoded-vs-container bound below.
		frameDuration := float64(largestFrameSamples) / float64(sampleRate)
		if semanticContainerDuration > frameDuration {
			semanticContainerDuration -= frameDuration
		}
	}
	if semanticContainerDuration > 0 && decodedDuration >= semanticContainerDuration &&
		decodedDuration-semanticContainerDuration <= correctionLimit {
		return semanticContainerDuration, nil
	}
	return decodedDuration, nil
}

func boundedContainerDurationCorrection(codecName string, sampleRate, largestFrameSamples int64) float64 {
	if !strings.EqualFold(strings.TrimSpace(codecName), "aac") || sampleRate <= 0 || largestFrameSamples <= 0 {
		return 0
	}
	// Native AAC encoders can expose up to roughly two decoded frames of
	// priming/padding. Derive the allowance from observed frame size, but cap it
	// so container timestamps can never hide an unbounded decoded duration.
	correction := 2 * float64(largestFrameSamples) / float64(sampleRate)
	if correction < transcriptionAACMinCorrectionSeconds {
		return transcriptionAACMinCorrectionSeconds
	}
	if correction > transcriptionAACMaxCorrectionSeconds {
		return transcriptionAACMaxCorrectionSeconds
	}
	return correction
}

func hasTopLevelMP4Fragment(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil || info.Size() < 8 {
		return false
	}
	fileSize := info.Size()
	var header [16]byte
	for offset, boxes := int64(0), 0; offset+8 <= fileSize && boxes < 1<<16; boxes++ {
		if _, err := file.ReadAt(header[:8], offset); err != nil {
			return false
		}
		size32 := binary.BigEndian.Uint32(header[:4])
		boxType := string(header[4:8])
		headerSize := int64(8)
		var boxSize int64
		switch size32 {
		case 0:
			boxSize = fileSize - offset
		case 1:
			if offset+16 > fileSize {
				return false
			}
			if _, err := file.ReadAt(header[8:16], offset+8); err != nil {
				return false
			}
			extended := binary.BigEndian.Uint64(header[8:16])
			if extended > uint64(math.MaxInt64) {
				return false
			}
			boxSize = int64(extended)
			headerSize = 16
		default:
			boxSize = int64(size32)
		}
		if boxSize < headerSize || boxSize > fileSize-offset {
			return false
		}
		if boxType == "moof" {
			return true
		}
		offset += boxSize
	}
	return false
}

type transcriptionRateWindow struct {
	startedAt time.Time
	count     int
}

type transcriptionUserLimiter struct {
	slots chan struct{}
	refs  int
}

type transcriptionRuntime struct {
	cfg    config.TranscriptionConfig
	prober transcriptionAudioProber
	now    func() time.Time

	global          chan struct{}
	mu              sync.Mutex
	users           map[int64]*transcriptionUserLimiter
	inflight        map[string]struct{}
	userRPM         map[int64]transcriptionRateWindow
	ipRPM           map[string]transcriptionRateWindow
	lastRateCleanup time.Time
}

func newTranscriptionRuntime(cfg config.TranscriptionConfig) *transcriptionRuntime {
	globalLimit := cfg.MaxConcurrentGlobal
	if globalLimit <= 0 {
		globalLimit = 1
	}
	probeTimeout := time.Duration(cfg.ProbeTimeoutSeconds) * time.Second
	if probeTimeout <= 0 {
		probeTimeout = 3 * time.Second
	}
	probePath := strings.TrimSpace(cfg.FFprobePath)
	if probePath == "" {
		probePath = "ffprobe"
	}
	return &transcriptionRuntime{
		cfg: cfg,
		prober: ffprobeAudioProber{
			path:               probePath,
			timeout:            probeTimeout,
			maxDurationSeconds: cfg.MaxDurationSeconds,
		},
		now:      time.Now,
		global:   make(chan struct{}, globalLimit),
		users:    make(map[int64]*transcriptionUserLimiter),
		inflight: make(map[string]struct{}),
		userRPM:  make(map[int64]transcriptionRateWindow),
		ipRPM:    make(map[string]transcriptionRateWindow),
	}
}

type transcriptionAdmissionError string

const (
	transcriptionAdmissionDuplicate transcriptionAdmissionError = "duplicate"
	transcriptionAdmissionRate      transcriptionAdmissionError = "rate"
	transcriptionAdmissionBusy      transcriptionAdmissionError = "busy"
	transcriptionAdmissionCanceled  transcriptionAdmissionError = "canceled"
)

func (r *transcriptionRuntime) Acquire(ctx context.Context, userID int64, clientIP, idempotencyKey string) (func(), transcriptionAdmissionError) {
	if r == nil {
		return nil, transcriptionAdmissionBusy
	}
	now := r.now()
	inflightKey := fmt.Sprintf("%d:%s", userID, idempotencyKey)
	r.mu.Lock()
	if _, exists := r.inflight[inflightKey]; exists {
		r.mu.Unlock()
		return nil, transcriptionAdmissionDuplicate
	}
	if !r.allowRateLocked(now, userID, clientIP) {
		r.mu.Unlock()
		return nil, transcriptionAdmissionRate
	}
	r.inflight[inflightKey] = struct{}{}
	userLimiter := r.userLimiterLocked(userID)
	userSlots := userLimiter.slots
	r.mu.Unlock()

	queueTimeout := time.Duration(r.cfg.QueueTimeoutMS) * time.Millisecond
	if queueTimeout <= 0 {
		queueTimeout = 2 * time.Second
	}
	queueCtx, cancel := context.WithTimeout(ctx, queueTimeout)
	defer cancel()
	claimedUser := false
	claimedGlobal := false
	cleanupClaim := func() {
		if claimedGlobal {
			<-r.global
		}
		if claimedUser {
			<-userSlots
		}
		r.mu.Lock()
		delete(r.inflight, inflightKey)
		userLimiter.refs--
		if userLimiter.refs == 0 && r.users[userID] == userLimiter {
			delete(r.users, userID)
		}
		r.mu.Unlock()
	}
	select {
	case userSlots <- struct{}{}:
		claimedUser = true
	case <-queueCtx.Done():
		cleanupClaim()
		if errors.Is(ctx.Err(), context.Canceled) {
			return nil, transcriptionAdmissionCanceled
		}
		return nil, transcriptionAdmissionBusy
	}
	select {
	case r.global <- struct{}{}:
		claimedGlobal = true
	case <-queueCtx.Done():
		cleanupClaim()
		if errors.Is(ctx.Err(), context.Canceled) {
			return nil, transcriptionAdmissionCanceled
		}
		return nil, transcriptionAdmissionBusy
	}
	var once sync.Once
	return func() { once.Do(cleanupClaim) }, ""
}

func (r *transcriptionRuntime) userLimiterLocked(userID int64) *transcriptionUserLimiter {
	if limiter := r.users[userID]; limiter != nil {
		limiter.refs++
		return limiter
	}
	limit := r.cfg.MaxConcurrentPerUser
	if limit <= 0 {
		limit = 1
	}
	limiter := &transcriptionUserLimiter{slots: make(chan struct{}, limit), refs: 1}
	r.users[userID] = limiter
	return limiter
}

func (r *transcriptionRuntime) allowRateLocked(now time.Time, userID int64, clientIP string) bool {
	if r.lastRateCleanup.IsZero() || now.Sub(r.lastRateCleanup) >= 5*time.Minute {
		for key, window := range r.userRPM {
			if now.Before(window.startedAt) || now.Sub(window.startedAt) >= 2*time.Minute {
				delete(r.userRPM, key)
			}
		}
		for key, window := range r.ipRPM {
			if now.Before(window.startedAt) || now.Sub(window.startedAt) >= 2*time.Minute {
				delete(r.ipRPM, key)
			}
		}
		r.lastRateCleanup = now
	}
	userWindow := r.userRPM[userID]
	if userWindow.startedAt.IsZero() || now.Before(userWindow.startedAt) || now.Sub(userWindow.startedAt) >= time.Minute {
		userWindow = transcriptionRateWindow{startedAt: now}
	}
	ipKey := strings.TrimSpace(clientIP)
	if ipKey == "" {
		ipKey = "unknown"
	}
	ipWindow := r.ipRPM[ipKey]
	if ipWindow.startedAt.IsZero() || now.Before(ipWindow.startedAt) || now.Sub(ipWindow.startedAt) >= time.Minute {
		ipWindow = transcriptionRateWindow{startedAt: now}
	}
	if userWindow.count >= r.cfg.UserRequestsPerMinute || ipWindow.count >= r.cfg.IPRequestsPerMinute {
		return false
	}
	userWindow.count++
	ipWindow.count++
	r.userRPM[userID] = userWindow
	r.ipRPM[ipKey] = ipWindow
	return true
}

type transcriptionRequestError struct {
	status  int
	reason  string
	message string
}

func (e *transcriptionRequestError) Error() string { return e.reason }

type transcriptionUpload struct {
	path        string
	contentType string
	extension   string
	size        int64
}

func (u *transcriptionUpload) Cleanup() {
	if u != nil && u.path != "" {
		_ = os.Remove(u.path)
	}
}

func parseTranscriptionUpload(c *gin.Context, cfg config.TranscriptionConfig) (*transcriptionUpload, error) {
	contentType, params, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || !strings.EqualFold(contentType, "multipart/form-data") || strings.TrimSpace(params["boundary"]) == "" {
		return nil, &transcriptionRequestError{http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "Content-Type must be multipart/form-data"}
	}
	reader, err := c.Request.MultipartReader()
	if err != nil {
		return nil, transcriptionMultipartReadError(err)
	}
	part, err := reader.NextPart()
	if errors.Is(err, io.EOF) {
		return nil, &transcriptionRequestError{http.StatusBadRequest, "AUDIO_FILE_REQUIRED", "Audio file is required"}
	}
	if err != nil {
		return nil, transcriptionMultipartReadError(err)
	}
	defer func() { _ = part.Close() }()
	if part.FormName() != "file" {
		return nil, &transcriptionRequestError{http.StatusBadRequest, "INVALID_TRANSCRIPTION_FIELDS", "Only one file field is allowed"}
	}
	if strings.TrimSpace(part.FileName()) == "" {
		return nil, &transcriptionRequestError{http.StatusBadRequest, "AUDIO_FILE_REQUIRED", "Exactly one audio file is required"}
	}
	declaredType, _, err := mime.ParseMediaType(part.Header.Get("Content-Type"))
	declaredType = strings.ToLower(strings.TrimSpace(declaredType))
	if err != nil || !acceptedTranscriptionMIME(cfg.AcceptedMIMETypes, declaredType) {
		return nil, &transcriptionRequestError{http.StatusUnsupportedMediaType, "UNSUPPORTED_AUDIO_TYPE", "Audio type is not supported"}
	}

	destination, err := transcriptiontemp.Create("audio-*")
	if err != nil {
		return nil, &transcriptionRequestError{http.StatusServiceUnavailable, "TRANSCRIPTION_UNAVAILABLE", "Voice transcription is temporarily unavailable"}
	}
	path := destination.Name()
	cleanup := func() {
		_ = destination.Close()
		_ = os.Remove(path)
	}
	written, copyErr := io.Copy(destination, io.LimitReader(part, cfg.MaxUploadBytes+1))
	closeErr := destination.Close()
	if copyErr != nil {
		cleanup()
		return nil, transcriptionMultipartReadError(copyErr)
	}
	if closeErr != nil {
		cleanup()
		return nil, &transcriptionRequestError{http.StatusBadRequest, "INVALID_AUDIO", "Audio file cannot be read"}
	}
	if written <= 0 {
		cleanup()
		return nil, &transcriptionRequestError{http.StatusBadRequest, "EMPTY_AUDIO", "Audio file is empty"}
	}
	if written > cfg.MaxUploadBytes {
		cleanup()
		return nil, &transcriptionRequestError{http.StatusRequestEntityTooLarge, "AUDIO_TOO_LARGE", "Audio upload is too large"}
	}
	if err := part.Close(); err != nil {
		cleanup()
		return nil, transcriptionMultipartReadError(err)
	}
	nextPart, nextErr := reader.NextPart()
	if nextPart != nil {
		_ = nextPart.Close()
	}
	if nextErr != nil && !errors.Is(nextErr, io.EOF) {
		cleanup()
		return nil, transcriptionMultipartReadError(nextErr)
	}
	if nextPart != nil {
		cleanup()
		return nil, &transcriptionRequestError{http.StatusBadRequest, "INVALID_TRANSCRIPTION_FIELDS", "Only one file field is allowed"}
	}
	container, canonicalType, extension, err := inspectTranscriptionContainer(path)
	if err != nil || !transcriptionMIMEMatchesContainer(declaredType, container) {
		cleanup()
		return nil, &transcriptionRequestError{http.StatusUnsupportedMediaType, "INVALID_AUDIO_CONTAINER", "Audio container does not match its declared type"}
	}
	return &transcriptionUpload{path: path, contentType: canonicalType, extension: extension, size: written}, nil
}

func transcriptionMultipartReadError(err error) *transcriptionRequestError {
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		return &transcriptionRequestError{http.StatusRequestEntityTooLarge, "AUDIO_TOO_LARGE", "Audio upload is too large"}
	}
	var networkErr net.Error
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &networkErr) && networkErr.Timeout()) {
		return &transcriptionRequestError{http.StatusRequestTimeout, "AUDIO_UPLOAD_TIMEOUT", "Audio upload timed out"}
	}
	return &transcriptionRequestError{http.StatusBadRequest, "INVALID_MULTIPART", "Invalid multipart request"}
}

func transcriptionUploadReadDeadline(ctx context.Context, now time.Time, configuredTimeout time.Duration) (time.Time, bool) {
	if ctx == nil || configuredTimeout <= 0 || ctx.Err() != nil {
		return time.Time{}, false
	}
	deadline := now.Add(configuredTimeout)
	if workflowDeadline, ok := ctx.Deadline(); ok && workflowDeadline.Before(deadline) {
		deadline = workflowDeadline
	}
	return deadline, deadline.After(now)
}

func setTranscriptionReadDeadline(c *gin.Context, deadline time.Time) (func(), error) {
	if c == nil || c.Writer == nil || deadline.IsZero() {
		return nil, errors.New("transcription upload deadline is unavailable")
	}
	controller := http.NewResponseController(c.Writer)
	if err := controller.SetReadDeadline(deadline); err != nil {
		return nil, err
	}
	var once sync.Once
	return func() {
		once.Do(func() {
			_ = controller.SetReadDeadline(time.Time{})
		})
	}, nil
}

func acceptedTranscriptionMIME(accepted []string, candidate string) bool {
	for _, value := range accepted {
		mediaType, _, err := mime.ParseMediaType(value)
		if err == nil && strings.EqualFold(strings.TrimSpace(mediaType), candidate) {
			return true
		}
	}
	return false
}

func inspectTranscriptionContainer(path string) (container, contentType, extension string, err error) {
	file, err := os.Open(path)
	if err != nil {
		return "", "", "", err
	}
	defer func() { _ = file.Close() }()
	header := make([]byte, 16)
	n, readErr := io.ReadFull(file, header)
	if readErr != nil && !errors.Is(readErr, io.ErrUnexpectedEOF) {
		return "", "", "", readErr
	}
	header = header[:n]
	switch {
	case len(header) >= 12 && bytes.Equal(header[:4], []byte("RIFF")) && bytes.Equal(header[8:12], []byte("WAVE")):
		return "wav", "audio/wav", "wav", nil
	case len(header) >= 4 && bytes.Equal(header[:4], []byte{0x1a, 0x45, 0xdf, 0xa3}):
		return "webm", "audio/webm", "webm", nil
	case len(header) >= 4 && bytes.Equal(header[:4], []byte("OggS")):
		return "ogg", "audio/ogg", "ogg", nil
	case len(header) >= 4 && bytes.Equal(header[:4], []byte("fLaC")):
		return "flac", "audio/flac", "flac", nil
	case len(header) >= 8 && bytes.Equal(header[4:8], []byte("ftyp")):
		return "mp4", "audio/mp4", "m4a", nil
	case len(header) >= 3 && bytes.Equal(header[:3], []byte("ID3")):
		return "mp3", "audio/mpeg", "mp3", nil
	case len(header) >= 2 && header[0] == 0xff && header[1]&0xe0 == 0xe0:
		return "mp3", "audio/mpeg", "mp3", nil
	default:
		return "", "", "", errors.New("unsupported audio container")
	}
}

func transcriptionMIMEMatchesContainer(mediaType, container string) bool {
	switch container {
	case "wav":
		return mediaType == "audio/wav" || mediaType == "audio/x-wav"
	case "webm":
		return mediaType == "audio/webm" || mediaType == "video/webm"
	case "ogg":
		return mediaType == "audio/ogg" || mediaType == "application/ogg"
	case "flac":
		return mediaType == "audio/flac"
	case "mp4":
		return mediaType == "audio/mp4" || mediaType == "video/mp4"
	case "mp3":
		return mediaType == "audio/mpeg"
	default:
		return false
	}
}
