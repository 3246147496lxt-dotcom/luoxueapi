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

  it('requests the read-only receipt endpoint and normalizes nested receipt fields', async () => {
    get.mockResolvedValue({
      data: {
        data: {
          receipts: [
            {
              id: 17,
              receipt_id: 'rcpt-chat-17',
              request_id: 'req-chat-17',
              user: { id: 9, email: 'user@example.com' },
              requested_model: 'gpt-5.5',
              upstream_model: 'gpt-5.5-2026-07-01',
              tokens: {
                input: 120,
                output: 42,
                cache: 18,
              },
              gross_amount: '0.012345',
              actual_cost: '0.010000',
              balance_before: '10.000000',
              balance_after: '9.990000',
              billing_status: 'charged',
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
      receipt_id: 'rcpt-chat-17',
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
        receipt_id: 'rcpt-chat-17',
        status: 'charged',
      }),
      signal: controller.signal,
    })
    expect(result).toEqual({
      items: [
        expect.objectContaining({
          id: 17,
          receipt_id: 'rcpt-chat-17',
          request_id: 'req-chat-17',
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
