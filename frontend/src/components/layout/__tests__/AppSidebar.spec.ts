import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../AppSidebar.vue')
const componentSource = readFileSync(componentPath, 'utf8')
const frameSource = readFileSync(
  resolve(dirname(fileURLToPath(import.meta.url)), '../WorkspaceSidebarFrame.vue'),
  'utf8',
)
const headerSource = readFileSync(
  resolve(dirname(fileURLToPath(import.meta.url)), '../WorkspaceSidebarHeader.vue'),
  'utf8',
)
const responsiveSource = readFileSync(
  resolve(dirname(fileURLToPath(import.meta.url)), '../workspaceResponsive.ts'),
  'utf8',
)
const workspaceTokens = readFileSync(
  resolve(dirname(fileURLToPath(import.meta.url)), '../../../styles/luoxue-clay-tokens.css'),
  'utf8',
)
const userNavigationSource = readFileSync(
  resolve(dirname(fileURLToPath(import.meta.url)), '../sidebar/userNavigation.ts'),
  'utf8',
)

describe('AppSidebar custom SVG styles', () => {
  it('does not override uploaded SVG fill or stroke colors', () => {
    expect(componentSource).toContain('.sidebar-svg-icon {')
    expect(componentSource).toContain('color: currentColor;')
    expect(componentSource).toContain('display: block;')
    expect(componentSource).not.toContain('stroke: currentColor;')
    expect(componentSource).not.toContain('fill: none;')
  })
})

describe('AppSidebar scroll position persistence', () => {
  it('binds a template ref to the sidebar nav element', () => {
    expect(componentSource).toContain('ref="sidebarNavRef"')
    expect(componentSource).toContain('workspace-sidebar-navigation')
  })

  it('declares sidebarNavRef in script setup', () => {
    expect(componentSource).toContain("const sidebarNavRef = ref<HTMLElement | null>(null)")
  })

  it('saves scroll position on beforeUnmount', () => {
    expect(componentSource).toContain('onBeforeUnmount')
    expect(componentSource).toContain('appStore.sidebarScrollTop')
    expect(componentSource).toContain('sidebarNavRef.value.scrollTop')
  })

  it('restores scroll position on mount', () => {
    expect(componentSource).toContain('onMounted')
    expect(componentSource).toContain('appStore.sidebarScrollTop')
    expect(componentSource).toContain('nextTick')
  })
})

describe('AppSidebar navigation shell', () => {
  it('composes the shared Frame and Header around the Work navigation', () => {
    expect(componentSource).toContain('<WorkspaceSidebarFrame')
    expect(componentSource).toContain('id="app-sidebar"')
    expect(componentSource).toContain(':label="t(\'nav.workMode\')"')
    expect(componentSource).toContain(':collapsed="sidebarCollapsed"')
    expect(componentSource).toContain(':mobile="mobileViewport"')
    expect(componentSource).toContain(":placement=\"personalNarrowViewport ? 'flow' : 'fixed'\"")
    expect(componentSource).toContain(':overlay="personalNarrowViewport"')
    expect(componentSource).toContain('surface="work"')
    expect(componentSource).toContain('<template #header>')
    expect(componentSource).toContain('<WorkspaceSidebarHeader')
    expect(componentSource).toContain('data-testid="sidebar-brand-row"')
    expect(componentSource).not.toContain('class="sidebar-header"')
    expect(componentSource).toContain('controls="app-sidebar"')
    expect(componentSource).toContain('toggle-test-id="sidebar-collapse-toggle"')
    expect(componentSource).toContain('toggle-icon-test-id="sidebar-collapse-toggle-icon"')
    expect(componentSource).toContain('@toggle="toggleSidebar"')
    expect(componentSource).toContain('@close="closeNarrowSidebar"')
    expect(componentSource).toContain('<WorkspaceSidebarBrand')
    expect(componentSource).toContain('v-if="isPersonalWorkWorkspace"')
    expect(componentSource).toContain('home-path="/dashboard"')
    expect(componentSource).toContain('<AppModeSwitch active-mode="work" @change="handleModeChange" />')
    expect(componentSource).toContain('useWorkspaceSidebarCollapse({')
    expect(componentSource).toMatch(/collapsed:\s*sidebarCollapsed/)
    expect(componentSource).toMatch(/expand:\s*expandSidebar/)
    expect(componentSource).toMatch(/toggle:\s*toggleSidebar/)
    expect(componentSource).not.toContain('VersionBadge')
    expect(componentSource).not.toContain('sidebar-footer')
    expect(componentSource).toContain(
      '<UserAccountCard context="work" :collapsed="sidebarCollapsed" />',
    )
  })

  it('does not draw a divider between the brand row and navigation sections', () => {
    const headerRules = headerSource.match(/\.workspace-sidebar-header\s*\{[^}]*\}/g) ?? []

    expect(headerRules.length).toBeGreaterThan(0)
    expect(headerRules.every(rule => !/border-bottom\s*:/.test(rule))).toBe(true)
    expect(componentSource).not.toContain('class="sidebar-header border-b')
  })

  it('pins the shared personal brand left and the compact admin logo in the same Header', () => {
    const brandRule = componentSource.match(
      /\.workspace-sidebar-brand--work\s*\{[^}]*\}/,
    )?.[0]
    const actionsRule = headerSource.match(
      /\.workspace-sidebar-header__actions\s*\{[^}]*\}/,
    )?.[0]

    expect(componentSource).toMatch(
      /<WorkspaceSidebarHeader[\s\S]*?<template #brand>[\s\S]*?<WorkspaceSidebarBrand[\s\S]*?v-if="isPersonalWorkWorkspace"[\s\S]*?home-path="\/dashboard"[\s\S]*?<AppBrand[\s\S]*?v-else/,
    )
    expect(brandRule).toContain('width: 2.25rem;')
    expect(brandRule).toContain('height: 2.25rem;')
    expect(brandRule).toContain('flex: 0 0 2.25rem;')
    expect(actionsRule).toContain('justify-content: flex-end;')
    expect(componentSource).toMatch(
      /\.workspace-sidebar-brand--work :deep\(\.app-brand-logo-image\)\s*\{[^}]*width: 1\.25rem;[^}]*height: 1\.25rem;/,
    )
    expect(componentSource).toMatch(
      /\.workspace-sidebar-brand--work :deep\(\.app-brand-logo-image-default\)\s*\{[^}]*transform: scale\(1\.4\);/,
    )
  })

  it('matches the target collapse button hover and cursor behavior', () => {
    expect(headerSource).toMatch(
      /\.workspace-sidebar-header__toggle\s*\{[^}]*cursor: w-resize;/,
    )
    expect(headerSource).toMatch(
      /\.workspace-sidebar-header__action:hover\s*\{[^}]*background: var\(--workspace-hover\);/,
    )
    expect(headerSource).toMatch(
      /\.workspace-sidebar-header__toggle--collapsed\s*\{[^}]*cursor: pointer;/,
    )
    expect(headerSource).toContain(
      ":global([dir='rtl']) .workspace-sidebar-header__toggle",
    )
    expect(headerSource).toContain(
      ":global([dir='rtl']) .workspace-sidebar-header__toggle--collapsed",
    )
  })

  it('uses matching compact and expanded widths inside a full-height desktop rail', () => {
    expect(frameSource).toContain('width: var(--workspace-sidebar-width);')
    expect(frameSource).toContain('min-width: var(--workspace-sidebar-width);')
    expect(frameSource).toContain('width: var(--workspace-sidebar-width-collapsed);')
    expect(frameSource).toContain('min-width: var(--workspace-sidebar-width-collapsed);')
    expect(workspaceTokens).toContain('--workspace-sidebar-width: 260px;')
    expect(workspaceTokens).toContain('--workspace-sidebar-width-collapsed: 68px;')
    expect(workspaceTokens).toContain('--workspace-sidebar-width-mobile: min(84vw, 288px);')
    expect(workspaceTokens).not.toContain('--workspace-chat-sidebar-width-collapsed')
    expect(frameSource).toContain('top: 0;')
    expect(frameSource).toContain('top: var(--app-shell-top-offset);')
    expect(frameSource).not.toContain('top: 5.0625rem;')
    expect(frameSource).toContain('padding: var(--workspace-sidebar-content-padding);')
    expect(componentSource).toContain('min-height: 2.25rem;')
    expect(componentSource).toContain('padding-left: 0.75rem;')
    expect(componentSource).toContain('padding-right: 0.75rem;')
    expect(componentSource).toContain('min-height: 2.75rem;')
    expect(componentSource).toMatch(
      /\.sidebar-link-collapsed\s*\{[^}]*width: var\(--workspace-sidebar-touch-target\);[^}]*justify-content: flex-start;[^}]*padding-left: var\(--workspace-space-3\);/s,
    )
    expect(componentSource).not.toMatch(
      /\.sidebar-link-collapsed\s*\{[^}]*justify-content: center;/s,
    )
    expect(componentSource).toMatch(
      /\.sidebar-label-collapsed\s*\{[^}]*max-width: 0;[^}]*opacity: 0;/s,
    )
    expect(componentSource).not.toMatch(
      /\.sidebar-label-collapsed\s*\{[^}]*transform:/s,
    )
    expect(componentSource).toContain('border-width: 0 1px 0 0;')
    expect(headerSource).toContain('height: var(--workspace-sidebar-header-height);')
    expect(headerSource).toContain('flex: 0 0 var(--workspace-sidebar-header-height);')
    expect(frameSource).not.toContain('font-family:')
    expect(workspaceTokens).toContain('--workspace-sidebar-header-height: 52px;')
    expect(workspaceTokens).toContain(
      '--workspace-font-ui: Inter, "PingFang SC", "Microsoft YaHei", sans-serif;',
    )
    expect(headerSource).toContain('.workspace-sidebar-header--collapsed {')
    expect(componentSource).toContain(
      "'sidebar-mobile-hidden': mobileViewport && !mobileOpen,",
    )
    expect(componentSource).toContain('transform: translateX(-100%);')
  })

  it('removes the off-canvas mobile navigation from keyboard and screen-reader access', () => {
    expect(componentSource).toContain(
      "const mobileNavigationHidden = computed(() => mobileViewport.value && !mobileOpen.value)",
    )
    expect(componentSource).toContain(":aria-hidden=\"mobileNavigationHidden ? 'true' : undefined\"")
    expect(componentSource).toContain(':inert="mobileNavigationHidden"')
    expect(responsiveSource).toContain('window.matchMedia(WORKSPACE_MOBILE_DRAWER_MEDIA_QUERY)')
    expect(responsiveSource).toContain('workspaceMobileDrawerFallback()')
    expect(componentSource).not.toContain("window.matchMedia('(max-width: 1023px)')")
  })

  it('keeps every mobile sidebar link at least 44px tall', () => {
    const mobileRules = componentSource.slice(
      componentSource.indexOf(
        '@media (max-width: 767px) and (hover: none) and (pointer: coarse)',
      ),
    )

    expect(mobileRules).toMatch(
      /\.sidebar-link,[\s\S]*?min-height: 2\.75rem;/,
    )
  })

  it('keeps the mobile overlay and configurable glass fallback above it', () => {
    expect(componentSource).toContain(
      'class="app-sidebar-backdrop fixed inset-0 z-30 border-0 p-0 backdrop-blur-[1px]"',
    )
    expect(componentSource).toMatch(
      /\.app-sidebar-backdrop\s*\{[^}]*display: none;[^}]*background: var\(--workspace-overlay-backdrop\);/,
    )
    expect(componentSource).toMatch(
      /@media \(max-width: 767px\) and \(hover: none\) and \(pointer: coarse\)[\s\S]*?\.app-sidebar-backdrop\s*\{[^}]*display: block;/,
    )
    expect(componentSource).toContain(
      'backdrop-filter: var(--app-shell-sidebar-backdrop, saturate(1.7) blur(36px));',
    )
  })

  it('matches the production menu rhythm and tokenized selected state', () => {
    expect(componentSource).toContain('gap: 0.625rem;')
    expect(componentSource).toContain('padding-top: 0.25rem;')
    expect(componentSource).toContain('line-height: 1.5rem;')
    expect(componentSource).toContain('font-weight: 400;')
    expect(componentSource).toContain(
      'color: var(--app-shell-sidebar-active-color, var(--workspace-text));',
    )
    expect(componentSource).toContain(
      'background: var(--app-shell-sidebar-active-bg, var(--workspace-selected));',
    )
    expect(componentSource).toContain(
      'color: var(--app-shell-sidebar-active-icon, var(--workspace-text));',
    )
    expect(componentSource).toContain(
      'display: var(--app-shell-sidebar-active-marker, none);',
    )
    expect(componentSource).toContain('width: 2px;')
    expect(componentSource).toContain('height: 1rem;')
    expect(componentSource).toContain('font-size: 0.75rem;')
    expect(componentSource).toContain('font-weight: 400;')
    expect(componentSource).toMatch(
      /\.sidebar--personal-work \.sidebar-link\s*\{[\s\S]*?font-size: var\(--workspace-type-navigation-size\);[\s\S]*?font-weight: var\(--workspace-type-navigation-weight\);/,
    )
    expect(componentSource).toMatch(
      /\.sidebar--personal-work \.sidebar-section-title[\s\S]*?font-size: var\(--workspace-type-secondary-size\);[\s\S]*?font-weight: var\(--workspace-type-secondary-weight\);/,
    )
  })

  it('pins expandable group chevrons to the trailing edge', () => {
    expect(componentSource).toContain('.sidebar-label-flex {')
    expect(componentSource).toContain('flex: 1 1 auto;')
    expect(componentSource).toContain('justify-content: space-between;')
  })

  it('uses a logical trailing edge and RTL mirror for destination jump cues', () => {
    expect(userNavigationSource).toContain("trailingIcon: 'destinationArrowUpRight'")
    expect(componentSource).toContain('data-testid="sidebar-nav-trailing-icon"')
    expect(componentSource).not.toContain('name="externalLink"')
    expect(componentSource).not.toContain('sidebar-support-external')
    expect(componentSource).toContain('margin-inline-start: auto;')
    expect(componentSource).toContain(":global([dir='rtl']) .sidebar-nav-trailing-icon")
    expect(componentSource).toContain('transform: scaleX(-1);')
  })
})

describe('AppSidebar utility actions', () => {
  it('keeps support actions in scrolling navigation and account actions in the dock', () => {
    const navStart = componentSource.indexOf('<nav')
    const navEnd = componentSource.indexOf('</nav>')
    const support = componentSource.indexOf('data-testid="sidebar-support-section"')
    const dock = componentSource.indexOf(
      '<UserAccountCard context="work" :collapsed="sidebarCollapsed" />',
    )

    expect(support).toBeGreaterThan(navStart)
    expect(support).toBeLessThan(navEnd)
    expect(navEnd).toBeGreaterThan(-1)
    expect(dock).toBeGreaterThan(navEnd)
    expect(componentSource).not.toContain('data-testid="sidebar-announcements"')
    expect(componentSource).not.toContain('data-testid="sidebar-settings"')
    expect(componentSource).not.toContain("t('nav.userSections.resources')")
    expect(componentSource).toContain("'sidebar-docs-tutorial'")
    expect(componentSource).toContain("'sidebar-contact-us'")
    expect(componentSource).not.toContain('data-testid="sidebar-help-resources"')
    expect(componentSource).toContain('sidebarSupportLinks')
    expect(componentSource).toContain("import UserAccountCard from './UserAccountCard.vue'")
    expect(componentSource).not.toContain('toggleTheme')
    expect(componentSource).not.toContain("t('nav.lightMode')")
    expect(componentSource).not.toContain("t('nav.darkMode')")
  })
})

describe('AppSidebar account destination ownership', () => {
  it('surfaces task-oriented account destinations in the personal workspace', () => {
    expect(userNavigationSource).toContain('ACCOUNT_DESTINATION_PATHS')
    expect(userNavigationSource).toContain('ACCOUNT_DESTINATION_PATHS.quotaViewer')
    expect(userNavigationSource).toContain('ACCOUNT_DESTINATION_PATHS.subscriptions')
    expect(userNavigationSource).toContain('ACCOUNT_DESTINATION_PATHS.wallet')
    expect(userNavigationSource).toContain('ACCOUNT_DESTINATION_PATHS.orders')
    expect(userNavigationSource).toContain('USER_ACCOUNT_PATHS')
    expect(userNavigationSource).not.toContain("id: 'more'")
    expect(userNavigationSource).not.toContain("label: context.t('nav.userSections.more')")
    expect(componentSource).not.toContain('data-testid="sidebar-destination-links"')
  })
})
