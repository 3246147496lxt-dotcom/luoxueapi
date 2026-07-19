<template>
  <AppLayout>
    <div class="yunwu-dashboard space-y-4 pt-[4.75rem] sm:pt-0">
      <header class="flex items-center justify-between gap-4 py-1">
        <h1 class="min-w-0 truncate text-2xl font-semibold leading-8 text-gray-800 dark:text-gray-100">
          {{ t('dashboard.greeting.line', { period: greetingPeriod, name: dashboardUserName }) }}
        </h1>

        <div class="flex shrink-0 items-center gap-2">
          <button
            type="button"
            class="dashboard-icon-button"
            :aria-label="t('dashboard.openFilters')"
            :title="t('dashboard.openFilters')"
            @click="openFilters"
          >
            <Icon name="search" size="sm" :stroke-width="1.8" />
          </button>
          <button
            type="button"
            class="dashboard-icon-button"
            :disabled="isRefreshing"
            :aria-label="t('dashboard.refreshDashboard')"
            :title="t('dashboard.refreshDashboard')"
            @click="refreshAll"
          >
            <Icon name="refresh" size="sm" :stroke-width="1.8" :class="{ 'animate-spin': isRefreshing }" />
          </button>
        </div>
      </header>

      <div v-if="loading && !stats" class="space-y-4" aria-live="polite">
        <div class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
          <div v-for="index in 4" :key="index" class="dashboard-panel min-h-[205px] p-5">
            <div class="skeleton h-5 w-28"></div>
            <div class="mt-8 space-y-5">
              <div class="skeleton h-12 w-full"></div>
              <div class="skeleton h-12 w-full"></div>
            </div>
          </div>
        </div>
        <div class="grid gap-4 xl:grid-cols-[minmax(0,3fr)_minmax(17.5rem,1fr)]">
          <div class="dashboard-panel min-h-[550px] p-6">
            <div class="skeleton h-5 w-36"></div>
            <div class="skeleton mt-8 h-[410px] w-full"></div>
          </div>
          <div class="dashboard-panel min-h-[550px] p-6">
            <div class="skeleton h-5 w-28"></div>
            <div class="skeleton mt-8 h-24 w-full"></div>
          </div>
        </div>
      </div>

      <div v-else-if="!stats" class="dashboard-panel flex min-h-[360px] flex-col items-center justify-center px-6 text-center">
        <Icon name="exclamationCircle" size="xl" class="text-amber-500" />
        <h2 class="mt-4 text-base font-semibold text-gray-900 dark:text-white">
          {{ t('dashboard.loadErrorTitle') }}
        </h2>
        <p class="mt-2 max-w-md text-sm leading-6 text-gray-500 dark:text-dark-400">
          {{ t('dashboard.loadErrorDescription') }}
        </p>
        <button type="button" class="btn btn-primary mt-5" @click="refreshAll">
          <Icon name="refresh" size="sm" />
          {{ t('dashboard.retry') }}
        </button>
      </div>

      <template v-else>
        <UserDashboardStats
          :stats="stats"
          :balance="user?.balance || 0"
          :is-simple="authStore.isSimpleMode"
          :range-metrics="rangeMetrics"
          :trend="trendData"
          :start-date="startDate"
          :end-date="endDate"
          :granularity="granularity"
        />

        <div class="grid items-stretch gap-4 xl:grid-cols-[minmax(0,3fr)_minmax(17.5rem,1fr)]">
          <UserDashboardCharts
            :loading="loadingCharts"
            :start-date="startDate"
            :end-date="endDate"
            :granularity="granularity"
            :trend="trendData"
            :models="modelStats"
          />
          <UserDashboardApiInfo
            :api-base-url="apiBaseUrl"
            :custom-endpoints="customEndpoints"
          />
        </div>

        <UserDashboardSupportPanels
          :announcements="announcementStore.announcements"
          :monitors="monitors"
          :loading-announcements="announcementStore.loading"
          :loading-monitors="loadingMonitors"
          :monitor-enabled="monitorEnabled"
        />
      </template>

      <footer class="pb-2 pt-4 text-center text-xs text-gray-400 dark:text-dark-500">
        © {{ currentYear }} {{ appStore.siteName }} · {{ t('dashboard.footer') }}
      </footer>
    </div>

    <BaseDialog
      :show="showFilters"
      :title="t('dashboard.filterTitle')"
      width="narrow"
      @close="showFilters = false"
    >
      <div class="space-y-4">
        <label class="block">
          <span class="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-200">
            {{ t('dashboard.filterStart') }}
          </span>
          <input
            v-model="draftStartDate"
            type="date"
            class="input w-full"
            :max="draftEndDate"
          />
        </label>
        <label class="block">
          <span class="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-200">
            {{ t('dashboard.filterEnd') }}
          </span>
          <input
            v-model="draftEndDate"
            type="date"
            class="input w-full"
            :min="draftStartDate"
            :max="today"
          />
        </label>
        <label class="block">
          <span class="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-200">
            {{ t('dashboard.filterGranularity') }}
          </span>
          <select v-model="draftGranularity" class="input w-full">
            <option value="day">{{ t('dashboard.day') }}</option>
            <option value="hour">{{ t('dashboard.hour') }}</option>
          </select>
        </label>
      </div>

      <template #footer>
        <button type="button" class="btn btn-secondary" @click="showFilters = false">
          {{ t('common.cancel') }}
        </button>
        <button type="button" class="btn btn-primary" :disabled="!canApplyFilters" @click="applyFilters">
          {{ t('common.confirm') }}
        </button>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { useAnnouncementStore } from '@/stores/announcements'
import { usageAPI, type UserDashboardStats as UserStatsType } from '@/api/usage'
import { channelMonitorUserAPI, type UserMonitorView } from '@/api/channelMonitor'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import UserDashboardStats from '@/components/user/dashboard/UserDashboardStats.vue'
import UserDashboardCharts from '@/components/user/dashboard/UserDashboardCharts.vue'
import UserDashboardApiInfo from '@/components/user/dashboard/UserDashboardApiInfo.vue'
import UserDashboardSupportPanels from '@/components/user/dashboard/UserDashboardSupportPanels.vue'
import type { CustomEndpoint, ModelStat, TrendDataPoint } from '@/types'
import { formatDateLocalInput } from '@/utils/format'
import { aggregateDashboardRange } from '@/utils/dashboardMetrics'

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()
const announcementStore = useAnnouncementStore()
const user = computed(() => authStore.user)

const stats = ref<UserStatsType | null>(null)
const loading = ref(true)
const loadingCharts = ref(false)
const loadingMonitors = ref(false)
const trendData = ref<TrendDataPoint[]>([])
const modelStats = ref<ModelStat[]>([])
const monitors = ref<UserMonitorView[]>([])
const currentYear = new Date().getFullYear()
const today = formatDateLocalInput(new Date())

const startDate = ref(formatDateLocalInput(new Date(Date.now() - 6 * 86_400_000)))
const endDate = ref(today)
const granularity = ref<'day' | 'hour'>('day')
const showFilters = ref(false)
const draftStartDate = ref(startDate.value)
const draftEndDate = ref(endDate.value)
const draftGranularity = ref<'day' | 'hour'>(granularity.value)
let monitorController: AbortController | null = null
let chartsRequestId = 0

const greetingPeriod = computed(() => {
  const hour = new Date().getHours()
  if (hour < 12) return t('dashboard.greeting.morning')
  if (hour < 18) return t('dashboard.greeting.afternoon')
  return t('dashboard.greeting.evening')
})

const dashboardUserName = computed(() => (
  user.value?.username
  || user.value?.email?.split('@')[0]
  || appStore.siteName
))

const rangeMetrics = computed(() => aggregateDashboardRange(
  trendData.value,
  startDate.value,
  endDate.value,
))

const apiBaseUrl = computed(() => (
  appStore.cachedPublicSettings?.api_base_url
  || appStore.apiBaseUrl
  || (typeof window === 'undefined' ? '' : window.location.origin)
))
const customEndpoints = computed<CustomEndpoint[]>(() => appStore.cachedPublicSettings?.custom_endpoints || [])
const monitorEnabled = computed(() => appStore.cachedPublicSettings?.channel_monitor_enabled !== false)
const isRefreshing = computed(() => loading.value || loadingCharts.value || loadingMonitors.value || announcementStore.loading)
const canApplyFilters = computed(() => Boolean(
  draftStartDate.value
  && draftEndDate.value
  && draftStartDate.value <= draftEndDate.value
  && draftEndDate.value <= today,
))

async function loadStats(): Promise<void> {
  loading.value = true
  try {
    await authStore.refreshUser()
    stats.value = await usageAPI.getDashboardStats()
  } catch (error) {
    console.error('Failed to load dashboard stats:', error)
  } finally {
    loading.value = false
  }
}

async function loadCharts(): Promise<void> {
  const requestId = ++chartsRequestId
  loadingCharts.value = true
  try {
    const [trendResult, modelsResult] = await Promise.all([
      usageAPI.getDashboardTrend({
        start_date: startDate.value,
        end_date: endDate.value,
        granularity: granularity.value,
      }),
      usageAPI.getDashboardModels({
        start_date: startDate.value,
        end_date: endDate.value,
      }),
    ])
    if (requestId !== chartsRequestId) return
    trendData.value = trendResult.trend || []
    modelStats.value = modelsResult.models || []
  } catch (error) {
    if (requestId === chartsRequestId) {
      console.error('Failed to load dashboard charts:', error)
    }
  } finally {
    if (requestId === chartsRequestId) loadingCharts.value = false
  }
}

async function loadMonitors(): Promise<void> {
  if (!monitorEnabled.value) {
    monitors.value = []
    return
  }

  monitorController?.abort()
  monitorController = new AbortController()
  loadingMonitors.value = true
  try {
    const response = await channelMonitorUserAPI.list({ signal: monitorController.signal })
    monitors.value = response.items || []
  } catch (error) {
    if (!(error instanceof DOMException && error.name === 'AbortError')) {
      console.warn('Failed to load channel monitors:', error)
      monitors.value = []
    }
  } finally {
    loadingMonitors.value = false
  }
}

async function refreshAll(): Promise<void> {
  await appStore.fetchPublicSettings()
  await Promise.allSettled([
    loadStats(),
    loadCharts(),
    announcementStore.fetchAnnouncements(true),
    loadMonitors(),
  ])
}

function openFilters(): void {
  draftStartDate.value = startDate.value
  draftEndDate.value = endDate.value
  draftGranularity.value = granularity.value
  showFilters.value = true
}

async function applyFilters(): Promise<void> {
  if (!canApplyFilters.value) return
  startDate.value = draftStartDate.value
  endDate.value = draftEndDate.value
  granularity.value = draftGranularity.value
  showFilters.value = false
  await loadCharts()
}

onMounted(() => {
  void refreshAll()
})

onBeforeUnmount(() => {
  monitorController?.abort()
  chartsRequestId += 1
})
</script>

<style scoped>
.dashboard-panel {
  border: 1px solid rgb(229 231 235 / 0.78);
  border-radius: 16px;
  background: rgb(255 255 255);
  box-shadow: 0 0 1px rgb(15 23 42 / 0.16), 0 7px 18px rgb(15 23 42 / 0.07);
}

.dashboard-icon-button {
  display: inline-flex;
  width: 2.25rem;
  height: 2.25rem;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 10px;
  color: rgb(107 114 128);
  background: transparent;
  box-shadow: none;
  transition: color 150ms ease, background-color 150ms ease;
}

.dashboard-icon-button:hover:not(:disabled) {
  color: rgb(79 105 224);
  background: transparent;
}

.dashboard-icon-button:focus-visible {
  outline: 2px solid rgb(129 140 248 / 0.55);
  outline-offset: 2px;
}

.dashboard-icon-button:disabled {
  cursor: wait;
  opacity: 0.6;
}

:global(.dark) .dashboard-panel {
  border-color: rgb(51 65 85 / 0.86);
  background: rgb(30 41 59);
  box-shadow: 0 0 1px rgb(0 0 0 / 0.45), 0 7px 20px rgb(0 0 0 / 0.2);
}

:global(.dark) .dashboard-icon-button {
  color: rgb(148 163 184);
  background: transparent;
  box-shadow: none;
}

:global(.dark) .dashboard-icon-button:hover:not(:disabled) {
  color: rgb(165 180 252);
  background: transparent;
}
</style>
