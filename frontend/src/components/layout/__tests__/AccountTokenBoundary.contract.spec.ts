import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const layoutDirectory = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const tokenPath = resolve(layoutDirectory, '../../styles/luoxue-clay-tokens.css')
const tokenSource = readFileSync(tokenPath, 'utf8')
const accountFiles = [
  'SidebarAccountDock.vue',
  'SidebarAccountOverlay.vue',
  'UserAccountCard.vue',
  'AccountMenuIcon.vue',
]

function readAccountSource(file: string) {
  return readFileSync(resolve(layoutDirectory, file), 'utf8')
}

function readStyleBlocks(source: string) {
  return [...source.matchAll(/<style\b[^>]*>([\s\S]*?)<\/style>/g)]
    .map((match) => match[1])
    .join('\n')
}

describe('Account Workspace token boundary', () => {
  it('keeps every Account color in the canonical Workspace token source', () => {
    const canonicalTokens = new Set(
      [...tokenSource.matchAll(/(--workspace-[a-z0-9-]+)\s*:/g)].map((match) => match[1]),
    )

    for (const file of accountFiles) {
      const source = readAccountSource(file)
      const styles = readStyleBlocks(source)
      const tokenReferences = [...styles.matchAll(/var\(\s*(--workspace-[a-z0-9-]+)/g)]
        .map((match) => match[1])

      expect(styles, `${file} must not own color literals`).not.toMatch(
        /#[0-9a-f]{3,8}\b|\b(?:rgb|rgba|hsl|hsla|color-mix)\s*\(/i,
      )
      expect(styles, `${file} must only consume Workspace custom properties`).not.toMatch(
        /var\(\s*--(?!workspace-)[a-z0-9-]+/i,
      )
      expect(styles, `${file} must not shadow a canonical Workspace token`).not.toMatch(
        /--workspace-[a-z0-9-]+\s*:/i,
      )
      expect(source, `${file} must not revive the retired Account namespace`).not.toMatch(
        /--account-[a-z0-9-]+/i,
      )

      for (const token of tokenReferences) {
        expect(canonicalTokens, `${file} references undefined ${token}`).toContain(token)
      }
    }
  })

  it('routes the approved Account geometry through Workspace metrics', () => {
    const dockSource = readAccountSource('SidebarAccountDock.vue')
    const overlaySource = readAccountSource('SidebarAccountOverlay.vue')

    expect(dockSource).toContain('min-height: var(--workspace-sidebar-footer-row-height);')
    expect(dockSource).toContain('width: var(--workspace-avatar-size-md);')
    expect(dockSource).toContain('border-radius: var(--workspace-radius-button);')
    expect(overlaySource).toContain('width: var(--workspace-popover-width);')
    expect(overlaySource).toContain('border-radius: var(--workspace-radius-popover);')
    expect(overlaySource).toContain('min-height: var(--workspace-menu-row-height);')
    expect(overlaySource).toContain('width: var(--workspace-avatar-size-sm);')
    expect(overlaySource).toContain('width: var(--workspace-avatar-size-sheet);')

    expect(tokenSource).toContain('--workspace-popover-width: 248px;')
    expect(tokenSource).toContain('--workspace-avatar-size-sm: var(--workspace-space-6);')
    expect(tokenSource).toContain('--workspace-avatar-size-md: var(--workspace-space-8);')
    expect(tokenSource).toContain('--workspace-avatar-size-sheet: 42px;')
    expect(tokenSource).toContain('--workspace-menu-row-height: 36px;')
    expect(tokenSource).toContain('--workspace-menu-row-height-touch: 44px;')
    expect(tokenSource).toContain('--workspace-popover-identity-height: 51px;')
    expect(tokenSource).toContain('--workspace-sidebar-footer-row-height: var(--workspace-sidebar-header-height);')
    expect(tokenSource).toContain('--workspace-type-secondary-size: 12px;')
    expect(tokenSource).toContain('--workspace-type-body-weight: 400;')
    expect(tokenSource).toContain(
      '--workspace-font-ui: Inter, "PingFang SC", "Microsoft YaHei", sans-serif;',
    )
    expect(tokenSource).not.toContain('--workspace-font-native:')
    expect(overlaySource).not.toContain('font-family:')
  })

  it('maps the Account palette to the shared light and dark semantic roles', () => {
    const exactTokenDeclarations = [
      '--workspace-overlay-backdrop: rgb(15 23 42 / 0.32);',
      '--workspace-identity-avatar-text: #1c1f23;',
      '--workspace-identity-avatar-surface: #fce865;',
      '--workspace-dock-text: var(--workspace-text-secondary);',
      '--workspace-dock-text-strong: var(--workspace-text);',
      '--workspace-dock-text-muted: var(--workspace-text-secondary);',
      '--workspace-dock-hover: var(--workspace-hover);',
      '--workspace-menu-surface: var(--workspace-popup-surface);',
      '--workspace-menu-divider: var(--workspace-divider);',
      '--workspace-menu-text: var(--workspace-text-secondary);',
      '--workspace-menu-row-text: var(--workspace-text-secondary);',
      '--workspace-menu-text-strong: var(--workspace-text);',
      '--workspace-menu-text-muted: var(--workspace-text-secondary);',
      '--workspace-menu-hover: var(--workspace-hover);',
      '--workspace-menu-danger: rgb(220 38 38);',
      '--workspace-dark-popover-surface: var(--workspace-dark-popup-surface);',
      '--workspace-dark-popover-text-secondary: #b4b4b4;',
      '--workspace-menu-danger: rgb(248 113 113);',
    ]

    for (const declaration of exactTokenDeclarations) {
      expect(tokenSource).toContain(declaration)
    }
  })
})
