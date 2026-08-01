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

- `--migrate-only` 只加载配置、连接 PostgreSQL、调用嵌入式 migration runner 后退出；它不初始化 Redis、HTTP、worker、系统密钥或 simple-mode seed。
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
export M195_RUN_DIR="/var/tmp/luoxue-m195-${M195_RUN_ID}"
export M195_RELEASE_ROOT="/path/to/verified/release"
export M195_DEPLOY_DIR="/root/luoxueapi/deploy"
export M195_DB_SERVICE="approved-libpq-service"
export M195_APP_SERVICE="sub2api"
export M195_PG_SERVICE="postgres"
export M195_REDIS_SERVICE="redis"
export M195_REDIS_DB="0"
install -d -m 700 "$M195_RUN_DIR"
```

数据库认证使用权限为 `0600` 的 `PGPASSFILE` 或受管 secret provider。不要把密码放入 `--dbname`、DSN、命令行、shell history 或证据文件。

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

从当前生产容器标签读取真实 Compose project、工作目录和文件链：

```bash
docker inspect sub2api \
  --format '{{ index .Config.Labels "com.docker.compose.project" }}' \
  > "$M195_RUN_DIR/compose-project.txt"
docker inspect sub2api \
  --format '{{ index .Config.Labels "com.docker.compose.project.working_dir" }}' \
  > "$M195_RUN_DIR/compose-working-dir.txt"
docker inspect sub2api \
  --format '{{ index .Config.Labels "com.docker.compose.project.config_files" }}' \
  > "$M195_RUN_DIR/compose-config-files.txt"
```

后续所有 Compose 命令必须复用标签中记录的 project 与全部配置文件。以下仅展示已知双文件形式：

```bash
cd "$M195_DEPLOY_DIR"
test -r docker-compose.local.yml
test -r docker-compose.custom.yml
docker compose \
  -f docker-compose.local.yml \
  -f docker-compose.custom.yml \
  config --quiet
docker compose \
  -f docker-compose.local.yml \
  -f docker-compose.custom.yml \
  config --services > "$M195_RUN_DIR/compose-services.txt"
docker compose \
  -f docker-compose.local.yml \
  -f docker-compose.custom.yml \
  config --images > "$M195_RUN_DIR/compose-images.txt"
```

不要保存完整 `docker compose config` 输出，它可能展开 secret。

## 4. 预检数据库角色

建议使用独立、短期、只读的预检账号。最小权限为：

- 目标数据库 `CONNECT`；
- `public` schema `USAGE`；
- `groups`、`user_subscriptions`、`billing_usage_entries`、`usage_logs`、`schema_migrations` 的 `SELECT`；
- `pg_read_all_stats` 成员资格，或等价的受控监控角色，以完整观察其他角色的 writer/transaction。

不需要读取 `users`、`accounts`、`api_keys`。`pg_read_all_stats` 具有较广的运行状态可见性，账号应受管、审计并按组织策略撤销。缺少该能力时 online 模式只会告警，maintenance 模式会直接阻断。

预检脚本同时使用 PostgreSQL `default_transaction_read_only=on` 与 `REPEATABLE READ READ ONLY`，只输出无维度聚合 JSON，不输出 ID、PII、完整 Key、单条金额或请求样本。

## 5. 在线盘点

### 5.1 保留旧镜像与固定候选镜像

```bash
export M195_OLD_IMAGE_ID="$(docker inspect sub2api --format '{{.Image}}')"
test -n "$M195_OLD_IMAGE_ID"
docker image inspect "$M195_OLD_IMAGE_ID" \
  --format '{{.Id}} {{json .RepoDigests}}' \
  > "$M195_RUN_DIR/old-image.txt"
docker image tag "$M195_OLD_IMAGE_ID" \
  "luoxueapi:rollback-m195-${M195_RUN_ID}"

export M195_CANDIDATE_IMAGE="approved-candidate:immutable-tag"
export M195_CANDIDATE_IMAGE_ID="$(
  docker image inspect "$M195_CANDIDATE_IMAGE" --format '{{.Id}}'
)"
test -n "$M195_CANDIDATE_IMAGE_ID"
docker image inspect "$M195_CANDIDATE_IMAGE_ID" \
  --format '{{.Id}} {{json .RepoDigests}}' \
  > "$M195_RUN_DIR/candidate-image.txt"
```

在切换结束前，不删除旧镜像或 rollback 标签。

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
  docker image inspect "$M195_CANDIDATE_IMAGE_ID" \
    --format '{{ index .Config.Labels "org.opencontainers.image.revision" }}'
)"
test "$M195_IMAGE_REVISION" = "$M195_RELEASE_COMMIT"

docker run --rm --entrypoint /app/sub2api \
  "$M195_CANDIDATE_IMAGE_ID" --version > "$M195_RUN_DIR/candidate-version.txt" 2>&1
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

docker run --rm --entrypoint /app/sub2api \
  "$M195_CANDIDATE_IMAGE_ID" --migration-manifest \
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
docker run --rm --entrypoint /app/sub2api \
  "$M195_CANDIDATE_IMAGE_ID" --help 2>&1 \
  | grep -F -- '-migrate-only'
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
docker compose \
  -f docker-compose.local.yml \
  -f docker-compose.custom.yml \
  stop -t 120 "$M195_APP_SERVICE"
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
```

## 7. 备份与恢复点

Writer 停止后建立逻辑备份，并在基础设施侧确认一致性快照/PITR 恢复点已经完成：

```bash
cd "$M195_DEPLOY_DIR"
docker compose \
  -f docker-compose.local.yml \
  -f docker-compose.custom.yml \
  exec -T "$M195_PG_SERVICE" sh -ec '
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
docker compose \
  -f docker-compose.local.yml \
  -f docker-compose.custom.yml \
  exec -T "$M195_PG_SERVICE" pg_restore --list \
  < "$M195_RUN_DIR/pre-195.dump" \
  > "$M195_RUN_DIR/pre-195.restore-list.txt"
```

放行条件：备份可列出、校验和已记录、PITR/快照已完成、恢复流程已在隔离环境演练、备份不与数据库共用单一故障域。

## 8. Maintenance preflight

只有确认所有 writer 已停止后才能使用 `--confirm-writers-stopped`：

```bash
cd "$M195_RELEASE_ROOT"
backend/scripts/preflight-subscription-anchored-monthly-quota.sh \
  --mode maintenance \
  --dbname "service=${M195_DB_SERVICE}" \
  --legacy-writers-stopped-at "$M195_WRITERS_STOPPED_AT" \
  --term-guard-hours 2 \
  --quiescence-seconds 60 \
  --confirm-writers-stopped \
  > "$M195_RUN_DIR/preflight-maintenance.json" \
  2> "$M195_RUN_DIR/preflight-maintenance.log"

jq -e '
  .contract == "migration-195-preflight/v1"
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

创建只覆盖应用镜像的临时 Compose 文件，使 one-shot 容器固定到候选 image ID，同时继承生产 service 的网络、配置与挂载：

```bash
set -Eeuo pipefail
[[ "$M195_APP_SERVICE" =~ ^[A-Za-z0-9][A-Za-z0-9_.-]*$ ]]
[[ "$M195_CANDIDATE_IMAGE_ID" =~ ^sha256:[0-9a-f]{64}$ ]]
export M195_IMAGE_OVERRIDE="$M195_RUN_DIR/candidate-image.override.yml"
cat > "$M195_IMAGE_OVERRIDE" <<YAML
services:
  ${M195_APP_SERVICE}:
    image: ${M195_CANDIDATE_IMAGE_ID}
YAML
chmod 600 "$M195_IMAGE_OVERRIDE"

cd "$M195_DEPLOY_DIR"
docker compose \
  -f docker-compose.local.yml \
  -f docker-compose.custom.yml \
  -f "$M195_IMAGE_OVERRIDE" \
  config --quiet
docker compose \
  -f docker-compose.local.yml \
  -f docker-compose.custom.yml \
  -f "$M195_IMAGE_OVERRIDE" \
  config --format json \
  | jq -er --arg service "$M195_APP_SERVICE" \
      '.services[$service].image // empty' \
  | grep -Fx "$M195_CANDIDATE_IMAGE_ID"
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
docker compose \
  -f docker-compose.local.yml \
  -f docker-compose.custom.yml \
  -f "$M195_IMAGE_OVERRIDE" \
  run --no-deps --pull never \
  --name "$M195_MIGRATION_CONTAINER" \
  "$M195_APP_SERVICE" --migrate-only \
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
backend/scripts/preflight-subscription-anchored-monthly-quota.sh \
  --mode maintenance \
  --dbname "service=${M195_DB_SERVICE}" \
  --legacy-writers-stopped-at "$M195_WRITERS_STOPPED_AT" \
  --term-guard-hours 2 \
  --quiescence-seconds 60 \
  --confirm-writers-stopped \
  > "$M195_RUN_DIR/preflight-post-migration.json" \
  2> "$M195_RUN_DIR/preflight-post-migration.log"

jq -e '
  .already_applied == true
  and .schema_ok == true
  and .read_only_verified == true
  and .writer_watermark_observation_stable == true
  and .writer_process_stop_acknowledged == true
  and .counts.database_prepared_transactions == "0"
  and (.blockers | length) == 0
' "$M195_RUN_DIR/preflight-post-migration.json"
```

该次结果的 `safe_to_apply=false` 是正确行为，因为 migration 已应用。额外确认三个约束存在且已验证：

```sql
SELECT conname, convalidated
FROM pg_constraint
WHERE conrelid IN (
    'public.user_subscriptions'::regclass,
    'public.billing_usage_entries'::regclass
)
  AND conname IN (
    'billing_usage_entries_subscription_amount_check',
    'user_subscriptions_weekly_window_anchored_check',
    'user_subscriptions_monthly_window_anchored_check'
)
ORDER BY conname;
```

必须恰好三行且全部 `convalidated=true`。

## 11. 精确失效会员缓存

全部应用实例仍停止时，只删除配置中目标 Redis DB 的 `billing:sub:*`。禁止 `KEYS`、`FLUSHDB`、`FLUSHALL`，也不得删除其他前缀：

```bash
cd "$M195_DEPLOY_DIR"
docker compose \
  -f docker-compose.local.yml \
  -f docker-compose.custom.yml \
  exec -T -e M195_REDIS_DB="$M195_REDIS_DB" \
  "$M195_REDIS_SERVICE" sh -eu -c '
    keys_file="$(mktemp /tmp/luoxue-m195-redis-keys.XXXXXX)"
    trap '\''rm -f "$keys_file"'\'' EXIT HUP INT TERM

    test "$(redis-cli -n "$M195_REDIS_DB" --raw PING)" = "PONG"
    redis-cli -n "$M195_REDIS_DB" --scan --pattern "billing:sub:*" \
      > "$keys_file"
    while IFS= read -r key; do
      [ -n "$key" ] || continue
      case "$key" in
        billing:sub:*) ;;
        *) exit 1 ;;
      esac
      redis-cli -n "$M195_REDIS_DB" UNLINK "$key" >/dev/null
    done < "$keys_file"

    : > "$keys_file"
    redis-cli -n "$M195_REDIS_DB" --scan --pattern "billing:sub:*" \
      > "$keys_file"
    test ! -s "$keys_file"
  '
```

若生产 Redis 为外部服务或启用 TLS，使用同等受控的生产工具执行相同 prefix-scoped `SCAN` + `UNLINK` 语义。

## 12. 新版本 canary 与恢复流量

使用同一候选 image override 只启动一个新版本应用实例，入口继续保持维护状态：

```bash
cd "$M195_DEPLOY_DIR"
docker compose \
  -f docker-compose.local.yml \
  -f docker-compose.custom.yml \
  -f "$M195_IMAGE_OVERRIDE" \
  up -d --no-deps --force-recreate "$M195_APP_SERVICE"
```

必须验证：

- 容器 image ID 与候选一致、restart count 为 0、`/health` 与 `/readyz` 正常；
- 新会员在创建时立刻写入准确 `starts_at`、weekly/monthly anchor；
- 7 天与 30 天边界均为半开区间，非零点、跨月、跨年、DST 不改变固定秒数；
- 未中断续费不改 anchor，过期重开和换套餐建立准确新 anchor；
- 显式月额度优先，未设置时服务端使用周额度乘 4，显式 `0` 不得当成未设置；
- member key 耗尽且余额充足时 member 仍 blocked，balance key 可 usable，不发生自动余额回退；
- Redis/DB/快照不一致或查询失败时返回 `stale/unknown`，不渲染为 0；
- viewer token 的 audience、`quota:read` scope、过期和设备撤销均按合同生效；
- overview/progress DTO 金额为十进制定点字符串、ID 为字符串，不含完整 API Key、上游额度或无关 PII；
- 一笔受控会员请求产生 `billing_type=1` 且非空 `subscription_amount` 的 receipt；
- 首次失败不产生 0 余额或 0% 假数据，旧快照正确携带 `as_of`、`fresh_until`、`stale`。

只有 canary 全部通过后才解除入口维护并逐步恢复流量。恢复后继续观察错误率、unknown/stale 比例、会员 blocked 原因和 receipt 缺失情况。

## 13. 回滚边界

### 迁移事务提交前

Migration 失败会整体回滚。保持维护状态，确认 195 未记录、schema 未部分变化后，可以修复数据并重试；若决定取消发布，可恢复旧镜像。不得只根据容器退出码推断回滚，必须检查数据库。

### 迁移已提交、尚未开放新写流量

旧二进制业务不兼容，不能只切回旧镜像。首选保持维护并向前修复。确需回到旧版时，只能在 writer 持续停止的条件下恢复已验证的 migration 前数据库快照/PITR，再启动旧镜像，并接受恢复点之后数据被回退的影响。

### 新版本已产生写入

默认只能向前修复。直接恢复旧镜像会重新引入旧窗口/计费语义；直接回数据库快照会丢失新写入。只有经过事故指挥、数据损失评估和完整恢复方案批准后，才能执行灾难恢复。

## 14. 发布前测试门槛

代码侧至少通过：

```bash
cd backend
go test ./migrations \
  -run '^TestSubscriptionAnchoredMonthlyQuotaMigration' -count=1
go test -tags=integration ./internal/repository \
  -run '^TestMigration195' -count=1
go test -tags=integration ./internal/repository \
  -run '^TestMigration195Preflight' -count=1
go test ./internal/repository \
  -run '^(TestApplyConfiguredMigrationsRejects|TestMigration195PreflightSQL)' -count=1
go test ./cmd/server -count=1
```

PostgreSQL 默认常为 `max_prepared_transactions=0`，真实 2PC 集成用例会在该环境明确 skip；无论是否启用 2PC，静态门禁测试都必须通过。生产预检仍查询 `pg_prepared_xacts`，不得因为测试容器禁用 2PC 而移除该检查。

并按 `quota-viewer/docs/QUOTA_OVERVIEW_API.zh-CN.md` 第 7、8 节完成合同/安全矩阵，重点覆盖：错误 audience/scope/过期/撤销设备、用户隔离、无完整 Key、7d/30d 边界、续费/重开/换套餐、显式/回退月额度、member 与 balance 独立状态、缓存不一致、首次失败、stale 快照和 `period_usage=unknown`。

## 15. 完成证据

变更单至少附：

- release SHA-256、无 AppleDouble 证明、候选与旧镜像不可变 ID；
- 实际 Compose project/文件链、候选/checkout migration manifest、构建 provenance 和数据库 pending migration 清单；
- online、maintenance、post-migration 三份预检 JSON 及校验和；
- 数据库备份校验和、PITR/快照 ID、恢复演练记录；
- migrate-only 容器 image ID、独立退出码文件和日志；
- 缓存 prefix 清理结果；
- canary 验收结果、恢复流量时间与观察结论；
- 明确保留的 rollback 镜像和清理计划。
