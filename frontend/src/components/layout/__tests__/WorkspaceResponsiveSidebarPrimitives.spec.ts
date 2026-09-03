import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import WorkspaceResponsiveSidebarIcon from '../WorkspaceResponsiveSidebarIcon.vue'
import WorkspaceSidebarOverlayLayer from '../WorkspaceSidebarOverlayLayer.vue'
import WorkspaceSidebarOverlayTrigger from '../WorkspaceSidebarOverlayTrigger.vue'

const directory = dirname(fileURLToPath(import.meta.url))
const iconSource = readFileSync(resolve(directory, '../WorkspaceResponsiveSidebarIcon.vue'), 'utf8')
const triggerSource = readFileSync(resolve(directory, '../WorkspaceSidebarOverlayTrigger.vue'), 'utf8')
const layerSource = readFileSync(resolve(directory, '../WorkspaceSidebarOverlayLayer.vue'), 'utf8')
const mountedWrappers: VueWrapper[] = []

function track<T extends VueWrapper>(wrapper: T): T {
  mountedWrappers.push(wrapper)
  return wrapper
}

function installVisibleRects() {
  return vi.spyOn(HTMLElement.prototype, 'getClientRects').mockImplementation(function (this: HTMLElement) {
    return this.isConnected
      ? [new DOMRect(0, 0, 36, 36)] as unknown as DOMRectList
      : [] as unknown as DOMRectList
  })
}

afterEach(() => {
  mountedWrappers.splice(0).reverse().forEach(wrapper => wrapper.unmount())
  document.body.innerHTML = ''
  document.body.style.overflow = ''
  vi.restoreAllMocks()
})

describe('WorkspaceResponsiveSidebarIcon', () => {
  it.each([
    [
      'open',
      '0 0 20 20',
      'm11.666 12.669.135.013a.665.665 0 0 1 0 1.303l-.135.014H3.333a.665.665 0 0 1 0-1.33zm5-6.667.135.013a.665.665 0 0 1 0 1.303l-.135.014H3.333a.665.665 0 0 1 0-1.33z',
    ],
    [
      'close',
      '0 0 20 20',
      'M14.255 4.755a.7.7 0 0 1 .99.99L10.99 10l4.255 4.255.09.11a.7.7 0 0 1-.97.97l-.11-.09L10 10.99l-4.255 4.255a.7.7 0 0 1-.99-.99L9.01 10 4.755 5.745l-.09-.11a.7.7 0 0 1 .97-.97l.11.09L10 9.01z',
    ],
    [
      'search',
      '0 0 24 24',
      'M10.993 2.904a8 8 0 0 1 6.122 13.146l3.923 3.923a.75.75 0 0 1-1.06 1.06l-3.931-3.93a8 8 0 1 1-5.054-14.2m0 1.5a6.5 6.5 0 1 0 0 13 6.5 6.5 0 0 0 0-13',
    ],
  ] as const)('renders the dedicated %s glyph', (name, viewBox, path) => {
    const wrapper = track(mount(WorkspaceResponsiveSidebarIcon, { props: { name } }))

    expect(wrapper.attributes('viewBox')).toBe(viewBox)
    expect(wrapper.attributes('fill')).toBe('currentColor')
    expect(wrapper.attributes('aria-hidden')).toBe('true')
    expect(wrapper.get('path').attributes('d')).toBe(path)
    expect(wrapper.classes()).toContain(`workspace-responsive-sidebar-icon--${name}`)
  })

  it('keeps open/search at 24px and close at 20px through Workspace spacing tokens', () => {
    expect(iconSource).toMatch(
      /\.workspace-responsive-sidebar-icon--open,[\s\S]*?width: var\(--workspace-space-6\);[\s\S]*?height: var\(--workspace-space-6\);/,
    )
    expect(iconSource).toMatch(
      /\.workspace-responsive-sidebar-icon--close\s*\{[^}]*width: var\(--workspace-space-5\);[^}]*height: var\(--workspace-space-5\);/s,
    )
  })
})

describe('WorkspaceSidebarOverlayTrigger', () => {
  it('exposes one fixed accessible opener and hides it while the overlay is open', async () => {
    const wrapper = track(mount(WorkspaceSidebarOverlayTrigger, {
      props: {
        id: 'workspace-overlay-trigger',
        controls: 'workspace-overlay-panel',
        label: '打开侧边栏',
        open: false,
      },
    }))
    const button = wrapper.get('button')

    expect(button.attributes('id')).toBe('workspace-overlay-trigger')
    expect(button.attributes('type')).toBe('button')
    expect(button.attributes('aria-controls')).toBe('workspace-overlay-panel')
    expect(button.attributes('aria-expanded')).toBe('false')
    expect(button.attributes('aria-label')).toBe('打开侧边栏')
    expect(button.attributes('title')).toBe('打开侧边栏')
    expect(button.find('.workspace-responsive-sidebar-icon--open').exists()).toBe(true)

    await button.trigger('click')
    expect(wrapper.emitted('open')).toEqual([[]])

    await wrapper.setProps({ open: true })
    expect(button.attributes('aria-expanded')).toBe('true')
    expect(button.classes()).toContain('workspace-sidebar-overlay-trigger--open')
  })

  it('uses the requested fixed 8px/36px/8px geometry and token-only colors', () => {
    expect(triggerSource).toMatch(
      /\.workspace-sidebar-overlay-trigger\s*\{[^}]*position: fixed;[^}]*top: var\(--workspace-space-2\);[^}]*left: var\(--workspace-space-2\);/s,
    )
    expect(triggerSource).toMatch(
      /\.workspace-sidebar-overlay-trigger\s*\{[^}]*width: var\(--workspace-sidebar-action-size\);[^}]*height: var\(--workspace-sidebar-action-size\);[^}]*border-radius: var\(--workspace-radius-compact\);/s,
    )
    expect(triggerSource).toMatch(/@media \(prefers-reduced-motion: reduce\)/)
    expect(triggerSource).not.toMatch(/#[0-9a-f]{3,8}\b|(?:rgb|hsl)a?\(/i)
  })
})

describe('WorkspaceSidebarOverlayLayer', () => {
  it('renders only the slot when the narrow overlay mode is inactive', () => {
    const wrapper = track(mount(WorkspaceSidebarOverlayLayer, {
      props: {
        active: false,
        open: false,
        label: '侧边栏导航',
        returnFocusId: 'workspace-overlay-trigger',
      },
      slots: {
        default: '<nav data-testid="workspace-navigation-slot">Navigation</nav>',
      },
    }))

    expect(wrapper.get('[data-testid="workspace-navigation-slot"]').text()).toBe('Navigation')
    expect(document.querySelector('[data-testid="workspace-sidebar-overlay-layer"]')).toBeNull()
  })

  it('keeps the active overlay mounted but inert, hidden, and unlocked while closed', async () => {
    track(mount(WorkspaceSidebarOverlayLayer, {
      attachTo: document.body,
      props: {
        active: true,
        open: false,
        label: '侧边栏导航',
        returnFocusId: 'workspace-overlay-trigger',
      },
      slots: {
        default: '<nav data-testid="workspace-navigation-slot">Navigation</nav>',
      },
    }))
    await flushPromises()

    const layer = document.querySelector<HTMLElement>('[data-testid="workspace-sidebar-overlay-layer"]')
    expect(layer).not.toBeNull()
    expect(layer?.getAttribute('aria-hidden')).toBe('true')
    expect(layer?.hasAttribute('inert')).toBe(true)
    expect(layer?.classList.contains('workspace-sidebar-overlay-layer--open')).toBe(false)
    expect(layer?.querySelector('[data-testid="workspace-navigation-slot"]')).not.toBeNull()
    expect(document.body.style.overflow).toBe('')
  })

  it('locks scrolling, focuses close, closes from Escape/scrim, and restores the trigger', async () => {
    const trigger = document.createElement('button')
    trigger.id = 'workspace-overlay-trigger'
    document.body.append(trigger)
    trigger.focus()

    const wrapper = track(mount(WorkspaceSidebarOverlayLayer, {
      attachTo: document.body,
      props: {
        active: true,
        open: true,
        label: '关闭侧边栏',
        returnFocusId: trigger.id,
      },
      slots: {
        default: `
          <button data-workspace-sidebar-overlay-close>Close</button>
          <button data-testid="panel-action">Panel action</button>
        `,
      },
    }))
    await flushPromises()

    const close = document.querySelector<HTMLButtonElement>(
      '[data-workspace-sidebar-overlay-close]',
    )
    const layer = document.querySelector<HTMLElement>('[data-testid="workspace-sidebar-overlay-layer"]')
    expect(document.body.style.overflow).toBe('hidden')
    expect(document.activeElement).toBe(close)
    expect(layer?.getAttribute('aria-hidden')).toBeNull()
    expect(layer?.hasAttribute('inert')).toBe(false)

    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    expect(wrapper.emitted('close')).toEqual([[]])

    document.querySelector<HTMLButtonElement>('.workspace-sidebar-overlay-layer__scrim')?.click()
    expect(wrapper.emitted('close')).toEqual([[], []])

    await wrapper.setProps({ open: false })
    await flushPromises()
    expect(document.body.style.overflow).toBe('')
    expect(document.activeElement).toBe(trigger)
  })

  it('traps forward and reverse Tab focus inside the topmost open panel', async () => {
    installVisibleRects()
    const wrapper = track(mount(WorkspaceSidebarOverlayLayer, {
      attachTo: document.body,
      props: {
        active: true,
        open: true,
        label: '关闭侧边栏',
        returnFocusId: 'workspace-overlay-trigger',
      },
      slots: {
        default: `
          <button data-workspace-sidebar-overlay-close data-testid="panel-first">First</button>
          <button data-testid="panel-last">Last</button>
        `,
      },
    }))
    await flushPromises()

    const close = document.querySelector<HTMLButtonElement>('[data-testid="panel-first"]')!
    const last = document.querySelector<HTMLButtonElement>('[data-testid="panel-last"]')!

    last.focus()
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', bubbles: true }))
    expect(document.activeElement).toBe(close)

    close.focus()
    document.dispatchEvent(new KeyboardEvent('keydown', {
      key: 'Tab',
      shiftKey: true,
      bubbles: true,
    }))
    expect(document.activeElement).toBe(last)
    expect(wrapper.emitted('close')).toBeUndefined()
  })

  it('uses the shared modal stack so only the topmost layer handles Escape', async () => {
    const first = track(mount(WorkspaceSidebarOverlayLayer, {
      attachTo: document.body,
      props: {
        active: true,
        open: true,
        label: '第一层侧边栏',
        returnFocusId: 'first-trigger',
      },
      slots: {
        default: '<button data-workspace-sidebar-overlay-close>Close first</button>',
      },
    }))
    const second = track(mount(WorkspaceSidebarOverlayLayer, {
      attachTo: document.body,
      props: {
        active: true,
        open: true,
        label: '第二层侧边栏',
        returnFocusId: 'second-trigger',
      },
      slots: {
        default: '<button data-workspace-sidebar-overlay-close>Close second</button>',
      },
    }))
    await flushPromises()

    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))

    expect(first.emitted('close')).toBeUndefined()
    expect(second.emitted('close')).toEqual([[]])
  })

  it('uses a fixed 260px z-50 Workspace surface with motion-safe slide states', () => {
    expect(layerSource).toMatch(
      /\.workspace-sidebar-overlay-layer\s*\{[^}]*position: fixed;[^}]*inset: 0;[^}]*z-index: var\(--workspace-layer-sidebar-overlay\);/s,
    )
    expect(layerSource).toMatch(
      /\.workspace-sidebar-overlay-layer__panel\s*\{[^}]*width: var\(--workspace-sidebar-width\);[^}]*background: var\(--workspace-sidebar-surface\);[^}]*box-shadow: var\(--workspace-sidebar-overlay-shadow\);[^}]*transform: translateX\(-100%\);[^}]*transition: transform 190ms ease-out;/s,
    )
    expect(layerSource).toMatch(
      /\.workspace-sidebar-overlay-layer--open \.workspace-sidebar-overlay-layer__panel\s*\{[^}]*transform: translateX\(0\);/s,
    )
    expect(layerSource).toMatch(
      /\.workspace-sidebar-overlay-layer__content\s*\{[^}]*overflow-x: hidden;[^}]*overflow-y: auto;/s,
    )
    expect(layerSource).toMatch(/@media \(prefers-reduced-motion: reduce\)/)
    expect(layerSource).not.toMatch(/#[0-9a-f]{3,8}\b|(?:rgb|hsl)a?\(/i)
  })
})
