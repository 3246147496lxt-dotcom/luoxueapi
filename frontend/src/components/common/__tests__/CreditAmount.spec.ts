import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import CreditAmount from '../CreditAmount.vue'

describe('CreditAmount', () => {
  it('pairs a bare numeric value with the dollar symbol', () => {
    const wrapper = mount(CreditAmount, {
      props: {
        value: '25.34',
        iconSize: 'md',
        label: '历史消耗 25.34',
      },
    })

    expect(wrapper.get('[data-testid="credit-amount-value"]').text()).toBe('25.34')
    expect(wrapper.text()).toContain('$25.34')
    expect(wrapper.attributes('role')).toBe('group')
    expect(wrapper.attributes('aria-label')).toBe('历史消耗 25.34')
    expect(wrapper.get('[data-testid="credit-amount-value"]').attributes('aria-hidden')).toBe('true')
    expect(wrapper.get('[data-testid="credit-amount-symbol"]').text()).toBe('$')
  })

  it('provides a dollar unit when callers omit a custom label', () => {
    const wrapper = mount(CreditAmount, {
      props: { value: '12.50' },
    })

    expect(wrapper.attributes('aria-label')).toBe('$12.50')
  })
})
