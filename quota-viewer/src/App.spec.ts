import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import App from './App.vue'
import MainPanel from './components/MainPanel.vue'
import { demoOverview } from './data/demo'

const legacyCurrencyMarker = String.fromCodePoint(0x2744)
const hiddenDashboardCopy = [
  '账户余额',
  '今日消费',
  '本月消费',
  legacyCurrencyMarker,
  '128.64 积分',
  '136.20 积分',
  '63.80 积分',
  'Token',
  '请求',
  '本周期已用',
  '1,906,000',
  '606'
]

const expectMembershipOnly = (text: string) => {
  for (const copy of hiddenDashboardCopy) {
    expect(text).not.toContain(copy)
  }
  expect(text).not.toMatch(/\d+(?:\.\d+)?\s*积分/u)
}

const tapQuotaWidget = async (
  wrapper: ReturnType<typeof mount>,
  _pointerId: number
) => {
  await wrapper.get('.floating-quota-widget').trigger('click')
}

describe('desktop floating quota App', () => {
  beforeEach(() => {
    delete document.documentElement.dataset.runtime
    document.documentElement.dataset.surface = 'main'
    window.history.replaceState({}, '', '/')
    vi.useFakeTimers()
    const resetAt = new Date(demoOverview.quotas[0]?.resetsAt ?? 0)
    vi.setSystemTime(new Date(resetAt.getTime() - 3 * 24 * 60 * 60 * 1000))
  })

  afterEach(() => {
    vi.useRealTimers()
    delete document.documentElement.dataset.surface
  })

  it('renders the approved 314px membership-only circular design', async () => {
    const wrapper = mount(App)
    await flushPromises()

    expect(wrapper.get('.floating-monitor-stage').classes()).toContain(
      'floating-monitor-stage--expanded'
    )
    expect(wrapper.get('.floating-quota-viewport').attributes('aria-label')).toBe(
      'Codex 会员周积分'
    )
    expect(wrapper.get('.floating-quota-widget__plan').text()).toBe(
      'CODEX · PRO'
    )
    expect(wrapper.get('.floating-quota-widget__percent').text()).toBe('32%')
    expect(wrapper.get('.floating-quota-widget__ring').attributes('aria-valuenow')).toBe(
      '32'
    )
    expect(wrapper.find('.membership-card').exists()).toBe(false)
    expect(wrapper.find('.tray-balance').exists()).toBe(false)
    expectMembershipOnly(wrapper.text())
    wrapper.unmount()
  })

  it('switches between the 80px compact value and expanded ring', async () => {
    const wrapper = mount(App)
    await flushPromises()

    await tapQuotaWidget(wrapper, 1)
    await flushPromises()

    expect(wrapper.get('.floating-quota-widget').classes()).toContain(
      'floating-quota-widget--collapsed'
    )
    expect(wrapper.get('.floating-quota-widget__compact').text()).toBe('32%')
    expect(wrapper.find('.floating-quota-widget__ring').exists()).toBe(false)

    await tapQuotaWidget(wrapper, 2)
    await flushPromises()

    expect(wrapper.get('.floating-monitor-stage').classes()).toContain(
      'floating-monitor-stage--expanded'
    )
    expect(wrapper.find('.floating-quota-widget__ring').exists()).toBe(true)
    wrapper.unmount()
  })

  it('keeps all seven fidelity states in the browser preview', async () => {
    document.documentElement.dataset.runtime = 'browser'
    const wrapper = mount(App)
    await flushPromises()

    for (const [state, expectedClass] of [
      ['available', 'available'],
      ['warning', 'warning'],
      ['exhausted', 'exhausted'],
      ['expired', 'expired'],
      ['unknown', 'unavailable'],
      ['stale', 'stale'],
      ['no-membership', 'no-membership']
    ] as const) {
      await wrapper.get(`[data-preview-state="${state}"]`).trigger('click')
      expect(wrapper.get('.floating-quota-widget').classes()).toContain(
        `floating-quota-widget--${expectedClass}`
      )
      expectMembershipOnly(wrapper.text())
    }
    wrapper.unmount()
  })

  it('uses the same compact component for disconnected account recovery', async () => {
    const wrapper = mount(MainPanel, {
      props: {
        overview: null,
        status: 'disconnected',
        collapsed: false
      }
    })

    expect(wrapper.get('.floating-quota-widget').classes()).toContain(
      'floating-quota-widget--unavailable'
    )
    expect(wrapper.text()).toContain('尚未连接落雪账户')
    expect(wrapper.text()).toContain('连接账户')
    await wrapper.get('.floating-quota-widget__refresh').trigger('click')
    expect(wrapper.emitted('connect')).toHaveLength(1)
    wrapper.unmount()
  })
})
