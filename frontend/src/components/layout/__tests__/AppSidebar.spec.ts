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
  it('exposes a stable aria target without duplicating the header brand or collapse control', () => {
    expect(componentSource).toContain('id="app-sidebar"')
    expect(componentSource).toContain('aria-label="Sidebar"')
    expect(componentSource).not.toContain('sidebar-header')
    expect(componentSource).not.toContain('sidebar-brand')
    expect(componentSource).not.toContain('VersionBadge')
    expect(componentSource).not.toContain('sidebar-footer')
    expect(componentSource).not.toContain('toggleSidebar')
  })

  it('uses compact and expanded widths inside a floating rounded desktop shell', () => {
    expect(componentSource).toContain("? 'w-[60px]'\n        : 'w-44 min-[1025px]:w-[188px] min-[1281px]:w-[200px]'")
    expect(componentSource.match(/top: 5\.0625rem;/g)).toHaveLength(2)
    expect(componentSource).toContain('@apply px-2 py-3;')
    expect(componentSource).toContain('min-height: 2rem;')
    expect(componentSource).toContain('padding-left: 0.75rem;')
    expect(componentSource).toContain('padding-right: 0.75rem;')
    expect(componentSource).toContain('min-height: 2.75rem;')
    expect(componentSource).toContain('right: auto;')
    expect(componentSource.match(/bottom: 1rem;/g)).toHaveLength(2)
    expect(componentSource).toContain('left: 0.5rem;')
    expect(componentSource).toContain('border-radius: 1rem;')
    expect(componentSource).toContain("{ 'sidebar-mobile-hidden': !mobileOpen }")
    expect(componentSource).toContain('transform: translateX(calc(-100% - 0.5rem));')
  })

  it('matches the reference menu rhythm and selected state', () => {
    expect(componentSource).toContain('gap: 0.625rem;')
    expect(componentSource).toContain('padding-top: 0.25rem;')
    expect(componentSource).toContain('line-height: 1.5rem;')
    expect(componentSource).toContain('background: rgb(234 245 255);')
    expect(componentSource).toContain('width: 2px;')
    expect(componentSource).toContain('height: 1rem;')
    expect(componentSource).toContain('font-size: 0.75rem;')
    expect(componentSource).toContain('font-weight: 400;')
  })
})

describe('AppSidebar utility actions', () => {
  it('does not render or manage the theme toggle', () => {
    expect(componentSource).not.toContain('toggleTheme')
    expect(componentSource).not.toContain("t('nav.lightMode')")
    expect(componentSource).not.toContain("t('nav.darkMode')")
  })
})
