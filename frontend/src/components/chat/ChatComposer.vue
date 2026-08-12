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
      ref="composerRef"
      class="chat-composer"
      :class="{
        'chat-composer--expanded': composerExpanded,
        'chat-composer--maximized': composerMaximized,
        'chat-composer--overflowing': inputOverflowing,
        'chat-composer--has-expand-toggle': showExpandToggle,
        'chat-composer--has-attachments': hasAttachments,
      }"
      :data-expanded="composerExpanded ? '' : undefined"
      :data-expanded-composer="composerMaximized ? '' : undefined"
      :aria-busy="submissionBusy || submitting ? 'true' : undefined"
      @submit.prevent="submit"
    >
      <div v-if="hasAttachments" class="chat-composer__attachments">
        <slot name="attachments"></slot>
      </div>

      <div v-if="$slots.leading" class="chat-composer__leading">
        <slot name="leading"></slot>
      </div>

      <div class="chat-composer__input-shell">
        <textarea
          :id="composerInputId"
          ref="textareaRef"
          v-model="draft"
          class="chat-composer__input"
          rows="1"
          :maxlength="maxLength"
          :placeholder="placeholder"
          :disabled="disabled || insufficientBalance"
          :aria-label="t('chat.composer.label')"
          @input="resize"
          @keydown="onKeydown"
        ></textarea>

        <button
          v-if="showExpandToggle"
          type="button"
          class="chat-composer__expand-toggle"
          :aria-label="composerMaximized ? t('chat.composer.collapse') : t('chat.composer.expand')"
          :title="composerMaximized ? t('chat.composer.collapse') : t('chat.composer.expand')"
          :aria-controls="composerInputId"
          :aria-expanded="composerMaximized"
          data-test="chat-composer-expand"
          @click="toggleComposerMaximized"
        >
          <Icon
            :name="composerMaximized ? 'chatComposerCollapse' : 'chatComposerExpand'"
            size="md"
            aria-hidden="true"
          />
        </button>
      </div>

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

      <textarea
        ref="measureRef"
        class="chat-composer__measure"
        :value="draft"
        aria-hidden="true"
        tabindex="-1"
        readonly
      ></textarea>
    </form>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, useId, watch } from 'vue'
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
const composerInputId = useId()
const composerRef = ref<HTMLFormElement | null>(null)
const textareaRef = ref<HTMLTextAreaElement | null>(null)
const measureRef = ref<HTMLTextAreaElement | null>(null)
const draft = ref(props.modelValue)
const multiline = ref(props.modelValue.includes('\n'))
const composerMaximized = ref(false)
const inputOverflowing = ref(false)
const showExpandToggle = ref(false)
const submitting = ref(false)
const COMPOSER_COMPACT_INPUT_HEIGHT = 36
const COMPOSER_STACKED_INPUT_HEIGHT = 48
const COMPOSER_SINGLE_LINE_THRESHOLD = 54
const COMPOSER_EXPAND_TOGGLE_THRESHOLD = 140
const COMPOSER_COLLAPSED_VIEWPORT_RATIO = 0.3
const COMPOSER_MAXIMIZED_VIEWPORT_RATIO = 0.75
const COMPOSER_STACKED_CHROME_HEIGHT = 54
let queuedResizeFrame: number | null = null
let composerResizeObserver: ResizeObserver | null = null
let ensureSelectionVisibleAfterResize = false
const observedWidths = new WeakMap<Element, number>()

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
  multiline.value
  || draft.value.includes('\n')
  || composerMaximized.value
))

watch(composerExpanded, () => {
  void nextTick(resize)
})

watch(() => props.modelValue, (value) => {
  if (value === draft.value) return
  draft.value = value
  if (!value) composerMaximized.value = false
  void nextTick(resize)
})

watch(() => props.hasAttachments, () => {
  void nextTick(resize)
})

watch(draft, (value) => {
  emit('update:modelValue', value)
  if (!value) composerMaximized.value = false
})

function viewportHeight(): number {
  if (typeof window === 'undefined') return 800
  return window.visualViewport?.height || window.innerHeight || 800
}

function collapsedInputMaxHeight(): number {
  return Math.max(80, Math.round(viewportHeight() * COMPOSER_COLLAPSED_VIEWPORT_RATIO))
}

function maximizedInputHeight(): number {
  return Math.max(
    collapsedInputMaxHeight(),
    Math.round(viewportHeight() * COMPOSER_MAXIMIZED_VIEWPORT_RATIO)
      - COMPOSER_STACKED_CHROME_HEIGHT,
  )
}

function narrowComposerLayout(): boolean {
  return typeof window !== 'undefined'
    && window.matchMedia?.('(max-width: 720px)').matches
}

function outerWidth(element: Element | null): number {
  if (!(element instanceof HTMLElement)) return 0
  const style = window.getComputedStyle(element)
  return element.getBoundingClientRect().width
    + (Number.parseFloat(style.marginInlineStart) || 0)
    + (Number.parseFloat(style.marginInlineEnd) || 0)
}

function compactContentHeight(fallbackHeight: number): number {
  const form = composerRef.value
  const mirror = measureRef.value
  if (!form || !mirror || typeof window === 'undefined') return fallbackHeight

  const formStyle = window.getComputedStyle(form)
  const formWidth = form.getBoundingClientRect().width || form.clientWidth
  if (formWidth <= 0) return fallbackHeight

  const innerWidth = formWidth
    - (Number.parseFloat(formStyle.paddingInlineStart) || 0)
    - (Number.parseFloat(formStyle.paddingInlineEnd) || 0)
  const reservedWidth = outerWidth(form.querySelector('.chat-composer__leading'))
    + outerWidth(form.querySelector('.chat-composer__trailing'))
    + outerWidth(
      form.querySelector('.chat-composer__action, .chat-composer__empty-action'),
    )
  const compactWidth = Math.max(150, Math.floor(innerWidth - reservedWidth))

  mirror.value = draft.value
  mirror.style.width = `${compactWidth}px`
  mirror.style.paddingInlineStart = '7px'
  mirror.style.paddingInlineEnd = '6px'
  return mirror.scrollHeight || fallbackHeight
}

function restoreScrollPosition(
  textarea: HTMLTextAreaElement,
  previousScrollTop: number,
  overflowing: boolean,
  ensureSelectionVisible = false,
) {
  if (!overflowing) {
    textarea.scrollTop = 0
    return
  }

  const maxScrollTop = Math.max(0, textarea.scrollHeight - textarea.clientHeight)
  const collapsedCaretAtEnd = document.activeElement === textarea
    && textarea.selectionStart === textarea.selectionEnd
    && textarea.selectionEnd === textarea.value.length

  if (collapsedCaretAtEnd) {
    textarea.scrollTop = maxScrollTop
    return
  }

  textarea.scrollTop = Math.min(previousScrollTop, maxScrollTop)
  if (ensureSelectionVisible) ensureActiveSelectionVisible(textarea)
}

function ensureActiveSelectionVisible(textarea: HTMLTextAreaElement) {
  const mirror = measureRef.value
  if (!mirror || typeof window === 'undefined') return

  const style = window.getComputedStyle(textarea)
  const activeSelectionEdge = textarea.selectionDirection === 'backward'
    ? textarea.selectionStart
    : textarea.selectionEnd
  const lineHeight = Number.parseFloat(style.lineHeight) || 26
  const paddingBlockEnd = Number.parseFloat(style.paddingBlockEnd) || 0

  mirror.style.width = `${textarea.clientWidth}px`
  mirror.style.paddingInlineStart = style.paddingInlineStart
  mirror.style.paddingInlineEnd = style.paddingInlineEnd
  mirror.value = `${textarea.value.slice(0, activeSelectionEdge)}\u200b`

  const activeEdgeBottom = Math.max(lineHeight, mirror.scrollHeight - paddingBlockEnd)
  const activeEdgeTop = Math.max(0, activeEdgeBottom - lineHeight)
  const visibleTop = textarea.scrollTop
  const visibleBottom = visibleTop + textarea.clientHeight

  if (activeEdgeBottom > visibleBottom) {
    textarea.scrollTop = activeEdgeBottom - textarea.clientHeight
  } else if (activeEdgeTop < visibleTop) {
    textarea.scrollTop = activeEdgeTop
  }
}

function resize() {
  const textarea = textareaRef.value
  if (!textarea) return

  const previousHeight = textarea.getBoundingClientRect().height
  const previousScrollTop = textarea.scrollTop
  textarea.style.height = 'auto'
  const contentHeight = textarea.scrollHeight
  const shouldUseStackedLayout = draft.value.includes('\n')
    || (
      narrowComposerLayout()
        ? contentHeight > COMPOSER_SINGLE_LINE_THRESHOLD
        : compactContentHeight(contentHeight) > COMPOSER_SINGLE_LINE_THRESHOLD
    )

  if (multiline.value !== shouldUseStackedLayout) {
    multiline.value = shouldUseStackedLayout
    textarea.style.height = previousHeight > 0
      ? `${previousHeight}px`
      : `${COMPOSER_COMPACT_INPUT_HEIGHT}px`
    textarea.style.overflowY = 'hidden'
    return
  }

  const shouldShowExpandToggle = Boolean(draft.value)
    && (
      composerMaximized.value
      || contentHeight >= Math.min(
        COMPOSER_EXPAND_TOGGLE_THRESHOLD,
        collapsedInputMaxHeight(),
      )
    )

  if (showExpandToggle.value !== shouldShowExpandToggle) {
    showExpandToggle.value = shouldShowExpandToggle
    textarea.style.height = previousHeight > 0
      ? `${previousHeight}px`
      : `${COMPOSER_COMPACT_INPUT_HEIGHT}px`
    void nextTick(resize)
    return
  }

  const minimumHeight = composerExpanded.value
    ? COMPOSER_STACKED_INPUT_HEIGHT
    : COMPOSER_COMPACT_INPUT_HEIGHT
  const maximumHeight = composerMaximized.value
    ? maximizedInputHeight()
    : collapsedInputMaxHeight()
  const nextHeight = composerMaximized.value
    ? maximumHeight
    : Math.min(Math.max(contentHeight, minimumHeight), maximumHeight)
  const overflowing = contentHeight > nextHeight + 0.5
  inputOverflowing.value = overflowing
  textarea.style.overflowY = overflowing ? 'auto' : 'hidden'
  textarea.style.height = `${nextHeight}px`
  restoreScrollPosition(
    textarea,
    previousScrollTop,
    overflowing,
    ensureSelectionVisibleAfterResize,
  )
  ensureSelectionVisibleAfterResize = false
}

function queueResize() {
  if (queuedResizeFrame !== null) cancelAnimationFrame(queuedResizeFrame)
  if (typeof requestAnimationFrame !== 'function') {
    void nextTick(resize)
    return
  }
  queuedResizeFrame = requestAnimationFrame(() => {
    queuedResizeFrame = null
    resize()
  })
}

function toggleComposerMaximized() {
  const textarea = textareaRef.value
  if (!textarea) return

  const selectionStart = textarea.selectionStart
  const selectionEnd = textarea.selectionEnd
  const selectionDirection = textarea.selectionDirection
  const previousScrollTop = textarea.scrollTop
  ensureSelectionVisibleAfterResize = composerMaximized.value
  composerMaximized.value = !composerMaximized.value

  void nextTick(() => {
    textarea.focus({ preventScroll: true })
    textarea.setSelectionRange(selectionStart, selectionEnd, selectionDirection)
    textarea.scrollTop = previousScrollTop
    resize()
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

onMounted(() => {
  resize()

  if (typeof ResizeObserver !== 'undefined' && composerRef.value) {
    composerResizeObserver = new ResizeObserver((entries) => {
      const widthChanged = entries.some((entry) => {
        const previousWidth = observedWidths.get(entry.target)
        observedWidths.set(entry.target, entry.contentRect.width)
        return previousWidth !== undefined
          && Math.abs(previousWidth - entry.contentRect.width) >= 0.5
      })
      if (widthChanged) queueResize()
    })

    const responsiveElements = [
      composerRef.value.parentElement,
      composerRef.value,
      composerRef.value.querySelector('.chat-composer__leading'),
      composerRef.value.querySelector('.chat-composer__trailing'),
    ].filter((element): element is Element => element instanceof Element)

    responsiveElements.forEach((element) => composerResizeObserver?.observe(element))
  }

  window.addEventListener('resize', queueResize)
  window.visualViewport?.addEventListener('resize', queueResize)
})

onBeforeUnmount(() => {
  if (queuedResizeFrame !== null) cancelAnimationFrame(queuedResizeFrame)
  composerResizeObserver?.disconnect()
  window.removeEventListener('resize', queueResize)
  window.visualViewport?.removeEventListener('resize', queueResize)
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

  position: relative;
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
    "composer-leading composer-input composer-trailing composer-action";
  grid-template-rows: auto 36px;
  row-gap: 18px;
}

.chat-composer--expanded.chat-composer--has-attachments {
  grid-template-areas:
    "composer-attachments composer-attachments composer-attachments composer-attachments"
    "composer-input composer-input composer-input composer-input"
    "composer-leading . composer-trailing composer-action";
  grid-template-rows: auto minmax(36px, auto) 36px;
  row-gap: 2px;
}

.chat-composer__attachments {
  grid-area: composer-attachments;
  min-width: 0;
  padding: 0;
}

.chat-composer--expanded.chat-composer--has-attachments .chat-composer__attachments {
  padding-block-end: 16px;
}

.chat-composer__leading {
  grid-area: composer-leading;
  display: flex;
  min-width: 36px;
  align-items: center;
}

.chat-composer__input-shell {
  grid-area: composer-input;
  position: relative;
  min-width: 0;
}

.chat-composer__input {
  display: block;
  box-sizing: border-box;
  width: 100%;
  min-width: 0;
  min-height: 36px;
  max-height: none;
  resize: none;
  border: 0;
  padding-block: 5px;
  padding-inline: 7px 6px;
  overflow-y: hidden;
  color: var(--chat-composer-primary-fg);
  background: transparent;
  font-size: 16px;
  font-weight: 400;
  line-height: 26px;
  letter-spacing: normal;
  outline: none;
  scrollbar-width: thin;
  scrollbar-color: #e5e5e5 transparent;
}

.chat-composer--expanded .chat-composer__input {
  padding: 5px 10px;
}

.chat-composer--has-expand-toggle .chat-composer__input {
  padding-inline-end: 46px;
}

.chat-composer__input::-webkit-scrollbar {
  width: 6px;
}

.chat-composer__input::-webkit-scrollbar-track {
  background: transparent;
}

.chat-composer__input::-webkit-scrollbar-thumb {
  border-radius: 999px;
  background: #e5e5e5;
}

.chat-composer__input::placeholder {
  color: var(--chat-composer-placeholder-fg);
  font-size: inherit;
  font-weight: inherit;
  line-height: inherit;
  letter-spacing: inherit;
  opacity: 1;
}

.chat-composer__input:disabled {
  cursor: not-allowed;
}

.chat-composer__expand-toggle {
  position: absolute;
  inset-block-start: 2px;
  inset-inline-end: 12px;
  z-index: 2;
  display: grid;
  width: 36px;
  height: 36px;
  place-items: center;
  border: 0;
  border-radius: 50%;
  padding: 0;
  color: var(--chat-composer-muted-fg);
  background: transparent;
  cursor: pointer;
  transition:
    color 150ms ease,
    background-color 150ms ease;
}

.chat-composer__expand-toggle:hover {
  color: var(--chat-composer-primary-fg);
  background: var(--chat-composer-secondary-hover);
}

.chat-composer__expand-toggle:focus-visible {
  outline: 2px solid var(--lx-clay-accent);
  outline-offset: 1px;
}

.chat-composer__measure {
  position: absolute;
  inset: 0 auto auto 0;
  z-index: -1;
  box-sizing: border-box;
  max-width: 100%;
  height: 0;
  min-height: 0;
  resize: none;
  border: 0;
  padding-block: 5px;
  padding-inline: 7px 6px;
  overflow: hidden;
  visibility: hidden;
  pointer-events: none;
  white-space: pre-wrap;
  overflow-wrap: break-word;
  color: transparent;
  background: transparent;
  font: inherit;
  font-size: 16px;
  font-weight: 400;
  line-height: 26px;
  letter-spacing: normal;
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
    row-gap: 0;
  }

  .chat-composer--has-attachments .chat-composer__attachments,
  .chat-composer--expanded.chat-composer--has-attachments .chat-composer__attachments {
    padding-block-end: 12px;
  }

  .chat-composer__input {
    padding: 5px 10px;
    font-size: 16px;
  }

  .chat-composer--has-expand-toggle .chat-composer__input {
    padding-inline-end: 46px;
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

:global(html.dark .chat-composer__input) {
  scrollbar-color: #4d4d4d transparent;
}

:global(html.dark .chat-composer__input::-webkit-scrollbar-thumb) {
  background: #4d4d4d;
}

@media (forced-colors: active) {
  .chat-composer {
    border: 1px solid CanvasText;
  }
}

@media (prefers-reduced-motion: reduce) {
  .chat-composer,
  .chat-composer__expand-toggle,
  .chat-composer__action {
    transition: none;
  }
}
</style>
