import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'

import UserDashboardRecentUsage from '../UserDashboardRecentUsage.vue'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: {
    en: {
      dashboard: {
        recentUsage: 'Recent usage',
        last7Days: 'Last 7 days',
        noUsageRecords: 'No usage records',
        startUsingApi: 'Start using the API',
        model: 'Model',
        tokens: 'Tokens',
        actual: 'Actual',
        standard: 'Standard',
        viewAllUsage: 'View all usage',
        creditUnit: 'Snow credits',
      },
    },
  },
})

describe('UserDashboardRecentUsage', () => {
  it('uses snow credits for actual cost and USD for standard cost', () => {
    const wrapper = mount(UserDashboardRecentUsage, {
      props: {
        loading: false,
        data: [{
          id: 1,
          model: 'gpt-test',
          input_tokens: 100,
          output_tokens: 20,
          actual_cost: 0.125,
          total_cost: 0.25,
          created_at: '2026-07-27T00:00:00Z',
        }] as never,
      },
      global: {
        plugins: [i18n],
        stubs: {
          Icon: true,
          RouterLink: { template: '<a><slot /></a>' },
        },
      },
    })

    expect(wrapper.findAll('[data-testid="credit-amount-value"]').map((item) => item.text()))
      .toEqual(['0.1250'])
    expect(wrapper.findAll('[data-testid="snowflake-credit-icon"]')).toHaveLength(1)
    expect(wrapper.text()).toContain('$0.2500')
  })
})
