import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const layoutDirectory = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const layoutSource = readFileSync(resolve(layoutDirectory, 'AppLayout.vue'), 'utf8')
const headerSource = readFileSync(resolve(layoutDirectory, 'AppHeader.vue'), 'utf8')
const sidebarSource = readFileSync(resolve(layoutDirectory, 'AppSidebar.vue'), 'utf8')

describe('authenticated mixed application shell palette', () => {
  it('uses the canonical canvas for the shared application shell', () => {
    expect(layoutSource).toContain('app-layout app-layout--snow-shell')
    expect(layoutSource).toContain('background: var(--lx-clay-canvas) !important;')
    expect(layoutSource).toContain('font-family: var(--lx-clay-font-ui);')
    expect(layoutSource).not.toContain('#f5f7fb')
    expect(layoutSource).not.toContain('#0f1115')
  })

  it('keeps the header on the original production glass and typography contract', () => {
    expect(headerSource).toContain('class="app-header fixed inset-x-0 top-0')
    expect(headerSource).toContain('font-family: system-ui, -apple-system')
    expect(headerSource).toContain('border-width: 1px 1px 0;')
    expect(headerSource).toContain(
      'background: linear-gradient(rgb(248 251 255 / 0.32), rgb(235 242 252 / 0.1));',
    )
    expect(headerSource).toContain('backdrop-filter: saturate(1.7) blur(36px);')
    expect(headerSource).toContain(
      'linear-gradient(rgb(10 12 18 / 0.92), rgb(8 10 16 / 0.82))',
    )
    expect(headerSource).not.toContain('background: var(--lx-clay-surface-elevated);')
  })

  it('keeps the sidebar on the original production glass and blue interaction contract', () => {
    expect(sidebarSource).toContain('class="sidebar"')
    expect(sidebarSource).toContain('w-44 min-[1025px]:w-[188px] min-[1281px]:w-[200px]')
    expect(sidebarSource).toContain('background: linear-gradient(rgb(248 251 255 / 0.32), rgb(235 242 252 / 0.1)) !important;')
    expect(sidebarSource).toContain(':global(.dark .sidebar)')
    expect(sidebarSource).toContain('background: rgb(234 245 255);')
    expect(sidebarSource).not.toContain('sidebar--snow-clay')
  })
})
