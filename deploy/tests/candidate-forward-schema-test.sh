#!/usr/bin/env bash
set -Eeuo pipefail
umask 077

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

: "${CANDIDATE_IMAGE:?CANDIDATE_IMAGE must be an immutable image reference}"
: "${CANDIDATE_COMMIT:?CANDIDATE_COMMIT is required}"
: "${CANDIDATE_VERSION:?CANDIDATE_VERSION is required}"

old_commit="${OLD_APPLICATION_COMMIT:-d7d138bb776a3608aea0c04634c01ff32e002f36}"
expected_old_migrations="${EXPECTED_OLD_MIGRATIONS:-246}"
expected_candidate_migrations="${EXPECTED_CANDIDATE_MIGRATIONS:-256}"
temp_root="${TMPDIR:-/tmp}"
temp_root="${temp_root%/}"
temp_dir=""
project_name="candidate-forward-$RANDOM-$$"
old_image="local/luoxueapi-old:sha-${old_commit}"

die() {
  printf 'candidate forward-schema test failed: %s\n' "$*" >&2
  exit 1
}

[[ "$CANDIDATE_IMAGE" =~ ^[^[:space:]@]+@sha256:[0-9a-f]{64}$ ]] || die "candidate image must use an immutable digest"
[[ "$CANDIDATE_COMMIT" =~ ^[0-9a-f]{40}$ ]] || die "candidate commit must be a full lowercase SHA"
[[ "$old_commit" =~ ^[0-9a-f]{40}$ ]] || die "old application commit must be a full lowercase SHA"
[[ "$expected_old_migrations" =~ ^[0-9]+$ ]] || die "old migration count must be numeric"
[[ "$expected_candidate_migrations" =~ ^[0-9]+$ ]] || die "candidate migration count must be numeric"

for command_name in git tar docker jq; do
  command -v "$command_name" >/dev/null 2>&1 || die "required command not found: $command_name"
done

temp_dir="$(mktemp -d "$temp_root/candidate-forward.XXXXXX")"
identity_file="$temp_dir/database-identity.json"
identity_stdout_file="$temp_dir/database-identity.stdout"
archive="$temp_dir/old-source.tar"
old_source="$temp_dir/old-source"
touch "$identity_file"
touch "$identity_stdout_file"

export CANDIDATE_IDENTITY_FILE="$identity_file"
export SMOKE_APP_CONTAINER_NAME="${project_name}-app"
export SMOKE_POSTGRES_CONTAINER_NAME="${project_name}-postgres"
export SMOKE_REDIS_CONTAINER_NAME="${project_name}-redis"
export POSTGRES_PASSWORD="candidate-postgres-password"
export REDIS_PASSWORD="candidate-redis-password"
export JWT_SECRET="0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
export TOTP_ENCRYPTION_KEY="abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
export ADMIN_EMAIL="candidate-admin@example.invalid"
export ADMIN_PASSWORD="candidate-admin-password-123"
export SMOKE_IMAGE="$old_image"

compose=(
  docker compose
  --project-name "$project_name"
  --env-file "$repo_dir/deploy/.env.example"
  -f "$repo_dir/deploy/docker-compose.yml"
  -f "$repo_dir/deploy/docker-compose.smoke.yml"
  -f "$repo_dir/deploy/docker-compose.release-smoke.yml"
  -f "$repo_dir/deploy/docker-compose.candidate-ci.yml"
)

cleanup() {
  local exit_code=$?
  set +e
  if ((exit_code != 0)); then
    "${compose[@]}" ps --all
    "${compose[@]}" logs --no-color --timestamps sub2api
  fi
  if [[ "$project_name" =~ ^candidate-forward-[0-9]+-[0-9]+$ ]]; then
    "${compose[@]}" down --volumes --remove-orphans >/dev/null 2>&1
  else
    printf 'refusing Compose cleanup for unexpected project: %s\n' "$project_name" >&2
    exit_code=1
  fi
  docker image rm "$old_image" >/dev/null 2>&1 || true
  if [[ -n "$temp_dir" && -d "$temp_dir" && ! -L "$temp_dir" ]]; then
    case "$temp_dir" in
      "$temp_root"/candidate-forward.*)
        rm -rf -- "$temp_dir"
        ;;
      *)
        printf 'refusing cleanup for unexpected temp path: %s\n' "$temp_dir" >&2
        exit_code=1
        ;;
    esac
  fi
  exit "$exit_code"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
trap 'exit 129' HUP

git -C "$repo_dir" cat-file -e "${old_commit}^{commit}"
git -C "$repo_dir" archive --format=tar --prefix=old-source/ "$old_commit" >"$archive"
[[ -s "$archive" ]] || die "old source archive is empty"
if tar -tf "$archive" | awk '
  /^\// { bad=1 }
  /(^|\/)\._[^/]*$/ { bad=1 }
  /(^|\/)\.\.[/]/ { bad=1 }
  /(^|\/)\.codex-deploy-tmp([/]|$)/ { bad=1 }
  /(^|\/)backend\/cmd\/skill-market-quickprep([/]|$)/ { bad=1 }
  END { exit bad ? 0 : 1 }
'; then
  die "old source archive contains an unsafe or forbidden entry"
fi
# Reproduce normal Git checkout readability for the historical source tree.
# The surrounding runner keeps umask 077 for identities and other evidence,
# but a 077 extraction turns tracked 100644 files into 0600 files. The old
# Dockerfile uses `chmod +x` for its entrypoint, which would then create 0711
# instead of the release-equivalent 0755 and fail before schema validation.
(
  umask 022
  tar -xf "$archive" -C "$temp_dir"
)

docker buildx build \
  --platform linux/amd64 \
  --file "$old_source/Dockerfile" \
  --tag "$old_image" \
  --build-arg "VERSION=$CANDIDATE_VERSION" \
  --build-arg "COMMIT=$old_commit" \
  --label "org.opencontainers.image.revision=$old_commit" \
  --load \
  "$old_source"

old_revision="$(docker image inspect --format '{{ index .Config.Labels "org.opencontainers.image.revision" }}' "$old_image")"
[[ "$old_revision" == "$old_commit" ]] || die "old image revision label mismatch"
old_version_output="$(docker run --rm --platform linux/amd64 --entrypoint /app/sub2api \
  "$old_image" --version 2>&1)"
grep -Fq -- "commit: ${old_commit}," <<<"$old_version_output" || die "old binary revision mismatch"
old_manifest_count="$(docker run --rm --platform linux/amd64 --entrypoint /app/sub2api \
  "$old_image" --migration-manifest | jq -er '.migrations | length')"
[[ "$old_manifest_count" == "$expected_old_migrations" ]] || die "old image migration count mismatch"

"${compose[@]}" up --no-build --detach --wait --wait-timeout 180

db_scalar() {
  local query="$1"
  # Variables expand in the container shell, not in this process.
  # shellcheck disable=SC2016
  "${compose[@]}" exec -T postgres sh -ec \
    'PGPASSWORD="$POSTGRES_PASSWORD" exec psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Atc "$1"' \
    candidate-db-query "$query"
}

wait_ready() {
  local ready_json
  ready_json="$("${compose[@]}" exec -T sub2api wget -q -T 10 -O - http://127.0.0.1:8080/readyz)"
  jq -e '
    .status == "ready"
    and .checks.draining.status == "ready"
    and .checks.components.status == "ready"
    and .checks.database.status == "ready"
    and .checks.redis.status == "ready"
    and .checks.scheduler.status == "ready"
  ' <<<"$ready_json" >/dev/null
}

assert_projects_schema() {
  [[ "$(db_scalar "SELECT to_regclass('public.chat_projects') IS NOT NULL")" == "t" ]] ||
    die "chat_projects table is missing"
  [[ "$(db_scalar "SELECT to_regclass('public.chat_project_files') IS NOT NULL")" == "t" ]] ||
    die "chat_project_files table is missing"
  [[ "$(db_scalar "SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'chat_conversations' AND column_name = 'project_id')")" == "t" ]] ||
    die "chat_conversations.project_id column is missing"

  project_column_count="$(db_scalar "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'chat_projects' AND column_name IN ('id','public_id','user_id','name','icon','color','instructions','memory_mode','created_at','updated_at','deleted_at')")"
  [[ "$project_column_count" == "11" ]] || die "chat_projects schema is incomplete"
  project_file_column_count="$(db_scalar "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'chat_project_files' AND column_name IN ('project_id','library_file_id','user_id','created_at')")"
  [[ "$project_file_column_count" == "4" ]] || die "chat_project_files schema is incomplete"

  for index_name in \
    idx_chat_projects_user_updated \
    idx_chat_conversations_user_project_updated \
    idx_chat_project_files_user; do
    [[ "$(db_scalar "SELECT to_regclass('public.' || '$index_name') IS NOT NULL")" == "t" ]] ||
      die "project migration index is missing: $index_name"
  done

  project_constraint_count="$(db_scalar "SELECT COUNT(*) FROM pg_constraint AS c JOIN pg_class AS tbl ON tbl.oid = c.conrelid JOIN pg_namespace AS ns ON ns.oid = tbl.relnamespace WHERE ns.nspname = 'public' AND c.conname IN ('chat_projects_name_check','chat_projects_memory_mode_check','chat_conversations_project_fkey','chat_project_files_user_project_fkey','chat_project_files_user_library_fkey')")"
  [[ "$project_constraint_count" == "5" ]] || die "project migration constraints are incomplete"
}

wait_ready
[[ "$(db_scalar 'SELECT COUNT(*) FROM schema_migrations')" == "$expected_old_migrations" ]] || die "old baseline is not $expected_old_migrations migrations"
"${compose[@]}" stop -t 30 sub2api

export SMOKE_IMAGE="$CANDIDATE_IMAGE"
"${compose[@]}" run --rm --no-deps --entrypoint /app/sub2api sub2api \
  --database-identity >"$identity_stdout_file"
# Docker Compose may emit one-off container lifecycle messages to stdout.
# The binary's identity contract is one compact JSON object; require exactly
# one such line so progress text cannot corrupt or ambiguously select evidence.
if ! awk -f "$repo_dir/deploy/tests/extract-candidate-database-identity.awk" \
  "$identity_stdout_file" >"$identity_file"; then
  die "candidate database identity stdout is missing or ambiguous"
fi
jq -e '
  type == "object"
  and (keys == [
    "contract",
    "database",
    "in_recovery",
    "system_identifier"
  ])
  and .contract == "sub2api-database-identity/v2"
  and ((.database | type) == "string")
  and ((.database | length) > 0)
  and ((.system_identifier | type) == "string")
  and (.system_identifier | test("^[0-9]+$"))
  and .in_recovery == false
' "$identity_file" >/dev/null

for replay in first second; do
  "${compose[@]}" run --rm --no-deps --entrypoint /app/sub2api sub2api \
    --migrate-only \
    --expected-database-identity-file /run/candidate/database-identity.json
  [[ "$(db_scalar 'SELECT COUNT(*) FROM schema_migrations')" == "$expected_candidate_migrations" ]] || die "candidate migrate-only $replay did not reach $expected_candidate_migrations"
done

candidate_tail="$(db_scalar "SELECT string_agg(filename || '=' || checksum, ',' ORDER BY filename) FROM schema_migrations WHERE filename IN ('201_library_files.sql','201a_library_alias_unique_index_notx.sql','201b_library_alias_constraints.sql','202_chat_message_activities.sql','231_add_users_email_alias_dedup_index_notx.sql','232_add_users_email_normalized_index_notx.sql','233_group_profit_control.sql','234_add_usage_log_upstream_response_model.sql','235_add_usage_log_upstream_model_mismatch_index_notx.sql','236_projects.sql')")"
expected_tail="201_library_files.sql=03f6a53d92e93fbfee37b9e5dd78dc11cae49dd921812253d0425154f1a9c23e,201a_library_alias_unique_index_notx.sql=ba15a71ce63180c21f8addda85351b13171a0e6c22f7427bcb4b8c955499e564,201b_library_alias_constraints.sql=f52ac96a80583b4e7a3c7c5f9923eee5d95a47c4a2b2d9844c864f31abe83833,202_chat_message_activities.sql=e2ee8b4480af916327f132d378eb70b2291c85efba0ced4555452147b56fdb8f,231_add_users_email_alias_dedup_index_notx.sql=fd103466b72b14919fc7a0b02135f019f9fe7a409a434726467c3649551321e4,232_add_users_email_normalized_index_notx.sql=052a61bf4bdc89a5215970059a61096f4eaea5c244b6781f3ec42d6ac8e8bb5d,233_group_profit_control.sql=b39b90d72d8869dc46beeb426f5db112ff04235c89ddb6d0ecee61a9bea95381,234_add_usage_log_upstream_response_model.sql=cad520cbfcf7af7ea9acae92e5bcbe27501fd9e3ad5b02e306f4f97be4410a82,235_add_usage_log_upstream_model_mismatch_index_notx.sql=692f2a75f0c62670b4d68986912bf24eb92f6377ec904d3806ff7d62b0da8355,236_projects.sql=050ad388c07995c4167ebd5ef52211f5cc2f04dfb74d6ab6655403d03f9936ce"
[[ "$candidate_tail" == "$expected_tail" ]] || die "candidate migration tail differs"
assert_projects_schema

"${compose[@]}" up --no-build --detach --wait --wait-timeout 180 sub2api
wait_ready
candidate_container="$("${compose[@]}" ps -q sub2api)"
[[ -n "$candidate_container" ]] || die "candidate application container is missing"
candidate_running_image="$(docker inspect --format '{{.Image}}' "$candidate_container")"
expected_candidate_image_id="$(docker image inspect --format '{{.Id}}' "$CANDIDATE_IMAGE")"
[[ "$candidate_running_image" == "$expected_candidate_image_id" ]] || die "candidate container image ID mismatch"
[[ "$(docker inspect --format '{{.RestartCount}}' "$candidate_container")" == "0" ]] || die "candidate application restarted"
"${compose[@]}" exec -T sub2api wget -q -T 10 -O - \
  http://127.0.0.1:8080/health >/dev/null
"${compose[@]}" exec -T sub2api wget -q -T 10 -O - \
  http://127.0.0.1:8080/readyz | jq -e '.status == "ready"' >/dev/null
"${compose[@]}" stop -t 30 sub2api

export SMOKE_IMAGE="$old_image"
"${compose[@]}" up --no-build --detach --wait --wait-timeout 180 sub2api
wait_ready
[[ "$(db_scalar 'SELECT COUNT(*) FROM schema_migrations')" == "$expected_candidate_migrations" ]] || die "old application changed the forward schema"
old_container="$("${compose[@]}" ps -q sub2api)"
[[ -n "$old_container" ]] || die "forward-schema old application container is missing"
[[ "$(docker inspect --format '{{.RestartCount}}' "$old_container")" == "0" ]] || die "forward-schema old application restarted"

printf 'candidate forward-schema contract passed: old=%s candidate=%s migrations=%s\n' \
  "$old_commit" "$CANDIDATE_COMMIT" "$expected_candidate_migrations"
