import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import type { TrendDataPoint } from '@/types'
import TokenUsageTrend from '../TokenUsageTrend.vue'

const messages: Record<string, string> = {
  'admin.dashboard.usageTrendEyebrow': 'USAGE TREND',
  'admin.dashboard.tokenUsageTrend': 'Token Usage Trend',
  'admin.dashboard.noDataAvailable': 'No data available',
  'admin.dashboard.startUsingApi': 'Usage will appear here after API calls begin.',
  'admin.dashboard.failedToLoad': 'Failed to load dashboard statistics',
  'admin.dashboard.retry': 'Reload',
  'admin.dashboard.tokens': 'Tokens',
  'admin.dashboard.spendShort': 'Spend',
  'admin.dashboard.input': 'Input',
  'admin.dashboard.output': 'Output',
  'admin.dashboard.cache': 'Cache',
  'admin.dashboard.actual': 'Actual',
  'admin.dashboard.standard': 'Standard',
  'admin.dashboard.totalTokens': 'Total Tokens',
  'admin.dashboard.totalCost': 'Total Cost',
  'admin.dashboard.currentRangeTotal': 'Selected-period total',
  'admin.dashboard.currentRangeCost': 'Selected-period cost',
  'common.loading': 'Loading...',
  'usage.time': 'Time',
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

vi.mock('vue-chartjs', () => ({
  Line: {
    props: ['data', 'options'],
    template: '<div class="chart-data">{{ JSON.stringify(data) }}</div>',
  },
}))

const createPoint = (overrides: Partial<TrendDataPoint> = {}): TrendDataPoint => ({
  date: '2026-05-08',
  requests: 1,
  input_tokens: 500,
  output_tokens: 100,
  cache_creation_tokens: 0,
  cache_read_tokens: 1500,
  total_tokens: 2100,
  cost: 0.01,
  actual_cost: 0.005,
  ...overrides,
})

const mountTrend = (props: InstanceType<typeof TokenUsageTrend>['$props']) => mount(
  TokenUsageTrend,
  {
    props,
    global: {
      stubs: {
        LoadingSpinner: true,
      },
    },
  }
)

const readChartData = (wrapper: ReturnType<typeof mountTrend>) => JSON.parse(
  wrapper.find('.chart-data').text()
)

afterEach(() => {
  document.documentElement.classList.remove('dark')
})

describe('TokenUsageTrend', () => {
  it('calculates cache hit rate against all prompt tokens in the default variant', () => {
    const wrapper = mountTrend({
      trendData: [createPoint()],
    })

    const chartData = readChartData(wrapper)
    const hitRateDataset = chartData.datasets.find(
      (dataset: { label: string }) => dataset.label === 'Cache Hit Rate'
    )
    // Hit rate = 1500 / (500 + 1500 + 0) * 100 = 75%
    expect(hitRateDataset.data[0]).toBe(75)
  })

  it('returns 0 hit rate when all prompt tokens are zero', () => {
    const wrapper = mountTrend({
      trendData: [createPoint({
        requests: 0,
        input_tokens: 0,
        output_tokens: 0,
        cache_creation_tokens: 0,
        cache_read_tokens: 0,
        total_tokens: 0,
        cost: 0,
        actual_cost: 0,
      })],
    })

    const chartData = readChartData(wrapper)
    const hitRateDataset = chartData.datasets.find(
      (dataset: { label: string }) => dataset.label === 'Cache Hit Rate'
    )
    expect(hitRateDataset.data[0]).toBe(0)
  })

  it('includes cache creation tokens in the hit-rate denominator', () => {
    const wrapper = mountTrend({
      trendData: [createPoint({
        input_tokens: 200,
        output_tokens: 50,
        cache_creation_tokens: 300,
        cache_read_tokens: 500,
        total_tokens: 1050,
        cost: 0.02,
        actual_cost: 0.01,
      })],
    })

    const chartData = readChartData(wrapper)
    const hitRateDataset = chartData.datasets.find(
      (dataset: { label: string }) => dataset.label === 'Cache Hit Rate'
    )
    // Hit rate = 500 / (200 + 500 + 300) * 100 = 50%
    expect(hitRateDataset.data[0]).toBe(50)
  })

  it('uses backend totals and combines both cache token fields in the home-clay view', () => {
    const wrapper = mountTrend({
      variant: 'home-clay',
      trendData: [
        createPoint({
          date: '00:00',
          input_tokens: 600,
          output_tokens: 200,
          cache_creation_tokens: 50,
          cache_read_tokens: 150,
          total_tokens: 1000,
        }),
        createPoint({
          date: '08:00',
          input_tokens: 1200,
          output_tokens: 400,
          cache_creation_tokens: 100,
          cache_read_tokens: 300,
          total_tokens: 2000,
        }),
      ],
    })

    expect(wrapper.get('[data-testid="token-trend-total"]').text()).toBe('3.00K')
    expect(wrapper.get('[role="img"]').attributes('aria-label')).toContain('Selected-period total 3.00K')

    const chartData = readChartData(wrapper)
    expect(chartData.datasets.map((dataset: { label: string }) => dataset.label)).toEqual([
      'Input',
      'Output',
      'Cache',
    ])
    expect(chartData.datasets[2].data).toEqual([200, 400])
  })

  it('switches to real actual and standard cost series without deriving a fake split', async () => {
    const wrapper = mountTrend({
      variant: 'home-clay',
      trendData: [
        createPoint({ date: '00:00', actual_cost: 0.25, cost: 0.5 }),
        createPoint({ date: '08:00', actual_cost: 1.5, cost: 2 }),
      ],
    })

    const buttons = wrapper.findAll('.token-trend-switch button')
    expect(buttons[0].attributes('aria-pressed')).toBe('true')
    await buttons[1].trigger('click')

    expect(buttons[1].attributes('aria-pressed')).toBe('true')
    expect(wrapper.get('[data-testid="token-trend-total"]').text()).toBe('$1.75')

    const chartData = readChartData(wrapper)
    expect(chartData.datasets.map((dataset: { label: string }) => dataset.label)).toEqual([
      'Actual',
      'Standard',
    ])
    expect(chartData.datasets[0].data).toEqual([0.25, 1.5])
    expect(chartData.datasets[1].data).toEqual([0.5, 2])
  })

  it('uses the dark semantic chart palette when dark mode is active', () => {
    document.documentElement.classList.add('dark')
    const wrapper = mountTrend({
      variant: 'home-clay',
      trendData: [createPoint()],
    })

    const chartData = readChartData(wrapper)
    expect(chartData.datasets[0].borderColor).toBe('#a78bfa')
    expect(chartData.datasets[1].borderColor).toBe('#38bdf8')
    expect(chartData.datasets[2].borderColor).toBe('#34d399')
  })

  it('renders an accessible skeleton and an explanatory empty state', async () => {
    const wrapper = mountTrend({
      variant: 'home-clay',
      trendData: [],
      loading: true,
    })

    expect(wrapper.get('[role="status"]').text()).toContain('Loading...')
    expect(wrapper.find('.token-trend-skeleton-chart').exists()).toBe(true)

    await wrapper.setProps({ loading: false })
    expect(wrapper.get('[data-testid="token-trend-empty"]').text()).toContain('No data available')
    expect(wrapper.get('[data-testid="token-trend-empty"]').text()).toContain(
      'Usage will appear here after API calls begin.'
    )
  })

  it('announces failures and emits retry when requested', async () => {
    const wrapper = mountTrend({
      variant: 'home-clay',
      trendData: [],
      error: 'Network unavailable',
    })

    const errorState = wrapper.get('[data-testid="token-trend-error"]')
    expect(errorState.attributes('role')).toBe('alert')
    expect(errorState.text()).toContain('Network unavailable')

    await errorState.get('button').trigger('click')
    expect(wrapper.emitted('retry')).toHaveLength(1)
  })
})
