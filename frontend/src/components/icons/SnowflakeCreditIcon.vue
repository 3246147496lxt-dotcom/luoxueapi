<template>
  <img
    :src="snowflakeCreditUrl"
    :alt="label || ''"
    :aria-hidden="label ? undefined : 'true'"
    :class="[
      'inline-block shrink-0 object-contain align-[-0.125em]',
      sizeClasses[size],
    ]"
    draggable="false"
    data-testid="snowflake-credit-icon"
  >
</template>

<script setup lang="ts">
import snowflakeCreditUrl from '@/assets/icons/snowflake-credit.svg'

type SnowflakeCreditIconSize = 'xs' | 'sm' | 'md' | 'lg' | 'xl'

withDefaults(defineProps<{
  size?: SnowflakeCreditIconSize
  label?: string
}>(), {
  size: 'md',
  label: '',
})

const sizeClasses: Record<SnowflakeCreditIconSize, string> = {
  xs: 'h-3 w-3',
  sm: 'h-4 w-4',
  md: 'h-5 w-5',
  lg: 'h-6 w-6',
  xl: 'h-8 w-8',
}
</script>
