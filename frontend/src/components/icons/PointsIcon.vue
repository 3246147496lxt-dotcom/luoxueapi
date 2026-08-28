<template>
  <img
    :src="pointsUrl"
    :alt="label || ''"
    :aria-hidden="label ? undefined : 'true'"
    :class="[
      'inline-block shrink-0 object-contain align-[-0.125em]',
      sizeClasses[size],
    ]"
    draggable="false"
    :data-testid="testId"
    data-icon="points"
  >
</template>

<script setup lang="ts">
import pointsUrl from '@/assets/icons/points.svg'

export type PointsIconSize = 'xs' | 'sm' | 'md' | 'lg' | 'xl'

withDefaults(defineProps<{
  size?: PointsIconSize
  label?: string
  /** Allows the legacy facade to preserve its historical selector. */
  testId?: string
}>(), {
  size: 'md',
  label: '',
  testId: 'points-icon',
})

const sizeClasses: Record<PointsIconSize, string> = {
  xs: 'h-3 w-3',
  sm: 'h-4 w-4',
  md: 'h-5 w-5',
  lg: 'h-6 w-6',
  xl: 'h-8 w-8',
}
</script>
