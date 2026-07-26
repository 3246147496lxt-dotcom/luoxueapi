import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import AccountStatusIndicator from '../AccountStatusIndicator.vue'
import { publishAccountUsage } from '@/composables/useAccountUsageHealth'
import type { Account } from '@/types'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

vi.mock('@/utils/format', async () => {
  const actual = await vi.importActual<typeof import('@/utils/format')>('@/utils/format')
  return {
    ...actual,
    formatCountdown: () => '1h'
  }
})

function makeAccount(overrides: Partial<Account>): Account {
  return {
    id: 1,
    name: 'account',
    platform: 'antigravity',
    type: 'oauth',
    proxy_id: null,
    concurrency: 1,
    priority: 1,
    status: 'active',
    error_message: null,
    last_used_at: null,
    expires_at: null,
    auto_pause_on_expired: true,
    created_at: '2026-03-15T00:00:00Z',
    updated_at: '2026-03-15T00:00:00Z',
    schedulable: true,
    rate_limited_at: null,
    rate_limit_reset_at: null,
    overload_until: null,
    temp_unschedulable_until: null,
    temp_unschedulable_reason: null,
    session_window_start: null,
    session_window_end: null,
    session_window_status: null,
    ...overrides,
  }
}

describe('AccountStatusIndicator', () => {
  it('Grok 账号额度限流时显示自动恢复时间而非临时不可调度', () => {
    const wrapper = mount(AccountStatusIndicator, {
      props: {
        account: makeAccount({
          id: 5,
          name: 'grok-free-1',
          platform: 'grok',
          rate_limited_at: '2026-07-11T12:00:00Z',
          rate_limit_reset_at: '2099-07-11T13:00:00Z',
          temp_unschedulable_until: '2099-07-11T12:30:00Z',
          temp_unschedulable_reason: 'legacy grok rate limited'
        })
      },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    expect(wrapper.find('.badge-warning').text()).toBe('admin.accounts.status.rateLimited')
    expect(wrapper.text()).toContain('admin.accounts.status.rateLimitedAutoResume')
    expect(wrapper.text()).not.toContain('admin.accounts.status.tempUnschedulable')
  })

  it('模型限流 + overages 启用 + 无 AICredits key → 显示 ⚡ (credits_active)', () => {
    const wrapper = mount(AccountStatusIndicator, {
      props: {
        account: makeAccount({
          id: 1,
          name: 'ag-1',
          extra: {
            allow_overages: true,
            model_rate_limits: {
              'claude-sonnet-4-5': {
                rate_limited_at: '2026-03-15T00:00:00Z',
                rate_limit_reset_at: '2099-03-15T00:00:00Z'
              }
            }
          }
        })
      },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    expect(wrapper.text()).toContain('⚡')
    expect(wrapper.text()).toContain('CSon45')
  })

  it('模型限流 + overages 未启用 → 普通限流样式（无 ⚡）', () => {
    const wrapper = mount(AccountStatusIndicator, {
      props: {
        account: makeAccount({
          id: 2,
          name: 'ag-2',
          extra: {
            model_rate_limits: {
              'claude-sonnet-4-5': {
                rate_limited_at: '2026-03-15T00:00:00Z',
                rate_limit_reset_at: '2099-03-15T00:00:00Z'
              }
            }
          }
        })
      },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    expect(wrapper.text()).toContain('CSon45')
    expect(wrapper.text()).not.toContain('⚡')
  })

  it('AICredits key 生效 → 显示积分已用尽 (credits_exhausted)', () => {
    const wrapper = mount(AccountStatusIndicator, {
      props: {
        account: makeAccount({
          id: 3,
          name: 'ag-3',
          extra: {
            allow_overages: true,
            model_rate_limits: {
              'AICredits': {
                rate_limited_at: '2026-03-15T00:00:00Z',
                rate_limit_reset_at: '2099-03-15T00:00:00Z'
              }
            }
          }
        })
      },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    expect(wrapper.text()).toContain('admin.accounts.status.creditsExhausted')
  })

  it('模型限流 + overages 启用 + AICredits key 生效 → 普通限流样式（积分耗尽，无 ⚡）', () => {
    const wrapper = mount(AccountStatusIndicator, {
      props: {
        account: makeAccount({
          id: 4,
          name: 'ag-4',
          extra: {
            allow_overages: true,
            model_rate_limits: {
              'claude-sonnet-4-5': {
                rate_limited_at: '2026-03-15T00:00:00Z',
                rate_limit_reset_at: '2099-03-15T00:00:00Z'
              },
              'AICredits': {
                rate_limited_at: '2026-03-15T00:00:00Z',
                rate_limit_reset_at: '2099-03-15T00:00:00Z'
              }
            }
          }
        })
      },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    // 模型限流 + 积分耗尽 → 不应显示 ⚡
    expect(wrapper.text()).toContain('CSon45')
    expect(wrapper.text()).not.toContain('⚡')
    // AICredits 积分耗尽状态应显示
    expect(wrapper.text()).toContain('admin.accounts.status.creditsExhausted')
  })

  it('compact 模式把正常账号收敛为绿色状态点和活跃短标签', () => {
    const wrapper = mount(AccountStatusIndicator, {
      props: {
        account: makeAccount({
          platform: 'openai',
          status: 'active',
          schedulable: true,
          concurrency: 4,
          current_concurrency: 0
        }),
        compact: true
      }
    })

    expect(wrapper.text()).toBe('admin.accounts.workbench.activeStatus')
    expect(wrapper.get('.account-status-summary__dot').classes()).toContain('bg-emerald-500')
    expect(wrapper.find('.badge').exists()).toBe(false)
  })

  it('compact 模式将并发占满显示为琥珀色满载状态', () => {
    const wrapper = mount(AccountStatusIndicator, {
      props: {
        account: makeAccount({
          platform: 'anthropic',
          status: 'active',
          schedulable: true,
          concurrency: 2,
          current_concurrency: 2
        }),
        compact: true
      }
    })

    expect(wrapper.text()).toBe('admin.accounts.workbench.fullStatus')
    expect(wrapper.get('.account-status-summary__dot').classes()).toContain('bg-amber-500')
  })

  it('compact 模式将共享额度快照的 100% 显示为红色额度耗尽', () => {
    const account = makeAccount({
      id: 101,
      platform: 'openai',
      status: 'active',
      schedulable: true
    })
    publishAccountUsage(account.id, {
      five_hour: {
        utilization: 100,
        resets_at: '2099-07-25T13:00:00Z',
        remaining_seconds: 3600
      }
    } as any, { authoritative: true })

    const wrapper = mount(AccountStatusIndicator, {
      props: { account, compact: true }
    })

    expect(wrapper.text()).toBe('admin.accounts.status.quotaExceeded')
    expect(wrapper.get('.account-status-summary__dot').classes()).toContain('bg-red-500')
    expect(wrapper.get('.account-status-summary').attributes('title')).toBe(
      'admin.accounts.status.quotaExceededUntil'
    )
  })

  it('compact 模式优先显示用量接口返回的权限与重新授权错误', () => {
    const forbiddenAccount = makeAccount({ id: 102, platform: 'antigravity' })
    publishAccountUsage(forbiddenAccount.id, {
      is_forbidden: true,
      forbidden_type: 'validation',
      forbidden_reason: 'verification required'
    } as any, { authoritative: true })

    const forbidden = mount(AccountStatusIndicator, {
      props: { account: forbiddenAccount, compact: true }
    })
    expect(forbidden.text()).toBe('admin.accounts.forbiddenValidation')
    expect(forbidden.get('.account-status-summary__dot').classes()).toContain('bg-red-500')

    const reauthAccount = makeAccount({ id: 103, platform: 'openai' })
    publishAccountUsage(reauthAccount.id, {
      needs_reauth: true,
      error_code: 'unauthenticated',
      error: 'token expired'
    } as any, { authoritative: true })

    const reauth = mount(AccountStatusIndicator, {
      props: { account: reauthAccount, compact: true }
    })
    expect(reauth.text()).toBe('admin.accounts.needsReauth')
    expect(reauth.get('.account-status-summary__dot').classes()).toContain('bg-red-500')
  })

  it('compact 模式让限流与模型局部限制显示黄色短状态', () => {
    const rateLimited = mount(AccountStatusIndicator, {
      props: {
        account: makeAccount({
          id: 104,
          rate_limit_reset_at: '2099-07-25T13:00:00Z'
        }),
        compact: true
      }
    })
    expect(rateLimited.text()).toBe('admin.accounts.status.rateLimited')
    expect(rateLimited.get('.account-status-summary__dot').classes()).toContain('bg-amber-500')

    const modelLimited = mount(AccountStatusIndicator, {
      props: {
        account: makeAccount({
          id: 105,
          extra: {
            model_rate_limits: {
              'claude-sonnet-4-5': {
                rate_limited_at: '2026-07-25T12:00:00Z',
                rate_limit_reset_at: '2099-07-25T13:00:00Z'
              }
            }
          }
        }),
        compact: true
      }
    })
    expect(modelLimited.text()).toBe('admin.accounts.status.limited')
    expect(modelLimited.get('.account-status-summary__dot').classes()).toContain('bg-amber-500')
  })

  it('compact 模式不会把带错误信息但状态字段仍为 active 的账号显示为活跃', () => {
    const wrapper = mount(AccountStatusIndicator, {
      props: {
        account: makeAccount({
          id: 106,
          status: 'active',
          error_message: 'Access forbidden (403): permission denied'
        }),
        compact: true
      }
    })

    expect(wrapper.text()).toBe('admin.accounts.forbidden')
    expect(wrapper.get('.account-status-summary__dot').classes()).toContain('bg-red-500')
  })
})
