<template>
  <AppLayout>
    <div class="yunwu-dashboard">
      <header class="dashboard-page-header">
        <div class="min-w-0">
          <h1>{{ t('dashboard.workspace.greeting', { period: greetingPeriod, name: dashboardUserName }) }}</h1>
          <p>{{ t('dashboard.workspace.welcomeDescription') }}</p>
        </div>

        <div class="dashboard-page-actions">
          <button
            type="button"
            class="dashboard-toolbar-button dashboard-toolbar-button--range"
            :aria-label="t('dashboard.openFilters')"
            @click="openFilters"
          >
            <Icon name="calendar" size="sm" :stroke-width="1.7" aria-hidden="true" />
            <span>{{ formattedDateRange }}</span>
          </button>
          <button
            type="button"
            class="dashboard-toolbar-button dashboard-toolbar-button--icon"
            :disabled="isRefreshing"
            :aria-label="t('dashboard.refreshDashboard')"
            :title="t('dashboard.refreshDashboard')"
            @click="refreshAll"
          >
            <Icon
              name="refresh"
              size="sm"
              :stroke-width="1.7"
              :class="{ 'animate-spin': isRefreshing }"
              aria-hidden="true"
            />
          </button>
        </div>
      </header>

      <div v-if="loading && !stats" class="dashboard-loading" aria-live="polite">
        <div class="dashboard-loading-metrics">
          <div v-for="index in 4" :key="index" class="dashboard-panel dashboard-loading-card">
            <span class="skeleton h-4 w-24" />
            <span class="skeleton mt-6 h-9 w-32" />
            <span class="skeleton mt-3 h-3 w-40 max-w-full" />
          </div>
        </div>
        <div class="dashboard-loading-body">
          <div class="dashboard-panel min-h-[420px] p-5">
            <span class="skeleton h-5 w-32" />
            <span class="skeleton mt-8 block h-[300px] w-full" />
          </div>
          <div class="space-y-4">
            <div class="dashboard-panel min-h-[210px] p-5">
              <span class="skeleton h-5 w-28" />
            </div>
            <div class="dashboard-panel min-h-[210px] p-5">
              <span class="skeleton h-5 w-28" />
            </div>
          </div>
        </div>
      </div>

      <div v-else-if="!stats" class="dashboard-panel dashboard-error" role="alert">
        <Icon name="exclamationCircle" size="lg" aria-hidden="true" />
        <h2>{{ t('dashboard.loadErrorTitle') }}</h2>
        <p>{{ t('dashboard.loadErrorDescription') }}</p>
        <button type="button" class="dashboard-primary-button" @click="refreshAll">
          <Icon name="refresh" size="sm" aria-hidden="true" />
          {{ t('dashboard.retry') }}
        </button>
      </div>

      <template v-else>
        <UserDashboardStats
          :stats="stats"
          :balance="user?.balance || 0"
          :plan-name="currentPlanName"
          :plan-expires-at="primarySubscription?.expiresAt || null"
          :plan-loading="membership.state.value === 'pending' || subscriptionsLoading"
          :subscriptions-loaded="subscriptionsLoaded"
          :has-active-subscription="membership.hasActiveMembership.value"
        />

        <div class="dashboard-secondary-grid">
          <UserDashboardCharts
            class="dashboard-secondary-grid__models"
            :loading="loadingCharts"
            :start-date="startDate"
            :end-date="endDate"
            :trend="trendData"
            :models="modelStats"
          />
          <UserDashboardRecentConversations
            :conversations="chatStore.conversations"
            :loading="loadingConversations || chatStore.hydrating || chatStore.loadingConversationPage"
            @select="openConversation"
          />
          <UserDashboardQuickActions />
        </div>

        <div class="dashboard-tertiary-grid">
          <UserDashboardApiInfo
            :api-base-url="apiBaseUrl"
            :custom-endpoints="customEndpoints"
          />
          <UserDashboardAccountInfo
            :display-name="dashboardUserName"
            :email="user?.email || t('dashboard.workspace.notAvailable')"
            :status="user?.status || 'active'"
            :active-api-keys="stats.active_api_keys"
            :total-api-keys="stats.total_api_keys"
            :created-at="user?.created_at || ''"
          />
        </div>
      </template>
    </div>

    <BaseDialog
      :show="showFilters"
      :title="t('dashboard.filterTitle')"
      width="narrow"
      @close="showFilters = false"
    >
      <div class="space-y-4">
        <label class="block">
          <span class="dashboard-filter-label">{{ t('dashboard.filterStart') }}</span>
          <input
            v-model="draftStartDate"
            type="date"
            class="dashboard-filter-input"
            :max="draftEndDate"
          />
        </label>
        <label class="block">
          <span class="dashboard-filter-label">{{ t('dashboard.filterEnd') }}</span>
          <input
            v-model="draftEndDate"
            type="date"
            class="dashboard-filter-input"
            :min="draftStartDate"
            :max="today"
          />
        </label>
        <label class="block">
          <span class="dashboard-filter-label">{{ t('dashboard.filterGranularity') }}</span>
          <select v-model="draftGranularity" class="dashboard-filter-input">
            <option value="day">{{ t('dashboard.day') }}</option>
            <option value="hour">{{ t('dashboard.hour') }}</option>
          </select>
        </label>
      </div>

      <template #footer>
        <button type="button" class="dashboard-dialog-button" @click="showFilters = false">
          {{ t('common.cancel') }}
        </button>
        <button
          type="button"
          class="dashboard-dialog-button dashboard-dialog-button--primary"
          :disabled="!canApplyFilters"
          @click="applyFilters"
        >
          {{ t('common.confirm') }}
        </button>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { useUserProfileStore } from '@/stores/userProfile'
import { useChatStore } from '@/stores/chat'
import { useUserMembership } from '@/composables/useUserMembership'
import { usageAPI, type UserDashboardStats as UserStatsType } from '@/api/usage'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import UserDashboardStats from '@/components/user/dashboard/UserDashboardStats.vue'
import UserDashboardCharts from '@/components/user/dashboard/UserDashboardCharts.vue'
import UserDashboardApiInfo from '@/components/user/dashboard/UserDashboardApiInfo.vue'
import UserDashboardRecentConversations from '@/components/user/dashboard/UserDashboardRecentConversations.vue'
import UserDashboardQuickActions from '@/components/user/dashboard/UserDashboardQuickActions.vue'
import UserDashboardAccountInfo from '@/components/user/dashboard/UserDashboardAccountInfo.vue'
import type { CustomEndpoint, ModelStat, TrendDataPoint } from '@/types'
import { formatDateLocalInput } from '@/utils/format'

const { t, locale } = useI18n()
const router = useRouter()
const authStore = useAuthStore()
const appStore = useAppStore()
const userProfileStore = useUserProfileStore()
const chatStore = useChatStore()
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
const user = computed(() => authStore.user)

const stats = ref<UserStatsType | null>(null)
const loading = ref(true)
const loadingCharts = ref(false)
const loadingConversations = ref(false)
const trendData = ref<TrendDataPoint[]>([])
const modelStats = ref<ModelStat[]>([])
const today = formatDateLocalInput(new Date())

const startDate = ref(formatDateLocalInput(new Date(Date.now() - 6 * 86_400_000)))
const endDate = ref(today)
const granularity = ref<'day' | 'hour'>('day')
const showFilters = ref(false)
const draftStartDate = ref(startDate.value)
const draftEndDate = ref(endDate.value)
const draftGranularity = ref<'day' | 'hour'>(granularity.value)
let chartsRequestId = 0

const greetingPeriod = computed(() => {
  const hour = new Date().getHours()
  if (hour < 12) return t('dashboard.greeting.morning')
  if (hour < 18) return t('dashboard.greeting.afternoon')
  return t('dashboard.greeting.evening')
})

const dashboardUserName = computed(() => (
  profile.value?.displayName
  || user.value?.email?.split('@')[0]
  || appStore.siteName
))

const currentPlanName = membership.accountPlanLabel

const formattedDateRange = computed(() => {
  const start = new Date(`${startDate.value}T00:00:00`)
  const end = new Date(`${endDate.value}T00:00:00`)
  if (!Number.isFinite(start.getTime()) || !Number.isFinite(end.getTime())) {
    return `${startDate.value} – ${endDate.value}`
  }
  const formatter = new Intl.DateTimeFormat(locale.value.startsWith('zh') ? 'zh-CN' : 'en-US', {
    month: 'short',
    day: 'numeric',
  })
  return `${formatter.format(start)} – ${formatter.format(end)}`
})

const apiBaseUrl = computed(() => (
  appStore.cachedPublicSettings?.api_base_url
  || appStore.apiBaseUrl
  || (typeof window === 'undefined' ? '' : window.location.origin)
))
const customEndpoints = computed<CustomEndpoint[]>(() => appStore.cachedPublicSettings?.custom_endpoints || [])
const isRefreshing = computed(() => (
  loading.value
  || loadingCharts.value
  || loadingConversations.value
  || subscriptionsLoading.value
))
const canApplyFilters = computed(() => Boolean(
  draftStartDate.value
  && draftEndDate.value
  && draftStartDate.value <= draftEndDate.value
  && draftEndDate.value <= today,
))

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

async function loadRecentConversations(): Promise<void> {
  const userId = user.value?.id
  if (userId === null || userId === undefined) return
  loadingConversations.value = true
  try {
    await chatStore.hydrate(userId)
    if (chatStore.userId !== String(userId)) return
    await chatStore.syncHistory()
    if (chatStore.userId !== String(userId)) return
    await chatStore.loadConversationPage(true)
  } catch (error) {
    console.warn('Failed to load recent conversations:', error)
  } finally {
    loadingConversations.value = false
  }
}

async function loadDashboard(refreshProfile: boolean): Promise<void> {
  await appStore.fetchPublicSettings()
  await Promise.allSettled([
    loadStats(),
    loadCharts(),
    ...(refreshProfile ? [userProfileStore.refreshProfile()] : []),
  ])
  await loadRecentConversations()
}

async function refreshAll(): Promise<void> {
  await loadDashboard(true)
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

async function openConversation(id: string): Promise<void> {
  if (!chatStore.selectConversation(id)) return
  void chatStore.loadConversationDetail(id)
  await router.push('/chat')
}

onMounted(() => {
  void loadDashboard(false)
})

onBeforeUnmount(() => {
  chartsRequestId += 1
})
</script>

<style scoped>
.yunwu-dashboard {
  display: flex;
  width: 100%;
  min-width: 0;
  flex-direction: column;
  gap: var(--workspace-space-6);
  color: var(--workspace-work-text);
  background: var(--workspace-canvas);
}

.dashboard-page-header {
  display: flex;
  min-height: 64px;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--workspace-space-6);
}

.dashboard-page-header h1 {
  color: var(--workspace-work-text);
  font-size: var(--workspace-type-page-title-size);
  font-weight: var(--workspace-type-page-title-weight);
  line-height: 1.2;
  letter-spacing: -0.025em;
}

.dashboard-page-header p {
  max-width: 65ch;
  margin-top: 7px;
  color: var(--workspace-work-text-muted);
  font-size: var(--workspace-type-body-size);
  font-weight: var(--workspace-type-body-weight);
  line-height: 1.35rem;
}

.dashboard-page-actions {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 8px;
}

.dashboard-toolbar-button,
.dashboard-primary-button,
.dashboard-dialog-button {
  display: inline-flex;
  min-height: 38px;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: 1px solid var(--workspace-border);
  border-radius: var(--workspace-radius-work-button);
  color: var(--workspace-work-text-secondary);
  background: var(--workspace-card-surface);
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
  transition: color 140ms ease, background-color 140ms ease, border-color 140ms ease;
}

.dashboard-toolbar-button:hover:not(:disabled),
.dashboard-dialog-button:hover:not(:disabled) {
  border-color: var(--workspace-border-strong);
  color: var(--workspace-work-text);
  background: var(--workspace-surface-subtle);
}

.dashboard-toolbar-button:focus-visible,
.dashboard-primary-button:focus-visible,
.dashboard-dialog-button:focus-visible,
.dashboard-filter-input:focus-visible {
  outline: 2px solid var(--workspace-work-accent);
  outline-offset: 2px;
}

.dashboard-toolbar-button:disabled,
.dashboard-dialog-button:disabled {
  cursor: wait;
  opacity: 0.55;
}

.dashboard-toolbar-button--range {
  padding: 0 12px;
}

.dashboard-toolbar-button--icon {
  width: 38px;
  padding: 0;
}

.dashboard-panel {
  min-width: 0;
  border: 1px solid var(--workspace-border);
  border-radius: var(--workspace-radius-work-card);
  background: var(--workspace-card-surface);
  box-shadow: var(--workspace-work-shadow-card);
}

.dashboard-loading,
.dashboard-loading-metrics,
.dashboard-loading-body {
  display: grid;
  gap: var(--workspace-space-4);
}

.dashboard-loading-metrics {
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.dashboard-loading-card {
  min-height: 180px;
  padding: var(--workspace-space-5);
}

.dashboard-loading-body {
  grid-template-columns: minmax(0, 1.6fr) minmax(18rem, 0.75fr);
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
  color: var(--workspace-work-text);
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
}

.dashboard-error p {
  max-width: 30rem;
  margin-top: 8px;
  color: var(--workspace-work-text-muted);
  font-size: var(--workspace-type-body-size);
  font-weight: var(--workspace-type-body-weight);
  line-height: 1.3rem;
}

.dashboard-primary-button,
.dashboard-dialog-button--primary {
  border-color: var(--workspace-work-accent);
  color: var(--workspace-light-surface);
  background: var(--workspace-work-accent);
}

.dashboard-primary-button {
  margin-top: 20px;
  padding: 0 14px;
}

.dashboard-primary-button:hover,
.dashboard-dialog-button--primary:hover:not(:disabled) {
  border-color: var(--workspace-work-accent-hover);
  color: var(--workspace-light-surface);
  background: var(--workspace-work-accent-hover);
}

.dashboard-secondary-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.55fr) minmax(18rem, 0.8fr);
  gap: var(--workspace-space-4);
  align-items: stretch;
}

.dashboard-secondary-grid__models {
  grid-row: span 2;
}

.dashboard-tertiary-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.25fr) minmax(19rem, 0.75fr);
  gap: var(--workspace-space-4);
  align-items: stretch;
}

.dashboard-filter-label {
  display: block;
  margin-bottom: 6px;
  color: var(--workspace-work-text-secondary);
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
}

.dashboard-filter-input {
  width: 100%;
  min-height: 40px;
  padding: 0 12px;
  border: 1px solid var(--workspace-border-strong);
  border-radius: var(--workspace-radius-button);
  color: var(--workspace-work-text);
  background: var(--workspace-card-surface);
  font-size: var(--workspace-type-body-size);
  font-weight: var(--workspace-type-body-weight);
}

.dashboard-dialog-button {
  padding: 0 14px;
}

@media (max-width: 1279px) {
  .dashboard-loading-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .dashboard-secondary-grid,
  .dashboard-tertiary-grid,
  .dashboard-loading-body {
    grid-template-columns: minmax(0, 1fr);
  }

  .dashboard-secondary-grid__models {
    grid-row: auto;
  }
}

@media (max-width: 767px) {
  .yunwu-dashboard {
    gap: 20px;
  }

  .dashboard-page-header {
    flex-direction: column;
    gap: var(--workspace-space-4);
  }

  .dashboard-page-actions {
    width: 100%;
  }

  .dashboard-toolbar-button {
    min-height: 44px;
  }

  .dashboard-primary-button,
  .dashboard-dialog-button {
    min-height: 44px;
  }

  .dashboard-toolbar-button--range {
    min-width: 0;
    flex: 1 1 auto;
  }

  .dashboard-toolbar-button--icon {
    width: 44px;
  }
}

@media (max-width: 479px) {
  .dashboard-loading-metrics {
    grid-template-columns: minmax(0, 1fr);
  }
}

@media (prefers-reduced-motion: reduce) {
  .dashboard-toolbar-button,
  .dashboard-primary-button,
  .dashboard-dialog-button {
    transition-duration: 0.01ms;
  }
}

:global(html.dark) .yunwu-dashboard,
:global(html.dark) .dashboard-page-header h1,
:global(html.dark) .dashboard-error h2 {
  color: var(--workspace-dark-text);
}

:global(html.dark) .yunwu-dashboard {
  background: var(--workspace-canvas);
}

:global(html.dark) .dashboard-page-header p,
:global(html.dark) .dashboard-error p {
  color: var(--workspace-dark-text-muted);
}

:global(html.dark) .dashboard-panel,
:global(html.dark) .dashboard-toolbar-button,
:global(html.dark) .dashboard-dialog-button,
:global(html.dark) .dashboard-filter-input {
  border-color: var(--workspace-border);
  color: var(--workspace-dark-text-secondary);
  background: var(--workspace-card-surface);
}

:global(html.dark) .dashboard-filter-label {
  color: var(--workspace-dark-text-secondary);
}
</style>
