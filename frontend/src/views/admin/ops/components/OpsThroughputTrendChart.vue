<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  BarElement,
  CategoryScale,
  Chart as ChartJS,
  Filler,
  Legend,
  LineElement,
  LinearScale,
  PointElement,
  Title,
  Tooltip,
} from 'chart.js'
import { Bar, Line } from 'vue-chartjs'
import type { ChartComponentRef } from 'vue-chartjs'
import type {
  OpsDashboardOverview,
  OpsThroughputGroupBreakdownItem,
  OpsThroughputPlatformBreakdownItem,
  OpsThroughputTrendPoint,
} from '@/api/admin/ops'
import EmptyState from '@/components/common/EmptyState.vue'
import Icon from '@/components/icons/Icon.vue'
import { formatCompactNumber } from '@/utils/format'
import { getLuoxueClayChartTheme } from '@/utils/luoxueClayChartTheme'
import type { ChartState } from '../types'
import { formatHistoryLabel, sumNumbers } from '../utils/opsFormatters'

ChartJS.register(
  Title,
  Tooltip,
  Legend,
  LineElement,
  BarElement,
  LinearScale,
  PointElement,
  CategoryScale,
  Filler,
)

type ThroughputView = 'trend' | 'distribution' | 'tokens'

interface Props {
  points: OpsThroughputTrendPoint[]
  loading: boolean
  timeRange: string
  overview?: OpsDashboardOverview | null
  byPlatform?: OpsThroughputPlatformBreakdownItem[]
  topGroups?: OpsThroughputGroupBreakdownItem[]
  fullscreen?: boolean
}

const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'selectPlatform', platform: string): void
  (e: 'selectGroup', groupId: number): void
  (e: 'openDetails'): void
}>()

const { t } = useI18n()
const activeView = ref<ThroughputView>('trend')
const throughputChartRef = ref<ChartComponentRef | null>(null)

const viewOptions = computed<Array<{ id: ThroughputView; label: string }>>(() => [
  { id: 'trend', label: t('admin.ops.charts.trendView') },
  { id: 'distribution', label: t('admin.ops.charts.distributionView') },
  { id: 'tokens', label: t('admin.ops.charts.tokenStatsView') },
])

watch(
  [() => props.timeRange, activeView],
  () => {
    setTimeout(() => {
      const chart: any = throughputChartRef.value?.chart
      if (chart && typeof chart.resetZoom === 'function') chart.resetZoom()
    }, 100)
  },
)

const themeRevision = ref(0)
let themeObserver: MutationObserver | null = null

const isDarkMode = computed(() => {
  themeRevision.value
  return document.documentElement.classList.contains('dark')
})

const colors = computed(() => {
  const theme = getLuoxueClayChartTheme(isDarkMode.value)
  return {
    ...theme,
    successSoft: isDarkMode.value ? 'rgba(52, 211, 153, 0.11)' : 'rgba(16, 185, 129, 0.11)',
    surface: isDarkMode.value ? '#251e2f' : '#ffffff',
    textStrong: isDarkMode.value ? '#f8f5fc' : '#332f3a',
  }
})

onMounted(() => {
  if (typeof MutationObserver === 'undefined') return
  themeObserver = new MutationObserver(() => {
    themeRevision.value += 1
  })
  themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
})

onBeforeUnmount(() => {
  themeObserver?.disconnect()
})

function finiteOrNull(value: number | null | undefined): number | null {
  return typeof value === 'number' && Number.isFinite(value) ? value : null
}

function average(values: Array<number | null | undefined>): number | null {
  const finiteValues = values.filter(
    (value): value is number => typeof value === 'number' && Number.isFinite(value),
  )
  if (!finiteValues.length) return null
  return sumNumbers(finiteValues) / finiteValues.length
}

function formatMetric(value: number | null, fractionDigits = 2): string {
  if (value == null) return '—'
  return new Intl.NumberFormat('en-US', {
    minimumFractionDigits: fractionDigits,
    maximumFractionDigits: fractionDigits,
  }).format(value)
}

function formatCompactMetric(value: number | null): string {
  if (value == null) return '—'
  return formatCompactNumber(value).replace('K', 'k')
}

const totalRequests = computed(() => sumNumbers(props.points.map((point) => point.request_count)))
const hasTrendData = computed(() => (
  props.points.length > 0
  && (
    totalRequests.value > 0
    || props.points.some((point) => (point.qps ?? 0) !== 0 || (point.tps ?? 0) !== 0)
  )
))

const averageQps = computed(() => (
  finiteOrNull(props.overview?.qps?.avg)
  ?? average(props.points.map((point) => point.qps))
))

const peakQps = computed(() => {
  const overviewPeak = finiteOrNull(props.overview?.qps?.peak)
  if (overviewPeak != null) return overviewPeak
  const values = props.points
    .map((point) => finiteOrNull(point.qps))
    .filter((value): value is number => value != null)
  return values.length ? Math.max(...values) : null
})

const currentTps = computed(() => {
  const overviewCurrent = finiteOrNull(props.overview?.tps?.current)
  if (overviewCurrent != null) return overviewCurrent
  for (let index = props.points.length - 1; index >= 0; index -= 1) {
    const value = finiteOrNull(props.points[index]?.tps)
    if (value != null) return value
  }
  return null
})

const totalTokens = computed(() => (
  finiteOrNull(props.overview?.token_consumed)
  ?? sumNumbers(props.points.map((point) => point.token_consumed))
))

const averageProgress = computed(() => {
  const averageValue = averageQps.value
  const peakValue = peakQps.value
  if (averageValue == null || peakValue == null || peakValue <= 0) return 0
  return Math.min(100, Math.max(0, (averageValue / peakValue) * 100))
})

const peakVsAverage = computed(() => {
  const averageValue = averageQps.value
  const peakValue = peakQps.value
  if (averageValue == null || peakValue == null || averageValue <= 0) return null
  return Math.max(0, ((peakValue - averageValue) / averageValue) * 100)
})

const peakComparisonLabel = computed(() => {
  if (peakVsAverage.value == null) return t('admin.ops.charts.noWindowBaseline')
  return t('admin.ops.charts.peakVsWindowAverage', {
    percent: peakVsAverage.value.toFixed(1),
  })
})

const tokenWindowLabel = computed(() => t('admin.ops.charts.windowTokenTotal', {
  count: formatCompactMetric(totalTokens.value),
}))

const trendChartData = computed(() => {
  if (!hasTrendData.value) return null
  return {
    labels: props.points.map((point) => formatHistoryLabel(point.bucket_start, props.timeRange)),
    datasets: [
      {
        label: 'QPS',
        data: props.points.map((point) => point.qps ?? 0),
        borderColor: colors.value.info,
        backgroundColor: colors.value.infoSoft,
        fill: true,
        tension: 0.4,
        pointRadius: 0,
        pointHitRadius: 10,
      },
      {
        label: t('admin.ops.tpsK'),
        data: props.points.map((point) => (point.tps ?? 0) / 1000),
        borderColor: colors.value.success,
        backgroundColor: colors.value.successSoft,
        fill: true,
        tension: 0.4,
        pointRadius: 0,
        pointHitRadius: 10,
        yAxisID: 'y1',
      },
    ],
  }
})

const tokenChartData = computed(() => {
  if (!hasTrendData.value) return null
  return {
    labels: props.points.map((point) => formatHistoryLabel(point.bucket_start, props.timeRange)),
    datasets: [
      {
        label: t('admin.ops.tpsK'),
        data: props.points.map((point) => (point.tps ?? 0) / 1000),
        borderColor: colors.value.success,
        backgroundColor: colors.value.successSoft,
        fill: true,
        tension: 0.4,
        pointRadius: 0,
        pointHitRadius: 10,
      },
    ],
  }
})

const distributionEntries = computed(() => {
  if ((props.topGroups?.length ?? 0) > 0) {
    return props.topGroups!.map((group) => ({
      id: `group-${group.group_id}`,
      label: group.group_name || `#${group.group_id}`,
      value: group.request_count,
      groupId: group.group_id,
      platform: null,
    }))
  }
  return (props.byPlatform ?? []).map((platform) => ({
    id: `platform-${platform.platform}`,
    label: platform.platform,
    value: platform.request_count,
    groupId: null,
    platform: platform.platform,
  }))
})

const distributionChartData = computed(() => {
  if (!distributionEntries.value.length) return null
  return {
    labels: distributionEntries.value.map((entry) => entry.label),
    datasets: [
      {
        label: t('admin.ops.requests'),
        data: distributionEntries.value.map((entry) => entry.value),
        borderRadius: 8,
        borderSkipped: false,
        backgroundColor: '#8b5cf6',
        maxBarThickness: 24,
      },
    ],
  }
})

const state = computed<ChartState>(() => {
  const hasData = activeView.value === 'distribution'
    ? Boolean(distributionChartData.value)
    : Boolean(trendChartData.value)
  if (hasData) return 'ready'
  if (props.loading) return 'loading'
  return 'empty'
})

const lineOptions = computed(() => {
  const color = colors.value
  return {
    responsive: true,
    maintainAspectRatio: false,
    interaction: { intersect: false, mode: 'index' as const },
    plugins: {
      legend: { display: false },
      tooltip: {
        backgroundColor: color.surface,
        titleColor: color.textStrong,
        bodyColor: color.text,
        borderColor: color.grid,
        borderWidth: 1,
        padding: 10,
        displayColors: true,
        callbacks: {
          label: (context: any) => {
            const label = context.dataset.label ? `${context.dataset.label}: ` : ''
            return `${label}${context.parsed.y.toFixed(1)}`
          },
        },
      },
      zoom: {
        pan: { enabled: true, mode: 'x' as const, modifierKey: 'ctrl' as const },
        zoom: { wheel: { enabled: true }, pinch: { enabled: true }, mode: 'x' as const },
      },
    },
    scales: {
      x: {
        type: 'category' as const,
        grid: { display: false },
        ticks: {
          color: color.text,
          font: { size: 10 },
          maxTicksLimit: 8,
          autoSkip: true,
          autoSkipPadding: 10,
        },
      },
      y: {
        type: 'linear' as const,
        display: true,
        position: 'left' as const,
        grid: { color: color.grid, borderDash: [4, 4] },
        ticks: { color: color.text, font: { size: 10 } },
      },
      y1: {
        type: 'linear' as const,
        display: activeView.value === 'trend',
        position: 'right' as const,
        grid: { display: false },
        ticks: { color: color.success, font: { size: 10 } },
      },
    },
  }
})

const distributionOptions = computed(() => {
  const color = colors.value
  return {
    responsive: true,
    maintainAspectRatio: false,
    indexAxis: 'y' as const,
    plugins: {
      legend: { display: false },
      tooltip: {
        backgroundColor: color.surface,
        titleColor: color.textStrong,
        bodyColor: color.text,
        borderColor: color.grid,
        borderWidth: 1,
      },
    },
    scales: {
      x: {
        beginAtZero: true,
        grid: { color: color.grid, borderDash: [4, 4] },
        ticks: { color: color.text, font: { size: 10 }, precision: 0 },
      },
      y: {
        grid: { display: false },
        ticks: { color: color.text, font: { size: 10, weight: 700 as const } },
      },
    },
  }
})

function selectView(view: ThroughputView): void {
  activeView.value = view
}

function handleViewKeydown(event: KeyboardEvent, index: number): void {
  let nextIndex = index
  if (event.key === 'ArrowRight') nextIndex = (index + 1) % viewOptions.value.length
  else if (event.key === 'ArrowLeft') nextIndex = (index - 1 + viewOptions.value.length) % viewOptions.value.length
  else if (event.key === 'Home') nextIndex = 0
  else if (event.key === 'End') nextIndex = viewOptions.value.length - 1
  else return

  event.preventDefault()
  const nextView = viewOptions.value[nextIndex]
  if (!nextView) return
  selectView(nextView.id)
  const tabs = (event.currentTarget as HTMLElement).parentElement?.querySelectorAll<HTMLButtonElement>('[role="tab"]')
  setTimeout(() => tabs?.[nextIndex]?.focus())
}

function selectDistributionEntry(entry: (typeof distributionEntries.value)[number]): void {
  if (entry.groupId != null) emit('selectGroup', entry.groupId)
  else if (entry.platform) emit('selectPlatform', entry.platform)
}

function csvCell(value: string | number): string {
  const text = String(value)
  return /[",\n]/.test(text) ? `"${text.replace(/"/g, '""')}"` : text
}

function exportData(): void {
  if (state.value !== 'ready') return

  let rows: Array<Array<string | number>>
  if (activeView.value === 'distribution') {
    rows = [
      ['name', 'request_count'],
      ...distributionEntries.value.map((entry) => [entry.label, entry.value]),
    ]
  } else {
    rows = [
      ['bucket_start', 'request_count', 'token_consumed', 'qps', 'tps'],
      ...props.points.map((point) => [
        point.bucket_start,
        point.request_count,
        point.token_consumed,
        point.qps,
        point.tps,
      ]),
    ]
  }

  const blob = new Blob([rows.map((row) => row.map(csvCell).join(',')).join('\n')], {
    type: 'text/csv;charset=utf-8',
  })
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = `ops-throughput-${activeView.value}-${new Date().toISOString().slice(0, 19).replace(/[:T]/g, '-')}.csv`
  anchor.click()
  URL.revokeObjectURL(url)
}
</script>

<template>
  <section
    class="ops-chart-panel ops-throughput-view"
    :data-view="activeView"
    aria-labelledby="ops-traffic-heading"
  >
    <header class="ops-chart-header ops-throughput-view__header">
      <div class="ops-throughput-view__title-stack">
        <h2 id="ops-traffic-heading" class="ops-chart-title">{{ t('admin.ops.throughputTrend') }}</h2>
        <p>{{ t('admin.ops.charts.throughputSubtitle') }}</p>
      </div>

      <div class="ops-throughput-view__tabs" role="tablist" :aria-label="t('admin.ops.charts.viewMode')">
        <button
          v-for="(view, index) in viewOptions"
          :id="`ops-throughput-tab-${view.id}`"
          :key="view.id"
          type="button"
          role="tab"
          :aria-selected="activeView === view.id"
          :aria-controls="`ops-throughput-panel-${view.id}`"
          :tabindex="activeView === view.id ? 0 : -1"
          :class="['ops-throughput-view__tab', { 'ops-throughput-view__tab--active': activeView === view.id }]"
          @click="selectView(view.id)"
          @keydown="handleViewKeydown($event, index)"
        >
          {{ view.label }}
        </button>
      </div>
    </header>

    <div class="ops-throughput-view__insights" data-testid="ops-throughput-insights">
      <article class="ops-throughput-metric" data-testid="ops-throughput-average">
        <span class="ops-throughput-metric__label">{{ t('admin.ops.charts.currentAverage') }}</span>
        <strong>{{ formatMetric(averageQps) }} <small>QPS</small></strong>
        <div class="ops-throughput-metric__progress" aria-hidden="true">
          <span :style="{ width: `${averageProgress}%` }"></span>
        </div>
      </article>

      <article class="ops-throughput-metric" data-testid="ops-throughput-peak">
        <span class="ops-throughput-metric__label">{{ t('admin.ops.charts.observedPeak') }}</span>
        <strong>{{ formatMetric(peakQps) }} <small>QPS</small></strong>
        <span class="ops-throughput-metric__positive">{{ peakComparisonLabel }}</span>
      </article>

      <article class="ops-throughput-metric" data-testid="ops-throughput-token-load">
        <span class="ops-throughput-metric__label">{{ t('admin.ops.charts.tokenLoad') }}</span>
        <strong>{{ formatCompactMetric(currentTps) }} <small>TPS</small></strong>
        <span class="ops-throughput-metric__hint">{{ tokenWindowLabel }}</span>
      </article>
    </div>

    <div
      :id="`ops-throughput-panel-${activeView}`"
      class="ops-throughput-view__chart-card"
      role="tabpanel"
      :aria-labelledby="`ops-throughput-tab-${activeView}`"
    >
      <div class="ops-throughput-view__legend" :aria-label="t('admin.ops.charts.periodLegend')">
        <span class="ops-throughput-view__legend-item">
          <i class="ops-throughput-view__legend-dot ops-throughput-view__legend-dot--current" aria-hidden="true"></i>
          {{ t('admin.ops.charts.currentPeriod') }}
        </span>
        <span class="ops-throughput-view__legend-item ops-throughput-view__legend-item--muted">
          <i class="ops-throughput-view__legend-dot ops-throughput-view__legend-dot--comparison" aria-hidden="true"></i>
          {{ t('admin.ops.charts.comparisonPeriod') }}
        </span>
      </div>

      <div class="ops-chart-plot">
        <Line
          v-if="activeView === 'trend' && state === 'ready' && trendChartData"
          ref="throughputChartRef"
          :data="trendChartData"
          :options="lineOptions"
        />
        <Bar
          v-else-if="activeView === 'distribution' && state === 'ready' && distributionChartData"
          ref="throughputChartRef"
          :data="distributionChartData"
          :options="distributionOptions"
        />
        <Line
          v-else-if="activeView === 'tokens' && state === 'ready' && tokenChartData"
          ref="throughputChartRef"
          :data="tokenChartData"
          :options="lineOptions"
        />
        <div v-else class="ops-chart-state">
          <div v-if="state === 'loading'" class="ops-chart-loading" role="status" aria-live="polite">
            <span class="sr-only">{{ t('common.loading') }}</span>
            <span class="ops-chart-loading__line ops-chart-loading__line--short"></span>
            <span class="ops-chart-loading__line"></span>
            <span class="ops-chart-loading__line ops-chart-loading__line--medium"></span>
          </div>
          <EmptyState
            v-else
            class="ops-chart-empty"
            :title="t('admin.ops.charts.throughputEmptyTitle')"
            description=""
          >
            <template #icon>
              <Icon name="chartNoAxesColumn" size="xl" aria-hidden="true" />
            </template>
            <template #action>
              <div class="ops-throughput-view__metric-tags" aria-hidden="true">
                <span>QPS</span>
                <span>TPS</span>
              </div>
            </template>
          </EmptyState>
        </div>
      </div>

      <footer class="ops-throughput-view__chart-footer">
        <div v-if="activeView === 'distribution' && distributionEntries.length" class="ops-chart-drilldowns">
          <button
            v-for="entry in distributionEntries"
            :key="entry.id"
            type="button"
            class="ops-chart-filter"
            @click="selectDistributionEntry(entry)"
          >
            <span class="ops-chart-filter__label">{{ entry.label }}</span>
            <span class="ops-chart-filter__value">{{ formatCompactMetric(entry.value) }}</span>
          </button>
        </div>
        <div v-else aria-hidden="true"></div>

        <div v-if="!props.fullscreen" class="ops-chart-actions">
          <button type="button" class="ops-chart-action ops-chart-action--primary" @click="emit('openDetails')">
            {{ t('admin.ops.charts.viewRequestDetails') }}
          </button>
          <button
            type="button"
            class="ops-chart-action"
            :disabled="state !== 'ready'"
            :title="t('admin.ops.charts.exportDataHint')"
            @click="exportData"
          >
            {{ t('admin.ops.charts.exportData') }}
          </button>
        </div>
      </footer>
    </div>
  </section>
</template>

<style scoped lang="css">
.ops-throughput-view {
  container: ops-throughput / inline-size;
  display: flex;
  height: auto;
  min-height: 0;
  flex-direction: column;
  gap: 24px;
  overflow: visible;
  padding: 0;
  border: 0;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
  font-family: var(--lx-clay-font-ui);
}

.ops-throughput-view__header {
  display: flex;
  min-height: 52px;
  align-items: flex-end;
  justify-content: space-between;
  flex-wrap: nowrap;
  gap: 16px;
  margin: 0;
}

.ops-throughput-view__title-stack {
  min-width: 0;
}

.ops-throughput-view__title-stack .ops-chart-title {
  margin: 0;
  color: #332f3a;
  font-family: var(--lx-clay-font-display);
  font-size: 24px;
  font-weight: 900;
  line-height: 32px;
  letter-spacing: -0.025em;
}

.ops-throughput-view__title-stack p {
  margin: 4px 0 0;
  color: #6b7280;
  font-size: 14px;
  line-height: 20px;
}

.ops-throughput-view__tabs {
  display: flex;
  min-width: max-content;
  flex: 0 0 auto;
  align-items: center;
  gap: 8px;
}

.ops-throughput-view__tab {
  display: inline-flex;
  height: 36px;
  align-items: center;
  justify-content: center;
  padding: 0 16px;
  border: 1px solid transparent;
  border-radius: 8px;
  color: #9ca3af;
  background: #f9fafb;
  font-size: 12px;
  font-weight: 700;
  line-height: 16px;
  white-space: nowrap;
  cursor: pointer;
  transition: color 160ms ease, background-color 160ms ease, border-color 160ms ease, box-shadow 160ms ease;
}

.ops-throughput-view__tab:hover {
  color: #6b7280;
  background: #f3f4f6;
}

.ops-throughput-view__tab--active,
.ops-throughput-view__tab--active:hover {
  border-color: #e5e7eb;
  color: #332f3a;
  background: #ffffff;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
}

.ops-throughput-view__tab:focus-visible,
.ops-chart-action:focus-visible,
.ops-chart-filter:focus-visible {
  outline: 2px solid #7c3aed;
  outline-offset: 2px;
}

.ops-throughput-view__insights {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
}

.ops-throughput-metric {
  display: flex;
  min-width: 0;
  min-height: 126px;
  flex-direction: column;
  justify-content: center;
  padding: 24px;
  border: 1px solid #f3f4f6;
  border-radius: 16px;
  background: #ffffff;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04);
}

.ops-throughput-metric__label {
  margin-bottom: 4px;
  overflow: hidden;
  color: #9ca3af;
  font-size: 11px;
  font-weight: 700;
  line-height: 16px;
  letter-spacing: 0.05em;
  text-overflow: ellipsis;
  text-transform: uppercase;
  white-space: nowrap;
}

.ops-throughput-metric strong {
  color: #1f2937;
  font-size: 30px;
  font-variant-numeric: tabular-nums;
  font-weight: 900;
  line-height: 36px;
  letter-spacing: -0.025em;
  white-space: nowrap;
}

.ops-throughput-metric strong small {
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0;
}

.ops-throughput-metric__progress {
  height: 6px;
  overflow: hidden;
  margin-top: 8px;
  border-radius: 999px;
  background: #f3f4f6;
}

.ops-throughput-metric__progress > span {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: #8b5cf6;
  transition: width 240ms ease;
}

.ops-throughput-metric__positive,
.ops-throughput-metric__hint {
  margin-top: 4px;
  overflow: hidden;
  font-size: 12px;
  line-height: 16px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ops-throughput-metric__positive {
  color: #059669;
  font-weight: 700;
}

.ops-throughput-metric__hint {
  color: #9ca3af;
  font-weight: 400;
}

.ops-throughput-view__chart-card {
  position: relative;
  display: flex;
  min-height: 440px;
  flex-direction: column;
  padding: 32px;
  border: 1px solid #f3f4f6;
  border-radius: 16px;
  background: #ffffff;
  box-shadow: 0 4px 14px rgba(70, 55, 96, 0.055);
}

.ops-throughput-view__legend {
  display: flex;
  min-height: 16px;
  flex: 0 0 auto;
  align-items: center;
  gap: 16px;
}

.ops-throughput-view__legend-item {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: #4b5563;
  font-size: 12px;
  font-weight: 700;
  line-height: 16px;
}

.ops-throughput-view__legend-item--muted {
  color: #9ca3af;
}

.ops-throughput-view__legend-dot {
  width: 12px;
  height: 12px;
  flex: 0 0 auto;
  border-radius: 999px;
}

.ops-throughput-view__legend-dot--current {
  background: #8b5cf6;
}

.ops-throughput-view__legend-dot--comparison {
  background: #e5e7eb;
}

.ops-chart-plot {
  min-height: 300px;
  flex: 1 1 auto;
  padding: 24px 0 16px;
}

.ops-chart-plot :deep(canvas) {
  width: 100% !important;
  height: 100% !important;
}

.ops-chart-state {
  display: flex;
  height: 100%;
  min-height: 300px;
  align-items: center;
  justify-content: center;
}

.ops-chart-loading {
  display: grid;
  width: min(100%, 22rem);
  gap: 10px;
}

.ops-chart-loading__line {
  display: block;
  height: 10px;
  border-radius: 999px;
  background: #f3f4f6;
  animation: ops-throughput-pulse 1.7s ease-in-out infinite;
}

.ops-chart-loading__line--short {
  width: 42%;
}

.ops-chart-loading__line--medium {
  width: 72%;
}

.ops-chart-empty :deep(.empty-state) {
  padding: 16px;
}

.ops-chart-empty :deep(.empty-state > div:first-child) {
  width: 60px;
  height: 60px;
  margin-bottom: 16px;
  color: #f3f4f6;
  background: transparent !important;
}

.ops-chart-empty :deep(.empty-state svg) {
  width: 60px;
  height: 60px;
  color: currentColor;
}

.ops-chart-empty :deep(.empty-state-title) {
  margin: 0;
  color: #d1d5db;
  font-size: 14px;
  font-weight: 500;
  line-height: 20px;
}

.ops-chart-empty :deep(.empty-state-description:empty) {
  display: none;
}

.ops-chart-empty :deep(.mt-6) {
  margin-top: 16px;
}

.ops-throughput-view__metric-tags {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.ops-throughput-view__metric-tags span {
  padding: 2px 8px;
  border: 1px solid #f3f4f6;
  border-radius: 4px;
  color: #d1d5db;
  font-size: 12px;
  line-height: 16px;
}

.ops-throughput-view__chart-footer {
  display: flex;
  min-height: 30px;
  flex: 0 0 auto;
  align-items: flex-end;
  justify-content: space-between;
  gap: 16px;
}

.ops-chart-actions {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 8px;
  margin-left: auto;
}

.ops-chart-action,
.ops-chart-filter {
  display: inline-flex;
  min-height: 30px;
  align-items: center;
  justify-content: center;
  padding: 6px 12px;
  border: 0;
  border-radius: 8px;
  color: #6b7280;
  background: #f9fafb;
  font-family: var(--lx-clay-font-ui);
  font-size: 11px;
  font-weight: 700;
  line-height: 16px;
  cursor: pointer;
  transition: color 160ms ease, background-color 160ms ease;
}

.ops-chart-action:hover:not(:disabled),
.ops-chart-filter:hover {
  color: #4b5563;
  background: #f3f4f6;
}

.ops-chart-action--primary,
.ops-chart-action--primary:hover {
  color: #7c3aed;
  background: #f5f3ff;
}

.ops-chart-action--primary:hover {
  background: #ede9fe;
}

.ops-chart-action:disabled {
  cursor: not-allowed;
  opacity: 0.58;
}

.ops-chart-drilldowns {
  display: flex;
  min-width: 0;
  flex: 1 1 auto;
  gap: 6px;
  overflow-x: auto;
  padding: 2px;
  scrollbar-width: none;
}

.ops-chart-drilldowns::-webkit-scrollbar {
  display: none;
}

.ops-chart-filter {
  flex: 0 0 auto;
  gap: 8px;
}

.ops-chart-filter__label {
  max-width: 180px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ops-chart-filter__value {
  color: #9ca3af;
  font-variant-numeric: tabular-nums;
}

@keyframes ops-throughput-pulse {
  0%,
  100% {
    opacity: 0.45;
  }
  50% {
    opacity: 1;
  }
}

@container ops-throughput (max-width: 560px) {
  .ops-throughput-view__header {
    align-items: flex-start;
    flex-direction: column;
  }

  .ops-throughput-view__tabs {
    width: 100%;
    min-width: 0;
  }

  .ops-throughput-view__tab {
    min-width: 0;
    flex: 1 1 0;
    padding-inline: 10px;
  }

  .ops-throughput-view__insights {
    grid-template-columns: minmax(0, 1fr);
  }

  .ops-throughput-view__chart-card {
    min-height: 390px;
    padding: 24px;
  }
}

@container ops-throughput (min-width: 561px) and (max-width: 640px) {
  .ops-throughput-view__title-stack .ops-chart-title {
    font-size: 22px;
    line-height: 30px;
  }

  .ops-throughput-view__title-stack p {
    font-size: 13px;
    line-height: 18px;
  }

  .ops-throughput-view__tabs {
    gap: 6px;
  }

  .ops-throughput-view__tab {
    padding-inline: 12px;
    font-size: 11px;
  }

  .ops-throughput-metric {
    padding: 20px;
  }

  .ops-throughput-metric strong {
    font-size: 28px;
    line-height: 34px;
  }

  .ops-throughput-view__chart-card {
    padding: 28px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .ops-throughput-view__tab,
  .ops-throughput-metric__progress > span,
  .ops-chart-action,
  .ops-chart-filter,
  .ops-chart-loading__line {
    animation: none;
    transition: none;
  }
}
</style>
