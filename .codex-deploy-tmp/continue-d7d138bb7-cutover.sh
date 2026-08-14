#!/usr/bin/env bash
set -Eeuo pipefail
umask 077

# Continue a previously verified hotfix deployment after the first script stopped
# before cutover because docker-compose.custom.yml quoted its image value.
BASE_COMMIT="d0a46e04eb8fbf94a90bad184fbb094736e8ca63"
COMMIT="d7d138bb776a3608aea0c04634c01ff32e002f36"
VERSION="2.0.0-rc.5"
EXPECTED_OLD_REF="luoxueapi:d0a46e04e-skillimport-safety-r1"
EXPECTED_OLD_ID="sha256:5574502c04397c5af6b12be28fa82a015f7879bef004c5029f5386efd7e815d9"
CANDIDATE_REF="luoxueapi:d7d138bb7-stage-null-20260814T010147Z"
CANDIDATE_ID="sha256:d2a85a33c1055d4088282a1cb712a676c7c505bd5e7e315627d7f2157a054300"
ROLLBACK_REF="luoxueapi:rollback-d0a46e04e-before-d7d138bb7-20260814T010147Z"

DEPLOY_DIR="/root/luoxueapi/deploy"
LOCAL_COMPOSE="${DEPLOY_DIR}/docker-compose.local.yml"
CUSTOM_COMPOSE="${DEPLOY_DIR}/docker-compose.custom.yml"
RELEASE_PARENT="/root/luoxueapi/releases"
STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
RUN_ID="hotfix-d7d138bb7-cutover-${STAMP}"
RELEASE_DIR="${RELEASE_PARENT}/${RUN_ID}"
LATEST_FILE="${RELEASE_PARENT}/hotfix-d7d138bb7-cutover.latest"
STATUS_FILE="${RELEASE_DIR}/status"
LOG_FILE="${RELEASE_DIR}/deploy.log"
CUSTOM_BEFORE="${RELEASE_DIR}/docker-compose.custom.before.yml"
CUSTOM_WORK="${DEPLOY_DIR}/.docker-compose.custom.${RUN_ID}.tmp"

PHASE="bootstrap"
CUTOVER_STARTED=0
ROLLBACK_DONE=0
ERROR_LINE=""
ERROR_COMMAND=""
OLD_ENV_SHA=""
OLD_MOUNTS_SHA=""
OLD_PORTS_SHA=""
OLD_MODEL_SHA=""
PG_ID=""
PG_STARTED=""
PG_RESTARTS=""
REDIS_ID=""
REDIS_STARTED=""
REDIS_RESTARTS=""

test -d "$DEPLOY_DIR"
exec 9>"${DEPLOY_DIR}/.codex-deploy.lock"
if ! flock -n 9; then
  printf 'another deployment owns %s\n' "${DEPLOY_DIR}/.codex-deploy.lock" >&2
  exit 75
fi
mkdir -p "$RELEASE_PARENT"
test ! -e "$RELEASE_DIR"
mkdir -m 700 "$RELEASE_DIR"
printf '%s\n' "$RELEASE_DIR" > "${LATEST_FILE}.tmp"
mv -f "${LATEST_FILE}.tmp" "$LATEST_FILE"
exec > >(tee -a "$LOG_FILE") 2>&1

write_status() {
  local state="$1" tmp="${STATUS_FILE}.tmp"
  {
    printf 'state=%s\nphase=%s\nrun_id=%s\ncommit=%s\n' "$state" "$PHASE" "$RUN_ID" "$COMMIT"
    printf 'image_ref=%s\nimage_id=%s\nrollback_ref=%s\n' "$CANDIDATE_REF" "$CANDIDATE_ID" "$ROLLBACK_REF"
    printf 'cutover_started=%s\nrollback_done=%s\n' "$CUTOVER_STARTED" "$ROLLBACK_DONE"
    printf 'error_line=%s\nerror_command=%q\n' "$ERROR_LINE" "$ERROR_COMMAND"
    printf 'updated_at=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  } > "$tmp"
  mv -f "$tmp" "$STATUS_FILE"
}

set_phase() {
  PHASE="$1"
  printf '[phase] %s\n' "$PHASE"
  write_status running
}

container_env_sha() {
  docker inspect "$1" --format '{{range .Config.Env}}{{println .}}{{end}}' \
    | LC_ALL=C sort | sha256sum | awk '{print $1}'
}

container_mounts_sha() {
  docker inspect "$1" --format '{{json .Mounts}}' | jq -S 'map({Type,Source,Destination,RW,Propagation})' \
    | sha256sum | awk '{print $1}'
}

container_ports_sha() {
  docker inspect "$1" --format '{{json .HostConfig.PortBindings}}' | jq -S . \
    | sha256sum | awk '{print $1}'
}

compose_model_sha() {
  docker compose --project-directory "$DEPLOY_DIR" --env-file "${DEPLOY_DIR}/.env" \
    -f "$LOCAL_COMPOSE" -f "$1" config --format json \
    | jq -S 'del(.services.sub2api.image)' | sha256sum | awk '{print $1}'
}

assert_dependencies_unchanged() {
  test "$(docker inspect sub2api-postgres --format '{{.Id}}')" = "$PG_ID"
  test "$(docker inspect sub2api-postgres --format '{{.State.StartedAt}}')" = "$PG_STARTED"
  test "$(docker inspect sub2api-postgres --format '{{.RestartCount}}')" = "$PG_RESTARTS"
  test "$(docker inspect sub2api-postgres --format '{{.State.Health.Status}}')" = healthy
  test "$(docker inspect sub2api-redis --format '{{.Id}}')" = "$REDIS_ID"
  test "$(docker inspect sub2api-redis --format '{{.State.StartedAt}}')" = "$REDIS_STARTED"
  test "$(docker inspect sub2api-redis --format '{{.RestartCount}}')" = "$REDIS_RESTARTS"
  test "$(docker inspect sub2api-redis --format '{{.State.Health.Status}}')" = healthy
}

wait_for_app() {
  local expected_id="$1" prefix="$2" attempt state health
  for attempt in $(seq 1 72); do
    if docker inspect sub2api >/dev/null 2>&1; then
      state="$(docker inspect sub2api --format '{{.State.Status}}')"
      health="$(docker inspect sub2api --format '{{if .State.Health}}{{.State.Health.Status}}{{end}}')"
      if test "$state" = running && test "$health" = healthy; then
        if test "$(docker inspect sub2api --format '{{.Image}}')" != "$expected_id"; then return 1; fi
        if test "$(docker inspect sub2api --format '{{.RestartCount}}')" != 0; then return 1; fi
        if ! docker exec sub2api wget -q -T 5 -O - http://127.0.0.1:8080/health > "${RELEASE_DIR}/${prefix}-health.json"; then return 1; fi
        if ! docker exec sub2api wget -q -T 5 -O - http://127.0.0.1:8080/readyz > "${RELEASE_DIR}/${prefix}-readyz.json"; then return 1; fi
        if ! jq -e '.status == "ok"' "${RELEASE_DIR}/${prefix}-health.json" >/dev/null; then return 1; fi
        if ! jq -e '.status == "ready" and .checks.database.status == "ready" and .checks.redis.status == "ready" and .checks.components.status == "ready" and .checks.scheduler.status == "ready"' "${RELEASE_DIR}/${prefix}-readyz.json" >/dev/null; then return 1; fi
        return 0
      fi
      if test "$state" = exited || test "$state" = dead; then
        docker logs --tail 200 sub2api > "${RELEASE_DIR}/${prefix}-failed.log" 2>&1 || true
        return 1
      fi
    fi
    sleep 5
  done
  docker logs --tail 200 sub2api > "${RELEASE_DIR}/${prefix}-timeout.log" 2>&1 || true
  return 1
}

restore_custom_atomically() {
  local tmp="${DEPLOY_DIR}/.docker-compose.custom.rollback.${RUN_ID}.tmp"
  cp -a "$CUSTOM_BEFORE" "$tmp"
  mv -f "$tmp" "$CUSTOM_COMPOSE"
}

rollback_app() {
  local failed=0
  set +e
  PHASE="rollback"
  write_status rolling_back
  printf '[rollback] restoring %s\n' "$EXPECTED_OLD_ID" >&2
  docker tag "$ROLLBACK_REF" "$EXPECTED_OLD_REF" || failed=1
  restore_custom_atomically || failed=1
  docker compose --project-directory "$DEPLOY_DIR" --env-file "${DEPLOY_DIR}/.env" \
    -f "$LOCAL_COMPOSE" -f "$CUSTOM_COMPOSE" config --quiet || failed=1
  docker compose --project-directory "$DEPLOY_DIR" --env-file "${DEPLOY_DIR}/.env" \
    -f "$LOCAL_COMPOSE" -f "$CUSTOM_COMPOSE" up -d --pull never --no-deps --force-recreate sub2api || failed=1
  wait_for_app "$EXPECTED_OLD_ID" rollback || failed=1
  assert_dependencies_unchanged || failed=1
  test "$(container_env_sha sub2api)" = "$OLD_ENV_SHA" || failed=1
  test "$(container_mounts_sha sub2api)" = "$OLD_MOUNTS_SHA" || failed=1
  test "$(container_ports_sha sub2api)" = "$OLD_PORTS_SHA" || failed=1
  if test "$failed" = 0; then
    ROLLBACK_DONE=1
    write_status rolled_back
    printf '[rollback] complete\n' >&2
  fi
  set -e
  test "$failed" = 0
}

on_error() {
  local rc="$1" line="$2" command="$3"
  ERROR_LINE="$line"
  ERROR_COMMAND="$command"
  printf '[error] rc=%s phase=%s line=%s command=%q\n' "$rc" "$PHASE" "$line" "$command" >&2
  write_status failed
}

on_exit() {
  local rc="$?" final_rc="$?"
  final_rc="$rc"
  trap - EXIT ERR HUP INT TERM
  if test "$rc" -ne 0 && test "$CUTOVER_STARTED" = 1 && test "$ROLLBACK_DONE" = 0; then
    if ! rollback_app; then
      PHASE="rollback_failed"
      write_status rollback_failed
      final_rc=70
    fi
  fi
  if test "$final_rc" -eq 0; then
    PHASE="complete"
    write_status success
    printf '[success] deployed %s as %s\n' "$COMMIT" "$CANDIDATE_ID"
  elif test "$ROLLBACK_DONE" = 1; then
    PHASE="rolled_back"
    write_status rolled_back
    printf '[failed-safe] deployment failed and old image is healthy\n' >&2
  else
    write_status failed
    printf '[failed] stopped in %s (rc=%s)\n' "$PHASE" "$final_rc" >&2
  fi
  exit "$final_rc"
}

trap 'on_error "$?" "${BASH_LINENO[0]}" "$BASH_COMMAND"' ERR
trap on_exit EXIT
trap 'ERROR_COMMAND="signal HUP"; exit 129' HUP
trap 'ERROR_COMMAND="signal INT"; exit 130' INT
trap 'ERROR_COMMAND="signal TERM"; exit 143' TERM
write_status running

set_phase preflight
for cmd in awk cmp curl docker flock grep jq mv seq sha256sum sort tee; do
  command -v "$cmd" >/dev/null
done
test -f "$LOCAL_COMPOSE" && test ! -L "$LOCAL_COMPOSE"
test -f "$CUSTOM_COMPOSE" && test ! -L "$CUSTOM_COMPOSE"
docker info >/dev/null
docker compose --project-directory "$DEPLOY_DIR" --env-file "${DEPLOY_DIR}/.env" \
  -f "$LOCAL_COMPOSE" -f "$CUSTOM_COMPOSE" config --quiet
test "$(docker inspect sub2api --format '{{.Config.Image}}')" = "$EXPECTED_OLD_REF"
test "$(docker inspect sub2api --format '{{.Image}}')" = "$EXPECTED_OLD_ID"
test "$(docker inspect sub2api --format '{{.State.Status}}')" = running
test "$(docker inspect sub2api --format '{{.State.Health.Status}}')" = healthy
test "$(docker inspect sub2api --format '{{.RestartCount}}')" = 0
test "$(docker compose --project-directory "$DEPLOY_DIR" --env-file "${DEPLOY_DIR}/.env" \
  -f "$LOCAL_COMPOSE" -f "$CUSTOM_COMPOSE" config --format json | jq -r '.services.sub2api.image')" = "$EXPECTED_OLD_REF"
test "$(docker image inspect "$EXPECTED_OLD_REF" --format '{{.Id}}')" = "$EXPECTED_OLD_ID"
test "$(docker image inspect "$ROLLBACK_REF" --format '{{.Id}}')" = "$EXPECTED_OLD_ID"
test "$(docker image inspect "$CANDIDATE_REF" --format '{{.Id}}')" = "$CANDIDATE_ID"
test "$(docker image inspect "$CANDIDATE_REF" --format '{{ index .Config.Labels "org.opencontainers.image.revision" }}')" = "$COMMIT"
test "$(docker image inspect "$CANDIDATE_REF" --format '{{ index .Config.Labels "org.opencontainers.image.version" }}')" = "$VERSION"
docker image inspect "$CANDIDATE_REF" --format '{{json .Config.Env}}' | jq -e 'all(.[]; startswith("NODE_OPTIONS=") | not)' >/dev/null
docker run --rm --pull never --entrypoint /bin/sh "$CANDIDATE_ID" -ec 'test "$(stat -c %a /app/docker-entrypoint.sh)" = 755'
docker run --rm --pull never --entrypoint /app/sub2api "$EXPECTED_OLD_ID" --migration-manifest > "${RELEASE_DIR}/old-migrations.json"
docker run --rm --pull never --entrypoint /app/sub2api "$CANDIDATE_ID" --migration-manifest > "${RELEASE_DIR}/candidate-migrations.json"
cmp -s "${RELEASE_DIR}/old-migrations.json" "${RELEASE_DIR}/candidate-migrations.json"
docker run --rm --pull never --entrypoint /app/sub2api "$CANDIDATE_ID" --version > "${RELEASE_DIR}/candidate-version.txt" 2>&1
grep -Fq "LuoxueAPI ${VERSION}" "${RELEASE_DIR}/candidate-version.txt"
grep -Fq "commit: ${COMMIT}," "${RELEASE_DIR}/candidate-version.txt"
curl --fail --silent --show-error --max-time 15 http://127.0.0.1:8080/health > "${RELEASE_DIR}/before-health.json"
jq -e '.status == "ok"' "${RELEASE_DIR}/before-health.json" >/dev/null
PG_ID="$(docker inspect sub2api-postgres --format '{{.Id}}')"
PG_STARTED="$(docker inspect sub2api-postgres --format '{{.State.StartedAt}}')"
PG_RESTARTS="$(docker inspect sub2api-postgres --format '{{.RestartCount}}')"
REDIS_ID="$(docker inspect sub2api-redis --format '{{.Id}}')"
REDIS_STARTED="$(docker inspect sub2api-redis --format '{{.State.StartedAt}}')"
REDIS_RESTARTS="$(docker inspect sub2api-redis --format '{{.RestartCount}}')"
assert_dependencies_unchanged
OLD_ENV_SHA="$(container_env_sha sub2api)"
OLD_MOUNTS_SHA="$(container_mounts_sha sub2api)"
OLD_PORTS_SHA="$(container_ports_sha sub2api)"
OLD_MODEL_SHA="$(compose_model_sha "$CUSTOM_COMPOSE")"
cp -a "$CUSTOM_COMPOSE" "$CUSTOM_BEFORE"

set_phase prepare_cutover
awk -v old="$EXPECTED_OLD_REF" -v new="$CANDIDATE_REF" '
  {
    line=$0
    v=$0
    sub(/^[[:space:]]*image:[[:space:]]*/, "", v)
    sub(/[[:space:]]*$/, "", v)
    if ((substr(v,1,1)=="\"" && substr(v,length(v),1)=="\"") ||
        (substr(v,1,1)=="\047" && substr(v,length(v),1)=="\047")) {
      v=substr(v,2,length(v)-2)
    }
    if (v==old) {
      sub(old,new,line)
      n++
    }
    print line
  }
  END {if(n!=1) exit 42}
' "$CUSTOM_BEFORE" > "$CUSTOM_WORK"
chown --reference="$CUSTOM_BEFORE" "$CUSTOM_WORK"
chmod --reference="$CUSTOM_BEFORE" "$CUSTOM_WORK"
docker compose --project-directory "$DEPLOY_DIR" --env-file "${DEPLOY_DIR}/.env" \
  -f "$LOCAL_COMPOSE" -f "$CUSTOM_WORK" config --quiet
test "$(docker compose --project-directory "$DEPLOY_DIR" --env-file "${DEPLOY_DIR}/.env" \
  -f "$LOCAL_COMPOSE" -f "$CUSTOM_WORK" config --format json | jq -r '.services.sub2api.image')" = "$CANDIDATE_REF"
test "$(compose_model_sha "$CUSTOM_WORK")" = "$OLD_MODEL_SHA"
assert_dependencies_unchanged
test "$(docker inspect sub2api --format '{{.Image}}')" = "$EXPECTED_OLD_ID"
cmp -s "$CUSTOM_BEFORE" "$CUSTOM_COMPOSE"

set_phase cutover
CUTOVER_STARTED=1
write_status running
mv -f "$CUSTOM_WORK" "$CUSTOM_COMPOSE"
docker compose --project-directory "$DEPLOY_DIR" --env-file "${DEPLOY_DIR}/.env" \
  -f "$LOCAL_COMPOSE" -f "$CUSTOM_COMPOSE" up -d --pull never --no-deps --force-recreate sub2api
wait_for_app "$CANDIDATE_ID" candidate

set_phase post_cutover_verify
assert_dependencies_unchanged
test "$(container_env_sha sub2api)" = "$OLD_ENV_SHA"
test "$(container_mounts_sha sub2api)" = "$OLD_MOUNTS_SHA"
test "$(container_ports_sha sub2api)" = "$OLD_PORTS_SHA"
test "$(compose_model_sha "$CUSTOM_COMPOSE")" = "$OLD_MODEL_SHA"
test "$(docker inspect sub2api --format '{{.Config.Image}}')" = "$CANDIDATE_REF"
test "$(docker inspect sub2api --format '{{.Image}}')" = "$CANDIDATE_ID"
test "$(docker inspect sub2api --format '{{.RestartCount}}')" = 0
test "$(docker compose --project-directory "$DEPLOY_DIR" --env-file "${DEPLOY_DIR}/.env" \
  -f "$LOCAL_COMPOSE" -f "$CUSTOM_COMPOSE" ps -q sub2api | wc -l)" -eq 1
docker exec sub2api /app/sub2api --version > "${RELEASE_DIR}/running-version.txt" 2>&1
grep -Fq "commit: ${COMMIT}," "${RELEASE_DIR}/running-version.txt"
docker logs --since 5m sub2api > "${RELEASE_DIR}/app-last-5m.log" 2>&1
if grep -Eia 'panic:|fatal:|failed to initialize|migration[^[:cntrl:]]*failed' "${RELEASE_DIR}/app-last-5m.log" >/dev/null; then false; fi
for attempt in $(seq 1 5); do
  if curl --fail --silent --show-error --max-time 15 https://luoxueapi.cc/health > "${RELEASE_DIR}/public-health.json" \
    && jq -e '.status == "ok"' "${RELEASE_DIR}/public-health.json" >/dev/null; then break; fi
  if test "$attempt" = 5; then false; fi
  sleep 5
done
sleep 15
wait_for_app "$CANDIDATE_ID" final
assert_dependencies_unchanged
{
  printf 'run_id=%s\ncommit=%s\nversion=%s\nimage_ref=%s\nimage_id=%s\n' "$RUN_ID" "$COMMIT" "$VERSION" "$CANDIDATE_REF" "$CANDIDATE_ID"
  printf 'old_image_id=%s\nrollback_ref=%s\ncustom_backup=%s\n' "$EXPECTED_OLD_ID" "$ROLLBACK_REF" "$CUSTOM_BEFORE"
  printf 'runtime_env_sha256=%s\nruntime_mounts_sha256=%s\nruntime_ports_sha256=%s\n' "$OLD_ENV_SHA" "$OLD_MOUNTS_SHA" "$OLD_PORTS_SHA"
  printf 'postgres_id=%s\npostgres_started_at=%s\nredis_id=%s\nredis_started_at=%s\n' "$PG_ID" "$PG_STARTED" "$REDIS_ID" "$REDIS_STARTED"
  printf 'completed_at=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
} > "${RELEASE_DIR}/final-evidence.txt"
