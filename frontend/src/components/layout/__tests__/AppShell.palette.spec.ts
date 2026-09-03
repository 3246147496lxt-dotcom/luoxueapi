import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const layoutDirectory = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const layoutSource = readFileSync(resolve(layoutDirectory, 'AppLayout.vue'), 'utf8')
const globalStyleSource = readFileSync(resolve(layoutDirectory, '../../style.css'), 'utf8')
const mobileHeaderSource = readFileSync(resolve(layoutDirectory, 'AppMobileHeader.vue'), 'utf8')
const sidebarSource = readFileSync(resolve(layoutDirectory, 'AppSidebar.vue'), 'utf8')
const sidebarFrameSource = readFileSync(
  resolve(layoutDirectory, 'WorkspaceSidebarFrame.vue'),
  'utf8',
)
const sidebarHeaderSource = readFileSync(
  resolve(layoutDirectory, 'WorkspaceSidebarHeader.vue'),
  'utf8',
)
const accountDockSource = readFileSync(resolve(layoutDirectory, 'SidebarAccountDock.vue'), 'utf8')
const brandSource = readFileSync(resolve(layoutDirectory, 'AppBrand.vue'), 'utf8')
const workspaceTokens = readFileSync(
  resolve(layoutDirectory, '../../styles/luoxue-clay-tokens.css'),
  'utf8',
)

describe('authenticated mixed application shell palette', () => {
  it('keeps every authenticated shell on the canonical semantic palette', () => {
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
    expect(layoutSource).toContain('--app-shell-canvas: var(--workspace-canvas);')
    expect(layoutSource).toContain('--app-shell-sidebar-bg: var(--workspace-sidebar-surface);')
    expect(layoutSource).toContain('--app-shell-sidebar-border: var(--workspace-divider);')
    expect(layoutSource).toContain('--app-shell-sidebar-hover-color: var(--workspace-text);')
    expect(layoutSource).toContain('--app-shell-sidebar-hover-bg: var(--workspace-hover);')
    expect(layoutSource).toContain('--app-shell-sidebar-active-color: var(--workspace-text);')
    expect(layoutSource).toContain('--app-shell-sidebar-active-icon: var(--workspace-text);')
    expect(layoutSource).toContain('--app-shell-sidebar-active-bg: var(--workspace-selected);')
    expect(layoutSource).toContain('--app-shell-sidebar-active-marker: none;')
    expect(layoutSource).toContain("const FLAT_WORKSPACE_BODY_CLASS = 'app-flat-workspace-active'")
    expect(layoutSource).toContain(
      ':global(body.app-flat-workspace-active:not(.admin-home-clay-portals))',
    )
    expect(layoutSource).toContain('background: var(--workspace-canvas);')
    expect(layoutSource).not.toContain(
      ':global(html:not(.dark) .app-layout--snow-shell.app-layout--admin-shell)',
    )
    expect(layoutSource).not.toContain('font-family: var(--workspace-font-content);')
    expect(globalStyleSource).toMatch(
      /body\.app-flat-workspace-active:not\(\.admin-home-clay-portals\)\s*\{[^}]*font-family: var\(--workspace-font-ui\);/s,
    )
    expect(layoutSource).toContain(
      'padding: var(--workspace-space-8) var(--workspace-space-7) var(--workspace-space-12);',
    )
    expect(layoutSource).toContain('--app-shell-canvas: var(--workspace-canvas);')
    expect(layoutSource).toContain('--app-shell-sidebar-bg: var(--workspace-sidebar-surface);')
    expect(layoutSource).toContain('--app-shell-sidebar-border: var(--workspace-divider);')
    expect(workspaceTokens).toContain(
      '--workspace-font-ui: Inter, "PingFang SC", "Microsoft YaHei", sans-serif;',
    )
    expect(workspaceTokens).toContain('--workspace-space-7: 28px;')
    expect(workspaceTokens).toContain('--workspace-space-8: 32px;')
    expect(workspaceTokens).toContain('--workspace-space-12: 48px;')
    expect(layoutSource).not.toMatch(/#(?:f5f7fb|0f1115|fcfcfc|e5e7eb)\b/i)
  })

  it('keeps the top chrome mobile-only and shares the shell offset token', () => {
    expect(layoutSource).toContain('<AppMobileHeader v-if="!isChatShell" />')
    expect(layoutSource).toContain('<AppSidebar v-if="!isChatShell"')
    expect(layoutSource).toContain("const isChatShell = computed(() => resolvedShellMode.value === 'chat')")
    expect(layoutSource).toMatch(
      /\.app-layout--snow-shell\.app-layout--chat-shell \.app-main-shell\.app-layout--chat \.app-main-content\s*\{\s*padding: 0;\s*\}/,
    )
    expect(mobileHeaderSource).toContain('class="app-mobile-header"')
    expect(mobileHeaderSource).toMatch(
      /@media \(max-width: 767px\) and \(hover: none\) and \(pointer: coarse\)[\s\S]*?\.app-mobile-header\s*\{[^}]*display: block;/,
    )
    expect(mobileHeaderSource).toContain('height: var(--app-shell-top-offset);')
    expect(mobileHeaderSource).toContain(
      'background: var(--app-shell-sidebar-bg, var(--workspace-sidebar-surface));',
    )
    expect(mobileHeaderSource).toContain(
      'border-bottom: 1px solid var(--app-shell-sidebar-border, var(--workspace-divider));',
    )
    expect(mobileHeaderSource).toContain('box-shadow: 0 1px 0 var(--workspace-divider);')
    expect(mobileHeaderSource).toContain('background: var(--workspace-hover);')
    expect(mobileHeaderSource).toContain('backdrop-filter: blur(18px) saturate(1.25);')
    expect(mobileHeaderSource).toContain('data-testid="mobile-header-menu"')
    expect(mobileHeaderSource).not.toContain('SidebarCollapseIcon')
    expect(layoutSource).not.toContain('AppHeader')
  })

  it('keeps sidebar geometry stable while every neutral surface uses semantic roles', () => {
    expect(sidebarSource).toContain('<WorkspaceSidebarFrame')
    expect(sidebarSource).toContain('class="app-sidebar"')
    expect(sidebarSource).not.toContain('class="sidebar"')
    expect(sidebarFrameSource).toContain('width: var(--workspace-sidebar-width);')
    expect(sidebarFrameSource).toContain('width: var(--workspace-sidebar-width-collapsed);')
    expect(sidebarFrameSource).not.toContain('font-family: var(--workspace-font-navigation);')
    expect(workspaceTokens).toContain('--workspace-sidebar-width: 260px;')
    expect(workspaceTokens).toContain('--workspace-sidebar-width-collapsed: 68px;')
    expect(workspaceTokens).toContain('--workspace-sidebar-width-mobile: min(84vw, 288px);')
    expect(workspaceTokens).not.toContain('--workspace-chat-sidebar-width-collapsed')
    expect(sidebarSource).toContain('<WorkspaceSidebarBrand')
    expect(sidebarSource).toContain('v-if="isPersonalWorkWorkspace"')
    expect(sidebarSource).toContain('toggle-test-id="sidebar-collapse-toggle"')
    expect(sidebarSource).toContain('toggle-icon-test-id="sidebar-collapse-toggle-icon"')
    expect(sidebarFrameSource).toContain('top: 0;')
    expect(sidebarSource).toContain('border-radius: 0;')
    expect(sidebarSource).toContain('box-shadow: none;')
    expect(sidebarHeaderSource).not.toMatch(
      /\.workspace-sidebar-header\s*\{[^}]*border-bottom\s*:/,
    )
    expect(sidebarSource).toContain('--app-shell-sidebar-bg,')
    expect(sidebarSource).toContain('--app-shell-sidebar-backdrop, saturate(1.7) blur(36px)')
    expect(sidebarSource).toContain('--app-shell-sidebar-decoration, none')
    expect(sidebarSource).toContain('--app-shell-sidebar-active-color, var(--workspace-text)')
    expect(sidebarSource).toContain('--app-shell-sidebar-active-bg, var(--workspace-selected)')
    expect(sidebarSource).toContain('--app-shell-sidebar-active-marker, none')
    expect(sidebarSource).toContain(':global(.dark .app-sidebar)')
    expect(sidebarSource).not.toContain('sidebar--snow-clay')
    expect(sidebarSource).toContain(
      '<UserAccountCard context="work" :collapsed="sidebarCollapsed" />',
    )
    expect(accountDockSource).toContain('var(--workspace-space-1-5)')
    expect(accountDockSource).toContain('calc(var(--workspace-space-1-5) + env(safe-area-inset-bottom))')
    expect(accountDockSource).toContain('var(--workspace-space-2);')
    expect(accountDockSource).toContain('color: var(--workspace-dock-text);')
    expect(accountDockSource).not.toContain(':global(html.dark .sidebar-account-row)')
    expect(workspaceTokens).toContain('--workspace-sidebar-footer-row-height: var(--workspace-sidebar-header-height);')
    expect(workspaceTokens).toContain('--workspace-dock-text: var(--workspace-text-secondary);')
  })

  it('scales the canonical default logo without retaining legacy brand styles', () => {
    expect(brandSource).toMatch(
      /\.app-brand-logo-image-default\s*\{[^}]*transform: scale\(1\.1\);/,
    )
    expect(brandSource).not.toContain('app-brand-logo-image-luoxue')
  })
})
