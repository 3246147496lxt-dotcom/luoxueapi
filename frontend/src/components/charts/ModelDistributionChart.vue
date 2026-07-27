<template>
  <div
    class="card h-full p-5 md:p-6"
    :class="{ 'model-distribution--home-clay': isHomeClay }"
    :aria-busy="loading || (activeView === 'spending_ranking' && rankingLoading)"
  >
    <div class="model-distribution-heading mb-5 flex items-center justify-between gap-3">
      <div>
        <span v-if="isHomeClay" class="home-chart-eyebrow">{{ t('admin.dashboard.modelMixEyebrow') }}</span>
        <h3 class="text-base font-semibold text-gray-900 dark:text-white">
          {{ !enableRankingView || activeView === 'model_distribution'
            ? (isHomeClay ? t('admin.dashboard.modelUsageDistribution') : t('admin.dashboard.modelDistribution'))
            : t('admin.dashboard.spendingRankingTitle') }}
        </h3>
      </div>
      <div class="flex flex-wrap items-center justify-end gap-2">
        <div
          v-if="showSourceToggle"
          class="chart-toggle-group inline-flex rounded-lg border border-gray-200 bg-gray-50 p-0.5 dark:border-gray-700 dark:bg-dark-800"
          role="group"
          :aria-label="t('admin.dashboard.model')"
        >
          <button
            type="button"
            class="chart-toggle-button rounded-md px-2.5 py-1 text-xs font-medium transition-colors"
            :class="source === 'requested'
              ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
              : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200'"
            :aria-pressed="source === 'requested'"
            @click="emit('update:source', 'requested')"
          >
            {{ t('usage.requestedModel') }}
          </button>
          <button
            type="button"
            class="chart-toggle-button rounded-md px-2.5 py-1 text-xs font-medium transition-colors"
            :class="source === 'upstream'
              ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
              : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200'"
            :aria-pressed="source === 'upstream'"
            @click="emit('update:source', 'upstream')"
          >
            {{ t('usage.upstreamModel') }}
          </button>
          <button
            type="button"
            class="chart-toggle-button rounded-md px-2.5 py-1 text-xs font-medium transition-colors"
            :class="source === 'mapping'
              ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
              : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200'"
            :aria-pressed="source === 'mapping'"
            @click="emit('update:source', 'mapping')"
          >
            {{ t('usage.mapping') }}
          </button>
        </div>
        <div
          v-if="showMetricToggle"
          class="chart-toggle-group inline-flex rounded-lg border border-gray-200 bg-gray-50 p-0.5 dark:border-gray-700 dark:bg-dark-800"
          role="group"
          :aria-label="t('admin.dashboard.modelDistribution')"
        >
          <button
            type="button"
            class="chart-toggle-button rounded-md px-2.5 py-1 text-xs font-medium transition-colors"
            :class="metric === 'tokens'
              ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
              : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200'"
            :aria-pressed="metric === 'tokens'"
            @click="emit('update:metric', 'tokens')"
          >
            {{ t('admin.dashboard.metricTokens') }}
          </button>
          <button
            type="button"
            class="chart-toggle-button rounded-md px-2.5 py-1 text-xs font-medium transition-colors"
            :class="metric === 'actual_cost'
              ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
              : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200'"
            :aria-pressed="metric === 'actual_cost'"
            @click="emit('update:metric', 'actual_cost')"
          >
            {{ t('admin.dashboard.metricActualCost') }}
          </button>
        </div>
        <div
          v-if="enableRankingView"
          class="chart-toggle-group inline-flex rounded-lg bg-gray-100 p-1 dark:bg-dark-800"
          role="group"
          :aria-label="t('admin.dashboard.modelDistribution')"
        >
          <button
            type="button"
            class="chart-toggle-button rounded-md px-2.5 py-1 text-xs font-medium transition-colors"
            :class="
              activeView === 'model_distribution'
                ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
                : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200'
            "
            :aria-pressed="activeView === 'model_distribution'"
            @click="activeView = 'model_distribution'"
          >
            {{ t('admin.dashboard.viewModelDistribution') }}
          </button>
          <button
            type="button"
            class="chart-toggle-button rounded-md px-2.5 py-1 text-xs font-medium transition-colors"
            :class="
              activeView === 'spending_ranking'
                ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
                : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200'
            "
            :aria-pressed="activeView === 'spending_ranking'"
            @click="activeView = 'spending_ranking'"
          >
            {{ t('admin.dashboard.viewSpendingRanking') }}
          </button>
        </div>
      </div>
    </div>

    <template v-if="activeView === 'model_distribution'">
      <div v-if="isHomeClay" class="home-model-content">
        <div
          v-if="loading && !homeDistributionHasData"
          class="home-model-skeleton"
          role="status"
          aria-live="polite"
        >
          <span class="sr-only">{{ t('common.loading') }}</span>
          <div class="home-model-skeleton-ring" aria-hidden="true"></div>
          <div class="home-model-skeleton-list" aria-hidden="true">
            <span v-for="index in 4" :key="index"></span>
          </div>
        </div>

        <div
          v-else-if="hasDistributionError && !homeDistributionHasData"
          class="home-model-state"
          role="alert"
        >
          <Icon name="exclamationTriangle" size="lg" aria-hidden="true" />
          <p>{{ distributionErrorMessage }}</p>
          <button type="button" @click="emit('retry')">{{ t('admin.dashboard.retry') }}</button>
        </div>

        <div v-else-if="homeDistributionHasData" class="home-model-data">
          <p v-if="hasDistributionError" class="home-model-inline-error" role="alert">
            {{ distributionErrorMessage }}
          </p>
          <span v-if="loading" class="sr-only" role="status" aria-live="polite">
            {{ t('common.loading') }}
          </span>

          <div class="home-model-layout" :class="{ 'is-updating': loading }">
            <div
              class="home-model-visual"
              role="img"
              :aria-label="distributionAccessibilityLabel"
            >
              <Doughnut
                :data="safeChartData"
                :options="doughnutOptions"
                aria-hidden="true"
              />
              <div class="home-model-total" aria-hidden="true">
                <span>{{ distributionTotalLabel }}</span>
                <strong>
                  <CreditAmount
                    v-if="metric === 'actual_cost' && creditMode"
                    :value="formatCost(distributionTotal)"
                    icon-size="sm"
                  />
                  <template v-else>{{ distributionTotalDisplay }}</template>
                </strong>
              </div>
            </div>

            <ol class="home-model-legend" :aria-label="t('admin.dashboard.modelDistribution')">
              <li
                v-for="item in homeModelLegend"
                :key="item.model.model"
                :style="{ '--model-color': item.color }"
              >
                <button
                  v-if="enableBreakdown"
                  type="button"
                  class="home-model-row"
                  :aria-expanded="expandedKey === `model-${item.model.model}`"
                  @click="toggleBreakdown('model', item.model.model)"
                >
                  <span class="home-model-dot" aria-hidden="true"></span>
                  <span class="home-model-copy">
                    <strong :title="item.model.model">{{ item.model.model }}</strong>
                    <small>
                      <CreditAmount
                        v-if="metric === 'actual_cost' && creditMode"
                        :value="formatCost(item.value)"
                        icon-size="xs"
                      />
                      <template v-else>{{ item.valueLabel }}</template>
                      · {{ formatNumber(item.model.requests) }} {{ t('admin.dashboard.requestsShort') }}
                    </small>
                  </span>
                  <span class="home-model-share">{{ item.percentageLabel }}</span>
                  <Icon
                    name="chevronRight"
                    size="sm"
                    class="home-model-chevron"
                    :class="{ expanded: expandedKey === `model-${item.model.model}` }"
                    aria-hidden="true"
                  />
                </button>
                <div v-else class="home-model-row is-static">
                  <span class="home-model-dot" aria-hidden="true"></span>
                  <span class="home-model-copy">
                    <strong :title="item.model.model">{{ item.model.model }}</strong>
                    <small>
                      <CreditAmount
                        v-if="metric === 'actual_cost' && creditMode"
                        :value="formatCost(item.value)"
                        icon-size="xs"
                      />
                      <template v-else>{{ item.valueLabel }}</template>
                      · {{ formatNumber(item.model.requests) }} {{ t('admin.dashboard.requestsShort') }}
                    </small>
                  </span>
                  <span class="home-model-share">{{ item.percentageLabel }}</span>
                </div>
              </li>
            </ol>
          </div>

          <div v-if="expandedModel" class="home-model-breakdown">
            <div class="home-model-breakdown-heading">
              <strong :title="expandedModel">{{ expandedModel }}</strong>
              <span>{{ t('admin.dashboard.spendingRankingUsage') }}</span>
            </div>
            <UserBreakdownSubTable
              :items="breakdownItems"
              :loading="breakdownLoading"
              :show-account-cost="showAccountCost"
              :credit-mode="creditMode"
            />
          </div>
        </div>

        <div v-else class="home-model-state" role="status">
          <Icon name="clock" size="lg" aria-hidden="true" />
          <p>{{ t('admin.dashboard.noDataAvailable') }}</p>
        </div>
      </div>

      <template v-else>
        <div v-if="loading" class="flex h-48 items-center justify-center">
          <LoadingSpinner />
        </div>
        <div v-else-if="displayModelStats.length > 0 && chartData" class="flex items-center gap-6">
          <div class="h-48 w-48">
            <Doughnut :data="chartData" :options="doughnutOptions" />
          </div>
          <div class="max-h-48 flex-1 overflow-y-auto">
            <table class="w-full text-xs">
              <thead>
                <tr class="text-gray-500 dark:text-gray-400">
                  <th class="pb-2 text-left">{{ t('admin.dashboard.model') }}</th>
                  <th class="pb-2 text-right">{{ t('admin.dashboard.requests') }}</th>
                  <th class="pb-2 text-right">{{ t('admin.dashboard.tokens') }}</th>
                  <th class="pb-2 text-right">{{ t('admin.dashboard.actual') }}</th>
                  <th v-if="showAccountCost" class="pb-2 text-right">{{ t('admin.dashboard.accountCost') }}</th>
                  <th class="pb-2 text-right">{{ t('admin.dashboard.standard') }}</th>
                </tr>
              </thead>
              <tbody>
                <template v-for="model in displayModelStats" :key="model.model">
                  <tr
                    class="border-t border-gray-100 transition-colors dark:border-gray-700"
                    :class="enableBreakdown ? 'cursor-pointer hover:bg-gray-50 dark:hover:bg-dark-700/40' : ''"
                    @click="enableBreakdown && toggleBreakdown('model', model.model)"
                  >
                    <td
                      class="max-w-[100px] truncate py-1.5 font-medium"
                      :class="enableBreakdown ? 'text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300' : 'text-gray-900 dark:text-white'"
                      :title="model.model"
                    >
                      <span class="inline-flex items-center gap-1">
                        <Icon v-if="enableBreakdown && expandedKey === `model-${model.model}`" name="chevronDown" size="xs" class="shrink-0" aria-hidden="true" />
                        <Icon v-else-if="enableBreakdown" name="chevronRight" size="xs" class="shrink-0" aria-hidden="true" />
                        {{ model.model }}
                      </span>
                    </td>
                    <td class="py-1.5 text-right text-gray-600 dark:text-gray-400">
                      {{ formatNumber(model.requests) }}
                    </td>
                    <td class="py-1.5 text-right text-gray-600 dark:text-gray-400">
                      {{ formatTokens(model.total_tokens) }}
                    </td>
                    <td class="py-1.5 text-right text-green-600 dark:text-green-400">
                      <CreditAmount
                        v-if="creditMode"
                        :value="formatCost(model.actual_cost)"
                        icon-size="xs"
                      />
                      <template v-else>${{ formatCost(model.actual_cost) }}</template>
                    </td>
                    <td v-if="showAccountCost" class="py-1.5 text-right text-orange-500 dark:text-orange-400">
                      ${{ formatCost(model.account_cost) }}
                    </td>
                    <td class="py-1.5 text-right text-gray-400 dark:text-gray-500">
                      ${{ formatCost(model.cost) }}
                    </td>
                  </tr>
                  <tr v-if="expandedKey === `model-${model.model}`">
                    <td :colspan="distributionColspan" class="p-0">
                      <UserBreakdownSubTable
                        :items="breakdownItems"
                        :loading="breakdownLoading"
                        :show-account-cost="showAccountCost"
                        :credit-mode="creditMode"
                      />
                    </td>
                  </tr>
                </template>
              </tbody>
            </table>
          </div>
        </div>
        <div v-else class="flex h-48 items-center justify-center text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.dashboard.noDataAvailable') }}
        </div>
      </template>
    </template>

    <div v-else-if="rankingLoading" class="flex h-48 items-center justify-center">
      <LoadingSpinner />
    </div>
    <div
      v-else-if="rankingError"
      class="flex h-48 items-center justify-center text-sm text-gray-500 dark:text-gray-400"
    >
      {{ t('admin.dashboard.failedToLoad') }}
    </div>
    <div v-else-if="rankingDisplayItems.length > 0 && rankingChartData" class="flex items-center gap-6">
      <div class="h-48 w-48">
        <Doughnut :data="rankingChartData" :options="rankingDoughnutOptions" />
      </div>
      <div class="max-h-48 flex-1 overflow-y-auto">
        <table class="w-full text-xs">
          <thead>
            <tr class="text-gray-500 dark:text-gray-400">
              <th class="pb-2 text-left">{{ t('admin.dashboard.spendingRankingUser') }}</th>
              <th class="pb-2 text-right">{{ t('admin.dashboard.spendingRankingRequests') }}</th>
              <th class="pb-2 text-right">{{ t('admin.dashboard.spendingRankingTokens') }}</th>
              <th class="pb-2 text-right">{{ t('admin.dashboard.spendingRankingSpend') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="(item, index) in rankingDisplayItems"
              :key="item.isOther ? 'others' : `${item.user_id}-${index}`"
              class="border-t border-gray-100 transition-colors dark:border-gray-700"
              :class="item.isOther
                ? 'bg-gray-50/70 dark:bg-dark-700/20'
                : 'cursor-pointer hover:bg-gray-50 dark:hover:bg-dark-700/40'"
              @click="item.isOther ? undefined : emit('ranking-click', item)"
            >
              <td class="py-1.5">
                <div class="flex min-w-0 items-center gap-2">
                  <span class="shrink-0 text-[11px] font-semibold text-gray-500 dark:text-gray-400">
                    {{ item.isOther ? 'Σ' : `#${index + 1}` }}
                  </span>
                  <span
                    class="block max-w-[140px] truncate font-medium text-gray-900 dark:text-white"
                    :title="getRankingRowLabel(item)"
                  >
                    {{ getRankingRowLabel(item) }}
                  </span>
                </div>
              </td>
              <td class="py-1.5 text-right text-gray-600 dark:text-gray-400">
                {{ formatNumber(item.requests) }}
              </td>
              <td class="py-1.5 text-right text-gray-600 dark:text-gray-400">
                {{ formatTokens(item.tokens) }}
              </td>
              <td class="py-1.5 text-right text-green-600 dark:text-green-400">
                <CreditAmount
                  v-if="creditMode"
                  :value="formatCost(item.actual_cost)"
                  icon-size="xs"
                />
                <template v-else>${{ formatCost(item.actual_cost) }}</template>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
    <div
      v-else
      class="flex h-48 items-center justify-center text-sm text-gray-500 dark:text-gray-400"
    >
      {{ t('admin.dashboard.noDataAvailable') }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Chart as ChartJS, ArcElement, Tooltip, Legend } from 'chart.js'
import { Doughnut } from 'vue-chartjs'
import CreditAmount from '@/components/common/CreditAmount.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'
import UserBreakdownSubTable from './UserBreakdownSubTable.vue'
import type { ModelStat, UserSpendingRankingItem, UserBreakdownItem } from '@/types'
import { getUserBreakdown } from '@/api/admin/dashboard'
import { LUOXUE_CLAY_CHART_CATEGORICAL } from '@/utils/luoxueClayChartTheme'

ChartJS.register(ArcElement, Tooltip, Legend)

const { t } = useI18n()

type DistributionMetric = 'tokens' | 'actual_cost'
type ModelSource = 'requested' | 'upstream' | 'mapping'
type RankingDisplayItem = UserSpendingRankingItem & { isOther?: boolean }
const props = withDefaults(defineProps<{
  modelStats: ModelStat[]
  upstreamModelStats?: ModelStat[]
  mappingModelStats?: ModelStat[]
  source?: ModelSource
  enableRankingView?: boolean
  rankingItems?: UserSpendingRankingItem[]
  rankingTotalActualCost?: number
  rankingTotalRequests?: number
  rankingTotalTokens?: number
  loading?: boolean
  metric?: DistributionMetric
  showSourceToggle?: boolean
  showMetricToggle?: boolean
  enableBreakdown?: boolean
  showAccountCost?: boolean
  rankingLoading?: boolean
  rankingError?: boolean
  error?: boolean | string | null
  startDate?: string
  endDate?: string
  filters?: Record<string, any>
  variant?: 'default' | 'home-clay'
  creditMode?: boolean
}>(), {
  upstreamModelStats: () => [],
  mappingModelStats: () => [],
  source: 'requested',
  enableRankingView: false,
  rankingItems: () => [],
  rankingTotalActualCost: 0,
  rankingTotalRequests: 0,
  rankingTotalTokens: 0,
  loading: false,
  metric: 'tokens',
  showSourceToggle: false,
  showMetricToggle: false,
  enableBreakdown: true,
  showAccountCost: true,
  rankingLoading: false,
  rankingError: false,
  error: false,
  variant: 'default',
  creditMode: false,
})

const expandedKey = ref<string | null>(null)
const breakdownItems = ref<UserBreakdownItem[]>([])
const breakdownLoading = ref(false)
let breakdownRequestSeq = 0

const toggleBreakdown = async (type: string, id: string) => {
  const key = `${type}-${id}`
  if (expandedKey.value === key) {
    breakdownRequestSeq += 1
    expandedKey.value = null
    breakdownItems.value = []
    breakdownLoading.value = false
    return
  }
  const requestSeq = ++breakdownRequestSeq
  expandedKey.value = key
  breakdownLoading.value = true
  breakdownItems.value = []
  try {
    const res = await getUserBreakdown({
      ...props.filters,
      start_date: props.startDate,
      end_date: props.endDate,
      model: id,
      model_source: props.source,
    })
    if (requestSeq !== breakdownRequestSeq || expandedKey.value !== key) return
    breakdownItems.value = res.users || []
  } catch {
    if (requestSeq !== breakdownRequestSeq || expandedKey.value !== key) return
    breakdownItems.value = []
  } finally {
    if (requestSeq === breakdownRequestSeq) breakdownLoading.value = false
  }
}

const emit = defineEmits<{
  'update:metric': [value: DistributionMetric]
  'update:source': [value: ModelSource]
  'ranking-click': [item: UserSpendingRankingItem]
  retry: []
}>()

const enableRankingView = computed(() => props.enableRankingView)
const showAccountCost = computed(() => props.showAccountCost)
const distributionColspan = computed(() => showAccountCost.value ? 6 : 5)
const activeView = ref<'model_distribution' | 'spending_ranking'>('model_distribution')
const isHomeClay = computed(() => props.variant === 'home-clay')
const creditMode = computed(() => props.creditMode)

const chartColors = computed(() => [...LUOXUE_CLAY_CHART_CATEGORICAL])

const displayModelStats = computed(() => {
  const sourceStats = props.source === 'upstream'
    ? props.upstreamModelStats
    : props.source === 'mapping'
      ? props.mappingModelStats
      : props.modelStats
  if (!sourceStats?.length) return []

  const metricKey = props.metric === 'actual_cost' ? 'actual_cost' : 'total_tokens'
  return [...sourceStats].sort((a, b) => toFiniteNumber(b[metricKey]) - toFiniteNumber(a[metricKey]))
})

const chartModelStats = computed(() => displayModelStats.value.filter((model) => {
  const value = props.metric === 'actual_cost' ? model.actual_cost : model.total_tokens
  return toFiniteNumber(value) > 0
}))

const getChartColor = (index: number): string => chartColors.value[index % chartColors.value.length] ?? '#94a3b8'

const chartData = computed(() => {
  if (!chartModelStats.value.length) return null

  return {
    labels: chartModelStats.value.map((m) => m.model),
    datasets: [
      {
        data: chartModelStats.value.map((m) => toFiniteNumber(props.metric === 'actual_cost' ? m.actual_cost : m.total_tokens)),
        backgroundColor: chartModelStats.value.map((_, index) => getChartColor(index)),
        borderWidth: 0,
        borderRadius: isHomeClay.value ? 5 : 0,
        hoverOffset: isHomeClay.value ? 4 : 0,
        spacing: isHomeClay.value ? 2 : 0,
      }
    ]
  }
})

const safeChartData = computed(() => chartData.value ?? {
  labels: [],
  datasets: [{ data: [], backgroundColor: [] }]
})

const distributionTotal = computed(() => chartModelStats.value.reduce((sum, model) => {
  const value = props.metric === 'actual_cost' ? model.actual_cost : model.total_tokens
  return sum + Math.max(toFiniteNumber(value), 0)
}, 0))

const distributionTotalLabel = computed(() => props.metric === 'actual_cost'
  ? t('admin.dashboard.totalCost')
  : t('admin.dashboard.totalTokens'))

const distributionTotalDisplay = computed(() => props.metric === 'actual_cost'
  ? formatActualCostText(distributionTotal.value)
  : formatTokens(distributionTotal.value))

const formatPercentage = (value: number): string => {
  if (value >= 10) return `${value.toFixed(0)}%`
  if (value >= 0.1) return `${value.toFixed(1)}%`
  return value > 0 ? '<0.1%' : '0%'
}

const homeModelLegend = computed(() => chartModelStats.value.map((model, index) => {
  const value = toFiniteNumber(props.metric === 'actual_cost' ? model.actual_cost : model.total_tokens)
  const percentage = distributionTotal.value > 0 ? (value / distributionTotal.value) * 100 : 0
  return {
    model,
    color: getChartColor(index),
    value,
    valueLabel: props.metric === 'actual_cost' ? formatActualCostText(value) : `${formatTokens(value)} Token`,
    percentage,
    percentageLabel: formatPercentage(percentage),
  }
}))

const homeDistributionHasData = computed(() => Boolean(chartData.value && distributionTotal.value > 0))
const hasDistributionError = computed(() => props.error === true || (typeof props.error === 'string' && props.error.trim().length > 0))
const distributionErrorMessage = computed(() => typeof props.error === 'string' && props.error.trim()
  ? props.error
  : t('admin.dashboard.failedToLoad'))
const expandedModel = computed(() => chartModelStats.value.find(
  (model) => expandedKey.value === `model-${model.model}`
)?.model ?? null)
const distributionAccessibilityLabel = computed(() => {
  const entries = homeModelLegend.value
    .map((item) => `${item.model.model}: ${item.valueLabel}, ${item.percentageLabel}`)
    .join('; ')
  return `${t('admin.dashboard.modelDistribution')}. ${distributionTotalLabel.value}: ${distributionTotalDisplay.value}. ${entries}`
})

const rankingChartData = computed(() => {
  if (!props.rankingItems?.length) return null

  const labels = props.rankingItems.map((item, index) => `#${index + 1} ${getRankingUserLabel(item)}`)
  const data = props.rankingItems.map((item) => toFiniteNumber(item.actual_cost))
  const backgroundColor = props.rankingItems.map((_, index) => getChartColor(index))

  if (otherRankingItem.value) {
    labels.push(t('admin.dashboard.spendingRankingOther'))
    data.push(otherRankingItem.value.actual_cost)
    backgroundColor.push('#94a3b8')
  }

  return {
    labels,
    datasets: [
      {
        data,
        backgroundColor,
        borderWidth: 0
      }
    ]
  }
})

const otherRankingItem = computed<RankingDisplayItem | null>(() => {
  if (!props.rankingItems?.length) return null

  const rankedActualCost = props.rankingItems.reduce((sum, item) => sum + toFiniteNumber(item.actual_cost), 0)
  const rankedRequests = props.rankingItems.reduce((sum, item) => sum + toFiniteNumber(item.requests), 0)
  const rankedTokens = props.rankingItems.reduce((sum, item) => sum + toFiniteNumber(item.tokens), 0)

  const otherActualCost = Math.max((props.rankingTotalActualCost || 0) - rankedActualCost, 0)
  const otherRequests = Math.max((props.rankingTotalRequests || 0) - rankedRequests, 0)
  const otherTokens = Math.max((props.rankingTotalTokens || 0) - rankedTokens, 0)

  if (otherActualCost <= 0.000001 && otherRequests <= 0 && otherTokens <= 0) return null

  return {
    user_id: 0,
    email: '',
    username: '',
    main_model: '',
    actual_cost: otherActualCost,
    requests: otherRequests,
    tokens: otherTokens,
    isOther: true
  }
})

const rankingDisplayItems = computed<RankingDisplayItem[]>(() => {
  if (!props.rankingItems?.length) return []
  return otherRankingItem.value
    ? [...props.rankingItems, otherRankingItem.value]
    : [...props.rankingItems]
})

const doughnutOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  cutout: isHomeClay.value ? '64%' : undefined,
  animation: isHomeClay.value ? false : undefined,
  plugins: {
    legend: {
      display: false
    },
    tooltip: {
      callbacks: {
        label: (context: any) => {
          const value = context.raw as number
          const total = context.dataset.data.reduce((a: number, b: number) => a + b, 0)
          const percentage = total > 0 ? ((value / total) * 100).toFixed(1) : '0.0'
          const formattedValue = props.metric === 'actual_cost'
            ? formatActualCostText(value)
            : formatTokens(value)
          return `${context.label}: ${formattedValue} (${percentage}%)`
        }
      }
    }
  }
}))

watch(
  () => [props.source, props.startDate, props.endDate, props.filters] as const,
  () => {
    breakdownRequestSeq += 1
    expandedKey.value = null
    breakdownItems.value = []
    breakdownLoading.value = false
  }
)

const rankingDoughnutOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: {
      display: false
    },
    tooltip: {
      callbacks: {
        label: (context: any) => {
          const value = context.raw as number
          const total = context.dataset.data.reduce((a: number, b: number) => a + b, 0)
          const percentage = total > 0 ? ((value / total) * 100).toFixed(1) : '0.0'
          return `${context.label}: ${formatActualCostText(value)} (${percentage}%)`
        }
      }
    }
  }
}))

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

const formatNumber = (value: number): string => {
  return toFiniteNumber(value).toLocaleString()
}

const getRankingUserLabel = (item: UserSpendingRankingItem): string => {
  if (item.email) return item.email
  return t('admin.redeem.userPrefix', { id: item.user_id })
}

const getRankingRowLabel = (item: RankingDisplayItem): string => {
  if (item.isOther) return t('admin.dashboard.spendingRankingOther')
  return getRankingUserLabel(item)
}

const toFiniteNumber = (value: unknown): number => {
  const numberValue = Number(value)
  return Number.isFinite(numberValue) ? numberValue : 0
}

const formatCost = (value: number | null | undefined): string => {
  const safeValue = toFiniteNumber(value)
  if (safeValue >= 1000) {
    return (safeValue / 1000).toFixed(2) + 'K'
  } else if (safeValue >= 1) {
    return safeValue.toFixed(2)
  } else if (safeValue >= 0.01) {
    return safeValue.toFixed(3)
  }
  return safeValue.toFixed(4)
}

const formatActualCostText = (value: number | null | undefined): string => (
  creditMode.value
    ? `${t('dashboard.creditUnit')} ${formatCost(value)}`
    : `$${formatCost(value)}`
)
</script>

<style scoped>
.model-distribution--home-clay {
  container-type: inline-size;
  color: var(--clay-text, var(--lx-clay-text, #332f3a));
}

.model-distribution--home-clay .model-distribution-heading {
  align-items: flex-start;
  margin-bottom: 16px;
}

.home-chart-eyebrow {
  display: block;
  margin-bottom: 5px;
  color: var(--clay-muted, var(--lx-clay-text-muted));
  font-size: 9px;
  font-weight: 900;
  letter-spacing: 0.16em;
}

.model-distribution--home-clay .model-distribution-heading h3 {
  margin: 0;
  color: var(--clay-text, var(--lx-clay-text));
  font-family: var(--lx-clay-font-display);
  font-size: 18px;
  font-weight: 900;
  letter-spacing: -0.025em;
}

.model-distribution--home-clay .chart-toggle-group {
  border: 1px solid var(--clay-border, var(--lx-clay-border, rgba(91, 80, 112, 0.14)));
  border-radius: 13px;
  background: var(--clay-recessed, var(--lx-clay-recessed, #efebf5));
  box-shadow: var(--clay-recessed-shadow, none);
}

.model-distribution--home-clay .chart-toggle-button {
  min-height: 30px;
  border-radius: 10px;
  color: var(--clay-secondary, var(--lx-clay-text-secondary, #635f69));
}

.model-distribution--home-clay .chart-toggle-button.bg-white,
:global(html.dark) .model-distribution--home-clay .chart-toggle-button.dark\:bg-dark-700 {
  color: var(--clay-violet, var(--lx-clay-accent, #7c3aed));
  background: var(--clay-surface, var(--lx-clay-surface, #fff));
  box-shadow: 0 4px 12px rgba(77, 60, 103, 0.09);
}

.model-distribution--home-clay .chart-toggle-button:focus-visible {
  outline: 3px solid rgba(124, 58, 237, 0.38);
  outline-offset: 2px;
}

.home-model-content,
.home-model-data {
  min-width: 0;
}

.home-model-layout {
  display: grid;
  grid-template-columns: minmax(176px, 0.88fr) minmax(220px, 1.12fr);
  align-items: center;
  gap: 24px;
  min-height: 250px;
  transition: opacity 180ms ease-out;
}

.home-model-layout.is-updating {
  opacity: 0.68;
}

.home-model-visual {
  position: relative;
  width: min(100%, 198px);
  aspect-ratio: 1;
  margin: 0 auto;
  filter: drop-shadow(0 14px 18px rgba(91, 33, 182, 0.12));
}

.home-model-visual :deep(canvas) {
  width: 100% !important;
  height: 100% !important;
}

.home-model-total {
  position: absolute;
  inset: 22%;
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--clay-border, var(--lx-clay-border, rgba(91, 80, 112, 0.14)));
  border-radius: 50%;
  background: var(--clay-surface, var(--lx-clay-surface, #fff));
  box-shadow: 0 5px 14px rgba(54, 42, 73, 0.1);
  flex-direction: column;
  pointer-events: none;
}

.home-model-total span {
  max-width: 90%;
  overflow: hidden;
  color: var(--clay-secondary, var(--lx-clay-text-secondary, #635f69));
  font-size: 10px;
  font-weight: 650;
  line-height: 1.2;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.home-model-total strong {
  max-width: 88%;
  margin-top: 3px;
  overflow: hidden;
  color: var(--clay-text, var(--lx-clay-text, #332f3a));
  font-family: var(--lx-clay-font-display, inherit);
  font-size: 23px;
  font-variant-numeric: tabular-nums;
  font-weight: 900;
  letter-spacing: -0.035em;
  line-height: 1.1;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.home-model-legend {
  max-height: 270px;
  margin: 0;
  padding: 0;
  overflow-y: auto;
  list-style: none;
}

.home-model-legend li {
  border-bottom: 1px solid var(--clay-border, var(--lx-clay-border, rgba(91, 80, 112, 0.14)));
}

.home-model-legend li:last-child {
  border-bottom: 0;
}

.home-model-row {
  display: grid;
  width: 100%;
  min-width: 0;
  min-height: 56px;
  grid-template-columns: 9px minmax(0, 1fr) auto 16px;
  align-items: center;
  gap: 10px;
  padding: 7px 4px;
  border: 0;
  border-radius: 10px;
  color: inherit;
  background: transparent;
  text-align: left;
}

button.home-model-row {
  cursor: pointer;
  transition: color 160ms ease-out, background-color 160ms ease-out;
}

button.home-model-row:hover {
  background: var(--clay-surface-soft, var(--lx-clay-surface-soft, rgba(124, 58, 237, 0.045)));
}

button.home-model-row:focus-visible {
  outline: 3px solid rgba(124, 58, 237, 0.36);
  outline-offset: -1px;
}

.home-model-row.is-static {
  grid-template-columns: 9px minmax(0, 1fr) auto;
}

.home-model-dot {
  width: 9px;
  height: 9px;
  border-radius: 3px;
  background: var(--model-color, #7c3aed);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--model-color, #7c3aed) 14%, transparent);
}

.home-model-copy {
  display: flex;
  min-width: 0;
  flex-direction: column;
}

.home-model-copy strong {
  overflow: hidden;
  color: var(--clay-text, var(--lx-clay-text, #332f3a));
  font-size: 12px;
  font-weight: 750;
  line-height: 1.35;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.home-model-copy small {
  margin-top: 2px;
  overflow: hidden;
  color: var(--clay-secondary, var(--lx-clay-text-secondary, #635f69));
  font-size: 10px;
  font-variant-numeric: tabular-nums;
  line-height: 1.3;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.home-model-share {
  color: var(--clay-secondary, var(--lx-clay-text-secondary, #635f69));
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  font-weight: 800;
  text-align: right;
}

.home-model-chevron {
  width: 16px;
  height: 16px;
  color: var(--clay-muted, var(--lx-clay-text-muted, #756e80));
  fill: none;
  stroke: currentcolor;
  stroke-linecap: round;
  stroke-linejoin: round;
  stroke-width: 1.8;
  transition: transform 160ms ease-out;
}

.home-model-chevron.expanded {
  transform: rotate(90deg);
}

.home-model-breakdown {
  margin-top: 14px;
  overflow-x: auto;
  border: 1px solid var(--clay-border, var(--lx-clay-border, rgba(91, 80, 112, 0.14)));
  border-radius: var(--lx-clay-radius-table, 14px);
  background: var(--clay-surface-soft, var(--lx-clay-surface-soft, rgba(255, 255, 255, 0.66)));
}

.home-model-breakdown-heading {
  display: flex;
  min-width: 520px;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 9px 12px;
  border-bottom: 1px solid var(--clay-border, var(--lx-clay-border, rgba(91, 80, 112, 0.14)));
  color: var(--clay-secondary, var(--lx-clay-text-secondary, #635f69));
  font-size: 11px;
}

.home-model-breakdown-heading strong {
  max-width: 75%;
  overflow: hidden;
  color: var(--clay-text, var(--lx-clay-text, #332f3a));
  text-overflow: ellipsis;
  white-space: nowrap;
}

.home-model-breakdown :deep(table) {
  min-width: 520px;
}

.home-model-inline-error {
  margin: 0 0 12px;
  padding: 9px 11px;
  border: 1px solid var(--lx-clay-danger, #c2415b);
  border-radius: 11px;
  color: var(--lx-clay-danger, #c2415b);
  background: var(--lx-clay-danger-soft, rgba(220, 76, 100, 0.11));
  font-size: 12px;
}

.home-model-state {
  display: flex;
  min-height: 250px;
  align-items: center;
  justify-content: center;
  color: var(--clay-secondary, var(--lx-clay-text-secondary, #635f69));
  flex-direction: column;
  text-align: center;
}

.home-model-state svg {
  width: 30px;
  height: 30px;
  margin-bottom: 10px;
  color: var(--clay-muted, var(--lx-clay-text-muted, #756e80));
  fill: none;
  stroke: currentcolor;
  stroke-linecap: round;
  stroke-linejoin: round;
  stroke-width: 1.6;
}

.home-model-state p {
  margin: 0;
  font-size: 13px;
}

.home-model-state button {
  min-height: 36px;
  margin-top: 12px;
  padding: 0 14px;
  border: 1px solid var(--clay-border-strong, var(--lx-clay-border-strong, rgba(91, 80, 112, 0.23)));
  border-radius: 11px;
  color: var(--clay-violet, var(--lx-clay-accent, #7c3aed));
  background: var(--clay-surface, var(--lx-clay-surface, #fff));
  font-size: 12px;
  font-weight: 750;
  cursor: pointer;
}

.home-model-state button:focus-visible {
  outline: 3px solid rgba(124, 58, 237, 0.36);
  outline-offset: 2px;
}

.home-model-skeleton {
  display: grid;
  min-height: 250px;
  grid-template-columns: minmax(176px, 0.88fr) minmax(220px, 1.12fr);
  align-items: center;
  gap: 24px;
}

.home-model-skeleton-ring {
  width: min(100%, 198px);
  aspect-ratio: 1;
  margin: 0 auto;
  border: 27px solid var(--clay-recessed-strong, var(--lx-clay-recessed-strong, #e8e2f0));
  border-radius: 50%;
  animation: home-model-pulse 1.5s ease-in-out infinite;
}

.home-model-skeleton-list {
  display: grid;
}

.home-model-skeleton-list span {
  height: 56px;
  border-bottom: 1px solid var(--clay-border, var(--lx-clay-border, rgba(91, 80, 112, 0.14)));
  background: linear-gradient(
    90deg,
    transparent,
    var(--clay-surface-soft, var(--lx-clay-surface-soft, rgba(255, 255, 255, 0.66))),
    transparent
  );
  animation: home-model-pulse 1.5s ease-in-out infinite;
}

@keyframes home-model-pulse {
  0%,
  100% {
    opacity: 0.48;
  }

  50% {
    opacity: 0.9;
  }
}

@container (max-width: 540px) {
  .model-distribution-heading {
    align-items: flex-start;
    flex-direction: column;
  }

  .model-distribution-heading > div {
    width: 100%;
    justify-content: flex-start;
  }

  .home-model-layout,
  .home-model-skeleton {
    grid-template-columns: 1fr;
    gap: 18px;
  }

  .home-model-visual,
  .home-model-skeleton-ring {
    width: min(100%, 188px);
  }

  .home-model-legend {
    width: 100%;
    max-height: 292px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .home-model-layout,
  .home-model-chevron,
  button.home-model-row {
    transition: none;
  }

  .home-model-skeleton-ring,
  .home-model-skeleton-list span {
    animation: none;
  }
}

@media (max-width: 640px) {
  .model-distribution--home-clay .chart-toggle-button {
    min-height: 38px;
  }
}
</style>
