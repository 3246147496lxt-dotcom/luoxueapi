import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get } = vi.hoisted(() => ({
  get: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
  },
}))

import { listBillingReceipts } from '@/api/admin/usage'

describe('admin usage billing receipts API', () => {
  beforeEach(() => {
    get.mockReset()
  })

  it('keeps the real billing ledger id and request id as separate identifiers', async () => {
    get.mockResolvedValue({
      data: {
        data: {
          items: [
            {
              id: 17,
              request_id: 'client:11111111-1111-4111-8111-111111111111',
              source: 'web_chat',
              user_id: 9,
              user_email: 'user@example.com',
              requested_model: 'gpt-5.5',
              model: 'gpt-5.5-2026-07-01',
              input_tokens: 120,
              output_tokens: 42,
              cache_creation_tokens: 6,
              cache_read_tokens: 12,
              gross_amount: '0.012345',
              charged_amount: '0.010000',
              balance_before: '10.000000',
              balance_after: '9.990000',
              status: 'charged',
              created_at: '2026-07-25T01:02:03Z',
            },
          ],
          total: 1,
          page: 2,
          page_size: 20,
          pages: 1,
        },
      },
    })
    const controller = new AbortController()

    const result = await listBillingReceipts({
      page: 2,
      page_size: 20,
      source: 'web_chat',
      user_id: 9,
      model: 'gpt-5.5',
      receipt_id: '11111111-1111-4111-8111-111111111111',
      status: 'charged',
      start_date: '2026-07-24',
      end_date: '2026-07-25',
    }, {
      signal: controller.signal,
    })

    expect(get).toHaveBeenCalledWith('/admin/billing/receipts', {
      params: expect.objectContaining({
        source: 'web_chat',
        user_id: 9,
        receipt_id: '11111111-1111-4111-8111-111111111111',
        status: 'charged',
      }),
      signal: controller.signal,
    })
    expect(result).toEqual({
      items: [
        expect.objectContaining({
          id: 17,
          row_key: 'receipt:17',
          receipt_id: '17',
          request_id: 'client:11111111-1111-4111-8111-111111111111',
          user_id: 9,
          user_email: 'user@example.com',
          source: 'web_chat',
          requested_model: 'gpt-5.5',
          actual_model: 'gpt-5.5-2026-07-01',
          tokens: expect.objectContaining({
            input_tokens: 120,
            output_tokens: 42,
            cache_tokens: 18,
          }),
          gross_cost: 0.012345,
          charged_amount: 0.01,
          balance_before: 10,
          balance_after: 9.99,
          status: 'charged',
        }),
      ],
      total: 1,
      page: 2,
      page_size: 20,
      pages: 1,
    })
    expect(result.items[0]?.user).toBeNull()
    expect(result.items[0]?.username).toBeNull()
  })

  it('prefers a future explicit receipt id without conflating the request id', async () => {
    get.mockResolvedValue({
      data: {
        items: [{
          id: 18,
          receipt_id: 'rcpt-chat-18',
          request_id: 'req-chat-18',
          user_id: 10,
        }],
      },
    })

    const result = await listBillingReceipts({ page: 1, page_size: 20 })

    expect(result.items[0]).toEqual(expect.objectContaining({
      id: 18,
      row_key: 'receipt:rcpt-chat-18',
      receipt_id: 'rcpt-chat-18',
      request_id: 'req-chat-18',
    }))
  })

  it('keeps unpersisted request attempts distinct without inventing receipt ID 0', async () => {
    get.mockResolvedValue({
      data: {
        items: [
          {
            id: 0,
            request_id: 'client:attempt-one',
            user_id: 10,
            status: 'failed',
            created_at: '2026-07-25T01:02:03Z',
          },
          {
            id: 0,
            request_id: 'client:attempt-two',
            user_id: 10,
            status: 'pending',
            created_at: '2026-07-25T01:03:03Z',
          },
        ],
      },
    })

    const result = await listBillingReceipts({ page: 1, page_size: 20 })

    expect(result.items.map(({ receipt_id, row_key, request_id }) => ({
      receipt_id,
      row_key,
      request_id,
    }))).toEqual([
      {
        receipt_id: '',
        row_key: 'request:client:attempt-one',
        request_id: 'client:attempt-one',
      },
      {
        receipt_id: '',
        row_key: 'request:client:attempt-two',
        request_id: 'client:attempt-two',
      },
    ])
  })

  it('accepts the standard items payload without requiring a second data wrapper', async () => {
    get.mockResolvedValue({
      data: {
        items: [],
        total: 0,
        page: 1,
        page_size: 50,
        pages: 0,
      },
    })

    await expect(listBillingReceipts({ page: 1, page_size: 50 })).resolves.toEqual({
      items: [],
      total: 0,
      page: 1,
      page_size: 50,
      pages: 0,
    })
  })
})
