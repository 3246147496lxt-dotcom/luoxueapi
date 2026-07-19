import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const source = readFileSync(resolve(dirname(fileURLToPath(import.meta.url)), '../index.ts'), 'utf8')

describe('integrated recharge and redeem routes', () => {
  it('keeps the purchase page reachable when online payment is disabled', () => {
    const purchaseRoute = source.match(/\{\s*path: '\/purchase',[\s\S]*?\n\s*\},\n\s*\{\s*path: '\/orders'/)?.[0] || ''

    expect(purchaseRoute).toContain("path: '/purchase'")
    expect(purchaseRoute).not.toContain('requiresPayment: true')
  })

  it('redirects the legacy redeem deep link to the integrated panel', () => {
    expect(source).toContain("beforeEnter: () => ({ path: '/purchase', hash: '#redeem' })")
  })
})
