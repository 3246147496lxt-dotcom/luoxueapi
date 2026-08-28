import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import EndpointDistributionChart from '../EndpointDistributionChart.vue'

const messages: Record<string, string> = {
  'usage.endpointDistribution': 'Endpoint Distribution',
  'usage.endpoint': 'Endpoint',
  'admin.dashboard.requests': 'Requests',
  'admin.dashboard.tokens': 'Tokens',
  'admin.dashboard.actual': 'Actual',
  'admin.dashboard.standard': 'Standard',
  'admin.dashboard.noDataAvailable': 'No data available',
  'dashboard.creditUnit': 'Points',
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
  Doughnut: {
    props: ['data'],
    template: '<div class="chart-data">{{ JSON.stringify(data) }}</div>',
  },
}))

describe('EndpointDistributionChart', () => {
  const endpointStats = [
    { endpoint: '/v1/messages', requests: 9, total_tokens: 1200, cost: 1.8, actual_cost: 0.1 },
    { endpoint: '/v1/responses', requests: 4, total_tokens: 600, cost: 0.7, actual_cost: 0.9 },
  ]

  it('renders actual cost as points while preserving standard USD in credit mode', () => {
    const wrapper = mount(EndpointDistributionChart, {
      props: {
        endpointStats,
        metric: 'actual_cost',
        enableBreakdown: false,
        creditMode: true,
      },
    })

    expect(wrapper.findAll('[data-testid="credit-amount-value"]').map((item) => item.text()))
      .toEqual(['0.900', '0.100'])
    expect(wrapper.text()).toContain('$0.700')
    expect(wrapper.text()).toContain('$1.80')

    const options = (wrapper.vm as any).$?.setupState.doughnutOptions
    expect(options.plugins.tooltip.callbacks.label({
      label: '/v1/responses',
      raw: 0.9,
      dataset: { data: [0.9, 0.1] },
    })).toBe('/v1/responses: Points 0.900 (90.0%)')
  })

  it('preserves dollar formatting by default for admin consumers', () => {
    const wrapper = mount(EndpointDistributionChart, {
      props: { endpointStats, enableBreakdown: false },
    })

    expect(wrapper.text()).toContain('$0.100')
    expect(wrapper.text()).toContain('$1.80')
    expect(wrapper.find('[data-testid="credit-amount"]').exists()).toBe(false)
  })
})
