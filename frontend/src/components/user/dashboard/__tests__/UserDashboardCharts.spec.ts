import { afterAll, beforeAll, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import type { TrendDataPoint } from '@/types'
import UserDashboardCharts from '../UserDashboardCharts.vue'

vi.mock('vue-chartjs', () => ({
  Line: {
    name: 'Line',
    props: ['data', 'options', 'plugins'],
    template: '<div class="line-chart" />',
  },
  Bar: {
    name: 'Bar',
    props: ['data', 'options', 'plugins'],
    template: '<div class="bar-chart" />',
  },
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const { ref } = await import('vue')
  const messages: Record<string, string> = {
    'common.loading': '加载中',
    'usage.time': '时间',
    'dashboard.workspace.usageStatistics': '使用统计',
    'dashboard.workspace.statisticsPeriod': '统计周期',
    'dashboard.workspace.periods.today': '今天',
    'dashboard.workspace.periods.week': '本周',
    'dashboard.workspace.periods.month': '本月',
    'dashboard.workspace.periods.thirtyDays': '近30天',
    'dashboard.workspace.creditsTrend': '积分消耗趋势',
    'dashboard.workspace.creditsUsed': '积分消耗',
    'dashboard.workspace.creditsUnit': '积分',
    'dashboard.workspace.token': 'Token',
    'dashboard.workspace.requests': '请求次数',
    'dashboard.workspace.cachedInput': '输入（缓存）',
    'dashboard.workspace.uncachedInput': '输入（未缓存）',
    'dashboard.workspace.outputTokens': '输出',
    'dashboard.workspace.noUsageData': '暂无使用记录',
  }
  return {
    ...actual,
    useI18n: () => ({
      locale: ref('zh'),
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

const trend: TrendDataPoint[] = [
  {
    date: '2026-08-16 08:00:00',
    requests: 62,
    input_tokens: 600,
    output_tokens: 300,
    cache_creation_tokens: 100,
    cache_read_tokens: 200,
    total_tokens: 1200,
    cost: 20,
    actual_cost: 10,
  },
  {
    date: '2026-08-16 09:00:00',
    requests: 80,
    input_tokens: 900,
    output_tokens: 400,
    cache_creation_tokens: 100,
    cache_read_tokens: 400,
    total_tokens: 1800,
    cost: 30,
    actual_cost: 15,
  },
]

beforeAll(() => {
  vi.useFakeTimers()
  vi.setSystemTime(new Date(2026, 7, 16, 10, 0))
})

afterAll(() => {
  vi.useRealTimers()
})

function mountCharts() {
  return mount(UserDashboardCharts, {
    props: { loading: false, period: 'today', trend },
  })
}

describe('UserDashboardCharts', () => {
  it('uses one compact heading total for each of the three real usage charts', () => {
    const wrapper = mountCharts()

    expect(wrapper.get('[data-testid="credits-total"]').text()).toBe('25')
    expect(wrapper.get('[data-testid="tokens-total"]').text()).toBe('3,000')
    expect(wrapper.get('[data-testid="requests-total"]').text()).toBe('142')
    expect(wrapper.findAll('.dashboard-chart-card__heading')).toHaveLength(3)
    expect(wrapper.findAll('.dashboard-chart-card__heading > span')).toHaveLength(3)
    expect(wrapper.findAll('.dashboard-chart-card__heading > strong')).toHaveLength(0)
    expect(wrapper.findAll('.line-chart')).toHaveLength(1)
    expect(wrapper.findAll('.bar-chart')).toHaveLength(2)
  })

  it('keeps credits orange and the two bar charts in a restrained single-color purple hierarchy', () => {
    const wrapper = mountCharts()
    const line = wrapper.findComponent({ name: 'Line' })
    const bars = wrapper.findAllComponents({ name: 'Bar' })

    expect(line.props('data').datasets[0].borderColor).toBe('#FF8A00')
    expect(line.props('data').datasets[0].pointRadius).toBe(0)
    expect(bars[0].props('data').datasets).toHaveLength(1)
    expect(bars[0].props('data').datasets[0].backgroundColor).toBe('#8B5CF6')
    expect(bars[1].props('data').datasets).toHaveLength(1)
    expect(bars[1].props('data').datasets[0].backgroundColor).toBe('#A78BFA')
  })

  it('shares sparse axes, three y ticks and detailed token tooltip data', () => {
    const wrapper = mountCharts()
    const creditsChart = wrapper.findComponent({ name: 'Line' })
    const tokenChart = wrapper.findAllComponents({ name: 'Bar' })[0]
    const creditsOptions = creditsChart.props('options')
    const options = tokenChart.props('options')

    expect(creditsOptions.scales.x.ticks.font).toMatchObject({ size: 10, weight: 500 })
    expect(creditsOptions.scales.x.ticks.color).toBe('rgba(100, 116, 139, 0.60)')
    expect(creditsOptions.scales.y.grid.color).toBe('rgba(226, 232, 240, 0.70)')
    expect(options.scales.x.ticks.maxTicksLimit).toBe(4)
    expect(options.scales.x.ticks.font).toMatchObject({ size: 9, weight: 500 })
    expect(options.scales.y.ticks.count).toBe(3)
    expect(options.plugins.tooltip.cornerRadius).toBe(12)
    expect(options.plugins.tooltip.borderWidth).toBe(1)
    expect(options.plugins.tooltip.callbacks.label({ dataIndex: 8 })).toEqual([
      '输入（缓存）: 200',
      '输入（未缓存）: 700',
      '输出: 300',
    ])
    expect(tokenChart.props('plugins')).toHaveLength(1)
  })

  it('emits a single global period change for the complete statistics section', async () => {
    const wrapper = mountCharts()

    await wrapper.get('[data-testid="dashboard-period-trigger"]').trigger('click')
    expect(wrapper.get('[data-testid="dashboard-period-trigger"]').attributes('aria-expanded')).toBe('true')
    expect(wrapper.get('[data-testid="dashboard-period-menu"]')).toBeTruthy()

    await wrapper.get('[data-period="month"]').trigger('click')
    expect(wrapper.emitted('update:period')).toEqual([['month']])
    expect(wrapper.get('[data-testid="dashboard-period-trigger"]').attributes('aria-expanded')).toBe('false')
  })
})
