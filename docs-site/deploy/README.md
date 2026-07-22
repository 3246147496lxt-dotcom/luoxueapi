# `/tutorial-docs/` 随主服务发布

文档站的正式地址是：

```text
https://luoxueapi.cc/tutorial-docs/
```

文档站不是独立部署单元。正式 Docker 构建和 GitHub Release 会先构建
`docs-site`，再把 `dist/` 放入后端嵌入目录的 `tutorial-docs/` 子目录，最终随
Go 二进制和主站一起进入同一个镜像。因此主站、登录注册页、控制台和文档站
始终来自同一个版本，不再依赖 `/srv/luoxue-docs`、静态目录软链接或
`deploy/luoxue-docs-site.tar.gz`。

## 正式构建

仓库根目录的 `Dockerfile` 和 `deploy/Dockerfile` 都会执行以下流程：

1. 构建主前端。
2. 使用 `docs-site/package-lock.json` 冻结安装并构建文档站。
3. 运行 `npm run verify:build`，确认资源基路径是 `/tutorial-docs/`。
4. 将文档产物复制到 `backend/internal/web/dist/tutorial-docs/`。
5. 以 `-tags embed` 编译包含两套前端资源的 Go 二进制。

本地构建正式镜像：

```bash
docker build -t luoxueapi:local -f Dockerfile .
```

Release 工作流使用相同布局，并在 GoReleaser 前运行嵌入路由测试。不要手工上传
`docs-site/dist/`；只部署通过验证的完整镜像。

## Caddy 路由

Caddy 必须把主域的所有路径（包括 `/tutorial-docs/`）反向代理到后端：

```caddyfile
luoxueapi.cc {
	encode zstd gzip

	@tutorial_docs path /tutorial-docs /tutorial-docs/*
	header @tutorial_docs {
		Cross-Origin-Opener-Policy "same-origin"
		Cross-Origin-Resource-Policy "same-origin"
		Permissions-Policy "camera=(), geolocation=(), microphone=()"
		Strict-Transport-Security "max-age=31536000"
	}

	reverse_proxy 127.0.0.1:8080
}
```

不要保留旧的 `handle_path /tutorial-docs/*` 静态文件规则，否则它会在请求到达
新镜像前拦截路径，继续显示 `/srv/luoxue-docs` 中的旧版本。
文档路径仍由 Caddy 补充 HSTS、Permissions Policy、COOP 和 CORP；CSP 与缓存
策略由后端统一返回，Caddy 不覆盖它们。

后端负责文档路径语义：

- `/tutorial-docs` 永久跳转到 `/tutorial-docs/`。
- `/tutorial-docs/` 返回文档入口。
- `/tutorial-docs/orders` 等无扩展名深链回退到文档入口，而不是主站入口。
- 不存在的 JS、CSS、图片等资源返回 404。
- 带 Vite 内容哈希的资源使用一年期 `immutable` 缓存，HTML 使用 `no-cache`。

旧文档子域可以继续保留为兼容跳转：

```caddyfile
docs.luoxueapi.cc {
	redir https://luoxueapi.cc/tutorial-docs{uri} permanent
}
```

修改线上 Caddyfile 后先执行：

```bash
caddy fmt --diff /etc/caddy/Caddyfile
caddy validate --config /etc/caddy/Caddyfile --adapter caddyfile
systemctl reload caddy
```

## 上线验证

```bash
curl -I https://luoxueapi.cc/tutorial-docs
curl -I https://luoxueapi.cc/tutorial-docs/
curl -I https://luoxueapi.cc/tutorial-docs/orders
curl -I https://luoxueapi.cc/dashboard
curl -I https://luoxueapi.cc/api/v1/settings/public
curl -I https://docs.luoxueapi.cc/
```

确认精确路径跳转、文档首页和深链返回 HTML、主站与 API 正常、旧子域跳转。
浏览器中还要确认文档脚本和样式来自 `/tutorial-docs/assets/`，品牌设置能从
`/api/v1/settings/public` 同步，控制台没有新增错误。

## 回滚

文档与主站现在共用发布单元，回滚时应把应用镜像整体切回部署前记录的镜像
ID，并等待容器健康检查通过。不要只恢复旧静态目录，否则会形成主站与文档
版本不一致。若线上曾配置独立文档静态路由，也要确认回滚后的 Caddy 配置与
目标镜像相匹配。

服务器密码、Cloudflare Token、数据库连接串和私钥不得写入聊天记录或仓库。
