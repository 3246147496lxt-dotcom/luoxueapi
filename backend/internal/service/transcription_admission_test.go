package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type transcriptionAdmissionCacheStub struct {
	inputs   []TranscriptionAdmissionCacheInput
	released []string
}

func (s *transcriptionAdmissionCacheStub) GetSessionAccountID(context.Context, int64, string) (int64, error) {
	return 0, nil
}

func (s *transcriptionAdmissionCacheStub) SetSessionAccountID(context.Context, int64, string, int64, time.Duration) error {
	return nil
}

func (s *transcriptionAdmissionCacheStub) RefreshSessionTTL(context.Context, int64, string, time.Duration) error {
	return nil
}

func (s *transcriptionAdmissionCacheStub) DeleteSessionAccountID(context.Context, int64, string) error {
	return nil
}

func (s *transcriptionAdmissionCacheStub) AcquireTranscriptionAdmission(_ context.Context, input TranscriptionAdmissionCacheInput) (TranscriptionAdmissionDecision, error) {
	s.inputs = append(s.inputs, input)
	return TranscriptionAdmissionAcquired, nil
}

func (s *transcriptionAdmissionCacheStub) ReleaseTranscriptionAdmission(_ context.Context, _ int64, leaseID string) error {
	s.released = append(s.released, leaseID)
	return nil
}

func (s *transcriptionAdmissionCacheStub) ReserveTranscriptionDailyAudio(context.Context, int64, int, int) (bool, error) {
	return true, nil
}

func TestAcquireTranscriptionAdmissionUsesPrivateHashesIndependentTTLAndUniqueLeaseNonce(t *testing.T) {
	cache := &transcriptionAdmissionCacheStub{}
	secret := "transcription-admission-test-secret"
	svc := &OpenAIGatewayService{
		cache: cache,
		cfg:   &config.Config{JWT: config.JWTConfig{Secret: secret}},
	}
	cfg := config.TranscriptionConfig{
		UploadTimeoutSeconds:  30,
		ProbeTimeoutSeconds:   3,
		RequestTimeoutSeconds: 45,
		QueueTimeoutMS:        2000,
		IdempotencyTTLSeconds: 600,
		MaxConcurrentGlobal:   16,
		MaxConcurrentPerUser:  1,
		UserRequestsPerMinute: 6,
		IPRequestsPerMinute:   20,
	}

	releaseFirst, decision, err := svc.AcquireTranscriptionAdmission(context.Background(), 7, "192.0.2.7", "same-key", cfg)
	require.NoError(t, err)
	require.Equal(t, TranscriptionAdmissionAcquired, decision)
	releaseSecond, decision, err := svc.AcquireTranscriptionAdmission(context.Background(), 7, "192.0.2.7", "same-key", cfg)
	require.NoError(t, err)
	require.Equal(t, TranscriptionAdmissionAcquired, decision)

	require.Len(t, cache.inputs, 2)
	require.Equal(t, 140*time.Second, cache.inputs[0].LeaseTTL)
	require.Equal(t, 10*time.Minute, cache.inputs[0].IdempotencyTTL)
	require.Equal(t, hmacTranscriptionAdmissionValue(secret, "192.0.2.7"), cache.inputs[0].IPHash)
	require.NotEqual(t, hashTranscriptionAdmissionValue("192.0.2.7"), cache.inputs[0].IPHash)
	require.NotEqual(t, cache.inputs[0].LeaseID, cache.inputs[1].LeaseID)

	releaseFirst()
	releaseSecond()
	require.Equal(t, []string{cache.inputs[0].LeaseID, cache.inputs[1].LeaseID}, cache.released)
}

func TestTranscriptionWorkflowStopsBeforeDistributedLeaseExpires(t *testing.T) {
	cfg := config.TranscriptionConfig{
		UploadTimeoutSeconds:  30,
		ProbeTimeoutSeconds:   3,
		RequestTimeoutSeconds: 45,
		QueueTimeoutMS:        2000,
	}

	require.Equal(t, 140*time.Second, TranscriptionAdmissionLeaseTTL(cfg))
	require.Equal(t, 135*time.Second, TranscriptionWorkflowTimeout(cfg))
	require.Less(t, TranscriptionWorkflowTimeout(cfg), TranscriptionAdmissionLeaseTTL(cfg))
}

func TestAcquireTranscriptionAdmissionFailsClosedWithoutHMACSecret(t *testing.T) {
	cache := &transcriptionAdmissionCacheStub{}
	svc := &OpenAIGatewayService{cache: cache, cfg: &config.Config{}}

	release, decision, err := svc.AcquireTranscriptionAdmission(context.Background(), 7, "192.0.2.7", "key", config.TranscriptionConfig{})
	require.ErrorIs(t, err, ErrTranscriptionAdmissionUnavailable)
	require.Nil(t, release)
	require.Empty(t, decision)
	require.Empty(t, cache.inputs)
}
