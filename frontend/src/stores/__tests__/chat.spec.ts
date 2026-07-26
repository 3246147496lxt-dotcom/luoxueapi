import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const persistenceMocks = vi.hoisted(() => ({
  load: vi.fn(),
  save: vi.fn(),
  remove: vi.fn(),
}))

const chatApiMocks = vi.hoisted(() => ({
  createChatConversation: vi.fn(),
  deleteChatConversation: vi.fn(),
  getChatAttempt: vi.fn(),
  getChatConversation: vi.fn(),
  getChatConversationMessages: vi.fn(),
  getChatSync: vi.fn(),
  isAbortError: vi.fn(),
  listChatConversations: vi.fn(),
  patchChatConversation: vi.fn(),
  searchChatConversations: vi.fn(),
}))

vi.mock('@/features/chat/persistence', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/features/chat/persistence')>()
  return {
    ...actual,
    chatHistoryPersistence: {
      load: persistenceMocks.load,
      save: persistenceMocks.save,
      remove: persistenceMocks.remove,
    },
  }
})

vi.mock('@/api/chat', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/api/chat')>()
  return {
    ...actual,
    ...chatApiMocks,
  }
})

import { ChatAPIError } from '@/api/chat'
import { mergeChatHistoryStates } from '@/features/chat/persistence'
import type { ChatHistoryMutation } from '@/features/chat/persistence'
import { MAX_CHAT_CONVERSATIONS, useChatStore } from '@/stores/chat'
import type {
  ChatServerConversation,
  CreateChatConversationRequest,
} from '@/types/chat'

interface PersistedConversationRecord extends Record<string, unknown> {
  messages: Array<Record<string, unknown>>
}

interface PersistedStateRecord extends Record<string, unknown> {
  conversations: PersistedConversationRecord[]
}

const persistedBuckets = new Map<string, unknown>()
let saveFailuresRemaining = 0

function cloneValue<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T
}

function createDeferred<T>(): {
  promise: Promise<T>
  resolve: (value: T | PromiseLike<T>) => void
  reject: (reason?: unknown) => void
} {
  let resolve!: (value: T | PromiseLike<T>) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((promiseResolve, promiseReject) => {
    resolve = promiseResolve
    reject = promiseReject
  })
  return { promise, resolve, reject }
}

function persistedState(userId: string): PersistedStateRecord {
  const value = persistedBuckets.get(userId)
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    throw new Error(`Missing persisted chat state for ${userId}`)
  }
  return value as PersistedStateRecord
}

function serverConversation(
  id: string,
  overrides: Partial<ChatServerConversation> = {},
): ChatServerConversation {
  return {
    id,
    title: 'Server conversation',
    model: 'gpt-5',
    revision: 1,
    version: 1,
    headMessageId: null,
    messageCount: 0,
    messages: [],
    createdAt: 1,
    updatedAt: 1,
    ...overrides,
  }
}

function persistedChatState(
  userId: string,
  conversation: PersistedConversationRecord,
  overrides: Record<string, unknown> = {},
): Record<string, unknown> {
  return {
    version: 2,
    clearRevision: 0,
    deletedConversationIds: [],
    activeConversationId: conversation.id,
    conversations: [{ ...conversation, userId }],
    serverVersion: 0,
    outbox: [],
    legacyImportDecision: null,
    legacyConversationIds: [],
    ...overrides,
  }
}

describe('useChatStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    persistedBuckets.clear()
    saveFailuresRemaining = 0
    vi.resetAllMocks()
    persistenceMocks.load.mockImplementation(async (userId: string) => {
      const state = persistedBuckets.get(userId)
      return state === undefined ? null : cloneValue(state)
    })
    persistenceMocks.save.mockImplementation(async (
      userId: string,
      state: unknown,
      mutation: ChatHistoryMutation,
    ) => {
      if (saveFailuresRemaining > 0) {
        saveFailuresRemaining -= 1
        throw new DOMException('temporary transaction failure')
      }
      persistedBuckets.set(userId, cloneValue(mergeChatHistoryStates(
        persistedBuckets.get(userId) ?? null,
        state,
        userId,
        mutation,
      )))
    })
    persistenceMocks.remove.mockImplementation(async (userId: string) => {
      persistedBuckets.delete(userId)
    })
    chatApiMocks.getChatSync.mockResolvedValue({
      changes: [],
      latestVersion: 0,
      nextCursor: null,
      hasMore: false,
    })
    chatApiMocks.listChatConversations.mockResolvedValue({
      items: [],
      nextCursor: null,
      hasMore: false,
    })
    chatApiMocks.searchChatConversations.mockResolvedValue({
      items: [],
      nextCursor: null,
      hasMore: false,
    })
    chatApiMocks.getChatConversationMessages.mockResolvedValue({
      items: [],
      nextBeforePosition: null,
      hasMore: false,
    })
    chatApiMocks.isAbortError.mockReturnValue(false)
    chatApiMocks.deleteChatConversation.mockResolvedValue(undefined)
    chatApiMocks.createChatConversation.mockImplementation(
      async (request: CreateChatConversationRequest) => serverConversation(request.id, {
        title: request.title,
        model: request.model,
      }),
    )
  })

  it('按用户分桶持久化，切换用户时不泄漏会话', async () => {
    const store = useChatStore()
    await store.hydrate(101)
    const first = store.createConversation('gpt-4.1', '用户 101')!
    store.addMessage(first.id, { role: 'user', content: '你好' })
    await store.flushPersistence()

    await store.hydrate('202')
    expect(store.conversations).toEqual([])
    const second = store.createConversation('gpt-4o-mini', '用户 202')!
    await store.flushPersistence()

    await store.hydrate('101')
    expect(store.conversations).toHaveLength(1)
    expect(store.conversations[0].title).toBe('用户 101')
    expect(store.conversations[0].messages[0].content).toBe('你好')
    expect(JSON.stringify(persistedState('202'))).toContain(second.id)
  })

  it('两个标签页并发新建会话时都能持久化', async () => {
    setActivePinia(createPinia())
    const tabA = useChatStore()
    await tabA.hydrate('shared-user')

    setActivePinia(createPinia())
    const tabB = useChatStore()
    await tabB.hydrate('shared-user')

    const conversationA = tabA.createConversation('gpt-4.1', '标签 A')!
    const conversationB = tabB.createConversation('gpt-4.1', '标签 B')!
    await Promise.all([tabA.flushPersistence(), tabB.flushPersistence()])

    expect(persistedState('shared-user').conversations.map(({ id }) => id)).toEqual(
      expect.arrayContaining([conversationA.id, conversationB.id]),
    )
  })

  it('未 hydrate 有效用户时不创建会话', async () => {
    const store = useChatStore()
    expect(store.createConversation('gpt-4.1')).toBeNull()

    await store.hydrate('  ')
    expect(store.createConversation('gpt-4.1')).toBeNull()
    expect(store.hydrated).toBe(true)
  })

  it('支持会话 CRUD、选择和模型切换', async () => {
    const store = useChatStore()
    await store.hydrate('user-a')
    const first = store.createConversation('gpt-4.1', '第一个')!
    const second = store.createConversation('gpt-4o-mini', '第二个')!

    expect(store.activeConversationId).toBe(second.id)
    expect(store.selectConversation(first.id)).toBe(true)
    expect(store.renameConversation(first.id, '  新标题  ')).toBe(true)
    expect(store.setConversationModel(first.id, ' gpt-5 ')).toBe(true)
    expect(store.activeConversation?.title).toBe('新标题')
    expect(store.activeConversation?.model).toBe('gpt-5')
    expect(store.renameConversation(first.id, '  ')).toBe(false)
    expect(store.selectConversation('missing')).toBe(false)

    expect(store.deleteConversation(first.id)).toBe(true)
    expect(store.activeConversationId).toBe(second.id)
    expect(store.deleteConversation('missing')).toBe(false)

    store.clearConversations()
    await store.flushPersistence()
    expect(store.conversations).toEqual([])
    expect(persistedBuckets.has('user-a')).toBe(true)
    expect(persistedState('user-a').conversations).toEqual([])
    expect(persistenceMocks.remove).not.toHaveBeenCalled()
  })

  it('瞬时写入失败后按原顺序重放删除和清空意图', async () => {
    const store = useChatStore()
    await store.hydrate('retry-user')
    const deleted = store.createConversation('gpt-4.1', '待删除')!
    await store.flushPersistence()

    saveFailuresRemaining = 1
    expect(store.deleteConversation(deleted.id)).toBe(true)
    const afterDelete = store.createConversation('gpt-4.1', '删除后的新会话')!
    await store.flushPersistence()

    expect(persistedState('retry-user').conversations.map(({ id }) => id)).toEqual([afterDelete.id])
    expect(store.persistenceAvailable).toBe(true)

    saveFailuresRemaining = 1
    store.clearConversations()
    const afterClear = store.createConversation('gpt-4.1', '清空后的新会话')!
    await store.flushPersistence()

    expect(persistedState('retry-user').conversations.map(({ id }) => id)).toEqual([afterClear.id])
    expect(store.persistenceAvailable).toBe(true)
  })

  it('最多保留 50 个最近会话', async () => {
    const store = useChatStore()
    await store.hydrate('limited-user')

    for (let index = 0; index < MAX_CHAT_CONVERSATIONS + 3; index += 1) {
      store.createConversation('gpt-4.1', `会话 ${index}`)
    }
    await store.flushPersistence()

    expect(store.conversations).toHaveLength(MAX_CHAT_CONVERSATIONS)
    expect(store.conversations[0].title).toBe('会话 52')
    expect(store.conversations.some(({ title }) => title === '会话 0')).toBe(false)
    expect(persistedState('limited-user').conversations).toHaveLength(MAX_CHAT_CONVERSATIONS)
  })

  it('只持久化白名单会话和消息字段', async () => {
    const store = useChatStore()
    await store.hydrate('safe-user')
    const conversation = store.createConversation('gpt-4.1')!
    ;(conversation as unknown as Record<string, unknown>).apiKey = 'must-not-persist'
    const message = store.addMessage(conversation.id, {
      role: 'user',
      content: '正常内容',
    })!
    ;(message as unknown as Record<string, unknown>).authorization = 'Bearer secret'
    store.setConversationModel(conversation.id, 'gpt-4o')
    await store.flushPersistence()

    const persisted = persistedState('safe-user')
    expect(Object.keys(persisted.conversations[0]).sort()).toEqual([
      'createdAt',
      'id',
      'messages',
      'model',
      'title',
      'updatedAt',
      'userId',
    ])
    expect(persisted.conversations[0].apiKey).toBeUndefined()
    expect(persisted.conversations[0].messages[0].authorization).toBeUndefined()
    expect(JSON.stringify(persisted)).not.toContain('must-not-persist')
    expect(JSON.stringify(persisted)).not.toContain('Bearer secret')
  })

  it('贯通 assistant attempt、回执和结算字段的创建、patch 与恢复', async () => {
    const store = useChatStore()
    await store.hydrate('receipt-user')
    const conversation = store.createConversation('gpt-5.5')!
    const assistant = store.addMessage(conversation.id, {
      role: 'assistant',
      content: 'Answer',
      status: 'complete',
      attemptId: 'attempt-1',
      receiptId: 'receipt-1',
      settlementStatus: 'pending',
      requestedModel: 'gpt-5.5',
    })!

    expect(store.updateMessage(conversation.id, assistant.id, {
      settlementStatus: 'charged',
      usageLogId: 88,
      actualModel: 'gpt-5.5-2026-07-01',
      inputTokens: 120,
      outputTokens: 40,
      cacheCreationTokens: 10,
      cacheReadTokens: 80,
      totalTokens: 170,
      grossCost: 0.003,
      chargedAmount: 0.0024,
      billingType: 0,
      balanceBefore: 2,
      balanceAfter: 1.9976,
      receiptCreatedAt: '2026-07-25T01:02:03Z',
      excludedFromContext: true,
      supersededByMessageId: 'assistant-2',
    })).toBe(true)
    ;(assistant as unknown as Record<string, unknown>).authorization = 'must-not-persist'
    await store.flushPersistence()

    setActivePinia(createPinia())
    const restored = useChatStore()
    await restored.hydrate('receipt-user')

    expect(restored.activeConversation?.messages[0]).toMatchObject({
      attemptId: 'attempt-1',
      receiptId: 'receipt-1',
      settlementStatus: 'charged',
      usageLogId: 88,
      requestedModel: 'gpt-5.5',
      actualModel: 'gpt-5.5-2026-07-01',
      inputTokens: 120,
      outputTokens: 40,
      cacheCreationTokens: 10,
      cacheReadTokens: 80,
      totalTokens: 170,
      grossCost: 0.003,
      chargedAmount: 0.0024,
      billingType: 0,
      balanceBefore: 2,
      balanceAfter: 1.9976,
      receiptCreatedAt: '2026-07-25T01:02:03Z',
      excludedFromContext: true,
      supersededByMessageId: 'assistant-2',
    })
    expect(
      (restored.activeConversation?.messages[0] as unknown as Record<string, unknown>).authorization,
    ).toBeUndefined()
  })

  it('恢复时清洗数据并将遗留 streaming 消息标记为 stopped', async () => {
    persistedBuckets.set('restore-user', {
      version: 1,
      activeConversationId: 'valid-conversation',
      conversations: [
        {
          id: 'valid-conversation',
          userId: 'restore-user',
          title: '恢复会话',
          model: 'gpt-4.1',
          createdAt: 10,
          updatedAt: 20,
          unexpected: 'drop-me',
          messages: [
            {
              id: 'assistant-1',
              role: 'assistant',
              content: '未完成回答',
              createdAt: 15,
              status: 'streaming',
              secret: 'drop-me-too',
            },
            { id: 'invalid-message' },
          ],
        },
        {
          id: 'foreign',
          userId: 'another-user',
          title: '不应恢复',
          model: 'gpt-4.1',
          messages: [],
          createdAt: 1,
          updatedAt: 30,
        },
      ],
    })

    const store = useChatStore()
    await store.hydrate('restore-user')
    await store.flushPersistence()

    expect(store.conversations).toHaveLength(1)
    expect(store.activeConversation?.messages).toEqual([
      expect.objectContaining({
        id: 'assistant-1',
        status: 'stopped',
        finishReason: 'interrupted',
      }),
    ])
    const sanitized = persistedState('restore-user')
    expect(sanitized.conversations[0].unexpected).toBeUndefined()
    expect(sanitized.conversations[0].messages[0].secret).toBeUndefined()
  })

  it('损坏的持久化数据会被忽略且不影响后续使用', async () => {
    persistedBuckets.set('corrupt-user', 'not-a-chat-state')

    const store = useChatStore()
    await expect(store.hydrate('corrupt-user')).resolves.toBeUndefined()
    await store.flushPersistence()

    expect(store.conversations).toEqual([])
    expect(persistedBuckets.has('corrupt-user')).toBe(false)
    expect(store.createConversation('gpt-4.1')).not.toBeNull()
    expect(store.persistenceAvailable).toBe(true)
    await store.flushPersistence()
  })

  it('异步持久化不可用时仍可使用内存状态', async () => {
    persistenceMocks.load.mockRejectedValue(new DOMException('denied'))
    persistenceMocks.save.mockRejectedValue(new DOMException('quota'))
    persistenceMocks.remove.mockRejectedValue(new DOMException('denied'))

    const store = useChatStore()
    await expect(store.hydrate('memory-only')).resolves.toBeUndefined()
    const conversation = store.createConversation('gpt-4.1')
    await store.flushPersistence()

    expect(conversation).not.toBeNull()
    expect(store.conversations).toHaveLength(1)
    expect(store.persistenceAvailable).toBe(false)
  })

  it('首次读取瞬时失败后可通过只读探测恢复持久化状态', async () => {
    persistenceMocks.load.mockRejectedValueOnce(new DOMException('temporarily denied'))

    const store = useChatStore()
    await store.hydrate('probe-recovery')

    expect(store.persistenceAvailable).toBe(false)
    expect(persistenceMocks.load).toHaveBeenCalledTimes(1)

    persistenceMocks.load.mockResolvedValueOnce(null)
    await store.flushPersistence()

    expect(persistenceMocks.load).toHaveBeenCalledTimes(2)
    expect(persistenceMocks.save).not.toHaveBeenCalled()
    expect(store.persistenceAvailable).toBe(true)
  })

  it('只读恢复探测持续失败时保持持久化告警状态', async () => {
    persistenceMocks.load.mockRejectedValue(new DOMException('denied'))

    const store = useChatStore()
    await store.hydrate('probe-unavailable')
    await store.flushPersistence()

    expect(persistenceMocks.load).toHaveBeenCalledTimes(2)
    expect(persistenceMocks.save).not.toHaveBeenCalled()
    expect(store.persistenceAvailable).toBe(false)
  })

  it('切换账号后忽略旧账号迟到的读取成功状态', async () => {
    const staleAccountLoad = createDeferred<unknown | null>()
    persistenceMocks.load.mockImplementation((userId: string) => {
      if (userId === 'account-a') return staleAccountLoad.promise
      return Promise.reject(new DOMException('account-b read denied'))
    })
    persistenceMocks.save.mockRejectedValue(new DOMException('account-b write denied'))

    const store = useChatStore()
    const staleHydration = store.hydrate('account-a')
    await vi.waitFor(() => {
      expect(persistenceMocks.load).toHaveBeenCalledWith('account-a')
    })

    await store.hydrate('account-b')
    expect(store.createConversation('gpt-4.1', '账号 B 会话')).not.toBeNull()
    await store.flushPersistence()
    expect(store.persistenceAvailable).toBe(false)

    staleAccountLoad.resolve(null)
    await staleHydration

    expect(store.userId).toBe('account-b')
    expect(store.conversations).toHaveLength(1)
    expect(store.persistenceAvailable).toBe(false)
  })

  it('账号往返切换后忽略旧代际迟到的恢复探测', async () => {
    const staleProbe = createDeferred<unknown | null>()
    let accountALoadCount = 0
    persistenceMocks.load.mockImplementation((userId: string) => {
      if (userId === 'account-b') return Promise.resolve(null)
      accountALoadCount += 1
      if (accountALoadCount === 2) return staleProbe.promise
      return Promise.reject(new DOMException('account-a read denied'))
    })

    const store = useChatStore()
    await store.hydrate('account-a')
    expect(store.persistenceAvailable).toBe(false)

    const staleFlush = store.flushPersistence()
    await vi.waitFor(() => {
      expect(accountALoadCount).toBe(2)
    })

    await store.hydrate('account-b')
    expect(store.persistenceAvailable).toBe(true)
    await store.hydrate('account-a')
    expect(store.persistenceAvailable).toBe(false)

    staleProbe.resolve(null)
    await staleFlush

    expect(store.userId).toBe('account-a')
    expect(store.persistenceAvailable).toBe(false)
  })

  it('管理完整的流式回答生命周期', async () => {
    const store = useChatStore()
    await store.hydrate('stream-user')
    const conversation = store.createConversation('gpt-4.1')!
    const assistant = store.addMessage(conversation.id, {
      role: 'assistant',
      content: '',
      status: 'streaming',
    })!
    const controller = new AbortController()

    expect(store.startStreaming(conversation.id, assistant.id, controller)).toBe(true)
    await store.flushPersistence()
    const writesAfterStart = persistenceMocks.save.mock.calls.length
    expect(store.isStreaming).toBe(true)
    expect(store.appendStreamingContent(conversation.id, assistant.id, '你好')).toBe(true)
    expect(store.appendStreamingContent(conversation.id, assistant.id, '，世界')).toBe(true)
    expect(persistenceMocks.save).toHaveBeenCalledTimes(writesAfterStart)
    expect(store.finishStreaming(conversation.id, assistant.id, 'stop')).toBe(true)
    await store.flushPersistence()

    expect(assistant.content).toBe('你好，世界')
    expect(assistant.status).toBe('complete')
    expect(assistant.finishReason).toBe('stop')
    expect(store.isStreaming).toBe(false)
    expect(controller.signal.aborted).toBe(false)
    expect(store.appendStreamingContent(conversation.id, assistant.id, '迟到分片')).toBe(false)
    expect(persistedState('stream-user').conversations[0].messages[0].content).toBe('你好，世界')
  })

  it('即时保存会合并尚未到期的流式节流写入', async () => {
    const store = useChatStore()
    await store.hydrate('stream-pending-user')
    const streaming = store.createConversation('gpt-4.1', '流式会话')!
    const other = store.createConversation('gpt-4.1', '其他会话')!
    const assistant = store.addMessage(streaming.id, {
      role: 'assistant',
      content: '',
      status: 'streaming',
    })!
    store.startStreaming(streaming.id, assistant.id)
    await store.flushPersistence()

    expect(store.appendStreamingContent(streaming.id, assistant.id, '节流内容')).toBe(true)
    expect(store.selectConversation(other.id)).toBe(true)
    await store.flushPersistence()

    const persisted = persistedState('stream-pending-user')
    const persistedStreaming = persisted.conversations.find(({ id }) => id === streaming.id)!
    expect(persistedStreaming.messages[0].content).toBe('节流内容')
  })

  it('停止流式回答会取消请求并保留已生成内容', async () => {
    const store = useChatStore()
    await store.hydrate('stop-user')
    const conversation = store.createConversation('gpt-4.1')!
    const assistant = store.addMessage(conversation.id, {
      role: 'assistant',
      content: '已生成',
      status: 'streaming',
    })!
    const controller = new AbortController()
    store.startStreaming(conversation.id, assistant.id, controller)

    expect(store.stopStreaming()).toBe(true)
    await store.flushPersistence()
    expect(controller.signal.aborted).toBe(true)
    expect(assistant).toEqual(expect.objectContaining({
      content: '已生成',
      status: 'stopped',
      finishReason: 'stopped',
    }))
    expect(store.stopStreaming()).toBe(false)
  })

  it('流式失败保留消息级错误和当前错误', async () => {
    const store = useChatStore()
    await store.hydrate('error-user')
    const conversation = store.createConversation('gpt-4.1')!
    const assistant = store.addMessage(conversation.id, {
      role: 'assistant',
      content: '部分内容',
      status: 'streaming',
    })!
    store.startStreaming(conversation.id, assistant.id)

    expect(store.failStreaming(
      conversation.id,
      assistant.id,
      '上游暂时不可用',
      'UPSTREAM_UNAVAILABLE',
    )).toBe(true)
    await store.flushPersistence()

    expect(assistant).toEqual(expect.objectContaining({
      status: 'error',
      errorCode: 'UPSTREAM_UNAVAILABLE',
      errorMessage: '上游暂时不可用',
    }))
    expect(store.streamError).toEqual({
      code: 'UPSTREAM_UNAVAILABLE',
      message: '上游暂时不可用',
    })
    expect(store.isStreaming).toBe(false)

    const persisted = persistedState('error-user')
    expect(persisted.streamError).toBeUndefined()
    expect(persisted.isStreaming).toBeUndefined()
    expect(persisted.streamController).toBeUndefined()
  })

  it('新流式请求会停止旧请求并拒绝旧分片', async () => {
    const store = useChatStore()
    await store.hydrate('race-user')
    const conversation = store.createConversation('gpt-4.1')!
    const first = store.addMessage(conversation.id, {
      role: 'assistant',
      content: '',
      status: 'streaming',
    })!
    const second = store.addMessage(conversation.id, {
      role: 'assistant',
      content: '',
      status: 'streaming',
    })!
    const firstController = new AbortController()

    store.startStreaming(conversation.id, first.id, firstController)
    store.startStreaming(conversation.id, second.id)

    expect(firstController.signal.aborted).toBe(true)
    expect(first.status).toBe('stopped')
    expect(store.appendStreamingContent(conversation.id, first.id, '旧分片')).toBe(false)
    expect(store.appendStreamingContent(conversation.id, second.id, '新分片')).toBe(true)
    expect(second.content).toBe('新分片')
    store.stopStreaming()
    await store.flushPersistence()
  })

  it('v1 历史恢复后等待用户决定，不会静默加入服务端 outbox', async () => {
    persistedBuckets.set('legacy-user', {
      version: 1,
      activeConversationId: 'legacy-conversation',
      conversations: [{
        id: 'legacy-conversation',
        userId: 'legacy-user',
        title: '旧聊天',
        model: 'gpt-5',
        messages: [{
          id: 'legacy-message',
          role: 'user',
          content: '只保存在本机的正文',
          createdAt: 1,
          status: 'complete',
        }],
        createdAt: 1,
        updatedAt: 1,
      }],
    })

    const store = useChatStore()
    await store.hydrate('legacy-user')
    await store.flushPersistence()

    expect(store.legacyImportDecision).toBe('pending')
    expect(store.legacyConversationIds).toEqual(['legacy-conversation'])
    expect(store.legacyImportRequired).toBe(true)
    expect(store.outbox).toEqual([])
    expect(chatApiMocks.createChatConversation).not.toHaveBeenCalled()
    expect(persistedState('legacy-user')).toMatchObject({
      version: 2,
      legacyImportDecision: 'pending',
      legacyConversationIds: ['legacy-conversation'],
      outbox: [],
    })
  })

  it('接受旧记录导入时只创建一次，并只发送获准的正文白名单字段', async () => {
    persistedBuckets.set('legacy-accept', {
      version: 1,
      activeConversationId: 'legacy-conversation',
      conversations: [{
        id: 'legacy-conversation',
        userId: 'legacy-accept',
        title: '旧聊天',
        model: 'gpt-5',
        messages: [{
          id: 'legacy-assistant',
          role: 'assistant',
          content: '旧回答',
          createdAt: 1,
          status: 'error',
          attemptId: 'must-not-upload',
          receiptId: 'must-not-upload',
          chargedAmount: 99,
          actualModel: 'must-not-upload',
        }],
        createdAt: 1,
        updatedAt: 1,
      }],
    })

    const store = useChatStore()
    await store.hydrate('legacy-accept')
    expect(store.acceptLegacyImport()).toBe(1)
    expect(store.acceptLegacyImport()).toBe(1)
    expect(store.outbox).toHaveLength(1)

    await store.syncHistory()

    expect(chatApiMocks.createChatConversation).toHaveBeenCalledTimes(1)
    const request = chatApiMocks.createChatConversation.mock.calls[0]?.[0]
    expect(request).toEqual({
      id: 'legacy-conversation',
      title: '旧聊天',
      model: 'gpt-5',
      importedMessages: [{
        id: 'legacy-assistant',
        role: 'assistant',
        content: '旧回答',
        status: 'interrupted',
        createdAt: 1,
      }],
    })
    expect(JSON.stringify(request)).not.toContain('must-not-upload')
    expect(store.outbox).toEqual([])
    expect(store.acceptLegacyImport()).toBe(0)
  })

  it('拒绝旧记录导入后持久化决定，重新加载仍只保留在本机', async () => {
    persistedBuckets.set('legacy-decline', {
      version: 1,
      activeConversationId: 'legacy-conversation',
      conversations: [{
        id: 'legacy-conversation',
        userId: 'legacy-decline',
        title: '不上传',
        model: 'gpt-5',
        messages: [],
        createdAt: 1,
        updatedAt: 1,
      }],
    })

    const firstStore = useChatStore()
    await firstStore.hydrate('legacy-decline')
    firstStore.declineLegacyImport()
    await firstStore.flushPersistence()

    setActivePinia(createPinia())
    const restoredStore = useChatStore()
    await restoredStore.hydrate('legacy-decline')

    expect(restoredStore.legacyImportDecision).toBe('declined')
    expect(restoredStore.legacyImportRequired).toBe(false)
    expect(restoredStore.outbox).toEqual([])
    expect(chatApiMocks.createChatConversation).not.toHaveBeenCalled()
  })

  it('PATCH 遇到 409 时拉取最新 revision 后只重放一次', async () => {
    const conversation = {
      id: 'conflict-conversation',
      userId: 'conflict-user',
      title: '本地标题',
      model: 'gpt-5',
      messages: [],
      createdAt: 1,
      updatedAt: 2,
      serverRevision: 1,
      serverVersion: 1,
    }
    persistedBuckets.set('conflict-user', persistedChatState(
      'conflict-user',
      conversation,
      {
        serverVersion: 1,
        outbox: [{
          mutationId: 'patch:conflict-conversation',
          type: 'patch',
          conversationId: 'conflict-conversation',
          createdAt: 3,
          revision: 1,
          title: '本地标题',
        }],
      },
    ))
    chatApiMocks.patchChatConversation
      .mockRejectedValueOnce(new ChatAPIError('Revision conflict', { status: 409 }))
      .mockResolvedValueOnce(serverConversation('conflict-conversation', {
        title: '本地标题',
        revision: 3,
        version: 3,
        updatedAt: 3,
      }))
    chatApiMocks.getChatConversation.mockResolvedValueOnce(
      serverConversation('conflict-conversation', {
        title: '远端标题',
        revision: 2,
        version: 2,
        updatedAt: 2,
      }),
    )

    const store = useChatStore()
    await store.hydrate('conflict-user')
    await store.syncHistory()

    expect(chatApiMocks.patchChatConversation).toHaveBeenCalledTimes(2)
    expect(chatApiMocks.patchChatConversation.mock.calls.map((call) => call[1].revision))
      .toEqual([1, 2])
    expect(chatApiMocks.getChatConversation).toHaveBeenCalledTimes(1)
    expect(store.outbox).toEqual([])
    expect(store.conversations[0]).toMatchObject({
      title: '本地标题',
      serverRevision: 3,
    })
  })

  it('同步消息按 ID 合并，不丢失本机并发加入的消息', async () => {
    const conversation = {
      id: 'merge-conversation',
      userId: 'merge-user',
      title: '合并',
      model: 'gpt-5',
      messages: [
        {
          id: 'shared-message',
          role: 'assistant',
          content: '旧内容',
          createdAt: 1,
          status: 'complete',
        },
        {
          id: 'local-message',
          role: 'user',
          content: '本机并发消息',
          createdAt: 2,
          status: 'complete',
        },
      ],
      createdAt: 1,
      updatedAt: 2,
      serverRevision: 1,
      serverVersion: 1,
    }
    persistedBuckets.set('merge-user', persistedChatState('merge-user', conversation, {
      serverVersion: 1,
    }))
    chatApiMocks.getChatSync
      .mockResolvedValueOnce({
        changes: [{
          type: 'upsert',
          version: 2,
          conversationId: 'merge-conversation',
          conversation: serverConversation('merge-conversation', {
            title: '合并',
            revision: 2,
            version: 2,
            headMessageId: 'remote-message',
            messageCount: 3,
            messages: [
              {
                id: 'shared-message',
                role: 'assistant',
                content: '服务端新内容',
                createdAt: 1,
                position: 0,
                status: 'complete',
              },
              {
                id: 'remote-message',
                role: 'assistant',
                content: '远端并发消息',
                createdAt: 3,
                position: 2,
                status: 'complete',
              },
            ],
            updatedAt: 3,
          }),
        }],
        latestVersion: 2,
        nextCursor: null,
        hasMore: false,
      })
      .mockResolvedValueOnce({
        changes: [],
        latestVersion: 2,
        nextCursor: null,
        hasMore: false,
      })

    const store = useChatStore()
    await store.hydrate('merge-user')
    await store.syncHistory()

    expect(store.conversations[0].messages).toHaveLength(3)
    expect(store.conversations[0].messages.map(({ id }) => id)).toEqual(
      expect.arrayContaining(['shared-message', 'local-message', 'remote-message']),
    )
    expect(store.conversations[0].messages.find(({ id }) => id === 'shared-message')?.content)
      .toBe('服务端新内容')
  })

  it('同一批同步中删除墓碑优先于更旧的会话更新', async () => {
    const conversation = {
      id: 'deleted-conversation',
      userId: 'delete-user',
      title: '待删除',
      model: 'gpt-5',
      messages: [],
      createdAt: 1,
      updatedAt: 1,
      serverRevision: 1,
      serverVersion: 1,
    }
    persistedBuckets.set('delete-user', persistedChatState('delete-user', conversation, {
      serverVersion: 1,
    }))
    chatApiMocks.getChatSync
      .mockResolvedValueOnce({
        changes: [
          {
            type: 'delete',
            version: 3,
            conversationId: 'deleted-conversation',
            deletedAt: 3,
          },
          {
            type: 'upsert',
            version: 2,
            conversationId: 'deleted-conversation',
            conversation: serverConversation('deleted-conversation', {
              revision: 2,
              version: 2,
            }),
          },
        ],
        latestVersion: 3,
        nextCursor: null,
        hasMore: false,
      })
      .mockResolvedValueOnce({
        changes: [],
        latestVersion: 3,
        nextCursor: null,
        hasMore: false,
      })

    const store = useChatStore()
    await store.hydrate('delete-user')
    await store.syncHistory()

    expect(store.conversations).toEqual([])
    expect(store.serverVersion).toBe(3)
  })

  it('恢复 processing attempt 的 partial 内容时保持 stopped，且不创建新请求', async () => {
    const conversation = {
      id: 'attempt-conversation',
      userId: 'attempt-user',
      title: '断流恢复',
      model: 'gpt-5',
      messages: [{
        id: 'assistant-message',
        role: 'assistant',
        content: '旧 partial',
        createdAt: 1,
        status: 'stopped',
        attemptId: 'attempt-1',
      }],
      createdAt: 1,
      updatedAt: 1,
      serverRevision: 1,
      serverVersion: 1,
    }
    persistedBuckets.set('attempt-user', persistedChatState('attempt-user', conversation))
    chatApiMocks.getChatAttempt.mockResolvedValueOnce({
      attemptId: 'attempt-1',
      conversationId: 'attempt-conversation',
      assistantMessageId: 'assistant-message',
      status: 'processing',
      assistantMessage: {
        id: 'assistant-message',
        role: 'assistant',
        content: '服务端 partial',
        createdAt: 1,
        updatedAt: 2,
        status: 'stopped',
        attemptId: 'attempt-1',
      },
    })

    const store = useChatStore()
    await store.hydrate('attempt-user')
    const attempt = await store.recoverAttempt('attempt-conversation', 'assistant-message')

    expect(attempt?.status).toBe('processing')
    expect(store.conversations[0].messages[0]).toMatchObject({
      content: '服务端 partial',
      status: 'stopped',
      finishReason: 'interrupted',
    })
    expect(chatApiMocks.getChatAttempt).toHaveBeenCalledTimes(1)
    expect(chatApiMocks.createChatConversation).not.toHaveBeenCalled()
    expect(chatApiMocks.patchChatConversation).not.toHaveBeenCalled()
  })

  it('旧记录导入按 UTF-8 字节截断，完整请求保持在 2 MiB 内', async () => {
    const largeMessages = Array.from({ length: 3 }, (_, index) => ({
      id: `large-message-${index}`,
      role: index % 2 === 0 ? 'user' : 'assistant',
      content: 'x'.repeat(800_000),
      createdAt: index + 1,
      status: 'complete',
    }))
    persistedBuckets.set('legacy-budget', {
      version: 1,
      activeConversationId: 'legacy-large',
      conversations: [{
        id: 'legacy-large',
        userId: 'legacy-budget',
        title: '大记录',
        model: 'gpt-5',
        messages: largeMessages,
        createdAt: 1,
        updatedAt: 3,
      }],
    })

    const store = useChatStore()
    await store.hydrate('legacy-budget')
    store.acceptLegacyImport()
    await store.syncHistory()

    const request = chatApiMocks.createChatConversation.mock.calls[0]?.[0] as
      CreateChatConversationRequest
    expect(request.importedMessages).toHaveLength(2)
    expect(new TextEncoder().encode(JSON.stringify(request)).byteLength)
      .toBeLessThanOrEqual(2 * 1024 * 1024)
  })
})
