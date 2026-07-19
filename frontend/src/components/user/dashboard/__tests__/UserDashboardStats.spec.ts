import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import CreditAmount from '@/components/common/CreditAmount.vue'
import DashboardSparkline from '../DashboardSparkline.vue'
import UserDashboardStats from '../UserDashboardStats.vue'

const i18n = createI18n({
  legacy: false,
  locale: 'zh',
  messages: {
    zh: {
      dashboard: {
        accountMetrics: '账户指标',
        accountData: '账户数据',
        currentBalance: '当前余额',
        lifetimeSpend: '历史消耗',
        usageStatistics: '使用统计',
        lifetimeRequests: '历史请求',
        rangeRequests: '区间请求',
        resourceUsage: '资源使用',
        rangeSpend: '区间消耗',
        rangeTokens: '区间 Token',
        performance: '性能',
        averageRpm: '平均 RPM',
        averageTpm: '平均 TPM',
      },
    },
  },
})

describe('UserDashboardStats', () => {
  it('uses snowflake credits for balance and consumption metrics', () => {
    const wrapper = mount(UserDashboardStats, {
      props: {
        stats: {
          total_actual_cost: 25.34,
          total_requests: 4,
        } as never,
        balance: 0.66,
        isSimple: false,
        rangeMetrics: {
          requests: 3,
          actualCost: 1.25,
          tokens: 162,
          averageRpm: 0.5,
          averageTpm: 20,
        },
        trend: [
          {
            date: '2026-07-10',
            requests: 1,
            input_tokens: 0,
            output_tokens: 0,
            cache_creation_tokens: 0,
            cache_read_tokens: 0,
            total_tokens: 10,
            cost: 0.1,
            actual_cost: 0.1,
          },
          {
            date: '2026-07-12',
            requests: 3,
            input_tokens: 0,
            output_tokens: 0,
            cache_creation_tokens: 0,
            cache_read_tokens: 0,
            total_tokens: 30,
            cost: 0.3,
            actual_cost: 0.3,
          },
        ],
        startDate: '2026-07-10',
        endDate: '2026-07-12',
        granularity: 'day',
      },
      global: {
        plugins: [i18n],
        stubs: { Icon: true, DashboardSparkline: true },
      },
    })

    const credits = wrapper.findAllComponents(CreditAmount)
    expect(credits.map(component => component.props('value'))).toEqual(['0.66', '25.3400', '1.2500'])
    expect(wrapper.text()).not.toContain('$')

    const iconNames = wrapper.findAll('icon-stub').map(icon => icon.attributes('name'))
    expect(iconNames).toEqual([
      'wallet',
      'arrowLeftRight',
      'chartNoAxesColumn',
      'activity',
      'send',
      'activity',
      'zap',
      'coins',
      'type',
      'gauge',
      'timer',
      'send',
    ])

    const sparklines = wrapper.findAllComponents(DashboardSparkline)
    expect(sparklines).toHaveLength(5)
    expect(sparklines.map(component => component.props('color'))).toEqual([
      '#06b6d4',
      '#f59e0b',
      '#ec4899',
      '#6366f1',
      '#f97316',
    ])
    expect(sparklines.map(component => component.props('values'))).toEqual([
      [1, 0, 3],
      [0.1, 0, 0.3],
      [10, 0, 30],
      [1, 0, 3],
      [10, 0, 30],
    ])
  })
})
