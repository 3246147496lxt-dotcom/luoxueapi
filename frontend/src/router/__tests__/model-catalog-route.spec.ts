import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const source = readFileSync(resolve(dirname(fileURLToPath(import.meta.url)), '../index.ts'), 'utf8')

describe('model catalog routes', () => {
  it('keeps the public page on models.html and reserves an admin-only management route', () => {
    expect(source).toContain("path: '/models.html'")
    expect(source).toContain("path: '/admin/model-catalog'")
    expect(source).toContain("name: 'AdminModelCatalog'")
    expect(source).toContain("component: () => import('@/views/admin/ModelCatalogView.vue')")
  })
})
