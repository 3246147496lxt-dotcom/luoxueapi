package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/modules/quotaauth"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestQuotaAuthPairingStoreIsNamespacedExpiresAndConsumesOnce(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	store := NewQuotaAuthPairingStore(client).(*quotaAuthPairingStore)
	ctx := context.Background()
	pairing := quotaauth.Pairing{
		DeviceCodeHash: quotaAuthDeviceCodeHash("device-secret"),
		UserCode:       "ABCD-EFGH",
		ClientID:       quotaauth.ClientID,
		Scope:          quotaauth.ScopeRead,
		Status:         quotaauth.PairingStatusPending,
		ExpiresAt:      time.Now().Add(10 * time.Minute),
	}
	require.NoError(t, store.Create(ctx, pairing, 10*time.Minute))

	keys := server.Keys()
	require.Len(t, keys, 2)
	for _, key := range keys {
		require.Contains(t, key, "quota_auth:pair:")
		require.NotContains(t, key, "desktop:pair:")
	}

	claimed, err := store.BeginApproval(ctx, pairing.UserCode)
	require.NoError(t, err)
	require.Equal(t, quotaauth.PairingStatusApproving, claimed.Status)
	require.NoError(t, store.FinishApproval(ctx, pairing.UserCode, 7, 41))
	consumed, err := store.Consume(ctx, "device-secret")
	require.NoError(t, err)
	require.Equal(t, int64(7), consumed.UserID)
	require.Equal(t, int64(41), consumed.DeviceID)
	_, err = store.Consume(ctx, "device-secret")
	require.ErrorIs(t, err, quotaauth.ErrPairingConsumed)

	server.FastForward(11 * time.Minute)
	_, err = store.GetByDeviceCode(ctx, "device-secret")
	require.True(t, errors.Is(err, quotaauth.ErrPairingExpired))
}
