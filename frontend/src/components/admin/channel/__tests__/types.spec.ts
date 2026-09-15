import { describe, expect, it } from 'vitest'
import {
  officialPerTokenToChannelMTok,
  formIntervalsToAPI,
  syncedModelsToPricingEntries,
  validateIntervals,
  type IntervalFormEntry,
} from '../types'
import type { ModelDefaultPricing } from '@/api/admin/channels'

function makeInterval(over: Partial<IntervalFormEntry>): IntervalFormEntry {
  return {
    min_tokens: 0,
    max_tokens: null,
    tier_label: '',
    input_price: null,
    output_price: null,
    cache_write_price: null,
    cache_read_price: null,
    per_request_price: null,
    sort_order: 0,
    ...over,
  }
}

function t(key: string, params?: Record<string, unknown>): string {
  return `${key}${params ? ` ${JSON.stringify(params)}` : ''}`
}

describe('validateIntervals', () => {
  describe('token mode', () => {
    it('rejects unbounded interval that is not last', () => {
      const intervals: IntervalFormEntry[] = [
        makeInterval({ min_tokens: 0, max_tokens: null, input_price: 1, output_price: 1 }),
        makeInterval({ min_tokens: 200000, max_tokens: 500000, input_price: 2, output_price: 2 }),
      ]
      expect(validateIntervals(intervals, 'token', t)).toContain('unboundedLast')
    })

    it('accepts unbounded interval at the end', () => {
      const intervals: IntervalFormEntry[] = [
        makeInterval({ min_tokens: 0, max_tokens: 200000, input_price: 1, output_price: 1 }),
        makeInterval({ min_tokens: 200000, max_tokens: null, input_price: 2, output_price: 2 }),
      ]
      expect(validateIntervals(intervals, 'token', t)).toBeNull()
    })

    it('rejects overlapping intervals', () => {
      const intervals: IntervalFormEntry[] = [
        makeInterval({ min_tokens: 0, max_tokens: 250000, input_price: 1, output_price: 1 }),
        makeInterval({ min_tokens: 200000, max_tokens: 500000, input_price: 2, output_price: 2 }),
      ]
      expect(validateIntervals(intervals, 'token', t)).toContain('overlap')
    })

    it('rejects unbounded interval in token mode', () => {
      const intervals: IntervalFormEntry[] = [
        makeInterval({ min_tokens: 0, max_tokens: null, input_price: 1, output_price: 1 }),
        makeInterval({ min_tokens: 100, max_tokens: 200, input_price: 2, output_price: 2 }),
      ]
      expect(validateIntervals(intervals, 'token', t)).toContain('unboundedLast')
    })
  })

  describe('image / per_request mode', () => {
    it('allows multiple unbounded tiers identified by label', () => {
      const intervals: IntervalFormEntry[] = [
        makeInterval({ tier_label: '1K', per_request_price: 0.04 }),
        makeInterval({ tier_label: '2K', per_request_price: 0.06 }),
        makeInterval({ tier_label: '4K', per_request_price: 0.08 }),
      ]
      expect(validateIntervals(intervals, 'image', t)).toBeNull()
      expect(validateIntervals(intervals, 'per_request', t)).toBeNull()
    })

    it('still rejects negative prices', () => {
      const intervals: IntervalFormEntry[] = [
        makeInterval({ tier_label: '1K', per_request_price: -1 }),
      ]
      expect(validateIntervals(intervals, 'image', t)).toContain('negativePrice')
    })

    it('still rejects max <= min on a single tier', () => {
      const intervals: IntervalFormEntry[] = [
        makeInterval({ tier_label: '1K', min_tokens: 100, max_tokens: 50, per_request_price: 0.04 }),
      ]
      expect(validateIntervals(intervals, 'image', t)).toContain('maxGreaterThanMin')
    })
  })
})

describe('officialPerTokenToChannelMTok', () => {
  it('applies the fixed 1x channel multiplier to official token quotes', () => {
    expect(officialPerTokenToChannelMTok(5e-6)).toBe(5)
    expect(officialPerTokenToChannelMTok(30e-6)).toBe(30)
    expect(officialPerTokenToChannelMTok(0)).toBe(0)
    expect(officialPerTokenToChannelMTok(5e-6, 80)).toBe(400)
    expect(officialPerTokenToChannelMTok(-1e-6)).toBeNull()
    expect(officialPerTokenToChannelMTok(5e-6, 0)).toBeNull()
  })

  it('preserves missing and non-finite quotes as empty values', () => {
    expect(officialPerTokenToChannelMTok(null)).toBeNull()
    expect(officialPerTokenToChannelMTok(undefined)).toBeNull()
    expect(officialPerTokenToChannelMTok(Number.NaN)).toBeNull()
    expect(officialPerTokenToChannelMTok(Number.POSITIVE_INFINITY)).toBeNull()
  })
})

describe('syncedModelsToPricingEntries', () => {
  it('preserves MiniMax-M3 long-context tiers and unknown cache writes through sync and save', () => {
    const quote: ModelDefaultPricing = {
      found: true, input_price: 0.3e-6, output_price: 1.2e-6,
      cache_write_price: null, cache_read_price: 0.06e-6,
      channel_pricing_multiplier: 2,
      intervals: [
        { ...makeInterval({ min_tokens: 0, max_tokens: 524288 }), input_price: 0.3e-6, output_price: 1.2e-6, cache_read_price: 0.06e-6, cache_write_price: null, per_request_price: null },
        { ...makeInterval({ min_tokens: 524288, tier_label: 'long_context', sort_order: 1 }), input_price: 0.6e-6, output_price: 2.4e-6, cache_read_price: 0.12e-6, cache_write_price: null, per_request_price: null },
      ],
    }
    const [entry] = syncedModelsToPricingEntries(['minimax-m3'], { 'minimax-m3': quote })

    expect(entry.cache_write_price).toBeNull()
    expect(entry.intervals).toEqual([
      makeInterval({ max_tokens: 524288, input_price: 0.6, output_price: 2.4, cache_read_price: 0.12 }),
      makeInterval({ min_tokens: 524288, tier_label: 'long_context', sort_order: 1, input_price: 1.2, output_price: 4.8, cache_read_price: 0.24 }),
    ])
    const saved = formIntervalsToAPI(entry.intervals)
    expect(saved[0]).toMatchObject({ max_tokens: 524288, input_price: 0.6e-6, cache_write_price: null })
    expect(saved[1]).toMatchObject({ min_tokens: 524288, max_tokens: null, input_price: 1.2e-6, output_price: 4.8e-6, cache_read_price: 0.24e-6, cache_write_price: null })
    expect(quote.intervals?.[1].input_price).toBe(0.6e-6)
  })

  it('groups equal base prices only when their context tiers also match', () => {
    const base: ModelDefaultPricing = { found: true, input_price: 0.3e-6, output_price: 1.2e-6 }
    const interval = {
      min_tokens: 524288, max_tokens: null, tier_label: 'long_context', sort_order: 1,
      input_price: 0.6e-6, output_price: 2.4e-6, cache_write_price: null, cache_read_price: null, per_request_price: null,
    }
    const entries = syncedModelsToPricingEntries(['flat', 'm3', 'm3-same', 'other-threshold', 'other-price'], {
      flat: base,
      m3: { ...base, intervals: [interval] },
      'm3-same': { ...base, intervals: [{ ...interval }] },
      'other-threshold': { ...base, intervals: [{ ...interval, min_tokens: 256000 }] },
      'other-price': { ...base, intervals: [{ ...interval, output_price: 3e-6 }] },
    })

    expect(entries.map(entry => entry.models)).toEqual([
      ['flat'], ['m3', 'm3-same'], ['other-threshold'], ['other-price'],
    ])
  })

  it('keeps different model quotes in separate channel rows and applies ×1', () => {
    const pricing: Record<string, ModelDefaultPricing> = {
      'model-a': { found: true, input_price: 1e-6, output_price: 4e-6 },
      'model-b': { found: true, input_price: 2e-6, output_price: 8e-6 },
    }

    const entries = syncedModelsToPricingEntries(['model-a', 'model-b'], pricing)

    expect(entries).toHaveLength(2)
    expect(entries[0].models).toEqual(['model-a'])
    expect(entries[0].input_price).toBe(1)
    expect(entries[0].output_price).toBe(4)
    expect(entries[1].models).toEqual(['model-b'])
    expect(entries[1].input_price).toBe(2)
    expect(entries[1].output_price).toBe(8)
  })

  it('groups only models with identical converted defaults', () => {
    const pricing: Record<string, ModelDefaultPricing> = {
      'model-a': { found: true, input_price: 1e-6, output_price: 4e-6 },
      'model-b': { found: true, input_price: 1e-6, output_price: 4e-6 },
    }

    const entries = syncedModelsToPricingEntries(['model-a', 'model-b'], pricing)

    expect(entries).toHaveLength(1)
    expect(entries[0].models).toEqual(['model-a', 'model-b'])
    expect(entries[0].input_price).toBe(1)
  })

  it('does not inherit a quote for models missing from the sync pricing map', () => {
    const entries = syncedModelsToPricingEntries(
      ['known', 'unknown'],
      { known: { found: true, input_price: 1e-6, output_price: 4e-6 } },
    )

    expect(entries).toHaveLength(2)
    expect(entries[0].models).toEqual(['known'])
    expect(entries[0].input_price).toBe(1)
    expect(entries[1].models).toEqual(['unknown'])
    expect(entries[1].input_price).toBeNull()
    expect(entries[1].output_price).toBeNull()
  })

  it('deduplicates model names while preserving sync order', () => {
    const entries = syncedModelsToPricingEntries(
      ['model-b', 'model-a', 'model-b'],
      {
        'model-a': { found: true, input_price: 1e-6 },
        'model-b': { found: true, input_price: 2e-6 },
      },
    )

    expect(entries.flatMap(entry => entry.models)).toEqual(['model-b', 'model-a'])
  })

  it('chunks a large equal-price group to the API row limit', () => {
    const models = Array.from({ length: 201 }, (_, index) => `model-${index}`)
    const entries = syncedModelsToPricingEntries(models, {
      'model-0': { found: true, input_price: 1e-6 },
      ...Object.fromEntries(models.slice(1).map(model => [model, { found: true, input_price: 1e-6 }])),
    })

    expect(entries.map(entry => entry.models.length)).toEqual([100, 100, 1])
    expect(entries.every(entry => entry.input_price === 1)).toBe(true)
  })
})
