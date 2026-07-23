# ADMIN_PAYMENT_INTEGRATION_API

> 单文件中英双语文档 / Single-file bilingual documentation (Chinese + English)

---

## 中文

### 目标
本文档用于对接外部支付系统（如 `sub2apipay`）与落雪API的 Admin API，覆盖：
- 支付成功后充值
- 用户查询
- 人工余额修正
- 前端购买页参数透传

### 基础地址
- 生产：`https://<your-domain>`
- Beta：`http://<your-server-ip>:8084`

### 认证
推荐使用：
- `x-api-key: admin-<64hex>`
- `Content-Type: application/json`
- 幂等接口额外传：`Idempotency-Key`

说明：管理员 JWT 也可访问 admin 路由，但服务间调用建议使用 Admin API Key。

### 1) 一步完成创建并兑换
`POST /api/v1/admin/redeem-codes/create-and-redeem`

用途：原子完成“创建兑换码 + 兑换到指定用户”。

请求头：
- `x-api-key`
- `Idempotency-Key`

请求体示例：
```json
{
  "code": "s2p_cm1234567890",
  "type": "balance",
  "value": 100.0,
  "user_id": 123,
  "notes": "sub2apipay order: cm1234567890"
}
```

幂等语义：
- 同 `code` 且 `used_by` 一致：`200`
- 同 `code` 但 `used_by` 不一致：`409`
- 缺少 `Idempotency-Key`：`400`（`IDEMPOTENCY_KEY_REQUIRED`）

curl 示例：
```bash
curl -X POST "${BASE}/api/v1/admin/redeem-codes/create-and-redeem" \
  -H "x-api-key: ${KEY}" \
  -H "Idempotency-Key: pay-cm1234567890-success" \
  -H "Content-Type: application/json" \
  -d '{
    "code":"s2p_cm1234567890",
    "type":"balance",
    "value":100.00,
    "user_id":123,
    "notes":"sub2apipay order: cm1234567890"
  }'
```

### 2) 查询用户（可选前置校验）
`GET /api/v1/admin/users/:id`

```bash
curl -s "${BASE}/api/v1/admin/users/123" \
  -H "x-api-key: ${KEY}"
```

### 3) 余额调整（已有接口）
`POST /api/v1/admin/users/:id/balance`

用途：人工补偿 / 扣减，支持 `set` / `add` / `subtract`。

请求体示例（扣减）：
```json
{
  "balance": 100.0,
  "operation": "subtract",
  "notes": "manual correction"
}
```

```bash
curl -X POST "${BASE}/api/v1/admin/users/123/balance" \
  -H "x-api-key: ${KEY}" \
  -H "Idempotency-Key: balance-subtract-cm1234567890" \
  -H "Content-Type: application/json" \
  -d '{
    "balance":100.00,
    "operation":"subtract",
    "notes":"manual correction"
  }'
```

### 4) 自定义页面一次性启动码（iframe / 新窗口）
需要识别当前用户的 HTTPS 自定义页面，应把菜单项 `auth_mode` 配置为 `exchange_code`。Markdown 页面只支持 `none`。`purchase_subscription_url` 和 `auth_mode: none` 页面只接收匿名 UI 参数，不再接收 JWT 或用户 ID。

每次打开 iframe 或新窗口前，落雪API 前端分别调用：

`POST /api/v1/user/custom-pages/:id/launch`（用户 JWT 认证）

```json
{"theme":"light","lang":"zh-CN","ui_mode":"embedded"}
```

落雪API 会清除管理员配置 URL 中的全部 query 参数，只生成 `theme`、`lang`、`ui_mode`、`src_host`、`s2a_client_id` 和 60 秒一次性 `s2a_launch_code`。它永久不发送 `token`、`user_id` 或完整 `src_url`，且启动 URL 不得记录到日志。

外部页面应立即把 code 交给自己的后端，由后端调用（浏览器 CORS 不开放）：

`POST /api/v1/embedded-pages/exchange`

```json
{"client_id":"payment-page","code":"<s2a_launch_code>"}
```

成功响应只返回：

```json
{"code":0,"message":"success","data":{"user_id":123,"menu_item_id":"payment-page"}}
```

code 只能成功消费一次；过期、伪造、重放或菜单不匹配统一返回 `INVALID_EMBED_LAUNCH_CODE`，Redis 故障返回 HTTP 503。外部后端交换成功后应建立自己的短期会话，前端随即用 `history.replaceState` 清除地址栏中的 code。后续充值仍使用外部服务自己的 Admin API Key。

### 5) 失败处理建议
- 支付成功与充值成功分状态落库
- 回调验签成功后立即标记“支付成功”
- 支付成功但充值失败的订单允许后续重试
- 重试保持相同 `code`，并使用新的 `Idempotency-Key`

### 6) `doc_url` 配置建议
- 查看链接：`https://github.com/3246147496lxt-dotcom/luoxueapi/blob/main/docs/ADMIN_PAYMENT_INTEGRATION_API.md`
- 下载链接：`https://raw.githubusercontent.com/3246147496lxt-dotcom/luoxueapi/main/docs/ADMIN_PAYMENT_INTEGRATION_API.md`

---

## English

### Purpose
This document describes the minimal LuoxueAPI Admin API surface for external payment integrations (for example, `sub2apipay`), including:
- Recharge after payment success
- User lookup
- Manual balance correction
- Purchase page query parameter forwarding

### Base URL
- Production: `https://<your-domain>`
- Beta: `http://<your-server-ip>:8084`

### Authentication
Recommended headers:
- `x-api-key: admin-<64hex>`
- `Content-Type: application/json`
- `Idempotency-Key` for idempotent endpoints

Note: Admin JWT can also access admin routes, but Admin API Key is recommended for server-to-server integration.

### 1) Create and Redeem in one step
`POST /api/v1/admin/redeem-codes/create-and-redeem`

Use case: atomically create a redeem code and redeem it to a target user.

Headers:
- `x-api-key`
- `Idempotency-Key`

Request body:
```json
{
  "code": "s2p_cm1234567890",
  "type": "balance",
  "value": 100.0,
  "user_id": 123,
  "notes": "sub2apipay order: cm1234567890"
}
```

Idempotency behavior:
- Same `code` and same `used_by`: `200`
- Same `code` but different `used_by`: `409`
- Missing `Idempotency-Key`: `400` (`IDEMPOTENCY_KEY_REQUIRED`)

curl example:
```bash
curl -X POST "${BASE}/api/v1/admin/redeem-codes/create-and-redeem" \
  -H "x-api-key: ${KEY}" \
  -H "Idempotency-Key: pay-cm1234567890-success" \
  -H "Content-Type: application/json" \
  -d '{
    "code":"s2p_cm1234567890",
    "type":"balance",
    "value":100.00,
    "user_id":123,
    "notes":"sub2apipay order: cm1234567890"
  }'
```

### 2) Query User (optional pre-check)
`GET /api/v1/admin/users/:id`

```bash
curl -s "${BASE}/api/v1/admin/users/123" \
  -H "x-api-key: ${KEY}"
```

### 3) Balance Adjustment (existing API)
`POST /api/v1/admin/users/:id/balance`

Use case: manual correction with `set` / `add` / `subtract`.

Request body example (`subtract`):
```json
{
  "balance": 100.0,
  "operation": "subtract",
  "notes": "manual correction"
}
```

```bash
curl -X POST "${BASE}/api/v1/admin/users/123/balance" \
  -H "x-api-key: ${KEY}" \
  -H "Idempotency-Key: balance-subtract-cm1234567890" \
  -H "Content-Type: application/json" \
  -d '{
    "balance":100.00,
    "operation":"subtract",
    "notes":"manual correction"
  }'
```

### 4) One-time custom-page launch codes (iframe and new tab)
An HTTPS custom page that needs the current identity must set its menu item's `auth_mode` to `exchange_code`. Markdown pages support only `none`. `purchase_subscription_url` and `auth_mode: none` pages receive anonymous UI context only; JWTs and user IDs are never forwarded.

Before each iframe or new-tab opening, the LuoxueAPI frontend makes a separate authenticated request:

`POST /api/v1/user/custom-pages/:id/launch`

```json
{"theme":"light","lang":"en","ui_mode":"embedded"}
```

LuoxueAPI clears every query parameter from the administrator-configured URL and generates only `theme`, `lang`, `ui_mode`, `src_host`, `s2a_client_id`, and a 60-second one-time `s2a_launch_code`. It never sends `token`, `user_id`, or the full `src_url`, and the launch URL must not be logged.

The external page immediately sends the code to its own backend. That backend performs this server-to-server request (browser CORS is intentionally disabled):

`POST /api/v1/embedded-pages/exchange`

```json
{"client_id":"payment-page","code":"<s2a_launch_code>"}
```

The successful result is intentionally minimal:

```json
{"code":0,"message":"success","data":{"user_id":123,"menu_item_id":"payment-page"}}
```

A code can succeed only once. Expired, forged, replayed, or menu-mismatched codes all return `INVALID_EMBED_LAUNCH_CODE`; Redis outages return HTTP 503. After exchange, the external backend establishes its own short-lived session and the page removes the code from the address bar with `history.replaceState`. Subsequent recharge calls still use the external service's own Admin API Key.

### 5) Failure handling recommendations
- Persist payment success and recharge success as separate states
- Mark payment as successful immediately after verified callback
- Allow retry for orders with payment success but recharge failure
- Keep the same `code` for retry, and use a new `Idempotency-Key`

### 6) Recommended `doc_url`
- View URL: `https://github.com/3246147496lxt-dotcom/luoxueapi/blob/main/docs/ADMIN_PAYMENT_INTEGRATION_API.md`
- Download URL: `https://raw.githubusercontent.com/3246147496lxt-dotcom/luoxueapi/main/docs/ADMIN_PAYMENT_INTEGRATION_API.md`
