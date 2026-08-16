//go:build unit

package repository

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestLibraryDownloadTicketStoreAtomicallyComparesSecretAndConsumesOnce(t *testing.T) {
	ctx := context.Background()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { require.NoError(t, rdb.Close()) })
	store := NewLibraryDownloadTicketStore(rdb)

	handle := "ticket-handle"
	digest := strings.Repeat("a", 64)
	wrongDigest := strings.Repeat("b", 64)
	require.NoError(t, store.Put(ctx, handle, digest, []byte(`{"user_id":7}`), time.Minute))
	require.Equal(t, []string{libraryDownloadTicketKeyPrefix + handle}, mr.Keys())
	require.Equal(t, time.Minute, mr.TTL(libraryDownloadTicketKeyPrefix+handle))

	payload, found, err := store.Consume(ctx, handle, wrongDigest)
	require.NoError(t, err)
	require.False(t, found)
	require.Nil(t, payload)
	require.True(t, mr.Exists(libraryDownloadTicketKeyPrefix+handle), "wrong secret must not burn the ticket")

	const contenders = 32
	var foundCount atomic.Int64
	var wg sync.WaitGroup
	errorsCh := make(chan error, contenders)
	for i := 0; i < contenders; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			candidate := wrongDigest
			if index == contenders/2 {
				candidate = digest
			}
			got, consumed, consumeErr := store.Consume(ctx, handle, candidate)
			if consumeErr != nil {
				errorsCh <- consumeErr
				return
			}
			if consumed {
				foundCount.Add(1)
				if string(got) != `{"user_id":7}` {
					errorsCh <- errors.New("unexpected consumed payload")
				}
			}
		}(i)
	}
	wg.Wait()
	close(errorsCh)
	for consumeErr := range errorsCh {
		require.NoError(t, consumeErr)
	}
	require.EqualValues(t, 1, foundCount.Load())
	require.False(t, mr.Exists(libraryDownloadTicketKeyPrefix+handle))
}

func TestLibraryDownloadTicketIssueLimiterIsAtomicAcrossInstances(t *testing.T) {
	ctx := context.Background()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { require.NoError(t, rdb.Close()) })
	first := NewLibraryDownloadTicketStore(rdb)
	second := NewLibraryDownloadTicketStore(rdb)

	for i := 0; i < 30; i++ {
		store := first
		if i%2 == 1 {
			store = second
		}
		allowed, err := store.AllowIssue(ctx, 7, 30, time.Minute)
		require.NoError(t, err)
		require.True(t, allowed)
	}
	allowed, err := second.AllowIssue(ctx, 7, 30, time.Minute)
	require.NoError(t, err)
	require.False(t, allowed)
	require.Equal(t, time.Minute, mr.TTL(libraryDownloadTicketIssueKeyPrefix+"7"))

	allowed, err = first.AllowIssue(ctx, 8, 30, time.Minute)
	require.NoError(t, err)
	require.True(t, allowed, "issue limits must be isolated per user")
}

func TestLibraryDownloadTicketStoreNilRedisFailsClosed(t *testing.T) {
	store := NewLibraryDownloadTicketStore(nil)
	digest := strings.Repeat("a", 64)
	require.Error(t, store.Put(context.Background(), "handle", digest, []byte("ticket"), time.Minute))
	payload, found, err := store.Consume(context.Background(), "handle", digest)
	require.Error(t, err)
	require.False(t, found)
	require.Nil(t, payload)
	allowed, err := store.AllowIssue(context.Background(), 7, 30, time.Minute)
	require.Error(t, err)
	require.False(t, allowed)
}
