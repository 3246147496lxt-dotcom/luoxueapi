import { describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import AmountInput from '../AmountInput.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}))

describe('AmountInput', () => {
  it('keeps the preset grid visible directly above the custom amount input', () => {
    const wrapper = mount(AmountInput, {
      props: {
        modelValue: null,
        amounts: [10, 20, 50, 100],
        currency: 'USD',
        locale: 'en-US',
      },
    })

    const grid = wrapper.get('[data-testid="preset-amount-grid"]')
    const input = wrapper.get('#custom-recharge-amount')
    const choices = wrapper.findAll('[role="radio"]')

    expect(wrapper.text()).toContain('payment.chooseAmountTitle')
    expect(wrapper.text()).toContain('payment.customAmount')
    expect(wrapper.find('[role="tablist"]').exists()).toBe(false)
    expect(wrapper.find('[role="tab"]').exists()).toBe(false)
    expect(wrapper.find('[role="tabpanel"]').exists()).toBe(false)
    expect(choices).toHaveLength(4)
    expect(choices[0].text()).toContain('$10')
    expect(grid.classes()).toEqual(expect.arrayContaining([
      'grid',
      'grid-cols-2',
      'gap-3',
      'sm:grid-cols-4',
    ]))
    expect(grid.element.compareDocumentPosition(input.element) & Node.DOCUMENT_POSITION_FOLLOWING)
      .toBeTruthy()
  })

  it('selects a preset and keeps the custom input empty', async () => {
    const wrapper = mount(AmountInput, {
      props: {
        modelValue: 20,
        amounts: [10, 20, 50],
        currency: 'CNY',
        locale: 'zh-CN',
      },
    })

    const choices = wrapper.findAll('[role="radio"]')
    expect(choices.map(choice => choice.attributes('aria-checked'))).toEqual(['false', 'true', 'false'])
    expect(choices.map(choice => choice.attributes('tabindex'))).toEqual(['-1', '0', '-1'])
    expect(wrapper.get('input').element.value).toBe('')

    await choices[2].trigger('click')
    await wrapper.setProps({ modelValue: 50 })

    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([50])
    expect(wrapper.get('input').element.value).toBe('')
    expect(choices[2].attributes('aria-checked')).toBe('true')
  })

  it('renders a non-preset model value in the custom input without selecting a preset', () => {
    const wrapper = mount(AmountInput, {
      props: {
        modelValue: 35,
        amounts: [10, 20, 50],
        currency: 'USD',
        locale: 'en-US',
      },
    })

    expect(wrapper.get('input').element.value).toBe('35')
    expect(wrapper.findAll('[role="radio"]')
      .every(choice => choice.attributes('aria-checked') === 'false')).toBe(true)
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

  it('clears a custom value when a preset is clicked and keeps focus on that radio', async () => {
    const wrapper = mount(AmountInput, {
      attachTo: document.body,
      props: {
        modelValue: null,
        amounts: [10, 20],
        currency: 'CNY',
        locale: 'zh-CN',
      },
    })

    const input = wrapper.get('input')
    await input.setValue('35')
    await wrapper.setProps({ modelValue: 35 })
    expect(wrapper.findAll('[role="radio"]')
      .every(choice => choice.attributes('aria-checked') === 'false')).toBe(true)

    const choices = wrapper.findAll('[role="radio"]')
    ;(choices[1].element as HTMLElement).focus()
    await choices[1].trigger('click')
    await wrapper.setProps({ modelValue: 20 })

    expect(wrapper.emitted('update:modelValue')).toEqual([[35], [20]])
    expect(input.element.value).toBe('')
    expect(choices[1].attributes('aria-checked')).toBe('true')
    expect(choices.map(choice => choice.attributes('tabindex'))).toEqual(['-1', '0'])
    expect(document.activeElement).toBe(choices[1].element)
    wrapper.unmount()
  })

  it('emits custom decimal values and null when the input is cleared', async () => {
    const wrapper = mount(AmountInput, {
      props: {
        modelValue: 20,
        amounts: [10, 20, 50],
        currency: 'CNY',
        locale: 'zh-CN',
      },
    })

    const input = wrapper.get('input')
    await input.setValue('35.5')
    await wrapper.setProps({ modelValue: 35.5 })

    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([35.5])
    expect(input.element.value).toBe('35.5')
    expect(wrapper.findAll('[role="radio"]')
      .every(choice => choice.attributes('aria-checked') === 'false')).toBe(true)

    await input.setValue('')
    await wrapper.setProps({ modelValue: null })
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([null])
    expect(input.element.value).toBe('')
  })

  it('supports roving radio focus, arrow wrapping, and Home/End for presets', async () => {
    const wrapper = mount(AmountInput, {
      attachTo: document.body,
      props: {
        modelValue: 20,
        amounts: [10, 20, 50],
        currency: 'CNY',
        locale: 'zh-CN',
      },
    })

    const choices = wrapper.findAll('[role="radio"]')
    expect(choices.map(choice => choice.attributes('tabindex'))).toEqual(['-1', '0', '-1'])

    const move = async (fromIndex: number, key: string, amount: number, toIndex: number) => {
      await choices[fromIndex].trigger('keydown', { key })
      await nextTick()
      await wrapper.setProps({ modelValue: amount })

      expect(choices[toIndex].attributes('aria-checked')).toBe('true')
      expect(choices.map(choice => choice.attributes('tabindex'))).toEqual(
        choices.map((_, index) => index === toIndex ? '0' : '-1'),
      )
      expect(wrapper.get('input').element.value).toBe('')
      expect(document.activeElement).toBe(choices[toIndex].element)
    }

    ;(choices[1].element as HTMLElement).focus()
    await move(1, 'ArrowRight', 50, 2)
    await move(2, 'ArrowRight', 10, 0)
    await move(0, 'ArrowLeft', 50, 2)
    await move(2, 'Home', 10, 0)
    await move(0, 'End', 50, 2)
    await move(2, 'ArrowDown', 10, 0)
    await move(0, 'ArrowUp', 50, 2)

    expect(wrapper.emitted('update:modelValue')).toEqual([
      [50],
      [10],
      [50],
      [10],
      [50],
      [10],
      [50],
    ])
    wrapper.unmount()
  })

  it('keeps the custom input usable when all presets are outside the allowed range', async () => {
    const wrapper = mount(AmountInput, {
      props: {
        modelValue: null,
        amounts: [10, 20],
        min: 100,
        currency: 'CNY',
        locale: 'zh-CN',
      },
    })

    expect(wrapper.find('[data-testid="preset-amount-grid"]').exists()).toBe(true)
    expect(wrapper.findAll('[role="radio"]')).toHaveLength(0)
    const input = wrapper.get('input')
    await input.setValue('150')
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([150])
  })
})
