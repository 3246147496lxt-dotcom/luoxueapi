package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"strings"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var (
	ErrChatAttemptIDRequired = infraerrors.BadRequest(
		"CHAT_ATTEMPT_ID_REQUIRED",
		"A chat attempt id is required",
	)
	ErrChatAttemptIDInvalid = infraerrors.BadRequest(
		"CHAT_ATTEMPT_ID_INVALID",
		"The chat attempt id is invalid",
	)
	ErrChatAttemptConflict = infraerrors.Conflict(
		"CHAT_ATTEMPT_CONFLICT",
		"The chat attempt id was reused with different content",
	)
	ErrChatAttemptAlreadySubmitted = infraerrors.Conflict(
		"CHAT_ATTEMPT_ALREADY_SUBMITTED",
		"This chat attempt was already submitted",
	)
)

const (
	ChatAttemptStatusAccepted    = "accepted"
	ChatAttemptStatusProcessing  = "processing"
	ChatAttemptStatusCompleted   = "completed"
	ChatAttemptStatusInterrupted = "interrupted"
	ChatAttemptStatusFailed      = "failed"
)

const (
	ChatAttemptLeaseDuration         = 5 * time.Minute
	chatAttemptHeartbeatInterval     = 15 * time.Second
	chatAttemptRecoveryInterval      = 30 * time.Second
	chatAttemptRecoveryBatchSize     = 100
	chatAttemptLeaseOperationTimeout = 5 * time.Second
)

type ChatAttemptClaim struct {
	UserID          int64
	AttemptID       string
	ClientRequestID string
	RequestHash     string
}

type ChatAttemptClaimResult struct {
	Claimed         bool
	ClientRequestID string
}

type ChatAttemptRepository interface {
	Claim(ctx context.Context, claim ChatAttemptClaim) (*ChatAttemptClaimResult, error)
	MarkOutcome(
		ctx context.Context,
		userID int64,
		attemptID string,
		status string,
		httpStatus int,
		failureCode string,
		failureReason string,
	) error
	RenewLease(
		ctx context.Context,
		userID int64,
		attemptID string,
		leaseExpiresAt time.Time,
	) (bool, error)
	RecoverExpiredLeases(
		ctx context.Context,
		expiredAt time.Time,
		limit int,
	) (int, error)
}

type ChatAttemptService struct {
	repo              ChatAttemptRepository
	now               func() time.Time
	leaseDuration     time.Duration
	heartbeatInterval time.Duration
	recoveryInterval  time.Duration

	recoveryMu     sync.Mutex
	recoveryCancel context.CancelFunc
	recoveryDone   chan struct{}
}

func NewChatAttemptService(repo ChatAttemptRepository) *ChatAttemptService {
	return &ChatAttemptService{
		repo:              repo,
		now:               time.Now,
		leaseDuration:     ChatAttemptLeaseDuration,
		heartbeatInterval: chatAttemptHeartbeatInterval,
		recoveryInterval:  chatAttemptRecoveryInterval,
	}
}

func (s *ChatAttemptService) Claim(
	ctx context.Context,
	userID int64,
	attemptID string,
	clientRequestID string,
	requestPayload []byte,
) (*ChatAttemptClaimResult, error) {
	attemptID = strings.TrimSpace(attemptID)
	clientRequestID = strings.TrimSpace(clientRequestID)
	if attemptID == "" {
		return nil, ErrChatAttemptIDRequired
	}
	if !validChatAttemptID(attemptID) {
		return nil, ErrChatAttemptIDInvalid
	}
	if s == nil || s.repo == nil || userID <= 0 || clientRequestID == "" {
		return nil, ErrChatCatalogUnavailable
	}

	sum := sha256.Sum256(requestPayload)
	result, err := s.repo.Claim(ctx, ChatAttemptClaim{
		UserID:          userID,
		AttemptID:       attemptID,
		ClientRequestID: clientRequestID,
		RequestHash:     hex.EncodeToString(sum[:]),
	})
	if err != nil {
		return nil, err
	}
	if result == nil || strings.TrimSpace(result.ClientRequestID) == "" {
		return nil, ErrChatCatalogUnavailable
	}
	if result.Claimed {
		return result, nil
	}
	return result, ErrChatAttemptAlreadySubmitted.WithMetadata(map[string]string{
		"receipt_id": result.ClientRequestID,
	})
}

func validChatAttemptID(value string) bool {
	if len(value) < 8 || len(value) > 64 {
		return false
	}
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-', r == '_':
		default:
			return false
		}
	}
	return true
}

func (s *ChatAttemptService) MarkOutcome(
	ctx context.Context,
	userID int64,
	attemptID string,
	status string,
	httpStatus int,
	failureCode string,
	failureReason string,
) error {
	if s == nil || s.repo == nil || userID <= 0 {
		return ErrChatCatalogUnavailable
	}
	attemptID = strings.TrimSpace(attemptID)
	if !validChatAttemptID(attemptID) {
		return ErrChatAttemptIDInvalid
	}
	switch status {
	case ChatAttemptStatusProcessing,
		ChatAttemptStatusCompleted,
		ChatAttemptStatusInterrupted,
		ChatAttemptStatusFailed:
	default:
		return ErrChatAttemptConflict
	}
	return s.repo.MarkOutcome(
		ctx,
		userID,
		attemptID,
		status,
		httpStatus,
		strings.TrimSpace(failureCode),
		strings.TrimSpace(failureReason),
	)
}

// StartLeaseHeartbeat keeps a claimed Web Chat attempt live independently of
// browser cancellation. The returned stop function waits for an in-flight
// renewal so callers can keep it running through detached drain and Finalize.
func (s *ChatAttemptService) StartLeaseHeartbeat(userID int64, attemptID string) func() {
	attemptID = strings.TrimSpace(attemptID)
	if s == nil || s.repo == nil || userID <= 0 || !validChatAttemptID(attemptID) {
		return func() {}
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		interval := s.heartbeatInterval
		if interval <= 0 {
			interval = chatAttemptHeartbeatInterval
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				renewed, err := s.renewLease(ctx, userID, attemptID)
				if err != nil {
					slog.Warn("renew chat attempt lease failed",
						"user_id", userID,
						"attempt_id", attemptID,
						"error", err,
					)
					continue
				}
				if !renewed {
					return
				}
			}
		}
	}()

	var stopOnce sync.Once
	return func() {
		stopOnce.Do(cancel)
		<-done
	}
}

func (s *ChatAttemptService) renewLease(
	ctx context.Context,
	userID int64,
	attemptID string,
) (bool, error) {
	if s == nil || s.repo == nil {
		return false, ErrChatCatalogUnavailable
	}
	now := time.Now
	if s.now != nil {
		now = s.now
	}
	duration := s.leaseDuration
	if duration <= 0 {
		duration = ChatAttemptLeaseDuration
	}
	renewCtx, cancel := context.WithTimeout(ctx, chatAttemptLeaseOperationTimeout)
	defer cancel()
	return s.repo.RenewLease(
		renewCtx,
		userID,
		attemptID,
		now().UTC().Add(duration),
	)
}

func (s *ChatAttemptService) ReconcileExpiredAttempts(ctx context.Context) (int, error) {
	if s == nil || s.repo == nil {
		return 0, ErrChatCatalogUnavailable
	}
	now := time.Now
	if s.now != nil {
		now = s.now
	}
	return s.repo.RecoverExpiredLeases(
		ctx,
		now().UTC(),
		chatAttemptRecoveryBatchSize,
	)
}

func (s *ChatAttemptService) StartRecovery(ctx context.Context) error {
	if s == nil || s.repo == nil {
		return ErrChatCatalogUnavailable
	}
	if ctx == nil {
		ctx = context.Background()
	}

	s.recoveryMu.Lock()
	defer s.recoveryMu.Unlock()
	if s.recoveryCancel != nil {
		return nil
	}
	recoveryCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	s.recoveryCancel = cancel
	s.recoveryDone = done
	go s.runRecovery(recoveryCtx, done)
	return nil
}

func (s *ChatAttemptService) StopRecovery(ctx context.Context) error {
	if s == nil {
		return nil
	}
	s.recoveryMu.Lock()
	cancel := s.recoveryCancel
	done := s.recoveryDone
	s.recoveryMu.Unlock()
	if cancel == nil || done == nil {
		return nil
	}
	cancel()
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *ChatAttemptService) runRecovery(ctx context.Context, done chan<- struct{}) {
	defer close(done)
	reconcile := func() {
		reconcileCtx, cancel := context.WithTimeout(ctx, chatAttemptRecoveryInterval)
		defer cancel()
		recovered, err := s.ReconcileExpiredAttempts(reconcileCtx)
		if err != nil {
			if ctx.Err() == nil {
				slog.Warn("recover expired chat attempts failed", "error", err)
			}
			return
		}
		if recovered > 0 {
			slog.Info("recovered expired chat attempts", "count", recovered)
		}
	}

	reconcile()
	interval := s.recoveryInterval
	if interval <= 0 {
		interval = chatAttemptRecoveryInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			reconcile()
		}
	}
}
