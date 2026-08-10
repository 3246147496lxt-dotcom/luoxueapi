<template>
  <svg
    class="workspace-responsive-sidebar-icon"
    :class="`workspace-responsive-sidebar-icon--${name}`"
    :viewBox="definition.viewBox"
    fill="currentColor"
    aria-hidden="true"
    focusable="false"
  >
    <path :d="definition.path" />
  </svg>
</template>

<script setup lang="ts">
import { computed } from 'vue'

type WorkspaceResponsiveSidebarIconName = 'open' | 'close' | 'search'

const props = defineProps<{
  name: WorkspaceResponsiveSidebarIconName
}>()

const definitions = {
  open: {
    viewBox: '0 0 20 20',
    path: 'm11.666 12.669.135.013a.665.665 0 0 1 0 1.303l-.135.014H3.333a.665.665 0 0 1 0-1.33zm5-6.667.135.013a.665.665 0 0 1 0 1.303l-.135.014H3.333a.665.665 0 0 1 0-1.33z',
  },
  close: {
    viewBox: '0 0 20 20',
    path: 'M14.255 4.755a.7.7 0 0 1 .99.99L10.99 10l4.255 4.255.09.11a.7.7 0 0 1-.97.97l-.11-.09L10 10.99l-4.255 4.255a.7.7 0 0 1-.99-.99L9.01 10 4.755 5.745l-.09-.11a.7.7 0 0 1 .97-.97l.11.09L10 9.01z',
  },
  search: {
    viewBox: '0 0 24 24',
    path: 'M10.993 2.904a8 8 0 0 1 6.122 13.146l3.923 3.923a.75.75 0 0 1-1.06 1.06l-3.931-3.93a8 8 0 1 1-5.054-14.2m0 1.5a6.5 6.5 0 1 0 0 13 6.5 6.5 0 0 0 0-13',
  },
} as const

const definition = computed(() => definitions[props.name])
</script>

<style scoped>
.workspace-responsive-sidebar-icon {
  display: block;
  flex: 0 0 auto;
}

.workspace-responsive-sidebar-icon--open,
.workspace-responsive-sidebar-icon--search {
  width: var(--workspace-space-6);
  height: var(--workspace-space-6);
}

.workspace-responsive-sidebar-icon--close {
  width: var(--workspace-space-5);
  height: var(--workspace-space-5);
}
</style>
