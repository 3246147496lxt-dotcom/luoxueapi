export type ChatMessageRole = 'user' | 'assistant'

export type ChatMessageStatus = 'complete' | 'streaming' | 'stopped' | 'error'

export type ChatAttachmentKind = 'image' | 'document'

export type ChatAttachmentStatus = 'ready' | 'expired'

export interface ChatAttachment {
  id: string
  name: string
  kind: ChatAttachmentKind
  mimeType: string
  size: number
  status: ChatAttachmentStatus
  expiresAt: string
  pageCount?: number
  width?: number
  height?: number
}

export type ChatReceiptStatus =
  | 'pending'
  | 'charged'
  | 'not_charged'
  | 'subscription'
  | 'failed'

export interface ChatReceipt {
  receiptId: string
  status: ChatReceiptStatus
  usageLogId?: number
  model?: string
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
  createdAt?: string
}

export interface ChatMessage {
  id: string
  role: ChatMessageRole
  content: string
  createdAt: number
  updatedAt?: number
  position?: number
  status: ChatMessageStatus
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
}

export interface ChatConversation {
  id: string
  userId: string
  title: string
  model: string
  messages: ChatMessage[]
  createdAt: number
  updatedAt: number
  serverRevision?: number
  serverVersion?: number
  headMessageId?: string | null
  messageCount?: number
  messagesBeforePosition?: number | null
  messagesHasMore?: boolean
}

export interface ChatModelPricing {
  input?: number | null
  output?: number | null
  currency?: string
  unit?: string
}

export interface ChatModel {
  id: string
  object?: string
  display_name?: string
  description?: string
  owned_by?: string
  provider?: string
  recommended?: boolean
  input_price?: number | null
  output_price?: number | null
  pricing?: ChatModelPricing
  supports_vision?: boolean
  supports_reasoning_slider?: boolean
}

export interface ChatCatalog {
  models: ChatModel[]
  balance: number
  transcription?: ChatTranscriptionCapability
}

export interface ChatCapabilities {
  transcription?: ChatTranscriptionCapability
}

export interface ChatTranscriptionCapability {
  enabled: boolean
  billing_mode?: string
  max_upload_bytes?: number
  max_duration_seconds?: number
  accepted_mime_types?: string[]
}

export interface ChatTranscriptionResult {
  text: string
}

export type ChatCompletionMessageRole = 'system' | 'developer' | ChatMessageRole
export type ChatReasoningEffort = 'low' | 'medium' | 'high' | 'xhigh'

export interface ChatCompletionMessage {
  role: ChatCompletionMessageRole
  content: string
}

export interface ChatCompletionUserMessage {
  id: string
  content: string
  attachmentIds?: string[]
}

interface ChatCompletionHistoryRequestBase {
  conversationId: string
  model: string
  reasoningEffort?: ChatReasoningEffort
  expectedHeadMessageId: string | null
  assistantMessageId: string
}

export type ChatCompletionRequest =
  | (ChatCompletionHistoryRequestBase & {
      userMessage: ChatCompletionUserMessage
      retryOfMessageId?: never
    })
  | (ChatCompletionHistoryRequestBase & {
      userMessage?: never
      retryOfMessageId: string
    })

export interface ChatCompletionUsage {
  prompt_tokens: number
  completion_tokens: number
  total_tokens: number
  prompt_tokens_details?: Record<string, number>
  completion_tokens_details?: Record<string, number>
}

export interface ChatCompletionDelta {
  role?: ChatCompletionMessageRole
  content?: string | null
  reasoning_content?: string | null
}

export interface ChatCompletionChunkChoice {
  index: number
  delta: ChatCompletionDelta
  finish_reason: string | null
}

export interface ChatCompletionChunk {
  id?: string
  object?: string
  created?: number
  model?: string
  choices: ChatCompletionChunkChoice[]
  usage?: ChatCompletionUsage | null
  system_fingerprint?: string
  service_tier?: string
}

export interface ChatCompletionStreamResult {
  receivedDone: boolean
  finishReason: string | null
  usage: ChatCompletionUsage | null
  receiptId: string | null
}

export interface ChatCompletionStreamHandlers {
  onAccepted?: () => void
  onChunk?: (chunk: ChatCompletionChunk) => void
  onContent?: (content: string, chunk: ChatCompletionChunk) => void
  onReasoningContent?: (content: string, chunk: ChatCompletionChunk) => void
  onFinish?: (finishReason: string, chunk: ChatCompletionChunk) => void
  onUsage?: (usage: ChatCompletionUsage, chunk: ChatCompletionChunk) => void
  onReceiptId?: (receiptId: string) => void
  onDone?: (result: ChatCompletionStreamResult) => void
}

export interface ChatCompletionStreamOptions {
  signal?: AbortSignal
  requestId?: string
  attemptId?: string
}

export interface ChatReceiptPollOptions {
  signal?: AbortSignal
  delays?: readonly number[]
}

export interface ChatServerMessage extends ChatMessage {
  updatedAt?: number
  position?: number
}

export interface ChatServerConversation {
  id: string
  title: string
  model: string
  revision: number
  version: number
  headMessageId: string | null
  messageCount: number
  messages: ChatServerMessage[]
  createdAt: number
  updatedAt: number
}

export interface ChatConversationPage {
  items: ChatServerConversation[]
  nextCursor: string | null
  hasMore: boolean
}

export interface ChatMessagePage {
  items: ChatServerMessage[]
  nextBeforePosition: number | null
  hasMore: boolean
}

export interface CreateChatConversationRequest {
  id: string
  title: string
  model: string
  importedMessages?: ChatImportedMessage[]
}

export interface ChatImportedMessage {
  id: string
  role: ChatMessageRole
  content: string
  status: 'completed' | 'interrupted'
  createdAt: number
}

export interface PatchChatConversationRequest {
  revision: number
  title?: string
  model?: string
}

export interface DeleteChatConversationRequest {
  revision: number
}

export interface SearchChatConversationsRequest {
  query: string
  cursor?: string | null
  limit?: number
}

export type ChatSyncChangeType = 'upsert' | 'delete'

export interface ChatSyncChange {
  type: ChatSyncChangeType
  version: number
  conversationId: string
  conversation?: ChatServerConversation
  deletedAt?: number
}

export interface ChatSyncPage {
  changes: ChatSyncChange[]
  latestVersion: number
  nextCursor: string | null
  hasMore: boolean
}

export type ChatAttemptStatus = 'processing' | 'completed' | 'failed' | 'stopped'

export interface ChatAttempt {
  attemptId: string
  conversationId: string
  assistantMessageId: string
  status: ChatAttemptStatus
  assistantMessage?: ChatServerMessage
  receiptId?: string
  failureCode?: string
  failureReason?: string
  updatedAt?: number
}

export type ChatHistoryMutationType = 'create' | 'patch' | 'delete'

export interface ChatHistoryOutboxMutation {
  mutationId: string
  type: ChatHistoryMutationType
  conversationId: string
  createdAt: number
  revision?: number
  title?: string
  model?: string
  legacyImport?: boolean
}

export type ChatLegacyImportDecision = 'pending' | 'accepted' | 'declined' | null

export type ChatHistorySyncStatus =
  | 'idle'
  | 'syncing'
  | 'offline'
  | 'unavailable'
  | 'error'
