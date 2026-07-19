<template>
  <section class="card overflow-hidden rounded-[28px]" aria-labelledby="recent-usage-title">
    <div
      class="flex items-center justify-between gap-4 border-b border-gray-100 px-5 py-4 md:px-6 dark:border-dark-700"
    >
      <div class="flex items-center gap-2">
        <span class="h-2 w-2 rounded-full bg-primary-600 dark:bg-primary-400"></span>
        <h2 id="recent-usage-title" class="text-base font-semibold text-gray-950 dark:text-white">
          {{ t('dashboard.recentUsage') }}
        </h2>
      </div>
      <span class="badge badge-gray">{{ t('dashboard.last7Days') }}</span>
    </div>

    <div>
      <div v-if="loading" class="flex items-center justify-center py-12">
        <LoadingSpinner size="lg" />
      </div>
      <div v-else-if="data.length === 0" class="px-5 py-8 md:px-6">
        <EmptyState :title="t('dashboard.noUsageRecords')" :description="t('dashboard.startUsingApi')" />
      </div>

      <div v-else>
        <div
          class="hidden grid-cols-[minmax(0,1fr)_7rem_7rem_7rem] items-center gap-4 border-b border-gray-100 bg-gray-50/70 px-6 py-2.5 text-xs font-medium text-gray-500 sm:grid dark:border-dark-700 dark:bg-dark-900/45 dark:text-dark-400"
        >
          <span>{{ t('dashboard.model') }}</span>
          <span class="text-right">{{ t('dashboard.tokens') }}</span>
          <span class="text-right">{{ t('dashboard.actual') }}</span>
          <span class="text-right">{{ t('dashboard.standard') }}</span>
        </div>

        <div class="divide-y divide-gray-100 dark:divide-dark-700">
          <div
            v-for="log in data"
            :key="log.id"
            class="grid grid-cols-1 gap-3 px-5 py-4 transition-colors duration-150 hover:bg-gray-50/70 sm:grid-cols-[minmax(0,1fr)_7rem_7rem_7rem] sm:items-center sm:gap-4 md:px-6 dark:hover:bg-dark-900/35"
          >
            <div class="flex min-w-0 items-center gap-3">
              <div
                class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-2xl bg-primary-50 dark:bg-primary-900/30"
              >
                <Icon name="beaker" size="md" class="text-primary-700 dark:text-primary-300" />
              </div>
              <div class="min-w-0">
                <p class="truncate text-sm font-semibold text-gray-950 dark:text-white" :title="log.model">
                  {{ log.model }}
                </p>
                <p class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">
                  {{ formatDateTime(log.created_at) }}
                </p>
              </div>
            </div>

            <div class="grid grid-cols-3 gap-3 sm:contents">
              <div class="min-w-0 sm:text-right">
                <span class="block text-[11px] text-gray-400 sm:hidden dark:text-dark-500">
                  {{ t('dashboard.tokens') }}
                </span>
                <span class="mt-0.5 block truncate text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ (log.input_tokens + log.output_tokens).toLocaleString() }}
                </span>
              </div>

              <div class="min-w-0 sm:text-right">
                <span class="block text-[11px] text-gray-400 sm:hidden dark:text-dark-500">
                  {{ t('dashboard.actual') }}
                </span>
                <span
                  class="mt-0.5 block truncate text-sm font-semibold text-primary-700 dark:text-primary-300"
                  :title="t('dashboard.actual')"
                >
                  <CreditAmount
                    :value="formatCost(log.actual_cost)"
                    icon-size="sm"
                    :label="`${t('dashboard.actual')} ${formatCost(log.actual_cost)}`"
                  />
                </span>
              </div>

              <div class="min-w-0 sm:text-right">
                <span class="block text-[11px] text-gray-400 sm:hidden dark:text-dark-500">
                  {{ t('dashboard.standard') }}
                </span>
                <span
                  class="mt-0.5 block truncate text-sm text-gray-400 dark:text-gray-500"
                  :title="t('dashboard.standard')"
                >
                  ${{ formatCost(log.total_cost) }}
                </span>
              </div>
            </div>
          </div>
        </div>

        <router-link
          to="/usage"
          class="flex items-center justify-center gap-2 border-t border-gray-100 bg-gray-50/50 px-5 py-3.5 text-sm font-semibold text-primary-700 transition-colors hover:bg-primary-50 hover:text-primary-800 md:px-6 dark:border-dark-700 dark:bg-dark-900/30 dark:text-primary-300 dark:hover:bg-primary-950/30 dark:hover:text-primary-200"
        >
          {{ t('dashboard.viewAllUsage') }}
          <Icon name="arrowRight" size="sm" />
        </router-link>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import CreditAmount from '@/components/common/CreditAmount.vue'
import Icon from '@/components/icons/Icon.vue'
import { formatDateTime } from '@/utils/format'
import type { UsageLog } from '@/types'

defineProps<{
  data: UsageLog[]
  loading: boolean
}>()
const { t } = useI18n()
const formatCost = (c: number) => c.toFixed(4)
</script>
