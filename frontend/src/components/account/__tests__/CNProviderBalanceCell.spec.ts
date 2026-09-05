import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import CNProviderBalanceCell from '../CNProviderBalanceCell.vue'
import type { Account } from '@/types'

const { queryBalance } = vi.hoisted(() => ({
  queryBalance: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    cnProviders: { queryBalance }
  }
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

const account = {
  id: 8,
  platform: 'deepseek',
  type: 'apikey',
  credentials: { account_mode: 'payg' },
  extra: {
    deepseek_balance: 12.5,
    deepseek_balance_currency: 'CNY'
  }
} as Account

describe('CNProviderBalanceCell', () => {
  beforeEach(() => {
    queryBalance.mockReset()
  })

  it('renders the snapshot separately from an explicit Query action', async () => {
    const wrapper = mount(CNProviderBalanceCell, { props: { account } })
    await flushPromises()

    expect(queryBalance).not.toHaveBeenCalled()
    expect(wrapper.get('[data-test="cn-provider-balance-value"]').text())
      .toContain('CNY 12.50')
    const probe = wrapper.get('[data-test="cn-provider-balance-probe"]')
    expect(probe.text()).toBe('admin.accounts.cnProviders.probe')

    queryBalance.mockResolvedValue({
      provider: 'deepseek',
      success: true,
      balance: 3,
      currency: 'USD',
      balances: [{ currency: 'USD', balance: 3 }],
      available: true,
      fetched_at: Date.now(),
      persisted: true
    })
    await probe.trigger('click')
    await flushPromises()

    expect(queryBalance).toHaveBeenCalledWith(account.id)
    expect(wrapper.get('[data-test="cn-provider-balance-value"]').text())
      .toContain('USD 3.00')
  })

  it('keeps the persisted balance visible when a probe fails', async () => {
    queryBalance.mockResolvedValue({ success: false, error: 'HTTP 401' })
    const wrapper = mount(CNProviderBalanceCell, { props: { account } })

    await wrapper.get('[data-test="cn-provider-balance-probe"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-test="cn-provider-balance-value"]').text())
      .toContain('CNY 12.50')
    expect(wrapper.text()).toContain('HTTP 401')
  })
})
