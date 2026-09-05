import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick, reactive } from 'vue'
import CNProviderQuotaCell from '../CNProviderQuotaCell.vue'
import type { Account } from '@/types'

const { queryQuota } = vi.hoisted(() => ({
  queryQuota: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    cnProviders: { queryQuota }
  }
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

const account = {
  id: 7,
  platform: 'zhipu',
  type: 'apikey',
  credentials: { account_mode: 'coding' },
  extra: {
    zhipu_5h_used_percent: 0,
    zhipu_weekly_used_percent: 27,
    zhipu_5h_reset_at: '2099-08-18T12:30:00+08:00',
    zhipu_weekly_reset_at: '2099-08-22T00:00:00+08:00',
    zhipu_usage_updated_at: '2099-08-18T00:00:00+08:00'
  }
} as Account

describe('CNProviderQuotaCell', () => {
  beforeEach(() => {
    queryQuota.mockReset()
  })

  it('uses a dedicated Query action and keeps tier hooks stable', async () => {
    const wrapper = mount(CNProviderQuotaCell, { props: { account } })
    await flushPromises()

    const root = wrapper.get('[data-test="cn-provider-quota"]')
    expect(root.classes()).toContain('min-w-[220px]')
    expect(queryQuota).not.toHaveBeenCalled()
    expect(root.get('[data-test="cn-provider-quota-probe"]').text())
      .toBe('admin.accounts.cnProviders.probe')
    expect(root.findAll('[data-test="cn-provider-quota-tier"]')).toHaveLength(2)
    expect(root.findAll('[data-test="cn-provider-quota-label"]')).toHaveLength(2)

    queryQuota.mockResolvedValue({
      provider: 'zhipu',
      success: true,
      credential_valid: true,
      fetched_at: Date.now(),
      persisted: true,
      tiers: [
        { window: '5h', used_percent: 12 },
        { window: 'weekly', used_percent: 34 }
      ]
    })
    await root.get('[data-test="cn-provider-quota-probe"]').trigger('click')
    await flushPromises()

    expect(queryQuota).toHaveBeenCalledWith(account.id)
    expect(root.text()).toContain('12%')
    expect(root.text()).toContain('34%')
  })

  it('compact mode still exposes Query while leaving detailed tiers to the parent bar', async () => {
    const wrapper = mount(CNProviderQuotaCell, {
      props: { account, compact: true, autoProbe: false }
    })
    await flushPromises()

    expect(wrapper.get('[data-test="cn-provider-quota-probe"]')).toBeTruthy()
    expect(wrapper.findAll('[data-test="cn-provider-quota-tier"]')).toHaveLength(0)
    expect(wrapper.text()).toContain('admin.accounts.cnProviders.probe')
    expect(queryQuota).not.toHaveBeenCalled()
  })

  it('tracks account.extra snapshots when a list row is refreshed in place', async () => {
    const mutableAccount = reactive({
      ...account,
      extra: { ...account.extra }
    }) as unknown as Account
    const wrapper = mount(CNProviderQuotaCell, { props: { account: mutableAccount } })
    await flushPromises()

    mutableAccount.extra!.zhipu_weekly_used_percent = 81
    await nextTick()
    expect(wrapper.text()).toContain('81%')

    delete mutableAccount.extra!.zhipu_5h_used_percent
    delete mutableAccount.extra!.zhipu_weekly_used_percent
    await nextTick()
    expect(wrapper.findAll('[data-test="cn-provider-quota-tier"]')).toHaveLength(0)
  })
})
