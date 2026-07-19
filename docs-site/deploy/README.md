# `/tutorial-docs/` 同域部署

文档站的正式地址是：

```text
https://luoxueapi.cc/tutorial-docs/
```

静态文件仍放在 `/srv/luoxue-docs`。Caddy 使用 `handle_path` 去掉 `/tutorial-docs` 前缀后读取该目录；其他主域请求继续反向代理到 `127.0.0.1:8080`。旧地址 `docs.luoxueapi.cc` 最终会以 301 跳转到主域下的对应文档路径。

## 1. 构建和发布静态文件

在 `docs-site` 目录执行：

```bash
npm ci
npm run build
```

确认 `dist/index.html` 内的脚本、样式和图标 URL 均以 `/tutorial-docs/` 开头，再将 `dist/` 中的全部文件解压到一个新的版本目录。第一阶段不要切换 `/srv/luoxue-docs` 软链接；旧子域仍需要它指向旧版本。不要覆盖或删除上一个版本目录，它是静态文件的回滚点。

## 2. 分两阶段切换 Caddy

`Caddyfile.production` 是最终完整配置，`Caddyfile` 内容与其一致，可作为合并参考。上线前先备份线上 Caddyfile。

第一阶段先合并 `luoxueapi.cc` 中的 `/tutorial-docs` 路由，让该路由的 `root` 临时指向刚解压的新版本绝对路径；同时保留线上现有的 `docs.luoxueapi.cc` 静态站点块和旧软链接。这样可以先验证新地址，同时旧地址仍可用。每次修改后都执行：

```bash
caddy fmt --diff /etc/caddy/Caddyfile
caddy validate --config /etc/caddy/Caddyfile --adapter caddyfile
systemctl reload caddy
```

验证新地址成功并更新后台链接后，先原子切换 `/srv/luoxue-docs` 到新版本，再安装 `Caddyfile.production`。最终配置会让主域文档重新读取软链接，并将旧子域站点块替换为：

```caddyfile
docs.luoxueapi.cc {
	redir https://luoxueapi.cc/tutorial-docs{uri} permanent
}
```

重新校验并 reload。Cloudflare 中现有 `docs.luoxueapi.cc` DNS 记录应保留，才能让旧链接继续跳转。

## 3. 缓存和安全策略

- `/tutorial-docs` 精确请求以 301 规范到 `/tutorial-docs/`。
- HTML 和未哈希资源返回 `Cache-Control: no-cache`。
- 构建生成且文件名包含至少 8 位 Vite 内容哈希的 JS、CSS 和字体返回一年期 `immutable` 缓存；可替换的教程截图不使用长期缓存。
- CSP 仅允许同源脚本和 API 请求；头像允许 HTTPS 图片，脚本不允许第三方来源。
- 文档禁止被 iframe 嵌入，并启用 HSTS、`nosniff`、权限限制等响应头。

若以后增加第三方脚本、字体或 API，不要直接放宽 `default-src`；只在对应 CSP 指令中加入经过确认的具体域名。

## 4. 更新后台文档链接

先备份设置：

```bash
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f deploy/backup-doc-settings.sql
```

确认新地址可访问后执行：

```bash
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f deploy/activate-doc-link.sql
```

脚本会把 `doc_url` 设为 `https://luoxueapi.cc/tutorial-docs/`，并删除旧的 `md:guide` 重复菜单项。现有前端仍会在新标签页打开文档。

## 5. 验证清单

```bash
curl -I https://luoxueapi.cc/tutorial-docs
curl -I https://luoxueapi.cc/tutorial-docs/
curl -I https://docs.luoxueapi.cc/
curl -I https://luoxueapi.cc/dashboard
curl -I https://luoxueapi.cc/api/v1/auth/me
```

确认精确路径返回 301、带斜杠路径返回文档、旧子域返回 301，且主站与 API 没有被静态路由拦截。再用浏览器覆盖未登录、已登录、过期 Token、跨标签登录/退出、桌面端与移动端。

## 回滚

1. 将 `/srv/luoxue-docs` 软链接切回上一版本，确认旧静态包可读取。
2. 恢复部署前备份的 Caddyfile，执行 `caddy validate` 后 reload，使 `docs.luoxueapi.cc` 恢复独立托管。
3. 从 `backup-doc-settings.sql` 生成的 CSV 恢复 `doc_url` 和 `custom_menu_items`；不要只恢复其中一个键：

```sql
BEGIN;
CREATE TEMP TABLE restored_doc_settings (
  key text,
  value text,
  updated_at timestamptz
);
COPY restored_doc_settings (key, value, updated_at)
FROM '/tmp/luoxue-doc-settings-before-same-origin.csv'
WITH (FORMAT csv, HEADER true);
INSERT INTO settings (key, value, updated_at)
SELECT key, value, updated_at FROM restored_doc_settings
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value,
    updated_at = EXCLUDED.updated_at;
COMMIT;
```

4. 验证后台文档入口重新打开旧地址、主站和 API 正常后，再清理失败版本。不要提前删除上一版本、Caddy 备份或设置 CSV。

服务器密码、Cloudflare Token、数据库连接串和私钥不得写入聊天记录或仓库。
