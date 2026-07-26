package service

import (
	"context"
	"errors"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type chatAttemptRepositoryStub struct {
	claim          ChatAttemptClaim
	result         *ChatAttemptClaimResult
	err            error
	renewResult    bool
	renewedLease   chan time.Time
	recoveryResult int
	recoveryAt     time.Time
	recoveryLimit  int
}

func (s *chatAttemptRepositoryStub) Claim(
	_ context.Context,
	claim ChatAttemptClaim,
) (*ChatAttemptClaimResult, error) {
	s.claim = claim
	return s.result, s.err
}

func (s *chatAttemptRepositoryStub) MarkOutcome(
	context.Context,
	int64,
	string,
	string,
	int,
	string,
	string,
) error {
	return nil
}

func (s *chatAttemptRepositoryStub) RenewLease(
	_ context.Context,
	_ int64,
	_ string,
	leaseExpiresAt time.Time,
) (bool, error) {
	if s.renewedLease != nil {
		select {
		case s.renewedLease <- leaseExpiresAt:
		default:
		}
	}
	return s.renewResult, s.err
}

func (s *chatAttemptRepositoryStub) RecoverExpiredLeases(
	_ context.Context,
	expiredAt time.Time,
	limit int,
) (int, error) {
	s.recoveryAt = expiredAt
	s.recoveryLimit = limit
	return s.recoveryResult, s.err
}

func TestChatAttemptServiceClaimValidatesAttemptID(t *testing.T) {
	t.Parallel()

	svc := NewChatAttemptService(&chatAttemptRepositoryStub{})
	for _, attemptID := range []string{"", "short", "contains spaces", "bad/slash"} {
		_, err := svc.Claim(context.Background(), 42, attemptID, "server-id", []byte(`{}`))
		require.Error(t, err)
	}
}

func TestChatAttemptServiceClaimPersistsServerReceiptCorrelation(t *testing.T) {
	t.Parallel()

	repo := &chatAttemptRepositoryStub{
		result: &ChatAttemptClaimResult{
			Claimed:         true,
			ClientRequestID: "server-id",
		},
	}
	svc := NewChatAttemptService(repo)

	result, err := svc.Claim(
		context.Background(),
		42,
		"attempt-12345678",
		"server-id",
		[]byte(`{"model":"gpt-5.5"}`),
	)

	require.NoError(t, err)
	require.True(t, result.Claimed)
	require.Equal(t, int64(42), repo.claim.UserID)
	require.Equal(t, "attempt-12345678", repo.claim.AttemptID)
	require.Equal(t, "server-id", repo.claim.ClientRequestID)
	require.Len(t, repo.claim.RequestHash, 64)
}

func TestChatAttemptServiceClaimReturnsOriginalReceiptForReplay(t *testing.T) {
	t.Parallel()

	repo := &chatAttemptRepositoryStub{
		result: &ChatAttemptClaimResult{
			Claimed:         false,
			ClientRequestID: "original-server-id",
		},
	}
	svc := NewChatAttemptService(repo)

	result, err := svc.Claim(
		context.Background(),
		42,
		"attempt-12345678",
		"new-server-id",
		[]byte(`{"model":"gpt-5.5"}`),
	)

	require.Equal(t, "original-server-id", result.ClientRequestID)
	require.True(t, errors.Is(err, ErrChatAttemptAlreadySubmitted))
	require.Equal(t, "original-server-id", infraerrors.FromError(err).Metadata["receipt_id"])
}

func TestChatAttemptLeaseHeartbeatUsesBackgroundLifetime(t *testing.T) {
	t.Parallel()

	fixedNow := time.Date(2026, 7, 25, 14, 0, 0, 0, time.UTC)
	repo := &chatAttemptRepositoryStub{
		renewResult:  true,
		renewedLease: make(chan time.Time, 1),
	}
	svc := NewChatAttemptService(repo)
	svc.now = func() time.Time { return fixedNow }
	svc.heartbeatInterval = time.Millisecond

	stop := svc.StartLeaseHeartbeat(42, "attempt-12345678")
	defer stop()

	select {
	case leaseExpiresAt := <-repo.renewedLease:
		require.Equal(t, fixedNow.Add(ChatAttemptLeaseDuration), leaseExpiresAt)
	case <-time.After(time.Second):
		t.Fatal("chat attempt lease was not renewed")
	}
}

func TestChatAttemptRecoveryUsesCurrentTimeAndFixedBatch(t *testing.T) {
	t.Parallel()

	fixedNow := time.Date(2026, 7, 25, 14, 30, 0, 0, time.UTC)
	repo := &chatAttemptRepositoryStub{recoveryResult: 3}
	svc := NewChatAttemptService(repo)
	svc.now = func() time.Time { return fixedNow }

	recovered, err := svc.ReconcileExpiredAttempts(context.Background())

	require.NoError(t, err)
	require.Equal(t, 3, recovered)
	require.Equal(t, fixedNow, repo.recoveryAt)
	require.Equal(t, chatAttemptRecoveryBatchSize, repo.recoveryLimit)
}
