export type QuotaTone = 'violet' | 'sky'
export type QuotaIcon = 'crown' | 'gauge'
export type QuotaState = 'available' | 'warning' | 'exhausted' | 'expired' | 'unknown'
export type MembershipStatus =
  | 'active'
  | 'expired'
  | 'suspended'
  | 'revoked'
  | 'unknown'
export type UsagePointState = 'complete' | 'partial'
export type ViewerDataStatus = 'ready' | 'stale'
export type AccountQuotaState =
  | 'all_resources_available'
  | 'partially_restricted'
  | 'all_resources_restricted'
  | 'no_enabled_keys'
  | 'unknown'
export type WalletState = 'available' | 'exhausted' | 'unknown'

export interface DailyUsagePoint {
  date: string
  label: string
  state: UsagePointState
  amount: number
  totalTokens: number
  cacheHitTokens: number
  cacheMissTokens: number
  outputTokens: number
  requests: number
}

export interface QuotaItem {
  id: string
  name: string
  kindLabel: string
  periodLabel: string
  membershipStatus: MembershipStatus
  statusDetailLabel: string
  state: QuotaState
  usedPercent: number | null
  usedLabel: string
  limitLabel: string
  remainingLabel: string
  resetLabel: string
  periodStartLabel: string | null
  periodEndLabel: string | null
  tone: QuotaTone
  icon: QuotaIcon
  usageAvailable: boolean
  totalTokens: number | null
  totalRequests: number | null
  points: DailyUsagePoint[]
}

export interface QuotaOverview {
  accountName: string
  accountState: AccountQuotaState
  dataScopeLabel: string
  currency: string
  balance: string | null
  todaySpend: string | null
  monthSpend: string | null
  reservedBalance: string | null
  balanceKeyCount: number
  walletState: WalletState
  updatedAt: string
  generatedAt: string
  freshUntil: string
  quotas: QuotaItem[]
}
