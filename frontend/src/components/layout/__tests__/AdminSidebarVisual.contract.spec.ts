import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const directory = dirname(fileURLToPath(import.meta.url))
const sidebarSource = readFileSync(resolve(directory, '../AppSidebar.vue'), 'utf8')
const adminThemeSource = readFileSync(resolve(directory, '../AdminClayTheme.css'), 'utf8')

const adminSidebarStart = adminThemeSource.indexOf(' * Administrator sidebar baseline')
const adminSidebarEnd = adminThemeSource.indexOf(
  '/* Page density is explicit.',
  adminSidebarStart,
)
const adminSidebarStyles = adminThemeSource.slice(adminSidebarStart, adminSidebarEnd)

describe('Admin sidebar visual contract', () => {
  it('keeps the visual scope on admin routes without borrowing personal behavior', () => {
    expect(sidebarSource).toContain("'sidebar--admin-workspace': isAdminWorkspace")
    expect(adminSidebarStart).toBeGreaterThan(-1)
    expect(adminSidebarEnd).toBeGreaterThan(adminSidebarStart)
    expect(adminSidebarStyles).toContain('.app-sidebar.sidebar--admin-workspace')
    expect(adminSidebarStyles).not.toContain('.sidebar--personal-work')
    expect(adminSidebarStyles).not.toContain('admin-ops-option-b')
  })

  it('consumes the personal Work typography, rhythm, states, and divider roles', () => {
    expect(adminSidebarStyles).toContain('font-family: var(--workspace-font-ui);')
    expect(adminSidebarStyles).toContain(
      '--app-shell-sidebar-bg: var(--workspace-sidebar-surface);',
    )
    expect(adminSidebarStyles).toContain(
      '--app-shell-sidebar-border: var(--workspace-divider);',
    )
    expect(adminSidebarStyles).toContain(
      '--app-shell-sidebar-hover-bg: var(--workspace-hover);',
    )
    expect(adminSidebarStyles).toContain(
      '--app-shell-sidebar-active-bg: var(--workspace-selected);',
    )
    expect(adminSidebarStyles).toContain(
      'padding: var(--workspace-sidebar-content-padding) !important;',
    )
    expect(adminSidebarStyles).toContain(
      'padding: var(--workspace-sidebar-content-padding-collapsed) !important;',
    )
    expect(adminSidebarStyles).toContain(
      'font-size: var(--workspace-type-navigation-size);',
    )
    expect(adminSidebarStyles).toContain(
      'font-weight: var(--workspace-type-navigation-weight);',
    )
    expect(adminSidebarStyles).toContain(
      'font-size: var(--workspace-type-secondary-size);',
    )
    expect(adminSidebarStyles).toContain(
      'border-top: 1px solid var(--workspace-footer-divider);',
    )
    expect(adminSidebarStyles).toContain('padding-top: var(--workspace-space-2);')
  })

  it('matches the personal account rhythm without overriding the compact rail', () => {
    expect(adminSidebarStyles).toMatch(
      /\.sidebar-account-dock:not\(\.sidebar-account-dock--collapsed\)[\s\S]*?grid-template-columns: 24px minmax\(0, 1fr\) auto;[\s\S]*?gap: var\(--workspace-space-2\);[\s\S]*?padding: var\(--workspace-space-2\);/,
    )
    expect(adminSidebarStyles).toMatch(
      /\.sidebar-account-trigger__name\s*\{[^}]*font-size: var\(--workspace-type-navigation-size\);[^}]*font-weight: var\(--workspace-type-body-weight\);[^}]*line-height: 20px;/s,
    )
    expect(adminSidebarStyles).toMatch(
      /\.sidebar-account-trigger__meta\s*\{[^}]*font-size: var\(--workspace-type-secondary-size\);[^}]*font-weight: var\(--workspace-type-secondary-weight\);[^}]*line-height: 16px;/s,
    )
    expect(adminSidebarStyles).not.toContain(
      '.sidebar-link.sidebar-link-collapsed',
    )
  })

  it('leaves icon geometry and theme primitives to their existing sources', () => {
    expect(adminSidebarStyles).not.toContain('.sidebar-nav-icon')
    expect(adminSidebarStyles).not.toMatch(/\b(?:fill|stroke)\s*:/)
    expect(adminSidebarStyles).not.toMatch(/#[0-9a-f]{3,8}\b/i)
    expect(adminSidebarStyles).not.toMatch(/--workspace-(?:light|dark)-/)
  })
})
