<template>
  <AppLayout>
    <div class="yunwu-dashboard">
      <header class="dashboard-page-header">
        <h1>{{ t('dashboard.title') }}</h1>
        <div class="dashboard-notification">
          <DashboardNotificationPopover />
        </div>
      </header>

      <div v-if="loading && !stats" class="dashboard-loading" aria-live="polite">
        <div class="dashboard-loading-metrics">
          <div v-for="index in 4" :key="index" class="dashboard-panel dashboard-loading-card">
            <span class="skeleton h-4 w-24" />
            <span class="skeleton mt-6 h-9 w-32" />
            <span v-if="index === 2 || index === 3" class="skeleton mt-3 h-3 w-40 max-w-full" />
          </div>
        </div>
        <div class="dashboard-loading-section">
          <div class="dashboard-loading-section__header">
            <span class="skeleton h-5 w-24" />
            <span class="skeleton h-9 w-32" />
          </div>
          <div class="dashboard-loading-body">
            <div class="dashboard-panel min-h-[408px] p-5">
              <span class="skeleton h-5 w-40" />
              <span class="skeleton mt-8 block h-[320px] w-full" />
            </div>
            <div class="space-y-4">
              <div class="dashboard-panel min-h-[228px] p-5">
                <span class="skeleton h-5 w-44" />
                <span class="skeleton mt-5 block h-36 w-full" />
              </div>
              <div class="dashboard-panel min-h-[164px] p-5">
                <span class="skeleton h-5 w-36" />
                <span class="skeleton mt-5 block h-16 w-full" />
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-else-if="!stats" class="dashboard-panel dashboard-error" role="alert">
        <Icon name="exclamationCircle" size="lg" aria-hidden="true" />
        <h2>{{ t('dashboard.loadErrorTitle') }}</h2>
        <p>{{ t('dashboard.loadErrorDescription') }}</p>
        <button type="button" class="dashboard-primary-button" :disabled="isRefreshing" @click="refreshAll">
          <Icon name="refresh" size="sm" aria-hidden="true" />
          {{ t('dashboard.retry') }}
        </button>
      </div>

      <template v-else>
        <UserDashboardStats
          :stats="stats"
          :balance="accountBalance"
          :plan-name="currentPlanName"
          :quota-remaining-percent="quotaRemainingPercent"
          :plan-loading="planLoading"
        />

        <UserDashboardCharts
          :loading="loadingCharts"
          :period="usagePeriod"
          :trend="trendData"
          @update:period="usagePeriod = $event"
        />

        <UserDashboardInsights :period="usagePeriod" />
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useUserProfileStore } from '@/stores/userProfile'
import { useUserMembership } from '@/composables/useUserMembership'
import { usageAPI, type UserDashboardStats as UserStatsType } from '@/api/usage'
import subscriptionsAPI from '@/api/subscriptions'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import DashboardNotificationPopover from '@/components/user/dashboard/DashboardNotificationPopover.vue'
import UserDashboardStats from '@/components/user/dashboard/UserDashboardStats.vue'
import UserDashboardCharts from '@/components/user/dashboard/UserDashboardCharts.vue'
import UserDashboardInsights from '@/components/user/dashboard/UserDashboardInsights.vue'
import {
  resolveDashboardUsagePeriod,
  type DashboardUsagePeriod,
} from '@/components/user/dashboard/dashboardUsage'
import type { SubscriptionProgressInfo, SubscriptionUsageWindowProgress, TrendDataPoint } from '@/types'

const { t } = useI18n()
const authStore = useAuthStore()
const userProfileStore = useUserProfileStore()
const {
  profile,
  primarySubscription,
  subscriptionsLoading,
  subscriptionsLoaded,
  activeSubscriptionCount,
} = storeToRefs(userProfileStore)
const membership = useUserMembership({
  subscriptionsLoaded,
  activeSubscriptionCount,
  primarySubscription,
})

const stats = ref<UserStatsType | null>(null)
const loading = ref(true)
const loadingCharts = ref(false)
const quotaLoading = ref(false)
const trendData = ref<TrendDataPoint[]>([])
const subscriptionProgress = ref<SubscriptionProgressInfo[]>([])
const usagePeriod = ref<DashboardUsagePeriod>('today')
let chartsRequestId = 0
let quotaRequestId = 0

const currentPlanName = membership.accountPlanLabel
const accountBalance = computed(() => (
  profile.value?.availableBalance
  ?? authStore.user?.balance
  ?? 0
))
const planLoading = computed(() => (
  membership.state.value === 'pending'
  || subscriptionsLoading.value
  || quotaLoading.value
))
const isRefreshing = computed(() => loading.value || loadingCharts.value || quotaLoading.value)

const quotaRemainingPercent = computed<number | null>(() => {
  const subscriptionId = primarySubscription.value?.id
  if (subscriptionId == null || membership.state.value !== 'active') return null
  const progress = subscriptionProgress.value.find(
    item => item.subscription.id === subscriptionId,
  )?.progress
  if (!progress) return null

  const windows: Array<SubscriptionUsageWindowProgress | undefined> = [
    progress.daily,
    progress.weekly,
    progress.monthly,
  ]
  const usedPercentages = windows
    .filter((window): window is SubscriptionUsageWindowProgress => (
      window?.state !== 'unknown'
      && typeof window?.percentage === 'number'
      && Number.isFinite(window.percentage)
    ))
    .map(window => Math.min(100, Math.max(0, Number(window.percentage))))

  if (!usedPercentages.length) return null
  return Math.max(0, 100 - Math.max(...usedPercentages))
})

async function loadStats(): Promise<void> {
  loading.value = true
  try {
    stats.value = await usageAPI.getDashboardStats()
  } catch (error) {
    console.error('Failed to load dashboard stats:', error)
  } finally {
    loading.value = false
  }
}

async function loadCharts(): Promise<void> {
  const requestId = ++chartsRequestId
  const range = resolveDashboardUsagePeriod(usagePeriod.value)
  loadingCharts.value = true
  try {
    const result = await usageAPI.getDashboardTrend({
      start_date: range.startDate,
      end_date: range.endDate,
      granularity: range.granularity,
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    })
    if (requestId !== chartsRequestId) return
    trendData.value = result.trend || []
  } catch (error) {
    if (requestId === chartsRequestId) {
      console.error('Failed to load dashboard charts:', error)
    }
  } finally {
    if (requestId === chartsRequestId) loadingCharts.value = false
  }
}

async function loadQuotaProgress(): Promise<void> {
  const requestId = ++quotaRequestId
  quotaLoading.value = true
  try {
    const result = await subscriptionsAPI.getSubscriptionsProgress()
    if (requestId !== quotaRequestId) return
    subscriptionProgress.value = result
  } catch (error) {
    if (requestId === quotaRequestId) {
      subscriptionProgress.value = []
      console.warn('Failed to load subscription quota progress:', error)
    }
  } finally {
    if (requestId === quotaRequestId) quotaLoading.value = false
  }
}

async function loadDashboard(refreshProfile: boolean): Promise<void> {
  if (refreshProfile) {
    try {
      await userProfileStore.refreshProfile()
    } catch (error) {
      console.warn('Failed to refresh dashboard profile:', error)
    }
  }
  await Promise.allSettled([
    loadStats(),
    loadCharts(),
    loadQuotaProgress(),
  ])
}

async function refreshAll(): Promise<void> {
  await loadDashboard(true)
}

watch(usagePeriod, () => {
  void loadCharts()
})

watch(() => primarySubscription.value?.id, (current, previous) => {
  if (current !== previous) void loadQuotaProgress()
})

onMounted(() => {
  void loadDashboard(false)
})

onBeforeUnmount(() => {
  chartsRequestId += 1
  quotaRequestId += 1
})
</script>

<style scoped>
.yunwu-dashboard {
  display: flex;
  width: 100%;
  min-width: 0;
  flex-direction: column;
  gap: var(--workspace-space-8);
  color: var(--workspace-work-text);
  background: var(--workspace-canvas);
}

.dashboard-page-header {
  display: flex;
  min-height: 44px;
  align-items: center;
  justify-content: space-between;
  gap: var(--workspace-space-6);
}

.dashboard-page-header h1 {
  color: var(--workspace-dashboard-text-strong);
  font-size: calc(var(--workspace-type-page-title-size) + 0.125rem);
  font-weight: 700;
  line-height: 1.2;
}

.dashboard-notification {
  display: flex;
  align-items: center;
  justify-content: center;
}


.dashboard-panel {
  min-width: 0;
  border: 1px solid var(--workspace-dashboard-card-border);
  border-radius: 24px;
  background: var(--workspace-card-surface);
  box-shadow: var(--workspace-dashboard-card-shadow);
}

.dashboard-loading,
.dashboard-loading-metrics,
.dashboard-loading-body {
  display: grid;
  gap: var(--workspace-space-4);
}

.dashboard-loading-metrics {
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: var(--workspace-space-6);
}

.dashboard-loading-card {
  min-height: 138px;
  padding: var(--workspace-space-6);
}

.dashboard-loading-section__header {
  display: flex;
  min-height: 44px;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--workspace-space-4);
}

.dashboard-loading-body {
  grid-template-columns: minmax(0, 3fr) minmax(20rem, 2fr);
  gap: var(--workspace-space-6);
}

.dashboard-error {
  display: flex;
  min-height: 360px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 32px;
  text-align: center;
}

.dashboard-error > :deep(svg) {
  color: var(--workspace-work-accent);
}

.dashboard-error h2 {
  margin-top: 16px;
  color: var(--workspace-dashboard-text-strong);
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
}

.dashboard-error p {
  max-width: 30rem;
  margin-top: 8px;
  color: var(--workspace-dashboard-text-muted);
  font-size: var(--workspace-type-body-size);
  font-weight: var(--workspace-type-body-weight);
  line-height: 1.3rem;
}

.dashboard-primary-button {
  display: inline-flex;
  min-height: 38px;
  align-items: center;
  justify-content: center;
  gap: 8px;
  margin-top: 20px;
  padding: 0 14px;
  border: 1px solid var(--workspace-work-accent);
  border-radius: var(--workspace-radius-work-button);
  color: var(--workspace-work-on-accent);
  background: var(--workspace-work-accent);
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
  transition: background-color 140ms ease, border-color 140ms ease;
}

.dashboard-primary-button:hover:not(:disabled) {
  border-color: var(--workspace-work-accent-hover);
  background: var(--workspace-work-accent-hover);
}

.dashboard-primary-button:focus-visible {
  outline: 2px solid var(--workspace-work-accent);
  outline-offset: 2px;
}

.dashboard-primary-button:disabled {
  cursor: wait;
  opacity: 0.55;
}

@media (max-width: 1279px) {
  .dashboard-loading-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .dashboard-loading-body {
    grid-template-columns: minmax(0, 1fr);
  }
}

@media (max-width: 767px) {
  .yunwu-dashboard {
    gap: 20px;
  }

  .dashboard-primary-button {
    min-height: 44px;
  }
}

@media (max-width: 639px) {
  .dashboard-loading-metrics {
    grid-template-columns: minmax(0, 1fr);
  }
}

@media (prefers-reduced-motion: reduce) {
  .dashboard-primary-button {
    transition-duration: 0.01ms;
  }
}

</style>
