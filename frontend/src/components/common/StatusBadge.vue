<template>
  <div class="status-badge flex items-center gap-1.5" :data-status="status">
    <span
      :class="[
        'status-badge__dot inline-block h-2 w-2 rounded-full',
        variantClass
      ]"
      aria-hidden="true"
    ></span>
    <span class="status-badge__label text-sm">
      {{ label }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  status: string
  label: string
}>()

const variantClass = computed(() => {
  switch (props.status) {
    case 'active':
    case 'success':
      return 'status-badge__dot--success'
    case 'disabled':
    case 'inactive':
      return 'status-badge__dot--neutral'
    case 'warning':
      return 'status-badge__dot--warning'
    case 'error':
    case 'danger':
      return 'status-badge__dot--danger'
    default:
      return 'status-badge__dot--neutral'
  }
})
</script>
