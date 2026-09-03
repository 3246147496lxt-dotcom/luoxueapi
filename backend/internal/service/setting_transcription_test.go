package service

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type transcriptionSettingRepositoryStub struct {
	values map[string]string
}

func (s *transcriptionSettingRepositoryStub) Get(context.Context, string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *transcriptionSettingRepositoryStub) GetValue(_ context.Context, key string) (string, error) {
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", ErrSettingNotFound
}

func (s *transcriptionSettingRepositoryStub) Set(_ context.Context, key, value string) error {
	if s.values == nil {
		s.values = make(map[string]string)
	}
	s.values[key] = value
	return nil
}

func (s *transcriptionSettingRepositoryStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *transcriptionSettingRepositoryStub) SetMultiple(context.Context, map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *transcriptionSettingRepositoryStub) GetAll(context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *transcriptionSettingRepositoryStub) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

type transcriptionGroupReaderStub struct {
	groups map[int64]*Group
}

func (s *transcriptionGroupReaderStub) GetByID(_ context.Context, id int64) (*Group, error) {
	if group, ok := s.groups[id]; ok {
		return group, nil
	}
	return nil, ErrGroupNotFound
}

func transcriptionTestExecutable(t *testing.T) string {
	t.Helper()
	path, err := os.Executable()
	require.NoError(t, err)
	return path
}

func TestSettingServiceGetWebChatTranscriptionSettingsFallsBackToYAML(t *testing.T) {
	repo := &transcriptionSettingRepositoryStub{}
	cfg := &config.Config{
		RunMode: config.RunModeStandard,
		Transcription: config.TranscriptionConfig{
			Enabled:               true,
			Model:                 "yaml-transcribe-model",
			GroupIDs:              []int64{17, 9},
			UserDailyAudioSeconds: 2400,
		},
	}
	svc := NewSettingService(repo, cfg)

	settings, managed, err := svc.GetWebChatTranscriptionSettings(context.Background())

	require.NoError(t, err)
	require.False(t, managed)
	require.Equal(t, &WebChatTranscriptionSettings{
		Enabled:               true,
		Model:                 "yaml-transcribe-model",
		GroupIDs:              []int64{17, 9},
		UserDailyAudioSeconds: 2400,
	}, settings)

	settings.GroupIDs[0] = 99
	require.Equal(t, []int64{17, 9}, cfg.Transcription.GroupIDs, "fallback must not alias deployment config")
}

func TestSettingServiceGetWebChatTranscriptionSettingsBackfillsDailyAudioForLegacyJSON(t *testing.T) {
	repo := &transcriptionSettingRepositoryStub{values: map[string]string{
		SettingKeyWebChatTranscriptionSettings: `{"enabled":true,"model":"legacy-model","group_ids":[9]}`,
	}}
	cfg := &config.Config{
		RunMode: config.RunModeStandard,
		Transcription: config.TranscriptionConfig{
			UserDailyAudioSeconds: 90000,
		},
	}
	svc := NewSettingService(repo, cfg)

	settings, managed, err := svc.GetWebChatTranscriptionSettings(context.Background())

	require.NoError(t, err)
	require.True(t, managed)
	require.Equal(t, ManagedTranscriptionDailyAudioSecondsDefault, settings.UserDailyAudioSeconds)

	effective, err := svc.EffectiveTranscriptionConfig(context.Background())
	require.NoError(t, err)
	require.Equal(t, ManagedTranscriptionDailyAudioSecondsDefault, effective.UserDailyAudioSeconds)
}

func TestSettingServiceFreshYAMLDailyAudioSeedIsNormalizedToManagedContract(t *testing.T) {
	tests := []struct {
		name string
		seed int
		want int
	}{
		{name: "invalid seed uses fixed default", seed: 0, want: ManagedTranscriptionDailyAudioSecondsDefault},
		{name: "oversized legacy seed is capped", seed: 90000, want: ManagedTranscriptionDailyAudioSecondsMax},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewSettingService(&transcriptionSettingRepositoryStub{}, &config.Config{
				RunMode: config.RunModeStandard,
				Transcription: config.TranscriptionConfig{
					Model:                 "yaml-model",
					UserDailyAudioSeconds: tt.seed,
				},
			})

			settings, managed, err := svc.GetWebChatTranscriptionSettings(context.Background())
			require.NoError(t, err)
			require.False(t, managed)
			require.Equal(t, tt.want, settings.UserDailyAudioSeconds)

			effective, err := svc.EffectiveTranscriptionConfig(context.Background())
			require.NoError(t, err)
			require.Equal(t, tt.want, effective.UserDailyAudioSeconds)
		})
	}
}

func TestSettingServiceStoredOutOfContractDailyAudioIsCorruption(t *testing.T) {
	for _, raw := range []string{
		`{"enabled":false,"model":"stored-model","group_ids":[],"user_daily_audio_seconds":0}`,
		`{"enabled":false,"model":"stored-model","group_ids":[],"user_daily_audio_seconds":86401}`,
	} {
		repo := &transcriptionSettingRepositoryStub{values: map[string]string{
			SettingKeyWebChatTranscriptionSettings: raw,
		}}
		svc := NewSettingService(repo, &config.Config{RunMode: config.RunModeStandard})

		_, managed, err := svc.GetWebChatTranscriptionSettings(context.Background())

		require.Error(t, err)
		require.True(t, managed)
		require.Contains(t, err.Error(), "invalid stored web chat transcription settings")
	}
}

func TestSettingServiceEffectiveTranscriptionConfigOverlaysDatabaseAndPreservesHardLimits(t *testing.T) {
	stored, err := json.Marshal(WebChatTranscriptionSettings{
		Enabled:               true,
		Model:                 "db-transcribe-model",
		GroupIDs:              []int64{23, 11},
		UserDailyAudioSeconds: 4321,
	})
	require.NoError(t, err)

	repo := &transcriptionSettingRepositoryStub{values: map[string]string{
		SettingKeyWebChatTranscriptionSettings: string(stored),
	}}
	cfg := &config.Config{
		RunMode: config.RunModeStandard,
		Transcription: config.TranscriptionConfig{
			Enabled:                  false,
			Provider:                 "openai_compatible",
			Model:                    "yaml-transcribe-model",
			GroupIDs:                 []int64{3},
			MaxUploadBytes:           12 << 20,
			UploadTimeoutSeconds:     41,
			MaxDurationSeconds:       321,
			RequestTimeoutSeconds:    73,
			QueueTimeoutMS:           1900,
			IdempotencyTTLSeconds:    444,
			MaxConcurrentGlobal:      8,
			MaxConcurrentPerUser:     2,
			UserRequestsPerMinute:    7,
			IPRequestsPerMinute:      13,
			UserDailyAudioSeconds:    987,
			AcceptedMIMETypes:        []string{"audio/webm", "audio/wav"},
			UpstreamResponseMaxBytes: 64 << 10,
			ProbeTimeoutSeconds:      6,
			FFprobePath:              "/usr/local/bin/ffprobe",
		},
	}
	svc := NewSettingService(repo, cfg)

	effective, err := svc.EffectiveTranscriptionConfig(context.Background())

	require.NoError(t, err)
	require.True(t, effective.Enabled)
	require.Equal(t, "db-transcribe-model", effective.Model)
	require.Equal(t, []int64{11, 23}, effective.GroupIDs)
	require.Equal(t, cfg.Transcription.Provider, effective.Provider)
	require.Equal(t, cfg.Transcription.MaxUploadBytes, effective.MaxUploadBytes)
	require.Equal(t, cfg.Transcription.UploadTimeoutSeconds, effective.UploadTimeoutSeconds)
	require.Equal(t, cfg.Transcription.MaxDurationSeconds, effective.MaxDurationSeconds)
	require.Equal(t, cfg.Transcription.RequestTimeoutSeconds, effective.RequestTimeoutSeconds)
	require.Equal(t, cfg.Transcription.QueueTimeoutMS, effective.QueueTimeoutMS)
	require.Equal(t, cfg.Transcription.IdempotencyTTLSeconds, effective.IdempotencyTTLSeconds)
	require.Equal(t, cfg.Transcription.MaxConcurrentGlobal, effective.MaxConcurrentGlobal)
	require.Equal(t, cfg.Transcription.MaxConcurrentPerUser, effective.MaxConcurrentPerUser)
	require.Equal(t, cfg.Transcription.UserRequestsPerMinute, effective.UserRequestsPerMinute)
	require.Equal(t, cfg.Transcription.IPRequestsPerMinute, effective.IPRequestsPerMinute)
	require.Equal(t, 4321, effective.UserDailyAudioSeconds, "database-managed daily allowance must override the YAML seed")
	require.Equal(t, cfg.Transcription.AcceptedMIMETypes, effective.AcceptedMIMETypes)
	require.Equal(t, cfg.Transcription.UpstreamResponseMaxBytes, effective.UpstreamResponseMaxBytes)
	require.Equal(t, cfg.Transcription.ProbeTimeoutSeconds, effective.ProbeTimeoutSeconds)
	require.Equal(t, cfg.Transcription.FFprobePath, effective.FFprobePath)

	effective.AcceptedMIMETypes[0] = "audio/changed"
	require.Equal(t, "audio/webm", cfg.Transcription.AcceptedMIMETypes[0], "effective limits must not alias deployment config")
}

func TestSettingServiceValidateWebChatTranscriptionSettings(t *testing.T) {
	validGroups := &transcriptionGroupReaderStub{groups: map[int64]*Group{
		4: {
			ID:               4,
			Platform:         PlatformOpenAI,
			SubscriptionType: SubscriptionTypeStandard,
			Status:           StatusActive,
		},
		9: {
			ID:               9,
			Platform:         PlatformOpenAI,
			SubscriptionType: SubscriptionTypeStandard,
			Status:           StatusActive,
		},
		20: {
			ID:               20,
			Platform:         PlatformAnthropic,
			SubscriptionType: SubscriptionTypeStandard,
			Status:           StatusActive,
		},
	}}
	svc := NewSettingService(&transcriptionSettingRepositoryStub{}, &config.Config{
		RunMode: config.RunModeStandard,
		Gateway: config.GatewayConfig{MaxBodySize: 32 << 20},
		Server:  config.ServerConfig{MaxRequestBodySize: 32 << 20},
		Transcription: config.TranscriptionConfig{
			FFprobePath: transcriptionTestExecutable(t),
		},
	})
	svc.SetDefaultSubscriptionGroupReader(validGroups)

	normalized, err := svc.ValidateWebChatTranscriptionSettings(context.Background(), &WebChatTranscriptionSettings{
		Enabled:               true,
		Model:                 "  transcribe-v2  ",
		GroupIDs:              []int64{9, 4},
		UserDailyAudioSeconds: 1,
	})
	require.NoError(t, err)
	require.Equal(t, "transcribe-v2", normalized.Model)
	require.Equal(t, []int64{4, 9}, normalized.GroupIDs)
	require.Equal(t, 1, normalized.UserDailyAudioSeconds)

	maximum, err := svc.ValidateWebChatTranscriptionSettings(context.Background(), &WebChatTranscriptionSettings{
		Enabled:               true,
		Model:                 "transcribe-v2",
		GroupIDs:              []int64{4},
		UserDailyAudioSeconds: 86400,
	})
	require.NoError(t, err)
	require.Equal(t, 86400, maximum.UserDailyAudioSeconds)

	tests := []struct {
		name     string
		settings *WebChatTranscriptionSettings
		reason   string
	}{
		{
			name:     "nil settings",
			settings: nil,
			reason:   "INVALID_TRANSCRIPTION_SETTINGS",
		},
		{
			name:     "empty model",
			settings: &WebChatTranscriptionSettings{Model: " \t ", UserDailyAudioSeconds: 1200},
			reason:   "INVALID_TRANSCRIPTION_SETTINGS",
		},
		{
			name:     "model contains tab",
			settings: &WebChatTranscriptionSettings{Model: "foo\tbar", UserDailyAudioSeconds: 1200},
			reason:   "INVALID_TRANSCRIPTION_SETTINGS",
		},
		{
			name:     "model contains escape",
			settings: &WebChatTranscriptionSettings{Model: "foo\x1bbar", UserDailyAudioSeconds: 1200},
			reason:   "INVALID_TRANSCRIPTION_SETTINGS",
		},
		{
			name:     "enabled without group",
			settings: &WebChatTranscriptionSettings{Enabled: true, Model: "transcribe-v2", UserDailyAudioSeconds: 1200},
			reason:   "INVALID_TRANSCRIPTION_SETTINGS",
		},
		{
			name:     "duplicate group",
			settings: &WebChatTranscriptionSettings{Model: "transcribe-v2", GroupIDs: []int64{4, 4}, UserDailyAudioSeconds: 1200},
			reason:   "INVALID_TRANSCRIPTION_SETTINGS",
		},
		{
			name:     "non-positive group",
			settings: &WebChatTranscriptionSettings{Model: "transcribe-v2", GroupIDs: []int64{0}, UserDailyAudioSeconds: 1200},
			reason:   "INVALID_TRANSCRIPTION_SETTINGS",
		},
		{
			name:     "unknown group",
			settings: &WebChatTranscriptionSettings{Enabled: true, Model: "transcribe-v2", GroupIDs: []int64{404}, UserDailyAudioSeconds: 1200},
			reason:   "INVALID_TRANSCRIPTION_GROUP",
		},
		{
			name:     "non OpenAI group",
			settings: &WebChatTranscriptionSettings{Enabled: true, Model: "transcribe-v2", GroupIDs: []int64{20}, UserDailyAudioSeconds: 1200},
			reason:   "INVALID_TRANSCRIPTION_GROUP",
		},
		{
			name:     "zero daily audio allowance",
			settings: &WebChatTranscriptionSettings{Model: "transcribe-v2", UserDailyAudioSeconds: 0},
			reason:   "INVALID_TRANSCRIPTION_SETTINGS",
		},
		{
			name:     "negative daily audio allowance",
			settings: &WebChatTranscriptionSettings{Model: "transcribe-v2", UserDailyAudioSeconds: -1},
			reason:   "INVALID_TRANSCRIPTION_SETTINGS",
		},
		{
			name:     "daily audio allowance above maximum",
			settings: &WebChatTranscriptionSettings{Model: "transcribe-v2", UserDailyAudioSeconds: 86401},
			reason:   "INVALID_TRANSCRIPTION_SETTINGS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.ValidateWebChatTranscriptionSettings(context.Background(), tt.settings)
			require.Error(t, err)
			require.Equal(t, tt.reason, infraerrors.Reason(err))
		})
	}

	simple := NewSettingService(&transcriptionSettingRepositoryStub{}, &config.Config{RunMode: config.RunModeSimple})
	_, err = simple.ValidateWebChatTranscriptionSettings(context.Background(), &WebChatTranscriptionSettings{
		Enabled:               true,
		Model:                 "transcribe-v2",
		GroupIDs:              []int64{4},
		UserDailyAudioSeconds: 1200,
	})
	require.Error(t, err)
	require.Equal(t, "TRANSCRIPTION_UNSUPPORTED_RUN_MODE", infraerrors.Reason(err))

	disabled, err := svc.ValidateWebChatTranscriptionSettings(context.Background(), &WebChatTranscriptionSettings{
		Enabled:               false,
		Model:                 "transcribe-v2",
		GroupIDs:              []int64{404},
		UserDailyAudioSeconds: 1200,
	})
	require.NoError(t, err, "an unavailable group must not prevent an emergency disable")
	require.Equal(t, []int64{404}, disabled.GroupIDs)
}

func TestSettingServiceSetWebChatTranscriptionSettingsIsImmediatelyEffective(t *testing.T) {
	repo := &transcriptionSettingRepositoryStub{}
	cfg := &config.Config{
		RunMode: config.RunModeStandard,
		Gateway: config.GatewayConfig{MaxBodySize: 32 << 20},
		Server:  config.ServerConfig{MaxRequestBodySize: 32 << 20},
		Transcription: config.TranscriptionConfig{
			Enabled:               false,
			Model:                 "yaml-model",
			GroupIDs:              []int64{1},
			MaxUploadBytes:        5 << 20,
			QueueTimeoutMS:        2500,
			MaxDurationSeconds:    180,
			UserDailyAudioSeconds: 1200,
			FFprobePath:           transcriptionTestExecutable(t),
		},
	}
	svc := NewSettingService(repo, cfg)
	svc.SetDefaultSubscriptionGroupReader(&transcriptionGroupReaderStub{groups: map[int64]*Group{
		31: {
			ID:               31,
			Platform:         PlatformOpenAI,
			SubscriptionType: SubscriptionTypeStandard,
			Status:           StatusActive,
		},
	}})
	notifications := 0
	svc.SetOnUpdateCallback(func() { notifications++ })

	err := svc.SetWebChatTranscriptionSettings(context.Background(), &WebChatTranscriptionSettings{
		Enabled:               true,
		Model:                 "  live-model  ",
		GroupIDs:              []int64{31},
		UserDailyAudioSeconds: 7200,
	})

	require.NoError(t, err)
	require.Equal(t, 1, notifications)
	require.Contains(t, repo.values, SettingKeyWebChatTranscriptionSettings)

	settings, managed, err := svc.GetWebChatTranscriptionSettings(context.Background())
	require.NoError(t, err)
	require.True(t, managed)
	require.Equal(t, &WebChatTranscriptionSettings{
		Enabled:               true,
		Model:                 "live-model",
		GroupIDs:              []int64{31},
		UserDailyAudioSeconds: 7200,
	}, settings)

	effective, err := svc.EffectiveTranscriptionConfig(context.Background())
	require.NoError(t, err)
	require.True(t, effective.Enabled)
	require.Equal(t, "live-model", effective.Model)
	require.Equal(t, []int64{31}, effective.GroupIDs)
	require.Equal(t, cfg.Transcription.MaxUploadBytes, effective.MaxUploadBytes)
	require.Equal(t, cfg.Transcription.QueueTimeoutMS, effective.QueueTimeoutMS)
	require.Equal(t, 7200, effective.UserDailyAudioSeconds)
}

func TestSettingServiceEffectiveTranscriptionConfigHonorsRequestPin(t *testing.T) {
	repo := &transcriptionSettingRepositoryStub{}
	cfg := &config.Config{
		RunMode: config.RunModeStandard,
		Gateway: config.GatewayConfig{MaxBodySize: 32 << 20},
		Server:  config.ServerConfig{MaxRequestBodySize: 32 << 20},
		Transcription: config.TranscriptionConfig{
			Enabled:               true,
			Model:                 "initial-model",
			GroupIDs:              []int64{7},
			MaxUploadBytes:        7 << 20,
			UserDailyAudioSeconds: 1200,
			AcceptedMIMETypes:     []string{"audio/webm"},
			FFprobePath:           transcriptionTestExecutable(t),
		},
	}
	svc := NewSettingService(repo, cfg)
	svc.SetDefaultSubscriptionGroupReader(&transcriptionGroupReaderStub{groups: map[int64]*Group{
		8: {
			ID:               8,
			Platform:         PlatformOpenAI,
			SubscriptionType: SubscriptionTypeStandard,
			Status:           StatusActive,
		},
	}})

	initial, err := svc.EffectiveTranscriptionConfig(context.Background())
	require.NoError(t, err)
	pinnedCtx := WithEffectiveTranscriptionConfig(context.Background(), initial)
	initial.Model = "mutated-after-pin"
	initial.GroupIDs[0] = 999
	initial.AcceptedMIMETypes[0] = "audio/mutated"

	require.NoError(t, svc.SetWebChatTranscriptionSettings(context.Background(), &WebChatTranscriptionSettings{
		Enabled:               true,
		Model:                 "updated-model",
		GroupIDs:              []int64{8},
		UserDailyAudioSeconds: 7200,
	}))

	pinned, err := svc.EffectiveTranscriptionConfig(pinnedCtx)
	require.NoError(t, err)
	require.Equal(t, "initial-model", pinned.Model)
	require.Equal(t, []int64{7}, pinned.GroupIDs)
	require.Equal(t, 1200, pinned.UserDailyAudioSeconds)
	require.Equal(t, []string{"audio/webm"}, pinned.AcceptedMIMETypes)

	latest, err := svc.EffectiveTranscriptionConfig(context.Background())
	require.NoError(t, err)
	require.Equal(t, "updated-model", latest.Model)
	require.Equal(t, []int64{8}, latest.GroupIDs)
	require.Equal(t, 7200, latest.UserDailyAudioSeconds)
}

func TestOpenAIGatewayServiceEffectiveTranscriptionConfigFailsClosedOnUnreadyNode(t *testing.T) {
	stored, err := json.Marshal(WebChatTranscriptionSettings{
		Enabled:               true,
		Model:                 "shared-model",
		GroupIDs:              []int64{2},
		UserDailyAudioSeconds: 1200,
	})
	require.NoError(t, err)

	repo := &transcriptionSettingRepositoryStub{values: map[string]string{
		SettingKeyWebChatTranscriptionSettings: string(stored),
	}}
	unready := NewSettingService(repo, &config.Config{
		RunMode: config.RunModeStandard,
		Gateway: config.GatewayConfig{MaxBodySize: 1024},
		Server:  config.ServerConfig{MaxRequestBodySize: 1024},
		Transcription: config.TranscriptionConfig{
			MaxUploadBytes: 10 << 20,
			FFprobePath:    "definitely-missing-ffprobe-binary",
		},
	})
	gateway := &OpenAIGatewayService{settingService: unready}

	_, err = gateway.EffectiveTranscriptionConfig(context.Background())
	require.Error(t, err)
	require.Equal(t, "TRANSCRIPTION_BODY_LIMIT_TOO_SMALL", infraerrors.Reason(err))
}
