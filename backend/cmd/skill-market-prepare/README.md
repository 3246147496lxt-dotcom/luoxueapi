# skills.sh 市场包准备器

这个命令只生成本地审核/上传素材，不连接管理后台，也不会执行 Skill 包里的脚本。

## 推荐用法

先保存目标时刻的 `https://skills.sh/` 首页 HTML，再用同一输出目录反复运行：

```bash
cd backend
go run ./cmd/skill-market-prepare \
  -html /absolute/path/to/skills.sh.html \
  -out /absolute/path/to/skills-sh-top500 \
  -limit 500 \
  -min-records 600 \
  -acquire auto
```

`auto` 会优先读取 skills.sh 的公开下载快照；单条失败和非 GitHub 的 well-known 来源会回退到官方 CLI：

```text
DISABLE_TELEMETRY=1 npx -y skills@latest add <source-url> --skill <ids...> --agent codex --copy -y
```

如需严格走官方 CLI，使用 `-acquire cli`。完全断网检查已有缓存和断点时，使用 `-acquire offline -audit=false`。首次没有 `snapshot.json` 且未传 `-html` 时，命令可从 `https://skills.sh/api/skills/all-time/{page}` 获取至少三页；对于需要长期留证的正式收录，仍推荐传入保存好的 HTML。

## 输出

- `snapshot.json`：冻结后的榜单、全局唯一 slug 映射和上游输入哈希。已存在时不会重新读取榜单。
- `checkpoint.json`：按来源和 Skill 保存获取/打包进度；再次运行只重试未完成项。
- `packages/*.zip`：经过 `service.ValidateSkillArchive` 规范化的最终 ZIP。
- `metadata/*.json`：单 Skill 元数据、来源、审计、裁剪记录、许可证提示和校验结果。
- `manifest.json`：按榜单顺序汇总的导入清单。
- `failures.json`：未完成条目及阶段/错误。
- `cache/sources/*`：每个 source 隔离的 CLI 或 skills.sh 下载缓存。
- `cache/audits/*`：公开审计端点的成功响应缓存。

API 下载得到的是 skills.sh 自身的内容快照，不保证等于仓库当前 `HEAD`，因此这类条目的 `source_url` 只指向仓库根，并单独记录 `skills_sh_download_hash`。只有官方 CLI 实际按当前仓库获取且 `git ls-remote HEAD`、`skills-lock.json` 都可用时，才尽量生成固定 commit/tree 链接。

包内 `SKILL.md` 的 frontmatter 会改写为市场 slug，并加入 `skills_sh_import` 来源信息。被排除的文件会完整记录在 metadata；缺少 LICENSE/LICENCE/COPYING 时会标记 `license_unverified=true`，但不会推断或改变上游许可。

## 验证

```bash
cd backend
gofmt -w cmd/skill-market-prepare/*.go
go test ./cmd/skill-market-prepare
go test ./internal/service -run 'TestValidateSkillArchive|TestSkillArchiveWarnings'
```
