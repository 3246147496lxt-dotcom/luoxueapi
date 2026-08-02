package repository

import (
	"context"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const quotaViewerInstallerTicketKeyPrefix = "quota_viewer_installer_download:v1:"

var errQuotaViewerInstallerRedisUnavailable = errors.New("quota viewer installer Redis client is unavailable")

type redisQuotaViewerInstallerTicketStore struct {
	rdb *redis.Client
}

func NewQuotaViewerInstallerTicketStore(rdb *redis.Client) service.QuotaViewerInstallerTicketStore {
	return &redisQuotaViewerInstallerTicketStore{rdb: rdb}
}

func (s *redisQuotaViewerInstallerTicketStore) Put(
	ctx context.Context,
	digest string,
	payload []byte,
	ttl time.Duration,
) error {
	if s == nil || s.rdb == nil {
		return errQuotaViewerInstallerRedisUnavailable
	}
	return s.rdb.Set(ctx, quotaViewerInstallerTicketKeyPrefix+digest, payload, ttl).Err()
}

func (s *redisQuotaViewerInstallerTicketStore) Consume(
	ctx context.Context,
	digest string,
) ([]byte, bool, error) {
	if s == nil || s.rdb == nil {
		return nil, false, errQuotaViewerInstallerRedisUnavailable
	}

	payload, err := s.rdb.GetDel(ctx, quotaViewerInstallerTicketKeyPrefix+digest).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return payload, true, nil
}
