<template>
  <div
    v-if="hasVisibleSummary"
    class="chat-activity-layer"
    :class="{ 'chat-activity-layer--mobile': mobile }"
    data-test="chat-activity-layer"
  >
    <button
      v-if="mobile"
      type="button"
      class="chat-activity-layer__scrim"
      :aria-label="t('chat.activity.close')"
      tabindex="-1"
      @click="requestClose"
    ></button>

    <aside
      ref="panelRef"
      class="chat-activity-panel"
      :class="{ 'chat-activity-panel--mobile': mobile }"
      :role="mobile ? 'dialog' : 'complementary'"
      :aria-modal="mobile ? 'true' : undefined"
      :aria-labelledby="titleId"
      :tabindex="mobile ? -1 : undefined"
      data-test="chat-activity-panel"
      @keydown="onPanelKeydown"
    >
      <header class="chat-activity-panel__header">
        <div class="chat-activity-panel__heading">
          <h2 :id="titleId">{{ t('chat.activity.title') }}</h2>
          <span v-if="elapsedLabel" class="chat-activity-panel__elapsed">
            {{ elapsedLabel }}
          </span>
        </div>
        <button
          ref="closeButtonRef"
          type="button"
          class="chat-activity-panel__close"
          :aria-label="t('chat.activity.close')"
          :title="t('chat.activity.close')"
          @click="requestClose"
        >
          <Icon name="x" size="sm" aria-hidden="true" />
        </button>
      </header>

      <div
        v-if="isActive"
        class="chat-activity-panel__progress"
        role="progressbar"
        :aria-label="statusLabel"
      >
        <span />
      </div>

      <div class="chat-activity-panel__status-row">
        <Icon
          :name="isPro ? 'brain' : 'activity'"
          size="sm"
          class="chat-activity-panel__status-icon"
          aria-hidden="true"
        />
        <span>{{ statusLabel }}</span>
      </div>

      <div
        ref="scrollerRef"
        class="chat-activity-panel__body"
        data-test="chat-activity-scroll-region"
        @scroll.passive="onScroll"
      >
        <section
          v-for="entry in renderedParts"
          :key="entry.key"
          class="chat-activity-panel__part"
          :data-status="entry.status"
          data-test="chat-activity-part"
        >
          <div
            v-if="entry.streaming"
            class="chat-activity-panel__markdown chat-activity-panel__markdown--streaming"
          >
            <span v-if="entry.baseText" v-text="entry.baseText"></span>
            <span
              v-for="(segment, segmentIndex) in entry.chunks"
              :key="`${entry.key}:${segmentIndex}`"
              v-text="segment"
            ></span>
          </div>
          <div
            v-else
            class="chat-activity-panel__markdown"
            v-html="entry.html"
          ></div>
        </section>
      </div>

      <p
        class="sr-only"
        aria-live="polite"
        aria-atomic="true"
        data-test="chat-activity-live-region"
      >
        {{ liveAnnouncement }}
      </p>
    </aside>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import DOMPurify from 'dompurify'
import { marked } from 'marked'
import Icon from '@/components/icons/Icon.vue'
import type { ChatActivity, ChatActivityStatus } from '@/types/chat'
import { chatActivityPartHasText } from '@/features/chat/activity'
import { acquireBodyScrollLock, releaseBodyScrollLock } from '@/utils/bodyScrollLock'
import {
  getVisibleFocusableElements,
  isTopModalLayer,
  registerModalLayer,
  unregisterModalLayer,
} from '@/utils/modalStack'

const props = withDefaults(defineProps<{
  activities: ChatActivity[]
  mobile?: boolean
}>(), {
  mobile: false,
})

const emit = defineEmits<{
  close: []
}>()

const { t } = useI18n()
const titleId = `chat-activity-title-${Math.random().toString(36).slice(2, 9)}`
const panelRef = ref<HTMLElement | null>(null)
const closeButtonRef = ref<HTMLButtonElement | null>(null)
const scrollerRef = ref<HTMLElement | null>(null)
const now = ref(Date.now())
const followLatest = ref(true)
const liveAnnouncement = ref('')
let elapsedTimer: ReturnType<typeof setInterval> | null = null
let liveTimer: ReturnType<typeof setTimeout> | null = null
let lastAnnouncedPartKey = ''
let lastAnnouncedRevision = ''
let lastAnnouncedStatus = ''
const modalLayerToken = Symbol('chat-activity-panel')
const scrollLockToken = Symbol('chat-activity-panel-scroll-lock')
let modalActive = false

interface RenderedSummaryPart {
  key: string
  status: ChatActivityStatus
  streaming: boolean
  baseText: string
  chunks: string[]
  html: string
}

interface RenderCacheEntry {
  text: string
  html: string
}

const renderedSummaryCache = new Map<string, RenderCacheEntry>()

const sortedActivities = computed(() => [...props.activities].sort((left, right) => {
  const leftTime = left.startedAt ?? 0
  const rightTime = right.startedAt ?? 0
  if (leftTime !== rightTime) return leftTime - rightTime
  return left.key.localeCompare(right.key)
}))

const latestActivity = computed(() => sortedActivities.value.at(-1) ?? null)
const visibleParts = computed(() => sortedActivities.value.flatMap((activity) => (
  [...activity.items]
    .sort((left, right) => left.outputIndex - right.outputIndex || left.key.localeCompare(right.key))
    .flatMap((item) => [...item.parts].sort((left, right) => (
      left.summaryIndex - right.summaryIndex || left.key.localeCompare(right.key)
    )))
)).filter(chatActivityPartHasText))
const hasVisibleSummary = computed(() => visibleParts.value.length > 0)
const status = computed<ChatActivityStatus>(() => latestActivity.value?.status ?? 'pending')
const isActive = computed(() => status.value === 'pending' || status.value === 'streaming')
const isPro = computed(() => latestActivity.value?.reasoningMode === 'pro')
const statusLabel = computed(() => {
  if (isActive.value) {
    return t(isPro.value ? 'chat.activity.proThinking' : 'chat.activity.thinking')
  }
  const statusKey: Record<ChatActivityStatus, string> = {
    pending: 'chat.activity.pending',
    streaming: 'chat.activity.thinking',
    completed: 'chat.activity.completed',
    incomplete: 'chat.activity.incomplete',
    failed: 'chat.activity.failed',
    stopped: 'chat.activity.stopped',
    disconnected: 'chat.activity.disconnected',
  }
  return t(statusKey[status.value])
})

const elapsedLabel = computed(() => {
  const startedAt = latestActivity.value?.startedAt
  if (!startedAt) return ''
  const end = latestActivity.value?.completedAt ?? now.value
  const seconds = Math.max(0, Math.floor((end - startedAt) / 1000))
  const minutes = Math.floor(seconds / 60)
  const remainder = seconds % 60
  return minutes > 0 ? `${minutes}m ${remainder}s` : `${remainder}s`
})

const renderedParts = computed<RenderedSummaryPart[]>(() => {
  const visibleKeys = new Set<string>()
  const result = visibleParts.value.map((part) => {
    visibleKeys.add(part.key)
    const streaming = part.status === 'pending'
      || part.status === 'streaming'
      || (part.streamingTextChunks?.length ?? 0) > 0
    if (streaming) {
      renderedSummaryCache.delete(part.key)
      return {
        key: part.key,
        status: part.status,
        streaming: true,
        baseText: part.text,
        chunks: part.streamingTextChunks ?? [],
        html: '',
      }
    }
    const cached = renderedSummaryCache.get(part.key)
    let html: string
    if (cached?.text === part.text) {
      html = cached.html
    } else {
      html = renderSummary(part.text)
      renderedSummaryCache.set(part.key, {
        text: part.text,
        html,
      })
    }
    return {
      key: part.key,
      status: part.status,
      streaming: false,
      baseText: '',
      chunks: [],
      html,
    }
  })
  for (const key of renderedSummaryCache.keys()) {
    if (!visibleKeys.has(key)) renderedSummaryCache.delete(key)
  }
  return result
})

// Track only stable metadata. Building a signature from every full summary
// copied all accumulated text on each stream notification.
const summaryRevision = computed(() => visibleParts.value.map((part) => (
  `${part.key}:${part.lastSequenceNumber ?? ''}:${part.updatedAt}:${part.status}:`
  + `${part.text.length}:${part.streamingTextChunks?.length ?? 0}:`
  + `${part.streamingTextChunks?.at(-1)?.length ?? 0}`
)).join('|'))

watch(isActive, (active) => {
  updateElapsedTimer(active)
}, { immediate: true })

watch(summaryRevision, async () => {
  // Populate/correct the sanitized cache before the DOM scroll measurement.
  void renderedParts.value
  scheduleLiveAnnouncement()
  if (!followLatest.value) return
  await nextTick()
  scrollToLatest()
})

watch(statusLabel, () => scheduleLiveAnnouncement(), { immediate: true })

watch([() => props.mobile, hasVisibleSummary], ([mobile, visible]) => {
  if (mobile && visible) {
    void activateMobileModal()
    return
  }
  deactivateMobileModal()
}, { immediate: true, flush: 'post' })

onMounted(() => {
  scrollToLatest()
})

onBeforeUnmount(() => {
  deactivateMobileModal()
  updateElapsedTimer(false)
  if (liveTimer !== null) clearTimeout(liveTimer)
  renderedSummaryCache.clear()
})

function renderSummary(value: string): string {
  const rendered = marked.parse(value, { breaks: true, gfm: true })
  return DOMPurify.sanitize(String(rendered), {
    ALLOWED_TAGS: ['p', 'ul', 'ol', 'li', 'strong', 'em', 'code', 'pre', 'blockquote', 'br'],
    ALLOWED_ATTR: [],
    ALLOW_DATA_ATTR: false,
    ALLOW_ARIA_ATTR: false,
  })
}

function updateElapsedTimer(active: boolean) {
  if (elapsedTimer !== null) {
    clearInterval(elapsedTimer)
    elapsedTimer = null
  }
  now.value = Date.now()
  if (!active) return
  elapsedTimer = setInterval(() => {
    now.value = Date.now()
  }, 1000)
}

function scheduleLiveAnnouncement() {
  if (liveTimer !== null) return
  liveTimer = setTimeout(() => {
    liveTimer = null
    const tail = visibleParts.value.at(-1)
    const currentStatus = statusLabel.value
    let announcement = ''
    if (tail) {
      const revision = `${tail.lastSequenceNumber ?? ''}:${tail.updatedAt}:`
        + `${tail.status}:${tail.text.length}:${tail.streamingTextChunks?.length ?? 0}:`
        + `${tail.streamingTextChunks?.at(-1)?.length ?? 0}`
      const spokenText = revision === lastAnnouncedRevision && tail.key === lastAnnouncedPartKey
        ? ''
        : summaryAnnouncementText(tail.text, tail.streamingTextChunks ?? [], 320)
      lastAnnouncedPartKey = tail.key
      lastAnnouncedRevision = revision
      if (spokenText) announcement = `${currentStatus}. ${spokenText}`
    }
    if (!announcement && currentStatus !== lastAnnouncedStatus) announcement = currentStatus
    lastAnnouncedStatus = currentStatus
    if (announcement) liveAnnouncement.value = announcement
  }, 800)
}

function summaryTailText(baseText: string, chunks: string[], limit: number): string {
  const selected: string[] = []
  let remaining = limit
  for (let index = chunks.length - 1; index >= 0 && remaining > 0; index -= 1) {
    const chunk = chunks[index] ?? ''
    if (!chunk) continue
    const suffix = chunk.slice(-remaining)
    selected.push(suffix)
    remaining -= suffix.length
  }
  if (remaining > 0 && baseText) selected.push(baseText.slice(-remaining))
  return selected.reverse().join('')
}

function summaryAnnouncementText(baseText: string, chunks: string[], limit: number): string {
  // Work only on the bounded tail. The visual stream stays as safe text and
  // never invokes the Markdown parser until an authoritative terminal part.
  return summaryTailText(baseText, chunks, limit)
    .replace(/!\[([^\]]*)\]\([^)]*\)/g, '$1')
    .replace(/\[([^\]]+)\]\([^)]*\)/g, '$1')
    .replace(/^[\t ]{0,3}(?:#{1,6}|>|[-+] |\d+[.)] )/gm, '')
    .replace(/[*_~`]/g, '')
    .replace(/\s+/g, ' ')
    .trim()
}

function onScroll() {
  const scroller = scrollerRef.value
  if (!scroller) return
  followLatest.value = scroller.scrollHeight - scroller.scrollTop - scroller.clientHeight < 24
}

function scrollToLatest() {
  const scroller = scrollerRef.value
  if (!scroller) return
  scroller.scrollTop = scroller.scrollHeight
}

function onPanelKeydown(event: KeyboardEvent) {
  if (props.mobile) {
    if (!modalActive || !isTopModalLayer(modalLayerToken)) return
    handleMobileModalKeydown(event)
    return
  }
  if (event.key === 'Escape') {
    event.preventDefault()
    requestClose()
  }
}

function requestClose() {
  if (props.mobile && modalActive && !isTopModalLayer(modalLayerToken)) return
  emit('close')
}

function handleDocumentKeydown(event: KeyboardEvent) {
  if (
    event.defaultPrevented
    || !props.mobile
    || !modalActive
    || !isTopModalLayer(modalLayerToken)
  ) return
  handleMobileModalKeydown(event)
}

function handleMobileModalKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    event.preventDefault()
    event.stopPropagation()
    requestClose()
    return
  }
  if (event.key !== 'Tab' || !panelRef.value) return
  event.stopPropagation()
  const focusable = getVisibleFocusableElements(panelRef.value)
  if (focusable.length === 0) {
    event.preventDefault()
    panelRef.value.focus({ preventScroll: true })
    return
  }
  const first = focusable[0]
  const last = focusable[focusable.length - 1]
  if (!panelRef.value.contains(document.activeElement)) {
    event.preventDefault()
    const focusTarget = event.shiftKey ? last : first
    focusTarget?.focus({ preventScroll: true })
    return
  }
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault()
    last?.focus({ preventScroll: true })
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault()
    first?.focus({ preventScroll: true })
  }
}

async function activateMobileModal() {
  if (modalActive || typeof document === 'undefined') return
  modalActive = true
  registerModalLayer(modalLayerToken)
  acquireBodyScrollLock(scrollLockToken)
  document.addEventListener('keydown', handleDocumentKeydown)
  await nextTick()
  if (modalActive && props.mobile) {
    closeButtonRef.value?.focus({ preventScroll: true })
  }
}

function deactivateMobileModal() {
  if (!modalActive || typeof document === 'undefined') return
  modalActive = false
  document.removeEventListener('keydown', handleDocumentKeydown)
  unregisterModalLayer(modalLayerToken)
  releaseBodyScrollLock(scrollLockToken)
}

defineExpose({
  focus: () => closeButtonRef.value?.focus({ preventScroll: true }),
})
</script>

<style scoped>
.chat-activity-layer {
  display: flex;
  width: 392px;
  min-width: 360px;
  max-width: 420px;
  min-height: 0;
  flex: 0 0 392px;
}

.chat-activity-panel {
  position: relative;
  display: flex;
  width: 100%;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
  border-left: 1px solid var(--workspace-divider);
  color: var(--workspace-text);
  background: var(--workspace-surface);
}

.chat-activity-panel__header {
  display: flex;
  min-height: 64px;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 12px 16px 10px 20px;
}

.chat-activity-panel__heading {
  display: flex;
  min-width: 0;
  align-items: baseline;
  gap: 10px;
}

.chat-activity-panel__heading h2 {
  margin: 0;
  color: var(--workspace-text);
  font-size: 16px;
  font-weight: 600;
  letter-spacing: -0.01em;
  line-height: 22px;
}

.chat-activity-panel__elapsed {
  color: var(--workspace-text-secondary);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  line-height: 18px;
}

.chat-activity-panel__close {
  display: grid;
  width: 36px;
  height: 36px;
  flex: 0 0 36px;
  place-items: center;
  border: 0;
  border-radius: 8px;
  color: var(--workspace-text-secondary);
  background: transparent;
}

.chat-activity-panel__close:hover {
  color: var(--workspace-text);
  background: var(--workspace-hover);
}

.chat-activity-panel__close:focus-visible {
  outline: 2px solid var(--lx-clay-accent);
  outline-offset: 2px;
}

.chat-activity-panel__progress {
  position: relative;
  height: 2px;
  overflow: hidden;
  background: color-mix(in srgb, var(--lx-clay-accent) 12%, transparent);
}

.chat-activity-panel__progress span {
  position: absolute;
  inset-block: 0;
  width: 38%;
  border-radius: 2px;
  background: var(--lx-clay-accent);
  animation: chat-activity-progress 1.35s cubic-bezier(0.4, 0, 0.2, 1) infinite;
}

.chat-activity-panel__status-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 20px 8px;
  color: var(--workspace-text-secondary);
  font-size: 13px;
  font-weight: 600;
  line-height: 20px;
}

.chat-activity-panel__status-icon {
  color: var(--lx-clay-accent);
}

.chat-activity-panel__body {
  min-height: 0;
  flex: 1;
  overflow-x: hidden;
  overflow-y: auto;
  padding: 4px 20px 32px;
  overscroll-behavior: contain;
  scrollbar-gutter: stable;
}

.chat-activity-panel__part {
  position: relative;
  padding: 14px 0 14px 18px;
}

.chat-activity-panel__part::before {
  position: absolute;
  top: 23px;
  left: 1px;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--workspace-text-muted);
  content: '';
}

.chat-activity-panel__part + .chat-activity-panel__part {
  border-top: 1px solid var(--workspace-divider);
}

.chat-activity-panel__markdown {
  color: var(--workspace-text-secondary);
  font-size: 14px;
  line-height: 1.65;
  overflow-wrap: anywhere;
}

.chat-activity-panel__markdown--streaming {
  white-space: pre-wrap;
}

.chat-activity-panel__markdown :deep(> :first-child) { margin-top: 0; }
.chat-activity-panel__markdown :deep(> :last-child) { margin-bottom: 0; }
.chat-activity-panel__markdown :deep(p),
.chat-activity-panel__markdown :deep(pre),
.chat-activity-panel__markdown :deep(blockquote) {
  margin: 0 0 10px;
}

.chat-activity-panel__markdown :deep(ul),
.chat-activity-panel__markdown :deep(ol) {
  margin: 0 0 10px;
  padding-left: 20px;
}

.chat-activity-panel__markdown :deep(code) {
  border-radius: 6px;
  padding: 1px 5px;
  color: var(--workspace-text);
  background: var(--workspace-surface-subtle);
  font-family: var(--workspace-font-mono);
  font-size: 0.92em;
}

.chat-activity-panel__markdown :deep(pre) {
  overflow-x: auto;
  border-radius: 10px;
  padding: 12px;
  background: var(--workspace-surface-subtle);
}

.chat-activity-panel__markdown :deep(pre code) {
  padding: 0;
  background: transparent;
}

.chat-activity-layer--mobile {
  position: fixed;
  inset: 0;
  z-index: 46;
  display: flex;
  width: 100%;
  min-width: 0;
  max-width: none;
  justify-content: flex-end;
}

.chat-activity-layer__scrim {
  position: absolute;
  inset: 0;
  border: 0;
  background: rgb(15 23 42 / 0.38);
}

.chat-activity-panel--mobile {
  width: min(392px, 100vw);
  height: 100dvh;
  border-left: 0;
  padding-top: env(safe-area-inset-top);
  padding-bottom: env(safe-area-inset-bottom);
  box-shadow: -16px 0 40px rgb(15 23 42 / 0.18);
}

.chat-activity-panel--mobile .chat-activity-panel__close {
  width: 44px;
  height: 44px;
  flex-basis: 44px;
}

@keyframes chat-activity-progress {
  0% { transform: translateX(-110%); }
  55% { transform: translateX(125%); }
  100% { transform: translateX(285%); }
}

@media (max-width: 480px) {
  .chat-activity-panel--mobile { width: 100vw; }
}

@media (prefers-reduced-motion: reduce) {
  .chat-activity-panel__progress span {
    width: 100%;
    animation: none;
    opacity: 0.72;
  }
}
</style>
