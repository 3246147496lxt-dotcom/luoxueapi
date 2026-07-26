import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const source = readFileSync(resolve(dirname(fileURLToPath(import.meta.url)), '../index.ts'), 'utf8')

describe('integrated recharge and redeem routes', () => {
  it('keeps the purchase page reachable when online payment is disabled', () => {
    const purchaseRoute = source.match(
      /\n[ ]{2}\{\n[ ]{4}path: '\/purchase',[\s\S]*?\n[ ]{2}\},\n[ ]{2}\{\n[ ]{4}path: '\/pricing'/,
    )?.[0] || ''

    expect(purchaseRoute).toContain("path: '/purchase'")
    expect(purchaseRoute).not.toContain('requiresPayment: true')
    expect(purchaseRoute).not.toContain('beforeEnter')
    expect(source).toContain('resolveLegacySubscriptionPricingRedirect(to)')
    expect(source).toContain('isFreshSubscriptionPlanBridge(to)')
  })

  it('keeps the independent pricing page behind payment and simple-mode capability checks', () => {
    const pricingRoute = source.match(
      /\n[ ]{2}\{\n[ ]{4}path: '\/pricing',[\s\S]*?\n[ ]{2}\},\n[ ]{2}\{\n[ ]{4}path: '\/orders'/,
    )?.[0] || ''

    expect(pricingRoute).toContain("name: 'UserSubscriptionPricing'")
    expect(pricingRoute).toContain('requiresPayment: true')
    expect(source).toContain("'/pricing'")
  })

  it('redirects the legacy redeem deep link to the integrated panel', () => {
    expect(source).toContain("beforeEnter: () => ({ path: '/purchase', hash: '#redeem' })")
  })
})
