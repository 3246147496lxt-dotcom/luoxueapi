import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../TablePageLayout.vue')
const componentSource = readFileSync(componentPath, 'utf8')

describe('TablePageLayout responsive table scrolling', () => {
  it('exposes a table page kind and a dedicated header slot', () => {
    expect(componentSource).toContain('data-admin-page-kind="table"')
    expect(componentSource).toContain('<slot name="header" />')
  })

  it('does not disable the table horizontal scroll container in mobile mode', () => {
    const tableWrapperBlocks = Array.from(
      componentSource.matchAll(/([^{}]*:deep\(\.table-wrapper\)[^{}]*)\{([^{}]*)\}/g)
    )

    expect(tableWrapperBlocks.length).toBeGreaterThan(0)

    const baseBlock = tableWrapperBlocks.find(([selector]) => !selector.includes('.mobile-mode'))
    const mobileBlocks = tableWrapperBlocks.filter(([selector]) => selector.includes('.mobile-mode'))

    expect(baseBlock?.[2]).toContain('overflow-x-auto')
    expect(mobileBlocks.every(([, , declarations]) => !declarations.includes('overflow-visible'))).toBe(
      true
    )
  })

  it('removes the redundant outer card around mobile row cards', () => {
    expect(componentSource).toContain(
      '.table-page-layout.mobile-mode .table-scroll-container'
    )
    expect(componentSource).toContain('overflow-visible rounded-none border-0 bg-transparent')
    expect(componentSource).toContain('box-shadow: none')
  })
})
