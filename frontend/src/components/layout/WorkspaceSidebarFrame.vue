<template>
  <aside
    :id="id"
    class="workspace-sidebar-frame"
    :class="[
      `workspace-sidebar-frame--${placement}`,
      `workspace-sidebar-frame--${surface}`,
      `workspace-sidebar-frame--content-${contentMode}`,
      {
        'workspace-sidebar-frame--collapsed': effectiveCollapsed,
        'workspace-sidebar-frame--mobile': mobile,
        'workspace-sidebar-frame--overlay': overlay,
      },
    ]"
    :aria-label="label"
    :data-sidebar-collapsed="effectiveCollapsed"
    :data-sidebar-placement="placement"
  >
    <slot name="header" />
    <slot v-if="!effectiveCollapsed" name="mode-switch" />

    <div class="workspace-sidebar-frame__content">
      <slot />
    </div>

    <div v-if="$slots.footer" class="workspace-sidebar-frame__footer">
      <slot name="footer" />
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed } from 'vue'

export type WorkspaceSidebarPlacement = 'fixed' | 'flow'
export type WorkspaceSidebarSurface = 'work' | 'chat'
export type WorkspaceSidebarContentMode = 'workspace' | 'admin'

const props = withDefaults(defineProps<{
  id: string
  label: string
  collapsed?: boolean
  mobile?: boolean
  overlay?: boolean
  placement?: WorkspaceSidebarPlacement
  surface?: WorkspaceSidebarSurface
  contentMode?: WorkspaceSidebarContentMode
}>(), {
  collapsed: false,
  mobile: false,
  overlay: false,
  placement: 'fixed',
  surface: 'work',
  contentMode: 'workspace',
})

const effectiveCollapsed = computed(() => (
  props.collapsed
  && !props.mobile
  && !props.overlay
))
</script>

<style scoped>
.workspace-sidebar-frame {
  display: flex;
  width: var(--workspace-sidebar-width);
  min-width: var(--workspace-sidebar-width);
  height: 100%;
  flex: 0 0 auto;
  flex-direction: column;
  overflow: hidden;
  border-right: 1px solid var(--workspace-divider);
  background: var(--workspace-sidebar-surface);
  transition:
    width var(--workspace-sidebar-transition-duration) var(--workspace-sidebar-transition-easing),
    min-width var(--workspace-sidebar-transition-duration) var(--workspace-sidebar-transition-easing),
    transform var(--workspace-sidebar-transition-duration) var(--workspace-sidebar-transition-easing);
}

.workspace-sidebar-frame--collapsed {
  width: var(--workspace-sidebar-width-collapsed);
  min-width: var(--workspace-sidebar-width-collapsed);
}

.workspace-sidebar-frame--overlay,
.workspace-sidebar-frame--overlay.workspace-sidebar-frame--collapsed {
  width: var(--workspace-sidebar-width);
  min-width: var(--workspace-sidebar-width);
  transform: none;
}

.workspace-sidebar-frame--fixed {
  position: fixed;
  top: var(--app-shell-top-offset);
  bottom: 0;
  left: 0;
  z-index: 40;
  height: auto;
}

.workspace-sidebar-frame--flow {
  position: relative;
}

.workspace-sidebar-frame--work {
  border-color: var(--app-shell-sidebar-border, var(--workspace-divider));
  background: var(--app-shell-sidebar-bg, var(--workspace-sidebar-surface));
}

.workspace-sidebar-frame__content {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex: 1 1 auto;
  flex-direction: column;
  overflow: hidden;
  padding: var(--workspace-sidebar-content-padding);
}

.workspace-sidebar-frame--content-admin .workspace-sidebar-frame__content {
  padding: var(--workspace-sidebar-content-padding-admin);
}

.workspace-sidebar-frame--collapsed.workspace-sidebar-frame--content-workspace
  .workspace-sidebar-frame__content {
  padding: var(--workspace-sidebar-content-padding-collapsed);
}

.workspace-sidebar-frame__footer {
  min-width: 0;
  flex: 0 0 auto;
}

.workspace-sidebar-frame--content-workspace .workspace-sidebar-frame__footer {
  border-top: 1px solid var(--workspace-footer-divider);
  padding-top: var(--workspace-space-2);
}

.workspace-sidebar-frame--chat .workspace-sidebar-frame__footer {
  background: var(--workspace-sidebar-surface);
}

.workspace-sidebar-frame--content-workspace.workspace-sidebar-frame--collapsed
  .workspace-sidebar-frame__footer {
  border-top-color: transparent;
}

.workspace-sidebar-frame--mobile,
.workspace-sidebar-frame--mobile.workspace-sidebar-frame--collapsed {
  width: var(--workspace-sidebar-width-mobile);
  min-width: var(--workspace-sidebar-width-mobile);
}

.workspace-sidebar-frame--chat.workspace-sidebar-frame--mobile,
.workspace-sidebar-frame--chat.workspace-sidebar-frame--mobile.workspace-sidebar-frame--collapsed {
  width: var(--workspace-sidebar-width-mobile-chat);
  min-width: var(--workspace-sidebar-width-mobile-chat);
}

@media (min-width: 768px) {
  .workspace-sidebar-frame--fixed {
    top: 0;
  }
}

@media (max-width: 767px) and (hover: none) and (pointer: coarse) {
  .workspace-sidebar-frame--fixed,
  .workspace-sidebar-frame--fixed.workspace-sidebar-frame--collapsed {
    width: var(--workspace-sidebar-width-mobile);
    min-width: var(--workspace-sidebar-width-mobile);
  }
}

@media (prefers-reduced-motion: reduce) {
  .workspace-sidebar-frame {
    transition-duration: 0.01ms;
  }
}
</style>
