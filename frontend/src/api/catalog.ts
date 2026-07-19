import { apiClient } from './client'
import type { BillingMode } from '@/constants/channel'

export interface PublicModelCatalogPricingInterval {
  min_tokens: number
  max_tokens: number | null
  tier_label?: string
  input_price: number | null
  output_price: number | null
  cache_write_price: number | null
  cache_write_1h_price: number | null
  cache_read_price: number | null
  per_request_price: number | null
}

export interface PublicModelCatalogPeakRate {
  enabled: boolean
  start: string
  end: string
  multiplier: number
}

export interface PublicModelCatalogPricing {
  label: string
  billing_mode: BillingMode
  currency: 'USD' | string
  unit: 'per_token' | 'per_request' | string
  input_price: number | null
  output_price: number | null
  cache_write_price: number | null
  cache_write_1h_price: number | null
  cache_read_price: number | null
  priority_input_price: number | null
  priority_output_price: number | null
  priority_cache_write_price: number | null
  priority_cache_read_price: number | null
  image_input_price: number | null
  image_output_price: number | null
  per_request_price: number | null
  intervals: PublicModelCatalogPricingInterval[]
  peak_rate: PublicModelCatalogPeakRate
}

export interface PublicModelCatalogItem {
  slug: string
  model: string
  display_name: string
  summary: string
  provider: string
  logo_key: string
  category: string
  tags: string[]
  capabilities: string[]
  context_window: number | null
  max_output_tokens: number | null
  featured: boolean
  pricing: PublicModelCatalogPricing
}

export interface PublicModelCatalogResponse {
  items: PublicModelCatalogItem[]
  server_timezone: string
  pricing_updated_at: string
}

function nullableNumber(value: unknown): number | null {
  return typeof value === 'number' && Number.isFinite(value) ? value : null
}

function normalizePricing(
  pricing: Partial<PublicModelCatalogPricing> | null | undefined
): PublicModelCatalogPricing {
  const peakRate = pricing?.peak_rate
  const billingMode = pricing?.billing_mode
  return {
    label: typeof pricing?.label === 'string' ? pricing.label : '',
    billing_mode: billingMode === 'per_request' || billingMode === 'image' ? billingMode : 'token',
    currency: typeof pricing?.currency === 'string' ? pricing.currency : 'USD',
    unit: typeof pricing?.unit === 'string' ? pricing.unit : 'per_token',
    input_price: nullableNumber(pricing?.input_price),
    output_price: nullableNumber(pricing?.output_price),
    cache_write_price: nullableNumber(pricing?.cache_write_price),
    cache_write_1h_price: nullableNumber(pricing?.cache_write_1h_price),
    cache_read_price: nullableNumber(pricing?.cache_read_price),
    priority_input_price: nullableNumber(pricing?.priority_input_price),
    priority_output_price: nullableNumber(pricing?.priority_output_price),
    priority_cache_write_price: nullableNumber(pricing?.priority_cache_write_price),
    priority_cache_read_price: nullableNumber(pricing?.priority_cache_read_price),
    image_input_price: nullableNumber(pricing?.image_input_price),
    image_output_price: nullableNumber(pricing?.image_output_price),
    per_request_price: nullableNumber(pricing?.per_request_price),
    intervals: Array.isArray(pricing?.intervals)
      ? pricing.intervals.map((interval) => ({
        min_tokens: nullableNumber(interval?.min_tokens) ?? 0,
        max_tokens: nullableNumber(interval?.max_tokens),
        tier_label: typeof interval?.tier_label === 'string' ? interval.tier_label : undefined,
        input_price: nullableNumber(interval?.input_price),
        output_price: nullableNumber(interval?.output_price),
        cache_write_price: nullableNumber(interval?.cache_write_price),
        cache_write_1h_price: nullableNumber(interval?.cache_write_1h_price),
        cache_read_price: nullableNumber(interval?.cache_read_price),
        per_request_price: nullableNumber(interval?.per_request_price)
      }))
      : [],
    peak_rate: {
      enabled: peakRate?.enabled === true,
      start: typeof peakRate?.start === 'string' ? peakRate.start : '',
      end: typeof peakRate?.end === 'string' ? peakRate.end : '',
      multiplier: nullableNumber(peakRate?.multiplier) ?? 1
    }
  }
}

function stringArray(value: unknown): string[] {
  if (!Array.isArray(value)) return []
  return value.filter((item): item is string => typeof item === 'string' && item.trim().length > 0)
}

export function normalizePublicModelCatalogResponse(
  response: Partial<PublicModelCatalogResponse> | null | undefined
): PublicModelCatalogResponse {
  const rawItems = Array.isArray(response?.items) ? response.items : []
  const items = rawItems
    .filter((item) => item && typeof item.model === 'string' && item.model.trim().length > 0)
    .map((item) => ({
      slug: typeof item.slug === 'string' && item.slug ? item.slug : item.model,
      model: item.model,
      display_name: typeof item.display_name === 'string' && item.display_name
        ? item.display_name
        : item.model,
      summary: typeof item.summary === 'string' ? item.summary : '',
      provider: typeof item.provider === 'string' ? item.provider : '',
      logo_key: typeof item.logo_key === 'string' ? item.logo_key : '',
      category: typeof item.category === 'string' ? item.category : 'other',
      tags: stringArray(item.tags),
      capabilities: stringArray(item.capabilities),
      context_window: nullableNumber(item.context_window),
      max_output_tokens: nullableNumber(item.max_output_tokens),
      featured: item.featured === true,
      pricing: normalizePricing(item.pricing)
    }))

  return {
    items,
    server_timezone: typeof response?.server_timezone === 'string' ? response.server_timezone : '',
    pricing_updated_at: typeof response?.pricing_updated_at === 'string' ? response.pricing_updated_at : ''
  }
}

export async function getPublicModelCatalog(options?: {
  signal?: AbortSignal
}): Promise<PublicModelCatalogResponse> {
  const { data } = await apiClient.get<PublicModelCatalogResponse>('/catalog/models', {
    signal: options?.signal
  })
  return normalizePublicModelCatalogResponse(data)
}

export const publicModelCatalogAPI = {
  get: getPublicModelCatalog
}

export default publicModelCatalogAPI
