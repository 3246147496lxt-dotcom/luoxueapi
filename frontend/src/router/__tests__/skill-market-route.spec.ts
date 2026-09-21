import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const directory = dirname(fileURLToPath(import.meta.url))
const publicRouteSource = readFileSync(resolve(directory, '../routes/public.ts'), 'utf8')
const userRouteSource = readFileSync(resolve(directory, '../routes/user.ts'), 'utf8')
const guardSource = readFileSync(resolve(directory, '../guards.ts'), 'utf8')
const sidebarSource = readFileSync(resolve(directory, '../../components/layout/AppSidebar.vue'), 'utf8')
const publicLayoutSource = readFileSync(
  resolve(directory, '../../components/public/PublicSiteLayout.vue'),
  'utf8',
)
const externalLinksSource = readFileSync(
  resolve(directory, '../../navigation/externalLinks.ts'),
  'utf8',
)

describe('Skill marketplace navigation', () => {
  it('redirects the legacy catalog and detail routes to skills.sh', () => {
    expect(publicRouteSource).not.toContain("path: '/skills'")
    expect(userRouteSource).toContain("path: '/skills'")
    expect(userRouteSource).toContain("name: 'SkillMarket'")
    expect(userRouteSource).toContain("path: '/skills/:slug'")
    expect(userRouteSource.match(/beforeEnter: redirectToSkillsMarket/g)).toHaveLength(2)
    expect(userRouteSource).not.toContain('requiresSkillMarketplace: true')
    expect(externalLinksSource).toContain("https://www.skills.sh/")
  })

  it('keeps the admin skill tooling while exposing the external user link', () => {
    expect(sidebarSource).toContain("path: '/admin/skills'")
    expect(publicLayoutSource).toContain('SKILLS_MARKET_URL')
    expect(publicLayoutSource).not.toContain('skillMarketEntryVisible')
  })

  it('keeps the marketplace guard available for unrelated feature-gated routes', () => {
    expect(guardSource).toContain('fetchPublicSettings(true)')
  })
})
