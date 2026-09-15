import type { PublicModelCatalogItem, PublicModelCatalogPricing, PublicModelCatalogPricingInterval } from '@/api/catalog'

type Prices = Pick<PublicModelCatalogPricingInterval, 'input_price' | 'output_price' | 'cache_write_price' | 'cache_write_1h_price' | 'cache_read_price' | 'per_request_price'>
export interface CatalogComparisonRow {
  key: string
  context: string
  variant: 'standard' | 'priority' | 'image'
  paid: Prices | null
  official: Prices | null
}

export function catalogPlatform(item: PublicModelCatalogItem): string {
  return (item.public_group?.platform || item.provider || 'other').trim().toLowerCase()
}

export function catalogGroupKey(item: PublicModelCatalogItem): string {
  return item.public_group ? String(item.public_group.id) : `ungrouped-${catalogPlatform(item)}`
}

export function catalogRate(item: PublicModelCatalogItem): number | null {
  return item.rate_multiplier ?? item.public_group?.rate_multiplier ?? null
}

export function catalogRateLabel(rate: number | null): string {
  return rate == null ? '—' : `${Number(rate.toPrecision(10))}x`
}

export function catalogCurrency(currency: string): string {
  return ['USD', 'CREDIT'].includes(currency.toUpperCase()) ? '$' : currency.toUpperCase() === 'CNY' ? '¥' : currency.toUpperCase()
}

export function catalogAmount(value: number | null | undefined, currency: string, perToken: boolean): string {
  if (value == null || !Number.isFinite(value)) return '—'
  const scaled = Number((value * (perToken ? 1_000_000 : 1)).toPrecision(10))
  const amount = new Intl.NumberFormat('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 20 }).format(scaled)
  const symbol = catalogCurrency(currency)
  return `${symbol}${symbol.length > 1 ? ' ' : ''}${amount}`
}

function tokenCount(value: number): string {
  if (value >= 1_000_000 && value % 1_000_000 === 0) return `${value / 1_000_000}M`
  if (value >= 1_000 && value % 1_000 === 0) return `${value / 1_000}K`
  return String(value)
}

function rangeLabel(min: number, max: number | null): string {
  if (max == null) return min > 0 ? `>${tokenCount(min)}` : ''
  return min > 0 ? `${tokenCount(min)}–${tokenCount(max)}` : `≤${tokenCount(max)}`
}

function validIntervals(pricing: PublicModelCatalogPricing | null) {
  return (pricing?.intervals || []).filter(i => i.min_tokens >= 0 && (i.max_tokens == null || i.max_tokens > i.min_tokens))
}

function intervalAt(pricing: PublicModelCatalogPricing | null, min: number, max: number | null): Prices | null {
  if (!pricing) return null
  // Intervals replace the entire base quote in the billing resolver. Null fields
  // must remain unavailable, rather than inheriting an unrelated base price.
  return validIntervals(pricing).find(i => i.min_tokens <= min && (i.max_tokens == null || (max != null && i.max_tokens >= max))) || pricing
}

function variantPrices(pricing: PublicModelCatalogPricing | null, variant: 'priority' | 'image'): Prices | null {
  if (!pricing) return null
  const input = variant === 'priority' ? pricing.priority_input_price : pricing.image_input_price
  const output = variant === 'priority' ? pricing.priority_output_price : pricing.image_output_price
  const write = variant === 'priority' ? pricing.priority_cache_write_price : null
  const read = variant === 'priority' ? pricing.priority_cache_read_price : null
  if ([input, output, write, read].every(v => v == null)) return null
  return { input_price: input, output_price: output, cache_write_price: write, cache_read_price: read, cache_write_1h_price: null, per_request_price: null }
}

export function catalogComparisonRows(item: PublicModelCatalogItem): CatalogComparisonRow[] {
  const paid = item.pricing
  // Never compare different billing units as if they were the same price.
  const official = item.official_pricing?.billing_mode === paid.billing_mode ? item.official_pricing : null
  if (paid.billing_mode !== 'token') {
    const rows: CatalogComparisonRow[] = [{ key: 'base', context: '', variant: 'standard', paid, official }]
    const tiers = new Map<string, PublicModelCatalogPricingInterval>()
    for (const interval of [...validIntervals(paid), ...validIntervals(official)]) {
      tiers.set(`${interval.tier_label || ''}:${interval.min_tokens}:${interval.max_tokens}`, interval)
    }
    for (const [key, interval] of tiers) {
      const match = (p: PublicModelCatalogPricing | null) => validIntervals(p).find(i => i.tier_label === interval.tier_label && i.min_tokens === interval.min_tokens && i.max_tokens === interval.max_tokens) || null
      rows.push({ key, context: interval.tier_label || rangeLabel(interval.min_tokens, interval.max_tokens), variant: 'standard', paid: match(paid), official: match(official) })
    }
    return rows
  }

  // Split at both sets of boundaries; pairing tiers by array index would show
  // incorrect comparisons when the channel and official thresholds differ.
  const intervals = [...validIntervals(paid), ...validIntervals(official)]
  const boundaries = Array.from(new Set([0, ...intervals.flatMap(i => [i.min_tokens, ...(i.max_tokens == null ? [] : [i.max_tokens])])])).sort((a, b) => a - b)
  const rows: CatalogComparisonRow[] = boundaries.map((min, index) => {
    const max = boundaries[index + 1] ?? null
    return { key: `tokens-${min}`, context: intervals.length ? rangeLabel(min, max) : '', variant: 'standard', paid: intervalAt(paid, min, max), official: intervalAt(official, min, max) }
  })
  for (const variant of ['priority', 'image'] as const) {
    const paidVariant = variantPrices(paid, variant)
    const officialVariant = variantPrices(official, variant)
    if (paidVariant || officialVariant) rows.push({ key: variant, context: '', variant, paid: paidVariant, official: officialVariant })
  }
  return rows
}
