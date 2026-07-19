<template>
  <section aria-labelledby="dashboard-quick-actions-title">
    <div class="flex items-end justify-between gap-4">
      <h2
        id="dashboard-quick-actions-title"
        class="text-base font-semibold tracking-tight text-gray-950 dark:text-white"
      >
        {{ t('dashboard.quickActions') }}
      </h2>
    </div>

    <div class="mt-4 grid grid-cols-1 gap-3 md:grid-cols-3 md:gap-4">
      <button
        type="button"
        class="group flex min-h-[184px] w-full flex-col rounded-[24px] border border-gray-200/80 bg-white p-5 text-left shadow-card transition-colors duration-200 hover:border-primary-200 hover:bg-primary-50/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 focus-visible:ring-offset-2 dark:border-dark-700 dark:bg-dark-800 dark:hover:border-primary-800 dark:hover:bg-primary-950/20"
        @click="router.push('/keys')"
      >
        <div class="flex items-start justify-between gap-4">
          <div
            class="flex h-11 w-11 items-center justify-center rounded-2xl bg-primary-50 text-primary-700 dark:bg-primary-950/70 dark:text-primary-300"
          >
            <Icon name="key" size="lg" />
          </div>
          <Icon
            name="chevronRight"
            size="md"
            class="text-gray-300 transition-colors group-hover:text-primary-600 dark:text-dark-500 dark:group-hover:text-primary-300"
          />
        </div>
        <p class="mt-6 min-h-10 text-sm leading-5 text-gray-500 dark:text-dark-400">
          {{ t('dashboard.generateNewKey') }}
        </p>
        <div class="mt-auto flex items-center gap-2 border-t border-gray-100 pt-4 dark:border-dark-700">
          <span class="h-2 w-2 rounded-full bg-primary-600 dark:bg-primary-400"></span>
          <span class="text-sm font-semibold text-gray-950 dark:text-white">
            {{ t('dashboard.createApiKey') }}
          </span>
        </div>
      </button>

      <button
        type="button"
        class="group flex min-h-[184px] w-full flex-col rounded-[24px] border border-[#dcebe4] bg-[#eef8f3] p-5 text-left transition-colors duration-200 hover:border-primary-300 hover:bg-[#e7f5ef] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 focus-visible:ring-offset-2 dark:border-primary-950 dark:bg-primary-950/30 dark:hover:border-primary-800 dark:hover:bg-primary-950/50"
        @click="router.push('/usage')"
      >
        <div class="flex items-start justify-between gap-4">
          <div
            class="flex h-11 w-11 items-center justify-center rounded-2xl bg-white text-primary-700 shadow-sm dark:bg-dark-800 dark:text-primary-300"
          >
            <Icon name="chart" size="lg" />
          </div>
          <Icon
            name="chevronRight"
            size="md"
            class="text-primary-300 transition-colors group-hover:text-primary-700 dark:text-primary-800 dark:group-hover:text-primary-300"
          />
        </div>
        <p class="mt-6 min-h-10 text-sm leading-5 text-gray-600 dark:text-dark-300">
          {{ t('dashboard.checkDetailedLogs') }}
        </p>
        <div class="mt-auto flex items-center gap-2 border-t border-primary-100 pt-4 dark:border-primary-900/60">
          <span class="h-2 w-2 rounded-full bg-emerald-500 dark:bg-emerald-400"></span>
          <span class="text-sm font-semibold text-gray-950 dark:text-white">
            {{ t('dashboard.viewUsage') }}
          </span>
        </div>
      </button>

      <button
        type="button"
        class="group flex min-h-[184px] w-full flex-col rounded-[24px] border border-[#eee2ad] bg-[#fff8d8] p-5 text-left transition-colors duration-200 hover:border-amber-300 hover:bg-[#fff4c5] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-amber-500/40 focus-visible:ring-offset-2 dark:border-[#4d4424] dark:bg-[#302c1d] dark:hover:border-amber-800 dark:hover:bg-[#38321f]"
        @click="router.push({ path: '/purchase', hash: '#redeem' })"
      >
        <div class="flex items-start justify-between gap-4">
          <div
            class="flex h-11 w-11 items-center justify-center rounded-2xl bg-white text-amber-700 shadow-sm dark:bg-[#3a3420] dark:text-amber-300"
          >
            <Icon name="gift" size="lg" />
          </div>
          <Icon
            name="chevronRight"
            size="md"
            class="text-amber-300 transition-colors group-hover:text-amber-700 dark:text-amber-800 dark:group-hover:text-amber-300"
          />
        </div>
        <p class="mt-6 min-h-10 text-sm leading-5 text-[#75631d] dark:text-amber-100/70">
          {{ t('dashboard.addBalanceWithCode') }}
        </p>
        <div class="mt-auto flex items-center gap-2 border-t border-amber-200/80 pt-4 dark:border-amber-900/50">
          <span class="h-2 w-2 rounded-full bg-amber-500 dark:bg-amber-400"></span>
          <span class="text-sm font-semibold text-gray-950 dark:text-white">
            {{ t('dashboard.redeemCode') }}
          </span>
        </div>
      </button>
    </div>

    <button
      v-if="canUseBatchImage"
      type="button"
      class="group mt-3 flex w-full items-center gap-3 rounded-[20px] border border-primary-100 bg-primary-50/70 px-4 py-3 text-left transition-colors duration-200 hover:border-primary-200 hover:bg-primary-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 focus-visible:ring-offset-2 dark:border-primary-900/60 dark:bg-primary-950/30 dark:hover:border-primary-800 dark:hover:bg-primary-950/50"
      @click="router.push('/batch-image')"
    >
      <span
        class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-xl bg-white text-primary-700 shadow-sm dark:bg-dark-800 dark:text-primary-300"
      >
        <Icon name="sparkles" size="md" />
      </span>
      <span class="min-w-0 flex-1">
        <span class="block text-sm font-semibold text-gray-950 dark:text-white">
          {{ t('dashboard.batchImageAgent') }}
        </span>
        <span class="block truncate text-xs text-gray-500 dark:text-dark-400">
          {{ t('dashboard.batchImageAgentDesc') }}
        </span>
      </span>
      <span class="h-2 w-2 rounded-full bg-primary-500"></span>
      <Icon
        name="chevronRight"
        size="sm"
        class="text-primary-400 transition-colors group-hover:text-primary-700 dark:group-hover:text-primary-300"
      />
    </button>
  </section>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useBatchImageAccess } from '@/composables/useBatchImageAccess'
const router = useRouter()
const { t } = useI18n()
const { canUseBatchImage, refreshBatchImageAccess } = useBatchImageAccess()

onMounted(() => {
  void refreshBatchImageAccess()
})
</script>
