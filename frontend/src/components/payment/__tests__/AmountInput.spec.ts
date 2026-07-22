import { describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import AmountInput from '../AmountInput.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}))

describe('AmountInput', () => {
  it('uses selectable amount cards and emits the selected preset', async () => {
    const wrapper = mount(AmountInput, {
      props: {
        modelValue: null,
        amounts: [10, 20, 50, 100],
        currency: 'USD',
        locale: 'en-US',
      },
    })

    const choices = wrapper.findAll('[role="radio"]')
    expect(choices).toHaveLength(4)
    expect(wrapper.get('#amount-panel-preset').text()).toContain('payment.chooseAmountTitle')
    expect(choices[0].text()).toContain('$10')
    expect(wrapper.get('[data-testid="preset-amount-grid"]').classes()).toEqual(expect.arrayContaining([
      'grid',
      'grid-cols-2',
      'sm:grid-cols-4',
    ]))
    expect(choices[0].classes()).toEqual(expect.arrayContaining(['min-w-0']))
    expect(choices[0].classes()).not.toEqual(expect.arrayContaining(['min-w-28', 'snap-start']))
    await choices[1].trigger('click')
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([20])
  })

  it('renders the amount modes as compact flat controls', () => {
    const wrapper = mount(AmountInput, {
      props: { modelValue: null, currency: 'CNY', locale: 'zh-CN' },
    })

    expect(wrapper.text()).toContain('payment.amountType')
    const tabs = wrapper.findAll('[role="tab"]')
    expect(tabs).toHaveLength(2)
    expect(tabs[0].classes()).toEqual(expect.arrayContaining(['min-h-9', 'bg-primary-50']))
    expect(tabs[0].classes()).not.toContain('shadow-sm')
    expect(tabs[1].classes()).toContain('bg-gray-100')
  })

  it('switches to a focused custom input without showing both modes at once', async () => {
    const wrapper = mount(AmountInput, {
      attachTo: document.body,
      props: { modelValue: null, currency: 'CNY', locale: 'zh-CN' },
    })

    expect(wrapper.find('input').exists()).toBe(false)
    await wrapper.findAll('[role="tab"]')[1].trigger('click')
    await nextTick()

    const input = wrapper.get('input')
    expect(input.exists()).toBe(true)
    expect(document.activeElement).toBe(input.element)
    wrapper.unmount()
  })

  it('opens custom mode automatically when the current value is not a preset', () => {
    const wrapper = mount(AmountInput, {
      props: {
        modelValue: 35,
        amounts: [10, 20, 50],
        currency: 'USD',
        locale: 'en-US',
      },
    })

    expect(wrapper.get('input').element.value).toBe('35')
    expect(wrapper.findAll('[role="tab"]')[1].attributes('aria-selected')).toBe('true')
  })

  it('keeps long preset values wrap-safe inside the two-column mobile grid', () => {
    const wrapper = mount(AmountInput, {
      props: {
        modelValue: null,
        amounts: [10, 123456789],
        currency: 'CNY',
        locale: 'zh-CN',
      },
    })

    const values = wrapper.findAll('[data-testid="preset-amount-value"]')
    expect(values).toHaveLength(2)
    expect(values[1].classes()).toEqual(expect.arrayContaining([
      'max-w-full',
      '[overflow-wrap:anywhere]',
    ]))
  })
})
