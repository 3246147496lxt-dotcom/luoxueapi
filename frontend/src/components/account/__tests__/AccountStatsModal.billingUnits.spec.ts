import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import type { Component } from 'vue'

import AccountStatsModal from '../AccountStatsModal.vue'
import AdminAccountStatsModal from '@/components/admin/account/AccountStatsModal.vue'
import type { Account, AccountUsageStatsResponse } from '@/types'

const { getStats } = vi.hoisted(() => ({
  getStats: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: { getStats },
  },
}))

vi.mock('chart.js', () => ({
  Chart: { register: vi.fn() },
  CategoryScale: {},
  LinearScale: {},
  PointElement: {},
  LineElement: {},
  Title: {},
  Tooltip: {},
  Legend: {},
  Filler: {},
}))

vi.mock('vue-chartjs', () => ({
  Line: {
    name: 'LineChartStub',
    props: {
      data: Object,
      options: Object,
    },
    template: '<div data-testid="line-chart" />',
  },
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const messages: Record<string, string> = {
    'usage.accountBilled': 'Account billed',
    'usage.userBilled': 'User billed',
    'dashboard.creditUnit': 'Points',
  }
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

const account = {
  id: 7,
  name: 'Test account',
  status: 'active',
} as Account

const stats: AccountUsageStatsResponse = {
  history: [{
    date: '2026-07-27',
    label: 'Jul 27',
    requests: 12,
    tokens: 3456,
    cost: 6.66,
    actual_cost: 8.88,
    user_cost: 9.99,
  }],
  summary: {
    days: 30,
    actual_days_used: 5,
    total_cost: 11.11,
    total_user_cost: 21.234,
    total_standard_cost: 33.33,
    total_requests: 100,
    total_tokens: 20_000,
    avg_daily_cost: 4.44,
    avg_daily_user_cost: 0.4567,
    avg_daily_requests: 20,
    avg_daily_tokens: 4_000,
    avg_duration_ms: 850,
    today: {
      date: '2026-07-27',
      cost: 5.55,
      user_cost: 1.234,
      requests: 12,
      tokens: 3_456,
    },
    highest_cost_day: {
      date: '2026-07-26',
      label: 'Jul 26',
      cost: 6.66,
      user_cost: 2.345,
      requests: 15,
    },
    highest_request_day: {
      date: '2026-07-25',
      label: 'Jul 25',
      requests: 25,
      cost: 7.77,
      user_cost: 3.456,
    },
  },
  models: [],
  endpoints: [],
  upstream_endpoints: [],
}

const CreditAmountStub = {
  name: 'CreditAmount',
  props: {
    value: [String, Number],
    iconSize: String,
  },
  template: '<span data-testid="credit-amount" :data-value="value" :data-icon-size="iconSize">{{ value }}</span>',
}

const ModelDistributionChartStub = {
  name: 'ModelDistributionChart',
  props: {
    modelStats: Array,
    loading: Boolean,
    creditMode: Boolean,
  },
  template: '<div data-testid="model-distribution" :data-credit-mode="String(creditMode)" />',
}

const EndpointDistributionChartStub = {
  name: 'EndpointDistributionChart',
  props: {
    endpointStats: Array,
    loading: Boolean,
    title: String,
    creditMode: Boolean,
  },
  template: '<div data-testid="endpoint-distribution" :data-credit-mode="String(creditMode)" />',
}

const mountModal = async (component: Component) => {
  const wrapper = mount(component, {
    props: { show: false, account },
    global: {
      stubs: {
        BaseDialog: {
          props: ['show'],
          template: '<div v-if="show"><slot /><slot name="footer" /></div>',
        },
        CreditAmount: CreditAmountStub,
        EndpointDistributionChart: EndpointDistributionChartStub,
        Icon: true,
        LoadingSpinner: true,
        ModelDistributionChart: ModelDistributionChartStub,
      },
    },
  })

  await wrapper.setProps({ show: true })
  await flushPromises()
  return wrapper
}

describe.each([
  ['account modal', AccountStatsModal, ['21.23', '0.457', '1.23', '2.35', '3.46', '1.23']],
  ['admin account modal', AdminAccountStatsModal, ['21.23', '0.457', '1.23', '2.35', '3.46']],
] as const)('%s billing units', (_label, component, expectedCreditValues) => {
  beforeEach(() => {
    getStats.mockReset()
    getStats.mockResolvedValue(stats)
  })

  it('renders user charges as Points while account and standard costs remain USD', async () => {
    const wrapper = await mountModal(component)
    const creditValues = wrapper.findAll('[data-testid="credit-amount"]')
      .map((node) => node.attributes('data-value'))

    expect(creditValues).toEqual(expectedCreditValues)
    expect(wrapper.text()).toContain('$11.11')
    expect(wrapper.text()).toContain('$33.33')
    expect(wrapper.text()).toContain('$4.44')
    expect(wrapper.text()).toContain('$5.55')
    expect(wrapper.text()).toContain('$6.66')
    expect(wrapper.text()).toContain('$7.77')
    expect(wrapper.get('[data-testid="model-distribution"]').attributes('data-credit-mode')).toBe('false')
    expect(wrapper.findAll('[data-testid="endpoint-distribution"]')).toHaveLength(2)
    expect(wrapper.findAll('[data-testid="endpoint-distribution"]')
      .every((node) => node.attributes('data-credit-mode') === 'false')).toBe(true)
    expectedCreditValues.forEach((value) => {
      expect(wrapper.text()).not.toContain(`$${value}`)
    })
  }, 30_000)

  it('keeps account history in USD and user charges in Points', async () => {
    const wrapper = await mountModal(component)
    const line = wrapper.getComponent({ name: 'LineChartStub' })
    const data = line.props('data') as {
      datasets: Array<{ label: string; data: number[]; yAxisID: string; billingUnit: string }>
    }
    const options = line.props('options') as {
      plugins: { tooltip: { callbacks: { label: (context: unknown) => string } } }
      scales: {
        yUsd: {
          ticks: { callback: (value: number) => string }
          title: { text: string }
        }
        yCredit: {
          ticks: { callback: (value: number) => string }
          title: { text: string }
        }
      }
    }

    expect(data.datasets[0]).toMatchObject({
      label: 'Account billed (USD)',
      data: [8.88],
      yAxisID: 'yUsd',
      billingUnit: 'USD',
    })
    expect(data.datasets[1]).toMatchObject({
      label: 'User billed (Points)',
      data: [9.99],
      yAxisID: 'yCredit',
      billingUnit: 'CREDIT',
    })
    expect(options.plugins.tooltip.callbacks.label({
      dataset: data.datasets[0],
      raw: 8.88,
    })).toBe('Account billed (USD): $8.88')
    expect(options.plugins.tooltip.callbacks.label({
      dataset: data.datasets[1],
      raw: 9.99,
    })).toBe('User billed (Points): 9.99 Points')
    expect(options.scales.yUsd.ticks.callback(8.88)).toBe('$8.88')
    expect(options.scales.yUsd.title.text).toBe('Account billed (USD)')
    expect(options.scales.yCredit.ticks.callback(9.99)).toBe('9.99')
    expect(options.scales.yCredit.title.text).toBe('User billed (Points)')
  }, 30_000)
})
