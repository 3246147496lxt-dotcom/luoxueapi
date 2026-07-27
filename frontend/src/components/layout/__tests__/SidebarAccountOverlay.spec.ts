import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { nextTick } from 'vue'
import {
  registerModalLayer,
  unregisterModalLayer,
} from '@/utils/modalStack'
import SidebarAccountOverlay from '../SidebarAccountOverlay.vue'
import type {
  AccountPanelLink,
  AccountPanelSummary,
  AccountResourceLink,
} from '../accountPanelTypes'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../SidebarAccountOverlay.vue')
const componentSource = readFileSync(componentPath, 'utf8')

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

const summary: AccountPanelSummary = {
  displayName: 'Riley Quinn',
  email: 'riley@example.com',
  initials: 'RQ',
  avatarUrl: '',
  frozenBalance: 0,
  formattedAvailableBalance: '12.50',
  formattedFrozenBalance: '0.00',
  activeSubscriptionCount: 0,
  subscriptionsLoaded: true,
}

const defaultProps = {
  open: true,
  anchorElement: null as HTMLElement | null,
  summary,
  purchaseLink: {
    id: 'wallet',
    label: 'Add funds',
    to: '/purchase',
    icon: 'wallet',
  } as AccountPanelLink,
  resourceLinks: [
    {
      id: 'home',
      label: 'Home',
      href: '/home',
      icon: 'destinationHome',
    },
  ] as AccountResourceLink[],
  showOnboarding: false,
}

const originalMatchMedia = window.matchMedia
const originalInnerWidth = window.innerWidth
const mountedWrappers = new Set<VueWrapper>()

function installViewport(mobile: boolean) {
  Object.defineProperty(window, 'innerWidth', {
    configurable: true,
    value: mobile ? 390 : 1440,
  })
  Object.defineProperty(window, 'matchMedia', {
    configurable: true,
    value: vi.fn((query: string) => ({
      matches: query === '(max-width: 1023px)' ? mobile : false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  })
}

function mountOverlay(
  mobile = false,
  propOverrides: Partial<typeof defaultProps> = {},
) {
  installViewport(mobile)

  const shell = document.createElement('main')
  shell.className = 'app-layout'
  shell.inert = false
  const anchor = document.createElement('button')
  anchor.type = 'button'
  const mountPoint = document.createElement('div')
  shell.append(anchor, mountPoint)
  document.body.append(shell)

  const wrapper = mount(SidebarAccountOverlay, {
    attachTo: mountPoint,
    props: {
      ...defaultProps,
      ...propOverrides,
      anchorElement: anchor,
    },
    global: {
      stubs: {
        AnnouncementBell: true,
        CreditAmount: {
          props: ['value', 'iconSize', 'label'],
          template: '<span data-testid="credit-amount" :data-value="value" :aria-label="label">{{ value }}</span>',
        },
        Icon: true,
        RouterLink: true,
      },
    },
  })
  mountedWrappers.add(wrapper)

  return { anchor, shell, wrapper }
}

afterEach(() => {
  for (const wrapper of mountedWrappers) {
    wrapper.unmount()
  }
  mountedWrappers.clear()
  document.body.innerHTML = ''
  document.body.style.overflow = ''
  Object.defineProperty(window, 'innerWidth', {
    configurable: true,
    value: originalInnerWidth,
  })
  Object.defineProperty(window, 'matchMedia', {
    configurable: true,
    value: originalMatchMedia,
  })
  vi.clearAllMocks()
  vi.restoreAllMocks()
})

describe('SidebarAccountOverlay', () => {
  it('renders available and frozen balances as Snow credits', async () => {
    mountOverlay(false, {
      summary: {
        ...summary,
        frozenBalance: 3.5,
        formattedFrozenBalance: '3.50',
      },
    })
    await nextTick()

    const amounts = Array.from(
      document.body.querySelectorAll<HTMLElement>('[data-testid="credit-amount"]'),
    )
    expect(amounts.map((amount) => amount.dataset.value)).toEqual(['12.50', '3.50'])
    expect(amounts.map((amount) => amount.getAttribute('aria-label'))).toEqual([
      'accountDock.availableBalance 12.50',
      'accountDock.frozenBalance 3.50',
    ])
  })

  it('uses the compact desktop menu measurements while retaining mobile touch targets', () => {
    expect(componentSource).toContain('width: 248px;')
    expect(componentSource).toContain('padding: 6px 0;')
    expect(componentSource).toContain('border-radius: 16px;')
    expect(componentSource).toContain('min-height: 36px;')
    expect(componentSource).toMatch(
      /@media \(max-width: 1023px\)[\s\S]*\.account-panel__row,[\s\S]*min-height: 44px;/,
    )
    expect(componentSource).not.toContain('account-panel__summary')
  })

  it('teleports an open desktop panel to body and exposes it as a dialog', async () => {
    const { shell, wrapper } = mountOverlay()
    await nextTick()

    const panel = document.body.querySelector<HTMLElement>('[data-testid="sidebar-account-panel"]')

    expect(panel).not.toBeNull()
    expect(panel?.getAttribute('role')).toBe('dialog')
    expect(panel?.getAttribute('aria-modal')).toBeNull()
    expect(panel?.closest('.app-layout')).toBeNull()
    expect(shell.querySelector('[data-testid="sidebar-account-panel"]')).toBeNull()
    expect(wrapper.html()).not.toContain('data-testid="sidebar-account-panel"')
  })

  it('positions the desktop panel before focusing it without scrolling the page', async () => {
    const nativeFocus = HTMLElement.prototype.focus
    let styleAtFocus: Pick<CSSStyleDeclaration, 'left' | 'bottom' | 'maxHeight'> | null = null
    const focus = vi.spyOn(HTMLElement.prototype, 'focus').mockImplementation(function focus(
      this: HTMLElement,
      options?: FocusOptions,
    ) {
      if (this.dataset.testid === 'sidebar-account-panel') {
        styleAtFocus = {
          left: this.style.left,
          bottom: this.style.bottom,
          maxHeight: this.style.maxHeight,
        }
      }
      nativeFocus.call(this, options)
    })

    mountOverlay()
    await nextTick()
    await nextTick()
    await nextTick()

    expect(styleAtFocus).toEqual({
      left: '8px',
      bottom: `${window.innerHeight + 8}px`,
      maxHeight: '0px',
    })
    expect(focus).toHaveBeenCalledWith({ preventScroll: true })
  })

  it('closes an open desktop panel on Escape', async () => {
    const { wrapper } = mountOverlay()
    await nextTick()

    document.dispatchEvent(new KeyboardEvent('keydown', {
      key: 'Escape',
      bubbles: true,
      cancelable: true,
    }))

    expect(wrapper.emitted('close')).toEqual([[true]])
  })

  it('lets only the top modal layer respond to Escape', async () => {
    const { wrapper } = mountOverlay()
    const childToken = Symbol('nested-account-child')
    await nextTick()

    registerModalLayer(childToken)
    try {
      document.dispatchEvent(new KeyboardEvent('keydown', {
        key: 'Escape',
        bubbles: true,
        cancelable: true,
      }))
      expect(wrapper.emitted('close')).toBeUndefined()
    } finally {
      unregisterModalLayer(childToken)
    }

    document.dispatchEvent(new KeyboardEvent('keydown', {
      key: 'Escape',
      bubbles: true,
      cancelable: true,
    }))
    expect(wrapper.emitted('close')).toEqual([[true]])
  })

  it('closes an open desktop panel only when pointerdown occurs outside the panel and anchor', async () => {
    const { anchor, wrapper } = mountOverlay()
    await nextTick()
    const panel = document.body.querySelector<HTMLElement>('[data-testid="sidebar-account-panel"]')

    panel?.dispatchEvent(new MouseEvent('pointerdown', { bubbles: true }))
    anchor.dispatchEvent(new MouseEvent('pointerdown', { bubbles: true }))
    expect(wrapper.emitted('close')).toBeUndefined()

    document.body.dispatchEvent(new MouseEvent('pointerdown', { bubbles: true }))
    expect(wrapper.emitted('close')).toEqual([[false]])
  })

  it('closes for purchase navigation without restoring focus', async () => {
    const { wrapper } = mountOverlay()
    await nextTick()

    const purchase = document.body.querySelector<HTMLElement>('router-link-stub')
    expect(purchase).not.toBeNull()

    purchase?.click()
    expect(wrapper.emitted('close')).toEqual([[false]])
  })

  it('renders a modal bottom sheet on mobile and restores the background lock on unmount', async () => {
    const { shell, wrapper } = mountOverlay(true)
    await nextTick()
    await nextTick()

    const panel = document.body.querySelector<HTMLElement>('[data-testid="sidebar-account-panel"]')
    const backdrop = document.body.querySelector<HTMLButtonElement>('.account-panel-backdrop')

    expect(panel?.getAttribute('aria-modal')).toBe('true')
    expect(panel?.classList.contains('account-panel--mobile')).toBe(true)
    expect(backdrop).not.toBeNull()
    expect(shell.getAttribute('aria-hidden')).toBe('true')
    expect(shell.inert).toBe(true)
    expect(document.body.style.overflow).toBe('hidden')

    backdrop?.click()
    expect(wrapper.emitted('close')).toEqual([[true]])

    wrapper.unmount()
    mountedWrappers.delete(wrapper)

    expect(shell.getAttribute('aria-hidden')).toBeNull()
    expect(shell.inert).toBe(false)
    expect(document.body.style.overflow).toBe('')
  })

  it('keeps preferences and deep account pages out of the first level', async () => {
    const { wrapper } = mountOverlay()
    await nextTick()

    const panel = document.body.querySelector<HTMLElement>('[data-testid="sidebar-account-panel"]')
    const firstLevelLinks = panel?.querySelectorAll('a') ?? []

    expect(document.body.querySelector('[data-testid="account-theme-toggle"]')).toBeNull()
    expect(document.body.querySelector('.account-panel__locale-switcher')).toBeNull()
    expect(Array.from(firstLevelLinks).map((link) => link.getAttribute('href'))).not.toEqual(
      expect.arrayContaining(['/profile', '/subscriptions', '/orders']),
    )
    expect(wrapper.emitted('toggle-theme')).toBeUndefined()
  })

  it('keeps hidden disclosure and section controls out of the mobile focus loop', async () => {
    const hasHiddenAncestor = (element: HTMLElement | null): boolean => {
      if (!element) return false
      if (
        element.hidden
        || element.inert
        || element.style.display === 'none'
        || element.style.visibility === 'hidden'
      ) {
        return true
      }
      return hasHiddenAncestor(element.parentElement)
    }

    vi.spyOn(HTMLElement.prototype, 'getClientRects').mockImplementation(function getClientRects(
      this: HTMLElement,
    ) {
      if (hasHiddenAncestor(this)) return [] as unknown as DOMRectList
      return [{} as DOMRect] as unknown as DOMRectList
    })

    mountOverlay(true)
    await nextTick()
    await nextTick()

    const close = document.body.querySelector<HTMLButtonElement>('.account-panel__close')
    const resourcesButton = document.body.querySelector<HTMLButtonElement>(
      '[data-testid="account-resources-toggle"]',
    )
    const dangerSection = document.body.querySelector<HTMLElement>(
      '.account-panel__section--danger',
    )

    expect(close).not.toBeNull()
    expect(resourcesButton).not.toBeNull()
    expect(dangerSection).not.toBeNull()

    if (dangerSection) dangerSection.style.display = 'none'
    close?.focus()
    document.dispatchEvent(new KeyboardEvent('keydown', {
      key: 'Tab',
      shiftKey: true,
      bubbles: true,
      cancelable: true,
    }))

    expect(document.activeElement).toBe(resourcesButton)
  })

  it('emits the settings action and progressively reveals the remaining help resources', async () => {
    const { wrapper } = mountOverlay()
    await nextTick()

    const settingsButton = document.body.querySelector<HTMLButtonElement>(
      '[data-testid="account-open-settings"]',
    )
    const resourcesButton = document.body.querySelector<HTMLButtonElement>(
      '[data-testid="account-resources-toggle"]',
    )
    const resources = document.body.querySelector<HTMLElement>('#account-resource-links')

    expect(settingsButton).not.toBeNull()
    expect(resourcesButton?.getAttribute('aria-expanded')).toBe('false')
    expect(resources?.style.display).toBe('none')

    settingsButton?.click()
    expect(wrapper.emitted('open-settings')).toEqual([[]])

    resourcesButton?.click()
    await nextTick()

    expect(resourcesButton?.getAttribute('aria-expanded')).toBe('true')
    expect(resources?.style.display).not.toBe('none')
    expect(Array.from(resources?.querySelectorAll('a') ?? []).map((link) => link.getAttribute('href')))
      .toEqual(['/home'])
    expect(resources?.textContent).not.toContain('Documentation')

    resources?.querySelector<HTMLAnchorElement>('a')?.click()
    expect(wrapper.emitted('close')).toEqual([[false]])
  })

  it('keeps the administrator guide inside the resources disclosure', async () => {
    mountOverlay(false, { showOnboarding: true })
    await nextTick()

    const resourcesButton = document.body.querySelector<HTMLButtonElement>(
      '[data-testid="account-resources-toggle"]',
    )
    const resources = document.body.querySelector<HTMLElement>('#account-resource-links')

    expect(resources?.textContent).toContain('accountDock.adminGuide')
    expect(resources?.style.display).toBe('none')

    resourcesButton?.click()
    await nextTick()

    expect(resources?.style.display).not.toBe('none')
  })
})
