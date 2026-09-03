<template>
  <slot v-if="!active" />

  <Teleport v-else to="body">
    <div
      class="workspace-sidebar-overlay-layer"
      :class="{ 'workspace-sidebar-overlay-layer--open': open }"
      :aria-hidden="open ? undefined : 'true'"
      :inert="open ? undefined : true"
      data-testid="workspace-sidebar-overlay-layer"
    >
      <button
        type="button"
        class="workspace-sidebar-overlay-layer__scrim"
        tabindex="-1"
        aria-hidden="true"
        @click="requestClose"
      ></button>

      <section
        ref="panelRef"
        class="workspace-sidebar-overlay-layer__panel"
        role="dialog"
        aria-modal="true"
        :aria-label="label"
        tabindex="-1"
        data-testid="workspace-sidebar-overlay-panel"
      >
        <div class="workspace-sidebar-overlay-layer__content">
          <slot />
        </div>
      </section>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { acquireBodyScrollLock, releaseBodyScrollLock } from '@/utils/bodyScrollLock'
import {
  getVisibleFocusableElements,
  isTopModalLayer,
  registerModalLayer,
  unregisterModalLayer,
} from '@/utils/modalStack'

const props = defineProps<{
  active: boolean
  open: boolean
  label: string
  returnFocusId: string
}>()

const emit = defineEmits<{
  close: []
}>()

const panelRef = ref<HTMLElement | null>(null)
const modalLayerToken = Symbol('workspace-sidebar-overlay-layer')
const scrollLockToken = Symbol('workspace-sidebar-overlay-scroll-lock')
let modalActive = false

function requestClose() {
  if (!props.open || !isTopModalLayer(modalLayerToken)) return
  emit('close')
}

function handleDocumentKeydown(event: KeyboardEvent) {
  if (!props.open || !modalActive || !isTopModalLayer(modalLayerToken)) return

  if (event.key === 'Escape') {
    event.preventDefault()
    event.stopPropagation()
    emit('close')
    return
  }

  if (event.key !== 'Tab') return

  const panel = panelRef.value
  if (!panel) return

  const focusable = getVisibleFocusableElements(panel)
  if (focusable.length === 0) {
    event.preventDefault()
    panel.focus({ preventScroll: true })
    return
  }

  const first = focusable[0]
  const last = focusable.at(-1)
  const focused = document.activeElement
  const focusIsOutside = !panel.contains(focused)

  if (
    event.shiftKey
    && (focused === first || focused === panel || focusIsOutside)
  ) {
    event.preventDefault()
    last?.focus({ preventScroll: true })
  } else if (
    !event.shiftKey
    && (focused === last || focused === panel || focusIsOutside)
  ) {
    event.preventDefault()
    first.focus({ preventScroll: true })
  }
}

function restoreTriggerFocus() {
  const trigger = document.getElementById(props.returnFocusId)
  if (trigger instanceof HTMLElement && trigger.isConnected) {
    trigger.focus({ preventScroll: true })
  }
}

async function activateModal() {
  if (modalActive || typeof document === 'undefined') return

  modalActive = true
  registerModalLayer(modalLayerToken)
  acquireBodyScrollLock(scrollLockToken)
  document.addEventListener('keydown', handleDocumentKeydown)

  await nextTick()
  if (modalActive && props.active && props.open) {
    const closeButton = panelRef.value?.querySelector<HTMLElement>(
      '[data-workspace-sidebar-overlay-close]',
    )
    ;(closeButton ?? panelRef.value)?.focus({ preventScroll: true })
  }
}

function deactivateModal(restoreFocus: boolean) {
  if (!modalActive || typeof document === 'undefined') return

  modalActive = false
  document.removeEventListener('keydown', handleDocumentKeydown)
  unregisterModalLayer(modalLayerToken)
  releaseBodyScrollLock(scrollLockToken)

  if (restoreFocus) {
    void nextTick(restoreTriggerFocus)
  }
}

watch(
  [() => props.active, () => props.open],
  ([active, open]) => {
    if (active && open) {
      void activateModal()
      return
    }
    deactivateModal(true)
  },
  { immediate: true, flush: 'post' },
)

onBeforeUnmount(() => {
  deactivateModal(false)
})
</script>

<style scoped>
.workspace-sidebar-overlay-layer {
  position: fixed;
  inset: 0;
  z-index: var(--workspace-layer-sidebar-overlay);
  visibility: hidden;
  pointer-events: none;
  transition: visibility 0s linear 190ms;
}

.workspace-sidebar-overlay-layer--open {
  visibility: visible;
  pointer-events: auto;
  transition-delay: 0s;
}

.workspace-sidebar-overlay-layer__scrim {
  position: absolute;
  inset: 0;
  border: 0;
  padding: 0;
  opacity: 0;
  background: var(--workspace-sidebar-overlay-backdrop);
  cursor: default;
  transition: opacity 190ms ease-out;
}

.workspace-sidebar-overlay-layer__panel {
  position: absolute;
  inset: 0 auto 0 0;
  width: var(--workspace-sidebar-width);
  max-width: 100%;
  overflow: hidden;
  color: var(--workspace-text);
  background: var(--workspace-sidebar-surface);
  box-shadow: var(--workspace-sidebar-overlay-shadow);
  transform: translateX(-100%);
  transition: transform 190ms ease-out;
}

.workspace-sidebar-overlay-layer--open .workspace-sidebar-overlay-layer__scrim {
  opacity: 1;
}

.workspace-sidebar-overlay-layer--open .workspace-sidebar-overlay-layer__panel {
  transform: translateX(0);
}

.workspace-sidebar-overlay-layer__panel:focus-visible {
  outline: var(--workspace-space-0-5) solid var(--workspace-text-secondary);
  outline-offset: calc(-1 * var(--workspace-space-0-5));
}

.workspace-sidebar-overlay-layer__content {
  width: 100%;
  height: 100%;
  overflow-x: hidden;
  overflow-y: auto;
  overscroll-behavior: contain;
}

@media (prefers-reduced-motion: reduce) {
  .workspace-sidebar-overlay-layer,
  .workspace-sidebar-overlay-layer__scrim,
  .workspace-sidebar-overlay-layer__panel {
    transition: none;
  }
}
</style>
