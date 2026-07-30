# 落雪额度

独立的 macOS / Windows 额度查看器。它不接入现有 `desktop/`，拥有自己的
Tauri 壳、包名、窗口和托盘入口。

产品定位是“整账户额度状态助手”，不是缩小版数据看板。客户打开应用后应能
快速确认哪些 API Key 可以继续使用、它们使用哪类权益、会员额度何时重置，
以及受限后的下一步操作。

已确认的 P0 口径：

- 展示整个账户及全部启用 API Key；
- 余额计费 Key 只扣账户雪花余额；
- 会员 Key 只扣会员 7 天额度，耗尽后不自动回退余额；
- 会员 7 天周期以会员开通准确时间为锚点；
- 不展示 Codex 或上游账号池伪额度；
- 可展示归属于当前会员本周期的真实请求数和 Token 用量，但它们不参与额度判断。

详细文档：

- [P0 产品需求](docs/PRODUCT_REQUIREMENTS.zh-CN.md)
- [只读额度接口契约](docs/QUOTA_OVERVIEW_API.zh-CN.md)

当前已完成独立小窗、会员剩余环、详情图表、无会员空状态，以及可用、警告、
耗尽、失效和待确认状态。Tauri 应用通过独立 `quota:read` 设备授权读取真实接口：

- 系统浏览器确认授权，应用不接收网站密码、验证码或 API Key；
- Refresh Token、安装标识和缓存密钥存放在系统钥匙串；
- Refresh Token 使用可恢复的 `candidate-v1` 轮换；网络回包丢失或应用重启时
  复用已持久化的同一轮换组合，不会因盲目重试把正常设备误判为重放；
- 额度快照使用 AES-256-GCM 加密落盘，并按账户授权与展示时区隔离；
- 账户余额、今日消费和本月消费来自整账户真实账务记录；
- 网络失败时明确显示“上次数据”或“暂时不可用”，不会用假 `0` 替代未知值；
- 退出连接会清除本地令牌和额度缓存。

浏览器开发预览仍使用 `src/data/demo.ts`，用于快速检查视觉状态；生产 Tauri
运行时不会加载演示数据。会员卡中的本周期请求与 Token 只有在服务端返回
`subscriptions[].period_usage` 时才显示真实数值，否则明确显示“暂无本周期用量
明细”，且不影响余额和会员剩余额度。

浏览器开发预览右侧提供“状态预览”切换器，用于检查上述状态的完整文案、数值和
颜色；Tauri 小窗及生产构建不会显示该控件。

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
