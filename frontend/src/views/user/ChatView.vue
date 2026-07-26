<template>
  <AppLayout variant="chat">
    <div class="chat-workspace">
      <ChatHistoryPanel
        class="chat-workspace__history chat-workspace__history--desktop"
        :conversations="historyConversations"
        :active-id="chatStore.activeConversationId"
        :search-query="historySearchQuery"
        :searching="chatStore.searchingHistory"
        :has-more="chatStore.conversationsHaveMore"
        :loading-more="chatStore.loadingConversationPage"
        @new="startNewConversation"
        @select="selectConversation"
        @rename="chatStore.renameConversation"
        @delete="confirmDelete"
        @clear="confirmClear"
        @update:search-query="updateHistorySearch"
        @search="runHistorySearch"
        @load-more="chatStore.loadConversationPage()"
      />

      <Transition name="chat-drawer">
        <div
          v-if="historyModalActive"
          ref="historyDrawerRef"
          class="chat-workspace__drawer"
          role="dialog"
          aria-modal="true"
          :aria-label="t('chat.history.title')"
          tabindex="-1"
          @keydown="onHistoryDrawerKeydown"
        >
          <button
            type="button"
            class="chat-workspace__scrim"
            :aria-label="t('chat.actions.closeHistory')"
            tabindex="-1"
            @click="closeHistory()"
          ></button>
          <ChatHistoryPanel
            mobile
            :conversations="historyConversations"
            :active-id="chatStore.activeConversationId"
            :search-query="historySearchQuery"
            :searching="chatStore.searchingHistory"
            :has-more="chatStore.conversationsHaveMore"
            :loading-more="chatStore.loadingConversationPage"
            @new="startNewConversation"
            @close="closeHistory"
            @select="selectConversation"
            @rename="chatStore.renameConversation"
            @delete="confirmDelete"
            @clear="confirmClear"
            @update:search-query="updateHistorySearch"
            @search="runHistorySearch"
            @load-more="chatStore.loadConversationPage()"
          />
        </div>
      </Transition>

      <section
        ref="chatMainRef"
        class="chat-workspace__main"
        :aria-label="t('chat.title')"
        :aria-hidden="historyModalActive ? 'true' : undefined"
        :inert="historyModalActive ? true : undefined"
        tabindex="-1"
      >
        <header class="chat-toolbar">
          <div class="chat-toolbar__primary">
            <button
              ref="historyTriggerRef"
              type="button"
              class="chat-toolbar__history-button"
              :aria-label="t('chat.history.title')"
              :title="t('chat.history.title')"
              @click="openHistory"
            >
              <Icon name="menu" size="sm" />
            </button>

            <Select
              v-model="selectedModel"
              class="chat-toolbar__model"
              :options="modelOptions"
              :disabled="modelsLoading || chatStore.isStreaming || modelOptions.length === 0"
              :placeholder="modelsLoading ? t('chat.models.loading') : t('chat.models.select')"
              :aria-label="t('chat.models.select')"
              :empty-text="t('chat.models.empty')"
            >
              <template #selected="{ option }">
                <span v-if="option" class="chat-model-option chat-model-option--selected">
                  <Icon name="sparkles" size="xs" />
                  <span>{{ option.label }}</span>
                </span>
              </template>
              <template #option="{ option, selected }">
                <span class="chat-model-option">
                  <span class="chat-model-option__mark"><Icon name="sparkles" size="xs" /></span>
                  <span class="chat-model-option__copy">
                    <strong>{{ option.label }}</strong>
                    <small v-if="option.description">{{ option.description }}</small>
                  </span>
                  <span v-if="option.recommended" class="chat-model-option__recommended">
                    {{ t('chat.models.recommended') }}
                  </span>
                  <Icon v-if="selected" name="check" size="sm" class="chat-model-option__check" />
                </span>
              </template>
            </Select>

            <span class="chat-toolbar__divider" aria-hidden="true"></span>
            <div class="chat-toolbar__session" :title="activeTitle">
              <Icon name="chatBubble" size="xs" />
              <span>{{ activeTitle }}</span>
            </div>
          </div>

          <router-link
            class="chat-toolbar__balance"
            to="/purchase"
            :aria-label="`${t('chat.balance.available')} ${formattedBalance}`"
          >
            <Icon name="wallet" size="sm" />
            <span>
              <small>{{ t('chat.balance.available') }}</small>
              <strong>{{ formattedBalance }}</strong>
            </span>
          </router-link>
        </header>

        <div v-if="modelsError" class="chat-catalog-error" role="alert">
          <Icon name="exclamationTriangle" size="md" />
          <div>
            <strong>{{ t('chat.models.loadFailed') }}</strong>
            <span>{{ modelsError }}</span>
          </div>
          <button type="button" class="btn btn-secondary" @click="loadCatalog()">
            <Icon name="refresh" size="sm" />
            {{ t('chat.actions.retry') }}
          </button>
        </div>

        <div
          v-if="chatStore.hydrated && !chatStore.persistenceAvailable"
          class="chat-persistence-warning"
          role="alert"
          data-test="chat-persistence-warning"
        >
          <Icon name="exclamationTriangle" size="md" />
          <div>
            <strong>{{ t('chat.persistence.title') }}</strong>
            <span>{{ t('chat.persistence.description') }}</span>
          </div>
          <button
            type="button"
            class="btn btn-secondary"
            :disabled="persistenceRetrying"
            :aria-label="t('chat.persistence.retry')"
            :title="t('chat.persistence.retry')"
            data-test="chat-persistence-retry"
            @click="retryPersistence"
          >
            <Icon name="refresh" size="sm" />
            <span>{{ t('chat.persistence.retry') }}</span>
          </button>
        </div>

        <div
          v-if="chatStore.syncStatus === 'offline' || chatStore.syncStatus === 'error'"
          class="chat-persistence-warning"
          role="status"
          data-test="chat-sync-warning"
        >
          <Icon name="cloud" size="md" />
          <div>
            <strong>{{ t(`chat.sync.${chatStore.syncStatus}Title`) }}</strong>
            <span>{{ t(`chat.sync.${chatStore.syncStatus}Description`) }}</span>
          </div>
          <button
            type="button"
            class="btn btn-secondary"
            @click="syncServerHistory"
          >
            <Icon name="refresh" size="sm" />
            <span>{{ t('chat.sync.retry') }}</span>
          </button>
        </div>

        <p class="sr-only" aria-live="polite">{{ streamAnnouncement }}</p>

        <div class="chat-messages-region">
          <div
            ref="messageScrollerRef"
            class="chat-messages"
            @scroll.passive="handleMessageScroll"
            @wheel.passive="handleMessageWheel"
            @touchstart.passive="handleMessageTouchStart"
            @touchmove.passive="handleMessageTouchMove"
            @touchend.passive="handleMessageTouchEnd"
            @touchcancel.passive="handleMessageTouchEnd"
          >
            <div v-if="!activeConversation || activeConversation.messages.length === 0" class="chat-empty-state">
              <div class="chat-empty-state__icon"><Icon name="sparkles" size="xl" :stroke-width="1.7" /></div>
              <h2>{{ t('chat.empty.title') }}</h2>
              <p>{{ t('chat.empty.description') }}</p>
            </div>

            <div v-else class="chat-messages__inner">
              <button
                v-if="activeConversation.messagesHasMore"
                type="button"
                class="chat-messages__load-older"
                :disabled="chatStore.loadingConversationMessages.has(activeConversation.id)"
                @click="chatStore.loadOlderConversationMessages(activeConversation.id)"
              >
                {{ t('chat.history.loadOlderMessages') }}
              </button>
              <ChatMessageItem
                v-for="(message, index) in activeConversation.messages"
                :key="message.id"
                :message="message"
                :retryable="canRetryMessage(message, index)"
                @retry="retryMessage(message.id)"
              />
            </div>
          </div>

          <Transition name="chat-scroll-control">
            <button
              v-if="showScrollToLatest"
              type="button"
              class="chat-scroll-to-latest"
              :aria-label="t('chat.actions.scrollToLatest')"
              :title="t('chat.actions.scrollToLatest')"
              data-test="chat-scroll-to-latest"
              @click="scrollToLatest"
            >
              <Icon name="arrowDown" size="sm" />
            </button>
          </Transition>
        </div>

        <div class="chat-composer-region">
          <ChatComposer
            ref="composerRef"
            v-model="composerDraft"
            :streaming="chatStore.isStreaming"
            :disabled="!chatReady || !selectedModelAvailable"
            :insufficient-balance="insufficientBalance"
            @send="sendMessage"
            @stop="stopStreaming"
          />
        </div>
      </section>
    </div>

    <BaseDialog
      :show="confirmation !== null"
      :title="confirmation?.kind === 'clear' ? t('chat.confirm.clearTitle') : t('chat.confirm.deleteTitle')"
      width="narrow"
      @close="confirmation = null"
    >
      <p class="text-sm leading-6 text-gray-600 dark:text-dark-300">
        {{ confirmation?.kind === 'clear' ? t('chat.confirm.clearDescription') : t('chat.confirm.deleteDescription') }}
      </p>
      <template #footer>
        <button type="button" class="btn btn-secondary" @click="confirmation = null">
          {{ t('common.cancel') }}
        </button>
        <button type="button" class="btn btn-danger" @click="applyConfirmation">
          {{ t('common.delete') }}
        </button>
      </template>
    </BaseDialog>

    <BaseDialog
      :show="legacyImportPromptVisible && chatStore.legacyImportRequired"
      :title="t('chat.import.title')"
      width="narrow"
      @close="deferLegacyImport"
    >
      <p class="text-sm leading-6 text-gray-600 dark:text-dark-300">
        {{ t('chat.import.description', { count: chatStore.legacyConversationIds.length }) }}
      </p>
      <template #footer>
        <button type="button" class="btn btn-secondary" @click="deferLegacyImport">
          {{ t('chat.import.later') }}
        </button>
        <button type="button" class="btn btn-secondary" @click="declineLegacyImport">
          {{ t('chat.import.keepLocal') }}
        </button>
        <button type="button" class="btn btn-primary" @click="acceptLegacyImport">
          {{ t('chat.import.accept') }}
        </button>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import ChatComposer from '@/components/chat/ChatComposer.vue'
import ChatHistoryPanel from '@/components/chat/ChatHistoryPanel.vue'
import ChatMessageItem from '@/components/chat/ChatMessageItem.vue'
import {
  ChatAPIError,
  createChatAttemptId,
  getChatModels,
  isAbortError,
  pollChatReceipt,
  streamChatCompletion,
} from '@/api/chat'
import { useAuthStore } from '@/stores/auth'
import { useChatStore, type ChatMessagePatch } from '@/stores/chat'
import type {
  ChatCompletionRequest,
  ChatMessage,
  ChatModel,
  ChatReceipt,
} from '@/types/chat'

interface ChatModelOption extends SelectOption {
  value: string
  label: string
  description: string
  recommended: boolean
}

type Confirmation =
  | { kind: 'delete'; conversationId: string }
  | { kind: 'clear' }

const SCROLL_FOLLOW_RESUME_DISTANCE = 8

const { t } = useI18n()
const authStore = useAuthStore()
const chatStore = useChatStore()

const models = ref<ChatModel[]>([])
const modelsLoading = ref(true)
const modelsError = ref('')
const defaultModel = ref('')
const catalogBalance = ref<number | null>(null)
const catalogBalanceAuthStateVersion = ref(-1)
const authBalanceStateVersion = ref(0)
const balanceRejected = ref(false)
const persistenceRetrying = ref(false)
const composerDraft = ref('')
const historySearchQuery = ref('')
const legacyImportPromptVisible = ref(true)
const historyOpen = ref(false)
const mobileHistoryLayout = ref(true)
const confirmation = ref<Confirmation | null>(null)
const messageScrollerRef = ref<HTMLElement | null>(null)
const composerRef = ref<InstanceType<typeof ChatComposer> | null>(null)
const historyDrawerRef = ref<HTMLElement | null>(null)
const historyTriggerRef = ref<HTMLButtonElement | null>(null)
const chatMainRef = ref<HTMLElement | null>(null)
const shouldFollowStream = ref(true)
const isAwayFromLatest = ref(false)
let scrollFrame = 0
let historySearchTimer: ReturnType<typeof setTimeout> | null = null
let lastMessageScrollTop = 0
let lastMessageTouchY: number | null = null
let catalogRequestVersion = 0
let receiptBalanceVersion = 0
let observedAuthUserId: string | undefined
let viewDisposed = false
let historyMediaQuery: MediaQueryList | null = null
let messageResizeObserver: ResizeObserver | null = null
const receiptPolls = new Map<string, {
  receiptId: string
  controller: AbortController
  promise: Promise<void>
}>()

const activeConversation = computed(() => chatStore.activeConversation)
const historyConversations = computed(() => (
  historySearchQuery.value.trim()
    ? chatStore.searchResults
    : chatStore.conversations
))
const activeTitle = computed(() => activeConversation.value?.title || t('chat.history.newConversation'))
const historyModalActive = computed(() => historyOpen.value && mobileHistoryLayout.value)
const chatReady = computed(() => {
  const authenticatedUserId = authStore.user?.id
  return authenticatedUserId !== null
    && authenticatedUserId !== undefined
    && chatStore.hydrated
    && chatStore.userId === String(authenticatedUserId)
})
const streamAnnouncement = computed(() => (
  chatStore.isStreaming ? t('chat.message.generating') : ''
))
const currentBalance = computed(() => {
  const accountBalance = normalizedBalance(authStore.user?.balance)
  return catalogBalance.value !== null
    && catalogBalanceAuthStateVersion.value === authBalanceStateVersion.value
    ? catalogBalance.value
    : accountBalance
})
const formattedBalance = computed(() => currentBalance.value.toFixed(2))
const insufficientBalance = computed(() => currentBalance.value <= 0 || balanceRejected.value)

const modelOptions = computed<ChatModelOption[]>(() => models.value.map((model) => {
  const label = model.display_name?.trim() || model.id
  const description = model.description?.trim() || ''
  return {
    value: model.id,
    label,
    description: description === label || description === model.id ? '' : description,
    recommended: model.recommended === true,
  }
}))
const availableModelIds = computed(() => new Set(models.value.map((model) => model.id)))
const preferredModel = computed(() => (
  models.value.find((model) => model.recommended)?.id || models.value[0]?.id || ''
))

const selectedModel = computed<string>({
  get: () => activeConversation.value?.model || defaultModel.value,
  set: (value) => {
    if (!value || !availableModelIds.value.has(value)) return
    defaultModel.value = value
    if (activeConversation.value) {
      chatStore.setConversationModel(activeConversation.value.id, value)
    }
  },
})
const selectedModelAvailable = computed(() => (
  !modelsLoading.value
  && !modelsError.value
  && !!selectedModel.value
  && availableModelIds.value.has(selectedModel.value)
))

const emptyComposerReady = computed(() => (
  chatReady.value
  && selectedModelAvailable.value
  && !insufficientBalance.value
  && (!activeConversation.value || activeConversation.value.messages.length === 0)
))
const showScrollToLatest = computed(() => (
  !shouldFollowStream.value
  && isAwayFromLatest.value
  && (activeConversation.value?.messages.length ?? 0) > 0
))

watch(
  [() => authStore.user, () => authStore.user?.balance],
  () => {
    authBalanceStateVersion.value += 1
  },
  { flush: 'sync' },
)

watch(
  () => authStore.user?.id,
  (userId) => {
    const normalizedUserId = normalizeUserId(userId)
    abortReceiptPolls()
    if (observedAuthUserId !== undefined && observedAuthUserId !== normalizedUserId) {
      composerDraft.value = ''
      historySearchQuery.value = ''
    }
    observedAuthUserId = normalizedUserId
    legacyImportPromptVisible.value = true
    resetCatalogState(userId)
    const hydration = chatStore.hydrate(userId)
    void Promise.resolve(hydration).then(() => {
      if (
        !viewDisposed
        && normalizedUserId
        && normalizedUserId === currentAuthUserId()
      ) {
        resumePendingReceipts(normalizedUserId)
        void initializeServerHistory(normalizedUserId)
      }
    })
    if (userId !== null && userId !== undefined) void loadCatalog()
  },
  { immediate: true, flush: 'sync' },
)

watch(
  [
    chatReady,
    () => chatStore.activeConversationId,
    () => activeConversation.value?.model,
    () => models.value,
  ],
  () => reconcileSelectedModel(),
  { flush: 'sync' },
)

watch(emptyComposerReady, (ready) => {
  if (ready) void focusComposer()
})

watch(
  () => activeConversation.value?.messages.map((message) => `${message.id}:${message.content.length}:${message.status}`).join('|'),
  () => scheduleScrollToBottom(),
)

watch(() => chatStore.activeConversationId, () => {
  if (historyOpen.value) void closeHistory()
  shouldFollowStream.value = true
  isAwayFromLatest.value = false
  lastMessageScrollTop = 0
  scheduleScrollToBottom()
})

function onHistoryBreakpointChange(event: MediaQueryListEvent) {
  mobileHistoryLayout.value = event.matches
  if (!event.matches && historyOpen.value) void closeHistory(chatMainRef.value)
}

onMounted(() => {
  window.addEventListener('online', syncServerHistory)
  if (typeof ResizeObserver === 'function' && messageScrollerRef.value) {
    messageResizeObserver = new ResizeObserver(() => {
      const scroller = messageScrollerRef.value
      if (scroller) {
        const { currentScrollTop } = syncMessageScrollState(scroller)
        lastMessageScrollTop = currentScrollTop
      }
      scheduleScrollToBottom()
    })
    messageResizeObserver.observe(messageScrollerRef.value)
  }
  if (typeof window.matchMedia === 'function') {
    historyMediaQuery = window.matchMedia('(max-width: 1023px)')
    mobileHistoryLayout.value = historyMediaQuery.matches
    historyMediaQuery.addEventListener('change', onHistoryBreakpointChange)
  }
})

onBeforeUnmount(() => {
  viewDisposed = true
  cancelPendingMessageScroll()
  abortReceiptPolls()
  if (historySearchTimer !== null) clearTimeout(historySearchTimer)
  window.removeEventListener('online', syncServerHistory)
  messageResizeObserver?.disconnect()
  historyMediaQuery?.removeEventListener('change', onHistoryBreakpointChange)
  if (chatStore.isStreaming) chatStore.stopStreaming()
})

async function loadCatalog(silent = false) {
  const requestedUserId = currentAuthUserId()
  if (!requestedUserId) {
    modelsLoading.value = false
    return
  }
  const requestVersion = ++catalogRequestVersion
  const requestedAuthStateVersion = authBalanceStateVersion.value
  const requestedReceiptBalanceVersion = receiptBalanceVersion
  if (!silent) {
    modelsLoading.value = true
    modelsError.value = ''
  }

  try {
    const catalog = await getChatModels()
    if (!isCurrentCatalogRequest(requestVersion, requestedUserId)) return
    models.value = catalog.models
    if (requestedReceiptBalanceVersion === receiptBalanceVersion) {
      catalogBalance.value = catalog.balance
      catalogBalanceAuthStateVersion.value = requestedAuthStateVersion
    }
    balanceRejected.value = false

    if (models.value.length === 0) {
      defaultModel.value = ''
      modelsError.value = t('chat.errors.noModelsAvailable')
      return
    }
    modelsError.value = ''
    reconcileSelectedModel()
  } catch (error) {
    if (!isCurrentCatalogRequest(requestVersion, requestedUserId)) return
    if (error instanceof ChatAPIError && String(error.code || '') === 'INSUFFICIENT_BALANCE') {
      balanceRejected.value = true
    }
    if (!silent) modelsError.value = localizedError(error, 'models')
  } finally {
    if (!silent && isCurrentCatalogRequest(requestVersion, requestedUserId)) {
      modelsLoading.value = false
    }
  }
}

function currentAuthUserId() {
  return normalizeUserId(authStore.user?.id)
}

function normalizeUserId(id: string | number | null | undefined) {
  return id === null || id === undefined ? '' : String(id)
}

function normalizedBalance(balance: unknown) {
  const candidate = Number(balance ?? 0)
  return Number.isFinite(candidate) ? candidate : 0
}

function isCurrentCatalogRequest(version: number, userId: string) {
  return version === catalogRequestVersion && userId === currentAuthUserId()
}

function resetCatalogState(userId: string | number | null | undefined) {
  catalogRequestVersion += 1
  receiptBalanceVersion += 1
  models.value = []
  modelsError.value = ''
  defaultModel.value = ''
  catalogBalance.value = null
  catalogBalanceAuthStateVersion.value = -1
  balanceRejected.value = false
  modelsLoading.value = userId !== null && userId !== undefined
}

function reconcileSelectedModel() {
  const preferred = preferredModel.value
  if (!preferred) {
    defaultModel.value = ''
    return
  }

  if (!availableModelIds.value.has(defaultModel.value)) defaultModel.value = preferred

  const conversation = activeConversation.value
  if (
    chatReady.value
    && conversation
    && !availableModelIds.value.has(conversation.model)
  ) {
    chatStore.setConversationModel(conversation.id, preferred)
  }
}

async function focusComposer() {
  await nextTick()
  composerRef.value?.focus()
}

async function retryPersistence() {
  if (persistenceRetrying.value) return
  persistenceRetrying.value = true
  try {
    await chatStore.flushPersistence()
  } catch {
    // The warning remains visible while persistence is unavailable.
  } finally {
    persistenceRetrying.value = false
  }
}

async function initializeServerHistory(expectedUserId: string) {
  await chatStore.syncHistory()
  if (viewDisposed || expectedUserId !== currentAuthUserId()) return
  await chatStore.loadConversationPage(true)
  if (viewDisposed || expectedUserId !== currentAuthUserId()) return
  await recoverInterruptedAttempts(expectedUserId)
  resumePendingReceipts(expectedUserId)
}

async function syncServerHistory() {
  const expectedUserId = currentAuthUserId()
  if (!expectedUserId) return
  await initializeServerHistory(expectedUserId)
}

function updateHistorySearch(value: string) {
  historySearchQuery.value = value
  if (historySearchTimer !== null) clearTimeout(historySearchTimer)
  historySearchTimer = setTimeout(() => {
    historySearchTimer = null
    void runHistorySearch()
  }, 250)
}

async function runHistorySearch() {
  if (historySearchTimer !== null) {
    clearTimeout(historySearchTimer)
    historySearchTimer = null
  }
  await chatStore.searchHistory(historySearchQuery.value)
}

function deferLegacyImport() {
  legacyImportPromptVisible.value = false
}

function declineLegacyImport() {
  chatStore.declineLegacyImport()
  legacyImportPromptVisible.value = false
}

function acceptLegacyImport() {
  chatStore.acceptLegacyImport()
  legacyImportPromptVisible.value = false
  void syncServerHistory()
}

async function recoverInterruptedAttempts(expectedUserId: string) {
  const candidates = chatStore.conversations.flatMap((conversation) => (
    conversation.messages
      .filter((message) => (
        message.role === 'assistant'
        && Boolean(message.attemptId)
        && (
          message.status === 'stopped'
          || message.finishReason === 'interrupted'
          || message.errorCode === 'INCOMPLETE_STREAM'
          || message.errorCode === 'NETWORK_ERROR'
        )
      ))
      .map((message) => ({
        conversationId: conversation.id,
        messageId: message.id,
      }))
  ))
  for (const candidate of candidates) {
    if (viewDisposed || expectedUserId !== currentAuthUserId()) return
    const attempt = await chatStore.recoverAttempt(
      candidate.conversationId,
      candidate.messageId,
    )
    if (attempt?.receiptId) {
      recordPendingReceipt(
        candidate.conversationId,
        candidate.messageId,
        attempt.receiptId,
      )
      void syncChatReceipt(
        candidate.conversationId,
        candidate.messageId,
        attempt.receiptId,
        expectedUserId,
      )
    }
  }
}

function startNewConversation() {
  if (chatStore.isStreaming) stopStreaming()
  if (activeConversation.value && availableModelIds.value.has(activeConversation.value.model)) {
    defaultModel.value = activeConversation.value.model
  }
  chatStore.selectConversation(null)
  reconcileSelectedModel()
  composerDraft.value = ''
  if (historyOpen.value) void closeHistory()
  void focusComposer()
}

function selectConversation(id: string) {
  if (chatStore.isStreaming && chatStore.streamingConversationId !== id) {
    stopStreaming()
  }
  if (!chatStore.selectConversation(id)) return
  void chatStore.loadConversationDetail(id)
  reconcileSelectedModel()
  if (historyOpen.value) void closeHistory()
}

async function openHistory() {
  if (!mobileHistoryLayout.value) return
  historyOpen.value = true
  await nextTick()
  const firstControl = historyDrawerRef.value?.querySelector<HTMLElement>(
    'button:not([disabled]):not([tabindex="-1"]), input:not([disabled]), [tabindex]:not([tabindex="-1"])',
  )
  ;(firstControl ?? historyDrawerRef.value)?.focus()
}

async function closeHistory(focusTarget: HTMLElement | null = historyTriggerRef.value) {
  if (!historyOpen.value) return
  historyOpen.value = false
  await nextTick()
  focusTarget?.focus()
}

function onHistoryDrawerKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    event.preventDefault()
    void closeHistory()
    return
  }
  if (event.key !== 'Tab') return

  const controls = Array.from(historyDrawerRef.value?.querySelectorAll<HTMLElement>(
    'button:not([disabled]):not([tabindex="-1"]), input:not([disabled]), [tabindex]:not([tabindex="-1"])',
  ) ?? []).filter((element) => element.getClientRects().length > 0)
  if (controls.length === 0) {
    event.preventDefault()
    historyDrawerRef.value?.focus()
    return
  }

  const first = controls[0]
  const last = controls[controls.length - 1]
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault()
    last.focus()
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault()
    first.focus()
  }
}

function confirmDelete(id: string) {
  confirmation.value = { kind: 'delete', conversationId: id }
}

function confirmClear() {
  confirmation.value = { kind: 'clear' }
}

function applyConfirmation() {
  const action = confirmation.value
  confirmation.value = null
  if (!action) return
  if (action.kind === 'clear') chatStore.clearConversations()
  else chatStore.deleteConversation(action.conversationId)
}

function buildTitle(content: string) {
  const normalized = content.replace(/\s+/g, ' ').trim()
  return normalized.length > 42 ? `${normalized.slice(0, 42)}...` : normalized
}

async function sendMessage(content: string) {
  if (!chatReady.value || !selectedModelAvailable.value || chatStore.isStreaming || insufficientBalance.value) return
  const requestModel = selectedModel.value

  let conversation = activeConversation.value
  if (!conversation) {
    conversation = chatStore.createConversation(requestModel, buildTitle(content))
  }
  if (!conversation) return

  if (conversation.messages.length === 0) {
    chatStore.renameConversation(conversation.id, buildTitle(content))
  }
  chatStore.setConversationModel(conversation.id, requestModel)
  if (!await chatStore.prepareConversationForCompletion(conversation.id)) return
  const expectedHeadMessageId = conversation.headMessageId
    ?? conversation.messages[conversation.messages.length - 1]?.id
    ?? null
  const userMessage = chatStore.addMessage(
    conversation.id,
    { role: 'user', content, status: 'complete' },
  )
  if (!userMessage) return
  const attemptId = createChatAttemptId()
  const assistant = chatStore.addMessage(conversation.id, {
    role: 'assistant',
    content: '',
    status: 'streaming',
    attemptId,
    requestedModel: requestModel,
  })
  if (!assistant) return

  await runStream({
    conversationId: conversation.id,
    model: requestModel,
    expectedHeadMessageId,
    userMessage: {
      id: userMessage.id,
      content: userMessage.content,
    },
    assistantMessageId: assistant.id,
  }, attemptId)
}

function canRetryMessage(message: ChatMessage, index: number) {
  return message.role === 'assistant'
    && message.status !== 'streaming'
    && index === (activeConversation.value?.messages.length ?? 0) - 1
    && !chatStore.isStreaming
    && selectedModelAvailable.value
    && !insufficientBalance.value
}

async function retryMessage(messageId: string) {
  const conversation = activeConversation.value
  if (!conversation || !selectedModelAvailable.value || chatStore.isStreaming || insufficientBalance.value) return
  const requestModel = selectedModel.value
  const index = conversation.messages.findIndex((message) => message.id === messageId)
  if (index < 0) return

  if (!await chatStore.prepareConversationForCompletion(conversation.id)) return
  const expectedHeadMessageId = conversation.headMessageId
    ?? conversation.messages[conversation.messages.length - 1]?.id
    ?? null
  const attemptId = createChatAttemptId()
  const replacement = chatStore.addMessage(conversation.id, {
    role: 'assistant',
    content: '',
    status: 'streaming',
    attemptId,
    requestedModel: requestModel,
  })
  if (!replacement) return
  chatStore.updateMessage(conversation.id, messageId, {
    excludedFromContext: true,
    supersededByMessageId: replacement.id,
  })
  await runStream({
    conversationId: conversation.id,
    model: requestModel,
    expectedHeadMessageId,
    assistantMessageId: replacement.id,
    retryOfMessageId: messageId,
  }, attemptId)
}

async function runStream(
  request: ChatCompletionRequest,
  attemptId: string,
) {
  const conversationId = request.conversationId
  const assistantMessageId = request.assistantMessageId
  const streamUserId = currentAuthUserId()
  const controller = new AbortController()
  if (!chatStore.startStreaming(conversationId, assistantMessageId, controller)) return
  shouldFollowStream.value = true
  scheduleScrollToBottom()

  try {
    const result = await streamChatCompletion(
      request,
      {
        onReceiptId: (receiptId) => {
          recordPendingReceipt(conversationId, assistantMessageId, receiptId)
        },
        onContent: (content) => {
          chatStore.appendStreamingContent(conversationId, assistantMessageId, content)
        },
      },
      { signal: controller.signal, attemptId },
    )
    if (result.receiptId) {
      recordPendingReceipt(conversationId, assistantMessageId, result.receiptId)
    }
    chatStore.finishStreaming(conversationId, assistantMessageId, result.finishReason)
    const receiptId = result.receiptId
      || findMessage(conversationId, assistantMessageId)?.receiptId
    if (receiptId && streamUserId) {
      void syncChatReceipt(conversationId, assistantMessageId, receiptId, streamUserId)
    }
  } catch (error) {
    const duplicateReceiptId = submittedAttemptReceiptId(error)
    if (duplicateReceiptId) {
      recordPendingReceipt(conversationId, assistantMessageId, duplicateReceiptId)
    }
    if (isAbortError(error)) {
      const receiptId = findMessage(conversationId, assistantMessageId)?.receiptId
      if (receiptId && streamUserId) {
        void syncChatReceipt(conversationId, assistantMessageId, receiptId, streamUserId)
      }
      if (streamUserId) {
        void recoverStreamAttempt(
          conversationId,
          assistantMessageId,
          streamUserId,
        )
      }
      return
    }
    const code = error instanceof ChatAPIError && typeof error.code === 'string' ? error.code : undefined
    if (code === 'INSUFFICIENT_BALANCE') balanceRejected.value = true
    chatStore.failStreaming(
      conversationId,
      assistantMessageId,
      localizedError(error, 'completion'),
      code,
    )
    const receiptId = duplicateReceiptId
      || findMessage(conversationId, assistantMessageId)?.receiptId
    if (receiptId && streamUserId) {
      void syncChatReceipt(conversationId, assistantMessageId, receiptId, streamUserId)
    }
    if (streamUserId) {
      void recoverStreamAttempt(
        conversationId,
        assistantMessageId,
        streamUserId,
      )
    }
  } finally {
    if (!viewDisposed && streamUserId && streamUserId === currentAuthUserId()) {
      void refreshAccountState(streamUserId)
    }
  }
}

async function recoverStreamAttempt(
  conversationId: string,
  assistantMessageId: string,
  expectedUserId: string,
) {
  const attempt = await chatStore.recoverAttempt(conversationId, assistantMessageId)
  if (
    !attempt?.receiptId
    || viewDisposed
    || expectedUserId !== currentAuthUserId()
  ) {
    return
  }
  recordPendingReceipt(conversationId, assistantMessageId, attempt.receiptId)
  void syncChatReceipt(
    conversationId,
    assistantMessageId,
    attempt.receiptId,
    expectedUserId,
  )
}

function stopStreaming() {
  const conversationId = chatStore.streamingConversationId
  const messageId = chatStore.streamingMessageId
  const receiptId = conversationId && messageId
    ? findMessage(conversationId, messageId)?.receiptId
    : undefined
  const streamUserId = currentAuthUserId()
  const stopped = chatStore.stopStreaming()
  if (stopped && conversationId && messageId && receiptId && streamUserId) {
    void syncChatReceipt(conversationId, messageId, receiptId, streamUserId)
  }
}

async function refreshAccountState(expectedUserId = currentAuthUserId()) {
  if (!expectedUserId || expectedUserId !== currentAuthUserId()) return
  const catalogRefresh = loadCatalog(true)
  const userRefresh = authStore.refreshUser()
  await Promise.allSettled([
    catalogRefresh,
    userRefresh,
  ])
}

function localizedError(error: unknown, source: 'models' | 'completion') {
  const code = error instanceof ChatAPIError ? String(error.code || '') : ''
  if (code === 'INSUFFICIENT_BALANCE') return t('chat.errors.insufficientBalance')
  if (code === 'CHAT_ATTEMPT_ALREADY_SUBMITTED') {
    return t('chat.errors.attemptAlreadySubmitted')
  }
  if (code === 'CHAT_MODEL_NOT_AVAILABLE' || code === 'MODEL_NOT_AVAILABLE' || code === 'MODEL_NOT_FOUND') {
    return t('chat.errors.modelUnavailable')
  }
  if (error instanceof ChatAPIError && error.status === 503) return t('chat.errors.serviceUnavailable')
  if (source === 'models') return t('chat.errors.modelsUnavailable')
  return error instanceof Error && error.message ? error.message : t('chat.errors.requestFailed')
}

function findMessage(conversationId: string, messageId: string): ChatMessage | undefined {
  return chatStore.conversations
    .find((conversation) => conversation.id === conversationId)
    ?.messages.find((message) => message.id === messageId)
}

function submittedAttemptReceiptId(error: unknown): string {
  if (
    !(error instanceof ChatAPIError)
    || String(error.code || '') !== 'CHAT_ATTEMPT_ALREADY_SUBMITTED'
    || !error.metadata
    || typeof error.metadata !== 'object'
  ) {
    return ''
  }
  const metadata = error.metadata as Record<string, unknown>
  const value = metadata.receipt_id ?? metadata.receiptId
  return typeof value === 'string' ? value.trim() : ''
}

function recordPendingReceipt(
  conversationId: string,
  messageId: string,
  receiptId: string,
): void {
  const normalizedReceiptId = receiptId.trim()
  if (!normalizedReceiptId) return
  const message = findMessage(conversationId, messageId)
  if (!message) return
  chatStore.updateMessage(conversationId, messageId, {
    receiptId: normalizedReceiptId,
    settlementStatus: message.settlementStatus ?? 'pending',
  })
}

function receiptMessagePatch(receipt: ChatReceipt): ChatMessagePatch {
  const patch: ChatMessagePatch = {
    receiptId: receipt.receiptId,
    settlementStatus: receipt.status,
  }
  if (receipt.usageLogId !== undefined) patch.usageLogId = receipt.usageLogId
  if (receipt.model !== undefined) patch.actualModel = receipt.model
  if (receipt.inputTokens !== undefined) patch.inputTokens = receipt.inputTokens
  if (receipt.outputTokens !== undefined) patch.outputTokens = receipt.outputTokens
  if (receipt.cacheCreationTokens !== undefined) {
    patch.cacheCreationTokens = receipt.cacheCreationTokens
  }
  if (receipt.cacheReadTokens !== undefined) patch.cacheReadTokens = receipt.cacheReadTokens
  if (receipt.totalTokens !== undefined) patch.totalTokens = receipt.totalTokens
  if (receipt.grossCost !== undefined) patch.grossCost = receipt.grossCost
  if (receipt.chargedAmount !== undefined) patch.chargedAmount = receipt.chargedAmount
  if (receipt.billingType !== undefined) patch.billingType = receipt.billingType
  if (receipt.balanceBefore !== undefined) patch.balanceBefore = receipt.balanceBefore
  if (receipt.balanceAfter !== undefined) patch.balanceAfter = receipt.balanceAfter
  if (receipt.createdAt !== undefined) patch.receiptCreatedAt = receipt.createdAt
  return patch
}

function applyChatReceipt(
  conversationId: string,
  messageId: string,
  receipt: ChatReceipt,
  expectedUserId: string,
): void {
  if (
    viewDisposed
    || expectedUserId !== currentAuthUserId()
    || chatStore.userId !== expectedUserId
    || !findMessage(conversationId, messageId)
  ) {
    return
  }
  chatStore.updateMessage(conversationId, messageId, receiptMessagePatch(receipt))
  if (receipt.balanceAfter !== undefined) {
    receiptBalanceVersion += 1
    catalogBalance.value = receipt.balanceAfter
    catalogBalanceAuthStateVersion.value = authBalanceStateVersion.value
    if (receipt.balanceAfter > 0) balanceRejected.value = false
  }
}

function receiptPollKey(conversationId: string, messageId: string): string {
  return `${conversationId}:${messageId}`
}

function syncChatReceipt(
  conversationId: string,
  messageId: string,
  receiptId: string,
  expectedUserId: string,
): Promise<void> {
  if (
    viewDisposed
    || !receiptId.trim()
    || expectedUserId !== currentAuthUserId()
    || chatStore.userId !== expectedUserId
  ) {
    return Promise.resolve()
  }
  const key = receiptPollKey(conversationId, messageId)
  const existing = receiptPolls.get(key)
  if (existing?.receiptId === receiptId) return existing.promise
  existing?.controller.abort()

  const controller = new AbortController()
  const promise = pollChatReceipt(receiptId, { signal: controller.signal })
    .then((receipt) => {
      applyChatReceipt(conversationId, messageId, receipt, expectedUserId)
    })
    .catch((error) => {
      if (!isAbortError(error)) {
        recordPendingReceipt(conversationId, messageId, receiptId)
      }
    })
    .finally(() => {
      if (receiptPolls.get(key)?.controller === controller) receiptPolls.delete(key)
    })
  receiptPolls.set(key, { receiptId, controller, promise })
  return promise
}

function resumePendingReceipts(expectedUserId: string): void {
  for (const conversation of chatStore.conversations) {
    for (const message of conversation.messages) {
      if (
        message.role === 'assistant'
        && message.receiptId
        && (!message.settlementStatus || message.settlementStatus === 'pending')
      ) {
        void syncChatReceipt(conversation.id, message.id, message.receiptId, expectedUserId)
      }
    }
  }
}

function abortReceiptPolls(): void {
  for (const poll of receiptPolls.values()) poll.controller.abort()
  receiptPolls.clear()
}

function handleMessageScroll() {
  const scroller = messageScrollerRef.value
  if (!scroller) return
  const { currentScrollTop, distanceFromBottom } = syncMessageScrollState(scroller)
  const movedTowardHistory = currentScrollTop + 0.5 < lastMessageScrollTop
  lastMessageScrollTop = currentScrollTop
  if (movedTowardHistory) {
    if (
      !shouldFollowStream.value
      || distanceFromBottom > SCROLL_FOLLOW_RESUME_DISTANCE
    ) {
      pauseMessageFollow()
    }
    return
  }
  if (distanceFromBottom <= SCROLL_FOLLOW_RESUME_DISTANCE) {
    shouldFollowStream.value = true
  }
}

function handleMessageWheel(event: WheelEvent) {
  const scroller = messageScrollerRef.value
  if (
    event.deltaY < 0
    && !event.ctrlKey
    && scroller
    && scroller.scrollHeight - scroller.clientHeight > SCROLL_FOLLOW_RESUME_DISTANCE
  ) {
    lastMessageScrollTop = Math.max(0, scroller.scrollTop)
    pauseMessageFollow()
  }
}

function handleMessageTouchStart(event: TouchEvent) {
  lastMessageTouchY = event.touches[0]?.clientY ?? null
  if (messageScrollerRef.value) {
    lastMessageScrollTop = Math.max(0, messageScrollerRef.value.scrollTop)
  }
}

function handleMessageTouchMove(event: TouchEvent) {
  const currentY = event.touches[0]?.clientY
  const scroller = messageScrollerRef.value
  if (
    currentY !== undefined
    && lastMessageTouchY !== null
    && currentY > lastMessageTouchY
    && scroller
    && scroller.scrollHeight - scroller.clientHeight > SCROLL_FOLLOW_RESUME_DISTANCE
  ) {
    pauseMessageFollow()
  }
  lastMessageTouchY = currentY ?? null
}

function handleMessageTouchEnd() {
  lastMessageTouchY = null
}

function pauseMessageFollow() {
  shouldFollowStream.value = false
  cancelPendingMessageScroll()
}

function cancelPendingMessageScroll() {
  if (scrollFrame === 0) return
  cancelAnimationFrame(scrollFrame)
  scrollFrame = 0
}

function scrollToLatest() {
  shouldFollowStream.value = true
  scheduleScrollToBottom()
}

function syncMessageScrollState(scroller: HTMLElement) {
  const maxScrollTop = Math.max(0, scroller.scrollHeight - scroller.clientHeight)
  const currentScrollTop = Math.max(0, scroller.scrollTop)
  const distanceFromBottom = Math.max(0, maxScrollTop - currentScrollTop)
  isAwayFromLatest.value = maxScrollTop > 0 && distanceFromBottom > SCROLL_FOLLOW_RESUME_DISTANCE
  return { currentScrollTop, distanceFromBottom }
}

function scheduleScrollToBottom() {
  cancelPendingMessageScroll()
  scrollFrame = requestAnimationFrame(async () => {
    scrollFrame = 0
    await nextTick()
    if (viewDisposed) return
    const scroller = messageScrollerRef.value
    if (!scroller) return
    const { currentScrollTop } = syncMessageScrollState(scroller)
    if (!shouldFollowStream.value) return
    lastMessageScrollTop = currentScrollTop
    isAwayFromLatest.value = false
    if (typeof scroller.scrollTo === 'function') {
      scroller.scrollTo({
        top: scroller.scrollHeight,
        behavior: 'auto',
      })
    } else {
      scroller.scrollTop = scroller.scrollHeight
    }
  })
}
</script>

<style scoped>
.chat-workspace {
  display: flex;
  width: 100%;
  height: 100%;
  min-height: 0;
  overflow: hidden;
  border: 1px solid var(--lx-clay-border);
  border-radius: 8px;
  color: var(--lx-clay-text);
  background: var(--lx-clay-surface);
  box-shadow: var(--lx-clay-shadow-form);
}

.chat-workspace__main {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex: 1;
  flex-direction: column;
  background: color-mix(in srgb, var(--lx-clay-canvas) 64%, var(--lx-clay-surface));
}

.chat-toolbar {
  display: flex;
  height: 64px;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex: 0 0 auto;
  border-bottom: 1px solid var(--lx-clay-border);
  padding: 0 24px;
  background: var(--lx-clay-surface-soft);
  backdrop-filter: blur(10px);
}

.chat-toolbar__primary {
  display: flex;
  min-width: 0;
  flex: 1;
  align-items: center;
  gap: 10px;
}

.chat-toolbar__history-button {
  display: none;
  place-items: center;
  width: 38px;
  height: 38px;
  flex: 0 0 38px;
  border: 1px solid var(--lx-clay-border);
  border-radius: 8px;
  color: var(--lx-clay-text-secondary);
  background: var(--lx-clay-surface);
}

.chat-toolbar__model {
  width: min(220px, 28vw);
  flex: 0 1 220px;
}

.chat-toolbar__divider {
  width: 1px;
  height: 18px;
  flex: 0 0 1px;
  background: var(--lx-clay-border);
}

.chat-toolbar__session {
  display: flex;
  min-width: 0;
  max-width: 360px;
  align-items: center;
  gap: 7px;
  color: var(--lx-clay-text-muted);
  font-size: 12px;
  font-weight: 650;
}

.chat-toolbar__session > span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chat-model-option {
  display: flex;
  min-width: 0;
  width: 100%;
  align-items: center;
  gap: 8px;
}

.chat-model-option--selected > span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chat-model-option__mark {
  display: grid;
  place-items: center;
  width: 26px;
  height: 26px;
  flex: 0 0 26px;
  border-radius: 7px;
  color: var(--lx-clay-accent);
  background: var(--lx-clay-accent-soft);
}

.chat-model-option__copy {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
}

.chat-model-option__copy strong,
.chat-model-option__copy small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chat-model-option__copy strong { font-size: 13px; }
.chat-model-option__copy small { color: var(--lx-clay-text-muted); font-size: 10px; }

.chat-model-option__recommended {
  border-radius: 6px;
  padding: 2px 5px;
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-accent-soft);
  font-size: 9px;
  font-weight: 800;
}

.chat-model-option__check {
  flex: 0 0 auto;
  color: var(--lx-clay-accent);
}

.chat-toolbar__balance {
  display: flex;
  min-width: 112px;
  min-height: 40px;
  align-items: center;
  gap: 8px;
  border: 1px solid var(--lx-clay-border);
  border-radius: 8px;
  padding: 5px 10px;
  color: var(--lx-clay-accent);
  background: var(--lx-clay-surface);
  text-decoration: none;
}

.chat-toolbar__balance > span {
  display: flex;
  min-width: 0;
  align-items: baseline;
  gap: 5px;
}

.chat-toolbar__balance small {
  color: var(--lx-clay-text-muted);
  font-size: 10px;
}

.chat-toolbar__balance strong {
  color: var(--lx-clay-text);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}

.chat-catalog-error {
  display: flex;
  align-items: center;
  gap: 12px;
  margin: 12px 16px 0;
  border: 1px solid color-mix(in srgb, var(--lx-clay-danger) 30%, transparent);
  border-radius: 8px;
  padding: 10px 12px;
  color: var(--lx-clay-danger);
  background: var(--lx-clay-danger-soft);
}

.chat-catalog-error > div {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  font-size: 12px;
}

.chat-catalog-error strong { font-size: 13px; }
.chat-catalog-error .btn { flex: 0 0 auto; }

.chat-persistence-warning {
  display: flex;
  align-items: center;
  gap: 12px;
  margin: 12px 16px 0;
  border: 1px solid color-mix(in srgb, var(--lx-clay-warning) 38%, var(--lx-clay-border));
  border-radius: 8px;
  padding: 10px 12px;
  color: var(--lx-clay-warning);
  background: var(--lx-clay-warning-soft);
}

.chat-persistence-warning > div {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  font-size: 12px;
}

.chat-persistence-warning strong { font-size: 13px; }
.chat-persistence-warning .btn { flex: 0 0 auto; }

.chat-messages-region {
  position: relative;
  display: flex;
  min-height: 0;
  flex: 1;
}

.chat-messages {
  width: 100%;
  min-height: 0;
  flex: 1;
  overflow-y: auto;
  padding: 32px 24px 10px;
  overscroll-behavior: contain;
}

.chat-messages__inner {
  display: flex;
  flex-direction: column;
  gap: 18px;
  padding-bottom: 16px;
}

.chat-messages__load-older {
  align-self: center;
  min-height: 36px;
  border: 0;
  border-radius: 8px;
  padding: 0 14px;
  color: var(--lx-clay-accent);
  background: var(--lx-clay-accent-soft);
  font-size: 12px;
  font-weight: 700;
}

.chat-messages__load-older:disabled {
  cursor: wait;
  opacity: 0.6;
}

.chat-scroll-to-latest {
  position: absolute;
  bottom: 12px;
  left: 50%;
  z-index: 1;
  display: grid;
  width: 40px;
  height: 40px;
  place-items: center;
  border: 1px solid var(--lx-clay-border-strong);
  border-radius: 50%;
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-surface-elevated);
  box-shadow: var(--lx-clay-shadow-surface);
  transform: translateX(-50%);
  transition:
    border-color 160ms ease,
    background-color 160ms ease,
    color 160ms ease,
    opacity 160ms ease,
    transform 160ms ease;
}

.chat-scroll-to-latest:hover {
  border-color: color-mix(in srgb, var(--lx-clay-accent) 34%, var(--lx-clay-border));
  color: var(--lx-clay-accent);
  background: var(--lx-clay-accent-soft);
}

.chat-scroll-control-enter-from,
.chat-scroll-control-leave-to {
  opacity: 0;
  transform: translate(-50%, 6px);
}

.chat-scroll-control-enter-active,
.chat-scroll-control-leave-active {
  transition:
    opacity 160ms ease,
    transform 160ms ease;
}

.chat-empty-state {
  display: flex;
  min-height: 100%;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  padding: 32px 20px;
  text-align: center;
}

.chat-empty-state__icon {
  display: grid;
  place-items: center;
  width: 80px;
  height: 80px;
  border: 1px solid color-mix(in srgb, var(--lx-clay-accent) 18%, transparent);
  border-radius: 8px;
  color: var(--lx-clay-accent);
  background: var(--lx-clay-accent-soft);
  box-shadow: var(--lx-clay-shadow-form);
}

.chat-empty-state h2 {
  margin: 24px 0 0;
  color: var(--lx-clay-text);
  font-family: var(--lx-clay-font-display);
  font-size: 24px;
  font-weight: 900;
  line-height: 1.25;
}

.chat-empty-state p {
  max-width: 480px;
  margin: 10px 0 0;
  color: var(--lx-clay-text-secondary);
  font-size: 14px;
  line-height: 1.7;
}

.chat-composer-region {
  flex: 0 0 auto;
  padding: 12px 24px 20px;
  background: transparent;
}

.chat-workspace__drawer {
  display: none;
}

.chat-workspace :deep(button:focus-visible),
.chat-workspace :deep(a:focus-visible) {
  outline: 2px solid var(--lx-clay-accent);
  outline-offset: 2px;
}

@media (max-width: 1023px) {
  .chat-workspace__history--desktop { display: none; }
  .chat-toolbar__history-button {
    display: grid;
    width: 44px;
    height: 44px;
    flex-basis: 44px;
  }

  .chat-workspace__drawer {
    position: fixed;
    inset: 81px 0 0;
    z-index: 45;
    display: flex;
  }

  .chat-workspace__scrim {
    position: absolute;
    inset: 0;
    border: 0;
    background: rgba(23, 19, 31, 0.34);
    backdrop-filter: blur(2px);
  }

  .chat-workspace__drawer :deep(.chat-history) {
    position: relative;
    z-index: 1;
    box-shadow: 18px 0 40px rgba(23, 19, 31, 0.18);
  }
}

@media (max-width: 700px) {
  .chat-toolbar {
    height: 64px;
    gap: 8px;
    padding: 0 10px;
  }

  .chat-toolbar__primary { gap: 8px; }
  .chat-toolbar__model { min-width: 0; width: auto; flex: 1; }
  .chat-toolbar__divider,
  .chat-toolbar__session { display: none; }
  .chat-toolbar__balance { min-width: 88px; padding-inline: 9px; }
  .chat-toolbar__balance small { display: none; }

  .chat-messages {
    padding: 14px 8px 6px;
  }

  .chat-scroll-to-latest { bottom: 8px; }

  .chat-composer-region {
    padding: 8px 8px max(7px, env(safe-area-inset-bottom));
  }

  .chat-catalog-error {
    align-items: flex-start;
    margin: 8px 8px 0;
  }

  .chat-catalog-error .btn span { display: none; }

  .chat-persistence-warning {
    align-items: flex-start;
    margin: 8px 8px 0;
  }

  .chat-persistence-warning .btn span { display: none; }
}

.chat-drawer-enter-active,
.chat-drawer-leave-active {
  transition: opacity 180ms ease;
}

.chat-drawer-enter-from,
.chat-drawer-leave-to {
  opacity: 0;
}

@media (prefers-reduced-motion: reduce) {
  .chat-drawer-enter-active,
  .chat-drawer-leave-active,
  .chat-scroll-control-enter-active,
  .chat-scroll-control-leave-active {
    transition: none;
  }
}
</style>
