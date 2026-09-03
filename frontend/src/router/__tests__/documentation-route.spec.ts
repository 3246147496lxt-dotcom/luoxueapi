import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const directory = dirname(fileURLToPath(import.meta.url))
const routerSource = readFileSync(resolve(directory, '../routes/admin.ts'), 'utf8')
const sidebarSource = readFileSync(resolve(directory, '../../components/layout/AppSidebar.vue'), 'utf8')
const viewSource = readFileSync(resolve(directory, '../../views/admin/DocumentationView.vue'), 'utf8')

describe('documentation management navigation', () => {
  it('registers an admin-only editor route and keeps it in the admin sidebar', () => {
    expect(routerSource).toContain("path: '/admin/documentation'")
    expect(routerSource).toContain("name: 'AdminDocumentation'")
    expect(routerSource).toContain("component: () => import('@/views/admin/DocumentationView.vue')")
    expect(routerSource).toContain("titleKey: 'admin.documentation.title'")
    expect(sidebarSource).toContain("path: '/admin/documentation'")
    expect(sidebarSource).toContain("label: t('nav.documentationManagement')")
    expect(viewSource).toContain('onBeforeRouteLeave')
    expect(viewSource).toContain("t('admin.documentation.leaveDirtyConfirm')")
  })
})
