import type {
  LocationQuery,
  RouteLocationNormalized,
  RouteLocationRaw,
} from 'vue-router'
import {
  hasCompleteWechatResumeQuery,
  readSingleQueryString,
} from '@/navigation/purchaseQueryState'

const PURCHASE_SELECTION_QUERY_KEYS = ['tab', 'plan', 'group'] as const

type PurchaseRouteState = Pick<
  RouteLocationNormalized,
  'path' | 'query' | 'hash'
>

function hasOwnQueryKey(query: LocationQuery, key: string): boolean {
  return Object.prototype.hasOwnProperty.call(query, key)
}

function hasPurchaseSelectionQuery(query: LocationQuery): boolean {
  return PURCHASE_SELECTION_QUERY_KEYS.some(
    (key) => hasOwnQueryKey(query, key),
  )
}

function withoutPurchaseSelection(query: LocationQuery): LocationQuery {
  const nextQuery = { ...query }
  PURCHASE_SELECTION_QUERY_KEYS.forEach((key) => {
    delete nextQuery[key]
  })
  return nextQuery
}

function hasPositivePlanId(query: LocationQuery): boolean {
  const value = readSingleQueryString(query, 'plan')
  if (!/^\d+$/.test(value)) {
    return false
  }

  const planId = Number(value)
  return Number.isSafeInteger(planId) && planId > 0
}

/**
 * A fresh plan bridge is the sole subscription checkout URL that should be
 * capability-gated like `/pricing`. Only a syntactically complete WeChat
 * resume takes precedence when the URL also carries `tab` and `plan`.
 */
export function isFreshSubscriptionPlanBridge(
  to: PurchaseRouteState,
): boolean {
  return to.path === '/purchase'
    && to.hash !== '#redeem'
    && readSingleQueryString(to.query, 'tab') === 'subscription'
    && hasPositivePlanId(to.query)
    && !hasCompleteWechatResumeQuery(to.query)
}

/**
 * Canonicalize ambiguous purchase state before PaymentView or capability
 * guards interpret it. Redemption is an ordinary purchase mode, repeated
 * tabs are invalid, and only a complete WeChat resume flow can retain a
 * subscription-shaped callback URL.
 */
export function resolveLegacySubscriptionPricingRedirect(
  to: PurchaseRouteState,
): RouteLocationRaw | undefined {
  if (to.path !== '/purchase') {
    return undefined
  }

  if (to.hash === '#redeem') {
    if (!hasPurchaseSelectionQuery(to.query)) {
      return undefined
    }

    return {
      path: '/purchase',
      query: withoutPurchaseSelection(to.query),
      hash: '#redeem',
      replace: true,
    }
  }

  if (Array.isArray(to.query.tab)) {
    return {
      path: '/purchase',
      query: withoutPurchaseSelection(to.query),
      hash: to.hash,
      replace: true,
    }
  }

  if (
    readSingleQueryString(to.query, 'tab') !== 'subscription'
    || isFreshSubscriptionPlanBridge(to)
    || hasCompleteWechatResumeQuery(to.query)
  ) {
    return undefined
  }

  const query = { ...to.query }
  delete query.tab
  delete query.plan

  return {
    path: '/pricing',
    query,
    hash: to.hash,
    replace: true,
  }
}
