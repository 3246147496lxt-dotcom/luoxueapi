# 落雪API 文档站模板

这是一个 React + Vite 文档站，正式部署在主站同源路径：

```text
https://luoxueapi.cc/tutorial-docs/
```

同源部署让文档页可以安全复用主站现有登录态，同时保持公开访问和新标签页打开。旧地址 `https://docs.luoxueapi.cc` 仅用于兼容历史链接，并永久跳转到新地址。

教程内容会在页面打开后从同源接口 `/api/v1/public/documentation` 读取已发布版本；请求失败、接口未开启或内容校验失败时，页面会继续显示打包内置内容，不会出现空白页。浏览器会按接口的 ETag 与缓存响应正常复用缓存。

## 最常修改的地方

- `src/content.js`：站点名称、主站地址、导航，以及接口不可用时显示的内置兜底教程。
- `public/assets/brand-mark.svg`：站点图标。
- `public/assets/*.jpg|png`：教程截图。
- `src/styles.css`：颜色、字号、间距和响应式样式。

主站地址集中在 `src/content.js` 的 `MAIN_SITE_URL`。示例密钥 `sk-your-api-key` 只用于演示，不要换成真实密钥。

## 维护内置兜底教程

日常教程更新应在管理后台发布，无需重新构建文档站。只有需要调整接口不可用时显示的兜底内容，才在 `src/content.js` 的 `tutorials` 数组中复制或修改对象：

- `id`：英文路由标识，例如 `faq`。
- `tabLabel`：顶部标签名称。
- `description`：教程摘要。
- `steps`：教程步骤。

步骤可以包含 `note`、`code`、`image` 和 `link`，不需要的字段可直接删除。

## 本地预览

```bash
npm install
npm run dev
```

开发服务器默认把 `/api` 代理到 `http://localhost:8080`。如后端使用其他地址，可在启动前设置 `VITE_DEV_PROXY_TARGET`。

## 构建

```bash
npm run build
```

构建结果位于 `dist/`。生产构建的基础路径必须是 `/tutorial-docs/`，发布前检查 `dist/index.html` 中的脚本、样式和图标地址均带有该前缀。

## 随主站发布

文档站不是独立部署单元。根目录 Dockerfile 和 GitHub Release 会构建主前端与
`docs-site`，把两者组装进 Go 后端的嵌入资源，再生成同一个完整镜像。不要把
`dist/` 单独上传到 `/srv/luoxue-docs`，也不要让 Caddy 用旧静态目录拦截
`/tutorial-docs/*`。

1. 构建或拉取包含主站与文档站的完整镜像。
2. 保留当前运行镜像的不可变 ID 和回滚标签，再整体切换应用容器。
3. 在后台“设置 → 站点设置”中将“文档链接”设为 `https://luoxueapi.cc/tutorial-docs/`。
4. 验证登录用户在文档页显示用户信息并能进入控制台，未登录用户仍可阅读文档。
5. 保留 `docs.luoxueapi.cc` 到主域文档路径的永久跳转，兼容旧书签和历史公告。

需要回滚时，主站与文档站必须一起切回发布前记录的完整镜像，避免两套界面版本
错配。完整的构建、Caddy、安全响应头、验证和整体回滚步骤见 `deploy/README.md`。
