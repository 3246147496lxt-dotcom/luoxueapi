<template>
  <div class="chat-composer-wrap">
    <div v-if="insufficientBalance" class="chat-composer__balance" role="status">
      <span>
        <Icon name="exclamationCircle" size="sm" />
        {{ t('chat.balance.insufficient') }}
      </span>
      <router-link to="/purchase">{{ t('chat.balance.recharge') }}</router-link>
    </div>

    <form
      class="chat-composer"
      :class="{
        'chat-composer--expanded': composerExpanded,
        'chat-composer--has-attachments': hasAttachments,
      }"
      :aria-busy="submissionBusy || submitting ? 'true' : undefined"
      @submit.prevent="submit"
    >
      <div v-if="hasAttachments" class="chat-composer__attachments">
        <slot name="attachments"></slot>
      </div>

      <div v-if="$slots.leading" class="chat-composer__leading">
        <slot name="leading"></slot>
      </div>

      <textarea
        ref="textareaRef"
        v-model="draft"
        rows="1"
        :maxlength="maxLength"
        :placeholder="placeholder"
        :disabled="disabled || insufficientBalance"
        :aria-label="t('chat.composer.label')"
        @input="resize"
        @keydown="onKeydown"
      ></textarea>

      <div v-if="$slots.trailing || $slots.controls" class="chat-composer__trailing">
        <slot name="trailing">
          <slot name="controls"></slot>
        </slot>
      </div>

      <button
        v-if="streaming"
        type="button"
        class="chat-composer__action chat-composer__action--stop"
        :aria-label="t('chat.actions.stop')"
        :title="t('chat.actions.stop')"
        @click="$emit('stop')"
      >
        <span aria-hidden="true"></span>
      </button>
      <div
        v-else-if="!hasContent && $slots['empty-action']"
        class="chat-composer__empty-action"
      >
        <slot name="empty-action"></slot>
      </div>
      <button
        v-else
        type="submit"
        class="chat-composer__action chat-composer__action--send"
        :disabled="!canSubmit"
        :aria-label="t('chat.actions.send')"
        :title="t('chat.actions.send')"
      >
        <Icon name="chatSend" size="md" aria-hidden="true" />
      </button>
    </form>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

const props = withDefaults(defineProps<{
  modelValue?: string
  streaming?: boolean
  disabled?: boolean
  insufficientBalance?: boolean
  submissionBusy?: boolean
  hasAttachments?: boolean
  attachmentsValid?: boolean
  maxLength?: number
}>(), {
  modelValue: '',
  streaming: false,
  disabled: false,
  insufficientBalance: false,
  submissionBusy: false,
  hasAttachments: false,
  attachmentsValid: true,
  maxLength: 20_000,
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
  send: [value: string, acknowledge: (accepted: boolean) => void]
  stop: []
}>()

const { t } = useI18n()
const textareaRef = ref<HTMLTextAreaElement | null>(null)
const draft = ref(props.modelValue)
const multiline = ref(props.modelValue.includes('\n'))
const submitting = ref(false)
const COMPOSER_RESIZE_DURATION_MS = 180
let resizeFrame: number | null = null
let collapseTimer: ReturnType<typeof setTimeout> | null = null

const hasContent = computed(() => draft.value.trim().length > 0 || props.hasAttachments)

const canSubmit = computed(() => (
  hasContent.value
  && props.attachmentsValid
  && !props.disabled
  && !props.insufficientBalance
  && !props.streaming
  && !props.submissionBusy
  && !submitting.value
))

const placeholder = computed(() => (
  props.insufficientBalance
    ? t('chat.composer.rechargePlaceholder')
    : t('chat.composer.placeholder')
))

const composerExpanded = computed(() => (
  props.hasAttachments
  || multiline.value
  || draft.value.includes('\n')
))

watch(() => props.modelValue, (value) => {
  if (value === draft.value) return
  draft.value = value
  void nextTick(resize)
})

watch(draft, (value) => emit('update:modelValue', value))

function resize() {
  const textarea = textareaRef.value
  if (!textarea) return

  if (collapseTimer !== null) {
    clearTimeout(collapseTimer)
    collapseTimer = null
  }

  const previousHeight = textarea.getBoundingClientRect().height
  textarea.style.height = 'auto'
  const contentHeight = textarea.scrollHeight
  const nextHeight = Math.min(Math.max(contentHeight, 36), 180)
  const shouldExpand = contentHeight > 54
  const collapseAfterResize = multiline.value && !shouldExpand

  if (resizeFrame !== null) {
    cancelAnimationFrame(resizeFrame)
    resizeFrame = null
  }

  const reduceMotion = typeof window !== 'undefined'
    && window.matchMedia?.('(prefers-reduced-motion: reduce)').matches
  const canAnimate = !reduceMotion
    && previousHeight > 0
    && Math.abs(previousHeight - nextHeight) >= 0.5
    && typeof requestAnimationFrame === 'function'

  if (shouldExpand) multiline.value = true
  else if (!canAnimate) multiline.value = false

  if (!canAnimate) {
    textarea.style.height = `${nextHeight}px`
    return
  }

  textarea.style.height = `${previousHeight}px`
  void textarea.offsetHeight
  resizeFrame = requestAnimationFrame(() => {
    textarea.style.height = `${nextHeight}px`
    resizeFrame = null
    if (collapseAfterResize) {
      collapseTimer = setTimeout(() => {
        multiline.value = false
        collapseTimer = null
      }, COMPOSER_RESIZE_DURATION_MS)
    }
  })
}

function onKeydown(event: KeyboardEvent) {
  if (event.key !== 'Enter' || event.shiftKey || event.isComposing) return
  event.preventDefault()
  submit()
}

function submit() {
  const value = draft.value.trim()
  if (!canSubmit.value) return
  submitting.value = true
  let acknowledged = false
  emit('send', value, (accepted) => {
    if (acknowledged) return
    acknowledged = true
    submitting.value = false
    if (!accepted) return
    draft.value = ''
    void nextTick(() => {
      resize()
      textareaRef.value?.focus()
    })
  })
}

function focus() {
  textareaRef.value?.focus()
}

function insertText(value: string): boolean {
  const text = value.trim()
  if (!text) return false

  const textarea = textareaRef.value
  const start = textarea?.selectionStart ?? draft.value.length
  const end = textarea?.selectionEnd ?? start
  const retainedLength = draft.value.length - Math.max(0, end - start)
  const availableLength = Math.max(0, props.maxLength - retainedLength)
  if (availableLength === 0) return false

  const insertingAtEnd = start === end && start === draft.value.length
  const separator = insertingAtEnd && draft.value.trim().length > 0 ? '\n' : ''
  const requestedInsertion = `${separator}${text}`
  if (requestedInsertion.length > availableLength) return false
  const insertion = requestedInsertion

  draft.value = `${draft.value.slice(0, start)}${insertion}${draft.value.slice(end)}`
  const nextCursor = start + insertion.length
  void nextTick(() => {
    resize()
    textareaRef.value?.focus()
    textareaRef.value?.setSelectionRange(nextCursor, nextCursor)
  })
  return true
}

onMounted(resize)

onBeforeUnmount(() => {
  if (resizeFrame !== null) cancelAnimationFrame(resizeFrame)
  if (collapseTimer !== null) clearTimeout(collapseTimer)
})

defineExpose({ focus, insertText })
</script>

<style scoped>
.chat-composer-wrap {
  width: min(100%, 768px);
  margin: 0 auto;
}

.chat-composer__balance {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  margin-bottom: 8px;
  border: 1px solid color-mix(in srgb, var(--lx-clay-warning) 32%, transparent);
  border-radius: 8px;
  padding: 9px 12px;
  color: var(--lx-clay-text);
  background: var(--lx-clay-warning-soft);
  font-size: var(--workspace-type-body-size);
  font-weight: var(--workspace-type-body-weight);
}

.chat-composer__balance span {
  display: flex;
  align-items: center;
  gap: 7px;
}

.chat-composer__balance a {
  flex: none;
  color: var(--lx-clay-accent-deep);
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
  text-decoration: none;
}

.chat-composer {
  --chat-composer-surface: #fff;
  --chat-composer-primary-fg: #0d0d0d;
  --chat-composer-muted-fg: #5d5d5d;
  --chat-composer-placeholder-fg: #8f8f8f;
  --chat-composer-tertiary-fg: #8f8f8f;
  --chat-composer-shadow:
    0 0 0 1px rgb(0 0 0 / 4%),
    0 2px 8px rgb(0 0 0 / 4%),
    0 4px 80px 8px rgb(0 0 0 / 2.4%);
  --chat-composer-action-bg: #000;
  --chat-composer-action-fg: #fff;
  --chat-composer-action-hover: #2f2f2f;
  --chat-composer-secondary-fg: #0d0d0d;
  --chat-composer-secondary-hover: rgb(0 0 0 / 5%);

  display: grid;
  grid-template-columns: auto minmax(150px, 1fr) auto auto;
  grid-template-areas: "composer-leading composer-input composer-trailing composer-action";
  align-items: center;
  column-gap: 0;
  min-height: 52px;
  border: 0;
  border-radius: 28px;
  padding: 8px;
  color: var(--chat-composer-primary-fg);
  background: var(--chat-composer-surface);
  box-shadow: var(--chat-composer-shadow);
  transition: box-shadow 150ms ease;
}

.chat-composer:focus-within {
  box-shadow: var(--chat-composer-shadow);
}

.chat-composer--expanded {
  grid-template-columns: auto minmax(0, 1fr) auto auto;
  grid-template-areas:
    "composer-input composer-input composer-input composer-input"
    "composer-leading . composer-trailing composer-action";
  grid-template-rows: minmax(36px, auto) 36px;
  align-items: center;
  row-gap: 2px;
  border-radius: 28px;
  padding: 8px;
}

.chat-composer--has-attachments {
  grid-template-areas:
    "composer-attachments composer-attachments composer-attachments composer-attachments"
    "composer-input composer-input composer-input composer-input"
    "composer-leading . composer-trailing composer-action";
  grid-template-rows: auto minmax(36px, auto) 36px;
}

.chat-composer__attachments {
  grid-area: composer-attachments;
  min-width: 0;
  padding: 0 4px 4px;
}

.chat-composer__leading {
  grid-area: composer-leading;
  display: flex;
  min-width: 36px;
  align-items: center;
}

.chat-composer textarea {
  grid-area: composer-input;
  width: 100%;
  min-width: 0;
  min-height: 36px;
  max-height: 180px;
  resize: none;
  border: 0;
  padding-block: 5px;
  padding-inline: 7px 6px;
  overflow-y: auto;
  color: var(--chat-composer-primary-fg);
  background: transparent;
  font-size: 16px;
  font-weight: 400;
  line-height: 26px;
  letter-spacing: normal;
  outline: none;
  scrollbar-width: thin;
  transition: height 180ms cubic-bezier(0.22, 1, 0.36, 1);
}

.chat-composer--expanded textarea {
  padding: 5px 10px;
}

.chat-composer textarea::placeholder {
  color: var(--chat-composer-placeholder-fg);
  font-size: inherit;
  font-weight: inherit;
  line-height: inherit;
  letter-spacing: inherit;
  opacity: 1;
}

.chat-composer textarea:disabled {
  cursor: not-allowed;
}

.chat-composer__trailing {
  grid-area: composer-trailing;
  display: flex;
  width: auto;
  min-width: 0;
  align-items: center;
  justify-content: flex-end;
  gap: 4px;
}

.chat-composer__empty-action {
  grid-area: composer-action;
  display: grid;
  width: 36px;
  height: 36px;
  margin-inline-start: 8px;
  place-items: center;
}

.chat-composer__action {
  grid-area: composer-action;
  display: grid;
  place-items: center;
  width: 36px;
  height: 36px;
  margin-inline-start: 8px;
  border: 0;
  border-radius: 50%;
  color: var(--chat-composer-action-fg);
  background: var(--chat-composer-action-bg);
  box-shadow: none;
  transition:
    transform 150ms ease,
    color 150ms ease,
    background-color 150ms ease,
    box-shadow 150ms ease;
}

.chat-composer__action:hover:not(:disabled) {
  color: var(--chat-composer-action-fg);
  background: var(--chat-composer-action-hover);
  transform: none;
  box-shadow: none;
}

.chat-composer__action:active:not(:disabled) {
  transform: scale(0.96);
}

.chat-composer__action:focus-visible {
  outline: 3px solid var(--lx-clay-accent-soft);
  outline-offset: 1px;
}

.chat-composer__action:disabled {
  cursor: not-allowed;
  color: var(--lx-clay-text-subtle);
  background: var(--lx-clay-recessed-strong);
  box-shadow: none;
}

.chat-composer__action--stop span {
  width: 12px;
  height: 12px;
  border-radius: 2px;
  background: currentColor;
}

.chat-composer__leading,
.chat-composer__trailing,
.chat-composer__empty-action,
.chat-composer__action {
  align-self: center;
}

@media (max-width: 720px) {
  .chat-composer__balance {
    align-items: flex-start;
  }

  .chat-composer {
    grid-template-columns: auto minmax(0, 1fr) auto auto;
    grid-template-areas:
      "composer-input composer-input composer-input composer-input"
      "composer-leading . composer-trailing composer-action";
    grid-template-rows: minmax(36px, auto) 36px;
    align-items: center;
    min-height: 84px;
    row-gap: 0;
    border-radius: 28px;
    padding: 6px 8px;
  }

  .chat-composer--has-attachments {
    grid-template-areas:
      "composer-attachments composer-attachments composer-attachments composer-attachments"
      "composer-input composer-input composer-input composer-input"
      "composer-leading . composer-trailing composer-action";
    grid-template-rows: auto minmax(36px, auto) 36px;
  }

  .chat-composer textarea {
    padding: 5px 10px;
    font-size: 16px;
  }

  .chat-composer--expanded {
    padding: 6px 8px;
  }
}

:global(html.dark .chat-composer) {
  --chat-composer-surface: rgb(33 33 33 / 90%);
  --chat-composer-primary-fg: #fff;
  --chat-composer-muted-fg: #afafaf;
  --chat-composer-placeholder-fg: #afafaf;
  --chat-composer-tertiary-fg: #afafaf;
  --chat-composer-shadow: inset 0 0 1px rgb(255 255 255 / 20%);
  --chat-composer-action-bg: #fff;
  --chat-composer-action-fg: #000;
  --chat-composer-action-hover: #e8e8e8;
  --chat-composer-secondary-fg: #fff;
  --chat-composer-secondary-hover: rgb(255 255 255 / 10%);
}

@media (forced-colors: active) {
  .chat-composer {
    border: 1px solid CanvasText;
  }
}

@media (prefers-reduced-motion: reduce) {
  .chat-composer,
  .chat-composer textarea,
  .chat-composer__action {
    transition: none;
  }
}
</style>
