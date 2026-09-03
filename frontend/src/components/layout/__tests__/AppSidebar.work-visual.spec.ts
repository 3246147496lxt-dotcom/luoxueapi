import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const directory = dirname(fileURLToPath(import.meta.url))
const sidebarSource = readFileSync(resolve(directory, '../AppSidebar.vue'), 'utf8')
const layoutSource = readFileSync(resolve(directory, '../AppLayout.vue'), 'utf8')
const modeSwitchSource = readFileSync(resolve(directory, '../AppModeSwitch.vue'), 'utf8')
const brandSource = readFileSync(resolve(directory, '../WorkspaceSidebarBrand.vue'), 'utf8')
const sidebarHeaderSource = readFileSync(resolve(directory, '../WorkspaceSidebarHeader.vue'), 'utf8')
const accountDockSource = readFileSync(resolve(directory, '../SidebarAccountDock.vue'), 'utf8')
const chatHistorySource = readFileSync(resolve(directory, '../../chat/ChatHistoryPanel.vue'), 'utf8')
const workspaceTokens = readFileSync(
  resolve(directory, '../../../styles/luoxue-clay-tokens.css'),
  'utf8',
)

describe('Work sidebar visual scope', () => {
  it('applies the approved neutral navigation palette only to the personal Work rail', () => {
    expect(sidebarSource).toContain("'sidebar--personal-work': isPersonalWorkWorkspace")
    expect(layoutSource).toContain(
      ':global(html:not(.dark) .app-layout--snow-shell.app-layout--personal-work-shell)',
    )
    expect(layoutSource).toContain(
      '--app-shell-sidebar-hover-bg: var(--workspace-hover);',
    )
    expect(layoutSource).toContain('--app-shell-sidebar-active-color: var(--workspace-work-text);')
    expect(layoutSource).toContain('--app-shell-sidebar-active-icon: var(--workspace-work-text);')
    expect(layoutSource).toContain('--app-shell-sidebar-active-bg: var(--workspace-selected);')
    expect(layoutSource).toContain('--app-shell-sidebar-active-marker: none;')
    expect(workspaceTokens).toContain('--workspace-light-mode-switch-track: #f7f7f8;')
    expect(workspaceTokens).toContain('--workspace-light-work-text: #0d0d0d;')
    expect(workspaceTokens).toContain('--workspace-light-hover: #ececec;')
    expect(workspaceTokens).toContain('--workspace-light-selected: #e5e5e5;')
    expect(sidebarSource).not.toMatch(/--workspace-[a-z0-9_-]+\s*:/)
  })

  it('keeps the Work shell dimensions and dark palette scoped to the personal rail', () => {
    expect(layoutSource).toContain(
      ':global(html.dark .app-layout--snow-shell.app-layout--personal-work-shell)',
    )
    expect(layoutSource).toContain('--app-shell-sidebar-bg: var(--workspace-sidebar-surface);')
    expect(layoutSource).toContain('--app-shell-sidebar-hover-bg: var(--workspace-hover);')
    expect(layoutSource).toContain('--app-shell-sidebar-active-color: var(--workspace-text);')
    expect(layoutSource).toContain('--app-shell-sidebar-active-icon: var(--workspace-text);')
    expect(layoutSource).toContain('--app-shell-sidebar-active-bg: var(--workspace-selected);')
    expect(modeSwitchSource).toMatch(
      /\.app-mode-switch\s*\{[^}]*height: var\(--workspace-mode-switch-height\);[^}]*flex: 0 0 var\(--workspace-mode-switch-height\);/s,
    )
    expect(modeSwitchSource).not.toContain('font-family:')
    expect(modeSwitchSource).toContain('font-size: var(--workspace-type-navigation-size);')
    expect(modeSwitchSource).toContain('font-weight: var(--workspace-type-navigation-weight);')
    expect(workspaceTokens).toContain('--workspace-mode-switch-height: 46px;')
    expect(workspaceTokens).toContain(
      '--workspace-mode-switch-track: var(--workspace-surface-subtle);',
    )
    expect(workspaceTokens).toContain('--workspace-dark-surface-subtle: #171717;')
    expect(workspaceTokens).toContain('--workspace-dark-mode-switch-active: #212121;')
    expect(workspaceTokens).toContain('--workspace-dark-hover: #212121;')
    expect(workspaceTokens).toContain('--workspace-dark-active: #212121;')
    expect(modeSwitchSource).toMatch(
      /\.app-mode-switch::after\s*\{[^}]*height: 1px;[^}]*background: var\(--workspace-mode-switch-divider\);/s,
    )
    expect(modeSwitchSource).toMatch(
      /\.app-mode-switch__option--active,[\s\S]*?box-shadow: var\(--workspace-mode-switch-active-shadow\);/,
    )
    expect(sidebarSource).not.toMatch(/:deep\(\s*\.app-mode-switch/)
    expect(sidebarHeaderSource).toMatch(
      /\.workspace-sidebar-header\s*\{[^}]*height: var\(--workspace-sidebar-header-height\);[^}]*flex: 0 0 var\(--workspace-sidebar-header-height\);/s,
    )
    expect(workspaceTokens).toContain('--workspace-sidebar-header-height: 52px;')
    expect(sidebarSource).toMatch(
      /\.sidebar--personal-work \.sidebar-link-active\s*\{[^}]*font-size: var\(--workspace-type-navigation-size\);[^}]*font-weight: var\(--workspace-type-navigation-weight\);/,
    )
    expect(sidebarSource).toContain('<WorkspaceSidebarBrand')
    expect(sidebarSource).toContain('v-if="isPersonalWorkWorkspace"')
    expect(sidebarSource).toContain('home-path="/dashboard"')
    expect(sidebarSource).not.toContain(
      '.sidebar--personal-work .workspace-sidebar-brand--work :deep(.app-brand-logo-frame)',
    )
    expect(brandSource).toMatch(
      /\.workspace-sidebar-brand__copy strong,\s*\.workspace-sidebar-brand__copy span\s*\{[^}]*font-size: 18px;[^}]*font-weight: 600;[^}]*letter-spacing: -0\.27px;[^}]*line-height: 26px;/s,
    )
    expect(brandSource).toContain("const brandName = '落雪AI'")
  })

  it('keeps one shared personal account identity treatment in Chat and Work', () => {
    expect(accountDockSource).toContain(
      "'sidebar-account-dock--work': context === 'work' && !isAdminWorkspace",
    )
    expect(accountDockSource).not.toContain(
      'html:not(.dark) .sidebar-account-dock--work .sidebar-account-trigger__avatar',
    )
    expect(accountDockSource).not.toContain(
      'html:not(.dark) .sidebar-account-dock--work .sidebar-account-trigger__meta',
    )
    expect(accountDockSource).toMatch(
      /\.sidebar-account-dock--personal \.sidebar-account-trigger__avatar\s*\{[^}]*width: 24px;[^}]*height: 24px;/s,
    )
    expect(accountDockSource).toMatch(
      /\.sidebar-account-dock--personal \.sidebar-account-trigger__name\s*\{[^}]*font-size: 14px;[^}]*font-weight: 400;[^}]*line-height: 20px;/s,
    )
    expect(accountDockSource).toMatch(
      /\.sidebar-account-dock--personal \.sidebar-account-trigger__meta\s*\{[^}]*font-size: 12px;[^}]*font-weight: 400;[^}]*line-height: 16px;/s,
    )
    expect(chatHistorySource).toContain('<template v-if="shell" #footer>')
    expect(chatHistorySource).toContain(
      '<UserAccountCard context="chat" :collapsed="sidebarCollapsed" />',
    )
    expect(chatHistorySource).toMatch(
      /<WorkspaceSidebarHeader[\s\S]*?<template #brand>[\s\S]*?<WorkspaceSidebarBrand home-path="\/chat" \/>/,
    )
    expect(chatHistorySource).not.toContain('<AppBrand')
    expect(chatHistorySource).not.toContain('<ChatSidebarBrand')
  })
})
