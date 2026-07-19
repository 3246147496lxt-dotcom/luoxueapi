<template>
  <span
    class="inline-flex min-w-0 items-center gap-1 whitespace-nowrap tabular-nums"
    role="group"
    :aria-label="accessibleLabel"
    data-testid="credit-amount"
  >
    <SnowflakeCreditIcon :size="iconSize" />
    <span
      class="min-w-0 truncate"
      aria-hidden="true"
      data-testid="credit-amount-value"
    >{{ value }}</span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import SnowflakeCreditIcon from '@/components/icons/SnowflakeCreditIcon.vue'
import { getLocale, i18n } from '@/i18n'

type SnowflakeCreditIconSize = 'xs' | 'sm' | 'md' | 'lg' | 'xl'

const props = withDefaults(defineProps<{
  value: string | number
  iconSize?: SnowflakeCreditIconSize
  label?: string
}>(), {
  iconSize: 'sm',
  label: '',
})

const accessibleLabel = computed(() => {
  const explicitLabel = props.label.trim()
  if (explicitLabel) return explicitLabel

  const translationKey = 'dashboard.creditUnit'
  const unit = i18n.global.te(translationKey)
    ? i18n.global.t(translationKey)
    : (getLocale() === 'zh' ? '雪花额度' : 'Snow credits')

  return `${props.value} ${unit}`
})
</script>
