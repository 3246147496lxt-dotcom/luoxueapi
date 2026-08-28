import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import CatalogPriceAmount from '../CatalogPriceAmount.vue'
import CreditAmount from '../CreditAmount.vue'

describe('CatalogPriceAmount', () => {
  it('renders effective CREDIT prices with the shared points amount semantics', () => {
    const wrapper = mount(CatalogPriceAmount, {
      props: {
        value: 0.00035,
        scale: 1_000_000,
        currency: 'credit',
      },
    })

    expect(wrapper.getComponent(CreditAmount).props('value')).toBe('350')
    expect(wrapper.find('[data-testid="points-icon"]').exists()).toBe(true)
    expect(wrapper.text()).not.toMatch(/[$¥]/)
  })

  it('keeps an explicitly monetary USD price in dollars', () => {
    const wrapper = mount(CatalogPriceAmount, {
      props: {
        value: 0.000005,
        scale: 1_000_000,
        currency: 'USD',
      },
    })

    expect(wrapper.text()).toBe('$5')
    expect(wrapper.findComponent(CreditAmount).exists()).toBe(false)
  })
})
