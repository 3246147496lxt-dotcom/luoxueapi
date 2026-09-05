package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"time"
)

const (
	accountSchedulingThresholdsCacheTTL  = 30 * time.Second
	accountSchedulingThresholdsErrorTTL  = 5 * time.Second
	accountSchedulingThresholdsDBTimeout = 2 * time.Second
)

type cachedAccountSchedulingThresholds struct {
	thresholds map[string]int
	expiresAt  int64
}

// defaultAccountSchedulingThresholds returns a fail-open policy. A threshold
// of 100 means the provider is eligible for scheduling until an operator
// explicitly chooses a lower value.
func defaultAccountSchedulingThresholds() map[string]int {
	return map[string]int{
		PlatformOpenAI:    100,
		PlatformAnthropic: 100,
		PlatformGrok:      100,
		PlatformZhipu:     100,
	}
}

func cloneAccountSchedulingThresholds(input map[string]int) map[string]int {
	if len(input) == 0 {
		return defaultAccountSchedulingThresholds()
	}
	out := make(map[string]int, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func parseAccountSchedulingThresholdsSetting(raw string) (map[string]int, error) {
	normalized := defaultAccountSchedulingThresholds()
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return normalized, nil
	}
	var parsed map[string]int
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return normalized, err
	}
	for platform, value := range parsed {
		if !isAllowedSchedulingThresholdPlatform(platform) {
			return normalized, fmt.Errorf("unsupported scheduling threshold platform %q", platform)
		}
		if value < 1 || value > 100 {
			return normalized, fmt.Errorf("threshold for %s must be between 1 and 100", platform)
		}
		normalized[platform] = value
	}
	return normalized, nil
}

func validateAndNormalizeAccountSchedulingThresholds(input map[string]int) (map[string]int, error) {
	normalized := defaultAccountSchedulingThresholds()
	for platform, value := range input {
		if !isAllowedSchedulingThresholdPlatform(platform) {
			return nil, fmt.Errorf("unsupported scheduling threshold platform %q", platform)
		}
		if value < 1 || value > 100 {
			return nil, fmt.Errorf("threshold for %s must be between 1 and 100", platform)
		}
		normalized[platform] = value
	}
	return normalized, nil
}

// GetAccountSchedulingThresholds reads the global percentage thresholds with a
// short stale-while-revalidate cache. Errors fail open to the defaults so an
// unavailable settings database cannot take every account out of rotation.
func (s *SettingService) GetAccountSchedulingThresholds(ctx context.Context) map[string]int {
	defaults := defaultAccountSchedulingThresholds()
	if s == nil || s.settingRepo == nil {
		return defaults
	}
	now := time.Now().UnixNano()
	if cached, ok := s.accountSchedulingThresholdsCache.Load().(*cachedAccountSchedulingThresholds); ok &&
		cached != nil && cached.expiresAt > now && len(cached.thresholds) > 0 {
		return cloneAccountSchedulingThresholds(cached.thresholds)
	}

	if ctx == nil {
		ctx = context.Background()
	}
	result, err, _ := s.accountSchedulingThresholdsSF.Do(SettingKeyAccountSchedulingThresholds, func() (any, error) {
		now := time.Now().UnixNano()
		if cached, ok := s.accountSchedulingThresholdsCache.Load().(*cachedAccountSchedulingThresholds); ok &&
			cached != nil && cached.expiresAt > now && len(cached.thresholds) > 0 {
			return cloneAccountSchedulingThresholds(cached.thresholds), nil
		}
		readCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), accountSchedulingThresholdsDBTimeout)
		defer cancel()
		raw, readErr := s.settingRepo.GetValue(readCtx, SettingKeyAccountSchedulingThresholds)
		if readErr != nil {
			if !errors.Is(readErr, ErrSettingNotFound) {
				slog.Warn("account_scheduling_thresholds_read_failed", "error", readErr)
			}
			s.accountSchedulingThresholdsCache.Store(&cachedAccountSchedulingThresholds{
				thresholds: cloneAccountSchedulingThresholds(defaults),
				expiresAt:  time.Now().Add(accountSchedulingThresholdsErrorTTL).UnixNano(),
			})
			return cloneAccountSchedulingThresholds(defaults), nil
		}
		thresholds := defaults
		if parsed, parseErr := parseAccountSchedulingThresholdsSetting(raw); parseErr != nil {
			slog.Warn("account_scheduling_thresholds_parse_failed", "error", parseErr)
		} else {
			thresholds = parsed
		}
		s.accountSchedulingThresholdsCache.Store(&cachedAccountSchedulingThresholds{
			thresholds: cloneAccountSchedulingThresholds(thresholds),
			expiresAt:  time.Now().Add(accountSchedulingThresholdsCacheTTL).UnixNano(),
		})
		return cloneAccountSchedulingThresholds(thresholds), nil
	})
	if err != nil {
		return defaults
	}
	if thresholds, ok := result.(map[string]int); ok {
		return cloneAccountSchedulingThresholds(thresholds)
	}
	return defaults
}

// SetAccountSchedulingThresholds validates and persists the global policy.
func (s *SettingService) SetAccountSchedulingThresholds(ctx context.Context, thresholds map[string]int) error {
	if s == nil || s.settingRepo == nil {
		return errors.New("setting service is not configured")
	}
	normalized, err := validateAndNormalizeAccountSchedulingThresholds(thresholds)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(normalized)
	if err != nil {
		return err
	}
	if err := s.settingRepo.Set(ctx, SettingKeyAccountSchedulingThresholds, string(raw)); err != nil {
		return err
	}
	s.accountSchedulingThresholdsSF.Forget(SettingKeyAccountSchedulingThresholds)
	s.accountSchedulingThresholdsCache.Store(&cachedAccountSchedulingThresholds{
		thresholds: cloneAccountSchedulingThresholds(normalized),
		expiresAt:  time.Now().Add(accountSchedulingThresholdsCacheTTL).UnixNano(),
	})
	return nil
}

// accountSchedulingThresholdValue is useful to callers that accept JSON
// numbers from credentials or HTTP payloads. It intentionally rounds decimal
// values before validation, matching the pure evaluator.
func accountSchedulingThresholdValue(raw any) (int, bool) {
	switch value := raw.(type) {
	case int:
		return value, true
	case int64:
		return int(value), true
	case float64:
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return 0, false
		}
		return int(math.Round(value)), true
	case json.Number:
		parsed, err := value.Float64()
		if err != nil {
			return 0, false
		}
		return int(math.Round(parsed)), true
	case string:
		n, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil {
			return 0, false
		}
		return int(math.Round(n)), true
	default:
		return 0, false
	}
}
