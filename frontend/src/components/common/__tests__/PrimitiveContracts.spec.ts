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

  it('exposes localized select labeling and listbox ownership', async () => {
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
    expect(trigger.attributes('aria-haspopup')).toBe('listbox')
    expect(trigger.attributes('aria-expanded')).toBe('false')
    expect(trigger.attributes('aria-controls')).toBeUndefined()
    expect(trigger.attributes('aria-activedescendant')).toBeUndefined()

    await trigger.trigger('click')
    await wrapper.vm.$nextTick()

    const listbox = wrapper.get('[role="listbox"]')
    expect(trigger.attributes('aria-expanded')).toBe('true')
    expect(trigger.attributes('aria-controls')).toBe(listbox.attributes('id'))
    expect(listbox.attributes('aria-label')).toBe('选择平台')
    expect(listbox.findAll(':scope > [role="option"]')).toHaveLength(1)
    expect(
      Array.from(listbox.element.children).every((child) => child.getAttribute('role') === 'option')
    ).toBe(true)
  })

  it('keeps a searchable select input named and outside the listbox role', async () => {
    const wrapper = mount(Select, {
      props: {
        modelValue: null,
        options: [
          { value: 'gpt', label: 'GPT' },
          { value: 'gemini', label: 'Gemini' }
        ],
        searchable: true,
        searchPlaceholder: '搜索平台'
      },
      attachTo: document.body,
      global: {
        stubs: { Teleport: true }
      }
    })

    const trigger = wrapper.get('.select-trigger')
    await trigger.trigger('click')
    await wrapper.vm.$nextTick()

    const searchInput = wrapper.get('.select-search-input')
    const listbox = wrapper.get('[role="listbox"]')
    expect(searchInput.attributes('aria-label')).toBe('搜索平台')
    expect(searchInput.attributes('aria-controls')).toBe(listbox.attributes('id'))
    expect(searchInput.attributes('aria-activedescendant')).toMatch(/-option-0$/)
    expect(trigger.attributes('aria-activedescendant')).toBeUndefined()
    expect(listbox.element.contains(searchInput.element)).toBe(false)
    expect(document.activeElement).toBe(searchInput.element)

    await searchInput.setValue('不存在的平台')
    expect(searchInput.attributes('aria-label')).toBe('搜索平台')
    expect(searchInput.attributes('aria-activedescendant')).toBeUndefined()
    expect(wrapper.get('[role="status"]').element.parentElement).toBe(listbox.element.parentElement)

    wrapper.unmount()
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
      attachTo: document.body,
      global: {
        stubs: { Teleport: true }
      }
    })

    const trigger = wrapper.get('.select-trigger')
    await trigger.trigger('keydown', { key: 'ArrowDown' })
    await wrapper.vm.$nextTick()
    expect(trigger.attributes('aria-expanded')).toBe('true')
    expect(trigger.attributes('aria-activedescendant')).toBeUndefined()

    const listbox = wrapper.get('[role="listbox"]')
    expect(listbox.attributes('aria-activedescendant')).toMatch(/-option-0$/)
    expect(listbox.attributes('tabindex')).toBe('-1')
    expect(document.activeElement).toBe(listbox.element)

    await listbox.trigger('keydown', { key: 'Enter' })
    expect(wrapper.emitted('update:modelValue')).toEqual([['gpt']])
    expect(trigger.attributes('aria-expanded')).toBe('false')
    expect(trigger.attributes('aria-controls')).toBeUndefined()

    wrapper.unmount()
  })

  it.each([
    ['non-searchable', false],
    ['searchable', true]
  ])('restores the trigger as the native Tab-order anchor for a %s select', async (_label, searchable) => {
    const wrapper = mount(Select, {
      props: {
        modelValue: null,
        options: [
          { value: 'gpt', label: 'GPT' },
          { value: 'gemini', label: 'Gemini' }
        ],
        searchable
      },
      attachTo: document.body
    })

    const trigger = wrapper.get('.select-trigger')

    for (const shiftKey of [false, true]) {
      await trigger.trigger('click')
      await wrapper.vm.$nextTick()

      const listboxId = trigger.attributes('aria-controls')
      expect(listboxId).toBeTruthy()
      const listbox = listboxId ? document.getElementById(listboxId) : null
      const focusTarget = searchable
        ? listbox?.parentElement?.querySelector<HTMLInputElement>('.select-search-input')
        : listbox

      expect(focusTarget).not.toBeNull()
      expect(document.activeElement).toBe(focusTarget)

      const tabEvent = new KeyboardEvent('keydown', {
        key: 'Tab',
        shiftKey,
        bubbles: true,
        cancelable: true
      })
      focusTarget?.dispatchEvent(tabEvent)

      expect(tabEvent.defaultPrevented).toBe(false)
      expect(document.activeElement).toBe(trigger.element)

      await wrapper.vm.$nextTick()
      expect(trigger.attributes('aria-expanded')).toBe('false')
      expect(trigger.attributes('aria-controls')).toBeUndefined()
    }

    wrapper.unmount()
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
