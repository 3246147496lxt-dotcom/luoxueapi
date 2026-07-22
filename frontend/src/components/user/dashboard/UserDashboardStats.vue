<template>
  <section
    class="dashboard-metric-grid grid grid-cols-2 gap-3 sm:gap-4 md:grid-cols-2"
    :class="isSimple ? 'xl:grid-cols-3' : 'xl:grid-cols-4'"
    data-testid="dashboard-metric-grid"
    :aria-label="t('dashboard.accountMetrics')"
  >
    <article v-if="!isSimple" class="dashboard-panel dashboard-summary-card min-h-[205px] overflow-hidden">
      <DashboardCardHeader icon="wallet" :title="t('dashboard.accountData')" />
      <div class="dashboard-summary-content space-y-4 px-5 py-5">
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

    <article class="dashboard-panel dashboard-summary-card min-h-[205px] overflow-hidden">
      <DashboardCardHeader icon="activity" :title="t('dashboard.usageStatistics')" />
      <div class="dashboard-summary-content space-y-4 px-5 py-5">
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
          sparkline-color="#0b8bed"
        />
      </div>
    </article>

    <article class="dashboard-panel dashboard-summary-card min-h-[205px] overflow-hidden">
      <DashboardCardHeader icon="zap" :title="t('dashboard.resourceUsage')" />
      <div class="dashboard-summary-content space-y-4 px-5 py-5">
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

    <article class="dashboard-panel dashboard-summary-card min-h-[205px] overflow-hidden">
      <DashboardCardHeader icon="gauge" :title="t('dashboard.performance')" />
      <div class="dashboard-summary-content space-y-4 px-5 py-5">
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
      class: 'dashboard-summary-card-header flex h-[61px] items-center gap-2 border-b border-gray-100 px-5 text-sm font-medium text-gray-800 dark:border-dark-700 dark:text-gray-100',
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
    return () => h('div', { class: 'dashboard-metric-row flex min-w-0 items-center' }, [
      h('span', {
        class: `dashboard-metric-row-icon mr-3 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg ${toneClasses[props.tone]}`,
      }, [h(Icon, { name: props.icon, size: 'sm', strokeWidth: 2 })]),
      h('span', { class: 'dashboard-metric-row-copy min-w-0 flex-1' }, [
        h('span', { class: 'block text-xs leading-4 text-gray-500 dark:text-dark-400' }, props.label),
        h('strong', {
          class: 'dashboard-metric-row-value mt-0.5 block truncate text-lg font-semibold leading-6 text-gray-800 dark:text-gray-100',
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
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-surface);
  background: var(--lx-clay-surface);
  box-shadow: var(--lx-clay-shadow-form);
}

@media (max-width: 639px) {
  .dashboard-summary-card-header {
    gap: 0.375rem;
    padding-right: 0.75rem;
    padding-left: 0.75rem;
  }

  .dashboard-summary-content {
    padding: 0.875rem 0.75rem;
  }

  .dashboard-metric-row-icon {
    width: 1.75rem;
    height: 1.75rem;
    margin-right: 0.5rem;
  }

  .dashboard-metric-row-value {
    font-size: 1rem;
    line-height: 1.375rem;
  }

  .dashboard-metric-row :deep(.dashboard-sparkline) {
    width: 2.375rem;
    height: 1.75rem;
    margin-left: 0.25rem;
    flex-basis: 2.375rem;
  }
}

@media (max-width: 359px) {
  .dashboard-metric-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
