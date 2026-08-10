import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { nextTick, ref } from 'vue'
import { createPinia } from 'pinia'
import { CHAT_GREETINGS } from '@/features/chat/chatGreetings'
import { toChatConversationTitlePreview } from '@/features/chat/conversationTitle'
import {
  WORKSPACE_MOBILE_DRAWER_MEDIA_QUERY,
  WORKSPACE_NARROW_SIDEBAR_MEDIA_QUERY,
} from '@/components/layout/workspaceResponsive'
import type { ChatAttachment, ChatConversation, ChatMessage } from '@/types/chat'

const apiMocks = vi.hoisted(() => ({
  createChatAttemptId: vi.fn(),
  createChatIdempotencyKey: vi.fn(),
  getChatCapabilities: vi.fn(),
  getChatModels: vi.fn(),
  isAbortError: vi.fn(),
  pollChatReceipt: vi.fn(),
  streamChatCompletion: vi.fn(),
  transcribeChatAudio: vi.fn(),
}))

vi.mock('@/api/chat', () => {
  class MockChatAPIError extends Error {
    status: number
    code: string | number | null
    metadata: unknown

    constructor(
      message: string,
      options: {
        status?: number
        code?: string | number | null
        metadata?: unknown
      } = {},
    ) {
      super(message)
      this.name = 'ChatAPIError'
      this.status = options.status ?? 0
      this.code = options.code ?? null
      this.metadata = options.metadata
    }
  }

  return {
    ChatAPIError: MockChatAPIError,
    createChatAttemptId: apiMocks.createChatAttemptId,
    createChatIdempotencyKey: apiMocks.createChatIdempotencyKey,
    getChatCapabilities: apiMocks.getChatCapabilities,
    getChatModels: apiMocks.getChatModels,
    isAbortError: apiMocks.isAbortError,
    pollChatReceipt: apiMocks.pollChatReceipt,
    streamChatCompletion: apiMocks.streamChatCompletion,
    transcribeChatAudio: apiMocks.transcribeChatAudio,
  }
})

vi.mock('@/stores/auth', async () => {
  const { reactive } = await vi.importActual<typeof import('vue')>('vue')
  const store = reactive({
    user: { id: 7, balance: 10 },
    refreshUser: vi.fn(),
  })
  return { useAuthStore: () => store }
})

vi.mock('@/stores/chat', async () => {
  const { reactive } = await vi.importActual<typeof import('vue')>('vue')
  const store = reactive({
    userId: null as string | null,
    conversations: [] as ChatConversation[],
    activeConversationId: null as string | null,
    activeConversation: null as ChatConversation | null,
    hydrated: false,
    hydrating: false,
    persistenceAvailable: true,
    syncStatus: 'idle' as const,
    serverHistoryAvailable: true as boolean | null,
    legacyImportRequired: false,
    legacyConversationIds: [] as string[],
    searchResults: [] as ChatConversation[],
    conversationsHaveMore: false,
    loadingConversationPage: false,
    searchingHistory: false,
    loadingConversationMessages: new Set<string>(),
    isStreaming: false,
    streamingConversationId: null as string | null,
    streamingMessageId: null as string | null,
    hydrate: vi.fn(),
    flushPersistence: vi.fn(),
    syncHistory: vi.fn(),
    loadConversationPage: vi.fn(),
    searchHistory: vi.fn(),
    loadConversationDetail: vi.fn(),
    loadOlderConversationMessages: vi.fn(),
    recoverAttempt: vi.fn(),
    prepareConversationForCompletion: vi.fn(),
    acceptLegacyImport: vi.fn(),
    declineLegacyImport: vi.fn(),
    createConversation: vi.fn(),
    selectConversation: vi.fn(),
    renameConversation: vi.fn(),
    deleteConversation: vi.fn(),
    clearConversations: vi.fn(),
    setConversationModel: vi.fn(),
    addMessage: vi.fn(),
    updateMessage: vi.fn(),
    removeMessages: vi.fn(),
    startStreaming: vi.fn(),
    appendStreamingContent: vi.fn(),
    finishStreaming: vi.fn(),
    failStreaming: vi.fn(),
    stopStreaming: vi.fn(),
  })
  return { useChatStore: () => store }
})

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { useChatStore } from '@/stores/chat'
import { ChatAPIError } from '@/api/chat'
import ChatView from '../ChatView.vue'

const AppLayoutStub = {
  props: ['variant'],
  template: '<div data-test="app-layout"><slot /></div>',
}

const composerInsertText = vi.fn()

const ChatComposerStub = {
  props: [
    'modelValue',
    'streaming',
    'disabled',
    'insufficientBalance',
    'submissionBusy',
    'hasAttachments',
    'attachmentsValid',
  ],
  emits: ['update:modelValue', 'send', 'stop'],
  setup(_props: unknown, { expose }: {
    expose: (value: { focus: () => void; insertText: (text: string) => boolean }) => void
  }) {
    const input = ref<HTMLTextAreaElement | null>(null)
    expose({
      focus: () => input.value?.focus(),
      insertText: composerInsertText,
    })
    return { input }
  },
  template: `
    <div
      data-test="chat-composer"
      :data-disabled="String(disabled)"
      :data-streaming="String(streaming)"
      :data-insufficient-balance="String(insufficientBalance)"
      :data-submission-busy="String(submissionBusy)"
    >
      <textarea ref="input" data-test="composer-input"></textarea>
      <slot name="attachments" />
      <slot name="leading" />
      <slot name="trailing" />
      <slot name="empty-action" />
      <button type="button" data-test="composer-stop" @click="$emit('stop')">Stop</button>
    </div>
  `,
}

const attachmentPickerReady = ref<ChatAttachment[]>([])
const attachmentPickerAddFiles = vi.fn()
const attachmentPickerCommitAll = vi.fn()
const attachmentPickerDiscardAll = vi.fn().mockResolvedValue(undefined)

type DataTransferStub = {
  types: string[]
  files: File[]
  dropEffect: DataTransfer['dropEffect']
}

function dataTransfer(files: File[] = [], types = ['Files']): DataTransferStub {
  return { types, files, dropEffect: 'none' }
}

function dispatchDataTransferEvent(
  target: Element,
  type: 'dragenter' | 'dragover' | 'dragleave' | 'drop',
  transfer: DataTransferStub,
  relatedTarget: EventTarget | null = null,
): Event {
  const event = new Event(type, { bubbles: true, cancelable: true })
  Object.defineProperty(event, 'dataTransfer', { value: transfer })
  Object.defineProperty(event, 'relatedTarget', { value: relatedTarget })
  target.dispatchEvent(event)
  return event
}

const ChatAttachmentPickerStub = {
  props: ['disabled', 'supportsVision'],
  emits: ['change', 'busy-change', 'valid-change'],
  setup(_props: unknown, { expose }: {
    expose: (value: Record<string, unknown>) => void
  }) {
    expose({
      cancel: vi.fn(),
      retry: vi.fn(),
      remove: vi.fn(),
      addFiles: attachmentPickerAddFiles,
      getReadyAttachments: () => attachmentPickerReady.value,
      commitAll: attachmentPickerCommitAll,
      discardAll: attachmentPickerDiscardAll,
    })
  },
  template: `
    <div
      data-test="chat-attachment-picker"
      :data-disabled="String(disabled)"
      :data-supports-vision="String(supportsVision)"
    />
  `,
}

const ChatAttachmentPreviewListStub = {
  props: ['items', 'supportsVision'],
  template: '<div data-test="chat-attachment-preview" />',
}

const ChatVoiceInputStub = {
  props: [
    'disabled',
    'contextKey',
    'maxDurationMs',
    'maxBytes',
    'acceptedMimeTypes',
  ],
  emits: ['busy-change', 'transcribed'],
  template: `
    <div
      data-test="chat-voice-input"
      :data-disabled="String(disabled)"
      :data-context-key="contextKey"
      :data-max-duration-ms="String(maxDurationMs)"
      :data-max-bytes="String(maxBytes)"
      :data-accepted-mime-types="acceptedMimeTypes.join(',')"
    >
      <button type="button" data-test="voice-busy" @click="$emit('busy-change', true)">Busy</button>
      <button type="button" data-test="voice-idle" @click="$emit('busy-change', false)">Idle</button>
      <button type="button" data-test="voice-transcribed" @click="$emit('transcribed', '语音草稿')">Text</button>
    </div>
  `,
}

const ChatVoiceModeButtonStub = {
  props: ['available', 'disabled'],
  emits: ['activate'],
  template: `
    <button
      type="button"
      data-test="chat-voice-mode"
      :data-available="String(available)"
      :disabled="disabled"
      :aria-disabled="disabled || !available"
      @click="available && $emit('activate')"
    >Voice</button>
  `,
}

const ChatModelSettingsStub = {
  props: ['modelValue', 'reasoningEffort', 'modelOptions', 'disabled', 'loading'],
  emits: ['update:modelValue', 'update:reasoningEffort'],
  template: `
    <div
      data-test="chat-model-settings"
      :data-disabled="String(disabled)"
      :data-option-count="String(modelOptions.length)"
      :data-first-option="modelOptions[0]?.value || ''"
      :data-model="modelValue"
      :data-reasoning-effort="reasoningEffort"
      :data-reasoning-slider="String(
        modelOptions.find((option) => option.value === modelValue)?.supportsReasoningSlider === true
      )"
    />
  `,
}

const ChatHistoryPanelStub = {
  props: {
    mobile: {
      type: Boolean,
      default: false,
    },
    overlay: {
      type: Boolean,
      default: false,
    },
    sidebarId: {
      type: String,
      default: '',
    },
  },
  emits: ['new', 'close', 'select', 'rename', 'delete', 'clear'],
  template: `
    <aside
      :id="sidebarId || undefined"
      data-test="chat-history"
      :data-mobile="String(!!mobile)"
      :data-overlay="String(!!overlay)"
    >
      <template v-if="mobile || overlay">
        <button type="button" data-test="history-first">First</button>
        <input data-test="history-middle" />
        <button type="button" data-test="history-last">Last</button>
      </template>
    </aside>
  `,
}

const ChatMessageItemStub = {
  props: ['message', 'retryable'],
  emits: ['retry'],
  template: `
    <article :data-message-id="message.id">
      <button
        v-if="retryable"
        type="button"
        data-test="retry-message"
        @click="$emit('retry')"
      >
        Retry
      </button>
    </article>
  `,
}

const BaseDialogStub = {
  props: ['show', 'title', 'variant', 'showCloseButton', 'descriptionId'],
  emits: ['close'],
  template: `
    <div
      v-if="show"
      data-test="base-dialog"
      :data-title="title"
      :data-variant="variant"
      :data-show-close="String(showCloseButton)"
      :aria-describedby="descriptionId || undefined"
    >
      <slot />
      <slot name="footer" />
    </div>
  `,
}

const IconStub = {
  props: ['name'],
  template: '<span :data-icon="name" />',
}

const RouterLinkStub = {
  props: ['to'],
  template: '<a :href="to"><slot /></a>',
}

interface AuthStoreHarness {
  user: { id: number; balance: number } | null
  refreshUser: ReturnType<typeof vi.fn>
}

interface ChatStoreHarness {
  userId: string | null
  conversations: ChatConversation[]
  activeConversationId: string | null
  activeConversation: ChatConversation | null
  hydrated: boolean
  hydrating: boolean
  persistenceAvailable: boolean
  syncStatus: 'idle' | 'syncing' | 'offline' | 'error' | 'unavailable'
  serverHistoryAvailable: boolean | null
  legacyImportRequired: boolean
  legacyConversationIds: string[]
  searchResults: ChatConversation[]
  conversationsHaveMore: boolean
  loadingConversationPage: boolean
  searchingHistory: boolean
  loadingConversationMessages: Set<string>
  isStreaming: boolean
  streamingConversationId: string | null
  streamingMessageId: string | null
  hydrate: ReturnType<typeof vi.fn>
  flushPersistence: ReturnType<typeof vi.fn>
  syncHistory: ReturnType<typeof vi.fn>
  loadConversationPage: ReturnType<typeof vi.fn>
  searchHistory: ReturnType<typeof vi.fn>
  loadConversationDetail: ReturnType<typeof vi.fn>
  loadOlderConversationMessages: ReturnType<typeof vi.fn>
  recoverAttempt: ReturnType<typeof vi.fn>
  prepareConversationForCompletion: ReturnType<typeof vi.fn>
  acceptLegacyImport: ReturnType<typeof vi.fn>
  declineLegacyImport: ReturnType<typeof vi.fn>
  createConversation: ReturnType<typeof vi.fn>
  selectConversation: ReturnType<typeof vi.fn>
  renameConversation: ReturnType<typeof vi.fn>
  deleteConversation: ReturnType<typeof vi.fn>
  clearConversations: ReturnType<typeof vi.fn>
  setConversationModel: ReturnType<typeof vi.fn>
  addMessage: ReturnType<typeof vi.fn>
  updateMessage: ReturnType<typeof vi.fn>
  removeMessages: ReturnType<typeof vi.fn>
  startStreaming: ReturnType<typeof vi.fn>
  appendStreamingContent: ReturnType<typeof vi.fn>
  finishStreaming: ReturnType<typeof vi.fn>
  failStreaming: ReturnType<typeof vi.fn>
  stopStreaming: ReturnType<typeof vi.fn>
}

const authStore = useAuthStore() as unknown as AuthStoreHarness
const chatStore = useChatStore() as unknown as ChatStoreHarness
let wrapper: VueWrapper | undefined
let generatedMessageId = 0

async function mountView(): Promise<VueWrapper> {
  wrapper = mount(ChatView, {
    attachTo: document.body,
    global: {
      plugins: [createPinia()],
      stubs: {
        AppLayout: AppLayoutStub,
        BaseDialog: BaseDialogStub,
        ChatComposer: ChatComposerStub,
        ChatAttachmentPicker: ChatAttachmentPickerStub,
        ChatAttachmentPreviewList: ChatAttachmentPreviewListStub,
        ChatHistoryPanel: ChatHistoryPanelStub,
        ChatMessageItem: ChatMessageItemStub,
        ChatModelSettings: ChatModelSettingsStub,
        ChatVoiceInput: ChatVoiceInputStub,
        ChatVoiceModeButton: ChatVoiceModeButtonStub,
        Icon: IconStub,
        RouterLink: RouterLinkStub,
        Transition: true,
      },
    },
  })
  await flushPromises()
  await nextTick()
  return wrapper
}

function installAnimationFrameHarness() {
  let nextFrameId = 1
  const callbacks = new Map<number, FrameRequestCallback>()

  vi.stubGlobal('requestAnimationFrame', vi.fn((callback: FrameRequestCallback) => {
    const frameId = nextFrameId
    nextFrameId += 1
    callbacks.set(frameId, callback)
    return frameId
  }))
  vi.stubGlobal('cancelAnimationFrame', vi.fn((frameId: number) => {
    callbacks.delete(frameId)
  }))

  return {
    pendingCount: () => callbacks.size,
    runAll: async () => {
      const pendingCallbacks = Array.from(callbacks.values())
      callbacks.clear()
      pendingCallbacks.forEach((callback) => callback(0))
      await nextTick()
      await flushPromises()
    },
  }
}

function installResizeObserverHarness() {
  let callback: ResizeObserverCallback | undefined
  let observer: ResizeObserver | undefined
  const observe = vi.fn()
  const unobserve = vi.fn()
  const disconnect = vi.fn()

  class ResizeObserverStub {
    observe = observe
    unobserve = unobserve
    disconnect = disconnect

    constructor(nextCallback: ResizeObserverCallback) {
      callback = nextCallback
      observer = this as unknown as ResizeObserver
    }
  }

  vi.stubGlobal('ResizeObserver', ResizeObserverStub)

  return {
    observe,
    disconnect,
    trigger: () => {
      if (!callback || !observer) throw new Error('ResizeObserver fixture is not mounted')
      callback([], observer)
    },
  }
}

function stubMatchMediaWithoutReducedMotion() {
  vi.stubGlobal('matchMedia', vi.fn((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  } as unknown as MediaQueryList)))
}

function installMessageScrollerHarness(view: VueWrapper) {
  const scroller = view.get('.chat-messages')
  const element = scroller.element as HTMLElement
  const metrics = {
    scrollHeight: 1000,
    clientHeight: 400,
    scrollTop: 600,
  }
  const scrollTo = vi.fn((options: ScrollToOptions) => {
    if (typeof options.top === 'number') {
      metrics.scrollTop = Math.min(
        Math.max(0, options.top),
        Math.max(0, metrics.scrollHeight - metrics.clientHeight),
      )
    }
  })

  Object.defineProperties(element, {
    scrollHeight: {
      configurable: true,
      get: () => metrics.scrollHeight,
    },
    clientHeight: {
      configurable: true,
      get: () => metrics.clientHeight,
    },
    scrollTop: {
      configurable: true,
      get: () => metrics.scrollTop,
      set: (value: number) => {
        metrics.scrollTop = value
      },
    },
    scrollTo: {
      configurable: true,
      value: scrollTo,
    },
  })

  return { element, metrics, scroller, scrollTo }
}

function prepareStreamingConversation() {
  const conversation: ChatConversation = {
    id: 'scroll-conversation',
    userId: '7',
    title: 'Scrollable conversation',
    model: 'gpt-5',
    messages: [
      {
        id: 'scroll-user',
        role: 'user',
        content: 'Explain this in detail.',
        createdAt: 1,
        status: 'complete',
      },
      {
        id: 'scroll-assistant',
        role: 'assistant',
        content: 'First part.',
        createdAt: 2,
        status: 'streaming',
      },
    ],
    createdAt: 1,
    updatedAt: 2,
  }

  chatStore.userId = '7'
  chatStore.hydrated = true
  chatStore.conversations = [conversation]
  chatStore.activeConversationId = conversation.id
  chatStore.activeConversation = conversation
  chatStore.isStreaming = true
  chatStore.streamingConversationId = conversation.id
  chatStore.streamingMessageId = 'scroll-assistant'
}

function appendStreamingChunk(content: string) {
  const message = chatStore.activeConversation?.messages.find(
    ({ id }) => id === chatStore.streamingMessageId,
  )
  if (!message) throw new Error('Streaming message fixture is missing')
  message.content += content
}

describe('ChatView catalog and hydration gates', () => {
  beforeEach(() => {
    vi.resetAllMocks()
    window.history.replaceState(null, '', '/chat')
    generatedMessageId = 0
    attachmentPickerReady.value = []
    attachmentPickerAddFiles.mockReset()
    attachmentPickerCommitAll.mockReset()
    attachmentPickerDiscardAll.mockReset().mockResolvedValue(undefined)
    authStore.user = { id: 7, balance: 10 }
    chatStore.userId = null
    chatStore.conversations = []
    chatStore.activeConversationId = null
    chatStore.activeConversation = null
    chatStore.hydrated = false
    chatStore.hydrating = false
    chatStore.persistenceAvailable = true
    chatStore.syncStatus = 'idle'
    chatStore.serverHistoryAvailable = true
    chatStore.legacyImportRequired = false
    chatStore.legacyConversationIds = []
    chatStore.searchResults = []
    chatStore.conversationsHaveMore = false
    chatStore.loadingConversationPage = false
    chatStore.searchingHistory = false
    chatStore.loadingConversationMessages = new Set<string>()
    chatStore.isStreaming = false
    chatStore.streamingConversationId = null
    chatStore.streamingMessageId = null
    chatStore.hydrate.mockResolvedValue(undefined)
    chatStore.flushPersistence.mockResolvedValue(undefined)
    chatStore.syncHistory.mockResolvedValue(undefined)
    chatStore.loadConversationPage.mockResolvedValue(undefined)
    chatStore.searchHistory.mockResolvedValue(undefined)
    chatStore.loadConversationDetail.mockResolvedValue(true)
    chatStore.loadOlderConversationMessages.mockResolvedValue(true)
    chatStore.recoverAttempt.mockResolvedValue(null)
    chatStore.prepareConversationForCompletion.mockResolvedValue(true)
    chatStore.acceptLegacyImport.mockReturnValue(0)
    chatStore.declineLegacyImport.mockReturnValue(undefined)
    chatStore.selectConversation.mockImplementation((id: string | null) => {
      chatStore.activeConversationId = id
      chatStore.activeConversation = id === null
        ? null
        : (chatStore.conversations.find((conversation) => conversation.id === id) ?? null)
      return true
    })
    chatStore.setConversationModel.mockImplementation((id: string, model: string) => {
      const conversation = chatStore.conversations.find((candidate) => candidate.id === id)
      if (!conversation) return false
      conversation.model = model
      if (chatStore.activeConversationId === id) chatStore.activeConversation = conversation
      return true
    })
    chatStore.addMessage.mockImplementation((
      conversationId: string,
      input: Partial<ChatMessage> & Pick<ChatMessage, 'role' | 'content'>,
    ) => {
      const conversation = chatStore.conversations.find(({ id }) => id === conversationId)
      if (!conversation) return null
      generatedMessageId += 1
      const message: ChatMessage = {
        ...input,
        id: input.id || `generated-message-${generatedMessageId}`,
        role: input.role,
        content: input.content,
        createdAt: input.createdAt ?? Date.now(),
        status: input.status ?? 'complete',
      }
      conversation.messages.push(message)
      return message
    })
    chatStore.updateMessage.mockImplementation((
      conversationId: string,
      messageId: string,
      patch: Partial<ChatMessage>,
    ) => {
      const message = chatStore.conversations
        .find(({ id }) => id === conversationId)
        ?.messages.find(({ id }) => id === messageId)
      if (!message) return false
      Object.assign(message, patch)
      return true
    })
    chatStore.removeMessages.mockImplementation((
      conversationId: string,
      messageIds: string[],
    ) => {
      const conversation = chatStore.conversations.find(({ id }) => id === conversationId)
      if (!conversation) return false
      const ids = new Set(messageIds)
      const before = conversation.messages.length
      conversation.messages = conversation.messages.filter(({ id }) => !ids.has(id))
      if (chatStore.activeConversationId === conversationId) {
        chatStore.activeConversation = conversation
      }
      return before !== conversation.messages.length
    })
    chatStore.startStreaming.mockImplementation((
      conversationId: string,
      messageId: string,
    ) => {
      chatStore.isStreaming = true
      chatStore.streamingConversationId = conversationId
      chatStore.streamingMessageId = messageId
      return true
    })
    chatStore.finishStreaming.mockImplementation((
      conversationId: string,
      messageId: string,
      finishReason: string | null,
    ) => {
      const message = chatStore.conversations
        .find(({ id }) => id === conversationId)
        ?.messages.find(({ id }) => id === messageId)
      if (message) {
        message.status = 'complete'
        message.finishReason = finishReason
      }
      chatStore.isStreaming = false
      chatStore.streamingConversationId = null
      chatStore.streamingMessageId = null
      return true
    })
    chatStore.failStreaming.mockImplementation((
      conversationId: string,
      messageId: string,
      errorMessage: string,
      errorCode?: string,
    ) => {
      const message = chatStore.conversations
        .find(({ id }) => id === conversationId)
        ?.messages.find(({ id }) => id === messageId)
      if (message) {
        message.status = 'error'
        message.errorMessage = errorMessage
        message.errorCode = errorCode
      }
      chatStore.isStreaming = false
      chatStore.streamingConversationId = null
      chatStore.streamingMessageId = null
      return true
    })
    chatStore.stopStreaming.mockReturnValue(false)
    authStore.refreshUser.mockResolvedValue(authStore.user)
    apiMocks.isAbortError.mockImplementation((error: unknown) => (
      !!error && typeof error === 'object' && (error as { name?: unknown }).name === 'AbortError'
    ))
    apiMocks.getChatModels.mockResolvedValue({
      models: [{
        id: 'gpt-5',
        display_name: 'GPT-5',
        recommended: true,
        supports_vision: true,
      }],
      balance: 10,
    })
    apiMocks.getChatCapabilities.mockResolvedValue({})
    apiMocks.createChatAttemptId.mockReturnValue('attempt-generated')
    apiMocks.createChatIdempotencyKey.mockReturnValue('11111111-2222-4333-8444-555555555555')
    composerInsertText.mockReturnValue(true)
    apiMocks.pollChatReceipt.mockResolvedValue({
      receiptId: 'receipt-pending',
      status: 'pending',
    })
    apiMocks.streamChatCompletion.mockImplementation(async (
      _request: unknown,
      handlers: { onAccepted?: () => void },
    ) => {
      handlers.onAccepted?.()
      return {
        receivedDone: true,
        finishReason: 'stop',
        usage: null,
        receiptId: null,
      }
    })
  })

  afterEach(() => {
    wrapper?.unmount()
    wrapper = undefined
    window.history.replaceState(null, '', '/chat')
    document.body.innerHTML = ''
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
    vi.unstubAllGlobals()
  })

  it('在真正的新对话中展示随机 Greeting 与单一 composer', async () => {
    vi.spyOn(Math, 'random').mockReturnValue(0)
    chatStore.userId = '7'
    chatStore.hydrated = true

    const view = await mountView()

    expect(view.findAll('[data-test="chat-composer"]')).toHaveLength(1)
    expect(view.get('[data-test="chat-new-chat-hero"]').text()).toBe(CHAT_GREETINGS[0])
    expect(view.get('[data-test="chat-new-chat-hero"]').findAll('h2')).toHaveLength(1)
    expect(view.get('[data-test="chat-new-chat-hero"]').find('p').exists()).toBe(false)
    expect(view.find('[data-test="chat-new-chat-shortcuts"]').exists()).toBe(false)
    expect(view.find('header.chat-toolbar').exists()).toBe(false)
    expect(view.find('.chat-toolbar__session').exists()).toBe(false)
    expect(view.find('.chat-mobile-actions__history-button').exists()).toBe(true)
    expect(view.get('.chat-workspace__main').classes()).toContain('chat-workspace__main--new-chat')
  })

  it('按 ChatGPT 结构确认删除时展示真实标题，并仅在确认后删除目标会话', async () => {
    const conversation: ChatConversation = {
      id: 'delete-target',
      userId: '7',
      title: '我现在的网站需要一个语音。对话聊天的功能，然后的话，我需要把这个语音聊天的音频就是它...',
      model: 'gpt-5',
      messages: [],
      createdAt: 1,
      updatedAt: 1,
    }
    chatStore.userId = '7'
    chatStore.hydrated = true
    chatStore.conversations = [conversation]

    const view = await mountView()
    view.findComponent(ChatHistoryPanelStub).vm.$emit('delete', conversation.id)
    await flushPromises()

    const dialog = view.get('[data-test="base-dialog"]')
    expect(dialog.attributes('data-title')).toBe('chat.confirm.deleteTitle')
    expect(dialog.attributes('data-variant')).toBe('workspace-confirm')
    expect(dialog.attributes('data-show-close')).toBe('false')
    expect(dialog.attributes('aria-describedby')).toBe('chat-delete-confirm-description')
    const displayedTitle = dialog.get('.chat-delete-confirm__message strong')
    expect(displayedTitle.text()).toBe(toChatConversationTitlePreview(conversation.title))
    expect(displayedTitle.attributes('aria-label')).toBe(conversation.title)
    expect(displayedTitle.attributes('title')).toBe(conversation.title)
    expect(dialog.get('.chat-delete-confirm__memory').text()).toContain('chat.confirm.memorySettings')

    await dialog.get('.chat-delete-confirm__button--cancel').trigger('click')
    expect(chatStore.deleteConversation).not.toHaveBeenCalled()
    expect(view.find('[data-test="base-dialog"]').exists()).toBe(false)

    view.findComponent(ChatHistoryPanelStub).vm.$emit('delete', conversation.id)
    await flushPromises()
    await view.get('.chat-delete-confirm__button--danger').trigger('click')

    expect(chatStore.deleteConversation).toHaveBeenCalledTimes(1)
    expect(chatStore.deleteConversation).toHaveBeenCalledWith(conversation.id)
    expect(view.find('[data-test="base-dialog"]').exists()).toBe(false)
  })

  it('重复创建新聊天时也会换一条 Greeting', async () => {
    vi.spyOn(Math, 'random')
      .mockReturnValueOnce(0)
      .mockReturnValueOnce(0.999999)
    chatStore.userId = '7'
    chatStore.hydrated = true

    const view = await mountView()
    expect(view.get('[data-test="chat-new-chat-hero"]').text()).toBe(CHAT_GREETINGS[0])

    view.findComponent(ChatHistoryPanelStub).vm.$emit('new')
    await nextTick()

    expect(view.get('[data-test="chat-new-chat-hero"]').text()).toBe(CHAT_GREETINGS.at(-1))
    expect(Math.random).toHaveBeenCalledTimes(2)
    expect(chatStore.selectConversation).toHaveBeenCalledWith(null)
    expect(new URL(window.location.href).searchParams.get('conversation')).toBe('new')
    expect(chatStore.createConversation).not.toHaveBeenCalled()
    expect(chatStore.addMessage).not.toHaveBeenCalled()
    expect(apiMocks.streamChatCompletion).not.toHaveBeenCalled()
  })

  it('刷新入口声明新聊天时会在历史恢复前保持空态，并在选择会话后清除声明', async () => {
    const conversation: ChatConversation = {
      id: 'persisted-conversation',
      userId: '7',
      title: 'Persisted conversation',
      model: 'gpt-5.6-sol',
      messages: [{
        id: 'persisted-message',
        role: 'user',
        content: 'Old message',
        createdAt: 1,
        status: 'complete',
      }],
      createdAt: 1,
      updatedAt: 1,
    }
    window.history.replaceState(null, '', '/chat?conversation=new')
    chatStore.userId = '7'
    chatStore.hydrated = true
    chatStore.conversations = [conversation]
    chatStore.activeConversationId = conversation.id
    chatStore.activeConversation = conversation

    const view = await mountView()
    await nextTick()

    expect(chatStore.selectConversation).toHaveBeenCalledWith(null)
    expect(chatStore.activeConversationId).toBeNull()
    expect(view.find('[data-test="chat-new-chat-hero"]').exists()).toBe(true)
    expect(new URL(window.location.href).searchParams.get('conversation')).toBe('new')

    view.findComponent(ChatHistoryPanelStub).vm.$emit('select', conversation.id)
    await nextTick()

    expect(chatStore.activeConversationId).toBe(conversation.id)
    expect(new URL(window.location.href).searchParams.has('conversation')).toBe(false)
  })

  it('新聊天发送首条消息创建会话时会同步清除刷新声明', async () => {
    window.history.replaceState(null, '', '/chat?conversation=new')
    chatStore.userId = '7'
    chatStore.hydrated = true
    chatStore.createConversation.mockImplementation((model: string, title: string) => {
      const conversation: ChatConversation = {
        id: 'first-message-conversation',
        userId: '7',
        title,
        model,
        messages: [],
        createdAt: 1,
        updatedAt: 1,
      }
      chatStore.conversations = [conversation]
      chatStore.activeConversationId = conversation.id
      chatStore.activeConversation = conversation
      return conversation
    })

    const view = await mountView()
    view.findComponent(ChatComposerStub).vm.$emit('send', 'First message')

    expect(chatStore.activeConversationId).toBe('first-message-conversation')
    expect(new URL(window.location.href).searchParams.has('conversation')).toBe(false)
    await flushPromises()
  })

  it('历史摘要尚未加载消息详情时不会误显示新聊天首页', async () => {
    const conversation: ChatConversation = {
      id: 'history-summary',
      userId: '7',
      title: 'History summary',
      model: 'gpt-5.5',
      messages: [],
      messageCount: 4,
      createdAt: 1,
      updatedAt: 2,
    }
    chatStore.userId = '7'
    chatStore.hydrated = true
    chatStore.conversations = [conversation]
    chatStore.activeConversationId = conversation.id
    chatStore.activeConversation = conversation

    const view = await mountView()

    expect(view.find('[data-test="chat-new-chat-hero"]').exists()).toBe(false)
    expect(view.findAll('[data-test="chat-composer"]')).toHaveLength(1)
    expect(view.get('.chat-workspace__main').classes()).not.toContain('chat-workspace__main--new-chat')
  })

  it('在当前用户 history hydration 完成前保持 composer 禁用', async () => {
    chatStore.hydrate.mockImplementation(() => new Promise<void>(() => undefined))
    const view = await mountView()

    expect(chatStore.hydrate).toHaveBeenCalledWith(7)
    expect(view.get('[data-test="chat-composer"]').attributes('data-disabled')).toBe('true')

    chatStore.userId = '7'
    chatStore.hydrated = true
    await nextTick()

    expect(view.get('[data-test="chat-composer"]').attributes('data-disabled')).toBe('false')
  })

  it('从页面任意位置拖入文件时显示全屏接收态并原样交给上传器', async () => {
    chatStore.userId = '7'
    chatStore.hydrated = true
    const view = await mountView()
    const picker = view.findComponent(ChatAttachmentPickerStub)
    const files = [
      new File(['image'], 'photo.png', { type: 'image/png' }),
      new File(['document'], 'brief.pdf', { type: 'application/pdf' }),
    ]
    const transfer = dataTransfer(files)

    expect(picker.props('disabled')).toBe(false)
    const enter = dispatchDataTransferEvent(document.body, 'dragenter', transfer)
    await nextTick()

    const overlay = document.querySelector('[data-test="chat-file-drop-overlay"]')
    expect(enter.defaultPrevented).toBe(true)
    expect(transfer.dropEffect).toBe('copy')
    expect(overlay?.textContent).toContain('chat.attachments.dropHere')

    const over = dispatchDataTransferEvent(view.get('.chat-messages').element, 'dragover', transfer)
    const drop = dispatchDataTransferEvent(view.get('.chat-workspace__main').element, 'drop', transfer)
    await nextTick()

    expect(over.defaultPrevented).toBe(true)
    expect(drop.defaultPrevented).toBe(true)
    expect(attachmentPickerAddFiles).toHaveBeenCalledOnce()
    expect(attachmentPickerAddFiles).toHaveBeenCalledWith(files)
    expect(document.querySelector('[data-test="chat-file-drop-overlay"]')).toBeNull()
  })

  it('跨过页面内嵌套区域时保持单一接收态，最后离开才收起', async () => {
    chatStore.userId = '7'
    chatStore.hydrated = true
    const view = await mountView()
    const transfer = dataTransfer([
      new File(['image'], 'photo.png', { type: 'image/png' }),
    ])
    const workspace = view.get('.chat-workspace').element
    const messagesRegion = view.get('.chat-messages-region').element

    dispatchDataTransferEvent(workspace, 'dragenter', transfer)
    dispatchDataTransferEvent(messagesRegion, 'dragenter', transfer)
    dispatchDataTransferEvent(messagesRegion, 'dragleave', transfer, workspace)
    await nextTick()
    expect(document.querySelector('[data-test="chat-file-drop-overlay"]')).not.toBeNull()

    dispatchDataTransferEvent(workspace, 'dragleave', transfer)
    await nextTick()
    expect(document.querySelector('[data-test="chat-file-drop-overlay"]')).toBeNull()
  })

  it('不拦截文本拖放，并在不可上传时只阻止浏览器打开文件', async () => {
    const view = await mountView()
    const textTransfer = dataTransfer([], ['text/plain', 'text/uri-list'])
    const fileTransfer = dataTransfer([
      new File(['image'], 'photo.png', { type: 'image/png' }),
    ])

    const textDrop = dispatchDataTransferEvent(document.body, 'drop', textTransfer)
    const fileEnter = dispatchDataTransferEvent(document.body, 'dragenter', fileTransfer)
    const fileDrop = dispatchDataTransferEvent(document.body, 'drop', fileTransfer)
    await nextTick()

    expect(textDrop.defaultPrevented).toBe(false)
    expect(fileEnter.defaultPrevented).toBe(true)
    expect(fileDrop.defaultPrevented).toBe(true)
    expect(fileTransfer.dropEffect).toBe('none')
    expect(document.querySelector('[data-test="chat-file-drop-overlay"]')).toBeNull()
    expect(attachmentPickerAddFiles).not.toHaveBeenCalled()
    expect(view.findComponent(ChatAttachmentPickerStub).props('disabled')).toBe(true)
  })

  it('拖动中开始生成会立即收起接收态，空文件和卸载后不残留监听', async () => {
    chatStore.userId = '7'
    chatStore.hydrated = true
    const view = await mountView()
    const transfer = dataTransfer([
      new File(['image'], 'photo.png', { type: 'image/png' }),
    ])

    dispatchDataTransferEvent(document.body, 'dragenter', transfer)
    await nextTick()
    expect(document.querySelector('[data-test="chat-file-drop-overlay"]')).not.toBeNull()

    chatStore.isStreaming = true
    await nextTick()
    expect(document.querySelector('[data-test="chat-file-drop-overlay"]')).toBeNull()

    const blockedDrop = dispatchDataTransferEvent(document.body, 'drop', transfer)
    const emptyDrop = dispatchDataTransferEvent(document.body, 'drop', dataTransfer())
    expect(blockedDrop.defaultPrevented).toBe(true)
    expect(emptyDrop.defaultPrevented).toBe(true)
    expect(attachmentPickerAddFiles).not.toHaveBeenCalled()

    view.unmount()
    wrapper = undefined
    const afterUnmount = dispatchDataTransferEvent(document.body, 'drop', transfer)
    expect(afterUnmount.defaultPrevented).toBe(false)
  })

  it('capabilities 缺少或明确关闭 transcription 时隐藏麦克风但保留语音模式入口', async () => {
    chatStore.userId = '7'
    chatStore.hydrated = true
    const view = await mountView()
    expect(view.find('[data-test="chat-voice-input"]').exists()).toBe(false)
    expect(view.get('[data-test="chat-voice-mode"]').attributes('data-available')).toBe('false')

    view.unmount()
    wrapper = undefined

    apiMocks.getChatCapabilities.mockResolvedValue({
      transcription: { enabled: false },
    })
    const disabledView = await mountView()
    expect(disabledView.find('[data-test="chat-voice-input"]').exists()).toBe(false)
    expect(disabledView.get('[data-test="chat-voice-mode"]').attributes('data-available')).toBe('false')
  })

  it('按 capability 展示语音入口、回填草稿且转写期间拒绝发送', async () => {
    chatStore.userId = '7'
    chatStore.hydrated = true
    apiMocks.getChatCapabilities.mockResolvedValue({
      transcription: {
        enabled: true,
        max_upload_bytes: 20 * 1024 * 1024,
        max_duration_seconds: 120,
        accepted_mime_types: ['audio/webm;codecs=opus', 'audio/mp4'],
      },
    })
    const view = await mountView()
    const voice = view.get('[data-test="chat-voice-input"]')

    expect(view.get('[data-test="chat-model-settings"]').attributes('data-reasoning-effort'))
      .toBe('low')
    expect(voice.attributes('data-max-duration-ms')).toBe('60000')
    expect(voice.attributes('data-max-bytes')).toBe(String(10 * 1024 * 1024))
    expect(voice.attributes('data-accepted-mime-types'))
      .toBe('audio/webm;codecs=opus,audio/mp4')
    expect(view.get('[data-test="chat-voice-mode"]').attributes('data-available')).toBe('false')

    await view.get('[data-test="voice-transcribed"]').trigger('click')
    expect(composerInsertText).toHaveBeenCalledWith('语音草稿')
    expect(apiMocks.streamChatCompletion).not.toHaveBeenCalled()

    await view.get('[data-test="voice-busy"]').trigger('click')
    expect(view.get('[data-test="chat-composer"]').attributes('data-submission-busy')).toBe('true')
    view.findComponent(ChatComposerStub).vm.$emit('send', '不得发送')
    await nextTick()
    expect(chatStore.createConversation).not.toHaveBeenCalled()
    expect(apiMocks.streamChatCompletion).not.toHaveBeenCalled()
  })

  it('持久化不可用且 hydration 完成后显示历史丢失告警', async () => {
    chatStore.userId = '7'
    chatStore.hydrated = true
    chatStore.persistenceAvailable = false

    const view = await mountView()
    const warning = view.get('[data-test="chat-persistence-warning"]')

    expect(warning.attributes('role')).toBe('alert')
    expect(warning.text()).toContain('chat.persistence.title')
    expect(warning.text()).toContain('chat.persistence.description')
  })

  it('持久化告警的重试操作会刷新保存状态', async () => {
    chatStore.userId = '7'
    chatStore.hydrated = true
    chatStore.persistenceAvailable = false
    chatStore.flushPersistence.mockImplementation(async () => {
      chatStore.persistenceAvailable = true
    })

    const view = await mountView()
    await view.get('[data-test="chat-persistence-retry"]').trigger('click')
    await flushPromises()

    expect(chatStore.flushPersistence).toHaveBeenCalledTimes(1)
    expect(view.find('[data-test="chat-persistence-warning"]').exists()).toBe(false)
  })

  it('持久化正常时不显示历史丢失告警', async () => {
    chatStore.userId = '7'
    chatStore.hydrated = true
    chatStore.persistenceAvailable = true

    const view = await mountView()

    expect(view.find('[data-test="chat-persistence-warning"]').exists()).toBe(false)
  })

  it('history hydration 恢复目录外模型时不在前端改写，交由发送接口最终校验', async () => {
    const view = await mountView()
    const conversation: ChatConversation = {
      id: 'hydrated-conversation',
      userId: '7',
      title: 'Hydrated conversation',
      model: 'gpt-retired',
      messages: [],
      createdAt: 1,
      updatedAt: 1,
    }

    chatStore.conversations = [conversation]
    chatStore.activeConversationId = conversation.id
    chatStore.activeConversation = conversation
    chatStore.userId = '7'
    chatStore.hydrated = true
    await nextTick()

    expect(chatStore.setConversationModel).not.toHaveBeenCalled()
    expect(conversation.model).toBe('gpt-retired')
    expect(view.get('[data-test="chat-composer"]').attributes('data-disabled')).toBe('false')
  })

  it('切换到使用目录外模型的历史会话时保留原始模型', async () => {
    const current: ChatConversation = {
      id: 'current-conversation',
      userId: '7',
      title: 'Current conversation',
      model: 'gpt-5.5',
      messages: [],
      createdAt: 1,
      updatedAt: 2,
    }
    const retired: ChatConversation = {
      id: 'retired-conversation',
      userId: '7',
      title: 'Retired conversation',
      model: 'gpt-retired',
      messages: [],
      createdAt: 1,
      updatedAt: 1,
    }
    chatStore.userId = '7'
    chatStore.hydrated = true
    chatStore.conversations = [current, retired]
    chatStore.activeConversationId = current.id
    chatStore.activeConversation = current

    await mountView()
    chatStore.activeConversationId = retired.id
    chatStore.activeConversation = retired
    await nextTick()

    expect(chatStore.setConversationModel).not.toHaveBeenCalledWith(
      retired.id,
      'gpt-5.6-sol',
    )
    expect(retired.model).toBe('gpt-retired')
  })

  it('目录外历史模型仍可发送，并把服务端模型错误保留在消息流', async () => {
    const conversation: ChatConversation = {
      id: 'unavailable-conversation',
      userId: '7',
      title: 'Unavailable conversation',
      model: 'gpt-retired',
      messages: [],
      createdAt: 1,
      updatedAt: 1,
    }
    chatStore.userId = '7'
    chatStore.hydrated = true
    chatStore.conversations = [conversation]
    chatStore.activeConversationId = conversation.id
    chatStore.activeConversation = conversation
    apiMocks.streamChatCompletion.mockRejectedValueOnce(new ChatAPIError('Unavailable', {
      status: 503,
      code: 'CHAT_MODEL_NOT_AVAILABLE',
    }))

    const view = await mountView()
    expect(view.get('[data-test="chat-composer"]').attributes('data-disabled')).toBe('false')

    view.findComponent(ChatComposerStub).vm.$emit('send', 'validate on send')
    await flushPromises()

    expect(apiMocks.streamChatCompletion.mock.calls[0]?.[0]).toMatchObject({
      model: 'gpt-retired',
    })
    expect(chatStore.failStreaming).toHaveBeenCalledWith(
      conversation.id,
      expect.any(String),
      'chat.errors.modelUnavailable',
      'CHAT_MODEL_NOT_AVAILABLE',
    )
  })

  it('空会话在目录和 history 就绪后聚焦 composer', async () => {
    chatStore.userId = '7'
    chatStore.hydrated = true

    const view = await mountView()

    expect(document.activeElement).toBe(view.get('[data-test="composer-input"]').element)
  })

  it('固定产品配置始终以 gpt-5.6-sol 和极速推理作为新会话默认值', async () => {
    chatStore.userId = '7'
    chatStore.hydrated = true

    const view = await mountView()
    const settings = view.get('[data-test="chat-model-settings"]')

    expect(settings.attributes('data-option-count')).toBe('4')
    expect(settings.attributes('data-first-option')).toBe('gpt-5.6-sol')
    expect(settings.attributes('data-model')).toBe('gpt-5.6-sol')
    expect(settings.attributes('data-reasoning-effort')).toBe('low')
    expect(settings.attributes('data-reasoning-slider')).toBe('true')
  })

  it('固定产品模型均启用四档推理滑块', async () => {
    chatStore.userId = '7'
    chatStore.hydrated = true

    const view = await mountView()

    expect(view.get('[data-test="chat-model-settings"]').attributes('data-reasoning-slider'))
      .toBe('true')
  })

  it('从旧模型会话新建对话时恢复 Sol 默认值', async () => {
    const conversation: ChatConversation = {
      id: 'conversation-old-model',
      userId: '7',
      title: 'Old model',
      model: 'gpt-5.5',
      messages: [],
      createdAt: 1,
      updatedAt: 1,
    }
    chatStore.userId = '7'
    chatStore.hydrated = true
    chatStore.conversations = [conversation]
    chatStore.activeConversationId = conversation.id
    chatStore.activeConversation = conversation

    const view = await mountView()
    expect(view.get('[data-test="chat-model-settings"]').attributes('data-model')).toBe('gpt-5.5')
    expect(view.get('[data-test="chat-model-settings"]').attributes('data-reasoning-slider'))
      .toBe('true')

    view.findComponent(ChatHistoryPanelStub).vm.$emit('new')
    await nextTick()

    expect(chatStore.selectConversation).toHaveBeenCalledWith(null)
    expect(view.get('[data-test="chat-model-settings"]').attributes('data-model'))
      .toBe('gpt-5.6-sol')
  })

  it('初始化、切换账号和重新挂载都不请求模型列表', async () => {
    const view = await mountView()
    expect(apiMocks.getChatModels).not.toHaveBeenCalled()
    expect(view.get('[data-test="chat-model-settings"]').attributes('data-first-option'))
      .toBe('gpt-5.6-sol')

    authStore.user = { id: 8, balance: 2 }
    chatStore.userId = '8'
    chatStore.hydrated = true
    await flushPromises()
    await nextTick()

    expect(apiMocks.getChatModels).not.toHaveBeenCalled()
    expect(view.find('[data-test="chat-balance-state"]').exists()).toBe(false)

    view.unmount()
    wrapper = undefined
    const remounted = await mountView()
    expect(apiMocks.getChatModels).not.toHaveBeenCalled()
    expect(remounted.get('[data-test="chat-model-settings"]').attributes('data-option-count')).toBe('4')
  })

  it('capabilities 请求长期 pending 时文本 Chat 仍立即可用', async () => {
    chatStore.userId = '7'
    chatStore.hydrated = true
    apiMocks.getChatCapabilities.mockImplementationOnce(() => new Promise(() => undefined))

    const view = await mountView()

    expect(apiMocks.getChatCapabilities).toHaveBeenCalledTimes(1)
    expect(apiMocks.getChatModels).not.toHaveBeenCalled()
    expect(view.get('[data-test="chat-model-settings"]').attributes('data-model'))
      .toBe('gpt-5.6-sol')
    expect(view.get('[data-test="chat-composer"]').attributes('data-disabled')).toBe('false')
  })

  it('账户对象刷新保留草稿，只有用户 identity 变化才清空草稿', async () => {
    chatStore.userId = '7'
    chatStore.hydrated = true
    const view = await mountView()
    const composer = view.findComponent(ChatComposerStub)

    composer.vm.$emit('update:modelValue', 'private draft')
    await nextTick()
    expect(composer.props('modelValue')).toBe('private draft')

    authStore.user = { id: 7, balance: 6 }
    await nextTick()
    expect(composer.props('modelValue')).toBe('private draft')

    authStore.user = { id: 8, balance: 6 }
    await nextTick()
    expect(composer.props('modelValue')).toBe('')
  })

  it('切换账号会恢复极速推理并在新账号请求中显式携带 low', async () => {
    chatStore.userId = '7'
    chatStore.hydrated = true
    const view = await mountView()
    const settings = view.findComponent(ChatModelSettingsStub)

    settings.vm.$emit('update:reasoningEffort', 'high')
    await nextTick()
    expect(settings.props('reasoningEffort')).toBe('high')

    authStore.user = { id: 8, balance: 10 }
    chatStore.userId = '8'
    chatStore.hydrated = true
    await flushPromises()
    await nextTick()

    expect(settings.props('reasoningEffort')).toBe('low')

    const conversation: ChatConversation = {
      id: 'new-user-conversation',
      userId: '8',
      title: 'New user',
      model: 'gpt-5',
      messages: [],
      createdAt: 1,
      updatedAt: 1,
    }
    chatStore.conversations = [conversation]
    chatStore.activeConversationId = conversation.id
    chatStore.activeConversation = conversation

    view.findComponent(ChatComposerStub).vm.$emit('send', 'First message after switching')
    await flushPromises()

    expect(apiMocks.streamChatCompletion.mock.calls[0]?.[0]).toMatchObject({
      conversationId: conversation.id,
      model: 'gpt-5',
      reasoningEffort: 'low',
    })
  })

  it('账户余额为零时不在前端拦截发送，由后端统一判断会员与钱包额度', async () => {
    const conversation: ChatConversation = {
      id: 'entitlement-conversation',
      userId: '7',
      title: 'Entitlement admission',
      model: 'gpt-5.6-sol',
      messages: [],
      createdAt: 1,
      updatedAt: 1,
    }
    chatStore.userId = '7'
    chatStore.hydrated = true
    chatStore.conversations = [conversation]
    chatStore.activeConversationId = conversation.id
    chatStore.activeConversation = conversation
    authStore.user = { id: 7, balance: 0 }
    const view = await mountView()

    expect(view.find('[data-test="chat-balance-state"]').exists()).toBe(false)
    expect(view.findComponent(ChatComposerStub).props('insufficientBalance')).toBeUndefined()

    view.findComponent(ChatComposerStub).vm.$emit('send', 'Use my available entitlement')
    await flushPromises()

    expect(apiMocks.streamChatCompletion).toHaveBeenCalledTimes(1)
    expect(apiMocks.streamChatCompletion.mock.calls[0]?.[0]).toMatchObject({
      conversationId: conversation.id,
      model: 'gpt-5.6-sol',
    })
  })

  it('服务端额度不足只形成当前消息错误，不留下会永久锁死输入框的本地余额状态', async () => {
    const conversation: ChatConversation = {
      id: 'server-admission-conversation',
      userId: '7',
      title: 'Server admission',
      model: 'gpt-5.6-sol',
      messages: [],
      createdAt: 1,
      updatedAt: 1,
    }
    chatStore.userId = '7'
    chatStore.hydrated = true
    chatStore.conversations = [conversation]
    chatStore.activeConversationId = conversation.id
    chatStore.activeConversation = conversation
    authStore.user = { id: 7, balance: 0 }
    apiMocks.streamChatCompletion.mockRejectedValueOnce(new ChatAPIError(
      'No usable entitlement',
      { status: 403, code: 'INSUFFICIENT_BALANCE' },
    ))
    const view = await mountView()
    const composer = view.findComponent(ChatComposerStub)

    composer.vm.$emit('send', 'First admission attempt')
    await flushPromises()

    expect(apiMocks.streamChatCompletion).toHaveBeenCalledTimes(1)
    expect(chatStore.failStreaming).toHaveBeenCalledWith(
      conversation.id,
      'generated-message-2',
      'chat.errors.insufficientBalance',
      'INSUFFICIENT_BALANCE',
    )
    expect(composer.props('insufficientBalance')).toBeUndefined()

    composer.vm.$emit('send', 'Retry after entitlement refresh')
    await flushPromises()

    expect(apiMocks.streamChatCompletion).toHaveBeenCalledTimes(2)
  })

  it('模型接口异常不参与 Chat 初始化且不显示顶部错误', async () => {
    chatStore.userId = '7'
    chatStore.hydrated = true
    apiMocks.getChatModels.mockRejectedValueOnce(new ChatAPIError('Unavailable', {
      status: 503,
      code: 'CHAT_CATALOG_UNAVAILABLE',
    }))

    const view = await mountView()

    expect(apiMocks.getChatModels).not.toHaveBeenCalled()
    expect(view.find('.chat-catalog-error').exists()).toBe(false)
    expect(view.get('[data-test="chat-model-settings"]').attributes('data-option-count')).toBe('4')
    expect(view.get('[data-test="chat-composer"]').attributes('data-disabled')).toBe('false')
  })

  it('capabilities 异常不阻断固定模型与文本聊天', async () => {
    chatStore.userId = '7'
    chatStore.hydrated = true
    authStore.user = { id: 7, balance: 10 }
    apiMocks.getChatCapabilities.mockRejectedValueOnce(new ChatAPIError('Unavailable', {
      status: 503,
      code: 'CHAT_CAPABILITIES_UNAVAILABLE',
    }))

    const view = await mountView()

    expect(view.find('[data-test="chat-balance-state"]').exists()).toBe(false)
    expect(view.get('[data-test="chat-composer"]').attributes('data-disabled')).toBe('false')
    expect(view.find('.chat-catalog-error').exists()).toBe(false)
  })

  it('流式更新排入 RAF 后，用户向上 wheel 或滚离底部都会阻止待执行的自动滚动', async () => {
    const animationFrames = installAnimationFrameHarness()
    stubMatchMediaWithoutReducedMotion()
    prepareStreamingConversation()
    const view = await mountView()
    const { metrics, scroller, scrollTo } = installMessageScrollerHarness(view)

    appendStreamingChunk(' Queued before the wheel gesture.')
    await nextTick()
    expect(animationFrames.pendingCount()).toBeGreaterThan(0)

    await scroller.trigger('wheel', { deltaY: -80 })
    await animationFrames.runAll()
    expect(scrollTo).not.toHaveBeenCalled()

    metrics.scrollTop = metrics.scrollHeight - metrics.clientHeight
    await scroller.trigger('scroll')
    appendStreamingChunk(' Queued before scrolling away.')
    await nextTick()
    expect(animationFrames.pendingCount()).toBeGreaterThan(0)

    metrics.scrollTop = 300
    await scroller.trigger('scroll')
    await animationFrames.runAll()
    expect(scrollTo).not.toHaveBeenCalled()
  })

  it('用户手动回到距底部 8px 内后，后续流式更新以 auto 行为恢复跟随', async () => {
    const animationFrames = installAnimationFrameHarness()
    stubMatchMediaWithoutReducedMotion()
    prepareStreamingConversation()
    const view = await mountView()
    const { metrics, scroller, scrollTo } = installMessageScrollerHarness(view)

    metrics.scrollTop = 300
    await scroller.trigger('scroll')
    metrics.scrollTop = metrics.scrollHeight - metrics.clientHeight - 8
    await scroller.trigger('scroll')

    metrics.scrollHeight = 1040
    appendStreamingChunk(' Continue following.')
    await nextTick()
    expect(animationFrames.pendingCount()).toBeGreaterThan(0)
    await animationFrames.runAll()

    expect(scrollTo).toHaveBeenCalledWith({
      top: 1040,
      behavior: 'auto',
    })
  })

  it('点击回到最新消息按钮会恢复跟随并滚动到底部', async () => {
    const animationFrames = installAnimationFrameHarness()
    stubMatchMediaWithoutReducedMotion()
    prepareStreamingConversation()
    const view = await mountView()
    const { metrics, scroller, scrollTo } = installMessageScrollerHarness(view)

    await scroller.trigger('wheel', { deltaY: -24 })
    metrics.scrollTop = 300
    await scroller.trigger('scroll')
    await nextTick()

    await view.get('[data-test="chat-scroll-to-latest"]').trigger('click')
    await animationFrames.runAll()

    expect(scrollTo).toHaveBeenCalledWith(expect.objectContaining({
      top: metrics.scrollHeight,
    }))
  })

  it('程序化 scrollTo 后的中间 scroll 事件不会暂停跟随或闪现回到最新按钮', async () => {
    const animationFrames = installAnimationFrameHarness()
    stubMatchMediaWithoutReducedMotion()
    prepareStreamingConversation()
    const view = await mountView()
    const { metrics, scroller, scrollTo } = installMessageScrollerHarness(view)

    scrollTo.mockImplementation(() => undefined)
    metrics.scrollHeight = 1040
    appendStreamingChunk(' Start a programmatic scroll.')
    await nextTick()
    expect(animationFrames.pendingCount()).toBeGreaterThan(0)
    await animationFrames.runAll()
    expect(scrollTo).toHaveBeenCalledWith({
      top: 1040,
      behavior: 'auto',
    })

    metrics.scrollTop = 620
    await scroller.trigger('scroll')
    await nextTick()
    expect(view.find('[data-test="chat-scroll-to-latest"]').exists()).toBe(false)

    appendStreamingChunk(' Continue after the intermediate scroll event.')
    await nextTick()
    expect(animationFrames.pendingCount()).toBeGreaterThan(0)
  })

  it('用户实际向上滚动时，即使仍在距底部 8px 内也会暂停流式跟随', async () => {
    const animationFrames = installAnimationFrameHarness()
    stubMatchMediaWithoutReducedMotion()
    prepareStreamingConversation()
    const view = await mountView()
    const { metrics, scroller, scrollTo } = installMessageScrollerHarness(view)

    await scroller.trigger('scroll')
    await scroller.trigger('wheel', { deltaY: -24 })
    metrics.scrollTop = metrics.scrollHeight - metrics.clientHeight - 5
    await scroller.trigger('scroll')
    await nextTick()

    expect(view.find('[data-test="chat-scroll-to-latest"]').exists()).toBe(false)
    scrollTo.mockClear()
    appendStreamingChunk(' Must not pull the reader down.')
    await nextTick()
    expect(animationFrames.pendingCount()).toBeGreaterThan(0)
    await animationFrames.runAll()
    expect(scrollTo).not.toHaveBeenCalled()
  })

  it('ResizeObserver 不会恢复距底部 5px 的用户暂停状态', async () => {
    const animationFrames = installAnimationFrameHarness()
    const resizeObserver = installResizeObserverHarness()
    stubMatchMediaWithoutReducedMotion()
    prepareStreamingConversation()
    const view = await mountView()
    const { element, metrics, scroller, scrollTo } = installMessageScrollerHarness(view)

    expect(resizeObserver.observe).toHaveBeenCalledWith(element)
    await scroller.trigger('scroll')
    await scroller.trigger('wheel', { deltaY: -24 })
    metrics.scrollTop = metrics.scrollHeight - metrics.clientHeight - 5
    await scroller.trigger('scroll')
    await nextTick()

    scrollTo.mockClear()
    resizeObserver.trigger()
    expect(animationFrames.pendingCount()).toBeGreaterThan(0)
    await animationFrames.runAll()
    expect(scrollTo).not.toHaveBeenCalled()

    appendStreamingChunk(' Resize must not resume following.')
    await nextTick()
    expect(animationFrames.pendingCount()).toBeGreaterThan(0)
    await animationFrames.runAll()
    expect(scrollTo).not.toHaveBeenCalled()
  })

  it('Ctrl 加向上滚轮手势不会暂停流式跟随', async () => {
    const animationFrames = installAnimationFrameHarness()
    stubMatchMediaWithoutReducedMotion()
    prepareStreamingConversation()
    const view = await mountView()
    const { element, scroller, scrollTo } = installMessageScrollerHarness(view)

    await scroller.trigger('scroll')
    element.dispatchEvent(new WheelEvent('wheel', {
      bubbles: true,
      ctrlKey: true,
      deltaY: -24,
    }))
    await nextTick()
    appendStreamingChunk(' Continue after browser zoom gesture.')
    await nextTick()
    expect(animationFrames.pendingCount()).toBeGreaterThan(0)
    await animationFrames.runAll()

    expect(scrollTo).toHaveBeenCalledWith({
      top: 1000,
      behavior: 'auto',
    })
  })

  it('消息无滚动溢出或用户恢复到底部时不显示回到最新按钮', async () => {
    installAnimationFrameHarness()
    stubMatchMediaWithoutReducedMotion()
    prepareStreamingConversation()
    const view = await mountView()
    const { metrics, scroller } = installMessageScrollerHarness(view)

    metrics.scrollHeight = 400
    metrics.clientHeight = 400
    metrics.scrollTop = 0
    await scroller.trigger('wheel', { deltaY: -24 })
    await scroller.trigger('scroll')
    await nextTick()
    expect(view.find('[data-test="chat-scroll-to-latest"]').exists()).toBe(false)

    metrics.scrollHeight = 1000
    metrics.scrollTop = 300
    await scroller.trigger('wheel', { deltaY: -24 })
    await scroller.trigger('scroll')
    await nextTick()
    expect(view.find('[data-test="chat-scroll-to-latest"]').exists()).toBe(true)

    metrics.scrollTop = metrics.scrollHeight - metrics.clientHeight
    await scroller.trigger('scroll')
    await nextTick()
    expect(view.find('[data-test="chat-scroll-to-latest"]').exists()).toBe(false)
  })

  it('composer stop 事件停止当前流', async () => {
    chatStore.userId = '7'
    chatStore.hydrated = true
    chatStore.isStreaming = true
    const view = await mountView()

    await view.get('[data-test="composer-stop"]').trigger('click')

    expect(chatStore.stopStreaming).toHaveBeenCalledTimes(1)
  })

  it('显式停止导致 Abort 后核对 receipt，但不刷新模型列表', async () => {
    const conversation: ChatConversation = {
      id: 'stopped-conversation',
      userId: '7',
      title: 'Stopped conversation',
      model: 'gpt-5',
      messages: [
        {
          id: 'user-stop',
          role: 'user',
          content: 'Keep going',
          createdAt: 1,
          status: 'complete',
        },
        {
          id: 'assistant-stop',
          role: 'assistant',
          content: 'Partial',
          createdAt: 2,
          status: 'stopped',
        },
      ],
      createdAt: 1,
      updatedAt: 2,
    }
    chatStore.userId = '7'
    chatStore.hydrated = true
    chatStore.conversations = [conversation]
    chatStore.activeConversationId = conversation.id
    chatStore.activeConversation = conversation

    let activeController: AbortController | null = null
    chatStore.startStreaming.mockImplementation((
      conversationId: string,
      messageId: string,
      controller: AbortController,
    ) => {
      activeController = controller
      chatStore.isStreaming = true
      chatStore.streamingConversationId = conversationId
      chatStore.streamingMessageId = messageId
      return true
    })
    chatStore.stopStreaming.mockImplementation(() => {
      if (!activeController) return false
      chatStore.isStreaming = false
      chatStore.streamingConversationId = null
      chatStore.streamingMessageId = null
      activeController.abort()
      return true
    })
    apiMocks.streamChatCompletion.mockImplementationOnce((
      _request: unknown,
      handlers: { onReceiptId?: (receiptId: string) => void },
      options: { signal?: AbortSignal },
    ) => {
      handlers.onReceiptId?.('receipt-stop')
      return new Promise((_resolve, reject) => {
        options.signal?.addEventListener('abort', () => {
          reject(new DOMException('The operation was aborted.', 'AbortError'))
        }, { once: true })
      })
    })
    apiMocks.pollChatReceipt.mockResolvedValueOnce({
      receiptId: 'receipt-stop',
      status: 'charged',
      balanceAfter: 4,
    })
    const view = await mountView()
    await view.get('[data-test="retry-message"]').trigger('click')
    await nextTick()
    expect(apiMocks.streamChatCompletion).toHaveBeenCalledTimes(1)

    await view.get('[data-test="composer-stop"]').trigger('click')
    await flushPromises()
    await nextTick()

    expect(chatStore.stopStreaming).toHaveBeenCalledTimes(1)
    expect(chatStore.failStreaming).not.toHaveBeenCalled()
    expect(apiMocks.pollChatReceipt).toHaveBeenCalledWith(
      'receipt-stop',
      { signal: expect.any(AbortSignal) },
    )
    expect(authStore.refreshUser).not.toHaveBeenCalled()
    expect(apiMocks.getChatModels).not.toHaveBeenCalled()
    expect(view.find('[data-test="chat-balance-state"]').exists()).toBe(false)
  })

  it('重试保留旧 assistant attempt，并用服务端会话信封创建独立 attempt', async () => {
    const conversation: ChatConversation = {
      id: 'conversation-1',
      userId: '7',
      title: 'Existing conversation',
      model: 'gpt-5',
      messages: [
        {
          id: 'user-1',
          role: 'user',
          content: 'Original question',
          createdAt: 1,
          status: 'complete',
        },
        {
          id: 'assistant-1',
          role: 'assistant',
          content: 'Failed answer',
          createdAt: 2,
          status: 'error',
          errorMessage: 'Request failed',
        },
      ],
      createdAt: 1,
      updatedAt: 2,
    }
    chatStore.userId = '7'
    chatStore.hydrated = true
    chatStore.conversations = [conversation]
    chatStore.activeConversationId = conversation.id
    chatStore.activeConversation = conversation
    apiMocks.streamChatCompletion.mockImplementationOnce(async (
      _request: unknown,
      handlers: { onContent?: (content: string) => void },
    ) => {
      handlers.onContent?.('Replacement answer')
      return { receivedDone: true, finishReason: 'stop', usage: null, receiptId: null }
    })

    const view = await mountView()
    await view.get('[data-test="retry-message"]').trigger('click')
    await flushPromises()

    expect(chatStore.updateMessage).toHaveBeenCalledWith(
      conversation.id,
      'assistant-1',
      {
        excludedFromContext: true,
        supersededByMessageId: 'generated-message-1',
      },
    )
    expect(conversation.messages[1]).toMatchObject({
      id: 'assistant-1',
      content: 'Failed answer',
      status: 'error',
      excludedFromContext: true,
      supersededByMessageId: 'generated-message-1',
    })
    expect(conversation.messages[2]).toMatchObject({
      id: 'generated-message-1',
      role: 'assistant',
      attemptId: 'attempt-generated',
      requestedModel: 'gpt-5',
    })
    expect(apiMocks.streamChatCompletion.mock.calls[0]?.[0]).toEqual({
      conversationId: 'conversation-1',
      model: 'gpt-5',
      reasoningEffort: 'low',
      expectedHeadMessageId: 'assistant-1',
      assistantMessageId: 'generated-message-1',
      retryOfMessageId: 'assistant-1',
    })
    expect(chatStore.startStreaming).toHaveBeenCalledWith(
      conversation.id,
      'generated-message-1',
      expect.any(AbortController),
    )
    expect(chatStore.appendStreamingContent).toHaveBeenCalledWith(
      conversation.id,
      'generated-message-1',
      'Replacement answer',
    )
    expect(chatStore.finishStreaming).toHaveBeenCalledWith(
      conversation.id,
      'generated-message-1',
      'stop',
    )
    expect(apiMocks.streamChatCompletion.mock.calls[0]?.[2]).toMatchObject({
      attemptId: 'attempt-generated',
      signal: expect.any(AbortSignal),
    })
  })

  it('成功流在响应头到达时持久化 receipt，并用最终服务端回执更新消息与余额', async () => {
    const conversation: ChatConversation = {
      id: 'receipt-conversation',
      userId: '7',
      title: 'Receipt conversation',
      model: 'gpt-5',
      messages: [
        {
          id: 'user-1',
          role: 'user',
          content: 'Question',
          createdAt: 1,
          status: 'complete',
        },
        {
          id: 'assistant-1',
          role: 'assistant',
          content: 'Old answer',
          createdAt: 2,
          status: 'complete',
        },
      ],
      createdAt: 1,
      updatedAt: 2,
    }
    chatStore.userId = '7'
    chatStore.hydrated = true
    chatStore.conversations = [conversation]
    chatStore.activeConversationId = conversation.id
    chatStore.activeConversation = conversation
    apiMocks.streamChatCompletion.mockImplementationOnce(async (
      _request: unknown,
      handlers: { onReceiptId?: (receiptId: string) => void },
    ) => {
      handlers.onReceiptId?.('receipt-final')
      return {
        receivedDone: true,
        finishReason: 'stop',
        usage: null,
        receiptId: 'receipt-final',
      }
    })
    apiMocks.pollChatReceipt.mockResolvedValueOnce({
      receiptId: 'receipt-final',
      status: 'charged',
      usageLogId: 99,
      model: 'gpt-5.5-2026-07-01',
      inputTokens: 100,
      outputTokens: 20,
      cacheCreationTokens: 5,
      cacheReadTokens: 40,
      totalTokens: 125,
      grossCost: 0.003,
      chargedAmount: 0.002,
      billingType: 0,
      balanceBefore: 0.752,
      balanceAfter: 0.75,
      createdAt: '2026-07-25T01:02:03Z',
    })

    const view = await mountView()
    await view.get('[data-test="retry-message"]').trigger('click')
    await flushPromises()

    expect(chatStore.updateMessage).toHaveBeenCalledWith(
      conversation.id,
      'generated-message-1',
      {
        receiptId: 'receipt-final',
        settlementStatus: 'pending',
      },
    )
    expect(chatStore.updateMessage).toHaveBeenCalledWith(
      conversation.id,
      'generated-message-1',
      expect.objectContaining({
        receiptId: 'receipt-final',
        settlementStatus: 'charged',
        usageLogId: 99,
        actualModel: 'gpt-5.5-2026-07-01',
        inputTokens: 100,
        outputTokens: 20,
        grossCost: 0.003,
        chargedAmount: 0.002,
        balanceBefore: 0.752,
        balanceAfter: 0.75,
      }),
    )
    expect(apiMocks.pollChatReceipt).toHaveBeenCalledTimes(1)
    expect(view.find('[data-test="chat-balance-state"]').exists()).toBe(false)
  })

  it('响应头后发生断连时保留消息错误，并继续核对已确认的 receipt', async () => {
    const conversation: ChatConversation = {
      id: 'disconnect-conversation',
      userId: '7',
      title: 'Disconnected stream',
      model: 'gpt-5',
      messages: [
        {
          id: 'user-1',
          role: 'user',
          content: 'Question',
          createdAt: 1,
          status: 'complete',
        },
        {
          id: 'assistant-1',
          role: 'assistant',
          content: '',
          createdAt: 2,
          status: 'error',
        },
      ],
      createdAt: 1,
      updatedAt: 2,
    }
    chatStore.userId = '7'
    chatStore.hydrated = true
    chatStore.conversations = [conversation]
    chatStore.activeConversationId = conversation.id
    chatStore.activeConversation = conversation
    apiMocks.streamChatCompletion.mockImplementationOnce(async (
      _request: unknown,
      handlers: { onReceiptId?: (receiptId: string) => void },
    ) => {
      handlers.onReceiptId?.('receipt-disconnect')
      throw new ChatAPIError('Connection lost', { code: 'NETWORK_ERROR' })
    })
    apiMocks.pollChatReceipt.mockResolvedValueOnce({
      receiptId: 'receipt-disconnect',
      status: 'charged',
      chargedAmount: 0.001,
      balanceBefore: 2,
      balanceAfter: 1.999,
    })

    const view = await mountView()
    await view.get('[data-test="retry-message"]').trigger('click')
    await flushPromises()

    expect(chatStore.failStreaming).toHaveBeenCalledWith(
      conversation.id,
      'generated-message-1',
      'Connection lost',
      'NETWORK_ERROR',
    )
    expect(apiMocks.pollChatReceipt).toHaveBeenCalledWith(
      'receipt-disconnect',
      { signal: expect.any(AbortSignal) },
    )
    expect(conversation.messages[2]).toMatchObject({
      status: 'error',
      receiptId: 'receipt-disconnect',
      settlementStatus: 'charged',
      chargedAmount: 0.001,
      balanceAfter: 1.999,
    })
  })

  it('恢复历史时自动补查持久化的 pending receipt', async () => {
    const conversation: ChatConversation = {
      id: 'pending-conversation',
      userId: '7',
      title: 'Pending receipt',
      model: 'gpt-5',
      messages: [{
        id: 'assistant-pending',
        role: 'assistant',
        content: 'Persisted answer',
        createdAt: 1,
        status: 'complete',
        attemptId: 'attempt-persisted',
        receiptId: 'receipt-persisted',
        settlementStatus: 'pending',
      }],
      createdAt: 1,
      updatedAt: 1,
    }
    chatStore.userId = '7'
    chatStore.hydrated = true
    chatStore.conversations = [conversation]
    chatStore.activeConversationId = conversation.id
    chatStore.activeConversation = conversation
    apiMocks.pollChatReceipt.mockResolvedValueOnce({
      receiptId: 'receipt-persisted',
      status: 'subscription',
      model: 'gpt-5.5',
      chargedAmount: 0,
      billingType: 1,
    })

    await mountView()
    await flushPromises()

    expect(apiMocks.pollChatReceipt).toHaveBeenCalledWith(
      'receipt-persisted',
      { signal: expect.any(AbortSignal) },
    )
    expect(conversation.messages[0]).toMatchObject({
      settlementStatus: 'subscription',
      actualModel: 'gpt-5.5',
      chargedAmount: 0,
      billingType: 1,
    })
  })

  it('409 duplicate attempt 接回原 receipt，不触发第三次上游请求', async () => {
    const conversation: ChatConversation = {
      id: 'duplicate-conversation',
      userId: '7',
      title: 'Duplicate attempt',
      model: 'gpt-5',
      messages: [
        {
          id: 'user-1',
          role: 'user',
          content: 'Question',
          createdAt: 1,
          status: 'complete',
        },
        {
          id: 'assistant-1',
          role: 'assistant',
          content: '',
          createdAt: 2,
          status: 'error',
        },
      ],
      createdAt: 1,
      updatedAt: 2,
    }
    chatStore.userId = '7'
    chatStore.hydrated = true
    chatStore.conversations = [conversation]
    chatStore.activeConversationId = conversation.id
    chatStore.activeConversation = conversation
    apiMocks.streamChatCompletion.mockRejectedValueOnce(new ChatAPIError(
      'Already submitted',
      {
        status: 409,
        code: 'CHAT_ATTEMPT_ALREADY_SUBMITTED',
        metadata: { receipt_id: 'receipt-original' },
      },
    ))
    apiMocks.pollChatReceipt.mockResolvedValueOnce({
      receiptId: 'receipt-original',
      status: 'not_charged',
      chargedAmount: 0,
    })

    const view = await mountView()
    await view.get('[data-test="retry-message"]').trigger('click')
    await flushPromises()

    expect(apiMocks.streamChatCompletion).toHaveBeenCalledTimes(1)
    expect(apiMocks.pollChatReceipt).toHaveBeenCalledWith(
      'receipt-original',
      { signal: expect.any(AbortSignal) },
    )
    expect(chatStore.failStreaming).toHaveBeenCalledWith(
      conversation.id,
      'generated-message-1',
      'chat.errors.attemptAlreadySubmitted',
      'CHAT_ATTEMPT_ALREADY_SUBMITTED',
    )
    expect(conversation.messages[2]).toMatchObject({
      attemptId: 'attempt-generated',
      receiptId: 'receipt-original',
      settlementStatus: 'not_charged',
      chargedAmount: 0,
    })
  })

  it('后续消息上传新增信封与所选推理强度，不再由客户端上传历史上下文', async () => {
    const conversation: ChatConversation = {
      id: 'context-conversation',
      userId: '7',
      title: 'Context filtering',
      model: 'gpt-5',
      messages: [
        {
          id: 'user-1',
          role: 'user',
          content: 'Original question',
          createdAt: 1,
          status: 'complete',
        },
        {
          id: 'assistant-old',
          role: 'assistant',
          content: 'Superseded answer',
          createdAt: 2,
          status: 'complete',
          excludedFromContext: true,
          supersededByMessageId: 'assistant-current',
        },
        {
          id: 'assistant-current',
          role: 'assistant',
          content: 'Current answer',
          createdAt: 3,
          status: 'complete',
        },
      ],
      createdAt: 1,
      updatedAt: 3,
    }
    chatStore.userId = '7'
    chatStore.hydrated = true
    chatStore.conversations = [conversation]
    chatStore.activeConversationId = conversation.id
    chatStore.activeConversation = conversation

    const view = await mountView()
    view.findComponent(ChatModelSettingsStub).vm.$emit('update:reasoningEffort', 'high')
    await nextTick()
    view.findComponent(ChatComposerStub).vm.$emit('send', 'Follow up')
    await flushPromises()

    expect(apiMocks.streamChatCompletion.mock.calls[0]?.[0]).toEqual({
      conversationId: 'context-conversation',
      model: 'gpt-5',
      reasoningEffort: 'high',
      expectedHeadMessageId: 'assistant-current',
      userMessage: {
        id: 'generated-message-1',
        content: 'Follow up',
      },
      assistantMessageId: 'generated-message-2',
    })
  })

  it('614px fine-pointer 窄桌面隐藏 docked Sidebar，并用完整 overlay 替代移动抽屉', async () => {
    vi.stubGlobal('matchMedia', vi.fn((query: string) => ({
      matches: query === WORKSPACE_NARROW_SIDEBAR_MEDIA_QUERY,
      media: query,
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    } as unknown as MediaQueryList)))
    chatStore.userId = '7'
    chatStore.hydrated = true
    const view = await mountView()
    const appStore = useAppStore()

    expect(appStore.workspaceNarrowSidebar).toBe(true)
    expect(appStore.workspaceMobileDrawer).toBe(false)
    expect(appStore.workspaceNarrowSidebarOpen).toBe(false)
    expect(view.get('.chat-workspace__history--desktop').attributes('style')).toContain('display: none')
    expect(view.find('.chat-workspace__drawer').exists()).toBe(false)
    expect(view.get('.chat-workspace__main').attributes('inert')).toBeUndefined()

    appStore.setSidebarCollapsed(true)
    appStore.setWorkspaceNarrowSidebarOpen(true)
    await nextTick()
    await flushPromises()

    expect(window.matchMedia).toHaveBeenCalledWith(WORKSPACE_MOBILE_DRAWER_MEDIA_QUERY)
    expect(window.matchMedia).toHaveBeenCalledWith(WORKSPACE_NARROW_SIDEBAR_MEDIA_QUERY)
    expect(view.find('.chat-workspace__drawer').exists()).toBe(false)
    expect(appStore.workspaceNarrowSidebarOpen).toBe(true)
    expect(appStore.sidebarCollapsed).toBe(true)
    expect(view.get('.chat-workspace__main').attributes('inert')).toBeDefined()
    expect(view.get('.chat-workspace__main').attributes('aria-hidden')).toBe('true')

    const layer = document.body.querySelector<HTMLElement>(
      '[data-testid="workspace-sidebar-overlay-layer"]',
    )
    const overlaySidebar = document.getElementById('workspace-chat-sidebar-overlay')
    const closeButton = document.body.querySelector<HTMLButtonElement>(
      '.workspace-sidebar-overlay-layer__close',
    )
    expect(layer?.classList.contains('workspace-sidebar-overlay-layer--open')).toBe(true)
    expect(overlaySidebar?.dataset.mobile).toBe('false')
    expect(overlaySidebar?.dataset.overlay).toBe('true')
    expect(closeButton?.querySelector('.workspace-responsive-sidebar-icon--close')).not.toBeNull()

    const overlayHistory = view.findAllComponents(ChatHistoryPanelStub)
      .find(panel => panel.props('overlay') === true)
    overlayHistory?.vm.$emit('close')
    await nextTick()

    expect(overlayHistory).toBeDefined()
    expect(appStore.workspaceNarrowSidebarOpen).toBe(false)
    expect(appStore.sidebarCollapsed).toBe(true)
    expect(view.get('.chat-workspace__main').attributes('inert')).toBeUndefined()
    expect(view.get('.chat-workspace__main').attributes('aria-hidden')).toBeUndefined()
  })

  it('移动历史抽屉隔离 scrim 焦点，并在 Escape 与 Tab 导航中管理焦点', async () => {
    vi.stubGlobal('matchMedia', vi.fn((query: string) => ({
      matches: query === WORKSPACE_MOBILE_DRAWER_MEDIA_QUERY,
      media: query,
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    } as unknown as MediaQueryList)))
    vi.spyOn(HTMLElement.prototype, 'getClientRects').mockReturnValue([
      { width: 1, height: 1 },
    ] as unknown as DOMRectList)
    chatStore.userId = '7'
    chatStore.hydrated = true
    const view = await mountView()
    const trigger = view.get('.chat-mobile-actions__history-button').element as HTMLButtonElement

    await view.get('.chat-mobile-actions__history-button').trigger('click')
    await nextTick()

    const drawer = view.get('.chat-workspace__drawer')
    const scrim = drawer.get('.chat-workspace__scrim').element as HTMLButtonElement
    const first = drawer.get('[data-test="history-first"]').element as HTMLButtonElement
    const last = drawer.get('[data-test="history-last"]').element as HTMLButtonElement
    expect(scrim.tabIndex).toBe(-1)
    expect(document.activeElement).toBe(first)
    expect(document.activeElement).not.toBe(scrim)

    await drawer.trigger('keydown', { key: 'Tab', shiftKey: true })
    expect(document.activeElement).toBe(last)

    await drawer.trigger('keydown', { key: 'Tab' })
    expect(document.activeElement).toBe(first)

    await drawer.trigger('keydown', { key: 'Escape' })
    await nextTick()
    expect(view.find('.chat-workspace__drawer').exists()).toBe(false)
    expect(document.activeElement).toBe(trigger)
  })

  it('历史抽屉打开后切换到桌面断点会解除主区 inert 并恢复焦点', async () => {
    let mobileMatches = true
    let breakpointListener: ((event: MediaQueryListEvent) => void) | undefined
    vi.stubGlobal('matchMedia', vi.fn((query: string) => ({
      get matches() {
        return query === WORKSPACE_MOBILE_DRAWER_MEDIA_QUERY && mobileMatches
      },
      media: query,
      onchange: null,
      addEventListener: vi.fn((_type: string, listener: (event: MediaQueryListEvent) => void) => {
        if (query === WORKSPACE_MOBILE_DRAWER_MEDIA_QUERY) breakpointListener = listener
      }),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    } as unknown as MediaQueryList)))
    chatStore.userId = '7'
    chatStore.hydrated = true
    const view = await mountView()

    await view.get('.chat-mobile-actions__history-button').trigger('click')
    await nextTick()
    expect(view.get('.chat-workspace__main').attributes('inert')).toBeDefined()

    mobileMatches = false
    breakpointListener?.({ matches: false } as MediaQueryListEvent)
    await nextTick()
    await nextTick()

    expect(view.find('.chat-workspace__drawer').exists()).toBe(false)
    expect(view.get('.chat-workspace__main').element.hasAttribute('inert')).toBe(false)
    expect(view.get('.chat-workspace__main').attributes('aria-hidden')).toBeUndefined()
    expect(document.activeElement).toBe(view.get('.chat-workspace__main').element)
  })

  it('支持纯附件发送，并将附件同时写入本地消息和 completion 信封', async () => {
    const conversation: ChatConversation = {
      id: 'attachment-conversation',
      userId: '7',
      title: 'New conversation',
      model: 'gpt-5.5',
      messages: [],
      createdAt: 1,
      updatedAt: 1,
    }
    chatStore.userId = '7'
    chatStore.hydrated = true
    chatStore.conversations = [conversation]
    chatStore.activeConversationId = conversation.id
    chatStore.activeConversation = conversation
    attachmentPickerReady.value = [{
      id: 'attachment-doc-1',
      name: 'brief.pdf',
      kind: 'document',
      mimeType: 'application/pdf',
      size: 2048,
      status: 'ready',
      expiresAt: '2099-01-01T00:00:00Z',
      pageCount: 3,
    }]

    const view = await mountView()
    expect(view.get('[data-test="chat-attachment-picker"]')
      .attributes('data-supports-vision')).toBe('true')

    view.findComponent(ChatComposerStub).vm.$emit('send', '')
    await flushPromises()

    expect(conversation.messages[0]).toMatchObject({
      role: 'user',
      content: '',
      attachments: attachmentPickerReady.value,
    })
    expect(apiMocks.streamChatCompletion.mock.calls[0]?.[0]).toMatchObject({
      conversationId: conversation.id,
      model: 'gpt-5.5',
      reasoningEffort: 'low',
      userMessage: {
        id: conversation.messages[0]?.id,
        content: '',
        attachmentIds: ['attachment-doc-1'],
      },
    })
    expect(attachmentPickerCommitAll).toHaveBeenCalledTimes(1)
  })

  it('附件发送遇到目录不可用时保留消息流错误，不回滚已展示的消息', async () => {
    const conversation: ChatConversation = {
      id: 'attachment-model-error',
      userId: '7',
      title: 'Model error',
      model: 'gpt-5.6-sol',
      messages: [],
      createdAt: 1,
      updatedAt: 1,
    }
    chatStore.userId = '7'
    chatStore.hydrated = true
    chatStore.conversations = [conversation]
    chatStore.activeConversationId = conversation.id
    chatStore.activeConversation = conversation
    attachmentPickerReady.value = [{
      id: 'attachment-model-error-1',
      name: 'prompt.png',
      kind: 'image',
      mimeType: 'image/png',
      size: 1024,
      status: 'ready',
      expiresAt: '2099-01-01T00:00:00Z',
    }]
    apiMocks.streamChatCompletion.mockRejectedValueOnce(new ChatAPIError('Unavailable', {
      status: 503,
      code: 'CHAT_CATALOG_UNAVAILABLE',
    }))

    const view = await mountView()
    view.findComponent(ChatComposerStub).vm.$emit('send', '')
    await flushPromises()

    expect(chatStore.failStreaming).toHaveBeenCalledWith(
      conversation.id,
      expect.any(String),
      'chat.errors.modelUnavailable',
      'CHAT_CATALOG_UNAVAILABLE',
    )
    expect(chatStore.removeMessages).not.toHaveBeenCalled()
    expect(conversation.messages).toHaveLength(2)
    expect(conversation.messages[1]).toMatchObject({
      role: 'assistant',
      status: 'error',
      errorMessage: 'chat.errors.modelUnavailable',
    })
    expect(attachmentPickerCommitAll).not.toHaveBeenCalled()
  })

  it('点击发送瞬间附件过期时整轮拒绝且保留文字草稿', async () => {
    const conversation: ChatConversation = {
      id: 'expired-click-conversation',
      userId: '7',
      title: 'Expiry boundary',
      model: 'gpt-5',
      messages: [],
      createdAt: 1,
      updatedAt: 1,
    }
    chatStore.userId = '7'
    chatStore.hydrated = true
    chatStore.conversations = [conversation]
    chatStore.activeConversationId = conversation.id
    chatStore.activeConversation = conversation
    attachmentPickerReady.value = []

    const view = await mountView()
    const composer = view.findComponent(ChatComposerStub)
    const picker = view.findComponent(ChatAttachmentPickerStub)
    composer.vm.$emit('update:modelValue', '这段文字必须保留')
    picker.vm.$emit('change', [{
      key: 'expired-draft',
      file: new File(['pdf'], 'expired.pdf', { type: 'application/pdf' }),
      kind: 'document',
      state: 'ready',
      progress: 100,
      attachment: {
        id: 'expired-attachment',
        name: 'expired.pdf',
        kind: 'document',
        mimeType: 'application/pdf',
        size: 3,
        status: 'expired',
        expiresAt: '2026-08-06T00:00:00Z',
      },
    }])
    await nextTick()
    const acknowledge = vi.fn()

    composer.vm.$emit('send', '这段文字必须保留', acknowledge)
    await flushPromises()

    expect(acknowledge).toHaveBeenCalledWith(false)
    expect(composer.props('modelValue')).toBe('这段文字必须保留')
    expect(chatStore.addMessage).not.toHaveBeenCalled()
    expect(apiMocks.streamChatCompletion).not.toHaveBeenCalled()
    expect(attachmentPickerCommitAll).not.toHaveBeenCalled()
  })

  it('服务端接纳前无法启动请求时保留附件和草稿并回滚乐观消息', async () => {
    const conversation: ChatConversation = {
      id: 'pre-accept-failure',
      userId: '7',
      title: 'Pre-accept failure',
      model: 'gpt-5',
      messages: [],
      createdAt: 1,
      updatedAt: 1,
    }
    const readyAttachment: ChatAttachment = {
      id: 'attachment-pre-accept',
      name: 'brief.pdf',
      kind: 'document',
      mimeType: 'application/pdf',
      size: 2048,
      status: 'ready',
      expiresAt: '2099-01-01T00:00:00Z',
    }
    chatStore.userId = '7'
    chatStore.hydrated = true
    chatStore.conversations = [conversation]
    chatStore.activeConversationId = conversation.id
    chatStore.activeConversation = conversation
    attachmentPickerReady.value = [readyAttachment]
    chatStore.startStreaming.mockReturnValueOnce(false)

    const view = await mountView()
    const composer = view.findComponent(ChatComposerStub)
    const picker = view.findComponent(ChatAttachmentPickerStub)
    composer.vm.$emit('update:modelValue', '网络失败也保留')
    picker.vm.$emit('change', [{
      key: 'pre-accept-draft',
      file: new File(['pdf'], 'brief.pdf', { type: 'application/pdf' }),
      kind: 'document',
      state: 'ready',
      progress: 100,
      attachment: readyAttachment,
    }])
    await nextTick()
    const acknowledge = vi.fn()

    composer.vm.$emit('send', '网络失败也保留', acknowledge)
    await flushPromises()

    expect(apiMocks.streamChatCompletion).not.toHaveBeenCalled()
    expect(attachmentPickerCommitAll).not.toHaveBeenCalled()
    expect(chatStore.removeMessages).toHaveBeenCalledWith(
      conversation.id,
      ['generated-message-1', 'generated-message-2'],
    )
    expect(conversation.messages).toEqual([])
    expect(acknowledge).toHaveBeenCalledWith(false)
    expect(composer.props('modelValue')).toBe('网络失败也保留')
  })

  it('切换会话时清理尚未绑定的附件', async () => {
    const first: ChatConversation = {
      id: 'attachment-context-a',
      userId: '7',
      title: 'A',
      model: 'gpt-5',
      messages: [],
      createdAt: 1,
      updatedAt: 2,
    }
    const second = { ...first, id: 'attachment-context-b', title: 'B', updatedAt: 1 }
    chatStore.userId = '7'
    chatStore.hydrated = true
    chatStore.conversations = [first, second]
    chatStore.activeConversationId = first.id
    chatStore.activeConversation = first

    await mountView()
    attachmentPickerDiscardAll.mockClear()
    chatStore.activeConversationId = second.id
    chatStore.activeConversation = second
    await nextTick()
    await flushPromises()

    expect(attachmentPickerDiscardAll).toHaveBeenCalledTimes(1)
  })
})
