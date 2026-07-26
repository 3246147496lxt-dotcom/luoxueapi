import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { nextTick, ref } from 'vue'
import type { ChatConversation, ChatMessage } from '@/types/chat'

const apiMocks = vi.hoisted(() => ({
  createChatAttemptId: vi.fn(),
  getChatModels: vi.fn(),
  isAbortError: vi.fn(),
  pollChatReceipt: vi.fn(),
  streamChatCompletion: vi.fn(),
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
    getChatModels: apiMocks.getChatModels,
    isAbortError: apiMocks.isAbortError,
    pollChatReceipt: apiMocks.pollChatReceipt,
    streamChatCompletion: apiMocks.streamChatCompletion,
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
import { useChatStore } from '@/stores/chat'
import { ChatAPIError } from '@/api/chat'
import ChatView from '../ChatView.vue'

const AppLayoutStub = {
  props: ['variant'],
  template: '<div data-test="app-layout"><slot /></div>',
}

const ChatComposerStub = {
  props: ['modelValue', 'streaming', 'disabled', 'insufficientBalance'],
  emits: ['update:modelValue', 'send', 'stop'],
  setup(_props: unknown, { expose }: { expose: (value: { focus: () => void }) => void }) {
    const input = ref<HTMLTextAreaElement | null>(null)
    expose({ focus: () => input.value?.focus() })
    return { input }
  },
  template: `
    <div
      data-test="chat-composer"
      :data-disabled="String(disabled)"
      :data-streaming="String(streaming)"
      :data-insufficient-balance="String(insufficientBalance)"
    >
      <textarea ref="input" data-test="composer-input"></textarea>
      <button type="button" data-test="composer-stop" @click="$emit('stop')">Stop</button>
    </div>
  `,
}

const SelectStub = {
  props: ['modelValue', 'options', 'disabled'],
  emits: ['update:modelValue'],
  template: `
    <div
      data-test="model-select"
      :data-disabled="String(disabled)"
      :data-option-count="String(options.length)"
      :data-first-option="options[0]?.value || ''"
    />
  `,
}

const ChatHistoryPanelStub = {
  props: {
    mobile: {
      type: Boolean,
      default: false,
    },
  },
  emits: ['new', 'close', 'select', 'rename', 'delete', 'clear'],
  template: `
    <aside data-test="chat-history" :data-mobile="String(!!mobile)">
      <template v-if="mobile">
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
  props: ['show', 'title'],
  emits: ['close'],
  template: '<div v-if="show"><slot /><slot name="footer" /></div>',
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
      stubs: {
        AppLayout: AppLayoutStub,
        BaseDialog: BaseDialogStub,
        ChatComposer: ChatComposerStub,
        ChatHistoryPanel: ChatHistoryPanelStub,
        ChatMessageItem: ChatMessageItemStub,
        Icon: IconStub,
        RouterLink: RouterLinkStub,
        Select: SelectStub,
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
    generatedMessageId = 0
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
      models: [{ id: 'gpt-5', display_name: 'GPT-5', recommended: true }],
      balance: 10,
    })
    apiMocks.createChatAttemptId.mockReturnValue('attempt-generated')
    apiMocks.pollChatReceipt.mockResolvedValue({
      receiptId: 'receipt-pending',
      status: 'pending',
    })
    apiMocks.streamChatCompletion.mockResolvedValue({
      receivedDone: true,
      finishReason: 'stop',
      usage: null,
      receiptId: null,
    })
  })

  afterEach(() => {
    wrapper?.unmount()
    wrapper = undefined
    document.body.innerHTML = ''
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
    vi.unstubAllGlobals()
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

  it('目录先于 history hydration 返回时，将恢复出的退役模型纠正为当前推荐模型', async () => {
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

    expect(chatStore.setConversationModel).toHaveBeenCalledWith(conversation.id, 'gpt-5')
    expect(conversation.model).toBe('gpt-5')
    expect(view.get('[data-test="chat-composer"]').attributes('data-disabled')).toBe('false')
  })

  it('切换到使用目录外模型的历史会话时自动回退到当前推荐模型', async () => {
    const current: ChatConversation = {
      id: 'current-conversation',
      userId: '7',
      title: 'Current conversation',
      model: 'gpt-5',
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

    expect(chatStore.setConversationModel).toHaveBeenCalledWith(retired.id, 'gpt-5')
    expect(retired.model).toBe('gpt-5')
  })

  it('目录 membership 无法确认时即使收到 send 事件也拒绝创建请求', async () => {
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
    chatStore.setConversationModel.mockReturnValue(false)

    const view = await mountView()
    expect(view.get('[data-test="chat-composer"]').attributes('data-disabled')).toBe('true')

    view.findComponent(ChatComposerStub).vm.$emit('send', 'must not be sent')
    await nextTick()

    expect(chatStore.addMessage).not.toHaveBeenCalled()
    expect(apiMocks.streamChatCompletion).not.toHaveBeenCalled()
  })

  it('空会话在目录和 history 就绪后聚焦 composer', async () => {
    chatStore.userId = '7'
    chatStore.hydrated = true

    const view = await mountView()

    expect(document.activeElement).toBe(view.get('[data-test="composer-input"]').element)
  })

  it('切换账号后忽略旧账号迟到的目录响应', async () => {
    let resolveFirst!: (value: unknown) => void
    apiMocks.getChatModels
      .mockImplementationOnce(() => new Promise((resolve) => {
        resolveFirst = resolve
      }))
      .mockResolvedValueOnce({
        models: [{ id: 'gpt-new', display_name: 'GPT New', recommended: true }],
        balance: 3,
      })

    const view = await mountView()
    expect(apiMocks.getChatModels).toHaveBeenCalledTimes(1)

    authStore.user = { id: 8, balance: 2 }
    chatStore.userId = '8'
    chatStore.hydrated = true
    await flushPromises()
    await nextTick()

    expect(apiMocks.getChatModels).toHaveBeenCalledTimes(2)
    expect(view.get('[data-test="model-select"]').attributes('data-first-option')).toBe('gpt-new')
    expect(view.get('.chat-toolbar__balance strong').text()).toBe('3.00')

    resolveFirst({
      models: [{ id: 'gpt-old', display_name: 'GPT Old', recommended: true }],
      balance: 99,
    })
    await flushPromises()
    await nextTick()

    expect(view.get('[data-test="model-select"]').attributes('data-first-option')).toBe('gpt-new')
    expect(view.get('.chat-toolbar__balance strong').text()).toBe('3.00')
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

  it('账户刷新后的余额淘汰旧目录余额快照', async () => {
    chatStore.userId = '7'
    chatStore.hydrated = true
    apiMocks.getChatModels.mockResolvedValueOnce({
      models: [{ id: 'gpt-5', display_name: 'GPT-5', recommended: true }],
      balance: 3,
    })
    const view = await mountView()

    expect(view.get('.chat-toolbar__balance strong').text()).toBe('3.00')

    authStore.user = { id: 7, balance: 6 }
    await nextTick()

    expect(view.get('.chat-toolbar__balance strong').text()).toBe('6.00')
  })

  it('空模型目录显示可重试错误，重试成功后恢复 composer', async () => {
    chatStore.userId = '7'
    chatStore.hydrated = true
    apiMocks.getChatModels
      .mockResolvedValueOnce({ models: [], balance: 10 })
      .mockResolvedValueOnce({
        models: [{ id: 'gpt-5', display_name: 'GPT-5', recommended: true }],
        balance: 10,
      })

    const view = await mountView()

    expect(view.get('.chat-catalog-error').text()).toContain('chat.errors.noModelsAvailable')
    expect(view.get('[data-test="chat-composer"]').attributes('data-disabled')).toBe('true')
    expect(view.get('[data-test="model-select"]').attributes('data-option-count')).toBe('0')

    await view.get('.chat-catalog-error button').trigger('click')
    await flushPromises()
    await nextTick()

    expect(apiMocks.getChatModels).toHaveBeenCalledTimes(2)
    expect(view.find('.chat-catalog-error').exists()).toBe(false)
    expect(view.get('[data-test="model-select"]').attributes('data-option-count')).toBe('1')
    expect(view.get('[data-test="chat-composer"]').attributes('data-disabled')).toBe('false')
  })

  it('目录接口以余额不足拒绝时，以服务端结果覆盖当前用户的正余额状态', async () => {
    chatStore.userId = '7'
    chatStore.hydrated = true
    authStore.user = { id: 7, balance: 10 }
    apiMocks.getChatModels.mockRejectedValueOnce(new ChatAPIError('Insufficient balance', {
      status: 403,
      code: 'INSUFFICIENT_BALANCE',
    }))

    const view = await mountView()

    expect(view.get('.chat-toolbar__balance strong').text()).toBe('10.00')
    expect(view.get('[data-test="chat-composer"]').attributes('data-insufficient-balance')).toBe('true')
    expect(view.get('.chat-catalog-error').text()).toContain('chat.errors.insufficientBalance')
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

  it('显式停止导致 Abort 后仍刷新同一用户的账户与模型目录', async () => {
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
    authStore.refreshUser.mockImplementation(async () => {
      await Promise.resolve()
      authStore.user = { id: 7, balance: 4 }
      return authStore.user
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
    expect(authStore.refreshUser).toHaveBeenCalledTimes(1)
    expect(apiMocks.getChatModels).toHaveBeenCalledTimes(2)
    expect(view.get('.chat-toolbar__balance strong').text()).toBe('4.00')
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
    expect(view.get('.chat-toolbar__balance strong').text()).toBe('0.75')
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

  it('后续消息只上传新增消息信封，不再由客户端上传历史上下文', async () => {
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
    view.findComponent(ChatComposerStub).vm.$emit('send', 'Follow up')
    await flushPromises()

    expect(apiMocks.streamChatCompletion.mock.calls[0]?.[0]).toEqual({
      conversationId: 'context-conversation',
      model: 'gpt-5',
      expectedHeadMessageId: 'assistant-current',
      userMessage: {
        id: 'generated-message-1',
        content: 'Follow up',
      },
      assistantMessageId: 'generated-message-2',
    })
  })

  it('移动历史抽屉隔离 scrim 焦点，并在 Escape 与 Tab 导航中管理焦点', async () => {
    vi.spyOn(HTMLElement.prototype, 'getClientRects').mockReturnValue([
      { width: 1, height: 1 },
    ] as unknown as DOMRectList)
    chatStore.userId = '7'
    chatStore.hydrated = true
    const view = await mountView()
    const trigger = view.get('.chat-toolbar__history-button').element as HTMLButtonElement

    await view.get('.chat-toolbar__history-button').trigger('click')
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
    let breakpointListener: ((event: MediaQueryListEvent) => void) | undefined
    const mediaQuery = {
      matches: true,
      media: '(max-width: 1023px)',
      onchange: null,
      addEventListener: vi.fn((_type: string, listener: (event: MediaQueryListEvent) => void) => {
        breakpointListener = listener
      }),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    } as unknown as MediaQueryList
    vi.stubGlobal('matchMedia', vi.fn(() => mediaQuery))
    chatStore.userId = '7'
    chatStore.hydrated = true
    const view = await mountView()

    await view.get('.chat-toolbar__history-button').trigger('click')
    await nextTick()
    expect(view.get('.chat-workspace__main').attributes('inert')).toBeDefined()

    breakpointListener?.({ matches: false } as MediaQueryListEvent)
    await nextTick()
    await nextTick()

    expect(view.find('.chat-workspace__drawer').exists()).toBe(false)
    expect(view.get('.chat-workspace__main').element.hasAttribute('inert')).toBe(false)
    expect(view.get('.chat-workspace__main').attributes('aria-hidden')).toBeUndefined()
    expect(document.activeElement).toBe(view.get('.chat-workspace__main').element)
  })
})
