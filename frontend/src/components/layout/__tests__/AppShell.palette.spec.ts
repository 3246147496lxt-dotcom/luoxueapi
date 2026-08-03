import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const layoutDirectory = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const layoutSource = readFileSync(resolve(layoutDirectory, 'AppLayout.vue'), 'utf8')
const mobileHeaderSource = readFileSync(resolve(layoutDirectory, 'AppMobileHeader.vue'), 'utf8')
const sidebarSource = readFileSync(resolve(layoutDirectory, 'AppSidebar.vue'), 'utf8')
const accountDockSource = readFileSync(resolve(layoutDirectory, 'SidebarAccountDock.vue'), 'utf8')
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
    expect(layoutSource).toContain('--app-shell-sidebar-bg: #fcfcfc;')
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

  it('keeps the top chrome mobile-only and shares the shell offset token', () => {
    expect(layoutSource).toContain('<AppMobileHeader />')
    expect(mobileHeaderSource).toContain('class="app-mobile-header lg:hidden"')
    expect(mobileHeaderSource).toContain('height: var(--app-shell-top-offset);')
    expect(mobileHeaderSource).toContain(
      'background: color-mix(in srgb, var(--app-shell-canvas, #fff) 94%, transparent);',
    )
    expect(mobileHeaderSource).toContain('backdrop-filter: blur(18px) saturate(1.25);')
    expect(mobileHeaderSource).toContain('data-testid="mobile-header-menu"')
    expect(mobileHeaderSource).not.toContain('SidebarCollapseIcon')
    expect(layoutSource).not.toContain('AppHeader')
  })

  it('keeps the original sidebar as fallback while the flat workspace removes its glass', () => {
    expect(sidebarSource).toContain('class="sidebar"')
    expect(sidebarSource).toContain(
      'w-[min(84vw,288px)] lg:w-[260px]',
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
    expect(sidebarSource).toContain('<SidebarAccountDock />')
    expect(accountDockSource).toContain('padding: 0 6px calc(6px + env(safe-area-inset-bottom)) 8px;')
    expect(accountDockSource).toContain(':global(html.dark .sidebar-account-row)')
  })

  it('scales the canonical default logo without retaining legacy brand styles', () => {
    expect(brandSource).toMatch(
      /\.app-brand-logo-image-default\s*\{[^}]*transform: scale\(1\.1\);/,
    )
    expect(brandSource).not.toContain('app-brand-logo-image-luoxue')
  })
})
