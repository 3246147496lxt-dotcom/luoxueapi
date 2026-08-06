# Mihomo 多容器出口部署

本方案为 Sub2API 提供三个相互独立、可按账号固定绑定的出口：

```text
Sub2API -> mihomo-a:7890 -> 出口 A
        -> mihomo-b:7890 -> 出口 B
        -> mihomo-c:7890 -> 出口 C
```

三个服务只加入 `sub2api-network`，不向宿主机或公网发布代理端口。出口 A/B/C 必须使用不同的远端服务器、线路或公网出口；复制同一节点不能增加真实容量。

## 1. 准备每个实例的私有配置

在 `deploy` 目录执行：

```bash
for shard in a b c; do
  install -d -m 700 "mihomo/instances/${shard}/etc/providers"
  install -d -m 700 "mihomo/instances/${shard}/state"
  test -e "mihomo/instances/${shard}/etc/config.yaml" || \
    install -m 600 mihomo/config.example.yaml "mihomo/instances/${shard}/etc/config.yaml"
  test -e "mihomo/instances/${shard}/etc/providers/egress.yaml" || \
    install -m 600 mihomo/egress.example.yaml "mihomo/instances/${shard}/etc/providers/egress.yaml"
done
```

这些命令只会创建缺失的示例配置，重复执行不会覆盖已有实例。`mihomo/instances/` 已被 Git 忽略。不要提交其中的订阅地址、节点密钥、用户名或密码。

分别编辑三个实例：

1. 在 `etc/config.yaml` 中替换 `REPLACE_WITH_PROXY_USERNAME` 和 `REPLACE_WITH_PROXY_PASSWORD`。建议每个实例使用不同密码，可用 `openssl rand -hex 24` 生成。
2. 在 `etc/providers/egress.yaml` 中粘贴该实例使用的完整 Mihomo 节点定义。
3. 每个 provider 建议只放一个主节点；如需故障切换，可在主节点之后放一个专用备用节点。`fallback` 只在主节点不可用时切换，不用于日常分摊吞吐。
4. A/B/C 不要共用同一个主节点，也不要让三个小组无条件共用一个容量不足的备用节点。

如果当前节点来自远程订阅，也可以把 `type: file` provider 改成现有的 `type: http` 配置，并使用精确的 `filter` 确保每个实例只选中自己的节点。订阅 URL 仍只能写入被忽略的实例配置。Mihomo 不会对 YAML 中的通用 `${VAR}` 做环境变量替换，不要把 listener 密码写成 `${MIHOMO_PASSWORD}` 后期待自动展开。

## 2. 预检配置

先验证 Compose 合并结果，不会启动或重建现有服务：

```bash
docker compose \
  -f docker-compose.local.yml \
  -f docker-compose.mihomo.yml \
  config -q
```

运行仓库提供的预检脚本。它会拒绝占位凭据、空 provider 和权限过宽的敏感文件，并使用覆盖文件相同的固定镜像逐个执行 Mihomo 静态配置检查：

```bash
./mihomo/preflight.sh
```

`mihomo -t` 不会真正加载 `type: file` provider，预检脚本只能对 provider 做非空结构检查。节点协议字段、认证和真实出口必须在容器启动后通过 Sub2API 的“测试连接”和“质量检测”验证。若服务器实际使用的主 Compose 文件名或 Compose project name 不同，后续命令必须沿用同一个文件名与 project name，才能加入现有 `sub2api-network`。

## 3. 启动三个出口

```bash
docker compose \
  -f docker-compose.local.yml \
  -f docker-compose.mihomo.yml \
  up -d mihomo-a mihomo-b mihomo-c

docker compose \
  -f docker-compose.local.yml \
  -f docker-compose.mihomo.yml \
  ps mihomo-a mihomo-b mihomo-c

docker compose \
  -f docker-compose.local.yml \
  -f docker-compose.mihomo.yml \
  logs --tail=100 mihomo-a mihomo-b mihomo-c
```

覆盖文件只有 `expose: 7890`，没有 `ports:`。不要为了测试改成 `0.0.0.0:7890:7890`；应通过 Sub2API 后台的代理测试功能在 Docker 私网内验证。

Compose healthcheck 只检测容器内的 HTTP listener 是否正在监听，不验证远端出口，也不会仅因 `unhealthy` 自动重启容器。真实出口健康仍以 provider 健康检查、后台代理测试和业务流量指标为准。

## 4. 在 Sub2API 中登记并灰度迁移

在代理管理中新增三条 HTTP 代理，认证信息与各自 `config.yaml` 保持一致：

| 名称 | 地址 |
|---|---|
| Linux Mihomo A | `mihomo-a:7890` |
| Linux Mihomo B | `mihomo-b:7890` |
| Linux Mihomo C | `mihomo-c:7890` |

迁移顺序：

1. 保留原 `mihomo:7890` 和原账号绑定，不要立即删除。
2. 每个新出口先迁移一个低流量账号，运行一个峰值窗口。
3. 确认错误率、首字延迟和流式稳定性正常后，再按账号并发总量分配其余账号。九个负载相近的账号可以从 3/3/3 起步。
4. 任一出口出现问题时，把对应账号重新绑定到旧代理即可回滚。
5. 连续观察至少一至两个完整峰值窗口后，再决定是否下线旧入口。

不要把账号并发设置为 `0`；在当前应用中它表示不限制。每个出口应保留约 20%–40% 的流式请求波动余量。

覆盖文件默认给每个实例最多 2 CPU、1 GiB 内存和 512 个进程，并将镜像固定到 v1.19.28 的多架构摘要。若压测看到 CPU throttling、OOM 或连接容量不足，可在 `.env` 中调整 `MIHOMO_CPUS_PER_INSTANCE`、`MIHOMO_MEM_LIMIT_PER_INSTANCE`、`MIHOMO_MEM_RESERVATION_PER_INSTANCE`、`MIHOMO_PIDS_LIMIT_PER_INSTANCE`；自定义 `MIHOMO_IMAGE` 时仍应使用 `tag@sha256:digest`。

## 5. 停止或回滚容器

停止新出口不会删除实例配置：

```bash
docker compose \
  -f docker-compose.local.yml \
  -f docker-compose.mihomo.yml \
  stop mihomo-a mihomo-b mihomo-c
```

确认账号已全部迁回旧代理后，才可移除新容器：

```bash
docker compose \
  -f docker-compose.local.yml \
  -f docker-compose.mihomo.yml \
  rm -f mihomo-a mihomo-b mihomo-c
```
