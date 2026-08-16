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
import {
  chatActivityPartKey,
  chatActivityPartText,
} from '@/features/chat/activity'
import { mergeChatHistoryStates } from '@/features/chat/persistence'
import type { ChatHistoryMutation } from '@/features/chat/persistence'
import { MAX_CHAT_CONVERSATIONS, useChatStore } from '@/stores/chat'
import type {
  ChatActivity,
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

function reasoningActivity(
  responseId: string,
  text = '',
  overrides: Partial<ChatActivity> = {},
): ChatActivity {
  const itemId = `reasoning-${responseId}`
  const partKey = chatActivityPartKey(responseId, itemId, 0, 0)
  return {
    key: responseId,
    responseId,
    status: 'streaming',
    reasoningMode: 'pro',
    reasoningEffort: 'medium',
    startedAt: 100,
    updatedAt: 101,
    lastSequenceNumber: 2,
    items: [{
      key: JSON.stringify([responseId, itemId, 0]),
      itemId,
      outputIndex: 0,
      status: 'streaming',
      startedAt: 100,
      updatedAt: 101,
      lastSequenceNumber: 2,
      parts: [{
        key: partKey,
        itemId,
        outputIndex: 0,
        summaryIndex: 0,
        text,
        status: 'streaming',
        startedAt: 100,
        updatedAt: 101,
        lastSequenceNumber: 2,
      }],
    }],
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

  it('从 v2 历史恢复附件元数据，并以 v5 schema 持久化', async () => {
    const conversation = {
      id: 'attachment-conversation',
      userId: 'attachment-user',
      title: 'Attachment history',
      model: 'gpt-5',
      messages: [{
        id: 'attachment-message',
        role: 'user',
        content: '',
        createdAt: 1,
        status: 'complete',
        attachments: [{
          id: 'attachment-1',
          name: 'brief.pdf',
          kind: 'document',
          mimeType: 'application/pdf',
          size: 2048,
          status: 'ready',
          expiresAt: '2099-01-01T00:00:00Z',
          pageCount: 3,
          ignoredObjectUrl: 'blob:must-not-persist',
        }],
      }],
      createdAt: 1,
      updatedAt: 2,
    }
    persistedBuckets.set(
      'attachment-user',
      persistedChatState('attachment-user', conversation),
    )

    const store = useChatStore()
    await store.hydrate('attachment-user')
    expect(store.conversations[0]?.messages[0]?.attachments).toEqual([{
      id: 'attachment-1',
      name: 'brief.pdf',
      kind: 'document',
      mimeType: 'application/pdf',
      size: 2048,
      status: 'ready',
      expiresAt: '2099-01-01T00:00:00Z',
      pageCount: 3,
    }])

    store.addMessage('attachment-conversation', { role: 'assistant', content: 'Done' })
    await store.flushPersistence()
    const persisted = persistedState('attachment-user')
    expect(persisted.version).toBe(5)
    expect(persisted.conversations[0]?.messages[0]?.attachments).toEqual(
      store.conversations[0]?.messages[0]?.attachments,
    )
    expect(JSON.stringify(persisted)).not.toContain('blob:must-not-persist')
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

  it('从搜索结果删除会话后同步移除结果行', async () => {
    const store = useChatStore()
    await store.hydrate('search-delete-user')
    const conversation = store.createConversation('gpt-5', '待删除的搜索结果')!
    chatApiMocks.searchChatConversations.mockResolvedValueOnce({
      items: [serverConversation(conversation.id, {
        title: conversation.title,
        model: conversation.model,
      })],
      nextCursor: null,
      hasMore: false,
    })

    await store.searchHistory('待删除')
    expect(store.searchResults.map(({ id }) => id)).toEqual([conversation.id])

    expect(store.deleteConversation(conversation.id)).toBe(true)
    expect(store.searchResults).toEqual([])
  })

  it('点击新聊天后刷新及服务端历史合并仍保持新聊天', async () => {
    const firstStore = useChatStore()
    await firstStore.hydrate('new-chat-refresh-user')
    const first = firstStore.createConversation('gpt-5', '第一段对话')!
    const second = firstStore.createConversation('gpt-5', '第二段对话')!
    await firstStore.flushPersistence()

    expect(firstStore.selectConversation(null)).toBe(true)
    await firstStore.flushPersistence()
    expect(persistedState('new-chat-refresh-user')).toMatchObject({
      activeConversationId: null,
      activeConversationSelectionResolved: true,
    })

    chatApiMocks.listChatConversations.mockResolvedValueOnce({
      items: [
        serverConversation(second.id, { title: second.title, updatedAt: 20 }),
        serverConversation(first.id, { title: first.title, updatedAt: 10 }),
      ],
      nextCursor: null,
      hasMore: false,
    })

    setActivePinia(createPinia())
    const restoredStore = useChatStore()
    await restoredStore.hydrate('new-chat-refresh-user')
    expect(restoredStore.conversations).toHaveLength(2)
    expect(restoredStore.activeConversationId).toBeNull()
    expect(restoredStore.activeConversation).toBeNull()

    await restoredStore.syncHistory()
    await restoredStore.loadConversationPage(true)
    expect(restoredStore.activeConversationId).toBeNull()
    expect(restoredStore.activeConversation).toBeNull()
  })

  it('首次服务端同步完成排序后才选择最新会话', async () => {
    chatApiMocks.getChatSync
      .mockResolvedValueOnce({
        changes: [
          {
            type: 'upsert',
            version: 1,
            conversationId: 'older-conversation',
            conversation: serverConversation('older-conversation', { updatedAt: 10 }),
          },
          {
            type: 'upsert',
            version: 2,
            conversationId: 'newer-conversation',
            conversation: serverConversation('newer-conversation', { updatedAt: 20 }),
          },
        ],
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
    await store.hydrate('initial-server-selection')
    await store.syncHistory()
    await store.flushPersistence()

    expect(store.conversations.map(({ id }) => id)).toEqual([
      'newer-conversation',
      'older-conversation',
    ])
    expect(store.activeConversationId).toBe('newer-conversation')
    expect(persistedState('initial-server-selection').activeConversationId)
      .toBe('newer-conversation')
  })

  it('全量同步 502 后仍针对当前会话重放并准备 completion', async () => {
    chatApiMocks.getChatSync.mockRejectedValue(
      new ChatAPIError('Bad Gateway', { status: 502 }),
    )

    const store = useChatStore()
    await store.hydrate('sync-failed-targeted-success')
    const conversation = store.createConversation('gpt-5', '保持聊天可用')!

    await store.syncHistory()

    expect(store.syncStatus).toBe('error')
    expect(store.streamError).toBeNull()

    const remote = serverConversation(conversation.id, {
      title: conversation.title,
      model: conversation.model,
      revision: 2,
      version: 2,
      updatedAt: conversation.updatedAt,
    })
    chatApiMocks.getChatConversation.mockResolvedValueOnce(remote)

    await expect(store.prepareConversationForCompletion(conversation.id))
      .resolves.toBe(true)

    expect(chatApiMocks.getChatSync).toHaveBeenCalledTimes(1)
    expect(chatApiMocks.createChatConversation).toHaveBeenCalledWith(
      {
        id: conversation.id,
        title: conversation.title,
        model: conversation.model,
      },
      expect.any(AbortSignal),
    )
    expect(chatApiMocks.getChatConversation).toHaveBeenCalledWith(
      conversation.id,
      expect.any(AbortSignal),
    )
    expect(store.outbox).toEqual([])
    expect(store.syncStatus).toBe('error')
    expect(store.streamError).toBeNull()
  })

  it('针对当前会话的云端准备也 502 时使用本地会话继续并持久化', async () => {
    const store = useChatStore()
    await store.hydrate('targeted-history-fallback')
    const conversation = store.createConversation('gpt-5', '本地降级会话')!
    store.addMessage(conversation.id, {
      id: 'local-user-message',
      role: 'user',
      content: '即使云端同步失败也要保留',
      status: 'complete',
    })
    store.addMessage(conversation.id, {
      id: 'local-assistant-message',
      role: 'assistant',
      content: '已保留在本地。',
      status: 'complete',
    })
    chatApiMocks.createChatConversation.mockRejectedValueOnce(
      new ChatAPIError('Bad Gateway', { status: 502 }),
    )

    await expect(store.prepareConversationForCompletion(conversation.id))
      .resolves.toBe(true)

    expect(store.syncStatus).toBe('error')
    expect(store.outbox).toEqual([
      expect.objectContaining({
        mutationId: `create:${conversation.id}`,
        conversationId: conversation.id,
        type: 'create',
      }),
    ])
    expect(chatApiMocks.getChatConversation).not.toHaveBeenCalled()

    await store.flushPersistence()

    setActivePinia(createPinia())
    const restoredStore = useChatStore()
    await restoredStore.hydrate('targeted-history-fallback')

    expect(restoredStore.activeConversation).toMatchObject({
      id: conversation.id,
      title: '本地降级会话',
    })
    expect(restoredStore.activeConversation?.messages).toEqual([
      expect.objectContaining({
        id: 'local-user-message',
        role: 'user',
        content: '即使云端同步失败也要保留',
      }),
      expect.objectContaining({
        id: 'local-assistant-message',
        role: 'assistant',
        content: '已保留在本地。',
      }),
    ])
    expect(restoredStore.outbox).toEqual([
      expect.objectContaining({
        mutationId: `create:${conversation.id}`,
        conversationId: conversation.id,
        type: 'create',
      }),
    ])
  })

  it('云端历史接口 501 时标记 unavailable 但仍允许 completion', async () => {
    const store = useChatStore()
    await store.hydrate('history-unavailable-fallback')
    const conversation = store.createConversation('gpt-5', '云端历史不可用')!
    chatApiMocks.createChatConversation.mockRejectedValueOnce(
      new ChatAPIError('Not Implemented', { status: 501 }),
    )

    await expect(store.prepareConversationForCompletion(conversation.id))
      .resolves.toBe(true)

    expect(store.syncStatus).toBe('unavailable')
    expect(store.serverHistoryAvailable).toBe(false)
    expect(store.syncError).toBeNull()
    expect(store.streamError).toBeNull()
  })

  it('completion 被接纳后持久化最新 head，且不会改写同步错误状态', async () => {
    chatApiMocks.getChatSync.mockRejectedValue(
      new ChatAPIError('Bad Gateway', { status: 502 }),
    )

    const store = useChatStore()
    await store.hydrate('accepted-head-persistence')
    const conversation = store.createConversation('gpt-5', '回答接纳状态')!
    store.addMessage(conversation.id, {
      id: 'accepted-user-message',
      role: 'user',
      content: '请继续',
      status: 'complete',
    })
    store.addMessage(conversation.id, {
      id: 'accepted-assistant-message',
      role: 'assistant',
      content: '',
      status: 'streaming',
    })
    await store.syncHistory()

    expect(store.syncStatus).toBe('error')
    expect(store.markCompletionAccepted(
      conversation.id,
      'accepted-assistant-message',
    )).toBe(true)
    expect(store.syncStatus).toBe('error')
    expect(store.activeConversation).toMatchObject({
      headMessageId: 'accepted-assistant-message',
      messageCount: 2,
    })

    await store.flushPersistence()

    setActivePinia(createPinia())
    const restoredStore = useChatStore()
    await restoredStore.hydrate('accepted-head-persistence')

    expect(restoredStore.activeConversation).toMatchObject({
      id: conversation.id,
      headMessageId: 'accepted-assistant-message',
      messageCount: 2,
    })
    expect(restoredStore.activeConversation?.messages.map(({ id }) => id)).toEqual([
      'accepted-user-message',
      'accepted-assistant-message',
    ])
  })

  it('账号切换会中止旧会话的 completion 准备，且不污染新账号', async () => {
    const staleCreate = createDeferred<ChatServerConversation>()
    chatApiMocks.createChatConversation.mockReturnValueOnce(staleCreate.promise)

    const store = useChatStore()
    await store.hydrate('completion-account-a')
    const staleConversation = store.createConversation('gpt-5', '账号 A 会话')!
    const stalePreparation = store.prepareConversationForCompletion(staleConversation.id)
    await vi.waitFor(() => {
      expect(chatApiMocks.createChatConversation).toHaveBeenCalledTimes(1)
    })

    await store.hydrate('completion-account-b')
    const currentConversation = store.createConversation('gpt-5', '账号 B 会话')!
    staleCreate.resolve(serverConversation(staleConversation.id, {
      title: '迟到的账号 A 会话',
    }))

    await expect(stalePreparation).resolves.toBe(false)
    expect(store.userId).toBe('completion-account-b')
    expect(store.conversations.map(({ id }) => id)).toEqual([currentConversation.id])
    expect(store.conversations.some(({ id }) => id === staleConversation.id)).toBe(false)
  })

  it('同账号 hydration 期间点击新聊天不会被迟到的旧选择覆盖', async () => {
    const oldConversation = {
      id: 'old-active-conversation',
      userId: 'hydration-selection-race',
      title: '旧会话',
      model: 'gpt-5',
      messages: [],
      createdAt: 10,
      updatedAt: 20,
    }
    const delayedLoad = createDeferred<unknown | null>()
    persistedBuckets.set(
      'hydration-selection-race',
      persistedChatState('hydration-selection-race', oldConversation),
    )
    persistenceMocks.load.mockReturnValueOnce(delayedLoad.promise)

    const store = useChatStore()
    const hydration = store.hydrate('hydration-selection-race')
    await vi.waitFor(() => expect(persistenceMocks.load).toHaveBeenCalled())
    expect(store.selectConversation(null)).toBe(true)
    delayedLoad.resolve(persistedChatState('hydration-selection-race', oldConversation))
    await hydration
    await store.flushPersistence()

    expect(store.conversations.map(({ id }) => id)).toEqual(['old-active-conversation'])
    expect(store.activeConversationId).toBeNull()
    expect(persistedState('hydration-selection-race')).toMatchObject({
      activeConversationId: null,
      activeConversationSelectionResolved: true,
    })
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

  it('持久化待同步停止意图并在服务端确认后清除', async () => {
    const store = useChatStore()
    await store.hydrate('stop-intent-user')
    const conversation = store.createConversation('gpt-5.6-sol')!
    const assistant = store.addMessage(conversation.id, {
      role: 'assistant',
      content: 'Partial',
      status: 'stopped',
      attemptId: 'attempt-stop-intent',
    })!

    expect(store.updateMessage(conversation.id, assistant.id, {
      pendingStopRequestedAt: 1_765_800_000_000,
    })).toBe(true)
    await store.flushPersistence()

    setActivePinia(createPinia())
    const restored = useChatStore()
    await restored.hydrate('stop-intent-user')
    expect(restored.activeConversation?.messages[0]?.pendingStopRequestedAt)
      .toBe(1_765_800_000_000)

    expect(restored.updateMessage(conversation.id, assistant.id, {
      pendingStopRequestedAt: null,
    })).toBe(true)
    await restored.flushPersistence()

    setActivePinia(createPinia())
    const confirmed = useChatStore()
    await confirmed.hydrate('stop-intent-user')
    expect(confirmed.activeConversation?.messages[0]?.pendingStopRequestedAt).toBeUndefined()
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

  it('恢复部分 reasoning Activity 文本并把开放状态标记为 disconnected', async () => {
    const conversation = {
      id: 'partial-activity-conversation',
      userId: 'partial-activity-user',
      title: 'Partial activity',
      model: 'gpt-5',
      messages: [{
        id: 'partial-assistant',
        role: 'assistant',
        content: '',
        createdAt: 10,
        status: 'streaming',
        activities: [{
          response_id: 'resp-partial',
          status: 'streaming',
          reasoning_mode: 'pro',
          reasoning_effort: 'medium',
          started_at: 11,
          updated_at: 12,
          items: [{
            item_id: 'reasoning-partial',
            output_index: 0,
            status: 'streaming',
            started_at: 11,
            updated_at: 12,
            summary: [{
              type: 'summary_text',
              summary_index: 0,
              text: 'partial safe summary',
              status: 'streaming',
              started_at: 11,
              updated_at: 12,
            }],
          }],
        }],
      }],
      createdAt: 10,
      updatedAt: 12,
    }
    persistedBuckets.set(
      'partial-activity-user',
      persistedChatState('partial-activity-user', conversation, { version: 4 }),
    )

    const store = useChatStore()
    await store.hydrate('partial-activity-user')

    const message = store.conversations[0]?.messages[0]
    expect(message?.status).toBe('stopped')
    expect(message?.activities?.[0]).toMatchObject({
      responseId: 'resp-partial',
      status: 'disconnected',
      items: [{
        status: 'disconnected',
        parts: [{ text: 'partial safe summary', status: 'disconnected' }],
      }],
    })
    await store.flushPersistence()
    expect(persistedState('partial-activity-user').version).toBe(5)
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
    expect(store.persistenceDiagnostic).toMatchObject({
      code: 'CHAT_PERSISTENCE_PAYLOAD_INVALID',
      message: 'Error',
    })
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
    expect(store.persistenceDiagnostic).toMatchObject({
      code: expect.stringMatching(/^CHAT_PERSISTENCE_/),
      message: expect.stringMatching(/Error$/),
    })
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

  it('创建、增量校正并即时持久化 assistant Activity', async () => {
    const store = useChatStore()
    await store.hydrate('activity-user')
    const conversation = store.createConversation('gpt-5')!
    const assistant = store.addMessage(conversation.id, {
      id: 'assistant-activity',
      role: 'assistant',
      content: '',
      status: 'streaming',
    })!
    store.startStreaming(conversation.id, assistant.id)
    await store.flushPersistence()

    const activity = reasoningActivity('resp-store')
    const partKey = activity.items[0]!.parts[0]!.key
    expect(store.upsertMessageActivity(conversation.id, assistant.id, activity)).toBe(true)
    expect(store.appendMessageActivityDelta(
      conversation.id,
      assistant.id,
      activity.key,
      partKey,
      'draft',
    )).toBe(true)
    expect(assistant.activities?.[0]?.items[0]?.parts[0]).toMatchObject({
      text: '',
      streamingTextChunks: ['draft'],
    })
    expect(store.replaceMessageActivityText(
      conversation.id,
      assistant.id,
      activity.key,
      partKey,
      'authoritative summary',
    )).toBe(true)
    expect(assistant.activities?.[0]?.items[0]?.parts[0]?.streamingTextChunks).toBeUndefined()
    expect(store.setMessageActivityTerminal(
      conversation.id,
      assistant.id,
      activity.key,
      'completed',
    )).toBe(true)
    await store.flushPersistence()

    expect(assistant.activities?.[0]).toMatchObject({
      responseId: 'resp-store',
      status: 'completed',
      reasoningMode: 'pro',
      items: [{ parts: [{ text: 'authoritative summary', status: 'completed' }] }],
    })
    const persisted = persistedState('activity-user')
    expect(persisted.version).toBe(5)
    expect(persisted.conversations[0]?.messages[0]?.activities).toEqual(assistant.activities)

    setActivePinia(createPinia())
    const restored = useChatStore()
    await restored.hydrate('activity-user')
    expect(restored.conversations[0]?.messages[0]?.activities).toEqual(assistant.activities)
  })

  it('Activity delta 以分片追加，并仅在终态一次性合并正文', async () => {
    const store = useChatStore()
    await store.hydrate('activity-chunk-user')
    const conversation = store.createConversation('gpt-5')!
    const assistant = store.addMessage(conversation.id, {
      id: 'assistant-activity-chunks',
      role: 'assistant',
      content: '',
      status: 'streaming',
    })!
    store.startStreaming(conversation.id, assistant.id)

    const activity = reasoningActivity('resp-chunks', 'Base ')
    const partKey = activity.items[0]!.parts[0]!.key
    expect(store.upsertMessageActivity(conversation.id, assistant.id, activity)).toBe(true)
    expect(store.appendMessageActivityDelta(
      conversation.id,
      assistant.id,
      activity.key,
      partKey,
      'first ',
    )).toBe(true)
    expect(store.appendMessageActivityDelta(
      conversation.id,
      assistant.id,
      activity.key,
      partKey,
      'second',
    )).toBe(true)

    const streamingPart = assistant.activities?.[0]?.items[0]?.parts[0]
    expect(streamingPart).toMatchObject({
      text: 'Base ',
      streamingTextChunks: ['first ', 'second'],
      status: 'streaming',
    })
    expect(chatActivityPartText(streamingPart!)).toBe('Base first second')

    expect(store.setMessageActivityTerminal(
      conversation.id,
      assistant.id,
      activity.key,
      'completed',
    )).toBe(true)
    expect(streamingPart).toMatchObject({
      text: 'Base first second',
      status: 'completed',
    })
    expect(streamingPart?.streamingTextChunks).toBeUndefined()
  })

  it('keeps Activities isolated by conversation and assistant message during retries', async () => {
    const store = useChatStore()
    await store.hydrate('activity-isolation-user')
    const firstConversation = store.createConversation('gpt-5', 'First')!
    const secondConversation = store.createConversation('gpt-5', 'Second')!
    const firstAssistant = store.addMessage(firstConversation.id, {
      id: 'shared-assistant-id',
      role: 'assistant',
      content: '',
      status: 'streaming',
    })!
    const secondAssistant = store.addMessage(secondConversation.id, {
      id: 'shared-assistant-id',
      role: 'assistant',
      content: '',
      status: 'streaming',
    })!

    const firstActivity = reasoningActivity('resp-first', 'first')
    const secondActivity = reasoningActivity('resp-second', 'second')
    expect(store.upsertMessageActivity(
      firstConversation.id,
      firstAssistant.id,
      firstActivity,
    )).toBe(true)
    expect(store.upsertMessageActivity(
      secondConversation.id,
      secondAssistant.id,
      secondActivity,
    )).toBe(true)

    expect(firstAssistant.activities?.map(({ responseId }) => responseId)).toEqual(['resp-first'])
    expect(secondAssistant.activities?.map(({ responseId }) => responseId)).toEqual(['resp-second'])
    expect(store.selectConversation(firstConversation.id)).toBe(true)
    expect(secondAssistant.activities?.[0]?.items[0]?.parts[0]?.text).toBe('second')
  })

  it('stops the old assistant Activity before starting a retry assistant', async () => {
    const store = useChatStore()
    await store.hydrate('activity-retry-user')
    const conversation = store.createConversation('gpt-5')!
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
    store.startStreaming(conversation.id, first.id)
    const firstActivity = reasoningActivity('resp-old', 'old')
    store.upsertMessageActivity(conversation.id, first.id, firstActivity)

    store.startStreaming(conversation.id, second.id)
    const secondActivity = reasoningActivity('resp-new', 'new')
    store.upsertMessageActivity(conversation.id, second.id, secondActivity)
    store.upsertMessageActivity(
      conversation.id,
      first.id,
      reasoningActivity('resp-late-buffer', 'late buffered summary'),
    )

    expect(first.activities?.[0]?.status).toBe('stopped')
    expect(first.activities?.find(({ responseId }) => responseId === 'resp-late-buffer')?.status)
      .toBe('stopped')
    expect(store.appendMessageActivityDelta(
      conversation.id,
      first.id,
      firstActivity.key,
      firstActivity.items[0]!.parts[0]!.key,
      ' late',
    )).toBe(false)
    expect(second.activities?.[0]?.items[0]?.parts[0]?.text).toBe('new')
    store.stopStreaming()
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
      version: 5,
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

  it('同一毫秒创建的一轮消息仍保持 user 在 assistant 前', async () => {
    const store = useChatStore()
    await store.hydrate('ordered-local-user')
    const conversation = store.createConversation('gpt-5', '顺序测试')!

    const user = store.addMessage(conversation.id, {
      id: 'user-z',
      role: 'user',
      content: '你好',
      createdAt: 10,
    })
    const assistant = store.addMessage(conversation.id, {
      id: 'assistant-a',
      role: 'assistant',
      content: '你好！',
      createdAt: 10,
    })

    expect([user?.position, assistant?.position]).toEqual([1, 2])
    expect(conversation.messages.map(({ id }) => id)).toEqual(['user-z', 'assistant-a'])
  })

  it('首次加载服务端摘要时补拉消息，并把新历史持久化到本地', async () => {
    const remote = serverConversation('remote-history', {
      title: '你好',
      headMessageId: 'assistant-2',
      messageCount: 2,
      createdAt: 10,
      updatedAt: 11,
    })
    chatApiMocks.listChatConversations.mockResolvedValueOnce({
      items: [remote],
      nextCursor: null,
      hasMore: false,
    })
    chatApiMocks.getChatConversation.mockResolvedValueOnce(remote)
    chatApiMocks.getChatConversationMessages.mockResolvedValueOnce({
      items: [
        {
          id: 'assistant-2',
          role: 'assistant',
          content: '有什么可以帮你的吗？',
          createdAt: 20,
          position: 2,
          status: 'complete',
        },
        {
          id: 'user-1',
          role: 'user',
          content: '你好',
          createdAt: 20,
          position: 1,
          status: 'complete',
        },
      ],
      nextBeforePosition: null,
      hasMore: false,
    })

    const store = useChatStore()
    await store.hydrate('remote-history-user')
    await store.loadConversationPage(true)
    await store.flushPersistence()

    expect(chatApiMocks.getChatConversationMessages).toHaveBeenCalledWith(
      'remote-history',
      { beforePosition: null, limit: 100 },
    )
    expect(store.activeConversation?.messages.map(({ id }) => id)).toEqual([
      'user-1',
      'assistant-2',
    ])

    setActivePinia(createPinia())
    const restoredStore = useChatStore()
    await restoredStore.hydrate('remote-history-user')
    expect(restoredStore.activeConversation?.messages.map(({ id }) => id)).toEqual([
      'user-1',
      'assistant-2',
    ])
  })

  it('忽略账号切换前迟到的历史列表响应', async () => {
    const stalePage = createDeferred<{
      items: ChatServerConversation[]
      nextCursor: string | null
      hasMore: boolean
    }>()
    chatApiMocks.listChatConversations
      .mockReturnValueOnce(stalePage.promise)
      .mockResolvedValueOnce({
        items: [serverConversation('current-conversation', { title: '当前账号' })],
        nextCursor: null,
        hasMore: false,
      })

    const store = useChatStore()
    await store.hydrate('stale-account')
    const staleLoad = store.loadConversationPage(true)
    await vi.waitFor(() => {
      expect(chatApiMocks.listChatConversations).toHaveBeenCalledTimes(1)
    })

    await store.hydrate('current-account')
    await store.loadConversationPage(true)
    stalePage.resolve({
      items: [serverConversation('stale-conversation', { title: '旧账号' })],
      nextCursor: 'stale-cursor',
      hasMore: true,
    })
    await staleLoad

    expect(store.userId).toBe('current-account')
    expect(store.conversations.map(({ id }) => id)).toEqual(['current-conversation'])
    expect(store.conversationsHaveMore).toBe(false)
    expect(store.loadingConversationPage).toBe(false)
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

  it('元数据同步不会按随机消息 ID 反排同毫秒的一轮对话', async () => {
    const conversation = {
      id: 'stable-order-conversation',
      userId: 'stable-order-user',
      title: '稳定顺序',
      model: 'gpt-5',
      messages: [
        {
          id: 'user-z',
          role: 'user',
          content: '你好',
          createdAt: 20,
          status: 'complete',
        },
        {
          id: 'assistant-a',
          role: 'assistant',
          content: '你好！',
          createdAt: 20,
          status: 'complete',
        },
      ],
      headMessageId: 'assistant-a',
      messageCount: 2,
      createdAt: 10,
      updatedAt: 20,
      serverRevision: 1,
      serverVersion: 1,
    }
    persistedBuckets.set(
      'stable-order-user',
      persistedChatState('stable-order-user', conversation, { serverVersion: 1 }),
    )
    chatApiMocks.getChatSync
      .mockResolvedValueOnce({
        changes: [{
          type: 'upsert',
          version: 2,
          conversationId: 'stable-order-conversation',
          conversation: serverConversation('stable-order-conversation', {
            title: '稳定顺序',
            revision: 2,
            version: 2,
            headMessageId: 'assistant-a',
            messageCount: 2,
            messages: [],
            createdAt: 10,
            updatedAt: 20,
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
    await store.hydrate('stable-order-user')
    await store.syncHistory()

    expect(store.activeConversation?.messages.map(({ id }) => id)).toEqual([
      'user-z',
      'assistant-a',
    ])
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
    expect(store.conversations[0].messages.map(({ id }) => id)).toEqual([
      'shared-message',
      'local-message',
      'remote-message',
    ])
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

  it('详情消息页会修复已持久化的反序历史', async () => {
    const conversation = {
      id: 'reversed-conversation',
      userId: 'reversed-user',
      title: '反序历史',
      model: 'gpt-5',
      messages: [
        {
          id: 'assistant-2',
          role: 'assistant',
          content: '回答',
          createdAt: 20,
          status: 'complete',
        },
        {
          id: 'user-1',
          role: 'user',
          content: '你好',
          createdAt: 20,
          status: 'complete',
        },
      ],
      headMessageId: 'assistant-2',
      messageCount: 2,
      createdAt: 10,
      updatedAt: 20,
      serverRevision: 1,
      serverVersion: 1,
    }
    persistedBuckets.set(
      'reversed-user',
      persistedChatState('reversed-user', conversation),
    )
    const remote = serverConversation('reversed-conversation', {
      title: '反序历史',
      headMessageId: 'assistant-2',
      messageCount: 2,
      createdAt: 10,
      updatedAt: 20,
    })
    chatApiMocks.getChatConversation.mockResolvedValueOnce(remote)
    chatApiMocks.getChatConversationMessages.mockResolvedValueOnce({
      items: [
        {
          id: 'assistant-2',
          role: 'assistant',
          content: '回答',
          createdAt: 20,
          position: 2,
          status: 'complete',
        },
        {
          id: 'user-1',
          role: 'user',
          content: '你好',
          createdAt: 20,
          position: 1,
          status: 'complete',
        },
      ],
      nextBeforePosition: null,
      hasMore: false,
    })

    const store = useChatStore()
    await store.hydrate('reversed-user')
    await store.loadConversationDetail('reversed-conversation')

    expect(store.activeConversation?.messages.map(({ id }) => id)).toEqual([
      'user-1',
      'assistant-2',
    ])
    expect(store.activeConversation).toMatchObject({
      messagesBeforePosition: null,
      messagesHasMore: false,
    })
  })

  it('加载更早消息时按 position 升序合并、去重并推进游标', async () => {
    const conversation = {
      id: 'paged-conversation',
      userId: 'paged-user',
      title: '分页历史',
      model: 'gpt-5',
      messages: [
        {
          id: 'user-3',
          role: 'user',
          content: '第三条',
          createdAt: 30,
          position: 3,
          status: 'complete',
        },
        {
          id: 'assistant-4',
          role: 'assistant',
          content: '第四条',
          createdAt: 40,
          position: 4,
          status: 'complete',
        },
      ],
      headMessageId: 'assistant-4',
      messageCount: 4,
      messagesBeforePosition: 3,
      messagesHasMore: true,
      createdAt: 10,
      updatedAt: 40,
      serverRevision: 1,
      serverVersion: 1,
    }
    persistedBuckets.set('paged-user', persistedChatState('paged-user', conversation))
    chatApiMocks.getChatConversationMessages.mockResolvedValueOnce({
      items: [
        {
          id: 'user-3',
          role: 'user',
          content: '第三条（服务端）',
          createdAt: 30,
          position: 3,
          status: 'complete',
        },
        {
          id: 'assistant-2',
          role: 'assistant',
          content: '第二条',
          createdAt: 20,
          position: 2,
          status: 'complete',
        },
        {
          id: 'user-1',
          role: 'user',
          content: '第一条',
          createdAt: 10,
          position: 1,
          status: 'complete',
        },
      ],
      nextBeforePosition: null,
      hasMore: false,
    })

    const store = useChatStore()
    await store.hydrate('paged-user')
    await store.loadOlderConversationMessages('paged-conversation')

    expect(chatApiMocks.getChatConversationMessages).toHaveBeenCalledWith(
      'paged-conversation',
      { beforePosition: 3, limit: 100 },
    )
    expect(store.activeConversation?.messages.map(({ id }) => id)).toEqual([
      'user-1',
      'assistant-2',
      'user-3',
      'assistant-4',
    ])
    expect(store.activeConversation?.messages.find(({ id }) => id === 'user-3')?.content)
      .toBe('第三条（服务端）')
    expect(store.activeConversation).toMatchObject({
      messagesBeforePosition: null,
      messagesHasMore: false,
    })
  })

  it('attempt 只给 assistant 补 position 时仍保留同轮 user 在前', async () => {
    const conversation = {
      id: 'mixed-position-conversation',
      userId: 'mixed-position-user',
      title: '混合位置',
      model: 'gpt-5',
      messages: [
        {
          id: 'user-z',
          role: 'user',
          content: '你好',
          createdAt: 20,
          status: 'complete',
        },
        {
          id: 'assistant-a',
          role: 'assistant',
          content: '旧回答',
          createdAt: 20,
          status: 'stopped',
          attemptId: 'attempt-mixed',
        },
      ],
      createdAt: 10,
      updatedAt: 20,
      serverRevision: 1,
      serverVersion: 1,
    }
    persistedBuckets.set(
      'mixed-position-user',
      persistedChatState('mixed-position-user', conversation),
    )
    chatApiMocks.getChatAttempt.mockResolvedValueOnce({
      attemptId: 'attempt-mixed',
      conversationId: 'mixed-position-conversation',
      assistantMessageId: 'assistant-a',
      status: 'processing',
      assistantMessage: {
        id: 'assistant-a',
        role: 'assistant',
        content: '服务端回答',
        createdAt: 20,
        position: 2,
        status: 'stopped',
        attemptId: 'attempt-mixed',
      },
    })

    const store = useChatStore()
    await store.hydrate('mixed-position-user')
    await store.recoverAttempt('mixed-position-conversation', 'assistant-a')

    expect(store.activeConversation?.messages.map(({ id }) => id)).toEqual([
      'user-z',
      'assistant-a',
    ])
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

  it('rejects an older server streaming Activity but accepts a server terminal snapshot', async () => {
    const store = useChatStore()
    await store.hydrate('activity-server-merge')
    const conversation = store.createConversation('gpt-5')!
    const assistant = store.addMessage(conversation.id, {
      role: 'assistant',
      content: 'answer',
      status: 'streaming',
    })!
    store.startStreaming(conversation.id, assistant.id)
    const local = reasoningActivity('resp-merge', 'new local stream', {
      updatedAt: 500,
      lastSequenceNumber: 10,
    })
    local.items[0]!.updatedAt = 500
    local.items[0]!.lastSequenceNumber = 10
    local.items[0]!.parts[0]!.updatedAt = 500
    local.items[0]!.parts[0]!.lastSequenceNumber = 10
    store.upsertMessageActivity(conversation.id, assistant.id, local)

    const olderStreaming = reasoningActivity('resp-merge', 'old server stream', {
      updatedAt: 50,
      lastSequenceNumber: 2,
    })
    chatApiMocks.getChatConversation.mockResolvedValueOnce(serverConversation(conversation.id, {
      headMessageId: assistant.id,
      messageCount: 1,
      messages: [{ ...assistant, activities: [olderStreaming] }],
    }))
    await store.loadConversationDetail(conversation.id)
    let mergedAssistant = store.conversations
      .find(({ id }) => id === conversation.id)
      ?.messages.find(({ id }) => id === assistant.id)
    expect(mergedAssistant?.activities?.[0]?.items[0]?.parts[0]?.text).toBe('new local stream')
    expect(mergedAssistant?.activities?.[0]?.status).toBe('streaming')

    const terminal = reasoningActivity('resp-merge', 'server final', {
      status: 'completed',
      updatedAt: 60,
      completedAt: 60,
      lastSequenceNumber: 3,
    })
    terminal.items[0]!.status = 'completed'
    terminal.items[0]!.updatedAt = 60
    terminal.items[0]!.completedAt = 60
    terminal.items[0]!.lastSequenceNumber = 3
    terminal.items[0]!.parts[0]!.status = 'completed'
    terminal.items[0]!.parts[0]!.updatedAt = 60
    terminal.items[0]!.parts[0]!.completedAt = 60
    terminal.items[0]!.parts[0]!.lastSequenceNumber = 3
    chatApiMocks.getChatConversation.mockResolvedValueOnce(serverConversation(conversation.id, {
      headMessageId: assistant.id,
      messageCount: 1,
      messages: [{ ...assistant, activities: [terminal] }],
    }))
    await store.loadConversationDetail(conversation.id)

    mergedAssistant = store.conversations
      .find(({ id }) => id === conversation.id)
      ?.messages.find(({ id }) => id === assistant.id)
    expect(mergedAssistant?.activities?.[0]).toMatchObject({
      status: 'completed',
      items: [{ parts: [{ text: 'server final', status: 'completed' }] }],
    })
    store.stopStreaming()
  })

  it('does not let a processing history checkpoint freeze the current live Activity', async () => {
    const store = useChatStore()
    await store.hydrate('activity-live-checkpoint')
    const conversation = store.createConversation('gpt-5')!
    const assistant = store.addMessage(conversation.id, {
      role: 'assistant',
      content: 'live answer',
      status: 'streaming',
    })!
    store.startStreaming(conversation.id, assistant.id)

    const local = reasoningActivity('resp-live-checkpoint', 'new local summary', {
      updatedAt: 500,
      lastSequenceNumber: 10,
    })
    local.items[0]!.updatedAt = 500
    local.items[0]!.lastSequenceNumber = 10
    local.items[0]!.parts[0]!.updatedAt = 500
    local.items[0]!.parts[0]!.lastSequenceNumber = 10
    store.upsertMessageActivity(conversation.id, assistant.id, local)

    const staleCheckpoint = reasoningActivity('resp-live-checkpoint', 'old server summary', {
      status: 'disconnected',
      updatedAt: 100,
      completedAt: 100,
      lastSequenceNumber: 3,
    })
    staleCheckpoint.items[0]!.status = 'disconnected'
    staleCheckpoint.items[0]!.updatedAt = 100
    staleCheckpoint.items[0]!.completedAt = 100
    staleCheckpoint.items[0]!.lastSequenceNumber = 3
    staleCheckpoint.items[0]!.parts[0]!.status = 'disconnected'
    staleCheckpoint.items[0]!.parts[0]!.updatedAt = 100
    staleCheckpoint.items[0]!.parts[0]!.completedAt = 100
    staleCheckpoint.items[0]!.parts[0]!.lastSequenceNumber = 3
    chatApiMocks.getChatConversation.mockResolvedValueOnce(serverConversation(conversation.id, {
      headMessageId: assistant.id,
      messageCount: 1,
      messages: [{
        ...assistant,
        status: 'stopped',
        finishReason: 'interrupted',
        activities: [staleCheckpoint],
      }],
    }))

    await store.loadConversationDetail(conversation.id)
    let mergedActivity = store.conversations
      .find(({ id }) => id === conversation.id)
      ?.messages.find(({ id }) => id === assistant.id)
      ?.activities?.[0]
    expect(mergedActivity).toMatchObject({
      status: 'streaming',
      lastSequenceNumber: 10,
      items: [{ parts: [{ text: 'new local summary', status: 'streaming' }] }],
    })

    const nextLocal = reasoningActivity('resp-live-checkpoint', 'newest local summary', {
      updatedAt: 600,
      lastSequenceNumber: 11,
    })
    nextLocal.items[0]!.updatedAt = 600
    nextLocal.items[0]!.lastSequenceNumber = 11
    nextLocal.items[0]!.parts[0]!.updatedAt = 600
    nextLocal.items[0]!.parts[0]!.lastSequenceNumber = 11
    expect(store.upsertMessageActivity(conversation.id, assistant.id, nextLocal)).toBe(true)
    mergedActivity = store.conversations
      .find(({ id }) => id === conversation.id)
      ?.messages.find(({ id }) => id === assistant.id)
      ?.activities?.[0]
    expect(mergedActivity).toMatchObject({
      status: 'streaming',
      lastSequenceNumber: 11,
      items: [{ parts: [{ text: 'newest local summary', status: 'streaming' }] }],
    })
    store.stopStreaming()
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
