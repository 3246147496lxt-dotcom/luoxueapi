package repository

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyRepositoryServiceTierPreferenceRoundTripSQLite(t *testing.T) {
	repo, client := newAPIKeyRepoSQLite(t)
	ctx := context.Background()
	user := mustCreateAPIKeyRepoUser(t, ctx, client, "service-tier-roundtrip@test.com")

	key := &service.APIKey{
		UserID: user.ID,
		Key:    "sk-service-tier-roundtrip",
		Name:   "Service tier round trip",
		Status: service.StatusActive,
	}
	require.NoError(t, repo.Create(ctx, key))
	require.Equal(t, service.ServiceTierPreferenceStandard, key.ServiceTierPreference)

	got, err := repo.GetByKey(ctx, key.Key)
	require.NoError(t, err)
	require.Equal(t, service.ServiceTierPreferenceStandard, got.ServiceTierPreference)
	authGot, err := repo.GetByKeyForAuth(ctx, key.Key)
	require.NoError(t, err)
	require.Equal(t, service.ServiceTierPreferenceStandard, authGot.ServiceTierPreference)

	key.ServiceTierPreference = service.ServiceTierPreferencePriority
	require.NoError(t, repo.Update(ctx, key))
	got, err = repo.GetByKey(ctx, key.Key)
	require.NoError(t, err)
	require.Equal(t, service.ServiceTierPreferencePriority, got.ServiceTierPreference)
	authGot, err = repo.GetByKeyForAuth(ctx, key.Key)
	require.NoError(t, err)
	require.Equal(t, service.ServiceTierPreferencePriority, authGot.ServiceTierPreference)

	invalid := *key
	invalid.ServiceTierPreference = "fast"
	require.ErrorIs(t, repo.Update(ctx, &invalid), service.ErrInvalidServiceTierPreference)
}
