import type {
  ChatHistoryOutboxMutation,
  ChatLegacyImportDecision,
} from '@/types/chat'

const CHAT_HISTORY_DATABASE_NAME = 'luoxueapi-chat'
const CHAT_HISTORY_DATABASE_VERSION = 2
const CHAT_HISTORY_STORE_NAME = 'history'
const CHAT_HISTORY_MAX_CONVERSATIONS = 50
const CHAT_HISTORY_MAX_FUTURE_SKEW_MS = 5 * 60 * 1000

interface PersistedChatHistoryRecord {
  userId: string
  state: unknown
  updatedAt: number
}

interface MergeableChatHistoryState {
  version: number
  clearRevision: number
  deletedConversationIds: string[]
  activeConversationId: string | null
  conversations: Array<Record<string, unknown>>
  serverVersion: number
  outbox: ChatHistoryOutboxMutation[]
  legacyImportDecision: ChatLegacyImportDecision
  legacyConversationIds: string[]
}

export interface ChatHistoryPersistence {
  load(userId: string): Promise<unknown | null>
  save(userId: string, state: unknown, mutation: ChatHistoryMutation): Promise<void>
  remove(userId: string): Promise<void>
}

export interface ChatHistoryMutation {
  upsertConversationIds?: string[]
  createdConversations?: Array<{ id: string; operationAt: number }>
  deletedConversationIds?: string[]
  sanitizeConversations?: Array<{ id: string; expectedUpdatedAt: number }>
  activeConversationChanged?: boolean
  clear?: boolean
  enqueueOutboxMutationIds?: string[]
  acknowledgedOutboxMutationIds?: string[]
  syncStateChanged?: boolean
}

export class ChatHistoryPersistenceUnavailableError extends Error {
  constructor(message = 'IndexedDB is unavailable') {
    super(message)
    this.name = 'ChatHistoryPersistenceUnavailableError'
  }
}

function resolveIndexedDB(): IDBFactory | null {
  try {
    return typeof globalThis.indexedDB === 'undefined' ? null : globalThis.indexedDB
  } catch {
    return null
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function normalizeClearRevision(value: unknown): number {
  return typeof value === 'number' && Number.isFinite(value) && value >= 0 ? value : 0
}

function isFiniteTimestamp(value: unknown): value is number {
  return typeof value === 'number'
    && Number.isFinite(value)
    && value >= 0
}

function isFiniteNumber(value: unknown): value is number {
  return typeof value === 'number' && Number.isFinite(value)
}

function isNonNegativeInteger(value: unknown): value is number {
  return isFiniteNumber(value) && Number.isSafeInteger(value) && value >= 0
}

function isChatReceiptStatus(value: unknown): boolean {
  return value === 'pending'
    || value === 'charged'
    || value === 'not_charged'
    || value === 'subscription'
    || value === 'failed'
}

function copyCanonicalAssistantMetadata(
  target: Record<string, unknown>,
  source: Record<string, unknown>,
): void {
  if (target.role !== 'assistant') return

  for (const field of [
    'attemptId',
    'receiptId',
    'requestedModel',
    'actualModel',
    'receiptCreatedAt',
    'supersededByMessageId',
  ] as const) {
    const value = source[field]
    if (typeof value === 'string' && value.trim()) target[field] = value.trim()
  }
  if (typeof source.billingType === 'string' && source.billingType.trim()) {
    target.billingType = source.billingType.trim()
  } else if (isFiniteNumber(source.billingType)) {
    target.billingType = source.billingType
  }
  if (isChatReceiptStatus(source.settlementStatus)) {
    target.settlementStatus = source.settlementStatus
  }
  for (const field of [
    'usageLogId',
    'inputTokens',
    'outputTokens',
    'cacheCreationTokens',
    'cacheReadTokens',
    'totalTokens',
  ] as const) {
    if (isNonNegativeInteger(source[field])) target[field] = source[field]
  }
  for (const field of [
    'grossCost',
    'chargedAmount',
    'balanceBefore',
    'balanceAfter',
  ] as const) {
    if (isFiniteNumber(source[field])) target[field] = source[field]
  }
  if (typeof source.excludedFromContext === 'boolean') {
    target.excludedFromContext = source.excludedFromContext
  }
}

function isImplausiblyFutureTimestamp(value: number, now: number): boolean {
  return value > now + CHAT_HISTORY_MAX_FUTURE_SKEW_MS
}

function compareConversationsByRecency(
  left: Record<string, unknown>,
  right: Record<string, unknown>,
  now: number,
): number {
  const leftUpdatedAt = Number(left.updatedAt)
  const rightUpdatedAt = Number(right.updatedAt)
  const leftIsFuture = isImplausiblyFutureTimestamp(leftUpdatedAt, now)
  const rightIsFuture = isImplausiblyFutureTimestamp(rightUpdatedAt, now)
  if (leftIsFuture !== rightIsFuture) return leftIsFuture ? 1 : -1
  return rightUpdatedAt - leftUpdatedAt
}

function normalizeFutureTimestamps(
  conversation: Record<string, unknown>,
  now: number,
): Record<string, unknown> {
  const normalized = { ...conversation }
  const createdAt = Number(normalized.createdAt)
  const updatedAt = Number(normalized.updatedAt)
  normalized.createdAt = isImplausiblyFutureTimestamp(createdAt, now) ? now : createdAt
  normalized.updatedAt = isImplausiblyFutureTimestamp(updatedAt, now)
    ? Math.max(Number(normalized.createdAt), now)
    : updatedAt
  normalized.messages = (normalized.messages as Array<Record<string, unknown>>).map((message) => {
    const messageCreatedAt = Number(message.createdAt)
    return isImplausiblyFutureTimestamp(messageCreatedAt, now)
      ? { ...message, createdAt: now }
      : message
  })
  return normalized
}

function canonicalMessage(value: unknown): Record<string, unknown> | null {
  if (!isRecord(value)) return null
  if (
    typeof value.id !== 'string'
    || value.id.length === 0
    || (value.role !== 'user' && value.role !== 'assistant')
    || typeof value.content !== 'string'
    || !isFiniteTimestamp(value.createdAt)
    || (
      value.status !== 'complete'
      && value.status !== 'streaming'
      && value.status !== 'stopped'
      && value.status !== 'error'
    )
  ) {
    return null
  }

  const message: Record<string, unknown> = {
    id: value.id,
    role: value.role,
    content: value.content,
    createdAt: value.createdAt,
    status: value.status,
  }
  if (typeof value.finishReason === 'string' || value.finishReason === null) {
    message.finishReason = value.finishReason
  }
  if (typeof value.errorCode === 'string') message.errorCode = value.errorCode
  if (typeof value.errorMessage === 'string') message.errorMessage = value.errorMessage
  if (isFiniteTimestamp(value.updatedAt)) message.updatedAt = value.updatedAt
  if (isNonNegativeInteger(value.position)) message.position = value.position
  copyCanonicalAssistantMetadata(message, value)
  return message
}

function canonicalConversation(
  value: unknown,
  expectedUserId: string,
): Record<string, unknown> | null {
  if (
    !isRecord(value)
    || typeof value.id !== 'string'
    || value.id.length === 0
    || value.userId !== expectedUserId
    || typeof value.title !== 'string'
    || typeof value.model !== 'string'
    || value.model.trim().length === 0
    || !isFiniteTimestamp(value.createdAt)
    || !isFiniteTimestamp(value.updatedAt)
    || value.updatedAt < value.createdAt
    || !Array.isArray(value.messages)
  ) {
    return null
  }

  const messageIds = new Set<string>()
  const messages = value.messages.reduce<Record<string, unknown>[]>((result, candidate) => {
    const message = canonicalMessage(candidate)
    const id = typeof message?.id === 'string' ? message.id : ''
    if (message && id && !messageIds.has(id)) {
      messageIds.add(id)
      result.push(message)
    }
    return result
  }, [])

  const conversation: Record<string, unknown> = {
    id: value.id,
    userId: expectedUserId,
    title: value.title,
    model: value.model.trim(),
    messages,
    createdAt: value.createdAt,
    updatedAt: value.updatedAt,
  }
  if (isNonNegativeInteger(value.serverRevision)) {
    conversation.serverRevision = value.serverRevision
  }
  if (isNonNegativeInteger(value.serverVersion)) {
    conversation.serverVersion = value.serverVersion
  }
  if (typeof value.headMessageId === 'string' || value.headMessageId === null) {
    conversation.headMessageId = value.headMessageId
  }
  if (isNonNegativeInteger(value.messageCount)) {
    conversation.messageCount = value.messageCount
  }
  if (isNonNegativeInteger(value.messagesBeforePosition) || value.messagesBeforePosition === null) {
    conversation.messagesBeforePosition = value.messagesBeforePosition
  }
  if (typeof value.messagesHasMore === 'boolean') {
    conversation.messagesHasMore = value.messagesHasMore
  }
  return conversation
}

function canonicalOutboxMutation(value: unknown): ChatHistoryOutboxMutation | null {
  if (!isRecord(value)) return null
  const mutationId = typeof value.mutationId === 'string' ? value.mutationId.trim() : ''
  const conversationId = typeof value.conversationId === 'string'
    ? value.conversationId.trim()
    : ''
  if (
    !mutationId
    || !conversationId
    || (value.type !== 'create' && value.type !== 'patch' && value.type !== 'delete')
    || !isFiniteTimestamp(value.createdAt)
  ) {
    return null
  }

  const mutation: ChatHistoryOutboxMutation = {
    mutationId,
    type: value.type,
    conversationId,
    createdAt: value.createdAt,
  }
  if (isNonNegativeInteger(value.revision)) mutation.revision = value.revision
  if (typeof value.title === 'string' && value.title.trim()) mutation.title = value.title.trim()
  if (typeof value.model === 'string' && value.model.trim()) mutation.model = value.model.trim()
  if (value.legacyImport === true) mutation.legacyImport = true
  return mutation
}

function parseMergeableState(
  value: unknown,
  expectedUserId: string,
): MergeableChatHistoryState | null {
  if (
    !isRecord(value)
    || !Number.isSafeInteger(value.version)
    || Number(value.version) < 1
    || !Array.isArray(value.conversations)
  ) {
    return null
  }

  const deletedConversationIds = new Set(
    Array.isArray(value.deletedConversationIds)
      ? value.deletedConversationIds.filter(
      (id): id is string => typeof id === 'string' && id.length > 0,
      )
      : [],
  )
  const conversationIds = new Set<string>()
  const conversations = value.conversations.reduce<Record<string, unknown>[]>((result, candidate) => {
    const conversation = canonicalConversation(candidate, expectedUserId)
    const id = typeof conversation?.id === 'string' ? conversation.id : ''
    if (conversation && id && !deletedConversationIds.has(id) && !conversationIds.has(id)) {
      conversationIds.add(id)
      result.push(conversation)
    }
      return result
    }, [])
  const outboxIds = new Set<string>()
  const outbox = (
    Array.isArray(value.outbox) ? value.outbox : []
  ).reduce<ChatHistoryOutboxMutation[]>((result, candidate) => {
    const mutation = canonicalOutboxMutation(candidate)
    if (mutation && !outboxIds.has(mutation.mutationId)) {
      outboxIds.add(mutation.mutationId)
      result.push(mutation)
    }
    return result
  }, [])

  const parsedVersion = Number(value.version)
  const legacyImportDecision: ChatLegacyImportDecision =
    value.legacyImportDecision === 'pending'
    || value.legacyImportDecision === 'accepted'
    || value.legacyImportDecision === 'declined'
      ? value.legacyImportDecision
      : (parsedVersion === 1 && conversations.length > 0 ? 'pending' : null)
  const legacyConversationIds = new Set(
    Array.isArray(value.legacyConversationIds)
      ? value.legacyConversationIds.filter(
        (id): id is string => typeof id === 'string' && conversationIds.has(id),
      )
      : (parsedVersion === 1 ? conversations.map(({ id }) => String(id)) : []),
  )

  return {
    version: parsedVersion,
    clearRevision: normalizeClearRevision(value.clearRevision),
    deletedConversationIds: [],
    activeConversationId: typeof value.activeConversationId === 'string'
      ? value.activeConversationId
      : null,
    conversations,
    serverVersion: isNonNegativeInteger(value.serverVersion) ? value.serverVersion : 0,
    outbox,
    legacyImportDecision,
    legacyConversationIds: [...legacyConversationIds],
  }
}

function normalizedMergeableState(
  state: MergeableChatHistoryState,
  normalizeTimestamps = true,
): MergeableChatHistoryState {
  const deleted = new Set(state.deletedConversationIds)
  const now = Date.now()
  let conversations = state.conversations
    .filter((conversation) => !deleted.has(String(conversation.id)))
    .sort((left, right) => compareConversationsByRecency(left, right, now))
    .slice(0, CHAT_HISTORY_MAX_CONVERSATIONS)
  if (normalizeTimestamps) {
    conversations = conversations.map((conversation) => normalizeFutureTimestamps(conversation, now))
  }
  const availableIds = new Set(conversations.map((conversation) => String(conversation.id)))

  return {
    version: state.version,
    clearRevision: state.clearRevision,
    deletedConversationIds: [...deleted],
    activeConversationId: state.activeConversationId && availableIds.has(state.activeConversationId)
      ? state.activeConversationId
      : (String(conversations[0]?.id ?? '') || null),
    conversations,
    serverVersion: state.serverVersion,
    outbox: state.outbox,
    legacyImportDecision: state.legacyImportDecision,
    legacyConversationIds: state.legacyConversationIds.filter((id) => availableIds.has(id)),
  }
}

// IndexedDB serializes readwrite transactions, so applying an explicit local
// mutation preserves unrelated tab data without accepting stale whole snapshots.
export function mergeChatHistoryStates(
  existing: unknown,
  incoming: unknown,
  expectedUserId: string,
  mutation: ChatHistoryMutation,
): unknown {
  const next = parseMergeableState(incoming, expectedUserId)
  if (!next) {
    const current = parseMergeableState(existing, expectedUserId)
    return current ? normalizedMergeableState(current) : null
  }

  const current = parseMergeableState(existing, expectedUserId)
  const canonicalCurrent = current
    ? normalizedMergeableState({
      ...current,
      version: next.version,
    }, false)
    : normalizedMergeableState({
      version: next.version,
      clearRevision: next.clearRevision,
      deletedConversationIds: [],
      activeConversationId: null,
      conversations: [],
      serverVersion: 0,
      outbox: [],
      legacyImportDecision: null,
      legacyConversationIds: [],
    }, false)

  const nextOutboxById = new Map(next.outbox.map((item) => [item.mutationId, item]))
  const outboxById = new Map(
    canonicalCurrent.outbox.map((item) => [item.mutationId, item]),
  )
  for (const id of new Set(mutation.enqueueOutboxMutationIds ?? [])) {
    const item = nextOutboxById.get(id)
    const existingItem = outboxById.get(id)
    if (item && (!existingItem || item.createdAt >= existingItem.createdAt)) {
      outboxById.set(id, item)
    }
  }
  for (const id of new Set(mutation.acknowledgedOutboxMutationIds ?? [])) {
    outboxById.delete(id)
  }
  const serverVersion = mutation.syncStateChanged
    ? Math.max(canonicalCurrent.serverVersion, next.serverVersion)
    : canonicalCurrent.serverVersion
  const legacyImportDecision = mutation.syncStateChanged
    ? next.legacyImportDecision
    : canonicalCurrent.legacyImportDecision
  const legacyConversationIds = mutation.syncStateChanged
    ? next.legacyConversationIds
    : canonicalCurrent.legacyConversationIds

  if (mutation.clear) {
    const clearAt = next.clearRevision > 0 ? next.clearRevision : Date.now()
    return normalizedMergeableState({
      version: next.version,
      clearRevision: Math.max(canonicalCurrent.clearRevision, clearAt),
      deletedConversationIds: [],
      activeConversationId: canonicalCurrent.activeConversationId,
      conversations: canonicalCurrent.conversations.filter(
        (conversation) => Number(conversation.createdAt) > clearAt,
      ),
      serverVersion,
      outbox: [...outboxById.values()],
      legacyImportDecision,
      legacyConversationIds,
    })
  }

  const nextById = new Map(next.conversations.map((conversation) => [String(conversation.id), conversation]))
  const byId = new Map(canonicalCurrent.conversations.map(
    (conversation) => [String(conversation.id), conversation],
  ))
  const createdAtById = new Map(
    (mutation.createdConversations ?? [])
      .filter(({ id, operationAt }) => id.length > 0 && Number.isFinite(operationAt))
      .map(({ id, operationAt }) => [id, operationAt]),
  )
  for (const target of mutation.sanitizeConversations ?? []) {
    const conversation = nextById.get(target.id)
    const currentConversation = byId.get(target.id)
    if (
      conversation
      && currentConversation
      && Number(currentConversation.updatedAt) === target.expectedUpdatedAt
    ) {
      byId.set(target.id, conversation)
    }
  }
  for (const id of new Set(mutation.upsertConversationIds ?? [])) {
    const conversation = nextById.get(id)
    if (!conversation) continue
    if (!byId.has(id)) {
      const operationAt = createdAtById.get(id)
      if (
        operationAt === undefined
        || operationAt !== Number(conversation.createdAt)
        || operationAt <= canonicalCurrent.clearRevision
      ) {
        continue
      }
    }
    byId.set(id, conversation)
  }
  for (const id of new Set(mutation.deletedConversationIds ?? [])) {
    byId.delete(id)
  }

  return normalizedMergeableState({
    version: next.version,
    clearRevision: canonicalCurrent.clearRevision,
    deletedConversationIds: [],
    activeConversationId: mutation.activeConversationChanged
      ? next.activeConversationId
      : canonicalCurrent.activeConversationId,
    conversations: [...byId.values()],
    serverVersion,
    outbox: [...outboxById.values()],
    legacyImportDecision,
    legacyConversationIds,
  })
}

function requestResult<T>(request: IDBRequest<T>): Promise<T> {
  return new Promise<T>((resolve, reject) => {
    request.onsuccess = () => resolve(request.result)
    request.onerror = () => reject(
      request.error ?? new ChatHistoryPersistenceUnavailableError('IndexedDB request failed'),
    )
  })
}

function transactionCompletion(transaction: IDBTransaction): Promise<void> {
  return new Promise<void>((resolve, reject) => {
    transaction.oncomplete = () => resolve()
    transaction.onerror = () => reject(
      transaction.error ?? new ChatHistoryPersistenceUnavailableError('IndexedDB transaction failed'),
    )
    transaction.onabort = () => reject(
      transaction.error ?? new ChatHistoryPersistenceUnavailableError('IndexedDB transaction was aborted'),
    )
  })
}

export class IndexedDBChatHistoryPersistence implements ChatHistoryPersistence {
  private databasePromise: Promise<IDBDatabase> | null = null

  private openDatabase(): Promise<IDBDatabase> {
    if (this.databasePromise) return this.databasePromise

    const factory = resolveIndexedDB()
    if (!factory) {
      return Promise.reject(new ChatHistoryPersistenceUnavailableError())
    }

    this.databasePromise = new Promise<IDBDatabase>((resolve, reject) => {
      const request = factory.open(CHAT_HISTORY_DATABASE_NAME, CHAT_HISTORY_DATABASE_VERSION)
      let settled = false

      request.onupgradeneeded = () => {
        const database = request.result
        if (!database.objectStoreNames.contains(CHAT_HISTORY_STORE_NAME)) {
          database.createObjectStore(CHAT_HISTORY_STORE_NAME, { keyPath: 'userId' })
        }
      }
      request.onsuccess = () => {
        const database = request.result
        if (settled) {
          database.close()
          return
        }
        settled = true
        database.onversionchange = () => {
          database.close()
          this.databasePromise = null
        }
        resolve(database)
      }
      request.onerror = () => {
        if (settled) return
        settled = true
        this.databasePromise = null
        reject(request.error ?? new ChatHistoryPersistenceUnavailableError('Unable to open IndexedDB'))
      }
      request.onblocked = () => {
        if (settled) return
        settled = true
        this.databasePromise = null
        reject(new ChatHistoryPersistenceUnavailableError('IndexedDB upgrade was blocked'))
      }
    })

    return this.databasePromise
  }

  async load(userId: string): Promise<unknown | null> {
    const database = await this.openDatabase()
    const transaction = database.transaction(CHAT_HISTORY_STORE_NAME, 'readonly')
    const completed = transactionCompletion(transaction)
    const request = transaction.objectStore(CHAT_HISTORY_STORE_NAME).get(userId)
    const record = await requestResult<PersistedChatHistoryRecord | undefined>(request)
    await completed
    return record?.state ?? null
  }

  async save(userId: string, state: unknown, mutation: ChatHistoryMutation): Promise<void> {
    const database = await this.openDatabase()
    const transaction = database.transaction(CHAT_HISTORY_STORE_NAME, 'readwrite')
    const completed = transactionCompletion(transaction)
    const store = transaction.objectStore(CHAT_HISTORY_STORE_NAME)
    const existing = await requestResult<PersistedChatHistoryRecord | undefined>(store.get(userId))
    store.put({
      userId,
      state: mergeChatHistoryStates(existing?.state ?? null, state, userId, mutation),
      updatedAt: Date.now(),
    } satisfies PersistedChatHistoryRecord)
    await completed
  }

  async remove(userId: string): Promise<void> {
    const database = await this.openDatabase()
    const transaction = database.transaction(CHAT_HISTORY_STORE_NAME, 'readwrite')
    const completed = transactionCompletion(transaction)
    transaction.objectStore(CHAT_HISTORY_STORE_NAME).delete(userId)
    await completed
  }
}

export const chatHistoryPersistence: ChatHistoryPersistence = new IndexedDBChatHistoryPersistence()
