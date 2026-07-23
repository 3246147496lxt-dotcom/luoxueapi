package service

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// StartRuntime performs the settings work that requires live infrastructure.
// Construction and Wire providers deliberately do not read or mutate the
// database and do not install package-global callbacks.
func (s *SettingService) StartRuntime(ctx context.Context) error {
	if s == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := s.LoadAPIKeyACLTrustForwardedIPSetting(ctx); err != nil {
		logger.LegacyPrintf("service.setting", "Warning: load api key acl forwarded ip setting failed: %v", err)
	}
	if err := s.MigrateOpenAIAllowClaudeCodeCodexPluginSetting(ctx); err != nil {
		logger.LegacyPrintf("service.setting", "Warning: migrate openai allow Claude Code Codex plugin setting failed: %v", err)
	}
	if err := s.MigrateCodexBodyFingerprintToSignals(ctx); err != nil {
		logger.LegacyPrintf("service.setting", "Warning: migrate codex body fingerprint to signals failed: %v", err)
	}
	s.WarmOpenAIQuotaAutoPauseSettings(ctx)
	antigravity.SetUserAgentVersionResolver(s.GetAntigravityUserAgentVersion)
	s.rebuildWebSearchManager(ctx)
	s.notifyUpdated()
	return nil
}

// StopRuntime removes callbacks whose lifetime is owned by the process
// supervisor. It is safe to call repeatedly.
func (s *SettingService) StopRuntime() {
	s.ConfigureWebSearchManagerBuilder(nil)
	antigravity.SetUserAgentVersionResolver(nil)
	SetWebSearchManager(nil)
}
