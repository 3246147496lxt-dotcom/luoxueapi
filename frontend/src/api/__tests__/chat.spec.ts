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
          { id: 'sol', supports_reasoning_slider: true },
          { id: 'legacy', supportsReasoningSlider: false },
          { id: 'spoofed', supports_reasoning_slider: 'true' },
        ],
        balance: 2,
      },
    })

    await expect(getChatModels()).resolves.toEqual({
      models: [
        { id: 'sol', supports_reasoning_slider: true },
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
      reasoning_effort: 'xhigh',
      expected_head_message_id: 'assistant-previous',
      user_message: { id: 'user-1', content: 'Hi' },
      assistant_message_id: 'assistant-1',
    })
    expect(Object.keys(JSON.parse(String(init.body))).sort()).toEqual([
      'assistant_message_id',
      'conversation_id',
      'expected_head_message_id',
      'model',
      'reasoning_effort',
      'user_message',
    ])
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
