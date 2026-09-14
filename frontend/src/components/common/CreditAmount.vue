<template>
  <span
    class="inline-flex min-w-0 items-center gap-1 whitespace-nowrap tabular-nums"
    role="group"
    :aria-label="accessibleLabel"
    data-testid="credit-amount"
  >
    <span
      class="inline-flex shrink-0 items-center justify-center font-semibold leading-none"
      :class="symbolSizeClasses[iconSize]"
      aria-hidden="true"
      data-testid="credit-amount-symbol"
    >$</span>
    <span
      class="min-w-0 truncate"
      aria-hidden="true"
      data-testid="credit-amount-value"
    >{{ value }}</span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'

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

  return `$${props.value}`
})

const symbolSizeClasses: Record<PointsIconSize, string> = {
  xs: 'text-[10px]',
  sm: 'text-xs',
  md: 'text-sm',
  lg: 'text-base',
  xl: 'text-lg',
}
</script>
