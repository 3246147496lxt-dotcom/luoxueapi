<template>
  <Teleport to="body">
    <Transition name="modal">
      <div
        v-if="show"
        ref="dialogRef"
        class="modal-overlay"
        :class="{ 'modal-overlay--workspace-confirm': variant === 'workspace-confirm' }"
        :style="zIndexStyle"
        :aria-labelledby="dialogId"
        :aria-describedby="descriptionId || undefined"
        role="dialog"
        aria-modal="true"
        tabindex="-1"
        @click.self="handleClose"
      >
        <!-- Modal panel -->
        <div
          class="modal-content"
          :class="[
            widthClasses,
            { 'modal-content--workspace-confirm': variant === 'workspace-confirm' },
          ]"
          @click.stop
        >
          <!-- Header -->
          <div class="modal-header">
            <h3 :id="dialogId" class="modal-title">
              {{ title }}
            </h3>
            <button
              v-if="showCloseButton"
              @click="emit('close')"
              class="-mr-2 rounded-xl p-2 text-[var(--lx-clay-text-muted)] transition-colors hover:bg-[var(--lx-clay-hover)] hover:text-[var(--lx-clay-text)] focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/30 focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--lx-clay-surface-elevated)]"
              aria-label="Close modal"
            >
              <Icon name="x" size="md" />
            </button>
          </div>

          <!-- Body -->
          <div class="modal-body">
            <slot></slot>
          </div>

          <!-- Footer -->
          <div v-if="$slots.footer" class="modal-footer">
            <slot name="footer"></slot>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, watch, onUnmounted, ref, nextTick } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import { acquireBodyScrollLock, releaseBodyScrollLock } from '@/utils/bodyScrollLock'
import {
  getVisibleFocusableElements,
  isTopModalLayer,
  registerModalLayer,
  unregisterModalLayer,
} from '@/utils/modalStack'

// 生成唯一ID以避免多个对话框时ID冲突
let dialogIdCounter = 0
const dialogId = `modal-title-${++dialogIdCounter}`

// 焦点管理
const dialogRef = ref<HTMLElement | null>(null)
let previousActiveElement: HTMLElement | null = null
const modalLayerToken = Symbol('base-dialog')
const scrollLockToken = Symbol('base-dialog-scroll-lock')
let modalActive = false

type DialogWidth = 'narrow' | 'normal' | 'wide' | 'extra-wide' | 'full'
type DialogVariant = 'default' | 'workspace-confirm'

interface Props {
  show: boolean
  title: string
  width?: DialogWidth
  closeOnEscape?: boolean
  closeOnClickOutside?: boolean
  showCloseButton?: boolean
  zIndex?: number
  variant?: DialogVariant
  descriptionId?: string
}

interface Emits {
  (e: 'close'): void
}

const props = withDefaults(defineProps<Props>(), {
  width: 'normal',
  closeOnEscape: true,
  closeOnClickOutside: false,
  showCloseButton: true,
  zIndex: 50,
  variant: 'default',
  descriptionId: '',
})

const emit = defineEmits<Emits>()

// Custom z-index style (overrides the default z-50 from CSS)
const zIndexStyle = computed(() => {
  return props.zIndex !== 50 ? { zIndex: props.zIndex } : undefined
})

const widthClasses = computed(() => {
  // Width guidance: narrow=confirm/short prompts, normal=standard forms,
  // wide=multi-section forms or rich content, extra-wide=analytics/tables,
  // full=full-screen or very dense layouts.
  const widths: Record<DialogWidth, string> = {
    narrow: 'max-w-md',
    normal: 'max-w-lg',
    wide: 'w-full sm:max-w-2xl md:max-w-3xl lg:max-w-4xl',
    'extra-wide': 'w-full sm:max-w-3xl md:max-w-4xl lg:max-w-5xl xl:max-w-6xl',
    full: 'w-full sm:max-w-4xl md:max-w-5xl lg:max-w-6xl xl:max-w-7xl'
  }
  return widths[props.width]
})

const handleClose = () => {
  if (props.closeOnClickOutside && isTopModalLayer(modalLayerToken)) {
    emit('close')
  }
}

const handleDocumentKeydown = (event: KeyboardEvent) => {
  if (!props.show || !modalActive || !isTopModalLayer(modalLayerToken)) return

  if (event.key === 'Escape' && props.closeOnEscape) {
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

  if (
    event.shiftKey
    && (activeElement === first || activeElement === dialog || focusIsOutside)
  ) {
    event.preventDefault()
    last?.focus({ preventScroll: true })
  } else if (
    !event.shiftKey
    && (activeElement === last || activeElement === dialog || focusIsOutside)
  ) {
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
  document.body.classList.add('modal-open')
  document.addEventListener('keydown', handleDocumentKeydown)

  await nextTick()
  const dialog = dialogRef.value
  if (!modalActive || !dialog) return
  const firstFocusable = getVisibleFocusableElements(dialog)[0]
  ;(firstFocusable ?? dialog).focus({ preventScroll: true })
}

function deactivateModal(restoreFocus: boolean) {
  if (!modalActive || typeof document === 'undefined') return

  modalActive = false
  document.removeEventListener('keydown', handleDocumentKeydown)
  unregisterModalLayer(modalLayerToken)
  releaseBodyScrollLock(scrollLockToken)
  document.body.classList.remove('modal-open')

  const returnTarget = previousActiveElement
  previousActiveElement = null
  if (restoreFocus && returnTarget?.isConnected) {
    void nextTick(() => returnTarget.focus({ preventScroll: true }))
  }
}

watch(
  () => props.show,
  (isOpen) => {
    if (isOpen) {
      void activateModal()
    } else {
      deactivateModal(true)
    }
  },
  { immediate: true, flush: 'post' },
)

onUnmounted(() => {
  deactivateModal(false)
})
</script>

<style scoped>
.modal-overlay--workspace-confirm {
  padding-block: var(--workspace-space-4);
  padding-inline: var(--workspace-space-2);
  padding-block-end: calc(var(--workspace-space-4) + 88px);
  background: var(--workspace-confirm-backdrop);
  backdrop-filter: blur(1px);
}

.modal-content--workspace-confirm {
  overflow: hidden;
  border: 0;
  border-radius: var(--workspace-radius-card);
  color: var(--workspace-confirm-text);
  background: var(--workspace-confirm-surface);
  box-shadow: var(--workspace-confirm-shadow);
}

.modal-content--workspace-confirm .modal-header {
  height: 52px;
  padding: 10px 10px 10px var(--workspace-space-4);
  border: 0;
}

.modal-content--workspace-confirm .modal-title {
  color: var(--workspace-confirm-text);
  font-size: 18px;
  font-weight: var(--workspace-type-body-weight);
  line-height: 28px;
}

.modal-content--workspace-confirm .modal-body {
  padding: var(--workspace-space-1) var(--workspace-space-4) 0;
  overflow: visible;
}

.modal-content--workspace-confirm .modal-footer {
  min-height: 68px;
  padding: var(--workspace-space-4);
  gap: var(--workspace-space-3);
  border: 0;
}

@media (max-height: 360px) {
  .modal-overlay--workspace-confirm {
    padding-block-end: var(--workspace-space-4);
  }
}
</style>
