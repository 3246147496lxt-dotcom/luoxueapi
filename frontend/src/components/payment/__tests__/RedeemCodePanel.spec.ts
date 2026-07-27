import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import RedeemCodePanel from '../RedeemCodePanel.vue'

const redeem = vi.hoisted(() => vi.fn())
const getHistory = vi.hoisted(() => vi.fn())
const refreshUser = vi.hoisted(() => vi.fn())
const fetchActiveSubscriptions = vi.hoisted(() => vi.fn())
const showSuccess = vi.hoisted(() => vi.fn())
const showError = vi.hoisted(() => vi.fn())
const showWarning = vi.hoisted(() => vi.fn())
const user = vi.hoisted(() => ({ balance: 5, concurrency: 2 }))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => params ? `${key}:${JSON.stringify(params)}` : key,
    }),
  }
})

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({ user, refreshUser }),
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    contactInfo: 'support@example.com',
    cachedPublicSettings: null,
    showSuccess,
    showError,
    showWarning,
  }),
}))

vi.mock('@/stores/subscriptions', () => ({
  useSubscriptionStore: () => ({ fetchActiveSubscriptions }),
}))

vi.mock('@/api/redeem', () => ({
  redeemAPI: { redeem, getHistory },
}))

function redeemResult(overrides: Record<string, unknown> = {}) {
  return {
    id: 1,
    code: 'CODE-001',
    type: 'balance',
    value: 10,
    status: 'used',
    used_at: '2026-07-19T00:00:00Z',
    created_at: '2026-07-18T00:00:00Z',
    ...overrides,
  }
}

describe('RedeemCodePanel', () => {
  beforeEach(() => {
    user.balance = 5
    user.concurrency = 2
    redeem.mockReset()
    getHistory.mockReset().mockResolvedValue([])
    refreshUser.mockReset().mockImplementation(async () => {
      user.balance = 15
    })
    fetchActiveSubscriptions.mockReset().mockResolvedValue(undefined)
    showSuccess.mockReset()
    showError.mockReset()
    showWarning.mockReset()
  })

  it('loads history independently and exposes an accessible redemption form', async () => {
    const wrapper = mount(RedeemCodePanel)
    await flushPromises()

    expect(getHistory).toHaveBeenCalledTimes(1)
    expect(wrapper.get('section').attributes('aria-labelledby')).toBe('redeem-panel-title')
    expect(wrapper.get('section').classes()).toContain('rounded-2xl')
    expect(wrapper.get('input').attributes('spellcheck')).toBe('false')
    expect(wrapper.get('button[type="submit"]').classes()).toContain('btn-primary')
    expect(wrapper.get('button[type="submit"]').attributes()).toHaveProperty('disabled')
  })

  it('uses the compact embedded layout without description or activity history', async () => {
    const wrapper = mount(RedeemCodePanel, { props: { embedded: true } })
    await flushPromises()

    expect(getHistory).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('redeem.quickRedeemTitle')
    expect(wrapper.text()).not.toContain('redeem.quickRedeemDescription')
    expect(wrapper.text()).not.toContain('redeem.recentActivity')
    expect(wrapper.find('details').exists()).toBe(false)
    expect(wrapper.get('label').classes()).toContain('sr-only')
    expect(wrapper.get('section').classes()).toContain('redeem-panel--embedded')
    expect(wrapper.get('section').classes()).not.toContain('rounded-2xl')
    expect(wrapper.get('section').classes()).not.toContain('border')

    const formRow = wrapper.get('[data-testid="redeem-form-row"]')
    expect(formRow.classes()).toContain('flex-col')
    expect(formRow.classes()).toContain('sm:flex-row')

    const submitButton = wrapper.get('button[type="submit"]')
    expect(submitButton.classes()).toContain('btn-secondary')
    expect(submitButton.classes()).toContain('w-full')
    expect(submitButton.classes()).toContain('sm:w-auto')
    expect(submitButton.classes()).not.toContain('btn-primary')
  })

  it('keeps redemption behavior and live feedback in the embedded layout', async () => {
    redeem.mockResolvedValue(redeemResult())
    const wrapper = mount(RedeemCodePanel, { props: { embedded: true } })
    await flushPromises()

    await wrapper.get('input').setValue('EMBEDDED-001')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(redeem).toHaveBeenCalledWith('EMBEDDED-001')
    expect(refreshUser).toHaveBeenCalledTimes(1)
    expect(getHistory).not.toHaveBeenCalled()
    expect(wrapper.get('[role="status"]').classes()).toContain('redeem-panel__feedback--success')
    expect(wrapper.emitted('redeemed')).toHaveLength(1)
  })

  it('trims the code, refreshes the account, and renders a live success result', async () => {
    redeem.mockResolvedValue(redeemResult())
    const wrapper = mount(RedeemCodePanel)
    await flushPromises()

    await wrapper.get('input').setValue('  CODE-001  ')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(redeem).toHaveBeenCalledWith('CODE-001')
    expect(refreshUser).toHaveBeenCalledTimes(1)
    expect(wrapper.get('[role="status"]').text()).toContain('redeem.balanceAddedAmount')
    expect(wrapper.get('[role="status"]').text()).toContain('redeem.currentBalance')
    expect(wrapper.get('[role="status"]').findAll('[data-testid="credit-amount"]')).toHaveLength(2)
    expect(wrapper.get('[role="status"]').text()).not.toContain('$')
    expect(wrapper.get('input').element.value).toBe('')
    expect(showSuccess).toHaveBeenCalledWith('redeem.codeRedeemSuccess')
  })

  it('refreshes active subscriptions after redeeming a subscription code', async () => {
    redeem.mockResolvedValue(redeemResult({
      type: 'subscription',
      value: 30,
      validity_days: 30,
      group: { id: 3, name: 'Pro' },
    }))
    const wrapper = mount(RedeemCodePanel)
    await flushPromises()

    await wrapper.get('input').setValue('SUB-001')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(fetchActiveSubscriptions).toHaveBeenCalledWith(true)
    expect(wrapper.get('[role="status"]').text()).toContain('redeem.subscriptionRedeemSummary')
  })

  it('keeps a successful redemption when refreshing the account fails', async () => {
    redeem.mockResolvedValue(redeemResult())
    refreshUser.mockRejectedValue(new Error('temporary refresh failure'))
    const wrapper = mount(RedeemCodePanel)
    await flushPromises()

    await wrapper.get('input').setValue('CODE-001')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(wrapper.get('[role="status"]').text()).toContain('redeem.balanceAddedAmount')
    expect(wrapper.get('[role="status"]').findAll('[data-testid="credit-amount"]')).toHaveLength(2)
    expect(showSuccess).toHaveBeenCalledWith('redeem.codeRedeemSuccess')
    expect(showWarning).toHaveBeenCalledWith('redeem.accountRefreshFailed')
    expect(showError).not.toHaveBeenCalled()
  })

  it('keeps the entered code and shows the normalized backend message on failure', async () => {
    redeem.mockRejectedValue({ message: '兑换码已被使用' })
    const wrapper = mount(RedeemCodePanel)
    await flushPromises()

    await wrapper.get('input').setValue('USED-001')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(wrapper.get('[role="alert"]').text()).toContain('兑换码已被使用')
    expect(wrapper.get('input').element.value).toBe('USED-001')
    expect(showError).toHaveBeenCalledWith('redeem.redeemFailed')
  })
})
