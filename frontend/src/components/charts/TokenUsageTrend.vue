<template>
  <div
    class="card h-full p-5 md:p-6"
    :class="{ 'token-trend--home-clay': isHomeClay }"
    :aria-busy="loading"
  >
    <div v-if="isHomeClay" class="token-trend-header">
      <div>
        <span class="token-trend-eyebrow">{{ t('admin.dashboard.usageTrendEyebrow') }}</span>
        <h3 class="token-trend-title">
          {{ t('admin.dashboard.tokenUsageTrend') }}
        </h3>
      </div>
      <div
        class="token-trend-switch"
        role="group"
        :aria-label="t('admin.dashboard.tokenUsageTrend')"
      >
        <button
          type="button"
          :class="{ active: activeMetric === 'tokens' }"
          :aria-pressed="activeMetric === 'tokens'"
          @click="activeMetric = 'tokens'"
        >
          {{ t('admin.dashboard.tokens') }}
        </button>
        <button
          type="button"
          :class="{ active: activeMetric === 'cost' }"
          :aria-pressed="activeMetric === 'cost'"
          @click="activeMetric = 'cost'"
        >
          {{ t('admin.dashboard.spendShort') }}
        </button>
      </div>
    </div>
    <h3 v-else class="mb-5 text-base font-semibold text-gray-900 dark:text-white">
      {{ t('admin.dashboard.tokenUsageTrend') }}
    </h3>

    <div
      v-if="loading"
      class="token-trend-state"
      :class="isHomeClay ? 'token-trend-loading' : 'h-48'"
      role="status"
      aria-live="polite"
    >
      <template v-if="isHomeClay">
        <span class="sr-only">{{ t('common.loading') }}</span>
        <div class="token-trend-skeleton-summary" aria-hidden="true"></div>
        <div class="token-trend-skeleton-chart" aria-hidden="true">
          <i v-for="index in 4" :key="index"></i>
        </div>
      </template>
      <LoadingSpinner v-else />
    </div>

    <div
      v-else-if="error"
      class="token-trend-state token-trend-error"
      role="alert"
      data-testid="token-trend-error"
    >
      <span class="token-trend-state-icon" aria-hidden="true"><Icon name="exclamationTriangle" size="lg" /></span>
      <strong>{{ t('admin.dashboard.failedToLoad') }}</strong>
      <p v-if="error !== t('admin.dashboard.failedToLoad')">{{ error }}</p>
      <button type="button" @click="emit('retry')">
        {{ t('admin.dashboard.retry') }}
      </button>
    </div>

    <div v-else-if="hasTrendData && chartData" class="token-trend-content">
      <template v-if="isHomeClay">
        <div class="token-trend-summary">
          <div class="token-trend-total">
            <span>{{ summaryLabel }}</span>
            <strong data-testid="token-trend-total">{{ summaryValue }}</strong>
          </div>
          <div class="token-trend-legend" aria-hidden="true">
            <template v-if="activeMetric === 'tokens'">
              <span><i class="input"></i>{{ t('admin.dashboard.input') }}</span>
              <span><i class="output"></i>{{ t('admin.dashboard.output') }}</span>
              <span><i class="cache"></i>{{ t('admin.dashboard.cache') }}</span>
            </template>
            <template v-else>
              <span><i class="actual"></i>{{ t('admin.dashboard.actual') }}</span>
              <span><i class="standard"></i>{{ t('admin.dashboard.standard') }}</span>
            </template>
          </div>
        </div>
        <div
          class="token-trend-chart token-trend-chart--home"
          role="img"
          :aria-label="chartAriaLabel"
        >
          <Line
            :key="`${activeMetric}-${themeRevision}`"
            :data="chartData"
            :options="lineOptions"
          />
        </div>

        <table class="sr-only">
          <caption>{{ t('admin.dashboard.tokenUsageTrend') }}</caption>
          <thead>
            <tr>
              <th scope="col">{{ t('usage.time') }}</th>
              <template v-if="activeMetric === 'tokens'">
                <th scope="col">{{ t('admin.dashboard.input') }}</th>
                <th scope="col">{{ t('admin.dashboard.output') }}</th>
                <th scope="col">{{ t('admin.dashboard.cache') }}</th>
              </template>
              <template v-else>
                <th scope="col">{{ t('admin.dashboard.actual') }}</th>
                <th scope="col">{{ t('admin.dashboard.standard') }}</th>
              </template>
            </tr>
          </thead>
          <tbody>
            <tr v-for="point in normalizedTrendData" :key="point.date">
              <th scope="row">{{ point.date }}</th>
              <template v-if="activeMetric === 'tokens'">
                <td>{{ point.inputTokens }}</td>
                <td>{{ point.outputTokens }}</td>
                <td>{{ point.cacheTokens }}</td>
              </template>
              <template v-else>
                <td>${{ formatCostTotal(point.actualCost) }}</td>
                <td>${{ formatCostTotal(point.standardCost) }}</td>
              </template>
            </tr>
          </tbody>
        </table>
      </template>

      <div v-else class="token-trend-chart h-48">
        <Line :key="themeRevision" :data="chartData" :options="lineOptions" />
      </div>
    </div>

    <div
      v-else
      class="token-trend-state token-trend-empty"
      :class="isHomeClay ? '' : 'h-48'"
      role="status"
      data-testid="token-trend-empty"
    >
      <span v-if="isHomeClay" class="token-trend-empty-mark" aria-hidden="true"><Icon name="chartNoAxesColumn" size="lg" /></span>
      <strong>{{ t('admin.dashboard.noDataAvailable') }}</strong>
      <p v-if="isHomeClay">{{ t('admin.dashboard.startUsingApi') }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler
} from 'chart.js'
import { Line } from 'vue-chartjs'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'
import type { TrendDataPoint } from '@/types'
import { getLuoxueClayChartTheme } from '@/utils/luoxueClayChartTheme'

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler
)

type TrendMetric = 'tokens' | 'cost'

interface NormalizedTrendPoint {
  date: string
  inputTokens: number
  outputTokens: number
  cacheCreationTokens: number
  cacheReadTokens: number
  cacheTokens: number
  totalTokens: number
  actualCost: number
  standardCost: number
}

const { t } = useI18n()

const props = withDefaults(defineProps<{
  trendData: TrendDataPoint[]
  loading?: boolean
  error?: string | null
  variant?: 'default' | 'home-clay'
}>(), {
  loading: false,
  error: null,
  variant: 'default'
})

const emit = defineEmits<{
  retry: []
}>()

const activeMetric = ref<TrendMetric>('tokens')
const themeRevision = ref(0)
let themeObserver: MutationObserver | null = null

const isHomeClay = computed(() => props.variant === 'home-clay')

const isDarkMode = computed(() => {
  themeRevision.value
  return document.documentElement.classList.contains('dark')
})

const chartColors = computed(() => isHomeClay.value
  ? (() => {
      const theme = getLuoxueClayChartTheme(isDarkMode.value)
      return {
        text: theme.text,
        grid: theme.grid,
        input: theme.primary,
        output: theme.info,
        cacheCreation: theme.primarySoft,
        cacheRead: theme.success,
        cache: theme.success,
        cacheHitRate: theme.neutral,
        actual: theme.primary,
        standard: theme.neutral
      }
    })()
  : {
      text: isDarkMode.value ? '#e5e7eb' : '#374151',
      grid: isDarkMode.value ? '#374151' : '#e5e7eb',
      input: '#3b82f6',
      output: '#10b981',
      cacheCreation: '#f59e0b',
      cacheRead: '#0b8bed',
      cache: '#0b8bed',
      cacheHitRate: '#8b5cf6',
      actual: '#3b82f6',
      standard: isDarkMode.value ? '#9ca3af' : '#6b7280'
    })

const toFiniteNumber = (value: unknown): number => {
  const normalized = Number(value)
  return Number.isFinite(normalized) ? normalized : 0
}

const normalizedTrendData = computed<NormalizedTrendPoint[]>(() => props.trendData.map((point) => {
  const inputTokens = toFiniteNumber(point.input_tokens)
  const outputTokens = toFiniteNumber(point.output_tokens)
  const cacheCreationTokens = toFiniteNumber(point.cache_creation_tokens)
  const cacheReadTokens = toFiniteNumber(point.cache_read_tokens)
  const derivedTotalTokens = inputTokens + outputTokens + cacheCreationTokens + cacheReadTokens
  const totalTokens = typeof point.total_tokens === 'number' && Number.isFinite(point.total_tokens)
    ? point.total_tokens
    : derivedTotalTokens

  return {
    date: point.date,
    inputTokens,
    outputTokens,
    cacheCreationTokens,
    cacheReadTokens,
    cacheTokens: cacheCreationTokens + cacheReadTokens,
    totalTokens,
    actualCost: toFiniteNumber(point.actual_cost),
    standardCost: toFiniteNumber(point.cost)
  }
}))

const hasTrendData = computed(() => normalizedTrendData.value.length > 0)

const totalTokens = computed(() => normalizedTrendData.value.reduce(
  (total, point) => total + point.totalTokens,
  0
))

const totalActualCost = computed(() => normalizedTrendData.value.reduce(
  (total, point) => total + point.actualCost,
  0
))

const summaryLabel = computed(() => {
  if (isHomeClay.value) {
    return activeMetric.value === 'tokens'
      ? t('admin.dashboard.currentRangeTotal')
      : t('admin.dashboard.currentRangeCost')
  }

  return activeMetric.value === 'tokens'
    ? t('admin.dashboard.totalTokens')
    : t('admin.dashboard.totalCost')
})

const summaryValue = computed(() => activeMetric.value === 'tokens'
  ? formatTokens(totalTokens.value)
  : `$${formatCostTotal(totalActualCost.value)}`)

const chartAriaLabel = computed(() => (
  `${t('admin.dashboard.tokenUsageTrend')}: ${summaryLabel.value} ${summaryValue.value}`
))

const homeTokenChartData = computed(() => ({
  labels: normalizedTrendData.value.map((point) => point.date),
  datasets: [
    {
      label: t('admin.dashboard.input'),
      data: normalizedTrendData.value.map((point) => point.inputTokens),
      borderColor: chartColors.value.input,
      backgroundColor: `${chartColors.value.input}20`,
      fill: true,
      tension: 0.36,
      borderWidth: 2.4,
      pointRadius: 0,
      pointHoverRadius: 4
    },
    {
      label: t('admin.dashboard.output'),
      data: normalizedTrendData.value.map((point) => point.outputTokens),
      borderColor: chartColors.value.output,
      backgroundColor: `${chartColors.value.output}16`,
      fill: false,
      tension: 0.36,
      borderWidth: 2.4,
      pointRadius: 0,
      pointHoverRadius: 4
    },
    {
      label: t('admin.dashboard.cache'),
      data: normalizedTrendData.value.map((point) => point.cacheTokens),
      borderColor: chartColors.value.cache,
      backgroundColor: `${chartColors.value.cache}16`,
      fill: false,
      tension: 0.36,
      borderWidth: 2.4,
      pointRadius: 0,
      pointHoverRadius: 4
    }
  ]
}))

const homeCostChartData = computed(() => ({
  labels: normalizedTrendData.value.map((point) => point.date),
  datasets: [
    {
      label: t('admin.dashboard.actual'),
      data: normalizedTrendData.value.map((point) => point.actualCost),
      borderColor: chartColors.value.actual,
      backgroundColor: `${chartColors.value.actual}20`,
      fill: true,
      tension: 0.36,
      borderWidth: 2.4,
      pointRadius: 0,
      pointHoverRadius: 4
    },
    {
      label: t('admin.dashboard.standard'),
      data: normalizedTrendData.value.map((point) => point.standardCost),
      borderColor: chartColors.value.standard,
      backgroundColor: `${chartColors.value.standard}12`,
      borderDash: [5, 5],
      fill: false,
      tension: 0.36,
      borderWidth: 2,
      pointRadius: 0,
      pointHoverRadius: 4
    }
  ]
}))

const defaultChartData = computed(() => ({
  labels: normalizedTrendData.value.map((point) => point.date),
  datasets: [
    {
      label: 'Input',
      data: normalizedTrendData.value.map((point) => point.inputTokens),
      borderColor: chartColors.value.input,
      backgroundColor: `${chartColors.value.input}20`,
      fill: true,
      tension: 0.3
    },
    {
      label: 'Output',
      data: normalizedTrendData.value.map((point) => point.outputTokens),
      borderColor: chartColors.value.output,
      backgroundColor: `${chartColors.value.output}20`,
      fill: true,
      tension: 0.3
    },
    {
      label: 'Cache Creation',
      data: normalizedTrendData.value.map((point) => point.cacheCreationTokens),
      borderColor: chartColors.value.cacheCreation,
      backgroundColor: `${chartColors.value.cacheCreation}20`,
      fill: true,
      tension: 0.3
    },
    {
      label: 'Cache Read',
      data: normalizedTrendData.value.map((point) => point.cacheReadTokens),
      borderColor: chartColors.value.cacheRead,
      backgroundColor: `${chartColors.value.cacheRead}20`,
      fill: true,
      tension: 0.3
    },
    {
      label: 'Cache Hit Rate',
      data: normalizedTrendData.value.map((point) => {
        const totalPromptTokens = point.inputTokens + point.cacheReadTokens + point.cacheCreationTokens
        return totalPromptTokens > 0 ? (point.cacheReadTokens / totalPromptTokens) * 100 : 0
      }),
      borderColor: chartColors.value.cacheHitRate,
      backgroundColor: `${chartColors.value.cacheHitRate}20`,
      borderDash: [5, 5],
      fill: false,
      tension: 0.3,
      yAxisID: 'yPercent'
    }
  ]
}))

const chartData = computed(() => {
  if (!hasTrendData.value) return null
  if (!isHomeClay.value) return defaultChartData.value
  return activeMetric.value === 'tokens' ? homeTokenChartData.value : homeCostChartData.value
})

const lineOptions = computed(() => {
  const isCostMetric = isHomeClay.value && activeMetric.value === 'cost'

  if (isHomeClay.value) {
    return {
      responsive: true,
      maintainAspectRatio: false,
      animation: false as const,
      interaction: {
        intersect: false,
        mode: 'index' as const
      },
      plugins: {
        legend: {
          display: false
        },
        tooltip: {
          displayColors: true,
          itemSort: (a: any, b: any) => toFiniteNumber(b.raw) - toFiniteNumber(a.raw),
          callbacks: {
            label: (context: any) => isCostMetric
              ? `${context.dataset.label}: $${formatCost(toFiniteNumber(context.raw))}`
              : `${context.dataset.label}: ${formatTokens(toFiniteNumber(context.raw))}`,
            footer: (tooltipItems: any[]) => {
              if (isCostMetric) return ''
              const dataIndex = tooltipItems[0]?.dataIndex
              const point = normalizedTrendData.value[dataIndex]
              if (!point) return ''
              return `${t('admin.dashboard.actual')}: $${formatCost(point.actualCost)} · ${t('admin.dashboard.standard')}: $${formatCost(point.standardCost)}`
            }
          }
        }
      },
      scales: {
        x: {
          border: {
            display: false
          },
          grid: {
            display: false
          },
          ticks: {
            color: chartColors.value.text,
            maxRotation: 0,
            autoSkip: true,
            maxTicksLimit: 4,
            padding: 10,
            font: {
              size: 10
            }
          }
        },
        y: {
          beginAtZero: true,
          border: {
            display: false
          },
          grid: {
            color: chartColors.value.grid
          },
          ticks: {
            color: chartColors.value.text,
            maxTicksLimit: 5,
            padding: 8,
            font: {
              size: 10
            },
            callback: (value: string | number) => isCostMetric
              ? `$${formatCost(toFiniteNumber(value))}`
              : formatTokens(toFiniteNumber(value))
          }
        }
      }
    }
  }

  return {
    responsive: true,
    maintainAspectRatio: false,
    interaction: {
      intersect: false,
      mode: 'index' as const
    },
    plugins: {
      legend: {
        position: 'top' as const,
        labels: {
          color: chartColors.value.text,
          usePointStyle: true,
          pointStyle: 'circle',
          padding: 15,
          font: {
            size: 11
          }
        }
      },
      tooltip: {
        callbacks: {
          label: (context: any) => {
            if (context.dataset.yAxisID === 'yPercent') {
              return `${context.dataset.label}: ${toFiniteNumber(context.raw).toFixed(1)}%`
            }
            return `${context.dataset.label}: ${formatTokens(toFiniteNumber(context.raw))}`
          },
          footer: (tooltipItems: any[]) => {
            const dataIndex = tooltipItems[0]?.dataIndex
            const point = normalizedTrendData.value[dataIndex]
            if (!point) return ''
            return `${t('admin.dashboard.actual')}: $${formatCost(point.actualCost)} | ${t('admin.dashboard.standard')}: $${formatCost(point.standardCost)}`
          }
        }
      }
    },
    scales: {
      x: {
        grid: {
          color: chartColors.value.grid
        },
        ticks: {
          color: chartColors.value.text,
          font: {
            size: 10
          }
        }
      },
      y: {
        grid: {
          color: chartColors.value.grid
        },
        ticks: {
          color: chartColors.value.text,
          font: {
            size: 10
          },
          callback: (value: string | number) => formatTokens(toFiniteNumber(value))
        }
      },
      yPercent: {
        position: 'right' as const,
        min: 0,
        max: 100,
        grid: {
          drawOnChartArea: false
        },
        ticks: {
          color: chartColors.value.cacheHitRate,
          font: {
            size: 10
          },
          callback: (value: string | number) => `${value}%`
        }
      }
    }
  }
})

const formatTokens = (value: number): string => {
  if (value >= 1_000_000_000) {
    return `${(value / 1_000_000_000).toFixed(2)}B`
  } else if (value >= 1_000_000) {
    return `${(value / 1_000_000).toFixed(2)}M`
  } else if (value >= 1_000) {
    return `${(value / 1_000).toFixed(2)}K`
  }
  return value.toLocaleString()
}

const formatCost = (value: number): string => {
  if (value >= 1000) {
    return `${(value / 1000).toFixed(2)}K`
  } else if (value >= 1) {
    return value.toFixed(2)
  } else if (value >= 0.01) {
    return value.toFixed(3)
  }
  return value.toFixed(4)
}

const formatCostTotal = (value: number): string => value.toLocaleString(undefined, {
  minimumFractionDigits: 2,
  maximumFractionDigits: value >= 1 ? 2 : 4
})

onMounted(() => {
  themeObserver = new MutationObserver(() => {
    themeRevision.value += 1
  })
  themeObserver.observe(document.documentElement, {
    attributes: true,
    attributeFilter: ['class']
  })
})

onBeforeUnmount(() => {
  themeObserver?.disconnect()
  themeObserver = null
})
</script>

<style scoped>
.token-trend--home-clay {
  display: flex;
  min-height: 354px;
  flex-direction: column;
  color: var(--lx-clay-text);
  font-family: var(--lx-clay-font-ui);
}

.token-trend-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 18px;
}

.token-trend-title {
  margin: 0;
  color: var(--lx-clay-text);
  font-size: 18px;
  font-weight: 800;
  letter-spacing: -0.02em;
  text-wrap: balance;
}

.token-trend-eyebrow {
  display: block;
  margin-bottom: 5px;
  color: var(--lx-clay-text-muted);
  font-size: 9px;
  font-weight: 900;
  letter-spacing: 0.16em;
}

.token-trend-switch {
  display: inline-flex;
  flex: none;
  gap: 2px;
  padding: 3px;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-control);
  background: var(--lx-clay-recessed);
  box-shadow: var(--lx-clay-shadow-inset);
}

.token-trend-switch button {
  min-width: 58px;
  min-height: 36px;
  padding: 0 12px;
  border: 0;
  border-radius: 10px;
  color: var(--lx-clay-text-muted);
  background: transparent;
  font: inherit;
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
  transition:
    color 160ms ease,
    background-color 160ms ease,
    box-shadow 160ms ease;
}

.token-trend-switch button:hover:not(.active) {
  color: var(--lx-clay-text);
  background: var(--lx-clay-surface-soft);
}

.token-trend-switch button.active {
  color: var(--lx-clay-accent);
  background: var(--lx-clay-surface);
  box-shadow: var(--lx-clay-shadow-flat);
}

.token-trend-switch button:focus-visible,
.token-trend-error button:focus-visible {
  outline: 2px solid var(--lx-clay-accent);
  outline-offset: 2px;
}

.token-trend-content {
  display: flex;
  min-height: 0;
  flex: 1;
  flex-direction: column;
}

.token-trend-summary {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 18px;
  margin-bottom: 10px;
  padding-left: 42px;
}

.token-trend-total {
  display: flex;
  min-width: 0;
  flex-direction: column;
}

.token-trend-total span {
  color: var(--lx-clay-text-muted);
  font-size: 11px;
  font-weight: 600;
}

.token-trend-total strong {
  margin-top: 2px;
  color: var(--lx-clay-text);
  font-family: var(--lx-clay-font-display);
  font-size: 24px;
  font-variant-numeric: tabular-nums;
  font-weight: 900;
  letter-spacing: -0.025em;
}

.token-trend-legend {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px 14px;
  color: var(--lx-clay-text-muted);
  font-size: 11px;
  font-weight: 600;
}

.token-trend-legend span {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  white-space: nowrap;
}

.token-trend-legend i {
  width: 16px;
  height: 3px;
  border-radius: 999px;
}

.token-trend-legend .input,
.token-trend-legend .actual {
  background: var(--lx-clay-accent);
}

.token-trend-legend .output {
  background: var(--lx-clay-info);
}

.token-trend-legend .cache {
  background: var(--lx-clay-success-bright);
}

.token-trend-legend .standard {
  background: var(--lx-clay-text-subtle);
}

.token-trend-chart--home {
  min-height: 218px;
  flex: 1;
}

.token-trend-state {
  display: flex;
  flex: 1;
  align-items: center;
  justify-content: center;
  color: var(--lx-clay-text-muted, #6b7280);
  font-size: 13px;
  text-align: center;
}

.token-trend-loading {
  position: relative;
  min-height: 270px;
  align-items: stretch;
  flex-direction: column;
  justify-content: flex-start;
  gap: 20px;
}

.token-trend-skeleton-summary {
  width: 118px;
  height: 44px;
  margin-left: 42px;
  border-radius: 10px;
  background: var(--lx-clay-recessed);
}

.token-trend-skeleton-chart {
  display: flex;
  min-height: 190px;
  flex: 1;
  flex-direction: column;
  justify-content: space-between;
  padding: 12px 0 22px 42px;
}

.token-trend-skeleton-chart i {
  display: block;
  width: 100%;
  height: 1px;
  background: var(--lx-clay-border);
}

.token-trend-loading::after {
  position: absolute;
  top: 88px;
  left: 42px;
  width: 34%;
  height: 12px;
  border-radius: 999px;
  background: var(--lx-clay-recessed-strong);
  content: "";
}

.token-trend-error,
.token-trend-empty {
  min-height: 250px;
  flex-direction: column;
  gap: 8px;
}

.token-trend-error strong,
.token-trend-empty strong {
  color: var(--lx-clay-text, #374151);
  font-size: 14px;
}

.token-trend-error p,
.token-trend-empty p {
  max-width: 42ch;
  margin: 0;
  color: var(--lx-clay-text-muted, #6b7280);
  line-height: 1.55;
}

.token-trend-state-icon {
  display: grid;
  width: 32px;
  height: 32px;
  margin-bottom: 3px;
  place-items: center;
  border-radius: 50%;
  color: var(--lx-clay-danger, #c2415b);
  background: var(--lx-clay-danger-soft, #fff1f2);
  font-weight: 900;
}

.token-trend-error button {
  min-height: 36px;
  margin-top: 4px;
  padding: 0 14px;
  border: 1px solid var(--lx-clay-border-strong, #d1d5db);
  border-radius: 10px;
  color: var(--lx-clay-accent, #7c3aed);
  background: var(--lx-clay-surface, #fff);
  font: inherit;
  font-weight: 700;
  cursor: pointer;
}

.token-trend-empty-mark {
  display: grid;
  width: 42px;
  height: 42px;
  place-items: center;
  margin-bottom: 6px;
  border-radius: 14px;
  color: var(--lx-clay-accent);
  background: var(--lx-clay-accent-soft);
}

@media (max-width: 640px) {
  .token-trend--home-clay {
    min-height: 390px;
  }

  .token-trend-header {
    align-items: flex-start;
  }

  .token-trend-switch button {
    min-width: 52px;
    padding: 0 9px;
  }

  .token-trend-summary {
    align-items: flex-start;
    flex-direction: column;
    gap: 10px;
    padding-left: 0;
  }

  .token-trend-legend {
    justify-content: flex-start;
  }

  .token-trend-chart--home {
    min-height: 238px;
  }
}

@media (pointer: coarse) {
  .token-trend-switch button,
  .token-trend-error button {
    min-height: 44px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .token-trend-switch button {
    transition: none;
  }
}
</style>
