package service

import (
	"context"
	"fmt"
	"strconv"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// UpdateSkillMarketplaceEnabled writes the public Skill gate through the
// canonical settings mutation path so runtime consumers invalidate injected
// HTML and other derived settings snapshots immediately.
func (s *SettingService) UpdateSkillMarketplaceEnabled(ctx context.Context, enabled bool) error {
	if s == nil || s.settingRepo == nil {
		return infraerrors.InternalServer(
			"SKILL_CONFIG_UNAVAILABLE",
			"skill marketplace config storage is unavailable",
		)
	}
	if err := s.settingRepo.Set(ctx, SettingKeySkillMarketplaceEnabled, strconv.FormatBool(enabled)); err != nil {
		return fmt.Errorf("update skill marketplace config: %w", err)
	}
	s.notifyUpdated()
	return nil
}
