<template>
  <AppLayout>
    <div class="space-y-7 md:space-y-9">
      <header class="flex flex-col gap-5 md:flex-row md:items-end md:justify-between">
        <div class="max-w-3xl">
          <div class="mb-3 flex items-center gap-2.5 text-xs font-medium uppercase tracking-[0.18em] text-gray-500 dark:text-dark-400">
            <span class="h-2 w-2 rounded-full bg-primary-500"></span>
            <span>{{ appStore.siteName }} · {{ t('dashboard.overviewEyebrow') }}</span>
          </div>
          <h2 class="text-[clamp(2rem,4vw,4.25rem)] font-semibold leading-[0.98] tracking-[-0.045em] text-gray-950 dark:text-white">
            {{ t('dashboard.overviewTitle') }}
          </h2>
          <p class="mt-4 max-w-2xl text-sm leading-6 text-gray-500 dark:text-dark-400 md:text-base">
            {{ t('dashboard.overviewDescription') }}
          </p>
        </div>

        <button
          type="button"
          class="btn btn-secondary self-start rounded-full px-5 md:self-auto"
          :disabled="isRefreshing"
          @click="refreshAll"
        >
          <Icon name="refresh" size="sm" :class="{ 'animate-spin': isRefreshing }" />
          {{ t('common.refresh') }}
        </button>
      </header>

      <div v-if="loading" class="space-y-6" aria-live="polite">
        <div class="grid grid-cols-1 gap-5 md:grid-cols-3">
          <div v-for="index in 3" :key="index" class="card flex min-h-56 flex-col justify-between p-6 md:p-7">
            <div class="skeleton h-3 w-3 rounded-full"></div>
            <div>
              <div class="skeleton h-7 w-36"></div>
              <div class="skeleton mt-3 h-3 w-44"></div>
            </div>
          </div>
        </div>
        <div class="card grid grid-cols-2 gap-px overflow-hidden bg-gray-100 md:grid-cols-4 dark:bg-dark-700">
          <div v-for="index in 4" :key="index" class="bg-white p-6 dark:bg-dark-800">
            <div class="skeleton h-3 w-24"></div>
            <div class="skeleton mt-5 h-9 w-20"></div>
          </div>
        </div>
      </div>
      <template v-else-if="stats">
        <UserDashboardQuickActions />
        <UserDashboardStats
          :stats="stats"
          :balance="user?.balance || 0"
          :is-simple="authStore.isSimpleMode"
          :platform-quotas="platformQuotas"
        />
        <UserDashboardCharts
          v-model:startDate="startDate"
          v-model:endDate="endDate"
          v-model:granularity="granularity"
          :loading="loadingCharts"
          :trend="trendData"
          :models="modelStats"
          @dateRangeChange="loadCharts"
          @granularityChange="loadCharts"
          @refresh="refreshAll"
        />
        <UserDashboardRecentUsage :data="recentUsage" :loading="loadingUsage" />
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { usageAPI, type UserDashboardStats as UserStatsType } from '@/api/usage'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import UserDashboardStats from '@/components/user/dashboard/UserDashboardStats.vue'
import UserDashboardCharts from '@/components/user/dashboard/UserDashboardCharts.vue'
import UserDashboardRecentUsage from '@/components/user/dashboard/UserDashboardRecentUsage.vue'
import UserDashboardQuickActions from '@/components/user/dashboard/UserDashboardQuickActions.vue'
import type { UsageLog, TrendDataPoint, ModelStat, PlatformQuotaItem } from '@/types'
import { getMyPlatformQuotas } from '@/api/user'
import { formatDateLocalInput } from '@/utils/format'

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()
const user = computed(() => authStore.user)
const stats = ref<UserStatsType | null>(null)
const loading = ref(false)
const loadingUsage = ref(false)
const loadingCharts = ref(false)
const isRefreshing = computed(() => loading.value || loadingUsage.value || loadingCharts.value)
const trendData = ref<TrendDataPoint[]>([])
const modelStats = ref<ModelStat[]>([])
const recentUsage = ref<UsageLog[]>([])
const platformQuotas = ref<PlatformQuotaItem[] | null>(null)

const startDate = ref(formatDateLocalInput(new Date(Date.now() - 6 * 86400000)))
const endDate = ref(formatDateLocalInput(new Date()))
const granularity = ref('day')

const loadStats = async () => { loading.value = true; try { await authStore.refreshUser(); stats.value = await usageAPI.getDashboardStats() } catch (error) { console.error('Failed to load dashboard stats:', error) } finally { loading.value = false } }
const loadCharts = async () => { loadingCharts.value = true; try { const res = await Promise.all([usageAPI.getDashboardTrend({ start_date: startDate.value, end_date: endDate.value, granularity: granularity.value as any }), usageAPI.getDashboardModels({ start_date: startDate.value, end_date: endDate.value })]); trendData.value = res[0].trend || []; modelStats.value = res[1].models || [] } catch (error) { console.error('Failed to load charts:', error) } finally { loadingCharts.value = false } }
const loadRecent = async () => { loadingUsage.value = true; try { const res = await usageAPI.getByDateRange(startDate.value, endDate.value); recentUsage.value = res.items.slice(0, 5) } catch (error) { console.error('Failed to load recent usage:', error) } finally { loadingUsage.value = false } }
const loadPlatformQuotas = async () => { try { const data = await getMyPlatformQuotas(); platformQuotas.value = data.platform_quotas ?? [] } catch (error) { console.warn('Failed to load platform quotas:', error); platformQuotas.value = [] } }
const refreshAll = () => { loadStats(); loadCharts(); loadRecent(); loadPlatformQuotas() }

onMounted(() => { refreshAll() })
</script>
