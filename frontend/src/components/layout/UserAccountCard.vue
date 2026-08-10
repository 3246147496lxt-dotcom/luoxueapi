<template>
  <SidebarAccountDock :context="context" :collapsed="collapsed" />
</template>

<script setup lang="ts">
import SidebarAccountDock from './SidebarAccountDock.vue'

withDefaults(defineProps<{
  context?: 'work' | 'chat'
  collapsed?: boolean
}>(), {
  context: 'work',
  collapsed: false,
})
</script>
