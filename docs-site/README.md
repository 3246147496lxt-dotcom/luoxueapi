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

## 接入现有中转站

1. 将 `dist/` 发布到服务器 `/srv/luoxue-docs`。
2. 按 `deploy/README.md` 将主域 `/tutorial-docs/*` 接入 Caddy，并保留其余请求到现有主程序的反向代理。
3. 在后台“设置 → 站点设置”中将“文档链接”设为 `https://luoxueapi.cc/tutorial-docs/`。
4. 验证登录用户在文档页显示用户信息并能进入控制台，未登录用户仍可阅读文档。
5. 最后将 `docs.luoxueapi.cc` 改为 301 跳转，兼容旧书签和历史公告。

完整的发布顺序、安全头、缓存规则、验证和回滚步骤见 `deploy/README.md`。
