<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Chart as ChartJS, CategoryScale, Filler, Legend, LineElement, LinearScale, PointElement, Title, Tooltip } from 'chart.js'
import { Line } from 'vue-chartjs'
import type { ChartComponentRef } from 'vue-chartjs'
import type { OpsThroughputGroupBreakdownItem, OpsThroughputPlatformBreakdownItem, OpsThroughputTrendPoint } from '@/api/admin/ops'
import type { ChartState } from '../types'
import { formatHistoryLabel, sumNumbers } from '../utils/opsFormatters'
import HelpTooltip from '@/components/common/HelpTooltip.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import { formatNumber } from '@/utils/format'
import { getLuoxueClayChartTheme } from '@/utils/luoxueClayChartTheme'

ChartJS.register(Title, Tooltip, Legend, LineElement, LinearScale, PointElement, CategoryScale, Filler)

interface Props {
  points: OpsThroughputTrendPoint[]
  loading: boolean
  timeRange: string
  byPlatform?: OpsThroughputPlatformBreakdownItem[]
  topGroups?: OpsThroughputGroupBreakdownItem[]
  fullscreen?: boolean
}

const props = defineProps<Props>()
const { t } = useI18n()
const emit = defineEmits<{
  (e: 'selectPlatform', platform: string): void
  (e: 'selectGroup', groupId: number): void
  (e: 'openDetails'): void
}>()

const throughputChartRef = ref<ChartComponentRef | null>(null)
watch(
  () => props.timeRange,
  () => {
    setTimeout(() => {
      const chart: any = throughputChartRef.value?.chart
      if (chart && typeof chart.resetZoom === 'function') {
        chart.resetZoom()
      }
    }, 100)
  }
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
    textStrong: isDarkMode.value ? '#f8f5fc' : '#332f3a'
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

const totalRequests = computed(() => sumNumbers(props.points.map((p) => p.request_count)))

const chartData = computed(() => {
  if (!props.points.length || totalRequests.value <= 0) return null
  return {
    labels: props.points.map((p) => formatHistoryLabel(p.bucket_start, props.timeRange)),
    datasets: [
      {
        label: 'QPS',
        data: props.points.map((p) => p.qps ?? 0),
        borderColor: colors.value.info,
        backgroundColor: colors.value.infoSoft,
        fill: true,
        tension: 0.4,
        pointRadius: 0,
        pointHitRadius: 10
      },
      {
        label: t('admin.ops.tpsK'),
        data: props.points.map((p) => (p.tps ?? 0) / 1000),
        borderColor: colors.value.success,
        backgroundColor: colors.value.successSoft,
        fill: true,
        tension: 0.4,
        pointRadius: 0,
        pointHitRadius: 10,
        yAxisID: 'y1'
      }
    ]
  }
})

const state = computed<ChartState>(() => {
  if (chartData.value) return 'ready'
  if (props.loading) return 'loading'
  return 'empty'
})

const options = computed(() => {
  const c = colors.value
  return {
    responsive: true,
    maintainAspectRatio: false,
    interaction: { intersect: false, mode: 'index' as const },
    plugins: {
      legend: {
        position: 'top' as const,
        align: 'end' as const,
        labels: { color: c.text, usePointStyle: true, boxWidth: 6, padding: 12, font: { size: 10 } }
      },
      tooltip: {
        backgroundColor: c.surface,
        titleColor: c.textStrong,
        bodyColor: c.text,
        borderColor: c.grid,
        borderWidth: 1,
        padding: 10,
        displayColors: true,
        callbacks: {
          label: (context: any) => {
            let label = context.dataset.label || ''
            if (label) label += ': '
            if (context.raw !== null) label += context.parsed.y.toFixed(1)
            return label
          }
        }
      },
      // Optional: if chartjs-plugin-zoom is installed, these options will enable zoom/pan.
      zoom: {
        pan: { enabled: true, mode: 'x' as const, modifierKey: 'ctrl' as const },
        zoom: { wheel: { enabled: true }, pinch: { enabled: true }, mode: 'x' as const }
      }
    },
    scales: {
      x: {
        type: 'category' as const,
        grid: { display: false },
        ticks: {
          color: c.text,
          font: { size: 10 },
          maxTicksLimit: 8,
          autoSkip: true,
          autoSkipPadding: 10
        }
      },
      y: {
        type: 'linear' as const,
        display: true,
        position: 'left' as const,
        grid: { color: c.grid, borderDash: [4, 4] },
        ticks: { color: c.text, font: { size: 10 } }
      },
      y1: {
        type: 'linear' as const,
        display: true,
        position: 'right' as const,
        grid: { display: false },
        ticks: { color: c.success, font: { size: 10 } }
      }
    }
  }
})

function resetZoom() {
  const chart: any = throughputChartRef.value?.chart
  if (chart && typeof chart.resetZoom === 'function') chart.resetZoom()
}

function downloadChart() {
  const chart: any = throughputChartRef.value?.chart
  if (!chart || typeof chart.toBase64Image !== 'function') return
  const url = chart.toBase64Image('image/png', 1)
  const a = document.createElement('a')
  a.href = url
  a.download = `ops-throughput-${new Date().toISOString().slice(0, 19).replace(/[:T]/g, '-')}.png`
  a.click()
}
</script>

<template>
  <section class="ops-chart-panel" aria-labelledby="ops-throughput-trend-title">
    <header class="ops-chart-header">
      <div class="ops-chart-title-group">
        <span class="ops-chart-icon ops-chart-icon--info" aria-hidden="true">
          <svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6" />
          </svg>
        </span>
        <h3 id="ops-throughput-trend-title" class="ops-chart-title">{{ t('admin.ops.throughputTrend') }}</h3>
        <HelpTooltip
          v-if="!props.fullscreen"
          :content="t('admin.ops.tooltips.throughputTrend')"
          trigger="click"
        >
          <template #trigger>
            <button
              type="button"
              class="ops-chart-help"
              :aria-label="t('admin.ops.tooltips.throughputTrend')"
            >
              <svg aria-hidden="true" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M12 17h.01M12 14a3 3 0 10-3-3m3-6a8 8 0 110 16"
                />
              </svg>
            </button>
          </template>
        </HelpTooltip>
      </div>
      <div v-if="!props.fullscreen" class="ops-chart-actions">
          <button
            type="button"
            class="ops-chart-action"
            :disabled="state !== 'ready'"
            :title="t('admin.ops.requestDetails.title')"
            @click="emit('openDetails')"
          >
            {{ t('admin.ops.requestDetails.details') }}
          </button>
          <button
            type="button"
            class="ops-chart-action"
            :disabled="state !== 'ready'"
            :title="t('admin.ops.charts.resetZoomHint')"
            @click="resetZoom"
          >
            {{ t('admin.ops.charts.resetZoom') }}
          </button>
          <button
            type="button"
            class="ops-chart-action"
            :disabled="state !== 'ready'"
            :title="t('admin.ops.charts.downloadChartHint')"
            @click="downloadChart"
          >
            {{ t('admin.ops.charts.downloadChart') }}
          </button>
      </div>
    </header>

    <!-- Drilldown chips (baseline interaction: click to set global filter) -->
    <div v-if="(props.topGroups?.length ?? 0) > 0" class="ops-chart-drilldowns" :aria-label="t('admin.ops.throughputTrend')">
      <button
        v-for="g in props.topGroups"
        :key="g.group_id"
        type="button"
        class="ops-chart-filter"
        @click="emit('selectGroup', g.group_id)"
      >
        <span class="ops-chart-filter__label">{{ g.group_name || `#${g.group_id}` }}</span>
        <span class="ops-chart-filter__value">{{ formatNumber(g.request_count) }}</span>
      </button>
    </div>

    <div v-else-if="(props.byPlatform?.length ?? 0) > 0" class="ops-chart-drilldowns" :aria-label="t('admin.ops.throughputTrend')">
      <button
        v-for="p in props.byPlatform"
        :key="p.platform"
        type="button"
        class="ops-chart-filter"
        @click="emit('selectPlatform', p.platform)"
      >
        <span class="ops-chart-filter__label uppercase">{{ p.platform }}</span>
        <span class="ops-chart-filter__value">{{ formatNumber(p.request_count) }}</span>
      </button>
    </div>

    <div class="ops-chart-plot">
      <Line v-if="state === 'ready' && chartData" ref="throughputChartRef" :data="chartData" :options="options" />
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
          :title="t('common.noData')"
          :description="t('admin.ops.charts.emptyRequest')"
        />
      </div>
    </div>
  </section>
</template>

<style scoped>
.ops-chart-panel {
  display: flex;
  min-height: 20rem;
  height: 100%;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-ops);
  background: var(--lx-clay-surface);
  box-shadow: none;
  padding: 1rem;
}

.ops-chart-header {
  display: flex;
  min-height: 2.75rem;
  flex: 0 0 auto;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 0.75rem;
  margin-bottom: 0.625rem;
}

.ops-chart-title-group,
.ops-chart-actions {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 0.625rem;
}

.ops-chart-title-group {
  flex: 1 1 12rem;
}

.ops-chart-actions {
  flex: 0 0 auto;
  gap: 0.375rem;
  margin-left: auto;
}

.ops-chart-icon {
  display: inline-flex;
  width: 2rem;
  height: 2rem;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  border-radius: 0.625rem;
}

.ops-chart-icon svg {
  width: 1rem;
  height: 1rem;
}

.ops-chart-icon--info {
  color: var(--lx-clay-info-deep);
  background: var(--lx-clay-info-soft);
}

.ops-chart-title {
  min-width: 0;
  color: var(--lx-clay-text);
  font-family: var(--lx-clay-font-ui);
  font-size: 0.875rem;
  font-weight: 800;
  line-height: 1.25;
  text-wrap: balance;
}

.ops-chart-help {
  display: inline-flex;
  width: 2.75rem;
  height: 2.75rem;
  align-items: center;
  justify-content: center;
  border-radius: 0.6875rem;
  color: var(--lx-clay-text-muted);
  background: transparent;
  transition: background-color 150ms ease, color 150ms ease;
}

.ops-chart-help svg {
  width: 1rem;
  height: 1rem;
}

.ops-chart-help:hover {
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-accent-soft);
}

.ops-chart-help:focus-visible {
  outline: 3px solid color-mix(in srgb, var(--lx-clay-accent) 34%, transparent);
  outline-offset: 2px;
}

html.dark .ops-chart-help:hover {
  color: var(--lx-clay-accent);
}

.ops-chart-action,
.ops-chart-filter {
  display: inline-flex;
  min-height: 2.75rem;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--lx-clay-border);
  border-radius: 0.6875rem;
  padding: 0.5rem 0.625rem;
  color: var(--lx-clay-text-secondary);
  background: var(--lx-clay-surface-soft);
  font-family: var(--lx-clay-font-ui);
  font-size: 0.6875rem;
  font-weight: 750;
  line-height: 1.15;
  transition: border-color 150ms ease, background-color 150ms ease, color 150ms ease;
}

.ops-chart-action:hover:not(:disabled),
.ops-chart-filter:hover {
  border-color: var(--lx-clay-border-strong);
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-accent-soft);
}

.ops-chart-action:focus-visible,
.ops-chart-filter:focus-visible {
  outline: 3px solid color-mix(in srgb, var(--lx-clay-accent) 34%, transparent);
  outline-offset: 2px;
}

.ops-chart-action:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

html.dark .ops-chart-action:hover:not(:disabled),
html.dark .ops-chart-filter:hover {
  color: var(--lx-clay-accent);
}

.ops-chart-drilldowns {
  display: flex;
  flex: 0 0 auto;
  gap: 0.375rem;
  overflow-x: auto;
  margin: 0 -0.375rem 0.25rem;
  padding: 0.375rem;
  scrollbar-width: thin;
  scrollbar-color: var(--lx-clay-border-strong) transparent;
}

.ops-chart-filter {
  flex: 0 0 auto;
  gap: 0.5rem;
  padding-inline: 0.75rem;
}

.ops-chart-filter__label {
  max-width: 11.25rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ops-chart-filter__value {
  color: var(--lx-clay-text-muted);
  font-variant-numeric: tabular-nums;
}

.ops-chart-plot {
  min-height: 0;
  flex: 1;
}

.ops-chart-state {
  display: flex;
  height: 100%;
  min-height: 13rem;
  align-items: center;
  justify-content: center;
}

.ops-chart-loading {
  display: grid;
  width: min(100%, 22rem);
  gap: 0.625rem;
}

.ops-chart-loading__line {
  display: block;
  height: 0.625rem;
  border-radius: 999px;
  background: var(--lx-clay-recessed-strong);
  animation: ops-chart-pulse 1.7s ease-in-out infinite;
}

.ops-chart-loading__line--short {
  width: 42%;
}

.ops-chart-loading__line--medium {
  width: 72%;
}

.ops-chart-empty :deep(.empty-state) {
  padding: 1rem;
}

.ops-chart-empty :deep(.empty-state > div:first-child) {
  width: 2.75rem;
  height: 2.75rem;
  margin-bottom: 0.625rem;
  border-radius: 0.75rem;
  background: var(--lx-clay-recessed);
}

.ops-chart-empty :deep(.empty-state-icon) {
  width: 1.375rem;
  height: 1.375rem;
  margin: 0;
  color: var(--lx-clay-text-muted);
}

.ops-chart-empty :deep(.empty-state-title) {
  margin-bottom: 0.25rem;
  font-size: 0.875rem;
  font-weight: 750;
}

.ops-chart-empty :deep(.empty-state-description) {
  color: var(--lx-clay-text-muted);
  font-size: 0.75rem;
  line-height: 1.5;
}

@keyframes ops-chart-pulse {
  0%,
  100% {
    opacity: 0.45;
  }
  50% {
    opacity: 1;
  }
}

@media (prefers-reduced-motion: reduce) {
  .ops-chart-action,
  .ops-chart-filter,
  .ops-chart-help,
  .ops-chart-loading__line {
    animation: none;
    transition: none;
  }
}

@media (max-width: 760px) {
  .ops-chart-header {
    align-items: flex-start;
  }

  .ops-chart-actions {
    width: 100%;
  }

  .ops-chart-action {
    flex: 1 1 0;
  }
}

@media (max-width: 640px) {
  .ops-chart-panel {
    min-height: 18rem;
    padding: 0.875rem;
  }
}
</style>
