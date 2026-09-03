package repository

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const (
	libraryDownloadTicketKeyPrefix      = "library_download:v2:ticket:"
	libraryDownloadTicketIssueKeyPrefix = "library_download:v2:issue:"
)

var (
	errLibraryDownloadTicketRedisUnavailable = errors.New("library download ticket Redis client is unavailable")
	errLibraryDownloadTicketHandleCollision  = errors.New("library download ticket handle already exists")
)

var libraryDownloadTicketIssueLimitScript = redis.NewScript(`
local current = redis.call('INCR', KEYS[1])
local ttl = redis.call('PTTL', KEYS[1])
if current == 1 or ttl == -1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
return current
`)

var libraryDownloadTicketConsumeScript = redis.NewScript(`
local value = redis.call('GET', KEYS[1])
if not value then
  return nil
end
local separator = string.find(value, '\n', 1, true)
if not separator then
  return nil
end
if string.sub(value, 1, separator - 1) ~= ARGV[1] then
  return nil
end
redis.call('DEL', KEYS[1])
return string.sub(value, separator + 1)
`)

type redisLibraryDownloadTicketStore struct{ rdb *redis.Client }

func NewLibraryDownloadTicketStore(rdb *redis.Client) service.LibraryDownloadTicketStore {
	return &redisLibraryDownloadTicketStore{rdb: rdb}
}

func (s *redisLibraryDownloadTicketStore) Put(
	ctx context.Context,
	handle string,
	secretDigest string,
	payload []byte,
	ttl time.Duration,
) error {
	if s == nil || s.rdb == nil {
		return errLibraryDownloadTicketRedisUnavailable
	}
	if !validLibraryDownloadTicketSecretDigest(secretDigest) {
		return fmt.Errorf("invalid library download ticket secret digest")
	}
	storedPayload := make([]byte, 0, len(secretDigest)+1+len(payload))
	storedPayload = append(storedPayload, secretDigest...)
	storedPayload = append(storedPayload, '\n')
	storedPayload = append(storedPayload, payload...)
	stored, err := s.rdb.SetNX(ctx, libraryDownloadTicketKeyPrefix+handle, storedPayload, ttl).Result()
	if err != nil {
		return err
	}
	if !stored {
		return errLibraryDownloadTicketHandleCollision
	}
	return nil
}

func (s *redisLibraryDownloadTicketStore) Consume(
	ctx context.Context,
	handle string,
	secretDigest string,
) ([]byte, bool, error) {
	if s == nil || s.rdb == nil {
		return nil, false, errLibraryDownloadTicketRedisUnavailable
	}
	if !validLibraryDownloadTicketSecretDigest(secretDigest) {
		return nil, false, nil
	}
	payloadText, err := libraryDownloadTicketConsumeScript.Run(
		ctx,
		s.rdb,
		[]string{libraryDownloadTicketKeyPrefix + handle},
		secretDigest,
	).Text()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return []byte(payloadText), true, nil
}

func validLibraryDownloadTicketSecretDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 32
}

func (s *redisLibraryDownloadTicketStore) AllowIssue(
	ctx context.Context,
	userID int64,
	limit int64,
	window time.Duration,
) (bool, error) {
	if s == nil || s.rdb == nil {
		return false, errLibraryDownloadTicketRedisUnavailable
	}
	if userID <= 0 || limit <= 0 || window <= 0 {
		return false, fmt.Errorf("invalid library download ticket issue limit")
	}
	windowMilliseconds := window.Milliseconds()
	if windowMilliseconds < 1 {
		windowMilliseconds = 1
	}
	count, err := libraryDownloadTicketIssueLimitScript.Run(
		ctx,
		s.rdb,
		[]string{libraryDownloadTicketIssueKeyPrefix + strconv.FormatInt(userID, 10)},
		windowMilliseconds,
	).Int64()
	if err != nil {
		return false, err
	}
	return count <= limit, nil
}
