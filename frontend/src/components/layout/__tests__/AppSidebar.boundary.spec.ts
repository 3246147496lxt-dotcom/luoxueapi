import { readdirSync, readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const directory = dirname(fileURLToPath(import.meta.url))
const sidebarSource = readFileSync(resolve(directory, '../AppSidebar.vue'), 'utf8')
const layoutSource = readFileSync(resolve(directory, '../AppLayout.vue'), 'utf8')
const loaderSource = readFileSync(resolve(directory, '../sidebar/adminNavigationLoader.ts'), 'utf8')
const userNavigationSource = readFileSync(resolve(directory, '../sidebar/userNavigation.ts'), 'utf8')
const adminNavigationSource = readFileSync(resolve(directory, '../sidebar/adminNavigation.ts'), 'utf8')
const userViewsDirectory = resolve(directory, '../../../views/user')

function collectVueSources(root: string): Array<{ file: string, source: string }> {
  return readdirSync(root, { withFileTypes: true }).flatMap((entry) => {
    const entryPath = resolve(root, entry.name)

    if (entry.isDirectory()) {
      return collectVueSources(entryPath)
    }

    if (!entry.isFile() || !entry.name.endsWith('.vue')) {
      return []
    }

    return [{ file: entryPath, source: readFileSync(entryPath, 'utf8') }]
  })
}

describe('sidebar workspace module boundaries', () => {
  it('keeps the personal-workspace static graph free of administrator navigation and stores', () => {
    expect(sidebarSource).toContain("import { buildUserNavigation } from './sidebar/userNavigation'")
    expect(sidebarSource).toContain("import { loadAdminNavigation } from './sidebar/adminNavigationLoader'")
    expect(sidebarSource).not.toMatch(/^import .*sidebar\/adminNavigation['"]/m)
    expect(sidebarSource).not.toMatch(/^import .*stores\/adminSettings['"]/m)
    expect(loaderSource).toContain("import('./adminNavigation')")
    expect(layoutSource).toContain('loadAdminNavigation()')
  })

  it('owns user and administrator route definitions in separate modules', () => {
    expect(userNavigationSource).toContain("path: '/dashboard'")
    expect(userNavigationSource).not.toContain("path: '/admin/dashboard'")
    expect(adminNavigationSource).toContain("path: '/admin/dashboard'")
    expect(adminNavigationSource).not.toContain("path: '/dashboard'")
  })

  it('prevents user pages from overriding the shared Work sidebar width or main offset', () => {
    const violations = collectVueSources(userViewsDirectory).flatMap(({ file, source }) => {
      const rules = [
        { label: '#app-sidebar selector', pattern: /#app-sidebar/ },
        { label: 'sidebar collapse selector', pattern: /\[data-sidebar-collapsed/ },
        {
          label: 'app-main-shell margin override',
          pattern: /:global\([^)]*\.app-main-shell[^)]*\)\s*\{[^}]*margin-left\s*:/s,
        },
      ]

      return rules
        .filter(({ pattern }) => pattern.test(source))
        .map(({ label }) => `${file}: ${label}`)
    })

    expect(violations).toEqual([])
  })
})
