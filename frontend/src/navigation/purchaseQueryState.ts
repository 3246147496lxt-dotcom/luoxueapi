import type { LocationQuery } from 'vue-router'

/** Supported intents for the subscription catalogue. */
export const MEMBERSHIP_MODES = ['subscribe', 'renew', 'upgrade'] as const
export type MembershipMode = (typeof MEMBERSHIP_MODES)[number]

/** Query-visible catalogue tiers. Keep these stable for deep links. */
export const PRICING_TIERS = ['low', 'mid', 'high'] as const
export type PricingTier = (typeof PRICING_TIERS)[number]

/**
 * Read only an unambiguous scalar query value. Repeated parameters are treated
 * as invalid so route guards and purchase views cannot interpret them
 * differently.
 */
export function readSingleQueryString(
  query: LocationQuery,
  key: string,
): string {
  const value = query[key]
  return typeof value === 'string' ? value : ''
}

/**
 * Read the optional subscription intent from a route query.
 *
 * A repeated `mode` is intentionally treated as absent by
 * `readSingleQueryString`; unknown values are also ignored so a copied or
 * stale link cannot silently select a different purchase flow.
 */
export function readMembershipMode(
  query: LocationQuery,
): MembershipMode | null {
  const value = readSingleQueryString(query, 'mode').trim()
  return (MEMBERSHIP_MODES as readonly string[]).includes(value)
    ? value as MembershipMode
    : null
}

// Keep the naming explicit for callers that refer to the subscription
// catalogue rather than the broader membership surface.
export const readSubscriptionMode = readMembershipMode

/** Read a positive integer query value without accepting repeated/ambiguous values. */
export function readPositiveQueryInteger(
  query: LocationQuery,
  key: string,
): number | null {
  const value = readSingleQueryString(query, key)
  if (!/^\d+$/.test(value)) return null

  const parsed = Number(value)
  return Number.isSafeInteger(parsed) && parsed > 0 ? parsed : null
}

/** Read a known pricing tier from a deep-link query. */
export function readPricingTier(
  query: LocationQuery,
): PricingTier | null {
  const value = readSingleQueryString(query, 'tier')
  return (PRICING_TIERS as readonly string[]).includes(value)
    ? value as PricingTier
    : null
}

/**
 * A syntactically complete WeChat payment return always carries the explicit
 * resume marker plus at least one unambiguous scalar credential. Auxiliary
 * callback fields alone must never turn a fresh subscription checkout into a
 * recovery flow.
 */
export function hasCompleteWechatResumeQuery(
  query: LocationQuery,
): boolean {
  if (readSingleQueryString(query, 'wechat_resume') !== '1') {
    return false
  }

  return readSingleQueryString(query, 'wechat_resume_token').trim() !== ''
    || readSingleQueryString(query, 'openid').trim() !== ''
}
