<template>
  <span
    class="inline-flex min-w-0 items-center gap-1 whitespace-nowrap tabular-nums"
    role="group"
    :aria-label="accessibleLabel"
    data-testid="credit-amount"
  >
    <PointsIcon :size="iconSize" />
    <span
      class="min-w-0 truncate"
      aria-hidden="true"
      data-testid="credit-amount-value"
    >{{ value }}</span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import PointsIcon from '@/components/icons/PointsIcon.vue'
import { getLocale, i18n } from '@/i18n'

type PointsIconSize = 'xs' | 'sm' | 'md' | 'lg' | 'xl'

const props = withDefaults(defineProps<{
  value: string | number
  iconSize?: PointsIconSize
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
    : (getLocale() === 'zh' ? '积分' : 'Points')

  return `${props.value} ${unit}`
})
</script>
