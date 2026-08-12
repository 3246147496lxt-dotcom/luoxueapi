<template>
  <AppLayout variant="chat" shell-mode="chat">
    <h1 class="sr-only">{{ t('chat.title') }}</h1>
    <ChatFileDropOverlay :active="pageFileDragActive" />

    <div class="chat-workspace">
      <ChatHistoryPanel
        v-show="!narrowSidebar"
        shell
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
          v-if="historyOpen && mobileHistoryLayout"
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
            shell
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

      <WorkspaceSidebarOverlayLayer
        v-if="narrowSidebar"
        active
        :open="narrowSidebarOpen"
        :label="t('chat.history.title')"
        return-focus-id="workspace-sidebar-overlay-trigger"
        @close="closeHistory()"
      >
        <ChatHistoryPanel
          shell
          overlay
          sidebar-id="workspace-chat-sidebar-overlay"
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
      </WorkspaceSidebarOverlayLayer>

      <section
        ref="chatMainRef"
        class="chat-workspace__main"
        :class="{ 'chat-workspace__main--new-chat': isNewConversationHome }"
        :aria-label="t('chat.title')"
        :aria-hidden="historyModalActive ? 'true' : undefined"
        :inert="historyModalActive ? true : undefined"
        tabindex="-1"
      >
        <div class="chat-mobile-actions">
          <button
            ref="historyTriggerRef"
            type="button"
            class="chat-mobile-actions__history-button"
            :aria-label="t('chat.history.title')"
            :title="t('chat.history.title')"
            @click="openHistory"
          >
            <Icon name="menu" size="sm" />
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
          v-if="chatStore.persistenceAvailable && chatStore.syncStatus === 'offline'"
          class="chat-sync-notice"
          data-test="chat-sync-offline"
        >
          <Icon name="cloud" size="sm" aria-hidden="true" />
          <p role="status" aria-live="polite" aria-atomic="true">
            {{ t('chat.sync.offlineDescription') }}
          </p>
        </div>

        <div
          v-if="chatStore.persistenceAvailable && (historySyncRetrying || chatStore.syncStatus === 'error' || chatStore.syncStatus === 'unavailable')"
          class="chat-sync-notice"
          data-test="chat-sync-warning"
        >
          <Icon name="cloud" size="sm" aria-hidden="true" />
          <p role="status" aria-live="polite" aria-atomic="true">
            {{ t(historySyncRetrying ? 'chat.sync.syncing' : 'chat.sync.errorDescription') }}
          </p>
          <button
            type="button"
            class="btn btn-secondary"
            :disabled="historySyncRetrying"
            :aria-busy="historySyncRetrying ? 'true' : undefined"
            :aria-label="t(historySyncRetrying ? 'chat.sync.syncing' : 'chat.sync.retry')"
            :title="t(historySyncRetrying ? 'chat.sync.syncing' : 'chat.sync.retry')"
            data-test="chat-sync-retry"
            @click="retryServerHistory"
          >
            <Icon name="refresh" size="sm" aria-hidden="true" />
            <span>{{ t(historySyncRetrying ? 'chat.sync.syncing' : 'chat.sync.retry') }}</span>
          </button>
        </div>

        <p class="sr-only" aria-live="polite">{{ historySyncAnnouncement }}</p>

        <p class="sr-only" aria-live="polite">{{ streamAnnouncement }}</p>

        <div
          class="chat-conversation-flow"
          :class="{ 'chat-conversation-flow--new-chat': isNewConversationHome }"
          data-test="chat-conversation-flow"
        >
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
            <div v-if="isNewConversationHome" class="chat-empty-state" data-test="chat-new-chat-hero">
              <h2>{{ newChatGreeting }}</h2>
            </div>

            <div v-else class="chat-messages__inner">
              <button
                v-if="activeConversation?.messagesHasMore"
                type="button"
                class="chat-messages__load-older"
                :disabled="chatStore.loadingConversationMessages.has(activeConversation?.id ?? '')"
                @click="activeConversation && chatStore.loadOlderConversationMessages(activeConversation.id)"
              >
                {{ t('chat.history.loadOlderMessages') }}
              </button>
              <ChatMessageItem
                v-for="(message, index) in activeConversation?.messages ?? []"
                :key="message.id"
                :message="message"
                :retryable="canRetryMessage(message, index)"
                :retrying="retryPendingMessageId === message.id"
                :announce-failure="liveFailureMessageId === message.id"
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
            :submission-busy="voiceBusy || attachmentBusy"
            :has-attachments="attachmentDrafts.length > 0"
            :attachments-valid="attachmentValid"
            @send="sendMessage"
            @stop="stopStreaming"
          >
            <template #attachments>
              <ChatAttachmentPreviewList
                :items="attachmentDrafts"
                :supports-vision="selectedModelSupportsVision"
                @cancel="attachmentPickerRef?.cancel($event)"
                @retry="attachmentPickerRef?.retry($event)"
                @remove="attachmentPickerRef?.remove($event)"
              />
            </template>
            <template #leading>
              <ChatAttachmentPicker
                ref="attachmentPickerRef"
                :disabled="attachmentInputDisabled"
                :supports-vision="selectedModelSupportsVision"
                @change="attachmentDrafts = $event"
                @busy-change="attachmentBusy = $event"
                @valid-change="attachmentValid = $event"
              />
            </template>
            <template #trailing>
              <ChatModelSettings
                v-model="selectedModel"
                v-model:reasoning-effort="selectedReasoningEffort"
                :model-options="modelOptions"
                :disabled="chatStore.isStreaming || voiceBusy"
              />
              <ChatVoiceInput
                v-if="transcriptionEnabled"
                :key="voiceCapabilityKey"
                :context-key="voiceContextKey"
                :disabled="!chatReady || !selectedModelAvailable || chatStore.isStreaming"
                :max-duration-ms="voiceMaxDurationMs"
                :max-bytes="voiceMaxBytes"
                :accepted-mime-types="voiceAcceptedMimeTypes"
                @busy-change="voiceBusy = $event"
                @transcribed="insertVoiceTranscription"
              />
            </template>
            <template #empty-action>
              <ChatVoiceModeButton
                :available="false"
                :disabled="!chatReady || !selectedModelAvailable"
              />
            </template>
          </ChatComposer>
        </div>
        </div>
      </section>
    </div>

    <BaseDialog
      :show="confirmation !== null"
      :title="confirmation?.kind === 'clear' ? t('chat.confirm.clearTitle') : t('chat.confirm.deleteTitle')"
      width="narrow"
      :variant="confirmation?.kind === 'delete' ? 'workspace-confirm' : 'default'"
      :show-close-button="confirmation?.kind !== 'delete'"
      :description-id="confirmation?.kind === 'delete' ? 'chat-delete-confirm-description' : undefined"
      @close="confirmation = null"
    >
      <div
        v-if="confirmation?.kind === 'delete'"
        id="chat-delete-confirm-description"
        class="chat-delete-confirm__copy"
      >
        <p class="chat-delete-confirm__message">
          <span>{{ t('chat.confirm.deleteDescriptionPrefix') }}</span><strong
            :aria-label="confirmation.title"
            :title="confirmation.title"
          >{{ confirmation.displayTitle }}</strong><span>{{ t('chat.confirm.deleteDescriptionSuffix') }}</span>
        </p>
        <p class="chat-delete-confirm__memory"><span>{{ t('chat.confirm.memoryPrefix') }}</span><span class="chat-delete-confirm__settings-word">{{ t('chat.confirm.memorySettings') }}</span><span>{{ t('chat.confirm.memorySuffix') }}</span></p>
      </div>
      <p v-else class="text-sm leading-6 text-gray-600 dark:text-dark-300">
        {{ t('chat.confirm.clearDescription') }}
      </p>
      <template #footer>
        <template v-if="confirmation?.kind === 'delete'">
          <button
            type="button"
            class="chat-delete-confirm__button chat-delete-confirm__button--cancel"
            @click="confirmation = null"
          >
            {{ t('common.cancel') }}
          </button>
          <button
            type="button"
            class="chat-delete-confirm__button chat-delete-confirm__button--danger"
            @click="applyConfirmation"
          >
            {{ t('common.delete') }}
          </button>
        </template>
        <template v-else>
          <button type="button" class="btn btn-secondary" @click="confirmation = null">
            {{ t('common.cancel') }}
          </button>
          <button type="button" class="btn btn-danger" @click="applyConfirmation">
            {{ t('common.delete') }}
          </button>
        </template>
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
import WorkspaceSidebarOverlayLayer from '@/components/layout/WorkspaceSidebarOverlayLayer.vue'
import { useWorkspaceResponsiveState } from '@/components/layout/workspaceResponsive'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import ChatComposer from '@/components/chat/ChatComposer.vue'
import ChatAttachmentPicker from '@/components/chat/ChatAttachmentPicker.vue'
import ChatAttachmentPreviewList from '@/components/chat/ChatAttachmentPreviewList.vue'
import ChatFileDropOverlay from '@/components/chat/ChatFileDropOverlay.vue'
import type { ChatAttachmentDraft } from '@/components/chat/chatAttachmentUi'
import ChatHistoryPanel from '@/components/chat/ChatHistoryPanel.vue'
import ChatMessageItem from '@/components/chat/ChatMessageItem.vue'
import ChatVoiceInput from '@/components/chat/ChatVoiceInput.vue'
import ChatVoiceModeButton from '@/components/chat/ChatVoiceModeButton.vue'
import {
  CHAT_AUDIO_MAX_BYTES,
  CHAT_AUDIO_MAX_DURATION_MS,
} from '@/composables/useAudioRecorder'
import ChatModelSettings, {
  type ChatModelSettingsOption,
} from '@/components/chat/ChatModelSettings.vue'
import {
  ChatAPIError,
  createChatAttemptId,
  getChatCapabilities,
  isAbortError,
  pollChatReceipt,
  streamChatCompletion,
} from '@/api/chat'
import { pickChatGreeting } from '@/features/chat/chatGreetings'
import {
  describeChatError,
  describeChatMessageError,
  logChatCompletionError,
} from '@/features/chat/chatErrorHandler'
import { toChatConversationTitlePreview } from '@/features/chat/conversationTitle'
import {
  CHAT_PRODUCT_MODELS,
  DEFAULT_CHAT_PRODUCT_MODEL_ID,
} from '@/features/chat/chatProductModels'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { useChatStore, type ChatMessagePatch } from '@/stores/chat'
import type {
  ChatCompletionRequest,
  ChatAttempt,
  ChatMessage,
  ChatModel,
  ChatReasoningEffort,
  ChatReceipt,
  ChatTranscriptionCapability,
} from '@/types/chat'

type Confirmation =
  | { kind: 'delete'; conversationId: string; title: string; displayTitle: string }
  | { kind: 'clear' }

const SCROLL_FOLLOW_RESUME_DISTANCE = 8
const DEFAULT_REASONING_EFFORT: ChatReasoningEffort = 'low'
const NEW_CHAT_QUERY_KEY = 'conversation'
const NEW_CHAT_QUERY_VALUE = 'new'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const chatStore = useChatStore()

function hasNewChatRouteIntent(): boolean {
  if (typeof window === 'undefined') return false
  try {
    return new URL(window.location.href).searchParams.get(NEW_CHAT_QUERY_KEY)
      === NEW_CHAT_QUERY_VALUE
  } catch {
    return false
  }
}

function setNewChatRouteIntent(enabled: boolean): void {
  if (typeof window === 'undefined') return
  try {
    const url = new URL(window.location.href)
    if (enabled) {
      url.searchParams.set(NEW_CHAT_QUERY_KEY, NEW_CHAT_QUERY_VALUE)
    } else if (url.searchParams.get(NEW_CHAT_QUERY_KEY) === NEW_CHAT_QUERY_VALUE) {
      url.searchParams.delete(NEW_CHAT_QUERY_KEY)
    } else {
      return
    }
    window.history.replaceState(
      window.history.state,
      '',
      `${url.pathname}${url.search}${url.hash}`,
    )
  } catch {
    // IndexedDB remains the fallback when the current environment cannot replace the URL.
  }
}

const models: readonly ChatModel[] = CHAT_PRODUCT_MODELS
const transcriptionCapability = ref<ChatTranscriptionCapability | null>(null)
const defaultModel = ref(DEFAULT_CHAT_PRODUCT_MODEL_ID)
const persistenceRetrying = ref(false)
const historySyncRetrying = ref(false)
const historySyncAnnouncement = ref('')
const composerDraft = ref('')
const newChatGreeting = ref(pickChatGreeting())
const voiceBusy = ref(false)
const attachmentDrafts = ref<ChatAttachmentDraft[]>([])
const attachmentBusy = ref(false)
const attachmentValid = ref(true)
const pageFileDragActive = ref(false)
const selectedReasoningEffort = ref<ChatReasoningEffort>(DEFAULT_REASONING_EFFORT)
const historySearchQuery = ref('')
const legacyImportPromptVisible = ref(true)
const historyOpen = ref(false)
const {
  mobileDrawer: mobileHistoryLayout,
  narrowSidebar,
} = useWorkspaceResponsiveState()
const narrowSidebarOpen = computed(() => appStore.workspaceNarrowSidebarOpen)
const confirmation = ref<Confirmation | null>(null)
const messageScrollerRef = ref<HTMLElement | null>(null)
const composerRef = ref<InstanceType<typeof ChatComposer> | null>(null)
const attachmentPickerRef = ref<InstanceType<typeof ChatAttachmentPicker> | null>(null)
const historyDrawerRef = ref<HTMLElement | null>(null)
const historyTriggerRef = ref<HTMLButtonElement | null>(null)
const chatMainRef = ref<HTMLElement | null>(null)
const shouldFollowStream = ref(true)
const isAwayFromLatest = ref(false)
const retryPendingMessageId = ref<string | null>(null)
const liveFailureMessageId = ref<string | null>(null)
let scrollFrame = 0
let historySearchTimer: ReturnType<typeof setTimeout> | null = null
let lastMessageScrollTop = 0
let lastMessageTouchY: number | null = null
let observedAuthUserId: string | undefined
let viewDisposed = false
let attachmentContextChangeOwnedBySend = false
let pageFileDragDepth = 0
let messageResizeObserver: ResizeObserver | null = null
let capabilitiesController: AbortController | null = null
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
const transcriptionEnabled = computed(() => transcriptionCapability.value?.enabled === true)
const voiceContextKey = computed(() => (
  `${currentAuthUserId()}:${chatStore.activeConversationId ?? 'new'}`
))
const voiceMaxDurationMs = computed(() => Math.min(
  CHAT_AUDIO_MAX_DURATION_MS,
  (transcriptionCapability.value?.max_duration_seconds ?? 60) * 1_000,
))
const voiceMaxBytes = computed(() => Math.min(
  CHAT_AUDIO_MAX_BYTES,
  transcriptionCapability.value?.max_upload_bytes ?? CHAT_AUDIO_MAX_BYTES,
))
const voiceAcceptedMimeTypes = computed(() => (
  transcriptionCapability.value?.accepted_mime_types ?? []
))
const voiceCapabilityKey = computed(() => [
  voiceMaxDurationMs.value,
  voiceMaxBytes.value,
  ...voiceAcceptedMimeTypes.value,
].join(':'))
const historyModalActive = computed(() => (
  (historyOpen.value && mobileHistoryLayout.value)
  || (narrowSidebar.value && narrowSidebarOpen.value)
))
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
const modelOptions = computed<ChatModelSettingsOption[]>(() => models.map((model) => {
  const label = model.display_name?.trim() || model.id
  const description = model.description?.trim() || ''
  return {
    value: model.id,
    label,
    description: description === label || description === model.id ? '' : description,
    recommended: model.recommended === true,
    supportsReasoningSlider: true,
  }
}))
const availableModelIds = new Set(models.map((model) => model.id))
const preferredModel = DEFAULT_CHAT_PRODUCT_MODEL_ID

const selectedModel = computed<string>({
  get: () => activeConversation.value?.model || defaultModel.value,
  set: (value) => {
    if (!value || !availableModelIds.has(value)) return
    defaultModel.value = value
    if (activeConversation.value) {
      chatStore.setConversationModel(activeConversation.value.id, value)
    }
  },
})
const selectedModelAvailable = computed(() => (
  selectedModel.value.trim().length > 0
))
const selectedModelSupportsVision = computed(() => (
  models.find((model) => model.id === selectedModel.value)?.supports_vision === true
))
const attachmentInputDisabled = computed(() => (
  !chatReady.value
  || !selectedModelAvailable.value
  || chatStore.isStreaming
))
const attachmentDropEnabled = computed(() => (
  !attachmentInputDisabled.value
  && !historyModalActive.value
  && confirmation.value === null
  && !(legacyImportPromptVisible.value && chatStore.legacyImportRequired)
))

const isNewConversationHome = computed(() => {
  const conversation = activeConversation.value
  if (!conversation) return true
  return conversation.messages.length === 0 && (conversation.messageCount ?? 0) === 0
})
const emptyComposerReady = computed(() => (
  chatReady.value
  && selectedModelAvailable.value
  && isNewConversationHome.value
))
const showScrollToLatest = computed(() => (
  !shouldFollowStream.value
  && isAwayFromLatest.value
  && (activeConversation.value?.messages.length ?? 0) > 0
))

watch(
  () => authStore.user?.id,
  (userId) => {
    const normalizedUserId = normalizeUserId(userId)
    abortReceiptPolls()
    if (observedAuthUserId !== undefined && observedAuthUserId !== normalizedUserId) {
      void discardPendingAttachments()
      composerDraft.value = ''
      selectedReasoningEffort.value = DEFAULT_REASONING_EFFORT
      historySearchQuery.value = ''
    }
    observedAuthUserId = normalizedUserId
    historySyncAnnouncement.value = ''
    legacyImportPromptVisible.value = true
    resetChatProductState()
    const hydration = chatStore.hydrate(userId)
    if (normalizedUserId && hasNewChatRouteIntent()) {
      chatStore.selectConversation(null)
    }
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
  },
  { immediate: true, flush: 'sync' },
)

watch(
  [
    chatReady,
    () => chatStore.activeConversationId,
    () => activeConversation.value?.model,
  ],
  () => reconcileSelectedModel(),
  { flush: 'sync' },
)

watch(emptyComposerReady, (ready) => {
  if (ready) void focusComposer()
}, { immediate: true })

watch(attachmentDropEnabled, (enabled) => {
  if (!enabled) resetPageFileDrag()
}, { flush: 'sync' })

watch(
  () => activeConversation.value?.messages.map((message) => `${message.id}:${message.content.length}:${message.status}`).join('|'),
  () => scheduleScrollToBottom(),
)

watch(() => chatStore.activeConversationId, (conversationId) => {
  if (conversationId !== null) setNewChatRouteIntent(false)
}, { flush: 'sync' })

watch(() => chatStore.activeConversationId, () => {
  liveFailureMessageId.value = null
  if (attachmentContextChangeOwnedBySend) {
    attachmentContextChangeOwnedBySend = false
  } else {
    void discardPendingAttachments()
  }
  if (historyModalActive.value) void closeHistory()
  shouldFollowStream.value = true
  isAwayFromLatest.value = false
  lastMessageScrollTop = 0
  scheduleScrollToBottom()
})

watch(mobileHistoryLayout, (mobile) => {
  if (!mobile && historyOpen.value) void closeHistory(chatMainRef.value)
})

onMounted(() => {
  void loadChatCapabilities()
  window.addEventListener('online', syncServerHistory)
  window.addEventListener('dragenter', onPageFileDragEnter, true)
  window.addEventListener('dragover', onPageFileDragOver, true)
  window.addEventListener('dragleave', onPageFileDragLeave, true)
  window.addEventListener('drop', onPageFileDrop, true)
  window.addEventListener('dragend', resetPageFileDrag, true)
  window.addEventListener('blur', resetPageFileDrag)
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
})

onBeforeUnmount(() => {
  viewDisposed = true
  capabilitiesController?.abort()
  capabilitiesController = null
  void discardPendingAttachments()
  cancelPendingMessageScroll()
  abortReceiptPolls()
  if (historySearchTimer !== null) clearTimeout(historySearchTimer)
  window.removeEventListener('online', syncServerHistory)
  window.removeEventListener('dragenter', onPageFileDragEnter, true)
  window.removeEventListener('dragover', onPageFileDragOver, true)
  window.removeEventListener('dragleave', onPageFileDragLeave, true)
  window.removeEventListener('drop', onPageFileDrop, true)
  window.removeEventListener('dragend', resetPageFileDrag, true)
  window.removeEventListener('blur', resetPageFileDrag)
  resetPageFileDrag()
  messageResizeObserver?.disconnect()
  if (chatStore.isStreaming) chatStore.stopStreaming()
})

async function loadChatCapabilities() {
  capabilitiesController?.abort()
  const controller = new AbortController()
  capabilitiesController = controller
  try {
    const capabilities = await getChatCapabilities(controller.signal)
    if (!viewDisposed && capabilitiesController === controller) {
      transcriptionCapability.value = capabilities.transcription ?? null
    }
  } catch (error) {
    if (!isAbortError(error) && !viewDisposed && capabilitiesController === controller) {
      transcriptionCapability.value = null
    }
  } finally {
    if (capabilitiesController === controller) capabilitiesController = null
  }
}

function currentAuthUserId() {
  return normalizeUserId(authStore.user?.id)
}

function normalizeUserId(id: string | number | null | undefined) {
  return id === null || id === undefined ? '' : String(id)
}

function resetChatProductState() {
  defaultModel.value = DEFAULT_CHAT_PRODUCT_MODEL_ID
}

function reconcileSelectedModel() {
  if (!availableModelIds.has(defaultModel.value)) defaultModel.value = preferredModel
}

async function focusComposer() {
  await nextTick()
  composerRef.value?.focus()
}

function insertVoiceTranscription(text: string, acknowledge?: (inserted: boolean) => void) {
  const inserted = composerRef.value?.insertText?.(text) === true
  acknowledge?.(inserted)
}

function isPageFileDrag(event: DragEvent): boolean {
  const transfer = event.dataTransfer
  if (!transfer) return false
  return Array.from(transfer.types ?? []).includes('Files') || transfer.files.length > 0
}

function setPageFileDropEffect(event: DragEvent): void {
  if (!event.dataTransfer) return
  event.dataTransfer.dropEffect = attachmentDropEnabled.value ? 'copy' : 'none'
}

function resetPageFileDrag(): void {
  pageFileDragDepth = 0
  pageFileDragActive.value = false
}

function onPageFileDragEnter(event: DragEvent): void {
  if (!isPageFileDrag(event)) return
  event.preventDefault()
  setPageFileDropEffect(event)
  if (!attachmentDropEnabled.value) return
  pageFileDragDepth += 1
  pageFileDragActive.value = true
}

function onPageFileDragOver(event: DragEvent): void {
  if (!isPageFileDrag(event)) return
  event.preventDefault()
  setPageFileDropEffect(event)
}

function onPageFileDragLeave(event: DragEvent): void {
  if (!pageFileDragActive.value) return
  const leftViewport = event.relatedTarget === null
    && (event.target === document.body || event.target === document.documentElement)
  if (leftViewport) {
    resetPageFileDrag()
    return
  }
  pageFileDragDepth = Math.max(0, pageFileDragDepth - 1)
  if (pageFileDragDepth === 0) pageFileDragActive.value = false
}

function onPageFileDrop(event: DragEvent): void {
  if (!isPageFileDrag(event)) return
  event.preventDefault()
  const files = Array.from(event.dataTransfer?.files ?? [])
  const accepted = attachmentDropEnabled.value
  resetPageFileDrag()
  if (accepted && files.length > 0) attachmentPickerRef.value?.addFiles(files)
}

async function discardPendingAttachments(): Promise<void> {
  const picker = attachmentPickerRef.value
  attachmentDrafts.value = []
  attachmentBusy.value = false
  attachmentValid.value = true
  if (picker) await picker.discardAll()
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

async function initializeServerHistory(expectedUserId: string): Promise<boolean> {
  await chatStore.syncHistory()
  if (viewDisposed || expectedUserId !== currentAuthUserId()) return false
  if (chatStore.syncStatus !== 'idle' || chatStore.serverHistoryAvailable !== true) return false
  await chatStore.loadConversationPage(true)
  if (viewDisposed || expectedUserId !== currentAuthUserId()) return false
  if (chatStore.syncStatus !== 'idle' || chatStore.serverHistoryAvailable !== true) return false
  await recoverInterruptedAttempts(expectedUserId)
  resumePendingReceipts(expectedUserId)
  return true
}

async function syncServerHistory(): Promise<boolean> {
  const expectedUserId = currentAuthUserId()
  if (!expectedUserId) return false
  return initializeServerHistory(expectedUserId)
}

async function retryServerHistory() {
  if (historySyncRetrying.value) return
  historySyncRetrying.value = true
  historySyncAnnouncement.value = ''
  try {
    if (await syncServerHistory()) {
      historySyncAnnouncement.value = t('chat.sync.success')
      await focusComposer()
    }
  } finally {
    historySyncRetrying.value = false
  }
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
  newChatGreeting.value = pickChatGreeting(newChatGreeting.value)
  defaultModel.value = preferredModel
  setNewChatRouteIntent(true)
  chatStore.selectConversation(null)
  reconcileSelectedModel()
  composerDraft.value = ''
  if (historyModalActive.value) void closeHistory()
  void focusComposer()
}

function selectConversation(id: string) {
  if (chatStore.isStreaming && chatStore.streamingConversationId !== id) {
    stopStreaming()
  }
  if (!chatStore.selectConversation(id)) return
  void chatStore.loadConversationDetail(id)
  reconcileSelectedModel()
  if (historyModalActive.value) void closeHistory()
}

async function openHistory() {
  if (!mobileHistoryLayout.value) return
  historyOpen.value = true
  await nextTick()
  const firstControl = historyDrawerRef.value?.querySelector<HTMLElement>(
    'a[href]:not([tabindex="-1"]), button:not([disabled]):not([tabindex="-1"]), input:not([disabled]), [tabindex]:not([tabindex="-1"])',
  )
  ;(firstControl ?? historyDrawerRef.value)?.focus()
}

async function closeHistory(focusTarget: HTMLElement | null = historyTriggerRef.value) {
  const mobileWasOpen = historyOpen.value
  const narrowWasOpen = narrowSidebarOpen.value
  if (!mobileWasOpen && !narrowWasOpen) return
  if (mobileWasOpen) historyOpen.value = false
  if (narrowWasOpen) appStore.setWorkspaceNarrowSidebarOpen(false)
  await nextTick()
  if (mobileWasOpen) focusTarget?.focus()
}

function onHistoryDrawerKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    event.preventDefault()
    void closeHistory()
    return
  }
  if (event.key !== 'Tab') return

  const controls = Array.from(historyDrawerRef.value?.querySelectorAll<HTMLElement>(
    'a[href]:not([tabindex="-1"]), button:not([disabled]):not([tabindex="-1"]), input:not([disabled]), [tabindex]:not([tabindex="-1"])',
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

async function confirmDelete(id: string) {
  const conversation = historyConversations.value.find((item) => item.id === id)
    ?? chatStore.conversations.find((item) => item.id === id)
  const title = conversation?.title.trim() || t('chat.actions.newChat')
  if (historyModalActive.value) await closeHistory()
  confirmation.value = {
    kind: 'delete',
    conversationId: id,
    title,
    displayTitle: toChatConversationTitlePreview(title),
  }
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

async function sendMessage(
  content: string,
  acknowledge?: (accepted: boolean) => void,
) {
  const attachments = attachmentPickerRef.value?.getReadyAttachments() ?? []
  if (
    !chatReady.value
    || !selectedModelAvailable.value
    || chatStore.isStreaming
    || voiceBusy.value
    || attachmentBusy.value
    || !attachmentValid.value
    || (attachmentDrafts.value.length > 0 && attachments.length !== attachmentDrafts.value.length)
    || (!content.trim() && attachments.length === 0)
  ) {
    acknowledge?.(false)
    return
  }
  const requestModel = selectedModel.value
  const titleSource = content.trim() || attachments[0]?.name || t('chat.history.newConversation')
  liveFailureMessageId.value = null

  let conversation = activeConversation.value
  if (!conversation) {
    attachmentContextChangeOwnedBySend = true
    conversation = chatStore.createConversation(requestModel, buildTitle(titleSource))
  }
  if (!conversation) {
    attachmentContextChangeOwnedBySend = false
    acknowledge?.(false)
    return
  }

  if (conversation.messages.length === 0) {
    chatStore.renameConversation(conversation.id, buildTitle(titleSource))
  }
  chatStore.setConversationModel(conversation.id, requestModel)
  const expectedUserId = currentAuthUserId()
  const completionReady = await chatStore.prepareConversationForCompletion(conversation.id)
  if (
    !completionReady
    || viewDisposed
    || expectedUserId !== currentAuthUserId()
    || activeConversation.value?.id !== conversation.id
  ) {
    acknowledge?.(false)
    return
  }
  const expectedHeadMessageId = conversation.headMessageId
    ?? conversation.messages[conversation.messages.length - 1]?.id
    ?? null
  const userMessage = chatStore.addMessage(
    conversation.id,
    { role: 'user', content, attachments, status: 'complete' },
  )
  if (!userMessage) {
    acknowledge?.(false)
    return
  }
  const attemptId = createChatAttemptId()
  const assistant = chatStore.addMessage(conversation.id, {
    role: 'assistant',
    content: '',
    status: 'streaming',
    attemptId,
    requestedModel: requestModel,
  })
  if (!assistant) {
    chatStore.removeMessages(conversation.id, [userMessage.id])
    acknowledge?.(false)
    return
  }

  if (attachments.length === 0) acknowledge?.(true)

  const streamResult = await runStream({
    conversationId: conversation.id,
    model: requestModel,
    reasoningEffort: selectedReasoningEffort.value,
    expectedHeadMessageId,
    userMessage: {
      id: userMessage.id,
      content: userMessage.content,
      ...(attachments.length > 0
        ? { attachmentIds: attachments.map(({ id }) => id) }
        : {}),
    },
    assistantMessageId: assistant.id,
  }, attemptId, attachments.length > 0
    ? () => {
        attachmentPickerRef.value?.commitAll()
        acknowledge?.(true)
      }
    : undefined)

  if (attachments.length > 0 && !streamResult.accepted) {
    if (!streamResult.keepMessages) {
      chatStore.removeMessages(conversation.id, [userMessage.id, assistant.id])
    }
    acknowledge?.(false)
  }
}

function canRetryMessage(message: ChatMessage, index: number) {
  const retryableState = message.status === 'error'
    ? describeChatMessageError(message).retryable
    : message.status === 'complete' || message.status === 'stopped'
  return message.role === 'assistant'
    && retryableState
    && !message.excludedFromContext
    && index === (activeConversation.value?.messages.length ?? 0) - 1
    && !chatStore.isStreaming
    && retryPendingMessageId.value === null
    && selectedModelAvailable.value
}

async function retryMessage(messageId: string) {
  const conversation = activeConversation.value
  if (
    !conversation
    || !selectedModelAvailable.value
    || chatStore.isStreaming
    || retryPendingMessageId.value !== null
  ) return
  const requestModel = selectedModel.value
  const index = conversation.messages.findIndex((message) => message.id === messageId)
  if (index < 0) return

  retryPendingMessageId.value = messageId
  liveFailureMessageId.value = null
  try {
    const expectedUserId = currentAuthUserId()
    if (!await chatStore.prepareConversationForCompletion(conversation.id)) return
    if (
      viewDisposed
      || expectedUserId !== currentAuthUserId()
      || activeConversation.value?.id !== conversation.id
    ) return
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
    let replacementAccepted = false
    await runStream({
      conversationId: conversation.id,
      model: requestModel,
      reasoningEffort: selectedReasoningEffort.value,
      expectedHeadMessageId,
      assistantMessageId: replacement.id,
      retryOfMessageId: messageId,
    }, attemptId, () => {
      if (replacementAccepted) return
      replacementAccepted = true
      chatStore.updateMessage(conversation.id, messageId, {
        excludedFromContext: true,
        supersededByMessageId: replacement.id,
      })
    })

    if (!replacementAccepted) {
      chatStore.removeMessages(conversation.id, [replacement.id])
    }
  } finally {
    if (retryPendingMessageId.value === messageId) {
      retryPendingMessageId.value = null
    }
  }
}

async function runStream(
  request: ChatCompletionRequest,
  attemptId: string,
  onAccepted?: () => void,
): Promise<{ accepted: boolean; keepMessages: boolean }> {
  const conversationId = request.conversationId
  const assistantMessageId = request.assistantMessageId
  const streamUserId = currentAuthUserId()
  const controller = new AbortController()
  if (!chatStore.startStreaming(conversationId, assistantMessageId, controller)) {
    return { accepted: false, keepMessages: false }
  }
  let accepted = false
  const acceptStream = () => {
    if (accepted) return
    accepted = true
    chatStore.markCompletionAccepted(conversationId, assistantMessageId)
    onAccepted?.()
  }
  shouldFollowStream.value = true
  scheduleScrollToBottom()

  try {
    const result = await streamChatCompletion(
      request,
      {
        onAccepted: () => {
          acceptStream()
        },
        onReceiptId: (receiptId) => {
          recordPendingReceipt(conversationId, assistantMessageId, receiptId)
        },
        onContent: (content) => {
          chatStore.appendStreamingContent(conversationId, assistantMessageId, content)
        },
      },
      { signal: controller.signal, attemptId },
    )
    acceptStream()
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
      acceptStream()
      recordPendingReceipt(conversationId, assistantMessageId, duplicateReceiptId)
    }
    if (isAbortError(error)) {
      const receiptId = findMessage(conversationId, assistantMessageId)?.receiptId
      if (receiptId && streamUserId) {
        void syncChatReceipt(conversationId, assistantMessageId, receiptId, streamUserId)
      }
      if (streamUserId) {
        const recoveredAttempt = await recoverStreamAttempt(
          conversationId,
          assistantMessageId,
          streamUserId,
        )
        if (recoveredAttempt && !accepted) {
          acceptStream()
        }
      }
      return { accepted, keepMessages: accepted }
    }
    const presentation = describeChatError(error)
    if (!duplicateReceiptId) {
      logChatCompletionError(error, {
        conversationId,
        messageId: assistantMessageId,
        attemptId,
      }, presentation)
    }
    chatStore.failStreaming(
      conversationId,
      assistantMessageId,
      t(presentation.messageKey),
      presentation.code,
    )
    liveFailureMessageId.value = assistantMessageId
    const receiptId = duplicateReceiptId
      || findMessage(conversationId, assistantMessageId)?.receiptId
    if (receiptId && streamUserId) {
      void syncChatReceipt(conversationId, assistantMessageId, receiptId, streamUserId)
    }
    if (streamUserId) {
      const recoveredAttempt = await recoverStreamAttempt(
        conversationId,
        assistantMessageId,
        streamUserId,
      )
      if (recoveredAttempt && !accepted) {
        acceptStream()
      }
    }
  }
  return { accepted, keepMessages: true }
}

async function recoverStreamAttempt(
  conversationId: string,
  assistantMessageId: string,
  expectedUserId: string,
): Promise<ChatAttempt | null> {
  const attempt = await chatStore.recoverAttempt(conversationId, assistantMessageId)
  if (
    !attempt
    || viewDisposed
    || expectedUserId !== currentAuthUserId()
  ) {
    return null
  }
  if (!attempt.receiptId) {
    return attempt
  }
  recordPendingReceipt(conversationId, assistantMessageId, attempt.receiptId)
  void syncChatReceipt(
    conversationId,
    assistantMessageId,
    attempt.receiptId,
    expectedUserId,
  )
  return attempt
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
  border: 0;
  border-radius: 0;
  color: var(--lx-clay-text);
  background: var(--workspace-canvas);
  box-shadow: none;
}

.chat-workspace__main {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex: 1;
  flex-direction: column;
  background: var(--workspace-canvas);
}

.chat-mobile-actions {
  display: none;
  align-items: center;
  flex: 0 0 auto;
}

.chat-mobile-actions__history-button {
  display: grid;
  place-items: center;
  width: 44px;
  height: 44px;
  flex: 0 0 44px;
  border: 1px solid var(--workspace-border);
  border-radius: 8px;
  color: var(--lx-clay-text-secondary);
  background: var(--lx-clay-surface);
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
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
}

.chat-catalog-error strong {
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
}
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
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
}

.chat-persistence-warning strong {
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
}
.chat-persistence-warning .btn { flex: 0 0 auto; }

.chat-sync-notice {
  display: flex;
  align-items: center;
  gap: 10px;
  margin: 8px 16px 0;
  border: 1px solid var(--workspace-divider);
  border-radius: 8px;
  padding: 7px 10px;
  color: var(--workspace-text-secondary);
  background: color-mix(in srgb, var(--workspace-surface) 94%, var(--lx-clay-accent));
}

.chat-sync-notice p {
  min-width: 0;
  flex: 1;
  margin: 0;
  overflow-wrap: anywhere;
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
  line-height: 1.5;
}

.chat-sync-notice .btn { flex: 0 0 auto; }

.chat-conversation-flow {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex: 1;
  flex-direction: column;
}

.chat-conversation-flow--new-chat {
  overflow-x: hidden;
  overflow-y: auto;
  overscroll-behavior: contain;
  animation: chat-new-chat-enter 400ms cubic-bezier(0, 0, 0.2, 1) both;
}

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
  padding: 52px 24px 10px;
  overscroll-behavior: contain;
}

.chat-messages__inner {
  display: flex;
  flex-direction: column;
  gap: 28px;
  padding-bottom: 16px;
}

.chat-messages__load-older {
  align-self: center;
  min-height: 36px;
  border: 1px solid var(--workspace-border);
  border-radius: 8px;
  padding: 0 14px;
  color: var(--workspace-text-secondary);
  background: var(--workspace-surface);
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
}

.chat-messages__load-older:hover:not(:disabled) {
  background: var(--workspace-surface-subtle);
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
  border: 1px solid var(--workspace-border);
  border-radius: 50%;
  color: var(--workspace-text-secondary);
  background: var(--workspace-surface);
  box-shadow: 0 4px 14px rgb(17 24 39 / 0.08);
  transform: translateX(-50%);
  transition:
    border-color 160ms ease,
    background-color 160ms ease,
    color 160ms ease,
    opacity 160ms ease,
    transform 160ms ease;
}

.chat-scroll-to-latest:hover {
  border-color: var(--workspace-border-strong);
  color: var(--workspace-text);
  background: var(--workspace-surface-subtle);
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

.chat-empty-state h2 {
  display: inline-flex;
  min-height: 42px;
  align-items: baseline;
  margin: 0;
  color: var(--workspace-text);
  font-size: 24px;
  font-weight: 400;
  letter-spacing: normal;
  line-height: 28px;
  text-wrap: balance;
}

.chat-composer-region {
  flex: 0 0 auto;
  padding: 12px 24px 20px;
  background: transparent;
}

.chat-conversation-flow--new-chat .chat-messages-region {
  min-height: 232px;
  flex: 0 0 max(232px, 42svh);
}

.chat-conversation-flow--new-chat .chat-messages {
  overflow: visible;
  padding: 0;
}

.chat-conversation-flow--new-chat .chat-empty-state {
  min-height: 100%;
  justify-content: flex-end;
  padding: 24px 20px 22px;
}

.chat-conversation-flow--new-chat .chat-composer-region {
  padding: 0 24px 48px;
}

@keyframes chat-new-chat-enter {
  from {
    opacity: 0;
    transform: translateY(10px);
  }

  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.chat-workspace__drawer {
  display: none;
}

.chat-delete-confirm__message,
.chat-delete-confirm__memory {
  margin: 0;
}

.chat-delete-confirm__message {
  color: var(--workspace-confirm-text);
  font-size: 16px;
  font-weight: var(--workspace-type-body-weight);
  line-height: 24px;
}

.chat-delete-confirm__message strong {
  font-weight: 700;
}

.chat-delete-confirm__memory {
  margin-top: var(--workspace-space-2);
  color: var(--workspace-confirm-text-secondary);
  font-size: var(--workspace-type-body-size);
  font-weight: var(--workspace-type-body-weight);
  line-height: 20px;
}

.chat-delete-confirm__settings-word {
  text-decoration: underline;
  text-underline-offset: 2px;
}

.chat-delete-confirm__button {
  display: inline-flex;
  height: 36px;
  align-items: center;
  justify-content: center;
  appearance: none;
  padding: 0 var(--workspace-space-3);
  border: 1px solid transparent;
  border-radius: var(--workspace-radius-pill);
  font: inherit;
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
  line-height: 20px;
  cursor: pointer;
  transition: none;
}

.chat-delete-confirm__button--cancel {
  border-color: var(--workspace-confirm-cancel-border);
  color: var(--workspace-confirm-text);
  background: var(--workspace-confirm-surface);
}

.chat-delete-confirm__button--danger {
  color: var(--lx-clay-on-accent);
  background: var(--workspace-confirm-danger);
}

.chat-delete-confirm__button:focus-visible {
  outline: 1.5px solid var(--workspace-confirm-text);
  outline-offset: 2.5px;
}

@media (hover: hover) and (pointer: fine) {
  .chat-delete-confirm__button--cancel:hover {
    background: var(--workspace-confirm-cancel-hover);
  }

  .chat-delete-confirm__button--danger:hover {
    background: var(--workspace-confirm-danger-hover);
  }
}

.chat-workspace :deep(button:focus-visible),
.chat-workspace :deep(a:focus-visible) {
  outline: 2px solid var(--lx-clay-accent);
  outline-offset: 2px;
}

@media (max-width: 767px) and (hover: none) and (pointer: coarse) {
  .chat-workspace__history--desktop { display: none; }

  .chat-mobile-actions {
    display: flex;
    height: 56px;
    padding: 0 10px;
  }

  .chat-conversation-flow--new-chat .chat-messages-region {
    min-height: 176px;
    flex-basis: max(176px, calc(42svh - 56px));
  }

  .chat-workspace__drawer {
    position: fixed;
    inset: 0;
    z-index: 45;
    display: flex;
  }

  .chat-workspace__scrim {
    position: absolute;
    inset: 0;
    border: 0;
    background: rgb(15 23 42 / 0.36);
    backdrop-filter: blur(2px);
  }

  .chat-workspace__drawer :deep(.chat-history) {
    position: relative;
    z-index: 1;
    box-shadow: 18px 0 40px rgb(15 23 42 / 0.18);
  }
}

@media (max-width: 700px) {
  .chat-messages {
    padding: 24px 8px 6px;
  }

  .chat-messages__inner {
    gap: 24px;
  }

  .chat-scroll-to-latest { bottom: 8px; }

  .chat-composer-region {
    padding: 8px 16px max(7px, env(safe-area-inset-bottom));
  }

  .chat-conversation-flow--new-chat .chat-messages-region {
    min-height: 148px;
    flex-basis: max(148px, calc(42svh - 56px));
  }

  .chat-conversation-flow--new-chat .chat-empty-state {
    padding: 20px 16px 22px;
  }

  .chat-conversation-flow--new-chat .chat-composer-region {
    padding: 0 16px max(24px, env(safe-area-inset-bottom));
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

  .chat-sync-notice {
    align-items: flex-start;
    margin: 8px 8px 0;
  }

  .chat-sync-notice .btn span { display: none; }
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
  .chat-conversation-flow--new-chat {
    animation: none;
  }

  .chat-drawer-enter-active,
  .chat-drawer-leave-active,
  .chat-scroll-control-enter-active,
  .chat-scroll-control-leave-active {
    transition: none;
  }
}
</style>
