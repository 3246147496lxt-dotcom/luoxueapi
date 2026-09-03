<template>
  <section
    class="dashboard-insights"
    :aria-label="t('dashboard.workspace.insightsSection')"
  >
    <article class="dashboard-insights-card dashboard-model-ranking">
      <header class="dashboard-insights-card__header">
        <h2>{{ t('dashboard.workspace.modelRanking') }}</h2>
        <RouterLink class="dashboard-insights-card__link" to="/usage">
          {{ t('dashboard.workspace.viewAll') }}
        </RouterLink>
      </header>

      <div class="dashboard-model-table" role="table" :aria-label="t('dashboard.workspace.modelRanking')">
        <div class="dashboard-model-row dashboard-model-row--header" role="row">
          <span role="columnheader">{{ t('dashboard.workspace.model') }}</span>
          <span role="columnheader">{{ t('dashboard.workspace.requestCount') }}</span>
          <span role="columnheader">{{ t('dashboard.workspace.tokens') }}</span>
          <span role="columnheader">{{ t('dashboard.workspace.creditsUsed') }}</span>
          <span role="columnheader">{{ t('dashboard.workspace.usageShare') }}</span>
        </div>

        <div v-if="modelLoading && !modelRows.length" class="dashboard-insights-state" role="status">
          <span class="skeleton h-4 w-40" aria-hidden="true" />
          <span class="skeleton mt-3 h-4 w-56" aria-hidden="true" />
          <span class="sr-only">{{ t('dashboard.workspace.loadingModels') }}</span>
        </div>
        <div v-else-if="!modelRows.length" class="dashboard-insights-state" role="status">
          {{ t('dashboard.workspace.noModelUsage') }}
        </div>
        <template v-else>
          <div
            v-for="row in modelRows"
            :key="row.model"
            class="dashboard-model-row"
            role="row"
          >
            <div class="dashboard-model-row__model" role="cell">
              <span
                class="dashboard-model-row__icon"
                :class="{ 'dashboard-model-row__icon--openai': isOpenAIModel(row.model) }"
                aria-hidden="true"
              >
                <ModelIcon :model="row.model" size="18px" />
              </span>
              <span class="dashboard-model-row__name" :title="row.model">{{ row.model }}</span>
            </div>
            <span class="dashboard-model-row__number" role="cell">{{ formatInteger(row.requests) }}</span>
            <span class="dashboard-model-row__number" role="cell">{{ formatInteger(row.totalTokens) }}</span>
            <span class="dashboard-model-row__number" role="cell">{{ formatCredits(row.actualCost) }}</span>
            <div class="dashboard-model-row__share" role="cell">
              <span>{{ formatShare(row.share) }}</span>
              <span class="dashboard-model-row__progress" aria-hidden="true">
                <span :style="{ width: `${row.share}%` }" />
              </span>
            </div>
          </div>
        </template>
      </div>

      <p v-if="modelError" class="dashboard-insights-hint" role="status">
        {{ t('dashboard.workspace.modelRankingLoadError') }}
      </p>
    </article>

    <article class="dashboard-insights-card dashboard-channel-card">
      <header class="dashboard-channel-card__header">
        <div class="dashboard-channel-card__title-group">
          <h2>{{ t('dashboard.workspace.channelStatus') }}</h2>
          <span>{{ t('dashboard.workspace.last24Hours') }}</span>
        </div>
        <label class="dashboard-channel-select">
          <span class="sr-only">{{ t('dashboard.workspace.channelSelector') }}</span>
          <select v-model="selectedMonitorId" :disabled="channelLoading || !channelMonitors.length">
            <option v-if="!channelMonitors.length" value="">
              {{ t('dashboard.workspace.noChannels') }}
            </option>
            <option v-for="monitor in channelMonitors" :key="monitor.id" :value="monitor.id">
              {{ monitor.name }}
            </option>
          </select>
        </label>
      </header>

      <div class="dashboard-channel-card__body">
        <div v-if="channelLoading" class="dashboard-channel-card__loading" role="status">
          <span class="skeleton h-24 w-52 max-w-full rounded-full" aria-hidden="true" />
          <span class="sr-only">{{ t('common.loading') }}</span>
        </div>
        <template v-else>
          <svg
            class="dashboard-channel-gauge"
            viewBox="0 0 100 58"
            role="img"
            :aria-label="`${t('dashboard.workspace.channelStatus')} ${formattedSuccessRate}`"
          >
            <line
              v-for="index in gaugeSegmentCount"
              :key="index"
              x1="50"
              y1="7"
              x2="50"
              y2="15"
              stroke-linecap="round"
              stroke-width="3.5"
              :stroke="index <= filledGaugeSegments ? gaugeColor : gaugeTrackColor"
              :transform="`rotate(${-90 + ((index - 1) * 180) / (gaugeSegmentCount - 1)} 50 50)`"
            />
          </svg>
          <strong class="dashboard-channel-card__value">{{ formattedSuccessRate }}</strong>
          <button
            type="button"
            class="dashboard-channel-card__details"
            :disabled="!selectedMonitor"
            @click="openDetails"
          >
            {{ t('dashboard.workspace.viewDetails') }}
          </button>
          <p v-if="channelError" class="dashboard-insights-hint" role="status">
            {{ t('dashboard.workspace.channelLoadError') }}
          </p>
        </template>
      </div>
    </article>
  </section>

  <MonitorDetailDialog
    :show="showDetail"
    :monitor-id="selectedMonitor?.id ?? null"
    :title="selectedMonitor?.name || t('channelStatus.detailTitle')"
    @close="showDetail = false"
  />
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ModelStat } from '@/types'
import { usageAPI } from '@/api/usage'
import {
  channelMonitorUserAPI,
  type UserMonitorView,
  type MonitorTimelinePoint,
} from '@/api/channelMonitor'
import ModelIcon from '@/components/common/ModelIcon.vue'
import MonitorDetailDialog from '@/components/user/MonitorDetailDialog.vue'
import { resolveDashboardUsagePeriod, type DashboardUsagePeriod } from './dashboardUsage'

const props = defineProps<{
  period: DashboardUsagePeriod
}>()

const { t, locale } = useI18n()
const modelStats = ref<ModelStat[]>([])
const modelLoading = ref(false)
const modelError = ref(false)
const channelMonitors = ref<UserMonitorView[]>([])
const channelLoading = ref(false)
const channelError = ref(false)
const selectedMonitorId = ref<number | null>(null)
const showDetail = ref(false)
let modelRequestId = 0
let channelAbortController: AbortController | null = null

const numberLocale = computed(() => locale.value.startsWith('zh') ? 'zh-CN' : 'en-US')

const modelRows = computed(() => {
  const normalized = modelStats.value
    .map((item) => ({
      model: item.model || '—',
      requests: finiteNumber(item.requests),
      totalTokens: finiteNumber(item.total_tokens),
      actualCost: finiteNumber(item.actual_cost),
    }))
  const total = normalized.reduce((sum, item) => sum + item.actualCost, 0)
  return normalized
    .sort((a, b) => b.actualCost - a.actualCost)
    .slice(0, 5)
    .map(item => ({
      ...item,
      share: total > 0 ? Math.max(0, Math.min(100, (item.actualCost / total) * 100)) : 0,
    }))
})

const selectedMonitor = computed(() => (
  channelMonitors.value.find(item => item.id === selectedMonitorId.value) ?? null
))

interface ChannelHealth {
  rate: number | null
  status: 'operational' | 'degraded' | 'failed' | 'unknown'
}

const channelHealth = computed<ChannelHealth>(() => {
  const monitor = selectedMonitor.value
  if (!monitor) return { rate: null, status: 'unknown' }

  const cutoff = Date.now() - 24 * 60 * 60 * 1000
  const points = (monitor.timeline || []).filter(point => {
    const timestamp = Date.parse(point.checked_at)
    return Number.isFinite(timestamp) && timestamp >= cutoff
  })
  if (!points.length) return { rate: null, status: 'unknown' }

  const available = points.filter(point => point.status === 'operational' || point.status === 'degraded').length
  const latestStatus = latestStatusFor([monitor])
  return {
    rate: (available / points.length) * 100,
    status: latestStatus,
  }
})

const formattedSuccessRate = computed(() => (
  channelHealth.value.rate == null ? '--%' : `${channelHealth.value.rate.toFixed(1)}%`
))

const gaugeSegmentCount = 24
const filledGaugeSegments = computed(() => (
  channelHealth.value.rate == null
    ? 0
    : Math.round((Math.max(0, Math.min(100, channelHealth.value.rate)) / 100) * gaugeSegmentCount)
))

const gaugeColor = computed(() => {
  switch (channelHealth.value.status) {
    case 'operational': return 'var(--workspace-dashboard-success)'
    case 'degraded': return 'var(--lx-clay-warning-bright)'
    case 'failed': return 'var(--lx-clay-danger)'
    default: return 'var(--workspace-dashboard-text-subtle)'
  }
})
const gaugeTrackColor = 'var(--workspace-dashboard-track)'

async function loadModels(): Promise<void> {
  const requestId = ++modelRequestId
  modelLoading.value = true
  modelError.value = false
  const range = resolveDashboardUsagePeriod(props.period)
  try {
    const result = await usageAPI.getDashboardModels({
      start_date: range.startDate,
      end_date: range.endDate,
      model_source: 'requested',
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    })
    if (requestId !== modelRequestId) return
    modelStats.value = result.models || []
  } catch (error) {
    if (requestId !== modelRequestId) return
    modelError.value = true
    console.error('Failed to load dashboard model ranking:', error)
  } finally {
    if (requestId === modelRequestId) modelLoading.value = false
  }
}

async function loadChannels(): Promise<void> {
  channelAbortController?.abort()
  const controller = new AbortController()
  channelAbortController = controller
  channelLoading.value = true
  channelError.value = false
  try {
    const result = await channelMonitorUserAPI.list({ signal: controller.signal })
    if (controller.signal.aborted || channelAbortController !== controller) return
    channelMonitors.value = result.items || []
    if (!channelMonitors.value.some(item => item.id === selectedMonitorId.value)) {
      const preferredMonitor = channelMonitors.value.find(item => item.provider === 'openai')
        ?? channelMonitors.value[0]
      selectedMonitorId.value = preferredMonitor?.id ?? null
    }
  } catch (error) {
    const candidate = error as { name?: string; code?: string }
    if (candidate.name === 'AbortError' || candidate.code === 'ERR_CANCELED') return
    if (channelAbortController === controller) {
      channelError.value = true
      console.error('Failed to load dashboard channel status:', error)
    }
  } finally {
    if (channelAbortController === controller) {
      channelLoading.value = false
      channelAbortController = null
    }
  }
}

function latestStatusFor(items: UserMonitorView[]): ChannelHealth['status'] {
  const latest: MonitorTimelinePoint[] = items
    .flatMap(item => item.timeline || [])
    .filter(point => Number.isFinite(Date.parse(point.checked_at)))
    .sort((a, b) => Date.parse(b.checked_at) - Date.parse(a.checked_at))
  if (!latest.length) return 'unknown'
  if (latest.some(point => point.status === 'failed' || point.status === 'error')) return 'failed'
  if (latest.some(point => point.status === 'degraded')) return 'degraded'
  return 'operational'
}

function openDetails(): void {
  if (selectedMonitor.value) showDetail.value = true
}

function finiteNumber(value: number | null | undefined): number {
  return Number.isFinite(value) ? Number(value) : 0
}

function formatInteger(value: number): string {
  return new Intl.NumberFormat(numberLocale.value, { maximumFractionDigits: 0 }).format(value)
}

function formatCredits(value: number): string {
  return new Intl.NumberFormat(numberLocale.value, { maximumFractionDigits: 0 }).format(value)
}

function formatShare(value: number): string {
  return `${value.toFixed(1)}%`
}

function isOpenAIModel(model: string): boolean {
  const normalized = model.toLowerCase()
  return normalized.startsWith('gpt')
    || normalized.startsWith('o1')
    || normalized.startsWith('o3')
    || normalized.startsWith('o4')
    || normalized.includes('chatgpt')
}

watch(() => props.period, () => {
  void loadModels()
})

onMounted(() => {
  void loadModels()
  void loadChannels()
})

onBeforeUnmount(() => {
  modelRequestId += 1
  channelAbortController?.abort()
})
</script>

<style scoped>
.dashboard-insights {
  display: grid;
  min-width: 0;
  grid-template-columns: minmax(0, 3fr) minmax(20rem, 2fr);
  gap: var(--workspace-space-6);
  align-items: start;
}

.dashboard-insights-card {
  min-width: 0;
  padding: var(--workspace-space-6);
  border: 1px solid var(--workspace-dashboard-card-border);
  border-radius: 24px;
  background: var(--workspace-card-surface);
  box-shadow: var(--workspace-dashboard-card-shadow);
}

.dashboard-insights-card__header,
.dashboard-channel-card__header {
  display: flex;
  min-width: 0;
  align-items: baseline;
  justify-content: space-between;
  gap: var(--workspace-space-4);
  margin-bottom: var(--workspace-space-6);
}

.dashboard-insights-card__header h2,
.dashboard-channel-card__title-group h2 {
  min-width: 0;
  color: var(--workspace-dashboard-text-heading);
  font-size: calc(var(--workspace-type-navigation-size) + 0.125rem);
  font-weight: 700;
  line-height: 1.25rem;
}

.dashboard-insights-card__link {
  flex: 0 0 auto;
  color: var(--workspace-dashboard-text-subtle);
  font-size: calc(var(--workspace-type-secondary-size) - 1px);
  font-weight: 500;
  line-height: 1rem;
  text-decoration: none;
}

.dashboard-insights-card__link:hover,
.dashboard-insights-card__link:focus-visible {
  color: var(--workspace-work-accent);
  text-decoration: underline;
}

.dashboard-model-table {
  min-width: 0;
  overflow-x: visible;
}

.dashboard-model-row {
  display: grid;
  min-height: 48px;
  grid-template-columns: minmax(0, 34%) 14% 20% 14% 18%;
  align-items: center;
  border-top: 1px solid var(--workspace-dashboard-subtle-divider);
  column-gap: 0;
}

.dashboard-model-row--header {
  min-height: 36px;
  border-top: 0;
  color: var(--workspace-dashboard-text-subtle);
  font-size: calc(var(--workspace-type-secondary-size) - 1px);
  font-weight: 600;
}

.dashboard-model-row--header > span:not(:first-child),
.dashboard-model-row__number {
  text-align: right;
}

.dashboard-model-row__model,
.dashboard-model-row__share {
  display: flex;
  min-width: 0;
  align-items: center;
}

.dashboard-model-row__model {
  gap: 8px;
  padding-right: 12px;
}

.dashboard-model-row__model :deep(.model-icon),
.dashboard-model-row__model :deep(.model-icon-fallback) {
  flex: 0 0 auto;
}

.dashboard-model-row__icon {
  display: inline-flex;
  width: 20px;
  height: 20px;
  flex: 0 0 20px;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  background: var(--workspace-dashboard-track);
}

.dashboard-model-row__name {
  min-width: 0;
  overflow: hidden;
  color: var(--workspace-dashboard-text-strong);
  font-size: calc(var(--workspace-type-secondary-size) - 1px);
  font-weight: 500;
  line-height: 1.25rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-model-row__number,
.dashboard-model-row__share {
  color: var(--workspace-dashboard-text-muted);
  font-size: calc(var(--workspace-type-secondary-size) - 1px);
  font-weight: 500;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.dashboard-model-row__share {
  justify-content: flex-end;
  gap: 6px;
  padding-left: 8px;
}

.dashboard-model-row__progress {
  display: block;
  width: 32px;
  height: 4px;
  overflow: hidden;
  flex: 0 0 32px;
  border-radius: 999px;
  background: var(--workspace-dashboard-track);
}

.dashboard-model-row__progress > span {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--workspace-work-accent);
}

.dashboard-insights-state {
  display: flex;
  min-height: 180px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: var(--workspace-dashboard-text-muted);
  font-size: calc(var(--workspace-type-secondary-size) - 1px);
  text-align: center;
}

.dashboard-insights-hint {
  margin-top: 12px;
  color: var(--workspace-dashboard-text-subtle);
  font-size: calc(var(--workspace-type-secondary-size) - 1px);
  line-height: 1.1rem;
}

.dashboard-channel-card {
  display: flex;
  min-height: 288px;
  flex-direction: column;
}

.dashboard-channel-card__header {
  align-items: baseline;
  margin-bottom: var(--workspace-space-4);
}

.dashboard-channel-card__title-group {
  display: flex;
  min-width: 0;
  align-items: baseline;
  gap: 8px;
}

.dashboard-channel-card__title-group > span {
  color: var(--workspace-dashboard-text-subtle);
  font-size: calc(var(--workspace-type-secondary-size) - 2px);
  font-weight: 500;
  white-space: nowrap;
}

.dashboard-channel-select {
  display: inline-flex;
  min-width: 0;
  align-items: center;
}

.dashboard-channel-select select {
  width: min(14rem, 48%);
  max-width: 14rem;
  min-width: 0;
  min-height: 32px;
  padding: 0 24px 0 8px;
  border: 1px solid var(--workspace-dashboard-card-border);
  border-radius: 8px;
  outline: none;
  color: var(--workspace-dashboard-text-heading);
  background: var(--workspace-dashboard-track);
  font-size: calc(var(--workspace-type-secondary-size) - 1px);
  font-weight: 700;
  cursor: pointer;
}

.dashboard-channel-select select:focus-visible {
  border-color: var(--workspace-work-accent);
  outline: 2px solid color-mix(in srgb, var(--workspace-work-accent) 28%, transparent);
  outline-offset: 1px;
}

.dashboard-channel-select select:disabled {
  cursor: not-allowed;
  opacity: 0.62;
}

.dashboard-channel-card__body {
  display: flex;
  min-height: 220px;
  flex: 1 1 auto;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding-top: var(--workspace-space-4);
}

.dashboard-channel-card__loading {
  display: flex;
  min-height: 180px;
  align-items: center;
  justify-content: center;
}

.dashboard-channel-gauge {
  display: block;
  width: min(100%, 14rem);
  height: auto;
  overflow: visible;
}

.dashboard-channel-card__value {
  margin-top: -2px;
  color: var(--workspace-dashboard-text-strong);
  font-size: calc(var(--workspace-type-page-title-size) + 0.125rem);
  font-weight: 700;
  line-height: calc(var(--workspace-space-8) + 4px);
  font-variant-numeric: tabular-nums;
}

.dashboard-channel-card__details {
  min-height: 28px;
  margin-top: var(--workspace-space-4);
  padding: 0 24px;
  border: 1px solid var(--workspace-dashboard-card-border);
  border-radius: 12px;
  color: var(--workspace-dashboard-text-heading);
  background: var(--workspace-card-surface);
  font-size: calc(var(--workspace-type-secondary-size) - 1px);
  font-weight: 700;
  transition: border-color 140ms ease, color 140ms ease;
}

.dashboard-channel-card__details:hover:not(:disabled),
.dashboard-channel-card__details:focus-visible:not(:disabled) {
  border-color: var(--workspace-work-accent);
  color: var(--workspace-work-accent);
}

.dashboard-channel-card__details:focus-visible {
  outline: 2px solid var(--workspace-work-accent);
  outline-offset: 2px;
}

.dashboard-channel-card__details:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

@media (max-width: 1279px) {
  .dashboard-insights {
    grid-template-columns: minmax(0, 1fr);
  }
}

@media (max-width: 639px) {
  .dashboard-insights-card {
    padding: 18px;
  }

  .dashboard-model-table {
    min-width: 38rem;
    overflow-x: auto;
  }

  .dashboard-channel-card {
    min-height: 270px;
  }
}

:global(html.dark .dashboard-insights .dashboard-channel-select select) {
  background: var(--workspace-card-surface);
}

:global(html.dark .dashboard-insights .dashboard-model-row__icon--openai .model-icon) {
  filter: invert(1);
}
</style>
