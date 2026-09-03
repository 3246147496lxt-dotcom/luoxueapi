import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { createDemoOverview, demoOverview } from '@/data/demo'
import type { ViewerUiStatus } from '@/composables/useQuotaViewer'
import type { QuotaOverview } from '@/types'
import TrayPopover from './TrayPopover.vue'

const mountPopover = (
  overview: QuotaOverview | null = demoOverview,
  status: ViewerUiStatus = 'ready'
) =>
  mount(TrayPopover, {
    props: {
      overview,
      status,
      dataStatus: 'ready',
      refreshing: false,
      selectedQuotaId: null
    }
  })

describe('menu bar quota popover', () => {
  it('keeps the previous account and circular membership overview', () => {
    const wrapper = mountPopover({
      ...demoOverview,
      quotas: [
        {
          ...demoOverview.quotas[0],
          usedPercent: 0.02357628572
        }
      ]
    })

    expect(wrapper.text()).toContain('账户余额')
    expect(wrapper.text()).toContain('128.64 积分')
    expect(wrapper.text()).toContain('今日消费')
    expect(wrapper.text()).toContain('1.16 积分')
    expect(wrapper.text()).toContain('本月消费')
    expect(wrapper.text()).toContain('10.74 积分')
    expect(wrapper.get('.quota-ring__copy strong').text()).toBe('100%')
    expect(wrapper.get('.quota-ring__copy').text()).toBe('100%')
    expect(wrapper.get('.membership-period-label').text()).toBe('周剩余：')
    expect(wrapper.text()).toContain('本周期 Token')
    expect(wrapper.text()).toContain('本周期剩余积分')
    expect(wrapper.text()).toContain('1,906,000')
    expect(
      wrapper.get('.membership-summary > span:nth-child(3)').text()
    ).toContain('63.80 积分')
    expect(wrapper.text()).not.toContain('本周期请求')
    expect(wrapper.text()).not.toContain('99.97642371428%')
    expect(wrapper.find('.floating-quota-widget').exists()).toBe(false)
    expect(wrapper.find('img').exists()).toBe(false)

    const offset = Number(
      wrapper
        .get('.quota-ring__progress')
        .attributes('stroke-dashoffset')
    )
    expect(offset).toBeGreaterThan(0)
    expect(offset).toBeLessThan(1)
    wrapper.unmount()
  })

  it('keeps warning, exhausted, expired, and unknown visually distinct', () => {
    const warning = mountPopover(createDemoOverview('warning'))
    const exhausted = mountPopover(createDemoOverview('exhausted'))
    const expired = mountPopover(createDemoOverview('expired'))
    const unknown = mountPopover(createDemoOverview('unknown'))

    expect(warning.get('.membership-card').classes()).toContain(
      'membership-card--warning'
    )
    expect(warning.text()).toContain('即将耗尽')
    expect(exhausted.get('.membership-card').classes()).toContain(
      'membership-card--exhausted'
    )
    expect(exhausted.text()).toContain('0%')
    expect(expired.get('.membership-card').classes()).toContain(
      'membership-card--expired'
    )
    expect(expired.text()).toContain('当前不可用')
    expect(unknown.get('.membership-card').classes()).toContain(
      'membership-card--unknown'
    )
    expect(unknown.text()).toContain('待确认')
    expect(unknown.text()).not.toContain('0%')
    expect(
      unknown.get('.membership-summary > span:nth-child(3) strong').text()
    ).toBe('—')
    expect(
      unknown
        .get('.membership-summary > span:nth-child(3) strong')
        .classes()
    ).toContain('metric-unavailable')

    warning.unmount()
    exhausted.unmount()
    expired.unmount()
    unknown.unmount()
  })

  it('shows no membership without hiding wallet spending', () => {
    const wrapper = mountPopover(createDemoOverview('no-membership'))

    expect(wrapper.text()).toContain('暂无会员订阅')
    expect(wrapper.text()).toContain('今日消费')
    expect(wrapper.text()).toContain('本月消费')
    expect(wrapper.find('.membership-card').exists()).toBe(false)
    wrapper.unmount()
  })

  it('marks stale data while retaining its values and gray ring', () => {
    const wrapper = mount(TrayPopover, {
      props: {
        overview: demoOverview,
        status: 'stale',
        dataStatus: 'stale'
      }
    })

    expect(wrapper.text()).toContain('上次数据')
    expect(wrapper.text()).toContain('上次同步数据')
    expect(wrapper.text()).toContain('128.64 积分')
    expect(wrapper.get('.membership-card').classes()).toContain(
      'membership-card--stale'
    )
    wrapper.unmount()
  })

  it('emits refresh, detail selection, and open-main actions', async () => {
    const wrapper = mountPopover()

    expect(wrapper.get('.membership-card').attributes('aria-expanded')).toBe(
      'false'
    )
    await wrapper.get('[aria-label="刷新积分"]').trigger('click')
    await wrapper.get('.membership-card').trigger('click')
    await wrapper.get('.tray-open-main').trigger('click')

    expect(wrapper.emitted('refresh')).toHaveLength(1)
    expect(wrapper.emitted('selectQuota')).toEqual([
      [demoOverview.quotas[0]?.id]
    ])
    expect(wrapper.emitted('openMain')).toHaveLength(1)
    wrapper.unmount()
  })

  it('routes disconnected setup to the expanded main window', async () => {
    const wrapper = mountPopover(null, 'disconnected')

    expect(wrapper.text()).toContain('尚未连接账户')
    await wrapper.get('.tray-open-main').trigger('click')
    expect(wrapper.emitted('openMain')).toHaveLength(1)
    wrapper.unmount()
  })
})
