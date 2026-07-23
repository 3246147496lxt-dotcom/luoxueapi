package repository

import (
	"context"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const embeddedPageLaunchKeyPrefix = "embedded_page_launch:v1:"

var errEmbeddedPageLaunchRedisUnavailable = errors.New("embedded page launch Redis client is unavailable")

type redisEmbeddedPageLaunchStore struct {
	rdb *redis.Client
}

// NewEmbeddedPageLaunchStore adapts Redis to the service launch-ticket port.
func NewEmbeddedPageLaunchStore(rdb *redis.Client) service.EmbeddedPageLaunchStore {
	return &redisEmbeddedPageLaunchStore{rdb: rdb}
}

func (s *redisEmbeddedPageLaunchStore) Put(ctx context.Context, digest string, payload []byte, ttl time.Duration) error {
	if s == nil || s.rdb == nil {
		return errEmbeddedPageLaunchRedisUnavailable
	}
	return s.rdb.Set(ctx, embeddedPageLaunchKeyPrefix+digest, payload, ttl).Err()
}

func (s *redisEmbeddedPageLaunchStore) Consume(ctx context.Context, digest string) ([]byte, bool, error) {
	if s == nil || s.rdb == nil {
		return nil, false, errEmbeddedPageLaunchRedisUnavailable
	}

	payload, err := s.rdb.GetDel(ctx, embeddedPageLaunchKeyPrefix+digest).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return payload, true, nil
}
