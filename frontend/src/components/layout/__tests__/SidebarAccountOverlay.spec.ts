import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { mount, type VueWrapper } from '@vue/test-utils'
import { nextTick } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  registerModalLayer,
  unregisterModalLayer,
} from '@/utils/modalStack'
import SidebarAccountOverlay from '../SidebarAccountOverlay.vue'
import type { AccountPanelSummary } from '../accountPanelTypes'

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
        CreditAmount: {
          props: ['value', 'iconSize', 'label'],
          template: '<span data-testid="credit-amount" :data-value="value" :aria-label="label">{{ value }}</span>',
        },
        Icon: {
          props: ['name'],
          template: '<span :data-icon="name" />',
        },
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
  it('renders labeled available and frozen balances as Snow credits', async () => {
    mountOverlay(false, {
      summary: {
        ...summary,
        frozenBalance: 3.5,
        formattedFrozenBalance: '3.50',
      },
    })
    await nextTick()

    const panel = document.body.querySelector<HTMLElement>('[data-testid="sidebar-account-panel"]')
    const amounts = Array.from(
      document.body.querySelectorAll<HTMLElement>('[data-testid="credit-amount"]'),
    )
    expect(panel?.textContent).toContain('accountDock.balanceShort')
    expect(amounts.map((amount) => amount.dataset.value)).toEqual(['12.50', '3.50'])
    expect(amounts.map((amount) => amount.getAttribute('aria-label'))).toEqual([
      'accountDock.availableBalance 12.50',
      'accountDock.frozenBalance 3.50',
    ])
  })

  it('uses compact desktop measurements while retaining mobile touch targets', () => {
    expect(componentSource).toContain('width: 248px;')
    expect(componentSource).toContain('padding: 6px 0;')
    expect(componentSource).toContain('border-radius: 16px;')
    expect(componentSource).toContain('min-height: 36px;')
    expect(componentSource).toMatch(
      /@media \(max-width: 1023px\)[\s\S]*\.account-panel__row \{[\s\S]*min-height: 44px;/,
    )
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

  it('closes a desktop panel only when pointerdown occurs outside the panel and anchor', async () => {
    const { anchor, wrapper } = mountOverlay()
    await nextTick()
    const panel = document.body.querySelector<HTMLElement>('[data-testid="sidebar-account-panel"]')

    panel?.dispatchEvent(new MouseEvent('pointerdown', { bubbles: true }))
    anchor.dispatchEvent(new MouseEvent('pointerdown', { bubbles: true }))
    expect(wrapper.emitted('close')).toBeUndefined()

    document.body.dispatchEvent(new MouseEvent('pointerdown', { bubbles: true }))
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

  it('only keeps profile, preferences, and logout in the regular account menu', async () => {
    const { wrapper } = mountOverlay()
    await nextTick()

    const panel = document.body.querySelector<HTMLElement>('[data-testid="sidebar-account-panel"]')
    const profile = panel?.querySelector<HTMLButtonElement>('[data-testid="account-open-profile"]')
    const preferences = panel?.querySelector<HTMLButtonElement>(
      '[data-testid="account-open-preferences"]',
    )

    expect(profile?.textContent).toContain('accountDock.personalProfile')
    expect(preferences?.textContent).toContain('accountDock.personalPreferences')
    expect(panel?.querySelector('[data-testid="account-admin-guide"]')).toBeNull()
    expect(panel?.querySelector('[data-testid="account-logout"]')).not.toBeNull()
    expect(panel?.querySelector('a')).toBeNull()
    expect(panel?.querySelector('[data-testid="account-resources-toggle"]')).toBeNull()
    expect(panel?.querySelector('[data-testid="account-subscriptions-link"]')).toBeNull()

    profile?.click()
    preferences?.click()
    expect(wrapper.emitted('open-settings')).toEqual([['account'], ['general']])
  })

  it('only exposes the administrator guide when requested and emits replay directly', async () => {
    const { wrapper } = mountOverlay(false, { showOnboarding: true })
    await nextTick()

    const guide = document.body.querySelector<HTMLButtonElement>(
      '[data-testid="account-admin-guide"]',
    )
    expect(guide?.textContent).toContain('accountDock.adminGuide')

    guide?.click()
    expect(wrapper.emitted('replay')).toEqual([[]])
  })

  it('keeps every visible mobile action inside the focus loop', async () => {
    vi.spyOn(HTMLElement.prototype, 'getClientRects').mockImplementation(
      () => ([{} as DOMRect] as unknown as DOMRectList),
    )

    mountOverlay(true)
    await nextTick()
    await nextTick()

    const close = document.body.querySelector<HTMLButtonElement>('.account-panel__close')
    const logout = document.body.querySelector<HTMLButtonElement>('[data-testid="account-logout"]')
    expect(close).not.toBeNull()
    expect(logout).not.toBeNull()

    close?.focus()
    document.dispatchEvent(new KeyboardEvent('keydown', {
      key: 'Tab',
      shiftKey: true,
      bubbles: true,
      cancelable: true,
    }))

    expect(document.activeElement).toBe(logout)
  })
})
