package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

type TranscriptionAdmissionDecision string

const (
	TranscriptionAdmissionAcquired  TranscriptionAdmissionDecision = "acquired"
	TranscriptionAdmissionDuplicate TranscriptionAdmissionDecision = "duplicate"
	TranscriptionAdmissionRate      TranscriptionAdmissionDecision = "rate_limited"
	TranscriptionAdmissionBusy      TranscriptionAdmissionDecision = "busy"
)

type TranscriptionAdmissionCacheInput struct {
	UserID                int64
	IPHash                string
	IdempotencyKeyHash    string
	LeaseID               string
	LeaseTTL              time.Duration
	IdempotencyTTL        time.Duration
	MaxConcurrentGlobal   int
	MaxConcurrentPerUser  int
	UserRequestsPerMinute int
	IPRequestsPerMinute   int
}

// TranscriptionAdmissionCache is intentionally narrower than GatewayCache so
// existing cache stubs and constructors remain source compatible. Production's
// Redis-backed gateway cache implements it; absence is a fail-closed condition.
type TranscriptionAdmissionCache interface {
	AcquireTranscriptionAdmission(context.Context, TranscriptionAdmissionCacheInput) (TranscriptionAdmissionDecision, error)
	ReleaseTranscriptionAdmission(context.Context, int64, string) error
	ReserveTranscriptionDailyAudio(context.Context, int64, int, int) (bool, error)
}

var ErrTranscriptionAdmissionUnavailable = errors.New("transcription distributed admission is unavailable")

const transcriptionWorkflowDeadlineSafetyMargin = 5 * time.Second

// TranscriptionAdmissionLeaseTTL is shared with the HTTP workflow deadline so
// an admitted request always stops before its Redis concurrency lease can
// expire. Keep one source of truth for both sides of that invariant.
func TranscriptionAdmissionLeaseTTL(cfg config.TranscriptionConfig) time.Duration {
	return time.Duration(cfg.UploadTimeoutSeconds+cfg.ProbeTimeoutSeconds+cfg.RequestTimeoutSeconds)*time.Second +
		time.Duration(cfg.QueueTimeoutMS)*time.Millisecond + time.Minute
}

func TranscriptionWorkflowTimeout(cfg config.TranscriptionConfig) time.Duration {
	timeout := TranscriptionAdmissionLeaseTTL(cfg) - transcriptionWorkflowDeadlineSafetyMargin
	if timeout <= 0 {
		return 0
	}
	return timeout
}

func (s *OpenAIGatewayService) AcquireTranscriptionAdmission(
	ctx context.Context,
	userID int64,
	clientIP,
	idempotencyKey string,
	cfg config.TranscriptionConfig,
) (func(), TranscriptionAdmissionDecision, error) {
	cache, ok := s.transcriptionAdmissionCache()
	if !ok {
		return nil, "", ErrTranscriptionAdmissionUnavailable
	}
	if s.cfg == nil || strings.TrimSpace(s.cfg.JWT.Secret) == "" {
		return nil, "", ErrTranscriptionAdmissionUnavailable
	}
	keyHash := hashTranscriptionAdmissionValue(idempotencyKey)
	leaseID, err := newTranscriptionAdmissionLeaseID()
	if err != nil {
		return nil, "", ErrTranscriptionAdmissionUnavailable
	}
	leaseTTL := TranscriptionAdmissionLeaseTTL(cfg)
	if leaseTTL <= 0 {
		return nil, "", ErrTranscriptionAdmissionUnavailable
	}
	idempotencyTTL := time.Duration(cfg.IdempotencyTTLSeconds) * time.Second
	if idempotencyTTL <= 0 {
		return nil, "", ErrTranscriptionAdmissionUnavailable
	}
	decision, err := cache.AcquireTranscriptionAdmission(ctx, TranscriptionAdmissionCacheInput{
		UserID:                userID,
		IPHash:                hmacTranscriptionAdmissionValue(s.cfg.JWT.Secret, strings.TrimSpace(clientIP)),
		IdempotencyKeyHash:    keyHash,
		LeaseID:               leaseID,
		LeaseTTL:              leaseTTL,
		IdempotencyTTL:        idempotencyTTL,
		MaxConcurrentGlobal:   cfg.MaxConcurrentGlobal,
		MaxConcurrentPerUser:  cfg.MaxConcurrentPerUser,
		UserRequestsPerMinute: cfg.UserRequestsPerMinute,
		IPRequestsPerMinute:   cfg.IPRequestsPerMinute,
	})
	if err != nil || decision != TranscriptionAdmissionAcquired {
		return nil, decision, err
	}
	var once sync.Once
	release := func() {
		once.Do(func() {
			releaseCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := cache.ReleaseTranscriptionAdmission(releaseCtx, userID, leaseID); err != nil {
				logger.L().Warn(
					"transcription.admission_release_failed",
					zap.Int64("user_id", userID),
					zap.Error(err),
				)
			}
		})
	}
	return release, decision, nil
}

func (s *OpenAIGatewayService) ReserveTranscriptionDailyAudio(
	ctx context.Context,
	userID int64,
	durationSeconds float64,
	limitSeconds int,
) (bool, error) {
	cache, ok := s.transcriptionAdmissionCache()
	if !ok {
		return false, ErrTranscriptionAdmissionUnavailable
	}
	seconds := int(math.Ceil(durationSeconds))
	if seconds <= 0 || limitSeconds <= 0 {
		return false, nil
	}
	return cache.ReserveTranscriptionDailyAudio(ctx, userID, seconds, limitSeconds)
}

func (s *OpenAIGatewayService) transcriptionAdmissionCache() (TranscriptionAdmissionCache, bool) {
	if s == nil || s.cache == nil {
		return nil, false
	}
	cache, ok := s.cache.(TranscriptionAdmissionCache)
	return cache, ok && cache != nil
}

func hashTranscriptionAdmissionValue(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func hmacTranscriptionAdmissionValue(secret, value string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}

func newTranscriptionAdmissionLeaseID() (string, error) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	return hex.EncodeToString(nonce), nil
}
