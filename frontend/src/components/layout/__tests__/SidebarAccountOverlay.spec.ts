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
const workspaceTokens = readFileSync(
  resolve(dirname(fileURLToPath(import.meta.url)), '../../../styles/luoxue-clay-tokens.css'),
  'utf8',
)
const frameSource = readFileSync(
  resolve(dirname(fileURLToPath(import.meta.url)), '../WorkspaceSidebarFrame.vue'),
  'utf8',
)

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

type OverlayProps = {
  open: boolean
  anchorElement: HTMLElement | null
  summary: AccountPanelSummary
  showOnboarding: boolean
  context: 'work' | 'chat'
  variant: 'personal' | 'admin'
  appearance?: 'personal' | 'admin'
  planLabel: string
  helpHref: string
  workspaceTarget: { href: string; label: string } | null
}

const defaultProps: OverlayProps = {
  open: true,
  anchorElement: null,
  summary,
  showOnboarding: false,
  context: 'work',
  variant: 'personal',
  planLabel: 'Pro',
  helpHref: '/tutorial-docs/',
  workspaceTarget: null,
}

const originalMatchMedia = window.matchMedia
const originalInnerWidth = window.innerWidth
const mountedWrappers = new Set<VueWrapper>()
const viewportListeners = new Map<string, Set<(event: MediaQueryListEvent) => void>>()

function installViewport(mobile: boolean) {
  viewportListeners.clear()
  Object.defineProperty(window, 'innerWidth', {
    configurable: true,
    value: mobile ? 390 : 1440,
  })
  Object.defineProperty(window, 'matchMedia', {
    configurable: true,
    value: vi.fn((query: string) => ({
      matches: ['(max-width: 1023px)', '(max-width: 767px)'].includes(query)
        ? mobile
        : false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn((type: string, listener: (event: MediaQueryListEvent) => void) => {
        if (type !== 'change') return
        const listeners = viewportListeners.get(query) ?? new Set()
        listeners.add(listener)
        viewportListeners.set(query, listeners)
      }),
      removeEventListener: vi.fn((type: string, listener: (event: MediaQueryListEvent) => void) => {
        if (type === 'change') viewportListeners.get(query)?.delete(listener)
      }),
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
        AccountMenuIcon: {
          props: ['name', 'size'],
          template: '<span :data-account-menu-icon="name" :data-size="size" />',
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
  viewportListeners.clear()
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
  it('shows only the bound identity and plan in the personal account header', async () => {
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
    expect(panel?.textContent).toContain('Riley Quinn')
    expect(panel?.textContent).toContain('Pro')
    expect(panel?.textContent).not.toContain('accountDock.balanceShort')
    expect(panel?.textContent).not.toContain('12.50')
    expect(amounts).toHaveLength(0)
  })

  it('uses the same plan-first identity in Chat without exposing email or billing', async () => {
    mountOverlay(false, { context: 'chat' })
    await nextTick()

    const panel = document.body.querySelector<HTMLElement>('[data-testid="sidebar-account-panel"]')
    expect(panel?.textContent).toContain('Pro')
    expect(panel?.textContent).not.toContain('riley@example.com')
    expect(panel?.textContent).not.toContain('accountDock.balanceShort')
    expect(panel?.querySelector('[data-testid="credit-amount"]')).toBeNull()
  })

  it('preserves the existing administrator balance summary', async () => {
    mountOverlay(false, {
      variant: 'admin',
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
  })

  it('applies the personal visual language without removing administrator actions', async () => {
    mountOverlay(false, {
      variant: 'admin',
      appearance: 'personal',
      showOnboarding: true,
      workspaceTarget: {
        href: '/dashboard',
        label: 'nav.switchToPersonalWorkspace',
      },
      summary: {
        ...summary,
        frozenBalance: 3.5,
        formattedFrozenBalance: '3.50',
      },
    })
    await nextTick()

    const panel = document.body.querySelector<HTMLElement>('[data-testid="sidebar-account-panel"]')
    const actionIds = Array.from(
      panel?.querySelectorAll<HTMLElement>('[data-testid]') ?? [],
    ).map(element => element.dataset.testid)
    const icons = Array.from(
      panel?.querySelectorAll<HTMLElement>('[data-account-menu-icon]') ?? [],
    ).map(icon => icon.dataset.accountMenuIcon)

    expect(panel?.classList.contains('account-panel--personal')).toBe(true)
    expect(panel?.textContent).toContain('Riley Quinn')
    expect(panel?.textContent).toContain('Pro')
    expect(panel?.textContent).not.toContain('accountDock.balanceShort')
    expect(panel?.querySelectorAll('[data-testid="credit-amount"]')).toHaveLength(0)
    expect(panel?.querySelectorAll('.account-panel__divider')).toHaveLength(2)
    expect(actionIds).toEqual([
      'account-open-profile',
      'account-open-preferences',
      'account-admin-guide',
      'account-switch-workspace',
      'account-logout',
    ])
    expect(icons).toEqual(['chevronRight', 'avatar', 'settings', 'help', 'exit'])
    expect(panel?.querySelector('[data-testid="account-switch-workspace"]')?.getAttribute('href'))
      .toBe('/dashboard')
    expect(panel?.textContent).toContain('nav.switchToPersonalWorkspace')
  })

  it('offers administrators an explicit cross-workspace return path', async () => {
    mountOverlay(false, {
      variant: 'admin',
      workspaceTarget: {
        href: '/admin/dashboard',
        label: 'nav.switchToAdminWorkspace',
      },
    })
    await nextTick()

    const link = document.body.querySelector<HTMLAnchorElement>(
      '[data-testid="account-switch-workspace"]',
    )
    expect(link?.getAttribute('href')).toBe('/admin/dashboard')
    expect(link?.textContent).toContain('nav.switchToAdminWorkspace')
  })

  it('keeps the measured ChatGPT fixed-card geometry at every viewport size', () => {
    const personalRule = componentSource.match(
      /\.account-panel--personal\s*\{[^}]*\}/,
    )?.[0]
    const identityRule = componentSource.match(
      /\.account-panel--personal \.account-panel__identity\s*\{[^}]*\}/,
    )?.[0]
    const avatarRule = componentSource.match(
      /\.account-panel--personal \.account-panel__avatar\s*\{[^}]*\}/,
    )?.[0]
    const dividerRule = componentSource.match(
      /\.account-panel__divider\s*\{[^}]*\}/,
    )?.[0]
    const rowRule = componentSource.match(
      /\.account-panel--personal \.account-panel__row\s*\{[^}]*\}/,
    )?.[0]
    const reservedRowRule = componentSource.match(
      /\.account-panel--personal \.account-panel__row--reserved\s*\{[^}]*\}/,
    )?.[0]

    expect(componentSource).toContain('width: var(--workspace-popover-width);')
    expect(personalRule).toContain('padding: var(--workspace-space-2-5) 0;')
    expect(personalRule).toContain('border-radius: var(--workspace-radius-popover);')
    expect(personalRule).not.toContain('font-family:')
    expect(personalRule).toContain('font-weight: var(--workspace-type-body-weight);')
    expect(personalRule).toContain('letter-spacing: normal;')
    expect(identityRule).toContain('min-height: var(--workspace-popover-identity-height);')
    expect(identityRule).toContain('gap: var(--workspace-space-2);')
    expect(identityRule).toContain('margin: 0 var(--workspace-space-2-5);')
    expect(identityRule).toContain(
      'padding: var(--workspace-space-1-5) var(--workspace-space-2-5);',
    )
    expect(identityRule).toContain('border-radius: var(--workspace-radius-input);')
    expect(avatarRule).toContain('width: var(--workspace-avatar-size-sm);')
    expect(avatarRule).toContain('height: var(--workspace-avatar-size-sm);')
    expect(dividerRule).toContain('width: auto;')
    expect(dividerRule).toContain('height: var(--workspace-space-0-25);')
    expect(dividerRule).toContain(
      'margin: var(--workspace-space-2) var(--workspace-space-4);',
    )
    expect(rowRule).toContain('width: calc(100% - var(--workspace-space-5));')
    expect(rowRule).toContain('min-height: var(--workspace-menu-row-height);')
    expect(rowRule).toContain('gap: var(--workspace-space-1-5);')
    expect(rowRule).toContain('margin: 0 var(--workspace-space-2-5);')
    expect(rowRule).toContain(
      'padding: var(--workspace-space-1-5) var(--workspace-space-2-5);',
    )
    expect(rowRule).toContain('border-radius: var(--workspace-radius-input);')
    expect(rowRule).toContain('font-size: var(--workspace-type-navigation-size);')
    expect(rowRule).toContain('font-weight: var(--workspace-type-navigation-weight);')
    expect(rowRule).toContain('line-height: 1.25rem;')
    expect(rowRule).toContain('letter-spacing: normal;')
    expect(reservedRowRule).toContain('padding-right: var(--workspace-space-8);')
    expect(workspaceTokens).toContain('--workspace-popover-width: 248px;')
    expect(workspaceTokens).toContain('--workspace-avatar-size-sm: var(--workspace-space-6);')
    expect(workspaceTokens).toContain('--workspace-menu-row-height: 36px;')
    expect(workspaceTokens).toContain('--workspace-popover-identity-height: 51px;')
    expect(workspaceTokens).toContain('--workspace-radius-popover: 20px;')
    expect(workspaceTokens).toContain('--workspace-radius-input: 12px;')
    expect(workspaceTokens).toContain('--workspace-space-1-5: 6px;')
    expect(workspaceTokens).toContain('--workspace-space-2: 8px;')
    expect(workspaceTokens).toContain('--workspace-space-2-5: 10px;')
    expect(workspaceTokens).toContain('--workspace-space-4: 16px;')
    expect(workspaceTokens).toContain('--workspace-space-5: 20px;')
    expect(workspaceTokens).toContain('--workspace-space-8: 32px;')
    expect(workspaceTokens).toContain(
      '--workspace-font-ui: Inter, "PingFang SC", "Microsoft YaHei", sans-serif;',
    )
    expect(workspaceTokens).not.toContain('--workspace-font-native:')
    expect(componentSource).toContain(
      "const usesMobileSheet = computed(() => isMobile.value && props.variant === 'admin')",
    )
    expect(componentSource).not.toMatch(
      /@media \(max-width: 1023px\)[\s\S]*\.account-panel--personal \.account-panel__row \{[\s\S]*min-height: 44px;/,
    )
  })

  it('uses the canonical dark popup hierarchy for the complete personal popover', () => {
    const personalRule = componentSource.match(
      /\.account-panel--personal\s*\{[^}]*\}/,
    )?.[0]
    const personalDarkRule = componentSource.match(
      /:global\(html\.dark \.account-panel\.account-panel--personal\)\s*\{[^}]*\}/,
    )?.[0]

    expect(personalDarkRule).toContain('border-width: 0;')
    expect(personalRule).toContain('border: 1px solid var(--workspace-popover-border);')
    expect(personalRule).toContain('color: var(--workspace-popover-text);')
    expect(personalRule).toContain('background: var(--workspace-popover-surface);')
    expect(personalRule).toContain('box-shadow: var(--workspace-popover-shadow);')
    expect(workspaceTokens).toContain(
      '--workspace-dark-popover-surface: var(--workspace-dark-popup-surface);',
    )
    expect(workspaceTokens).toContain(
      '--workspace-dark-popover-border: var(--workspace-dark-border);',
    )
    expect(workspaceTokens).toContain(
      '--workspace-dark-popover-divider: var(--workspace-dark-divider);',
    )
    expect(workspaceTokens).toContain('--workspace-dark-popover-text: #ececec;')
    expect(workspaceTokens).toContain('--workspace-dark-popover-text-secondary: #b4b4b4;')
    expect(workspaceTokens).toContain('--workspace-dark-active: #212121;')
    expect(workspaceTokens).toContain(
      '--workspace-dark-shadow-popover: inset 0 0 1px rgb(255 255 255 / 0.2);',
    )
    expect(workspaceTokens).toContain(
      '--workspace-popover-surface: var(--workspace-popup-surface);',
    )
    expect(workspaceTokens).toContain(
      '--workspace-popover-hover: var(--workspace-hover);',
    )
    expect(componentSource).toContain(
      'background: var(--workspace-popover-identity-surface);',
    )
  })

  it('teleports an open desktop panel to body and exposes it as a dialog', async () => {
    const { shell, wrapper } = mountOverlay()
    await nextTick()

    const panel = document.body.querySelector<HTMLElement>('[data-testid="sidebar-account-panel"]')
    const overlay = document.body.querySelector<HTMLElement>('[data-testid="sidebar-account-overlay"]')

    expect(panel).not.toBeNull()
    expect(panel?.getAttribute('role')).toBe('dialog')
    expect(panel?.getAttribute('aria-modal')).toBeNull()
    expect(panel?.closest('.app-layout')).toBeNull()
    expect(overlay?.parentElement).toBe(document.body)
    expect(shell.querySelector('[data-testid="sidebar-account-panel"]')).toBeNull()
    expect(wrapper.html()).not.toContain('data-testid="sidebar-account-panel"')
  })

  it('keeps the portal layer above the Sidebar while animating only its panel', () => {
    const overlayRule = componentSource.match(
      /\.sidebar-account-overlay\s*\{[^}]*\}/,
    )?.[0]

    expect(overlayRule).toContain('position: fixed;')
    expect(overlayRule).toContain('inset: 0;')
    expect(overlayRule).toContain('z-index: var(--workspace-layer-account-overlay);')
    expect(overlayRule).toContain('pointer-events: none;')
    expect(componentSource).toContain('.sidebar-account-overlay .account-panel,')
    expect(componentSource).toContain('pointer-events: auto;')
    expect(componentSource).toContain('.account-panel-enter-from .account-panel,')
    expect(componentSource).not.toMatch(
      /\.account-panel-enter-from,\s*\n\.account-panel-leave-to\s*\{[^}]*transform:/,
    )
    expect(frameSource).toContain('z-index: 40;')
    expect(workspaceTokens).toContain('--workspace-layer-account-overlay: 80;')
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

  it('keeps the personal account menu as the same anchored card on a narrow viewport', async () => {
    const { anchor, shell, wrapper } = mountOverlay(true)
    await nextTick()
    await nextTick()

    const panel = document.body.querySelector<HTMLElement>('[data-testid="sidebar-account-panel"]')

    expect(panel?.getAttribute('aria-modal')).toBeNull()
    expect(panel?.classList.contains('account-panel--mobile')).toBe(false)
    expect(document.body.querySelector('.account-panel-backdrop')).toBeNull()
    expect(document.body.querySelector('.account-panel__handle')).toBeNull()
    expect(panel?.style.left).toBe('8px')
    expect(shell.getAttribute('aria-hidden')).toBeNull()
    expect(shell.inert).toBe(false)
    expect(document.body.style.overflow).toBe('')

    anchor.dispatchEvent(new MouseEvent('pointerdown', { bubbles: true }))
    expect(wrapper.emitted('close')).toBeUndefined()
    document.body.dispatchEvent(new MouseEvent('pointerdown', { bubbles: true }))
    expect(wrapper.emitted('close')).toEqual([[false]])
  })

  it('closes an open personal card when the sidebar crosses below 768px', async () => {
    const { wrapper } = mountOverlay(false)
    await nextTick()

    viewportListeners.get('(max-width: 767px)')?.forEach((listener) => {
      listener({ matches: true } as MediaQueryListEvent)
    })

    expect(wrapper.emitted('close')).toEqual([[false]])
  })

  it('retains the administrator bottom sheet and restores its background lock on unmount', async () => {
    const { shell, wrapper } = mountOverlay(true, {
      variant: 'admin',
      appearance: 'personal',
    })
    await nextTick()
    await nextTick()

    const panel = document.body.querySelector<HTMLElement>('[data-testid="sidebar-account-panel"]')
    const backdrop = document.body.querySelector<HTMLButtonElement>('.account-panel-backdrop')

    expect(panel?.getAttribute('aria-modal')).toBe('true')
    expect(panel?.classList.contains('account-panel--mobile')).toBe(true)
    expect(panel?.classList.contains('account-panel--personal')).toBe(true)
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

  it('renders the approved grouped personal account menu and reuses existing actions', async () => {
    const { wrapper } = mountOverlay()
    await nextTick()

    const panel = document.body.querySelector<HTMLElement>('[data-testid="sidebar-account-panel"]')
    const actionIds = Array.from(
      panel?.querySelectorAll<HTMLElement>('[data-testid]') ?? [],
    ).map((element) => element.dataset.testid)
    const preferences = panel?.querySelector<HTMLButtonElement>('[data-testid="account-open-preferences"]')
    const profile = panel?.querySelector<HTMLButtonElement>('[data-testid="account-open-profile"]')
    const settings = panel?.querySelector<HTMLButtonElement>('[data-testid="account-open-settings"]')
    const help = panel?.querySelector<HTMLAnchorElement>('[data-testid="account-help"]')

    expect(actionIds).toEqual([
      'account-open-preferences',
      'account-open-profile',
      'account-open-settings',
      'account-help',
      'account-logout',
    ])
    expect(preferences?.textContent).toContain('accountDock.personalization')
    expect(profile?.textContent).toContain('accountDock.personalProfile')
    expect(settings?.textContent).toContain('accountDock.settings')
    expect(help?.textContent).toContain('accountDock.help')
    expect(help?.getAttribute('href')).toBe('/tutorial-docs/')
    expect(help?.getAttribute('target')).toBe('_blank')
    const personalGroups = Array.from(
      panel?.querySelectorAll<HTMLElement>('nav.account-panel__section--personal') ?? [],
    )
    expect(personalGroups.map((group) => group.getAttribute('aria-label'))).toEqual([
      'accountDock.commonSettings',
      'accountDock.assistance',
    ])
    expect(panel?.querySelector('.account-panel__group-label')).toBeNull()
    expect(panel?.textContent).not.toContain('accountDock.commonSettings')
    expect(panel?.textContent).not.toContain('accountDock.assistance')
    expect(panel?.querySelector('[data-testid="account-admin-guide"]')).toBeNull()
    expect(panel?.querySelector('[data-testid="account-logout"]')).not.toBeNull()
    const accountIcons = Array.from(
      panel?.querySelectorAll<HTMLElement>('[data-account-menu-icon]') ?? [],
    )
    expect(accountIcons.map((icon) => icon.dataset.accountMenuIcon)).toEqual([
      'chevronRight',
      'face',
      'avatar',
      'settings',
      'help',
      'chevronRight',
      'exit',
    ])
    expect(accountIcons.filter((icon) => icon.dataset.accountMenuIcon === 'chevronRight'))
      .toHaveLength(2)
    expect(
      panel?.querySelector('.account-panel__identity [data-account-menu-icon="chevronRight"]'),
    ).not.toBeNull()
    expect(
      help?.querySelector('[data-account-menu-icon="chevronRight"][data-size="16"]'),
    ).not.toBeNull()

    preferences?.click()
    profile?.click()
    settings?.click()
    expect(wrapper.emitted('open-settings')).toEqual([['general'], ['account'], ['general']])
  })

  it('only exposes the administrator guide when requested and emits replay directly', async () => {
    const { wrapper } = mountOverlay(false, { showOnboarding: true, variant: 'admin' })
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

    mountOverlay(true, { variant: 'admin', appearance: 'personal' })
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
