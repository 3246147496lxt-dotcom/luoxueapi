package service

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type publicModelCatalogRuntimeRepoStub struct {
	SettingRepository

	mu               sync.Mutex
	values           map[string]string
	getMultipleCalls int
	getMultipleErr   error
}

func (s *publicModelCatalogRuntimeRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.getMultipleCalls++
	if s.getMultipleErr != nil {
		return nil, s.getMultipleErr
	}
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		out[key] = s.values[key]
	}
	return out, nil
}

func (s *publicModelCatalogRuntimeRepoStub) SetMultiple(_ context.Context, settings map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.values == nil {
		s.values = make(map[string]string)
	}
	for key, value := range settings {
		s.values[key] = value
	}
	return nil
}

func (s *publicModelCatalogRuntimeRepoStub) calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getMultipleCalls
}

func TestPublicModelCatalogRuntimeCachesDisabledState(t *testing.T) {
	repo := &publicModelCatalogRuntimeRepoStub{values: map[string]string{
		SettingKeyPublicModelCatalogEnabled: "false",
		SettingKeyBackendModeEnabled:        "false",
	}}
	service := NewSettingService(repo, nil)

	for range 20 {
		runtime := service.GetPublicModelCatalogRuntime(context.Background())
		require.False(t, runtime.Enabled)
		require.False(t, runtime.BackendMode)
	}
	require.Equal(t, 1, repo.calls())
}

func TestPublicModelCatalogRuntimeCachesFailClosedStoreError(t *testing.T) {
	repo := &publicModelCatalogRuntimeRepoStub{getMultipleErr: errors.New("settings unavailable")}
	service := NewSettingService(repo, nil)

	require.Equal(t, PublicModelCatalogRuntime{}, service.GetPublicModelCatalogRuntime(context.Background()))
	require.Equal(t, PublicModelCatalogRuntime{}, service.GetPublicModelCatalogRuntime(context.Background()))
	require.Equal(t, 1, repo.calls())
}

func TestUpdateSettingsRefreshesPublicModelCatalogRuntimeCache(t *testing.T) {
	repo := &publicModelCatalogRuntimeRepoStub{values: map[string]string{
		SettingKeyPublicModelCatalogEnabled: "false",
		SettingKeyBackendModeEnabled:        "false",
	}}
	service := NewSettingService(repo, nil)

	require.False(t, service.GetPublicModelCatalogRuntime(context.Background()).Enabled)
	require.Equal(t, 1, repo.calls())

	require.NoError(t, service.UpdateSettings(context.Background(), &SystemSettings{
		PublicModelCatalogEnabled: true,
		BackendModeEnabled:        false,
	}))
	runtime := service.GetPublicModelCatalogRuntime(context.Background())
	require.True(t, runtime.Enabled)
	require.False(t, runtime.BackendMode)
	// The successful update refreshes the gate snapshot directly; it neither
	// serves the stale disabled value nor needs another settings read.
	require.Equal(t, 1, repo.calls())
}
