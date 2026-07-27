<template>
  <Teleport to="body">
    <Transition name="api-key-detail-sheet">
      <div
        v-if="show"
        class="api-key-detail-sheet__overlay"
        data-test="api-key-detail-sheet-overlay"
        @click.self="requestClose"
      >
        <section
          ref="sheetRef"
          class="api-key-detail-sheet"
          role="dialog"
          aria-modal="true"
          :aria-labelledby="titleId"
          :aria-describedby="subtitle ? subtitleId : undefined"
          tabindex="-1"
          data-test="api-key-detail-sheet"
          @click.stop
        >
          <header class="api-key-detail-sheet__header">
            <div class="api-key-detail-sheet__identity">
              <div class="api-key-detail-sheet__info-icon" aria-hidden="true">
                <KeysLucideIcon name="info" :size="20" />
              </div>
              <div class="min-w-0">
                <h2 :id="titleId" class="api-key-detail-sheet__title truncate">
                  {{ title }}
                </h2>
                <p
                  v-if="subtitle"
                  :id="subtitleId"
                  class="api-key-detail-sheet__subtitle truncate"
                  :title="subtitle"
                >
                  {{ subtitle }}
                </p>
              </div>
            </div>
            <button
              ref="closeButtonRef"
              type="button"
              class="api-key-detail-sheet__close"
              :title="t('common.close')"
              :aria-label="t('common.close')"
              data-test="api-key-detail-sheet-close"
              @click.stop="requestClose"
            >
              <KeysLucideIcon name="x" :size="24" />
            </button>
          </header>

          <div class="api-key-detail-sheet__content" data-test="api-key-detail-sheet-content">
            <slot name="content">
              <slot />
            </slot>
          </div>
        </section>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import KeysLucideIcon from '@/components/keys/KeysLucideIcon.vue'

let detailSheetId = 0
const instanceId = ++detailSheetId
const titleId = `api-key-detail-sheet-title-${instanceId}`
const subtitleId = `api-key-detail-sheet-subtitle-${instanceId}`
const BODY_LOCK_CLASS = 'api-key-detail-sheet-open'
const FOCUSABLE_SELECTOR = [
  'a[href]',
  'button:not([disabled])',
  'input:not([disabled])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  '[tabindex]:not([tabindex="-1"])',
].join(',')

const props = withDefaults(defineProps<{
  show: boolean
  title: string
  subtitle?: string
}>(), {
  subtitle: '',
})

const emit = defineEmits<{
  (event: 'close'): void
}>()

const { t } = useI18n()
const sheetRef = ref<HTMLElement | null>(null)
const closeButtonRef = ref<HTMLButtonElement | null>(null)
let previousActiveElement: HTMLElement | null = null
let ownsBodyLock = false

function requestClose() {
  emit('close')
}

function lockBodyScroll() {
  if (typeof document === 'undefined' || document.body.classList.contains(BODY_LOCK_CLASS)) return
  document.body.classList.add(BODY_LOCK_CLASS)
  ownsBodyLock = true
}

function unlockBodyScroll() {
  if (typeof document === 'undefined' || !ownsBodyLock) return
  document.body.classList.remove(BODY_LOCK_CLASS)
  ownsBodyLock = false
}

function restoreFocus() {
  const target = previousActiveElement
  previousActiveElement = null
  if (!target?.isConnected) return
  try {
    target.focus({ preventScroll: true })
  } catch {
    target.focus()
  }
}

function focusSheet() {
  const target = closeButtonRef.value || sheetRef.value
  if (!target) return
  try {
    target.focus({ preventScroll: true })
  } catch {
    target.focus()
  }
}

function focusableElements() {
  if (!sheetRef.value) return []
  return Array.from(sheetRef.value.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR))
    .filter((element) => element.getAttribute('aria-hidden') !== 'true')
}

function handleDocumentKeydown(event: KeyboardEvent) {
  if (!props.show) return

  if (event.key === 'Escape') {
    event.preventDefault()
    event.stopPropagation()
    requestClose()
    return
  }

  if (event.key !== 'Tab') return
  const focusable = focusableElements()
  if (focusable.length === 0) {
    event.preventDefault()
    sheetRef.value?.focus()
    return
  }

  const first = focusable[0]
  const last = focusable[focusable.length - 1]
  const active = document.activeElement as HTMLElement | null

  if (!sheetRef.value?.contains(active)) {
    event.preventDefault()
    ;(event.shiftKey ? last : first).focus()
    return
  }
  if (event.shiftKey && active === first) {
    event.preventDefault()
    last.focus()
  } else if (!event.shiftKey && active === last) {
    event.preventDefault()
    first.focus()
  }
}

watch(
  () => props.show,
  async (isOpen) => {
    if (isOpen) {
      if (typeof document !== 'undefined') {
        previousActiveElement = document.activeElement as HTMLElement | null
      }
      lockBodyScroll()
      await nextTick()
      if (props.show) focusSheet()
      return
    }

    unlockBodyScroll()
    restoreFocus()
  },
  { immediate: true },
)

onMounted(() => {
  document.addEventListener('keydown', handleDocumentKeydown, true)
})

onBeforeUnmount(() => {
  document.removeEventListener('keydown', handleDocumentKeydown, true)
  unlockBodyScroll()
  restoreFocus()
})
</script>

<style scoped>
:global(body.api-key-detail-sheet-open) {
  overflow: hidden;
}

.api-key-detail-sheet__overlay {
  position: fixed;
  inset: 0;
  z-index: 100;
  display: flex;
  align-items: flex-end;
  justify-content: center;
  background: rgb(0 0 0 / 0.7);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
}

.api-key-detail-sheet {
  display: flex;
  width: 100%;
  max-height: 94vh;
  min-height: 0;
  flex-direction: column;
  overflow: hidden;
  border: 0;
  border-radius: 32px 32px 0 0;
  outline: none;
  color: #332f3a;
  background: #f4f1fa;
  box-shadow: 0 25px 50px -12px rgb(0 0 0 / 0.25);
  font-family: "DM Sans", "PingFang SC", "Microsoft YaHei", system-ui, sans-serif;
}

.api-key-detail-sheet:focus-visible {
  outline: 2px solid var(--lx-clay-accent);
  outline-offset: -2px;
}

.api-key-detail-sheet__header {
  position: sticky;
  top: 0;
  z-index: 10;
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: space-between;
  gap: 1.5rem;
  border-bottom: 1px solid rgb(243 244 246);
  border-radius: 32px 32px 0 0;
  background: #fff;
  padding: 1.5rem;
  box-shadow: 0 1px 2px 0 rgb(0 0 0 / 0.05);
}

.api-key-detail-sheet__identity {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 0.75rem;
}

.api-key-detail-sheet__info-icon {
  display: grid;
  width: 2.5rem;
  height: 2.5rem;
  flex: 0 0 2.5rem;
  place-items: center;
  border-radius: 0.75rem;
  color: #7c3aed;
  background: #ede9fe;
}

.api-key-detail-sheet__title {
  margin: 0;
  color: var(--lx-clay-text);
  font-size: 1rem;
  font-weight: 900;
  line-height: 1.25rem;
}

.api-key-detail-sheet__subtitle {
  margin: 0.125rem 0 0;
  color: #9ca3af;
  font-size: 0.625rem;
  font-weight: 900;
  line-height: 0.875rem;
  letter-spacing: 0;
  text-transform: uppercase;
}

.api-key-detail-sheet__close {
  display: inline-flex;
  width: 3rem;
  height: 3rem;
  flex: 0 0 3rem;
  align-items: center;
  justify-content: center;
  color: #9ca3af;
  transition: color 150ms ease, transform 150ms ease;
}

.api-key-detail-sheet__close:hover {
  color: var(--lx-clay-text);
}

.api-key-detail-sheet__close:active {
  transform: scale(0.9);
}

.api-key-detail-sheet__close:focus-visible {
  outline: 2px solid var(--lx-clay-accent);
  outline-offset: 2px;
}

.api-key-detail-sheet__content {
  display: flex;
  min-height: 0;
  flex: 1 1 auto;
  overflow: hidden;
  overscroll-behavior: contain;
}

.api-key-detail-sheet__content :deep(.api-key-inspector__footer) {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.75rem;
  flex: 0 0 auto;
  border-top: 1px solid rgb(243 244 246);
  background: #fff;
  padding: 1.5rem;
  padding-bottom: max(2rem, calc(1.5rem + env(safe-area-inset-bottom)));
}

.api-key-detail-sheet-enter-active,
.api-key-detail-sheet-leave-active {
  transition: opacity 300ms ease;
}

.api-key-detail-sheet-enter-active .api-key-detail-sheet,
.api-key-detail-sheet-leave-active .api-key-detail-sheet {
  transition: transform 300ms cubic-bezier(0.4, 0, 0.2, 1);
}

.api-key-detail-sheet-enter-from,
.api-key-detail-sheet-leave-to {
  opacity: 0;
}

.api-key-detail-sheet-enter-from .api-key-detail-sheet,
.api-key-detail-sheet-leave-to .api-key-detail-sheet {
  transform: translateY(100%);
}

:global(.dark) .api-key-detail-sheet {
  color: #f8f5fc;
  background: #17131f;
}

:global(.dark) .api-key-detail-sheet__header,
:global(.dark) .api-key-detail-sheet__content :deep(.api-key-inspector__footer) {
  border-color: rgb(255 255 255 / 0.1);
  background: #251e2f;
}

:global(.dark) .api-key-detail-sheet__info-icon {
  color: #7c3aed;
  background: rgb(76 29 149 / 0.3);
}

@media (prefers-reduced-motion: reduce) {
  .api-key-detail-sheet__close,
  .api-key-detail-sheet-enter-active,
  .api-key-detail-sheet-leave-active,
  .api-key-detail-sheet-enter-active .api-key-detail-sheet,
  .api-key-detail-sheet-leave-active .api-key-detail-sheet {
    transition-duration: 0.01ms;
  }
}
</style>
