import { readFileSync, readdirSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const frontendDirectory = resolve(dirname(fileURLToPath(import.meta.url)), '../../..')
const readFrontendFile = (path: string): string => (
  readFileSync(resolve(frontendDirectory, path), 'utf8')
)

function collectVueSources(directory: string): Array<{ file: string; source: string }> {
  return readdirSync(resolve(frontendDirectory, directory), { withFileTypes: true }).flatMap((entry) => {
    const relativePath = `${directory}/${entry.name}`
    if (entry.isDirectory()) return collectVueSources(relativePath)
    if (!entry.name.endsWith('.vue')) return []
    return [{ file: relativePath, source: readFrontendFile(relativePath) }]
  })
}

const workspaceSources = [
  ...collectVueSources('src/components/chat'),
  ...collectVueSources('src/components/payment'),
  ...collectVueSources('src/components/user/dashboard'),
  ...collectVueSources('src/views/user'),
  ...[
    'src/components/layout/AppLayout.vue',
    'src/components/layout/WorkspaceSidebarFrame.vue',
    'src/components/layout/WorkspaceSidebarBrand.vue',
    'src/components/layout/AppSidebar.vue',
    'src/components/layout/AppModeSwitch.vue',
    'src/components/layout/AppBrand.vue',
    'src/components/layout/AppMobileHeader.vue',
    'src/components/layout/SidebarAccountDock.vue',
    'src/components/layout/SidebarAccountOverlay.vue',
    'src/components/layout/WalletSubscriptionSettingsPanel.vue',
  ].map((file) => ({ file, source: readFrontendFile(file) })),
]

describe('Workspace typography contract', () => {
  it('exposes one UI font entry and exactly six semantic type roles', () => {
    const tokens = readFrontendFile('src/styles/luoxue-clay-tokens.css')
    const expectedRoles = new Map([
      ['brand', ['18px', '700']],
      ['page-title', ['28px', '600']],
      ['navigation', ['14px', '500']],
      ['body', ['14px', '400']],
      ['secondary', ['12px', '400']],
      ['numeric', ['32px', '600']],
    ])

    expect(tokens).toContain(
      '--workspace-font-ui: Inter, "PingFang SC", "Microsoft YaHei", sans-serif;',
    )
    expect(tokens).toContain('--workspace-font-mono: var(--lx-clay-font-mono);')

    const declaredRoles = [
      ...tokens.matchAll(/--workspace-type-([a-z-]+)-(size|weight):\s*([^;]+);/g),
    ]
    expect(new Set(declaredRoles.map((match) => match[1]))).toEqual(
      new Set(expectedRoles.keys()),
    )

    for (const [role, [size, weight]] of expectedRoles) {
      expect(tokens).toContain(`--workspace-type-${role}-size: ${size};`)
      expect(tokens).toContain(`--workspace-type-${role}-weight: ${weight};`)
    }
  })

  it('applies the UI stack once at the personal Workspace body boundary', () => {
    const globalStyles = readFrontendFile('src/style.css')
    const tailwind = readFrontendFile('tailwind.config.js')

    expect(globalStyles).toMatch(
      /body\.app-flat-workspace-active:not\(\.admin-home-clay-portals\)\s*\{[^}]*font-family:\s*var\(--workspace-font-ui\);/s,
    )
    expect(tailwind).toContain("workspace: ['var(--workspace-font-ui)']")
  })

  it.each(workspaceSources)('$file does not create a second UI font stack', ({ source }) => {
    const familyDeclarations = source.match(/font-family:\s*[^;]+;/g) ?? []
    const technicalOrInheritedFamilies = familyDeclarations.filter((declaration) => (
      declaration.includes('var(--workspace-font-mono)')
      || declaration.includes('ui-monospace')
      || declaration === 'font-family: inherit;'
    ))

    expect(familyDeclarations).toEqual(technicalOrInheritedFamilies)
    expect(source).not.toMatch(/--workspace-font-(?:native|content|navigation|dashboard|account|composer)/)
  })

  it.each(workspaceSources)('$file uses only standard Workspace font weights', ({ source }) => {
    const literalWeights = [...source.matchAll(/font-weight:\s*(\d+)\b/g)]
      .map((match) => Number(match[1]))

    expect(literalWeights.every((weight) => [400, 500, 600, 700].includes(weight))).toBe(true)
    expect(source).not.toMatch(/\bfont-(?:black|extrabold)\b/)
  })
})
