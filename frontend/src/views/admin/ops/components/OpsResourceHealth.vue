<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'

import {
  opsAPI,
  type AlertEvent,
  type AlertEventsQuery,
  type OpsAccountPoolAnomaly,
  type OpsAccountPoolGroupSummary,
  type OpsAccountPoolResponse,
  type OpsProxyHealthItem,
  type OpsProxyHealthResponse,
  type OpsProxyHealthSummary,
  type OpsResourceView
} from '@/api/admin/ops'
import OpsWorkspaceNav from './OpsWorkspaceNav.vue'

interface Props {
  resource: OpsResourceView
  platformFilter?: string
  groupIdFilter?: number | null
  refreshToken: number
}

interface Emits {
  (event: 'update:resource', value: OpsResourceView): void
}

type PendingSeverity = 'critical' | 'warning' | 'notice'
type AccountLedgerScope = 'capacity' | 'manual' | 'automatic'
type ProxyLedgerScope = 'issues' | 'stale' | 'healthy'
type CapacityTone = 'success' | 'warning' | 'danger' | 'neutral'

interface PendingItem {
  key: string
  severity: PendingSeverity
  priority: number
  kindLabel: string
  title: string
  detail: string
  account?: OpsAccountPoolAnomaly
  group?: OpsAccountPoolGroupSummary
  proxy?: OpsProxyHealthItem
}

interface CapacitySegment {
  key: string
  label: string
  value: number
  tone: CapacityTone
}

type DetailSelection =
  | { kind: 'account'; account: OpsAccountPoolAnomaly }
  | { kind: 'group'; group: OpsAccountPoolGroupSummary }
  | { kind: 'proxy'; proxy: OpsProxyHealthItem }

const props = withDefaults(defineProps<Props>(), {
  platformFilter: '',
  groupIdFilter: null
})
const emit = defineEmits<Emits>()

const router = useRouter()
const { t, locale } = useI18n()

const ACCOUNT_ANOMALY_LIMIT = 20
const DETAIL_LIST_LIMIT = 20

const accountSnapshot = ref<OpsAccountPoolResponse | null>(null)
const proxySnapshot = ref<OpsProxyHealthResponse | null>(null)
const alertSnapshot = ref<AlertEvent[] | null>(null)
const accountLoading = ref(false)
const proxyLoading = ref(false)
const alertLoading = ref(false)
const accountError = ref('')
const proxyError = ref('')
const alertError = ref('')
const testingProxyIds = ref<Set<number>>(new Set())
const proxyActionErrors = ref<Record<number, string>>({})
const actionAnnouncement = ref('')
const selectedDetail = ref<DetailSelection | null>(null)
const detailDrawerOpen = ref(false)
const accountLedgerScope = ref<AccountLedgerScope>('capacity')
const proxyLedgerScope = ref<ProxyLedgerScope>('issues')
const detailDrawer = ref<HTMLElement | null>(null)
const detailLoading = ref(false)
const detailError = ref('')

let activeController: AbortController | null = null
let detailController: AbortController | null = null
let loadSequence = 0
let detailTriggerElement: HTMLElement | null = null
let previousBodyOverflow = ''

const navItems = computed(() => [
  { id: 'overview', label: t('admin.ops.resourceHealth.nav.overview') },
  { id: 'accounts', label: t('admin.ops.resourceHealth.nav.accounts') },
  { id: 'proxies', label: t('admin.ops.resourceHealth.nav.proxies') }
])

const currentPanelId = computed(() => `ops-workspace-panel-${props.resource}`)
const currentPanelLabel = computed(() => navItems.value.find((item) => item.id === props.resource)?.label || '')
const initialLoading = computed(() => {
  if (props.resource === 'accounts') return accountLoading.value && !accountSnapshot.value
  if (props.resource === 'proxies') return proxyLoading.value && !proxySnapshot.value
  const hasSnapshot = Boolean(accountSnapshot.value || proxySnapshot.value || alertSnapshot.value)
  return !hasSnapshot && (accountLoading.value || proxyLoading.value || alertLoading.value)
})

const manualRecoveryAccounts = computed(() => accountSnapshot.value?.actionable_anomalies || [])
const automaticRecoveryAccounts = computed(() => accountSnapshot.value?.auto_recovering_anomalies || [])
const accountSummary = computed(() => accountSnapshot.value?.summary)
const zeroCapacityGroups = computed(() => accountSnapshot.value?.groups.filter((group) => group.zero_capacity) || [])
const capacityRiskGroups = computed(
  () => accountSnapshot.value?.groups.filter((group) => group.zero_capacity || group.low_redundancy) || []
)
const healthyGroups = computed(
  () => accountSnapshot.value?.groups.filter((group) => !group.zero_capacity && !group.low_redundancy) || []
)
const overviewCapacityRiskGroups = computed(() =>
  [...capacityRiskGroups.value]
    .sort(
      (left, right) =>
        Number(right.zero_capacity) - Number(left.zero_capacity) ||
        left.base_schedulable_count - right.base_schedulable_count ||
        left.group_id - right.group_id
    )
    .slice(0, 5)
)

const proxies = computed(() => proxySnapshot.value?.items || [])

function isProxyStale(proxy: OpsProxyHealthItem): boolean {
  return proxy.connectivity_stale || proxy.quality_stale
}

function isProxyAnomaly(proxy: OpsProxyHealthItem): boolean {
  return proxy.lifecycle !== 'active' || proxy.health !== 'healthy'
}

const proxyAnomalies = computed(() => proxies.value.filter(isProxyAnomaly))
const staleProxies = computed(() => proxies.value.filter(isProxyStale))
const proxyIssues = computed(() => proxies.value.filter((proxy) => isProxyAnomaly(proxy) || isProxyStale(proxy)))
const healthyProxies = computed(
  () => proxies.value.filter((proxy) => proxy.lifecycle === 'active' && proxy.health === 'healthy' && !isProxyStale(proxy))
)
const healthyProxyCount = computed(() => proxySnapshot.value?.summary.healthy || 0)
const affectedProxyAccountCount = computed(() => proxySnapshot.value?.summary.affected_accounts || 0)

function isSevereAlert(alert: AlertEvent): boolean {
  return ['p0', 'p1', 'critical', 'severe'].includes(String(alert.severity || '').trim().toLowerCase())
}

const unresolvedSevereAlerts = computed(() =>
  (alertSnapshot.value || []).filter((alert) => alert.status === 'firing' && isSevereAlert(alert))
)
const accountDataAvailable = computed(() => accountSnapshot.value !== null && !accountError.value)
const proxyDataAvailable = computed(() => proxySnapshot.value !== null && !proxyError.value)
const alertDataAvailable = computed(() => alertSnapshot.value !== null && !alertError.value)

const overviewMetrics = computed(() => [
  {
    key: 'schedulableAccounts',
    label: t('admin.ops.resourceHealth.metrics.schedulableAccounts'),
    value: accountDataAvailable.value ? accountSummary.value?.base_schedulable_count || 0 : '—',
    hint: accountDataAvailable.value
      ? t('admin.ops.resourceHealth.metrics.accountsTotal', { total: accountSummary.value?.total_accounts || 0 })
      : t('admin.ops.resourceHealth.metrics.dataUnavailable'),
    tone: accountDataAvailable.value ? ((accountSummary.value?.base_schedulable_count || 0) > 0 ? 'success' : 'danger') : 'neutral'
  },
  {
    key: 'groupCapacity',
    label: t('admin.ops.resourceHealth.metrics.groupCapacity'),
    value: accountDataAvailable.value
      ? `${accountSummary.value?.zero_capacity_group_count || 0} / ${accountSummary.value?.low_redundancy_group_count || 0}`
      : '—',
    hint: accountDataAvailable.value
      ? t('admin.ops.resourceHealth.metrics.groupCapacityLegend')
      : t('admin.ops.resourceHealth.metrics.dataUnavailable'),
    tone: accountDataAvailable.value
      ? (accountSummary.value?.zero_capacity_group_count || 0) > 0
        ? 'danger'
        : (accountSummary.value?.low_redundancy_group_count || 0) > 0
          ? 'warning'
          : 'success'
      : 'neutral'
  },
  {
    key: 'proxyHealth',
    label: t('admin.ops.resourceHealth.metrics.proxyHealth'),
    value: proxyDataAvailable.value ? `${healthyProxyCount.value} / ${affectedProxyAccountCount.value}` : '—',
    hint: proxyDataAvailable.value
      ? t('admin.ops.resourceHealth.metrics.proxyHealthLegend')
      : t('admin.ops.resourceHealth.metrics.dataUnavailable'),
    tone: proxyDataAvailable.value ? (affectedProxyAccountCount.value > 0 ? 'warning' : 'success') : 'neutral'
  },
  {
    key: 'severeAlerts',
    label: t('admin.ops.resourceHealth.metrics.severeAlerts'),
    value: alertDataAvailable.value ? unresolvedSevereAlerts.value.length : '—',
    hint: alertDataAvailable.value
      ? t('admin.ops.resourceHealth.metrics.severeAlertsHint')
      : t('admin.ops.resourceHealth.metrics.dataUnavailable'),
    tone: alertDataAvailable.value ? (unresolvedSevereAlerts.value.length > 0 ? 'danger' : 'success') : 'neutral'
  }
])

const accountCompositionSegments = computed<CapacitySegment[]>(() => [
  {
    key: 'schedulable',
    label: t('admin.ops.resourceHealth.composition.schedulable'),
    value: accountSummary.value?.base_schedulable_count || 0,
    tone: 'success'
  },
  {
    key: 'automatic',
    label: t('admin.ops.resourceHealth.composition.automatic'),
    value: accountSummary.value?.auto_recovering_count || 0,
    tone: 'warning'
  },
  {
    key: 'manual',
    label: t('admin.ops.resourceHealth.composition.manual'),
    value: accountSummary.value?.actionable_count || 0,
    tone: 'danger'
  },
  {
    key: 'excluded',
    label: t('admin.ops.resourceHealth.composition.excluded'),
    value: (accountSummary.value?.inactive_count || 0) + (accountSummary.value?.manual_unschedulable_count || 0),
    tone: 'neutral'
  }
])

const accountCompositionSum = computed(() =>
  accountCompositionSegments.value.reduce((total, segment) => total + segment.value, 0)
)
const accountCompositionTotal = computed(() => accountSummary.value?.total_accounts || 0)
const accountCompositionComplete = computed(
  () =>
    accountDataAvailable.value &&
    accountCompositionTotal.value > 0 &&
    accountCompositionSum.value === accountCompositionTotal.value
)
const accountSnapshotTime = computed(() =>
  accountSnapshot.value?.collected_at
    ? formatDateTime(accountSnapshot.value.collected_at)
    : t('admin.ops.resourceHealth.unknownTime')
)

const accountMetrics = computed(() => [
  {
    key: 'total',
    label: t('admin.ops.resourceHealth.accounts.total'),
    value: accountDataAvailable.value ? accountSummary.value?.total_accounts || 0 : '—',
    tone: 'neutral'
  },
  {
    key: 'available',
    label: t('admin.ops.resourceHealth.accounts.available'),
    value: accountDataAvailable.value ? accountSummary.value?.base_schedulable_count || 0 : '—',
    tone: accountDataAvailable.value ? ((accountSummary.value?.base_schedulable_count || 0) > 0 ? 'success' : 'danger') : 'neutral'
  },
  {
    key: 'automatic',
    label: t('admin.ops.resourceHealth.accounts.automaticRecovery'),
    value: accountDataAvailable.value ? accountSummary.value?.auto_recovering_count || 0 : '—',
    tone: accountDataAvailable.value ? ((accountSummary.value?.auto_recovering_count || 0) > 0 ? 'warning' : 'success') : 'neutral'
  },
  {
    key: 'manual',
    label: t('admin.ops.resourceHealth.accounts.manualRecovery'),
    value: accountDataAvailable.value ? accountSummary.value?.actionable_count || 0 : '—',
    tone: accountDataAvailable.value ? ((accountSummary.value?.actionable_count || 0) > 0 ? 'danger' : 'success') : 'neutral'
  }
])

const proxyMetrics = computed(() => [
  {
    key: 'total',
    label: t('admin.ops.resourceHealth.proxies.total'),
    value: proxyDataAvailable.value ? proxySnapshot.value?.summary.total || 0 : '—',
    tone: 'neutral'
  },
  {
    key: 'healthy',
    label: t('admin.ops.resourceHealth.proxies.healthy'),
    value: proxyDataAvailable.value ? healthyProxyCount.value : '—',
    tone: proxyDataAvailable.value ? 'success' : 'neutral'
  },
  {
    key: 'anomalies',
    label: t('admin.ops.resourceHealth.proxies.anomalies'),
    value: proxyDataAvailable.value ? proxyAnomalies.value.length : '—',
    tone: proxyDataAvailable.value ? (proxyAnomalies.value.length > 0 ? 'danger' : 'success') : 'neutral'
  },
  {
    key: 'stale',
    label: t('admin.ops.resourceHealth.proxies.stale'),
    value: proxyDataAvailable.value ? staleProxies.value.length : '—',
    tone: proxyDataAvailable.value ? (staleProxies.value.length > 0 ? 'warning' : 'success') : 'neutral'
  }
])

const pendingItems = computed<PendingItem[]>(() => {
  const items: PendingItem[] = [
    ...zeroCapacityGroups.value.map((group) => ({
      key: `group-zero-${group.group_id}`,
      severity: 'critical' as const,
      priority: 0,
      kindLabel: t('admin.ops.resourceHealth.pending.zeroCapacityGroup'),
      title: group.group_name || `#${group.group_id}`,
      detail: t('admin.ops.resourceHealth.pending.zeroCapacityDetail', { platform: group.platform }),
      group
    })),
    ...[...proxyIssues.value]
      .sort((left, right) => right.active_account_count - left.active_account_count || right.id - left.id)
      .map((proxy) => ({
        key: `proxy-issue-${proxy.id}`,
        severity: 'critical' as const,
        priority: 1,
        kindLabel: t('admin.ops.resourceHealth.pending.proxyImpact', { count: proxy.active_account_count }),
        title: proxy.name || `#${proxy.id}`,
        detail: isProxyStale(proxy) ? proxyCheckedLabel(proxy) : proxyAnomalyReason(proxy),
        proxy
      })),
    ...manualRecoveryAccounts.value.map((account) => ({
      key: `account-manual-${account.account_id}`,
      severity: 'critical' as const,
      priority: 2,
      kindLabel: t('admin.ops.resourceHealth.pending.accountManual'),
      title: account.account_name || `#${account.account_id}`,
      detail: manualRecoveryReason(account),
      account
    })),
    ...automaticRecoveryAccounts.value.map((account) => ({
      key: `account-automatic-${account.account_id}`,
      severity: 'warning' as const,
      priority: 3,
      kindLabel: t('admin.ops.resourceHealth.pending.accountAutomatic'),
      title: account.account_name || `#${account.account_id}`,
      detail: automaticRecoveryReason(account),
      account
    }))
  ]

  return items.sort((a, b) => a.priority - b.priority).slice(0, 5)
})

const pendingTotalCount = computed(
  () =>
    (accountSummary.value?.zero_capacity_group_count ?? zeroCapacityGroups.value.length) +
    proxyIssues.value.length +
    (accountSummary.value?.actionable_count ?? manualRecoveryAccounts.value.length) +
    (accountSummary.value?.auto_recovering_count ?? automaticRecoveryAccounts.value.length)
)

const visibleManualAccounts = computed(() => manualRecoveryAccounts.value.slice(0, DETAIL_LIST_LIMIT))
const visibleAutomaticAccounts = computed(() => automaticRecoveryAccounts.value.slice(0, DETAIL_LIST_LIMIT))
const visibleProxyIssues = computed(() => proxyIssues.value.slice(0, DETAIL_LIST_LIMIT))
const visibleStaleProxies = computed(() => staleProxies.value.slice(0, DETAIL_LIST_LIMIT))
const visibleHealthyProxies = computed(() => healthyProxies.value.slice(0, DETAIL_LIST_LIMIT))

const accountLedgerCount = computed(() => {
  if (accountLedgerScope.value === 'manual') return accountSummary.value?.actionable_count || 0
  if (accountLedgerScope.value === 'automatic') return accountSummary.value?.auto_recovering_count || 0
  return capacityRiskGroups.value.length
})

const proxyLedgerCount = computed(() => {
  if (proxyLedgerScope.value === 'stale') return staleProxies.value.length
  if (proxyLedgerScope.value === 'healthy') return healthyProxies.value.length
  return proxyIssues.value.length
})

function selectResource(value: string): void {
  if (value === 'overview' || value === 'accounts' || value === 'proxies') {
    emit('update:resource', value)
  }
}

const knownAccountReasons = new Set([
  'account_error',
  'expired_auto_paused',
  'known_quota_exhausted',
  'rate_limited',
  'overloaded',
  'temporary_cooldown'
])

const knownProxyReasons = new Set([
  'not_checked',
  'connectivity_stale',
  'connectivity_failed',
  'quality_stale',
  'all_supported_targets_passed',
  'challenge_detected',
  'supported_target_warning_or_failure',
  'quality_unknown'
])

function accountReason(account: OpsAccountPoolAnomaly): string {
  if (knownAccountReasons.has(account.primary_reason)) {
    return t(`admin.ops.resourceHealth.accounts.reasons.${account.primary_reason}`)
  }
  return account.detail || t('admin.ops.resourceHealth.accounts.unavailableReason')
}

function manualRecoveryReason(account: OpsAccountPoolAnomaly): string {
  return accountReason(account)
}

function automaticRecoveryReason(account: OpsAccountPoolAnomaly): string {
  const reason = accountReason(account)
  if (!account.recover_at) return reason
  return t('admin.ops.resourceHealth.accounts.recoversAt', {
    reason,
    time: formatDateTime(account.recover_at)
  })
}

function proxyHealthReason(proxy: OpsProxyHealthItem): string {
  if (knownProxyReasons.has(proxy.health_reason)) {
    return t(`admin.ops.resourceHealth.proxies.reasons.${proxy.health_reason}`)
  }
  return proxy.health_reason || t('admin.ops.resourceHealth.proxies.unknownAnomaly')
}

function proxyAnomalyReason(proxy: OpsProxyHealthItem): string {
  if (proxy.lifecycle === 'expired') return t('admin.ops.resourceHealth.proxies.reasons.expired')
  if (proxy.lifecycle === 'inactive') return t('admin.ops.resourceHealth.proxies.reasons.inactive')
  if (proxy.lifecycle === 'expiring_soon') return t('admin.ops.resourceHealth.proxies.reasons.expiringSoon')
  return proxyHealthReason(proxy)
}

function proxyCheckedLabel(proxy: OpsProxyHealthItem): string {
  if (proxy.connectivity_stale && proxy.quality_stale) {
    return t('admin.ops.resourceHealth.proxies.bothChecksStale')
  }
  if (proxy.connectivity_stale) {
    return proxy.connectivity_checked_at
      ? t('admin.ops.resourceHealth.proxies.connectivityLastCheckedAt', { time: formatDateTime(proxy.connectivity_checked_at) })
      : t('admin.ops.resourceHealth.proxies.connectivityNeverChecked')
  }
  if (proxy.quality_stale) {
    return proxy.quality_checked_at
      ? t('admin.ops.resourceHealth.proxies.lastCheckedAt', { time: formatDateTime(proxy.quality_checked_at) })
      : t('admin.ops.resourceHealth.proxies.neverChecked')
  }
  return t('admin.ops.resourceHealth.proxies.checksCurrent')
}

function proxyAddress(proxy: OpsProxyHealthItem): string {
  return proxy.exit_ip || t('admin.ops.resourceHealth.proxies.unknownAddress')
}

function proxyLocation(proxy: OpsProxyHealthItem): string {
  return [proxy.city, proxy.region, proxy.country_code || proxy.country].filter(Boolean).join(', ')
}

function proxyMetadata(proxy: OpsProxyHealthItem): string {
  const parts = [proxy.protocol.toUpperCase(), t('admin.ops.resourceHealth.proxies.accountCount', { count: proxy.account_count })]
  if (typeof proxy.latency_ms === 'number') {
    parts.push(t('admin.ops.resourceHealth.proxies.latency', { value: proxy.latency_ms }))
  }
  const location = proxyLocation(proxy)
  if (location) parts.push(location)
  return parts.join(' / ')
}

function formatDateTime(value: string): string {
  const parsed = Date.parse(value)
  if (!Number.isFinite(parsed)) return t('admin.ops.resourceHealth.unknownTime')
  return new Intl.DateTimeFormat(locale.value || undefined, {
    month: 'short',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  }).format(new Date(parsed))
}

function formatCountdown(seconds?: number): string {
  if (typeof seconds !== 'number' || !Number.isFinite(seconds)) {
    return t('admin.ops.resourceHealth.detail.noCountdown')
  }
  const safe = Math.max(0, Math.floor(seconds))
  if (safe === 0) return t('admin.ops.resourceHealth.detail.countdownElapsed')
  const days = Math.floor(safe / 86_400)
  const hours = Math.floor((safe % 86_400) / 3_600)
  const minutes = Math.max(1, Math.floor((safe % 3_600) / 60))
  if (days > 0) return t('admin.ops.resourceHealth.detail.countdownDays', { days, hours })
  if (hours > 0) return t('admin.ops.resourceHealth.detail.countdownHours', { hours, minutes })
  return t('admin.ops.resourceHealth.detail.countdownMinutes', { minutes })
}

function countdownUntil(value?: string): string {
  if (!value) return t('admin.ops.resourceHealth.detail.noCountdown')
  const timestamp = Date.parse(value)
  if (!Number.isFinite(timestamp)) return t('admin.ops.resourceHealth.detail.noCountdown')
  return formatCountdown((timestamp - Date.now()) / 1000)
}

function proxyManagementHealth(proxy: OpsProxyHealthItem): string {
  if (proxy.lifecycle === 'expiring_soon') return 'expiring'
  return proxy.health
}

function canReprobe(proxy: OpsProxyHealthItem): boolean {
  return proxy.lifecycle === 'active' || proxy.lifecycle === 'expiring_soon'
}

function getErrorMessage(error: unknown, fallbackKey: string): string {
  if (typeof error === 'object' && error !== null) {
    const maybeResponse = error as { response?: { data?: { detail?: string } }; message?: string }
    if (maybeResponse.response?.data?.detail) return maybeResponse.response.data.detail
    if (maybeResponse.message) return maybeResponse.message
  }
  return t(fallbackKey)
}

function groupCapacityReason(group: OpsAccountPoolGroupSummary): string {
  if (group.zero_capacity) return t('admin.ops.resourceHealth.accounts.zeroCapacityReason')
  if (group.low_redundancy) {
    return t('admin.ops.resourceHealth.accounts.lowRedundancyReason', {
      count: group.base_schedulable_count
    })
  }
  return t('admin.ops.resourceHealth.accounts.healthyCapacityReason', {
    count: group.base_schedulable_count
  })
}

function groupCapacityPercent(group: OpsAccountPoolGroupSummary): number {
  if (group.total_accounts <= 0) return 0
  return Math.max(0, Math.min(100, Math.round((group.base_schedulable_count / group.total_accounts) * 100)))
}

function groupCapacityLabel(group: OpsAccountPoolGroupSummary): string {
  return t('admin.ops.resourceHealth.accounts.capacityValue', {
    available: group.base_schedulable_count,
    total: group.total_accounts
  })
}

function accountCompositionPercent(value: number): number {
  if (accountCompositionTotal.value <= 0) return 0
  return Math.max(0, Math.min(100, (value / accountCompositionTotal.value) * 100))
}

function accountScopeSeverity(account: OpsAccountPoolAnomaly): PendingSeverity {
  return account.category === 'auto_recovering' ? 'warning' : 'critical'
}

function proxySeverity(proxy: OpsProxyHealthItem): PendingSeverity {
  if (proxy.lifecycle === 'expired' || proxy.health === 'failed' || proxy.health === 'suspected_restricted') return 'critical'
  if (proxy.lifecycle !== 'active' || proxy.health !== 'healthy') return 'warning'
  return 'notice'
}

function proxyHealthLabel(proxy: OpsProxyHealthItem): string {
  return t(`admin.ops.resourceHealth.proxies.healthStatus.${proxy.health}`)
}

function proxyLifecycleLabel(proxy: OpsProxyHealthItem): string {
  return t(`admin.ops.resourceHealth.proxies.lifecycle.${proxy.lifecycle}`)
}

function accountRelatedObjects(account: OpsAccountPoolAnomaly): string {
  const parts = [...account.group_names]
  if (account.proxy_id) parts.push(t('admin.ops.resourceHealth.detail.proxyReference', { id: account.proxy_id }))
  return parts.length ? parts.join(' / ') : t('admin.ops.resourceHealth.detail.none')
}

function proxyPlatformCoverage(impact: OpsProxyHealthItem['platforms'][number]): string {
  if (impact.coverage === 'uncovered') {
    return t('admin.ops.resourceHealth.detail.coverage.uncovered')
  }
  const status = String(impact.status || '').trim().toLowerCase()
  if (['pass', 'warn', 'fail', 'challenge', 'unknown'].includes(status)) {
    return t(`admin.ops.resourceHealth.detail.coverage.${status}`)
  }
  return t('admin.ops.resourceHealth.detail.coverage.unknown')
}

function proxyRelatedObjects(proxy: OpsProxyHealthItem): string {
  const platforms = proxy.platforms.map((impact) =>
    t('admin.ops.resourceHealth.detail.platformAccounts', {
      platform: impact.platform,
      count: impact.account_count,
      status: proxyPlatformCoverage(impact)
    })
  )
  return platforms.length ? platforms.join(' / ') : t('admin.ops.resourceHealth.detail.none')
}

function detailTitleFor(selection: DetailSelection | null): string {
  if (!selection) return ''
  if (selection.kind === 'account') return selection.account.account_name || `#${selection.account.account_id}`
  if (selection.kind === 'group') return selection.group.group_name || `#${selection.group.group_id}`
  return selection.proxy.name || `#${selection.proxy.id}`
}

function detailKindFor(selection: DetailSelection | null): string {
  if (selection?.kind === 'account') return t('admin.ops.resourceHealth.detail.accountKind')
  if (selection?.kind === 'group') return t('admin.ops.resourceHealth.detail.groupKind')
  return t('admin.ops.resourceHealth.detail.proxyKind')
}

function detailRowsFor(selection: DetailSelection | null): Array<{ label: string; value: string }> {
  if (!selection) return []
  if (selection.kind === 'account') {
    return [
      { label: t('admin.ops.resourceHealth.detail.reason'), value: accountReason(selection.account) },
      {
        label: t('admin.ops.resourceHealth.detail.detectedAt'),
        value: accountSnapshot.value?.collected_at
          ? formatDateTime(accountSnapshot.value.collected_at)
          : t('admin.ops.resourceHealth.unknownTime')
      },
      { label: t('admin.ops.resourceHealth.detail.countdown'), value: formatCountdown(selection.account.remaining_seconds) },
      { label: t('admin.ops.resourceHealth.detail.relatedObjects'), value: accountRelatedObjects(selection.account) }
    ]
  }
  if (selection.kind === 'group') {
    return [
      { label: t('admin.ops.resourceHealth.detail.reason'), value: groupCapacityReason(selection.group) },
      {
        label: t('admin.ops.resourceHealth.detail.detectedAt'),
        value: accountSnapshot.value?.collected_at
          ? formatDateTime(accountSnapshot.value.collected_at)
          : t('admin.ops.resourceHealth.unknownTime')
      },
      { label: t('admin.ops.resourceHealth.detail.countdown'), value: t('admin.ops.resourceHealth.detail.noCountdown') },
      {
        label: t('admin.ops.resourceHealth.detail.relatedObjects'),
        value: t('admin.ops.resourceHealth.detail.groupAccounts', {
          platform: selection.group.platform,
          count: selection.group.total_accounts
        })
      }
    ]
  }
  return [
    { label: t('admin.ops.resourceHealth.detail.reason'), value: proxyAnomalyReason(selection.proxy) },
    { label: t('admin.ops.resourceHealth.detail.detectedAt'), value: proxyCheckedLabel(selection.proxy) },
    { label: t('admin.ops.resourceHealth.detail.countdown'), value: countdownUntil(selection.proxy.expires_at) },
    { label: t('admin.ops.resourceHealth.detail.relatedObjects'), value: proxyRelatedObjects(selection.proxy) }
  ]
}

const detailTitle = computed(() => detailTitleFor(selectedDetail.value))
const detailKind = computed(() => detailKindFor(selectedDetail.value))
const detailRows = computed(() => detailRowsFor(selectedDetail.value))
const detailPlatform = computed(() => {
  if (selectedDetail.value?.kind === 'account') return selectedDetail.value.account.platform
  if (selectedDetail.value?.kind === 'group') return selectedDetail.value.group.platform
  if (selectedDetail.value?.kind === 'proxy') return selectedDetail.value.proxy.protocol.toUpperCase()
  return t('admin.ops.resourceHealth.detail.none')
})
const detailResourceCount = computed(() => {
  if (selectedDetail.value?.kind === 'account') {
    return t('admin.ops.resourceHealth.detail.groupMembershipCount', {
      count: selectedDetail.value.account.group_ids.length
    })
  }
  if (selectedDetail.value?.kind === 'group') {
    return t('admin.ops.resourceHealth.detail.accountCount', { count: selectedDetail.value.group.total_accounts })
  }
  if (selectedDetail.value?.kind === 'proxy') {
    return t('admin.ops.resourceHealth.detail.accountCount', { count: selectedDetail.value.proxy.account_count })
  }
  return t('admin.ops.resourceHealth.detail.none')
})
const detailBlockingReason = computed(() => {
  if (selectedDetail.value?.kind === 'account') return accountReason(selectedDetail.value.account)
  if (selectedDetail.value?.kind === 'group') return groupCapacityReason(selectedDetail.value.group)
  if (selectedDetail.value?.kind === 'proxy') return proxyAnomalyReason(selectedDetail.value.proxy)
  return t('admin.ops.resourceHealth.detail.none')
})

function isSelectedGroup(group: OpsAccountPoolGroupSummary): boolean {
  return selectedDetail.value?.kind === 'group' && selectedDetail.value.group.group_id === group.group_id
}

function isSelectedAccount(account: OpsAccountPoolAnomaly): boolean {
  return selectedDetail.value?.kind === 'account' && selectedDetail.value.account.account_id === account.account_id
}

function isSelectedProxy(proxy: OpsProxyHealthItem): boolean {
  return selectedDetail.value?.kind === 'proxy' && selectedDetail.value.proxy.id === proxy.id
}

async function selectLedgerDetail(selection: DetailSelection, loadProxyDetail = true): Promise<void> {
  detailController?.abort()
  detailController = null
  detailLoading.value = false
  detailError.value = ''
  detailDrawerOpen.value = false
  selectedDetail.value = selection

  if (selection.kind !== 'proxy' || !loadProxyDetail) return
  const controller = new AbortController()
  detailController = controller
  detailLoading.value = true
  try {
    const detail = await opsAPI.getProxyHealthDetail(selection.proxy.id, { signal: controller.signal })
    if (!controller.signal.aborted && selectedDetail.value?.kind === 'proxy' && selectedDetail.value.proxy.id === detail.id) {
      applyProxyHealthItem(detail)
    }
  } catch (error) {
    if (!controller.signal.aborted) {
      detailError.value = getErrorMessage(error, 'admin.ops.resourceHealth.detail.loadFailed')
    }
  } finally {
    if (!controller.signal.aborted) detailLoading.value = false
  }
}

async function showDetail(selection: DetailSelection): Promise<void> {
  detailTriggerElement = document.activeElement instanceof HTMLElement ? document.activeElement : null
  previousBodyOverflow = document.body.style.overflow
  document.body.style.overflow = 'hidden'
  void selectLedgerDetail(selection)
  detailDrawerOpen.value = true
  await nextTick()
  detailDrawer.value?.focus()
}

function closeDetail(): void {
  detailController?.abort()
  detailController = null
  detailLoading.value = false
  detailError.value = ''
  detailDrawerOpen.value = false
  document.body.style.overflow = previousBodyOverflow
  const trigger = detailTriggerElement
  detailTriggerElement = null
  void nextTick(() => trigger?.focus())
}

function handleDrawerKeydown(event: KeyboardEvent): void {
  if (event.key === 'Escape') {
    event.preventDefault()
    closeDetail()
    return
  }
  if (event.key !== 'Tab' || !detailDrawer.value) return

  const focusable = Array.from(
    detailDrawer.value.querySelectorAll<HTMLElement>(
      'button:not([disabled]), a[href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'
    )
  ).filter((element) => !element.hasAttribute('hidden'))
  if (!focusable.length) {
    event.preventDefault()
    detailDrawer.value.focus()
    return
  }

  const first = focusable[0]
  const last = focusable[focusable.length - 1]
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault()
    last.focus()
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault()
    first.focus()
  }
}

async function loadResources(force = false): Promise<void> {
  const needsAccount = props.resource === 'overview' || props.resource === 'accounts'
  const needsProxy = props.resource === 'overview' || props.resource === 'proxies'
  const needsAlert = props.resource === 'overview'
  const shouldLoadAccount = needsAccount && (force || !accountSnapshot.value || Boolean(accountError.value))
  const shouldLoadProxy = needsProxy && (force || !proxySnapshot.value || Boolean(proxyError.value))
  const shouldLoadAlert = needsAlert && (force || !alertSnapshot.value || Boolean(alertError.value))

  if (!shouldLoadAccount && !shouldLoadProxy && !shouldLoadAlert) return

  activeController?.abort()
  const controller = new AbortController()
  activeController = controller
  const sequence = ++loadSequence

  accountLoading.value = shouldLoadAccount
  proxyLoading.value = shouldLoadProxy
  alertLoading.value = shouldLoadAlert
  if (shouldLoadAccount) accountError.value = ''
  if (shouldLoadProxy) proxyError.value = ''
  if (shouldLoadAlert) alertError.value = ''

  const alertParams: AlertEventsQuery = {
    limit: 100,
    status: 'firing'
  }
  if (props.platformFilter) alertParams.platform = props.platformFilter
  if (typeof props.groupIdFilter === 'number' && props.groupIdFilter > 0) {
    alertParams.group_id = props.groupIdFilter
  }

  const isCurrent = () => !controller.signal.aborted && sequence === loadSequence

  const requests: Promise<unknown>[] = []

  if (shouldLoadAccount) {
    requests.push(
      opsAPI
        .getAccountPool(props.platformFilter, props.groupIdFilter, ACCOUNT_ANOMALY_LIMIT, { signal: controller.signal })
        .then((value) => {
          if (isCurrent()) accountSnapshot.value = value
        })
        .catch((error) => {
          if (isCurrent()) accountError.value = getErrorMessage(error, 'admin.ops.resourceHealth.accountLoadFailed')
        })
        .finally(() => {
          if (isCurrent()) accountLoading.value = false
        })
    )
  }

  if (shouldLoadProxy) {
    requests.push(
      opsAPI
        .getProxyHealth({}, { signal: controller.signal })
        .then((value) => {
          if (isCurrent()) proxySnapshot.value = value
        })
        .catch((error) => {
          if (isCurrent()) proxyError.value = getErrorMessage(error, 'admin.ops.resourceHealth.proxyLoadFailed')
        })
        .finally(() => {
          if (isCurrent()) proxyLoading.value = false
        })
    )
  }

  if (shouldLoadAlert) {
    requests.push(
      opsAPI
        .listAlertEvents(alertParams, { signal: controller.signal })
        .then((value) => {
          if (isCurrent()) alertSnapshot.value = value
        })
        .catch((error) => {
          if (isCurrent()) alertError.value = getErrorMessage(error, 'admin.ops.resourceHealth.alertLoadFailed')
        })
        .finally(() => {
          if (isCurrent()) alertLoading.value = false
        })
    )
  }

  await Promise.allSettled(requests)
}

function accountHealthFilter(account: OpsAccountPoolAnomaly): string {
  switch (account.primary_reason) {
    case 'expired_auto_paused':
      return 'expired'
    case 'known_quota_exhausted':
      return 'quota_exhausted'
    case 'rate_limited':
      return 'rate_limited'
    case 'overloaded':
      return 'overloaded'
    case 'temporary_cooldown':
      return 'temp_unschedulable'
    default:
      return 'error'
  }
}

function accountRouteQuery(account?: OpsAccountPoolAnomaly): Record<string, string> {
  const query: Record<string, string> = {}
  if (props.platformFilter) query.platform = props.platformFilter
  if (typeof props.groupIdFilter === 'number' && props.groupIdFilter > 0) {
    query.group = String(props.groupIdFilter)
  }
  if (account) {
    query.health = accountHealthFilter(account)
    query.account_id = String(account.account_id)
    if (account.platform) query.platform = account.platform
    if (!query.group && account.group_ids[0]) query.group = String(account.group_ids[0])
    if (account.proxy_id) query.proxy_id = String(account.proxy_id)
  }
  return query
}

function openAccount(account?: OpsAccountPoolAnomaly): void {
  void router.push({ name: 'AdminAccounts', query: accountRouteQuery(account) })
}

function openGroup(group: OpsAccountPoolGroupSummary): void {
  void router.push({
    name: 'AdminAccounts',
    query: {
      health: 'unschedulable',
      platform: group.platform,
      group: String(group.group_id)
    }
  })
}

function openProxy(proxy?: OpsProxyHealthItem, healthOverride?: string): void {
  const query: Record<string, string> = { open: 'health' }
  if (proxy) {
    query.health = healthOverride || proxyManagementHealth(proxy)
    query.status = proxy.lifecycle === 'inactive' || proxy.lifecycle === 'expired' ? proxy.lifecycle : 'active'
    query.protocol = proxy.protocol
    query.focus_id = String(proxy.id)
  }
  void router.push({ name: 'AdminProxies', query })
}

function openPending(item: PendingItem): void {
  if (item.group) void showDetail({ kind: 'group', group: item.group })
  if (item.account) void showDetail({ kind: 'account', account: item.account })
  if (item.proxy) void showDetail({ kind: 'proxy', proxy: item.proxy })
}

function openSelectedResource(): void {
  const selection = selectedDetail.value
  if (!selection) return
  if (selection.kind === 'account') openAccount(selection.account)
  if (selection.kind === 'group') openGroup(selection.group)
  if (selection.kind === 'proxy') {
    openProxy(selection.proxy, isProxyStale(selection.proxy) ? 'stale' : proxyManagementHealth(selection.proxy))
  }
}

function setProxyTesting(proxyId: number, testing: boolean): void {
  const next = new Set(testingProxyIds.value)
  if (testing) next.add(proxyId)
  else next.delete(proxyId)
  testingProxyIds.value = next
}

function summarizeProxyHealth(items: OpsProxyHealthItem[]): OpsProxyHealthSummary {
  const summary: OpsProxyHealthSummary = {
    total: items.length,
    operational: 0,
    inactive: 0,
    expired: 0,
    expiring_soon: 0,
    healthy: 0,
    degraded: 0,
    suspected_restricted: 0,
    failed: 0,
    unknown: 0,
    affected_accounts: 0
  }

  for (const item of items) {
    if (item.lifecycle === 'inactive') summary.inactive += 1
    else if (item.lifecycle === 'expired') summary.expired += 1
    else {
      summary.operational += 1
      if (item.lifecycle === 'expiring_soon') summary.expiring_soon += 1
      summary[item.health] += 1
    }
    if (item.lifecycle === 'inactive' || item.lifecycle === 'expired' || item.health !== 'healthy') {
      summary.affected_accounts += item.active_account_count
    }
  }

  return summary
}

function applyProxyHealthItem(result: OpsProxyHealthItem): void {
  if (!proxySnapshot.value) return
  const items = proxySnapshot.value.items.map((proxy) => (proxy.id === result.id ? result : proxy))
  proxySnapshot.value = {
    ...proxySnapshot.value,
    summary: summarizeProxyHealth(items),
    items
  }
  if (selectedDetail.value?.kind === 'proxy' && selectedDetail.value.proxy.id === result.id) {
    selectedDetail.value = { kind: 'proxy', proxy: result }
  }
}

async function retestProxy(proxy: OpsProxyHealthItem): Promise<void> {
  if (!canReprobe(proxy) || testingProxyIds.value.has(proxy.id)) return
  setProxyTesting(proxy.id, true)
  proxyActionErrors.value = { ...proxyActionErrors.value, [proxy.id]: '' }
  actionAnnouncement.value = ''

  try {
    const result = await opsAPI.reprobeProxyHealth(proxy.id)
    applyProxyHealthItem(result)
    actionAnnouncement.value = t('admin.ops.resourceHealth.proxies.retestSuccess', { name: proxy.name })
  } catch (error) {
    proxyActionErrors.value = {
      ...proxyActionErrors.value,
      [proxy.id]: getErrorMessage(error, 'admin.ops.resourceHealth.proxies.retestFailed')
    }
  } finally {
    setProxyTesting(proxy.id, false)
  }
}

function syncLedgerSelection(): void {
  if (detailDrawerOpen.value) return

  if (props.resource === 'accounts') {
    if (accountLedgerScope.value === 'capacity') {
      const current = capacityRiskGroups.value.find((group) => isSelectedGroup(group))
      if (!current) {
        const first = capacityRiskGroups.value[0]
        selectedDetail.value = first ? { kind: 'group', group: first } : null
      }
      return
    }

    const accounts = accountLedgerScope.value === 'manual' ? visibleManualAccounts.value : visibleAutomaticAccounts.value
    const current = accounts.find((account) => isSelectedAccount(account))
    if (!current) {
      const first = accounts[0]
      selectedDetail.value = first ? { kind: 'account', account: first } : null
    }
    return
  }

  if (props.resource === 'proxies') {
    const list =
      proxyLedgerScope.value === 'stale'
        ? visibleStaleProxies.value
        : proxyLedgerScope.value === 'healthy'
          ? visibleHealthyProxies.value
          : visibleProxyIssues.value
    const current = list.find((proxy) => isSelectedProxy(proxy))
    if (!current) {
      const first = list[0]
      selectedDetail.value = first ? { kind: 'proxy', proxy: first } : null
    }
  }
}

watch(
  () => props.resource,
  () => {
    if (detailDrawerOpen.value) closeDetail()
    syncLedgerSelection()
    void loadResources(false)
  },
  { immediate: true }
)

watch(
  [() => props.platformFilter, () => props.groupIdFilter],
  () => {
    accountSnapshot.value = null
    alertSnapshot.value = null
    accountError.value = ''
    alertError.value = ''
    void loadResources(false)
  }
)

watch(
  () => props.refreshToken,
  () => {
    void loadResources(true)
  }
)

watch(
  [accountLedgerScope, proxyLedgerScope, accountSnapshot, proxySnapshot],
  () => {
    syncLedgerSelection()
  },
  { flush: 'post' }
)

onBeforeUnmount(() => {
  activeController?.abort()
  detailController?.abort()
  document.body.style.overflow = previousBodyOverflow
})
</script>

<template>
  <section
    class="ops-resource-health"
    data-testid="ops-resource-health"
    :aria-busy="accountLoading || proxyLoading || alertLoading"
  >
    <OpsWorkspaceNav
      class="ops-resource-health__nav"
      :model-value="resource"
      :items="navItems"
      :label="t('admin.ops.resourceHealth.nav.label')"
      :tab-semantics="false"
      @update:model-value="selectResource"
    />

    <p class="sr-only" aria-live="polite">{{ actionAnnouncement }}</p>

    <div
      :id="currentPanelId"
      class="ops-resource-health__panel"
      :class="{ 'ops-resource-health__panel--detached': resource !== 'proxies' }"
      role="region"
      :aria-label="currentPanelLabel"
      tabindex="0"
    >
      <div
        v-if="(resource !== 'proxies' && accountError) || (resource !== 'accounts' && proxyError) || (resource === 'overview' && alertError)"
        class="ops-resource-health__source-errors"
      >
        <div v-if="resource !== 'proxies' && accountError" class="ops-resource-health__source-alert" role="alert" data-testid="account-source-error">
          <span><strong>{{ t('admin.ops.resourceHealth.accountSource') }}</strong> {{ accountError }}</span>
          <button type="button" class="ops-resource-health__text-button" @click="loadResources(true)">
            {{ t('admin.ops.resourceHealth.retry') }}
          </button>
        </div>
        <div v-if="resource !== 'accounts' && proxyError" class="ops-resource-health__source-alert" role="alert" data-testid="proxy-source-error">
          <span><strong>{{ t('admin.ops.resourceHealth.proxySource') }}</strong> {{ proxyError }}</span>
          <button type="button" class="ops-resource-health__text-button" @click="loadResources(true)">
            {{ t('admin.ops.resourceHealth.retry') }}
          </button>
        </div>
        <div v-if="resource === 'overview' && alertError" class="ops-resource-health__source-alert" role="alert" data-testid="alert-source-error">
          <span><strong>{{ t('admin.ops.resourceHealth.alertSource') }}</strong> {{ alertError }}</span>
          <button type="button" class="ops-resource-health__text-button" @click="loadResources(true)">
            {{ t('admin.ops.resourceHealth.retry') }}
          </button>
        </div>
      </div>

      <div v-if="initialLoading" class="ops-resource-health__loading" role="status" data-testid="resource-loading">
        <span>{{ t('admin.ops.resourceHealth.loading') }}</span>
        <div class="ops-resource-health__skeleton-metrics" aria-hidden="true">
          <span v-for="index in 4" :key="index" />
        </div>
        <div class="ops-resource-health__skeleton-list" aria-hidden="true">
          <span v-for="index in 3" :key="index" />
        </div>
      </div>

      <template v-else-if="resource === 'overview'">
        <h3 class="sr-only">{{ t('admin.ops.resourceHealth.overview.title') }}</h3>

        <section
          class="ops-resource-health__capacity-viz"
          aria-labelledby="ops-overview-capacity-viz-title"
          data-testid="overview-capacity-viz"
        >
          <div class="ops-resource-health__capacity-signals" data-testid="overview-metrics">
            <div
              v-for="metric in overviewMetrics"
              :key="metric.key"
              class="ops-resource-health__metric ops-resource-health__signal-tile"
              :data-tone="metric.tone"
              :data-testid="`overview-metric-${metric.key}`"
            >
              <span>{{ metric.label }}</span>
              <strong>{{ metric.value }}</strong>
              <small>{{ metric.hint }}</small>
            </div>
          </div>

          <div class="ops-resource-health__composition">
            <header class="ops-resource-health__composition-header">
              <div>
                <span>{{ t('admin.ops.resourceHealth.composition.eyebrow') }}</span>
                <h4 id="ops-overview-capacity-viz-title">{{ t('admin.ops.resourceHealth.composition.title') }}</h4>
              </div>
              <span class="ops-resource-health__snapshot">
                {{ t('admin.ops.resourceHealth.composition.snapshot', { time: accountSnapshotTime }) }}
              </span>
            </header>

            <div class="ops-resource-health__composition-legend">
              <span v-for="segment in accountCompositionSegments" :key="segment.key" :data-tone="segment.tone">
                <i aria-hidden="true" />
                {{ segment.label }}
                <strong>{{ accountDataAvailable ? segment.value : '—' }}</strong>
              </span>
              <span
                v-if="(accountSummary?.quota_coverage_unknown_count || 0) > 0"
                class="ops-resource-health__quota-chip"
                data-testid="overview-quota-coverage"
              >
                {{ t('admin.ops.resourceHealth.composition.quotaUnknown', { count: accountSummary?.quota_coverage_unknown_count || 0 }) }}
              </span>
            </div>

            <div
              v-if="accountCompositionComplete"
              class="ops-resource-health__composition-stack"
              role="img"
              :aria-label="t('admin.ops.resourceHealth.composition.chartLabel', { total: accountCompositionTotal })"
              data-testid="account-composition-stack"
            >
              <i
                v-for="segment in accountCompositionSegments"
                :key="segment.key"
                :data-tone="segment.tone"
                :style="{ width: `${accountCompositionPercent(segment.value)}%` }"
                :title="`${segment.label}: ${segment.value}`"
              />
            </div>

            <div
              v-else-if="accountDataAvailable && accountCompositionTotal > 0"
              class="ops-resource-health__composition-fallback"
              data-testid="account-composition-fallback"
            >
              <div v-for="segment in accountCompositionSegments" :key="segment.key">
                <span>{{ segment.label }}</span>
                <span class="ops-resource-health__composition-track" aria-hidden="true">
                  <i :data-tone="segment.tone" :style="{ width: `${accountCompositionPercent(segment.value)}%` }" />
                </span>
                <strong>{{ segment.value }}</strong>
              </div>
            </div>

            <div v-else class="ops-resource-health__composition-empty" role="status">
              {{ accountDataAvailable ? t('admin.ops.resourceHealth.composition.empty') : t('admin.ops.resourceHealth.metrics.dataUnavailable') }}
            </div>

            <p v-if="accountDataAvailable && accountCompositionTotal > 0 && !accountCompositionComplete" class="ops-resource-health__composition-note" role="note">
              {{ t('admin.ops.resourceHealth.composition.incomplete', { sum: accountCompositionSum, total: accountCompositionTotal }) }}
            </p>
            <p v-else class="ops-resource-health__composition-note">
              {{ t('admin.ops.resourceHealth.composition.description') }}
            </p>
          </div>
        </section>

        <div class="ops-resource-health__cockpit-grid" data-testid="overview-cockpit">
          <section class="ops-resource-health__list-section" aria-labelledby="ops-overview-capacity-title">
            <div class="ops-resource-health__list-heading">
              <div>
                <h4 id="ops-overview-capacity-title">{{ t('admin.ops.resourceHealth.accounts.capacityTitle') }}</h4>
                <p>{{ t('admin.ops.resourceHealth.accounts.capacityDescription') }}</p>
              </div>
              <span>{{ capacityRiskGroups.length }}</span>
            </div>
            <ul v-if="overviewCapacityRiskGroups.length" class="ops-resource-health__capacity-rows" data-testid="overview-capacity-list">
              <li v-for="group in overviewCapacityRiskGroups" :key="group.group_id" class="ops-resource-health__capacity-row">
                <button type="button" class="ops-resource-health__capacity-trigger" @click="showDetail({ kind: 'group', group })">
                  <span class="ops-resource-health__capacity-copy">
                    <strong>{{ group.group_name || `#${group.group_id}` }}</strong>
                    <small>{{ group.platform }}</small>
                  </span>
                  <span class="ops-resource-health__capacity-value">{{ groupCapacityLabel(group) }}</span>
                  <span class="ops-resource-health__capacity-track" aria-hidden="true">
                    <i :style="{ width: `${groupCapacityPercent(group)}%` }" :data-severity="group.zero_capacity ? 'critical' : 'warning'" />
                  </span>
                  <span class="ops-resource-health__capacity-reason">{{ groupCapacityReason(group) }}</span>
                </button>
              </li>
            </ul>
            <div v-else class="ops-resource-health__empty" role="status">
              <strong>{{ t('admin.ops.resourceHealth.accounts.noCapacityRiskTitle') }}</strong>
              <span>{{ t('admin.ops.resourceHealth.accounts.noCapacityRiskDescription') }}</span>
            </div>

            <p v-if="accountSnapshot && !accountSnapshot.group_counts_additive" class="ops-resource-health__inline-note" role="note">
              {{ t('admin.ops.resourceHealth.accounts.groupCountsNotAdditive') }}
            </p>

            <details v-if="healthyGroups.length" class="ops-resource-health__health-disclosure" data-testid="overview-healthy-group-disclosure">
              <summary>{{ t('admin.ops.resourceHealth.accounts.healthyGroups', { count: healthyGroups.length }) }}</summary>
              <ul class="ops-resource-health__rows">
                <li v-for="group in healthyGroups.slice(0, DETAIL_LIST_LIMIT)" :key="group.group_id" class="ops-resource-health__row">
                  <div class="ops-resource-health__row-main">
                    <strong>{{ group.group_name || `#${group.group_id}` }}</strong>
                    <span class="ops-resource-health__metadata">{{ groupCapacityLabel(group) }}</span>
                  </div>
                  <button type="button" class="ops-resource-health__row-action" @click="showDetail({ kind: 'group', group })">
                    {{ t('admin.ops.resourceHealth.viewDetails') }}
                  </button>
                </li>
              </ul>
            </details>
          </section>

          <section class="ops-resource-health__list-section" aria-labelledby="ops-resource-pending-title">
            <div class="ops-resource-health__list-heading">
              <div>
                <h4 id="ops-resource-pending-title">{{ t('admin.ops.resourceHealth.pending.title') }}</h4>
                <p>{{ t('admin.ops.resourceHealth.pending.description') }}</p>
              </div>
              <span>{{ pendingItems.length }}/5</span>
            </div>

            <ul v-if="pendingItems.length" class="ops-resource-health__rows" data-testid="pending-list">
              <li v-for="item in pendingItems" :key="item.key" class="ops-resource-health__row">
                <div class="ops-resource-health__row-main">
                  <span class="ops-resource-health__badge" :data-severity="item.severity">{{ item.kindLabel }}</span>
                  <strong>{{ item.title }}</strong>
                  <p>{{ item.detail }}</p>
                </div>
                <button type="button" class="ops-resource-health__row-action" @click="openPending(item)">
                  {{ t('admin.ops.resourceHealth.viewDetails') }}
                </button>
              </li>
            </ul>
            <div v-else class="ops-resource-health__empty" role="status" data-testid="pending-empty">
              <strong>{{ t('admin.ops.resourceHealth.pending.emptyTitle') }}</strong>
              <span>{{ t('admin.ops.resourceHealth.pending.emptyDescription') }}</span>
            </div>
            <p v-if="pendingTotalCount > pendingItems.length" class="ops-resource-health__truncated-note">
              {{ t('admin.ops.resourceHealth.showingFirst', { count: pendingItems.length, total: pendingTotalCount }) }}
            </p>
          </section>
        </div>
      </template>

      <template v-else-if="resource === 'accounts'">
        <h3 id="ops-account-summary-title" class="sr-only">{{ t('admin.ops.resourceHealth.accounts.title') }}</h3>

        <section class="ops-resource-health__account-summary" aria-labelledby="ops-account-summary-title">
          <div class="ops-resource-health__metrics" data-testid="account-metrics">
            <div
              v-for="metric in accountMetrics"
              :key="metric.key"
              class="ops-resource-health__metric"
              :data-tone="metric.tone"
            >
              <span>{{ metric.label }}</span>
              <strong>{{ metric.value }}</strong>
            </div>
          </div>

          <div
            v-if="(accountSnapshot && !accountSnapshot.group_counts_additive) || (accountSummary?.quota_coverage_unknown_count || 0) > 0"
            class="ops-resource-health__notes"
          >
            <p
              v-if="accountSnapshot && !accountSnapshot.group_counts_additive"
              class="ops-resource-health__nonadditive-note"
              role="note"
              data-testid="group-counts-note"
            >
              {{ t('admin.ops.resourceHealth.accounts.groupCountsNotAdditive') }}
            </p>
            <p
              v-if="(accountSummary?.quota_coverage_unknown_count || 0) > 0"
              class="ops-resource-health__quota-note"
              role="note"
              data-testid="quota-coverage-note"
            >
              {{ t('admin.ops.resourceHealth.accounts.quotaCoverageUnknown', { count: accountSummary?.quota_coverage_unknown_count || 0 }) }}
            </p>
          </div>
        </section>

        <section class="ops-resource-health__ledger-shell ops-resource-health__ledger-shell--account" data-testid="account-ledger">
          <div class="ops-resource-health__ledger-toolbar">
            <div class="ops-resource-health__segments" role="group" :aria-label="t('admin.ops.resourceHealth.accounts.workspaceTitle')">
              <button type="button" :aria-pressed="accountLedgerScope === 'capacity'" @click="accountLedgerScope = 'capacity'">
                {{ t('admin.ops.resourceHealth.accounts.capacityTitle') }} <span>{{ capacityRiskGroups.length }}</span>
              </button>
              <button type="button" :aria-pressed="accountLedgerScope === 'manual'" @click="accountLedgerScope = 'manual'">
                {{ t('admin.ops.resourceHealth.accounts.manualBadge') }} <span>{{ accountSummary?.actionable_count || 0 }}</span>
              </button>
              <button type="button" :aria-pressed="accountLedgerScope === 'automatic'" @click="accountLedgerScope = 'automatic'">
                {{ t('admin.ops.resourceHealth.accounts.automaticBadge') }} <span>{{ accountSummary?.auto_recovering_count || 0 }}</span>
              </button>
            </div>
            <button type="button" class="ops-resource-health__secondary-button" @click="openAccount()">
              {{ t('admin.ops.resourceHealth.accounts.openAll') }}
            </button>
          </div>

          <div class="ops-resource-health__ledger ops-resource-health__ledger--account">
            <div class="ops-resource-health__ledger-main">
              <div class="ops-resource-health__ledger-table" :aria-busy="accountLoading" data-testid="account-ledger-table">
                <table v-if="accountLedgerScope === 'capacity' && capacityRiskGroups.length" data-testid="capacity-risk-list">
                  <thead><tr><th>{{ t('admin.ops.resourceHealth.columns.status') }}</th><th>{{ t('admin.ops.resourceHealth.columns.name') }}</th><th>{{ t('admin.ops.resourceHealth.columns.platform') }}</th><th>{{ t('admin.ops.resourceHealth.columns.capacity') }}</th><th>{{ t('admin.ops.resourceHealth.columns.reason') }}</th><th><span class="sr-only">{{ t('admin.ops.resourceHealth.columns.actions') }}</span></th></tr></thead>
                  <tbody>
                    <tr v-for="group in capacityRiskGroups" :key="group.group_id" tabindex="0" :aria-selected="isSelectedGroup(group)" :data-severity="group.zero_capacity ? 'critical' : 'warning'" @click="selectLedgerDetail({ kind: 'group', group }, false)" @keydown.enter.prevent="selectLedgerDetail({ kind: 'group', group }, false)" @keydown.space.self.prevent="selectLedgerDetail({ kind: 'group', group }, false)">
                      <td><span class="ops-resource-health__badge" :data-severity="group.zero_capacity ? 'critical' : 'warning'">{{ group.zero_capacity ? t('admin.ops.resourceHealth.accounts.zeroCapacity') : t('admin.ops.resourceHealth.accounts.lowRedundancy') }}</span></td>
                      <td><strong>{{ group.group_name || `#${group.group_id}` }}</strong><small>#{{ group.group_id }}</small></td>
                      <td :data-label="t('admin.ops.resourceHealth.columns.platform')">{{ group.platform }}</td>
                      <td class="ops-resource-health__numeric" :data-label="t('admin.ops.resourceHealth.columns.capacity')">{{ groupCapacityLabel(group) }}</td>
                      <td :data-label="t('admin.ops.resourceHealth.columns.reason')">{{ groupCapacityReason(group) }}</td>
                      <td><button type="button" class="ops-resource-health__row-action" @click.stop="showDetail({ kind: 'group', group })">{{ t('admin.ops.resourceHealth.viewDetails') }}</button></td>
                    </tr>
                  </tbody>
                </table>

                <table v-else-if="accountLedgerScope === 'manual' && visibleManualAccounts.length" data-testid="manual-account-list">
                  <thead><tr><th>{{ t('admin.ops.resourceHealth.columns.status') }}</th><th>{{ t('admin.ops.resourceHealth.columns.name') }}</th><th>{{ t('admin.ops.resourceHealth.columns.platform') }}</th><th>{{ t('admin.ops.resourceHealth.columns.capacity') }}</th><th>{{ t('admin.ops.resourceHealth.columns.reason') }}</th><th><span class="sr-only">{{ t('admin.ops.resourceHealth.columns.actions') }}</span></th></tr></thead>
                  <tbody>
                    <tr v-for="account in visibleManualAccounts" :key="account.account_id" tabindex="0" :aria-selected="isSelectedAccount(account)" :data-severity="accountScopeSeverity(account)" @click="selectLedgerDetail({ kind: 'account', account }, false)" @keydown.enter.prevent="selectLedgerDetail({ kind: 'account', account }, false)" @keydown.space.self.prevent="selectLedgerDetail({ kind: 'account', account }, false)">
                      <td><span class="ops-resource-health__badge" :data-severity="accountScopeSeverity(account)">{{ t('admin.ops.resourceHealth.accounts.manualBadge') }}</span></td>
                      <td><strong>{{ account.account_name || `#${account.account_id}` }}</strong><small>{{ account.group_names.join(' / ') || t('admin.ops.resourceHealth.detail.none') }}</small></td>
                      <td :data-label="t('admin.ops.resourceHealth.columns.platform')">{{ account.platform }}</td>
                      <td class="ops-resource-health__numeric" :data-label="t('admin.ops.resourceHealth.columns.capacity')">—</td>
                      <td :data-label="t('admin.ops.resourceHealth.columns.reason')">{{ manualRecoveryReason(account) }}</td>
                      <td><button type="button" class="ops-resource-health__row-action" @click.stop="showDetail({ kind: 'account', account })">{{ t('admin.ops.resourceHealth.viewDetails') }}</button></td>
                    </tr>
                  </tbody>
                </table>

                <table v-else-if="accountLedgerScope === 'automatic' && visibleAutomaticAccounts.length" data-testid="automatic-account-list">
                  <thead><tr><th>{{ t('admin.ops.resourceHealth.columns.status') }}</th><th>{{ t('admin.ops.resourceHealth.columns.name') }}</th><th>{{ t('admin.ops.resourceHealth.columns.platform') }}</th><th>{{ t('admin.ops.resourceHealth.columns.capacity') }}</th><th>{{ t('admin.ops.resourceHealth.columns.reason') }}</th><th><span class="sr-only">{{ t('admin.ops.resourceHealth.columns.actions') }}</span></th></tr></thead>
                  <tbody>
                    <tr v-for="account in visibleAutomaticAccounts" :key="account.account_id" tabindex="0" :aria-selected="isSelectedAccount(account)" :data-severity="accountScopeSeverity(account)" @click="selectLedgerDetail({ kind: 'account', account }, false)" @keydown.enter.prevent="selectLedgerDetail({ kind: 'account', account }, false)" @keydown.space.self.prevent="selectLedgerDetail({ kind: 'account', account }, false)">
                      <td><span class="ops-resource-health__badge" :data-severity="accountScopeSeverity(account)">{{ t('admin.ops.resourceHealth.accounts.automaticBadge') }}</span></td>
                      <td><strong>{{ account.account_name || `#${account.account_id}` }}</strong><small>{{ account.group_names.join(' / ') || t('admin.ops.resourceHealth.detail.none') }}</small></td>
                      <td :data-label="t('admin.ops.resourceHealth.columns.platform')">{{ account.platform }}</td>
                      <td class="ops-resource-health__numeric" :data-label="t('admin.ops.resourceHealth.columns.capacity')">—</td>
                      <td :data-label="t('admin.ops.resourceHealth.columns.reason')">{{ automaticRecoveryReason(account) }}</td>
                      <td><button type="button" class="ops-resource-health__row-action" @click.stop="showDetail({ kind: 'account', account })">{{ t('admin.ops.resourceHealth.viewDetails') }}</button></td>
                    </tr>
                  </tbody>
                </table>

                <div v-else class="ops-resource-health__empty" role="status">
                  <strong>{{ accountLedgerScope === 'capacity' ? t('admin.ops.resourceHealth.accounts.noCapacityRiskTitle') : accountLedgerScope === 'manual' ? t('admin.ops.resourceHealth.accounts.noManualTitle') : t('admin.ops.resourceHealth.accounts.noAutomaticTitle') }}</strong>
                  <span>{{ accountLedgerScope === 'capacity' ? t('admin.ops.resourceHealth.accounts.noCapacityRiskDescription') : accountLedgerScope === 'manual' ? t('admin.ops.resourceHealth.accounts.noManualDescription') : t('admin.ops.resourceHealth.accounts.noAutomaticDescription') }}</span>
                </div>

                <p v-if="accountLedgerScope === 'manual' && accountLedgerCount > manualRecoveryAccounts.length" class="ops-resource-health__truncated-note">{{ t('admin.ops.resourceHealth.showingFirst', { count: manualRecoveryAccounts.length, total: accountLedgerCount }) }}</p>
                <p v-if="accountLedgerScope === 'automatic' && accountLedgerCount > automaticRecoveryAccounts.length" class="ops-resource-health__truncated-note">{{ t('admin.ops.resourceHealth.showingFirst', { count: automaticRecoveryAccounts.length, total: accountLedgerCount }) }}</p>
              </div>

              <details v-if="healthyGroups.length" class="ops-resource-health__health-disclosure ops-resource-health__health-disclosure--ledger" data-testid="healthy-group-disclosure">
                <summary>{{ t('admin.ops.resourceHealth.accounts.healthyGroups', { count: healthyGroups.length }) }}</summary>
                <ul class="ops-resource-health__rows">
                  <li v-for="group in healthyGroups.slice(0, DETAIL_LIST_LIMIT)" :key="group.group_id" class="ops-resource-health__row">
                    <div class="ops-resource-health__row-main">
                      <strong>{{ group.group_name || `#${group.group_id}` }}</strong>
                      <span class="ops-resource-health__metadata">{{ t('admin.ops.resourceHealth.accounts.schedulableInGroup', { count: group.base_schedulable_count }) }}</span>
                    </div>
                    <button type="button" class="ops-resource-health__row-action" @click="showDetail({ kind: 'group', group })">{{ t('admin.ops.resourceHealth.viewDetails') }}</button>
                  </li>
                </ul>
              </details>
            </div>

            <aside class="ops-resource-health__ledger-preview" :aria-label="t('admin.ops.resourceHealth.detail.previewTitle')" data-testid="ledger-preview">
              <template v-if="selectedDetail">
                <header><span>{{ t('admin.ops.resourceHealth.detail.previewTitle') }}</span><h4>{{ detailTitle }}</h4><small>{{ detailKind }}</small></header>
                <dl class="ops-resource-health__preview-summary">
                  <div><dt>{{ t('admin.ops.resourceHealth.detail.platform') }}</dt><dd>{{ detailPlatform }}</dd></div>
                  <div><dt>{{ t('admin.ops.resourceHealth.detail.resourceCount') }}</dt><dd>{{ detailResourceCount }}</dd></div>
                </dl>
                <div class="ops-resource-health__preview-cause"><span>{{ t('admin.ops.resourceHealth.detail.blockingReason') }}</span><strong>{{ detailBlockingReason }}</strong></div>
                <footer><button type="button" class="ops-resource-health__secondary-button ops-resource-health__secondary-button--full" @click="openSelectedResource">{{ t('admin.ops.resourceHealth.detail.openManagement') }}</button></footer>
              </template>
              <div v-else class="ops-resource-health__preview-empty">{{ t('admin.ops.resourceHealth.detail.selectPrompt') }}</div>
            </aside>
          </div>
        </section>
      </template>

      <template v-else>
        <header class="ops-resource-health__panel-header">
          <div>
            <h3>{{ t('admin.ops.resourceHealth.proxies.title') }}</h3>
            <p>{{ t('admin.ops.resourceHealth.proxies.description') }}</p>
            <p v-if="platformFilter || groupIdFilter" class="ops-resource-health__scope-note">
              {{ t('admin.ops.resourceHealth.proxies.globalScope') }}
            </p>
          </div>
          <button type="button" class="ops-resource-health__secondary-button" @click="openProxy()">
            {{ t('admin.ops.resourceHealth.proxies.openAll') }}
          </button>
        </header>

        <div
          v-if="proxySnapshot?.data_status === 'partial'"
          class="ops-resource-health__partial-notice"
          role="status"
          data-testid="proxy-data-partial"
        >
          {{ t('admin.ops.resourceHealth.proxies.partialData') }}
        </div>

        <div class="ops-resource-health__metrics" data-testid="proxy-metrics">
          <div
            v-for="metric in proxyMetrics"
            :key="metric.key"
            class="ops-resource-health__metric"
            :data-tone="metric.tone"
          >
            <span>{{ metric.label }}</span>
            <strong>{{ metric.value }}</strong>
          </div>
        </div>

        <section class="ops-resource-health__ledger-shell" data-testid="proxy-ledger">
          <div class="ops-resource-health__ledger-toolbar">
            <div><h4>{{ t('admin.ops.resourceHealth.proxies.workspaceTitle') }}</h4><p>{{ t('admin.ops.resourceHealth.proxies.workspaceDescription') }}</p></div>
            <div class="ops-resource-health__segments" role="group" :aria-label="t('admin.ops.resourceHealth.proxies.workspaceTitle')">
              <button type="button" :aria-pressed="proxyLedgerScope === 'issues'" @click="proxyLedgerScope = 'issues'">{{ t('admin.ops.resourceHealth.proxies.scopeIssues') }} <span>{{ proxyIssues.length }}</span></button>
              <button type="button" :aria-pressed="proxyLedgerScope === 'stale'" @click="proxyLedgerScope = 'stale'">{{ t('admin.ops.resourceHealth.proxies.scopeStale') }} <span>{{ staleProxies.length }}</span></button>
              <button type="button" :aria-pressed="proxyLedgerScope === 'healthy'" @click="proxyLedgerScope = 'healthy'">{{ t('admin.ops.resourceHealth.proxies.scopeHealthy') }} <span>{{ healthyProxies.length }}</span></button>
            </div>
          </div>

          <div class="ops-resource-health__ledger">
            <div class="ops-resource-health__ledger-table" :aria-busy="proxyLoading" data-testid="proxy-ledger-table">
              <table v-if="proxyLedgerCount" :data-testid="`proxy-${proxyLedgerScope}-list`">
                <thead><tr><th>{{ t('admin.ops.resourceHealth.columns.name') }}</th><th>{{ t('admin.ops.resourceHealth.columns.status') }}</th><th>{{ t('admin.ops.resourceHealth.columns.impact') }}</th><th>{{ t('admin.ops.resourceHealth.columns.connection') }}</th><th>{{ t('admin.ops.resourceHealth.columns.lastCheck') }}</th><th><span class="sr-only">{{ t('admin.ops.resourceHealth.columns.actions') }}</span></th></tr></thead>
                <tbody>
                  <tr v-for="proxy in proxyLedgerScope === 'stale' ? visibleStaleProxies : proxyLedgerScope === 'healthy' ? visibleHealthyProxies : visibleProxyIssues" :key="proxy.id" tabindex="0" :aria-selected="isSelectedProxy(proxy)" @click="selectLedgerDetail({ kind: 'proxy', proxy })" @keydown.enter.prevent="selectLedgerDetail({ kind: 'proxy', proxy })">
                    <td><strong>{{ proxy.name || `#${proxy.id}` }}</strong><small class="ops-resource-health__address">{{ proxyAddress(proxy) }}</small></td>
                    <td><span class="ops-resource-health__badge" :data-severity="proxySeverity(proxy)">{{ proxyHealthLabel(proxy) }}</span><small>{{ proxyLifecycleLabel(proxy) }}</small></td>
                    <td class="ops-resource-health__numeric">{{ t('admin.ops.resourceHealth.proxies.activeAccountCount', { count: proxy.active_account_count }) }}</td>
                    <td>{{ proxyMetadata(proxy) }}</td>
                    <td>{{ proxyCheckedLabel(proxy) }}<small v-if="proxyActionErrors[proxy.id]" class="ops-resource-health__action-error" role="alert">{{ proxyActionErrors[proxy.id] }}</small></td>
                    <td><button type="button" class="ops-resource-health__row-action" @click.stop="showDetail({ kind: 'proxy', proxy })">{{ t('admin.ops.resourceHealth.viewDetails') }}</button></td>
                  </tr>
                </tbody>
              </table>
              <div v-else class="ops-resource-health__empty" role="status"><strong>{{ proxyLedgerScope === 'stale' ? t('admin.ops.resourceHealth.proxies.noStaleTitle') : proxyLedgerScope === 'healthy' ? t('admin.ops.resourceHealth.proxies.noHealthyTitle') : t('admin.ops.resourceHealth.proxies.noAnomalyTitle') }}</strong><span>{{ proxyLedgerScope === 'stale' ? t('admin.ops.resourceHealth.proxies.noStaleDescription') : proxyLedgerScope === 'healthy' ? t('admin.ops.resourceHealth.proxies.noHealthyDescription') : t('admin.ops.resourceHealth.proxies.noAnomalyDescription') }}</span></div>
              <p v-if="proxyLedgerCount > DETAIL_LIST_LIMIT" class="ops-resource-health__truncated-note">{{ t('admin.ops.resourceHealth.showingFirst', { count: DETAIL_LIST_LIMIT, total: proxyLedgerCount }) }}</p>
            </div>

            <aside class="ops-resource-health__ledger-preview" :aria-label="t('admin.ops.resourceHealth.detail.previewTitle')" data-testid="ledger-preview">
              <template v-if="selectedDetail?.kind === 'proxy'">
                <header><span>{{ detailKind }}</span><h4>{{ detailTitle }}</h4></header>
                <p v-if="detailLoading" class="ops-resource-health__drawer-state" role="status">{{ t('admin.ops.resourceHealth.detail.loading') }}</p>
                <p v-if="detailError" class="ops-resource-health__drawer-state ops-resource-health__drawer-state--error" role="alert">{{ detailError }}</p>
                <dl class="ops-resource-health__detail-list"><div v-for="row in detailRows" :key="row.label"><dt>{{ row.label }}</dt><dd>{{ row.value }}</dd></div></dl>
                <footer>
                  <button type="button" class="ops-resource-health__secondary-button" @click="openSelectedResource">{{ t('admin.ops.resourceHealth.detail.openManagement') }}</button>
                  <button type="button" class="ops-resource-health__retest-button" :disabled="testingProxyIds.has(selectedDetail.proxy.id) || !canReprobe(selectedDetail.proxy)" :title="!canReprobe(selectedDetail.proxy) ? t('admin.ops.resourceHealth.proxies.retestUnavailable') : undefined" @click="retestProxy(selectedDetail.proxy)">{{ testingProxyIds.has(selectedDetail.proxy.id) ? t('admin.ops.resourceHealth.proxies.retesting') : t('admin.ops.resourceHealth.proxies.retest') }}</button>
                </footer>
              </template>
              <div v-else class="ops-resource-health__preview-empty">{{ t('admin.ops.resourceHealth.detail.selectPrompt') }}</div>
            </aside>
          </div>
        </section>
      </template>
    </div>

    <div
      v-if="detailDrawerOpen && selectedDetail"
      class="ops-resource-health__drawer-backdrop"
      data-testid="resource-detail-backdrop"
      @click.self="closeDetail"
      @keydown.stop="handleDrawerKeydown"
    >
      <aside
        ref="detailDrawer"
        class="ops-resource-health__drawer"
        role="dialog"
        aria-modal="true"
        aria-labelledby="ops-resource-detail-title"
        tabindex="-1"
        data-testid="resource-detail-drawer"
      >
        <header class="ops-resource-health__drawer-header">
          <div>
            <span>{{ detailKind }}</span>
            <h3 id="ops-resource-detail-title">{{ detailTitle }}</h3>
          </div>
          <button type="button" class="ops-resource-health__drawer-close" :aria-label="t('admin.ops.resourceHealth.detail.close')" @click="closeDetail">
            ×
          </button>
        </header>

        <p v-if="detailLoading" class="ops-resource-health__drawer-state" role="status">
          {{ t('admin.ops.resourceHealth.detail.loading') }}
        </p>
        <p v-if="detailError" class="ops-resource-health__drawer-state ops-resource-health__drawer-state--error" role="alert">
          {{ detailError }}
        </p>

        <dl class="ops-resource-health__detail-list">
          <div v-for="row in detailRows" :key="row.label">
            <dt>{{ row.label }}</dt>
            <dd>{{ row.value }}</dd>
          </div>
        </dl>

        <footer class="ops-resource-health__drawer-footer">
          <button type="button" class="ops-resource-health__secondary-button" data-testid="detail-open-management" @click="openSelectedResource">
            {{ t('admin.ops.resourceHealth.detail.openManagement') }}
          </button>
          <button
            v-if="selectedDetail.kind === 'proxy'"
            type="button"
            class="ops-resource-health__retest-button"
            data-testid="detail-reprobe"
            :disabled="testingProxyIds.has(selectedDetail.proxy.id) || !canReprobe(selectedDetail.proxy)"
            :title="!canReprobe(selectedDetail.proxy) ? t('admin.ops.resourceHealth.proxies.retestUnavailable') : undefined"
            @click="retestProxy(selectedDetail.proxy)"
          >
            {{ testingProxyIds.has(selectedDetail.proxy.id) ? t('admin.ops.resourceHealth.proxies.retesting') : t('admin.ops.resourceHealth.proxies.retest') }}
          </button>
          <button type="button" class="ops-resource-health__text-button" @click="closeDetail">
            {{ t('admin.ops.resourceHealth.detail.close') }}
          </button>
        </footer>
      </aside>
    </div>
  </section>
</template>

<style scoped>
.ops-resource-health {
  min-width: 0;
  color: var(--lx-clay-text);
  font-family: var(--lx-clay-font-ui);
}

.ops-resource-health__panel {
  min-width: 0;
  margin-top: 8px;
  overflow: hidden;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-surface);
  background: var(--lx-clay-surface);
}

.ops-resource-health__panel--detached {
  overflow: visible;
  border: 0;
  background: transparent;
}

.ops-resource-health__panel--detached > .ops-resource-health__source-errors,
.ops-resource-health__panel--detached > .ops-resource-health__loading {
  overflow: hidden;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-surface);
  background: var(--lx-clay-surface);
  box-shadow: var(--lx-clay-shadow-flat);
}

.ops-resource-health__panel:focus-visible {
  outline: 3px solid color-mix(in srgb, var(--lx-clay-accent) 34%, transparent);
  outline-offset: 3px;
}

.ops-resource-health__panel-header,
.ops-resource-health__list-heading,
.ops-resource-health__source-alert,
.ops-resource-health__row,
.ops-resource-health__row-actions {
  display: flex;
  align-items: center;
}

.ops-resource-health__panel-header {
  justify-content: space-between;
  gap: 20px;
  padding: 12px 16px;
}

.ops-resource-health__panel-header h3,
.ops-resource-health__list-heading h4 {
  margin: 0;
  color: var(--lx-clay-text);
}

.ops-resource-health__panel-header h3 {
  font-size: 1rem;
  font-weight: 800;
  line-height: 1.3;
}

.ops-resource-health__panel-header p,
.ops-resource-health__list-heading p,
.ops-resource-health__row-main p,
.ops-resource-health__truncated-note {
  margin: 0;
  color: var(--lx-clay-text-secondary);
}

.ops-resource-health__panel-header p {
  max-width: 62ch;
  margin-top: 4px;
  font-size: 0.78rem;
  line-height: 1.5;
}

.ops-resource-health__panel-header .ops-resource-health__scope-note {
  color: var(--lx-clay-warning);
  font-size: 0.72rem;
}

.ops-resource-health__freshness,
.ops-resource-health__metadata,
.ops-resource-health__address,
.ops-resource-health__truncated-note {
  font-size: 0.72rem;
}

.ops-resource-health__freshness,
.ops-resource-health__metadata {
  color: var(--lx-clay-text-muted);
}

.ops-resource-health__metrics {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 1px;
  border-top: 1px solid var(--lx-clay-border);
  border-bottom: 1px solid var(--lx-clay-border);
  background: var(--lx-clay-border);
}

.ops-resource-health__account-summary {
  overflow: hidden;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-surface);
  background: var(--lx-clay-surface);
  box-shadow: var(--lx-clay-shadow-flat);
}

.ops-resource-health__account-summary .ops-resource-health__metrics {
  border-top: 0;
  border-bottom: 0;
}

.ops-resource-health__metric {
  min-width: 0;
  padding: 11px 14px;
  background: var(--lx-clay-surface-soft);
}

.ops-resource-health__account-summary .ops-resource-health__metric {
  padding: 14px 18px;
  background: var(--lx-clay-surface);
}

.ops-resource-health__account-summary .ops-resource-health__metric span {
  min-height: 0;
}

.ops-resource-health__account-summary .ops-resource-health__metric strong {
  font-size: 1.25rem;
}

.ops-resource-health__metric span,
.ops-resource-health__metric strong,
.ops-resource-health__metric small {
  display: block;
}

.ops-resource-health__metric span {
  min-height: 2.4em;
  color: var(--lx-clay-text-secondary);
  font-size: 0.72rem;
  font-weight: 700;
  line-height: 1.2;
}

.ops-resource-health__metric strong {
  margin-top: 5px;
  color: var(--lx-clay-text);
  font-family: var(--lx-clay-font-mono);
  font-size: 1.35rem;
  font-variant-numeric: tabular-nums;
  line-height: 1;
}

.ops-resource-health__metric small {
  margin-top: 5px;
  color: var(--lx-clay-text-muted);
  font-size: 0.64rem;
  line-height: 1.3;
}

.ops-resource-health__metric[data-tone='success'] strong {
  color: var(--lx-clay-success);
}

.ops-resource-health__metric[data-tone='warning'] strong {
  color: var(--lx-clay-warning);
}

.ops-resource-health__metric[data-tone='danger'] strong {
  color: var(--lx-clay-danger);
}

.ops-resource-health__nav {
  position: sticky;
  z-index: 19;
  top: calc(var(--app-shell-top-offset) + 68px);
  padding: 3px;
  margin: -3px;
  border-radius: var(--lx-clay-radius-control);
  background: var(--app-shell-canvas, var(--lx-clay-canvas));
}

.ops-resource-health__capacity-viz {
  display: grid;
  overflow: hidden;
  grid-template-columns: minmax(0, 4fr) minmax(0, 8fr);
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-surface);
  background: var(--lx-clay-surface);
  box-shadow: var(--lx-clay-shadow-flat);
}

.ops-resource-health__capacity-signals {
  display: grid;
  min-width: 0;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 1px;
  align-content: stretch;
  border-right: 1px solid var(--lx-clay-border);
  background: var(--lx-clay-border);
}

.ops-resource-health__signal-tile {
  min-width: 0;
  min-height: 108px;
  padding: 15px 16px;
  background: var(--lx-clay-surface-soft);
}

.ops-resource-health__signal-tile span,
.ops-resource-health__signal-tile strong,
.ops-resource-health__signal-tile small {
  display: block;
}

.ops-resource-health__signal-tile span {
  color: var(--lx-clay-text-secondary);
  font-size: 0.7rem;
  font-weight: 800;
  line-height: 1.3;
}

.ops-resource-health__signal-tile strong {
  margin-top: 8px;
  color: var(--lx-clay-text);
  font-family: var(--lx-clay-font-mono);
  font-size: 1.32rem;
  font-variant-numeric: tabular-nums;
  line-height: 1;
}

.ops-resource-health__signal-tile small {
  margin-top: 7px;
  color: var(--lx-clay-text-muted);
  font-size: 0.65rem;
  line-height: 1.4;
}

.ops-resource-health__signal-tile[data-tone='success'] strong {
  color: var(--lx-clay-success);
}

.ops-resource-health__signal-tile[data-tone='warning'] strong {
  color: var(--lx-clay-warning);
}

.ops-resource-health__signal-tile[data-tone='danger'] strong {
  color: var(--lx-clay-danger);
}

.ops-resource-health__composition {
  min-width: 0;
  padding: 18px 20px;
}

.ops-resource-health__composition-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.ops-resource-health__composition-header span,
.ops-resource-health__snapshot {
  color: var(--lx-clay-text-muted);
  font-size: 0.66rem;
  font-weight: 800;
  letter-spacing: 0.045em;
}

.ops-resource-health__composition-header h4 {
  margin: 4px 0 0;
  color: var(--lx-clay-text);
  font-size: 0.92rem;
  font-weight: 850;
}

.ops-resource-health__snapshot {
  max-width: 44%;
  letter-spacing: 0;
  text-align: right;
}

.ops-resource-health__composition-legend {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 14px;
  margin-top: 17px;
}

.ops-resource-health__composition-legend > span {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  color: var(--lx-clay-text-secondary);
  font-size: 0.69rem;
}

.ops-resource-health__composition-legend i {
  width: 8px;
  height: 8px;
  border-radius: 3px;
  background: var(--lx-clay-text-muted);
}

.ops-resource-health__composition-legend strong {
  color: var(--lx-clay-text);
  font-family: var(--lx-clay-font-mono);
  font-variant-numeric: tabular-nums;
}

.ops-resource-health__composition-legend [data-tone='success'] i,
.ops-resource-health__composition-stack [data-tone='success'],
.ops-resource-health__composition-track [data-tone='success'] {
  background: var(--lx-clay-success);
}

.ops-resource-health__composition-legend [data-tone='warning'] i,
.ops-resource-health__composition-stack [data-tone='warning'],
.ops-resource-health__composition-track [data-tone='warning'] {
  background: var(--lx-clay-warning);
}

.ops-resource-health__composition-legend [data-tone='danger'] i,
.ops-resource-health__composition-stack [data-tone='danger'],
.ops-resource-health__composition-track [data-tone='danger'] {
  background: var(--lx-clay-danger);
}

.ops-resource-health__composition-legend [data-tone='neutral'] i,
.ops-resource-health__composition-stack [data-tone='neutral'],
.ops-resource-health__composition-track [data-tone='neutral'] {
  background: var(--lx-clay-text-muted);
}

.ops-resource-health__composition-legend .ops-resource-health__quota-chip {
  padding: 4px 8px;
  border: 1px solid var(--lx-clay-border);
  border-radius: 999px;
  color: var(--lx-clay-text-secondary);
  background: var(--lx-clay-recessed);
  font-weight: 750;
}

.ops-resource-health__composition-stack {
  display: flex;
  height: 32px;
  margin-top: 16px;
  overflow: hidden;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-ops);
  background: var(--lx-clay-recessed);
}

.ops-resource-health__composition-stack i {
  display: block;
  min-width: 0;
  height: 100%;
}

.ops-resource-health__composition-stack i + i {
  border-left: 1px solid color-mix(in srgb, var(--lx-clay-surface) 66%, transparent);
}

.ops-resource-health__composition-fallback {
  display: grid;
  gap: 8px;
  margin-top: 16px;
}

.ops-resource-health__composition-fallback > div {
  display: grid;
  grid-template-columns: minmax(90px, 0.7fr) minmax(80px, 2fr) auto;
  align-items: center;
  gap: 10px;
  color: var(--lx-clay-text-secondary);
  font-size: 0.68rem;
}

.ops-resource-health__composition-fallback strong {
  min-width: 2ch;
  color: var(--lx-clay-text);
  font-family: var(--lx-clay-font-mono);
  font-variant-numeric: tabular-nums;
  text-align: right;
}

.ops-resource-health__composition-track {
  height: 7px;
  overflow: hidden;
  border-radius: 999px;
  background: var(--lx-clay-recessed);
}

.ops-resource-health__composition-track i {
  display: block;
  height: 100%;
  border-radius: inherit;
}

.ops-resource-health__composition-empty {
  display: grid;
  min-height: 72px;
  margin-top: 16px;
  place-items: center;
  border: 1px dashed var(--lx-clay-border-strong);
  border-radius: var(--lx-clay-radius-ops);
  color: var(--lx-clay-text-muted);
  background: var(--lx-clay-surface-soft);
  font-size: 0.72rem;
}

.ops-resource-health__composition-note,
.ops-resource-health__inline-note {
  margin: 12px 0 0;
  color: var(--lx-clay-text-muted);
  font-size: 0.68rem;
  line-height: 1.5;
}

.ops-resource-health__inline-note {
  padding-top: 10px;
  border-top: 1px solid var(--lx-clay-border);
}

.ops-resource-health__cockpit-grid {
  display: grid;
  grid-template-columns: minmax(0, 7fr) minmax(300px, 5fr);
  gap: 20px;
  margin-top: 20px;
}

.ops-resource-health__cockpit-grid > .ops-resource-health__list-section + .ops-resource-health__list-section {
  border-left: 1px solid var(--lx-clay-border);
}

.ops-resource-health__cockpit-grid > .ops-resource-health__list-section {
  overflow: hidden;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-surface);
  background: var(--lx-clay-surface);
  box-shadow: var(--lx-clay-shadow-flat);
}

.ops-resource-health__cockpit-grid > .ops-resource-health__list-section + .ops-resource-health__list-section {
  border-left-color: var(--lx-clay-border);
}

.ops-resource-health__capacity-rows {
  margin: 0;
  padding: 0;
  list-style: none;
}

.ops-resource-health__capacity-row + .ops-resource-health__capacity-row {
  border-top: 1px solid var(--lx-clay-border);
}

.ops-resource-health__capacity-trigger {
  display: grid;
  width: 100%;
  min-height: 60px;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 5px 14px;
  padding: 9px 0;
  border: 0;
  color: inherit;
  background: transparent;
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.ops-resource-health__capacity-trigger:hover {
  background: var(--lx-clay-surface-soft);
}

.ops-resource-health__capacity-trigger:focus-visible {
  outline: 3px solid color-mix(in srgb, var(--lx-clay-accent) 34%, transparent);
  outline-offset: -3px;
}

.ops-resource-health__capacity-copy,
.ops-resource-health__capacity-value,
.ops-resource-health__capacity-reason {
  min-width: 0;
}

.ops-resource-health__capacity-copy strong,
.ops-resource-health__capacity-copy small {
  display: block;
}

.ops-resource-health__capacity-copy strong {
  overflow: hidden;
  font-size: 0.8rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ops-resource-health__capacity-copy small,
.ops-resource-health__capacity-reason {
  color: var(--lx-clay-text-muted);
  font-size: 0.72rem;
}

.ops-resource-health__capacity-value {
  color: var(--lx-clay-text-secondary);
  font-family: var(--lx-clay-font-mono);
  font-size: 0.75rem;
  font-variant-numeric: tabular-nums;
  font-weight: 800;
}

.ops-resource-health__capacity-track {
  height: 5px;
  overflow: hidden;
  border-radius: 999px;
  background: var(--lx-clay-recessed);
}

.ops-resource-health__capacity-track i {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--lx-clay-warning);
}

.ops-resource-health__capacity-track i[data-severity='critical'] {
  background: var(--lx-clay-danger);
}

.ops-resource-health__capacity-reason {
  overflow: hidden;
  text-align: right;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ops-resource-health__ledger-shell {
  border-top: 1px solid var(--lx-clay-border);
}

.ops-resource-health__ledger-shell--account {
  margin-top: 20px;
  overflow: hidden;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-surface);
  background: var(--lx-clay-surface);
  box-shadow: var(--lx-clay-shadow-flat);
}

.ops-resource-health__ledger-toolbar {
  display: flex;
  min-width: 0;
  align-items: flex-end;
  justify-content: space-between;
  gap: 18px;
  padding: 14px 16px;
  border-bottom: 1px solid var(--lx-clay-border);
}

.ops-resource-health__ledger-shell--account .ops-resource-health__ledger-toolbar {
  align-items: center;
  padding: 12px 16px;
  background: var(--lx-clay-surface);
}

.ops-resource-health__ledger-toolbar h4,
.ops-resource-health__ledger-toolbar p,
.ops-resource-health__ledger-preview h4 {
  margin: 0;
}

.ops-resource-health__ledger-toolbar h4,
.ops-resource-health__ledger-preview h4 {
  color: var(--lx-clay-text);
  font-size: 0.86rem;
  font-weight: 800;
}

.ops-resource-health__ledger-toolbar p {
  margin-top: 3px;
  color: var(--lx-clay-text-secondary);
  font-size: 0.72rem;
  line-height: 1.4;
}

.ops-resource-health__segments {
  display: flex;
  flex: 0 0 auto;
  gap: 3px;
  padding: 3px;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-control);
  background: var(--lx-clay-recessed);
}

.ops-resource-health__segments button {
  min-height: 44px;
  padding: 0 11px;
  border: 1px solid transparent;
  border-radius: var(--lx-clay-radius-ops);
  color: var(--lx-clay-text-secondary);
  background: transparent;
  font: inherit;
  font-size: 0.72rem;
  font-weight: 800;
  cursor: pointer;
}

.ops-resource-health__segments button[aria-pressed='true'] {
  border-color: color-mix(in srgb, var(--lx-clay-accent) 20%, transparent);
  color: var(--lx-clay-accent);
  background: var(--lx-clay-surface);
}

.ops-resource-health__segments button:focus-visible {
  outline: 3px solid color-mix(in srgb, var(--lx-clay-accent) 34%, transparent);
  outline-offset: 1px;
}

.ops-resource-health__segments span {
  margin-left: 4px;
  font-family: var(--lx-clay-font-mono);
  font-variant-numeric: tabular-nums;
}

.ops-resource-health__ledger {
  display: grid;
  min-height: 320px;
  grid-template-columns: minmax(0, 7fr) minmax(280px, 5fr);
}

.ops-resource-health__ledger--account {
  min-height: 0;
  grid-template-columns: minmax(0, 1fr) 340px;
  align-items: start;
  gap: 20px;
  padding: 20px;
  background: var(--app-shell-canvas, var(--lx-clay-canvas));
}

.ops-resource-health__ledger-main {
  min-width: 0;
  overflow: hidden;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-ops);
  background: var(--lx-clay-surface);
}

.ops-resource-health__ledger-table {
  min-width: 0;
  overflow: auto;
}

.ops-resource-health__ledger-table table {
  width: 100%;
  border-collapse: collapse;
  color: var(--lx-clay-text-secondary);
  font-size: 0.75rem;
  line-height: 1.4;
}

.ops-resource-health__ledger-table thead {
  position: sticky;
  z-index: 2;
  top: 0;
  background: var(--lx-clay-surface-soft);
}

.ops-resource-health__ledger-table th,
.ops-resource-health__ledger-table td {
  padding: 10px 12px;
  border-bottom: 1px solid var(--lx-clay-border);
  text-align: left;
  vertical-align: middle;
}

.ops-resource-health__ledger-table th {
  color: var(--lx-clay-text-muted);
  font-size: 0.68rem;
  font-weight: 800;
  letter-spacing: 0.02em;
  white-space: nowrap;
}

.ops-resource-health__ledger-shell--account .ops-resource-health__ledger-table th {
  padding: 12px 16px;
  font-size: 0.66rem;
  font-weight: 850;
  letter-spacing: 0.055em;
  text-transform: uppercase;
}

.ops-resource-health__ledger-shell--account .ops-resource-health__ledger-table td {
  padding: 14px 16px;
  font-size: 0.78rem;
}

.ops-resource-health__ledger-table tbody tr {
  min-height: 52px;
  cursor: pointer;
}

.ops-resource-health__ledger-table tbody tr:hover {
  background: var(--lx-clay-surface-soft);
}

.ops-resource-health__ledger-table tbody tr[aria-selected='true'] {
  background: var(--lx-clay-accent-soft);
}

.ops-resource-health__ledger-shell--account .ops-resource-health__ledger-table tbody tr td:first-child {
  border-left: 4px solid var(--lx-clay-text-muted);
}

.ops-resource-health__ledger-shell--account .ops-resource-health__ledger-table tbody tr[data-severity='critical'] td:first-child {
  border-left-color: var(--lx-clay-danger);
}

.ops-resource-health__ledger-shell--account .ops-resource-health__ledger-table tbody tr[data-severity='warning'] td:first-child {
  border-left-color: var(--lx-clay-warning);
}

.ops-resource-health__ledger-shell--account .ops-resource-health__ledger-table tbody tr[data-severity='notice'] td:first-child {
  border-left-color: var(--lx-clay-accent);
}

.ops-resource-health__ledger-table tbody tr:focus-visible {
  outline: 3px solid color-mix(in srgb, var(--lx-clay-accent) 34%, transparent);
  outline-offset: -3px;
}

.ops-resource-health__ledger-table td > strong,
.ops-resource-health__ledger-table td > small {
  display: block;
}

.ops-resource-health__ledger-table td > strong {
  max-width: 210px;
  overflow: hidden;
  color: var(--lx-clay-text);
  font-size: 0.78rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ops-resource-health__ledger-table td > small {
  max-width: 230px;
  margin-top: 3px;
  color: var(--lx-clay-text-muted);
  font-size: 0.69rem;
  line-height: 1.35;
  overflow-wrap: anywhere;
}

.ops-resource-health__numeric {
  font-family: var(--lx-clay-font-mono);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.ops-resource-health__ledger-preview {
  min-width: 0;
  border-left: 1px solid var(--lx-clay-border);
  background: var(--lx-clay-surface-soft);
}

.ops-resource-health__ledger--account .ops-resource-health__ledger-preview {
  overflow: hidden;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-ops);
  background: var(--lx-clay-surface);
  box-shadow: var(--lx-clay-shadow-flat);
}

.ops-resource-health__ledger-preview > header {
  padding: 16px 18px 12px;
  border-bottom: 1px solid var(--lx-clay-border);
}

.ops-resource-health__ledger-preview > header span {
  display: block;
  margin-bottom: 4px;
  color: var(--lx-clay-text-muted);
  font-size: 0.68rem;
  font-weight: 800;
}

.ops-resource-health__ledger-preview > header small {
  display: block;
  margin-top: 5px;
  color: var(--lx-clay-text-muted);
  font-size: 0.68rem;
}

.ops-resource-health__ledger-preview > footer {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding: 14px 18px 18px;
  border-top: 1px solid var(--lx-clay-border);
}

.ops-resource-health__preview-summary {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 1px;
  margin: 16px 18px 0;
  overflow: hidden;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-ops);
  background: var(--lx-clay-border);
}

.ops-resource-health__preview-summary > div {
  min-width: 0;
  padding: 12px;
  background: var(--lx-clay-surface-soft);
}

.ops-resource-health__preview-summary dt,
.ops-resource-health__preview-cause span {
  color: var(--lx-clay-text-muted);
  font-size: 0.66rem;
  font-weight: 800;
}

.ops-resource-health__preview-summary dd {
  margin: 5px 0 0;
  overflow: hidden;
  color: var(--lx-clay-text);
  font-family: var(--lx-clay-font-mono);
  font-size: 0.75rem;
  font-variant-numeric: tabular-nums;
  font-weight: 800;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ops-resource-health__preview-cause {
  margin: 14px 18px 18px;
  padding: 14px;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-ops);
  background: var(--lx-clay-recessed);
}

.ops-resource-health__preview-cause strong {
  display: block;
  margin-top: 7px;
  color: var(--lx-clay-text);
  font-size: 0.76rem;
  line-height: 1.5;
}

.ops-resource-health__secondary-button--full {
  width: 100%;
}

.ops-resource-health__preview-empty {
  display: grid;
  min-height: 100%;
  place-items: center;
  padding: 24px;
  color: var(--lx-clay-text-muted);
  font-size: 0.75rem;
  text-align: center;
}

.ops-resource-health__split {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.ops-resource-health__split > .ops-resource-health__list-section + .ops-resource-health__list-section {
  border-left: 1px solid var(--lx-clay-border);
}

.ops-resource-health__list-section {
  min-width: 0;
  padding: 18px 20px 20px;
}

.ops-resource-health__list-heading {
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 12px;
}

.ops-resource-health__list-heading h4 {
  font-size: 0.83rem;
  font-weight: 800;
}

.ops-resource-health__list-heading p {
  margin-top: 3px;
  font-size: 0.72rem;
  line-height: 1.4;
}

.ops-resource-health__list-heading > span {
  min-width: 30px;
  color: var(--lx-clay-text-secondary);
  font-family: var(--lx-clay-font-mono);
  font-size: 0.75rem;
  font-weight: 750;
  text-align: right;
}

.ops-resource-health__rows {
  margin: 0;
  padding: 0;
  list-style: none;
}

.ops-resource-health__row {
  min-width: 0;
  justify-content: space-between;
  gap: 14px;
  padding: 11px 0;
}

.ops-resource-health__row + .ops-resource-health__row {
  border-top: 1px solid var(--lx-clay-border);
}

.ops-resource-health__row-main {
  min-width: 0;
}

.ops-resource-health__row-main > strong,
.ops-resource-health__row-main > span {
  display: block;
}

.ops-resource-health__row-main > strong {
  margin-top: 5px;
  overflow: hidden;
  color: var(--lx-clay-text);
  font-size: 0.79rem;
  font-weight: 800;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ops-resource-health__row-main p {
  margin-top: 4px;
  overflow-wrap: anywhere;
  font-size: 0.72rem;
  line-height: 1.45;
}

.ops-resource-health__metadata,
.ops-resource-health__address {
  margin-top: 3px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ops-resource-health__address {
  color: var(--lx-clay-text-secondary);
  font-family: var(--lx-clay-font-mono);
  font-variant-numeric: tabular-nums;
}

.ops-resource-health__badge {
  width: fit-content;
  max-width: 100%;
  padding: 3px 7px;
  border-radius: 7px;
  color: var(--lx-clay-text-secondary);
  background: var(--lx-clay-recessed);
  font-size: 0.66rem;
  font-weight: 800;
  line-height: 1.2;
}

.ops-resource-health__badge[data-severity='critical'] {
  color: var(--lx-clay-danger);
  background: var(--lx-clay-danger-soft);
}

.ops-resource-health__badge[data-severity='warning'] {
  color: var(--lx-clay-warning);
  background: var(--lx-clay-warning-soft);
}

.ops-resource-health__badge[data-severity='notice'] {
  color: var(--lx-clay-accent);
  background: var(--lx-clay-accent-soft);
}

.ops-resource-health__row-actions {
  flex: 0 0 auto;
  gap: 6px;
}

.ops-resource-health__row-action,
.ops-resource-health__text-button,
.ops-resource-health__secondary-button,
.ops-resource-health__retest-button {
  min-height: 44px;
  border-radius: var(--lx-clay-radius-control);
  font: inherit;
  font-size: 0.72rem;
  font-weight: 800;
  line-height: 1;
  white-space: nowrap;
  cursor: pointer;
}

.ops-resource-health__row-action,
.ops-resource-health__text-button {
  border: 1px solid transparent;
  color: var(--lx-clay-accent);
  background: transparent;
}

.ops-resource-health__row-action {
  flex: 0 0 auto;
  padding: 0 10px;
}

.ops-resource-health__text-button {
  padding: 0 8px;
}

.ops-resource-health__secondary-button,
.ops-resource-health__retest-button {
  border: 1px solid var(--lx-clay-border-strong);
  color: var(--lx-clay-text);
  background: var(--lx-clay-surface-soft);
}

.ops-resource-health__secondary-button {
  padding: 0 14px;
}

.ops-resource-health__retest-button {
  padding: 0 11px;
}

.ops-resource-health__row-action:hover,
.ops-resource-health__text-button:hover,
.ops-resource-health__secondary-button:hover,
.ops-resource-health__retest-button:hover:not(:disabled) {
  border-color: var(--lx-clay-border-strong);
  background: var(--lx-clay-recessed);
}

.ops-resource-health__row-action:active,
.ops-resource-health__text-button:active,
.ops-resource-health__secondary-button:active,
.ops-resource-health__retest-button:active:not(:disabled) {
  transform: translateY(1px);
}

.ops-resource-health__row-action:focus-visible,
.ops-resource-health__text-button:focus-visible,
.ops-resource-health__secondary-button:focus-visible,
.ops-resource-health__retest-button:focus-visible {
  outline: 3px solid color-mix(in srgb, var(--lx-clay-accent) 34%, transparent);
  outline-offset: 1px;
}

.ops-resource-health__retest-button:disabled {
  cursor: wait;
  opacity: 0.58;
}

.ops-resource-health__empty,
.ops-resource-health__disabled {
  display: flex;
  min-height: 112px;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  gap: 5px;
  border: 1px dashed var(--lx-clay-border-strong);
  border-radius: var(--lx-clay-radius-ops);
  color: var(--lx-clay-text-secondary);
  background: var(--lx-clay-surface-soft);
  text-align: center;
}

.ops-resource-health__empty strong {
  color: var(--lx-clay-text);
  font-size: 0.78rem;
}

.ops-resource-health__empty span,
.ops-resource-health__disabled {
  font-size: 0.72rem;
}

.ops-resource-health__disabled {
  min-height: auto;
  margin: 0 20px 16px;
  padding: 12px;
}

.ops-resource-health__truncated-note {
  margin-top: 10px;
  text-align: center;
}

.ops-resource-health__nonadditive-note {
  margin: 0;
  padding: 10px 20px;
  border-bottom: 1px solid var(--lx-clay-border);
  color: var(--lx-clay-warning);
  background: var(--lx-clay-warning-soft);
  font-size: 0.72rem;
  line-height: 1.45;
}

.ops-resource-health__notes {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  border-bottom: 1px solid var(--lx-clay-border);
}

.ops-resource-health__account-summary .ops-resource-health__notes {
  border-top: 1px solid var(--lx-clay-border);
  border-bottom: 0;
}

.ops-resource-health__notes > p {
  border-bottom: 0;
}

.ops-resource-health__notes > p + p {
  border-left: 1px solid var(--lx-clay-border);
}

.ops-resource-health__quota-note {
  margin: 0;
  padding: 10px 20px;
  border-bottom: 1px solid var(--lx-clay-border);
  color: var(--lx-clay-text-secondary);
  background: var(--lx-clay-surface-soft);
  font-size: 0.72rem;
  line-height: 1.45;
}

.ops-resource-health__health-disclosure {
  margin: 0 20px 20px;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-ops);
  background: var(--lx-clay-surface-soft);
}

.ops-resource-health__list-section .ops-resource-health__health-disclosure,
.ops-resource-health__health-disclosure--ledger {
  margin: 14px 0 0;
}

.ops-resource-health__health-disclosure--ledger {
  border-right: 0;
  border-bottom: 0;
  border-left: 0;
  border-radius: 0;
}

.ops-resource-health__health-disclosure > summary {
  display: flex;
  min-height: 44px;
  align-items: center;
  gap: 8px;
  padding: 0 14px;
  color: var(--lx-clay-text-secondary);
  font-size: 0.75rem;
  font-weight: 800;
  cursor: pointer;
}

.ops-resource-health__health-disclosure > summary::before {
  content: '+';
  color: var(--lx-clay-accent);
  font-family: var(--lx-clay-font-mono);
  font-size: 0.9rem;
}

.ops-resource-health__health-disclosure[open] > summary::before {
  content: '−';
}

.ops-resource-health__health-disclosure[open] > summary {
  border-bottom: 1px solid var(--lx-clay-border);
}

.ops-resource-health__health-disclosure > .ops-resource-health__rows {
  padding: 0 14px;
}

.ops-resource-health__drawer-backdrop {
  position: fixed;
  z-index: 80;
  inset: 0;
  display: flex;
  justify-content: flex-end;
  background: color-mix(in srgb, var(--lx-clay-text) 28%, transparent);
}

.ops-resource-health__drawer {
  width: min(420px, calc(100vw - 24px));
  height: 100%;
  overflow-y: auto;
  border-left: 1px solid var(--lx-clay-border-strong);
  background: var(--lx-clay-surface);
  box-shadow: -16px 0 36px color-mix(in srgb, var(--lx-clay-text) 18%, transparent);
}

.ops-resource-health__drawer:focus-visible {
  outline: 3px solid color-mix(in srgb, var(--lx-clay-accent) 34%, transparent);
  outline-offset: -3px;
}

.ops-resource-health__drawer-header,
.ops-resource-health__drawer-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 16px 18px;
}

.ops-resource-health__drawer-header {
  border-bottom: 1px solid var(--lx-clay-border);
}

.ops-resource-health__drawer-header span {
  color: var(--lx-clay-text-muted);
  font-size: 0.69rem;
  font-weight: 800;
}

.ops-resource-health__drawer-header h3 {
  margin: 4px 0 0;
  color: var(--lx-clay-text);
  font-size: 1rem;
}

.ops-resource-health__drawer-close {
  width: 44px;
  min-width: 44px;
  height: 44px;
  border: 1px solid transparent;
  border-radius: var(--lx-clay-radius-control);
  color: var(--lx-clay-text-secondary);
  background: transparent;
  font: inherit;
  font-size: 1.35rem;
  cursor: pointer;
}

.ops-resource-health__drawer-close:hover,
.ops-resource-health__drawer-close:focus-visible {
  border-color: var(--lx-clay-border-strong);
  background: var(--lx-clay-recessed);
}

.ops-resource-health__detail-list {
  margin: 0;
  padding: 8px 18px 18px;
}

.ops-resource-health__drawer-state {
  margin: 0;
  padding: 10px 18px;
  border-bottom: 1px solid var(--lx-clay-border);
  color: var(--lx-clay-text-secondary);
  background: var(--lx-clay-surface-soft);
  font-size: 0.72rem;
}

.ops-resource-health__drawer-state--error {
  color: var(--lx-clay-danger);
  background: var(--lx-clay-danger-soft);
}

.ops-resource-health__detail-list > div {
  padding: 14px 0;
  border-bottom: 1px solid var(--lx-clay-border);
}

.ops-resource-health__detail-list dt {
  color: var(--lx-clay-text-muted);
  font-size: 0.68rem;
  font-weight: 800;
}

.ops-resource-health__detail-list dd {
  margin: 5px 0 0;
  color: var(--lx-clay-text);
  font-size: 0.78rem;
  line-height: 1.5;
  overflow-wrap: anywhere;
}

.ops-resource-health__drawer-footer {
  flex-wrap: wrap;
  border-top: 1px solid var(--lx-clay-border);
}

.ops-resource-health__source-errors {
  border-bottom: 1px solid var(--lx-clay-border);
}

.ops-resource-health__partial-notice {
  padding: 10px 20px;
  border-top: 1px solid var(--lx-clay-border);
  color: var(--lx-clay-warning);
  background: var(--lx-clay-warning-soft);
  font-size: 0.72rem;
  line-height: 1.45;
}

.ops-resource-health__source-alert {
  min-height: 44px;
  justify-content: space-between;
  gap: 12px;
  padding: 6px 12px 6px 16px;
  color: var(--lx-clay-danger);
  background: var(--lx-clay-danger-soft);
  font-size: 0.72rem;
  line-height: 1.45;
}

.ops-resource-health__source-alert + .ops-resource-health__source-alert {
  border-top: 1px solid var(--lx-clay-border);
}

.ops-resource-health__action-error {
  color: var(--lx-clay-danger) !important;
}

.ops-resource-health__loading {
  padding: 18px 20px 20px;
  color: var(--lx-clay-text-secondary);
  font-size: 0.76rem;
}

.ops-resource-health__skeleton-metrics {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 1px;
  margin-top: 12px;
  overflow: hidden;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-ops);
  background: var(--lx-clay-border);
}

.ops-resource-health__skeleton-metrics span {
  height: 64px;
  background: var(--lx-clay-recessed);
}

.ops-resource-health__skeleton-list {
  margin-top: 12px;
}

.ops-resource-health__skeleton-list span {
  display: block;
  height: 52px;
  border-top: 1px solid var(--lx-clay-border);
  background: var(--lx-clay-surface-soft);
}

@container ops-dashboard (max-width: 1080px) {
  .ops-resource-health__ledger--account {
    grid-template-columns: minmax(0, 1fr);
  }

  .ops-resource-health__ledger--account .ops-resource-health__ledger-preview {
    display: none;
  }
}

@media (max-width: 800px) {
  .ops-resource-health__metrics,
  .ops-resource-health__skeleton-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .ops-resource-health__split,
  .ops-resource-health__cockpit-grid,
  .ops-resource-health__ledger {
    grid-template-columns: minmax(0, 1fr);
  }

  .ops-resource-health__capacity-viz {
    grid-template-columns: minmax(0, 1fr);
  }

  .ops-resource-health__capacity-signals {
    border-right: 0;
    border-bottom: 1px solid var(--lx-clay-border);
  }

  .ops-resource-health__split > .ops-resource-health__list-section + .ops-resource-health__list-section,
  .ops-resource-health__cockpit-grid > .ops-resource-health__list-section + .ops-resource-health__list-section {
    border-top: 1px solid var(--lx-clay-border);
    border-left: 0;
  }

  .ops-resource-health__ledger-toolbar {
    align-items: stretch;
    flex-direction: column;
    gap: 10px;
  }

  .ops-resource-health__ledger-shell--account .ops-resource-health__ledger-toolbar {
    align-items: stretch;
  }

  .ops-resource-health__ledger-shell--account .ops-resource-health__secondary-button {
    width: 100%;
  }

  .ops-resource-health__segments {
    max-width: 100%;
    overflow-x: auto;
  }

  .ops-resource-health__ledger-preview {
    border-top: 1px solid var(--lx-clay-border);
    border-left: 0;
  }

  .ops-resource-health__ledger--account {
    padding: 12px;
  }
}

@media (max-width: 420px) {
  .ops-resource-health__panel-header,
  .ops-resource-health__list-section,
  .ops-resource-health__loading,
  .ops-resource-health__ledger-toolbar {
    padding-right: 12px;
    padding-left: 12px;
  }

  .ops-resource-health__composition {
    padding: 15px 12px;
  }

  .ops-resource-health__composition-header {
    flex-direction: column;
    gap: 6px;
  }

  .ops-resource-health__snapshot {
    max-width: none;
    text-align: left;
  }

  .ops-resource-health__signal-tile {
    min-height: 96px;
    padding: 12px;
  }

  .ops-resource-health__composition-fallback > div {
    grid-template-columns: minmax(82px, 1fr) minmax(70px, 1.5fr) auto;
  }

  .ops-resource-health__nav {
    top: calc(var(--app-shell-top-offset) + 64px);
  }

  .ops-resource-health__nonadditive-note {
    padding-right: 12px;
    padding-left: 12px;
  }

  .ops-resource-health__quota-note {
    padding-right: 12px;
    padding-left: 12px;
  }

  .ops-resource-health__notes {
    grid-template-columns: minmax(0, 1fr);
  }

  .ops-resource-health__notes > p + p {
    border-top: 1px solid var(--lx-clay-border);
    border-left: 0;
  }

  .ops-resource-health__health-disclosure {
    margin-right: 12px;
    margin-left: 12px;
  }

  .ops-resource-health__drawer {
    width: 100%;
  }

  .ops-resource-health__panel-header {
    align-items: flex-start;
    flex-direction: column;
    gap: 10px;
  }

  .ops-resource-health__panel-header > button {
    width: 100%;
  }

  .ops-resource-health__panel-header p,
  .ops-resource-health__ledger-toolbar p {
    display: none;
  }

  .ops-resource-health__metric {
    padding: 9px 10px;
  }

  .ops-resource-health__metric span {
    min-height: 0;
    font-size: 0.68rem;
  }

  .ops-resource-health__metric strong {
    margin-top: 4px;
    font-size: 1.2rem;
  }

  .ops-resource-health__ledger-toolbar {
    gap: 7px;
  }

  .ops-resource-health__row {
    align-items: stretch;
    flex-direction: column;
    gap: 4px;
  }

  .ops-resource-health__row-action,
  .ops-resource-health__row-actions {
    width: 100%;
  }

  .ops-resource-health__row-actions > button {
    flex: 1 1 0;
  }

  .ops-resource-health__row-actions .ops-resource-health__retest-button {
    display: none;
  }

  .ops-resource-health__ledger-table {
    padding: 8px 12px 12px;
    overflow: visible;
  }

  .ops-resource-health__ledger-table table,
  .ops-resource-health__ledger-table tbody,
  .ops-resource-health__ledger-table tr,
  .ops-resource-health__ledger-table td {
    display: block;
    width: 100%;
  }

  .ops-resource-health__ledger-shell--account .ops-resource-health__ledger-table td {
    padding: 4px 0;
    border-left: 0 !important;
  }

  .ops-resource-health__ledger-shell--account .ops-resource-health__ledger-table td[data-label] {
    display: grid;
    grid-template-columns: minmax(74px, auto) minmax(0, 1fr);
    gap: 10px;
  }

  .ops-resource-health__ledger-shell--account .ops-resource-health__ledger-table td[data-label]::before {
    content: attr(data-label);
    color: var(--lx-clay-text-muted);
    font-size: 0.66rem;
    font-weight: 800;
  }

  .ops-resource-health__ledger-table thead {
    display: none;
  }

  .ops-resource-health__ledger-table tbody tr {
    margin-top: 8px;
    padding: 10px 12px;
    border: 1px solid var(--lx-clay-border);
    border-radius: var(--lx-clay-radius-ops);
    background: var(--lx-clay-surface);
  }

  .ops-resource-health__ledger-table tbody tr[aria-selected='true'] {
    border-color: color-mix(in srgb, var(--lx-clay-accent) 30%, var(--lx-clay-border));
  }

  .ops-resource-health__ledger-table td {
    padding: 4px 0;
    border: 0;
  }

  .ops-resource-health__ledger-table td:last-child {
    padding-top: 8px;
  }

  .ops-resource-health__ledger-table td:last-child .ops-resource-health__row-action {
    width: 100%;
    border-color: var(--lx-clay-border-strong);
    background: var(--lx-clay-surface-soft);
  }

  .ops-resource-health__ledger-preview {
    display: none;
  }

  .ops-resource-health__source-alert {
    align-items: flex-start;
    flex-direction: column;
    gap: 0;
    padding: 10px 12px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .ops-resource-health__row-action,
  .ops-resource-health__text-button,
  .ops-resource-health__secondary-button,
  .ops-resource-health__retest-button {
    transition: none;
  }
}
</style>
