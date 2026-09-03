#!/usr/bin/env bash
set -Eeuo pipefail

backend_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
repo_dir="$(cd "$backend_dir/.." && pwd)"
project_name="luoxueapi-smoke-$$"

export SMOKE_APP_CONTAINER_NAME="${project_name}-app"
export SMOKE_POSTGRES_CONTAINER_NAME="${project_name}-postgres"
export SMOKE_REDIS_CONTAINER_NAME="${project_name}-redis"
export POSTGRES_PASSWORD="smoke-postgres-password"
export REDIS_PASSWORD="smoke-redis-password"
export JWT_SECRET="0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
export TOTP_ENCRYPTION_KEY="abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
export ADMIN_EMAIL="smoke-admin@example.com"
export ADMIN_PASSWORD="smoke-admin-password-123"

compose=(
  docker compose
  --project-name "$project_name"
  --env-file "$repo_dir/deploy/.env.example"
  -f "$repo_dir/deploy/docker-compose.yml"
  -f "$repo_dir/deploy/docker-compose.smoke.yml"
)
up_mode=(--build)
if [[ -n "${SMOKE_IMAGE:-}" ]]; then
  compose+=(
    -f "$repo_dir/deploy/docker-compose.release-smoke.yml"
  )
  up_mode=(--no-build)
fi

cleanup() {
  local exit_code=$?
  set +e
  if ((exit_code != 0)); then
    "${compose[@]}" ps --all
    "${compose[@]}" logs --no-color --timestamps sub2api
  fi
  "${compose[@]}" down --volumes --remove-orphans >/dev/null 2>&1 || true
  exit "$exit_code"
}
trap cleanup EXIT

"${compose[@]}" up "${up_mode[@]}" --detach --wait --wait-timeout 180

base_url="http://127.0.0.1:8080"
ready_json="$("${compose[@]}" exec -T sub2api \
  wget -q -T 5 -O - "$base_url/readyz")"
jq -e '
  .status == "ready"
  and .checks.draining.status == "ready"
  and .checks.components.status == "ready"
  and .checks.database.status == "ready"
  and .checks.redis.status == "ready"
  and .checks.scheduler.status == "ready"
' <<< "$ready_json" >/dev/null

login_response="$("${compose[@]}" exec -T sub2api \
  wget -q -T 10 -O - \
  --header='Content-Type: application/json' \
  --post-data='{"email":"smoke-admin@example.com","password":"smoke-admin-password-123"}' \
  "$base_url/api/v1/auth/login")"
access_token="$(jq -er '.data.access_token' <<< "$login_response")"

homepage="$("${compose[@]}" exec -T sub2api \
  wget -q -T 10 -O - "$base_url/")"
grep -qi '<!doctype html' <<< "$homepage"

compliance_status="$("${compose[@]}" exec -T sub2api \
  wget -q -T 10 -O - \
  --header="Authorization: Bearer $access_token" \
  "$base_url/api/v1/admin/compliance")"
compliance_phrase="$(jq -er '
  select(.code == 0 and .data.required == true)
  | .data.ack_phrase_en
  | select(type == "string" and length > 0)
' <<< "$compliance_status")"
compliance_payload="$(jq -cn --arg phrase "$compliance_phrase" \
  '{phrase: $phrase, language: "en"}')"

"${compose[@]}" exec -T sub2api \
  wget -q -T 10 -O - \
  --header='Content-Type: application/json' \
  --header="Authorization: Bearer $access_token" \
  --post-data="$compliance_payload" \
  "$base_url/api/v1/admin/compliance/accept" \
  | jq -e '.code == 0 and .data.required == false' >/dev/null

"${compose[@]}" exec -T sub2api \
  wget -q -T 10 -O - \
  --header="Authorization: Bearer $access_token" \
  "$base_url/api/v1/admin/channels/model-pricing?model=claude-3-haiku-20240307" \
  | jq -e '.code == 0 and .data.input_price != null' >/dev/null

echo "Compose smoke passed: readiness, login, embedded homepage, offline pricing fallback"
