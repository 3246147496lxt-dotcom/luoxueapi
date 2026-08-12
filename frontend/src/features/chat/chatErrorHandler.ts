import type { ChatMessage } from '@/types/chat'

export type ChatErrorMessageKey =
  | 'chat.errors.requestFailed'
  | 'chat.errors.insufficientBalance'
  | 'chat.errors.modelUnavailable'
  | 'chat.errors.reasoningUnavailable'
  | 'chat.errors.serviceUnavailable'
  | 'chat.errors.network'
  | 'chat.errors.timeout'
  | 'chat.errors.rateLimited'
  | 'chat.errors.sessionExpired'
  | 'chat.errors.sessionChanged'
  | 'chat.errors.permissionDenied'
  | 'chat.errors.invalidRequest'
  | 'chat.errors.resourceUnavailable'
  | 'chat.errors.conflict'
  | 'chat.errors.attemptAlreadySubmitted'

export interface ChatErrorPresentation {
  code: string
  messageKey: ChatErrorMessageKey
  retryable: boolean
}

export interface ChatErrorDebugDetails {
  status: number
  code: string
  reason: unknown
  requestId: string
  rawMessage: string
}

type ErrorLike = {
  status?: unknown
  code?: unknown
  reason?: unknown
  requestId?: unknown
  message?: unknown
}

const MODEL_UNAVAILABLE_CODES = new Set([
  'CHAT_MODEL_NOT_AVAILABLE',
  'MODEL_NOT_AVAILABLE',
  'MODEL_NOT_FOUND',
])

const AUTHENTICATION_CODES = new Set([
  'UNAUTHENTICATED',
  'UNAUTHORIZED',
  'TOKEN_REFRESH_FAILED',
  'AUTHENTICATION_REQUIRED',
])

const CONFLICT_CODES = new Set([
  'CHAT_ATTEMPT_CONFLICT',
  'CHAT_HISTORY_CONFLICT',
  'CHAT_HISTORY_REVISION_CONFLICT',
  'CHAT_TURN_IN_PROGRESS',
])

const INVALID_REQUEST_CODES = new Set([
  'INVALID_CHAT_REQUEST',
  'REQUEST_TOO_LARGE',
  'UNSUPPORTED_MEDIA_TYPE',
  'MODEL_REQUIRED',
  'CHAT_ATTACHMENT_INVALID',
  'CHAT_ATTACHMENT_LIMIT',
  'CHAT_ATTACHMENT_TOO_LARGE',
  'CHAT_HISTORY_IDS_REQUIRED',
  'INVALID_CHAT_HISTORY_ENVELOPE',
  'CHAT_ATTEMPT_ID_REQUIRED',
  'CHAT_ATTEMPT_ID_INVALID',
  'CHAT_HISTORY_INVALID',
])

const RESOURCE_UNAVAILABLE_CODES = new Set([
  'CHAT_ATTACHMENT_NOT_FOUND',
  'CHAT_CONVERSATION_NOT_FOUND',
  'CHAT_ATTEMPT_NOT_FOUND',
])

function asErrorLike(value: unknown): ErrorLike {
  return value !== null && typeof value === 'object' ? value as ErrorLike : {}
}

function nonEmptyString(value: unknown): string {
  return typeof value === 'string' ? value.trim() : ''
}

function finiteStatus(value: unknown): number {
  return typeof value === 'number' && Number.isFinite(value) ? Math.trunc(value) : 0
}

function normalizedNamedCode(value: string): string {
  return value.trim().toUpperCase().replace(/[^A-Z0-9]+/g, '_').replace(/^_+|_+$/g, '')
}

function statusFromLegacyMessage(message: string): number {
  const normalized = message.trim().toLowerCase()
  if (/^(401\s+)?unauthorized$/.test(normalized)) return 401
  if (/^(403\s+)?forbidden$/.test(normalized)) return 403
  if (/^(408\s+)?request timeout$/.test(normalized)) return 408
  if (/^(429\s+)?too many requests$/.test(normalized)) return 429
  if (/^(500\s+)?internal server error$/.test(normalized)) return 500
  if (/^(502\s+)?bad gateway$/.test(normalized)) return 502
  if (/^(503\s+)?service unavailable$/.test(normalized)) return 503
  if (/^(504\s+)?gateway timeout$/.test(normalized)) return 504
  return 0
}

function statusFromCode(code: string): number {
  const match = code.match(/^HTTP_(\d{3})$/)
  return match ? Number(match[1]) : 0
}

function preferredCode(source: ErrorLike, rawMessage: string): string {
  const rawCode = source.code
  const rawReason = nonEmptyString(source.reason)
  const status = finiteStatus(source.status)

  if (typeof rawCode === 'string' && rawCode.trim()) {
    const normalized = normalizedNamedCode(rawCode)
    if (normalized && !/^\d{3}$/.test(normalized)) return normalized
  }
  // Backend business reasons are more specific than a numeric HTTP code.
  if (rawReason && !/\s/.test(rawReason)) return normalizedNamedCode(rawReason)
  if (typeof rawCode === 'string' && /^\d{3}$/.test(rawCode.trim())) {
    return `HTTP_${rawCode.trim()}`
  }
  if (typeof rawCode === 'number' && Number.isFinite(rawCode)) {
    const numericCode = Math.trunc(rawCode)
    if (numericCode >= 100 && numericCode <= 599) return `HTTP_${numericCode}`
  }
  if (status >= 100 && status <= 599) return `HTTP_${status}`

  const legacyStatus = statusFromLegacyMessage(rawMessage)
  return legacyStatus ? `HTTP_${legacyStatus}` : 'CHAT_REQUEST_FAILED'
}

function presentationFor(code: string, status: number): ChatErrorPresentation {
  if (code === 'INSUFFICIENT_BALANCE') {
    return { code, messageKey: 'chat.errors.insufficientBalance', retryable: false }
  }
  if (code === 'CHAT_ATTEMPT_ALREADY_SUBMITTED') {
    return { code, messageKey: 'chat.errors.attemptAlreadySubmitted', retryable: false }
  }
  if (code === 'CHAT_SETTLEMENT_FAILED') {
    return { code, messageKey: 'chat.errors.requestFailed', retryable: false }
  }
  if (code === 'CHAT_REASONING_EFFORT_NOT_AVAILABLE') {
    return { code, messageKey: 'chat.errors.reasoningUnavailable', retryable: false }
  }
  if (code === 'CHAT_MODEL_VISION_UNAVAILABLE') {
    return { code, messageKey: 'chat.errors.modelUnavailable', retryable: false }
  }
  if (MODEL_UNAVAILABLE_CODES.has(code)) {
    return { code, messageKey: 'chat.errors.modelUnavailable', retryable: false }
  }
  if (code === 'CHAT_CATALOG_UNAVAILABLE') {
    return { code, messageKey: 'chat.errors.modelUnavailable', retryable: true }
  }
  if (code === 'AUTH_SESSION_CHANGED') {
    return { code, messageKey: 'chat.errors.sessionChanged', retryable: true }
  }
  if (AUTHENTICATION_CODES.has(code) || status === 401) {
    return { code, messageKey: 'chat.errors.sessionExpired', retryable: false }
  }
  if (code === 'FORBIDDEN' || code.includes('PERMISSION_DENIED') || status === 403) {
    return { code, messageKey: 'chat.errors.permissionDenied', retryable: false }
  }
  if (code === 'NETWORK_ERROR') {
    return { code, messageKey: 'chat.errors.network', retryable: true }
  }
  if (code.includes('TIMEOUT') || status === 408 || status === 504) {
    return { code, messageKey: 'chat.errors.timeout', retryable: true }
  }
  if (
    code.includes('RATE_LIMIT')
    || code.includes('TOO_MANY_REQUESTS')
    || status === 429
  ) {
    return { code, messageKey: 'chat.errors.rateLimited', retryable: true }
  }
  if (CONFLICT_CODES.has(code) || status === 409) {
    return { code, messageKey: 'chat.errors.conflict', retryable: false }
  }
  if (RESOURCE_UNAVAILABLE_CODES.has(code) || status === 404) {
    return { code, messageKey: 'chat.errors.resourceUnavailable', retryable: false }
  }
  if (
    INVALID_REQUEST_CODES.has(code)
    || code.includes('INVALID_REQUEST')
    || status === 400
    || status === 405
    || status === 413
    || status === 415
    || status === 422
  ) {
    return { code, messageKey: 'chat.errors.invalidRequest', retryable: false }
  }
  if (
    code === 'INCOMPLETE_STREAM'
    || code === 'EMPTY_STREAM'
    || code === 'STREAM_ERROR'
    || code === 'INVALID_STREAM_EVENT'
    || code === 'TOKEN_REFRESH_TEMPORARILY_UNAVAILABLE'
    || code === 'DATABASE_UNAVAILABLE'
    || code === 'BILLING_SERVICE_ERROR'
    || code.includes('UPSTREAM')
    || code.endsWith('_UNAVAILABLE')
    || (status >= 500 && status <= 599)
  ) {
    return { code, messageKey: 'chat.errors.serviceUnavailable', retryable: true }
  }
  // Unknown persisted failures no longer carry an HTTP status. Do not invite a
  // potentially billable retry unless the code is explicitly known to be safe.
  return { code, messageKey: 'chat.errors.requestFailed', retryable: false }
}

function describeSource(source: ErrorLike): ChatErrorPresentation {
  const rawMessage = nonEmptyString(source.message)
  const code = preferredCode(source, rawMessage)
  const status = finiteStatus(source.status) || statusFromCode(code) || statusFromLegacyMessage(rawMessage)
  return presentationFor(code, status)
}

export function describeChatError(error: unknown): ChatErrorPresentation {
  if (typeof error === 'string') return describeSource({ message: error })
  return describeSource(asErrorLike(error))
}

export function describeChatMessageError(
  message: Pick<ChatMessage, 'errorCode' | 'errorMessage'>,
): ChatErrorPresentation {
  return describeSource({
    code: message.errorCode,
    message: message.errorMessage,
  })
}

export function chatErrorDebugDetails(error: unknown): ChatErrorDebugDetails {
  const source = asErrorLike(error)
  const rawMessage = nonEmptyString(source.message)
  const presentation = describeSource(source)
  return {
    status: finiteStatus(source.status) || statusFromCode(presentation.code),
    code: presentation.code,
    reason: source.reason,
    requestId: nonEmptyString(source.requestId),
    rawMessage,
  }
}

export function logChatCompletionError(
  error: unknown,
  context: { conversationId: string; messageId: string; attemptId: string },
  presentation = describeChatError(error),
): void {
  if (!import.meta.env.DEV) return
  console.error('[Chat] 回答生成失败', {
    ...context,
    ...chatErrorDebugDetails(error),
    messageKey: presentation.messageKey,
    retryable: presentation.retryable,
  })
}
