import { nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import AccountStatusIndicator from '../AccountStatusIndicator.vue'
import {
  invalidateAccountUsageHealthSnapshot,
  publishAccountUsage
} from '@/composables/useAccountUsageHealth'
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

const makeAccount = (overrides: Partial<Account> = {}): Account => ({
  id: 9_001_001,
  name: 'timed-account',
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
  created_at: '2026-07-26T00:00:00Z',
  updated_at: '2026-07-26T00:00:00Z',
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

describe('AccountStatusIndicator timed status', () => {
  const mountedWrappers: Array<{ unmount: () => void }> = []
  const mountIndicator = (account: Account) => {
    const wrapper = mount(AccountStatusIndicator, {
      props: { account, compact: true },
      global: { stubs: { Icon: true } }
    })
    mountedWrappers.push(wrapper)
    return wrapper
  }

  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime('2026-07-26T08:00:00Z')
  })

  afterEach(() => {
    for (const wrapper of mountedWrappers.splice(0)) {
      wrapper.unmount()
    }
    invalidateAccountUsageHealthSnapshot(9_001_001)
    invalidateAccountUsageHealthSnapshot(9_001_002)
    vi.useRealTimers()
  })

  it('recomputes an account rate limit after its reset without new props', async () => {
    const account = makeAccount({
      rate_limit_reset_at: new Date(Date.now() + 1500).toISOString()
    })
    const wrapper = mountIndicator(account)

    expect(wrapper.text()).toBe('admin.accounts.status.rateLimited')

    await vi.advanceTimersByTimeAsync(2000)
    await nextTick()

    expect(wrapper.text()).toBe('admin.accounts.workbench.activeStatus')
  })

  it('recomputes an exhausted OAuth window after its reset without a list refresh', async () => {
    const account = makeAccount({ id: 9_001_002 })
    publishAccountUsage(account.id, {
      updated_at: new Date().toISOString(),
      five_hour: {
        utilization: 100,
        resets_at: new Date(Date.now() + 1500).toISOString(),
        remaining_seconds: 2
      },
      seven_day: null,
      seven_day_sonnet: null
    }, { authoritative: true })
    const wrapper = mountIndicator(account)

    expect(wrapper.text()).toBe('admin.accounts.status.quotaExceeded')

    await vi.advanceTimersByTimeAsync(2000)
    await nextTick()

    expect(wrapper.text()).toBe('admin.accounts.workbench.activeStatus')
  })
})
