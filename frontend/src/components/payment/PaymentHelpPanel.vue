<template>
  <section
    v-if="text || imageUrl"
    class="payment-help-panel rounded-xl p-4"
    :aria-label="t('payment.helpTitle')"
  >
    <div class="flex flex-col items-center gap-3">
      <button
        v-if="imageUrl"
        ref="triggerRef"
        type="button"
        class="payment-help-panel__preview-trigger rounded-lg"
        :aria-label="t('payment.previewHelpImage')"
        @click="openPreview"
      >
        <img
          :src="imageUrl"
          alt=""
          class="h-40 max-w-full rounded-lg object-contain transition-opacity hover:opacity-80 motion-reduce:transition-none"
        />
      </button>
      <p v-if="text" class="payment-help-panel__text text-center text-sm">
        {{ text }}
      </p>
    </div>

    <Teleport to="body">
      <Transition name="modal">
        <div
          v-if="previewOpen"
          role="dialog"
          aria-modal="true"
          :aria-label="t('payment.helpImagePreview')"
          class="fixed inset-0 z-[60] flex items-center justify-center bg-black/70 p-4 backdrop-blur-sm"
          tabindex="-1"
          @click.self="closePreview"
          @keydown="handleDialogKeydown"
        >
          <button
            ref="closeButtonRef"
            type="button"
            class="payment-help-panel__close absolute right-4 top-4 flex h-11 w-11 items-center justify-center rounded-full"
            :aria-label="t('common.close')"
            @click="closePreview"
          >
            <Icon name="x" size="md" aria-hidden="true" />
          </button>
          <img
            :src="imageUrl"
            :alt="t('payment.helpImagePreview')"
            class="max-h-[85vh] max-w-[90vw] rounded-xl object-contain shadow-2xl"
          />
        </div>
      </Transition>
    </Teleport>
  </section>
</template>

<script setup lang="ts">
import { nextTick, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

const props = withDefaults(defineProps<{
  text?: string
  imageUrl?: string
}>(), {
  text: '',
  imageUrl: '',
})

const { t } = useI18n()
const previewOpen = ref(false)
const triggerRef = ref<HTMLButtonElement | null>(null)
const closeButtonRef = ref<HTMLButtonElement | null>(null)

async function openPreview() {
  if (!props.imageUrl) return
  previewOpen.value = true
  await nextTick()
  closeButtonRef.value?.focus()
}

async function closePreview() {
  previewOpen.value = false
  await nextTick()
  triggerRef.value?.focus()
}

function handleDialogKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    event.preventDefault()
    event.stopPropagation()
    void closePreview()
    return
  }
  if (event.key === 'Tab') {
    event.preventDefault()
    closeButtonRef.value?.focus()
  }
}
</script>

<style scoped>
.payment-help-panel {
  color: var(--lx-clay-text);
  background: color-mix(in srgb, var(--lx-clay-recessed) 58%, var(--lx-clay-surface));
}

.payment-help-panel__text {
  color: var(--lx-clay-text-muted);
}

.payment-help-panel__preview-trigger,
.payment-help-panel__close {
  transition:
    opacity 150ms ease,
    box-shadow 150ms ease;
}

.payment-help-panel__preview-trigger:focus-visible,
.payment-help-panel__close:focus-visible {
  outline: 3px solid color-mix(in srgb, var(--lx-clay-accent) 42%, transparent);
  outline-offset: 3px;
}

.payment-help-panel__close {
  color: var(--lx-clay-text);
  background: var(--lx-clay-surface);
  box-shadow: var(--lx-clay-shadow-overlay);
}

@media (prefers-reduced-motion: reduce) {
  .payment-help-panel__preview-trigger,
  .payment-help-panel__close {
    transition-duration: 0.01ms;
  }
}
</style>
