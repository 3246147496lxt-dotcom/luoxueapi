import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import AccountTodayStatsCell from '../AccountTodayStatsCell.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

vi.mock('@/i18n', () => ({
  getLocale: () => 'en-US',
  i18n: {
    global: {
      te: () => true,
      t: (key: string) => key
    }
  }
}))

describe('AccountTodayStatsCell', () => {
  it('keeps account cost in USD and renders user cost as snowflake credits', () => {
    const wrapper = mount(AccountTodayStatsCell, {
      props: {
        stats: {
          requests: 12,
          tokens: 3456,
          cost: 1.25,
          standard_cost: 1.5,
          user_cost: 1.8
        }
      }
    })

    expect(wrapper.text()).toContain('$1.25')
    expect(wrapper.get('[data-testid="credit-amount-value"]').text()).toBe('1.80')
    expect(wrapper.text()).not.toContain('$1.80')
  })
})
