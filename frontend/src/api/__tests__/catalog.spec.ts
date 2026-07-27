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
  })

  it('defaults missing effective-price currency to CREDIT without rewriting explicit USD', () => {
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

    expect(missingCurrency.items[0].pricing.currency).toBe('CREDIT')
    expect(explicitUSD.items[0].pricing.currency).toBe('USD')
  })
})
