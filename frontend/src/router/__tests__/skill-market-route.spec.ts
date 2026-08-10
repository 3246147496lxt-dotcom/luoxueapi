import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const directory = dirname(fileURLToPath(import.meta.url))
const routeSource = readFileSync(resolve(directory, '../routes/public.ts'), 'utf8')
const guardSource = readFileSync(resolve(directory, '../guards.ts'), 'utf8')
const sidebarSource = readFileSync(resolve(directory, '../../components/layout/AppSidebar.vue'), 'utf8')
const publicLayoutSource = readFileSync(
  resolve(directory, '../../components/public/PublicSiteLayout.vue'),
  'utf8',
)

describe('Skill marketplace navigation', () => {
  it('registers opt-in public catalog and detail routes', () => {
    expect(routeSource).toContain("path: '/skills'")
    expect(routeSource).toContain("name: 'SkillMarket'")
    expect(routeSource).toContain("component: () => import('@/views/public/SkillMarketplaceView.vue')")
    expect(routeSource).toContain("path: '/skills/:slug'")
    expect(routeSource.match(/requiresSkillMarketplace: true/g)).toHaveLength(2)
    expect(guardSource).not.toContain("'/legal', '/skills'")
  })

  it('keeps public and admin entries behind their intended boundaries', () => {
    expect(sidebarSource).toContain("FeatureFlags.skillMarketplace")
    expect(sidebarSource).toContain("path: '/admin/skills'")
    expect(publicLayoutSource).toContain('skillMarketEntryVisible')
    expect(publicLayoutSource).toContain('skill_marketplace_enabled === true')
  })

  it('revalidates the public flag instead of trusting embedded HTML forever', () => {
    expect(guardSource).toContain('fetchPublicSettings(true)')
    expect(guardSource).toContain('refreshedSettings?.skill_marketplace_enabled !== true')
  })
})
