import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createDemoOverview, demoOverview } from '@/data/demo'
import type { QuotaOverview, QuotaState } from '@/types'
import FloatingQuotaWidget from './FloatingQuotaWidget.vue'

const desktopMocks = vi.hoisted(() => ({
  startQuotaViewerDrag: vi.fn()
}))

vi.mock('@/lib/desktop', () => desktopMocks)

const hiddenDashboardCopy = [
  '账户余额',
  '今日消费',
  '本月消费',
  '❄',
  'Token',
  '请求',
  '本周期已用',
  '1,906,000',
  '606'
]

const resetAt = new Date(demoOverview.quotas[0]?.resetsAt ?? 0)
const expiresAt = new Date(demoOverview.quotas[0]?.expiresAt ?? 0)
const beforeReset = new Date(resetAt.getTime() - 3 * 24 * 60 * 60 * 1000)
const formatLocalDate = (value: Date) =>
  `${value.getMonth() + 1}/${value.getDate()}`

const mountWidget = (overview: QuotaOverview | null = demoOverview, extra = {}) =>
  mount(FloatingQuotaWidget, {
    props: {
      overview,
      ...extra
    }
  })

const expectMembershipOnly = (text: string) => {
  for (const copy of hiddenDashboardCopy) {
    expect(text).not.toContain(copy)
  }
}

describe('FloatingQuotaWidget', () => {
  beforeEach(() => {
    desktopMocks.startQuotaViewerDrag.mockClear()
    vi.useFakeTimers()
    vi.setSystemTime(beforeReset)
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('renders the focused 314px weekly circular quota surface', () => {
    const wrapper = mountWidget()

    expect(wrapper.get('.floating-quota-widget').classes()).toContain(
      'floating-quota-widget--available'
    )
    expect(wrapper.get('.floating-quota-widget').attributes('role')).toBe('group')
    expect(wrapper.get('.floating-quota-widget__plan').text()).toBe(
      'CODEX · PRO'
    )
    expect(wrapper.text()).toContain('周剩余')
    expect(wrapper.get('.floating-quota-widget__percent').text()).toBe('32%')
    expect(wrapper.get('.floating-quota-widget__ring').attributes('role')).toBe(
      'progressbar'
    )
    expect(
      wrapper.get('.floating-quota-widget__ring').attributes('aria-valuenow')
    ).toBe('32')
    expect(wrapper.get('.floating-quota-widget__status-dot').attributes('title')).toBe(
      '可用'
    )
    expect(wrapper.text()).toContain('3 天 0 小时后重置')
    expect(wrapper.text()).not.toContain(formatLocalDate(resetAt))
    expect(wrapper.get('.floating-quota-widget__monthly-remaining').text()).toBe(
      `月剩余74% · ${formatLocalDate(expiresAt)} 到期`
    )
    expectMembershipOnly(wrapper.text())
    wrapper.unmount()
  })

  it('keeps the ring and label on the same integer percentage', () => {
    const wrapper = mountWidget({
      ...demoOverview,
      quotas: [
        {
          ...demoOverview.quotas[0],
          usedPercent: 0.02357628572
        }
      ]
    })

    expect(wrapper.get('.floating-quota-widget__percent').text()).toBe('100%')
    expect(wrapper.get('.floating-quota-widget').classes()).toContain(
      'floating-quota-widget--three-digits'
    )
    expect(wrapper.text()).not.toContain('99.97642371428%')
    const style = wrapper
      .get('.floating-quota-widget__ring-value')
      .attributes('style') ?? ''
    const offset = Number(
      style.match(/stroke-dashoffset:\s*([\d.]+)px/)?.[1]
    )
    expect(offset).toBe(0)
    wrapper.unmount()
  })

  it.each<[QuotaState, string, string]>([
    ['available', '32%', '可用'],
    ['warning', '8%', '即将耗尽'],
    ['exhausted', '0%', '已耗尽'],
    ['unknown', '—', '暂不可用']
  ])('renders %s with the expected ring status', (state, percentage, label) => {
    const wrapper = mountWidget(createDemoOverview(state))
    const expectedClass = state === 'unknown' ? 'unavailable' : state

    expect(wrapper.get('.floating-quota-widget').classes()).toContain(
      `floating-quota-widget--${expectedClass}`
    )
    expect(wrapper.get('.floating-quota-widget__percent').text()).toBe(percentage)
    expect(wrapper.get('.floating-quota-widget__status-dot').attributes('title')).toBe(
      label
    )
    expectMembershipOnly(wrapper.text())
    wrapper.unmount()
  })

  it('covers no-membership, temporary-unavailable, and stale states', async () => {
    const noMembership = mountWidget(createDemoOverview('no-membership'))
    const unavailable = mountWidget(null, {
      status: 'unavailable',
      errorMessage: '网络暂不可用'
    })
    const stale = mountWidget(demoOverview, { dataStatus: 'stale' })

    expect(noMembership.get('.floating-quota-widget').classes()).toContain(
      'floating-quota-widget--no-membership'
    )
    expect(noMembership.text()).toContain('暂无会员订阅')
    expect(unavailable.get('.floating-quota-widget').classes()).toContain(
      'floating-quota-widget--unavailable'
    )
    expect(unavailable.text()).toContain('网络暂不可用')
    expect(unavailable.get('.floating-quota-widget').attributes('role')).toBe('group')
    expect(unavailable.get('.floating-quota-widget__refresh').attributes('type')).toBe(
      'button'
    )
    await unavailable.get('.floating-quota-widget__refresh').trigger('click')
    expect(unavailable.emitted('refresh')).toHaveLength(1)
    await unavailable.setProps({ refreshing: true })
    expect(unavailable.text()).toContain('正在重新获取额度')
    expect(unavailable.get('.floating-quota-widget__refresh').text()).toBe('获取中')
    expect(unavailable.get('.floating-quota-widget__refresh').attributes()).toHaveProperty(
      'disabled'
    )
    await unavailable.setProps({
      refreshing: false,
      lastRefreshAt: beforeReset.getTime()
    })
    expect(unavailable.text()).toContain('获取失败，可再次重试')
    expect(stale.get('.floating-quota-widget').classes()).toContain(
      'floating-quota-widget--stale'
    )
    expect(stale.get('.floating-quota-widget__monthly-remaining').text()).toBe(
      `月剩余74% · ${formatLocalDate(expiresAt)} 到期 旧数据`
    )
    expect(stale.get('.floating-quota-widget__percent').text()).toBe('32%')

    noMembership.unmount()
    unavailable.unmount()
    stale.unmount()
  })

  it('confirms a completed refresh when the server quota contract is incomplete', () => {
    const wrapper = mountWidget(createDemoOverview('unknown'), {
      status: 'ready',
      lastRefreshAt: beforeReset.getTime()
    })

    expect(wrapper.text()).toContain('已检查，服务端额度待更新')
    expect(wrapper.get('.floating-quota-widget__refresh').text()).toBe('重新获取')
    wrapper.unmount()
  })

  it('does not render a historical membership as the current floating quota', () => {
    const wrapper = mountWidget(createDemoOverview('expired'))

    expect(wrapper.get('.floating-quota-widget').classes()).toContain(
      'floating-quota-widget--no-membership'
    )
    expect(wrapper.text()).toContain('暂无会员订阅')
    expect(wrapper.text()).not.toContain('已于 7/29 14:00 到期')
    expect(wrapper.get('.floating-quota-widget__percent').text()).toBe('—')
    wrapper.unmount()
  })

  it('keeps an expired membership date compact when explicitly previewed', () => {
    const expired = createDemoOverview('expired')
    expired.quotas[0].isCurrentMembership = true
    const wrapper = mountWidget(expired)
    const expiredAt = new Date(expired.quotas[0].expiresAt ?? 0)

    expect(wrapper.get('.floating-quota-widget').classes()).toContain(
      'floating-quota-widget--expired'
    )
    expect(wrapper.get('.floating-quota-widget__reset').text()).toBe(
      `已于 ${formatLocalDate(expiredAt)} 到期`
    )
    expect(wrapper.get('.floating-quota-widget__reset').text()).not.toContain(':')
    wrapper.unmount()
  })

  it('renders the 80px compact surface and keeps click-to-expand', async () => {
    const wrapper = mountWidget(demoOverview, { collapsed: true })

    expect(wrapper.get('.floating-quota-widget').classes()).toContain(
      'floating-quota-widget--collapsed'
    )
    expect(wrapper.get('.floating-quota-widget').attributes('role')).toBe('button')
    expect(wrapper.get('.floating-quota-widget__compact').text()).toBe('32%')
    expect(wrapper.find('.floating-quota-widget__header').exists()).toBe(false)
    expect(wrapper.find('.floating-quota-widget__ring').exists()).toBe(false)
    expectMembershipOnly(wrapper.text())
    const widget = wrapper.get('.floating-quota-widget')
    await widget.trigger('click')
    expect(wrapper.emitted('toggle')).toHaveLength(1)
    wrapper.unmount()
  })

  it('starts a native drag after movement and suppresses its trailing click', async () => {
    const wrapper = mountWidget(demoOverview, { collapsed: true })
    const widget = wrapper.get('.floating-quota-widget')

    await widget.trigger('mousedown', {
      button: 0,
      screenX: 12,
      screenY: 12
    })
    await widget.trigger('mousemove', {
      button: 0,
      screenX: 24,
      screenY: 12
    })
    await widget.trigger('mouseup', {
      button: 0,
      screenX: 24,
      screenY: 12
    })
    await widget.trigger('click', {
      screenX: 24,
      screenY: 12
    })

    expect(desktopMocks.startQuotaViewerDrag).toHaveBeenCalledOnce()
    expect(wrapper.emitted('toggle')).toBeUndefined()
    wrapper.unmount()
  })
})
