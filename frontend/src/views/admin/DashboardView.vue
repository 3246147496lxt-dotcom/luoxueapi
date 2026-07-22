<template>
  <AppLayout variant="home-clay">
    <div class="admin-dashboard-home" data-testid="dashboard-page" data-admin-page-kind="overview">
      <header class="dashboard-page-heading">
        <div>
          <h1>{{ t('admin.dashboard.title') }}</h1>
          <p>{{ t('admin.dashboard.description') }}</p>
        </div>
        <div class="dashboard-heading-actions">
          <div v-if="lastUpdatedLabel" class="dashboard-updated-at">
            <span>{{ t('admin.dashboard.lastUpdated') }}</span>
            <strong>{{ lastUpdatedLabel }}</strong>
          </div>
          <button
            type="button"
            class="dashboard-primary-button"
            data-testid="dashboard-refresh"
            :disabled="isRefreshing"
            @click="loadDashboardStats"
          >
            <Icon name="refresh" :class="{ 'dashboard-spin': isRefreshing }" />
            {{ t('admin.dashboard.refreshData') }}
          </button>
        </div>
      </header>

      <div v-if="loading" class="dashboard-loading" data-testid="dashboard-loading" aria-busy="true" aria-live="polite">
        <span class="sr-only">{{ t('common.loading') }}</span>
        <div class="dashboard-surface loading-overview">
          <div class="clay-skeleton skeleton-heading"></div>
          <div class="loading-metrics">
            <div v-for="index in 4" :key="index" class="loading-metric">
              <div class="clay-skeleton skeleton-icon"></div>
              <div class="clay-skeleton skeleton-label"></div>
              <div class="clay-skeleton skeleton-value"></div>
              <div class="clay-skeleton skeleton-detail"></div>
            </div>
          </div>
        </div>
        <div class="loading-secondary-grid">
          <div class="dashboard-surface"><div class="clay-skeleton skeleton-panel"></div></div>
          <div class="dashboard-surface"><div class="clay-skeleton skeleton-panel"></div></div>
        </div>
      </div>

      <section
        v-else-if="statsLoadError && !stats"
        class="dashboard-surface dashboard-error-state"
        data-testid="dashboard-error"
        aria-live="assertive"
      >
        <span class="error-state-icon"><Icon name="exclamationTriangle" size="xl" /></span>
        <h2>{{ t('admin.dashboard.loadErrorTitle') }}</h2>
        <p>{{ t('admin.dashboard.loadErrorDescription') }}</p>
        <button type="button" class="dashboard-primary-button" @click="loadDashboardStats">
          <Icon name="refresh" />
          {{ t('admin.dashboard.retry') }}
        </button>
      </section>

      <template v-else-if="stats">
        <section class="dashboard-surface overview-surface" data-testid="dashboard-overview" :aria-label="t('admin.dashboard.keyMetrics')">
          <div class="dashboard-section-heading">
            <h2>{{ t('admin.dashboard.keyMetrics') }}</h2>
            <span class="dashboard-health" :class="{ stale: stats.stats_stale }">
              <i></i>{{ stats.stats_stale ? t('admin.dashboard.dataStale') : t('admin.dashboard.dataFresh') }}
            </span>
          </div>
          <div class="overview-metrics">
            <article class="overview-metric api-key-overview-metric" data-testid="api-key-overview-metric">
              <div class="overview-metric-topline">
                <span class="metric-icon"><Icon name="key" :stroke-width="1.75" /></span>
                <span
                  class="overview-metric-comparison"
                  :class="apiKeyWeekComparison.tone"
                  :title="apiKeyWeekComparison.accessibleLabel"
                  :aria-label="apiKeyWeekComparison.accessibleLabel"
                  role="group"
                  data-testid="api-key-week-comparison"
                >
                  <strong>{{ apiKeyWeekComparison.value }}</strong>
                  <small>{{ apiKeyWeekComparison.label }}</small>
                </span>
              </div>
              <p>{{ t('admin.dashboard.apiKeys') }}</p>
              <strong>{{ formatNumber(stats.total_api_keys) }}</strong>
              <span class="api-key-activity-summary">{{ apiKeyActivitySummary }}</span>
            </article>
            <article class="overview-metric account-overview-metric" data-testid="account-overview-metric">
              <div class="overview-metric-topline">
                <span class="metric-icon"><Icon name="server" :stroke-width="1.75" /></span>
                <span
                  class="overview-metric-comparison"
                  :class="accountHealthComparison.tone"
                  :title="accountHealthComparison.accessibleLabel"
                  :aria-label="accountHealthComparison.accessibleLabel"
                  role="group"
                  data-testid="account-health-comparison"
                >
                  <strong>{{ accountHealthComparison.value }}</strong>
                  <small>{{ accountHealthComparison.label }}</small>
                </span>
              </div>
              <p>{{ t('admin.dashboard.accounts') }}</p>
              <strong>{{ formatNumber(stats.total_accounts) }}</strong>
              <span class="account-status-summary">
                {{ t('admin.dashboard.accountStatusSummary', {
                  healthy: formatNumber(stats.healthy_accounts),
                  error: formatNumber(stats.error_accounts)
                }) }}
              </span>
            </article>
            <article class="overview-metric request-overview-metric" data-testid="request-overview-metric">
              <div class="overview-metric-topline">
                <span class="metric-icon"><Icon name="arrowLeftRight" :stroke-width="1.75" /></span>
                <span
                  class="overview-metric-comparison"
                  :class="requestDayComparison.tone"
                  :title="requestDayComparison.accessibleLabel"
                  :aria-label="requestDayComparison.accessibleLabel"
                  role="group"
                  data-testid="request-day-comparison"
                >
                  <strong>{{ requestDayComparison.value }}</strong>
                  <small>{{ requestDayComparison.label }}</small>
                </span>
              </div>
              <p>{{ t('admin.dashboard.todayRequests') }}</p>
              <strong>{{ formatNumber(stats.today_requests) }}</strong>
              <span>{{ t('common.total') }}: {{ formatNumber(stats.total_requests) }}</span>
            </article>
            <article class="overview-metric new-user-overview-metric" data-testid="new-user-overview-metric">
              <div class="overview-metric-topline">
                <span class="metric-icon"><Icon name="userPlus" :stroke-width="1.75" /></span>
                <span
                  class="overview-metric-comparison"
                  :class="newUserDayComparison.tone"
                  :title="newUserDayComparison.accessibleLabel"
                  :aria-label="newUserDayComparison.accessibleLabel"
                  role="group"
                  data-testid="new-user-day-comparison"
                >
                  <strong>{{ newUserDayComparison.value }}</strong>
                  <small>{{ newUserDayComparison.label }}</small>
                </span>
              </div>
              <p>{{ t('admin.dashboard.newUsersToday') }}</p>
              <strong>{{ formatNumber(stats.today_new_users) }}</strong>
              <span>{{ t('admin.dashboard.totalUsersSummary', { count: formatNumber(stats.total_users) }) }}</span>
            </article>
          </div>
        </section>

        <div class="dashboard-operational-grid">
          <section class="dashboard-surface performance-surface">
            <div class="dashboard-section-heading">
              <h2>{{ t('admin.dashboard.performance') }}</h2>
              <span class="dashboard-sync-state" :class="{ stale: stats.stats_stale }">
                <i></i>{{ stats.stats_stale ? t('admin.dashboard.dataStale') : t('admin.dashboard.dataFresh') }}
              </span>
            </div>
            <div class="performance-metrics">
              <article>
                <span class="performance-icon"><Icon name="coins" /></span>
                <div>
                  <p>{{ t('admin.dashboard.todayTokens') }}</p>
                  <strong>{{ formatTokens(stats.today_tokens) }}</strong>
                  <span class="cost-breakdown">
                    <span :title="t('admin.dashboard.actual')">${{ formatCost(stats.today_actual_cost) }}</span>
                    <span :title="t('admin.dashboard.accountCost')">${{ formatCost(stats.today_account_cost) }}</span>
                    <span :title="t('admin.dashboard.standard')">${{ formatCost(stats.today_cost) }}</span>
                  </span>
                </div>
              </article>
              <article>
                <span class="performance-icon"><Icon name="database" /></span>
                <div>
                  <p>{{ t('admin.dashboard.totalTokens') }}</p>
                  <strong>{{ formatTokens(stats.total_tokens) }}</strong>
                  <span class="cost-breakdown">
                    <span :title="t('admin.dashboard.actual')">${{ formatCost(stats.total_actual_cost) }}</span>
                    <span :title="t('admin.dashboard.accountCost')">${{ formatCost(stats.total_account_cost) }}</span>
                    <span :title="t('admin.dashboard.standard')">${{ formatCost(stats.total_cost) }}</span>
                  </span>
                </div>
              </article>
              <article>
                <span class="performance-icon"><Icon name="gauge" /></span>
                <div>
                  <p>{{ t('admin.dashboard.performance') }}</p>
                  <strong>{{ formatTokens(stats.rpm) }} RPM</strong>
                  <span>{{ formatTokens(stats.tpm) }} TPM</span>
                </div>
              </article>
              <article>
                <span class="performance-icon"><Icon name="timer" /></span>
                <div>
                  <p>{{ t('admin.dashboard.avgResponse') }}</p>
                  <strong>{{ formatDuration(stats.average_duration_ms) }}</strong>
                  <span>{{ formatNumber(stats.active_users) }} {{ t('admin.dashboard.activeUsers') }}</span>
                </div>
              </article>
            </div>
          </section>

          <section class="dashboard-surface quick-actions-surface">
            <div class="dashboard-section-heading"><h2>{{ t('admin.dashboard.quickActions') }}</h2></div>
            <div class="quick-actions-list">
              <button v-if="canUseBatchImage" type="button" @click="router.push('/batch-image')">
                <span class="quick-action-icon"><Icon name="photo" /></span>
                <span><strong>{{ t('admin.dashboard.batchImage') }}</strong><small>{{ t('admin.dashboard.batchImageDesc') }}</small></span>
                <Icon name="chevronRight" size="sm" />
              </button>
              <button type="button" @click="router.push('/admin/groups')">
                <span class="quick-action-icon secondary"><Icon name="calculator" /></span>
                <span><strong>{{ t('admin.dashboard.groupPricing') }}</strong><small>{{ t('admin.dashboard.groupPricingDesc') }}</small></span>
                <Icon name="chevronRight" size="sm" />
              </button>
            </div>
          </section>
        </div>

        <section class="dashboard-filter-bar" data-testid="dashboard-date-filter" :aria-label="t('admin.dashboard.timeRange')">
          <div class="dashboard-filter-field">
            <span>{{ t('admin.dashboard.timeRange') }}</span>
            <DateRangePicker
              v-model:start-date="startDate"
              v-model:end-date="endDate"
              @change="onDateRangeChange"
            />
          </div>
          <button
            type="button"
            class="dashboard-filter-refresh"
            :disabled="isRefreshing"
            :aria-label="t('common.refresh')"
            :title="t('common.refresh')"
            @click="loadDashboardStats"
          >
            <Icon name="refresh" :class="{ 'dashboard-spin': isRefreshing }" />
          </button>
          <div class="dashboard-filter-field granularity-field" data-testid="dashboard-granularity">
            <span>{{ t('admin.dashboard.granularity') }}</span>
            <div class="granularity-select">
              <Select v-model="granularity" :options="granularityOptions" @change="loadChartData" />
            </div>
          </div>
        </section>

        <section class="dashboard-charts-grid">
          <div class="dashboard-chart-panel model-chart-panel">
            <ModelDistributionChart
              variant="home-clay"
              :model-stats="modelStats"
              :loading="chartsLoading"
              :error="chartsError ? t('admin.dashboard.failedToLoad') : null"
              :start-date="startDate"
              :end-date="endDate"
              @retry="loadChartData"
            />
          </div>
          <div class="dashboard-chart-panel token-chart-panel">
            <TokenUsageTrend
              variant="home-clay"
              :trend-data="trendData"
              :loading="chartsLoading"
              :error="chartsError ? t('admin.dashboard.failedToLoad') : null"
              @retry="loadChartData"
            />
          </div>
        </section>

        <DashboardTopUsers
          :items="rankingItems"
          :trend="userTrend"
          :loading="rankingLoading || userTrendLoading"
          :error="rankingError"
          :granularity="granularity"
          @select="goToUserUsage"
          @retry="loadTopUsers"
          @view-all="goToUserRanking"
        />
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'

const { t, locale } = useI18n()
import { adminAPI } from '@/api/admin'
import type {
  DashboardStats,
  TrendDataPoint,
  ModelStat,
  UserUsageTrendPoint,
  UserSpendingRankingItem
} from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import Select from '@/components/common/Select.vue'
import ModelDistributionChart from '@/components/charts/ModelDistributionChart.vue'
import TokenUsageTrend from '@/components/charts/TokenUsageTrend.vue'
import DashboardTopUsers from '@/components/admin/dashboard/DashboardTopUsers.vue'
import { useBatchImageAccess } from '@/composables/useBatchImageAccess'

const appStore = useAppStore()
const router = useRouter()
const { canUseBatchImage, refreshBatchImageAccess } = useBatchImageAccess()
const stats = ref<DashboardStats | null>(null)
const loading = ref(false)
const chartsLoading = ref(false)
const chartsError = ref(false)
const userTrendLoading = ref(false)
const rankingLoading = ref(false)
const rankingError = ref(false)
const statsLoadError = ref(false)

// Chart data
const trendData = ref<TrendDataPoint[]>([])
const modelStats = ref<ModelStat[]>([])
const userTrend = ref<UserUsageTrendPoint[]>([])
const rankingItems = ref<UserSpendingRankingItem[]>([])
const isRefreshing = computed(() => (
  loading.value || chartsLoading.value || userTrendLoading.value || rankingLoading.value
))
let chartLoadSeq = 0
let usersTrendLoadSeq = 0
let rankingLoadSeq = 0
const rankingLimit = 12

// Helper function to format date in local timezone
const formatLocalDate = (date: Date): string => {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

const getLast24HoursRangeDates = (): { start: string; end: string } => {
  const end = new Date()
  const start = new Date(end.getTime() - 24 * 60 * 60 * 1000)
  return {
    start: formatLocalDate(start),
    end: formatLocalDate(end)
  }
}

// Date range
const granularity = ref<'day' | 'hour'>('hour')
const defaultRange = getLast24HoursRangeDates()
const startDate = ref(defaultRange.start)
const endDate = ref(defaultRange.end)

// Granularity options for Select component
const granularityOptions = computed(() => [
  { value: 'day', label: t('admin.dashboard.day') },
  { value: 'hour', label: t('admin.dashboard.hour') }
])

const lastUpdatedLabel = computed(() => {
  const value = stats.value?.stats_updated_at?.trim()
  if (!value) return ''

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''

  return new Intl.DateTimeFormat(locale.value, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  }).format(date)
})

const toFiniteNumber = (value: unknown): number => {
  const numberValue = Number(value)
  return Number.isFinite(numberValue) ? numberValue : 0
}

const formatNumber = (value: number | null | undefined): string => {
  return toFiniteNumber(value).toLocaleString()
}

type MetricComparisonTone = 'positive' | 'negative' | 'neutral' | 'first'

const toOptionalCount = (value: unknown): number | null => {
  if (value === null || value === undefined || value === '') return null
  const count = Number(value)
  return Number.isFinite(count) ? Math.max(0, Math.trunc(count)) : null
}

const isParsableStatsTimestamp = (value: unknown): value is string => {
  if (typeof value !== 'string' || !value.trim()) return false
  return !Number.isNaN(new Date(value).getTime())
}

const apiKeyWeeklyData = computed(() => {
  const current = toOptionalCount(stats.value?.current_week_active_api_keys)
  const previous = toOptionalCount(stats.value?.previous_week_same_period_active_api_keys)
  const windows = [
    stats.value?.current_week_start_at,
    stats.value?.current_week_end_at,
    stats.value?.previous_week_same_period_start_at,
    stats.value?.previous_week_same_period_end_at
  ]

  return {
    available: current !== null && previous !== null && windows.every(isParsableStatsTimestamp),
    current,
    previous
  }
})

const apiKeyActivitySummary = computed(() => {
  const enabled = formatNumber(stats.value?.active_api_keys)
  const weekly = apiKeyWeeklyData.value.current
  return !apiKeyWeeklyData.value.available || weekly === null
    ? t('admin.dashboard.apiKeyWeeklyActivityUnavailableSummary', { enabled })
    : t('admin.dashboard.apiKeyWeeklyActivitySummary', {
        enabled,
        weekly: formatNumber(weekly)
      })
})

const resolveStatsTimezone = (value: string | null | undefined): string => {
  const timezone = value?.trim() || 'UTC'
  try {
    new Intl.DateTimeFormat('en-US', { timeZone: timezone }).format(new Date(0))
    return timezone
  } catch {
    return 'UTC'
  }
}

const formatStatsTimestamp = (value: string | null | undefined, timezone: string): string => {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '—'

  return new Intl.DateTimeFormat(locale.value, {
    timeZone: timezone,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hourCycle: 'h23'
  }).format(date)
}

const formatStatsWindow = (
  start: string | null | undefined,
  end: string | null | undefined,
  timezone: string
): string => `${formatStatsTimestamp(start, timezone)} – ${formatStatsTimestamp(end, timezone)}`

const apiKeyWeekComparison = computed<{
  value: string
  label: string
  tone: MetricComparisonTone
  accessibleLabel: string
}>(() => {
  const current = apiKeyWeeklyData.value.available ? apiKeyWeeklyData.value.current : null
  const previous = apiKeyWeeklyData.value.available ? apiKeyWeeklyData.value.previous : null
  let value = '—'
  let label = t('admin.dashboard.apiKeyNoWeeklyComparison')
  let tone: MetricComparisonTone = 'neutral'

  if (current !== null && previous !== null && previous > 0) {
    const percentage = ((current - previous) / previous) * 100
    value = `${percentage > 0 ? '+' : ''}${percentage.toFixed(1)}%`
    label = t('admin.dashboard.apiKeyComparedToPreviousWeek')
    tone = percentage > 0 ? 'positive' : percentage < 0 ? 'negative' : 'neutral'
  } else if (current !== null && previous === 0 && current > 0) {
    value = t('admin.dashboard.apiKeyFirstActive')
    label = t('admin.dashboard.apiKeyPreviousWeekUnavailable')
    tone = 'first'
  }

  const timezone = resolveStatsTimezone(stats.value?.stats_timezone)
  const timezoneLabel = timezone === 'Asia/Shanghai'
    ? t('admin.dashboard.apiKeyBeijingTime')
    : timezone
  const currentWindow = formatStatsWindow(
    stats.value?.current_week_start_at,
    stats.value?.current_week_end_at,
    timezone
  )
  const previousWindow = formatStatsWindow(
    stats.value?.previous_week_same_period_start_at,
    stats.value?.previous_week_same_period_end_at,
    timezone
  )

  return {
    value,
    label,
    tone,
    accessibleLabel: t('admin.dashboard.apiKeyWeekComparisonAria', {
      current: current === null ? '—' : formatNumber(current),
      previous: previous === null ? '—' : formatNumber(previous),
      currentWindow,
      previousWindow,
      result: value,
      comparisonLabel: label,
      timezone: timezoneLabel
    })
  }
})

const accountHealthComparison = computed<{
  value: string
  label: string
  tone: 'positive' | 'warning' | 'negative' | 'neutral'
  accessibleLabel: string
}>(() => {
  const total = toOptionalCount(stats.value?.total_accounts) ?? 0
  const healthy = toOptionalCount(stats.value?.healthy_accounts) ?? 0
  const error = toOptionalCount(stats.value?.error_accounts) ?? 0
  const ratelimit = toOptionalCount(stats.value?.ratelimit_accounts) ?? 0
  const overload = toOptionalCount(stats.value?.overload_accounts) ?? 0
  const rate = total > 0 ? (healthy / total) * 100 : null
  const value = rate === null ? '—' : `${rate.toFixed(1)}%`
  const label = total > 0
    ? t('admin.dashboard.accountHealthRate')
    : t('admin.dashboard.accountNoAccounts')
  const tone = rate === null
    ? 'neutral'
    : rate >= 90
      ? 'positive'
      : rate >= 70
        ? 'warning'
        : 'negative'

  return {
    value,
    label,
    tone,
    accessibleLabel: t('admin.dashboard.accountHealthAria', {
      healthy: formatNumber(healthy),
      total: formatNumber(total),
      rate: value,
      error: formatNumber(error),
      ratelimit: formatNumber(ratelimit),
      overload: formatNumber(overload)
    })
  }
})

const requestDayData = computed(() => {
  const current = toOptionalCount(stats.value?.current_day_requests)
  const previous = toOptionalCount(stats.value?.previous_day_same_period_requests)
  const windows = [
    stats.value?.current_day_start_at,
    stats.value?.current_day_end_at,
    stats.value?.previous_day_same_period_start_at,
    stats.value?.previous_day_same_period_end_at
  ]

  return {
    available: current !== null && previous !== null && windows.every(isParsableStatsTimestamp),
    current,
    previous
  }
})

const requestDayComparison = computed<{
  value: string
  label: string
  tone: MetricComparisonTone
  accessibleLabel: string
}>(() => {
  const current = requestDayData.value.available ? requestDayData.value.current : null
  const previous = requestDayData.value.available ? requestDayData.value.previous : null
  let value = '—'
  let label = requestDayData.value.available
    ? t('admin.dashboard.requestNoComparison')
    : t('admin.dashboard.requestPreviousDayUnavailable')
  let tone: MetricComparisonTone = 'neutral'

  if (current !== null && previous !== null && previous > 0) {
    const percentage = ((current - previous) / previous) * 100
    value = `${percentage > 0 ? '+' : ''}${percentage.toFixed(1)}%`
    label = t('admin.dashboard.requestComparedToPreviousDay')
    tone = percentage > 0 ? 'positive' : percentage < 0 ? 'negative' : 'neutral'
  } else if (current !== null && previous === 0 && current > 0) {
    value = t('admin.dashboard.requestFirst')
    label = t('admin.dashboard.requestPreviousDayNone')
    tone = 'first'
  }

  const timezone = resolveStatsTimezone(stats.value?.stats_timezone)
  const timezoneLabel = timezone === 'Asia/Shanghai'
    ? t('admin.dashboard.apiKeyBeijingTime')
    : timezone
  const currentWindow = formatStatsWindow(
    stats.value?.current_day_start_at,
    stats.value?.current_day_end_at,
    timezone
  )
  const previousWindow = formatStatsWindow(
    stats.value?.previous_day_same_period_start_at,
    stats.value?.previous_day_same_period_end_at,
    timezone
  )

  return {
    value,
    label,
    tone,
    accessibleLabel: t('admin.dashboard.requestDayComparisonAria', {
      current: current === null ? '—' : formatNumber(current),
      previous: previous === null ? '—' : formatNumber(previous),
      currentWindow,
      previousWindow,
      result: value,
      comparisonLabel: label,
      timezone: timezoneLabel
    })
  }
})

const newUserDayData = computed(() => {
  const current = toOptionalCount(stats.value?.current_day_new_users)
  const previous = toOptionalCount(stats.value?.previous_day_same_period_new_users)
  const windows = [
    stats.value?.current_day_start_at,
    stats.value?.current_day_end_at,
    stats.value?.previous_day_same_period_start_at,
    stats.value?.previous_day_same_period_end_at
  ]

  return {
    available: current !== null && previous !== null && windows.every(isParsableStatsTimestamp),
    current,
    previous
  }
})

const newUserDayComparison = computed<{
  value: string
  label: string
  tone: MetricComparisonTone
  accessibleLabel: string
}>(() => {
  const current = newUserDayData.value.available ? newUserDayData.value.current : null
  const previous = newUserDayData.value.available ? newUserDayData.value.previous : null
  let value = '—'
  let label = newUserDayData.value.available
    ? t('admin.dashboard.newUserNoComparison')
    : t('admin.dashboard.newUserPreviousDayUnavailable')
  let tone: MetricComparisonTone = 'neutral'

  if (current !== null && previous !== null && previous > 0) {
    const percentage = ((current - previous) / previous) * 100
    value = `${percentage > 0 ? '+' : ''}${percentage.toFixed(1)}%`
    label = t('admin.dashboard.newUserComparedToPreviousDay')
    tone = percentage > 0 ? 'positive' : percentage < 0 ? 'negative' : 'neutral'
  } else if (current !== null && previous === 0 && current > 0) {
    value = t('admin.dashboard.newUserFirst')
    label = t('admin.dashboard.newUserPreviousDayNone')
    tone = 'first'
  }

  const timezone = resolveStatsTimezone(stats.value?.stats_timezone)
  const timezoneLabel = timezone === 'Asia/Shanghai'
    ? t('admin.dashboard.apiKeyBeijingTime')
    : timezone
  const currentWindow = formatStatsWindow(
    stats.value?.current_day_start_at,
    stats.value?.current_day_end_at,
    timezone
  )
  const previousWindow = formatStatsWindow(
    stats.value?.previous_day_same_period_start_at,
    stats.value?.previous_day_same_period_end_at,
    timezone
  )

  return {
    value,
    label,
    tone,
    accessibleLabel: t('admin.dashboard.newUserDayComparisonAria', {
      current: current === null ? '—' : formatNumber(current),
      previous: previous === null ? '—' : formatNumber(previous),
      currentWindow,
      previousWindow,
      result: value,
      comparisonLabel: label,
      timezone: timezoneLabel
    })
  }
})

const formatCost = (value: number | null | undefined): string => {
  const safeValue = toFiniteNumber(value)
  if (safeValue >= 1000) {
    return (safeValue / 1000).toFixed(2) + 'K'
  } else if (safeValue >= 1) {
    return safeValue.toFixed(2)
  } else if (safeValue >= 0.01) {
    return safeValue.toFixed(3)
  }
  return safeValue.toFixed(4)
}

const formatDuration = (ms: number): string => {
  if (ms >= 1000) {
    return `${(ms / 1000).toFixed(2)}s`
  }
  return `${Math.round(ms)}ms`
}

const goToUserUsage = (item: UserSpendingRankingItem) => {
  void router.push({
    path: '/admin/usage',
    query: {
      user_id: String(item.user_id),
      start_date: startDate.value,
      end_date: endDate.value
    }
  })
}

const goToUserRanking = () => {
  void router.push({
    path: '/admin/usage',
    query: {
      tab: 'ranking',
      sort_by: 'actual_cost',
      start_date: startDate.value,
      end_date: endDate.value
    }
  })
}

const formatTokens = (value: number | null | undefined): string => {
  const safeValue = Number.isFinite(Number(value)) ? Number(value) : 0
  if (safeValue >= 1_000_000_000) return `${(safeValue / 1_000_000_000).toFixed(2)}B`
  if (safeValue >= 1_000_000) return `${(safeValue / 1_000_000).toFixed(2)}M`
  if (safeValue >= 1_000) return `${(safeValue / 1_000).toFixed(2)}K`
  return safeValue.toLocaleString()
}

// Date range change handler
const onDateRangeChange = (range: {
  startDate: string
  endDate: string
  preset: string | null
}) => {
  // Auto-select granularity based on date range
  const start = new Date(range.startDate)
  const end = new Date(range.endDate)
  const daysDiff = Math.ceil((end.getTime() - start.getTime()) / (1000 * 60 * 60 * 24))

  // If range is 1 day, use hourly granularity
  if (daysDiff <= 1) {
    granularity.value = 'hour'
  } else {
    granularity.value = 'day'
  }

  loadChartData()
}

// Load data
const loadDashboardSnapshot = async (includeStats: boolean) => {
  const currentSeq = ++chartLoadSeq
  if (includeStats && !stats.value) {
    loading.value = true
    statsLoadError.value = false
  }
  chartsLoading.value = true
  chartsError.value = false
  try {
    const response = await adminAPI.dashboard.getSnapshotV2({
      start_date: startDate.value,
      end_date: endDate.value,
      granularity: granularity.value,
      include_stats: includeStats,
      include_trend: true,
      include_model_stats: true,
      include_group_stats: false,
      include_users_trend: false
    })
    if (currentSeq !== chartLoadSeq) return
    if (includeStats) {
      if (!response.stats && !stats.value) {
        throw new Error('Dashboard snapshot did not include stats')
      }
      if (response.stats) {
        stats.value = response.stats
        statsLoadError.value = false
      }
    }
    trendData.value = response.trend || []
    modelStats.value = response.models || []
    chartsError.value = false
  } catch (error) {
    if (currentSeq !== chartLoadSeq) return
    if (includeStats && !stats.value) {
      statsLoadError.value = true
    }
    chartsError.value = true
    appStore.showError(t('admin.dashboard.failedToLoad'))
    console.error('Error loading dashboard snapshot:', error)
  } finally {
    if (currentSeq === chartLoadSeq) {
      loading.value = false
      chartsLoading.value = false
    }
  }
}

const loadUsersTrend = async () => {
  const currentSeq = ++usersTrendLoadSeq
  userTrendLoading.value = true
  try {
    const response = await adminAPI.dashboard.getUserUsageTrend({
      start_date: startDate.value,
      end_date: endDate.value,
      granularity: granularity.value,
      limit: 12
    })
    if (currentSeq !== usersTrendLoadSeq) return
    userTrend.value = response.trend || []
  } catch (error) {
    if (currentSeq !== usersTrendLoadSeq) return
    console.error('Error loading users trend:', error)
    userTrend.value = []
  } finally {
    if (currentSeq === usersTrendLoadSeq) {
      userTrendLoading.value = false
    }
  }
}

const loadUserSpendingRanking = async () => {
  const currentSeq = ++rankingLoadSeq
  rankingLoading.value = true
  rankingError.value = false
  try {
    const response = await adminAPI.dashboard.getUserSpendingRanking({
      start_date: startDate.value,
      end_date: endDate.value,
      limit: rankingLimit
    })
    if (currentSeq !== rankingLoadSeq) return
    rankingItems.value = response.ranking || []
  } catch (error) {
    if (currentSeq !== rankingLoadSeq) return
    console.error('Error loading user spending ranking:', error)
    rankingItems.value = []
    rankingError.value = true
  } finally {
    if (currentSeq === rankingLoadSeq) {
      rankingLoading.value = false
    }
  }
}

const loadTopUsers = async () => {
  await Promise.all([loadUsersTrend(), loadUserSpendingRanking()])
}

const loadDashboardStats = async () => {
  await Promise.all([
    loadDashboardSnapshot(true),
    loadUsersTrend(),
    loadUserSpendingRanking()
  ])
}

const loadChartData = async () => {
  await Promise.all([
    loadDashboardSnapshot(false),
    loadUsersTrend(),
    loadUserSpendingRanking()
  ])
}

onMounted(() => {
  void refreshBatchImageAccess()
  void loadDashboardStats()
})
</script>

<style scoped src="./DashboardView.clay.css"></style>
