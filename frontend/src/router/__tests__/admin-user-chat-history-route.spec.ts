import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const source = readFileSync(resolve(dirname(fileURLToPath(import.meta.url)), '../index.ts'), 'utf8')

describe('admin user chat history route', () => {
  it('is a dedicated JWT-admin route scoped to a positive user id', () => {
    expect(source).toContain("path: '/admin/users/:userId([1-9]\\\\d*)/chat-history'")
    expect(source).toContain("name: 'AdminUserChatHistory'")
    expect(source).toContain(
      "component: () => import('@/views/admin/AdminUserChatHistoryView.vue')"
    )

    const routeStart = source.indexOf("name: 'AdminUserChatHistory'")
    const routeEnd = source.indexOf("path: '/admin/groups'", routeStart)
    const routeSource = source.slice(routeStart, routeEnd)

    expect(routeSource).toContain('requiresAuth: true')
    expect(routeSource).toContain('requiresAdmin: true')
  })
})
