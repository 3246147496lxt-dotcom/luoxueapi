import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import UserBreakdownSubTable from '../UserBreakdownSubTable.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

const items = [{
  user_id: 1,
  email: 'user@example.com',
  requests: 2,
  input_tokens: 100,
  output_tokens: 20,
  cache_tokens: 0,
  total_tokens: 120,
  actual_cost: 0.1,
  account_cost: 0.08,
  cost: 0.2,
}]

describe('UserBreakdownSubTable', () => {
  it('renders actual cost as snow credits while preserving account and standard USD', () => {
    const wrapper = mount(UserBreakdownSubTable, {
      props: { items, creditMode: true },
    })

    expect(wrapper.get('[data-testid="credit-amount-value"]').text()).toBe('0.100')
    expect(wrapper.text()).toContain('$0.080')
    expect(wrapper.text()).toContain('$0.200')
  })

  it('preserves the default USD rendering for existing consumers', () => {
    const wrapper = mount(UserBreakdownSubTable, {
      props: { items },
    })

    expect(wrapper.find('[data-testid="credit-amount"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('$0.100')
  })
})
