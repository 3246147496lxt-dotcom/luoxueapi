import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const directory = dirname(fileURLToPath(import.meta.url))
const routerSource = readFileSync(resolve(directory, '../routes/admin.ts'), 'utf8')
const navSource = readFileSync(resolve(directory, '../../components/layout/sidebar/adminNavigation.ts'), 'utf8')
const viewSource = readFileSync(resolve(directory, '../../views/admin/AccountAssistantView.vue'), 'utf8')

describe('account assistant navigation', () => {
  it('registers a dedicated admin chat page after the account pool route', () => {
    expect(routerSource).toContain("path: '/admin/account-assistant'")
    expect(routerSource).toContain("name: 'AdminAccountAssistant'")
    expect(routerSource).toContain("component: () => import('@/views/admin/AccountAssistantView.vue')")
    expect(routerSource.indexOf("path: '/admin/accounts'")).toBeLessThan(
      routerSource.indexOf("path: '/admin/account-assistant'")
    )
    expect(navSource).toContain("path: '/admin/account-assistant'")
    expect(navSource).toContain("label: t('nav.accountAssistant')")
    expect(viewSource).toContain('content-mode="workbench"')
    expect(viewSource).toContain('data-admin-page-kind="ops"')
    expect(viewSource).toContain('AccountAssistantWorkspace')
  })
})
