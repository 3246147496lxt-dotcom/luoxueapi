import axios, { type AxiosProgressEvent } from 'axios'
import {
  authSession,
  isAuthSessionChangedError,
  type AuthSessionInvalidationExpectation,
} from '@/auth/authSession'
import { refreshAuthSession } from '@/auth/authRefresh'
import { getLocale } from '@/i18n'
import type {
  ChatCompletionChunk,
  ChatCompletionRequest,
  ChatCompletionStreamHandlers,
  ChatCompletionStreamOptions,
  ChatCompletionStreamResult,
  ChatCapabilities,
  ChatCatalog,
  ChatAttachment,
  ChatAttempt,
  ChatAttemptStatus,
  ChatConversationPage,
  ChatMessage,
  ChatMessagePage,
  ChatModel,
  ChatReceipt,
  ChatReceiptPollOptions,
  ChatReceiptStatus,
  ChatServerConversation,
  ChatServerMessage,
  ChatSyncChange,
  ChatSyncPage,
  ChatTranscriptionCapability,
  ChatTranscriptionResult,
  CreateChatConversationRequest,
  DeleteChatConversationRequest,
  PatchChatConversationRequest,
  SearchChatConversationsRequest,
} from '@/types/chat'
import { apiClient } from './client'
import { buildApiUrl } from './url'

export type ChatAPIErrorCode = string | number | null

interface ChatAPIErrorOptions {
  status?: number
  code?: ChatAPIErrorCode
  reason?: unknown
  metadata?: unknown
  requestId?: string
}

interface SSEEventState {
  name: string
  data: string[]
}

let fallbackRequestIdCounter = 0

export class ChatAPIError extends Error {
  readonly status: number
  readonly code: ChatAPIErrorCode
  readonly reason: unknown
  readonly metadata: unknown
  readonly requestId: string

  constructor(message: string, options: ChatAPIErrorOptions = {}) {
    super(message)
    this.name = 'ChatAPIError'
    this.status = options.status ?? 0
    this.code = options.code ?? null
    this.reason = options.reason
    this.metadata = options.metadata
    this.requestId = options.requestId ?? ''
  }
}

function asRecord(value: unknown): Record<string, unknown> | null {
  return value !== null && typeof value === 'object'
    ? value as Record<string, unknown>
    : null
}

function nonEmptyString(value: unknown): string | null {
  return typeof value === 'string' && value.trim() ? value.trim() : null
}

function errorCode(value: unknown): ChatAPIErrorCode {
  return typeof value === 'string' || typeof value === 'number' ? value : null
}

function createErrorFromPayload(
  payload: unknown,
  fallbackMessage: string,
  options: ChatAPIErrorOptions = {},
): ChatAPIError {
  const root = asRecord(payload)
  const nested = asRecord(root?.error)
  const stringPayload = nonEmptyString(payload)
  const stringError = nonEmptyString(root?.error)
  const reason = nested?.reason ?? root?.reason ?? options.reason
  const businessReason = nonEmptyString(reason)
  const payloadCode = errorCode(nested?.code)
    ?? errorCode(nested?.type)
    ?? errorCode(root?.code)
  const message = nonEmptyString(nested?.message)
    ?? nonEmptyString(root?.message)
    ?? nonEmptyString(root?.detail)
    ?? stringError
    ?? stringPayload
    ?? fallbackMessage

  return new ChatAPIError(message, {
    ...options,
    code: typeof payloadCode === 'string'
      ? payloadCode
      : (businessReason ?? payloadCode ?? options.code),
    reason,
    metadata: nested?.metadata ?? root?.metadata ?? options.metadata,
  })
}

function abortError(): Error {
  if (typeof DOMException !== 'undefined') {
    return new DOMException('The operation was aborted.', 'AbortError')
  }
  const error = new Error('The operation was aborted.')
  error.name = 'AbortError'
  return error
}

export function createChatRequestId(): string {
  try {
    if (typeof globalThis.crypto?.randomUUID === 'function') {
      return globalThis.crypto.randomUUID()
    }
    if (typeof globalThis.crypto?.getRandomValues === 'function') {
      const bytes = globalThis.crypto.getRandomValues(new Uint8Array(16))
      return Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join('')
    }
  } catch {
    // A request ID is correlation metadata, so a non-secret fallback is safe.
  }
  fallbackRequestIdCounter += 1
  return `chat-${Date.now().toString(36)}-${fallbackRequestIdCounter.toString(36)}`
}

export function createChatAttemptId(): string {
  return createChatRequestId()
}

export function createChatIdempotencyKey(): string {
  try {
    if (typeof globalThis.crypto?.randomUUID === 'function') {
      return globalThis.crypto.randomUUID()
    }
    if (typeof globalThis.crypto?.getRandomValues === 'function') {
      const bytes = globalThis.crypto.getRandomValues(new Uint8Array(16))
      bytes[6] = ((bytes[6] ?? 0) & 0x0f) | 0x40
      bytes[8] = ((bytes[8] ?? 0) & 0x3f) | 0x80
      const hex = Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join('')
      return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`
    }
  } catch {
    // The fallback remains a UUID-shaped idempotency token, not a credential.
  }
  const seed = `${Date.now().toString(16).padStart(12, '0')}${(++fallbackRequestIdCounter)
    .toString(16)
    .padStart(20, '0')}`
  return `${seed.slice(0, 8)}-${seed.slice(8, 12)}-4${seed.slice(13, 16)}-8${seed.slice(17, 20)}-${seed.slice(20, 32)}`
}

export function isAbortError(error: unknown): boolean {
  return axios.isCancel(error)
    || (
      !!error
      && typeof error === 'object'
      && (
        (error as { name?: unknown }).name === 'AbortError'
        || (error as { code?: unknown }).code === 'ERR_CANCELED'
      )
    )
}

function throwIfAborted(signal?: AbortSignal): void {
  if (signal?.aborted) throw abortError()
}

function parseJSON(value: string): unknown {
  try {
    return JSON.parse(value)
  } catch {
    throw new ChatAPIError('The chat stream returned an invalid event.', {
      code: 'INVALID_STREAM_EVENT',
      metadata: { data: value },
    })
  }
}

function isChatCompletionChunk(value: unknown): value is ChatCompletionChunk {
  const candidate = asRecord(value)
  return !!candidate && Array.isArray(candidate.choices)
}

function applyChunk(
  chunk: ChatCompletionChunk,
  handlers: ChatCompletionStreamHandlers,
  result: ChatCompletionStreamResult,
): void {
  handlers.onChunk?.(chunk)

  for (const choice of chunk.choices) {
    const content = choice?.delta?.content
    if (typeof content === 'string' && content) {
      handlers.onContent?.(content, chunk)
    }

    const reasoningContent = choice?.delta?.reasoning_content
    if (typeof reasoningContent === 'string' && reasoningContent) {
      handlers.onReasoningContent?.(reasoningContent, chunk)
    }

    if (typeof choice?.finish_reason === 'string' && choice.finish_reason) {
      result.finishReason = choice.finish_reason
      handlers.onFinish?.(choice.finish_reason, chunk)
    }
  }

  if (chunk.usage) {
    result.usage = chunk.usage
    handlers.onUsage?.(chunk.usage, chunk)
  }
}

function dispatchSSEEvent(
  event: SSEEventState,
  handlers: ChatCompletionStreamHandlers,
  result: ChatCompletionStreamResult,
): boolean {
  const eventName = event.name || 'message'
  const data = event.data.join('\n')
  event.name = ''
  event.data = []

  if (eventName === 'error' && !data.trim()) {
    throw new ChatAPIError('The chat stream reported an error.', {
      code: 'STREAM_ERROR',
    })
  }
  if (!data.trim()) return false
  if (data.trim() === '[DONE]') {
    result.receivedDone = true
    return true
  }

  const payload = parseJSON(data)
  const record = asRecord(payload)
  if (eventName === 'error' || record?.error !== undefined) {
    throw createErrorFromPayload(payload, 'The chat stream reported an error.', {
      code: 'STREAM_ERROR',
    })
  }
  if (!isChatCompletionChunk(payload)) {
    throw new ChatAPIError('The chat stream returned an unexpected event.', {
      code: 'INVALID_STREAM_EVENT',
      metadata: { event: eventName },
    })
  }

  applyChunk(payload, handlers, result)
  return false
}

/**
 * Parse an OpenAI-compatible SSE stream. It accepts arbitrary byte chunks,
 * every SSE line-ending variant, multi-line data fields, and a final event
 * that is not followed by a blank line.
 */
export async function parseChatCompletionSSE(
  stream: ReadableStream<Uint8Array>,
  handlers: ChatCompletionStreamHandlers = {},
  options: ChatCompletionStreamOptions = {},
): Promise<ChatCompletionStreamResult> {
  const { signal } = options
  throwIfAborted(signal)

  const reader = stream.getReader()
  const decoder = new TextDecoder()
  const event: SSEEventState = { name: '', data: [] }
  const result: ChatCompletionStreamResult = {
    receivedDone: false,
    finishReason: null,
    usage: null,
    receiptId: null,
  }
  let buffer = ''

  const processLine = (line: string): boolean => {
    if (line === '') return dispatchSSEEvent(event, handlers, result)
    if (line.startsWith(':')) return false

    const separator = line.indexOf(':')
    const field = separator === -1 ? line : line.slice(0, separator)
    let value = separator === -1 ? '' : line.slice(separator + 1)
    if (value.startsWith(' ')) value = value.slice(1)

    if (field === 'event') event.name = value
    if (field === 'data') event.data.push(value)
    return false
  }

  const consumeLines = (atEnd: boolean): boolean => {
    let offset = 0

    while (offset < buffer.length) {
      let lineEnd = offset
      while (lineEnd < buffer.length && buffer[lineEnd] !== '\n' && buffer[lineEnd] !== '\r') {
        lineEnd++
      }
      if (lineEnd === buffer.length) break
      if (buffer[lineEnd] === '\r' && lineEnd + 1 === buffer.length && !atEnd) break

      const line = buffer.slice(offset, lineEnd)
      const newlineLength = buffer[lineEnd] === '\r' && buffer[lineEnd + 1] === '\n' ? 2 : 1
      offset = lineEnd + newlineLength
      if (processLine(line)) {
        buffer = buffer.slice(offset)
        return true
      }
    }

    buffer = buffer.slice(offset)
    if (atEnd && buffer) {
      const tail = buffer
      buffer = ''
      if (processLine(tail)) return true
    }
    return false
  }

  let cancelPromise: Promise<void> | null = null
  const cancelReader = (): Promise<void> => {
    if (cancelPromise) return cancelPromise

    try {
      cancelPromise = reader.cancel().catch(() => undefined)
    } catch {
      cancelPromise = Promise.resolve()
    }
    return cancelPromise
  }
  const cancelOnAbort = () => {
    void cancelReader()
  }
  signal?.addEventListener('abort', cancelOnAbort, { once: true })

  try {
    let stopped = false
    while (!stopped) {
      throwIfAborted(signal)
      const { done, value } = await reader.read()
      throwIfAborted(signal)

      if (done) {
        buffer += decoder.decode()
        stopped = consumeLines(true)
        if (!stopped && (event.data.length > 0 || event.name === 'error')) {
          dispatchSSEEvent(event, handlers, result)
        }
        break
      }

      buffer += decoder.decode(value, { stream: true })
      stopped = consumeLines(false)
    }

    if (stopped) {
      await cancelReader()
    }

    handlers.onDone?.(result)
    return result
  } catch (error) {
    await cancelReader()
    if (signal?.aborted) throw abortError()
    throw error
  } finally {
    signal?.removeEventListener('abort', cancelOnAbort)
    try {
      reader.releaseLock()
    } catch {
      // A cancelled reader may already have released its lock.
    }
  }
}

function modelList(payload: unknown): unknown[] | null {
  const record = asRecord(payload)
  if (Array.isArray(record?.models)) return record.models
  return null
}

function normalizeChatModel(value: unknown): ChatModel | null {
  const candidate = asRecord(value)
  const id = nonEmptyString(candidate?.id)
  if (!candidate || !id) return null
  const model = { ...candidate, id } as unknown as ChatModel
  const supportsVision = candidate.supports_vision ?? candidate.supportsVision
  if (typeof supportsVision === 'boolean') model.supports_vision = supportsVision
  const supportsReasoningSlider = candidate.supports_reasoning_slider
    ?? candidate.supportsReasoningSlider
  delete (model as ChatModel & { supportsReasoningSlider?: unknown }).supportsReasoningSlider
  if (typeof supportsReasoningSlider === 'boolean') {
    model.supports_reasoning_slider = supportsReasoningSlider
  } else {
    delete model.supports_reasoning_slider
  }
  return model
}

function normalizeRequestError(error: unknown, fallbackMessage: string): ChatAPIError {
  if (error instanceof ChatAPIError) return error
  const candidate = asRecord(error)
  const status = typeof candidate?.status === 'number' && Number.isFinite(candidate.status)
    ? candidate.status
    : 0
  return createErrorFromPayload(error, fallbackMessage, {
    status,
    code: errorCode(candidate?.code),
    reason: candidate?.reason,
    metadata: candidate?.metadata,
    requestId: nonEmptyString(candidate?.requestId) ?? '',
  })
}

export async function getChatModels(): Promise<ChatCatalog> {
  try {
    const { data } = await apiClient.get<unknown>('/chat/models')
    const response = asRecord(data)
    const items = modelList(data)
    const balance = response?.balance
    if (!items || typeof balance !== 'number' || !Number.isFinite(balance)) {
      throw new ChatAPIError('The chat model list returned an invalid response.', {
        code: 'INVALID_MODELS_RESPONSE',
      })
    }
    const catalog: ChatCatalog = {
      models: items.map(normalizeChatModel).filter((model): model is ChatModel => model !== null),
      balance,
    }
    const transcription = normalizeTranscriptionCapability(response?.transcription)
    if (transcription) catalog.transcription = transcription
    return catalog
  } catch (error) {
    throw normalizeRequestError(error, 'Unable to load the chat model list.')
  }
}

export async function getChatCapabilities(signal?: AbortSignal): Promise<ChatCapabilities> {
  try {
    const { data } = await apiClient.get<unknown>('/chat/capabilities', { signal })
    const response = asRecord(data)
    if (!response) {
      throw new ChatAPIError('The chat capabilities endpoint returned an invalid response.', {
        code: 'INVALID_CHAT_CAPABILITIES_RESPONSE',
      })
    }
    const capabilities: ChatCapabilities = {}
    const transcription = normalizeTranscriptionCapability(response.transcription)
    if (transcription) capabilities.transcription = transcription
    return capabilities
  } catch (error) {
    if (signal?.aborted || isAbortError(error)) throw error
    throw normalizeRequestError(error, 'Unable to load chat capabilities.')
  }
}

export interface ChatTranscriptionOptions {
  signal?: AbortSignal
  timeoutMs?: number
  idempotencyKey?: string
}

function normalizePositiveInteger(value: unknown): number | undefined {
  return typeof value === 'number' && Number.isSafeInteger(value) && value > 0
    ? value
    : undefined
}

function normalizeNonNegativeInteger(value: unknown): number | undefined {
  return typeof value === 'number' && Number.isSafeInteger(value) && value >= 0
    ? value
    : undefined
}

export function normalizeChatAttachment(value: unknown): ChatAttachment | null {
  const candidate = asRecord(value)
  if (!candidate) return null

  const id = nonEmptyString(candidate.id)
  const name = nonEmptyString(candidate.name)
  const kind = candidate.kind
  const mimeType = nonEmptyString(candidate.mime_type ?? candidate.mimeType)
  const size = normalizeNonNegativeInteger(candidate.size)
  const status = candidate.status
  const expiresAt = nonEmptyString(candidate.expires_at ?? candidate.expiresAt)

  if (
    !id
    || !name
    || (kind !== 'image' && kind !== 'document')
    || !mimeType
    || size === undefined
    || (status !== 'ready' && status !== 'expired')
    || !expiresAt
  ) return null

  const attachment: ChatAttachment = {
    id,
    name,
    kind,
    mimeType,
    size,
    status,
    expiresAt,
  }
  const pageCount = normalizePositiveInteger(candidate.page_count ?? candidate.pageCount)
  const width = normalizePositiveInteger(candidate.width)
  const height = normalizePositiveInteger(candidate.height)
  if (pageCount !== undefined) attachment.pageCount = pageCount
  if (width !== undefined) attachment.width = width
  if (height !== undefined) attachment.height = height
  return attachment
}

export interface ChatAttachmentUploadOptions {
  signal?: AbortSignal
  onProgress?: (progress: number) => void
}

function attachmentUploadFile(file: File): File {
  if (file.type.trim()) return file
  const extension = file.name.split('.').pop()?.toLowerCase() ?? ''
  const canonicalMimeType: Record<string, string> = {
    jpg: 'image/jpeg',
    jpeg: 'image/jpeg',
    png: 'image/png',
    webp: 'image/webp',
    pdf: 'application/pdf',
    docx: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
  }
  const mimeType = canonicalMimeType[extension]
  return mimeType
    ? new File([file], file.name, { type: mimeType, lastModified: file.lastModified })
    : file
}

export async function uploadChatAttachment(
  file: File,
  options: ChatAttachmentUploadOptions = {},
): Promise<ChatAttachment> {
  throwIfAborted(options.signal)
  if (!(file instanceof File) || file.size <= 0) {
    throw new ChatAPIError('A non-empty attachment is required.', {
      code: 'EMPTY_ATTACHMENT',
    })
  }

  const formData = new FormData()
  formData.append('file', attachmentUploadFile(file), file.name)

  try {
    const { data } = await apiClient.post<unknown>('/chat/attachments', formData, {
      signal: options.signal,
      timeout: 0,
      headers: { 'Content-Type': undefined },
      onUploadProgress: (event: AxiosProgressEvent) => {
        const total = event.total && event.total > 0 ? event.total : file.size
        const progress = total > 0 ? Math.round((event.loaded / total) * 100) : 0
        options.onProgress?.(Math.max(0, Math.min(100, progress)))
      },
    })
    throwIfAborted(options.signal)
    const payload = asRecord(data)?.attachment ?? data
    const attachment = normalizeChatAttachment(payload)
    if (!attachment) {
      throw new ChatAPIError('The attachment upload returned an invalid response.', {
        code: 'INVALID_ATTACHMENT_RESPONSE',
      })
    }
    return attachment
  } catch (error) {
    if (options.signal?.aborted || isAbortError(error)) throw abortError()
    throw normalizeRequestError(error, 'Unable to upload the attachment.')
  }
}

export async function deleteChatAttachment(id: string, signal?: AbortSignal): Promise<void> {
  const normalizedId = id.trim()
  if (!normalizedId) {
    throw new ChatAPIError('An attachment ID is required.', {
      code: 'ATTACHMENT_ID_REQUIRED',
    })
  }
  throwIfAborted(signal)
  try {
    await apiClient.delete(`/chat/attachments/${encodeURIComponent(normalizedId)}`, { signal })
    throwIfAborted(signal)
  } catch (error) {
    if (signal?.aborted || isAbortError(error)) throw abortError()
    throw normalizeRequestError(error, 'Unable to delete the attachment.')
  }
}

export async function getChatAttachmentContent(
  id: string,
  signal?: AbortSignal,
): Promise<Blob> {
  const normalizedId = id.trim()
  if (!normalizedId) {
    throw new ChatAPIError('An attachment ID is required.', {
      code: 'ATTACHMENT_ID_REQUIRED',
    })
  }
  throwIfAborted(signal)
  try {
    const { data } = await apiClient.get<Blob>(
      `/chat/attachments/${encodeURIComponent(normalizedId)}/content`,
      { signal, responseType: 'blob' },
    )
    throwIfAborted(signal)
    if (!(data instanceof Blob)) {
      throw new ChatAPIError('The attachment content response was invalid.', {
        code: 'INVALID_ATTACHMENT_CONTENT_RESPONSE',
      })
    }
    return data
  } catch (error) {
    if (signal?.aborted || isAbortError(error)) throw abortError()
    throw normalizeRequestError(error, 'Unable to load the attachment.')
  }
}

function normalizeTranscriptionCapability(value: unknown): ChatTranscriptionCapability | undefined {
  const candidate = asRecord(value)
  if (!candidate || typeof candidate.enabled !== 'boolean') return undefined
  const capability: ChatTranscriptionCapability = {
    enabled: candidate.enabled,
  }
  const billingMode = nonEmptyString(candidate.billing_mode ?? candidate.billingMode)
  const maxUploadBytes = normalizePositiveInteger(
    candidate.max_upload_bytes ?? candidate.maxUploadBytes,
  )
  const maxDurationSeconds = normalizePositiveInteger(
    candidate.max_duration_seconds ?? candidate.maxDurationSeconds,
  )
  const rawMimeTypes = candidate.accepted_mime_types ?? candidate.acceptedMimeTypes
  const acceptedMimeTypes = Array.isArray(rawMimeTypes)
    ? rawMimeTypes
        .map((mimeType) => nonEmptyString(mimeType))
        .filter((mimeType): mimeType is string => mimeType !== null)
    : []
  if (billingMode) capability.billing_mode = billingMode
  if (maxUploadBytes !== undefined) capability.max_upload_bytes = maxUploadBytes
  if (maxDurationSeconds !== undefined) capability.max_duration_seconds = maxDurationSeconds
  if (acceptedMimeTypes.length > 0) capability.accepted_mime_types = acceptedMimeTypes
  return capability
}

function audioFileExtension(mimeType: string): string {
  const normalized = mimeType.toLowerCase()
  if (normalized.includes('mp4') || normalized.includes('m4a')) return 'm4a'
  if (normalized.includes('ogg')) return 'ogg'
  return 'webm'
}

export async function transcribeChatAudio(
  audio: Blob,
  options: ChatTranscriptionOptions = {},
): Promise<ChatTranscriptionResult> {
  const {
    signal,
    // Upload and provider deadlines are enforced by the server and may be
    // configured above 90 seconds. Axios 0 disables its shorter client timer;
    // callers can still cancel immediately through AbortSignal.
    timeoutMs = 0,
    idempotencyKey = createChatIdempotencyKey(),
  } = options
  throwIfAborted(signal)

  if (audio.size <= 0) {
    throw new ChatAPIError('A non-empty audio recording is required.', {
      code: 'EMPTY_AUDIO',
    })
  }

  const formData = new FormData()
  const extension = audioFileExtension(audio.type)
  formData.append('file', audio, `chat-recording.${extension}`)

  try {
    const { data } = await apiClient.post<unknown>(
      '/chat/transcriptions',
      formData,
      {
        signal,
        timeout: timeoutMs,
        // Clear the instance-wide JSON default before Axios transforms the
        // body. The browser adapter will then add multipart/form-data with its
        // generated boundary.
        headers: {
          'Content-Type': undefined,
          'Idempotency-Key': idempotencyKey,
        },
      },
    )
    throwIfAborted(signal)
    const text = nonEmptyString(asRecord(data)?.text)
    if (!text) {
      throw new ChatAPIError('The transcription response did not include text.', {
        code: 'INVALID_TRANSCRIPTION_RESPONSE',
      })
    }
    return { text }
  } catch (error) {
    if (signal?.aborted || isAbortError(error)) throw abortError()
    throw normalizeRequestError(error, 'Unable to transcribe the audio recording.')
  }
}

const CHAT_RECEIPT_STATUSES = new Set<ChatReceiptStatus>([
  'pending',
  'charged',
  'not_charged',
  'subscription',
  'failed',
])
const DEFAULT_CHAT_RECEIPT_POLL_DELAYS = [150, 300, 600, 1200, 2400] as const

function optionalFiniteNumber(value: unknown): number | undefined {
  if (typeof value === 'number' && Number.isFinite(value)) return value
  if (typeof value === 'string' && value.trim()) {
    const parsed = Number(value)
    if (Number.isFinite(parsed)) return parsed
  }
  return undefined
}

function optionalNonNegativeInteger(value: unknown): number | undefined {
  const parsed = optionalFiniteNumber(value)
  return parsed !== undefined && Number.isSafeInteger(parsed) && parsed >= 0 ? parsed : undefined
}

function receiptField(
  record: Record<string, unknown>,
  snakeCase: string,
  camelCase: string,
): unknown {
  return record[snakeCase] ?? record[camelCase]
}

function normalizeChatReceipt(
  payload: unknown,
  fallbackReceiptId: string,
  pendingFallback = false,
): ChatReceipt | null {
  const candidate = asRecord(payload)
  if (!candidate) {
    return pendingFallback ? { receiptId: fallbackReceiptId, status: 'pending' } : null
  }

  const receiptId = nonEmptyString(receiptField(candidate, 'receipt_id', 'receiptId'))
    ?? fallbackReceiptId
  const rawStatus = nonEmptyString(candidate.status)
  const status = rawStatus && CHAT_RECEIPT_STATUSES.has(rawStatus as ChatReceiptStatus)
    ? rawStatus as ChatReceiptStatus
    : (pendingFallback ? 'pending' : null)
  if (!receiptId || !status) return null

  const receipt: ChatReceipt = { receiptId, status }
  const usageLogId = optionalNonNegativeInteger(receiptField(candidate, 'usage_log_id', 'usageLogId'))
  const model = nonEmptyString(candidate.model)
  const inputTokens = optionalNonNegativeInteger(receiptField(candidate, 'input_tokens', 'inputTokens'))
  const outputTokens = optionalNonNegativeInteger(receiptField(candidate, 'output_tokens', 'outputTokens'))
  const cacheCreationTokens = optionalNonNegativeInteger(
    receiptField(candidate, 'cache_creation_tokens', 'cacheCreationTokens'),
  )
  const cacheReadTokens = optionalNonNegativeInteger(
    receiptField(candidate, 'cache_read_tokens', 'cacheReadTokens'),
  )
  const totalTokens = optionalNonNegativeInteger(receiptField(candidate, 'total_tokens', 'totalTokens'))
  const grossCost = optionalFiniteNumber(receiptField(candidate, 'gross_cost', 'grossCost'))
  const chargedAmount = optionalFiniteNumber(
    receiptField(candidate, 'charged_amount', 'chargedAmount'),
  )
  const rawBillingType = receiptField(candidate, 'billing_type', 'billingType')
  const billingType = typeof rawBillingType === 'string'
    ? nonEmptyString(rawBillingType) ?? undefined
    : optionalFiniteNumber(rawBillingType)
  const balanceBefore = optionalFiniteNumber(
    receiptField(candidate, 'balance_before', 'balanceBefore'),
  )
  const balanceAfter = optionalFiniteNumber(
    receiptField(candidate, 'balance_after', 'balanceAfter'),
  )
  const createdAt = nonEmptyString(receiptField(candidate, 'created_at', 'createdAt'))

  if (usageLogId !== undefined) receipt.usageLogId = usageLogId
  if (model) receipt.model = model
  if (inputTokens !== undefined) receipt.inputTokens = inputTokens
  if (outputTokens !== undefined) receipt.outputTokens = outputTokens
  if (cacheCreationTokens !== undefined) receipt.cacheCreationTokens = cacheCreationTokens
  if (cacheReadTokens !== undefined) receipt.cacheReadTokens = cacheReadTokens
  if (totalTokens !== undefined) receipt.totalTokens = totalTokens
  if (grossCost !== undefined) receipt.grossCost = grossCost
  if (chargedAmount !== undefined) receipt.chargedAmount = chargedAmount
  if (billingType !== undefined) receipt.billingType = billingType
  if (balanceBefore !== undefined) receipt.balanceBefore = balanceBefore
  if (balanceAfter !== undefined) receipt.balanceAfter = balanceAfter
  if (createdAt) receipt.createdAt = createdAt
  return receipt
}

export async function getChatReceipt(
  receiptId: string,
  signal?: AbortSignal,
): Promise<ChatReceipt> {
  const normalizedReceiptId = receiptId.trim()
  if (!normalizedReceiptId) {
    throw new ChatAPIError('A chat receipt ID is required.', {
      code: 'RECEIPT_ID_REQUIRED',
    })
  }
  throwIfAborted(signal)

  try {
    const response = await apiClient.get<unknown>(
      `/chat/receipts/${encodeURIComponent(normalizedReceiptId)}`,
      { signal },
    )
    throwIfAborted(signal)
    const pending = response.status === 202
    const receipt = normalizeChatReceipt(response.data, normalizedReceiptId, pending)
    if (!receipt) {
      throw new ChatAPIError('The chat receipt returned an invalid response.', {
        status: response.status,
        code: 'INVALID_RECEIPT_RESPONSE',
      })
    }
    return receipt
  } catch (error) {
    if (signal?.aborted || isAbortError(error)) throw abortError()
    throw normalizeRequestError(error, 'Unable to load the chat receipt.')
  }
}

function waitForReceiptPoll(delay: number, signal?: AbortSignal): Promise<void> {
  throwIfAborted(signal)
  return new Promise<void>((resolve, reject) => {
    const timeoutId = setTimeout(() => {
      signal?.removeEventListener('abort', onAbort)
      resolve()
    }, Math.max(0, delay))
    const onAbort = () => {
      clearTimeout(timeoutId)
      reject(abortError())
    }
    signal?.addEventListener('abort', onAbort, { once: true })
  })
}

export async function pollChatReceipt(
  receiptId: string,
  options: ChatReceiptPollOptions = {},
): Promise<ChatReceipt> {
  const normalizedReceiptId = receiptId.trim()
  if (!normalizedReceiptId) {
    throw new ChatAPIError('A chat receipt ID is required.', {
      code: 'RECEIPT_ID_REQUIRED',
    })
  }

  const delays = options.delays ?? DEFAULT_CHAT_RECEIPT_POLL_DELAYS
  let receipt: ChatReceipt = { receiptId: normalizedReceiptId, status: 'pending' }

  for (let attempt = 0; attempt <= delays.length; attempt += 1) {
    receipt = await getChatReceipt(normalizedReceiptId, options.signal)
    if (receipt.status !== 'pending') return receipt
    if (attempt < delays.length) {
      await waitForReceiptPoll(delays[attempt] ?? 0, options.signal)
    }
  }

  return receipt
}

function optionalTimestamp(value: unknown): number | undefined {
  if (typeof value === 'string' && value.trim()) {
    const numeric = Number(value)
    if (Number.isFinite(numeric)) return optionalTimestamp(numeric)
    const parsed = Date.parse(value)
    return Number.isFinite(parsed) ? parsed : undefined
  }
  if (typeof value !== 'number' || !Number.isFinite(value) || value < 0) return undefined
  return value > 0 && value < 100_000_000_000 ? value * 1000 : value
}

function optionalBoolean(value: unknown): boolean | undefined {
  return typeof value === 'boolean' ? value : undefined
}

function normalizeServerMessage(value: unknown): ChatServerMessage | null {
  const candidate = asRecord(value)
  const id = nonEmptyString(candidate?.id ?? candidate?.message_id)
  const role = candidate?.role
  if (!candidate || !id || (role !== 'user' && role !== 'assistant')) return null

  const content = typeof candidate.content === 'string' ? candidate.content : ''
  const rawStatus = nonEmptyString(candidate.status)
  const acceptedStatus: ChatMessage['status'] =
    rawStatus === 'streaming'
      || rawStatus === 'processing'
      || rawStatus === 'interrupted'
      ? 'stopped'
      : (
          rawStatus === 'complete'
          || rawStatus === 'completed'
          || rawStatus === 'stopped'
          || rawStatus === 'error'
        )
        ? (rawStatus === 'completed' ? 'complete' : rawStatus)
        : 'complete'
  const createdAt = optionalTimestamp(candidate.created_at ?? candidate.createdAt) ?? Date.now()
  const message: ChatServerMessage = {
    id,
    role,
    content,
    createdAt,
    status: acceptedStatus,
  }
  if (Array.isArray(candidate.attachments)) {
    const attachments = candidate.attachments
      .map(normalizeChatAttachment)
      .filter((attachment): attachment is ChatAttachment => attachment !== null)
    if (attachments.length > 0) message.attachments = attachments
  }
  if (
    rawStatus === 'streaming'
    || rawStatus === 'processing'
    || rawStatus === 'interrupted'
  ) {
    message.finishReason = 'interrupted'
  } else if (typeof candidate.finish_reason === 'string' || candidate.finish_reason === null) {
    message.finishReason = candidate.finish_reason as string | null
  } else if (typeof candidate.finishReason === 'string' || candidate.finishReason === null) {
    message.finishReason = candidate.finishReason as string | null
  }

  const updatedAt = optionalTimestamp(candidate.updated_at ?? candidate.updatedAt)
  const position = optionalNonNegativeInteger(candidate.position)
  const attemptId = nonEmptyString(candidate.attempt_id ?? candidate.attemptId)
  const receiptRecord = asRecord(candidate.receipt)
  const receiptId = nonEmptyString(
    candidate.receipt_id
      ?? candidate.receiptId
      ?? receiptRecord?.receipt_id
      ?? receiptRecord?.receiptId,
  )
  const settlementStatus = nonEmptyString(
    candidate.settlement_status
      ?? candidate.settlementStatus
      ?? receiptRecord?.status,
  )
  const requestedModel = nonEmptyString(candidate.requested_model ?? candidate.requestedModel)
  const actualModel = nonEmptyString(
    candidate.actual_model
      ?? candidate.actualModel
      ?? receiptRecord?.model,
  )
  const errorCodeValue = nonEmptyString(candidate.error_code ?? candidate.errorCode)
  const errorMessageValue = nonEmptyString(candidate.error_message ?? candidate.errorMessage)
  const supersededByMessageId = nonEmptyString(
    candidate.superseded_by_message_id ?? candidate.supersededByMessageId,
  )
  const billingSource = receiptRecord ?? candidate
  const usageLogId = optionalNonNegativeInteger(
    billingSource.usage_log_id ?? billingSource.usageLogId,
  )
  const inputTokens = optionalNonNegativeInteger(
    billingSource.input_tokens ?? billingSource.inputTokens,
  )
  const outputTokens = optionalNonNegativeInteger(
    billingSource.output_tokens ?? billingSource.outputTokens,
  )
  const cacheCreationTokens = optionalNonNegativeInteger(
    billingSource.cache_creation_tokens ?? billingSource.cacheCreationTokens,
  )
  const cacheReadTokens = optionalNonNegativeInteger(
    billingSource.cache_read_tokens ?? billingSource.cacheReadTokens,
  )
  const totalTokens = optionalNonNegativeInteger(
    billingSource.total_tokens ?? billingSource.totalTokens,
  )
  const grossCost = optionalFiniteNumber(billingSource.gross_cost ?? billingSource.grossCost)
  const chargedAmount = optionalFiniteNumber(
    billingSource.charged_amount ?? billingSource.chargedAmount,
  )
  const balanceBefore = optionalFiniteNumber(
    billingSource.balance_before ?? billingSource.balanceBefore,
  )
  const balanceAfter = optionalFiniteNumber(
    billingSource.balance_after ?? billingSource.balanceAfter,
  )
  const rawBillingType = billingSource.billing_type ?? billingSource.billingType
  const billingType = typeof rawBillingType === 'string'
    ? nonEmptyString(rawBillingType) ?? undefined
    : optionalFiniteNumber(rawBillingType)
  const receiptCreatedAt = nonEmptyString(
    billingSource.created_at ?? billingSource.createdAt,
  )

  if (updatedAt !== undefined) message.updatedAt = updatedAt
  if (position !== undefined) message.position = position
  if (attemptId) message.attemptId = attemptId
  if (receiptId) message.receiptId = receiptId
  if (settlementStatus && CHAT_RECEIPT_STATUSES.has(settlementStatus as ChatReceiptStatus)) {
    message.settlementStatus = settlementStatus as ChatReceiptStatus
  }
  if (requestedModel) message.requestedModel = requestedModel
  if (actualModel) message.actualModel = actualModel
  if (errorCodeValue) message.errorCode = errorCodeValue
  if (errorMessageValue) message.errorMessage = errorMessageValue
  if (supersededByMessageId) message.supersededByMessageId = supersededByMessageId
  if (optionalBoolean(candidate.excluded_from_context ?? candidate.excludedFromContext) !== undefined) {
    message.excludedFromContext = Boolean(
      candidate.excluded_from_context ?? candidate.excludedFromContext,
    )
  }
  if (usageLogId !== undefined) message.usageLogId = usageLogId
  if (inputTokens !== undefined) message.inputTokens = inputTokens
  if (outputTokens !== undefined) message.outputTokens = outputTokens
  if (cacheCreationTokens !== undefined) message.cacheCreationTokens = cacheCreationTokens
  if (cacheReadTokens !== undefined) message.cacheReadTokens = cacheReadTokens
  if (totalTokens !== undefined) message.totalTokens = totalTokens
  if (grossCost !== undefined) message.grossCost = grossCost
  if (chargedAmount !== undefined) message.chargedAmount = chargedAmount
  if (billingType !== undefined) message.billingType = billingType
  if (balanceBefore !== undefined) message.balanceBefore = balanceBefore
  if (balanceAfter !== undefined) message.balanceAfter = balanceAfter
  if (receiptCreatedAt) message.receiptCreatedAt = receiptCreatedAt
  return message
}

function sortServerMessages(messages: ChatServerMessage[]): ChatServerMessage[] {
  const hasCanonicalPositions = messages.every(({ position }) => position !== undefined)
  return messages.sort((left, right) => {
    if (hasCanonicalPositions) {
      return (left.position ?? 0) - (right.position ?? 0)
        || left.createdAt - right.createdAt
    }
    return left.createdAt - right.createdAt
  })
}

function normalizeServerConversation(value: unknown): ChatServerConversation | null {
  const candidate = asRecord(value)
  const id = nonEmptyString(candidate?.id ?? candidate?.conversation_id)
  const title = nonEmptyString(candidate?.title)
  const model = nonEmptyString(candidate?.model)
  if (!candidate || !id || !title || !model) return null

  const messagesValue = Array.isArray(candidate.messages) ? candidate.messages : []
  const messages = sortServerMessages(
    messagesValue
      .map(normalizeServerMessage)
      .filter((message): message is ChatServerMessage => message !== null),
  )
  const revision = optionalNonNegativeInteger(candidate.revision) ?? 0
  const version = optionalNonNegativeInteger(candidate.version) ?? revision
  const createdAt = optionalTimestamp(candidate.created_at ?? candidate.createdAt) ?? Date.now()
  const updatedAt = optionalTimestamp(candidate.updated_at ?? candidate.updatedAt) ?? createdAt
  const messageCount = optionalNonNegativeInteger(
    candidate.message_count ?? candidate.messageCount,
  ) ?? messages.length
  const headMessageId = nonEmptyString(
    candidate.head_message_id ?? candidate.headMessageId,
  ) ?? messages[messages.length - 1]?.id ?? null

  return {
    id,
    title,
    model,
    revision,
    version,
    headMessageId,
    messageCount,
    messages,
    createdAt,
    updatedAt,
  }
}

function normalizeConversationPage(payload: unknown): ChatConversationPage {
  const candidate = asRecord(payload)
  const rawItems = Array.isArray(candidate?.items)
    ? candidate.items
    : (Array.isArray(candidate?.conversations) ? candidate.conversations : null)
  if (!rawItems) {
    throw new ChatAPIError('The conversation list returned an invalid response.', {
      code: 'INVALID_CONVERSATION_LIST_RESPONSE',
    })
  }
  const items = rawItems
    .map(normalizeServerConversation)
    .filter((conversation): conversation is ChatServerConversation => conversation !== null)
  const nextCursor = nonEmptyString(candidate?.next_cursor ?? candidate?.nextCursor)
  return {
    items,
    nextCursor,
    hasMore: optionalBoolean(candidate?.has_more ?? candidate?.hasMore) ?? Boolean(nextCursor),
  }
}

export async function createChatConversation(
  request: CreateChatConversationRequest,
  signal?: AbortSignal,
): Promise<ChatServerConversation> {
  try {
    const { data } = await apiClient.post<unknown>('/chat/conversations', {
      id: request.id,
      title: request.title,
      model: request.model,
      ...(request.importedMessages
        ? {
            imported_messages: request.importedMessages.map((message) => ({
              id: message.id,
              role: message.role,
              content: message.content,
              status: message.status,
              created_at: new Date(message.createdAt).toISOString(),
            })),
          }
        : {}),
    }, { signal })
    const conversation = normalizeServerConversation(
      asRecord(data)?.conversation ?? data,
    )
    if (!conversation) {
      throw new ChatAPIError('The created conversation returned an invalid response.', {
        code: 'INVALID_CONVERSATION_RESPONSE',
      })
    }
    return conversation
  } catch (error) {
    if (signal?.aborted || isAbortError(error)) throw abortError()
    throw normalizeRequestError(error, 'Unable to create the conversation.')
  }
}

export async function listChatConversations(
  options: { cursor?: string | null; limit?: number; signal?: AbortSignal } = {},
): Promise<ChatConversationPage> {
  try {
    const { data } = await apiClient.get<unknown>('/chat/conversations', {
      params: {
        ...(options.cursor ? { cursor: options.cursor } : {}),
        ...(options.limit ? { limit: options.limit } : {}),
      },
      signal: options.signal,
    })
    return normalizeConversationPage(data)
  } catch (error) {
    if (options.signal?.aborted || isAbortError(error)) throw abortError()
    throw normalizeRequestError(error, 'Unable to load conversations.')
  }
}

export async function getChatConversation(
  conversationId: string,
  signal?: AbortSignal,
): Promise<ChatServerConversation> {
  try {
    const { data } = await apiClient.get<unknown>(
      `/chat/conversations/${encodeURIComponent(conversationId)}`,
      { signal },
    )
    const conversation = normalizeServerConversation(
      asRecord(data)?.conversation ?? data,
    )
    if (!conversation) {
      throw new ChatAPIError('The conversation returned an invalid response.', {
        code: 'INVALID_CONVERSATION_RESPONSE',
      })
    }
    return conversation
  } catch (error) {
    if (signal?.aborted || isAbortError(error)) throw abortError()
    throw normalizeRequestError(error, 'Unable to load the conversation.')
  }
}

export async function getChatConversationMessages(
  conversationId: string,
  options: { beforePosition?: number | null; limit?: number; signal?: AbortSignal } = {},
): Promise<ChatMessagePage> {
  try {
    const { data } = await apiClient.get<unknown>(
      `/chat/conversations/${encodeURIComponent(conversationId)}/messages`,
      {
        params: {
          ...(options.beforePosition !== null && options.beforePosition !== undefined
            ? { before_position: options.beforePosition }
            : {}),
          ...(options.limit ? { limit: options.limit } : {}),
        },
        signal: options.signal,
      },
    )
    const candidate = asRecord(data)
    const rawItems = Array.isArray(candidate?.items)
      ? candidate.items
      : (Array.isArray(candidate?.messages) ? candidate.messages : null)
    if (!rawItems) {
      throw new ChatAPIError('The message list returned an invalid response.', {
        code: 'INVALID_MESSAGE_LIST_RESPONSE',
      })
    }
    const items = sortServerMessages(
      rawItems
        .map(normalizeServerMessage)
        .filter((message): message is ChatServerMessage => message !== null),
    )
    const nextBeforePosition = optionalNonNegativeInteger(
      candidate?.next_before_position ?? candidate?.nextBeforePosition,
    ) ?? null
    return {
      items,
      nextBeforePosition,
      hasMore: optionalBoolean(candidate?.has_more ?? candidate?.hasMore)
        ?? nextBeforePosition !== null,
    }
  } catch (error) {
    if (options.signal?.aborted || isAbortError(error)) throw abortError()
    throw normalizeRequestError(error, 'Unable to load conversation messages.')
  }
}

export async function searchChatConversations(
  request: SearchChatConversationsRequest,
  signal?: AbortSignal,
): Promise<ChatConversationPage> {
  try {
    const { data } = await apiClient.post<unknown>('/chat/conversations/search', {
      query: request.query,
      ...(request.cursor ? { cursor: request.cursor } : {}),
      ...(request.limit ? { limit: request.limit } : {}),
    }, { signal })
    return normalizeConversationPage(data)
  } catch (error) {
    if (signal?.aborted || isAbortError(error)) throw abortError()
    throw normalizeRequestError(error, 'Unable to search conversations.')
  }
}

export async function patchChatConversation(
  conversationId: string,
  request: PatchChatConversationRequest,
  signal?: AbortSignal,
): Promise<ChatServerConversation> {
  try {
    const { data } = await apiClient.patch<unknown>(
      `/chat/conversations/${encodeURIComponent(conversationId)}`,
      {
        revision: request.revision,
        ...(request.title !== undefined ? { title: request.title } : {}),
        ...(request.model !== undefined ? { model: request.model } : {}),
      },
      { signal },
    )
    const conversation = normalizeServerConversation(
      asRecord(data)?.conversation ?? data,
    )
    if (!conversation) {
      throw new ChatAPIError('The updated conversation returned an invalid response.', {
        code: 'INVALID_CONVERSATION_RESPONSE',
      })
    }
    return conversation
  } catch (error) {
    if (signal?.aborted || isAbortError(error)) throw abortError()
    throw normalizeRequestError(error, 'Unable to update the conversation.')
  }
}

export async function deleteChatConversation(
  conversationId: string,
  request: DeleteChatConversationRequest,
  signal?: AbortSignal,
): Promise<void> {
  try {
    await apiClient.delete(`/chat/conversations/${encodeURIComponent(conversationId)}`, {
      data: { revision: request.revision },
      signal,
    })
  } catch (error) {
    if (signal?.aborted || isAbortError(error)) throw abortError()
    throw normalizeRequestError(error, 'Unable to delete the conversation.')
  }
}

function normalizeSyncChange(value: unknown): ChatSyncChange | null {
  const candidate = asRecord(value)
  const rawType = nonEmptyString(candidate?.type)
  const type = rawType === 'deleted' || rawType === 'tombstone' ? 'delete' : rawType
  if (!candidate || (type !== 'upsert' && type !== 'delete')) return null
  const conversationValue = candidate.conversation
  const conversation = conversationValue ? normalizeServerConversation(conversationValue) : null
  const conversationId = nonEmptyString(
    candidate.conversation_id ?? candidate.conversationId,
  ) ?? conversation?.id
  const version = optionalNonNegativeInteger(candidate.version)
  if (!conversationId || version === undefined) return null
  return {
    type,
    version,
    conversationId,
    ...(conversation ? { conversation } : {}),
    ...(optionalTimestamp(candidate.deleted_at ?? candidate.deletedAt) !== undefined
      ? { deletedAt: optionalTimestamp(candidate.deleted_at ?? candidate.deletedAt) }
      : {}),
  }
}

export async function getChatSync(
  options: { afterVersion?: number; cursor?: string | null; signal?: AbortSignal } = {},
): Promise<ChatSyncPage> {
  try {
    const { data } = await apiClient.get<unknown>('/chat/sync', {
      params: {
        after_version: options.afterVersion ?? 0,
        ...(options.cursor ? { cursor: options.cursor } : {}),
      },
      signal: options.signal,
    })
    const candidate = asRecord(data)
    if (!candidate || !Array.isArray(candidate.changes)) {
      throw new ChatAPIError('The chat sync endpoint returned an invalid response.', {
        code: 'INVALID_CHAT_SYNC_RESPONSE',
      })
    }
    const changes = candidate.changes
      .map(normalizeSyncChange)
      .filter((change): change is ChatSyncChange => change !== null)
    const latestVersion = optionalNonNegativeInteger(
      candidate.latest_version ?? candidate.latestVersion,
    ) ?? Math.max(options.afterVersion ?? 0, ...changes.map(({ version }) => version))
    const nextCursor = nonEmptyString(candidate.next_cursor ?? candidate.nextCursor)
    return {
      changes,
      latestVersion,
      nextCursor,
      hasMore: optionalBoolean(candidate.has_more ?? candidate.hasMore) ?? Boolean(nextCursor),
    }
  } catch (error) {
    if (options.signal?.aborted || isAbortError(error)) throw abortError()
    throw normalizeRequestError(error, 'Unable to synchronize chat history.')
  }
}

export async function getChatAttempt(
  attemptId: string,
  signal?: AbortSignal,
): Promise<ChatAttempt> {
  try {
    const { data } = await apiClient.get<unknown>(
      `/chat/attempts/${encodeURIComponent(attemptId)}`,
      { signal },
    )
    const candidate = asRecord(asRecord(data)?.attempt ?? data)
    const normalizedAttemptId = nonEmptyString(candidate?.attempt_id ?? candidate?.attemptId)
      ?? attemptId
    const conversationId = nonEmptyString(
      candidate?.conversation_id ?? candidate?.conversationId,
    )
    const assistantMessageValue = candidate?.assistant_message ?? candidate?.assistantMessage
    const assistantMessage = assistantMessageValue
      ? normalizeServerMessage(assistantMessageValue)
      : null
    const assistantMessageId = nonEmptyString(
      candidate?.assistant_message_id ?? candidate?.assistantMessageId,
    ) ?? assistantMessage?.id
    const rawStatus = nonEmptyString(candidate?.status)
    const status: ChatAttemptStatus | null =
      rawStatus === 'interrupted'
        ? 'stopped'
        : rawStatus === 'processing'
      || rawStatus === 'completed'
      || rawStatus === 'failed'
      || rawStatus === 'stopped'
        ? rawStatus
        : null
    if (!candidate || !conversationId || !assistantMessageId || !status) {
      throw new ChatAPIError('The chat attempt returned an invalid response.', {
        code: 'INVALID_CHAT_ATTEMPT_RESPONSE',
      })
    }
    const receiptId = nonEmptyString(candidate.receipt_id ?? candidate.receiptId)
    const failureCode = nonEmptyString(candidate.failure_code ?? candidate.failureCode)
    const failureReason = nonEmptyString(candidate.failure_reason ?? candidate.failureReason)
    const updatedAt = optionalTimestamp(candidate.updated_at ?? candidate.updatedAt)
    return {
      attemptId: normalizedAttemptId,
      conversationId,
      assistantMessageId,
      status,
      ...(assistantMessage ? { assistantMessage } : {}),
      ...(receiptId ? { receiptId } : {}),
      ...(failureCode ? { failureCode } : {}),
      ...(failureReason ? { failureReason } : {}),
      ...(updatedAt !== undefined ? { updatedAt } : {}),
    }
  } catch (error) {
    if (signal?.aborted || isAbortError(error)) throw abortError()
    throw normalizeRequestError(error, 'Unable to restore the chat attempt.')
  }
}

async function responseError(response: Response): Promise<ChatAPIError> {
  const requestId = response.headers.get('X-Request-Id')
    ?? response.headers.get('X-Request-ID')
    ?? ''
  let raw = ''
  try {
    raw = await response.text()
  } catch (error) {
    if (isAbortError(error)) throw error
  }

  let payload: unknown = null
  if (raw) {
    try {
      payload = JSON.parse(raw)
    } catch {
      payload = raw
    }
  }
  return createErrorFromPayload(
    payload,
    response.statusText || `HTTP ${response.status}`,
    { status: response.status, code: response.status, requestId },
  )
}

async function postCompletion(
  request: ChatCompletionRequest,
  accessToken: string,
  requestId: string,
  attemptId: string,
  signal?: AbortSignal,
): Promise<Response> {
  return fetch(buildApiUrl('/chat/completions'), {
    method: 'POST',
    credentials: 'include',
    headers: {
      Accept: 'text/event-stream',
      'Accept-Language': getLocale(),
      Authorization: `Bearer ${accessToken}`,
      'Content-Type': 'application/json',
      'X-Request-ID': requestId,
      'X-Chat-Attempt-ID': attemptId,
    },
    body: JSON.stringify({
      conversation_id: request.conversationId,
      model: request.model.trim(),
      ...(request.reasoningEffort
        ? { reasoning_effort: request.reasoningEffort }
        : {}),
      expected_head_message_id: request.expectedHeadMessageId,
      ...(request.userMessage
        ? {
            user_message: {
              id: request.userMessage.id,
              content: request.userMessage.content,
              ...(request.userMessage.attachmentIds?.length
                ? { attachment_ids: request.userMessage.attachmentIds }
                : {}),
            },
          }
        : {}),
      assistant_message_id: request.assistantMessageId,
      ...(request.retryOfMessageId
        ? { retry_of_message_id: request.retryOfMessageId }
        : {}),
    }),
    signal,
  })
}

type AuthSessionIdentity = AuthSessionInvalidationExpectation

function authSessionChangedError(requestId: string): ChatAPIError {
  return new ChatAPIError('Authentication session changed. Please retry the request.', {
    status: 409,
    code: 'AUTH_SESSION_CHANGED',
    requestId,
  })
}

function isCurrentAuthSession(expected: AuthSessionIdentity): boolean {
  const current = authSession.getSnapshot()
  return authSession.getGeneration() === expected.generation
    && current.accessToken === expected.accessToken
    && current.refreshToken === expected.refreshToken
    && (current.user?.id ?? null) === expected.userId
}

function invalidateExpiredSession(
  expected: AuthSessionIdentity,
  requestId: string,
): ChatAPIError {
  if (!authSession.invalidate(expected)) {
    return authSessionChangedError(requestId)
  }

  sessionStorage.setItem('auth_expired', '1')
  if (!window.location.pathname.includes('/login')) {
    window.location.href = '/login'
  }

  return new ChatAPIError('Session expired. Please log in again.', {
    status: 401,
    code: 'TOKEN_REFRESH_FAILED',
    requestId,
  })
}

function refreshFailureStatus(error: unknown): number {
  if (axios.isAxiosError(error)) return error.response?.status ?? 0
  const status = asRecord(error)?.status
  return typeof status === 'number' && Number.isFinite(status) ? status : 0
}

function shouldInvalidateAfterRefreshFailure(error: unknown): boolean {
  if (axios.isAxiosError(error)) {
    const status = error.response?.status
    return status === 400 || status === 401 || status === 403
  }

  const status = asRecord(error)?.status
  if (typeof status === 'number') return status === 400 || status === 401 || status === 403

  // A successful refresh response with an invalid payload is not retryable
  // with the same token family.
  return true
}

async function cancelResponseBody(response: Response): Promise<void> {
  try {
    await response.body?.cancel()
  } catch {
    // An error response may not expose a cancellable body.
  }
}

export async function streamChatCompletion(
  request: ChatCompletionRequest,
  handlers: ChatCompletionStreamHandlers = {},
  options: ChatCompletionStreamOptions = {},
): Promise<ChatCompletionStreamResult> {
  if (!request.model.trim()) {
    throw new ChatAPIError('A chat model is required.', { code: 'MODEL_REQUIRED' })
  }
  if (!request.conversationId.trim() || !request.assistantMessageId.trim()) {
    throw new ChatAPIError('Conversation and assistant message IDs are required.', {
      code: 'CHAT_HISTORY_IDS_REQUIRED',
    })
  }
  const hasUserMessage = Boolean(
    request.userMessage?.id.trim()
    && (
      request.userMessage.content.trim()
      || request.userMessage.attachmentIds?.some((id) => id.trim())
    ),
  )
  const hasRetryMessage = Boolean(request.retryOfMessageId?.trim())
  if (hasUserMessage === hasRetryMessage) {
    throw new ChatAPIError('Provide either a user message or a retry target.', {
      code: 'INVALID_CHAT_HISTORY_ENVELOPE',
    })
  }

  const { signal } = options
  const requestId = options.requestId?.trim() || createChatRequestId()
  const attemptId = options.attemptId?.trim() || createChatAttemptId()
  throwIfAborted(signal)

  try {
    let session = authSession.getSnapshot()
    const sessionGeneration = authSession.getGeneration()
    let sessionIdentity: AuthSessionIdentity = {
      generation: sessionGeneration,
      accessToken: session.accessToken,
      refreshToken: session.refreshToken,
      userId: session.user?.id ?? null,
    }
    if (!session.accessToken) {
      throw new ChatAPIError('Authentication is required.', {
        status: 401,
        code: 'UNAUTHENTICATED',
      })
    }

    let response = await postCompletion(request, session.accessToken, requestId, attemptId, signal)
    if (response.status === 401) {
      await cancelResponseBody(response)
      throwIfAborted(signal)
      if (!isCurrentAuthSession(sessionIdentity)) {
        throw authSessionChangedError(requestId)
      }
      if (!session.refreshToken) {
        throw invalidateExpiredSession(sessionIdentity, requestId)
      }

      try {
        session = await refreshAuthSession()
      } catch (refreshError) {
        if (signal?.aborted || isAbortError(refreshError)) throw abortError()
        if (isAuthSessionChangedError(refreshError)) {
          throw authSessionChangedError(requestId)
        }
        if (shouldInvalidateAfterRefreshFailure(refreshError)) {
          throw invalidateExpiredSession(sessionIdentity, requestId)
        }
        throw new ChatAPIError('Token refresh is temporarily unavailable. Please try again.', {
          status: refreshFailureStatus(refreshError),
          code: 'TOKEN_REFRESH_TEMPORARILY_UNAVAILABLE',
          requestId,
        })
      }

      throwIfAborted(signal)
      const refreshedIdentity: AuthSessionIdentity = {
        generation: sessionGeneration,
        accessToken: session.accessToken,
        refreshToken: session.refreshToken,
        userId: session.user?.id ?? null,
      }
      if (
        !session.accessToken
        || refreshedIdentity.userId !== sessionIdentity.userId
        || !isCurrentAuthSession(refreshedIdentity)
      ) {
        if (!session.accessToken && isCurrentAuthSession(sessionIdentity)) {
          throw invalidateExpiredSession(sessionIdentity, requestId)
        }
        throw authSessionChangedError(requestId)
      }
      sessionIdentity = refreshedIdentity
      response = await postCompletion(request, session.accessToken, requestId, attemptId, signal)
      if (response.status === 401) {
        await cancelResponseBody(response)
        throwIfAborted(signal)
        throw invalidateExpiredSession(sessionIdentity, requestId)
      }
    }

    // X-Client-Request-ID is present on every response for request tracing,
    // including validation failures that happen before a chat attempt exists.
    // Only the dedicated receipt header proves that the server persisted (or
    // replayed) the attempt and that attachment drafts may be committed.
    const receiptId = nonEmptyString(response.headers.get('X-Chat-Receipt-ID'))
    if (receiptId) {
      handlers.onReceiptId?.(receiptId)
      handlers.onAccepted?.()
    }
    if (!response.ok) throw await responseError(response)
    if (!receiptId) handlers.onAccepted?.()
    if (!response.body) {
      throw new ChatAPIError('The chat response did not include a stream.', {
        status: response.status,
        code: 'EMPTY_STREAM',
      })
    }
    const result = await parseChatCompletionSSE(response.body, handlers, options)
    result.receiptId = receiptId
    if (!result.receivedDone) {
      throw new ChatAPIError('The chat stream ended before it completed.', {
        status: response.status,
        code: 'INCOMPLETE_STREAM',
        requestId: response.headers.get('X-Request-ID') ?? requestId,
      })
    }
    return result
  } catch (error) {
    if (signal?.aborted || isAbortError(error)) throw abortError()
    if (error instanceof ChatAPIError) throw error
    if (isAuthSessionChangedError(error)) {
      throw authSessionChangedError(requestId)
    }
    throw new ChatAPIError(
      error instanceof Error ? error.message : 'Unable to connect to the chat service.',
      { code: 'NETWORK_ERROR' },
    )
  }
}

export const chatAPI = {
  getModels: getChatModels,
  getReceipt: getChatReceipt,
  pollReceipt: pollChatReceipt,
  createConversation: createChatConversation,
  listConversations: listChatConversations,
  getConversation: getChatConversation,
  getConversationMessages: getChatConversationMessages,
  searchConversations: searchChatConversations,
  patchConversation: patchChatConversation,
  deleteConversation: deleteChatConversation,
  getSync: getChatSync,
  getAttempt: getChatAttempt,
  streamCompletion: streamChatCompletion,
}

export default chatAPI
