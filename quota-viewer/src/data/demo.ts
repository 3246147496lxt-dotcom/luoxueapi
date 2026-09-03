import type {
  DailyUsagePoint,
  QuotaItem,
  QuotaOverview,
  QuotaState
} from '@/types'

export type DemoPreviewState = QuotaState | 'no-membership'

const demoTimestampFromNow = (milliseconds: number) =>
  new Date(Date.now() + milliseconds).toISOString()
const demoResetsAt = demoTimestampFromNow((2 * 24 + 14) * 60 * 60 * 1000)
const demoExpiresAt = demoTimestampFromNow(23 * 24 * 60 * 60 * 1000)
const demoExpiredAt = demoTimestampFromNow(-4 * 24 * 60 * 60 * 1000)

// Visual-reference data only. The illustrative monthly percentage is never
// derived in production; production shows —% unless quota overview supplies an
// authoritative monthly_window. Request counts and categorized Token usage are
// likewise only present here for visual-state previews.
const proPoints: DailyUsagePoint[] = [
  { date: '2026-07-25', label: '7/25', state: 'complete', amount: 2.80, totalTokens: 346000, cacheHitTokens: 170000, cacheMissTokens: 82000, outputTokens: 94000, requests: 110 },
  { date: '2026-07-26', label: '7/26', state: 'complete', amount: 3.05, totalTokens: 382000, cacheHitTokens: 188000, cacheMissTokens: 90000, outputTokens: 104000, requests: 118 },
  { date: '2026-07-27', label: '7/27', state: 'complete', amount: 3.16, totalTokens: 397000, cacheHitTokens: 196000, cacheMissTokens: 93000, outputTokens: 108000, requests: 125 },
  { date: '2026-07-28', label: '7/28', state: 'complete', amount: 3.35, totalTokens: 421000, cacheHitTokens: 210000, cacheMissTokens: 97000, outputTokens: 114000, requests: 132 },
  { date: '2026-07-29', label: '7/29', state: 'partial', amount: 2.90, totalTokens: 360000, cacheHitTokens: 174000, cacheMissTokens: 86000, outputTokens: 100000, requests: 121 }
]

const baseQuota: QuotaItem = {
  id: 'pro-monthly',
  name: 'Pro 月度会员',
  kindLabel: '订阅会员',
  periodLabel: '本周期',
  membershipStatus: 'active',
  statusDetailLabel: '',
  expiresAt: demoExpiresAt,
  isCurrentMembership: true,
  state: 'available',
  usedPercent: 68,
  usedLabel: '已用 136.20 积分',
  limitLabel: '额度 200.00 积分',
  remainingLabel: '剩余 63.80 积分',
  resetLabel: '2 天 14 小时后重置',
  resetsAt: demoResetsAt,
  monthlyRemainingPercent: 74,
  periodStartLabel: '7/25 14:00',
  periodEndLabel: '8/1 14:00',
  tone: 'violet',
  icon: 'crown',
  usageAvailable: true,
  totalTokens: 1_906_000,
  totalRequests: 606,
  points: proPoints
}

const quotaStateOverrides: Record<QuotaState, Partial<QuotaItem>> = {
  available: {
    state: 'available',
    usedPercent: 68,
    usedLabel: '已用 136.20 积分',
    remainingLabel: '剩余 63.80 积分',
    resetLabel: '2 天 14 小时后重置'
  },
  warning: {
    state: 'warning',
    usedPercent: 92,
    usedLabel: '已用 184.00 积分',
    remainingLabel: '剩余 16.00 积分',
    resetLabel: '2 天 14 小时后重置'
  },
  exhausted: {
    state: 'exhausted',
    usedPercent: 100,
    usedLabel: '已用 200.00 积分',
    remainingLabel: '剩余 0.00 积分',
    resetLabel: '2 天 14 小时后重置'
  },
  expired: {
    membershipStatus: 'expired',
    statusDetailLabel: '会员已到期',
    expiresAt: demoExpiredAt,
    isCurrentMembership: false,
    state: 'expired',
    usedPercent: 100,
    usedLabel: '已用 136.20 积分',
    remainingLabel: '剩余 0.00 积分',
    resetLabel: '会员已到期',
    resetsAt: null,
    monthlyRemainingPercent: null,
    periodStartLabel: null,
    periodEndLabel: null,
    usageAvailable: false,
    totalTokens: null,
    totalRequests: null,
    points: []
  },
  unknown: {
    state: 'unknown',
    usedPercent: null,
    usedLabel: '已用 —',
    remainingLabel: '剩余 —',
    resetLabel: '重置时间待确认',
    resetsAt: null,
    usageAvailable: false,
    totalTokens: null,
    totalRequests: null,
    points: []
  }
}

const baseOverview: Omit<QuotaOverview, 'quotas'> = {
  accountName: '落雪用户',
  accountState: 'all_resources_available',
  dataScopeLabel: '全部 API Key',
  currency: 'CNY',
  balance: '128.64',
  todaySpend: '1.16',
  monthSpend: '10.74',
  reservedBalance: '0',
  balanceKeyCount: 2,
  walletState: 'available',
  updatedAt: '刚刚',
  generatedAt: '2026-07-29T07:30:00Z',
  freshUntil: '2026-07-29T07:35:00Z'
}

export const createDemoOverview = (
  state: DemoPreviewState = 'available'
): QuotaOverview => ({
  ...baseOverview,
  quotas:
    state === 'no-membership'
      ? []
      : [
          {
            ...baseQuota,
            ...quotaStateOverrides[state]
          }
        ]
})

export const demoOverview = createDemoOverview()
