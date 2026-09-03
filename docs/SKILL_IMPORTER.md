# Skill 收录器

Skill 收录器是部署在现有后端中的持久任务系统，用来把外部 Skill 目录持续同步到本站市场。它与一次性脚本不同：来源身份、每次快照、重试、转换、校验、版本和发布结果都保存在 PostgreSQL 中，服务重启后可继续。

## 工作流

1. 来源适配器分页发现条目，并冻结响应哈希、抓取时间和适配器版本。
2. 系统用 `(source_id, namespace, external_id)` 识别上游条目；排名和 slug 不参与身份判断。
3. 只抓取需要检查的内容，不执行上游代码、安装命令或 Skill 指令。
4. 统一归一化器生成确定性 ZIP，并调用市场现有的 `ValidateSkillArchive` 校验。
5. 相同规范化包哈希记为 `unchanged`；内容变化按 patch 版本递增。
6. 归一化 ZIP 与目标元数据先暂存在任务条目中，此时不会创建正式 Skill、版本或来源绑定。符合门禁的完整条目集合随后在一个 PostgreSQL 事务中成批物化并发布；任一项失败会整体回滚，失败或需要人工复核的条目不会混入发布集合。取消任务会清除尚未发布的暂存 ZIP。

来源消失只记录 `last_seen`，默认不会自动归档市场中的 Skill。

公开目录的审核后翻译保存在独立的 `skill_catalog_localizations` 中，以市场 slug 和 locale 标识。收录器继续刷新 `skills` 中的上游原文，不会覆盖翻译；公开 API 按请求语言读取翻译，缺少对应语言时回退上游原文。`SKILL.md`、ZIP 和版本摘要保持上游内容，不属于目录展示文案翻译范围。

## 内置适配器

- `skills_sh`：读取排行榜；GitHub 来源默认优先走共享的 GitHub 仓库快照缓存，skills.sh download 仅作为每小时最多 60 次的回退。download 返回的 `hash` 只作为上游快照版本标识，不冒充内容摘要，因此该回退通道的产物会进入人工复核；非 GitHub 来源必须同时配置 `source_base_urls` 并加入 `allowed_source_hosts`。
- `github`：同步一个公开 GitHub 仓库中的明确 Skill 路径或 skill id；支持服务端 GitHub token，固定 commit 后逐文件校验 Git object id。
- `well_known`：先读取站点的 `/.well-known/agent-skills/index.json`，再回退到 `/.well-known/skills/index.json`；支持带 SHA-256 的 Skill Markdown、文件集合或 ZIP。
- `manifest`：读取 HTTPS JSON/CSV manifest 或单个 ZIP URL；后台也可以上传不超过 5 MiB 的 JSON、CSV、ZIP，上传内容只进入不可变 run 配置，不落本机临时文件。

任意网站并不会被“通用爬虫”盲目解析。静态目录可用 manifest；具有专用 API、登录、验证码或动态页面的网站应新增一个小型、受测试的 `SourceAdapter`。

## 默认策略

迁移会幂等创建：

- `skills.sh All Time` 来源，来源优先级 100；
- `Manual Manifest Upload` 来源，可直接上传 JSON、CSV 或 ZIP；
- `Daily Top 500` 计划，`0 3 * * *`、`Asia/Shanghai`；
- 计划默认 **禁用**，必须先完成 dry-run 再由管理员启用；
- 一次 bootstrap 任务，把已经发布且 ZIP 内含明确 skills.sh provenance 的 Skill 绑定到稳定来源身份，避免首次同步重复创建。无法明确证明身份的现有 Skill 只记为 skipped，不会猜测绑定。

没有许可证文件的包会标记 `license_unverified` 并保留来源/版权说明；它不是静默授权结论。没有独立摘要校验的上游内容、安全归一化导致文件排除、缺失或重写 frontmatter 等情况默认进入人工复核，不会越过自动发布安全门。私钥、路径穿越、软链接、ZIP bomb、超限内容和不安全网络目标会阻断。

## 管理后台

入口：`/admin/skills/imports`。

- **收录任务**：查看任务进度、条目、证据和错误；取消任务；仅重试失败条目；将整个 eligible cohort 在一个事务中发布。
- **收录计划**：维护来源、cron、时区、排名范围、模式、元数据策略和安全门禁。
- **Skill 目录**：继续使用原市场管理页处理人工创建和单条版本操作。

运行中的页面每 2 秒刷新；失败后按 2/5/10/30 秒退避，页面隐藏时暂停轮询。

## 服务配置

完整示例见 `deploy/config.example.yaml` 和 `deploy/.env.example`。常用环境变量：

```text
SKILL_IMPORT_ENABLED=true
SKILL_IMPORT_WORKER_ENABLED=true
SKILL_IMPORT_WORKER_CONCURRENCY=2
SKILL_IMPORT_PER_HOST_CONCURRENCY=1
SKILL_IMPORT_POLL_INTERVAL_SECONDS=2
SKILL_IMPORT_LEASE_TTL_SECONDS=300
SKILL_IMPORT_MAX_ATTEMPTS=5
SKILL_IMPORT_HTTP_TIMEOUT_SECONDS=30
SKILL_IMPORT_GITHUB_TOKEN=
```

GitHub token 只从服务端配置读取，不写入来源配置、任务快照、事件或日志。生产部署首版与 API 服务同容器运行；任务状态全部在 PostgreSQL，未来可用同一二进制的 worker role 拆成独立容器。

每个任务会冻结 adapter、normalizer 和 validator 的规则版本。修改任一发现、获取、清洗或校验规则时，必须同步提升对应版本；旧任务只会由版本一致的实现继续执行，不能在新规则下静默续跑。

skills.sh download 的 60 次/小时配额目前由单进程限流器保护。首版建议只启用一个收录 worker 实例；拆成多实例 worker 前，应把该配额迁移为 PostgreSQL 或 Redis 中的全局限流器。

## 上线检查

1. 应用迁移 199，并确认默认计划仍为 disabled。
2. 启动服务，等待 bootstrap 任务结束；检查 conflict/failed 项，不做模糊绑定。
3. 对目标来源手动运行 `dry_run`，检查 blocked、failed、license 和 excluded files。
4. 用小范围（例如前 5 条）运行 review 或 auto-publish，验证市场数量、版本 SHA 和下载包。
5. 再扩大到目标范围；只有确认主机限流和失败率正常后才启用定时计划。

数据库 lease 使用 `FOR UPDATE SKIP LOCKED`，崩溃或实例切换后会回收过期租约。重复任务依靠 Idempotency-Key、稳定来源身份和包哈希保持幂等。
