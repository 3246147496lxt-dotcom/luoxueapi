import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import UserDashboardStats from '../UserDashboardStats.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const { ref } = await import('vue')
  const messages: Record<string, string> = {
    'dashboard.accountMetrics': '账户指标',
    'dashboard.workspace.accountBalance': '账户余额',
    'dashboard.workspace.planQuota': '{plan}额度',
    'dashboard.workspace.planLoading': '正在读取套餐信息',
    'dashboard.workspace.remaining': '剩余',
    'dashboard.workspace.cumulativeTokens': '累计Token',
    'dashboard.workspace.cumulativeSpend': '累计积分消耗',
    'dashboard.workspace.creditsUnit': '积分',
    'dashboard.workspace.tokenUnit': 'Token',
    'dashboard.workspace.tokenBreakdownHint': '输入 {input} · 输出 {output}',
    'payment.tabTopUp': '充值',
  }
  return {
    ...actual,
    useI18n: () => ({
      locale: ref('zh'),
      t: (key: string, params: Record<string, unknown> = {}) => Object.entries(params)
        .reduce((text, [name, value]) => text.replace(`{${name}}`, String(value)), messages[key] ?? key),
    }),
  }
})

const stats = {
  total_tokens: 200,
  total_input_tokens: 100,
  total_output_tokens: 50,
  total_cache_creation_tokens: 20,
  total_cache_read_tokens: 30,
  total_actual_cost: 3420,
} as never

function mountStats(overrides: Record<string, unknown> = {}) {
  return mount(UserDashboardStats, {
    props: {
      stats,
      balance: 1280,
      planName: 'Ultra',
      quotaRemainingPercent: 72.8,
      planLoading: false,
      ...overrides,
    },
    global: {
      stubs: {
        RouterLink: {
          props: ['to'],
          template: '<a :href="to"><slot /></a>',
        },
      },
    },
  })
}

describe('UserDashboardStats', () => {
  it('renders the four stable account status cards from real summary data', () => {
    const wrapper = mountStats()

    expect(wrapper.get('[data-testid="dashboard-metric-grid"]').findAll('.dashboard-metric-card')).toHaveLength(4)
    expect(wrapper.text()).toContain('账户余额')
    expect(wrapper.text()).toContain('1,280')
    expect(wrapper.text()).toContain('Ultra额度')
    expect(wrapper.text()).toContain('72.8%')
    expect(wrapper.text()).toContain('累计Token')
    expect(wrapper.text()).toContain('200')
    expect(wrapper.text()).toContain('累计积分消耗')
    expect(wrapper.text()).toContain('3,420')

    const recharge = wrapper.get('[data-testid="dashboard-balance-recharge"]')
    expect(recharge.text()).toBe('充值')
    expect(recharge.attributes('href')).toBe('/purchase')
  })

  it('keeps the cumulative token breakdown weak and reconciled with cached input', () => {
    const wrapper = mountStats()

    expect(wrapper.get('.dashboard-metric-card__detail').text()).toBe('输入 150 · 输出 50')
    expect(wrapper.findAll('a')).toHaveLength(1)
  })

  it.each([
    [72.8, 'healthy'],
    [35, 'attention'],
    [12, 'critical'],
    [null, 'neutral'],
  ])('maps %s percent remaining to the %s quota state', (quotaRemainingPercent, tone) => {
    const wrapper = mountStats({ quotaRemainingPercent })

    expect(wrapper.get('.dashboard-quota-track__value').classes()).toContain(
      `dashboard-quota-track__value--${tone}`,
    )
  })
})
