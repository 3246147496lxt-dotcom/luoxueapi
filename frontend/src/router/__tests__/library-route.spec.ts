import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const directory = dirname(fileURLToPath(import.meta.url))
const routeSource = readFileSync(resolve(directory, '../routes/user.ts'), 'utf8')

describe('library route', () => {
  it('registers one authenticated user route inside the Chat shell', () => {
    const routeBlock = routeSource.match(
      /\n[ ]{2}\{\n[ ]{4}path: '\/library',[\s\S]*?\n[ ]{2}\},\n[ ]{2}\{\n[ ]{4}path: '\/keys'/,
    )?.[0] ?? ''

    expect(routeSource.match(/path: '\/library'/g)).toHaveLength(1)
    expect(routeBlock).toContain("name: 'Library'")
    expect(routeBlock).toContain(
      "component: () => import('@/views/user/LibraryView.vue')",
    )
    expect(routeBlock).toContain('requiresAuth: true')
    expect(routeBlock).toContain('requiresAdmin: false')
    expect(routeBlock).toContain("title: 'Library'")
    expect(routeBlock).toContain("titleKey: 'library.title'")
    expect(routeBlock).toContain("shellMode: 'chat'")
  })
})
