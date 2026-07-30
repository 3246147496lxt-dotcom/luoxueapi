import { describe, expect, it } from 'vitest'
import { adaptQuotaOverview } from './quotaOverview'

const fixture = {
  schema_version: 1,
  generated_at: '2026-07-30T10:30:00Z',
  as_of: '2026-07-30T10:30:00Z',
  fresh_until: '2026-07-30T10:35:00Z',
  display_timezone: 'Asia/Shanghai',
  freshness: 'fresh',
  account: {
    display_label: 'p***@example.com',
    data_scope: 'all_enabled_api_keys',
    quota_state: 'all_resources_available'
  },
  wallet: {
    unit: 'snow_credit',
    state: 'available',
    available: '128.6400000000',
    reserved: '0.0000000000',
    balance_billed_key_count: 2
  },
  subscriptions: [
    {
      id: '9007199254740995',
      group_id: '9007199254740994',
      name: 'Pro 会员',
      status: 'active',
      starts_at: '2026-07-25T09:30:00Z',
      expires_at: '2026-08-25T09:30:00Z',
      weekly_window: {
        kind: '7d_from_subscription_start',
        state: 'active',
        anchor_at: '2026-07-25T09:30:00Z',
        period_start: '2026-07-25T09:30:00Z',
        period_end: '2026-08-01T09:30:00Z',
        resets_at: '2026-08-01T09:30:00Z',
        limit: '200.0000000000',
        used: '136.2000000000',
        remaining: '63.8000000000',
        used_percent: 68
      }
    }
  ]
}

describe('quota overview adapter', () => {
  it('keeps authoritative quota values while locally degrading absent period usage', () => {
    const overview = adaptQuotaOverview(fixture)

    expect(overview.balance).toBe('128.6400000000')
    expect(overview.quotas[0]).toMatchObject({
      id: '9007199254740995',
      usedPercent: 68,
      usedLabel: '已用 ❄136.20',
      remainingLabel: '剩余 ❄63.80',
      usageAvailable: false,
      totalTokens: null,
      totalRequests: null,
      points: []
    })
  })

  it('uses server totals and excludes future buckets from the chart', () => {
    const withUsage = structuredClone(fixture)
    Object.assign(withUsage.subscriptions[0], {
      period_usage: {
        state: 'available',
        total_requests: 58,
        total_tokens: 182000,
        points: [
          {
            index: 1,
            start_at: '2026-07-25T09:30:00Z',
            end_at: '2026-07-26T09:30:00Z',
            state: 'complete',
            requests: 58,
            cache_hit_tokens: 92000,
            cache_miss_tokens: 41000,
            output_tokens: 49000,
            total_tokens: 182000
          },
          {
            index: 2,
            start_at: '2026-07-26T09:30:00Z',
            end_at: '2026-07-27T09:30:00Z',
            state: 'future',
            requests: null,
            cache_hit_tokens: null,
            cache_miss_tokens: null,
            output_tokens: null,
            total_tokens: null
          }
        ]
      }
    })

    const quota = adaptQuotaOverview(withUsage).quotas[0]
    expect(quota.totalTokens).toBe(182000)
    expect(quota.totalRequests).toBe(58)
    expect(quota.points).toHaveLength(1)
    expect(quota.points[0].state).toBe('complete')
  })

  it('does not render an unknown weekly window as zero percent', () => {
    const unknown = structuredClone(fixture)
    unknown.subscriptions[0].weekly_window.state = 'unknown'
    unknown.subscriptions[0].weekly_window.used_percent = null as never
    unknown.subscriptions[0].weekly_window.used = null as never
    unknown.subscriptions[0].weekly_window.remaining = null as never

    const quota = adaptQuotaOverview(unknown).quotas[0]
    expect(quota.state).toBe('unknown')
    expect(quota.usedPercent).toBeNull()
    expect(quota.usedLabel).toBe('已用 —')
  })

  it('does not present an unknown subscription status as expired', () => {
    const unknown = structuredClone(fixture)
    unknown.subscriptions[0].status = 'unknown'

    const quota = adaptQuotaOverview(unknown).quotas[0]
    expect(quota.state).toBe('unknown')
  })

  it.each([
    ['expired', '到期'],
    ['suspended', '暂停'],
    ['revoked', '撤销']
  ])(
    'maps %s membership lifecycle without inventing a reset or period',
    (status, expectedCopy) => {
      const inactive = structuredClone(fixture)
      inactive.subscriptions[0].status = status
      inactive.subscriptions[0].weekly_window.state = 'unknown'
      inactive.subscriptions[0].weekly_window.period_start = null as never
      inactive.subscriptions[0].weekly_window.period_end = null as never
      inactive.subscriptions[0].weekly_window.resets_at = null as never

      const quota = adaptQuotaOverview(inactive).quotas[0]
      expect(quota.state).toBe('expired')
      expect(quota.membershipStatus).toBe(status)
      expect(quota.statusDetailLabel).toContain(expectedCopy)
      expect(quota.resetLabel).toBe(quota.statusDetailLabel)
      expect(quota.resetLabel).not.toContain('重置')
      expect(quota.periodStartLabel).toBeNull()
      expect(quota.periodEndLabel).toBeNull()
    }
  )
})
