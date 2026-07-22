import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import OpsErrorDistributionChart from '../OpsErrorDistributionChart.vue'
import OpsErrorTrendChart from '../OpsErrorTrendChart.vue'
import OpsLatencyChart from '../OpsLatencyChart.vue'
import OpsSwitchRateTrendChart from '../OpsSwitchRateTrendChart.vue'
import OpsThroughputTrendChart from '../OpsThroughputTrendChart.vue'

vi.mock('chart.js', () => ({
  Chart: { register: vi.fn() },
  ArcElement: {},
  BarElement: {},
  CategoryScale: {},
  Filler: {},
  Legend: {},
  LineElement: {},
  LinearScale: {},
  PointElement: {},
  Title: {},
  Tooltip: {}
}))

vi.mock('vue-chartjs', async () => {
  const { defineComponent } = await import('vue')

  const chartProps = {
    data: { type: Object, required: true },
    options: { type: Object, default: () => ({}) }
  }

  return {
    Bar: defineComponent({ name: 'BarChartStub', props: chartProps, template: '<div class="bar-stub" />' }),
    Doughnut: defineComponent({ name: 'Doughnut', props: chartProps, template: '<div class="doughnut-stub" />' }),
    Line: defineComponent({ name: 'LineChartStub', props: chartProps, template: '<div class="line-stub" />' })
  }
})

vi.mock('../../utils/opsFormatters', () => ({
  formatHistoryLabel: (date: string | undefined) => date ?? '',
  sumNumbers: (values: Array<number | null | undefined>) =>
    values.reduce<number>((total, value) => total + (typeof value === 'number' && Number.isFinite(value) ? value : 0), 0)
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()

  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const HelpTooltipStub = defineComponent({
  name: 'HelpTooltip',
  props: { content: { type: String, default: '' } },
  template: '<span class="help-tooltip-stub"><slot name="trigger" /></span>'
})

const EmptyStateStub = defineComponent({
  name: 'EmptyState',
  props: {
    title: { type: String, default: '' },
    description: { type: String, default: '' }
  },
  template: '<div class="empty-state-stub" />'
})

const global = {
  stubs: {
    HelpTooltip: HelpTooltipStub,
    EmptyState: EmptyStateStub
  }
}

const throughputPoint = {
  bucket_start: '2026-07-21T04:00:00Z',
  request_count: 12,
  token_consumed: 2400,
  switch_count: 2,
  qps: 1.2,
  tps: 720
}

describe('Ops chart visual contracts', () => {
  it('五类图表都使用紧凑的语义面板结构', () => {
    const wrappers = [
      mount(OpsLatencyChart, {
        props: { loading: false, latencyData: null },
        global
      }),
      mount(OpsErrorDistributionChart, {
        props: { loading: false, data: null },
        global
      }),
      mount(OpsErrorTrendChart, {
        props: { loading: false, points: [], timeRange: '1h' },
        global
      }),
      mount(OpsThroughputTrendChart, {
        props: { loading: false, points: [], timeRange: '1h' },
        global
      }),
      mount(OpsSwitchRateTrendChart, {
        props: { loading: false, points: [], timeRange: '5h' },
        global
      })
    ]

    for (const wrapper of wrappers) {
      expect(wrapper.element.tagName).toBe('SECTION')
      expect(wrapper.classes()).toContain('ops-chart-panel')
      expect(wrapper.find('.ops-chart-header').exists()).toBe(true)
      expect(wrapper.find('.ops-chart-title').exists()).toBe(true)
      expect(wrapper.find('.ops-chart-help').attributes('aria-label')).toBeTruthy()
      expect(wrapper.find('.ops-chart-empty').exists()).toBe(true)
      wrapper.unmount()
    }
  })

  it('错误分布详情操作保留可用态与下钻事件', async () => {
    const wrapper = mount(OpsErrorDistributionChart, {
      props: {
        loading: false,
        data: {
          total: 3,
          items: [{ status_code: 500, total: 3, sla: 3, business_limited: 0 }]
        }
      },
      global
    })

    const action = wrapper.get('.ops-chart-action')
    expect(action.attributes('disabled')).toBeUndefined()
    await action.trigger('click')
    expect(wrapper.emitted('openDetails')).toHaveLength(1)
  })

  it('错误趋势的两类详情操作仍按原事件分别下钻', async () => {
    const wrapper = mount(OpsErrorTrendChart, {
      props: {
        loading: false,
        timeRange: '1h',
        points: [
          {
            bucket_start: '2026-07-21T04:00:00Z',
            error_count_total: 5,
            business_limited_count: 0,
            error_count_sla: 2,
            upstream_error_count_excl_429_529: 3,
            upstream_429_count: 0,
            upstream_529_count: 0
          }
        ]
      },
      global
    })

    const actions = wrapper.findAll('.ops-chart-action')
    expect(actions).toHaveLength(2)
    await actions[0].trigger('click')
    await actions[1].trigger('click')
    expect(wrapper.emitted('openRequestErrors')).toHaveLength(1)
    expect(wrapper.emitted('openUpstreamErrors')).toHaveLength(1)
  })

  it('吞吐图保留分组筛选和详情下钻事件', async () => {
    const wrapper = mount(OpsThroughputTrendChart, {
      props: {
        loading: false,
        timeRange: '1h',
        points: [throughputPoint],
        topGroups: [{ group_id: 8, group_name: 'Claude', request_count: 12, token_consumed: 2400 }]
      },
      global
    })

    await wrapper.get('.ops-chart-filter').trigger('click')
    await wrapper.findAll('.ops-chart-action')[0].trigger('click')
    expect(wrapper.emitted('selectGroup')).toEqual([[8]])
    expect(wrapper.emitted('openDetails')).toHaveLength(1)
  })

  it('吞吐图没有分组数据时仍保留平台筛选事件', async () => {
    const wrapper = mount(OpsThroughputTrendChart, {
      props: {
        loading: false,
        timeRange: '1h',
        points: [throughputPoint],
        byPlatform: [{ platform: 'claude', request_count: 12, token_consumed: 2400 }]
      },
      global
    })

    await wrapper.get('.ops-chart-filter').trigger('click')
    expect(wrapper.emitted('selectPlatform')).toEqual([['claude']])
  })

  it('加载态使用可播报的紧凑骨架而不是装饰性转圈', () => {
    const wrapper = mount(OpsLatencyChart, {
      props: { loading: true, latencyData: null },
      global
    })

    const loading = wrapper.get('[role="status"]')
    expect(loading.attributes('aria-live')).toBe('polite')
    expect(loading.findAll('.ops-chart-loading__line')).toHaveLength(3)
  })
})
