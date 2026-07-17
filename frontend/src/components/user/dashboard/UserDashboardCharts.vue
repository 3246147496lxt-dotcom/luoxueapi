<template>
  <section
    class="card overflow-hidden rounded-[28px]"
    :aria-label="`${t('dashboard.modelDistribution')} / ${t('dashboard.tokenUsageTrend')}`"
  >
    <div class="border-b border-gray-100 bg-gray-50/70 px-5 py-4 md:px-6 dark:border-dark-700 dark:bg-dark-900/45">
      <div class="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
        <div class="min-w-0 flex-1">
          <span class="mb-2 block text-xs font-medium text-gray-500 dark:text-dark-400">
            {{ t('dashboard.timeRange') }}
          </span>
          <DateRangePicker
            :start-date="startDate"
            :end-date="endDate"
            @update:startDate="$emit('update:startDate', $event)"
            @update:endDate="$emit('update:endDate', $event)"
            @change="$emit('dateRangeChange', $event)"
          />
        </div>

        <div class="flex flex-wrap items-end gap-3">
          <div>
            <span class="mb-2 block text-xs font-medium text-gray-500 dark:text-dark-400">
              {{ t('dashboard.granularity') }}
            </span>
            <div class="w-32">
              <Select
                :model-value="granularity"
                :options="[
                  { value: 'day', label: t('dashboard.day') },
                  { value: 'hour', label: t('dashboard.hour') }
                ]"
                @update:model-value="$emit('update:granularity', $event)"
                @change="$emit('granularityChange')"
              />
            </div>
          </div>
          <button
            type="button"
            class="btn btn-secondary min-h-10 px-4"
            :disabled="loading"
            @click="$emit('refresh')"
          >
            {{ t('common.refresh') }}
          </button>
        </div>
      </div>
    </div>

    <div class="grid grid-cols-1 divide-y divide-gray-100 lg:grid-cols-[minmax(0,0.95fr)_minmax(0,1.05fr)] lg:divide-x lg:divide-y-0 dark:divide-dark-700">
      <div class="relative min-w-0 p-5 md:p-6">
        <div
          v-if="loading"
          class="absolute inset-0 z-10 flex items-center justify-center bg-white/85 dark:bg-dark-800/85"
        >
          <LoadingSpinner size="md" />
        </div>

        <div class="mb-5 flex items-center gap-2">
          <span class="h-2 w-2 rounded-full bg-primary-600 dark:bg-primary-400"></span>
          <h3 class="text-base font-semibold text-gray-950 dark:text-white">
            {{ t('dashboard.modelDistribution') }}
          </h3>
        </div>

        <div class="flex flex-col gap-5 sm:flex-row sm:items-center">
          <div class="mx-auto h-44 w-44 flex-shrink-0 sm:mx-0">
            <Doughnut v-if="modelData" :data="modelData" :options="doughnutOptions" />
            <div
              v-else
              class="flex h-full items-center justify-center rounded-full bg-gray-50 px-5 text-center text-sm text-gray-500 dark:bg-dark-900/50 dark:text-gray-400"
            >
              {{ t('dashboard.noDataAvailable') }}
            </div>
          </div>

          <div class="max-h-52 min-w-0 flex-1 overflow-auto">
            <table class="w-full min-w-[430px] text-xs">
              <thead class="sticky top-0 z-[1] bg-white dark:bg-dark-800">
                <tr class="text-gray-500 dark:text-gray-400">
                  <th class="pb-2 text-left font-medium">{{ t('dashboard.model') }}</th>
                  <th class="pb-2 text-right font-medium">{{ t('dashboard.requests') }}</th>
                  <th class="pb-2 text-right font-medium">{{ t('dashboard.tokens') }}</th>
                  <th class="pb-2 text-right font-medium">{{ t('dashboard.actual') }}</th>
                  <th class="pb-2 text-right font-medium">{{ t('dashboard.standard') }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                <tr v-for="model in models" :key="model.model">
                  <td
                    class="max-w-[120px] truncate py-2 pr-3 font-medium text-gray-900 dark:text-white"
                    :title="model.model"
                  >
                    {{ model.model }}
                  </td>
                  <td class="py-2 text-right text-gray-600 dark:text-gray-400">
                    {{ formatNumber(model.requests) }}
                  </td>
                  <td class="py-2 text-right text-gray-600 dark:text-gray-400">
                    {{ formatTokens(model.total_tokens) }}
                  </td>
                  <td class="py-2 text-right font-medium text-primary-700 dark:text-primary-300">
                    ${{ formatCost(model.actual_cost) }}
                  </td>
                  <td class="py-2 text-right text-gray-400 dark:text-gray-500">
                    ${{ formatCost(model.cost) }}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>

      <div
        class="min-w-0 p-5 md:p-6 [&>div]:h-full [&>div]:rounded-none [&>div]:border-0 [&>div]:bg-transparent [&>div]:p-0 [&>div]:shadow-none dark:[&>div]:bg-transparent"
      >
        <TokenUsageTrend :trend-data="trend" :loading="loading" />
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import Select from '@/components/common/Select.vue'
import { Doughnut } from 'vue-chartjs'
import TokenUsageTrend from '@/components/charts/TokenUsageTrend.vue'
import type { TrendDataPoint, ModelStat } from '@/types'
import { formatCostFixed as formatCost, formatNumberLocaleString as formatNumber, formatTokensK as formatTokens } from '@/utils/format'
import { Chart as ChartJS, CategoryScale, LinearScale, PointElement, LineElement, ArcElement, Title, Tooltip, Legend, Filler } from 'chart.js'
ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, ArcElement, Title, Tooltip, Legend, Filler)

const props = defineProps<{ loading: boolean, startDate: string, endDate: string, granularity: string, trend: TrendDataPoint[], models: ModelStat[] }>()
defineEmits(['update:startDate', 'update:endDate', 'update:granularity', 'dateRangeChange', 'granularityChange', 'refresh'])
const { t } = useI18n()

const modelData = computed(() => !props.models?.length ? null : {
  labels: props.models.map((m: ModelStat) => m.model),
  datasets: [{
    data: props.models.map((m: ModelStat) => m.total_tokens),
    backgroundColor: ['#0f766e', '#14b8a6', '#5eead4', '#d4a72c', '#f4d35e', '#64748b', '#94a3b8', '#0d9488']
  }]
})

const doughnutOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { display: false },
    tooltip: {
      callbacks: {
        label: (context: any) => `${context.label}: ${formatTokens(context.parsed)} tokens`
      }
    }
  }
}
</script>
