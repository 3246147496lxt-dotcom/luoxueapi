import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { useAppStore } from '@/stores/app'
import WorkspaceSidebarFrame from '../WorkspaceSidebarFrame.vue'
import WorkspaceSidebarHeader from '../WorkspaceSidebarHeader.vue'
import { useWorkspaceSidebarCollapse } from '../useWorkspaceSidebarCollapse'

const directory = dirname(fileURLToPath(import.meta.url))
const frameSource = readFileSync(resolve(directory, '../WorkspaceSidebarFrame.vue'), 'utf8')
const headerSource = readFileSync(resolve(directory, '../WorkspaceSidebarHeader.vue'), 'utf8')
const workspaceTokens = readFileSync(
  resolve(directory, '../../../styles/luoxue-clay-tokens.css'),
  'utf8',
)

const IconStub = {
  template: '<svg data-testid="icon-stub" />',
}

const SidebarCollapseIconStub = {
  props: ['collapsed'],
  template: '<svg data-testid="collapse-icon-stub" :data-collapsed="String(collapsed)" />',
}

describe('WorkspaceSidebarFrame', () => {
  it('renders the shared shell slots in header, mode switch, content, footer order', () => {
    const wrapper = mount(WorkspaceSidebarFrame, {
      props: {
        id: 'workspace-shell',
        label: 'Workspace',
        placement: 'fixed',
        surface: 'work',
      },
      slots: {
        header: '<div data-slot="header" />',
        'mode-switch': '<div data-slot="mode-switch" />',
        default: '<div data-slot="content" />',
        footer: '<div data-slot="footer" />',
      },
    })

    expect(wrapper.element.tagName).toBe('ASIDE')
    expect(wrapper.attributes('id')).toBe('workspace-shell')
    expect(wrapper.attributes('aria-label')).toBe('Workspace')
    expect(wrapper.attributes('data-sidebar-collapsed')).toBe('false')
    expect(wrapper.attributes('data-sidebar-placement')).toBe('fixed')
    expect(wrapper.classes()).toEqual(expect.arrayContaining([
      'workspace-sidebar-frame--fixed',
      'workspace-sidebar-frame--work',
      'workspace-sidebar-frame--content-workspace',
    ]))

    const frameElement = wrapper.element as HTMLElement
    const childOrder = Array.from(frameElement.children).map((element) => {
      if (element.classList.contains('workspace-sidebar-frame__content')) return 'content'
      if (element.classList.contains('workspace-sidebar-frame__footer')) return 'footer'
      return element.getAttribute('data-slot')
    })
    expect(childOrder).toEqual(['header', 'mode-switch', 'content', 'footer'])
    expect(wrapper.find('.workspace-sidebar-frame__content [data-slot="content"]').exists()).toBe(true)
  })

  it('uses one collapsed state and hides only the mode-switch slot', () => {
    const wrapper = mount(WorkspaceSidebarFrame, {
      props: {
        id: 'workspace-shell',
        label: 'Workspace',
        collapsed: true,
        placement: 'flow',
        surface: 'chat',
      },
      slots: {
        header: '<div data-slot="header" />',
        'mode-switch': '<div data-slot="mode-switch" />',
        default: '<div data-slot="content" />',
        footer: '<div data-slot="footer" />',
      },
    })

    expect(wrapper.attributes('data-sidebar-collapsed')).toBe('true')
    expect(wrapper.classes()).toEqual(expect.arrayContaining([
      'workspace-sidebar-frame--collapsed',
      'workspace-sidebar-frame--flow',
      'workspace-sidebar-frame--chat',
    ]))
    expect(wrapper.find('[data-slot="mode-switch"]').exists()).toBe(false)
    expect(wrapper.find('[data-slot="header"]').exists()).toBe(true)
    expect(wrapper.find('[data-slot="content"]').exists()).toBe(true)
    expect(wrapper.find('[data-slot="footer"]').exists()).toBe(true)
  })

  it('normalizes mobile drawers to the expanded state', () => {
    const wrapper = mount(WorkspaceSidebarFrame, {
      props: {
        id: 'workspace-shell',
        label: 'Workspace',
        collapsed: true,
        mobile: true,
      },
      slots: {
        'mode-switch': '<div data-slot="mode-switch" />',
      },
    })

    expect(wrapper.attributes('data-sidebar-collapsed')).toBe('false')
    expect(wrapper.classes()).not.toContain('workspace-sidebar-frame--collapsed')
    expect(wrapper.find('[data-slot="mode-switch"]').exists()).toBe(true)
  })

  it('normalizes narrow overlays to the full 260px expanded state', () => {
    const wrapper = mount(WorkspaceSidebarFrame, {
      props: {
        id: 'workspace-shell',
        label: 'Workspace',
        collapsed: true,
        overlay: true,
      },
      slots: {
        'mode-switch': '<div data-slot="mode-switch" />',
      },
    })

    expect(wrapper.attributes('data-sidebar-collapsed')).toBe('false')
    expect(wrapper.classes()).toContain('workspace-sidebar-frame--overlay')
    expect(wrapper.classes()).not.toContain('workspace-sidebar-frame--collapsed')
    expect(wrapper.find('[data-slot="mode-switch"]').exists()).toBe(true)
  })

  it('owns expanded, collapsed, mobile, and content padding geometry through Workspace tokens', () => {
    expect(frameSource).toMatch(
      /\.workspace-sidebar-frame\s*\{[^}]*width: var\(--workspace-sidebar-width\);[^}]*min-width: var\(--workspace-sidebar-width\);/s,
    )
    expect(frameSource).toMatch(
      /\.workspace-sidebar-frame--collapsed\s*\{[^}]*width: var\(--workspace-sidebar-width-collapsed\);[^}]*min-width: var\(--workspace-sidebar-width-collapsed\);/s,
    )
    expect(frameSource).toMatch(
      /\.workspace-sidebar-frame__content\s*\{[^}]*padding: var\(--workspace-sidebar-content-padding\);/s,
    )
    expect(frameSource).toContain('padding: var(--workspace-sidebar-content-padding-collapsed);')
    expect(frameSource).toContain('padding: var(--workspace-sidebar-content-padding-admin);')
    expect(frameSource).toContain('<slot name="footer" />')
    expect(frameSource).toContain('class="workspace-sidebar-frame__footer"')
    expect(frameSource).toMatch(
      /\.workspace-sidebar-frame--content-workspace \.workspace-sidebar-frame__footer\s*\{[^}]*border-top: 1px solid var\(--workspace-footer-divider\);[^}]*padding-top: var\(--workspace-space-2\);/s,
    )
    expect(frameSource).not.toMatch(
      /\.workspace-sidebar-frame--content-admin \.workspace-sidebar-frame__footer\s*\{[^}]*border-top:/s,
    )
    expect(frameSource).toMatch(
      /\.workspace-sidebar-frame--content-workspace\.workspace-sidebar-frame--collapsed\s+\.workspace-sidebar-frame__footer\s*\{[^}]*border-top-color: transparent;/s,
    )

    expect(workspaceTokens).toContain('--workspace-sidebar-width: 260px;')
    expect(workspaceTokens).toContain('--workspace-sidebar-width-collapsed: 68px;')
    expect(workspaceTokens).toContain(
      '--workspace-sidebar-content-padding: var(--workspace-space-1) var(--workspace-space-2)',
    )
    expect(workspaceTokens).toContain(
      '--workspace-sidebar-content-padding-collapsed: var(--workspace-space-2);',
    )
    expect(workspaceTokens).not.toContain('--workspace-chat-sidebar-width-collapsed')
  })
})

describe('WorkspaceSidebarHeader', () => {
  it('owns search and collapse controls while preserving the brand slot', async () => {
    const wrapper = mount(WorkspaceSidebarHeader, {
      attachTo: document.body,
      props: {
        controls: 'workspace-shell',
        showSearch: true,
        searchExpanded: false,
        searchControls: 'workspace-search',
        searchLabel: 'Search',
        collapseLabel: 'Collapse',
        expandLabel: 'Expand',
        closeLabel: 'Close',
      },
      slots: { brand: '<span data-testid="brand-slot">Brand</span>' },
      global: { stubs: { Icon: IconStub, SidebarCollapseIcon: SidebarCollapseIconStub } },
    })

    expect(wrapper.get('[data-testid="brand-slot"]').text()).toBe('Brand')
    const search = wrapper.get('button[aria-label="Search"]')
    expect(search.attributes('aria-controls')).toBe('workspace-search')
    expect(search.find('.workspace-desktop-sidebar-header-icon--search').exists()).toBe(true)
    const toggle = wrapper.get('[data-testid="workspace-sidebar-collapse-toggle"]')
    expect(toggle.attributes('aria-controls')).toBe('workspace-shell')
    expect(toggle.attributes('aria-expanded')).toBe('true')
    expect(toggle.find('.workspace-desktop-sidebar-header-icon--collapse').exists()).toBe(true)
    expect(toggle.find('.workspace-sidebar-header__toggle-icon').exists()).toBe(false)

    await wrapper.get('button[aria-label="Search"]').trigger('click')
    await toggle.trigger('click')
    expect(wrapper.emitted('search')).toHaveLength(1)
    expect(wrapper.emitted('toggle')).toHaveLength(1)

    ;(wrapper.vm as unknown as { focusToggle: () => void }).focusToggle()
    expect(document.activeElement).toBe(toggle.element)
    wrapper.unmount()
  })

  it('keeps one stable collapse selector and exposes the collapsed state accessibly', async () => {
    const wrapper = mount(WorkspaceSidebarHeader, {
      props: {
        controls: 'workspace-shell',
        collapsed: true,
        collapsedLogoSrc: '/brand/workspace-mark.svg',
        showSearch: true,
        searchLabel: 'Search',
        collapseLabel: 'Collapse',
        expandLabel: 'Expand',
        closeLabel: 'Close',
      },
      slots: { brand: '<span data-testid="brand-slot">Brand</span>' },
      global: { stubs: { Icon: IconStub, SidebarCollapseIcon: SidebarCollapseIconStub } },
    })

    expect(wrapper.classes()).toContain('workspace-sidebar-header--collapsed')
    expect(wrapper.find('[data-testid="brand-slot"]').exists()).toBe(false)
    expect(wrapper.find('button[aria-label="Search"]').exists()).toBe(false)
    const toggle = wrapper.get('[data-testid="workspace-sidebar-collapse-toggle"]')
    expect(toggle.attributes('aria-label')).toBe('Expand')
    expect(toggle.attributes('aria-expanded')).toBe('false')
    expect(wrapper.findAll('button')).toHaveLength(1)
    expect(toggle.findAll('button')).toHaveLength(0)
    expect(toggle.get('.workspace-sidebar-header__collapsed-brand-image').attributes('src'))
      .toBe('/brand/workspace-mark.svg')
    expect(toggle.get('.workspace-sidebar-header__collapsed-brand-mark').attributes('aria-hidden'))
      .toBe('true')
    expect(toggle.get('[data-testid="workspace-sidebar-collapse-toggle-icon"]')
      .attributes('data-collapsed')).toBe('true')

    await toggle.trigger('click')
    expect(wrapper.emitted('toggle')).toEqual([[]])
  })

  it('keeps the mobile close control first and removes the desktop collapse control', async () => {
    const wrapper = mount(WorkspaceSidebarHeader, {
      props: {
        controls: 'workspace-shell',
        collapsed: true,
        mobile: true,
        showSearch: true,
        searchLabel: 'Search',
        collapseLabel: 'Collapse',
        expandLabel: 'Expand',
        closeLabel: 'Close',
      },
      slots: { brand: '<span data-testid="brand-slot">Brand</span>' },
      global: { stubs: { Icon: IconStub, SidebarCollapseIcon: SidebarCollapseIconStub } },
    })

    const buttons = wrapper.findAll('button')
    expect(buttons[0]?.attributes('aria-label')).toBe('Close')
    expect(wrapper.find('[data-testid="brand-slot"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="workspace-sidebar-collapse-toggle"]').exists()).toBe(false)
    expect(wrapper.find('button[aria-label="Search"]').exists()).toBe(true)

    await buttons[0]!.trigger('click')
    expect(wrapper.emitted('close')).toHaveLength(1)
  })

  it('uses the dedicated close and search controls in a narrow overlay', async () => {
    const wrapper = mount(WorkspaceSidebarHeader, {
      props: {
        controls: 'workspace-shell',
        collapsed: true,
        overlay: true,
        showSearch: true,
        showClose: true,
        searchLabel: 'Search',
        collapseLabel: 'Collapse',
        expandLabel: 'Expand',
        closeLabel: 'Close',
      },
      slots: { brand: '<span data-testid="brand-slot">Brand</span>' },
      global: { stubs: { Icon: IconStub, SidebarCollapseIcon: SidebarCollapseIconStub } },
    })

    expect(wrapper.classes()).toContain('workspace-sidebar-header--overlay')
    expect(wrapper.classes()).not.toContain('workspace-sidebar-header--collapsed')
    expect(wrapper.find('[data-testid="brand-slot"]').exists()).toBe(false)
    expect(wrapper.get('.workspace-sidebar-header__overlay-brand-image').attributes('src'))
      .toBe('/logo.png')
    expect(wrapper.find('[data-testid="workspace-sidebar-collapse-toggle"]').exists()).toBe(false)
    expect(wrapper.find('button[aria-label="Close"] .workspace-responsive-sidebar-icon--close')
      .exists()).toBe(true)
    expect(wrapper.find('button[aria-label="Search"] .workspace-responsive-sidebar-icon--search')
      .exists()).toBe(true)

    await wrapper.get('button[aria-label="Close"]').trigger('click')
    expect(wrapper.emitted('close')).toEqual([[]])
  })

  it('owns the shared 52px header geometry and tokenized actions', () => {
    expect(headerSource).toMatch(
      /\.workspace-sidebar-header\s*\{[^}]*height: var\(--workspace-sidebar-header-height\);[^}]*flex: 0 0 var\(--workspace-sidebar-header-height\);/s,
    )
    expect(headerSource).toContain('padding: var(--workspace-space-2) var(--workspace-sidebar-header-padding-inline);')
    expect(headerSource).toContain(
      'padding-inline-end: var(--workspace-sidebar-header-actions-inset-end);',
    )
    expect(headerSource).toContain('width: var(--workspace-sidebar-action-size);')
    expect(headerSource).not.toContain(':deep(svg)')
    expect(headerSource).toContain('height: var(--workspace-sidebar-action-size);')
    expect(headerSource).toMatch(
      /\.workspace-sidebar-header__actions\s*\{[^}]*gap: 0;/s,
    )
    expect(headerSource).toContain('color: var(--workspace-sidebar-header-action);')
    expect(headerSource).toContain('<WorkspaceDesktopSidebarHeaderIcon')
    expect(headerSource).toContain('name="collapse"')
    expect(headerSource).toContain('width: var(--workspace-sidebar-touch-target);')
    expect(headerSource).toContain('height: var(--workspace-sidebar-touch-target);')
    expect(headerSource).toMatch(
      /\.workspace-sidebar-header--collapsed\s*\{[^}]*padding-top: var\(--workspace-space-1\);[^}]*padding-bottom: var\(--workspace-space-1\);/s,
    )
    expect(headerSource).toMatch(
      /\.workspace-sidebar-header--collapsed\s*\{[^}]*justify-content: start;[^}]*padding-right: var\(--workspace-space-2\);[^}]*padding-left: var\(--workspace-space-2\);/s,
    )
    expect(headerSource).not.toMatch(
      /\.workspace-sidebar-header--collapsed\s*\{[^}]*justify-content: center;/s,
    )
    expect(headerSource).toMatch(
      /\.workspace-sidebar-header--mobile\s*\{[^}]*padding-top: var\(--workspace-space-1\);[^}]*padding-bottom: var\(--workspace-space-1\);/s,
    )
    expect(headerSource).toContain(':data-testid="toggleTestId"')
    expect(headerSource).toContain("toggleTestId: 'workspace-sidebar-collapse-toggle'")
    expect(headerSource).toContain('grid-area: 1 / 1;')
    expect(headerSource).toContain('opacity 180ms cubic-bezier(0.16, 1, 0.3, 1)')
    expect(headerSource).toContain('transform 180ms cubic-bezier(0.16, 1, 0.3, 1)')
    expect(headerSource).toContain('.workspace-sidebar-header__toggle--collapsed:hover')
    expect(headerSource).toContain('.workspace-sidebar-header__toggle--collapsed:focus-visible')
    expect(headerSource).toMatch(
      /@media \(prefers-reduced-motion: reduce\)[\s\S]*?\.workspace-sidebar-header__collapsed-brand-mark,[\s\S]*?transition: none;/,
    )

    expect(workspaceTokens).toContain('--workspace-sidebar-header-height: 52px;')
    expect(workspaceTokens).toContain(
      '--workspace-sidebar-header-padding-inline: var(--workspace-space-3);',
    )
    expect(workspaceTokens).toContain(
      '--workspace-sidebar-header-actions-inset-end: var(--workspace-space-1-75);',
    )
    expect(workspaceTokens).toContain('--workspace-sidebar-action-size: 36px;')
    expect(workspaceTokens).toContain(
      '--workspace-sidebar-header-action: var(--workspace-identity-text-tertiary);',
    )
    expect(workspaceTokens).toContain('--workspace-sidebar-touch-target: 44px;')
    expect(workspaceTokens).toContain('--workspace-sidebar-transition-duration: 300ms;')
    expect(workspaceTokens).toContain('--workspace-sidebar-transition-easing: ease-out;')
  })
})

describe('useWorkspaceSidebarCollapse', () => {
  it('keeps the manual desktop rail request while narrow Chat and Work overlays render expanded', async () => {
    setActivePinia(createPinia())
    const appStore = useAppStore()
    const overlay = ref(false)
    const workState = useWorkspaceSidebarCollapse({ overlay })
    const chatState = useWorkspaceSidebarCollapse({ overlay })
    appStore.setSidebarCollapsed(true)

    expect(workState.collapsed.value).toBe(true)
    expect(chatState.collapsed.value).toBe(true)

    overlay.value = true
    expect(appStore.sidebarCollapsed).toBe(true)
    expect(workState.requestedCollapsed.value).toBe(true)
    expect(chatState.requestedCollapsed.value).toBe(true)
    expect(workState.collapsed.value).toBe(false)
    expect(chatState.collapsed.value).toBe(false)

    await workState.toggle()
    workState.expand()
    expect(appStore.sidebarCollapsed).toBe(true)

    overlay.value = false
    expect(workState.collapsed.value).toBe(true)
    expect(chatState.collapsed.value).toBe(true)
  })

  it('shares one Pinia state and post-render focus handoff across shell hosts', async () => {
    setActivePinia(createPinia())
    const appStore = useAppStore()
    const beforeCollapse = vi.fn()
    const workFocusToggle = vi.fn()
    const chatFocusToggle = vi.fn()
    const workState = useWorkspaceSidebarCollapse({
      beforeCollapse,
    })
    const chatState = useWorkspaceSidebarCollapse()
    workState.headerRef.value = {
      focusSearch: vi.fn(),
      focusToggle: workFocusToggle,
      focusClose: vi.fn(),
    }
    chatState.headerRef.value = {
      focusSearch: vi.fn(),
      focusToggle: chatFocusToggle,
      focusClose: vi.fn(),
    }

    await workState.toggle()
    expect(appStore.sidebarCollapsed).toBe(true)
    expect(workState.collapsed.value).toBe(true)
    expect(chatState.collapsed.value).toBe(true)
    expect(beforeCollapse).toHaveBeenCalledTimes(1)
    expect(workFocusToggle).toHaveBeenCalledTimes(1)

    chatState.expand()
    expect(appStore.sidebarCollapsed).toBe(false)
    expect(workState.collapsed.value).toBe(false)
    expect(chatState.collapsed.value).toBe(false)

    await chatState.toggle()
    expect(appStore.sidebarCollapsed).toBe(true)
    expect(beforeCollapse).toHaveBeenCalledTimes(1)
    expect(chatFocusToggle).toHaveBeenCalledTimes(1)
  })

  it('keeps the desktop request while mobile renders expanded', async () => {
    setActivePinia(createPinia())
    const appStore = useAppStore()
    const mobile = ref(true)
    const state = useWorkspaceSidebarCollapse({ mobile })

    appStore.setSidebarCollapsed(true)
    expect(state.requestedCollapsed.value).toBe(true)
    expect(state.collapsed.value).toBe(false)

    state.expand()
    await state.toggle()
    expect(appStore.sidebarCollapsed).toBe(true)

    mobile.value = false
    expect(state.collapsed.value).toBe(true)
  })
})
