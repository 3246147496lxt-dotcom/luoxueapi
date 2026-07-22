<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Chart as ChartJS, ArcElement, Legend, Tooltip } from 'chart.js'
import { Doughnut } from 'vue-chartjs'
import type { OpsErrorDistributionResponse } from '@/api/admin/ops'
import type { ChartState } from '../types'
import HelpTooltip from '@/components/common/HelpTooltip.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import { getLuoxueClayChartTheme } from '@/utils/luoxueClayChartTheme'

ChartJS.register(ArcElement, Tooltip, Legend)

interface Props {
  data: OpsErrorDistributionResponse | null
  loading: boolean
}

const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'openDetails'): void
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
    warning: isDarkMode.value ? '#fbbf24' : '#b45309',
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

const totalSlaErrors = computed(() =>
  (props.data?.items ?? []).reduce((total, item) => total + Number(item.sla || 0), 0)
)

const hasData = computed(() => totalSlaErrors.value > 0)

const state = computed<ChartState>(() => {
  if (hasData.value) return 'ready'
  if (props.loading) return 'loading'
  return 'empty'
})

interface ErrorCategory {
  label: string
  count: number
  color: string
}

const categories = computed<ErrorCategory[]>(() => {
  if (!props.data) return []

  let upstream = 0 // 502, 503, 504
  let client = 0 // 4xx
  let system = 0 // 500
  let other = 0

  for (const item of props.data.items || []) {
    const code = Number(item.status_code || 0)
    const count = Number(item.sla || 0)
    if (!Number.isFinite(code) || !Number.isFinite(count)) continue

    if ([502, 503, 504].includes(code)) upstream += count
    else if (code >= 400 && code < 500) client += count
    else if (code === 500) system += count
    else other += count
  }

  const out: ErrorCategory[] = []
  if (upstream > 0) out.push({ label: t('admin.ops.upstream'), count: upstream, color: colors.value.warning })
  if (client > 0) out.push({ label: t('admin.ops.client'), count: client, color: colors.value.info })
  if (system > 0) out.push({ label: t('admin.ops.system'), count: system, color: colors.value.danger })
  if (other > 0) out.push({ label: t('admin.ops.other'), count: other, color: colors.value.neutral })
  return out
})

const topReason = computed(() => {
  if (categories.value.length === 0) return null
  return categories.value.reduce((prev, cur) => (cur.count > prev.count ? cur : prev))
})

const chartData = computed(() => {
  if (!hasData.value || categories.value.length === 0) return null
  return {
    labels: categories.value.map((c) => c.label),
    datasets: [
      {
        data: categories.value.map((c) => c.count),
        backgroundColor: categories.value.map((c) => c.color),
        borderWidth: 0
      }
    ]
  }
})

const options = computed(() => {
  const c = colors.value
  return {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      legend: { display: false },
      tooltip: {
        backgroundColor: c.surface,
        titleColor: c.textStrong,
        bodyColor: c.text,
        borderColor: c.grid,
        borderWidth: 1,
        padding: 10
      }
    }
  }
})
</script>

<template>
  <section class="ops-chart-panel" aria-labelledby="ops-error-distribution-title">
    <header class="ops-chart-header">
      <div class="ops-chart-title-group">
        <span class="ops-chart-icon ops-chart-icon--danger" aria-hidden="true">
          <svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
          />
          </svg>
        </span>
        <h3 id="ops-error-distribution-title" class="ops-chart-title">{{ t('admin.ops.errorDistribution') }}</h3>
        <HelpTooltip :content="t('admin.ops.tooltips.errorDistribution')" trigger="click">
          <template #trigger>
            <button
              type="button"
              class="ops-chart-help"
              :aria-label="t('admin.ops.tooltips.errorDistribution')"
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
      <button
        type="button"
        class="ops-chart-action"
        :disabled="state !== 'ready'"
        :title="t('admin.ops.errorTrend')"
        @click="emit('openDetails')"
      >
        {{ t('admin.ops.requestDetails.details') }}
      </button>
    </header>

    <div class="ops-chart-plot">
      <div v-if="state === 'ready' && chartData" class="ops-distribution-ready">
        <div class="ops-donut-stage">
          <Doughnut :data="chartData" :options="{ ...options, cutout: '65%' }" />
          <div v-if="topReason" class="ops-donut-summary" aria-hidden="true">
            <span>{{ t('admin.ops.top') }}</span>
            <strong :style="{ color: topReason.color }">{{ topReason.label }}</strong>
          </div>
        </div>
        <div class="ops-chart-legend" :aria-label="t('admin.ops.errorDistribution')">
          <div v-for="item in categories" :key="item.label" class="ops-chart-legend__item">
            <span class="ops-chart-legend__dot" :style="{ backgroundColor: item.color }" aria-hidden="true"></span>
            <span class="ops-chart-legend__label">{{ item.label }}</span>
            <strong class="ops-chart-legend__value">{{ item.count }}</strong>
          </div>
        </div>
      </div>

      <div v-else class="ops-chart-state">
        <div v-if="state === 'loading'" class="ops-chart-loading" role="status" aria-live="polite">
          <span class="sr-only">{{ t('common.loading') }}</span>
          <span class="ops-chart-loading__ring"></span>
          <span class="ops-chart-loading__line"></span>
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
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--lx-clay-border);
  border-radius: 0.6875rem;
  padding: 0.5rem 0.75rem;
  color: var(--lx-clay-text-secondary);
  background: var(--lx-clay-surface-soft);
  font-family: var(--lx-clay-font-ui);
  font-size: 0.75rem;
  font-weight: 750;
  line-height: 1;
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

.ops-distribution-ready {
  display: flex;
  height: 100%;
  min-height: 0;
  flex-direction: column;
}

.ops-donut-stage {
  position: relative;
  min-height: 10rem;
  flex: 1;
}

.ops-donut-summary {
  position: absolute;
  top: 50%;
  left: 50%;
  display: grid;
  max-width: 5.5rem;
  gap: 0.125rem;
  color: var(--lx-clay-text-muted);
  font-size: 0.625rem;
  line-height: 1.15;
  text-align: center;
  transform: translate(-50%, -50%);
  pointer-events: none;
}

.ops-donut-summary strong {
  overflow: hidden;
  font-size: 0.75rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ops-chart-legend {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.375rem 0.75rem;
  padding-top: 0.625rem;
  border-top: 1px solid var(--lx-clay-border);
}

.ops-chart-legend__item {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 0.375rem;
  min-width: 0;
  color: var(--lx-clay-text-secondary);
  font-size: 0.6875rem;
}

.ops-chart-legend__dot {
  width: 0.5rem;
  height: 0.5rem;
  border-radius: 50%;
}

.ops-chart-legend__label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ops-chart-legend__value {
  color: var(--lx-clay-text);
  font-variant-numeric: tabular-nums;
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
  width: min(100%, 9rem);
  justify-items: center;
  gap: 0.875rem;
}

.ops-chart-loading__ring {
  width: 5.5rem;
  height: 5.5rem;
  border: 0.75rem solid var(--lx-clay-recessed-strong);
  border-radius: 50%;
  animation: ops-chart-pulse 1.7s ease-in-out infinite;
}

.ops-chart-loading__line {
  display: block;
  width: 72%;
  height: 0.625rem;
  border-radius: 999px;
  background: var(--lx-clay-recessed-strong);
  animation: ops-chart-pulse 1.7s ease-in-out infinite;
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
  .ops-chart-loading__ring,
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
