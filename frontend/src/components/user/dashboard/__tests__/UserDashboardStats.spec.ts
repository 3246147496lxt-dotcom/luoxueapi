import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import CreditAmount from '@/components/common/CreditAmount.vue'
import UserDashboardStats from '../UserDashboardStats.vue'

const i18n = createI18n({
  legacy: false,
  locale: 'zh',
  messages: {
    zh: {
      dashboard: {
        accountMetrics: '账户指标',
        workspace: {
          balance: '可用余额',
          balanceHint: '余额说明',
          manageBalance: '管理余额',
          todayUsage: '今日使用',
          todayRequestsHint: '今日共 {count} 次请求',
          viewUsage: '查看使用记录',
          tokenConsumption: 'Token 消耗',
          tokenBreakdownHint: '输入 {input} · 输出 {output}',
          currentPlan: '当前套餐',
          planLoading: '正在读取套餐信息',
          planExpires: '有效期至 {date}',
          planNoExpiry: '当前套餐长期有效',
          planFlexible: '按需使用，随时管理',
          managePlan: '管理套餐',
        },
      },
    },
  },
})

const stats = {
  today_actual_cost: 1.25,
  today_requests: 3,
  today_tokens: 162,
  today_input_tokens: 100,
  today_output_tokens: 62,
} as never

function mountStats(overrides: Record<string, unknown> = {}) {
  return mount(UserDashboardStats, {
    props: {
      stats,
      balance: 0.66,
      planName: 'Pro',
      planExpiresAt: '2026-08-31T00:00:00Z',
      planLoading: false,
      subscriptionsLoaded: true,
      hasActiveSubscription: true,
      ...overrides,
    },
    global: {
      plugins: [i18n],
      stubs: {
        Icon: true,
        RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' },
      },
    },
  })
}

describe('UserDashboardStats', () => {
  it('renders the four end-user overview cards requested by the workspace IA', () => {
    const wrapper = mountStats()

    expect(wrapper.get('[data-testid="dashboard-metric-grid"]').findAll('.dashboard-metric-card')).toHaveLength(4)
    expect(wrapper.text()).toContain('dashboard.workspace.balance')
    expect(wrapper.text()).toContain('dashboard.workspace.todayUsage')
    expect(wrapper.text()).toContain('dashboard.workspace.tokenConsumption')
    expect(wrapper.text()).toContain('dashboard.workspace.currentPlan')
    expect(wrapper.text()).toContain('Pro')
    expect(wrapper.text()).toContain('dashboard.workspace.todayRequestsHint')
    expect(wrapper.text()).toContain('dashboard.workspace.tokenBreakdownHint')
  })

  it('uses the current balance and today actual cost without surfacing lifetime admin-like metrics', () => {
    const wrapper = mountStats()
    const credits = wrapper.findAllComponents(CreditAmount)

    expect(credits.map(component => component.props('value'))).toEqual(['0.66', '1.25'])
    expect(wrapper.text()).not.toContain('历史消耗')
    expect(wrapper.text()).not.toContain('RPM')
    expect(wrapper.text()).not.toContain('TPM')
  })

  it('uses Free as the membership label when there is no active subscription', () => {
    const wrapper = mountStats({
      planName: 'Free',
      planExpiresAt: null,
      hasActiveSubscription: false,
    })

    expect(wrapper.text()).toContain('Free')
    expect(wrapper.text()).toContain('dashboard.workspace.planFlexible')
  })
})
