import { describe, expect, it } from 'vitest'
import type { Account, UsageProgress } from '@/types'
import {
  resolveAccountEffectiveStatus,
  type OAuthUsageHealthSnapshot
} from '../accountEffectiveStatus'

const NOW = '2026-07-25T12:00:00.000Z'
const FUTURE = '2026-07-25T13:00:00Z'
const LATER = '2026-07-26T12:00:00Z'
const PAST = '2026-07-25T11:00:00Z'

const makeAccount = (overrides: Partial<Account> = {}): Account => ({
  id: 41,
  name: 'status-test',
  platform: 'openai',
  type: 'oauth',
  credentials: {},
  extra: {},
  proxy_id: null,
  concurrency: 4,
  priority: 10,
  status: 'active',
  error_message: null,
  last_used_at: null,
  expires_at: null,
  auto_pause_on_expired: true,
  created_at: '2026-07-01T00:00:00Z',
  updated_at: '2026-07-25T11:55:00Z',
  schedulable: true,
  rate_limited_at: null,
  rate_limit_reset_at: null,
  overload_until: null,
  temp_unschedulable_until: null,
  temp_unschedulable_reason: null,
  session_window_start: null,
  session_window_end: null,
  session_window_status: null,
  ...overrides
})

const usageWindow = (
  utilization: number,
  resetsAt: string | null = FUTURE
): UsageProgress => ({
  utilization,
  resets_at: resetsAt,
  remaining_seconds: resetsAt ? 3600 : 0
})

const resolve = (
  account: Account,
  usage: OAuthUsageHealthSnapshot | null = null
) => resolveAccountEffectiveStatus(account, usage, { now: NOW })

describe('resolveAccountEffectiveStatus', () => {
  it('returns compact-ready active metadata', () => {
    expect(resolve(makeAccount())).toEqual({
      code: 'active',
      tone: 'success',
      labelKey: 'admin.accounts.workbench.activeStatus',
      titleKey: 'admin.accounts.workbench.activeStatus',
      titleParams: {},
      detail: null,
      resetAt: null,
      resetKind: null,
      affectedScopes: []
    })
  })

  it('classifies permission, authentication, and generic account errors', () => {
    const permission = resolve(
      makeAccount(),
      {
        is_forbidden: true,
        needs_reauth: true,
        forbidden_type: 'validation',
        forbidden_reason: 'Validation required (403)'
      }
    )
    expect(permission).toMatchObject({
      code: 'permission_error',
      tone: 'danger',
      labelKey: 'admin.accounts.forbiddenValidation',
      detail: 'Validation required (403)'
    })

    const authentication = resolve(
      makeAccount(),
      {
        needs_reauth: true,
        error_code: 'unauthenticated',
        error: 'token expired'
      }
    )
    expect(authentication).toMatchObject({
      code: 'authentication_error',
      tone: 'danger',
      labelKey: 'admin.accounts.needsReauth',
      detail: 'token expired'
    })

    const generic = resolve(makeAccount({
      status: 'error',
      error_message: 'unexpected upstream failure'
    }))
    expect(generic).toMatchObject({
      code: 'account_error',
      tone: 'danger',
      labelKey: 'admin.accounts.status.error',
      detail: 'unexpected upstream failure'
    })
  })

  it('classifies status=error messages as permission or authentication failures', () => {
    expect(resolve(makeAccount({
      status: 'error',
      error_message: 'Access forbidden (403): workspace denied'
    })).code).toBe('permission_error')

    expect(resolve(makeAccount({
      status: 'error',
      error_message: 'Authentication failed (401): token revoked'
    })).code).toBe('authentication_error')
  })

  it('does not report active when the row carries an inconsistent error message', () => {
    expect(resolve(makeAccount({
      status: 'active',
      error_message: 'Access forbidden (403): workspace denied'
    }))).toMatchObject({
      code: 'permission_error',
      tone: 'danger'
    })

    expect(resolve(makeAccount({
      status: 'active',
      error_message: 'unexpected upstream failure'
    }))).toMatchObject({
      code: 'account_error',
      tone: 'danger'
    })
  })

  it('summarizes and redacts legacy raw authentication responses', () => {
    const legacy = resolve(makeAccount({
      status: 'active',
      credentials: {
        access_token: 'secret-token-value'
      },
      error_message: `Authentication failed (401): ${JSON.stringify({
        error: {
          message: 'Provided authentication token is expired. access_token=secret-token-value',
          code: 'token_expired'
        },
        debug: 'must-not-be-rendered'
      }, null, 2)}`
    }))

    expect(legacy).toMatchObject({
      code: 'authentication_error',
      tone: 'danger',
      detail: 'Authentication failed (401): Provided authentication token is expired. access_token=[REDACTED] (token_expired)'
    })
    expect(legacy.detail).not.toContain('secret-token-value')
    expect(legacy.detail).not.toContain('must-not-be-rendered')
    expect(legacy.detail).not.toContain('\n')
  })

  it('enforces the documented priority from errors through active', () => {
    const localLimit = {
      model_rate_limits: {
        'claude-sonnet-4-5': {
          rate_limited_at: PAST,
          rate_limit_reset_at: FUTURE
        }
      }
    }
    const allLowerStates = makeAccount({
      status: 'inactive',
      schedulable: false,
      quota_limit: 10,
      quota_used: 10,
      overload_until: FUTURE,
      rate_limit_reset_at: FUTURE,
      temp_unschedulable_until: FUTURE,
      extra: localLimit
    })

    expect(resolve(allLowerStates, { is_forbidden: true }).code).toBe('permission_error')
    expect(resolve(allLowerStates, { needs_reauth: true }).code).toBe('authentication_error')
    expect(resolve(allLowerStates).code).toBe('quota_exhausted')

    expect(resolve(makeAccount({
      status: 'inactive',
      schedulable: false,
      overload_until: FUTURE,
      rate_limit_reset_at: FUTURE,
      temp_unschedulable_until: FUTURE,
      extra: localLimit
    })).code).toBe('overloaded')

    expect(resolve(makeAccount({
      status: 'inactive',
      schedulable: false,
      rate_limit_reset_at: FUTURE,
      temp_unschedulable_until: FUTURE,
      extra: localLimit
    })).code).toBe('rate_limited')

    expect(resolve(makeAccount({
      status: 'inactive',
      schedulable: false,
      temp_unschedulable_until: FUTURE,
      extra: localLimit
    })).code).toBe('temp_unschedulable')

    expect(resolve(makeAccount({
      status: 'inactive',
      schedulable: false,
      extra: localLimit
    })).code).toBe('inactive')

    expect(resolve(makeAccount({
      schedulable: false,
      extra: localLimit
    })).code).toBe('paused')

    expect(resolve(makeAccount({
      concurrency: 4,
      current_concurrency: 4,
      extra: localLimit
    })).code).toBe('model_rate_limited')
    expect(resolve(makeAccount({
      concurrency: 4,
      current_concurrency: 4
    })).code).toBe('at_capacity')
    expect(resolve(makeAccount()).code).toBe('active')
  })

  it('treats exactly 100% OAuth usage as exhausted but not values below 100%', () => {
    const exhausted = resolve(makeAccount(), {
      five_hour: usageWindow(100),
      seven_day: usageWindow(100, LATER)
    })
    expect(exhausted).toMatchObject({
      code: 'quota_exhausted',
      tone: 'danger',
      labelKey: 'admin.accounts.status.quotaExceeded',
      resetAt: LATER,
      resetKind: 'quota',
      affectedScopes: ['five_hour', 'seven_day']
    })

    expect(resolve(makeAccount(), {
      five_hour: usageWindow(99.999)
    }).code).toBe('active')
  })

  it('uses canonical passive Codex usage and ignores an exhausted window after reset', () => {
    const exhaustedAccount = makeAccount({
      extra: {
        codex_usage_updated_at: '2026-07-25T11:55:00Z',
        codex_5h_used_percent: 100,
        codex_5h_reset_at: FUTURE
      }
    })
    expect(resolve(exhaustedAccount)).toMatchObject({
      code: 'quota_exhausted',
      affectedScopes: ['five_hour'],
      resetAt: FUTURE
    })

    expect(resolve(makeAccount({
      extra: {
        codex_5h_used_percent: 100,
        codex_5h_reset_at: PAST
      }
    })).code).toBe('active')
  })

  it('returns actionable detail and reset metadata for runtime blocks', () => {
    const overloaded = resolve(makeAccount({ overload_until: FUTURE }))
    expect(overloaded).toMatchObject({
      code: 'overloaded',
      tone: 'danger',
      titleKey: 'admin.accounts.status.overloadedUntil',
      titleParams: { time: FUTURE },
      resetAt: FUTURE,
      resetKind: 'overload'
    })

    const rateLimited = resolve(makeAccount({ rate_limit_reset_at: FUTURE }))
    expect(rateLimited).toMatchObject({
      code: 'rate_limited',
      tone: 'warning',
      titleKey: 'admin.accounts.status.rateLimitedUntil',
      titleParams: { time: FUTURE },
      resetAt: FUTURE,
      resetKind: 'rate_limit'
    })

    const temporary = resolve(makeAccount({
      temp_unschedulable_until: FUTURE,
      temp_unschedulable_reason: 'OpenAI 403 temporary cooldown'
    }))
    expect(temporary).toMatchObject({
      code: 'temp_unschedulable',
      tone: 'warning',
      detail: 'OpenAI 403 temporary cooldown',
      resetAt: FUTURE,
      resetKind: 'temp_unschedulable'
    })
  })

  it('derives a global rate-limit reset from a usage retry duration', () => {
    expect(resolve(makeAccount(), {
      error_code: 'rate_limited',
      grok_retry_after_seconds: 90
    })).toMatchObject({
      code: 'rate_limited',
      resetAt: '2026-07-25T12:01:30.000Z',
      titleParams: { time: '2026-07-25T12:01:30.000Z' }
    })
  })

  it('distinguishes model limits, credit overages, and exhausted credits', () => {
    const modelLimit = {
      'claude-sonnet-4-5': {
        rate_limited_at: PAST,
        rate_limit_reset_at: FUTURE
      }
    }

    expect(resolve(makeAccount({
      extra: { model_rate_limits: modelLimit }
    }))).toMatchObject({
      code: 'model_rate_limited',
      tone: 'warning',
      labelKey: 'admin.accounts.status.limited',
      titleKey: 'admin.accounts.status.modelRateLimitedUntil',
      affectedScopes: ['claude-sonnet-4-5'],
      resetAt: FUTURE
    })

    expect(resolve(makeAccount({
      extra: {
        allow_overages: true,
        model_rate_limits: modelLimit
      }
    }))).toMatchObject({
      code: 'credits_active',
      tone: 'warning',
      titleKey: 'admin.accounts.status.modelCreditOveragesUntil'
    })

    expect(resolve(makeAccount({
      extra: {
        allow_overages: true,
        model_rate_limits: {
          ...modelLimit,
          AICredits: {
            rate_limited_at: PAST,
            rate_limit_reset_at: LATER
          }
        }
      }
    }))).toMatchObject({
      code: 'credits_exhausted',
      tone: 'danger',
      labelKey: 'admin.accounts.status.creditsExhausted',
      titleKey: 'admin.accounts.status.creditsExhaustedUntil',
      affectedScopes: ['AICredits'],
      resetAt: LATER
    })
  })

  it('treats model-specific OAuth quota exhaustion as a local restriction', () => {
    expect(resolve(makeAccount(), {
      seven_day_sonnet: usageWindow(100, FUTURE),
      antigravity_quota: {
        'gemini-3-pro': {
          utilization: 100,
          reset_time: LATER
        }
      }
    })).toMatchObject({
      code: 'model_rate_limited',
      tone: 'warning',
      affectedScopes: ['sonnet', 'gemini-3-pro'],
      resetAt: LATER
    })
  })

  it('ignores expired runtime and local reset timestamps', () => {
    expect(resolve(makeAccount({
      overload_until: PAST,
      rate_limit_reset_at: PAST,
      temp_unschedulable_until: PAST,
      extra: {
        model_rate_limits: {
          'claude-sonnet-4-5': {
            rate_limited_at: PAST,
            rate_limit_reset_at: PAST
          }
        }
      }
    }))).toMatchObject({
      code: 'active',
      tone: 'success'
    })
  })

  it('keeps inactive ahead of paused and maps paused to a muted compact tone', () => {
    expect(resolve(makeAccount({
      status: 'inactive',
      schedulable: false
    }))).toMatchObject({
      code: 'inactive',
      tone: 'muted',
      labelKey: 'admin.accounts.status.inactive'
    })

    expect(resolve(makeAccount({
      schedulable: false
    }))).toMatchObject({
      code: 'paused',
      tone: 'muted',
      labelKey: 'admin.accounts.status.paused'
    })
  })

  it('marks a saturated concurrency pool as warning-level at capacity', () => {
    expect(resolve(makeAccount({
      concurrency: 4,
      current_concurrency: 4
    }))).toMatchObject({
      code: 'at_capacity',
      tone: 'warning',
      labelKey: 'admin.accounts.workbench.fullStatus'
    })

    expect(resolve(makeAccount({
      concurrency: 4,
      current_concurrency: 3
    })).code).toBe('active')

    expect(resolve(makeAccount({
      concurrency: 0,
      current_concurrency: 8
    })).code).toBe('active')
  })
})
