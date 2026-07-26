import {
  computed,
  onBeforeUnmount,
  onMounted,
  readonly,
  shallowReactive,
  shallowRef,
  toValue,
  type ComputedRef,
  type MaybeRefOrGetter,
  type Ref
} from 'vue'
import { adminAPI } from '@/api/admin'
import type {
  Account,
  AccountUsageInfo,
  GrokQuotaWindow,
  UsageProgress
} from '@/types'
import { enqueueUsageRequest } from '@/utils/usageLoadQueue'

const GROK_FREE_TOKEN_LIMIT = 2_000_000

export type AccountUsageRequestSource = 'passive' | 'active'

export interface AccountUsageRequestOptions {
  source?: AccountUsageRequestSource
  bypassCache?: boolean
  force?: boolean
}

export interface AccountUsageHealthWindow {
  readonly label: string
  readonly utilization: number
  readonly resetsAt: string | null
}

export interface AccountUsageHealth {
  readonly isForbidden: boolean
  readonly forbiddenType: string | null
  readonly needsVerify: boolean
  readonly isBanned: boolean
  readonly needsReauth: boolean
  readonly errorCode: string | null
  readonly exhausted: boolean
  readonly maxUtilization: number | null
  readonly mostConstrainedWindow: Readonly<AccountUsageHealthWindow> | null
}

export interface AccountUsageHealthSnapshot {
  readonly accountId: number
  readonly usage: Readonly<AccountUsageInfo>
  readonly health: Readonly<AccountUsageHealth>
  readonly cachedAt: number
}

export interface AccountUsageRequest {
  readonly generation: number
  readonly authoritative: boolean
  readonly promise: Promise<AccountUsageInfo>
  isCurrent: () => boolean
}

interface InternalUsageRequest extends AccountUsageRequest {
  key: string
}

interface UsageRequestState {
  nextGeneration: number
  latestAuthoritativeGeneration: number
  publishedGeneration: number
  inFlight: Map<string, InternalUsageRequest>
}

interface PublishUsageOptions {
  generation?: number
  authoritative?: boolean
}

const usageSnapshots = shallowReactive(
  new Map<number, Readonly<AccountUsageHealthSnapshot>>()
)
const usageRequestStates = new Map<number, UsageRequestState>()
const accountStatusNow = shallowRef(Date.now())
let accountStatusClockSubscribers = 0
let accountStatusClock: ReturnType<typeof setInterval> | null = null

export const useAccountStatusClock = (): Readonly<Ref<number>> => {
  let subscribed = false

  onMounted(() => {
    subscribed = true
    accountStatusClockSubscribers += 1
    if (accountStatusClock) return

    accountStatusNow.value = Date.now()
    accountStatusClock = setInterval(() => {
      accountStatusNow.value = Date.now()
    }, 1000)
  })

  onBeforeUnmount(() => {
    if (!subscribed) return
    subscribed = false
    accountStatusClockSubscribers = Math.max(0, accountStatusClockSubscribers - 1)
    if (accountStatusClockSubscribers > 0 || !accountStatusClock) return

    clearInterval(accountStatusClock)
    accountStatusClock = null
  })

  return readonly(accountStatusNow)
}

const finiteUtilization = (value: number | null | undefined): number | null => {
  if (value == null || !Number.isFinite(value)) return null
  return Math.max(0, value)
}

const resetAtFromQuota = (quota: GrokQuotaWindow): string | null => {
  if (quota.reset_at) return quota.reset_at
  if (quota.reset_unix == null || !Number.isFinite(quota.reset_unix)) return null

  const epochMillis = quota.reset_unix < 1_000_000_000_000
    ? quota.reset_unix * 1000
    : quota.reset_unix
  const reset = new Date(epochMillis)
  return Number.isFinite(reset.getTime()) ? reset.toISOString() : null
}

const grokUsedPercent = (quota: GrokQuotaWindow | null | undefined): number | null => {
  if (!quota || quota.limit == null || quota.remaining == null || quota.limit <= 0) {
    return null
  }
  const remaining = Math.min(quota.limit, Math.max(0, quota.remaining))
  return ((quota.limit - remaining) / quota.limit) * 100
}

const firstFinite = (
  ...values: Array<number | null | undefined>
): number | null => {
  for (const value of values) {
    const normalized = finiteUtilization(value)
    if (normalized != null) return normalized
  }
  return null
}

const progressWindows: ReadonlyArray<{
  key: keyof AccountUsageInfo
  label: string
}> = [
  { key: 'five_hour', label: '5h' },
  { key: 'seven_day', label: '7d' },
  { key: 'seven_day_sonnet', label: '7d S' },
  { key: 'seven_day_fable', label: '7d F' },
  { key: 'gemini_shared_daily', label: 'Gemini 1d' },
  { key: 'gemini_pro_daily', label: 'Gemini Pro 1d' },
  { key: 'gemini_flash_daily', label: 'Gemini Flash 1d' },
  { key: 'gemini_shared_minute', label: 'Gemini 1m' },
  { key: 'gemini_pro_minute', label: 'Gemini Pro 1m' },
  { key: 'gemini_flash_minute', label: 'Gemini Flash 1m' }
]

const isUsageProgress = (value: unknown): value is UsageProgress => {
  if (!value || typeof value !== 'object') return false
  return typeof (value as UsageProgress).utilization === 'number'
}

const grokPlanLabelIsFree = (value: string) =>
  value.includes('free') || value.includes('basic')

const grokPlanLabelIsPaid = (value: string) =>
  value !== '' && !grokPlanLabelIsFree(value) && !value.includes('unknown')

const isGrokFreeUsage = (usage: Readonly<AccountUsageInfo>) => {
  const billing = usage.grok_billing
  if (
    billing?.usage_percent != null ||
    billing?.used_percent != null ||
    (billing?.monthly_limit_cents != null && billing.monthly_limit_cents > 0)
  ) {
    return false
  }

  const plan = (billing?.plan || '').trim().toLowerCase()
  const tier = (usage.subscription_tier || '').trim().toLowerCase()
  const entitlement = (usage.grok_entitlement_status || '').trim().toLowerCase()
  if (grokPlanLabelIsPaid(plan) || grokPlanLabelIsPaid(tier)) return false
  if (
    grokPlanLabelIsFree(plan) ||
    grokPlanLabelIsFree(tier) ||
    grokPlanLabelIsFree(entitlement)
  ) {
    return true
  }
  return billing != null
}

export const normalizeAccountUsageHealth = (
  usage: Readonly<AccountUsageInfo>
): Readonly<AccountUsageHealth> => {
  const windows: AccountUsageHealthWindow[] = []
  const addWindow = (
    label: string,
    utilization: number | null | undefined,
    resetsAt?: string | null
  ) => {
    const normalized = finiteUtilization(utilization)
    if (normalized == null) return
    windows.push(Object.freeze({
      label,
      utilization: normalized,
      resetsAt: resetsAt || null
    }))
  }

  for (const { key, label } of progressWindows) {
    const progress = usage[key]
    if (!isUsageProgress(progress)) continue
    addWindow(label, progress.utilization, progress.resets_at)
  }

  for (const [model, quota] of Object.entries(usage.antigravity_quota || {})) {
    addWindow(model, quota.utilization, quota.reset_time)
  }

  const billing = usage.grok_billing
  if (billing) {
    const usedCents = billing.used_cents ?? billing.included_used_cents
    const derivedUsedPercent =
      billing.monthly_limit_cents != null &&
      billing.monthly_limit_cents > 0 &&
      usedCents != null
        ? (usedCents / billing.monthly_limit_cents) * 100
        : null
    const billingLabel = billing.period_type?.toLowerCase() === 'weekly'
      ? 'Grok 7d'
      : billing.period_type?.toLowerCase() === 'monthly'
        ? 'Grok monthly'
        : 'Grok billing'
    addWindow(
      billingLabel,
      firstFinite(billing.usage_percent, billing.used_percent, derivedUsedPercent),
      billing.period_end || billing.billing_period_end || null
    )

    for (const product of billing.product_usage || []) {
      addWindow(
        `Grok ${product.product}`,
        product.usage_percent,
        billing.period_end || billing.billing_period_end || null
      )
    }
  }

  addWindow(
    'Grok requests',
    grokUsedPercent(usage.grok_request_quota),
    usage.grok_request_quota ? resetAtFromQuota(usage.grok_request_quota) : null
  )
  addWindow(
    'Grok tokens',
    grokUsedPercent(usage.grok_token_quota),
    usage.grok_token_quota ? resetAtFromQuota(usage.grok_token_quota) : null
  )

  if (isGrokFreeUsage(usage) && usage.grok_local_usage_24h) {
    addWindow(
      'Grok 24h',
      (Math.max(0, usage.grok_local_usage_24h.tokens || 0) / GROK_FREE_TOKEN_LIMIT) * 100
    )
  }

  const hasExhaustedCredits = Boolean(
    usage.ai_credits?.some((credit) => (
      credit.amount != null &&
      Number.isFinite(credit.amount) &&
      credit.minimum_balance != null &&
      Number.isFinite(credit.minimum_balance) &&
      credit.amount <= credit.minimum_balance
    ))
  )
  if (hasExhaustedCredits) addWindow('AI credits', 100)

  const mostConstrainedWindow = windows.reduce<AccountUsageHealthWindow | null>(
    (highest, candidate) => (
      !highest || candidate.utilization > highest.utilization ? candidate : highest
    ),
    null
  )
  const errorCode = usage.error_code?.trim() || null
  const forbiddenType = usage.forbidden_type?.trim() || null
  const isForbidden = Boolean(usage.is_forbidden || errorCode === 'forbidden')
  const needsReauth = Boolean(usage.needs_reauth || errorCode === 'unauthenticated')
  const maxUtilization = mostConstrainedWindow?.utilization ?? null

  return Object.freeze({
    isForbidden,
    forbiddenType,
    needsVerify: Boolean(
      usage.needs_verify ||
      (isForbidden && forbiddenType === 'validation')
    ),
    isBanned: Boolean(
      usage.is_banned ||
      (isForbidden && forbiddenType === 'violation')
    ),
    needsReauth,
    errorCode,
    exhausted: hasExhaustedCredits || (maxUtilization != null && maxUtilization >= 100),
    maxUtilization,
    mostConstrainedWindow
  })
}

const getUsageRequestState = (accountId: number): UsageRequestState => {
  let state = usageRequestStates.get(accountId)
  if (!state) {
    state = {
      nextGeneration: 0,
      latestAuthoritativeGeneration: 0,
      publishedGeneration: 0,
      inFlight: new Map()
    }
    usageRequestStates.set(accountId, state)
  }
  return state
}

export const getAccountUsageHealthSnapshot = (
  accountId: number
): Readonly<AccountUsageHealthSnapshot> | null => (
  usageSnapshots.get(accountId) ?? null
)

export const useAccountUsageHealthSnapshot = (
  accountId: MaybeRefOrGetter<number>
): ComputedRef<Readonly<AccountUsageHealthSnapshot> | null> => (
  computed(() => getAccountUsageHealthSnapshot(toValue(accountId)))
)

export const isAccountUsageHealthSnapshotFresh = (
  accountId: number,
  ttlMs: number,
  now = Date.now()
) => {
  const snapshot = getAccountUsageHealthSnapshot(accountId)
  return Boolean(snapshot && now - snapshot.cachedAt < ttlMs)
}

export const invalidateAccountUsageHealthSnapshot = (accountId: number) => {
  const state = getUsageRequestState(accountId)
  const generation = Math.max(
    state.nextGeneration,
    state.latestAuthoritativeGeneration,
    state.publishedGeneration
  ) + 1

  state.nextGeneration = generation
  state.latestAuthoritativeGeneration = generation
  state.publishedGeneration = generation
  state.inFlight.clear()
  return usageSnapshots.delete(accountId)
}

export const publishAccountUsage = (
  accountId: number,
  usage: AccountUsageInfo,
  options: PublishUsageOptions = {}
) => {
  const state = getUsageRequestState(accountId)
  const generation = options.generation ?? ++state.nextGeneration

  if (options.authoritative) {
    state.latestAuthoritativeGeneration = Math.max(
      state.latestAuthoritativeGeneration,
      generation
    )
  }
  if (
    generation < state.latestAuthoritativeGeneration ||
    generation < state.publishedGeneration
  ) {
    return false
  }

  state.publishedGeneration = generation
  usageSnapshots.set(accountId, Object.freeze({
    accountId,
    usage,
    health: normalizeAccountUsageHealth(usage),
    cachedAt: Date.now()
  }))
  return true
}

const isUsageGenerationCurrent = (
  accountId: number,
  generation: number
) => {
  const state = getUsageRequestState(accountId)
  return (
    generation >= state.latestAuthoritativeGeneration &&
    generation >= state.publishedGeneration
  )
}

export const requestAccountUsage = (
  account: Account,
  options: AccountUsageRequestOptions = {}
): AccountUsageRequest => {
  const accountId = account.id
  const state = getUsageRequestState(accountId)
  const authoritative = Boolean(
    options.bypassCache ||
    options.force ||
    options.source === 'active'
  )
  const requestKey = [
    options.source ?? 'default',
    options.bypassCache ? 'fresh' : 'normal',
    options.force ? 'force' : 'cached'
  ].join(':')
  const existing = state.inFlight.get(requestKey)
  if (existing) return existing

  if (!authoritative) {
    const authoritativeRequest = Array.from(state.inFlight.values())
      .filter(request => request.authoritative)
      .sort((a, b) => b.generation - a.generation)[0]
    if (authoritativeRequest) return authoritativeRequest
  }

  const generation = ++state.nextGeneration
  if (authoritative) {
    state.latestAuthoritativeGeneration = generation
  }

  const fetchUsage = () => {
    if (options.force) {
      return adminAPI.accounts.getUsage(accountId, options.source, true)
    }
    if (options.source) {
      return adminAPI.accounts.getUsage(accountId, options.source)
    }
    return adminAPI.accounts.getUsage(accountId)
  }

  const promise = Promise.resolve()
    .then(() => enqueueUsageRequest(account, fetchUsage))
    .then((usage) => {
      if (usage && typeof usage === 'object') {
        publishAccountUsage(accountId, usage, { generation })
      }
      return usage
    })
    .finally(() => {
      if (state.inFlight.get(requestKey)?.generation === generation) {
        state.inFlight.delete(requestKey)
      }
    })

  const request: InternalUsageRequest = {
    key: requestKey,
    generation,
    authoritative,
    promise,
    isCurrent: () => isUsageGenerationCurrent(accountId, generation)
  }
  state.inFlight.set(requestKey, request)
  return request
}
