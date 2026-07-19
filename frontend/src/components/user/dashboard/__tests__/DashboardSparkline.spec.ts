import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import DashboardSparkline from '../DashboardSparkline.vue'

vi.mock('vue-chartjs', () => ({
  Line: {
    name: 'LineChartStub',
    props: ['data', 'options'],
    template: '<span data-testid="line-chart" />',
  },
}))

describe('DashboardSparkline', () => {
  it('stays hidden without a non-zero finite value', () => {
    const empty = mount(DashboardSparkline, {
      props: { values: [], color: '#06b6d4' },
    })
    const zeroOnly = mount(DashboardSparkline, {
      props: { values: [0, Number.NaN, Number.POSITIVE_INFINITY, Number.NEGATIVE_INFINITY], color: '#06b6d4' },
    })

    expect(empty.find('[data-testid="line-chart"]').exists()).toBe(false)
    expect(zeroOnly.find('[data-testid="line-chart"]').exists()).toBe(false)
  })

  it('renders a sanitized, non-interactive straight line chart when data exists', () => {
    const wrapper = mount(DashboardSparkline, {
      props: {
        values: [2, Number.NaN, Number.POSITIVE_INFINITY, 5],
        color: '#f59e0b',
      },
    })

    expect(wrapper.get('[aria-hidden="true"]').classes()).toContain('dashboard-sparkline')

    const line = wrapper.getComponent({ name: 'LineChartStub' })
    const data = line.props('data')
    const options = line.props('options')

    expect(data.labels).toEqual(['0', '1', '2', '3'])
    expect(data.datasets[0]).toMatchObject({
      data: [2, 0, 0, 5],
      borderColor: '#f59e0b',
      borderWidth: 2,
      borderCapStyle: 'round',
      borderJoinStyle: 'round',
      pointRadius: 0,
      pointHoverRadius: 0,
      fill: false,
      tension: 0,
    })
    expect(options).toMatchObject({
      responsive: true,
      maintainAspectRatio: false,
      animation: false,
      events: [],
      plugins: {
        legend: { display: false },
        tooltip: { enabled: false },
      },
      scales: {
        x: { display: false },
        y: { display: false, beginAtZero: true },
      },
    })
  })
})
