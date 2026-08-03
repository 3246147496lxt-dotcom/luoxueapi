import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../AppSidebar.vue')
const componentSource = readFileSync(componentPath, 'utf8')

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
    expect(componentSource).toContain('sidebar-nav')
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
  it('exposes a stable aria target and owns the desktop brand and collapse control', () => {
    expect(componentSource).toContain('id="app-sidebar"')
    expect(componentSource).toContain('aria-label="Sidebar"')
    expect(componentSource).toContain('data-testid="sidebar-brand-row"')
    expect(componentSource).toContain('class="sidebar-header"')
    expect(componentSource).toContain('<AppBrand placement="sidebar" :collapsed="sidebarCollapsed" />')
    expect(componentSource).toContain('data-testid="sidebar-collapse-toggle"')
    expect(componentSource).toContain('aria-controls="app-sidebar"')
    expect(componentSource).toContain(':aria-expanded="!sidebarCollapsed"')
    expect(componentSource).toContain('data-testid="sidebar-collapse-toggle-icon"')
    expect(componentSource).toContain('<SidebarCollapseIcon')
    expect(componentSource).toContain(':collapsed="sidebarCollapsed"')
    expect(componentSource).toContain('@click="toggleSidebar"')
    expect(componentSource).toContain('appStore.toggleSidebar()')
    expect(componentSource).not.toContain('VersionBadge')
    expect(componentSource).not.toContain('sidebar-footer')
    expect(componentSource).toContain('<SidebarAccountDock />')
  })

  it('does not draw a divider between the brand row and navigation sections', () => {
    const headerRules = componentSource.match(/\.sidebar-header\s*\{[^}]*\}/g) ?? []

    expect(headerRules.length).toBeGreaterThan(0)
    expect(headerRules.every(rule => !/border-bottom\s*:/.test(rule))).toBe(true)
    expect(componentSource).not.toContain('class="sidebar-header border-b')
  })

  it('pins the compact brand left and the collapse control right', () => {
    const brandRule = componentSource.match(
      /\.sidebar-header :deep\(\.app-brand\)\s*\{[^}]*\}/,
    )?.[0]
    const collapseRule = componentSource.match(
      /\.sidebar-header \.sidebar-collapse-toggle\s*\{[^}]*\}/,
    )?.[0]

    expect(componentSource).toMatch(
      /<AppBrand placement="sidebar" :collapsed="sidebarCollapsed" \/>\s*<button[\s\S]*?data-testid="sidebar-collapse-toggle"/,
    )
    expect(brandRule).toContain('width: 2.25rem;')
    expect(brandRule).toContain('height: 2.25rem;')
    expect(brandRule).toContain('flex: 0 0 2.25rem;')
    expect(collapseRule).toContain('margin-left: auto;')
    expect(componentSource).toContain('class="h-5 w-5"')
    expect(componentSource).toMatch(
      /\.sidebar-header :deep\(\.app-brand-logo-image\)\s*\{[^}]*width: 1\.25rem;[^}]*height: 1\.25rem;/,
    )
    expect(componentSource).toMatch(
      /\.sidebar-header :deep\(\.app-brand-logo-image-default\)\s*\{[^}]*transform: scale\(1\.4\);/,
    )
  })

  it('matches the target collapse button hover and resize-cursor behavior', () => {
    expect(componentSource).toMatch(
      /\.sidebar-collapse-toggle\s*\{[^}]*cursor: w-resize;/,
    )
    expect(componentSource).toMatch(
      /\.sidebar-collapse-toggle\s*\{[^}]*transition: none;/,
    )
    expect(componentSource).toMatch(
      /\.sidebar-collapse-toggle:hover\s*\{[^}]*background: rgb\(0 0 0 \/ 0\.07\);[^}]*\}/,
    )
    expect(componentSource).toMatch(
      /\.sidebar-header-collapsed \.sidebar-collapse-toggle\s*\{[^}]*cursor: e-resize;/,
    )
    expect(componentSource).toContain(":global([dir='rtl']) .sidebar-collapse-toggle")
    expect(componentSource).toContain(
      ":global([dir='rtl']) .sidebar-header-collapsed .sidebar-collapse-toggle",
    )
  })

  it('uses matching compact and expanded widths inside a full-height desktop rail', () => {
    expect(componentSource).toContain(
      "? 'w-[60px] lg:w-[68px]'\n        : 'w-[min(84vw,288px)] lg:w-[260px]'",
    )
    expect(componentSource).toContain('top: 0;')
    expect(componentSource).toContain('top: var(--app-shell-top-offset);')
    expect(componentSource).not.toContain('top: 5.0625rem;')
    expect(componentSource).toContain('@apply px-2 py-3;')
    expect(componentSource).toContain('min-height: 2.25rem;')
    expect(componentSource).toContain('padding-left: 0.75rem;')
    expect(componentSource).toContain('padding-right: 0.75rem;')
    expect(componentSource).toContain('min-height: 2.75rem;')
    expect(componentSource).toContain('right: auto;')
    expect(componentSource.match(/\n {4}bottom: 0;/g)).toHaveLength(2)
    expect(componentSource.match(/\n {4}left: 0;/g)).toHaveLength(2)
    expect(componentSource.match(/\n {4}border-radius: 0;/g)).toHaveLength(2)
    expect(componentSource).toContain('border-width: 0 1px 0 0;')
    expect(componentSource).toContain('height: 3.25rem;')
    expect(componentSource).toContain('flex: 0 0 3.25rem;')
    expect(componentSource).toContain('.sidebar-header-collapsed {')
    expect(componentSource).toContain('padding-right: 0;')
    expect(componentSource).toContain('padding-left: 0;')
    expect(componentSource).toContain("{ 'sidebar-mobile-hidden': !mobileOpen }")
    expect(componentSource).toContain('transform: translateX(-100%);')
  })

  it('removes the off-canvas mobile navigation from keyboard and screen-reader access', () => {
    expect(componentSource).toContain(
      "const mobileNavigationHidden = computed(() => mobileViewport.value && !mobileOpen.value)",
    )
    expect(componentSource).toContain(":aria-hidden=\"mobileNavigationHidden ? 'true' : undefined\"")
    expect(componentSource).toContain(':inert="mobileNavigationHidden"')
    expect(componentSource).toContain("window.matchMedia('(max-width: 1023px)')")
  })

  it('keeps every mobile sidebar link at least 44px tall', () => {
    const mobileRules = componentSource.slice(componentSource.indexOf('@media (max-width: 1023px)'))

    expect(mobileRules).toMatch(
      /\.sidebar-link,[\s\S]*?min-height: 2\.75rem;/,
    )
  })

  it('keeps the mobile overlay and configurable glass fallback above it', () => {
    expect(componentSource).toContain(
      'class="fixed inset-0 z-30 border-0 bg-gray-950/5 p-0 backdrop-blur-[1px] lg:hidden dark:bg-black/20"',
    )
    expect(componentSource).not.toContain('sidebar-mobile-overlay')
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
      'color: var(--app-shell-sidebar-active-color, rgb(13 13 13));',
    )
    expect(componentSource).toContain(
      'background: var(--app-shell-sidebar-active-bg, rgb(0 0 0 / 0.05));',
    )
    expect(componentSource).toContain(
      'color: var(--app-shell-sidebar-active-icon, rgb(13 13 13));',
    )
    expect(componentSource).toContain(
      'display: var(--app-shell-sidebar-active-marker, none);',
    )
    expect(componentSource).toContain('width: 2px;')
    expect(componentSource).toContain('height: 1rem;')
    expect(componentSource).toContain('font-size: 0.75rem;')
    expect(componentSource).toContain('font-weight: 400;')
  })

  it('pins expandable group chevrons to the trailing edge', () => {
    expect(componentSource).toContain('.sidebar-label-flex {')
    expect(componentSource).toContain('flex: 1 1 auto;')
    expect(componentSource).toContain('justify-content: space-between;')
  })

  it('uses a logical trailing edge and RTL mirror for destination jump cues', () => {
    expect(componentSource).toContain("trailingIcon: 'destinationArrowUpRight'")
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
    const dock = componentSource.indexOf('<SidebarAccountDock />')

    expect(support).toBeGreaterThan(navStart)
    expect(support).toBeLessThan(navEnd)
    expect(navEnd).toBeGreaterThan(-1)
    expect(dock).toBeGreaterThan(navEnd)
    expect(componentSource).toContain('data-testid="sidebar-announcements"')
    expect(componentSource).toContain("'sidebar-docs-tutorial'")
    expect(componentSource).toContain("'sidebar-contact-us'")
    expect(componentSource).not.toContain('data-testid="sidebar-help-resources"')
    expect(componentSource).toContain('sidebarSupportLinks')
    expect(componentSource).toContain("import SidebarAccountDock from './SidebarAccountDock.vue'")
    expect(componentSource).not.toContain('toggleTheme')
    expect(componentSource).not.toContain("t('nav.lightMode')")
    expect(componentSource).not.toContain("t('nav.darkMode')")
  })
})

describe('AppSidebar account destination ownership', () => {
  it('surfaces task-oriented account destinations in the personal workspace', () => {
    expect(componentSource).toContain('ACCOUNT_DESTINATION_PATHS')
    expect(componentSource).toContain('ACCOUNT_DESTINATION_PATHS.quotaViewer')
    expect(componentSource).toContain('ACCOUNT_DESTINATION_PATHS.subscriptions')
    expect(componentSource).toContain('ACCOUNT_DESTINATION_PATHS.wallet')
    expect(componentSource).toContain('ACCOUNT_DESTINATION_PATHS.orders')
    expect(componentSource).toContain('USER_BILLING_PATHS')
    expect(componentSource).toContain("item.path !== ACCOUNT_DESTINATION_PATHS.profile")
    expect(componentSource).not.toContain('data-testid="sidebar-destination-links"')
  })
})
