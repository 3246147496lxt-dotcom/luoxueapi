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
const marketplaceSource = readFileSync(
  resolve(directory, '../../views/public/SkillMarketplaceView.vue'),
  'utf8',
)
const detailSource = readFileSync(
  resolve(directory, '../../views/public/SkillDetailView.vue'),
  'utf8',
)

describe('Skill marketplace navigation', () => {
  it('registers the catalog and detail as authenticated Work routes', () => {
    expect(publicRouteSource).not.toContain("path: '/skills'")
    expect(userRouteSource).toContain("path: '/skills'")
    expect(userRouteSource).toContain("name: 'SkillMarket'")
    expect(userRouteSource).toContain("component: () => import('@/views/public/SkillMarketplaceView.vue')")
    expect(userRouteSource).toContain("path: '/skills/:slug'")
    expect(userRouteSource.match(/requiresSkillMarketplace: true/g)).toHaveLength(2)
    expect(marketplaceSource).toContain('<AppLayout class="skill-market-page">')
    expect(detailSource).toContain('<AppLayout class="skill-detail-page">')
    expect(marketplaceSource).not.toContain('<PublicSiteLayout')
    expect(detailSource).not.toContain('<PublicSiteLayout')
    expect(guardSource).not.toContain("'/legal', '/skills'")
  })

  it('keeps public and admin entries behind their intended boundaries', () => {
    expect(sidebarSource).toContain("FeatureFlags.skillMarketplace")
    expect(sidebarSource).toContain("path: '/admin/skills'")
    expect(publicLayoutSource).toContain('skillMarketEntryVisible')
    expect(publicLayoutSource).toContain('skill_marketplace_enabled === true')
  })

  it('revalidates the flag instead of trusting embedded HTML forever', () => {
    expect(guardSource).toContain('fetchPublicSettings(true)')
    expect(guardSource).toContain('refreshedSettings?.skill_marketplace_enabled !== true')
    expect(guardSource).toContain("next('/dashboard')")
  })
})
