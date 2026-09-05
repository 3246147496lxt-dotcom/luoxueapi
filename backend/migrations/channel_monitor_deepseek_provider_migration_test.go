package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChannelMonitorDeepSeekProviderMigration(t *testing.T) {
	content, err := FS.ReadFile("248_channel_monitor_deepseek_provider.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "channel_monitors_provider_check")
	require.Contains(t, sql, "channel_monitor_request_templates_provider_check")
	require.Contains(t, sql, "CHECK (provider IN ('openai', 'anthropic', 'gemini', 'grok', 'deepseek'))")
	require.Contains(t, sql, "position('deepseek' IN monitor_constraint_def) = 0")
	require.Contains(t, sql, "position('deepseek' IN template_constraint_def) = 0")
}
