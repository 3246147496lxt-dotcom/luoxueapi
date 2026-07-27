import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import StripePopupView from '../StripePopupView.vue'

vi.mock('vue-router', () => ({
  useRoute: () => ({
    query: {
      order_id: '17',
      method: 'wechat_pay',
      amount: '80',
      currency: 'HKD',
    },
  }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

describe('StripePopupView payment currency', () => {
  it('formats the amount using the currency carried by the parent payment view', () => {
    const wrapper = mount(StripePopupView)

    expect(wrapper.text()).toMatch(/HK\$80\.00|HKD\s*80\.00/)
    expect(wrapper.text()).not.toContain('¥80.00')
    wrapper.unmount()
  })
})
