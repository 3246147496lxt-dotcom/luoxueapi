import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const directory = dirname(fileURLToPath(import.meta.url))
const routerSource = readFileSync(resolve(directory, '../index.ts'), 'utf8')

describe('desktop authorization route', () => {
  it('keeps the approval page behind the normal web login', () => {
    expect(routerSource).toContain("path: '/desktop/authorize'")
    expect(routerSource).toContain("name: 'DesktopAuthorize'")
    expect(routerSource).toContain("component: () => import('@/views/user/DesktopAuthorizeView.vue')")
    expect(routerSource).toContain("titleKey: 'desktopAuthorization.pageTitle'")
  })
})
