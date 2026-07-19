<template>
  <section
    class="grid grid-cols-1 gap-4 md:grid-cols-2"
    :class="isSimple ? 'xl:grid-cols-3' : 'xl:grid-cols-4'"
    data-testid="dashboard-metric-grid"
    :aria-label="t('dashboard.accountMetrics')"
  >
    <article v-if="!isSimple" class="dashboard-panel min-h-[205px] overflow-hidden">
      <DashboardCardHeader icon="wallet" :title="t('dashboard.accountData')" />
      <div class="space-y-4 px-5 py-5">
        <DashboardMetricRow
          icon="arrowLeftRight"
          tone="blue"
          :label="t('dashboard.currentBalance')"
          :value="formatBalance(balance)"
          credit
        />
        <DashboardMetricRow
          icon="chartNoAxesColumn"
          tone="violet"
          :label="t('dashboard.lifetimeSpend')"
          :value="formatCost(stats.total_actual_cost)"
          credit
        />
      </div>
    </article>

    <article class="dashboard-panel min-h-[205px] overflow-hidden">
      <DashboardCardHeader icon="activity" :title="t('dashboard.usageStatistics')" />
      <div class="space-y-4 px-5 py-5">
        <DashboardMetricRow
          icon="send"
          tone="emerald"
          :label="t('dashboard.lifetimeRequests')"
          :value="formatNumber(stats.total_requests)"
        />
        <DashboardMetricRow
          icon="activity"
          tone="sky"
          :label="t('dashboard.rangeRequests')"
          :value="formatNumber(rangeMetrics.requests)"
          :sparkline-values="sparklineSeries.requests"
          sparkline-color="#06b6d4"
        />
      </div>
    </article>

    <article class="dashboard-panel min-h-[205px] overflow-hidden">
      <DashboardCardHeader icon="zap" :title="t('dashboard.resourceUsage')" />
      <div class="space-y-4 px-5 py-5">
        <DashboardMetricRow
          icon="coins"
          tone="amber"
          :label="t('dashboard.rangeSpend')"
          :value="formatCost(rangeMetrics.actualCost)"
          :sparkline-values="sparklineSeries.actualCost"
          sparkline-color="#f59e0b"
          credit
        />
        <DashboardMetricRow
          icon="type"
          tone="rose"
          :label="t('dashboard.rangeTokens')"
          :value="formatTokens(rangeMetrics.tokens)"
          :sparkline-values="sparklineSeries.tokens"
          sparkline-color="#ec4899"
        />
      </div>
    </article>

    <article class="dashboard-panel min-h-[205px] overflow-hidden">
      <DashboardCardHeader icon="gauge" :title="t('dashboard.performance')" />
      <div class="space-y-4 px-5 py-5">
        <DashboardMetricRow
          icon="timer"
          tone="indigo"
          :label="t('dashboard.averageRpm')"
          :value="formatRate(rangeMetrics.averageRpm)"
          :sparkline-values="sparklineSeries.requests"
          sparkline-color="#6366f1"
        />
        <DashboardMetricRow
          icon="send"
          tone="orange"
          :label="t('dashboard.averageTpm')"
          :value="formatRate(rangeMetrics.averageTpm)"
          :sparkline-values="sparklineSeries.tokens"
          sparkline-color="#f97316"
        />
      </div>
    </article>
  </section>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, type PropType } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import CreditAmount from '@/components/common/CreditAmount.vue'
import DashboardSparkline from '@/components/user/dashboard/DashboardSparkline.vue'
import type { UserDashboardStats as UserStatsType } from '@/api/usage'
import type { TrendDataPoint } from '@/types'
import type { DashboardRangeMetrics } from '@/utils/dashboardMetrics'
import { buildDashboardSparklineSeries } from '@/utils/dashboardMetrics'
import {
  formatCostFixed as formatCost,
  formatNumberLocaleString as formatNumber,
  formatTokensK as formatTokens,
} from '@/utils/format'

const props = defineProps<{
  stats: UserStatsType
  balance: number
  isSimple: boolean
  rangeMetrics: DashboardRangeMetrics
  trend: TrendDataPoint[]
  startDate: string
  endDate: string
  granularity: 'day' | 'hour'
}>()

const { t } = useI18n()
const sparklineSeries = computed(() => buildDashboardSparklineSeries(
  props.trend,
  props.startDate,
  props.endDate,
  props.granularity,
))

type DashboardIconName =
  | 'wallet'
  | 'activity'
  | 'zap'
  | 'gauge'
  | 'arrowLeftRight'
  | 'chartNoAxesColumn'
  | 'send'
  | 'coins'
  | 'type'
  | 'timer'

const toneClasses = {
  blue: 'bg-blue-50 text-blue-500 dark:bg-blue-950/40 dark:text-blue-300',
  violet: 'bg-violet-50 text-violet-500 dark:bg-violet-950/40 dark:text-violet-300',
  emerald: 'bg-emerald-50 text-emerald-500 dark:bg-emerald-950/40 dark:text-emerald-300',
  sky: 'bg-sky-50 text-sky-500 dark:bg-sky-950/40 dark:text-sky-300',
  amber: 'bg-amber-50 text-amber-500 dark:bg-amber-950/40 dark:text-amber-300',
  rose: 'bg-pink-50 text-pink-500 dark:bg-pink-950/40 dark:text-pink-300',
  indigo: 'bg-indigo-50 text-indigo-500 dark:bg-indigo-950/40 dark:text-indigo-300',
  orange: 'bg-orange-50 text-orange-500 dark:bg-orange-950/40 dark:text-orange-300',
} as const

const DashboardCardHeader = defineComponent({
  name: 'DashboardCardHeader',
  props: {
    icon: { type: String as PropType<DashboardIconName>, required: true },
    title: { type: String, required: true },
  },
  setup(props) {
    return () => h('div', {
      class: 'flex h-[61px] items-center gap-2 border-b border-gray-100 px-5 text-sm font-medium text-gray-800 dark:border-dark-700 dark:text-gray-100',
    }, [
      h(Icon, { name: props.icon, size: 'sm', strokeWidth: 2 }),
      h('span', props.title),
    ])
  },
})

const DashboardMetricRow = defineComponent({
  name: 'DashboardMetricRow',
  props: {
    icon: { type: String as PropType<DashboardIconName>, required: true },
    tone: { type: String as PropType<keyof typeof toneClasses>, required: true },
    label: { type: String, required: true },
    value: { type: String, required: true },
    credit: { type: Boolean, default: false },
    sparklineValues: { type: Array as PropType<number[]>, default: () => [] },
    sparklineColor: { type: String, default: '' },
  },
  setup(props) {
    return () => h('div', { class: 'flex min-w-0 items-center' }, [
      h('span', {
        class: `mr-3 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg ${toneClasses[props.tone]}`,
      }, [h(Icon, { name: props.icon, size: 'sm', strokeWidth: 2 })]),
      h('span', { class: 'min-w-0 flex-1' }, [
        h('span', { class: 'block text-xs leading-4 text-gray-500 dark:text-dark-400' }, props.label),
        h('strong', {
          class: 'mt-0.5 block truncate text-lg font-semibold leading-6 text-gray-800 dark:text-gray-100',
          title: props.value,
        }, props.credit
          ? [h(CreditAmount, {
              value: props.value,
              iconSize: 'md',
              label: `${props.label} ${props.value}`,
            })]
          : props.value),
      ]),
      props.sparklineColor
        ? h(DashboardSparkline, {
            class: 'ml-3',
            values: props.sparklineValues,
            color: props.sparklineColor,
          })
        : null,
    ])
  },
})

function formatRate(value: number): string {
  if (!Number.isFinite(value)) return '0.000'
  if (value >= 1000) return formatTokens(value)
  return value.toFixed(3)
}

function formatBalance(value: number): string {
  if (!Number.isFinite(value)) return '0.00'
  return new Intl.NumberFormat('en-US', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(value)
}
</script>

<style scoped>
.dashboard-panel {
  border: 1px solid rgb(229 231 235 / 0.78);
  border-radius: 16px;
  background: rgb(255 255 255);
  box-shadow: 0 0 1px rgb(15 23 42 / 0.16), 0 7px 18px rgb(15 23 42 / 0.07);
}

:global(.dark) .dashboard-panel {
  border-color: rgb(51 65 85 / 0.86);
  background: rgb(30 41 59);
  box-shadow: 0 0 1px rgb(0 0 0 / 0.45), 0 7px 20px rgb(0 0 0 / 0.2);
}
</style>
