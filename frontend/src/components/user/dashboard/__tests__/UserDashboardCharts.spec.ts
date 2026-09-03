import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
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

const lightChartTokens: Record<string, string> = {
  '--workspace-dashboard-text-muted': '#64748b',
  '--workspace-dashboard-card-border': '#f1f5f9',
  '--workspace-dashboard-text-subtle': '#64748b',
  '--workspace-dashboard-tooltip-surface': '#0f172a',
  '--workspace-dashboard-tooltip-text': '#f8fafc',
  '--workspace-dashboard-tooltip-title': '#cbd5e1',
  '--workspace-dashboard-tooltip-border': 'rgb(255 255 255 / 0.1)',
  '--workspace-card-surface': '#ffffff',
  '--workspace-work-chart-cost': '#ea580c',
  '--workspace-work-chart-cost-fill-strong': 'rgb(234 88 12 / 0.18)',
  '--workspace-work-chart-cost-fill-soft': 'rgb(234 88 12 / 0.06)',
  '--workspace-work-chart-cost-fill-transparent': 'rgb(234 88 12 / 0)',
  '--workspace-work-chart-primary': '#2563eb',
  '--workspace-work-chart-primary-fill-strong': 'rgb(37 99 235 / 0.18)',
  '--workspace-work-chart-primary-fill-soft': 'rgb(37 99 235 / 0.06)',
  '--workspace-work-chart-primary-fill-transparent': 'rgb(37 99 235 / 0)',
  '--workspace-work-chart-secondary': '#60a5fa',
  '--workspace-work-accent': '#2563eb',
  '--workspace-work-accent-hover': '#1d4ed8',
}

const darkChartTokens: Record<string, string> = {
  '--workspace-dashboard-text-muted': '#8a8a8a',
  '--workspace-dashboard-card-border': 'rgb(255 255 255 / 0.1)',
  '--workspace-dashboard-text-subtle': '#8a8a8a',
  '--workspace-dashboard-tooltip-surface': '#212121',
  '--workspace-dashboard-tooltip-text': '#ececec',
  '--workspace-dashboard-tooltip-title': '#b4b4b4',
  '--workspace-dashboard-tooltip-border': 'rgb(255 255 255 / 0.16)',
  '--workspace-card-surface': '#171717',
  '--workspace-work-chart-cost': '#fb923c',
  '--workspace-work-chart-cost-fill-strong': 'rgb(251 146 60 / 0.18)',
  '--workspace-work-chart-cost-fill-soft': 'rgb(251 146 60 / 0.06)',
  '--workspace-work-chart-cost-fill-transparent': 'rgb(251 146 60 / 0)',
  '--workspace-work-chart-primary': '#60a5fa',
  '--workspace-work-chart-primary-fill-strong': 'rgb(96 165 250 / 0.18)',
  '--workspace-work-chart-primary-fill-soft': 'rgb(96 165 250 / 0.06)',
  '--workspace-work-chart-primary-fill-transparent': 'rgb(96 165 250 / 0)',
  '--workspace-work-chart-secondary': '#93c5fd',
  '--workspace-work-accent': '#60a5fa',
  '--workspace-work-accent-hover': '#93c5fd',
}

function applyChartTokens(tokens: Record<string, string>): void {
  for (const [token, value] of Object.entries(tokens)) {
    document.documentElement.style.setProperty(token, value)
  }
}

beforeAll(() => {
  vi.useFakeTimers()
  vi.setSystemTime(new Date(2026, 7, 16, 10, 0))
})

beforeEach(() => {
  document.documentElement.classList.remove('dark')
  applyChartTokens(lightChartTokens)
})

afterEach(() => {
  document.documentElement.classList.remove('dark')
  for (const token of new Set([...Object.keys(lightChartTokens), ...Object.keys(darkChartTokens)])) {
    document.documentElement.style.removeProperty(token)
  }
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
    expect(wrapper.findAll('.line-chart')).toHaveLength(2)
    expect(wrapper.findAll('.bar-chart')).toHaveLength(1)
  })

  it('uses matching orange and blue area lines while keeping requests as a lighter bar chart', () => {
    const wrapper = mountCharts()
    const lines = wrapper.findAllComponents({ name: 'Line' })
    const bar = wrapper.findComponent({ name: 'Bar' })

    expect(lines[0].props('data').datasets[0].borderColor).toBe('#ea580c')
    expect(lines[0].props('data').datasets[0].pointRadius).toBe(0)
    expect(lines[1].props('data').datasets[0]).toMatchObject({
      borderColor: '#2563eb',
      borderWidth: 2,
      fill: true,
      tension: 0.36,
      cubicInterpolationMode: 'monotone',
      pointRadius: 0,
      pointHoverBorderColor: '#2563eb',
    })
    expect(lines[1].props('data').datasets[0].backgroundColor({
      chart: { chartArea: undefined },
    })).toBe('rgb(37 99 235 / 0.18)')
    expect(bar.props('data').datasets).toHaveLength(1)
    expect(bar.props('data').datasets[0].backgroundColor).toBe('#60a5fa')
    expect(bar.props('data').datasets[0].hoverBackgroundColor).toBe('#2563eb')
  })

  it('shares sparse axes, three y ticks and detailed token tooltip data', () => {
    const wrapper = mountCharts()
    const [creditsChart, tokenChart] = wrapper.findAllComponents({ name: 'Line' })
    const creditsOptions = creditsChart.props('options')
    const options = tokenChart.props('options')

    expect(creditsOptions.scales.x.ticks.font).toMatchObject({ size: 10, weight: 500 })
    expect(creditsOptions.scales.x.ticks.color).toBe('#64748b')
    expect(creditsOptions.scales.y.grid.color).toBe('#f1f5f9')
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

  it('resolves the dark chart palette from the active semantic tokens', () => {
    document.documentElement.classList.add('dark')
    applyChartTokens(darkChartTokens)

    const wrapper = mountCharts()
    const lines = wrapper.findAllComponents({ name: 'Line' })
    const bar = wrapper.findComponent({ name: 'Bar' })
    const options = lines[1].props('options')

    expect(lines[0].props('data').datasets[0].borderColor).toBe('#fb923c')
    expect(lines[1].props('data').datasets[0].borderColor).toBe('#60a5fa')
    expect(lines[1].props('data').datasets[0].pointHoverBackgroundColor).toBe('#171717')
    expect(bar.props('data').datasets[0].backgroundColor).toBe('#93c5fd')
    expect(options.scales.x.ticks.color).toBe('#8a8a8a')
    expect(options.plugins.tooltip.backgroundColor).toBe('#212121')
    expect(options.plugins.tooltip.borderColor).toBe('rgb(255 255 255 / 0.16)')
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
