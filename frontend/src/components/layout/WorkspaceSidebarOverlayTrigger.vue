<template>
  <button
    :id="id"
    type="button"
    class="workspace-sidebar-overlay-trigger"
    :class="{ 'workspace-sidebar-overlay-trigger--open': open }"
    :aria-expanded="open"
    :aria-controls="controls"
    :aria-label="label"
    :title="label"
    @click="emit('open')"
  >
    <WorkspaceResponsiveSidebarIcon name="open" />
  </button>
</template>

<script setup lang="ts">
import WorkspaceResponsiveSidebarIcon from './WorkspaceResponsiveSidebarIcon.vue'

defineProps<{
  id: string
  controls: string
  label: string
  open: boolean
}>()

const emit = defineEmits<{
  open: []
}>()
</script>

<style scoped>
.workspace-sidebar-overlay-trigger {
  position: fixed;
  top: var(--workspace-space-2);
  left: var(--workspace-space-2);
  z-index: calc(var(--workspace-layer-sidebar-overlay) - 10);
  display: grid;
  width: var(--workspace-sidebar-action-size);
  height: var(--workspace-sidebar-action-size);
  place-items: center;
  border: 0;
  border-radius: var(--workspace-radius-compact);
  padding: 0;
  color: var(--workspace-sidebar-overlay-icon);
  background: transparent;
  cursor: pointer;
  transition:
    color 160ms ease-out,
    background-color 160ms ease-out,
    opacity 160ms ease-out;
}

.workspace-sidebar-overlay-trigger:hover {
  color: var(--workspace-text);
  background: var(--workspace-hover);
}

.workspace-sidebar-overlay-trigger:focus-visible {
  outline: var(--workspace-space-0-5) solid var(--workspace-text-secondary);
  outline-offset: var(--workspace-space-0-5);
}

.workspace-sidebar-overlay-trigger--open {
  visibility: hidden;
  opacity: 0;
  pointer-events: none;
}

@media (prefers-reduced-motion: reduce) {
  .workspace-sidebar-overlay-trigger {
    transition: none;
  }
}
</style>
