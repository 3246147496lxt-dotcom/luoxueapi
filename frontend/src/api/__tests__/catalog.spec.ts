import { describe, expect, it } from 'vitest'

import { normalizePublicModelCatalogResponse } from '@/api/catalog'

describe('normalizePublicModelCatalogResponse', () => {
  it('preserves zero prices while normalizing absent optional collections and peak data', () => {
    const response = normalizePublicModelCatalogResponse({
      items: [{
        slug: 'gpt-test',
        model: 'gpt-test',
        display_name: 'GPT Test',
        summary: '',
        provider: 'openai',
        logo_key: 'openai',
        category: 'chat',
        tags: undefined as unknown as string[],
        capabilities: undefined as unknown as string[],
        context_window: null,
        max_output_tokens: null,
        featured: false,
        pricing: {
          label: '公开标准价',
          billing_mode: 'token',
          currency: 'CREDIT',
          unit: 'per_token',
          input_price: 0,
          output_price: null,
          cache_write_price: null,
          cache_write_1h_price: null,
          cache_read_price: null,
          priority_input_price: null,
          priority_output_price: null,
          priority_cache_write_price: null,
          priority_cache_read_price: null,
          image_input_price: null,
          image_output_price: null,
          per_request_price: null,
          intervals: undefined as unknown as [],
          peak_rate: undefined as never
        }
      }],
      server_timezone: 'Asia/Shanghai',
      pricing_updated_at: ''
    })

    expect(response.items[0].pricing.input_price).toBe(0)
    expect(response.items[0].pricing.currency).toBe('CREDIT')
    expect(response.items[0].pricing.output_price).toBeNull()
    expect(response.items[0].pricing.intervals).toEqual([])
    expect(response.items[0].pricing.peak_rate).toEqual({
      enabled: false,
      start: '',
      end: '',
      multiplier: 1
    })
    expect(response.items[0].tags).toEqual([])
    expect(response.items[0].public_group).toBeNull()
    expect(response.items[0].rate_multiplier).toBeNull()
    expect(response.items[0].official_pricing).toBeNull()
  })

  it('defaults missing effective-price currency to USD without rewriting explicit USD', () => {
    const basePricing = {
      label: '公开标准价',
      billing_mode: 'token' as const,
      unit: 'per_token',
      input_price: 0.000005,
      output_price: null,
      cache_write_price: null,
      cache_write_1h_price: null,
      cache_read_price: null,
      priority_input_price: null,
      priority_output_price: null,
      priority_cache_write_price: null,
      priority_cache_read_price: null,
      image_input_price: null,
      image_output_price: null,
      per_request_price: null,
      intervals: [],
      peak_rate: { enabled: false, start: '', end: '', multiplier: 1 }
    }
    const item = {
      slug: 'gpt-test',
      model: 'gpt-test',
      display_name: 'GPT Test',
      summary: '',
      provider: 'openai',
      logo_key: 'openai',
      category: 'chat',
      tags: [],
      capabilities: [],
      context_window: null,
      max_output_tokens: null,
      featured: false
    }

    const missingCurrency = normalizePublicModelCatalogResponse({
      items: [{
        ...item,
        pricing: { ...basePricing, currency: undefined as unknown as string }
      }],
      server_timezone: '',
      pricing_updated_at: ''
    })
    const explicitUSD = normalizePublicModelCatalogResponse({
      items: [{ ...item, pricing: { ...basePricing, currency: 'USD' } }],
      server_timezone: '',
      pricing_updated_at: ''
    })

    expect(missingCurrency.items[0].pricing.currency).toBe('USD')
    expect(explicitUSD.items[0].pricing.currency).toBe('USD')
  })

  it('keeps official and paid prices separate, including zero rates and prices', () => {
    const base = normalizePublicModelCatalogResponse({ items: [{ model: 'exact-model' }] }).items[0]
    const group = {
      id: 7, name: 'Public group', platform: 'openai', rate_multiplier: 0,
      account_ids: [99], channel_name: 'internal', model_routing: { secret: [99] }
    }
    const response = normalizePublicModelCatalogResponse({
      items: [{
        ...base,
        public_group: group,
        rate_multiplier: 0,
        pricing: { ...base.pricing, input_price: 0, output_price: 0 },
        official_pricing: {
          ...base.pricing,
          currency: 'EUR',
          input_price: 0.000005,
          output_price: 0,
          cache_read_price: null,
          intervals: [{
            min_tokens: 272000, max_tokens: null, input_price: 0.00001,
            output_price: 0, cache_write_price: null, cache_write_1h_price: null,
            cache_read_price: null, per_request_price: null
          }]
        }
      }]
    })
    const item = response.items[0]
    expect(item.public_group).toEqual({ id: 7, name: 'Public group', platform: 'openai', rate_multiplier: 0 })
    expect(item.rate_multiplier).toBe(0)
    expect(item.pricing.input_price).toBe(0)
    expect(item.official_pricing?.input_price).toBe(0.000005)
    expect(item.official_pricing?.output_price).toBe(0)
    expect(item.official_pricing?.cache_read_price).toBeNull()
    expect(item.official_pricing?.currency).toBe('EUR')
    expect(item.official_pricing?.intervals[0].input_price).toBe(0.00001)
  })

  it('does not infer missing official prices or malformed group rates', () => {
    for (const rate of [-1, Number.NaN, Number.POSITIVE_INFINITY]) {
      const response = normalizePublicModelCatalogResponse({
        items: [{
          model: 'custom-model',
          public_group: { id: 7, name: 'Public', platform: 'openai', rate_multiplier: rate },
          rate_multiplier: rate,
          official_pricing: null
        }]
      })
      expect(response.items[0].public_group).toBeNull()
      expect(response.items[0].rate_multiplier).toBeNull()
      expect(response.items[0].official_pricing).toBeNull()
    }
  })

  it('preserves independent image rates and per-request units without currency conversion', () => {
    const base = normalizePublicModelCatalogResponse({ items: [{ model: 'image-model' }] }).items[0]
    const response = normalizePublicModelCatalogResponse({
      items: [{
        ...base,
        public_group: { id: 7, name: 'Images', platform: 'openai', rate_multiplier: 0.1 },
        rate_multiplier: 0.2,
        pricing: { ...base.pricing, billing_mode: 'image', unit: 'per_request', currency: 'USD', per_request_price: 0.02 },
        official_pricing: { ...base.pricing, billing_mode: 'image', unit: 'per_request', currency: 'USD', per_request_price: 0.1 }
      }]
    })
    expect(response.items[0].rate_multiplier).toBe(0.2)
    expect(response.items[0].public_group?.rate_multiplier).toBe(0.1)
    expect(response.items[0].pricing.per_request_price).toBe(0.02)
    expect(response.items[0].official_pricing?.per_request_price).toBe(0.1)
    expect(response.items[0].official_pricing?.unit).toBe('per_request')
    expect(response.items[0].official_pricing?.currency).toBe('USD')
  })
})
