<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Chart as ChartJS,
  CategoryScale,
  Filler,
  Legend,
  LineElement,
  LinearScale,
  PointElement,
  Title,
  Tooltip
} from 'chart.js'
import { Line } from 'vue-chartjs'
import type { OpsErrorTrendPoint } from '@/api/admin/ops'
import type { ChartState } from '../types'
import { formatHistoryLabel, sumNumbers } from '../utils/opsFormatters'
import HelpTooltip from '@/components/common/HelpTooltip.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import { getLuoxueClayChartTheme } from '@/utils/luoxueClayChartTheme'

ChartJS.register(Title, Tooltip, Legend, LineElement, LinearScale, PointElement, CategoryScale, Filler)

interface Props {
  points: OpsErrorTrendPoint[]
  loading: boolean
  timeRange: string
}

const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'openRequestErrors'): void
  (e: 'openUpstreamErrors'): void
}>()
const { t } = useI18n()

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
    danger: isDarkMode.value ? '#fb7185' : '#c2415b',
    dangerSoft: isDarkMode.value ? 'rgba(251, 113, 133, 0.12)' : 'rgba(220, 76, 100, 0.11)',
    warning: isDarkMode.value ? '#fbbf24' : '#b45309',
    warningSoft: isDarkMode.value ? 'rgba(251, 191, 36, 0.11)' : 'rgba(245, 158, 11, 0.12)',
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

const totalRequestErrors = computed(() => sumNumbers(props.points.map((p) => p.error_count_sla ?? 0)))

const totalUpstreamErrors = computed(() =>
  sumNumbers(
    props.points.map((p) => (p.upstream_error_count_excl_429_529 ?? 0) + (p.upstream_429_count ?? 0) + (p.upstream_529_count ?? 0))
  )
)

const totalDisplayed = computed(() =>
  sumNumbers(props.points.map((p) => (p.error_count_sla ?? 0) + (p.upstream_error_count_excl_429_529 ?? 0) + (p.business_limited_count ?? 0)))
)

const hasRequestErrors = computed(() => totalRequestErrors.value > 0)
const hasUpstreamErrors = computed(() => totalUpstreamErrors.value > 0)

const chartData = computed(() => {
  if (!props.points.length || totalDisplayed.value <= 0) return null
  return {
    labels: props.points.map((p) => formatHistoryLabel(p.bucket_start, props.timeRange)),
    datasets: [
      {
        label: t('admin.ops.errorsSla'),
        data: props.points.map((p) => p.error_count_sla ?? 0),
        borderColor: colors.value.danger,
        backgroundColor: colors.value.dangerSoft,
        fill: true,
        tension: 0.35,
        pointRadius: 0,
        pointHitRadius: 10
      },
      {
        label: t('admin.ops.upstreamExcl429529'),
        data: props.points.map((p) => p.upstream_error_count_excl_429_529 ?? 0),
        borderColor: colors.value.warning,
        backgroundColor: colors.value.warningSoft,
        fill: true,
        tension: 0.35,
        pointRadius: 0,
        pointHitRadius: 10
      },
      {
        label: t('admin.ops.businessLimited'),
        data: props.points.map((p) => p.business_limited_count ?? 0),
        borderColor: colors.value.neutral,
        backgroundColor: 'transparent',
        borderDash: [6, 6],
        fill: false,
        tension: 0.35,
        pointRadius: 0,
        pointHitRadius: 10
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
        displayColors: true
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
        ticks: { color: c.text, font: { size: 10 }, precision: 0 }
      }
    }
  }
})
</script>

<template>
  <section class="ops-chart-panel" aria-labelledby="ops-error-trend-title">
    <header class="ops-chart-header">
      <div class="ops-chart-title-group">
        <span class="ops-chart-icon ops-chart-icon--danger" aria-hidden="true">
          <svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M13 17h8m0 0V9m0 8l-8-8-4 4-6-6"
          />
          </svg>
        </span>
        <h3 id="ops-error-trend-title" class="ops-chart-title">{{ t('admin.ops.errorTrend') }}</h3>
        <HelpTooltip :content="t('admin.ops.tooltips.errorTrend')" trigger="click">
          <template #trigger>
            <button type="button" class="ops-chart-help" :aria-label="t('admin.ops.tooltips.errorTrend')">
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
      <div class="ops-chart-actions">
        <button
          type="button"
          class="ops-chart-action"
          :disabled="!hasRequestErrors"
          @click="emit('openRequestErrors')"
        >
          {{ t('admin.ops.errorDetails.requestErrors') }}
        </button>
        <button
          type="button"
          class="ops-chart-action"
          :disabled="!hasUpstreamErrors"
          @click="emit('openUpstreamErrors')"
        >
          {{ t('admin.ops.errorDetails.upstreamErrors') }}
        </button>
      </div>
    </header>

    <div class="ops-chart-plot">
      <Line v-if="state === 'ready' && chartData" :data="chartData" :options="options" />
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
          :description="t('admin.ops.charts.emptyError')"
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
  margin-bottom: 0.75rem;
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

.ops-chart-icon--danger {
  color: var(--lx-clay-danger);
  background: var(--lx-clay-danger-soft);
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

.ops-chart-action {
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
  text-align: center;
  transition: border-color 150ms ease, background-color 150ms ease, color 150ms ease;
}

.ops-chart-action:hover:not(:disabled) {
  border-color: var(--lx-clay-border-strong);
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-accent-soft);
}

.ops-chart-action:focus-visible {
  outline: 3px solid color-mix(in srgb, var(--lx-clay-accent) 34%, transparent);
  outline-offset: 2px;
}

.ops-chart-action:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

html.dark .ops-chart-action:hover:not(:disabled) {
  color: var(--lx-clay-accent);
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
  width: min(100%, 17rem);
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
  .ops-chart-help,
  .ops-chart-loading__line {
    animation: none;
    transition: none;
  }
}

@media (max-width: 640px) {
  .ops-chart-panel {
    min-height: 18rem;
    padding: 0.875rem;
  }

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
</style>
