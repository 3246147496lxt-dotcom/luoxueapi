import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it } from 'vitest'
import App from './App.vue'
import MainPanel from './components/MainPanel.vue'
import { createDemoOverview, demoOverview } from './data/demo'
import type { QuotaState } from './types'

describe('quota viewer', () => {
  beforeEach(() => {
    delete document.documentElement.dataset.runtime
  })

  it('shows the account balance and opens a quota detail panel', async () => {
    const wrapper = mount(App)
    await flushPromises()

    expect(wrapper.get('#quota-viewer-title').text()).toBe('落雪额度')
    expect(wrapper.text()).toContain('账户余额')
    expect(wrapper.text()).toContain('❄128.64')
    expect(wrapper.text()).toContain('今日消费')
    expect(wrapper.text()).toContain('❄1.16')
    expect(wrapper.text()).toContain('本月消费')
    expect(wrapper.text()).toContain('❄10.74')
    expect(wrapper.text()).not.toContain('¥')
    expect(wrapper.text()).not.toContain('Codex 周额度')
    expect(wrapper.text()).toContain('剩余')
    expect(wrapper.text()).toContain('32%')
    expect(wrapper.text()).toContain('1,906,000')
    expect(wrapper.text()).toContain('606')
    expect(wrapper.text()).toContain('本周期 Token')
    expect(wrapper.text()).toContain('本周期请求')
    expect(wrapper.text()).not.toContain('近 7 日')
    expect(wrapper.get('.brand-mark img').attributes('src')).toBe('/logo.png')
    expect(wrapper.find('.monitor-panel--detail').exists()).toBe(false)

    await wrapper.get('.membership-card').trigger('click')

    expect(wrapper.find('.monitor-panel--detail').exists()).toBe(true)
    expect(wrapper.text()).toContain('本周期 Token 消耗')
    expect(wrapper.text()).toContain('7/25 14:00 至今')
    expect(wrapper.findAll('.detail-chart__column')).toHaveLength(5)
    expect(wrapper.get('.detail-chart__column--selected').text()).toContain('7/29')
    expect(wrapper.text()).not.toContain('7/30')
  })

  it('shows the empty membership state when the account has no subscription', () => {
    const wrapper = mount(MainPanel, {
      props: {
        overview: {
          ...demoOverview,
          quotas: []
        },
        selectedQuotaId: null
      }
    })

    expect(wrapper.text()).toContain('暂无会员订阅')
    expect(wrapper.find('.membership-card').exists()).toBe(false)
  })

  it.each<[QuotaState, string, string]>([
    ['available', '32%', '❄63.80'],
    ['warning', '8%', '❄16.00'],
    ['exhausted', '0%', '❄0.00'],
    ['expired', '已到期', '当前不可用']
  ])(
    'renders the %s membership fidelity state with matching data',
    (state, primaryCopy, secondaryCopy) => {
      const wrapper = mount(MainPanel, {
        props: {
          overview: createDemoOverview(state),
          selectedQuotaId: null
        }
      })

      expect(wrapper.get('.membership-card').classes()).toContain(
        `membership-card--${state}`
      )
      expect(wrapper.get('.membership-card').text()).toContain(primaryCopy)
      expect(wrapper.get('.membership-card').text()).toContain(secondaryCopy)
    }
  )

  it('exposes a browser-only development state switcher', async () => {
    document.documentElement.dataset.runtime = 'browser'
    const wrapper = mount(App)
    await flushPromises()

    const expectAccountSpendUnchanged = () => {
      const spendMetrics = wrapper.findAll('.balance-card .spend-metric')

      expect(spendMetrics).toHaveLength(2)
      expect(spendMetrics[0]?.text()).toContain('今日消费')
      expect(spendMetrics[0]?.text()).toContain('❄1.16')
      expect(spendMetrics[1]?.text()).toContain('本月消费')
      expect(spendMetrics[1]?.text()).toContain('❄10.74')
    }

    expect(wrapper.find('.dev-state-switcher').exists()).toBe(true)
    expectAccountSpendUnchanged()

    await wrapper.get('[data-preview-state="warning"]').trigger('click')
    expect(wrapper.get('.membership-card').classes()).toContain(
      'membership-card--warning'
    )
    expect(wrapper.get('.membership-card').text()).toContain('8%')
    expectAccountSpendUnchanged()

    await wrapper.get('[data-preview-state="exhausted"]').trigger('click')
    expect(wrapper.get('.membership-card').classes()).toContain(
      'membership-card--exhausted'
    )
    expect(wrapper.get('.membership-card').text()).toContain('❄0.00')
    expectAccountSpendUnchanged()

    await wrapper.get('[data-preview-state="expired"]').trigger('click')
    expect(wrapper.get('.membership-card').classes()).toContain(
      'membership-card--expired'
    )
    expect(wrapper.get('.membership-card').text()).toContain('已于 7/29 14:00 到期')
    expect(wrapper.find('.membership-summary').exists()).toBe(false)
    expectAccountSpendUnchanged()

    await wrapper.get('.membership-card').trigger('click')
    expect(wrapper.get('.detail-lifecycle-card').text()).toContain('会员已到期')
    expect(wrapper.get('.detail-lifecycle-card').text()).toContain(
      '已于 7/29 14:00 到期'
    )
    expect(wrapper.find('.detail-metrics').exists()).toBe(false)
    expect(wrapper.find('.detail-chart-card').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('重置时间待确认')
    expect(wrapper.text()).not.toContain('— - —')

    await wrapper.get('[data-preview-state="no-membership"]').trigger('click')
    expect(wrapper.text()).toContain('暂无会员订阅')
    expect(wrapper.find('.membership-card').exists()).toBe(false)
    expectAccountSpendUnchanged()
  })
})
