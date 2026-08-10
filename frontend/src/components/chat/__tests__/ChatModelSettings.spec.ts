import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { nextTick } from 'vue'
import AppIcon from '@/components/icons/Icon.vue'
import zhChat from '@/i18n/locales/zh/chat'
import ChatModelSettings from '../ChatModelSettings.vue'

const COMPONENT_SOURCE = readFileSync(
  resolve(process.cwd(), 'src/components/chat/ChatModelSettings.vue'),
  'utf8',
)
const SCOPED_STYLE = COMPONENT_SOURCE.match(/<style scoped>([\s\S]*?)<\/style>/)?.[1] ?? ''
const GLOBAL_STYLE = Array.from(
  COMPONENT_SOURCE.matchAll(/<style>([\s\S]*?)<\/style>/g),
  (match) => match[1] ?? '',
).join('\n')

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

const IconStub = {
  props: ['name'],
  template: '<span :data-icon="name" />',
}

const MODEL_OPTIONS = [
  { value: 'gpt-5', label: 'GPT-5', description: 'Legacy description' },
  {
    value: 'gpt-5.6-sol',
    label: 'GPT-5.6 Sol',
    recommended: true,
    supportsReasoningSlider: true,
  },
]

let wrapper: VueWrapper | undefined

function panelElement(selector: string): HTMLElement {
  const element = document.body.querySelector<HTMLElement>(selector)
  if (!element) throw new Error(`Missing teleported element: ${selector}`)
  return element
}

function press(target: HTMLElement, key: string) {
  target.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true }))
}

function pointerEvent(
  type: string,
  init: MouseEventInit & {
    isPrimary?: boolean
    pointerId?: number
    pointerType?: string
  } = {},
) {
  const event = new MouseEvent(type, { bubbles: true, ...init })
  Object.defineProperty(event, 'pointerId', {
    configurable: true,
    value: init.pointerId ?? 1,
  })
  Object.defineProperty(event, 'pointerType', {
    configurable: true,
    value: init.pointerType ?? 'mouse',
  })
  Object.defineProperty(event, 'isPrimary', {
    configurable: true,
    value: init.isPrimary ?? true,
  })
  return event
}

function animationEvent(animationName: string) {
  const event = new Event('animationend', { bubbles: true })
  Object.defineProperty(event, 'animationName', {
    configurable: true,
    value: animationName,
  })
  return event
}

function enableFineHover() {
  vi.mocked(window.matchMedia).mockImplementation((query: string) => ({
    matches: query === '(hover: hover) and (pointer: fine)',
    media: query,
    onchange: null,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }))
}

async function expandAdvancedSettings() {
  panelElement('[data-test="chat-settings-advanced-toggle"]').click()
  await nextTick()
}

function mountSettings(overrides: Record<string, unknown> = {}) {
  wrapper = mount(ChatModelSettings, {
    attachTo: document.body,
    props: {
      modelValue: 'gpt-5.6-sol',
      reasoningEffort: 'low',
      modelOptions: MODEL_OPTIONS,
      ...overrides,
    },
    global: {
      stubs: {
        Icon: IconStub,
        ReasoningMaxCanvas: {
          template: '<canvas data-test="chat-settings-maximum-canvas" />',
        },
        Transition: true,
      },
    },
  })
  return wrapper
}

beforeEach(() => {
  vi.spyOn(window, 'matchMedia').mockImplementation((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }))
  Object.defineProperty(window, 'innerWidth', {
    configurable: true,
    writable: true,
    value: 1024,
  })
  Object.defineProperty(window, 'innerHeight', {
    configurable: true,
    writable: true,
    value: 768,
  })
})

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  document.body.innerHTML = ''
  vi.useRealTimers()
  vi.restoreAllMocks()
})

describe('ChatModelSettings', () => {
  it('uses the product term 思考强度 in visible and accessible Chinese copy', () => {
    expect(zhChat.chat.settings.label).toBe('模型 {model}，思考强度 {effort}')
    expect(zhChat.chat.settings.reasoning).toBe('思考强度')
    expect(zhChat.chat.settings.tooltip).toBe('思考强度')
  })

  it('shows only the selected reasoning level in the compact trigger', () => {
    const view = mountSettings()
    const trigger = view.get('[data-test="chat-model-settings-trigger"]')
    const tooltipId = trigger.attributes('aria-describedby')
    const tooltip = tooltipId ? document.getElementById(tooltipId) : null

    expect(tooltip).not.toBeNull()

    expect(trigger.text()).toContain('chat.settings.reasoningLevels.low')
    expect(trigger.text()).not.toContain('GPT-5.6 Sol')
    expect(trigger.find('[data-icon="sparkles"]').exists()).toBe(false)
    expect(trigger.find('[data-icon="chevronDown"]').exists()).toBe(true)
    expect(trigger.get('.chat-model-settings__selection').attributes('aria-hidden')).toBe('true')
    expect(view.get('.chat-model-settings__sizer').text()).toBe(
      'chat.settings.reasoningLevels.low',
    )
    expect(trigger.attributes('aria-label')).toBe('chat.settings.label')
    expect(trigger.attributes('aria-keyshortcuts')).toBe('Control+Shift+M')
    expect(trigger.attributes('aria-describedby')).toBe(tooltip?.id)
    expect(tooltip?.textContent).toContain('chat.settings.tooltip')
    expect(tooltip?.textContent).toContain('M')
  })

  it('toggles the settings menu from the advertised keyboard shortcut', async () => {
    const view = mountSettings()
    const trigger = view.get('[data-test="chat-model-settings-trigger"]')
    const shortcut = new KeyboardEvent('keydown', {
      code: 'KeyM',
      ctrlKey: true,
      shiftKey: true,
      bubbles: true,
      cancelable: true,
    })

    window.dispatchEvent(shortcut)
    await nextTick()

    expect(shortcut.defaultPrevented).toBe(true)
    expect(trigger.attributes('aria-expanded')).toBe('true')
    expect(trigger.attributes('aria-describedby')).toBeUndefined()
    expect(document.body.querySelector('[data-test="chat-settings-root-panel"]')).not.toBeNull()

    const closeShortcut = new KeyboardEvent('keydown', {
      code: 'KeyM',
      ctrlKey: true,
      shiftKey: true,
      bubbles: true,
      cancelable: true,
    })
    window.dispatchEvent(closeShortcut)
    await nextTick()

    expect(closeShortcut.defaultPrevented).toBe(true)
    expect(trigger.attributes('aria-expanded')).toBe('false')
    expect(document.body.querySelector('[data-test="chat-settings-root-panel"]')).toBeNull()
  })

  it('shows a compact non-default model name beside the selected reasoning level', () => {
    const view = mountSettings({
      modelValue: 'gpt-5.5',
      reasoningEffort: 'xhigh',
      modelOptions: [{ value: 'gpt-5.5', label: 'GPT-5.5' }],
    })
    const trigger = view.get('[data-test="chat-model-settings-trigger"]')

    expect(trigger.get('[data-test="chat-model-settings-trigger-model"]').text()).toBe('5.5')
    expect(trigger.get('[data-test="chat-model-settings-trigger-value"]').text())
      .toBe('chat.settings.reasoningLevels.xhigh')
    expect(trigger.text()).not.toContain('GPT-5.5')
    expect(view.get('.chat-model-settings__sizer').text())
      .toContain('5.5chat.settings.reasoningLevels.xhigh')
    expect(trigger.attributes('aria-label')).toBe('chat.settings.label')
  })

  it('keeps a content-sized slot while the trigger expands toward the inline start', async () => {
    const view = mountSettings()
    const trigger = view.get('[data-test="chat-model-settings-trigger"]')

    expect(SCOPED_STYLE).toContain('width: max-content;')
    expect(SCOPED_STYLE).toContain('margin-inline-start: 4px;')
    expect(SCOPED_STYLE).toContain('inset-inline-end: 0;')
    expect(SCOPED_STYLE).toContain('padding: 0 12px 0 14px;')
    expect(SCOPED_STYLE).toContain('gap: 6px;')
    expect(SCOPED_STYLE).toContain('width: 14px;')
    expect(SCOPED_STYLE).toContain('margin-inline-end: -2px;')
    expect(trigger.classes()).not.toContain('chat-model-settings__trigger--open')

    await trigger.trigger('click')
    expect(trigger.classes()).toContain('chat-model-settings__trigger--open')
    expect(SCOPED_STYLE).toContain('width: 164px;')
  })

  it('renders the four-step capability control for Sol and no automatic or Pro mode', async () => {
    const view = mountSettings()
    await view.get('[data-test="chat-model-settings-trigger"]').trigger('click')

    const root = panelElement('[data-test="chat-settings-root-panel"]')
    const visual = panelElement('[data-test="chat-settings-capability-visual"]')
    const slider = panelElement('[data-test="chat-settings-capability-slider"]') as HTMLInputElement
    const advanced = panelElement('[data-test="chat-settings-advanced-toggle"]')
    expect(root.textContent).toBe('chat.settings.advanced')
    expect(advanced.compareDocumentPosition(visual) & Node.DOCUMENT_POSITION_FOLLOWING)
      .not.toBe(0)
    expect(visual.style.getPropertyValue('--reasoning-progress').trim()).toBe('0')
    expect(visual.style.getPropertyValue('--reasoning-position').trim()).toBe('13px')
    expect(visual.style.getPropertyValue('--reasoning-fill-width').trim()).toBe('0px')
    expect(visual.style.getPropertyValue('--reasoning-fill-inset').trim()).toBe('196px')
    expect(slider.min).toBe('0')
    expect(slider.max).toBe('3')
    expect(slider.value).toBe('0')
    expect(slider.dir).toBe('ltr')
    const ticks = Array.from(
      visual.querySelectorAll<HTMLElement>('.chat-model-settings-popover__range-ticks > span'),
    )
    expect(ticks.map((tick) => tick.style.left)).toEqual([
      '13px',
      '69.66666666666666px',
      '126.33333333333333px',
      '183px',
    ])
    expect(ticks.every((tick) => (
      !tick.classList.contains('chat-model-settings-popover__range-tick--nearby')
    ))).toBe(true)
    expect(ticks.every((tick) => tick.getAttribute('style')?.includes(
      'width',
    ) !== true)).toBe(true)
    expect(ticks.filter((tick) => tick.classList.contains(
      'chat-model-settings-popover__range-tick--active',
    ))).toHaveLength(1)

    await expandAdvancedSettings()
    expect(root.textContent).toContain('GPT-5.6 Sol')
    expect(root.classList).toContain('chat-model-settings-popover__surface--advanced')
    expect(document.body.querySelector('[data-test="chat-settings-capability-slider"]')).toBeNull()

    const rows = panelElement('.chat-model-settings-popover__rows')
    const divider = panelElement('.chat-model-settings-popover__divider')
    const expandedAdvanced = panelElement('[data-test="chat-settings-advanced-toggle"]')
    expect(rows.compareDocumentPosition(divider) & Node.DOCUMENT_POSITION_FOLLOWING)
      .not.toBe(0)
    expect(divider.compareDocumentPosition(expandedAdvanced) & Node.DOCUMENT_POSITION_FOLLOWING)
      .not.toBe(0)

    panelElement('[data-test="chat-settings-reasoning-menu"]').click()
    await nextTick()

    const reasoningSubmenu = panelElement('[data-test="chat-settings-submenu"]')
    expect(reasoningSubmenu.querySelector('.chat-model-settings-popover__header')).toBeNull()
    expect(reasoningSubmenu.getAttribute('aria-label')).toBe('chat.settings.reasoning')
    const values = Array.from(document.body.querySelectorAll<HTMLElement>('[role="menuitemradio"]'))
      .map((element) => element.dataset.value)
    expect(values).toEqual(['low', 'medium', 'high', 'xhigh'])
    expect(document.body.textContent).not.toContain('chat.settings.reasoningLevels.auto')
    expect(document.body.textContent).not.toContain('chat.settings.reasoningDescriptions')
    expect(document.body.textContent).not.toMatch(/\bPro\b|专业/)
  })

  it('previews desktop submenus on hover without moving focus', async () => {
    vi.useFakeTimers()
    enableFineHover()
    const view = mountSettings()
    await view.get('[data-test="chat-model-settings-trigger"]').trigger('click')
    await expandAdvancedSettings()

    const advancedToggle = panelElement('[data-test="chat-settings-advanced-toggle"]')
    const modelRow = panelElement('[data-test="chat-settings-model-menu"]')
    const reasoningRow = panelElement('[data-test="chat-settings-reasoning-menu"]')
    expect(document.activeElement).toBe(advancedToggle)

    modelRow.dispatchEvent(pointerEvent('pointerenter'))
    vi.advanceTimersByTime(99)
    await nextTick()
    expect(document.body.querySelector('[data-test="chat-settings-submenu"]')).toBeNull()

    vi.advanceTimersByTime(1)
    await nextTick()

    expect(modelRow.getAttribute('aria-expanded')).toBe('true')
    expect(modelRow.getAttribute('data-state')).toBe('open')
    expect(reasoningRow.getAttribute('aria-expanded')).toBe('false')
    expect(reasoningRow.getAttribute('data-state')).toBe('closed')
    expect(panelElement('[data-test="chat-settings-submenu"]').getAttribute('aria-label'))
      .toBe('chat.settings.model')
    expect(document.activeElement).toBe(advancedToggle)

    reasoningRow.dispatchEvent(pointerEvent('pointerenter'))
    vi.advanceTimersByTime(100)
    await nextTick()

    expect(modelRow.getAttribute('aria-expanded')).toBe('false')
    expect(reasoningRow.getAttribute('aria-expanded')).toBe('true')
    expect(panelElement('[data-test="chat-settings-submenu"]').getAttribute('aria-label'))
      .toBe('chat.settings.reasoning')
    expect(document.activeElement).toBe(advancedToggle)
  })

  it('keeps the hover tunnel open and closes a preview after a short pointer exit', async () => {
    vi.useFakeTimers()
    enableFineHover()
    const view = mountSettings()
    await view.get('[data-test="chat-model-settings-trigger"]').trigger('click')
    await expandAdvancedSettings()

    const modelRow = panelElement('[data-test="chat-settings-model-menu"]')
    modelRow.dispatchEvent(pointerEvent('pointerenter'))
    vi.advanceTimersByTime(100)
    await nextTick()
    const submenu = panelElement('[data-test="chat-settings-submenu"]')

    modelRow.dispatchEvent(pointerEvent('pointerleave', { relatedTarget: submenu }))
    vi.advanceTimersByTime(200)
    await nextTick()
    expect(document.body.querySelector('[data-test="chat-settings-submenu"]')).not.toBeNull()

    submenu.dispatchEvent(pointerEvent('pointerleave'))
    vi.advanceTimersByTime(149)
    await nextTick()
    expect(document.body.querySelector('[data-test="chat-settings-submenu"]')).not.toBeNull()

    vi.advanceTimersByTime(1)
    await nextTick()
    expect(document.body.querySelector('[data-test="chat-settings-submenu"]')).toBeNull()
    expect(modelRow.getAttribute('aria-expanded')).toBe('false')
  })

  it('closes the active preview after leaving a second row before its intent delay', async () => {
    vi.useFakeTimers()
    enableFineHover()
    const view = mountSettings()
    await view.get('[data-test="chat-model-settings-trigger"]').trigger('click')
    await expandAdvancedSettings()

    const modelRow = panelElement('[data-test="chat-settings-model-menu"]')
    const reasoningRow = panelElement('[data-test="chat-settings-reasoning-menu"]')
    modelRow.dispatchEvent(pointerEvent('pointerenter'))
    vi.advanceTimersByTime(100)
    await nextTick()
    expect(modelRow.getAttribute('aria-expanded')).toBe('true')

    reasoningRow.dispatchEvent(pointerEvent('pointerenter'))
    vi.advanceTimersByTime(99)
    reasoningRow.dispatchEvent(pointerEvent('pointerleave'))
    vi.advanceTimersByTime(150)
    await nextTick()

    expect(document.body.querySelector('[data-test="chat-settings-submenu"]')).toBeNull()
    expect(modelRow.getAttribute('aria-expanded')).toBe('false')
    expect(reasoningRow.getAttribute('aria-expanded')).toBe('false')
  })

  it('promotes a hovered submenu on click and keeps it open after pointer exit', async () => {
    vi.useFakeTimers()
    enableFineHover()
    const view = mountSettings()
    await view.get('[data-test="chat-model-settings-trigger"]').trigger('click')
    await expandAdvancedSettings()

    const modelRow = panelElement('[data-test="chat-settings-model-menu"]')
    modelRow.dispatchEvent(pointerEvent('pointerenter'))
    vi.advanceTimersByTime(100)
    await nextTick()
    modelRow.click()
    await nextTick()

    const selectedOption = panelElement('[role="menuitemradio"][data-value="gpt-5.6-sol"]')
    expect(document.activeElement).toBe(selectedOption)

    modelRow.dispatchEvent(pointerEvent('pointerleave'))
    panelElement('[data-test="chat-settings-submenu"]')
      .dispatchEvent(pointerEvent('pointerleave'))
    vi.advanceTimersByTime(300)
    await nextTick()

    expect(document.body.querySelector('[data-test="chat-settings-submenu"]')).not.toBeNull()
    expect(modelRow.getAttribute('aria-expanded')).toBe('true')
  })

  it('does not preview submenus for touch, coarse pointers, or compact layouts', async () => {
    enableFineHover()
    const view = mountSettings()
    await view.get('[data-test="chat-model-settings-trigger"]').trigger('click')
    await expandAdvancedSettings()
    const modelRow = panelElement('[data-test="chat-settings-model-menu"]')

    modelRow.dispatchEvent(pointerEvent('pointerenter', { pointerType: 'touch' }))
    await nextTick()
    expect(document.body.querySelector('[data-test="chat-settings-submenu"]')).toBeNull()

    vi.mocked(window.matchMedia).mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }))
    modelRow.dispatchEvent(pointerEvent('pointerenter'))
    await nextTick()
    expect(document.body.querySelector('[data-test="chat-settings-submenu"]')).toBeNull()

    Object.defineProperty(window, 'innerWidth', {
      configurable: true,
      writable: true,
      value: 390,
    })
    enableFineHover()
    window.dispatchEvent(new Event('resize'))
    await nextTick()
    modelRow.dispatchEvent(pointerEvent('pointerenter'))
    await nextTick()
    expect(document.body.querySelector('[data-test="chat-settings-submenu"]')).toBeNull()

    modelRow.click()
    await nextTick()
    expect(document.body.querySelector('[data-test="chat-settings-submenu"]')).not.toBeNull()
  })

  it('matches the target menu geometry, neutral typography, and semantic theme palette', () => {
    expect(GLOBAL_STYLE).toContain('--chat-settings-surface: var(--workspace-popup-surface);')
    expect(GLOBAL_STYLE).toContain('--chat-settings-text-primary: var(--workspace-text);')
    expect(GLOBAL_STYLE).toContain('--chat-settings-text-secondary: var(--workspace-text-secondary);')
    expect(GLOBAL_STYLE).toContain('--chat-settings-text-tertiary: var(--workspace-text-muted);')
    expect(GLOBAL_STYLE).toContain('--chat-settings-divider: var(--workspace-divider);')
    expect(GLOBAL_STYLE).toContain('--chat-settings-hover: var(--workspace-hover);')
    expect(GLOBAL_STYLE).toContain('0 0 0 1px var(--workspace-border),')
    expect(GLOBAL_STYLE).toContain('var(--workspace-popover-shadow);')
    expect(GLOBAL_STYLE).toMatch(
      /html\.dark \.chat-model-settings-popover\s*{[\s\S]*?--chat-settings-surface: #353535;[\s\S]*?--chat-settings-slider-track: #4a4a4a;[\s\S]*?--chat-settings-slider-tick: #7d7d7d;[\s\S]*?--chat-settings-slider-thumb: #fff;/,
    )
    expect(GLOBAL_STYLE).toContain(
      '--chat-reasoning-slider-track: var(--chat-settings-slider-track);',
    )
    expect(GLOBAL_STYLE).toContain(
      '--chat-reasoning-slider-tick: var(--chat-settings-slider-tick);',
    )
    expect(GLOBAL_STYLE).toMatch(
      /__range-thumb\s*{[\s\S]*?background: var\(--chat-settings-slider-thumb\);/,
    )
    expect(GLOBAL_STYLE).not.toContain('font-family:')
    expect(GLOBAL_STYLE).toMatch(
      /__surface--root\s*{[\s\S]*?grid-template-rows: 96px;[\s\S]*?grid-template-rows 400ms cubic-bezier\(0\.19, 1, 0\.22, 1\);/,
    )
    expect(GLOBAL_STYLE).toMatch(
      /__surface--advanced\s*{\s*grid-template-rows: 132px;/,
    )
    expect(GLOBAL_STYLE).toMatch(
      /__surface--advanced\s+\.chat-model-settings-popover__advanced\s*{\s*top: 90px;/,
    )
    expect(GLOBAL_STYLE).toMatch(
      /__divider\s*{[\s\S]*?top: 85px;[\s\S]*?right: 6px;[\s\S]*?left: 6px;/,
    )
    expect(GLOBAL_STYLE).toMatch(/__rows\s*{[\s\S]*?top: 10px;/)
    expect(GLOBAL_STYLE).toMatch(/__advanced--open svg\s*{\s*transform: rotate\(-90deg\);/)
    expect(GLOBAL_STYLE).toMatch(
      /__advanced\s*{[\s\S]*?color: var\(--chat-settings-text-secondary\);[\s\S]*?font-size: var\(--workspace-type-body-size\);[\s\S]*?font-weight: var\(--workspace-type-body-weight\);[\s\S]*?letter-spacing: normal;[\s\S]*?line-height: 20px;/,
    )
    expect(GLOBAL_STYLE).toMatch(
      /__row-copy small\s*{[\s\S]*?color: var\(--chat-settings-text-primary\);[\s\S]*?font-size: var\(--workspace-type-body-size\);[\s\S]*?font-weight: var\(--workspace-type-body-weight\);[\s\S]*?letter-spacing: normal;[\s\S]*?line-height: 20px;/,
    )
    expect(GLOBAL_STYLE).toMatch(
      /__row-trailing strong\s*{[\s\S]*?color: var\(--chat-settings-text-secondary\);[\s\S]*?font-size: var\(--workspace-type-body-size\);[\s\S]*?font-weight: var\(--workspace-type-body-weight\);[\s\S]*?letter-spacing: normal;[\s\S]*?line-height: 20px;/,
    )
    expect(GLOBAL_STYLE).toMatch(
      /__option > strong\s*{[\s\S]*?color: inherit;[\s\S]*?font-size: var\(--workspace-type-body-size\);[\s\S]*?font-weight: var\(--workspace-type-body-weight\);[\s\S]*?letter-spacing: normal;[\s\S]*?line-height: 20px;/,
    )
    expect(GLOBAL_STYLE).toMatch(
      /__row-trailing svg\s*{[\s\S]*?width: 16px;[\s\S]*?height: 16px;/,
    )
    expect(GLOBAL_STYLE).toMatch(
      /__submenu\s*{[\s\S]*?width: max-content;[\s\S]*?min-width: 100px;[\s\S]*?max-width: min\(320px, calc\(100vw - 24px\)\);/,
    )
    expect(GLOBAL_STYLE).toMatch(
      /__submenu--right\s*{\s*left: calc\(100% - 8px\);/,
    )
    expect(GLOBAL_STYLE).toMatch(
      /__submenu--left\s*{\s*right: calc\(100% - 8px\);/,
    )
    expect(GLOBAL_STYLE).toMatch(
      /__options\s*{[\s\S]*?padding: 10px 0;/,
    )
    expect(GLOBAL_STYLE).toMatch(
      /__option\s*{[\s\S]*?width: calc\(100% - 20px\);[\s\S]*?gap: 24px;[\s\S]*?margin: 0 10px;/,
    )
    expect(GLOBAL_STYLE).toMatch(
      /__check-slot\s*{[\s\S]*?width: 16px;[\s\S]*?flex: 0 0 16px;/,
    )
  })

  it('uses the target 16px filled check glyph without changing the shared check icon', () => {
    wrapper = mount(AppIcon, {
      props: {
        name: 'chatCheck',
        size: 'sm',
      },
    })

    const svg = wrapper.get('svg')
    const group = svg.get('g')
    const path = group.get('path')
    expect(svg.attributes('viewBox')).toBe('0 0 16 16')
    expect(svg.classes()).toEqual(expect.arrayContaining(['h-4', 'w-4']))
    expect(group.attributes('fill')).toBe('currentColor')
    expect(group.attributes('stroke')).toBe('none')
    expect(path.attributes('d')).toBe(
      'M12.096 2.914a.7.7 0 0 1 1.134.772l-.069.125-6.25 9.166a.7.7 0 0 1-1.073.102l-3.75-3.75-.09-.11a.7.7 0 0 1 .97-.97l.11.09 3.153 3.152 5.774-8.469z',
    )
  })

  it('also renders the capability slider for an explicitly enabled Sol alias', async () => {
    const view = mountSettings({
      modelValue: 'sol-stable-alias',
      modelOptions: [{
        value: 'sol-stable-alias',
        label: 'Sol Stable',
        supportsReasoningSlider: true,
      }],
    })

    await view.get('[data-test="chat-model-settings-trigger"]').trigger('click')

    expect(document.body.querySelector('[data-test="chat-settings-capability-slider"]')).not.toBeNull()
    expect(document.body.querySelector('[data-test="chat-settings-capability-visual"]')).not.toBeNull()
  })

  it('falls back unknown reasoning values to low and the first slider position', async () => {
    const view = mountSettings({ reasoningEffort: 'unknown' })
    expect(view.get('[data-test="chat-model-settings-trigger"]').text())
      .toContain('chat.settings.reasoningLevels.low')

    await view.get('[data-test="chat-model-settings-trigger"]').trigger('click')
    expect((panelElement('[data-test="chat-settings-capability-slider"]') as HTMLInputElement).value)
      .toBe('0')
  })

  it('toggles the right-anchored expanded trigger state without changing its host footprint', async () => {
    const view = mountSettings()
    const host = view.get('.chat-model-settings')
    const trigger = view.get('[data-test="chat-model-settings-trigger"]')

    expect(host.classes()).toContain('chat-model-settings')
    expect(trigger.classes()).not.toContain('chat-model-settings__trigger--open')
    expect(trigger.attributes('data-state')).toBe('closed')
    expect(trigger.attributes('aria-expanded')).toBe('false')

    await trigger.trigger('click')
    expect(trigger.classes()).toContain('chat-model-settings__trigger--open')
    expect(trigger.attributes('data-state')).toBe('open')
    expect(trigger.attributes('aria-expanded')).toBe('true')
    expect(document.body.querySelector('[data-test="chat-model-settings-popover"]')).not.toBeNull()

    await trigger.trigger('click')
    await nextTick()
    expect(trigger.classes()).not.toContain('chat-model-settings__trigger--open')
    expect(trigger.attributes('data-state')).toBe('closed')
    expect(trigger.attributes('aria-expanded')).toBe('false')
    expect(document.body.querySelector('[data-test="chat-model-settings-popover"]')).toBeNull()
  })

  it('uses a stable readable trigger with a reduced-motion override', () => {
    expect(SCOPED_STYLE).toContain('max-width: min(164px, 42vw);')
    expect(SCOPED_STYLE).toContain('width: max-content;')
    expect(SCOPED_STYLE).toContain('width: 100%;')
    expect(SCOPED_STYLE).toContain('width: 164px;')
    expect(SCOPED_STYLE).toContain('border-radius: 999px;')
    expect(SCOPED_STYLE).toContain(
      'color: var(--chat-composer-muted-fg, var(--lx-clay-text-secondary));',
    )
    expect(SCOPED_STYLE).not.toContain('font-family:')
    expect(SCOPED_STYLE).toContain('--chat-model-trigger-font-size: 16px;')
    expect(SCOPED_STYLE).toContain('--chat-model-trigger-font-weight: 400;')
    expect(SCOPED_STYLE).toContain('--chat-model-trigger-line-height: 26px;')
    expect(SCOPED_STYLE).toMatch(
      /__sizer-selection strong\s*\{[^}]*font-size: var\(--chat-model-trigger-font-size\);[^}]*font-weight: var\(--chat-model-trigger-font-weight\);[^}]*line-height: var\(--chat-model-trigger-line-height\);/,
    )
    expect(SCOPED_STYLE).toMatch(
      /__selection strong\s*\{[^}]*font-size: var\(--chat-model-trigger-font-size\);[^}]*font-weight: var\(--chat-model-trigger-font-weight\);[^}]*line-height: var\(--chat-model-trigger-line-height\);/,
    )
    expect(SCOPED_STYLE).toMatch(
      /__trigger-effort--solo\s*\{[^}]*color: var\(--chat-composer-tertiary-fg, var\(--lx-clay-text-muted\)\);/,
    )
    expect(SCOPED_STYLE).toContain('.chat-model-settings__selection')
    expect(SCOPED_STYLE).not.toContain('clip-path:')
    expect(SCOPED_STYLE).not.toContain('transform: scaleX(')
    expect(SCOPED_STYLE).toMatch(
      /@media \(prefers-reduced-motion: reduce\)[\s\S]*?__trigger,[\s\S]*?__chevron[\s\S]*?transition: none;/,
    )
  })

  it.each([
    ['GPT-5.5', { value: 'gpt-5.5', label: 'GPT-5.5' }],
    ['a model without the legacy capability flag', { value: 'gpt-5.6-terra', label: 'GPT-5.6 Terra' }],
    ['a model with an explicit legacy false flag', {
      value: 'gpt-5.6-luna',
      label: 'GPT-5.6 Luna',
      supportsReasoningSlider: false,
    }],
  ])('renders the four-level capability slider for %s', async (_name, option) => {
    const view = mountSettings({
      modelValue: option.value,
      modelOptions: [option],
    })

    await view.get('[data-test="chat-model-settings-trigger"]').trigger('click')

    expect(document.body.querySelector('[data-test="chat-settings-capability-slider"]')).not.toBeNull()
    expect(document.body.querySelector('[data-test="chat-settings-capability-visual"]')).not.toBeNull()

    await expandAdvancedSettings()
    panelElement('[data-test="chat-settings-reasoning-menu"]').click()
    await nextTick()
    const values = Array.from(document.body.querySelectorAll<HTMLElement>('[role="menuitemradio"]'))
      .map((element) => element.dataset.value)
    expect(values).toEqual(['low', 'medium', 'high', 'xhigh'])
  })

  it('keeps the root menu fixed and opens a desktop submenu to the left near the right edge', async () => {
    Object.defineProperty(window, 'innerWidth', {
      configurable: true,
      writable: true,
      value: 1024,
    })
    const view = mountSettings({ modelValue: 'gpt-5' })
    const trigger = view.get('[data-test="chat-model-settings-trigger"]')
    vi.spyOn(HTMLElement.prototype, 'offsetWidth', 'get').mockImplementation(function (this: HTMLElement) {
      return this.matches('[data-test="chat-settings-submenu"]') ? 168 : 0
    })
    const triggerRectSpy = vi.spyOn(trigger.element, 'getBoundingClientRect').mockReturnValue({
      x: 700,
      y: 500,
      left: 700,
      top: 500,
      right: 864,
      bottom: 536,
      width: 164,
      height: 36,
      toJSON: () => ({}),
    } as DOMRect)

    await trigger.trigger('click')
    await nextTick()
    const initialPanelLeft = panelElement('[data-test="chat-model-settings-popover"]').style.left
    expect(initialPanelLeft).toBe('670px')
    expect(panelElement('[data-test="chat-model-settings-popover"]').style.bottom).toBe('272px')

    triggerRectSpy.mockReturnValue({
      x: 704,
      y: 500,
      left: 704,
      top: 500,
      right: 868,
      bottom: 536,
      width: 164,
      height: 36,
      toJSON: () => ({}),
    } as DOMRect)
    const widthTransitionEnd = new Event('transitionend', { bubbles: true })
    Object.defineProperty(widthTransitionEnd, 'propertyName', { value: 'width' })
    trigger.element.dispatchEvent(widthTransitionEnd)
    await nextTick()
    expect(panelElement('[data-test="chat-model-settings-popover"]').style.left).toBe('674px')

    await expandAdvancedSettings()
    panelElement('[data-test="chat-settings-model-menu"]').click()
    await nextTick()
    window.dispatchEvent(new Event('scroll'))
    await nextTick()

    expect(document.body.querySelector('[data-test="chat-settings-root-panel"]')).not.toBeNull()
    const submenuClasses = panelElement('[data-test="chat-settings-submenu"]').classList
    expect(submenuClasses).toContain('chat-model-settings-popover__submenu--left')
    expect(submenuClasses).not.toContain('chat-model-settings-popover__submenu--right')
    expect(panelElement('[data-test="chat-model-settings-popover"]').style.left).toBe('674px')

    panelElement('[role="menuitemradio"][data-value="gpt-5.6-sol"]').click()
    await nextTick()

    expect(view.emitted('update:modelValue')).toEqual([['gpt-5.6-sol']])
    expect(document.body.querySelector('.chat-model-settings-popover')).toBeNull()
    expect(document.activeElement).toBe(trigger.element)

    await trigger.trigger('click')
    await expandAdvancedSettings()
    panelElement('[data-test="chat-settings-reasoning-menu"]').click()
    await nextTick()
    panelElement('[role="menuitemradio"][data-value="high"]').click()
    await nextTick()

    expect(view.emitted('update:reasoningEffort')).toEqual([['high']])
    expect(document.activeElement).toBe(trigger.element)
  })

  it('prefers opening a desktop submenu to the right without moving the root menu', async () => {
    const view = mountSettings({ modelValue: 'gpt-5' })
    const trigger = view.get('[data-test="chat-model-settings-trigger"]')
    vi.spyOn(HTMLElement.prototype, 'offsetWidth', 'get').mockImplementation(function (this: HTMLElement) {
      return this.matches('[data-test="chat-settings-submenu"]') ? 168 : 0
    })
    vi.spyOn(trigger.element, 'getBoundingClientRect').mockReturnValue({
      x: 400,
      y: 500,
      left: 400,
      top: 500,
      right: 564,
      bottom: 536,
      width: 164,
      height: 36,
      toJSON: () => ({}),
    } as DOMRect)

    await trigger.trigger('click')
    await nextTick()
    const initialLeft = panelElement('[data-test="chat-model-settings-popover"]').style.left
    expect(initialLeft).toBe('370px')

    await expandAdvancedSettings()
    panelElement('[data-test="chat-settings-model-menu"]').click()
    await nextTick()

    const submenuClasses = panelElement('[data-test="chat-settings-submenu"]').classList
    expect(submenuClasses).toContain('chat-model-settings-popover__submenu--right')
    expect(submenuClasses).not.toContain('chat-model-settings-popover__submenu--left')
    expect(panelElement('[data-test="chat-model-settings-popover"]').style.left).toBe(initialLeft)
  })

  it('keeps the root menu fixed while switching between differently sized submenus', async () => {
    const view = mountSettings({ modelValue: 'gpt-5' })
    const trigger = view.get('[data-test="chat-model-settings-trigger"]')
    let submenuWidth = 168
    vi.spyOn(HTMLElement.prototype, 'offsetWidth', 'get').mockImplementation(function (this: HTMLElement) {
      return this.matches('[data-test="chat-settings-submenu"]') ? submenuWidth : 0
    })
    vi.spyOn(trigger.element, 'getBoundingClientRect').mockReturnValue({
      x: 640,
      y: 500,
      left: 640,
      top: 500,
      right: 804,
      bottom: 536,
      width: 164,
      height: 36,
      toJSON: () => ({}),
    } as DOMRect)

    await trigger.trigger('click')
    await expandAdvancedSettings()
    const rootLeft = panelElement('[data-test="chat-model-settings-popover"]').style.left
    expect(rootLeft).toBe('610px')

    panelElement('[data-test="chat-settings-model-menu"]').click()
    await nextTick()
    expect(panelElement('[data-test="chat-settings-submenu"]').classList)
      .toContain('chat-model-settings-popover__submenu--right')
    expect(panelElement('[data-test="chat-model-settings-popover"]').style.left).toBe(rootLeft)

    submenuWidth = 220
    panelElement('[data-test="chat-settings-reasoning-menu"]').click()
    await nextTick()
    await nextTick()
    expect(panelElement('[data-test="chat-settings-submenu"]').classList)
      .toContain('chat-model-settings-popover__submenu--left')
    expect(panelElement('[data-test="chat-model-settings-popover"]').style.left).toBe(rootLeft)
  })

  it.each([
    {
      name: 'above',
      rect: { top: 300, right: 370, bottom: 336 },
      expectedTop: '12px',
      expectedMaxHeight: '284px',
    },
    {
      name: 'below',
      rect: { top: 20, right: 370, bottom: 56 },
      expectedTop: '60px',
      expectedMaxHeight: '303px',
    },
  ])('clamps a compact submenu to the larger $name viewport region', async ({
    rect,
    expectedTop,
    expectedMaxHeight,
  }) => {
    Object.defineProperty(window, 'innerWidth', {
      configurable: true,
      writable: true,
      value: 390,
    })
    Object.defineProperty(window, 'innerHeight', {
      configurable: true,
      writable: true,
      value: 375,
    })
    vi.spyOn(HTMLElement.prototype, 'offsetHeight', 'get').mockImplementation(function (this: HTMLElement) {
      if (this.matches('[data-test="chat-settings-root-panel"]')) return 150
      if (this.matches('[data-test="chat-settings-submenu"]')) return 280
      return 0
    })

    const view = mountSettings({ modelValue: 'gpt-5' })
    const trigger = view.get('[data-test="chat-model-settings-trigger"]')
    vi.spyOn(trigger.element, 'getBoundingClientRect').mockReturnValue({
      x: rect.right - 48,
      y: rect.top,
      left: rect.right - 48,
      top: rect.top,
      right: rect.right,
      bottom: rect.bottom,
      width: 48,
      height: 36,
      toJSON: () => ({}),
    } as DOMRect)

    await trigger.trigger('click')
    await expandAdvancedSettings()
    panelElement('[data-test="chat-settings-model-menu"]').click()
    await nextTick()
    window.dispatchEvent(new Event('scroll'))
    await nextTick()

    const popover = panelElement('[data-test="chat-model-settings-popover"]')
    expect(popover.classList).toContain('chat-model-settings-popover--compact')
    expect(popover.style.top).toBe(expectedTop)
    expect(popover.style.maxHeight).toBe(expectedMaxHeight)
    expect(popover.style.bottom).toBe('')
    expect(panelElement('[data-test="chat-settings-submenu"]').classList)
      .toContain(`chat-model-settings-popover__submenu--${rect.top > 100 ? 'above' : 'below'}`)
  })

  it('maps the capability slider directly to the four supported efforts', async () => {
    const view = mountSettings()
    await view.get('[data-test="chat-model-settings-trigger"]').trigger('click')
    const slider = panelElement('[data-test="chat-settings-capability-slider"]') as HTMLInputElement

    slider.value = '3'
    slider.dispatchEvent(new Event('input', { bubbles: true }))
    await nextTick()

    expect(view.emitted('update:reasoningEffort')).toEqual([['xhigh']])
    expect(slider.getAttribute('aria-valuetext')).toBe('chat.settings.reasoningLevels.xhigh')
    await view.setProps({ reasoningEffort: 'xhigh' })
    const visual = panelElement('[data-test="chat-settings-capability-visual"]')
    expect(visual.style.getPropertyValue('--reasoning-progress').trim()).toBe('1')
    expect(visual.style.getPropertyValue('--reasoning-position').trim()).toBe('183px')
    expect(visual.style.getPropertyValue('--reasoning-fill-width').trim()).toBe('196px')
    expect(visual.style.getPropertyValue('--reasoning-fill-inset').trim()).toBe('0px')
    expect(visual.querySelectorAll(
      '.chat-model-settings-popover__range-tick--active',
    )).toHaveLength(4)
    expect(document.body.querySelector('.chat-model-settings-popover')).not.toBeNull()
  })

  it('runs the purple terminal effect only at the maximum effort and supports interruption', async () => {
    const view = mountSettings()
    await view.get('[data-test="chat-model-settings-trigger"]').trigger('click')
    const slider = panelElement('[data-test="chat-settings-capability-slider"]') as HTMLInputElement
    const visual = panelElement('[data-test="chat-settings-capability-visual"]')

    const setValue = async (value: string) => {
      slider.value = value
      slider.dispatchEvent(new Event('input', { bubbles: true }))
      await nextTick()
    }

    expect(visual.classList).not.toContain(
      'chat-model-settings-popover__range-wrap--maximum',
    )
    expect(visual.dataset.max).toBeUndefined()

    await setValue('3')
    expect(visual.classList).toContain(
      'chat-model-settings-popover__range-wrap--maximum',
    )
    expect(visual.dataset.max).toBe('true')
    expect(panelElement('[data-test="chat-settings-maximum-effects"]')).toBeTruthy()
    expect(panelElement('[data-test="chat-settings-maximum-canvas"]')).toBeTruthy()
    const maximumEffects = panelElement('[data-test="chat-settings-maximum-effects"]')
    expect(maximumEffects.classList).toContain(
      'chat-model-settings-popover__max-effects--entering',
    )
    expect(visual.querySelectorAll(
      '.chat-model-settings-popover__max-track-particles > span',
    )).toHaveLength(14)
    expect(visual.querySelectorAll(
      '.chat-model-settings-popover__max-burst > span',
    )).toHaveLength(16)
    const initialBurst = panelElement('.chat-model-settings-popover__max-burst')
    const initialToken = initialBurst.dataset.entryToken
    expect(slider.getAttribute('aria-valuetext')).toBe('chat.settings.reasoningLevels.xhigh')

    await view.setProps({ reasoningEffort: 'xhigh' })
    expect(panelElement('.chat-model-settings-popover__max-burst')).toBe(initialBurst)
    expect(initialBurst.dataset.entryToken).toBe(initialToken)

    await setValue('3')
    expect(panelElement('.chat-model-settings-popover__max-burst')).toBe(initialBurst)
    expect(initialBurst.dataset.entryToken).toBe(initialToken)

    panelElement('.chat-model-settings-popover__max-fill').dispatchEvent(
      animationEvent('chat-reasoning-max-reveal'),
    )
    await nextTick()
    expect(maximumEffects.classList).not.toContain(
      'chat-model-settings-popover__max-effects--entering',
    )
    initialBurst.dispatchEvent(animationEvent('chat-reasoning-max-burst-lifecycle'))
    await nextTick()
    expect(visual.querySelector('.chat-model-settings-popover__max-burst')).toBeNull()

    await setValue('2')
    expect(visual.classList).not.toContain(
      'chat-model-settings-popover__range-wrap--maximum',
    )
    expect(visual.querySelector('[data-test="chat-settings-maximum-effects"]')).toBeNull()
    expect(visual.querySelector('.chat-model-settings-popover__max-burst')).toBeNull()
    await view.setProps({ reasoningEffort: 'high' })

    await setValue('3')
    expect(visual.classList).toContain(
      'chat-model-settings-popover__range-wrap--maximum',
    )
    expect(visual.querySelectorAll(
      '.chat-model-settings-popover__max-burst > span',
    )).toHaveLength(16)
    expect(panelElement('.chat-model-settings-popover__max-burst').dataset.entryToken)
      .not.toBe(initialToken)
    expect(view.emitted('update:reasoningEffort')).toEqual([
      ['xhigh'],
      ['high'],
      ['xhigh'],
    ])
  })

  it('opens an existing maximum effort in its stable state without replaying entry motion', async () => {
    const view = mountSettings({ reasoningEffort: 'xhigh' })
    const trigger = view.get('[data-test="chat-model-settings-trigger"]')

    await trigger.trigger('click')
    let maximumEffects = panelElement('[data-test="chat-settings-maximum-effects"]')
    expect(maximumEffects.classList).not.toContain(
      'chat-model-settings-popover__max-effects--entering',
    )
    expect(document.body.querySelector('.chat-model-settings-popover__max-burst')).toBeNull()
    expect(panelElement('[data-test="chat-settings-maximum-canvas"]')).toBeTruthy()

    await view.setProps({ reasoningEffort: 'high' })
    expect(document.body.querySelector('[data-test="chat-settings-maximum-effects"]')).toBeNull()
    await view.setProps({ reasoningEffort: 'xhigh' })
    maximumEffects = panelElement('[data-test="chat-settings-maximum-effects"]')
    expect(maximumEffects.classList).not.toContain(
      'chat-model-settings-popover__max-effects--entering',
    )
    expect(document.body.querySelector('.chat-model-settings-popover__max-burst')).toBeNull()

    panelElement('[data-test="chat-settings-advanced-toggle"]').click()
    await nextTick()
    panelElement('[data-test="chat-settings-advanced-toggle"]').click()
    await nextTick()
    expect(panelElement('[data-test="chat-settings-maximum-effects"]').classList).not.toContain(
      'chat-model-settings-popover__max-effects--entering',
    )
    expect(document.body.querySelector('.chat-model-settings-popover__max-burst')).toBeNull()

    await trigger.trigger('click')
    await trigger.trigger('click')
    maximumEffects = panelElement('[data-test="chat-settings-maximum-effects"]')
    expect(maximumEffects.classList).not.toContain(
      'chat-model-settings-popover__max-effects--entering',
    )
    expect(document.body.querySelector('.chat-model-settings-popover__max-burst')).toBeNull()

    await view.setProps({ modelValue: 'gpt-5' })
    maximumEffects = panelElement('[data-test="chat-settings-maximum-effects"]')
    expect(maximumEffects.classList).not.toContain(
      'chat-model-settings-popover__max-effects--entering',
    )
    expect(document.body.querySelector('.chat-model-settings-popover__max-burst')).toBeNull()
  })

  it('ignores stale maximum animation events after a rapid re-entry', async () => {
    const view = mountSettings()
    await view.get('[data-test="chat-model-settings-trigger"]').trigger('click')
    const slider = panelElement('[data-test="chat-settings-capability-slider"]') as HTMLInputElement
    const setValue = async (value: string) => {
      slider.value = value
      slider.dispatchEvent(new Event('input', { bubbles: true }))
      await nextTick()
    }

    await setValue('3')
    const staleFill = panelElement('.chat-model-settings-popover__max-fill')
    const staleBurst = panelElement('.chat-model-settings-popover__max-burst')
    const staleToken = staleBurst.dataset.entryToken
    await setValue('2')
    await setValue('3')

    const currentEffects = panelElement('[data-test="chat-settings-maximum-effects"]')
    const currentBurst = panelElement('.chat-model-settings-popover__max-burst')
    expect(currentBurst.dataset.entryToken).not.toBe(staleToken)

    staleFill.dispatchEvent(animationEvent('chat-reasoning-max-reveal'))
    staleBurst.dispatchEvent(animationEvent('chat-reasoning-max-burst-lifecycle'))
    await nextTick()

    expect(currentEffects.classList).toContain(
      'chat-model-settings-popover__max-effects--entering',
    )
    expect(panelElement('.chat-model-settings-popover__max-burst')).toBe(currentBurst)
  })

  it('blocks Shift text selection without canceling the native range interaction', async () => {
    const view = mountSettings()
    const trigger = view.get('[data-test="chat-model-settings-trigger"]')
    const triggerSelection = new Event('selectstart', { bubbles: true, cancelable: true })
    trigger.element.dispatchEvent(triggerSelection)
    expect(triggerSelection.defaultPrevented).toBe(true)

    await trigger.trigger('click')
    const popover = panelElement('[data-test="chat-model-settings-popover"]')
    const panelSelection = new Event('selectstart', { bubbles: true, cancelable: true })
    const panelDrag = new Event('dragstart', { bubbles: true, cancelable: true })
    popover.dispatchEvent(panelSelection)
    popover.dispatchEvent(panelDrag)
    expect(panelSelection.defaultPrevented).toBe(true)
    expect(panelDrag.defaultPrevented).toBe(true)

    const removeAllRanges = vi.fn()
    vi.spyOn(window, 'getSelection').mockReturnValue({
      removeAllRanges,
    } as unknown as Selection)
    const slider = panelElement('[data-test="chat-settings-capability-slider"]')
    const shiftedPointerDown = pointerEvent('pointerdown', {
      pointerId: 31,
      shiftKey: true,
      cancelable: true,
    })
    slider.dispatchEvent(shiftedPointerDown)
    await nextTick()
    expect(shiftedPointerDown.defaultPrevented).toBe(false)
    expect(removeAllRanges).toHaveBeenCalledOnce()

    slider.focus()
    const shiftedArrow = new KeyboardEvent('keydown', {
      key: 'ArrowRight',
      shiftKey: true,
      bubbles: true,
      cancelable: true,
    })
    slider.dispatchEvent(shiftedArrow)
    expect(shiftedArrow.defaultPrevented).toBe(false)
    expect(document.activeElement).toBe(slider)

    slider.dispatchEvent(pointerEvent('pointerup', { pointerId: 31, shiftKey: true }))
  })

  it('uses the short snap timing from pointer down and springs back on release', async () => {
    const view = mountSettings()
    await view.get('[data-test="chat-model-settings-trigger"]').trigger('click')
    const slider = panelElement('[data-test="chat-settings-capability-slider"]')
    const visual = panelElement('[data-test="chat-settings-capability-visual"]')

    slider.dispatchEvent(pointerEvent('pointerdown', { clientX: 20 }))
    await nextTick()
    expect(visual.classList).toContain('chat-model-settings-popover__range-wrap--dragging')

    slider.dispatchEvent(pointerEvent('pointerup', { clientX: 24 }))
    await nextTick()
    expect(visual.classList).not.toContain(
      'chat-model-settings-popover__range-wrap--dragging',
    )
    expect(visual.classList).toContain(
      'chat-model-settings-popover__range-wrap--settling',
    )

    const thumb = visual.querySelector<HTMLElement>(
      '.chat-model-settings-popover__range-thumb',
    )
    const animationEnd = new Event('animationend', { bubbles: true })
    Object.defineProperty(animationEnd, 'animationName', {
      configurable: true,
      value: 'chat-reasoning-thumb-settle',
    })
    thumb?.dispatchEvent(animationEnd)
    await nextTick()
    expect(visual.classList).not.toContain(
      'chat-model-settings-popover__range-wrap--settling',
    )
  })

  it('captures the pointer and clears dragging after capture loss or menu close', async () => {
    const view = mountSettings()
    const trigger = view.get('[data-test="chat-model-settings-trigger"]')
    await trigger.trigger('click')
    const slider = panelElement('[data-test="chat-settings-capability-slider"]')
    const visual = panelElement('[data-test="chat-settings-capability-visual"]')
    const setPointerCapture = vi.fn()
    const hasPointerCapture = vi.fn(() => true)
    const releasePointerCapture = vi.fn()
    Object.defineProperties(slider, {
      setPointerCapture: { configurable: true, value: setPointerCapture },
      hasPointerCapture: { configurable: true, value: hasPointerCapture },
      releasePointerCapture: { configurable: true, value: releasePointerCapture },
    })

    slider.dispatchEvent(pointerEvent('pointerdown', { clientX: 20, pointerId: 7 }))
    expect(setPointerCapture).toHaveBeenCalledWith(7)
    slider.dispatchEvent(pointerEvent('pointerdown', { clientX: 24, pointerId: 8 }))
    slider.dispatchEvent(pointerEvent('lostpointercapture', { pointerId: 8 }))
    await nextTick()
    expect(setPointerCapture).not.toHaveBeenCalledWith(8)
    expect(visual.classList).toContain('chat-model-settings-popover__range-wrap--dragging')

    slider.dispatchEvent(pointerEvent('pointermove', { clientX: 24, pointerId: 7 }))
    await nextTick()
    expect(visual.classList).toContain('chat-model-settings-popover__range-wrap--dragging')

    slider.dispatchEvent(pointerEvent('lostpointercapture', { pointerId: 7 }))
    await nextTick()
    expect(visual.classList).not.toContain('chat-model-settings-popover__range-wrap--dragging')
    expect(visual.classList).toContain('chat-model-settings-popover__range-wrap--settling')

    slider.dispatchEvent(pointerEvent('pointerdown', { clientX: 20, pointerId: 8 }))
    slider.dispatchEvent(pointerEvent('pointermove', { clientX: 24, pointerId: 8 }))
    slider.dispatchEvent(pointerEvent('pointerup', { clientX: 24, pointerId: 8 }))
    await nextTick()
    expect(hasPointerCapture).toHaveBeenCalledWith(8)
    expect(releasePointerCapture).toHaveBeenCalledWith(8)

    slider.dispatchEvent(pointerEvent('pointerdown', { clientX: 20, pointerId: 9 }))
    slider.dispatchEvent(pointerEvent('pointermove', { clientX: 24, pointerId: 9 }))
    await trigger.trigger('click')
    await trigger.trigger('click')
    expect(panelElement('[data-test="chat-settings-capability-visual"]').classList).not.toContain(
      'chat-model-settings-popover__range-wrap--dragging',
    )
  })

  it('enlarges only the nearby stop and clears proximity for gaps, touch, and exit', async () => {
    const view = mountSettings()
    await view.get('[data-test="chat-model-settings-trigger"]').trigger('click')
    const slider = panelElement('[data-test="chat-settings-capability-slider"]')
    const visual = panelElement('[data-test="chat-settings-capability-visual"]')
    vi.spyOn(visual, 'getBoundingClientRect').mockReturnValue({
      x: 100,
      y: 200,
      left: 100,
      top: 200,
      right: 296,
      bottom: 232,
      width: 196,
      height: 32,
      toJSON: () => ({}),
    } as DOMRect)
    const ticks = Array.from(
      visual.querySelectorAll<HTMLElement>('.chat-model-settings-popover__range-ticks > span'),
    )

    slider.dispatchEvent(pointerEvent('pointermove', { clientX: 113, clientY: 216 }))
    await nextTick()
    expect(visual.classList).toContain(
      'chat-model-settings-popover__range-wrap--thumb-nearby',
    )
    expect(ticks.filter((tick) => tick.classList.contains(
      'chat-model-settings-popover__range-tick--nearby',
    ))).toHaveLength(0)

    slider.dispatchEvent(pointerEvent('pointermove', {
      clientX: 169.6667,
      clientY: 216,
    }))
    await nextTick()
    expect(visual.classList).not.toContain(
      'chat-model-settings-popover__range-wrap--thumb-nearby',
    )
    expect(ticks[1]?.classList).toContain(
      'chat-model-settings-popover__range-tick--nearby',
    )
    expect(ticks.filter((tick) => tick.classList.contains(
      'chat-model-settings-popover__range-tick--nearby',
    ))).toHaveLength(1)
    expect((slider as HTMLInputElement).value).toBe('0')
    expect(view.emitted('update:reasoningEffort')).toBeUndefined()

    slider.dispatchEvent(pointerEvent('pointermove', { clientX: 141, clientY: 216 }))
    await nextTick()
    expect(visual.querySelector(
      '.chat-model-settings-popover__range-tick--nearby',
    )).toBeNull()

    slider.dispatchEvent(pointerEvent('pointermove', {
      clientX: 226.3333,
      clientY: 216,
      pointerType: 'touch',
    }))
    await nextTick()
    expect(visual.querySelector(
      '.chat-model-settings-popover__range-tick--nearby',
    )).toBeNull()

    slider.dispatchEvent(pointerEvent('pointermove', { clientX: 226.3333, clientY: 216 }))
    slider.dispatchEvent(pointerEvent('pointerleave', { clientX: 300, clientY: 216 }))
    await nextTick()
    expect(visual.querySelector(
      '.chat-model-settings-popover__range-tick--nearby',
    )).toBeNull()
  })

  it('keeps the visual thumb, fill, native hit area, and reduced motion in one contract', () => {
    expect(GLOBAL_STYLE).toContain(
      'clip-path: inset(0 var(--reasoning-fill-inset) 0 0 round 999px);',
    )
    expect(GLOBAL_STYLE).toContain(
      'transition: clip-path var(--reasoning-motion-duration) var(--reasoning-motion-easing);',
    )
    expect(GLOBAL_STYLE).toMatch(
      /__max-effects-viewport[\s\S]*?clip-path: inset\(0 var\(--reasoning-fill-inset\) 0 0 round 999px\);[\s\S]*?transition: clip-path var\(--reasoning-motion-duration\)/,
    )
    expect(GLOBAL_STYLE).toMatch(
      /__range-thumb-track[\s\S]*?translate3d\(var\(--reasoning-position\), 0, 0\);[\s\S]*?transition: transform var\(--reasoning-motion-duration\) var\(--reasoning-motion-easing\);/,
    )
    expect(GLOBAL_STYLE).toContain('left: -1px;')
    expect(GLOBAL_STYLE).toContain('width: calc(100% + 2px);')
    expect(GLOBAL_STYLE).toMatch(
      /__range::\-webkit-slider-thumb[\s\S]*?width: var\(--reasoning-thumb-size\);[\s\S]*?height: var\(--reasoning-thumb-size\);/,
    )
    expect(GLOBAL_STYLE).toMatch(
      /__range::\-moz-range-thumb[\s\S]*?width: var\(--reasoning-thumb-size\);[\s\S]*?height: var\(--reasoning-thumb-size\);/,
    )
    expect(GLOBAL_STYLE).not.toContain('range-tick--default')
    expect(GLOBAL_STYLE).toContain('range-tick--nearby')
    expect(GLOBAL_STYLE).toContain('translate(-50%, -50%) scale(2)')
    expect(GLOBAL_STYLE).toContain('range-wrap--thumb-nearby')
    expect(SCOPED_STYLE).toMatch(
      /\.chat-model-settings\s*\{[\s\S]*?-webkit-user-select: none;[\s\S]*?user-select: none;/,
    )
    expect(GLOBAL_STYLE).toMatch(
      /\.chat-model-settings-popover\s*\{[\s\S]*?-webkit-user-select: none;[\s\S]*?user-select: none;/,
    )
    expect(GLOBAL_STYLE).toMatch(
      /__range-wrap\s*\{[\s\S]*?touch-action: none;[\s\S]*?user-select: none;/,
    )
    expect(GLOBAL_STYLE).toContain('@property --chat-reasoning-max-fill-mask-position')
    expect(GLOBAL_STYLE).toContain('--chat-reasoning-max-gradient-start: #250e7a;')
    expect(GLOBAL_STYLE).toContain('--chat-reasoning-max-gradient-middle: #c775e9;')
    expect(GLOBAL_STYLE).toContain('--chat-reasoning-max-gradient-end: #7849d1;')
    expect(GLOBAL_STYLE).toMatch(
      /__max-effects--entering[\s\S]*?__max-fill[\s\S]*?animation: chat-reasoning-max-reveal 2s ease both;/,
    )
    expect(GLOBAL_STYLE).toMatch(
      /__max-effects--entering[\s\S]*?__max-track-particles[\s\S]*?animation: chat-reasoning-max-particles-enter 150ms cubic-bezier\(0\.23, 1, 0\.32, 1\) both;/,
    )
    expect(GLOBAL_STYLE).toContain(
      'animation: chat-reasoning-max-particle-burst 620ms cubic-bezier(0.25, 1, 0.5, 1) both;',
    )
    expect(GLOBAL_STYLE).toContain('opacity: var(--particle-opacity, 0.82);')
    expect(GLOBAL_STYLE).toContain('scale(var(--particle-scale, 0.82))')
    expect(GLOBAL_STYLE).toContain('opacity: 0.4;')
    expect(GLOBAL_STYLE).toContain('scale(0.52)')
    expect(GLOBAL_STYLE).toContain(
      'animation: chat-reasoning-max-burst-lifecycle 640ms linear both;',
    )
    expect(GLOBAL_STYLE).toContain('@keyframes chat-reasoning-max-burst-lifecycle')
    expect(GLOBAL_STYLE).toMatch(
      /chat-reasoning-max-effects-leave-to[\s\S]*?opacity:\s*0;[\s\S]*?translateX\(50px\)/,
    )
    expect(GLOBAL_STYLE).toMatch(
      /chat-reasoning-max-effects-leave-active[\s\S]*?opacity 220ms[\s\S]*?transform 220ms/,
    )
    expect(GLOBAL_STYLE).toContain('--reasoning-motion-duration: 150ms;')
    expect(GLOBAL_STYLE).toMatch(/opacity\s+200ms ease-in-out/)
    expect(GLOBAL_STYLE).toContain(
      'animation: chat-reasoning-thumb-settle 350ms linear both;',
    )
    expect(GLOBAL_STYLE).toContain(
      '9.1% { transform: translate(-50%, -50%) scale(1.104); }',
    )
    expect(GLOBAL_STYLE).toMatch(
      /__capability[\s\S]*?top: 54px;/,
    )
    expect(GLOBAL_STYLE).toMatch(
      /@media \(prefers-reduced-motion: reduce\)[\s\S]*?__range-fill,[\s\S]*?__range-thumb,[\s\S]*?transition: none;/,
    )
    expect(GLOBAL_STYLE).toMatch(
      /@media \(prefers-reduced-motion: reduce\)[\s\S]*?__max-fill,[\s\S]*?animation: none;/,
    )
    expect(GLOBAL_STYLE).toMatch(
      /@media \(prefers-reduced-motion: reduce\)[\s\S]*?__max-burst,[\s\S]*?animation: none;/,
    )
  })

  it('does not leave a deferred settle animation in reduced-motion mode', async () => {
    vi.spyOn(window, 'matchMedia').mockImplementation((query: string) => ({
      matches: query === '(prefers-reduced-motion: reduce)',
      media: query,
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }))
    const view = mountSettings()
    await view.get('[data-test="chat-model-settings-trigger"]').trigger('click')
    const slider = panelElement('[data-test="chat-settings-capability-slider"]') as HTMLInputElement
    const visual = panelElement('[data-test="chat-settings-capability-visual"]')

    slider.value = '3'
    slider.dispatchEvent(new Event('input', { bubbles: true }))
    await nextTick()
    expect(panelElement('[data-test="chat-settings-maximum-effects"]').classList).not.toContain(
      'chat-model-settings-popover__max-effects--entering',
    )
    expect(visual.querySelector('.chat-model-settings-popover__max-burst')).toBeNull()

    slider.dispatchEvent(pointerEvent('pointerdown', { pointerId: 21 }))
    slider.dispatchEvent(pointerEvent('pointerup', { pointerId: 21 }))
    await nextTick()

    expect(visual.classList).not.toContain(
      'chat-model-settings-popover__range-wrap--settling',
    )
  })

  it('shows the slider focus treatment only for keyboard-opened menus', async () => {
    const view = mountSettings()
    const trigger = view.get('[data-test="chat-model-settings-trigger"]')

    await trigger.trigger('pointerdown')
    await trigger.trigger('click')
    await nextTick()
    expect(panelElement('[data-test="chat-settings-capability-visual"]').classList)
      .not.toContain('chat-model-settings-popover__range-wrap--keyboard-focus')

    await trigger.trigger('click')
    await trigger.trigger('keydown', { key: 'Enter' })
    await trigger.trigger('click')
    await nextTick()
    press(panelElement('[data-test="chat-settings-advanced-toggle"]'), 'ArrowDown')
    await nextTick()
    expect(panelElement('[data-test="chat-settings-capability-visual"]').classList)
      .toContain('chat-model-settings-popover__range-wrap--keyboard-focus')
  })

  it('keeps more than six models scrollable and selectable without search or a back button', async () => {
    const modelOptions = Array.from({ length: 8 }, (_, index) => ({
      value: `model-${index}`,
      label: index === 6 ? 'GPT-5.6 Sol' : `Model ${index}`,
      description: `Description ${index}`,
      recommended: index === 6,
    }))
    const view = mountSettings({
      modelValue: 'model-6',
      modelOptions,
    })

    await view.get('[data-test="chat-model-settings-trigger"]').trigger('click')
    await expandAdvancedSettings()
    panelElement('[data-test="chat-settings-model-menu"]').click()
    await nextTick()

    const options = panelElement('[data-test="chat-settings-model-options"]')
    const modelSubmenu = panelElement('[data-test="chat-settings-submenu"]')
    const selectedOption = panelElement('[role="menuitemradio"][data-value="model-6"]')
    expect(document.body.querySelector('input[type="search"]')).toBeNull()
    expect(document.body.querySelector('[data-test="chat-settings-back"]')).toBeNull()
    expect(modelSubmenu.querySelector('.chat-model-settings-popover__header')).toBeNull()
    expect(modelSubmenu.getAttribute('aria-label')).toBe('chat.settings.model')
    expect(options.classList).toContain('chat-model-settings-popover__options')
    expect(options.querySelectorAll('[role="menuitemradio"]')).toHaveLength(8)
    expect(options.querySelectorAll('.chat-model-settings-popover__check-slot')).toHaveLength(8)
    expect(options.querySelectorAll('[data-icon="chatCheck"]')).toHaveLength(1)
    expect(options.querySelector('.chat-model-settings-popover__check-slot')?.getAttribute('aria-hidden'))
      .toBe('true')
    expect(document.activeElement).toBe(selectedOption)
    expect(document.body.textContent).not.toContain('Description 6')
    expect(document.body.textContent).not.toContain('chat.models.recommended')

    panelElement('[role="menuitemradio"][data-value="model-7"]').click()
    await nextTick()
    expect(view.emitted('update:modelValue')).toEqual([['model-7']])
    expect(document.body.querySelector('.chat-model-settings-popover')).toBeNull()
  })

  it('supports cyclic navigation, Home/End, submenu arrows, and Escape', async () => {
    const view = mountSettings()
    const trigger = view.get('[data-test="chat-model-settings-trigger"]')
    await trigger.trigger('click')

    const slider = panelElement('[data-test="chat-settings-capability-slider"]')
    const advancedToggle = panelElement('[data-test="chat-settings-advanced-toggle"]')
    expect(document.activeElement).toBe(advancedToggle)
    press(advancedToggle, 'ArrowDown')
    expect(document.activeElement).toBe(slider)
    press(slider, 'ArrowDown')
    expect(document.activeElement).toBe(slider)
    advancedToggle.click()
    await nextTick()

    const modelRow = panelElement('[data-test="chat-settings-model-menu"]')
    const reasoningRow = panelElement('[data-test="chat-settings-reasoning-menu"]')
    const expandedAdvancedToggle = panelElement('[data-test="chat-settings-advanced-toggle"]')
    expect(document.activeElement).toBe(expandedAdvancedToggle)
    modelRow.focus()
    press(modelRow, 'End')
    expect(document.activeElement).toBe(expandedAdvancedToggle)
    press(reasoningRow, 'ArrowDown')
    expect(document.activeElement).toBe(expandedAdvancedToggle)

    modelRow.focus()
    press(modelRow, 'ArrowRight')
    await nextTick()
    const selectedModelOption = panelElement('[role="menuitemradio"][data-value="gpt-5.6-sol"]')
    expect(document.activeElement).toBe(selectedModelOption)
    expect(document.body.querySelector('[data-test="chat-settings-back"]')).toBeNull()

    press(selectedModelOption, 'ArrowLeft')
    await nextTick()
    expect(document.activeElement).toBe(modelRow)
    expect(document.body.querySelector('[data-test="chat-settings-submenu"]')).toBeNull()

    press(modelRow, 'Escape')
    await nextTick()
    expect(document.body.querySelector('.chat-model-settings-popover')).toBeNull()
    expect(document.activeElement).toBe(trigger.element)
  })

  it('focuses Advanced first and exposes the slider for every selected model', async () => {
    const view = mountSettings({ modelValue: 'gpt-5' })
    await view.get('[data-test="chat-model-settings-trigger"]').trigger('click')

    expect(document.body.querySelector('[data-test="chat-settings-capability-slider"]')).not.toBeNull()
    expect(document.activeElement).toBe(panelElement('[data-test="chat-settings-advanced-toggle"]'))
  })

  it('keeps the compact root visible and closes after a submenu selection', async () => {
    Object.defineProperty(window, 'innerWidth', {
      configurable: true,
      writable: true,
      value: 390,
    })
    const view = mountSettings()

    await view.get('[data-test="chat-model-settings-trigger"]').trigger('click')
    await expandAdvancedSettings()
    panelElement('[data-test="chat-settings-model-menu"]').click()
    await nextTick()

    expect(document.body.querySelector('[data-test="chat-settings-root-panel"]')).not.toBeNull()
    expect(panelElement('[data-test="chat-settings-submenu"]').classList)
      .toContain('chat-model-settings-popover__submenu--compact')
    expect(document.body.querySelector('input[type="search"]')).toBeNull()
    expect(document.body.querySelector('[data-test="chat-settings-back"]')).toBeNull()

    panelElement('[role="menuitemradio"][data-value="gpt-5"]').click()
    await nextTick()
    expect(view.emitted('update:modelValue')).toEqual([['gpt-5']])
    expect(document.body.querySelector('.chat-model-settings-popover')).toBeNull()
  })

  it('toggles compact inline submenus with stable ARIA wiring', async () => {
    Object.defineProperty(window, 'innerWidth', {
      configurable: true,
      writable: true,
      value: 390,
    })
    const view = mountSettings()
    const trigger = view.get('[data-test="chat-model-settings-trigger"]')

    await trigger.trigger('click')
    await expandAdvancedSettings()
    const modelRow = panelElement('[data-test="chat-settings-model-menu"]')
    const reasoningRow = panelElement('[data-test="chat-settings-reasoning-menu"]')
    modelRow.click()
    await nextTick()
    const submenuId = modelRow.getAttribute('aria-controls')
    expect(submenuId).toBeTruthy()
    expect(modelRow.getAttribute('aria-expanded')).toBe('true')
    expect(reasoningRow.getAttribute('aria-expanded')).toBe('false')
    expect(panelElement('[data-test="chat-settings-submenu"]').id).toBe(submenuId)

    reasoningRow.click()
    await nextTick()
    expect(modelRow.getAttribute('aria-expanded')).toBe('false')
    expect(reasoningRow.getAttribute('aria-expanded')).toBe('true')
    expect(panelElement('[data-test="chat-settings-submenu"]').id).toBe(submenuId)
    expect(document.body.querySelector('[role="menuitemradio"][data-value="xhigh"]')).not.toBeNull()

    reasoningRow.click()
    await nextTick()
    expect(reasoningRow.getAttribute('aria-expanded')).toBe('false')
    expect(document.body.querySelector('[data-test="chat-settings-submenu"]')).toBeNull()
    expect(document.activeElement).toBe(reasoningRow)
    expect(document.body.querySelector('[data-test="chat-settings-root-panel"]')).not.toBeNull()

    expect(trigger.attributes('aria-expanded')).toBe('true')
  })

  it('folds compact submenus before closing and preserves invisible keyboard return', async () => {
    Object.defineProperty(window, 'innerWidth', {
      configurable: true,
      writable: true,
      value: 390,
    })
    const view = mountSettings()
    const trigger = view.get('[data-test="chat-model-settings-trigger"]')

    await trigger.trigger('click')
    await expandAdvancedSettings()
    const modelRow = panelElement('[data-test="chat-settings-model-menu"]')
    modelRow.click()
    await nextTick()
    const selectedOption = panelElement('[role="menuitemradio"][data-value="gpt-5.6-sol"]')
    press(selectedOption, 'ArrowLeft')
    await nextTick()
    expect(document.body.querySelector('[data-test="chat-settings-submenu"]')).toBeNull()
    expect(document.activeElement).toBe(modelRow)

    modelRow.click()
    await nextTick()
    press(panelElement('[role="menuitemradio"][data-value="gpt-5.6-sol"]'), 'Escape')
    await nextTick()
    expect(document.body.querySelector('.chat-model-settings-popover')).not.toBeNull()
    expect(document.body.querySelector('[data-test="chat-settings-submenu"]')).toBeNull()
    expect(document.activeElement).toBe(modelRow)

    press(modelRow, 'Escape')
    await nextTick()
    expect(document.body.querySelector('.chat-model-settings-popover')).toBeNull()
    expect(document.activeElement).toBe(trigger.element)

    await trigger.trigger('click')
    await expandAdvancedSettings()
    panelElement('[data-test="chat-settings-model-menu"]').click()
    await nextTick()
    await trigger.trigger('click')
    expect(document.body.querySelector('.chat-model-settings-popover')).toBeNull()

    await trigger.trigger('click')
    await expandAdvancedSettings()
    panelElement('[data-test="chat-settings-model-menu"]').click()
    await nextTick()
    document.body.dispatchEvent(new MouseEvent('pointerdown', { bubbles: true }))
    await nextTick()
    expect(document.body.querySelector('.chat-model-settings-popover')).toBeNull()
  })

  it('closes without stealing focus when pointer interaction moves outside', async () => {
    const view = mountSettings()
    const trigger = view.get('[data-test="chat-model-settings-trigger"]')
    await trigger.trigger('click')

    document.body.dispatchEvent(new MouseEvent('pointerdown', { bubbles: true }))
    await nextTick()

    expect(document.body.querySelector('.chat-model-settings-popover')).toBeNull()
    expect(document.activeElement).not.toBe(trigger.element)
  })

  it('closes without stealing focus when Tab moves outside the teleported menu', async () => {
    const outsideButton = document.createElement('button')
    outsideButton.textContent = 'Outside control'
    document.body.append(outsideButton)
    const view = mountSettings()

    await view.get('[data-test="chat-model-settings-trigger"]').trigger('click')
    await expandAdvancedSettings()
    const reasoningRow = panelElement('[data-test="chat-settings-reasoning-menu"]')
    reasoningRow.focus()
    press(reasoningRow, 'Tab')
    outsideButton.focus()
    await nextTick()

    expect(document.body.querySelector('.chat-model-settings-popover')).toBeNull()
    expect(document.activeElement).toBe(outsideButton)
  })
})
