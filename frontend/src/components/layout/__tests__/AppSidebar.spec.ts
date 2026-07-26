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
      /\.sidebar-header :deep\(\.app-brand-logo-image-luoxue\)\s*\{[^}]*transform: scale\(1\.6\);/,
    )
  })

  it('uses matching compact and expanded widths inside a full-height desktop rail', () => {
    expect(componentSource).toContain(
      "? 'w-[60px] lg:w-[68px]'\n        : 'w-44 lg:w-[184px] min-[1025px]:w-[196px] min-[1281px]:w-[208px]'",
    )
    expect(componentSource).toContain('top: 0;')
    expect(componentSource.match(/top: 5\.0625rem;/g)).toHaveLength(1)
    expect(componentSource).toContain('@apply px-2 py-3;')
    expect(componentSource).toContain('min-height: 2rem;')
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
})

describe('AppSidebar utility actions', () => {
  it('does not render or manage the theme toggle', () => {
    expect(componentSource).not.toContain('toggleTheme')
    expect(componentSource).not.toContain("t('nav.lightMode')")
    expect(componentSource).not.toContain("t('nav.darkMode')")
  })
})

describe('AppSidebar pinned destinations', () => {
  it('keeps the destination block outside the scrolling navigation region', () => {
    const navEnd = componentSource.indexOf('</nav>')
    const destinations = componentSource.indexOf('data-testid="sidebar-destination-links"')

    expect(navEnd).toBeGreaterThan(-1)
    expect(destinations).toBeGreaterThan(navEnd)
    expect(componentSource).toContain('.sidebar-destination-links {')
    expect(componentSource).toContain('flex: 0 0 auto;')
  })

  it('matches the compact reference rhythm and keeps a visible keyboard focus state', () => {
    expect(componentSource).toContain('min-height: 2.25rem;')
    expect(componentSource).toContain('padding: 0.5rem 0.75rem;')
    expect(componentSource).toContain('border-radius: 0.625rem;')
    expect(componentSource).toContain('font-size: 0.8125rem;')
    expect(componentSource).toContain('.sidebar-destination-link:focus-visible')
  })

  it('uses the tokenized blue hover contract and recolors both destination glyphs', () => {
    expect(componentSource).toContain(
      'color: var(--app-shell-sidebar-hover-color, rgb(0 132 255));',
    )
    expect(componentSource).toContain(
      'background: var(--app-shell-sidebar-hover-bg, rgb(0 132 255 / 0.08));',
    )
    expect(componentSource).toContain(
      'outline: 2px solid var(--app-shell-sidebar-focus, rgb(0 132 255 / 0.5));',
    )
    expect(componentSource).toContain('.sidebar-destination-label {')
    expect(componentSource).toContain('color: inherit;')
  })

  it('disables the destination lift when reduced motion is requested', () => {
    expect(componentSource).toMatch(
      /@media \(prefers-reduced-motion: reduce\)[\s\S]*\.sidebar-destination-leading > :deep\(svg\)[\s\S]*transition-duration:\s*0\.01ms/,
    )
    expect(componentSource).toMatch(
      /@media \(prefers-reduced-motion: reduce\)[\s\S]*\.sidebar-destination-link:hover,[\s\S]*transform:\s*none/,
    )
  })
})
