import { nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => ({
  setLocale: vi.fn(),
}))

vi.mock('vue-i18n', async () => {
  const { ref } = await vi.importActual<typeof import('vue')>('vue')

  return {
    useI18n: () => ({ locale: ref('en') }),
  }
})

vi.mock('@/i18n', () => ({
  setLocale: mocks.setLocale,
  availableLocales: [
    { code: 'en', name: 'English', flag: '🇺🇸' },
    { code: 'zh', name: '中文', flag: '🇨🇳' },
  ],
}))

import LocaleSwitcher from '../LocaleSwitcher.vue'

async function expectMenuClosed(wrapper: ReturnType<typeof mount>) {
  await vi.waitFor(() => {
    expect(wrapper.find('[role="menu"]').exists()).toBe(false)
  })
}

describe('LocaleSwitcher', () => {
  beforeEach(() => {
    mocks.setLocale.mockReset()
    mocks.setLocale.mockResolvedValue(undefined)
  })

  it('exposes menu-button semantics and 44px touch targets', async () => {
    const wrapper = mount(LocaleSwitcher, { attachTo: document.body })
    const trigger = wrapper.get('button')

    expect(trigger.attributes('aria-haspopup')).toBe('menu')
    expect(trigger.attributes('aria-expanded')).toBe('false')
    expect(trigger.attributes('aria-controls')).toBeTruthy()
    expect(trigger.classes()).toEqual(expect.arrayContaining(['min-h-11', 'min-w-11']))
    expect(trigger.find('.locale-switcher-icon').exists()).toBe(true)
    expect(trigger.text()).not.toContain('EN')

    await trigger.trigger('click')
    await nextTick()

    const menu = wrapper.get('[role="menu"]')
    expect(trigger.attributes('aria-expanded')).toBe('true')
    expect(menu.attributes('id')).toBe(trigger.attributes('aria-controls'))
    expect(menu.attributes('aria-labelledby')).toBe(trigger.attributes('id'))

    const items = wrapper.findAll('[role="menuitemradio"]')
    expect(items).toHaveLength(2)
    expect(items[0].attributes('aria-checked')).toBe('true')
    expect(items[1].attributes('aria-checked')).toBe('false')
    expect(items[1].text()).toContain('中文')
    expect(items.every((item) => item.classes().includes('min-h-11'))).toBe(true)
    expect(document.activeElement).toBe(items[0].element)

    wrapper.unmount()
  })

  it('opens with ArrowUp or ArrowDown and cycles focus through the menu', async () => {
    const wrapper = mount(LocaleSwitcher, { attachTo: document.body })
    const trigger = wrapper.get('button')

    await trigger.trigger('keydown', { key: 'ArrowDown' })
    await nextTick()

    const items = wrapper.findAll('[role="menuitemradio"]')
    expect(document.activeElement).toBe(items[0].element)

    await items[0].trigger('keydown', { key: 'ArrowDown' })
    expect(document.activeElement).toBe(items[1].element)

    await items[1].trigger('keydown', { key: 'ArrowDown' })
    expect(document.activeElement).toBe(items[0].element)

    await items[0].trigger('keydown', { key: 'ArrowUp' })
    expect(document.activeElement).toBe(items[1].element)

    await items[1].trigger('keydown', { key: 'Escape' })
    await expectMenuClosed(wrapper)

    await trigger.trigger('keydown', { key: 'ArrowUp' })
    await nextTick()

    const reopenedItems = wrapper.findAll('[role="menuitemradio"]')
    expect(document.activeElement).toBe(reopenedItems[reopenedItems.length - 1].element)

    wrapper.unmount()
  })

  it('closes on Escape and restores focus to the trigger', async () => {
    const wrapper = mount(LocaleSwitcher, { attachTo: document.body })
    const trigger = wrapper.get('button')

    await trigger.trigger('click')
    await nextTick()
    await wrapper.get('[role="menuitemradio"]').trigger('keydown', { key: 'Escape' })
    await expectMenuClosed(wrapper)

    expect(trigger.attributes('aria-expanded')).toBe('false')
    expect(document.activeElement).toBe(trigger.element)

    wrapper.unmount()
  })

  it('preserves click-outside closing without stealing focus', async () => {
    const wrapper = mount(LocaleSwitcher, { attachTo: document.body })
    const trigger = wrapper.get('button')
    const outsideButton = document.createElement('button')
    document.body.appendChild(outsideButton)

    await trigger.trigger('click')
    await nextTick()
    outsideButton.focus()
    outsideButton.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await expectMenuClosed(wrapper)

    expect(document.activeElement).toBe(outsideButton)

    outsideButton.remove()
    wrapper.unmount()
  })

  it('switches locale, closes the menu, and restores trigger focus', async () => {
    const wrapper = mount(LocaleSwitcher, { attachTo: document.body })
    const trigger = wrapper.get('button')

    await trigger.trigger('click')
    await nextTick()
    await wrapper.findAll('[role="menuitemradio"]')[1].trigger('click')
    await nextTick()

    expect(mocks.setLocale).toHaveBeenCalledWith('zh')
    await expectMenuClosed(wrapper)
    expect(document.activeElement).toBe(trigger.element)

    wrapper.unmount()
  })
})
