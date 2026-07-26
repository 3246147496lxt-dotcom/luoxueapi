import { DOMWrapper, mount, type VueWrapper } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  isTopModalLayer,
  registerModalLayer,
  unregisterModalLayer,
} from '@/utils/modalStack'
import SettingsChoiceMenu, {
  type SettingsChoiceOption,
} from '../SettingsChoiceMenu.vue'

const wrappers = new Set<VueWrapper>()
const layerTokens = new Set<symbol>()

const options: SettingsChoiceOption[] = [
  { value: 'system', label: 'System' },
  { value: 'light', label: 'Light' },
  { value: 'dark', label: 'Dark' },
]

function rect(overrides: Partial<DOMRect> = {}): DOMRect {
  return {
    bottom: 136,
    height: 36,
    left: 584,
    right: 704,
    top: 100,
    width: 120,
    x: 584,
    y: 100,
    toJSON: () => ({}),
    ...overrides,
  }
}

function menuElement() {
  const element = document.body.querySelector<HTMLElement>(
    '[data-ui-portal="settings-choice-menu"]',
  )
  if (!element) throw new Error('Settings choice menu is not open')
  return new DOMWrapper(element)
}

function optionElements() {
  return Array.from(document.body.querySelectorAll<HTMLButtonElement>(
    '[role="menuitemradio"]',
  )).map((element) => new DOMWrapper(element))
}

async function waitForMenuToClose() {
  await vi.waitFor(() => {
    expect(document.body.querySelector('[role="menu"]')).toBeNull()
  })
}

function mountMenu(
  props: Partial<{
    modelValue: string
    options: readonly SettingsChoiceOption[]
    ariaLabel: string
    disabled: boolean
    testId: string
  }> = {},
) {
  const wrapper = mount(SettingsChoiceMenu, {
    attachTo: document.body,
    props: {
      modelValue: 'system',
      options,
      ariaLabel: 'Appearance',
      testId: 'choice-trigger',
      ...props,
    },
    global: {
      stubs: {
        Icon: {
          props: ['name'],
          template: '<span :data-icon="name" />',
        },
      },
    },
  })
  wrappers.add(wrapper)
  const trigger = wrapper.get<HTMLButtonElement>('[data-testid="choice-trigger"]')
  trigger.element.getBoundingClientRect = vi.fn(() => rect())
  return { trigger, wrapper }
}

beforeEach(() => {
  Object.defineProperty(window, 'innerWidth', {
    configurable: true,
    value: 800,
  })
  Object.defineProperty(window, 'innerHeight', {
    configurable: true,
    value: 600,
  })
  Object.defineProperty(HTMLElement.prototype, 'scrollIntoView', {
    configurable: true,
    value: vi.fn(),
  })
})

afterEach(() => {
  for (const wrapper of wrappers) wrapper.unmount()
  wrappers.clear()
  for (const token of layerTokens) unregisterModalLayer(token)
  layerTokens.clear()
  document.body.replaceChildren()
  vi.restoreAllMocks()
})

describe('SettingsChoiceMenu', () => {
  it('teleports a fixed 220px menu and exposes single-choice semantics', async () => {
    const { trigger, wrapper } = mountMenu()

    expect(trigger.attributes()).toMatchObject({
      'aria-haspopup': 'menu',
      'aria-expanded': 'false',
      'aria-label': 'Appearance',
    })

    await trigger.trigger('click')
    await wrapper.vm.$nextTick()

    const menu = menuElement()
    const items = optionElements()
    expect(menu.attributes('role')).toBe('menu')
    expect(menu.attributes('aria-labelledby')).toBe(trigger.attributes('id'))
    expect(menu.element.parentElement).toBe(document.body)
    expect(menu.attributes('style')).toContain('position: fixed')
    expect(menu.attributes('style')).toContain('width: 220px')
    expect(items).toHaveLength(3)
    expect(items.map((item) => item.attributes('aria-checked')))
      .toEqual(['true', 'false', 'false'])
    expect(items[0]?.find('[data-icon="check"]').exists()).toBe(true)
    expect(document.activeElement).toBe(items[0]?.element)
  })

  it('supports Arrow, Home, End, Enter, and Space while skipping disabled choices', async () => {
    const { trigger, wrapper } = mountMenu({
      options: [
        options[0]!,
        { ...options[1]!, disabled: true },
        options[2]!,
      ],
    })

    await trigger.trigger('keydown', { key: 'ArrowDown' })
    await wrapper.vm.$nextTick()
    let items = optionElements()
    expect(document.activeElement).toBe(items[0]?.element)

    await items[0]!.trigger('keydown', { key: 'ArrowDown' })
    expect(document.activeElement).toBe(items[2]?.element)

    await items[2]!.trigger('keydown', { key: 'ArrowDown' })
    expect(document.activeElement).toBe(items[0]?.element)

    await items[0]!.trigger('keydown', { key: 'End' })
    expect(document.activeElement).toBe(items[2]?.element)
    await items[2]!.trigger('keydown', { key: 'Home' })
    expect(document.activeElement).toBe(items[0]?.element)

    await items[0]!.trigger('keydown', { key: 'End' })
    await items[2]!.trigger('keydown', { key: 'Enter' })
    await wrapper.vm.$nextTick()

    expect(wrapper.emitted('update:modelValue')).toEqual([['dark']])
    await waitForMenuToClose()
    expect(document.activeElement).toBe(trigger.element)

    await trigger.trigger('keydown', { key: ' ' })
    await wrapper.vm.$nextTick()
    items = optionElements()
    await items[0]!.trigger('keydown', { key: ' ' })
    expect(wrapper.emitted('update:modelValue')).toEqual([['dark'], ['system']])
  })

  it('owns only the top Escape and restores focus without closing its parent layer', async () => {
    const parentToken = Symbol('parent-settings')
    registerModalLayer(parentToken)
    layerTokens.add(parentToken)
    const { trigger, wrapper } = mountMenu()

    await trigger.trigger('click')
    await wrapper.vm.$nextTick()
    expect(isTopModalLayer(parentToken)).toBe(false)

    await optionElements()[0]!.trigger('keydown', { key: 'Escape' })
    await wrapper.vm.$nextTick()

    await waitForMenuToClose()
    expect(isTopModalLayer(parentToken)).toBe(true)
    expect(document.activeElement).toBe(trigger.element)
  })

  it('handles Tab and outside pointerdown without leaking focus into the background', async () => {
    const { trigger, wrapper } = mountMenu()
    await trigger.trigger('click')
    await wrapper.vm.$nextTick()

    await optionElements()[0]!.trigger('keydown', { key: 'Tab' })
    await wrapper.vm.$nextTick()
    await waitForMenuToClose()
    expect(document.activeElement).toBe(trigger.element)

    const outside = document.createElement('button')
    document.body.appendChild(outside)
    await trigger.trigger('click')
    await wrapper.vm.$nextTick()
    outside.focus()
    outside.dispatchEvent(new MouseEvent('pointerdown', { bubbles: true }))
    await wrapper.vm.$nextTick()

    await waitForMenuToClose()
    expect(document.activeElement).toBe(outside)
  })

  it('repositions on viewport changes and constrains long menus to a scrollable height', async () => {
    const longOptions = Array.from({ length: 20 }, (_, index) => ({
      value: `locale-${index}`,
      label: `Locale ${index}`,
    }))
    const { trigger, wrapper } = mountMenu({
      modelValue: 'locale-0',
      options: longOptions,
    })
    let triggerRect = rect()
    trigger.element.getBoundingClientRect = vi.fn(() => triggerRect)

    await trigger.trigger('click')
    await wrapper.vm.$nextTick()

    const initialStyle = menuElement().attributes('style')
    expect(initialStyle).toContain('max-height: 320px')

    triggerRect = rect({
      bottom: 566,
      top: 530,
      y: 530,
    })
    window.dispatchEvent(new Event('resize'))
    await wrapper.vm.$nextTick()

    const repositionedStyle = menuElement().attributes('style')
    expect(repositionedStyle).not.toBe(initialStyle)
    expect(repositionedStyle).toContain('top:')

    const items = optionElements()
    await items[0]!.trigger('keydown', { key: 'End' })
    expect(HTMLElement.prototype.scrollIntoView).toHaveBeenCalled()
    expect(document.activeElement).toBe(items.at(-1)?.element)
  })
})
