<template>
  <div class="group-option-item grid min-w-0 flex-1 items-start gap-x-4 gap-y-1">
    <!-- Left: name + description -->
    <div
      class="group-option-item__details flex min-w-0 flex-1 flex-col items-start"
      :title="description || undefined"
    >
      <!-- Row 1: platform badge (name bold) -->
      <GroupBadge
        :name="name"
        :title="name"
        :platform="platform"
        :subscription-type="subscriptionType"
        :show-rate="false"
        class="groupOptionItemBadge"
      />
      <!-- Row 2: description with top spacing -->
      <span
        v-if="description"
        class="group-option-item__description mt-1.5 w-full text-left text-xs leading-relaxed text-gray-500 dark:text-gray-400"
      >
        {{ description }}
      </span>
    </div>

    <!-- Right: rate pill + checkmark aligned with the name -->
    <div class="group-option-item__meta flex shrink-0 items-start gap-2 pt-0.5">
      <!-- Rate pill (platform color) -->
      <span v-if="rateMultiplier !== undefined" :class="['inline-flex items-center whitespace-nowrap rounded-full px-3 py-1 text-xs font-semibold', ratePillClass]">
        <template v-if="hasCustomRate">
          <span class="mr-1 line-through opacity-50">{{ rateMultiplier }}x</span>
          <span class="font-bold">{{ userRateMultiplier }}x</span>
        </template>
        <template v-else>
          {{ rateMultiplier }}x {{ t('admin.groups.rateLabel') }}
        </template>
      </span>
      <!-- Checkmark -->
      <svg
        v-if="showCheckmark && selected"
        class="h-4 w-4 shrink-0 text-primary-600 dark:text-primary-400"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
        stroke-width="2"
      >
        <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
      </svg>
    </div>
    <!-- Keep long peak windows from taking space away from the group name. -->
    <span
      v-if="hasPeakRate"
      class="col-span-2 inline-flex items-center justify-self-end whitespace-nowrap rounded-full bg-amber-50 px-3 py-1 text-xs font-semibold text-amber-700 dark:bg-amber-900/20 dark:text-amber-300"
      :title="peakRateTitle"
    >
      {{ peakRateText }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import GroupBadge from './GroupBadge.vue'
import type { SubscriptionType, GroupPlatform } from '@/types'
import { useAppStore } from '@/stores/app'
import { formatPeakRateWindow, serverTimezoneLabel } from '@/utils/peak-rate'

const { t } = useI18n()

interface Props {
  name: string
  platform: GroupPlatform
  subscriptionType?: SubscriptionType
  rateMultiplier?: number
  userRateMultiplier?: number | null
  peakRateEnabled?: boolean
  peakStart?: string
  peakEnd?: string
  peakRateMultiplier?: number
  description?: string | null
  selected?: boolean
  showCheckmark?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  subscriptionType: 'standard',
  selected: false,
  showCheckmark: true,
  userRateMultiplier: null,
  peakRateEnabled: false
})

// Whether user has a custom rate different from default
const hasCustomRate = computed(() => {
  return (
    props.userRateMultiplier !== null &&
    props.userRateMultiplier !== undefined &&
    props.rateMultiplier !== undefined &&
    props.userRateMultiplier !== props.rateMultiplier
  )
})

const appStore = useAppStore()

const hasPeakRate = computed(() => {
  return Boolean(props.peakRateEnabled && props.peakStart && props.peakEnd)
})

const peakRateText = computed(() => {
  return formatPeakRateWindow(
    {
      peak_rate_enabled: props.peakRateEnabled,
      peak_start: props.peakStart,
      peak_end: props.peakEnd,
      peak_rate_multiplier: props.peakRateMultiplier
    },
    serverTimezoneLabel(appStore.cachedPublicSettings?.server_utc_offset)
  )
})

const peakRateTitle = computed(() => {
  return t('common.peakRateTooltip', { window: peakRateText.value })
})

// Rate pill color matches platform badge color
const ratePillClass = computed(() => {
  switch (props.platform) {
    case 'anthropic':
      return 'bg-amber-50 text-amber-700 dark:bg-amber-900/20 dark:text-amber-400'
    case 'openai':
      return 'bg-green-50 text-green-700 dark:bg-green-900/20 dark:text-green-400'
    case 'gemini':
      return 'bg-sky-50 text-sky-700 dark:bg-sky-900/20 dark:text-sky-400'
    case 'deepseek':
      return 'bg-sky-50 text-sky-700 dark:bg-sky-900/20 dark:text-sky-400'
    case 'zhipu':
      return 'bg-indigo-50 text-indigo-700 dark:bg-indigo-900/20 dark:text-indigo-400'
    default: // antigravity and others
      return 'bg-violet-50 text-violet-700 dark:bg-violet-900/20 dark:text-violet-400'
  }
})
</script>

<style scoped>
.group-option-item {
  width: 100%;
  grid-template-columns: minmax(0, 1fr) auto;
}

.group-option-item__details {
  min-width: 0;
}

.group-option-item__meta {
  flex-shrink: 0;
}

/* Bold the group name inside GroupBadge when used in dropdown option */
.groupOptionItemBadge {
  display: inline-flex;
  width: fit-content;
  max-width: 100%;
  min-width: 0;
  align-items: flex-start;
}

.groupOptionItemBadge :deep(span.truncate) {
  font-weight: 600;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  display: block;
  line-height: 1.35;
}

.group-option-item__description {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
