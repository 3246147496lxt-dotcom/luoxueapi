import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const sourceDirectory = resolve(dirname(fileURLToPath(import.meta.url)), '../..')
const frontendDirectory = resolve(sourceDirectory, '..')

const readSource = (path: string) => readFileSync(resolve(sourceDirectory, path), 'utf8')

const tokenSource = readSource('styles/luoxue-clay-tokens.css')
const globalStyleSource = readSource('style.css')
const tailwindConfig = readFileSync(resolve(frontendDirectory, 'tailwind.config.js'), 'utf8')
const layoutSource = readSource('components/layout/AppLayout.vue')
const frameSource = readSource('components/layout/WorkspaceSidebarFrame.vue')
const sidebarSource = readSource('components/layout/AppSidebar.vue')
const mobileHeaderSource = readSource('components/layout/AppMobileHeader.vue')
const accountOverlaySource = readSource('components/layout/SidebarAccountOverlay.vue')
const modelSettingsSource = readSource('components/chat/ChatModelSettings.vue')
const selectSource = readSource('components/common/Select.vue')
const baseDialogSource = readSource('components/common/BaseDialog.vue')
const emptyStateSource = readSource('components/common/EmptyState.vue')
const chatViewSource = readSource('views/user/ChatView.vue')
const dashboardSource = readSource('views/user/DashboardView.vue')
const keysSource = readSource('views/user/KeysView.vue')
const batchImageGuideSource = readSource('views/user/BatchImageGuideView.vue')

describe('authenticated theme surface contract', () => {
  it('exposes one semantic surface hierarchy to CSS and Tailwind consumers', () => {
    for (const declaration of [
      '--workspace-light-sidebar-surface: var(--workspace-light-canvas);',
      '--workspace-dark-sidebar-surface: var(--workspace-dark-canvas);',
      '--workspace-canvas: var(--workspace-light-canvas);',
      '--workspace-sidebar-surface: var(--workspace-light-sidebar-surface);',
      '--workspace-card-surface: var(--workspace-light-surface);',
      '--workspace-popup-surface: var(--workspace-light-popup-surface);',
      '--workspace-hover: var(--workspace-light-hover);',
      '--workspace-selected: var(--workspace-light-selected);',
      '--workspace-divider: var(--workspace-light-divider);',
      '--workspace-footer-divider: var(--workspace-light-footer-divider);',
      '--workspace-selection-background: var(--workspace-light-selection-background);',
      '--workspace-sidebar-overlay-backdrop: var(--workspace-light-sidebar-overlay-backdrop);',
      '--workspace-sidebar-overlay-shadow: var(--workspace-light-sidebar-overlay-shadow);',
      '--workspace-canvas: var(--workspace-dark-canvas);',
      '--workspace-sidebar-surface: var(--workspace-dark-sidebar-surface);',
      '--workspace-card-surface: var(--workspace-dark-surface);',
      '--workspace-popup-surface: var(--workspace-dark-popup-surface);',
      '--workspace-hover: var(--workspace-dark-hover);',
      '--workspace-selected: var(--workspace-dark-active);',
      '--workspace-divider: var(--workspace-dark-divider);',
      '--workspace-footer-divider: var(--workspace-dark-footer-divider);',
      '--workspace-selection-background: var(--workspace-dark-selection-background);',
      '--workspace-sidebar-overlay-backdrop: var(--workspace-dark-sidebar-overlay-backdrop);',
      '--workspace-sidebar-overlay-shadow: var(--workspace-dark-sidebar-overlay-shadow);',
    ]) {
      expect(tokenSource).toContain(declaration)
    }

    for (const mapping of [
      "canvas: 'var(--workspace-canvas)'",
      "sidebar: 'var(--workspace-sidebar-surface)'",
      "card: 'var(--workspace-card-surface)'",
      "popup: 'var(--workspace-popup-surface)'",
      "hover: 'var(--workspace-hover)'",
      "selected: 'var(--workspace-selected)'",
      "divider: 'var(--workspace-divider)'",
      "border: 'var(--workspace-border)'",
      "text: 'var(--workspace-text)'",
      "'text-secondary': 'var(--workspace-text-secondary)'",
      "muted: 'var(--workspace-text-muted)'",
    ]) {
      expect(tailwindConfig).toContain(mapping)
    }
  })

  it('keeps personal Workspace text selection neutral without changing public or admin selection', () => {
    expect(globalStyleSource).toMatch(
      /body\.app-flat-workspace-active:not\(\.admin-home-clay-portals\)::selection,\s*body\.app-flat-workspace-active:not\(\.admin-home-clay-portals\) ::selection\s*\{[^}]*color: currentColor;[^}]*background: var\(--workspace-selection-background\);/s,
    )
    expect(globalStyleSource).toMatch(
      /\/\* 选中文本样式 \*\/\s*::selection\s*\{[^}]*@apply bg-primary-500\/20 text-primary-900 dark:text-primary-100;/s,
    )
    expect(batchImageGuideSource).not.toMatch(
      /(?:dark:)?selection:(?:bg|text)-primary-/,
    )
  })

  it('keeps Chat, Work, and API Key states on the same Workspace canvas', () => {
    expect(globalStyleSource).toMatch(
      /html\.dark:has\(> body\.app-flat-workspace-active:not\(\.admin-home-clay-portals\)\)\s*\{[^}]*background: var\(--workspace-canvas\);/s,
    )
    expect(chatViewSource).toMatch(
      /\.chat-workspace__main\s*\{[^}]*background: var\(--workspace-canvas\);/s,
    )
    expect(chatViewSource).not.toMatch(
      /\.chat-workspace__main--new-chat\s*\{[^}]*background:/s,
    )
    expect(dashboardSource).toMatch(
      /\.yunwu-dashboard\s*\{[^}]*background: var\(--workspace-canvas\);/s,
    )
    expect(keysSource).toMatch(
      /\.keys-content\s*\{[^}]*background: var\(--workspace-canvas\);/s,
    )
    expect(keysSource).toMatch(
      /\.keys-master-pane\s*\{[^}]*background: var\(--workspace-canvas\);/s,
    )

    for (const source of [tokenSource, layoutSource, chatViewSource, dashboardSource, keysSource]) {
      expect(source).not.toMatch(
        /--workspace-(?:chat|work)-(?:canvas|surface|surface-subtle|border|border-strong|divider)(?:\s*:|\))/,
      )
    }
  })

  it('routes the user shell through semantic colors without changing its geometry', () => {
    for (const bridge of [
      '--lx-clay-canvas: var(--workspace-canvas);',
      '--lx-clay-surface: var(--workspace-card-surface);',
      '--lx-clay-surface-elevated: var(--workspace-popup-surface);',
      '--lx-clay-sidebar: var(--workspace-sidebar-surface);',
      '--lx-clay-recessed-strong: var(--workspace-selected);',
      '--lx-clay-text: var(--workspace-text);',
      '--lx-clay-text-secondary: var(--workspace-text-secondary);',
      '--lx-clay-text-muted: var(--workspace-text-muted);',
      '--lx-clay-border: var(--workspace-border);',
      '--lx-clay-hover: var(--workspace-hover);',
      '--lx-clay-selected: var(--workspace-selected);',
    ]) {
      expect(layoutSource).toContain(bridge)
    }

    expect(frameSource).toContain('border-right: 1px solid var(--workspace-divider);')
    expect(frameSource).toContain('border-top: 1px solid var(--workspace-footer-divider);')
    expect(frameSource).toContain('background: var(--workspace-sidebar-surface);')
    expect(frameSource).toContain('width: var(--workspace-sidebar-width);')
    expect(frameSource).toContain('width: var(--workspace-sidebar-width-collapsed);')
    expect(sidebarSource).toContain(
      'background: var(--app-shell-sidebar-active-bg, var(--workspace-selected));',
    )
    expect(sidebarSource).toContain(
      'background: var(--app-shell-sidebar-hover-bg, var(--workspace-hover));',
    )
    expect(mobileHeaderSource).toContain(
      'background: var(--app-shell-sidebar-bg, var(--workspace-sidebar-surface));',
    )
    expect(mobileHeaderSource).toContain(
      'border-bottom: 1px solid var(--app-shell-sidebar-border, var(--workspace-divider));',
    )
    expect(layoutSource).toContain(
      'padding: var(--workspace-space-8) var(--workspace-space-7) var(--workspace-space-12);',
    )
  })

  it('themes dialogs, account popovers, and Select dropdowns from semantic roles', () => {
    expect(accountOverlaySource).toContain('background: var(--workspace-popover-surface);')
    expect(accountOverlaySource).toContain('border: 1px solid var(--workspace-popover-border);')
    expect(accountOverlaySource).toContain('background: var(--workspace-popover-hover);')

    expect(modelSettingsSource).toContain('--chat-settings-surface: var(--workspace-popup-surface);')
    expect(modelSettingsSource).toContain('--chat-settings-divider: var(--workspace-divider);')
    expect(modelSettingsSource).toContain('--chat-settings-hover: var(--workspace-hover);')
    expect(modelSettingsSource).toContain('var(--workspace-popover-shadow);')

    expect(selectSource).toContain('background: var(--lx-clay-surface-elevated);')
    expect(selectSource).toContain('border-color: var(--lx-clay-border);')
    expect(selectSource).toContain('background: var(--lx-clay-hover);')
    expect(selectSource).toContain('background: var(--lx-clay-selected);')
    expect(selectSource).not.toMatch(
      /\b(?:bg|text|border)-(?:white|gray-\d+|dark-\d+)|dark:(?:bg|text|border)-dark-\d+/,
    )

    expect(baseDialogSource).toContain("class=\"modal-overlay\"")
    expect(baseDialogSource).toContain("class=\"modal-content\"")
    expect(baseDialogSource).toMatch(/:class="\[\s*widthClasses,/)
    expect(globalStyleSource).toMatch(
      /\.modal-content\s*\{[^}]*border-color: var\(--lx-clay-border\);[^}]*color: var\(--lx-clay-text\);[^}]*background: var\(--lx-clay-surface-elevated\);/s,
    )
    expect(globalStyleSource).toMatch(
      /\.dropdown\s*\{[^}]*border-color: var\(--lx-clay-border\);[^}]*color: var\(--lx-clay-text\);[^}]*background: var\(--lx-clay-surface-elevated\);/s,
    )
  })

  it('keeps empty and API-key states neutral while preserving their interaction layout', () => {
    expect(emptyStateSource).toContain('bg-[var(--lx-clay-surface-subtle)]')
    expect(globalStyleSource).toMatch(
      /\.empty-state-title\s*\{[^}]*color: var\(--lx-clay-text\);/s,
    )
    expect(globalStyleSource).toMatch(
      /\.empty-state-description\s*\{[^}]*color: var\(--lx-clay-text-secondary\);/s,
    )

    for (const declaration of [
      'color: var(--workspace-text);',
      'background: var(--workspace-canvas);',
      'background: var(--workspace-card-surface);',
      'background: var(--workspace-popup-surface);',
      'border: 1px solid var(--workspace-border);',
      'background: var(--workspace-hover);',
      'color: var(--workspace-text-secondary);',
      'color: var(--workspace-text-muted);',
    ]) {
      expect(keysSource).toContain(declaration)
    }
    expect(keysSource).toContain('height: 56px;')
    expect(keysSource).toContain('min-height: 44px;')
    expect(keysSource).not.toMatch(
      /#(?:fafafa|f9fafb|f5f5f5|111827|374151|6b7280|e5e7eb|0f1115|17191f|20232b|353535|afafaf)\b/i,
    )
  })
})
