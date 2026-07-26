import type { Account, AccountUsageInfo } from '@/types'

export type OAuthUsageHealthSnapshot = Partial<Pick<AccountUsageInfo,
  | 'updated_at'
  | 'five_hour'
  | 'seven_day'
  | 'seven_day_sonnet'
  | 'seven_day_fable'
  | 'gemini_shared_daily'
  | 'gemini_pro_daily'
  | 'gemini_flash_daily'
  | 'gemini_shared_minute'
  | 'gemini_pro_minute'
  | 'gemini_flash_minute'
  | 'antigravity_quota'
  | 'grok_request_quota'
  | 'grok_token_quota'
  | 'grok_retry_after_seconds'
  | 'grok_entitlement_status'
  | 'grok_last_status_code'
  | 'grok_billing'
  | 'is_forbidden'
  | 'forbidden_reason'
  | 'forbidden_type'
  | 'validation_url'
  | 'needs_verify'
  | 'is_banned'
  | 'needs_reauth'
  | 'error_code'
  | 'error'
>>

export type AccountEffectiveStatusCode =
  | 'permission_error'
  | 'authentication_error'
  | 'account_error'
  | 'quota_exhausted'
  | 'overloaded'
  | 'rate_limited'
  | 'temp_unschedulable'
  | 'inactive'
  | 'paused'
  | 'credits_exhausted'
  | 'model_rate_limited'
  | 'credits_active'
  | 'at_capacity'
  | 'active'

export type AccountEffectiveStatusTone = 'danger' | 'warning' | 'muted' | 'success'

export type AccountEffectiveStatusResetKind =
  | 'quota'
  | 'overload'
  | 'rate_limit'
  | 'temp_unschedulable'
  | 'local_limit'

export interface AccountEffectiveStatus {
  code: AccountEffectiveStatusCode
  tone: AccountEffectiveStatusTone
  labelKey: string
  titleKey: string
  titleParams: Readonly<Record<string, string | number>>
  /**
   * Concise, redacted reason derived from the account or usage response.
   * Consumers should not treat this as translated UI copy.
   */
  detail: string | null
  resetAt: string | null
  resetKind: AccountEffectiveStatusResetKind | null
  affectedScopes: readonly string[]
}

export interface ResolveAccountEffectiveStatusOptions {
  now?: Date | string | number
}

interface TimedScope {
  scope: string
  resetAt: string | null
  resetMs: number | null
}

interface LocalRestriction extends TimedScope {
  kind: 'credits_exhausted' | 'model_rate_limited' | 'credits_active'
}

const PERMISSION_ERROR_PATTERN =
  /(?:\b403\b|forbidden|permission(?:_denied)?|access denied|violation|validation required|suspend(?:ed)?|banned)/i
const AUTHENTICATION_ERROR_PATTERN =
  /(?:\b401\b|unauthenticated|unauthorized|authentication failed|invalid_grant|token (?:invalid|revoked|expired)|re-?auth)/i
const ACCOUNT_ERROR_DETAIL_MAX_LENGTH = 240
const SENSITIVE_DETAIL_KEY_PATTERN =
  /(?:token|secret|password|authorization|api[_-]?key)/i
const NAMED_SECRET_PATTERN =
  /(authorization|proxy-authorization|x-api-key|api[_-]?key|access[_-]?token|refresh[_-]?token|id[_-]?token|password|client[_-]?secret)\s*[:=]\s*(?:bearer\s+)?[^\s,;]+/gi
const BEARER_SECRET_PATTERN = /\bbearer\s+[^\s,;]+/gi
const OPENAI_KEY_PATTERN = /\bsk-[A-Za-z0-9_-]{6,}/g
const JWT_PATTERN = /\beyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\b/g

const asRecord = (value: unknown): Record<string, unknown> | null =>
  value !== null && typeof value === 'object' && !Array.isArray(value)
    ? value as Record<string, unknown>
    : null

const asFiniteNumber = (value: unknown): number | null =>
  typeof value === 'number' && Number.isFinite(value) ? value : null

const asNonEmptyString = (value: unknown): string | null =>
  typeof value === 'string' && value.trim() ? value.trim() : null

const parsedErrorDetail = (raw: string): { message: string; code: string } | null => {
  const jsonStart = raw.indexOf('{')
  if (jsonStart < 0) return null

  try {
    const payload = asRecord(JSON.parse(raw.slice(jsonStart)))
    if (!payload) return null
    const error = asRecord(payload.error)
    const detail = asRecord(payload.detail)
    const message = asNonEmptyString(error?.message) ??
      asNonEmptyString(detail?.message) ??
      asNonEmptyString(payload.message) ??
      asNonEmptyString(payload.detail) ??
      asNonEmptyString(payload.error)
    if (!message) return null
    return {
      message,
      code: asNonEmptyString(error?.code) ??
        asNonEmptyString(detail?.code) ??
        asNonEmptyString(payload.code) ??
        ''
    }
  } catch {
    return null
  }
}

const summarizeErrorDetail = (account: Account, raw: string | null): string | null => {
  if (!raw) return null

  const parsed = parsedErrorDetail(raw)
  let summary = raw
  if (parsed) {
    const prefix = raw.slice(0, raw.indexOf('{')).trim().replace(/:\s*$/, '')
    summary = prefix ? `${prefix}: ${parsed.message}` : parsed.message
    if (parsed.code && !summary.toLowerCase().includes(parsed.code.toLowerCase())) {
      summary += ` (${parsed.code})`
    }
  }

  summary = summary
    .replace(NAMED_SECRET_PATTERN, '$1=[REDACTED]')
    .replace(BEARER_SECRET_PATTERN, 'Bearer [REDACTED]')
    .replace(OPENAI_KEY_PATTERN, '[REDACTED]')
    .replace(JWT_PATTERN, '[REDACTED]')

  for (const [key, value] of Object.entries(account.credentials ?? {})) {
    if (!SENSITIVE_DETAIL_KEY_PATTERN.test(key) || typeof value !== 'string' || value.length < 4) {
      continue
    }
    summary = summary.split(value).join('[REDACTED]')
  }

  summary = summary.replace(/\s+/g, ' ').trim()
  if (summary.length <= ACCOUNT_ERROR_DETAIL_MAX_LENGTH) return summary
  return `${summary.slice(0, ACCOUNT_ERROR_DETAIL_MAX_LENGTH - 3).trimEnd()}...`
}

const toTimestamp = (value: Date | string | number | null | undefined): number | null => {
  if (value instanceof Date) {
    const timestamp = value.getTime()
    return Number.isFinite(timestamp) ? timestamp : null
  }
  if (typeof value === 'number') {
    return Number.isFinite(value) ? value : null
  }
  const normalized = asNonEmptyString(value)
  if (!normalized) return null
  const timestamp = new Date(normalized).getTime()
  return Number.isFinite(timestamp) ? timestamp : null
}

const resolveNow = (value: ResolveAccountEffectiveStatusOptions['now']): number =>
  toTimestamp(value) ?? Date.now()

const futureReset = (
  value: string | null | undefined,
  nowMs: number
): TimedScope['resetAt'] => {
  const normalized = asNonEmptyString(value)
  if (!normalized) return null
  const timestamp = toTimestamp(normalized)
  return timestamp !== null && timestamp > nowMs ? normalized : null
}

const derivedResetAt = (
  explicit: unknown,
  seconds: unknown,
  baseAt: unknown,
  nowMs: number
): string | null => {
  const explicitReset = asNonEmptyString(explicit)
  if (explicitReset && toTimestamp(explicitReset) !== null) return explicitReset

  const remainingSeconds = asFiniteNumber(seconds)
  if (remainingSeconds === null || remainingSeconds <= 0) return null
  const baseMs = toTimestamp(asNonEmptyString(baseAt)) ?? nowMs
  return new Date(baseMs + remainingSeconds * 1000).toISOString()
}

const addTimedScope = (
  scopes: TimedScope[],
  scope: string,
  resetAt: string | null,
  nowMs: number,
  allowUnknownReset: boolean
) => {
  const resetMs = toTimestamp(resetAt)
  if (resetMs !== null && resetMs <= nowMs) return
  if (resetMs === null && !allowUnknownReset) return
  scopes.push({
    scope,
    resetAt: resetMs === null ? null : resetAt,
    resetMs
  })
}

const latestSharedReset = (scopes: readonly TimedScope[]): string | null => {
  if (scopes.length === 0 || scopes.some(scope => scope.resetMs === null)) return null
  return scopes.reduce((latest, current) => (
    (current.resetMs ?? 0) > (latest.resetMs ?? 0) ? current : latest
  )).resetAt
}

const uniqueScopes = (scopes: readonly TimedScope[]): string[] =>
  Array.from(new Set(scopes.map(scope => scope.scope)))

const isExhausted = (used: unknown, limit: unknown): boolean => {
  const usedValue = asFiniteNumber(used)
  const limitValue = asFiniteNumber(limit)
  return usedValue !== null && limitValue !== null && limitValue > 0 && usedValue >= limitValue
}

const statusResult = (
  code: AccountEffectiveStatusCode,
  tone: AccountEffectiveStatusTone,
  labelKey: string,
  overrides: Partial<Omit<AccountEffectiveStatus, 'code' | 'tone' | 'labelKey'>> = {}
): AccountEffectiveStatus => ({
  code,
  tone,
  labelKey,
  titleKey: overrides.titleKey ?? labelKey,
  titleParams: overrides.titleParams ?? {},
  detail: overrides.detail ?? null,
  resetAt: overrides.resetAt ?? null,
  resetKind: overrides.resetKind ?? null,
  affectedScopes: overrides.affectedScopes ?? []
})

const forbiddenTextKey = (forbiddenType: string | null, detail: string | null): string => {
  const normalizedType = forbiddenType?.toLowerCase()
  const normalizedDetail = detail?.toLowerCase() ?? ''
  if (normalizedType === 'validation' || normalizedDetail.includes('validation')) {
    return 'admin.accounts.forbiddenValidation'
  }
  if (
    normalizedType === 'violation' ||
    normalizedDetail.includes('violation') ||
    normalizedDetail.includes('banned')
  ) {
    return 'admin.accounts.forbiddenViolation'
  }
  return 'admin.accounts.forbidden'
}

const resolveTopPriorityError = (
  account: Account,
  usage: OAuthUsageHealthSnapshot | null
): AccountEffectiveStatus | null => {
  const accountError = asNonEmptyString(account.error_message)
  const usageError = asNonEmptyString(usage?.error)
  const forbiddenReason = asNonEmptyString(usage?.forbidden_reason)
  const errorCode = asNonEmptyString(usage?.error_code)?.toLowerCase() ?? ''
  const entitlement = asNonEmptyString(usage?.grok_entitlement_status)?.toLowerCase() ?? ''
  const accountPermissionError = accountError ? PERMISSION_ERROR_PATTERN.test(accountError) : false
  const accountAuthenticationError = accountError
    ? AUTHENTICATION_ERROR_PATTERN.test(accountError)
    : false

  const isPermissionError = Boolean(
    usage?.is_forbidden ||
    usage?.needs_verify ||
    usage?.is_banned ||
    errorCode === 'forbidden' ||
    errorCode === 'permission_denied' ||
    usage?.grok_last_status_code === 403 ||
    ['forbidden', 'denied', 'suspended', 'banned'].includes(entitlement) ||
    accountPermissionError
  )
  if (isPermissionError) {
    const rawDetail = forbiddenReason ?? accountError ?? usageError
    const labelKey = forbiddenTextKey(asNonEmptyString(usage?.forbidden_type), rawDetail)
    return statusResult('permission_error', 'danger', labelKey, {
      detail: summarizeErrorDetail(account, rawDetail)
    })
  }

  const isAuthenticationError = Boolean(
    usage?.needs_reauth ||
    errorCode === 'unauthenticated' ||
    errorCode === 'unauthorized' ||
    usage?.grok_last_status_code === 401 ||
    accountAuthenticationError
  )
  if (isAuthenticationError) {
    return statusResult(
      'authentication_error',
      'danger',
      'admin.accounts.needsReauth',
      { detail: summarizeErrorDetail(account, accountError ?? usageError) }
    )
  }

  if (account.status === 'error' || accountError) {
    return statusResult(
      'account_error',
      'danger',
      'admin.accounts.status.error',
      { detail: summarizeErrorDetail(account, accountError) }
    )
  }
  return null
}

const collectQuotaExhaustion = (
  account: Account,
  usage: OAuthUsageHealthSnapshot | null,
  nowMs: number
): TimedScope[] => {
  const exhausted: TimedScope[] = []
  const addExhausted = (
    scope: string,
    matches: boolean,
    resetAt: string | null = null
  ) => {
    if (!matches) return
    addTimedScope(exhausted, scope, resetAt, nowMs, true)
  }

  addExhausted('total', isExhausted(account.quota_used, account.quota_limit))
  addExhausted(
    'daily',
    isExhausted(account.quota_daily_used, account.quota_daily_limit),
    asNonEmptyString(account.quota_daily_reset_at)
  )
  addExhausted(
    'weekly',
    isExhausted(account.quota_weekly_used, account.quota_weekly_limit),
    asNonEmptyString(account.quota_weekly_reset_at)
  )

  const extra = account.extra ?? {}
  const codexUpdatedAt = asNonEmptyString(extra.codex_usage_updated_at) ?? account.updated_at
  const addCodexWindow = (scope: string, prefix: 'codex_5h' | 'codex_7d') => {
    const utilization = asFiniteNumber(extra[`${prefix}_used_percent`])
    if (utilization === null || utilization < 100) return
    const resetAt = derivedResetAt(
      extra[`${prefix}_reset_at`],
      extra[`${prefix}_reset_after_seconds`],
      codexUpdatedAt,
      nowMs
    )
    addExhausted(scope, true, resetAt)
  }
  addCodexWindow('five_hour', 'codex_5h')
  addCodexWindow('seven_day', 'codex_7d')

  const addUsageWindow = (
    scope: string,
    window: OAuthUsageHealthSnapshot['five_hour']
  ) => {
    if (!window || !Number.isFinite(window.utilization) || window.utilization < 100) return
    const resetAt = derivedResetAt(
      window.resets_at,
      window.remaining_seconds,
      usage?.updated_at,
      nowMs
    )
    addExhausted(scope, true, resetAt)
  }

  addUsageWindow('five_hour', usage?.five_hour)
  addUsageWindow('seven_day', usage?.seven_day)
  addUsageWindow('gemini_shared_daily', usage?.gemini_shared_daily)
  addUsageWindow('gemini_shared_minute', usage?.gemini_shared_minute)

  const addGrokWindow = (
    scope: string,
    window: OAuthUsageHealthSnapshot['grok_request_quota']
  ) => {
    const limit = asFiniteNumber(window?.limit)
    if (!window || limit === null || limit <= 0) return
    const remaining = asFiniteNumber(window.remaining)
    if (remaining === null || remaining > 0) return
    const resetAt = asNonEmptyString(window.reset_at) ??
      (asFiniteNumber(window.reset_unix) !== null
        ? new Date((asFiniteNumber(window.reset_unix) as number) * 1000).toISOString()
        : null)
    addExhausted(scope, true, resetAt)
  }
  addGrokWindow('grok_requests', usage?.grok_request_quota)
  addGrokWindow('grok_tokens', usage?.grok_token_quota)

  const billing = usage?.grok_billing
  if (billing) {
    const utilization = Math.max(
      asFiniteNumber(billing.usage_percent) ?? 0,
      asFiniteNumber(billing.used_percent) ?? 0,
      isExhausted(billing.used_cents, billing.monthly_limit_cents) ? 100 : 0
    )
    addExhausted(
      'grok_billing',
      utilization >= 100,
      asNonEmptyString(billing.period_end) ?? asNonEmptyString(billing.billing_period_end)
    )
  }

  return exhausted
}

const collectLocalRestrictions = (
  account: Account,
  usage: OAuthUsageHealthSnapshot | null,
  nowMs: number
): LocalRestriction[] => {
  const restrictions: LocalRestriction[] = []
  const extra = account.extra ?? {}
  const modelLimits = asRecord(extra.model_rate_limits)
  const activeModelEntries = modelLimits
    ? Object.entries(modelLimits).flatMap(([model, rawInfo]) => {
        const info = asRecord(rawInfo)
        const resetAt = futureReset(asNonEmptyString(info?.rate_limit_reset_at), nowMs)
        return resetAt ? [{ model, resetAt }] : []
      })
    : []
  const creditsExhausted = activeModelEntries.some(entry => entry.model === 'AICredits')
  const allowOverages = extra.allow_overages === true

  for (const entry of activeModelEntries) {
    const kind = entry.model === 'AICredits'
      ? 'credits_exhausted'
      : allowOverages && !creditsExhausted
        ? 'credits_active'
        : 'model_rate_limited'
    restrictions.push({
      kind,
      scope: entry.model,
      resetAt: entry.resetAt,
      resetMs: toTimestamp(entry.resetAt)
    })
  }

  const addLocalUsageWindow = (
    scope: string,
    window: OAuthUsageHealthSnapshot['five_hour']
  ) => {
    if (!window || !Number.isFinite(window.utilization) || window.utilization < 100) return
    const resetAt = derivedResetAt(
      window.resets_at,
      window.remaining_seconds,
      usage?.updated_at,
      nowMs
    )
    const timed: TimedScope[] = []
    addTimedScope(timed, scope, resetAt, nowMs, true)
    if (timed[0]) restrictions.push({ ...timed[0], kind: 'model_rate_limited' })
  }

  addLocalUsageWindow('sonnet', usage?.seven_day_sonnet)
  addLocalUsageWindow('fable', usage?.seven_day_fable)
  addLocalUsageWindow('gemini_pro_daily', usage?.gemini_pro_daily)
  addLocalUsageWindow('gemini_flash_daily', usage?.gemini_flash_daily)
  addLocalUsageWindow('gemini_pro_minute', usage?.gemini_pro_minute)
  addLocalUsageWindow('gemini_flash_minute', usage?.gemini_flash_minute)

  for (const [model, quota] of Object.entries(usage?.antigravity_quota ?? {})) {
    if (!quota || !Number.isFinite(quota.utilization) || quota.utilization < 100) continue
    const timed: TimedScope[] = []
    addTimedScope(timed, model, asNonEmptyString(quota.reset_time), nowMs, true)
    if (timed[0]) restrictions.push({ ...timed[0], kind: 'model_rate_limited' })
  }

  return restrictions
}

const localRestrictionResult = (
  restrictions: readonly LocalRestriction[]
): AccountEffectiveStatus | null => {
  if (restrictions.length === 0) return null

  const selectedKind: LocalRestriction['kind'] = restrictions.some(
    restriction => restriction.kind === 'credits_exhausted'
  )
    ? 'credits_exhausted'
    : restrictions.some(restriction => restriction.kind === 'model_rate_limited')
      ? 'model_rate_limited'
      : 'credits_active'
  const selected = restrictions.filter(restriction => restriction.kind === selectedKind)
  const resetAt = latestSharedReset(selected)
  const affectedScopes = uniqueScopes(selected)
  const model = affectedScopes[0] ?? ''

  if (selectedKind === 'credits_exhausted') {
    return statusResult(
      'credits_exhausted',
      'danger',
      'admin.accounts.status.creditsExhausted',
      {
        titleKey: resetAt
          ? 'admin.accounts.status.creditsExhaustedUntil'
          : 'admin.accounts.status.creditsExhausted',
        titleParams: resetAt ? { time: resetAt } : {},
        resetAt,
        resetKind: 'local_limit',
        affectedScopes
      }
    )
  }

  if (selectedKind === 'model_rate_limited') {
    return statusResult(
      'model_rate_limited',
      'warning',
      'admin.accounts.status.limited',
      {
        titleKey: resetAt
          ? 'admin.accounts.status.modelRateLimitedUntil'
          : 'admin.accounts.status.limited',
        titleParams: resetAt ? { model, time: resetAt } : { model },
        resetAt,
        resetKind: 'local_limit',
        affectedScopes
      }
    )
  }

  return statusResult(
    'credits_active',
    'warning',
    'admin.accounts.status.limited',
    {
      titleKey: resetAt
        ? 'admin.accounts.status.modelCreditOveragesUntil'
        : 'admin.accounts.status.limited',
      titleParams: resetAt ? { model, time: resetAt } : { model },
      resetAt,
      resetKind: 'local_limit',
      affectedScopes
    }
  )
}

/**
 * Resolves the single effective status shown by compact account surfaces.
 *
 * Priority:
 * auth/permission error > quota exhausted > overload > global rate limit >
 * temporary unschedulable > inactive > paused > local model/credits limit >
 * concurrency full > active.
 */
export const resolveAccountEffectiveStatus = (
  account: Account,
  usage: OAuthUsageHealthSnapshot | null = null,
  options: ResolveAccountEffectiveStatusOptions = {}
): AccountEffectiveStatus => {
  const nowMs = resolveNow(options.now)

  const topPriorityError = resolveTopPriorityError(account, usage)
  if (topPriorityError) return topPriorityError

  const exhaustedQuota = collectQuotaExhaustion(account, usage, nowMs)
  if (exhaustedQuota.length > 0) {
    const resetAt = latestSharedReset(exhaustedQuota)
    return statusResult(
      'quota_exhausted',
      'danger',
      'admin.accounts.status.quotaExceeded',
      {
        resetAt,
        resetKind: 'quota',
        affectedScopes: uniqueScopes(exhaustedQuota)
      }
    )
  }

  const overloadResetAt = futureReset(account.overload_until, nowMs)
  if (overloadResetAt) {
    return statusResult(
      'overloaded',
      'danger',
      'admin.accounts.status.overloaded',
      {
        titleKey: 'admin.accounts.status.overloadedUntil',
        titleParams: { time: overloadResetAt },
        resetAt: overloadResetAt,
        resetKind: 'overload'
      }
    )
  }

  const accountRateLimitResetAt = futureReset(account.rate_limit_reset_at, nowMs)
  const retryAfterSeconds = asFiniteNumber(usage?.grok_retry_after_seconds)
  const usageRateLimited = usage?.error_code === 'rate_limited' ||
    (retryAfterSeconds !== null && retryAfterSeconds > 0) ||
    usage?.grok_last_status_code === 429
  if (accountRateLimitResetAt || usageRateLimited) {
    const resetAt = accountRateLimitResetAt ??
      (retryAfterSeconds !== null && retryAfterSeconds > 0
        ? new Date(nowMs + retryAfterSeconds * 1000).toISOString()
        : null)
    return statusResult(
      'rate_limited',
      'warning',
      'admin.accounts.status.rateLimited',
      {
        titleKey: resetAt
          ? 'admin.accounts.status.rateLimitedUntil'
          : 'admin.accounts.status.rateLimited',
        titleParams: resetAt ? { time: resetAt } : {},
        resetAt,
        resetKind: 'rate_limit'
      }
    )
  }

  const tempUnschedulableResetAt = futureReset(account.temp_unschedulable_until, nowMs)
  if (tempUnschedulableResetAt) {
    return statusResult(
      'temp_unschedulable',
      'warning',
      'admin.accounts.status.tempUnschedulable',
      {
        detail: asNonEmptyString(account.temp_unschedulable_reason),
        resetAt: tempUnschedulableResetAt,
        resetKind: 'temp_unschedulable'
      }
    )
  }

  if (account.status === 'inactive') {
    return statusResult(
      'inactive',
      'muted',
      'admin.accounts.status.inactive'
    )
  }

  if (!account.schedulable) {
    return statusResult(
      'paused',
      'muted',
      'admin.accounts.status.paused'
    )
  }

  const localRestriction = localRestrictionResult(
    collectLocalRestrictions(account, usage, nowMs)
  )
  if (localRestriction) return localRestriction

  const concurrencyLimit = Number(account.concurrency || 0)
  const currentConcurrency = Number(account.current_concurrency || 0)
  if (
    Number.isFinite(concurrencyLimit) &&
    Number.isFinite(currentConcurrency) &&
    concurrencyLimit > 0 &&
    currentConcurrency >= concurrencyLimit
  ) {
    return statusResult(
      'at_capacity',
      'warning',
      'admin.accounts.workbench.fullStatus'
    )
  }

  return statusResult(
    'active',
    'success',
    'admin.accounts.workbench.activeStatus'
  )
}
