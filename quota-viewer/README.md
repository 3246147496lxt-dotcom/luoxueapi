# 落雪额度

独立的 macOS / Windows 会员周额度只读查看器。它不接入现有 `desktop/`，
拥有自己的 Tauri 壳、包名、窗口和菜单栏入口。

## P0 产品定位

本产品的核心不是缩小版数据看板，也不承担账户管理。用户查看桌面悬浮窗时，
只需要快速确认三件事：

1. 当前会员套餐是什么；
2. 本周额度还剩多少；
3. 当前是否可用，以及何时准确重置。

桌面悬浮窗使用专用会员状态卡，只展示：

- 左上角 `CODEX · 套餐名` 和“周剩余”；
- 整数形式的剩余百分比及圆环进度；
- 右上角状态点；
- 根据当前时间动态计算的重置倒计时；
- 服务端返回的准确重置日期和时间。

P0 默认一个账户只有一个当前会员。没有会员时显示“暂无会员订阅”。状态覆盖
可用、警告、耗尽、失效、未知和旧数据；未知数据不能用 `0%` 代替。

桌面悬浮窗不展示账户余额、今日/月消费、雪花已用或剩余、Token、请求数、
API Key 分组、本地用量详情或图表。充值、购买/管理会员和其他账户操作统一
前往网站完成。

产品保留两套互不耦合的桌面入口：

- 桌面悬浮窗默认是 `80×80` 缩小态，只显示整数百分比；点击后扩展为
  `314×314` 圆环会员卡，再次点击或按 `Esc` 收起；
- 菜单栏弹窗保持既有 `406×500` 速览，继续显示账户余额、今日/月消费和
  会员圆环；点击会员卡可向右展开本周期详情。菜单栏内容不会反向影响悬浮窗。

会员额度每 7 天按会员开通准确时间刷新。服务端 `resets_at` 是重置时间的唯一
权威来源；客户端不得按自然周、本地零点或首次调用时间自行推算周期。

详细文档：

- [P0 产品需求](docs/PRODUCT_REQUIREMENTS.zh-CN.md)
- [只读额度接口契约](docs/QUOTA_OVERVIEW_API.zh-CN.md)

## 只读授权与本地数据

Tauri 应用通过独立 `quota:read` 设备授权读取真实接口：

- 使用系统浏览器确认授权，应用不接收网站密码、验证码或 API Key；
- Refresh Token、安装标识和缓存密钥存放在应用数据目录的
  `credentials-v1.json`；Windows 使用当前登录用户的 DPAPI 加密整份凭据，
  macOS/Linux 使用 `0700` 目录和 `0600` 文件权限；同一登录账户下运行的其他
  程序仍属于本地信任边界；
- Refresh Token 使用可恢复的 `candidate-v1` 轮换；网络回包丢失或应用重启时
  复用已持久化的同一轮换组合，不会因盲目重试把正常设备误判为重放；
- 额度快照使用 AES-256-GCM 加密落盘，并按账户授权与展示时区隔离；
- 网络失败时明确显示“上次数据”或“暂时不可用”，不会用假 `0` 代替未知值；
- 退出连接会清除本地令牌和额度缓存。

浏览器开发预览使用 `src/data/demo.ts` 检查各状态的文案和颜色；生产 Tauri
运行时不会加载演示数据。浏览器开发预览中的状态切换器不会进入生产小窗。

## 本地开发

```bash
corepack pnpm install
corepack pnpm dev
corepack pnpm test
corepack pnpm build
corepack pnpm tauri dev
```

生产构建默认连接 `https://luoxueapi.cc/`。本地联调时可在编译前设置独立查看器
专用地址，例如：

```bash
LUOXUE_QUOTA_API_BASE_URL=http://127.0.0.1:8080/ corepack pnpm tauri dev
```

## Windows 构建

Windows 版本支持 Windows 10 / 11 x64，使用 NSIS 生成安装程序。安装包内嵌
Microsoft Edge WebView2 Bootstrapper，并拒绝用旧版本安装包覆盖较新版本。

构建机需要 Node.js 24、pnpm 9.15.9、Rust 1.96.0 和 Visual Studio 2022 C++
生成工具。构建前还必须提供 Windows 多尺寸图标：
`src-tauri/icons/icon.ico`。

```powershell
corepack pnpm install --frozen-lockfile
corepack pnpm run typecheck
corepack pnpm run test
corepack pnpm run build
corepack pnpm run tauri:build:windows
```

release-profile NSIS 安装包输出到
`src-tauri/target/x86_64-pc-windows-msvc/release/bundle/nsis/`。CI 上传的 Windows
artifact 未签名，只用于安装和兼容性验证。任何面向用户的正式 Windows 发布都必须
先完成可验证的 Authenticode 代码签名；未签名安装包不得发布。
