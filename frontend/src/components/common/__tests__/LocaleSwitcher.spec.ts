import { nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => ({
  resolvedLocale: { value: 'en' as 'en' | 'zh' },
  setPreference: vi.fn(),
}))

vi.mock('@/composables/useLocalePreference', () => ({
  useLocalePreference: () => ({
    resolvedLocale: mocks.resolvedLocale,
    setPreference: mocks.setPreference,
  }),
}))

vi.mock('@/i18n', () => ({
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
    mocks.resolvedLocale.value = 'en'
    mocks.setPreference.mockReset()
    mocks.setPreference.mockResolvedValue(undefined)
  })

  it('exposes menu-button semantics and 44px touch targets', async () => {
    const wrapper = mount(LocaleSwitcher, { attachTo: document.body })
    const trigger = wrapper.get('button')

    expect(trigger.attributes('aria-haspopup')).toBe('menu')
    expect(trigger.attributes('aria-expanded')).toBe('false')
    expect(trigger.attributes('aria-controls')).toBeTruthy()
    expect(trigger.classes()).toContain('locale-switcher-trigger')
    expect(trigger.classes()).toEqual(expect.arrayContaining(['min-h-11', 'min-w-11']))
    expect(trigger.classes()).not.toEqual(expect.arrayContaining(['bg-gray-100', 'dark:bg-dark-800']))
    const languageIcon = trigger.get('svg.locale-switcher-icon')
    expect(languageIcon.attributes('viewBox')).toBe('0 0 1024 1024')
    expect(languageIcon.attributes('aria-hidden')).toBe('true')
    expect(languageIcon.findAll('path')).toHaveLength(3)
    expect(languageIcon.findAll('path').every(path => path.attributes('fill') === 'currentColor')).toBe(true)
    expect(languageIcon.find('script, foreignObject, [href], [xlink\\:href]').exists()).toBe(false)
    expect(trigger.text()).not.toContain('EN')

    await trigger.trigger('click')
    await nextTick()

    const menu = wrapper.get('[role="menu"]')
    expect(trigger.attributes('aria-expanded')).toBe('true')
    expect(trigger.classes()).toEqual(expect.arrayContaining(['bg-gray-100', 'dark:bg-dark-800']))
    expect(menu.attributes('id')).toBe(trigger.attributes('aria-controls'))
    expect(menu.attributes('aria-labelledby')).toBe(trigger.attributes('id'))

    const items = wrapper.findAll('[role="menuitemradio"]')
    expect(items).toHaveLength(2)
    expect(items[0].attributes('aria-checked')).toBe('true')
    expect(items[1].attributes('aria-checked')).toBe('false')
    expect(menu.text()).not.toContain('🇺🇸')
    expect(menu.text()).not.toContain('🇨🇳')
    expect(items[0].text()).toContain('English')
    expect(items[1].text()).toContain('中文')
    expect(items.every((item) => item.classes().includes('min-h-11'))).toBe(true)
    expect(items.every((item) => !item.classes().includes('locale-switcher-trigger'))).toBe(true)
    expect(menu.classes()).toEqual(expect.arrayContaining(['grid', 'gap-0.5']))
    expect(items.every((item) => item.classes().includes('grid-cols-[minmax(0,1fr)_1.25rem]')))
      .toBe(true)
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

    expect(mocks.setPreference).toHaveBeenCalledWith('zh')
    await expectMenuClosed(wrapper)
    expect(document.activeElement).toBe(trigger.element)

    wrapper.unmount()
  })

  it('matches the compact header action proportions and blue icon treatment', () => {
    const wrapper = mount(LocaleSwitcher, {
      props: { compact: true },
      attachTo: document.body,
    })
    const trigger = wrapper.get('button')
    const icon = trigger.get('svg.locale-switcher-icon')

    expect(trigger.classes()).toEqual(expect.arrayContaining([
      'h-8',
      'w-8',
      'text-[#007bff]',
      'hover:bg-[rgba(46,50,56,0.05)]',
    ]))
    expect(trigger.classes()).not.toContain('min-h-11')
    expect(icon.classes()).toEqual(expect.arrayContaining(['h-5', 'w-5']))

    wrapper.unmount()
  })

  it('renders the original Lucide languages outline for the clay public header', () => {
    const wrapper = mount(LocaleSwitcher, {
      props: { iconVariant: 'lucide' },
      attachTo: document.body,
    })
    const icon = wrapper.get('button svg.locale-switcher-icon')

    expect(icon.attributes()).toMatchObject({
      viewBox: '0 0 24 24',
      fill: 'none',
      stroke: 'currentColor',
      'stroke-width': '2',
      'aria-hidden': 'true',
    })
    expect(icon.get('path').attributes('d'))
      .toBe('M5 8l6 6M4 14l6-6 2-3M2 5h12M7 2h1M22 22l-5-10-5 10M14 18h6')

    wrapper.unmount()
  })
})
