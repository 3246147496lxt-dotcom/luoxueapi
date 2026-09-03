import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const directory = dirname(fileURLToPath(import.meta.url))
const routeSource = readFileSync(resolve(directory, '../routes/user.ts'), 'utf8')
const guardSource = readFileSync(resolve(directory, '../guards.ts'), 'utf8')

describe('integrated recharge and redeem routes', () => {
  it('keeps the purchase page reachable when online payment is disabled', () => {
    const purchaseRoute = routeSource.match(
      /\n[ ]{2}\{\n[ ]{4}path: '\/purchase',[\s\S]*?\n[ ]{2}\},\n[ ]{2}\{\n[ ]{4}path: '\/pricing'/,
    )?.[0] || ''

    expect(purchaseRoute).toContain("path: '/purchase'")
    expect(purchaseRoute).not.toContain('requiresPayment: true')
    expect(purchaseRoute).not.toContain('beforeEnter')
    expect(guardSource).toContain('resolveLegacySubscriptionPricingRedirect(to)')
    expect(guardSource).toContain('isFreshSubscriptionPlanBridge(to)')
  })

  it('keeps the independent pricing page behind payment and simple-mode capability checks', () => {
    const pricingRoute = routeSource.match(
      /\n[ ]{2}\{\n[ ]{4}path: '\/pricing',[\s\S]*?\n[ ]{2}\},\n[ ]{2}\{\n[ ]{4}path: '\/orders'/,
    )?.[0] || ''

    expect(pricingRoute).toContain("name: 'UserSubscriptionPricing'")
    expect(pricingRoute).toContain('requiresPayment: true')
    expect(guardSource).toContain("'/pricing'")
  })

  it('redirects the legacy redeem deep link to the integrated panel', () => {
    expect(routeSource).toContain("beforeEnter: () => ({ path: '/purchase', hash: '#redeem' })")
  })
})
