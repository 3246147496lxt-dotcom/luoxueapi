package service

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
)

type accountHealthSettingsRepoStub struct{ value string }

func (r *accountHealthSettingsRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if key != SettingKeyAccountHealthSettings {
		return "", nil
	}
	return r.value, nil
}
func (r *accountHealthSettingsRepoStub) Set(_ context.Context, key, value string) error {
	if key == SettingKeyAccountHealthSettings {
		r.value = value
	}
	return nil
}
func (r *accountHealthSettingsRepoStub) Get(context.Context, string) (*Setting, error) {
	return nil, nil
}
func (r *accountHealthSettingsRepoStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	return nil, nil
}
func (r *accountHealthSettingsRepoStub) SetMultiple(context.Context, map[string]string) error {
	return nil
}
func (r *accountHealthSettingsRepoStub) GetAll(context.Context) (map[string]string, error) {
	return nil, nil
}
func (r *accountHealthSettingsRepoStub) Delete(context.Context, string) error { return nil }

func TestAccountHealthSettingsDefaultsAndPersistence(t *testing.T) {
	repo := &accountHealthSettingsRepoStub{}
	svc := NewAccountHealthService(nil, nil)
	svc.SetSettingsRepository(repo)
	cfg, err := svc.GetSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1440, cfg.ScanIntervalMinutes)
	require.True(t, cfg.AutoQuarantine)
	require.True(t, cfg.RequireDeleteConfirmation)
	cfg.ScanIntervalMinutes = 60
	cfg.RequireDeleteConfirmation = false
	updated, err := svc.UpdateSettings(context.Background(), cfg)
	require.NoError(t, err)
	require.Equal(t, 60, updated.ScanIntervalMinutes)
	require.True(t, updated.RequireDeleteConfirmation)
	reloaded, err := svc.GetSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, 60, reloaded.ScanIntervalMinutes)
}

func TestAccountHealthSettingsRejectsInvalidInterval(t *testing.T) {
	svc := NewAccountHealthService(nil, nil)
	_, err := svc.UpdateSettings(context.Background(), &AccountHealthSettings{ScanIntervalMinutes: 1})
	require.Error(t, err)
}
