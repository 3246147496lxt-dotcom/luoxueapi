package repository

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestEmbeddedPageLaunchStoreUsesVersionedKeyAndAtomicConsume(t *testing.T) {
	ctx := context.Background()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { require.NoError(t, rdb.Close()) })
	store := NewEmbeddedPageLaunchStore(rdb)

	require.NoError(t, store.Put(ctx, "digest", []byte(`{"user_id":42}`), time.Minute))
	require.Equal(t, []string{embeddedPageLaunchKeyPrefix + "digest"}, mr.Keys())
	require.Equal(t, time.Minute, mr.TTL(embeddedPageLaunchKeyPrefix+"digest"))

	payload, found, err := store.Consume(ctx, "digest")
	require.NoError(t, err)
	require.True(t, found)
	require.JSONEq(t, `{"user_id":42}`, string(payload))

	payload, found, err = store.Consume(ctx, "digest")
	require.NoError(t, err)
	require.False(t, found)
	require.Nil(t, payload)
}

func TestEmbeddedPageLaunchStoreNilRedisReturnsError(t *testing.T) {
	store := NewEmbeddedPageLaunchStore(nil)

	require.Error(t, store.Put(context.Background(), "digest", []byte("ticket"), time.Minute))
	payload, found, err := store.Consume(context.Background(), "digest")
	require.Error(t, err)
	require.False(t, found)
	require.Nil(t, payload)
}
