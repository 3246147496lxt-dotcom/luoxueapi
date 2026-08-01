import type {
  AccountQuotaState,
  DailyUsagePoint,
  MembershipStatus,
  QuotaItem,
  QuotaOverview,
  QuotaState,
  WalletState
} from '@/types'

type DecimalString = string

interface QuotaWindowDTO {
  kind: string
  state: 'active' | 'exhausted' | 'unknown'
  anchor_at: string
  period_start: string | null
  period_end: string | null
  resets_at: string | null
  limit: DecimalString | null
  used: DecimalString | null
  remaining: DecimalString | null
  used_percent: number | null
}

export interface QuotaOverviewDTO {
  schema_version: number
  generated_at: string
  as_of: string
  fresh_until: string
  display_timezone: string
  freshness: 'fresh' | 'stale' | 'unknown'
  account: {
    display_label: string
    data_scope: string
    quota_state: AccountQuotaState
  }
  wallet: {
    unit: string
    state: WalletState
    available: DecimalString
    reserved: DecimalString
    balance_billed_key_count: number
    today_spend?: DecimalString | null
    month_spend?: DecimalString | null
  }
  subscriptions: QuotaSubscriptionDTO[]
}

interface QuotaSubscriptionDTO {
  id: string
  group_id: string
  name: string
  status: string
  starts_at: string
  expires_at: string
  weekly_window: QuotaWindowDTO
  monthly_window?: QuotaWindowDTO | null
  period_usage?: {
    state: 'available' | 'unknown'
    observed_until?: string
    bucket_kind?: string
    total_requests?: number | null
    total_tokens?: number | null
    points?: Array<{
      index: number
      start_at: string
      end_at: string
      state: 'complete' | 'partial' | 'future'
      requests: number | null
      cache_hit_tokens: number | null
      cache_miss_tokens: number | null
      output_tokens: number | null
      total_tokens: number | null
    }>
  }
}

const formatDateTime = (value: string | null | undefined) => {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '—'
  return new Intl.DateTimeFormat('zh-CN', {
    month: 'numeric',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false
  }).format(date)
}

const formatUpdatedAt = (value: string) => {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '未知时间'
  return new Intl.DateTimeFormat('zh-CN', {
    hour: '2-digit',
    minute: '2-digit',
    hour12: false
  }).format(date)
}

const validTimestampOrNull = (value: string | null | undefined) => {
  if (!value) return null
  return Number.isFinite(new Date(value).getTime()) ? value : null
}

const formatSnow = (value: DecimalString | null) =>
  value == null ? '—' : `❄${decimalToFixed(value, 2)}`

const decimalToFixed = (value: string, digits: number) => {
  const match = value.trim().match(/^(\d+)(?:\.(\d+))?$/)
  if (!match) return '—'
  const integer = match[1]
  const fraction = match[2] ?? ''
  const padded = `${fraction}${'0'.repeat(digits + 1)}`.slice(0, digits + 1)
  const scale = 10n ** BigInt(digits)
  const base = BigInt(integer) * scale + BigInt(padded.slice(0, digits) || '0')
  const rounded = base + (Number(padded[digits] ?? '0') >= 5 ? 1n : 0n)
  const resultInteger = rounded / scale
  const resultFraction = String(rounded % scale).padStart(digits, '0')
  return digits > 0 ? `${resultInteger}.${resultFraction}` : String(resultInteger)
}

const mapState = (subscription: QuotaSubscriptionDTO): QuotaState => {
  if (subscription.status !== 'active') {
    return ['expired', 'suspended', 'revoked'].includes(subscription.status)
      ? 'expired'
      : 'unknown'
  }

  const monthlyWindow = subscription.monthly_window
  const windows = monthlyWindow
    ? [subscription.weekly_window, monthlyWindow]
    : null
  if (
    !windows ||
    windows.some(
      (window) =>
        window.state === 'unknown' ||
        typeof window.used_percent !== 'number' ||
        !Number.isFinite(window.used_percent)
    )
  ) {
    return 'unknown'
  }
  if (windows.some((window) => window.state === 'exhausted')) return 'exhausted'
  return windows.some((window) => (window.used_percent ?? 0) >= 90)
    ? 'warning'
    : 'available'
}

const mapMembershipStatus = (status: string): MembershipStatus =>
  ['active', 'expired', 'suspended', 'revoked'].includes(status)
    ? (status as MembershipStatus)
    : 'unknown'

const membershipStatusDetail = (
  status: MembershipStatus,
  expiresAt: string | null
) => {
  switch (status) {
    case 'expired': {
      const formatted = formatDateTime(expiresAt)
      return formatted === '—' ? '会员已到期' : `已于 ${formatted} 到期`
    }
    case 'suspended':
      return '会员已暂停，请联系支持'
    case 'revoked':
      return '会员资格已撤销'
    case 'unknown':
      return '会员状态待确认'
    default:
      return ''
  }
}

const isUsageCount = (value: unknown): value is number =>
  typeof value === 'number' &&
  Number.isSafeInteger(value) &&
  value >= 0

const periodUsageIsAuthoritative = (subscription: QuotaSubscriptionDTO) => {
  const usage = subscription.period_usage
  if (
    usage?.state !== 'available' ||
    !isUsageCount(usage.total_requests) ||
    !isUsageCount(usage.total_tokens) ||
    !Array.isArray(usage.points)
  ) {
    return false
  }

  let requestTotal = 0
  let tokenTotal = 0
  for (const point of usage.points) {
    if (point.state === 'future') {
      if (
        point.requests != null ||
        point.cache_hit_tokens != null ||
        point.cache_miss_tokens != null ||
        point.output_tokens != null ||
        point.total_tokens != null
      ) {
        return false
      }
      continue
    }
    if (
      (point.state !== 'complete' && point.state !== 'partial') ||
      !isUsageCount(point.requests) ||
      !isUsageCount(point.cache_hit_tokens) ||
      !isUsageCount(point.cache_miss_tokens) ||
      !isUsageCount(point.output_tokens) ||
      !isUsageCount(point.total_tokens) ||
      point.total_tokens !==
        point.cache_hit_tokens + point.cache_miss_tokens + point.output_tokens
    ) {
      return false
    }
    requestTotal += point.requests
    tokenTotal += point.total_tokens
  }
  return (
    requestTotal === usage.total_requests &&
    tokenTotal === usage.total_tokens
  )
}

const mapUsagePoints = (
  subscription: QuotaSubscriptionDTO,
  usageAvailable: boolean
): DailyUsagePoint[] => {
  if (!usageAvailable) return []
  return (subscription.period_usage?.points ?? [])
    .filter(
      (point): point is typeof point & { state: 'complete' | 'partial' } =>
        point.state === 'complete' || point.state === 'partial'
    )
    .map((point) => ({
      date: point.start_at,
      label: formatDateTime(point.start_at).split(' ')[0],
      state: point.state,
      amount: 0,
      totalTokens: point.total_tokens as number,
      cacheHitTokens: point.cache_hit_tokens as number,
      cacheMissTokens: point.cache_miss_tokens as number,
      outputTokens: point.output_tokens as number,
      requests: point.requests as number
    }))
}

const timestamp = (value: string) => {
  const parsed = new Date(value).getTime()
  return Number.isFinite(parsed) ? parsed : Number.NEGATIVE_INFINITY
}

const isCurrentSubscription = (
  subscription: QuotaSubscriptionDTO,
  asOf: number
) => {
  if (subscription.status !== 'active' || !Number.isFinite(asOf)) return false
  const startsAt = timestamp(subscription.starts_at)
  const expiresAt = timestamp(subscription.expires_at)
  return (
    Number.isFinite(startsAt) &&
    Number.isFinite(expiresAt) &&
    startsAt <= asOf &&
    expiresAt > asOf
  )
}

const compareSubscriptionIdsDescending = (left: string, right: string) => {
  if (/^\d+$/.test(left) && /^\d+$/.test(right)) {
    const leftId = BigInt(left)
    const rightId = BigInt(right)
    if (leftId !== rightId) return leftId > rightId ? -1 : 1
  }
  return right.localeCompare(left)
}

const newestCurrentSubscriptionFirst = (
  subscriptions: QuotaSubscriptionDTO[],
  asOfValue: string
) => {
  const asOf = timestamp(asOfValue)
  return [...subscriptions].sort((left, right) => {
    const currentDifference =
      Number(isCurrentSubscription(right, asOf)) -
      Number(isCurrentSubscription(left, asOf))
    if (currentDifference !== 0) return currentDifference

    const startDifference =
      timestamp(right.starts_at) - timestamp(left.starts_at)
    if (startDifference !== 0) return startDifference

    const expiryDifference =
      timestamp(right.expires_at) - timestamp(left.expires_at)
    if (expiryDifference !== 0) return expiryDifference

    return compareSubscriptionIdsDescending(left.id, right.id)
  })
}

const mapSubscription = (
  subscription: QuotaSubscriptionDTO,
  asOf: number
): QuotaItem => {
  const window = subscription.weekly_window
  const monthlyWindow = subscription.monthly_window
  const monthlyUsedPercent = monthlyWindow?.used_percent
  const membershipStatus = mapMembershipStatus(subscription.status)
  const isActive = membershipStatus === 'active'
  const expiresAt = validTimestampOrNull(subscription.expires_at)
  const state = mapState(subscription)
  const resetsAt =
    isActive && state !== 'unknown'
      ? validTimestampOrNull(window.resets_at)
      : null
  const usageAvailable = periodUsageIsAuthoritative(subscription)
  const statusDetailLabel = membershipStatusDetail(
    membershipStatus,
    expiresAt
  )
  return {
    id: subscription.id,
    name: subscription.name,
    kindLabel: '订阅会员',
    periodLabel: '本周期',
    membershipStatus,
    statusDetailLabel,
    expiresAt,
    isCurrentMembership: isCurrentSubscription(subscription, asOf),
    state,
    usedPercent:
      window.state === 'unknown' ||
      typeof window.used_percent !== 'number' ||
      !Number.isFinite(window.used_percent)
        ? null
        : Math.max(0, Math.min(100, window.used_percent)),
    usedLabel: window.used == null ? '已用 —' : `已用 ${formatSnow(window.used)}`,
    limitLabel: window.limit == null ? '额度 —' : `额度 ${formatSnow(window.limit)}`,
    remainingLabel:
      window.remaining == null ? '剩余 —' : `剩余 ${formatSnow(window.remaining)}`,
    resetLabel:
      isActive
        ? resetsAt == null
          ? '重置时间待确认'
          : `${formatDateTime(resetsAt)} 重置`
        : statusDetailLabel,
    resetsAt,
    monthlyRemainingPercent:
      monthlyWindow?.state !== 'unknown' &&
      typeof monthlyUsedPercent === 'number' &&
      Number.isFinite(monthlyUsedPercent)
        ? Math.max(0, Math.min(100, 100 - monthlyUsedPercent))
        : null,
    periodStartLabel:
      isActive && window.period_start ? formatDateTime(window.period_start) : null,
    periodEndLabel:
      isActive && window.period_end ? formatDateTime(window.period_end) : null,
    tone: 'violet',
    icon: 'crown',
    usageAvailable,
    totalTokens: usageAvailable
      ? (subscription.period_usage?.total_tokens ?? null)
      : null,
    totalRequests: usageAvailable
      ? (subscription.period_usage?.total_requests ?? null)
      : null,
    points: mapUsagePoints(subscription, usageAvailable)
  }
}

export const adaptQuotaOverview = (input: unknown): QuotaOverview => {
  const dto = input as QuotaOverviewDTO
  if (
    !dto ||
    dto.schema_version !== 1 ||
    !dto.account ||
    !dto.wallet ||
    !Array.isArray(dto.subscriptions)
  ) {
    throw new Error('额度快照格式不受支持')
  }
  const asOf = timestamp(dto.as_of)
  return {
    accountName: dto.account.display_label || '落雪账户',
    accountState: dto.account.quota_state ?? 'unknown',
    dataScopeLabel:
      dto.account.data_scope === 'all_enabled_api_keys' ? '全部 API Key' : '账户额度',
    currency: dto.wallet.unit,
    balance: typeof dto.wallet.available === 'string' ? dto.wallet.available : null,
    todaySpend:
      typeof dto.wallet.today_spend === 'string' ? dto.wallet.today_spend : null,
    monthSpend:
      typeof dto.wallet.month_spend === 'string' ? dto.wallet.month_spend : null,
    reservedBalance:
      typeof dto.wallet.reserved === 'string' ? dto.wallet.reserved : null,
    balanceKeyCount: Number.isFinite(dto.wallet.balance_billed_key_count)
      ? dto.wallet.balance_billed_key_count
      : 0,
    walletState: dto.wallet.state ?? 'unknown',
    updatedAt: formatUpdatedAt(dto.generated_at),
    generatedAt: dto.generated_at,
    freshUntil: dto.fresh_until,
    quotas: newestCurrentSubscriptionFirst(dto.subscriptions, dto.as_of).map(
      (subscription) => mapSubscription(subscription, asOf)
    )
  }
}
