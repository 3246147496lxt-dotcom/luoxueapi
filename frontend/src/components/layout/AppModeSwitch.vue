<template>
  <div
    class="app-mode-switch"
    data-testid="app-mode-switch"
    :data-active-mode="activeMode"
  >
    <div
      class="app-mode-switch__control"
      role="group"
      :aria-label="t('nav.productMode')"
    >
      <button
        type="button"
        class="app-mode-switch__option"
        :class="{ 'app-mode-switch__option--active': activeMode === 'chat' }"
        :aria-pressed="activeMode === 'chat'"
        @click="requestMode('chat')"
      >
        {{ chatLabel || t('nav.chatMode') }}
      </button>
      <button
        type="button"
        class="app-mode-switch__option"
        :class="{ 'app-mode-switch__option--active': activeMode === 'work' }"
        :aria-pressed="activeMode === 'work'"
        @click="requestMode('work')"
      >
        {{ workLabel || t('nav.workMode') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, useAttrs } from 'vue'
import { useI18n } from 'vue-i18n'

export type AppShellMode = 'chat' | 'work'

defineOptions({ inheritAttrs: false })

const props = defineProps<{ activeMode: AppShellMode }>()
const attrs = useAttrs()
const chatLabel = computed(() => {
  const value = attrs.chatLabel ?? attrs['chat-label']
  return typeof value === 'string' ? value : ''
})
const workLabel = computed(() => {
  const value = attrs.workLabel ?? attrs['work-label']
  return typeof value === 'string' ? value : ''
})

const emit = defineEmits<{
  change: [mode: AppShellMode]
}>()

const { t } = useI18n()

function requestMode(mode: AppShellMode) {
  if (mode !== props.activeMode) emit('change', mode)
}
</script>

<style scoped>
.app-mode-switch {
  position: relative;
  z-index: 2;
  height: var(--workspace-mode-switch-height);
  flex: 0 0 var(--workspace-mode-switch-height);
  padding: 0 var(--workspace-space-3) var(--workspace-space-2);
}

.app-mode-switch::after {
  position: absolute;
  right: var(--workspace-space-3);
  bottom: 0;
  left: var(--workspace-space-3);
  height: 1px;
  background: var(--workspace-mode-switch-divider);
  content: '';
}

.app-mode-switch__control {
  display: grid;
  height: 100%;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--workspace-space-0-5);
  min-width: 0;
  padding: var(--workspace-space-mode-switch-inset);
  border: 1px solid var(--workspace-mode-switch-border);
  border-radius: var(--workspace-mode-switch-radius);
  background: var(--workspace-mode-switch-track);
}

.app-mode-switch__option {
  display: inline-flex;
  min-width: 0;
  min-height: 30px;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: var(--workspace-mode-switch-option-radius);
  padding: 0 var(--workspace-space-2);
  color: var(--workspace-mode-switch-text-muted);
  background: transparent;
  cursor: pointer;
  font: inherit;
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
  line-height: 1;
  transition: color 160ms ease, background-color 160ms ease, box-shadow 160ms ease;
}

.app-mode-switch__option:not(.app-mode-switch__option--active):hover {
  color: var(--workspace-mode-switch-text);
  background: var(--workspace-mode-switch-hover);
}

.app-mode-switch__option--active,
.app-mode-switch__option--active:hover {
  color: var(--workspace-mode-switch-text);
  background: var(--workspace-mode-switch-active);
  box-shadow: var(--workspace-mode-switch-active-shadow);
}

.app-mode-switch__option:focus-visible {
  outline: 2px solid var(--workspace-mode-switch-text-muted);
  outline-offset: 2px;
}

:global(.workspace-sidebar-frame--mobile) .app-mode-switch {
  height: var(--workspace-mode-switch-height-mobile);
  flex-basis: var(--workspace-mode-switch-height-mobile);
}

:global(.workspace-sidebar-frame--mobile) .app-mode-switch__option {
  min-height: var(--workspace-sidebar-touch-target);
}

@media (prefers-reduced-motion: reduce) {
  .app-mode-switch__option {
    transition: none;
  }
}
</style>
