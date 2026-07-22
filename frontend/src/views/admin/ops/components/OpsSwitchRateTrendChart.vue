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
import type { OpsThroughputTrendPoint } from '@/api/admin/ops'
import type { ChartState } from '../types'
import { formatHistoryLabel, sumNumbers } from '../utils/opsFormatters'
import HelpTooltip from '@/components/common/HelpTooltip.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import { getLuoxueClayChartTheme } from '@/utils/luoxueClayChartTheme'

ChartJS.register(Title, Tooltip, Legend, LineElement, LinearScale, PointElement, CategoryScale, Filler)

interface Props {
  points: OpsThroughputTrendPoint[]
  loading: boolean
  timeRange: string
  fullscreen?: boolean
}

const props = defineProps<Props>()
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
        label: t('admin.ops.switchRate'),
        data: props.points.map((p) => {
          const requests = p.request_count ?? 0
          const switches = p.switch_count ?? 0
          if (requests <= 0) return 0
          return switches / requests
        }),
        borderColor: colors.value.info,
        backgroundColor: colors.value.infoSoft,
        fill: true,
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
        displayColors: true,
        callbacks: {
          label: (context: any) => {
            const value = typeof context?.parsed?.y === 'number' ? context.parsed.y : 0
            return `${t('admin.ops.switchRate')}: ${value.toFixed(3)}`
          }
        }
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
        ticks: {
          color: c.text,
          font: { size: 10 },
          callback: (value: any) => Number(value).toFixed(3)
        }
      }
    }
  }
})
</script>

<template>
  <section class="ops-chart-panel" aria-labelledby="ops-switch-rate-title">
    <header class="ops-chart-header">
      <div class="ops-chart-title-group">
        <span class="ops-chart-icon ops-chart-icon--info" aria-hidden="true">
          <svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 7h10M7 12h6m-6 5h3" />
          </svg>
        </span>
        <h3 id="ops-switch-rate-title" class="ops-chart-title">{{ t('admin.ops.switchRateTrend') }}</h3>
        <HelpTooltip
          v-if="!props.fullscreen"
          :content="t('admin.ops.tooltips.switchRateTrend')"
          trigger="click"
        >
          <template #trigger>
            <button
              type="button"
              class="ops-chart-help"
              :aria-label="t('admin.ops.tooltips.switchRateTrend')"
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
  gap: 0.75rem;
  margin-bottom: 0.75rem;
}

.ops-chart-title-group {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 0.625rem;
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
}
</style>
