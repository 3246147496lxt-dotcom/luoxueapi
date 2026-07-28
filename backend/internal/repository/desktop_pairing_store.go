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

	"github.com/Wei-Shaw/sub2api/internal/modules/desktop"
	"github.com/redis/go-redis/v9"
)

const (
	desktopPairDevicePrefix = "desktop:pair:device:"
	desktopPairUserPrefix   = "desktop:pair:user:"
)

var (
	desktopPairCreateScript = redis.NewScript(`
if redis.call('EXISTS', KEYS[1]) == 1 or redis.call('EXISTS', KEYS[2]) == 1 then
  return 0
end
redis.call('HSET', KEYS[1], 'payload', ARGV[1], 'status', 'pending', 'user_id', '0', 'device_id', '0')
redis.call('PEXPIRE', KEYS[1], ARGV[2])
redis.call('SET', KEYS[2], ARGV[3], 'PX', ARGV[2])
return 1
`)
	desktopPairBeginApprovalScript = redis.NewScript(`
local device_hash = redis.call('GET', KEYS[1])
if not device_hash then return {'missing', ''} end
local device_key = ARGV[1] .. device_hash
local state = redis.call('HGET', device_key, 'status')
if not state then return {'missing', ''} end
if state ~= 'pending' then return {state, device_hash} end
redis.call('HSET', device_key, 'status', 'approving')
return {'ok', device_hash}
`)
	desktopPairFinishApprovalScript = redis.NewScript(`
local device_hash = redis.call('GET', KEYS[1])
if not device_hash then return 'missing' end
local device_key = ARGV[1] .. device_hash
local state = redis.call('HGET', device_key, 'status')
if not state then return 'missing' end
if state ~= 'approving' then return state end
redis.call('HSET', device_key, 'status', 'approved', 'user_id', ARGV[2], 'device_id', ARGV[3])
return 'ok'
`)
	desktopPairCancelApprovalScript = redis.NewScript(`
local device_hash = redis.call('GET', KEYS[1])
if not device_hash then return 0 end
local device_key = ARGV[1] .. device_hash
if redis.call('HGET', device_key, 'status') == 'approving' then
  redis.call('HSET', device_key, 'status', 'pending')
  return 1
end
return 0
`)
	desktopPairConsumeScript = redis.NewScript(`
local state = redis.call('HGET', KEYS[1], 'status')
if not state then return 'missing' end
if state ~= 'approved' then return state end
redis.call('HSET', KEYS[1], 'status', 'consumed')
return 'ok'
`)
	desktopPairRestoreConsumedScript = redis.NewScript(`
if redis.call('HGET', KEYS[1], 'status') == 'consumed' then
  redis.call('HSET', KEYS[1], 'status', 'approved')
  return 1
end
return 0
`)
)

type desktopPairingStore struct {
	rdb *redis.Client
}

func NewDesktopPairingStore(rdb *redis.Client) desktop.PairingStore {
	return &desktopPairingStore{rdb: rdb}
}

func (s *desktopPairingStore) Create(ctx context.Context, pairing desktop.Pairing, ttl time.Duration) error {
	if s == nil || s.rdb == nil {
		return errors.New("desktop pairing store is unavailable")
	}
	payload, err := json.Marshal(pairing)
	if err != nil {
		return fmt.Errorf("marshal desktop pairing: %w", err)
	}
	created, err := desktopPairCreateScript.Run(ctx, s.rdb,
		[]string{desktopPairDeviceKey(pairing.DeviceCodeHash), desktopPairUserKey(pairing.UserCode)},
		payload, ttl.Milliseconds(), pairing.DeviceCodeHash,
	).Int()
	if err != nil {
		return fmt.Errorf("create desktop pairing: %w", err)
	}
	if created != 1 {
		return desktop.ErrPairingCollision
	}
	return nil
}

func (s *desktopPairingStore) GetByUserCode(ctx context.Context, userCode string) (*desktop.Pairing, error) {
	deviceHash, err := s.rdb.Get(ctx, desktopPairUserKey(userCode)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, desktop.ErrPairingExpired
	}
	if err != nil {
		return nil, fmt.Errorf("read desktop pairing user code: %w", err)
	}
	return s.getByHash(ctx, deviceHash)
}

func (s *desktopPairingStore) GetByDeviceCode(ctx context.Context, deviceCode string) (*desktop.Pairing, error) {
	return s.getByHash(ctx, desktopDeviceCodeHash(deviceCode))
}

func (s *desktopPairingStore) BeginApproval(ctx context.Context, userCode string) (*desktop.Pairing, error) {
	result, err := desktopPairBeginApprovalScript.Run(ctx, s.rdb,
		[]string{desktopPairUserKey(userCode)}, desktopPairDevicePrefix,
	).Slice()
	if err != nil {
		return nil, fmt.Errorf("claim desktop pairing approval: %w", err)
	}
	state, deviceHash := scriptPair(result)
	if state == "missing" || deviceHash == "" {
		return nil, desktop.ErrPairingExpired
	}
	if state != "ok" {
		return nil, desktop.ErrPairingState
	}
	return s.getByHash(ctx, deviceHash)
}

func (s *desktopPairingStore) FinishApproval(ctx context.Context, userCode string, userID, deviceID int64) error {
	state, err := desktopPairFinishApprovalScript.Run(ctx, s.rdb,
		[]string{desktopPairUserKey(userCode)}, desktopPairDevicePrefix,
		strconv.FormatInt(userID, 10), strconv.FormatInt(deviceID, 10),
	).Text()
	if err != nil {
		return fmt.Errorf("finish desktop pairing approval: %w", err)
	}
	if state == "missing" {
		return desktop.ErrPairingExpired
	}
	if state != "ok" {
		return desktop.ErrPairingState
	}
	return nil
}

func (s *desktopPairingStore) CancelApproval(ctx context.Context, userCode string) error {
	if err := desktopPairCancelApprovalScript.Run(ctx, s.rdb,
		[]string{desktopPairUserKey(userCode)}, desktopPairDevicePrefix,
	).Err(); err != nil {
		return fmt.Errorf("cancel desktop pairing approval: %w", err)
	}
	return nil
}

func (s *desktopPairingStore) Consume(ctx context.Context, deviceCode string) (*desktop.Pairing, error) {
	deviceHash := desktopDeviceCodeHash(deviceCode)
	key := desktopPairDeviceKey(deviceHash)
	state, err := desktopPairConsumeScript.Run(ctx, s.rdb, []string{key}).Text()
	if err != nil {
		return nil, fmt.Errorf("consume desktop pairing: %w", err)
	}
	switch state {
	case "ok":
		return s.getByHash(ctx, deviceHash)
	case "missing":
		return nil, desktop.ErrPairingExpired
	case desktop.PairingStatusConsumed:
		return nil, desktop.ErrPairingConsumed
	default:
		return nil, desktop.ErrPairingState
	}
}

func (s *desktopPairingStore) RestoreConsumed(ctx context.Context, deviceCode string) error {
	if err := desktopPairRestoreConsumedScript.Run(ctx, s.rdb,
		[]string{desktopPairDeviceKey(desktopDeviceCodeHash(deviceCode))},
	).Err(); err != nil {
		return fmt.Errorf("restore desktop pairing exchange: %w", err)
	}
	return nil
}

func (s *desktopPairingStore) getByHash(ctx context.Context, deviceHash string) (*desktop.Pairing, error) {
	fields, err := s.rdb.HGetAll(ctx, desktopPairDeviceKey(deviceHash)).Result()
	if err != nil {
		return nil, fmt.Errorf("read desktop pairing: %w", err)
	}
	if len(fields) == 0 || fields["payload"] == "" {
		return nil, desktop.ErrPairingExpired
	}
	var pairing desktop.Pairing
	if err := json.Unmarshal([]byte(fields["payload"]), &pairing); err != nil {
		return nil, fmt.Errorf("decode desktop pairing: %w", err)
	}
	pairing.Status = fields["status"]
	pairing.UserID, _ = strconv.ParseInt(fields["user_id"], 10, 64)
	pairing.DeviceID, _ = strconv.ParseInt(fields["device_id"], 10, 64)
	if !pairing.ExpiresAt.After(time.Now()) {
		return nil, desktop.ErrPairingExpired
	}
	return &pairing, nil
}

func desktopPairDeviceKey(hash string) string { return desktopPairDevicePrefix + hash }

func desktopPairUserKey(code string) string {
	normalized := strings.ToUpper(strings.TrimSpace(code))
	digest := sha256.Sum256([]byte(normalized))
	return desktopPairUserPrefix + hex.EncodeToString(digest[:])
}

func desktopDeviceCodeHash(deviceCode string) string {
	digest := sha256.Sum256([]byte(deviceCode))
	return hex.EncodeToString(digest[:])
}

func scriptPair(result []any) (string, string) {
	if len(result) != 2 {
		return "missing", ""
	}
	return fmt.Sprint(result[0]), fmt.Sprint(result[1])
}
