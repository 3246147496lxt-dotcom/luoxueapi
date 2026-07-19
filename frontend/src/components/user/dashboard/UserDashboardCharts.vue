<template>
  <section
    class="dashboard-panel relative overflow-hidden"
    :aria-label="t('dashboard.modelAnalysis')"
  >
    <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700 md:px-6">
      <div class="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
        <div class="flex items-center gap-2 text-sm font-medium text-gray-800 dark:text-gray-100">
          <Icon name="chart" size="sm" :stroke-width="1.8" />
          <span>{{ t('dashboard.modelAnalysis') }}</span>
        </div>

        <div
          class="flex flex-wrap items-center gap-1.5"
          role="tablist"
          :aria-label="t('dashboard.analysisTabs.label')"
        >
          <button
            v-for="tab in tabs"
            :key="tab.id"
            type="button"
            role="tab"
            class="analysis-tab"
            :class="activeTab === tab.id ? 'analysis-tab-active' : ''"
            :aria-selected="activeTab === tab.id"
            @click="activeTab = tab.id"
          >
            <Icon :name="tab.icon" size="xs" :stroke-width="1.8" />
            <span>{{ t(tab.labelKey) }}</span>
          </button>
        </div>
      </div>
    </div>

    <div class="relative min-h-[430px] px-5 py-6 md:px-7 md:py-7">
      <div
        v-if="loading"
        class="absolute inset-0 z-10 flex items-center justify-center bg-white/80 backdrop-blur-[1px] dark:bg-dark-800/80"
      >
        <LoadingSpinner size="md" />
      </div>

      <div v-if="activeTab === 'calendar'" role="tabpanel" class="space-y-5">
        <div class="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <h3 class="text-base font-semibold text-gray-900 dark:text-white">
              {{ t('dashboard.analysisTabs.usageCalendar') }}
            </h3>
            <p class="mt-1 text-sm text-gray-400 dark:text-dark-500">
              {{ calendarMonthLabel }}
            </p>
          </div>
          <div class="text-sm text-gray-500 dark:text-dark-400">
            {{ t('dashboard.calendarSummary', { days: activeCalendarDays, requests: formatNumber(calendarMonthRequests) }) }}
          </div>
        </div>

        <div class="overflow-hidden rounded-xl border border-gray-100 dark:border-dark-700">
          <div class="grid grid-cols-7 border-b border-gray-100 bg-gray-50/75 dark:border-dark-700 dark:bg-dark-900/40">
            <div
              v-for="day in weekdays"
              :key="day"
              class="px-1 py-2 text-center text-[11px] font-medium text-gray-500 dark:text-dark-400"
            >
              {{ day }}
            </div>
          </div>
          <div class="grid grid-cols-7 bg-gray-100/80 gap-px dark:bg-dark-700">
            <div
              v-for="cell in calendarCells"
              :key="cell.key"
              class="relative min-h-[64px] bg-white p-2 dark:bg-dark-800"
              :class="cell.inMonth ? '' : 'text-gray-300 dark:text-dark-600'"
            >
              <span class="text-xs font-medium">{{ cell.day }}</span>
              <span
                v-if="cell.requests > 0"
                class="absolute bottom-2 left-2 right-2 rounded-md px-1.5 py-1 text-center text-[10px] font-semibold text-white"
                :style="{ backgroundColor: calendarCellColor(cell.requests) }"
                :title="t('dashboard.calendarRequests', { count: formatNumber(cell.requests) })"
              >
                {{ formatCompact(cell.requests) }}
              </span>
              <span v-else class="absolute bottom-3 left-1/2 h-1.5 w-1.5 -translate-x-1/2 rounded-full bg-gray-200 dark:bg-dark-600" />
            </div>
          </div>
        </div>
      </div>

      <div v-else-if="activeTab === 'costDistribution'" role="tabpanel">
        <ChartHeading
          :title="t('dashboard.analysisTabs.costDistributionTitle')"
          :summary="t('dashboard.analysisTotalCost', { value: formatCost(rangeCost) })"
        />
        <div v-if="costDistributionData" class="mt-5 h-[340px]">
          <Doughnut :data="costDistributionData" :options="costDoughnutOptions" />
        </div>
        <DashboardChartEmpty v-else />
      </div>

      <div v-else-if="activeTab === 'costTrend'" role="tabpanel">
        <ChartHeading
          :title="t('dashboard.analysisTabs.costTrendTitle')"
          :summary="t('dashboard.analysisTotalCost', { value: formatCost(rangeCost) })"
        />
        <div v-if="costTrendData" class="mt-5 h-[340px]">
          <Line :data="costTrendData" :options="lineOptions" />
        </div>
        <DashboardChartEmpty v-else />
      </div>

      <div v-else-if="activeTab === 'requestDistribution'" role="tabpanel">
        <ChartHeading
          :title="t('dashboard.analysisTabs.requestDistributionTitle')"
          :summary="t('dashboard.analysisTotalRequests', { value: formatNumber(rangeRequests) })"
        />
        <div v-if="requestDistributionData" class="mt-5 h-[340px]">
          <Doughnut :data="requestDistributionData" :options="requestDoughnutOptions" />
        </div>
        <DashboardChartEmpty v-else />
      </div>

      <div v-else role="tabpanel">
        <ChartHeading
          :title="t('dashboard.analysisTabs.requestRankingTitle')"
          :summary="t('dashboard.analysisTotalRequests', { value: formatNumber(rangeRequests) })"
        />
        <div v-if="requestRankingData" class="mt-5 h-[340px]">
          <Bar :data="requestRankingData" :options="rankingOptions" />
        </div>
        <DashboardChartEmpty v-else />
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Bar, Doughnut, Line } from 'vue-chartjs'
import {
  ArcElement,
  BarElement,
  CategoryScale,
  Chart as ChartJS,
  Filler,
  Legend,
  LinearScale,
  LineElement,
  PointElement,
  Tooltip,
} from 'chart.js'
import Icon from '@/components/icons/Icon.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import type { ModelStat, TrendDataPoint } from '@/types'
import {
  formatCostFixed as formatCost,
  formatNumberLocaleString as formatNumber,
  formatTokensK as formatCompact,
} from '@/utils/format'

ChartJS.register(
  ArcElement,
  BarElement,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Legend,
  Filler,
)

type TabId = 'calendar' | 'costDistribution' | 'costTrend' | 'requestDistribution' | 'requestRanking'
type ChartIconName = 'calendar' | 'chart' | 'trendingUp' | 'clock' | 'chartBar'

const props = defineProps<{
  loading: boolean
  startDate: string
  endDate: string
  granularity: string
  trend: TrendDataPoint[]
  models: ModelStat[]
}>()

defineEmits<{
  (event: 'update:startDate', value: string): void
  (event: 'update:endDate', value: string): void
  (event: 'update:granularity', value: string): void
  (event: 'dateRangeChange'): void
  (event: 'granularityChange'): void
  (event: 'refresh'): void
}>()

const { t, locale } = useI18n()
const activeTab = ref<TabId>('costDistribution')

const tabs: Array<{ id: TabId; icon: ChartIconName; labelKey: string }> = [
  { id: 'calendar', icon: 'calendar', labelKey: 'dashboard.analysisTabs.usageCalendar' },
  { id: 'costDistribution', icon: 'chart', labelKey: 'dashboard.analysisTabs.costDistribution' },
  { id: 'costTrend', icon: 'trendingUp', labelKey: 'dashboard.analysisTabs.costTrend' },
  { id: 'requestDistribution', icon: 'clock', labelKey: 'dashboard.analysisTabs.requestDistribution' },
  { id: 'requestRanking', icon: 'chartBar', labelKey: 'dashboard.analysisTabs.requestRanking' },
]

const palette = ['#5b7cfa', '#6fcf97', '#f2c94c', '#bb6bd9', '#56ccf2', '#f2994a', '#eb5757', '#7f8c8d']

const rangeCost = computed(() => props.trend.reduce((sum, point) => sum + (point.actual_cost || 0), 0))
const rangeRequests = computed(() => props.trend.reduce((sum, point) => sum + (point.requests || 0), 0))

const costDistributionModels = computed(() => props.models.filter((model) => model.actual_cost > 0))
const requestDistributionModels = computed(() => props.models.filter((model) => model.requests > 0))

const costDistributionData = computed(() => {
  if (costDistributionModels.value.length === 0) return null
  return {
    labels: costDistributionModels.value.map((model) => model.model),
    datasets: [{
      data: costDistributionModels.value.map((model) => model.actual_cost),
      backgroundColor: costDistributionModels.value.map((_, index) => palette[index % palette.length]),
      borderColor: '#ffffff',
      borderWidth: 2,
      hoverOffset: 6,
    }],
  }
})

const requestDistributionData = computed(() => {
  if (requestDistributionModels.value.length === 0) return null
  return {
    labels: requestDistributionModels.value.map((model) => model.model),
    datasets: [{
      data: requestDistributionModels.value.map((model) => model.requests),
      backgroundColor: requestDistributionModels.value.map((_, index) => palette[index % palette.length]),
      borderColor: '#ffffff',
      borderWidth: 2,
      hoverOffset: 6,
    }],
  }
})

const costTrendData = computed(() => {
  if (props.trend.length === 0) return null
  return {
    labels: props.trend.map((point) => formatTrendLabel(point.date)),
    datasets: [{
      label: t('dashboard.actualCost'),
      data: props.trend.map((point) => point.actual_cost),
      borderColor: '#5b7cfa',
      backgroundColor: 'rgba(91, 124, 250, 0.12)',
      pointBackgroundColor: '#5b7cfa',
      pointBorderColor: '#ffffff',
      pointBorderWidth: 2,
      pointRadius: props.trend.length > 20 ? 0 : 3,
      tension: 0.32,
      fill: true,
    }],
  }
})

const requestRankingData = computed(() => {
  const ranked = [...requestDistributionModels.value]
    .sort((a, b) => b.requests - a.requests || a.model.localeCompare(b.model))
    .slice(0, 10)
  if (ranked.length === 0) return null
  return {
    labels: ranked.map((model) => model.model),
    datasets: [{
      label: t('dashboard.requests'),
      data: ranked.map((model) => model.requests),
      backgroundColor: ranked.map((_, index) => `${palette[index % palette.length]}cc`),
      borderRadius: 5,
      barThickness: 18,
    }],
  }
})

const doughnutBaseOptions = {
  responsive: true,
  maintainAspectRatio: false,
  cutout: '58%',
  plugins: {
    legend: {
      display: true,
      position: 'bottom',
      labels: { usePointStyle: true, pointStyle: 'circle', padding: 18, boxWidth: 7, boxHeight: 7 },
    },
  },
}

const costDoughnutOptions: any = {
  ...doughnutBaseOptions,
  plugins: {
    ...doughnutBaseOptions.plugins,
    tooltip: {
      callbacks: {
        label: (context: any) => `${context.label}: $${formatCost(Number(context.parsed) || 0)}`,
      },
    },
  },
}

const requestDoughnutOptions: any = {
  ...doughnutBaseOptions,
  plugins: {
    ...doughnutBaseOptions.plugins,
    tooltip: {
      callbacks: {
        label: (context: any) => `${context.label}: ${formatNumber(Number(context.parsed) || 0)}`,
      },
    },
  },
}

const lineOptions: any = {
  responsive: true,
  maintainAspectRatio: false,
  interaction: { intersect: false, mode: 'index' },
  plugins: { legend: { display: false } },
  scales: {
    x: { grid: { display: false }, ticks: { maxRotation: 0, autoSkip: true, maxTicksLimit: 8 } },
    y: { beginAtZero: true, grid: { color: 'rgba(148, 163, 184, 0.16)' } },
  },
}

const rankingOptions: any = {
  responsive: true,
  maintainAspectRatio: false,
  indexAxis: 'y',
  plugins: { legend: { display: false } },
  scales: {
    x: { beginAtZero: true, grid: { color: 'rgba(148, 163, 184, 0.16)' } },
    y: { grid: { display: false }, ticks: { autoSkip: false } },
  },
}

const calendarMonth = computed(() => {
  const parsed = new Date(`${props.endDate}T00:00:00`)
  return Number.isFinite(parsed.getTime()) ? parsed : new Date()
})

const calendarMonthLabel = computed(() => calendarMonth.value.toLocaleDateString(
  locale.value.startsWith('zh') ? 'zh-CN' : 'en-US',
  { year: 'numeric', month: 'long' },
))

const weekdays = computed(() => locale.value.startsWith('zh')
  ? ['日', '一', '二', '三', '四', '五', '六']
  : ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'])

const calendarRequestMap = computed(() => props.trend.reduce((result, point) => {
  const date = normaliseTrendDate(point.date)
  result.set(date, (result.get(date) || 0) + (point.requests || 0))
  return result
}, new Map<string, number>()))

const calendarCells = computed(() => {
  const year = calendarMonth.value.getFullYear()
  const month = calendarMonth.value.getMonth()
  const first = new Date(year, month, 1)
  const start = new Date(year, month, 1 - first.getDay())
  return Array.from({ length: 42 }, (_, index) => {
    const date = new Date(start)
    date.setDate(start.getDate() + index)
    const key = formatLocalDate(date)
    return {
      key,
      day: date.getDate(),
      inMonth: date.getMonth() === month,
      requests: calendarRequestMap.value.get(key) || 0,
    }
  })
})

const maxCalendarRequests = computed(() => Math.max(1, ...calendarCells.value.map((cell) => cell.requests)))
const activeCalendarDays = computed(() => calendarCells.value.filter((cell) => cell.inMonth && cell.requests > 0).length)
const calendarMonthRequests = computed(() => calendarCells.value.reduce(
  (total, cell) => total + (cell.inMonth ? cell.requests : 0),
  0,
))

function calendarCellColor(requests: number): string {
  const strength = 0.44 + 0.48 * (requests / maxCalendarRequests.value)
  return `rgba(91, 124, 250, ${strength.toFixed(2)})`
}

function formatTrendLabel(value: string): string {
  const date = new Date(/^\d{4}-\d{2}-\d{2}$/.test(value) ? `${value}T00:00:00` : value)
  if (!Number.isFinite(date.getTime())) return value
  return date.toLocaleDateString(locale.value.startsWith('zh') ? 'zh-CN' : 'en-US', {
    month: '2-digit',
    day: '2-digit',
    ...(props.granularity === 'hour' ? { hour: '2-digit' } : {}),
  })
}

function normaliseTrendDate(value: string): string {
  const direct = value.match(/^\d{4}-\d{2}-\d{2}/)?.[0]
  if (direct) return direct
  const parsed = new Date(value)
  return Number.isFinite(parsed.getTime()) ? formatLocalDate(parsed) : value
}

function formatLocalDate(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

const ChartHeading = defineComponent({
  name: 'DashboardChartHeading',
  props: {
    title: { type: String, required: true },
    summary: { type: String, required: true },
  },
  setup(componentProps) {
    return () => h('div', {}, [
      h('h3', { class: 'text-base font-semibold text-gray-900 dark:text-white' }, componentProps.title),
      h('p', { class: 'mt-1 text-sm text-gray-400 dark:text-dark-500' }, componentProps.summary),
    ])
  },
})

const DashboardChartEmpty = defineComponent({
  name: 'DashboardChartEmpty',
  setup() {
    return () => h('div', {
      class: 'mt-5 flex h-[340px] flex-col items-center justify-center gap-3 text-center text-sm text-gray-400 dark:text-dark-500',
    }, [
      h(Icon, { name: 'chart', size: 'lg', strokeWidth: 1.5 }),
      h('span', t('dashboard.noDataAvailable')),
    ])
  },
})
</script>

<style scoped>
.dashboard-panel {
  border: 1px solid rgb(229 231 235 / 0.78);
  border-radius: 16px;
  background: rgb(255 255 255);
  box-shadow: 0 0 1px rgb(15 23 42 / 0.16), 0 7px 18px rgb(15 23 42 / 0.07);
}

.analysis-tab {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  border-radius: 10px;
  padding: 0.5rem 0.65rem;
  color: rgb(107 114 128);
  font-size: 0.75rem;
  font-weight: 500;
  line-height: 1rem;
  transition: background-color 160ms ease, color 160ms ease;
}

.analysis-tab:hover {
  color: rgb(55 65 81);
  background: rgb(249 250 251);
}

.analysis-tab-active {
  color: rgb(79 105 224);
  background: rgb(239 243 255);
}

:global(.dark) .dashboard-panel {
  border-color: rgb(51 65 85 / 0.86);
  background: rgb(30 41 59);
  box-shadow: 0 0 1px rgb(0 0 0 / 0.45), 0 7px 20px rgb(0 0 0 / 0.2);
}

:global(.dark) .analysis-tab {
  color: rgb(148 163 184);
}

:global(.dark) .analysis-tab:hover {
  color: rgb(226 232 240);
  background: rgb(15 23 42 / 0.55);
}

:global(.dark) .analysis-tab-active {
  color: rgb(165 180 252);
  background: rgb(49 46 129 / 0.38);
}
</style>
