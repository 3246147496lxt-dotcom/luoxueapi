import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const layoutDirectory = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const layoutSource = readFileSync(resolve(layoutDirectory, 'AppLayout.vue'), 'utf8')
const headerSource = readFileSync(resolve(layoutDirectory, 'AppHeader.vue'), 'utf8')
const sidebarSource = readFileSync(resolve(layoutDirectory, 'AppSidebar.vue'), 'utf8')
const brandSource = readFileSync(resolve(layoutDirectory, 'AppBrand.vue'), 'utf8')

describe('authenticated mixed application shell palette', () => {
  it('keeps the canonical fallback while flattening every light authenticated workspace', () => {
    expect(layoutSource).toContain('app-layout app-layout--snow-shell')
    expect(layoutSource).toContain("'app-layout--flat-workspace-shell': isFlatWorkspaceShell")
    expect(layoutSource).toContain("'app-layout--admin-shell': isAdminShell")
    expect(layoutSource).toContain(
      "() => props.variant === 'home-clay' || route.meta.requiresAdmin === true",
    )
    expect(layoutSource).toContain(
      '() => isAdminShell.value || route.meta.requiresAuth === true',
    )
    expect(layoutSource).toContain("isAdminShell && 'app-layout--home-clay'")
    expect(layoutSource).toContain("variant === 'chat' && 'app-layout--chat'")
    expect(layoutSource).toContain('background: var(--app-shell-canvas, var(--lx-clay-canvas)) !important;')
    expect(layoutSource).toContain(':global(html:not(.dark) .app-layout--snow-shell.app-layout--flat-workspace-shell)')
    expect(layoutSource).toContain('--app-shell-canvas: #ffffff;')
    expect(layoutSource).toContain('--app-shell-sidebar-bg: #f7f7f8;')
    expect(layoutSource).toContain('--app-shell-sidebar-border: #e5e7eb;')
    expect(layoutSource).toContain('--app-shell-sidebar-hover-color: #0d0d0d;')
    expect(layoutSource).toContain('--app-shell-sidebar-hover-bg: rgb(0 0 0 / 0.05);')
    expect(layoutSource).toContain('--app-shell-sidebar-active-color: #0d0d0d;')
    expect(layoutSource).toContain('--app-shell-sidebar-active-icon: #0d0d0d;')
    expect(layoutSource).toContain('--app-shell-sidebar-active-bg: rgb(0 0 0 / 0.05);')
    expect(layoutSource).toContain('--app-shell-sidebar-active-marker: none;')
    expect(layoutSource).toContain("const FLAT_WORKSPACE_BODY_CLASS = 'app-flat-workspace-active'")
    expect(layoutSource).toContain(':global(html:not(.dark) body.app-flat-workspace-active)')
    expect(layoutSource).toContain('background: #ffffff;')
    expect(layoutSource).not.toContain(
      ':global(html:not(.dark) .app-layout--snow-shell.app-layout--admin-shell)',
    )
    expect(layoutSource).toContain('font-family: var(--lx-clay-font-ui);')
    expect(layoutSource).not.toContain('#f5f7fb')
    expect(layoutSource).not.toContain('#0f1115')
  })

  it('keeps the header on the original production glass and typography contract', () => {
    expect(headerSource).toContain('class="app-header fixed left-0 right-0 top-0')
    expect(headerSource).toContain("? 'lg:left-[68px]'")
    expect(headerSource).toContain(
      ": 'lg:left-[184px] min-[1025px]:left-[196px] min-[1281px]:left-[208px]'",
    )
    expect(headerSource).toContain('font-family: system-ui, -apple-system')
    expect(headerSource).toContain('border-width: 1px 1px 0;')
    expect(headerSource).toContain(
      'background: linear-gradient(rgb(248 251 255 / 0.32), rgb(235 242 252 / 0.1));',
    )
    expect(headerSource).toContain('backdrop-filter: saturate(1.7) blur(36px);')
    expect(headerSource).toContain(
      'linear-gradient(rgb(10 12 18 / 0.92), rgb(8 10 16 / 0.82))',
    )
    expect(headerSource).toContain('data-testid="header-mobile-menu"')
    expect(headerSource).not.toContain('data-testid="header-sidebar-toggle"')
    expect(headerSource).not.toContain('SidebarCollapseIcon')
    expect(headerSource).not.toContain('background: var(--lx-clay-surface-elevated);')
  })

  it('keeps the original sidebar as fallback while the flat workspace removes its glass', () => {
    expect(sidebarSource).toContain('class="sidebar"')
    expect(sidebarSource).toContain(
      'w-44 lg:w-[184px] min-[1025px]:w-[196px] min-[1281px]:w-[208px]',
    )
    expect(sidebarSource).toContain('<AppBrand placement="sidebar" :collapsed="sidebarCollapsed" />')
    expect(sidebarSource).toContain('data-testid="sidebar-collapse-toggle"')
    expect(sidebarSource).toContain('data-testid="sidebar-collapse-toggle-icon"')
    expect(sidebarSource).toContain('top: 0;')
    expect(sidebarSource).toContain('border-radius: 0;')
    expect(sidebarSource).toContain('box-shadow: none;')
    expect(sidebarSource).not.toMatch(/\.sidebar-header\s*\{[^}]*border-bottom\s*:/)
    expect(sidebarSource).toContain('--app-shell-sidebar-bg,')
    expect(sidebarSource).toContain('linear-gradient(rgb(248 251 255 / 0.32), rgb(235 242 252 / 0.1))')
    expect(sidebarSource).toContain('--app-shell-sidebar-backdrop, saturate(1.7) blur(36px)')
    expect(sidebarSource).toContain('--app-shell-sidebar-active-color, rgb(13 13 13)')
    expect(sidebarSource).toContain('--app-shell-sidebar-active-marker, none')
    expect(sidebarSource).toContain(':global(.dark .sidebar)')
    expect(sidebarSource).not.toContain('sidebar--snow-clay')
  })

  it('keeps the Luoxue mark readable in the dark sidebar', () => {
    expect(brandSource).toContain(':global(html.dark .app-brand-logo-image-luoxue)')
    expect(brandSource).not.toContain(':global(html.dark) .app-brand-logo-image-luoxue')
  })
})
