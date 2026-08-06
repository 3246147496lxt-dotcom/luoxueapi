package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type skillSettingRepositoryStub struct {
	key   string
	value string
	err   error
}

func (s *skillSettingRepositoryStub) Get(context.Context, string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *skillSettingRepositoryStub) GetValue(context.Context, string) (string, error) {
	panic("unexpected GetValue call")
}

func (s *skillSettingRepositoryStub) Set(_ context.Context, key, value string) error {
	s.key = key
	s.value = value
	return s.err
}

func (s *skillSettingRepositoryStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *skillSettingRepositoryStub) SetMultiple(context.Context, map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *skillSettingRepositoryStub) GetAll(context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *skillSettingRepositoryStub) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

func TestUpdateSkillMarketplaceEnabledNotifiesRuntime(t *testing.T) {
	repo := &skillSettingRepositoryStub{}
	settings := NewSettingService(repo, &config.Config{})
	notified := 0
	settings.SetOnUpdateCallback(func() { notified++ })

	require.NoError(t, settings.UpdateSkillMarketplaceEnabled(context.Background(), true))
	require.Equal(t, SettingKeySkillMarketplaceEnabled, repo.key)
	require.Equal(t, "true", repo.value)
	require.Equal(t, 1, notified)
}
