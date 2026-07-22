import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import Input from '../Input.vue'
import Select from '../Select.vue'
import TextArea from '../TextArea.vue'
import Toggle from '../Toggle.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

describe('Snow Clay primitive contracts', () => {
  it('associates an input error with the field and exposes invalid state', () => {
    const wrapper = mount(Input, {
      props: {
        id: 'account-name',
        modelValue: '',
        label: '账号名称',
        error: '请输入账号名称',
        required: true
      },
      attrs: {
        name: 'accountName',
        inputmode: 'text'
      }
    })

    const input = wrapper.get('input')
    expect(input.attributes('aria-invalid')).toBe('true')
    expect(input.attributes('aria-describedby')).toBe('account-name-message')
    expect(wrapper.get('[role="alert"]').attributes('id')).toBe('account-name-message')
    expect(wrapper.get('label').attributes('for')).toBe('account-name')
    expect(input.attributes('name')).toBe('accountName')
    expect(input.attributes('inputmode')).toBe('text')
    expect(wrapper.attributes('name')).toBeUndefined()
  })

  it('associates textarea hint copy without marking the field invalid', () => {
    const wrapper = mount(TextArea, {
      props: {
        id: 'announcement-copy',
        modelValue: '',
        hint: '支持多行内容'
      }
    })

    const textarea = wrapper.get('textarea')
    expect(textarea.attributes('aria-invalid')).toBeUndefined()
    expect(textarea.attributes('aria-describedby')).toBe('announcement-copy-message')
    expect(wrapper.get('#announcement-copy-message').text()).toBe('支持多行内容')
  })

  it('exposes localized select labeling and listbox ownership', () => {
    const wrapper = mount(Select, {
      props: {
        modelValue: null,
        options: [{ value: 'gpt', label: 'GPT' }],
        placeholder: '选择平台'
      },
      global: {
        stubs: { Teleport: true }
      }
    })

    const trigger = wrapper.get('.select-trigger')
    expect(trigger.attributes('aria-label')).toBe('选择平台')
    expect(trigger.attributes('aria-controls')).toMatch(/^select-.+-listbox$/)
  })

  it('supports keyboard selection when the select has no search field', async () => {
    const wrapper = mount(Select, {
      props: {
        modelValue: null,
        options: [
          { value: 'gpt', label: 'GPT' },
          { value: 'gemini', label: 'Gemini' }
        ],
        searchable: false
      },
      global: {
        stubs: { Teleport: true }
      }
    })

    const trigger = wrapper.get('.select-trigger')
    await trigger.trigger('keydown', { key: 'ArrowDown' })
    await wrapper.vm.$nextTick()
    expect(trigger.attributes('aria-expanded')).toBe('true')
    expect(trigger.attributes('aria-activedescendant')).toMatch(/-option-0$/)

    await trigger.trigger('keydown', { key: 'Enter' })
    expect(wrapper.emitted('update:modelValue')).toEqual([['gpt']])
  })

  it('uses semantic toggle classes and blocks disabled changes', async () => {
    const wrapper = mount(Toggle, {
      props: {
        modelValue: true,
        disabled: true,
        ariaLabel: '启用渠道'
      }
    })

    const button = wrapper.get('button')
    expect(button.classes()).toContain('toggle-control--active')
    expect(button.attributes('aria-checked')).toBe('true')
    expect(button.attributes('aria-disabled')).toBe('true')
    expect(button.attributes('aria-label')).toBe('启用渠道')

    await button.trigger('click')
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
  })
})
