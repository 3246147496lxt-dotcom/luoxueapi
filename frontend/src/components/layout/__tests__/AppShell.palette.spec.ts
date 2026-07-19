import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const layoutDirectory = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const layoutSource = readFileSync(resolve(layoutDirectory, 'AppLayout.vue'), 'utf8')
const headerSource = readFileSync(resolve(layoutDirectory, 'AppHeader.vue'), 'utf8')
const sidebarSource = readFileSync(resolve(layoutDirectory, 'AppSidebar.vue'), 'utf8')
const globalStyleSource = readFileSync(resolve(layoutDirectory, '../../style.css'), 'utf8')

describe('authenticated shell palette', () => {
  it('uses the captured page canvas in both color schemes', () => {
    expect(layoutSource.match(/bg-\[#f5f7fb\]/g)).toHaveLength(2)
    expect(layoutSource.match(/dark:bg-\[#0f1115\]/g)).toHaveLength(2)
    expect(layoutSource).not.toContain('#eef1ef')
    expect(layoutSource).not.toContain('#0e1211')
    expect(globalStyleSource).toContain('background: #f5f7fb;')
    expect(globalStyleSource).toContain('background: #0f1115;')
    expect(globalStyleSource).not.toContain('background: #eef1ef;')
    expect(globalStyleSource).not.toContain('background: #0e1211;')
  })

  it('matches the captured header glass surfaces and highlights', () => {
    expect(headerSource).toContain(
      'linear-gradient(rgb(248 251 255 / 0.32), rgb(235 242 252 / 0.1))',
    )
    expect(headerSource).toContain('backdrop-filter: saturate(1.7) blur(36px);')
    expect(headerSource).toContain(
      'linear-gradient(rgb(10 12 18 / 0.92), rgb(8 10 16 / 0.82))',
    )
    expect(headerSource).toContain('backdrop-filter: saturate(1.6) blur(40px);')
    expect(headerSource).toContain('rgb(96 165 250 / 0.1)')
    expect(headerSource).toContain('rgb(167 139 250 / 0.1)')
  })

  it('keeps desktop and mobile sidebars on the same captured palette', () => {
    expect(sidebarSource.match(
      /linear-gradient\(rgb\(248 251 255 \/ 0\.32\), rgb\(235 242 252 \/ 0\.1\)\)/g,
    )).toHaveLength(2)
    expect(sidebarSource).toContain('backdrop-filter: saturate(1.7) blur(36px);')
    expect(sidebarSource.match(/background: rgb\(11 15 26\) !important;/g)).toHaveLength(2)
    expect(sidebarSource).toContain('background: rgb(234 245 255);')
    expect(sidebarSource).toContain('background: rgb(71 160 255 / 0.18);')
  })
})
