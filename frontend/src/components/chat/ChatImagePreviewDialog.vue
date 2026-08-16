<template>
  <Teleport to="body">
    <Transition name="chat-image-preview">
      <div
        v-if="show && src"
        ref="dialogRef"
        class="chat-image-preview"
        role="dialog"
        aria-modal="true"
        :aria-label="t('chat.attachments.imagePreview', { name })"
        tabindex="-1"
        data-test="chat-image-preview-dialog"
        @click.self="emit('close')"
      >
        <button
          ref="closeButtonRef"
          type="button"
          class="chat-image-preview__close"
          :aria-label="t('common.close')"
          data-test="chat-image-preview-close"
          @click="emit('close')"
        >
          <Icon name="x" size="md" aria-hidden="true" />
        </button>

        <img
          class="chat-image-preview__image"
          :src="src"
          :alt="name"
          data-test="chat-image-preview-image"
        />
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { acquireBodyScrollLock, releaseBodyScrollLock } from '@/utils/bodyScrollLock'
import {
  getVisibleFocusableElements,
  isTopModalLayer,
  registerModalLayer,
  unregisterModalLayer,
} from '@/utils/modalStack'

const props = withDefaults(defineProps<{
  show: boolean
  src?: string
  name?: string
}>(), {
  src: '',
  name: '',
})

const emit = defineEmits<{
  close: []
}>()

const { t } = useI18n()
const dialogRef = ref<HTMLElement | null>(null)
const closeButtonRef = ref<HTMLButtonElement | null>(null)
const modalLayerToken = Symbol('chat-image-preview')
const scrollLockToken = Symbol('chat-image-preview-scroll-lock')
let previousActiveElement: HTMLElement | null = null
let modalActive = false

function handleDocumentKeydown(event: KeyboardEvent) {
  if (!modalActive || !isTopModalLayer(modalLayerToken)) return

  if (event.key === 'Escape') {
    event.preventDefault()
    emit('close')
    return
  }

  if (event.key !== 'Tab') return
  const dialog = dialogRef.value
  if (!dialog) return
  const focusable = getVisibleFocusableElements(dialog)
  if (focusable.length === 0) {
    event.preventDefault()
    dialog.focus({ preventScroll: true })
    return
  }

  const first = focusable[0]
  const last = focusable.at(-1)
  const activeElement = document.activeElement
  const focusIsOutside = !dialog.contains(activeElement)
  if (event.shiftKey && (activeElement === first || activeElement === dialog || focusIsOutside)) {
    event.preventDefault()
    last?.focus({ preventScroll: true })
  } else if (!event.shiftKey && (activeElement === last || activeElement === dialog || focusIsOutside)) {
    event.preventDefault()
    first.focus({ preventScroll: true })
  }
}

async function activateModal() {
  if (modalActive || typeof document === 'undefined') return
  modalActive = true
  previousActiveElement = document.activeElement instanceof HTMLElement
    ? document.activeElement
    : null
  registerModalLayer(modalLayerToken)
  acquireBodyScrollLock(scrollLockToken)
  document.addEventListener('keydown', handleDocumentKeydown)

  await nextTick()
  if (modalActive) closeButtonRef.value?.focus({ preventScroll: true })
}

function deactivateModal(restoreFocus: boolean) {
  if (!modalActive || typeof document === 'undefined') return
  modalActive = false
  document.removeEventListener('keydown', handleDocumentKeydown)
  unregisterModalLayer(modalLayerToken)
  releaseBodyScrollLock(scrollLockToken)

  const returnTarget = previousActiveElement
  previousActiveElement = null
  if (restoreFocus && returnTarget?.isConnected) {
    void nextTick(() => returnTarget.focus({ preventScroll: true }))
  }
}

watch(
  () => props.show && Boolean(props.src),
  (isOpen) => {
    if (isOpen) void activateModal()
    else deactivateModal(true)
  },
  { immediate: true, flush: 'post' },
)

onBeforeUnmount(() => deactivateModal(false))
</script>

<style scoped>
.chat-image-preview {
  position: fixed;
  z-index: 70;
  inset: 0;
  display: grid;
  padding: 56px 24px 24px;
  place-items: center;
  background: rgb(0 0 0 / 86%);
  backdrop-filter: blur(2px);
}

.chat-image-preview__image {
  display: block;
  max-width: min(92vw, 1440px);
  max-height: calc(100vh - 80px);
  border-radius: 8px;
  object-fit: contain;
  box-shadow: 0 24px 80px rgb(0 0 0 / 42%);
}

.chat-image-preview__close {
  position: absolute;
  inset-block-start: 12px;
  inset-inline-end: 12px;
  display: grid;
  width: 44px;
  height: 44px;
  border: 0;
  border-radius: 50%;
  padding: 0;
  place-items: center;
  color: #212121;
  background: #fff;
  box-shadow: 0 2px 12px rgb(0 0 0 / 24%);
  cursor: pointer;
  transition: background-color 120ms ease, transform 120ms ease;
}

.chat-image-preview__close:hover {
  background: #ececec;
  transform: scale(1.04);
}

.chat-image-preview__close:focus-visible {
  outline: 3px solid var(--lx-clay-accent);
  outline-offset: 3px;
}

.chat-image-preview-enter-active,
.chat-image-preview-leave-active {
  transition: opacity 160ms ease;
}

.chat-image-preview-enter-from,
.chat-image-preview-leave-to {
  opacity: 0;
}

@media (max-width: 640px) {
  .chat-image-preview {
    padding: 64px 12px 16px;
  }

  .chat-image-preview__image {
    max-width: 100%;
    max-height: calc(100vh - 80px);
  }
}

@media (prefers-reduced-motion: reduce) {
  .chat-image-preview,
  .chat-image-preview__close {
    transition: none;
  }
}
</style>
