import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import CatalogPriceAmount from '../CatalogPriceAmount.vue'
import CreditAmount from '../CreditAmount.vue'

describe('CatalogPriceAmount', () => {
  it('renders effective wallet prices directly in USD', () => {
    const wrapper = mount(CatalogPriceAmount, {
      props: {
        value: 0.000021,
        scale: 1_000_000,
        currency: ' credit ',
        creditDisplay: 'cny',
      },
    })

    expect(wrapper.text()).toBe('$21')
    expect(wrapper.findComponent(CreditAmount).exists()).toBe(true)
    expect(wrapper.find('[data-testid="credit-amount-symbol"]').exists()).toBe(true)
  })

  it('preserves zero and fractional yuan values without padding or rounding them away', () => {
    const zero = mount(CatalogPriceAmount, {
      props: { value: 0, currency: 'CREDIT', creditDisplay: 'cny' },
    })
    const fractional = mount(CatalogPriceAmount, {
      props: { value: 0.005, currency: 'CREDIT', creditDisplay: 'cny' },
    })

    expect(zero.text()).toBe('$0')
    expect(fractional.text()).toBe('$0.005')
  })

  it.each([
    { value: null, emptyText: undefined, expected: '-' },
    { value: undefined, emptyText: '暂未提供', expected: '暂未提供' },
    { value: Number.NaN, emptyText: undefined, expected: '-' },
    { value: Number.POSITIVE_INFINITY, emptyText: undefined, expected: '-' },
  ])('does not turn a missing or invalid CREDIT price into CNY zero', ({ value, emptyText, expected }) => {
    const wrapper = mount(CatalogPriceAmount, {
      props: {
        value,
        currency: 'CREDIT',
        creditDisplay: 'cny',
        ...(emptyText == null ? {} : { emptyText }),
      },
    })

    expect(wrapper.text()).toBe(expected)
    expect(wrapper.findComponent(CreditAmount).exists()).toBe(false)
  })

  it('keeps wallet prices as dollars by default for non-catalog consumers', () => {
    const wrapper = mount(CatalogPriceAmount, {
      props: { value: 350, currency: 'CREDIT' },
    })

    expect(wrapper.getComponent(CreditAmount).props('value')).toBe('350')
    expect(wrapper.find('[data-testid="credit-amount-symbol"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('$350')
  })

  it('keeps an explicitly monetary USD price in dollars even in CNY display mode', () => {
    const wrapper = mount(CatalogPriceAmount, {
      props: {
        value: 0.000005,
        scale: 1_000_000,
        currency: 'USD',
        creditDisplay: 'cny',
      },
    })

    expect(wrapper.text()).toBe('$5')
    expect(wrapper.findComponent(CreditAmount).exists()).toBe(true)
  })
})
