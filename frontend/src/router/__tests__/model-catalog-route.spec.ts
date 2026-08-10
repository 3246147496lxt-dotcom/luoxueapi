import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const directory = dirname(fileURLToPath(import.meta.url))
const publicSource = readFileSync(resolve(directory, '../routes/public.ts'), 'utf8')
const userSource = readFileSync(resolve(directory, '../routes/user.ts'), 'utf8')
const adminSource = readFileSync(resolve(directory, '../routes/admin.ts'), 'utf8')

describe('model catalog routes', () => {
  it('reuses the public catalog in an authenticated Work shell and keeps management separate', () => {
    expect(publicSource).toContain("path: '/models.html'")
    expect(publicSource).toContain("name: 'PublicModelCatalog'")
    expect(publicSource).toContain("component: () => import('@/views/public/ModelCatalogView.vue')")

    expect(userSource).toContain("path: '/models'")
    expect(userSource).toContain("name: 'UserModelCatalog'")
    expect(userSource).toContain("component: () => import('@/views/public/ModelCatalogView.vue')")
    expect(userSource).toContain('props: { embedded: true }')
    expect(userSource).toContain('requiresAuth: true')

    expect(adminSource).toContain("path: '/admin/model-catalog'")
    expect(adminSource).toContain("name: 'AdminModelCatalog'")
    expect(adminSource).toContain("component: () => import('@/views/admin/ModelCatalogView.vue')")
  })
})
