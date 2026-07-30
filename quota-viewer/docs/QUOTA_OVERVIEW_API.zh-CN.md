# 落雪额度查看器只读接口契约（提案）

状态：P0 契约草案

接口版本：`schema_version = 1`

## 1. 目标与边界

为独立额度查看器提供一个覆盖整账户的只读快照：

```http
GET /api/v1/quota/overview
```

接口回答：

- 当前账户有哪些启用 API Key；
- 每个 Key 属于余额计费还是会员计费；
- 各计费分组当前是否可用；
- 账户余额、今日消费与本月累计消费；
- 每份会员唯一的 7 天额度、剩余量、准确重置时间和到期时间；
- 按会员聚合的当前 7 天额度周期真实请求数与 Token 用量。

接口不返回：

- 完整 API Key；
- 单请求明细、模型分布或预计可用 Token/请求数；
- Codex 或其他上游账号池额度；
- 由余额反推的预计 Token；
- 邮箱、第三方身份、支付资料等与额度查看无关的个人信息；
- 任何写操作能力。

## 2. 认证

查看器不得保存网站 JWT 或具有消费能力的 API Key。必须使用独立的设备只读授权：

- `client_id = luoxue-quota-viewer`
- `aud = luoxue-quota-api`
- `scope = quota:read`
- Access Token 有效期建议 10–15 分钟；
- Refresh Token 每次使用时轮换，服务端只保存哈希；
- 查看器刷新时使用 `candidate-v1`：客户端先在系统钥匙串持久化一次性的
  `rotation_id` 与候选 Refresh Token，再发送请求；超时或 `503` 时必须重试
  完全相同的组合，不能降级旧协议或生成新候选；
- 服务端对同一 predecessor、`rotation_id` 和候选 Token 哈希提供 10 分钟幂等
  恢复窗口，恢复时不延长候选 Token 的有效期；不同组合仍按重放攻击处理；
- 每台设备可单独撤销；
- 通过系统浏览器完成确认，应用内不接收密码、验证码或 API Key；
- 该令牌不能访问 API Key 增删改、充值、会员购买或任何推理接口。

服务端必须同时校验签发者、受众、scope、设备状态和用户归属。

## 3. 请求

```http
GET /api/v1/quota/overview?timezone=Asia%2FShanghai
Authorization: Bearer <quota-viewer-access-token>
Accept: application/json
```

参数：

| 参数 | 必填 | 说明 |
|---|---:|---|
| `timezone` | 否 | 决定“今日消费”和“本月消费”的日历边界，同时用于时间标签展示；会员 7 天周期仍只以开通时间锚定。所有权威时间返回 RFC 3339 时间点。非法值返回 `400`。 |

## 4. 成功响应示例

额度金额使用十进制定点字符串，避免 JavaScript 浮点误差。`used_percent` 仅用于显示，不能参与账务计算。

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "schema_version": 1,
    "request_id": "qov_01K1B5MM0VA9E9M6YQ6J41F80S",
    "generated_at": "2026-07-29T07:30:02Z",
    "as_of": "2026-07-29T07:30:00Z",
    "fresh_until": "2026-07-29T07:35:00Z",
    "display_timezone": "Asia/Shanghai",
    "freshness": "fresh",
    "coverage": {
      "included": ["wallet", "account_spend_today", "account_spend_month_to_date", "api_key_billing_groups", "subscription_7d", "subscription_period_usage"],
      "excluded": ["routing", "rpm", "concurrency", "upstream_quota", "codex_quota"]
    },
    "account": {
      "display_label": "pu***@example.com",
      "data_scope": "all_enabled_api_keys",
      "quota_state": "partially_restricted",
      "can_make_request": null,
      "usable_group_count": 1,
      "blocked_group_count": 1,
      "unknown_group_count": 0,
      "primary_issue": {
        "scope_type": "billing_group",
        "scope_id": "grp_member_pro",
        "reason_code": "subscription_weekly_exhausted",
        "recommended_action": "manage_api_keys",
        "recovers_at": "2026-08-01T09:30:00Z"
      }
    },
    "wallet": {
      "unit": "snow_credit",
      "state": "available",
      "available": "128.6400000000",
      "reserved": "0.0000000000",
      "today_spend": "1.1600000000",
      "month_spend": "10.7400000000",
      "balance_billed_key_count": 2
    },
    "billing_groups": [
      {
        "id": "grp_standard",
        "display_name": "标准计费",
        "billing_mode": "balance",
        "state": "usable",
        "reason_code": null,
        "recommended_action": "none",
        "fallback_policy": "none",
        "resource_ref": {
          "kind": "wallet",
          "id": null
        },
        "keys": [
          {
            "id": "key_101",
            "name": "工作电脑",
            "masked_key": "sk-••••7A9C",
            "state": "usable"
          }
        ]
      },
      {
        "id": "grp_member_pro",
        "display_name": "Pro 会员",
        "billing_mode": "subscription",
        "state": "blocked",
        "reason_code": "subscription_weekly_exhausted",
        "recommended_action": "manage_api_keys",
        "fallback_policy": "none",
        "resource_ref": {
          "kind": "subscription",
          "id": "sub_8"
        },
        "keys": [
          {
            "id": "key_202",
            "name": "Codex",
            "masked_key": "sk-••••42FD",
            "state": "blocked"
          }
        ]
      }
    ],
    "subscriptions": [
      {
        "id": "sub_8",
        "group_id": "grp_member_pro",
        "name": "Pro 会员",
        "status": "active",
        "starts_at": "2026-07-25T09:30:00Z",
        "expires_at": "2026-08-25T09:30:00Z",
        "weekly_window": {
          "kind": "7d_from_subscription_start",
          "state": "exhausted",
          "anchor_at": "2026-07-25T09:30:00Z",
          "period_start": "2026-07-25T09:30:00Z",
          "period_end": "2026-08-01T09:30:00Z",
          "resets_at": "2026-08-01T09:30:00Z",
          "limit": "200.0000000000",
          "used": "200.0000000000",
          "remaining": "0.0000000000",
          "used_percent": 100
        },
        "period_usage": {
          "state": "available",
          "observed_until": "2026-07-29T07:30:00Z",
          "bucket_kind": "anchored_24h",
          "total_requests": 279,
          "total_tokens": 921000,
          "points": [
            {
              "index": 1,
              "start_at": "2026-07-25T09:30:00Z",
              "end_at": "2026-07-26T09:30:00Z",
              "state": "complete",
              "requests": 58,
              "cache_hit_tokens": 92000,
              "cache_miss_tokens": 41000,
              "output_tokens": 49000,
              "total_tokens": 182000
            },
            {
              "index": 2,
              "start_at": "2026-07-26T09:30:00Z",
              "end_at": "2026-07-27T09:30:00Z",
              "state": "complete",
              "requests": 66,
              "cache_hit_tokens": 105000,
              "cache_miss_tokens": 52000,
              "output_tokens": 57000,
              "total_tokens": 214000
            },
            {
              "index": 3,
              "start_at": "2026-07-27T09:30:00Z",
              "end_at": "2026-07-28T09:30:00Z",
              "state": "complete",
              "requests": 73,
              "cache_hit_tokens": 118000,
              "cache_miss_tokens": 61000,
              "output_tokens": 67000,
              "total_tokens": 246000
            },
            {
              "index": 4,
              "start_at": "2026-07-28T09:30:00Z",
              "end_at": "2026-07-29T09:30:00Z",
              "state": "partial",
              "requests": 82,
              "cache_hit_tokens": 132000,
              "cache_miss_tokens": 67000,
              "output_tokens": 80000,
              "total_tokens": 279000
            },
            {
              "index": 5,
              "start_at": "2026-07-29T09:30:00Z",
              "end_at": "2026-07-30T09:30:00Z",
              "state": "future",
              "requests": null,
              "cache_hit_tokens": null,
              "cache_miss_tokens": null,
              "output_tokens": null,
              "total_tokens": null
            },
            {
              "index": 6,
              "start_at": "2026-07-30T09:30:00Z",
              "end_at": "2026-07-31T09:30:00Z",
              "state": "future",
              "requests": null,
              "cache_hit_tokens": null,
              "cache_miss_tokens": null,
              "output_tokens": null,
              "total_tokens": null
            },
            {
              "index": 7,
              "start_at": "2026-07-31T09:30:00Z",
              "end_at": "2026-08-01T09:30:00Z",
              "state": "future",
              "requests": null,
              "cache_hit_tokens": null,
              "cache_miss_tokens": null,
              "output_tokens": null,
              "total_tokens": null
            }
          ]
        },
        "next_event": {
          "kind": "reset",
          "at": "2026-08-01T09:30:00Z"
        }
      }
    ],
    "actions": {
      "recharge_url": "https://example.com/purchase",
      "manage_keys_url": "https://example.com/keys",
      "manage_subscriptions_url": "https://example.com/subscriptions"
    },
    "warnings": []
  }
}
```

## 5. 字段语义

### 5.1 快照与新鲜度

| 字段 | 语义 |
|---|---|
| `as_of` | 快照中最旧的权威数据时间，不是 HTTP 响应生成时间。 |
| `fresh_until` | 在该时间之前客户端可把快照标为当前数据；超过后必须标记为上次数据并刷新。 |
| `freshness` | `fresh`、`stale` 或 `unknown`。只要关键来源无法确认，就不能返回 `fresh`。 |
| `warnings` | 非致命的普通语言不可见警告代码；客户端根据代码显示降级状态。 |

客户端缓存快照时必须同时缓存 `as_of` 和 `fresh_until`。请求失败不得把旧值改成 0。

### 5.2 整账户状态

`account.quota_state` 只描述本接口覆盖的余额和会员额度，不保证某次请求一定成功。`account.can_make_request` 在 v1 固定为 `null`。

- `all_resources_available`：所有存在且有启用 Key 的计费分组均未被额度阻塞；
- `partially_restricted`：至少一个分组额度正常，至少一个分组受限或未知；
- `all_resources_restricted`：所有启用 Key 所属分组均被额度阻塞；
- `no_enabled_keys`：账户有权益但没有启用 Key；
- `unknown`：无法确认真实状态。

整账户额度状态不是“余额是否大于 0”的别名。服务端必须由各计费分组状态聚合。模型、路由、RPM、并发和上游健康度明确列入 `coverage.excluded`，客户端不得把额度状态渲染为无条件的“可正常调用”。

### 5.3 计费分组

`billing_mode`：

- `balance`：该组 Key 只使用账户余额；
- `subscription`：该组 Key 只使用 `resource_ref` 指向的会员额度。

`fallback_policy` 在 P0 固定为 `none`。会员分组耗尽后，服务端和客户端都不得暗示自动改扣余额。

`billing_groups[].state`：

- `usable`
- `partially_usable`
- `blocked`
- `no_enabled_keys`
- `unknown`

分组状态还应考虑 Key 是否启用、Key 自身限额和所关联资源状态，不能只看余额或会员剩余。

### 5.4 会员 7 天周期

会员只有 `weekly_window`，不存在 `daily_window` 或 `monthly_window`。

对于 `as_of >= anchor_at`：

```text
N = floor((as_of - anchor_at) / (7 × 24 小时))
period_start = anchor_at + N × 7 天
period_end = anchor_at + (N + 1) × 7 天
resets_at = period_end
```

规则：

- `anchor_at` 必须等于当前连续会员期限的准确 `starts_at`；
- 时间计算使用 UTC 时间点和连续 `7 × 24` 小时，不按自然周、时区零点或夏令时重排；
- 未中断续费只延长 `expires_at`，不改变 `anchor_at`；
- 过期后重新开通或更换套餐，以新的 `starts_at` 建立新锚点；
- `period_start` 和 `period_end` 采用半开区间 `[start, end)`；
- 如果会员到期早于下一次重置，`next_event.kind` 必须为 `expiry`；
- 客户端不得自行推算周期，只显示服务端值。

`weekly_window.state`：

- `active`
- `exhausted`
- `unknown`

`used_percent`：

- 仅在 `limit > 0` 且数据可确认时返回数字；
- 语义固定为已用百分比；
- 展示值封顶 100，但 `used` 保留真实值；
- 未知时返回 `null`，不能返回 0。

### 5.5 会员本周期真实用量

`subscriptions[].period_usage` 与当前 `weekly_window` 使用完全相同的周期边界。它只描述该周期内已经发生的历史使用，不代表可用额度。

- `observed_until = min(as_of, weekly_window.period_end, expires_at)`；
- 总计统计范围固定为 `[weekly_window.period_start, observed_until)`，不得另建滚动 7 日窗口；
- `bucket_kind` 在 v1 固定为 `anchored_24h`，从 `weekly_window.period_start` 起连续切分 7 个 24 小时桶，不按 UTC 或本地自然日重排；
- `points` 固定返回 7 个按 `index` 升序排列的桶；客户端只绘制 `complete` 和 `partial`，不为尚未发生的 `future` 桶绘制空柱；
- `complete` 与 `partial` 查询成功但无用量时返回真实 0；`future` 的请求数与 Token 字段返回 `null`；
- 显示时区只用于格式化每个桶的 `start_at` 标签，不能改变分桶边界；
- `weekly_window.state = unknown` 或周期边界缺失时，`period_usage.state` 必须为 `unknown`，客户端不得自行推算；
- 查询失败或无法确认时返回 `state = unknown`，`total_requests`、`total_tokens` 和 `points` 为 `null`，不能伪装成 0；
- 统计只包含 `usage_logs.subscription_id` 指向当前会员、用户归属一致且 `created_at` 落在半开区间内的请求；
- `cache_hit_tokens = cache_read_tokens`；
- `cache_miss_tokens = input_tokens + cache_creation_tokens`；
- `total_tokens = cache_hit_tokens + cache_miss_tokens + output_tokens`；
- 该统计最终一致，允许因异步用量落库而短暂滞后；
- 聚合失败不能拖垮余额和会员额度快照，应单独返回 `period_usage.state = unknown` 并附 warning。

### 5.6 会员状态

`subscriptions[].status`：

- `active`
- `expired`
- `suspended`
- `revoked`
- `unknown`

过期、暂停或撤销状态必须优先于窗口剩余量，不能因旧窗口仍有剩余而显示可用。

### 5.7 原因和操作

`reason_code` 最小集合：

- `wallet_empty`
- `subscription_weekly_exhausted`
- `subscription_expired`
- `subscription_suspended`
- `subscription_revoked`
- `no_active_subscription`
- `api_key_disabled`
- `api_key_limit_exhausted`
- `data_stale`
- `data_unavailable`
- `authorization_revoked`

`recommended_action`：

- `none`
- `recharge`
- `wait_for_reset`
- `renew_subscription`
- `manage_api_keys`
- `retry`
- `reconnect`
- `contact_support`

充值只能对应 `wallet_empty`。会员额度耗尽不得返回 `recharge`。

## 6. HTTP 与缓存行为

| 状态码 | 场景 |
|---:|---|
| `200` | 返回完整或明确标记降级的整账户快照。 |
| `400` | 非法时区或不支持的请求参数。 |
| `401` | Access Token 无效、过期或设备会话失效。 |
| `403` | audience 或 scope 不匹配。 |
| `429` | 设备刷新过于频繁，返回 `Retry-After`。 |
| `503` | 无法生成快照且服务端没有可安全返回的旧快照。 |

响应头：

```http
Cache-Control: private, no-store
Vary: Authorization
```

服务端可以按用户做 10–15 秒请求合并或内存短缓存，但不得让共享代理缓存用户数据。

客户端：

- 默认 5 分钟自动刷新，用户可选择 15 或 30 分钟；
- 手动刷新需要防抖；
- 网络失败时保留本地加密快照并标记“上次数据”；
- 退出账户或设备授权被撤销时清除本地快照。

## 7. 发布门槛与实现差距

该契约不能只做展示层换算。发布前，计费拦截、额度累计、重置和接口展示必须使用同一套 7 天窗口。

当前实现需要修正：

- 新会员创建时 `weekly_window_start` 可能为空，实际从首次使用开始计时；
- 过期会员重新开通时窗口起点可能被截到当天零点；
- 现有分组仍支持日、周、月多个限额；
- 旧接口和前端类型对订阅进度的 JSON 结构不一致。

P0 发布前必须保证：

1. 新会员开通时立即初始化准确的 `anchor_at = starts_at`；
2. 计费拦截和用量累计使用 `[anchor + N×7d, anchor + (N+1)×7d)`；
3. 日额度和月额度不再参与该会员分组的拦截；
4. 未中断续费不改变锚点；
5. 过期后重新开通使用准确的新生效时间，不截到零点；
6. Redis、数据库和接口快照对同一请求返回一致的周期与用量；
7. 任一一致性条件无法确认时，接口返回 `unknown`，不能猜测为可用；
8. 用量聚合失败时只降级 `period_usage`，不得影响余额和会员窗口的可用快照。

## 8. 合同与安全测试

至少覆盖：

- 错误 audience、缺少 `quota:read`、过期令牌和已撤销设备；
- 用户 A 无法读取用户 B 的余额、分组、会员或 Key 摘要；
- 响应、日志和本地缓存均不包含完整 API Key；
- 会员开通瞬间、周期结束前 1 毫秒和周期结束时的半开区间边界；
- `as_of = period_start` 时本周期 totals 为真实 0，所有未来桶均为 `null`；
- `as_of = period_end` 时立即进入新周期，旧周期数据不得泄漏到新周期；
- 非零点开通、跨月、跨年和夏令时不改变 7 个锚定的连续 24 小时桶；
- 跨越多个 7 天周期后仍以原开通时间为锚点；
- 未中断续费、过期重开和更换套餐；
- 会员到期早于下一次重置；
- 会员耗尽但余额充足时，会员组仍为 `blocked`，余额组为 `usable`；
- 余额为 0 但会员有剩余时，余额组为 `blocked`，会员组为 `usable`；
- 无启用 Key、Key 被禁用、Key 自身额度耗尽；
- Redis 与数据库不一致时返回权威值或 `unknown`；
- 首次拉取失败不产生 0 余额或 0% 假数据；
- 旧快照正确携带 `as_of`、`fresh_until` 和 `stale`；
- `used > limit` 时保留真实 `used`，`remaining = 0`，展示百分比封顶 100；
- 充值操作永远不作为会员额度耗尽的恢复方式。
- 本周期数据按 `user_id + subscription_id + [period_start, as_of)` 隔离，不能混入其他用户、其他会员、其他周期或余额计费请求；
- `period_usage` 覆盖零用量补零、周期边界、查询失败和异步落库滞后；
- `period_usage.state = unknown` 时 totals 为 `null`，客户端显示未知而不是 0。
