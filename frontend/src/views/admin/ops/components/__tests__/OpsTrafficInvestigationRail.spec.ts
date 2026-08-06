import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { OpsDashboardOverview } from '@/api/admin/ops'
import OpsTrafficInvestigationRail from '../OpsTrafficInvestigationRail.vue'
import railSource from '../OpsTrafficInvestigationRail.vue?raw'

const { getGroups } = vi.hoisted(() => ({
  getGroups: vi.fn(),
}))

vi.mock('@/api', () => ({
  adminAPI: { groups: { getAll: getGroups } },
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

const overview: OpsDashboardOverview = {
  start_time: '2026-08-06T00:00:00Z',
  end_time: '2026-08-06T01:00:00Z',
  platform: '',
  group_id: null,
  health_score: 54,
  success_count: 972,
  error_count_total: 28,
  business_limited_count: 4,
  error_count_sla: 24,
  request_count_total: 1000,
  request_count_sla: 996,
  token_consumed: 1_200_000,
  sla: 0.9716,
  error_rate: 0.0284,
  upstream_error_rate: 0.02,
  upstream_error_count_excl_429_529: 20,
  upstream_429_count: 3,
  upstream_529_count: 1,
  qps: { current: 0.3, peak: 1.84, avg: 0.31 },
  tps: { current: 19_944.6, peak: 21_000, avg: 18_500 },
  duration: {},
  ttft: {},
}

function mountRail(loading = false) {
  return mount(OpsTrafficInvestigationRail, {
    props: {
      overview,
      platform: '',
      groupId: null,
      timeRange: '1h',
      loading,
    },
    global: {
      stubs: {
        Icon: true,
        Select: true,
      },
    },
  })
}

describe('OpsTrafficInvestigationRail', () => {
  beforeEach(() => {
    getGroups.mockReset().mockResolvedValue([])
  })

  it('默认预选健康卡，并保留每张卡的语义色状态', async () => {
    const wrapper = mountRail()
    await flushPromises()

    const signals = wrapper.findAll<HTMLButtonElement>('.ops-traffic-signal')
    const health = wrapper.get('[data-testid="ops-traffic-signal-health"]')
    const sla = wrapper.get('[data-testid="ops-traffic-signal-sla"]')
    const errors = wrapper.get('[data-testid="ops-traffic-signal-errors"]')
    const throughput = wrapper.get('[data-testid="ops-traffic-signal-throughput"]')

    expect(signals).toHaveLength(5)
    expect(signals.every((signal) => signal.element.tagName === 'BUTTON')).toBe(true)
    expect(signals.every((signal) => signal.attributes('type') === 'button')).toBe(true)
    expect(health.classes()).toEqual(
      expect.arrayContaining(['ops-traffic-signal--active', 'ops-traffic-signal--warning']),
    )
    expect(health.attributes('aria-pressed')).toBe('true')
    expect(signals.filter((signal) => signal.attributes('aria-pressed') === 'true')).toHaveLength(1)
    expect(sla.classes()).toContain('ops-traffic-signal--success')
    expect(errors.classes()).toContain('ops-traffic-signal--danger')
    expect(throughput.classes()).not.toContain('ops-traffic-signal--success')
    expect(wrapper.emitted('select-signal')).toBeUndefined()

    wrapper.unmount()
  })

  it('点击卡片后只切换选中框，并同步 aria-pressed 与事件', async () => {
    const wrapper = mountRail()
    const health = wrapper.get('[data-testid="ops-traffic-signal-health"]')
    const errors = wrapper.get('[data-testid="ops-traffic-signal-errors"]')

    await errors.trigger('click')

    expect(health.classes()).not.toContain('ops-traffic-signal--active')
    expect(health.attributes('aria-pressed')).toBe('false')
    expect(errors.classes()).toEqual(
      expect.arrayContaining(['ops-traffic-signal--active', 'ops-traffic-signal--danger']),
    )
    expect(errors.attributes('aria-pressed')).toBe('true')
    expect(wrapper.emitted('select-signal')).toEqual([['errors']])

    wrapper.unmount()
  })

  it('加载期间禁用全部信号按钮', () => {
    const wrapper = mountRail(true)
    const signals = wrapper.findAll<HTMLButtonElement>('.ops-traffic-signal')

    expect(signals).toHaveLength(5)
    expect(signals.every((signal) => signal.element.disabled)).toBe(true)

    wrapper.unmount()
  })

  it('锁定 Superdesign 的无框、悬停、选中和语义色契约', () => {
    expect(railSource).toContain('border: 1px solid transparent;')
    expect(railSource).toContain('background: #f4f1fa;')
    expect(railSource).toContain('background: #efebf5;')
    expect(railSource).toContain('border-color: rgba(91, 80, 112, 0.23);')
    expect(railSource).toContain('color: #d97706;')
    expect(railSource).toContain('color: #059669;')
    expect(railSource).toContain('color: #e11d48;')
    expect(railSource).toContain('outline: 2px solid #7c3aed;')
    expect(railSource).not.toContain(
      '.ops-traffic-signal--warning:not(.ops-traffic-signal--active)',
    )
  })
})
