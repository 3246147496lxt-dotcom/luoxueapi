<template>
  <!--
    Compatibility facade for integrations that still import the historical
    component name. New code should use PointsIcon directly; both render the
    same points artwork and carry the points semantic data attribute.
  -->
  <PointsIcon v-bind="$props" test-id="snowflake-credit-icon" />
</template>

<script setup lang="ts">
import PointsIcon from './PointsIcon.vue'

defineOptions({ name: 'SnowflakeCreditIcon' })

type PointsIconSize = 'xs' | 'sm' | 'md' | 'lg' | 'xl'

defineProps<{
  size?: PointsIconSize
  label?: string
}>()
</script>
