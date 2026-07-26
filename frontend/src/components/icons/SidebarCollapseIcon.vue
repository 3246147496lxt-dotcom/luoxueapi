<template>
  <svg
    fill="none"
    viewBox="0 0 24 24"
    stroke="currentColor"
    stroke-width="1.75"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
  >
    <rect width="18" height="18" x="3" y="3" rx="2" />
    <path d="M9 3v18" />
    <path v-if="collapsed" d="m14 9 3 3-3 3" />
    <path v-else d="m16 15-3-3 3-3" />
  </svg>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  collapsed?: boolean
}>(), {
  collapsed: false
})
</script>
