package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/modules/desktop"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestDesktopPairingStoreExpiresAndConsumesOnce(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	store, ok := NewDesktopPairingStore(client).(*desktopPairingStore)
	require.True(t, ok)
	ctx := context.Background()
	pairing := desktop.Pairing{
		DeviceCodeHash: desktopDeviceCodeHash("device-secret"), UserCode: "ABCD-EFGH",
		Status: desktop.PairingStatusPending, ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	require.NoError(t, store.Create(ctx, pairing, 10*time.Minute))

	claimed, err := store.BeginApproval(ctx, pairing.UserCode)
	require.NoError(t, err)
	require.Equal(t, desktop.PairingStatusApproving, claimed.Status)
	require.NoError(t, store.FinishApproval(ctx, pairing.UserCode, 7, 41))
	consumed, err := store.Consume(ctx, "device-secret")
	require.NoError(t, err)
	require.Equal(t, int64(41), consumed.DeviceID)
	_, err = store.Consume(ctx, "device-secret")
	require.ErrorIs(t, err, desktop.ErrPairingConsumed)

	server.FastForward(11 * time.Minute)
	_, err = store.GetByDeviceCode(ctx, "device-secret")
	require.True(t, errors.Is(err, desktop.ErrPairingExpired))
}
