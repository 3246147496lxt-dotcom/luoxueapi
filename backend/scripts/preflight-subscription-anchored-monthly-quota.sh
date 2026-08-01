#!/usr/bin/env bash

set -Eeuo pipefail

umask 077

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/../.." && pwd)"
sql_file="${script_dir}/sql/preflight_subscription_anchored_monthly_quota.sql"
migration_file="${repo_root}/backend/migrations/195_subscription_anchored_monthly_quota.sql"

preflight_mode="online"
preflight_dbname=""
legacy_writers_stopped_at=""
term_guard_hours=2
quiescence_seconds=15
acknowledge_ambiguous_term_events=false
confirm_writers_stopped=false

usage() {
  cat <<'USAGE'
Usage:
  preflight-subscription-anchored-monthly-quota.sh [options]

Options:
  --mode online|maintenance
      online: inventory only; never reports safe_to_apply=true (default)
      maintenance: requires a stable write-watermark observation plus an
      external acknowledgement that every writer process is stopped
  --dbname NAME
      libpq database/service/URI without an embedded password. Prefer PGHOST,
      PGPORT, PGUSER, PGDATABASE, PGSSLMODE and PGPASSFILE.
  --legacy-writers-stopped-at RFC3339
      Exact old-writer cutoff. In maintenance mode the database time captured
      before the quiescence wait is used when omitted.
  --term-guard-hours N
      Conservative reopened-term review window, 1..24 hours (default: 2).
  --quiescence-seconds N
      Maintenance write-watermark observation interval, 5..300 (default: 15).
      This is drift detection, not a database writer fence.
  --acknowledge-ambiguous-reopened-term-events
      Demote only the aggregate ambiguous reopened-term check after a separate,
      authorized reconciliation. Definite old-term evidence remains blocking.
  --confirm-writers-stopped
      Required in maintenance mode. Confirms every application, worker, webhook,
      admin job and external script capable of writing quota tables is stopped.
  -h, --help

Output:
  stdout is exactly one compact JSON document. Diagnostics go to stderr.

Exit codes:
  0  inventory completed or maintenance gate passed/already applied
  2  invalid CLI arguments
  3  missing dependency or unsafe local configuration
  4  database connection/authentication/TLS failure
  5  schema or migration checksum mismatch
  10 data or writer-drain release gate blocked
  11 query failed or timed out
  70 internal JSON/contract failure
  130 interrupted by SIGINT
  143 terminated by SIGTERM
USAGE
}

log() {
  printf '[migration-195-preflight] %s\n' "$*" >&2
}

die_usage() {
  log "$1"
  usage >&2
  exit 2
}

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    log "required command is unavailable: $1"
    exit 3
  fi
}

on_int() {
  log "interrupted"
  exit 130
}

on_term() {
  log "terminated"
  exit 143
}

trap on_int INT
trap on_term TERM

while (($# > 0)); do
  case "$1" in
    --mode)
      (($# >= 2)) || die_usage "--mode requires a value"
      preflight_mode="$2"
      shift 2
      ;;
    --dbname)
      (($# >= 2)) || die_usage "--dbname requires a value"
      preflight_dbname="$2"
      shift 2
      ;;
    --legacy-writers-stopped-at)
      (($# >= 2)) || die_usage "--legacy-writers-stopped-at requires a value"
      legacy_writers_stopped_at="$2"
      shift 2
      ;;
    --term-guard-hours)
      (($# >= 2)) || die_usage "--term-guard-hours requires a value"
      term_guard_hours="$2"
      shift 2
      ;;
    --quiescence-seconds)
      (($# >= 2)) || die_usage "--quiescence-seconds requires a value"
      quiescence_seconds="$2"
      shift 2
      ;;
    --acknowledge-ambiguous-reopened-term-events)
      acknowledge_ambiguous_term_events=true
      shift
      ;;
    --confirm-writers-stopped)
      confirm_writers_stopped=true
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      die_usage "unknown argument: $1"
      ;;
  esac
done

[[ "$preflight_mode" == "online" || "$preflight_mode" == "maintenance" ]] \
  || die_usage "--mode must be online or maintenance"
[[ "$term_guard_hours" =~ ^[0-9]+$ ]] \
  || die_usage "--term-guard-hours must be an integer"
((term_guard_hours >= 1 && term_guard_hours <= 24)) \
  || die_usage "--term-guard-hours must be between 1 and 24"
[[ "$quiescence_seconds" =~ ^[0-9]+$ ]] \
  || die_usage "--quiescence-seconds must be an integer"
((quiescence_seconds >= 5 && quiescence_seconds <= 300)) \
  || die_usage "--quiescence-seconds must be between 5 and 300"
if [[ "$preflight_mode" == "maintenance" && "$confirm_writers_stopped" != "true" ]]; then
  die_usage "maintenance mode requires --confirm-writers-stopped"
fi

if [[ -n "$preflight_dbname" ]]; then
  if [[ "$preflight_dbname" =~ [Pp][Aa][Ss][Ss][Ww][Oo][Rr][Dd][[:space:]]*= ]] \
    || [[ "$preflight_dbname" =~ [\?\&][Pp][Aa][Ss][Ss][Ww][Oo][Rr][Dd]= ]] \
    || [[ "$preflight_dbname" =~ ://[^/]*@ ]]; then
    die_usage "--dbname must not contain an embedded password; use PGPASSFILE"
  fi
fi

require_command psql
require_command jq
require_command sleep

[[ -r "$sql_file" ]] || {
  log "preflight SQL is not readable"
  exit 3
}
[[ -r "$migration_file" ]] || {
  log "migration 195 is not readable"
  exit 3
}

sha256_stream() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum | awk '{print $1}'
    return
  fi
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 | awk '{print $1}'
    return
  fi
  if command -v openssl >/dev/null 2>&1; then
    openssl dgst -sha256 | awk '{print $NF}'
    return
  fi
  log "one of sha256sum, shasum, or openssl is required"
  exit 3
}

# Bash command substitution removes the trailing newline. Migration files begin
# with non-whitespace and end with SQL, so this matches Go strings.TrimSpace
# before SHA-256 without writing source or credentials to a temporary file.
migration_content="$(<"$migration_file")"
expected_migration_checksum="$(printf '%s' "$migration_content" | sha256_stream)"
[[ "$expected_migration_checksum" =~ ^[0-9a-f]{64}$ ]] || {
  log "failed to compute the migration checksum"
  exit 3
}
unset migration_content

psql_args=(
  psql
  -X
  --no-password
  --quiet
  --tuples-only
  --no-align
  --set=ON_ERROR_STOP=1
  --set=VERBOSITY=terse
)
if [[ -n "$preflight_dbname" ]]; then
  psql_args+=(--dbname="$preflight_dbname")
fi

preflight_pgoptions="${PGOPTIONS:-} -c default_transaction_read_only=on"
preflight_connect_timeout="${PGCONNECT_TIMEOUT:-10}"

run_psql() {
  PGOPTIONS="$preflight_pgoptions" \
    PGCONNECT_TIMEOUT="$preflight_connect_timeout" \
    "${psql_args[@]}" "$@"
}

map_psql_failure() {
  local status="$1"
  if ((status == 2)); then
    log "database connection, authentication, or TLS negotiation failed"
    exit 4
  fi
  log "read-only preflight query failed"
  exit 11
}

run_scalar() {
  local query="$1"
  local output
  local status
  set +e
  output="$(run_psql --command="$query")"
  status=$?
  set -e
  ((status == 0)) || map_psql_failure "$status"
  printf '%s' "$output"
}

schema_ready="$(run_scalar "
SELECT CASE WHEN
    to_regclass('public.schema_migrations') IS NOT NULL
    AND to_regclass('public.user_subscriptions') IS NOT NULL
    AND to_regclass('public.billing_usage_entries') IS NOT NULL
    AND to_regclass('public.usage_logs') IS NOT NULL
    AND to_regclass('public.groups') IS NOT NULL
THEN '1' ELSE '0' END;
")"
if [[ "$schema_ready" != "1" ]]; then
  jq -nc --arg migration_sha256 "$expected_migration_checksum" '{
    contract: "migration-195-preflight/v1",
    migration: "195_subscription_anchored_monthly_quota.sql",
    migration_sha256: $migration_sha256,
    status: "schema_blocked",
    safe_to_apply: false,
    already_applied: false,
    schema_ok: false,
    read_only_verified: true,
    counts: {},
    blockers: ["missing_schema_prerequisites"],
    warnings: []
  }'
  exit 5
fi

capture_watermark() {
  run_scalar "
BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY;
SET LOCAL statement_timeout = '120s';
SET LOCAL lock_timeout = '5s';
SET LOCAL idle_in_transaction_session_timeout = '130s';
SET LOCAL search_path = pg_catalog, public;
SELECT CONCAT_WS('|',
    COALESCE((SELECT MAX(id)::text FROM public.billing_usage_entries), '0'),
    COALESCE((SELECT MAX(id)::text FROM public.usage_logs), '0'),
    COALESCE((
        SELECT FLOOR(EXTRACT(EPOCH FROM MAX(updated_at)) * 1000000)::bigint::text
        FROM public.user_subscriptions
    ), '0')
);
COMMIT;
"
}

writer_watermark_observation_stable=false
if [[ "$preflight_mode" == "maintenance" ]]; then
  if [[ -z "$legacy_writers_stopped_at" ]]; then
    legacy_writers_stopped_at="$(run_scalar "
SELECT TO_CHAR(
    clock_timestamp() AT TIME ZONE 'UTC',
    'YYYY-MM-DD\"T\"HH24:MI:SS.US\"Z\"'
);
")"
    log "captured the database old-writer cutoff before quiescence observation"
  fi

  watermark_before="$(capture_watermark)"
  log "observing write watermarks for ${quiescence_seconds}s"
  sleep "$quiescence_seconds"
  watermark_after="$(capture_watermark)"
  if [[ "$watermark_before" == "$watermark_after" ]]; then
    writer_watermark_observation_stable=true
  fi
  unset watermark_before watermark_after
fi

set +e
base_payload="$({
  printf '%s\n' \
    'BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY;' \
    "SET LOCAL statement_timeout = '120s';" \
    "SET LOCAL lock_timeout = '5s';" \
    "SET LOCAL idle_in_transaction_session_timeout = '130s';" \
    'SET LOCAL search_path = pg_catalog, public;'
  cat "$sql_file"
  printf '%s\n' 'COMMIT;'
} | PGOPTIONS="$preflight_pgoptions" \
  PGCONNECT_TIMEOUT="$preflight_connect_timeout" \
  "${psql_args[@]}" \
  --set=preflight_mode="$preflight_mode" \
  --set=legacy_writers_stopped_at="$legacy_writers_stopped_at" \
  --set=term_guard_hours="$term_guard_hours" \
  --set=acknowledge_ambiguous_term_events="$acknowledge_ambiguous_term_events" \
  --set=expected_migration_checksum="$expected_migration_checksum")"
psql_status=$?
set -e
((psql_status == 0)) || map_psql_failure "$psql_status"

if ! jq -e '
  .contract == "migration-195-preflight/v1"
  and (.migration_sha256 | test("^[0-9a-f]{64}$"))
  and (.schema_ok | type == "boolean")
  and (.read_only_verified == true)
  and (.blockers | type == "array")
  and (.warnings | type == "array")
' >/dev/null 2>&1 <<<"$base_payload"; then
  log "preflight output contract validation failed"
  exit 70
fi

final_payload="$(jq -c \
  --arg mode "$preflight_mode" \
  --argjson writers_stopped "$confirm_writers_stopped" \
  --argjson watermark_stable "$writer_watermark_observation_stable" '
  .writer_watermark_observation_stable = $watermark_stable
  | .writer_process_stop_acknowledged = $writers_stopped
  | if $mode == "online" then
      .warnings = ((.warnings + ["writer_drain_not_verified"]) | unique)
      | .safe_to_apply = false
      | if .status != "blocked" then .status = "warn" else . end
    elif $watermark_stable == false then
      .blockers = ((.blockers + ["writer_watermark_changed"]) | unique)
      | .safe_to_apply = false
      | .status = "blocked"
    elif .already_applied == true and .schema_ok == true
      and (.blockers | length) == 0 then
      .safe_to_apply = false
      | .status = "already_applied"
    elif (.blockers | length) == 0 and .schema_ok == true
      and .read_only_verified == true then
      .safe_to_apply = true
      | if (.warnings | length) > 0
        then .status = "pass_with_warnings"
        else .status = "pass"
        end
    else
      .safe_to_apply = false
      | .status = "blocked"
    end
' <<<"$base_payload")"

if ! jq -e '.contract == "migration-195-preflight/v1"' >/dev/null 2>&1 \
  <<<"$final_payload"; then
  log "failed to finalize the JSON preflight report"
  exit 70
fi

printf '%s\n' "$final_payload"

if [[ "$(jq -r '.schema_ok' <<<"$final_payload")" != "true" ]]; then
  exit 5
fi
if [[ "$(jq -r '.status' <<<"$final_payload")" == "blocked" ]]; then
  exit 10
fi
