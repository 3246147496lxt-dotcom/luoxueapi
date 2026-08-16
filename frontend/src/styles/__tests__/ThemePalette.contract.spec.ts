import { readFileSync, readdirSync } from 'node:fs'
import { dirname, extname, resolve, sep } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const stylesDirectory = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const sourceDirectory = resolve(stylesDirectory, '..')
const frontendDirectory = resolve(sourceDirectory, '..')

const readFrontendFile = (path: string): string => readFileSync(resolve(frontendDirectory, path), 'utf8')

const workspaceTokenSourcePath = resolve(stylesDirectory, 'luoxue-clay-tokens.css')

function collectRuntimeSources(directory: string): string[] {
  return readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const path = resolve(directory, entry.name)
    if (entry.isDirectory()) return collectRuntimeSources(path)
    return ['.css', '.js', '.ts', '.vue'].includes(extname(entry.name)) ? [path] : []
  })
}

function isProductionSource(path: string): boolean {
  return !path.includes(`${sep}__tests__${sep}`) && !/\.(?:spec|test)\.[jt]s$/.test(path)
}

const retiredTeal = /#(?:f0fdfa|ccfbf1|99f6e4|5eead4|2dd4bf|14b8a6|0d9488|0f766e|115e59|134e4a|042f2e|ecfeff|cffafe|a5f3fc|67e8f9|22d3ee|06b6d4|0891b2|0e7490|155e75|164e63|083344)|rgba?\((?:20,?\s+184,?\s+166|6,?\s+182,?\s+212)|rgb\(15\s+118\s+110/i

describe('Snow Clay palette contract', () => {
  it('maps every legacy primary utility to the canonical violet ramp', () => {
    const tailwindConfig = readFrontendFile('tailwind.config.js')

    expect(tailwindConfig).toMatch(/500:\s*'#8b5cf6'/)
    expect(tailwindConfig).toMatch(/600:\s*'#7c3aed'/)
    expect(tailwindConfig).toMatch(/800:\s*'#5b21b6'/)
    expect(tailwindConfig).toContain(
      "'gradient-primary': 'linear-gradient(135deg, #8b5cf6 0%, #5b21b6 100%)'",
    )
    expect(tailwindConfig).not.toMatch(retiredTeal)
  })

  it('routes public and authentication shells through canonical runtime tokens', () => {
    const publicLayout = readFrontendFile('src/components/public/PublicSiteLayout.vue')
    const authLayout = readFrontendFile('src/components/layout/AuthLayout.vue')

    expect(publicLayout).toContain('--page: var(--lx-clay-canvas);')
    expect(publicLayout).toContain('--accent: var(--lx-clay-accent);')
    expect(publicLayout).toContain('--action-violet-start: var(--lx-clay-light-accent);')
    expect(authLayout).toContain('--auth-canvas: var(--lx-clay-canvas);')
    expect(authLayout).toContain('--auth-violet: var(--lx-clay-accent);')
    expect(authLayout).toContain('--auth-blue: var(--lx-clay-info);')
    expect(`${publicLayout}\n${authLayout}`).not.toMatch(retiredTeal)
  })

  it('keeps information, success, and payment brands separate from primary violet', () => {
    const legacyComponents = readFrontendFile('src/style.css')
    const snowComponents = readFrontendFile('src/styles/luoxue-clay-components.css')

    expect(legacyComponents).toMatch(/\.toast-info\s*\{[^}]*var\(--lx-clay-info\)/s)
    expect(snowComponents).toMatch(/\.badge-info\s*\{[^}]*var\(--lx-clay-info-deep\)[^}]*var\(--lx-clay-info-soft\)/s)
    expect(snowComponents).toContain('.badge-secondary {')
    expect(snowComponents).toMatch(/\.badge-success\s*\{[^}]*var\(--lx-clay-success/s)
    expect(legacyComponents).toContain('@apply bg-[#635bff] text-white')
    expect(legacyComponents).toContain('@apply bg-[#00AEEF] text-white')
    expect(legacyComponents).toContain('@apply bg-[#2BB741] text-white')
  })

  it('does not reintroduce retired teal values or teal/cyan utility roles at runtime', () => {
    const runtimeSource = collectRuntimeSources(sourceDirectory)
      .filter((path) => !path.endsWith('luoxue-clay-tokens.css'))
      .map((path) => readFileSync(path, 'utf8'))
      .join('\n')

    expect(runtimeSource).not.toMatch(retiredTeal)
    expect(runtimeSource).not.toMatch(/\b(?:teal|cyan)-/)
  })

  it('keeps authenticated Workspace tokens in one runtime source', () => {
    const tokenSource = readFileSync(workspaceTokenSourcePath, 'utf8')
    const sharedTokenNames = new Set(
      [...tokenSource.matchAll(/(--workspace-[a-z0-9-]+)\s*:/g)].map((match) => match[1]),
    )
    const productionFiles = collectRuntimeSources(sourceDirectory).filter(
      (path) => isProductionSource(path) && path !== workspaceTokenSourcePath,
    )
    const tailwindConfigPath = resolve(frontendDirectory, 'tailwind.config.js')

    for (const path of [...productionFiles, tailwindConfigPath]) {
      const source = readFileSync(path, 'utf8')
      const shadowedSharedTokens = [...source.matchAll(/(--workspace-[a-z0-9-]+)\s*:/g)]
        .map((match) => match[1])
        .filter((token) => sharedTokenNames.has(token))

      expect(path, `${path} must not use a retired Workspace namespace`).not.toMatch(
        /--(?:shell|work|account)-/,
      )
      expect(source, `${path} must not use a retired Workspace namespace`).not.toMatch(
        /--(?:shell|work|account)-/,
      )
      expect(source, `${path} must not add a fallback for a canonical Workspace token`).not.toMatch(
        /var\(\s*--workspace-[^,)]+,/,
      )
      expect(
        shadowedSharedTokens,
        `${path} must consume shared Workspace roles instead of redefining them`,
      ).toEqual([])
    }
  })

  it('locks the ChatGPT-like neutral Workspace palette while consolidating its names', () => {
    const tokenSource = readFileSync(workspaceTokenSourcePath, 'utf8')
    const exactValues = new Map([
      ['--workspace-light-canvas', '#f8fafc'],
      ['--workspace-light-sidebar-surface', '#ffffff'],
      ['--workspace-light-surface', '#ffffff'],
      ['--workspace-light-popup-surface', '#ffffff'],
      ['--workspace-light-surface-subtle', '#f7f7f8'],
      ['--workspace-light-mode-switch-track', '#f7f7f8'],
      ['--workspace-light-hover', '#ececec'],
      ['--workspace-light-selected', '#e5e5e5'],
      ['--workspace-light-divider', 'rgb(0 0 0 / 0.05)'],
      ['--workspace-light-footer-divider', 'rgb(0 0 0 / 0.05)'],
      ['--workspace-light-border', '#e5e5e5'],
      ['--workspace-light-text', '#0d0d0d'],
      ['--workspace-light-identity-text', '#0d0d0d'],
      ['--workspace-light-identity-text-tertiary', '#8f8f8f'],
      ['--workspace-light-work-text', '#0d0d0d'],
      ['--workspace-light-text-secondary', '#5d5d5d'],
      ['--workspace-light-work-text-secondary', '#5d5d5d'],
      ['--workspace-light-text-muted', '#8e8e8e'],
      ['--workspace-light-sidebar-group-label', '#8f8f8f'],
      ['--workspace-light-selection-background', 'color-mix(in oklab, #cdcdcd 40%, transparent)'],
      ['--workspace-light-sidebar-overlay-backdrop', 'rgb(249 250 251 / 0.5)'],
      ['--workspace-light-sidebar-overlay-shadow', '0 0 64px rgb(0 0 0 / 0.07)'],
      ['--workspace-light-work-accent', '#7c3aed'],
      ['--workspace-light-dashboard-card-border', '#f1f5f9'],
      ['--workspace-light-dashboard-text-strong', '#0f172a'],
      ['--workspace-light-dashboard-text-heading', '#475569'],
      ['--workspace-light-dashboard-text-muted', '#64748b'],
      ['--workspace-light-dashboard-text-subtle', '#94a3b8'],
      ['--workspace-light-dashboard-period-text', '#334155'],
      ['--workspace-light-dashboard-divider', '#f8fafc'],
      ['--workspace-light-dashboard-track', '#f1f5f9'],
      ['--workspace-light-dashboard-success', '#22c55e'],
      ['--workspace-dark-canvas', '#000000'],
      ['--workspace-dark-sidebar-surface', 'var(--workspace-dark-canvas)'],
      ['--workspace-dark-surface', '#171717'],
      ['--workspace-dark-popup-surface', '#171717'],
      ['--workspace-dark-surface-subtle', '#171717'],
      ['--workspace-dark-mode-switch-active', '#212121'],
      ['--workspace-dark-popover-surface', 'var(--workspace-dark-popup-surface)'],
      ['--workspace-dark-divider', 'rgb(255 255 255 / 0.1)'],
      ['--workspace-dark-footer-divider', 'rgb(255 255 255 / 0.06)'],
      ['--workspace-dark-border', 'rgb(255 255 255 / 0.1)'],
      ['--workspace-dark-text', '#ececec'],
      ['--workspace-dark-identity-text', '#ffffff'],
      ['--workspace-dark-identity-text-tertiary', '#afafaf'],
      ['--workspace-dark-text-secondary', '#b4b4b4'],
      ['--workspace-dark-text-muted', '#8a8a8a'],
      ['--workspace-dark-sidebar-group-label', '#8f8f8f'],
      ['--workspace-dark-selection-background', 'color-mix(in oklab, #cdcdcd 40%, transparent)'],
      ['--workspace-dark-sidebar-overlay-backdrop', 'rgb(0 0 0 / 0.5)'],
      ['--workspace-dark-sidebar-overlay-shadow', 'none'],
      ['--workspace-dark-hover', '#212121'],
      ['--workspace-dark-active', '#212121'],
      ['--workspace-dark-dashboard-card-border', 'var(--workspace-dark-border)'],
      ['--workspace-dark-dashboard-card-shadow', 'none'],
      ['--workspace-dark-dashboard-text-strong', 'var(--workspace-dark-text)'],
      ['--workspace-dark-dashboard-text-heading', 'var(--workspace-dark-text-secondary)'],
      ['--workspace-dark-dashboard-text-muted', 'var(--workspace-dark-text-muted)'],
      ['--workspace-dark-dashboard-text-subtle', 'var(--workspace-dark-text-muted)'],
      ['--workspace-dark-dashboard-period-text', 'var(--workspace-dark-text)'],
      ['--workspace-dark-dashboard-divider', 'var(--workspace-dark-divider)'],
      ['--workspace-dark-dashboard-track', 'var(--workspace-dark-border)'],
      ['--workspace-radius-compact', '8px'],
      ['--workspace-radius-mode-switch-option', '7px'],
      ['--workspace-radius-work-card', '14px'],
      ['--workspace-radius-card', '16px'],
      ['--workspace-radius-popover', '20px'],
      ['--workspace-space-2', '8px'],
      ['--workspace-space-4', '16px'],
      ['--workspace-space-4-5', '18px'],
      ['--workspace-space-6', '24px'],
      ['--workspace-sidebar-width', '260px'],
      ['--workspace-sidebar-width-collapsed', '68px'],
      ['--workspace-popover-width', '248px'],
      ['--workspace-sidebar-header-height', '52px'],
      ['--workspace-sidebar-header-padding-inline', 'var(--workspace-space-3)'],
      ['--workspace-sidebar-header-actions-inset-end', 'var(--workspace-space-1-75)'],
      ['--workspace-sidebar-action-size', '36px'],
      ['--workspace-sidebar-touch-target', '44px'],
      ['--workspace-mode-switch-height', '46px'],
      ['--workspace-mode-switch-height-mobile', '60px'],
      ['--workspace-type-brand-size', '18px'],
      ['--workspace-type-brand-weight', '700'],
      ['--workspace-type-page-title-size', '28px'],
      ['--workspace-type-page-title-weight', '600'],
      ['--workspace-type-navigation-size', '14px'],
      ['--workspace-type-navigation-weight', '500'],
      ['--workspace-type-body-size', '14px'],
      ['--workspace-type-body-weight', '400'],
      ['--workspace-type-secondary-size', '12px'],
      ['--workspace-type-secondary-weight', '400'],
      ['--workspace-type-numeric-size', '32px'],
      ['--workspace-type-numeric-weight', '600'],
      ['--workspace-sidebar-group-label-size', '14px'],
      ['--workspace-sidebar-group-label-weight', '500'],
      ['--workspace-sidebar-group-label-line-height', '20px'],
    ])

    for (const [token, value] of exactValues) {
      expect(tokenSource).toMatch(new RegExp(`${token}:\\s*${value.replace(/[()]/g, '\\$&')};`))
    }

    expect(tokenSource).toContain('--workspace-sidebar-width-mobile: min(84vw, 288px);')
    expect(tokenSource).toContain(
      '--workspace-sidebar-width-mobile-chat: min(288px, calc(100vw - 40px));',
    )
    expect(tokenSource).not.toContain('--workspace-chat-sidebar-width-collapsed')
    for (const retiredBackgroundToken of [
      '--workspace-work-canvas',
      '--workspace-work-surface',
      '--workspace-work-surface-subtle',
      '--workspace-light-work-surface-subtle',
      '--workspace-dark-work-surface-strong',
    ]) {
      expect(tokenSource).not.toContain(`${retiredBackgroundToken}:`)
    }
    expect(tokenSource).toContain(
      '--workspace-font-ui: Inter, "PingFang SC", "Microsoft YaHei", sans-serif;',
    )
    expect(tokenSource).toContain('--workspace-font-mono: var(--lx-clay-font-mono);')
    for (const alias of [
      '--workspace-canvas: var(--workspace-light-canvas);',
      '--workspace-sidebar-surface: var(--workspace-light-sidebar-surface);',
      '--workspace-card-surface: var(--workspace-light-surface);',
      '--workspace-popup-surface: var(--workspace-light-popup-surface);',
      '--workspace-hover: var(--workspace-light-hover);',
      '--workspace-selected: var(--workspace-light-selected);',
      '--workspace-divider: var(--workspace-light-divider);',
      '--workspace-footer-divider: var(--workspace-light-footer-divider);',
      '--workspace-text: var(--workspace-light-text);',
      '--workspace-identity-text: var(--workspace-light-identity-text);',
      '--workspace-identity-text-tertiary: var(--workspace-light-identity-text-tertiary);',
      '--workspace-text-secondary: var(--workspace-light-text-secondary);',
      '--workspace-text-muted: var(--workspace-light-text-muted);',
      '--workspace-sidebar-group-label: var(--workspace-light-sidebar-group-label);',
      '--workspace-sidebar-header-action: var(--workspace-identity-text-tertiary);',
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
      '--workspace-text: var(--workspace-dark-text);',
      '--workspace-identity-text: var(--workspace-dark-identity-text);',
      '--workspace-identity-text-tertiary: var(--workspace-dark-identity-text-tertiary);',
      '--workspace-text-secondary: var(--workspace-dark-text-secondary);',
      '--workspace-text-muted: var(--workspace-dark-text-muted);',
      '--workspace-sidebar-group-label: var(--workspace-dark-sidebar-group-label);',
      '--workspace-selection-background: var(--workspace-dark-selection-background);',
      '--workspace-sidebar-overlay-backdrop: var(--workspace-dark-sidebar-overlay-backdrop);',
      '--workspace-sidebar-overlay-shadow: var(--workspace-dark-sidebar-overlay-shadow);',
      '--workspace-dashboard-card-border: var(--workspace-light-dashboard-card-border);',
      '--workspace-dashboard-card-shadow: var(--workspace-light-dashboard-card-shadow);',
      '--workspace-dashboard-text-strong: var(--workspace-light-dashboard-text-strong);',
      '--workspace-dashboard-text-heading: var(--workspace-light-dashboard-text-heading);',
      '--workspace-dashboard-text-muted: var(--workspace-light-dashboard-text-muted);',
      '--workspace-dashboard-text-subtle: var(--workspace-light-dashboard-text-subtle);',
      '--workspace-dashboard-period-text: var(--workspace-light-dashboard-period-text);',
      '--workspace-dashboard-card-border: var(--workspace-dark-dashboard-card-border);',
      '--workspace-dashboard-card-shadow: var(--workspace-dark-dashboard-card-shadow);',
      '--workspace-dashboard-text-strong: var(--workspace-dark-dashboard-text-strong);',
      '--workspace-dashboard-text-heading: var(--workspace-dark-dashboard-text-heading);',
      '--workspace-dashboard-text-muted: var(--workspace-dark-dashboard-text-muted);',
      '--workspace-dashboard-text-subtle: var(--workspace-dark-dashboard-text-subtle);',
      '--workspace-dashboard-period-text: var(--workspace-dark-dashboard-period-text);',
    ]) {
      expect(tokenSource).toContain(alias)
    }
    for (const retiredToken of [
      '--workspace-font-content',
      '--workspace-font-navigation',
      '--workspace-font-dashboard',
      '--workspace-font-account',
      '--workspace-font-composer',
      '--workspace-mode-switch-font-size',
      '--workspace-mode-switch-font-weight',
      '--workspace-font-native',
      '--workspace-type-caption-size',
      '--workspace-type-caption-weight',
    ]) {
      expect(tokenSource).not.toContain(`${retiredToken}:`)
    }
  })

  it('keeps legacy Snow Clay neutral aliases on the same light and dark hierarchy', () => {
    const tokenSource = readFileSync(workspaceTokenSourcePath, 'utf8')
    const exactValues = new Map([
      ['--lx-clay-light-canvas', '#ffffff'],
      ['--lx-clay-light-surface', '#ffffff'],
      ['--lx-clay-light-surface-soft', '#f7f7f8'],
      ['--lx-clay-light-surface-subtle', '#f7f7f8'],
      ['--lx-clay-light-surface-elevated', '#ffffff'],
      ['--lx-clay-light-overlay-surface-soft', '#ffffff'],
      ['--lx-clay-light-sidebar', '#f7f7f8'],
      ['--lx-clay-light-recessed', '#f7f7f8'],
      ['--lx-clay-light-recessed-strong', '#e5e5e5'],
      ['--lx-clay-light-text', '#0d0d0d'],
      ['--lx-clay-light-text-secondary', '#5d5d5d'],
      ['--lx-clay-light-text-muted', '#8e8e8e'],
      ['--lx-clay-light-text-subtle', '#8e8e8e'],
      ['--lx-clay-light-border', '#e5e5e5'],
      ['--lx-clay-light-border-strong', '#d4d4d4'],
      ['--lx-clay-light-hover', '#ececec'],
      ['--lx-clay-light-selected', '#e5e5e5'],
      ['--lx-clay-dark-canvas', '#212121'],
      ['--lx-clay-dark-surface', '#2f2f2f'],
      ['--lx-clay-dark-surface-soft', '#2a2a2a'],
      ['--lx-clay-dark-surface-subtle', '#2a2a2a'],
      ['--lx-clay-dark-surface-elevated', '#2f2f2f'],
      ['--lx-clay-dark-overlay-surface-soft', '#2f2f2f'],
      ['--lx-clay-dark-sidebar', '#171717'],
      ['--lx-clay-dark-recessed', '#212121'],
      ['--lx-clay-dark-recessed-strong', '#343434'],
      ['--lx-clay-dark-text', '#ececec'],
      ['--lx-clay-dark-text-secondary', '#b4b4b4'],
      ['--lx-clay-dark-text-muted', '#8e8e8e'],
      ['--lx-clay-dark-text-subtle', '#8e8e8e'],
      ['--lx-clay-dark-border', '#424242'],
      ['--lx-clay-dark-border-strong', '#565656'],
      ['--lx-clay-dark-hover', '#2a2a2a'],
      ['--lx-clay-dark-selected', '#343434'],
    ])

    for (const [token, value] of exactValues) {
      expect(tokenSource).toMatch(new RegExp(`${token}:\\s*${value};`))
    }

    expect(tokenSource).toContain('--lx-clay-hover: var(--lx-clay-light-hover);')
    expect(tokenSource).toContain('--lx-clay-selected: var(--lx-clay-light-selected);')
    expect(tokenSource).toContain('--lx-clay-hover: var(--lx-clay-dark-hover);')
    expect(tokenSource).toContain('--lx-clay-selected: var(--lx-clay-dark-selected);')
  })
})
