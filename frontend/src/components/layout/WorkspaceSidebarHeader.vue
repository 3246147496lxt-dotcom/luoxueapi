<template>
  <header
    class="workspace-sidebar-header"
    :class="{
      'workspace-sidebar-header--collapsed': effectiveCollapsed,
      'workspace-sidebar-header--mobile': mobile,
      'workspace-sidebar-header--overlay': overlay,
    }"
  >
    <button
      v-if="mobile && showClose"
      ref="closeTriggerRef"
      type="button"
      class="workspace-sidebar-header__action workspace-sidebar-header__close"
      :aria-label="closeLabel"
      :title="closeLabel"
      @click="$emit('close')"
    >
      <Icon name="x" size="sm" aria-hidden="true" />
    </button>

    <div
      v-if="overlay"
      class="workspace-sidebar-header__overlay-brand-mark"
      aria-hidden="true"
    >
      <img
        :src="brandLogoSrc"
        alt=""
        class="workspace-sidebar-header__overlay-brand-image"
        :class="{
          'workspace-sidebar-header__overlay-brand-image--default': isDefaultLogo,
        }"
        draggable="false"
      >
    </div>

    <div v-else-if="!effectiveCollapsed" class="workspace-sidebar-header__brand">
      <slot name="brand" />
    </div>

    <div v-if="!effectiveCollapsed" class="workspace-sidebar-header__actions">
      <button
        v-if="showSearch"
        ref="searchTriggerRef"
        type="button"
        class="workspace-sidebar-header__action workspace-sidebar-header__search"
        :aria-label="searchLabel"
        :title="searchLabel"
        :aria-expanded="searchExpanded"
        :aria-controls="searchControls || undefined"
        @click="$emit('search')"
      >
        <WorkspaceResponsiveSidebarIcon
          v-if="overlay"
          name="search"
          :variant="searchGlyph"
          aria-hidden="true"
        />
        <WorkspaceDesktopSidebarHeaderIcon
          v-else
          name="search"
          :variant="searchGlyph"
          aria-hidden="true"
        />
      </button>

      <button
        v-if="!mobile && !overlay"
        ref="toggleTriggerRef"
        type="button"
        class="workspace-sidebar-header__action workspace-sidebar-header__toggle"
        :data-testid="toggleTestId"
        :aria-label="collapseLabel"
        :title="collapseLabel"
        :aria-controls="controls"
        aria-expanded="true"
        @click="$emit('toggle')"
      >
        <WorkspaceDesktopSidebarHeaderIcon
          name="collapse"
          :variant="searchGlyph"
          :data-testid="toggleIconTestId"
          aria-hidden="true"
        />
      </button>
    </div>

    <button
      v-if="overlay && showClose"
      ref="closeTriggerRef"
      type="button"
      class="workspace-sidebar-header__action workspace-sidebar-header__close"
      data-workspace-sidebar-overlay-close
      :aria-label="closeLabel"
      :title="closeLabel"
      @click="$emit('close')"
    >
      <WorkspaceResponsiveSidebarIcon name="close" aria-hidden="true" />
    </button>

    <button
      v-if="effectiveCollapsed && !mobile && !overlay"
      ref="toggleTriggerRef"
      type="button"
      class="workspace-sidebar-header__action workspace-sidebar-header__toggle workspace-sidebar-header__toggle--collapsed"
      :data-testid="toggleTestId"
      :aria-label="expandLabel"
      :title="expandLabel"
      :aria-controls="controls"
      aria-expanded="false"
      @click="$emit('toggle')"
    >
      <span class="workspace-sidebar-header__collapsed-brand-mark" aria-hidden="true">
        <img
          :src="brandLogoSrc"
          alt=""
          class="workspace-sidebar-header__collapsed-brand-image"
          :class="{
            'workspace-sidebar-header__collapsed-brand-image--default': isDefaultLogo,
          }"
          draggable="false"
        >
      </span>
      <WorkspaceDesktopSidebarHeaderIcon
        v-if="searchGlyph === 'chatgpt'"
        name="collapse"
        variant="chatgpt"
        class="workspace-sidebar-header__toggle-icon"
        :data-testid="toggleIconTestId"
        aria-hidden="true"
      />
      <SidebarCollapseIcon
        v-else
        class="workspace-sidebar-header__toggle-icon"
        :collapsed="true"
        :data-testid="toggleIconTestId"
        aria-hidden="true"
      />
    </button>
  </header>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import SidebarCollapseIcon from '@/components/icons/SidebarCollapseIcon.vue'
import { sanitizeUrl } from '@/utils/url'
import WorkspaceDesktopSidebarHeaderIcon from './WorkspaceDesktopSidebarHeaderIcon.vue'
import WorkspaceResponsiveSidebarIcon from './WorkspaceResponsiveSidebarIcon.vue'

const props = withDefaults(defineProps<{
  controls: string
  collapsed?: boolean
  mobile?: boolean
  overlay?: boolean
  showSearch?: boolean
  searchGlyph?: 'default' | 'chatgpt'
  showClose?: boolean
  searchExpanded?: boolean
  searchControls?: string
  searchLabel: string
  collapseLabel: string
  expandLabel: string
  closeLabel: string
  collapsedLogoSrc?: string
  toggleTestId?: string
  toggleIconTestId?: string
}>(), {
  collapsed: false,
  mobile: false,
  overlay: false,
  showSearch: false,
  searchGlyph: 'default',
  showClose: true,
  searchExpanded: false,
  searchControls: '',
  collapsedLogoSrc: '',
  toggleTestId: 'workspace-sidebar-collapse-toggle',
  toggleIconTestId: 'workspace-sidebar-collapse-toggle-icon',
})

const effectiveCollapsed = computed(() => (
  props.collapsed
  && !props.mobile
  && !props.overlay
))
const siteLogo = computed(() => sanitizeUrl(props.collapsedLogoSrc, {
  allowRelative: true,
  allowDataUrl: true,
}))
const brandLogoSrc = computed(() => siteLogo.value || '/logo.png')
const isDefaultLogo = computed(() => brandLogoSrc.value === '/logo.png')

defineEmits<{
  search: []
  toggle: []
  close: []
}>()

const searchTriggerRef = ref<HTMLButtonElement | null>(null)
const toggleTriggerRef = ref<HTMLButtonElement | null>(null)
const closeTriggerRef = ref<HTMLButtonElement | null>(null)

defineExpose({
  focusSearch: () => searchTriggerRef.value?.focus(),
  focusToggle: () => toggleTriggerRef.value?.focus(),
  focusClose: () => closeTriggerRef.value?.focus(),
})
</script>

<style scoped>
.workspace-sidebar-header {
  display: grid;
  height: var(--workspace-sidebar-header-height);
  min-width: 0;
  flex: 0 0 var(--workspace-sidebar-header-height);
  grid-template-columns: minmax(0, 1fr) auto auto;
  align-items: center;
  gap: var(--workspace-space-1);
  padding: var(--workspace-space-2) var(--workspace-sidebar-header-padding-inline);
  padding-inline-end: var(--workspace-sidebar-header-actions-inset-end);
}

.workspace-sidebar-header__brand {
  min-width: 0;
  grid-column: 1;
  grid-row: 1;
}

.workspace-sidebar-header__brand :slotted(*) {
  width: 100%;
  min-width: 0;
}

.workspace-sidebar-header__overlay-brand-mark {
  display: grid;
  width: var(--workspace-sidebar-action-size);
  height: var(--workspace-sidebar-action-size);
  place-items: center;
  grid-column: 1;
  grid-row: 1;
}

.workspace-sidebar-header__overlay-brand-image {
  display: block;
  width: var(--workspace-space-6);
  height: var(--workspace-space-6);
  object-fit: contain;
}

.workspace-sidebar-header__overlay-brand-image--default {
  transform: scale(1.08);
  transform-origin: center;
}

.workspace-sidebar-header__actions {
  display: flex;
  min-width: 0;
  grid-column: 2 / span 2;
  grid-row: 1;
  align-items: center;
  justify-content: flex-end;
  gap: 0;
}

/*
 * ChatGPT's expanded desktop rail keeps a 245px content column inside the
 * 260px shell.  The two 36px controls are flush-right with an 8px inset;
 * keeping this as an opt-in modifier leaves the Work/Account sidebars alone.
 */
@media (min-width: 768px) {
  .workspace-sidebar-header--chatgpt:not(.workspace-sidebar-header--collapsed) {
    width: 245px;
    box-sizing: border-box;
    grid-template-columns: minmax(0, 1fr) calc(var(--workspace-sidebar-action-size) * 2);
    gap: 0;
    padding: 8px;
  }

  .workspace-sidebar-header--chatgpt:not(.workspace-sidebar-header--collapsed)
    .workspace-sidebar-header__actions {
    width: calc(var(--workspace-sidebar-action-size) * 2);
    grid-column: 2;
    gap: 0;
  }
}

.workspace-sidebar-header__action {
  display: grid;
  width: var(--workspace-sidebar-action-size);
  height: var(--workspace-sidebar-action-size);
  place-items: center;
  flex: 0 0 var(--workspace-sidebar-action-size);
  border: 0;
  border-radius: var(--workspace-radius-compact);
  padding: 0;
  color: var(--workspace-sidebar-header-action);
  background: transparent;
  transition:
    color 150ms ease,
    background-color 150ms ease;
}

.workspace-sidebar-header__action:hover {
  color: var(--workspace-text);
  background: var(--workspace-hover);
}

.workspace-sidebar-header__action:focus-visible {
  outline: 2px solid var(--workspace-text-secondary);
  outline-offset: 2px;
}

.workspace-sidebar-header__toggle {
  cursor: w-resize;
}

.workspace-sidebar-header__toggle--collapsed {
  width: var(--workspace-sidebar-touch-target);
  height: var(--workspace-sidebar-touch-target);
  flex-basis: var(--workspace-sidebar-touch-target);
  cursor: pointer;
}

.workspace-sidebar-header__toggle-icon {
  width: 18px;
  height: 18px;
}

.workspace-sidebar-header__collapsed-brand-mark,
.workspace-sidebar-header__toggle--collapsed .workspace-sidebar-header__toggle-icon {
  grid-area: 1 / 1;
  pointer-events: none;
  transition:
    opacity 180ms cubic-bezier(0.16, 1, 0.3, 1),
    transform 180ms cubic-bezier(0.16, 1, 0.3, 1);
}

.workspace-sidebar-header__collapsed-brand-mark {
  display: grid;
  width: 28px;
  height: 28px;
  place-items: center;
  opacity: 1;
  transform: scale(1);
}

.workspace-sidebar-header__collapsed-brand-image {
  display: block;
  width: 100%;
  height: 100%;
  max-width: none;
  object-fit: contain;
}

.workspace-sidebar-header__collapsed-brand-image--default {
  transform: scale(1.1);
  transform-origin: center;
}

.workspace-sidebar-header__toggle--collapsed .workspace-sidebar-header__toggle-icon {
  opacity: 0;
  transform: scale(0.9);
}

@media (hover: hover) and (pointer: fine) {
  .workspace-sidebar-header__toggle--collapsed:hover
    .workspace-sidebar-header__collapsed-brand-mark {
    opacity: 0;
    transform: scale(0.9);
  }

  .workspace-sidebar-header__toggle--collapsed:hover
    .workspace-sidebar-header__toggle-icon {
    opacity: 1;
    transform: scale(1);
  }
}

.workspace-sidebar-header__toggle--collapsed:focus-visible
  .workspace-sidebar-header__collapsed-brand-mark {
  opacity: 0;
  transform: scale(0.9);
}

.workspace-sidebar-header__toggle--collapsed:focus-visible
  .workspace-sidebar-header__toggle-icon {
  opacity: 1;
  transform: scale(1);
}

.workspace-sidebar-header__close {
  grid-column: 3;
  grid-row: 1;
}

.workspace-sidebar-header--collapsed {
  grid-template-columns: var(--workspace-sidebar-touch-target);
  justify-content: start;
  padding-top: var(--workspace-space-1);
  padding-right: var(--workspace-space-2);
  padding-bottom: var(--workspace-space-1);
  padding-left: var(--workspace-space-2);
}

.workspace-sidebar-header--collapsed .workspace-sidebar-header__toggle {
  grid-column: 1;
  grid-row: 1;
}

.workspace-sidebar-header--mobile .workspace-sidebar-header__action {
  width: var(--workspace-sidebar-touch-target);
  height: var(--workspace-sidebar-touch-target);
  flex-basis: var(--workspace-sidebar-touch-target);
}

.workspace-sidebar-header--mobile {
  padding-top: var(--workspace-space-1);
  padding-bottom: var(--workspace-space-1);
}

.workspace-sidebar-header--mobile .workspace-sidebar-header__actions {
  grid-column: 2;
}

.workspace-sidebar-header--overlay {
  grid-template-columns:
    minmax(0, 1fr)
    var(--workspace-sidebar-action-size)
    var(--workspace-sidebar-action-size);
  gap: 0;
  padding-block: var(--workspace-space-2);
  padding-inline-start: var(--workspace-space-2);
  padding-inline-end: var(--workspace-space-6);
}

.workspace-sidebar-header--overlay .workspace-sidebar-header__actions {
  grid-column: 2;
  gap: 0;
}

.workspace-sidebar-header--overlay .workspace-sidebar-header__action {
  color: var(--workspace-sidebar-overlay-icon);
}

.workspace-sidebar-header--overlay .workspace-sidebar-header__close {
  grid-column: 3;
}

:global([dir='rtl']) .workspace-sidebar-header__toggle {
  cursor: e-resize;
}

:global([dir='rtl']) .workspace-sidebar-header__toggle--collapsed {
  cursor: pointer;
}

@media (prefers-reduced-motion: reduce) {
  .workspace-sidebar-header__action {
    transition: none;
  }

  .workspace-sidebar-header__collapsed-brand-mark,
  .workspace-sidebar-header__toggle--collapsed .workspace-sidebar-header__toggle-icon {
    transition: none;
  }
}
</style>
