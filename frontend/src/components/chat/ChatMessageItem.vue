<template>
  <article
    class="chat-message"
    :class="`chat-message--${message.role}`"
    :aria-label="message.role === 'user' ? t('chat.message.you') : t('chat.message.assistant')"
  >
    <div class="chat-message__avatar" aria-hidden="true">
      <Icon :name="message.role === 'user' ? 'user' : 'sparkles'" size="sm" :stroke-width="1.9" />
    </div>

    <div class="chat-message__body">
      <div class="chat-message__meta">
        <span>{{ message.role === 'user' ? t('chat.message.you') : t('chat.message.assistant') }}</span>
        <span v-if="message.status === 'streaming'" class="chat-message__streaming">
          {{ t('chat.message.generating') }}
        </span>
        <span v-else-if="message.excludedFromContext" class="chat-message__status">
          {{ t('chat.message.superseded') }}
        </span>
        <span v-else-if="message.status === 'stopped'" class="chat-message__status">
          {{ t('chat.message.stopped') }}
        </span>
      </div>

      <div
        v-if="message.role === 'assistant' && message.content && message.status !== 'streaming'"
        class="chat-message__markdown"
        v-html="renderedContent"
      ></div>
      <p v-else-if="message.content" class="chat-message__plain">{{ message.content }}</p>

      <div v-if="message.status === 'streaming' && !message.content" class="chat-message__thinking" aria-live="polite">
        <span></span><span></span><span></span>
      </div>

      <div v-if="message.status === 'error'" class="chat-message__error" role="alert">
        <Icon name="exclamationCircle" size="sm" />
        <span>{{ message.errorMessage || t('chat.errors.requestFailed') }}</span>
      </div>

      <footer
        v-if="showReceipt"
        class="chat-message__receipt"
        :class="`chat-message__receipt--${settlementStatus}`"
        data-test="chat-receipt"
      >
        <div class="chat-message__receipt-status" :aria-live="settlementStatus === 'pending' ? 'polite' : undefined">
          <Icon :name="settlementIcon" size="xs" :stroke-width="2" />
          <span>{{ settlementLabel }}</span>
        </div>

        <div v-if="settlementStatus !== 'pending'" class="chat-message__receipt-details">
          <span v-if="message.actualModel">
            {{ t('chat.receipt.model', { model: message.actualModel }) }}
          </span>
          <span v-if="message.inputTokens !== undefined">
            {{ t('chat.receipt.inputTokens', { count: formatTokens(message.inputTokens) }) }}
          </span>
          <span v-if="message.outputTokens !== undefined">
            {{ t('chat.receipt.outputTokens', { count: formatTokens(message.outputTokens) }) }}
          </span>
          <span v-if="message.cacheCreationTokens !== undefined">
            {{ t('chat.receipt.cacheCreationTokens', { count: formatTokens(message.cacheCreationTokens) }) }}
          </span>
          <span v-if="message.cacheReadTokens !== undefined">
            {{ t('chat.receipt.cacheReadTokens', { count: formatTokens(message.cacheReadTokens) }) }}
          </span>
          <span v-if="message.grossCost !== undefined">
            {{ t('chat.receipt.grossCost', { amount: formatCurrency(message.grossCost) }) }}
          </span>
          <span v-if="message.chargedAmount !== undefined" class="chat-message__receipt-charge">
            {{ t('chat.receipt.chargedAmount', { amount: formatCurrency(message.chargedAmount) }) }}
          </span>
          <span v-if="hasBalanceRange">
            {{
              t('chat.receipt.balanceRange', {
                before: formatCurrency(message.balanceBefore),
                after: formatCurrency(message.balanceAfter),
              })
            }}
          </span>
          <span v-else-if="message.balanceAfter !== undefined">
            {{ t('chat.receipt.balanceAfter', { amount: formatCurrency(message.balanceAfter) }) }}
          </span>
        </div>

        <router-link
          v-if="showLowBalance"
          class="chat-message__low-balance"
          to="/purchase"
          data-test="chat-receipt-recharge"
        >
          <Icon name="wallet" size="xs" />
          <span>{{ t('chat.receipt.lowBalance') }}</span>
          <strong>{{ t('chat.balance.recharge') }}</strong>
        </router-link>
      </footer>

      <div v-if="message.role === 'assistant' && message.status !== 'streaming'" class="chat-message__actions">
        <button
          v-if="message.content"
          type="button"
          class="chat-message__icon-button"
          :title="copied ? t('chat.actions.copied') : t('chat.actions.copy')"
          :aria-label="copied ? t('chat.actions.copied') : t('chat.actions.copy')"
          @click="copyMessage"
        >
          <Icon :name="copied ? 'check' : 'copy'" size="sm" />
        </button>
        <button
          v-if="retryable"
          type="button"
          class="chat-message__icon-button"
          :title="t('chat.actions.retry')"
          :aria-label="t('chat.actions.retry')"
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
import { marked } from 'marked'
import Icon from '@/components/icons/Icon.vue'
import type { ChatMessage } from '@/types/chat'
import { formatCurrency } from '@/utils/format'

const props = withDefaults(defineProps<{
  message: ChatMessage
  retryable?: boolean
}>(), {
  retryable: false,
})

defineEmits<{
  retry: []
}>()

const { t } = useI18n()
const copied = ref(false)
let copiedTimer: ReturnType<typeof setTimeout> | undefined

const showReceipt = computed(() => (
  props.message.role === 'assistant'
  && (!!props.message.receiptId || !!props.message.settlementStatus)
))
const settlementStatus = computed(() => props.message.settlementStatus ?? 'pending')
const settlementIcon = computed<'checkCircle' | 'clock' | 'infoCircle' | 'xCircle'>(() => {
  if (settlementStatus.value === 'pending') return 'clock'
  if (settlementStatus.value === 'charged' || settlementStatus.value === 'subscription') {
    return 'checkCircle'
  }
  if (settlementStatus.value === 'failed') return 'xCircle'
  return 'infoCircle'
})
const settlementLabel = computed(() => {
  if (settlementStatus.value === 'charged') return t('chat.receipt.status.charged')
  if (settlementStatus.value === 'not_charged') return t('chat.receipt.status.notCharged')
  if (settlementStatus.value === 'subscription') return t('chat.receipt.status.subscription')
  if (settlementStatus.value === 'failed') return t('chat.receipt.status.failed')
  return t('chat.receipt.status.pending')
})
const hasBalanceRange = computed(() => (
  props.message.balanceBefore !== undefined && props.message.balanceAfter !== undefined
))
const showLowBalance = computed(() => (
  settlementStatus.value !== 'pending'
  && props.message.balanceAfter !== undefined
  && props.message.balanceAfter < 1
))

const renderedContent = computed(() => {
  const html = marked.parse(props.message.content, {
    breaks: true,
    gfm: true,
  })

  return DOMPurify.sanitize(String(html), {
    ALLOWED_TAGS: [
      'a', 'blockquote', 'br', 'code', 'del', 'em', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6',
      'hr', 'li', 'ol', 'p', 'pre', 'strong', 'table', 'tbody', 'td', 'th', 'thead', 'tr', 'ul',
    ],
    ALLOWED_ATTR: ['href', 'title'],
    ALLOW_DATA_ATTR: false,
    ALLOW_ARIA_ATTR: false,
  })
})

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

function formatTokens(value: number): string {
  return new Intl.NumberFormat(undefined, { maximumFractionDigits: 0 }).format(value)
}

onBeforeUnmount(() => {
  if (copiedTimer) clearTimeout(copiedTimer)
})
</script>

<style scoped>
.chat-message {
  display: grid;
  grid-template-columns: 34px minmax(0, 1fr);
  gap: 12px;
  width: min(100%, 860px);
  margin: 0 auto;
  padding: 18px 22px;
}

.chat-message--assistant {
  border: 1px solid var(--lx-clay-border);
  border-radius: 8px;
  background: color-mix(in srgb, var(--lx-clay-surface) 94%, transparent);
  box-shadow: var(--lx-clay-shadow-flat);
}

.chat-message__avatar {
  display: grid;
  place-items: center;
  width: 32px;
  height: 32px;
  border-radius: 8px;
  color: var(--lx-clay-accent);
  background: var(--lx-clay-accent-soft);
}

.chat-message--user .chat-message__avatar {
  color: var(--lx-clay-text-secondary);
  background: var(--lx-clay-recessed);
}

.chat-message__body {
  min-width: 0;
}

.chat-message__meta {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 24px;
  margin-bottom: 5px;
  color: var(--lx-clay-text);
  font-size: 13px;
  font-weight: 800;
}

.chat-message__streaming,
.chat-message__status {
  color: var(--lx-clay-text-secondary);
  font-size: 12px;
  font-weight: 600;
}

.chat-message__plain {
  margin: 0;
  color: var(--lx-clay-text);
  font-size: 15px;
  line-height: 1.75;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.chat-message__markdown {
  color: var(--lx-clay-text);
  font-size: 15px;
  line-height: 1.75;
  overflow-wrap: anywhere;
}

.chat-message__markdown :deep(> :first-child) {
  margin-top: 0;
}

.chat-message__markdown :deep(> :last-child) {
  margin-bottom: 0;
}

.chat-message__markdown :deep(p),
.chat-message__markdown :deep(ul),
.chat-message__markdown :deep(ol),
.chat-message__markdown :deep(blockquote),
.chat-message__markdown :deep(pre),
.chat-message__markdown :deep(table) {
  margin: 0 0 12px;
}

.chat-message__markdown :deep(ul),
.chat-message__markdown :deep(ol) {
  padding-left: 22px;
}

.chat-message__markdown :deep(a) {
  color: var(--lx-clay-accent);
  text-decoration: underline;
  text-underline-offset: 3px;
}

.chat-message__markdown :deep(code) {
  border-radius: 5px;
  padding: 2px 5px;
  color: var(--lx-clay-text);
  background: var(--lx-clay-recessed);
  font-family: var(--lx-clay-font-mono);
  font-size: 0.88em;
}

.chat-message__markdown :deep(pre) {
  max-width: 100%;
  overflow-x: auto;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 8px;
  padding: 14px 16px;
  color: #f8f5fc;
  background: #120f18;
}

.chat-message__markdown :deep(pre code) {
  padding: 0;
  color: inherit;
  background: transparent;
}

.chat-message__markdown :deep(blockquote) {
  border: 1px solid var(--lx-clay-border);
  border-radius: 6px;
  padding: 9px 12px;
  color: var(--lx-clay-text-secondary);
  background: var(--lx-clay-recessed);
}

.chat-message__markdown :deep(table) {
  display: block;
  max-width: 100%;
  overflow-x: auto;
  border-collapse: collapse;
}

.chat-message__markdown :deep(th),
.chat-message__markdown :deep(td) {
  border: 1px solid var(--lx-clay-border);
  padding: 7px 10px;
  text-align: left;
}

.chat-message__thinking {
  display: flex;
  align-items: center;
  gap: 4px;
  height: 24px;
}

.chat-message__thinking span {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--lx-clay-accent);
  animation: chat-thinking 1.15s ease-in-out infinite;
}

.chat-message__thinking span:nth-child(2) { animation-delay: 0.14s; }
.chat-message__thinking span:nth-child(3) { animation-delay: 0.28s; }

.chat-message__error {
  display: flex;
  align-items: flex-start;
  gap: 7px;
  margin-top: 8px;
  color: var(--lx-clay-danger, #b91c1c);
  font-size: 13px;
  line-height: 1.5;
}

.chat-message__receipt {
  display: flex;
  align-items: flex-start;
  gap: 8px 14px;
  flex-wrap: wrap;
  margin-top: 13px;
  border-top: 1px solid var(--lx-clay-border);
  padding-top: 10px;
  color: var(--lx-clay-text-muted);
  font-size: 11px;
  line-height: 1.55;
}

.chat-message__receipt-status {
  display: inline-flex;
  min-height: 22px;
  align-items: center;
  gap: 5px;
  color: var(--lx-clay-text-secondary);
  font-weight: 700;
}

.chat-message__receipt--charged .chat-message__receipt-status,
.chat-message__receipt--subscription .chat-message__receipt-status {
  color: var(--lx-clay-success-text);
}

.chat-message__receipt--failed .chat-message__receipt-status {
  color: var(--lx-clay-danger);
}

.chat-message__receipt-details {
  display: flex;
  min-width: min(100%, 520px);
  flex: 1;
  align-items: center;
  gap: 3px 12px;
  flex-wrap: wrap;
  font-variant-numeric: tabular-nums;
}

.chat-message__receipt-details > span {
  white-space: nowrap;
}

.chat-message__receipt-charge {
  color: var(--lx-clay-text-secondary);
  font-weight: 700;
}

.chat-message__low-balance {
  display: inline-flex;
  min-height: 28px;
  align-items: center;
  gap: 5px;
  color: var(--lx-clay-warning);
  text-decoration: none;
}

.chat-message__low-balance strong {
  color: var(--lx-clay-accent-deep);
  font-weight: 800;
}

.chat-message__low-balance:hover strong {
  text-decoration: underline;
  text-underline-offset: 3px;
}

.chat-message__actions {
  display: flex;
  gap: 4px;
  min-height: 30px;
  margin-top: 8px;
}

.chat-message__icon-button {
  display: grid;
  place-items: center;
  width: 30px;
  height: 30px;
  border: 0;
  border-radius: 7px;
  color: var(--lx-clay-text-secondary);
  background: transparent;
  transition: color 150ms ease, background-color 150ms ease;
}

.chat-message__icon-button:hover {
  color: var(--lx-clay-accent);
  background: var(--lx-clay-accent-soft);
}

.chat-message__icon-button:focus-visible {
  outline: 2px solid var(--lx-clay-accent);
  outline-offset: 2px;
}

@keyframes chat-thinking {
  0%, 70%, 100% { opacity: 0.35; transform: translateY(0); }
  35% { opacity: 1; transform: translateY(-3px); }
}

@media (max-width: 640px) {
  .chat-message {
    grid-template-columns: 30px minmax(0, 1fr);
    gap: 9px;
    padding: 15px 12px;
  }

  .chat-message__avatar {
    width: 30px;
    height: 30px;
  }

  .chat-message__icon-button {
    width: 44px;
    height: 44px;
  }

  .chat-message__receipt {
    gap: 5px 10px;
  }

  .chat-message__receipt-status,
  .chat-message__receipt-details,
  .chat-message__low-balance {
    width: 100%;
  }

  .chat-message__receipt-details > span {
    white-space: normal;
  }
}

@media (prefers-reduced-motion: reduce) {
  .chat-message__thinking span {
    animation: none;
    opacity: 0.7;
  }
}
</style>
