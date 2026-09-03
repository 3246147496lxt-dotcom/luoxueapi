import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const directory = dirname(fileURLToPath(import.meta.url))
const routeSource = readFileSync(resolve(directory, '../routes/admin.ts'), 'utf8')
const catalogSource = readFileSync(resolve(directory, '../../views/admin/SkillsView.vue'), 'utf8')
const importSource = readFileSync(resolve(directory, '../../views/admin/SkillImportsView.vue'), 'utf8')

describe('Skill import administration route', () => {
  it('registers the durable importer behind the admin boundary', () => {
    expect(routeSource).toContain("path: '/admin/skills/imports'")
    expect(routeSource).toContain("name: 'AdminSkillImports'")
    expect(routeSource).toContain("component: () => import('@/views/admin/SkillImportsView.vue')")
    expect(routeSource).toContain("titleKey: 'admin.skills.imports.title'")
  })

  it('keeps the catalog and importer in one local Skill workspace', () => {
    expect(catalogSource).toContain('<SkillMarketNav active="catalog" />')
    expect(importSource).toContain('<SkillMarketNav :active="activeSection" />')
    expect(importSource).toContain('concept seed 89f5e4b5')
  })
})
