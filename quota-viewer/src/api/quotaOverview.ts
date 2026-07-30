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
  weekly_window: {
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
  if (
    subscription.weekly_window.state === 'unknown' ||
    subscription.weekly_window.used_percent == null
  ) {
    return 'unknown'
  }
  if (subscription.weekly_window.state === 'exhausted') return 'exhausted'
  return subscription.weekly_window.used_percent >= 90 ? 'warning' : 'available'
}

const mapMembershipStatus = (status: string): MembershipStatus =>
  ['active', 'expired', 'suspended', 'revoked'].includes(status)
    ? (status as MembershipStatus)
    : 'unknown'

const membershipStatusDetail = (
  status: MembershipStatus,
  expiresAt: string
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

const mapUsagePoints = (subscription: QuotaSubscriptionDTO): DailyUsagePoint[] => {
  if (subscription.period_usage?.state !== 'available') return []
  return (subscription.period_usage.points ?? [])
    .filter(
      (point): point is typeof point & { state: 'complete' | 'partial' } =>
        point.state === 'complete' || point.state === 'partial'
    )
    .map((point) => ({
      date: point.start_at,
      label: formatDateTime(point.start_at).split(' ')[0],
      state: point.state,
      amount: 0,
      totalTokens: point.total_tokens ?? 0,
      cacheHitTokens: point.cache_hit_tokens ?? 0,
      cacheMissTokens: point.cache_miss_tokens ?? 0,
      outputTokens: point.output_tokens ?? 0,
      requests: point.requests ?? 0
    }))
}

const mapSubscription = (subscription: QuotaSubscriptionDTO): QuotaItem => {
  const window = subscription.weekly_window
  const membershipStatus = mapMembershipStatus(subscription.status)
  const isActive = membershipStatus === 'active'
  const usageAvailable = subscription.period_usage?.state === 'available'
  const statusDetailLabel = membershipStatusDetail(
    membershipStatus,
    subscription.expires_at
  )
  return {
    id: subscription.id,
    name: subscription.name,
    kindLabel: '订阅会员',
    periodLabel: '本周期',
    membershipStatus,
    statusDetailLabel,
    state: mapState(subscription),
    usedPercent:
      window.used_percent == null
        ? null
        : Math.max(0, Math.min(100, window.used_percent)),
    usedLabel: window.used == null ? '已用 —' : `已用 ${formatSnow(window.used)}`,
    limitLabel: window.limit == null ? '额度 —' : `额度 ${formatSnow(window.limit)}`,
    remainingLabel:
      window.remaining == null ? '剩余 —' : `剩余 ${formatSnow(window.remaining)}`,
    resetLabel:
      isActive
        ? window.resets_at == null
          ? '重置时间待确认'
          : `${formatDateTime(window.resets_at)} 重置`
        : statusDetailLabel,
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
    points: mapUsagePoints(subscription)
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
    quotas: dto.subscriptions.map(mapSubscription)
  }
}
