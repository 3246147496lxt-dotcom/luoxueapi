package migrations

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGroupProfitControlMigrationAddsOnlyInertColumns(t *testing.T) {
	content, err := FS.ReadFile("233_group_profit_control.sql")
	require.NoError(t, err)

	sql := compactSQLWithoutLineComments(string(content))
	require.Contains(t, sql, "ALTER TABLE groups")
	require.Equal(t, 3, strings.Count(sql, "ADD COLUMN IF NOT EXISTS"))
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS profit_control_enabled BOOLEAN NOT NULL DEFAULT FALSE")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS profit_min_margin NUMERIC(10,4) NOT NULL DEFAULT 0")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS profit_safety_buffer NUMERIC(10,4) NOT NULL DEFAULT 0")

	// This migration must remain an additive, default-off schema change. In
	// particular, it must not rewrite existing group rows or remove/rename any
	// existing schema objects.
	destructive := regexp.MustCompile(`(?i)\b(DROP|DELETE|TRUNCATE|UPDATE|INSERT|RENAME)\b`)
	require.Falsef(t, destructive.MatchString(sql), "migration contains a destructive or data-rewriting statement: %s", sql)
	require.Equal(t, 1, strings.Count(sql, "ALTER TABLE"))
	require.Equal(t, 1, strings.Count(sql, ";"))
}

func compactSQLWithoutLineComments(sql string) string {
	lines := strings.Split(sql, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		if idx := strings.Index(line, "--"); idx >= 0 {
			line = line[:idx]
		}
		kept = append(kept, line)
	}
	return strings.Join(strings.Fields(strings.Join(kept, "\n")), " ")
}
