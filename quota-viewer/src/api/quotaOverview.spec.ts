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
      },
      monthly_window: {
        kind: '30d_from_subscription_start',
        state: 'active',
        anchor_at: '2026-07-25T09:30:00Z',
        period_start: '2026-07-25T09:30:00Z',
        period_end: '2026-08-24T09:30:00Z',
        resets_at: '2026-08-24T09:30:00Z',
        limit: '800.0000000000',
        used: '208.0000000000',
        remaining: '592.0000000000',
        used_percent: 26
      }
    }
  ]
}

describe('quota overview adapter', () => {
  it('puts the newest current membership ahead of older subscription groups', () => {
    const multipleMemberships = structuredClone(fixture)
    multipleMemberships.subscriptions = [
      {
        ...structuredClone(fixture.subscriptions[0]),
        id: '100',
        group_id: '10',
        name: '很久以前的会员',
        status: 'expired',
        starts_at: '2025-01-01T00:00:00Z',
        expires_at: '2025-02-01T00:00:00Z'
      },
      {
        ...structuredClone(fixture.subscriptions[0]),
        id: '200',
        group_id: '20',
        name: '最新会员',
        starts_at: '2026-07-29T09:30:00Z',
        expires_at: '2026-08-29T09:30:00Z'
      }
    ]

    const overview = adaptQuotaOverview(multipleMemberships)

    expect(overview.quotas.map((quota) => quota.name)).toEqual([
      '最新会员',
      '很久以前的会员'
    ])
    expect(overview.quotas.map((quota) => quota.isCurrentMembership)).toEqual([
      true,
      false
    ])
  })

  it('uses the newest start time when more than one membership is current', () => {
    const multipleCurrentMemberships = structuredClone(fixture)
    multipleCurrentMemberships.subscriptions = [
      {
        ...structuredClone(fixture.subscriptions[0]),
        id: '100',
        name: '较早会员',
        starts_at: '2026-07-01T09:30:00Z',
        expires_at: '2026-08-01T09:30:00Z'
      },
      {
        ...structuredClone(fixture.subscriptions[0]),
        id: '200',
        name: '较新会员',
        starts_at: '2026-07-29T09:30:00Z',
        expires_at: '2026-08-29T09:30:00Z'
      }
    ]

    const quotas = adaptQuotaOverview(multipleCurrentMemberships).quotas
    expect(quotas[0].name).toBe('较新会员')
    expect(quotas.every((quota) => quota.isCurrentMembership)).toBe(true)
  })

  it('does not let a malformed active record outrank a valid current membership', () => {
    const memberships = structuredClone(fixture)
    memberships.subscriptions = [
      {
        ...structuredClone(fixture.subscriptions[0]),
        id: '300',
        name: '异常会员记录',
        starts_at: 'not-a-date',
        expires_at: '2027-01-01T00:00:00Z'
      },
      {
        ...structuredClone(fixture.subscriptions[0]),
        id: '200',
        name: '有效当前会员'
      }
    ]

    const quotas = adaptQuotaOverview(memberships).quotas
    expect(quotas[0].name).toBe('有效当前会员')
    expect(quotas[0].isCurrentMembership).toBe(true)
    expect(quotas[1].isCurrentMembership).toBe(false)
  })

  it('falls back to the newest historical membership when none is current', () => {
    const historicalMemberships = structuredClone(fixture)
    historicalMemberships.subscriptions = [
      {
        ...structuredClone(fixture.subscriptions[0]),
        id: '100',
        name: '更早历史会员',
        status: 'expired',
        starts_at: '2024-01-01T00:00:00Z',
        expires_at: '2024-02-01T00:00:00Z'
      },
      {
        ...structuredClone(fixture.subscriptions[0]),
        id: '200',
        name: '最近历史会员',
        status: 'expired',
        starts_at: '2025-01-01T00:00:00Z',
        expires_at: '2025-02-01T00:00:00Z'
      }
    ]

    const quotas = adaptQuotaOverview(historicalMemberships).quotas
    expect(quotas[0].name).toBe('最近历史会员')
    expect(quotas.every((quota) => !quota.isCurrentMembership)).toBe(true)
    expect(quotas[0].expiresAt).toBe('2025-02-01T00:00:00Z')
  })

  it('keeps authoritative quota values while locally degrading absent period usage', () => {
    const overview = adaptQuotaOverview(fixture)

    expect(overview.balance).toBe('128.6400000000')
    expect(overview.quotas[0]).toMatchObject({
      id: '9007199254740995',
      expiresAt: '2026-08-25T09:30:00Z',
      isCurrentMembership: true,
      usedPercent: 68,
      usedLabel: '已用 136.20 积分',
      remainingLabel: '剩余 63.80 积分',
      resetsAt: '2026-08-01T09:30:00Z',
      monthlyRemainingPercent: 74,
      usageAvailable: false,
      totalTokens: null,
      totalRequests: null,
      points: []
    })
  })

  it('uses the authoritative anchored 30-day monthly window', () => {
    const quota = adaptQuotaOverview(fixture).quotas[0]

    expect(fixture.subscriptions[0].monthly_window.kind).toBe(
      '30d_from_subscription_start'
    )
    expect(fixture.subscriptions[0].monthly_window.period_end).toBe(
      '2026-08-24T09:30:00Z'
    )
    expect(quota.monthlyRemainingPercent).toBe(74)
  })

  it('fails closed when the authoritative monthly window is absent', () => {
    const missingMonthly = structuredClone(fixture)
    missingMonthly.subscriptions[0].monthly_window = null as never

    const quota = adaptQuotaOverview(missingMonthly).quotas[0]
    expect(quota.state).toBe('unknown')
    expect(quota.monthlyRemainingPercent).toBeNull()
  })

  it.each([
    ['weekly_window', 'unknown', null, 'unknown'],
    ['weekly_window', 'exhausted', 100, 'exhausted'],
    ['weekly_window', 'active', 92, 'warning'],
    ['monthly_window', 'unknown', null, 'unknown'],
    ['monthly_window', 'exhausted', 100, 'exhausted'],
    ['monthly_window', 'active', 92, 'warning']
  ] as const)(
    'includes %s %s state in the combined membership status',
    (windowKey, windowState, usedPercent, expectedState) => {
      const overview = structuredClone(fixture)
      overview.subscriptions[0][windowKey].state = windowState
      overview.subscriptions[0][windowKey].used_percent = usedPercent as never

      expect(adaptQuotaOverview(overview).quotas[0].state).toBe(expectedState)
    }
  )

  it('prefers unknown when one window is exhausted but the other is unverified', () => {
    const mixed = structuredClone(fixture)
    mixed.subscriptions[0].weekly_window.state = 'exhausted'
    mixed.subscriptions[0].weekly_window.used_percent = 100
    mixed.subscriptions[0].monthly_window.state = 'unknown'
    mixed.subscriptions[0].monthly_window.used_percent = null as never

    expect(adaptQuotaOverview(mixed).quotas[0].state).toBe('unknown')
  })

  it('keeps only valid expiry timestamps and derives current membership at as_of', () => {
    const malformed = structuredClone(fixture)
    malformed.subscriptions[0].expires_at = 'not-a-date'

    const quota = adaptQuotaOverview(malformed).quotas[0]
    expect(quota.expiresAt).toBeNull()
    expect(quota.isCurrentMembership).toBe(false)
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

  it('treats malformed available usage as unknown instead of filling nulls with zero', () => {
    const malformed = structuredClone(fixture)
    Object.assign(malformed.subscriptions[0], {
      period_usage: {
        state: 'available',
        total_requests: 1,
        total_tokens: 1,
        points: [
          {
            index: 1,
            start_at: '2026-07-25T09:30:00Z',
            end_at: '2026-07-26T09:30:00Z',
            state: 'complete',
            requests: null,
            cache_hit_tokens: null,
            cache_miss_tokens: null,
            output_tokens: null,
            total_tokens: null
          }
        ]
      }
    })

    const quota = adaptQuotaOverview(malformed).quotas[0]
    expect(quota.usageAvailable).toBe(false)
    expect(quota.totalTokens).toBeNull()
    expect(quota.totalRequests).toBeNull()
    expect(quota.points).toEqual([])
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
    expect(quota.resetsAt).toBeNull()
    expect(quota.resetLabel).toBe('重置时间待确认')
  })

  it('does not present an unknown subscription status as expired', () => {
    const unknown = structuredClone(fixture)
    unknown.subscriptions[0].status = 'unknown'

    const quota = adaptQuotaOverview(unknown).quotas[0]
    expect(quota.state).toBe('unknown')
    expect(quota.resetsAt).toBeNull()
  })

  it('fails closed when the reset timestamp is malformed', () => {
    const malformed = structuredClone(fixture)
    malformed.subscriptions[0].weekly_window.resets_at = 'not-a-date'

    const quota = adaptQuotaOverview(malformed).quotas[0]
    expect(quota.resetsAt).toBeNull()
    expect(quota.resetLabel).toBe('重置时间待确认')
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
      expect(quota.resetsAt).toBeNull()
      expect(quota.periodStartLabel).toBeNull()
      expect(quota.periodEndLabel).toBeNull()
      expect(quota.isCurrentMembership).toBe(false)
    }
  )
})
