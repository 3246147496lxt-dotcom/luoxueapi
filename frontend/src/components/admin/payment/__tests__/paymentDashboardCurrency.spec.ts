import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import OrderStatsCards from '../OrderStatsCards.vue'
import DailyRevenueChart from '../DailyRevenueChart.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

vi.mock('chart.js', () => ({
  Chart: { register: vi.fn() },
  CategoryScale: {},
  LinearScale: {},
  PointElement: {},
  LineElement: {},
  Tooltip: {},
  Legend: {},
  Filler: {},
}))

vi.mock('vue-chartjs', () => ({
  Line: {
    name: 'LineChartStub',
    props: ['data', 'options'],
    template: '<div data-testid="line-chart" />',
  },
}))

const stats = {
  currency: 'HKD',
  available_currencies: ['CNY', 'HKD'],
  today_amount: 12.5,
  total_amount: 80,
  today_count: 2,
  total_count: 4,
  avg_amount: 20,
  pending_orders: 1,
  daily_series: [],
  payment_methods: [],
  top_users: [],
}

describe('payment dashboard currency', () => {
  it('formats summary cards in the selected payment currency', () => {
    const wrapper = mount(OrderStatsCards, { props: { stats } })

    expect(wrapper.text()).toMatch(/HK\$12\.50|HKD\s*12\.50/)
    expect(wrapper.text()).toMatch(/HK\$80\.00|HKD\s*80\.00/)
    expect(wrapper.text()).not.toMatch(/(?:^|[^A-Z])\$12\.50/)
  })

  it('labels and formats the revenue axis in the selected payment currency', () => {
    const wrapper = mount(DailyRevenueChart, {
      props: {
        currency: 'HKD',
        data: [{ date: '2026-07-27', amount: 80, count: 2 }],
      },
    })
    const line = wrapper.getComponent({ name: 'LineChartStub' })
    const data = line.props('data') as { datasets: Array<{ label: string }> }
    const options = line.props('options') as {
      scales: { y: { ticks: { callback: (value: number) => string } } }
    }

    expect(data.datasets[0].label).toContain('(HKD)')
    expect(options.scales.y.ticks.callback(80)).toMatch(/HK\$80\.00|HKD\s*80\.00/)
  })
})
