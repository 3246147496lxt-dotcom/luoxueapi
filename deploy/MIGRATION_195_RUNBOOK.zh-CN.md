# Migration 195 生产发布 Runbook

适用迁移：`195_subscription_anchored_monthly_quota.sql`

本手册只定义发布步骤，不授权任何生产操作。所有停止服务、备份、迁移、缓存删除和恢复流量动作都必须进入变更单并由执行人与复核人双人确认。命令示例不包含密码、Token、Cookie、API Key 或私钥。

## 1. 发布模型与硬边界

Migration 195 将会员周额度统一为 `starts_at` 锚定的固定 `7x24h` 窗口，并恢复同一锚点下固定 `30x24h` 月窗口。它会从 durable billing ledger 重建当前月用量，并对不可重建、计费类型冲突、多重匹配和负金额失败关闭。

这不是滚动升级。唯一允许的顺序是：

1. 在线只读盘点；
2. 入口进入维护状态，停止并排空全部旧 writer；
3. 建立已验证的数据库恢复点并保留旧镜像；
4. maintenance preflight 返回可迁移；
5. 用固定候选镜像执行 `sub2api --migrate-only`；
6. 复检 migration checksum、约束和数据状态；
7. 精确失效 `billing:sub:*` 缓存；
8. 仅启动新版本 canary，验证后恢复流量。

硬边界：

- `--migrate-only` 必须同时传入只读挂载的 `--expected-database-identity-file`；它只加载配置、在 migration runner 固定的同一数据库连接上验证 v2 身份与主库状态、执行嵌入式 migration 后退出，不初始化 Redis、HTTP、worker、系统密钥或 simple-mode seed。
- 对已有数据库，195 是 maintenance-only migration；普通服务启动发现它 pending 时必须拒绝启动，不能自动执行。只有检查当下既不存在 `schema_migrations`、public schema 也没有既有业务表的全新库可以走首次完整自动初始化；仅缺少迁移元数据不能证明数据库为空。持久化 origin 为 `fresh` 只描述数据库最初来源，不代表后续升级仍可绕过维护门禁。首次初始化若已创建迁移表后中断，恢复也必须显式使用 `--migrate-only`。
- `--database-identity` 只加载同一份数据库配置并输出非敏感数据库身份，不运行 migration 或初始化应用组件。
- migration runner 会应用候选镜像内的全部 pending migrations，不保证只执行 195。任何未审批的 pending 文件都阻断发布。
- runner 的 advisory lock 只串行化 migration，不能替代业务 writer drain。
- 195 是事务型迁移。提交前失败会整体回滚；提交后旧二进制在业务语义上不兼容，禁止重新承接写流量。
- 迁移提交后不得修改 195 文件；修复必须使用新的 migration。
- 禁止 `docker compose down -v`、`FLUSHDB`、`FLUSHALL`、复制运行中的 `PGDATA` 充当备份，或在没有上下文时恢复旧镜像。

## 2. 运行变量与证据目录

以下路径必须按生产实际值确认。已知生产链路通常是 `/root/luoxueapi/deploy/docker-compose.local.yml` 与 `/root/luoxueapi/deploy/docker-compose.custom.yml`，但容器标签才是最终事实。

```bash
umask 077
export M195_RUN_ID="$(date -u +%Y%m%dT%H%M%SZ)"
[[ "$M195_RUN_ID" =~ ^[0-9]{8}T[0-9]{6}Z$ ]]
export M195_RUN_DIR="/var/tmp/luoxue-m195-${M195_RUN_ID}"
export M195_RELEASE_ROOT="/path/to/verified/release"
export M195_DEPLOY_DIR="/root/luoxueapi/deploy"
export M195_DB_SERVICE="approved-libpq-service"
export M195_APP_SERVICE="sub2api"
export M195_PG_SERVICE="postgres"
export M195_CANDIDATE_DIGEST_REF="ghcr.io/approved/luoxueapi@sha256:<64-hex>"
export M195_ROLLBACK_DIGEST_REF="ghcr.io/approved/luoxueapi@sha256:<64-hex>"
test ! -e "$M195_RUN_DIR"
install -d -m 700 "$M195_RUN_DIR"
printf 'sub2api-migration-195-evidence/v1\t%s\n' "$M195_RUN_ID" \
  > "$M195_RUN_DIR/evidence-context.tsv"
chmod 400 "$M195_RUN_DIR/evidence-context.tsv"
```

两个镜像引用都必须由已批准的 registry 发布流程给出真实 digest，不能保留占位符，也不能只提供 tag。旧版本若尚未存在于 registry，必须在维护开始前把当前容器的 image ID 发布到受控 registry，再解析成 digest，并证明重新拉取后的 image ID 与当前容器一致；本地 rollback tag 只能作为便利副本，不能作为恢复依据。

每个候选源码 commit 或 Migration 195 SHA 必须使用全新的 `M195_RUN_ID` 和此前不存在的 `M195_RUN_DIR`。旧阶段证据只能作为历史附件，禁止复制进新目录或据此放行。传输层断联时优先重新附着仍存活的持久 shell，并先验证只读的 `evidence-context.tsv` 及 5.2 节候选绑定合同；原 shell 进程丢失时按第 3 节的失败关闭规则暂停，不能重新执行本节的目录创建命令。

数据库认证使用权限为 `0600` 的 `PGPASSFILE` 或受管 secret provider。不要把密码放入 `--dbname`、DSN、命令行、shell history 或证据文件。所有命令禁止在 `set -x` 下执行；Redis 密码只在内存中比较，清理命令复用目标 Redis 容器已有的 `REDISCLI_AUTH`，证据只记录 `password`/`none` 认证模式。

## 3. 发布物、AppleDouble 与 Compose 核验

该产品的发布单元是包含前端资源和 Go 后端的完整 Docker 镜像，不是单独的前端 `dist`。

macOS 归档前、归档内、服务器解包后三处均须确认没有 `._*` AppleDouble 文件：

```bash
cd "$M195_RELEASE_ROOT"
test -z "$(find . -type f -name '._*' -print -quit)"

COPYFILE_DISABLE=1 tar --no-xattrs -czf \
  "$M195_RUN_DIR/release.tar.gz" .
test -z "$(tar -tzf "$M195_RUN_DIR/release.tar.gz" | grep -E '(^|/)\._' || true)"
sha256sum "$M195_RUN_DIR/release.tar.gz" \
  > "$M195_RUN_DIR/release.tar.gz.sha256"
```

从当前生产容器标签读取真实 Compose project、工作目录和文件链，并在同一个受控 Bash 进程中构造后续唯一允许使用的 Compose 命令：

```bash
set -Eeuo pipefail
export M195_COMPOSE_PROJECT="$(
  docker inspect sub2api \
    --format '{{ index .Config.Labels "com.docker.compose.project" }}'
)"
export M195_COMPOSE_WORKING_DIR="$(
  docker inspect sub2api \
    --format '{{ index .Config.Labels "com.docker.compose.project.working_dir" }}'
)"
export M195_COMPOSE_CONFIG_FILES="$(
  docker inspect sub2api \
    --format '{{ index .Config.Labels "com.docker.compose.project.config_files" }}'
)"
test -n "$M195_COMPOSE_PROJECT"
test -d "$M195_COMPOSE_WORKING_DIR"
test -n "$M195_COMPOSE_CONFIG_FILES"

# 当前生产链使用工作目录中的 .env 做 Compose 插值；显式固定它，避免后续
# cd 到 release checkout 后 Compose 从其他目录加载同名文件。
export M195_COMPOSE_DOTENV_FILE="$M195_COMPOSE_WORKING_DIR/.env"
test -f "$M195_COMPOSE_DOTENV_FILE"
test ! -L "$M195_COMPOSE_DOTENV_FILE"
M195_COMPOSE_DOTENV_FILE="$(realpath -e -- "$M195_COMPOSE_DOTENV_FILE")"
export M195_COMPOSE_DOTENV_FILE
printf '%s\n' "$M195_COMPOSE_PROJECT" \
  > "$M195_RUN_DIR/compose-project.txt"
printf '%s\n' "$M195_COMPOSE_WORKING_DIR" \
  > "$M195_RUN_DIR/compose-working-dir.txt"
printf '%s\n' "$M195_COMPOSE_CONFIG_FILES" \
  > "$M195_RUN_DIR/compose-config-files.txt"

IFS=',' read -r -a m195_compose_config_paths \
  <<< "$M195_COMPOSE_CONFIG_FILES"
test "${#m195_compose_config_paths[@]}" -gt 0
M195_COMPOSE_ARGS=(
  --project-directory "$M195_COMPOSE_WORKING_DIR"
  --env-file "$M195_COMPOSE_DOTENV_FILE"
  -p "$M195_COMPOSE_PROJECT"
)
: > "$M195_RUN_DIR/compose-config-files-resolved.txt"
for m195_compose_config_path in "${m195_compose_config_paths[@]}"; do
  case "$m195_compose_config_path" in
    /*) ;;
    *) m195_compose_config_path="$M195_COMPOSE_WORKING_DIR/$m195_compose_config_path" ;;
  esac
  m195_compose_config_path="$(realpath -e -- "$m195_compose_config_path")"
  test -r "$m195_compose_config_path"
  printf '%s\n' "$m195_compose_config_path" \
    >> "$M195_RUN_DIR/compose-config-files-resolved.txt"
  M195_COMPOSE_ARGS+=(-f "$m195_compose_config_path")
done
unset m195_compose_config_path m195_compose_config_paths

m195_compose() {
  docker compose "${M195_COMPOSE_ARGS[@]}" "$@"
}

m195_assert_container_compose_identity() {
  local phase="$1"
  case "$phase" in
    ''|*[!a-z0-9-]*) return 2 ;;
  esac
  local actual_project actual_working_dir actual_config_files actual_path
  local -a actual_config_paths
  actual_project="$(
    docker inspect sub2api \
      --format '{{ index .Config.Labels "com.docker.compose.project" }}'
  )"
  actual_working_dir="$(
    docker inspect sub2api \
      --format '{{ index .Config.Labels "com.docker.compose.project.working_dir" }}'
  )"
  actual_config_files="$(
    docker inspect sub2api \
      --format '{{ index .Config.Labels "com.docker.compose.project.config_files" }}'
  )"
  test "$actual_project" = "$M195_COMPOSE_PROJECT"
  test "$(realpath -e -- "$actual_working_dir")" \
    = "$(realpath -e -- "$M195_COMPOSE_WORKING_DIR")"
  IFS=',' read -r -a actual_config_paths <<< "$actual_config_files"
  test "${#actual_config_paths[@]}" -gt 0
  : > "$M195_RUN_DIR/compose-config-files-resolved-${phase}.txt"
  for actual_path in "${actual_config_paths[@]}"; do
    case "$actual_path" in
      /*) ;;
      *) actual_path="$actual_working_dir/$actual_path" ;;
    esac
    realpath -e -- "$actual_path" \
      >> "$M195_RUN_DIR/compose-config-files-resolved-${phase}.txt"
  done
  cmp -s \
    "$M195_RUN_DIR/compose-config-files-resolved.txt" \
    "$M195_RUN_DIR/compose-config-files-resolved-${phase}.txt"
}

export M195_DEPLOY_DIR="$M195_COMPOSE_WORKING_DIR"
cd "$M195_COMPOSE_WORKING_DIR"
m195_compose config --quiet
m195_compose config --services > "$M195_RUN_DIR/compose-services.txt"
m195_compose config --images > "$M195_RUN_DIR/compose-images.txt"

# 当前 local.yml + custom.yml 生产链把最后的镜像覆盖持久写入 custom.yml。
# 若该文件由配置管理生成，应把下面的结构化更新应用到权威源并重新生成，
# 不能只修改服务器上的派生副本。
export M195_IMAGE_PIN_FILE="$M195_COMPOSE_WORKING_DIR/docker-compose.custom.yml"
M195_IMAGE_PIN_FILE="$(realpath -e -- "$M195_IMAGE_PIN_FILE")"
export M195_IMAGE_PIN_FILE
test -f "$M195_IMAGE_PIN_FILE"
test -w "$M195_IMAGE_PIN_FILE"
grep -Fx "$M195_IMAGE_PIN_FILE" \
  "$M195_RUN_DIR/compose-config-files-resolved.txt"
test "$(tail -n 1 "$M195_RUN_DIR/compose-config-files-resolved.txt")" \
  = "$M195_IMAGE_PIN_FILE"
export M195_YQ_PATH="$(command -v yq)"
M195_YQ_PATH="$(realpath -e -- "$M195_YQ_PATH")"
export M195_YQ_PATH
yq --version \
  | tee "$M195_RUN_DIR/yq-version.txt" \
  | grep -Eq 'version v?4\.'
sha256sum "$M195_YQ_PATH" > "$M195_RUN_DIR/yq-binary.sha256"
export M195_FLOCK_PATH="$(command -v flock)"
M195_FLOCK_PATH="$(realpath -e -- "$M195_FLOCK_PATH")"
export M195_FLOCK_PATH
flock --version > "$M195_RUN_DIR/flock-version.txt"
sha256sum "$M195_FLOCK_PATH" > "$M195_RUN_DIR/flock-binary.sha256"
printf '%s\n' "$M195_IMAGE_PIN_FILE" \
  > "$M195_RUN_DIR/persistent-image-config-path.txt"
sha256sum "$M195_IMAGE_PIN_FILE" \
  > "$M195_RUN_DIR/persistent-image-config.before.sha256"

# 将全部 Compose YAML 与已固定的 .env 一起纳入 checksum 和配置冻结。
{
  sed -n '/./p' "$M195_RUN_DIR/compose-config-files-resolved.txt"
  printf '%s\n' "$M195_COMPOSE_DOTENV_FILE"
} | LC_ALL=C sort -u > "$M195_RUN_DIR/compose-input-paths.txt"

m195_record_compose_input_checksums() {
  local output_file="$1" input_path
  : > "$output_file"
  while IFS= read -r input_path; do
    test -f "$input_path"
    sha256sum "$input_path" >> "$output_file"
  done < "$M195_RUN_DIR/compose-input-paths.txt"
}

m195_record_compose_input_checksums_with_pin() {
  local staged_pin="$1" output_file="$2" input_path staged_sha256
  staged_sha256="$(sha256sum "$staged_pin" | awk '{print $1}')"
  [[ "$staged_sha256" =~ ^[0-9a-f]{64}$ ]]
  : > "$output_file"
  while IFS= read -r input_path; do
    test -f "$input_path"
    if test "$input_path" = "$M195_IMAGE_PIN_FILE"; then
      printf '%s  %s\n' "$staged_sha256" "$M195_IMAGE_PIN_FILE" \
        >> "$output_file"
    else
      sha256sum "$input_path" >> "$output_file"
    fi
  done < "$M195_RUN_DIR/compose-input-paths.txt"
}

m195_image_pin_state_matches() {
  local state="$1"
  case "$state" in before|candidate|rollback) ;; *) return 2 ;; esac
  test -s "$M195_RUN_DIR/persistent-image-config.${state}.sha256"
  test -s "$M195_RUN_DIR/compose-inputs.${state}.sha256"
  sha256sum --check \
    "$M195_RUN_DIR/persistent-image-config.${state}.sha256" \
    >/dev/null 2>&1
  sha256sum --check "$M195_RUN_DIR/compose-inputs.${state}.sha256" \
    >/dev/null 2>&1
}

m195_transition_image_pin() {
  local state="$1" image_ref="$2" staged_pin staged_sha256
  case "$state" in candidate|rollback) ;; *) return 2 ;; esac
  [[ "$M195_RUN_ID" =~ ^[0-9]{8}T[0-9]{6}Z$ ]]
  m195_assert_digest_ref "$image_ref"
  staged_pin="${M195_IMAGE_PIN_FILE}.m195-${M195_RUN_ID}.${state}.stage"

  # 断联遗留的 stage 不是证据。始终从当前已锁定、已校验的生产文件
  # 重新生成，避免旧 stage 携带 image 之外的陈旧配置。
  if test -e "$staged_pin"; then
    test -f "$staged_pin"
    test ! -L "$staged_pin"
    rm -f -- "$staged_pin"
  fi
  cp --preserve=all -- "$M195_IMAGE_PIN_FILE" "$staged_pin"
  M195_PIN_IMAGE_REF="$image_ref" \
    yq -i \
      '.services[strenv(M195_APP_SERVICE)].image = strenv(M195_PIN_IMAGE_REF)' \
      "$staged_pin"
  chown --reference="$M195_IMAGE_PIN_FILE" "$staged_pin"
  chmod --reference="$M195_IMAGE_PIN_FILE" "$staged_pin"
  M195_PIN_IMAGE_REF="$image_ref" \
    yq -e \
      '.services[strenv(M195_APP_SERVICE)].image == strenv(M195_PIN_IMAGE_REF)' \
      "$staged_pin" >/dev/null

  staged_sha256="$(sha256sum "$staged_pin" | awk '{print $1}')"
  [[ "$staged_sha256" =~ ^[0-9a-f]{64}$ ]]
  printf '%s  %s\n' "$staged_sha256" "$M195_IMAGE_PIN_FILE" \
    > "$M195_RUN_DIR/persistent-image-config.${state}.sha256"
  m195_record_compose_input_checksums_with_pin "$staged_pin" \
    "$M195_RUN_DIR/compose-inputs.${state}.sha256"

  # staged_pin 与目标在同一目录，rename 是单文件系统内原子替换；
  # checksum journal 在替换前完成，因此断联后可以判定并恢复状态。
  mv -f -- "$staged_pin" "$M195_IMAGE_PIN_FILE"
  m195_image_pin_state_matches "$state"
}

declare -a M195_COMPOSE_INPUT_LOCK_FDS=()
m195_lock_compose_inputs() {
  local input_path lock_file lock_fd
  test "${#M195_COMPOSE_INPUT_LOCK_FDS[@]}" = "0"
  while IFS= read -r input_path; do
    lock_file="${input_path}.m195.lock"
    : > "$lock_file"
    chmod 600 "$lock_file"
    exec {lock_fd}> "$lock_file"
    flock --exclusive --nonblock "$lock_fd"
    M195_COMPOSE_INPUT_LOCK_FDS+=("$lock_fd")
  done < "$M195_RUN_DIR/compose-input-paths.txt"
  test "${#M195_COMPOSE_INPUT_LOCK_FDS[@]}" \
    = "$(wc -l < "$M195_RUN_DIR/compose-input-paths.txt" | tr -d ' ')"
}

m195_assert_compose_input_locks() {
  local lock_fd
  test "${#M195_COMPOSE_INPUT_LOCK_FDS[@]}" -gt 0
  for lock_fd in "${M195_COMPOSE_INPUT_LOCK_FDS[@]}"; do
    test -e "/proc/$$/fd/$lock_fd"
  done
}

m195_record_compose_input_checksums \
  "$M195_RUN_DIR/compose-inputs.before.sha256"
```

不要保存完整 `docker compose config` 输出，它可能展开 secret。`M195_IMAGE_PIN_FILE` 必须是正常 Compose 文件链中已有的持久文件，禁止指向 `$M195_RUN_DIR`、临时 override 或未纳入生产启动命令的第三份文件。若生产不以该文件为权威源，必须在变更单中指定等价的受管源、应用同一 digest pin，并重新生成当前文件链后再继续。

整个维护窗口必须运行在受控的持久终端会话中。VNC/SSH 断开但原 shell 仍存活时，只能重新附着并验证锁 FD、candidate binding 和当前 pin journal；不得重跑本节的 baseline 写入命令。若原 shell 进程丢失，advisory file lock 也随之丢失，本次自动流程立即暂停：保持入口维护状态，由复核人按现有只读证据和 `before/candidate/rollback` journal 单独批准恢复步骤。禁止直接整节重跑、覆盖 `before` 证据、退回手写 `-f`、遗漏 `-p`，或在新 shell 中沿用未经重建和复核的函数/secret。

## 4. 预检数据库角色

建议使用独立、短期、只读的预检账号。最小权限为：

- 目标数据库 `CONNECT`；
- `public` schema `USAGE`；
- `groups`、`user_subscriptions`、`billing_usage_entries`、`usage_logs`、`schema_migrations` 的 `SELECT`；
- `pg_control_system()` 的受控 `EXECUTE` 权限，仅用于比对非敏感的 PostgreSQL `system_identifier`；
- `pg_read_all_stats` 成员资格，或等价的受控监控角色，以完整观察其他角色的 writer/transaction。

不需要读取 `users`、`accounts`、`api_keys`。`pg_read_all_stats` 具有较广的运行状态可见性，账号应受管、审计并按组织策略撤销。缺少该能力时 online 模式只会告警，maintenance 模式会直接阻断。

预检脚本同时使用 PostgreSQL `default_transaction_read_only=on` 与 `REPEATABLE READ READ ONLY`，只输出无维度聚合 JSON，不输出 ID、PII、完整 Key、单条金额或请求样本。

## 5. 在线盘点

### 5.1 保留旧镜像与固定候选镜像

```bash
set -Eeuo pipefail
case "$-" in
  *x*) printf '%s\n' 'Disable shell xtrace before release operations.' >&2; exit 1 ;;
esac

m195_assert_digest_ref() {
  local image_ref="$1"
  [[ "$image_ref" =~ ^[^@[:space:]]+@sha256:[0-9a-f]{64}$ ]]
}

m195_assert_digest_ref "$M195_CANDIDATE_DIGEST_REF"
m195_assert_digest_ref "$M195_ROLLBACK_DIGEST_REF"

export M195_OLD_IMAGE_ID="$(docker inspect sub2api --format '{{.Image}}')"
[[ "$M195_OLD_IMAGE_ID" =~ ^sha256:[0-9a-f]{64}$ ]]
docker image inspect "$M195_OLD_IMAGE_ID" \
  --format '{{.Id}} {{json .RepoDigests}}' \
  > "$M195_RUN_DIR/old-image.txt"
# 该本地 tag 不是持久回滚点；持久回滚点是下面验证的 registry digest。
docker image tag "$M195_OLD_IMAGE_ID" \
  "luoxueapi:rollback-m195-${M195_RUN_ID}"

export M195_ORIGINAL_PRODUCTION_IMAGE_REF="$(
  m195_compose config --format json \
    | jq -er --arg service "$M195_APP_SERVICE" \
        '.services[$service].image // empty'
)"
test -n "$M195_ORIGINAL_PRODUCTION_IMAGE_REF"
test "$(
  docker image inspect "$M195_ORIGINAL_PRODUCTION_IMAGE_REF" --format '{{.Id}}'
)" = "$M195_OLD_IMAGE_ID"
printf '%s\n' "$M195_ORIGINAL_PRODUCTION_IMAGE_REF" \
  > "$M195_RUN_DIR/original-production-image-ref.txt"

# Pulling an @sha256 reference verifies registry availability and content identity.
# A locally built or locally retagged image without this round trip is not releasable.
docker pull "$M195_ROLLBACK_DIGEST_REF" \
  > "$M195_RUN_DIR/rollback-image-pull.log" 2>&1
export M195_ROLLBACK_IMAGE_ID="$(
  docker image inspect "$M195_ROLLBACK_DIGEST_REF" --format '{{.Id}}'
)"
test "$M195_ROLLBACK_IMAGE_ID" = "$M195_OLD_IMAGE_ID"
docker image inspect "$M195_ROLLBACK_DIGEST_REF" \
  --format '{{json .RepoDigests}}' \
  | jq -e --arg ref "$M195_ROLLBACK_DIGEST_REF" 'index($ref) != null'
printf '%s\n' "$M195_ROLLBACK_DIGEST_REF" \
  > "$M195_RUN_DIR/rollback-image-digest-ref.txt"

docker pull "$M195_CANDIDATE_DIGEST_REF" \
  > "$M195_RUN_DIR/candidate-image-pull.log" 2>&1
export M195_CANDIDATE_IMAGE_ID="$(
  docker image inspect "$M195_CANDIDATE_DIGEST_REF" --format '{{.Id}}'
)"
[[ "$M195_CANDIDATE_IMAGE_ID" =~ ^sha256:[0-9a-f]{64}$ ]]
test "$M195_CANDIDATE_IMAGE_ID" != "$M195_OLD_IMAGE_ID"
docker image inspect "$M195_CANDIDATE_DIGEST_REF" \
  --format '{{.Id}} {{json .RepoDigests}}' \
  > "$M195_RUN_DIR/candidate-image.txt"
docker image inspect "$M195_CANDIDATE_DIGEST_REF" \
  --format '{{json .RepoDigests}}' \
  | jq -e --arg ref "$M195_CANDIDATE_DIGEST_REF" 'index($ref) != null'
printf '%s\n' "$M195_CANDIDATE_DIGEST_REF" \
  > "$M195_RUN_DIR/candidate-image-digest-ref.txt"

export M195_IMAGE_OVERRIDE="$M195_RUN_DIR/candidate-image.override.yml"
cat > "$M195_IMAGE_OVERRIDE" <<YAML
services:
  ${M195_APP_SERVICE}:
    image: ${M195_CANDIDATE_DIGEST_REF}
YAML
chmod 600 "$M195_IMAGE_OVERRIDE"
m195_compose -f "$M195_IMAGE_OVERRIDE" config --quiet
m195_compose -f "$M195_IMAGE_OVERRIDE" config --format json \
  | jq -er --arg service "$M195_APP_SERVICE" \
      '.services[$service].image // empty' \
  | grep -Fx "$M195_CANDIDATE_DIGEST_REF"
```

`M195_CANDIDATE_DIGEST_REF` 必须来自候选 CI 的同一 registry provenance，`M195_ROLLBACK_DIGEST_REF` 必须重新拉取后精确命中当前容器 image ID。任一 digest 在 registry 不可拉取、RepoDigests 不包含精确引用、旧 digest 与运行中 image ID 不同或候选与旧版相同都阻断。不得用 `docker tag`、`docker save/load` 或 `sha256:<local-image-id>` 伪装 registry digest。

在切换结束前，不删除旧镜像、registry rollback digest 或本地便利标签。

### 5.2 确认 pending migration 清单

候选镜像、release checkout 和数据库必须绑定到同一条迁移谱系。镜像只带有自声明 label 不足以放行：它还必须来自已验证的构建 provenance；若没有可验证 provenance，就从下面确认过的干净 checkout 重新构建候选镜像并重新记录 image ID。

```bash
set -Eeuo pipefail
cd "$M195_RELEASE_ROOT"
git diff --quiet
git diff --cached --quiet
test -z "$(git ls-files --others --exclude-standard)"
export M195_RELEASE_COMMIT="$(git rev-parse --verify HEAD)"
test -n "$M195_RELEASE_COMMIT"

export M195_IMAGE_REVISION="$(
  docker image inspect "$M195_CANDIDATE_DIGEST_REF" \
    --format '{{ index .Config.Labels "org.opencontainers.image.revision" }}'
)"
test "$M195_IMAGE_REVISION" = "$M195_RELEASE_COMMIT"

docker run --rm --pull never --entrypoint /app/sub2api \
  "$M195_CANDIDATE_DIGEST_REF" --version > "$M195_RUN_DIR/candidate-version.txt" 2>&1
grep -Fq -- "commit: ${M195_RELEASE_COMMIT}," \
  "$M195_RUN_DIR/candidate-version.txt"

shopt -s nullglob extglob
migration_paths=(backend/migrations/[0-9][0-9][0-9]*.sql)
test "${#migration_paths[@]}" -gt 0
for migration_path in "${migration_paths[@]}"; do
  migration_content="$(<"$migration_path")"
  migration_content="${migration_content##+([[:space:]])}"
  migration_content="${migration_content%%+([[:space:]])}"
  migration_checksum="$(
    printf '%s' "$migration_content" | sha256sum | awk '{print $1}'
  )"
  printf '%s\t%s\n' "$(basename "$migration_path")" "$migration_checksum"
done | LC_ALL=C sort -t $'\t' -k1,1 \
  > "$M195_RUN_DIR/checkout-migrations.tsv"
cut -f1 "$M195_RUN_DIR/checkout-migrations.tsv" \
  > "$M195_RUN_DIR/artifact-migrations.txt"

docker run --rm --pull never --entrypoint /app/sub2api \
  "$M195_CANDIDATE_DIGEST_REF" --migration-manifest \
  > "$M195_RUN_DIR/candidate-migrations.json"
jq -e '
  .contract == "sub2api-migration-manifest/v1"
  and (.set_sha256 | test("^[0-9a-f]{64}$"))
  and (.migrations | type == "array" and length > 0)
  and all(.migrations[];
    (.filename | test("^[0-9][0-9][0-9][a-zA-Z0-9_.-]*\\.sql$"))
    and (.sha256 | test("^[0-9a-f]{64}$"))
  )
' "$M195_RUN_DIR/candidate-migrations.json"
jq -r '.migrations[] | [.filename, .sha256] | @tsv' \
  "$M195_RUN_DIR/candidate-migrations.json" \
  | LC_ALL=C sort -t $'\t' -k1,1 \
  > "$M195_RUN_DIR/candidate-migrations.tsv"
cmp -s \
  "$M195_RUN_DIR/checkout-migrations.tsv" \
  "$M195_RUN_DIR/candidate-migrations.tsv"

export M195_CANDIDATE_195_SHA256="$(
  jq -er '
    [.migrations[]
      | select(.filename == "195_subscription_anchored_monthly_quota.sql")]
    | if length == 1 then .[0].sha256 else error("expected exactly one migration 195") end
  ' "$M195_RUN_DIR/candidate-migrations.json"
)"
[[ "$M195_CANDIDATE_195_SHA256" =~ ^[0-9a-f]{64}$ ]]

# 首次运行写入，断联恢复只允许逐字匹配；绝不以新候选覆盖旧绑定。
export M195_CANDIDATE_BINDING_FILE="$M195_RUN_DIR/candidate-binding.json"
m195_candidate_binding="$(
  jq -cnS \
    --arg run_id "$M195_RUN_ID" \
    --arg release_commit "$M195_RELEASE_COMMIT" \
    --arg candidate_digest_ref "$M195_CANDIDATE_DIGEST_REF" \
    --arg candidate_image_id "$M195_CANDIDATE_IMAGE_ID" \
    --arg rollback_digest_ref "$M195_ROLLBACK_DIGEST_REF" \
    --arg rollback_image_id "$M195_ROLLBACK_IMAGE_ID" \
    --arg migration_195_sha256 "$M195_CANDIDATE_195_SHA256" '
      {
        contract: "sub2api-migration-195-candidate-binding/v1",
        run_id: $run_id,
        release_commit: $release_commit,
        candidate_digest_ref: $candidate_digest_ref,
        candidate_image_id: $candidate_image_id,
        rollback_digest_ref: $rollback_digest_ref,
        rollback_image_id: $rollback_image_id,
        migration_195_sha256: $migration_195_sha256
      }
    '
)"
if test -e "$M195_CANDIDATE_BINDING_FILE"; then
  test -f "$M195_CANDIDATE_BINDING_FILE"
  test ! -L "$M195_CANDIDATE_BINDING_FILE"
  sha256sum --check "$M195_RUN_DIR/candidate-binding.sha256"
  test "$(jq -ceS . "$M195_CANDIDATE_BINDING_FILE")" \
    = "$m195_candidate_binding"
else
  printf '%s\n' "$m195_candidate_binding" \
    > "$M195_CANDIDATE_BINDING_FILE"
  chmod 400 "$M195_CANDIDATE_BINDING_FILE"
  sha256sum "$M195_CANDIDATE_BINDING_FILE" \
    > "$M195_RUN_DIR/candidate-binding.sha256"
  chmod 400 "$M195_RUN_DIR/candidate-binding.sha256"
fi
unset m195_candidate_binding

psql -X --no-password --quiet --tuples-only --no-align \
  --field-separator=$'\t' \
  --dbname="service=${M195_DB_SERVICE}" \
  --command='SELECT filename, checksum FROM schema_migrations ORDER BY filename COLLATE "C"' \
  > "$M195_RUN_DIR/database-migrations.tsv"
cut -f1 "$M195_RUN_DIR/database-migrations.tsv" \
  > "$M195_RUN_DIR/applied-migrations.txt"

comm -13 \
  "$M195_RUN_DIR/artifact-migrations.txt" \
  "$M195_RUN_DIR/applied-migrations.txt" \
  > "$M195_RUN_DIR/database-only-migrations.txt"
test ! -s "$M195_RUN_DIR/database-only-migrations.txt"

comm -23 \
  "$M195_RUN_DIR/artifact-migrations.txt" \
  "$M195_RUN_DIR/applied-migrations.txt" \
  | tee "$M195_RUN_DIR/pending-migrations.txt"
```

`checkout-migrations.tsv` 使用与 Go runner 相同的 `strings.TrimSpace` 后 SHA-256 口径，并已与候选二进制直接导出的 manifest 逐行相等。只有变更单明确列出的 pending migrations 可以继续；候选与 checkout manifest 不同、数据库存在 checkout 缺失的 migration、source commit 不一致、provenance 不可验证或候选镜像内没有真实入口都必须阻断。Runner 仍会在真正执行前逐项校验数据库 checksum，包括代码中明确白名单的历史兼容 checksum：

```bash
docker run --rm --pull never --entrypoint /app/sub2api \
  "$M195_CANDIDATE_DIGEST_REF" --help 2>&1 \
  | grep -F -- '-migrate-only'
docker run --rm --pull never --entrypoint /app/sub2api \
  "$M195_CANDIDATE_DIGEST_REF" --help 2>&1 \
  | grep -F -- '-database-identity'
docker run --rm --pull never --entrypoint /app/sub2api \
  "$M195_CANDIDATE_DIGEST_REF" --help 2>&1 \
  | grep -F -- '-expected-database-identity-file'
```

### 5.3 Online preflight

在受信任的管理主机、经校验的 release checkout 中运行：

```bash
cd "$M195_RELEASE_ROOT"
backend/scripts/preflight-subscription-anchored-monthly-quota.sh \
  --mode online \
  --dbname "service=${M195_DB_SERVICE}" \
  > "$M195_RUN_DIR/preflight-online.json" \
  2> "$M195_RUN_DIR/preflight-online.log"

jq -e --arg expected "$M195_CANDIDATE_195_SHA256" '
  .contract == "migration-195-preflight/v1"
  and .migration_sha256 == $expected
  and .read_only_verified == true
' "$M195_RUN_DIR/preflight-online.json"
```

Online 模式永远不会返回 `safe_to_apply=true`。必须审阅：

- receipt 可重建性、计费身份、多候选和负金额；
- 当前权威 weekly counter 与 durable ledger 差异；
- active plan 和 subscription 时间不变量；
- reopened term 歧义；
- 目标表规模、dead rows、writer locks 与观察权限；
- 需要重锚的窗口数量。

使用同规模 staging 副本演练实际 migration 时长。`--migrate-only` 的超时为 10 分钟；预估无法稳定留出足够余量时，停止本次发布并先拆分/优化 migration。

### 5.4 绑定预检、备份与迁移数据库身份

候选二进制的 `--database-identity` 只加载迁移配置、连接 PostgreSQL、读取 `current_database()`、`pg_control_system().system_identifier` 与 `pg_is_in_recovery()` 后输出 `sub2api-database-identity/v2`。它不运行 migration，也不初始化 Redis、HTTP、worker、密钥或 seed。用它将候选的真实配置连接、预检连接与 Compose PostgreSQL 备份目标绑定为同一可写主库：

```bash
m195_validate_database_identity_file() {
  local identity_file="$1"
  test -f "$identity_file"
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
}

m195_capture_database_identities() {
  local phase="$1"
  case "$phase" in
    ''|*[!a-z0-9-]*) return 2 ;;
  esac

  psql -X --no-password --quiet --tuples-only --no-align \
    --set=ON_ERROR_STOP=1 \
    --dbname="service=${M195_DB_SERVICE}" \
    --command='SELECT ROW_TO_JSON(identity)::text FROM (SELECT current_database() AS database, system_identifier::text AS system_identifier, pg_is_in_recovery() AS in_recovery FROM pg_control_system()) AS identity' \
    | jq -ceS '. + {contract: "sub2api-database-identity/v2"}' \
    > "$M195_RUN_DIR/database-identity-preflight-${phase}.json"

  m195_compose -f "$M195_IMAGE_OVERRIDE" \
    run --rm --no-deps --pull never --entrypoint /app/sub2api \
    "$M195_APP_SERVICE" --database-identity \
    | jq -ceS . \
    > "$M195_RUN_DIR/database-identity-candidate-${phase}.json"

  m195_compose exec -T "$M195_PG_SERVICE" sh -ec '
    export PGPASSWORD="$POSTGRES_PASSWORD"
    unset POSTGRES_PASSWORD
    exec psql -X --no-password --quiet --tuples-only --no-align \
      --set=ON_ERROR_STOP=1 \
      --host=127.0.0.1 --port="${PGPORT:-5432}" \
      --username="$POSTGRES_USER" --dbname="$POSTGRES_DB" \
      --command="SELECT ROW_TO_JSON(identity)::text FROM (SELECT current_database() AS database, system_identifier::text AS system_identifier, pg_is_in_recovery() AS in_recovery FROM pg_control_system()) AS identity"
  ' | jq -ceS '. + {contract: "sub2api-database-identity/v2"}' \
    > "$M195_RUN_DIR/database-identity-backup-${phase}.json"

  m195_validate_database_identity_file \
    "$M195_RUN_DIR/database-identity-preflight-${phase}.json"
  m195_validate_database_identity_file \
    "$M195_RUN_DIR/database-identity-candidate-${phase}.json"
  m195_validate_database_identity_file \
    "$M195_RUN_DIR/database-identity-backup-${phase}.json"

  cmp -s \
    "$M195_RUN_DIR/database-identity-preflight-${phase}.json" \
    "$M195_RUN_DIR/database-identity-candidate-${phase}.json"
  cmp -s \
    "$M195_RUN_DIR/database-identity-preflight-${phase}.json" \
    "$M195_RUN_DIR/database-identity-backup-${phase}.json"
}

m195_capture_database_identities online
```

三份规范化 v2 JSON 必须逐字相等，且 `in_recovery` 必须显式为 `false`。缺少权限、任一路径无法查询、合同/字段集合不精确、database 名或 `system_identifier` 不同、任一目标为物理副本都立即阻断。不得通过修改证据文件、跳过某一路径、删除合同字段或仅比较数据库名放行。

### 5.5 绑定应用实际 Redis 目标

禁止手填 Redis service、host、port、DB、TLS 或认证参数。以下函数从合并后的应用 service 环境派生目标，并在不落盘密码的前提下与当前运行容器逐字段比较。

本次 Runbook 只允许当前生产使用的内部 Compose Redis：`REDIS_HOST` 必须精确等于 Compose service 名、`REDIS_ENABLE_TLS=false`，并且应用密码必须与该 Redis service 的 Compose 配置及运行容器环境逐字相等。这样清理命令可以在目标 Redis 容器内连接 loopback，完全绕开 DNS、代理和负载均衡。若派生结果为外部 Redis 或 TLS Redis，本节会在发送凭据或执行删除前失败关闭；不得用通用 `redis-cli --tls` 绕过，因为 SNI 不等于证书 SAN/CN hostname 验证：

```bash
m195_compose_redis_config_json() {
  m195_compose config --format json \
    | jq -ceS --arg service "$M195_APP_SERVICE" '
        def required($env; $name):
          if ($env | has($name)) then $env[$name]
          else error("missing app Redis setting: " + $name)
          end;

        . as $root
        | ($root.services[$service] // error("missing app service")) as $app
        | ($app.environment // error("missing app environment")) as $env
        | (required($env; "REDIS_HOST") | tostring) as $host
        | (required($env; "REDIS_PORT") | tostring) as $port
        | (required($env; "REDIS_PASSWORD")) as $password
        | (required($env; "REDIS_DB") | tostring) as $db
        | (required($env; "REDIS_ENABLE_TLS") | tostring) as $tls
        | if (($host | length) > 0
              and ($host | test("^[^[:space:]/]+$"))
              and ($host | contains("://") | not))
          then . else error("invalid REDIS_HOST") end
        | if ($port | test("^[1-9][0-9]*$")
              and (($port | tonumber) >= 1)
              and (($port | tonumber) <= 65535))
          then . else error("invalid REDIS_PORT") end
        | if ($db | test("^(0|[1-9][0-9]*)$")
              and (($db | tonumber) <= 2147483647))
          then . else error("invalid REDIS_DB") end
        | if ($tls == "true" or $tls == "false")
          then . else error("REDIS_ENABLE_TLS must be true or false") end
        | if ($password | type) == "string"
          then . else error("REDIS_PASSWORD must resolve to a string") end
        | ($root.services | has($host)) as $internal
        | {
            contract: "migration-195-redis-config-identity/v1",
            app_service: $service,
            endpoint_kind: (if $internal then "compose-service" else "external" end),
            redis_service: (if $internal then $host else null end),
            host: $host,
            port: ($port | tonumber),
            db: ($db | tonumber),
            tls: ($tls == "true"),
            auth_mode: (if ($password | length) > 0 then "password" else "none" end)
          }
      '
}

m195_assert_running_app_redis_config() {
  case "$-" in
    *x*) printf '%s\n' 'Disable shell xtrace before Redis checks.' >&2; return 1 ;;
  esac

  local compose_values running_values
  export M195_APP_CONTAINER_ID="$(
    m195_compose ps --all --quiet "$M195_APP_SERVICE"
  )"
  test -n "$M195_APP_CONTAINER_ID"
  test "$(printf '%s\n' "$M195_APP_CONTAINER_ID" | wc -l | tr -d ' ')" = "1"
  test "$(docker inspect "$M195_APP_CONTAINER_ID" --format '{{.State.Status}}')" \
    = "running"

  compose_values="$(
    m195_compose config --format json \
      | jq -ce --arg service "$M195_APP_SERVICE" '
          .services[$service].environment as $env
          | if ($env | has("REDIS_HOST")
                and has("REDIS_PORT")
                and has("REDIS_PASSWORD")
                and has("REDIS_DB")
                and has("REDIS_ENABLE_TLS"))
            then [
              $env.REDIS_HOST,
              $env.REDIS_PORT,
              $env.REDIS_PASSWORD,
              $env.REDIS_DB,
              $env.REDIS_ENABLE_TLS
            ]
            else error("incomplete Compose Redis environment") end
        '
  )" || return 1

  running_values="$(
    docker inspect "$M195_APP_CONTAINER_ID" --format '{{json .Config.Env}}' \
      | jq -ce '
          reduce .[] as $item ({};
            ($item | index("=")) as $split
            | if $split == null then .
              else .[$item[0:$split]] = $item[($split + 1):]
              end
          )
          | if (has("REDIS_HOST")
                and has("REDIS_PORT")
                and has("REDIS_PASSWORD")
                and has("REDIS_DB")
                and has("REDIS_ENABLE_TLS"))
            then [
              .REDIS_HOST,
              .REDIS_PORT,
              .REDIS_PASSWORD,
              .REDIS_DB,
              .REDIS_ENABLE_TLS
            ]
            else error("incomplete running Redis environment") end
        '
  )" || return 1

  test "$compose_values" = "$running_values"
  unset compose_values running_values
}

m195_bind_internal_redis_target() {
  case "$-" in
    *x*) printf '%s\n' 'Disable shell xtrace before Redis checks.' >&2; return 1 ;;
  esac

  jq -e '
    .endpoint_kind == "compose-service"
    and .redis_service == .host
    and .tls == false
  ' "$M195_RUN_DIR/redis-config-identity-online.json"

  export M195_REDIS_SERVICE="$(jq -er '.redis_service // empty' \
    "$M195_RUN_DIR/redis-config-identity-online.json")"
  export M195_REDIS_PORT="$(jq -er '.port' \
    "$M195_RUN_DIR/redis-config-identity-online.json")"
  export M195_REDIS_DB="$(jq -er '.db' \
    "$M195_RUN_DIR/redis-config-identity-online.json")"
  test -n "$M195_REDIS_SERVICE"

  export M195_REDIS_CONTAINER_ID="$(
    m195_compose ps --all --quiet "$M195_REDIS_SERVICE"
  )"
  test -n "$M195_REDIS_CONTAINER_ID"
  test "$(printf '%s\n' "$M195_REDIS_CONTAINER_ID" | wc -l | tr -d ' ')" = "1"
  test "$(docker inspect "$M195_REDIS_CONTAINER_ID" --format '{{.State.Status}}')" \
    = "running"
  test "$(
    docker inspect "$M195_REDIS_CONTAINER_ID" \
      --format '{{ index .Config.Labels "com.docker.compose.project" }}'
  )" = "$M195_COMPOSE_PROJECT"
  test "$(
    docker inspect "$M195_REDIS_CONTAINER_ID" \
      --format '{{ index .Config.Labels "com.docker.compose.service" }}'
  )" = "$M195_REDIS_SERVICE"
  export M195_REDIS_IMAGE_ID="$(
    docker inspect "$M195_REDIS_CONTAINER_ID" --format '{{.Image}}'
  )"
  [[ "$M195_REDIS_IMAGE_ID" =~ ^sha256:[0-9a-f]{64}$ ]]

  m195_assert_internal_redis_auth_config
}

m195_assert_internal_redis_auth_config() {
  case "$-" in
    *x*) printf '%s\n' 'Disable shell xtrace before Redis checks.' >&2; return 1 ;;
  esac

  local app_password_json server_password_json running_password_json

  app_password_json="$(
    m195_compose config --format json \
      | jq -ce --arg service "$M195_APP_SERVICE" \
          '.services[$service].environment.REDIS_PASSWORD'
  )" || return 1
  server_password_json="$(
    m195_compose config --format json \
      | jq -ce --arg service "$M195_REDIS_SERVICE" \
          '.services[$service].environment.REDISCLI_AUTH'
  )" || return 1
  running_password_json="$(
    docker inspect "$M195_REDIS_CONTAINER_ID" --format '{{json .Config.Env}}' \
      | jq -ce '
          reduce .[] as $item ({};
            ($item | index("=")) as $split
            | if $split == null then .
              else .[$item[0:$split]] = $item[($split + 1):]
              end
          )
          | if has("REDISCLI_AUTH") then .REDISCLI_AUTH
            else error("running Redis lacks REDISCLI_AUTH") end
        '
  )" || return 1
  test "$app_password_json" = "$server_password_json"
  test "$app_password_json" = "$running_password_json"
  unset app_password_json server_password_json running_password_json
}

m195_compose_redis_networks_json() {
  m195_compose config --format json \
    | jq -ceS \
        --arg app_service "$M195_APP_SERVICE" \
        --arg redis_service "$M195_REDIS_SERVICE" '
      def service_network_keys($service):
        if (($service.networks // null) == null) then ["default"]
        elif (($service.networks | type) == "object")
          then ($service.networks | keys)
        else error("service networks must be an object")
        end;

      . as $root
      | ($root.services[$app_service] // error("missing app service")) as $app
      | ($root.services[$redis_service] // error("missing Redis service")) as $redis
      | service_network_keys($app) as $app_keys
      | service_network_keys($redis) as $redis_keys
      | [$app_keys[] as $key
          | ($root.networks[$key].name
              // error("missing resolved app network: " + $key))]
        | unique | sort as $app_networks
      | [$redis_keys[] as $key
          | ($root.networks[$key].name
              // error("missing resolved Redis network: " + $key))]
        | unique | sort as $redis_networks
      | [$app_networks[] as $name
          | select(($redis_networks | index($name)) != null)
          | $name]
        | unique | sort as $shared_networks
      | if ($shared_networks | length) > 0 then {
          contract: "migration-195-redis-compose-networks/v1",
          app_service: $app_service,
          redis_service: $redis_service,
          host: $redis_service,
          app_networks: $app_networks,
          redis_networks: $redis_networks,
          shared_networks: $shared_networks
        }
        else error("app and Redis have no shared Compose network")
        end
    '
}

m195_capture_redis_network_identity() {
  local phase compose_networks app_networks redis_networks
  phase="$1"
  case "$phase" in
    ''|*[!a-z0-9-]*) return 2 ;;
  esac

  test -n "$M195_APP_CONTAINER_ID"
  test "$(m195_compose ps --all --quiet "$M195_APP_SERVICE")" \
    = "$M195_APP_CONTAINER_ID"
  test "$(m195_compose ps --all --quiet "$M195_REDIS_SERVICE")" \
    = "$M195_REDIS_CONTAINER_ID"

  compose_networks="$(m195_compose_redis_networks_json)" || return 1
  app_networks="$(
    docker inspect "$M195_APP_CONTAINER_ID" \
      --format '{{json .NetworkSettings.Networks}}'
  )" || return 1
  redis_networks="$(
    docker inspect "$M195_REDIS_CONTAINER_ID" \
      --format '{{json .NetworkSettings.Networks}}'
  )" || return 1

  jq -cnS \
    --arg app_container_id "$M195_APP_CONTAINER_ID" \
    --arg redis_container_id "$M195_REDIS_CONTAINER_ID" \
    --argjson compose "$compose_networks" \
    --argjson app "$app_networks" \
    --argjson redis "$redis_networks" '
      ($app | keys | sort) as $running_app_networks
      | ($redis | keys | sort) as $running_redis_networks
      | if $running_app_networks != $compose.app_networks
        then error("running app networks differ from Compose")
        elif $running_redis_networks != $compose.redis_networks
        then error("running Redis networks differ from Compose")
        else .
        end
      | [$compose.shared_networks[] as $network
          | ($app[$network] // error("app is missing shared network")) as $app_net
          | ($redis[$network] // error("Redis is missing shared network")) as $redis_net
          | if (($app_net.NetworkID | type) == "string"
                and ($app_net.NetworkID | length) > 0
                and $app_net.NetworkID == $redis_net.NetworkID
                and (($redis_net.Aliases // []) | index($compose.host)) != null)
            then {
              name: $network,
              network_id: $app_net.NetworkID,
              redis_host_alias_verified: true
            }
            else error("shared network ID or Redis service alias mismatch")
            end]
        as $bound_networks
      | {
          contract: "migration-195-redis-network-identity/v1",
          app_service: $compose.app_service,
          app_container_id: $app_container_id,
          redis_service: $compose.redis_service,
          redis_container_id: $redis_container_id,
          host: $compose.host,
          app_networks: $compose.app_networks,
          redis_networks: $compose.redis_networks,
          bound_shared_networks: $bound_networks
        }
    ' > "$M195_RUN_DIR/redis-network-identity-${phase}.json"

  unset compose_networks app_networks redis_networks
}

m195_run_internal_redis_probe() {
  local current_container_id probe
  current_container_id="$(m195_compose ps --all --quiet "$M195_REDIS_SERVICE")"
  test "$current_container_id" = "$M195_REDIS_CONTAINER_ID"
  m195_assert_internal_redis_auth_config

  probe="$(
    docker exec \
      -e M195_REDIS_PORT="$M195_REDIS_PORT" \
      -e M195_REDIS_DB="$M195_REDIS_DB" \
      "$M195_REDIS_CONTAINER_ID" sh -eu -c '
      case "$M195_REDIS_PORT" in ""|*[!0-9]*) exit 21 ;; esac
      case "$M195_REDIS_DB" in ""|*[!0-9]*) exit 22 ;; esac
      test "${REDISCLI_AUTH+x}" = x
      redis_call() {
        redis-cli -e --raw -h 127.0.0.1 -p "$M195_REDIS_PORT" \
          -n "$M195_REDIS_DB" "$@"
      }
      test "$(redis_call PING)" = PONG
      client_info="$(redis_call CLIENT INFO)"
      selected_db="$(
        printf "%s\n" "$client_info" \
          | tr " " "\n" \
          | awk -F= '\''$1 == "db" { sub(/\r$/, "", $2); print $2 }'\''
      )"
      test "$selected_db" = "$M195_REDIS_DB"
      server_info="$(redis_call INFO server)"
      run_id="$(
        printf "%s\n" "$server_info" \
          | awk -F: '\''$1 == "run_id" { sub(/\r$/, "", $2); print $2 }'\''
      )"
      test -n "$run_id"
      role="$(redis_call ROLE | sed -n '\''1 { s/\r$//; p; }'\'')"
      test "$role" = master
      run_id_sha256="$(printf "%s" "$run_id" | sha256sum | awk '\''{print $1}'\'')"
      if [ -n "$REDISCLI_AUTH" ]; then auth_mode=password; else auth_mode=none; fi
      unset client_info run_id server_info
      printf "%s\t%s\t%s\t%s\n" \
        "$auth_mode" "$run_id_sha256" "$role" "$selected_db"
    '
  )" || return 1
  test "$(m195_compose ps --all --quiet "$M195_REDIS_SERVICE")" \
    = "$M195_REDIS_CONTAINER_ID"
  printf '%s\n' "$probe"
}

m195_capture_redis_server_identity() {
  local phase probe auth_mode run_id_sha256 role selected_db extra
  phase="$1"
  case "$phase" in
    ''|*[!a-z0-9-]*) return 2 ;;
  esac

  m195_compose_redis_config_json \
    > "$M195_RUN_DIR/redis-config-identity-${phase}.json"
  jq -e \
    --arg service "$M195_REDIS_SERVICE" \
    --argjson port "$M195_REDIS_PORT" \
    --argjson db "$M195_REDIS_DB" '
      .endpoint_kind == "compose-service"
      and .redis_service == $service
      and .host == $service
      and .port == $port
      and .db == $db
      and .tls == false
    ' "$M195_RUN_DIR/redis-config-identity-${phase}.json"

  probe="$(m195_run_internal_redis_probe)" || return 1
  IFS=$'\t' read -r auth_mode run_id_sha256 role selected_db extra \
    <<< "$probe"
  test -z "${extra:-}"
  [[ "$run_id_sha256" =~ ^[0-9a-f]{64}$ ]]
  test "$role" = master
  test "$selected_db" = "$M195_REDIS_DB"

  jq -cS \
    --arg redis_service "$M195_REDIS_SERVICE" \
    --arg container_id "$M195_REDIS_CONTAINER_ID" \
    --arg image_id "$M195_REDIS_IMAGE_ID" \
    --arg auth_mode "$auth_mode" \
    --arg run_id_sha256 "$run_id_sha256" \
    --arg role "$role" \
    --argjson selected_db "$selected_db" \
    '{
      contract: "migration-195-redis-server-identity/v1",
      endpoint_kind: .endpoint_kind,
      redis_service: $redis_service,
      host: .host,
      port: .port,
      db: .db,
      selected_db: $selected_db,
      tls: .tls,
      auth_mode: $auth_mode,
      container_id: $container_id,
      image_id: $image_id,
      run_id_sha256: $run_id_sha256,
      role: $role
    }' "$M195_RUN_DIR/redis-config-identity-${phase}.json" \
    > "$M195_RUN_DIR/redis-server-identity-${phase}.json"

  jq -e --slurpfile config \
    "$M195_RUN_DIR/redis-config-identity-${phase}.json" '
      . as $server
      | $config[0] as $config
      | $server.host == $config.host
      and $server.port == $config.port
      and $server.db == $config.db
      and $server.selected_db == $config.db
      and $server.tls == $config.tls
      and $server.auth_mode == $config.auth_mode
      and $server.role == "master"
    ' "$M195_RUN_DIR/redis-server-identity-${phase}.json"
}

m195_assert_running_app_redis_config
m195_compose_redis_config_json \
  > "$M195_RUN_DIR/redis-config-identity-online.json"
m195_bind_internal_redis_target
m195_capture_redis_network_identity online
m195_capture_redis_server_identity online
```

`redis-config-identity-online.json` 只包含 service/host/port/DB/TLS 和认证模式，不包含密码。应用、应用运行容器、Redis service 配置和 Redis 运行容器的认证值必须在内存中逐字一致；证据额外绑定 Redis 容器 ID、image ID、`run_id` 哈希和 `master` 角色。

如果 `m195_bind_internal_redis_target` 因 `endpoint_kind=external` 或 `tls=true` 失败，发布在此停止。外部/TLS Redis 只有在另一个已审批变更提供 app-native maintenance 子命令后才能继续；该子命令必须复用应用的 TLS `ServerName` hostname verification，在同一已绑定连接上完成 server identity、prefix-scoped SCAN+UNLINK 和清理后复核，并输出同等非敏感合同证据。当前 Runbook 不允许手工替代。

## 6. 停止并排空全部 writer

先在边缘阻止新请求，再逐项停止和排空：

- 全部 HTTP、SSE、WebSocket gateway 实例和所有模型入口；
- 统一计费及 legacy/degraded post-billing 路径；
- 异步 `usage_logs` 写入、重试和补偿消费者；
- batch image/video reserve、capture、release 和最终结算 worker；
- 支付 webhook、订单履约、支付重试；
- 兑换码、默认会员分配、续费、过期重开和换套餐；
- admin reset、extend、revoke、restore；
- 订阅过期、窗口维护、缓存修复和其他后台任务；
- 能写 `user_subscriptions`、`billing_usage_entries`、`usage_logs` 的脚本、BI、cron、临时容器和旧主机。

外部 webhook 应在入口缓冲或返回可重试状态，不得在未落库时确认成功。

```bash
cd "$M195_DEPLOY_DIR"
# 从进入维护起冻结全部 Compose YAML 与 .env，并暂停所有不遵守 advisory
# lock 的配置管理 reconciler；这些锁保持到恢复流量后的最终复核。
m195_lock_compose_inputs
m195_assert_compose_input_locks
sha256sum --check "$M195_RUN_DIR/compose-inputs.before.sha256"
m195_compose stop -t 120 "$M195_APP_SERVICE"
docker inspect sub2api --format '{{.State.Status}} {{.State.FinishedAt}}' \
  > "$M195_RUN_DIR/old-app-stopped.txt"
```

不要停止 PostgreSQL 和 Redis。排空标准：在途请求结束、所有相关队列为零、Compose 外 writer 已确认停止、当前数据库无 prepared transaction、无 idle-in-transaction、无目标表 writer/waiting lock。

用数据库时钟记录准确 cutoff：

```bash
export M195_WRITERS_STOPPED_AT="$(
  psql -X --no-password --quiet --tuples-only --no-align \
    --dbname="service=${M195_DB_SERVICE}" \
    --command="SELECT TO_CHAR(clock_timestamp() AT TIME ZONE 'UTC', 'YYYY-MM-DD\"T\"HH24:MI:SS.US\"Z\"')"
)"
printf '%s\n' "$M195_WRITERS_STOPPED_AT" \
  > "$M195_RUN_DIR/writers-stopped-at.txt"

m195_capture_database_identities maintenance-before-backup
cmp -s \
  "$M195_RUN_DIR/database-identity-preflight-online.json" \
  "$M195_RUN_DIR/database-identity-preflight-maintenance-before-backup.json"
```

## 7. 备份与恢复点

Writer 停止后建立逻辑备份，并在基础设施侧确认一致性快照/PITR 恢复点已经完成：

```bash
cd "$M195_DEPLOY_DIR"
m195_compose exec -T "$M195_PG_SERVICE" sh -ec '
    pgpass_file="$(mktemp /tmp/luoxue-m195-pgpass.XXXXXX)"
    trap '\''rm -f "$pgpass_file"'\'' EXIT HUP INT TERM
    chmod 600 "$pgpass_file"
    pgpass_escape() {
      printf "%s" "$1" | sed -e "s/\\\\/\\\\\\\\/g" -e "s/:/\\\\:/g"
    }
    pgpass_db="$(pgpass_escape "$POSTGRES_DB")"
    pgpass_user="$(pgpass_escape "$POSTGRES_USER")"
    pgpass_password="$(pgpass_escape "$POSTGRES_PASSWORD")"
    printf "127.0.0.1:%s:%s:%s:%s\n" \
      "${PGPORT:-5432}" "$pgpass_db" "$pgpass_user" "$pgpass_password" \
      > "$pgpass_file"
    unset pgpass_password POSTGRES_PASSWORD

    PGPASSFILE="$pgpass_file" pg_dump \
      --format=custom --no-owner --no-acl \
      --host=127.0.0.1 --port="${PGPORT:-5432}" \
      --username="$POSTGRES_USER" --dbname="$POSTGRES_DB"
  ' > "$M195_RUN_DIR/pre-195.dump"

test -s "$M195_RUN_DIR/pre-195.dump"
sha256sum "$M195_RUN_DIR/pre-195.dump" \
  > "$M195_RUN_DIR/pre-195.dump.sha256"
m195_compose exec -T "$M195_PG_SERVICE" pg_restore --list \
  < "$M195_RUN_DIR/pre-195.dump" \
  > "$M195_RUN_DIR/pre-195.restore-list.txt"
```

放行条件：备份可列出、校验和已记录、PITR/快照已完成、恢复流程已在隔离环境演练、备份不与数据库共用单一故障域。

## 8. Maintenance preflight

只有确认所有 writer 已停止后才能使用 `--confirm-writers-stopped`：

```bash
cd "$M195_RELEASE_ROOT"
test "$(git rev-parse --verify HEAD)" = "$M195_RELEASE_COMMIT"
git diff --quiet
git diff --cached --quiet
test -z "$(git ls-files --others --exclude-standard)"
backend/scripts/preflight-subscription-anchored-monthly-quota.sh \
  --mode maintenance \
  --dbname "service=${M195_DB_SERVICE}" \
  --legacy-writers-stopped-at "$M195_WRITERS_STOPPED_AT" \
  --term-guard-hours 2 \
  --quiescence-seconds 60 \
  --confirm-writers-stopped \
  > "$M195_RUN_DIR/preflight-maintenance.json" \
  2> "$M195_RUN_DIR/preflight-maintenance.log"

jq -e --arg expected "$M195_CANDIDATE_195_SHA256" '
  .contract == "migration-195-preflight/v1"
  and .migration_sha256 == $expected
  and .safe_to_apply == true
  and .schema_ok == true
  and .read_only_verified == true
  and .writer_watermark_observation_stable == true
  and .writer_process_stop_acknowledged == true
  and .counts.database_prepared_transactions == "0"
  and .counts.database_writer_visibility == "1"
  and (.blockers | length) == 0
' "$M195_RUN_DIR/preflight-maintenance.json"
```

`writer_watermark_observation_stable=true` 只表示两次聚合 watermark 之间没有观察到变化，不是数据库写栅栏，也不能排除第二次观测后的 TOCTOU。只有外部 writer stop/queue drain 证据、`writer_process_stop_acknowledged=true`、数据库 lock/transaction 检查和该稳定观测同时成立才可放行。当前数据库任意 `pg_prepared_xacts` 都会以 `database_prepared_transactions` 阻断 maintenance。

若 `already_applied=true`，脚本会返回 `safe_to_apply=false`；不要再次迁移，直接调查为什么数据库先于变更单到达该状态，并完成 checksum/约束复核。

`--acknowledge-ambiguous-reopened-term-events` 只能在单独的受权逐笔对账完成并留有审计记录后使用。它不能降级 definite old-term evidence。

退出码：`0` 为盘点完成/门禁通过/已应用，`2` 参数错误，`3` 本地依赖或安全配置错误，`4` 连接认证/TLS 错误，`5` schema/checksum 错误，`10` 数据或 writer drain 阻断，`11` 查询/超时，`70` JSON 合同错误。始终同时检查 JSON，不能只看退出码。

## 9. 只执行 migration

创建只覆盖应用镜像的临时 Compose 文件，使 one-shot 容器配置固定到候选 registry digest、运行 image ID 固定到已验证的本地对象，同时继承生产 service 的网络、配置与挂载：

```bash
set -Eeuo pipefail
[[ "$M195_APP_SERVICE" =~ ^[A-Za-z0-9][A-Za-z0-9_.-]*$ ]]
[[ "$M195_CANDIDATE_IMAGE_ID" =~ ^sha256:[0-9a-f]{64}$ ]]
m195_assert_digest_ref "$M195_CANDIDATE_DIGEST_REF"
sha256sum --check "$M195_RUN_DIR/candidate-binding.sha256"
test "$(jq -er '.run_id' "$M195_CANDIDATE_BINDING_FILE")" = "$M195_RUN_ID"
test "$(jq -er '.release_commit' "$M195_CANDIDATE_BINDING_FILE")" \
  = "$M195_RELEASE_COMMIT"
test "$(jq -er '.candidate_digest_ref' "$M195_CANDIDATE_BINDING_FILE")" \
  = "$M195_CANDIDATE_DIGEST_REF"
test "$(jq -er '.candidate_image_id' "$M195_CANDIDATE_BINDING_FILE")" \
  = "$M195_CANDIDATE_IMAGE_ID"
test "$(jq -er '.migration_195_sha256' "$M195_CANDIDATE_BINDING_FILE")" \
  = "$M195_CANDIDATE_195_SHA256"

docker pull "$M195_CANDIDATE_DIGEST_REF" \
  > "$M195_RUN_DIR/candidate-image-repull-before-migrate.log" 2>&1
docker image inspect "$M195_CANDIDATE_DIGEST_REF" \
  --format '{{.Id}} {{json .RepoDigests}}' \
  > "$M195_RUN_DIR/candidate-image-repull-before-migrate.txt"
test "$(docker image inspect "$M195_CANDIDATE_DIGEST_REF" --format '{{.Id}}')" \
  = "$M195_CANDIDATE_IMAGE_ID"
docker image inspect "$M195_CANDIDATE_DIGEST_REF" \
  --format '{{json .RepoDigests}}' \
  | jq -e --arg ref "$M195_CANDIDATE_DIGEST_REF" 'index($ref) != null'
test -r "$M195_IMAGE_OVERRIDE"
cd "$M195_DEPLOY_DIR"
m195_compose -f "$M195_IMAGE_OVERRIDE" config --quiet
m195_compose -f "$M195_IMAGE_OVERRIDE" config --format json \
  | jq -er --arg service "$M195_APP_SERVICE" \
      '.services[$service].image // empty' \
  | grep -Fx "$M195_CANDIDATE_DIGEST_REF"

m195_capture_database_identities maintenance-before-migrate
cmp -s \
  "$M195_RUN_DIR/database-identity-preflight-online.json" \
  "$M195_RUN_DIR/database-identity-preflight-maintenance-before-migrate.json"

export M195_EXPECTED_DATABASE_IDENTITY_FILE="$M195_RUN_DIR/database-identity-preflight-maintenance-before-migrate.json"
export M195_EXPECTED_DATABASE_IDENTITY_CONTAINER_FILE="/run/m195/expected-database-identity.json"
m195_validate_database_identity_file "$M195_EXPECTED_DATABASE_IDENTITY_FILE"
test "$(realpath -e -- "$M195_EXPECTED_DATABASE_IDENTITY_FILE")" \
  = "$(realpath -e -- "$M195_RUN_DIR")/database-identity-preflight-maintenance-before-migrate.json"
chmod 400 "$M195_EXPECTED_DATABASE_IDENTITY_FILE"
```

执行 one-shot。它不发布端口，不启动依赖服务，不启动应用 worker：

```bash
set -Eeuo pipefail
export M195_MIGRATION_CONTAINER="luoxue-m195-${M195_RUN_ID}"
if docker container inspect "$M195_MIGRATION_CONTAINER" >/dev/null 2>&1; then
  printf 'Refusing to reuse migration container: %s\n' \
    "$M195_MIGRATION_CONTAINER" >&2
  exit 1
fi

set +e
m195_compose -f "$M195_IMAGE_OVERRIDE" run -T --no-deps --pull never \
  --name "$M195_MIGRATION_CONTAINER" \
  --entrypoint /app/sub2api \
  --volume "$M195_EXPECTED_DATABASE_IDENTITY_FILE:$M195_EXPECTED_DATABASE_IDENTITY_CONTAINER_FILE:ro" \
  "$M195_APP_SERVICE" \
  --migrate-only \
  --expected-database-identity-file "$M195_EXPECTED_DATABASE_IDENTITY_CONTAINER_FILE" \
  > "$M195_RUN_DIR/migrate-only.log" 2>&1
migrate_status=$?
set -e
printf '%s\n' "$migrate_status" > "$M195_RUN_DIR/migrate-only.exit-code.txt"
test "$migrate_status" = "0"

test "$(docker inspect "$M195_MIGRATION_CONTAINER" --format '{{.Image}}')" \
  = "$M195_CANDIDATE_IMAGE_ID"
test "$(docker inspect "$M195_MIGRATION_CONTAINER" --format '{{.State.ExitCode}}')" \
  = "0"
```

非零退出立即停止。不要启动应用，不要手工向 `schema_migrations` 插行，也不要绕过 checksum。

## 10. 提交后复检

保持所有业务 writer 停止，再次运行 maintenance preflight。预期：

```bash
cd "$M195_RELEASE_ROOT"
test "$(git rev-parse --verify HEAD)" = "$M195_RELEASE_COMMIT"
git diff --quiet
git diff --cached --quiet
test -z "$(git ls-files --others --exclude-standard)"
m195_capture_database_identities maintenance-after-migrate
cmp -s \
  "$M195_RUN_DIR/database-identity-preflight-online.json" \
  "$M195_RUN_DIR/database-identity-preflight-maintenance-after-migrate.json"

backend/scripts/preflight-subscription-anchored-monthly-quota.sh \
  --mode maintenance \
  --dbname "service=${M195_DB_SERVICE}" \
  --legacy-writers-stopped-at "$M195_WRITERS_STOPPED_AT" \
  --term-guard-hours 2 \
  --quiescence-seconds 60 \
  --confirm-writers-stopped \
  > "$M195_RUN_DIR/preflight-post-migration.json" \
  2> "$M195_RUN_DIR/preflight-post-migration.log"

jq -e --arg expected "$M195_CANDIDATE_195_SHA256" '
  .contract == "migration-195-preflight/v1"
  and .migration_sha256 == $expected
  and .already_applied == true
  and .schema_ok == true
  and .read_only_verified == true
  and .writer_watermark_observation_stable == true
  and .writer_process_stop_acknowledged == true
  and .counts.database_prepared_transactions == "0"
  and (.blockers | length) == 0
' "$M195_RUN_DIR/preflight-post-migration.json"
```

该次结果的 `safe_to_apply=false` 是正确行为，因为 migration 已应用。额外确认五个金额/窗口约束存在且已验证：

```sql
SELECT conname, convalidated
FROM pg_constraint
WHERE conrelid IN (
    'public.groups'::regclass,
    'public.user_subscriptions'::regclass,
    'public.billing_usage_entries'::regclass
)
  AND conname IN (
    'billing_usage_entries_subscription_amount_check',
    'groups_subscription_quota_limits_check',
    'user_subscriptions_usage_amounts_check',
    'user_subscriptions_weekly_window_anchored_check',
    'user_subscriptions_monthly_window_anchored_check'
)
ORDER BY conname;
```

必须恰好五行且全部 `convalidated=true`。

## 11. 精确失效会员缓存

全部应用实例仍停止时，只删除应用真实 Redis DB 中、且精确符合代码生成格式 `billing:sub:<positive-user-id>:<positive-group-id>` 的 hash。禁止 `KEYS`、`FLUSHDB`、`FLUSHALL`，也不得删除其他前缀。先重新派生配置并确认 Redis 网络与 server identity 均和在线盘点逐字相等：

```bash
cd "$M195_DEPLOY_DIR"
m195_assert_compose_input_locks
sha256sum --check "$M195_RUN_DIR/compose-inputs.before.sha256"
test -z "$(m195_compose ps --status running --quiet "$M195_APP_SERVICE")"
m195_capture_redis_network_identity maintenance-before-cache-clear
m195_capture_redis_server_identity maintenance-before-cache-clear
cmp -s \
  "$M195_RUN_DIR/redis-config-identity-online.json" \
  "$M195_RUN_DIR/redis-config-identity-maintenance-before-cache-clear.json"
cmp -s \
  "$M195_RUN_DIR/redis-network-identity-online.json" \
  "$M195_RUN_DIR/redis-network-identity-maintenance-before-cache-clear.json"
cmp -s \
  "$M195_RUN_DIR/redis-server-identity-online.json" \
  "$M195_RUN_DIR/redis-server-identity-maintenance-before-cache-clear.json"

m195_clear_subscription_cache() {
  local current_container_id result
  current_container_id="$(m195_compose ps --all --quiet "$M195_REDIS_SERVICE")"
  test "$current_container_id" = "$M195_REDIS_CONTAINER_ID"
  m195_assert_internal_redis_auth_config
  result="$(
    docker exec \
      -e M195_REDIS_PORT="$M195_REDIS_PORT" \
      -e M195_REDIS_DB="$M195_REDIS_DB" \
      "$M195_REDIS_CONTAINER_ID" sh -eu -c '
      case "$M195_REDIS_PORT" in ""|*[!0-9]*) exit 21 ;; esac
      case "$M195_REDIS_DB" in ""|*[!0-9]*) exit 22 ;; esac
      test "${REDISCLI_AUTH+x}" = x
      redis_call() {
        redis-cli -e --raw -h 127.0.0.1 -p "$M195_REDIS_PORT" \
          -n "$M195_REDIS_DB" "$@"
      }

      test "$(redis_call PING)" = PONG
      client_info="$(redis_call CLIENT INFO)"
      selected_db=""
      for client_field in $client_info; do
        case "$client_field" in db=*) selected_db="${client_field#db=}" ;; esac
      done
      test "$selected_db" = "$M195_REDIS_DB"
      unset client_field client_info

      subscription_batch_script="
local result = redis.call(\"SCAN\", ARGV[1], \"MATCH\", \"billing:sub:*\", \"COUNT\", ARGV[2])
local cursor = result[1]
local keys = result[2]
local mode = ARGV[3]
if mode ~= \"validate\" and mode ~= \"unlink\" then
  return redis.error_reply(\"invalid cache maintenance mode\")
end
for _, key in ipairs(keys) do
  if not string.match(key, \"^billing:sub:[1-9][0-9]*:[1-9][0-9]*$\") then
    return redis.error_reply(\"unexpected subscription cache key shape\")
  end
  local type_reply = redis.call(\"TYPE\", key)
  local key_type = type(type_reply) == \"table\" and type_reply[\"ok\"] or type_reply
  if key_type ~= \"none\" then
    if key_type ~= \"hash\" then
      return redis.error_reply(\"unexpected subscription cache value type\")
    end
    local schema = redis.call(\"HGET\", key, \"schema_version\")
    if not schema or not string.match(schema, \"^[1-9][0-9]*$\") then
      return redis.error_reply(\"invalid subscription cache schema\")
    end
    local ttl = redis.call(\"PTTL\", key)
    if ttl == -1 or ttl < -2 then
      return redis.error_reply(\"subscription cache key lacks a valid boundary TTL\")
    end
  end
end
local unlinked = 0
if mode == \"unlink\" then
  for _, key in ipairs(keys) do
    unlinked = unlinked + redis.call(\"UNLINK\", key)
  end
end
return cursor .. \":\" .. tostring(#keys) .. \":\" .. tostring(unlinked)
"

      scan_subscription_cache() {
        mode="$1"
        case "$mode" in validate|unlink) ;; *) return 40 ;; esac
        cursor=0
        scan_matched=0
        scan_unlinked=0
        scan_batches=0
        while :; do
          scan_batches=$((scan_batches + 1))
          test "$scan_batches" -le 1000000
          batch_result="$(
            redis_call EVAL "$subscription_batch_script" 0 \
              "$cursor" 500 "$mode"
          )"
          IFS=: read -r next_cursor batch_matched batch_unlinked extra <<EOF
$batch_result
EOF
          test -z "${extra:-}"
          case "$next_cursor" in ""|*[!0-9]*) return 41 ;; esac
          case "$batch_matched" in ""|*[!0-9]*) return 42 ;; esac
          case "$batch_unlinked" in ""|*[!0-9]*) return 43 ;; esac
          scan_matched=$((scan_matched + batch_matched))
          scan_unlinked=$((scan_unlinked + batch_unlinked))
          cursor="$next_cursor"
          test "$cursor" = 0 && break
        done
        printf "%s:%s:%s\n" \
          "$scan_matched" "$scan_unlinked" "$scan_batches"
      }

      dbsize_before="$(redis_call DBSIZE)"
      case "$dbsize_before" in ""|*[!0-9]*) exit 44 ;; esac
      validation_result="$(scan_subscription_cache validate)"
      IFS=: read -r validated_count validation_unlinked validation_batches extra <<EOF
$validation_result
EOF
      test -z "${extra:-}"
      test "$validation_unlinked" = 0

      matched_count=0
      unlinked_count=0
      total_batches="$validation_batches"
      sweeps=0
      remaining_count=-1
      while [ "$sweeps" -lt 8 ]; do
        sweeps=$((sweeps + 1))
        sweep_result="$(scan_subscription_cache unlink)"
        IFS=: read -r sweep_matched sweep_unlinked sweep_batches extra <<EOF
$sweep_result
EOF
        test -z "${extra:-}"
        matched_count=$((matched_count + sweep_matched))
        unlinked_count=$((unlinked_count + sweep_unlinked))
        total_batches=$((total_batches + sweep_batches))
        if [ "$sweep_matched" = 0 ]; then
          remaining_count=0
          break
        fi
      done
      test "$remaining_count" = 0
      dbsize_after="$(redis_call DBSIZE)"
      case "$dbsize_after" in ""|*[!0-9]*) exit 45 ;; esac
      printf "%s:%s:%s:%s:%s:%s:%s:%s:%s\n" \
        "$selected_db" "$validated_count" "$matched_count" \
        "$unlinked_count" "$remaining_count" "$dbsize_before" \
        "$dbsize_after" "$sweeps" "$total_batches"
    '
  )" || return 1
  test "$(m195_compose ps --all --quiet "$M195_REDIS_SERVICE")" \
    = "$M195_REDIS_CONTAINER_ID"
  printf '%s\n' "$result"
}

cache_clear_result="$(m195_clear_subscription_cache)"
IFS=: read -r selected_db validated_count matched_count unlinked_count \
  remaining_count dbsize_before dbsize_after sweeps scan_batches extra \
  <<< "$cache_clear_result"
test -z "${extra:-}"
test "$selected_db" = "$M195_REDIS_DB"
[[ "$validated_count" =~ ^[0-9]+$ ]]
[[ "$matched_count" =~ ^[0-9]+$ ]]
[[ "$unlinked_count" =~ ^[0-9]+$ ]]
test "$remaining_count" = "0"
jq -cS -n \
  --argjson selected_db "$selected_db" \
  --argjson validated "$validated_count" \
  --argjson matched "$matched_count" \
  --argjson unlinked "$unlinked_count" \
  --argjson remaining "$remaining_count" \
  --argjson dbsize_before "$dbsize_before" \
  --argjson dbsize_after "$dbsize_after" \
  --argjson sweeps "$sweeps" \
  --argjson scan_batches "$scan_batches" \
  '{
    contract: "migration-195-redis-cache-invalidation/v1",
    prefix: "billing:sub:*",
    accepted_key_shape: "billing:sub:<positive-user-id>:<positive-group-id>",
    selected_db: $selected_db,
    validated_before_unlink: $validated,
    matched: $matched,
    unlinked: $unlinked,
    remaining: $remaining,
    dbsize_before: $dbsize_before,
    dbsize_after: $dbsize_after,
    sweeps: $sweeps,
    scan_batches: $scan_batches
  }' > "$M195_RUN_DIR/redis-cache-invalidation.json"
unset cache_clear_result selected_db validated_count matched_count unlinked_count
unset remaining_count dbsize_before dbsize_after sweeps scan_batches

m195_capture_redis_network_identity maintenance-after-cache-clear
m195_capture_redis_server_identity maintenance-after-cache-clear
cmp -s \
  "$M195_RUN_DIR/redis-config-identity-online.json" \
  "$M195_RUN_DIR/redis-config-identity-maintenance-after-cache-clear.json"
cmp -s \
  "$M195_RUN_DIR/redis-network-identity-online.json" \
  "$M195_RUN_DIR/redis-network-identity-maintenance-after-cache-clear.json"
cmp -s \
  "$M195_RUN_DIR/redis-server-identity-online.json" \
  "$M195_RUN_DIR/redis-server-identity-maintenance-after-cache-clear.json"
```

清理命令使用 `docker exec` 直接绑定已审计的不可变 Redis container ID，并只连接其 loopback；它不按 service 名二次解析容器，也不接触应用的其他环境、secret 或数据卷。键名始终留在 Redis 进程内，分批 Lua 先完成全量只读验证，之后才允许 `UNLINK`；任一非正整数 ID、非 hash、缺失 schema、无 TTL 的同前缀 key 都阻断而不是删除。最后必须出现一次从 cursor 0 开始、完整结束且匹配数为 0 的 clean sweep。任一容器 ID/network ID/server identity 变化、所选 DB 漂移、认证/`INFO`/`ROLE` 失败、非 `master`、SCAN/UNLINK 异常或 clean sweep 不为零都保持维护并阻断 canary。外部/TLS Redis 没有手工例外，证据不记录 key 名或密码。

## 12. 新版本 canary、持久提升与恢复流量

使用同一候选 image override 只启动一个新版本应用实例，入口继续保持维护状态：

```bash
cd "$M195_DEPLOY_DIR"
m195_compose -f "$M195_IMAGE_OVERRIDE" \
  up -d --no-deps --force-recreate --no-build --pull never \
  "$M195_APP_SERVICE"
test "$(docker inspect sub2api --format '{{.Image}}')" \
  = "$M195_CANDIDATE_IMAGE_ID"
test "$(docker inspect sub2api --format '{{.Config.Image}}')" \
  = "$M195_CANDIDATE_DIGEST_REF"
```

必须验证：

- 容器 image ID 与候选一致、restart count 为 0、`/health` 与 `/readyz` 正常；
- 新会员在创建时立刻写入准确 `starts_at`、weekly/monthly anchor；
- 7 天与 30 天边界均为半开区间，非零点、跨月、跨年、DST 不改变固定秒数；
- 未中断续费不改 anchor，过期重开和换套餐建立准确新 anchor；
- 显式月额度优先，未设置时服务端使用周额度乘 4，显式 `0` 不得当成未设置；
- member key 耗尽且余额充足时 member 仍 blocked，balance key 可 usable，不发生自动余额回退；
- Redis/DB/快照不一致或查询失败时返回 `stale/unknown`，不渲染为 0；
- 受控会员缓存的 Redis `PTTL` 不晚于权威 weekly reset、monthly reset、会员 expiry 三者中的最早边界；跨过任一边界后旧 key 已过期，重新查询从 DB 建立新窗口，不能沿用旧 usage；
- viewer token 的 audience、`quota:read` scope、过期和设备撤销均按合同生效；
- overview/progress DTO 金额为十进制定点字符串、ID 为字符串，不含完整 API Key、上游额度或无关 PII；
- 一笔受控会员请求产生 `billing_type=1` 且非空 `subscription_amount` 的 receipt；
- 首次失败不产生 0 余额或 0% 假数据，旧快照正确携带 `as_of`、`fresh_until`、`stale`。

Canary 通过后，入口仍保持维护状态。临时 override 不能成为生产部署状态；必须把候选 registry digest 持久写入当前 `local.yml + custom.yml` 正常链中的最终 override。以下结构化更新只修改 `services.<app>.image`，并以更新前 checksum 防止覆盖并发配置变更：

```bash
set -Eeuo pipefail
[[ "$M195_CANDIDATE_IMAGE_ID" =~ ^sha256:[0-9a-f]{64}$ ]]
m195_assert_digest_ref "$M195_CANDIDATE_DIGEST_REF"
m195_assert_digest_ref "$M195_ROLLBACK_DIGEST_REF"
sha256sum --check "$M195_RUN_DIR/candidate-binding.sha256"
test "$(jq -er '.candidate_digest_ref' "$M195_CANDIDATE_BINDING_FILE")" \
  = "$M195_CANDIDATE_DIGEST_REF"
test "$(jq -er '.candidate_image_id' "$M195_CANDIDATE_BINDING_FILE")" \
  = "$M195_CANDIDATE_IMAGE_ID"
docker pull "$M195_CANDIDATE_DIGEST_REF" \
  > "$M195_RUN_DIR/candidate-image-repull-before-promote.log" 2>&1
docker image inspect "$M195_CANDIDATE_DIGEST_REF" \
  --format '{{.Id}} {{json .RepoDigests}}' \
  > "$M195_RUN_DIR/candidate-image-repull-before-promote.txt"
test "$(docker image inspect "$M195_CANDIDATE_DIGEST_REF" --format '{{.Id}}')" \
  = "$M195_CANDIDATE_IMAGE_ID"
docker image inspect "$M195_CANDIDATE_DIGEST_REF" \
  --format '{{json .RepoDigests}}' \
  | jq -e --arg ref "$M195_CANDIDATE_DIGEST_REF" 'index($ref) != null'
grep -Fx "$M195_IMAGE_PIN_FILE" \
  "$M195_RUN_DIR/compose-config-files-resolved.txt"

# 确认进入维护时取得的全部 Compose/.env lock 仍由当前 shell 持有。
m195_assert_compose_input_locks

test "$(docker inspect sub2api --format '{{.Image}}')" \
  = "$M195_CANDIDATE_IMAGE_ID"
test "$(docker inspect sub2api --format '{{.Config.Image}}')" \
  = "$M195_CANDIDATE_DIGEST_REF"

export M195_SENSITIVE_IMAGE_CONFIG_BACKUP="${M195_IMAGE_PIN_FILE}.m195-${M195_RUN_ID}.bak"
if m195_image_pin_state_matches candidate; then
  test -f "$M195_SENSITIVE_IMAGE_CONFIG_BACKUP"
elif m195_image_pin_state_matches before; then
  if ! test -e "$M195_SENSITIVE_IMAGE_CONFIG_BACKUP"; then
    install -m 600 "$M195_IMAGE_PIN_FILE" \
      "$M195_SENSITIVE_IMAGE_CONFIG_BACKUP"
  fi
  m195_transition_image_pin candidate "$M195_CANDIDATE_DIGEST_REF"
else
  printf '%s\n' \
    'Persistent image config matches neither before nor candidate journal.' >&2
  exit 1
fi
test "$(sha256sum "$M195_SENSITIVE_IMAGE_CONFIG_BACKUP" | awk '{print $1}')" \
  = "$(awk 'NR == 1 {print $1}' \
      "$M195_RUN_DIR/persistent-image-config.before.sha256")"

m195_assert_persistent_candidate_pin() {
  m195_assert_compose_input_locks
  m195_image_pin_state_matches candidate
  m195_compose config --quiet
  m195_compose config --format json \
    | jq -er --arg service "$M195_APP_SERVICE" \
        '.services[$service].image // empty' \
    | grep -Fx "$M195_CANDIDATE_DIGEST_REF"
}

cd "$M195_DEPLOY_DIR"
m195_assert_persistent_candidate_pin
printf '%s\n' "$M195_CANDIDATE_DIGEST_REF" \
  | tee "$M195_RUN_DIR/persistent-production-image-ref.txt" \
  | grep -Fx "$M195_CANDIDATE_DIGEST_REF" >/dev/null
m195_compose up -d --no-deps --force-recreate --no-build --pull never \
  "$M195_APP_SERVICE"

m195_assert_persistent_candidate_pin
test "$(docker inspect sub2api --format '{{.Image}}')" \
  = "$M195_CANDIDATE_IMAGE_ID"
test "$(docker inspect sub2api --format '{{.Config.Image}}')" \
  = "$M195_CANDIDATE_DIGEST_REF"
m195_assert_container_compose_identity persistent-recreate

# 验证普通 Compose restart 仍使用同一 digest；该命令不带临时 override。
m195_compose restart -t 120 "$M195_APP_SERVICE"
for attempt in $(seq 1 90); do
  health_status="$(
    docker inspect sub2api --format '{{.State.Health.Status}}' 2>/dev/null \
      || true
  )"
  case "$health_status" in
    healthy) break ;;
    unhealthy) break ;;
  esac
  sleep 2
done
test "$health_status" = "healthy"
test "$(docker inspect sub2api --format '{{.Image}}')" \
  = "$M195_CANDIDATE_IMAGE_ID"
test "$(docker inspect sub2api --format '{{.Config.Image}}')" \
  = "$M195_CANDIDATE_DIGEST_REF"
m195_assert_container_compose_identity persistent-restart
m195_assert_persistent_candidate_pin
docker inspect sub2api \
  --format '{{.Image}} {{.Config.Image}} {{.RestartCount}} {{.State.Health.Status}}' \
  > "$M195_RUN_DIR/promoted-production-container.txt"
```

若 `docker-compose.custom.yml` 由 Ansible、GitOps、控制面或其他配置管理生成，直接修改服务器文件不构成持久提升：必须先暂停 reconciler，在同一冻结窗口内把同一 digest 写入权威源、取得变更 ID、重新物化文件；若物化后的合法字节变化，重新生成 `persistent-image-config.candidate.sha256`，再完整重跑 normal-chain recreate/restart 验证。禁止把 mutable tag 写回 `.env`，也禁止用本地 `docker tag` 代替该步骤。

`M195_SENSITIVE_IMAGE_CONFIG_BACKUP` 可能含 secret，只能留在生产部署目录、保持 `0600` 并按凭据材料管理，不得复制到 `$M195_RUN_DIR` 或附入变更单。它只用于人工恢复配置文件损坏；镜像版本回滚仍必须走下面的结构化 digest pin，不能用整文件覆盖吞掉并发配置变更。

对脱离 override 后的生产容器重新执行 `/health`、`/readyz` 和上述关键业务验收。所有配置 writer/reconciler 保持冻结，锁 FD 不得关闭；解除入口维护前最后一个命令必须再次执行 `m195_assert_persistent_candidate_pin`，随后记录恢复流量时间，才可释放文件锁和恢复配置管理。只有 registry digest、运行 image ID、正常 Compose 文件链、recreate、restart 和权威配置源全部一致，才解除入口维护并逐步恢复流量。恢复后继续观察错误率、unknown/stale 比例、会员 blocked 原因和 receipt 缺失情况。

## 13. 回滚边界

镜像回滚同样必须使用已验证的 registry digest 并持久更新正常 Compose 链，不能把旧 image ID 本地 retag 成 mutable ref。调用前同样暂停所有配置 writer/reconciler 并保持到回滚验收结束；以下函数只能在对应数据库回滚边界满足后调用。若当前配置 checksum 既不是候选 pin、也不是原始配置，说明存在并发修改，立即停止而不是覆盖：

```bash
m195_pin_verified_rollback_image() {
  set -Eeuo pipefail
  m195_assert_digest_ref "$M195_ROLLBACK_DIGEST_REF"
  sha256sum --check "$M195_RUN_DIR/candidate-binding.sha256"
  test "$(jq -er '.rollback_digest_ref' "$M195_CANDIDATE_BINDING_FILE")" \
    = "$M195_ROLLBACK_DIGEST_REF"
  test "$(jq -er '.rollback_image_id' "$M195_CANDIDATE_BINDING_FILE")" \
    = "$M195_ROLLBACK_IMAGE_ID"
  if test "${#M195_COMPOSE_INPUT_LOCK_FDS[@]}" = "0"; then
    m195_lock_compose_inputs
  fi
  m195_assert_compose_input_locks
  docker pull "$M195_ROLLBACK_DIGEST_REF" \
    > "$M195_RUN_DIR/rollback-image-repull.log" 2>&1
  docker image inspect "$M195_ROLLBACK_DIGEST_REF" \
    --format '{{.Id}} {{json .RepoDigests}}' \
    > "$M195_RUN_DIR/rollback-image-repull.txt"
  test "$(docker image inspect "$M195_ROLLBACK_DIGEST_REF" --format '{{.Id}}')" \
    = "$M195_ROLLBACK_IMAGE_ID"
  docker image inspect "$M195_ROLLBACK_DIGEST_REF" \
    --format '{{json .RepoDigests}}' \
    | jq -e --arg ref "$M195_ROLLBACK_DIGEST_REF" 'index($ref) != null'

  if m195_image_pin_state_matches rollback; then
    :
  elif m195_image_pin_state_matches candidate \
    || m195_image_pin_state_matches before; then
    m195_transition_image_pin rollback "$M195_ROLLBACK_DIGEST_REF"
  else
    printf '%s\n' \
      'Persistent image config matches no reviewed transition journal.' >&2
    return 1
  fi
  m195_image_pin_state_matches rollback

  m195_compose config --quiet
  m195_compose config --format json \
    | jq -er --arg service "$M195_APP_SERVICE" \
        '.services[$service].image // empty' \
    | grep -Fx "$M195_ROLLBACK_DIGEST_REF"
  m195_compose up -d --no-deps --force-recreate --no-build --pull never \
    "$M195_APP_SERVICE"
  m195_assert_compose_input_locks
  sha256sum --check "$M195_RUN_DIR/compose-inputs.rollback.sha256"
  sha256sum --check \
    "$M195_RUN_DIR/persistent-image-config.rollback.sha256"
  m195_compose config --format json \
    | jq -er --arg service "$M195_APP_SERVICE" \
        '.services[$service].image // empty' \
    | grep -Fx "$M195_ROLLBACK_DIGEST_REF"
  test "$(docker inspect sub2api --format '{{.Image}}')" \
    = "$M195_ROLLBACK_IMAGE_ID"
  test "$(docker inspect sub2api --format '{{.Config.Image}}')" \
    = "$M195_ROLLBACK_DIGEST_REF"
  m195_assert_container_compose_identity rollback-recreate

  # 证明脱离临时 override 后，正常 Compose restart 仍使用持久 rollback digest。
  m195_compose restart -t 120 "$M195_APP_SERVICE"
  rollback_health_status=""
  for attempt in $(seq 1 90); do
    rollback_health_status="$(
      docker inspect sub2api --format '{{.State.Health.Status}}' 2>/dev/null \
        || true
    )"
    case "$rollback_health_status" in
      healthy) break ;;
      unhealthy) break ;;
    esac
    sleep 2
  done
  test "$rollback_health_status" = "healthy"
  m195_image_pin_state_matches rollback
  test "$(docker inspect sub2api --format '{{.Image}}')" \
    = "$M195_ROLLBACK_IMAGE_ID"
  test "$(docker inspect sub2api --format '{{.Config.Image}}')" \
    = "$M195_ROLLBACK_DIGEST_REF"
  m195_assert_container_compose_identity rollback-restart
  docker inspect sub2api \
    --format '{{.Image}} {{.Config.Image}} {{.RestartCount}} {{.State.Health.Status}}' \
    > "$M195_RUN_DIR/rollback-production-container.txt"
}
```

### 迁移事务提交前

Migration 失败会整体回滚。保持维护状态，确认 195 未记录、schema 未部分变化后，可以修复数据并重试；若决定取消发布，执行 `m195_pin_verified_rollback_image`，等待旧版 `/health`、`/readyz` 和正常 Compose restart 验证后再恢复流量。不得只根据容器退出码推断数据库回滚，也不得依赖本地 rollback tag。

### 迁移已提交、尚未开放新写流量

旧二进制业务不兼容，不能只切回旧镜像。首选保持维护并向前修复。确需回到旧版时，只能在 writer 持续停止的条件下先恢复并验证 migration 前数据库快照/PITR，确认 195 不存在且数据库身份正确，再执行 `m195_pin_verified_rollback_image`；顺序不得颠倒，并接受恢复点之后数据被回退的影响。

### 新版本已产生写入

默认只能向前修复。直接恢复旧镜像会重新引入旧窗口/计费语义；直接回数据库快照会丢失新写入。只有经过事故指挥、数据损失评估和完整恢复方案批准后，才能先恢复数据库、再持久 pin `M195_ROLLBACK_DIGEST_REF` 并执行灾难恢复。

## 14. 发布前测试门槛

代码侧至少通过：

```bash
cd backend
go test ./migrations \
  -run '^TestSubscriptionAnchoredMonthlyQuotaMigration' -count=1
CI=1 go test -tags=integration ./internal/repository \
  -run '^TestMigration195' -count=1
CI=1 go test -tags=integration ./internal/repository \
  -run '^TestMigration195Preflight' -count=1
go test ./internal/repository \
  -run '^(TestApplyConfiguredMigrationsRejects|TestMigration195PreflightSQL|TestRejectPendingMaintenanceOnlyMigrations|TestQueryDatabaseIdentity)' -count=1
go test ./internal/repository \
  -run '^TestSubscriptionCacheTTLCannotCrossResetOrExpiry$' -count=1
go test ./cmd/server -count=1
```

PostgreSQL 默认常为 `max_prepared_transactions=0`，真实 2PC 集成用例会在该环境明确 skip；无论是否启用 2PC，静态门禁测试都必须通过。生产预检仍查询 `pg_prepared_xacts`，不得因为测试容器禁用 2PC 而移除该检查。

并按 `quota-viewer/docs/QUOTA_OVERVIEW_API.zh-CN.md` 第 7、8 节完成合同/安全矩阵，重点覆盖：错误 audience/scope/过期/撤销设备、用户隔离、无完整 Key、7d/30d 边界、续费/重开/换套餐、显式/回退月额度、member 与 balance 独立状态、缓存不一致、首次失败、stale 快照和 `period_usage=unknown`。

## 15. 完成证据

变更单至少附：

- 全新 run ID/evidence context、候选绑定合同及 checksum；不得附旧阶段文件冒充本次证据；
- release SHA-256、无 AppleDouble 证明、候选/回滚 registry digest、首次及迁移前/提升前/回滚前重新拉取证明、精确 RepoDigest 与对应本地 image ID；
- 实际 Compose project/解析后的完整文件链、候选/checkout migration manifest、构建 provenance 和数据库 pending migration 清单；
- 持久镜像配置文件路径、变更前/候选 pin checksum、配置管理变更 ID、正常 Compose recreate/restart 后的 digest 与容器 image ID；
- online、maintenance-before-backup、maintenance-before-migrate、maintenance-after-migrate 四阶段中，预检/候选配置/备份目标逐字相等且 `in_recovery=false` 的 v2 数据库身份 JSON，以及 one-shot 只读挂载 expected 文件的命令证据；
- online、maintenance、post-migration 三份预检 JSON 及校验和；
- 数据库备份校验和、PITR/快照 ID、恢复演练记录；
- migrate-only 容器 image ID、独立退出码文件和日志；
- Redis Compose 配置身份、应用/Redis 共享网络及 alias 证明、目标 Redis 容器/image ID、selected DB、run_id 哈希、online/清理前/清理后 server identity、keyspace 大小与精确键格式清理计数；
- canary 验收结果、持久生产镜像引用、脱离临时 override 后的容器 image ID/Compose 标签、恢复流量时间与观察结论；
- 明确保留的 registry rollback digest、本地便利副本和清理计划。
