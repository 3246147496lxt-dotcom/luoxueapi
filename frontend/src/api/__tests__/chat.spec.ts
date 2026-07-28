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
  deleteChatConversation,
  getChatAttempt,
  getChatSync,
  getChatReceipt,
  getChatModels,
  listChatConversations,
  parseChatCompletionSSE,
  patchChatConversation,
  pollChatReceipt,
  searchChatConversations,
  streamChatCompletion,
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
      { 'X-Client-Request-ID': 'receipt-stream' },
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
        { 'X-Client-Request-ID': 'receipt-from-401' },
      ))
      .mockResolvedValueOnce(successfulStreamResponse(
        'data: [DONE]\n\n',
        { 'X-Client-Request-ID': 'receipt-after-refresh' },
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
})
