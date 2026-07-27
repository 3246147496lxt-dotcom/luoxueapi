import { describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import PaymentMethodSelector from '@/components/payment/PaymentMethodSelector.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, fallback?: string) => fallback ?? key,
  }),
}))

describe('PaymentMethodSelector', () => {
  it('shows the configured display name for custom EasyPay methods', () => {
    const wrapper = mount(PaymentMethodSelector, {
      props: {
        selected: 'ldc',
        methods: [{ type: 'ldc', display_name: 'LDC Pay', fee_rate: 0, available: true }],
      },
    })

    expect(wrapper.text()).toContain('LDC Pay')
    expect(wrapper.text()).not.toContain('ldc')
    expect(wrapper.text()).not.toContain('payment.methods.ldc')
    expect(wrapper.get('button').classes()).toEqual(expect.arrayContaining(['p-4', 'rounded-xl']))
    expect(wrapper.get('button').classes()).not.toContain('rounded-2xl')
    expect(wrapper.get('img').classes()).toEqual(expect.arrayContaining(['h-6', 'w-6']))
    expect(wrapper.get('[role="radiogroup"]').classes()).toEqual(expect.arrayContaining([
      'grid-cols-1',
      'gap-4',
      'sm:grid-cols-2',
    ]))
  })

  it('uses one violet selected state instead of provider-colored selection logic', () => {
    const wrapper = mount(PaymentMethodSelector, {
      props: {
        selected: 'alipay',
        methods: [{ type: 'alipay', display_name: 'Alipay', fee_rate: 0, available: true }],
      },
    })

    const button = wrapper.get('button')
    expect(button.classes()).toEqual(expect.arrayContaining(['payment-method-option', 'is-selected']))
    expect(wrapper.find('[data-testid="payment-method-selected"]').exists()).toBe(true)
    expect(wrapper.html()).not.toContain('primary-')
    expect(wrapper.html()).not.toContain('border-[#02A9F1]')
  })

  it('describes disabled methods only with the available data it receives', () => {
    const wrapper = mount(PaymentMethodSelector, {
      props: {
        selected: 'alipay',
        methods: [
          { type: 'stripe', display_name: 'Stripe', fee_rate: 2.5, available: false },
        ],
      },
    })

    const button = wrapper.get('button')
    const status = wrapper.get('[data-testid="payment-method-unavailable"]')
    expect(button.attributes('disabled')).toBeDefined()
    expect(button.attributes('aria-disabled')).toBe('true')
    expect(button.attributes('aria-describedby')).toBe(status.attributes('id'))
    expect(status.text()).toBe('payment.amountUnavailable')
    expect(wrapper.text()).not.toContain('payment.amountNoMethod')
    expect(wrapper.text()).not.toContain('2.5%')
  })

  it('emits selection only for available methods', async () => {
    const wrapper = mount(PaymentMethodSelector, {
      attachTo: document.body,
      props: {
        selected: '',
        methods: [
          { type: 'alipay', display_name: 'Alipay', fee_rate: 0, available: true },
          { type: 'stripe', display_name: 'Stripe', fee_rate: 0, available: false },
        ],
      },
    })

    const buttons = wrapper.findAll('button')
    expect(buttons.map(button => button.attributes('tabindex'))).toEqual(['0', '-1'])

    buttons[0].element.focus()
    await buttons[0].trigger('click')
    await wrapper.setProps({ selected: 'alipay' })
    await buttons[1].trigger('click')
    expect(wrapper.emitted('select')).toEqual([['alipay']])
    expect(buttons.map(button => button.attributes('tabindex'))).toEqual(['0', '-1'])
    expect(document.activeElement).toBe(buttons[0].element)
    wrapper.unmount()
  })

  it('uses roving focus, skips disabled methods, and wraps with arrow keys and Home/End', async () => {
    const wrapper = mount(PaymentMethodSelector, {
      attachTo: document.body,
      props: {
        selected: 'alipay',
        methods: [
          { type: 'alipay', display_name: 'Alipay', fee_rate: 0, available: true },
          { type: 'wxpay', display_name: 'WeChat Pay', fee_rate: 0, available: false },
          { type: 'stripe', display_name: 'Stripe', fee_rate: 0, available: true },
          { type: 'airwallex', display_name: 'Airwallex', fee_rate: 0, available: true },
        ],
      },
    })

    const buttons = wrapper.findAll<HTMLButtonElement>('[role="radio"]')
    const buttonByType = (type: string) => wrapper.get<HTMLButtonElement>(`[data-method-type="${type}"]`)
    const enabledTypes = ['alipay', 'stripe', 'airwallex']

    expect(buttons.map(button => button.attributes('tabindex'))).toEqual(['0', '-1', '-1', '-1'])
    expect(buttonByType('wxpay').attributes('disabled')).toBeDefined()

    const move = async (fromType: string, key: string, toType: string) => {
      await buttonByType(fromType).trigger('keydown', { key })
      await nextTick()
      await wrapper.setProps({ selected: toType })

      expect(buttonByType(toType).attributes('aria-checked')).toBe('true')
      for (const type of enabledTypes) {
        expect(buttonByType(type).attributes('tabindex')).toBe(type === toType ? '0' : '-1')
      }
      expect(buttonByType('wxpay').attributes('tabindex')).toBe('-1')
      expect(document.activeElement).toBe(buttonByType(toType).element)
    }

    buttonByType('alipay').element.focus()
    await move('alipay', 'ArrowRight', 'stripe')
    await move('stripe', 'ArrowDown', 'airwallex')
    await move('airwallex', 'ArrowRight', 'alipay')
    await move('alipay', 'ArrowLeft', 'airwallex')
    await move('airwallex', 'Home', 'alipay')
    await move('alipay', 'End', 'airwallex')
    await move('airwallex', 'ArrowUp', 'stripe')

    expect(wrapper.emitted('select')).toEqual([
      ['stripe'],
      ['airwallex'],
      ['alipay'],
      ['airwallex'],
      ['alipay'],
      ['airwallex'],
      ['stripe'],
    ])
    wrapper.unmount()
  })
})
