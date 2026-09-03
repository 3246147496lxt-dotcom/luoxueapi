import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { AuthSessionInvalidationExpectation } from '@/auth/authSession'

const mocks = vi.hoisted(() => ({
  getSnapshot: vi.fn(),
  getGeneration: vi.fn(),
  invalidate: vi.fn(),
  isAuthSessionChangedError: vi.fn(),
  refreshAuthSession: vi.fn(),
  apiGet: vi.fn(),
  apiPost: vi.fn(),
  apiPatch: vi.fn(),
  apiDelete: vi.fn(),
}))

vi.mock('@/auth/authSession', () => ({
  authSession: {
    getSnapshot: mocks.getSnapshot,
    getGeneration: mocks.getGeneration,
    invalidate: mocks.invalidate,
  },
  isAuthSessionChangedError: mocks.isAuthSessionChangedError,
}))

vi.mock('@/auth/authRefresh', () => ({
  refreshAuthSession: mocks.refreshAuthSession,
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get: mocks.apiGet,
    post: mocks.apiPost,
    patch: mocks.apiPatch,
    delete: mocks.apiDelete,
  },
}))

import {
  ChatAPIError,
  buildChatReasoningPayload,
  createChatConversation,
  deleteChatAttachment,
  deleteChatConversation,
  getChatAttempt,
  getChatConversationMessages,
  getChatCapabilities,
  getChatSync,
  getChatReceipt,
  getChatModels,
  getChatAttachmentContent,
  listChatConversations,
  parseChatCompletionSSE,
  normalizeChatAttachment,
  patchChatConversation,
  pollChatReceipt,
  searchChatConversations,
  stopChatAttempt,
  streamChatCompletion,
  transcribeChatAudio,
  uploadChatAttachment,
} from '@/api/chat'

const encoder = new TextEncoder()
const initialSession = {
  accessToken: 'jwt-one',
  refreshToken: 'refresh-one',
  expiresAt: null,
  user: { id: 7 },
}
const refreshedSession = {
  accessToken: 'jwt-two',
  refreshToken: 'refresh-two',
  expiresAt: null,
  user: { id: 7 },
}

describe('buildChatReasoningPayload', () => {
  it('keeps standard effort and Pro mode mutually exclusive', () => {
    expect(buildChatReasoningPayload({
      reasoningMode: 'standard',
      reasoningEffort: 'xhigh',
    })).toEqual({ mode: 'standard', effort: 'xhigh', summary: 'auto' })

    expect(buildChatReasoningPayload({
      reasoningMode: 'pro',
      reasoningEffort: 'xhigh',
    })).toEqual({ mode: 'pro', summary: 'auto' })
  })
})

function sessionIdentity(
  session: {
    accessToken: string | null
    refreshToken: string | null
    user: { id: number } | null
  },
): AuthSessionInvalidationExpectation {
  return {
    generation: 'generation-one',
    accessToken: session.accessToken,
    refreshToken: session.refreshToken,
    userId: session.user?.id ?? null,
  }
}

function streamFromBytes(bytes: Uint8Array, cuts: number[] = []): ReadableStream<Uint8Array> {
  const chunks: Uint8Array[] = []
  let start = 0
  for (const end of cuts) {
    chunks.push(bytes.slice(start, end))
    start = end
  }
  chunks.push(bytes.slice(start))

  return new ReadableStream<Uint8Array>({
    start(controller) {
      for (const chunk of chunks) controller.enqueue(chunk)
      controller.close()
    },
  })
}

function openStreamFromSource(
  source: string,
  cancel: (reason?: unknown) => void | PromiseLike<void>,
): ReadableStream<Uint8Array> {
  return new ReadableStream<Uint8Array>({
    start(controller) {
      controller.enqueue(encoder.encode(source))
    },
    cancel,
  })
}

function successfulStreamResponse(
  source: string,
  headers: HeadersInit = {},
): Response {
  return {
    ok: true,
    status: 200,
    statusText: 'OK',
    headers: new Headers(headers),
    body: streamFromBytes(encoder.encode(source)),
  } as Response
}

function unauthorizedResponse(
  cancel = vi.fn().mockResolvedValue(undefined),
  headers: HeadersInit = {},
): Response {
  return {
    ok: false,
    status: 401,
    statusText: 'Unauthorized',
    headers: new Headers(headers),
    body: { cancel },
  } as unknown as Response
}

function historyCompletionRequest(model = 'gpt-5') {
  return {
    conversationId: 'conversation-1',
    model,
    expectedHeadMessageId: 'assistant-previous',
    userMessage: { id: 'user-1', content: 'Hi' },
    assistantMessageId: 'assistant-1',
  }
}

describe('parseChatCompletionSSE', () => {
  it('handles arbitrary byte chunks, CRLF, multi-line data, usage, and DONE', async () => {
    const source = [
      ': keepalive\r\n',
      'data: {"choices":[{"index":0,\r\n',
      'data: "delta":{"content":"你"},"finish_reason":null}]}\r\n\r\n',
      'data: {"choices":[{"index":0,"delta":{"content":"好"},"finish_reason":"stop"}],',
      '"usage":{"prompt_tokens":2,"completion_tokens":2,"total_tokens":4}}\r\n\r\n',
      'data: [DONE]\r\n\r\n',
    ].join('')
    const bytes = encoder.encode(source)
    const firstChineseByte = bytes.findIndex((value) => value > 127)
    const stream = streamFromBytes(bytes, [1, 7, 19, firstChineseByte + 1, bytes.length - 3])
    const content: string[] = []
    const finishReasons: string[] = []
    const onDone = vi.fn()

    const result = await parseChatCompletionSSE(stream, {
      onContent: (delta) => content.push(delta),
      onFinish: (reason) => finishReasons.push(reason),
      onDone,
    })

    expect(content).toEqual(['你', '好'])
    expect(finishReasons).toEqual(['stop'])
    expect(result).toEqual({
      receivedDone: true,
      finishReason: 'stop',
      usage: { prompt_tokens: 2, completion_tokens: 2, total_tokens: 4 },
      receiptId: null,
    })
    expect(onDone).toHaveBeenCalledWith(result)
  })

  it('dispatches a trailing event without a final newline or DONE marker', async () => {
    const content: string[] = []
    const source = 'data: {"choices":[{"index":0,"delta":{"content":"tail"},"finish_reason":null}]}'

    const result = await parseChatCompletionSSE(
      streamFromBytes(encoder.encode(source), [2, 11, 37]),
      { onContent: (delta) => content.push(delta) },
    )

    expect(content).toEqual(['tail'])
    expect(result.receivedDone).toBe(false)
  })

  it('parses mixed Chat Completions chunks and private Responses activity envelopes', async () => {
    const envelope = (eventType: string, payload: unknown) => (
      `data: ${JSON.stringify({ source: 'openai_responses', eventType, payload })}\n\n`
    )
    const source = [
      'data: {"choices":[{"index":0,"delta":{"content":"answer ","reasoning_content":"legacy-private"},"finish_reason":null}]}\n\n',
      envelope('response.created', {
        sequence_number: 0,
        response: { id: 'resp-mixed' },
      }),
      envelope('response.output_item.added', {
        sequence_number: 1,
        output_index: 0,
        item: { id: 'reasoning-mixed', type: 'reasoning' },
      }),
      envelope('response.reasoning_summary_part.added', {
        sequence_number: 2,
        item_id: 'reasoning-mixed',
        output_index: 0,
        summary_index: 0,
        part: { type: 'summary_text' },
      }),
      envelope('response.reasoning_summary_text.delta', {
        sequence_number: 3,
        item_id: 'reasoning-mixed',
        output_index: 0,
        summary_index: 0,
        delta: 'Checking ',
      }),
      envelope('response.output_text.delta', {
        sequence_number: 4,
        delta: 'must not enter activity',
      }),
      envelope('response.reasoning_text.delta', {
        sequence_number: 5,
        delta: 'private chain of thought',
      }),
      envelope('response.reasoning_summary_text.done', {
        sequence_number: 6,
        item_id: 'reasoning-mixed',
        output_index: 0,
        summary_index: 0,
        text: 'Checking safely.',
      }),
      envelope('response.completed', {
        sequence_number: 7,
        response: { id: 'resp-mixed' },
      }),
      'data: {"choices":[{"index":0,"delta":{"content":"done"},"finish_reason":"stop"}]}\n\n',
      'data: [DONE]\n\n',
    ].join('')
    const content: string[] = []
    const legacyReasoning: string[] = []
    const events: string[] = []
    const activities: Array<{ status: string; text: string; mode?: string; effort?: string }> = []

    await parseChatCompletionSSE(streamFromBytes(encoder.encode(source)), {
      onContent: (delta) => content.push(delta),
      onReasoningContent: (delta) => legacyReasoning.push(delta),
      onActivityEvent: ({ eventType }) => events.push(eventType),
      onActivity: (activity) => activities.push({
        status: activity.status,
        text: activity.items[0]?.parts[0]?.text ?? '',
        mode: activity.reasoningMode,
        effort: activity.reasoningEffort,
      }),
    }, { reasoningMode: 'pro', reasoningEffort: 'medium' })

    expect(content).toEqual(['answer ', 'done'])
    expect(legacyReasoning).toEqual(['legacy-private'])
    expect(events).toEqual([
      'response.created',
      'response.output_item.added',
      'response.reasoning_summary_part.added',
      'response.reasoning_summary_text.delta',
      'response.reasoning_summary_text.done',
      'response.completed',
    ])
    expect(activities.at(-1)).toEqual({
      status: 'completed',
      text: 'Checking safely.',
      mode: 'pro',
      effort: undefined,
    })
    expect(JSON.stringify(activities)).not.toContain('legacy-private')
    expect(JSON.stringify(activities)).not.toContain('must not enter activity')
    expect(JSON.stringify(activities)).not.toContain('private chain of thought')
  })

  it('flushes batched summary deltas as disconnected when the stream ends without DONE', async () => {
    const envelope = (eventType: string, payload: unknown) => (
      `data: ${JSON.stringify({ source: 'openai_responses', eventType, payload })}\n\n`
    )
    const source = [
      envelope('response.created', {
        sequence_number: 0,
        response: { id: 'resp-disconnected' },
      }),
      envelope('response.output_item.added', {
        sequence_number: 1,
        output_index: 0,
        item: { id: 'reasoning-disconnected', type: 'reasoning' },
      }),
      envelope('response.reasoning_summary_part.added', {
        sequence_number: 2,
        item_id: 'reasoning-disconnected',
        output_index: 0,
        summary_index: 0,
        part: { type: 'summary_text' },
      }),
      envelope('response.reasoning_summary_text.delta', {
        sequence_number: 3,
        item_id: 'reasoning-disconnected',
        output_index: 0,
        summary_index: 0,
        delta: 'batched ',
      }),
      envelope('response.reasoning_summary_text.delta', {
        sequence_number: 4,
        item_id: 'reasoning-disconnected',
        output_index: 0,
        summary_index: 0,
        delta: 'text',
      }),
    ].join('')
    const activities: Array<{ status: string; text: string }> = []

    const result = await parseChatCompletionSSE(
      streamFromBytes(encoder.encode(source)),
      {
        onActivity: (activity) => activities.push({
          status: activity.status,
          text: activity.items[0]?.parts[0]?.text ?? '',
        }),
      },
    )

    expect(result.receivedDone).toBe(false)
    expect(activities.at(-1)).toEqual({
      status: 'disconnected',
      text: 'batched text',
    })
    expect(activities.filter(({ text }) => text !== '')).toHaveLength(1)
  })

  it('safely ignores malformed or unknown private Responses envelopes', async () => {
    const source = [
      'data: {"source":"openai_responses","eventType":"response.reasoning_summary_text.delta","payload":"malformed"}\n\n',
      'data: {"source":"openai_responses","eventType":"response.unknown","payload":{"text":"ignored"}}\n\n',
      'data: [DONE]\n\n',
    ].join('')
    const onActivity = vi.fn()

    await expect(parseChatCompletionSSE(
      streamFromBytes(encoder.encode(source)),
      { onActivity },
    )).resolves.toMatchObject({ receivedDone: true })
    expect(onActivity).not.toHaveBeenCalled()
  })

  it('turns an SSE error event into a structured ChatAPIError', async () => {
    const source = [
      'event: error\r',
      'data: {"error":\n',
      'data: {"message":"余额不足","code":"INSUFFICIENT_BALANCE"}}\r\r\n',
    ].join('')
    const cancel = vi.fn().mockResolvedValue(undefined)

    await expect(parseChatCompletionSSE(openStreamFromSource(source, cancel)))
      .rejects.toMatchObject({
        name: 'ChatAPIError',
        message: '余额不足',
        code: 'INSUFFICIENT_BALANCE',
      })
    expect(cancel).toHaveBeenCalledOnce()
  })

  it('uses an OpenAI error type as the business code', async () => {
    const cancel = vi.fn().mockResolvedValue(undefined)
    const source = [
      'event: error\n',
      'data: {"error":{"message":"余额不足","type":"INSUFFICIENT_BALANCE","code":null}}\n\n',
    ].join('')

    await expect(parseChatCompletionSSE(openStreamFromSource(source, cancel)))
      .rejects.toMatchObject({
        name: 'ChatAPIError',
        message: '余额不足',
        code: 'INSUFFICIENT_BALANCE',
      })
    expect(cancel).toHaveBeenCalledOnce()
  })

  it('cancels an open source when an event cannot be parsed', async () => {
    const cancel = vi.fn().mockResolvedValue(undefined)
    const stream = openStreamFromSource('data: {not-json}\n\n', cancel)

    await expect(parseChatCompletionSSE(stream)).rejects.toMatchObject({
      name: 'ChatAPIError',
      code: 'INVALID_STREAM_EVENT',
    })
    expect(cancel).toHaveBeenCalledOnce()
  })

  it('preserves callback errors when cancelling the open source fails', async () => {
    const callbackError = new Error('chunk handler failed')
    const cancel = vi.fn().mockRejectedValue(new Error('cancel failed'))
    const source = 'data: {"choices":[{"index":0,"delta":{"content":"hello"}}]}\n\n'

    await expect(parseChatCompletionSSE(openStreamFromSource(source, cancel), {
      onChunk: () => {
        throw callbackError
      },
    })).rejects.toBe(callbackError)
    expect(cancel).toHaveBeenCalledOnce()
  })

  it('cancels an open source only once after the DONE marker', async () => {
    const cancel = vi.fn().mockResolvedValue(undefined)

    await expect(parseChatCompletionSSE(
      openStreamFromSource('data: [DONE]\n\n', cancel),
    )).resolves.toMatchObject({ receivedDone: true })
    expect(cancel).toHaveBeenCalledOnce()
  })

  it('does not misclassify AbortError as a stream protocol failure', async () => {
    const abortController = new AbortController()
    let finishRead: ((result: ReadableStreamReadResult<Uint8Array>) => void) | undefined
    const cancel = vi.fn(() => {
      finishRead?.({ done: true, value: undefined })
      return Promise.resolve()
    })
    const stream = {
      getReader: () => ({
        read: () => new Promise<ReadableStreamReadResult<Uint8Array>>((resolve) => {
          finishRead = resolve
        }),
        cancel,
        releaseLock: vi.fn(),
      }),
    } as unknown as ReadableStream<Uint8Array>

    const parsing = parseChatCompletionSSE(stream, {}, { signal: abortController.signal })
    abortController.abort()

    await expect(parsing).rejects.toMatchObject({ name: 'AbortError' })
    expect(cancel).toHaveBeenCalledOnce()
  })
})

describe('chatAPI', () => {
  let originalLocation: Location

  beforeEach(() => {
    vi.resetAllMocks()
    sessionStorage.clear()
    originalLocation = window.location
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { ...originalLocation, pathname: '/chat', href: '/chat' },
      writable: true,
    })
    mocks.getSnapshot.mockReturnValue(initialSession)
    mocks.getGeneration.mockReturnValue('generation-one')
    mocks.invalidate.mockImplementation((expected: AuthSessionInvalidationExpectation) => {
      const current = mocks.getSnapshot()
      return mocks.getGeneration() === expected.generation
        && current.accessToken === expected.accessToken
        && current.refreshToken === expected.refreshToken
        && (current.user?.id ?? null) === expected.userId
    })
    mocks.isAuthSessionChangedError.mockReturnValue(false)
  })

  afterEach(() => {
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: originalLocation,
      writable: true,
    })
    vi.unstubAllGlobals()
  })

  it('returns the model list and balance snapshot after the standard client unwrap', async () => {
    mocks.apiGet.mockResolvedValue({
      data: {
        models: [
          { id: 'gpt-5', display_name: 'GPT-5' },
          { id: '   ' },
        ],
        balance: 12.5,
      },
    })

    await expect(getChatModels()).resolves.toEqual({
      models: [{ id: 'gpt-5', display_name: 'GPT-5' }],
      balance: 12.5,
    })
    expect(mocks.apiGet).toHaveBeenCalledWith('/chat/models')
  })

  it('loads non-blocking Chat capabilities from the endpoint that does not return models', async () => {
    mocks.apiGet.mockResolvedValue({
      data: {
        transcription: {
          enabled: true,
          billing_mode: 'included',
          max_upload_bytes: 8_388_608,
          max_duration_seconds: 120,
          accepted_mime_types: ['audio/webm;codecs=opus', ' ', 'audio/mp4'],
        },
      },
    })

    await expect(getChatCapabilities()).resolves.toEqual({
      transcription: {
        enabled: true,
        billing_mode: 'included',
        max_upload_bytes: 8_388_608,
        max_duration_seconds: 120,
        accepted_mime_types: ['audio/webm;codecs=opus', 'audio/mp4'],
      },
    })
    expect(mocks.apiGet).toHaveBeenCalledWith('/chat/capabilities', { signal: undefined })
  })

  it('normalizes the Sol reasoning slider capability and fails closed for invalid values', async () => {
    mocks.apiGet.mockResolvedValue({
      data: {
        models: [
          {
            id: 'sol',
            supports_reasoning_slider: true,
            supports_responses: true,
            supports_reasoning_summary: true,
            supports_reasoning_pro_mode: true,
            supported_reasoning_efforts: ['low', 'medium', 'invalid', 'xhigh'],
          },
          { id: 'legacy', supportsReasoningSlider: false },
          {
            id: 'spoofed',
            supports_reasoning_slider: 'true',
            supports_responses: 'true',
            supported_reasoning_efforts: 'medium',
          },
        ],
        balance: 2,
      },
    })

    await expect(getChatModels()).resolves.toEqual({
      models: [
        {
          id: 'sol',
          supports_reasoning_slider: true,
          supports_responses: true,
          supports_reasoning_summary: true,
          supports_reasoning_pro_mode: true,
          supported_reasoning_efforts: ['low', 'medium', 'xhigh'],
        },
        { id: 'legacy', supports_reasoning_slider: false },
        { id: 'spoofed' },
      ],
      balance: 2,
    })
  })

  it('normalizes the optional transcription capability without breaking legacy catalogs', async () => {
    mocks.apiGet.mockResolvedValue({
      data: {
        models: [{ id: 'gpt-5' }],
        balance: 3,
        transcription: {
          enabled: true,
          billing_mode: 'included',
          max_upload_bytes: 8_388_608,
          max_duration_seconds: 120,
          accepted_mime_types: ['audio/webm;codecs=opus', ' ', 'audio/mp4'],
        },
      },
    })

    await expect(getChatModels()).resolves.toMatchObject({
      transcription: {
        enabled: true,
        billing_mode: 'included',
        max_upload_bytes: 8_388_608,
        max_duration_seconds: 120,
        accepted_mime_types: ['audio/webm;codecs=opus', 'audio/mp4'],
      },
    })
  })

  it('uploads recorded audio as FormData without overriding the multipart boundary', async () => {
    const controller = new AbortController()
    const audio = new Blob(['recorded voice'], { type: 'audio/webm;codecs=opus' })
    const idempotencyKey = '11111111-2222-4333-8444-555555555555'
    mocks.apiPost.mockResolvedValue({ data: { text: '  你好世界  ' } })

    await expect(transcribeChatAudio(audio, {
      signal: controller.signal,
      idempotencyKey,
    })).resolves.toEqual({ text: '你好世界' })

    const [url, formData, config] = mocks.apiPost.mock.calls[0]
    expect(url).toBe('/chat/transcriptions')
    expect(formData).toBeInstanceOf(FormData)
    expect((formData as FormData).has('duration_ms')).toBe(false)
    const uploadedFile = (formData as FormData).get('file')
    expect(uploadedFile).toBeInstanceOf(File)
    expect((uploadedFile as File).name).toBe('chat-recording.webm')
    expect((uploadedFile as File).type).toBe('audio/webm;codecs=opus')
    expect(config).toEqual({
      signal: controller.signal,
      timeout: 0,
      headers: {
        'Content-Type': undefined,
        'Idempotency-Key': idempotencyKey,
      },
    })
  })

  it('rejects empty audio and invalid transcription payloads', async () => {
    await expect(transcribeChatAudio(new Blob([]))).rejects.toMatchObject({
      name: 'ChatAPIError',
      code: 'EMPTY_AUDIO',
    })
    expect(mocks.apiPost).not.toHaveBeenCalled()

    mocks.apiPost.mockResolvedValue({ data: { text: '   ' } })
    await expect(transcribeChatAudio(new Blob(['voice']))).rejects.toMatchObject({
      name: 'ChatAPIError',
      code: 'INVALID_TRANSCRIPTION_RESPONSE',
    })
  })

  it('honors an already-aborted transcription signal before uploading', async () => {
    const controller = new AbortController()
    controller.abort()

    await expect(transcribeChatAudio(new Blob(['voice']), {
      signal: controller.signal,
    })).rejects.toMatchObject({ name: 'AbortError' })
    expect(mocks.apiPost).not.toHaveBeenCalled()
  })

  it('preserves the dedicated daily transcription quota reason and reset metadata', async () => {
    mocks.apiPost.mockRejectedValue({
      status: 429,
      code: 429,
      reason: 'TRANSCRIPTION_DAILY_QUOTA_EXCEEDED',
      message: 'Daily voice transcription limit reached',
      metadata: {
        limit_seconds: '1200',
        reset_in_seconds: '3600',
        reset_at: '2026-08-08T00:00:00Z',
      },
    })

    await expect(transcribeChatAudio(new Blob(['voice']))).rejects.toMatchObject({
      name: 'ChatAPIError',
      status: 429,
      code: 'TRANSCRIPTION_DAILY_QUOTA_EXCEEDED',
      reason: 'TRANSCRIPTION_DAILY_QUOTA_EXCEEDED',
      metadata: {
        limit_seconds: '1200',
        reset_in_seconds: '3600',
        reset_at: '2026-08-08T00:00:00Z',
      },
    })
  })

  it('normalizes model-list business reasons into ChatAPIError codes', async () => {
    mocks.apiGet.mockRejectedValue({
      status: 403,
      code: 403,
      reason: 'INSUFFICIENT_BALANCE',
      message: 'Insufficient account balance',
    })

    await expect(getChatModels()).rejects.toMatchObject({
      name: 'ChatAPIError',
      status: 403,
      code: 'INSUFFICIENT_BALANCE',
      reason: 'INSUFFICIENT_BALANCE',
    })
  })

  it('normalizes final receipt fields from the authenticated receipt endpoint', async () => {
    const controller = new AbortController()
    mocks.apiGet.mockResolvedValue({
      status: 200,
      data: {
        receipt_id: 'receipt/one',
        status: 'charged',
        usage_log_id: 42,
        model: 'gpt-5.5',
        input_tokens: 120,
        output_tokens: 30,
        cache_creation_tokens: 10,
        cache_read_tokens: 80,
        total_tokens: 160,
        gross_cost: '0.0032',
        charged_amount: 0.0024,
        billing_type: 0,
        balance_before: 5,
        balance_after: 4.9976,
        created_at: '2026-07-25T01:02:03Z',
      },
    })

    await expect(getChatReceipt(' receipt/one ', controller.signal)).resolves.toEqual({
      receiptId: 'receipt/one',
      status: 'charged',
      usageLogId: 42,
      model: 'gpt-5.5',
      inputTokens: 120,
      outputTokens: 30,
      cacheCreationTokens: 10,
      cacheReadTokens: 80,
      totalTokens: 160,
      grossCost: 0.0032,
      chargedAmount: 0.0024,
      billingType: 0,
      balanceBefore: 5,
      balanceAfter: 4.9976,
      createdAt: '2026-07-25T01:02:03Z',
    })
    expect(mocks.apiGet).toHaveBeenCalledWith(
      '/chat/receipts/receipt%2Fone',
      { signal: controller.signal },
    )
  })

  it('polls pending receipts with bounded backoff and returns the final receipt', async () => {
    mocks.apiGet
      .mockResolvedValueOnce({
        status: 202,
        data: { receipt_id: 'receipt-poll', status: 'pending' },
      })
      .mockResolvedValueOnce({
        status: 200,
        data: {
          receipt_id: 'receipt-poll',
          status: 'not_charged',
          charged_amount: 0,
        },
      })

    await expect(pollChatReceipt('receipt-poll', { delays: [0] })).resolves.toEqual({
      receiptId: 'receipt-poll',
      status: 'not_charged',
      chargedAmount: 0,
    })
    expect(mocks.apiGet).toHaveBeenCalledTimes(2)
  })

  it('returns pending after the bounded receipt poll window without reporting a false failure', async () => {
    mocks.apiGet.mockResolvedValue({
      status: 202,
      data: { receipt_id: 'receipt-pending', status: 'pending' },
    })

    await expect(pollChatReceipt('receipt-pending', { delays: [0, 0] })).resolves.toEqual({
      receiptId: 'receipt-pending',
      status: 'pending',
    })
    expect(mocks.apiGet).toHaveBeenCalledTimes(3)
  })

  it('aborts receipt backoff immediately when its AbortSignal is cancelled', async () => {
    mocks.apiGet.mockResolvedValue({
      status: 202,
      data: { receipt_id: 'receipt-abort', status: 'pending' },
    })
    const controller = new AbortController()
    const polling = pollChatReceipt('receipt-abort', {
      signal: controller.signal,
      delays: [10_000],
    })
    await vi.waitFor(() => {
      expect(mocks.apiGet).toHaveBeenCalledOnce()
    })

    controller.abort()

    await expect(polling).rejects.toMatchObject({ name: 'AbortError' })
    expect(mocks.apiGet).toHaveBeenCalledOnce()
  })

  it('uses public IDs for idempotent legacy import and sends only approved message fields', async () => {
    mocks.apiPost.mockResolvedValue({
      data: {
        id: 'conversation-import',
        title: 'Imported',
        model: 'gpt-5',
        revision: 1,
        version: 3,
        head_message_id: 'assistant-import',
        message_count: 2,
        created_at: '2026-07-25T01:00:00Z',
        updated_at: '2026-07-25T01:01:00Z',
      },
    })

    await createChatConversation({
      id: 'conversation-import',
      title: 'Imported',
      model: 'gpt-5',
      importedMessages: [
        {
          id: 'user-import',
          role: 'user',
          content: 'Private legacy question',
          status: 'completed',
          createdAt: Date.parse('2026-07-25T01:00:00Z'),
        },
        {
          id: 'assistant-import',
          role: 'assistant',
          content: 'Legacy answer',
          status: 'interrupted',
          createdAt: Date.parse('2026-07-25T01:01:00Z'),
        },
      ],
    })

    expect(mocks.apiPost).toHaveBeenCalledWith(
      '/chat/conversations',
      {
        id: 'conversation-import',
        title: 'Imported',
        model: 'gpt-5',
        imported_messages: [
          {
            id: 'user-import',
            role: 'user',
            content: 'Private legacy question',
            status: 'completed',
            created_at: '2026-07-25T01:00:00.000Z',
          },
          {
            id: 'assistant-import',
            role: 'assistant',
            content: 'Legacy answer',
            status: 'interrupted',
            created_at: '2026-07-25T01:01:00.000Z',
          },
        ],
      },
      { signal: undefined },
    )
    const serialized = JSON.stringify(mocks.apiPost.mock.calls[0]?.[1])
    expect(serialized).not.toMatch(
      /receipt|token|gross|charged|balance|actual_model|attempt|superseded/i,
    )
  })

  it('keeps list cursors in query params and search text in a POST body', async () => {
    mocks.apiGet.mockResolvedValueOnce({
      data: { items: [], next_cursor: 'cursor-2', has_more: true },
    })
    mocks.apiPost.mockResolvedValueOnce({
      data: { conversations: [], next_cursor: null, has_more: false },
    })

    await expect(listChatConversations({ cursor: 'cursor-1', limit: 20 }))
      .resolves.toMatchObject({ nextCursor: 'cursor-2', hasMore: true })
    await searchChatConversations({ query: 'private phrase', limit: 30 })

    expect(mocks.apiGet).toHaveBeenCalledWith('/chat/conversations', {
      params: { cursor: 'cursor-1', limit: 20 },
      signal: undefined,
    })
    expect(mocks.apiPost).toHaveBeenCalledWith(
      '/chat/conversations/search',
      { query: 'private phrase', limit: 30 },
      { signal: undefined },
    )
  })

  it('normalizes history message pages into canonical position order', async () => {
    mocks.apiGet.mockResolvedValueOnce({
      data: {
        items: [
          {
            id: 'assistant-2',
            role: 'assistant',
            content: '回答',
            status: 'completed',
            position: 2,
            created_at: '2026-08-09T08:00:00Z',
          },
          {
            id: 'user-1',
            role: 'user',
            content: '你好',
            status: 'completed',
            position: 1,
            created_at: '2026-08-09T08:00:00Z',
          },
        ],
        next_before_position: 1,
        has_more: true,
      },
    })

    const page = await getChatConversationMessages('conversation-1', {
      beforePosition: 3,
      limit: 100,
    })

    expect(page.items.map(({ id, position }) => ({ id, position }))).toEqual([
      { id: 'user-1', position: 1 },
      { id: 'assistant-2', position: 2 },
    ])
    expect(page).toMatchObject({ nextBeforePosition: 1, hasMore: true })
    expect(mocks.apiGet).toHaveBeenCalledWith(
      '/chat/conversations/conversation-1/messages',
      {
        params: { before_position: 3, limit: 100 },
        signal: undefined,
      },
    )
  })

  it('repairs legacy settlement-only errors when delivery already terminated', async () => {
    mocks.apiGet.mockResolvedValueOnce({
      data: {
        items: [{
          id: 'assistant-settlement-only',
          role: 'assistant',
          content: 'The complete delivered answer.',
          status: 'error',
          finish_reason: 'stop',
          error_code: 'CHAT_SETTLEMENT_FAILED',
          error_message: 'Chat usage settlement could not be completed',
          settlement_status: 'failed',
          position: 1,
          created_at: '2026-08-09T08:00:00Z',
        }],
        next_before_position: null,
        has_more: false,
      },
    })

    const page = await getChatConversationMessages('conversation-settlement-only')

    expect(page.items[0]).toMatchObject({
      id: 'assistant-settlement-only',
      content: 'The complete delivered answer.',
      status: 'complete',
      finishReason: 'stop',
      settlementStatus: 'failed',
    })
    expect(page.items[0]?.errorCode).toBeUndefined()
    expect(page.items[0]?.errorMessage).toBeUndefined()
  })

  it('keeps legacy settlement errors when the stream has no terminal delivery evidence', async () => {
    mocks.apiGet.mockResolvedValueOnce({
      data: {
        items: [{
          id: 'assistant-settlement-interrupted',
          role: 'assistant',
          content: 'Only a partial answer',
          status: 'error',
          finish_reason: 'disconnected',
          error_code: 'CHAT_SETTLEMENT_FAILED',
          error_message: 'Chat usage settlement could not be completed',
          position: 1,
          created_at: '2026-08-09T08:00:00Z',
        }],
        next_before_position: null,
        has_more: false,
      },
    })

    const page = await getChatConversationMessages('conversation-settlement-interrupted')

    expect(page.items[0]).toMatchObject({
      status: 'error',
      finishReason: 'disconnected',
      errorCode: 'CHAT_SETTLEMENT_FAILED',
    })
  })

  it('keeps genuine generation errors even when they include content and a finish reason', async () => {
    mocks.apiGet.mockResolvedValueOnce({
      data: {
        items: [{
          id: 'assistant-upstream-error',
          role: 'assistant',
          content: 'Visible partial output',
          status: 'error',
          finish_reason: 'stop',
          error_code: 'UPSTREAM_ERROR',
          error_message: 'The upstream request failed',
          position: 1,
          created_at: '2026-08-09T08:00:00Z',
        }],
        next_before_position: null,
        has_more: false,
      },
    })

    const page = await getChatConversationMessages('conversation-upstream-error')

    expect(page.items[0]).toMatchObject({
      status: 'error',
      finishReason: 'stop',
      errorCode: 'UPSTREAM_ERROR',
    })
  })

  it('normalizes snake_case reasoning activities from server history messages', async () => {
    mocks.apiGet.mockResolvedValueOnce({
      data: {
        items: [{
          id: 'assistant-activity',
          role: 'assistant',
          content: 'Final answer',
          status: 'completed',
          position: 1,
          created_at: '2026-08-09T08:00:00Z',
          activities: [{
            response_id: 'resp-history',
            source: 'openai_responses',
            activity_type: 'reasoning_summary',
            item_id: 'reasoning-history',
            output_index: 0,
            summary_index: 0,
            sort_order: 1,
            status: 'completed',
            text: 'Checked the persisted evidence.',
            sequence_start: 1,
            sequence_end: 7,
            reasoning_mode: 'pro',
            reasoning_effort: 'high',
            started_at: '2026-08-09T08:00:01Z',
            completed_at: '2026-08-09T08:00:02Z',
            metadata: { last_event: 'response.completed' },
            reasoning_content: 'must not survive',
          }],
        }],
        next_before_position: null,
        has_more: false,
      },
    })

    const page = await getChatConversationMessages('conversation-activity')
    const activity = page.items[0]?.activities?.[0]
    expect(activity).toMatchObject({
      key: 'resp-history',
      responseId: 'resp-history',
      status: 'completed',
      reasoningMode: 'pro',
      items: [{
        itemId: 'reasoning-history',
        outputIndex: 0,
        parts: [{ text: 'Checked the persisted evidence.' }],
      }],
    })
    expect(activity?.reasoningEffort).toBeUndefined()
    expect(JSON.stringify(page.items[0]?.activities)).not.toContain('must not survive')
  })

  it('sends optimistic revisions for patch and delete', async () => {
    mocks.apiPatch.mockResolvedValue({
      data: {
        id: 'conversation-1',
        title: 'Renamed',
        model: 'gpt-5',
        revision: 8,
        version: 11,
        created_at: 1_700_000_000_000,
        updated_at: 1_700_000_001_000,
      },
    })
    mocks.apiDelete.mockResolvedValue({ data: null })

    await patchChatConversation('conversation-1', {
      revision: 7,
      title: 'Renamed',
    })
    await deleteChatConversation('conversation-1', { revision: 8 })

    expect(mocks.apiPatch).toHaveBeenCalledWith(
      '/chat/conversations/conversation-1',
      { revision: 7, title: 'Renamed' },
      { signal: undefined },
    )
    expect(mocks.apiDelete).toHaveBeenCalledWith(
      '/chat/conversations/conversation-1',
      { data: { revision: 8 }, signal: undefined },
    )
  })

  it('normalizes incremental tombstones and attempt recovery without a completion retry', async () => {
    mocks.apiGet
      .mockResolvedValueOnce({
        data: {
          changes: [{
            type: 'tombstone',
            conversation_id: 'deleted-conversation',
            version: 12,
            deleted_at: '2026-07-25T02:00:00Z',
          }],
          latest_version: 12,
          next_cursor: null,
          has_more: false,
        },
      })
      .mockResolvedValueOnce({
        data: {
          attempt_id: 'attempt-1',
          conversation_id: 'conversation-1',
          assistant_message_id: 'assistant-1',
          status: 'interrupted',
          assistant_message: {
            id: 'assistant-1',
            role: 'assistant',
            content: 'Partial server content',
            status: 'interrupted',
            created_at: '2026-07-25T02:00:00Z',
          },
        },
      })

    await expect(getChatSync({ afterVersion: 9 })).resolves.toMatchObject({
      latestVersion: 12,
      changes: [{ type: 'delete', conversationId: 'deleted-conversation' }],
    })
    await expect(getChatAttempt('attempt-1')).resolves.toMatchObject({
      attemptId: 'attempt-1',
      status: 'stopped',
      assistantMessage: {
        content: 'Partial server content',
        status: 'stopped',
        finishReason: 'interrupted',
      },
    })
  })

  it('trusts an authoritative completed message for a settlement-only failed attempt', async () => {
    mocks.apiGet.mockResolvedValueOnce({
      data: {
        attempt_id: 'attempt-settlement-only',
        conversation_id: 'conversation-settlement-only',
        assistant_message_id: 'assistant-settlement-only',
        status: 'failed',
        failure_code: 'CHAT_SETTLEMENT_FAILED',
        failure_reason: 'Chat usage settlement could not be completed',
        assistant_message: {
          id: 'assistant-settlement-only',
          role: 'assistant',
          content: '',
          status: 'completed',
          created_at: '2026-08-09T08:00:00Z',
        },
      },
    })

    const attempt = await getChatAttempt('attempt-settlement-only')

    expect(attempt).toMatchObject({
      status: 'completed',
      assistantMessage: {
        status: 'complete',
        content: '',
      },
    })
    expect(attempt.failureCode).toBeUndefined()
    expect(attempt.failureReason).toBeUndefined()
    expect(attempt.assistantMessage?.errorCode).toBeUndefined()
  })

  it('records an explicit stop against the authenticated attempt endpoint', async () => {
    mocks.apiPost.mockResolvedValueOnce({
      data: {
        attempt_id: 'attempt-12345678',
        accepted: true,
        attempt_status: 'interrupted',
        delivery_status: 'stopped',
        stopped_at: '2026-08-14T08:00:00Z',
      },
    })

    await expect(stopChatAttempt(' attempt-12345678 ')).resolves.toMatchObject({
      attemptId: 'attempt-12345678',
      accepted: true,
      attemptStatus: 'interrupted',
      deliveryStatus: 'stopped',
      stoppedAt: Date.parse('2026-08-14T08:00:00Z'),
    })

    expect(mocks.apiPost).toHaveBeenCalledWith(
      '/chat/attempts/attempt-12345678/stop',
    )
  })

  it('rejects an empty stop attempt ID without sending a request', async () => {
    await expect(stopChatAttempt('   ')).rejects.toMatchObject({
      code: 'INVALID_CHAT_ATTEMPT_ID',
    })
    expect(mocks.apiPost).not.toHaveBeenCalled()
  })

  it('retries a transient stop failure against the idempotent endpoint', async () => {
    mocks.apiPost
      .mockRejectedValueOnce({ status: 503, message: 'temporarily unavailable' })
      .mockResolvedValueOnce({
        data: {
          attempt_id: 'attempt-12345678',
          accepted: true,
          attempt_status: 'interrupted',
          delivery_status: 'stopped',
        },
      })

    await expect(stopChatAttempt('attempt-12345678', { delays: [0] }))
      .resolves.toMatchObject({ accepted: true, deliveryStatus: 'stopped' })
    expect(mocks.apiPost).toHaveBeenCalledTimes(2)
  })

  it('returns a completed-before-stop race for caller reconciliation', async () => {
    mocks.apiPost.mockResolvedValueOnce({
      data: {
        attempt_id: 'attempt-12345678',
        accepted: false,
        attempt_status: 'completed',
        delivery_status: 'completed',
      },
    })

    await expect(stopChatAttempt('attempt-12345678')).resolves.toEqual({
      attemptId: 'attempt-12345678',
      accepted: false,
      attemptStatus: 'completed',
      deliveryStatus: 'completed',
    })
  })

  it('does not retry a terminal stop request error', async () => {
    mocks.apiPost.mockRejectedValueOnce({ status: 404, message: 'not found' })

    await expect(stopChatAttempt('attempt-12345678', { delays: [0, 0] }))
      .rejects.toMatchObject({ status: 404 })
    expect(mocks.apiPost).toHaveBeenCalledTimes(1)
  })

  it('posts a streaming request with the JWT and emits content', async () => {
    const fetchMock = vi.fn().mockResolvedValue(successfulStreamResponse(
      [
        'data: {"choices":[{"index":0,"delta":{"content":"hello"},"finish_reason":null}]}\n\n',
        'data: [DONE]\n\n',
      ].join(''),
      { 'X-Chat-Receipt-ID': 'receipt-stream' },
    ))
    vi.stubGlobal('fetch', fetchMock)
    const content: string[] = []
    const onReceiptId = vi.fn()

    const result = await streamChatCompletion(
      {
        ...historyCompletionRequest(' gpt-5 '),
        reasoningMode: 'pro',
        reasoningEffort: 'xhigh',
      },
      {
        onContent: (delta) => content.push(delta),
        onReceiptId,
      },
      { attemptId: 'attempt-one' },
    )

    expect(content).toEqual(['hello'])
    expect(onReceiptId).toHaveBeenCalledWith('receipt-stream')
    expect(result.receiptId).toBe('receipt-stream')
    expect(fetchMock).toHaveBeenCalledOnce()
    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toBe('/api/v1/chat/completions')
    expect(init.headers).toMatchObject({
      Authorization: 'Bearer jwt-one',
      'X-Chat-Attempt-ID': 'attempt-one',
    })
    expect(JSON.parse(String(init.body))).toMatchObject({
      conversation_id: 'conversation-1',
      model: 'gpt-5',
      reasoning: {
        mode: 'pro',
        summary: 'auto',
      },
      expected_head_message_id: 'assistant-previous',
      user_message: { id: 'user-1', content: 'Hi' },
      assistant_message_id: 'assistant-1',
    })
    expect(Object.keys(JSON.parse(String(init.body))).sort()).toEqual([
      'assistant_message_id',
      'conversation_id',
      'expected_head_message_id',
      'model',
      'reasoning',
      'user_message',
    ])
    expect(JSON.parse(String(init.body)).reasoning).not.toHaveProperty('effort')
  })

  it('seeds Activity snapshots from the completion request without rewriting envelopes', async () => {
    const payload = {
      sequence_number: 1,
      output_index: 0,
      item: { id: 'reasoning-request', type: 'reasoning' },
    }
    const source = [
      `data: ${JSON.stringify({
        source: 'openai_responses',
        eventType: 'response.created',
        payload: { sequence_number: 0, response: { id: 'resp-request' } },
      })}\n\n`,
      `data: ${JSON.stringify({
        source: 'openai_responses',
        eventType: 'response.output_item.added',
        payload,
      })}\n\n`,
      `data: ${JSON.stringify({
        source: 'openai_responses',
        eventType: 'response.reasoning_summary_text.delta',
        payload: {
          sequence_number: 2,
          item_id: 'reasoning-request',
          output_index: 0,
          summary_index: 0,
          delta: 'Request metadata is preserved.',
        },
      })}\n\n`,
      'data: [DONE]\n\n',
    ].join('')
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(successfulStreamResponse(source)))
    const onActivity = vi.fn()
    const onActivityEvent = vi.fn()

    await streamChatCompletion({
      ...historyCompletionRequest(),
      reasoningMode: 'pro',
      reasoningEffort: 'high',
    }, { onActivity, onActivityEvent })

    const lastActivity = onActivity.mock.calls.at(-1)?.[0]
    expect(lastActivity).toEqual(expect.objectContaining({
      responseId: 'resp-request',
      reasoningMode: 'pro',
    }))
    expect(lastActivity?.reasoningEffort).toBeUndefined()
    expect(onActivityEvent).toHaveBeenCalledWith({
      source: 'openai_responses',
      eventType: 'response.output_item.added',
      payload,
    })
  })

  it('refreshes an expired JWT once before opening the stream', async () => {
    const cancel = vi.fn().mockResolvedValue(undefined)
    const fetchMock = vi.fn()
      .mockResolvedValueOnce({ status: 401, body: { cancel } })
      .mockResolvedValueOnce(successfulStreamResponse('data: [DONE]\n\n'))
    vi.stubGlobal('fetch', fetchMock)
    mocks.refreshAuthSession.mockImplementation(async () => {
      mocks.getSnapshot.mockReturnValue(refreshedSession)
      return refreshedSession
    })

    await streamChatCompletion(historyCompletionRequest())

    expect(cancel).toHaveBeenCalledOnce()
    expect(mocks.refreshAuthSession).toHaveBeenCalledOnce()
    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(fetchMock.mock.calls[1]?.[1]?.headers).toMatchObject({
      Authorization: 'Bearer jwt-two',
    })
    expect(mocks.invalidate).not.toHaveBeenCalled()
    expect(sessionStorage.getItem('auth_expired')).toBeNull()
  })

  it('uses only the final successful response receipt header after a 401 refresh', async () => {
    const cancel = vi.fn().mockResolvedValue(undefined)
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(unauthorizedResponse(
        cancel,
        { 'X-Chat-Receipt-ID': 'receipt-from-401' },
      ))
      .mockResolvedValueOnce(successfulStreamResponse(
        'data: [DONE]\n\n',
        { 'X-Chat-Receipt-ID': 'receipt-after-refresh' },
      ))
    vi.stubGlobal('fetch', fetchMock)
    mocks.refreshAuthSession.mockImplementation(async () => {
      mocks.getSnapshot.mockReturnValue(refreshedSession)
      return refreshedSession
    })
    const onReceiptId = vi.fn()

    const result = await streamChatCompletion(
      historyCompletionRequest(),
      { onReceiptId },
      { attemptId: 'attempt-stable' },
    )

    expect(result.receiptId).toBe('receipt-after-refresh')
    expect(onReceiptId).toHaveBeenCalledTimes(1)
    expect(onReceiptId).toHaveBeenCalledWith('receipt-after-refresh')
    expect(fetchMock.mock.calls[0]?.[1]?.headers).toMatchObject({
      'X-Chat-Attempt-ID': 'attempt-stable',
    })
    expect(fetchMock.mock.calls[1]?.[1]?.headers).toMatchObject({
      'X-Chat-Attempt-ID': 'attempt-stable',
    })
  })

  it('invalidates the observed session and redirects after a 401 without a refresh token', async () => {
    const cancel = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(unauthorizedResponse(cancel)))
    mocks.getSnapshot.mockReturnValue({ ...initialSession, refreshToken: null })

    await expect(streamChatCompletion(historyCompletionRequest())).rejects.toMatchObject({
      status: 401,
      code: 'TOKEN_REFRESH_FAILED',
    })

    expect(cancel).toHaveBeenCalledOnce()
    expect(mocks.refreshAuthSession).not.toHaveBeenCalled()
    expect(mocks.invalidate).toHaveBeenCalledOnce()
    expect(mocks.invalidate).toHaveBeenCalledWith(sessionIdentity({
      ...initialSession,
      refreshToken: null,
    }))
    expect(sessionStorage.getItem('auth_expired')).toBe('1')
    expect(window.location.href).toBe('/login')
  })

  it.each([400, 401, 403])(
    'invalidates the observed session when refresh fails fatally with %s',
    async (status) => {
      vi.stubGlobal('fetch', vi.fn().mockResolvedValue(unauthorizedResponse()))
      mocks.refreshAuthSession.mockRejectedValue({
        isAxiosError: true,
        message: 'refresh rejected',
        response: { status },
      })

      await expect(streamChatCompletion(historyCompletionRequest())).rejects.toMatchObject({
        status: 401,
        code: 'TOKEN_REFRESH_FAILED',
      })

      expect(mocks.invalidate).toHaveBeenCalledOnce()
      expect(mocks.invalidate).toHaveBeenCalledWith(sessionIdentity(initialSession))
      expect(sessionStorage.getItem('auth_expired')).toBe('1')
      expect(window.location.href).toBe('/login')
    },
  )

  it('invalidates the observed session when refresh returns an invalid payload', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(unauthorizedResponse()))
    mocks.refreshAuthSession.mockRejectedValue(new Error('Token refresh failed'))

    await expect(streamChatCompletion(historyCompletionRequest())).rejects.toMatchObject({
      status: 401,
      code: 'TOKEN_REFRESH_FAILED',
    })

    expect(mocks.invalidate).toHaveBeenCalledOnce()
    expect(sessionStorage.getItem('auth_expired')).toBe('1')
    expect(window.location.href).toBe('/login')
  })

  it.each([
    ['a network error', undefined, 0],
    ['rate limiting', 429, 429],
    ['a server error', 500, 500],
  ])('preserves the session when refresh encounters %s', async (_label, status, expectedStatus) => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(unauthorizedResponse()))
    mocks.refreshAuthSession.mockRejectedValue({
      isAxiosError: true,
      message: 'refresh temporarily unavailable',
      response: status === undefined ? undefined : { status },
    })

    await expect(streamChatCompletion(historyCompletionRequest())).rejects.toMatchObject({
      status: expectedStatus,
      code: 'TOKEN_REFRESH_TEMPORARILY_UNAVAILABLE',
    })

    expect(mocks.invalidate).not.toHaveBeenCalled()
    expect(sessionStorage.getItem('auth_expired')).toBeNull()
    expect(window.location.href).toBe('/chat')
  })

  it('invalidates the refreshed session when the retried stream still returns 401', async () => {
    const firstCancel = vi.fn().mockResolvedValue(undefined)
    const secondCancel = vi.fn().mockResolvedValue(undefined)
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(unauthorizedResponse(firstCancel))
      .mockResolvedValueOnce(unauthorizedResponse(secondCancel))
    vi.stubGlobal('fetch', fetchMock)
    mocks.refreshAuthSession.mockImplementation(async () => {
      mocks.getSnapshot.mockReturnValue(refreshedSession)
      return refreshedSession
    })

    await expect(streamChatCompletion(historyCompletionRequest())).rejects.toMatchObject({
      status: 401,
      code: 'TOKEN_REFRESH_FAILED',
    })

    expect(firstCancel).toHaveBeenCalledOnce()
    expect(secondCancel).toHaveBeenCalledOnce()
    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(mocks.invalidate).toHaveBeenCalledOnce()
    expect(mocks.invalidate).toHaveBeenCalledWith(sessionIdentity(refreshedSession))
    expect(sessionStorage.getItem('auth_expired')).toBe('1')
    expect(window.location.href).toBe('/login')
  })

  it('does not refresh or retry when the session changes before refresh starts', async () => {
    const cancel = vi.fn().mockResolvedValue(undefined)
    const fetchMock = vi.fn().mockResolvedValue({ status: 401, body: { cancel } })
    vi.stubGlobal('fetch', fetchMock)
    mocks.getSnapshot
      .mockReturnValueOnce({
        accessToken: 'jwt-one',
        refreshToken: 'refresh-one',
        expiresAt: null,
        user: { id: 7 },
      })
      .mockReturnValueOnce({
        accessToken: 'jwt-other',
        refreshToken: 'refresh-other',
        expiresAt: null,
        user: { id: 8 },
      })
    mocks.getGeneration
      .mockReturnValueOnce('generation-one')
      .mockReturnValueOnce('generation-two')

    await expect(streamChatCompletion(historyCompletionRequest())).rejects.toMatchObject({
      name: 'ChatAPIError',
      status: 409,
      code: 'AUTH_SESSION_CHANGED',
    })

    expect(cancel).toHaveBeenCalledOnce()
    expect(mocks.refreshAuthSession).not.toHaveBeenCalled()
    expect(fetchMock).toHaveBeenCalledOnce()
    expect(mocks.invalidate).not.toHaveBeenCalled()
    expect(sessionStorage.getItem('auth_expired')).toBeNull()
    expect(window.location.href).toBe('/chat')
  })

  it.each([
    ['access token', { ...initialSession, accessToken: 'jwt-other' }],
    ['refresh token', { ...initialSession, refreshToken: 'refresh-other' }],
    ['user', { ...initialSession, user: { id: 8 } }],
  ])('does not refresh when the same-generation %s changes', async (_label, changedSession) => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(unauthorizedResponse()))
    mocks.getSnapshot
      .mockReturnValueOnce(initialSession)
      .mockReturnValue(changedSession)

    await expect(streamChatCompletion(historyCompletionRequest())).rejects.toMatchObject({
      status: 409,
      code: 'AUTH_SESSION_CHANGED',
    })

    expect(mocks.refreshAuthSession).not.toHaveBeenCalled()
    expect(mocks.invalidate).not.toHaveBeenCalled()
    expect(sessionStorage.getItem('auth_expired')).toBeNull()
    expect(window.location.href).toBe('/chat')
  })

  it('does not retry with refreshed credentials after the session generation changes', async () => {
    const cancel = vi.fn().mockResolvedValue(undefined)
    const fetchMock = vi.fn().mockResolvedValue({ status: 401, body: { cancel } })
    vi.stubGlobal('fetch', fetchMock)
    mocks.getGeneration
      .mockReturnValueOnce('generation-one')
      .mockReturnValueOnce('generation-one')
      .mockReturnValueOnce('generation-two')
    mocks.refreshAuthSession.mockResolvedValue({
      accessToken: 'jwt-two',
      refreshToken: 'refresh-two',
      expiresAt: null,
      user: { id: 7 },
    })

    await expect(streamChatCompletion(historyCompletionRequest())).rejects.toMatchObject({
      name: 'ChatAPIError',
      status: 409,
      code: 'AUTH_SESSION_CHANGED',
    })

    expect(cancel).toHaveBeenCalledOnce()
    expect(mocks.refreshAuthSession).toHaveBeenCalledOnce()
    expect(fetchMock).toHaveBeenCalledOnce()
    expect(mocks.invalidate).not.toHaveBeenCalled()
    expect(sessionStorage.getItem('auth_expired')).toBeNull()
    expect(window.location.href).toBe('/chat')
  })

  it('does not invalidate a same-generation rotation after a fatal refresh failure', async () => {
    const rotatedSession = {
      ...initialSession,
      accessToken: 'jwt-rotated',
      refreshToken: 'refresh-rotated',
    }
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(unauthorizedResponse()))
    mocks.refreshAuthSession.mockImplementation(async () => {
      mocks.getSnapshot.mockReturnValue(rotatedSession)
      throw {
        isAxiosError: true,
        message: 'stale refresh rejected',
        response: { status: 401 },
      }
    })

    await expect(streamChatCompletion(historyCompletionRequest())).rejects.toMatchObject({
      status: 409,
      code: 'AUTH_SESSION_CHANGED',
    })

    expect(mocks.invalidate).toHaveBeenCalledOnce()
    expect(mocks.invalidate).toHaveBeenCalledWith(sessionIdentity(initialSession))
    expect(sessionStorage.getItem('auth_expired')).toBeNull()
    expect(window.location.href).toBe('/chat')
  })

  it('does not invalidate a same-generation rotation before the retried 401 settles', async () => {
    const rotatedSession = { ...refreshedSession, refreshToken: 'refresh-three' }
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(unauthorizedResponse())
      .mockImplementationOnce(async () => {
        mocks.getSnapshot.mockReturnValue(rotatedSession)
        return unauthorizedResponse()
      })
    vi.stubGlobal('fetch', fetchMock)
    mocks.refreshAuthSession.mockImplementation(async () => {
      mocks.getSnapshot.mockReturnValue(refreshedSession)
      return refreshedSession
    })

    await expect(streamChatCompletion(historyCompletionRequest())).rejects.toMatchObject({
      status: 409,
      code: 'AUTH_SESSION_CHANGED',
    })

    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(mocks.invalidate).toHaveBeenCalledOnce()
    expect(mocks.invalidate).toHaveBeenCalledWith(sessionIdentity(refreshedSession))
    expect(sessionStorage.getItem('auth_expired')).toBeNull()
    expect(window.location.href).toBe('/chat')
  })

  it('does not clear a newer session when compare-generation invalidation loses the race', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(unauthorizedResponse()))
    mocks.getSnapshot.mockReturnValue({ ...initialSession, refreshToken: null })
    mocks.invalidate.mockReturnValue(false)

    await expect(streamChatCompletion(historyCompletionRequest())).rejects.toMatchObject({
      status: 409,
      code: 'AUTH_SESSION_CHANGED',
    })

    expect(mocks.invalidate).toHaveBeenCalledWith(sessionIdentity({
      ...initialSession,
      refreshToken: null,
    }))
    expect(sessionStorage.getItem('auth_expired')).toBeNull()
    expect(window.location.href).toBe('/chat')
  })

  it('does not invalidate when refresh reports that the auth session changed', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(unauthorizedResponse()))
    mocks.refreshAuthSession.mockRejectedValue(new Error('session changed'))
    mocks.isAuthSessionChangedError.mockReturnValue(true)

    await expect(streamChatCompletion(historyCompletionRequest())).rejects.toMatchObject({
      status: 409,
      code: 'AUTH_SESSION_CHANGED',
    })

    expect(mocks.invalidate).not.toHaveBeenCalled()
    expect(sessionStorage.getItem('auth_expired')).toBeNull()
    expect(window.location.href).toBe('/chat')
  })

  it('preserves the insufficient-balance business code from HTTP errors', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false,
      status: 402,
      statusText: 'Payment Required',
      headers: new Headers({ 'X-Request-Id': 'request-1' }),
      text: vi.fn().mockResolvedValue(JSON.stringify({
        code: 403,
        message: '请先充值',
        reason: 'INSUFFICIENT_BALANCE',
      })),
    }))

    const error = await streamChatCompletion(historyCompletionRequest()).catch((reason: unknown) => reason)

    expect(error).toBeInstanceOf(ChatAPIError)
    expect(error).toMatchObject({
      status: 402,
      code: 'INSUFFICIENT_BALANCE',
      reason: 'INSUFFICIENT_BALANCE',
      message: '请先充值',
      requestId: 'request-1',
    })
  })

  it('uploads one multipart attachment and normalizes server metadata', async () => {
    const progress = vi.fn()
    mocks.apiPost.mockImplementation(async (
      path: string,
      body: FormData,
      config: { onUploadProgress?: (event: { loaded: number; total: number }) => void },
    ) => {
      expect(path).toBe('/chat/attachments')
      expect(body).toBeInstanceOf(FormData)
      expect(body.get('file')).toBeInstanceOf(File)
      config.onUploadProgress?.({ loaded: 3, total: 3 })
      return {
        data: {
          id: 'attachment-1',
          name: 'snow.png',
          kind: 'image',
          mime_type: 'image/png',
          size: 3,
          status: 'ready',
          expires_at: '2099-01-01T00:00:00Z',
          width: 32,
          height: 24,
        },
      }
    })

    await expect(uploadChatAttachment(
      new File(['png'], 'snow.png', { type: 'image/png' }),
      { onProgress: progress },
    )).resolves.toEqual({
      id: 'attachment-1',
      name: 'snow.png',
      kind: 'image',
      mimeType: 'image/png',
      size: 3,
      status: 'ready',
      expiresAt: '2099-01-01T00:00:00Z',
      width: 32,
      height: 24,
    })
    expect(progress).toHaveBeenCalledWith(100)
  })

  it.each([
    [
      'empty-type.docx',
      'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
    ],
    ['empty-type.png', 'image/png'],
  ])('supplies the canonical MIME for %s when the browser omits File.type', async (
    name,
    expectedMimeType,
  ) => {
    mocks.apiPost.mockImplementation(async (_path: string, body: FormData) => {
      const uploaded = body.get('file')
      expect(uploaded).toBeInstanceOf(File)
      expect((uploaded as File).type).toBe(expectedMimeType)
      return {
        data: {
          id: `attachment-${name}`,
          name,
          kind: name.endsWith('.png') ? 'image' : 'document',
          mime_type: expectedMimeType,
          size: 4,
          status: 'ready',
          expires_at: '2099-01-01T00:00:00Z',
        },
      }
    })

    await uploadChatAttachment(new File(['file'], name, { type: '' }))
    expect(mocks.apiPost).toHaveBeenCalledTimes(1)
  })

  it('deletes unbound attachments and loads image bytes through the authenticated client', async () => {
    const blob = new Blob(['image'], { type: 'image/png' })
    mocks.apiDelete.mockResolvedValue({ data: null })
    mocks.apiGet.mockResolvedValue({ data: blob })

    await deleteChatAttachment('attachment/with space')
    await expect(getChatAttachmentContent('attachment/with space')).resolves.toBe(blob)

    expect(mocks.apiDelete).toHaveBeenCalledWith(
      '/chat/attachments/attachment%2Fwith%20space',
      { signal: undefined },
    )
    expect(mocks.apiGet).toHaveBeenCalledWith(
      '/chat/attachments/attachment%2Fwith%20space/content',
      { signal: undefined, responseType: 'blob' },
    )
  })

  it('normalizes attachment metadata and accepts an attachment-only completion envelope', async () => {
    expect(normalizeChatAttachment({
      id: 'document-1',
      name: 'brief.pdf',
      kind: 'document',
      mime_type: 'application/pdf',
      size: 2048,
      status: 'ready',
      expires_at: '2099-01-01T00:00:00Z',
      page_count: 3,
    })).toMatchObject({
      id: 'document-1',
      mimeType: 'application/pdf',
      pageCount: 3,
    })

		expect(normalizeChatAttachment({
			id: 'library-file-1',
			name: 'notes.txt',
			kind: 'document',
			mime_type: 'text/plain',
			size: 12,
			status: 'ready',
			expires_at: '2126-01-01T00:00:00Z',
		})).toMatchObject({ id: 'library-file-1', expiresAt: '2126-01-01T00:00:00Z' })

    const fetchMock = vi.fn().mockResolvedValue(successfulStreamResponse('data: [DONE]\n\n'))
    vi.stubGlobal('fetch', fetchMock)
    const onAccepted = vi.fn()
    await streamChatCompletion({
      conversationId: 'conversation-attachment',
      model: 'gpt-5',
      expectedHeadMessageId: null,
      userMessage: {
        id: 'user-attachment',
        content: '',
        attachmentIds: ['document-1'],
      },
      assistantMessageId: 'assistant-attachment',
    }, { onAccepted })

    const request = fetchMock.mock.calls[0]?.[1] as RequestInit
    expect(JSON.parse(String(request.body))).toMatchObject({
      user_message: {
        id: 'user-attachment',
        content: '',
        attachment_ids: ['document-1'],
      },
    })
    expect(onAccepted).toHaveBeenCalledTimes(1)
  })

  it('serializes library references separately from uploaded attachment ids', async () => {
    const fetchMock = vi.fn().mockResolvedValue(successfulStreamResponse('data: [DONE]\n\n'))
    vi.stubGlobal('fetch', fetchMock)

    await streamChatCompletion({
      conversationId: 'conversation-library',
      model: 'gpt-5',
      expectedHeadMessageId: null,
      userMessage: {
        id: 'user-library',
        content: 'Summarize this library file',
        attachments: [{ source: 'library', fileId: 'library-file-1' }],
      },
      assistantMessageId: 'assistant-library',
    })

    const request = fetchMock.mock.calls[0]?.[1] as RequestInit
    const body = JSON.parse(String(request.body)) as {
      user_message: Record<string, unknown>
    }
    expect(body.user_message).toEqual({
      id: 'user-library',
      content: 'Summarize this library file',
      attachments: [{ source: 'library', file_id: 'library-file-1' }],
    })
    expect(body.user_message).not.toHaveProperty('attachment_ids')
  })

  it('treats a non-2xx completion with an explicit chat receipt as accepted before surfacing the error', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false,
      status: 422,
      statusText: 'Unprocessable Entity',
      headers: new Headers({ 'X-Chat-Receipt-ID': 'receipt-bound' }),
      text: vi.fn().mockResolvedValue(JSON.stringify({
        code: 'CHAT_PROVIDER_FAILED',
        message: 'Provider failed after prepare',
      })),
    }))
    const onAccepted = vi.fn()
    const onReceiptId = vi.fn()

    await expect(streamChatCompletion(
      historyCompletionRequest(),
      { onAccepted, onReceiptId },
    )).rejects.toMatchObject({ code: 'CHAT_PROVIDER_FAILED' })

    expect(onReceiptId).toHaveBeenCalledWith('receipt-bound')
    expect(onAccepted).toHaveBeenCalledTimes(1)
  })

  it('does not treat the universal request correlation header as chat acceptance', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false,
      status: 503,
      statusText: 'Service Unavailable',
      headers: new Headers({ 'X-Client-Request-ID': 'correlation-only' }),
      text: vi.fn().mockResolvedValue(JSON.stringify({
        code: 'CHAT_CATALOG_UNAVAILABLE',
        message: 'Catalog unavailable before prepare',
      })),
    }))
    const onAccepted = vi.fn()
    const onReceiptId = vi.fn()

    await expect(streamChatCompletion(
      historyCompletionRequest(),
      { onAccepted, onReceiptId },
    )).rejects.toMatchObject({ code: 'CHAT_CATALOG_UNAVAILABLE' })

    expect(onReceiptId).not.toHaveBeenCalled()
    expect(onAccepted).not.toHaveBeenCalled()
  })

  it('keeps a non-2xx completion without a receipt header in pre-accept state', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false,
      status: 409,
      statusText: 'Conflict',
      headers: new Headers(),
      text: vi.fn().mockResolvedValue(JSON.stringify({
        code: 'CHAT_HISTORY_CONFLICT',
        message: 'Not prepared',
      })),
    }))
    const onAccepted = vi.fn()

    await expect(streamChatCompletion(
      historyCompletionRequest(),
      { onAccepted },
    )).rejects.toMatchObject({ code: 'CHAT_HISTORY_CONFLICT' })

    expect(onAccepted).not.toHaveBeenCalled()
  })
})
