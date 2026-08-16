import { describe, expect, it } from 'vitest'
import type { LocationQuery } from 'vue-router'

import {
  readMembershipMode,
  readPositiveQueryInteger,
  readPricingTier,
  readSubscriptionMode,
} from '@/navigation/purchaseQueryState'

describe('subscription purchase query state', () => {
  it.each(['subscribe', 'renew', 'upgrade'] as const)('accepts %s', (mode) => {
    expect(readMembershipMode({ mode })).toBe(mode)
    expect(readSubscriptionMode({ mode })).toBe(mode)
  })

  const invalidQueries: LocationQuery[] = [
    {},
    { mode: '' },
    { mode: 'unknown' },
    { mode: 'RENEW' },
    { mode: ['renew', 'upgrade'] },
  ]

  it.each(invalidQueries)('returns null for an absent, invalid, or repeated mode: %j', (query) => {
    expect(readMembershipMode(query)).toBeNull()
  })

  it('accepts only unambiguous positive integer group values', () => {
    expect(readPositiveQueryInteger({ group: '17' }, 'group')).toBe(17)
    expect(readPositiveQueryInteger({ group: '0' }, 'group')).toBeNull()
    expect(readPositiveQueryInteger({ group: '-1' }, 'group')).toBeNull()
    expect(readPositiveQueryInteger({ group: ['17', '18'] }, 'group')).toBeNull()
  })

  it('keeps pricing tier query values on the canonical contract', () => {
    expect(readPricingTier({ tier: 'low' })).toBe('low')
    expect(readPricingTier({ tier: 'medium' })).toBeNull()
    expect(readPricingTier({ tier: ['high', 'low'] })).toBeNull()
  })
})
