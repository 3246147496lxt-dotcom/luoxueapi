#!/usr/bin/env bash
set -Eeuo pipefail
umask 077

# One-shot production hotfix deploy. Intended to be transferred as base64 and
# launched from the provider VNC console. It never requires SSH or credentials.
BASE_COMMIT="d0a46e04eb8fbf94a90bad184fbb094736e8ca63"
COMMIT="d7d138bb776a3608aea0c04634c01ff32e002f36"
SHORT="d7d138bb7"
VERSION="2.0.0-rc.5"
REPO="3246147496lxt-dotcom/luoxueapi"
EXPECTED_OLD_REF="luoxueapi:d0a46e04e-skillimport-safety-r1"
EXPECTED_OLD_ID="sha256:5574502c04397c5af6b12be28fa82a015f7879bef004c5029f5386efd7e815d9"
BASE_ARCHIVE_SHA="9b08603b3d4d6f230aa9965ae54636c97b354aaf223035e7cca071d5d02bb3b1"
NEW_ARCHIVE_SHA="53dc85fb34a59f7a0fb567a5c45b33ff2668da55e65e83c753e31a6f1aaa5de3"
BASE_TREE_SHA="dbbe1409321a6a03cdfcbf45e626c815d79f4e4a234c1067f867a0d0cae807a9"
NEW_TREE_SHA="4f18dd3b3db59bf3ff89b5de217be72afc9bc56e6b3dbeed1cb301dcd59a0c18"

DEPLOY_DIR="/root/luoxueapi/deploy"
LOCAL_COMPOSE="${DEPLOY_DIR}/docker-compose.local.yml"
CUSTOM_COMPOSE="${DEPLOY_DIR}/docker-compose.custom.yml"
RELEASE_PARENT="/root/luoxueapi/releases"
STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
RUN_ID="hotfix-${SHORT}-${STAMP}"
RELEASE_DIR="${RELEASE_PARENT}/${RUN_ID}"
LATEST_FILE="${RELEASE_PARENT}/hotfix-${SHORT}.latest"
BASE_ARCHIVE="${RELEASE_DIR}/base-${BASE_COMMIT}.tar.gz"
NEW_ARCHIVE="${RELEASE_DIR}/source-${COMMIT}.tar.gz"
BASE_DIR="${RELEASE_DIR}/base"
SOURCE_DIR="${RELEASE_DIR}/source"
LOG_FILE="${RELEASE_DIR}/deploy.log"
STATUS_FILE="${RELEASE_DIR}/status"
CUSTOM_BEFORE="${RELEASE_DIR}/docker-compose.custom.before.yml"
CUSTOM_WORK="${DEPLOY_DIR}/.docker-compose.custom.${RUN_ID}.tmp"
IMAGE_REF="luoxueapi:${SHORT}-stage-null-${STAMP}"
ROLLBACK_REF="luoxueapi:rollback-d0a46e04e-before-${SHORT}-${STAMP}"

PHASE="bootstrap"
CUTOVER_STARTED=0
ROLLBACK_DONE=0
NEW_IMAGE_ID=""
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
    printf 'image_ref=%s\nimage_id=%s\nrollback_ref=%s\n' "$IMAGE_REF" "$NEW_IMAGE_ID" "$ROLLBACK_REF"
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
  local rc="$?" final_rc
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
    printf '[success] deployed %s as %s\n' "$COMMIT" "$NEW_IMAGE_ID"
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
for cmd in awk cmp curl diff docker find flock grep jq mv nproc sed seq sha256sum sort stat tar tee; do
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
test "$(docker image inspect "$EXPECTED_OLD_ID" --format '{{ index .Config.Labels "org.opencontainers.image.revision" }}')" = "$BASE_COMMIT"
test "$(docker compose --project-directory "$DEPLOY_DIR" --env-file "${DEPLOY_DIR}/.env" \
  -f "$LOCAL_COMPOSE" -f "$CUSTOM_COMPOSE" config --format json | jq -r '.services.sub2api.image')" = "$EXPECTED_OLD_REF"
docker run --rm --pull never --entrypoint /app/sub2api "$EXPECTED_OLD_ID" --version > "${RELEASE_DIR}/old-version.txt" 2>&1
grep -Fq "LuoxueAPI ${VERSION}" "${RELEASE_DIR}/old-version.txt"
grep -Fq "commit: ${BASE_COMMIT}," "${RELEASE_DIR}/old-version.txt"
docker run --rm --pull never --entrypoint /app/sub2api "$EXPECTED_OLD_ID" --migration-manifest > "${RELEASE_DIR}/old-migrations.json"
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
test "$(nproc)" -ge 2
ROOT_FREE="$(df -PB1 /root | awk 'NR==2 {print $4}')"
DOCKER_FREE="$(df -PB1 /var/lib/docker | awk 'NR==2 {print $4}')"
MEM_TOTAL="$(awk '/^MemTotal:/ {printf "%.0f", $2 * 1024}' /proc/meminfo)"
MEM_AVAILABLE="$(awk '/^MemAvailable:/ {printf "%.0f", $2 * 1024}' /proc/meminfo)"
SWAP_FREE="$(awk '/^SwapFree:/ {printf "%.0f", $2 * 1024}' /proc/meminfo)"
test "$ROOT_FREE" -ge $((5 * 1024 * 1024 * 1024))
test "$DOCKER_FREE" -ge $((5 * 1024 * 1024 * 1024))
test "$MEM_TOTAL" -ge $((3500 * 1024 * 1024))
test $((MEM_AVAILABLE + SWAP_FREE)) -ge $((4 * 1024 * 1024 * 1024))
printf 'root_free=%s\ndocker_free=%s\nmem_total=%s\nmem_available=%s\nswap_free=%s\n' \
  "$ROOT_FREE" "$DOCKER_FREE" "$MEM_TOTAL" "$MEM_AVAILABLE" "$SWAP_FREE" > "${RELEASE_DIR}/resource-gate.txt"
docker system df > "${RELEASE_DIR}/docker-system-df-before.txt"

set_phase download_and_verify_source
curl --proto '=https' --tlsv1.2 --fail --location --silent --show-error --retry 5 --retry-connrefused \
  --connect-timeout 20 --max-time 600 -o "$BASE_ARCHIVE" "https://codeload.github.com/${REPO}/tar.gz/${BASE_COMMIT}"
curl --proto '=https' --tlsv1.2 --fail --location --silent --show-error --retry 5 --retry-connrefused \
  --connect-timeout 20 --max-time 600 -o "$NEW_ARCHIVE" "https://codeload.github.com/${REPO}/tar.gz/${COMMIT}"
printf '%s  %s\n' "$BASE_ARCHIVE_SHA" "$BASE_ARCHIVE" | sha256sum --check --strict
printf '%s  %s\n' "$NEW_ARCHIVE_SHA" "$NEW_ARCHIVE" | sha256sum --check --strict
for archive in "$BASE_ARCHIVE" "$NEW_ARCHIVE"; do
  test "$(stat -c %s "$archive")" -lt $((64 * 1024 * 1024))
  tar -tzf "$archive" > "${archive}.list"
  test -s "${archive}.list"
  if grep -E '(^|/)\._' "${archive}.list" >/dev/null; then false; fi
  awk -F/ 'NF < 2 {exit 1} {root=$1; if (NR==1) first=root; if (root!=first) exit 1; for(i=1;i<=NF;i++) if($i==".." || $i==".") exit 1}' "${archive}.list"
  tar -tvzf "$archive" | awk 'substr($0,1,1)!="-" && substr($0,1,1)!="d" {exit 1}'
done
mkdir -m 700 "$BASE_DIR" "$SOURCE_DIR"
tar -xzf "$BASE_ARCHIVE" -C "$BASE_DIR" --strip-components=1 --no-same-owner
tar -xzf "$NEW_ARCHIVE" -C "$SOURCE_DIR" --strip-components=1 --no-same-owner
if find "$BASE_DIR" "$SOURCE_DIR" -name '._*' -o -type l | grep . >/dev/null; then false; fi
if find "$BASE_DIR" "$SOURCE_DIR" -mindepth 1 ! -type f ! -type d | grep . >/dev/null; then false; fi
test "$(find "$BASE_DIR" -type f | wc -l)" -eq 4023
test "$(find "$SOURCE_DIR" -type f | wc -l)" -eq 4023
tree_sha() { (cd "$1" && find . -type f -print0 | LC_ALL=C sort -z | xargs -0 sha256sum) | sha256sum | awk '{print $1}'; }
test "$(tree_sha "$BASE_DIR")" = "$BASE_TREE_SHA"
test "$(tree_sha "$SOURCE_DIR")" = "$NEW_TREE_SHA"
set +e
diff -qr "$BASE_DIR" "$SOURCE_DIR" > "${RELEASE_DIR}/source-diff.txt"
DIFF_RC="$?"
set -e
test "$DIFF_RC" = 1
sed -n "s#^Files ${BASE_DIR}/\\(.*\\) and ${SOURCE_DIR}/\\1 differ\$#\\1#p" "${RELEASE_DIR}/source-diff.txt" | LC_ALL=C sort > "${RELEASE_DIR}/changed-paths.txt"
cat > "${RELEASE_DIR}/expected-changed-paths.txt" <<'EOF'
backend/internal/repository/skill_import_repo.go
backend/internal/repository/skill_import_repo_integration_test.go
backend/internal/repository/skill_import_repo_test.go
EOF
cmp -s "${RELEASE_DIR}/expected-changed-paths.txt" "${RELEASE_DIR}/changed-paths.txt"
test "$(wc -l < "${RELEASE_DIR}/source-diff.txt")" -eq 3
printf '%s  %s\n' '21af821f8ed26af12e765d1dc5e1d4d33013c192bc013102473e0ff976c343d5' "$SOURCE_DIR/backend/internal/repository/skill_import_repo.go" | sha256sum --check --strict
printf '%s  %s\n' '0b291e35579e547a806e294948d83c09a7646864a07b2383bcb123af63d0ea2c' "$SOURCE_DIR/backend/internal/repository/skill_import_repo_integration_test.go" | sha256sum --check --strict
printf '%s  %s\n' 'acda14ccec4f4f5f9095744541f152c2178d1587c5079b494cc0a6118f6cbf9b' "$SOURCE_DIR/backend/internal/repository/skill_import_repo_test.go" | sha256sum --check --strict
test "$(sha256sum "$SOURCE_DIR/Dockerfile" | awk '{print $1}')" = 'dde3826ab66211f1ced555242bd0b75c4359b550ba990b6e5eb897e61bfc7771'
test "$(tr -d '\r\n' < "$SOURCE_DIR/backend/cmd/server/VERSION")" = "$VERSION"

set_phase build_and_verify_image
if docker image inspect "$IMAGE_REF" >/dev/null 2>&1; then false; fi
BUILD_DOCKERFILE="${RELEASE_DIR}/Dockerfile.build-overlay"
awk '
  $0 == "RUN chmod +x /app/docker-entrypoint.sh" {print "RUN chmod 755 /app/docker-entrypoint.sh"; next}
  {print}
  $0 == "FROM ${NODE_IMAGE} AS frontend-builder" {print "ARG NODE_OPTIONS=--max-old-space-size=3072"; print "ENV NODE_OPTIONS=${NODE_OPTIONS}"}
' "$SOURCE_DIR/Dockerfile" > "$BUILD_DOCKERFILE"
diff -u "$SOURCE_DIR/Dockerfile" "$BUILD_DOCKERFILE" > "${RELEASE_DIR}/Dockerfile.build-overlay.diff" || test "$?" = 1
test "$(grep -Ec '^[+-](ARG|ENV) NODE_OPTIONS=|^[+-]RUN chmod (\+x|755) /app/docker-entrypoint\.sh$' "${RELEASE_DIR}/Dockerfile.build-overlay.diff")" = 4
BUILD_DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
DOCKER_BUILDKIT=1 docker build --progress=plain --file "$BUILD_DOCKERFILE" \
  --build-arg NODE_OPTIONS=--max-old-space-size=3072 --build-arg VERSION="$VERSION" \
  --build-arg COMMIT="$COMMIT" --build-arg DATE="$BUILD_DATE" \
  --label org.opencontainers.image.revision="$COMMIT" --label org.opencontainers.image.version="$VERSION" \
  --label org.opencontainers.image.created="$BUILD_DATE" --label com.openai.codex.deploy.run_id="$RUN_ID" \
  --tag "$IMAGE_REF" "$SOURCE_DIR" > "${RELEASE_DIR}/image-build.log" 2>&1
NEW_IMAGE_ID="$(docker image inspect "$IMAGE_REF" --format '{{.Id}}')"
[[ "$NEW_IMAGE_ID" =~ ^sha256:[0-9a-f]{64}$ ]]
test "$NEW_IMAGE_ID" != "$EXPECTED_OLD_ID"
test "$(docker image inspect "$IMAGE_REF" --format '{{ index .Config.Labels "org.opencontainers.image.revision" }}')" = "$COMMIT"
test "$(docker image inspect "$IMAGE_REF" --format '{{ index .Config.Labels "org.opencontainers.image.version" }}')" = "$VERSION"
docker image inspect "$IMAGE_REF" --format '{{json .Config.Env}}' | jq -e 'all(.[]; startswith("NODE_OPTIONS=") | not)' >/dev/null
docker run --rm --pull never --entrypoint /bin/sh "$NEW_IMAGE_ID" -ec 'test "$(stat -c %a /app/docker-entrypoint.sh)" = 755'
docker run --rm --pull never --entrypoint /app/sub2api "$NEW_IMAGE_ID" --version > "${RELEASE_DIR}/candidate-version.txt" 2>&1
grep -Fq "LuoxueAPI ${VERSION}" "${RELEASE_DIR}/candidate-version.txt"
grep -Fq "commit: ${COMMIT}," "${RELEASE_DIR}/candidate-version.txt"
docker run --rm --pull never --entrypoint /app/sub2api "$NEW_IMAGE_ID" --migration-manifest > "${RELEASE_DIR}/candidate-migrations.json"
cmp -s "${RELEASE_DIR}/old-migrations.json" "${RELEASE_DIR}/candidate-migrations.json"
docker image inspect "$IMAGE_REF" > "${RELEASE_DIR}/candidate-image-inspect.json"
printf '%s\n' "$NEW_IMAGE_ID" > "${RELEASE_DIR}/candidate-image-id.txt"

set_phase prepare_cutover
if docker image inspect "$ROLLBACK_REF" >/dev/null 2>&1; then false; fi
docker tag "$EXPECTED_OLD_ID" "$ROLLBACK_REF"
test "$(docker image inspect "$ROLLBACK_REF" --format '{{.Id}}')" = "$EXPECTED_OLD_ID"
awk -v old="$EXPECTED_OLD_REF" -v new="$IMAGE_REF" '
  {v=$0; sub(/^[[:space:]]*image:[[:space:]]*/, "", v); sub(/[[:space:]]*$/, "", v)}
  v==old {sub(old, new); n++}
  {print}
  END {if(n!=1) exit 42}
' "$CUSTOM_BEFORE" > "$CUSTOM_WORK"
chown --reference="$CUSTOM_BEFORE" "$CUSTOM_WORK"
chmod --reference="$CUSTOM_BEFORE" "$CUSTOM_WORK"
docker compose --project-directory "$DEPLOY_DIR" --env-file "${DEPLOY_DIR}/.env" \
  -f "$LOCAL_COMPOSE" -f "$CUSTOM_WORK" config --quiet
test "$(docker compose --project-directory "$DEPLOY_DIR" --env-file "${DEPLOY_DIR}/.env" \
  -f "$LOCAL_COMPOSE" -f "$CUSTOM_WORK" config --format json | jq -r '.services.sub2api.image')" = "$IMAGE_REF"
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
wait_for_app "$NEW_IMAGE_ID" candidate

set_phase post_cutover_verify
assert_dependencies_unchanged
test "$(container_env_sha sub2api)" = "$OLD_ENV_SHA"
test "$(container_mounts_sha sub2api)" = "$OLD_MOUNTS_SHA"
test "$(container_ports_sha sub2api)" = "$OLD_PORTS_SHA"
test "$(compose_model_sha "$CUSTOM_COMPOSE")" = "$OLD_MODEL_SHA"
test "$(docker inspect sub2api --format '{{.Config.Image}}')" = "$IMAGE_REF"
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
test "$(docker inspect sub2api --format '{{.State.Health.Status}}')" = healthy
test "$(docker inspect sub2api --format '{{.RestartCount}}')" = 0
assert_dependencies_unchanged
docker system df > "${RELEASE_DIR}/docker-system-df-after.txt"
{
  printf 'run_id=%s\ncommit=%s\nversion=%s\nimage_ref=%s\nimage_id=%s\n' "$RUN_ID" "$COMMIT" "$VERSION" "$IMAGE_REF" "$NEW_IMAGE_ID"
  printf 'old_image_id=%s\nrollback_ref=%s\ncustom_backup=%s\n' "$EXPECTED_OLD_ID" "$ROLLBACK_REF" "$CUSTOM_BEFORE"
  printf 'runtime_env_sha256=%s\nruntime_mounts_sha256=%s\nruntime_ports_sha256=%s\n' "$OLD_ENV_SHA" "$OLD_MOUNTS_SHA" "$OLD_PORTS_SHA"
  printf 'postgres_id=%s\npostgres_started_at=%s\nredis_id=%s\nredis_started_at=%s\n' "$PG_ID" "$PG_STARTED" "$REDIS_ID" "$REDIS_STARTED"
  printf 'completed_at=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
} > "${RELEASE_DIR}/final-evidence.txt"
