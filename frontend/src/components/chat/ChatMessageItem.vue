<template>
  <article
    class="chat-message"
    :class="`chat-message--${message.role}`"
    :aria-label="message.role === 'user' ? t('chat.message.you') : t('chat.message.assistant')"
    :aria-busy="message.status === 'streaming' ? 'true' : undefined"
  >
    <div class="chat-message__body">
      <div
        v-if="message.excludedFromContext || message.status === 'stopped'"
        class="chat-message__meta"
      >
        <span v-if="message.excludedFromContext" class="chat-message__status">
          {{ t('chat.message.superseded') }}
        </span>
        <span v-else-if="message.status === 'stopped'" class="chat-message__status">
          {{ t('chat.message.stopped') }}
        </span>
      </div>

      <ChatMessageAttachments
        v-if="message.attachments?.length"
        :attachments="message.attachments"
      />

      <div
        v-if="message.role === 'assistant' && hasRenderableContent"
        class="chat-message__markdown"
        :class="{ 'chat-message__markdown--streaming': message.status === 'streaming' }"
        v-html="renderedContent"
      ></div>
      <p v-else-if="hasRenderableContent" class="chat-message__plain">{{ message.content }}</p>

      <div
        v-else-if="showStreamingIndicator"
        class="chat-message__streaming-placeholder"
        aria-hidden="true"
      >
        <span class="chat-message__streaming-dot"></span>
      </div>

      <div
        v-if="message.status === 'error' && !message.excludedFromContext"
        class="chat-message__failure"
        role="group"
        :aria-label="t('chat.message.failed')"
      >
        <div
          class="chat-message__error"
          :role="announceFailure ? 'alert' : undefined"
        >
          <Icon name="exclamationCircle" size="sm" aria-hidden="true" />
          <span>{{ t(errorPresentation.messageKey) }}</span>
        </div>
        <button
          v-if="(retryable || retrying) && errorPresentation.retryable"
          type="button"
          class="chat-message__retry-button"
          :disabled="retrying"
          :aria-busy="retrying ? 'true' : undefined"
          @click="$emit('retry')"
        >
          <Icon name="refresh" size="sm" aria-hidden="true" />
          <span>{{ t(retrying ? 'chat.actions.retrying' : 'chat.actions.retryFailed') }}</span>
        </button>
      </div>

      <div
        v-if="message.role === 'assistant' && message.status !== 'streaming' && (hasRenderableContent || ((retryable || retrying) && message.status !== 'error'))"
        class="chat-message__actions"
      >
        <button
          v-if="hasRenderableContent"
          type="button"
          class="chat-message__icon-button"
          :title="copied ? t('chat.actions.copied') : t('chat.actions.copy')"
          :aria-label="copied ? t('chat.actions.copied') : t('chat.actions.copy')"
          @click="copyMessage"
        >
          <Icon :name="copied ? 'check' : 'copy'" size="sm" />
        </button>
        <button
          v-if="(retryable || retrying) && message.status !== 'error'"
          type="button"
          class="chat-message__icon-button"
          :title="t(retrying ? 'chat.actions.retrying' : 'chat.actions.retry')"
          :aria-label="t(retrying ? 'chat.actions.retrying' : 'chat.actions.retry')"
          :disabled="retrying"
          :aria-busy="retrying ? 'true' : undefined"
          @click="$emit('retry')"
        >
          <Icon name="refresh" size="sm" />
        </button>
      </div>
    </div>
  </article>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import DOMPurify from 'dompurify'
import { marked, Renderer } from 'marked'
import ChatMessageAttachments from '@/components/chat/ChatMessageAttachments.vue'
import Icon from '@/components/icons/Icon.vue'
import { describeChatMessageError } from '@/features/chat/chatErrorHandler'
import type { ChatMessage } from '@/types/chat'

const props = withDefaults(defineProps<{
  message: ChatMessage
  retryable?: boolean
  retrying?: boolean
  announceFailure?: boolean
}>(), {
  retryable: false,
  retrying: false,
  announceFailure: false,
})

defineEmits<{
  retry: []
}>()

const { t } = useI18n()
const copied = ref(false)
let copiedTimer: ReturnType<typeof setTimeout> | undefined
const hasRenderableContent = computed(() => props.message.content.trim().length > 0)
const errorPresentation = computed(() => describeChatMessageError(props.message))
const showStreamingIndicator = computed(() => (
  props.message.role === 'assistant'
  && props.message.status === 'streaming'
  && !props.message.excludedFromContext
  && !hasRenderableContent.value
))

const markdownRenderer = new Renderer()

markdownRenderer.code = ({ text, lang }) => {
  const language = formatCodeLanguage(lang)
  return `<pre data-language="${escapeHtml(language)}"><code>${escapeHtml(text)}</code></pre>`
}

const renderedContent = computed(() => {
  const html = marked.parse(props.message.content, {
    breaks: true,
    gfm: true,
    renderer: markdownRenderer,
  })

  return DOMPurify.sanitize(String(html), {
    ALLOWED_TAGS: [
      'a', 'blockquote', 'br', 'code', 'del', 'em', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6',
      'hr', 'li', 'ol', 'p', 'pre', 'strong', 'table', 'tbody', 'td', 'th', 'thead', 'tr', 'ul',
    ],
    ALLOWED_ATTR: ['data-language', 'href', 'title'],
    ALLOW_DATA_ATTR: false,
    ALLOW_ARIA_ATTR: false,
  })
})

function escapeHtml(value: string): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

function formatCodeLanguage(rawLanguage: string | undefined): string {
  const normalized = rawLanguage?.trim().split(/\s+/, 1)[0]?.toLowerCase() ?? 'text'
  const labels: Record<string, string> = {
    bash: 'Bash',
    css: 'CSS',
    go: 'Go',
    html: 'HTML',
    js: 'JavaScript',
    javascript: 'JavaScript',
    json: 'JSON',
    jsx: 'JSX',
    markdown: 'Markdown',
    md: 'Markdown',
    plaintext: 'Text',
    py: 'Python',
    python: 'Python',
    rust: 'Rust',
    sh: 'Shell',
    shell: 'Shell',
    sql: 'SQL',
    text: 'Text',
    ts: 'TypeScript',
    tsx: 'TSX',
    txt: 'Text',
    vue: 'Vue',
    yaml: 'YAML',
    yml: 'YAML',
  }
  if (labels[normalized]) return labels[normalized]

  const safeLabel = normalized.match(/^[a-z0-9+#.\-]+/)?.[0]?.slice(0, 24) ?? ''
  return safeLabel || 'Text'
}

async function copyMessage() {
  if (!props.message.content || !navigator.clipboard) return
  try {
    await navigator.clipboard.writeText(props.message.content)
    copied.value = true
    if (copiedTimer) clearTimeout(copiedTimer)
    copiedTimer = setTimeout(() => {
      copied.value = false
    }, 1600)
  } catch {
    copied.value = false
  }
}

onBeforeUnmount(() => {
  if (copiedTimer) clearTimeout(copiedTimer)
})
</script>

<style scoped>
.chat-message {
  --chat-message-text: #0d0d0d;
  --chat-message-user-text: #0d0d0d;
  --chat-message-user-surface: #f4f4f4;
  --chat-message-inline-code-surface: #ececec;
  --chat-message-code-surface: #f4f4f4;
  --chat-message-code-header: #e8e8e8;
  --chat-message-divider: rgb(0 0 0 / 0.15);
  --chat-message-divider-subtle: rgb(0 0 0 / 0.05);
  --chat-message-action-hover: #ececec;
  --chat-message-streaming-dot: #0d0d0d;

  display: block;
  min-width: 0;
  width: min(100%, 768px);
  margin: 0 auto;
  padding: 0;
  color: var(--chat-message-text);
  font-size: 16px;
  font-weight: 400;
  line-height: 26px;
}

.chat-message--assistant {
  border: 0;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
}

.chat-message__body {
  min-width: 0;
}

.chat-message--user .chat-message__body {
  display: flex;
  width: 100%;
  flex-direction: column;
  align-items: flex-end;
}

.chat-message__meta {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 20px;
  margin-bottom: 8px;
  color: var(--workspace-text-secondary);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
}

.chat-message__status {
  color: var(--workspace-text-secondary);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
}

.chat-message__plain {
  margin: 0;
  color: var(--chat-message-text);
  font-size: 16px;
  font-weight: 400;
  line-height: 26px;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.chat-message--user .chat-message__plain {
  width: fit-content;
  max-width: min(70%, 640px);
  border-radius: 22px;
  padding: 10px 16px;
  color: var(--chat-message-user-text);
  background: var(--chat-message-user-surface);
  line-height: 24px;
}

.chat-message__markdown {
  color: var(--chat-message-text);
  font-size: 16px;
  font-weight: 400;
  line-height: 26px;
  overflow-wrap: anywhere;
}

.chat-message__markdown--streaming {
  animation: chat-streaming-content-enter 180ms ease-out both;
}

.chat-message__markdown :deep(> :first-child) {
  margin-top: 0;
}

.chat-message__markdown :deep(> :last-child) {
  margin-bottom: 0;
}

.chat-message__markdown :deep(p) {
  margin: 0 0 16px;
}

.chat-message__markdown :deep(h1),
.chat-message__markdown :deep(h2),
.chat-message__markdown :deep(h3),
.chat-message__markdown :deep(h4),
.chat-message__markdown :deep(h5),
.chat-message__markdown :deep(h6) {
  color: inherit;
  font-weight: 600;
  letter-spacing: normal;
}

.chat-message__markdown :deep(h1) {
  margin: 28px 0 8px;
  font-size: 24px;
  line-height: 32px;
}

.chat-message__markdown :deep(h2) {
  margin: 28px 0 8px;
  font-size: 20px;
  line-height: 28px;
}

.chat-message__markdown :deep(h3) {
  margin: 16px 0 8px;
  font-size: 18px;
  line-height: 28px;
}

.chat-message__markdown :deep(h4),
.chat-message__markdown :deep(h5),
.chat-message__markdown :deep(h6) {
  margin: 16px 0 0;
  font-size: 16px;
  line-height: 24px;
}

.chat-message__markdown :deep(ul),
.chat-message__markdown :deep(ol) {
  margin: 16px 0 8px;
  padding-left: 26px;
}

.chat-message__markdown :deep(ul) {
  list-style: disc;
}

.chat-message__markdown :deep(ol) {
  list-style: decimal;
}

.chat-message__markdown :deep(li) {
  margin: 0;
  padding-left: 6px;
  line-height: 26px;
}

.chat-message__markdown :deep(li::marker) {
  color: currentColor;
  font-size: 16px;
  font-weight: 700;
}

.chat-message__markdown :deep(li > p) {
  margin: 0;
}

.chat-message__markdown :deep(li > p + p) {
  margin-top: 8px;
}

.chat-message__markdown :deep(li > ul),
.chat-message__markdown :deep(li > ol) {
  margin: 8px 0 0;
}

.chat-message__markdown :deep(ul ul) {
  list-style: circle;
}

.chat-message__markdown :deep(ul ul ul) {
  list-style: square;
}

.chat-message__markdown :deep(strong) {
  font-weight: 600;
}

.chat-message__markdown :deep(del) {
  color: var(--workspace-text-secondary);
}

.chat-message__markdown :deep(a) {
  color: var(--lx-clay-accent);
  text-decoration: underline;
  text-underline-offset: 3px;
}

.chat-message__markdown :deep(code) {
  border-radius: 6px;
  padding: 2px 5px;
  color: var(--chat-message-text);
  background: var(--chat-message-inline-code-surface);
  font-family: var(--workspace-font-mono);
  font-size: 14px;
  font-weight: 500;
  line-height: 26px;
}

.chat-message__markdown :deep(pre) {
  position: relative;
  max-width: 100%;
  margin: 8px 0 16px;
  overflow-x: auto;
  border: 0;
  border-radius: 24px;
  padding: 0 16px 16px;
  color: var(--chat-message-text);
  background: var(--chat-message-code-surface);
  box-shadow: inset 0 0 0 1px var(--workspace-border);
  font-size: 14px;
  font-weight: 400;
  line-height: 24px;
  tab-size: 2;
}

.chat-message__markdown :deep(pre::before) {
  content: attr(data-language);
  display: block;
  min-width: max-content;
  margin: 0 -16px 14px;
  border-bottom: 1px solid var(--workspace-divider);
  padding: 8px 12px;
  color: var(--workspace-text-secondary);
  background: var(--chat-message-code-header);
  font-size: 14px;
  font-weight: 500;
  line-height: 24px;
}

.chat-message__markdown :deep(pre code) {
  padding: 0;
  color: inherit;
  background: transparent;
  font-size: 12.25px;
  font-weight: 400;
  line-height: 20px;
}

.chat-message__markdown :deep(blockquote) {
  position: relative;
  margin: 16px 0 8px;
  border: 0;
  padding: 8px 0 8px 24px;
  color: inherit;
  background: transparent;
  line-height: 24px;
}

.chat-message__markdown :deep(blockquote::after) {
  content: '';
  position: absolute;
  top: 8px;
  left: 0;
  width: 4px;
  height: 28px;
  border-radius: 999px;
  background: var(--chat-message-divider);
}

.chat-message__markdown :deep(blockquote > :last-child) {
  margin-bottom: 0;
}

.chat-message__markdown :deep(hr) {
  height: 1px;
  margin: 28px 0;
  border: 0;
  background: var(--chat-message-divider);
}

.chat-message__markdown :deep(> hr:last-child) {
  margin-bottom: 28px;
}

.chat-message__markdown :deep(table) {
  display: block;
  width: 100%;
  max-width: 100%;
  margin: 16px 0 8px;
  overflow-x: auto;
  border: 0;
  border-radius: 0;
  border-collapse: separate;
  border-spacing: 0;
  font-size: 14px;
  font-weight: 400;
  line-height: 24px;
}

.chat-message__markdown :deep(th),
.chat-message__markdown :deep(td) {
  min-width: 120px;
  border: 0;
  text-align: left;
}

.chat-message__markdown :deep(th) {
  border-bottom: 1px solid var(--chat-message-divider);
  padding: 8px 24px 8px 0;
  background: transparent;
  font-weight: 600;
  line-height: 16px;
  vertical-align: bottom;
}

.chat-message__markdown :deep(td) {
  border-bottom: 1px solid var(--chat-message-divider-subtle);
  padding: 10px 24px 10px 0;
  line-height: 24px;
  vertical-align: baseline;
}

.chat-message__markdown :deep(tr:last-child td) {
  border-bottom: 0;
}

.chat-message__markdown :deep(th:last-child),
.chat-message__markdown :deep(td:last-child) {
  padding-right: 0;
}

.chat-message__streaming-placeholder {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  height: 26px;
  margin: 0;
  padding: 0;
}

.chat-message__streaming-dot {
  display: block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--chat-message-streaming-dot);
  transform-origin: center;
  animation: chat-streaming-breathe 1.2s ease-in-out infinite;
}

.chat-message__failure {
  display: flex;
  align-items: flex-start;
  flex-wrap: wrap;
  gap: 10px 14px;
  margin-top: 8px;
}

.chat-message__error {
  display: flex;
  align-items: flex-start;
  gap: 7px;
  min-width: 0;
  color: var(--lx-clay-danger, #b91c1c);
  font-size: var(--workspace-type-body-size);
  font-weight: var(--workspace-type-body-weight);
  line-height: 1.5;
}

.chat-message__error span {
  min-width: 0;
  overflow-wrap: anywhere;
}

.chat-message__retry-button {
  display: inline-flex;
  min-height: 36px;
  align-items: center;
  justify-content: center;
  gap: 7px;
  border: 1px solid var(--workspace-border);
  border-radius: 10px;
  padding: 6px 12px;
  color: var(--workspace-text);
  background: transparent;
  font-size: 14px;
  font-weight: 500;
  line-height: 20px;
  transition: color 150ms ease, background-color 150ms ease, border-color 150ms ease;
}

.chat-message__retry-button:hover {
  border-color: var(--workspace-text-secondary);
  background: var(--chat-message-action-hover);
}

.chat-message__retry-button:disabled,
.chat-message__icon-button:disabled {
  cursor: wait;
  opacity: 0.6;
}

.chat-message__retry-button:focus-visible {
  outline: 2px solid var(--lx-clay-accent);
  outline-offset: 2px;
}

.chat-message__actions {
  display: flex;
  gap: 4px;
  min-height: 32px;
  margin-top: 10px;
}

.chat-message__icon-button {
  display: grid;
  place-items: center;
  width: 32px;
  height: 32px;
  border: 0;
  border-radius: 7px;
  color: var(--lx-clay-text-secondary);
  background: transparent;
  transition: color 150ms ease, background-color 150ms ease;
}

.chat-message__icon-button:hover {
  color: var(--workspace-text);
  background: var(--chat-message-action-hover);
}

:global(html.dark .chat-message) {
  --chat-message-text: #ffffff;
  --chat-message-user-text: #000000;
  --chat-message-user-surface: #ececec;
  --chat-message-inline-code-surface: #424242;
  --chat-message-code-surface: #212121;
  --chat-message-code-header: #171717;
  --chat-message-divider: rgb(255 255 255 / 0.15);
  --chat-message-divider-subtle: rgb(255 255 255 / 0.05);
  --chat-message-action-hover: #212121;
  --chat-message-streaming-dot: #ffffff;
}

.chat-message__icon-button:focus-visible {
  outline: 2px solid var(--lx-clay-accent);
  outline-offset: 2px;
}

@keyframes chat-streaming-breathe {
  0%, 100% { opacity: 0.35; transform: scale(0.72); }
  50% { opacity: 1; transform: scale(1); }
}

@keyframes chat-streaming-content-enter {
  from { opacity: 0; }
  to { opacity: 1; }
}

@media (max-width: 640px) {
  .chat-message {
    padding: 0 8px;
  }

  .chat-message--user .chat-message__plain {
    max-width: 86%;
  }

  .chat-message__icon-button {
    width: 44px;
    height: 44px;
  }

}

@media (hover: none) and (pointer: coarse) {
  .chat-message__retry-button {
    min-height: 44px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .chat-message__markdown--streaming,
  .chat-message__streaming-dot {
    animation: none;
  }

  .chat-message__streaming-dot {
    opacity: 1;
    transform: none;
  }
}
</style>
