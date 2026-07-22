import { readFileSync, readdirSync } from 'node:fs'
import { dirname, extname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const stylesDirectory = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const sourceDirectory = resolve(stylesDirectory, '..')
const frontendDirectory = resolve(sourceDirectory, '..')

const readFrontendFile = (path: string): string => readFileSync(resolve(frontendDirectory, path), 'utf8')

function collectRuntimeSources(directory: string): string[] {
  return readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const path = resolve(directory, entry.name)
    if (entry.isDirectory()) return collectRuntimeSources(path)
    return ['.css', '.js', '.ts', '.vue'].includes(extname(entry.name)) ? [path] : []
  })
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
})
