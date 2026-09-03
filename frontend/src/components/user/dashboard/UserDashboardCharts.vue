<template>
  <section class="dashboard-usage" :aria-label="t('dashboard.workspace.usageStatistics')">
    <header class="dashboard-usage__header">
      <h2>{{ t('dashboard.workspace.usageStatistics') }}</h2>
      <div
        ref="periodMenuRef"
        class="dashboard-period-select"
        :class="{ 'is-open': periodMenuOpen }"
      >
        <button
          type="button"
          class="dashboard-period-select__trigger"
          data-testid="dashboard-period-trigger"
          :aria-label="t('dashboard.workspace.statisticsPeriod')"
          :aria-expanded="periodMenuOpen"
          aria-haspopup="listbox"
          aria-controls="dashboard-period-menu"
          @click="togglePeriodMenu"
          @keydown="handlePeriodTriggerKeydown"
        >
          <span class="dashboard-period-select__label">
            {{ t('dashboard.workspace.statisticsPeriod') }}
          </span>
          <span class="dashboard-period-select__value">{{ currentPeriodLabel }}</span>
          <Icon
            :name="periodMenuOpen ? 'chevronUp' : 'chevronDown'"
            size="sm"
            :stroke-width="2"
            class="dashboard-period-select__icon"
            aria-hidden="true"
          />
        </button>

        <Transition name="dashboard-period-dropdown">
          <div
            v-if="periodMenuOpen"
            id="dashboard-period-menu"
            class="dashboard-period-select__menu"
            data-testid="dashboard-period-menu"
            role="listbox"
            :aria-label="t('dashboard.workspace.statisticsPeriod')"
          >
            <button
              v-for="option in periodOptions"
              :key="option.value"
              type="button"
              class="dashboard-period-select__option"
              role="option"
              :data-period="option.value"
              :aria-selected="period === option.value"
              @click="selectPeriod(option.value)"
            >
              {{ t(option.labelKey) }}
            </button>
          </div>
        </Transition>
      </div>
    </header>

    <div class="dashboard-usage__grid">
      <article class="dashboard-chart-card dashboard-chart-card--credits">
        <div class="dashboard-chart-card__heading">
          <h3>{{ t('dashboard.workspace.creditsTrend') }}</h3>
          <span data-testid="credits-total">{{ formattedCreditsTotal }}</span>
        </div>
        <div class="dashboard-chart dashboard-chart--credits">
          <div v-if="loading" class="dashboard-chart__state" role="status">
            <span class="skeleton h-full w-full" aria-hidden="true" />
            <span class="sr-only">{{ t('common.loading') }}</span>
          </div>
          <div v-else-if="!hasData" class="dashboard-chart__state dashboard-chart__empty">
            {{ t('dashboard.workspace.noUsageData') }}
          </div>
          <Line
            v-else
            :key="`credits-${period}-${themeRevision}`"
            :data="creditsChartData"
            :options="creditsChartOptions"
            :plugins="lineChartPlugins"
            role="img"
            :aria-label="`${t('dashboard.workspace.creditsTrend')} ${formattedCreditsTotal}`"
          />
        </div>
      </article>

      <div class="dashboard-usage__side">
        <article class="dashboard-chart-card dashboard-chart-card--token">
          <div class="dashboard-chart-card__heading">
            <h3>{{ t('dashboard.workspace.token') }}</h3>
            <span data-testid="tokens-total">{{ formattedTokensTotal }}</span>
          </div>
          <div class="dashboard-chart dashboard-chart--token">
            <div v-if="loading" class="dashboard-chart__state" role="status">
              <span class="skeleton h-full w-full" aria-hidden="true" />
              <span class="sr-only">{{ t('common.loading') }}</span>
            </div>
            <div v-else-if="!hasData" class="dashboard-chart__state dashboard-chart__empty">
              {{ t('dashboard.workspace.noUsageData') }}
            </div>
            <Line
              v-else
              :key="`tokens-${period}-${themeRevision}`"
              :data="tokensChartData"
              :options="tokensChartOptions"
              :plugins="lineChartPlugins"
              role="img"
              :aria-label="`${t('dashboard.workspace.token')} ${formattedTokensTotal}`"
            />
          </div>
        </article>

        <article class="dashboard-chart-card dashboard-chart-card--requests">
          <div class="dashboard-chart-card__heading">
            <h3>{{ t('dashboard.workspace.requestCount') }}</h3>
            <span data-testid="requests-total">{{ formattedRequestsTotal }}</span>
          </div>
          <div class="dashboard-chart dashboard-chart--requests">
            <div v-if="loading" class="dashboard-chart__state" role="status">
              <span class="skeleton h-full w-full" aria-hidden="true" />
              <span class="sr-only">{{ t('common.loading') }}</span>
            </div>
            <div v-else-if="!hasData" class="dashboard-chart__state dashboard-chart__empty">
              {{ t('dashboard.workspace.noUsageData') }}
            </div>
            <Bar
              v-else
              :key="`requests-${period}-${themeRevision}`"
              :data="requestsChartData"
              :options="requestsChartOptions"
              :plugins="barChartPlugins"
              role="img"
              :aria-label="`${t('dashboard.workspace.requestCount')} ${formattedRequestsTotal}`"
            />
          </div>
        </article>
      </div>
    </div>

    <table v-if="hasData" class="sr-only">
      <caption>{{ t('dashboard.workspace.usageStatistics') }}</caption>
      <thead>
        <tr>
          <th scope="col">{{ t('usage.time') }}</th>
          <th scope="col">{{ t('dashboard.workspace.creditsTrend') }}</th>
          <th scope="col">{{ t('dashboard.workspace.cachedInput') }}</th>
          <th scope="col">{{ t('dashboard.workspace.uncachedInput') }}</th>
          <th scope="col">{{ t('dashboard.workspace.outputTokens') }}</th>
          <th scope="col">{{ t('dashboard.workspace.requestCount') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="point in usagePoints" :key="point.date">
          <th scope="row">{{ tooltipRange(point.date) }}</th>
          <td>{{ formatCredits(point.credits) }}</td>
          <td>{{ formatInteger(point.cachedInputTokens) }}</td>
          <td>{{ formatInteger(point.uncachedInputTokens) }}</td>
          <td>{{ formatInteger(point.outputTokens) }}</td>
          <td>{{ formatInteger(point.requests) }}</td>
        </tr>
      </tbody>
    </table>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import {
  BarElement,
  CategoryScale,
  Chart as ChartJS,
  Filler,
  LineElement,
  LinearScale,
  PointElement,
  Tooltip,
  type ChartData,
  type ChartOptions,
  type Plugin,
} from 'chart.js'
import { Bar, Line } from 'vue-chartjs'
import type { TrendDataPoint } from '@/types'
import {
  buildDashboardUsageSeries,
  type DashboardUsagePeriod,
} from './dashboardUsage'

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  BarElement,
  Tooltip,
  Filler,
)

const props = defineProps<{
  loading: boolean
  period: DashboardUsagePeriod
  trend: TrendDataPoint[]
}>()

const emit = defineEmits<{
  'update:period': [period: DashboardUsagePeriod]
}>()

const { t, locale } = useI18n()
const themeRevision = ref(0)
const periodMenuRef = ref<HTMLElement | null>(null)
const periodMenuOpen = ref(false)
let themeObserver: MutationObserver | null = null

const periodOptions = [
  { value: 'today' as DashboardUsagePeriod, labelKey: 'dashboard.workspace.periods.today' },
  { value: 'week' as DashboardUsagePeriod, labelKey: 'dashboard.workspace.periods.week' },
  { value: 'month' as DashboardUsagePeriod, labelKey: 'dashboard.workspace.periods.month' },
  { value: 'thirtyDays' as DashboardUsagePeriod, labelKey: 'dashboard.workspace.periods.thirtyDays' },
]

const currentPeriodLabel = computed(() => {
  const option = periodOptions.find((candidate) => candidate.value === props.period) || periodOptions[0]
  return t(option.labelKey)
})

const usagePoints = computed(() => buildDashboardUsageSeries(props.trend, props.period))
const hasData = computed(() => props.trend.length > 0)
const numberLocale = computed(() => locale.value.startsWith('zh') ? 'zh-CN' : 'en-US')
const isDark = computed(() => {
  themeRevision.value
  return typeof document !== 'undefined' && document.documentElement.classList.contains('dark')
})

function readThemeColor(token: string, lightFallback: string, darkFallback = lightFallback): string {
  themeRevision.value
  const fallback = isDark.value ? darkFallback : lightFallback
  if (typeof window === 'undefined') return fallback
  return window.getComputedStyle(document.documentElement).getPropertyValue(token).trim() || fallback
}

const chartTheme = computed(() => ({
  text: readThemeColor('--workspace-dashboard-text-muted', '#64748b', '#8a8a8a'),
  grid: readThemeColor('--workspace-dashboard-card-border', '#f1f5f9', 'rgb(255 255 255 / 0.1)'),
  guide: readThemeColor('--workspace-dashboard-text-subtle', '#64748b', '#8a8a8a'),
  tooltip: readThemeColor('--workspace-dashboard-tooltip-surface', '#0f172a', '#212121'),
  tooltipText: readThemeColor('--workspace-dashboard-tooltip-text', '#f8fafc', '#ececec'),
  tooltipTitle: readThemeColor('--workspace-dashboard-tooltip-title', '#cbd5e1', '#b4b4b4'),
  tooltipBorder: readThemeColor('--workspace-dashboard-tooltip-border', 'rgb(255 255 255 / 0.1)', 'rgb(255 255 255 / 0.16)'),
  point: readThemeColor('--workspace-card-surface', '#ffffff', '#171717'),
  cost: readThemeColor('--workspace-work-chart-cost', '#ea580c', '#fb923c'),
  costFillStrong: readThemeColor('--workspace-work-chart-cost-fill-strong', 'rgb(234 88 12 / 0.18)', 'rgb(251 146 60 / 0.18)'),
  costFillSoft: readThemeColor('--workspace-work-chart-cost-fill-soft', 'rgb(234 88 12 / 0.06)', 'rgb(251 146 60 / 0.06)'),
  costFillTransparent: readThemeColor('--workspace-work-chart-cost-fill-transparent', 'rgb(234 88 12 / 0)', 'rgb(251 146 60 / 0)'),
  token: readThemeColor('--workspace-work-chart-primary', '#2563eb', '#60a5fa'),
  tokenFillStrong: readThemeColor('--workspace-work-chart-primary-fill-strong', 'rgb(37 99 235 / 0.18)', 'rgb(96 165 250 / 0.18)'),
  tokenFillSoft: readThemeColor('--workspace-work-chart-primary-fill-soft', 'rgb(37 99 235 / 0.06)', 'rgb(96 165 250 / 0.06)'),
  tokenFillTransparent: readThemeColor('--workspace-work-chart-primary-fill-transparent', 'rgb(37 99 235 / 0)', 'rgb(96 165 250 / 0)'),
  request: readThemeColor('--workspace-work-chart-secondary', '#60a5fa', '#93c5fd'),
  requestHover: readThemeColor('--workspace-work-accent', '#2563eb', '#60a5fa'),
}))

const creditsTotal = computed(() => usagePoints.value.reduce((sum, point) => sum + point.credits, 0))
const tokensTotal = computed(() => usagePoints.value.reduce((sum, point) => sum + point.totalTokens, 0))
const requestsTotal = computed(() => usagePoints.value.reduce((sum, point) => sum + point.requests, 0))
const formattedCreditsTotal = computed(() => formatCredits(creditsTotal.value, creditsTotal.value >= 100))
const formattedTokensTotal = computed(() => formatInteger(tokensTotal.value))
const formattedRequestsTotal = computed(() => formatInteger(requestsTotal.value))
const chartLabels = computed(() => usagePoints.value.map((point) => axisLabel(point.date)))

function createAreaGradient(
  context: any,
  strong: string,
  soft: string,
  transparent: string,
): string | CanvasGradient {
  const chart = context.chart
  const area = chart.chartArea
  if (!area) return strong
  const gradient = chart.ctx.createLinearGradient(0, area.top, 0, area.bottom)
  gradient.addColorStop(0, strong)
  gradient.addColorStop(0.62, soft)
  gradient.addColorStop(1, transparent)
  return gradient
}

const creditsChartData = computed<ChartData<'line'>>(() => ({
  labels: chartLabels.value,
  datasets: [{
    label: t('dashboard.workspace.creditsTrend'),
    data: usagePoints.value.map((point) => point.credits),
    borderColor: chartTheme.value.cost,
    backgroundColor: (context: any) => createAreaGradient(
      context,
      chartTheme.value.costFillStrong,
      chartTheme.value.costFillSoft,
      chartTheme.value.costFillTransparent,
    ),
    borderWidth: 2,
    fill: true,
    tension: 0.36,
    cubicInterpolationMode: 'monotone',
    pointRadius: 0,
    pointHoverRadius: 4,
    pointHitRadius: 12,
    pointHoverBackgroundColor: chartTheme.value.point,
    pointHoverBorderColor: chartTheme.value.cost,
    pointHoverBorderWidth: 2,
  }],
}))

const tokensChartData = computed<ChartData<'line'>>(() => ({
  labels: chartLabels.value,
  datasets: [{
    label: t('dashboard.workspace.token'),
    data: usagePoints.value.map((point) => point.totalTokens),
    borderColor: chartTheme.value.token,
    backgroundColor: (context: any) => createAreaGradient(
      context,
      chartTheme.value.tokenFillStrong,
      chartTheme.value.tokenFillSoft,
      chartTheme.value.tokenFillTransparent,
    ),
    borderWidth: 2,
    fill: true,
    tension: 0.36,
    cubicInterpolationMode: 'monotone',
    pointRadius: 0,
    pointHoverRadius: 4,
    pointHitRadius: 12,
    pointHoverBackgroundColor: chartTheme.value.point,
    pointHoverBorderColor: chartTheme.value.token,
    pointHoverBorderWidth: 2,
  }],
}))

const requestsChartData = computed<ChartData<'bar'>>(() => ({
  labels: chartLabels.value,
  datasets: [{
    label: t('dashboard.workspace.requestCount'),
    data: usagePoints.value.map((point) => point.requests),
    backgroundColor: chartTheme.value.request,
    hoverBackgroundColor: chartTheme.value.requestHover,
    borderRadius: 3,
    borderSkipped: false,
    barPercentage: 0.64,
    categoryPercentage: 0.72,
  }],
}))

const creditsChartOptions = computed<ChartOptions<'line'>>(() => ({
  ...baseOptions<'line'>(usagePoints.value.map((point) => point.credits), formatAxisCredit, 10),
  plugins: {
    legend: { display: false },
    tooltip: tooltipOptions((index) => [
      `${t('dashboard.workspace.creditsUsed')}: ${formatCredits(usagePoints.value[index]?.credits ?? 0)} ${t('dashboard.workspace.creditsUnit')}`,
    ]),
  },
}))

const tokensChartOptions = computed<ChartOptions<'line'>>(() => ({
  ...baseOptions<'line'>(usagePoints.value.map((point) => point.totalTokens), formatAxisNumber, 9),
  plugins: {
    legend: { display: false },
    tooltip: tooltipOptions((index) => {
      const point = usagePoints.value[index]
      if (!point) return []
      return [
        `${t('dashboard.workspace.cachedInput')}: ${formatInteger(point.cachedInputTokens)}`,
        `${t('dashboard.workspace.uncachedInput')}: ${formatInteger(point.uncachedInputTokens)}`,
        `${t('dashboard.workspace.outputTokens')}: ${formatInteger(point.outputTokens)}`,
      ]
    }),
  },
}))

const requestsChartOptions = computed<ChartOptions<'bar'>>(() => ({
  ...baseOptions<'bar'>(usagePoints.value.map((point) => point.requests), formatAxisNumber, 9),
  plugins: {
    legend: { display: false },
    tooltip: tooltipOptions((index) => [
      `${t('dashboard.workspace.requestCount')}: ${formatInteger(usagePoints.value[index]?.requests ?? 0)}`,
    ]),
  },
}))

function createHoverGuidePlugin<T extends 'line' | 'bar'>(): Plugin<T> {
  return {
    id: 'dashboardHoverGuide',
    afterDatasetsDraw(chart) {
      const active = chart.tooltip?.getActiveElements?.()?.[0]
      if (!active) return
      const x = active.element.x
      const { top, bottom } = chart.chartArea
      const context = chart.ctx
      context.save()
      context.beginPath()
      context.setLineDash([4, 4])
      context.moveTo(x, top)
      context.lineTo(x, bottom)
      context.lineWidth = 1
      context.strokeStyle = chartTheme.value.guide
      context.stroke()
      context.restore()
    },
  }
}

const lineChartPlugins: Plugin<'line'>[] = [createHoverGuidePlugin<'line'>()]
const barChartPlugins: Plugin<'bar'>[] = [createHoverGuidePlugin<'bar'>()]

function baseOptions<T extends 'line' | 'bar'>(
  values: number[],
  tickFormatter: (value: number) => string,
  axisFontSize: number,
): ChartOptions<T> {
  return {
    responsive: true,
    maintainAspectRatio: false,
    animation: false,
    interaction: { intersect: false, mode: 'index' },
    layout: { padding: { top: 6 } },
    scales: {
      x: {
        border: { display: false },
        grid: { display: false },
        ticks: {
          color: chartTheme.value.text,
          font: {
            family: 'Inter, "PingFang SC", "Microsoft YaHei", sans-serif',
            size: axisFontSize,
            weight: 500,
          },
          autoSkip: true,
          maxTicksLimit: 4,
          maxRotation: 0,
          padding: 8,
        },
      },
      y: {
        beginAtZero: true,
        max: niceMaximum(values),
        border: { display: false },
        grid: { color: chartTheme.value.grid, drawTicks: false },
        ticks: {
          color: chartTheme.value.text,
          font: {
            family: 'Inter, "PingFang SC", "Microsoft YaHei", sans-serif',
            size: axisFontSize,
            weight: 500,
          },
          count: 3,
          padding: 8,
          callback: (value: string | number) => tickFormatter(Number(value)),
        },
      },
    },
  } as unknown as ChartOptions<T>
}

function tooltipOptions(lines: (index: number) => string[]) {
  return {
    enabled: true,
    displayColors: false,
    backgroundColor: chartTheme.value.tooltip,
    titleColor: chartTheme.value.tooltipTitle,
    bodyColor: chartTheme.value.tooltipText,
    borderColor: chartTheme.value.tooltipBorder,
    borderWidth: 1,
    cornerRadius: 12,
    padding: 12,
    caretPadding: 8,
    titleFont: { size: 11, weight: 500 },
    bodyFont: { size: 11, weight: 400 },
    titleMarginBottom: 6,
    callbacks: {
      title: (items: any[]) => {
        const index = items[0]?.dataIndex ?? 0
        return tooltipRange(usagePoints.value[index]?.date ?? '')
      },
      label: (context: any) => lines(context.dataIndex),
    },
  }
}

function niceMaximum(values: number[]): number {
  const maximum = Math.max(0, ...values)
  if (maximum <= 0) return 1
  const exponent = 10 ** Math.floor(Math.log10(maximum))
  const normalized = maximum / exponent
  const step = normalized <= 1 ? 1 : normalized <= 2 ? 2 : normalized <= 5 ? 5 : 10
  return step * exponent
}

function parsePointDate(value: string): Date | null {
  const normalized = /^\d{4}-\d{2}-\d{2}$/.test(value)
    ? `${value}T00:00:00`
    : value.replace(' ', 'T')
  const date = new Date(normalized)
  return Number.isFinite(date.getTime()) ? date : null
}

function axisLabel(value: string): string {
  const date = parsePointDate(value)
  if (!date) return value
  if (props.period === 'today') {
    return `${String(date.getHours()).padStart(2, '0')}:00`
  }
  if (props.period === 'week') {
    return new Intl.DateTimeFormat(numberLocale.value, { weekday: 'short' }).format(date)
  }
  if (props.period === 'month') return String(date.getDate())
  return new Intl.DateTimeFormat(numberLocale.value, { month: 'numeric', day: 'numeric' }).format(date)
}

function tooltipRange(value: string): string {
  const date = parsePointDate(value)
  if (!date) return value
  if (props.period === 'today') {
    const start = `${String(date.getHours()).padStart(2, '0')}:00`
    const endDate = new Date(date)
    endDate.setHours(date.getHours() + 1)
    const end = `${String(endDate.getHours()).padStart(2, '0')}:00`
    return `${start}~${end}`
  }
  return new Intl.DateTimeFormat(numberLocale.value, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  }).format(date)
}

function formatInteger(value: number): string {
  return new Intl.NumberFormat(numberLocale.value, { maximumFractionDigits: 0 }).format(value)
}

function formatCredits(value: number, forceInteger = false): string {
  return new Intl.NumberFormat(numberLocale.value, {
    minimumFractionDigits: 0,
    maximumFractionDigits: forceInteger ? 0 : value > 0 && value < 0.01 ? 4 : 2,
  }).format(value)
}

function formatAxisNumber(value: number): string {
  return new Intl.NumberFormat(numberLocale.value, {
    notation: value >= 1_000 ? 'compact' : 'standard',
    maximumFractionDigits: 1,
  }).format(value)
}

function formatAxisCredit(value: number): string {
  return new Intl.NumberFormat(numberLocale.value, { maximumFractionDigits: value < 10 ? 1 : 0 }).format(value)
}

function togglePeriodMenu(): void {
  periodMenuOpen.value = !periodMenuOpen.value
}

function selectPeriod(value: DashboardUsagePeriod): void {
  periodMenuOpen.value = false
  if (value !== props.period) emit('update:period', value)
}

function handlePeriodTriggerKeydown(event: KeyboardEvent): void {
  if (event.key === 'Escape') {
    periodMenuOpen.value = false
    return
  }
  if (event.key === 'ArrowDown' || event.key === 'Enter' || event.key === ' ') {
    event.preventDefault()
    periodMenuOpen.value = true
  }
}

function handlePeriodOutsidePointerDown(event: PointerEvent): void {
  if (!periodMenuRef.value?.contains(event.target as Node)) {
    periodMenuOpen.value = false
  }
}

onMounted(() => {
  document.addEventListener('pointerdown', handlePeriodOutsidePointerDown)

  if (typeof MutationObserver === 'undefined') return
  themeObserver = new MutationObserver(() => {
    themeRevision.value += 1
  })
  themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', handlePeriodOutsidePointerDown)
  themeObserver?.disconnect()
})
</script>

<style scoped>
.dashboard-usage {
  min-width: 0;
}

.dashboard-usage__header {
  display: flex;
  min-height: 44px;
  align-items: center;
  justify-content: space-between;
  gap: var(--workspace-space-4);
  margin-bottom: var(--workspace-space-4);
}

.dashboard-usage__header h2 {
  color: var(--workspace-dashboard-text-strong);
  font-size: calc(var(--workspace-type-navigation-size) + 0.375rem);
  font-weight: 700;
  line-height: var(--workspace-space-7);
}

.dashboard-period-select {
  position: relative;
  display: flex;
  min-height: 40px;
  align-items: center;
  z-index: 1;
}

.dashboard-period-select__trigger {
  display: flex;
  min-width: 142px;
  min-height: 40px;
  align-items: center;
  gap: var(--workspace-space-3);
  padding: 0 var(--workspace-space-3);
  border: 0;
  border-radius: 0;
  color: var(--workspace-dashboard-period-text);
  background: var(--workspace-canvas);
  font-size: var(--workspace-type-body-size);
  font-weight: var(--workspace-type-body-weight);
  line-height: 1.25rem;
  cursor: pointer;
}

.dashboard-period-select__trigger:focus {
  outline: none;
}

.dashboard-period-select__trigger:focus-visible {
  outline: none;
}

.dashboard-period-select__label {
  color: var(--workspace-dashboard-text-subtle);
  font-weight: var(--workspace-type-navigation-weight);
  white-space: nowrap;
}

.dashboard-period-select__value {
  color: var(--workspace-dashboard-period-text);
  font-weight: 700;
  white-space: nowrap;
}

.dashboard-period-select__icon {
  flex: 0 0 auto;
  margin-left: auto;
  color: var(--workspace-dashboard-text-muted);
}

.dashboard-period-select.is-open {
  z-index: 30;
}

.dashboard-period-select.is-open .dashboard-period-select__trigger {
  background: transparent;
}

.dashboard-period-select.is-open .dashboard-period-select__trigger:focus-visible {
  outline: none;
}

.dashboard-period-select__menu {
  position: absolute;
  top: calc(100% + var(--workspace-space-2));
  right: 0;
  z-index: 40;
  width: 128px;
  padding: var(--workspace-space-2) 0;
  border: 1px solid var(--workspace-dashboard-card-border);
  border-radius: 16px;
  background: var(--workspace-card-surface);
  box-shadow: var(--workspace-dashboard-card-shadow);
}

.dashboard-period-select__option {
  display: flex;
  width: 100%;
  min-height: 32px;
  align-items: center;
  padding: var(--workspace-space-2) var(--workspace-space-4);
  border: 0;
  color: var(--workspace-dashboard-text-heading);
  background: transparent;
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-body-weight);
  line-height: 1rem;
  text-align: left;
  cursor: pointer;
}

.dashboard-period-select__option:hover {
  background: var(--workspace-dashboard-hover);
}

.dashboard-period-dropdown-enter-active,
.dashboard-period-dropdown-leave-active {
  transition: opacity 120ms ease, transform 120ms ease;
  transform-origin: top right;
}

.dashboard-period-dropdown-enter-from,
.dashboard-period-dropdown-leave-to {
  opacity: 0;
  transform: translateY(-4px) scale(0.98);
}

.dashboard-usage__grid {
  display: grid;
  grid-template-columns: minmax(0, 3fr) minmax(20rem, 2fr);
  gap: var(--workspace-space-6);
  align-items: stretch;
}

.dashboard-usage__side {
  display: grid;
  min-width: 0;
  gap: var(--workspace-space-6);
}

.dashboard-chart-card {
  min-width: 0;
  padding: var(--workspace-space-6);
  border: 1px solid var(--workspace-dashboard-card-border);
  border-radius: 24px;
  background: var(--workspace-card-surface);
  box-shadow: var(--workspace-dashboard-card-shadow);
}

.dashboard-chart-card__heading {
  display: flex;
  min-width: 0;
  min-height: 36px;
  align-items: baseline;
  gap: 8px;
  margin-bottom: 8px;
  white-space: nowrap;
}

.dashboard-chart-card__heading h3 {
  flex: 0 0 auto;
  color: var(--workspace-dashboard-text-heading);
  font-size: var(--workspace-type-navigation-size);
  font-weight: 700;
  line-height: 1.25rem;
}

.dashboard-chart-card__heading > span {
  overflow: hidden;
  color: var(--workspace-dashboard-text-subtle);
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-body-weight);
  line-height: 1.25rem;
  letter-spacing: -0.025em;
  text-overflow: ellipsis;
  font-variant-numeric: tabular-nums;
}

.dashboard-chart-card--credits .dashboard-chart-card__heading {
  min-height: 52px;
  margin-bottom: var(--workspace-space-8);
}

.dashboard-chart-card--requests .dashboard-chart-card__heading {
  margin-bottom: var(--workspace-space-1);
}

.dashboard-chart {
  position: relative;
  min-width: 0;
}

.dashboard-chart--credits {
  height: 320px;
}

.dashboard-chart--token {
  height: 168px;
}

.dashboard-chart--requests {
  height: 86px;
}

.dashboard-chart__state {
  display: flex;
  width: 100%;
  height: 100%;
  align-items: center;
  justify-content: center;
}

.dashboard-chart__empty {
  color: var(--workspace-dashboard-text-muted);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
  text-align: center;
}

@media (max-width: 1279px) {
  .dashboard-usage__grid {
    grid-template-columns: minmax(0, 1fr);
  }

  .dashboard-usage__side {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 767px) {
  .dashboard-usage__header {
    align-items: flex-start;
  }

  .dashboard-period-select {
    min-height: 44px;
  }

  .dashboard-period-select__trigger {
    min-height: 42px;
  }

  .dashboard-usage__side {
    grid-template-columns: minmax(0, 1fr);
  }

  .dashboard-chart-card {
    padding: 18px;
  }

  .dashboard-chart--credits {
    height: 260px;
  }
}

@media (max-width: 479px) {
  .dashboard-usage__header {
    flex-direction: column;
  }

  .dashboard-period-select {
    width: 100%;
  }

  .dashboard-period-select__trigger {
    width: 100%;
    justify-content: flex-start;
  }
}

</style>
