import { nextTick, ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Account, AccountUsageInfo } from '@/types'

const { getUsage } = vi.hoisted(() => ({
  getUsage: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      getUsage
    }
  }
}))

vi.mock('@/utils/usageLoadQueue', () => ({
  enqueueUsageRequest: (
    _account: Account,
    fetchUsage: () => Promise<AccountUsageInfo>
  ) => fetchUsage()
}))

import {
  getAccountUsageHealthSnapshot,
  invalidateAccountUsageHealthSnapshot,
  isAccountUsageHealthSnapshotFresh,
  normalizeAccountUsageHealth,
  publishAccountUsage,
  requestAccountUsage,
  useAccountUsageHealthSnapshot
} from '../useAccountUsageHealth'

const makeUsage = (
  overrides: Partial<AccountUsageInfo> = {}
): AccountUsageInfo => ({
  updated_at: null,
  five_hour: null,
  seven_day: null,
  seven_day_sonnet: null,
  ...overrides
})

const makeAccount = (id: number): Account => ({
  id,
  name: `account-${id}`,
  platform: 'openai',
  type: 'oauth',
  proxy_id: null,
  concurrency: 1,
  priority: 1,
  status: 'active',
  error_message: null,
  last_used_at: null,
  expires_at: null,
  auto_pause_on_expired: true,
  created_at: '2026-07-25T00:00:00Z',
  updated_at: '2026-07-25T00:00:00Z',
  schedulable: true,
  rate_limited_at: null,
  rate_limit_reset_at: null,
  overload_until: null,
  temp_unschedulable_until: null,
  temp_unschedulable_reason: null,
  session_window_start: null,
  session_window_end: null,
  session_window_status: null
})

describe('useAccountUsageHealth', () => {
  beforeEach(() => {
    getUsage.mockReset()
  })

  it('normalizes OAuth and Gemini windows into the most constrained window', () => {
    const health = normalizeAccountUsageHealth(makeUsage({
      five_hour: {
        utilization: 25,
        resets_at: '2026-07-25T12:00:00Z',
        remaining_seconds: 3600
      },
      seven_day_sonnet: {
        utilization: 80,
        resets_at: '2026-07-31T12:00:00Z',
        remaining_seconds: 3600
      },
      gemini_flash_minute: {
        utilization: 105,
        resets_at: '2026-07-25T12:01:00Z',
        remaining_seconds: 30
      }
    }))

    expect(health.mostConstrainedWindow).toEqual({
      label: 'Gemini Flash 1m',
      utilization: 105,
      resetsAt: '2026-07-25T12:01:00Z'
    })
    expect(health.maxUtilization).toBe(105)
    expect(health.exhausted).toBe(true)
  })

  it('normalizes Antigravity and converts Grok remaining quota to used percent', () => {
    const health = normalizeAccountUsageHealth(makeUsage({
      antigravity_quota: {
        'gemini-3-pro-high': {
          utilization: 91,
          reset_time: '2026-07-25T13:00:00Z'
        }
      },
      grok_request_quota: {
        limit: 100,
        remaining: 20,
        reset_at: '2026-07-25T14:00:00Z'
      },
      grok_token_quota: {
        limit: 200,
        remaining: 0,
        reset_unix: 1_784_989_200
      }
    }))

    expect(health.mostConstrainedWindow).toEqual({
      label: 'Grok tokens',
      utilization: 100,
      resetsAt: new Date(1_784_989_200 * 1000).toISOString()
    })
    expect(health.exhausted).toBe(true)
  })

  it('derives authorization health and exhausted AI credits from machine-readable fields', () => {
    const unauthenticated = normalizeAccountUsageHealth(makeUsage({
      error_code: 'unauthenticated'
    }))
    const forbidden = normalizeAccountUsageHealth(makeUsage({
      error_code: 'forbidden',
      forbidden_type: 'validation'
    }))
    const exhaustedCredits = normalizeAccountUsageHealth(makeUsage({
      ai_credits: [{
        credit_type: 'GOOGLE_ONE_AI',
        amount: 5,
        minimum_balance: 5
      }]
    }))

    expect(unauthenticated.needsReauth).toBe(true)
    expect(forbidden).toMatchObject({
      isForbidden: true,
      needsVerify: true
    })
    expect(exhaustedCredits).toMatchObject({
      exhausted: true,
      maxUtilization: 100,
      mostConstrainedWindow: {
        label: 'AI credits',
        utilization: 100,
        resetsAt: null
      }
    })
  })

  it('publishes one readonly reactive snapshot for a deduplicated cold request', async () => {
    const account = makeAccount(8_001_001)
    const accountId = ref(account.id)
    const snapshot = useAccountUsageHealthSnapshot(accountId)
    getUsage.mockResolvedValue(makeUsage({
      seven_day: {
        utilization: 68,
        resets_at: '2026-07-31T12:00:00Z',
        remaining_seconds: 3600
      }
    }))

    const first = requestAccountUsage(account)
    const second = requestAccountUsage(account)
    expect(first).toBe(second)

    await first.promise
    await nextTick()

    expect(getUsage).toHaveBeenCalledTimes(1)
    expect(snapshot.value?.health.maxUtilization).toBe(68)
    expect(getAccountUsageHealthSnapshot(account.id)).toBe(snapshot.value)
    expect(Object.isFrozen(snapshot.value)).toBe(true)
    expect(Object.isFrozen(snapshot.value?.health)).toBe(true)
    expect(isAccountUsageHealthSnapshotFresh(account.id, 60_000)).toBe(true)
  })

  it('does not let an older automatic request overwrite a newer active result', async () => {
    const account = makeAccount(8_001_002)
    let resolveAutomatic!: (usage: AccountUsageInfo) => void
    const automaticUsage = new Promise<AccountUsageInfo>((resolve) => {
      resolveAutomatic = resolve
    })
    getUsage
      .mockReturnValueOnce(automaticUsage)
      .mockResolvedValueOnce(makeUsage({
        five_hour: {
          utilization: 100,
          resets_at: '2026-07-25T12:00:00Z',
          remaining_seconds: 3600
        }
      }))

    const automatic = requestAccountUsage(account)
    await Promise.resolve()
    const active = requestAccountUsage(account, {
      source: 'active',
      bypassCache: true,
      force: true
    })
    await active.promise

    resolveAutomatic(makeUsage({
      five_hour: {
        utilization: 10,
        resets_at: '2026-07-25T12:00:00Z',
        remaining_seconds: 3600
      }
    }))
    await automatic.promise

    expect(automatic.isCurrent()).toBe(false)
    expect(getAccountUsageHealthSnapshot(account.id)?.health).toMatchObject({
      exhausted: true,
      maxUtilization: 100
    })
  })

  it('invalidates the snapshot and fences every request created before invalidation', async () => {
    const account = makeAccount(8_001_003)
    const staleUsage = makeUsage({
      is_forbidden: true,
      error_code: 'forbidden'
    })
    const freshUsage = makeUsage({
      five_hour: {
        utilization: 20,
        resets_at: '2026-07-25T12:00:00Z',
        remaining_seconds: 3600
      }
    })
    let resolveStaleRequest!: (usage: AccountUsageInfo) => void
    getUsage
      .mockReturnValueOnce(new Promise<AccountUsageInfo>((resolve) => {
        resolveStaleRequest = resolve
      }))
      .mockResolvedValueOnce(freshUsage)
    publishAccountUsage(account.id, staleUsage, { authoritative: true })

    const staleRequest = requestAccountUsage(account)
    await Promise.resolve()
    expect(getUsage).toHaveBeenCalledTimes(1)

    expect(invalidateAccountUsageHealthSnapshot(account.id)).toBe(true)
    expect(getAccountUsageHealthSnapshot(account.id)).toBeNull()

    const freshRequest = requestAccountUsage(account)
    expect(freshRequest).not.toBe(staleRequest)
    await freshRequest.promise
    expect(getUsage).toHaveBeenCalledTimes(2)

    resolveStaleRequest(staleUsage)
    await staleRequest.promise

    expect(staleRequest.isCurrent()).toBe(false)
    expect(getAccountUsageHealthSnapshot(account.id)?.usage).toBe(freshUsage)
    expect(getAccountUsageHealthSnapshot(account.id)?.health.isForbidden).toBe(false)
  })
})
