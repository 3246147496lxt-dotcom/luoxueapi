import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const directory = dirname(fileURLToPath(import.meta.url))
const routerSource = readFileSync(resolve(directory, '../routes/admin.ts'), 'utf8')

describe('admin dashboard home-style preview route', () => {
  it('keeps the visual experiment local and leaves the real admin route in place', () => {
    expect(routerSource).toContain('const designPreviewRoutes: RouteRecordRaw[] = import.meta.env.DEV')
    expect(routerSource).toContain("path: '/design-preview/admin-dashboard-home'")
    expect(routerSource).toContain("component: () => import('@/views/design-preview/AdminDashboardHomePreviewView.vue')")
    expect(routerSource).toContain('requiresAuth: false')
    expect(routerSource).toContain("path: '/admin/dashboard'")
    expect(routerSource).toContain("component: () => import('@/views/admin/DashboardView.vue')")
  })
})
