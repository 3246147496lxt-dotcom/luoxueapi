import { describe, expect, it } from 'vitest'
import type { PublicModelCatalogItem, PublicModelCatalogPricing, PublicModelCatalogPricingInterval } from '@/api/catalog'
import { catalogAmount, catalogComparisonRows } from '../modelCatalogTable'

function pricing(overrides: Partial<PublicModelCatalogPricing> = {}): PublicModelCatalogPricing {
  return {
    label: '公开标准价', billing_mode: 'token', currency: 'USD', unit: 'per_token',
    input_price: 1, output_price: 2,
    cache_write_price: 3, cache_write_1h_price: 4, cache_read_price: 5,
    priority_input_price: null, priority_output_price: null,
    priority_cache_write_price: null, priority_cache_read_price: null,
    image_input_price: null, image_output_price: null, per_request_price: null,
    intervals: [], peak_rate: { enabled: false, start: '', end: '', multiplier: 1 },
    ...overrides,
  }
}
function interval(overrides: Partial<PublicModelCatalogPricingInterval> = {}): PublicModelCatalogPricingInterval {
  return {
    min_tokens: 128_000, max_tokens: null,
    input_price: null, output_price: null,
    cache_write_price: null, cache_write_1h_price: null, cache_read_price: null, per_request_price: null,
    ...overrides,
  }
}
function item(paid: PublicModelCatalogPricing, official: PublicModelCatalogPricing | null): PublicModelCatalogItem {
  return {
    slug: 'test-model', model: 'test-model', display_name: 'Test model', summary: '',
    provider: 'openai', logo_key: 'openai', category: 'chat', tags: [], capabilities: [],
    context_window: null, max_output_tokens: null, featured: false,
    public_group: { id: 101, name: '测试分组', platform: 'openai', rate_multiplier: 0.1 },
    rate_multiplier: 0.1, pricing: paid, official_pricing: official,
  }
}

describe('catalogComparisonRows', () => {
  it('splits both quotes at their combined thresholds rather than pairing intervals by position', () => {
    const rows = catalogComparisonRows(item(
      pricing({ input_price: 1, intervals: [interval({ min_tokens: 128_000, input_price: 2 })] }),
      pricing({ input_price: 10, intervals: [interval({ min_tokens: 272_000, input_price: 30 })] }),
    ))
    expect(rows.map(row => row.context)).toEqual(['≤128K', '128K–272K', '>272K'])
    expect(rows.map(row => [row.paid?.input_price, row.official?.input_price])).toEqual([[1, 10], [2, 10], [2, 30]])
  })

  it('includes finite upper bounds and falls back to each base quote outside its own interval', () => {
    const rows = catalogComparisonRows(item(
      pricing({ input_price: 1, intervals: [interval({ min_tokens: 128_000, max_tokens: 256_000, input_price: 2 })] }),
      pricing({ input_price: 10, intervals: [interval({ min_tokens: 200_000, input_price: 20 })] }),
    ))
    expect(rows.map(row => row.context)).toEqual(['≤128K', '128K–200K', '200K–256K', '>256K'])
    expect(rows.map(row => [row.paid?.input_price, row.official?.input_price])).toEqual([[1, 10], [2, 10], [2, 20], [1, 20]])
  })

  it('preserves null interval fields rather than inheriting available base prices', () => {
    const rows = catalogComparisonRows(item(
      pricing({ intervals: [interval({ output_price: 0 })] }),
      pricing({ input_price: 10, intervals: [interval({ output_price: 20 })] }),
    ))
    const longContext = rows.find(row => row.context === '>128K')!
    expect(longContext.paid).toEqual(expect.objectContaining({
      input_price: null, output_price: 0, cache_write_price: null, cache_write_1h_price: null, cache_read_price: null,
    }))
    expect(longContext.official).toEqual(expect.objectContaining({ input_price: null, output_price: 20 }))
    expect(catalogAmount(longContext.paid?.input_price, 'USD', true)).toBe('—')
    expect(catalogAmount(longContext.paid?.output_price, 'USD', true)).toBe('$0.00')
  })

  it.each([
    { paid: 'token' as const, official: 'per_request' as const },
    { paid: 'per_request' as const, official: 'image' as const },
    { paid: 'image' as const, official: 'token' as const },
  ])('does not compare a $paid quote with an official $official quote', ({ paid, official }) => {
    const rows = catalogComparisonRows(item(
      pricing({ billing_mode: paid, per_request_price: 0.02 }),
      pricing({ billing_mode: official, per_request_price: 5, priority_input_price: 100 }),
    ))
    expect(rows).toHaveLength(1)
    expect(rows.every(row => row.official === null)).toBe(true)
    expect(catalogAmount(rows[0].official?.input_price, 'USD', paid === 'token')).toBe('—')
  })

  it('keeps Priority and image extras distinct, including zero and one-sided quotes', () => {
    const rows = catalogComparisonRows(item(
      pricing({ priority_input_price: 0, priority_cache_write_price: 0.5, priority_cache_read_price: 0.1 }),
      pricing({ priority_input_price: 10, priority_output_price: 20, image_input_price: 40, image_output_price: 50 }),
    ))
    expect(rows.map(row => row.variant)).toEqual(['standard', 'priority', 'image'])
    const priority = rows.find(row => row.variant === 'priority')!
    expect(priority.paid).toEqual({
      input_price: 0, output_price: null, cache_write_price: 0.5, cache_read_price: 0.1,
      cache_write_1h_price: null, per_request_price: null,
    })
    expect(priority.official).toEqual(expect.objectContaining({ input_price: 10, output_price: 20 }))
    const image = rows.find(row => row.variant === 'image')!
    expect(image.paid).toBeNull()
    expect(image.official).toEqual(expect.objectContaining({ input_price: 40, output_price: 50, cache_write_price: null, cache_read_price: null }))
  })

  it('matches fixed-price tiers by identity and leaves unavailable counterparts empty', () => {
    const small = interval({ min_tokens: 0, tier_label: '1024x1024', per_request_price: 0.02 })
    const large = interval({ min_tokens: 0, tier_label: '2048x2048', per_request_price: 0.08 })
    const rows = catalogComparisonRows(item(
      pricing({ billing_mode: 'image', per_request_price: 0.01, intervals: [small, large] }),
      pricing({ billing_mode: 'image', per_request_price: 0.1, intervals: [{ ...large, per_request_price: 0.8 }] }),
    ))
    expect(rows.find(row => row.context === '1024x1024')?.paid?.per_request_price).toBe(0.02)
    expect(rows.find(row => row.context === '1024x1024')?.official).toBeNull()
    expect(rows.find(row => row.context === '2048x2048')?.official?.per_request_price).toBe(0.8)
    expect(rows[0].paid?.per_request_price).toBe(0.01)
    expect(rows[0].official?.per_request_price).toBe(0.1)
  })
})

describe('catalogAmount', () => {
  it('preserves zero and tiny token prices and leaves fixed prices unscaled', () => {
    expect(catalogAmount(0, 'USD', true)).toBe('$0.00')
    expect(catalogAmount(1e-14, 'USD', true)).toBe('$0.00000001')
    expect(catalogAmount(1e-18, 'USD', true)).toBe('$0.000000000001')
    expect(catalogAmount(0.00000002, 'USD', true)).toBe('$0.02')
    expect(catalogAmount(0.02, 'USD', false)).toBe('$0.02')
  })
  it.each([null, undefined, Number.NaN, Number.POSITIVE_INFINITY, Number.NEGATIVE_INFINITY])(
    'renders an unavailable %s price as a dash rather than zero',
    value => expect(catalogAmount(value, 'USD', true)).toBe('—'),
  )
})
