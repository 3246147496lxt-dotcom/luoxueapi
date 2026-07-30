package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/modules/quotaauth"
	"github.com/redis/go-redis/v9"
)

const (
	quotaAuthPairDevicePrefix = "quota_auth:pair:device:"
	quotaAuthPairUserPrefix   = "quota_auth:pair:user:"
)

var (
	quotaAuthPairCreateScript = redis.NewScript(`
if redis.call('EXISTS', KEYS[1]) == 1 or redis.call('EXISTS', KEYS[2]) == 1 then
  return 0
end
redis.call('HSET', KEYS[1], 'payload', ARGV[1], 'status', 'pending', 'user_id', '0', 'device_id', '0')
redis.call('PEXPIRE', KEYS[1], ARGV[2])
redis.call('SET', KEYS[2], ARGV[3], 'PX', ARGV[2])
return 1
`)
	quotaAuthPairBeginApprovalScript = redis.NewScript(`
local device_hash = redis.call('GET', KEYS[1])
if not device_hash then return {'missing', ''} end
local device_key = ARGV[1] .. device_hash
local state = redis.call('HGET', device_key, 'status')
if not state then return {'missing', ''} end
if state ~= 'pending' then return {state, device_hash} end
redis.call('HSET', device_key, 'status', 'approving')
return {'ok', device_hash}
`)
	quotaAuthPairFinishApprovalScript = redis.NewScript(`
local device_hash = redis.call('GET', KEYS[1])
if not device_hash then return 'missing' end
local device_key = ARGV[1] .. device_hash
local state = redis.call('HGET', device_key, 'status')
if not state then return 'missing' end
if state ~= 'approving' then return state end
redis.call('HSET', device_key, 'status', 'approved', 'user_id', ARGV[2], 'device_id', ARGV[3])
return 'ok'
`)
	quotaAuthPairCancelApprovalScript = redis.NewScript(`
local device_hash = redis.call('GET', KEYS[1])
if not device_hash then return 0 end
local device_key = ARGV[1] .. device_hash
if redis.call('HGET', device_key, 'status') == 'approving' then
  redis.call('HSET', device_key, 'status', 'pending')
  return 1
end
return 0
`)
	quotaAuthPairConsumeScript = redis.NewScript(`
local state = redis.call('HGET', KEYS[1], 'status')
if not state then return 'missing' end
if state ~= 'approved' then return state end
redis.call('HSET', KEYS[1], 'status', 'consumed')
return 'ok'
`)
	quotaAuthPairRestoreConsumedScript = redis.NewScript(`
if redis.call('HGET', KEYS[1], 'status') == 'consumed' then
  redis.call('HSET', KEYS[1], 'status', 'approved')
  return 1
end
return 0
`)
)

type quotaAuthPairingStore struct {
	rdb *redis.Client
}

func NewQuotaAuthPairingStore(rdb *redis.Client) quotaauth.PairingStore {
	return &quotaAuthPairingStore{rdb: rdb}
}

func (s *quotaAuthPairingStore) Create(ctx context.Context, pairing quotaauth.Pairing, ttl time.Duration) error {
	if s == nil || s.rdb == nil {
		return errors.New("quota viewer pairing store is unavailable")
	}
	payload, err := json.Marshal(pairing)
	if err != nil {
		return fmt.Errorf("marshal quota viewer pairing: %w", err)
	}
	created, err := quotaAuthPairCreateScript.Run(
		ctx,
		s.rdb,
		[]string{
			quotaAuthPairDeviceKey(pairing.DeviceCodeHash),
			quotaAuthPairUserKey(pairing.UserCode),
		},
		payload,
		ttl.Milliseconds(),
		pairing.DeviceCodeHash,
	).Int()
	if err != nil {
		return fmt.Errorf("create quota viewer pairing: %w", err)
	}
	if created != 1 {
		return quotaauth.ErrPairingCollision
	}
	return nil
}

func (s *quotaAuthPairingStore) GetByUserCode(ctx context.Context, userCode string) (*quotaauth.Pairing, error) {
	if s == nil || s.rdb == nil {
		return nil, errors.New("quota viewer pairing store is unavailable")
	}
	deviceHash, err := s.rdb.Get(ctx, quotaAuthPairUserKey(userCode)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, quotaauth.ErrPairingExpired
	}
	if err != nil {
		return nil, fmt.Errorf("read quota viewer pairing user code: %w", err)
	}
	return s.getByHash(ctx, deviceHash)
}

func (s *quotaAuthPairingStore) GetByDeviceCode(ctx context.Context, deviceCode string) (*quotaauth.Pairing, error) {
	if s == nil || s.rdb == nil {
		return nil, errors.New("quota viewer pairing store is unavailable")
	}
	return s.getByHash(ctx, quotaAuthDeviceCodeHash(deviceCode))
}

func (s *quotaAuthPairingStore) BeginApproval(ctx context.Context, userCode string) (*quotaauth.Pairing, error) {
	if s == nil || s.rdb == nil {
		return nil, errors.New("quota viewer pairing store is unavailable")
	}
	result, err := quotaAuthPairBeginApprovalScript.Run(
		ctx,
		s.rdb,
		[]string{quotaAuthPairUserKey(userCode)},
		quotaAuthPairDevicePrefix,
	).Slice()
	if err != nil {
		return nil, fmt.Errorf("claim quota viewer pairing approval: %w", err)
	}
	state, deviceHash := quotaAuthScriptPair(result)
	if state == "missing" || deviceHash == "" {
		return nil, quotaauth.ErrPairingExpired
	}
	if state != "ok" {
		return nil, quotaauth.ErrPairingState
	}
	return s.getByHash(ctx, deviceHash)
}

func (s *quotaAuthPairingStore) FinishApproval(
	ctx context.Context,
	userCode string,
	userID, deviceID int64,
) error {
	if s == nil || s.rdb == nil {
		return errors.New("quota viewer pairing store is unavailable")
	}
	state, err := quotaAuthPairFinishApprovalScript.Run(
		ctx,
		s.rdb,
		[]string{quotaAuthPairUserKey(userCode)},
		quotaAuthPairDevicePrefix,
		strconv.FormatInt(userID, 10),
		strconv.FormatInt(deviceID, 10),
	).Text()
	if err != nil {
		return fmt.Errorf("finish quota viewer pairing approval: %w", err)
	}
	if state == "missing" {
		return quotaauth.ErrPairingExpired
	}
	if state != "ok" {
		return quotaauth.ErrPairingState
	}
	return nil
}

func (s *quotaAuthPairingStore) CancelApproval(ctx context.Context, userCode string) error {
	if s == nil || s.rdb == nil {
		return errors.New("quota viewer pairing store is unavailable")
	}
	if err := quotaAuthPairCancelApprovalScript.Run(
		ctx,
		s.rdb,
		[]string{quotaAuthPairUserKey(userCode)},
		quotaAuthPairDevicePrefix,
	).Err(); err != nil {
		return fmt.Errorf("cancel quota viewer pairing approval: %w", err)
	}
	return nil
}

func (s *quotaAuthPairingStore) Consume(ctx context.Context, deviceCode string) (*quotaauth.Pairing, error) {
	if s == nil || s.rdb == nil {
		return nil, errors.New("quota viewer pairing store is unavailable")
	}
	deviceHash := quotaAuthDeviceCodeHash(deviceCode)
	key := quotaAuthPairDeviceKey(deviceHash)
	state, err := quotaAuthPairConsumeScript.Run(ctx, s.rdb, []string{key}).Text()
	if err != nil {
		return nil, fmt.Errorf("consume quota viewer pairing: %w", err)
	}
	switch state {
	case "ok":
		return s.getByHash(ctx, deviceHash)
	case "missing":
		return nil, quotaauth.ErrPairingExpired
	case quotaauth.PairingStatusConsumed:
		return nil, quotaauth.ErrPairingConsumed
	default:
		return nil, quotaauth.ErrPairingState
	}
}

func (s *quotaAuthPairingStore) RestoreConsumed(ctx context.Context, deviceCode string) error {
	if s == nil || s.rdb == nil {
		return errors.New("quota viewer pairing store is unavailable")
	}
	if err := quotaAuthPairRestoreConsumedScript.Run(
		ctx,
		s.rdb,
		[]string{quotaAuthPairDeviceKey(quotaAuthDeviceCodeHash(deviceCode))},
	).Err(); err != nil {
		return fmt.Errorf("restore quota viewer pairing exchange: %w", err)
	}
	return nil
}

func (s *quotaAuthPairingStore) getByHash(ctx context.Context, deviceHash string) (*quotaauth.Pairing, error) {
	fields, err := s.rdb.HGetAll(ctx, quotaAuthPairDeviceKey(deviceHash)).Result()
	if err != nil {
		return nil, fmt.Errorf("read quota viewer pairing: %w", err)
	}
	if len(fields) == 0 || fields["payload"] == "" {
		return nil, quotaauth.ErrPairingExpired
	}
	var pairing quotaauth.Pairing
	if err := json.Unmarshal([]byte(fields["payload"]), &pairing); err != nil {
		return nil, fmt.Errorf("decode quota viewer pairing: %w", err)
	}
	pairing.Status = fields["status"]
	pairing.UserID, _ = strconv.ParseInt(fields["user_id"], 10, 64)
	pairing.DeviceID, _ = strconv.ParseInt(fields["device_id"], 10, 64)
	if !pairing.ExpiresAt.After(time.Now()) {
		return nil, quotaauth.ErrPairingExpired
	}
	return &pairing, nil
}

func quotaAuthPairDeviceKey(hash string) string {
	return quotaAuthPairDevicePrefix + hash
}

func quotaAuthPairUserKey(code string) string {
	normalized := strings.ToUpper(strings.TrimSpace(code))
	digest := sha256.Sum256([]byte(normalized))
	return quotaAuthPairUserPrefix + hex.EncodeToString(digest[:])
}

func quotaAuthDeviceCodeHash(deviceCode string) string {
	digest := sha256.Sum256([]byte(deviceCode))
	return hex.EncodeToString(digest[:])
}

func quotaAuthScriptPair(result []any) (string, string) {
	if len(result) != 2 {
		return "missing", ""
	}
	return fmt.Sprint(result[0]), fmt.Sprint(result[1])
}
