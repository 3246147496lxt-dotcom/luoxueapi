import { defineStore } from 'pinia'
import { computed, ref, shallowRef } from 'vue'
import {
  ChatAPIError,
  createChatConversation as createRemoteConversation,
  deleteChatConversation as deleteRemoteConversation,
  getChatAttempt,
  getChatConversation,
  getChatConversationMessages,
  getChatSync,
  isAbortError,
  listChatConversations,
  patchChatConversation as patchRemoteConversation,
  searchChatConversations,
} from '@/api/chat'
import {
  cloneChatActivity,
  finalizeChatActivityPartText,
  isTerminalChatActivityStatus,
  mergeChatActivities,
  normalizeChatActivities,
} from '@/features/chat/activity'
import { chatHistoryPersistence } from '@/features/chat/persistence'
import type { ChatHistoryMutation } from '@/features/chat/persistence'
import { isSettlementFailureWithCompletedDelivery } from '@/features/chat/settlementDelivery'
import type {
  ChatAttempt,
  ChatActivity,
  ChatActivityError,
  ChatActivityStatus,
  ChatAttachment,
  ChatConversation,
  ChatHistoryOutboxMutation,
  ChatHistorySyncStatus,
  ChatImportedMessage,
  ChatLegacyImportDecision,
  ChatMessage,
  ChatReceiptStatus,
  ChatServerConversation,
  ChatServerMessage,
} from '@/types/chat'

export const MAX_CHAT_CONVERSATIONS = 50

const STORAGE_VERSION = 5
const DEFAULT_CONVERSATION_TITLE = '新对话'
const STREAM_PERSIST_THROTTLE_MS = 300
const MAX_FUTURE_TIMESTAMP_SKEW_MS = 5 * 60 * 1000
const TIMESTAMP_INCREMENT_MS = 0.001
const MAX_LEGACY_IMPORT_REQUEST_BYTES = 2 * 1024 * 1024
const LEGACY_IMPORT_ENVELOPE_RESERVE_BYTES = 4 * 1024
const MAX_LEGACY_IMPORT_MESSAGES = 200

interface PersistedChatState {
  version: typeof STORAGE_VERSION
  clearRevision: number
  deletedConversationIds: string[]
  activeConversationId: string | null
  activeConversationSelectionResolved: boolean
  conversations: ChatConversation[]
  serverVersion: number
  outbox: ChatHistoryOutboxMutation[]
  legacyImportDecision: ChatLegacyImportDecision
  legacyConversationIds: string[]
}

interface ChatStreamError {
  code?: string
  message: string
}

export interface ChatPersistenceDiagnostic {
  code: 'CHAT_PERSISTENCE_READ_FAILED'
    | 'CHAT_PERSISTENCE_WRITE_FAILED'
    | 'CHAT_PERSISTENCE_RETRY_FAILED'
    | 'CHAT_PERSISTENCE_PAYLOAD_INVALID'
    | 'CHAT_PERSISTENCE_PROBE_FAILED'
  message: string
  occurredAt: number
}

export interface CreateChatMessage {
  id?: string
  role: ChatMessage['role']
  content: string
  createdAt?: number
  status?: ChatMessage['status']
  finishReason?: string | null
  errorCode?: string
  errorMessage?: string
  attemptId?: string
  receiptId?: string
  settlementStatus?: ChatReceiptStatus
  usageLogId?: number
  requestedModel?: string
  actualModel?: string
  inputTokens?: number
  outputTokens?: number
  cacheCreationTokens?: number
  cacheReadTokens?: number
  totalTokens?: number
  grossCost?: number
  chargedAmount?: number
  billingType?: string | number
  balanceBefore?: number
  balanceAfter?: number
  receiptCreatedAt?: string
  excludedFromContext?: boolean
  supersededByMessageId?: string
  attachments?: ChatAttachment[]
  activities?: ChatActivity[]
}

export type ChatMessagePatch = Partial<
  Pick<
    ChatMessage,
    | 'content'
    | 'status'
    | 'finishReason'
    | 'errorCode'
    | 'errorMessage'
    | 'attemptId'
    | 'receiptId'
    | 'settlementStatus'
    | 'usageLogId'
    | 'requestedModel'
    | 'actualModel'
    | 'inputTokens'
    | 'outputTokens'
    | 'cacheCreationTokens'
    | 'cacheReadTokens'
    | 'totalTokens'
    | 'grossCost'
    | 'chargedAmount'
    | 'billingType'
    | 'balanceBefore'
    | 'balanceAfter'
    | 'receiptCreatedAt'
    | 'excludedFromContext'
    | 'supersededByMessageId'
    | 'activities'
  >
> & { pendingStopRequestedAt?: number | null }

let generatedIdCounter = 0

function createId(prefix: 'conversation' | 'message' | 'mutation'): string {
  try {
    if (typeof globalThis.crypto?.randomUUID === 'function') {
      return `${prefix}-${globalThis.crypto.randomUUID()}`
    }
  } catch {
    // Fall back for restricted browser contexts.
  }

  generatedIdCounter += 1
  return `${prefix}-${Date.now().toString(36)}-${generatedIdCounter.toString(36)}`
}

function normalizeUserId(value: string | number | null | undefined): string | null {
  if (value === null || value === undefined) return null
  const normalized = String(value).trim()
  return normalized || null
}

function epochNow(): number {
  try {
    const timeOrigin = globalThis.performance?.timeOrigin
    const elapsed = globalThis.performance?.now()
    if (Number.isFinite(timeOrigin) && Number.isFinite(elapsed)) {
      return Number(timeOrigin) + Number(elapsed)
    }
  } catch {
    // Date.now remains available in restricted browser contexts.
  }
  return Date.now()
}

function buildLegacyImportedMessages(messages: ChatMessage[]): ChatImportedMessage[] {
  const encoder = new TextEncoder()
  const byteBudget = MAX_LEGACY_IMPORT_REQUEST_BYTES - LEGACY_IMPORT_ENVELOPE_RESERVE_BYTES
  const importedMessages: ChatImportedMessage[] = []
  let encodedBytes = 2

  for (const message of messages.slice(0, MAX_LEGACY_IMPORT_MESSAGES)) {
    const imported: ChatImportedMessage = {
      id: message.id,
      role: message.role,
      content: message.content,
      status: message.status === 'complete' ? 'completed' : 'interrupted',
      createdAt: message.createdAt,
    }
    const messageBytes = encoder.encode(JSON.stringify(imported)).byteLength
      + (importedMessages.length > 0 ? 1 : 0)
    if (encodedBytes + messageBytes > byteBudget) break
    importedMessages.push(imported)
    encodedBytes += messageBytes
  }

  return importedMessages
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
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

function isChatReceiptStatus(value: unknown): value is ChatReceiptStatus {
  return value === 'pending'
    || value === 'charged'
    || value === 'not_charged'
    || value === 'subscription'
    || value === 'failed'
}

function sanitizeAttachment(value: unknown): ChatAttachment | null {
  if (!isRecord(value)) return null
  const id = typeof value.id === 'string' ? value.id.trim() : ''
  const name = typeof value.name === 'string' ? value.name.trim() : ''
  const mimeType = typeof value.mimeType === 'string' ? value.mimeType.trim() : ''
  const expiresAt = typeof value.expiresAt === 'string' ? value.expiresAt.trim() : ''
  if (
    !id
    || !name
    || !mimeType
    || !expiresAt
    || (value.kind !== 'image' && value.kind !== 'document')
    || (value.status !== 'ready' && value.status !== 'expired')
    || !isNonNegativeInteger(value.size)
  ) return null

  const attachment: ChatAttachment = {
    id,
    name,
    kind: value.kind,
    mimeType,
    size: value.size,
    status: value.status,
    expiresAt,
  }
  if (isNonNegativeInteger(value.pageCount) && value.pageCount > 0) {
    attachment.pageCount = value.pageCount
  }
  if (isNonNegativeInteger(value.width) && value.width > 0) attachment.width = value.width
  if (isNonNegativeInteger(value.height) && value.height > 0) attachment.height = value.height
  return attachment
}

function sanitizedAttachments(value: unknown): ChatAttachment[] {
  if (!Array.isArray(value)) return []
  const ids = new Set<string>()
  return value.reduce<ChatAttachment[]>((result, candidate) => {
    const attachment = sanitizeAttachment(candidate)
    if (attachment && !ids.has(attachment.id)) {
      ids.add(attachment.id)
      result.push(attachment)
    }
    return result
  }, [])
}

function sanitizedActivities(
  value: unknown,
  fallbackStartedAt: number,
  markDisconnected = false,
): ChatActivity[] {
  return normalizeChatActivities(value, { fallbackStartedAt, markDisconnected })
}

function copyAssistantMetadata(
  target: ChatMessage,
  source: Record<string, unknown>,
): void {
  if (target.role !== 'assistant') return

  if (typeof source.attemptId === 'string' && source.attemptId.trim()) {
    target.attemptId = source.attemptId.trim()
  }
  if (isFiniteTimestamp(source.pendingStopRequestedAt)) {
    target.pendingStopRequestedAt = source.pendingStopRequestedAt
  }
  if (typeof source.receiptId === 'string' && source.receiptId.trim()) {
    target.receiptId = source.receiptId.trim()
  }
  if (isChatReceiptStatus(source.settlementStatus)) {
    target.settlementStatus = source.settlementStatus
  }
  if (isNonNegativeInteger(source.usageLogId)) target.usageLogId = source.usageLogId
  if (typeof source.requestedModel === 'string' && source.requestedModel.trim()) {
    target.requestedModel = source.requestedModel.trim()
  }
  if (typeof source.actualModel === 'string' && source.actualModel.trim()) {
    target.actualModel = source.actualModel.trim()
  }
  if (isNonNegativeInteger(source.inputTokens)) target.inputTokens = source.inputTokens
  if (isNonNegativeInteger(source.outputTokens)) target.outputTokens = source.outputTokens
  if (isNonNegativeInteger(source.cacheCreationTokens)) {
    target.cacheCreationTokens = source.cacheCreationTokens
  }
  if (isNonNegativeInteger(source.cacheReadTokens)) target.cacheReadTokens = source.cacheReadTokens
  if (isNonNegativeInteger(source.totalTokens)) target.totalTokens = source.totalTokens
  if (isFiniteNumber(source.grossCost)) target.grossCost = source.grossCost
  if (isFiniteNumber(source.chargedAmount)) target.chargedAmount = source.chargedAmount
  if (typeof source.billingType === 'string' && source.billingType.trim()) {
    target.billingType = source.billingType.trim()
  } else if (isFiniteNumber(source.billingType)) {
    target.billingType = source.billingType
  }
  if (isFiniteNumber(source.balanceBefore)) target.balanceBefore = source.balanceBefore
  if (isFiniteNumber(source.balanceAfter)) target.balanceAfter = source.balanceAfter
  if (typeof source.receiptCreatedAt === 'string' && source.receiptCreatedAt.trim()) {
    target.receiptCreatedAt = source.receiptCreatedAt.trim()
  }
  if (typeof source.excludedFromContext === 'boolean') {
    target.excludedFromContext = source.excludedFromContext
  }
  if (
    typeof source.supersededByMessageId === 'string'
    && source.supersededByMessageId.trim()
  ) {
    target.supersededByMessageId = source.supersededByMessageId.trim()
  }
}

function isImplausiblyFutureTimestamp(value: number, now: number): boolean {
  return value > now + MAX_FUTURE_TIMESTAMP_SKEW_MS
}

function compareConversationsByRecency(
  left: ChatConversation,
  right: ChatConversation,
  now: number,
): number {
  const leftIsFuture = isImplausiblyFutureTimestamp(left.updatedAt, now)
  const rightIsFuture = isImplausiblyFutureTimestamp(right.updatedAt, now)
  if (leftIsFuture !== rightIsFuture) return leftIsFuture ? 1 : -1
  return right.updatedAt - left.updatedAt
}

function normalizeFutureTimestamps(conversation: ChatConversation, now: number): ChatConversation {
  if (isImplausiblyFutureTimestamp(conversation.createdAt, now)) {
    conversation.createdAt = now
  }
  if (isImplausiblyFutureTimestamp(conversation.updatedAt, now)) {
    conversation.updatedAt = Math.max(conversation.createdAt, now)
  }
  for (const message of conversation.messages) {
    if (isImplausiblyFutureTimestamp(message.createdAt, now)) message.createdAt = now
  }
  return conversation
}

function sanitizeMessage(value: unknown): ChatMessage | null {
  if (!isRecord(value)) return null
  if (typeof value.id !== 'string' || !value.id) return null
  if (value.role !== 'user' && value.role !== 'assistant') return null
  if (typeof value.content !== 'string') return null
  if (!isFiniteTimestamp(value.createdAt)) return null

  const validStatuses: ChatMessage['status'][] = ['complete', 'streaming', 'stopped', 'error']
  if (!validStatuses.includes(value.status as ChatMessage['status'])) return null

  const restoredStatus = value.status as ChatMessage['status']
  const status: ChatMessage['status'] = restoredStatus === 'streaming' ? 'stopped' : restoredStatus
  const message: ChatMessage = {
    id: value.id,
    role: value.role,
    content: value.content,
    createdAt: value.createdAt,
    status,
  }

  if (isFiniteTimestamp(value.updatedAt)) message.updatedAt = value.updatedAt
  if (isNonNegativeInteger(value.position)) message.position = value.position
  if (typeof value.finishReason === 'string' || value.finishReason === null) {
    message.finishReason = value.finishReason
  } else if (restoredStatus === 'streaming') {
    message.finishReason = 'interrupted'
  }
  if (typeof value.errorCode === 'string') message.errorCode = value.errorCode
  if (typeof value.errorMessage === 'string') message.errorMessage = value.errorMessage
  if (isSettlementFailureWithCompletedDelivery(message)) {
    message.status = 'complete'
    delete message.errorCode
    delete message.errorMessage
  }
  copyAssistantMetadata(message, value)
  const attachments = sanitizedAttachments(value.attachments)
  if (attachments.length > 0) message.attachments = attachments
  if (message.role === 'assistant') {
    const activities = sanitizedActivities(
      value.activities,
      value.createdAt,
      true,
    )
    if (activities.length > 0) message.activities = activities
  }

  return message
}

function sanitizeConversation(value: unknown, userId: string): ChatConversation | null {
  if (!isRecord(value)) return null
  if (typeof value.id !== 'string' || !value.id) return null
  if (value.userId !== userId) return null
  if (typeof value.title !== 'string') return null
  if (typeof value.model !== 'string' || !value.model.trim()) return null
  if (!Array.isArray(value.messages)) return null
  if (!isFiniteTimestamp(value.createdAt) || !isFiniteTimestamp(value.updatedAt)) return null
  if (value.updatedAt < value.createdAt) return null

  const messageIds = new Set<string>()
  const messages = value.messages.reduce<ChatMessage[]>((result, candidate) => {
    const message = sanitizeMessage(candidate)
    if (message && !messageIds.has(message.id)) {
      messageIds.add(message.id)
      result.push(message)
    }
    return result
  }, [])

  const conversation: ChatConversation = {
    id: value.id,
    userId,
    title: value.title.trim() || DEFAULT_CONVERSATION_TITLE,
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

function createPersistedMessage(message: ChatMessage): ChatMessage {
  const persisted: ChatMessage = {
    id: message.id,
    role: message.role,
    content: message.content,
    createdAt: message.createdAt,
    status: message.status,
  }

  if (message.updatedAt !== undefined) persisted.updatedAt = message.updatedAt
  if (message.position !== undefined) persisted.position = message.position
  if (message.finishReason !== undefined) persisted.finishReason = message.finishReason
  if (message.errorCode !== undefined) persisted.errorCode = message.errorCode
  if (message.errorMessage !== undefined) persisted.errorMessage = message.errorMessage
  copyAssistantMetadata(persisted, message as unknown as Record<string, unknown>)
  const attachments = sanitizedAttachments(message.attachments)
  if (attachments.length > 0) persisted.attachments = attachments
  if (message.role === 'assistant') {
    const activities = sanitizedActivities(message.activities, message.createdAt)
    if (activities.length > 0) persisted.activities = activities.map(cloneChatActivity)
  }
  return persisted
}

function createPersistedConversation(conversation: ChatConversation): ChatConversation {
  const persisted: ChatConversation = {
    id: conversation.id,
    userId: conversation.userId,
    title: conversation.title,
    model: conversation.model,
    messages: conversation.messages.map(createPersistedMessage),
    createdAt: conversation.createdAt,
    updatedAt: conversation.updatedAt,
  }
  if (conversation.serverRevision !== undefined) {
    persisted.serverRevision = conversation.serverRevision
  }
  if (conversation.serverVersion !== undefined) {
    persisted.serverVersion = conversation.serverVersion
  }
  if (conversation.headMessageId !== undefined) {
    persisted.headMessageId = conversation.headMessageId
  }
  if (conversation.messageCount !== undefined) {
    persisted.messageCount = conversation.messageCount
  }
  if (conversation.messagesBeforePosition !== undefined) {
    persisted.messagesBeforePosition = conversation.messagesBeforePosition
  }
  if (conversation.messagesHasMore !== undefined) {
    persisted.messagesHasMore = conversation.messagesHasMore
  }
  return persisted
}

function sanitizeOutboxMutation(value: unknown): ChatHistoryOutboxMutation | null {
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

export const useChatStore = defineStore('chat', () => {
  const userId = ref<string | null>(null)
  const conversations = ref<ChatConversation[]>([])
  const activeConversationId = ref<string | null>(null)
  const hydrated = ref(false)
  const hydrating = ref(false)
  const persistenceAvailable = ref(true)
  const persistenceDiagnostic = ref<ChatPersistenceDiagnostic | null>(null)
  const serverVersion = ref(0)
  const outbox = ref<ChatHistoryOutboxMutation[]>([])
  const syncStatus = ref<ChatHistorySyncStatus>('idle')
  const syncError = ref<string | null>(null)
  const serverHistoryAvailable = ref<boolean | null>(null)
  const legacyImportDecision = ref<ChatLegacyImportDecision>(null)
  const legacyConversationIds = ref<string[]>([])
  const conversationCursor = ref<string | null>(null)
  const conversationsHaveMore = ref(false)
  const loadingConversationPage = ref(false)
  const searchResults = ref<ChatConversation[]>([])
  const searchingHistory = ref(false)
  const loadingConversationMessages = ref(new Set<string>())

  const streamingConversationId = ref<string | null>(null)
  const streamingMessageId = ref<string | null>(null)
  const streamError = ref<ChatStreamError | null>(null)
  const streamController = shallowRef<AbortController | null>(null)
  let persistTimeoutId: ReturnType<typeof setTimeout> | null = null
  let persistenceChain: Promise<void> = Promise.resolve()
  let hydrationVersion = 0
  let hydrationPromise: Promise<void> | null = null
  let syncPromise: Promise<void> | null = null
  let syncController: AbortController | null = null
  let completionPreparationController: AbortController | null = null
  let searchRequestSequence = 0
  let clearRevision = 0
  let activeConversationSelectionResolved = false
  let activeConversationSelectionRevision = 0
  let pendingInitialConversationSelection = false
  let pendingServerSelectionReplacementId: string | null = null
  let deletedConversationIds = new Set<string>()
  let pendingPersistConversationIds = new Set<string>()
  type PersistenceOperation = () => Promise<void>
  let failedPersistenceOperations: PersistenceOperation[] = []

  interface PersistenceContext {
    userId: string | null
    hydrationVersion: number
  }

  const activeConversation = computed(() =>
    conversations.value.find((conversation) => conversation.id === activeConversationId.value) ?? null
  )
  const hasConversations = computed(() => conversations.value.length > 0)
  const legacyImportRequired = computed(() => (
    legacyImportDecision.value === 'pending'
    && legacyConversationIds.value.some((id) => Boolean(findConversation(id)))
  ))
  const isStreaming = computed(() =>
    streamingConversationId.value !== null && streamingMessageId.value !== null
  )

  function findConversation(conversationId: string): ChatConversation | undefined {
    return conversations.value.find((conversation) => conversation.id === conversationId)
  }

  function findMessage(conversationId: string, messageId: string): ChatMessage | undefined {
    return findConversation(conversationId)?.messages.find((message) => message.id === messageId)
  }

  function clearStreamingRuntime(abort: boolean): void {
    if (abort) streamController.value?.abort()
    streamController.value = null
    streamingConversationId.value = null
    streamingMessageId.value = null
  }

  function cancelScheduledPersist(): void {
    if (persistTimeoutId === null) return
    clearTimeout(persistTimeoutId)
    persistTimeoutId = null
  }

  function resetState(): void {
    cancelScheduledPersist()
    clearStreamingRuntime(true)
    syncController?.abort()
    syncController = null
    syncPromise = null
    completionPreparationController?.abort()
    completionPreparationController = null
    searchRequestSequence += 1
    userId.value = null
    conversations.value = []
    activeConversationId.value = null
    streamError.value = null
    hydrated.value = false
    hydrating.value = false
    persistenceDiagnostic.value = null
    serverVersion.value = 0
    outbox.value = []
    syncStatus.value = 'idle'
    syncError.value = null
    serverHistoryAvailable.value = null
    legacyImportDecision.value = null
    legacyConversationIds.value = []
    conversationCursor.value = null
    conversationsHaveMore.value = false
    loadingConversationPage.value = false
    searchResults.value = []
    searchingHistory.value = false
    loadingConversationMessages.value = new Set<string>()
    clearRevision = 0
    activeConversationSelectionResolved = false
    activeConversationSelectionRevision = 0
    pendingInitialConversationSelection = false
    pendingServerSelectionReplacementId = null
    deletedConversationIds = new Set<string>()
    pendingPersistConversationIds = new Set<string>()
  }

  function capturePersistenceContext(contextUserId = userId.value): PersistenceContext {
    return {
      userId: contextUserId,
      hydrationVersion,
    }
  }

  function isCurrentPersistenceContext(context: PersistenceContext): boolean {
    return context.userId === userId.value && context.hydrationVersion === hydrationVersion
  }

  function setPersistenceAvailable(context: PersistenceContext, available: boolean): void {
    if (isCurrentPersistenceContext(context)) persistenceAvailable.value = available
  }

  function recordPersistenceFailure(
    context: PersistenceContext,
    code: ChatPersistenceDiagnostic['code'],
    error: unknown,
  ): void {
    if (!isCurrentPersistenceContext(context)) return
    persistenceAvailable.value = false
    // Keep diagnostics useful without copying IndexedDB payloads, user text,
    // filesystem paths, or arbitrary exception messages into application state.
    const errorName = error instanceof Error && error.name.trim()
      ? error.name.trim()
      : 'UnknownError'
    persistenceDiagnostic.value = {
      code,
      message: errorName.slice(0, 80),
      occurredAt: Date.now(),
    }
  }

  async function runPersistenceOperations(operation?: PersistenceOperation): Promise<void> {
    const operations = failedPersistenceOperations
    failedPersistenceOperations = []
    if (operation) operations.push(operation)

    for (let index = 0; index < operations.length; index += 1) {
      try {
        await operations[index]()
      } catch (error) {
        failedPersistenceOperations.push(...operations.slice(index))
        throw error
      }
    }
  }

  function enqueuePersistence(
    operation: PersistenceOperation,
    context = capturePersistenceContext(),
  ): void {
    persistenceChain = persistenceChain
      .then(() => runPersistenceOperations(operation))
      .then(() => {
        setPersistenceAvailable(context, true)
      })
      .catch((error) => {
        recordPersistenceFailure(context, 'CHAT_PERSISTENCE_WRITE_FAILED', error)
      })
  }

  async function retryFailedPersistenceOperations(context: PersistenceContext): Promise<boolean> {
    await persistenceChain
    if (failedPersistenceOperations.length === 0) return true

    let succeeded = false
    persistenceChain = runPersistenceOperations()
      .then(() => {
        setPersistenceAvailable(context, true)
        succeeded = true
      })
      .catch((error) => {
        recordPersistenceFailure(context, 'CHAT_PERSISTENCE_RETRY_FAILED', error)
      })
    await persistenceChain
    return succeeded
  }

  function persistedSnapshot(): PersistedChatState {
    return {
      version: STORAGE_VERSION,
      clearRevision,
      deletedConversationIds: [...deletedConversationIds],
      activeConversationId: activeConversationId.value,
      activeConversationSelectionResolved,
      conversations: conversations.value.map(createPersistedConversation),
      serverVersion: serverVersion.value,
      outbox: outbox.value.map((mutation) => ({ ...mutation })),
      legacyImportDecision: legacyImportDecision.value,
      legacyConversationIds: [...legacyConversationIds.value],
    }
  }

  function persist(mutation: ChatHistoryMutation = {}): void {
    cancelScheduledPersist()
    if (!userId.value) {
      pendingPersistConversationIds.clear()
      return
    }
    const bucketUserId = userId.value
    const context = capturePersistenceContext(bucketUserId)
    const snapshot = persistedSnapshot()
    const upsertConversationIds = new Set([
      ...pendingPersistConversationIds,
      ...(mutation.upsertConversationIds ?? []),
    ])
    pendingPersistConversationIds.clear()
    const persistedMutation: ChatHistoryMutation = {
      ...mutation,
      ...(upsertConversationIds.size > 0
        ? { upsertConversationIds: [...upsertConversationIds] }
        : {}),
    }
    enqueuePersistence(() => chatHistoryPersistence.save(
      bucketUserId,
      snapshot,
      persistedMutation,
    ), context)
  }

  function schedulePersist(conversationId: string): void {
    if (!userId.value) return
    pendingPersistConversationIds.add(conversationId)
    if (!persistenceAvailable.value) return
    if (persistTimeoutId !== null) return
    persistTimeoutId = setTimeout(() => {
      persistTimeoutId = null
      persist()
    }, STREAM_PERSIST_THROTTLE_MS)
  }

  function removePersistedBucket(
    bucketUserId: string,
    context: PersistenceContext,
  ): void {
    enqueuePersistence(() => chatHistoryPersistence.remove(bucketUserId), context)
  }

  async function readPersistedState(
    bucketUserId: string,
    context: PersistenceContext,
  ): Promise<PersistedChatState | null> {
    if (!await retryFailedPersistenceOperations(context)) return null
    let parsed: unknown
    try {
      parsed = await chatHistoryPersistence.load(bucketUserId)
      setPersistenceAvailable(context, true)
    } catch (error) {
      recordPersistenceFailure(context, 'CHAT_PERSISTENCE_READ_FAILED', error)
      return null
    }
    if (parsed === null) return null

    try {
      if (
        !isRecord(parsed)
        || (
          parsed.version !== 1
          && parsed.version !== 2
          && parsed.version !== 3
          && parsed.version !== 4
          && parsed.version !== STORAGE_VERSION
        )
        || !Array.isArray(parsed.conversations)
      ) {
        throw new Error('Invalid chat storage payload')
      }

      const restoredClearRevision = typeof parsed.clearRevision === 'number'
        && Number.isFinite(parsed.clearRevision)
        && parsed.clearRevision >= 0
        ? parsed.clearRevision
        : 0
      const restoredDeletedIds = new Set(
        Array.isArray(parsed.deletedConversationIds)
          ? parsed.deletedConversationIds.filter(
            (id): id is string => typeof id === 'string' && id.length > 0,
          )
          : [],
      )

      const conversationIds = new Set<string>()
      const normalizationTime = Date.now()
      const restored = parsed.conversations.reduce<ChatConversation[]>((result, candidate) => {
        const conversation = sanitizeConversation(candidate, bucketUserId)
        if (
          conversation
          && !restoredDeletedIds.has(conversation.id)
          && !conversationIds.has(conversation.id)
        ) {
          conversationIds.add(conversation.id)
          result.push(conversation)
        }
        return result
      }, [])
        .sort((left, right) => compareConversationsByRecency(left, right, normalizationTime))
        .slice(0, MAX_CHAT_CONVERSATIONS)
        .map((conversation) => normalizeFutureTimestamps(conversation, normalizationTime))

      const requestedActiveId = typeof parsed.activeConversationId === 'string'
        ? parsed.activeConversationId
        : null
      const hasExplicitEmptySelection = Object.prototype.hasOwnProperty.call(
        parsed,
        'activeConversationId',
      ) && parsed.activeConversationId === null
      const restoredSelectionResolved =
        typeof parsed.activeConversationSelectionResolved === 'boolean'
          ? parsed.activeConversationSelectionResolved
          : requestedActiveId !== null
            || hasExplicitEmptySelection
            || restoredClearRevision > 0
      const outboxIds = new Set<string>()
      const restoredOutbox = (
        Array.isArray(parsed.outbox) ? parsed.outbox : []
      ).reduce<ChatHistoryOutboxMutation[]>((result, candidate) => {
        const mutation = sanitizeOutboxMutation(candidate)
        if (mutation && !outboxIds.has(mutation.mutationId)) {
          outboxIds.add(mutation.mutationId)
          result.push(mutation)
        }
        return result
      }, [])
      const restoredLegacyImportDecision: ChatLegacyImportDecision =
        parsed.legacyImportDecision === 'pending'
        || parsed.legacyImportDecision === 'accepted'
        || parsed.legacyImportDecision === 'declined'
          ? parsed.legacyImportDecision
          : (parsed.version === 1 && restored.length > 0 ? 'pending' : null)
      const restoredLegacyConversationIds = new Set(
        Array.isArray(parsed.legacyConversationIds)
          ? parsed.legacyConversationIds.filter(
            (id): id is string => typeof id === 'string' && conversationIds.has(id),
          )
          : (parsed.version === 1 ? restored.map(({ id }) => id) : []),
      )

      return {
        version: STORAGE_VERSION,
        clearRevision: restoredClearRevision,
        deletedConversationIds: [...restoredDeletedIds],
        activeConversationId: !restoredSelectionResolved
          ? null
          : parsed.activeConversationId === null
            ? null
            : restored.some(({ id }) => id === requestedActiveId)
              ? requestedActiveId
              : (restored[0]?.id ?? null),
        activeConversationSelectionResolved: restoredSelectionResolved,
        conversations: restored,
        serverVersion: isNonNegativeInteger(parsed.serverVersion) ? parsed.serverVersion : 0,
        outbox: restoredOutbox,
        legacyImportDecision: restoredLegacyImportDecision,
        legacyConversationIds: [...restoredLegacyConversationIds],
      }
    } catch (error) {
      recordPersistenceFailure(context, 'CHAT_PERSISTENCE_PAYLOAD_INVALID', error)
      removePersistedBucket(bucketUserId, context)
      return null
    }
  }

  async function hydrate(nextUserId: string | number | null | undefined): Promise<void> {
    const normalizedUserId = normalizeUserId(nextUserId)
    if ((hydrated.value || hydrating.value) && userId.value === normalizedUserId) {
      return hydrationPromise ?? Promise.resolve()
    }

    if (isStreaming.value) stopStreaming()
    resetState()
    hydrationVersion += 1
    const version = hydrationVersion
    const persistenceContext: PersistenceContext = {
      userId: normalizedUserId,
      hydrationVersion: version,
    }
    persistenceAvailable.value = true
    userId.value = normalizedUserId
    hydrating.value = true
    const selectionRevision = activeConversationSelectionRevision

    hydrationPromise = (async () => {
      try {
        if (!normalizedUserId) return
        const persisted = await readPersistedState(normalizedUserId, persistenceContext)
        if (version !== hydrationVersion || userId.value !== normalizedUserId || !persisted) return

        conversations.value = persisted.conversations
        if (selectionRevision === activeConversationSelectionRevision) {
          activeConversationId.value = persisted.activeConversationId
          activeConversationSelectionResolved = persisted.activeConversationSelectionResolved
        }
        clearRevision = persisted.clearRevision
        deletedConversationIds = new Set(persisted.deletedConversationIds)
        serverVersion.value = persisted.serverVersion
        outbox.value = persisted.outbox
        legacyImportDecision.value = persisted.legacyImportDecision
        legacyConversationIds.value = persisted.legacyConversationIds
        // Persist the sanitized snapshot, including interrupted stream normalization.
        persist({
          sanitizeConversations: persisted.conversations.map(({ id, updatedAt }) => ({
            id,
              expectedUpdatedAt: updatedAt,
            })),
          syncStateChanged: true,
        })
      } finally {
        if (version === hydrationVersion) {
          hydrating.value = false
          hydrated.value = true
        }
      }
    })()

    return hydrationPromise
  }

  function moveConversationToFront(conversation: ChatConversation, timestamp = epochNow()): void {
    const newestTimestamp = conversations.value.reduce(
      (newest, candidate) => Math.max(newest, candidate.updatedAt),
      0,
    )
    conversation.updatedAt = Math.max(timestamp, newestTimestamp + TIMESTAMP_INCREMENT_MS)
    const index = conversations.value.findIndex(({ id }) => id === conversation.id)
    if (index > 0) {
      conversations.value.splice(index, 1)
      conversations.value.unshift(conversation)
    }
  }

  function legacyConversationIsLocalOnly(conversationId: string): boolean {
    return legacyConversationIds.value.includes(conversationId)
      && legacyImportDecision.value !== 'accepted'
  }

  function queueConversationCreate(
    conversation: ChatConversation,
    legacyImport = false,
  ): string {
    const existing = outbox.value.find((mutation) => (
      mutation.conversationId === conversation.id && mutation.type === 'create'
    ))
    if (existing) {
      existing.title = conversation.title
      existing.model = conversation.model
      if (legacyImport) existing.legacyImport = true
      return existing.mutationId
    }
    const mutation: ChatHistoryOutboxMutation = {
      mutationId: `create:${conversation.id}`,
      type: 'create',
      conversationId: conversation.id,
      createdAt: conversation.createdAt,
      title: conversation.title,
      model: conversation.model,
      ...(legacyImport ? { legacyImport: true } : {}),
    }
    outbox.value.push(mutation)
    return mutation.mutationId
  }

  function queueConversationPatch(
    conversation: ChatConversation,
    patch: { title?: string; model?: string },
  ): string | null {
    if (legacyConversationIsLocalOnly(conversation.id)) return null
    const pendingCreate = outbox.value.find((mutation) => (
      mutation.conversationId === conversation.id && mutation.type === 'create'
    ))
    if (pendingCreate) {
      pendingCreate.createdAt = epochNow()
      if (patch.title !== undefined) pendingCreate.title = patch.title
      if (patch.model !== undefined) pendingCreate.model = patch.model
      return pendingCreate.mutationId
    }

    const existing = outbox.value.find((mutation) => (
      mutation.conversationId === conversation.id && mutation.type === 'patch'
    ))
    if (existing) {
      existing.createdAt = epochNow()
      existing.revision = conversation.serverRevision ?? existing.revision ?? 0
      if (patch.title !== undefined) existing.title = patch.title
      if (patch.model !== undefined) existing.model = patch.model
      return existing.mutationId
    }

    const mutation: ChatHistoryOutboxMutation = {
      mutationId: `patch:${conversation.id}`,
      type: 'patch',
      conversationId: conversation.id,
      createdAt: epochNow(),
      revision: conversation.serverRevision ?? 0,
      ...patch,
    }
    outbox.value.push(mutation)
    return mutation.mutationId
  }

  function queueConversationDelete(conversation: ChatConversation): string | null {
    if (legacyConversationIsLocalOnly(conversation.id)) return null
    const existing = outbox.value.find((mutation) => (
      mutation.conversationId === conversation.id && mutation.type === 'delete'
    ))
    if (existing) {
      existing.createdAt = epochNow()
      existing.revision = conversation.serverRevision ?? existing.revision ?? 0
      return existing.mutationId
    }
    const mutation: ChatHistoryOutboxMutation = {
      mutationId: `delete:${conversation.id}`,
      type: 'delete',
      conversationId: conversation.id,
      createdAt: epochNow(),
      revision: conversation.serverRevision ?? 0,
    }
    outbox.value.push(mutation)
    return mutation.mutationId
  }

  function acceptLegacyImport(): number {
    if (legacyConversationIds.value.length === 0) {
      legacyImportDecision.value = null
      persist({ syncStateChanged: true })
      return 0
    }
    legacyImportDecision.value = 'accepted'
    const mutationIds: string[] = []
    for (const id of legacyConversationIds.value.slice(0, MAX_CHAT_CONVERSATIONS)) {
      const conversation = findConversation(id)
      if (!conversation || conversation.serverRevision !== undefined) continue
      mutationIds.push(queueConversationCreate(conversation, true))
    }
    persist({
      ...(mutationIds.length > 0 ? { enqueueOutboxMutationIds: mutationIds } : {}),
      syncStateChanged: true,
    })
    return mutationIds.length
  }

  function declineLegacyImport(): void {
    legacyImportDecision.value = 'declined'
    const legacyMutationIds = outbox.value
      .filter((mutation) => mutation.legacyImport === true)
      .map(({ mutationId }) => mutationId)
    const acknowledged = acknowledgeOutbox(legacyMutationIds)
    persist({
      ...(acknowledged.length > 0
        ? { acknowledgedOutboxMutationIds: acknowledged }
        : {}),
      syncStateChanged: true,
    })
  }

  function createConversation(model: string, title = DEFAULT_CONVERSATION_TITLE): ChatConversation | null {
    if (!userId.value || !hydrated.value) return null
    const normalizedModel = model.trim()
    if (!normalizedModel) return null

    const operationAt = epochNow()
    const updatedAt = Math.max(
      operationAt,
      conversations.value.reduce(
        (newest, candidate) => Math.max(
          newest,
          candidate.updatedAt + TIMESTAMP_INCREMENT_MS,
        ),
        0,
      ),
    )
    const conversation: ChatConversation = {
      id: createId('conversation'),
      userId: userId.value,
      title: title.trim() || DEFAULT_CONVERSATION_TITLE,
      model: normalizedModel,
      messages: [],
      createdAt: operationAt,
      updatedAt,
    }

    conversations.value.unshift(conversation)
    activeConversationId.value = conversation.id
    activeConversationSelectionResolved = true
    activeConversationSelectionRevision += 1
    const createMutationId = queueConversationCreate(conversation)

    if (conversations.value.length > MAX_CHAT_CONVERSATIONS) {
      const removed = conversations.value.splice(MAX_CHAT_CONVERSATIONS)
      if (removed.some(({ id }) => id === streamingConversationId.value)) {
        clearStreamingRuntime(true)
      }
    }
    persist({
      upsertConversationIds: [conversation.id],
      createdConversations: [{ id: conversation.id, operationAt }],
      activeConversationChanged: true,
      enqueueOutboxMutationIds: [createMutationId],
    })
    return conversation
  }

  function selectConversation(conversationId: string | null): boolean {
    if (conversationId === null) {
      activeConversationId.value = null
      activeConversationSelectionResolved = true
      activeConversationSelectionRevision += 1
      persist({ activeConversationChanged: true })
      return true
    }
    if (!findConversation(conversationId)) return false

    activeConversationId.value = conversationId
    activeConversationSelectionResolved = true
    activeConversationSelectionRevision += 1
    persist({ activeConversationChanged: true })
    return true
  }

  function renameConversation(conversationId: string, title: string): boolean {
    const conversation = findConversation(conversationId)
    const normalizedTitle = title.trim()
    if (!conversation || !normalizedTitle) return false

    conversation.title = normalizedTitle
    moveConversationToFront(conversation)
    const mutationId = queueConversationPatch(conversation, { title: normalizedTitle })
    persist({
      upsertConversationIds: [conversationId],
      ...(mutationId ? { enqueueOutboxMutationIds: [mutationId] } : {}),
    })
    return true
  }

  function deleteConversation(conversationId: string): boolean {
    const index = conversations.value.findIndex((conversation) => conversation.id === conversationId)
    if (index < 0) return false

    invalidateHistorySearch()
    if (streamingConversationId.value === conversationId) clearStreamingRuntime(true)
    const mutationId = queueConversationDelete(conversations.value[index]!)
    conversations.value.splice(index, 1)
    searchResults.value = searchResults.value.filter(
      (conversation) => conversation.id !== conversationId,
    )
    deletedConversationIds.add(conversationId)
    legacyConversationIds.value = legacyConversationIds.value.filter((id) => id !== conversationId)
    const activeConversationChanged = activeConversationId.value === conversationId
    if (activeConversationChanged) {
      activeConversationId.value = conversations.value[index]?.id
        ?? conversations.value[index - 1]?.id
        ?? null
      activeConversationSelectionResolved = true
      activeConversationSelectionRevision += 1
    }
    persist({
      deletedConversationIds: [conversationId],
      activeConversationChanged,
      ...(mutationId ? { enqueueOutboxMutationIds: [mutationId] } : {}),
      syncStateChanged: true,
    })
    return true
  }

  function clearConversations(): void {
    invalidateHistorySearch()
    if (!userId.value) return
    const deleteMutationIds = conversations.value
      .map(queueConversationDelete)
      .filter((id): id is string => Boolean(id))
    clearStreamingRuntime(true)
    conversations.value = []
    activeConversationId.value = null
    activeConversationSelectionResolved = true
    activeConversationSelectionRevision += 1
    streamError.value = null
    clearRevision = Math.max(epochNow(), clearRevision + TIMESTAMP_INCREMENT_MS)
    deletedConversationIds = new Set<string>()
    legacyConversationIds.value = []
    persist({
      clear: true,
      activeConversationChanged: true,
      ...(deleteMutationIds.length > 0
        ? { enqueueOutboxMutationIds: deleteMutationIds }
        : {}),
      syncStateChanged: true,
    })
  }

  async function flushPersistence(): Promise<void> {
    const context = capturePersistenceContext()
    if (persistTimeoutId !== null) persist()
    const writesRecovered = await retryFailedPersistenceOperations(context)
    const bucketUserId = context.userId
    if (
      !isCurrentPersistenceContext(context)
      || !writesRecovered
      || persistenceAvailable.value
      || !bucketUserId
    ) return

    try {
      await chatHistoryPersistence.load(bucketUserId)
      setPersistenceAvailable(context, true)
    } catch (error) {
      recordPersistenceFailure(context, 'CHAT_PERSISTENCE_PROBE_FAILED', error)
    }
  }

  function setConversationModel(conversationId: string, model: string): boolean {
    const conversation = findConversation(conversationId)
    const normalizedModel = model.trim()
    if (!conversation || !normalizedModel) return false

    conversation.model = normalizedModel
    moveConversationToFront(conversation)
    const mutationId = queueConversationPatch(conversation, { model: normalizedModel })
    persist({
      upsertConversationIds: [conversationId],
      ...(mutationId ? { enqueueOutboxMutationIds: [mutationId] } : {}),
    })
    return true
  }

  function addMessage(conversationId: string, input: CreateChatMessage): ChatMessage | null {
    const conversation = findConversation(conversationId)
    if (!conversation || (input.role !== 'user' && input.role !== 'assistant')) return null
    if (typeof input.content !== 'string') return null

    const highestKnownPosition = conversation.messages.reduce(
      (highest, { position }) => Math.max(highest, position ?? 0),
      0,
    )
    const nextPosition = Math.max(
      conversation.messageCount ?? 0,
      conversation.messages.length,
      highestKnownPosition,
    ) + 1

    const message: ChatMessage = {
      id: input.id?.trim() || createId('message'),
      role: input.role,
      content: input.content,
      createdAt: isFiniteTimestamp(input.createdAt) ? input.createdAt : Date.now(),
      position: nextPosition,
      status: input.status ?? 'complete',
    }
    if (conversation.messages.some(({ id }) => id === message.id)) return null
    if (input.finishReason !== undefined) message.finishReason = input.finishReason
    if (input.errorCode !== undefined) message.errorCode = input.errorCode
    if (input.errorMessage !== undefined) message.errorMessage = input.errorMessage
    copyAssistantMetadata(message, input as unknown as Record<string, unknown>)
    const attachments = sanitizedAttachments(input.attachments)
    if (attachments.length > 0) message.attachments = attachments
    if (message.role === 'assistant') {
      const activities = sanitizedActivities(input.activities, message.createdAt)
      if (activities.length > 0) message.activities = activities
    }

    conversation.messages.push(message)
    moveConversationToFront(conversation)
    persist({ upsertConversationIds: [conversationId] })
    return message
  }

  function updateMessage(
    conversationId: string,
    messageId: string,
    patch: ChatMessagePatch,
  ): boolean {
    const conversation = findConversation(conversationId)
    const message = conversation?.messages.find((candidate) => candidate.id === messageId)
    if (!conversation || !message) return false

    if (typeof patch.content === 'string') message.content = patch.content
    if (
      patch.status === 'complete'
      || patch.status === 'streaming'
      || patch.status === 'stopped'
      || patch.status === 'error'
    ) {
      message.status = patch.status
    }
    if (typeof patch.finishReason === 'string' || patch.finishReason === null) {
      message.finishReason = patch.finishReason
    }
    if (typeof patch.errorCode === 'string') message.errorCode = patch.errorCode
    if (typeof patch.errorMessage === 'string') message.errorMessage = patch.errorMessage
    if (patch.pendingStopRequestedAt === null) delete message.pendingStopRequestedAt
    else if (isFiniteTimestamp(patch.pendingStopRequestedAt)) {
      message.pendingStopRequestedAt = patch.pendingStopRequestedAt
    }
    copyAssistantMetadata(message, patch as unknown as Record<string, unknown>)
    if (message.role === 'assistant' && patch.activities !== undefined) {
      const activities = sanitizedActivities(patch.activities, message.createdAt)
      if (activities.length > 0) message.activities = activities
      else delete message.activities
    }

    // Message patches are also used for asynchronous receipt/billing
    // reconciliation. Those callbacks can arrive for any old conversation
    // and must not change the conversation's recency (or reorder the history
    // sidebar based on network completion order). Real conversation activity
    // already updates recency through add/start/finish/stop and explicit
    // conversation mutations.
    persist({ upsertConversationIds: [conversationId] })
    return true
  }

  function removeMessages(conversationId: string, messageIds: string[]): boolean {
    const conversation = findConversation(conversationId)
    const ids = new Set(messageIds.map((id) => id.trim()).filter(Boolean))
    if (!conversation || ids.size === 0) return false

    const nextMessages = conversation.messages.filter((message) => !ids.has(message.id))
    if (nextMessages.length === conversation.messages.length) return false
    if (
      streamingConversationId.value === conversationId
      && streamingMessageId.value
      && ids.has(streamingMessageId.value)
    ) {
      clearStreamingRuntime(true)
    }
    conversation.messages = nextMessages
    // This is currently used to remove optimistic messages after a failed or
    // cancelled request. A late cleanup must not make an old conversation
    // jump to the top of the history sidebar.
    persist({ upsertConversationIds: [conversationId] })
    return true
  }

  // Activity and stream deltas are high-frequency progress updates. The turn
  // lifecycle touches conversation recency once; individual deltas must not
  // reorder the history sidebar.
  function upsertMessageActivity(
    conversationId: string,
    messageId: string,
    activity: ChatActivity,
  ): boolean {
    const conversation = findConversation(conversationId)
    const message = findMessage(conversationId, messageId)
    if (!conversation || !message || message.role !== 'assistant') return false
    const normalized = sanitizedActivities([activity], message.createdAt)
    if (normalized.length !== 1) return false
    if (!isTerminalChatActivityStatus(normalized[0]!.status) && message.status !== 'streaming') {
      if (message.status === 'stopped') {
        applyActivityTerminal(normalized[0]!, 'stopped')
      } else if (message.status === 'error') {
        applyActivityTerminal(normalized[0]!, 'failed', {
          ...(message.errorCode ? { code: message.errorCode } : {}),
          ...(message.errorMessage ? { message: message.errorMessage } : {}),
        })
      } else {
        applyActivityTerminal(normalized[0]!, 'completed')
      }
    }
    message.activities = mergeChatActivities(message.activities, normalized)
    if (isTerminalChatActivityStatus(normalized[0]!.status)) {
      persist({ upsertConversationIds: [conversationId] })
    } else {
      schedulePersist(conversationId)
    }
    return true
  }

  function activityPart(
    message: ChatMessage,
    activityKey: string,
    partKey: string,
  ): {
    activity: ChatActivity
    item: ChatActivity['items'][number]
    part: ChatActivity['items'][number]['parts'][number]
  } | null {
    const activity = message.activities?.find((candidate) => candidate.key === activityKey)
    if (!activity) return null
    for (const item of activity.items) {
      const part = item.parts.find((candidate) => candidate.key === partKey)
      if (part) return { activity, item, part }
    }
    return null
  }

  function appendMessageActivityDelta(
    conversationId: string,
    messageId: string,
    activityKey: string,
    partKey: string,
    delta: string,
  ): boolean {
    if (!delta) return false
    const conversation = findConversation(conversationId)
    const message = findMessage(conversationId, messageId)
    if (!conversation || !message || message.role !== 'assistant') return false
    const target = activityPart(message, activityKey, partKey)
    if (!target || isTerminalChatActivityStatus(target.activity.status)) return false
    const now = epochNow()
    if (!target.part.streamingTextChunks) target.part.streamingTextChunks = []
    target.part.streamingTextChunks.push(delta)
    target.part.status = 'streaming'
    target.part.updatedAt = now
    target.item.status = 'streaming'
    target.item.updatedAt = now
    target.activity.status = 'streaming'
    target.activity.updatedAt = now
    schedulePersist(conversationId)
    return true
  }

  function replaceMessageActivityText(
    conversationId: string,
    messageId: string,
    activityKey: string,
    partKey: string,
    text: string,
  ): boolean {
    const conversation = findConversation(conversationId)
    const message = findMessage(conversationId, messageId)
    if (!conversation || !message || message.role !== 'assistant') return false
    const target = activityPart(message, activityKey, partKey)
    if (!target || isTerminalChatActivityStatus(target.activity.status)) return false
    const now = epochNow()
    target.part.text = text
    delete target.part.streamingTextChunks
    target.part.status = 'completed'
    target.part.updatedAt = now
    target.part.completedAt = now
    target.item.status = 'streaming'
    target.item.updatedAt = now
    target.activity.status = 'streaming'
    target.activity.updatedAt = now
    schedulePersist(conversationId)
    return true
  }

  function applyActivityTerminal(
    activity: ChatActivity,
    status: Exclude<ChatActivityStatus, 'pending' | 'streaming'>,
    error?: ChatActivityError,
  ): void {
    const now = epochNow()
    activity.status = status
    activity.updatedAt = now
    activity.completedAt = now
    if (error && (error.code || error.message)) activity.error = { ...error }
    else if (status !== 'failed') delete activity.error
    for (const item of activity.items) {
      if (!isTerminalChatActivityStatus(item.status)) item.status = status
      item.updatedAt = Math.max(item.updatedAt, now)
      item.completedAt = item.completedAt ?? now
      for (const part of item.parts) {
        finalizeChatActivityPartText(part)
        if (!isTerminalChatActivityStatus(part.status)) part.status = status
        part.updatedAt = Math.max(part.updatedAt, now)
        part.completedAt = part.completedAt ?? now
      }
    }
  }

  function setMessageActivityTerminal(
    conversationId: string,
    messageId: string,
    activityKey: string,
    status: Exclude<ChatActivityStatus, 'pending' | 'streaming'>,
    error?: ChatActivityError,
  ): boolean {
    const conversation = findConversation(conversationId)
    const message = findMessage(conversationId, messageId)
    const activity = message?.activities?.find((candidate) => candidate.key === activityKey)
    if (!conversation || !message || message.role !== 'assistant' || !activity) return false
    applyActivityTerminal(activity, status, error)
    persist({ upsertConversationIds: [conversationId] })
    return true
  }

  function applyOpenMessageActivities(
    message: ChatMessage,
    status: Exclude<ChatActivityStatus, 'pending' | 'streaming'>,
    error?: ChatActivityError,
  ): boolean {
    const openActivities = message.activities?.filter(
      (activity) => !isTerminalChatActivityStatus(activity.status),
    ) ?? []
    for (const activity of openActivities) applyActivityTerminal(activity, status, error)
    return openActivities.length > 0
  }

  function stopMessageActivities(
    conversationId: string,
    messageId: string,
    status: Exclude<ChatActivityStatus, 'pending' | 'streaming'> = 'stopped',
    error?: ChatActivityError,
  ): boolean {
    const conversation = findConversation(conversationId)
    const message = findMessage(conversationId, messageId)
    if (!conversation || !message || message.role !== 'assistant') return false
    if (!applyOpenMessageActivities(message, status, error)) return false
    persist({ upsertConversationIds: [conversationId] })
    return true
  }

  function startStreaming(
    conversationId: string,
    messageId: string,
    controller: AbortController | null = null,
  ): boolean {
    const conversation = findConversation(conversationId)
    const message = findMessage(conversationId, messageId)
    if (!conversation || !message || message.role !== 'assistant') return false

    if (isStreaming.value) stopStreaming()
    message.status = 'streaming'
    delete message.finishReason
    delete message.errorCode
    delete message.errorMessage
    streamingConversationId.value = conversationId
    streamingMessageId.value = messageId
    streamController.value = controller
    streamError.value = null
    moveConversationToFront(conversation)
    persist({ upsertConversationIds: [conversationId] })
    return true
  }

  function isCurrentStream(conversationId: string, messageId: string): boolean {
    return streamingConversationId.value === conversationId && streamingMessageId.value === messageId
  }

  function appendStreamingContent(conversationId: string, messageId: string, chunk: string): boolean {
    if (!chunk || !isCurrentStream(conversationId, messageId)) return false
    const conversation = findConversation(conversationId)
    const message = findMessage(conversationId, messageId)
    if (!conversation || !message || message.status !== 'streaming') return false

    message.content += chunk
    schedulePersist(conversationId)
    return true
  }

  function finishStreaming(
    conversationId: string,
    messageId: string,
    finishReason: string | null = null,
  ): boolean {
    if (!isCurrentStream(conversationId, messageId)) return false
    const conversation = findConversation(conversationId)
    const message = findMessage(conversationId, messageId)
    if (!conversation || !message) {
      clearStreamingRuntime(false)
      return false
    }

    message.status = 'complete'
    message.finishReason = finishReason
    applyOpenMessageActivities(message, 'completed')
    delete message.errorCode
    delete message.errorMessage
    clearStreamingRuntime(false)
    streamError.value = null
    moveConversationToFront(conversation)
    persist({ upsertConversationIds: [conversationId] })
    return true
  }

  function failStreaming(
    conversationId: string,
    messageId: string,
    errorMessage: string,
    errorCode?: string,
  ): boolean {
    if (!isCurrentStream(conversationId, messageId)) return false
    const conversation = findConversation(conversationId)
    const message = findMessage(conversationId, messageId)
    if (!conversation || !message) {
      clearStreamingRuntime(false)
      return false
    }

    message.status = 'error'
    message.errorMessage = errorMessage
    if (errorCode) message.errorCode = errorCode
    else delete message.errorCode
    applyOpenMessageActivities(message, 'failed', {
      ...(errorCode ? { code: errorCode } : {}),
      message: errorMessage,
    })
    streamError.value = { message: errorMessage, ...(errorCode ? { code: errorCode } : {}) }
    clearStreamingRuntime(false)
    moveConversationToFront(conversation)
    persist({ upsertConversationIds: [conversationId] })
    return true
  }

  function stopStreaming(): boolean {
    if (!streamingConversationId.value || !streamingMessageId.value) return false
    const conversation = findConversation(streamingConversationId.value)
    const message = findMessage(streamingConversationId.value, streamingMessageId.value)

    clearStreamingRuntime(true)
    streamError.value = null
    if (!conversation || !message) return false

    message.status = 'stopped'
    message.finishReason = 'stopped'
    applyOpenMessageActivities(message, 'stopped')
    moveConversationToFront(conversation)
    persist({ upsertConversationIds: [conversation.id] })
    return true
  }

  function browserIsOnline(): boolean {
    try {
      return typeof navigator === 'undefined' || navigator.onLine !== false
    } catch {
      return true
    }
  }

  function chatErrorStatus(error: unknown): number {
    if (error instanceof ChatAPIError) return error.status
    if (isRecord(error) && typeof error.status === 'number') return error.status
    return 0
  }

  function isHistoryUnavailableError(error: unknown): boolean {
    const status = chatErrorStatus(error)
    return status === 405 || status === 501
      || (
        status === 404
        && serverHistoryAvailable.value !== true
      )
  }

  function isConflictError(error: unknown): boolean {
    return chatErrorStatus(error) === 409
  }

  function isNotFoundError(error: unknown): boolean {
    return chatErrorStatus(error) === 404
  }

  function recordHistorySyncFailure(error: unknown, fallbackMessage: string): void {
    if (isHistoryUnavailableError(error)) {
      serverHistoryAvailable.value = false
      syncStatus.value = 'unavailable'
      syncError.value = null
      return
    }
    syncStatus.value = browserIsOnline() ? 'error' : 'offline'
    syncError.value = error instanceof Error ? error.message : fallbackMessage
  }

  function sortMessages(messages: ChatMessage[]): ChatMessage[] {
    const hasCanonicalPositions = messages.every(({ position }) => position !== undefined)
    return messages.sort((left, right) => {
      if (hasCanonicalPositions) {
        return (left.position ?? 0) - (right.position ?? 0)
          || left.createdAt - right.createdAt
      }
      // Legacy and in-flight records can temporarily lack a server position.
      // Keep equal timestamps stable instead of letting random UUIDs reorder a turn.
      return left.createdAt - right.createdAt
    })
  }

  function conversationNeedsCanonicalMessages(conversation: ChatConversation): boolean {
    if (conversation.messages.some(({ position }) => position === undefined)) return true
    if ((conversation.messageCount ?? 0) > 0 && conversation.messages.length === 0) return true
    return Boolean(
      conversation.headMessageId
      && !conversation.messages.some(({ id }) => id === conversation.headMessageId),
    )
  }

  function mergeServerMessage(
    existing: ChatMessage | undefined,
    incoming: ChatServerMessage,
    conversationId: string,
  ): ChatMessage {
    const protectsCurrentStream = Boolean(
      existing
      && isCurrentStream(conversationId, existing.id)
      && existing.status === 'streaming',
    )
    // The API intentionally maps a server-side processing checkpoint to a
    // stopped/disconnected snapshot for cold history recovery. While this tab
    // still owns the live stream, that synthetic terminal must not outrank a
    // newer local Activity. A genuinely completed/failed server response keeps
    // normal server authority.
    const incomingIsProcessingCheckpoint = protectsCurrentStream
      && incoming.status === 'stopped'
      && incoming.finishReason === 'interrupted'
    const merged: ChatMessage = existing
      ? { ...existing, ...incoming }
      : { ...incoming }
    if (incoming.activities?.length) {
      merged.activities = mergeChatActivities(existing?.activities, incoming.activities, {
        incomingIsServer: !incomingIsProcessingCheckpoint,
      })
    } else if (existing?.activities?.length) {
      merged.activities = existing.activities.map(cloneChatActivity)
    }
    if (!protectsCurrentStream && incoming.status === 'complete') {
      delete merged.errorCode
      delete merged.errorMessage
    }
    if (protectsCurrentStream && existing) {
      merged.status = 'streaming'
      if (existing.content.length > incoming.content.length) merged.content = existing.content
      delete merged.finishReason
    }
    return merged
  }

  function pendingConversationMutation(
    conversationId: string,
  ): ChatHistoryOutboxMutation | undefined {
    return [...outbox.value]
      .reverse()
      .find((mutation) => (
        mutation.conversationId === conversationId
        && (mutation.type === 'create' || mutation.type === 'patch')
      ))
  }

  function applyServerConversation(incoming: ChatServerConversation): string | null {
    if (!userId.value || deletedConversationIds.has(incoming.id)) return null
    const existing = findConversation(incoming.id)
    const messagesById = new Map(
      (existing?.messages ?? []).map((message) => [message.id, message]),
    )
    for (const message of incoming.messages) {
      messagesById.set(
        message.id,
        mergeServerMessage(messagesById.get(message.id), message, incoming.id),
      )
    }
    const messages = sortMessages([...messagesById.values()])
    const remoteWinsMetadata = !existing
      || incoming.revision >= (existing.serverRevision ?? 0)
    const conversation: ChatConversation = existing ?? {
      id: incoming.id,
      userId: userId.value,
      title: incoming.title,
      model: incoming.model,
      messages: [],
      createdAt: incoming.createdAt,
      updatedAt: incoming.updatedAt,
    }

    if (remoteWinsMetadata) {
      conversation.title = incoming.title
      conversation.model = incoming.model
      conversation.createdAt = incoming.createdAt
      conversation.updatedAt = incoming.updatedAt
      conversation.serverRevision = incoming.revision
      conversation.serverVersion = incoming.version
      conversation.headMessageId = incoming.headMessageId
      conversation.messageCount = incoming.messageCount
    }
    conversation.messages = messages

    const pending = pendingConversationMutation(incoming.id)
    if (pending) {
      if (pending.title) conversation.title = pending.title
      if (pending.model) conversation.model = pending.model
      conversation.updatedAt = Math.max(conversation.updatedAt, pending.createdAt)
    }

    if (!existing) conversations.value.push(conversation)
    deletedConversationIds.delete(incoming.id)
    conversations.value.sort((left, right) => (
      compareConversationsByRecency(left, right, Date.now())
    ))
    return conversation.id
  }

  function resolveInitialConversationSelection(): void {
    if (activeConversationSelectionResolved || conversations.value.length === 0) return
    activeConversationId.value = conversations.value[0]!.id
    activeConversationSelectionResolved = true
    activeConversationSelectionRevision += 1
    pendingInitialConversationSelection = true
  }

  function acknowledgeOutbox(ids: Iterable<string>): string[] {
    const acknowledged = new Set(ids)
    if (acknowledged.size === 0) return []
    const existingIds = new Set(outbox.value.map(({ mutationId }) => mutationId))
    outbox.value = outbox.value.filter(({ mutationId }) => !acknowledged.has(mutationId))
    return [...acknowledged].filter((id) => existingIds.has(id))
  }

  function applyServerDeletion(conversationId: string): {
    deleted: boolean
    acknowledgedMutationIds: string[]
  } {
    const index = conversations.value.findIndex(({ id }) => id === conversationId)
    if (index >= 0) {
      if (streamingConversationId.value === conversationId) clearStreamingRuntime(true)
      conversations.value.splice(index, 1)
      if (activeConversationId.value === conversationId) {
        if (pendingServerSelectionReplacementId === null) {
          pendingServerSelectionReplacementId = conversationId
        }
        activeConversationId.value = conversations.value[index]?.id
          ?? conversations.value[index - 1]?.id
          ?? null
        activeConversationSelectionResolved = true
        activeConversationSelectionRevision += 1
      }
    }
    deletedConversationIds.add(conversationId)
    const acknowledgedMutationIds = acknowledgeOutbox(
      outbox.value
        .filter((mutation) => mutation.conversationId === conversationId)
        .map(({ mutationId }) => mutationId),
    )
    return { deleted: index >= 0, acknowledgedMutationIds }
  }

  function persistServerMerge(
    upsertConversationIds: string[],
    deletedIds: string[] = [],
    acknowledgedMutationIds: string[] = [],
  ): void {
    const createdConversations = upsertConversationIds.flatMap((id) => {
      const conversation = findConversation(id)
      return conversation
        ? [{ id, operationAt: conversation.createdAt }]
        : []
    })
    const initializeActiveConversation = pendingInitialConversationSelection
    const replaceActiveConversationIfId = pendingServerSelectionReplacementId
    pendingInitialConversationSelection = false
    pendingServerSelectionReplacementId = null
    persist({
      ...(upsertConversationIds.length > 0 ? { upsertConversationIds } : {}),
      ...(createdConversations.length > 0 ? { createdConversations } : {}),
      ...(deletedIds.length > 0 ? { deletedConversationIds: deletedIds } : {}),
      ...(acknowledgedMutationIds.length > 0
        ? { acknowledgedOutboxMutationIds: acknowledgedMutationIds }
        : {}),
      ...(initializeActiveConversation ? { initializeActiveConversation: true } : {}),
      ...(replaceActiveConversationIfId
        ? { replaceActiveConversationIfId }
        : {}),
      syncStateChanged: true,
    })
  }

  async function pullServerChanges(signal?: AbortSignal): Promise<void> {
    let cursor: string | null = null
    let pageCount = 0
    const upserted = new Set<string>()
    const deleted = new Set<string>()
    const acknowledged = new Set<string>()
    do {
      const page = await getChatSync({
        afterVersion: serverVersion.value,
        cursor,
        signal,
      })
      serverHistoryAvailable.value = true
      for (const change of page.changes) {
        if (change.type === 'delete') {
          const result = applyServerDeletion(change.conversationId)
          if (result.deleted) deleted.add(change.conversationId)
          result.acknowledgedMutationIds.forEach((id) => acknowledged.add(id))
        } else if (change.conversation) {
          const id = applyServerConversation(change.conversation)
          if (id) upserted.add(id)
        }
        serverVersion.value = Math.max(serverVersion.value, change.version)
      }
      serverVersion.value = Math.max(serverVersion.value, page.latestVersion)
      cursor = page.hasMore ? page.nextCursor : null
      pageCount += 1
    } while (cursor && pageCount < 100)

    resolveInitialConversationSelection()
    persistServerMerge([...upserted], [...deleted], [...acknowledged])
  }

  async function fetchConflictConversation(
    conversationId: string,
    signal?: AbortSignal,
    context?: PersistenceContext,
  ): Promise<ChatServerConversation | null> {
    try {
      const remote = await getChatConversation(conversationId, signal)
      if (signal?.aborted || (context && !isCurrentPersistenceContext(context))) return null
      const id = applyServerConversation(remote)
      if (id) persistServerMerge([id])
      return remote
    } catch (error) {
      if (signal?.aborted || (context && !isCurrentPersistenceContext(context))) return null
      if (isNotFoundError(error) && serverHistoryAvailable.value === true) {
        const deletion = applyServerDeletion(conversationId)
        persistServerMerge(
          [],
          deletion.deleted ? [conversationId] : [],
          deletion.acknowledgedMutationIds,
        )
        return null
      }
      throw error
    }
  }

  async function replayOutboxMutation(
    mutation: ChatHistoryOutboxMutation,
    signal?: AbortSignal,
    retryAfterConflict = true,
    context?: PersistenceContext,
  ): Promise<void> {
    let remote: ChatServerConversation | null = null
    try {
      if (mutation.type === 'create') {
        const conversation = findConversation(mutation.conversationId)
        if (!conversation) {
          const acknowledged = acknowledgeOutbox([mutation.mutationId])
          persistServerMerge([], [], acknowledged)
          return
        }
        remote = await createRemoteConversation({
          id: mutation.conversationId,
          title: mutation.title ?? conversation.title,
          model: mutation.model ?? conversation.model,
          ...(mutation.legacyImport
            ? {
                importedMessages: buildLegacyImportedMessages(conversation.messages),
              }
            : {}),
        }, signal)
      } else if (mutation.type === 'patch') {
        remote = await patchRemoteConversation(mutation.conversationId, {
          revision: mutation.revision ?? 0,
          ...(mutation.title !== undefined ? { title: mutation.title } : {}),
          ...(mutation.model !== undefined ? { model: mutation.model } : {}),
        }, signal)
      } else {
        await deleteRemoteConversation(mutation.conversationId, {
          revision: mutation.revision ?? 0,
        }, signal)
      }
      if (signal?.aborted || (context && !isCurrentPersistenceContext(context))) return
    } catch (error) {
      if (mutation.type === 'create' && isConflictError(error)) {
        remote = await getChatConversation(mutation.conversationId, signal)
        if (signal?.aborted || (context && !isCurrentPersistenceContext(context))) return
      } else if (
        retryAfterConflict
        && isConflictError(error)
        && mutation.type !== 'create'
      ) {
        const canonical = await fetchConflictConversation(mutation.conversationId, signal, context)
        if (!canonical) return
        mutation.revision = canonical.revision
        persist({
          enqueueOutboxMutationIds: [mutation.mutationId],
          syncStateChanged: true,
        })
        await replayOutboxMutation(mutation, signal, false, context)
        return
      } else if (
        mutation.type === 'delete'
        && isNotFoundError(error)
        && serverHistoryAvailable.value === true
      ) {
        // A repeated delete is already converged.
      } else {
        throw error
      }
    }

    if (signal?.aborted || (context && !isCurrentPersistenceContext(context))) return
    const acknowledged = acknowledgeOutbox([mutation.mutationId])
    const upserted: string[] = []
    if (remote) {
      const id = applyServerConversation(remote)
      if (id) upserted.push(id)
    }
    persistServerMerge(upserted, [], acknowledged)
  }

  async function replayOutbox(
    signal?: AbortSignal,
    conversationId?: string,
    context?: PersistenceContext,
  ): Promise<void> {
    const snapshot = outbox.value
      .filter((mutation) => !conversationId || mutation.conversationId === conversationId)
      .sort((left, right) => left.createdAt - right.createdAt)
    for (const candidate of snapshot) {
      if (signal?.aborted || (context && !isCurrentPersistenceContext(context))) return
      const current = outbox.value.find(({ mutationId }) => (
        mutationId === candidate.mutationId
      ))
      if (current) await replayOutboxMutation(current, signal, true, context)
    }
  }

  async function syncHistory(): Promise<void> {
    if (!userId.value || !hydrated.value) return
    if (!browserIsOnline()) {
      syncStatus.value = 'offline'
      syncError.value = null
      return
    }
    if (syncPromise) return syncPromise

    const expectedUserId = userId.value
    const controller = new AbortController()
    syncController?.abort()
    syncController = controller
    syncStatus.value = 'syncing'
    syncError.value = null
    syncPromise = (async () => {
      try {
        await pullServerChanges(controller.signal)
        if (expectedUserId !== userId.value) return
        await replayOutbox(controller.signal)
        if (expectedUserId !== userId.value) return
        await pullServerChanges(controller.signal)
        if (expectedUserId !== userId.value) return
        syncStatus.value = 'idle'
        serverHistoryAvailable.value = true
      } catch (error) {
        if (controller.signal.aborted || isAbortError(error)) return
        recordHistorySyncFailure(error, 'Unable to synchronize history.')
      } finally {
        if (syncController === controller) syncController = null
        syncPromise = null
      }
    })()
    return syncPromise
  }

  async function prepareConversationForCompletion(conversationId: string): Promise<boolean> {
    if (!browserIsOnline()) {
      syncStatus.value = 'offline'
      return false
    }
    const context = capturePersistenceContext()
    if (!isCurrentPersistenceContext(context) || !findConversation(conversationId)) return false
    completionPreparationController?.abort()
    const controller = new AbortController()
    completionPreparationController = controller
    try {
      await replayOutbox(controller.signal, conversationId, context)
      if (!isCurrentPersistenceContext(context) || !findConversation(conversationId)) return false
      const remote = await getChatConversation(conversationId, controller.signal)
      if (!isCurrentPersistenceContext(context) || !findConversation(conversationId)) return false
      const id = applyServerConversation(remote)
      if (id) persistServerMerge([id])
      return Boolean(id)
    } catch (error) {
      if (
        !controller.signal.aborted
        && !isAbortError(error)
        && isCurrentPersistenceContext(context)
      ) {
        recordHistorySyncFailure(error, 'Unable to prepare conversation.')
      }
      // Cloud history is best-effort. As long as this is still the same local
      // conversation and the browser is online, let /chat/completions make the
      // independent decision instead of treating sync failure as a hard gate.
      return !controller.signal.aborted
        && browserIsOnline()
        && isCurrentPersistenceContext(context)
        && Boolean(findConversation(conversationId))
    } finally {
      if (completionPreparationController === controller) {
        completionPreparationController = null
      }
    }
  }

  function markCompletionAccepted(
    conversationId: string,
    assistantMessageId: string,
  ): boolean {
    const conversation = findConversation(conversationId)
    const assistant = findMessage(conversationId, assistantMessageId)
    if (!conversation || !assistant || assistant.role !== 'assistant') return false
    conversation.headMessageId = assistantMessageId
    conversation.messageCount = Math.max(
      conversation.messageCount ?? 0,
      assistant.position ?? conversation.messages.length,
    )
    persist({ upsertConversationIds: [conversationId] })
    return true
  }

  async function loadConversationPage(reset = false): Promise<void> {
    if (!userId.value || loadingConversationPage.value || !browserIsOnline()) return
    const context = capturePersistenceContext()
    let conversationToHydrate: string | null = null
    loadingConversationPage.value = true
    try {
      const page = await listChatConversations({
        cursor: reset ? null : conversationCursor.value,
        limit: MAX_CHAT_CONVERSATIONS,
      })
      if (!isCurrentPersistenceContext(context)) return
      serverHistoryAvailable.value = true
      const upserted = page.items
        .map(applyServerConversation)
        .filter((id): id is string => Boolean(id))
      conversationCursor.value = page.nextCursor
      conversationsHaveMore.value = page.hasMore
      if (reset) resolveInitialConversationSelection()
      persistServerMerge(upserted)
      const active = reset && activeConversationId.value
        ? findConversation(activeConversationId.value)
        : undefined
      if (active && conversationNeedsCanonicalMessages(active)) {
        conversationToHydrate = active.id
      }
    } catch (error) {
      if (!isCurrentPersistenceContext(context)) return
      if (!isAbortError(error)) recordHistorySyncFailure(error, 'Unable to load conversations.')
    } finally {
      if (isCurrentPersistenceContext(context)) loadingConversationPage.value = false
    }
    if (conversationToHydrate && isCurrentPersistenceContext(context)) {
      await loadConversationDetail(conversationToHydrate)
    }
  }

  function localConversationSearch(query: string): ChatConversation[] {
    const normalized = query.trim().toLocaleLowerCase()
    if (!normalized) return []
    return conversations.value.filter((conversation) => (
      conversation.title.toLocaleLowerCase().includes(normalized)
      || conversation.messages.some((message) => (
        message.content.toLocaleLowerCase().includes(normalized)
      ))
    ))
  }

  async function searchHistory(query: string): Promise<void> {
    const requestSequence = ++searchRequestSequence
    const context = capturePersistenceContext()
    const normalized = query.trim()
    if (!normalized) {
      if (requestSequence === searchRequestSequence && isCurrentPersistenceContext(context)) {
        searchResults.value = []
        searchingHistory.value = false
      }
      return
    }
    if (!browserIsOnline() || serverHistoryAvailable.value === false) {
      if (requestSequence === searchRequestSequence && isCurrentPersistenceContext(context)) {
        searchResults.value = localConversationSearch(normalized)
        searchingHistory.value = false
      }
      return
    }
    searchingHistory.value = true
    try {
      const page = await searchChatConversations({ query: normalized, limit: 50 })
      if (requestSequence !== searchRequestSequence || !isCurrentPersistenceContext(context)) return
      serverHistoryAvailable.value = true
      const resultIds: string[] = []
      const upserted: string[] = []
      for (const remote of page.items) {
        if (requestSequence !== searchRequestSequence) return
        const id = applyServerConversation(remote)
        if (id) {
          upserted.push(id)
          resultIds.push(id)
        }
      }
      persistServerMerge(upserted)
      searchResults.value = resultIds
        .map((id) => findConversation(id))
        .filter((conversation): conversation is ChatConversation => Boolean(conversation))
    } catch (error) {
      if (requestSequence !== searchRequestSequence || !isCurrentPersistenceContext(context)) return
      searchResults.value = localConversationSearch(normalized)
      if (isHistoryUnavailableError(error)) serverHistoryAvailable.value = false
    } finally {
      if (requestSequence === searchRequestSequence && isCurrentPersistenceContext(context)) {
        searchingHistory.value = false
      }
    }
  }

  // Invalidate an in-flight search as soon as its input changes. The view
  // debounces the next request, so waiting for searchHistory() itself would
  // leave a window where an old response could still reorder the sidebar.
  function invalidateHistorySearch(): void {
    searchRequestSequence += 1
    searchingHistory.value = false
  }

  async function loadConversationDetail(conversationId: string): Promise<boolean> {
    if (!browserIsOnline() || serverHistoryAvailable.value === false) return false
    const context = capturePersistenceContext()
    try {
      const remote = await getChatConversation(conversationId)
      if (!isCurrentPersistenceContext(context)) return false
      const id = applyServerConversation(remote)
      if (id) persistServerMerge([id])
      const conversation = id ? findConversation(id) : undefined
      if (id && conversation && conversationNeedsCanonicalMessages(conversation)) {
        await loadConversationMessagesPage(id, null)
      }
      return Boolean(id)
    } catch (error) {
      if (!isCurrentPersistenceContext(context)) return false
      if (isNotFoundError(error) && serverHistoryAvailable.value === true) {
        const deletion = applyServerDeletion(conversationId)
        persistServerMerge(
          [],
          deletion.deleted ? [conversationId] : [],
          deletion.acknowledgedMutationIds,
        )
      }
      return false
    }
  }

  async function loadConversationMessagesPage(
    conversationId: string,
    beforePosition: number | null,
  ): Promise<boolean> {
    const context = capturePersistenceContext()
    const existingConversation = findConversation(conversationId)
    if (
      !existingConversation
      || !browserIsOnline()
      || loadingConversationMessages.value.has(conversationId)
    ) {
      return false
    }
    const loading = new Set(loadingConversationMessages.value)
    loading.add(conversationId)
    loadingConversationMessages.value = loading
    try {
      const page = await getChatConversationMessages(conversationId, {
        beforePosition,
        limit: 100,
      })
      if (!isCurrentPersistenceContext(context)) return false
      const conversation = findConversation(conversationId)
      if (!conversation) return false
      const messagesById = new Map(
        conversation.messages.map((message) => [message.id, message]),
      )
      for (const message of page.items) {
        messagesById.set(
          message.id,
          mergeServerMessage(messagesById.get(message.id), message, conversationId),
        )
      }
      conversation.messages = sortMessages([...messagesById.values()])
      conversation.messagesBeforePosition = page.nextBeforePosition
      conversation.messagesHasMore = page.hasMore
      serverHistoryAvailable.value = true
      persist({ upsertConversationIds: [conversationId], syncStateChanged: true })
      return true
    } catch {
      return false
    } finally {
      if (isCurrentPersistenceContext(context)) {
        const nextLoading = new Set(loadingConversationMessages.value)
        nextLoading.delete(conversationId)
        loadingConversationMessages.value = nextLoading
      }
    }
  }

  async function loadOlderConversationMessages(conversationId: string): Promise<boolean> {
    const conversation = findConversation(conversationId)
    if (!conversation) return false
    return loadConversationMessagesPage(
      conversationId,
      conversation.messagesBeforePosition ?? null,
    )
  }

  async function recoverAttempt(
    conversationId: string,
    messageId: string,
  ): Promise<ChatAttempt | null> {
    const message = findMessage(conversationId, messageId)
    if (!message?.attemptId || !browserIsOnline()) return null
    try {
      const attempt = await getChatAttempt(message.attemptId)
      if (
        attempt.conversationId !== conversationId
        || attempt.assistantMessageId !== messageId
      ) {
        return null
      }
      if (attempt.assistantMessage) {
        const conversation = findConversation(conversationId)
        if (!conversation) return null
        const index = conversation.messages.findIndex(({ id }) => id === messageId)
        const merged = mergeServerMessage(
          index >= 0 ? conversation.messages[index] : undefined,
          attempt.assistantMessage,
          conversationId,
        )
        if (attempt.status === 'processing') {
          merged.status = 'stopped'
          merged.finishReason = 'interrupted'
        } else if (attempt.status === 'failed') {
          merged.status = 'error'
          if (attempt.failureCode) merged.errorCode = attempt.failureCode
          if (attempt.failureReason) merged.errorMessage = attempt.failureReason
        }
        if (attempt.receiptId) {
          merged.receiptId = attempt.receiptId
          merged.settlementStatus = merged.settlementStatus ?? 'pending'
        }
        if (index >= 0) conversation.messages[index] = merged
        else conversation.messages.push(merged)
        sortMessages(conversation.messages)
        persist({ upsertConversationIds: [conversationId], syncStateChanged: true })
      } else if (attempt.receiptId) {
        updateMessage(conversationId, messageId, {
          receiptId: attempt.receiptId,
          settlementStatus: message.settlementStatus ?? 'pending',
        })
      }
      if (attempt.status === 'completed') void loadConversationDetail(conversationId)
      return attempt
    } catch {
      return null
    }
  }

  return {
    userId,
    conversations,
    activeConversationId,
    activeConversation,
    hasConversations,
    hydrated,
    hydrating,
    persistenceAvailable,
    persistenceDiagnostic,
    serverVersion,
    outbox,
    syncStatus,
    syncError,
    serverHistoryAvailable,
    legacyImportDecision,
    legacyConversationIds,
    legacyImportRequired,
    conversationCursor,
    conversationsHaveMore,
    loadingConversationPage,
    searchResults,
    searchingHistory,
    loadingConversationMessages,
    streamingConversationId,
    streamingMessageId,
    streamError,
    isStreaming,
    hydrate,
    createConversation,
    selectConversation,
    renameConversation,
    deleteConversation,
    clearConversations,
    setConversationModel,
    addMessage,
    updateMessage,
    removeMessages,
    upsertMessageActivity,
    appendMessageActivityDelta,
    replaceMessageActivityText,
    setMessageActivityTerminal,
    stopMessageActivities,
    startStreaming,
    appendStreamingContent,
    finishStreaming,
    failStreaming,
    stopStreaming,
    flushPersistence,
    acceptLegacyImport,
    declineLegacyImport,
    syncHistory,
    prepareConversationForCompletion,
    markCompletionAccepted,
    loadConversationPage,
    searchHistory,
    invalidateHistorySearch,
    loadConversationDetail,
    loadOlderConversationMessages,
    recoverAttempt,
  }
})
