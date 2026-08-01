package repository

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigration195PreflightScriptRejectsCredentialsInDBName(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("preflight script requires bash")
	}

	for _, dbname := range []string{
		"host=localhost dbname=sub2api PASSWORD=secret",
		"postgres://viewer:secret@localhost/sub2api",
		"postgres://viewer@localhost/sub2api",
		"postgres://localhost/sub2api?PASSWORD=secret",
	} {
		t.Run(dbname, func(t *testing.T) {
			command := exec.Command("bash", migration195PreflightScriptPath(t), "--dbname", dbname)
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			command.Stdout = &stdout
			command.Stderr = &stderr

			err := command.Run()
			var exitError *exec.ExitError
			require.ErrorAs(t, err, &exitError)
			require.Equal(t, 2, exitError.ExitCode())
			require.Empty(t, stdout.String())
			require.NotContains(t, stderr.String(), "secret")
			require.Contains(t, stderr.String(), "must not contain an embedded password")
		})
	}
}

func TestMigration195PreflightScriptEmitsOneOnlineJSONDocument(t *testing.T) {
	payload := runMigration195PreflightScriptWithStub(t, "online")
	require.Equal(t, "migration-195-preflight/v1", payload["contract"])
	require.Equal(t, false, payload["safe_to_apply"])
	require.Equal(t, false, payload["writer_watermark_observation_stable"])
	require.Contains(t, payload["warnings"], "writer_drain_not_verified")
}

func TestMigration195PreflightScriptRequiresStableMaintenanceObservation(t *testing.T) {
	payload := runMigration195PreflightScriptWithStub(t, "maintenance")
	require.Equal(t, true, payload["safe_to_apply"])
	require.Equal(t, true, payload["writer_watermark_observation_stable"])
	require.Equal(t, true, payload["writer_process_stop_acknowledged"])
}

func TestMigration195PreflightScriptChecksEveryRequiredTable(t *testing.T) {
	content, err := os.ReadFile(migration195PreflightScriptPath(t))
	require.NoError(t, err)

	for _, table := range []string{
		"schema_migrations",
		"user_subscriptions",
		"billing_usage_entries",
		"usage_logs",
		"api_keys",
		"groups",
	} {
		require.Contains(t, string(content), "to_regclass('public."+table+"') IS NOT NULL")
	}
}

func runMigration195PreflightScriptWithStub(t *testing.T, mode string) map[string]any {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("preflight script requires bash")
	}

	binDirectory := t.TempDir()
	psqlStub := `#!/bin/sh
checksum=
for argument do
  case "$argument" in
    *to_regclass*) printf '1\n'; exit 0 ;;
    *CONCAT_WS*) printf '1|2|3\n'; exit 0 ;;
    --set=expected_migration_checksum=*) checksum="${argument#--set=expected_migration_checksum=}" ;;
  esac
done
[ -n "$checksum" ] || exit 11
# The real psql consumes the piped SQL before returning. Drain stdin so the
# producer cannot receive SIGPIPE under pipefail when this stub exits quickly.
cat >/dev/null
printf '{"contract":"migration-195-preflight/v1","migration":"195_subscription_anchored_monthly_quota.sql","migration_sha256":"%s","status":"pass","safe_to_apply":false,"already_applied":false,"schema_ok":true,"read_only_verified":true,"counts":{"database_prepared_transactions":"0","database_writer_visibility":"1"},"blockers":[],"warnings":[]}\n' "$checksum"
`
	require.NoError(t, os.WriteFile(filepath.Join(binDirectory, "psql"), []byte(psqlStub), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(binDirectory, "sleep"), []byte("#!/bin/sh\nexit 0\n"), 0o700))

	arguments := []string{
		migration195PreflightScriptPath(t),
		"--mode", mode,
		"--dbname", "service=migration195-test",
	}
	if mode == "maintenance" {
		arguments = append(arguments,
			"--legacy-writers-stopped-at", "2026-07-31T12:00:00Z",
			"--quiescence-seconds", "5",
			"--confirm-writers-stopped",
		)
	}

	command := exec.Command("bash", arguments...)
	// Keep exactly one PATH entry. Duplicate environment keys are not
	// normalized by exec.Cmd and can make the real psql/sleep win over the stubs.
	for _, variable := range os.Environ() {
		if !strings.HasPrefix(variable, "PATH=") {
			command.Env = append(command.Env, variable)
		}
	}
	command.Env = append(
		command.Env,
		"PATH="+binDirectory+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	require.NoError(t, command.Run(), stderr.String())

	trimmed := strings.TrimSpace(stdout.String())
	require.NotEmpty(t, trimmed)
	decoder := json.NewDecoder(strings.NewReader(trimmed))
	var payload map[string]any
	require.NoError(t, decoder.Decode(&payload))
	var extra any
	require.Error(t, decoder.Decode(&extra), "stdout must contain exactly one JSON document")
	return payload
}

func migration195PreflightScriptPath(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join(
		"..",
		"..",
		"scripts",
		"preflight-subscription-anchored-monthly-quota.sh",
	))
	require.NoError(t, err)
	return path
}
