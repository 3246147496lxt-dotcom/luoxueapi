import { describe, expect, it, vi } from 'vitest'
import {
  describeChatError,
  describeChatMessageError,
  logChatCompletionError,
} from '../chatErrorHandler'

describe('Chat error handler', () => {
  it.each([
    [{ status: 401, message: 'Unauthorized' }, 'HTTP_401', 'chat.errors.sessionExpired', false],
    [{ status: 403, message: 'Forbidden' }, 'HTTP_403', 'chat.errors.permissionDenied', false],
    [{ status: 402, message: 'Payment Required' }, 'HTTP_402', 'chat.errors.requestFailed', false],
    [{ status: 502, message: 'Bad Gateway' }, 'HTTP_502', 'chat.errors.serviceUnavailable', true],
    [{ status: 429, message: 'Too Many Requests' }, 'HTTP_429', 'chat.errors.rateLimited', true],
    [{ status: 504, message: 'Gateway Timeout' }, 'HTTP_504', 'chat.errors.timeout', true],
    [{ code: 'NETWORK_ERROR', message: 'Failed to fetch' }, 'NETWORK_ERROR', 'chat.errors.network', true],
  ] as const)(
    'maps %# to a safe presentation',
    (error, code, messageKey, retryable) => {
      expect(describeChatError(error)).toEqual({ code, messageKey, retryable })
    },
  )

  it('gives business error codes priority over their HTTP status', () => {
    expect(describeChatError({
      status: 403,
      code: 'INSUFFICIENT_BALANCE',
      message: 'Forbidden',
    })).toEqual({
      code: 'INSUFFICIENT_BALANCE',
      messageKey: 'chat.errors.insufficientBalance',
      retryable: false,
    })

    expect(describeChatError({
      status: 503,
      code: 'CHAT_MODEL_NOT_AVAILABLE',
      message: 'Bad Gateway',
    })).toEqual({
      code: 'CHAT_MODEL_NOT_AVAILABLE',
      messageKey: 'chat.errors.modelUnavailable',
      retryable: false,
    })
  })

  it('uses a backend business reason before a numeric HTTP code', () => {
    expect(describeChatError({
      status: 403,
      code: 403,
      reason: 'INSUFFICIENT_BALANCE',
      message: 'Forbidden',
    })).toEqual({
      code: 'INSUFFICIENT_BALANCE',
      messageKey: 'chat.errors.insufficientBalance',
      retryable: false,
    })
  })

  it('maps PRO_REASONING_UNAVAILABLE to the dedicated safe presentation', () => {
    const expected = {
      code: 'PRO_REASONING_UNAVAILABLE',
      messageKey: 'chat.errors.proReasoningUnavailable' as const,
      retryable: false,
    }

    expect(describeChatError({
      status: 400,
      code: 'PRO_REASONING_UNAVAILABLE',
      message: 'raw backend detail',
    })).toEqual(expected)
    expect(describeChatMessageError({
      errorCode: 'PRO_REASONING_UNAVAILABLE',
      errorMessage: 'raw persisted detail',
    })).toEqual(expected)
  })

  it.each([
    ['Unauthorized', 'chat.errors.sessionExpired', false],
    ['Forbidden', 'chat.errors.permissionDenied', false],
    ['Bad Gateway', 'chat.errors.serviceUnavailable', true],
  ] as const)('sanitizes legacy persisted message %s', (raw, messageKey, retryable) => {
    const presentation = describeChatMessageError({ errorMessage: raw })
    expect(presentation.messageKey).toBe(messageKey)
    expect(presentation.retryable).toBe(retryable)
    expect(JSON.stringify(presentation)).not.toContain(raw)
  })

  it('never returns an unknown raw error message to the UI', () => {
    const raw = 'upstream socket 10.0.0.8:443 reset by peer'
    const presentation = describeChatError(new Error(raw))

    expect(presentation).toEqual({
      code: 'CHAT_REQUEST_FAILED',
      messageKey: 'chat.errors.requestFailed',
      retryable: false,
    })
    expect(JSON.stringify(presentation)).not.toContain(raw)
  })

  it.each([
    ['CHAT_ATTACHMENT_INVALID', 400, 'chat.errors.invalidRequest'],
    ['CHAT_ATTACHMENT_LIMIT', 400, 'chat.errors.invalidRequest'],
    ['CHAT_ATTACHMENT_TOO_LARGE', 413, 'chat.errors.invalidRequest'],
    ['CHAT_ATTACHMENT_NOT_FOUND', 404, 'chat.errors.resourceUnavailable'],
    ['CHAT_MODEL_VISION_UNAVAILABLE', 400, 'chat.errors.modelUnavailable'],
    ['CHAT_SETTLEMENT_FAILED', 500, 'chat.errors.requestFailed'],
  ] as const)(
    'keeps persisted %s non-retryable with or without HTTP status',
    (code, status, messageKey) => {
      const persisted = describeChatMessageError({
        errorCode: code,
        errorMessage: 'raw persisted failure',
      })
      const live = describeChatError({
        status,
        code,
        message: 'raw live failure',
      })

      expect(persisted).toEqual({ code, messageKey, retryable: false })
      expect(live).toEqual(persisted)
    },
  )

  it('keeps structured raw diagnostics in console without making them presentation copy', () => {
    const error = Object.assign(new Error('Bad Gateway'), {
      status: 502,
      code: 502,
      reason: 'UPSTREAM_FAILURE',
      requestId: 'request-debug-1',
    })
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined)

    logChatCompletionError(error, {
      conversationId: 'conversation-1',
      messageId: 'message-1',
      attemptId: 'attempt-1',
    })

    expect(consoleError).toHaveBeenCalledTimes(1)
    expect(consoleError.mock.calls[0]).toHaveLength(2)
    expect(consoleError.mock.calls[0]?.[0]).toBe('[Chat] 回答生成失败')
    expect(consoleError.mock.calls[0]?.[1]).toEqual(
      expect.objectContaining({
        status: 502,
        code: 'UPSTREAM_FAILURE',
        reason: 'UPSTREAM_FAILURE',
        requestId: 'request-debug-1',
        rawMessage: 'Bad Gateway',
        messageKey: 'chat.errors.serviceUnavailable',
        retryable: true,
      }),
    )
    expect(consoleError.mock.calls[0]).not.toContain(error)
  })
})
